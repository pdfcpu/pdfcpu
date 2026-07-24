/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sign

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"golang.org/x/crypto/ocsp"
)

type ocspAggregationFixture struct {
	issuer      *x509.Certificate
	issuerKey   *rsa.PrivateKey
	certificate *x509.Certificate
	now         time.Time
	good        []byte
	revoked     []byte
	stale       []byte
}

func newOCSPAggregationFixture(t *testing.T) ocspAggregationFixture {
	t.Helper()
	issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Aggregated Certificate")
	now := time.Now().UTC()
	return ocspAggregationFixture{
		issuer:      issuer,
		issuerKey:   issuerKey,
		certificate: cert,
		now:         now,
		good: testOCSPResponseBytesWithStatus(
			t,
			issuer,
			issuer,
			issuerKey,
			cert.SerialNumber,
			nil,
			now.Add(-10*time.Minute),
			now.Add(time.Hour),
			ocsp.Good,
		),
		revoked: testOCSPResponseBytesWithStatus(
			t,
			issuer,
			issuer,
			issuerKey,
			cert.SerialNumber,
			nil,
			now.Add(-5*time.Minute),
			now.Add(time.Hour),
			ocsp.Revoked,
		),
		stale: testOCSPResponseBytesWithStatus(
			t,
			issuer,
			issuer,
			issuerKey,
			cert.SerialNumber,
			nil,
			now.Add(-2*time.Hour),
			now.Add(-time.Hour),
			ocsp.Good,
		),
	}
}

// TestArchivedOCSPObservationOrderIndependent verifies every archived
// candidate is retained without producing a historical conclusion.
func TestArchivedOCSPObservationOrderIndependent(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	tests := []struct {
		name      string
		responses [][]byte
	}{
		{"GoodThenRevoked", [][]byte{fixture.good, fixture.revoked}},
		{"RevokedThenGood", [][]byte{fixture.revoked, fixture.good}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details, err := processArchivedOCSPResponses(
				fixture.certificate,
				fixture.issuer,
				tt.responses,
			)
			requireArchivedOCSPObservations(t, details, err, 2)
		})
	}
}

// TestCurrentOCSPFirstConclusionWins verifies online processing may stop after
// the first authenticated, applicable, conclusive response.
func TestCurrentOCSPFirstConclusionWins(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	tests := []struct {
		name      string
		responses [][]byte
		want      int
	}{
		{"GoodThenRevoked", [][]byte{fixture.good, fixture.revoked}, model.True},
		{"RevokedThenGood", [][]byte{fixture.revoked, fixture.good}, model.False},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details, err := processCurrentOCSPFixture(t, fixture, tt.responses)
			if err != nil {
				t.Fatal(err)
			}
			if details == nil || details.Status != tt.want || len(details.OCSPs) != 1 {
				t.Fatalf("first online conclusion not retained: %+v", details)
			}
		})
	}
}

// TestOCSPConflictingEqualTimeEvidenceIsUnknown verifies conflicting archived
// claims remain observations without a historical applicability decision.
func TestOCSPConflictingEqualTimeEvidenceIsUnknown(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	thisUpdate := fixture.now.Add(-time.Minute)
	good := testOCSPResponseBytesWithStatus(
		t,
		fixture.issuer,
		fixture.issuer,
		fixture.issuerKey,
		fixture.certificate.SerialNumber,
		nil,
		thisUpdate,
		fixture.now.Add(time.Hour),
		ocsp.Good,
	)
	revoked := testOCSPResponseBytesWithStatus(
		t,
		fixture.issuer,
		fixture.issuer,
		fixture.issuerKey,
		fixture.certificate.SerialNumber,
		nil,
		thisUpdate,
		fixture.now.Add(time.Hour),
		ocsp.Revoked,
	)
	for _, responses := range [][][]byte{{good, revoked}, {revoked, good}} {
		details, err := processArchivedOCSPResponses(
			fixture.certificate,
			fixture.issuer,
			responses,
		)
		if err != nil {
			t.Fatal(err)
		}
		if details.Status != model.Unknown ||
			details.Reason != "OCSP: archived response applicability unavailable" ||
			len(details.OCSPs) != 2 {
			t.Fatalf("conflicting OCSP evidence produced a conclusion: %+v", details)
		}
	}
}

// TestArchivedOCSPRevocationRemainsObservation verifies a revocation claim
// cannot become conclusive without a selected assessment time.
func TestArchivedOCSPRevocationRemainsObservation(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	thisUpdate := fixture.now.Add(-time.Minute)
	good := testOCSPResponseBytesWithStatus(
		t,
		fixture.issuer,
		fixture.issuer,
		fixture.issuerKey,
		fixture.certificate.SerialNumber,
		nil,
		thisUpdate,
		fixture.now.Add(time.Hour),
		ocsp.Good,
	)
	revoked := testEffectiveRevokedOCSPResponse(
		t,
		fixture,
		thisUpdate,
		fixture.now.Add(-time.Hour),
	)
	details, err := processArchivedOCSPResponses(
		fixture.certificate,
		fixture.issuer,
		[][]byte{good, revoked},
	)
	requireArchivedOCSPObservations(t, details, err, 2)
}

// TestOCSPRetainsNonconclusiveObservations verifies failures, stale responses
// and inconclusive delegated responders cannot establish or hide a status.
func TestOCSPRetainsNonconclusiveObservations(t *testing.T) {
	t.Run("ArchivedGoodPlusStale", testArchivedGoodPlusStale)
	t.Run("CurrentInvalidThenGood", testCurrentInvalidThenGood)
	t.Run("CurrentTwoFailedResponders", testCurrentTwoFailedResponders)
	t.Run("CurrentDelegatedResponderInconclusive", testCurrentDelegatedResponderInconclusive)
}

func testArchivedGoodPlusStale(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	details, err := processArchivedOCSPResponses(
		fixture.certificate,
		fixture.issuer,
		[][]byte{fixture.good, fixture.stale},
	)
	requireArchivedOCSPObservations(t, details, err, 2)
}

func testCurrentInvalidThenGood(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	details, err := processCurrentOCSPFixture(t, fixture, [][]byte{{1}, fixture.good})
	requireGoodWithRejectedObservation(t, details, err)
}

func testCurrentTwoFailedResponders(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	details, err := processCurrentOCSPFixture(t, fixture, [][]byte{{1}, {2}})
	if err == nil || details == nil || details.Status != model.Unknown || len(details.OCSPs) != 2 {
		t.Fatalf("failed responder evidence: details=%+v err=%v", details, err)
	}
	for _, want := range []string{"responder 1", "responder 2"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in %v", want, err)
		}
	}
}

func testCurrentDelegatedResponderInconclusive(t *testing.T) {
	fixture := newOCSPAggregationFixture(t)
	responderKey := testRSAKey(t)
	template := testCertTemplate("Delegated Aggregated Responder", false)
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning}
	responder := testCertificate(
		t,
		template,
		fixture.issuer,
		&responderKey.PublicKey,
		fixture.issuerKey,
	)
	response := testOCSPResponseBytes(
		t,
		fixture.issuer,
		responder,
		responderKey,
		fixture.certificate.SerialNumber,
		responder,
		fixture.now.Add(-time.Minute),
		fixture.now.Add(time.Hour),
	)
	details, err := processCurrentOCSPFixture(t, fixture, [][]byte{response})
	if err != nil {
		t.Fatal(err)
	}
	if details.Status != model.Unknown ||
		details.Reason != "OCSP: delegated responder revocation not assessed" ||
		details.OCSP == nil ||
		details.OCSP.Authenticated != model.True ||
		details.OCSP.Applicable != model.True ||
		details.OCSP.ResponderRevocation != model.Unknown {
		t.Fatalf("delegated responder produced a conclusion: %+v", details)
	}
}

func processCurrentOCSPFixture(
	t *testing.T,
	fixture ocspAggregationFixture,
	responses [][]byte,
) (*model.RevocationDetails, error) {
	t.Helper()
	urls := make([]string, len(responses))
	responseByURL := map[string][]byte{}
	for i, response := range responses {
		url := fmt.Sprintf("https://ocsp.test/%d", i+1)
		urls[i] = url
		responseByURL[url] = response
	}
	fixture.certificate.OCSPServer = urls
	return processCurrentOCSPResponses(
		fixture.certificate,
		fixture.issuer,
		testCurrentCRLClient(responseByURL),
	)
}

func requireArchivedOCSPObservations(
	t *testing.T,
	details *model.RevocationDetails,
	err error,
	count int,
) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if details == nil ||
		details.Status != model.Unknown ||
		details.Reason != "OCSP: archived response applicability unavailable" ||
		len(details.OCSPs) != count {
		t.Fatalf("archived OCSP observations: %+v", details)
	}
	for _, evidence := range details.OCSPs {
		if evidence.Applicable != model.Unknown {
			t.Fatalf("archived OCSP evidence became applicable: %+v", evidence)
		}
	}
}

func requireGoodWithRejectedObservation(
	t *testing.T,
	details *model.RevocationDetails,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if details == nil || details.Status != model.True || len(details.OCSPs) != 2 {
		t.Fatalf("OCSP observation aggregation: %+v", details)
	}
	for _, evidence := range details.OCSPs {
		if evidence.Error != "" && evidence.Authenticated != model.True {
			return
		}
	}
	t.Fatalf("rejected candidate missing from observations: %+v", details.OCSPs)
}

func testEffectiveRevokedOCSPResponse(
	t *testing.T,
	fixture ocspAggregationFixture,
	thisUpdate, revokedAt time.Time,
) []byte {
	t.Helper()
	bb, err := ocsp.CreateResponse(
		fixture.issuer,
		fixture.issuer,
		ocsp.Response{
			Status:       ocsp.Revoked,
			SerialNumber: fixture.certificate.SerialNumber,
			ThisUpdate:   thisUpdate,
			NextUpdate:   fixture.now.Add(time.Hour),
			RevokedAt:    revokedAt,
		},
		fixture.issuerKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	return bb
}

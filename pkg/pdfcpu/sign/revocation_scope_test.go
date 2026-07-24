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
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestLeafRevocationDoesNotClaimParentAssessment verifies authenticated,
// applicable leaf evidence establishes the compatibility revocation reason
// without implying that the parent certificate was revocation-checked.
func TestLeafRevocationDoesNotClaimParentAssessment(t *testing.T) {
	issuer, issuerKey, leaf := testCurrentCRLChain(t, "Leaf Revocation Issuer")
	assessmentTime := time.Now()
	crl := testCurrentCRL(
		t,
		issuer,
		issuerKey,
		assessmentTime.Add(-time.Hour),
		assessmentTime.Add(time.Hour),
		[]x509.RevocationListEntry{{
			SerialNumber:   leaf.SerialNumber,
			RevocationTime: assessmentTime.Add(-time.Minute),
		}},
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(crl)
	}))
	defer server.Close()
	leaf.CRLDistributionPoints = []string{server.URL}

	roots := x509.NewCertPool()
	roots.AddCert(issuer)
	signer := &model.Signer{}
	result := unknownSignatureResult()
	conf := model.NewDefaultConfiguration()
	conf.PreferredCertRevocationChecker = model.CRL
	conf.AllowedRevocationHosts = []string{revocationTestHost(t, server.URL)}

	validateCertChains(
		[][]*x509.Certificate{{leaf, issuer}},
		true,
		roots,
		signer,
		[][]byte{crl},
		nil,
		result,
		conf,
	)

	if result.Reason != model.SignatureReasonCertRevoked ||
		result.Status != model.SignatureStatusUnknown {
		t.Fatalf("leaf revocation result: %+v", result)
	}
	if got := result.Reason.String(); got != "signer's certificate has been revoked" {
		t.Fatalf("leaf revocation text: got %q", got)
	}
	if strings.Contains(result.Reason.String(), "parent") {
		t.Fatalf("leaf-only assessment claims parent revocation: %q", result.Reason)
	}

	leafDetails := signer.Certificate
	if leafDetails == nil ||
		leafDetails.Revocation.Status != model.False ||
		leafDetails.Revocation.CRL == nil ||
		!authenticatedApplicableCRL(leafDetails.Revocation.CRL) {
		t.Fatalf("authenticated leaf revocation evidence missing: %+v", leafDetails)
	}
	parent := leafDetails.IssuerCertificate
	if parent == nil {
		t.Fatal("parent certificate details missing")
	}
	if parent.Revocation.Status != model.Unknown ||
		parent.Revocation.CRL != nil ||
		parent.Revocation.OCSP != nil ||
		len(parent.Revocation.CRLs) != 0 ||
		len(parent.Revocation.OCSPs) != 0 {
		t.Fatalf("parent certificate unexpectedly has revocation evidence: %+v", parent.Revocation)
	}
}

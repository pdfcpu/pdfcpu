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
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"golang.org/x/crypto/ocsp"
)

// TestClaimedSigningTimeDoesNotSelectCertificateAssessmentTime verifies a
// claimed time remains displayed while certificate validity is assessed now.
func TestClaimedSigningTimeDoesNotSelectCertificateAssessmentTime(t *testing.T) {
	claimedTime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	fixture := newDetachedP7SignerFixture(
		t,
		[]pkcs7.Attribute{{Type: oidSigningTime, Value: claimedTime}},
		nil,
	)
	for _, cert := range fixture.certs {
		cert.NotBefore = claimedTime.Add(-time.Hour)
		cert.NotAfter = claimedTime.Add(time.Hour)
	}
	result := unknownSignatureResult()

	err := verifyP7Signer(
		fixture.signer,
		fixture.certs,
		fixture.roots,
		fixture.content,
		fixture.content,
		true,
		false,
		false,
		0,
		0,
		result,
		fixture.ctx,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Details.SigningTime.Equal(claimedTime) {
		t.Fatalf("got displayed signing time %v, want %v", result.Details.SigningTime, claimedTime)
	}
	if len(result.Details.Signers) != 1 || result.Details.Signers[0].Certificate == nil {
		t.Fatalf("missing certificate assessment: %+v", result.Details.Signers)
	}
	certificate := result.Details.Signers[0].Certificate
	if !certificate.Expired {
		t.Fatalf("claimed time suppressed current expiration: %+v", certificate)
	}
	if result.Reason != model.SignatureReasonCertExpired {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertExpired)
	}
	if !certificate.ValidationTime.Time.IsZero() {
		t.Fatalf("claimed time became certificate validation time: %+v", certificate.ValidationTime)
	}
}

// TestClaimedSigningTimeDoesNotEnableArchivedCRLConclusion verifies archived
// CRL applicability remains unknown without a certificate assessment time.
func TestClaimedSigningTimeDoesNotEnableArchivedCRLConclusion(t *testing.T) {
	issuer, issuerKey, cert := testCurrentCRLChain(t, "Archived CRL Issuer")
	claimedTime := time.Now().Add(-48 * time.Hour)
	crl := testCurrentCRL(
		t,
		issuer,
		issuerKey,
		claimedTime.Add(-time.Hour),
		claimedTime.Add(time.Hour),
		nil,
	)
	certDetails, result := assessRevocationWithoutTimestamp(
		cert,
		issuer,
		[][]byte{crl},
		nil,
		model.CRL,
	)

	requireUnknownArchivedRevocation(t, certDetails, result)
}

// TestClaimedSigningTimeDoesNotEnableArchivedOCSPConclusion verifies archived
// OCSP applicability remains unknown without a certificate assessment time.
func TestClaimedSigningTimeDoesNotEnableArchivedOCSPConclusion(t *testing.T) {
	now := time.Now()
	cert, issuer, _, response := testArchivedOCSPFixture(
		t,
		now.Add(-time.Hour),
		now.Add(time.Hour),
		now.Add(time.Hour),
	)
	certDetails, result := assessRevocationWithoutTimestamp(
		cert,
		issuer,
		nil,
		[][]byte{response},
		model.OCSP,
	)

	requireUnknownArchivedRevocation(t, certDetails, result)
}

// TestClaimedGenTimeIsEvidenceNotArchivalValidationTime verifies a
// locally validated timestamp observation remains display evidence and is
// not consumed as an archival validation time.
func TestClaimedGenTimeIsEvidenceNotArchivalValidationTime(t *testing.T) {
	genTime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	evidence := timestampEvidence{
		Kind:                  timestampKindDocument,
		SigningTime:           genTime,
		Present:               true,
		TokenInfo:             &timestampTokenInfo{GeneratedAt: genTime},
		DigestVerified:        true,
		SignatureVerified:     true,
		CorrectProfile:        true,
		LocalTSAPathValidated: true,
	}
	if !isCryptographicallyAuthenticatedDocumentTimestampEvidence(evidence) ||
		!isLocallyValidatedDocumentTimestampEvidence(evidence) {
		t.Fatalf("missing locally validated timestamp evidence: %+v", evidence)
	}

	certificate := &model.CertificateDetails{}
	signer := &model.Signer{Certificate: certificate}
	result := unknownSignatureResult()
	applyTimestampEvidence(evidence, timestampApplication{
		signer:               signer,
		result:               result,
		setResultSigningTime: true,
		problemPrefix:        "timestamp contract",
	})
	if !signer.HasTimestamp ||
		!signer.Timestamp.Equal(genTime) ||
		!result.Details.SigningTime.Equal(genTime) ||
		evidence.TokenInfo == nil ||
		!evidence.TokenInfo.GeneratedAt.Equal(genTime) ||
		!certificate.ValidationTime.Time.IsZero() {
		t.Fatalf("claimed genTime was not retained as observed evidence: signer=%+v result=%+v", signer, result)
	}
}

// TestClaimedGenTimeDoesNotConcludeArchivedRevocation verifies archived CRL
// and OCSP material remains inconclusive without a certificate assessment time.
func TestClaimedGenTimeDoesNotConcludeArchivedRevocation(t *testing.T) {
	claimedGenTime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	t.Run("CRL", func(t *testing.T) {
		issuer, issuerKey, cert := testCurrentCRLChain(t, "Claimed genTime CRL Issuer")
		crl := testCurrentCRL(
			t,
			issuer,
			issuerKey,
			claimedGenTime.Add(-time.Hour),
			claimedGenTime.Add(time.Hour),
			nil,
		)
		certificate, result := assessRevocationWithoutTimestamp(
			cert,
			issuer,
			[][]byte{crl},
			nil,
			model.CRL,
		)
		requireUnknownArchivedRevocation(t, certificate, result)
	})

	t.Run("OCSP", func(t *testing.T) {
		issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Claimed genTime Certificate")
		response := testOCSPResponseBytes(
			t,
			issuer,
			issuer,
			issuerKey,
			cert.SerialNumber,
			nil,
			claimedGenTime.Add(-time.Hour),
			claimedGenTime.Add(time.Hour),
		)
		certificate, result := assessRevocationWithoutTimestamp(
			cert,
			issuer,
			nil,
			[][]byte{response},
			model.OCSP,
		)
		requireUnknownArchivedRevocation(t, certificate, result)
	})
}

// TestArchivedRevocationWithoutAssessmentTimeIsObservationOnly verifies
// authenticated archived status claims remain recorded but inconclusive.
func TestArchivedRevocationWithoutAssessmentTimeIsObservationOnly(t *testing.T) {
	observedTime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)

	t.Run("CRL", func(t *testing.T) {
		issuer, issuerKey, cert := testCurrentCRLChain(t, "Observation-only CRL Issuer")
		tests := []struct {
			name    string
			entries []x509.RevocationListEntry
		}{
			{name: "Good"},
			{
				name: "Revoked",
				entries: []x509.RevocationListEntry{{
					SerialNumber:   cert.SerialNumber,
					RevocationTime: observedTime.Add(-time.Minute),
				}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				crl := testCurrentCRL(
					t,
					issuer,
					issuerKey,
					observedTime.Add(-time.Hour),
					observedTime.Add(time.Hour),
					tt.entries,
				)
				certificate, result := assessRevocationWithoutTimestamp(
					cert,
					issuer,
					[][]byte{crl},
					nil,
					model.CRL,
				)
				requireUnknownArchivedRevocation(t, certificate, result)
				requireArchivedEvidenceCannotSatisfyRevocationGood(t, certificate)
				evidence := certificate.Revocation.CRL
				if evidence == nil ||
					len(certificate.Revocation.CRLs) != 1 ||
					evidence.Source != model.RevocationEvidenceSourceArchived ||
					evidence.SignatureValid != model.True ||
					evidence.Applicable != model.Unknown ||
					len(evidence.Entries) != len(tt.entries) {
					t.Fatalf("archived CRL observation missing: %+v", certificate.Revocation)
				}
			})
		}
	})

	t.Run("OCSP", func(t *testing.T) {
		issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Observation-only OCSP Certificate")
		tests := []struct {
			name       string
			status     int
			wantStatus int
		}{
			{name: "Good", status: ocsp.Good, wantStatus: model.True},
			{name: "Revoked", status: ocsp.Revoked, wantStatus: model.False},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				response := testOCSPResponseBytesWithStatus(
					t,
					issuer,
					issuer,
					issuerKey,
					cert.SerialNumber,
					nil,
					observedTime.Add(-time.Hour),
					observedTime.Add(time.Hour),
					tt.status,
				)
				certificate, result := assessRevocationWithoutTimestamp(
					cert,
					issuer,
					nil,
					[][]byte{response},
					model.OCSP,
				)
				requireUnknownArchivedRevocation(t, certificate, result)
				requireArchivedEvidenceCannotSatisfyRevocationGood(t, certificate)
				requireArchivedOCSPObservation(t, certificate, tt.wantStatus)
			})
		}
	})
}

// TestObservedGenTimeDoesNotSelectRevocationAssessmentTime verifies the DTS
// production path records genTime only after CRL and OCSP assessment.
func TestObservedGenTimeDoesNotSelectRevocationAssessmentTime(t *testing.T) {
	genTime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)

	t.Run("CRL", func(t *testing.T) {
		issuer, issuerKey, cert := testCurrentCRLChain(t, "Observed genTime CRL Issuer")
		crl := testCurrentCRL(
			t,
			issuer,
			issuerKey,
			genTime.Add(-time.Hour),
			genTime.Add(time.Hour),
			[]x509.RevocationListEntry{{
				SerialNumber:   cert.SerialNumber,
				RevocationTime: genTime.Add(-time.Minute),
			}},
		)
		requireDTSGenTimeNotUsedForRevocation(t, cert, issuer, genTime, [][]byte{crl}, nil)
	})

	t.Run("OCSP", func(t *testing.T) {
		issuer, issuerKey, cert := testOCSPIssuerAndCertificate(t, "Observed genTime OCSP Certificate")
		response := testOCSPResponseBytesWithStatus(
			t,
			issuer,
			issuer,
			issuerKey,
			cert.SerialNumber,
			nil,
			genTime.Add(-time.Hour),
			genTime.Add(time.Hour),
			ocsp.Revoked,
		)
		requireDTSGenTimeNotUsedForRevocation(t, cert, issuer, genTime, nil, [][]byte{response})
	})
}

// TestManualContextDTSRemainsObservedTimeOnly verifies ctx.DTS carries no
// reconstructable digest, CMS, profile or TSA-path evidence.
func TestManualContextDTSRemainsObservedTimeOnly(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	observedTime := time.Now().Add(-time.Hour)
	fixture.ctx.DTS = observedTime
	signer := &model.Signer{}
	result := unknownSignatureResult()

	checkTimestampToken(fixture.signer, fixture.ctx, signer, result)

	observed := documentTimestampEvidence(fixture.ctx.DTS)
	if !observed.Present ||
		!observed.SigningTime.Equal(observedTime) ||
		isCryptographicallyAuthenticatedDocumentTimestampEvidence(observed) ||
		isLocallyValidatedDocumentTimestampEvidence(observed) ||
		signer.HasTimestamp ||
		!signer.Timestamp.IsZero() ||
		!result.Details.SigningTime.IsZero() ||
		!fixture.ctx.DTS.Equal(observedTime) {
		t.Fatalf(
			"manual ctx.DTS crossed the presentation boundary: observed=%+v signer=%+v result=%+v",
			observed,
			signer,
			result,
		)
	}
}

func requireDTSGenTimeNotUsedForRevocation(
	t *testing.T,
	cert, issuer *x509.Certificate,
	genTime time.Time,
	crls, ocsps [][]byte,
) {
	t.Helper()
	roots := x509.NewCertPool()
	roots.AddCert(issuer)
	signer := &model.Signer{}
	result := unknownSignatureResult()
	conf := model.NewDefaultConfiguration()
	conf.Offline = true
	ctx := &model.Context{
		Configuration: conf,
		XRefTable:     &model.XRefTable{},
	}

	if _, err := validateDTSCert(
		cert,
		[]*x509.Certificate{cert, issuer},
		roots,
		crls,
		ocsps,
		signer,
		result,
		ctx,
	); err != nil {
		t.Fatal(err)
	}
	if signer.Certificate == nil ||
		signer.Certificate.Revocation.Status != model.Unknown ||
		!signer.Certificate.ValidationTime.Time.IsZero() ||
		result.Reason == model.SignatureReasonCertRevoked {
		t.Fatalf("genTime selected revocation assessment time: signer=%+v result=%+v", signer, result)
	}

	applyTimestampEvidence(documentTimestampEvidence(genTime), timestampApplication{
		signer:        signer,
		result:        result,
		problemPrefix: "timestamp contract",
	})
	if !signer.Certificate.ValidationTime.Time.IsZero() ||
		signer.Certificate.Revocation.Status != model.Unknown ||
		result.Reason == model.SignatureReasonCertRevoked {
		t.Fatalf("reported genTime changed revocation conclusion: signer=%+v result=%+v", signer, result)
	}
}

func assessRevocationWithoutTimestamp(
	cert, issuer *x509.Certificate,
	crls, ocsps [][]byte,
	preferred int,
) (*model.CertificateDetails, *model.SignatureValidationResult) {
	signer := &model.Signer{}
	certDetails := &model.CertificateDetails{}
	result := unknownSignatureResult()
	conf := model.NewDefaultConfiguration()
	conf.Offline = true
	conf.PreferredCertRevocationChecker = preferred

	checkRevocation(
		cert,
		issuer,
		x509.NewCertPool(),
		signer,
		certDetails,
		crls,
		ocsps,
		result,
		conf,
	)
	return certDetails, result
}

func requireUnknownArchivedRevocation(
	t *testing.T,
	certificate *model.CertificateDetails,
	result *model.SignatureValidationResult,
) {
	t.Helper()
	if certificate.Revocation.Status != model.Unknown {
		t.Fatalf("archived evidence produced status %d: %+v", certificate.Revocation.Status, certificate.Revocation)
	}
	if result.Reason != model.SignatureReasonCertRevocationUnknown {
		t.Fatalf(
			"archived evidence reason: got %s, want %s",
			result.Reason,
			model.SignatureReasonCertRevocationUnknown,
		)
	}
}

func requireArchivedEvidenceCannotSatisfyRevocationGood(
	t *testing.T,
	certificate *model.CertificateDetails,
) {
	t.Helper()
	assessment := completedLocalSignatureAssessment()
	assessment.applyCertificateAssessment(certificateAssessment{
		Certificate:           certificate,
		CertificatePathStatus: model.True,
	})
	if assessment.RevocationGood {
		t.Fatalf("archived-only evidence satisfied RevocationGood: %+v", certificate.Revocation)
	}
	result := unknownSignatureResult()
	if finalizeLocalSignatureResult(result, assessment) {
		t.Fatalf("archived-only evidence finalized a valid signature: %+v", result)
	}
}

func requireArchivedOCSPObservation(
	t *testing.T,
	certificate *model.CertificateDetails,
	wantStatus int,
) {
	t.Helper()
	evidence := certificate.Revocation.OCSP
	if evidence == nil ||
		len(certificate.Revocation.OCSPs) != 1 ||
		evidence.Source != model.RevocationEvidenceSourceArchived ||
		evidence.Authenticated != model.True ||
		evidence.Applicable != model.Unknown ||
		evidence.CertificateStatus != wantStatus {
		t.Fatalf("archived OCSP observation missing: %+v", certificate.Revocation)
	}
	if evidence.Authenticated == evidence.Applicable {
		t.Fatalf("OCSP authentication and applicability were conflated: %+v", evidence)
	}
}

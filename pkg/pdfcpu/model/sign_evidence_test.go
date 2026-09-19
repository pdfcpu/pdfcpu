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

package model

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// TestSignerValidationEvidenceZeroValue verifies absent observations do not
// establish positive or negative signer and timestamp conclusions.
func TestSignerValidationEvidenceZeroValue(t *testing.T) {
	var evidence SignerValidationEvidence
	checks := []struct {
		name   string
		status int
	}{
		{"SignatureAuthenticated", evidence.SignatureAuthenticated},
		{"DigestVerified", evidence.DigestVerified},
		{"ProfileValidated", evidence.ProfileValidated},
		{"CertificateIdentified", evidence.CertificateIdentified},
		{"TimestampDigestVerified", evidence.Timestamp.DigestVerified},
		{"TimestampSignatureAuthenticated", evidence.Timestamp.SignatureAuthenticated},
		{"TimestampProfileValidated", evidence.Timestamp.ProfileValidated},
		{"TimestampCertificatePathValidated", evidence.Timestamp.CertificatePathValidated},
	}

	for _, check := range checks {
		if check.status != Unknown {
			t.Errorf("%s: got %d, want Unknown", check.name, check.status)
		}
	}
	if evidence.Timestamp.Kind != TimestampKindUnspecified ||
		evidence.Timestamp.Present ||
		!evidence.Timestamp.Time.IsZero() {
		t.Fatalf("zero timestamp evidence established an observation: %+v", evidence.Timestamp)
	}
}

// TestTimestampValidationEvidencePreservesIndependentChecks verifies timestamp
// observations do not collapse their independent assessment results.
func TestTimestampValidationEvidencePreservesIndependentChecks(t *testing.T) {
	wantTime := time.Date(2026, time.September, 18, 12, 0, 0, 0, time.UTC)
	evidence := TimestampValidationEvidence{
		Kind:                     TimestampKindSignature,
		Present:                  true,
		Time:                     wantTime,
		DigestVerified:           True,
		SignatureAuthenticated:   False,
		ProfileValidated:         Unknown,
		CertificatePathValidated: True,
	}

	if evidence.Kind != TimestampKindSignature ||
		!evidence.Present ||
		!evidence.Time.Equal(wantTime) ||
		evidence.DigestVerified != True ||
		evidence.SignatureAuthenticated != False ||
		evidence.ProfileValidated != Unknown ||
		evidence.CertificatePathValidated != True {
		t.Fatalf("timestamp evidence checks were collapsed: %+v", evidence)
	}
}

// TestTimestampKindValues locks the exported timestamp-kind values.
func TestTimestampKindValues(t *testing.T) {
	tests := []struct {
		kind TimestampKind
		want TimestampKind
	}{
		{TimestampKindUnspecified, 0},
		{TimestampKindSignature, 1},
		{TimestampKindDocument, 2},
	}

	for _, tt := range tests {
		if tt.kind != tt.want {
			t.Errorf("timestamp kind: got %d, want %d", tt.kind, tt.want)
		}
	}
}

// TestCertificatePathEvidenceDoesNotChangeLocalPathOutput locks down the existing
// CertificateDetails presentation while structured evidence is populated.
func TestCertificatePathEvidenceDoesNotChangeLocalPathOutput(t *testing.T) {
	certDetails := CertificateDetails{
		Trust: TrustDetails{
			Status: False,
			Reason: "certificate path was not resolved using the configured local certificate store",
		},
	}
	want := certDetails.String()
	wantLocalPath := "Local path:\n" +
		"  Status: not ok\n" +
		"  Reason: certificate path was not resolved using the configured local certificate store"
	if !strings.Contains(want, wantLocalPath) {
		t.Fatalf("local path output is misaligned:\n%s", want)
	}
	certDetails.PathEvidence = CertificatePathEvidence{
		AssessmentScope: AssessmentScopeLocal,
		Method:          CertificatePathMethodLocalTrustStore,
		Status:          False,
		Reason:          "certificate path was not resolved using the configured local certificate store",
	}

	if got := certDetails.String(); got != want {
		t.Fatalf("path evidence changed local path output:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestRevocationEvidenceDoesNotChangeLocalAssessmentOutput locks down the
// RevocationDetails presentation while structured CRL and OCSP evidence is populated.
func TestRevocationEvidenceDoesNotChangeLocalAssessmentOutput(t *testing.T) {
	details := RevocationDetails{
		Status: True,
		Reason: "CRL: certificate status good",
	}
	want := details.String()
	if !strings.HasPrefix(want, "Status: ok") {
		t.Fatalf("revocation output does not scope its assessment as local:\n%s", want)
	}

	details.CRL = &CRLEvidence{
		AssessmentScope: AssessmentScopeLocal,
		Source:          RevocationEvidenceSourceArchived,
		IssuerMatched:   True,
		SignatureValid:  True,
		Applicable:      True,
	}
	if got := details.String(); got != want {
		t.Fatalf("CRL evidence changed revocation output:\ngot:\n%s\nwant:\n%s", got, want)
	}

	details.CRL = nil
	details.OCSP = &OCSPEvidence{
		AssessmentScope: AssessmentScopeLocal,
		Source:          RevocationEvidenceSourceOnline,
		Responder:       OCSPResponderIssuer,
		Authenticated:   True,
	}
	if got := details.String(); got != want {
		t.Fatalf("OCSP evidence changed revocation output:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestSignatureDetailTextWrapsAndAligns verifies long and multiline evidence
// remains within the output width and aligned beneath its value column.
func TestSignatureDetailTextWrapsAndAligns(t *testing.T) {
	reason := "OCSP: " + strings.Repeat("response unavailable ", 8) + "\nCRL: fetch failed"
	details := RevocationDetails{Status: Unknown, Reason: reason}
	lines := strings.Split(details.String(), "\n")

	continuation := strings.Repeat(" ", len("Reason: "))
	crlAligned := false
	for _, line := range lines {
		if utf8.RuneCountInString(line) > signatureOutputMaxWidth {
			t.Errorf("line exceeds %d columns: %q", signatureOutputMaxWidth, line)
		}
		if strings.TrimSpace(line) == "CRL: fetch failed" {
			crlAligned = strings.HasPrefix(line, continuation)
		}
	}
	if !crlAligned {
		t.Fatalf("multiline reason is not aligned:\n%s", details)
	}

	signer := Signer{Problems: []string{reason}}
	problemCRLAligned := false
	for _, line := range strings.Split(signer.String(false), "\n") {
		if utf8.RuneCountInString(line) > signatureOutputMaxWidth {
			t.Errorf("problem line exceeds %d columns: %q", signatureOutputMaxWidth, line)
		}
		if strings.TrimSpace(line) == "CRL: fetch failed" {
			problemCRLAligned = strings.HasPrefix(line, "  ")
		}
	}
	if !problemCRLAligned {
		t.Fatalf("multiline problem is not aligned:\n%s", signer.String(false))
	}
}

// TestSignerStringOmitsLegacyLTVConclusion verifies the compatibility field is
// not presented as an LTV validation conclusion.
func TestSignerStringOmitsLegacyLTVConclusion(t *testing.T) {
	got := (Signer{LTVEnabled: true}).String(false)
	if strings.Contains(got, "LTVEnabled") {
		t.Fatalf("signer output exposes legacy LTV conclusion:\n%s", got)
	}
}

// TestCertificateDetailsUsesQCPolicyLabel verifies certificate-policy evidence
// is not presented as a legal qualification conclusion.
func TestCertificateDetailsUsesQCPolicyLabel(t *testing.T) {
	got := (CertificateDetails{Qualified: true}).String()
	if !strings.Contains(got, "QC policy:") {
		t.Fatalf("certificate output omits recognized QC policy evidence:\n%s", got)
	}
	if strings.Contains(got, "Qualified Evidence:") || strings.Contains(got, "Qualified:") {
		t.Fatalf("certificate output uses legacy qualified conclusion:\n%s", got)
	}
}

func TestCertificateChainHeadingsAlign(t *testing.T) {
	certificate := CertificateDetails{IssuerCertificate: &CertificateDetails{
		CA: true,
		IssuerCertificate: &CertificateDetails{
			CA: true,
		},
	}}
	got := strings.Join(certificate.chain("  "), "\n")
	for _, heading := range []string{"Certificate:", "Intermediate CA:", "Root CA:"} {
		if !strings.Contains("\n"+got, "\n  "+heading+"\n") {
			t.Errorf("%s is not aligned with the certificate chain:\n%s", heading, got)
		}
	}
}

func TestSignatureOutputFieldsAlignValuesWithinSection(t *testing.T) {
	got := strings.Join(signatureOutputFields("  ", []signatureOutputField{
		{label: "Document modified", value: "false"},
		{label: "Status", value: "unknown"},
		{label: "Reason", value: "certificate path not resolved"},
	}), "\n")
	want := "  Document modified: false\n" +
		"  Status:            unknown\n" +
		"  Reason:            certificate path not resolved"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestSignatureReasonCertRevokedCompatibility locks the exported identifier,
// numeric value and narrowed leaf-certificate presentation.
func TestSignatureReasonCertRevokedCompatibility(t *testing.T) {
	const compatibilityValue SignatureReason = 1 << 9
	const want = "signer's certificate has been revoked"

	if SignatureReasonCertRevoked != compatibilityValue {
		t.Fatalf(
			"SignatureReasonCertRevoked value changed: got %d, want %d",
			SignatureReasonCertRevoked,
			compatibilityValue,
		)
	}
	if got := SignatureReasonStrings[SignatureReasonCertRevoked]; got != want {
		t.Fatalf("revoked reason mapping: got %q, want %q", got, want)
	}
	if got := SignatureReasonCertRevoked.String(); got != want {
		t.Fatalf("revoked reason string: got %q, want %q", got, want)
	}
}

// TestSignatureReasonCertRevocationUnknownCompatibility verifies the new
// reason is appended without renumbering existing compatibility values.
func TestSignatureReasonCertRevocationUnknownCompatibility(t *testing.T) {
	const (
		selfSignedCompatibilityValue SignatureReason = 1 << 11
		unknownCompatibilityValue    SignatureReason = 1 << 12
		want                                         = "signer's certificate revocation status is unknown"
	)

	if SignatureReasonSelfSignedCertErr != selfSignedCompatibilityValue {
		t.Fatalf(
			"SignatureReasonSelfSignedCertErr value changed: got %d, want %d",
			SignatureReasonSelfSignedCertErr,
			selfSignedCompatibilityValue,
		)
	}
	if SignatureReasonCertRevocationUnknown != unknownCompatibilityValue {
		t.Fatalf(
			"SignatureReasonCertRevocationUnknown value: got %d, want %d",
			SignatureReasonCertRevocationUnknown,
			unknownCompatibilityValue,
		)
	}
	if got := SignatureReasonCertRevocationUnknown.String(); got != want {
		t.Fatalf("revocation-unknown reason string: got %q, want %q", got, want)
	}
}

// TestSignatureReasonNumericCompatibility locks every exported reason value.
func TestSignatureReasonNumericCompatibility(t *testing.T) {
	tests := []struct {
		reason SignatureReason
		want   SignatureReason
	}{
		{SignatureReasonUnknown, 1 << 0},
		{SignatureReasonDocNotModified, 1 << 1},
		{SignatureReasonDocModified, 1 << 2},
		{SignatureReasonSignatureForged, 1 << 3},
		{SignatureReasonSigningTimeInvalid, 1 << 4},
		{SignatureReasonTimestampTokenInvalid, 1 << 5},
		{SignatureReasonCertInvalid, 1 << 6},
		{SignatureReasonCertNotTrusted, 1 << 7},
		{SignatureReasonCertExpired, 1 << 8},
		{SignatureReasonCertRevoked, 1 << 9},
		{SignatureReasonInternal, 1 << 10},
		{SignatureReasonSelfSignedCertErr, 1 << 11},
		{SignatureReasonCertRevocationUnknown, 1 << 12},
		{SignatureReasonMalformed, 1 << 13},
		{SignatureReasonUnsupported, 1 << 14},
	}
	for _, tt := range tests {
		if tt.reason != tt.want {
			t.Errorf("reason value changed: got %d, want %d", tt.reason, tt.want)
		}
	}
}

// TestSignatureReasonLocalCertificateWording locks source wording for local
// certificate-path conclusions shared by every signature type.
func TestSignatureReasonLocalCertificateWording(t *testing.T) {
	tests := []struct {
		reason SignatureReason
		want   string
	}{
		{
			SignatureReasonCertNotTrusted,
			"signer's certificate path was not resolved using the configured local certificate store",
		},
		{
			SignatureReasonSelfSignedCertErr,
			"signer's self-signed certificate was not accepted by the configured local certificate assessment",
		},
		{
			SignatureReasonCertRevocationUnknown,
			"signer's certificate revocation status is unknown",
		},
		{
			SignatureReasonMalformed,
			"signature data is malformed",
		},
		{
			SignatureReasonUnsupported,
			"signature profile or algorithm is unsupported",
		},
	}
	for _, tt := range tests {
		if got := tt.reason.String(); got != tt.want {
			t.Errorf("reason %d: got %q, want %q", tt.reason, got, tt.want)
		}
	}
}

// TestDTSPresentationUsesLocalValidationLanguage verifies document timestamp
// presentation reports local assessment and retains path and revocation
// evidence without presenting a trust decision.
func TestDTSPresentationUsesLocalValidationLanguage(t *testing.T) {
	result := SignatureValidationResult{
		Signature: Signature{
			Type:   SigTypeDTS,
			Signed: true,
		},
		Status: SignatureStatusValid,
		Reason: SignatureReasonDocNotModified,
		Details: SignatureDetails{
			SubFilter: "ETSI.RFC3161",
			Signers: []*Signer{{
				Certificate: &CertificateDetails{
					Leaf: true,
					Trust: TrustDetails{
						Status: True,
						Reason: "certificate path resolved using the configured local certificate store",
					},
					Revocation: RevocationDetails{
						Status: True,
						Reason: "CRL: certificate status good",
					},
				},
			}},
		},
		DocModified: False,
	}

	got := result.String()
	for _, want := range []string{
		"document timestamp (locally validated, invisible, signed)",
		"Local path:",
		"Revocation:",
		"Status: ok",
		"CRL: certificate status good",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DTS presentation missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "document timestamp (trusted") || strings.Contains(got, "Trust:") {
		t.Fatalf("DTS presentation implies a trust decision:\n%s", got)
	}
}

func presentationResult(sigType int, reason SignatureReason, signerProblem, resultProblem string) SignatureValidationResult {
	return SignatureValidationResult{
		Signature: Signature{Type: sigType, Signed: true},
		Status:    SignatureStatusUnknown,
		Reason:    reason,
		Details: SignatureDetails{
			Signers: []*Signer{{
				Certificate: &CertificateDetails{
					Leaf: true,
					Trust: TrustDetails{
						Status: Unknown,
						Reason: "certificate path was not resolved using the configured local certificate store",
					},
					Revocation: RevocationDetails{
						Status: Unknown,
						Reason: SignatureReasonCertRevocationUnknown.String(),
					},
				},
				Problems: []string{signerProblem},
			}},
		},
		Problems: []string{resultProblem},
	}
}

// TestSignaturePresentationUsesStoredConclusions verifies ordinary signatures,
// document timestamps and usage-rights signatures render identical structured
// conclusion text and preserve Problems exactly as stored.
func TestSignaturePresentationUsesStoredConclusions(t *testing.T) {
	const (
		signerProblem = "signer problem: KEEP /Name <Value> unchanged"
		resultProblem = "result problem: KEEP [one two] unchanged"
		pathReason    = "certificate path was not resolved using the configured local certificate store"
	)
	tests := []struct {
		name       string
		sigType    int
		typeString string
	}{
		{"Ordinary", SigTypeForm, "form signature (invisible, signed)"},
		{"DTS", SigTypeDTS, "document timestamp (not locally validated, invisible, signed)"},
		{"UsageRights", SigTypeUR, "usage rights signature (invisible, signed)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := presentationResult(
				tt.sigType,
				SignatureReasonCertNotTrusted,
				signerProblem,
				resultProblem,
			)
			got := result.String()
			for _, want := range []string{
				tt.typeString,
				SignatureReasonCertNotTrusted.String(),
				"Local path:",
				pathReason,
				SignatureReasonCertRevocationUnknown.String(),
				signerProblem,
				resultProblem,
			} {
				if !strings.Contains(got, want) {
					t.Errorf("presentation missing stored text %q:\n%s", want, got)
				}
			}
			if strings.Contains(got, "                             Trust:") {
				t.Errorf("presentation implies an unrestricted trust decision:\n%s", got)
			}
		})
	}
}

// TestSignaturePresentationUsesConsistentFailureReasons verifies signature
// type does not alter self-signed or revocation-unknown conclusions.
func TestSignaturePresentationUsesConsistentFailureReasons(t *testing.T) {
	for _, reason := range []SignatureReason{
		SignatureReasonSelfSignedCertErr,
		SignatureReasonCertRevocationUnknown,
	} {
		want := reason.String()
		for _, sigType := range []int{SigTypeForm, SigTypeDTS, SigTypeUR} {
			result := presentationResult(sigType, reason, "signer evidence", "result evidence")
			if got := result.String(); !strings.Contains(got, "  Reason:            "+want) {
				t.Errorf("type %d changed reason %q:\n%s", sigType, want, got)
			}
		}
	}
}

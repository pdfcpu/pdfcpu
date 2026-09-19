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

package api

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func useEmptySignatureTrustStore(t *testing.T) {
	t.Helper()
	oldDir := model.TrustedCertDir
	model.TrustedCertDir = t.TempDir()
	pdfcpu.InvalidateCertificatePool()
	t.Cleanup(func() {
		model.TrustedCertDir = oldDir
		pdfcpu.InvalidateCertificatePool()
	})
}

type signatureOutputTestCase struct {
	name string
	file string
	want []string
}

func signatureOutputTestCases() []signatureOutputTestCase {
	return []signatureOutputTestCase{
		{
			name: "ETSI.CAdES.detached",
			file: filepath.Join("..", "testdata", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf"),
			want: []string{
				"",
				"1 form signature (authoritative, visible, signed) on page 1",
				"  Integrity: signature authenticated, signed content digest verified",
				"     Status: validity of the signature is unknown",
				"     Reason: signer's certificate or one of its parent certificates has expired",
				"     Signed: 2024-03-04 14:25:54 +0200",
			},
		},
		{
			name: "adbe.pkcs7.detached",
			file: filepath.Join("..", "testdata", "signatures", "adbe.pkcs7.detached", "sample1.pdf"),
			want: []string{
				"",
				"1 form signature (authoritative, visible, signed) on page 1",
				"  Integrity: signature authenticated, signed content digest verified",
				"     Status: validity of the signature is unknown",
				"     Reason: signer's certificate or one of its parent certificates has expired",
				"     Signed: 2009-07-16 10:47:47 -0400",
			},
		},
		{
			name: "adbe.x509.rsa_sha1",
			file: filepath.Join("..", "testdata", "signatures", "adbe.x509.rsa_sha1", "sample01.pdf"),
			want: []string{
				"",
				"1 form signature (authoritative, visible, signed) on page 1",
				"  Integrity: signature unknown, signed content digest unknown",
				"     Status: validity of the signature is unknown",
				"     Reason: signer's certificate is invalid",
				"     Signed: 2009-10-02 00:11:31 +0500",
			},
		},
		{
			name: "adbe.pkcs7.detached.usageRights",
			file: filepath.Join("..", "testdata", "signatures", "adbe.pkcs7.detached", "usageRights.pdf"),
			want: []string{
				"",
				"1 usage rights signature (invisible, signed)",
				"  Integrity: signature authenticated, signed content digest verified",
				"     Status: validity of the signature is unknown",
				"     Reason: signer's certificate path was not resolved using the configured local certificate store",
				"     Signed: 2022-12-15 12:08:57 -0500",
			},
		},
	}
}

func requireSignatureOutput(
	t *testing.T,
	tests []signatureOutputTestCase,
	evaluate func(signatureOutputTestCase, *model.Configuration) ([]string, error),
) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.Offline = true
			got, err := evaluate(tt, conf)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("unexpected API output:\ngot:  %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

// TestValidateSignaturesFileOutput locks down presentation of observed
// signature, certificate, timestamp and revocation evidence together with the
// configuration-dependent local assessment.
func TestValidateSignaturesFileOutput(t *testing.T) {
	useEmptySignatureTrustStore(t)
	requireSignatureOutput(
		t,
		signatureOutputTestCases(),
		func(tt signatureOutputTestCase, conf *model.Configuration) ([]string, error) {
			return ValidateSignaturesFile(t.Context(), tt.file, false, false, conf)
		},
	)
}

// TestValidateSignaturesRawMatchesFileBehavior verifies stream and filename
// inputs produce the same observed evidence, configuration-dependent local
// assessment, and compatible presentation.
func TestValidateSignaturesRawMatchesFileBehavior(t *testing.T) {
	useEmptySignatureTrustStore(t)
	tt := signatureOutputTestCases()[0]
	conf := model.NewDefaultConfiguration()
	conf.Offline = true
	want, err := ValidateSignaturesFile(t.Context(), tt.file, false, false, conf)
	if err != nil {
		t.Fatal(err)
	}
	bb, err := os.ReadFile(tt.file)
	if err != nil {
		t.Fatal(err)
	}
	results, err := ValidateSignaturesRaw(t.Context(), bytes.NewReader(bb), false, conf)
	if err != nil {
		t.Fatal(err)
	}
	got, err := digest(t.Context(), results, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("raw output diverged from file output:\ngot:  %#v\nwant: %#v", got, want)
	}
}

// TestValidateSignaturesRawPreservesPositivePKCS7Evidence verifies the public
// API retains established integrity evidence when local trust remains unknown.
func TestValidateSignaturesRawPreservesPositivePKCS7Evidence(t *testing.T) {
	useEmptySignatureTrustStore(t)
	inFile := filepath.Join("..", "testdata", "signatures", "ETSI.CAdES.detached", "testPAdES_BB.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	conf := model.NewDefaultConfiguration()
	conf.Offline = true
	results, err := ValidateSignaturesRaw(t.Context(), bytes.NewReader(bb), false, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Details.Signers) != 1 {
		t.Fatalf("got %d results, want one result with one signer", len(results))
	}

	result := results[0]
	evidence := result.Details.Signers[0].Evidence
	if evidence.CertificateIdentified != model.True ||
		evidence.SignatureAuthenticated != model.True ||
		evidence.DigestVerified != model.True ||
		evidence.ProfileValidated != model.True {
		t.Fatalf("positive PKCS#7 evidence was not retained: %+v", evidence)
	}
	if result.Status != model.SignatureStatusUnknown {
		t.Fatalf("positive integrity evidence changed local assessment: %s", result.Status)
	}
}

func TestValidateSignaturesRawPreservesUsageRightsEvidence(t *testing.T) {
	useEmptySignatureTrustStore(t)
	inFile := filepath.Join("..", "testdata", "signatures", "adbe.pkcs7.detached", "usageRights.pdf")
	bb, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	conf := model.NewDefaultConfiguration()
	conf.Offline = true
	results, err := ValidateSignaturesRaw(t.Context(), bytes.NewReader(bb), true, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Type != model.SigTypeUR || len(results[0].Details.Signers) != 1 {
		t.Fatalf("unexpected usage-rights result: %+v", results)
	}
	evidence := results[0].Details.Signers[0].Evidence
	if evidence.SignatureAuthenticated != model.True ||
		evidence.DigestVerified != model.True ||
		evidence.ProfileValidated != model.True ||
		evidence.CertificateIdentified != model.True {
		t.Fatalf("usage-rights evidence was not retained: %+v", evidence)
	}
	if results[0].Status != model.SignatureStatusUnknown {
		t.Fatalf("usage-rights evidence changed local assessment: %s", results[0].Status)
	}
}

func TestAggregateSignerEvidence(t *testing.T) {
	tests := []struct {
		name     string
		statuses []int
		want     int
	}{
		{name: "none", want: model.Unknown},
		{name: "all true", statuses: []int{model.True, model.True}, want: model.True},
		{name: "unknown", statuses: []int{model.True, model.Unknown}, want: model.Unknown},
		{name: "false beats unknown", statuses: []int{model.Unknown, model.False}, want: model.False},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var signers []*model.Signer
			for _, status := range tt.statuses {
				signers = append(signers, &model.Signer{Evidence: model.SignerValidationEvidence{
					SignatureAuthenticated: status,
				}})
			}
			got := aggregateSignerEvidence(signers, func(signer *model.Signer) int {
				return signer.Evidence.SignatureAuthenticated
			})
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}

	got := aggregateSignerEvidence([]*model.Signer{nil, {Evidence: model.SignerValidationEvidence{
		SignatureAuthenticated: model.True,
	}}}, func(signer *model.Signer) int {
		return signer.Evidence.SignatureAuthenticated
	})
	if got != model.Unknown {
		t.Fatalf("nil signer: got %d, want unknown", got)
	}
}

func TestSummarizeSignatureEvidence(t *testing.T) {
	svr := &model.SignatureValidationResult{Details: model.SignatureDetails{Signers: []*model.Signer{
		{
			Certificate:           &model.CertificateDetails{Revocation: model.RevocationDetails{Status: model.True}},
			CertificatePathStatus: model.True,
			Evidence: model.SignerValidationEvidence{
				SignatureAuthenticated: model.True,
				DigestVerified:         model.True,
				ProfileValidated:       model.Unknown,
				CertificateIdentified:  model.True,
			},
		},
		{
			CertificatePathStatus: model.False,
			Evidence: model.SignerValidationEvidence{
				SignatureAuthenticated: model.True,
				DigestVerified:         model.False,
				ProfileValidated:       model.True,
				CertificateIdentified:  model.False,
			},
		},
	}}}

	got := summarizeSignatureEvidence(svr)
	want := signatureEvidenceSummary{
		SignatureAuthenticated:   model.True,
		DigestVerified:           model.False,
		ProfileValidated:         model.Unknown,
		CertificateIdentified:    model.False,
		CertificatePathValidated: model.False,
		RevocationStatus:         model.Unknown,
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSignatureEvidenceWording(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "signature true", got: signatureAuthenticationString(model.True), want: "authenticated"},
		{name: "signature false", got: signatureAuthenticationString(model.False), want: "not authentic"},
		{name: "signature unknown", got: signatureAuthenticationString(model.Unknown), want: "unknown"},
		{name: "digest true", got: digestVerificationString(model.True), want: "verified"},
		{name: "digest false", got: digestVerificationString(model.False), want: "mismatch"},
		{name: "profile true", got: profileValidationString(model.True), want: "validated"},
		{name: "certificate false", got: certificateIdentificationString(model.False), want: "not identified"},
		{name: "path true", got: certificatePathString(model.True), want: "resolved using configured local store"},
		{name: "path false", got: certificatePathString(model.False), want: "not resolved using configured local store"},
		{name: "revocation unknown", got: revocationStatusString(model.Unknown), want: "unknown"},
		{
			name: "compact integrity",
			got: compactSignatureIntegrity(signatureEvidenceSummary{
				SignatureAuthenticated: model.True,
				DigestVerified:         model.False,
			}),
			want: "signature authenticated, signed content digest mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestFullSignatureFieldsAlignValuesWithinSection(t *testing.T) {
	got := strings.Join(fullSignatureFields("  ", []fullSignatureField{
		{label: "Cryptographic signature", value: "authenticated"},
		{label: "Signed content digest", value: "verified"},
		{label: "Signature profile", value: "validated"},
	}), "\n")
	want := "  Cryptographic signature: authenticated\n" +
		"  Signed content digest:   verified\n" +
		"  Signature profile:       validated"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestCompactDigestReportsIntegrityEvidence(t *testing.T) {
	tests := []struct {
		name      string
		signature int
		digest    int
		want      string
	}{
		{
			name:      "verified",
			signature: model.True,
			digest:    model.True,
			want:      "  Integrity: signature authenticated, signed content digest verified",
		},
		{
			name:      "failed",
			signature: model.False,
			digest:    model.False,
			want:      "  Integrity: signature not authentic, signed content digest mismatch",
		},
		{
			name: "unknown",
			want: "  Integrity: signature unknown, signed content digest unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.SignatureValidationResult{
				Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
				Details: model.SignatureDetails{Signers: []*model.Signer{{
					Evidence: model.SignerValidationEvidence{
						SignatureAuthenticated: tt.signature,
						DigestVerified:         tt.digest,
					},
				}}},
			}
			lines, err := digest(t.Context(), []*model.SignatureValidationResult{result}, false)
			if err != nil {
				t.Fatal(err)
			}
			if got := lines[2]; got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompactDigestReportsIntegrityForEachSignature(t *testing.T) {
	results := []*model.SignatureValidationResult{
		{
			Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
			Details: model.SignatureDetails{Signers: []*model.Signer{{Evidence: model.SignerValidationEvidence{
				SignatureAuthenticated: model.True,
				DigestVerified:         model.True,
			}}}},
		},
		{
			Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
			Details: model.SignatureDetails{Signers: []*model.Signer{{Evidence: model.SignerValidationEvidence{
				SignatureAuthenticated: model.False,
				DigestVerified:         model.False,
			}}}},
		},
	}
	lines, err := digest(t.Context(), results, false)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, line := range lines {
		if strings.Contains(line, "Integrity:") {
			got = append(got, line)
		}
	}
	want := []string{
		"Integrity: signature authenticated, signed content digest verified",
		"Integrity: signature not authentic, signed content digest mismatch",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestFullDigestReportsCoreEvidenceBeforeAssessment(t *testing.T) {
	result := &model.SignatureValidationResult{
		Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
		Status:    model.SignatureStatusUnknown,
		Details: model.SignatureDetails{Signers: []*model.Signer{{
			Certificate:           &model.CertificateDetails{},
			CertificatePathStatus: model.False,
			Evidence: model.SignerValidationEvidence{
				SignatureAuthenticated: model.True,
				DigestVerified:         model.True,
				ProfileValidated:       model.True,
				CertificateIdentified:  model.True,
			},
		}}},
	}
	lines, err := digest(t.Context(), []*model.SignatureValidationResult{result}, true)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"form signature (invisible, signed)",
		"",
		"Evidence:",
		"  Cryptographic signature: authenticated",
		"  Signed content digest:   verified",
		"  Signature profile:       validated",
		"  Signer certificate:      identified",
		"  Certificate path:        not resolved using configured local store",
		"  Revocation status:       unknown",
		"  Timestamp:               not observed",
		"",
		"Assessment:",
		"  Status:            validity of the signature is unknown",
	}, "\n")
	if !strings.Contains(got, want) {
		t.Fatalf("full output did not report evidence before assessment:\n%s", got)
	}
}

func TestFullDigestReportsSignatureTimestampEvidence(t *testing.T) {
	timestamp := time.Date(2026, time.September, 19, 9, 30, 0, 0, time.FixedZone("test", 2*60*60))
	result := &model.SignatureValidationResult{
		Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
		Details: model.SignatureDetails{Signers: []*model.Signer{{Evidence: model.SignerValidationEvidence{
			Timestamp: model.TimestampValidationEvidence{
				Kind:    model.TimestampKindSignature,
				Present: true,
				Time:    timestamp,
			},
		}}}},
	}
	lines, err := digest(t.Context(), []*model.SignatureValidationResult{result}, true)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"  Timestamp:               signature timestamp observed at 2026-09-19 09:30:00 +0200",
		"",
		"  Timestamp evidence:",
		"    Cryptographic signature: unknown",
		"    Message imprint:         unknown",
		"    Profile:                 unknown",
		"    Certificate path:        unknown",
	}, "\n")
	if !strings.Contains(got, want) {
		t.Fatalf("full output did not report signature timestamp evidence:\n%s", got)
	}
}

func TestFullDigestReportsDocumentTimestampEvidence(t *testing.T) {
	result := &model.SignatureValidationResult{
		Signature: model.Signature{Type: model.SigTypeDTS, Signed: true},
		Details: model.SignatureDetails{Signers: []*model.Signer{{Evidence: model.SignerValidationEvidence{
			Timestamp: model.TimestampValidationEvidence{
				Kind:                     model.TimestampKindDocument,
				Present:                  true,
				SignatureAuthenticated:   model.True,
				DigestVerified:           model.True,
				ProfileValidated:         model.True,
				CertificatePathValidated: model.True,
			},
		}}}},
	}
	lines, err := digest(t.Context(), []*model.SignatureValidationResult{result}, true)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"  Timestamp:               document timestamp observed, time unknown",
		"",
		"  Timestamp evidence:",
		"    Cryptographic signature: authenticated",
		"    Message imprint:         verified",
		"    Profile:                 validated",
		"    Certificate path:        resolved using configured local store",
	}, "\n")
	if !strings.Contains(got, want) {
		t.Fatalf("full output did not report document timestamp evidence:\n%s", got)
	}
}

func TestFullDigestKeepsTimestampEvidencePerSigner(t *testing.T) {
	result := &model.SignatureValidationResult{
		Signature: model.Signature{Type: model.SigTypeForm, Signed: true},
		Details: model.SignatureDetails{Signers: []*model.Signer{
			{},
			{Evidence: model.SignerValidationEvidence{Timestamp: model.TimestampValidationEvidence{
				Kind:    model.TimestampKindSignature,
				Present: true,
			}}},
		}},
	}
	lines, err := digest(t.Context(), []*model.SignatureValidationResult{result}, true)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(lines, "\n")
	want := strings.Join([]string{
		"  Timestamp:               reported per signer",
		"",
		"  Timestamp evidence:",
		"    Signer 1:",
		"      Observation: not observed",
		"    Signer 2:",
		"      Observation: signature timestamp observed, time unknown",
	}, "\n")
	if !strings.Contains(got, want) {
		t.Fatalf("full output collapsed per-signer timestamp evidence:\n%s", got)
	}
}

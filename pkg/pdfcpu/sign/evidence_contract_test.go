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
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestSignatureEvidenceContractFatalPositionalIO verifies positional input
// failures remain fatal and discoverable instead of becoming evidence.
func TestSignatureEvidenceContractFatalPositionalIO(t *testing.T) {
	cause := errors.New("positional read failed")
	fixture := signerPKCS7(t, 1)
	contents := types.HexLiteral(hex.EncodeToString(fixture))
	sigDict := types.Dict{
		"ByteRange": byteRange(
			types.Integer(0),
			types.Integer(1),
			types.Integer(1+len(contents.String())),
			types.Integer(0),
		),
		"Contents": contents,
	}
	result := unknownSignatureResult()

	err := ValidatePKCS7Signatures(
		signErrorReader{err: cause},
		sigDict,
		false,
		false,
		true,
		0,
		x509.NewCertPool(),
		result,
		signatureTestContext(),
	)

	if err == nil {
		t.Fatal("expected fatal positional I/O error")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, %v) = false", err, cause)
	}
	if len(result.Problems) != 0 {
		t.Fatalf("fatal positional I/O became evidence: %v", result.Problems)
	}
}

// TestSignatureEvidenceContractCMSFailures verifies malformed and unsupported
// CMS material remains reportable evidence instead of a fatal operation error.
func TestSignatureEvidenceContractCMSFailures(t *testing.T) {
	t.Run("Malformed", func(t *testing.T) {
		result := unknownSignatureResult()
		err := ValidatePKCS7Signatures(
			bytes.NewReader(nil),
			types.Dict{"Contents": types.HexLiteral("01")},
			false,
			false,
			true,
			0,
			x509.NewCertPool(),
			result,
			signatureTestContext(),
		)

		requireReportableCMSProblem(t, err, result, "pkcs7")
	})

	t.Run("Unsupported", func(t *testing.T) {
		fixture := newDetachedP7SignerFixture(t, nil, nil)
		p7Signer := fixture.signer
		p7Signer.DigestEncryptionAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3}
		result := unknownSignatureResult()

		err := verifyP7Signer(
			p7Signer,
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

		requireReportableCMSProblem(t, err, result, "unsupported")
	})
}

// TestSignatureEvidenceContractAbsentEvidence verifies an absent observation
// cannot establish the positive conclusion that depends on that observation.
func TestSignatureEvidenceContractAbsentEvidence(t *testing.T) {
	t.Run("RevocationIsNotGood", func(t *testing.T) {
		issuer, _, cert := testCurrentCRLChain(t, "Evidence Contract Issuer")
		cert.CRLDistributionPoints = []string{"https://crl.test/unavailable"}
		details, err := processCurrentCRLs(cert, issuer, testCurrentCRLClient(nil))
		if err == nil {
			t.Fatal("expected incomplete revocation observation")
		}
		if details == nil || details.Status != model.Unknown {
			t.Fatalf("incomplete revocation evidence produced Good: %+v", details)
		}
	})

	t.Run("CertificatePathIsNotTrusted", func(t *testing.T) {
		signer := &model.Signer{}
		result := unknownSignatureResult()

		validateCertChains(
			[][]*x509.Certificate{{nil}},
			false,
			x509.NewCertPool(),
			signer,
			nil,
			nil,
			result,
			model.NewDefaultConfiguration(),
		)

		if signer.Certificate == nil {
			t.Fatal("missing certificate-path evidence was not recorded")
		}
		if signer.Certificate.Trust.Status == model.True {
			t.Fatalf("incomplete certificate-path evidence produced Trusted: %+v", signer.Certificate)
		}
	})

	t.Run("IntegrityIsNotUnmodified", func(t *testing.T) {
		fixture := signerPKCS7(t, 1)
		result := unknownSignatureResult()
		err := ValidatePKCS7Signatures(
			bytes.NewReader(nil),
			types.Dict{"Contents": types.HexLiteral(hex.EncodeToString(fixture))},
			false,
			false,
			true,
			0,
			x509.NewCertPool(),
			result,
			signatureTestContext(),
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.DocModified == model.False {
			t.Fatalf("incomplete integrity evidence produced DocModified=False: %+v", result)
		}
	})

	t.Run("TimestampDoesNotPromotePAdES", func(t *testing.T) {
		signer := &model.Signer{PAdES: "B-B"}
		applyTimestampEvidence(timestampEvidence{
			Present: true,
			Err:     errors.New("timestamp token incomplete"),
		}, timestampApplication{
			signer:        signer,
			problemPrefix: "pkcs7",
		})

		if signer.PAdES != "B-B" {
			t.Fatalf("incomplete timestamp evidence promoted PAdES to %q", signer.PAdES)
		}
	})
}

// TestSignatureEvidenceContractCryptographicMismatch verifies a genuine
// cryptographic mismatch is Invalid rather than Internal.
func TestSignatureEvidenceContractCryptographicMismatch(t *testing.T) {
	signer := &model.Signer{}
	result := unknownSignatureResult()

	reportP7SignatureError(
		fmt.Errorf("%w: %w", pkcs7.ErrSignatureMismatch, rsa.ErrVerification),
		signer,
		result,
	)

	if result.Status != model.SignatureStatusInvalid {
		t.Fatalf("got status %s, want %s", result.Status, model.SignatureStatusInvalid)
	}
	if result.Reason != model.SignatureReasonSignatureForged {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonSignatureForged)
	}
	if result.Reason == model.SignatureReasonInternal {
		t.Fatal("cryptographic mismatch was classified as Internal")
	}
}

// TestSignatureEvidenceContractContentMismatch verifies a CMS content-binding
// mismatch is Invalid without being classified as a forged signature.
func TestSignatureEvidenceContractContentMismatch(t *testing.T) {
	signer := &model.Signer{}
	result := unknownSignatureResult()

	reportP7SignatureError(
		fmt.Errorf("verify content: %w", &pkcs7.MessageDigestMismatchError{
			ExpectedDigest: []byte{1},
			ActualDigest:   []byte{2},
		}),
		signer,
		result,
	)

	if result.Status != model.SignatureStatusInvalid ||
		result.Reason != model.SignatureReasonDocModified ||
		result.DocModified != model.True {
		t.Fatalf("unexpected content-mismatch assessment: %+v", result)
	}
	if result.Reason == model.SignatureReasonSignatureForged {
		t.Fatal("content mismatch was classified as a forged signature")
	}
}

// TestSignatureEvidenceContractOperationalRevocationFailure verifies an
// unavailable revocation operation does not claim certificate revocation.
func TestSignatureEvidenceContractOperationalRevocationFailure(t *testing.T) {
	issuer, _, cert := testCurrentCRLChain(t, "Offline Revocation Issuer")
	signer := &model.Signer{}
	certDetails := &model.CertificateDetails{}
	result := unknownSignatureResult()
	conf := model.NewDefaultConfiguration()
	conf.Offline = true

	checkRevocation(
		cert,
		issuer,
		x509.NewCertPool(),
		signer,
		certDetails,
		nil,
		nil,
		result,
		conf,
	)

	if result.Reason == model.SignatureReasonCertRevoked {
		t.Fatalf("operational failure masqueraded as revocation: %+v", result)
	}
	if result.Reason != model.SignatureReasonCertRevocationUnknown {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertRevocationUnknown)
	}
	if certDetails.Revocation.Status != model.Unknown {
		t.Fatalf("operational failure produced revocation status %d", certDetails.Revocation.Status)
	}
	if len(signer.Problems) == 0 {
		t.Fatal("operational revocation failure was not reported")
	}
}

// TestSignatureEvidenceContractRevocationSourceFailuresAreUnknown verifies
// unavailable responders and exhausted distribution points do not masquerade
// as certificate-path or revocation conclusions.
func TestSignatureEvidenceContractRevocationSourceFailuresAreUnknown(t *testing.T) {
	tests := []struct {
		name   string
		issuer bool
		setup  func(*x509.Certificate, *model.Configuration)
		want   string
	}{
		{
			name:   "ResponderUnavailable",
			issuer: false,
			setup: func(_ *x509.Certificate, conf *model.Configuration) {
				conf.PreferredCertRevocationChecker = model.OCSP
			},
			want: "OCSP: certificate issuer unavailable",
		},
		{
			name:   "AllDistributionPointsFail",
			issuer: true,
			setup: func(cert *x509.Certificate, conf *model.Configuration) {
				cert.CRLDistributionPoints = []string{"%"}
				conf.PreferredCertRevocationChecker = model.CRL
			},
			want: "CRL: fetch %",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer, _, cert := testCurrentCRLChain(t, tt.name)
			if !tt.issuer {
				issuer = nil
			}
			signer := &model.Signer{}
			certificate := &model.CertificateDetails{}
			result := unknownSignatureResult()
			conf := model.NewDefaultConfiguration()
			tt.setup(cert, conf)

			checkRevocation(
				cert,
				issuer,
				x509.NewCertPool(),
				signer,
				certificate,
				nil,
				nil,
				result,
				conf,
			)

			if result.Reason != model.SignatureReasonCertRevocationUnknown {
				t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertRevocationUnknown)
			}
			if certificate.Revocation.Status != model.Unknown {
				t.Fatalf("source failure produced status %d", certificate.Revocation.Status)
			}
			if problems := strings.Join(signer.Problems, "\n"); !strings.Contains(problems, tt.want) {
				t.Fatalf("got problems %q, want %q", problems, tt.want)
			}
		})
	}
}

func unknownSignatureResult() *model.SignatureValidationResult {
	return &model.SignatureValidationResult{
		Status:      model.SignatureStatusUnknown,
		Reason:      model.SignatureReasonUnknown,
		DocModified: model.Unknown,
	}
}

func signatureTestContext() *model.Context {
	return &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable:     &model.XRefTable{},
	}
}

func requireReportableCMSProblem(
	t *testing.T,
	err error,
	result *model.SignatureValidationResult,
	want string,
) {
	t.Helper()
	if err != nil {
		t.Fatalf("CMS failure became fatal: %v", err)
	}
	problems := append([]string(nil), result.Problems...)
	for _, signer := range result.Details.Signers {
		problems = append(problems, signer.Problems...)
	}
	if got := strings.Join(problems, "\n"); !strings.Contains(got, want) {
		t.Fatalf("got Problems %q, want %q", got, want)
	}
}

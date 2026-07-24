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
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type fullPKCS7ValidationFixture struct {
	input   []byte
	sigDict types.Dict
	roots   *x509.CertPool
	ctx     *model.Context
}

// TestCAdESBaselineBFullValidationFlow locks profile classification into the
// complete PKCS#7 operation, including local path and revocation assessment.
func TestCAdESBaselineBFullValidationFlow(t *testing.T) {
	tests := []struct {
		name        string
		subFilter   string
		attributes  func(*testing.T) func(*x509.Certificate) []pkcs7.Attribute
		wantStatus  model.SignatureStatus
		wantReason  model.SignatureReason
		wantPAdES   string
		wantProblem bool
	}{
		{
			name:        "SupportedBaselineB",
			subFilter:   "ETSI.CAdES.detached",
			attributes:  validBaselineAttributes,
			wantStatus:  model.SignatureStatusValid,
			wantReason:  model.SignatureReasonDocNotModified,
			wantPAdES:   "B-B",
			wantProblem: false,
		},
		{
			name:        "MissingESSBinding",
			subFilter:   "ETSI.CAdES.detached",
			attributes:  noSignedAttributes,
			wantStatus:  model.SignatureStatusUnknown,
			wantReason:  model.SignatureReasonMalformed,
			wantProblem: true,
		},
		{
			name:        "MalformedESSBinding",
			subFilter:   "ETSI.CAdES.detached",
			attributes:  malformedESSAttributes,
			wantStatus:  model.SignatureStatusUnknown,
			wantReason:  model.SignatureReasonMalformed,
			wantProblem: true,
		},
		{
			name:        "ESSCertificateMismatch",
			subFilter:   "ETSI.CAdES.detached",
			attributes:  mismatchedESSAttributes,
			wantStatus:  model.SignatureStatusInvalid,
			wantReason:  model.SignatureReasonCertInvalid,
			wantProblem: true,
		},
		{
			name:        "UnsupportedESSHash",
			subFilter:   "ETSI.CAdES.detached",
			attributes:  unsupportedESSHashAttributes,
			wantStatus:  model.SignatureStatusUnknown,
			wantReason:  model.SignatureReasonUnsupported,
			wantProblem: true,
		},
		{
			name:        "AdobeDetachedWithoutESS",
			subFilter:   "adbe.pkcs7.detached",
			attributes:  noSignedAttributes,
			wantStatus:  model.SignatureStatusValid,
			wantReason:  model.SignatureReasonDocNotModified,
			wantProblem: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newFullPKCS7ValidationFixture(t, tt.attributes(t))
			result := unknownSignatureResult()
			result.Details.SubFilter = tt.subFilter

			err := ValidatePKCS7Signatures(
				bytes.NewReader(fixture.input),
				fixture.sigDict,
				false,
				false,
				true,
				0,
				fixture.roots,
				result,
				fixture.ctx,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != tt.wantStatus || result.Reason != tt.wantReason {
				t.Fatalf(
					"got status=%s reason=%s, want status=%s reason=%s; Problems=%v",
					result.Status,
					result.Reason,
					tt.wantStatus,
					tt.wantReason,
					result.Problems,
				)
			}
			if len(result.Details.Signers) != 1 {
				t.Fatalf("got %d signers, want 1", len(result.Details.Signers))
			}
			signer := result.Details.Signers[0]
			if signer.PAdES != tt.wantPAdES {
				t.Fatalf("got PAdES %q, want %q; Problems=%v", signer.PAdES, tt.wantPAdES, signer.Problems)
			}
			gotProblem := len(signer.Problems) > 0
			if gotProblem != tt.wantProblem {
				t.Fatalf("got profile problem=%t, want %t; Problems=%v", gotProblem, tt.wantProblem, signer.Problems)
			}
			if tt.wantStatus != model.SignatureStatusValid && signer.PAdES != "" {
				t.Fatalf("failed profile retained PAdES %q", signer.PAdES)
			}
			if tt.wantReason == model.SignatureReasonUnsupported &&
				result.Reason == model.SignatureReasonInternal {
				t.Fatal("unsupported ESS hash was classified as internal")
			}
		})
	}
}

func newFullPKCS7ValidationFixture(
	t *testing.T,
	attributes func(*x509.Certificate) []pkcs7.Attribute,
) fullPKCS7ValidationFixture {
	t.Helper()
	var crl []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(crl)
	}))
	t.Cleanup(server.Close)

	issuer, issuerKey := cadesTestIssuer(t)
	signer, signerKey := cadesTestSigner(t, issuer, issuerKey, server.URL)
	crl = testCurrentCRL(
		t,
		issuer,
		issuerKey,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
		nil,
	)
	content := []byte("complete CAdES validation flow")
	raw := signedPKCS7Fixture(t, content, signer, signerKey, issuer, attributes(signer))
	contents := types.HexLiteral(hex.EncodeToString(raw))
	gap := []byte("<" + contents.Value() + ">")
	input := append(append([]byte(nil), content...), gap...)
	sigDict := types.Dict{
		"ByteRange": byteRange(
			types.Integer(0),
			types.Integer(len(content)),
			types.Integer(len(input)),
			types.Integer(0),
		),
		"Contents": contents,
	}
	roots := x509.NewCertPool()
	roots.AddCert(issuer)
	conf := model.NewDefaultConfiguration()
	conf.PreferredCertRevocationChecker = model.CRL
	conf.AllowedRevocationHosts = []string{revocationTestHost(t, server.URL)}
	return fullPKCS7ValidationFixture{
		input:   input,
		sigDict: sigDict,
		roots:   roots,
		ctx: &model.Context{
			Configuration: conf,
			XRefTable:     &model.XRefTable{},
		},
	}
}

func cadesTestIssuer(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key := testRSAKey(t)
	template := testCertTemplate("CAdES Test Issuer", true)
	template.KeyUsage |= x509.KeyUsageCRLSign
	return testCertificate(t, template, template, &key.PublicKey, key), key
}

func cadesTestSigner(
	t *testing.T,
	issuer *x509.Certificate,
	issuerKey *rsa.PrivateKey,
	crlURL string,
) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key := testRSAKey(t)
	template := testCertTemplate("CAdES Test Signer", false)
	template.CRLDistributionPoints = []string{crlURL}
	return testCertificate(t, template, issuer, &key.PublicKey, issuerKey), key
}

func signedPKCS7Fixture(
	t *testing.T,
	content []byte,
	signer *x509.Certificate,
	signerKey *rsa.PrivateKey,
	issuer *x509.Certificate,
	attributes []pkcs7.Attribute,
) []byte {
	t.Helper()
	digest := sha256.Sum256(content)
	signedData, err := pkcs7.NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	if err := signedData.AddSignerChain(
		signer,
		signerKey,
		digest[:],
		pkcs7.OIDDigestAlgorithmSHA256,
		[]*x509.Certificate{issuer},
		pkcs7.SignerInfoConfig{ExtraSignedAttributes: attributes},
	); err != nil {
		t.Fatal(err)
	}
	raw, err := signedData.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func noSignedAttributes(*testing.T) func(*x509.Certificate) []pkcs7.Attribute {
	return func(*x509.Certificate) []pkcs7.Attribute {
		return nil
	}
}

func malformedESSAttributes(*testing.T) func(*x509.Certificate) []pkcs7.Attribute {
	return func(*x509.Certificate) []pkcs7.Attribute {
		return []pkcs7.Attribute{{
			Type:  oidSigningCertificateV2,
			Value: struct{}{},
		}}
	}
}

func mismatchedESSAttributes(t *testing.T) func(*x509.Certificate) []pkcs7.Attribute {
	t.Helper()
	otherKey := testRSAKey(t)
	otherTemplate := testCertTemplate("Other ESS Certificate", true)
	other := testCertificate(t, otherTemplate, otherTemplate, &otherKey.PublicKey, otherKey)
	return func(*x509.Certificate) []pkcs7.Attribute {
		return validBaselineAttributes(t)(other)
	}
}

func unsupportedESSHashAttributes(t *testing.T) func(*x509.Certificate) []pkcs7.Attribute {
	t.Helper()
	return func(cert *x509.Certificate) []pkcs7.Attribute {
		return []pkcs7.Attribute{
			timestampSigningCertificateAttributeV2(
				t,
				cert,
				crypto.SHA256,
				asn1.ObjectIdentifier{1, 2, 3, 4},
				true,
			),
		}
	}
}

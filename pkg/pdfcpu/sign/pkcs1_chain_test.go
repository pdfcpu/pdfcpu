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
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestBuildP1CertChainsUsesLegacyIntermediates verifies certificates following
// the signer in the legacy Cert array participate in path construction.
func TestBuildP1CertChainsUsesLegacyIntermediates(t *testing.T) {
	root, intermediate, leaf := p1TestChain(t, nil)
	roots := x509.NewCertPool()
	roots.AddCert(root)
	signer := &model.Signer{}
	result := unknownSignatureResult()

	chains := buildP1CertChains(
		leaf,
		[]*x509.Certificate{intermediate},
		roots,
		signer,
		result,
	)
	if len(chains) == 0 || len(chains[0]) != 3 {
		t.Fatalf("legacy intermediate was not used: chains=%v signer=%+v result=%+v", chains, signer, result)
	}
	if result.Details.SignerIdentity != leaf.Subject.CommonName {
		t.Fatalf("got signer identity %q, want %q", result.Details.SignerIdentity, leaf.Subject.CommonName)
	}

	withoutIntermediate := unknownSignatureResult()
	if chains := buildP1CertChains(
		leaf,
		nil,
		roots,
		&model.Signer{},
		withoutIntermediate,
	); len(chains) != 0 {
		t.Fatalf("chain unexpectedly resolved without legacy intermediate: %v", chains)
	}
}

// TestBuildP1CertChainsUsesExplicitDocumentSigningUsage verifies chain
// construction does not inherit x509.Verify's default server-auth usage.
func TestBuildP1CertChainsUsesExplicitDocumentSigningUsage(t *testing.T) {
	root, intermediate, leaf := p1TestChain(t, []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning})
	roots := x509.NewCertPool()
	roots.AddCert(root)
	result := unknownSignatureResult()

	chains := buildP1CertChains(
		leaf,
		[]*x509.Certificate{intermediate},
		roots,
		&model.Signer{},
		result,
	)
	if len(chains) == 0 {
		t.Fatalf("explicit document-signing verification usage rejected chain: %+v", result)
	}
}

// TestP1SigningKeyUsage verifies a present KeyUsage extension must permit
// signature creation and that a missing extension remains acceptable.
func TestP1SigningKeyUsage(t *testing.T) {
	_, _, cert := p1TestChain(t, nil)

	absent := *cert
	absent.Extensions = removeCertificateExtension(absent.Extensions, oidExtensionKeyUsage)
	absent.KeyUsage = 0

	contentCommitment := *cert
	contentCommitment.KeyUsage = x509.KeyUsageContentCommitment

	certSignOnly := *cert
	certSignOnly.KeyUsage = x509.KeyUsageCertSign

	tests := []struct {
		name string
		cert *x509.Certificate
		ok   bool
	}{
		{name: "ExtensionAbsent", cert: &absent, ok: true},
		{name: "DigitalSignature", cert: cert, ok: true},
		{name: "ContentCommitment", cert: &contentCommitment, ok: true},
		{name: "CertSignOnly", cert: &certSignOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("p1SigningCertificate panicked: %v", recovered)
				}
			}()
			_, _, err := p1SigningCertificate([]*x509.Certificate{tt.cert})
			if tt.ok && err != nil {
				t.Fatalf("permitted signing KeyUsage rejected: %v", err)
			}
			if !tt.ok && (err == nil ||
				!strings.Contains(err.Error(), "invalid signing KeyUsage: does not permit signature creation")) {
				t.Fatalf("got %v, want reportable signing KeyUsage problem", err)
			}
		})
	}
}

// TestValidateP1SigningKeyUsageIsReportable verifies a prohibited signing
// KeyUsage remains certificate evidence at the validation boundary.
func TestValidateP1SigningKeyUsageIsReportable(t *testing.T) {
	key := testRSAKey(t)
	template := testCertTemplate("PKCS1 KeyUsage", false)
	template.KeyUsage = x509.KeyUsageCertSign
	cert := testCertificate(t, template, template, &key.PublicKey, key)

	requireP1ValidationProblem(
		t,
		types.Dict{
			"Cert": types.Array{
				types.HexLiteral(hex.EncodeToString(cert.Raw)),
			},
		},
		"invalid signing KeyUsage: does not permit signature creation",
	)
}

// TestBuildP1CertChainsUsesCurrentTime verifies legacy PKCS#1 path assessment
// remains based on the current time.
func TestBuildP1CertChainsUsesCurrentTime(t *testing.T) {
	root, intermediate, leaf := p1TestChain(t, nil)
	expired := *leaf
	expired.NotBefore = time.Now().Add(-2 * time.Hour)
	expired.NotAfter = time.Now().Add(-time.Hour)
	roots := x509.NewCertPool()
	roots.AddCert(root)
	result := unknownSignatureResult()

	chains := buildP1CertChains(
		&expired,
		[]*x509.Certificate{intermediate},
		roots,
		&model.Signer{},
		result,
	)
	if len(chains) != 0 || result.Reason != model.SignatureReasonCertExpired {
		t.Fatalf("expired certificate escaped current-time assessment: chains=%v result=%+v", chains, result)
	}
}

// TestBuildP1CertChainsRejectsMissingIntermediateWithoutPanic verifies malformed
// legacy chain entries remain reportable certificate evidence.
func TestBuildP1CertChainsRejectsMissingIntermediateWithoutPanic(t *testing.T) {
	_, _, leaf := p1TestChain(t, nil)
	signer := &model.Signer{}
	result := unknownSignatureResult()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("buildP1CertChains panicked: %v", recovered)
		}
	}()
	if chains := buildP1CertChains(
		leaf,
		[]*x509.Certificate{nil},
		x509.NewCertPool(),
		signer,
		result,
	); len(chains) != 0 {
		t.Fatalf("missing intermediate produced chains: %v", chains)
	}
	if result.Reason != model.SignatureReasonCertInvalid ||
		len(signer.Problems) != 1 ||
		!strings.Contains(signer.Problems[0], "array index 2: missing certificate") {
		t.Fatalf("missing intermediate was not reportable evidence: signer=%+v result=%+v", signer, result)
	}
}

func p1TestChain(
	t *testing.T,
	leafEKU []x509.ExtKeyUsage,
) (*x509.Certificate, *x509.Certificate, *x509.Certificate) {
	t.Helper()
	rootKey := testRSAKey(t)
	rootTemplate := testCertTemplate("PKCS1 Root", true)
	root := testCertificate(t, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)

	intermediateKey := testRSAKey(t)
	intermediateTemplate := testCertTemplate("PKCS1 Intermediate", true)
	intermediate := testCertificate(
		t,
		intermediateTemplate,
		root,
		&intermediateKey.PublicKey,
		rootKey,
	)

	leafKey := testRSAKey(t)
	leafTemplate := testCertTemplate("PKCS1 Signer", false)
	leafTemplate.ExtKeyUsage = leafEKU
	leaf := testCertificate(
		t,
		leafTemplate,
		intermediate,
		&leafKey.PublicKey,
		intermediateKey,
	)
	return root, intermediate, leaf
}

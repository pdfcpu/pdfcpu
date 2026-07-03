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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func byteRange(values ...types.Object) types.Array {
	return types.Array(values)
}

// TestBytesForByteRange verifies safe extraction of signature byte ranges.
func TestBytesForByteRange(t *testing.T) {
	ra := bytes.NewReader([]byte("0123456789"))
	arr := byteRange(types.Integer(0), types.Integer(3), types.Integer(7), types.Integer(3))

	got, err := bytesForByteRange(ra, arr)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("012789"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestBytesForByteRangeRejectsInvalidRanges verifies malformed range handling.
func TestBytesForByteRangeRejectsInvalidRanges(t *testing.T) {
	ra := bytes.NewReader([]byte("0123456789"))
	tests := []struct {
		name string
		arr  types.Array
	}{
		{"Length", byteRange(types.Integer(0), types.Integer(3))},
		{"Type", byteRange(types.Integer(0), types.Name("x"), types.Integer(7), types.Integer(3))},
		{"Negative", byteRange(types.Integer(0), types.Integer(-1), types.Integer(7), types.Integer(3))},
		{"Overlap", byteRange(types.Integer(0), types.Integer(5), types.Integer(4), types.Integer(3))},
		{"OffsetOverflow", byteRange(types.Integer(math.MaxInt), types.Integer(1), types.Integer(math.MaxInt), types.Integer(0))},
		{"OutOfBounds", byteRange(types.Integer(0), types.Integer(3), types.Integer(7), types.Integer(100))},
		{"HugeOutOfBounds", byteRange(types.Integer(0), types.Integer(0), types.Integer(0), types.Integer(math.MaxInt))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := bytesForByteRange(ra, tt.arr); err == nil {
				t.Fatal("expected invalid ByteRange error")
			}
		})
	}
}

// TestHandleCertVerifyErrReportsUnknownAuthority verifies trust-store failures are user actionable.
func TestHandleCertVerifyErrReportsUnknownAuthority(t *testing.T) {
	cert := &x509.Certificate{SerialNumber: big.NewInt(42)}
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	err := errors.Wrap(x509.UnknownAuthorityError{Cert: cert}, "wrapped")

	handleCertVerifyErr(err, cert, signer, result)

	if result.Reason != model.SignatureReasonCertNotTrusted {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertNotTrusted)
	}
	if len(signer.Problems) != 2 {
		t.Fatalf("got %d problems, want 2: %v", len(signer.Problems), signer.Problems)
	}
	if !strings.Contains(signer.Problems[0], "certificate chain is not trusted") {
		t.Fatalf("missing not trusted problem: %q", signer.Problems[0])
	}
	if signer.Problems[1] != certImportHint {
		t.Fatalf("got hint %q, want %q", signer.Problems[1], certImportHint)
	}
}

// TestHandleCertParseErrReportsMalformedCertificate verifies x509 parse errors are classified as certificate problems.
func TestHandleCertParseErrReportsMalformedCertificate(t *testing.T) {
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	err := errors.New("x509: SAN rfc822Name is malformed")

	handleCertParseErr(err, result)

	if result.Reason != model.SignatureReasonCertInvalid {
		t.Fatalf("got reason %s, want %s", result.Reason, model.SignatureReasonCertInvalid)
	}
	if len(result.Problems) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(result.Problems), result.Problems)
	}
	if !strings.Contains(result.Problems[0], "certificate data is malformed") {
		t.Fatalf("missing malformed certificate problem: %q", result.Problems[0])
	}
}

// TestBuildP7CertChainsSetsPrimarySignerIdentity verifies signer identity reporting.
func TestBuildP7CertChainsSetsPrimarySignerIdentity(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	roots := x509.NewCertPool()
	roots.AddCert(root)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	chains := buildP7CertChains(true, leaf, []*x509.Certificate{root, leaf}, roots, &model.Signer{}, nil, result)

	if len(chains) == 0 {
		t.Fatal("expected trusted certificate chain")
	}
	if got := result.Details.SignerIdentity; got != "Alice Signer" {
		t.Fatalf("got signer identity %q, want Alice Signer", got)
	}
}

// TestBuildP7CertChainsDoesNotSetIdentityForUntrustedCert verifies issue 1271 behavior.
func TestBuildP7CertChainsDoesNotSetIdentityForUntrustedCert(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}

	chains := buildP7CertChains(true, leaf, []*x509.Certificate{root, leaf}, x509.NewCertPool(), &model.Signer{}, nil, result)

	if len(chains) != 0 {
		t.Fatal("expected no trusted certificate chain")
	}
	if got := result.Details.SignerIdentity; got != "" {
		t.Fatalf("got signer identity %q, want empty", got)
	}
}

// TestBuildP7CertChainsDoesNotOverwriteSignerIdentity verifies secondary signer handling.
func TestBuildP7CertChainsDoesNotOverwriteSignerIdentity(t *testing.T) {
	root, leaf := testCertChain(t, "Root CA", "Alice Signer")
	roots := x509.NewCertPool()
	roots.AddCert(root)
	result := &model.SignatureValidationResult{Reason: model.SignatureReasonUnknown}
	result.Details.SignerIdentity = "Primary Signer"

	chains := buildP7CertChains(false, leaf, []*x509.Certificate{root, leaf}, roots, &model.Signer{}, nil, result)

	if len(chains) == 0 {
		t.Fatal("expected trusted certificate chain")
	}
	if got := result.Details.SignerIdentity; got != "Primary Signer" {
		t.Fatalf("got signer identity %q, want Primary Signer", got)
	}
}

func testCertChain(t *testing.T, rootCN, leafCN string) (*x509.Certificate, *x509.Certificate) {
	t.Helper()

	rootKey := testRSAKey(t)
	rootTemplate := testCertTemplate(rootCN, true)
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey := testRSAKey(t)
	leafTemplate := testCertTemplate(leafCN, false)
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	return root, leaf
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testCertTemplate(commonName string, ca bool) *x509.Certificate {
	keyUsage := x509.KeyUsageDigitalSignature
	if ca {
		keyUsage |= x509.KeyUsageCertSign
	}
	return &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
		IsCA:                  ca,
	}
}

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

package pkcs7

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestGetCertFromCertsRejectsMissingSignerIdentifier(t *testing.T) {
	var got error
	requireErrorWithoutPanic(t, "find signer certificate", func() error {
		_, got = GetCertFromCertsByIssuerAndSerial(nil, issuerAndSerial{})
		return got
	})
	if !errors.Is(got, ErrMissingSignerIdentifier) {
		t.Fatalf("got %v, want ErrMissingSignerIdentifier", got)
	}
}

func TestGetCertFromCertsSkipsNilCertificates(t *testing.T) {
	issuer := []byte{0x30, 0x00}
	cert := &x509.Certificate{
		RawIssuer:    issuer,
		SerialNumber: big.NewInt(7),
	}
	ias := issuerAndSerial{
		IssuerName:   asn1.RawValue{FullBytes: issuer},
		SerialNumber: big.NewInt(7),
	}

	got, err := GetCertFromCertsByIssuerAndSerial(
		[]*x509.Certificate{nil, {}, cert},
		ias,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != cert {
		t.Fatalf("got %p, want %p", got, cert)
	}
}

func TestGetCertFromCertsReturnsNilWhenNoCertificateMatches(t *testing.T) {
	ias := issuerAndSerial{
		IssuerName:   asn1.RawValue{FullBytes: []byte{0x30, 0x00}},
		SerialNumber: big.NewInt(7),
	}
	cert, err := GetCertFromCertsByIssuerAndSerial(nil, ias)
	if err != nil {
		t.Fatal(err)
	}
	if cert != nil {
		t.Fatalf("got %v, want no matching certificate", cert)
	}
}

func TestVerifyCertChainRejectsMissingCertificatesWithoutPanic(t *testing.T) {
	requireErrorWithoutPanic(t, "end-entity certificate is missing", func() error {
		_, err := VerifyCertChain(nil, nil, x509.NewCertPool(), time.Time{})
		return err
	})

	_, leaf := testCertificateChain(t)
	requireErrorWithoutPanic(t, "intermediate certificate index 1 is missing", func() error {
		_, err := VerifyCertChain(leaf, []*x509.Certificate{nil}, x509.NewCertPool(), time.Time{})
		return err
	})
}

func TestVerifyCertChainPreservesUnknownAuthorityCause(t *testing.T) {
	_, leaf := testCertificateChain(t)
	_, err := VerifyCertChain(
		leaf,
		nil,
		x509.NewCertPool(),
		time.Date(2020, time.June, 1, 0, 0, 0, 0, time.UTC),
	)
	var unknownAuthority x509.UnknownAuthorityError
	if !errors.As(err, &unknownAuthority) {
		t.Fatalf("got %v, want x509.UnknownAuthorityError", err)
	}
	if !strings.Contains(err.Error(), "pkcs7: verify certificate chain") {
		t.Fatalf("got %v, want certificate-chain phase", err)
	}
}

func TestVerifyCertChainUsesValidationTime(t *testing.T) {
	root, leaf := testCertificateChain(t)
	roots := x509.NewCertPool()
	roots.AddCert(root)

	chains, err := VerifyCertChain(
		leaf,
		nil,
		roots,
		time.Date(2020, time.June, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("expected historical verification to succeed: %v", err)
	}
	if len(chains) == 0 {
		t.Fatal("expected certificate chain")
	}

	_, err = VerifyCertChain(
		leaf,
		nil,
		roots,
		time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC),
	)
	var invalid x509.CertificateInvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("got %v, want x509.CertificateInvalidError", err)
	}
}

func testCertificateChain(t *testing.T) (*x509.Certificate, *x509.Certificate) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root"},
		NotBefore:             time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(
		rand.Reader,
		rootTemplate,
		rootTemplate,
		&rootKey.PublicKey,
		rootKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Signer"},
		NotBefore:    rootTemplate.NotBefore,
		NotAfter:     rootTemplate.NotAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	leafDER, err := x509.CreateCertificate(
		rand.Reader,
		leafTemplate,
		root,
		&leafKey.PublicKey,
		rootKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	return root, leaf
}

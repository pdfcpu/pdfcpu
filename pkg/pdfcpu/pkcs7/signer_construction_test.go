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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestAddSignerChainRejectsMissingInputsWithoutPanic(t *testing.T) {
	cert, key := signerCertificate(t, "Signer")
	digest := sha256.Sum256([]byte("content"))

	var missing *SignedData
	requireErrorWithoutPanic(t, "signed data is missing", func() error {
		return missing.AddSigner(cert, key, digest[:], OIDDigestAlgorithmSHA256, SignerInfoConfig{})
	})

	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	requireErrorWithoutPanic(t, "signer certificate is missing", func() error {
		return sd.AddSigner(nil, key, digest[:], OIDDigestAlgorithmSHA256, SignerInfoConfig{})
	})
	requireErrorWithoutPanic(t, "parent certificate index 2 is missing", func() error {
		return sd.AddSignerChain(
			cert,
			key,
			digest[:],
			OIDDigestAlgorithmSHA256,
			[]*x509.Certificate{cert, nil},
			SignerInfoConfig{},
		)
	})
}

func TestAddSignerPreservesUnsupportedKeyClassification(t *testing.T) {
	cert, _ := signerCertificate(t, "Signer")
	digest := sha256.Sum256([]byte("content"))
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}

	err = sd.AddSigner(cert, struct{}{}, digest[:], OIDDigestAlgorithmSHA256, SignerInfoConfig{})
	if !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", err)
	}
}

func TestOIDForEncryptionAlgorithmRejectsUnsupportedDigest(t *testing.T) {
	_, key := signerCertificate(t, "Signer")
	_, err := OIDForEncryptionAlgorithm(key, asn1.ObjectIdentifier{1, 2, 3})
	if !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", err)
	}
}

func TestAddSignerRejectsTypedNilPrivateKeyWithoutPanic(t *testing.T) {
	cert, _ := signerCertificate(t, "Signer")
	digest := sha256.Sum256([]byte("content"))
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}

	requireErrorWithoutPanic(t, "RSA private key is missing", func() error {
		return sd.AddSigner(
			cert,
			(*rsa.PrivateKey)(nil),
			digest[:],
			OIDDigestAlgorithmSHA256,
			SignerInfoConfig{},
		)
	})
}

func TestAddSignerRejectsMismatchedPrivateKey(t *testing.T) {
	cert, _ := signerCertificate(t, "Signer")
	_, otherKey := signerCertificate(t, "Other")
	digest := sha256.Sum256([]byte("content"))
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}

	err = sd.AddSigner(cert, otherKey, digest[:], OIDDigestAlgorithmSHA256, SignerInfoConfig{})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("private key does not match")) {
		t.Fatalf("expected key-mismatch error, got %v", err)
	}
}

func TestAddSignerFailureIsTransactional(t *testing.T) {
	cert, key := signerCertificate(t, "Signer")
	digest := sha256.Sum256([]byte("content"))
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	config := SignerInfoConfig{
		ExtraSignedAttributes: []Attribute{{
			Type:  OIDData,
			Value: make(chan int),
		}},
	}

	err = sd.AddSigner(cert, key, digest[:], OIDDigestAlgorithmSHA256, config)
	if err == nil {
		t.Fatal("expected attribute construction error")
	}
	if len(sd.sd.DigestAlgorithmIdentifiers) != 0 ||
		len(sd.sd.SignerInfos) != 0 ||
		len(sd.certs) != 0 {
		t.Fatalf("failed addition mutated SignedData: %+v, certs=%d", sd.sd, len(sd.certs))
	}
}

func TestAddSignerRejectsDuplicateMandatoryAttributeWithoutMutation(t *testing.T) {
	cert, key := signerCertificate(t, "Signer")
	digest := sha256.Sum256([]byte("content"))
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	config := SignerInfoConfig{
		ExtraSignedAttributes: []Attribute{{
			Type:  oidAttributeMessageDigest,
			Value: digest[:],
		}},
	}

	err = sd.AddSigner(cert, key, digest[:], OIDDigestAlgorithmSHA256, config)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("duplicates a mandatory attribute")) {
		t.Fatalf("expected duplicate mandatory-attribute error, got %v", err)
	}
	if len(sd.sd.DigestAlgorithmIdentifiers) != 0 ||
		len(sd.sd.SignerInfos) != 0 ||
		len(sd.certs) != 0 {
		t.Fatal("failed addition mutated SignedData")
	}
}

func TestAddSignerRejectsWrongDigestLengthWithoutMutation(t *testing.T) {
	cert, key := signerCertificate(t, "Signer")
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}

	err = sd.AddSigner(cert, key, []byte("short"), OIDDigestAlgorithmSHA256, SignerInfoConfig{})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("message digest length")) {
		t.Fatalf("expected digest-length error, got %v", err)
	}
	if len(sd.sd.DigestAlgorithmIdentifiers) != 0 ||
		len(sd.sd.SignerInfos) != 0 ||
		len(sd.certs) != 0 {
		t.Fatal("failed addition mutated SignedData")
	}
}

func TestAddSignerUsesCertificateIssuerForSignerIdentifier(t *testing.T) {
	cert, key := signerCertificate(t, "Signer")
	parent, _ := signerCertificate(t, "Unrelated Parent")
	digest := sha256.Sum256([]byte("content"))
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}

	err = sd.AddSignerChain(
		cert,
		key,
		digest[:],
		OIDDigestAlgorithmSHA256,
		[]*x509.Certificate{parent},
		SignerInfoConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := sd.sd.SignerInfos[0].IssuerAndSerialNumber.IssuerName.FullBytes
	if !bytes.Equal(got, cert.RawIssuer) {
		t.Fatalf("got issuer %x, want signer certificate issuer %x", got, cert.RawIssuer)
	}
}

func signerCertificate(t *testing.T, commonName string) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

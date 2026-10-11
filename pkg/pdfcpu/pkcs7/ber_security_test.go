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
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"math/big"
	"strings"
	"testing"
	"time"
)

func (o failingASN1Object) encodedLen() (int, error) {
	return 2, nil
}

func nestedSequenceBER(t *testing.T, depth int, indefinite bool) []byte {
	t.Helper()
	ber := []byte{0x02, 0x01, 0x01}
	for range depth {
		var outer bytes.Buffer
		outer.WriteByte(0x30)
		if indefinite {
			outer.WriteByte(0x80)
		} else {
			if err := encodeLength(&outer, len(ber)); err != nil {
				t.Fatal(err)
			}
		}
		outer.Write(ber)
		if indefinite {
			outer.Write([]byte{0, 0})
		}
		ber = outer.Bytes()
	}
	return ber
}

func TestBERConstructedDepthLimit(t *testing.T) {
	for _, tt := range []struct {
		name       string
		indefinite bool
	}{
		{"definite", false},
		{"indefinite", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ber2der(nestedSequenceBER(t, maxBERDepth, tt.indefinite)); err != nil {
				t.Fatalf("depth %d: %v", maxBERDepth, err)
			}
			_, err := ber2der(nestedSequenceBER(t, maxBERDepth+1, tt.indefinite))
			if err == nil || !strings.Contains(err.Error(), "BER depth exceeds limit 64") {
				t.Fatalf("got %v, want BER depth limit error", err)
			}
		})
	}
}

func TestBERObjectCountLimit(t *testing.T) {
	ber := make([]byte, 0, 4+maxBERObjects*2)
	ber = append(ber, 0x30, 0x80)
	for range maxBERObjects {
		ber = append(ber, 0x05, 0x00)
	}
	ber = append(ber, 0, 0)
	_, err := ber2der(ber)
	if err == nil || !strings.Contains(err.Error(), "BER object count exceeds limit") {
		t.Fatalf("got %v, want BER object count limit error", err)
	}
}

func TestDEROutputSizeLimit(t *testing.T) {
	if n, err := encodedObjectLength(1, maxBERBytes-6); err != nil || n != maxBERBytes {
		t.Fatalf("got length %d, error %v, want limit %d", n, err, maxBERBytes)
	}
	if _, err := encodedObjectLength(1, maxBERBytes-5); err == nil {
		t.Fatal("expected DER output size limit error")
	}
}

func largeCRLCertificate(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	cert, key := signerCertificate(t, "large CRL signer")
	cert.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	cert.BasicConstraintsValid = true
	cert.IsCA = true
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err = x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

func largeCRLCMS(t *testing.T, count int) ([]byte, *x509.Certificate) {
	t.Helper()
	cert, key := largeCRLCertificate(t)
	now := time.Date(2026, time.October, 9, 0, 0, 0, 0, time.UTC)
	entries := make([]x509.RevocationListEntry, count)
	for i := range entries {
		entries[i] = x509.RevocationListEntry{
			SerialNumber: big.NewInt(int64(1_000_000 + i)), RevocationTime: now, ReasonCode: 1,
		}
	}
	crl, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number: big.NewInt(1), ThisUpdate: now, NextUpdate: now.Add(time.Hour), RevokedCertificateEntries: entries,
	}, cert, key)
	if err != nil {
		t.Fatal(err)
	}
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("hello"))
	if err := sd.AddSigner(cert, key, digest[:], OIDDigestAlgorithmSHA256, SignerInfoConfig{}); err != nil {
		t.Fatal(err)
	}
	sd.sd.CRLs = []asn1.RawValue{{FullBytes: crl}}
	cms, err := sd.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return cms, cert
}

func indefiniteOuterSequence(t *testing.T, der []byte) []byte {
	t.Helper()
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(der, &raw); err != nil {
		t.Fatal(err)
	}
	return append(append([]byte{0x30, 0x80}, raw.Bytes...), 0, 0)
}

// TestParseLargeCRL verifies valid CMS with more than 100,000 ASN.1 values retains all revocation entries and signatures.
func TestParseLargeCRL(t *testing.T) {
	const count = 15_000
	der, cert := largeCRLCMS(t, count)
	for _, input := range [][]byte{der, indefiniteOuterSequence(t, der), append(bytes.Clone(der), 0, 0)} {
		normalized, err := ber2der(input)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(normalized, der) {
			t.Fatal("normalization changed the signed CMS bytes")
		}
		p7, err := Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(p7.CRLs) != 1 || len(p7.CRLs[0].RevokedCertificateEntries) != count {
			t.Fatal("revocation entries were lost")
		}
		for i, entry := range p7.CRLs[0].RevokedCertificateEntries {
			if entry.SerialNumber.Int64() != int64(1_000_000+i) || entry.ReasonCode != 1 {
				t.Fatalf("revocation entry %d was changed", i)
			}
		}
		if err := p7.CRLs[0].CheckSignatureFrom(cert); err != nil {
			t.Fatal(err)
		}
		if len(p7.Signers) != 1 {
			t.Fatal("signer was lost")
		}
		if err := CheckSignature(cert, p7.Signers[0], []byte("hello")); err != nil {
			t.Fatal(err)
		}
	}
}

func definiteBERSequence(t *testing.T, content []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	out.WriteByte(0x30)
	if err := encodeLength(&out, len(content)); err != nil {
		t.Fatal(err)
	}
	out.Write(content)
	return out.Bytes()
}

// TestBERDefiniteSubtreeAllocations bounds memory overhead independently of the number of ASN.1 values.
func TestBERDefiniteSubtreeAllocations(t *testing.T) {
	der := definiteBERSequence(t, bytes.Repeat([]byte{0x05, 0x00}, maxBERObjects+1))
	// The mixed case retains an indefinite parent around a large definite subtree.
	mixed := append(append([]byte{0x30, 0x80}, der...), 0, 0)
	for _, input := range [][]byte{der, mixed} {
		allocations := testing.AllocsPerRun(3, func() {
			if _, err := ber2der(input); err != nil {
				t.Fatal(err)
			}
		})
		if allocations > 32 {
			t.Fatalf("conversion allocated %.0f objects, want <=32", allocations)
		}
	}
}

// TestBERDefiniteSubtreeRejectsMalformedTail ensures large definite subtrees still validate every nested value.
func TestBERDefiniteSubtreeRejectsMalformedTail(t *testing.T) {
	prefix := bytes.Repeat([]byte{0x05, 0x00}, maxBERObjects+1)
	for _, tail := range [][]byte{
		{0x1f},                         // Truncated high tag.
		{0x04, 0x02, 0x01},             // Content length exceeds input.
		{0x30, 0x02, 0x02, 0x01, 0x01}, // Child crosses its parent's boundary.
		{0x00, 0x00},                   // EOC is invalid inside definite content.
		{0x04, 0x80, 0x00, 0x00},       // A primitive value cannot have indefinite length.
	} {
		input := definiteBERSequence(t, append(bytes.Clone(prefix), tail...))
		if _, err := ber2der(input); err == nil {
			t.Fatalf("accepted malformed tail %x", tail)
		}
	}
}

// TestBERDefiniteSubtreeConversion preserves normalization when a nested value still requires rewriting.
func TestBERDefiniteSubtreeConversion(t *testing.T) {
	for _, tt := range []struct{ input, want []byte }{
		{[]byte{0x30, 0x04, 0x02, 0x81, 0x01, 0x01}, []byte{0x30, 0x03, 0x02, 0x01, 0x01}},
		{[]byte{0x30, 0x07, 0x30, 0x80, 0x02, 0x01, 0x01, 0, 0}, []byte{0x30, 0x05, 0x30, 0x03, 0x02, 0x01, 0x01}},
	} {
		got, err := ber2der(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, tt.want) {
			t.Fatalf("got %x, want %x", got, tt.want)
		}
	}
}

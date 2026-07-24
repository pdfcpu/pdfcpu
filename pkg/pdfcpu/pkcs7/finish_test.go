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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"strings"
	"testing"
)

func TestAddCertificateRejectsMissingInputsWithoutPanic(t *testing.T) {
	var sd *SignedData
	requireErrorWithoutPanic(t, "signed data is missing", func() error {
		return sd.AddCertificate(&x509.Certificate{Raw: []byte{0x01}})
	})

	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	requireErrorWithoutPanic(t, "certificate is missing", func() error {
		return sd.AddCertificate(nil)
	})
	requireErrorWithoutPanic(t, "certificate DER is missing", func() error {
		return sd.AddCertificate(&x509.Certificate{})
	})
	var got error
	requireErrorWithoutPanic(t, "add certificate: parse DER", func() error {
		got = sd.AddCertificate(&x509.Certificate{Raw: []byte{0x30, 0x01}})
		return got
	})
	if !errors.Is(got, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", got)
	}
	if !strings.Contains(got.Error(), "x509:") {
		t.Fatalf("got %v, want preserved X.509 cause", got)
	}
}

func TestMarshalCertificatesRetainsIndexAndX509Cause(t *testing.T) {
	var got error
	requireErrorWithoutPanic(t, "certificate index 2: parse DER", func() error {
		_, got = marshalCertificates([]*x509.Certificate{
			{Raw: signerCertificateDER(t, "Valid")},
			{Raw: []byte{0x30, 0x01}},
		})
		return got
	})
	if !errors.Is(got, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", got)
	}
	if !strings.Contains(got.Error(), "x509:") {
		t.Fatalf("got %v, want preserved X.509 cause", got)
	}
}

func TestMarshalCertificatesSortsAndDeduplicates(t *testing.T) {
	certA, _ := signerCertificate(t, "A")
	certB, _ := signerCertificate(t, "B")
	raw, err := marshalCertificates([]*x509.Certificate{certB, certA, certB})
	if err != nil {
		t.Fatal(err)
	}
	certs, _, err := parseRawCertificateSet(raw.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(certs) != 2 {
		t.Fatalf("got %d certificates, want two unique certificates", len(certs))
	}
	if bytes.Compare(certs[0].Raw, certs[1].Raw) >= 0 {
		t.Fatal("certificate set is not in DER order")
	}
}

func TestFinishRejectsNilReceiverWithoutPanic(t *testing.T) {
	var sd *SignedData
	requireErrorWithoutPanic(t, "finish signed data: signed data is missing", func() error {
		_, err := sd.Finish()
		return err
	})
}

func TestFinishFailureDoesNotMutateSignedData(t *testing.T) {
	cert, _ := signerCertificate(t, "Signer")
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	if err := sd.AddCertificate(cert); err != nil {
		t.Fatal(err)
	}
	sd.sd.DigestAlgorithmIdentifiers = []pkix.AlgorithmIdentifier{{
		Algorithm: asn1.ObjectIdentifier{3},
	}}

	_, err = sd.Finish()
	if err == nil {
		t.Fatal("expected signed-data encoding error")
	}
	var structuralErr asn1.StructuralError
	if !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", err)
	}
	if len(sd.sd.Certificates.Raw) != 0 {
		t.Fatal("failed Finish mutated the SignedData certificate field")
	}
}

func TestFinishCertificateOnlyRoundTrip(t *testing.T) {
	cert, _ := signerCertificate(t, "Signer")
	sd, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	if err := sd.AddCertificate(cert); err != nil {
		t.Fatal(err)
	}
	if err := sd.AddCertificate(cert); err != nil {
		t.Fatal(err)
	}

	data, err := sd.Finish()
	if err != nil {
		t.Fatal(err)
	}
	p7, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(p7.Certificates) != 1 {
		t.Fatalf("got %d certificates, want one unique certificate", len(p7.Certificates))
	}
	if len(p7.Signers) != 0 {
		t.Fatalf("got %d signers, want zero", len(p7.Signers))
	}
}

func signerCertificateDER(t *testing.T, commonName string) []byte {
	t.Helper()
	cert, _ := signerCertificate(t, commonName)
	return cert.Raw
}

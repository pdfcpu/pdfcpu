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
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"strings"
	"testing"
)

func TestParseSignedDataRejectsTrailingData(t *testing.T) {
	data := append(marshalSignedData(t, nil, nil), 0x05, 0x00)
	var got error
	requireErrorWithoutPanic(t, "decode signed data", func() error {
		_, got = parseSignedData(data)
		return got
	})
	var syntaxErr asn1.SyntaxError
	if !errors.As(got, &syntaxErr) {
		t.Fatalf("got %v, want preserved ASN.1 syntax cause", got)
	}
}

func TestParseSignedDataConcatenatesConstructedOctetString(t *testing.T) {
	content := marshalRawValue(t, asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagOctetString,
		IsCompound: true,
		Bytes: append(
			marshalRawValue(t, asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte("first")}),
			marshalRawValue(t, asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte("second")})...,
		),
	})

	p7, err := parseSignedData(marshalSignedData(t, content, nil))
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("firstsecond"); !bytes.Equal(p7.Content, want) {
		t.Fatalf("got %q, want %q", p7.Content, want)
	}
}

func TestParseSignedDataRejectsMalformedOctetStringSegment(t *testing.T) {
	content := marshalRawValue(t, asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagOctetString,
		IsCompound: true,
		Bytes: append(
			marshalRawValue(t, asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte("valid")}),
			[]byte{0x02, 0x01, 0x01}...,
		),
	})

	var got error
	requireErrorWithoutPanic(t, "OCTET STRING segment 2", func() error {
		_, got = parseSignedData(marshalSignedData(t, content, nil))
		return got
	})
	var structuralErr asn1.StructuralError
	if !errors.As(got, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", got)
	}
}

func TestParseSignedDataRejectsTrailingEncapsulatedContent(t *testing.T) {
	content := append(
		marshalRawValue(t, asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte("content")}),
		0x05, 0x00,
	)
	var got error
	requireErrorWithoutPanic(t, "trailing data after encapsulated content", func() error {
		_, got = parseSignedData(marshalSignedData(t, content, nil))
		return got
	})
	var syntaxErr asn1.SyntaxError
	if !errors.As(got, &syntaxErr) {
		t.Fatalf("got %v, want preserved ASN.1 syntax cause", got)
	}
}

func TestParseSignedDataRetainsCRLIndexAndCause(t *testing.T) {
	crls := []asn1.RawValue{{FullBytes: []byte{0x30, 0x03, 0x02, 0x01, 0x01}}}
	var got error
	requireErrorWithoutPanic(t, "CRL index 1: parse revocation list", func() error {
		_, got = parseSignedData(marshalSignedData(t, nil, crls))
		return got
	})
	if !strings.Contains(got.Error(), "x509:") {
		t.Fatalf("got %v, want preserved X.509 cause", got)
	}
}

func TestParseSignedDataAllowsZeroSigners(t *testing.T) {
	p7, err := parseSignedData(marshalSignedData(t, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(p7.Signers) != 0 {
		t.Fatalf("got %d signers, want zero", len(p7.Signers))
	}
}

func TestParseSignedDataRequiresSignerDigestInAlgorithmSet(t *testing.T) {
	data, err := asn1.Marshal(signedData{
		Version: 1,
		DigestAlgorithmIdentifiers: []pkix.AlgorithmIdentifier{{
			Algorithm: OIDDigestAlgorithmSHA256,
		}},
		ContentInfo: contentInfo{ContentType: OIDData},
		SignerInfos: []SignerInfo{{
			IssuerAndSerialNumber: issuerAndSerial{
				IssuerName:   asn1.RawValue{FullBytes: []byte{0x30, 0x00}},
				SerialNumber: big.NewInt(1),
			},
			DigestAlgorithm: pkix.AlgorithmIdentifier{
				Algorithm: OIDDigestAlgorithmSHA384,
			},
			DigestEncryptionAlgorithm: pkix.AlgorithmIdentifier{
				Algorithm: OIDEncryptionAlgorithmRSA,
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = parseSignedData(data)
	if !errors.Is(err, ErrAlgorithmMismatch) {
		t.Fatalf("got %v, want ErrAlgorithmMismatch", err)
	}
	if !strings.Contains(err.Error(), "signer 1 digest OID") ||
		!strings.Contains(err.Error(), "SignedData digestAlgorithms") {
		t.Fatalf("missing CMS algorithm context: %v", err)
	}
}

func marshalSignedData(t *testing.T, content []byte, crls []asn1.RawValue) []byte {
	t.Helper()
	info := contentInfo{ContentType: OIDData}
	if content != nil {
		info.Content = asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      content,
		}
	}
	data, err := asn1.Marshal(signedData{
		Version:     1,
		ContentInfo: info,
		CRLs:        crls,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func marshalRawValue(t *testing.T, value asn1.RawValue) []byte {
	t.Helper()
	data, err := asn1.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

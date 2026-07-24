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
	"strings"
	"testing"
)

type malformedCertificateContentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

type malformedCertificateRawSet struct {
	Raw asn1.RawContent
}

type malformedCertificateSignedData struct {
	Version                    int                        `asn1:"default:1"`
	DigestAlgorithmIdentifiers []pkix.AlgorithmIdentifier `asn1:"set"`
	ContentInfo                malformedCertificateContentInfo
	Certificates               malformedCertificateRawSet `asn1:"optional,tag:0"`
	SignerInfos                []asn1.RawValue            `asn1:"set"`
}

func malformedCertificateMessage(t *testing.T) []byte {
	t.Helper()
	signedData, err := asn1.Marshal(malformedCertificateSignedData{
		Version:     1,
		ContentInfo: malformedCertificateContentInfo{ContentType: OIDData},
		Certificates: malformedCertificateRawSet{
			Raw: []byte{0xa0, 0x05, 0x30, 0x03, 0x02, 0x01, 0x01},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := asn1.Marshal(malformedCertificateContentInfo{
		ContentType: OIDSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      signedData,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func TestParseClassifiesMalformedCertificate(t *testing.T) {
	_, err := Parse(malformedCertificateMessage(t))
	if err == nil {
		t.Fatal("expected certificate parse error")
	}
	if !errors.Is(err, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", err)
	}

	var parseErr *CertificateParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("got %T, want *CertificateParseError", err)
	}
	cause := errors.Unwrap(parseErr)
	if cause == nil {
		t.Fatal("expected underlying certificate parse cause")
	}
	if !errors.Is(err, cause) {
		t.Fatalf("underlying cause %v is not discoverable through errors.Is", cause)
	}
	for _, want := range []string{
		"pkcs7: parse certificate set",
		"pkcs7: certificate parse error",
		"certificate-set entry 1",
		"x509:",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("got %q, want phase %q", err, want)
		}
	}
}

func TestParseRawCertificateSetRejectsTrailingWrapperData(t *testing.T) {
	raw := append(rawCertificateSet(t), 0x05, 0x00)
	var got error
	requireErrorWithoutPanic(t, "decode certificate-set wrapper", func() error {
		_, _, got = parseRawCertificateSet(raw)
		return got
	})
	if !errors.Is(got, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", got)
	}
	var syntaxErr asn1.SyntaxError
	if !errors.As(got, &syntaxErr) {
		t.Fatalf("got %v, want preserved ASN.1 syntax cause", got)
	}
}

func TestParseRawCertificateSetRetainsOneBasedEntryIndex(t *testing.T) {
	raw := rawCertificateSet(
		t,
		[]byte{0xa1, 0x00},
		[]byte{0x30, 0x03, 0x02, 0x01, 0x01},
	)
	var got error
	requireErrorWithoutPanic(t, "certificate-set entry 2", func() error {
		_, _, got = parseRawCertificateSet(raw)
		return got
	})
	if !errors.Is(got, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", got)
	}
}

func TestParseRawCertificateSetSkipsNonCertificateChoice(t *testing.T) {
	certs, crls, err := parseRawCertificateSet(rawCertificateSet(t, []byte{0xa1, 0x00}))
	if err != nil {
		t.Fatalf("expected unsupported CertificateChoice to be skipped: %v", err)
	}
	if len(certs) != 0 || len(crls) != 0 {
		t.Fatalf("expected no certificates or CRLs, got %d and %d", len(certs), len(crls))
	}
}

func TestParseRawCertificateSetRejectsPrimitiveWrapper(t *testing.T) {
	var got error
	requireErrorWithoutPanic(t, "certificate-set wrapper is not constructed", func() error {
		_, _, got = parseRawCertificateSet([]byte{0x80, 0x00})
		return got
	})
	if !errors.Is(got, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", got)
	}
	var structuralErr asn1.StructuralError
	if !errors.As(got, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", got)
	}
}

func TestParseRawCertificateSetRejectsUniversalNonCertificateChoice(t *testing.T) {
	raw := rawCertificateSet(t, []byte{0x02, 0x01, 0x01})
	var got error
	requireErrorWithoutPanic(t, "certificate-set entry 1", func() error {
		_, _, got = parseRawCertificateSet(raw)
		return got
	})
	if !errors.Is(got, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", got)
	}
	var structuralErr asn1.StructuralError
	if !errors.As(got, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", got)
	}
}

func rawCertificateSet(t *testing.T, entries ...[]byte) []byte {
	t.Helper()
	content := bytes.Join(entries, nil)
	raw, err := asn1.Marshal(asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      content,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestParseRawCertificateSetPreservesASN1Cause(t *testing.T) {
	_, _, err := parseRawCertificateSet([]byte{0xa0, 0x01, 0x30})
	if !errors.Is(err, ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", err)
	}
	var syntaxErr asn1.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("got %v, want preserved ASN.1 syntax cause", err)
	}
}

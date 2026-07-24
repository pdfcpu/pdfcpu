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
	"encoding/asn1"
	"errors"
	"strings"
	"testing"
)

func TestEmptyInputClassification(t *testing.T) {
	_, err := Parse(nil)
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("got %v, want ErrEmptyInput", err)
	}
}

func TestUnsupportedAlgorithmClassification(t *testing.T) {
	_, err := HashForOID(asn1.ObjectIdentifier{1, 2, 3})
	if !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", err)
	}
}

func TestMissingSignerIdentifierClassification(t *testing.T) {
	err := validateSignerIdentifier(issuerAndSerial{})
	if !errors.Is(err, ErrMissingSignerIdentifier) {
		t.Fatalf("got %v, want ErrMissingSignerIdentifier", err)
	}
}

func TestMissingAttributeClassification(t *testing.T) {
	var value []byte
	err := unmarshalAttribute(nil, OIDData, &value)
	if !errors.Is(err, ErrMalformedAttribute) {
		t.Fatalf("got %v, want ErrMalformedAttribute", err)
	}
}

func TestMalformedAttributePreservesASN1Cause(t *testing.T) {
	attrs := []attribute{{
		Type: OIDData,
		Value: asn1.RawValue{
			Bytes: []byte{0x30, 0x01},
		},
	}}
	var value []byte
	err := unmarshalAttribute(attrs, OIDData, &value)
	if !errors.Is(err, ErrMalformedAttribute) {
		t.Fatalf("got %v, want ErrMalformedAttribute", err)
	}
	var structuralErr asn1.StructuralError
	if !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", err)
	}
}

func TestParsePreservesContentInfoASN1Cause(t *testing.T) {
	_, err := Parse([]byte{0x30, 0x03, 0x06, 0x01, 0x80})
	if err == nil {
		t.Fatal("expected malformed content-info error")
	}
	var syntaxErr asn1.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("got %v, want preserved ASN.1 syntax cause", err)
	}
	if !strings.Contains(err.Error(), "pkcs7: parse content info") {
		t.Fatalf("got %v, want content-info phase", err)
	}
}

func TestParsePreservesSignedDataASN1Cause(t *testing.T) {
	message, err := asn1.Marshal(contentInfo{
		ContentType: OIDSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      []byte{0x02, 0x01, 0x01},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Parse(message)
	if err == nil {
		t.Fatal("expected malformed signed-data error")
	}
	var structuralErr asn1.StructuralError
	if !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", err)
	}
	if !strings.Contains(err.Error(), "pkcs7: parse signed data") {
		t.Fatalf("got %v, want signed-data phase", err)
	}
}

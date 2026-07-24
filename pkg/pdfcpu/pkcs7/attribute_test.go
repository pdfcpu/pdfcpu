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

func TestAttributesForMarshallingAcceptsNilReceiver(t *testing.T) {
	var attrs *attributes
	got, err := attrs.ForMarshalling()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("got %v, want nil attributes", got)
	}
}

func TestAttributesForMarshallingRejectsMismatchedSlicesWithoutPanic(t *testing.T) {
	attrs := &attributes{types: []asn1.ObjectIdentifier{OIDData}}
	requireErrorWithoutPanic(t, "mismatched types and values", func() error {
		_, err := attrs.ForMarshalling()
		return err
	})
}

func TestAttributesForMarshallingPreservesValueCauseAndIndex(t *testing.T) {
	attrs := &attributes{
		types:  []asn1.ObjectIdentifier{OIDData},
		values: []interface{}{make(chan int)},
	}
	var got error
	requireErrorWithoutPanic(t, "attribute index 1: encode value", func() error {
		_, got = attrs.ForMarshalling()
		return got
	})
	var structuralErr asn1.StructuralError
	if !errors.As(got, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", got)
	}
}

func TestAttributesForMarshallingPreservesAttributeCauseAndIndex(t *testing.T) {
	attrs := &attributes{
		types:  []asn1.ObjectIdentifier{{3}},
		values: []interface{}{1},
	}
	var got error
	requireErrorWithoutPanic(t, "attribute index 1: encode attribute", func() error {
		_, got = attrs.ForMarshalling()
		return got
	})
	var structuralErr asn1.StructuralError
	if !errors.As(got, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", got)
	}
}

func TestMarshalAttributesPreservesASN1Cause(t *testing.T) {
	_, err := marshalAttributes([]attribute{{Type: asn1.ObjectIdentifier{3}}})
	if err == nil {
		t.Fatal("expected attribute-set marshal error")
	}
	if !strings.Contains(err.Error(), "marshal attributes: encode set") {
		t.Fatalf("got %v, want attribute-set phase", err)
	}
	var structuralErr asn1.StructuralError
	if !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", err)
	}
}

func TestUnmarshalAttributeRejectsDuplicateValues(t *testing.T) {
	value := attributeValue(t, []byte("digest"))
	attrs := []attribute{
		{Type: oidAttributeMessageDigest, Value: value},
		{Type: oidAttributeMessageDigest, Value: value},
	}
	var digest []byte
	err := unmarshalAttribute(attrs, oidAttributeMessageDigest, &digest)
	if !errors.Is(err, ErrMalformedAttribute) {
		t.Fatalf("got %v, want ErrMalformedAttribute", err)
	}
	if !strings.Contains(err.Error(), "attribute indexes 1 and 2") {
		t.Fatalf("got %v, want duplicate attribute indexes", err)
	}
}

func TestUnmarshalAttributeRejectsTrailingValueData(t *testing.T) {
	value := attributeValue(t, []byte("digest"))
	value.Bytes = append(value.Bytes, 0x05, 0x00)
	var digest []byte
	err := unmarshalAttribute(
		[]attribute{{Type: oidAttributeMessageDigest, Value: value}},
		oidAttributeMessageDigest,
		&digest,
	)
	if !errors.Is(err, ErrMalformedAttribute) {
		t.Fatalf("got %v, want ErrMalformedAttribute", err)
	}
	var syntaxErr asn1.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("got %v, want preserved ASN.1 syntax cause", err)
	}
	if !strings.Contains(err.Error(), "attribute index 1") {
		t.Fatalf("got %v, want one-based attribute index", err)
	}
}

func attributeValue(t *testing.T, value interface{}) asn1.RawValue {
	t.Helper()
	bb, err := asn1.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return asn1.RawValue{Bytes: bb}
}

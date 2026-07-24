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
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"strings"
	"testing"
)

func TestHashForOIDPreservesUnsupportedAlgorithmClassification(t *testing.T) {
	oid := asn1.ObjectIdentifier{1, 2, 3}
	_, err := HashForOID(oid)
	if !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", err)
	}
	if !strings.Contains(err.Error(), oid.String()) {
		t.Fatalf("got %v, want digest OID context", err)
	}
}

func TestVerifyMessageDigestDetachedPreservesUnsupportedAlgorithm(t *testing.T) {
	signer := SignerInfo{
		DigestAlgorithm: pkix.AlgorithmIdentifier{Algorithm: asn1.ObjectIdentifier{1, 2, 3}},
	}
	var got error
	requireErrorWithoutPanic(t, "verify detached message digest: select algorithm", func() error {
		got = VerifyMessageDigestDetached(signer, nil)
		return got
	})
	if !errors.Is(got, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", got)
	}
}

func TestVerifyMessageDigestDetachedPreservesMalformedAttribute(t *testing.T) {
	signer := SignerInfo{
		DigestAlgorithm: pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256},
	}
	var got error
	requireErrorWithoutPanic(t, "read message-digest attribute", func() error {
		got = VerifyMessageDigestDetached(signer, nil)
		return got
	})
	if !errors.Is(got, ErrMalformedAttribute) {
		t.Fatalf("got %v, want ErrMalformedAttribute", got)
	}
}

func TestVerifyMessageDigestDetachedPreservesAttributeASN1Cause(t *testing.T) {
	signer := SignerInfo{
		DigestAlgorithm: pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256},
		AuthenticatedAttributes: []attribute{{
			Type:  oidAttributeMessageDigest,
			Value: asn1.RawValue{Bytes: []byte{0x30, 0x01}},
		}},
	}
	err := VerifyMessageDigestDetached(signer, nil)
	if !errors.Is(err, ErrMalformedAttribute) {
		t.Fatalf("got %v, want ErrMalformedAttribute", err)
	}
	var structuralErr asn1.StructuralError
	if !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", err)
	}
}

func TestVerifyMessageDigestDetachedSuccess(t *testing.T) {
	data := []byte("detached content")
	digest := sha256.Sum256(data)
	signer := SignerInfo{
		DigestAlgorithm:         pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256},
		AuthenticatedAttributes: []attribute{messageDigestAttribute(t, digest[:])},
	}
	if err := VerifyMessageDigestDetached(signer, data); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyMessageDigestMismatchClassification(t *testing.T) {
	tests := []struct {
		name  string
		phase string
		call  func() error
	}{
		{
			"detached",
			"verify detached message digest",
			func() error {
				signer := SignerInfo{
					DigestAlgorithm:         pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256},
					AuthenticatedAttributes: []attribute{messageDigestAttribute(t, []byte("wrong"))},
				}
				return VerifyMessageDigestDetached(signer, []byte("content"))
			},
		},
		{
			"embedded",
			"verify embedded message digest",
			func() error {
				return VerifyMessageDigestEmbedded([]byte("wrong"), []byte("content"))
			},
		},
		{
			"timestamp token",
			"verify timestamp-token message digest",
			func() error {
				return VerifyMessageDigestTSToken(
					OIDDigestAlgorithmSHA256,
					[]byte("wrong"),
					[]byte("content"),
				)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			var mismatch *MessageDigestMismatchError
			if !errors.As(err, &mismatch) {
				t.Fatalf("got %v, want MessageDigestMismatchError", err)
			}
			if !strings.Contains(err.Error(), tt.phase) {
				t.Fatalf("got %v, want phase %q", err, tt.phase)
			}
		})
	}
}

func TestVerifyMessageDigestEmbeddedAndTimestampSuccess(t *testing.T) {
	data := []byte("content")
	sha1Digest := sha1.Sum(data)
	if err := VerifyMessageDigestEmbedded(sha1Digest[:], data); err != nil {
		t.Fatal(err)
	}
	sha256Digest := sha256.Sum256(data)
	if err := VerifyMessageDigestTSToken(OIDDigestAlgorithmSHA256, sha256Digest[:], data); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyTimestampDigestPreservesUnsupportedAlgorithm(t *testing.T) {
	var got error
	requireErrorWithoutPanic(t, "verify timestamp-token message digest: select algorithm", func() error {
		got = VerifyMessageDigestTSToken(asn1.ObjectIdentifier{1, 2, 3}, nil, nil)
		return got
	})
	if !errors.Is(got, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", got)
	}
}

func TestCompareMessageDigestRejectsUnavailableHashWithoutPanic(t *testing.T) {
	var got error
	requireErrorWithoutPanic(t, "unavailable", func() error {
		got = compareMessageDigest(crypto.Hash(255), nil, nil)
		return got
	})
	if !errors.Is(got, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", got)
	}
}

func messageDigestAttribute(t *testing.T, digest []byte) attribute {
	t.Helper()
	value, err := asn1.Marshal(digest)
	if err != nil {
		t.Fatal(err)
	}
	return attribute{
		Type:  oidAttributeMessageDigest,
		Value: asn1.RawValue{Bytes: value},
	}
}

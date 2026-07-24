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
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
)

type failingRandomReader struct {
	err error
}

func (r failingRandomReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestSignAttributesRejectsUnavailableHashWithoutPanic(t *testing.T) {
	_, key := signerCertificate(t, "Signer")
	var got error
	requireErrorWithoutPanic(t, "signing hash 255 is unavailable", func() error {
		_, got = signAttributes(nil, key, crypto.Hash(255))
		return got
	})
	if !errors.Is(got, ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want ErrUnsupportedAlgorithm", got)
	}
}

func TestSignAttributesRejectsTypedNilKeyWithoutPanic(t *testing.T) {
	requireErrorWithoutPanic(t, "RSA private key is missing", func() error {
		_, err := signAttributes(nil, (*rsa.PrivateKey)(nil), crypto.SHA256)
		return err
	})
}

func TestSignAttributesPreservesRandomSourceFailure(t *testing.T) {
	_, key := signerCertificate(t, "Signer")
	cause := errors.New("entropy unavailable")
	_, err := signAttributesWithRandom(
		nil,
		key,
		crypto.SHA256,
		failingRandomReader{err: cause},
	)
	if !errors.Is(err, cause) {
		t.Fatalf("got %v, want preserved random-source cause", err)
	}
	if !strings.Contains(err.Error(), "sign attributes: sign digest") {
		t.Fatalf("got %v, want signing phase", err)
	}
}

func TestSignAttributesRejectsMissingRandomSourceWithoutPanic(t *testing.T) {
	_, key := signerCertificate(t, "Signer")
	requireErrorWithoutPanic(t, "random source is missing", func() error {
		_, err := signAttributesWithRandom(nil, key, crypto.SHA256, nil)
		return err
	})
}

func TestSignAttributesProducesVerifiableSignature(t *testing.T) {
	cert, key := signerCertificate(t, "Signer")
	attrs := []attribute{messageDigestAttribute(t, []byte("digest"))}
	signature, err := signAttributes(attrs, key, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	content, err := marshalAttributes(attrs)
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.CheckSignature(x509.ECDSAWithSHA256, content, signature); err != nil {
		t.Fatalf("verify generated signature: %v", err)
	}
}

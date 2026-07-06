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

package pdfcpu

import (
	"errors"
	"testing"
)

func TestValidateEncryptLengthClassifiesUnsupportedEncryption(t *testing.T) {
	t.Parallel()
	_, err := validateEncryptLength(nil, 7)
	if !errors.Is(err, ErrUnsupportedEncryption) {
		t.Fatalf("got %v, want ErrUnsupportedEncryption", err)
	}
}

func TestPasswordHashEqual(t *testing.T) {
	t.Parallel()
	if !passwordHashEqual([]byte("digest"), []byte("digest")) {
		t.Fatal("equal hashes must match")
	}
	if passwordHashEqual([]byte("digest"), []byte("digesu")) {
		t.Fatal("different hashes must not match")
	}
	if passwordHashEqual([]byte("digest"), []byte("digest-longer")) {
		t.Fatal("hashes of different lengths must not match")
	}
}

func TestPasswordHashPrefixEqual(t *testing.T) {
	t.Parallel()
	if !passwordHashPrefixEqual([]byte("digest-and-salt"), []byte("digest")) {
		t.Fatal("equal hash prefix must match")
	}
	if passwordHashPrefixEqual([]byte("digesu-and-salt"), []byte("digest")) {
		t.Fatal("different hash prefix must not match")
	}
	if passwordHashPrefixEqual([]byte("short"), []byte("digest")) {
		t.Fatal("short stored hash must not match")
	}
}

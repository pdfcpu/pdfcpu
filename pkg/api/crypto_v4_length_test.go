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

package api

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestDecryptV4WithoutEncryptLength covers issue #1522 for 128-bit AES and RC4 without top-level Length.
func TestDecryptV4WithoutEncryptLength(t *testing.T) {
	source, err := os.ReadFile("../testdata/test.pdf")
	if err != nil {
		t.Fatal(err)
	}

	for _, algorithm := range []struct {
		name string
		conf *model.Configuration
	}{
		{"AES-128", model.NewAESConfiguration("secret", "owner", 128)},
		{"RC4-128", model.NewRC4Configuration("secret", "owner", 128)},
	} {
		t.Run(algorithm.name, func(t *testing.T) {
			var encrypted bytes.Buffer
			if err := Encrypt(t.Context(), bytes.NewReader(source), &encrypted, algorithm.conf); err != nil {
				t.Fatal(err)
			}

			withLength := []byte("/Filter/Standard/Length 128")
			if n := bytes.Count(encrypted.Bytes(), withLength); n != 1 {
				t.Fatalf("found %d top-level encrypt Length entries, want 1", n)
			}
			withoutLength := bytes.Replace(encrypted.Bytes(), withLength, []byte("/Filter/Standard/Lxngth 128"), 1)

			for _, tc := range []struct {
				name, userPW, ownerPW string
				wrong                 bool
			}{
				{"user", "secret", "", false},
				{"owner", "", "owner", false},
				{"wrong", "incorrect", "incorrect", true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					conf := model.NewDefaultConfiguration()
					conf.UserPW, conf.OwnerPW = tc.userPW, tc.ownerPW
					var decrypted bytes.Buffer
					err := Decrypt(t.Context(), bytes.NewReader(withoutLength), &decrypted, conf)
					if tc.wrong {
						if !errors.Is(err, pdfcpu.ErrWrongPassword) {
							t.Fatalf("got %v, want ErrWrongPassword", err)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if err := Validate(t.Context(), bytes.NewReader(decrypted.Bytes()), nil, nil); err != nil {
						t.Fatalf("decrypted output is invalid: %v", err)
					}
				})
			}
		})
	}
}

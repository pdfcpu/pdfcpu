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
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestEncryptionDictionaryErrorsClassified(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "missing filter",
			err: func() error {
				_, err := validateEncryptFilter(nil)
				return err
			}(),
			want: ErrMalformedEncryption,
		},
		{
			name: "unsupported filter",
			err: func() error {
				_, err := validateEncryptFilter(types.Dict{"Filter": types.Name("Adobe.PubSec")})
				return err
			}(),
			want: ErrUnsupportedEncryptionFeature,
		},
		{
			name: "missing V",
			err: func() error {
				_, err := validateEncryptV(nil)
				return err
			}(),
			want: ErrMalformedEncryption,
		},
		{
			name: "invalid V",
			err: func() error {
				_, err := validateEncryptV(types.Dict{"V": types.Integer(9)})
				return err
			}(),
			want: ErrMalformedEncryption,
		},
		{
			name: "unsupported V",
			err: func() error {
				_, err := validateEncryptV(types.Dict{"V": types.Integer(6)})
				return err
			}(),
			want: ErrUnsupportedEncryptionFeature,
		},
		{
			name: "invalid length",
			err: func() error {
				_, err := validateEncryptLength(types.Dict{"Length": types.Integer(44)}, 2)
				return err
			}(),
			want: ErrMalformedEncryption,
		},
		{
			name: "missing P",
			err:  missingEncryptPError(t),
			want: ErrMalformedEncryption,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("got %v, want %v", tt.err, tt.want)
			}
		})
	}
}

func TestValidateEncryptLengthClassifiesInvalidVersion(t *testing.T) {
	t.Parallel()
	_, err := validateEncryptLength(nil, 7)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want ErrMalformedEncryption", err)
	}
}

func TestAES256ParameterErrorsClassifiedMalformedEncryption(t *testing.T) {
	t.Parallel()

	d := types.Dict{
		"OE":    types.HexLiteral(strings.Repeat("00", 31)),
		"UE":    types.HexLiteral(strings.Repeat("00", 32)),
		"Perms": types.HexLiteral(strings.Repeat("00", 16)),
	}

	_, _, _, err := validateAES256Parameters(d, 6)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want ErrMalformedEncryption", err)
	}
}

func TestValidateOAndUErrorsClassifiedMalformedEncryption(t *testing.T) {
	t.Parallel()

	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	setStrictPDF17XRefTable(ctx)

	d := types.Dict{
		"O": types.HexLiteral(strings.Repeat("00", 32)),
		"U": types.HexLiteral(strings.Repeat("00", 31)),
	}

	_, _, err = validateOAndU(ctx, d, 4)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want ErrMalformedEncryption", err)
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

func missingEncryptPError(t *testing.T) error {
	t.Helper()

	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	setStrictPDF17XRefTable(ctx)

	d := types.Dict{
		"Filter": types.Name("Standard"),
		"V":      types.Integer(1),
		"R":      types.Integer(2),
		"O":      types.HexLiteral(strings.Repeat("00", 32)),
		"U":      types.HexLiteral(strings.Repeat("00", 32)),
	}

	_, err = supportedEncryption(ctx, d)
	return err
}

func setStrictPDF17XRefTable(ctx *model.Context) {
	v := model.V17
	ctx.XRefTable = &model.XRefTable{
		HeaderVersion:  &v,
		ValidationMode: model.ValidationStrict,
	}
}

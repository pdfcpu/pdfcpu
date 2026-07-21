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
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestClassifyEncryptionDictionaryErrorDistinguishesStructuralFailures verifies I/O errors remain operational.
func TestClassifyEncryptionDictionaryErrorDistinguishesStructuralFailures(t *testing.T) {
	ioErr := errors.New("read failed")
	tests := []struct {
		name      string
		err       error
		malformed bool
	}{
		{name: "missing reader object", err: errUnregisteredObject, malformed: true},
		{name: "nil reader object", err: errNilDereferencedObject, malformed: true},
		{name: "free reader object", err: ErrReferenceDoesNotExist, malformed: true},
		{name: "wrong reader type", err: errCorruptDictObject, malformed: true},
		{name: "missing model object", err: model.ErrMissingEncryptDictObject, malformed: true},
		{name: "wrong model type", err: model.ErrWrongTypeEncryptDictObject, malformed: true},
		{name: "corrupt dictionary syntax", err: model.ErrDictionaryCorrupt, malformed: true},
		{name: "read failure", err: ioErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyEncryptionDictionaryError(tt.err)
			if !errors.Is(err, tt.err) {
				t.Fatalf("lost lower cause %v: %v", tt.err, err)
			}
			if got := errors.Is(err, ErrMalformedEncryption); got != tt.malformed {
				t.Fatalf("malformed classification=%t, want %t: %v", got, tt.malformed, err)
			}
		})
	}
}

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

	d := types.Dict{
		"O": types.HexLiteral(strings.Repeat("00", 32)),
		"U": types.HexLiteral(strings.Repeat("00", 31)),
	}

	_, _, err := validateOAndU(d, 4, false)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want ErrMalformedEncryption", err)
	}
}

func TestValidateOAndURejectsShortR4EntriesInStrictMode(t *testing.T) {
	tests := []struct {
		name string
		oLen int
		uLen int
	}{
		{name: "short O", oLen: 31, uLen: 32},
		{name: "short U", oLen: 32, uLen: 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := types.Dict{
				"O": types.HexLiteral(strings.Repeat("00", tt.oLen)),
				"U": types.HexLiteral(strings.Repeat("00", tt.uLen)),
			}

			_, _, err := validateOAndU(d, 4, false)
			if !errors.Is(err, ErrMalformedEncryption) {
				t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
			}
		})
	}
}

func TestValidateOAndURetainsR4RelaxedModeBehavior(t *testing.T) {
	d := types.Dict{
		"O": types.HexLiteral("00"),
		"U": types.HexLiteral("00"),
	}

	o, u, err := validateOAndU(d, 4, true)
	if err != nil {
		t.Fatalf("expected relaxed mode to accept short R4 entries: %v", err)
	}
	if len(o) != 1 || len(u) != 1 {
		t.Fatalf("got O/U lengths %d/%d, want 1/1", len(o), len(u))
	}
}

func TestValidateOAndURejectsShortR5AndR6EntriesInAllModes(t *testing.T) {
	tests := []struct {
		name     string
		revision int
		relaxed  bool
		oLen     int
		uLen     int
	}{
		{name: "R5 strict short O", revision: 5, oLen: 47, uLen: 48},
		{name: "R5 strict short U", revision: 5, oLen: 48, uLen: 47},
		{name: "R5 relaxed short O", revision: 5, relaxed: true, oLen: 47, uLen: 48},
		{name: "R5 relaxed short U", revision: 5, relaxed: true, oLen: 48, uLen: 47},
		{name: "R6 strict short O", revision: 6, oLen: 47, uLen: 48},
		{name: "R6 strict short U", revision: 6, oLen: 48, uLen: 47},
		{name: "R6 relaxed short O", revision: 6, relaxed: true, oLen: 47, uLen: 48},
		{name: "R6 relaxed short U", revision: 6, relaxed: true, oLen: 48, uLen: 47},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := types.Dict{
				"O": types.HexLiteral(strings.Repeat("00", tt.oLen)),
				"U": types.HexLiteral(strings.Repeat("00", tt.uLen)),
			}

			_, _, err := validateOAndU(d, tt.revision, tt.relaxed)
			if !errors.Is(err, ErrMalformedEncryption) {
				t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
			}
		})
	}
}

func TestValidateOAndURejectsShortAES256EntriesInRelaxedMode(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.XRefTable.HeaderVersion = &v
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed

	d := newEncryptDict(false, true, 256, 0)
	d["O"] = types.HexLiteral(strings.Repeat("00", 47))

	_, err = supportedEncryption(ctx, d)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !strings.Contains(err.Error(), `entry "O"`) {
		t.Fatalf("expected O entry context, got %q", err)
	}
}

func TestSupportedEncryptionClassifiesRevision7Unsupported(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.XRefTable.HeaderVersion = &v
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed

	d := newEncryptDict(false, true, 256, 0)
	d["R"] = types.Integer(7)

	_, err = supportedEncryption(ctx, d)
	if !errors.Is(err, ErrUnsupportedEncryptionFeature) {
		t.Fatalf("got %v, want %v", err, ErrUnsupportedEncryptionFeature)
	}
	if !strings.Contains(err.Error(), `encrypt "R" 7`) {
		t.Fatalf("expected revision context, got %q", err)
	}
}

func TestValidateCryptFilterRecipientsIncludesEntryContext(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	cfm := "AESV3"
	d := types.Dict{"Recipients": *types.NewIndirectRef(99, 0)}

	err = validateCryptFilterRecipients(ctx, d, &cfm)
	if err == nil || !strings.Contains(err.Error(), `crypt filter entry "Recipients"`) {
		t.Fatalf("expected Recipients dereference context, got %v", err)
	}
}

func TestValidateStmfIncludesEncryptEntryContext(t *testing.T) {
	ctx, err := model.NewContext(bytes.NewReader(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.XRefTable.HeaderVersion = &v

	d := types.Dict{"StmF": types.Name("StdCF")}
	cfDict := types.Dict{
		"StdCF": types.Dict{
			"CFM":    types.Name("bad"),
			"Length": types.Integer(128),
		},
	}

	err = validateStmf(ctx, d, cfDict, 4, false, false)
	if !errors.Is(err, ErrMalformedEncryption) {
		t.Fatalf("got %v, want %v", err, ErrMalformedEncryption)
	}
	if !strings.Contains(err.Error(), `encrypt dict entry "StmF"`) {
		t.Fatalf("expected StmF context, got %q", err)
	}
}

func TestDecryptKeyDoesNotMutateInput(t *testing.T) {
	backing := bytes.Repeat([]byte{0xA5}, 32)
	key := backing[:5]
	want := bytes.Clone(backing)

	_ = decryptKey(7, 0, key, false)
	if !bytes.Equal(backing, want) {
		t.Fatalf("input key backing array mutated: got %x, want %x", backing, want)
	}
}

func TestDecryptAESBytesRejectsIVOnlyCiphertext(t *testing.T) {
	_, err := decryptAESBytes(make([]byte, 16), make([]byte, 16))
	if err == nil || !strings.Contains(err.Error(), "ciphertext too short") {
		t.Fatalf("expected short ciphertext error, got %v", err)
	}
}

func TestAESBytesRoundTrip(t *testing.T) {
	key := make([]byte, 16)
	want := []byte("encrypted content")
	encrypted, err := encryptAESBytes(bytes.Clone(want), key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decryptAESBytes(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestValidatePermissionsChecksEncryptMetadata verifies the encrypted permission flag matches the encryption dictionary.
func TestValidatePermissionsChecksEncryptMetadata(t *testing.T) {
	for _, encryptMetadata := range []bool{false, true} {
		t.Run(fmt.Sprintf("encryptMetadata=%t", encryptMetadata), func(t *testing.T) {
			ctx := &model.Context{
				XRefTable: &model.XRefTable{
					E: &model.Enc{
						P:     -4,
						R:     6,
						Emd:   encryptMetadata,
						Perms: make([]byte, 16),
					},
					EncKey: make([]byte, 32),
				},
			}
			if err := writePermissions(ctx, types.Dict{}); err != nil {
				t.Fatal(err)
			}

			ok, err := validatePermissions(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatal("matching EncryptMetadata flag rejected")
			}

			ctx.E.Emd = !ctx.E.Emd
			ok, err = validatePermissions(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				t.Fatal("mismatched EncryptMetadata flag accepted")
			}
		})
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

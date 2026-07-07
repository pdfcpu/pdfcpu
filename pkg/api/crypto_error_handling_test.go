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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type cryptoErrorReadSeeker struct {
	err    error
	offset int64
}

func (r *cryptoErrorReadSeeker) Read([]byte) (int, error) {
	return 0, r.err
}

func (r *cryptoErrorReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.offset = offset
	case io.SeekCurrent:
		r.offset += offset
	case io.SeekEnd:
		r.offset = 1 + offset
	}
	return r.offset, nil
}

func TestCryptoArgumentErrors(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	tests := []struct {
		name string
		fn   func() error
		want error
	}{
		{
			name: "encrypt missing reader",
			fn: func() error {
				return Encrypt(nil, io.Discard, conf)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "encrypt missing writer",
			fn: func() error {
				return Encrypt(bytes.NewReader(nil), nil, conf)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "encrypt missing configuration",
			fn: func() error {
				return Encrypt(bytes.NewReader(nil), io.Discard, nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "decrypt missing reader",
			fn: func() error {
				return Decrypt(nil, io.Discard, conf)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "decrypt missing writer",
			fn: func() error {
				return Decrypt(bytes.NewReader(nil), nil, conf)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "decrypt missing configuration",
			fn: func() error {
				return Decrypt(bytes.NewReader(nil), io.Discard, nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "encrypt file missing configuration",
			fn: func() error {
				return EncryptFile("in.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "encrypt file missing input",
			fn: func() error {
				return EncryptFile("", filepath.Join(t.TempDir(), "out.pdf"), conf)
			},
			want: ErrMissingPDFInput,
		},
		{
			name: "decrypt file missing configuration",
			fn: func() error {
				return DecryptFile("in.pdf", filepath.Join(t.TempDir(), "out.pdf"), nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "decrypt file missing input",
			fn: func() error {
				return DecryptFile("", filepath.Join(t.TempDir(), "out.pdf"), conf)
			},
			want: ErrMissingPDFInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCryptoOperationContextPreservesReadError(t *testing.T) {
	readErr := errors.New("read failed")
	tests := []struct {
		name string
		op   string
		fn   func(io.ReadSeeker, io.Writer, *model.Configuration) error
	}{
		{name: "encrypt", op: "encrypt:", fn: Encrypt},
		{name: "decrypt", op: "decrypt:", fn: Decrypt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(&cryptoErrorReadSeeker{err: readErr}, io.Discard, model.NewDefaultConfiguration())
			if !errors.Is(err, readErr) {
				t.Fatalf("got %v, want %v", err, readErr)
			}
			if !strings.Contains(err.Error(), tt.op) {
				t.Fatalf("expected %q context, got %q", tt.op, err)
			}
			if count := strings.Count(err.Error(), tt.op); count != 1 {
				t.Fatalf("expected one %q label, got %d in %q", tt.op, count, err)
			}
			wantPrefix := tt.name + ": prepare PDF context: read context:"
			if !strings.HasPrefix(err.Error(), wantPrefix) {
				t.Fatalf("expected prefix %q, got %q", wantPrefix, err)
			}
			if strings.Contains(err.Error(), tt.name+": optimize:") {
				t.Fatalf("unexpected public optimize phase: %q", err)
			}
		})
	}
}

func encryptedCryptoTestFile(t *testing.T) (string, string) {
	t.Helper()
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "encrypted.pdf")
	conf := model.NewDefaultConfiguration()
	conf.UserPW = "user"
	conf.OwnerPW = "owner"
	if err := EncryptFile(inFile, outFile, conf); err != nil {
		t.Fatal(err)
	}
	return inFile, outFile
}

func requireCryptoError(t *testing.T, err, want error, contexts ...string) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	for _, context := range contexts {
		if !strings.Contains(err.Error(), context) {
			t.Fatalf("missing %q context: %v", context, err)
		}
	}
}

// TestCryptoFileOperationSentinels verifies exported file APIs preserve encryption causes.
func TestCryptoFileOperationSentinels(t *testing.T) {
	inFile, encryptedFile := encryptedCryptoTestFile(t)

	t.Run("encrypt already encrypted", func(t *testing.T) {
		conf := model.NewDefaultConfiguration()
		conf.UserPW = "user"
		conf.OwnerPW = "owner"
		err := EncryptFile(encryptedFile, filepath.Join(t.TempDir(), "out.pdf"), conf)
		requireCryptoError(t, err, pdfcpu.ErrEncrypted, "encrypt:", "encryption status")
	})

	t.Run("decrypt unencrypted", func(t *testing.T) {
		err := DecryptFile(inFile, filepath.Join(t.TempDir(), "out.pdf"), model.NewDefaultConfiguration())
		requireCryptoError(t, err, pdfcpu.ErrNotEncrypted, "decrypt:", "encryption status")
	})

	tests := []struct {
		name string
		upw  string
	}{
		{name: "wrong password", upw: "wrong"},
		{name: "missing password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.UserPW = tt.upw
			err := DecryptFile(encryptedFile, filepath.Join(t.TempDir(), "out.pdf"), conf)
			requireCryptoError(t, err, pdfcpu.ErrWrongPassword, "decrypt:", "encryption setup")
			if count := strings.Count(err.Error(), "decrypt:"); count != 1 {
				t.Fatalf("expected one decrypt label, got %d in %q", count, err)
			}
			wantPrefix := "decrypt: prepare PDF context: read context: encryption setup:"
			if !strings.HasPrefix(err.Error(), wantPrefix) {
				t.Fatalf("expected prefix %q, got %q", wantPrefix, err)
			}
		})
	}
}

func TestCryptoFileIOErrorContext(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		fn   func(string, string, *model.Configuration) error
	}{
		{name: "encrypt", fn: EncryptFile},
		{name: "decrypt", fn: DecryptFile},
	}

	for _, tt := range tests {
		t.Run(tt.name+" open input", func(t *testing.T) {
			err := tt.fn(missingInput, "", conf)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": open input " + missingInput
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q context, got %q", want, err)
			}
		})

		t.Run(tt.name+" create output", func(t *testing.T) {
			err := tt.fn(inFile, missingOutput, conf)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("got %v, want %v", err, os.ErrNotExist)
			}
			want := tt.name + ": create output"
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q context, got %q", want, err)
			}
		})
	}
}

// TestProcessSecurityFilePreservesExistingOutputOnFailure verifies protected destinations survive operation failure.
func TestProcessSecurityFilePreservesExistingOutputOnFailure(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantOutput := []byte("existing output")
	if err := os.WriteFile(outFile, wantOutput, 0o600); err != nil {
		t.Fatal(err)
	}

	operationErr := errors.New("encryption failed")
	err := processSecurityFile(
		inFile,
		outFile,
		model.NewDefaultConfiguration(),
		"encrypt",
		func(io.ReadSeeker, io.Writer, *model.Configuration) error {
			return operationErr
		},
	)
	if !errors.Is(err, operationErr) {
		t.Fatalf("got %v, want %v", err, operationErr)
	}
	gotOutput, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotOutput, wantOutput) {
		t.Fatalf("existing output changed: got %q, want %q", gotOutput, wantOutput)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary outputs remain: %v", matches)
	}
}

// TestCleanupSecurityFilesPreservesOperationAndCleanupErrors verifies cleanup does not hide the primary cause.
// TestFinalizeSecurityFilesReportsCloseAndReplaceErrors verifies finalization phases remain identifiable.

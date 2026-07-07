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

type securityErrorWriter struct {
	err error
}

func (w securityErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func requireExistingSecurityOutputPreserved(
	t *testing.T,
	op string,
	run func(string, string) error,
) {
	t.Helper()

	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o640); err != nil {
		t.Fatal(err)
	}

	err := run(inFile, outFile)
	if err == nil || !strings.Contains(err.Error(), op+":") {
		t.Fatalf("expected %q operation error, got %v", op, err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".out.pdf.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary outputs remain: %v", matches)
	}
}

// TestPasswordChangeArgumentErrors verifies public password APIs reject malformed caller input.
func TestPasswordChangeArgumentErrors(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "user missing reader",
			run: func() error {
				return ChangeUserPassword(nil, io.Discard, "old", "new", conf)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "user missing writer",
			run: func() error {
				return ChangeUserPassword(bytes.NewReader(nil), nil, "old", "new", conf)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "user missing configuration",
			run: func() error {
				return ChangeUserPassword(bytes.NewReader(nil), io.Discard, "old", "new", nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "user file missing configuration",
			run: func() error {
				return ChangeUserPasswordFile("in.pdf", "", "old", "new", nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "user file missing input",
			run: func() error {
				return ChangeUserPasswordFile("", "", "old", "new", conf)
			},
			want: ErrMissingPDFInput,
		},
		{
			name: "owner missing reader",
			run: func() error {
				return ChangeOwnerPassword(nil, io.Discard, "old", "new", conf)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "owner missing writer",
			run: func() error {
				return ChangeOwnerPassword(bytes.NewReader(nil), nil, "old", "new", conf)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "owner missing configuration",
			run: func() error {
				return ChangeOwnerPassword(bytes.NewReader(nil), io.Discard, "old", "new", nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "owner empty new password",
			run: func() error {
				return ChangeOwnerPassword(bytes.NewReader(nil), io.Discard, "old", "", conf)
			},
			want: pdfcpu.ErrOwnerPasswordRequired,
		},
		{
			name: "owner file missing configuration",
			run: func() error {
				return ChangeOwnerPasswordFile("in.pdf", "", "old", "new", nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "owner file missing input",
			run: func() error {
				return ChangeOwnerPasswordFile("", "", "old", "new", conf)
			},
			want: ErrMissingPDFInput,
		},
		{
			name: "owner file empty new password",
			run: func() error {
				return ChangeOwnerPasswordFile("in.pdf", "", "old", "", conf)
			},
			want: pdfcpu.ErrOwnerPasswordRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

// TestPasswordChangeReadErrorsIncludeOperationContext verifies each stream API owns one operation label.
func TestPasswordChangeReadErrorsIncludeOperationContext(t *testing.T) {
	readErr := errors.New("read failed")
	tests := []struct {
		name string
		run  func(io.ReadSeeker) error
	}{
		{
			name: "change user password",
			run: func(rs io.ReadSeeker) error {
				return ChangeUserPassword(rs, io.Discard, "old", "new", model.NewDefaultConfiguration())
			},
		},
		{
			name: "change owner password",
			run: func(rs io.ReadSeeker) error {
				return ChangeOwnerPassword(rs, io.Discard, "old", "new", model.NewDefaultConfiguration())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(&cryptoErrorReadSeeker{err: readErr})
			if !errors.Is(err, readErr) {
				t.Fatalf("got %v, want %v", err, readErr)
			}
			wantPrefix := tt.name + ": prepare PDF context: read context:"
			if !strings.HasPrefix(err.Error(), wantPrefix) {
				t.Fatalf("expected prefix %q, got %q", wantPrefix, err)
			}
			if strings.Count(err.Error(), tt.name+":") != 1 {
				t.Fatalf("expected one operation label, got %q", err)
			}
			if strings.Contains(err.Error(), "optimize:") {
				t.Fatalf("unexpected public optimize phase: %q", err)
			}
		})
	}
}

// TestPasswordChangeWriteErrorsIncludeOperationContext verifies write failures preserve their cause and phase.
func TestPasswordChangeWriteErrorsIncludeOperationContext(t *testing.T) {
	_, encryptedFile := encryptedCryptoTestFile(t)
	tests := []struct {
		name string
		run  func(io.ReadSeeker, io.Writer) error
	}{
		{
			name: "change user password",
			run: func(rs io.ReadSeeker, w io.Writer) error {
				conf := model.NewDefaultConfiguration()
				conf.OwnerPW = "owner"
				return ChangeUserPassword(rs, w, "user", "new-user", conf)
			},
		},
		{
			name: "change owner password",
			run: func(rs io.ReadSeeker, w io.Writer) error {
				conf := model.NewDefaultConfiguration()
				conf.UserPW = "user"
				return ChangeOwnerPassword(rs, w, "owner", "new-owner", conf)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(encryptedFile)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			writeErr := errors.New("write failed")
			err = tt.run(f, securityErrorWriter{err: writeErr})
			if !errors.Is(err, writeErr) {
				t.Fatalf("got %v, want %v", err, writeErr)
			}
			want := tt.name + ": write output"
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
			if strings.Count(err.Error(), tt.name+":") != 1 {
				t.Fatalf("expected one operation label, got %q", err)
			}
		})
	}
}

// TestPasswordChangePreservesAuthenticationSentinel verifies wrong current passwords remain classifiable.
func TestPasswordChangePreservesAuthenticationSentinel(t *testing.T) {
	_, encryptedFile := encryptedCryptoTestFile(t)
	tests := []struct {
		name string
		run  func(string) error
		want error
	}{
		{
			name: "change user password",
			run: func(outFile string) error {
				conf := model.NewDefaultConfiguration()
				conf.OwnerPW = "owner"
				return ChangeUserPasswordFile(encryptedFile, outFile, "wrong", "new-user", conf)
			},
			want: pdfcpu.ErrWrongPassword,
		},
		{
			name: "change owner password",
			run: func(outFile string) error {
				conf := model.NewDefaultConfiguration()
				conf.UserPW = "user"
				return ChangeOwnerPasswordFile(encryptedFile, outFile, "wrong", "new-owner", conf)
			},
			want: pdfcpu.ErrOwnerPasswordRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(filepath.Join(t.TempDir(), "out.pdf"))
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if !strings.Contains(err.Error(), tt.name+": prepare PDF context") {
				t.Fatalf("expected %q operation context, got %q", tt.name, err)
			}
		})
	}
}

// TestPasswordChangeFilesPreserveExistingOutputOnFailure verifies protected output handling for both password commands.
func TestPasswordChangeFilesPreserveExistingOutputOnFailure(t *testing.T) {
	requireExistingSecurityOutputPreserved(t, "change user password", func(inFile, outFile string) error {
		return ChangeUserPasswordFile(inFile, outFile, "old", "new", model.NewDefaultConfiguration())
	})
	requireExistingSecurityOutputPreserved(t, "change owner password", func(inFile, outFile string) error {
		return ChangeOwnerPasswordFile(inFile, outFile, "old", "new", model.NewDefaultConfiguration())
	})
}

// TestPasswordChangeFileIOErrorsIncludeOperationContext verifies file entry points identify their I/O phase.
func TestPasswordChangeFileIOErrorsIncludeOperationContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	tests := []struct {
		name string
		run  func(string, string) error
	}{
		{
			name: "change user password",
			run: func(inFile, outFile string) error {
				return ChangeUserPasswordFile(inFile, outFile, "old", "new", model.NewDefaultConfiguration())
			},
		},
		{
			name: "change owner password",
			run: func(inFile, outFile string) error {
				return ChangeOwnerPasswordFile(inFile, outFile, "old", "new", model.NewDefaultConfiguration())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" open input", func(t *testing.T) {
			err := tt.run(missingInput, "")
			if !errors.Is(err, os.ErrNotExist) ||
				!strings.Contains(err.Error(), tt.name+": open input "+missingInput) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		t.Run(tt.name+" create output", func(t *testing.T) {
			err := tt.run(inFile, missingOutput)
			if !errors.Is(err, os.ErrNotExist) ||
				!strings.Contains(err.Error(), tt.name+": create output") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

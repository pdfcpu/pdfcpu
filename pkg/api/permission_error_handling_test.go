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

// TestPermissionsListRejectsMissingReader verifies the formatted permissions API preserves its input sentinel.
func TestPermissionsListRejectsMissingReader(t *testing.T) {
	if _, err := PermissionsList(nil, nil); !errors.Is(err, ErrMissingPDFReadSeeker) {
		t.Fatalf("got %v, want %v", err, ErrMissingPDFReadSeeker)
	}
}

// TestPermissionArgumentErrors verifies public permission APIs reject malformed caller input.
func TestPermissionArgumentErrors(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	tests := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "permissions missing reader",
			run: func() error {
				_, err := Permissions(nil, conf)
				return err
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "set missing reader",
			run: func() error {
				return SetPermissions(nil, io.Discard, conf)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "set missing writer",
			run: func() error {
				return SetPermissions(bytes.NewReader(nil), nil, conf)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "set missing configuration",
			run: func() error {
				return SetPermissions(bytes.NewReader(nil), io.Discard, nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "set file missing configuration",
			run: func() error {
				return SetPermissionsFile("in.pdf", "", nil)
			},
			want: ErrMissingConfiguration,
		},
		{
			name: "set file missing input",
			run: func() error {
				return SetPermissionsFile("", "", conf)
			},
			want: ErrMissingPDFInput,
		},
		{
			name: "get missing reader",
			run: func() error {
				_, err := GetPermissions(nil, conf)
				return err
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "get file missing input",
			run: func() error {
				_, err := GetPermissionsFile("", conf)
				return err
			},
			want: ErrMissingPDFInput,
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

// TestPermissionReadErrorsIncludeOperationContext verifies list, set, and get own their API phase.
func TestPermissionReadErrorsIncludeOperationContext(t *testing.T) {
	readErr := errors.New("read failed")
	tests := []struct {
		name string
		run  func(io.ReadSeeker) error
		want string
	}{
		{
			name: "permissions",
			run: func(rs io.ReadSeeker) error {
				_, err := Permissions(rs, model.NewDefaultConfiguration())
				return err
			},
			want: "list permissions: prepare PDF context: read context:",
		},
		{
			name: "permissions list",
			run: func(rs io.ReadSeeker) error {
				_, err := PermissionsList(rs, model.NewDefaultConfiguration())
				return err
			},
			want: "list permissions: prepare PDF context: read context:",
		},
		{
			name: "set permissions",
			run: func(rs io.ReadSeeker) error {
				return SetPermissions(rs, io.Discard, model.NewDefaultConfiguration())
			},
			want: "set permissions: prepare PDF context: read context:",
		},
		{
			name: "get permissions",
			run: func(rs io.ReadSeeker) error {
				_, err := GetPermissions(rs, model.NewDefaultConfiguration())
				return err
			},
			want: "get permissions: read context:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(&cryptoErrorReadSeeker{err: readErr})
			if !errors.Is(err, readErr) {
				t.Fatalf("got %v, want %v", err, readErr)
			}
			if !strings.HasPrefix(err.Error(), tt.want) {
				t.Fatalf("expected prefix %q, got %q", tt.want, err)
			}
			op := strings.SplitN(tt.want, ":", 2)[0] + ":"
			if strings.Count(err.Error(), op) != 1 {
				t.Fatalf("expected one %q label, got %q", op, err)
			}
		})
	}
}

// TestSetPermissionsPreservesSentinelAndWriteError verifies semantic and output causes remain discoverable.
func TestSetPermissionsPreservesSentinelAndWriteError(t *testing.T) {
	inFile, encryptedFile := encryptedCryptoTestFile(t)

	t.Run("unencrypted", func(t *testing.T) {
		f, err := os.Open(inFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		err = SetPermissions(f, io.Discard, model.NewDefaultConfiguration())
		if !errors.Is(err, pdfcpu.ErrNotEncrypted) ||
			!strings.Contains(err.Error(), "set permissions: prepare PDF context") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("write", func(t *testing.T) {
		f, err := os.Open(encryptedFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		conf := model.NewDefaultConfiguration()
		conf.UserPW = "user"
		conf.OwnerPW = "owner"
		conf.Permissions = model.PermissionsAll
		writeErr := errors.New("write failed")
		err = SetPermissions(f, securityErrorWriter{err: writeErr}, conf)
		if !errors.Is(err, writeErr) ||
			!strings.Contains(err.Error(), "set permissions: write output") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		f, err := os.Open(encryptedFile)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		conf := model.NewDefaultConfiguration()
		conf.UserPW = "user"
		conf.OwnerPW = "wrong"
		err = SetPermissions(f, io.Discard, conf)
		if !errors.Is(err, pdfcpu.ErrOwnerPasswordRequired) ||
			!strings.Contains(err.Error(), "set permissions: prepare PDF context") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestSetPermissionsFilePreservesExistingOutputOnFailure verifies protected output handling for permission updates.
func TestSetPermissionsFilePreservesExistingOutputOnFailure(t *testing.T) {
	requireExistingSecurityOutputPreserved(t, "set permissions", func(inFile, outFile string) error {
		return SetPermissionsFile(inFile, outFile, model.NewDefaultConfiguration())
	})
}

// TestPermissionFileIOErrorsIncludeOperationContext verifies file entry points identify open and create phases.
func TestPermissionFileIOErrorsIncludeOperationContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")

	err := SetPermissionsFile(missingInput, "", model.NewDefaultConfiguration())
	if !errors.Is(err, os.ErrNotExist) ||
		!strings.Contains(err.Error(), "set permissions: open input "+missingInput) {
		t.Fatalf("unexpected set open error: %v", err)
	}

	err = SetPermissionsFile(inFile, missingOutput, model.NewDefaultConfiguration())
	if !errors.Is(err, os.ErrNotExist) ||
		!strings.Contains(err.Error(), "set permissions: create output") {
		t.Fatalf("unexpected set create error: %v", err)
	}

	_, err = GetPermissionsFile(missingInput, model.NewDefaultConfiguration())
	if !errors.Is(err, os.ErrNotExist) ||
		!strings.Contains(err.Error(), "get permissions: open input "+missingInput) {
		t.Fatalf("unexpected get open error: %v", err)
	}
}

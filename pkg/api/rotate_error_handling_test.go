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
)

// TestRotateArgumentValidation verifies public rotate argument validation.
func TestRotateArgumentValidation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "reader", err: Rotate(nil, io.Discard, 90, nil, nil), want: ErrMissingPDFReadSeeker},
		{name: "writer", err: Rotate(bytes.NewReader(nil), nil, 90, nil, nil), want: ErrMissingPDFWriter},
		{name: "rotation", err: Rotate(bytes.NewReader(nil), io.Discard, 45, nil, nil), want: ErrInvalidRotation},
		{name: "input", err: RotateFile("", "", 90, nil, nil), want: ErrMissingPDFInput},
		{name: "file rotation", err: RotateFile("missing.pdf", "", -45, nil, nil), want: ErrInvalidRotation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestValidateRotation verifies accepted and rejected rotation values.
func TestValidateRotation(t *testing.T) {
	for _, rotation := range []int{-180, -90, 0, 90, 180} {
		if err := validateRotation(rotation); err != nil {
			t.Fatalf("rotation %d: %v", rotation, err)
		}
	}

	err := validateRotation(1)
	if !errors.Is(err, ErrInvalidRotation) {
		t.Fatalf("expected %v, got %v", ErrInvalidRotation, err)
	}
	if !strings.Contains(err.Error(), "multiple of 90") {
		t.Fatalf("expected rotation constraint, got %q", err)
	}
}

// TestRotateReadErrorIncludesPhaseContext verifies preparation error context.
func TestRotateReadErrorIncludesPhaseContext(t *testing.T) {
	err := Rotate(bytes.NewReader(nil), io.Discard, 90, nil, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "rotate: prepare PDF context") {
		t.Fatalf("expected rotate preparation context, got %q", err.Error())
	}
}

// TestRotatePageSelectionErrorIncludesPhaseContext verifies page-selection error context.
func TestRotatePageSelectionErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err := Rotate(openAPITestPDF(t, inFile), io.Discard, 90, []string{"foo"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "rotate: parse page selection") {
		t.Fatalf("expected page selection context, got %q", err.Error())
	}
}

// TestRotateWriteErrorIncludesPhaseContext verifies output-writing error context.
func TestRotateWriteErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("rotate write failed")
	err := Rotate(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, 90, []string{"1"}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "rotate: write output") {
		t.Fatalf("expected write output context, got %q", err.Error())
	}
}

// TestRotateFileOpenErrorIncludesInputContext verifies input-opening error context.
func TestRotateFileOpenErrorIncludesInputContext(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "missing.pdf")
	err := RotateFile(inFile, "", 90, nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "rotate: open input "+inFile) {
		t.Fatalf("expected input context, got %q", err.Error())
	}
}

// TestRotateFileCreateOutputErrorIncludesPhaseContext verifies output-creation error context.
func TestRotateFileCreateOutputErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
	err := RotateFile(inFile, outFile, 90, nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "rotate: create output") {
		t.Fatalf("expected output creation context, got %q", err.Error())
	}
}

// TestRotateCloseFilePreservesCause verifies close error wrapping.
func TestRotateCloseFilePreservesCause(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "rotate-close-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = closeFile(f, "rotate: close input")
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "rotate: close input") {
		t.Fatalf("expected close context, got %q", err.Error())
	}
}

func newRotateTestFile(t *testing.T, pattern string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestCleanupRotateFilesPreservesOperationAndCleanupErrors verifies failed-operation cleanup.
// TestFinalizeRotateFilesOutputCloseFailureCleansUp verifies failed output finalization.
// TestFinalizeRotateFilesInputCloseFailureCleansUpReplacement verifies failed input finalization.
// TestFinalizeRotateFilesRenameErrorIncludesPhaseContext verifies replacement error context.
// TestFinalizeRotateFilesReplacesInput verifies successful in-place finalization.
// TestRotateFileOperationFailureRemovesOutput verifies failed output cleanup.
func TestRotateFileOperationFailureRemovesOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	err := RotateFile(inFile, outFile, 90, []string{"foo"}, nil)
	if err == nil || !strings.Contains(err.Error(), "rotate: parse page selection") {
		t.Fatalf("expected page selection error, got %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed output removal, got %v", statErr)
	}
}

// TestRotateFileFailurePreservesExistingOutput verifies delayed output replacement.
func TestRotateFileFailurePreservesExistingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := RotateFile(inFile, outFile, 90, []string{"foo"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestRotateFileSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestRotateFileSuccessReplacesExistingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RotateFile(inFile, outFile, 90, []string{"1"}, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Fatalf("expected replacement PDF, got %q", got)
	}
}

func copyRotateTestInput(t *testing.T) (string, []byte) {
	t.Helper()
	source := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	b, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	inFile := filepath.Join(t.TempDir(), "in.pdf")
	if err := os.WriteFile(inFile, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return inFile, b
}

// TestRotateFileFailurePreservesInputAliases verifies symlink and hardlink safety.
func TestRotateFileFailurePreservesInputAliases(t *testing.T) {
	tests := []struct {
		name string
		link func(string, string) error
	}{
		{name: "symlink", link: os.Symlink},
		{name: "hardlink", link: os.Link},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inFile, want := copyRotateTestInput(t)
			outFile := filepath.Join(t.TempDir(), "alias.pdf")
			if err := tt.link(inFile, outFile); err != nil {
				t.Skipf("create %s: %v", tt.name, err)
			}

			err := RotateFile(inFile, outFile, 90, []string{"foo"}, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			for _, fileName := range []string{inFile, outFile} {
				got, readErr := os.ReadFile(fileName)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s changed after failed rotation", fileName)
				}
			}
		})
	}
}

// TestRotateFileSuccessReplacesInputAliases verifies successful alias-path replacement.
func TestRotateFileSuccessReplacesInputAliases(t *testing.T) {
	tests := []struct {
		name string
		link func(string, string) error
	}{
		{name: "symlink", link: os.Symlink},
		{name: "hardlink", link: os.Link},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inFile, want := copyRotateTestInput(t)
			inputInfo, err := os.Stat(inFile)
			if err != nil {
				t.Fatal(err)
			}
			outFile := filepath.Join(t.TempDir(), "alias.pdf")
			if err := tt.link(inFile, outFile); err != nil {
				t.Skipf("create %s: %v", tt.name, err)
			}

			if err := RotateFile(inFile, outFile, 90, []string{"1"}, nil); err != nil {
				t.Fatal(err)
			}
			gotInput, err := os.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotInput, want) {
				t.Fatal("original input changed after successful alias replacement")
			}
			inputInfoAfter, err := os.Stat(inFile)
			if err != nil {
				t.Fatal(err)
			}
			outInfo, err := os.Stat(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(inputInfo, inputInfoAfter) {
				t.Fatal("original input inode changed")
			}
			if os.SameFile(inputInfoAfter, outInfo) {
				t.Fatal("output alias still references the input inode")
			}
			gotOutput, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(gotOutput, []byte("%PDF-")) {
				t.Fatalf("expected rotated PDF output, got %q", gotOutput)
			}
		})
	}
}

// TestRotateRemoveFilePreservesCause verifies removal error wrapping.
func TestRotateRemoveFilePreservesCause(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "non-empty")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := removeFile(dir, "rotate: remove output")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "rotate: remove output") {
		t.Fatalf("expected removal context, got %q", err.Error())
	}
}

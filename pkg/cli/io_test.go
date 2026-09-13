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

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type cancelingReader struct {
	cancel context.CancelFunc
	reads  int
}

func (r *cancelingReader) Read(p []byte) (int, error) {
	r.reads++
	if r.reads > 1 {
		return 0, errors.New("read after cancellation")
	}
	n := copy(p, "partial stdin")
	r.cancel()
	return n, nil
}

// TestReadSeekerFromStdinSpoolsToTemporaryInput verifies the corresponding behavior.
func TestReadSeekerFromStdinSpoolsToTemporaryInput(t *testing.T) {
	stdin := os.Stdin
	source, err := os.CreateTemp(t.TempDir(), "stdin-source-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte("pdf-data"), 1<<17)
	if _, err := source.Write(want); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	os.Stdin = source
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = source.Close()
	})

	in, err := readSeekerFromStdin(t.Context(), nil, "test operation")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := in.path
	if tmpPath == source.Name() {
		t.Fatal("expected a distinct temporary input")
	}
	got, err := io.ReadAll(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("spooled input length: got %d, want %d", len(got), len(want))
	}
	if err := in.finalize("test operation", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tmpPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temporary input removal, got %v", err)
	}
}

// TestReadSeekerFromStdinCancellationRemovesTemporaryInput verifies cancellation while spooling stdin.
func TestReadSeekerFromStdinCancellationRemovesTemporaryInput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TMPDIR", tmpDir)
	ctx, cancel := context.WithCancel(t.Context())
	r := &cancelingReader{cancel: cancel}

	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxInputBytes = int64(len("partial stdin"))
	in, err := readSeekerFromReader(ctx, conf, "test operation", r)
	if in != nil {
		t.Fatal("expected no temporary input")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if !strings.Contains(err.Error(), "test operation: read stdin") {
		t.Fatalf("expected contextual stdin error, got %q", err)
	}
	if r.reads != 1 {
		t.Fatalf("stdin reads: got %d, want 1", r.reads)
	}
	matches, globErr := filepath.Glob(filepath.Join(tmpDir, "pdfcpu-stdin-*.pdf"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary stdin was not removed: %v", matches)
	}
}

// TestTemporaryInputCleanupJoinsContextualFailures verifies the corresponding behavior.
func TestTemporaryInputCleanupJoinsContextualFailures(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdin-cleanup-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	removeErr := errors.New("remove failed")
	in := &temporaryInput{
		file:   f,
		path:   f.Name(),
		remove: func(string) error { return removeErr },
	}
	opErr := errors.New("operation failed")

	err = in.finalize("test operation", opErr)
	for _, wantErr := range []error{opErr, removeErr} {
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	}
	for _, want := range []string{"test operation: close temporary input", "test operation: remove temporary input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %q", want, err.Error())
		}
	}
}

// TestTemporaryStdinRemovedWhenOutputCreationFails verifies the corresponding behavior.
func TestTemporaryStdinRemovedWhenOutputCreationFails(t *testing.T) {
	stdin := os.Stdin
	source, err := os.CreateTemp(t.TempDir(), "stdin-source-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.WriteString("%PDF-1.7\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	os.Stdin = source
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = source.Close()
	})

	tmpDir := t.TempDir()
	t.Setenv("TMPDIR", tmpDir)
	outFile := filepath.Join(tmpDir, "missing", "out.pdf")
	_, _, _, err = streamInOutForOperation(t.Context(), nil, "-", outFile, "test operation")
	if err == nil {
		t.Fatal("expected output creation failure")
	}
	if !strings.Contains(err.Error(), "test operation: create output") {
		t.Fatalf("expected contextual output creation error, got %q", err)
	}

	matches, err := filepath.Glob(filepath.Join(tmpDir, "pdfcpu-stdin-*.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary stdin was not removed: %v", matches)
	}
}

// TestStreamInOutFailurePreservesExistingOutput verifies staged CLI publication.
func TestStreamInOutFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	_, w, finalize, err := streamInOutForOperation(t.Context(), nil, "", outFile, "test operation")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("partial output")); err != nil {
		t.Fatal(err)
	}
	processErr := errors.New("process failed")
	if err := finalize(processErr); !errors.Is(err, processErr) {
		t.Fatalf("expected %v, got %v", processErr, err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

// TestStdinStorageCreationFailures verifies storage errors survive CLI context wrapping without touching output.
func TestStdinStorageCreationFailures(t *testing.T) {
	for _, cause := range []error{os.ErrPermission, syscall.ENOSPC} {
		t.Run(cause.Error(), func(t *testing.T) {
			originalCreate := createTemporaryInputFile
			t.Cleanup(func() { createTemporaryInputFile = originalCreate })
			createTemporaryInputFile = func(string, string) (*os.File, error) {
				return nil, &os.PathError{Op: "open", Path: "stdin-spool", Err: cause}
			}
			outFile := filepath.Join(t.TempDir(), "output.pdf")
			if err := os.WriteFile(outFile, []byte("original"), 0600); err != nil {
				t.Fatal(err)
			}
			_, _, _, err := streamInOutForOperation(t.Context(), nil, "-", outFile, "test storage")
			if !errors.Is(err, cause) || !strings.Contains(err.Error(), "create temporary input") {
				t.Fatalf("expected contextual storage error, got %v", err)
			}
			bb, err := os.ReadFile(outFile)
			if err != nil || string(bb) != "original" {
				t.Fatalf("destination changed: %q, %v", bb, err)
			}
		})
	}
}

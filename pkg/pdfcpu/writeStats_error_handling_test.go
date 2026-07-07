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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type appendStatsTestFile struct {
	writeErrs []error
	writes    []string
	closeErr  error
	closed    bool
}

func (f *appendStatsTestFile) WriteString(s string) (int, error) {
	f.writes = append(f.writes, s)
	i := len(f.writes) - 1
	if i < len(f.writeErrs) && f.writeErrs[i] != nil {
		return 0, f.writeErrs[i]
	}
	return len(s), nil
}

func (f *appendStatsTestFile) Close() error {
	f.closed = true
	return f.closeErr
}

func appendStatsTestContext(t *testing.T, fileName string) *model.Context {
	t.Helper()
	ctx, err := model.NewContext(bytes.NewReader(nil), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	version := model.V17
	size := 0
	ctx.HeaderVersion = &version
	ctx.Size = &size
	ctx.StatsFileName = fileName
	return ctx
}

func createAppendStatsTestFile(file statsAppendFile) openStatsAppendFile {
	call := 0
	return func(string, int, os.FileMode) (statsAppendFile, error) {
		call++
		if call == 1 {
			return nil, os.ErrNotExist
		}
		return file, nil
	}
}

// TestAppendStatsHeaderWriteFailureClosesFile verifies new-file header errors
// cannot bypass close handling.
func TestAppendStatsHeaderWriteFailureClosesFile(t *testing.T) {
	writeErr := errors.New("write header")
	file := &appendStatsTestFile{writeErrs: []error{writeErr}}

	err := appendStatsFile(appendStatsTestContext(t, "stats.csv"), createAppendStatsTestFile(file))
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected %v, got %v", writeErr, err)
	}
	if !file.closed {
		t.Fatal("expected file to be closed")
	}
	if got, want := len(file.writes), 1; got != want {
		t.Fatalf("writes: got %d, want %d", got, want)
	}
}

// TestAppendStatsLineWriteFailureClosesFile verifies append write errors cannot
// bypass close handling.
func TestAppendStatsLineWriteFailureClosesFile(t *testing.T) {
	writeErr := errors.New("write statistics line")
	file := &appendStatsTestFile{writeErrs: []error{writeErr}}
	openFile := func(string, int, os.FileMode) (statsAppendFile, error) {
		return file, nil
	}

	err := appendStatsFile(appendStatsTestContext(t, "stats.csv"), openFile)
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected %v, got %v", writeErr, err)
	}
	if !file.closed {
		t.Fatal("expected file to be closed")
	}
	if got, want := len(file.writes), 1; got != want {
		t.Fatalf("writes: got %d, want %d", got, want)
	}
}

// TestAppendStatsCloseFailureIsReturned verifies successful writes do not hide
// close failures.
func TestAppendStatsCloseFailureIsReturned(t *testing.T) {
	closeErr := errors.New("close statistics file")
	file := &appendStatsTestFile{closeErr: closeErr}
	openFile := func(string, int, os.FileMode) (statsAppendFile, error) {
		return file, nil
	}

	err := appendStatsFile(appendStatsTestContext(t, "stats.csv"), openFile)
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected %v, got %v", closeErr, err)
	}
}

// TestAppendStatsWriteAndCloseFailuresAreJoined verifies both lifecycle errors
// remain inspectable.
func TestAppendStatsWriteAndCloseFailuresAreJoined(t *testing.T) {
	writeErr := errors.New("write statistics line")
	closeErr := errors.New("close statistics file")
	file := &appendStatsTestFile{
		writeErrs: []error{writeErr},
		closeErr:  closeErr,
	}
	openFile := func(string, int, os.FileMode) (statsAppendFile, error) {
		return file, nil
	}

	err := appendStatsFile(appendStatsTestContext(t, "stats.csv"), openFile)
	if !errors.Is(err, writeErr) || !errors.Is(err, closeErr) {
		t.Fatalf("expected write and close failures, got %v", err)
	}
}

// TestAppendStatsFileAppendsWithoutRewritingHeader verifies the existing
// append/create behavior remains unchanged.
func TestAppendStatsFileAppendsWithoutRewritingHeader(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "stats.csv")
	ctx := appendStatsTestContext(t, fileName)

	if err := AppendStatsFile(ctx); err != nil {
		t.Fatal(err)
	}
	if err := AppendStatsFile(ctx); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}
	want := *statsHeadLine() + *statsLine(ctx) + *statsLine(ctx)
	if string(got) != want {
		t.Fatalf("statistics output differs: got %d bytes, want %d", len(got), len(want))
	}
}

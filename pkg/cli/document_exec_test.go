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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type validationErrorWriter struct {
	err error
}

func (w validationErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestValidateSingleValidFile(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	if _, err := Validate(ValidateCommand([]string{inFile}, nil)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSingleMissingFilePreservesNotExist(t *testing.T) {
	_, err := Validate(ValidateCommand([]string{"missing.pdf"}, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
}

func TestValidateMultipleFilesReturnsJoinedErrorsWithoutReporter(t *testing.T) {
	inFiles := []string{"missing1.pdf", "missing2.pdf"}

	_, err := Validate(ValidateCommand(inFiles, nil))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	for _, fn := range inFiles {
		if !bytes.Contains([]byte(err.Error()), []byte(fn)) {
			t.Fatalf("expected %q in error, got %q", fn, err.Error())
		}
	}
}

func TestValidateMultipleFilesStreamsFailuresAndContinues(t *testing.T) {
	validFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inFiles := []string{"missing1.pdf", validFile, "missing2.pdf"}
	cmd := ValidateCommand(inFiles, nil)
	var errorOutput bytes.Buffer
	cmd.ErrorOutput = &errorOutput

	_, err := Validate(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "validation failed: 2 of 3 files invalid" {
		t.Fatalf("got %q", err.Error())
	}

	got := errorOutput.String()
	first := strings.Index(got, "missing1.pdf")
	second := strings.Index(got, "missing2.pdf")
	if first < 0 || second < 0 {
		t.Fatalf("expected both failures in error output, got %q", got)
	}
	if first >= second {
		t.Fatalf("expected input order in error output, got %q", got)
	}
	for _, fn := range []string{"missing1.pdf", "missing2.pdf"} {
		if strings.Count(got, fn) != 2 {
			t.Fatalf("expected no additional multi-file wrapping for %q, got %q", fn, got)
		}
	}
}

func TestValidateProgressPrecedesEachInput(t *testing.T) {
	validFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inFiles := []string{"missing.pdf", validFile}
	cmd := ValidateCommand(inFiles, nil)
	var errorOutput bytes.Buffer
	cmd.BoolVal1 = true
	cmd.ErrorOutput = &errorOutput

	_, err := Validate(cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	got := errorOutput.String()
	missingProgress := strings.Index(got, "validating(mode=relaxed) missing.pdf ...")
	missingFailure := strings.Index(got, "validate: open missing.pdf")
	validProgress := strings.Index(got, "validating(mode=relaxed) "+validFile+" ...")
	if missingProgress < 0 || missingFailure < 0 || validProgress < 0 {
		t.Fatalf("expected progress and failure output, got %q", got)
	}
	if missingProgress >= missingFailure || missingFailure >= validProgress {
		t.Fatalf("expected progress and failure in input order, got %q", got)
	}
}

func TestValidateProgressLabelsStandardInput(t *testing.T) {
	withStdinFile(t, "not a pdf")
	cmd := ValidateCommand([]string{"-"}, nil)
	var errorOutput bytes.Buffer
	cmd.BoolVal1 = true
	cmd.ErrorOutput = &errorOutput

	if _, err := Validate(cmd); err == nil {
		t.Fatal("expected error")
	}
	if got, want := errorOutput.String(), "validating(mode=relaxed) stdin ...\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateProgressWriterFailurePreservesCause(t *testing.T) {
	wantErr := errors.New("progress writer failed")
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	cmd := ValidateCommand([]string{inFile}, nil)
	cmd.BoolVal1 = true
	cmd.ErrorOutput = validationErrorWriter{err: wantErr}

	_, err := Validate(cmd)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "write validation progress") {
		t.Fatalf("expected progress write context, got %q", err.Error())
	}
}

func TestDumpMissingFilePreservesNotExist(t *testing.T) {
	cmd := DumpCommand("missing.pdf", []int{0, 0}, nil)

	_, err := Dump(cmd)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
}

func TestOptimizeCLIPlumbingErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(t *testing.T) error
		wantErr error
		want    string
	}{
		{
			name: "read stdin",
			fn: func(t *testing.T) error {
				withStdinFile(t, "")
				_, err := Optimize(OptimizeCommand("-", "-", nil))
				return err
			},
			want: "optimize: read stdin",
		},
		{
			name: "open input",
			fn: func(t *testing.T) error {
				_, err := Optimize(OptimizeCommand("missing.pdf", "-", nil))
				return err
			},
			wantErr: os.ErrNotExist,
			want:    "optimize: open input missing.pdf",
		},
		{
			name: "create output",
			fn: func(t *testing.T) error {
				withStdinFile(t, "not a pdf")
				outFile := filepath.Join(t.TempDir(), "missing-dir", "out.pdf")
				_, err := Optimize(OptimizeCommand("-", outFile, nil))
				return err
			},
			wantErr: os.ErrNotExist,
			want:    "optimize: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestTrimStreamSetupErrorsIncludePhaseContext(t *testing.T) {
	withStdinFile(t, "")

	_, err := Trim(TrimCommand("-", "-", []string{"1"}, nil))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "trim: prepare input/output") {
		t.Fatalf("expected trim input/output context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "stdin is empty") {
		t.Fatalf("expected stdin setup error, got %q", err.Error())
	}
}

func withStdinFile(t *testing.T, content string) {
	t.Helper()

	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = f.Close()
	})
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
}

func TestMergeCreateRawRemovesOutputOnFailure(t *testing.T) {
	withStdinFile(t, "not a pdf")

	outFile := filepath.Join(t.TempDir(), "out.pdf")
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	cmd := MergeCreateCommand([]string{"-", inFile}, outFile, false, nil)

	_, err := MergeCreate(cmd)
	if err == nil {
		t.Fatal("expected merge failure")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

// TestListInfoFileJSONClosesEachInputBeforeNext verifies multi-file processing
// never accumulates deferred file descriptors.
func TestListInfoFileJSONClosesEachInputBeforeNext(t *testing.T) {
	const fileCount = 128
	dir := t.TempDir()
	files := make([]string, fileCount)
	for i := range files {
		files[i] = filepath.Join(dir, fmt.Sprintf("input-%03d.pdf", i))
		if err := os.WriteFile(files[i], nil, 0600); err != nil {
			t.Fatal(err)
		}
	}

	var previous *os.File
	process := func(
		rs io.ReadSeeker,
		_ string,
		_ []string,
		_ bool,
		_ *model.Configuration,
	) (*pdfcpu.PDFInfo, error) {
		if previous != nil {
			if got, want := previous.Fd(), ^uintptr(0); got != want {
				t.Fatalf("previous input descriptor: got %d, want %d", got, want)
			}
		}
		previous = rs.(*os.File)
		return &pdfcpu.PDFInfo{}, nil
	}

	for _, fileName := range files {
		if _, err := listInfoFileJSON(fileName, nil, false, nil, process); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := previous.Fd(), ^uintptr(0); got != want {
		t.Fatalf("last input descriptor: got %d, want %d", got, want)
	}
}

// TestListInfoFileJSONJoinsProcessAndCloseFailures verifies neither lifecycle
// failure is lost.
func TestListInfoFileJSONJoinsProcessAndCloseFailures(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "input.pdf")
	if err := os.WriteFile(fileName, nil, 0600); err != nil {
		t.Fatal(err)
	}
	processErr := errors.New("process input")
	process := func(
		rs io.ReadSeeker,
		_ string,
		_ []string,
		_ bool,
		_ *model.Configuration,
	) (*pdfcpu.PDFInfo, error) {
		if err := rs.(*os.File).Close(); err != nil {
			t.Fatal(err)
		}
		return nil, processErr
	}

	_, err := listInfoFileJSON(fileName, nil, false, nil, process)
	if !errors.Is(err, processErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected process and close failures, got %v", err)
	}
}

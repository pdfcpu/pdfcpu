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
			if _, err := previous.Stat(); !errors.Is(err, os.ErrClosed) {
				t.Fatalf("previous input remains open: %v", err)
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
	if _, err := previous.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("last input remains open: %v", err)
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

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
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func resizeTestConfiguration() *model.Resize {
	return &model.Resize{Scale: 0.5}
}

func resizeTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

// TestResizeArgumentValidation verifies public resize boundary guards.
func TestResizeArgumentValidation(t *testing.T) {
	res := resizeTestConfiguration()
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "reader", err: Resize(nil, io.Discard, nil, res, nil), want: ErrMissingPDFReadSeeker},
		{name: "writer", err: Resize(bytes.NewReader(nil), nil, nil, res, nil), want: ErrMissingPDFWriter},
		{name: "configuration", err: Resize(bytes.NewReader(nil), io.Discard, nil, nil, nil), want: ErrMissingResizeConfiguration},
		{name: "input", err: ResizeFile("", "", nil, res, nil), want: ErrMissingPDFInput},
		{name: "file configuration", err: ResizeFile("missing.pdf", "", nil, nil, nil), want: ErrMissingResizeConfiguration},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestResizeRejectsInvalidConfigurations verifies direct API configuration validation.
func TestResizeRejectsInvalidConfigurations(t *testing.T) {
	tests := []struct {
		name string
		res  *model.Resize
		want string
	}{
		{name: "empty", res: &model.Resize{}, want: "missing scale factor or dimensions"},
		{name: "negative scale", res: &model.Resize{Scale: -0.5}, want: "scale factor"},
		{name: "identity scale", res: &model.Resize{Scale: 1}, want: "scale factor"},
		{name: "non-finite scale", res: &model.Resize{Scale: math.Inf(1)}, want: "scale factor"},
		{name: "scale and dimensions", res: &model.Resize{Scale: 0.5, PageDim: &types.Dim{Width: 10}}, want: "scale factor"},
		{name: "zero dimensions", res: &model.Resize{PageDim: &types.Dim{}}, want: "dimensions"},
		{name: "negative dimension", res: &model.Resize{PageDim: &types.Dim{Width: -1, Height: 10}}, want: "dimensions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Resize(bytes.NewReader(nil), io.Discard, nil, tt.res, nil)
			if !errors.Is(err, ErrInvalidResizeConfiguration) {
				t.Fatalf("expected %v, got %v", ErrInvalidResizeConfiguration, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err.Error())
			}
			if errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("configuration validation must precede PDF reading: %v", err)
			}
		})
	}
}

// TestResizeConfigurationErrorsIncludeValidationContext verifies configuration errors at both public API boundaries.
func TestResizeConfigurationErrorsIncludeValidationContext(t *testing.T) {
	configurations := []struct {
		name string
		res  *model.Resize
		want error
	}{
		{name: "missing", want: ErrMissingResizeConfiguration},
		{name: "empty", res: &model.Resize{}, want: ErrInvalidResizeConfiguration},
		{name: "negative scale", res: &model.Resize{Scale: -0.5}, want: ErrInvalidResizeConfiguration},
		{name: "identity scale", res: &model.Resize{Scale: 1}, want: ErrInvalidResizeConfiguration},
		{name: "non-finite scale", res: &model.Resize{Scale: math.Inf(1)}, want: ErrInvalidResizeConfiguration},
		{name: "scale and dimensions", res: &model.Resize{Scale: 0.5, PageDim: &types.Dim{Width: 10}}, want: ErrInvalidResizeConfiguration},
		{name: "zero dimensions", res: &model.Resize{PageDim: &types.Dim{}}, want: ErrInvalidResizeConfiguration},
		{name: "negative dimension", res: &model.Resize{PageDim: &types.Dim{Width: -1, Height: 10}}, want: ErrInvalidResizeConfiguration},
	}
	entryPoints := []struct {
		name string
		run  func(*model.Resize) error
	}{
		{name: "Resize", run: func(res *model.Resize) error {
			return Resize(bytes.NewReader(nil), io.Discard, nil, res, nil)
		}},
		{name: "ResizeFile", run: func(res *model.Resize) error {
			return ResizeFile("missing.pdf", "", nil, res, nil)
		}},
	}

	for _, entryPoint := range entryPoints {
		for _, configuration := range configurations {
			t.Run(entryPoint.name+"/"+configuration.name, func(t *testing.T) {
				err := entryPoint.run(configuration.res)
				if !errors.Is(err, configuration.want) {
					t.Fatalf("expected %v, got %v", configuration.want, err)
				}
				if !strings.Contains(err.Error(), "resize: validate configuration") {
					t.Fatalf("expected resize validation context, got %q", err)
				}
			})
		}
	}
}

// TestResizeReadErrorIncludesPhaseContext verifies PDF preparation context.
func TestResizeReadErrorIncludesPhaseContext(t *testing.T) {
	err := Resize(bytes.NewReader(nil), io.Discard, nil, resizeTestConfiguration(), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "resize: prepare PDF context") {
		t.Fatalf("expected resize preparation context, got %q", err.Error())
	}
}

// TestResizePageSelectionErrorIncludesPhaseContext verifies page-selection context.
func TestResizePageSelectionErrorIncludesPhaseContext(t *testing.T) {
	err := Resize(openAPITestPDF(t, resizeTestInputFile()), io.Discard, []string{"foo"}, resizeTestConfiguration(), nil)
	if err == nil || !strings.Contains(err.Error(), "resize: parse page selection") {
		t.Fatalf("expected page-selection context, got %v", err)
	}
}

// TestResizeWriteErrorIncludesPhaseContext verifies output-writing context and cause preservation.
func TestResizeWriteErrorIncludesPhaseContext(t *testing.T) {
	wantErr := errors.New("resize write failed")
	err := Resize(openAPITestPDF(t, resizeTestInputFile()), failingWriter{err: wantErr}, []string{"1"}, resizeTestConfiguration(), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "resize: write output") {
		t.Fatalf("expected output context, got %q", err.Error())
	}
}

// TestResizeFileIOErrorContext verifies file opening and creation context.
func TestResizeFileIOErrorContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	err := ResizeFile(missingInput, "", nil, resizeTestConfiguration(), nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "resize: open input "+missingInput) {
		t.Fatalf("expected input context, got %v", err)
	}

	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	err = ResizeFile(resizeTestInputFile(), missingOutput, nil, resizeTestConfiguration(), nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "resize: create output") {
		t.Fatalf("expected output context, got %v", err)
	}
}

func newResizeTestFile(t *testing.T, pattern string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestCleanupResizeFilesPreservesOperationAndCleanupErrors verifies joined cleanup failures.
// TestFinalizeResizeFilesReplacesInput verifies successful in-place replacement.
// TestResizeFileFailurePreservesExistingOutput verifies delayed output replacement.
func TestResizeFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}
	err := ResizeFile(resizeTestInputFile(), outFile, []string{"foo"}, resizeTestConfiguration(), nil)
	if err == nil || !strings.Contains(err.Error(), "resize: parse page selection") {
		t.Fatalf("expected page-selection error, got %v", err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestResizeFileSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestResizeFileSuccessReplacesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ResizeFile(resizeTestInputFile(), outFile, []string{"1"}, resizeTestConfiguration(), nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("expected replacement PDF, got %q", b)
	}
}

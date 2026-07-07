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

func zoomTestConfiguration() *model.Zoom {
	return &model.Zoom{Factor: 0.5}
}

func zoomTestInputFile() string {
	return filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
}

// TestZoomArgumentValidation verifies public zoom boundary guards.
func TestZoomArgumentValidation(t *testing.T) {
	zoom := zoomTestConfiguration()
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "reader", err: Zoom(nil, io.Discard, nil, zoom, nil), want: ErrMissingPDFReadSeeker},
		{name: "writer", err: Zoom(bytes.NewReader(nil), nil, nil, zoom, nil), want: ErrMissingPDFWriter},
		{name: "configuration", err: Zoom(bytes.NewReader(nil), io.Discard, nil, nil, nil), want: ErrMissingZoomConfiguration},
		{name: "input", err: ZoomFile("", "", nil, zoom, nil), want: ErrMissingPDFInput},
		{name: "file configuration", err: ZoomFile("missing.pdf", "", nil, nil, nil), want: ErrMissingZoomConfiguration},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestZoomRejectsInvalidConfigurationsBeforeReading verifies direct API configuration validation.
func TestZoomRejectsInvalidConfigurationsBeforeReading(t *testing.T) {
	tests := []struct {
		name string
		zoom *model.Zoom
		want string
	}{
		{name: "empty", zoom: &model.Zoom{}, want: "exactly one factor or margin"},
		{name: "negative factor", zoom: &model.Zoom{Factor: -0.5}, want: "factor"},
		{name: "identity factor", zoom: &model.Zoom{Factor: 1}, want: "factor"},
		{name: "non-finite factor", zoom: &model.Zoom{Factor: math.Inf(1)}, want: "non-finite factor or margin"},
		{name: "non-finite horizontal margin", zoom: &model.Zoom{HMargin: math.NaN()}, want: "non-finite factor or margin"},
		{name: "NaN vertical margin", zoom: &model.Zoom{VMargin: math.NaN()}, want: "non-finite factor or margin"},
		{name: "infinite vertical margin", zoom: &model.Zoom{VMargin: math.Inf(1)}, want: "non-finite factor or margin"},
		{name: "factor and horizontal margin", zoom: &model.Zoom{Factor: 0.5, HMargin: 10}, want: "exactly one factor or margin"},
		{name: "horizontal and vertical margin", zoom: &model.Zoom{HMargin: 10, VMargin: 10}, want: "exactly one factor or margin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Zoom(bytes.NewReader(nil), io.Discard, nil, tt.zoom, nil)
			if !errors.Is(err, ErrInvalidZoomConfiguration) {
				t.Fatalf("expected %v, got %v", ErrInvalidZoomConfiguration, err)
			}
			if errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("configuration validation must precede PDF reading: %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q context, got %q", tt.want, err)
			}
		})
	}
}

// TestZoomFileRejectsInvalidConfigurationBeforeOpening verifies file API validation order.
func TestZoomFileRejectsInvalidConfigurationBeforeOpening(t *testing.T) {
	err := ZoomFile("missing.pdf", "", nil, &model.Zoom{}, nil)
	if !errors.Is(err, ErrInvalidZoomConfiguration) {
		t.Fatalf("expected %v, got %v", ErrInvalidZoomConfiguration, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("configuration validation must precede opening the PDF: %v", err)
	}
}

// TestZoomConfigurationValidationDoesNotMutate verifies caller-owned configuration remains unchanged.
func TestZoomConfigurationValidationDoesNotMutate(t *testing.T) {
	zoom := &model.Zoom{HMargin: -10}
	want := *zoom
	err := Zoom(bytes.NewReader(nil), io.Discard, nil, zoom, nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if *zoom != want {
		t.Fatalf("zoom configuration mutated: got %+v, want %+v", *zoom, want)
	}
}

// TestZoomAPIOwnsMissingFactorOrMarginValidation documents structural validation ownership.
func TestZoomAPIOwnsMissingFactorOrMarginValidation(t *testing.T) {
	zoom, err := pdfcpu.ParseZoomConfig("border:true", types.POINTS)
	if err != nil {
		t.Fatalf("parser should accept parameter-complete configuration: %v", err)
	}
	err = Zoom(bytes.NewReader(nil), io.Discard, nil, zoom, nil)
	if !errors.Is(err, ErrInvalidZoomConfiguration) {
		t.Fatalf("expected %v, got %v", ErrInvalidZoomConfiguration, err)
	}
	if !strings.Contains(err.Error(), "exactly one factor or margin") {
		t.Fatalf("expected structural configuration context, got %q", err)
	}
	if errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("API validation must precede PDF reading: %v", err)
	}
}

// TestZoomConfigurationErrorsIncludePhaseContext verifies configuration context at both API boundaries.
func TestZoomConfigurationErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "Zoom", run: func() error {
			return Zoom(bytes.NewReader(nil), io.Discard, nil, nil, nil)
		}},
		{name: "ZoomFile", run: func() error {
			return ZoomFile("missing.pdf", "", nil, nil, nil)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, ErrMissingZoomConfiguration) {
				t.Fatalf("expected %v, got %v", ErrMissingZoomConfiguration, err)
			}
			if !strings.Contains(err.Error(), "zoom: validate configuration") {
				t.Fatalf("expected zoom validation context, got %q", err)
			}
		})
	}
}

// TestZoomReadErrorIncludesPhaseContext verifies PDF preparation context.
func TestZoomReadErrorIncludesPhaseContext(t *testing.T) {
	err := Zoom(bytes.NewReader(nil), io.Discard, nil, zoomTestConfiguration(), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "zoom: prepare PDF context") {
		t.Fatalf("expected zoom preparation context, got %q", err.Error())
	}
	if got := strings.Count(err.Error(), "prepare PDF context"); got != 1 {
		t.Fatalf("expected preparation context once, got %d in %q", got, err.Error())
	}
}

// TestZoomPageSelectionErrorIncludesPhaseContext verifies page-selection context.
func TestZoomPageSelectionErrorIncludesPhaseContext(t *testing.T) {
	err := Zoom(openAPITestPDF(t, zoomTestInputFile()), io.Discard, []string{"foo"}, zoomTestConfiguration(), nil)
	if err == nil || !strings.Contains(err.Error(), "zoom: parse page selection") {
		t.Fatalf("expected page-selection context, got %v", err)
	}
}

// TestZoomApplyErrorIncludesPhaseContext verifies operation and page context propagation.
func TestZoomApplyErrorIncludesPhaseContext(t *testing.T) {
	err := Zoom(openAPITestPDF(t, zoomTestInputFile()), io.Discard, []string{"1"}, &model.Zoom{HMargin: 10000}, nil)
	if err == nil || !strings.Contains(err.Error(), "zoom: apply pages: page 1: derive factor and margins") {
		t.Fatalf("expected operation context, got %v", err)
	}
}

// TestZoomWriteErrorIncludesPhaseContext verifies output-writing context and cause preservation.
func TestZoomWriteErrorIncludesPhaseContext(t *testing.T) {
	wantErr := errors.New("zoom write failed")
	err := Zoom(openAPITestPDF(t, zoomTestInputFile()), failingWriter{err: wantErr}, []string{"1"}, zoomTestConfiguration(), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "zoom: write output") {
		t.Fatalf("expected output context, got %q", err.Error())
	}
}

// TestZoomFileIOErrorContext verifies file opening and creation context.
func TestZoomFileIOErrorContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	err := ZoomFile(missingInput, "", nil, zoomTestConfiguration(), nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "zoom: open input "+missingInput) {
		t.Fatalf("expected input context, got %v", err)
	}

	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	err = ZoomFile(zoomTestInputFile(), missingOutput, nil, zoomTestConfiguration(), nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "zoom: create output") {
		t.Fatalf("expected output context, got %v", err)
	}
}

func newZoomTestFile(t *testing.T, pattern string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestCleanupZoomFilesPreservesOperationAndCleanupErrors verifies joined cleanup failures.
// TestCleanupZoomFilesPreservesPrimaryCloseAndRemovalErrors verifies all cleanup causes remain discoverable.
// TestFinalizeZoomFilesOutputCloseFailureCleansUp verifies failed output finalization.
// TestFinalizeZoomFilesInputCloseFailureCleansUp verifies failed input finalization.
// TestFinalizeZoomFilesReplacesInput verifies successful in-place replacement.
// TestFinalizeZoomFilesReplaceOutFailureCleansUp verifies replaceOut rename failure and cleanup.
// TestZoomFileFailurePreservesExistingOutput verifies delayed output replacement.
func TestZoomFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}
	err := ZoomFile(zoomTestInputFile(), outFile, []string{"foo"}, zoomTestConfiguration(), nil)
	if err == nil || !strings.Contains(err.Error(), "zoom: parse page selection") {
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

// TestZoomFileSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestZoomFileSuccessReplacesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ZoomFile(zoomTestInputFile(), outFile, []string{"1"}, zoomTestConfiguration(), nil); err != nil {
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

// TestZoomDoesNotMutateConfiguration verifies successful processing preserves caller-owned data.
func TestZoomDoesNotMutateConfiguration(t *testing.T) {
	zoom := &model.Zoom{HMargin: -10}
	want := *zoom
	var out bytes.Buffer
	if err := Zoom(openAPITestPDF(t, zoomTestInputFile()), &out, []string{"1"}, zoom, nil); err != nil {
		t.Fatal(err)
	}
	if *zoom != want {
		t.Fatalf("zoom configuration mutated: got %+v, want %+v", *zoom, want)
	}
}

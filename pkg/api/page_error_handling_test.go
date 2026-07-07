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
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestValidatePageConfigurationAllowsInheritedDimensions verifies optional dimension semantics.
func TestValidatePageConfigurationAllowsInheritedDimensions(t *testing.T) {
	tests := []struct {
		name     string
		pageConf *pdfcpu.PageConfiguration
	}{
		{name: "nil configuration"},
		{name: "nil dimensions", pageConf: &pdfcpu.PageConfiguration{}},
		{
			name:     "positive finite dimensions",
			pageConf: &pdfcpu.PageConfiguration{PageDim: &types.Dim{Width: 10, Height: 20}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePageConfiguration(tt.pageConf); err != nil {
				t.Fatalf("expected valid configuration, got %v", err)
			}
		})
	}
}

// TestInsertPagesAcceptsValidPageConfiguration verifies both public page insertion boundaries.
func TestInsertPagesAcceptsValidPageConfiguration(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	entryPoints := []struct {
		name string
		run  func(*pdfcpu.PageConfiguration) error
	}{
		{
			name: "InsertPages",
			run: func(pageConf *pdfcpu.PageConfiguration) error {
				return InsertPages(openAPITestPDF(t, inFile), io.Discard, []string{"1"}, false, pageConf, nil)
			},
		},
		{
			name: "InsertPagesFile",
			run: func(pageConf *pdfcpu.PageConfiguration) error {
				return InsertPagesFile(inFile, filepath.Join(t.TempDir(), "out.pdf"), []string{"1"}, false, pageConf, nil)
			},
		},
	}
	configurations := []struct {
		name     string
		pageConf *pdfcpu.PageConfiguration
	}{
		{name: "nil configuration"},
		{
			name:     "positive finite dimensions",
			pageConf: &pdfcpu.PageConfiguration{PageDim: &types.Dim{Width: 10, Height: 20}},
		},
	}

	for _, entryPoint := range entryPoints {
		t.Run(entryPoint.name, func(t *testing.T) {
			for _, configuration := range configurations {
				t.Run(configuration.name, func(t *testing.T) {
					if err := entryPoint.run(configuration.pageConf); err != nil {
						t.Fatalf("expected valid configuration, got %v", err)
					}
				})
			}
		})
	}
}

// TestInsertPagesRejectsInvalidPageConfiguration verifies configuration validation and ordering.
func TestInsertPagesRejectsInvalidPageConfiguration(t *testing.T) {
	tests := []struct {
		name string
		dim  types.Dim
	}{
		{name: "zero width", dim: types.Dim{Height: 10}},
		{name: "negative width", dim: types.Dim{Width: -1, Height: 10}},
		{name: "NaN width", dim: types.Dim{Width: math.NaN(), Height: 10}},
		{name: "positive infinite width", dim: types.Dim{Width: math.Inf(1), Height: 10}},
		{name: "negative infinite width", dim: types.Dim{Width: math.Inf(-1), Height: 10}},
		{name: "zero height", dim: types.Dim{Width: 10}},
		{name: "negative height", dim: types.Dim{Width: 10, Height: -1}},
		{name: "NaN height", dim: types.Dim{Width: 10, Height: math.NaN()}},
		{name: "positive infinite height", dim: types.Dim{Width: 10, Height: math.Inf(1)}},
		{name: "negative infinite height", dim: types.Dim{Width: 10, Height: math.Inf(-1)}},
	}
	entryPoints := []struct {
		name string
		run  func(*pdfcpu.PageConfiguration) error
	}{
		{
			name: "InsertPages",
			run: func(pageConf *pdfcpu.PageConfiguration) error {
				return InsertPages(bytes.NewReader(nil), io.Discard, nil, false, pageConf, nil)
			},
		},
		{
			name: "InsertPagesFile",
			run: func(pageConf *pdfcpu.PageConfiguration) error {
				return InsertPagesFile("missing.pdf", "", nil, false, pageConf, nil)
			},
		},
	}

	for _, entryPoint := range entryPoints {
		t.Run(entryPoint.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					pageConf := &pdfcpu.PageConfiguration{PageDim: &tt.dim}
					err := entryPoint.run(pageConf)
					if !errors.Is(err, ErrInvalidPageConfiguration) {
						t.Fatalf("expected %v, got %v", ErrInvalidPageConfiguration, err)
					}
					if !strings.Contains(err.Error(), "insert pages: validate page configuration") {
						t.Fatalf("expected validation context, got %q", err)
					}
					if errors.Is(err, pdfcpu.ErrEmptyInput) || errors.Is(err, os.ErrNotExist) {
						t.Fatalf("configuration validation must precede PDF access: %v", err)
					}
				})
			}
		})
	}
}

// TestPageAPIMissingStreamArguments verifies the page API stream contracts.
func TestPageAPIMissingStreamArguments(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want error
	}{
		{
			name: "insert pages missing reader",
			fn: func() error {
				return InsertPages(nil, io.Discard, nil, false, nil, nil)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "insert pages missing writer",
			fn: func() error {
				return InsertPages(bytes.NewReader(nil), nil, nil, false, nil, nil)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "remove pages missing reader",
			fn: func() error {
				return RemovePages(nil, io.Discard, nil, nil)
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "remove pages missing writer",
			fn: func() error {
				return RemovePages(bytes.NewReader(nil), nil, nil, nil)
			},
			want: ErrMissingPDFWriter,
		},
		{
			name: "page count missing reader",
			fn: func() error {
				_, err := PageCount(nil, nil)
				return err
			},
			want: ErrMissingPDFReadSeeker,
		},
		{
			name: "page dimensions missing reader",
			fn: func() error {
				_, err := PageDims(nil, nil)
				return err
			},
			want: ErrMissingPDFReadSeeker,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

// TestPageAPISelectionErrorsIncludePhaseContext verifies page selection error context.
func TestPageAPISelectionErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	operations := []struct {
		name string
		fn   func([]string) error
		want string
	}{
		{
			name: "insert pages",
			fn: func(selection []string) error {
				return InsertPages(openAPITestPDF(t, inFile), io.Discard, selection, false, nil, nil)
			},
			want: "insert pages: parse page selection",
		},
		{
			name: "remove pages",
			fn: func(selection []string) error {
				return RemovePages(openAPITestPDF(t, inFile), io.Discard, selection, nil)
			},
			want: "remove pages: parse page selection",
		},
	}
	selections := []struct {
		name       string
		value      []string
		want       string
		conversion bool
	}{
		{
			name:  "empty token",
			value: []string{""},
			want:  "page selection token 1: empty",
		},
		{
			name:       "invalid second token",
			value:      []string{"1", "foo"},
			want:       `page selection token 2 "foo"`,
			conversion: true,
		},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			for _, selection := range selections {
				t.Run(selection.name, func(t *testing.T) {
					err := operation.fn(selection.value)
					if err == nil {
						t.Fatal("expected error")
					}
					for _, want := range []string{operation.want, selection.want} {
						if !strings.Contains(err.Error(), want) {
							t.Fatalf("expected %q in error, got %q", want, err)
						}
					}
					if selection.conversion {
						var numErr *strconv.NumError
						if !errors.As(err, &numErr) {
							t.Fatalf("expected conversion error, got %v", err)
						}
					}
				})
			}
		})
	}
}

// TestRemovePagesRejectsRemovingEveryPage verifies an empty result is an explicit error.
func TestRemovePagesRejectsRemovingEveryPage(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err := RemovePages(openAPITestPDF(t, inFile), io.Discard, []string{"1-"}, nil)
	if !errors.Is(err, pdfcpu.ErrMissingPageNumbers) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrMissingPageNumbers, err)
	}
	if !strings.Contains(err.Error(), "remove pages: no pages remaining") {
		t.Fatalf("expected empty-result context, got %q", err)
	}
}

// TestRemovePagesFileEmptyResultPreservesExistingOutput verifies failed output replacement.
func TestRemovePagesFileEmptyResultPreservesExistingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := RemovePagesFile(inFile, outFile, []string{"1-"}, nil)
	if !errors.Is(err, pdfcpu.ErrMissingPageNumbers) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrMissingPageNumbers, err)
	}
	if !strings.Contains(err.Error(), "remove pages: no pages remaining") {
		t.Fatalf("expected empty-result context, got %q", err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestRemovePagesFileEmptyResultPreservesInPlaceInput verifies failed in-place replacement.
func TestRemovePagesFileEmptyResultPreservesInPlaceInput(t *testing.T) {
	source := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	want, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		explicitIn bool
	}{
		{name: "implicit"},
		{name: "explicit", explicitIn: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inFile := filepath.Join(t.TempDir(), "in.pdf")
			if err := os.WriteFile(inFile, want, 0o600); err != nil {
				t.Fatal(err)
			}
			outFile := ""
			if tt.explicitIn {
				outFile = inFile
			}

			err := RemovePagesFile(inFile, outFile, []string{"1-"}, nil)
			if !errors.Is(err, pdfcpu.ErrMissingPageNumbers) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrMissingPageNumbers, err)
			}
			got, err := os.ReadFile(inFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatal("in-place input changed")
			}
		})
	}
}

// TestPageAPIWriteErrorsIncludePhaseContext verifies write error context and sentinel preservation.
func TestPageAPIWriteErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("write failed")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "insert pages",
			fn: func() error {
				return InsertPages(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, []string{"1"}, false, nil, nil)
			},
			want: "insert pages: write output",
		},
		{
			name: "remove pages",
			fn: func() error {
				return RemovePages(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, []string{"2"}, nil)
			},
			want: "remove pages: write output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err)
			}
		})
	}
}

// TestPageFileAPIMissingInput verifies the page file API input contract.
func TestPageFileAPIMissingInput(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "insert pages",
			fn: func() error {
				return InsertPagesFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, false, nil, nil)
			},
		},
		{
			name: "remove pages",
			fn: func() error {
				return RemovePagesFile("", filepath.Join(t.TempDir(), "out.pdf"), nil, nil)
			},
		},
		{
			name: "page count",
			fn: func() error {
				_, err := PageCountFile("")
				return err
			},
		},
		{
			name: "page dimensions",
			fn: func() error {
				_, err := PageDimsFile("")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrMissingPDFInput) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFInput, err)
			}
		})
	}
}

// TestPageFileAPIErrorsIncludeFileContext verifies input and output file error context.
func TestPageFileAPIErrorsIncludeFileContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "insert pages open input",
			fn: func() error {
				return InsertPagesFile(missingInput, "", nil, false, nil, nil)
			},
			want: "insert pages: open input " + missingInput,
		},
		{
			name: "remove pages open input",
			fn: func() error {
				return RemovePagesFile(missingInput, "", nil, nil)
			},
			want: "remove pages: open input " + missingInput,
		},
		{
			name: "page count open input",
			fn: func() error {
				_, err := PageCountFile(missingInput)
				return err
			},
			want: "page count: open input " + missingInput,
		},
		{
			name: "page dimensions open input",
			fn: func() error {
				_, err := PageDimsFile(missingInput)
				return err
			},
			want: "page dimensions: open input " + missingInput,
		},
		{
			name: "insert pages create output",
			fn: func() error {
				return InsertPagesFile(inFile, missingOutput, nil, false, nil, nil)
			},
			want: "insert pages: create output",
		},
		{
			name: "remove pages create output",
			fn: func() error {
				return RemovePagesFile(inFile, missingOutput, nil, nil)
			},
			want: "remove pages: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err)
			}
		})
	}
}

// TestCleanupPageFilesPreservesPrimaryError verifies joined cleanup failures.
// TestFinalizePageFilesRenameErrorIncludesPhaseContext verifies replacement error cleanup.
// TestPageFileFailurePreservesExistingOutput verifies delayed output replacement.
func TestPageFileFailurePreservesExistingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		fn   func(string) error
		want string
	}{
		{
			name: "insert pages",
			fn: func(outFile string) error {
				return InsertPagesFile(inFile, outFile, []string{"foo"}, false, nil, nil)
			},
			want: "insert pages: parse page selection",
		},
		{
			name: "remove pages",
			fn: func(outFile string) error {
				return RemovePagesFile(inFile, outFile, []string{"foo"}, nil)
			},
			want: "remove pages: parse page selection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			want := []byte("existing output")
			if err := os.WriteFile(outFile, want, 0o600); err != nil {
				t.Fatal(err)
			}
			err := tt.fn(outFile)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			got, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("existing output changed: got %q, want %q", got, want)
			}
		})
	}
}

// TestPageFileSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestPageFileSuccessReplacesExistingOutput(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inputPageCount, err := PageCountFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		fn        func(string) error
		pageDelta int
	}{
		{
			name:      "insert pages",
			pageDelta: 1,
			fn: func(outFile string) error {
				return InsertPagesFile(inFile, outFile, []string{"1"}, false, nil, nil)
			},
		},
		{
			name: "remove pages",
			fn: func(outFile string) error {
				return RemovePagesFile(inFile, outFile, []string{"2"}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "out.pdf")
			if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := tt.fn(outFile); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(got, []byte("%PDF-")) {
				t.Fatalf("expected replacement PDF, got %q", got)
			}
			pageCount, err := PageCountFile(outFile)
			if err != nil {
				t.Fatal(err)
			}
			if want := inputPageCount + tt.pageDelta; pageCount != want {
				t.Fatalf("expected %d pages, got %d", want, pageCount)
			}
		})
	}
}

// TestPageAPIReadErrorsIncludeOperationContext verifies read error context and sentinel preservation.
func TestPageAPIReadErrorsIncludeOperationContext(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "insert pages",
			fn: func() error {
				return InsertPages(bytes.NewReader(nil), io.Discard, nil, false, nil, nil)
			},
			want: "insert pages: prepare PDF context: read context",
		},
		{
			name: "remove pages",
			fn: func() error {
				return RemovePages(bytes.NewReader(nil), io.Discard, nil, nil)
			},
			want: "remove pages: prepare PDF context: read context",
		},
		{
			name: "page count",
			fn: func() error {
				_, err := PageCount(bytes.NewReader(nil), nil)
				return err
			},
			want: "page count: read context",
		},
		{
			name: "page dimensions",
			fn: func() error {
				_, err := PageDims(bytes.NewReader(nil), nil)
				return err
			},
			want: "page dimensions: read context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err)
			}
		})
	}
}

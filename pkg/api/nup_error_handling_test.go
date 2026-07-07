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
	"fmt"
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

func nUpTestConfiguration(t *testing.T, images bool) *model.NUp {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)
	if images {
		nup, err = ImageNUpConfig(4, "", nil)
	} else {
		nup, err = PDFNUpConfig(4, "", nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return nup
}

func TestNUpValuesReturnsCopy(t *testing.T) {
	values := NUpValues()
	if len(values) == 0 {
		t.Fatal("expected n-up values")
	}
	values[0] = 99
	if got := NUpValues()[0]; got != 2 {
		t.Fatalf("expected protected n-up values, got %d", got)
	}
}

func TestNUpParsersRejectMissingConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{name: "details", run: func() error { return ParseNUpDetails("", nil) }},
		{name: "value", run: func() error { return ParseNUpValue(4, nil) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrMissingNUpConfiguration) {
				t.Fatalf("expected %v, got %v", ErrMissingNUpConfiguration, err)
			}
		})
	}
}

func TestNUpEntryPointsRejectMissingConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{name: "images", run: func() error {
			_, err := NUpFromImage(nil, []string{"unused"}, nil)
			return err
		}},
		{name: "stream", run: func() error {
			return NUp(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
		}},
		{name: "file", run: func() error {
			return NUpFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nil, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrMissingNUpConfiguration) {
				t.Fatalf("expected %v, got %v", ErrMissingNUpConfiguration, err)
			}
		})
	}
}

func TestNUpEntryPointsRejectMissingInput(t *testing.T) {
	for _, tt := range []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{name: "image context", run: func() error {
			_, err := NUpFromImage(nil, nil, nUpTestConfiguration(t, true))
			return err
		}, wantErr: ErrMissingImageInput},
		{name: "image stream", run: func() error {
			return NUp(nil, io.Discard, nil, nil, nUpTestConfiguration(t, true), nil)
		}, wantErr: ErrMissingImageInput},
		{name: "image file", run: func() error {
			return NUpFile(nil, filepath.Join(t.TempDir(), "out.pdf"), nil, nUpTestConfiguration(t, true), nil)
		}, wantErr: ErrMissingImageInput},
		{name: "PDF file", run: func() error {
			return NUpFile(nil, filepath.Join(t.TempDir(), "out.pdf"), nil, nUpTestConfiguration(t, false), nil)
		}, wantErr: ErrMissingPDFInput},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNUpFileRejectsMissingOutput(t *testing.T) {
	err := NUpFile([]string{"unused"}, "", nil, nUpTestConfiguration(t, false), nil)
	if !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFOutput, err)
	}
}

func TestNUpEntryPointsRejectInvalidGrid(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*model.NUp) error
	}{
		{name: "images", run: func(nup *model.NUp) error {
			_, err := NUpFromImage(nil, []string{"unused"}, nup)
			return err
		}},
		{name: "stream", run: func(nup *model.NUp) error {
			return NUp(bytes.NewReader(nil), io.Discard, nil, nil, nup, nil)
		}},
		{name: "file", run: func(nup *model.NUp) error {
			return NUpFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nup, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nup := nUpTestConfiguration(t, false)
			nup.Grid = nil
			err := tt.run(nup)
			if err == nil || !strings.Contains(err.Error(), "n-up: prepare configuration") ||
				!strings.Contains(err.Error(), "invalid configuration: missing page grid") {
				t.Fatalf("expected missing grid error, got %v", err)
			}
		})
	}
}

func TestNUpReadErrorIncludesPhaseContext(t *testing.T) {
	err := NUp(bytes.NewReader(nil), io.Discard, nil, nil, nUpTestConfiguration(t, false), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "n-up: read and validate: read context") {
		t.Fatalf("expected read phase context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "prepare PDF context") {
		t.Fatalf("unexpected optimized preparation context: %q", err.Error())
	}
}

func TestNUpPageSelectionErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err := NUp(openAPITestPDF(t, inFile), io.Discard, nil, []string{"bogus"}, nUpTestConfiguration(t, false), nil)
	if err == nil || !strings.Contains(err.Error(), "n-up: parse page selection") {
		t.Fatalf("expected page selection context, got %v", err)
	}
}

func TestNUpWriteErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("write failed")
	err := NUp(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil, nil, nUpTestConfiguration(t, false), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "n-up: write output") {
		t.Fatalf("expected write context, got %q", err.Error())
	}
}

func TestNUpRejectsInvalidConfigurationValues(t *testing.T) {
	tests := []struct {
		name string
		nup  func() *model.NUp
		want string
	}{
		{name: "zero grid", nup: func() *model.NUp {
			nup := nUpTestConfiguration(t, true)
			nup.Grid.Width = 0
			return nup
		}, want: "invalid page grid"},
		{name: "fractional grid", nup: func() *model.NUp {
			nup := nUpTestConfiguration(t, true)
			nup.Grid.Height = 1.5
			return nup
		}, want: "invalid page grid"},
		{name: "infinite grid", nup: func() *model.NUp {
			nup := nUpTestConfiguration(t, true)
			nup.Grid.Width = math.Inf(1)
			return nup
		}, want: "invalid page grid"},
		{name: "oversized grid", nup: func() *model.NUp {
			nup := nUpTestConfiguration(t, true)
			nup.Grid.Width = math.MaxFloat64
			return nup
		}, want: "invalid page grid"},
		{name: "unknown page size", nup: func() *model.NUp {
			nup := nUpTestConfiguration(t, true)
			nup.PageSize = "unknown"
			return nup
		}, want: "unknown page size"},
		{name: "invalid page dimensions", nup: func() *model.NUp {
			nup := nUpTestConfiguration(t, true)
			nup.PageDim = &types.Dim{Width: math.NaN(), Height: 100}
			return nup
		}, want: "invalid page dimensions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NUpFromImage(nil, []string{"unused"}, tt.nup())
			if err == nil || !strings.Contains(err.Error(), "n-up: prepare configuration") ||
				!strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestNUpFromImageErrorIncludesPhaseContext(t *testing.T) {
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	for _, imageFileNames := range [][]string{{missingImage}, {missingImage, missingImage}} {
		_, err := NUpFromImage(nil, imageFileNames, nUpTestConfiguration(t, true))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
		}
		if !strings.Contains(err.Error(), "n-up: impose images") {
			t.Fatalf("expected image imposition context, got %q", err.Error())
		}
	}
}

func TestNUpFromImageDefaultsConfiguration(t *testing.T) {
	nup := nUpTestConfiguration(t, true)
	ctx, err := NUpFromImage(nil, []string{filepath.Join(t.TempDir(), "missing.png")}, nup)
	if err == nil {
		t.Fatal("expected missing image error")
	}
	if ctx == nil || ctx.Configuration == nil {
		t.Fatal("expected default configuration")
	}
	if ctx.Configuration.Cmd != model.NUP {
		t.Fatalf("expected n-up command, got %v", ctx.Configuration.Cmd)
	}
}

func TestNUpImagePageDimensionDoesNotAliasPaperSize(t *testing.T) {
	nup := nUpTestConfiguration(t, true)
	_, _ = NUpFromImage(nil, []string{filepath.Join(t.TempDir(), "missing")}, nup)
	if nup.PageDim == nil {
		t.Fatal("expected resolved page dimensions")
	}
	if nup.PageDim == types.PaperSize[nup.PageSize] {
		t.Fatal("expected an independent page dimension")
	}
}

func TestNUpPDFConfigurationAllowsDerivedPageDimension(t *testing.T) {
	nup := nUpTestConfiguration(t, false)
	if nup.PageDim != nil {
		t.Fatal("expected page dimensions to be derived from the source PDF")
	}
	if err := prepareNUpConfiguration(nup, false); err != nil {
		t.Fatal(err)
	}
	if nup.PageDim != nil {
		t.Fatal("expected page dimensions to remain unset")
	}
}

func TestNUpFileErrorsIncludeFilePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		in   string
		out  string
		want string
	}{
		{name: "open input", in: filepath.Join(t.TempDir(), "missing.pdf"), out: filepath.Join(t.TempDir(), "out.pdf"), want: "n-up: open input"},
		{name: "create output", in: inFile, out: filepath.Join(t.TempDir(), "missing", "out.pdf"), want: "n-up: create output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NUpFile([]string{tt.in}, tt.out, nil, nUpTestConfiguration(t, false), nil)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestNUpFileRemovesNewOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	err := NUpFile([]string{inFile}, outFile, []string{"bogus"}, nUpTestConfiguration(t, false), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got %v", statErr)
	}
}

func TestNUpFilePreservesExistingOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := NUpFile([]string{inFile}, outFile, []string{"bogus"}, nUpTestConfiguration(t, false), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected existing output %q, got %q", want, got)
	}
}

func TestNUpFilePreservesExistingOutputOnImageFailure(t *testing.T) {
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := NUpFile([]string{missingImage}, outFile, nil, nUpTestConfiguration(t, true), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected existing output %q, got %q", want, got)
	}
}

func TestNUpFileRejectsImageOutputAliasing(t *testing.T) {
	imageFiles := []string{
		filepath.Join(t.TempDir(), "first.png"),
		filepath.Join(t.TempDir(), "later.png"),
	}
	want := []byte("image input")
	for _, fileName := range imageFiles {
		if err := os.WriteFile(fileName, want, 0600); err != nil {
			t.Fatal(err)
		}
	}

	for i, outFile := range imageFiles {
		t.Run(fmt.Sprintf("image_%d", i+1), func(t *testing.T) {
			err := NUpFile(imageFiles, outFile, nil, nUpTestConfiguration(t, true), nil)
			if !errors.Is(err, ErrNUpImageOutputConflict) {
				t.Fatalf("expected %v, got %v", ErrNUpImageOutputConflict, err)
			}
			wantContext := fmt.Sprintf("n-up image %d %q: output aliases input", i+1, outFile)
			if !strings.Contains(err.Error(), wantContext) {
				t.Fatalf("expected %q, got %q", wantContext, err)
			}
			got, readErr := os.ReadFile(outFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("image input changed: got %q, want %q", got, want)
			}
		})
	}
}

func TestNUpFileRejectsImageFilesystemAliases(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("hard link", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "hard-link.png")
		if err := os.Link(source, outFile); err != nil {
			t.Fatal(err)
		}
		err := NUpFile([]string{source}, outFile, nil, nUpTestConfiguration(t, true), nil)
		if !errors.Is(err, ErrNUpImageOutputConflict) {
			t.Fatalf("expected %v, got %v", ErrNUpImageOutputConflict, err)
		}
	})

	t.Run("symbolic link", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "symlink.png")
		if err := os.Symlink(source, outFile); err != nil {
			if errors.Is(err, os.ErrPermission) {
				t.Skipf("symbolic links unsupported: %v", err)
			}
			t.Fatal(err)
		}
		err := NUpFile([]string{source}, outFile, nil, nUpTestConfiguration(t, true), nil)
		if !errors.Is(err, ErrNUpImageOutputConflict) {
			t.Fatalf("expected %v, got %v", ErrNUpImageOutputConflict, err)
		}
	})
}

func TestNUpFileSafelyReplacesInput(t *testing.T) {
	source := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inOutFile := filepath.Join(t.TempDir(), "nup.pdf")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(inOutFile, bb, 0600); err != nil {
		t.Fatal(err)
	}

	if err = NUpFile([]string{inOutFile}, inOutFile, nil, nUpTestConfiguration(t, false), nil); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadContextFile(inOutFile); err != nil {
		t.Fatalf("expected valid replacement PDF: %v", err)
	}
}

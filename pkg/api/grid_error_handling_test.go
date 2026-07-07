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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func gridTestConfiguration(t *testing.T, images bool) *model.NUp {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)
	if images {
		nup, err = ImageGridConfig(2, 3, "", nil)
	} else {
		nup, err = PDFGridConfig(2, 3, "", nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return nup
}

func TestGridConfigurations(t *testing.T) {
	for _, tt := range []struct {
		name   string
		config func() (*model.NUp, error)
		images bool
	}{
		{name: "PDF", config: func() (*model.NUp, error) {
			return PDFGridConfig(2, 3, "", nil)
		}},
		{name: "images", config: func() (*model.NUp, error) {
			return ImageGridConfig(2, 3, "", nil)
		}, images: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nup, err := tt.config()
			if err != nil {
				t.Fatal(err)
			}
			if !nup.PageGrid {
				t.Fatal("expected page grid configuration")
			}
			if nup.ImgInputFile != tt.images {
				t.Fatalf("expected image input %t, got %t", tt.images, nup.ImgInputFile)
			}
			if nup.Grid == nil || nup.Grid.Width != 3 || nup.Grid.Height != 2 {
				t.Fatalf("expected 2x3 grid, got %v", nup.Grid)
			}
			if strings.Contains(nup.String(), "N-Up conf") || !strings.Contains(nup.String(), "Grid conf") {
				t.Fatalf("expected grid configuration label, got %q", nup.String())
			}
		})
	}
}

func TestGridConfigurationsRejectInvalidDimensions(t *testing.T) {
	for _, tt := range []struct {
		name   string
		config func() (*model.NUp, error)
	}{
		{name: "rows", config: func() (*model.NUp, error) {
			return PDFGridConfig(0, 1, "", nil)
		}},
		{name: "columns", config: func() (*model.NUp, error) {
			return ImageGridConfig(1, 0, "", nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.config(); err == nil {
				t.Fatal("expected invalid grid dimensions")
			}
		})
	}
}

func TestGridParserRejectsMissingConfiguration(t *testing.T) {
	if err := ParseGridDefinition(2, 3, nil); !errors.Is(err, ErrMissingGridConfiguration) {
		t.Fatalf("expected %v, got %v", ErrMissingGridConfiguration, err)
	}
}

func TestGridSentinelsPreserveOperationIdentity(t *testing.T) {
	for _, tt := range []struct {
		name    string
		gridErr error
		nupErr  error
	}{
		{name: "configuration", gridErr: ErrMissingGridConfiguration, nupErr: ErrMissingNUpConfiguration},
		{name: "image output conflict", gridErr: ErrGridImageOutputConflict, nupErr: ErrNUpImageOutputConflict},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := fmt.Errorf("grid: %w", tt.gridErr)
			if !errors.Is(err, tt.gridErr) {
				t.Fatalf("expected %v, got %v", tt.gridErr, err)
			}
			if errors.Is(err, tt.nupErr) {
				t.Fatalf("grid error must not match n-up sentinel %v", tt.nupErr)
			}
		})
	}
}

func TestGridEntryPointsRejectMissingConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{name: "images", run: func() error {
			_, err := GridFromImage(nil, []string{"unused"}, nil)
			return err
		}},
		{name: "stream", run: func() error {
			return Grid(bytes.NewReader(nil), io.Discard, nil, nil, nil, nil)
		}},
		{name: "file", run: func() error {
			return GridFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nil, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrMissingGridConfiguration) {
				t.Fatalf("expected %v, got %v", ErrMissingGridConfiguration, err)
			}
		})
	}
}

func TestGridEntryPointsRejectMissingInput(t *testing.T) {
	for _, tt := range []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{name: "image context", run: func() error {
			_, err := GridFromImage(nil, nil, gridTestConfiguration(t, true))
			return err
		}, wantErr: ErrMissingImageInput},
		{name: "image stream", run: func() error {
			return Grid(nil, io.Discard, nil, nil, gridTestConfiguration(t, true), nil)
		}, wantErr: ErrMissingImageInput},
		{name: "PDF stream", run: func() error {
			return Grid(nil, io.Discard, nil, nil, gridTestConfiguration(t, false), nil)
		}, wantErr: ErrMissingPDFReadSeeker},
		{name: "image file", run: func() error {
			return GridFile(nil, filepath.Join(t.TempDir(), "out.pdf"), nil, gridTestConfiguration(t, true), nil)
		}, wantErr: ErrMissingImageInput},
		{name: "PDF file", run: func() error {
			return GridFile(nil, filepath.Join(t.TempDir(), "out.pdf"), nil, gridTestConfiguration(t, false), nil)
		}, wantErr: ErrMissingPDFInput},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGridEntryPointsRejectMissingOutput(t *testing.T) {
	if err := Grid(bytes.NewReader(nil), nil, nil, nil, gridTestConfiguration(t, false), nil); !errors.Is(err, ErrMissingPDFWriter) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFWriter, err)
	}
	if err := GridFile([]string{"unused"}, "", nil, gridTestConfiguration(t, false), nil); !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("expected %v, got %v", ErrMissingPDFOutput, err)
	}
}

func TestGridEntryPointsSetCommandMode(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	err := Grid(bytes.NewReader(nil), io.Discard, nil, nil, gridTestConfiguration(t, false), conf)
	if err == nil {
		t.Fatal("expected invalid PDF error")
	}
	if conf.Cmd != model.GRID {
		t.Fatalf("expected command %d, got %d", model.GRID, conf.Cmd)
	}

	conf = model.NewDefaultConfiguration()
	_, err = GridFromImage(conf, []string{filepath.Join(t.TempDir(), "missing.png")}, gridTestConfiguration(t, true))
	if err == nil {
		t.Fatal("expected missing image error")
	}
	if conf.Cmd != model.GRID {
		t.Fatalf("expected command %d, got %d", model.GRID, conf.Cmd)
	}
}

func TestGridEntryPointsRejectNUpConfiguration(t *testing.T) {
	nup := nUpTestConfiguration(t, false)
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{name: "images", run: func() error {
			_, err := GridFromImage(nil, []string{"unused"}, nup)
			return err
		}},
		{name: "stream", run: func() error {
			return Grid(bytes.NewReader(nil), io.Discard, nil, nil, nup, nil)
		}},
		{name: "file", run: func() error {
			return GridFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nup, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), "grid: prepare configuration") ||
				!strings.Contains(err.Error(), "page grid disabled") {
				t.Fatalf("expected grid configuration error, got %v", err)
			}
		})
	}
}

func TestGridReadErrorIncludesPreparationContext(t *testing.T) {
	err := Grid(bytes.NewReader(nil), io.Discard, nil, nil, gridTestConfiguration(t, false), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "grid: prepare PDF context: read context") {
		t.Fatalf("expected preparation context, got %q", err.Error())
	}
}

func TestGridOperationErrorsIncludePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	t.Run("page selection", func(t *testing.T) {
		err := Grid(openAPITestPDF(t, inFile), io.Discard, nil, []string{"bogus"}, gridTestConfiguration(t, false), nil)
		if err == nil || !strings.Contains(err.Error(), "grid: parse page selection") {
			t.Fatalf("expected page selection context, got %v", err)
		}
	})

	t.Run("write", func(t *testing.T) {
		wantErr := errors.New("write failed")
		err := Grid(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil, nil, gridTestConfiguration(t, false), nil)
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if !strings.Contains(err.Error(), "grid: write output") {
			t.Fatalf("expected write context, got %q", err.Error())
		}
	})

	t.Run("images", func(t *testing.T) {
		missingImage := filepath.Join(t.TempDir(), "missing.png")
		err := Grid(nil, io.Discard, []string{missingImage}, nil, gridTestConfiguration(t, true), nil)
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
		}
		if !strings.Contains(err.Error(), "grid: impose images") {
			t.Fatalf("expected image context, got %q", err.Error())
		}
	})
}

func TestGridFileErrorsIncludeFilePhaseContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	err := GridFile([]string{missingInput}, filepath.Join(t.TempDir(), "out.pdf"), nil, gridTestConfiguration(t, false), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "grid: open input") {
		t.Fatalf("expected open input context, got %q", err.Error())
	}
}

func TestGridFileRejectsImageOutputAliasing(t *testing.T) {
	imageFile := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(imageFile, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}

	err := GridFile([]string{imageFile}, imageFile, nil, gridTestConfiguration(t, true), nil)
	if !errors.Is(err, ErrGridImageOutputConflict) {
		t.Fatalf("expected %v, got %v", ErrGridImageOutputConflict, err)
	}
	wantContext := fmt.Sprintf("grid image 1 %q: output aliases input", imageFile)
	if !strings.Contains(err.Error(), wantContext) {
		t.Fatalf("expected %q, got %q", wantContext, err)
	}
}

func TestGridEntryPointsRejectInvalidGrid(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(*model.NUp) error
	}{
		{name: "images", run: func(nup *model.NUp) error {
			_, err := GridFromImage(nil, []string{"unused"}, nup)
			return err
		}},
		{name: "stream", run: func(nup *model.NUp) error {
			return Grid(bytes.NewReader(nil), io.Discard, nil, nil, nup, nil)
		}},
		{name: "file", run: func(nup *model.NUp) error {
			return GridFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nup, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nup := gridTestConfiguration(t, false)
			nup.Grid = nil
			err := tt.run(nup)
			if err == nil || !strings.Contains(err.Error(), "grid: prepare configuration") ||
				!strings.Contains(err.Error(), "invalid configuration: missing page grid") {
				t.Fatalf("expected missing grid error, got %v", err)
			}
		})
	}
}

func TestGridFileCreateOutputErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "missing", "out.pdf")
	err := GridFile([]string{inFile}, outFile, nil, gridTestConfiguration(t, false), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "grid: create output") {
		t.Fatalf("expected create output context, got %q", err.Error())
	}
}

func TestGridFileMalformedInputPreservesCauseAndCleansOutput(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "broken.pdf")
	if err := os.WriteFile(inFile, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	err := GridFile([]string{inFile}, outFile, nil, gridTestConfiguration(t, false), nil)
	if !errors.Is(err, pdfcpu.ErrCorruptHeader) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrCorruptHeader, err)
	}
	if !strings.Contains(err.Error(), "grid: prepare PDF context: read context") {
		t.Fatalf("expected preparation context, got %q", err.Error())
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got %v", statErr)
	}
}

func TestGridFileRemovesNewOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	err := GridFile([]string{inFile}, outFile, []string{"bogus"}, gridTestConfiguration(t, false), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got %v", statErr)
	}
}

func TestGridFilePreservesExistingOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := GridFile([]string{inFile}, outFile, []string{"bogus"}, gridTestConfiguration(t, false), nil)
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

func TestGridFilePreservesExistingOutputOnImageFailure(t *testing.T) {
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := GridFile([]string{missingImage}, outFile, nil, gridTestConfiguration(t, true), nil)
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

func TestGridFileRejectsImageFilesystemAliases(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("hard link", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "hard-link.png")
		if err := os.Link(source, outFile); err != nil {
			t.Fatal(err)
		}
		err := GridFile([]string{source}, outFile, nil, gridTestConfiguration(t, true), nil)
		if !errors.Is(err, ErrGridImageOutputConflict) {
			t.Fatalf("expected %v, got %v", ErrGridImageOutputConflict, err)
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
		err := GridFile([]string{source}, outFile, nil, gridTestConfiguration(t, true), nil)
		if !errors.Is(err, ErrGridImageOutputConflict) {
			t.Fatalf("expected %v, got %v", ErrGridImageOutputConflict, err)
		}
	})
}

func TestGridFileSafelyReplacesInput(t *testing.T) {
	source := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inOutFile := filepath.Join(t.TempDir(), "grid.pdf")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(inOutFile, bb, 0600); err != nil {
		t.Fatal(err)
	}

	if err = GridFile([]string{inOutFile}, inOutFile, nil, gridTestConfiguration(t, false), nil); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadContextFile(inOutFile); err != nil {
		t.Fatalf("expected valid replacement PDF: %v", err)
	}
}

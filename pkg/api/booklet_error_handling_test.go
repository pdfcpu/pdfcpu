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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func bookletTestConfiguration(t *testing.T, images bool) *model.NUp {
	t.Helper()

	var (
		nup *model.NUp
		err error
	)
	if images {
		nup, err = ImageBookletConfig(2, "", nil)
	} else {
		nup, err = PDFBookletConfig(2, "", nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return nup
}

func containsAny(s string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(s, value) {
			return true
		}
	}
	return false
}

func TestNUpValuesForBookletsReturnsCopy(t *testing.T) {
	values := NUpValuesForBooklets()
	if len(values) == 0 {
		t.Fatal("expected booklet values")
	}
	values[0] = 99
	if got := NUpValuesForBooklets()[0]; got != 2 {
		t.Fatalf("expected protected booklet values, got %d", got)
	}
}

func TestBookletReadErrorIncludesPhaseContext(t *testing.T) {
	err := Booklet(bytes.NewReader(nil), io.Discard, nil, nil, bookletTestConfiguration(t, false), nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "booklet: read and validate: read context") {
		t.Fatalf("expected read phase context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "prepare PDF context") {
		t.Fatalf("unexpected optimized preparation context: %q", err.Error())
	}
}

func TestBookletMalformedInputPreservesCause(t *testing.T) {
	err := Booklet(bytes.NewReader([]byte("not a PDF")), io.Discard, nil, nil, bookletTestConfiguration(t, false), nil)
	if !errors.Is(err, pdfcpu.ErrCorruptHeader) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrCorruptHeader, err)
	}
	if !strings.Contains(err.Error(), "booklet: read and validate: read context") {
		t.Fatalf("expected read phase context, got %q", err.Error())
	}
}

func TestBookletClosedInputPreservesCause(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	f, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	err = Booklet(f, io.Discard, nil, nil, bookletTestConfiguration(t, false), nil)
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected %v, got %v", os.ErrClosed, err)
	}
	if !strings.Contains(err.Error(), "booklet: read and validate: read context") {
		t.Fatalf("expected read phase context, got %q", err.Error())
	}
}

func TestBookletPageSelectionErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	err := Booklet(openAPITestPDF(t, inFile), io.Discard, nil, []string{"bogus"}, bookletTestConfiguration(t, false), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "booklet: parse page selection") {
		t.Fatalf("expected page selection context, got %q", err.Error())
	}
}

func TestBookletWriteErrorIncludesPhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	wantErr := errors.New("write failed")
	err := Booklet(openAPITestPDF(t, inFile), failingWriter{err: wantErr}, nil, nil, bookletTestConfiguration(t, false), nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "booklet: write output") {
		t.Fatalf("expected write context, got %q", err.Error())
	}
}

func TestBookletImageErrorIncludesPhaseContext(t *testing.T) {
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	err := Booklet(bytes.NewReader(nil), io.Discard, []string{missingImage}, nil, bookletTestConfiguration(t, true), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "booklet: impose images") {
		t.Fatalf("expected image imposition context, got %q", err.Error())
	}
}

func TestBookletFromImagesAcceptsNilPDFReader(t *testing.T) {
	imageFile := filepath.Join("..", "samples", "images", "any.jpg")
	var out bytes.Buffer
	err := Booklet(nil, &out, []string{imageFile}, nil, bookletTestConfiguration(t, true), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out.Bytes(), []byte("%PDF-")) {
		t.Fatal("expected PDF output")
	}
}

func TestBookletFromImagesDefaultsConfiguration(t *testing.T) {
	imageFile := filepath.Join("..", "samples", "images", "any.jpg")
	nup := bookletTestConfiguration(t, true)
	ctx, err := BookletFromImages(nil, []string{imageFile}, nup)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Configuration == nil {
		t.Fatal("expected default configuration")
	}
	if ctx.Configuration.Cmd != model.BOOKLET {
		t.Fatalf("expected booklet command, got %v", ctx.Configuration.Cmd)
	}
	if nup.PageDim == types.PaperSize[nup.PageSize] {
		t.Fatal("expected an independent page dimension")
	}
}

func TestBookletFromImagesRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		nup  func() *model.NUp
		want string
	}{
		{name: "missing grid", nup: DefaultBookletConfig, want: "missing page grid"},
		{name: "unknown page size", nup: func() *model.NUp {
			nup := bookletTestConfiguration(t, true)
			nup.PageDim = nil
			nup.PageSize = "unknown"
			return nup
		}, want: "unknown page size"},
		{name: "invalid page dimensions", nup: func() *model.NUp {
			nup := bookletTestConfiguration(t, true)
			nup.PageDim = &types.Dim{}
			return nup
		}, want: "invalid page dimensions"},
		{name: "invalid page grid", nup: func() *model.NUp {
			nup := bookletTestConfiguration(t, true)
			nup.Grid.Width = 1.5
			return nup
		}, want: "invalid page grid"},
		{name: "unsupported pages per sheet", nup: func() *model.NUp {
			nup := bookletTestConfiguration(t, true)
			nup.Grid = &types.Dim{Width: 3, Height: 1}
			return nup
		}, want: "unsupported pages per sheet"},
		{name: "unsupported booklet type", nup: func() *model.NUp {
			nup := bookletTestConfiguration(t, true)
			nup.BookletType = model.BookletType(99)
			return nup
		}, want: "unsupported booklet type"},
		{name: "invalid folio size", nup: func() *model.NUp {
			nup := bookletTestConfiguration(t, true)
			nup.MultiFolio = true
			nup.FolioSize = 0
			return nup
		}, want: "folio size must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BookletFromImages(nil, []string{"unused"}, tt.nup())
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "booklet: prepare configuration") || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err.Error())
			}
			if strings.Contains(err.Error(), "prepare configuration: booklet:") {
				t.Fatalf("unexpected repeated operation context: %q", err)
			}
		})
	}
}

func TestBookletEntryPointsRejectMissingConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{name: "images", run: func() error {
			_, err := BookletFromImages(nil, []string{"unused"}, nil)
			return err
		}},
		{name: "stream", run: func() error {
			return Booklet(nil, io.Discard, nil, nil, nil, nil)
		}},
		{name: "file", run: func() error {
			return BookletFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nil, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrMissingBookletConfiguration) {
				t.Fatalf("expected %v, got %v", ErrMissingBookletConfiguration, err)
			}
		})
	}
}

func TestBookletConfigurationErrorsIncludePhaseAndPreserveCause(t *testing.T) {
	wantErr := errors.New("configuration failed")
	if err := wrapBookletConfigurationError(wantErr); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}

	for _, tt := range []struct {
		name string
		run  func(*model.NUp) error
	}{
		{name: "images", run: func(nup *model.NUp) error {
			_, err := BookletFromImages(nil, []string{"unused"}, nup)
			return err
		}},
		{name: "stream", run: func(nup *model.NUp) error {
			return Booklet(nil, io.Discard, nil, nil, nup, nil)
		}},
		{name: "file", run: func(nup *model.NUp) error {
			return BookletFile([]string{"unused"}, filepath.Join(t.TempDir(), "out.pdf"), nil, nup, nil)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(DefaultBookletConfig())
			if err == nil || !strings.Contains(err.Error(), "booklet: prepare configuration") {
				t.Fatalf("expected configuration phase context, got %v", err)
			}
			if errors.Unwrap(err) == nil {
				t.Fatalf("expected wrapped configuration cause, got %v", err)
			}
		})
	}
}

func TestBookletFileValidatesImageConfigurationBeforeAliasCheck(t *testing.T) {
	nup := DefaultBookletConfig()
	nup.ImgInputFile = true
	invalidPath := string([]byte{'i', 'n', 'v', 'a', 'l', 'i', 'd', 0})

	err := BookletFile([]string{invalidPath}, filepath.Join(t.TempDir(), "out.pdf"), nil, nup, nil)
	if err == nil {
		t.Fatal("expected configuration error")
	}
	if !strings.Contains(err.Error(), "booklet: prepare configuration") || !strings.Contains(err.Error(), "missing page grid") {
		t.Fatalf("expected configuration phase error, got %q", err)
	}
	if strings.Contains(err.Error(), "check output alias") {
		t.Fatalf("unexpected alias-check error: %q", err)
	}
}

func TestBookletFileErrorsIncludeFilePhaseContext(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	tests := []struct {
		name string
		in   string
		out  string
		want string
	}{
		{name: "open input", in: filepath.Join(t.TempDir(), "missing.pdf"), out: filepath.Join(t.TempDir(), "out.pdf"), want: "booklet: open input"},
		{name: "create output", in: inFile, out: filepath.Join(t.TempDir(), "missing", "out.pdf"), want: "booklet: create output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BookletFile([]string{tt.in}, tt.out, nil, bookletTestConfiguration(t, false), nil)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestBookletFileRemovesNewOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	err := BookletFile([]string{inFile}, outFile, []string{"bogus"}, bookletTestConfiguration(t, false), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got %v", statErr)
	}
}

func TestBookletFilePreservesExistingOutputOnFailure(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := BookletFile([]string{inFile}, outFile, []string{"bogus"}, bookletTestConfiguration(t, false), nil)
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

func TestBookletFileMissingFirstImageUsesIndexedContext(t *testing.T) {
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	err := BookletFile([]string{missingImage}, outFile, nil, bookletTestConfiguration(t, true), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "booklet image 1") ||
		!strings.Contains(err.Error(), missingImage) ||
		!strings.Contains(err.Error(), ": open") {
		t.Fatalf("expected indexed image path context, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "booklet: open input") {
		t.Fatalf("unexpected PDF input context: %q", err.Error())
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got %v", statErr)
	}
}

func TestBookletFileMissingImageWithExistingOutputUsesImageOpenContext(t *testing.T) {
	missingImage := filepath.Join(t.TempDir(), "missing.png")
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := BookletFile([]string{missingImage}, outFile, nil, bookletTestConfiguration(t, true), nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	wantContext := fmt.Sprintf("booklet: impose images: booklet image 1 %q: open", missingImage)
	if !strings.Contains(err.Error(), wantContext) {
		t.Fatalf("expected %q, got %q", wantContext, err)
	}
	if strings.Contains(err.Error(), "check output alias") {
		t.Fatalf("unexpected alias-check context: %q", err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

func TestBookletFileRejectsImageOutputAliasing(t *testing.T) {
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
			err := BookletFile(imageFiles, outFile, nil, bookletTestConfiguration(t, true), nil)
			if !errors.Is(err, ErrBookletImageOutputConflict) {
				t.Fatalf("expected %v, got %v", ErrBookletImageOutputConflict, err)
			}
			wantContext := fmt.Sprintf("booklet image %d %q: output aliases input", i+1, outFile)
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

func TestBookletFileRejectsImageFilesystemAliases(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("hard link", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "hard-link.png")
		if err := os.Link(source, outFile); err != nil {
			t.Fatal(err)
		}
		err := BookletFile([]string{source}, outFile, nil, bookletTestConfiguration(t, true), nil)
		if !errors.Is(err, ErrBookletImageOutputConflict) {
			t.Fatalf("expected %v, got %v", ErrBookletImageOutputConflict, err)
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
		err := BookletFile([]string{source}, outFile, nil, bookletTestConfiguration(t, true), nil)
		if !errors.Is(err, ErrBookletImageOutputConflict) {
			t.Fatalf("expected %v, got %v", ErrBookletImageOutputConflict, err)
		}
	})
}

func TestBookletImageOutputAliasCheckAllowsMissingOutput(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(inFile, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "new.pdf")

	aliases, err := bookletImageOutputAliasesInput(inFile, outFile)
	if err != nil {
		t.Fatal(err)
	}
	if aliases {
		t.Fatal("missing output must not alias image input")
	}
}

func TestBookletImageOutputAliasCheckPreservesFilesystemErrors(t *testing.T) {
	validImage := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(validImage, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	existingOutput := filepath.Join(t.TempDir(), "existing.pdf")
	if err := os.WriteFile(existingOutput, []byte("output"), 0600); err != nil {
		t.Fatal(err)
	}
	invalidPath := string([]byte{'i', 'n', 'v', 'a', 'l', 'i', 'd', 0})

	for _, tt := range []struct {
		name    string
		inFile  string
		outFile string
		phases  []string
	}{
		{
			name:    "input",
			inFile:  invalidPath,
			outFile: existingOutput,
			phases:  []string{"resolve input path", "stat input"},
		},
		{
			name:    "output",
			inFile:  validImage,
			outFile: invalidPath,
			phases:  []string{"resolve output path", "stat output"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := BookletFile([]string{tt.inFile}, tt.outFile, nil, bookletTestConfiguration(t, true), nil)
			if err == nil {
				t.Fatal("expected alias-check error")
			}
			wantContext := fmt.Sprintf("booklet image 1 %q: check output alias", tt.inFile)
			if !strings.Contains(err.Error(), wantContext) || !containsAny(err.Error(), tt.phases...) {
				t.Fatalf("expected %q and one of %q, got %q", wantContext, tt.phases, err)
			}
		})
	}
}

func TestBookletImageOutputAliasCheckPreservesResolverErrors(t *testing.T) {
	wantErr := errors.New("alias resolution failed")
	identity := func(path string) (string, error) { return path, nil }
	missing := func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }

	for _, tt := range []struct {
		name  string
		abs   func(string) (string, error)
		stat  func(string) (os.FileInfo, error)
		phase string
	}{
		{name: "absolute path", abs: func(string) (string, error) { return "", wantErr }, stat: missing, phase: "resolve input path"},
		{name: "output absolute path", abs: func(path string) (string, error) {
			if path == "out.pdf" {
				return "", wantErr
			}
			return path, nil
		}, stat: missing, phase: "resolve output path"},
		{name: "stat", abs: identity, stat: func(string) (os.FileInfo, error) { return nil, wantErr }, phase: "stat output"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := bookletImageOutputAliasesInputWith("in.png", "out.pdf", tt.abs, tt.stat)
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected %v, got %v", wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.phase) {
				t.Fatalf("expected %q, got %q", tt.phase, err)
			}
		})
	}
}

func TestBookletFileSafelyReplacesInput(t *testing.T) {
	source := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inOutFile := filepath.Join(t.TempDir(), "booklet.pdf")
	bb, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(inOutFile, bb, 0600); err != nil {
		t.Fatal(err)
	}

	if err = BookletFile([]string{inOutFile}, inOutFile, nil, bookletTestConfiguration(t, false), nil); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadContextFile(inOutFile); err != nil {
		t.Fatalf("expected valid replacement PDF: %v", err)
	}
}

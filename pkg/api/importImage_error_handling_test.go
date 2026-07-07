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
	"image"
	"image/png"
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

type importImageFailingReader struct {
	err error
}

func (r importImageFailingReader) Read([]byte) (int, error) {
	return 0, r.err
}

type importImageErrorCloser struct {
	err error
}

func (c importImageErrorCloser) Close() error {
	return c.err
}

func importImageTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeImportImageTestPNG(t *testing.T, fileName string) {
	t.Helper()
	if err := os.WriteFile(fileName, importImageTestPNG(t), 0600); err != nil {
		t.Fatal(err)
	}
}

func importImagesError(imgs []io.Reader, imp *pdfcpu.Import) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("panic: %v", v)
		}
	}()
	return ImportImages(nil, io.Discard, imgs, imp, nil)
}

// TestImportImagesArgumentValidation verifies public import-images boundary guards.
func TestImportImagesArgumentValidation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "missing writer",
			err:  ImportImages(nil, nil, []io.Reader{bytes.NewReader(importImageTestPNG(t))}, nil, nil),
			want: ErrMissingPDFWriter,
		},
		{
			name: "missing images",
			err:  ImportImages(nil, io.Discard, nil, nil, nil),
			want: ErrMissingImageInput,
		},
		{
			name: "missing image reader",
			err:  ImportImages(nil, io.Discard, []io.Reader{nil}, nil, nil),
			want: ErrMissingImageReader,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestImportImagesFileArgumentValidation verifies file boundary guards run before filesystem access.
func TestImportImagesFileArgumentValidation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "missing images",
			err:  ImportImagesFile(nil, filepath.Join(t.TempDir(), "out.pdf"), nil, nil),
			want: ErrMissingImageInput,
		},
		{
			name: "empty image",
			err:  ImportImagesFile([]string{""}, filepath.Join(t.TempDir(), "out.pdf"), nil, nil),
			want: ErrMissingImageInput,
		},
		{
			name: "missing output",
			err:  ImportImagesFile([]string{"image.png"}, "", nil, nil),
			want: ErrMissingPDFOutput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestImportImagesFileValidatesConfigurationBeforeFilesystemAccess verifies preflight protects output state.
func TestImportImagesFileValidatesConfigurationBeforeFilesystemAccess(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	imp := DefaultImportConfig()
	imp.PageDim = nil

	err := ImportImagesFile([]string{"missing.png"}, outFile, imp, nil)
	if !errors.Is(err, ErrInvalidImportConfiguration) {
		t.Fatalf("expected %v, got %v", ErrInvalidImportConfiguration, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("configuration validation reached filesystem access: %v", err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("configuration validation created output: %v", statErr)
	}
}

// TestPrepareImportConfigurationAppliesDefaultsAndValidates verifies the I/O-free API preflight.
func TestPrepareImportConfigurationAppliesDefaultsAndValidates(t *testing.T) {
	imp, err := PrepareImportConfiguration(nil)
	if err != nil {
		t.Fatal(err)
	}
	if imp == nil || imp.PageDim == nil {
		t.Fatal("expected default import configuration")
	}

	invalid := DefaultImportConfig()
	invalid.PageDim = nil
	if _, err := PrepareImportConfiguration(invalid); !errors.Is(err, ErrInvalidImportConfiguration) {
		t.Fatalf("expected %v, got %v", ErrInvalidImportConfiguration, err)
	}
}

// TestValidateImportImagesOutputRejectsAliases verifies direct and filesystem aliases share a sentinel.
func TestValidateImportImagesOutputRejectsAliases(t *testing.T) {
	tests := []struct {
		name  string
		alias func(string, string) error
	}{
		{name: "same path"},
		{name: "hard link", alias: os.Link},
		{name: "symlink", alias: os.Symlink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			imageFile := filepath.Join(dir, "image.png")
			writeImportImageTestPNG(t, imageFile)
			outFile := imageFile
			if tt.alias != nil {
				outFile = filepath.Join(dir, "out.pdf")
				if err := tt.alias(imageFile, outFile); err != nil {
					t.Fatal(err)
				}
			}

			err := ValidateImportImagesOutput([]string{"-", imageFile}, outFile)
			if !errors.Is(err, ErrImportImagesOutputConflict) {
				t.Fatalf("expected %v, got %v", ErrImportImagesOutputConflict, err)
			}
			want := fmt.Sprintf("import images: image 2 %q: output aliases input", imageFile)
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err)
			}
		})
	}
}

// TestValidateImportImagesOutputSkipsStdinMarker verifies CLI preflight does not stat "-".
func TestValidateImportImagesOutputSkipsStdinMarker(t *testing.T) {
	if err := ValidateImportImagesOutput([]string{"-"}, "-"); err != nil {
		t.Fatalf("stdin marker triggered filesystem alias validation: %v", err)
	}
}

// TestValidateImportImagesOutputReportsLookupErrors verifies alias inspection preserves its cause and context.
func TestValidateImportImagesOutputReportsLookupErrors(t *testing.T) {
	dir := t.TempDir()
	imageFile := filepath.Join(dir, "image.png")
	writeImportImageTestPNG(t, imageFile)
	outFile := filepath.Join(dir, "out.pdf")
	if err := os.Symlink(filepath.Base(outFile), outFile); err != nil {
		t.Fatal(err)
	}

	err := ValidateImportImagesOutput([]string{imageFile}, outFile)
	if err == nil {
		t.Fatal("expected error")
	}
	want := fmt.Sprintf("import images: image 1 %q: check output alias", imageFile)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestImportImagesFileRejectsOutputAliasBeforeSourceIO verifies preflight precedes opening any image.
func TestImportImagesFileRejectsOutputAliasBeforeSourceIO(t *testing.T) {
	dir := t.TempDir()
	missingImage := filepath.Join(dir, "missing.png")
	imageFile := filepath.Join(dir, "image.png")
	want := importImageTestPNG(t)
	if err := os.WriteFile(imageFile, want, 0600); err != nil {
		t.Fatal(err)
	}

	err := ImportImagesFile([]string{missingImage, imageFile}, imageFile, nil, nil)
	if !errors.Is(err, ErrImportImagesOutputConflict) {
		t.Fatalf("expected %v, got %v", ErrImportImagesOutputConflict, err)
	}
	wantContext := fmt.Sprintf("image 2 %q: output aliases input", imageFile)
	if !strings.Contains(err.Error(), wantContext) {
		t.Fatalf("expected %q, got %q", wantContext, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("alias preflight opened an earlier missing image: %v", err)
	}
	got, err := os.ReadFile(imageFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("aliased image input changed")
	}
}

// TestImportImagesFileRejectsOutputAliasesPreservingInput verifies all aliases leave the source intact.
func TestImportImagesFileRejectsOutputAliasesPreservingInput(t *testing.T) {
	tests := []struct {
		name  string
		alias func(string, string) error
	}{
		{name: "identical path"},
		{name: "hard link", alias: os.Link},
		{name: "symlink", alias: os.Symlink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			imageFile := filepath.Join(dir, "scan.jpg")
			want := []byte("original image")
			if err := os.WriteFile(imageFile, want, 0600); err != nil {
				t.Fatal(err)
			}
			outFile := imageFile
			if tt.alias != nil {
				outFile = filepath.Join(dir, "out.pdf")
				if err := tt.alias(imageFile, outFile); err != nil {
					t.Fatal(err)
				}
			}

			err := ImportImagesFile([]string{imageFile}, outFile, nil, nil)
			if !errors.Is(err, ErrImportImagesOutputConflict) {
				t.Fatalf("expected %v, got %v", ErrImportImagesOutputConflict, err)
			}
			wantContext := fmt.Sprintf("image 1 %q: output aliases input", imageFile)
			if !strings.Contains(err.Error(), wantContext) {
				t.Fatalf("expected %q, got %q", wantContext, err)
			}
			got, readErr := os.ReadFile(imageFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("image changed: got %q, want %q", got, want)
			}
		})
	}
}

// TestImportImagesFileTreatsDashAsLiteralFile verifies file API alias checks never apply stdin semantics.
func TestImportImagesFileTreatsDashAsLiteralFile(t *testing.T) {
	tests := []struct {
		name    string
		outFile string
		alias   func() error
	}{
		{name: "identical output path", outFile: "-"},
		{name: "hard-link output", outFile: "out.pdf", alias: func() error { return os.Link("-", "out.pdf") }},
		{name: "symlink output", outFile: "out.pdf", alias: func() error { return os.Symlink("-", "out.pdf") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			want := []byte("literal dash input")
			if err := os.WriteFile("-", want, 0600); err != nil {
				t.Fatal(err)
			}
			if tt.alias != nil {
				if err := tt.alias(); err != nil {
					t.Fatal(err)
				}
			}
			before := importImageTestDirectoryEntries(t, ".")

			err := ImportImagesFile([]string{"-"}, tt.outFile, nil, nil)
			if !errors.Is(err, ErrImportImagesOutputConflict) {
				t.Fatalf("expected %v, got %v", ErrImportImagesOutputConflict, err)
			}
			if !strings.Contains(err.Error(), `image 1 "-": output aliases input`) {
				t.Fatalf("expected literal dash context, got %q", err)
			}
			got, readErr := os.ReadFile("-")
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("input changed: got %q, want %q", got, want)
			}
			aliases, aliasErr := outputAliasesInput("-", tt.outFile)
			if aliasErr != nil {
				t.Fatal(aliasErr)
			}
			if !aliases {
				t.Fatal("pre-existing output alias was replaced")
			}
			after := importImageTestDirectoryEntries(t, ".")
			if before != after {
				t.Fatalf("preflight created artifacts: before %q, after %q", before, after)
			}
		})
	}
}

func importImageTestDirectoryEntries(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return strings.Join(names, "\n")
}

// TestImportImagesFilePreflightDoesNotCreateOutput verifies rejection precedes output creation.
func TestImportImagesFilePreflightDoesNotCreateOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "missing.png")
	err := ImportImagesFile([]string{outFile}, outFile, nil, nil)
	if !errors.Is(err, ErrImportImagesOutputConflict) {
		t.Fatalf("expected %v, got %v", ErrImportImagesOutputConflict, err)
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("preflight created output: %v", statErr)
	}
}

// TestImportReportsConfigurationContext verifies the API owns import parsing context.
func TestImportReportsConfigurationContext(t *testing.T) {
	_, err := Import("offset:x 1", types.POINTS)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError, got %v", err)
	}
	if !strings.Contains(err.Error(), "import images: parse configuration") {
		t.Fatalf("expected import parsing context, got %q", err)
	}
}

// TestImportImagesReadErrorContext verifies an existing PDF failure stops at the validated-context boundary.
func TestImportImagesReadErrorContext(t *testing.T) {
	err := ImportImages(
		bytes.NewReader(nil),
		io.Discard,
		[]io.Reader{bytes.NewReader(importImageTestPNG(t))},
		nil,
		nil,
	)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "import images: prepare PDF context") {
		t.Fatalf("expected context preparation phase, got %q", err)
	}
}

// TestImportImagesReportsImageIndex verifies image failures identify their slice position.
func TestImportImagesReportsImageIndex(t *testing.T) {
	wantErr := errors.New("image read failure")
	err := ImportImages(
		nil,
		io.Discard,
		[]io.Reader{
			bytes.NewReader(importImageTestPNG(t)),
			importImageFailingReader{err: wantErr},
		},
		nil,
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "import images: image 2: create pages") {
		t.Fatalf("expected image index context, got %q", err)
	}
}

// TestImportImagesWriteErrorContext verifies output errors preserve their cause.
func TestImportImagesWriteErrorContext(t *testing.T) {
	wantErr := errors.New("write failure")
	err := ImportImages(
		nil,
		failingWriter{err: wantErr},
		[]io.Reader{bytes.NewReader(importImageTestPNG(t))},
		nil,
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "import images: write output") {
		t.Fatalf("expected output writing context, got %q", err)
	}
}

// TestImportImagesRejectsInvalidConfiguration verifies malformed caller configurations do not panic.
func TestImportImagesRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*pdfcpu.Import)
	}{
		{name: "missing dimensions", mutate: func(imp *pdfcpu.Import) { imp.PageDim = nil }},
		{name: "invalid width", mutate: func(imp *pdfcpu.Import) { imp.PageDim.Width = math.NaN() }},
		{name: "invalid height", mutate: func(imp *pdfcpu.Import) { imp.PageDim.Height = math.Inf(1) }},
		{name: "zero scale", mutate: func(imp *pdfcpu.Import) { imp.Scale = 0 }},
		{name: "relative scale", mutate: func(imp *pdfcpu.Import) { imp.Scale = 2 }},
		{name: "negative DPI", mutate: func(imp *pdfcpu.Import) { imp.DPI = -1 }},
		{name: "invalid offset", mutate: func(imp *pdfcpu.Import) { imp.Dx = math.NaN() }},
		{name: "invalid position", mutate: func(imp *pdfcpu.Import) { imp.Pos = types.Anchor(-1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := DefaultImportConfig()
			pageDim := *imp.PageDim
			imp.PageDim = &pageDim
			tt.mutate(imp)

			err := importImagesError(
				[]io.Reader{bytes.NewReader(importImageTestPNG(t))},
				imp,
			)
			if strings.Contains(fmt.Sprint(err), "panic:") {
				t.Fatalf("unexpected panic: %v", err)
			}
			if !errors.Is(err, ErrInvalidImportConfiguration) {
				t.Fatalf("expected %v, got %v", ErrInvalidImportConfiguration, err)
			}
			if !strings.Contains(err.Error(), "import images: validate configuration") {
				t.Fatalf("expected configuration validation context, got %q", err)
			}
		})
	}
}

// TestImportImagesFileMissingOutputDoesNotPanic verifies creating a new output is panic-safe.
func TestImportImagesFileMissingOutputDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	imageFile := filepath.Join(dir, "image.png")
	outFile := filepath.Join(dir, "out.pdf")
	writeImportImageTestPNG(t, imageFile)

	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ImportImagesFile panicked: %v", v)
		}
	}()
	if err := ImportImagesFile([]string{imageFile}, outFile, nil, nil); err != nil {
		t.Fatal(err)
	}
}

// TestImportImagesFileOpenImageErrorContext verifies source paths and causes are preserved.
func TestImportImagesFileOpenImageErrorContext(t *testing.T) {
	dir := t.TempDir()
	imageFile := filepath.Join(dir, "missing.png")
	outFile := filepath.Join(dir, "out.pdf")

	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ImportImagesFile panicked: %v", v)
		}
	}()
	err := ImportImagesFile([]string{imageFile}, outFile, nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	want := fmt.Sprintf("import images: image 1 %q: open", imageFile)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

// TestImportImagesFileCreateOutputErrorContext verifies output creation failures preserve their cause.
func TestImportImagesFileCreateOutputErrorContext(t *testing.T) {
	dir := t.TempDir()
	imageFile := filepath.Join(dir, "image.png")
	outFile := filepath.Join(dir, "missing", "out.pdf")
	writeImportImageTestPNG(t, imageFile)

	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ImportImagesFile panicked: %v", v)
		}
	}()
	err := ImportImagesFile([]string{imageFile}, outFile, nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	if !strings.Contains(err.Error(), "import images: create output") {
		t.Fatalf("expected output creation context, got %q", err)
	}
}

// TestImportImagesFilePreflightOutputErrorContext verifies output lookup errors precede inspection.
func TestImportImagesFilePreflightOutputErrorContext(t *testing.T) {
	dir := t.TempDir()
	imageFile := filepath.Join(dir, "image.png")
	outFile := filepath.Join(dir, "out.pdf")
	writeImportImageTestPNG(t, imageFile)
	if err := os.Symlink(filepath.Base(outFile), outFile); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ImportImagesFile panicked: %v", v)
		}
	}()
	err := ImportImagesFile([]string{imageFile}, outFile, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "check output alias") {
		t.Fatalf("expected output alias preflight context, got %q", err)
	}
}

// TestCloseImportImageFilesPreservesAllErrors verifies image cleanup does not stop at the first failure.
func TestCloseImportImageFilesPreservesAllErrors(t *testing.T) {
	err1 := errors.New("close image 1")
	err2 := errors.New("close image 2")
	err := closeImportImageInputs([]importImageFileCloser{
		{closer: importImageErrorCloser{err: err1}, imageIndex: 1, fileName: "one.png"},
		{closer: importImageErrorCloser{err: err2}, imageIndex: 3, fileName: "three.jpg"},
	})
	for _, want := range []error{err1, err2} {
		if !errors.Is(err, want) {
			t.Fatalf("expected %v, got %v", want, err)
		}
	}
	for _, want := range []string{
		`import images: image 1 "one.png": close`,
		`import images: image 3 "three.jpg": close`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

// TestFinishImportImagesFileReportsCleanupErrors verifies failed operations report cleanup phases.
// TestFinishImportImagesFilePreservesPrimaryAndCleanupErrors verifies cleanup augments the operation failure.
// TestFinishImportImagesFileRenameErrorContext verifies replacement failures retain filesystem causes.
// TestImportImagesFileAppendsWithoutReplacingOnFailure verifies an existing PDF remains intact after an image error.
func TestImportImagesFileAppendsWithoutReplacingOnFailure(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.pdf")
	f, err := os.Create(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := ImportImages(
		nil,
		f,
		[]io.Reader{bytes.NewReader(importImageTestPNG(t))},
		nil,
		nil,
	); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	missingImage := filepath.Join(dir, "missing.png")
	err = ImportImagesFile([]string{missingImage}, outFile, nil, nil)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	after, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("existing output changed after failed append")
	}
}

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
	"image"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func imageTestPDF() string {
	return filepath.Join("..", "samples", "images", "test.pdf")
}

func imageTestImage() string {
	return filepath.Join("..", "samples", "images", "test_1_Im1.png")
}

// TestImageArgumentValidation verifies public image boundary guards.
func TestImageArgumentValidation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "images reader", err: func() error {
			_, err := Images(nil, nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "list reader", err: func() error {
			_, err := ListImages(nil, nil, nil)
			return err
		}(), want: ErrMissingPDFReadSeeker},
		{name: "update reader", err: UpdateImages(nil, bytes.NewReader(nil), io.Discard, 1, 0, "", nil), want: ErrMissingPDFReadSeeker},
		{name: "update image", err: UpdateImages(bytes.NewReader(nil), nil, io.Discard, 1, 0, "", nil), want: ErrMissingImageInput},
		{name: "update writer", err: UpdateImages(bytes.NewReader(nil), bytes.NewReader(nil), nil, 1, 0, "", nil), want: ErrMissingPDFWriter},
		{name: "file input", err: UpdateImagesFile("", "image.png", "", 1, 0, "", nil), want: ErrMissingPDFInput},
		{name: "file image", err: UpdateImagesFile("input.pdf", "", "", 1, 0, "", nil), want: ErrMissingImageInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, tt.err)
			}
		})
	}
}

// TestValidateUpdateImagesOutputRejectsMissingImage verifies alias validation requires an image path.
func TestValidateUpdateImagesOutputRejectsMissingImage(t *testing.T) {
	if err := ValidateUpdateImagesOutput("", ""); !errors.Is(err, ErrMissingImageInput) {
		t.Fatalf("expected %v, got %v", ErrMissingImageInput, err)
	}
}

// TestImageSelectionValidation verifies invalid selectors fail before PDF reading.
func TestImageSelectionValidation(t *testing.T) {
	tests := []struct {
		name   string
		objNr  int
		pageNr int
		id     string
		want   string
	}{
		{name: "negative object", objNr: -1, pageNr: 1, id: "Im0", want: "negative object number"},
		{name: "object and page", objNr: 1, pageNr: 1, want: "object number conflicts with page resource"},
		{name: "object and resource id", objNr: 1, id: "Im0", want: "object number conflicts with page resource"},
		{name: "missing page", want: "missing page number"},
		{name: "negative page", pageNr: -1, id: "Im0", want: "missing page number"},
		{name: "missing resource id", pageNr: 1, want: "missing resource id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateImages(bytes.NewReader(nil), bytes.NewReader(nil), io.Discard, tt.objNr, tt.pageNr, tt.id, nil)
			if !errors.Is(err, ErrInvalidImageSelection) {
				t.Fatalf("expected %v, got %v", ErrInvalidImageSelection, err)
			}
			if errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("selection validation must precede PDF reading: %v", err)
			}
			for _, want := range []string{"update images: validate selection", tt.want} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q context, got %q", want, err)
				}
			}
		})
	}
}

// TestImageSelectionFromFile verifies filename parsing and parse-cause preservation.
func TestImageSelectionFromFile(t *testing.T) {
	pageNr, id, err := imageSelectionFromFile(filepath.Join("dir_with_underscores", "mountain_12_Im0.png"))
	if err != nil {
		t.Fatal(err)
	}
	if pageNr != 12 || id != "Im0" {
		t.Fatalf("expected page 12 and id Im0, got page %d and id %q", pageNr, id)
	}

	_, _, err = imageSelectionFromFile("image.png")
	if !errors.Is(err, ErrInvalidImageSelection) {
		t.Fatalf("expected %v, got %v", ErrInvalidImageSelection, err)
	}

	_, _, err = imageSelectionFromFile("mountain_x_Im0.png")
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected strconv.NumError, got %v", err)
	}
	if !strings.Contains(err.Error(), `page number "x"`) {
		t.Fatalf("expected page token context, got %q", err)
	}
}

// TestUpdateImagesFileRejectsPartialSelection verifies explicit selector data is not overwritten from a filename.
func TestUpdateImagesFileRejectsPartialSelection(t *testing.T) {
	err := UpdateImagesFile("missing.pdf", "mountain_1_Im0.png", "", 0, 1, "", nil)
	if !errors.Is(err, ErrInvalidImageSelection) {
		t.Fatalf("expected %v, got %v", ErrInvalidImageSelection, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("selection validation must precede opening input: %v", err)
	}
}

// TestImagesReadErrorsIncludePhaseContext verifies list preparation errors preserve their cause.
func TestImagesReadErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "raw", run: func() error {
			_, err := Images(bytes.NewReader(nil), nil, nil)
			return err
		}},
		{name: "formatted", run: func() error {
			_, err := ListImages(bytes.NewReader(nil), nil, nil)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, pdfcpu.ErrEmptyInput) {
				t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
			}
			if !strings.Contains(err.Error(), "list images: prepare PDF context") {
				t.Fatalf("expected list preparation context, got %q", err)
			}
		})
	}
}

// TestImagesPageSelectionErrorsIncludePhaseContext verifies page-selection context for both list entry points.
func TestImagesPageSelectionErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "raw", run: func() error {
			_, err := Images(openAPITestPDF(t, imageTestPDF()), []string{"foo"}, nil)
			return err
		}},
		{name: "formatted", run: func() error {
			_, err := ListImages(openAPITestPDF(t, imageTestPDF()), []string{"foo"}, nil)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), "list images: parse page selection") {
				t.Fatalf("expected page-selection context, got %v", err)
			}
		})
	}
}

// TestListImagesReturnsFormattedOutput verifies the API adapter preserves CLI-oriented formatting.
func TestListImagesReturnsFormattedOutput(t *testing.T) {
	ss, err := ListImages(openAPITestPDF(t, imageTestPDF()), []string{"1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ss) == 0 || !strings.Contains(ss[0], "images available") {
		t.Fatalf("expected formatted image summary, got %v", ss)
	}
}

// TestUpdateImagesErrorsIncludePhaseContext verifies preparation and operation errors preserve their causes.
func TestUpdateImagesErrorsIncludePhaseContext(t *testing.T) {
	err := UpdateImages(bytes.NewReader(nil), bytes.NewReader(nil), io.Discard, 1, 0, "", nil)
	if !errors.Is(err, pdfcpu.ErrEmptyInput) {
		t.Fatalf("expected %v, got %v", pdfcpu.ErrEmptyInput, err)
	}
	if !strings.Contains(err.Error(), "update images: prepare PDF context") {
		t.Fatalf("expected update preparation context, got %q", err)
	}

	err = UpdateImages(openAPITestPDF(t, imageTestPDF()), bytes.NewReader(nil), io.Discard, 8, 0, "", nil)
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	if !strings.Contains(err.Error(), "update images: replace image object") {
		t.Fatalf("expected update operation context, got %q", err)
	}
}

// TestUpdateImagesWriteErrorIncludesPhaseContext verifies output errors preserve their cause.
func TestUpdateImagesWriteErrorIncludesPhaseContext(t *testing.T) {
	wantErr := errors.New("image write failed")
	err := UpdateImages(
		openAPITestPDF(t, imageTestPDF()),
		openAPITestPDF(t, imageTestImage()),
		failingWriter{err: wantErr},
		8,
		0,
		"",
		nil,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "update images: write output") {
		t.Fatalf("expected output context, got %q", err)
	}
}

// TestUpdateImagesFileIOErrorContext verifies file opening and creation context.
func TestUpdateImagesFileIOErrorContext(t *testing.T) {
	missingInput := filepath.Join(t.TempDir(), "missing.pdf")
	err := UpdateImagesFile(missingInput, "image.png", "", 1, 0, "", nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "update images: open input "+missingInput) {
		t.Fatalf("expected input context, got %v", err)
	}

	missingImage := filepath.Join(t.TempDir(), "missing.png")
	err = UpdateImagesFile(imageTestPDF(), missingImage, "", 8, 0, "", nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "update images: open image "+missingImage) {
		t.Fatalf("expected image context, got %v", err)
	}

	missingOutput := filepath.Join(t.TempDir(), "missing", "out.pdf")
	err = UpdateImagesFile(imageTestPDF(), imageTestImage(), missingOutput, 8, 0, "", nil)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), "update images: create output") {
		t.Fatalf("expected output context, got %v", err)
	}
}

func newUpdateImagesTestFile(t *testing.T, pattern string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), pattern)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestCleanupUpdateImagesFilesPreservesOperationAndCleanupErrors verifies joined cleanup failures.
// TestCleanupUpdateImagesFilesPreservesRemovalError verifies cleanup removal causes remain discoverable.
// TestFinalizeUpdateImagesFilesOutputCloseFailureCleansUp verifies failed output finalization.
// TestFinalizeUpdateImagesFilesImageCloseFailureCleansUp verifies failed image-input finalization.
// TestFinalizeUpdateImagesFilesInputCloseFailureCleansUp verifies failed PDF-input finalization.
// TestFinalizeUpdateImagesFilesReplaceOutputFailureCleansUp verifies replacement and cleanup context.
// TestUpdateImagesFileFailurePreservesExistingOutput verifies delayed output replacement.
func TestUpdateImagesFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	want := []byte("existing output")
	if err := os.WriteFile(outFile, want, 0o600); err != nil {
		t.Fatal(err)
	}
	imageFile := filepath.Join(t.TempDir(), "invalid.png")
	if err := os.WriteFile(imageFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err := UpdateImagesFile(imageTestPDF(), imageFile, outFile, 8, 0, "", nil)
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	got, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("existing output changed: got %q, want %q", got, want)
	}
}

// TestUpdateImagesFileSuccessReplacesExistingOutput verifies successful delayed replacement.
func TestUpdateImagesFileSuccessReplacesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	if err := os.WriteFile(outFile, []byte("existing output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpdateImagesFile(imageTestPDF(), imageTestImage(), outFile, 8, 0, "", nil); err != nil {
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

// TestUpdateImagesFileFailurePreservesInput verifies failed in-place updates leave the PDF unchanged.
func TestUpdateImagesFileFailurePreservesInput(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "input.pdf")
	want, err := os.ReadFile(imageTestPDF())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inFile, want, 0o600); err != nil {
		t.Fatal(err)
	}
	imageFile := filepath.Join(t.TempDir(), "invalid.png")
	if err := os.WriteFile(imageFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	err = UpdateImagesFile(inFile, imageFile, "", 8, 0, "", nil)
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	got, readErr := os.ReadFile(inFile)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("failed in-place update changed the input PDF")
	}
}

// TestUpdateImagesFileSuccessReplacesInput verifies successful in-place replacement.
func TestUpdateImagesFileSuccessReplacesInput(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "input.pdf")
	b, err := os.ReadFile(imageTestPDF())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inFile, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpdateImagesFile(inFile, imageTestImage(), "", 8, 0, "", nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) || bytes.Equal(got, b) {
		t.Fatal("expected updated in-place PDF")
	}
}

// TestUpdateImagesFileRejectsImageOutputAlias verifies output cannot replace the source image.
func TestUpdateImagesFileRejectsImageOutputAlias(t *testing.T) {
	err := UpdateImagesFile(imageTestPDF(), imageTestImage(), imageTestImage(), 8, 0, "", nil)
	if !errors.Is(err, ErrUpdateImagesOutputConflict) {
		t.Fatalf("expected %v, got %v", ErrUpdateImagesOutputConflict, err)
	}
	if !strings.Contains(err.Error(), "aliases image input") {
		t.Fatalf("expected alias context, got %q", err)
	}
}

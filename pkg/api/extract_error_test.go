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
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestUnsupportedResourceErrorPreservesCauses verifies the corresponding behavior.
func TestUnsupportedResourceErrorPreservesCauses(t *testing.T) {
	cause := fmt.Errorf("image obj#7: %w (%w)", pdfcpu.ErrUnsupportedResource, filter.ErrUnsupportedFilter)
	err := unsupportedResourceError(cause)

	var unsupportedErr *UnsupportedResourceError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected %T, got %T", unsupportedErr, err)
	}
	for _, wantErr := range []error{pdfcpu.ErrUnsupportedResource, filter.ErrUnsupportedFilter} {
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	}
}

// TestUnsupportedResourceErrorRemovedFromMixedCleanupFailure verifies the corresponding behavior.
func TestUnsupportedResourceErrorRemovedFromMixedCleanupFailure(t *testing.T) {
	cause := fmt.Errorf("image obj#7: %w (%w)", pdfcpu.ErrUnsupportedResource, filter.ErrUnsupportedFilter)
	skipErr := fmt.Errorf("extract images input.pdf: %w", unsupportedResourceError(cause))
	cleanupErr := errors.New("close failed")
	err := joinExtractionCleanupError(skipErr, cleanupErr)
	if got, want := err.Error(), skipErr.Error()+"\n"+cleanupErr.Error(); got != want {
		t.Fatalf("error text: got %q, want %q", got, want)
	}

	var unsupportedErr *UnsupportedResourceError
	if errors.As(err, &unsupportedErr) {
		t.Fatalf("mixed failure must not be classified as skip-only: %v", err)
	}
	for _, wantErr := range []error{pdfcpu.ErrUnsupportedResource, filter.ErrUnsupportedFilter, cleanupErr} {
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
	}
}

// TestSkipUnsupportedResourceHonorsPolicy verifies operational extraction policy handling.
func TestSkipUnsupportedResourceHonorsPolicy(t *testing.T) {
	err := fmt.Errorf("image obj#7: %w", pdfcpu.ErrUnsupportedResource)
	conf := model.NewDefaultConfiguration()

	conf.UnsupportedResourcePolicy = model.UnsupportedResourceSkip
	if !skipUnsupportedResource(err, conf) {
		t.Fatal("expected unsupported resource to be skipped")
	}

	conf.UnsupportedResourcePolicy = model.UnsupportedResourceFail
	if skipUnsupportedResource(err, conf) {
		t.Fatal("expected unsupported resource to fail")
	}

	if skipUnsupportedResource(errors.New("decode failed"), conf) {
		t.Fatal("expected non-unsupported error to fail")
	}
}

// TestWriteImageToDiskRejectsNilReader verifies missing image data is not reported as successful output.
func TestWriteImageToDiskRejectsNilReader(t *testing.T) {
	var typedNil *bytes.Reader
	tests := []model.Image{
		{ObjNr: 7},
		{Reader: typedNil, ObjNr: 8},
	}

	for _, img := range tests {
		t.Run(fmt.Sprintf("object %d", img.ObjNr), func(t *testing.T) {
			outDir := t.TempDir()
			err := WriteImageToDisk(outDir, "input")(img, false, 1)
			if !errors.Is(err, ErrMissingImageReader) {
				t.Fatalf("expected %v, got %v", ErrMissingImageReader, err)
			}
			if want := fmt.Sprintf("image obj#%d", img.ObjNr); !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %q", want, err.Error())
			}
			entries, readErr := os.ReadDir(outDir)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("expected no output, got %v", entries)
			}
		})
	}
}

func assertNoExtractedOutput(t *testing.T, outDir string) {
	t.Helper()

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no output, got %v", entries)
	}
}

// TestWriteFontToDiskRejectsNilReader verifies font wrappers expose missing embedded readers.
func TestWriteFontToDiskRejectsNilReader(t *testing.T) {
	var typedNil *bytes.Reader
	tests := []pdfcpu.Font{
		{Name: "NilFont", Type: "ttf", ObjNr: 7},
		{Reader: typedNil, Name: "TypedNilFont", Type: "ttf", ObjNr: 8},
	}

	for _, font := range tests {
		t.Run(font.Name, func(t *testing.T) {
			outDir := t.TempDir()
			err := WriteFontToDisk(outDir, "input")(font)
			if !errors.Is(err, ErrMissingReader) {
				t.Fatalf("expected %v, got %v", ErrMissingReader, err)
			}
			assertNoExtractedOutput(t, outDir)
		})
	}
}

// TestWriteMetadataToDiskRejectsNilReader verifies metadata wrappers expose missing embedded readers.
func TestWriteMetadataToDiskRejectsNilReader(t *testing.T) {
	var typedNil *bytes.Reader
	tests := []pdfcpu.Metadata{
		{ObjNr: 70, ParentObjNr: 7, ParentType: "Catalog"},
		{Reader: typedNil, ObjNr: 80, ParentObjNr: 8, ParentType: "Catalog"},
	}

	for _, md := range tests {
		t.Run(fmt.Sprintf("object %d", md.ObjNr), func(t *testing.T) {
			outDir := t.TempDir()
			err := WriteMetadataToDisk(outDir, "input")(md)
			if !errors.Is(err, ErrMissingReader) {
				t.Fatalf("expected %v, got %v", ErrMissingReader, err)
			}
			assertNoExtractedOutput(t, outDir)
		})
	}
}

// TestDigestImagesUsesObjectNumberOrder verifies the corresponding behavior.
func TestDigestImagesUsesObjectNumberOrder(t *testing.T) {
	images := map[int]model.Image{
		9: {ObjNr: 9},
		2: {ObjNr: 2},
		5: {ObjNr: 5},
	}
	var got []int
	objNr, err := digestImages(images, false, 1, func(img model.Image, _ bool, _ int) error {
		got = append(got, img.ObjNr)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if objNr != 0 {
		t.Fatalf("failing object number: got %d, want 0", objNr)
	}
	if want := []int{2, 5, 9}; !slices.Equal(got, want) {
		t.Fatalf("callback order: got %v, want %v", got, want)
	}

	wantErr := errors.New("digest failed")
	objNr, err = digestImages(images, false, 1, func(model.Image, bool, int) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if objNr != 2 {
		t.Fatalf("failing object number: got %d, want 2", objNr)
	}
}

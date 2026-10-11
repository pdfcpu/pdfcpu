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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func mergeContextAPICalls(c context.Context) []struct {
	name string
	call func() error
} {
	return []struct {
		name string
		call func() error
	}{
		{"merge raw", func() error { return MergeRaw(c, nil, nil, false, nil) }},
		{"merge", func() error { return Merge(c, "", nil, nil, nil, false) }},
		{"merge create file", func() error { return MergeCreateFile(c, nil, "", false, nil) }},
		{"merge append file", func() error { return MergeAppendFile(c, nil, "", false, nil) }},
		{"merge create zip", func() error { return MergeCreateZip(c, nil, nil, nil, nil) }},
		{"merge create zip file", func() error { return MergeCreateZipFile(c, "", "", "", nil) }},
	}
}

func TestMergeContextAPIsRejectNilContext(t *testing.T) {
	for _, tt := range mergeContextAPICalls(nil) {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, ErrMissingContext) {
				t.Fatalf("got %v, want ErrMissingContext", err)
			}
		})
	}
}

func TestMergeContextAPIsReturnCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()
	for _, tt := range mergeContextAPICalls(c) {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want context.Canceled", err)
			}
		})
	}
}

func TestMergeCreateFileCancellationPreservesDestination(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := MergeCreateFile(c, nil, outFile, false, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

type mergeValidationModeTest struct {
	name string
	run  func(*model.Configuration) error
}

func mergeValidationModeTests(c context.Context, validFile, relaxedOnlyFile string) []mergeValidationModeTest {
	valid := strictValidationTestPDF()
	relaxedOnly := relaxedOnlyValidationTestPDF()

	return []mergeValidationModeTest{
		{"raw destination", func(conf *model.Configuration) error {
			return MergeRaw(
				c,
				[]io.ReadSeeker{bytes.NewReader(relaxedOnly), bytes.NewReader(valid)},
				io.Discard,
				false,
				conf,
			)
		}},
		{"raw source", func(conf *model.Configuration) error {
			return MergeRaw(
				c,
				[]io.ReadSeeker{bytes.NewReader(valid), bytes.NewReader(relaxedOnly)},
				io.Discard,
				false,
				conf,
			)
		}},
		{"create destination", func(conf *model.Configuration) error {
			return Merge(c, "", []string{relaxedOnlyFile, validFile}, io.Discard, conf, false)
		}},
		{"create source", func(conf *model.Configuration) error {
			return Merge(c, "", []string{validFile, relaxedOnlyFile}, io.Discard, conf, false)
		}},
		{"append destination", func(conf *model.Configuration) error {
			return Merge(c, relaxedOnlyFile, []string{validFile}, io.Discard, conf, false)
		}},
		{"append source", func(conf *model.Configuration) error {
			return Merge(c, validFile, []string{relaxedOnlyFile}, io.Discard, conf, false)
		}},
		{"zip destination", func(conf *model.Configuration) error {
			return MergeCreateZip(c, bytes.NewReader(relaxedOnly), bytes.NewReader(valid), io.Discard, conf)
		}},
		{"zip source", func(conf *model.Configuration) error {
			return MergeCreateZip(c, bytes.NewReader(valid), bytes.NewReader(relaxedOnly), io.Discard, conf)
		}},
	}
}

func mergeValidationConfiguration(mode int) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = mode
	conf.CreateBookmarks = false
	conf.OptimizeBeforeWriting = false
	return conf
}

func TestMergeHonorsValidationModeForEverySubject(t *testing.T) {
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "valid.pdf")
	relaxedOnlyFile := filepath.Join(tmpDir, "relaxed-only.pdf")
	if err := os.WriteFile(validFile, strictValidationTestPDF(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relaxedOnlyFile, relaxedOnlyValidationTestPDF(), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range mergeValidationModeTests(t.Context(), validFile, relaxedOnlyFile) {
		t.Run(tt.name, func(t *testing.T) {
			strict := mergeValidationConfiguration(model.ValidationStrict)
			err := tt.run(strict)
			var validationErr *model.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("strict mode: expected validation error, got %v", err)
			}
			if strict.ValidationMode != model.ValidationStrict {
				t.Fatalf("caller validation mode: got %d, want %d", strict.ValidationMode, model.ValidationStrict)
			}

			relaxed := mergeValidationConfiguration(model.ValidationRelaxed)
			if err := tt.run(relaxed); err != nil {
				t.Fatalf("relaxed mode: %v", err)
			}
			if relaxed.ValidationMode != model.ValidationRelaxed {
				t.Fatalf("caller validation mode: got %d, want %d", relaxed.ValidationMode, model.ValidationRelaxed)
			}
		})
	}
}

// TestMergeAppendFileFailurePreservesExistingOutput verifies staged append publication.
func TestMergeAppendFileFailurePreservesExistingOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "append.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0640); err != nil {
		t.Fatal(err)
	}

	if err := MergeAppendFile(t.Context(), nil, outFile, false, nil); err == nil {
		t.Fatal("expected merge-append failure")
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

func TestMergeAppendFileCancellationPreservesDestination(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "append.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := MergeAppendFile(c, nil, outFile, false, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

func TestMergeCreateZipFileCancellationPreservesDestination(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "zip.pdf")
	original := []byte("existing output")
	if err := os.WriteFile(outFile, original, 0o640); err != nil {
		t.Fatal(err)
	}
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := MergeCreateZipFile(c, "", "", outFile, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	bb, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bb, original) {
		t.Fatalf("existing output changed: got %q, want %q", bb, original)
	}
}

func zipMergeAttrsPDF(pageCount int, rootAttrs, pageAttrs, content string) []byte {
	kids := "3 0 R"
	if pageCount == 2 {
		kids += " 6 0 R"
	}
	page := "<< /Type /Page /Parent 2 0 R /Contents 4 0 R " + pageAttrs + " >>"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d %s >>", kids, pageCount, rootAttrs),
		page,
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		page,
		"<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>",
		"[0 0 250 350]",
		"<< /Font << /F1 5 0 R >> >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = appendValidationTestObject(&b, i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Root 1 0 R /Size %d >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

func checkZipMergeAttrs(t *testing.T, ctx *model.Context, pageNr int, media, crop *types.Rectangle, rotate int, font string) {
	t.Helper()
	_, _, attrs, err := ctx.PageDict(t.Context(), pageNr, false)
	if err != nil {
		t.Fatal(err)
	}
	if attrs.MediaBox == nil || !attrs.MediaBox.Equals(*media) {
		t.Errorf("page %d: MediaBox = %v, want %v", pageNr, attrs.MediaBox, media)
	}
	effectiveCrop := attrs.CropBox
	if effectiveCrop == nil {
		effectiveCrop = attrs.MediaBox
	}
	if effectiveCrop == nil || !effectiveCrop.Equals(*crop) {
		t.Errorf("page %d: CropBox = %v, want %v", pageNr, effectiveCrop, crop)
	}
	if attrs.Rotate != rotate {
		t.Errorf("page %d: Rotate = %d, want %d", pageNr, attrs.Rotate, rotate)
	}
	fonts, err := ctx.DereferenceDict(attrs.Resources["Font"])
	if err != nil {
		t.Fatal(err)
	}
	if font == "" {
		if len(fonts) != 0 {
			t.Errorf("page %d: unexpected inherited fonts: %v", pageNr, fonts)
		}
		return
	}
	fd, err := ctx.DereferenceDict(fonts["F1"])
	if err != nil {
		t.Fatal(err)
	}
	if got := fd.NameEntry("BaseFont"); got == nil || *got != font {
		t.Errorf("page %d: F1 font = %v, want %s", pageNr, fd, font)
	}
}

// TestMergeZipPreservesPageAttributes covers inserted and appended pages with inherited and local attributes.
func TestMergeZipPreservesPageAttributes(t *testing.T) {
	const text = "BT /F1 12 Tf 10 10 Td (Hi) Tj ET"
	const resources = "/Resources << /Font << /F1 5 0 R >> >>"
	const crop = "/CropBox [0 0 250 350] "
	dest := zipMergeAttrsPDF(1,
		"/MediaBox [0 0 600 800] /Rotate 180 /CropBox [0 0 100 100] /Resources << /Font << /F1 7 0 R >> >>",
		"", text)
	tests := []struct {
		name, root, page, content, font string
		crop                            *types.Rectangle
	}{
		{"inherited", crop + resources, "", text, "Helvetica", types.RectForDim(250, 350)},
		{"indirect inherited", "/CropBox 8 0 R /Resources 9 0 R", "", text, "Helvetica", types.RectForDim(250, 350)},
		{"local overrides", crop + resources,
			"/CropBox [10 20 200 300] /Resources << /Font << /F1 7 0 R >> >>",
			text, "Courier", types.NewRectangle(10, 20, 200, 300)},
		{"MediaBox default", resources, "", text, "Helvetica", types.RectForDim(300, 400)},
		{"empty local resources", crop + resources, "/Resources << >>", "0 0 m 10 10 l S", "", types.RectForDim(250, 350)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := zipMergeAttrsPDF(2, "/MediaBox [0 0 300 400] /Rotate 90 "+tc.root, tc.page, tc.content)
			conf := model.NewStatelessConfiguration()
			conf.ValidationMode = model.ValidationStrict
			conf.Offline = true
			var out bytes.Buffer
			if err := MergeCreateZip(t.Context(), bytes.NewReader(dest), bytes.NewReader(src), &out, conf); err != nil {
				t.Fatal(err)
			}
			ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(out.Bytes()), conf)
			if err != nil {
				t.Fatal(err)
			}
			if ctx.PageCount != 3 {
				t.Fatalf("page count = %d, want 3", ctx.PageCount)
			}
			checkZipMergeAttrs(t, ctx, 1, types.RectForDim(600, 800), types.RectForDim(100, 100), 180, "Courier")
			// Page 2 is woven in; page 3 is the source remainder appended after the destination.
			for _, pageNr := range []int{2, 3} {
				checkZipMergeAttrs(t, ctx, pageNr, types.RectForDim(300, 400), tc.crop, 90, tc.font)
			}
		})
	}
}

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
	"compress/zlib"
	"errors"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func flateImageData(t *testing.T, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zlib.NewWriter(&b)
	if _, err := zw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func imageStreamObject(dict string, data []byte) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "<< %s /Length %d >>\nstream\n", dict, len(data))
	b.Write(data)
	b.WriteString("\nendstream")
	return b.Bytes()
}

func imageDecodeLimitPDF(primaryDict string, primary []byte, supportDict string, support []byte) []byte {
	objects := [][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] " +
			"/Resources << /XObject << /Im1 4 0 R >> >> /Contents 5 0 R >>"),
		imageStreamObject(primaryDict, primary),
		imageStreamObject("", nil),
	}
	if supportDict != "" {
		objects = append(objects, imageStreamObject(supportDict, support))
	}

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		objNr := i + 1
		offsets[objNr] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n", objNr)
		pdf.Write(object)
		pdf.WriteString("\nendobj\n")
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f\n", len(offsets))
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n\n", offsets[i])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Root 1 0 R /Size %d >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return pdf.Bytes()
}

func extractImagesWithDecodeLimit(t *testing.T, pdf []byte, limit int64) ([]map[int]model.Image, error) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationStrict
	conf.Limits.MaxDecodeBytes = limit
	return ExtractImagesRaw(t.Context(), bytes.NewReader(pdf), nil, conf)
}

func TestExtractImagesRawEnforcesPrimaryImageDecodeLimit(t *testing.T) {
	raw := []byte{0x12, 0x34, 0x56}
	pdf := imageDecodeLimitPDF(
		"/Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceRGB "+
			"/BitsPerComponent 8 /Filter /FlateDecode",
		flateImageData(t, raw),
		"",
		nil,
	)

	if _, err := extractImagesWithDecodeLimit(t, pdf, int64(len(raw)-1)); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("got %v, want %v", err, filter.ErrDecodeLimitExceeded)
	}

	images, err := extractImagesWithDecodeLimit(t, pdf, int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || len(images[0]) != 1 {
		t.Fatalf("got %d pages of images, want one image on one page", len(images))
	}
}

func TestExtractImagesRawEnforcesSoftMaskDecodeLimit(t *testing.T) {
	pdf := imageDecodeLimitPDF(
		"/Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceRGB "+
			"/BitsPerComponent 8 /SMask 6 0 R",
		[]byte{0x12, 0x34, 0x56},
		"/Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceGray "+
			"/BitsPerComponent 8 /Filter /FlateDecode",
		flateImageData(t, []byte{0, 0, 0, 0}),
	)

	if _, err := extractImagesWithDecodeLimit(t, pdf, 3); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("got %v, want %v", err, filter.ErrDecodeLimitExceeded)
	}
}

func TestExtractImagesRawEnforcesColorLookupDecodeLimit(t *testing.T) {
	pdf := imageDecodeLimitPDF(
		"/Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace [/Indexed /DeviceRGB 0 6 0 R] "+
			"/BitsPerComponent 8",
		[]byte{0},
		"/Filter /FlateDecode",
		flateImageData(t, []byte{0x12, 0x34, 0x56, 0x78}),
	)

	if _, err := extractImagesWithDecodeLimit(t, pdf, 3); !errors.Is(err, filter.ErrDecodeLimitExceeded) {
		t.Fatalf("got %v, want %v", err, filter.ErrDecodeLimitExceeded)
	}
}

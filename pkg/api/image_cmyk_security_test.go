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
	"fmt"
	"strings"
	"testing"

	"github.com/hhrutter/tiff"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func cmykExtractionPDF(bpc int, pixels []byte) []byte {
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	off := make([]int, 6)
	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 10 10] /Resources << /XObject << /Im1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n",
	}
	for i, obj := range objects {
		off[i+1] = pdf.Len()
		pdf.WriteString(obj)
	}
	off[4] = pdf.Len()
	fmt.Fprintf(&pdf, "4 0 obj\n<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceCMYK /BitsPerComponent %d /Length %d >>\nstream\n", bpc, len(pixels))
	pdf.Write(pixels)
	pdf.WriteString("\nendstream\nendobj\n")
	off[5] = pdf.Len()
	pdf.WriteString("5 0 obj\n<< /Length 0 >>\nstream\n\nendstream\nendobj\n")
	xref := pdf.Len()
	pdf.WriteString("xref\n0 6\n0000000000 65535 f\n")
	for i := 1; i < len(off); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n\n", off[i])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Root 1 0 R /Size 6 >>\nstartxref\n%d\n%%%%EOF\n", xref)
	return pdf.Bytes()
}

func TestExtractImagesRawRejectsInvalidCMYKSamples(t *testing.T) {
	for _, tt := range []struct {
		name   string
		bpc    int
		pixels []byte
		want   string
	}{
		{"short 8-bit", 8, []byte{0}, "corrupt image object: need 4 bytes, have 1"},
		{"packed 1-bit", 1, []byte{0}, "unsupported bits per component 1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conf := model.NewStatelessConfiguration()
			conf.ValidationMode = model.ValidationStrict
			_, err := ExtractImagesRaw(t.Context(), bytes.NewReader(cmykExtractionPDF(tt.bpc, tt.pixels)), nil, conf)
			if err == nil || !strings.Contains(err.Error(), "image obj#4 CMYK: "+tt.want) {
				t.Fatalf("got %v, want CMYK sample error", err)
			}
		})
	}
}

func TestExtractImagesRawAcceptsValidCMYKSamples(t *testing.T) {
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationStrict
	images, err := ExtractImagesRaw(t.Context(), bytes.NewReader(cmykExtractionPDF(8, []byte{1, 2, 3, 4})), nil, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || len(images[0]) != 1 {
		t.Fatalf("got %d pages of images, want one image on one page", len(images))
	}
	im, ok := images[0][4]
	if !ok || im.FileType != "tif" {
		t.Fatalf("got image %v, want TIFF for object 4", im)
	}
	decoded, err := tiff.Decode(im.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 1 || decoded.Bounds().Dy() != 1 {
		t.Fatalf("got image bounds %v, want 1x1", decoded.Bounds())
	}
}

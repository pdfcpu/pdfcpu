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

// GHSA-9mmx-88p2-c8c6: packed DeviceRGB and CalRGB samples must not be read past the end of the image data.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-9mmx-88p2-c8c6

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ghsa9mmxRGBImagePDF(colorSpace string, bpc int) []byte {
	const height = 2
	rowBytes := (3*bpc + 7) / 8
	imageData := make([]byte, rowBytes*height)
	for i := rowBytes; i < len(imageData); i++ {
		imageData[i] = 0xff
	}
	pageContent := []byte("q 1 0 0 2 0 0 cm /Im0 Do Q")

	objects := [][]byte{
		nil,
		[]byte("<</Type/Catalog/Pages 2 0 R>>"),
		[]byte("<</Type/Pages/Kids[3 0 R]/Count 1>>"),
		[]byte("<</Type/Page/Parent 2 0 R/MediaBox[0 0 1 2]/Resources<</XObject<</Im0 5 0 R>>>>/Contents 4 0 R>>"),
		fmt.Appendf(nil, "<</Length %d>>\nstream\n%s\nendstream", len(pageContent), pageContent),
		fmt.Appendf(
			nil,
			"<</Type/XObject/Subtype/Image/Width 1/Height %d/BitsPerComponent %d/ColorSpace%s/Length %d>>\nstream\n",
			height, bpc, colorSpace, len(imageData),
		),
	}
	objects[5] = append(objects[5], imageData...)
	objects[5] = append(objects[5], []byte("\nendstream")...)

	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i := 1; i < len(objects); i++ {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n", i)
		b.Write(objects[i])
		b.WriteString("\nendobj\n")
	}

	xrefOffset := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects))
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objects), xrefOffset)
	return b.Bytes()
}

// TestGHSA_9mmx_88p2_c8c6_ExtractImagesRawRGBBitsPerComponent verifies packed RGB samples and row padding are rendered
// safely.
func TestGHSA_9mmx_88p2_c8c6_ExtractImagesRawRGBBitsPerComponent(t *testing.T) {
	colorSpaces := []struct {
		name  string
		value string
	}{
		{"DeviceRGB", "/DeviceRGB"},
		{"CalRGB", "[/CalRGB<</WhitePoint[0.9505 1 1.089]>>]"},
	}
	for _, cs := range colorSpaces {
		for _, bpc := range []int{1, 2, 4, 8, 16} {
			t.Run(fmt.Sprintf("%s/%d", cs.name, bpc), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.ValidationMode = model.ValidationStrict
				images, err := ExtractImagesRaw(t.Context(), bytes.NewReader(ghsa9mmxRGBImagePDF(cs.value, bpc)), nil, conf)
				if err != nil {
					t.Fatal(err)
				}
				if len(images) != 1 || len(images[0]) != 1 {
					t.Fatalf("got image maps %v, want one image", images)
				}
				im, ok := images[0][5]
				if !ok {
					t.Fatal("missing image object 5")
				}
				ghsa9mmxCheckBlackOverWhite(t, im.Reader)
			})
		}
	}
}

// ghsa9mmxCheckBlackOverWhite decodes a 1x2 PNG and checks for a black top and a white bottom pixel.
func ghsa9mmxCheckBlackOverWhite(t *testing.T, r io.Reader) {
	t.Helper()
	decoded, err := png.Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded.Bounds().Size(); got.X != 1 || got.Y != 2 {
		t.Fatalf("got image size %v, want 1x2", got)
	}
	for y, want := range []uint32{0, 0xffff} {
		r, g, b, a := decoded.At(0, y).RGBA()
		if r != want || g != want || b != want || a != 0xffff {
			t.Fatalf("got RGBA %04x/%04x/%04x/%04x at row %d, want %04x/%04x/%04x/ffff", r, g, b, a, y, want, want, want)
		}
	}
}

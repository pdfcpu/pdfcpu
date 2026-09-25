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

package model

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/hhrutter/tiff"
)

// TestImageBufferRejectsOverflow verifies a decoded image cannot panic during buffer sizing.
func TestImageBufferRejectsOverflow(t *testing.T) {
	conf := NewDefaultConfiguration()
	conf.Limits.MaxImagePixels = math.MaxInt64
	conf.Limits.MaxImageBytes = math.MaxInt64
	xRefTable := newXRefTable(conf)

	tests := []struct {
		name string
		img  image.Image
	}{
		{"RGBA", &image.RGBA{Rect: image.Rect(0, 0, math.MaxInt/3+1, 1)}},
		{"RGBA64", &image.RGBA64{Rect: image.Rect(0, 0, math.MaxInt/6+1, 1)}},
		{"NRGBA", &image.NRGBA{Rect: image.Rect(0, 0, math.MaxInt/3+1, 1)}},
		{"NRGBA64", &image.NRGBA64{Rect: image.Rect(0, 0, math.MaxInt/6+1, 1)}},
		{"Gray", &image.Gray{Rect: image.Rect(0, 0, math.MaxInt, 2)}},
		{"Gray16", &image.Gray16{Rect: image.Rect(0, 0, math.MaxInt/2+1, 1)}},
		{"CMYK", &image.CMYK{Rect: image.Rect(0, 0, math.MaxInt/4+1, 1)}},
		{"Paletted", &image.Paletted{Rect: image.Rect(0, 0, math.MaxInt, 2), Palette: color.Palette{color.Black}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("buffer sizing panicked: %v", r)
				}
			}()

			var err error
			if img, ok := tt.img.(*image.Paletted); ok {
				_, err = createPalettedImageStreamDict(xRefTable, img)
			} else {
				_, _, _, _, err = createImageBuf(xRefTable, tt.img, nil, "png")
			}
			if err == nil {
				t.Fatal("expected buffer size error")
			}
		})
	}
}

// TestCreateImageStreamDictHonorsBufferLimit verifies the limit covers the actual 16-bit RGB buffer.
func TestCreateImageStreamDictHonorsBufferLimit(t *testing.T) {
	img := image.NewRGBA64(image.Rect(0, 0, 1, 1))
	img.SetRGBA64(0, 0, color.RGBA64{R: 0x1234, G: 0x5678, B: 0x9abc, A: 0xffff})
	data := encodedImage(t, img)

	for _, tt := range []struct {
		limit   int64
		wantErr bool
	}{
		{limit: 5, wantErr: true},
		{limit: 8},
	} {
		xRefTable := imageConstructionXRefTable(false)
		xRefTable.Conf.Limits.MaxImageBytes = tt.limit
		_, _, _, err := CreateImageStreamDict(xRefTable, bytes.NewReader(data))
		if tt.wantErr {
			if err == nil || !strings.Contains(err.Error(), "byte") {
				t.Errorf("limit %d: expected byte limit error, got %v", tt.limit, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("limit %d: unexpected error: %v", tt.limit, err)
		}
	}
}

// TestCreateImageResourcesEnforcesEachTIFFPageLimit verifies later pages cannot bypass image limits.
func TestCreateImageResourcesEnforcesEachTIFFPageLimit(t *testing.T) {
	var buf bytes.Buffer
	pages := []image.Image{
		image.NewGray(image.Rect(0, 0, 1, 1)),
		image.NewGray(image.Rect(0, 0, 2, 1)),
	}
	if err := tiff.EncodeAll(&buf, pages, nil); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name      string
		maxPixels int64
		maxBytes  int64
		wantErr   string
	}{
		{name: "pixel limit", maxPixels: 1, maxBytes: 8, wantErr: "pixel count"},
		{name: "byte limit", maxPixels: 2, maxBytes: 4, wantErr: "byte size"},
		{name: "within limits", maxPixels: 2, maxBytes: 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := imageConstructionXRefTable(false)
			xRefTable.Conf.Limits.MaxImagePixels = tt.maxPixels
			xRefTable.Conf.Limits.MaxImageBytes = tt.maxBytes
			resources, err := CreateImageResources(xRefTable, bytes.NewReader(buf.Bytes()), false, false)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected %q limit error, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil || len(resources) != len(pages) {
				t.Errorf("got %d resources, error %v", len(resources), err)
			}
		})
	}
}

// TestCreateImageResourcesChecksTIFF16BitPageBeforeDecode verifies the decoded buffer limit for later pages.
func TestCreateImageResourcesChecksTIFF16BitPageBeforeDecode(t *testing.T) {
	var buf bytes.Buffer
	pages := []image.Image{
		image.NewGray(image.Rect(0, 0, 1, 1)),
		image.NewRGBA64(image.Rect(0, 0, 1, 1)),
	}
	if err := tiff.EncodeAll(&buf, pages, nil); err != nil {
		t.Fatal(err)
	}

	xRefTable := imageConstructionXRefTable(false)
	xRefTable.Conf.Limits.MaxImageBytes = 4
	_, err := CreateImageResources(xRefTable, bytes.NewReader(buf.Bytes()), false, false)
	if err == nil || !strings.Contains(err.Error(), "validate TIFF image at offset") || !strings.Contains(err.Error(), "byte size") {
		t.Errorf("expected 16-bit TIFF page to exceed the predecode byte limit, got %v", err)
	}
}

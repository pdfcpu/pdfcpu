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

package pdfcpu

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestRenderDeviceCMYKRejectsShortSoftMaskedImage(t *testing.T) {
	im := &PDFImage{
		objNr:    7,
		sd:       &types.StreamDict{Content: []byte{0}},
		bpc:      8,
		w:        1,
		h:        1,
		softMask: []byte{255},
	}
	_, _, err := renderDeviceCMYKToTIFF(im)
	if err == nil || !strings.Contains(err.Error(), "image obj#7 CMYK: corrupt image object") {
		t.Fatalf("got %v, want CMYK sample error", err)
	}
}

// TestRenderDeviceGray16BPC verifies that a valid 16-bit grayscale image renders without a panic.
func TestRenderDeviceGray16BPC(t *testing.T) {
	sd := &types.StreamDict{
		Dict: types.Dict{
			"BitsPerComponent": types.Integer(16),
			"ColorSpace":       types.Name(model.DeviceGrayCS),
			"Height":           types.Integer(1),
			"Width":            types.Integer(2),
		},
		Content: []byte{0, 0, 0xff, 0xff},
	}

	defer func() {
		if p := recover(); p != nil {
			t.Errorf("RenderImage panicked: %v", p)
		}
	}()

	r, ext, err := RenderImage(xRefTable, sd, false, "", 7)
	if err != nil {
		t.Fatal(err)
	}
	if ext != "png" {
		t.Fatalf("got extension %q, want png", ext)
	}
	img, err := png.Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []uint32{0, 0xffff} {
		got, _, _, _ := img.At(x, 0).RGBA()
		if got != want {
			t.Errorf("pixel %d: got gray %d, want %d", x, got, want)
		}
	}
}

// TestRenderIndexedPaletteBounds verifies that samples are clamped to the declared palette range.
func TestRenderIndexedPaletteBounds(t *testing.T) {
	tests := []struct {
		name   string
		baseCS types.Object
		lookup types.HexLiteral
	}{
		{"DeviceGray", types.Name(model.DeviceGrayCS), types.HexLiteral("00")},
		{"DeviceRGB", types.Name(model.DeviceRGBCS), types.HexLiteral("000000")},
		{"DeviceCMYK", types.Name(model.DeviceCMYKCS), types.HexLiteral("00000000")},
		{"CalRGB", types.Array{types.Name(model.CalRGBCS), types.Dict{}}, types.HexLiteral("000000")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, pixel := range []byte{0, 0xff} {
				t.Run(fmt.Sprintf("index_%d", pixel), func(t *testing.T) {
					defer func() {
						if p := recover(); p != nil {
							t.Errorf("RenderImage panicked: %v", p)
						}
					}()
					sd := &types.StreamDict{
						Dict: types.Dict{
							"BitsPerComponent": types.Integer(8),
							"ColorSpace": types.Array{
								types.Name(model.IndexedCS), tt.baseCS, types.Integer(0), tt.lookup,
							},
							"Height": types.Integer(1),
							"Width":  types.Integer(1),
						},
						Content: []byte{pixel},
					}
					_, _, err := RenderImage(xRefTable, sd, false, "", 7)
					if err != nil {
						t.Fatalf("clamped palette index: %v", err)
					}
				})
			}
		})
	}
}

// TestRenderIndexedDecode verifies Decode mapping, rounding, and clamping in supported Indexed colour spaces.
func TestRenderIndexedDecode(t *testing.T) {
	gray := types.HexLiteral("00ff")
	rgb := types.HexLiteral("000000ffffff")
	cmyk := types.HexLiteral("000000ff00000000")
	bases := []struct {
		name   string
		cs     types.Object
		lookup types.HexLiteral
		masked bool
	}{
		{"DeviceGray", types.Name(model.DeviceGrayCS), gray, false},
		{"DeviceRGB", types.Name(model.DeviceRGBCS), rgb, false},
		{"DeviceCMYK", types.Name(model.DeviceCMYKCS), cmyk, false},
		{"CalRGB", types.Array{types.Name(model.CalRGBCS), types.Dict{}}, rgb, false},
		{"ICCGray", indexedTestICCBase(1), gray, false},
		{"ICCRGB", indexedTestICCBase(3), rgb, false},
		{"ICCCMYK", indexedTestICCBase(4), cmyk, false},
		{"DeviceRGBMasked", types.Name(model.DeviceRGBCS), rgb, true},
		{"DeviceCMYKMasked", types.Name(model.DeviceCMYKCS), cmyk, true},
	}
	tests := []struct {
		name    string
		decode  types.Array
		samples []byte
		white   []bool
	}{
		{"default", nil, []byte{0, 1, 255}, []bool{false, true, true}},
		{"scaled", types.NewNumberArray(0, 1), []byte{0, 127, 128, 255}, []bool{false, false, true, true}},
		{"scaled_max", types.NewNumberArray(0, 1), []byte{255}, []bool{true}},
		{"inverted", types.NewNumberArray(1, 0), []byte{0, 127, 128, 255}, []bool{true, true, false, false}},
		{"half_up", types.NewNumberArray(0, 1.5), []byte{0, 85, 170, 255}, []bool{false, true, true, true}},
		{"below_half", types.NewNumberArray(0.25, 0.49), []byte{0, 255}, []bool{false, false}},
		{"clipped", types.NewNumberArray(-1, 2), []byte{0, 85, 170, 255}, []bool{false, false, true, true}},
	}
	for _, base := range bases {
		for _, tt := range tests {
			t.Run(base.name+"/"+tt.name, func(t *testing.T) {
				sd := indexedTestStream(base.cs, base.lookup, 8, len(tt.samples), 1, tt.samples)
				if tt.decode != nil {
					sd.Dict["Decode"] = tt.decode
				}
				alpha := uint32(0xffff)
				if base.masked {
					alpha = 0x8080
					mask := indexedTestStream(types.Name(model.DeviceGrayCS), "", 8, len(tt.samples), 1,
						bytes.Repeat([]byte{128}, len(tt.samples)))
					mask.Dict["ColorSpace"] = types.Name(model.DeviceGrayCS)
					sd.Dict["SMask"] = *mask
				}
				r, _, err := RenderImage(xRefTable, sd, false, "", 7)
				if err != nil {
					t.Fatal(err)
				}
				img, _, err := image.Decode(r)
				if err != nil {
					t.Fatal(err)
				}
				for x, white := range tt.white {
					want := uint32(0)
					if white {
						want = alpha
					}
					r, g, b, a := img.At(x, 0).RGBA()
					if r != want || g != want || b != want || a != alpha {
						t.Errorf("pixel %d: got RGBA (%d,%d,%d,%d), want (%d,%d,%d,%d)", x, r, g, b, a, want, want, want, alpha)
					}
				}
			})
		}
	}
}

func indexedTestICCBase(n int) types.Array {
	return types.Array{types.Name(model.ICCBasedCS), types.StreamDict{Dict: types.Dict{"N": types.Integer(n)}}}
}

func indexedTestStream(base types.Object, lookup types.HexLiteral, bpc, w, h int, samples []byte) *types.StreamDict {
	return &types.StreamDict{
		Dict: types.Dict{
			"BitsPerComponent": types.Integer(bpc),
			"ColorSpace":       types.Array{types.Name(model.IndexedCS), base, types.Integer(1), lookup},
			"Height":           types.Integer(h),
			"Width":            types.Integer(w),
		},
		Content: samples,
	}
}

// TestRenderIndexedPackedDecode verifies row padding and eight-bit palette values at every supported sample depth.
func TestRenderIndexedPackedDecode(t *testing.T) {
	tests := []struct {
		bpc     int
		samples []byte
	}{
		{1, []byte{0x5f, 0xbf}},
		{2, []byte{0x33, 0xcf}},
		{4, []byte{0x0f, 0x0f, 0xf0, 0xff}},
		{8, []byte{0, 255, 0, 255, 0, 255}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("bpc_%d", tt.bpc), func(t *testing.T) {
			sd := indexedTestStream(types.Name(model.DeviceGrayCS), types.HexLiteral("4080"), tt.bpc, 3, 2, tt.samples)
			sd.Dict["Decode"] = types.NewNumberArray(0, 1)
			r, _, err := RenderImage(xRefTable, sd, false, "", 7)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(r)
			if err != nil {
				t.Fatal(err)
			}
			for y, row := range [][]uint32{{0x4040, 0x8080, 0x4040}, {0x8080, 0x4040, 0x8080}} {
				for x, want := range row {
					got, _, _, _ := img.At(x, y).RGBA()
					if got != want {
						t.Errorf("pixel (%d,%d): got gray %d, want %d", x, y, got, want)
					}
				}
			}
		})
	}
}

// TestRenderIndexedRejectsWideSamples verifies that unsupported sample widths return an error.
func TestRenderIndexedRejectsWideSamples(t *testing.T) {
	for _, bpc := range []int{9, 16} {
		t.Run(fmt.Sprintf("bpc_%d", bpc), func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("RenderImage panicked: %v", p)
				}
			}()
			sd := &types.StreamDict{
				Dict: types.Dict{
					"BitsPerComponent": types.Integer(bpc),
					"ColorSpace": types.Array{
						types.Name(model.IndexedCS), types.Name(model.DeviceRGBCS),
						types.Integer(0), types.HexLiteral("000000"),
					},
					"Height": types.Integer(1),
					"Width":  types.Integer(1),
				},
				Content: []byte{0, 0},
			}
			if _, _, err := RenderImage(xRefTable, sd, false, "", 7); err == nil {
				t.Fatal("expected unsupported bits per component error")
			}
		})
	}
}

// TestRenderIndexedICCPackedGrayRows verifies that packed Indexed rows use their own byte offsets.
func TestRenderIndexedICCPackedGrayRows(t *testing.T) {
	profile := types.StreamDict{Dict: types.Dict{"N": types.Integer(1)}}
	sd := &types.StreamDict{
		Dict: types.Dict{
			"BitsPerComponent": types.Integer(1),
			"ColorSpace": types.Array{
				types.Name(model.IndexedCS),
				types.Array{types.Name(model.ICCBasedCS), profile},
				types.Integer(1), types.HexLiteral("00ff"),
			},
			"Height": types.Integer(2),
			"Width":  types.Integer(2),
		},
		Content: []byte{0x40, 0x80},
	}

	defer func() {
		if p := recover(); p != nil {
			t.Errorf("RenderImage panicked: %v", p)
		}
	}()

	r, _, err := RenderImage(xRefTable, sd, false, "", 7)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	for y, row := range [][]uint32{{0, 0xffff}, {0xffff, 0}} {
		for x, want := range row {
			got, _, _, _ := img.At(x, y).RGBA()
			if got != want {
				t.Errorf("pixel (%d,%d): got gray %d, want %d", x, y, got, want)
			}
		}
	}
}

// TestRenderIndexedCalRGBRejectsShortLookup verifies that the palette covers its declared HiVal.
func TestRenderIndexedCalRGBRejectsShortLookup(t *testing.T) {
	sd := &types.StreamDict{
		Dict: types.Dict{
			"BitsPerComponent": types.Integer(8),
			"ColorSpace": types.Array{
				types.Name(model.IndexedCS),
				types.Array{types.Name(model.CalRGBCS), types.Dict{}},
				types.Integer(1), types.HexLiteral("000000"),
			},
			"Height": types.Integer(1),
			"Width":  types.Integer(1),
		},
		Content: []byte{0},
	}

	if _, _, err := RenderImage(xRefTable, sd, false, "", 7); err == nil {
		t.Fatal("expected error for short CalRGB lookup table")
	}
}

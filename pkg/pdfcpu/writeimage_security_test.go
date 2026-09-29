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
	"fmt"
	"image/png"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

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

// TestRenderIndexedPaletteBounds verifies that a short valid palette renders and an invalid index returns an error.
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
					if pixel == 0 && err != nil {
						t.Fatalf("valid palette index: %v", err)
					}
					if pixel != 0 && err == nil {
						t.Fatal("expected error for palette index beyond HiVal")
					}
				})
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

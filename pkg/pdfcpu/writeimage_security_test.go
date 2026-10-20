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
	"image/color"
	"strings"
	"testing"

	"github.com/hhrutter/tiff"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestRenderDeviceCMYKRejectsShortSoftMaskedImage checks sample length before applying a soft mask.
func TestRenderDeviceCMYKRejectsShortSoftMaskedImage(t *testing.T) {
	im := &PDFImage{
		objNr:    7,
		comp:     4,
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

// TestRenderCMYKSampleDepths checks each sample depth, row padding, and soft masks through TIFF output.
func TestRenderCMYKSampleDepths(t *testing.T) {
	for _, tc := range []struct {
		bpc     int
		samples []byte
		pixels  []color.CMYK
	}{
		{1, []byte{0xcf, 0x3a}, []color.CMYK{{255, 255, 0, 0}, {0, 0, 255, 255}}},
		{2, []byte{0xe4, 0x1b}, []color.CMYK{{255, 170, 85, 0}, {0, 85, 170, 255}}},
		{4, []byte{0xf8, 0x40, 0x04, 0x8f}, []color.CMYK{{255, 136, 68, 0}, {0, 68, 136, 255}}},
		{8, []byte{255, 128, 64, 0, 0, 64, 128, 255}, []color.CMYK{{255, 128, 64, 0}, {0, 64, 128, 255}}},
		{16, []byte{255, 255, 128, 0, 64, 0, 0, 0, 0, 0, 64, 0, 128, 0, 255, 255}, []color.CMYK{{255, 128, 64, 0}, {0, 64, 128, 255}}},
	} {
		for _, masked := range []bool{false, true} {
			t.Run(fmt.Sprintf("bpc=%d/masked=%t", tc.bpc, masked), func(t *testing.T) {
				sd := cmykTestStream(tc.bpc, 1, 2, tc.samples)
				if masked {
					mask := cmykTestStream(8, 1, 2, []byte{255, 128})
					mask.Dict["ColorSpace"] = types.Name(model.DeviceGrayCS)
					sd.Dict["SMask"] = *mask
				}
				r, ext, err := RenderImage(xRefTable, sd, false, "", 7)
				if err != nil {
					t.Fatal(err)
				}
				if ext != "tif" {
					t.Fatalf("extension = %q, want tif", ext)
				}
				img, err := tiff.Decode(r)
				if err != nil {
					t.Fatal(err)
				}
				for y, want := range tc.pixels {
					checkCMYKTestPixel(t, img.At(0, y), want, masked, []uint8{255, 128}[y])
				}
			})
		}
	}
}

func cmykTestStream(bpc, w, h int, samples []byte) *types.StreamDict {
	return &types.StreamDict{Dict: types.Dict{
		"ColorSpace":       types.Name(model.DeviceCMYKCS),
		"BitsPerComponent": types.Integer(bpc),
		"Width":            types.Integer(w),
		"Height":           types.Integer(h),
	}, Content: samples}
}

func checkCMYKTestPixel(t *testing.T, got color.Color, want color.CMYK, masked bool, alpha uint8) {
	t.Helper()
	if !masked {
		if got != want {
			t.Fatalf("CMYK pixel = %v, want %v", got, want)
		}
		return
	}
	r, g, b := color.CMYKToRGB(want.C, want.M, want.Y, want.K)
	wr, wg, wb, wa := (color.NRGBA{R: r, G: g, B: b, A: alpha}).RGBA()
	gr, gg, gb, ga := got.RGBA()
	if gr != wr || gg != wg || gb != wb || ga != wa {
		t.Fatalf("RGBA pixel = (%d,%d,%d,%d), want (%d,%d,%d,%d)", gr, gg, gb, ga, wr, wg, wb, wa)
	}
}

// TestRenderCMYKRejectsInvalidSamples checks truncated rows and unsupported sample depths.
func TestRenderCMYKRejectsInvalidSamples(t *testing.T) {
	for _, bpc := range []int{1, 2, 4, 8, 16} {
		t.Run(fmt.Sprintf("short_bpc=%d", bpc), func(t *testing.T) {
			rowBytes := (3*4*bpc + 7) / 8
			sd := cmykTestStream(bpc, 3, 2, make([]byte, 2*rowBytes-1))
			if _, _, err := RenderImage(xRefTable, sd, false, "", 7); err == nil {
				t.Fatal("expected error for truncated CMYK data")
			}
		})
	}
	for _, bpc := range []int{0, 3, 32} {
		t.Run(fmt.Sprintf("unsupported_bpc=%d", bpc), func(t *testing.T) {
			sd := cmykTestStream(bpc, 1, 1, make([]byte, 16))
			if _, _, err := RenderImage(xRefTable, sd, false, "", 7); err == nil {
				t.Fatal("expected error for unsupported sample depth")
			}
		})
	}
}

// TestRenderCMYKComponentStride checks shared CMYK rendering with additional source components.
func TestRenderCMYKComponentStride(t *testing.T) {
	for _, masked := range []bool{false, true} {
		im := &PDFImage{
			objNr: 7, comp: 5, bpc: 8, w: 2, h: 1,
			sd: &types.StreamDict{Content: []byte{255, 128, 64, 0, 99, 0, 64, 128, 0, 77}},
		}
		if masked {
			im.softMask = []byte{255, 128}
		}
		r, _, err := renderDeviceCMYKToTIFF(im)
		if err != nil {
			t.Fatal(err)
		}
		img, err := tiff.Decode(r)
		if err != nil {
			t.Fatal(err)
		}
		checkCMYKTestPixel(t, img.At(0, 0), color.CMYK{255, 128, 64, 0}, masked, 255)
		checkCMYKTestPixel(t, img.At(1, 0), color.CMYK{0, 64, 128, 0}, masked, 128)
		im.comp = 3
		if _, _, err := renderDeviceCMYKToTIFF(im); err == nil {
			t.Fatal("expected error for fewer than four components")
		}
	}
}

// TestRenderIndexedRejectsMalformedColorSpaces checks array bounds before rendering an Indexed image.
func TestRenderIndexedRejectsMalformedColorSpaces(t *testing.T) {
	for _, tc := range []struct {
		name string
		cs   types.Array
	}{
		{"empty", types.Array{}},
		{"missing base", types.Array{types.Name(model.IndexedCS)}},
		{"missing hival and lookup", types.Array{types.Name(model.IndexedCS), types.Name(model.DeviceRGBCS)}},
		{"missing lookup", types.Array{types.Name(model.IndexedCS), types.Name(model.DeviceRGBCS), types.Integer(0)}},
		{"empty base", types.Array{types.Name(model.IndexedCS), types.Array{}, types.Integer(0), types.HexLiteral("000000")}},
		{"missing ICC profile", types.Array{types.Name(model.IndexedCS), types.Array{types.Name(model.ICCBasedCS)}, types.Integer(0), types.HexLiteral("000000")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sd := &types.StreamDict{
				Dict: types.Dict{
					"ColorSpace":       tc.cs,
					"BitsPerComponent": types.Integer(8),
					"Width":            types.Integer(1),
					"Height":           types.Integer(1),
				},
				Content: []byte{0},
			}
			if _, _, err := RenderImage(xRefTable, sd, false, "", 7); err == nil {
				t.Fatal("expected an error for malformed Indexed colorspace")
			}
		})
	}
}

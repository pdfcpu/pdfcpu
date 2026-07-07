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
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type failingImageReader struct {
	err error
}

func (r failingImageReader) Read([]byte) (int, error) {
	return 0, r.err
}

func imageConstructionXRefTable(brokenFreeList bool) *XRefTable {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	size := 1
	offset := int64(0)
	generation := types.FreeHeadGeneration
	xRefTable.Size = &size
	xRefTable.Table[0] = &XRefTableEntry{
		Free:       !brokenFreeList,
		Offset:     &offset,
		Generation: &generation,
	}
	return xRefTable
}

func encodedImage(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// TestCreateImageStreamDictReadErrorContext verifies image input failures preserve their cause.
func TestCreateImageStreamDictReadErrorContext(t *testing.T) {
	wantErr := errors.New("image read failed")
	_, _, _, err := CreateImageStreamDict(nil, failingImageReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "read image") {
		t.Fatalf("expected image read context, got %q", err)
	}
}

// TestImageConstructionRejectsMissingReader verifies model entry points share the missing-reader sentinel.
func TestImageConstructionRejectsMissingReader(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "resources",
			fn: func() error {
				_, err := CreateImageResources(nil, nil, false, false)
				return err
			},
		},
		{
			name: "stream dictionary",
			fn: func() error {
				_, _, _, err := CreateImageStreamDict(nil, nil)
				return err
			},
		},
		{
			name: "resource",
			fn: func() error {
				_, _, _, err := CreateImageResource(imageConstructionXRefTable(false), nil)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrMissingImageReader) {
				t.Fatalf("expected %v, got %v", ErrMissingImageReader, err)
			}
		})
	}
}

// TestCreateImageResourceValidatesReaderBeforeXRefTable verifies reader validation has precedence.
func TestCreateImageResourceValidatesReaderBeforeXRefTable(t *testing.T) {
	_, _, _, err := CreateImageResource(nil, nil)
	if !errors.Is(err, ErrMissingImageReader) {
		t.Fatalf("expected %v, got %v", ErrMissingImageReader, err)
	}
	if errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("unexpected lower-priority %v", ErrMissingXRefTable)
	}
}

// TestImageConstructionStreamByteLimit verifies exact-limit input succeeds and one extra byte is rejected.
func TestImageConstructionStreamByteLimit(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xFF, A: 0xFF})
	data := encodedImage(t, img)

	tests := []struct {
		name string
		fn   func(*XRefTable, io.Reader) error
	}{
		{
			name: "resources",
			fn: func(xRefTable *XRefTable, r io.Reader) error {
				_, err := CreateImageResources(xRefTable, r, false, false)
				return err
			},
		},
		{
			name: "stream dictionary",
			fn: func(xRefTable *XRefTable, r io.Reader) error {
				_, _, _, err := CreateImageStreamDict(xRefTable, r)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+" exact limit", func(t *testing.T) {
			xRefTable := imageConstructionXRefTable(false)
			xRefTable.Conf.Limits.MaxStreamBytes = int64(len(data))
			if err := tt.fn(xRefTable, bytes.NewReader(data)); err != nil {
				t.Fatalf("exact-limit image rejected: %v", err)
			}
		})
		t.Run(tt.name+" limit plus one", func(t *testing.T) {
			xRefTable := imageConstructionXRefTable(false)
			limit := int64(len(data) - 1)
			xRefTable.Conf.Limits.MaxStreamBytes = limit
			err := tt.fn(xRefTable, bytes.NewReader(data))
			want := fmt.Sprintf("read image: input size %d exceeds limit %d", len(data), limit)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %v", want, err)
			}
		})
	}
}

// TestCreateImageStreamDictDecodeErrorContext verifies decode configuration failures preserve their cause.
func TestCreateImageStreamDictDecodeErrorContext(t *testing.T) {
	_, _, _, err := CreateImageStreamDict(nil, bytes.NewReader(nil))
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	if !strings.Contains(err.Error(), "decode image configuration") {
		t.Fatalf("expected image decode context, got %q", err)
	}
}

// TestCreateImageResourcesDecodeErrorContext verifies resource decoding failures identify their phase.
func TestCreateImageResourcesDecodeErrorContext(t *testing.T) {
	_, err := CreateImageResources(nil, bytes.NewReader(nil), false, false)
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	if !strings.Contains(err.Error(), "decode image configuration") {
		t.Fatalf("expected image decode context, got %q", err)
	}
}

// TestCreateImageStreamDictSoftMaskObjectErrorContext verifies the complete soft-mask construction chain.
func TestCreateImageStreamDictSoftMaskObjectErrorContext(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xFF, A: 0x80})

	_, _, _, err := CreateImageStreamDict(
		imageConstructionXRefTable(true),
		bytes.NewReader(encodedImage(t, img)),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		"create image stream",
		"encode image stream",
		"create soft mask",
		"create object",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

// TestCreateImageResourceObjectErrorContext verifies main image xref insertion context.
func TestCreateImageResourceObjectErrorContext(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xFF, A: 0xFF})

	_, _, _, err := CreateImageResource(
		imageConstructionXRefTable(true),
		bytes.NewReader(encodedImage(t, img)),
	)
	if err == nil || !strings.Contains(err.Error(), "create image object") {
		t.Fatalf("expected image object context, got %v", err)
	}
}

// TestImageConstructionRejectsMissingXRefTable verifies soft masks and resources do not panic without xref state.
func TestImageConstructionRejectsMissingXRefTable(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xFF, A: 0x80})
	data := encodedImage(t, img)

	_, _, _, err := CreateImageStreamDict(nil, bytes.NewReader(data))
	if !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
	if !strings.Contains(err.Error(), "create soft mask") {
		t.Fatalf("expected soft-mask context, got %q", err)
	}

	_, _, _, err = CreateImageResource(nil, bytes.NewReader(data))
	if !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}

	_, err = CreateImageResources(nil, bytes.NewReader(data), false, false)
	if !errors.Is(err, ErrMissingXRefTable) {
		t.Fatalf("expected %v, got %v", ErrMissingXRefTable, err)
	}
}

// TestCreatePalettedImageStreamDictRejectsInvalidPaletteIndex verifies malformed palette data does not panic.
func TestCreatePalettedImageStreamDictRejectsInvalidPaletteIndex(t *testing.T) {
	img := &image.Paletted{
		Pix:     []byte{1},
		Stride:  1,
		Rect:    image.Rect(0, 0, 1, 1),
		Palette: color.Palette{color.Black},
	}
	_, err := createPalettedImageStreamDict(imageConstructionXRefTable(false), img)
	if err == nil || !strings.Contains(err.Error(), "paletted image: palette index 1 at (0,0) exceeds palette length 1") {
		t.Fatalf("expected palette index context, got %v", err)
	}
}

// TestCreatePalettedImageStreamDictRejectsMissingPalette verifies empty palette handling.
func TestCreatePalettedImageStreamDictRejectsMissingPalette(t *testing.T) {
	img := &image.Paletted{Pix: []byte{0}, Stride: 1, Rect: image.Rect(0, 0, 1, 1)}
	_, err := createPalettedImageStreamDict(imageConstructionXRefTable(false), img)
	if err == nil || !strings.Contains(err.Error(), "paletted image: missing palette") {
		t.Fatalf("expected missing palette context, got %v", err)
	}
}

// TestCreateImageStreamDictWithColorSpaceRejectsInvalidJPEGColorSpace verifies the internal invariant is explicit.
func TestCreateImageStreamDictWithColorSpaceRejectsInvalidJPEGColorSpace(t *testing.T) {
	_, err := createImageStreamDictWithColorSpace(nil, nil, nil, 1, 1, 8, "jpeg", types.Array{})
	if err == nil || !strings.Contains(err.Error(), "JPEG image stream: expected name color space, got types.Array") {
		t.Fatalf("expected JPEG color-space context, got %v", err)
	}
}

// TestHandleRGBImagePreservesRGBA64Alpha verifies 16-bit premultiplied alpha produces a soft mask.
func TestHandleRGBImagePreservesRGBA64Alpha(t *testing.T) {
	img := image.NewRGBA64(image.Rect(0, 0, 1, 1))
	img.SetRGBA64(0, 0, color.RGBA64{R: 0x1234, G: 0x2345, B: 0x3456, A: 0x4567})

	buf, softMask, bpc, cs, err := handleRGBImage(imageConstructionXRefTable(false), img)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != 6 || len(softMask) != 2 || bpc != 16 || cs != DeviceRGBCS {
		t.Fatalf("unexpected RGBA64 result: buf=%d softMask=%d bpc=%d cs=%s", len(buf), len(softMask), bpc, cs)
	}
	if softMask[0] != 0x45 || softMask[1] != 0x67 {
		t.Fatalf("expected alpha 0x4567, got %x", softMask)
	}
}

// TestImageBufferHandlersRejectUnexpectedTypes verifies internal type invariants are explicit.
func TestImageBufferHandlersRejectUnexpectedTypes(t *testing.T) {
	img := image.NewUniform(color.Black)
	if _, _, _, _, err := handleRGBImage(nil, img); err == nil || !strings.Contains(err.Error(), "unexpected RGB image type") {
		t.Fatalf("expected RGB type error, got %v", err)
	}
	if _, _, _, _, err := handleGrayImage(nil, img, nil); err == nil || !strings.Contains(err.Error(), "unexpected gray image type") {
		t.Fatalf("expected gray type error, got %v", err)
	}
}

// TestValidateImageResourceLimitsAddsOverflowContext verifies arithmetic failures identify their phase.
func TestValidateImageResourceLimitsAddsOverflowContext(t *testing.T) {
	conf := NewDefaultConfiguration()
	conf.Limits.MaxImagePixels = math.MaxInt64
	conf.Limits.MaxImageBytes = math.MaxInt64
	xRefTable := newXRefTable(conf)

	err := validateImageResourceLimits(xRefTable, image.Config{Width: math.MaxInt32, Height: math.MaxInt32})
	if err == nil || !strings.Contains(err.Error(), "image render byte size") {
		t.Fatalf("expected render-size overflow context, got %v", err)
	}
}

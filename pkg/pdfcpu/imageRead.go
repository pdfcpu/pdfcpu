/*
Copyright 2018 The pdfcpu Authors.

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
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pkg/errors"
)

func createSMaskObject(xRefTable *XRefTable, buf []byte, w, h int) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("XObject"),
				"Subtype":          Name("Image"),
				"BitsPerComponent": Integer(8),
				"ColorSpace":       Name(DeviceGrayCS),
				"Width":            Integer(w),
				"Height":           Integer(h),
			},
		),
		Content:        buf,
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}}

	sd.InsertName("Filter", filter.Flate)

	if err := encodeStream(sd); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createFlateImageObject(xRefTable *XRefTable, buf, sm []byte, w, h, bpc int, cs string) (*StreamDict, error) {

	var softMaskIndRef *IndirectRef

	if sm != nil {
		var err error
		softMaskIndRef, err = createSMaskObject(xRefTable, sm, w, h)
		if err != nil {
			return nil, err
		}
	}

	sd, _ := xRefTable.NewStreamDictForBuf(buf)
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Image")
	sd.InsertInt("Width", w)
	sd.InsertInt("Height", h)
	sd.InsertInt("BitsPerComponent", bpc)
	sd.InsertName("ColorSpace", cs)

	if softMaskIndRef != nil {
		sd.Insert("SMask", *softMaskIndRef)
	}

	if w < 1000 || h < 1000 {
		sd.Insert("Interpolate", Boolean(true))
	}

	if err := encodeStream(sd); err != nil {
		return nil, err
	}

	return sd, nil
}

func createDCTImageObject(xRefTable *XRefTable, buf, sm []byte, w, h int, cs string) (*StreamDict, error) {

	var softMaskIndRef *IndirectRef

	if sm != nil {
		var err error
		softMaskIndRef, err = createSMaskObject(xRefTable, sm, w, h)
		if err != nil {
			return nil, err
		}
	}

	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("XObject"),
				"Subtype":          Name("Image"),
				"Width":            Integer(w),
				"Height":           Integer(h),
				"BitsPerComponent": Integer(8),
				"ColorSpace":       Name(cs),
			},
		),
		Content:        buf,
		FilterPipeline: nil,
	}

	if cs == DeviceCMYKCS {
		sd.Insert("Decode", NewIntegerArray(1, 0, 1, 0, 1, 0, 1, 0))
	}

	if w < 1000 || h < 1000 {
		sd.Insert("Interpolate", Boolean(true))
	}

	sd.InsertName("Filter", filter.DCT)

	if softMaskIndRef != nil {
		sd.Insert("SMask", *softMaskIndRef)
	}

	if err := encodeStream(sd); err != nil {
		return nil, err
	}

	sd.FilterPipeline = []PDFFilter{{Name: filter.DCT, DecodeParms: nil}}

	return sd, nil
}

func writeRGBAImageBuf(img image.Image) []byte {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h*3)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.RGBA)
			buf[i] = c.R
			buf[i+1] = c.G
			buf[i+2] = c.B
			i += 3
		}
	}

	return buf
}

func writeRGBA64ImageBuf(img image.Image) []byte {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h*6)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.RGBA64)
			buf[i] = uint8(c.R >> 8)
			buf[i+1] = uint8(c.R & 0x00FF)
			buf[i+2] = uint8(c.G >> 8)
			buf[i+3] = uint8(c.G & 0x00FF)
			buf[i+4] = uint8(c.B >> 8)
			buf[i+5] = uint8(c.B & 0x00FF)
			i += 6
		}
	}

	return buf
}

func writeYCbCrToRGBAImageBuf(img image.Image) []byte {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h*3)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.YCbCr)
			r, g, b, _ := c.RGBA()
			buf[i] = uint8(r >> 8 & 0xFF)
			buf[i+1] = uint8(g >> 8 & 0xFF)
			buf[i+2] = uint8(b >> 8 & 0xFF)
			i += 3
		}
	}

	return buf
}
func writeNRGBAImageBuf(xRefTable *XRefTable, img image.Image) ([]byte, []byte) {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h*3)
	var sm []byte
	var softMask bool

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.NRGBA)
			if !softMask {
				if xRefTable != nil && c.A != 0xFF {
					softMask = true
					sm = []byte{}
					for index := 0; index < y*h+x; index++ {
						sm = append(sm, 0xFF)
					}
					sm = append(sm, c.A)
				}
			} else {
				sm = append(sm, c.A)
			}

			buf[i] = c.R
			buf[i+1] = c.G
			buf[i+2] = c.B
			i += 3
		}
	}

	return buf, sm
}

func writeGrayImageBuf(img image.Image) []byte {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.Gray)
			buf[i] = c.Y
			i++
		}
	}

	return buf
}

func writeCMYKImageBuf(img image.Image) []byte {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h*4)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.CMYK)
			buf[i] = c.C
			buf[i+1] = c.M
			buf[i+2] = c.Y
			buf[i+3] = c.K
			i += 4
			//fmt.Printf("x:%3d(%3d) y:%3d(%3d) c:#%02x m:#%02x y:#%02x k:#%02x\n", x1, x, y1, y, c.C, c.M, c.Y, c.K)
		}
	}

	return buf
}

func imgToImageDict(xRefTable *XRefTable, img image.Image) (*StreamDict, error) {

	bpc := 8

	// TODO if dpi != 72 resample (applies to PNG,JPG,TIFF)

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	var buf []byte
	var sm []byte
	var cs string

	switch img.(type) {
	case *image.RGBA:
		// A 32-bit alpha-premultiplied color, having 8 bits for each of red, green, blue and alpha.
		// An alpha-premultiplied color component C has been scaled by alpha (A), so it has valid values 0 <= C <= A.
		cs = DeviceRGBCS
		buf = writeRGBAImageBuf(img)

	case *image.RGBA64:
		// A 64-bit alpha-premultiplied color, having 16 bits for each of red, green, blue and alpha.
		// An alpha-premultiplied color component C has been scaled by alpha (A), so it has valid values 0 <= C <= A.
		cs = DeviceRGBCS
		bpc = 16
		buf = writeRGBA64ImageBuf(img)

	case *image.NRGBA:
		// Non-alpha-premultiplied 32-bit color.
		cs = DeviceRGBCS
		buf, sm = writeNRGBAImageBuf(xRefTable, img)

	case *image.NRGBA64:
		return nil, errors.New("unsupported image type: NRGBA66")

	case *image.Alpha:
		return nil, errors.New("unsupported image type: Alpha")

	case *image.Alpha16:
		return nil, errors.New("unsupported image type: Alpha16")

	case *image.Gray:
		// 8-bit grayscale color.
		cs = DeviceGrayCS
		buf = writeGrayImageBuf(img)

	case *image.Gray16:
		return nil, errors.New("unsupported image type: Gray16")

	case *image.CMYK:
		// Opaque CMYK color, having 8 bits for each of cyan, magenta, yellow and black.
		cs = DeviceCMYKCS
		buf = writeCMYKImageBuf(img)

	case *image.YCbCr:
		return nil, errors.New("unsupported image type: YCbCr")

	case *image.NYCbCrA:
		return nil, errors.New("unsupported image type: NYCbCrA")

	case *image.Paletted:
		// uint8 indices into a given color palette.
		cs = DeviceRGBCS
		buf = writeRGBAImageBuf(img)

	default:
		return nil, errors.Errorf("unsupported image type: %T", img)
	}

	//fmt.Printf("old w:%3d, h:%3d, new w:%3d, h:%3d\n", img.Bounds().Dx(), img.Bounds().Dy(), w, h)

	return createFlateImageObject(xRefTable, buf, sm, w, h, bpc, cs)
}

// ReadJPEG generates a PDF image object for a JPEG stream
// and appends this object to the cross reference table.
func ReadJPEG(xRefTable *XRefTable, buf []byte, c image.Config) (*StreamDict, error) {

	var cs string

	switch c.ColorModel {

	case color.GrayModel:
		cs = DeviceGrayCS

	case color.YCbCrModel:
		cs = DeviceRGBCS

	case color.CMYKModel:
		cs = DeviceCMYKCS

	default:
		return nil, errors.New("pdfcpu: unexpected color model for JPEG")

	}

	return createDCTImageObject(xRefTable, buf, nil, c.Width, c.Height, cs)
}

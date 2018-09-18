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
	"image/png"
	"os"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/tiff"
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

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createImageObject(xRefTable *XRefTable, buf, sm []byte, w, h int, cs string) (*StreamDict, error) {

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
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}}

	sd.InsertName("Filter", filter.Flate)

	if softMaskIndRef != nil {
		sd.Insert("SMask", *softMaskIndRef)
	}

	err := encodeStream(sd)
	if err != nil {
		return nil, err
	}

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
			//fmt.Printf("x:%3d y:%3d r:#%02x g:#%02x b:#%02x a:#%02x\n", x, y, c.R, c.G, c.B, c.A)
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
			//fmt.Printf("x:%3d y:%3d r:#%02x g:#%02x b:#%02x a:#%02x\n", x, y, c.R, c.G, c.B, c.A)
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

	// Supporting 8 bits per component.

	// TODO if dpi != 72 resample (applies to PNG,JPG,TIFF)

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	var buf []byte
	var sm []byte
	var cs string

	switch img.ColorModel() {

	case color.RGBAModel:
		// Represents a traditional 32-bit alpha-premultiplied color, having 8 bits for each of red, green, blue and alpha.
		//fmt.Println("RGBA")
		cs = DeviceRGBCS
		buf = writeRGBAImageBuf(img)

	case color.RGBA64Model:
		//fmt.Println("RGBA64")
		return nil, ErrUnsupportedColorSpace

	case color.NRGBAModel:
		// Non-alpha-premultiplied 32-bit color.
		//fmt.Println("NRGBA")
		cs = DeviceRGBCS
		buf, sm = writeNRGBAImageBuf(xRefTable, img)

	case color.NRGBA64Model:
		//fmt.Println("NRGBA64")
		return nil, ErrUnsupportedColorSpace

	case color.AlphaModel:
		//fmt.Println("Alpha")
		return nil, ErrUnsupportedColorSpace

	case color.Alpha16Model:
		//fmt.Println("Alpha16")
		return nil, ErrUnsupportedColorSpace

	case color.GrayModel:
		//fmt.Println("Gray")
		cs = DeviceGrayCS
		buf = writeGrayImageBuf(img)

	case color.Gray16Model:
		//fmt.Println("Gray16")
		return nil, ErrUnsupportedColorSpace

	case color.CMYKModel:
		//fmt.Println("CMYK")
		cs = DeviceCMYKCS
		buf = writeCMYKImageBuf(img)

	default:
		//fmt.Println("unknown color model")
		return nil, ErrUnsupportedColorSpace

	}

	//fmt.Printf("old w:%3d, h:%3d, new w:%3d, h:%3d\n", img.Bounds().Dx(), img.Bounds().Dy(), w, h)

	return createImageObject(xRefTable, buf, sm, w, h, cs)
}

// ReadPNGFile generates a PDF image object for a PNG file
// and appends this object to the cross reference table.
func ReadPNGFile(xRefTable *XRefTable, fileName string) (*StreamDict, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	return imgToImageDict(xRefTable, img)
}

// ReadTIFFFile generates a PDF image object for a TIFF file
// and appends this object to the cross reference table.
func ReadTIFFFile(xRefTable *XRefTable, fileName string) (*StreamDict, error) {

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := tiff.Decode(f)
	if err != nil {
		return nil, err
	}

	return imgToImageDict(xRefTable, img)
}

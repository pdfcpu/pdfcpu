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
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pkg/errors"
	_ "golang.org/x/image/webp"
)

// ImageFileName returns true for supported image file types.
func ImageFileName(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return MemberOf(ext, []string{".png", ".webp", ".tif", ".tiff", ".jpg", ".jpeg"})
}

// ImageFileNames returns a slice of image file names contained in dir.
func ImageFileNames(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fn := []string{}
	for _, fi := range files {
		if ImageFileName(fi.Name()) {
			fn = append(fn, filepath.Join(dir, fi.Name()))
		}
	}
	return fn, nil
}

func createSMaskObject(xRefTable *XRefTable, buf []byte, w, h, bpc int) (*IndirectRef, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("XObject"),
				"Subtype":          Name("Image"),
				"BitsPerComponent": Integer(bpc),
				"ColorSpace":       Name(DeviceGrayCS),
				"Width":            Integer(w),
				"Height":           Integer(h),
			},
		),
		Content:        buf,
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func createFlateImageObject(xRefTable *XRefTable, buf, sm []byte, w, h, bpc int, cs string) (*StreamDict, error) {
	var softMaskIndRef *IndirectRef
	if sm != nil {
		var err error
		softMaskIndRef, err = createSMaskObject(xRefTable, sm, w, h, bpc)
		if err != nil {
			return nil, err
		}
	}

	// Create Flate stream dict.
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

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return sd, nil
}

func createDCTImageObject(xRefTable *XRefTable, buf []byte, w, h, bpc int, cs string) (*StreamDict, error) {
	sd := &StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":             Name("XObject"),
				"Subtype":          Name("Image"),
				"Width":            Integer(w),
				"Height":           Integer(h),
				"BitsPerComponent": Integer(bpc),
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

	// Calling Encode without FilterPipeline ensures an encoded stream in sd.Raw.
	if err := sd.Encode(); err != nil {
		return nil, err
	}

	sd.Content = nil

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
					for j := 0; j < y*h+x; j++ {
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

func writeNRGBA64ImageBuf(xRefTable *XRefTable, img image.Image) ([]byte, []byte) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, w*h*6)
	var sm []byte
	var softMask bool

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.NRGBA64)
			if !softMask {
				if xRefTable != nil && c.A != 0xFFFF {
					softMask = true
					sm = []byte{}
					for j := 0; j < y*h+x; j++ {
						sm = append(sm, 0xFF)
						sm = append(sm, 0xFF)
					}
					sm = append(sm, uint8(c.A>>8))
					sm = append(sm, uint8(c.A&0x00FF))
				}
			} else {
				sm = append(sm, uint8(c.A>>8))
				sm = append(sm, uint8(c.A&0x00FF))
			}

			buf[i] = uint8(c.R >> 8)
			buf[i+1] = uint8(c.R & 0x00FF)
			buf[i+2] = uint8(c.G >> 8)
			buf[i+3] = uint8(c.G & 0x00FF)
			buf[i+4] = uint8(c.B >> 8)
			buf[i+5] = uint8(c.B & 0x00FF)
			i += 6
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

func writeGray16ImageBuf(img image.Image) []byte {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	buf := make([]byte, 2*w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.Gray16)
			buf[i] = uint8(c.Y >> 8)
			buf[i+1] = uint8(c.Y & 0x00FF)
			i += 2
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

func convertToRGBA(img image.Image) *image.RGBA {
	b := img.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m
}

func convertToGray(img image.Image) *image.Gray {
	b := img.Bounds()
	m := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m
}

func convertToSepia(img image.Image) *image.RGBA {
	m := convertToRGBA(img)
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := m.At(x, y).(color.RGBA)
			r := math.Round((float64(c.R) * .393) + (float64(c.G) * .769) + (float64(c.B) * .189))
			if r > 255 {
				r = 255
			}
			g := math.Round((float64(c.R) * .349) + (float64(c.G) * .686) + (float64(c.B) * .168))
			if g > 255 {
				g = 255
			}
			b := math.Round((float64(c.R) * .272) + (float64(c.G) * .534) + (float64(c.B) * .131))
			if b > 255 {
				b = 255
			}
			m.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), c.A})
		}
	}
	return m
}

func createImageDict(xRefTable *XRefTable, buf, softMask []byte, w, h, bpc int, format, cs string) (*StreamDict, int, int, error) {
	var (
		sd  *StreamDict
		err error
	)
	switch format {
	case "jpeg":
		sd, err = createDCTImageObject(xRefTable, buf, w, h, bpc, cs)
	default:
		sd, err = createFlateImageObject(xRefTable, buf, softMask, w, h, bpc, cs)
	}
	return sd, w, h, err
}

func encodeJPEG(img image.Image) ([]byte, string, error) {
	var cs string
	switch img.(type) {
	case *image.Gray, *image.Gray16:
		cs = DeviceGrayCS
	case *image.YCbCr:
		cs = DeviceRGBCS
	case *image.CMYK:
		cs = DeviceCMYKCS
	default:
		return nil, "", errors.Errorf("pdfcpu: unexpected color model for JPEG: %s", cs)
	}
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	return buf.Bytes(), cs, err
}

func createImageBuf(xRefTable *XRefTable, img image.Image, format string) ([]byte, []byte, int, string, error) {
	var buf []byte
	var sm []byte // soft mask aka alpha mask
	var bpc int
	// TODO if dpi != 72 resample (applies to PNG,JPG,TIFF)

	if format == "jpeg" {
		bb, cs, err := encodeJPEG(img)
		return bb, sm, 8, cs, err
	}

	var cs string

	switch img.(type) {
	case *image.RGBA:
		// A 32-bit alpha-premultiplied color, having 8 bits for each of red, green, blue and alpha.
		// An alpha-premultiplied color component C has been scaled by alpha (A), so it has valid values 0 <= C <= A.
		cs = DeviceRGBCS
		bpc = 8
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
		bpc = 8
		buf, sm = writeNRGBAImageBuf(xRefTable, img)

	case *image.NRGBA64:
		// Non-alpha-premultiplied 64-bit color.
		cs = DeviceRGBCS
		bpc = 16
		buf, sm = writeNRGBA64ImageBuf(xRefTable, img)

	case *image.Alpha:
		return buf, sm, bpc, cs, errors.New("pdfcpu: unsupported image type: Alpha")

	case *image.Alpha16:
		return buf, sm, bpc, cs, errors.New("pdfcpu: unsupported image type: Alpha16")

	case *image.Gray:
		// 8-bit grayscale color.
		cs = DeviceGrayCS
		bpc = 8
		buf = writeGrayImageBuf(img)

	case *image.Gray16:
		// 16-bit grayscale color.
		cs = DeviceGrayCS
		bpc = 16
		buf = writeGray16ImageBuf(img)

	case *image.CMYK:
		// Opaque CMYK color, having 8 bits for each of cyan, magenta, yellow and black.
		cs = DeviceCMYKCS
		bpc = 8
		buf = writeCMYKImageBuf(img)

	case *image.YCbCr:
		cs = DeviceRGBCS
		bpc = 8
		buf = writeRGBAImageBuf(convertToRGBA(img))

	case *image.NYCbCrA:
		return buf, sm, bpc, cs, errors.New("pdfcpu: unsupported image type: NYCbCrA")

	case *image.Paletted:
		// In-memory image of uint8 indices into a given palette.
		cs = DeviceRGBCS
		bpc = 8
		buf = writeRGBAImageBuf(convertToRGBA(img))

	default:
		return buf, sm, bpc, cs, errors.Errorf("pdfcpu: unsupported image type: %T", img)
	}

	return buf, sm, bpc, cs, nil
}

func colorSpaceForJPEGColorModel(cm color.Model) string {
	switch cm {
	case color.GrayModel:
		return DeviceGrayCS
	case color.YCbCrModel:
		return DeviceRGBCS
	case color.CMYKModel:
		return DeviceCMYKCS
	}
	return ""
}

func createDCTImageObjectForJPEG(xRefTable *XRefTable, c image.Config, bb bytes.Buffer) (*StreamDict, int, int, error) {
	cs := colorSpaceForJPEGColorModel(c.ColorModel)
	if cs == "" {
		return nil, 0, 0, errors.New("pdfcpu: unexpected color model for JPEG")
	}

	buf, err := ioutil.ReadAll(&bb)
	if err != nil {
		return nil, 0, 0, err
	}

	sd, err := createDCTImageObject(xRefTable, buf, c.Width, c.Height, 8, cs)

	return sd, c.Width, c.Height, err
}

func CreateImageStreamDict(xRefTable *XRefTable, r io.Reader, gray, sepia bool) (*StreamDict, int, int, error) {

	var bb bytes.Buffer
	tee := io.TeeReader(r, &bb)
	sniff, err := ioutil.ReadAll(tee)
	if err != nil {
		return nil, 0, 0, err
	}

	c, format, err := image.DecodeConfig(bytes.NewBuffer(sniff))
	if err != nil {
		return nil, 0, 0, err
	}

	if format == "jpeg" && !gray && !sepia {
		return createDCTImageObjectForJPEG(xRefTable, c, bb)
	}

	img, format, err := image.Decode(&bb)
	if err != nil {
		return nil, 0, 0, err
	}

	if gray {
		switch img.(type) {
		case *image.Gray, *image.Gray16:
		default:
			img = convertToGray(img)
		}
	}

	if sepia {
		switch img.(type) {
		case *image.Gray, *image.Gray16:
		default:
			img = convertToSepia(img)
		}
	}

	buf, softMask, bpc, cs, err := createImageBuf(xRefTable, img, format)
	if err != nil {
		return nil, 0, 0, err
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	return createImageDict(xRefTable, buf, softMask, w, h, bpc, format, cs)
}

func CreateImageResource(xRefTable *XRefTable, r io.Reader, gray, sepia bool) (*IndirectRef, int, int, error) {
	sd, w, h, err := CreateImageStreamDict(xRefTable, r, gray, sepia)
	if err != nil {
		return nil, 0, 0, err
	}
	indRef, err := xRefTable.IndRefForNewObject(*sd)
	return indRef, w, h, err
}

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

package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"

	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hhrutter/tiff"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
	_ "golang.org/x/image/webp"
)

// Image is a Reader representing an image resource.
type Image struct {
	io.Reader
	Name        string // Resource name
	FileType    string
	PageNr      int
	ObjNr       int
	Width       int    // "Width"
	Height      int    // "Height"
	Bpc         int    // "BitsPerComponent"
	Cs          string // "ColorSpace"
	Comp        int    // color component count
	IsImgMask   bool   // "ImageMask"
	HasImgMask  bool   // "Mask"
	HasSMask    bool   // "SMask"
	Thumb       bool   // "Thumbnail"
	Interpol    bool   // "Interpolate"
	Size        int64  // "Length"
	Filter      string // filter pipeline
	DecodeParms string
}

// ImageFileName returns true for supported image file types.
func ImageFileName(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return types.MemberOf(ext, []string{".png", ".webp", ".tif", ".tiff", ".jpg", ".jpeg"})
}

// ImageFileNames returns a slice of image file names contained in dir constrained by maxFileSize.
func ImageFileNames(dir string, maxFileSize types.ByteSize) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fn := []string{}
	for i := 0; i < len(files); i++ {
		fi := files[i]
		fileInfo, err := fi.Info()
		if err != nil {
			continue
		}
		if types.ByteSize(fileInfo.Size()) > maxFileSize {
			continue
		}
		if ImageFileName(fi.Name()) {
			fn = append(fn, filepath.Join(dir, fi.Name()))
		}
	}
	return fn, nil
}

func createSMaskObject(xRefTable *XRefTable, buf []byte, w, h, bpc int) (*types.IndirectRef, error) {
	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":             types.Name("XObject"),
				"Subtype":          types.Name("Image"),
				"BitsPerComponent": types.Integer(bpc),
				"ColorSpace":       types.Name(DeviceGrayCS),
				"Width":            types.Integer(w),
				"Height":           types.Integer(h),
			},
		),
		Content:        buf,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

// CreateFlateImageStreamDict returns a flate stream dict.
func CreateFlateImageStreamDict(xRefTable *XRefTable, buf, sm []byte, w, h, bpc int, cs string) (*types.StreamDict, error) {
	var softMaskIndRef *types.IndirectRef
	if sm != nil {
		var err error
		softMaskIndRef, err = createSMaskObject(xRefTable, sm, w, h, bpc)
		if err != nil {
			return nil, err
		}
	}

	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":             types.Name("XObject"),
				"Subtype":          types.Name("Image"),
				"Width":            types.Integer(w),
				"Height":           types.Integer(h),
				"BitsPerComponent": types.Integer(bpc),
				"ColorSpace":       types.Name(cs),
			},
		),
		Content:        buf,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if softMaskIndRef != nil {
		sd.Insert("SMask", *softMaskIndRef)
	}

	if w < 1000 || h < 1000 {
		sd.Insert("Interpolate", types.Boolean(true))
	}

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return sd, nil
}

// CreateDCTImageStreamDict returns a DCT encoded stream dict.
func CreateDCTImageStreamDict(xRefTable *XRefTable, buf []byte, w, h, bpc int, cs string) (*types.StreamDict, error) {
	sd := &types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":             types.Name("XObject"),
				"Subtype":          types.Name("Image"),
				"Width":            types.Integer(w),
				"Height":           types.Integer(h),
				"BitsPerComponent": types.Integer(bpc),
				"ColorSpace":       types.Name(cs),
			},
		),
		Content:        buf,
		FilterPipeline: nil,
	}

	if cs == DeviceCMYKCS {
		sd.Insert("Decode", types.NewIntegerArray(1, 0, 1, 0, 1, 0, 1, 0))
	}

	if w < 1000 || h < 1000 {
		sd.Insert("Interpolate", types.Boolean(true))
	}

	sd.InsertName("Filter", filter.DCT)

	// Calling Encode without FilterPipeline ensures an encoded stream in sd.Raw.
	if err := sd.Encode(); err != nil {
		return nil, err
	}

	sd.Content = nil

	sd.FilterPipeline = []types.PDFFilter{{Name: filter.DCT, DecodeParms: nil}}

	return sd, nil
}

func writeRGBAImageBuf(img image.Image) ([]byte, []byte) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	i := 0
	var sm []byte
	buf := make([]byte, w*h*3)
	var softMask bool

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(x, y).(color.RGBA)
			if !softMask {
				if c.A != 0xFF {
					softMask = true
					sm = []byte{}
					for j := 0; j < y*w+x; j++ {
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
					for j := 0; j < y*w+x; j++ {
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
					for j := 0; j < y*w+x; j++ {
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
	m := image.NewRGBA(b)
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m
}

func convertNYCbCrAToRGBA(img *image.NYCbCrA) *image.RGBA {
	b := img.Bounds()
	m := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			ycbr := img.YCbCrAt(x, y)
			stride := img.Bounds().Dx()
			alphaOffset := (y-b.Min.Y)*stride + (x - b.Min.X)
			alpha := img.A[alphaOffset]
			r, g, b := color.YCbCrToRGB(ycbr.Y, ycbr.Cb, ycbr.Cr)
			m.Set(x, y, color.RGBA{R: r, G: g, B: b, A: alpha})
		}
	}
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

func createImageStreamDict(xRefTable *XRefTable, buf, softMask []byte, w, h, bpc int, format, cs string) (*types.StreamDict, error) {
	var (
		sd  *types.StreamDict
		err error
	)
	switch format {
	case "jpeg":
		sd, err = CreateDCTImageStreamDict(xRefTable, buf, w, h, bpc, cs)
	default:
		sd, err = CreateFlateImageStreamDict(xRefTable, buf, softMask, w, h, bpc, cs)
	}
	return sd, err
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

	switch img := img.(type) {
	case *image.RGBA:
		// A 32-bit alpha-premultiplied color, having 8 bits for each of red, green, blue and alpha.
		// An alpha-premultiplied color component C has been scaled by alpha (A), so it has valid values 0 <= C <= A.
		cs = DeviceRGBCS
		bpc = 8
		buf, sm = writeRGBAImageBuf(img)

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
		buf, sm = writeRGBAImageBuf(convertToRGBA(img))

	case *image.NYCbCrA:
		cs = DeviceRGBCS
		bpc = 8
		buf, sm = writeRGBAImageBuf(convertNYCbCrAToRGBA(img))

	case *image.Paletted:
		// In-memory image of uint8 indices into a given palette.
		cs = DeviceRGBCS
		bpc = 8
		buf, sm = writeRGBAImageBuf(convertToRGBA(img))

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

func createDCTImageStreamDictForJPEG(xRefTable *XRefTable, c image.Config, bb bytes.Buffer) (*types.StreamDict, error) {
	cs := colorSpaceForJPEGColorModel(c.ColorModel)
	if cs == "" {
		return nil, errors.New("pdfcpu: unexpected color model for JPEG")
	}

	return CreateDCTImageStreamDict(xRefTable, bb.Bytes(), c.Width, c.Height, 8, cs)
}

func createImageResourcesForJPEG(xRefTable *XRefTable, c image.Config, bb bytes.Buffer) ([]ImageResource, error) {
	sd, err := createDCTImageStreamDictForJPEG(xRefTable, c, bb)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	res := Resource{ID: "Im0", IndRef: indRef}
	ir := ImageResource{Res: res, Width: c.Width, Height: c.Height}
	return []ImageResource{ir}, err
}

func decodeImage(xRefTable *XRefTable, buf *bytes.Reader, currentOffset int64, gray, sepia bool, byteOrder binary.ByteOrder, imgResources *[]ImageResource) (int64, error) {
	img, err := tiff.DecodeAt(buf, currentOffset)
	if err != nil {
		return 0, err
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

	imgBuf, softMask, bpc, cs, err := createImageBuf(xRefTable, img, "tiff")
	if err != nil {
		return 0, err
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	sd, err := createImageStreamDict(xRefTable, imgBuf, softMask, w, h, bpc, "tiff", cs)
	if err != nil {
		return 0, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return 0, err
	}

	res := Resource{ID: "Im0", IndRef: indRef}
	ir := ImageResource{Res: res, Width: w, Height: h}
	*imgResources = append(*imgResources, ir)

	if _, err := buf.Seek(currentOffset, io.SeekStart); err != nil {
		return 0, err
	}

	var numEntries uint16
	if err := binary.Read(buf, byteOrder, &numEntries); err != nil {
		return 0, err
	}

	if _, err := buf.Seek(int64(numEntries)*12, io.SeekCurrent); err != nil {
		return 0, err
	}

	var nextIFDOffset uint32
	if err := binary.Read(buf, byteOrder, &nextIFDOffset); err != nil {
		return 0, err
	}

	// if nextIFDOffset >= uint32(bb.Len()) {
	// 	fmt.Println("Invalid next IFD offset, stopping.")
	// 	break
	// }

	return int64(nextIFDOffset), nil
}

func createImageResourcesForTIFF(xRefTable *XRefTable, bb bytes.Buffer, gray, sepia bool) ([]ImageResource, error) {
	imgResources := []ImageResource{}

	buf := bytes.NewReader(bb.Bytes())

	var header [8]byte
	if _, err := io.ReadFull(buf, header[:]); err != nil {
		return nil, err
	}

	var byteOrder binary.ByteOrder
	if string(header[:2]) == "II" {
		byteOrder = binary.LittleEndian
	} else if string(header[:2]) == "MM" {
		byteOrder = binary.BigEndian
	} else {
		return nil, fmt.Errorf("invalid TIFF byte order")
	}

	firstIFDOffset := byteOrder.Uint32(header[4:])
	if firstIFDOffset < 8 || firstIFDOffset >= uint32(bb.Len()) {
		return nil, fmt.Errorf("invalid TIFF file: no valid IFD")
	}

	var err error

	off := int64(firstIFDOffset)

	for off != 0 && off < int64(bb.Len()) {
		off, err = decodeImage(xRefTable, buf, off, gray, sepia, byteOrder, &imgResources)
		if err != nil {
			return nil, err
		}
	}

	return imgResources, nil
}

func createImageResources(xRefTable *XRefTable, c image.Config, bb bytes.Buffer, gray, sepia bool) ([]ImageResource, error) {
	img, format, err := image.Decode(&bb)
	if err != nil {
		return nil, err
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

	imgBuf, softMask, bpc, cs, err := createImageBuf(xRefTable, img, format)
	if err != nil {
		return nil, err
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w != c.Width || h != c.Height {
		return nil, errors.New("pdfcpu: unexpected width or height")
	}

	sd, err := createImageStreamDict(xRefTable, imgBuf, softMask, w, h, bpc, format, cs)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	res := Resource{ID: "Im0", IndRef: indRef}
	ir := ImageResource{Res: res, Width: w, Height: h}
	return []ImageResource{ir}, err
}

// CreateImageResources creates a new XObject for given image data represented by r and applies optional filters.
func CreateImageResources(xRefTable *XRefTable, r io.Reader, gray, sepia bool) ([]ImageResource, error) {

	var bb bytes.Buffer
	tee := io.TeeReader(r, &bb)

	var sniff bytes.Buffer
	if _, err := io.Copy(&sniff, tee); err != nil {
		return nil, err
	}

	c, format, err := image.DecodeConfig(&sniff)
	if err != nil {
		return nil, err
	}

	if format == "tiff" {
		return createImageResourcesForTIFF(xRefTable, bb, gray, sepia)
	}

	if format == "jpeg" && !gray && !sepia {
		return createImageResourcesForJPEG(xRefTable, c, bb)
	}

	return createImageResources(xRefTable, c, bb, gray, sepia)
}

// CreateImageStreamDict returns a stream dict for image data represented by r and applies optional filters.
func CreateImageStreamDict(xRefTable *XRefTable, r io.Reader) (*types.StreamDict, int, int, error) {

	var bb bytes.Buffer
	tee := io.TeeReader(r, &bb)

	var sniff bytes.Buffer
	if _, err := io.Copy(&sniff, tee); err != nil {
		return nil, 0, 0, err
	}

	c, format, err := image.DecodeConfig(&sniff)
	if err != nil {
		return nil, 0, 0, err
	}

	if format == "jpeg" {
		sd, err := createDCTImageStreamDictForJPEG(xRefTable, c, bb)
		if err != nil {
			return nil, 0, 0, err
		}
		return sd, c.Width, c.Height, nil
	}

	img, format, err := image.Decode(&bb)
	if err != nil {
		return nil, 0, 0, err
	}

	imgBuf, softMask, bpc, cs, err := createImageBuf(xRefTable, img, format)
	if err != nil {
		return nil, 0, 0, err
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w != c.Width || h != c.Height {
		return nil, 0, 0, errors.New("pdfcpu: unexpected width or height")
	}

	sd, err := createImageStreamDict(xRefTable, imgBuf, softMask, w, h, bpc, format, cs)
	if err != nil {
		return nil, 0, 0, err
	}
	return sd, c.Width, c.Height, nil
}

// CreateImageResource creates a new XObject for given image data represented by r and applies optional filters.
func CreateImageResource(xRefTable *XRefTable, r io.Reader) (*types.IndirectRef, int, int, error) {
	sd, w, h, err := CreateImageStreamDict(xRefTable, r)
	if err != nil {
		return nil, 0, 0, err
	}
	indRef, err := xRefTable.IndRefForNewObject(*sd)
	return indRef, w, h, err
}

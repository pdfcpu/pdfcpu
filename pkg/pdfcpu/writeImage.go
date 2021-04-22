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
	"image/png"
	"io"
	"os"

	"github.com/hhrutter/tiff"
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// Errors to be identified.
var (
	ErrUnsupported16BPC = errors.New("unsupported 16 bits per component")
)

// colValRange defines a numeric range for color space component values that may be inverted.
type colValRange struct {
	min, max float64
	inv      bool
}

// PDFImage represents a XObject of subtype image.
type PDFImage struct {
	objNr    int
	sd       *StreamDict
	bpc      int
	w, h     int
	softMask []byte
	decode   []colValRange
}

func decodeArr(a Array) []colValRange {

	if a == nil {
		//println("decodearr == nil")
		return nil
	}

	var decode []colValRange
	var min, max, f64 float64

	for i, f := range a {
		switch o := f.(type) {
		case Integer:
			f64 = float64(o.Value())
		case Float:
			f64 = o.Value()
		}
		if i%2 == 0 {
			min = f64
			continue
		}
		max = f64
		var inv bool
		if min > max {
			min, max = max, min
			inv = true
		}
		decode = append(decode, colValRange{min: min, max: max, inv: inv})
	}

	return decode
}

func pdfImage(xRefTable *XRefTable, sd *StreamDict, objNr int) (*PDFImage, error) {

	bpc := *sd.IntEntry("BitsPerComponent")
	//if bpc == 16 {
	//	return nil, ErrUnsupported16BPC
	//}

	w := *sd.IntEntry("Width")
	h := *sd.IntEntry("Height")

	decode := decodeArr(sd.ArrayEntry("Decode"))
	//fmt.Printf("decode: %v\n", decode)

	sm, err := softMask(xRefTable, sd, w, h, objNr)
	if err != nil {
		return nil, err
	}

	return &PDFImage{
		objNr:    objNr,
		sd:       sd,
		bpc:      bpc,
		w:        w,
		h:        h,
		softMask: sm,
		decode:   decode,
	}, nil
}

// Identify the color lookup table for an Indexed color space.
func colorLookupTable(xRefTable *XRefTable, o Object) ([]byte, error) {

	o, _ = xRefTable.Dereference(o)

	switch o := o.(type) {

	case StringLiteral:
		return []byte(o), nil

	case HexLiteral:
		return o.Bytes()

	case StreamDict:
		return streamBytes(&o)

	}

	return nil, nil
}

func decodePixelColorValue(p uint8, bpc, c int, decode []colValRange) uint8 {

	// p ...the color value for this pixel
	// c ...applicable index of a color component in the decode array for this pixel.

	if decode == nil {
		decode = []colValRange{{min: 0, max: 255}}
	}

	min := decode[c].min
	max := decode[c].max

	q := 1
	for i := 1; i < bpc; i++ {
		q = 2*q + 1
	}

	v := uint8(min + (float64(p) * (max - min) / float64(q)))

	if decode[c].inv {
		v = v ^ 0xff
	}

	return v
}

func streamBytes(sd *StreamDict) ([]byte, error) {

	fpl := sd.FilterPipeline
	if fpl == nil {
		log.Info.Printf("streamBytes: no filter pipeline\n")
		if err := sd.Decode(); err != nil {
			return nil, err
		}
		return sd.Content, nil
	}

	// Ignore filter chains with length > 1
	if len(fpl) > 1 {
		log.Info.Printf("streamBytes: more than 1 filter\n")
		return nil, nil
	}

	switch fpl[0].Name {

	case filter.Flate:
		if err := sd.Decode(); err != nil {
			return nil, err
		}

	default:
		log.Debug.Printf("streamBytes: filter not \"Flate\": %s\n", fpl[0].Name)
		return nil, nil
	}

	return sd.Content, nil
}

// Return the soft mask for this image or nil.
func softMask(xRefTable *XRefTable, d *StreamDict, w, h, objNr int) ([]byte, error) {

	// TODO Process optional "Matte".

	o, _ := d.Find("SMask")
	if o == nil {
		// No soft mask available.
		return nil, nil
	}

	// Soft mask present.

	sd, _, err := xRefTable.DereferenceStreamDict(o)
	if err != nil {
		return nil, err
	}

	sm, err := streamBytes(sd)
	if err != nil {
		return nil, err
	}

	bpc := sd.IntEntry("BitsPerComponent")
	if bpc == nil {
		log.Info.Printf("softMask: obj#%d - ignoring soft mask without bpc\n%s\n", objNr, sd)
		return nil, nil
	}

	// TODO support soft masks with bpc != 8
	// Will need to return the softmask bpc to caller.
	if *bpc != 8 {
		log.Info.Printf("softMask: obj#%d - ignoring soft mask with bpc=%d\n", objNr, *bpc)
		return nil, nil
	}

	if sm != nil {
		if len(sm) != (*bpc*w*h+7)/8 {
			log.Info.Printf("softMask: obj#%d - ignoring corrupt softmask\n%s\n", objNr, sd)
			return nil, nil
		}
	}

	return sm, nil
}

func renderDeviceCMYKToTIFF(im *PDFImage, resourceName string) (*Image, error) {

	b := im.sd.Content

	log.Debug.Printf("renderDeviceCMYKToTIFF: CMYK objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	img := image.NewCMYK(image.Rect(0, 0, im.w, im.h))

	i := 0

	// TODO support bpc, decode and softMask.

	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			img.Set(x, y, color.CMYK{C: b[i], M: b[i+1], Y: b[i+2], K: b[i+3]})
			i += 4
		}
	}

	var buf bytes.Buffer
	// TODO softmask handling.
	if err := tiff.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return &Image{&buf, 0, resourceName, "tif"}, nil
}

func renderDeviceGrayToPNG(im *PDFImage, resourceName string) (*Image, error) {

	b := im.sd.Content

	log.Debug.Printf("renderDeviceGrayToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// Validate buflen.
	// For streams not using compression there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (im.bpc*im.w*im.h+7)/8 {
		return nil, errors.Errorf("pdfcpu: renderDeviceGrayToPNG: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	img := image.NewGray(image.Rect(0, 0, im.w, im.h))

	// TODO support softmask.
	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				pix := p >> (8 - uint8(im.bpc))
				v := decodePixelColorValue(pix, im.bpc, 0, im.decode)
				//fmt.Printf("x=%d y=%d pix=#%02x v=#%02x\n", x, y, pix, v)
				img.Set(x, y, color.Gray{Y: v})
				p <<= uint8(im.bpc)
				x++
			}
			i++
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &Image{&buf, 0, resourceName, "png"}, nil
}

func renderDeviceRGBToPNG(im *PDFImage, resourceName string) (*Image, error) {

	b := im.sd.Content

	log.Debug.Printf("renderDeviceRGBToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (3*im.bpc*im.w*im.h+7)/8 {
		return nil, errors.Errorf("pdfcpu: renderDeviceRGBToPNG: objNr=%d corrupt image object\n", im.objNr)
	}

	// TODO Support bpc and decode.
	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))

	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			alpha := uint8(255)
			if im.softMask != nil {
				alpha = im.softMask[y*im.w+x]
			}
			img.Set(x, y, color.NRGBA{R: b[i], G: b[i+1], B: b[i+2], A: alpha})
			i += 3
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &Image{&buf, 0, resourceName, "png"}, nil
}

func ensureDeviceRGBCS(xRefTable *XRefTable, o Object) bool {

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return false
	}

	switch altCS := o.(type) {
	case Name:
		return altCS == DeviceRGBCS
	}

	return false
}

func renderCalRGBToPNG(im *PDFImage, resourceName string) (*Image, error) {

	b := im.sd.Content

	log.Debug.Printf("renderCalRGBToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	if len(b) < (3*im.bpc*im.w*im.h+7)/8 {
		return nil, errors.Errorf("pdfcpu:renderCalRGBToPNG: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	// Optional int array "Range", length 2*N specifies min,max values of color components.
	// This information can be validated against the iccProfile.

	// RGB
	// TODO Support bpc, decode and softmask.
	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))
	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			img.Set(x, y, color.NRGBA{R: b[i], G: b[i+1], B: b[i+2], A: 255})
			i += 3
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &Image{&buf, 0, resourceName, "png"}, nil
}

func renderICCBased(xRefTable *XRefTable, im *PDFImage, resourceName string, cs Array) (*Image, error) {

	//  Any ICC profile >= ICC.1:2004:10 is sufficient for any PDF version <= 1.7
	//  If the embedded ICC profile version is newer than the one used by the Reader, substitute with Alternate color space.

	iccProfileStream, _, _ := xRefTable.DereferenceStreamDict(cs[1])

	b := im.sd.Content

	log.Debug.Printf("renderICCBasedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// 1,3 or 4 color components.
	n := *iccProfileStream.IntEntry("N")

	if !IntMemberOf(n, []int{1, 3, 4}) {
		return nil, errors.Errorf("pdfcpu: renderICCBasedToPNGFile: objNr=%d, N must be 1,3 or 4, got:%d\n", im.objNr, n)
	}

	// TODO: Transform linear XYZ to RGB according to ICC profile.
	// For now we fall back to appropriate color spaces for n
	// regardless of a specified alternate color space.

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (n*im.bpc*im.w*im.h+7)/8 {
		return nil, errors.Errorf("pdfcpu: renderICCBased: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	switch n {
	case 1:
		// Gray
		return renderDeviceGrayToPNG(im, resourceName)

	case 3:
		// RGB
		return renderDeviceRGBToPNG(im, resourceName)

	case 4:
		// CMYK
		return renderDeviceCMYKToTIFF(im, resourceName)
	}

	return nil, nil
}

func renderIndexedRGBToPNG(im *PDFImage, resourceName string, lookup []byte) (*Image, error) {

	b := im.sd.Content

	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))

	// TODO handle decode.

	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				ind := p >> (8 - uint8(im.bpc))
				//fmt.Printf("x=%d y=%d i=%d j=%d p=#%02x ind=#%02x\n", x, y, i, j, p, ind)
				alpha := uint8(255)
				if im.softMask != nil {
					alpha = im.softMask[y*im.w+x]
				}
				l := 3 * int(ind)
				img.Set(x, y, color.NRGBA{R: lookup[l], G: lookup[l+1], B: lookup[l+2], A: alpha})
				p <<= uint8(im.bpc)
				x++
			}
			i++
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return &Image{&buf, 0, resourceName, "png"}, nil
}

func renderIndexedCMYKToTIFF(im *PDFImage, resourceName string, lookup []byte) (*Image, error) {

	b := im.sd.Content

	img := image.NewCMYK(image.Rect(0, 0, im.w, im.h))

	// TODO handle decode and softmask.

	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				ind := p >> (8 - uint8(im.bpc))
				//fmt.Printf("x=%d y=%d i=%d j=%d p=#%02x ind=#%02x\n", x, y, i, j, p, ind)
				l := 4 * int(ind)
				img.Set(x, y, color.CMYK{C: lookup[l], M: lookup[l+1], Y: lookup[l+2], K: lookup[l+3]})
				p <<= uint8(im.bpc)
				x++
			}
			i++
		}
	}

	var buf bytes.Buffer
	// TODO softmask handling.
	if err := tiff.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return &Image{&buf, 0, resourceName, "tif"}, nil
}

func renderIndexedNameCS(im *PDFImage, resourceName string, cs Name, maxInd int, lookup []byte) (*Image, error) {

	switch cs {

	case DeviceRGBCS:

		if len(lookup) < 3*(maxInd+1) {
			return nil, errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceRGB lookup table\n", im.objNr)
		}

		return renderIndexedRGBToPNG(im, resourceName, lookup)

	case DeviceCMYKCS:

		if len(lookup) < 4*(maxInd+1) {
			return nil, errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceCMYK lookup table\n", im.objNr)
		}

		return renderIndexedCMYKToTIFF(im, resourceName, lookup)
	}

	log.Info.Printf("renderIndexedNameCS: objNr=%d, unsupported base colorspace %s\n", im.objNr, cs.String())

	return nil, nil
}

func renderIndexedArrayCS(xRefTable *XRefTable, im *PDFImage, resourceName string, csa Array, maxInd int, lookup []byte) (*Image, error) {

	b := im.sd.Content

	cs, _ := csa[0].(Name)

	switch cs {

	case ICCBasedCS:

		iccProfileStream, _, _ := xRefTable.DereferenceStreamDict(csa[1])

		// 1,3 or 4 color components.
		n := *iccProfileStream.IntEntry("N")
		if !IntMemberOf(n, []int{1, 3, 4}) {
			return nil, errors.Errorf("pdfcpu: renderIndexedArrayCS: objNr=%d, N must be 1,3 or 4, got:%d\n", im.objNr, n)
		}

		// Validate the lookup table.
		if len(lookup) < n*(maxInd+1) {
			return nil, errors.Errorf("pdfcpu: renderIndexedArrayCS: objNr=%d, corrupt ICCBased lookup table\n", im.objNr)
		}

		// TODO: Transform linear XYZ to RGB according to ICC profile.
		// For now we fall back to approriate color spaces for n
		// regardless of a specified alternate color space.

		switch n {
		case 1:
			// Gray
			// TODO use lookupTable!
			// TODO handle bpc, decode and softmask.
			img := image.NewGray(image.Rect(0, 0, im.w, im.h))
			i := 0
			for y := 0; y < im.h; y++ {
				for x := 0; x < im.w; x++ {
					img.Set(x, y, color.Gray{Y: b[i]})
					i++
				}
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return nil, err
			}
			return &Image{&buf, 0, "", "png"}, nil

		case 3:
			// RGB
			return renderIndexedRGBToPNG(im, resourceName, lookup)

		case 4:
			// CMYK
			log.Debug.Printf("renderIndexedArrayCS: CMYK objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))
			return renderIndexedCMYKToTIFF(im, resourceName, lookup)
		}
	}

	log.Info.Printf("renderIndexedArrayCS: objNr=%d, unsupported base colorspace %s\n", im.objNr, csa)

	return nil, nil
}

func renderIndexed(xRefTable *XRefTable, im *PDFImage, resourceName string, cs Array) (*Image, error) {

	// Identify the base color space.
	baseCS, _ := xRefTable.Dereference(cs[1])

	// Identify the max index into the color lookup table.
	maxInd, _ := xRefTable.DereferenceInteger(cs[2])

	// Identify the color lookup table.
	var lookup []byte
	lookup, err := colorLookupTable(xRefTable, cs[3])
	if err != nil {
		return nil, err
	}
	if lookup == nil {
		return nil, errors.Errorf("pdfcpu: renderIndexed: objNr=%d IndexedCS with corrupt lookup table %s\n", im.objNr, cs)
	}
	//fmt.Printf("lookup: \n%s\n", hex.Dump(l))

	b := im.sd.Content

	log.Debug.Printf("renderIndexed: objNr=%d w=%d h=%d bpc=%d buflen=%d maxInd=%d\n", im.objNr, im.w, im.h, im.bpc, len(b), maxInd)

	// Validate buflen.
	// The image data is a sequence of index values for pixels.
	// Sometimes there is a trailing 0x0A.
	if len(b) < (im.bpc*im.w*im.h+7)/8 {
		return nil, errors.Errorf("pdfcpu: renderIndexed: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	switch cs := baseCS.(type) {
	case Name:
		return renderIndexedNameCS(im, resourceName, cs, maxInd.Value(), lookup)

	case Array:
		return renderIndexedArrayCS(xRefTable, im, resourceName, cs, maxInd.Value(), lookup)
	}

	return nil, nil
}

func renderFlateEncodedImage(xRefTable *XRefTable, sd *StreamDict, resourceName string, objNr int) (*Image, error) {

	pdfImage, err := pdfImage(xRefTable, sd, objNr)
	if err != nil {
		return nil, err
	}

	o, err := xRefTable.DereferenceDictEntry(sd.Dict, "ColorSpace")
	if err != nil {
		return nil, err
	}

	switch cs := o.(type) {

	case Name:
		switch cs {

		case DeviceGrayCS:
			return renderDeviceGrayToPNG(pdfImage, resourceName)

		case DeviceRGBCS:
			return renderDeviceRGBToPNG(pdfImage, resourceName)

		case DeviceCMYKCS:
			return renderDeviceCMYKToTIFF(pdfImage, resourceName)

		default:
			log.Info.Printf("renderFlateEncodedImage: objNr=%d, unsupported name colorspace %s\n", objNr, cs.String())
		}

	case Array:
		csn, _ := cs[0].(Name)

		switch csn {

		case CalRGBCS:
			return renderCalRGBToPNG(pdfImage, resourceName)

		case ICCBasedCS:
			return renderICCBased(xRefTable, pdfImage, resourceName, cs)

		case IndexedCS:
			return renderIndexed(xRefTable, pdfImage, resourceName, cs)

		default:
			log.Info.Printf("renderFlateEncodedImage: objNr=%d, unsupported array colorspace %s\n", objNr, csn)
		}

	}

	return nil, nil
}

// RenderImage returns a reader for the encoded image bytes.
// for extract
func RenderImage(xRefTable *XRefTable, sd *StreamDict, resourceName string, objNr int) (*Image, error) {
	switch sd.FilterPipeline[0].Name {

	case filter.Flate, filter.CCITTFax:
		// If color space is CMYK then write .tif else write .png
		return renderFlateEncodedImage(xRefTable, sd, resourceName, objNr)

	case filter.DCT:
		// Write original stream data.
		return &Image{bytes.NewReader(sd.Raw), 0, resourceName, "jpg"}, nil

	case filter.JPX:
		// Write original stream data.
		return &Image{bytes.NewReader(sd.Raw), 0, resourceName, "jpx"}, nil
	}

	return nil, nil
}

func WriteReader(path string, r io.Reader) error {
	w, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, r); err != nil {
		return err
	}
	return w.Close()
}

// WriteImage writes a PDF image object to disk.
func WriteImage(xRefTable *XRefTable, fileName string, sd *StreamDict, objNr int) (string, error) {
	img, err := RenderImage(xRefTable, sd, fileName, objNr)
	if err != nil {
		return "", err
	}
	return fileName, WriteReader(fileName, img)
}

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
	"encoding/gob"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"

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
}

// PDFImage represents a XObject of subtype image.
type PDFImage struct {
	objNr     int
	sd        *StreamDict
	comp      int
	bpc       int
	w, h      int
	softMask  []byte
	decode    []colValRange
	imageMask bool
	thumb     bool
}

func decodeArr(a Array) []colValRange {
	if a == nil {
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
		decode = append(decode, colValRange{min, max})
	}

	return decode
}

func pdfImage(xRefTable *XRefTable, sd *StreamDict, thumb bool, objNr int) (*PDFImage, error) {
	comp, err := xRefTable.ColorSpaceComponents(sd)
	if err != nil {
		return nil, err
	}

	bpc := *sd.IntEntry("BitsPerComponent")

	w := *sd.IntEntry("Width")
	h := *sd.IntEntry("Height")

	decode := decodeArr(sd.ArrayEntry("Decode"))

	var imgMask bool
	if im := sd.BooleanEntry("ImageMask"); im != nil && *im {
		imgMask = true
	}

	sm, err := softMask(xRefTable, sd, w, h, objNr)
	if err != nil {
		return nil, err
	}

	return &PDFImage{
		objNr:     objNr,
		sd:        sd,
		comp:      comp,
		bpc:       bpc,
		w:         w,
		h:         h,
		imageMask: imgMask,
		softMask:  sm,
		decode:    decode,
		thumb:     thumb,
	}, nil
}

// Identify the color lookup table for an Indexed color space.
func colorLookupTable(xRefTable *XRefTable, o Object) ([]byte, error) {
	o, _ = xRefTable.Dereference(o)

	switch o := o.(type) {

	case StringLiteral:
		return Unescape(o.Value())

	case HexLiteral:
		return o.Bytes()

	case StreamDict:
		return streamBytes(&o)

	}

	return nil, nil
}

func decodePixelValue(v uint8, bpc int, r colValRange) uint8 {

	// Odd way to calc 2**bpc-1
	q := 1
	for i := 1; i < bpc; i++ {
		q = 2*q + 1
	}

	f := r.min + (float64(v) * (r.max - r.min) / float64(q))

	return uint8(f * float64(q))
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

	var fName string
	var s []string
	for _, filter := range fpl {
		s = append(s, filter.Name)
		fName = filter.Name
	}
	filters := strings.Join(s, ",")

	f := fName

	switch f {

	case filter.DCT, filter.Flate, filter.CCITTFax, filter.ASCII85, filter.RunLength:
		// If color space is CMYK then write .tif else write .png
		if err := sd.Decode(); err != nil {
			return nil, err
		}

	case filter.JPX:
		//imageObj.Extension = "jpx"

	default:
		log.Debug.Printf("streamBytes: skip img, filter %s unsupported\n", filters)
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

func renderDeviceCMYKToTIFF(im *PDFImage, resourceName string) (io.Reader, string, error) {
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
		return nil, "", err
	}

	return &buf, "tif", nil
}

func renderDeviceGrayToPNG(im *PDFImage, resourceName string) (io.Reader, string, error) {
	b := im.sd.Content
	log.Debug.Printf("renderDeviceGrayToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// Validate buflen.
	// For streams not using compression there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu: renderDeviceGrayToPNG: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	img := image.NewGray(image.Rect(0, 0, im.w, im.h))

	// TODO support softmask.
	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				pix := p >> (8 - uint8(im.bpc))
				dec := []colValRange{{0, 1}}
				if !im.imageMask && im.decode != nil {
					dec = im.decode
				}
				v := decodePixelValue(pix, im.bpc, dec[0])
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
		return nil, "", err
	}

	return &buf, "png", nil
}

func renderDeviceRGBToPNG(im *PDFImage, resourceName string) (io.Reader, string, error) {
	b := im.sd.Content
	log.Debug.Printf("renderDeviceRGBToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (3*im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu: renderDeviceRGBToPNG: objNr=%d corrupt image object\n", im.objNr)
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
		return nil, "", err
	}

	return &buf, "png", nil
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

func renderCalRGBToPNG(im *PDFImage, resourceName string) (io.Reader, string, error) {
	b := im.sd.Content
	log.Debug.Printf("renderCalRGBToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	if len(b) < (3*im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu:renderCalRGBToPNG: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
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
		return nil, "", err
	}

	return &buf, "png", nil
}

func renderICCBased(xRefTable *XRefTable, im *PDFImage, resourceName string, cs Array) (io.Reader, string, error) {
	//  Any ICC profile >= ICC.1:2004:10 is sufficient for any PDF version <= 1.7
	//  If the embedded ICC profile version is newer than the one used by the Reader, substitute with Alternate color space.

	iccProfileStream, _, _ := xRefTable.DereferenceStreamDict(cs[1])

	b := im.sd.Content

	log.Debug.Printf("renderICCBasedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// 1,3 or 4 color components.
	n := *iccProfileStream.IntEntry("N")

	if !IntMemberOf(n, []int{1, 3, 4}) {
		return nil, "", errors.Errorf("pdfcpu: renderICCBasedToPNGFile: objNr=%d, N must be 1,3 or 4, got:%d\n", im.objNr, n)
	}

	// TODO: Transform linear XYZ to RGB according to ICC profile.
	// For now we fall back to appropriate color spaces for n
	// regardless of a specified alternate color space.

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (n*im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu: renderICCBased: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
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

	return nil, "", nil
}

func renderIndexedRGBToPNG(im *PDFImage, resourceName string, lookup []byte) (io.Reader, string, error) {
	b := im.sd.Content

	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))

	i := 0
	// TODO: For (some) Runlength encoded images the line sequence is reversed.
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
		return nil, "", err
	}

	return &buf, "png", nil
}

func renderIndexedCMYKToTIFF(im *PDFImage, resourceName string, lookup []byte) (io.Reader, string, error) {
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
		return nil, "", err
	}

	return &buf, "tif", nil
}

func renderIndexedNameCS(im *PDFImage, resourceName string, cs Name, maxInd int, lookup []byte) (io.Reader, string, error) {
	switch cs {

	case DeviceRGBCS:
		if len(lookup) < 3*(maxInd+1) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceRGB lookup table\n", im.objNr)
		}
		return renderIndexedRGBToPNG(im, resourceName, lookup)

	case DeviceCMYKCS:
		if len(lookup) < 4*(maxInd+1) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceCMYK lookup table\n", im.objNr)
		}
		return renderIndexedCMYKToTIFF(im, resourceName, lookup)
	}

	log.Info.Printf("renderIndexedNameCS: objNr=%d, unsupported base colorspace %s\n", im.objNr, cs.String())

	return nil, "", nil
}

func renderIndexedArrayCS(xRefTable *XRefTable, im *PDFImage, resourceName string, csa Array, maxInd int, lookup []byte) (io.Reader, string, error) {
	b := im.sd.Content

	cs, _ := csa[0].(Name)

	switch cs {

	//case CalGrayCS:

	case CalRGBCS:
		return renderIndexedRGBToPNG(im, resourceName, lookup)

	//case LabCS:
	//	return renderIndexedRGBToPNG(im, resourceName, lookup)

	case ICCBasedCS:

		iccProfileStream, _, _ := xRefTable.DereferenceStreamDict(csa[1])

		// 1,3 or 4 color components.
		n := *iccProfileStream.IntEntry("N")
		if !IntMemberOf(n, []int{1, 3, 4}) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedArrayCS: objNr=%d, N must be 1,3 or 4, got:%d\n", im.objNr, n)
		}

		// Validate the lookup table.
		if len(lookup) < n*(maxInd+1) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedArrayCS: objNr=%d, corrupt ICCBased lookup table\n", im.objNr)
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
				return nil, "", err
			}
			return &buf, "png", nil

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

	return nil, "", nil
}

func renderIndexed(xRefTable *XRefTable, im *PDFImage, resourceName string, cs Array) (io.Reader, string, error) {
	// Identify the base color space.
	baseCS, _ := xRefTable.Dereference(cs[1])

	// Identify the max index into the color lookup table.
	maxInd, _ := xRefTable.DereferenceInteger(cs[2])

	// Identify the color lookup table.
	var lookup []byte
	lookup, err := colorLookupTable(xRefTable, cs[3])
	if err != nil {
		return nil, "", err
	}

	if lookup == nil {
		return nil, "", errors.Errorf("pdfcpu: renderIndexed: objNr=%d IndexedCS with corrupt lookup table %s\n", im.objNr, cs)
	}

	b := im.sd.Content

	log.Debug.Printf("renderIndexed: objNr=%d w=%d h=%d bpc=%d buflen=%d maxInd=%d\n", im.objNr, im.w, im.h, im.bpc, len(b), maxInd)

	// Validate buflen.
	// The image data is a sequence of index values for pixels.
	// Sometimes there is a trailing 0x0A.
	if len(b) < (im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu: renderIndexed: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	switch cs := baseCS.(type) {
	case Name:
		return renderIndexedNameCS(im, resourceName, cs, maxInd.Value(), lookup)

	case Array:
		return renderIndexedArrayCS(xRefTable, im, resourceName, cs, maxInd.Value(), lookup)
	}

	return nil, "", nil
}

func renderFlateEncodedImage(xRefTable *XRefTable, sd *StreamDict, thumb bool, resourceName string, objNr int) (io.Reader, string, error) {
	// If color space is CMYK then write .tif else write .png

	pdfImage, err := pdfImage(xRefTable, sd, thumb, objNr)
	if err != nil {
		return nil, "", err
	}

	o, err := xRefTable.DereferenceDictEntry(sd.Dict, "ColorSpace")
	if err != nil {
		return nil, "", err
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

	return nil, "", nil
}

func renderGrayToPng(im *PDFImage, resourceName string) (io.Reader, string, error) {
	bb := bytes.NewReader(im.sd.Content)
	dec := gob.NewDecoder(bb)

	var img image.Gray
	if err := dec.Decode(&img); err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, &img); err != nil {
		return nil, "", err
	}

	return &buf, "png", nil
}

func renderRGBToPng(im *PDFImage, resourceName string) (io.Reader, string, error) {
	bb := bytes.NewReader(im.sd.Content)
	dec := gob.NewDecoder(bb)

	var img image.YCbCr
	if err := dec.Decode(&img); err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, &img); err != nil {
		return nil, "", err
	}

	return &buf, "png", nil
}

// func decode8BitColVal(v uint8, r colValRange) uint8 {
// 	return uint8(r.min + (float64(v) * (r.max - r.min) / 255.0))
// }

func decodeCMYK(c, m, y, k uint8, decode []colValRange) (uint8, uint8, uint8, uint8) {
	if len(decode) == 0 {
		return c, m, y, k
	}
	c = decodePixelValue(c, 8, decode[0])
	m = decodePixelValue(m, 8, decode[1])
	y = decodePixelValue(y, 8, decode[2])
	k = decodePixelValue(k, 8, decode[3])
	return c, m, y, k
}

func renderCMYKToPng(im *PDFImage, resourceName string) (io.Reader, string, error) {
	bb := bytes.NewReader(im.sd.Content)
	dec := gob.NewDecoder(bb)

	var img image.CMYK
	if err := dec.Decode(&img); err != nil {
		return nil, "", err
	}

	img1 := image.NewRGBA(image.Rect(0, 0, im.w, im.h))

	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			//c := img.At(x, y)
			a := img.At(x, y).(color.CMYK)
			cyan, mag, yel, blk := decodeCMYK(255-a.C, 255-a.M, 255-a.Y, 255-a.K, im.decode)
			r, g, b := color.CMYKToRGB(cyan, mag, yel, blk)
			// TODO Apply Decode array
			img1.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img1); err != nil {
		return nil, "", err
	}

	return &buf, "png", nil
}

func renderDCTEncodedImage(xRefTable *XRefTable, sd *StreamDict, thumb bool, resourceName string, objNr int) (io.Reader, string, error) {
	im, err := pdfImage(xRefTable, sd, thumb, objNr)
	if err != nil {
		return nil, "", err
	}

	switch im.comp {
	case 1:
		return renderGrayToPng(im, resourceName)
	case 3:
		return renderRGBToPng(im, resourceName)
	case 4:
		return renderCMYKToPng(im, resourceName)
	}

	return nil, "", errors.Errorf("renderDCTEncodedImage: invalid number of components: %d", im.comp)
}

// RenderImage returns a reader for a decoded image stream.
func RenderImage(xRefTable *XRefTable, sd *StreamDict, thumb bool, resourceName string, objNr int) (io.Reader, string, error) {
	// The real image compression is the last filter in the pipeline.
	f := sd.FilterPipeline[len(sd.FilterPipeline)-1].Name

	switch f {

	case filter.Flate, filter.CCITTFax, filter.RunLength:
		return renderFlateEncodedImage(xRefTable, sd, thumb, resourceName, objNr)

	case filter.DCT:
		return renderDCTEncodedImage(xRefTable, sd, thumb, resourceName, objNr)

	case filter.JPX:
		// Exception: Write original encoded stream data.
		return bytes.NewReader(sd.Raw), "jpx", nil
	}

	return nil, "", nil
}

// WriteReader consumes r's content by writing it to a file at path.
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
func WriteImage(xRefTable *XRefTable, fileName string, sd *StreamDict, thumb bool, objNr int) (string, error) {
	r, _, err := RenderImage(xRefTable, sd, thumb, fileName, objNr)
	if err != nil {
		return "", err
	}
	return fileName, WriteReader(fileName, r)
}

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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
	sd        *types.StreamDict
	comp      int
	bpc       int
	w, h      int
	softMask  []byte
	decode    []colValRange
	imageMask bool
	thumb     bool
}

func decodeArr(a types.Array) []colValRange {
	if a == nil {
		return nil
	}

	var decode []colValRange
	var min, max, f64 float64

	for i, f := range a {
		switch o := f.(type) {
		case types.Integer:
			f64 = float64(o.Value())
		case types.Float:
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

func pdfImage(xRefTable *model.XRefTable, sd *types.StreamDict, thumb bool, objNr int) (*PDFImage, error) {
	comp, err := ColorSpaceComponents(xRefTable, sd)
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
func colorLookupTable(xRefTable *model.XRefTable, o types.Object) ([]byte, error) {
	o, _ = xRefTable.Dereference(o)

	switch o := o.(type) {

	case types.StringLiteral:
		return types.Unescape(o.Value(), false)

	case types.HexLiteral:
		return o.Bytes()

	case types.StreamDict:
		return streamBytes(&o)

	}

	return nil, nil
}

func maxValForBits(bpc int) int {
	return 1<<bpc - 1
}

// Decode v into the bpc deep color image space using the applicable DecodeArray component.
func decodePixelValue(v uint8, bpc int, r colValRange) uint8 {
	q := float64(maxValForBits(bpc))
	f := r.min + (float64(v) * (r.max - r.min) / q)
	return uint8(f * q)
}

func streamBytes(sd *types.StreamDict) ([]byte, error) {
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
func softMask(xRefTable *model.XRefTable, d *types.StreamDict, w, h, objNr int) ([]byte, error) {

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

func imageForCMYKWithoutSoftMask(im *PDFImage) image.Image {

	// Preserve CMYK color model for print applications.

	// TODO support bpc, decode.

	img := image.NewCMYK(image.Rect(0, 0, im.w, im.h))
	b := im.sd.Content
	i := 0

	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			img.Set(x, y, color.CMYK{C: b[i], M: b[i+1], Y: b[i+2], K: b[i+3]})
			i += 4
		}
	}

	return img

}

func imageForCMYKWithSoftMask(im *PDFImage) image.Image {

	// TODO support bpc, decode.

	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))
	b := im.sd.Content
	i := 0

	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; x++ {
			cr, cg, cb := color.CMYKToRGB(b[i], b[i+1], b[i+2], b[i+3])
			alpha := im.softMask[y*im.w+x]
			img.Set(x, y, color.NRGBA{cr, cg, cb, alpha})
			i += 4
		}
	}

	return img
}

func renderDeviceCMYKToTIFF(im *PDFImage, resourceName string) (io.Reader, string, error) {
	b := im.sd.Content
	log.Debug.Printf("renderDeviceCMYKToTIFF: CMYK objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	var img image.Image
	if im.softMask != nil {
		img = imageForCMYKWithSoftMask(im)
	} else {
		img = imageForCMYKWithoutSoftMask(im)
	}

	var buf bytes.Buffer
	if err := tiff.Encode(&buf, img, nil); err != nil {
		return nil, "", err
	}

	return &buf, "tif", nil
}

func scaleToBPC8(v uint8, bpc int) uint8 {
	return uint8(float64(v) * 255.0 / float64(maxValForBits(bpc)))
}

func renderDeviceGrayToPNG(im *PDFImage, resourceName string) (io.Reader, string, error) {
	b := im.sd.Content
	log.Debug.Printf("renderDeviceGrayToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// Validate buflen.
	// For streams not using compression there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu: renderDeviceGrayToPNG: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	cvr := colValRange{0, 1}
	if im.decode != nil {
		cvr = im.decode[0]
	}

	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))

	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				pix := p >> (8 - uint8(im.bpc))
				v := decodePixelValue(pix, im.bpc, cvr)
				if im.bpc < 8 {
					v = scaleToBPC8(v, im.bpc)
				}
				alpha := uint8(255)
				if im.softMask != nil {
					alpha = im.softMask[y*im.w+x]
				}
				img.Set(x, y, color.NRGBA{R: v, G: v, B: v, A: alpha})
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

func renderICCBased(xRefTable *model.XRefTable, im *PDFImage, resourceName string, cs types.Array) (io.Reader, string, error) {
	//  Any ICC profile >= ICC.1:2004:10 is sufficient for any PDF version <= 1.7
	//  If the embedded ICC profile version is newer than the one used by the Reader, substitute with Alternate color space.

	iccProfileStream, _, _ := xRefTable.DereferenceStreamDict(cs[1])

	b := im.sd.Content

	log.Debug.Printf("renderICCBasedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// 1,3 or 4 color components.
	n := *iccProfileStream.IntEntry("N")

	if !types.IntMemberOf(n, []int{1, 3, 4}) {
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

func renderIndexedGrayToPNG(im *PDFImage, resourceName string, lookup []byte) (io.Reader, string, error) {
	b := im.sd.Content
	log.Debug.Printf("renderIndexedGrayToPNG: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", im.objNr, im.w, im.h, im.bpc, len(b))

	// Validate buflen.
	// For streams not using compression there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (im.bpc*im.w*im.h+7)/8 {
		return nil, "", errors.Errorf("pdfcpu: renderIndexedGrayToPNG: objNr=%d corrupt image object %v\n", im.objNr, *im.sd)
	}

	cvr := colValRange{0, 1}
	if im.decode != nil {
		cvr = im.decode[0]
	}

	img := image.NewGray(image.Rect(0, 0, im.w, im.h))

	// TODO support softmask.
	i := 0
	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				ind := p >> (8 - uint8(im.bpc))
				v := decodePixelValue(lookup[ind], im.bpc, cvr)
				if im.bpc < 8 {
					v = scaleToBPC8(v, im.bpc)
				}
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

func imageForIndexedCMYKWithoutSoftMask(im *PDFImage, lookup []byte) image.Image {

	// Preserve CMYK color model for print applications.

	// TODO handle decode

	img := image.NewCMYK(image.Rect(0, 0, im.w, im.h))
	b := im.sd.Content
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

	return img
}

func imageForIndexedCMYKWithSoftMask(im *PDFImage, lookup []byte) image.Image {

	// TODO handle decode

	img := image.NewNRGBA(image.Rect(0, 0, im.w, im.h))
	b := im.sd.Content
	i := 0

	for y := 0; y < im.h; y++ {
		for x := 0; x < im.w; {
			p := b[i]
			for j := 0; j < 8/im.bpc; j++ {
				ind := p >> (8 - uint8(im.bpc))
				//fmt.Printf("x=%d y=%d i=%d j=%d p=#%02x ind=#%02x\n", x, y, i, j, p, ind)
				l := 4 * int(ind)
				cr, cg, cb := color.CMYKToRGB(lookup[l], lookup[l+1], lookup[l+2], lookup[l+3])
				alpha := im.softMask[y*im.w+x]
				img.Set(x, y, color.NRGBA{cr, cg, cb, alpha})
				p <<= uint8(im.bpc)
				x++
			}
			i++
		}
	}

	return img
}

func renderIndexedCMYKToTIFF(im *PDFImage, resourceName string, lookup []byte) (io.Reader, string, error) {

	var img image.Image
	if im.softMask != nil {
		img = imageForIndexedCMYKWithSoftMask(im, lookup)
	} else {
		img = imageForIndexedCMYKWithoutSoftMask(im, lookup)
	}

	var buf bytes.Buffer
	if err := tiff.Encode(&buf, img, nil); err != nil {
		return nil, "", err
	}

	return &buf, "tif", nil
}

func renderIndexedNameCS(im *PDFImage, resourceName string, cs types.Name, maxInd int, lookup []byte) (io.Reader, string, error) {
	switch cs {

	case model.DeviceGrayCS:
		if len(lookup) < 1*(maxInd+1) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceGray lookup table\n", im.objNr)
		}
		return renderIndexedGrayToPNG(im, resourceName, lookup)

	case model.DeviceRGBCS:
		if len(lookup) < 3*(maxInd+1) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceRGB lookup table\n", im.objNr)
		}
		return renderIndexedRGBToPNG(im, resourceName, lookup)

	case model.DeviceCMYKCS:
		if len(lookup) < 4*(maxInd+1) {
			return nil, "", errors.Errorf("pdfcpu: renderIndexedNameCS: objNr=%d, corrupt DeviceCMYK lookup table\n", im.objNr)
		}
		return renderIndexedCMYKToTIFF(im, resourceName, lookup)
	}

	log.Info.Printf("renderIndexedNameCS: objNr=%d, unsupported base colorspace %s\n", im.objNr, cs.String())

	return nil, "", nil
}

func renderIndexedArrayCS(xRefTable *model.XRefTable, im *PDFImage, resourceName string, csa types.Array, maxInd int, lookup []byte) (io.Reader, string, error) {
	b := im.sd.Content

	cs, _ := csa[0].(types.Name)

	switch cs {

	//case CalGrayCS:

	case model.CalRGBCS:
		return renderIndexedRGBToPNG(im, resourceName, lookup)

	//case LabCS:
	//	return renderIndexedRGBToPNG(im, resourceName, lookup)

	case model.ICCBasedCS:

		iccProfileStream, _, _ := xRefTable.DereferenceStreamDict(csa[1])

		// 1,3 or 4 color components.
		n := *iccProfileStream.IntEntry("N")
		if !types.IntMemberOf(n, []int{1, 3, 4}) {
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

func renderIndexed(xRefTable *model.XRefTable, im *PDFImage, resourceName string, cs types.Array) (io.Reader, string, error) {
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
	case types.Name:
		return renderIndexedNameCS(im, resourceName, cs, maxInd.Value(), lookup)

	case types.Array:
		return renderIndexedArrayCS(xRefTable, im, resourceName, cs, maxInd.Value(), lookup)
	}

	return nil, "", nil
}

func renderDeviceN(xRefTable *model.XRefTable, im *PDFImage, resourceName string, cs types.Array) (io.Reader, string, error) {

	switch im.comp {
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

func renderFlateEncodedImage(xRefTable *model.XRefTable, sd *types.StreamDict, thumb bool, resourceName string, objNr int) (io.Reader, string, error) {
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

	case types.Name:
		switch cs {

		case model.DeviceGrayCS:
			return renderDeviceGrayToPNG(pdfImage, resourceName)

		case model.DeviceRGBCS:
			return renderDeviceRGBToPNG(pdfImage, resourceName)

		case model.DeviceCMYKCS:
			return renderDeviceCMYKToTIFF(pdfImage, resourceName)

		default:
			log.Info.Printf("renderFlateEncodedImage: objNr=%d, unsupported name colorspace %s\n", objNr, cs.String())
		}

	case types.Array:
		csn, _ := cs[0].(types.Name)

		switch csn {

		case model.CalRGBCS:
			return renderCalRGBToPNG(pdfImage, resourceName)

		case model.DeviceNCS:
			return renderDeviceN(xRefTable, pdfImage, resourceName, cs)

		case model.ICCBasedCS:
			return renderICCBased(xRefTable, pdfImage, resourceName, cs)

		case model.IndexedCS:
			return renderIndexed(xRefTable, pdfImage, resourceName, cs)

		case model.SeparationCS:
			return renderDeviceN(xRefTable, pdfImage, resourceName, cs)

		default:
			log.Info.Printf("renderFlateEncodedImage: objNr=%d, unsupported array colorspace %s\n", objNr, csn)
		}

	}

	return nil, "", nil
}

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

func renderDCTToPNG(xRefTable *model.XRefTable, sd *types.StreamDict, thumb bool, resourceName string, objNr int) (io.Reader, string, error) {
	im, err := pdfImage(xRefTable, sd, thumb, objNr)
	if err != nil {
		return nil, "", err
	}

	return renderCMYKToPng(im, resourceName)
}

// RenderImage returns a reader for a decoded image stream.
func RenderImage(xRefTable *model.XRefTable, sd *types.StreamDict, thumb bool, resourceName string, objNr int) (io.Reader, string, error) {
	// Image compression is the last filter in the pipeline.

	f := sd.FilterPipeline[len(sd.FilterPipeline)-1].Name

	switch f {

	case filter.Flate, filter.CCITTFax, filter.RunLength:
		return renderFlateEncodedImage(xRefTable, sd, thumb, resourceName, objNr)

	case filter.DCT:
		if sd.CSComponents == 4 {
			return renderDCTToPNG(xRefTable, sd, thumb, resourceName, objNr)
		}
		return bytes.NewReader(sd.Content), "jpg", nil

	case filter.JPX:
		return bytes.NewReader(sd.Content), "jpx", nil
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
func WriteImage(xRefTable *model.XRefTable, fileName string, sd *types.StreamDict, thumb bool, objNr int) (string, error) {
	r, _, err := RenderImage(xRefTable, sd, thumb, fileName, objNr)
	if err != nil {
		return "", err
	}
	if r == nil {
		return "", errors.Errorf("pdfcpu: unable to extract image from obj#%d", objNr)
	}
	return fileName, WriteReader(fileName, r)
}

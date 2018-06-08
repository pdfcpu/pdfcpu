package pdfcpu

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// ErrUnsupportedColorSpace indicates an unsupported color space.
var ErrUnsupportedColorSpace = errors.New("unsupported color space")

func writeImgToPNG(fileName string, img image.Image) error {

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// Return the soft mask for this image or nil.
func softMask(ctx *PDFContext, d *PDFStreamDict, w, h, objNr int) ([]byte, error) {

	// TODO Process the optional "Matte" entry.

	o, _ := d.Find("SMask")
	if o == nil {
		// No soft mask available.
		return nil, nil
	}

	// Soft mask present.

	sd, err := ctx.DereferenceStreamDict(o)
	if err != nil {
		return nil, err
	}

	sm, err := streamBytes(sd)
	if err != nil {
		return nil, err
	}

	if sm != nil {
		if len(sm) != w*h {
			return nil, errors.Errorf("writeImage: objNr=%d, corrupt softmask\n", objNr)
		}
	}

	return sm, nil
}

func writeDeviceGrayToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject) error {

	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")
	bpc := *io.ImageDict.IntEntry("BitsPerComponent")

	// TODO Handle SMask with DeviceGrayCS (eg. ekanna.pdf)

	b := io.Data()
	log.Debug.Printf("writeDeviceGrayToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	//if bpc*len(b) < bpc*w*h {
	if len(b) < (bpc*w*h+7)/8 {
		return errors.Errorf("writeDeviceGrayToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// We support 8 bits per component/color only.
	if bpc != 8 {
		return errors.Errorf("writeDeviceGrayToPNGFile: objNr=%d, must be 8 bits per component, got: %d\n", objNr, bpc)
	}

	img := image.NewGray(image.Rect(0, 0, w, h))

	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.Gray{Y: b[i]})
			i++
		}
	}

	return writeImgToPNG(fileName, img)
}

func writeDeviceRGBToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject) error {

	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")
	bpc := *io.ImageDict.IntEntry("BitsPerComponent")

	// TODO Handle SMask with DeviceGrayCS (eg. ekanna.pdf)

	b := io.Data()
	log.Debug.Printf("writeDeviceRGBToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if bpc*len(b) < 3*bpc*w*h {
		return errors.Errorf("writeDeviceRGBToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// We support 8 bits per component/color only.
	if bpc != 8 {
		return errors.Errorf("writeDeviceRGBToPNGFile: objNr=%d, must be 8 bits per component, got: %d\n", objNr, bpc)
	}

	sm, err := softMask(ctx, io.ImageDict, w, h, objNr)
	if err != nil {
		return err
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			alpha := uint8(255)
			if sm != nil {
				alpha = sm[y*w+x]
			}
			img.Set(x, y, color.NRGBA{R: b[i], G: b[i+1], B: b[i+2], A: alpha})
			i += 3
		}
	}

	return writeImgToPNG(fileName, img)
}

func writeDeviceCMYKToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject) error {

	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")
	bpc := *io.ImageDict.IntEntry("BitsPerComponent")

	b := io.Data()
	log.Debug.Printf("writeDeviceCMYKToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if bpc*len(b) < 4*bpc*w*h {
		return errors.Errorf("writeDeviceCMYKToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// We support 8 bits per component/color only.
	if bpc != 8 {
		return errors.Errorf("writeDeviceCMYKToPNGFile: objNr=%d, must be 8 bits per component, got: %d\n", objNr, bpc)
	}

	img := image.NewCMYK(image.Rect(0, 0, w, h))

	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.CMYK{C: b[i], M: b[i+1], Y: b[i+2], K: b[i+3]})
			i += 4
		}
	}

	return writeImgToPNG(fileName, img)
}

func ensureDeviceRGBCS(ctx *PDFContext, o PDFObject) bool {

	o, err := ctx.Dereference(o)
	if err != nil {
		return false
	}

	switch altCS := o.(type) {
	case PDFName:
		return altCS == DeviceRGBCS
	}

	return false
}

func streamBytes(sd *PDFStreamDict) ([]byte, error) {

	fpl := sd.FilterPipeline
	if fpl == nil {
		log.Info.Printf("streamBytes: no filter pipeline\n")
		return nil, nil
	}

	// Ignore filter chains with length > 1
	if len(fpl) > 1 {
		log.Info.Printf("streamBytes: more than 1 filter\n")
		return nil, nil
	}

	switch fpl[0].Name {

	case filter.Flate:
		err := decodeStream(sd)
		if err != nil {
			return nil, err
		}

	default:
		log.Debug.Printf("streamBytes: filter not \"Flate\"\n")
		return nil, nil
	}

	return sd.Content, nil
}

func writeCalRGBToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject) error {

	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")
	bpc := *io.ImageDict.IntEntry("BitsPerComponent")

	b := io.Data()
	log.Debug.Printf("writeICCBasedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	if bpc*len(b) < 3*bpc*w*h {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// We support 8 bits per component/color only.
	if bpc != 8 {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d, must be 8 bits per component, got: %d\n", objNr, bpc)
	}

	// Optional int array "Range", length 2*N specifies min,max values of color components.
	// This information can be validated against the iccProfile.

	// For now use alternate color space DeviceRGB for n=3 and DeviceCMYK for n=4 !

	// TODO: Transform linear XYZ to RGB according to ICC profile.

	// RGB
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: b[i], G: b[i+1], B: b[i+2], A: 255})
			i += 3
		}
	}
	return writeImgToPNG(fileName, img)
}

func writeICCBasedToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, iccProfileStream *PDFStreamDict) error {

	//  Any ICC profile >= ICC.1:2004:10 is sufficient for any PDF version <= 1.7
	//  If the embedded ICC profile version is newer than the one used by the Reader, substitute with Alternate color space.

	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")
	bpc := *io.ImageDict.IntEntry("BitsPerComponent")

	b := io.Data()
	log.Debug.Printf("writeICCBasedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// 1,3 or 4 color components.
	n := *iccProfileStream.IntEntry("N")

	if !intMemberOf(n, []int{1, 3, 4}) {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d, N must be 1,3 or 4, got:%d\n", objNr, n)
	}
	//if n != 3 {
	//	return errors.Errorf("writeICCBasedToPNGFile: objNr=%d, N must be 3, got:%d\n", objNr, n)
	//}
	//if n == 4 use DeviceCMYK regardless.

	// iccBytes, err := streamBytes(iccProfileStream)
	// if err != nil || iccBytes == nil {
	// 	return err
	// }
	// p := &iccProfile{b: iccBytes}
	// p.init()
	// log.Debug.Printf("ICC-Profile:\n%s", p)

	// DeviceRGB is the only supported alternate color space.
	// This has to be a color space with N color components.
	// var alternateDeviceRGBCS bool
	// o, found := iccProfileStream.Find("Alternate")
	// if found {
	// 	alternateDeviceRGBCS = ensureDeviceRGBCS(ctx, o)
	// }
	//if !alternateDeviceRGBCS {
	//	return errors.Errorf("writeICCBasedToPNGFile: objNr=%d, missing alternative DeviceRGB color space\n", objNr)
	//}

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if bpc*len(b) < n*bpc*w*h {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// We support 8 bits per component/color only.
	if bpc != 8 {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d, must be 8 bits per component, got: %d\n", objNr, bpc)
	}

	// Optional int array "Range", length 2*N specifies min,max values of color components.
	// This information can be validated against the iccProfile.

	// For now use alternate color space DeviceRGB for n=3 and DeviceCMYK for n=4 !

	// TODO: Transform linear XYZ to RGB according to ICC profile.

	sm, err := softMask(ctx, io.ImageDict, w, h, objNr)
	if err != nil {
		return err
	}

	if n == 1 {
		// Gray
		img := image.NewGray(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, color.Gray{Y: b[i]})
				i++
			}
		}
		return writeImgToPNG(fileName, img)
	}

	if n == 3 {
		// RGB
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				alpha := uint8(255)
				if sm != nil {
					alpha = sm[y*w+x]
				}
				img.Set(x, y, color.NRGBA{R: b[i], G: b[i+1], B: b[i+2], A: alpha})
				i += 3
			}
		}
		return writeImgToPNG(fileName, img)
	}

	// n == 4 => CMYK
	log.Debug.Printf("writeICCBasedToPNGFile: CMYK objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))
	img := image.NewCMYK(image.Rect(0, 0, w, h))
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.CMYK{C: b[i], M: b[i+1], Y: b[i+2], K: b[i+3]})
			i += 4
		}
	}
	return writeImgToPNG(fileName, img)
}

func writeIndexedNameCS(cs PDFName, objNr, w, h int, b, lookup, sm []byte, fileName string) error {

	var err error

	switch cs {

	case DeviceRGBCS:
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				alpha := uint8(255)
				if sm != nil {
					alpha = sm[y*w+x]
				}
				j := 3 * int(b[i])
				img.Set(x, y, color.NRGBA{R: lookup[j], G: lookup[j+1], B: lookup[j+2], A: alpha})
				i++
			}
		}
		err = writeImgToPNG(fileName, img)

	case DeviceCMYKCS:
		img := image.NewCMYK(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				j := 4 * int(b[i])
				img.Set(x, y, color.CMYK{C: lookup[j], M: lookup[j+1], Y: lookup[j+2], K: lookup[j+3]})
				i++
			}
		}
		err = writeImgToPNG(fileName, img)

	default:
		err = errors.Errorf("writeIndexedToPNGFile: objNr=%d unsupported base colorspace %s\n", objNr, cs.String())
	}

	return err
}

func writeIndexedArrayCS(ctx *PDFContext, csa PDFArray, objNr, w, h, bpc int, b, lookup, sm []byte, fileName string) error {

	cs, _ := csa[0].(PDFName)

	switch cs {

	case ICCBasedCS:

		iccProfileStream, _ := ctx.DereferenceStreamDict(csa[1])

		// 1,3 or 4 color components.
		n := *iccProfileStream.IntEntry("N")
		if !intMemberOf(n, []int{1, 3, 4}) {
			return errors.Errorf("writeIndexedToPNGFile: objNr=%d, N must be 1,3 or 4, got:%d\n", objNr, n)
		}

		// TODO: Transform linear XYZ to RGB according to ICC profile.
		var alternateDeviceRGBCS bool
		o, found := iccProfileStream.Find("Alternate")
		if found {
			alternateDeviceRGBCS = ensureDeviceRGBCS(ctx, o)
		}
		if !alternateDeviceRGBCS {
			return errors.Errorf("writeIndexedToPNGFile: objNr=%d, missing alternative DeviceRGB color space\n", objNr)
		}

		if n == 1 {
			// Gray
			img := image.NewGray(image.Rect(0, 0, w, h))
			i := 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					img.Set(x, y, color.Gray{Y: b[i]})
					i++
				}
			}
			return writeImgToPNG(fileName, img)
		}

		if n == 3 {
			img := image.NewNRGBA(image.Rect(0, 0, w, h))
			i := 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					alpha := uint8(255)
					if sm != nil {
						alpha = sm[y*w+x]
					}
					j := 3 * int(b[i])
					img.Set(x, y, color.NRGBA{R: lookup[j], G: lookup[j+1], B: lookup[j+2], A: alpha})
					i++
				}
			}
			return writeImgToPNG(fileName, img)
		}

		// n == 4 => CMYK
		log.Debug.Printf("writeICCBasedToPNGFile: CMYK objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))
		img := image.NewCMYK(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				j := 4 * int(b[i])
				img.Set(x, y, color.CMYK{C: lookup[j], M: lookup[j+1], Y: lookup[j+2], K: lookup[j+4]})
				i++
			}
		}
		return writeImgToPNG(fileName, img)

	}

	return errors.Errorf("writeIndexedToPNGFile: objNr=%d, unsupported base colorspace %s\n", objNr, csa)
}

func writeIndexedToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, baseCS PDFObject, maxInd int, lookup []byte) error {

	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")
	bpc := *io.ImageDict.IntEntry("BitsPerComponent")

	b := io.Data()
	log.Debug.Printf("writeIndexedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// The image data is a sequence of index values for pixels.
	// Sometimes there is a trailing 0x0A.
	if len(b) < w*h {
		return errors.Errorf("writeIndexedToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// We support 8 bits per component/color only.
	if bpc != 8 {
		return errors.Errorf("writeIndexedToPNGFile: objNr=%d, must be 8 bits per component, got: %d\n", objNr, bpc)
	}

	// Validate the lookup table.
	if len(lookup) != 3*(maxInd+1) {
		return errors.Errorf("writeIndexedToPNGFile: objNr=%d, corrupt lookup table\n", objNr)
	}

	sm, err := softMask(ctx, io.ImageDict, w, h, objNr)
	if err != nil {
		return err
	}

	switch cs := baseCS.(type) {
	case PDFName:
		err = writeIndexedNameCS(cs, objNr, w, h, b, lookup, sm, fileName)

	case PDFArray:
		err = writeIndexedArrayCS(ctx, cs, objNr, w, h, bpc, b, lookup, sm, fileName)
	}

	return err
}

// Identify the color lookup table for an Indexed color space.
func lookup(ctx *PDFContext, o PDFObject) ([]byte, error) {

	var lookup []byte
	var err error

	o, _ = ctx.Dereference(o)

	switch o := o.(type) {

	case PDFStringLiteral:
		lookup = []byte(o.String())

	case PDFHexLiteral:
		lookup, err = o.Bytes()
		if err != nil {
			return nil, err
		}

	case PDFStreamDict:
		lookup, err = streamBytes(&o)
		if err != nil || lookup == nil {
			return nil, err
		}
	}

	return lookup, nil
}

// WritePNGFile creates a PNG file for an image object.
func WritePNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject) error {

	o, _ := io.ImageDict.Find("ColorSpace")
	o, err := ctx.Dereference(o)
	if err != nil {
		return err
	}

	switch cs := o.(type) {

	case PDFName:
		switch cs {

		case DeviceGrayCS:
			err = writeDeviceGrayToPNGFile(ctx, fileName, objNr, io)

		case DeviceRGBCS:
			err = writeDeviceRGBToPNGFile(ctx, fileName, objNr, io)

		case DeviceCMYKCS:
			err = writeDeviceCMYKToPNGFile(ctx, fileName, objNr, io)

		default:
			log.Info.Printf("WritePNGFile: objNr=%d, unsupported name colorspace %s\n", objNr, cs.String())
			err = ErrUnsupportedColorSpace
		}

	case PDFArray:
		csn, _ := cs[0].(PDFName)

		switch csn {

		case CalRGBCS:
			// TODO Buggy? Erste.pdf
			err = writeCalRGBToPNGFile(ctx, fileName, objNr, io)

		case ICCBasedCS:
			iccProfile, _ := ctx.DereferenceStreamDict(cs[1])
			err = writeICCBasedToPNGFile(ctx, fileName, objNr, io, iccProfile)

		case IndexedCS:
			baseCS, _ := ctx.Dereference(cs[1])
			maxInd, _ := ctx.DereferenceInteger(cs[2])

			// Identify the color lookup table.
			var l []byte
			l, err = lookup(ctx, cs[3])
			if err != nil {
				return err
			}

			err = writeIndexedToPNGFile(ctx, fileName, objNr, io, baseCS, maxInd.Value(), l)

		default:
			log.Info.Printf("WritePNGFile: objNr=%d, unsupported array colorspace %s\n", objNr, csn)
			err = ErrUnsupportedColorSpace

		}

	}

	return err
}

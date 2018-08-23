package pdfcpu

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/trussworks/pdfcpu/pkg/filter"
	"github.com/trussworks/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// Errors to be identified.
var (
	ErrUnsupportedColorSpace = errors.New("unsupported color space")
	ErrUnsupportedBPC        = errors.New("unsupported bitsPerComponent")
)

func writeImgToPNG(fileName string, img image.Image) error {

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	//fmt.Println("png written")

	return png.Encode(f, img)
}

func streamBytes(sd *PDFStreamDict) ([]byte, error) {

	fpl := sd.FilterPipeline
	if fpl == nil {
		log.Info.Printf("streamBytes: no filter pipeline\n")
		err := decodeStream(sd)
		if err != nil {
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
		err := decodeStream(sd)
		if err != nil {
			return nil, err
		}

	default:
		log.Debug.Printf("streamBytes: filter not \"Flate\": %s\n", fpl[0].Name)
		return nil, nil
	}

	return sd.Content, nil
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

func writeDeviceGrayToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, bpc, w, h int) error {

	b := io.Data()
	log.Debug.Printf("writeDeviceGrayToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// For streams not using compression there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (bpc*w*h+7)/8 {
		return errors.Errorf("writeDeviceGrayToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	img := image.NewGray(image.Rect(0, 0, w, h))

	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; {
			p := b[i]
			for j := 0; j < 8/bpc; j++ {
				pix := p >> (8 - uint8(bpc))
				img.Set(x, y, color.Gray{Y: pix})
				p <<= uint8(bpc)
				x++
			}
			i++
		}
	}

	return writeImgToPNG(fileName, img)
}

func writeDeviceRGBToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, bpc, w, h int) error {

	b := io.Data()
	log.Debug.Printf("writeDeviceRGBToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (3*bpc*w*h+7)/8 {
		return errors.Errorf("writeDeviceRGBToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	sm, err := softMask(ctx, io.ImageDict, w, h, objNr)
	if err != nil {
		return err
	}

	// TODO Support bpc.
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

func writeDeviceCMYKToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, bpc, w, h int) error {

	b := io.Data()
	log.Debug.Printf("writeDeviceCMYKToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	// Validate buflen.
	// Sometimes there is a trailing 0x0A in addition to the imagebytes.
	if len(b) < (4*bpc*w*h+7)/8 {
		return errors.Errorf("writeDeviceCMYKToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// TODO Support bpc.
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

func writeCalRGBToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, bpc, w, h int) error {

	b := io.Data()
	log.Debug.Printf("writeICCBasedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))

	if len(b) < (3*bpc*w*h+7)/8 {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	// Optional int array "Range", length 2*N specifies min,max values of color components.
	// This information can be validated against the iccProfile.

	// RGB
	// TODO Support bpc, softmask.
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

func writeICCBasedToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, iccProfileStream *PDFStreamDict, bpc, w, h int) error {

	//  Any ICC profile >= ICC.1:2004:10 is sufficient for any PDF version <= 1.7
	//  If the embedded ICC profile version is newer than the one used by the Reader, substitute with Alternate color space.

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
	if len(b) < (n*bpc*w*h+7)/8 {
		return errors.Errorf("writeICCBasedToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
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
		// TODO support bpc.
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
		// TODO support bpc.
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
	// TODO support bpc.
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

func writeIndexedNameCS(cs PDFName, objNr, w, h, bpc, maxInd int, b, lookup, sm []byte, fileName string) error {

	switch cs {

	case DeviceRGBCS:

		// Validate the lookup table.
		if len(lookup) < 3*(maxInd+1) {
			return errors.Errorf("writeIndexedNameCS: objNr=%d, corrupt DeviceRGB lookup table\n", objNr)
		}

		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; {
				p := b[i]
				for j := 0; j < 8/bpc; j++ {
					ind := p >> (8 - uint8(bpc))
					//fmt.Printf("x=%d y=%d i=%d j=%d p=#%02x ind=#%02x\n", x, y, i, j, p, ind)
					alpha := uint8(255)
					if sm != nil {
						alpha = sm[y*w+x]
					}
					l := 3 * int(ind)
					img.Set(x, y, color.NRGBA{R: lookup[l], G: lookup[l+1], B: lookup[l+2], A: alpha})
					p <<= uint8(bpc)
					x++
				}
				i++
			}
		}
		return writeImgToPNG(fileName, img)

	case DeviceCMYKCS:

		// Validate the lookup table.
		if len(lookup) < 4*(maxInd+1) {
			return errors.Errorf("writeIndexedNameCS: objNr=%d, corrupt DeviceCMYK lookup table\n", objNr)
		}

		img := image.NewCMYK(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; {
				p := b[i]
				for j := 0; j < 8/bpc; j++ {
					ind := p >> (8 - uint8(bpc))
					//fmt.Printf("x=%d y=%d i=%d j=%d p=#%02x ind=#%02x\n", x, y, i, j, p, ind)
					l := 4 * int(ind)
					img.Set(x, y, color.CMYK{C: lookup[l], M: lookup[l+1], Y: lookup[l+2], K: lookup[l+3]})
					p <<= uint8(bpc)
					x++
				}
				i++
			}
		}
		return writeImgToPNG(fileName, img)

	}

	log.Info.Printf("writeIndexedToPNGFile: objNr=%d, unsupported base colorspace %s\n", objNr, cs.String())

	return ErrUnsupportedColorSpace
}

func writeIndexedArrayCS(ctx *PDFContext, csa PDFArray, objNr, w, h, bpc, maxInd int, b, lookup, sm []byte, fileName string) error {

	cs, _ := csa[0].(PDFName)

	switch cs {

	case ICCBasedCS:

		iccProfileStream, _ := ctx.DereferenceStreamDict(csa[1])

		// 1,3 or 4 color components.
		n := *iccProfileStream.IntEntry("N")
		if !intMemberOf(n, []int{1, 3, 4}) {
			return errors.Errorf("writeIndexedToPNGFile: objNr=%d, N must be 1,3 or 4, got:%d\n", objNr, n)
		}

		// Validate the lookup table.
		if len(lookup) < n*(maxInd+1) {
			return errors.Errorf("writeIndexedToPNGFile: objNr=%d, corrupt ICCBased lookup table\n", objNr)
		}

		// TODO: Transform linear XYZ to RGB according to ICC profile.
		// var alternateDeviceRGBCS bool
		// o, found := iccProfileStream.Find("Alternate")
		// if found {
		// 	alternateDeviceRGBCS = ensureDeviceRGBCS(ctx, o)
		// }
		// if !alternateDeviceRGBCS {
		// 	return errors.Errorf("writeIndexedToPNGFile: objNr=%d, missing alternative DeviceRGB color space\n", objNr)
		// }

		if n == 1 {
			// => Gray
			// TODO support bpc and softmask.
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
			// => RGB
			img := image.NewNRGBA(image.Rect(0, 0, w, h))
			i := 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; {
					p := b[i]
					for j := 0; j < 8/bpc; j++ {
						ind := p >> (8 - uint8(bpc))
						//fmt.Printf("x=%d y=%d i=%d j=%d p=#%02x ind=#%02x\n", x, y, i, j, p, ind)
						alpha := uint8(255)
						if sm != nil {
							alpha = sm[y*w+x]
						}
						l := 3 * int(ind)
						img.Set(x, y, color.NRGBA{R: lookup[l], G: lookup[l+1], B: lookup[l+2], A: alpha})
						p <<= uint8(bpc)
						x++
					}
					i++
				}
			}
			return writeImgToPNG(fileName, img)
		}

		// n == 4 => CMYK
		// TODO support bpc.
		log.Debug.Printf("writeIndexedArrayCS: CMYK objNr=%d w=%d h=%d bpc=%d buflen=%d\n", objNr, w, h, bpc, len(b))
		img := image.NewCMYK(image.Rect(0, 0, w, h))
		i := 0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				j := 4 * int(b[i])
				img.Set(x, y, color.CMYK{C: lookup[j], M: lookup[j+1], Y: lookup[j+2], K: lookup[j+3]})
				i++
			}
		}
		return writeImgToPNG(fileName, img)

	}

	log.Info.Printf("writeIndexedToPNGFile: objNr=%d, unsupported base colorspace %s\n", objNr, csa)

	return ErrUnsupportedColorSpace

}

func writeIndexedToPNGFile(ctx *PDFContext, fileName string, objNr int, io *ImageObject, baseCS PDFObject, maxInd int, lookup []byte, bpc, w, h int) error {

	b := io.Data()
	log.Debug.Printf("writeIndexedToPNGFile: objNr=%d w=%d h=%d bpc=%d buflen=%d maxInd=%d\n", objNr, w, h, bpc, len(b), maxInd)

	// Validate buflen.
	// The image data is a sequence of index values for pixels.
	// Sometimes there is a trailing 0x0A.
	if len(b) < (bpc*w*h+7)/8 {
		return errors.Errorf("writeIndexedToPNGFile: objNr=%d corrupt image object %v\n", objNr, *io)
	}

	sm, err := softMask(ctx, io.ImageDict, w, h, objNr)
	if err != nil {
		return err
	}

	switch cs := baseCS.(type) {
	case PDFName:
		err = writeIndexedNameCS(cs, objNr, w, h, bpc, maxInd, b, lookup, sm, fileName)

	case PDFArray:
		err = writeIndexedArrayCS(ctx, cs, objNr, w, h, bpc, maxInd, b, lookup, sm, fileName)
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

	bpc := *io.ImageDict.IntEntry("BitsPerComponent")
	w := *io.ImageDict.IntEntry("Width")
	h := *io.ImageDict.IntEntry("Height")

	o, _ := io.ImageDict.Find("ColorSpace")
	o, err := ctx.Dereference(o)
	if err != nil {
		return err
	}

	switch cs := o.(type) {

	case PDFName:
		switch cs {

		case DeviceGrayCS:
			err = writeDeviceGrayToPNGFile(ctx, fileName, objNr, io, bpc, w, h)

		case DeviceRGBCS:
			err = writeDeviceRGBToPNGFile(ctx, fileName, objNr, io, bpc, w, h)

		case DeviceCMYKCS:
			err = writeDeviceCMYKToPNGFile(ctx, fileName, objNr, io, bpc, w, h)

		default:
			log.Info.Printf("WritePNGFile: objNr=%d, unsupported name colorspace %s\n", objNr, cs.String())
			err = ErrUnsupportedColorSpace
		}

	case PDFArray:
		csn, _ := cs[0].(PDFName)

		switch csn {

		case CalRGBCS:
			err = writeCalRGBToPNGFile(ctx, fileName, objNr, io, bpc, w, h)

		case ICCBasedCS:
			iccProfile, _ := ctx.DereferenceStreamDict(cs[1])
			err = writeICCBasedToPNGFile(ctx, fileName, objNr, io, iccProfile, bpc, w, h)

		case IndexedCS:
			baseCS, _ := ctx.Dereference(cs[1])
			maxInd, _ := ctx.DereferenceInteger(cs[2])

			// Identify the color lookup table.
			var l []byte
			l, err = lookup(ctx, cs[3])
			if err != nil {
				return err
			}
			if l == nil {
				return errors.Errorf("WritePNGFile: objNr=%d IndexedCS with corrupt lookup table %s\n", objNr, csn)
			}
			//fmt.Printf("lookup: \n%s\n", hex.Dump(l))

			err = writeIndexedToPNGFile(ctx, fileName, objNr, io, baseCS, maxInd.Value(), l, bpc, w, h)

		default:
			log.Info.Printf("WritePNGFile: objNr=%d, unsupported array colorspace %s\n", objNr, csn)
			err = ErrUnsupportedColorSpace

		}

	}

	return err
}

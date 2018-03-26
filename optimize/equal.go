package optimize

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/read"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func equalPDFObjects(o1, o2 types.PDFObject, ctx *types.PDFContext) (ok bool, err error) {

	o1Type := fmt.Sprintf("%T", o1)
	o2Type := fmt.Sprintf("%T", o2)

	log.Debug.Printf("equalPDFObjects: comparing %s with %s \n", o1Type, o2Type)

	if o1Type != o2Type {
		return false, nil
	}

	o1, err = ctx.Dereference(o1)
	if err != nil {
		return false, err
	}

	o2, err = ctx.Dereference(o2)
	if err != nil {
		return false, err
	}

	switch o1.(type) {

	case types.PDFName, types.PDFStringLiteral, types.PDFHexLiteral,
		types.PDFInteger, types.PDFFloat, types.PDFBoolean:
		ok = o1 == o2

	case types.PDFDict:

		d1 := o1.(types.PDFDict)
		d2 := o2.(types.PDFDict)
		ok, err = equalPDFDicts(&d1, &d2, ctx)

	case types.PDFStreamDict:

		sd1 := o1.(types.PDFStreamDict)
		sd2 := o2.(types.PDFStreamDict)
		ok, err = equalPDFStreamDicts(&sd1, &sd2, ctx)

	case types.PDFArray:

		arr1 := o1.(types.PDFArray)
		arr2 := o2.(types.PDFArray)
		ok, err = equalPDFArrays(&arr1, &arr2, ctx)

	default:
		err = errors.Errorf("equalPDFObjects: unhandled compare for type %s\n", o1Type)
	}

	return ok, err
}

func equalPDFArrays(arr1, arr2 *types.PDFArray, ctx *types.PDFContext) (bool, error) {

	if len(*arr1) != len(*arr2) {
		return false, nil
	}

	for i, o1 := range *arr1 {

		o2 := (*arr2)[i]

		o1r := fmt.Sprintf("%T", o1)
		o2r := fmt.Sprintf("%T", o2)

		if o1r != o2r {
			return false, nil
		}

		ok, err := equalPDFObjects(o1, o2, ctx)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func equalPDFStreamDicts(sd1, sd2 *types.PDFStreamDict, ctx *types.PDFContext) (bool, error) {

	ok, err := equalPDFDicts(&sd1.PDFDict, &sd2.PDFDict, ctx)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil
	}

	encodedStream1, err := read.LoadEncodedStreamContent(ctx, sd1)
	if err != nil {
		return false, err
	}

	encodedStream2, err := read.LoadEncodedStreamContent(ctx, sd2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(encodedStream1, encodedStream2), nil
}

func equalFontNames(v1, v2 types.PDFObject, ctx *types.PDFContext) (bool, error) {

	v1, err := ctx.XRefTable.Dereference(v1)
	if err != nil {
		return false, err
	}
	bf1, ok := v1.(types.PDFName)
	if !ok {
		return false, errors.Errorf("equalFontNames: type cast problem")
	}

	v2, err = ctx.XRefTable.Dereference(v2)
	if err != nil {
		return false, err
	}
	bf2 := v2.(types.PDFName)
	if !ok {
		return false, errors.Errorf("equalFontNames: type cast problem")
	}

	// Ignore fontname prefix
	i := strings.Index(string(bf1), "+")
	if i > 0 {
		bf1 = bf1[i+1:]
	}

	i = strings.Index(string(bf2), "+")
	if i > 0 {
		bf2 = bf2[i+1:]
	}

	log.Debug.Printf("equalFontNames: bf1=%s fb2=%s\n", bf1, bf2)

	return bf1 == bf2, nil
}

func equalPDFDicts(d1, d2 *types.PDFDict, ctx *types.PDFContext) (bool, error) {

	log.Debug.Printf("equalPDFDicts: %v\n%v\n", d1, d2)

	if len(d1.Dict) != len(d2.Dict) {
		return false, nil
	}

	for key, v1 := range d1.Dict {

		v2, found := d2.Dict[key]
		if !found {
			log.Debug.Printf("equalPDFDict: return false, key=%s\n", key)
			return false, nil
		}

		// Special treatment for font dicts
		if key == "BaseFont" || key == "FontName" || key == "Name" {

			ok, err := equalFontNames(v1, v2, ctx)
			if err != nil {
				log.Debug.Printf("equalPDFDict: return2 false, key=%s v1=%v\nv2=%v\n", key, v1, v2)
				return false, err
			}

			if !ok {
				log.Debug.Printf("equalPDFDict: return3 false, key=%s v1=%v\nv2=%v\n", key, v1, v2)
				return false, nil
			}

			continue
		}

		ok, err := equalPDFObjects(v1, v2, ctx)
		if err != nil {
			log.Debug.Printf("equalPDFDict: return4 false, key=%s v1=%v\nv2=%v\n%v\n", key, v1, v2, err)
			return false, err
		}

		if !ok {
			log.Debug.Printf("equalPDFDict: return5 false, key=%s v1=%v\nv2=%v\n", key, v1, v2)
			return false, nil
		}

	}

	log.Debug.Println("equalPDFDict: return true")

	return true, nil
}

func equalFontDicts(fd1, fd2 *types.PDFDict, ctx *types.PDFContext) (bool, error) {

	log.Debug.Printf("equalFontDicts: %v\n%v\n", fd1, fd2)

	if fd1 == fd2 {
		return true, nil
	}

	if fd1 == nil {
		return fd2 == nil, nil
	}

	if fd2 == nil {
		return false, nil
	}

	ok, err := equalPDFDicts(fd1, fd2, ctx)
	if err != nil {
		return false, err
	}

	return ok, nil
}

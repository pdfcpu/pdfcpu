package optimize

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/EndFirstCorp/pdflib/read"
	"github.com/EndFirstCorp/pdflib/types"
	"github.com/pkg/errors"
)

func equalPDFObjects(o1, o2 interface{}, ctx *types.PDFContext) (bool, error) {

	o1Type := fmt.Sprintf("%T", o1)
	o2Type := fmt.Sprintf("%T", o2)

	if o1Type != o2Type {
		return false, nil
	}

	switch o1.(type) {

	case types.PDFName, types.PDFStringLiteral, types.PDFHexLiteral,
		types.PDFInteger, types.PDFFloat, types.PDFBoolean:
		if o1 != o2 {
			return false, nil
		}

	case types.PDFDict:

		d1 := o1.(types.PDFDict)
		d2 := o2.(types.PDFDict)

		ok, err := equalPDFDicts(d1, d2, ctx)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}

	case types.PDFStreamDict:

		sd1 := o1.(types.PDFStreamDict)
		sd2 := o2.(types.PDFStreamDict)

		ok, err := equalPDFStreamDicts(&sd1, &sd2, ctx)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}

	case types.PDFArray:

		arr1 := o1.(types.PDFArray)
		arr2 := o2.(types.PDFArray)

		ok, err := equalPDFArrays(arr1, arr2, ctx)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}

	case types.PDFIndirectRef:

		ir1 := o1.(types.PDFIndirectRef)
		o1, err := ctx.XRefTable.Dereference(ir1)
		if err != nil {
			return false, err
		}

		ir2 := o2.(types.PDFIndirectRef)
		o2, err := ctx.XRefTable.Dereference(ir2)
		if err != nil {
			return false, err
		}

		ok, err := equalPDFObjects(o1, o2, ctx)
		if err != nil {
			return false, nil
		}
		if !ok {
			return false, nil
		}

	default:
		return false, errors.Errorf("equalPDFObjects: unhandled compare for type %s\n", o1Type)
	}

	return true, nil
}

func equalPDFArrays(arr1, arr2 types.PDFArray, ctx *types.PDFContext) (bool, error) {

	if len(arr1) != len(arr2) {
		return false, nil
	}

	for i, o1 := range arr1 {

		o2 := arr2[i]

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

	ok, err := equalPDFDicts(sd1.PDFDict, sd2.PDFDict, ctx)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil
	}

	encodedStream1, err := read.GetEncodedStreamContent(ctx, sd1)
	if err != nil {
		return false, err
	}

	encodedStream2, err := read.GetEncodedStreamContent(ctx, sd2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(encodedStream1, encodedStream2), nil
}

func equalFontNames(v1, v2 interface{}, ctx *types.PDFContext) (bool, error) {

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

	return bf1 != bf2, nil
}

func equalPDFDicts(d1, d2 types.PDFDict, ctx *types.PDFContext) (bool, error) {

	if len(d1.Dict) != len(d2.Dict) {
		return false, nil
	}

	for key, v1 := range d1.Dict {

		v2, found := d2.Dict[key]
		if !found {
			return false, nil
		}

		// Special treatment for font dicts
		if key == "BaseFont" || key == "FontName" || key == "Name" {

			ok, err := equalFontNames(v1, v2, ctx)
			if err != nil {
				return false, err
			}

			if !ok {
				return false, nil
			}

			continue
		}

		ok, err := equalPDFObjects(v1, v2, ctx)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}

	}

	return true, nil
}

func equalFontDicts(fd1, fd2 *types.PDFDict, ctx *types.PDFContext) (bool, error) {

	if fd1 == fd2 {
		return true, nil
	}

	if fd1 == nil {
		return fd2 == nil, nil
	}

	if fd2 == nil {
		return false, nil
	}

	ok, err := equalPDFDicts(*fd1, *fd2, ctx)
	if err != nil {
		return false, err
	}

	return ok, nil
}

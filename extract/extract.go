// Package extract provides functions for extracting fonts, images, pages and page content.
package extract

import (
	"strings"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/optimize"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// ImageData extracts image data for objNr.
// Supported imgTypes: DCTDecode, JPXDecode
func ImageData(ctx *types.PDFContext, objNr int) (*types.ImageObject, error) {

	imageObj := ctx.Optimize.ImageObjects[objNr]

	imageDict := imageObj.ImageDict

	fpl := imageDict.FilterPipeline
	if fpl == nil {
		return nil, nil
	}

	var s []string
	for _, filter := range fpl {
		s = append(s, filter.Name)
	}
	filters := strings.Join(s, ",")

	// Ignore filter chains with length > 1
	if len(fpl) > 1 {
		log.Info.Printf("ImageData: ignore obj# %d, more than 1 filter:%s\n", objNr, filters)
		return nil, nil
	}

	// Ignore imageMasks
	if im := imageDict.BooleanEntry("ImageMask"); im != nil && *im {
		log.Info.Printf("ImageData: ignore obj# %d, imageMask\n", objNr)
		return nil, nil
	}

	switch fpl[0].Name {

	case "DCTDecode":
		imageObj.Extension = "jpg"

	case "JPXDecode":
		imageObj.Extension = "jpx"

	default:
		log.Debug.Printf("ImageData: ignore obj# %d filter neither \"DCTDecode\" nor \"JPXDecode\"\n%s", objNr, filters)
		return nil, nil
	}

	return imageObj, nil
}

// FontData extracts font data (the "fontfile") for objNr.
// Supported fontTypes: TrueType
func FontData(ctx *types.PDFContext, objNr int) (*types.FontObject, error) {

	fontObject := ctx.Optimize.FontObjects[objNr]

	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		log.Debug.Printf("FontData: ignoring obj#%d - non embedded font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	dict, err := optimize.FontDescriptor(ctx.XRefTable, fontObject.FontDict, objNr)
	if err != nil {
		return nil, err
	}

	if dict == nil {
		log.Debug.Printf("FontData: ignoring obj#%d - no fontDescriptor available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	indRef := optimize.FontDescriptorFontFileIndirectObjectRef(dict)
	if indRef == nil {
		log.Debug.Printf("FontData: ignoring obj#%d - no font file available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	fontType := fontObject.SubType()

	switch fontType {

	case "TrueType":
		// ttf ... true type file
		// ttc ... true type collection
		// This is just me guessing..
		sd, err := ctx.DereferenceStreamDict(*indRef)
		if err != nil {
			return nil, err
		}
		if sd == nil {
			return nil, errors.Errorf("FontData: corrupt font obj#%d for font: %s\n", objNr, fontObject.FontName)
		}

		// Decode streamDict if used filter is supported only.
		err = filter.DecodeStream(sd)
		if err == filter.ErrUnsupportedFilter {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		fontObject.Data = sd.Content
		fontObject.Extension = "ttf"

	default:
		log.Info.Printf("FontData: ignoring obj#%d - unsupported fonttype %s -  font: %s\n", objNr, fontType, fontObject.FontName)
		return nil, nil
	}

	return fontObject, nil
}

// ContentData extracts page content in PDF notation for objNr.
func ContentData(ctx *types.PDFContext, objNr int) (data []byte, err error) {

	// Get object for objNr.
	obj, err := ctx.FindObject(objNr)
	if err != nil {
		return nil, err
	}

	// Content stream must be a stream dict.
	sd, err := ctx.DereferenceStreamDict(obj)
	if err != nil {
		return nil, err
	}

	// Decode streamDict for supported filters only.
	err = filter.DecodeStream(sd)
	if err == filter.ErrUnsupportedFilter {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return sd.Content, nil
}

// TextData extracts text out of the page content for objNr.
// func TextData(ctx *types.PDFContext, objNr int) (data []byte, err error) {
// 	// TODO
// 	return nil, nil
// }

package pdfcpu

import (
	"strings"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

// extractImageData extracts image data for objNr.
// Supported imgTypes: DCTDecode, JPXDecode
// Note: This is a naive implementation that just returns encoded image bytes.
// Hence TODO: Implementation and usage of these filters: DCTDecode and JPXDecode.
func extractImageData(ctx *PDFContext, objNr int) (*ImageObject, error) {

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
		log.Info.Printf("extractImageData: ignore obj# %d, more than 1 filter:%s\n", objNr, filters)
		return nil, nil
	}

	// Ignore imageMasks
	if im := imageDict.BooleanEntry("ImageMask"); im != nil && *im {
		log.Info.Printf("extractImageData: ignore obj# %d, imageMask\n", objNr)
		return nil, nil
	}

	switch fpl[0].Name {

	case "DCTDecode":
		imageObj.Extension = "jpg"

	case "JPXDecode":
		imageObj.Extension = "jpx"

	default:
		log.Debug.Printf("extractImageData: ignore obj# %d filter neither \"DCTDecode\" nor \"JPXDecode\"\n%s", objNr, filters)
		return nil, nil
	}

	return imageObj, nil
}

// extractFontData extracts font data (the "fontfile") for objNr.
// Supported fontTypes: TrueType
func extractFontData(ctx *PDFContext, objNr int) (*FontObject, error) {

	fontObject := ctx.Optimize.FontObjects[objNr]

	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		log.Debug.Printf("extractFontData: ignoring obj#%d - non embedded font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	dict, err := fontDescriptor(ctx.XRefTable, fontObject.FontDict, objNr)
	if err != nil {
		return nil, err
	}

	if dict == nil {
		log.Debug.Printf("extractFontData: ignoring obj#%d - no fontDescriptor available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	indRef := fontDescriptorFontFileIndirectObjectRef(dict)
	if indRef == nil {
		log.Debug.Printf("extractFontData: ignoring obj#%d - no font file available for font: %s\n", objNr, fontObject.FontName)
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
			return nil, errors.Errorf("extractFontData: corrupt font obj#%d for font: %s\n", objNr, fontObject.FontName)
		}

		// Decode streamDict if used filter is supported only.
		err = encodeStream(sd)
		if err == filter.ErrUnsupportedFilter {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		fontObject.Data = sd.Content
		fontObject.Extension = "ttf"

	default:
		log.Info.Printf("extractFontData: ignoring obj#%d - unsupported fonttype %s -  font: %s\n", objNr, fontType, fontObject.FontName)
		return nil, nil
	}

	return fontObject, nil
}

// ContentData extracts page content in PDF notation for objNr.
func extractContentData(ctx *PDFContext, objNr int) (data []byte, err error) {

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
	err = decodeStream(sd)
	if err == filter.ErrUnsupportedFilter {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return sd.Content, nil
}

// TextData extracts text out of the page content for objNr.
// func TextData(ctx *PDFContext, objNr int) (data []byte, err error) {
// 	// TODO
// 	return nil, nil
// }

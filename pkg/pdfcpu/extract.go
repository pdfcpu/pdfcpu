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
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// ExtractImageData extracts image data for objNr.
// Supported imgTypes: FlateDecode, DCTDecode, JPXDecode
// TODO: Implementation and usage of these filters: DCTDecode and JPXDecode.
// TODO: Should an error be returned instead of nil, nil when filters are not supported?
func ExtractImageData(ctx *Context, objNr int) (*ImageObject, error) {

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

	f := fpl[0].Name

	// We do not extract imageMasks with the exception of CCITTDecoded images
	if im := imageDict.BooleanEntry("ImageMask"); im != nil && *im {
		if f != filter.CCITTFax {
			log.Info.Printf("extractImageData: ignore obj# %d, imageMask\n", objNr)
			return nil, nil
		}
	}

	// Ignore if image has a soft mask defined.
	// if sm, _ := imageDict.Find("SMask"); sm != nil {
	// 	log.Info.Printf("extractImageData: ignore obj# %d, unsupported \"SMask\"\n", objNr)
	// 	return nil, nil
	// }

	// Ignore if image has a Mask defined.
	if sm, _ := imageDict.Find("Mask"); sm != nil {
		log.Info.Printf("extractImageData: ignore obj# %d, unsupported \"Mask\"\n", objNr)
		return nil, nil
	}

	// CCITTDecoded images sometimes don't have a ColorSpace attribute.
	if f == filter.CCITTFax {
		_, err := ctx.DereferenceDictEntry(imageDict.Dict, "ColorSpace")
		if err != nil {
			imageDict.InsertName("ColorSpace", DeviceGrayCS)
		}
	}

	switch f {

	case filter.Flate, filter.CCITTFax:
		// If color space is CMYK then write .tif else write .png
		err := decodeStream(imageDict)
		if err != nil {
			return nil, err
		}

	case filter.DCT:
		//imageObj.Extension = "jpg"

	case filter.JPX:
		//imageObj.Extension = "jpx"

	default:
		log.Debug.Printf("extractImageData: ignore obj# %d filter %s unsupported\n", objNr, filters)
		return nil, nil
	}

	return imageObj, nil
}

// ExtractFontData extracts font data (the "fontfile") for objNr.
// Supported fontTypes: TrueType
func ExtractFontData(ctx *Context, objNr int) (*FontObject, error) {

	fontObject := ctx.Optimize.FontObjects[objNr]

	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		log.Debug.Printf("extractFontData: ignoring obj#%d - non embedded font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	d, err := fontDescriptor(ctx.XRefTable, fontObject.FontDict, objNr)
	if err != nil {
		return nil, err
	}

	if d == nil {
		log.Debug.Printf("extractFontData: ignoring obj#%d - no fontDescriptor available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	ir := fontDescriptorFontFileIndirectObjectRef(d)
	if ir == nil {
		log.Debug.Printf("extractFontData: ignoring obj#%d - no font file available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	fontType := fontObject.SubType()

	switch fontType {

	case "TrueType":
		// ttf ... true type file
		// ttc ... true type collection
		// This is just me guessing..
		sd, err := ctx.DereferenceStreamDict(*ir)
		if err != nil {
			return nil, err
		}
		if sd == nil {
			return nil, errors.Errorf("extractFontData: corrupt font obj#%d for font: %s\n", objNr, fontObject.FontName)
		}

		// Decode streamDict if used filter is supported only.
		err = decodeStream(sd)
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

// ExtractStreamData extracts the content of a stream dict for a specific objNr.
func ExtractStreamData(ctx *Context, objNr int) (data []byte, err error) {

	// Get object for objNr.
	o, err := ctx.FindObject(objNr)
	if err != nil {
		return nil, err
	}

	sd, err := ctx.DereferenceStreamDict(o)
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
// func TextData(ctx *Context, objNr int) (data []byte, err error) {
// 	// TODO
// 	return nil, nil
// }

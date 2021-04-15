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
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// Image is a Reader representing an image resource.
type Image struct {
	io.Reader
	PageNr int
	Name   string // Resource name
	Type   string // File type
}

// ImageObjNrs returns all image dict objNrs for pageNr.
// Requires an optimized context.
func (ctx *Context) ImageObjNrs(pageNr int) []int {
	// TODO Exclude SMask image objects.
	objNrs := []int{}
	for k, v := range ctx.Optimize.PageImages[pageNr-1] {
		if v {
			objNrs = append(objNrs, k)
		}
	}
	return objNrs
}

// ExtractImage extracts an image from image dict referenced by objNr.
// Supported imgTypes: FlateDecode, DCTDecode, JPXDecode
func (ctx *Context) ExtractImage(objNr int) (*Image, error) {
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
		log.Info.Printf("ExtractImage(%d): skip img with more than 1 filter: %s\n", objNr, filters)
		return nil, nil
	}

	f := fpl[0].Name

	// We do not extract imageMasks with the exception of CCITTDecoded images
	if im := imageDict.BooleanEntry("ImageMask"); im != nil && *im {
		if f != filter.CCITTFax {
			log.Info.Printf("ExtractImage(%d): skip img with imageMask\n", objNr)
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
		log.Info.Printf("ExtractImage(%d): skip image, unsupported \"Mask\"\n", objNr)
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
		if err := imageDict.Decode(); err != nil {
			return nil, err
		}

	case filter.DCT:
		//imageObj.Extension = "jpg"

	case filter.JPX:
		//imageObj.Extension = "jpx"

	default:
		log.Debug.Printf("ExtractImage(%d): skip img, filter %s unsupported\n", objNr, filters)
		return nil, nil
	}

	resourceName := imageObj.ResourceNames[0]
	return RenderImage(ctx.XRefTable, imageDict, resourceName, objNr)
}

// ExtractPageImages extracts all images used by pageNr.
func (ctx *Context) ExtractPageImages(pageNr int) ([]Image, error) {
	ii := []Image{}
	for _, objNr := range ctx.ImageObjNrs(pageNr) {
		i, err := ctx.ExtractImage(objNr)
		if err != nil {
			return nil, err
		}
		if i != nil {
			i.PageNr = pageNr
			ii = append(ii, *i)
		}
	}
	return ii, nil
}

// Font is a Reader representing an embedded font.
type Font struct {
	io.Reader
	Name string
	Type string
}

// FontObjNrs returns all font dict objNrs for pageNr.
// Requires an optimized context.
func (ctx *Context) FontObjNrs(pageNr int) []int {
	objNrs := []int{}
	for k, v := range ctx.Optimize.PageFonts[pageNr-1] {
		if v {
			objNrs = append(objNrs, k)
		}
	}
	return objNrs
}

// ExtractFont extracts a font from font dict by objNr.
func (ctx *Context) ExtractFont(objNr int) (*Font, error) {
	fontObject := ctx.Optimize.FontObjects[objNr]

	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		log.Debug.Printf("ExtractFont: ignoring obj#%d - non embedded font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	d, err := fontDescriptor(ctx.XRefTable, fontObject.FontDict, objNr)
	if err != nil {
		return nil, err
	}

	if d == nil {
		log.Debug.Printf("ExtractFont: ignoring obj#%d - no fontDescriptor available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	ir := fontDescriptorFontFileIndirectObjectRef(d)
	if ir == nil {
		log.Debug.Printf("ExtractFont: ignoring obj#%d - no font file available for font: %s\n", objNr, fontObject.FontName)
		return nil, nil
	}

	var f *Font

	fontType := fontObject.SubType()

	switch fontType {

	case "TrueType":
		// ttf ... true type file
		// ttc ... true type collection
		sd, _, err := ctx.DereferenceStreamDict(*ir)
		if err != nil {
			return nil, err
		}
		if sd == nil {
			return nil, errors.Errorf("extractFontData: corrupt font obj#%d for font: %s\n", objNr, fontObject.FontName)
		}

		// Decode streamDict if used filter is supported only.
		err = sd.Decode()
		if err == filter.ErrUnsupportedFilter {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}

		f = &Font{bytes.NewReader(sd.Content), fontObject.FontName, "ttf"}

	default:
		log.Info.Printf("extractFontData: ignoring obj#%d - unsupported fonttype %s -  font: %s\n", objNr, fontType, fontObject.FontName)
		return nil, nil
	}

	return f, nil
}

// ExtractPageFonts extracts all fonts used by pageNr.
func (ctx *Context) ExtractPageFonts(pageNr int) ([]Font, error) {
	ff := []Font{}
	for _, i := range ctx.FontObjNrs(pageNr) {
		f, err := ctx.ExtractFont(i)
		if err != nil {
			return nil, err
		}
		if f != nil {
			ff = append(ff, *f)
		}
	}
	return ff, nil
}

// ExtractPage extracts pageNr into a new single page context.
func (ctx *Context) ExtractPage(pageNr int) (*Context, error) {
	return ctx.ExtractPages([]int{pageNr}, false)
}

// ExtractPages extracts pageNrs into a new single page context.
func (ctx *Context) ExtractPages(pageNrs []int, usePgCache bool) (*Context, error) {
	ctxDest, err := CreateContextWithXRefTable(nil, PaperSize["A4"])
	if err != nil {
		return nil, err
	}

	if err := AddPages(ctx, ctxDest, pageNrs, usePgCache); err != nil {
		return nil, err
	}

	return ctxDest, nil
}

// ExtractPageContent extracts the consolidated page content stream for pageNr.
func (ctx *Context) ExtractPageContent(pageNr int) (io.Reader, error) {
	consolidateRes := false
	d, _, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return nil, err
	}
	bb, err := ctx.PageContent(d)
	if err != nil && err != errNoContent {
		return nil, err
	}
	return bytes.NewReader(bb), nil
}

// Metadata is a Reader representing a metadata dict.
type Metadata struct {
	io.Reader          // metadata
	ObjNr       int    // metadata dict objNr
	ParentObjNr int    // container object number
	ParentType  string // container dict type
}

func extractMetadataFromDict(ctx *Context, d Dict, parentObjNr int) (*Metadata, error) {
	o, found := d.Find("Metadata")
	if !found || o == nil {
		return nil, nil
	}
	sd, _, err := ctx.DereferenceStreamDict(o)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		return nil, nil
	}
	// Get metadata dict object number.
	ir, _ := o.(IndirectRef)
	mdObjNr := ir.ObjectNumber.Value()
	// Get container dict type.
	dt := "unknown"
	if d.Type() != nil {
		dt = *d.Type()
	}
	// Decode streamDict for supported filters only.
	if err = sd.Decode(); err == filter.ErrUnsupportedFilter {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Metadata{bytes.NewReader(sd.Content), mdObjNr, parentObjNr, dt}, nil
}

// ExtractMetadata returns all metadata of ctx.
func (ctx *Context) ExtractMetadata() ([]Metadata, error) {
	mm := []Metadata{}
	for k, v := range ctx.Table {
		if v.Free || v.Compressed {
			continue
		}
		switch d := v.Object.(type) {
		case Dict:
			md, err := extractMetadataFromDict(ctx, d, k)
			if err != nil {
				return nil, err
			}
			if md == nil {
				continue
			}
			mm = append(mm, *md)

		case StreamDict:
			md, err := extractMetadataFromDict(ctx, d.Dict, k)
			if err != nil {
				return nil, err
			}
			if md == nil {
				continue
			}
			mm = append(mm, *md)
		}
	}
	return mm, nil
}

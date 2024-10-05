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
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ImageObjNrs returns all image dict objNrs for pageNr.
// Requires an optimized context.
func ImageObjNrs(ctx *model.Context, pageNr int) []int {
	// TODO Exclude SMask image objects.
	objNrs := []int{}

	if pageNr < 1 {
		return objNrs
	}

	imgObjNrs := ctx.Optimize.PageImages
	if len(imgObjNrs) == 0 {
		return objNrs
	}

	pageImgObjNrs := imgObjNrs[pageNr-1]
	if pageImgObjNrs == nil {
		return objNrs
	}

	for k, v := range pageImgObjNrs {
		if v {
			objNrs = append(objNrs, k)
		}
	}
	return objNrs
}

// StreamLength returns sd's stream length.
func StreamLength(ctx *model.Context, sd *types.StreamDict) (int64, error) {
	if val := sd.Int64Entry("Length"); val != nil {
		return *val, nil
	}

	indRef := sd.IndirectRefEntry("Length")
	if indRef == nil {
		return 0, nil
	}

	i, err := ctx.DereferenceInteger(*indRef)
	if err != nil || i == nil {
		return 0, err
	}

	return int64(*i), nil
}

// ColorSpaceString returns a string representation for sd's colorspace.
func ColorSpaceString(ctx *model.Context, sd *types.StreamDict) (string, error) {
	o, found := sd.Find("ColorSpace")
	if !found {
		return "", nil
	}

	o, err := ctx.Dereference(o)
	if err != nil {
		return "", err
	}

	switch cs := o.(type) {

	case types.Name:
		return string(cs), nil

	case types.Array:
		return string(cs[0].(types.Name)), nil
	}

	return "", nil
}

func colorSpaceNameComponents(cs types.Name) int {
	switch cs {

	case model.DeviceGrayCS:
		return 1

	case model.DeviceRGBCS:
		return 3

	case model.DeviceCMYKCS:
		return 4
	}

	return 0
}

func indexedColorSpaceComponents(xRefTable *model.XRefTable, cs types.Array) (int, error) {
	baseCS, err := xRefTable.Dereference(cs[1])
	if err != nil {
		return 0, err
	}

	switch cs := baseCS.(type) {
	case types.Name:
		return colorSpaceNameComponents(cs), nil

	case types.Array:
		switch cs[0].(types.Name) {

		case model.CalGrayCS:
			return 1, nil

		case model.CalRGBCS:
			return 3, nil

		case model.LabCS:
			return 3, nil

		case model.ICCBasedCS:
			iccProfileStream, _, err := xRefTable.DereferenceStreamDict(cs[1])
			if err != nil {
				return 0, err
			}
			n := iccProfileStream.IntEntry("N")
			i := 0
			if n != nil {
				i = *n
			}
			return i, nil

		case model.SeparationCS:
			return 1, nil

		case model.DeviceNCS:
			return len(cs[1].(types.Array)), nil
		}
	}

	return 0, nil
}

// ColorSpaceComponents returns the corresponding number of used color components for sd's colorspace.
func ColorSpaceComponents(xRefTable *model.XRefTable, sd *types.StreamDict) (int, error) {
	o, found := sd.Find("ColorSpace")
	if !found {
		return 0, nil
	}

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return 0, err
	}

	switch cs := o.(type) {
	case types.Name:
		return colorSpaceNameComponents(cs), nil

	case types.Array:
		switch cs[0].(types.Name) {

		case model.CalGrayCS:
			return 1, nil

		case model.CalRGBCS:
			return 3, nil

		case model.LabCS:
			return 3, nil

		case model.ICCBasedCS:
			iccProfileStream, _, err := xRefTable.DereferenceStreamDict(cs[1])
			if err != nil {
				return 0, err
			}
			n := iccProfileStream.IntEntry("N")
			i := 0
			if n != nil {
				i = *n
			}
			return i, nil

		case model.SeparationCS:
			return 1, nil

		case model.DeviceNCS:
			return len(cs[1].(types.Array)), nil

		case model.IndexedCS:
			return indexedColorSpaceComponents(xRefTable, cs)

		}
	}

	return 0, nil
}

func imageWidth(ctx *model.Context, sd *types.StreamDict, objNr int) (int, error) {
	obj, ok := sd.Find("Width")
	if !ok {
		return 0, errors.Errorf("pdfcpu: missing image width obj#%d", objNr)
	}
	i, err := ctx.DereferenceInteger(obj)
	if err != nil {
		return 0, err
	}
	return i.Value(), nil
}

func imageHeight(ctx *model.Context, sd *types.StreamDict, objNr int) (int, error) {
	obj, ok := sd.Find("Height")
	if !ok {
		return 0, errors.Errorf("pdfcpu: missing image height obj#%d", objNr)
	}
	i, err := ctx.DereferenceInteger(obj)
	if err != nil {
		return 0, err
	}
	return i.Value(), nil
}

func imageStub(
	ctx *model.Context,
	sd *types.StreamDict,
	resourceId, filters, lastFilter string,
	decodeParms types.Dict,
	thumb, imgMask bool,
	objNr int) (*model.Image, error) {

	w, err := imageWidth(ctx, sd, objNr)
	if err != nil {
		return nil, err
	}

	h, err := imageHeight(ctx, sd, objNr)
	if err != nil {
		return nil, err
	}

	cs, err := ColorSpaceString(ctx, sd)
	if err != nil {
		return nil, err
	}

	comp, err := ColorSpaceComponents(ctx.XRefTable, sd)
	if err != nil {
		return nil, err
	}
	if lastFilter == filter.CCITTFax {
		comp = 1
	}

	bpc := 0
	if i := sd.IntEntry("BitsPerComponent"); i != nil {
		bpc = *i
	}
	// if jpx, bpc is undefined
	if imgMask {
		bpc = 1
	}

	var sMask bool
	if sm, _ := sd.Find("SMask"); sm != nil {
		sMask = true
	}

	var mask bool
	if sm, _ := sd.Find("Mask"); sm != nil {
		mask = true
	}

	var interpol bool
	if b := sd.BooleanEntry("Interpolate"); b != nil && *b {
		interpol = true
	}

	size, err := StreamLength(ctx, sd)
	if err != nil {
		return nil, err
	}

	var s string
	if decodeParms != nil {
		s = decodeParms.String()
	}

	img := &model.Image{
		ObjNr:       objNr,
		Name:        resourceId,
		Thumb:       thumb,
		IsImgMask:   imgMask,
		HasImgMask:  mask,
		HasSMask:    sMask,
		Width:       w,
		Height:      h,
		Cs:          cs,
		Comp:        comp,
		Bpc:         bpc,
		Interpol:    interpol,
		Size:        size,
		Filter:      filters,
		DecodeParms: s,
	}

	return img, nil
}

func prepareExtractImage(sd *types.StreamDict) (string, string, types.Dict, bool) {
	var imgMask bool
	if im := sd.BooleanEntry("ImageMask"); im != nil && *im {
		imgMask = true
	}

	var (
		filters    string
		lastFilter string
		d          types.Dict
	)

	fpl := sd.FilterPipeline
	if fpl != nil {
		var s []string
		for _, filter := range fpl {
			s = append(s, filter.Name)
			lastFilter = filter.Name
			if filter.DecodeParms != nil {
				d = filter.DecodeParms
			}
		}
		filters = strings.Join(s, ",")
	}

	return filters, lastFilter, d, imgMask
}
func decodeImage(ctx *model.Context, sd *types.StreamDict, filters, lastFilter string, objNr int) error {
	// CCITTDecoded images / (bit) masks don't have a ColorSpace attribute, but we render image files.
	if lastFilter == filter.CCITTFax {
		if _, err := ctx.DereferenceDictEntry(sd.Dict, "ColorSpace"); err != nil {
			sd.InsertName("ColorSpace", model.DeviceGrayCS)
		}
	}

	if lastFilter == filter.DCT {
		comp, err := ColorSpaceComponents(ctx.XRefTable, sd)
		if err != nil {
			return err
		}
		sd.CSComponents = comp
	}

	switch lastFilter {

	case filter.DCT, filter.JPX, filter.Flate, filter.LZW, filter.CCITTFax, filter.RunLength:
		if err := sd.Decode(); err != nil {
			return err
		}

	default:
		msg := fmt.Sprintf("pdfcpu: ExtractImage(obj#%d): skipping img, filter %s unsupported", objNr, filters)
		if log.DebugEnabled() {
			log.Debug.Println(msg)
		}
		if log.CLIEnabled() {
			log.CLI.Println(msg)
		}
		return nil
	}

	return nil
}

func img(
	ctx *model.Context,
	sd *types.StreamDict,
	thumb, imgMask bool,
	resourceID, filters, lastFilter string,
	objNr int) (*model.Image, error) {

	if sd.FilterPipeline == nil {
		sd.Content = sd.Raw
	} else {
		if err := decodeImage(ctx, sd, filters, lastFilter, objNr); err != nil {
			return nil, err
		}
	}

	r, t, err := RenderImage(ctx.XRefTable, sd, thumb, resourceID, objNr)
	if err != nil {
		return nil, err
	}

	img := &model.Image{
		Reader:   r,
		Name:     resourceID,
		ObjNr:    objNr,
		Thumb:    thumb,
		FileType: t,
	}

	return img, nil
}

// ExtractImage extracts an image from sd.
func ExtractImage(ctx *model.Context, sd *types.StreamDict, thumb bool, resourceID string, objNr int, stub bool) (*model.Image, error) {
	if sd == nil {
		return nil, nil
	}

	filters, lastFilter, decodeParms, imgMask := prepareExtractImage(sd)

	if stub {
		return imageStub(ctx, sd, resourceID, filters, lastFilter, decodeParms, thumb, imgMask, objNr)
	}

	return img(ctx, sd, thumb, imgMask, resourceID, filters, lastFilter, objNr)
}

// ExtractPageImages extracts all images used by pageNr.
// Optionally return stubs only.
func ExtractPageImages(ctx *model.Context, pageNr int, stub bool) (map[int]model.Image, error) {
	m := map[int]model.Image{}
	for _, objNr := range ImageObjNrs(ctx, pageNr) {
		imageObj := ctx.Optimize.ImageObjects[objNr]
		img, err := ExtractImage(ctx, imageObj.ImageDict, false, imageObj.ResourceNames[pageNr-1], objNr, stub)
		if err != nil {
			return nil, err
		}
		if img != nil {
			img.PageNr = pageNr
			m[objNr] = *img
		}
	}
	// Extract thumbnail for pageNr
	if indRef, ok := ctx.PageThumbs[pageNr]; ok {
		objNr := indRef.ObjectNumber.Value()
		sd, _, err := ctx.DereferenceStreamDict(indRef)
		if err != nil {
			return nil, err
		}
		img, err := ExtractImage(ctx, sd, true, "", objNr, stub)
		if err != nil {
			return nil, err
		}
		if img != nil {
			img.PageNr = pageNr
			m[objNr] = *img
		}
	}
	return m, nil
}

// Font is a Reader representing an embedded font.
type Font struct {
	io.Reader
	Name string
	Type string
}

// FontObjNrs returns all font dict objNrs for pageNr.
// Requires an optimized context.
func FontObjNrs(ctx *model.Context, pageNr int) []int {
	objNrs := []int{}

	if pageNr < 1 {
		return objNrs
	}

	fontObjNrs := ctx.Optimize.PageFonts
	if len(fontObjNrs) == 0 {
		return objNrs
	}

	pageFontObjNrs := fontObjNrs[pageNr-1]
	if pageFontObjNrs == nil {
		return objNrs
	}

	for k, v := range pageFontObjNrs {
		if v {
			objNrs = append(objNrs, k)
		}
	}
	return objNrs
}

// ExtractFont extracts a font from fontObject.
func ExtractFont(ctx *model.Context, fontObject model.FontObject, objNr int) (*Font, error) {
	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		if log.DebugEnabled() {
			log.Debug.Printf("ExtractFont: ignoring obj#%d - non embedded font: %s\n", objNr, fontObject.FontName)
		}
		return nil, nil
	}

	d, err := fontDescriptor(ctx.XRefTable, fontObject.FontDict, objNr)
	if err != nil {
		return nil, err
	}

	if d == nil {
		if log.DebugEnabled() {
			log.Debug.Printf("ExtractFont: ignoring obj#%d - no fontDescriptor available for font: %s\n", objNr, fontObject.FontName)
		}
		return nil, nil
	}

	ir := fontDescriptorFontFileIndirectObjectRef(d)
	if ir == nil {
		if log.DebugEnabled() {
			log.Debug.Printf("ExtractFont: ignoring obj#%d - no font file available for font: %s\n", objNr, fontObject.FontName)
		}
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
		if log.InfoEnabled() {
			log.Info.Printf("extractFontData: ignoring obj#%d - unsupported fonttype %s -  font: %s\n", objNr, fontType, fontObject.FontName)
		}
		return nil, nil
	}

	return f, nil
}

// ExtractPageFonts extracts all fonts used by pageNr.
func ExtractPageFonts(ctx *model.Context, pageNr int) ([]Font, error) {
	ff := []Font{}
	for _, i := range FontObjNrs(ctx, pageNr) {
		fontObject := ctx.Optimize.FontObjects[i]
		f, err := ExtractFont(ctx, *fontObject, i)
		if err != nil {
			return nil, err
		}
		if f != nil {
			ff = append(ff, *f)
		}
	}
	return ff, nil
}

// ExtractPageFonts extracts all form fonts.
func ExtractFormFonts(ctx *model.Context) ([]Font, error) {
	ff := []Font{}
	for i, fontObject := range ctx.Optimize.FormFontObjects {
		f, err := ExtractFont(ctx, *fontObject, i)
		if err != nil {
			return nil, err
		}
		if f != nil {
			ff = append(ff, *f)
		}
	}
	return ff, nil
}

// ExtractPages extracts pageNrs into a new single page context.
func ExtractPages(ctx *model.Context, pageNrs []int, usePgCache bool) (*model.Context, error) {
	ctxDest, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		return nil, err
	}

	if err := AddPages(ctx, ctxDest, pageNrs, usePgCache); err != nil {
		return nil, err
	}

	return ctxDest, nil
}

// ExtractPageContent extracts the consolidated page content stream for pageNr.
func ExtractPageContent(ctx *model.Context, pageNr int) (io.Reader, error) {
	consolidateRes := false
	d, _, _, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return nil, err
	}
	bb, err := ctx.PageContent(d)
	if err != nil && err != model.ErrNoContent {
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

func extractMetadataFromDict(ctx *model.Context, d types.Dict, parentObjNr int) (*Metadata, error) {
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
	ir, _ := o.(types.IndirectRef)
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
func ExtractMetadata(ctx *model.Context) ([]Metadata, error) {
	mm := []Metadata{}
	for k, v := range ctx.Table {
		if v.Free || v.Compressed {
			continue
		}
		switch d := v.Object.(type) {
		case types.Dict:
			md, err := extractMetadataFromDict(ctx, d, k)
			if err != nil {
				return nil, err
			}
			if md == nil {
				continue
			}
			mm = append(mm, *md)

		case types.StreamDict:
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

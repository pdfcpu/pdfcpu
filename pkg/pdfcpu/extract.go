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
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func sortedObjectNumbers[V any](m map[int]V) []int {
	objNrs := make([]int, 0, len(m))
	for objNr := range m {
		objNrs = append(objNrs, objNr)
	}
	sort.Ints(objNrs)
	return objNrs
}

// ImageObjNrs returns all image dict objNrs for pageNr.
// Requires an optimized context.
func ImageObjNrs(ctx *model.Context, pageNr int) []int {
	// TODO Exclude SMask image objects.
	objNrs := []int{}

	if err := requireOptimizedContext(ctx); err != nil || pageNr < 1 {
		return objNrs
	}

	imgObjNrs := ctx.Optimize.PageImages
	if len(imgObjNrs) < pageNr {
		return objNrs
	}

	pageImgObjNrs := imgObjNrs[pageNr-1]
	if pageImgObjNrs == nil {
		return objNrs
	}

	for _, objNr := range sortedObjectNumbers(pageImgObjNrs) {
		if pageImgObjNrs[objNr] {
			objNrs = append(objNrs, objNr)
		}
	}
	return objNrs
}

// StreamLength returns sd's stream length.
func StreamLength(ctx *model.Context, sd *types.StreamDict) (int64, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return 0, err
	}
	if err := requireStreamDict(sd); err != nil {
		return 0, err
	}

	if val := sd.Int64Entry("Length"); val != nil {
		return *val, nil
	}

	indRef := sd.IndirectRefEntry("Length")
	if indRef == nil {
		return 0, nil
	}

	i, err := ctx.DereferenceInteger(*indRef)
	if err != nil {
		return 0, fmt.Errorf("stream length: %w", err)
	}
	if i == nil {
		return 0, fmt.Errorf("stream length: missing integer value")
	}
	return int64(*i), nil
}

// ColorSpaceString returns a string representation for sd's colorspace.
func ColorSpaceString(ctx *model.Context, sd *types.StreamDict) (string, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return "", err
	}
	if err := requireStreamDict(sd); err != nil {
		return "", err
	}

	o, found := sd.Find("ColorSpace")
	if !found {
		return "", nil
	}

	csObject := o
	o, err := ctx.Dereference(csObject)
	if err != nil {
		return "", colorSpaceDereferenceError(err, csObject, "colorspace")
	}

	switch cs := o.(type) {

	case types.Name:
		return string(cs), nil

	case types.Array:
		name, err := colorSpaceArrayName(cs)
		if err != nil {
			return "", err
		}
		return string(name), nil
	}

	return "", nil
}

func colorSpaceArrayName(cs types.Array) (types.Name, error) {
	if len(cs) == 0 {
		return "", fmt.Errorf("colorspace: empty array")
	}
	name, ok := cs[0].(types.Name)
	if !ok {
		return "", fmt.Errorf("colorspace: expected name, got %T", cs[0])
	}
	return name, nil
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

func colorSpaceArrayEntry(cs types.Array, index int) (types.Object, error) {
	if len(cs) <= index {
		return nil, fmt.Errorf("colorspace: missing array entry %d", index)
	}
	return cs[index], nil
}

func colorSpaceDereferenceError(err error, o types.Object, topic string) error {
	if indRef, ok := o.(types.IndirectRef); ok {
		return fmt.Errorf("%s obj#%d: dereference: %w", topic, indRef.ObjectNumber.Value(), err)
	}
	return fmt.Errorf("%s: dereference: %w", topic, err)
}

func dereferenceRequiredStreamDict(xRefTable *model.XRefTable, o types.Object, topic string) (*types.StreamDict, error) {
	if xRefTable == nil {
		return nil, ErrMissingXRefTable
	}
	sd, _, err := xRefTable.DereferenceStreamDict(o)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", topic, err)
	}
	if sd == nil {
		return nil, fmt.Errorf("%s: %w", topic, ErrMissingStreamDict)
	}
	return sd, nil
}

func indexedColorSpaceComponents(xRefTable *model.XRefTable, cs types.Array) (int, error) {
	if xRefTable == nil {
		return 0, ErrMissingXRefTable
	}

	o, err := colorSpaceArrayEntry(cs, 1)
	if err != nil {
		return 0, err
	}

	baseCS, err := xRefTable.Dereference(o)
	if err != nil {
		return 0, colorSpaceDereferenceError(err, o, "colorspace Indexed base")
	}

	switch cs := baseCS.(type) {
	case types.Name:
		return colorSpaceNameComponents(cs), nil

	case types.Array:
		return colorSpaceArrayComponents(xRefTable, cs)
	}

	return 0, nil
}

func iccBasedColorSpaceComponents(xRefTable *model.XRefTable, cs types.Array) (int, error) {
	if xRefTable == nil {
		return 0, ErrMissingXRefTable
	}

	o, err := colorSpaceArrayEntry(cs, 1)
	if err != nil {
		return 0, err
	}
	iccProfileStream, err := dereferenceRequiredStreamDict(xRefTable, o, "colorspace ICCBased profile")
	if err != nil {
		return 0, err
	}
	n := iccProfileStream.IntEntry("N")
	if n == nil {
		return 0, nil
	}
	return *n, nil
}

func deviceNColorSpaceComponents(cs types.Array) (int, error) {
	o, err := colorSpaceArrayEntry(cs, 1)
	if err != nil {
		return 0, err
	}
	colorants, ok := o.(types.Array)
	if !ok {
		return 0, fmt.Errorf("colorspace: DeviceN colorants: expected array, got %T", o)
	}
	return len(colorants), nil
}

func colorSpaceArrayComponents(xRefTable *model.XRefTable, cs types.Array) (int, error) {
	name, err := colorSpaceArrayName(cs)
	if err != nil {
		return 0, err
	}

	switch name {
	case model.CalGrayCS:
		return 1, nil
	case model.CalRGBCS, model.LabCS:
		return 3, nil
	case model.ICCBasedCS:
		return iccBasedColorSpaceComponents(xRefTable, cs)
	case model.SeparationCS:
		return 1, nil
	case model.DeviceNCS:
		return deviceNColorSpaceComponents(cs)
	case model.IndexedCS:
		return indexedColorSpaceComponents(xRefTable, cs)
	}

	return 0, nil
}

// ColorSpaceComponents returns the corresponding number of used color components for sd's colorspace.
func ColorSpaceComponents(xRefTable *model.XRefTable, sd *types.StreamDict) (int, error) {
	if xRefTable == nil {
		return 0, ErrMissingXRefTable
	}
	if err := requireStreamDict(sd); err != nil {
		return 0, err
	}

	o, found := sd.Find("ColorSpace")
	if !found {
		return 0, nil
	}

	csObject := o
	o, err := xRefTable.Dereference(csObject)
	if err != nil {
		return 0, colorSpaceDereferenceError(err, csObject, "colorspace")
	}

	switch cs := o.(type) {
	case types.Name:
		return colorSpaceNameComponents(cs), nil

	case types.Array:
		return colorSpaceArrayComponents(xRefTable, cs)
	}

	return 0, nil
}

func imageWidth(ctx *model.Context, sd *types.StreamDict, objNr int) (int, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return 0, err
	}
	if err := requireStreamDict(sd); err != nil {
		return 0, err
	}

	obj, ok := sd.Find("Width")
	if !ok {
		return 0, fmt.Errorf("missing image width obj#%d", objNr)
	}
	i, err := ctx.DereferenceInteger(obj)
	if err != nil {
		return 0, fmt.Errorf("image obj#%d width: %w", objNr, err)
	}
	if i == nil {
		return 0, fmt.Errorf("image obj#%d width: missing integer value", objNr)
	}
	return i.Value(), nil
}

func imageHeight(ctx *model.Context, sd *types.StreamDict, objNr int) (int, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return 0, err
	}
	if err := requireStreamDict(sd); err != nil {
		return 0, err
	}

	obj, ok := sd.Find("Height")
	if !ok {
		return 0, fmt.Errorf("missing image height obj#%d", objNr)
	}
	i, err := ctx.DereferenceInteger(obj)
	if err != nil {
		return 0, fmt.Errorf("image obj#%d height: %w", objNr, err)
	}
	if i == nil {
		return 0, fmt.Errorf("image obj#%d height: missing integer value", objNr)
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
		return nil, fmt.Errorf("image obj#%d: %w", objNr, err)
	}

	comp, err := ColorSpaceComponents(ctx.XRefTable, sd)
	if err != nil {
		return nil, fmt.Errorf("image obj#%d colorspace components: %w", objNr, err)
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

	var interpol bool
	if b := sd.BooleanEntry("Interpolate"); b != nil && *b {
		interpol = true
	}

	size, err := StreamLength(ctx, sd)
	if err != nil {
		return nil, fmt.Errorf("image obj#%d stream length: %w", objNr, err)
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
		HasImgMask:  sd.Dict.HasEntry("Mask"),
		HasSMask:    sd.Dict.HasEntry("SMask"),
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
			return fmt.Errorf("image obj#%d colorspace components: %w", objNr, err)
		}
		sd.CSComponents = comp
	}

	switch lastFilter {

	case filter.DCT, filter.JPX, filter.JBIG2, filter.Flate, filter.LZW, filter.CCITTFax, filter.RunLength:
		if err := sd.Decode(); errors.Is(err, filter.ErrUnsupportedFilter) {
			return fmt.Errorf("image obj#%d filter %s: %w (%w)", objNr, filters, ErrUnsupportedResource, err)
		} else if err != nil {
			return fmt.Errorf("image obj#%d decode: %w", objNr, err)
		}

	default:
		return fmt.Errorf("image obj#%d filter %s: %w", objNr, filters, ErrUnsupportedResource)
	}

	return nil
}

func img(
	ctx *model.Context,
	sd *types.StreamDict,
	thumb bool,
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
		if errors.Is(err, ErrUnsupportedResource) {
			return nil, err
		}
		return nil, fmt.Errorf("image obj#%d render: %w", objNr, err)
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
	if err := requireStreamDict(sd); err != nil {
		return nil, err
	}
	if err := requireContextWithXRefTable(ctx); err != nil {
		return nil, err
	}

	filters, lastFilter, decodeParms, imgMask := prepareExtractImage(sd)

	if stub {
		return imageStub(ctx, sd, resourceID, filters, lastFilter, decodeParms, thumb, imgMask, objNr)
	}

	return img(ctx, sd, thumb, resourceID, filters, lastFilter, objNr)
}

func validatePageNumber(ctx *model.Context, pageNr int) error {
	if pageNr < 1 || pageNr > ctx.PageCount {
		return fmt.Errorf("%w: %d outside 1..%d", ErrInvalidPageNumber, pageNr, ctx.PageCount)
	}
	return nil
}

func failOnUnsupportedResource(ctx *model.Context) bool {
	if ctx == nil {
		return false
	}
	if ctx.Configuration != nil {
		return ctx.UnsupportedResourcePolicy == model.UnsupportedResourceFail
	}
	return ctx.XRefTable != nil && ctx.XRefTable.Conf != nil &&
		ctx.XRefTable.Conf.UnsupportedResourcePolicy == model.UnsupportedResourceFail
}

func skipUnsupportedResource(ctx *model.Context, err error) bool {
	return errors.Is(err, ErrUnsupportedResource) && !failOnUnsupportedResource(ctx)
}

// ExtractPageImages extracts all images used by pageNr.
// Optionally return stubs only.
// Unsupported resources are handled according to ctx.UnsupportedResourcePolicy.
func ExtractPageImages(ctx *model.Context, pageNr int, stub bool) (map[int]model.Image, error) {
	if err := requireOptimizedContext(ctx); err != nil {
		return nil, err
	}
	if err := validatePageNumber(ctx, pageNr); err != nil {
		return nil, err
	}

	m := map[int]model.Image{}
	var skipErr error
	for _, objNr := range ImageObjNrs(ctx, pageNr) {
		imageObj := ctx.Optimize.ImageObjects[objNr]
		if imageObj == nil {
			return nil, fmt.Errorf("page %d image obj#%d: missing optimized image object", pageNr, objNr)
		}

		resourceName, ok := imageObj.ResourceNames[pageNr-1]
		if !ok {
			return nil, fmt.Errorf("page %d image obj#%d: missing resource name", pageNr, objNr)
		}

		img, err := ExtractImage(ctx, imageObj.ImageDict, false, resourceName, objNr, stub)
		if err != nil {
			if skipUnsupportedResource(ctx, err) {
				skipErr = errors.Join(skipErr, fmt.Errorf("page %d: %w", pageNr, err))
				continue
			}
			return nil, fmt.Errorf("page %d image obj#%d: %w", pageNr, objNr, err)
		}
		if img != nil {
			img.PageNr = pageNr
			m[objNr] = *img
		}
	}
	// Extract thumbnail for pageNr
	if indRef, ok := ctx.PageThumbs[pageNr]; ok {
		objNr := indRef.ObjectNumber.Value()
		sd, err := dereferenceRequiredStreamDict(ctx.XRefTable, indRef, fmt.Sprintf("thumbnail obj#%d", objNr))
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNr, err)
		}
		img, err := ExtractImage(ctx, sd, true, "", objNr, stub)
		if err != nil {
			if skipUnsupportedResource(ctx, err) {
				skipErr = errors.Join(skipErr, fmt.Errorf("page %d thumbnail obj#%d: %w", pageNr, objNr, err))
				return m, skipErr
			}
			return nil, fmt.Errorf("page %d thumbnail obj#%d: %w", pageNr, objNr, err)
		}
		if img != nil {
			img.PageNr = pageNr
			m[objNr] = *img
		}
	}
	return m, skipErr
}

// Font is a Reader representing an embedded font.
type Font struct {
	io.Reader
	Name  string
	Type  string
	ObjNr int
}

// FontObjNrs returns all font dict objNrs for pageNr.
// Requires an optimized context.
func FontObjNrs(ctx *model.Context, pageNr int) []int {
	objNrs := []int{}

	if err := requireOptimizedContext(ctx); err != nil || pageNr < 1 {
		return objNrs
	}

	fontObjNrs := ctx.Optimize.PageFonts
	if len(fontObjNrs) < pageNr {
		return objNrs
	}

	pageFontObjNrs := fontObjNrs[pageNr-1]
	if pageFontObjNrs == nil {
		return objNrs
	}

	for _, objNr := range sortedObjectNumbers(pageFontObjNrs) {
		if pageFontObjNrs[objNr] {
			objNrs = append(objNrs, objNr)
		}
	}
	return objNrs
}

// ExtractFont extracts a font from fontObject.
func ExtractFont(ctx *model.Context, fontObject model.FontObject, objNr int) (*Font, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return nil, err
	}

	d, err := font.FontDescriptor(ctx.XRefTable, fontObject.FontDict, objNr)
	if err != nil {
		return nil, fmt.Errorf("font obj#%d descriptor: %w", objNr, err)
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
		sd, err := dereferenceRequiredStreamDict(ctx.XRefTable, *ir, fmt.Sprintf("font obj#%d file", objNr))
		if err != nil {
			return nil, err
		}

		// Decode streamDict if used filter is supported only.
		if err = sd.Decode(); errors.Is(err, filter.ErrUnsupportedFilter) {
			return nil, fmt.Errorf("font %q obj#%d: %w (%w)", fontObject.FontName, objNr, ErrUnsupportedResource, err)
		} else if err != nil {
			return nil, fmt.Errorf("font obj#%d decode: %w", objNr, err)
		}

		f = &Font{Reader: bytes.NewReader(sd.Content), Name: fontObject.FontName, Type: "ttf", ObjNr: objNr}

	default:
		return nil, fmt.Errorf("font %q obj#%d type %s: %w", fontObject.FontName, objNr, fontType, ErrUnsupportedResource)
	}

	return f, nil
}

// ExtractPageFonts extracts all fonts used by pageNr.
// Unsupported resources are handled according to ctx.UnsupportedResourcePolicy.
func ExtractPageFonts(ctx *model.Context, pageNr int, objNrs, skipped types.IntSet) ([]Font, error) {
	if objNrs == nil {
		objNrs = types.IntSet{}
	}
	if skipped == nil {
		skipped = types.IntSet{}
	}
	if err := requireOptimizedContext(ctx); err != nil {
		return nil, err
	}
	if err := validatePageNumber(ctx, pageNr); err != nil {
		return nil, err
	}

	ff := []Font{}
	var skipErr error
	for _, i := range FontObjNrs(ctx, pageNr) {
		if objNrs[i] || skipped[i] {
			continue
		}
		fontObject := ctx.Optimize.FontObjects[i]
		if fontObject == nil {
			return nil, fmt.Errorf("page %d font obj#%d: missing optimized font object", pageNr, i)
		}
		f, err := ExtractFont(ctx, *fontObject, i)
		if err != nil {
			if skipUnsupportedResource(ctx, err) {
				skipped[i] = true
				skipErr = errors.Join(skipErr, fmt.Errorf("page %d: %w", pageNr, err))
				continue
			}
			return nil, fmt.Errorf("page %d: %w", pageNr, err)
		}
		if f != nil {
			ff = append(ff, *f)
			objNrs[i] = true
		} else {
			skipped[i] = true
		}
	}
	return ff, skipErr
}

// ExtractFormFonts extracts all form fonts.
// Unsupported resources are handled according to ctx.UnsupportedResourcePolicy.
func ExtractFormFonts(ctx *model.Context) ([]Font, error) {
	if err := requireOptimizedContext(ctx); err != nil {
		return nil, err
	}

	ff := []Font{}
	var skipErr error
	for _, i := range sortedObjectNumbers(ctx.Optimize.FormFontObjects) {
		fontObject := ctx.Optimize.FormFontObjects[i]
		if fontObject == nil {
			return nil, fmt.Errorf("form font obj#%d: missing optimized font object", i)
		}
		f, err := ExtractFont(ctx, *fontObject, i)
		if err != nil {
			if skipUnsupportedResource(ctx, err) {
				skipErr = errors.Join(skipErr, fmt.Errorf("form: %w", err))
				continue
			}
			return nil, fmt.Errorf("form: %w", err)
		}
		if f != nil {
			ff = append(ff, *f)
		}
	}
	return ff, skipErr
}

// ExtractPages extracts pageNrs into a new single page context.
func ExtractPages(ctx *model.Context, pageNrs []int, usePgCache bool) (*model.Context, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return nil, fmt.Errorf("extract pages: source context: %w", err)
	}

	if len(pageNrs) == 0 {
		return nil, ErrMissingPageNumbers
	}

	for _, pageNr := range pageNrs {
		if pageNr < 1 || pageNr > ctx.PageCount {
			return nil, fmt.Errorf("%w: %d outside 1..%d", ErrInvalidPageNumber, pageNr, ctx.PageCount)
		}
	}

	ctxDest, err := CreateContextWithXRefTable(ctx.Conf, types.PaperSize["A4"])
	if err != nil {
		return nil, fmt.Errorf("extract pages: create destination context: %w", err)
	}

	if err := AddPages(ctx, ctxDest, pageNrs, usePgCache); err != nil {
		return nil, fmt.Errorf("extract pages %v: %w", pageNrs, err)
	}

	return ctxDest, nil
}

// ExtractPageContent extracts the consolidated page content stream for pageNr.
func ExtractPageContent(ctx *model.Context, pageNr int) (io.Reader, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return nil, err
	}
	if err := validatePageNumber(ctx, pageNr); err != nil {
		return nil, err
	}

	consolidateRes := false
	d, _, _, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return nil, fmt.Errorf("page %d: page dict: %w", pageNr, err)
	}
	bb, err := ctx.PageContent(d, pageNr)
	if err != nil && err != model.ErrNoContent {
		return nil, fmt.Errorf("page %d: page content: %w", pageNr, err)
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
	sd, err := dereferenceRequiredStreamDict(ctx.XRefTable, o, "metadata")
	if err != nil {
		return nil, err
	}
	// Get metadata dict object number.
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return nil, fmt.Errorf("metadata: expected indirect ref, got %T", o)
	}
	objNr := ir.ObjectNumber.Value()
	// Get container dict type.
	dt := "unknown"
	if d.Type() != nil {
		dt = *d.Type()
	}
	// Decode streamDict for supported filters only.
	if err = sd.Decode(); errors.Is(err, filter.ErrUnsupportedFilter) {
		return nil, fmt.Errorf("metadata obj#%d: %w (%w)", objNr, ErrUnsupportedResource, err)
	} else if err != nil {
		return nil, fmt.Errorf("metadata obj#%d decode: %w", objNr, err)
	}
	return &Metadata{bytes.NewReader(sd.Content), objNr, parentObjNr, dt}, nil
}

// ExtractMetadata returns all metadata of ctx.
// Unsupported resources are handled according to ctx.UnsupportedResourcePolicy.
func ExtractMetadata(ctx *model.Context) ([]Metadata, error) {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return nil, err
	}

	mm := []Metadata{}
	var skipErr error
	for _, k := range sortedObjectNumbers(ctx.Table) {
		v := ctx.Table[k]
		if v == nil || v.Free || v.Compressed {
			continue
		}
		switch d := v.Object.(type) {
		case types.Dict:
			md, err := extractMetadataFromDict(ctx, d, k)
			if err != nil {
				if skipUnsupportedResource(ctx, err) {
					skipErr = errors.Join(skipErr, fmt.Errorf("metadata parent obj#%d: %w", k, err))
					continue
				}
				return nil, fmt.Errorf("metadata parent obj#%d: %w", k, err)
			}
			if md == nil {
				continue
			}
			mm = append(mm, *md)

		case types.StreamDict:
			md, err := extractMetadataFromDict(ctx, d.Dict, k)
			if err != nil {
				if skipUnsupportedResource(ctx, err) {
					skipErr = errors.Join(skipErr, fmt.Errorf("metadata parent obj#%d: %w", k, err))
					continue
				}
				return nil, fmt.Errorf("metadata parent obj#%d: %w", k, err)
			}
			if md == nil {
				continue
			}
			mm = append(mm, *md)
		}
	}
	return mm, skipErr
}

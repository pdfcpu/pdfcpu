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
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func optimizeContentStreamUsage(ctx *model.Context, sd *types.StreamDict, objNr int) (*types.IndirectRef, error) {
	f := ctx.Optimize.ContentStreamCache
	if len(f) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	if f[objNr] != nil {
		return nil, nil
	}

	cachedObjNrs := []int{}
	for objNr, sd1 := range f {
		if *sd1.StreamLength == *sd.StreamLength {
			cachedObjNrs = append(cachedObjNrs, objNr)
		}
	}
	if len(cachedObjNrs) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	for _, objNr := range cachedObjNrs {
		sd1 := f[objNr]
		if bytes.Equal(sd.Raw, sd1.Raw) {
			ir := types.NewIndirectRef(objNr, 0)
			ctx.IncrementRefCount(ir)
			return ir, nil
		}
	}

	f[objNr] = sd
	return nil, nil
}

func removeEmptyContentStreams(ctx *model.Context, pageDict types.Dict, obj types.Object, pageObjNumber int) error {
	var contentArr types.Array

	if ir, ok := obj.(types.IndirectRef); ok {

		objNr := ir.ObjectNumber.Value()
		entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
		if !found {
			return fmt.Errorf("removeEmptyContentStreams: obj#:%d illegal indRef for Contents", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.StreamDict)
		if ok {
			if err := contentStreamDict.Decode(); err != nil {
				return fmt.Errorf("content stream obj#%d: decode: %w", objNr, err)
			}
			if len(contentStreamDict.Content) == 0 {
				pageDict.Delete("Contents")
			}
			return nil
		}

		contentArr, ok = entry.Object.(types.Array)
		if !ok {
			return fmt.Errorf("removeEmptyContentStreams: obj#:%d page content entry neither stream dict nor array", pageObjNumber)
		}

	} else if contentArr, ok = obj.(types.Array); !ok {
		return fmt.Errorf("removeEmptyContentStreams: obj#:%d corrupt page content array", pageObjNumber)
	}

	var newContentArr types.Array

	for _, c := range contentArr {

		ir, ok := c.(types.IndirectRef)
		if !ok {
			return fmt.Errorf("removeEmptyContentStreams: obj#:%d corrupt page content array entry", pageObjNumber)
		}

		objNr := ir.ObjectNumber.Value()
		entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
		if !found {
			return fmt.Errorf("removeEmptyContentStreams: obj#:%d illegal indRef for Contents", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.StreamDict)
		if !ok {
			return fmt.Errorf("identifyPageContent: obj#:%d page content entry is no stream dict", pageObjNumber)
		}

		if err := contentStreamDict.Decode(); err != nil {
			return fmt.Errorf("content stream obj#%d: decode: %w", objNr, err)
		}
		if len(contentStreamDict.Content) > 0 {
			newContentArr = append(newContentArr, c)
		}
	}

	pageDict["Contents"] = newContentArr

	return nil
}

func optimizePageContent(ctx *model.Context, pageDict types.Dict, pageObjNumber int) error {
	o, found := pageDict.Find("Contents")
	if !found {
		return nil
	}

	if err := removeEmptyContentStreams(ctx, pageDict, o, pageObjNumber); err != nil {
		return fmt.Errorf("remove empty content streams: %w", err)
	}

	o, found = pageDict.Find("Contents")
	if !found {
		return nil
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("identifyPageContent begin")
	}

	var contentArr types.Array

	if ir, ok := o.(types.IndirectRef); ok {

		objNr := ir.ObjectNumber.Value()
		entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
		if !found {
			return fmt.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.StreamDict)
		if ok {
			ir, err := optimizeContentStreamUsage(ctx, &contentStreamDict, objNr)
			if err != nil {
				return fmt.Errorf("content stream obj#%d: optimize usage: %w", objNr, err)
			}
			if ir != nil {
				pageDict["Contents"] = *ir
			}
			contentStreamDict.IsPageContent = true
			entry.Object = contentStreamDict
			if log.OptimizeEnabled() {
				log.Optimize.Printf("identifyPageContent end: ok obj#%d\n", objNr)
			}
			return nil
		}

		contentArr, ok = entry.Object.(types.Array)
		if !ok {
			return fmt.Errorf("identifyPageContent: obj#:%d page content entry neither stream dict nor array", pageObjNumber)
		}

	} else if contentArr, ok = o.(types.Array); !ok {
		return fmt.Errorf("identifyPageContent: obj#:%d corrupt page content array", pageObjNumber)
	}

	// TODO Activate content array optimization as soon as we have a proper test file.

	_ = contentArr

	// for i, c := range contentArr {

	// 	ir, ok := c.(IndirectRef)
	// 	if !ok {
	// 		return fmt.Errorf("identifyPageContent: obj#:%d corrupt page content array entry", pageObjNumber)
	// 	}

	// 	objNr := ir.ObjectNumber.Value()
	// 	entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
	// 	if !found {
	// 		return fmt.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents", pageObjNumber)
	// 	}

	// 	contentStreamDict, ok := entry.Object.(StreamDict)
	// 	if !ok {
	// 		return fmt.Errorf("identifyPageContent: obj#:%d page content entry is no stream dict", pageObjNumber)
	// 	}

	// 	ir1, err := optimizeContentStreamUsage(ctx, &contentStreamDict, objNr)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if ir1 != nil {
	// 		contentArr[i] = *ir1
	// 	}

	// 	contentStreamDict.IsPageContent = true
	// 	entry.Object = contentStreamDict
	// 	log.Optimize.Printf("identifyPageContent: ok obj#%d\n", ir.GenerationNumber.Value())
	// }

	if log.OptimizeEnabled() {
		log.Optimize.Println("identifyPageContent end")
	}

	return nil
}

// resourcesDictForPageDict returns the resource dict for a page dict if there is any.
func resourcesDictForPageDict(xRefTable *model.XRefTable, pageDict types.Dict, pageObjNumber int) (types.Dict, error) {
	o, found := pageDict.Find("Resources")
	if !found {
		if log.OptimizeEnabled() {
			log.Optimize.Printf("resourcesDictForPageDict end: No resources dict for page object %d, may be inherited\n", pageObjNumber)
		}
		return nil, nil
	}

	return xRefTable.DereferenceDict(o)
}

// handleDuplicateFontObject returns nil or the object number of the registered font if it matches this font.
func handleDuplicateFontObject(ctx *model.Context, fontDict types.Dict, fName, rName string, objNr, pageNr int) (*int, error) {
	// Get a slice of all font object numbers for font name.
	fontObjNrs, found := ctx.Optimize.Fonts[fName]
	if !found {
		// There is no registered font with fName.
		return nil, nil
	}

	// Get the set of font object numbers for pageNr.
	pageFonts := ctx.Optimize.PageFonts[pageNr]

	// Iterate over all registered font object numbers for font name.
	// Check if this font dict matches the font dict of each font object number.
	for _, fontObjNr := range fontObjNrs {

		if fontObjNr == objNr {
			continue
		}

		// Get the font object from the lookup table.
		fontObject, ok := ctx.Optimize.FontObjects[fontObjNr]
		if !ok {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateFontObject: comparing with fontDict Obj %d\n", fontObjNr)
		}

		// Check if the input fontDict matches the fontDict of this fontObject.
		ok, err := model.EqualObjects(fontObject.FontDict, fontDict, ctx.XRefTable, nil)
		if err != nil {
			return nil, fmt.Errorf("compare font obj#%d with obj#%d: %w", objNr, fontObjNr, err)
		}

		if !ok {
			// No match!
			continue
		}

		// We have detected a redundant font dict!
		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateFontObject: redundant fontObj#:%d basefont %s already registered with obj#:%d !\n", objNr, fName, fontObjNr)
		}

		// Register new page font with pageNr.
		// The font for font object number is used instead of objNr.
		pageFonts[fontObjNr] = true

		// Add the resource name of this duplicate font to the list of registered resource names.
		fontObject.AddResourceName(rName)

		// Register fontDict as duplicate.
		ctx.Optimize.DuplicateFonts[objNr] = fontDict

		// Return the fontObjectNumber that will be used instead of objNr.
		return &fontObjNr, nil
	}

	return nil, nil
}

func pageImages(ctx *model.Context, pageNr int) types.IntSet {
	pageImages := ctx.Optimize.PageImages[pageNr]
	if pageImages == nil {
		pageImages = types.IntSet{}
		ctx.Optimize.PageImages[pageNr] = pageImages
	}

	return pageImages
}

func pageFonts(ctx *model.Context, pageNr int) types.IntSet {
	pageFonts := ctx.Optimize.PageFonts[pageNr]
	if pageFonts == nil {
		pageFonts = types.IntSet{}
		ctx.Optimize.PageFonts[pageNr] = pageFonts
	}

	return pageFonts
}

func registerFontDictObjNr(ctx *model.Context, fName string, objNr int) {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeFontResourcesDict: adding new font %s obj#%d\n", fName, objNr)
	}

	fontObjNrs, found := ctx.Optimize.Fonts[fName]
	if found {
		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: appending %d to %s\n", objNr, fName)
		}
		ctx.Optimize.Fonts[fName] = append(fontObjNrs, objNr)
	} else {
		ctx.Optimize.Fonts[fName] = []int{objNr}
	}
}

func checkForEmbeddedFont(ctx *model.Context) bool {
	return log.StatsEnabled() || ctx.Cmd == model.LISTINFO || ctx.Cmd == model.EXTRACTFONTS
}

func qualifiedRName(rNamePrefix, rName string) string {
	s := rName
	if rNamePrefix != "" {
		s = rNamePrefix + "." + rName
	}
	return s
}

// Get rid of redundant fonts for given fontResources dictionary.
func optimizeFontResourcesDict(ctx *model.Context, rDict types.Dict, pageNr int, rNamePrefix string) error {
	pageFonts := pageFonts(ctx, pageNr)

	recordedCorrupt := false

	// Iterate over font resource dict.
	for rName, v := range rDict {

		if v == nil {
			if !recordedCorrupt {
				// fontId with missing fontDict indRef.
				ctx.Optimize.CorruptFontResDicts = append(ctx.Optimize.CorruptFontResDicts, rDict)
				recordedCorrupt = true
			}
			continue
		}

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		objNr := int(indRef.ObjectNumber)

		qualifiedRName := qualifiedRName(rNamePrefix, rName)

		if _, found := ctx.Optimize.FontObjects[objNr]; found {
			// This font has already been registered.
			pageFonts[objNr] = true
			continue
		}

		// We are dealing with a new font.
		fontDict, err := ctx.DereferenceFontDict(indRef)
		if err != nil {
			if ctx.XRefTable.ValidationMode == model.ValidationStrict {
				return fmt.Errorf("font resource %s obj#%d: dereference font dict: %w", qualifiedRName, objNr, err)
			}

			fontDict = nil
		}
		if fontDict == nil {
			continue
		}

		// Get the unique font name.
		prefix, fName, err := pdffont.Name(ctx.XRefTable, fontDict, objNr)
		if err != nil {
			return fmt.Errorf("font resource %s obj#%d: parse font name: %w", qualifiedRName, objNr, err)
		}

		// Check if fontDict is a duplicate and if so return the object number of the original.
		originalObjNr, err := handleDuplicateFontObject(ctx, fontDict, fName, qualifiedRName, objNr, pageNr)
		if err != nil {
			return fmt.Errorf("font resource %s obj#%d: duplicate check: %w", qualifiedRName, objNr, err)
		}

		if originalObjNr != nil {
			// We have identified a redundant fontDict!
			// Update font resource dict so that rName points to the original.
			ir := types.NewIndirectRef(*originalObjNr, 0)
			rDict[rName] = *ir
			ctx.IncrementRefCount(ir)
			if log.OptimizeEnabled() {
				log.Optimize.Printf("optimizeFontResourcesDict: redundant fontDict prefix=%s name=%s (objNr#%d -> objNr#%d)\n", prefix, fName, objNr, originalObjNr)
			}
			continue
		}

		registerFontDictObjNr(ctx, fName, objNr)

		fontObj := model.FontObject{
			ResourceNames: []string{qualifiedRName},
			Prefix:        prefix,
			FontName:      fName,
			FontDict:      fontDict,
		}

		if checkForEmbeddedFont(ctx) {
			fontObj.Embedded, err = pdffont.Embedded(ctx.XRefTable, fontDict, objNr)
			if err != nil {
				return fmt.Errorf("font resource %s obj#%d: detect embedded font: %w", qualifiedRName, objNr, err)
			}
		}

		ctx.Optimize.FontObjects[objNr] = &fontObj

		pageFonts[objNr] = true
	}

	return nil
}

func imageObjectHashes(ctx *model.Context) map[[sha256.Size]byte][]int {
	hashes := ctx.Optimize.ImageObjectHashes
	if hashes != nil {
		return hashes
	}

	hashes = map[[sha256.Size]byte][]int{}
	for objNr, imageObject := range ctx.Optimize.ImageObjects {
		h := sha256.Sum256(imageObject.ImageDict.Raw)
		hashes[h] = append(hashes[h], objNr)
	}
	ctx.Optimize.ImageObjectHashes = hashes
	return hashes
}

// handleDuplicateImageObject returns nil or the object number of the registered image if it matches this image.
func handleDuplicateImageObject(ctx *model.Context, imageDict *types.StreamDict, resourceName string, objNr, pageNr int) (*int, bool, error) {
	// Get the set of image object numbers for pageNr.
	pageImages := ctx.Optimize.PageImages[pageNr]

	if duplImgObj, ok := ctx.Optimize.DuplicateImages[objNr]; ok {

		newObjNr := duplImgObj.NewObjNr
		// We have detected a redundant image dict.
		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateImageObject: redundant imageObj#:%d already registered with obj#:%d !\n", objNr, newObjNr)
		}

		// Register new page image for pageNr.
		// The image for image object number is used instead of objNr.
		pageImages[newObjNr] = true

		// Add the resource name of this duplicate image to the list of registered resource names.
		ctx.Optimize.ImageObjects[newObjNr].AddResourceName(pageNr, resourceName)

		// Return the imageObjectNumber that will be used instead of objNr.
		return &newObjNr, false, nil
	}

	if imageObject, ok := ctx.Optimize.ImageObjects[objNr]; ok {
		imageObject.AddResourceName(pageNr, resourceName)
		return nil, true, nil
	}

	// Process image dict, check if this is a duplicate.
	h := sha256.Sum256(imageDict.Raw)
	imageHashes := imageObjectHashes(ctx)
	for _, imageObjNr := range imageHashes[h] {
		imageObject := ctx.Optimize.ImageObjects[imageObjNr]

		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateImageObject: comparing with imagedict Obj %d\n", imageObjNr)
		}

		// Check if the input imageDict matches the imageDict of this imageObject.
		ok, err := model.EqualObjects(*imageObject.ImageDict, *imageDict, ctx.XRefTable, nil)
		if err != nil {
			return nil, false, fmt.Errorf("compare image obj#%d with obj#%d: %w", objNr, imageObjNr, err)
		}

		if !ok {
			// No match!
			continue
		}

		// We have detected a redundant image dict.
		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateImageObject: redundant imageObj#:%d already registered with obj#:%d !\n", objNr, imageObjNr)
		}

		// Register new page image for pageNr.
		// The image for image object number is used instead of objNr.
		pageImages[imageObjNr] = true

		// Add the resource name of this duplicate image to the list of registered resource names.
		imageObject.AddResourceName(pageNr, resourceName)

		// Register imageDict as duplicate.
		ctx.Optimize.DuplicateImages[objNr] = &model.DuplicateImageObject{ImageDict: imageDict, NewObjNr: imageObjNr}

		// Return the imageObjectNumber that will be used instead of objNr.
		return &imageObjNr, false, nil
	}

	imageHashes[h] = append(imageHashes[h], objNr)
	return nil, false, nil
}

func optimizeXObjectImage(ctx *model.Context, osd *types.StreamDict, rNamePrefix, rName string, rDict types.Dict, objNr, pageNr, pageObjNumber int, pageImages types.IntSet) error {
	qualifiedRName := rName
	if rNamePrefix != "" {
		qualifiedRName = rNamePrefix + "." + rName
	}

	// Check if image is a duplicate and if so return the object number of the original.
	originalObjNr, alreadyDupl, err := handleDuplicateImageObject(ctx, osd, qualifiedRName, objNr, pageNr)
	if err != nil {
		return fmt.Errorf("image resource %s obj#%d: duplicate check: %w", qualifiedRName, objNr, err)
	}

	if originalObjNr != nil {
		// We have identified a redundant image!
		// Update xobject resource dict so that rName points to the original.
		ir := types.NewIndirectRef(*originalObjNr, 0)
		ctx.IncrementRefCount(ir)
		rDict[rName] = *ir
		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeXObjectImage: redundant xobject name=%s (objNr#%d -> objNr#%d)\n", qualifiedRName, objNr, originalObjNr)
		}
		return nil
	}

	if !alreadyDupl {
		// Register new image dict.
		ctx.Optimize.ImageObjects[objNr] =
			&model.ImageObject{
				ResourceNames: map[int]string{pageNr: qualifiedRName},
				ImageDict:     osd,
			}
	}

	pageImages[objNr] = true
	return nil
}

func optimizeXObjectForm(ctx *model.Context, sd *types.StreamDict, objNr int) (*types.IndirectRef, error) {
	f := ctx.Optimize.FormStreamCache
	if len(f) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	if f[objNr] != nil {
		return nil, nil
	}

	cachedObjNrs := []int{}
	for objNr, sd1 := range f {
		if *sd1.StreamLength == *sd.StreamLength {
			cachedObjNrs = append(cachedObjNrs, objNr)
		}
	}
	if len(cachedObjNrs) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	for _, objNr1 := range cachedObjNrs {
		sd1 := f[objNr1]
		ok, err := model.EqualObjects(*sd, *sd1, ctx.XRefTable, nil)
		if err != nil {
			return nil, fmt.Errorf("compare form XObject obj#%d with obj#%d: %w", objNr, objNr1, err)
		}
		if ok {
			ir := types.NewIndirectRef(objNr1, 0)
			ctx.IncrementRefCount(ir)
			return ir, nil
		}
	}

	f[objNr] = sd
	return nil, nil
}

func optimizeFormResources(ctx *model.Context, o types.Object, pageNr, pageObjNumber int, rName string, visitedRes []types.Object) error {
	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("form resource %s: dereference resources: %w", rName, err)
	}
	if d != nil {
		// Optimize image and font resources.
		if err = optimizeResources(ctx, d, pageNr, pageObjNumber, rName, visitedRes); err != nil {
			return fmt.Errorf("form resource %s: optimize resources: %w", rName, err)
		}
	}
	return nil
}

func visited(o types.Object, visited []types.Object) bool {
	return slices.Contains(visited, o)
}

func formResourcesVisited(ctx *model.Context, pageNr, objNr int) bool {
	cache := ctx.Optimize.FormResourceCache
	if cache == nil {
		cache = map[int]types.IntSet{}
		ctx.Optimize.FormResourceCache = cache
	}

	forms := cache[pageNr]
	if forms == nil {
		forms = types.IntSet{}
		cache[pageNr] = forms
	}
	if forms[objNr] {
		return true
	}
	forms[objNr] = true
	return false
}

func optimizeForm(ctx *model.Context, osd *types.StreamDict, rNamePrefix, rName string, rDict types.Dict, objNr, pageNr, pageObjNumber int, vis []types.Object) error {
	ir, err := optimizeXObjectForm(ctx, osd, objNr)
	if err != nil {
		return fmt.Errorf("form XObject %s obj#%d: optimize usage: %w", qualifiedRName(rNamePrefix, rName), objNr, err)
	}

	if ir != nil {
		rDict[rName] = *ir
		return nil
	}

	if formResourcesVisited(ctx, pageNr, objNr) {
		return nil
	}

	o, found := osd.Find("Resources")
	if !found {
		return nil
	}

	indRef, ok := o.(types.IndirectRef)
	if ok {
		if visited(indRef, vis) {
			return nil
		}
		vis = append(vis, indRef)
	}

	qualifiedRName := rName
	if rNamePrefix != "" {
		qualifiedRName = rNamePrefix + "." + rName
	}

	return optimizeFormResources(ctx, o, pageNr, pageObjNumber, qualifiedRName, vis)
}

func optimizeExtGStateResources(ctx *model.Context, rDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string, vis []types.Object) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeExtGStateResources page#%dbegin: %s\n", pageObjNumber, rDict)
	}

	pageImages := pageImages(ctx, pageNr)

	s, found := rDict.Find("SMask")
	if found {
		dict, ok := s.(types.Dict)
		if ok {
			if err := optimizeSMaskResources(dict, vis, rNamePrefix, ctx, rDict, pageNr, pageImages, pageObjNumber); err != nil {
				return fmt.Errorf("SMask: %w", err)
			}
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeExtGStateResources end")
	}

	return nil
}

func optimizeSMaskResources(dict types.Dict, vis []types.Object, rNamePrefix string, ctx *model.Context, rDict types.Dict, pageNr int, pageImages types.IntSet, pageObjNumber int) error {
	indRef := dict.IndirectRefEntry("G")
	if indRef == nil {
		return nil
	}

	if visited(*indRef, vis) {
		return nil
	}

	vis = append(vis, indRef)

	objNr := int(indRef.ObjectNumber)

	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeSMaskResources: processing \"G\", obj#=%d\n", objNr)
	}

	sd, err := ctx.DereferenceXObjectDict(*indRef)
	if err != nil {
		return fmt.Errorf("SMask G obj#%d: dereference XObject: %w", objNr, err)
	}
	if sd == nil {
		return nil
	}

	subtype := sd.Subtype()
	if subtype == nil || len(*subtype) == 0 {
		model.ShowSkipped(fmt.Sprintf("unclassifiable XObject SMask G obj#%d: missing Subtype", objNr))
		return nil
	}

	if *subtype == "Image" {
		if err := optimizeXObjectImage(ctx, sd, rNamePrefix, "G", rDict, objNr, pageNr, pageObjNumber, pageImages); err != nil {
			return fmt.Errorf("SMask G image: %w", err)
		}
	}

	if *subtype == "Form" {
		if err := optimizeForm(ctx, sd, rNamePrefix, "G", rDict, objNr, pageNr, pageObjNumber, vis); err != nil {
			return fmt.Errorf("SMask G form: %w", err)
		}
	}

	return nil
}

func optimizeExtGStateResourcesDict(ctx *model.Context, rDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string, vis []types.Object) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeExtGStateResourcesDict page#%dbegin: %s\n", pageObjNumber, rDict)
	}

	for rName, v := range rDict {

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		if visited(indRef, vis) {
			continue
		}

		vis = append(vis, indRef)

		objNr := int(indRef.ObjectNumber)

		qualifiedRName := qualifiedRName(rNamePrefix, rName)

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeExtGStateResourcesDict: processing XObject: %s, obj#=%d\n", qualifiedRName, objNr)
		}

		rDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return fmt.Errorf("ExtGState resource %s obj#%d: dereference dict: %w", qualifiedRName, objNr, err)
		}
		if rDict == nil {
			continue
		}

		if err := optimizeExtGStateResources(ctx, rDict, pageNr, pageObjNumber, qualifiedRName, vis); err != nil {
			return fmt.Errorf("ExtGState resource %s obj#%d: optimize resources: %w", qualifiedRName, objNr, err)
		}

	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeXObjectResourcesDict end")
	}

	return nil
}

func optimizeXObjectResource(ctx *model.Context, sd *types.StreamDict, rDict types.Dict, rNamePrefix, rName string,
	qualifiedRName string, objNr, pageNr, pageObjNumber int, pageImages types.IntSet, vis []types.Object) error {
	subtype := sd.Subtype()
	if subtype == nil || len(*subtype) == 0 {
		model.ShowSkipped(fmt.Sprintf("unclassifiable XObject resource %s obj#%d: missing Subtype", qualifiedRName, objNr))
		return nil
	}

	if *subtype == "Image" {
		if err := optimizeXObjectImage(ctx, sd, rNamePrefix, rName, rDict, objNr, pageNr, pageObjNumber, pageImages); err != nil {
			return fmt.Errorf("XObject resource %s obj#%d: image: %w", qualifiedRName, objNr, err)
		}
	}

	if *subtype == "Form" {
		// Get rid of PieceInfo dict from form XObjects.
		if err := ctx.DeleteDictEntry(sd.Dict, "PieceInfo"); err != nil {
			return fmt.Errorf("XObject resource %s obj#%d: delete PieceInfo: %w", qualifiedRName, objNr, err)
		}
		if err := optimizeForm(ctx, sd, rNamePrefix, rName, rDict, objNr, pageNr, pageObjNumber, vis); err != nil {
			return fmt.Errorf("XObject resource %s obj#%d: form: %w", qualifiedRName, objNr, err)
		}
	}

	return nil
}

func optimizeXObjectResourcesDict(ctx *model.Context, rDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string, vis []types.Object) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeXObjectResourcesDict page#%dbegin: %s\n", pageObjNumber, rDict)
	}

	pageImages := pageImages(ctx, pageNr)

	for rName, v := range rDict {

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		if visited(indRef, vis) {
			continue
		}

		vis = append(vis, indRef)

		objNr := int(indRef.ObjectNumber)

		qualifiedRName := qualifiedRName(rNamePrefix, rName)

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeXObjectResourcesDict: processing XObject: %s, obj#=%d\n", qualifiedRName, objNr)
		}

		sd, err := ctx.DereferenceXObjectDict(indRef)
		if err != nil {
			return fmt.Errorf("XObject resource %s obj#%d: dereference XObject: %w", qualifiedRName, objNr, err)
		}
		if sd == nil {
			continue
		}

		if err := optimizeXObjectResource(
			ctx, sd, rDict, rNamePrefix, rName, qualifiedRName, objNr, pageNr, pageObjNumber, pageImages, vis); err != nil {
			return err
		}

	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeXObjectResourcesDict end")
	}

	return nil
}

func processFontResources(ctx *model.Context, obj types.Object, pageNr, pageObjNumber int, rNamePrefix string) error {
	d, err := ctx.DereferenceDict(obj)
	if err != nil {
		return fmt.Errorf("font resources: dereference dict: %w", err)
	}

	if d == nil {
		return fmt.Errorf("font resource dict is null for page %d pageObj %d", pageNr, pageObjNumber)
	}

	if err := optimizeFontResourcesDict(ctx, d, pageNr, rNamePrefix); err != nil {
		return fmt.Errorf("font resources: optimize dict: %w", err)
	}
	return nil
}

func processXObjectResources(ctx *model.Context, obj types.Object, pageNr, pageObjNumber int, rNamePrefix string, visitedRes []types.Object) error {
	d, err := ctx.DereferenceDict(obj)
	if err != nil {
		return fmt.Errorf("XObject resources: dereference dict: %w", err)
	}

	if d == nil {
		return fmt.Errorf("xObject resource dict is null for page %d pageObj %d", pageNr, pageObjNumber)
	}

	if err := optimizeXObjectResourcesDict(ctx, d, pageNr, pageObjNumber, rNamePrefix, visitedRes); err != nil {
		return fmt.Errorf("XObject resources: optimize dict: %w", err)
	}
	return nil
}

func processExtGStateResources(ctx *model.Context, obj types.Object, pageNr, pageObjNumber int, rNamePrefix string, visitedRes []types.Object) error {
	d, err := ctx.DereferenceDict(obj)
	if err != nil {
		return fmt.Errorf("ExtGState resources: dereference dict: %w", err)
	}

	if d == nil {
		return fmt.Errorf("processExtGStateResources: extGState resource dict is null for page %d pageObj %d", pageNr, pageObjNumber)
	}

	if err := optimizeExtGStateResourcesDict(ctx, d, pageNr, pageObjNumber, rNamePrefix, visitedRes); err != nil {
		return fmt.Errorf("ExtGState resources: optimize dict: %w", err)
	}
	return nil
}

// Optimize given resource dictionary by removing redundant fonts and images.
func optimizeResources(ctx *model.Context, resourcesDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string, visitedRes []types.Object) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeResources begin: pageNr=%d pageObjNumber=%d\n", pageNr, pageObjNumber)
	}

	if resourcesDict == nil {
		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeResources end: No resources dict available")
		}
		return nil
	}

	obj, found := resourcesDict.Find("Font")
	if found {
		// Process Font resource dict, get rid of redundant fonts.
		if err := processFontResources(ctx, obj, pageNr, pageObjNumber, rNamePrefix); err != nil {
			return fmt.Errorf("Font: %w", err)
		}
	}

	obj, found = resourcesDict.Find("XObject")
	if found {
		// Process XObject resource dict, get rid of redundant images.
		if err := processXObjectResources(ctx, obj, pageNr, pageObjNumber, rNamePrefix, visitedRes); err != nil {
			return fmt.Errorf("XObject: %w", err)
		}
	}

	obj, found = resourcesDict.Find("ExtGState")
	if found {
		// An ExtGState resource dict may contain binary content in the following entries: "SMask", "HT".
		if err := processExtGStateResources(ctx, obj, pageNr, pageObjNumber, rNamePrefix, visitedRes); err != nil {
			return fmt.Errorf("ExtGState: %w", err)
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeResources end")
	}

	return nil
}

// Process the resources dictionary for given page number and optimize by removing redundant resources.
func parseResourcesDict(ctx *model.Context, pageDict types.Dict, pageNr, pageObjNumber int) error {
	if ctx.Optimize.Cache[pageObjNumber] {
		return nil
	}
	ctx.Optimize.Cache[pageObjNumber] = true

	// The logical pageNr is pageNr+1.
	if log.OptimizeEnabled() {
		log.Optimize.Printf("parseResourcesDict begin page: %d, object:%d\n", pageNr+1, pageObjNumber)
	}

	// Get resources dict for this page.
	d, err := resourcesDictForPageDict(ctx.XRefTable, pageDict, pageObjNumber)
	if err != nil {
		return fmt.Errorf("page %d obj#%d: resources dict: %w", pageNr+1, pageObjNumber, err)
	}

	// dict may be nil for inherited resource dicts.
	if d != nil {

		// Optimize image and font resources.
		if err = optimizeResources(ctx, d, pageNr, pageObjNumber, "", []types.Object{}); err != nil {
			return fmt.Errorf("page %d obj#%d: optimize resources: %w", pageNr+1, pageObjNumber, err)
		}

	}

	if log.OptimizeEnabled() {
		log.Optimize.Printf("parseResourcesDict end page: %d, object:%d\n", pageNr+1, pageObjNumber)
	}

	return nil
}

func parsePageTreeKid(ctx *model.Context, v types.Object, kidNr, pageNr int) (int, error) {
	if v == nil {
		return pageNr, nil
	}

	ir, ok := v.(types.IndirectRef)
	if !ok {
		return 0, fmt.Errorf("kid %d: expected indirect reference, got %T", kidNr, v)
	}

	d, err := ctx.DereferencePageNodeDict(ir)
	if err != nil {
		return 0, fmt.Errorf("kid %d obj#%d: dereference page node: %w", kidNr, ir.ObjectNumber.Value(), err)
	}

	if *d.Type() == "Pages" {
		pageNr, err = parsePagesDict(ctx, d, pageNr)
		if err != nil {
			return 0, fmt.Errorf("kid %d pages obj#%d: %w", kidNr, ir.ObjectNumber.Value(), err)
		}
		return pageNr, nil
	}

	if ctx.OptimizeDuplicateContentStreams {
		if err = optimizePageContent(ctx, d, int(ir.ObjectNumber)); err != nil {
			return 0, fmt.Errorf("page %d obj#%d: optimize content: %w", pageNr+1, ir.ObjectNumber.Value(), err)
		}
	}

	if err := ctx.DeleteDictEntry(d, "PieceInfo"); err != nil {
		return 0, fmt.Errorf("page %d obj#%d: delete PieceInfo: %w", pageNr+1, ir.ObjectNumber.Value(), err)
	}

	if err = parseResourcesDict(ctx, d, pageNr, int(ir.ObjectNumber)); err != nil {
		return 0, err
	}

	return pageNr + 1, nil
}

// Iterate over all pages and optimize content & resources.
func parsePagesDict(ctx *model.Context, pagesDict types.Dict, pageNr int) (int, error) {
	// TODO Integrate resource consolidation based on content stream requirements.

	_, found := pagesDict.Find("Count")
	if !found {
		return pageNr, errors.New("parsePagesDict: missing Count")
	}

	ctx.Optimize.Cache = map[int]bool{}

	// Iterate over page tree.
	o, found := pagesDict.Find("Kids")
	if !found {
		return pageNr, fmt.Errorf("corrupt \"Kids\" entry %s", pagesDict)
	}

	kids, err := ctx.DereferenceArray(o)
	if err != nil || kids == nil {
		if err != nil {
			return pageNr, fmt.Errorf("dereference Kids: %w", err)
		}
		return pageNr, fmt.Errorf("corrupt \"Kids\" entry: %s", pagesDict)
	}

	for i, v := range kids {
		pageNr, err = parsePageTreeKid(ctx, v, i+1, pageNr)
		if err != nil {
			return 0, err
		}
	}

	return pageNr, nil
}

func traverse(xRefTable *model.XRefTable, value types.Object, duplObjs types.IntSet) error {
	if indRef, ok := value.(types.IndirectRef); ok {
		duplObjs[int(indRef.ObjectNumber)] = true
		o, err := xRefTable.Dereference(indRef)
		if err != nil {
			return fmt.Errorf("obj#%d: dereference duplicate graph object: %w", indRef.ObjectNumber.Value(), err)
		}
		if err := traverseObjectGraphAndMarkDuplicates(xRefTable, o, duplObjs); err != nil {
			return fmt.Errorf("obj#%d: traverse duplicate graph object: %w", indRef.ObjectNumber.Value(), err)
		}
	}
	if d, ok := value.(types.Dict); ok {
		if err := traverseObjectGraphAndMarkDuplicates(xRefTable, d, duplObjs); err != nil {
			return err
		}
	}
	if sd, ok := value.(types.StreamDict); ok {
		if err := traverseObjectGraphAndMarkDuplicates(xRefTable, sd, duplObjs); err != nil {
			return err
		}
	}
	if a, ok := value.(types.Array); ok {
		if err := traverseObjectGraphAndMarkDuplicates(xRefTable, a, duplObjs); err != nil {
			return err
		}
	}

	return nil
}

// Traverse the object graph for a Object and mark all objects as potential duplicates.
func traverseObjectGraphAndMarkDuplicates(xRefTable *model.XRefTable, obj types.Object, duplObjs types.IntSet) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("traverseObjectGraphAndMarkDuplicates begin type=%T\n", obj)
	}

	switch x := obj.(type) {

	case types.Dict:
		if log.OptimizeEnabled() {
			log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: dict")
		}
		for _, value := range x {
			if err := traverse(xRefTable, value, duplObjs); err != nil {
				return fmt.Errorf("dict entry: %w", err)
			}
		}

	case types.StreamDict:
		if log.OptimizeEnabled() {
			log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: streamDict")
		}
		for _, value := range x.Dict {
			if err := traverse(xRefTable, value, duplObjs); err != nil {
				return fmt.Errorf("stream dict entry: %w", err)
			}
		}

	case types.Array:
		if log.OptimizeEnabled() {
			log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: arr")
		}
		for i, value := range x {
			if err := traverse(xRefTable, value, duplObjs); err != nil {
				return fmt.Errorf("array[%d]: %w", i, err)
			}
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("traverseObjectGraphAndMarkDuplicates end")
	}

	return nil
}

// Identify and mark all potential duplicate objects.
func calcRedundantObjects(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantObjects begin")
	}

	for i, fontDict := range ctx.Optimize.DuplicateFonts {
		ctx.Optimize.DuplicateFontObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant font.
		if err := traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, fontDict, ctx.Optimize.DuplicateFontObjs); err != nil {
			return fmt.Errorf("duplicate font obj#%d: traverse object graph: %w", i, err)
		}
	}

	for i, obj := range ctx.Optimize.DuplicateImages {
		ctx.Optimize.DuplicateImageObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant image.
		if err := traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, *obj.ImageDict, ctx.Optimize.DuplicateImageObjs); err != nil {
			return fmt.Errorf("duplicate image obj#%d: traverse object graph: %w", i, err)
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantObjects end")
	}

	return nil
}

func fixCorruptFontResDicts(ctx *model.Context) error {
	// TODO: hacky, also because we don't reall y take the fontDict type into account.
	for _, d := range ctx.Optimize.CorruptFontResDicts {
		for k, v := range d {
			if v == nil {
				for fn, objNrs := range ctx.Optimize.Fonts {

					if strings.HasPrefix(fn, "Arial") && (len(fn) == 5 || fn[5] != '-') {
						model.ShowRepaired(fmt.Sprintf("font %s mapped to objNr %d", k, objNrs[0]))
						d[k] = *types.NewIndirectRef(objNrs[0], 0)
						break
					}
				}
			}
			// if d[k] == nil {
			// 	d[k] = *types.NewIndirectRef(objNrs[0], 0)
			// }
		}
	}
	return nil
}

// Iterate over all pages and optimize resources.
// Get rid of duplicate embedded fonts and images.
func optimizeFontAndImages(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeFontAndImages begin")
	}

	// Get a reference to the PDF indirect reference of the page tree root dict.
	indRefPages, err := ctx.Pages()
	if err != nil {
		return fmt.Errorf("page tree root: %w", err)
	}

	// Dereference and get a reference to the page tree root dict.
	pageTreeRootDict, err := ctx.XRefTable.DereferenceDict(*indRefPages)
	if err != nil {
		return fmt.Errorf("page tree root obj#%d: dereference dict: %w", indRefPages.ObjectNumber.Value(), err)
	}

	// Prepare optimization environment.
	ctx.Optimize.PageFonts = make([]types.IntSet, ctx.PageCount)
	ctx.Optimize.PageImages = make([]types.IntSet, ctx.PageCount)
	ctx.Optimize.FormResourceCache = map[int]types.IntSet{}

	// Iterate over page dicts and optimize resources.
	_, err = parsePagesDict(ctx, pageTreeRootDict, 0)
	if err != nil {
		return fmt.Errorf("page tree: %w", err)
	}

	if err := fixCorruptFontResDicts(ctx); err != nil {
		return fmt.Errorf("fix corrupt font resources: %w", err)
	}

	ctx.Optimize.ContentStreamCache = map[int]*types.StreamDict{}
	ctx.Optimize.FormStreamCache = map[int]*types.StreamDict{}

	// Identify all duplicate objects.
	if err = calcRedundantObjects(ctx); err != nil {
		return fmt.Errorf("calculate redundant objects: %w", err)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeFontAndImages end")
	}

	return nil
}

// Return stream length for font file object.
func streamLengthFontFile(xRefTable *model.XRefTable, indirectRef *types.IndirectRef) (*int64, error) {
	if log.OptimizeEnabled() {
		log.Optimize.Println("streamLengthFontFile begin")
	}

	objectNumber := indirectRef.ObjectNumber

	sd, _, err := xRefTable.DereferenceStreamDict(*indirectRef)
	if err != nil {
		return nil, fmt.Errorf("font file obj#%d: dereference stream dict: %w", objectNumber.Value(), err)
	}

	if sd == nil || (*sd).StreamLength == nil {
		return nil, fmt.Errorf("fontFile Streamlength is nil for object %d", objectNumber)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("streamLengthFontFile end")
	}

	return (*sd).StreamLength, nil
}

// Calculate amount of memory used by embedded fonts for stats.
func calcEmbeddedFontsMemoryUsage(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("calcEmbeddedFontsMemoryUsage begin: %d fontObjects\n", len(ctx.Optimize.FontObjects))
	}

	fontFileIndRefs := map[types.IndirectRef]bool{}

	var objNrs []int

	// Sorting unnecessary.
	for k := range ctx.Optimize.FontObjects {
		objNrs = append(objNrs, k)
	}
	sort.Ints(objNrs)

	// Iterate over all embedded font objects and record font file references.
	for _, objNr := range objNrs {

		fontObject := ctx.Optimize.FontObjects[objNr]

		// Only embedded fonts have binary data.
		ok, err := pdffont.Embedded(ctx.XRefTable, fontObject.FontDict, objNr)
		if err != nil {
			return fmt.Errorf("font obj#%d: detect embedded font: %w", objNr, err)
		}
		if !ok {
			continue
		}

		if err := processFontFilesForFontDict(ctx.XRefTable, fontObject.FontDict, objNr, fontFileIndRefs); err != nil {
			return fmt.Errorf("font obj#%d: collect font files: %w", objNr, err)
		}
	}

	// Iterate over font file references and calculate total font size.
	for ir := range fontFileIndRefs {
		streamLength, err := streamLengthFontFile(ctx.XRefTable, &ir)
		if err != nil {
			return fmt.Errorf("font file obj#%d: stream length: %w", ir.ObjectNumber.Value(), err)
		}
		ctx.Read.BinaryFontSize += *streamLength
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcEmbeddedFontsMemoryUsage end")
	}

	return nil
}

// fontDescriptorFontFileIndirectObjectRef returns the indirect object for the font file for given font descriptor.
func fontDescriptorFontFileIndirectObjectRef(fontDescriptorDict types.Dict) *types.IndirectRef {
	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptorFontFileIndirectObjectRef begin")
	}

	ir := fontDescriptorDict.IndirectRefEntry("FontFile")

	if ir == nil {
		ir = fontDescriptorDict.IndirectRefEntry("FontFile2")
	}

	if ir == nil {
		ir = fontDescriptorDict.IndirectRefEntry("FontFile3")
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("FontDescriptorFontFileIndirectObjectRef end")
	}

	return ir
}

// Record font file objects referenced by this fonts font descriptor for stats and size calculation.
func processFontFilesForFontDict(xRefTable *model.XRefTable, fontDict types.Dict, objectNumber int, indRefsMap map[types.IndirectRef]bool) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("processFontFilesForFontDict begin")
	}

	// Note:
	// "ToUnicode" is also an entry containing binary content that could be inspected for duplicate content.

	d, err := pdffont.FontDescriptor(xRefTable, fontDict, objectNumber)
	if err != nil {
		return fmt.Errorf("font obj#%d: font descriptor: %w", objectNumber, err)
	}

	if d != nil {
		if ir := fontDescriptorFontFileIndirectObjectRef(d); ir != nil {
			indRefsMap[*ir] = true
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("processFontFilesForFontDict end")
	}

	return nil
}

// Calculate amount of memory used by duplicate embedded fonts for stats.
func calcRedundantEmbeddedFontsMemoryUsage(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantEmbeddedFontsMemoryUsage begin")
	}

	fontFileIndRefs := map[types.IndirectRef]bool{}

	// Iterate over all duplicate fonts and record font file references.
	for objectNumber, fontDict := range ctx.Optimize.DuplicateFonts {

		// Duplicate Fonts have to be embedded, so no check here.
		if err := processFontFilesForFontDict(ctx.XRefTable, fontDict, objectNumber, fontFileIndRefs); err != nil {
			return fmt.Errorf("duplicate font obj#%d: collect font files: %w", objectNumber, err)
		}

	}

	// Iterate over font file references and calculate total font size.
	for ir := range fontFileIndRefs {

		streamLength, err := streamLengthFontFile(ctx.XRefTable, &ir)
		if err != nil {
			return fmt.Errorf("duplicate font file obj#%d: stream length: %w", ir.ObjectNumber.Value(), err)
		}

		ctx.Read.BinaryFontDuplSize += *streamLength
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantEmbeddedFontsMemoryUsage end")
	}

	return nil
}

// Calculate amount of memory used by embedded fonts and duplicate embedded fonts for stats.
func calcFontBinarySizes(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcFontBinarySizes begin")
	}

	if err := calcEmbeddedFontsMemoryUsage(ctx); err != nil {
		return fmt.Errorf("embedded fonts: %w", err)
	}

	if err := calcRedundantEmbeddedFontsMemoryUsage(ctx); err != nil {
		return fmt.Errorf("duplicate embedded fonts: %w", err)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcFontBinarySizes end")
	}

	return nil
}

// Calculate amount of memory used by images and duplicate images for stats.
func calcImageBinarySizes(ctx *model.Context) {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcImageBinarySizes begin")
	}

	// Calc memory usage for images.
	for _, imageObject := range ctx.Optimize.ImageObjects {
		ctx.Read.BinaryImageSize += *imageObject.ImageDict.StreamLength
	}

	// Calc memory usage for duplicate images.
	for _, obj := range ctx.Optimize.DuplicateImages {
		ctx.Read.BinaryImageDuplSize += *obj.ImageDict.StreamLength
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcImageBinarySizes end")
	}
}

// Calculate memory usage of binary data for stats.
func calcBinarySizes(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcBinarySizes begin")
	}

	// Calculate font memory usage for stats.
	if err := calcFontBinarySizes(ctx); err != nil {
		return fmt.Errorf("font binary sizes: %w", err)
	}

	// Calculate image memory usage for stats.
	calcImageBinarySizes(ctx)

	// Note: Content streams also represent binary content.

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcBinarySizes end")
	}

	return nil
}

func fixDeepDict(ctx *model.Context, d types.Dict) error {
	for k, v := range d {
		ir, err := fixDeepObject(ctx, v)
		if err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
		if ir != nil {
			d[k] = *ir
		}
	}

	return nil
}

func fixDeepArray(ctx *model.Context, a types.Array) error {
	for i, v := range a {
		ir, err := fixDeepObject(ctx, v)
		if err != nil {
			return fmt.Errorf("[%d]: %w", i, err)
		}
		if ir != nil {
			a[i] = *ir
		}
	}

	return nil
}

func fixDirectObject(ctx *model.Context, o types.Object) error {
	switch o := o.(type) {
	case types.Dict:
		for k, v := range o {
			ir, err := fixDeepObject(ctx, v)
			if err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
			if ir != nil {
				o[k] = *ir
			}
		}
	case types.Array:
		for i, v := range o {
			ir, err := fixDeepObject(ctx, v)
			if err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
			if ir != nil {
				o[i] = *ir
			}
		}
	}

	return nil
}

func fixIndirectObject(ctx *model.Context, ir *types.IndirectRef) error {
	objNr := int(ir.ObjectNumber)

	if ctx.Optimize.Cache[objNr] {
		return nil
	}
	ctx.Optimize.Cache[objNr] = true

	entry, found := ctx.Find(objNr)
	if !found {
		return nil
	}

	if entry.Free {
		// This is a reference to a free object that needs to be fixed.

		//fmt.Printf("fixNullObject: #%d g%d\n", objNr, genNr)

		if ctx.Optimize.NullObjNr == nil {
			nr, err := ctx.InsertObject(nil)
			if err != nil {
				return fmt.Errorf("insert null object for free obj#%d: %w", objNr, err)
			}
			ctx.Optimize.NullObjNr = &nr
		}

		ir.ObjectNumber = types.Integer(*ctx.Optimize.NullObjNr)

		return nil
	}

	var err error

	switch o := entry.Object.(type) {

	case types.Dict:
		err = fixDeepDict(ctx, o)

	case types.StreamDict:
		err = fixDeepDict(ctx, o.Dict)

	case types.Array:
		err = fixDeepArray(ctx, o)

	}

	if err != nil {
		return fmt.Errorf("obj#%d: %w", objNr, err)
	}
	return nil
}

func fixDeepObject(ctx *model.Context, o types.Object) (*types.IndirectRef, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return nil, fixDirectObject(ctx, o)
	}

	err := fixIndirectObject(ctx, &ir)
	return &ir, err
}

func fixReferencesToFreeObjects(ctx *model.Context) error {
	if err := fixDirectObject(ctx, ctx.RootDict); err != nil {
		return fmt.Errorf("root dict: %w", err)
	}
	return nil
}

// CacheFormFonts caches form fonts referenced by ctx.
func CacheFormFonts(ctx *model.Context) error {
	d, err := primitives.FormFontResDict(ctx.XRefTable)
	if err != nil {
		return fmt.Errorf("form font resources: %w", err)
	}

	// Iterate over font resource dict.
	for rName, v := range d {

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: processing font: %s, %s\n", rName, indRef)
		}

		objNr := int(indRef.ObjectNumber)

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: objectNumber = %d\n", objNr)
		}

		fontDict, err := ctx.DereferenceFontDict(indRef)
		if err != nil {
			return fmt.Errorf("form font obj#%d: dereference font dict: %w", objNr, err)
		}
		if fontDict == nil {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: fontDict: %s\n", fontDict)
		}

		// Get the unique font name.
		prefix, fName, err := pdffont.Name(ctx.XRefTable, fontDict, objNr)
		if err != nil {
			return fmt.Errorf("form font obj#%d: parse font name: %w", objNr, err)
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: baseFont: prefix=%s name=%s\n", prefix, fName)
		}

		registerFontDictObjNr(ctx, fName, objNr)

		ctx.Optimize.FormFontObjects[objNr] =
			&model.FontObject{
				ResourceNames: []string{rName},
				Prefix:        prefix,
				FontName:      fName,
				FontDict:      fontDict,
			}
	}

	return nil
}

func optimizeResourceDicts(ctx *model.Context) error {
	if err := ctx.ConsolidatePageResources(); err != nil {
		return err
	}
	// TODO Remove resource dicts from inner nodes.
	return nil
}

func resolveWidth(ctx *model.Context, sd *types.StreamDict) error {
	if obj, ok := sd.Find("Width"); ok {
		w, err := ctx.DereferenceNumber(obj)
		if err != nil {
			return fmt.Errorf("dereference Width: %w", err)
		}
		sd.Dict["Width"] = types.Integer(w)
	}
	return nil
}

func ensureDirectWidthForXObjs(ctx *model.Context) error {
	for _, imgObjs := range ctx.Optimize.PageImages {
		for objNr, v := range imgObjs {
			if v {
				imageObj := ctx.Optimize.ImageObjects[objNr]
				if err := resolveWidth(ctx, imageObj.ImageDict); err != nil {
					return fmt.Errorf("image obj#%d: resolve width: %w", objNr, err)
				}
			}
		}
	}
	return nil
}

// OptimizeXRefTable optimizes an xRefTable by locating and getting rid of redundant embedded fonts and images.
func OptimizeXRefTable(ctx *model.Context) error {
	if ctx.PageCount == 0 {
		return nil
	}

	// Sometimes free objects are used although they are part of the free object list.
	// Replace references to free xref table entries with a reference to a NULL object.
	if err := fixReferencesToFreeObjects(ctx); err != nil {
		return fmt.Errorf("fix references to free objects: %w", err)
	}

	if (ctx.Cmd == model.VALIDATE ||
		ctx.Cmd == model.OPTIMIZE ||
		ctx.Cmd == model.LISTIMAGES ||
		ctx.Cmd == model.EXTRACTIMAGES ||
		ctx.Cmd == model.UPDATEIMAGES) &&
		ctx.Conf.OptimizeResourceDicts {
		// Extra step with potential for performance hit when processing large files.
		if err := optimizeResourceDicts(ctx); err != nil {
			return fmt.Errorf("optimize resources: %w", err)
		}
	}

	// Get rid of duplicate embedded fonts and images.
	if err := optimizeFontAndImages(ctx); err != nil {
		return fmt.Errorf("optimize fonts and images: %w", err)
	}

	if err := ensureDirectWidthForXObjs(ctx); err != nil {
		return fmt.Errorf("resolve image widths: %w", err)
	}

	// Get rid of PieceInfo dict from root.
	if err := ctx.DeleteDictEntry(ctx.RootDict, "PieceInfo"); err != nil {
		return fmt.Errorf("delete root PieceInfo: %w", err)
	}

	// Calculate memory usage of binary content for stats.
	if log.StatsEnabled() {
		if err := calcBinarySizes(ctx); err != nil {
			return fmt.Errorf("calculate binary sizes: %w", err)
		}
	}

	ctx.Optimized = true

	return nil
}

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
	"fmt"
	"sort"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// Mark all content streams for a page dictionary (for stats).
func identifyPageContent(xRefTable *XRefTable, pageDict Dict, pageNumber, pageObjNumber int) error {

	log.Optimize.Println("identifyPageContent begin")

	o, found := pageDict.Find("Contents")
	if !found {
		log.Optimize.Println("identifyPageContent end: no \"Contents\"")
		return nil
	}

	var contentArr Array

	if ir, ok := o.(IndirectRef); ok {

		entry, found := xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
		if !found {
			return errors.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(StreamDict)
		if ok {
			contentStreamDict.IsPageContent = true
			entry.Object = contentStreamDict
			log.Optimize.Printf("identifyPageContent end: ok obj#%d\n", ir.ObjectNumber.Value())
			return nil
		}

		contentArr, ok = entry.Object.(Array)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d page content entry neither stream dict nor array.\n", pageObjNumber)
		}

	} else if contentArr, ok = o.(Array); !ok {
		return errors.Errorf("identifyPageContent: obj#:%d corrupt page content array\n", pageObjNumber)
	}

	for _, c := range contentArr {

		ir, ok := c.(IndirectRef)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d corrupt page content array entry\n", pageObjNumber)
		}

		entry, found := xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
		if !found {
			return errors.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(StreamDict)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d page content entry is no stream dict\n", pageObjNumber)
		}

		contentStreamDict.IsPageContent = true
		entry.Object = contentStreamDict
		log.Optimize.Printf("identifyPageContent: ok obj#%d\n", ir.GenerationNumber.Value())
	}

	log.Optimize.Println("identifyPageContent end")

	return nil
}

// resourcesDictForPageDict returns the resource dict for a page dict if there is any.
func resourcesDictForPageDict(xRefTable *XRefTable, pageDict Dict, pageObjNumber int) (Dict, error) {

	o, found := pageDict.Find("Resources")
	if !found {
		log.Optimize.Printf("resourcesDictForPageDict end: No resources dict for page object %d, may be inheritated\n", pageObjNumber)
		return nil, nil
	}

	return xRefTable.DereferenceDict(o)
}

// handleDuplicateFontObject returns nil or the object number of the registered font if it matches this font.
func handleDuplicateFontObject(ctx *Context, fontDict Dict, fName, rName string, objNr, pageNumber int) (*int, error) {

	// Get a slice of all font object numbers for font name.
	fontObjNrs, found := ctx.Optimize.Fonts[fName]
	if !found {
		// There is no registered font with fName.
		return nil, nil
	}

	// Get the set of font object numbers for pageNumber.
	pageFonts := ctx.Optimize.PageFonts[pageNumber]

	// Iterate over all registered font object numbers for font name.
	// Check if this font dict matches the font dict of each font object number.
	for _, fontObjNr := range fontObjNrs {

		// Get the font object from the lookup table.
		fontObject := ctx.Optimize.FontObjects[fontObjNr]

		log.Optimize.Printf("handleDuplicateFontObject: comparing with fontDict Obj %d\n", fontObjNr)

		// Check if the input fontDict matches the fontDict of this fontObject.
		ok, err := equalFontDicts(fontObject.FontDict, fontDict, ctx.XRefTable)
		if err != nil {
			return nil, err
		}

		if !ok {
			// No match!
			continue
		}

		// We have detected a redundant font dict!
		log.Optimize.Printf("handleDuplicateFontObject: redundant fontObj#:%d basefont %s already registered with obj#:%d !\n", objNr, fName, fontObjNr)

		// Register new page font with pageNumber.
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

func pageImages(ctx *Context, pageNumber int) IntSet {

	pageImages := ctx.Optimize.PageImages[pageNumber]

	if pageImages == nil {
		pageImages = IntSet{}
		ctx.Optimize.PageImages[pageNumber] = pageImages
	}

	return pageImages
}

func pageFonts(ctx *Context, pageNumber int) IntSet {

	pageFonts := ctx.Optimize.PageFonts[pageNumber]

	if pageFonts == nil {
		pageFonts = IntSet{}
		ctx.Optimize.PageFonts[pageNumber] = pageFonts
	}

	return pageFonts
}

func fontName(ctx *Context, fontDict Dict, objNumber int) (prefix, fontName string, err error) {

	var found bool
	var o Object

	if *fontDict.Subtype() != "Type3" {

		o, found = fontDict.Find("BaseFont")
		if !found {
			o, found = fontDict.Find("Name")
			if !found {
				return "", "", errors.New("fontName: missing fontDict entries \"BaseFont\" and \"Name\"")
			}
		}

	} else {

		// Type3 fonts only have Name in V1.0 else use generic name.

		o, found = fontDict.Find("Name")
		if !found {
			return "", fmt.Sprintf("Type3_%d", objNumber), nil
		}

	}

	o, err = ctx.Dereference(o)
	if err != nil {
		return "", "", err
	}

	baseFont, ok := o.(Name)
	if !ok {
		return "", "", errors.New("fontName: corrupt fontDict entry BaseFont")
	}

	n := string(baseFont)

	// Isolate Postscript prefix.
	var p string

	i := strings.Index(n, "+")

	if i > 0 {
		p = n[:i]
		n = n[i+1:]
	}

	return p, n, nil
}

// Get rid of redundant fonts for given fontResources dictionary.
func optimizeFontResourcesDict(ctx *Context, rDict Dict, pageNumber, pageObjNumber int) error {

	log.Optimize.Printf("optimizeFontResourcesDict begin: page=%d pageObjNumber=%d %s\nPageFonts=%v\n", pageNumber, pageObjNumber, rDict, ctx.Optimize.PageFonts)

	pageFonts := pageFonts(ctx, pageNumber)

	// Iterate over font resource dict.
	for rName, v := range rDict {

		indRef, ok := v.(IndirectRef)
		if !ok {
			return errors.Errorf("optimizeFontResourcesDict: missing indirect object ref for Font: %s\n", rName)
		}

		log.Optimize.Printf("optimizeFontResourcesDict: processing font: %s, %s\n", rName, indRef)
		objNr := int(indRef.ObjectNumber)
		log.Optimize.Printf("optimizeFontResourcesDict: objectNumber = %d\n", objNr)

		if _, found := ctx.Optimize.FontObjects[objNr]; found {
			// This font has already been registered.
			//logInfoOptimizePrintf("optimizeFontResourcesDict: Fontobject %d already registered\n", objectNumber)
			pageFonts[objNr] = true
			continue
		}

		// We are dealing with a new font.
		// Dereference the font dict.
		fontDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return err
		}

		log.Optimize.Printf("optimizeFontResourcesDict: fontDict: %s\n", fontDict)

		if fontDict.Type() == nil {
			return errors.Errorf("optimizeFontResourcesDict: missing dict type %s\n", v)
		}

		if *fontDict.Type() != "Font" {
			return errors.Errorf("optimizeFontResourcesDict: expected Type=Font, unexpected Type: %s", *fontDict.Type())
		}

		// Get the unique font name.
		prefix, fName, err := fontName(ctx, fontDict, objNr)
		if err != nil {
			return err
		}
		log.Optimize.Printf("optimizeFontResourcesDict: baseFont: prefix=%s name=%s\n", prefix, fName)

		// Check if fontDict is a duplicate and if so return the object number of the original.
		originalObjNr, err := handleDuplicateFontObject(ctx, fontDict, fName, rName, objNr, pageNumber)
		if err != nil {
			return err
		}

		if originalObjNr != nil {
			// We have identified a redundant fontDict!
			// Update font resource dict so that rName points to the original.
			rDict[rName] = *NewIndirectRef(*originalObjNr, 0)
			continue
		}

		// Register new font dict.
		log.Optimize.Printf("optimizeFontResourcesDict: adding new font %s obj#%d\n", fName, objNr)

		fontObjNrs, found := ctx.Optimize.Fonts[fName]
		if found {
			log.Optimize.Printf("optimizeFontResourcesDict: appending %d to %s\n", objNr, fName)
			ctx.Optimize.Fonts[fName] = append(fontObjNrs, objNr)
		} else {
			ctx.Optimize.Fonts[fName] = []int{objNr}
		}

		ctx.Optimize.FontObjects[objNr] =
			&FontObject{
				ResourceNames: []string{rName},
				Prefix:        prefix,
				FontName:      fName,
				FontDict:      fontDict,
			}

		pageFonts[objNr] = true

	}

	log.Optimize.Println("optimizeFontResourcesDict end:")

	return nil
}

// handleDuplicateImageObject returns nil or the object number of the registered image if it matches this image.
func handleDuplicateImageObject(ctx *Context, imageDict *StreamDict, resourceName string, objNr, pageNumber int) (*int, error) {

	// Get the set of image object numbers for pageNumber.
	pageImages := ctx.Optimize.PageImages[pageNumber]

	// Process image dict, check if this is a duplicate.
	for imageObjNr, imageObject := range ctx.Optimize.ImageObjects {

		log.Optimize.Printf("handleDuplicateImageObject: comparing with imagedict Obj %d\n", imageObjNr)

		// Check if the input imageDict matches the imageDict of this imageObject.
		ok, err := equalStreamDicts(imageObject.ImageDict, imageDict, ctx.XRefTable)
		if err != nil {
			return nil, err
		}

		if !ok {
			// No match!
			continue
		}

		// We have detected a redundant image dict.
		log.Optimize.Printf("handleDuplicateImageObject: redundant imageObj#:%d already registered with obj#:%d !\n", objNr, imageObjNr)

		// Register new page image for pageNumber.
		// The image for image object number is used instead of objNr.
		pageImages[imageObjNr] = true

		// Add the resource name of this duplicate image to the list of registered resource names.
		imageObject.AddResourceName(resourceName)

		// Register imageDict as duplicate.
		ctx.Optimize.DuplicateImages[objNr] = imageDict

		// Return the imageObjectNumber that will be used instead of objNr.
		return &imageObjNr, nil
	}

	return nil, nil
}

// Get rid of redundant XObjects e.g. embedded images.
func optimizeXObjectResourcesDict(ctx *Context, rDict Dict, pageNumber, pageObjNumber int) error {

	log.Optimize.Printf("optimizeXObjectResourcesDict begin: %s\n", rDict)

	pageImages := pageImages(ctx, pageNumber)

	// Iterate over XObject resource dict.
	for rName, v := range rDict {

		indRef, ok := v.(IndirectRef)
		if !ok {
			return errors.Errorf("optimizeXObjectResourcesDict: missing indirect object ref for resourceId: %s", rName)
		}

		log.Optimize.Printf("optimizeXObjectResourcesDict: processing xobject: %s, %s\n", rName, indRef)
		objNr := int(indRef.ObjectNumber)
		log.Optimize.Printf("optimizeXObjectResourcesDict: objectNumber = %d\n", objNr)

		// We are dealing with a new XObject..
		// Dereference the XObject stream dict.
		osd, err := ctx.DereferenceStreamDict(indRef)
		if err != nil {
			return err
		}

		log.Optimize.Printf("optimizeXObjectResourcesDict: dereferenced obj:%d\n%s", objNr, osd)

		if osd.Dict.Subtype() == nil {
			return errors.Errorf("optimizeXObjectResourcesDict: missing stream dict Subtype %s\n", v)
		}

		if *osd.Dict.Subtype() == "Image" {

			// Already registered image object that appears in different resources dicts.
			if _, found := ctx.Optimize.ImageObjects[objNr]; found {
				// This image has already been registered.
				//log.Optimize.Printf("optimizeXObjectResourcesDict: Imageobject %d already registered\n", objNr)
				pageImages[objNr] = true
				continue
			}

			// Check if image is a duplicate and if so return the object number of the original.
			originalObjNr, err := handleDuplicateImageObject(ctx, osd, rName, objNr, pageNumber)
			if err != nil {
				return err
			}

			if originalObjNr != nil {
				// We have identified a redundant image!
				// Update xobject resource dict so that rName points to the original.
				rDict[rName] = *NewIndirectRef(*originalObjNr, 0)
				continue
			}

			// Register new image dict.
			log.Optimize.Printf("optimizeXObjectResourcesDict: adding new image obj#%d\n", objNr)

			ctx.Optimize.ImageObjects[objNr] =
				&ImageObject{
					ResourceNames: []string{rName},
					ImageDict:     osd,
				}

			pageImages[objNr] = true

		}

		if *osd.Subtype() != "Form" {
			log.Optimize.Printf("optimizeXObjectResourcesDict: unexpected stream dict Subtype %s\n", *osd.Dict.Subtype())
			continue
		}

		// Process form dict
		log.Optimize.Printf("optimizeXObjectResourcesDict: parsing form dict obj:%d\n", objNr)
		parseResourcesDict(ctx, osd.Dict, pageNumber, objNr)
	}

	log.Optimize.Println("optimizeXObjectResourcesDict end")

	return nil
}

// Optimize given resource dictionary by removing redundant fonts and images.
func optimizeResources(ctx *Context, resourcesDict Dict, pageNumber, pageObjNumber int) error {

	log.Optimize.Printf("optimizeResources begin: pageNumber=%d pageObjNumber=%d\n", pageNumber, pageObjNumber)

	if resourcesDict == nil {
		log.Optimize.Printf("optimizeResources end: No resources dict available")
		return nil
	}

	// Process Font resource dict, get rid of redundant fonts.
	o, found := resourcesDict.Find("Font")
	if found {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d == nil {
			return errors.Errorf("optimizeResources: font resource dict is null for page %d pageObj %d\n", pageNumber, pageObjNumber)
		}

		err = optimizeFontResourcesDict(ctx, d, pageNumber, pageObjNumber)
		if err != nil {
			return err
		}

	}

	// Note: An optional ExtGState resource dict may contain binary content in the following entries: "SMask", "HT".

	// Process XObject resource dict, get rid of redundant images.
	o, found = resourcesDict.Find("XObject")
	if found {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d == nil {
			return errors.Errorf("optimizeResources: xobject resource dict is null for page %d pageObj %d\n", pageNumber, pageObjNumber)
		}

		err = optimizeXObjectResourcesDict(ctx, d, pageNumber, pageObjNumber)
		if err != nil {
			return err
		}

	}

	log.Optimize.Println("optimizeResources end")

	return nil
}

// Process the resources dictionary for given page number and optimize by removing redundant resources.
func parseResourcesDict(ctx *Context, pageDict Dict, pageNumber, pageObjNumber int) error {

	log.Optimize.Printf("parseResourcesDict begin page: %d, object:%d\n", pageNumber+1, pageObjNumber)

	// Get resources dict for this page.
	d, err := resourcesDictForPageDict(ctx.XRefTable, pageDict, pageObjNumber)
	if err != nil {
		return err
	}

	// dict may be nil for inheritated resource dicts.
	if d != nil {

		// Optimize image and font resources.
		err = optimizeResources(ctx, d, pageNumber, pageObjNumber)
		if err != nil {
			return err
		}

	}

	log.Optimize.Printf("parseResourcesDict end page: %d, object:%d\n", pageNumber+1, pageObjNumber)

	return nil
}

// Iterate over all pages and optimize resources.
func parsePagesDict(ctx *Context, pagesDict Dict, pageNumber int) (int, error) {

	log.Optimize.Printf("parsePagesDict begin (next page=%d): %s\n", pageNumber+1, pagesDict)

	// Get number of pages of this PDF file.
	count, found := pagesDict.Find("Count")
	if !found {
		return 0, errors.New("parsePagesDict: missing Count")
	}

	log.Optimize.Printf("parsePagesDict: This page node has %d pages\n", int(count.(Integer)))

	// Iterate over page tree.
	//kidsArray := pagesDict.ArrayEntry("Kids")
	for _, v := range pagesDict.ArrayEntry("Kids") {

		// Dereference next page node dict.
		ir, _ := v.(IndirectRef)
		log.Optimize.Printf("parsePagesDict PageNode: %s\n", ir)
		o, err := ctx.Dereference(ir)
		if err != nil {
			return 0, errors.Wrap(err, "parsePagesDict: can't locate Pagedict or Pagesdict")
		}

		pageNodeDict := o.(Dict)
		dictType := pageNodeDict.Type()
		if dictType == nil {
			return 0, errors.New("parsePagesDict: Missing dict type")
		}

		// Note: Pages may contain a to be inheritated ResourcesDict.

		if *dictType == "Pages" {

			// Recurse over pagetree and optimize resources.
			pageNumber, err = parsePagesDict(ctx, pageNodeDict, pageNumber)
			if err != nil {
				return 0, err
			}

			continue
		}

		if *dictType != "Page" {
			return 0, errors.Errorf("parsePagesDict: Unexpected dict type: %s\n", *dictType)
		}

		// Mark page content streams for stats.
		err = identifyPageContent(ctx.XRefTable, pageNodeDict, pageNumber, int(ir.ObjectNumber))
		if err != nil {
			return 0, err
		}

		// Parse and optimize resource dict for one page.
		err = parseResourcesDict(ctx, pageNodeDict, pageNumber, int(ir.ObjectNumber))
		if err != nil {
			return 0, err
		}

		pageNumber++
	}

	log.Optimize.Printf("parsePagesDict end: %s\n", pagesDict)

	return pageNumber, nil
}

func traverse(xRefTable *XRefTable, value Object, duplObjs IntSet) error {

	if indRef, ok := value.(IndirectRef); ok {
		duplObjs[int(indRef.ObjectNumber)] = true
		o, err := xRefTable.Dereference(indRef)
		if err != nil {
			return err
		}
		traverseObjectGraphAndMarkDuplicates(xRefTable, o, duplObjs)
	}
	if d, ok := value.(Dict); ok {
		traverseObjectGraphAndMarkDuplicates(xRefTable, d, duplObjs)
	}
	if sd, ok := value.(StreamDict); ok {
		traverseObjectGraphAndMarkDuplicates(xRefTable, sd, duplObjs)
	}
	if a, ok := value.(Array); ok {
		traverseObjectGraphAndMarkDuplicates(xRefTable, a, duplObjs)
	}

	return nil
}

// Traverse the object graph for a Object and mark all objects as potential duplicates.
func traverseObjectGraphAndMarkDuplicates(xRefTable *XRefTable, obj Object, duplObjs IntSet) error {

	log.Optimize.Printf("traverseObjectGraphAndMarkDuplicates begin type=%T\n", obj)

	switch x := obj.(type) {

	case Dict:
		log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: dict.")
		for _, value := range x {
			err := traverse(xRefTable, value, duplObjs)
			if err != nil {
				return err
			}
		}

	case StreamDict:
		log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: streamDict.")
		for _, value := range x.Dict {
			err := traverse(xRefTable, value, duplObjs)
			if err != nil {
				return err
			}
		}

	case Array:
		log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: arr.")
		for _, value := range x {
			err := traverse(xRefTable, value, duplObjs)
			if err != nil {
				return err
			}
		}
	}

	log.Optimize.Println("traverseObjectGraphAndMarkDuplicates end")

	return nil
}

// Identify and mark all potential duplicate objects.
func calcRedundantObjects(ctx *Context) error {

	log.Optimize.Println("calcRedundantObjects begin")

	for i, fontDict := range ctx.Optimize.DuplicateFonts {
		ctx.Optimize.DuplicateFontObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant font.
		err := traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, fontDict, ctx.Optimize.DuplicateFontObjs)
		if err != nil {
			return err
		}
	}

	for i, sd := range ctx.Optimize.DuplicateImages {
		ctx.Optimize.DuplicateImageObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant image.
		err := traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, *sd, ctx.Optimize.DuplicateImageObjs)
		if err != nil {
			return err
		}
	}

	log.Optimize.Println("calcRedundantObjects end")

	return nil
}

// Iterate over all pages and optimize resources.
// Get rid of duplicate embedded fonts and images.
func optimizeFontAndImages(ctx *Context) error {

	log.Optimize.Println("optimizeFontAndImages begin")

	// Get a reference to the PDF indirect reference of the page tree root dict.
	indRefPages, err := ctx.Pages()
	if err != nil {
		return err
	}

	// Dereference and get a reference to the page tree root dict.
	pageTreeRootDict, err := ctx.XRefTable.DereferenceDict(*indRefPages)
	if err != nil {
		return err
	}

	// Detect the number of pages of this PDF file.
	pageCount := pageTreeRootDict.IntEntry("Count")
	if pageCount == nil {
		return errors.New("optimizeFontAndImagess: missing \"Count\" in page root dict")
	}

	// If PageCount already set by validation doublecheck.
	if ctx.PageCount > 0 && ctx.PageCount != *pageCount {
		return errors.New("optimizeFontAndImagess: unexpected page root dict pageCount discrepancy")
	}

	// If we optimize w/o prior validation, set PageCount.
	if ctx.PageCount == 0 {
		ctx.PageCount = *pageCount
	}

	// Prepare optimization environment.
	ctx.Optimize.PageFonts = make([]IntSet, ctx.PageCount)
	ctx.Optimize.PageImages = make([]IntSet, ctx.PageCount)

	// Iterate over page dicts and optimize resources.
	_, err = parsePagesDict(ctx, pageTreeRootDict, 0)
	if err != nil {
		return err
	}

	// Identify all duplicate objects.
	err = calcRedundantObjects(ctx)
	if err != nil {
		return err
	}

	log.Optimize.Println("optimizeFontAndImages end")

	return nil
}

// Return stream length for font file object.
func streamLengthFontFile(xRefTable *XRefTable, indirectRef *IndirectRef) (*int64, error) {

	log.Optimize.Println("streamLengthFontFile begin")

	objectNumber := indirectRef.ObjectNumber

	sd, err := xRefTable.DereferenceStreamDict(*indirectRef)
	if err != nil {
		return nil, err
	}

	if sd == nil || (*sd).StreamLength == nil {
		return nil, errors.Errorf("streamLengthFontFile: fontFile Streamlength is nil for object %d\n", objectNumber)
	}

	log.Optimize.Println("streamLengthFontFile end")

	return (*sd).StreamLength, nil
}

// Calculate amount of memory used by embedded fonts for stats.
func calcEmbeddedFontsMemoryUsage(ctx *Context) error {

	log.Optimize.Printf("calcEmbeddedFontsMemoryUsage begin: %d fontObjects\n", len(ctx.Optimize.FontObjects))

	fontFileIndRefs := map[IndirectRef]bool{}

	var objectNumbers []int

	// Sorting unnecessary.
	for k := range ctx.Optimize.FontObjects {
		objectNumbers = append(objectNumbers, k)
	}
	sort.Ints(objectNumbers)

	// Iterate over all embedded font objects and record font file references.
	for _, objectNumber := range objectNumbers {

		fontObject := ctx.Optimize.FontObjects[objectNumber]

		// Only embedded fonts have binary data.
		if !fontObject.Embedded() {
			continue
		}

		err := processFontFilesForFontDict(ctx.XRefTable, fontObject.FontDict, objectNumber, fontFileIndRefs)
		if err != nil {
			return err
		}
	}

	// Iterate over font file references and calculate total font size.
	for ir := range fontFileIndRefs {
		streamLength, err := streamLengthFontFile(ctx.XRefTable, &ir)
		if err != nil {
			return err
		}
		ctx.Read.BinaryFontSize += *streamLength
	}

	log.Optimize.Println("calcEmbeddedFontsMemoryUsage end")

	return nil
}

// fontDescriptorFontFileIndirectObjectRef returns the indirect object for the font file for given font descriptor.
func fontDescriptorFontFileIndirectObjectRef(fontDescriptorDict Dict) *IndirectRef {

	log.Optimize.Println("fontDescriptorFontFileIndirectObjectRef begin")

	ir := fontDescriptorDict.IndirectRefEntry("FontFile")

	if ir == nil {
		ir = fontDescriptorDict.IndirectRefEntry("FontFile2")
	}

	if ir == nil {
		ir = fontDescriptorDict.IndirectRefEntry("FontFile3")
	}

	if ir == nil {
		//logInfoReader.Printf("FontDescriptorFontFileLength: FontDescriptor dict without fontFile: \n%s\n", fontDescriptorDict)
	}

	log.Optimize.Println("FontDescriptorFontFileIndirectObjectRef end")

	return ir
}

func trivialFontDescriptor(xRefTable *XRefTable, fontDict Dict, objNr int) (Dict, error) {

	o, ok := fontDict.Find("FontDescriptor")
	if !ok {
		return nil, nil
	}

	// fontDescriptor directly available.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	if d == nil {
		return nil, errors.Errorf("trivialFontDescriptor: FontDescriptor is null for font object %d\n", objNr)
	}

	if d.Type() != nil && *d.Type() != "FontDescriptor" {
		return nil, errors.Errorf("trivialFontDescriptor: FontDescriptor dict incorrect dict type for font object %d\n", objNr)
	}

	return d, nil
}

// FontDescriptor gets the font descriptor for this font.
func fontDescriptor(xRefTable *XRefTable, fontDict Dict, objNr int) (Dict, error) {

	log.Optimize.Println("fontDescriptor begin")

	d, err := trivialFontDescriptor(xRefTable, fontDict, objNr)
	if err != nil {
		return nil, err
	}
	if d != nil {
		return d, nil
	}

	// Try to access a fontDescriptor in a Descendent font for Type0 fonts.

	o, ok := fontDict.Find("DescendantFonts")
	if !ok {
		//logErrorOptimize.Printf("FontDescriptor: Neither FontDescriptor nor DescendantFonts for font object %d\n", objectNumber)
		return nil, nil
	}

	// A descendant font is contained in an array of size 1.

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, errors.Errorf("fontDescriptor: DescendantFonts: IndirectRef or Array wth length 1 expected for font object %d\n", objNr)
	}
	if len(a) > 1 {
		return nil, errors.Errorf("fontDescriptor: DescendantFonts Array length > 1 %v\n", a)
	}

	// dict is the fontDict of the descendant font.
	d, err = xRefTable.DereferenceDict(a[0])
	if err != nil {
		return nil, errors.Errorf("fontDescriptor: No descendant font dict for %v\n", a)
	}
	if d == nil {
		return nil, errors.Errorf("fontDescriptor: descendant font dict is null for %v\n", a)
	}

	if *d.Type() != "Font" {
		return nil, errors.Errorf("fontDescriptor: font dict with incorrect dict type for %v\n", d)
	}

	o, ok = d.Find("FontDescriptor")
	if !ok {
		log.Optimize.Printf("fontDescriptor: descendant font not embedded %s\n", d)
		return nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, errors.Errorf("fontDescriptor: No FontDescriptor dict for font object %d\n", objNr)
	}
	if d == nil {
		return nil, errors.Errorf("fontDescriptor: FontDescriptor dict is null for font object %d\n", objNr)
	}

	if d.Type() == nil {
		//logErrorOptimize.Printf("FontDescriptor: FontDescriptor without type \"FontDescriptor\" objNumber:%d\n", objNr)
	} else if *d.Type() != "FontDescriptor" {
		return nil, errors.Errorf("fontDescriptor: FontDescriptor dict incorrect dict type for font object %d\n", objNr)
	}

	log.Optimize.Println("fontDescriptor end")

	return d, nil
}

// Record font file objects referenced by this fonts font descriptor for stats and size calculation.
func processFontFilesForFontDict(xRefTable *XRefTable, fontDict Dict, objectNumber int, indRefsMap map[IndirectRef]bool) error {

	log.Optimize.Println("processFontFilesForFontDict begin")

	// Note:
	// "ToUnicode" is also an entry containing binary content that could be inspected for duplicate content.

	d, err := fontDescriptor(xRefTable, fontDict, objectNumber)
	if err != nil {
		return err
	}

	if d != nil {
		if ir := fontDescriptorFontFileIndirectObjectRef(d); ir != nil {
			indRefsMap[*ir] = true
		}
	}

	log.Optimize.Println("processFontFilesForFontDict end")

	return nil
}

// Calculate amount of memory used by duplicate embedded fonts for stats.
func calcRedundantEmbeddedFontsMemoryUsage(ctx *Context) error {

	log.Optimize.Println("calcRedundantEmbeddedFontsMemoryUsage begin")

	fontFileIndRefs := map[IndirectRef]bool{}

	// Iterate over all duplicate fonts and record font file references.
	for objectNumber, fontDict := range ctx.Optimize.DuplicateFonts {

		// Duplicate Fonts have to be embedded, so no check here.
		if err := processFontFilesForFontDict(ctx.XRefTable, fontDict, objectNumber, fontFileIndRefs); err != nil {
			return err
		}

	}

	// Iterate over font file references and calculate total font size.
	for ir := range fontFileIndRefs {

		streamLength, err := streamLengthFontFile(ctx.XRefTable, &ir)
		if err != nil {
			return err
		}

		ctx.Read.BinaryFontDuplSize += *streamLength
	}

	log.Optimize.Println("calcRedundantEmbeddedFontsMemoryUsage end")

	return nil
}

// Calculate amount of memory used by embedded fonts and duplicate embedded fonts for stats.
func calcFontBinarySizes(ctx *Context) error {

	log.Optimize.Println("calcFontBinarySizes begin")

	err := calcEmbeddedFontsMemoryUsage(ctx)
	if err != nil {
		return err
	}

	err = calcRedundantEmbeddedFontsMemoryUsage(ctx)
	if err != nil {
		return err
	}

	log.Optimize.Println("calcFontBinarySizes end")

	return nil
}

// Calculate amount of memory used by images and duplicate images for stats.
func calcImageBinarySizes(ctx *Context) {

	log.Optimize.Println("calcImageBinarySizes begin")

	// Calc memory usage for images.
	for _, imageObject := range ctx.Optimize.ImageObjects {
		ctx.Read.BinaryImageSize += *imageObject.ImageDict.StreamLength
	}

	// Calc memory usage for duplicate images.
	for _, imageDict := range ctx.Optimize.DuplicateImages {
		ctx.Read.BinaryImageDuplSize += *imageDict.StreamLength
	}

	log.Optimize.Println("calcImageBinarySizes end")
}

// Calculate memory usage of binary data for stats.
func calcBinarySizes(ctx *Context) error {

	log.Optimize.Println("calcBinarySizes begin")

	// Calculate font memory usage for stats.
	err := calcFontBinarySizes(ctx)
	if err != nil {
		return err
	}

	// Calculate image memory usage for stats.
	calcImageBinarySizes(ctx)

	// Note: Content streams also represent binary content.

	log.Optimize.Println("calcBinarySizes end")

	return nil
}

// OptimizeXRefTable optimizes an xRefTable by locating and getting rid of redundant embedded fonts and images.
func OptimizeXRefTable(ctx *Context) error {

	log.Info.Println("optimizing fonts & images")

	log.Optimize.Println("optimizeXRefTable begin")

	// Get rid of duplicate embedded fonts and images.
	err := optimizeFontAndImages(ctx)
	if err != nil {
		return err
	}

	// Calculate memory usage of binary content for stats.
	err = calcBinarySizes(ctx)
	if err != nil {
		return err
	}

	ctx.Optimized = true

	log.Optimize.Println("optimizeXRefTable end")

	return nil
}

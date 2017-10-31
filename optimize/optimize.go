// Package optimize contains code for optimizing the resources of a PDF file.
//
// Subject of optimization are embedded font files and images.
package optimize

import (
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var logDebugOptimize, logInfoOptimize, logErrorOptimize, logStatsOptimize *log.Logger

func init() {
	logDebugOptimize = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoOptimize = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorOptimize = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	logStatsOptimize = log.New(os.Stdout, "STATS: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	out := ioutil.Discard
	if verbose {
		out = os.Stdout
	}
	logInfoOptimize = log.New(out, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Mark all content streams for a page dictionary (for stats).
func identifyPageContent(xRefTable *types.XRefTable, pageDict *types.PDFDict, pageNumber, pageObjNumber int) (err error) {

	logDebugOptimize.Println("identifyPageContent begin")

	pdfObject, found := pageDict.Find("Contents")
	if !found {
		logDebugOptimize.Println("identifyPageContent end: no \"Contents\"")
		return nil
	}

	var contentArr types.PDFArray

	if indRef, ok := pdfObject.(types.PDFIndirectRef); ok {

		entry, found := xRefTable.FindTableEntry(indRef.ObjectNumber.Value(), indRef.GenerationNumber.Value())
		if !found {
			return errors.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.PDFStreamDict)
		if ok {
			contentStreamDict.IsPageContent = true
			entry.Object = contentStreamDict
			logDebugOptimize.Printf("identifyPageContent end: ok obj#%d\n", indRef.ObjectNumber.Value())
			return nil
		}

		contentArr, ok = entry.Object.(types.PDFArray)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d page content entry neither stream dict nor array.\n", pageObjNumber)
		}

	} else if contentArr, ok = pdfObject.(types.PDFArray); !ok {
		return errors.Errorf("identifyPageContent: obj#:%d corrupt page content array\n", pageObjNumber)
	}

	for _, c := range contentArr {

		indRef, ok := c.(types.PDFIndirectRef)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d corrupt page content array entry\n", pageObjNumber)
		}

		entry, found := xRefTable.FindTableEntry(indRef.ObjectNumber.Value(), indRef.GenerationNumber.Value())
		if !found {
			return errors.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.PDFStreamDict)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d page content entry is no stream dict\n", pageObjNumber)
		}

		contentStreamDict.IsPageContent = true
		entry.Object = contentStreamDict
		logDebugOptimize.Printf("identifyPageContent: ok obj#%d\n", indRef.GenerationNumber.Value())
	}

	logDebugOptimize.Println("identifyPageContent end")

	return
}

// ResourcesDictForPageDict returns the resource dict for a page dict if there is any.
func ResourcesDictForPageDict(xRefTable *types.XRefTable, pageDict *types.PDFDict, pageObjNumber int) (dict *types.PDFDict, err error) {

	obj, found := pageDict.Find("Resources")
	if !found {
		logInfoOptimize.Printf("ResourcesDictForPageDict end: No resources dict for page object %d, may be inheritated\n", pageObjNumber)
		return
	}

	return xRefTable.DereferenceDict(obj)
}

func handleDuplicateFontObject(font *types.PDFDict, indRef *types.PDFIndirectRef, fontName, resourceName string, pageNumber int, ctx *types.PDFContext) (bool, error) {

	fontObjectNumbers, found := ctx.Optimize.Fonts[fontName]
	if !found {
		return false, nil
	}

	objectNumber := int(indRef.ObjectNumber)
	pageFonts := ctx.Optimize.PageFonts[pageNumber]

	for _, fontObjectNumber := range *fontObjectNumbers {

		fontObject := ctx.Optimize.FontObjects[fontObjectNumber]

		logDebugOptimize.Printf("optimizeFontResourcesDict: comparing with fontDict Obj %d\n", fontObjectNumber)

		ok, err := equalFontDicts(fontObject.FontDict, font, ctx)
		if err != nil {
			return false, err
		}

		if ok {

			// We have detected a redundant font dict.
			logInfoOptimize.Printf("optimizeFontResourcesDict: redundant fontObj#:%d basefont %s already registered with obj#:%d !\n", objectNumber, fontName, fontObjectNumber)
			// This is an optimization patch of the fontobject for a fontResource

			// Modify the indirect object reference to point to the single instance of this fontDict.
			indRef.ObjectNumber = types.PDFInteger(fontObjectNumber)

			// Update resourceDict entry with patched indRef.
			font.Dict[resourceName] = *indRef

			pageFonts[fontObjectNumber] = true

			fontObject.AddResourceName(resourceName)

			ctx.Optimize.DuplicateFonts[objectNumber] = font

			return true, nil

		}
	}

	return false, nil
}

// Get rid of redundant fonts for given fontResources dictionary.
func optimizeFontResourcesDict(ctx *types.PDFContext, fontResourcesDict *types.PDFDict, pageNumber, pageObjNumber int) (err error) {

	logDebugOptimize.Printf("optimizeFontResourcesDict begin: page=%d pageObjNumber=%d %s\nPageFonts=%v\n", pageNumber, pageObjNumber, *fontResourcesDict, ctx.Optimize.PageFonts)

	pageFonts := ctx.Optimize.PageFonts[pageNumber]
	if pageFonts == nil {
		pageFonts = types.IntSet{}
		ctx.Optimize.PageFonts[pageNumber] = pageFonts
	}

	for resourceName, v := range fontResourcesDict.Dict {

		indRef, ok := v.(types.PDFIndirectRef)
		if !ok {
			return errors.Errorf("optimizeFontResourcesDict: missing indirect object ref for Font: %s\n", resourceName)
		}

		logDebugOptimize.Printf("optimizeFontResourcesDict: processing font: %s, %s\n", resourceName, indRef)
		objectNumber := int(indRef.ObjectNumber)
		logDebugOptimize.Printf("optimizeFontResourcesDict: objectNumber = %d\n", objectNumber)

		if _, found := ctx.Optimize.FontObjects[objectNumber]; found {
			//logInfoOptimizePrintf("optimizeFontResourcesDict: Fontobject %d already registered\n", objectNumber)
			pageFonts[objectNumber] = true
			continue
		}

		pdfObject, err := ctx.Dereference(indRef)
		if err != nil {
			return errors.Errorf("optimizeFontResourcesDict: missing obj for indirect object ref %d:\n%s", objectNumber, err)
		}

		fontDict := pdfObject.(types.PDFDict)
		logDebugOptimize.Printf("optimizeFontResourcesDict: fontDict: %s\n", fontDict)

		if fontDict.Type() == nil {
			return errors.Errorf("optimizeFontResourcesDict: missing dict type %s\n", v)
		}

		if *fontDict.Type() != "Font" {
			return errors.Errorf("optimizeFontResourcesDict: expected Type=Font, unexpected Type: %s", *fontDict.Type())
		}

		// Process font dict
		baseFont, found := fontDict.Find("BaseFont")
		if !found {
			baseFont, found = fontDict.Find("Name")
			if !found {
				return errors.New("optimizeFontResourcesDict: missing fontDict entries \"BaseFont\" and \"Name\"")
			}
		}

		baseFont, _ = ctx.Dereference(baseFont)
		baseF, ok := baseFont.(types.PDFName)
		if !ok {
			return errors.New("optimizeFontResourcesDict: corrupt fontDict entry BaseFont")
		}

		fontName := string(baseF)
		logDebugOptimize.Printf("optimizeFontResourcesDict: baseFont: %s\n", fontName)

		// Isolate fontname prefix
		var prefix string
		i := strings.Index(fontName, "+")

		if i > 0 {
			prefix = fontName[:i]
			fontName = fontName[i+1:]
		}

		duplicate, err := handleDuplicateFontObject(&fontDict, &indRef, fontName, resourceName, pageNumber, ctx)
		if err != nil {
			return err
		}

		if !duplicate {

			// add fontInfo entry into Fonts
			// add fontobject entry into fontObjects
			logDebugOptimize.Printf("optimizeFontResourcesDict: adding new font %s obj#%d\n", fontName, objectNumber)

			fontObjectNumbers, found := ctx.Optimize.Fonts[fontName]
			if found {
				logDebugOptimize.Printf("optimizeFontResourcesDict: appending %d to %s\n", objectNumber, fontName)
				*fontObjectNumbers = append(*fontObjectNumbers, objectNumber)
			} else {
				ctx.Optimize.Fonts[fontName] = &[]int{objectNumber}
			}

			ctx.Optimize.FontObjects[objectNumber] =
				&types.FontObject{
					ResourceNames: []string{resourceName},
					Prefix:        prefix,
					FontName:      fontName,
					FontDict:      &fontDict}

			pageFonts[objectNumber] = true

		}
	}

	logDebugOptimize.Println("optimizeFontResourcesDict end:")

	return
}

func handleDuplicateImageObject(image *types.PDFStreamDict, indRef *types.PDFIndirectRef, resourceName string, pageNumber int, ctx *types.PDFContext) (bool, error) {

	objectNumber := int(indRef.ObjectNumber)
	pageImages := ctx.Optimize.PageImages[pageNumber]

	// Process image dict, check if this is a duplicate.
	for imageObjectNumber, imageObject := range ctx.Optimize.ImageObjects {

		logDebugOptimize.Printf("handleDuplicateImageObject: comparing %d with imagedict Obj %d\n", objectNumber, imageObjectNumber)

		ok, err := equalPDFStreamDicts(image, imageObject.ImageDict, ctx)
		if err != nil {
			return false, err
		}

		if ok {

			// We have detected a redundant image dict.

			logInfoOptimize.Printf("handleDuplicateImageObject: redundant imageObj#:%d already registered with obj#:%d !\n", objectNumber, imageObjectNumber)

			// This is an optimization patch of the imageobject for an XObject Resource:

			// Modify the indirect object reference to point to the single instance of this imageDict.
			indRef.ObjectNumber = types.PDFInteger(imageObjectNumber)

			// Save patched indRef to resourceDict.
			image.Dict[resourceName] = *indRef

			pageImages[imageObjectNumber] = true

			imageObject.AddResourceName(resourceName)

			ctx.Optimize.DuplicateImages[objectNumber] = image

			logDebugOptimize.Printf("handleDuplicateImageObject: increment binary image duplsize for obj:%d: %d bytes\n", objectNumber, *image.StreamLength)

			return true, nil

		}
	}

	return false, nil
}

// Get rid of redundant XObjects e.g. embedded images.
func optimizeXObjectResourcesDict(ctx *types.PDFContext, xObjectResourcesDict *types.PDFDict, pageNumber, pageObjNumber int) (err error) {

	logDebugOptimize.Printf("optimizeXObjectResourcesDict begin: %s\n", *xObjectResourcesDict)

	pageImages := ctx.Optimize.PageImages[pageNumber]
	if pageImages == nil {
		pageImages = types.IntSet{}
		ctx.Optimize.PageImages[pageNumber] = pageImages
	}

	for resourceName, v := range xObjectResourcesDict.Dict {

		indRef, ok := v.(types.PDFIndirectRef)
		if !ok {
			return errors.Errorf("optimizeXObjectResourcesDict: missing indirect object ref for resourceId: %s", resourceName)
		}

		logDebugOptimize.Printf("optimizeXObjectResourcesDict: processing xobject: %s, %s\n", resourceName, indRef)
		objectNumber := int(indRef.ObjectNumber)
		logDebugOptimize.Printf("optimizeXObjectResourcesDict: objectNumber = %d\n", objectNumber)

		pdfObject, err := ctx.Dereference(indRef)
		if err != nil {
			return errors.Errorf("optimizeXObjectResourcesDict: missing obj for indirect object ref %d:\n%s", objectNumber, err)
		}

		logDebugOptimize.Printf("optimizeXObjectResourcesDict: dereferenced obj:%d\n%s", objectNumber, pdfObject)

		xObjectStreamDict, ok := pdfObject.(types.PDFStreamDict)
		if !ok {
			return errors.Errorf("optimizeXObjectResourcesDict: unexpected pdfObject: %s\n", v)
		}

		if xObjectStreamDict.PDFDict.Subtype() == nil {
			return errors.Errorf("optimizeXObjectResourcesDict: missing stream dict Subtype %s\n", v)
		}

		if *xObjectStreamDict.PDFDict.Subtype() == "Image" {

			// Already registered image object that appears in different resources dicts.
			if _, found := ctx.Optimize.ImageObjects[objectNumber]; found {
				logDebugOptimize.Printf("optimizeXObjectResourcesDict: Imageobject %d already registered\n", objectNumber)
				pageImages[objectNumber] = true
				continue
			}

			duplicate, err := handleDuplicateImageObject(&xObjectStreamDict, &indRef, resourceName, pageNumber, ctx)
			if err != nil {
				return err
			}

			if !duplicate {
				// Register new image dict.
				logDebugOptimize.Printf("optimizeXObjectResourcesDict: adding new image obj#%d\n", objectNumber)

				ctx.Optimize.ImageObjects[objectNumber] = &types.ImageObject{ResourceNames: []string{resourceName}, ImageDict: &xObjectStreamDict}
				pageImages[objectNumber] = true

				logDebugOptimize.Printf("optimizeXObjectResourcesDict: increment binary image size for obj:%d: %d bytes\n", objectNumber, *xObjectStreamDict.StreamLength)
			}

			continue
		}

		if *xObjectStreamDict.Subtype() != "Form" {
			logDebugOptimize.Printf("optimizeXObjectResourcesDict: unexpected stream dict Subtype %s\n", *xObjectStreamDict.PDFDict.Subtype())
			continue
		}

		// Process form dict
		logDebugOptimize.Printf("optimizeXObjectResourcesDict: parsing form dict obj:%d\n", objectNumber)
		parseResourcesDict(ctx, &xObjectStreamDict.PDFDict, pageNumber, objectNumber)
	}

	logDebugOptimize.Println("optimizeXObjectResourcesDict end")

	return
}

// Optimize given resource dictionary by removing redundant fonts and images.
func optimizeResources(ctx *types.PDFContext, resourcesDict *types.PDFDict, pageNumber, pageObjNumber int) (err error) {

	logDebugOptimize.Printf("optimizeResources begin: pageNumber=%d pageObjNumber=%d\n", pageNumber, pageObjNumber)

	if resourcesDict == nil {
		logInfoOptimize.Printf("optimizeResources end: No resources dict available")
		return nil
	}

	var dict *types.PDFDict

	// Process Font resource dict, get rid of redundant fonts.
	obj, found := resourcesDict.Find("Font")
	if found {

		dict, err = ctx.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if dict == nil {
			return errors.Errorf("optimizeResources: font resource dict is null for page %d pageObj %d\n", pageNumber, pageObjNumber)
		}

		err = optimizeFontResourcesDict(ctx, dict, pageNumber, pageObjNumber)
		if err != nil {
			return
		}

	}

	// Note: An optional ExtGState resource dict may contain binary content in the following entries: "SMask", "HT".

	// Process XObject resource dict, get rid of redundant images.
	obj, found = resourcesDict.Find("XObject")
	if found {

		dict, err = ctx.DereferenceDict(obj)
		if err != nil {
			return
		}

		if dict == nil {
			return errors.Errorf("optimizeResources: xobject resource dict is null for page %d pageObj %d\n", pageNumber, pageObjNumber)
		}

		err = optimizeXObjectResourcesDict(ctx, dict, pageNumber, pageObjNumber)
		if err != nil {
			return
		}

	}

	logDebugOptimize.Println("optimizeResources end")

	return
}

// Process the resources dictionary for given page number and optimize by removing redundant resources.
func parseResourcesDict(ctx *types.PDFContext, pageDict *types.PDFDict, pageNumber, pageObjNumber int) (err error) {

	logDebugOptimize.Printf("parseResourcesDict begin page: %d, object:%d\n", pageNumber+1, pageObjNumber)

	// Get resources dict for this page.
	dict, err := ResourcesDictForPageDict(ctx.XRefTable, pageDict, pageObjNumber)
	if err != nil {
		return
	}

	// dict may be nil for inheritated resource dicts.
	if dict != nil {

		// Optimize image and font resources.
		err = optimizeResources(ctx, dict, pageNumber, pageObjNumber)
		if err != nil {
			return
		}

	}

	logDebugOptimize.Printf("parseResourcesDict end page: %d, object:%d\n", pageNumber+1, pageObjNumber)

	return
}

// Iterate over all pages and optimize resources.
func parsePagesDict(ctx *types.PDFContext, pagesDict *types.PDFDict, pageNumber int) (int, error) {

	logDebugOptimize.Printf("parsePagesDict begin (next page=%d): %s\n", pageNumber+1, *pagesDict)

	// Get number of pages of this PDF file.
	count, found := pagesDict.Find("Count")
	if !found {
		return 0, errors.New("parsePagesDict: missing Count")
	}

	logDebugOptimize.Printf("parsePagesDict: This page node has %d pages\n", int(count.(types.PDFInteger)))

	// Iterate over page tree.
	kidsArray := pagesDict.PDFArrayEntry("Kids")
	for _, v := range *kidsArray {

		// Dereference next page node dict.
		indRef, _ := v.(types.PDFIndirectRef)
		logDebugOptimize.Printf("parsePagesDict PageNode: %s\n", indRef)
		pdfObject, err := ctx.Dereference(indRef)
		if err != nil {
			return 0, errors.Wrap(err, "parsePagesDict: can't locate Pagedict or Pagesdict")
		}

		pageNodeDict := pdfObject.(types.PDFDict)
		dictType := pageNodeDict.Type()
		if dictType == nil {
			return 0, errors.New("parsePagesDict: Missing dict type")
		}

		// Note: Pages may contain a to be inheritated ResourcesDict.

		if *dictType == "Pages" {

			// Recurse over pagetree and optimize resources.
			pageNumber, err = parsePagesDict(ctx, &pageNodeDict, pageNumber)
			if err != nil {
				return 0, err
			}

			continue
		}

		if *dictType != "Page" {
			return 0, errors.Errorf("parsePagesDict: Unexpected dict type: %s\n", *dictType)
		}

		// Mark page content streams for stats.
		err = identifyPageContent(ctx.XRefTable, &pageNodeDict, pageNumber, int(indRef.ObjectNumber))
		if err != nil {
			return 0, err
		}

		// Parse and optimize resource dict for one page.
		err = parseResourcesDict(ctx, &pageNodeDict, pageNumber, int(indRef.ObjectNumber))
		if err != nil {
			return 0, err
		}

		pageNumber++
	}

	logDebugOptimize.Printf("parsePagesDict end: %s\n", *pagesDict)

	return pageNumber, nil
}

// Traverse the object graph for a pdfObject and mark all objects as potential duplicates.
func traverseObjectGraphAndMarkDuplicates(xRefTable *types.XRefTable, obj interface{}, duplObjs types.IntSet) (err error) {

	logDebugOptimize.Printf("traverseObjectGraphAndMarkDuplicates begin type=%T\n", obj)

	switch x := obj.(type) {

	case types.PDFDict:
		logDebugOptimize.Println("traverseObjectGraphAndMarkDuplicates: dict.")
		for _, value := range x.Dict {
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				duplObjs[int(indRef.ObjectNumber)] = true
				o, err := xRefTable.Dereference(indRef)
				if err != nil {
					return err
				}
				traverseObjectGraphAndMarkDuplicates(xRefTable, o, duplObjs)
			}
			if dict, ok := value.(types.PDFDict); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, dict, duplObjs)
			}
			if streamDict, ok := value.(types.PDFStreamDict); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, streamDict, duplObjs)
			}
			if arr, ok := value.(types.PDFArray); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, arr, duplObjs)
			}
		}

	case types.PDFStreamDict:
		logDebugOptimize.Println("traverseObjectGraphAndMarkDuplicates: streamDict.")
		for _, value := range x.Dict {
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				duplObjs[int(indRef.ObjectNumber)] = true
				o, err := xRefTable.Dereference(indRef)
				if err != nil {
					return err
				}
				traverseObjectGraphAndMarkDuplicates(xRefTable, o, duplObjs)
			}
			if dict, ok := value.(types.PDFDict); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, dict, duplObjs)
			}
			if streamDict, ok := value.(types.PDFStreamDict); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, streamDict, duplObjs)
			}
			if arr, ok := value.(types.PDFArray); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, arr, duplObjs)
			}
		}

	case types.PDFArray:
		logDebugOptimize.Println("traverseObjectGraphAndMarkDuplicates: arr.")
		for _, value := range x {
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				duplObjs[int(indRef.ObjectNumber)] = true
				o, err := xRefTable.Dereference(indRef)
				if err != nil {
					return err
				}
				traverseObjectGraphAndMarkDuplicates(xRefTable, o, duplObjs)
			}
			if dict, ok := value.(types.PDFDict); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, dict, duplObjs)
			}
			if streamDict, ok := value.(types.PDFStreamDict); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, streamDict, duplObjs)
			}
			if arr, ok := value.(types.PDFArray); ok {
				traverseObjectGraphAndMarkDuplicates(xRefTable, arr, duplObjs)
			}
		}
	}

	logDebugOptimize.Println("traverseObjectGraphAndMarkDuplicates end")

	return nil
}

// Identify and mark all potential duplicate objects.
func calcRedundantObjects(ctx *types.PDFContext) (err error) {

	logDebugOptimize.Println("calcRedundantObjects begin")

	for i, fontDict := range ctx.Optimize.DuplicateFonts {
		ctx.Optimize.DuplicateFontObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant font.
		err = traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, *fontDict, ctx.Optimize.DuplicateFontObjs)
		if err != nil {
			return
		}
	}

	for i, streamDict := range ctx.Optimize.DuplicateImages {
		ctx.Optimize.DuplicateImageObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant image.
		err = traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, *streamDict, ctx.Optimize.DuplicateImageObjs)
		if err != nil {
			return
		}
	}

	logDebugOptimize.Println("calcRedundantObjects end")

	return
}

// Iterate over all pages and optimize resources.
// Get rid of duplicate embedded fonts and images.
func optimizeFontAndImages(ctx *types.PDFContext) (err error) {

	logInfoOptimize.Println("optimizeFontAndImages begin")

	// Get a reference to the PDF indirect reference of the page tree root dict.
	indRefPages, err := ctx.Pages()
	if err != nil {
		return
	}

	// Dereference and get a reference to the page tree root dict.
	pageTreeRootDict, err := ctx.XRefTable.DereferenceDict(*indRefPages)
	if err != nil {
		return
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
	ctx.Optimize.PageFonts = make([]types.IntSet, ctx.PageCount)
	ctx.Optimize.PageImages = make([]types.IntSet, ctx.PageCount)

	// Iterate over page dicts and optimize resources.
	_, err = parsePagesDict(ctx, pageTreeRootDict, 0)
	if err != nil {
		return
	}

	// Identify all duplicate objects.
	err = calcRedundantObjects(ctx)
	if err != nil {
		return
	}

	logInfoOptimize.Println("optimizeFontAndImages end")

	return
}

// Return stream length for font file object.
func streamLengthFontFile(xRefTable *types.XRefTable, indirectRef *types.PDFIndirectRef) (*int64, error) {

	logDebugOptimize.Println("streamLengthFontFile begin")

	objectNumber := indirectRef.ObjectNumber

	streamDict, err := xRefTable.DereferenceStreamDict(*indirectRef)
	if err != nil {
		return nil, err
	}

	if streamDict == nil || (*streamDict).StreamLength == nil {
		return nil, errors.Errorf("streamLengthFontFile: fontFile Streamlength is nil for object %d\n", objectNumber)
	}

	logDebugOptimize.Println("streamLengthFontFile end")

	return (*streamDict).StreamLength, nil
}

// Calculate amount of memory used by embedded fonts for stats.
func calcEmbeddedFontsMemoryUsage(ctx *types.PDFContext) error {

	logDebugOptimize.Printf("calcEmbeddedFontsMemoryUsage begin: %d fontObjects\n", len(ctx.Optimize.FontObjects))

	fontFileIndRefs := map[types.PDFIndirectRef]bool{}

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
	for indRef := range fontFileIndRefs {
		streamLength, err := streamLengthFontFile(ctx.XRefTable, &indRef)
		if err != nil {
			return err
		}
		ctx.Read.BinaryFontSize += *streamLength
	}

	logDebugOptimize.Println("calcEmbeddedFontsMemoryUsage end")

	return nil
}

// FontDescriptorFontFileIndirectObjectRef returns the indirect object for the font file for given font descriptor.
func FontDescriptorFontFileIndirectObjectRef(fontDescriptorDict *types.PDFDict) *types.PDFIndirectRef {

	logDebugOptimize.Println("FontDescriptorFontFileIndirectObjectRef begin")

	indirectRef := fontDescriptorDict.IndirectRefEntry("FontFile")

	if indirectRef == nil {
		indirectRef = fontDescriptorDict.IndirectRefEntry("FontFile2")
	}

	if indirectRef == nil {
		indirectRef = fontDescriptorDict.IndirectRefEntry("FontFile3")
	}

	if indirectRef == nil {
		//logInfoReader.Printf("FontDescriptorFontFileLength: FontDescriptor dict without fontFile: \n%s\n", fontDescriptorDict)
	}

	logDebugOptimize.Println("FontDescriptorFontFileIndirectObjectRef end")

	return indirectRef
}

// FontDescriptor gets the font descriptor for this font.
func FontDescriptor(xRefTable *types.XRefTable, fontDict *types.PDFDict, objectNumber int) (*types.PDFDict, error) {

	logDebugOptimize.Println("FontDescriptor begin")

	obj, ok := fontDict.Find("FontDescriptor")
	if ok {

		// fontDescriptor directly available.

		dict, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return nil, err
		}

		if dict == nil {
			return nil, errors.Errorf("FontDescriptor: FontDescriptor is null for font object %d\n", objectNumber)
		}

		if dict.Type() != nil && *dict.Type() != "FontDescriptor" {
			return nil, errors.Errorf("FontDescriptor: FontDescriptor dict incorrect dict type for font object %d\n", objectNumber)
		}

		return dict, nil
	}

	// Try to access a fontDescriptor in a Descendent font for Type0 fonts.

	obj, ok = fontDict.Find("DescendantFonts")
	if !ok {
		//logErrorOptimize.Printf("FontDescriptor: Neither FontDescriptor nor DescendantFonts for font object %d\n", objectNumber)
		return nil, nil
	}

	// A descendant font is contained in an array of size 1.

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil || arr == nil {
		return nil, errors.Errorf("FontDescriptor: DescendantFonts: IndirectRef or Array wth length 1 expected for font object %d\n", objectNumber)
	}

	if len(*arr) > 1 {
		return nil, errors.Errorf("FontDescriptor: DescendantFonts Array length > 1 %v\n", arr)
	}

	// dict is the fontDict of the descendant font.
	dict, err := xRefTable.DereferenceDict((*arr)[0])
	if err != nil {
		return nil, errors.Errorf("FontDescriptor: No descendant font dict for %v\n", arr)
	}

	if dict == nil {
		return nil, errors.Errorf("FontDescriptor: descendant font dict is null for %v\n", arr)
	}

	if *dict.Type() != "Font" {
		return nil, errors.Errorf("FontDescriptor: font dict with incorrect dict type for %v\n", dict)
	}

	obj, ok = (*dict).Find("FontDescriptor")
	if !ok {
		logInfoOptimize.Printf("FontDescriptor: descendant font not embedded %s\n", dict)
		return nil, nil
	}

	dict, err = xRefTable.DereferenceDict(obj)
	if err != nil {
		return nil, errors.Errorf("FontDescriptor: No FontDescriptor dict for font object %d\n", objectNumber)
	}

	if dict == nil {
		return nil, errors.Errorf("FontDescriptor: FontDescriptor dict is null for font object %d\n", objectNumber)
	}

	if dict.Type() == nil {
		logErrorOptimize.Printf("FontDescriptor: FontDescriptor without type \"FontDescriptor\" objNumber:%d\n", objectNumber)
	} else if *dict.Type() != "FontDescriptor" {
		return nil, errors.Errorf("FontDescriptor: FontDescriptor dict incorrect dict type for font object %d\n", objectNumber)
	}

	logDebugOptimize.Println("FontDescriptor end")

	return dict, nil
}

// Record font file objects referenced by this fonts font descriptor for stats and size calculation.
func processFontFilesForFontDict(xRefTable *types.XRefTable, fontDict *types.PDFDict, objectNumber int, indRefsMap map[types.PDFIndirectRef]bool) error {

	logDebugOptimize.Println("processFontFilesForFontDict begin")

	// Note:
	// "ToUnicode" is also an entry containing binary content that could be inspected for duplicate content.

	dict, err := FontDescriptor(xRefTable, fontDict, objectNumber)
	if err != nil {
		return err
	}

	if dict != nil {
		if indRef := FontDescriptorFontFileIndirectObjectRef(dict); indRef != nil {
			indRefsMap[*indRef] = true
		}
	}

	logDebugOptimize.Println("processFontFilesForFontDict end")

	return nil
}

// Calculate amount of memory used by duplicate embedded fonts for stats.
func calcRedundantEmbeddedFontsMemoryUsage(ctx *types.PDFContext) error {

	logDebugOptimize.Println("processFontFilesForFontDict begin")

	fontFileIndRefs := map[types.PDFIndirectRef]bool{}

	// Iterate over all duplicate fonts and record font file references.
	for objectNumber, fontDict := range ctx.Optimize.DuplicateFonts {

		// Duplicate Fonts have to be embedded, so no check here.
		if err := processFontFilesForFontDict(ctx.XRefTable, fontDict, objectNumber, fontFileIndRefs); err != nil {
			return err
		}

	}

	// Iterate over font file references and calculate total font size.
	for indRef := range fontFileIndRefs {

		streamLength, err := streamLengthFontFile(ctx.XRefTable, &indRef)
		if err != nil {
			return err
		}

		ctx.Read.BinaryFontDuplSize += *streamLength
	}

	logDebugOptimize.Println("processFontFilesForFontDict end")

	return nil
}

// Calculate amount of memory used by embedded fonts and duplicate embedded fonts for stats.
func calcFontBinarySizes(ctx *types.PDFContext) error {

	logDebugOptimize.Println("calcFontBinarySizes begin")

	err := calcEmbeddedFontsMemoryUsage(ctx)
	if err != nil {
		return err
	}

	err = calcRedundantEmbeddedFontsMemoryUsage(ctx)
	if err != nil {
		return err
	}

	logDebugOptimize.Println("calcFontBinarySizes end")

	return nil
}

// Calculate amount of memory used by images and duplicate images for stats.
func calcImageBinarySizes(ctx *types.PDFContext) {

	logDebugOptimize.Println("calcImageBinarySizes begin")

	// Calc memory usage for images.
	for _, imageObject := range ctx.Optimize.ImageObjects {
		ctx.Read.BinaryImageSize += *imageObject.ImageDict.StreamLength
	}

	// Calc memory usage for duplicate images.
	for _, imageDict := range ctx.Optimize.DuplicateImages {
		ctx.Read.BinaryImageDuplSize += *imageDict.StreamLength
	}

	logDebugOptimize.Println("calcImageBinarySizes end")
}

// Calculate memory usage of binary data for stats.
func calcBinarySizes(ctx *types.PDFContext) (err error) {

	logInfoOptimize.Println("calcBinarySizes begin")

	// Calculate font memory usage for stats.
	err = calcFontBinarySizes(ctx)
	if err != nil {
		return
	}

	// Calculate image memory usage for stats.
	calcImageBinarySizes(ctx)

	// Note: Content streams also represent binary content.

	logInfoOptimize.Println("calcBinarySizes end")

	return
}

// XRefTable optimizes an xRefTable by locating and getting rid of redundant embedded fonts and images.
func XRefTable(ctx *types.PDFContext) (err error) {

	logInfoOptimize.Println("XRefTable begin")

	// Get rid of duplicate embedded fonts and images.
	err = optimizeFontAndImages(ctx)
	if err != nil {
		return
	}

	// Calculate memory usage of binary content for stats.
	err = calcBinarySizes(ctx)
	if err != nil {
		return
	}

	ctx.Optimized = true

	logInfoOptimize.Println("XRefTable end")

	return
}

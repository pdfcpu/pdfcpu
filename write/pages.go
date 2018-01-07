package write

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// Write page entry to disk.
func writePageEntry(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) (err error) {

	var obj interface{}

	obj, err = writeEntry(ctx, dict, dictName, entryName)
	if err != nil {
		return
	}

	if obj != nil {
		ctx.Stats.AddPageAttr(statsAttr)
	}

	return
}

func writePageDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageDict *types.PDFDict) (err error) {

	objNumber := indRef.ObjectNumber.Value()
	genNumber := indRef.GenerationNumber.Value()

	logInfoWriter.Printf("writePageDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	dictName := "pageDict"

	// For extracted pages we do not generate Annotations.
	if ctx.Write.ReducedFeatureSet() {
		pageDict.Delete("Annots")
	}

	err = writePDFDictObject(ctx, objNumber, genNumber, *pageDict)
	if err != nil {
		return
	}

	logDebugWriter.Printf("writePageDict: new offset = %d\n", ctx.Write.Offset)

	if indref := pageDict.IndirectRefEntry("Parent"); indref == nil {
		return errors.New("writePageDict: missing parent")
	}

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Contents", types.PageContents},
		{"Resources", types.PageResources},
		{"MediaBox", types.PageMediaBox},
		{"CropBox", types.PageCropBox},
		{"BleedBox", types.PageBleedBox},
		{"TrimBox", types.PageTrimBox},
		{"ArtBox", types.PageArtBox},
		{"BoxColorInfo", types.PageBoxColorInfo},
		{"PieceInfo", types.PagePieceInfo},
		{"LastModified", types.PageLastModified},
		{"Rotate", types.PageRotate},
		{"Group", types.PageGroup},
		{"Annots", types.PageAnnots},
		{"Thumb", types.PageThumb},
		{"B", types.PageB},
		{"Dur", types.PageDur},
		{"Trans", types.PageTrans},
		{"AA", types.PageAA},
		{"Metadata", types.PageMetadata},
		{"StructParents", types.PageStructParents},
		{"ID", types.PageID},
		{"PZ", types.PagePZ},
		{"SeparationInfo", types.PageSeparationInfo},
		{"Tabs", types.PageTabs},
		{"TemplateInstantiated", types.PageTemplateInstantiated},
		{"PresSteps", types.PagePresSteps},
		{"UserUnit", types.PageUserUnit},
		{"VP", types.PageVP},
	} {
		err = writePageEntry(ctx, pageDict, dictName, e.entryName, e.statsAttr)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writePageDict end: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	return
}

func locateKidForPageNumber(ctx *types.PDFContext, kidsArray *types.PDFArray, pageCount *int, pageNumber int) (kid interface{}, err error) {

	for _, obj := range *kidsArray {

		if obj == nil {
			logDebugWriter.Println("locateKidForPageNumber: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return nil, errors.New("locateKidForPageNumber: missing indirect reference for kid")
		}

		logInfoWriter.Printf("locateKidForPageNumber: PageNode: %s pageCount:%d extractPageNr:%d\n", indRef, *pageCount, pageNumber)

		dict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return nil, errors.New("locateKidForPageNumber: cannot dereference pageNodeDict")
		}

		if dict == nil {
			return nil, errors.New("locateKidForPageNumber: pageNodeDict is null")
		}

		dictType := dict.Type()
		if dictType == nil {
			return nil, errors.New("locateKidForPageNumber: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Get number of pages of this PDF file.
			count, ok := dict.Find("Count")
			if !ok {
				return nil, errors.New("locateKidForPageNumber: missing \"Count\"")
			}
			pCount := int(count.(types.PDFInteger))

			if *pageCount+pCount < ctx.Write.ExtractPageNr {
				*pageCount += pCount
				logInfoWriter.Printf("locateKidForPageNumber: pageTree is no match: %d\n", ctx.Write.ExtractPageNr)
			} else {
				logInfoWriter.Printf("locateKidForPageNumber: pageTree is a match: %d\n", ctx.Write.ExtractPageNr)
				return obj, nil
			}

		case "Page":
			*pageCount++
			if *pageCount == ctx.Write.ExtractPageNr {
				logInfoWriter.Printf("locateKidForPageNumber: page is a match")
				return obj, nil
			}

			logInfoWriter.Printf("locateKidForPageNumber: page is no match")

		default:
			return nil, errors.Errorf("locateKidForPageNumber: Unexpected dict type: %s", *dictType)
		}

	}

	return nil, errors.Errorf("locateKidForPageNumber: Unable to locate kid: pageCount:%d extractPageNr:%d\n", *pageCount, pageNumber)
}

func pageNodeDict(ctx *types.PDFContext, o interface{}) (d *types.PDFDict, indRef *types.PDFIndirectRef, err error) {

	if o == nil {
		logDebugWriter.Println("pageNodeDict: is nil")
		return
	}

	// Dereference next page node dict.
	iRef, ok := o.(types.PDFIndirectRef)
	if !ok {
		err = errors.New("pageNodeDict: missing indirect reference")
		return
	}
	logInfoWriter.Printf("pageNodeDict: PageNode: %s\n", iRef)

	objNumber := int(iRef.ObjectNumber)
	//genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logInfoWriter.Printf("pageNodeDict: object #%d already written.\n", objNumber)
		return
	}

	d, err = ctx.DereferenceDict(iRef)
	if err != nil {
		err = errors.New("pageNodeDict: cannot dereference, pageNodeDict")
		return
	}
	if d == nil {
		err = errors.New("pageNodeDict: pageNodeDict is null")
		return
	}

	dictType := d.Type()
	if dictType == nil {
		err = errors.New("pageNodeDict: missing pageNodeDict type")
		return
	}

	indRef = &iRef

	return
}

func prepareSinglePageWrite(ctx *types.PDFContext, dict *types.PDFDict, kids *types.PDFArray, pageCount *int) error {

	kid, err := locateKidForPageNumber(ctx, kids, pageCount, ctx.Write.ExtractPageNr)
	if err != nil {
		return err
	}

	if *pageCount == ctx.Write.ExtractPageNr {
		// The identified kid is the page node for the page we are looking for.
		logInfoWriter.Printf("prepareSinglePageWrite: found page to be extracted, pageCount=%d, extractPageNr=%d\n", *pageCount, ctx.Write.ExtractPageNr)
	} else {
		// The identified kid is the page tree containing the page we are looking for.
		logInfoWriter.Printf("prepareSinglePageWrite: pageCount=%d, extractPageNr=%d\n", *pageCount, ctx.Write.ExtractPageNr)
	}

	// Modify KidsArray to hold a single entry for this kid
	dict.Update("Kids", types.PDFArray{kid})

	// Set Count =1
	dict.Update("Count", types.PDFInteger(1))

	return nil
}

func writeKids(ctx *types.PDFContext, arr *types.PDFArray, pageCount int) error {

	for _, obj := range *arr {

		d, indRef, err := pageNodeDict(ctx, obj)
		if err != nil {
			return err
		}
		if d == nil {
			continue
		}

		switch *d.Type() {

		case "Pages":
			// Recurse over pagetree
			err = writePagesDict(ctx, indRef, pageCount)

		case "Page":
			err = writePageDict(ctx, indRef, d)

		default:
			err = errors.Errorf("writeKids: Unexpected dict type: %s", *d.Type())

		}

		if err != nil {
			return err
		}

	}

	return nil
}

func writePagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount int) (err error) {

	logPages.Printf("*** writePagesDict begin: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		err = errors.Wrapf(err, "writePagesDict: unable to dereference indirect object #%d", objNumber)
		return
	}

	if dict == nil {
		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dictName := "pagesDict"

	// Get number of pages of this PDF file.
	count, ok := dict.Find("Count")
	if !ok {
		return errors.New("writePagesDict: missing \"Count\"")
	}

	c := int(count.(types.PDFInteger))
	logPages.Printf("writePagesDict: This page node has %d pages\n", c)

	if c == 0 {
		logPages.Printf("writePagesDict: Ignore empty pages dict.\n")
		return
	}

	kidsArrayOrig := dict.PDFArrayEntry("Kids")
	if kidsArrayOrig == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}

	// This is for Split and Extract when all we generate is a single page.
	if ctx.Write.ExtractPageNr > 0 {
		// Identify the kid containing the leaf for the page we are looking for aka the ExtractPageNr.
		// pageCount is either already the number of the page we are looking for and we have identified the kid for its page dict
		// or the number of pages before processing the next page tree containing the page we are looking for.
		// We need to write all original pagetree nodes leading to a specific leave in order not to miss any inheritated resources.
		logInfoWriter.Printf("kidsArrayOrig before: %v", kidsArrayOrig)
		err = prepareSinglePageWrite(ctx, dict, kidsArrayOrig, &pageCount)
		if err != nil {
			return
		}
		logInfoWriter.Printf("kidsArrayOrig after: %v", kidsArrayOrig)
	}

	err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("writePagesDict: %s\n", dict)

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Resources", types.PageResources},
		{"MediaBox", types.PageMediaBox},
		{"CropBox", types.PageCropBox},
		{"Rotate", types.PageRotate},
	} {
		err = writePageEntry(ctx, dict, dictName, e.entryName, e.statsAttr)
		if err != nil {
			return
		}
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}
	logInfoWriter.Printf("kidsArray: %v", kidsArray)

	err = writeKids(ctx, kidsArray, pageCount)
	if err != nil {
		return
	}

	dict.Update("Kids", *kidsArrayOrig)
	dict.Update("Count", count)

	logPages.Printf("*** writePagesDict end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

func trimPagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount *int) (count int, err error) {

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)

	obj, err := xRefTable.Dereference(*indRef)
	if err != nil {
		err = errors.Wrapf(err, "trimPagesDict: unable to dereference indirect object #%d", objNumber)
		return
	}

	if obj == nil {
		return 0, errors.Errorf("trimPagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return 0, errors.Errorf("trimPagesDict: corrupt pages dict, obj#%d", objNumber)
	}

	// Get number of pages of this PDF file.
	c, ok := dict.Find("Count")
	if !ok {
		return 0, errors.New("trimPagesDict: missing \"Count\"")
	}

	logPages.Printf("trimPagesDict: This page node has %d pages\n", int(c.(types.PDFInteger)))

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return 0, errors.New("trimPagesDict: corrupt \"Kids\" entry")
	}

	arr := types.PDFArray{}

	for _, obj := range *kidsArray {

		d, indRef, err := pageNodeDict(ctx, obj)
		if err != nil {
			return 0, err
		}
		if d == nil {
			continue
		}

		switch *d.Type() {

		case "Pages":
			// Recurse over pagetree
			trimmedCount, err := trimPagesDict(ctx, indRef, pageCount)
			if err != nil {
				return 0, err
			}

			if trimmedCount > 0 {
				count += trimmedCount
				arr = append(arr, obj)
			}

		case "Page":
			*pageCount++
			if ctx.Write.ExtractPage(*pageCount) {
				count++
				arr = append(arr, obj)
			}

		default:
			return 0, errors.Errorf("trimPagesDict: Unexpected dict type: %s", *d.Type())

		}

	}

	logPages.Printf("trimPagesDict end: This page node is trimmed to %d pages\n", count)
	dict.Update("Count", types.PDFInteger(count))

	logPages.Printf("trimPagesDict end: updated kids: %s\n", arr)
	dict.Update("Kids", arr)

	return
}

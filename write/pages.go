package write

import (
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// Write page entry to disk.
func writePageEntry(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) error {

	obj, err := writeEntry(ctx, dict, dictName, entryName)
	if err != nil {
		return err
	}

	if obj != nil {
		ctx.Stats.AddPageAttr(statsAttr)
	}

	return nil
}

func writePageDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageDict *types.PDFDict) error {

	objNumber := indRef.ObjectNumber.Value()
	genNumber := indRef.GenerationNumber.Value()

	log.Debug.Printf("writePageDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	dictName := "pageDict"

	// For extracted pages we do not generate Annotations.
	if ctx.Write.ReducedFeatureSet() {
		pageDict.Delete("Annots")
	}

	err := writePDFDictObject(ctx, objNumber, genNumber, *pageDict)
	if err != nil {
		return err
	}

	log.Debug.Printf("writePageDict: new offset = %d\n", ctx.Write.Offset)

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
			return err
		}
	}

	log.Debug.Printf("*** writePageDict end: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	return nil
}

func locateKidForPageNumber(ctx *types.PDFContext, kidsArray *types.PDFArray, pageCount *int, pageNumber int) (kid types.PDFObject, err error) {

	for _, obj := range *kidsArray {

		if obj == nil {
			log.Debug.Println("locateKidForPageNumber: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return nil, errors.New("locateKidForPageNumber: missing indirect reference for kid")
		}

		log.Debug.Printf("locateKidForPageNumber: PageNode: %s pageCount:%d extractPageNr:%d\n", indRef, *pageCount, pageNumber)

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
				log.Debug.Printf("locateKidForPageNumber: pageTree is no match: %d\n", ctx.Write.ExtractPageNr)
			} else {
				log.Debug.Printf("locateKidForPageNumber: pageTree is a match: %d\n", ctx.Write.ExtractPageNr)
				return obj, nil
			}

		case "Page":
			*pageCount++
			if *pageCount == ctx.Write.ExtractPageNr {
				log.Debug.Printf("locateKidForPageNumber: page is a match")
				return obj, nil
			}

			log.Debug.Printf("locateKidForPageNumber: page is no match")

		default:
			return nil, errors.Errorf("locateKidForPageNumber: Unexpected dict type: %s", *dictType)
		}

	}

	return nil, errors.Errorf("locateKidForPageNumber: Unable to locate kid: pageCount:%d extractPageNr:%d\n", *pageCount, pageNumber)
}

func pageNodeDict(ctx *types.PDFContext, o types.PDFObject) (d *types.PDFDict, indRef *types.PDFIndirectRef, err error) {

	if o == nil {
		log.Debug.Println("pageNodeDict: is nil")
		return nil, nil, nil
	}

	// Dereference next page node dict.
	iRef, ok := o.(types.PDFIndirectRef)
	if !ok {
		return nil, nil, errors.New("pageNodeDict: missing indirect reference")
	}
	log.Debug.Printf("pageNodeDict: PageNode: %s\n", iRef)

	objNumber := int(iRef.ObjectNumber)
	//genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		log.Debug.Printf("pageNodeDict: object #%d already written.\n", objNumber)
		return nil, nil, nil
	}

	d, err = ctx.DereferenceDict(iRef)
	if err != nil {
		return nil, nil, errors.New("pageNodeDict: cannot dereference, pageNodeDict")
	}
	if d == nil {
		return nil, nil, errors.New("pageNodeDict: pageNodeDict is null")
	}

	dictType := d.Type()
	if dictType == nil {
		return nil, nil, errors.New("pageNodeDict: missing pageNodeDict type")
	}

	return d, &iRef, nil
}

func prepareSinglePageWrite(ctx *types.PDFContext, dict *types.PDFDict, kids *types.PDFArray, pageCount *int) error {

	kid, err := locateKidForPageNumber(ctx, kids, pageCount, ctx.Write.ExtractPageNr)
	if err != nil {
		return err
	}

	if *pageCount == ctx.Write.ExtractPageNr {
		// The identified kid is the page node for the page we are looking for.
		log.Debug.Printf("prepareSinglePageWrite: found page to be extracted, pageCount=%d, extractPageNr=%d\n", *pageCount, ctx.Write.ExtractPageNr)
	} else {
		// The identified kid is the page tree containing the page we are looking for.
		log.Debug.Printf("prepareSinglePageWrite: pageCount=%d, extractPageNr=%d\n", *pageCount, ctx.Write.ExtractPageNr)
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

func writePagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount int) error {

	log.Debug.Printf("*** writePagesDict begin: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return errors.Wrapf(err, "writePagesDict: unable to dereference indirect object #%d", objNumber)
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
	log.Debug.Printf("writePagesDict: This page node has %d pages\n", c)

	if c == 0 {
		log.Debug.Printf("writePagesDict: Ignore empty pages dict.\n")
		return nil
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
		// We need to write all original pagetree nodes leading to a specific leaf in order not to miss any inheritated resources.
		log.Debug.Printf("kidsArrayOrig before: %v", kidsArrayOrig)
		err = prepareSinglePageWrite(ctx, dict, kidsArrayOrig, &pageCount)
		if err != nil {
			return err
		}
		log.Debug.Printf("kidsArrayOrig after: %v", kidsArrayOrig)
	}

	err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
	if err != nil {
		return err
	}

	log.Debug.Printf("writePagesDict: %s\n", dict)

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
			return err
		}
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}
	log.Debug.Printf("kidsArray: %v", kidsArray)

	err = writeKids(ctx, kidsArray, pageCount)
	if err != nil {
		return err
	}

	dict.Update("Kids", *kidsArrayOrig)
	dict.Update("Count", count)

	log.Debug.Printf("*** writePagesDict end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return nil
}

func trimPagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount *int) (count int, err error) {

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)

	obj, err := xRefTable.Dereference(*indRef)
	if err != nil {
		return 0, errors.Wrapf(err, "trimPagesDict: unable to dereference indirect object #%d", objNumber)
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

	log.Debug.Printf("trimPagesDict: This page node has %d pages\n", int(c.(types.PDFInteger)))

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

	log.Debug.Printf("trimPagesDict end: This page node is trimmed to %d pages\n", count)
	dict.Update("Count", types.PDFInteger(count))

	log.Debug.Printf("trimPagesDict end: updated kids: %s\n", arr)
	dict.Update("Kids", arr)

	return count, nil
}

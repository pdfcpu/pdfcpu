package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func processPageAnnotations(ctx *types.PDFContext, objNumber, genNumber int, pageDict types.PDFDict) (written bool, err error) {

	logPages.Printf("*** processPageAnnotations begin: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	written, err = writeEntry(ctx, &pageDict, "pageDict", "Annots")
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** processPageAnnotations end: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	return
}

func writePagesAnnotations(ctx *types.PDFContext, indRef types.PDFIndirectRef) (written bool, err error) {

	logInfoWriter.Printf("*** writePagesAnnotations begin: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	obj, err := ctx.Dereference(indRef)
	if err != nil {
		logInfoWriter.Printf("writePagesAnnotations end: obj#%d s nil\n", objNumber)
		return
	}

	pagesDict, ok := obj.(types.PDFDict)
	if !ok {
		return false, errors.New("writePagesAnnotations: corrupt pages dict")
	}

	// Get number of pages of this PDF file.
	count, ok := pagesDict.Find("Count")
	if !ok {
		return false, errors.New("writePagesAnnotations: missing \"Count\"")
	}

	pageCount := int(count.(types.PDFInteger))
	logInfoWriter.Printf("writePagesAnnotations: This page node has %d pages\n", pageCount)

	// Iterate over page tree.
	kidsArray := pagesDict.PDFArrayEntry("Kids")

	for _, v := range *kidsArray {

		// Dereference next page node dict.
		indRef, ok := v.(types.PDFIndirectRef)
		if !ok {
			return false, errors.New("writePagesAnnotations: corrupt page node dict")
		}

		logInfoWriter.Printf("writePagesAnnotations: PageNode: %s\n", indRef)

		objNumber = int(indRef.ObjectNumber)
		genNumber = int(indRef.GenerationNumber)

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return false, errors.New("writePagesAnnotations: corrupt pageNodeDict")
		}

		if pageNodeDict == nil {
			return false, errors.New("writePagesAnnotations: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if *dictType == "Pages" {
			// Recurse over pagetree
			smthWritten, err := writePagesAnnotations(ctx, indRef)
			if err != nil {
				return false, err
			}
			if !written {
				written = smthWritten
			}
			continue
		}

		if *dictType != "Page" {
			return false, errors.Errorf("writePagesAnnotations: expected dict type: %s\n", *dictType)
		}

		// Write page dict.
		smthWritten, err := processPageAnnotations(ctx, objNumber, genNumber, *pageNodeDict)
		if err != nil {
			return false, err
		}
		if !written {
			written = smthWritten
		}
	}

	logInfoWriter.Printf("*** writePagesAnnotations end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

func writePageEntry(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) (err error) {

	written, err := writeEntry(ctx, dict, dictName, entryName)
	if err != nil {
		return
	}

	if written {
		ctx.Stats.AddPageAttr(statsAttr)
	}

	return
}

func writePageDict(ctx *types.PDFContext, objNumber, genNumber int, pageDict *types.PDFDict) (err error) {

	logPages.Printf("*** writePageDict begin: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

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

	err = writePageEntry(ctx, pageDict, dictName, "Contents", types.PageContents)
	if err != nil {
		return err
	}

	err = writePageEntry(ctx, pageDict, dictName, "Resources", types.PageResources)
	if err != nil {
		return err
	}

	err = writePageEntry(ctx, pageDict, dictName, "MediaBox", types.PageMediaBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "CropBox", types.PageCropBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "BleedBox", types.PageBleedBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "TrimBox", types.PageTrimBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "ArtBox", types.PageArtBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "BoxColorInfo", types.PageBoxColorInfo)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "PieceInfo", types.PagePieceInfo)
	if err != nil {
		return err
	}

	err = writePageEntry(ctx, pageDict, dictName, "LastModified", types.PageLastModified)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "Rotate", types.PageRotate)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "Group", types.PageGroup)
	if err != nil {
		return
	}

	// Annotations
	// delay until processing of AcroForms.
	// see ...

	err = writePageEntry(ctx, pageDict, dictName, "Thumb", types.PageThumb)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "B", types.PageB)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "Dur", types.PageDur)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "Trans", types.PageTrans)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "AA", types.PageAA)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "Metadata", types.PageMetadata)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "StructParents", types.PageStructParents)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "ID", types.PageID)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "PZ", types.PagePZ)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "SeparationInfo", types.PageSeparationInfo)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "Tabs", types.PageTabs)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "TemplateInstantiated", types.PageTemplateInstantiated)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "PresSteps", types.PagePresSteps)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "UserUnit", types.PageUserUnit)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, pageDict, dictName, "VP", types.PageVP)
	if err != nil {
		return
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
				logInfoWriter.Printf("locateKidForPageNumber: pageTree is no match")
			} else {
				logInfoWriter.Printf("locateKidForPageNumber: pageTree is a match")
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

func writePagesDict(ctx *types.PDFContext, indRef types.PDFIndirectRef, pageCount int) (err error) {

	logPages.Printf("*** writePagesDict begin: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dict, err := xRefTable.DereferenceDict(indRef)
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
		kid, err := locateKidForPageNumber(ctx, kidsArrayOrig, &pageCount, ctx.Write.ExtractPageNr)
		if err != nil {
			return err
		}

		if pageCount == ctx.Write.ExtractPageNr {
			// The identified kid is the page node for the page we are looking for.
			logInfoWriter.Printf("writePagesDict: found page to be extracted, pageCount=%d, extractPageNr=%d\n", pageCount, ctx.Write.ExtractPageNr)
		} else {
			// The identified kid is the page tree containing the page we are looking for.
			logInfoWriter.Printf("writePagesDict: pageCount=%d, extractPageNr=%d\n", pageCount, ctx.Write.ExtractPageNr)
		}

		// Modify KidsArray to hold a single entry for this kid
		dict.Update("Kids", types.PDFArray{kid})

		// Set Count =1
		dict.Update("Count", types.PDFInteger(1))

	}

	err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("writePagesDict: %s\n", dict)

	err = writePageEntry(ctx, dict, dictName, "Resources", types.PageResources)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, dict, dictName, "MediaBox", types.PageMediaBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, dict, dictName, "CropBox", types.PageCropBox)
	if err != nil {
		return
	}

	err = writePageEntry(ctx, dict, dictName, "Rotate", types.PageRotate)
	if err != nil {
		return
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}

	for _, obj := range *kidsArray {

		if obj == nil {
			logDebugWriter.Println("writePagesDict: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writePagesDict: missing indirect reference for kid")
		}

		logInfoWriter.Printf("writePagesDict: PageNode: %s\n", indRef)

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("writePagesDict: object #%d already written.\n", objNumber)
			continue
		}

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return errors.New("writePagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return errors.New("validatePagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return errors.New("writePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			err = writePagesDict(ctx, indRef, pageCount)
			if err != nil {
				return err
			}

		case "Page":
			err = writePageDict(ctx, objNumber, genNumber, pageNodeDict)
			if err != nil {
				return err
			}

		default:
			return errors.Errorf("writePagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	dict.Update("Kids", *kidsArrayOrig)
	dict.Update("Count", count)

	logPages.Printf("*** writePagesDict end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

func trimPagesDict(ctx *types.PDFContext, indRef types.PDFIndirectRef, pageCount *int) (count int, err error) {

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)

	obj, err := xRefTable.Dereference(indRef)
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

		if obj == nil {
			logDebugWriter.Println("trimPagesDict: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return 0, errors.New("trimPagesDict: missing indirect reference for kid")
		}

		logInfoWriter.Printf("trimPagesDict: PageNode: %s\n", indRef)

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return 0, errors.New("trimPagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return 0, errors.New("trimPagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return 0, errors.New("writePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

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
			return 0, errors.Errorf("trimPagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	logPages.Printf("trimPagesDict end: This page node is trimmed to %d pages\n", count)
	dict.Update("Count", types.PDFInteger(count))

	logPages.Printf("trimPagesDict end: updated kids: %s\n", arr)
	dict.Update("Kids", arr)

	return
}

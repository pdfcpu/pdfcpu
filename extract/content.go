package extract

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func writeContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict, pageNumber, section int) error {

	var fileName string

	if section >= 0 {
		fileName = fmt.Sprintf("%s/content_p%d_%d.txt", ctx.Write.DirName, pageNumber, section)
	} else {
		fileName = fmt.Sprintf("%s/content_p%d.txt", ctx.Write.DirName, pageNumber)
	}

	// Decode streamDict for supported filters only.
	err := filter.DecodeStream(streamDict)
	if err == filter.ErrUnsupportedFilter {
		return nil
	}
	if err != nil {
		return err
	}

	logInfoExtract.Printf("writing to %s\n", fileName)

	// Dump decoded chunk to file.
	return ioutil.WriteFile(fileName, streamDict.Content, os.ModePerm)
}

// Process the content of a page which is a stream dict or an array of stream dicts.
func processPageDict(ctx *types.PDFContext, objNumber, genNumber int, dict *types.PDFDict, pageNumber int) error {

	logDebugExtract.Printf("processPageDict begin: page=%d\n", pageNumber)

	obj, found := dict.Find("Contents")
	if !found || obj == nil {
		return nil
	}

	obj, err := ctx.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		err = writeContent(ctx, &obj, pageNumber, -1)
		if err != nil {
			return err
		}

	case types.PDFArray:
		// process array of content stream dicts.
		for i, obj := range obj {

			streamDict, _ := ctx.DereferenceStreamDict(obj)

			err = writeContent(ctx, streamDict, pageNumber, i)
			if err != nil {
				return err
			}

		}

	}

	return nil
}

func needsPage(selectedPages types.IntSet, pageCount int) bool {

	return selectedPages == nil || len(selectedPages) == 0 || selectedPages[pageCount]
}

func processPagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount *int, selectedPages types.IntSet) error {

	logDebugExtract.Printf("processPagesDict begin: pageCount=%d\n", *pageCount)

	dict, _ := ctx.DereferenceDict(*indRef)

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")

	for _, obj := range *kidsArray {

		if obj == nil {
			continue
		}

		// Dereference next page node dict.
		indRef, _ := obj.(types.PDFIndirectRef)
		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		pageNodeDict, _ := ctx.DereferenceDict(indRef)
		if pageNodeDict == nil {
			return errors.New("processPagesDict: pageNodeDict is null")
		}

		var err error

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over pagetree
			err = processPagesDict(ctx, &indRef, pageCount, selectedPages)

		case "Page":
			*pageCount++
			// extractContent of a page if no pages selected or if page is selected.
			if needsPage(selectedPages, *pageCount) {
				err = processPageDict(ctx, objNumber, genNumber, pageNodeDict, *pageCount)
			}

		}

		if err != nil {
			return err
		}

	}

	logDebugExtract.Printf("processPagesDict end: pageCount=%d\n", *pageCount)

	return nil
}

// Content writes content streams for selected pages to dirOut.
// Each content stream results in a separate text file.
func Content(ctx *types.PDFContext, selectedPages types.IntSet) error {

	logDebugExtract.Printf("Content begin: dirOut=%s\n", ctx.Write.DirName)

	// Get an indirect reference to the root page dict.
	indRefRootPageDict, _ := ctx.Pages()

	pageCount := 0
	err := processPagesDict(ctx, indRefRootPageDict, &pageCount, selectedPages)
	if err != nil {
		return err
	}

	logDebugExtract.Println("Content end")

	return nil
}

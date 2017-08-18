package extract

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/EndFirstCorp/pdflib/filter"
	"github.com/EndFirstCorp/pdflib/types"
	"github.com/pkg/errors"
)

func writeContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict, pageNumber, section int) (err error) {

	var fileName string

	if section >= 0 {
		fileName = fmt.Sprintf("%s/content_p%d_%d.txt", ctx.Write.DirName, pageNumber, section)
	} else {
		fileName = fmt.Sprintf("%s/content_p%d.txt", ctx.Write.DirName, pageNumber)
	}

	// Decode streamDict if used filter is supported only.
	err = filter.DecodeStream(streamDict)
	if err == filter.ErrUnsupportedFilter {
		err = nil
		return
	}
	if err != nil {
		return
	}

	// Dump decoded chunk to file.
	err = ioutil.WriteFile(fileName, streamDict.Content, os.ModePerm)

	return
}

// Process the content of a page which is a stream dict or an array of stream dicts.
func processPageDict(ctx *types.PDFContext, objNumber, genNumber int, dict *types.PDFDict, pageNumber int) (err error) {

	logDebugExtract.Printf("processPageDict begin: page=%d\n", pageNumber)

	obj, found := dict.Find("Contents")
	if !found {
		return
	}

	if obj == nil {
		return
	}

	obj, err = ctx.Dereference(obj)
	if err != nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		err = writeContent(ctx, &obj, pageNumber, -1)
		if err != nil {
			return
		}

	case types.PDFArray:

		// process array of content stream dicts.

		for i, obj := range obj {

			streamDict, err := ctx.DereferenceStreamDict(obj)
			if err != nil {
				return err
			}

			err = writeContent(ctx, streamDict, pageNumber, i)
			if err != nil {
				return err
			}

		}

	default:
		err = errors.Errorf("writePageContents: page content must be stream dict or array")
		return
	}

	return
}

func needsPage(selectedPages types.IntSet, pageCount int) bool {

	return selectedPages == nil || len(selectedPages) == 0 || selectedPages[pageCount]
}

func processPagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount *int, selectedPages types.IntSet) (err error) {

	logDebugExtract.Printf("processPagesDict begin: pageCount=%d\n", *pageCount)

	dict, err := ctx.DereferenceDict(*indRef)
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
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writePagesDict: missing indirect reference for kid")
		}

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

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
			err = processPagesDict(ctx, &indRef, pageCount, selectedPages)
			if err != nil {
				return err
			}

		case "Page":
			*pageCount++
			// extractContent of a page if no pages selected or if page is selected.
			if needsPage(selectedPages, *pageCount) {
				err = processPageDict(ctx, objNumber, genNumber, pageNodeDict, *pageCount)
				if err != nil {
					return err
				}
			}

		default:
			return errors.Errorf("writePagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	logDebugExtract.Printf("processPagesDict end: pageCount=%d\n", *pageCount)

	return
}

// Content writes content streams for selected pages to dirOut.
// Each content stream results in a separate text file.
func Content(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Printf("Content begin: dirOut=%s\n", ctx.Write.DirName)

	// Get an indirect reference to the root page dict.
	indRefRootPageDict, err := ctx.Pages()
	if err != nil {
		return err
	}

	pageCount := 0
	err = processPagesDict(ctx, indRefRootPageDict, &pageCount, selectedPages)
	if err != nil {
		return err
	}

	logDebugExtract.Println("Content end")

	return
}

/*
	Copyright 2021 The pdfcpu Authors.

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
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// DefaultBookletConfig returns the default configuration for a booklet
func DefaultBookletConfig() *model.NUp {
	nup := model.DefaultNUpConfig()
	nup.Margin = 0
	nup.Border = false
	nup.BookletGuides = false
	nup.MultiFolio = false
	nup.FolioSize = 8
	return nup
}

// PDFBookletConfig returns an NUp configuration for booklet-ing PDF files.
func PDFBookletConfig(val int, desc string) (*model.NUp, error) {
	nup := DefaultBookletConfig()
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpValue(val, nup)
}

// ImageBookletConfig returns an NUp configuration for booklet-ing image files.
func ImageBookletConfig(val int, desc string) (*model.NUp, error) {
	nup, err := PDFBookletConfig(val, desc)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

func getPageNumber(pageNumbers []int, n int) int {
	if n >= len(pageNumbers) {
		// Zero represents blank page at end of booklet.
		return 0
	}
	return pageNumbers[n]
}

func nup2OutputPageNr(inputPageNr, inputPageCount int, pageNumbers []int) (int, bool) {
	var p int
	if inputPageNr%2 == 0 {
		p = inputPageCount - 1 - inputPageNr/2
	} else {
		p = (inputPageNr - 1) / 2
	}
	pageNr := getPageNumber(pageNumbers, p)

	// Rotate odd output pages (the back sides) by 180 degrees.
	var rotate bool
	if inputPageNr%4 < 2 {
		rotate = true
	}
	return pageNr, rotate
}

func nup4OutputPageNr(inputPageNr int, inputPageCount int, pageNumbers []int) (int, bool) {
	bookletPageNumber := inputPageNr / 4
	var p int
	if bookletPageNumber%2 == 0 {
		// front side
		switch inputPageNr % 4 {
		case 0:
			p = inputPageCount - 1 - bookletPageNumber
		case 1:
			p = bookletPageNumber
		case 2:
			p = inputPageCount/2 + bookletPageNumber
		case 3:
			p = inputPageCount/2 - 1 - bookletPageNumber
		}
	} else {
		// back side
		switch inputPageNr % 4 {
		case 0:
			p = bookletPageNumber
		case 1:
			p = inputPageCount - 1 - bookletPageNumber
		case 2:
			p = inputPageCount/2 - 1 - bookletPageNumber
		case 3:
			p = inputPageCount/2 + bookletPageNumber
		}
	}
	pageNr := getPageNumber(pageNumbers, p)

	// Rotate bottom row of each output page by 180 degrees.
	var rotate bool
	if inputPageNr%4 >= 2 {
		rotate = true
	}
	return pageNr, rotate
}

type bookletPage struct {
	number int
	rotate bool
}

func sortSelectedPagesForBooklet(pages types.IntSet, nup *model.NUp) []bookletPage {
	pageNumbers := sortSelectedPages(pages)
	pageCount := len(pageNumbers)

	// A sheet of paper consists of 2 consecutive output pages.
	sheetPageCount := 2 * nup.N()

	// pageCount must be a multiple of the number of pages per sheet.
	// If not, we will insert blank pages at the end of the booklet.
	if pageCount%sheetPageCount != 0 {
		pageCount += sheetPageCount - pageCount%sheetPageCount
	}

	bookletPages := make([]bookletPage, pageCount)

	switch nup.N() {
	case 2:
		// (output page, input page) = [(1,n), (2,1), (3, n-1), (4, 2), (5, n-2), (6, 3), ...]
		for i := 0; i < pageCount; i++ {
			pageNr, rotate := nup2OutputPageNr(i, pageCount, pageNumbers)
			bookletPages[i].number = pageNr
			bookletPages[i].rotate = rotate
		}

	case 4:
		// (output page, input page) = [(1,n), (2,1), (3, n/2+1), (4, n/2-0), (5, 2), (6, n-1), (7, n/2-1), (8, n/2+2) ...]
		for i := 0; i < pageCount; i++ {
			pageNr, rotate := nup4OutputPageNr(i, pageCount, pageNumbers)
			bookletPages[i].number = pageNr
			bookletPages[i].rotate = rotate
		}
	}

	return bookletPages
}

func bookletPages(
	ctx *model.Context,
	selectedPages types.IntSet,
	nup *model.NUp,
	pagesDict types.Dict,
	pagesIndRef *types.IndirectRef) error {

	var buf bytes.Buffer
	formsResDict := types.NewDict()
	rr := nup.RectsForGrid()

	for i, bp := range sortSelectedPagesForBooklet(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}
			buf.Reset()
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.number == 0 {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		if err := ctx.NUpTilePDFBytesForPDF(bp.number, formsResDict, &buf, rDest, nup, bp.rotate); err != nil {
			return err
		}
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// BookletFromImages creates a booklet version of the image sequence represented by fileNames.
func BookletFromImages(ctx *model.Context, fileNames []string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	// The order of images in fileNames corresponds to a desired booklet page sequence.
	selectedPages := types.IntSet{}
	for i := 1; i <= len(fileNames); i++ {
		selectedPages[i] = true
	}

	if nup.PageGrid {
		nup.PageDim.Width *= nup.Grid.Width
		nup.PageDim.Height *= nup.Grid.Height
	}

	xRefTable := ctx.XRefTable
	formsResDict := types.NewDict()
	var buf bytes.Buffer
	rr := nup.RectsForGrid()

	for i, bp := range sortSelectedPagesForBooklet(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.number == 0 {
			// This is an empty page at the end of a booklet.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileNames[bp.number-1])
		if err != nil {
			return err
		}

		imgIndRef, w, h, err := model.CreateImageResource(xRefTable, f, false, false)
		if err != nil {
			return err
		}

		if err := f.Close(); err != nil {
			return err
		}

		formIndRef, err := createNUpFormForImage(xRefTable, imgIndRef, w, h, i)
		if err != nil {
			return err
		}

		formResID := fmt.Sprintf("Fm%d", i)
		formsResDict.Insert(formResID, *formIndRef)

		// Append to content stream of booklet page i.
		enforceOrientation := false
		model.NUpTilePDFBytes(&buf, types.RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, bp.rotate, enforceOrientation)
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// BookletFromPDF creates a booklet version of the PDF represented by xRefTable.
func BookletFromPDF(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	n := int(nup.Grid.Width * nup.Grid.Height)
	if !(n == 2 || n == 4) {
		return fmt.Errorf("pdfcpu: booklet must have n={2,4} pages per sheet, got %d", n)
	}

	var mb *types.Rectangle

	if nup.PageDim == nil {
		nup.PageDim = types.PaperSize[nup.PageSize]
	}

	mb = types.RectForDim(nup.PageDim.Width, nup.PageDim.Height)

	pagesDict := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(0),
			"MediaBox": mb.Array(),
		},
	)

	pagesIndRef, err := ctx.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	nup.PageDim = &types.Dim{Width: mb.Width(), Height: mb.Height()}

	if nup.MultiFolio {
		pages := types.IntSet{}
		for _, i := range sortSelectedPages(selectedPages) {
			pages[i] = true
			if len(pages) == 4*nup.FolioSize {
				if err = bookletPages(ctx, pages, nup, pagesDict, pagesIndRef); err != nil {
					return err
				}
				pages = types.IntSet{}
			}
		}
		if len(pages) > 0 {
			if err = bookletPages(ctx, pages, nup, pagesDict, pagesIndRef); err != nil {
				return err
			}
		}

	} else {
		if err = bookletPages(ctx, selectedPages, nup, pagesDict, pagesIndRef); err != nil {
			return err
		}
	}

	// Replace original pagesDict.
	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	rootDict.Update("Pages", *pagesIndRef)
	return nil
}

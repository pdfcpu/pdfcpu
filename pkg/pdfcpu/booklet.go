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
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var errInvalidBookletAdvanced = errors.New("pdfcpu booklet advanced cannot have binding along the top (portrait short-edge, landscape long-edge). use plain booklet instead.")

var NUpValuesForBooklets = []int{2, 4, 6, 8}

// DefaultBookletConfig returns the default configuration for a booklet
func DefaultBookletConfig() *model.NUp {
	nup := model.DefaultNUpConfig()
	nup.Margin = 0
	nup.Border = false
	nup.BookletGuides = false
	nup.MultiFolio = false
	nup.FolioSize = 8
	nup.BookletType = model.Booklet
	nup.BookletBinding = model.LongEdge
	nup.Enforce = true
	return nup
}

// PDFBookletConfig returns an NUp configuration for booklet-ing PDF files.
func PDFBookletConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup := DefaultBookletConfig()
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	nup.InpUnit = conf.Unit
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	if !types.IntMemberOf(val, NUpValuesForBooklets) {
		ss := make([]string, len(NUpValuesForBooklets))
		for i, v := range NUpValuesForBooklets {
			ss[i] = strconv.Itoa(v)
		}
		return nil, errors.Errorf("pdfcpu: n must be one of %s", strings.Join(ss, ", "))
	}
	if err := ParseNUpValue(val, nup); err != nil {
		return nil, err
	}
	// 6up special cases
	if nup.IsBooklet() && val == 6 && nup.IsTopFoldBinding() {
		// You can't top fold a 6up with 3 rows.
		return nup, fmt.Errorf("pdfcpu booklet: n=6 must have binding on side (portrait long-edge or landscape short-edge)")
	}
	// bookletadvanced
	if nup.BookletType == model.BookletAdvanced && val == 4 && nup.IsTopFoldBinding() {
		return nup, errInvalidBookletAdvanced
	}
	return nup, nil
}

// ImageBookletConfig returns an NUp configuration for booklet-ing image files.
func ImageBookletConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup, err := PDFBookletConfig(val, desc, conf)
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

type pageNumberFunction func(inputPageNr int, pageCount int, pageNumbers []int, nup *model.NUp) (int, bool)

func nup2OutputPageNr(inputPageNr, inputPageCount int, pageNumbers []int, _ *model.NUp) (int, bool) {
	// (output page, input page) = [(1,n), (2,1), (3, n-1), (4, 2), (5, n-2), (6, 3), ...]
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

func get4upPos(pos int, isLandscape bool) (out int) {
	if isLandscape {
		switch pos % 4 {
		// landscape short-edge binding page ordering is rotated 90 degrees anti-clockwise from the portrait ordering on the back sides of the pages to make duplexing work
		// from portrait to lanscape map {0 => 3, 1 => 2, 2 => 1, 3 => 0}
		case 0:
			return 3
		case 1:
			return 2
		case 2:
			return 1
		case 3:
			return 0
		}
	}
	return pos % 4
}

func nup4OutputPageNr(inputPageNr int, pageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	switch nup.BookletType {
	case model.Booklet:
		// simple booklets are collated by collecting the top of the sheet, then the bottom, then the top of the next sheet, and so on.
		// this is conceptually easier for collation without specialized tools.
		if nup.IsTopFoldBinding() {
			return nup4BasicTopFoldOutputPageNr(inputPageNr, pageCount, pageNumbers, nup)
		} else {
			return nup4BasicSideFoldOutputPageNr(inputPageNr, pageCount, pageNumbers, nup)
		}
	case model.BookletAdvanced:
		// advanced booklets have a different collation pattern: collect the top of each sheet and then the bottom of each sheet.
		// this allows printers to fold the sheets twice and then cut along one of the folds.
		return nup4AdvancedSideFoldOutputPageNr(inputPageNr, pageCount, pageNumbers, nup)
	}
	return 0, false
}

func nup4BasicSideFoldOutputPageNr(positionNumber int, inputPageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	var p int
	bookletSheetSideNumber := positionNumber / 4
	bookletPageNumber := positionNumber / 8
	if bookletSheetSideNumber%2 == 0 {
		// front side
		n := bookletPageNumber * 4
		switch positionNumber % 4 {
		case 0:
			p = inputPageCount - n
		case 1:
			p = 1 + n
		case 2:
			p = 3 + n
		case 3:
			p = inputPageCount - 2 - n
		}
	} else {
		// back side
		n := bookletPageNumber * 4
		switch get4upPos(positionNumber, nup.PageDim.Landscape()) {
		case 0:
			p = 2 + n
		case 1:
			p = inputPageCount - 1 - n
		case 2:
			p = inputPageCount - 3 - n
		case 3:
			p = 4 + n
		}
	}
	pageNr := getPageNumber(pageNumbers, p-1) // p is one-indexed and we want zero-indexed
	// Rotate bottom row of each output sheet by 180 degrees.
	var rotate bool
	if positionNumber%4 >= 2 {
		rotate = true
	}
	return pageNr, rotate
}

func nup4BasicTopFoldOutputPageNr(positionNumber int, inputPageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	var p int
	bookletSheetSideNumber := positionNumber / 4
	bookletSheetNumber := positionNumber / 8
	if bookletSheetSideNumber%2 == 0 {
		// front side
		switch positionNumber % 4 {
		case 0:
			p = inputPageCount - 4*bookletSheetNumber
		case 1:
			p = 3 + 4*bookletSheetNumber
		case 2:
			p = 1 + 4*bookletSheetNumber
		case 3:
			p = inputPageCount - 2 - 4*bookletSheetNumber
		}
	} else {
		// back side
		switch get4upPos(positionNumber, nup.PageDim.Landscape()) {
		case 0:
			p = 4 + 4*bookletSheetNumber
		case 1:
			p = inputPageCount - 1 - 4*bookletSheetNumber
		case 2:
			p = inputPageCount - 3 - 4*bookletSheetNumber
		case 3:
			p = 2 + 4*bookletSheetNumber
		}
	}
	pageNr := getPageNumber(pageNumbers, p-1) // p is one-indexed and we want zero-indexed
	// Rotate right side of output page by 180 degrees.
	var rotate bool
	if positionNumber%2 == 1 {
		rotate = true
	}
	return pageNr, rotate
}

func nup4AdvancedSideFoldOutputPageNr(inputPageNr int, inputPageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	// (output page, input page) = [(1,n), (2,1), (3, n/2+1), (4, n/2-0), (5, 2), (6, n-1), (7, n/2-1), (8, n/2+2) ...]
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
		// back side (portrait)
		switch get4upPos(inputPageNr, nup.PageDim.Landscape()) {
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

func nupLRTBOutputPageNr(positionNumber int, inputPageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	// move from left to right and then from top to bottom with no rotation
	var p int
	N := nup.N()
	bookletSheetSideNumber := positionNumber / N
	bookletSheetNumber := positionNumber / (2 * N)
	if bookletSheetSideNumber%2 == 0 {
		// front side
		if positionNumber%2 == 0 {
			// left side - count down from end
			p = inputPageCount - N*bookletSheetNumber - positionNumber%N
		} else {
			// right side - count up from start
			p = N*bookletSheetNumber + positionNumber%N
		}
	} else {
		// back side
		if positionNumber%2 == 0 {
			// left side - count up from start
			p = 2 + N*bookletSheetNumber + positionNumber%N
		} else {
			// right side - count down from end
			p = inputPageCount - N*bookletSheetNumber - positionNumber%N
		}
	}
	pageNr := getPageNumber(pageNumbers, p-1) // p is one-indexed and we want zero-indexed
	return pageNr, false
}

func nup8OutputPageNr(portraitPositionNumber int, inputPageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	// 8up sheet has four rows and two columns
	// but the spreads are NOT across the two columns - instead the spreads are rotated 90deg to fit in a portrait orientation on the sheet
	// rather than coding up an entire new imposition, we're going to use the left-down-top-bottom imposition as a base
	// and the rotate the spreads (ie reorder) to fit on the sheet

	bookletSheetSideNumber := portraitPositionNumber / 8
	var landscapePositionNumber int
	switch bookletSheetSideNumber % 2 {
	case 0: // front side
		// rotate the block of four pages 90deg clockwise to go from portrait to landscape.             sequence=[1,3,0,2]
		// then because we are rotating the right side by 180deg - so need to change to those positions. sequence=[1,2,0,3]
		switch portraitPositionNumber % 4 {
		case 0:
			landscapePositionNumber = 1
		case 1:
			landscapePositionNumber = 2
		case 2:
			landscapePositionNumber = 0
		case 3:
			landscapePositionNumber = 3
		}
	case 1: // back side
		// rotate the block of four pages 90deg anti-clockwise to go from portrait to landscape.           sequence=[2,0,3,1]
		// then because we are rotating the *left* side by 180deg - so need to change to those positions. sequence=[3,0,2,1]
		// this is different from the front side because of the non-duplex sheet handling flip along the short edge

		switch portraitPositionNumber % 4 {
		case 0:
			landscapePositionNumber = 3
		case 1:
			landscapePositionNumber = 0
		case 2:
			landscapePositionNumber = 2
		case 3:
			landscapePositionNumber = 1
		}

	}
	positionNumber := landscapePositionNumber + portraitPositionNumber/4*4
	pageNumber, _ := nupLRTBOutputPageNr(positionNumber, inputPageCount, pageNumbers, nup)
	// rotate right side so that bottom edge of pages is on the center cut
	rotate := portraitPositionNumber%2 == 1
	return pageNumber, rotate
}

func nupPerfectBound(positionNumber int, inputPageCount int, pageNumbers []int, nup *model.NUp) (int, bool) {
	// input: positionNumber
	// output: original page number and rotation
	var p int
	var rotate bool
	N := nup.N()
	twoN := N * 2

	bookletSheetSideNumber := positionNumber / N
	bookletSheetNumber := positionNumber / twoN
	if bookletSheetSideNumber%2 == 0 {
		// front side
		p = bookletSheetNumber*twoN + 2*(positionNumber%twoN) + 1
	} else {
		// back side
		p = bookletSheetNumber*twoN + 2*((positionNumber-N)%twoN) + 2
		if N == 4 || N == 6 || N == 8 {
			if N == 4 && nup.PageDim.Landscape() { // landscape pages on portrait sheets
				// flip top and bottom rows to account for landscape rotation and the page handling flip (short edge flip, no duplex)
				if positionNumber%N < 2 { // top side
					p += 4
				} else { // bottom side
					p -= 4
				}
			} else { // portrait pages on portrait sheets
				// flip left and right columns to account for the page handling flip (short edge flip, no duplex)
				if positionNumber%2 == 0 { // left side
					p += 2
				} else { // right side
					p -= 2
				}
			}
		}
		// account for page handling flip (short edge flip, no duplex)
		rotate = N == 2 || nup.PageDim.Landscape()
	}
	return getPageNumber(pageNumbers, p-1), rotate // p is one-indexed and we want zero-indexed
}

func GetBookletOrdering(pages types.IntSet, nup *model.NUp) []model.BookletPage {
	pageNumbers := sortSelectedPages(pages)
	pageCount := len(pageNumbers)

	// A sheet of paper consists of 2 consecutive output pages.
	sheetPageCount := 2 * nup.N()

	// pageCount must be a multiple of the number of pages per sheet.
	// If not, we will insert blank pages at the end of the booklet.
	if pageCount%sheetPageCount != 0 {
		pageCount += sheetPageCount - pageCount%sheetPageCount
	}

	if nup.MultiFolio {
		bookletPages := make([]model.BookletPage, 0)
		// folioSize is the number of sheets - each "folio" has two sides and two pages per side
		nPagesPerSignature := nup.FolioSize * 4
		nSignaturesInBooklet := int(math.Ceil(float64(pageCount) / float64(nPagesPerSignature)))
		for j := 0; j < nSignaturesInBooklet; j++ {
			start := j * nPagesPerSignature
			stop := (j + 1) * nPagesPerSignature
			if stop > len(pageNumbers) {
				// last signature may be short
				stop = len(pageNumbers)
				nPagesPerSignature = pageCount - start
			}
			bookletPages = append(bookletPages, getBookletPageOrdering(nup, pageNumbers[start:stop], nPagesPerSignature)...)
		}
		return bookletPages
	}
	return getBookletPageOrdering(nup, pageNumbers, pageCount)
}

func getBookletPageOrdering(nup *model.NUp, pageNumbers []int, pageCount int) []model.BookletPage {
	bookletPages := make([]model.BookletPage, pageCount)

	var pageNumberFn pageNumberFunction
	switch nup.BookletType {
	case model.Booklet, model.BookletAdvanced:
		switch nup.N() {
		case 2:
			pageNumberFn = nup2OutputPageNr
		case 4:
			pageNumberFn = nup4OutputPageNr
		case 6:
			pageNumberFn = nupLRTBOutputPageNr
		case 8:
			if nup.BookletBinding == model.ShortEdge {
				pageNumberFn = nupLRTBOutputPageNr
			} else { // long edge
				pageNumberFn = nup8OutputPageNr
			}
		}
	case model.BookletPerfectBound:
		pageNumberFn = nupPerfectBound
	}

	for i := 0; i < pageCount; i++ {
		pageNr, rotate := pageNumberFn(i, pageCount, pageNumbers, nup)
		bookletPages[i].Number = pageNr
		bookletPages[i].Rotate = rotate
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

	for i, bp := range GetBookletOrdering(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}
			buf.Reset()
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.Number == 0 {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		if err := ctx.NUpTilePDFBytesForPDF(bp.Number, formsResDict, &buf, rDest, nup, bp.Rotate); err != nil {
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

	for i, bp := range GetBookletOrdering(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.Number == 0 {
			// This is an empty page at the end of a booklet.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileNames[bp.Number-1])
		if err != nil {
			return err
		}

		imgIndRef, w, h, err := model.CreateImageResource(xRefTable, f)
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
		model.NUpTilePDFBytes(&buf, types.RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, bp.Rotate)
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// BookletFromPDF creates a booklet version of the PDF represented by xRefTable.
func BookletFromPDF(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	n := int(nup.Grid.Width * nup.Grid.Height)
	if !(n == 2 || n == 4 || n == 6 || n == 8) {
		return fmt.Errorf("pdfcpu: booklet must have n={2,4,6,8} pages per sheet, got %d", n)
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

	if err = bookletPages(ctx, selectedPages, nup, pagesDict, pagesIndRef); err != nil {
		return err
	}

	// Replace original pagesDict.
	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	rootDict.Update("Pages", *pagesIndRef)
	return nil
}

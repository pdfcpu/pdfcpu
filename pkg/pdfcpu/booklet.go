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
	"sort"
)

// DefaultBookletConfig returns the default configuration for a booklet
func DefaultBookletConfig() *NUp {
	return &NUp{
		PageSize: "A4",
		Orient:   RightDown,
		Margin:   0,
		Border:   false,
	}
}

// PDFBookletConfig returns an NUp configuration for booklet-ing PDF files.
func PDFBookletConfig(val int, desc string) (*NUp, error) {
	nup := DefaultBookletConfig()
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpValue(val, nup)
}

// BookletFromPDF creates an booklet version of the PDF represented by xRefTable.
func (ctx *Context) BookletFromPDF(selectedPages IntSet, nup *NUp) error {
	n := int(nup.Grid.Width * nup.Grid.Height)
	if !(n == 2 || n == 4) {
		return fmt.Errorf("booklet must have n={2,4} pages per sheet, got %d", n)
	}
	if n == 2 && nup.PageDim.Landscape() {
		// transpose grid
		nup.Grid = &Dim{nup.Grid.Height, nup.Grid.Width}
	}
	return ctx.bookletFromPDF(selectedPages, nup)
}

func getPageNumber(pageNumbers []int, n int) int {
	if n >= len(pageNumbers) {
		// zero represents blank page at end of booklet
		return 0
	}
	return pageNumbers[n]
}

func sortedSelectedPagesBooklet(pages IntSet, nup *NUp) ([]int, []bool) {
	var pageNumbers []int
	for k, v := range pages {
		if v {
			pageNumbers = append(pageNumbers, k)
		}
	}
	sort.Ints(pageNumbers)

	nPagesPerSheetSide := int(nup.Grid.Height * nup.Grid.Width)
	nPagesPerSheet := 2 * nPagesPerSheetSide

	nPages := len(pageNumbers)
	rem := nPages % nPagesPerSheet
	if rem != 0 {
		rem = nPagesPerSheet - rem
	}
	nPages += rem

	// nPages must be a multiple of the number of pages per sheet
	// if not, we will insert blank pages at the end of the booklet
	pageNumbersBookletOrder := make([]int, nPages)
	shouldRotate := make([]bool, nPages)
	switch nPagesPerSheetSide {
	case 2:
		// (output page, input page) = [(1,n), (2,1), (3, n-1), (4, 2), (5, n-2), (6, 3), ...]
		for i := 0; i < nPages; i++ {
			if i%2 == 0 {
				pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages-1-i/2)
			} else {
				pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, (i-1)/2)
			}
			// odd output sheet sides (the back sides) should be upside-down
			if i%4 < 2 {
				shouldRotate[i] = true
			}
		}
	case 4:
		// (output page, input page) = [
		// (1,n), (2,1), (3, n/2+1), (4, n/2-0),
		// (5, 2), (6, n-1), (7, n/2-1), (8, n/2+2)
		// ...]
		for i := 0; i < nPages; i++ {
			bookletPageNumber := i / 4
			if bookletPageNumber%2 == 0 {
				// front side
				switch i % 4 {
				case 0:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages-1-bookletPageNumber)
				case 1:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, bookletPageNumber)
				case 2:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages/2+bookletPageNumber)
				case 3:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages/2-1-bookletPageNumber)
				}
			} else {
				// back side
				switch i % 4 {
				case 0:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, bookletPageNumber)
				case 1:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages-1-bookletPageNumber)
				case 2:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages/2-1-bookletPageNumber)
				case 3:
					pageNumbersBookletOrder[i] = getPageNumber(pageNumbers, nPages/2+bookletPageNumber)
				}
			}
			if i%4 >= 2 {
				// bottom row of each output page should be rotated
				shouldRotate[i] = true
			}
		}
	}
	return pageNumbersBookletOrder, shouldRotate
}

func calcTransMatrixForRectBooklet(r1, r2 *Rectangle, image bool) (float64, matrix, matrix, matrix) {
	rot, m1, m2, m3 := calcTransMatrixForRectNUp(r1, r2, image)
	// Rotation: booklet pages are rotated 180deg, in addition to any aspect rotation
	// this is equivalent to flipping the sign on first two rows/cols of the m2 matrix
	m2[0][0] *= -1
	m2[0][1] *= -1
	m2[1][0] *= -1
	m2[1][1] *= -1

	// Translation: we've rotated 180deg in addition to the original rotation (for aspect ratio)
	// so we need to translate to get the old page visiible on the new page
	if rot == 0 { // new rotation is 180deg
		m3[2][0] += r1.Width()
	} else { // new rotation is 270deg
		m3[2][0] -= r1.Width()
	}
	m3[2][1] += r1.Height()
	return rot, m1, m2, m3
}

func (ctx *Context) arrangePagesForBooklet(selectedPages IntSet, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	// this code is similar to nupPages, but for booklets
	var buf bytes.Buffer
	formsResDict := NewDict()
	rr := rectsForGrid(nup)

	pageNumbers, shouldRotatePage := sortedSelectedPagesBooklet(selectedPages, nup)
	for i, p := range pageNumbers {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete nUp page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = NewDict()
		}
		if p == 0 {
			// this is an empty page at the end of a booklet
			continue
		}
		isEmpty, cropBox, formResID, err := ctx.nupPrepPage(i, p, formsResDict)
		if err != nil {
			return err
		}
		if isEmpty {
			continue
		}
		var calc calcTransMatrixForRect
		if shouldRotatePage[i] {
			calc = calcTransMatrixForRectBooklet
		} else {
			calc = calcTransMatrixForRectNUp
		}
		nUpTilePDFBytes(&buf, cropBox, rr[i%len(rr)], formResID, nup, calc)
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// bookletFromPDF arranges the PDF represented by xRefTable into a booklet
// this code is similar to NUpFromPDF, but for booklets
func (ctx *Context) bookletFromPDF(selectedPages IntSet, nup *NUp) error {
	pagesDict, err := ctx.getNupPagesDict(nup)
	if err != nil {
		return err
	}
	pagesIndRef, err := ctx.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}
	if err = ctx.arrangePagesForBooklet(selectedPages, nup, pagesDict, pagesIndRef); err != nil {
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

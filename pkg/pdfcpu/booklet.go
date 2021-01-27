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
	"io"
	"math"
	"sort"

	"github.com/pkg/errors"
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

func calcTransMatrixForRectBookletRotated(r1, r2 *Rectangle, image bool) matrix {
	// ---- this block is a copy of the logic in nup ----
	var (
		w, h   float64
		dx, dy float64
		rot    float64
	)

	if r2.Landscape() && r1.Portrait() || r2.Portrait() && r1.Landscape() {
		rot = 90
		r1.UR.X, r1.UR.Y = r1.UR.Y, r1.UR.X
	}

	if r1.FitsWithin(r2) {
		// Translate r1 into center of r2 w/o scaling up.
		w = r1.Width()
		h = r1.Height()
	} else if r1.AspectRatio() <= r2.AspectRatio() {
		// Scale down r1 height to fit into r2 height.
		h = r2.Height()
		w = r1.ScaledWidth(h)
	} else {
		// Scale down r1 width to fit into r2 width.
		w = r2.Width()
		h = r1.ScaledHeight(w)
	}

	dx = r2.LL.X - r1.LL.X*w/r1.Width() + r2.Width()/2 - w/2
	dy = r2.LL.Y - r1.LL.Y*h/r1.Height() + r2.Height()/2 - h/2

	if rot > 0 {
		dx += w
		if !image {
			w /= r1.Width()
			h /= r1.Height()
		}
		w, h = h, w
	} else if !image {
		w /= r1.Width()
		h /= r1.Height()
	}

	// Scale
	m1 := identMatrix
	m1[0][0] = w
	m1[1][1] = h

	// Rotate
	m2 := identMatrix
	sin := math.Sin(float64(rot) * float64(degToRad))
	cos := math.Cos(float64(rot) * float64(degToRad))
	m2[0][0] = cos
	m2[0][1] = sin
	m2[1][0] = -sin
	m2[1][1] = cos

	// Translate
	m3 := identMatrix
	m3[2][0] = dx
	m3[2][1] = dy

	// ---- booklet specific modifications ----
	// Rotation: booklet pages are rotated 180deg, in addition to any aspect rotation
	// this is equivalent to flipping the sign on first two rows/cols of the m2 matrix
	m2[0][0] *= -1
	m2[0][1] *= -1
	m2[1][0] *= -1
	m2[1][1] *= -1

	// Translation: booklet pages are rotated 180deg in addition to the original rotation (for aspect ratio)
	// so we need to translate to get the old page visiible on the new page
	if rot == 0 { // new rotation is 180deg
		m3[2][0] += r1.Width()
	} else { // new rotation is 270deg
		m3[2][0] -= r1.Width()
	}
	m3[2][1] += r1.Height()
	return m1.multiply(m2).multiply(m3)
}

type calcTransMatrix func(r1, r2 *Rectangle, image bool) matrix

func bookletTilePDFBytes(wr io.Writer, r1, r2 *Rectangle, formResID string, nup *NUp, calc calcTransMatrix) {
	// Draw bounding box.
	if nup.Border {
		fmt.Fprintf(wr, "[]0 d 0.1 w %.2f %.2f m %.2f %.2f l %.2f %.2f l %.2f %.2f l s ",
			r2.LL.X, r2.LL.Y, r2.UR.X, r2.LL.Y, r2.UR.X, r2.UR.Y, r2.LL.X, r2.UR.Y,
		)
	}

	// Apply margin.
	croppedRect := r2.CroppedCopy(float64(nup.Margin))

	m := calc(r1, croppedRect, nup.ImgInputFile)

	fmt.Fprintf(wr, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s Do Q ",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], formResID)
}

func (ctx *Context) bookletPages(selectedPages IntSet, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	// this code is similar to nupPages, but for booklets
	xRefTable := ctx.XRefTable
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

		// this block is a copy of the logic in nupPages
		consolidateRes := true
		d, inhPAttrs, err := ctx.PageDict(p, consolidateRes)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("pdfcpu: unknown page number: %d\n", i)
		}

		// Retrieve content stream bytes.
		bb, err := xRefTable.PageContent(d)
		if err == errNoContent {
			continue
		}
		if err != nil {
			return err
		}

		// Create an object for this resDict in xRefTable.
		ir, err := ctx.IndRefForNewObject(inhPAttrs.resources)
		if err != nil {
			return err
		}

		cropBox := inhPAttrs.mediaBox
		if inhPAttrs.cropBox != nil {
			cropBox = inhPAttrs.cropBox
		}
		formIndRef, err := createNUpFormForPDFResource(xRefTable, ir, bb, cropBox)
		if err != nil {
			return err
		}

		formResID := fmt.Sprintf("Fm%d", i)
		formsResDict.Insert(formResID, *formIndRef)

		var calc calcTransMatrix
		if shouldRotatePage[i] {
			calc = calcTransMatrixForRectBookletRotated
		} else {
			calc = calcTransMatrixForRect
		}
		bookletTilePDFBytes(&buf, cropBox, rr[i%len(rr)], formResID, nup, calc)
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
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

	// below is a copy of the logic in NUpFromPDF
	var mb *Rectangle
	if nup.PageDim == nil {
		// No page dimensions specified, use mediaBox of page 1.
		consolidateRes := false
		d, inhPAttrs, err := ctx.PageDict(1, consolidateRes)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("unknown page number: %d\n", 1)
		}
		mb = inhPAttrs.mediaBox
	} else {
		mb = RectForDim(nup.PageDim.Width, nup.PageDim.Height)
	}

	if nup.PageGrid {
		mb.UR.X = mb.LL.X + float64(nup.Grid.Width)*mb.Width()
		mb.UR.Y = mb.LL.Y + float64(nup.Grid.Height)*mb.Height()
	}

	pagesDict := Dict(
		map[string]Object{
			"Type":     Name("Pages"),
			"Count":    Integer(0),
			"MediaBox": mb.Array(),
		},
	)

	pagesIndRef, err := ctx.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	nup.PageDim = &Dim{mb.Width(), mb.Height()}

	// instead of nup-ing the pages, we make them into a booklet
	if err = ctx.bookletPages(selectedPages, nup, pagesDict, pagesIndRef); err != nil {
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

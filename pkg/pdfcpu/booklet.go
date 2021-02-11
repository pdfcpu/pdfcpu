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
	"os"

	"github.com/pkg/errors"
)

// DefaultBookletConfig returns the default configuration for a booklet
func DefaultBookletConfig() *NUp {
	nup := DefaultNUpConfig()
	nup.Margin = 0
	nup.Border = false
	nup.BookletGuides = false
	return nup
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

// ImageBookletConfig returns an NUp configuration for booklet-ing image files.
func ImageBookletConfig(val int, desc string) (*NUp, error) {
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

func drawGuideLineLabel(w io.Writer, x, y float64, s string, mb *Rectangle, fm FontMap, rot int) {
	fontName := "Helvetica"
	td := TextDescriptor{
		FontName:  fontName,
		FontKey:   fm.EnsureKey(fontName),
		FontSize:  9,
		Scale:     1.0,
		ScaleAbs:  true,
		StrokeCol: Black,
		FillCol:   Black,
		X:         x,
		Y:         y,
		Rotation:  float64(rot),
		Text:      s,
	}
	WriteMultiLine(w, mb, nil, td)
}

func drawScissor(w io.Writer, mb *Rectangle, fm FontMap) {
	fontName := "ZapfDingbats"
	td := TextDescriptor{
		FontName:  fontName,
		FontKey:   fm.EnsureKey(fontName),
		FontSize:  12,
		Scale:     1.0,
		ScaleAbs:  true,
		StrokeCol: Black,
		FillCol:   Black,
		X:         0,
		Y:         mb.Height()/2 - 4,
		Text:      string([]byte{byte(34)}),
	}
	WriteMultiLine(w, mb, nil, td)
}

func drawBookletGuides(nup *NUp, w io.Writer) FontMap {
	width := nup.PageDim.Width
	height := nup.PageDim.Height
	var fm FontMap = FontMap{}
	mb := RectForDim(nup.PageDim.Width, nup.PageDim.Height)

	SetLineWidth(w, 0)
	SetStrokeColor(w, Gray)

	switch nup.N() {
	case 2:
		// Draw horizontal folding line.
		fmt.Fprint(w, "[3] 0 d ")
		DrawLine(w, 0, height/2, width, height/2)
		drawGuideLineLabel(w, 1, height/2+2, "Fold here", mb, fm, 0)
	case 4:
		// Draw vertical folding line.
		fmt.Fprint(w, "[3] 0 d ")
		DrawLine(w, width/2, 0, width/2, height)
		drawGuideLineLabel(w, width/2-23, 20, "Fold here", mb, fm, 90)

		// Draw horizontal cutting line.
		fmt.Fprint(w, "[3] 0 d ")
		DrawLine(w, 0, height/2, width, height/2)
		drawGuideLineLabel(w, width, height/2+2, "Fold & Cut here", mb, fm, 0)

		// Draw scissors over cutting line.
		drawScissor(w, mb, fm)
	}

	return fm
}

type bookletPage struct {
	number int
	rotate bool
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

func sortSelectedPagesForBooklet(pages IntSet, nup *NUp) []bookletPage {
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

func (ctx *Context) bookletPages(selectedPages IntSet, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	xRefTable := ctx.XRefTable
	var buf bytes.Buffer
	formsResDict := NewDict()
	rr := rectsForGrid(nup)

	for i, bp := range sortSelectedPagesForBooklet(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete booklet page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.number == 0 {
			// This is an empty page at the end of a booklet.
			if nup.BgColor != nil {
				FillRectStacked(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		consolidateRes := true
		d, inhPAttrs, err := ctx.PageDict(bp.number, consolidateRes)
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
		formIndRef, err := createNUpFormForPDF(xRefTable, ir, bb, cropBox)
		if err != nil {
			return err
		}

		formResID := fmt.Sprintf("Fm%d", i)
		formsResDict.Insert(formResID, *formIndRef)

		// Append to content stream of page i.
		nUpTilePDFBytes(&buf, cropBox, rDest, formResID, nup, bp.rotate)
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// BookletFromPDF creates a booklet version of the PDF represented by xRefTable.
func (ctx *Context) BookletFromPDF(selectedPages IntSet, nup *NUp) error {
	n := int(nup.Grid.Width * nup.Grid.Height)
	if !(n == 2 || n == 4) {
		return fmt.Errorf("pdfcpu: booklet must have n={2,4} pages per sheet, got %d", n)
	}

	var mb *Rectangle

	if nup.PageDim == nil {
		nup.PageDim = PaperSize[nup.PageSize]
	}

	mb = RectForDim(nup.PageDim.Width, nup.PageDim.Height)

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

// BookletFromImages creates a booklet version of the image sequence represented by fileNames.
func BookletFromImages(ctx *Context, fileNames []string, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	// The order of images in fileNames corresponds to a desired booklet page sequence.
	selectedPages := IntSet{}
	for i := 1; i <= len(fileNames); i++ {
		selectedPages[i] = true
	}

	if nup.PageGrid {
		nup.PageDim.Width *= nup.Grid.Width
		nup.PageDim.Height *= nup.Grid.Height
	}

	xRefTable := ctx.XRefTable
	formsResDict := NewDict()
	var buf bytes.Buffer
	rr := rectsForGrid(nup)

	for i, bp := range sortSelectedPagesForBooklet(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete booklet page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.number == 0 {
			// This is an empty page at the end of a booklet.
			if nup.BgColor != nil {
				FillRectStacked(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileNames[bp.number-1])
		if err != nil {
			return err
		}

		imgIndRef, w, h, err := createImageResource(xRefTable, f)
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
		nUpTilePDFBytes(&buf, RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, bp.rotate)
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

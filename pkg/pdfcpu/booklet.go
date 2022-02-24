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

type bookletType int

// These are the types of booklet layouts
const (
	Booklet bookletType = iota
	BookletAdvanced
	BookletPerfectBound
	BookletCover
	BookletCoverFullSpan
)

func (b bookletType) String() string {
	switch b {

	case Booklet:
		return "booklet"
	case BookletAdvanced:
		return "booklet advanced"
	case BookletPerfectBound:
		return "booklet perfect bound"
	case BookletCover:
		return "booklet cover"
	case BookletCoverFullSpan:
		return "booklet cover full span"

	}

	return ""
}

type bookletBinding int

const (
	shortEdge bookletBinding = iota
	longEdge
)

func (b bookletBinding) String() string {
	switch b {
	case shortEdge:
		return "short-edge"
	case longEdge:
		return "long-edge"
	}
	return ""
}

var (
	errInvalidBookletGridID          = errors.New("pdfcpu booklet: n must be one of 2, 4, 6, 8")
	errInvalidBookletCoverGridID     = errors.New("pdfcpu booklet cover: n must be one of 2, 4")
	errInvalidBookletCoverFullGridID = errors.New("pdfcpu booklet cover full-span: n must 2")
	errInvalidBookletAdvanced        = errors.New("pdfcpu booklet advanced cannot have binding along the top (portrait short-edge, landscape long-edge). use plain booklet instead.")
)

// DefaultBookletConfig returns the default configuration for a booklet
func DefaultBookletConfig() *NUp {
	nup := DefaultNUpConfig()
	nup.Margin = 0
	nup.Border = false
	nup.BookletGuides = false
	nup.MultiFolio = false
	nup.FolioSize = 8
	nup.BookletType = Booklet
	nup.BookletBinding = longEdge
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
	if err := ParseNUpValue(val, nup); err != nil {
		return nil, err
	}
	if !(val == 2 || val == 4 || val == 6 || val == 8) {
		return nup, errInvalidBookletGridID
	}
	// 6up and 8up special cases
	if nup.isBooklet() && val > 4 && nup.isTopFoldBinding() {
		// you can't top fold a 6up with 3 rows
		// TODO: support this for 8up
		return nup, fmt.Errorf("pdfcpu booklet: n>4 must have binding on side (portrait long-edge or landscape short-edge)")
	}
	// covers
	if nup.BookletType == BookletCover && !(val == 2 || val == 4) {
		return nup, errInvalidBookletCoverGridID
	}
	if nup.BookletType == BookletCoverFullSpan && val != 2 {
		return nup, errInvalidBookletCoverFullGridID
	}
	// bookletadvanced
	if nup.BookletType == BookletAdvanced && val == 4 && nup.isTopFoldBinding() {
		return nup, errInvalidBookletAdvanced
	}
	return nup, nil
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

func drawScissors(w io.Writer, mb *Rectangle, fm FontMap) {
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
		DrawLineSimple(w, 0, height/2, width, height/2)
		drawGuideLineLabel(w, 1, height/2+2, "Fold here", mb, fm, 0)
	case 4:
		// Draw vertical folding line.
		fmt.Fprint(w, "[3] 0 d ")
		DrawLineSimple(w, width/2, 0, width/2, height)
		drawGuideLineLabel(w, width/2-23, 20, "Fold here", mb, fm, 90)

		// Draw horizontal cutting line.
		fmt.Fprint(w, "[3] 0 d ")
		DrawLineSimple(w, 0, height/2, width, height/2)
		drawGuideLineLabel(w, width, height/2+2, "Fold & Cut here", mb, fm, 0)

		// Draw scissors over cutting line.
		drawScissors(w, mb, fm)
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

func nup4OutputPageNr(inputPageNr int, pageCount int, pageNumbers []int, nup *NUp) (int, bool) {
	switch nup.BookletType {
	case Booklet:
		// simple booklets are collated by collecting the top of the sheet, then the bottom, then the top of the next sheet, and so on.
		// this is conceptually easier for collation without specialized tools.
		if nup.isTopFoldBinding() {
			return nup4BasicTopFoldOutputPageNr(inputPageNr, pageCount, pageNumbers, nup)
		} else {
			return nup4BasicSideFoldOutputPageNr(inputPageNr, pageCount, pageNumbers, nup)
		}
	case BookletAdvanced:
		// advanced booklets have a different collation pattern: collect the top of each sheet and then the bottom of each sheet.
		// this allows printers to fold the sheets twice and then cut along one of the folds.
		return nup4AdvancedSideFoldOutputPageNr(inputPageNr, pageCount, pageNumbers, nup)
	}
	return 0, false
}

func nup4BasicSideFoldOutputPageNr(positionNumber int, inputPageCount int, pageNumbers []int, nup *NUp) (int, bool) {
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

func nup4BasicTopFoldOutputPageNr(positionNumber int, inputPageCount int, pageNumbers []int, nup *NUp) (int, bool) {
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

func nup4AdvancedSideFoldOutputPageNr(inputPageNr int, inputPageCount int, pageNumbers []int, nup *NUp) (int, bool) {
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

func nupLRTBOutputPageNr(positionNumber int, inputPageCount int, pageNumbers []int, nup *NUp) (int, bool) {
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

func nup8OutputPageNr(portraitPositionNumber int, inputPageCount int, pageNumbers []int, nup *NUp) (int, bool) {
	// 8up sheet has four rows and two columns
	// but the spreads are NOT across the two columns - instead the spreads are rotated 90deg to fit in a portrait orientation on the sheet
	// rather than coding up an entire new imposition, we're going to use the left-down-top-bottom imposition as a base
	// and the rotate the spreads (ie reorder) to fit on the sheet

	bookletSheetSideNumber := portraitPositionNumber / 8
	var landscapePositionNumber int
	switch bookletSheetSideNumber % 2 {
	case 0: // front side
		// rotate the block of four pages 90deg clockwise to go from portrait to landscape
		switch portraitPositionNumber % 4 {
		case 0:
			landscapePositionNumber = 1
		case 1:
			landscapePositionNumber = 3
		case 2:
			landscapePositionNumber = 0
		case 3:
			landscapePositionNumber = 2
		}
	case 1: // back side
		// rotate the block of four pages 90deg anti-clockwise to go from portrait to landscape
		// this is different from the front side because of the non-duplex sheet handling flip along the short edge
		switch portraitPositionNumber % 4 {
		case 0:
			landscapePositionNumber = 2
		case 1:
			landscapePositionNumber = 0
		case 2:
			landscapePositionNumber = 3
		case 3:
			landscapePositionNumber = 1
		}

	}
	positionNumber := landscapePositionNumber + portraitPositionNumber/4*4
	return nupLRTBOutputPageNr(positionNumber, inputPageCount, pageNumbers, nup)
}

func nupPerfectBound(positionNumber int, inputPageCount int, pageNumbers []int, nup *NUp) (int, bool) {
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

	switch nup.BookletType {
	case Booklet, BookletAdvanced:
		switch nup.N() {
		case 2:
			// (output page, input page) = [(1,n), (2,1), (3, n-1), (4, 2), (5, n-2), (6, 3), ...]
			for i := 0; i < pageCount; i++ {
				pageNr, rotate := nup2OutputPageNr(i, pageCount, pageNumbers)
				bookletPages[i].number = pageNr
				bookletPages[i].rotate = rotate
			}

		case 4:
			for i := 0; i < pageCount; i++ {
				pageNr, rotate := nup4OutputPageNr(i, pageCount, pageNumbers, nup)
				bookletPages[i].number = pageNr
				bookletPages[i].rotate = rotate
			}
		case 6:
			for i := 0; i < pageCount; i++ {
				pageNr, rotate := nupLRTBOutputPageNr(i, pageCount, pageNumbers, nup)
				bookletPages[i].number = pageNr
				bookletPages[i].rotate = rotate
			}
		case 8:
			for i := 0; i < pageCount; i++ {
				pageNr, rotate := nup8OutputPageNr(i, pageCount, pageNumbers, nup)
				bookletPages[i].number = pageNr
				bookletPages[i].rotate = rotate
			}
		}
	case BookletPerfectBound:
		for i := 0; i < pageCount; i++ {
			pageNr, rotate := nupPerfectBound(i, pageCount, pageNumbers, nup)
			bookletPages[i].number = pageNr
			bookletPages[i].rotate = rotate
		}
	case BookletCover:
		// covers can have either one (front) or two pages (front and back).
		// the front cover is on the right of the sheet (and the back on the left).
		// the bottom row should be rotated 180deg so that the cuts are always along the bottom of the booklet
		// the bottom row is also in the opposite page order
		switch nup.N() {
		case 2:
			// we are printing one cover per sheet
			switch len(pages) {
			case 1:
				// there is no back cover - just front (top/right of sheet)
				bookletPages = []bookletPage{{1, false}, {0, false}}
			case 2:
				// the cover has a front (top/right of sheet) and back (bottom/left of sheet)
				bookletPages = []bookletPage{{1, false}, {2, false}}
			}
		case 4:
			// we are printing two covers per sheet
			switch len(pages) {
			case 1:
				// there is no back cover
				bookletPages = []bookletPage{{0, false}, {1, false}, {1, true}, {0, true}}
			case 2:
				// the cover has a front and back
				if nup.isTopFoldBinding() {
					// the cover and it's back should both have top sides up
					bookletPages = []bookletPage{{2, true}, {1, true}, {1, false}, {2, false}}
				} else {
					bookletPages = []bookletPage{{2, false}, {1, false}, {1, true}, {2, true}}
				}
			}
		}
	case BookletCoverFullSpan:
		switch nup.N() {
		case 2:
			// we are printing two covers per sheet. full-span covers have just one pdf input, that spans front and back of the booklet
			// the bottom row should be rotated 180deg so that the cuts are always along the bottom of the booklet
			switch len(pages) {
			case 1:
				bookletPages = []bookletPage{{1, false}, {1, true}}
			}
		}
	}

	return bookletPages
}

func (ctx *Context) bookletPages(
	selectedPages IntSet,
	nup *NUp,
	pagesDict Dict,
	pagesIndRef *IndirectRef) error {

	var buf bytes.Buffer
	formsResDict := NewDict()
	rr := rectsForGrid(nup)

	if nup.BookletType == BookletCover {
		if len(selectedPages) > 2 {
			return fmt.Errorf("booklet covers must be either one or two pages")
		}
	}
	if nup.BookletType == BookletCoverFullSpan {
		if len(selectedPages) != 1 {
			return fmt.Errorf("booklet cover full span must have just one page")
		}
	}

	for i, bp := range sortSelectedPagesForBooklet(selectedPages, nup) {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}
			buf.Reset()
			formsResDict = NewDict()
		}

		rDest := rr[i%len(rr)]

		if bp.number == 0 {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		if err := ctx.nUpTilePDFBytesForPDF(bp.number, formsResDict, &buf, rDest, nup, bp.rotate); err != nil {
			return err
		}
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// BookletFromPDF creates a booklet version of the PDF represented by xRefTable.
func (ctx *Context) BookletFromPDF(selectedPages IntSet, nup *NUp) error {
	n := int(nup.Grid.Width * nup.Grid.Height)
	if !(n == 2 || n == 4 || n == 6 || n == 8) {
		return fmt.Errorf("pdfcpu: booklet must have n={2,4,6,8} pages per sheet, got %d", n)
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

	if nup.MultiFolio {
		pages := IntSet{}
		for _, i := range sortSelectedPages(selectedPages) {
			pages[i] = true
			if len(pages) == 4*nup.FolioSize {
				if err = ctx.bookletPages(pages, nup, pagesDict, pagesIndRef); err != nil {
					return err
				}
				pages = IntSet{}
			}
		}
		if len(pages) > 0 {
			if err = ctx.bookletPages(pages, nup, pagesDict, pagesIndRef); err != nil {
				return err
			}
		}

	} else {
		if err = ctx.bookletPages(selectedPages, nup, pagesDict, pagesIndRef); err != nil {
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

			// Wrap complete page.
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
				FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileNames[bp.number-1])
		if err != nil {
			return err
		}

		imgIndRef, w, h, err := CreateImageResource(xRefTable, f, false, false)
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
		nUpTilePDFBytes(&buf, RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, bp.rotate, enforceOrientation)
	}

	// Wrap incomplete booklet page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

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
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Booklet represents the command details for the command "Booklet".
type Booklet struct {
	PageSize  string      // Page size eg. A4L, A4P, A4(=default=A4P), see paperSize.go
	PageDim   *Dim        // Dimensions of the page of a booklet (and input files)
	SheetSize string      // Sheet size eg. A4L, A4P, A4(=default=A4P), see paperSize.go
	SheetDim  *Dim        // Dimensions of the sheet of paper printed (and output files)
	Margin    int         // Cropbox for booklet content within sheet
	InpUnit   DisplayUnit // input display unit.
}

// BookletConfig returns an Booklet configuration
func BookletConfig(desc string) (*Booklet, error) {
	b := DefaultBookletConfig()
	err := ParseBookletDetails(desc, b)
	return b, err
}

// DefaultBookletConfig returns the default NUp configuration.
func DefaultBookletConfig() *Booklet {
	return &Booklet{
		PageSize:  "A5",
		SheetSize: "A3",
		Margin:    0,
	}
}

func (b Booklet) String() string {
	return fmt.Sprintf("Booklet conf: input=%s %s, output=%s %s\n",
		b.PageSize, *b.PageDim, b.SheetSize, b.SheetDim)
}

// ParseBookletDetails parses a Booklet command string into an internal structure.
func ParseBookletDetails(s string, booklet *Booklet) error {
	err1 := errInvalidNUpConfig
	if s == "" {
		return err1
	}

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return err1
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := bookletParamMap.Handle(paramPrefix, paramValueStr, booklet); err != nil {
			return err
		}
	}

	return nil
}

type bkltParamMap map[string]func(string, *Booklet) error

// Handle applies parameter completion and if successful
// parses the parameter values into import.
func (m bkltParamMap) Handle(paramPrefix, paramValueStr string, booklet *Booklet) error {
	var param string

	// Completion support
	for k := range m {
		if !strings.HasPrefix(k, paramPrefix) {
			continue
		}
		if len(param) > 0 {
			return errors.Errorf("pdfcpu: ambiguous parameter prefix \"%s\"", paramPrefix)
		}
		param = k
	}

	if param == "" {
		return errors.Errorf("pdfcpu: ambiguous parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, booklet)
}

var bookletParamMap = bkltParamMap{
	"pagesize":  parseBookletPageSize,
	"sheetsize": parseBookletSheetSize,
	"margin":    parseBookletMargin,
}

func parseBookletPageSize(s string, b *Booklet) (err error) {
	b.PageDim, b.PageSize, err = parsePageFormat(s)
	return err
}

func parseBookletSheetSize(s string, b *Booklet) (err error) {
	b.SheetDim, b.SheetSize, err = parsePageFormat(s)
	return err
}

func parseBookletMargin(s string, b *Booklet) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}

	if f < 0 {
		return errors.New("pdfcpu: booklet margin, Please provide a positive value")
	}

	b.Margin = int(toUserSpace(f, b.InpUnit))

	return nil
}

// BookletFromPDF creates an booklet version of the PDF represented by xRefTable.
func (ctx *Context) BookletFromPDF(selectedPages IntSet, booklet *Booklet) error {
	grid := &Dim{
		math.Round(booklet.SheetDim.Width / booklet.PageDim.Width),
		math.Round(booklet.SheetDim.Height / booklet.PageDim.Height),
	}
	n := int(grid.Width * grid.Height)
	if !(n == 2 || n == 4) {
		return fmt.Errorf("booklet must have page and sheet dimensions that result in 2 or 4 pages per sheet, got %d", n)
	}
	// TODO: we're assuming that inputs are really booklet.PageSize
	nup := &NUp{
		PageDim:  booklet.SheetDim,
		PageSize: booklet.SheetSize,
		Orient:   RightDown,
		Grid:     grid,
		Margin:   booklet.Margin,
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

	switch nup.Grid.Height * nup.Grid.Width {
	case 2:
		nPages := len(pageNumbers)
		if nPages%2 != 0 {
			// nPages must be a multiple of 2
			// if not, we will insert a blank page at the end
			nPages++
		}
		out := make([]int, nPages)
		shouldRotate := make([]bool, nPages)
		// (output page, input page) = [(1,1), (2,n), (3, 2), (4, n-1), (5, 3), (6, n-2), ...]
		for i := 0; i < nPages; i++ {
			if i%2 == 0 {
				out[i] = getPageNumber(pageNumbers, i/2)
			} else {
				out[i] = getPageNumber(pageNumbers, nPages-1-(i-1)/2)
			}
			// odd output pages should be upside-down
			if i%4 < 2 {
				shouldRotate[i] = true
			}
		}
		return out, shouldRotate
	case 4:
		nPages := len(pageNumbers)
		rem := nPages % 8
		if rem != 0 {
			// nPages must be a multiple of 8
			// if not, we will insert blank pages at the end
			nPages += 8 - rem
		}
		out := make([]int, nPages)
		shouldRotate := make([]bool, nPages)
		// (output page, input page) = [
		// (1,n), (2,1), (3, n/2+1), (4, n/2-0),
		// (5, 2), (6, n-1), (7, n/2-1), (8, n/2+2)
		// ...]
		for i := 0; i < len(pageNumbers); i++ {
			bookletPageNumber := i / 4
			if bookletPageNumber%2 == 0 {
				// front side
				switch i % 4 {
				case 0:
					out[i] = getPageNumber(pageNumbers, nPages-1-bookletPageNumber)
				case 1:
					out[i] = getPageNumber(pageNumbers, bookletPageNumber)
				case 2:
					out[i] = getPageNumber(pageNumbers, nPages/2+bookletPageNumber)
				case 3:
					out[i] = getPageNumber(pageNumbers, nPages/2-1-bookletPageNumber)
				}
			} else {
				// back side
				switch i % 4 {
				case 0:
					out[i] = getPageNumber(pageNumbers, bookletPageNumber)
				case 1:
					out[i] = getPageNumber(pageNumbers, nPages-1-bookletPageNumber)
				case 2:
					out[i] = getPageNumber(pageNumbers, nPages/2-1-bookletPageNumber)
				case 3:
					out[i] = getPageNumber(pageNumbers, nPages/2+bookletPageNumber)
				}
			}
			if i%4 >= 2 {
				// bottom row of each output page should be rotated
				shouldRotate[i] = true
			}
		}
		return out, shouldRotate
	}

	return pageNumbers, nil
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
		nUpTilePDFBytes(&buf, cropBox, rr[i%len(rr)], formResID, nup, shouldRotatePage[i])
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

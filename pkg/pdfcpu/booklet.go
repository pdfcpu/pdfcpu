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
	"fmt"
	"math"
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
		Orient:   bookletOrient,
		Grid:     grid,
		Margin:   booklet.Margin,
	}
	return ctx.NUpFromPDF(selectedPages, nup)
}

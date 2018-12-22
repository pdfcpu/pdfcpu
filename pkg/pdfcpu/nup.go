/*
Copyright 2018 The pdfcpu Authors.

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
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/pkg/errors"
)

var (
	errInvalidNUpConfig = errors.New("Invalid nup configuration string. Please consult pdfcpu help nup")
	errInvalidGridID    = errors.New("grid definition n: one of 2, 4, 9, 16")
	errInvalidGridDef   = errors.New("grid definition n: one of 2, 4, 9, 16 or mxn")
)

type orientation int

func (o orientation) String() string {

	switch o {

	case RightDown:
		return "right down"

	case DownRight:
		return "down right"

	case LeftDown:
		return "left down"

	case DownLeft:
		return "down left"

	}

	return ""
}

// These are the defined anchors for relative positioning.
const (
	RightDown orientation = iota
	DownRight
	LeftDown
	DownLeft
)

func parseOrientation(s string) (orientation, error) {

	switch s {
	case "rd":
		return RightDown, nil
	case "dr":
		return DownRight, nil
	case "ld":
		return LeftDown, nil
	case "dl":
		return DownLeft, nil
	}

	return 0, errors.Errorf("unknown nUp orientation: %s", s)
}

// NUp represents the command details for the command "NUp".
type NUp struct {
	PageSize         string      // one of A0,A1,A2,A3,A4(=default),A5,A6,A7,A8,Letter,Legal,Ledger,Tabloid,Executive,ANSIC,ANSID,ANSIE.
	PageDim          dim         // page dimensions in user units.
	Orient           orientation // one of rd(=default),dr,ld,dl
	Grid             dim         // grid dimensions eg (2,2)
	PreservePageSize bool        // for PDF inputfiles only
	ImgInputFile     bool
}

func (nup NUp) String() string {

	return fmt.Sprintf("N-Up config: %s %s, orient=%s, grid=%s, preserverPageSize=%t, isImage=%t\n",
		nup.PageSize, nup.PageDim, nup.Orient, nup.Grid, nup.PreservePageSize, nup.ImgInputFile)
}

// DefaultNUpConfig returns the default configuration.
func DefaultNUpConfig() *NUp {
	return &NUp{
		PageDim:  PaperSize["A4"], // for image input file
		PageSize: "A4",            // for image input file
		Orient:   RightDown,       // for pdf input file
	}
}

// ParseNUpGridDefinition parses NUp grid dimensions into an internal structure.
func ParseNUpGridDefinition(s string, nUp *NUp) error {

	n, err := strconv.Atoi(s)
	if err != nil || !IntMemberOf(n, []int{2, 4, 9, 16}) {
		if nUp.ImgInputFile {
			return errInvalidGridID
		}
		ss := strings.Split(s, "x")
		if len(ss) != 2 {
			return errInvalidGridDef
		}
		m, err := strconv.Atoi(ss[0])
		if err != nil {
			return errInvalidGridDef
		}
		n, err := strconv.Atoi(ss[1])
		if err != nil {
			return errInvalidGridDef
		}
		nUp.Grid = dim{m, n}
		return nil
	}

	var d dim
	switch n {
	case 2:
		d = dim{1, 2}
	case 4:
		d = dim{2, 2}
	case 9:
		d = dim{3, 3}
	case 16:
		d = dim{4, 4}
	}
	nUp.Grid = d
	nUp.PreservePageSize = true

	return nil
}

// ParseNUpDetails parses a NUp command string into an internal structure.
func ParseNUpDetails(s string, nup *NUp) error {

	if s == "" {
		return nil
	}

	ss := strings.Split(s, ",")

	var setDim, setFormat bool

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return errInvalidNUpConfig
		}

		k := strings.TrimSpace(ss1[0])
		v := strings.TrimSpace(ss1[1])
		//fmt.Printf("key:<%s> value<%s>\n", k, v)

		var err error

		switch k {
		case "f": // page format
			nup.PageDim, nup.PageSize, err = parsePageFormat(v, setDim)
			setFormat = true

		case "d": // page dimensions
			nup.PageDim, nup.PageSize, err = parsePageDim(v, setFormat)
			setDim = true

		case "o": // offset
			nup.Orient, err = parseOrientation(v)

		default:
			err = errInvalidNUpConfig
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func nUpImagePDFBytes(wr io.Writer, imgWidth, imgHeight float64, nup *NUp) {

}

// NewNUpPageForImage creates a new page dict in xRefTable for given image filename and n-up config.
func NewNUpPageForImage(xRefTable *XRefTable, fileName string, parentIndRef *IndirectRef, nup *NUp) (*IndirectRef, error) {

	// create image dict.
	imgIndRef, w, h, err := createImageResource(xRefTable, fileName)
	if err != nil {
		return nil, err
	}

	// create resource dict for XObject.
	d := Dict(
		map[string]Object{
			"ProcSet": NewNameArray("PDF", "ImageB", "ImageC", "ImageI"),
			"XObject": Dict(map[string]Object{"Im0": *imgIndRef}),
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create content stream for Im0.
	contents := &StreamDict{Dict: NewDict()}
	contents.InsertName("Filter", filter.Flate)
	contents.FilterPipeline = []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}

	var buf bytes.Buffer
	nUpImagePDFBytes(&buf, float64(w), float64(h), nup)
	contents.Content = buf.Bytes()

	err = encodeStream(contents)
	if err != nil {
		return nil, err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*contents)
	if err != nil {
		return nil, err
	}

	// mediabox = physical page dimensions
	dim := nup.PageDim
	mediaBox := NewRectangle(0, 0, float64(dim.w), float64(dim.h))

	pageDict := Dict(
		map[string]Object{
			"Type":      Name("Page"),
			"Parent":    *parentIndRef,
			"MediaBox":  mediaBox,
			"Resources": *resIndRef,
			"Contents":  *contentsIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(pageDict)
}

// NUpFromImage creates a single page n-up PDF of an image.
func NUpFromImage(config *Configuration, imageFileName string, nup *NUp) (*Context, error) {

	ctx, err := CreateContextWithXRefTable(config, nup.PageDim)
	if err != nil {
		return nil, err
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, err
	}

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, err
	}

	indRef, err := NewNUpPageForImage(ctx.XRefTable, imageFileName, pagesIndRef, nup)
	if err != nil {
		return nil, err
	}

	err = AppendPageTree(indRef, 1, &pagesDict)
	if err != nil {
		return nil, err
	}

	ctx.PageCount++

	return ctx, nil
}

// NUpFromPDF creates an n-up version of the PDF represented by xRefTable.
func NUpFromPDF(xRefTable *XRefTable, nup *NUp) error {

	return nil
}

// ConstantPageDimension returns true if all pages use the same dimensions.
func ConstantPageDimension(xRefTable *XRefTable) (bool, error) {

	var d *dim

	for i := 0; i < xRefTable.PageCount; i++ {

		pageDim, err := xRefTable.PageDim(i)
		if err != nil {
			return false, err
		}

		if d == nil {
			d = &pageDim
			continue
		}

		if *d != pageDim {
			return false, nil
		}

	}

	return true, nil
}

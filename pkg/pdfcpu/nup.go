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
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pkg/errors"
)

var (
	errInvalidGridID     = errors.New("pdfcpu: nup: n: one of 2, 3, 4, 6, 8, 9, 12, 16")
	errInvalidGridDims   = errors.New("pdfcpu: grid: dimensions: m >= 0, n >= 0")
	errInvalidNUpConfig  = errors.New("pdfcpu: nup: invalid configuration string. Please consult pdfcpu help nup")
	errInvalidGridConfig = errors.New("pdfcpu: nup: invalid configuration string. Please consult pdfcpu help grid")
)

var (
	nUpValues = []int{2, 3, 4, 6, 8, 9, 12, 16}
	nUpDims   = map[int]Dim{
		2:  {2, 1},
		3:  {3, 1},
		4:  {2, 2},
		6:  {3, 2},
		8:  {4, 2},
		9:  {3, 3},
		12: {4, 3},
		16: {4, 4},
	}
)

type nUpParamMap map[string]func(string, *NUp) error

var nupParamMap = nUpParamMap{
	"dimensions":  parseDimensionsNUp,
	"formsize":    parsePageFormatNUp,
	"papersize":   parsePageFormatNUp,
	"orientation": parseOrientation,
	"border":      parseElementBorder,
	"margin":      parseElementMargin,
}

// Handle applies parameter completion and if successful
// parses the parameter values into import.
func (m nUpParamMap) Handle(paramPrefix, paramValueStr string, nup *NUp) error {
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

	return m[param](paramValueStr, nup)
}

// NUp represents the command details for the command "NUp".
type NUp struct {
	PageDim      *Dim        // Page dimensions in display unit.
	PageSize     string      // Paper size eg. A4L, A4P, A4(=default=A4P), see paperSize.go
	UserDim      bool        // true if one of dimensions or paperSize provided overriding the default.
	Orient       orientation // One of rd(=default),dr,ld,dl
	Grid         *Dim        // Intra page grid dimensions eg (2,2)
	PageGrid     bool        // Create a mxn grid of pages for PDF inputfiles only (think "extra page n-Up").
	ImgInputFile bool        // Process image or PDF input files.
	Margin       int         // Cropbox for n-Up content.
	Border       bool        // Draw bounding box.
	InpUnit      DisplayUnit // input display unit.
}

// DefaultNUpConfig returns the default NUp configuration.
func DefaultNUpConfig() *NUp {
	return &NUp{
		PageSize: "A4",
		Orient:   RightDown,
		Margin:   3,
		Border:   true,
	}
}

func (nup NUp) String() string {
	return fmt.Sprintf("N-Up conf: %s %s, orient=%s, grid=%s, pageGrid=%t, isImage=%t\n",
		nup.PageSize, *nup.PageDim, nup.Orient, *nup.Grid, nup.PageGrid, nup.ImgInputFile)
}

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
	bookletOrient // this is not an option that can be used from the cli - use pdfcpu booklet command instead
)

func parsePageFormatNUp(s string, nup *NUp) (err error) {
	if nup.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	nup.PageDim, nup.PageSize, err = parsePageFormat(s)
	nup.UserDim = true
	return err
}

func parseDimensionsNUp(s string, nup *NUp) (err error) {
	if nup.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	nup.PageDim, nup.PageSize, err = parsePageDim(s, nup.InpUnit)
	nup.UserDim = true
	return err
}

func parseOrientation(s string, nup *NUp) error {
	switch s {
	case "rd":
		nup.Orient = RightDown
	case "dr":
		nup.Orient = DownRight
	case "ld":
		nup.Orient = LeftDown
	case "dl":
		nup.Orient = DownLeft
	default:
		return errors.Errorf("pdfcpu: unknown nUp orientation: %s", s)
	}

	return nil
}

func parseElementBorder(s string, nup *NUp) error {
	switch strings.ToLower(s) {
	case "on", "true":
		nup.Border = true
	case "off", "false":
		nup.Border = false
	default:
		return errors.New("pdfcpu: nUp border, please provide one of: on/off true/false")
	}

	return nil
}

func parseElementMargin(s string, nup *NUp) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}

	if f < 0 {
		return errors.New("pdfcpu: nUp margin, Please provide a positive value")
	}

	nup.Margin = int(toUserSpace(f, nup.InpUnit))

	return nil
}

// ParseNUpDetails parses a NUp command string into an internal structure.
func ParseNUpDetails(s string, nup *NUp) error {
	err1 := errInvalidNUpConfig
	if nup.PageGrid {
		err1 = errInvalidGridConfig
	}

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

		if err := nupParamMap.Handle(paramPrefix, paramValueStr, nup); err != nil {
			return err
		}
	}

	return nil
}

// PDFNUpConfig returns an NUp configuration for Nup-ing PDF files.
func PDFNUpConfig(val int, desc string) (*NUp, error) {
	nup := DefaultNUpConfig()
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpValue(val, nup)
}

// ImageNUpConfig returns an NUp configuration for Nup-ing image files.
func ImageNUpConfig(val int, desc string) (*NUp, error) {
	nup, err := PDFNUpConfig(val, desc)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// PDFGridConfig returns a grid configuration for Nup-ing PDF files.
func PDFGridConfig(rows, cols int, desc string) (*NUp, error) {
	nup := DefaultNUpConfig()
	nup.PageGrid = true
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpGridDefinition(rows, cols, nup)
}

// ImageGridConfig returns a grid configuration for Nup-ing image files.
func ImageGridConfig(rows, cols int, desc string) (*NUp, error) {
	nup, err := PDFGridConfig(rows, cols, desc)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// ParseNUpValue parses the NUp value into an internal structure.
func ParseNUpValue(n int, nUp *NUp) error {
	if !IntMemberOf(n, nUpValues) {
		return errInvalidGridID
	}

	// The n-Up layout depends on the orientation of the chosen output paper size.
	// This optional paper size may also be specified by dimensions in user unit.
	// The default paper size is A4 or A4P (A4 in portrait mode) respectively.
	var portrait bool
	if nUp.PageDim == nil {
		portrait = PaperSize[nUp.PageSize].Portrait()
	} else {
		portrait = RectForDim(nUp.PageDim.Width, nUp.PageDim.Height).Portrait()
	}

	d := nUpDims[n]
	if portrait {
		d.Width, d.Height = d.Height, d.Width
	}

	nUp.Grid = &d

	return nil
}

// ParseNUpGridDefinition parses NUp grid dimensions into an internal structure.
func ParseNUpGridDefinition(rows, cols int, nUp *NUp) error {
	m := cols
	if m <= 0 {
		return errInvalidGridDims
	}

	n := rows
	if m <= 0 {
		return errInvalidGridDims
	}

	nUp.Grid = &Dim{float64(m), float64(n)}

	return nil
}

func rectsForGrid(nup *NUp) []*Rectangle {
	cols := int(nup.Grid.Width)
	rows := int(nup.Grid.Height)

	maxX := float64(nup.PageDim.Width)
	maxY := float64(nup.PageDim.Height)

	gw := maxX / float64(cols)
	gh := maxY / float64(rows)

	var llx, lly float64
	rr := []*Rectangle{}

	switch nup.Orient {

	case RightDown, bookletOrient:
		for i := rows - 1; i >= 0; i-- {
			for j := 0; j < cols; j++ {
				llx = float64(j) * gw
				lly = float64(i) * gh
				rr = append(rr, Rect(llx, lly, llx+gw, lly+gh))
			}
		}

	case DownRight:
		for i := 0; i < cols; i++ {
			for j := rows - 1; j >= 0; j-- {
				llx = float64(i) * gw
				lly = float64(j) * gh
				rr = append(rr, Rect(llx, lly, llx+gw, lly+gh))
			}
		}

	case LeftDown:
		for i := rows - 1; i >= 0; i-- {
			for j := cols - 1; j >= 0; j-- {
				llx = float64(j) * gw
				lly = float64(i) * gh
				rr = append(rr, Rect(llx, lly, llx+gw, lly+gh))
			}
		}

	case DownLeft:
		for i := cols - 1; i >= 0; i-- {
			for j := rows - 1; j >= 0; j-- {
				llx = float64(i) * gw
				lly = float64(j) * gh
				rr = append(rr, Rect(llx, lly, llx+gw, lly+gh))
			}
		}
	}

	return rr
}

// Calculate the matrix for transforming rectangle r1 with lower left corner in the origin into rectangle r2.
func calcTransMatrixForRect(r1, r2 *Rectangle, image bool, rotatePageForBooklet bool) matrix {
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
	r := float64(rot)
	if rotatePageForBooklet {
		// booklet pages are rotated 180deg, in addition to any aspect rotation
		r = math.Mod(rot+180, 360)
	}
	m2 := identMatrix
	sin := math.Sin(float64(r) * float64(degToRad))
	cos := math.Cos(float64(r) * float64(degToRad))
	m2[0][0] = cos
	m2[0][1] = sin
	m2[1][0] = -sin
	m2[1][1] = cos

	// Translate
	m3 := identMatrix
	m3[2][0] = dx
	m3[2][1] = dy

	m := m1.multiply(m2).multiply(m3)
	if rotatePageForBooklet {
		// we've rotated 180deg, so we need to translate to get the old page visiible on the new page
		if rot == 0 {
			m[2][0] += w * r1.Width()
			m[2][1] += h * r1.Height()
		} else {
			// 90 degree rotation because of paper, in addition to the 180 rotation for booklet
			// need to shift rotated page on sheet down and left
			m[2][0] -= h * r1.Width()
			m[2][1] += w * r1.Height()
		}
	}
	return m
}

func nUpTilePDFBytes(wr io.Writer, r1, r2 *Rectangle, formResID string, nup *NUp, rotatePageForBooklet bool) {
	// Draw bounding box.
	if nup.Border {
		fmt.Fprintf(wr, "[]0 d 0.1 w %.2f %.2f m %.2f %.2f l %.2f %.2f l %.2f %.2f l s ",
			r2.LL.X, r2.LL.Y, r2.UR.X, r2.LL.Y, r2.UR.X, r2.UR.Y, r2.LL.X, r2.UR.Y,
		)
	}

	// Apply margin.
	croppedRect := r2.CroppedCopy(float64(nup.Margin))

	m := calcTransMatrixForRect(r1, croppedRect, nup.ImgInputFile, rotatePageForBooklet)

	fmt.Fprintf(wr, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s Do Q ",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], formResID)
}

func nUpImagePDFBytes(wr io.Writer, imgWidth, imgHeight int, nup *NUp, formResID string) {
	for _, r := range rectsForGrid(nup) {
		nUpTilePDFBytes(wr, RectForDim(float64(imgWidth), float64(imgHeight)), r, formResID, nup, false)
	}
}

func createNUpForm(xRefTable *XRefTable, imgIndRef *IndirectRef, w, h, i int) (*IndirectRef, error) {
	imgResID := fmt.Sprintf("Im%d", i)
	bb := RectForDim(float64(w), float64(h))

	var b bytes.Buffer
	fmt.Fprintf(&b, "/%s Do ", imgResID)

	d := Dict(
		map[string]Object{
			"ProcSet": NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
			"XObject": Dict(map[string]Object{imgResID: *imgIndRef}),
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	sd := StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":      Name("XObject"),
				"Subtype":   Name("Form"),
				"BBox":      bb.Array(),
				"Matrix":    NewIntegerArray(1, 0, 0, 1, 0, 0),
				"Resources": *ir,
			},
		),
		Content:        b.Bytes(),
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if err = sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(sd)
}

func createNUpFormForPDFResource(xRefTable *XRefTable, resDict *IndirectRef, content []byte, cropBox *Rectangle) (*IndirectRef, error) {
	sd := StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":      Name("XObject"),
				"Subtype":   Name("Form"),
				"BBox":      cropBox.Array(),
				"Matrix":    NewIntegerArray(1, 0, 0, 1, 0, 0),
				"Resources": *resDict,
			},
		),
		Content:        content,
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(sd)
}

// NewNUpPageForImage creates a new page dict in xRefTable for given image filename and n-up conf.
func NewNUpPageForImage(xRefTable *XRefTable, fileName string, parentIndRef *IndirectRef, nup *NUp) (*IndirectRef, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// create image dict.
	imgIndRef, w, h, err := createImageResource(xRefTable, f)
	if err != nil {
		return nil, err
	}

	resID := 0

	formIndRef, err := createNUpForm(xRefTable, imgIndRef, w, h, resID)
	if err != nil {
		return nil, err
	}

	formResID := fmt.Sprintf("Fm%d", resID)

	resourceDict := Dict(
		map[string]Object{
			"XObject": Dict(map[string]Object{formResID: *formIndRef}),
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(resourceDict)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	nUpImagePDFBytes(&buf, w, h, nup, formResID)
	sd, _ := xRefTable.NewStreamDictForBuf(buf.Bytes())
	if err = sd.Encode(); err != nil {
		return nil, err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	// mediabox = physical page dimensions
	dim := nup.PageDim
	mediaBox := RectForDim(dim.Width, dim.Height)

	pageDict := Dict(
		map[string]Object{
			"Type":      Name("Page"),
			"Parent":    *parentIndRef,
			"MediaBox":  mediaBox.Array(),
			"Resources": *resIndRef,
			"Contents":  *contentsIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(pageDict)
}

// NUpFromOneImage creates one page with instances of one image.
func NUpFromOneImage(ctx *Context, fileName string, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	indRef, err := NewNUpPageForImage(ctx.XRefTable, fileName, pagesIndRef, nup)
	if err != nil {
		return err
	}

	if err = AppendPageTree(indRef, 1, pagesDict); err != nil {
		return err
	}

	ctx.PageCount++

	return nil
}

func wrapUpPage(ctx *Context, nup *NUp, d Dict, buf bytes.Buffer, pagesDict Dict, pagesIndRef *IndirectRef) error {
	xRefTable := ctx.XRefTable

	resourceDict := Dict(
		map[string]Object{
			"XObject": d,
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(resourceDict)
	if err != nil {
		return err
	}

	sd, _ := xRefTable.NewStreamDictForBuf(buf.Bytes())
	if err = sd.Encode(); err != nil {
		return err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	// mediabox = physical page dimensions
	dim := nup.PageDim
	mediaBox := RectForDim(dim.Width, dim.Height)

	pageDict := Dict(
		map[string]Object{
			"Type":      Name("Page"),
			"Parent":    *pagesIndRef,
			"MediaBox":  mediaBox.Array(),
			"Resources": *resIndRef,
			"Contents":  *contentsIndRef,
		},
	)

	indRef, err := xRefTable.IndRefForNewObject(pageDict)
	if err != nil {
		return err
	}

	if err = AppendPageTree(indRef, 1, pagesDict); err != nil {
		return err
	}

	ctx.PageCount++

	return nil
}

// NUpFromMultipleImages creates pages in NUp-style rendering each image once.
func NUpFromMultipleImages(ctx *Context, fileNames []string, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	if nup.PageGrid {
		nup.PageDim.Width *= nup.Grid.Width
		nup.PageDim.Height *= nup.Grid.Height
	}

	xRefTable := ctx.XRefTable
	formsResDict := NewDict()
	var buf bytes.Buffer
	rr := rectsForGrid(nup)

	for i, fileName := range fileNames {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete nUp page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = NewDict()
		}

		f, err := os.Open(fileName)
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

		formIndRef, err := createNUpForm(xRefTable, imgIndRef, w, h, i)
		if err != nil {
			return err
		}

		formResID := fmt.Sprintf("Fm%d", i)
		formsResDict.Insert(formResID, *formIndRef)

		nUpTilePDFBytes(&buf, RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, false)
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

func getPageNumber(pageNumbers []int, n int) int {
	if n >= len(pageNumbers) {
		// zero represents blank page at end of booklet
		return 0
	}
	return pageNumbers[n]
}

func sortedSelectedPages(pages IntSet, nup *NUp) ([]int, []bool) {
	var pageNumbers []int
	for k, v := range pages {
		if v {
			pageNumbers = append(pageNumbers, k)
		}
	}
	sort.Ints(pageNumbers)

	if nup.Orient == bookletOrient {
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
	}
	return pageNumbers, nil
}

func (ctx *Context) nupPages(selectedPages IntSet, nup *NUp, pagesDict Dict, pagesIndRef *IndirectRef) error {
	var buf bytes.Buffer
	xRefTable := ctx.XRefTable
	formsResDict := NewDict()
	rr := rectsForGrid(nup)

	pageNumbers, shouldRotatePage := sortedSelectedPages(selectedPages, nup)
	for i, p := range pageNumbers {

		if i > 0 && i%len(rr) == 0 {

			// Wrap complete nUp page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}

			buf.Reset()
			formsResDict = NewDict()
		}
		if p == 0 && nup.Orient == bookletOrient {
			// this is an empty page at the end of a bookletOrient
			continue
		}

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

		// inhPAttrs.mediaBox
		nUpTilePDFBytes(&buf, cropBox, rr[i%len(rr)], formResID, nup, nup.Orient == bookletOrient && shouldRotatePage[i])
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// NUpFromPDF creates an n-up version of the PDF represented by xRefTable.
func (ctx *Context) NUpFromPDF(selectedPages IntSet, nup *NUp) error {
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

	if err = ctx.nupPages(selectedPages, nup, pagesDict, pagesIndRef); err != nil {
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

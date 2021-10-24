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
	errInvalidGridID    = errors.New("pdfcpu nup: n must be one of 2, 3, 4, 6, 8, 9, 12, 16")
	errInvalidGridDims  = errors.New("pdfcpu grid: dimensions: m >= 0, n >= 0")
	errInvalidNUpConfig = errors.New("pdfcpu: invalid configuration string")
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
	"dimensions":      parseDimensionsNUp,
	"formsize":        parsePageFormatNUp,
	"papersize":       parsePageFormatNUp,
	"orientation":     parseOrientation,
	"border":          parseElementBorder,
	"margin":          parseElementMargin,
	"backgroundcolor": parseSheetBackgroundColor,
	"bgcolor":         parseSheetBackgroundColor,
	"guides":          parseBookletGuides,
	"multifolio":      parseBookletMultifolio,
	"foliosize":       parseBookletFolioSize,
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
	PageDim       *Dim         // Page dimensions in display unit.
	PageSize      string       // Paper size eg. A4L, A4P, A4(=default=A4P), see paperSize.go
	UserDim       bool         // true if one of dimensions or paperSize provided overriding the default.
	Orient        orientation  // One of rd(=default),dr,ld,dl
	Grid          *Dim         // Intra page grid dimensions eg (2,2)
	PageGrid      bool         // Create a m x n grid of pages for PDF inputfiles only (think "extra page n-Up").
	ImgInputFile  bool         // Process image or PDF input files.
	Margin        int          // Cropbox for n-Up content.
	Border        bool         // Draw bounding box.
	BookletGuides bool         // Draw folding and cutting lines.
	MultiFolio    bool         // Render booklet as sequence of folios.
	FolioSize     int          // Booklet multifolio folio size: default: 8
	InpUnit       DisplayUnit  // input display unit.
	BgColor       *SimpleColor // background color
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

// N returns the nUp value.
func (nup NUp) N() int {
	return int(nup.Grid.Height * nup.Grid.Width)
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
)

func parsePageFormatNUp(s string, nup *NUp) (err error) {
	if nup.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	nup.PageDim, nup.PageSize, err = ParsePageFormat(s)
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
	case "on", "true", "t":
		nup.Border = true
	case "off", "false", "f":
		nup.Border = false
	default:
		return errors.New("pdfcpu: nUp border, please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseBookletGuides(s string, nup *NUp) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		nup.BookletGuides = true
	case "off", "false", "f":
		nup.BookletGuides = false
	default:
		return errors.New("pdfcpu: booklet guides, please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseBookletMultifolio(s string, nup *NUp) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		nup.MultiFolio = true
	case "off", "false", "f":
		nup.MultiFolio = false
	default:
		return errors.New("pdfcpu: booklet guides, please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseBookletFolioSize(s string, nup *NUp) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("pdfcpu: illegal folio size: must be an numeric value, %s\n", s)
	}

	nup.FolioSize = i
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

func parseSheetBackgroundColor(s string, nup *NUp) error {
	c, err := ParseColor(s)
	if err != nil {
		return err
	}
	nup.BgColor = &c
	return nil
}

// ParseNUpDetails parses a NUp command string into an internal structure.
func ParseNUpDetails(s string, nup *NUp) error {
	if s == "" {
		return errInvalidNUpConfig
	}

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return errInvalidNUpConfig
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

	case RightDown:
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

func BestFitRectIntoRect(rSrc, rDest *Rectangle, enforceOrient bool) (w, h, dx, dy, rot float64) {
	if rSrc.FitsWithin(rDest) {
		// Translate rSrc into center of rDest without scaling.
		w = rSrc.Width()
		h = rSrc.Height()
		dx = rDest.Width()/2 - rSrc.Width()/2
		dy = rDest.Height()/2 - rSrc.Height()/2
		return
	}

	if rSrc.Landscape() {
		if rDest.Landscape() {
			if rSrc.AspectRatio() > rDest.AspectRatio() {
				w = rDest.Width()
				h = rSrc.ScaledHeight(w)
				dy = (rDest.Height() - h) / 2
			} else {
				h = rDest.Height()
				w = rSrc.ScaledWidth(h)
				dx = (rDest.Width() - w) / 2
			}
		} else {
			if enforceOrient {
				rot = 90
				if 1/rSrc.AspectRatio() < rDest.AspectRatio() {
					w = rDest.Height()
					h = rSrc.ScaledHeight(w)
					dx = (rDest.Width() - h) / 2
				} else {
					h = rDest.Width()
					w = rSrc.ScaledWidth(h)
					dy = (rDest.Height() - w) / 2
				}
				return
			}
			w = rDest.Width()
			h = rSrc.ScaledHeight(w)
			dy = (rDest.Height() - h) / 2
		}
		return
	}

	if rSrc.Portrait() {
		if rDest.Portrait() {
			if rSrc.AspectRatio() < rDest.AspectRatio() {
				h = rDest.Height()
				w = rSrc.ScaledWidth(h)
				dx = (rDest.Width() - w) / 2
			} else {
				w = rDest.Width()
				h = rSrc.ScaledHeight(w)
				dy = (rDest.Height() - h) / 2
			}
		} else {
			if enforceOrient {
				rot = 90
				if 1/rSrc.AspectRatio() > rDest.AspectRatio() {
					h = rDest.Width()
					w = rSrc.ScaledWidth(h)
					dy = (rDest.Height() - w) / 2
				} else {
					w = rDest.Height()
					h = rSrc.ScaledHeight(w)
					dx = (rDest.Width() - h) / 2
				}
				return
			}
			h = rDest.Height()
			w = rSrc.ScaledWidth(h)
			dx = (rDest.Width() - w) / 2
		}
		return
	}

	w = rDest.Height()
	if rDest.Portrait() {
		w = rDest.Width()
	}
	h = w
	dx = rDest.Width()/2 - rSrc.Width()/2
	dy = rDest.Height()/2 - rSrc.Height()/2

	return
}

func nUpTilePDFBytes(wr io.Writer, rSrc, rDest *Rectangle, formResID string, nup *NUp, rotate, enforceOrient bool) {

	// rScr is a rectangular region represented by form formResID in form space.

	// rDest is an arbitrary rectangular region in dest space.
	// It is the location where we want the form content to get rendered on a "best fit" basis.
	// Accounting for the aspect ratios of rSrc and rDest "best fit" tries to fit the largest version of rScr into rDest.
	// This may result in a 90 degree rotation.
	//
	// rotate:
	//			indicates if we need to apply a post rotation of 180 degrees eg for booklets.
	//
	// enforceOrient:
	//			indicates if we need to enforce dest's orientation.

	// Draw bounding box.
	if nup.Border {
		fmt.Fprintf(wr, "[]0 d 0.1 w %.2f %.2f m %.2f %.2f l %.2f %.2f l %.2f %.2f l s ",
			rDest.LL.X, rDest.LL.Y, rDest.UR.X, rDest.LL.Y, rDest.UR.X, rDest.UR.Y, rDest.LL.X, rDest.UR.Y,
		)
	}

	// Apply margin to rDest which potentially makes it smaller.
	rDestCr := rDest.CroppedCopy(float64(nup.Margin))

	// Calculate transform matrix.

	// Best fit translation of a source rectangle into a destination rectangle.
	// For nup we enforce the dest orientation,
	// whereas in cases where the original orientation needs to be preserved eg. for booklets, we don't.
	w, h, dx, dy, r := BestFitRectIntoRect(rSrc, rDestCr, enforceOrient)

	if nup.BgColor != nil {
		if nup.ImgInputFile {
			// Fill background.
			FillRectNoBorder(wr, rDest, *nup.BgColor)
		} else if nup.Margin > 0 {
			// Fill margins.
			m := float64(nup.Margin)
			drawMargins(wr, *nup.BgColor, rDest, 0, m, m, m, m)
		}
	}

	// Apply additional rotation.
	if rotate {
		r += 180
	}

	sx := w
	sy := h
	if !nup.ImgInputFile {
		sx /= rSrc.Width()
		sy /= rSrc.Height()
	}

	sin := math.Sin(r * float64(DegToRad))
	cos := math.Cos(r * float64(DegToRad))

	switch r {
	case 90:
		dx += h
	case 180:
		dx += w
		dy += h
	case 270:
		dy += w
	}

	dx += rDestCr.LL.X
	dy += rDestCr.LL.Y

	m := CalcTransformMatrix(sx, sy, sin, cos, dx, dy)

	// Apply transform matrix and display form.
	fmt.Fprintf(wr, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s Do Q ",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], formResID)
}

func nUpImagePDFBytes(w io.Writer, imgWidth, imgHeight int, nup *NUp, formResID string) {
	for _, r := range rectsForGrid(nup) {
		// Append to content stream.
		nUpTilePDFBytes(w, RectForDim(float64(imgWidth), float64(imgHeight)), r, formResID, nup, false, true)
	}
}

func createNUpFormForImage(xRefTable *XRefTable, imgIndRef *IndirectRef, w, h, i int) (*IndirectRef, error) {
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

func createNUpFormForPDF(xRefTable *XRefTable, resDict *IndirectRef, content []byte, cropBox *Rectangle) (*IndirectRef, error) {
	sd := StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":      Name("XObject"),
				"Subtype":   Name("Form"),
				"BBox":      cropBox.Array(),
				"Matrix":    NewNumberArray(1, 0, 0, 1, -cropBox.LL.X, -cropBox.LL.Y),
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
	imgIndRef, w, h, err := CreateImageResource(xRefTable, f, false, false)
	if err != nil {
		return nil, err
	}

	resID := 0

	formIndRef, err := createNUpFormForImage(xRefTable, imgIndRef, w, h, resID)
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

	var fm FontMap
	if nup.BookletGuides {
		// For booklets only.
		fm = drawBookletGuides(nup, &buf)
	}

	resourceDict := Dict(
		map[string]Object{
			"XObject": d,
		},
	)

	fontRes, err := fontResources(xRefTable, fm)
	if err != nil {
		return err
	}

	if len(fontRes) > 0 {
		resourceDict["Font"] = fontRes
	}

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

	// fileCount must be a multiple of n.
	// If not, we will insert blank pages at the end.
	fileCount := len(fileNames)
	if fileCount%nup.N() != 0 {
		fileCount += nup.N() - fileCount%nup.N()
	}

	for i := 0; i < fileCount; i++ {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete nUp page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}
			buf.Reset()
			formsResDict = NewDict()
		}

		rDest := rr[i%len(rr)]

		var fileName string
		if i < len(fileNames) {
			fileName = fileNames[i]
		}

		if fileName == "" {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileName)
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

		// Append to content stream of page i.
		nUpTilePDFBytes(&buf, RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, false, true)
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

func sortSelectedPages(pages IntSet) []int {
	var pageNumbers []int
	for k, v := range pages {
		if v {
			pageNumbers = append(pageNumbers, k)
		}
	}
	sort.Ints(pageNumbers)
	return pageNumbers
}

func nupPageNumber(i int, sortedPageNumbers []int) int {
	var pageNumber int
	if i < len(sortedPageNumbers) {
		pageNumber = sortedPageNumbers[i]
	}
	return pageNumber
}

func (ctx *Context) nUpTilePDFBytesForPDF(
	pageNr int,
	formsResDict Dict,
	buf *bytes.Buffer,
	rDest *Rectangle,
	nup *NUp,
	rotate bool) error {

	consolidateRes := true
	d, _, inhPAttrs, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.Errorf("pdfcpu: unknown page number: %d\n", pageNr)
	}

	// Retrieve content stream bytes.
	bb, err := ctx.PageContent(d)
	if err == errNoContent {
		// TODO render if has annotations.
		return nil
	}
	if err != nil {
		return err
	}

	// Create an object for this resDict in xRefTable.
	ir, err := ctx.IndRefForNewObject(inhPAttrs.Resources)
	if err != nil {
		return err
	}

	cropBox := inhPAttrs.MediaBox
	if inhPAttrs.CropBox != nil {
		cropBox = inhPAttrs.CropBox
	}

	// Account for existing rotation.
	if inhPAttrs.Rotate != 0 {
		if IntMemberOf(inhPAttrs.Rotate, []int{+90, -90, +270, -270}) {
			w := cropBox.Width()
			cropBox.UR.X = cropBox.LL.X + cropBox.Height()
			cropBox.UR.Y = cropBox.LL.Y + w
		}
		bb = append(contentBytesForPageRotation(inhPAttrs.Rotate, cropBox.Width(), cropBox.Height()), bb...)
	}

	formIndRef, err := createNUpFormForPDF(ctx.XRefTable, ir, bb, cropBox)
	if err != nil {
		return err
	}

	formResID := fmt.Sprintf("Fm%d", pageNr)
	formsResDict.Insert(formResID, *formIndRef)

	// Append to content stream buf of destination page.
	nUpTilePDFBytes(buf, cropBox, rDest, formResID, nup, rotate, true)

	return nil
}

func (ctx *Context) nupPages(
	selectedPages IntSet,
	nup *NUp,
	pagesDict Dict,
	pagesIndRef *IndirectRef) error {

	var buf bytes.Buffer
	formsResDict := NewDict()
	rr := rectsForGrid(nup)

	sortedPageNumbers := sortSelectedPages(selectedPages)
	pageCount := len(sortedPageNumbers)
	// pageCount must be a multiple of n.
	// If not, we will insert blank pages at the end.
	if pageCount%nup.N() != 0 {
		pageCount += nup.N() - pageCount%nup.N()
	}

	for i := 0; i < pageCount; i++ {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete page.
			if err := wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return err
			}
			buf.Reset()
			formsResDict = NewDict()
		}

		rDest := rr[i%len(rr)]

		pageNr := nupPageNumber(i, sortedPageNumbers)
		if pageNr == 0 {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		if err := ctx.nUpTilePDFBytesForPDF(pageNr, formsResDict, &buf, rDest, nup, false); err != nil {
			return err
		}
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// NUpFromPDF creates an n-up version of the PDF represented by xRefTable.
func (ctx *Context) NUpFromPDF(selectedPages IntSet, nup *NUp) error {
	var mb *Rectangle
	if nup.PageDim == nil {
		// No page dimensions specified, use cropBox of page 1 as mediaBox(=cropBox).
		consolidateRes := false
		d, _, inhPAttrs, err := ctx.PageDict(1, consolidateRes)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("unknown page number: %d\n", 1)
		}

		cropBox := inhPAttrs.MediaBox
		if inhPAttrs.CropBox != nil {
			cropBox = inhPAttrs.CropBox
		}

		// Account for existing rotation.
		if inhPAttrs.Rotate != 0 {
			if IntMemberOf(inhPAttrs.Rotate, []int{+90, -90, +270, -270}) {
				w := cropBox.Width()
				cropBox.UR.X = cropBox.LL.X + cropBox.Height()
				cropBox.UR.Y = cropBox.LL.Y + w
			}
		}

		mb = cropBox
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

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
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var (
	errInvalidNUpVal    = errors.New("pdfcpu nup: n must be one of 2, 3, 4, 6, 8, 9, 12, 16")
	errInvalidGridDims  = errors.New("pdfcpu grid: dimensions must be: m > 0, n > 0")
	errInvalidNUpConfig = errors.New("pdfcpu: invalid configuration string")
)

var (
	nUpValues = []int{2, 3, 4, 6, 8, 9, 12, 16}
	nUpDims   = map[int]types.Dim{
		2:  {Width: 2, Height: 1},
		3:  {Width: 3, Height: 1},
		4:  {Width: 2, Height: 2},
		6:  {Width: 3, Height: 2},
		8:  {Width: 4, Height: 2},
		9:  {Width: 3, Height: 3},
		12: {Width: 4, Height: 3},
		16: {Width: 4, Height: 4},
	}
)

type nUpParamMap map[string]func(string, *model.NUp) error

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
func (m nUpParamMap) Handle(paramPrefix, paramValueStr string, nup *model.NUp) error {
	var param string

	// Completion support
	for k := range m {
		if !strings.HasPrefix(k, strings.ToLower(paramPrefix)) {
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

func parsePageFormatNUp(s string, nup *model.NUp) (err error) {
	if nup.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	nup.PageDim, nup.PageSize, err = types.ParsePageFormat(s)
	nup.UserDim = true
	return err
}

func parseDimensionsNUp(s string, nup *model.NUp) (err error) {
	if nup.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	nup.PageDim, nup.PageSize, err = parsePageDim(s, nup.InpUnit)
	nup.UserDim = true
	return err
}

func parseOrientation(s string, nup *model.NUp) error {
	switch s {
	case "rd":
		nup.Orient = model.RightDown
	case "dr":
		nup.Orient = model.DownRight
	case "ld":
		nup.Orient = model.LeftDown
	case "dl":
		nup.Orient = model.DownLeft
	default:
		return errors.Errorf("pdfcpu: unknown nUp orientation: %s", s)
	}

	return nil
}

func parseElementBorder(s string, nup *model.NUp) error {
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

func parseBookletGuides(s string, nup *model.NUp) error {
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

func parseBookletMultifolio(s string, nup *model.NUp) error {
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

func parseBookletFolioSize(s string, nup *model.NUp) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("pdfcpu: illegal folio size: must be an numeric value, %s\n", s)
	}

	nup.FolioSize = i
	return nil
}

func parseElementMargin(s string, nup *model.NUp) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}

	if f < 0 {
		return errors.New("pdfcpu: nUp margin, Please provide a positive value")
	}

	nup.Margin = types.ToUserSpace(f, nup.InpUnit)

	return nil
}

func parseSheetBackgroundColor(s string, nup *model.NUp) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	nup.BgColor = &c
	return nil
}

// ParseNUpDetails parses a NUp command string into an internal structure.
func ParseNUpDetails(s string, nup *model.NUp) error {
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
func PDFNUpConfig(val int, desc string) (*model.NUp, error) {
	nup := model.DefaultNUpConfig()
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpValue(val, nup)
}

// ImageNUpConfig returns an NUp configuration for Nup-ing image files.
func ImageNUpConfig(val int, desc string) (*model.NUp, error) {
	nup, err := PDFNUpConfig(val, desc)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// PDFGridConfig returns a grid configuration for Nup-ing PDF files.
func PDFGridConfig(rows, cols int, desc string) (*model.NUp, error) {
	nup := model.DefaultNUpConfig()
	nup.PageGrid = true
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpGridDefinition(rows, cols, nup)
}

// ImageGridConfig returns a grid configuration for Nup-ing image files.
func ImageGridConfig(rows, cols int, desc string) (*model.NUp, error) {
	nup, err := PDFGridConfig(rows, cols, desc)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// ParseNUpValue parses the NUp value into an internal structure.
func ParseNUpValue(n int, nUp *model.NUp) error {
	if !types.IntMemberOf(n, nUpValues) {
		return errInvalidNUpVal
	}

	// The n-Up layout depends on the orientation of the chosen output paper size.
	// This optional paper size may also be specified by dimensions in user unit.
	// The default paper size is A4 or A4P (A4 in portrait mode) respectively.
	var portrait bool
	if nUp.PageDim == nil {
		portrait = types.PaperSize[nUp.PageSize].Portrait()
	} else {
		portrait = types.RectForDim(nUp.PageDim.Width, nUp.PageDim.Height).Portrait()
	}

	d := nUpDims[n]
	if portrait {
		d.Width, d.Height = d.Height, d.Width
	}

	nUp.Grid = &d

	return nil
}

// ParseNUpGridDefinition parses NUp grid dimensions into an internal structure.
func ParseNUpGridDefinition(rows, cols int, nUp *model.NUp) error {
	m := cols
	if m <= 0 {
		return errInvalidGridDims
	}

	n := rows
	if n <= 0 {
		return errInvalidGridDims
	}

	nUp.Grid = &types.Dim{Width: float64(m), Height: float64(n)}

	return nil
}

func nUpImagePDFBytes(w io.Writer, imgWidth, imgHeight int, nup *model.NUp, formResID string) {
	for _, r := range nup.RectsForGrid() {
		// Append to content stream.
		model.NUpTilePDFBytes(w, types.RectForDim(float64(imgWidth), float64(imgHeight)), r, formResID, nup, false, true)
	}
}

func createNUpFormForImage(xRefTable *model.XRefTable, imgIndRef *types.IndirectRef, w, h, i int) (*types.IndirectRef, error) {
	imgResID := fmt.Sprintf("Im%d", i)
	bb := types.RectForDim(float64(w), float64(h))

	var b bytes.Buffer
	fmt.Fprintf(&b, "/%s Do ", imgResID)

	d := types.Dict(
		map[string]types.Object{
			"ProcSet": types.NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
			"XObject": types.Dict(map[string]types.Object{imgResID: *imgIndRef}),
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	sd := types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":      types.Name("XObject"),
				"Subtype":   types.Name("Form"),
				"BBox":      bb.Array(),
				"Matrix":    types.NewIntegerArray(1, 0, 0, 1, 0, 0),
				"Resources": *ir,
			},
		),
		Content:        b.Bytes(),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	sd.InsertName("Filter", filter.Flate)

	if err = sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(sd)
}

// NewNUpPageForImage creates a new page dict in xRefTable for given image filename and n-up conf.
func NewNUpPageForImage(xRefTable *model.XRefTable, fileName string, parentIndRef *types.IndirectRef, nup *model.NUp) (*types.IndirectRef, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// create image dict.
	imgIndRef, w, h, err := model.CreateImageResource(xRefTable, f, false, false)
	if err != nil {
		return nil, err
	}

	resID := 0

	formIndRef, err := createNUpFormForImage(xRefTable, imgIndRef, w, h, resID)
	if err != nil {
		return nil, err
	}

	formResID := fmt.Sprintf("Fm%d", resID)

	resourceDict := types.Dict(
		map[string]types.Object{
			"XObject": types.Dict(map[string]types.Object{formResID: *formIndRef}),
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
	mediaBox := types.RectForDim(dim.Width, dim.Height)

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":      types.Name("Page"),
			"Parent":    *parentIndRef,
			"MediaBox":  mediaBox.Array(),
			"Resources": *resIndRef,
			"Contents":  *contentsIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(pageDict)
}

// NUpFromOneImage creates one page with instances of one image.
func NUpFromOneImage(ctx *model.Context, fileName string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	indRef, err := NewNUpPageForImage(ctx.XRefTable, fileName, pagesIndRef, nup)
	if err != nil {
		return err
	}

	if err = model.AppendPageTree(indRef, 1, pagesDict); err != nil {
		return err
	}

	ctx.PageCount++

	return nil
}

func wrapUpPage(ctx *model.Context, nup *model.NUp, d types.Dict, buf bytes.Buffer, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	xRefTable := ctx.XRefTable

	var fm model.FontMap
	if nup.BookletGuides {
		// For booklets only.
		fm = model.DrawBookletGuides(nup, &buf)
	}

	resourceDict := types.Dict(
		map[string]types.Object{
			"XObject": d,
		},
	)

	fontRes, err := pdffont.FontResources(xRefTable, fm)
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
	mediaBox := types.RectForDim(dim.Width, dim.Height)

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":      types.Name("Page"),
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

	if err = model.AppendPageTree(indRef, 1, pagesDict); err != nil {
		return err
	}

	ctx.PageCount++

	return nil
}

func nupPageNumber(i int, sortedPageNumbers []int) int {
	var pageNumber int
	if i < len(sortedPageNumbers) {
		pageNumber = sortedPageNumbers[i]
	}
	return pageNumber
}

func sortSelectedPages(pages types.IntSet) []int {
	var pageNumbers []int
	for k, v := range pages {
		if v {
			pageNumbers = append(pageNumbers, k)
		}
	}
	sort.Ints(pageNumbers)
	return pageNumbers
}

func nupPages(
	ctx *model.Context,
	selectedPages types.IntSet,
	nup *model.NUp,
	pagesDict types.Dict,
	pagesIndRef *types.IndirectRef) error {

	var buf bytes.Buffer
	formsResDict := types.NewDict()
	rr := nup.RectsForGrid()

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
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		pageNr := nupPageNumber(i, sortedPageNumbers)
		if pageNr == 0 {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		if err := ctx.NUpTilePDFBytesForPDF(pageNr, formsResDict, &buf, rDest, nup, false); err != nil {
			return err
		}
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// NUpFromMultipleImages creates pages in NUp-style rendering each image once.
func NUpFromMultipleImages(ctx *model.Context, fileNames []string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	if nup.PageGrid {
		nup.PageDim.Width *= nup.Grid.Width
		nup.PageDim.Height *= nup.Grid.Height
	}

	xRefTable := ctx.XRefTable
	formsResDict := types.NewDict()
	var buf bytes.Buffer
	rr := nup.RectsForGrid()

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
			formsResDict = types.NewDict()
		}

		rDest := rr[i%len(rr)]

		var fileName string
		if i < len(fileNames) {
			fileName = fileNames[i]
		}

		if fileName == "" {
			// This is an empty page at the end.
			if nup.BgColor != nil {
				draw.FillRectNoBorder(&buf, rDest, *nup.BgColor)
			}
			continue
		}

		f, err := os.Open(fileName)
		if err != nil {
			return err
		}

		imgIndRef, w, h, err := model.CreateImageResource(xRefTable, f, false, false)
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
		model.NUpTilePDFBytes(&buf, types.RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, false, true)
	}

	// Wrap incomplete nUp page.
	return wrapUpPage(ctx, nup, formsResDict, buf, pagesDict, pagesIndRef)
}

// NUpFromPDF creates an n-up version of the PDF represented by xRefTable.
func NUpFromPDF(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	var mb *types.Rectangle
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
			if types.IntMemberOf(inhPAttrs.Rotate, []int{+90, -90, +270, -270}) {
				w := cropBox.Width()
				cropBox.UR.X = cropBox.LL.X + cropBox.Height()
				cropBox.UR.Y = cropBox.LL.Y + w
			}
		}

		mb = cropBox
	} else {
		mb = types.RectForDim(nup.PageDim.Width, nup.PageDim.Height)
	}

	if nup.PageGrid {
		mb.UR.X = mb.LL.X + float64(nup.Grid.Width)*mb.Width()
		mb.UR.Y = mb.LL.Y + float64(nup.Grid.Height)*mb.Height()
	}

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

	if err = nupPages(ctx, selectedPages, nup, pagesDict, pagesIndRef); err != nil {
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

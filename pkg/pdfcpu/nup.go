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
	"errors"
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
)

var (
	errInvalidGridDims  = errors.New("grid: dimensions must be: m > 0, n > 0")
	errInvalidNUpConfig = errors.New("invalid configuration string")
)

var (
	NUpValues = []int{2, 3, 4, 6, 8, 9, 12, 16}
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

var nupParamMap = parameterMap[model.NUp]{
	"dimensions":      parseDimensionsNUp,
	"formsize":        parsePageFormatNUp,
	"papersize":       parsePageFormatNUp,
	"orientation":     parseOrientation,
	"border":          parseElementBorder,
	"cropboxborder":   parseElementBorderOnCropbox,
	"margin":          parseElementMargin,
	"backgroundcolor": parseSheetBackgroundColor,
	"bgcolor":         parseSheetBackgroundColor,
	"guides":          parseBookletGuides,
	"multifolio":      parseBookletMultifolio,
	"foliosize":       parseBookletFolioSize,
	"btype":           parseBookletType,
	"binding":         parseBookletBinding,
	"enforce":         parseEnforce,
}

func parsePageFormatNUp(s string, nup *model.NUp) (err error) {
	if nup.UserDim {
		return errAmbiguousPageDim
	}
	nup.PageDim, nup.PageSize, err = types.ParsePageFormat(s)
	nup.UserDim = true
	return err
}

func parseDimensionsNUp(s string, nup *model.NUp) (err error) {
	if nup.UserDim {
		return errAmbiguousPageDim
	}
	nup.PageDim, nup.PageSize, err = ParsePageDim(s, nup.InpUnit)
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
		return fmt.Errorf("unknown nUp orientation: %s", s)
	}

	return nil
}

func parseEnforce(s string, nup *model.NUp) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		nup.Enforce = true
	case "off", "false", "f":
		nup.Enforce = false
	default:
		return errors.New("enforce best-fit orientation of content, please provide one of: on/off true/false")
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
		return errors.New("nUp border, please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseElementBorderOnCropbox(s string, nup *model.NUp) error {
	// w
	// w r g b
	// w #c
	// w round
	// w round r g b
	// w round #c

	var err error

	b := strings.Split(s, " ")
	if len(b) == 0 || len(b) > 5 {
		return fmt.Errorf("borders: need 1,2,3,4 or 5 int values, %s", s)
	}

	switch b[0] {
	case "off", "false", "f":
		return nil
	case "on", "true", "t":
		nup.BorderOnCropbox = &model.BorderStyling{Width: 1}
		return nil
	}

	nup.BorderOnCropbox = &model.BorderStyling{}
	width, err := strconv.ParseFloat(b[0], 64)
	if err != nil {
		return err
	}
	if width == 0 {
		return errors.New("borders: need width > 0")
	}
	nup.BorderOnCropbox.Width = width

	if len(b) == 1 {
		return nil
	}
	if strings.HasPrefix("round", b[1]) {
		style := types.LJRound
		nup.BorderOnCropbox.LineStyle = &style
		if len(b) == 2 {
			return nil
		}
		c, err := color.ParseColor(strings.Join(b[2:], " "))
		nup.BorderOnCropbox.Color = &c
		return err
	}

	c, err := color.ParseColor(strings.Join(b[1:], " "))
	nup.BorderOnCropbox.Color = &c
	return err
}

func parseBookletGuides(s string, nup *model.NUp) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		nup.BookletGuides = true
	case "off", "false", "f":
		nup.BookletGuides = false
	default:
		return errors.New("booklet guides, please provide one of: on/off true/false t/f")
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
		return errors.New("booklet guides, please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseBookletFolioSize(s string, nup *model.NUp) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("illegal folio size: must be an numeric value, %s", s)
	}

	nup.FolioSize = i
	return nil
}

func parseBookletType(s string, nup *model.NUp) error {
	switch strings.ToLower(s) {
	case "booklet":
		nup.BookletType = model.Booklet
	case "bookletadvanced":
		nup.BookletType = model.BookletAdvanced
	case "perfectbound":
		nup.BookletType = model.BookletPerfectBound
	default:
		return errors.New("booklet type, please provide one of: booklet perfectbound")
	}
	return nil
}

func parseBookletBinding(s string, nup *model.NUp) error {
	switch strings.ToLower(s) {
	case "short":
		nup.BookletBinding = model.ShortEdge
	case "long":
		nup.BookletBinding = model.LongEdge
	default:
		return errors.New("booklet binding, please provide one of: short long")
	}
	return nil
}

func parseElementMargin(s string, nup *model.NUp) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}

	if f < 0 {
		return errors.New("nUp margin, Please provide a positive value")
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

		if err := handleParameter(nupParamMap, paramPrefix, paramValueStr, nup); err != nil {
			return err
		}
	}

	return nil
}

// PDFNUpConfig returns an NUp configuration for Nup-ing PDF files.
func PDFNUpConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup := model.DefaultNUpConfig()
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	nup.InpUnit = conf.Unit
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	if !types.IntMemberOf(val, NUpValues) {
		ss := make([]string, len(NUpValues))
		for i, v := range NUpValues {
			ss[i] = strconv.Itoa(v)
		}
		return nil, fmt.Errorf("n must be one of %s", strings.Join(ss, ", "))
	}
	return nup, ParseNUpValue(val, nup)
}

// ImageNUpConfig returns an NUp configuration for Nup-ing image files.
func ImageNUpConfig(val int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup, err := PDFNUpConfig(val, desc, conf)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// PDFGridConfig returns a grid configuration for PDF files.
func PDFGridConfig(rows, cols int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup := model.DefaultNUpConfig()
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	nup.InpUnit = conf.Unit
	nup.PageGrid = true
	if desc != "" {
		if err := ParseNUpDetails(desc, nup); err != nil {
			return nil, err
		}
	}
	return nup, ParseNUpGridDefinition(rows, cols, nup)
}

// ImageGridConfig returns a grid configuration for image files.
func ImageGridConfig(rows, cols int, desc string, conf *model.Configuration) (*model.NUp, error) {
	nup, err := PDFGridConfig(rows, cols, desc, conf)
	if err != nil {
		return nil, err
	}
	nup.ImgInputFile = true
	return nup, nil
}

// ParseNUpValue parses the NUp value into an internal structure.
func ParseNUpValue(n int, nUp *model.NUp) error {
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

// ParseNUpGridDefinition parses grid dimensions into a shared imposition configuration.
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
		model.NUpTilePDFBytes(w, types.RectForDim(float64(imgWidth), float64(imgHeight)), r, formResID, nup, false)
	}
}

func createNUpFormForImage(xRefTable *model.XRefTable, imgIndRef *types.IndirectRef, w, h, formIndex int) (*types.IndirectRef, error) {
	return createImageForm("n-up", xRefTable, imgIndRef, w, h, formIndex)
}

func createImageForm(operation string, xRefTable *model.XRefTable, imgIndRef *types.IndirectRef, w, h, formIndex int) (*types.IndirectRef, error) {
	imgResID := fmt.Sprintf("Im%d", formIndex)
	formNr := formIndex + 1
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
		return nil, fmt.Errorf("%s image form %d: store resource dictionary: %w", operation, formNr, err)
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
		return nil, fmt.Errorf("%s image form %d: encode form stream: %w", operation, formNr, err)
	}

	formIndRef, err := xRefTable.IndRefForNewObject(sd)
	if err != nil {
		return nil, fmt.Errorf("%s image form %d: store form stream: %w", operation, formNr, err)
	}
	return formIndRef, nil
}

func wrapImageError(operation string, imageNr int, fileName, phase string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s image %d %q: %s: %w", operation, imageNr, fileName, phase, err)
}

func loadImageResource(
	operation string,
	xRefTable *model.XRefTable,
	imageNr int,
	fileName string) (imgIndRef *types.IndirectRef, w, h int, err error) {
	return loadImageResourceWith(operation, xRefTable, imageNr, fileName, model.CreateImageResource)
}

func loadNUpImageResourceWith(
	xRefTable *model.XRefTable,
	imageNr int,
	fileName string,
	createImageResource func(*model.XRefTable, io.Reader) (*types.IndirectRef, int, int, error),
) (imgIndRef *types.IndirectRef, w, h int, err error) {
	return loadImageResourceWith("n-up", xRefTable, imageNr, fileName, createImageResource)
}

func loadImageResourceWith(
	operation string,
	xRefTable *model.XRefTable,
	imageNr int,
	fileName string,
	createImageResource func(*model.XRefTable, io.Reader) (*types.IndirectRef, int, int, error),
) (imgIndRef *types.IndirectRef, w, h int, err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, 0, 0, wrapImageError(operation, imageNr, fileName, "open", err)
	}
	defer func() {
		err = errors.Join(err, wrapImageError(operation, imageNr, fileName, "close", f.Close()))
	}()

	imgIndRef, w, h, err = createImageResource(xRefTable, f)
	if err != nil {
		err = wrapImageError(operation, imageNr, fileName, "create resource", err)
	}
	return imgIndRef, w, h, err
}

func newPageForImage(
	operation string,
	xRefTable *model.XRefTable,
	imageNr int,
	fileName string,
	parentIndRef *types.IndirectRef,
	nup *model.NUp) (*types.IndirectRef, error) {
	imgIndRef, w, h, err := loadImageResource(operation, xRefTable, imageNr, fileName)
	if err != nil {
		return nil, err
	}

	resID := 0
	formIndRef, err := createImageForm(operation, xRefTable, imgIndRef, w, h, resID)
	if err != nil {
		return nil, wrapImageError(operation, imageNr, fileName, "create form", err)
	}

	formResID := fmt.Sprintf("Fm%d", resID)
	resourceDict := types.Dict(
		map[string]types.Object{
			"XObject": types.Dict(map[string]types.Object{formResID: *formIndRef}),
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(resourceDict)
	if err != nil {
		return nil, wrapImageError(operation, imageNr, fileName, "store page resources", err)
	}

	var buf bytes.Buffer
	nUpImagePDFBytes(&buf, w, h, nup, formResID)
	sd, err := xRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, wrapImageError(operation, imageNr, fileName, "create page content", err)
	}
	if err = sd.Encode(); err != nil {
		return nil, wrapImageError(operation, imageNr, fileName, "encode page content", err)
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, wrapImageError(operation, imageNr, fileName, "store page content", err)
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

	indRef, err := xRefTable.IndRefForNewObject(pageDict)
	if err != nil {
		return nil, wrapImageError(operation, imageNr, fileName, "store page dictionary", err)
	}
	return indRef, nil
}

// NUpFromOneImage creates one page with instances of one image.
// On failure, ctx may contain partial objects and its PageCount remains unchanged.
func NUpFromOneImage(ctx *model.Context, fileName string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	return fromOneImage("n-up", ctx, fileName, nup, pagesDict, pagesIndRef)
}

// GridFromOneImage creates one grid page with instances of one image.
// On failure, ctx may contain partial objects and its PageCount remains unchanged.
func GridFromOneImage(ctx *model.Context, fileName string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	return fromOneImage("grid", ctx, fileName, nup, pagesDict, pagesIndRef)
}

func fromOneImage(operation string, ctx *model.Context, fileName string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	indRef, err := newPageForImage(operation, ctx.XRefTable, 1, fileName, pagesIndRef, nup)
	if err != nil {
		return err
	}

	if err := ctx.SetValid(*indRef); err != nil {
		return wrapImageError(operation, 1, fileName, "mark page valid", err)
	}

	if err = model.AppendPageTree(indRef, 1, pagesDict); err != nil {
		return wrapImageError(operation, 1, fileName, "append page tree", err)
	}

	ctx.PageCount++
	return nil
}

func wrapUpPage(ctx *model.Context, nup *model.NUp, d types.Dict, buf bytes.Buffer, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	return wrapUpPageForOperation("n-up", ctx, nup, d, buf, pagesDict, pagesIndRef)
}

func wrapUpPageForOperation(operation string, ctx *model.Context, nup *model.NUp, d types.Dict, buf bytes.Buffer, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	xRefTable := ctx.XRefTable

	var fm model.FontMap
	if nup.BookletGuides {
		// For booklets only.
		var err error
		fm, err = model.DrawBookletGuides(xRefTable, nup, &buf)
		if err != nil {
			return fmt.Errorf("%s output page: render booklet guides: %w", operation, err)
		}
	}

	resourceDict := types.Dict(
		map[string]types.Object{
			"XObject": d,
		},
	)

	fontRes, err := pdffont.FontResources(xRefTable, fm)
	if err != nil {
		return fmt.Errorf("%s output page: create font resources: %w", operation, err)
	}

	if len(fontRes) > 0 {
		resourceDict["Font"] = fontRes
	}

	resIndRef, err := xRefTable.IndRefForNewObject(resourceDict)
	if err != nil {
		return fmt.Errorf("%s output page: store resource dictionary: %w", operation, err)
	}

	bb := buf.Bytes()
	bbCopy := make([]byte, len(bb))
	copy(bbCopy, bb)

	sd, err := xRefTable.NewStreamDictForBuf(bbCopy)
	if err != nil {
		return fmt.Errorf("%s output page: create content stream: %w", operation, err)
	}

	if err = sd.Encode(); err != nil {
		return fmt.Errorf("%s output page: encode content stream: %w", operation, err)
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("%s output page: store content stream: %w", operation, err)
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
		return fmt.Errorf("%s output page: store page dictionary: %w", operation, err)
	}

	if err := ctx.SetValid(*indRef); err != nil {
		return fmt.Errorf("%s output page: mark page valid: %w", operation, err)
	}

	if err = model.AppendPageTree(indRef, 1, pagesDict); err != nil {
		return fmt.Errorf("%s output page: append page tree: %w", operation, err)
	}

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

func impositionPages(
	operation string,
	ctx *model.Context,
	selectedPages types.IntSet,
	nup *model.NUp,
	pagesDict types.Dict,
	pagesIndRef *types.IndirectRef) (int, error) {
	var buf bytes.Buffer
	formsResDict := types.NewDict()
	rr := nup.RectsForGrid()
	outputPageNr := 1

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
			if err := wrapUpPageForOperation(operation, ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return 0, fmt.Errorf("%s output page %d: wrap page: %w", operation, outputPageNr, err)
			}
			outputPageNr++
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

		if err := ctx.TilePDFBytesForImposition(operation, pageNr, formsResDict, &buf, rDest, nup, false); err != nil {
			return 0, fmt.Errorf("%s page imposition: %w", operation, err)
		}
	}

	// Wrap incomplete output page.
	if err := wrapUpPageForOperation(operation, ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
		return 0, fmt.Errorf("%s output page %d: wrap page: %w", operation, outputPageNr, err)
	}
	return outputPageNr, nil
}

func impositionImageConfiguration(nup *model.NUp) *model.NUp {
	operationNUp := *nup
	pageDim := *nup.PageDim
	if nup.PageGrid {
		pageDim.Width *= nup.Grid.Width
		pageDim.Height *= nup.Grid.Height
	}
	operationNUp.PageDim = &pageDim
	return &operationNUp
}

// NUpFromMultipleImages creates pages in NUp-style rendering each image once.
// On failure, ctx may contain partial objects or page-tree changes and its PageCount remains unchanged.
func NUpFromMultipleImages(ctx *model.Context, fileNames []string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	return fromMultipleImages("n-up", ctx, fileNames, nup, pagesDict, pagesIndRef)
}

// GridFromMultipleImages creates grid pages containing each image once.
// On failure, ctx may contain partial objects or page-tree changes and its PageCount remains unchanged.
func GridFromMultipleImages(ctx *model.Context, fileNames []string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	return fromMultipleImages("grid", ctx, fileNames, nup, pagesDict, pagesIndRef)
}

func fromMultipleImages(operation string, ctx *model.Context, fileNames []string, nup *model.NUp, pagesDict types.Dict, pagesIndRef *types.IndirectRef) error {
	nup = impositionImageConfiguration(nup)

	xRefTable := ctx.XRefTable
	formsResDict := types.NewDict()
	var buf bytes.Buffer
	rr := nup.RectsForGrid()
	outputPageNr := 1

	// fileCount must be a multiple of n.
	// If not, we will insert blank pages at the end.
	fileCount := len(fileNames)
	if fileCount%nup.N() != 0 {
		fileCount += nup.N() - fileCount%nup.N()
	}

	for i := 0; i < fileCount; i++ {

		if i > 0 && i%len(rr) == 0 {
			// Wrap complete nUp page.
			if err := wrapUpPageForOperation(operation, ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
				return fmt.Errorf("%s output page %d: wrap page: %w", operation, outputPageNr, err)
			}
			outputPageNr++
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

		imageNr := i + 1
		imgIndRef, w, h, err := loadImageResource(operation, xRefTable, imageNr, fileName)
		if err != nil {
			return err
		}

		formIndRef, err := createImageForm(operation, xRefTable, imgIndRef, w, h, i)
		if err != nil {
			return wrapImageError(operation, imageNr, fileName, "create form", err)
		}

		formResID := fmt.Sprintf("Fm%d", i)
		formsResDict.Insert(formResID, *formIndRef)

		// Append to content stream of page i.
		model.NUpTilePDFBytes(&buf, types.RectForDim(float64(w), float64(h)), rr[i%len(rr)], formResID, nup, false)
	}

	// Wrap incomplete nUp page.
	if err := wrapUpPageForOperation(operation, ctx, nup, formsResDict, buf, pagesDict, pagesIndRef); err != nil {
		return fmt.Errorf("%s output page %d: wrap page: %w", operation, outputPageNr, err)
	}
	ctx.PageCount += outputPageNr
	return nil
}

func impositionPDFConfiguration(operation string, ctx *model.Context, nup *model.NUp) (*model.NUp, *types.Rectangle, error) {
	operationNUp := *nup
	var mb *types.Rectangle
	if nup.PageDim == nil {
		consolidateRes := false
		d, _, inhPAttrs, err := ctx.PageDict(1, consolidateRes)
		if err != nil {
			return nil, nil, fmt.Errorf("%s page tree: derive dimensions from source page 1: %w", operation, err)
		}
		if d == nil {
			return nil, nil, fmt.Errorf("%s page tree: derive dimensions from source page 1: %w", operation, model.ErrPageNotFound)
		}
		cropBox := inhPAttrs.MediaBox
		if inhPAttrs.CropBox != nil {
			cropBox = inhPAttrs.CropBox
		}
		mb = cropBox.Clone()
		if types.IntMemberOf(inhPAttrs.Rotate, []int{+90, -90, +270, -270}) {
			w := mb.Width()
			mb.UR.X = mb.LL.X + mb.Height()
			mb.UR.Y = mb.LL.Y + w
		}
	} else {
		pageDim := *nup.PageDim
		operationNUp.PageDim = &pageDim
		mb = types.RectForDim(pageDim.Width, pageDim.Height)
	}

	if nup.PageGrid {
		mb.UR.X = mb.LL.X + nup.Grid.Width*mb.Width()
		mb.UR.Y = mb.LL.Y + nup.Grid.Height*mb.Height()
	}
	operationNUp.PageDim = &types.Dim{Width: mb.Width(), Height: mb.Height()}
	return &operationNUp, mb, nil
}

// NUpFromPDF creates an n-up version of the PDF represented by xRefTable.
// On failure, the original page tree and PageCount remain authoritative; ctx may contain orphaned partial output objects.
func NUpFromPDF(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	return fromPDF("n-up", ctx, selectedPages, nup)
}

// GridFromPDF creates a grid version of the PDF represented by xRefTable.
// On failure, the original page tree and PageCount remain authoritative; ctx may contain orphaned partial output objects.
func GridFromPDF(ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	return fromPDF("grid", ctx, selectedPages, nup)
}

func fromPDF(operation string, ctx *model.Context, selectedPages types.IntSet, nup *model.NUp) error {
	nup, mb, err := impositionPDFConfiguration(operation, ctx, nup)
	if err != nil {
		return err
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
		return fmt.Errorf("%s page tree: create root: %w", operation, err)
	}

	pageCount, err := impositionPages(operation, ctx, selectedPages, nup, pagesDict, pagesIndRef)
	if err != nil {
		return err
	}

	// Replace original pagesDict.
	rootDict, err := ctx.Catalog()
	if err != nil {
		return fmt.Errorf("%s page tree: access catalog: %w", operation, err)
	}

	rootDict.Update("Pages", *pagesIndRef)
	ctx.PageCount = pageCount
	return nil
}

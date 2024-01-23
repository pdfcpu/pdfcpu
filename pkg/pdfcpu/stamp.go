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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/format"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

const stampWithBBox = false

var (
	errNoWatermark        = errors.New("pdfcpu: no watermarks found")
	errCorruptOCGs        = errors.New("pdfcpu: OCProperties: corrupt OCGs element")
	ErrUnsupportedVersion = errors.New("pdfcpu: PDF 2.0 unsupported for this operation")
)

type watermarkParamMap map[string]func(string, *model.Watermark) error

func textDescriptor(wm model.Watermark, timestampFormat string, pageNr, pageCount int) (model.TextDescriptor, bool) {
	t, unique := format.Text(wm.TextString, timestampFormat, pageNr, pageCount)
	td := model.TextDescriptor{
		Text:           t,
		FontName:       wm.FontName,
		FontSize:       wm.FontSize,
		Scale:          wm.Scale,
		ScaleAbs:       wm.ScaleAbs,
		RMode:          wm.RenderMode,
		StrokeCol:      wm.StrokeColor,
		FillCol:        wm.FillColor,
		ShowBackground: true,
	}
	if wm.BgColor != nil {
		td.ShowTextBB = true
		td.BackgroundCol = *wm.BgColor
	}
	return td, unique
}

// Handle applies parameter completion and if successful
// parses the parameter values into import.
func (m watermarkParamMap) Handle(paramPrefix, paramValueStr string, imp *model.Watermark) error {
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
		return errors.Errorf("pdfcpu: unknown parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, imp)
}

var wmParamMap = watermarkParamMap{
	"aligntext":       parseTextHorAlignment,
	"backgroundcolor": parseBackgroundColor,
	"bgcolor":         parseBackgroundColor,
	"border":          parseBorder,
	"color":           parseFillColor,
	"diagonal":        parseDiagonal,
	"fillcolor":       parseFillColor,
	"fontname":        parseFontName,
	"margins":         parseMargins,
	"mode":            parseRenderMode,
	"offset":          parsePositionOffsetWM,
	"opacity":         parseOpacity,
	"points":          parseFontSize,
	"position":        parsePositionAnchorWM,
	"rendermode":      parseRenderMode,
	"rtl":             parseRightToLeft,
	"rotation":        parseRotation,
	"scalefactor":     parseScaleFactorWM,
	"strokecolor":     parseStrokeColor,
	"url":             parseURL,
}

func parseTextHorAlignment(s string, wm *model.Watermark) error {
	var a types.HAlignment
	switch s {
	case "l", "left":
		a = types.AlignLeft
	case "r", "right":
		a = types.AlignRight
	case "c", "center":
		a = types.AlignCenter
	case "j", "justify":
		a = types.AlignJustify
	default:
		return errors.Errorf("pdfcpu: unknown horizontal alignment (l,r,c,j): %s", s)
	}

	wm.HAlign = &a

	return nil
}

func parsePositionAnchorWM(s string, wm *model.Watermark) error {
	a, err := types.ParsePositionAnchor(s)
	if err != nil {
		return err
	}
	if a == types.Full {
		a = types.Center
	}
	wm.Pos = a
	return nil
}

func parsePositionOffsetWM(s string, wm *model.Watermark) error {
	d := strings.Split(s, " ")
	if len(d) != 2 {
		return errors.Errorf("pdfcpu: illegal position offset string: need 2 numeric values, %s\n", s)
	}

	f, err := strconv.ParseFloat(d[0], 64)
	if err != nil {
		return err
	}
	wm.Dx = types.ToUserSpace(f, wm.InpUnit)

	f, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	wm.Dy = types.ToUserSpace(f, wm.InpUnit)

	return nil
}

func parseScaleFactorWM(s string, wm *model.Watermark) (err error) {
	wm.Scale, wm.ScaleAbs, err = parseScaleFactor(s)
	return err
}

func parseFontName(s string, wm *model.Watermark) error {
	if !font.SupportedFont(s) {
		return errors.Errorf("pdfcpu: %s is unsupported, please refer to \"pdfcpu fonts list\".\n", s)
	}
	wm.FontName = s
	return nil
}

func parseURL(s string, wm *model.Watermark) error {
	if !wm.OnTop {
		return errors.Errorf("pdfcpu: \"url\" supported for stamps only.\n")
	}
	if !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}
	if _, err := url.ParseRequestURI(s); err != nil {
		return err
	}
	wm.URL = s
	return nil
}

func parseFontSize(s string, wm *model.Watermark) error {
	fs, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	wm.FontSize = fs

	return nil
}

func parseScaleFactor(s string) (float64, bool, error) {
	ss := strings.Split(s, " ")
	if len(ss) > 2 {
		return 0, false, errors.Errorf("pdfcpu: invalid factor string %s: 0.0 < i <= 1.0 {rel} | 0.0 < i {abs}\n", s)
	}

	sc, err := strconv.ParseFloat(ss[0], 64)
	if err != nil {
		return 0, false, errors.Errorf("pdfcpu: scale factor must be a float value: %s\n", ss[0])
	}

	if sc <= 0 {
		return 0, false, errors.Errorf("pdfcpu: invalid scale factor %.2f: 0.0 < i <= 1.0 {rel} | 0.0 < i {abs}\n", sc)
	}

	var scaleAbs bool

	if len(ss) == 1 {
		// Assume relative scaling for sc <= 1 and absolute scaling for sc > 1.
		scaleAbs = sc > 1
		return sc, scaleAbs, nil
	}

	switch ss[1] {
	case "a", "abs", "absolute":
		scaleAbs = true

	case "r", "rel", "relative":
		scaleAbs = false

	default:
		return 0, false, errors.Errorf("pdfcpu: illegal scale mode: abs|rel, %s\n", ss[1])
	}

	if !scaleAbs && sc > 1 {
		return 0, false, errors.Errorf("pdfcpu: invalid relative scale factor %.2f: 0.0 < i <= 1\n", sc)
	}

	return sc, scaleAbs, nil
}

func parseRightToLeft(s string, wm *model.Watermark) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		wm.RTL = true
	case "off", "false", "f":
		wm.RTL = false
	default:
		return errors.New("pdfcpu: rtl (right-to-left), please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseStrokeColor(s string, wm *model.Watermark) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	wm.StrokeColor = c
	return nil
}

func parseFillColor(s string, wm *model.Watermark) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	wm.FillColor = c
	return nil
}

func parseBackgroundColor(s string, wm *model.Watermark) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	wm.BgColor = &c
	return nil
}

func parseRotation(s string, wm *model.Watermark) error {
	if wm.UserRotOrDiagonal {
		return errors.New("pdfcpu: please specify rotation or diagonal (r or d)")
	}

	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("pdfcpu: rotation must be a float value: %s\n", s)
	}
	if r < -180 || r > 180 {
		return errors.Errorf("pdfcpu: illegal rotation: -180 <= r <= 180 degrees, %s\n", s)
	}

	wm.Rotation = r
	wm.Diagonal = model.NoDiagonal
	wm.UserRotOrDiagonal = true

	return nil
}

func parseDiagonal(s string, wm *model.Watermark) error {
	if wm.UserRotOrDiagonal {
		return errors.New("pdfcpu: please specify rotation or diagonal (r or d)")
	}

	d, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("pdfcpu: illegal diagonal value: allowed 1 or 2, %s\n", s)
	}
	if d != model.DiagonalLLToUR && d != model.DiagonalULToLR {
		return errors.New("pdfcpu: diagonal: 1..lower left to upper right, 2..upper left to lower right")
	}

	wm.Diagonal = d
	wm.Rotation = 0
	wm.UserRotOrDiagonal = true

	return nil
}

func parseOpacity(s string, wm *model.Watermark) error {
	o, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("pdfcpu: opacity must be a float value: %s\n", s)
	}
	if o < 0 || o > 1 {
		return errors.Errorf("pdfcpu: illegal opacity: 0.0 <= r <= 1.0, %s\n", s)
	}
	wm.Opacity = o

	return nil
}

func parseRenderMode(s string, wm *model.Watermark) error {
	m, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("pdfcpu: illegal render mode value: allowed 0,1,2, %s\n", s)
	}
	rm := draw.RenderMode(m)
	if rm != draw.RMFill && rm != draw.RMStroke && rm != draw.RMFillAndStroke {
		return errors.New("pdfcpu: valid rendermodes: 0..fill, 1..stroke, 2..fill&stroke")
	}
	wm.RenderMode = rm

	return nil
}

func parseMargins(s string, wm *model.Watermark) error {
	var err error

	m := strings.Split(s, " ")
	if len(m) == 0 || len(m) > 4 {
		return errors.Errorf("pdfcpu: margins: need 1,2,3 or 4 int values, %s\n", s)
	}

	f1, err := strconv.ParseFloat(m[0], 64)
	if err != nil {
		return err
	}

	if len(m) == 1 {
		wm.MLeft = f1
		wm.MRight = f1
		wm.MTop = f1
		wm.MBot = f1
		return nil
	}

	f2, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return err
	}

	if len(m) == 2 {
		wm.MTop, wm.MBot = f1, f1
		wm.MLeft, wm.MRight = f2, f2
		return nil
	}

	f3, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return err
	}

	if len(m) == 3 {
		wm.MTop = f1
		wm.MLeft, wm.MRight = f2, f2
		wm.MBot = f3
		return nil
	}

	f4, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return err
	}

	wm.MTop = f1
	wm.MRight = f2
	wm.MBot = f3
	wm.MLeft = f4
	return nil
}

func parseBorder(s string, wm *model.Watermark) error {
	// w
	// w r g b
	// w #c
	// w round
	// w round r g b
	// w round #c

	var err error

	b := strings.Split(s, " ")
	if len(b) == 0 || len(b) > 5 {
		return errors.Errorf("pdfcpu: borders: need 1,2,3,4 or 5 int values, %s\n", s)
	}

	wm.BorderWidth, err = strconv.ParseFloat(b[0], 64)
	if err != nil {
		return err
	}
	if wm.BorderWidth == 0 {
		return errors.New("pdfcpu: borders: need width > 0")
	}

	if len(b) == 1 {
		return nil
	}

	if strings.HasPrefix("round", b[1]) {
		wm.BorderStyle = types.LJRound
		if len(b) == 2 {
			return nil
		}
		c, err := color.ParseColor(strings.Join(b[2:], " "))
		wm.BorderColor = &c
		return err
	}

	c, err := color.ParseColor(strings.Join(b[1:], " "))
	wm.BorderColor = &c
	return err
}

func parseWatermarkDetails(mode int, modeParm, s string, onTop bool, u types.DisplayUnit) (*model.Watermark, error) {
	wm := model.DefaultWatermarkConfig()
	wm.OnTop = onTop
	wm.InpUnit = u

	ss := strings.Split(s, ",")
	if len(ss) > 0 && len(ss[0]) == 0 {
		return wm, setWatermarkType(mode, modeParm, wm)
	}

	for _, s := range ss {
		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, parseWatermarkError(onTop)
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := wmParamMap.Handle(paramPrefix, paramValueStr, wm); err != nil {
			return nil, err
		}
	}

	return wm, setWatermarkType(mode, modeParm, wm)
}

// ParseTextWatermarkDetails parses a text Watermark/Stamp command string into an internal structure.
func ParseTextWatermarkDetails(text, desc string, onTop bool, u types.DisplayUnit) (*model.Watermark, error) {
	return parseWatermarkDetails(model.WMText, text, desc, onTop, u)
}

// ParseImageWatermarkDetails parses an image Watermark/Stamp command string into an internal structure.
func ParseImageWatermarkDetails(fileName, desc string, onTop bool, u types.DisplayUnit) (*model.Watermark, error) {
	return parseWatermarkDetails(model.WMImage, fileName, desc, onTop, u)
}

// ParsePDFWatermarkDetails parses a PDF Watermark/Stamp command string into an internal structure.
func ParsePDFWatermarkDetails(fileName, desc string, onTop bool, u types.DisplayUnit) (*model.Watermark, error) {
	return parseWatermarkDetails(model.WMPDF, fileName, desc, onTop, u)
}

func onTopString(onTop bool) string {
	e := "watermark"
	if onTop {
		e = "stamp"
	}
	return e
}

func parseWatermarkError(onTop bool) error {
	s := onTopString(onTop)
	return errors.Errorf("Invalid %s configuration string. Please consult pdfcpu help %s.\n", s, s)
}

func setTextWatermark(s string, wm *model.Watermark) {
	wm.TextString = s
	if font.IsCoreFont(wm.FontName) {
		bb := []byte{}
		for _, r := range s {
			// Unicode => char code
			b := byte(0x20) // better use glyph: .notdef
			if r <= 0xff {
				b = byte(r)
			}
			bb = append(bb, b)
		}
		s = string(bb)
	} else {
		bb := []byte{}
		u := utf16.Encode([]rune(s))
		for _, i := range u {
			bb = append(bb, byte((i>>8)&0xFF))
			bb = append(bb, byte(i&0xFF))
		}
		s = string(bb)
	}
	s = strings.ReplaceAll(s, "\\n", "\n")
	wm.TextLines = append(wm.TextLines, strings.FieldsFunc(s, func(c rune) bool { return c == 0x0a })...)
}

func setImageWatermark(s string, wm *model.Watermark) error {
	if len(s) == 0 {
		// The caller is expected to provide: wm.Image (see api.ImageWatermarkForReader)
		return nil
	}
	if !model.ImageFileName(s) {
		return errors.New("imageFileName has to have one of these extensions: .jpg, .jpeg, .png, .tif, .tiff, .webp")
	}
	wm.FileName = s
	f, err := os.Open(wm.FileName)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return err
	}

	wm.Image = bytes.NewReader(buf.Bytes())
	return nil
}

func setPDFWatermark(s string, wm *model.Watermark) error {
	if len(s) == 0 {
		/*
			The caller is expected to provide:
				wm.PDF and optionally wm.PdfPageNrSrc (see api.PDFWatermarkForReadSeeker)
			or
				wm.PDF and wm.PdfMultiStartPageNrSrc and wm.PdfMultiStartPageNrDest (see api.PDFMultiWatermarkForReadSeeker)

			Supported usecases:

			pdfcpu stamp add -mode pdf -- "stamp.pdf:m"   "" in.pdf out.pdf ... single stamp using page n of source for selected pages of in.pdf

			pdfcpu stamp add -mode pdf -- "stamp.pdf"     "" in.pdf out.pdf ... multi stamp starting at the beginning of source and dest

			pdfcpu stamp add -mode pdf -- "stamp.pdf:m:n" "" in.pdf out.pdf ... multi stamp starting at source page m and dest page n

		*/
		return nil
	}
	i := strings.LastIndex(s, ":")
	if i < 1 {
		// No colon => multi stamp
		if strings.ToLower(filepath.Ext(s)) != ".pdf" {
			return errors.Errorf("%s is not a PDF file", s)
		}
		wm.FileName = s
		return nil
	}
	// We have at least one Colon.
	if strings.ToLower(filepath.Ext(s)) == ".pdf" {
		// We have an absolute DOS filename eg. C:\test.pdf => multi stamp
		wm.FileName = s
		return nil
	}

	pageNumberStr := s[i+1:]
	j, err := strconv.Atoi(pageNumberStr)
	if err != nil {
		return errors.Errorf("unable to detect PDF page number: %s\n", pageNumberStr)
	}

	s = s[:i]
	i = strings.LastIndex(s, ":")
	if i < 1 {
		// single stamp
		wm.PdfPageNrSrc = j
		if strings.ToLower(filepath.Ext(s)) != ".pdf" {
			return errors.Errorf("%s is not a PDF file", s)
		}
		wm.FileName = s
		return nil
	}

	// multi stamp

	wm.PdfMultiStartPageNrDest = j
	pageNumberStr = s[i+1:]
	wm.PdfMultiStartPageNrSrc, err = strconv.Atoi(pageNumberStr)
	if err != nil {
		return errors.Errorf("unable to detect PDF page number: %s\n", pageNumberStr)
	}

	s = s[:i]
	if strings.ToLower(filepath.Ext(s)) != ".pdf" {
		return errors.Errorf("%s is not a PDF file", s)
	}
	wm.FileName = s

	return nil
}

func setWatermarkType(mode int, s string, wm *model.Watermark) (err error) {
	wm.Mode = mode
	switch wm.Mode {
	case model.WMText:
		setTextWatermark(s, wm)

	case model.WMImage:
		err = setImageWatermark(s, wm)

	case model.WMPDF:
		err = setPDFWatermark(s, wm)
	}
	return err
}

func createPDFRes(ctx, otherCtx *model.Context, pageNrSrc, pageNrDest int, migrated map[int]int, wm *model.Watermark) error {
	pdfRes := model.PdfResources{}
	xRefTable := ctx.XRefTable
	otherXRefTable := otherCtx.XRefTable

	// Locate page dict & resource dict of PDF stamp.
	consolidateRes := true
	d, _, inhPAttrs, err := otherXRefTable.PageDict(pageNrSrc, consolidateRes)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.Errorf("pdfcpu: unknown page number: %d\n", pageNrSrc)
	}

	// Retrieve content stream bytes of page dict.
	pdfRes.Content, err = otherXRefTable.PageContent(d)
	if err != nil {
		return err
	}

	// Migrate external resource dict into ctx.
	if _, err = migrateObject(inhPAttrs.Resources, otherCtx, ctx, migrated); err != nil {
		return err
	}

	// Create an object for resource dict in xRefTable.
	if inhPAttrs.Resources != nil {
		ir, err := xRefTable.IndRefForNewObject(inhPAttrs.Resources)
		if err != nil {
			return err
		}
		pdfRes.ResDict = ir
	}

	pdfRes.Bb = viewPort(inhPAttrs)
	wm.PdfRes[pageNrDest] = pdfRes

	return nil
}

func createPDFResForWM(ctx *model.Context, wm *model.Watermark) error {
	// Note: The stamp pdf is assumed to be valid!
	var (
		otherCtx *model.Context
		err      error
	)
	if wm.PDF != nil {
		otherCtx, err = Read(wm.PDF, nil)
	} else {
		otherCtx, err = ReadFile(wm.FileName, nil)
	}
	if err != nil {
		return err
	}
	if otherCtx.Version() == model.V20 {
		return ErrUnsupportedVersion
	}

	if err := otherCtx.EnsurePageCount(); err != nil {
		return nil
	}

	migrated := map[int]int{}

	if !wm.MultiStamp() {
		return createPDFRes(ctx, otherCtx, wm.PdfPageNrSrc, wm.PdfPageNrSrc, migrated, wm)
	}

	j := otherCtx.PageCount
	if ctx.PageCount < otherCtx.PageCount {
		j = ctx.PageCount
	}

	destPageNr := wm.PdfMultiStartPageNrDest
	for srcPageNr := wm.PdfMultiStartPageNrSrc; srcPageNr <= j; srcPageNr++ {
		if err := createPDFRes(ctx, otherCtx, srcPageNr, destPageNr, migrated, wm); err != nil {
			return err
		}
		destPageNr++
	}

	return nil
}

func createImageResForWM(ctx *model.Context, wm *model.Watermark) (err error) {
	wm.Img, wm.Width, wm.Height, err = model.CreateImageResource(ctx.XRefTable, wm.Image, false, false)
	return err
}

func createFontResForWM(ctx *model.Context, wm *model.Watermark) (err error) {
	// TODO Reuse font dict.
	if font.IsUserFont(wm.FontName) {
		td, _ := setupTextDescriptor(*wm, "", 123456789, 0)
		model.WriteMultiLine(ctx.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	}
	wm.Font, err = pdffont.EnsureFontDict(ctx.XRefTable, wm.FontName, "", "", true, false, nil)
	return err
}

func createResourcesForWM(ctx *model.Context, wm *model.Watermark) error {
	if wm.IsPDF() {
		return createPDFResForWM(ctx, wm)
	}
	if wm.IsImage() {
		return createImageResForWM(ctx, wm)
	}
	return createFontResForWM(ctx, wm)
}

func ensureOCG(ctx *model.Context, onTop bool) (*types.IndirectRef, error) {
	name := "Background"
	subt := "BG"
	if onTop {
		name = "Watermark"
		subt = "FG"
	}

	d := types.Dict(
		map[string]types.Object{
			"Name": types.StringLiteral(name),
			"Type": types.Name("OCG"),
			"Usage": types.Dict(
				map[string]types.Object{
					"PageElement": types.Dict(map[string]types.Object{"Subtype": types.Name(subt)}),
					"View":        types.Dict(map[string]types.Object{"ViewState": types.Name("ON")}),
					"Print":       types.Dict(map[string]types.Object{"PrintState": types.Name("ON")}),
					"Export":      types.Dict(map[string]types.Object{"ExportState": types.Name("ON")}),
				},
			),
		},
	)

	return ctx.IndRefForNewObject(d)
}

func prepareOCPropertiesInRoot(ctx *model.Context, onTop bool) (*types.IndirectRef, error) {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, err
	}

	if o, ok := rootDict.Find("OCProperties"); ok {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return nil, err
		}

		o, found := d.Find("OCGs")
		if found {
			a, err := ctx.DereferenceArray(o)
			if err != nil {
				return nil, errCorruptOCGs
			}
			if len(a) > 0 {
				ir, ok := a[0].(types.IndirectRef)
				if !ok {
					return nil, errCorruptOCGs
				}
				return &ir, nil
			}
		}
	}

	ir, err := ensureOCG(ctx, onTop)
	if err != nil {
		return nil, err
	}

	optionalContentConfigDict := types.Dict(
		map[string]types.Object{
			"AS": types.Array{
				types.Dict(
					map[string]types.Object{
						"Category": types.NewNameArray("View"),
						"Event":    types.Name("View"),
						"OCGs":     types.Array{*ir},
					},
				),
				types.Dict(
					map[string]types.Object{
						"Category": types.NewNameArray("Print"),
						"Event":    types.Name("Print"),
						"OCGs":     types.Array{*ir},
					},
				),
				types.Dict(
					map[string]types.Object{
						"Category": types.NewNameArray("Export"),
						"Event":    types.Name("Export"),
						"OCGs":     types.Array{*ir},
					},
				),
			},
			"ON":       types.Array{*ir},
			"Order":    types.Array{},
			"RBGroups": types.Array{},
		},
	)

	d := types.Dict(
		map[string]types.Object{
			"OCGs": types.Array{*ir},
			"D":    optionalContentConfigDict,
		},
	)

	rootDict.Update("OCProperties", d)
	return ir, nil
}

func createFormResDict(ctx *model.Context, pageNr int, wm *model.Watermark) (*types.IndirectRef, error) {
	if wm.IsPDF() {
		i := wm.PdfResIndex(pageNr)
		return wm.PdfRes[i].ResDict, nil
	}

	if wm.IsImage() {
		d := types.Dict(
			map[string]types.Object{
				"ProcSet": types.NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
				"XObject": types.Dict(map[string]types.Object{"Im0": *wm.Img}),
			},
		)
		return ctx.IndRefForNewObject(d)
	}

	d := types.Dict(
		map[string]types.Object{
			"Font":    types.Dict(map[string]types.Object{"F1": *wm.Font}),
			"ProcSet": types.NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
		},
	)

	return ctx.IndRefForNewObject(d)
}

func cachedForm(wm model.Watermark) bool {
	return !wm.IsPDF() || !wm.MultiStamp()
}

func pdfFormContent(w io.Writer, pageNr int, wm model.Watermark) error {
	i := wm.PdfResIndex(pageNr)
	cs := wm.PdfRes[i].Content

	sc := wm.Scale
	if !wm.ScaleAbs {
		sc = wm.Bb.Width() / float64(wm.Width)
	}

	// Scale & translate into origin

	m1 := matrix.IdentMatrix
	m1[0][0] = sc
	m1[1][1] = sc

	m2 := matrix.IdentMatrix
	m2[2][0] = -wm.Bb.LL.X * wm.ScaleEff
	m2[2][1] = -wm.Bb.LL.Y * wm.ScaleEff

	m := m1.Multiply(m2)

	fmt.Fprintf(w, "%.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	_, err := w.Write(cs)
	return err
}

func imageFormContent(w io.Writer, wm model.Watermark) {
	fmt.Fprintf(w, "q %f 0 0 %f 0 0 cm /Im0 Do Q", wm.Bb.Width(), wm.Bb.Height()) // TODO dont need Q
}

func formContent(w io.Writer, pageNr int, wm model.Watermark) error {
	switch true {
	case wm.IsPDF():
		return pdfFormContent(w, pageNr, wm)
	case wm.IsImage():
		imageFormContent(w, wm)
	}
	return nil
}

func setupTextDescriptor(wm model.Watermark, timestampFormat string, pageNr, pageCount int) (model.TextDescriptor, bool) {
	// Set horizontal alignment.
	var hAlign types.HAlignment
	if wm.HAlign == nil {
		// Use alignment implied by anchor.
		_, _, hAlign, _ = model.AnchorPosAndAlign(wm.Pos, types.RectForDim(0, 0))
	} else {
		// Use manual alignment.
		hAlign = *wm.HAlign
	}

	// Set effective position and vertical alignment.
	x, y, _, vAlign := model.AnchorPosAndAlign(types.BottomLeft, wm.Vp)
	td, unique := textDescriptor(wm, timestampFormat, pageNr, pageCount)
	td.X, td.Y, td.HAlign, td.VAlign, td.FontKey = x, y, hAlign, vAlign, "F1"

	// Set right to left rendering.
	td.RTL = wm.RTL

	// Set margins.
	td.MLeft = wm.MLeft
	td.MRight = wm.MRight
	td.MTop = wm.MTop
	td.MBot = wm.MBot

	// Set border.
	td.BorderWidth = wm.BorderWidth
	td.BorderStyle = wm.BorderStyle
	if wm.BorderColor != nil {
		td.ShowBorder = true
		td.BorderCol = *wm.BorderColor
	}
	return td, unique
}

func drawBoundingBox(b *bytes.Buffer, wm model.Watermark, bb *types.Rectangle) {
	urx := bb.UR.X
	ury := bb.UR.Y
	if wm.IsPDF() {
		sc := wm.Scale
		if !wm.ScaleAbs {
			sc = bb.Width() / float64(wm.Width)
		}
		urx /= sc
		ury /= sc
	}
	fmt.Fprintf(b, "[]0 d 2 w %.2f %.2f m %.2f %.2f l %.2f %.2f l %.2f %.2f l s ",
		bb.LL.X, bb.LL.Y,
		urx, bb.LL.Y,
		urx, ury,
		bb.LL.X, ury,
	)
}

func calcFormBoundingBox(xRefTable *model.XRefTable, w io.Writer, timestampFormat string, pageNr, pageCount int, wm *model.Watermark) bool {
	var unique bool
	if wm.IsImage() || wm.IsPDF() {
		wm.CalcBoundingBox(pageNr)
	} else {
		var td model.TextDescriptor
		td, unique = setupTextDescriptor(*wm, timestampFormat, pageNr, pageCount)
		// Render td into b and return the bounding box.
		wm.Bb = model.WriteMultiLine(xRefTable, w, types.RectForDim(wm.Vp.Width(), wm.Vp.Height()), nil, td)
	}
	return unique
}

func createForm(ctx *model.Context, pageNr, pageCount int, wm *model.Watermark, withBB bool) error {
	var b bytes.Buffer
	unique := calcFormBoundingBox(ctx.XRefTable, &b, ctx.Configuration.TimestampFormat, pageNr, pageCount, wm)

	// The forms bounding box is dependent on the page dimensions.
	bb := wm.Bb

	maxStampPageNr := wm.PdfMultiStartPageNrSrc + len(wm.PdfRes) - 1

	if !unique && (cachedForm(*wm) || pageNr > maxStampPageNr) {
		// Use cached form.
		ir, ok := wm.FCache[*bb]
		if ok {
			wm.Form = ir
			return nil
		}
	}

	if wm.IsImage() || wm.IsPDF() {
		if err := formContent(&b, pageNr, *wm); err != nil {
			return err
		}
	}

	ir, err := createFormResDict(ctx, pageNr, wm)
	if err != nil {
		return err
	}

	bbox := bb.CroppedCopy(0)
	bbox.Translate(-bb.LL.X, -bb.LL.Y)

	// Paint bounding box
	if withBB {
		drawBoundingBox(&b, *wm, bbox)
	}

	sd := types.StreamDict{
		Dict: types.Dict(
			map[string]types.Object{
				"Type":    types.Name("XObject"),
				"Subtype": types.Name("Form"),
				"BBox":    bbox.Array(),
				"Matrix":  types.NewNumberArray(1, 0, 0, 1, 0, 0),
				"OC":      *wm.Ocg,
			},
		),
		Content:        b.Bytes(),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	if ir != nil {
		sd.Insert("Resources", *ir)
	}

	sd.InsertName("Filter", filter.Flate)

	if err = sd.Encode(); err != nil {
		return err
	}

	ir, err = ctx.IndRefForNewObject(sd)
	if err != nil {
		return err
	}

	wm.Form = ir

	if cachedForm(*wm) || pageNr >= len(wm.PdfRes) {
		// Cache form.
		wm.FCache[*wm.Bb] = ir
	}

	return nil
}

func createExtGStateForStamp(ctx *model.Context, opacity float64) (*types.IndirectRef, error) {
	d := types.Dict(
		map[string]types.Object{
			"Type": types.Name("ExtGState"),
			"CA":   types.Float(opacity),
			"ca":   types.Float(opacity),
		},
	)

	return ctx.IndRefForNewObject(d)
}

func insertPageResourcesForWM(ctx *model.Context, pageDict types.Dict, wm model.Watermark, gsID, xoID string) error {
	resourceDict := types.Dict(
		map[string]types.Object{
			"ExtGState": types.Dict(map[string]types.Object{gsID: *wm.ExtGState}),
			"XObject":   types.Dict(map[string]types.Object{xoID: *wm.Form}),
		},
	)

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func updatePageResourcesForWM(ctx *model.Context, resDict types.Dict, wm model.Watermark, gsID, xoID *string) error {
	o, ok := resDict.Find("ExtGState")
	if !ok {
		resDict.Insert("ExtGState", types.Dict(map[string]types.Object{*gsID: *wm.ExtGState}))
	} else {
		d, _ := ctx.DereferenceDict(o)
		for i := 0; i < 1000; i++ {
			*gsID = "GS" + strconv.Itoa(i)
			if _, found := d.Find(*gsID); !found {
				break
			}
		}
		d.Insert(*gsID, *wm.ExtGState)
	}

	o, ok = resDict.Find("XObject")
	if !ok {
		resDict.Insert("XObject", types.Dict(map[string]types.Object{*xoID: *wm.Form}))
	} else {
		d, _ := ctx.DereferenceDict(o)
		for i := 0; i < 1000; i++ {
			*xoID = "Fm" + strconv.Itoa(i)
			if _, found := d.Find(*xoID); !found {
				break
			}
		}
		d.Insert(*xoID, *wm.Form)
	}

	return nil
}

func wmContent(wm model.Watermark, gsID, xoID string) []byte {
	m := wm.CalcTransformMatrix()
	p1 := m.Transform(types.Point{X: wm.Bb.LL.X, Y: wm.Bb.LL.Y})
	p2 := m.Transform(types.Point{X: wm.Bb.UR.X, Y: wm.Bb.LL.Y})
	p3 := m.Transform(types.Point{X: wm.Bb.UR.X, Y: wm.Bb.UR.Y})
	p4 := m.Transform(types.Point{X: wm.Bb.LL.X, Y: wm.Bb.UR.Y})
	wm.BbTrans = types.QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}
	insertOCG := " /Artifact <</Subtype /Watermark /Type /Pagination >>BDC q %.5f %.5f %.5f %.5f %.5f %.5f cm /%s gs /%s Do Q EMC "
	var b bytes.Buffer
	fmt.Fprintf(&b, insertOCG, m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], gsID, xoID)
	return b.Bytes()
}

func insertPageContentsForWM(ctx *model.Context, pageDict types.Dict, wm model.Watermark, gsID, xoID string) error {
	sd, _ := ctx.NewStreamDictForBuf(wmContent(wm, gsID, xoID))
	if err := sd.Encode(); err != nil {
		return err
	}

	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *ir)

	return nil
}

func patchFirstContentStreamForWatermark(sd *types.StreamDict, gsID, xoID string, wm model.Watermark, isLast bool) error {
	err := sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		if log.InfoEnabled() {
			log.Info.Println("unsupported filter: unable to patch content with watermark.")
		}
		return nil
	}
	if err != nil {
		return err
	}

	wmbb := wmContent(wm, gsID, xoID)

	// stamp
	if wm.OnTop {
		bb := []byte(" q ")
		if wm.PageRot != 0 {
			bb = append(bb, model.ContentBytesForPageRotation(wm.PageRot, wm.Vp.Width(), wm.Vp.Height())...)
		}
		sd.Content = append(bb, sd.Content...)
		if !isLast {
			return sd.Encode()
		}
		sd.Content = append(sd.Content, []byte(" Q ")...)
		sd.Content = append(sd.Content, wmbb...)
		return sd.Encode()
	}

	// watermark
	if wm.PageRot == 0 {
		sd.Content = append(wmbb, sd.Content...)
		return sd.Encode()
	}

	bb := append([]byte(" q "), model.ContentBytesForPageRotation(wm.PageRot, wm.Vp.Width(), wm.Vp.Height())...)
	sd.Content = append(bb, sd.Content...)
	if isLast {
		sd.Content = append(sd.Content, []byte(" Q")...)
	}
	return sd.Encode()
}

func patchLastContentStreamForWatermark(sd *types.StreamDict, gsID, xoID string, wm model.Watermark) error {
	err := sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		if log.InfoEnabled() {
			log.Info.Println("unsupported filter: unable to patch content with watermark.")
		}
		return nil
	}
	if err != nil {
		return err
	}

	// stamp
	if wm.OnTop {
		sd.Content = append(sd.Content, []byte(" Q ")...)
		sd.Content = append(sd.Content, wmContent(wm, gsID, xoID)...)
		return sd.Encode()
	}

	// watermark
	if wm.PageRot != 0 {
		sd.Content = append(sd.Content, []byte(" Q")...)
		return sd.Encode()
	}

	return nil
}

func updatePageContentsForWM(ctx *model.Context, obj types.Object, wm model.Watermark, gsID, xoID string) error {
	var entry *model.XRefTableEntry
	var objNr int

	ir, ok := obj.(types.IndirectRef)
	if ok {
		objNr = ir.ObjectNumber.Value()
		if wm.Objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}
		genNr := ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		obj = entry.Object
	}

	switch o := obj.(type) {

	case types.StreamDict:

		err := patchFirstContentStreamForWatermark(&o, gsID, xoID, wm, true)
		if err != nil {
			return err
		}

		entry.Object = o
		wm.Objs[objNr] = true

	case types.Array:

		// Get stream dict for first array element.
		o1 := o[0]
		ir, _ := o1.(types.IndirectRef)
		objNr = ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(types.StreamDict)

		if wm.Objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}

		err := patchFirstContentStreamForWatermark(&sd, gsID, xoID, wm, len(o) == 1)
		if err != nil {
			return err
		}

		entry.Object = sd
		wm.Objs[objNr] = true
		if len(o) == 1 {
			return nil
		}

		// Get stream dict for last array element.
		o1 = o[len(o)-1]

		ir, _ = o1.(types.IndirectRef)
		objNr = ir.ObjectNumber.Value()
		if wm.Objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}

		genNr = ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		sd, _ = (entry.Object).(types.StreamDict)

		err = patchLastContentStreamForWatermark(&sd, gsID, xoID, wm)
		if err != nil {
			return err
		}

		entry.Object = sd
		wm.Objs[objNr] = true
	}

	return nil
}

func viewPort(a *model.InheritedPageAttrs) *types.Rectangle {
	visibleRegion := a.MediaBox
	if a.CropBox != nil {
		visibleRegion = a.CropBox
	}
	return visibleRegion
}

func handleLink(ctx *model.Context, pageIndRef *types.IndirectRef, d types.Dict, pageNr int, wm model.Watermark) error {
	if !wm.OnTop || wm.URL == "" {
		return nil
	}

	ann := model.NewLinkAnnotation(
		*wm.BbTrans.EnclosingRectangle(5.0),
		types.QuadPoints{wm.BbTrans},
		nil,
		wm.URL,
		"pdfcpu",
		model.AnnNoZoom+model.AnnNoRotate,
		nil,
		false)

	_, err := AddAnnotation(ctx, pageIndRef, d, pageNr, ann, false)

	return err
}

func addPageWatermark(ctx *model.Context, pageNr int, wm model.Watermark) error {
	if pageNr > ctx.PageCount {
		return errors.Errorf("pdfcpu: invalid page number: %d", pageNr)
	}

	if log.DebugEnabled() {
		log.Debug.Printf("addPageWatermark page:%d\n", pageNr)
	}

	if wm.Update {
		if log.DebugEnabled() {
			log.Debug.Println("Updating")
		}
		if _, err := removePageWatermark(ctx, pageNr); err != nil {
			return err
		}
	}

	consolidateRes := false
	d, pageIndRef, inhPAttrs, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return err
	}

	// Internalize page rotation into content stream.
	wm.PageRot = inhPAttrs.Rotate

	wm.Vp = viewPort(inhPAttrs)

	// Reset page rotation in page dict.
	if wm.PageRot != 0 {
		if types.IntMemberOf(wm.PageRot, []int{+90, -90, +270, -270}) {
			w := wm.Vp.Width()
			wm.Vp.UR.X = wm.Vp.LL.X + wm.Vp.Height()
			wm.Vp.UR.Y = wm.Vp.LL.Y + w
		}
		d.Update("MediaBox", wm.Vp.Array())
		d.Update("CropBox", wm.Vp.Array())
		d.Delete("Rotate")
	}

	if err = createForm(ctx, pageNr, ctx.PageCount, &wm, stampWithBBox); err != nil {
		return err
	}

	if log.DebugEnabled() {
		log.Debug.Printf("\n%s\n", wm)
	}

	gsID := "GS0"
	xoID := "Fm0"

	if inhPAttrs.Resources != nil {
		err = updatePageResourcesForWM(ctx, inhPAttrs.Resources, wm, &gsID, &xoID)
		d.Update("Resources", inhPAttrs.Resources)
	} else {
		err = insertPageResourcesForWM(ctx, d, wm, gsID, xoID)
	}
	if err != nil {
		return err
	}

	obj, found := d.Find("Contents")
	if found {
		err = updatePageContentsForWM(ctx, obj, wm, gsID, xoID)
	} else {
		err = insertPageContentsForWM(ctx, d, wm, gsID, xoID)
	}
	if err != nil {
		return err
	}

	return handleLink(ctx, pageIndRef, d, pageNr, wm)
}

func createResourcesForPageNr(
	ctx *model.Context,
	wm *model.Watermark,
	pageNr int,
	fm map[string]types.IntSet,
	ocgIndRef, extGStateIndRef *types.IndirectRef,
	onTop bool, opacity float64) error {

	wm.Ocg = ocgIndRef
	wm.ExtGState = extGStateIndRef
	wm.OnTop = onTop
	wm.Opacity = opacity

	if wm.IsImage() {
		return createImageResForWM(ctx, wm)
	}

	if wm.IsPDF() {
		return createPDFResForWM(ctx, wm)
	}

	// Text watermark

	if font.IsUserFont(wm.FontName) {
		td, _ := setupTextDescriptor(*wm, "", 123456789, 0)
		model.WriteMultiLine(ctx.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	}

	pageSet, found := fm[wm.FontName]
	if !found {
		fm[wm.FontName] = types.IntSet{pageNr: true}
	} else {
		pageSet[pageNr] = true
	}

	return nil
}

func createResourcesForWMMap(
	ctx *model.Context,
	m map[int]*model.Watermark,
	ocgIndRef, extGStateIndRef *types.IndirectRef,
	onTop bool,
	opacity float64) (map[string]types.IntSet, error) {

	fm := map[string]types.IntSet{}
	for pageNr, wm := range m {
		if err := createResourcesForPageNr(ctx, wm, pageNr, fm, ocgIndRef, extGStateIndRef, onTop, opacity); err != nil {
			return nil, err
		}
	}

	return fm, nil
}

func createResourcesForWMSliceMap(
	ctx *model.Context,
	m map[int][]*model.Watermark,
	ocgIndRef, extGStateIndRef *types.IndirectRef,
	onTop bool,
	opacity float64) (map[string]types.IntSet, error) {

	fm := map[string]types.IntSet{}
	for pageNr, wms := range m {
		for _, wm := range wms {
			if err := createResourcesForPageNr(ctx, wm, pageNr, fm, ocgIndRef, extGStateIndRef, onTop, opacity); err != nil {
				return nil, err
			}
		}
	}

	return fm, nil
}

// AddWatermarksMap adds watermarks in m to corresponding pages.
func AddWatermarksMap(ctx *model.Context, m map[int]*model.Watermark) error {
	var (
		onTop   bool
		opacity float64
	)
	for _, wm := range m {
		onTop = wm.OnTop
		opacity = wm.Opacity
		break
	}

	ocgIndRef, err := prepareOCPropertiesInRoot(ctx, onTop)
	if err != nil {
		return err
	}

	extGStateIndRef, err := createExtGStateForStamp(ctx, opacity)
	if err != nil {
		return err
	}

	fm, err := createResourcesForWMMap(ctx, m, ocgIndRef, extGStateIndRef, onTop, opacity)
	if err != nil {
		return err
	}

	// TODO Reuse font dict.
	for fontName, pageSet := range fm {
		ir, err := pdffont.EnsureFontDict(ctx.XRefTable, fontName, "", "", true, false, nil)
		if err != nil {
			return err
		}
		for pageNr, v := range pageSet {
			if !v {
				continue
			}
			wm := m[pageNr]
			if wm.IsText() && wm.FontName == fontName {
				m[pageNr].Font = ir
			}
		}
	}

	for k, wm := range m {
		if err := addPageWatermark(ctx, k, *wm); err != nil {
			return err
		}
	}

	ctx.EnsureVersionForWriting()
	return nil
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages.
func AddWatermarksSliceMap(ctx *model.Context, m map[int][]*model.Watermark) error {
	var (
		onTop   bool
		opacity float64
	)
	for _, wms := range m {
		onTop = wms[0].OnTop
		opacity = wms[0].Opacity
		break
	}

	ocgIndRef, err := prepareOCPropertiesInRoot(ctx, onTop)
	if err != nil {
		return err
	}

	extGStateIndRef, err := createExtGStateForStamp(ctx, opacity)
	if err != nil {
		return err
	}

	fm, err := createResourcesForWMSliceMap(ctx, m, ocgIndRef, extGStateIndRef, onTop, opacity)
	if err != nil {
		return err
	}

	// TODO Take existing font dicts in xref into account.
	for fontName, pageSet := range fm {
		ir, err := pdffont.EnsureFontDict(ctx.XRefTable, fontName, "", "", true, false, nil)
		if err != nil {
			return err
		}
		for pageNr, v := range pageSet {
			if !v {
				continue
			}
			for _, wm := range m[pageNr] {
				if wm.IsText() && wm.FontName == fontName {
					wm.Font = ir
				}
			}
		}
	}

	for k, wms := range m {
		for _, wm := range wms {
			if err := addPageWatermark(ctx, k, *wm); err != nil {
				return err
			}
		}
	}

	ctx.EnsureVersionForWriting()
	return nil
}

// AddWatermarks adds watermarks to all pages selected.
func AddWatermarks(ctx *model.Context, selectedPages types.IntSet, wm *model.Watermark) error {
	if log.DebugEnabled() {
		log.Debug.Printf("AddWatermarks wm:\n%s\n", wm)
	}
	var err error
	if wm.Ocg, err = prepareOCPropertiesInRoot(ctx, wm.OnTop); err != nil {
		return err
	}

	if err = createResourcesForWM(ctx, wm); err != nil {
		return err
	}

	if wm.ExtGState, err = createExtGStateForStamp(ctx, wm.Opacity); err != nil {
		return err
	}

	// if len(selectedPages) == 0 {
	// 	selectedPages = types.IntSet{}
	// 	for i := wm.PdfMultiStartPageNrDest; i <= ctx.PageCount; i++ {
	// 		selectedPages[i] = true
	// 	}
	// } else {
	// 	for k, v := range selectedPages {
	// 		if v && k < wm.PdfMultiStartPageNrDest {
	// 			selectedPages[k] = false
	// 		}
	// 	}
	// }

	// for k, v := range selectedPages {
	// 	if v {
	// 		if err = addPageWatermark(ctx, k, *wm); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	for i := wm.PdfMultiStartPageNrDest; i <= ctx.PageCount; i++ {
		if len(selectedPages) == 0 || selectedPages[i] {
			if err = addPageWatermark(ctx, i, *wm); err != nil {
				return err
			}
		}
	}

	ctx.EnsureVersionForWriting()
	return nil
}

func removeResDictEntry(ctx *model.Context, d types.Dict, entry string, ids []string, i int) error {
	o, ok := d.Find(entry)
	if !ok {
		return errors.Errorf("pdfcpu: page %d: corrupt resource dict", i)
	}

	d1, err := ctx.DereferenceDict(o)
	if err != nil {
		return err
	}

	for _, id := range ids {
		o, ok := d1.Find(id)
		if ok {
			err = ctx.DeleteObject(o)
			if err != nil {
				return err
			}
			d1.Delete(id)
		}
	}

	if d1.Len() == 0 {
		d.Delete(entry)
	}

	return nil
}

func removeExtGStates(ctx *model.Context, d types.Dict, ids []string, i int) error {
	return removeResDictEntry(ctx, d, "ExtGState", ids, i)
}

func removeForms(ctx *model.Context, d types.Dict, ids []string, i int) error {
	return removeResDictEntry(ctx, d, "XObject", ids, i)
}

func removeArtifacts(sd *types.StreamDict, i int) (ok bool, extGStates []string, forms []string, err error) {
	err = sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		if log.InfoEnabled() {
			log.Info.Printf("unsupported filter: unable to patch content with watermark for page %d\n", i)
		}
		return false, nil, nil, nil
	}
	if err != nil {
		return false, nil, nil, err
	}

	var patched bool

	// Watermarks may begin or end the content stream.

	for {
		s := string(sd.Content)
		beg := strings.Index(s, "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC")
		if beg < 0 {
			break
		}

		end := strings.Index(s[beg:], "EMC")
		if end < 0 {
			break
		}

		// Check for usage of resources.
		t := s[beg : beg+end]

		i := strings.Index(t, "/GS")
		if i > 0 {
			j := i + 3
			k := strings.Index(t[j:], " gs")
			if k > 0 {
				extGStates = append(extGStates, "GS"+t[j:j+k])
			}
		}

		i = strings.Index(t, "/Fm")
		if i > 0 {
			j := i + 3
			k := strings.Index(t[j:], " Do")
			if k > 0 {
				forms = append(forms, "Fm"+t[j:j+k])
			}
		}

		// TODO Remove whitespace until 0x0a
		sd.Content = append(sd.Content[:beg], sd.Content[beg+end+3:]...)
		patched = true
	}

	if patched {
		err = sd.Encode()
	}

	return patched, extGStates, forms, err
}

func removeArtifactsFromPage(ctx *model.Context, sd *types.StreamDict, resDict types.Dict, i int) (bool, error) {
	// Remove watermark artifacts and locate id's
	// of used extGStates and forms.
	ok, extGStates, forms, err := removeArtifacts(sd, i)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// Remove obsolete extGStates from page resource dict.
	err = removeExtGStates(ctx, resDict, extGStates, i)
	if err != nil {
		return false, err
	}

	// Remove obsolete forms from page resource dict.
	return true, removeForms(ctx, resDict, forms, i)
}

func locatePageContentAndResourceDict(ctx *model.Context, pageNr int) (types.Object, *types.IndirectRef, types.Dict, error) {
	consolidateRes := false
	d, pageDictIndRef, _, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return nil, nil, nil, err
	}

	o, found := d.Find("Resources")
	if !found {
		return nil, nil, nil, errors.Errorf("pdfcpu: page %d: no resource dict found\n", pageNr)
	}

	resDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, nil, nil, err
	}

	o, found = d.Find("Contents")
	if !found {
		return nil, nil, nil, errors.Errorf("pdfcpu: page %d: no page watermark found", pageNr)
	}

	return o, pageDictIndRef, resDict, nil
}

func removeArtifacts1(ctx *model.Context, o types.Object, entry *model.XRefTableEntry, resDict types.Dict, pageNr int) (bool, error) {
	found := false
	switch o := o.(type) {

	case types.StreamDict:
		ok, err := removeArtifactsFromPage(ctx, &o, resDict, pageNr)
		if err != nil {
			return false, err
		}
		if !found && ok {
			found = true
		}
		entry.Object = o

	case types.Array:
		// Get stream dict for first element.
		o1 := o[0]
		ir, _ := o1.(types.IndirectRef)
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(types.StreamDict)

		ok, err := removeArtifactsFromPage(ctx, &sd, resDict, pageNr)
		if err != nil {
			return false, err
		}
		if !found && ok {
			found = true
			entry.Object = sd
		}

		if len(o) > 1 {
			// Get stream dict for last element.
			o1 := o[len(o)-1]
			ir, _ := o1.(types.IndirectRef)
			objNr = ir.ObjectNumber.Value()
			genNr := ir.GenerationNumber.Value()
			entry, _ := ctx.FindTableEntry(objNr, genNr)
			sd, _ := (entry.Object).(types.StreamDict)

			ok, err = removeArtifactsFromPage(ctx, &sd, resDict, pageNr)
			if err != nil {
				return false, err
			}
			if !found && ok {
				found = true
				entry.Object = sd
			}
		}

	}
	return found, nil
}

func removePageWatermark(ctx *model.Context, pageNr int) (bool, error) {
	o, pageDictIndRef, resDict, err := locatePageContentAndResourceDict(ctx, pageNr)
	if err != nil {
		return false, err
	}

	var entry *model.XRefTableEntry

	ir, ok := o.(types.IndirectRef)
	if ok {
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		o = entry.Object
	}

	found, err := removeArtifacts1(ctx, o, entry, resDict, pageNr)
	if err != nil {
		return false, err
	}

	/*
		Supposedly the form needs a PieceInfo in order to be recognized by Acrobat like so:

			<PieceInfo, <<
				<ADBE_CompoundType, <<
					<DocSettings, (61 0 R)>
					<LastModified, (D:20190830152436+02'00')>
					<Private, Watermark>
				>>>
			>>>

	*/

	if found {
		// Remove any associated link annotations.
		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return false, err
		}
		objNr := pageDictIndRef.ObjectNumber.Value()
		if _, err = RemoveAnnotationsFromPageDict(ctx, nil, []string{"pdfcpu"}, nil, d, objNr, pageNr, false); err != nil {
			return false, err
		}
	}

	return found, nil
}

func locateOCGs(ctx *model.Context) (types.Array, error) {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, err
	}

	o, ok := rootDict.Find("OCProperties")
	if !ok {
		return nil, errNoWatermark
	}

	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	o, found := d.Find("OCGs")
	if !found {
		return nil, errNoWatermark
	}

	return ctx.DereferenceArray(o)
}

func detectStampOCG(ctx *model.Context, arr types.Array) error {
	for _, o := range arr {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		if *d.Type() != "OCG" {
			continue
		}

		n := d.StringEntry("Name")
		if n == nil {
			continue
		}

		if *n != "Background" && *n != "Watermark" {
			continue
		}

		return nil
	}

	return errNoWatermark
}

func removePageWatermarks(ctx *model.Context, selectedPages types.IntSet) error {
	var removed bool

	for k, v := range selectedPages {

		if !v {
			continue
		}

		ok, err := removePageWatermark(ctx, k)
		if err != nil {
			return err
		}

		if ok {
			removed = true
		}
	}

	if !removed {
		return errNoWatermark
	}

	return nil
}

// RemoveWatermarks removes watermarks for all pages selected.
func RemoveWatermarks(ctx *model.Context, selectedPages types.IntSet) error {
	if log.DebugEnabled() {
		log.Debug.Printf("RemoveWatermarks\n")
	}

	arr, err := locateOCGs(ctx)
	if err != nil {
		return err
	}

	if err := detectStampOCG(ctx, arr); err != nil {
		return err
	}

	return removePageWatermarks(ctx, selectedPages)
}

func detectArtifacts(sd *types.StreamDict) (bool, error) {
	if err := sd.Decode(); err != nil {
		return false, err
	}
	// Watermarks may begin or end the content stream.
	i := strings.Index(string(sd.Content), "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC")
	return i >= 0, nil
}

func findPageWatermarks(ctx *model.Context, pageDictIndRef *types.IndirectRef) (bool, error) {
	d, err := ctx.DereferenceDict(*pageDictIndRef)
	if err != nil {
		return false, err
	}

	o, found := d.Find("Contents")
	if !found {
		return false, model.ErrNoContent
	}

	var entry *model.XRefTableEntry

	ir, ok := o.(types.IndirectRef)
	if ok {
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		o = entry.Object
	}

	switch o := o.(type) {

	case types.StreamDict:
		return detectArtifacts(&o)

	case types.Array:
		// Get stream dict for first element.
		o1 := o[0]
		ir, _ := o1.(types.IndirectRef)
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(types.StreamDict)
		ok, err := detectArtifacts(&sd)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}

		if len(o) > 1 {
			// Get stream dict for last element.
			o1 := o[len(o)-1]
			ir, _ := o1.(types.IndirectRef)
			objNr = ir.ObjectNumber.Value()
			genNr := ir.GenerationNumber.Value()
			entry, _ := ctx.FindTableEntry(objNr, genNr)
			sd, _ := (entry.Object).(types.StreamDict)
			return detectArtifacts(&sd)
		}

	}

	return false, nil
}

func detectPageTreeWatermarks(ctx *model.Context, root *types.IndirectRef) error {
	d, err := ctx.DereferenceDict(*root)
	if err != nil {
		return err
	}

	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return nil
	}

	for _, o := range kids {

		if ctx.Watermarked {
			return nil
		}

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(types.IndirectRef)
		if !ok {
			return errors.Errorf("pdfcpu: detectPageTreeWatermarks: corrupt page node dict")
		}

		pageNodeDict, err := ctx.DereferenceDict(ir)
		if err != nil {
			return err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			if err := detectPageTreeWatermarks(ctx, &ir); err != nil {
				return err
			}

		case "Page":
			found, err := findPageWatermarks(ctx, &ir)
			if err != nil {
				return err
			}
			if found {
				ctx.Watermarked = true
				return nil
			}

		}
	}

	return nil
}

// DetectPageTreeWatermarks checks xRefTable's page tree for watermarks
// and records the result to xRefTable.Watermarked.
func DetectPageTreeWatermarks(ctx *model.Context) error {
	root, err := ctx.Pages()
	if err != nil {
		return err
	}
	return detectPageTreeWatermarks(ctx, root)
}

// DetectWatermarks checks ctx for watermarks
// and records the result to xRefTable.Watermarked.
func DetectWatermarks(ctx *model.Context) error {
	a, err := locateOCGs(ctx)
	if err != nil {
		if err == errNoWatermark {
			ctx.Watermarked = false
			return nil
		}
		return err
	}

	found := false

	for _, o := range a {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		if *d.Type() != "OCG" {
			continue
		}

		n := d.StringEntry("Name")
		if n == nil {
			continue
		}

		if *n != "Background" && *n != "Watermark" {
			continue
		}

		found = true
		break
	}

	if !found {
		ctx.Watermarked = false
		return nil
	}

	return DetectPageTreeWatermarks(ctx)
}

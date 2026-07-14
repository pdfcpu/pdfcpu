/*
Copyright 2025 The pdfcpu Authors.

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
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
)

const stampWithBBox = false

var (
	errNoWatermark = errors.New("no watermarks found")
	errCorruptOCGs = errors.New("OCProperties: corrupt OCGs element")
)

func textDescriptor(wm model.Watermark, timestampFormat string, pageNr, pageCount int) (model.TextDescriptor, bool) {
	t, unique := format.Text(wm.TextString, timestampFormat, pageNr, pageCount)
	td := model.TextDescriptor{
		Text:           t,
		FontName:       wm.FontName,
		FontSize:       float64(wm.FontSize),
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

var wmParamMap = parameterMap[model.Watermark]{
	"aligntext":       parseTextHorAlignment,
	"backgroundcolor": parseBackgroundColor,
	"bgcolor":         parseBackgroundColor,
	"border":          parseBorder,
	"color":           parseFillColor,
	"diagonal":        parseDiagonal,
	"fillcolor":       parseFillColor,
	"fontname":        parseFontName,
	"scriptname":      parseScriptName,
	"margins":         parseMargins,
	"maxWidth":        parseMaxWidth,
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
		return fmt.Errorf("unknown horizontal alignment (l,r,c,j): %s", s)
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
		return fmt.Errorf("illegal position offset string: need 2 numeric values, %s", s)
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
	supported, err := font.SupportedFont(s)
	if err != nil {
		return fmt.Errorf("font %s: load metrics: %w", s, err)
	}
	if !supported {
		return fmt.Errorf("%s is unsupported, please refer to \"pdfcpu fonts list\"", s)
	}
	wm.FontName = s
	if strings.HasSuffix(strings.ToUpper(wm.FontName), "GB2312") {
		wm.ScriptName = "HANS"
	}

	return nil
}

func parseScriptName(s string, wm *model.Watermark) error {
	script := strings.ToUpper(s)
	if !pdffont.SupportedScript(script) {
		return fmt.Errorf("unsupported font script \"%s\" - Supported are: HANS, HANT, HIRA, KANA, JPAN, HANG, KORE ", script)
	}

	wm.ScriptName = script

	return nil
}

func parseURL(s string, wm *model.Watermark) error {
	if !wm.OnTop {
		return fmt.Errorf("\"url\" supported for stamps only")
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

// parseMaxWidth parses the maximum column width for text wrapping.
// The value is interpreted in the current input display unit (InpUnit).
// When maxWidth > 0, text watermarks will automatically wrap text at word boundaries
// (including CJK character breaks) to fit within the specified width.
// A value of 0 (default) disables automatic text wrapping.
// Note: The effective font size is determined by points and scale parameters;
// maxWidth only controls where line breaks occur, not the font size itself.
func parseMaxWidth(s string, wm *model.Watermark) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	if f <= 0 {
		return errors.Errorf("pdfcpu: maxwidth must be a positive number: %s", s)
	}
	wm.MaxWidth = types.ToUserSpace(f, wm.InpUnit)
	return nil
}

func parseScaleFactor(s string) (float64, bool, error) {
	ss := strings.Split(s, " ")
	if len(ss) > 2 {
		return 0, false, fmt.Errorf("invalid factor string %s: 0.0 < i <= 1.0 {rel} | 0.0 < i {abs}", s)
	}

	sc, err := strconv.ParseFloat(ss[0], 64)
	if err != nil {
		return 0, false, fmt.Errorf("scale factor must be a float value %q: %w", ss[0], err)
	}

	if sc <= 0 {
		return 0, false, fmt.Errorf("invalid scale factor %.2f: 0.0 < i <= 1.0 {rel} | 0.0 < i {abs}", sc)
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
		return 0, false, fmt.Errorf("illegal scale mode: abs|rel, %s", ss[1])
	}

	if !scaleAbs && sc > 1 {
		return 0, false, fmt.Errorf("invalid relative scale factor %.2f: 0.0 < i <= 1", sc)
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
		return errors.New("rtl (right-to-left), please provide one of: on/off true/false t/f")
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
		return errors.New("please specify rotation or diagonal (r or d)")
	}

	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("rotation must be a float value: %s", s)
	}
	if r < -180 || r > 180 {
		return fmt.Errorf("illegal rotation: -180 <= r <= 180 degrees, %s", s)
	}

	wm.Rotation = r
	wm.Diagonal = model.NoDiagonal
	wm.UserRotOrDiagonal = true

	return nil
}

func parseDiagonal(s string, wm *model.Watermark) error {
	if wm.UserRotOrDiagonal {
		return errors.New("please specify rotation or diagonal (r or d)")
	}

	d, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("illegal diagonal value: allowed 1 or 2, %s", s)
	}
	if d != model.DiagonalLLToUR && d != model.DiagonalULToLR {
		return errors.New("diagonal: 1..lower left to upper right, 2..upper left to lower right")
	}

	wm.Diagonal = d
	wm.Rotation = 0
	wm.UserRotOrDiagonal = true

	return nil
}

func parseOpacity(s string, wm *model.Watermark) error {
	o, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("opacity must be a float value: %s", s)
	}
	if o < 0 || o > 1 {
		return fmt.Errorf("illegal opacity: 0.0 <= r <= 1.0, %s", s)
	}
	wm.Opacity = o

	return nil
}

func parseRenderMode(s string, wm *model.Watermark) error {
	m, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("illegal render mode value: allowed 0,1,2, %s", s)
	}
	rm := draw.RenderMode(m)
	if rm != draw.RMFill && rm != draw.RMStroke && rm != draw.RMFillAndStroke {
		return errors.New("valid rendermodes: 0..fill, 1..stroke, 2..fill&stroke")
	}
	wm.RenderMode = rm

	return nil
}

func parseMargins(s string, wm *model.Watermark) error {
	var err error

	m := strings.Split(s, " ")
	if len(m) == 0 || len(m) > 4 {
		return fmt.Errorf("margins: need 1,2,3 or 4 int values, %s", s)
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
		return fmt.Errorf("borders: need 1,2,3,4 or 5 int values, %s", s)
	}

	wm.BorderWidth, err = strconv.ParseFloat(b[0], 64)
	if err != nil {
		return err
	}
	if wm.BorderWidth == 0 {
		return errors.New("borders: need width > 0")
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

func watermarkModeParamName(mode int) string {
	switch mode {
	case model.WMText:
		return "text"
	case model.WMImage:
		return "image filename"
	case model.WMPDF:
		return "PDF filename"
	}
	return "mode parameter"
}

// ValidateWatermarkModeParam validates the user supplied watermark/stamp mode parameter.
func ValidateWatermarkModeParam(mode int, modeParm string, onTop bool) error {
	if strings.TrimSpace(modeParm) == "" {
		return fmt.Errorf("%s %s must not be empty", onTopString(onTop), watermarkModeParamName(mode))
	}
	return nil
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

		if err := handleParameter(wmParamMap, paramPrefix, paramValueStr, wm); err != nil {
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
	return fmt.Errorf("invalid %s configuration string", s)
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
			return fmt.Errorf("%s is not a PDF file", s)
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
		return fmt.Errorf("unable to detect PDF page number: %s", pageNumberStr)
	}

	s = s[:i]
	i = strings.LastIndex(s, ":")
	if i < 1 {
		// single stamp
		wm.PdfPageNrSrc = j
		if strings.ToLower(filepath.Ext(s)) != ".pdf" {
			return fmt.Errorf("%s is not a PDF file", s)
		}
		wm.FileName = s
		return nil
	}

	// multi stamp

	wm.PdfMultiStartPageNrDest = j
	pageNumberStr = s[i+1:]
	wm.PdfMultiStartPageNrSrc, err = strconv.Atoi(pageNumberStr)
	if err != nil {
		return fmt.Errorf("unable to detect PDF page number: %s", pageNumberStr)
	}

	s = s[:i]
	if strings.ToLower(filepath.Ext(s)) != ".pdf" {
		return fmt.Errorf("%s is not a PDF file", s)
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

func appearanceState(d, normalAppearanceDict types.Dict) string {
	if as := d.NameEntry("AS"); as != nil {
		if _, found := normalAppearanceDict.Find(*as); found {
			return *as
		}
	}

	for k := range normalAppearanceDict {
		if k != "Off" {
			return k
		}
	}

	return "Off"
}

func normalAppearanceObject(xRefTable *model.XRefTable, d types.Dict) (types.Object, bool, error) {
	o, found := d.Find("AP")
	if !found {
		return nil, false, nil
	}

	d1, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, false, err
	}

	o, found = d1.Find("N")
	if !found {
		return nil, false, nil
	}

	o1, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, false, err
	}

	normalAppearanceDict, ok := o1.(types.Dict)
	if !ok {
		_, ok = o1.(types.StreamDict)
		return o, ok, nil
	}

	o, found = normalAppearanceDict.Find(appearanceState(d, normalAppearanceDict))
	return o, found, nil
}

func ensureResourceDict(d types.Dict) types.Dict {
	if d != nil {
		return d
	}
	return types.Dict{}
}

func ensureXObjectResourceDict(ctx *model.Context, resDict types.Dict) (types.Dict, error) {
	o, found := resDict.Find("XObject")
	if !found {
		d := types.Dict{}
		resDict["XObject"] = d
		return d, nil
	}

	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, err
	}
	resDict["XObject"] = d
	return d, nil
}

func xObjectRefForAppearance(o types.Object, ctxSrc, ctxDest *model.Context, migrated map[int]int) (types.IndirectRef, error) {
	o, err := migrateObject(o, ctxSrc, ctxDest, migrated)
	if err != nil {
		return types.IndirectRef{}, err
	}

	if ir, ok := o.(types.IndirectRef); ok {
		return ir, nil
	}

	ir, err := ctxDest.IndRefForNewObject(o)
	if err != nil {
		return types.IndirectRef{}, err
	}
	return *ir, nil
}

func annotationRect(xRefTable *model.XRefTable, d types.Dict) (*types.Rectangle, error) {
	a, err := xRefTable.DereferenceArray(d["Rect"])
	if err != nil || len(a) != 4 {
		return nil, err
	}
	return xRefTable.RectForArray(a)
}

func appearanceBBox(xRefTable *model.XRefTable, ir types.IndirectRef) (*types.Rectangle, error) {
	sd, err := xRefTable.DereferenceXObjectDict(ir)
	if err != nil || sd == nil {
		return nil, err
	}

	a, err := xRefTable.DereferenceArray(sd.Dict["BBox"])
	if err != nil || len(a) != 4 {
		return nil, err
	}
	return xRefTable.RectForArray(a)
}

func appendAppearanceDo(w io.Writer, id string, rect, bbox *types.Rectangle) {
	sx := rect.Width() / bbox.Width()
	sy := rect.Height() / bbox.Height()
	tx := rect.LL.X - bbox.LL.X*sx
	ty := rect.LL.Y - bbox.LL.Y*sy
	fmt.Fprintf(w, " q %.5f 0 0 %.5f %.5f %.5f cm /%s Do Q ", sx, sy, tx, ty, id)
}

func appendAnnotationAppearance(
	w io.Writer,
	ann types.Dict,
	resDict types.Dict,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
) error {
	o, found, err := normalAppearanceObject(ctxSrc.XRefTable, ann)
	if err != nil {
		return fmt.Errorf("normal appearance: %w", err)
	}
	if !found {
		return nil
	}

	rect, err := annotationRect(ctxSrc.XRefTable, ann)
	if err != nil {
		return fmt.Errorf("rectangle: %w", err)
	}
	if rect == nil || !rect.Visible() {
		return nil
	}

	xo, err := ensureXObjectResourceDict(ctxDest, resDict)
	if err != nil {
		return fmt.Errorf("XObject resources: %w", err)
	}

	id := xo.NewIDForPrefix("Fm", 0)
	ir, err := xObjectRefForAppearance(o, ctxSrc, ctxDest, migrated)
	if err != nil {
		return fmt.Errorf("migrate XObject: %w", err)
	}
	xo[id] = ir

	bbox, err := appearanceBBox(ctxDest.XRefTable, ir)
	if err != nil {
		return fmt.Errorf("bounding box: %w", err)
	}
	if bbox == nil || !bbox.Visible() {
		return nil
	}

	appendAppearanceDo(w, id, rect, bbox)
	return nil
}

func appendAnnotationAppearances(
	w io.Writer,
	pageDict types.Dict,
	resDict types.Dict,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
) error {
	o, found := pageDict.Find("Annots")
	if !found {
		return nil
	}

	annots, err := ctxSrc.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("annotations array: %w", err)
	}

	for i, o := range annots {
		ann, err := ctxSrc.DereferenceDict(o)
		if err != nil {
			return fmt.Errorf("annotation %d: dictionary: %w", i+1, err)
		}
		if ann == nil {
			return fmt.Errorf("annotation %d: missing dictionary", i+1)
		}
		if err := appendAnnotationAppearance(w, ann, resDict, ctxSrc, ctxDest, migrated); err != nil {
			return fmt.Errorf("annotation %d: appearance: %w", i+1, err)
		}
	}
	return nil
}

func createPDFRes(ctx, otherCtx *model.Context, pageNrSrc, pageNrDest int, migrated map[int]int, wm *model.Watermark) error {
	pdfRes := model.PdfResources{}
	xRefTable := ctx.XRefTable
	otherXRefTable := otherCtx.XRefTable

	// Locate page dict & resource dict of PDF stamp.
	consolidateRes := true
	d, _, inhPAttrs, err := otherXRefTable.PageDict(pageNrSrc, consolidateRes)
	if err != nil {
		return fmt.Errorf("source page dictionary: %w", err)
	}
	if d == nil {
		return errors.New("missing source page dictionary")
	}
	if inhPAttrs == nil {
		return errors.New("missing source page attributes")
	}

	// Take into account existing rotation.
	wm.Rotation -= float64(inhPAttrs.Rotate % 360)

	// Retrieve content stream bytes of page dict.
	pdfRes.Content, err = otherXRefTable.PageContent(d, pageNrSrc)
	if err != nil && err != model.ErrNoContent {
		return fmt.Errorf("source page contents: %w", err)
	}

	// Migrate external resource dict into ctx.
	inhPAttrs.Resources = ensureResourceDict(inhPAttrs.Resources)
	if _, err = migrateObject(inhPAttrs.Resources, otherCtx, ctx, migrated); err != nil {
		return fmt.Errorf("migrate source page resources: %w", err)
	}

	var b bytes.Buffer
	b.Write(pdfRes.Content)
	if err := appendAnnotationAppearances(&b, d, inhPAttrs.Resources, otherCtx, ctx, migrated); err != nil {
		return fmt.Errorf("source page annotations: %w", err)
	}
	pdfRes.Content = b.Bytes()

	// Create an object for resource dict in xRefTable.
	ir, err := xRefTable.IndRefForNewObject(inhPAttrs.Resources)
	if err != nil {
		return fmt.Errorf("create resource dictionary object: %w", err)
	}
	pdfRes.ResDict = ir

	pdfRes.Bb = viewPort(inhPAttrs)
	if pdfRes.Bb == nil {
		return fmt.Errorf("PDF stamp page %d: missing media box", pageNrSrc)
	}
	wm.PdfRes[pageNrDest] = pdfRes

	return nil
}

func pdfResourcePageCount(destPageCount, srcPageCount, startPageNrSrc, startPageNrDest int) (int, error) {
	if startPageNrSrc < 1 || startPageNrSrc > srcPageCount {
		return 0, fmt.Errorf("invalid PDF stamp source page number: %d", startPageNrSrc)
	}
	if startPageNrDest < 1 || startPageNrDest > destPageCount {
		return 0, fmt.Errorf("invalid PDF stamp destination page number: %d", startPageNrDest)
	}
	srcPages := srcPageCount - startPageNrSrc + 1
	destPages := destPageCount - startPageNrDest + 1
	return min(srcPages, destPages), nil
}

func createPDFResForWM(ctx *model.Context, wm *model.Watermark) error {
	// Note: The stamp pdf is assumed to be valid!
	if wm.PDF == nil && wm.FileName == "" {
		return fmt.Errorf("missing PDF source: %w", ErrMissingWatermarkConfiguration)
	}
	if wm.PdfRes == nil {
		wm.PdfRes = map[int]model.PdfResources{}
	}

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
		return fmt.Errorf("read source: %w", err)
	}
	if otherCtx == nil {
		return ErrMissingPDFContext
	}
	if otherCtx.XRefTable == nil {
		return ErrMissingXRefTable
	}
	if otherCtx.XRefTable.Version() == model.V20 {
		return fmt.Errorf("source version: %w", ErrUnsupportedVersion)
	}

	if err := otherCtx.EnsurePageCount(); err != nil {
		return fmt.Errorf("source page count: %w", err)
	}

	migrated := map[int]int{}

	if !wm.MultiStamp() {
		if err := createPDFRes(ctx, otherCtx, wm.PdfPageNrSrc, wm.PdfPageNrSrc, migrated, wm); err != nil {
			return fmt.Errorf("source page %d: %w", wm.PdfPageNrSrc, err)
		}
		return nil
	}

	pageCount, err := pdfResourcePageCount(
		ctx.PageCount,
		otherCtx.PageCount,
		wm.PdfMultiStartPageNrSrc,
		wm.PdfMultiStartPageNrDest,
	)
	if err != nil {
		return fmt.Errorf("page range: %w", err)
	}

	for i := range pageCount {
		srcPageNr := wm.PdfMultiStartPageNrSrc + i
		destPageNr := wm.PdfMultiStartPageNrDest + i
		if err := createPDFRes(ctx, otherCtx, srcPageNr, destPageNr, migrated, wm); err != nil {
			return fmt.Errorf("source page %d to destination page %d: %w", srcPageNr, destPageNr, err)
		}
	}

	return nil
}

func createImageResForWM(ctx *model.Context, wm *model.Watermark) (err error) {
	if wm.Image == nil {
		return ErrMissingImageReader
	}
	wm.Img, wm.Width, wm.Height, err = model.CreateImageResource(ctx.XRefTable, wm.Image)
	return err
}

func createFontResForWM(ctx *model.Context, wm *model.Watermark, fonts map[string]types.IndirectRef) (err error) {
	if wm.FontName == "" {
		return fmt.Errorf("font name: %w", ErrMissingWatermarkConfiguration)
	}
	if fonts == nil {
		return errors.New("missing font cache")
	}
	if indRef, ok := fonts[wm.FontName]; ok {
		wm.Font = &indRef
		return nil
	}

	if font.IsCoreFont(wm.FontName) {
		indRef, err := pdffont.CoreFontDict(ctx.XRefTable, wm.FontName)
		if err != nil {
			return fmt.Errorf("font %s: create core font dictionary: %w", wm.FontName, err)
		}
		fonts[wm.FontName] = *indRef
		wm.Font = indRef
		return nil
	}

	if ctx.Optimize == nil {
		return fmt.Errorf("font %s: %w", wm.FontName, ErrMissingOptimizationContext)
	}
	for objNr, fo := range ctx.Optimize.FontObjects {
		if fo == nil {
			continue
		}
		if fo.FontName == wm.FontName && fo.Prefix != "" {
			indRef := types.NewIndirectRef(objNr, 0)
			fonts[wm.FontName] = *indRef
			wm.Font = indRef
			return nil
		}
	}

	indRef, err := pdffont.EnsureFontDict(ctx.XRefTable, wm.FontName, "", wm.ScriptName, false, nil)
	if err != nil {
		return fmt.Errorf("font %s: ensure font dictionary: %w", wm.FontName, err)
	}

	fonts[wm.FontName] = *indRef
	wm.Font = indRef
	return nil
}

func createResourcesForWM(ctx *model.Context, wm *model.Watermark, fonts map[string]types.IndirectRef) error {
	if wm.IsPDF() {
		if err := createPDFResForWM(ctx, wm); err != nil {
			return fmt.Errorf("PDF: %w", err)
		}
		return nil
	}
	if wm.IsImage() {
		if err := createImageResForWM(ctx, wm); err != nil {
			return fmt.Errorf("image: %w", err)
		}
		return nil
	}
	if err := createFontResForWM(ctx, wm, fonts); err != nil {
		return fmt.Errorf("font: %w", err)
	}
	return nil
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
		return nil, fmt.Errorf("catalog: %w", err)
	}

	if o, ok := rootDict.Find("OCProperties"); ok {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return nil, fmt.Errorf("optional content properties: dereference dictionary: %w", err)
		}
		if d == nil {
			return nil, errors.New("optional content properties: missing dictionary")
		}

		o, found := d.Find("OCGs")
		if found {
			a, err := ctx.DereferenceArray(o)
			if err != nil {
				return nil, fmt.Errorf("%w: dereference OCGs: %w", errCorruptOCGs, err)
			}
			if len(a) > 0 {
				ir, ok := a[0].(types.IndirectRef)
				if !ok {
					return nil, fmt.Errorf("%w: first OCG has type %T", errCorruptOCGs, a[0])
				}
				return &ir, nil
			}
		}
	}

	ir, err := ensureOCG(ctx, onTop)
	if err != nil {
		return nil, fmt.Errorf("create optional content group: %w", err)
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
		pdfRes, err := pdfResourceForPage(wm, pageNr)
		if err != nil {
			return nil, err
		}
		if pdfRes.ResDict == nil {
			return nil, fmt.Errorf("destination page %d: missing PDF stamp resource dictionary: %w", pageNr, ErrMissingWatermarkConfiguration)
		}
		return pdfRes.ResDict, nil
	}

	if wm.IsImage() {
		if wm.Img == nil {
			return nil, fmt.Errorf("missing image resource: %w", ErrMissingWatermarkConfiguration)
		}
		d := types.Dict(
			map[string]types.Object{
				"ProcSet": types.NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
				"XObject": types.Dict(map[string]types.Object{"Im0": *wm.Img}),
			},
		)
		return ctx.IndRefForNewObject(d)
	}

	if wm.Font == nil {
		return nil, fmt.Errorf("missing font resource: %w", ErrMissingWatermarkConfiguration)
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
	pdfRes, err := pdfResourceForPage(&wm, pageNr)
	if err != nil {
		return err
	}
	if wm.Bb == nil {
		return fmt.Errorf("missing bounding box: %w", ErrMissingWatermarkConfiguration)
	}

	sc := wm.Scale
	if !wm.ScaleAbs {
		if wm.Width == 0 {
			return fmt.Errorf("invalid zero width: %w", ErrMissingWatermarkConfiguration)
		}
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

	if _, err := fmt.Fprintf(w, "%.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1]); err != nil {
		return fmt.Errorf("write transform: %w", err)
	}

	if _, err := w.Write(pdfRes.Content); err != nil {
		return fmt.Errorf("write source content: %w", err)
	}
	return nil
}

func imageFormContent(w io.Writer, wm model.Watermark) error {
	if wm.Bb == nil {
		return fmt.Errorf("missing bounding box: %w", ErrMissingWatermarkConfiguration)
	}
	if _, err := fmt.Fprintf(w, "q %f 0 0 %f 0 0 cm /Im0 Do Q", wm.Bb.Width(), wm.Bb.Height()); err != nil { // TODO dont need Q
		return fmt.Errorf("write image content: %w", err)
	}
	return nil
}

func formContent(w io.Writer, pageNr int, wm model.Watermark) error {
	switch true {
	case wm.IsPDF():
		return pdfFormContent(w, pageNr, wm)
	case wm.IsImage():
		return imageFormContent(w, wm)
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

	td.Embed = wm.ScriptName == ""

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

func pdfResourceForPage(wm *model.Watermark, pageNr int) (model.PdfResources, error) {
	i := wm.PdfResIndex(pageNr)
	pdfRes, ok := wm.PdfRes[i]
	if !ok {
		return model.PdfResources{}, fmt.Errorf("destination page %d: missing PDF stamp resource: %w", pageNr, ErrMissingWatermarkConfiguration)
	}
	if pdfRes.Bb == nil {
		return model.PdfResources{}, fmt.Errorf("destination page %d: missing PDF stamp bounding box: %w", pageNr, ErrMissingWatermarkConfiguration)
	}
	return pdfRes, nil
}

func calcFormBoundingBox(xRefTable *model.XRefTable, w io.Writer, timestampFormat string, pageNr, pageCount int, wm *model.Watermark) (bool, error) {
	if wm.Vp == nil {
		return false, fmt.Errorf("missing viewport: %w", ErrMissingWatermarkConfiguration)
	}
	if wm.IsImage() && (wm.Width <= 0 || wm.Height <= 0) {
		return false, fmt.Errorf("invalid image dimensions %dx%d: %w", wm.Width, wm.Height, ErrMissingWatermarkConfiguration)
	}

	var unique bool
	if wm.IsPDF() {
		if _, err := pdfResourceForPage(wm, pageNr); err != nil {
			return false, err
		}
		wm.CalcBoundingBox(pageNr)
	} else if wm.IsImage() {
		wm.CalcBoundingBox(pageNr)
	} else {
		var td model.TextDescriptor
		td, unique = setupTextDescriptor(*wm, timestampFormat, pageNr, pageCount)
		var err error
		// Pre-wrap text when MaxWidth is set.
		if wm.MaxWidth > 0 {
			td.Text = strings.Join(model.WordWrap(td.Text, td.FontName, td.FontSize, wm.MaxWidth), "\n")
		}
		// Render td into b and return the bounding box.
		wm.Bb = model.WriteMultiLine(xRefTable, w, types.RectForDim(wm.Vp.Width(), wm.Vp.Height()), nil, td)
		//wm.Bb, err = model.WriteMultiLine(xRefTable, w, types.RectForDim(wm.Vp.Width(), wm.Vp.Height()), nil, td)
		if err != nil {
			return false, fmt.Errorf("render text watermark: %w", err)
		}
	}
	if wm.Bb == nil {
		return false, fmt.Errorf("missing bounding box: %w", ErrMissingWatermarkConfiguration)
	}
	return unique, nil
}

func writeFormContent(w io.Writer, pageNr int, wm model.Watermark) error {
	if !wm.IsImage() && !wm.IsPDF() {
		return nil
	}
	return formContent(w, pageNr, wm)
}

func cachedFormForPage(wm *model.Watermark, pageNr int, unique bool) (*types.IndirectRef, bool) {
	if unique {
		return nil, false
	}
	maxStampPageNr := wm.PdfMultiStartPageNrDest + len(wm.PdfRes) - 1
	if !cachedForm(*wm) && pageNr <= maxStampPageNr {
		return nil, false
	}
	ir, ok := wm.FCache[*wm.Bb]
	return ir, ok
}

func createFormStream(ctx *model.Context, b *bytes.Buffer, bb *types.Rectangle, wm *model.Watermark, res *types.IndirectRef, withBB bool) (*types.IndirectRef, error) {
	bbox := bb.CroppedCopy(0)
	bbox.Translate(-bb.LL.X, -bb.LL.Y)
	if withBB {
		drawBoundingBox(b, *wm, bbox)
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
	if res != nil {
		sd.Insert("Resources", *res)
	}
	sd.InsertName("Filter", filter.Flate)
	if err := sd.Encode(); err != nil {
		return nil, fmt.Errorf("encode form stream: %w", err)
	}

	ir, err := ctx.IndRefForNewObject(sd)
	if err != nil {
		return nil, fmt.Errorf("create form stream object: %w", err)
	}
	return ir, nil
}

func cacheFormForPage(wm *model.Watermark, pageNr int, ir *types.IndirectRef) {
	if cachedForm(*wm) || pageNr >= len(wm.PdfRes) {
		wm.FCache[*wm.Bb] = ir
	}
}

func ensureWatermarkCaches(wm *model.Watermark) {
	if wm.FCache == nil || wm.Objs == nil {
		wm.Recycle()
	}
}

func createForm(ctx *model.Context, pageNr, pageCount int, wm *model.Watermark, withBB bool) error {
	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}
	if wm.Ocg == nil {
		return fmt.Errorf("missing optional content group: %w", ErrMissingWatermarkConfiguration)
	}
	ensureWatermarkCaches(wm)

	var b bytes.Buffer
	unique, err := calcFormBoundingBox(ctx.XRefTable, &b, ctx.Configuration.TimestampFormat, pageNr, pageCount, wm)
	if err != nil {
		return fmt.Errorf("calculate bounding box: %w", err)
	}

	if ir, ok := cachedFormForPage(wm, pageNr, unique); ok {
		wm.Form = ir
		return nil
	}

	if err := writeFormContent(&b, pageNr, *wm); err != nil {
		return fmt.Errorf("write content: %w", err)
	}

	res, err := createFormResDict(ctx, pageNr, wm)
	if err != nil {
		return fmt.Errorf("create resource dictionary: %w", err)
	}

	ir, err := createFormStream(ctx, &b, wm.Bb, wm, res, withBB)
	if err != nil {
		return err
	}

	wm.Form = ir
	cacheFormForPage(wm, pageNr, ir)
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

func validatePageWatermarkResourceRefs(wm model.Watermark) error {
	if wm.ExtGState == nil {
		return fmt.Errorf("ExtGState: missing reference: %w", ErrMissingWatermarkConfiguration)
	}
	if wm.Form == nil {
		return fmt.Errorf("XObject: missing form reference: %w", ErrMissingWatermarkConfiguration)
	}
	return nil
}

func insertPageResourcesForWM(pageDict types.Dict, wm model.Watermark, gsID, xoID string) error {
	if pageDict == nil {
		return errors.New("missing page dictionary")
	}
	if err := validatePageWatermarkResourceRefs(wm); err != nil {
		return err
	}
	resourceDict := types.Dict(
		map[string]types.Object{
			"ExtGState": types.Dict(map[string]types.Object{gsID: *wm.ExtGState}),
			"XObject":   types.Dict(map[string]types.Object{xoID: *wm.Form}),
		},
	)

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func updatePageWatermarkResource(ctx *model.Context, resDict types.Dict, category, prefix, defaultID string, ref *types.IndirectRef) (string, error) {
	o, found := resDict.Find(category)
	if !found {
		resDict.Insert(category, types.Dict(map[string]types.Object{defaultID: *ref}))
		return defaultID, nil
	}

	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return "", fmt.Errorf("%s: dereference dictionary: %w", category, err)
	}
	if d == nil {
		return "", fmt.Errorf("%s: missing dictionary", category)
	}
	id := d.NewIDForPrefix(prefix, 0)
	d.Insert(id, *ref)
	return id, nil
}

func updatePageResourcesForWM(ctx *model.Context, resDict types.Dict, wm model.Watermark, gsID, xoID *string) error {
	if resDict == nil {
		return errors.New("missing page resource dictionary")
	}
	if gsID == nil || xoID == nil {
		return errors.New("missing page resource identifier")
	}
	if err := validatePageWatermarkResourceRefs(wm); err != nil {
		return err
	}

	id, err := updatePageWatermarkResource(ctx, resDict, "ExtGState", "GS", *gsID, wm.ExtGState)
	if err != nil {
		return err
	}
	*gsID = id

	id, err = updatePageWatermarkResource(ctx, resDict, "XObject", "Fm", *xoID, wm.Form)
	if err != nil {
		return err
	}
	*xoID = id

	return nil
}

func wmContent(wm *model.Watermark, gsID, xoID string) ([]byte, error) {
	if wm == nil {
		return nil, ErrMissingWatermarkConfiguration
	}
	if wm.Bb == nil {
		return nil, fmt.Errorf("missing bounding box: %w", ErrMissingWatermarkConfiguration)
	}
	if wm.Vp == nil {
		return nil, fmt.Errorf("missing viewport: %w", ErrMissingWatermarkConfiguration)
	}
	if gsID == "" || xoID == "" {
		return nil, errors.New("missing page resource identifier")
	}
	m := wm.CalcTransformMatrix()
	p1 := m.Transform(types.Point{X: wm.Bb.LL.X, Y: wm.Bb.LL.Y})
	p2 := m.Transform(types.Point{X: wm.Bb.UR.X, Y: wm.Bb.LL.Y})
	p3 := m.Transform(types.Point{X: wm.Bb.UR.X, Y: wm.Bb.UR.Y})
	p4 := m.Transform(types.Point{X: wm.Bb.LL.X, Y: wm.Bb.UR.Y})
	wm.BbTrans = types.QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}
	insertOCG := " /Artifact <</Subtype /Watermark /Type /Pagination >>BDC q %.5f %.5f %.5f %.5f %.5f %.5f cm /%s gs /%s Do Q EMC "
	var b bytes.Buffer
	fmt.Fprintf(&b, insertOCG, m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], gsID, xoID)
	return b.Bytes(), nil
}

func insertPageContentsForWM(ctx *model.Context, pageDict types.Dict, wm *model.Watermark, gsID, xoID string) error {
	if pageDict == nil {
		return errors.New("missing page dictionary")
	}
	if _, found := pageDict.Find("Contents"); found {
		return errors.New("page dictionary already has Contents")
	}
	bb, err := wmContent(wm, gsID, xoID)
	if err != nil {
		return fmt.Errorf("generate content: %w", err)
	}
	sd, err := ctx.NewStreamDictForBuf(bb)
	if err != nil {
		return fmt.Errorf("create stream dictionary: %w", err)
	}
	if sd == nil {
		return errors.New("create stream dictionary: missing stream dictionary")
	}
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode stream: %w", err)
	}

	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return fmt.Errorf("create stream object: %w", err)
	}

	pageDict.Insert("Contents", *ir)

	return nil
}

func patchFirstContentStreamForWatermark(sd *types.StreamDict, gsID, xoID string, wm *model.Watermark, isLast bool) error {
	if sd == nil {
		return errors.New("missing content stream")
	}
	err := sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		return fmt.Errorf("decode stream: %w", err)
	}
	if err != nil {
		return fmt.Errorf("decode stream: %w", err)
	}

	wmbb, err := wmContent(wm, gsID, xoID)
	if err != nil {
		return fmt.Errorf("generate content: %w", err)
	}

	// stamp
	if wm.OnTop {
		bb := []byte(" q ")
		if wm.PageRot != 0 {
			bb = append(bb, model.ContentBytesForPageRotation(wm.PageRot, wm.Vp.Width(), wm.Vp.Height())...)
		}
		sd.Content = append(bb, sd.Content...)
		if !isLast {
			if err := sd.Encode(); err != nil {
				return fmt.Errorf("encode leading stream: %w", err)
			}
			return nil
		}
		sd.Content = append(sd.Content, []byte(" Q ")...)
		sd.Content = append(sd.Content, wmbb...)
		if err := sd.Encode(); err != nil {
			return fmt.Errorf("encode stamped stream: %w", err)
		}
		return nil
	}

	// watermark
	if wm.PageRot == 0 {
		sd.Content = append(wmbb, sd.Content...)
		if err := sd.Encode(); err != nil {
			return fmt.Errorf("encode watermarked stream: %w", err)
		}
		return nil
	}

	bb := append([]byte(" q "), model.ContentBytesForPageRotation(wm.PageRot, wm.Vp.Width(), wm.Vp.Height())...)
	sd.Content = append(bb, sd.Content...)
	if isLast {
		sd.Content = append(sd.Content, []byte(" Q")...)
	}
	if err := sd.Encode(); err != nil {
		return fmt.Errorf("encode rotated stream: %w", err)
	}
	return nil
}

func newContentStreamForWatermark(ctx *model.Context, gsID, xoID string, wm *model.Watermark) (*types.IndirectRef, error) {
	if wm == nil {
		return nil, ErrMissingWatermarkConfiguration
	}
	var bb []byte
	if wm.OnTop {
		wmBB, err := wmContent(wm, gsID, xoID)
		if err != nil {
			return nil, fmt.Errorf("generate content: %w", err)
		}
		bb = append([]byte(" Q "), wmBB...)
	} else if wm.PageRot != 0 {
		bb = []byte(" Q ")
	}

	sd, err := ctx.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, fmt.Errorf("create stream dictionary: %w", err)
	}
	if sd == nil {
		return nil, errors.New("create stream dictionary: missing stream dictionary")
	}

	if err := sd.Encode(); err != nil {
		return nil, fmt.Errorf("encode stream: %w", err)
	}

	indRef, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return nil, fmt.Errorf("create stream object: %w", err)
	}

	return indRef, nil
}

func contentObjectForIndRef(ctx *model.Context, ir types.IndirectRef) (*model.XRefTableEntry, types.Object, int, error) {
	objNr := ir.ObjectNumber.Value()
	entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
	if !found || entry == nil {
		return nil, nil, objNr, fmt.Errorf("content stream obj#%d: missing xref entry", objNr)
	}
	if entry.Object == nil {
		return nil, nil, objNr, fmt.Errorf("content stream obj#%d: missing object", objNr)
	}
	return entry, entry.Object, objNr, nil
}

func patchPageWatermarkStream(d types.Dict, entry *model.XRefTableEntry, objNr int, sd types.StreamDict, gsID, xoID string, wm *model.Watermark, isLast bool) error {
	if objNr > 0 && wm.Objs[objNr] {
		return nil
	}
	if err := patchFirstContentStreamForWatermark(&sd, gsID, xoID, wm, isLast); err != nil {
		return err
	}
	if entry == nil {
		d["Contents"] = sd
		return nil
	}
	entry.Object = sd
	wm.Objs[objNr] = true
	return nil
}

func patchPageWatermarkContentArray(ctx *model.Context, d types.Dict, a types.Array, gsID, xoID string, wm *model.Watermark) error {
	if len(a) == 0 {
		return nil
	}
	ir, ok := a[0].(types.IndirectRef)
	if !ok {
		return fmt.Errorf("content array entry 1: expected indirect reference, got %T", a[0])
	}
	entry, obj, objNr, err := contentObjectForIndRef(ctx, ir)
	if err != nil {
		return fmt.Errorf("content array entry 1: %w", err)
	}
	sd, ok := obj.(types.StreamDict)
	if !ok {
		return fmt.Errorf("content array entry 1 obj#%d: expected stream dictionary, got %T", objNr, obj)
	}
	if wm.Objs[objNr] {
		return nil
	}
	if err := patchPageWatermarkStream(d, entry, objNr, sd, gsID, xoID, wm, len(a) == 1); err != nil {
		return fmt.Errorf("content array entry 1 obj#%d: patch stream: %w", objNr, err)
	}
	if len(a) == 1 {
		return nil
	}

	newIR, err := newContentStreamForWatermark(ctx, gsID, xoID, wm)
	if err != nil {
		return fmt.Errorf("append content stream: %w", err)
	}
	d["Contents"] = append(a, *newIR)
	wm.Objs[newIR.ObjectNumber.Value()] = true
	return nil
}

func updatePageContentsForWM(ctx *model.Context, d types.Dict, wm *model.Watermark, gsID, xoID string) error {
	if d == nil {
		return errors.New("missing page dictionary")
	}
	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}
	ensureWatermarkCaches(wm)
	obj, found := d.Find("Contents")
	if !found || obj == nil {
		return errors.New("missing Contents")
	}

	var entry *model.XRefTableEntry
	var objNr int
	if ir, ok := obj.(types.IndirectRef); ok {
		var err error
		entry, obj, objNr, err = contentObjectForIndRef(ctx, ir)
		if err != nil {
			return fmt.Errorf("Contents: %w", err)
		}
	}

	switch o := obj.(type) {
	case types.StreamDict:
		if err := patchPageWatermarkStream(d, entry, objNr, o, gsID, xoID, wm, true); err != nil {
			return fmt.Errorf("content stream: patch: %w", err)
		}
		return nil
	case types.Array:
		return patchPageWatermarkContentArray(ctx, d, o, gsID, xoID, wm)
	default:
		return fmt.Errorf("Contents: expected stream dictionary or array, got %T", obj)
	}
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
	if pageIndRef == nil {
		return errors.New("missing page dictionary reference")
	}
	if d == nil {
		return errors.New("missing page dictionary")
	}
	ann := model.NewLinkAnnotation(
		*wm.BbTrans.EnclosingRectangle(5.0), // rect
		0,                                   // apObjNr
		"",                                  // contents
		"pdfcpu",                            // id
		"",                                  // modDate
		model.AnnNoZoom+model.AnnNoRotate,   // f
		&color.Red,                          // borderCol
		nil,                                 // dest
		wm.URL,                              // uri
		types.QuadPoints{wm.BbTrans},        // quad
		false,                               // border
		0,                                   // borderWidth
		model.BSSolid,                       // borderStyle
	)

	if _, _, err := AddAnnotation(ctx, pageIndRef, d, pageNr, ann, false); err != nil {
		return fmt.Errorf("annotation: %w", err)
	}
	return nil
}

func addPageWatermarkResources(ctx *model.Context, d types.Dict, attrs *model.InheritedPageAttrs, wm model.Watermark) (string, string, error) {
	gsID := "GS0"
	xoID := "Fm0"
	if attrs.Resources != nil {
		if err := updatePageResourcesForWM(ctx, attrs.Resources, wm, &gsID, &xoID); err != nil {
			return "", "", fmt.Errorf("update resources: %w", err)
		}
		d.Update("Resources", attrs.Resources)
		return gsID, xoID, nil
	}
	if err := insertPageResourcesForWM(d, wm, gsID, xoID); err != nil {
		return "", "", fmt.Errorf("insert resources: %w", err)
	}
	return gsID, xoID, nil
}

func addPageWatermarkContents(ctx *model.Context, d types.Dict, wm *model.Watermark, gsID, xoID string) error {
	if _, found := d.Find("Contents"); found {
		if err := updatePageContentsForWM(ctx, d, wm, gsID, xoID); err != nil {
			return fmt.Errorf("update contents: %w", err)
		}
		return nil
	}
	if err := insertPageContentsForWM(ctx, d, wm, gsID, xoID); err != nil {
		return fmt.Errorf("insert contents: %w", err)
	}
	return nil
}

func updatePageWatermark(ctx *model.Context, pageNr int, update bool) error {
	if !update {
		return nil
	}
	if log.DebugEnabled() {
		log.Debug.Println("Updating")
	}
	if _, err := removePageWatermark(ctx, pageNr); err != nil {
		return fmt.Errorf("update existing watermark: %w", err)
	}
	return nil
}

func pageWatermarkContext(ctx *model.Context, pageNr int) (types.Dict, *types.IndirectRef, *model.InheritedPageAttrs, error) {
	d, pageIndRef, attrs, err := ctx.PageDict(pageNr, false)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("page dictionary: %w", err)
	}
	if d == nil {
		return nil, nil, nil, errors.New("missing page dictionary")
	}
	if pageIndRef == nil {
		return nil, nil, nil, errors.New("missing page dictionary reference")
	}
	if attrs == nil {
		return nil, nil, nil, errors.New("missing inherited page attributes")
	}
	return d, pageIndRef, attrs, nil
}

func normalizePageWatermark(d types.Dict, attrs *model.InheritedPageAttrs, wm *model.Watermark) error {
	wm.PageRot = attrs.Rotate
	wm.Vp = viewPort(attrs)
	if wm.Vp == nil {
		return errors.New("missing page viewport")
	}
	if wm.PageRot == 0 {
		return nil
	}
	if types.IntMemberOf(wm.PageRot, []int{+90, -90, +270, -270}) {
		w := wm.Vp.Width()
		wm.Vp.UR.X = wm.Vp.LL.X + wm.Vp.Height()
		wm.Vp.UR.Y = wm.Vp.LL.Y + w
	}
	d.Update("MediaBox", wm.Vp.Array())
	d.Update("CropBox", wm.Vp.Array())
	d.Delete("Rotate")
	return nil
}

func addPageWatermark(ctx *model.Context, pageNr int, wm model.Watermark) error {
	if pageNr < 1 || pageNr > ctx.PageCount {
		return ErrInvalidPageNumber
	}

	if log.DebugEnabled() {
		log.Debug.Printf("addPageWatermark page:%d\n", pageNr)
	}

	if err := updatePageWatermark(ctx, pageNr, wm.Update); err != nil {
		return err
	}

	d, pageIndRef, inhPAttrs, err := pageWatermarkContext(ctx, pageNr)
	if err != nil {
		return err
	}

	if err := normalizePageWatermark(d, inhPAttrs, &wm); err != nil {
		return err
	}

	if err = createForm(ctx, pageNr, ctx.PageCount, &wm, stampWithBBox); err != nil {
		return fmt.Errorf("create form: %w", err)
	}

	if log.DebugEnabled() {
		log.Debug.Printf("\n%s\n", wm)
	}

	gsID, xoID, err := addPageWatermarkResources(ctx, d, inhPAttrs, wm)
	if err != nil {
		return err
	}

	if err := addPageWatermarkContents(ctx, d, &wm, gsID, xoID); err != nil {
		return err
	}

	if err := handleLink(ctx, pageIndRef, d, pageNr, wm); err != nil {
		return fmt.Errorf("add link: %w", err)
	}
	return nil
}

// AddWatermarks adds watermarks to all pages selected.
func AddWatermarks(ctx *model.Context, selectedPages types.IntSet, wm *model.Watermark) error {
	if err := validateWatermarkContext(ctx); err != nil {
		return err
	}
	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}
	ensureWatermarkCaches(wm)

	if log.DebugEnabled() {
		log.Debug.Printf("AddWatermarks wm:\n%s\n", wm)
	}

	var err error

	if wm.Ocg, err = prepareOCPropertiesInRoot(ctx, wm.OnTop); err != nil {
		return fmt.Errorf("prepare optional content: %w", err)
	}

	if wm.ExtGState, err = createExtGStateForStamp(ctx, wm.Opacity); err != nil {
		return fmt.Errorf("create graphics state: %w", err)
	}

	fonts := map[string]types.IndirectRef{}

	if err = createResourcesForWM(ctx, wm, fonts); err != nil {
		return fmt.Errorf("prepare resources: %w", err)
	}

	for i := wm.PdfMultiStartPageNrDest; i <= ctx.PageCount; i++ {
		if len(selectedPages) == 0 || selectedPages[i] {
			if err = addPageWatermark(ctx, i, *wm); err != nil {
				return fmt.Errorf("page %d: %w", i, err)
			}
		}
	}

	if err := pdffont.UpdateUserfonts(ctx.XRefTable, fonts); err != nil {
		return fmt.Errorf("update user fonts: %w", err)
	}

	ctx.EnsureVersionForWriting()

	return nil
}

func sortedWatermarkPages[T any](m map[int]T) []int {
	pageNrs := make([]int, 0, len(m))
	for pageNr := range m {
		pageNrs = append(pageNrs, pageNr)
	}
	sort.Ints(pageNrs)
	return pageNrs
}

func validateSharedWatermarkSettings(wm, first *model.Watermark) error {
	if wm.OnTop != first.OnTop {
		return fmt.Errorf("inconsistent OnTop: got %t, want %t", wm.OnTop, first.OnTop)
	}
	if wm.Opacity != first.Opacity {
		return fmt.Errorf("inconsistent Opacity: got %g, want %g", wm.Opacity, first.Opacity)
	}
	return nil
}

func watermarkMapSettings(ctx *model.Context, m map[int]*model.Watermark, pageNrs []int) (bool, float64, error) {
	if len(m) == 0 {
		return false, 0, ErrMissingWatermarks
	}
	var first *model.Watermark
	for _, pageNr := range pageNrs {
		wm := m[pageNr]
		if pageNr < 1 || pageNr > ctx.PageCount {
			return false, 0, fmt.Errorf("page %d: %w", pageNr, ErrInvalidPageNumber)
		}
		if wm == nil {
			return false, 0, fmt.Errorf("page %d: %w", pageNr, ErrMissingWatermarkConfiguration)
		}
		ensureWatermarkCaches(wm)
		if first == nil {
			first = wm
			continue
		}
		if err := validateSharedWatermarkSettings(wm, first); err != nil {
			return false, 0, fmt.Errorf("page %d: %w", pageNr, err)
		}
	}
	return first.OnTop, first.Opacity, nil
}

func prepareSharedWatermarkResources(ctx *model.Context, onTop bool, opacity float64) (*types.IndirectRef, *types.IndirectRef, error) {
	ocgIndRef, err := prepareOCPropertiesInRoot(ctx, onTop)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare optional content: %w", err)
	}
	extGStateIndRef, err := createExtGStateForStamp(ctx, opacity)
	if err != nil {
		return nil, nil, fmt.Errorf("create graphics state: %w", err)
	}
	return ocgIndRef, extGStateIndRef, nil
}

// AddWatermarksMap adds watermarks in m to corresponding pages.
func AddWatermarksMap(ctx *model.Context, m map[int]*model.Watermark) error {
	if err := validateWatermarkContext(ctx); err != nil {
		return err
	}
	pageNrs := sortedWatermarkPages(m)
	onTop, opacity, err := watermarkMapSettings(ctx, m, pageNrs)
	if err != nil {
		return err
	}
	ocgIndRef, extGStateIndRef, err := prepareSharedWatermarkResources(ctx, onTop, opacity)
	if err != nil {
		return err
	}
	fonts := map[string]types.IndirectRef{}
	for _, pageNr := range pageNrs {
		wm := m[pageNr]
		if err := createResourcesForWM(ctx, wm, fonts); err != nil {
			return fmt.Errorf("page %d: prepare resources: %w", pageNr, err)
		}
	}
	for _, pageNr := range pageNrs {
		wm := m[pageNr]
		wm.Ocg = ocgIndRef
		wm.ExtGState = extGStateIndRef
		wm.OnTop = onTop
		wm.Opacity = opacity
		if err := addPageWatermark(ctx, pageNr, *wm); err != nil {
			return fmt.Errorf("page %d: %w", pageNr, err)
		}
	}
	if err := pdffont.UpdateUserfonts(ctx.XRefTable, fonts); err != nil {
		return fmt.Errorf("update user fonts: %w", err)
	}

	ctx.EnsureVersionForWriting()

	return nil
}

func watermarkSliceMapSettings(ctx *model.Context, m map[int][]*model.Watermark, pageNrs []int) (bool, float64, error) {
	if len(m) == 0 {
		return false, 0, ErrMissingWatermarks
	}
	var first *model.Watermark
	for _, pageNr := range pageNrs {
		wms := m[pageNr]
		if pageNr < 1 || pageNr > ctx.PageCount {
			return false, 0, fmt.Errorf("page %d: %w", pageNr, ErrInvalidPageNumber)
		}
		if len(wms) == 0 {
			return false, 0, fmt.Errorf("page %d: %w", pageNr, ErrMissingWatermarks)
		}
		for i, wm := range wms {
			if wm == nil {
				return false, 0, fmt.Errorf("page %d, watermark %d: %w", pageNr, i, ErrMissingWatermarkConfiguration)
			}
			ensureWatermarkCaches(wm)
			if first == nil {
				first = wm
				continue
			}
			if err := validateSharedWatermarkSettings(wm, first); err != nil {
				return false, 0, fmt.Errorf("page %d, watermark %d: %w", pageNr, i, err)
			}
		}
	}
	return first.OnTop, first.Opacity, nil
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages.
func AddWatermarksSliceMap(ctx *model.Context, m map[int][]*model.Watermark) error {
	if err := validateWatermarkContext(ctx); err != nil {
		return err
	}
	pageNrs := sortedWatermarkPages(m)
	onTop, opacity, err := watermarkSliceMapSettings(ctx, m, pageNrs)
	if err != nil {
		return err
	}
	ocgIndRef, extGStateIndRef, err := prepareSharedWatermarkResources(ctx, onTop, opacity)
	if err != nil {
		return err
	}
	fonts := map[string]types.IndirectRef{}
	for _, pageNr := range pageNrs {
		wms := m[pageNr]
		for i, wm := range wms {
			if err := createResourcesForWM(ctx, wm, fonts); err != nil {
				return fmt.Errorf("page %d, watermark %d: prepare resources: %w", pageNr, i, err)
			}
		}
	}
	for _, pageNr := range pageNrs {
		wms := m[pageNr]
		for i, wm := range wms {
			wm.Ocg = ocgIndRef
			wm.ExtGState = extGStateIndRef
			wm.OnTop = onTop
			wm.Opacity = opacity
			if err := addPageWatermark(ctx, pageNr, *wm); err != nil {
				return fmt.Errorf("page %d, watermark %d: %w", pageNr, i, err)
			}
		}
	}
	if err := pdffont.UpdateUserfonts(ctx.XRefTable, fonts); err != nil {
		return fmt.Errorf("update user fonts: %w", err)
	}

	ctx.EnsureVersionForWriting()

	return nil
}

func removeResDictEntry(ctx *model.Context, d types.Dict, entry string, ids []string, i int) error {
	if d == nil {
		return fmt.Errorf("page %d: missing resource dictionary", i)
	}
	o, ok := d.Find(entry)
	if !ok {
		return fmt.Errorf("page %d: resource category %s: missing dictionary", i, entry)
	}

	d1, err := ctx.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("page %d: resource category %s: dereference dictionary: %w", i, entry, err)
	}
	if d1 == nil {
		return fmt.Errorf("page %d: resource category %s: missing dictionary", i, entry)
	}

	for _, id := range ids {
		if _, ok := d1.Find(id); ok {
			d1.Delete(id)
		}
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
	if sd == nil {
		return false, nil, nil, errors.New("missing content stream")
	}
	err = sd.Decode()
	if err != nil {
		return false, nil, nil, fmt.Errorf("decode content stream: %w", err)
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
		if err := sd.Encode(); err != nil {
			return false, nil, nil, fmt.Errorf("encode content stream: %w", err)
		}
	}

	return patched, extGStates, forms, nil
}

func removeArtifactsFromPage(ctx *model.Context, sd *types.StreamDict, resDict types.Dict, i int) (bool, error) {
	if resDict == nil {
		return false, errors.New("missing page resource dictionary")
	}
	// Remove watermark artifacts and locate id's
	// of used extGStates and forms.
	ok, extGStates, forms, err := removeArtifacts(sd, i)
	if err != nil {
		return false, fmt.Errorf("remove artifacts: %w", err)
	}
	if !ok {
		return false, nil
	}

	// Remove obsolete extGStates from page resource dict.
	err = removeExtGStates(ctx, resDict, extGStates, i)
	if err != nil {
		return false, fmt.Errorf("remove ExtGState resources: %w", err)
	}

	// Remove obsolete forms from page resource dict.
	if err := removeForms(ctx, resDict, forms, i); err != nil {
		return false, fmt.Errorf("remove XObject resources: %w", err)
	}
	return true, nil
}

func locatePageContentAndResourceDict(ctx *model.Context, pageNr int) (types.Object, *types.IndirectRef, types.Dict, error) {
	consolidateRes := false
	d, pageDictIndRef, _, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("page dictionary: %w", err)
	}
	if d == nil {
		return nil, nil, nil, errors.New("missing page dictionary")
	}
	if pageDictIndRef == nil {
		return nil, nil, nil, errors.New("missing page dictionary reference")
	}

	o, found := d.Find("Resources")
	if !found {
		return nil, nil, nil, fmt.Errorf("page %d: no resource dict found", pageNr)
	}

	resDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("resource dictionary: dereference: %w", err)
	}
	if resDict == nil {
		return nil, nil, nil, fmt.Errorf("page %d: missing resource dictionary", pageNr)
	}

	o, found = d.Find("Contents")
	if !found {
		return nil, nil, nil, fmt.Errorf("page %d: no page watermark found", pageNr)
	}

	return o, pageDictIndRef, resDict, nil
}

func removeArtifactsFromStream(ctx *model.Context, sd types.StreamDict, entry *model.XRefTableEntry, resDict types.Dict, pageNr int) (bool, types.StreamDict, error) {
	found, err := removeArtifactsFromPage(ctx, &sd, resDict, pageNr)
	if err != nil {
		return false, sd, err
	}
	if found && entry != nil {
		entry.Object = sd
	}
	return found, sd, nil
}

func removeArtifactsFromContentRef(ctx *model.Context, o types.Object, resDict types.Dict, pageNr, pos int) (bool, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return false, fmt.Errorf("content array entry %d: expected indirect reference, got %T", pos, o)
	}
	entry, obj, objNr, err := contentObjectForIndRef(ctx, ir)
	if err != nil {
		return false, fmt.Errorf("content array entry %d: %w", pos, err)
	}
	sd, ok := obj.(types.StreamDict)
	if !ok {
		return false, fmt.Errorf("content array entry %d obj#%d: expected stream dictionary, got %T", pos, objNr, obj)
	}
	found, _, err := removeArtifactsFromStream(ctx, sd, entry, resDict, pageNr)
	if err != nil {
		return false, fmt.Errorf("content array entry %d obj#%d: %w", pos, objNr, err)
	}
	return found, nil
}

func removeArtifactsFromContentArray(ctx *model.Context, a types.Array, resDict types.Dict, pageNr int) (bool, error) {
	if len(a) == 0 {
		return false, nil
	}
	found, err := removeArtifactsFromContentRef(ctx, a[0], resDict, pageNr, 1)
	if err != nil || len(a) == 1 {
		return found, err
	}
	foundLast, err := removeArtifactsFromContentRef(ctx, a[len(a)-1], resDict, pageNr, len(a))
	return found || foundLast, err
}

func removeArtifacts1(ctx *model.Context, o types.Object, entry *model.XRefTableEntry, resDict types.Dict, pageNr int) (bool, types.Object, error) {
	switch o := o.(type) {
	case types.StreamDict:
		found, sd, err := removeArtifactsFromStream(ctx, o, entry, resDict, pageNr)
		return found, sd, err
	case types.Array:
		found, err := removeArtifactsFromContentArray(ctx, o, resDict, pageNr)
		return found, o, err
	default:
		return false, o, fmt.Errorf("Contents: expected stream dictionary or array, got %T", o)
	}
}

func removePageWatermark(ctx *model.Context, pageNr int) (bool, error) {
	o, pageDictIndRef, resDict, err := locatePageContentAndResourceDict(ctx, pageNr)
	if err != nil {
		return false, err
	}

	var entry *model.XRefTableEntry

	ir, ok := o.(types.IndirectRef)
	if ok {
		entry, o, _, err = contentObjectForIndRef(ctx, ir)
		if err != nil {
			return false, fmt.Errorf("Contents: %w", err)
		}
	}

	found, patchedContents, err := removeArtifacts1(ctx, o, entry, resDict, pageNr)
	if err != nil {
		return false, fmt.Errorf("remove page artifacts: %w", err)
	}
	if found && entry == nil {
		d, _, _, err := ctx.PageDict(pageNr, false)
		if err != nil {
			return false, fmt.Errorf("store direct content stream: page dictionary: %w", err)
		}
		if d == nil {
			return false, errors.New("store direct content stream: missing page dictionary")
		}
		d["Contents"] = patchedContents
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
			return false, fmt.Errorf("link annotations: page dictionary: %w", err)
		}
		if d == nil {
			return false, errors.New("link annotations: missing page dictionary")
		}
		objNr := pageDictIndRef.ObjectNumber.Value()
		if _, err = RemoveAnnotationsFromPageDict(ctx, nil, []string{"pdfcpu"}, nil, d, objNr, pageNr, false); err != nil {
			return false, fmt.Errorf("link annotations: remove: %w", err)
		}
	}

	return found, nil
}

func locateOCGs(ctx *model.Context) (types.Array, error) {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, fmt.Errorf("optional content properties: catalog: %w", err)
	}
	if rootDict == nil {
		return nil, errors.New("optional content properties: missing catalog")
	}

	o, ok := rootDict.Find("OCProperties")
	if !ok {
		return nil, errNoWatermark
	}

	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, fmt.Errorf("optional content properties: dereference dict: %w", err)
	}
	if d == nil {
		return nil, errors.New("optional content properties: missing dict")
	}

	o, found := d.Find("OCGs")
	if !found {
		return nil, errNoWatermark
	}

	a, err := ctx.DereferenceArray(o)
	if err != nil {
		return nil, fmt.Errorf("optional content groups: dereference array: %w", err)
	}
	return a, nil
}

func detectStampOCG(ctx *model.Context, arr types.Array) error {
	found, err := containsWatermarkOCG(ctx, arr)
	if err != nil {
		return err
	}
	if found {
		return nil
	}
	return errNoWatermark
}

func removePageWatermarks(ctx *model.Context, selectedPages types.IntSet) error {
	var removed bool

	for _, pageNr := range sortedWatermarkPages(selectedPages) {
		if !selectedPages[pageNr] {
			continue
		}

		ok, err := removePageWatermark(ctx, pageNr)
		if err != nil {
			return fmt.Errorf("page %d: %w", pageNr, err)
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
	if err := validateWatermarkContext(ctx); err != nil {
		return err
	}
	if log.DebugEnabled() {
		log.Debug.Printf("RemoveWatermarks\n")
	}

	arr, err := locateOCGs(ctx)
	if err != nil {
		return fmt.Errorf("locate optional content groups: %w", err)
	}

	if err := detectStampOCG(ctx, arr); err != nil {
		return fmt.Errorf("identify watermark optional content group: %w", err)
	}

	if err := removePageWatermarks(ctx, selectedPages); err != nil {
		return fmt.Errorf("remove page watermarks: %w", err)
	}
	return nil
}

func detectArtifacts(sd *types.StreamDict) (bool, error) {
	if sd == nil {
		return false, errors.New("missing content stream")
	}
	if err := sd.Decode(); err != nil {
		return false, fmt.Errorf("decode content stream: %w", err)
	}
	// Watermarks may begin or end the content stream.
	i := strings.Index(string(sd.Content), "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC")
	return i >= 0, nil
}

func detectArtifactsFromContentRef(ctx *model.Context, o types.Object, pos int) (bool, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return false, fmt.Errorf("content array entry %d: expected indirect reference, got %T", pos, o)
	}
	_, obj, objNr, err := contentObjectForIndRef(ctx, ir)
	if err != nil {
		return false, fmt.Errorf("content array entry %d: %w", pos, err)
	}
	sd, ok := obj.(types.StreamDict)
	if !ok {
		return false, fmt.Errorf("content array entry %d obj#%d: expected stream dictionary, got %T", pos, objNr, obj)
	}
	found, err := detectArtifacts(&sd)
	if err != nil {
		return false, fmt.Errorf("content array entry %d obj#%d: %w", pos, objNr, err)
	}
	return found, nil
}

func detectArtifactsFromContentArray(ctx *model.Context, a types.Array) (bool, error) {
	if len(a) == 0 {
		return false, nil
	}
	found, err := detectArtifactsFromContentRef(ctx, a[0], 1)
	if err != nil || found || len(a) == 1 {
		return found, err
	}
	return detectArtifactsFromContentRef(ctx, a[len(a)-1], len(a))
}

func detectArtifactsFromContents(ctx *model.Context, o types.Object) (bool, error) {
	if ir, ok := o.(types.IndirectRef); ok {
		_, obj, _, err := contentObjectForIndRef(ctx, ir)
		if err != nil {
			return false, fmt.Errorf("Contents: %w", err)
		}
		o = obj
	}
	switch o := o.(type) {
	case types.StreamDict:
		return detectArtifacts(&o)
	case types.Array:
		return detectArtifactsFromContentArray(ctx, o)
	default:
		return false, fmt.Errorf("Contents: expected stream dictionary or array, got %T", o)
	}
}

func findPageWatermarks(ctx *model.Context, pageDictIndRef *types.IndirectRef) (bool, error) {
	if pageDictIndRef == nil {
		return false, errors.New("missing page dictionary reference")
	}
	d, err := ctx.DereferenceDict(*pageDictIndRef)
	if err != nil {
		return false, fmt.Errorf("page dictionary: %w", err)
	}
	if d == nil {
		return false, errors.New("missing page dictionary")
	}

	o, found := d.Find("Contents")
	if !found || o == nil {
		return false, nil
	}
	return detectArtifactsFromContents(ctx, o)
}

func detectPageTreeChildWatermarks(ctx *model.Context, o types.Object, pos int) error {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return fmt.Errorf("page tree child %d: expected indirect reference, got %T", pos, o)
	}
	d, err := ctx.DereferenceDict(ir)
	if err != nil {
		return fmt.Errorf("page tree child %d obj#%d: dereference dictionary: %w", pos, ir.ObjectNumber.Value(), err)
	}
	if d == nil {
		return fmt.Errorf("page tree child %d obj#%d: missing dictionary", pos, ir.ObjectNumber.Value())
	}
	typ := d.Type()
	if typ == nil {
		return nil
	}
	if *typ == "Pages" {
		if err := detectPageTreeWatermarks(ctx, &ir); err != nil {
			return fmt.Errorf("page tree child %d obj#%d: nested pages: %w", pos, ir.ObjectNumber.Value(), err)
		}
		return nil
	}
	if *typ != "Page" {
		return nil
	}
	found, err := findPageWatermarks(ctx, &ir)
	if err != nil {
		return fmt.Errorf("page tree child %d obj#%d: page watermarks: %w", pos, ir.ObjectNumber.Value(), err)
	}
	ctx.Watermarked = found
	return nil
}

func detectPageTreeWatermarks(ctx *model.Context, root *types.IndirectRef) error {
	if root == nil {
		return errors.New("page tree: missing root reference")
	}
	d, err := ctx.DereferenceDict(*root)
	if err != nil {
		return fmt.Errorf("page tree: dereference root: %w", err)
	}
	if d == nil {
		return errors.New("page tree: missing root dictionary")
	}
	o, found := d.Find("Kids")
	if !found {
		return nil
	}
	kids, err := ctx.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("page tree: dereference Kids: %w", err)
	}
	for i, o := range kids {
		if ctx.Watermarked {
			return nil
		}
		if o == nil {
			continue
		}
		if err := detectPageTreeChildWatermarks(ctx, o, i+1); err != nil {
			return err
		}
	}
	return nil
}

func validateWatermarkContext(ctx *model.Context) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}
	return nil
}

// DetectPageTreeWatermarks checks xRefTable's page tree for watermarks
// and records the result to xRefTable.Watermarked.
func DetectPageTreeWatermarks(ctx *model.Context) error {
	if err := validateWatermarkContext(ctx); err != nil {
		return err
	}

	root, err := ctx.Pages()
	if err != nil {
		return fmt.Errorf("page tree watermarks: pages root: %w", err)
	}
	if root == nil {
		return errors.New("page tree watermarks: missing pages root")
	}
	if err := detectPageTreeWatermarks(ctx, root); err != nil {
		return fmt.Errorf("page tree watermarks: %w", err)
	}
	return nil
}

func isWatermarkOCG(ctx *model.Context, o types.Object, pos int) (bool, error) {
	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return false, fmt.Errorf("optional content group %d: dereference dictionary: %w", pos, err)
	}
	if d == nil {
		return false, nil
	}
	o, found := d.Find("Type")
	if !found {
		return false, nil
	}
	n, err := ctx.Dereference(o)
	if err != nil {
		return false, fmt.Errorf("optional content group %d: type: %w", pos, err)
	}
	typ, ok := n.(types.Name)
	if !ok || typ != "OCG" {
		return false, nil
	}
	o, found = d.Find("Name")
	if !found {
		return false, nil
	}
	n, err = ctx.Dereference(o)
	if err != nil {
		return false, fmt.Errorf("optional content group %d: name: %w", pos, err)
	}
	name, ok := n.(types.StringLiteral)
	return ok && (name == "Background" || name == "Watermark"), nil
}

func containsWatermarkOCG(ctx *model.Context, a types.Array) (bool, error) {
	for i, o := range a {
		if o == nil {
			continue
		}
		found, err := isWatermarkOCG(ctx, o, i+1)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// DetectWatermarks checks ctx for watermarks
// and records the result to xRefTable.Watermarked.
func DetectWatermarks(ctx *model.Context) error {
	if err := validateWatermarkContext(ctx); err != nil {
		return err
	}
	a, err := locateOCGs(ctx)
	if err != nil {
		if errors.Is(err, errNoWatermark) {
			ctx.Watermarked = false
			return nil
		}
		return fmt.Errorf("optional content groups: %w", err)
	}
	found, err := containsWatermarkOCG(ctx, a)
	if err != nil {
		return err
	}

	if !found {
		ctx.Watermarked = false
		return nil
	}

	return DetectPageTreeWatermarks(ctx)
}

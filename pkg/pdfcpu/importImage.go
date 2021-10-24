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

	"github.com/pdfcpu/pdfcpu/pkg/types"
	"github.com/pkg/errors"
)

type importParamMap map[string]func(string, *Import) error

// Handle applies parameter completion and if successful
// parses the parameter values into import.
func (m importParamMap) Handle(paramPrefix, paramValueStr string, imp *Import) error {

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
		return errors.Errorf("pdfcpu: unknown parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, imp)
}

var impParamMap = importParamMap{
	"dimensions":      parseDimensionsImp,
	"dpi":             parseDPI,
	"formsize":        parsePageFormatImp,
	"papersize":       parsePageFormatImp,
	"position":        parsePositionAnchorImp,
	"offset":          parsePositionOffsetImp,
	"scalefactor":     parseScaleFactorImp,
	"gray":            parseGray,
	"sepia":           parseSepia,
	"backgroundcolor": parseImportBackgroundColor,
	"bgcolor":         parseImportBackgroundColor,
}

// Import represents the command details for the command "ImportImage".
type Import struct {
	PageDim  *Dim        // page dimensions in display unit.
	PageSize string      // one of A0,A1,A2,A3,A4(=default),A5,A6,A7,A8,Letter,Legal,Ledger,Tabloid,Executive,ANSIC,ANSID,ANSIE.
	UserDim  bool        // true if one of dimensions or paperSize provided overriding the default.
	DPI      int         // destination resolution to apply in dots per inch.
	Pos      Anchor      // position anchor, one of tl,tc,tr,l,c,r,bl,bc,br,full.
	Dx, Dy   int         // anchor offset.
	Scale    float64     // relative scale factor. 0 <= x <= 1
	ScaleAbs bool        // true for absolute scaling.
	InpUnit  DisplayUnit // input display unit.
	Gray     bool        // true for rendering in Gray.
	Sepia    bool
	BgColor  *SimpleColor // background color
}

// DefaultImportConfig returns the default configuration.
func DefaultImportConfig() *Import {
	return &Import{
		PageDim:  PaperSize["A4"],
		PageSize: "A4",
		Pos:      Full,
		Scale:    0.5,
		InpUnit:  POINTS,
	}
}

func (imp Import) String() string {

	sc := "relative"
	if imp.ScaleAbs {
		sc = "absolute"
	}

	return fmt.Sprintf("Import conf: %s %s, pos=%s, dx=%d, dy=%d, scaling: %.1f %s\n",
		imp.PageSize, *imp.PageDim, imp.Pos, imp.Dx, imp.Dy, imp.Scale, sc)
}

func ParsePageFormat(v string) (*Dim, string, error) {

	// Optional: appended last letter L indicates landscape mode.
	// Optional: appended last letter P indicates portrait mode.
	// eg. A4L means A4 in landscape mode whereas A4 defaults to A4P
	// The default mode is defined implicitly via PaperSize dimensions.

	var land, port bool

	if strings.HasSuffix(v, "L") {
		v = v[:len(v)-1]
		land = true
	} else if strings.HasSuffix(v, "P") {
		v = v[:len(v)-1]
		port = true
	}

	d, ok := PaperSize[v]
	if !ok {
		return nil, v, errors.Errorf("pdfcpu: page format %s is unsupported.\n", v)
	}

	if d.Portrait() && land || d.Landscape() && port {
		d.Width, d.Height = d.Height, d.Width
	}

	return d, v, nil
}

func parsePageFormatImp(s string, imp *Import) (err error) {
	if imp.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	imp.PageDim, imp.PageSize, err = ParsePageFormat(s)
	imp.UserDim = true
	return err
}

func parsePageDim(v string, u DisplayUnit) (*Dim, string, error) {

	ss := strings.Split(v, " ")
	if len(ss) != 2 {
		return nil, v, errors.Errorf("pdfcpu: illegal dimension string: need 2 positive values, %s\n", v)
	}

	w, err := strconv.ParseFloat(ss[0], 64)
	if err != nil || w <= 0 {
		return nil, v, errors.Errorf("pdfcpu: dimension X must be a positiv numeric value: %s\n", ss[0])
	}

	h, err := strconv.ParseFloat(ss[1], 64)
	if err != nil || h <= 0 {
		return nil, v, errors.Errorf("pdfcpu: dimension Y must be a positiv numeric value: %s\n", ss[1])
	}

	d := Dim{toUserSpace(w, u), toUserSpace(h, u)}

	return &d, "", nil
}

func parseDimensionsImp(s string, imp *Import) (err error) {
	if imp.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	imp.PageDim, imp.PageSize, err = parsePageDim(s, imp.InpUnit)
	imp.UserDim = true
	return err
}

type Anchor int

func (a Anchor) String() string {

	switch a {

	case TopLeft:
		return "top left"

	case TopCenter:
		return "top center"

	case TopRight:
		return "top right"

	case Left:
		return "left"

	case Center:
		return "center"

	case Right:
		return "right"

	case BottomLeft:
		return "bottom left"

	case BottomCenter:
		return "bottom center"

	case BottomRight:
		return "bottom right"

	case Full:
		return "full"

	}

	return ""
}

// These are the defined anchors for relative positioning.
const (
	TopLeft Anchor = iota
	TopCenter
	TopRight
	Left
	Center // default
	Right
	BottomLeft
	BottomCenter
	BottomRight
	Full // special case, no anchor needed, imageSize = pageSize
)

func parsePositionAnchor(s string) (Anchor, error) {
	var a Anchor
	switch s {
	case "tl", "topleft", "top-left":
		a = TopLeft
	case "tc", "topcenter", "top-center":
		a = TopCenter
	case "tr", "topright", "top-right":
		a = TopRight
	case "l", "left":
		a = Left
	case "c", "center":
		a = Center
	case "r", "right":
		a = Right
	case "bl", "bottomleft", "bottom-left":
		a = BottomLeft
	case "bc", "bottomcenter", "bottom-center":
		a = BottomCenter
	case "br", "bottomright", "bottom-right":
		a = BottomRight
	case "f", "full":
		a = Full
	default:
		return a, errors.Errorf("pdfcpu: unknown position anchor: %s", s)
	}
	return a, nil
}

func parsePositionAnchorImp(s string, imp *Import) error {
	a, err := parsePositionAnchor(s)
	if err != nil {
		return err
	}
	imp.Pos = a
	return nil
}

func parsePositionOffsetImp(s string, imp *Import) error {

	d := strings.Split(s, " ")
	if len(d) != 2 {
		return errors.Errorf("pdfcpu: illegal position offset string: need 2 numeric values, %s\n", s)
	}

	f, err := strconv.ParseFloat(d[0], 64)
	if err != nil {
		return err
	}
	imp.Dx = int(toUserSpace(f, imp.InpUnit))

	f, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	imp.Dy = int(toUserSpace(f, imp.InpUnit))

	return nil
}

func parseScaleFactorImp(s string, imp *Import) (err error) {
	imp.Scale, imp.ScaleAbs, err = parseScaleFactor(s)
	return err
}

func parseDPI(s string, imp *Import) (err error) {
	imp.DPI, err = strconv.Atoi(s)
	return err
}

func parseGray(s string, imp *Import) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		imp.Gray = true
	case "off", "false", "f":
		imp.Gray = false
	default:
		return errors.New("pdfcpu: import gray, please provide one of: on/off true/false")
	}

	return nil
}

func parseSepia(s string, imp *Import) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		imp.Sepia = true
	case "off", "false", "f":
		imp.Sepia = false
	default:
		return errors.New("pdfcpu: import sepia, please provide one of: on/off true/false")
	}

	return nil
}

func parseImportBackgroundColor(s string, imp *Import) error {
	c, err := ParseColor(s)
	if err != nil {
		return err
	}
	imp.BgColor = &c
	return nil
}

// ParseImportDetails parses an Import command string into an internal structure.
func ParseImportDetails(s string, u DisplayUnit) (*Import, error) {

	if s == "" {
		return nil, nil
	}

	imp := DefaultImportConfig()
	imp.InpUnit = u

	ss := strings.Split(s, ",")

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errors.New("pdfcpu: Invalid import configuration string. Please consult pdfcpu help import")
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := impParamMap.Handle(paramPrefix, paramValueStr, imp); err != nil {
			return nil, err
		}
	}

	return imp, nil
}

// AppendPageTree appends a pagetree d1 to page tree d2.
func AppendPageTree(d1 *IndirectRef, countd1 int, d2 Dict) error {
	a := d2.ArrayEntry("Kids")
	a = append(a, *d1)
	d2.Update("Kids", a)
	return d2.IncrementBy("Count", countd1)
}

func lowerLeftCorner(vpw, vph, bbw, bbh float64, a Anchor) types.Point {

	var p types.Point

	switch a {

	case TopLeft:
		p.X = 0
		p.Y = vph - bbh

	case TopCenter:
		p.X = vpw/2 - bbw/2
		p.Y = vph - bbh

	case TopRight:
		p.X = vpw - bbw
		p.Y = vph - bbh

	case Left:
		p.X = 0
		p.Y = vph/2 - bbh/2

	case Center:
		p.X = vpw/2 - bbw/2
		p.Y = vph/2 - bbh/2

	case Right:
		p.X = vpw - bbw
		p.Y = vph/2 - bbh/2

	case BottomLeft:
		p.X = 0
		p.Y = 0

	case BottomCenter:
		p.X = vpw/2 - bbw/2
		p.Y = 0

	case BottomRight:
		p.X = vpw - bbw
		p.Y = 0
	}

	return p
}

func importImagePDFBytes(wr io.Writer, pageDim *Dim, imgWidth, imgHeight float64, imp *Import) {

	vpw := float64(pageDim.Width)
	vph := float64(pageDim.Height)

	bb := RectForDim(vpw, vph)
	if imp.BgColor != nil {
		FillRectNoBorder(wr, bb, *imp.BgColor)
	}

	if imp.Pos == Full {
		// The bounding box equals the page dimensions.
		bb.UR.X = bb.Width()
		bb.UR.Y = bb.UR.X / bb.AspectRatio()
		fmt.Fprintf(wr, "q %f 0 0 %f 0 0 cm /Im0 Do Q", bb.Width(), bb.Height())
		return
	}

	if imp.DPI > 0 {
		// NOTE: We could also set "UserUnit" in the page dict.
		imgWidth *= float64(72) / float64(imp.DPI)
		imgHeight *= float64(72) / float64(imp.DPI)
	}

	bb = RectForDim(imgWidth, imgHeight)
	ar := bb.AspectRatio()

	if imp.ScaleAbs {
		bb.UR.X = imp.Scale * bb.Width()
		bb.UR.Y = bb.UR.X / ar
	} else {
		if ar >= 1 {
			bb.UR.X = imp.Scale * vpw
			bb.UR.Y = bb.UR.X / ar
		} else {
			bb.UR.Y = imp.Scale * vph
			bb.UR.X = bb.UR.Y * ar
		}
	}

	m := identMatrix

	// Scale
	m[0][0] = bb.Width()
	m[1][1] = bb.Height()

	// Translate
	ll := lowerLeftCorner(vpw, vph, bb.Width(), bb.Height(), imp.Pos)
	m[2][0] = ll.X + float64(imp.Dx)
	m[2][1] = ll.Y + float64(imp.Dy)

	fmt.Fprintf(wr, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /Im0 Do Q",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])
}

// NewPageForImage creates a new page dict in xRefTable for given image reader r.
func NewPageForImage(xRefTable *XRefTable, r io.Reader, parentIndRef *IndirectRef, imp *Import) (*IndirectRef, error) {

	// create image dict.
	imgIndRef, w, h, err := CreateImageResource(xRefTable, r, imp.Gray, imp.Sepia)
	if err != nil {
		return nil, err
	}

	// create resource dict for XObject.
	d := Dict(
		map[string]Object{
			"ProcSet": NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
			"XObject": Dict(map[string]Object{"Im0": *imgIndRef}),
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	dim := &Dim{float64(w), float64(h)}
	if imp.Pos != Full {
		dim = imp.PageDim
	}
	// mediabox = physical page dimensions
	mediaBox := RectForDim(dim.Width, dim.Height)

	var buf bytes.Buffer
	importImagePDFBytes(&buf, dim, float64(w), float64(h), imp)
	sd, _ := xRefTable.NewStreamDictForBuf(buf.Bytes())
	if err = sd.Encode(); err != nil {
		return nil, err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

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

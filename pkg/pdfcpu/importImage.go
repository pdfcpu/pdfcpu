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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type importParamMap map[string]func(string, *Import) error

// Handle applies parameter completion and if successful
// parses the parameter values into import.
func (m importParamMap) Handle(paramPrefix, paramValueStr string, imp *Import) error {

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
	PageDim  *types.Dim        // page dimensions in display unit.
	PageSize string            // one of A0,A1,A2,A3,A4(=default),A5,A6,A7,A8,Letter,Legal,Ledger,Tabloid,Executive,ANSIC,ANSID,ANSIE.
	UserDim  bool              // true if one of dimensions or paperSize provided overriding the default.
	DPI      int               // destination resolution to apply in dots per inch.
	Pos      types.Anchor      // position anchor, one of tl,tc,tr,l,c,r,bl,bc,br,full.
	Dx, Dy   float64           // anchor offset.
	Scale    float64           // relative scale factor. 0 <= x <= 1
	ScaleAbs bool              // true for absolute scaling.
	InpUnit  types.DisplayUnit // input display unit.
	Gray     bool              // true for rendering in Gray.
	Sepia    bool
	BgColor  *color.SimpleColor // background color
}

// DefaultImportConfig returns the default configuration.
func DefaultImportConfig() *Import {
	return &Import{
		PageDim:  types.PaperSize["A4"],
		PageSize: "A4",
		Pos:      types.Full,
		Scale:    0.5,
		InpUnit:  types.POINTS,
	}
}

func (imp Import) String() string {

	sc := "relative"
	if imp.ScaleAbs {
		sc = "absolute"
	}

	return fmt.Sprintf("Import conf: %s %s, pos=%s, dx=%f.2, dy=%f.2, scaling: %.1f %s\n",
		imp.PageSize, *imp.PageDim, imp.Pos, imp.Dx, imp.Dy, imp.Scale, sc)
}

func parsePageFormatImp(s string, imp *Import) (err error) {
	if imp.UserDim {
		return errors.New("pdfcpu: only one of formsize(papersize) or dimensions allowed")
	}
	imp.PageDim, imp.PageSize, err = types.ParsePageFormat(s)
	imp.UserDim = true
	return err
}

func parsePageDim(v string, u types.DisplayUnit) (*types.Dim, string, error) {

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

	d := types.Dim{Width: types.ToUserSpace(w, u), Height: types.ToUserSpace(h, u)}

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

func parsePositionAnchorImp(s string, imp *Import) error {
	a, err := types.ParsePositionAnchor(s)
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
	imp.Dx = types.ToUserSpace(f, imp.InpUnit)

	f, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	imp.Dy = types.ToUserSpace(f, imp.InpUnit)

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
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	imp.BgColor = &c
	return nil
}

// ParseImportDetails parses an Import command string into an internal structure.
func ParseImportDetails(s string, u types.DisplayUnit) (*Import, error) {

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

func importImagePDFBytes(wr io.Writer, pageDim *types.Dim, imgWidth, imgHeight float64, imp *Import) {

	vpw := float64(pageDim.Width)
	vph := float64(pageDim.Height)
	vp := types.RectForDim(vpw, vph)

	if imp.BgColor != nil {
		draw.FillRectNoBorder(wr, vp, *imp.BgColor)
	}

	if imp.Pos == types.Full {
		fmt.Fprintf(wr, "q %f 0 0 %f 0 0 cm /Im0 Do Q", vp.Width(), vp.Height())
		return
	}

	if imp.DPI > 0 {
		// NOTE: We could also set "UserUnit" in the page dict.
		imgWidth *= float64(72) / float64(imp.DPI)
		imgHeight *= float64(72) / float64(imp.DPI)
	}

	bb := types.RectForDim(imgWidth, imgHeight)
	ar := bb.AspectRatio()

	if imp.ScaleAbs {
		bb.UR.X = imp.Scale * bb.Width()
		bb.UR.Y = bb.UR.X / ar
	} else {
		if ar >= 1 {
			if vp.AspectRatio() <= 1 {
				bb.UR.X = imp.Scale * vpw
				bb.UR.Y = bb.UR.X / ar
			} else {
				if ar >= vp.AspectRatio() {
					bb.UR.X = imp.Scale * vpw
					bb.UR.Y = bb.UR.X / ar
				} else {
					bb.UR.Y = imp.Scale * vph
					bb.UR.X = bb.UR.Y * ar
				}
			}
		} else {
			if vp.AspectRatio() >= 1 {
				bb.UR.Y = imp.Scale * vph
				bb.UR.X = bb.UR.Y * ar
			} else {
				if ar <= vp.AspectRatio() {
					bb.UR.Y = imp.Scale * vph
					bb.UR.X = bb.UR.Y * ar
				} else {
					bb.UR.X = imp.Scale * vpw
					bb.UR.Y = bb.UR.X / ar
				}
			}
		}
	}

	m := matrix.IdentMatrix

	// Scale
	m[0][0] = bb.Width()
	m[1][1] = bb.Height()

	// Translate
	ll := model.LowerLeftCorner(vp, bb.Width(), bb.Height(), imp.Pos)
	m[2][0] = ll.X + imp.Dx
	m[2][1] = ll.Y + imp.Dy

	fmt.Fprintf(wr, "q %.5f %.5f %.5f %.5f %.5f %.5f cm /Im0 Do Q",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])
}

// NewPageForImage creates a new page dict in xRefTable for given image reader r.
func NewPageForImage(xRefTable *model.XRefTable, r io.Reader, parentIndRef *types.IndirectRef, imp *Import) (*types.IndirectRef, error) {

	// create image dict.
	imgIndRef, w, h, err := model.CreateImageResource(xRefTable, r, imp.Gray, imp.Sepia)
	if err != nil {
		return nil, err
	}

	// create resource dict for XObject.
	d := types.Dict(
		map[string]types.Object{
			"ProcSet": types.NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
			"XObject": types.Dict(map[string]types.Object{"Im0": *imgIndRef}),
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	dim := &types.Dim{Width: float64(w), Height: float64(h)}
	if imp.Pos != types.Full {
		dim = imp.PageDim
	}
	// mediabox = physical page dimensions
	mediaBox := types.RectForDim(dim.Width, dim.Height)

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

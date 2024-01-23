/*
Copyright 2023 The pdfcpu Authors.

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

package model

import (
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type Cut struct {
	Hor      []float64          // Horizontal cut points
	Vert     []float64          // Vertical cut points
	Scale    float64            // scale factor x > 1 (poster)
	PageSize string             // paper/form size eg. A2,A3,A4,Legal,Ledger,...
	PageDim  *types.Dim         // page dimensions in display unit
	Unit     types.DisplayUnit  // display unit
	UserDim  bool               // true if dimensions set by dim rather than formsize
	Border   bool               // true to render crop box
	Margin   float64            // glue area in display unit
	BgColor  *color.SimpleColor // background color
	Origin   types.Corner       // one of 4 page corners, default = UpperLeft
}

type cutParameterMap map[string]func(string, *Cut) error

func parseHorCut(v string, cut *Cut) (err error) {

	for _, s := range strings.Split(v, " ") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return errors.Errorf("pdfcpu: cut position must be a float value: %s\n", s)
		}
		if f <= 0 || f >= 1 {
			return errors.Errorf("pdfcpu: invalid cut poistion %.2f: 0 < i < 1.0\n", f)
		}
		cut.Hor = append(cut.Hor, f)
	}

	return nil
}

func parseVertCut(v string, cut *Cut) (err error) {

	for _, s := range strings.Split(v, " ") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return errors.Errorf("pdfcpu: cut position must be a float value: %s\n", s)
		}
		if f <= 0 || f >= 1 {
			return errors.Errorf("pdfcpu: invalid cut poistion %.2f: 0 < i < 1.0\n", f)
		}
		cut.Vert = append(cut.Vert, f)
	}

	return nil
}

func parsePageDimCut(v string, u types.DisplayUnit) (*types.Dim, string, error) {

	ss := strings.Split(v, " ")
	if len(ss) != 2 {
		return nil, v, errors.Errorf("pdfcpu: illegal dimension string: need 2 values one may be 0, %s\n", v)
	}

	w, err := strconv.ParseFloat(ss[0], 64)
	if err != nil || w < 0 {
		return nil, v, errors.Errorf("pdfcpu: dimension width must be >= 0: %s\n", ss[0])
	}

	h, err := strconv.ParseFloat(ss[1], 64)
	if err != nil || h < 0 {
		return nil, v, errors.Errorf("pdfcpu: dimension height must >= 0: %s\n", ss[1])
	}

	d := types.Dim{Width: types.ToUserSpace(w, u), Height: types.ToUserSpace(h, u)}

	return &d, "", nil
}

func parseDimensionsCut(s string, cut *Cut) (err error) {
	cut.PageDim, _, err = parsePageDimCut(s, cut.Unit)
	if err != nil {
		return err
	}
	cut.UserDim = true
	return nil
}

func parsePageFormatCut(s string, cut *Cut) error {

	// Optional: appended last letter L indicates landscape mode.
	// Optional: appended last letter P indicates portrait mode.
	// eg. A4L means A4 in landscape mode whereas A4 defaults to A4P
	// The default mode is defined implicitly via PaperSize dimensions.

	var landscape, portrait bool

	v := s
	if strings.HasSuffix(v, "L") {
		v = v[:len(v)-1]
		landscape = true
	} else if strings.HasSuffix(v, "P") {
		v = v[:len(v)-1]
		portrait = true
	}

	d, ok := types.PaperSize[v]
	if !ok {
		return errors.Errorf("pdfcpu: page format %s is unsupported.\n", v)
	}

	if (d.Portrait() && landscape) || (d.Landscape() && portrait) {
		d.Width, d.Height = d.Height, d.Width
	}

	cut.PageDim = d
	cut.PageSize = v

	return nil
}

func parseScaleFactorCut(s string, cut *Cut) (err error) {

	sc, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("pdfcpu: scale factor must be a float value: %s\n", s)
	}

	if sc < 1 {
		return errors.Errorf("pdfcpu: invalid scale factor %.2f: i >= 1.0\n", sc)
	}

	cut.Scale = sc
	return nil
}

func parseBackgroundColorCut(s string, cut *Cut) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	cut.BgColor = &c
	return nil
}

func parseBorderCut(s string, cut *Cut) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		cut.Border = true
	case "off", "false", "f":
		cut.Border = false
	default:
		return errors.New("pdfcpu: cut border, please provide one of: on/off true/false t/f")
	}

	return nil
}

func parseMarginCut(s string, cut *Cut) error {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}

	if f < 0 {
		return errors.New("pdfcpu: cut margin, Please provide a positive value")
	}

	cut.Margin = types.ToUserSpace(f, cut.Unit)

	return nil
}

var CutParamMap = cutParameterMap{
	"horizontalCut": parseHorCut,
	"verticalCut":   parseVertCut,
	"dimensions":    parseDimensionsCut,
	"formsize":      parsePageFormatCut,
	"papersize":     parsePageFormatCut,
	"scalefactor":   parseScaleFactorCut,
	"border":        parseBorderCut,
	"margin":        parseMarginCut,
	"bgcolor":       parseBackgroundColorCut,
}

// Handle applies parameter completion and on success parse parameter values into resize.
func (m cutParameterMap) Handle(paramPrefix, paramValueStr string, cut *Cut) error {

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

	return m[param](paramValueStr, cut)
}

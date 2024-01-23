/*
Copyright 2021 The pdfcpu Authors.

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

type Resize struct {
	Scale         float64            // scale factor x > 0, x > 1 enlarges, x < 1 shrinks down
	Unit          types.DisplayUnit  // display unit
	PageDim       *types.Dim         // page dimensions in display unit
	PageSize      string             // paper size eg. A2,A3,A4,Legal,Ledger,...
	EnforceOrient bool               // enforce orientation of PageDim
	UserDim       bool               // true if dimensions set by dim rather than formsize
	Border        bool               // true to render original crop box
	BgColor       *color.SimpleColor // background color
}

func (r Resize) EnforceOrientation() bool {
	return r.EnforceOrient || strings.HasSuffix(r.PageSize, "P") || strings.HasSuffix(r.PageSize, "L")
}

func parsePageDimRes(v string, u types.DisplayUnit) (*types.Dim, string, error) {

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

func parseDimensionsRes(s string, res *Resize) (err error) {
	res.PageDim, _, err = parsePageDimRes(s, res.Unit)
	res.UserDim = true
	return err
}

func parseEnforceOrientation(s string, res *Resize) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		res.EnforceOrient = true
	case "off", "false", "f":
		res.EnforceOrient = false
	default:
		return errors.New("pdfcpu: enforce orientation, please provide one of: on/off true/false")
	}

	return nil
}

func parsePageFormatRes(s string, res *Resize) error {

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

	res.PageDim = d
	res.PageSize = v

	return nil
}

func parseScaleFactorSimple(s string) (float64, error) {

	sc, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.Errorf("pdfcpu: scale factor must be a float value: %s\n", s)
	}

	if sc <= 0 || sc == 1 {
		return 0, errors.Errorf("pdfcpu: invalid scale factor %.2f: 0.0 < i < 1.0 or i > 1.0\n", sc)
	}

	return sc, nil
}

func parseScaleFactorRes(s string, res *Resize) (err error) {
	res.Scale, err = parseScaleFactorSimple(s)
	return err
}

func parseBackgroundColorRes(s string, res *Resize) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	res.BgColor = &c
	return nil
}

func parseBorderRes(s string, res *Resize) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		res.Border = true
	case "off", "false", "f":
		res.Border = false
	default:
		return errors.New("pdfcpu: resize border, please provide one of: on/off true/false t/f")
	}

	return nil
}

type resizeParameterMap map[string]func(string, *Resize) error

var ResizeParamMap = resizeParameterMap{
	"dimensions":  parseDimensionsRes,
	"enforce":     parseEnforceOrientation,
	"formsize":    parsePageFormatRes,
	"papersize":   parsePageFormatRes,
	"scalefactor": parseScaleFactorRes,
	"bgcolor":     parseBackgroundColorRes,
	"border":      parseBorderRes,
}

// Handle applies parameter completion and on success parse parameter values into resize.
func (m resizeParameterMap) Handle(paramPrefix, paramValueStr string, res *Resize) error {

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

	return m[param](paramValueStr, res)
}

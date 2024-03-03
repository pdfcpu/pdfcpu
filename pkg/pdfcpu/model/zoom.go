/*
Copyright 2024 The pdfcpu Authors.

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

type Zoom struct {
	Factor  float64            // zoom factor x > 0, x > 1 zooms in, x < 1 zooms out
	HMargin float64            // horizontal margin implying some (usually negative) scale factor
	VMargin float64            // vertical margin implying some (usually negative) scale factor
	Unit    types.DisplayUnit  // display unit
	Border  bool               // border around page content when zooming out
	BgColor *color.SimpleColor // background color when zooming out
}

func (z *Zoom) EnsureFactorAndMargins(w, h float64) error {
	if z.Factor > 0 {
		z.HMargin = (w - (w * z.Factor)) / 2
		z.VMargin = (h - (h * z.Factor)) / 2
		return nil
	}
	if z.HMargin > 0 {
		z.Factor = (w - 2*z.HMargin) / w
		z.VMargin = (h - (h * z.Factor)) / 2
	}
	z.Factor = (h - 2*z.VMargin) / h
	z.HMargin = (w - (w * z.Factor)) / 2

	return nil
}

func parseHMargin(s string, zoom *Zoom) error {
	m, err := strconv.ParseFloat(s, 64)
	if err != nil || m == 0 {
		return errors.Errorf("pdfcpu: \"hmargin\" must be a numeric value and must not be 0, got %s\n", s)
	}

	if zoom.VMargin != 0 {
		return errors.New("pdfcpu: only one of \"hmargin\" and \"vmargin\" allowed")
	}

	zoom.HMargin = types.ToUserSpace(m, zoom.Unit)
	return nil
}

func parseVMargin(s string, zoom *Zoom) error {
	m, err := strconv.ParseFloat(s, 64)
	if err != nil || m == 0 {
		return errors.Errorf("pdfcpu: \"vmargin\" must be a numeric value and must not be 0, got %s\n", s)
	}

	if zoom.HMargin != 0 {
		return errors.New("pdfcpu: only one of \"hmargin\" and \"vmargin\" allowed")
	}

	zoom.VMargin = types.ToUserSpace(m, zoom.Unit)
	return nil
}

func parseZoomFactor(s string, zoom *Zoom) (err error) {
	zf, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("pdfcpu: zoom factor must be a float value: %s\n", s)
	}

	if zf <= 0 || zf == 1 {
		return errors.Errorf("pdfcpu: invalid zoom factor %.2f: 0.0 < i < 1.0 or i > 1.0\n", zf)
	}

	zoom.Factor = zf
	return err
}

func parseBackgroundColorZoom(s string, zoom *Zoom) error {
	c, err := color.ParseColor(s)
	if err != nil {
		return err
	}
	zoom.BgColor = &c
	return nil
}

func parseBorderZoom(s string, zoom *Zoom) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		zoom.Border = true
	case "off", "false", "f":
		zoom.Border = false
	default:
		return errors.New("pdfcpu: zoom border, please provide one of: on/off true/false t/f")
	}

	return nil
}

type zoomParameterMap map[string]func(string, *Zoom) error

var ZoomParamMap = zoomParameterMap{
	"factor":  parseZoomFactor,
	"hmargin": parseHMargin,
	"vmargin": parseVMargin,
	"bgcolor": parseBackgroundColorZoom,
	"border":  parseBorderZoom,
}

// Handle applies parameter completion and on success parse parameter values into zoom.
func (m zoomParameterMap) Handle(paramPrefix, paramValueStr string, zoom *Zoom) error {
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

	return m[param](paramValueStr, zoom)
}

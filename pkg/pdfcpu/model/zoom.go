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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Zoom represents page-content zoom configuration.
type Zoom struct {
	Factor  float64            // zoom factor x > 0, x > 1 zooms in, x < 1 zooms out
	HMargin float64            // horizontal margin implying some (usually negative) scale factor
	VMargin float64            // vertical margin implying some (usually negative) scale factor
	Unit    types.DisplayUnit  // display unit
	Border  bool               // border around page content when zooming out
	BgColor *color.SimpleColor // background color when zooming out
}

// EnsureFactorAndMargins ensures factor and margins.
func (z *Zoom) EnsureFactorAndMargins(w, h float64) error {
	if z.Factor > 0 {
		z.HMargin = (w - (w * z.Factor)) / 2
		z.VMargin = (h - (h * z.Factor)) / 2
		return nil
	}
	if z.HMargin != 0 {
		factor := (w - 2*z.HMargin) / w
		if factor <= 0 {
			return fmt.Errorf("horizontal margin %.2f yields non-positive zoom factor for page width %.2f", z.HMargin, w)
		}
		z.Factor = factor
		z.VMargin = (h - (h * factor)) / 2
		return nil
	}
	factor := (h - 2*z.VMargin) / h
	if factor <= 0 {
		return fmt.Errorf("vertical margin %.2f yields non-positive zoom factor for page height %.2f", z.VMargin, h)
	}
	z.Factor = factor
	z.HMargin = (w - (w * factor)) / 2
	return nil
}

func parseHMargin(s string, zoom *Zoom) error {
	m, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("\"hmargin\": parse numeric value %q: %w", s, err)
	}
	if m == 0 {
		return fmt.Errorf("\"hmargin\" must not be 0, got %s", s)
	}

	if zoom.VMargin != 0 {
		return errors.New("only one of \"hmargin\" and \"vmargin\" allowed")
	}

	zoom.HMargin = types.ToUserSpace(m, zoom.Unit)
	return nil
}

func parseVMargin(s string, zoom *Zoom) error {
	m, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("\"vmargin\": parse numeric value %q: %w", s, err)
	}
	if m == 0 {
		return fmt.Errorf("\"vmargin\" must not be 0, got %s", s)
	}

	if zoom.HMargin != 0 {
		return errors.New("only one of \"hmargin\" and \"vmargin\" allowed")
	}

	zoom.VMargin = types.ToUserSpace(m, zoom.Unit)
	return nil
}

func parseZoomFactor(s string, zoom *Zoom) error {
	zf, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("zoom factor: parse float value %q: %w", s, err)
	}

	if zf <= 0 || zf == 1 {
		return fmt.Errorf("invalid zoom factor %.2f: 0.0 < i < 1.0 or i > 1.0", zf)
	}

	zoom.Factor = zf
	return nil
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
		return errors.New("zoom border, please provide one of: on/off true/false t/f")
	}

	return nil
}

type zoomParameterMap map[string]func(string, *Zoom) error

// ZoomParamMap maps zoom configuration parameter names to parser functions.
var ZoomParamMap = zoomParameterMap{
	"factor":  parseZoomFactor,
	"hmargin": parseHMargin,
	"vmargin": parseVMargin,
	"bgcolor": parseBackgroundColorZoom,
	"border":  parseBorderZoom,
}

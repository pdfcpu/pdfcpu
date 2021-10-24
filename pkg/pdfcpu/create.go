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

package pdfcpu

import (
	"strings"

	"github.com/pkg/errors"
)

// Corner represents one of four rectangle corners.
type Corner int

// The four corners of a rectangle.
const (
	LowerLeft Corner = iota
	LowerRight
	UpperLeft
	UpperRight
)

func ParseOrigin(s string) (Corner, error) {
	var c Corner
	switch strings.ToLower(s) {
	case "ll", "lowerleft":
		c = LowerLeft
	case "lr", "lowerright":
		c = LowerRight
	case "ul", "upperleft":
		c = UpperLeft
	case "ur", "upperright":
		c = UpperRight
	default:
		return c, errors.Errorf("pdfcpu: unknown origin (ll, lr, ul, ur): %s", s)
	}
	return c, nil
}

func ParseAnchor(s string) (Anchor, error) {
	var a Anchor
	switch strings.ToLower(s) {
	case "tl", "topleft":
		a = TopLeft
	case "tc", "topcenter":
		a = TopCenter
	case "tr", "topright":
		a = TopRight
	case "l", "left":
		a = Left
	case "c", "center":
		a = Center
	case "r", "right":
		a = Right
	case "bl", "bottomleft":
		a = BottomLeft
	case "bc", "bottomcenter":
		a = BottomCenter
	case "br", "bottomright":
		a = BottomRight
	default:
		return a, errors.Errorf("pdfcpu: unknown anchor: %s", s)
	}
	return a, nil
}

func ParseRegionOrientation(s string) (Orientation, error) {
	var o Orientation
	switch strings.ToLower(s) {
	case "h", "hor", "horizontal":
		o = Horizontal
	case "v", "vert", "vertical":
		o = Vertical
	default:
		return o, errors.Errorf("pdfcpu: unknown region orientation (hor, vert): %s", s)
	}
	return o, nil
}

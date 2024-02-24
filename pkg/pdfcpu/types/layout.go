/*
Copyright 2022 The pdfcpu Authors.

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

package types

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

// HAlignment represents the horizontal alignment of text.
type HAlignment int

// These are the options for horizontal aligned text.
const (
	AlignLeft HAlignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// VAlignment represents the vertical alignment of text.
type VAlignment int

// These are the options for vertical aligned text.
const (
	AlignBaseline VAlignment = iota
	AlignTop
	AlignMiddle
	AlignBottom
)

// LineJoinStyle represents the shape to be used at the corners of paths that are stroked (see 8.4.3.4)
type LineJoinStyle int

// Render mode
const (
	LJMiter LineJoinStyle = iota
	LJRound
	LJBevel
)

func ParseHorAlignment(s string) (HAlignment, error) {
	var a HAlignment
	switch strings.ToLower(s) {
	case "l", "left":
		a = AlignLeft
	case "r", "right":
		a = AlignRight
	case "c", "center":
		a = AlignCenter
	case "j", "justify":
		a = AlignJustify
	default:
		return a, errors.Errorf("pdfcpu: unknown textfield alignment (left, center, right, justify): %s", s)
	}
	return a, nil
}

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

func ParsePositionAnchor(s string) (Anchor, error) {
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

func AnchorPosition(a Anchor, r *Rectangle, w, h float64) (x float64, y float64) {
	switch a {
	case TopLeft:
		x, y = 0, r.Height()-h
	case TopCenter:
		x, y = r.Width()/2-w/2, r.Height()-h
	case TopRight:
		x, y = r.Width()-w, r.Height()-h
	case Left:
		x, y = 0, r.Height()/2-h/2
	case Center:
		x, y = r.Width()/2-w/2, r.Height()/2-h/2
	case Right:
		x, y = r.Width()-w, r.Height()/2-h/2
	case BottomLeft:
		x, y = 0, 0
	case BottomCenter:
		x, y = r.Width()/2-w/2, 0
	case BottomRight:
		x, y = r.Width()-w, 0
	}
	return
}

// TODO Refactor because of orientation in nup.go
type Orientation int

const (
	Horizontal Orientation = iota
	Vertical
)

// RelPosition represents the relative position of a text field's label.
type RelPosition int

// These are the options for relative label positions.
const (
	RelPosLeft RelPosition = iota
	RelPosRight
	RelPosTop
	RelPosBottom
)

func ParseRelPosition(s string) (RelPosition, error) {
	var p RelPosition
	switch strings.ToLower(s) {
	case "l", "left":
		p = RelPosLeft
	case "r", "right":
		p = RelPosRight
	case "t", "top":
		p = RelPosTop
	case "b", "bottom":
		p = RelPosBottom
	default:
		return p, errors.Errorf("pdfcpu: unknown textfield alignment (left, right, top, bottom): %s", s)
	}
	return p, nil
}

// NormalizeCoord transfers P(x,y) from pdfcpu user space into PDF user space,
// which uses a coordinate system with origin in the lower left corner of r.
//
// pdfcpu user space coordinate systems have the origin in one of four corners of r:
//
// LowerLeft corner (default = PDF user space)
//
//	x extends to the right,
//	y extends upward
//
// LowerRight corner:
//
//	x extends to the left,
//	y extends upward
//
// UpperLeft corner:
//
//	x extends to the right,
//	y extends downward
//
// UpperRight corner:
//
//	x extends to the left,
//	y extends downward
func NormalizeCoord(x, y float64, r *Rectangle, origin Corner, absolute bool) (float64, float64) {
	switch origin {
	case UpperLeft:
		if y >= 0 {
			y = r.Height() - y
			if y < 0 {
				y = 0
			}
		}
	case LowerRight:
		if x >= 0 {
			x = r.Width() - x
			if x < 0 {
				x = 0
			}
		}
	case UpperRight:
		if x >= 0 {
			x = r.Width() - x
			if x < 0 {
				x = 0
			}
		}
		if y >= 0 {
			y = r.Height() - y
			if y < 0 {
				y = 0
			}
		}
	}
	if absolute {
		if x >= 0 {
			x += r.LL.X
		}
		if y >= 0 {
			y += r.LL.Y
		}
	}
	return x, y
}

// Normalize offset transfers x and y into offsets in the PDF user space.
func NormalizeOffset(x, y float64, origin Corner) (float64, float64) {
	switch origin {
	case UpperLeft:
		y = -y
	case LowerRight:
		x = -x
	case UpperRight:
		x = -x
		y = -y
	}
	return x, y
}

func BestFitRectIntoRect(rSrc, rDest *Rectangle, enforceOrient, scaleUp bool) (w, h, dx, dy, rot float64) {
	if !scaleUp && rSrc.FitsWithin(rDest) {
		// Translate rSrc into center of rDest without scaling.
		w = rSrc.Width()
		h = rSrc.Height()
		dx = rDest.Width()/2 - rSrc.Width()/2
		dy = rDest.Height()/2 - rSrc.Height()/2
		return
	}

	if rSrc.Landscape() {
		if rDest.Landscape() {
			if rSrc.AspectRatio() > rDest.AspectRatio() {
				w = rDest.Width()
				h = rSrc.ScaledHeight(w)
				dy = (rDest.Height() - h) / 2
			} else {
				h = rDest.Height()
				w = rSrc.ScaledWidth(h)
				dx = (rDest.Width() - w) / 2
			}
		} else {
			if enforceOrient {
				rot = 90
				if 1/rSrc.AspectRatio() < rDest.AspectRatio() {
					w = rDest.Height()
					h = rSrc.ScaledHeight(w)
					dx = (rDest.Width() - h) / 2
				} else {
					h = rDest.Width()
					w = rSrc.ScaledWidth(h)
					dy = (rDest.Height() - w) / 2
				}
				return
			}
			w = rDest.Width()
			h = rSrc.ScaledHeight(w)
			dy = (rDest.Height() - h) / 2
		}
		return
	}

	if rSrc.Portrait() {
		if rDest.Portrait() {
			if rSrc.AspectRatio() < rDest.AspectRatio() {
				h = rDest.Height()
				w = rSrc.ScaledWidth(h)
				dx = (rDest.Width() - w) / 2
			} else {
				w = rDest.Width()
				h = rSrc.ScaledHeight(w)
				dy = (rDest.Height() - h) / 2
			}
		} else {
			if enforceOrient {
				rot = 90
				if 1/rSrc.AspectRatio() > rDest.AspectRatio() {
					h = rDest.Width()
					w = rSrc.ScaledWidth(h)
					dy = (rDest.Height() - w) / 2
				} else {
					w = rDest.Height()
					h = rSrc.ScaledHeight(w)
					dx = (rDest.Width() - h) / 2
				}
				return
			}
			h = rDest.Height()
			w = rSrc.ScaledWidth(h)
			dx = (rDest.Width() - w) / 2
		}
		return
	}

	if rDest.Portrait() {
		w = rDest.Width()
		dy = rDest.Height()/2 - rSrc.ScaledHeight(w)/2
		h = w
	} else {
		h = rDest.Height()
		dx = rDest.Width()/2 - rSrc.ScaledWidth(h)/2
		w = h
	}

	return
}

func ParsePageFormat(v string) (*Dim, string, error) {

	// Optional: appended last letter L indicates landscape mode.
	// Optional: appended last letter P indicates portrait mode.
	// eg. A4L means A4 in landscape mode whereas A4 defaults to A4P
	// The default mode is defined implicitly via PaperSize dimensions.

	portrait := true

	if strings.HasSuffix(v, "L") {
		v = v[:len(v)-1]
		portrait = false
	} else {
		v = strings.TrimSuffix(v, "P")
	}

	d, ok := PaperSize[v]
	if !ok {
		return nil, v, errors.Errorf("pdfcpu: page format %s is unsupported.\n", v)
	}

	dim := Dim{d.Width, d.Height}
	if (d.Portrait() && !portrait) || (d.Landscape() && portrait) {
		dim.Width, dim.Height = dim.Height, dim.Width
	}

	return &dim, v, nil
}

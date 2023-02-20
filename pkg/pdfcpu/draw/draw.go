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

package draw

import (
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// RenderMode represents the text rendering mode (see 9.3.6)
type RenderMode int

// Render mode
const (
	RMFill RenderMode = iota
	RMStroke
	RMFillAndStroke
)

// SetLineJoinStyle sets the line join style for stroking operations.
func SetLineJoinStyle(w io.Writer, s types.LineJoinStyle) {
	fmt.Fprintf(w, "%d j ", s)
}

// SetLineWidth sets line width for stroking operations.
func SetLineWidth(w io.Writer, width float64) {
	fmt.Fprintf(w, "%.2f w ", width)
}

// SetStrokeColor sets the stroke color.
func SetStrokeColor(w io.Writer, c color.SimpleColor) {
	fmt.Fprintf(w, "%.2f %.2f %.2f RG ", c.R, c.G, c.B)
}

// SetFillColor sets the fill color.
func SetFillColor(w io.Writer, c color.SimpleColor) {
	fmt.Fprintf(w, "%.2f %.2f %.2f rg ", c.R, c.G, c.B)
}

// DrawLineSimple draws the path from P to Q.
func DrawLineSimple(w io.Writer, xp, yp, xq, yq float64) {
	fmt.Fprintf(w, "%.2f %.2f m %.2f %.2f l s ", xp, yp, xq, yq)
}

// DrawLine draws the path from P to Q using lineWidth, strokeColor and style.
func DrawLine(w io.Writer, xp, yp, xq, yq float64, lineWidth float64, strokeColor *color.SimpleColor, style *types.LineJoinStyle) {
	fmt.Fprintf(w, "q ")
	SetLineWidth(w, lineWidth)
	if strokeColor != nil {
		SetStrokeColor(w, *strokeColor)
	}
	if style != nil {
		SetLineJoinStyle(w, *style)
	}
	DrawLineSimple(w, xp, yp, xq, yq)
	fmt.Fprintf(w, "Q ")
}

// DrawRectSimple strokes a rectangular path for r.
func DrawRectSimple(w io.Writer, r *types.Rectangle) {
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f re s ", r.LL.X, r.LL.Y, r.Width(), r.Height())
}

// DrawRect strokes a rectangular path for r using lineWidth, strokeColor and style.
func DrawRect(w io.Writer, r *types.Rectangle, lineWidth float64, strokeColor *color.SimpleColor, style *types.LineJoinStyle) {
	fmt.Fprintf(w, "q ")
	SetLineWidth(w, lineWidth)
	if strokeColor != nil {
		SetStrokeColor(w, *strokeColor)
	}
	if style != nil {
		SetLineJoinStyle(w, *style)
	}
	DrawRectSimple(w, r)
	fmt.Fprintf(w, "Q ")
}

// FillRect fills a rectangular path for r using lineWidth, strokeCol, fillCol and style.
func FillRect(w io.Writer, r *types.Rectangle, lineWidth float64, strokeCol *color.SimpleColor, fillCol color.SimpleColor, style *types.LineJoinStyle) {
	fmt.Fprintf(w, "q ")
	SetLineWidth(w, lineWidth)
	c := fillCol
	if strokeCol != nil {
		c = *strokeCol
	}
	SetStrokeColor(w, c)
	SetFillColor(w, fillCol)
	if style != nil {
		SetLineJoinStyle(w, *style)
	}
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f re B ", r.LL.X, r.LL.Y, r.Width(), r.Height())
	fmt.Fprintf(w, "Q ")
}

// DrawCircle strokes a circle with optional filling.
func DrawCircle(w io.Writer, x, y, r float64, strokeCol color.SimpleColor, fillCol *color.SimpleColor) {
	f := .5523
	r1 := r - .1

	if fillCol != nil {
		fmt.Fprintf(w, "q %.2f %.2f %.2f rg 1 0 0 1 %.2f %.2f cm %.2f 0 m ", fillCol.R, fillCol.G, fillCol.B, x, y, r)
		fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
		fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
		fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
		fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
		fmt.Fprintf(w, "f Q ")
	}

	fmt.Fprintf(w, "q %.2f %.2f %.2f RG 1 0 0 1 %.2f %.2f cm %.2f 0 m ", strokeCol.R, strokeCol.G, strokeCol.B, x, y, r1)
	fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	fmt.Fprintf(w, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	fmt.Fprint(w, "s Q ")
}

// FillRectNoBorder fills a rectangular path for r using fillCol.
func FillRectNoBorder(w io.Writer, r *types.Rectangle, fillCol color.SimpleColor) {
	fmt.Fprintf(w, "q ")
	SetStrokeColor(w, fillCol)
	SetFillColor(w, fillCol)
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f re B ", r.LL.X, r.LL.Y, r.Width(), r.Height())
	fmt.Fprintf(w, "Q ")
}

// DrawGrid draws an x * y grid on r using strokeCol and fillCol.
func DrawGrid(w io.Writer, x, y int, r *types.Rectangle, strokeCol color.SimpleColor, fillCol *color.SimpleColor) {

	if fillCol != nil {
		FillRectNoBorder(w, r, *fillCol)
	}

	s := r.Width() / float64(x)
	for i := 0; i <= x; i++ {
		x := r.LL.X + float64(i)*s
		DrawLine(w, x, r.LL.Y, x, r.UR.Y, 0, &strokeCol, nil)
	}

	s = r.Height() / float64(y)
	for i := 0; i <= y; i++ {
		y := r.LL.Y + float64(i)*s
		DrawLine(w, r.LL.X, y, r.UR.X, y, 0, &strokeCol, nil)
	}
}

// DrawHairCross draw a haircross with origin x/y.
func DrawHairCross(w io.Writer, x, y float64, r *types.Rectangle) {
	x1, y1 := x, y
	if x == 0 {
		x1 = r.LL.X + r.Width()/2
	}
	if y == 0 {
		y1 = r.LL.Y + r.Height()/2
	}
	black := color.SimpleColor{}
	DrawLine(w, r.LL.X, y1, r.LL.X+r.Width(), y1, 0, &black, nil)  // Horizontal line
	DrawLine(w, x1, r.LL.Y, x1, r.LL.Y+r.Height(), 0, &black, nil) // Vertical line
}

// CLI drawing

const (
	HBar     = "\u2501"
	VBar     = "\u2502"
	CrossBar = "\u253f"
)

// HorSepLine renders a horizontal divider with optional column separators:
// ━━━━━━━━━━┿━━━━━━━━┿━━━━━━━━━━━━━━━━━━━━━━━━┿━━━━━━━┿━━━━━━━━┿━━━━━━━━
func HorSepLine(ii []int) string {
	s := ""
	for i, j := range ii {
		if i > 0 {
			s += CrossBar
		}
		s += strings.Repeat(HBar, j)
	}
	return s
}

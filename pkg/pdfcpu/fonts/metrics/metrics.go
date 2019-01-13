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

// Package metrics provides font metrics.
package metrics

import (
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu/fonts/metrics/standard"
	"github.com/hhrutter/pdfcpu/pkg/types"
)

// The PostScript names of 14 Type 1 fonts, known as the standard 14 fonts, are as follows:
//
// Times-Roman,
// Helvetica,
// Courier,
// Symbol,
// Times-Bold,
// Helvetica-Bold,
// Courier-Bold,
// ZapfDingbats,
// Times-Italic,
// Helvetica- Oblique,
// Courier-Oblique,
// Times-BoldItalic,
// Helvetica-BoldOblique,
// Courier-BoldOblique

// FontBoundingBox returns the font bounding box for a given font as specified in the corresponding AFM file.
func FontBoundingBox(fontName string) *types.Rectangle {
	return standardFonts[fontName].bbox
}

// CharWidth returns the character width for a char and font in glyph space units.
func CharWidth(fontName string, c int) int {

	f := standardFonts[fontName]

	w, found := f.charWidths[c]
	if !found {
		w = averageCharWidth(fontName)
	}

	return w
}

func averageCharWidth(fontName string) int {

	f := standardFonts[fontName]

	if f.averageWidth > 0 {
		return f.averageWidth
	}

	totalWidth := 0
	for _, w := range f.charWidths {
		totalWidth += w
	}

	f.averageWidth = totalWidth / len(f.charWidths)

	return f.averageWidth
}

func userSpaceUnits(glyphSpaceUnits float64, fontScalingFactor int) float64 {
	return float64(glyphSpaceUnits) / 1000 * float64(fontScalingFactor)
}

func fontScalingFactor(glyphSpaceUnits, userSpaceUnits float64) int {
	return int(userSpaceUnits / glyphSpaceUnits * 1000)
}

// TextWidth represents the width in user space units for a given text string, font name and font size.
func TextWidth(text, fontName string, fontSize int) float64 {
	var width float64
	for _, r := range text {
		w := CharWidth(fontName, int(r))
		//fmt.Printf("%d %c w=%d\n", r, r, w)
		width += userSpaceUnits(float64(w), fontSize)
	}
	return width
}

// FontSize returns the needed font size (aka. font scaling factor) in points
// for rendering a given text string using a given font name with a given user space width.
func FontSize(text, fontName string, width float64) int {
	var i int
	for _, r := range text {
		i += CharWidth(fontName, int(r))
	}
	return fontScalingFactor(float64(i), width)
}

// UserSpaceFontBBox returns the font box for given font name and font size in user space coordinates.
func UserSpaceFontBBox(fontName string, fontSize int) *types.Rectangle {
	fontBBox := FontBoundingBox(fontName)
	llx := userSpaceUnits(fontBBox.LL.X, fontSize)
	lly := userSpaceUnits(fontBBox.LL.Y, fontSize)
	urx := userSpaceUnits(fontBBox.UR.X, fontSize)
	ury := userSpaceUnits(fontBBox.UR.Y, fontSize)
	return types.NewRectangle(llx, lly, urx, ury)
}

// FontNames returns the list of supported font names.
func FontNames() []string {

	ss := make([]string, len(standardFonts))

	i := 0
	for k := range standardFonts {
		ss[i] = k
		i++
	}

	return ss
}

var standardFonts = map[string]struct {
	charWidths   map[int]int
	averageWidth int
	bbox         *types.Rectangle
}{
	"Helvetica":   {standard.FontWidthHelvetica, 0, types.NewRectangle(-166, -225, 1000, 931)},
	"Times-Roman": {standard.FontWidthTimesRoman, 0, types.NewRectangle(-168, -218, 1000, 898)},
	"Courier":     {map[int]int{}, 600, types.NewRectangle(-23, -250, 715, 805)},
}

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

// Package metrics provides font metrics for the PDF standard fonts.
package metrics

// The PostScript names of the 14 Type 1 fonts, aka the PDF core font set, are as follows:
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

// CoreFontCharWidth returns the character width for fontName and c in glyph space units.
func CoreFontCharWidth(fontName string, c int) int {
	var m map[int]string
	switch fontName {
	case "Symbol":
		m = SymbolGlyphMap
	case "ZapfDingbats":
		m = ZapfDingbatsGlyphMap
	default:
		m = WinAnsiGlyphMap
	}
	glyphName := m[c]
	fm := CoreFontMetrics[fontName]
	w, ok := fm.W[glyphName]
	if !ok {
		w = 1000 //m.W["bullet"]
	}
	return w
}

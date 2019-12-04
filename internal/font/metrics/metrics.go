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
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/types"
)

// TTFLight represents a TrueType font.
type TTFLight struct {
	PostscriptName     string            // name: NameID 6
	Protected          bool              // OS/2: fsType
	Ascent             int               // OS/2: sTypoAscender
	Descent            int               // OS/2: sTypoDescender
	CapHeight          int               // OS/2: sCapHeight
	FirstChar          uint16            // OS/2: fsFirstCharIndex
	LastChar           uint16            // OS/2: fsLastCharIndex
	LLx, LLy, URx, URy float64           // head: xMin, yMin, xMax, yMax (fontbox)
	ItalicAngle        float64           // post: italicAngle
	FixedPitch         bool              // post: isFixedPitch
	Bold               bool              // OS/2: usWeightClass == 7
	HorMetricsCount    int               // hhea: numOfLongHorMetrics
	GlyphCount         int               // maxp: numGlyphs
	GlyphWidths        []uint16          // hmtx: fd.HorMetricsCount.advanceWidth
	Chars              map[uint16]uint16 // cmap
}

func (fd TTFLight) String() string {
	return fmt.Sprintf(`
 PostscriptName = %s
      Protected = %t
         Ascent = %d
        Descent = %d
      CapHeight = %d
      FirstChar = %d
       LastChar = %d
FontBoundingBox = (%.2f, %.2f, %.2f, %.2f)
    ItalicAngle = %.2f
     FixedPitch = %t
           Bold = %t
HorMetricsCount = %d
     GlyphCount = %d`,
		fd.PostscriptName,
		fd.Protected,
		fd.Ascent,
		fd.Descent,
		fd.CapHeight,
		fd.FirstChar,
		fd.LastChar,
		fd.LLx, fd.LLy, fd.URx, fd.URy,
		fd.ItalicAngle,
		fd.FixedPitch,
		fd.Bold,
		fd.HorMetricsCount,
		fd.GlyphCount,
	)
}

// UserFontMetrics represents font metrics for user installed TrueType fonts.
var UserFontMetrics = map[string]TTFLight{}

func readGob(fileName string, fd *TTFLight) error {
	//fmt.Printf("reading gob from: %s\n", fileName)
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	return dec.Decode(fd)
}

// ReadFontFile reads in the font file bytes from gob
func ReadFontFile(fileName string) ([]byte, error) {
	dir, err := fontDir()
	if err != nil {
		return nil, err
	}
	fn := filepath.Join(dir, fileName+".gob")
	fmt.Printf("reading in fontFile from %s\n", fn)
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	ff := &struct{ FontFile []byte }{}
	err = dec.Decode(ff)
	return ff.FontFile, err
}

func isSupportedFontFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".gob")
}

func fontDir() (string, error) {
	// Get installed fonts from pdfcpu config dir in users home dir.
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, ".pdfcpu", "fonts"), nil
}

func init() {

	//fmt.Println("metrics.init")

	dir, err := fontDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed access to user home dir: %v\n", err)
		os.Exit(1)
	}

	//fmt.Printf("userDir = %s\n", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed access to fonts dir: %v\n", err)
		os.Exit(1)
	}

	for _, f := range files {
		if isSupportedFontFile(f.Name()) {
			// read gob
			ttf := TTFLight{}
			fn := filepath.Join(dir, f.Name())
			if err := readGob(fn, &ttf); err != nil {
				fmt.Fprintf(os.Stderr, "Failed access to %s:  %v\n", f.Name(), err)
				os.Exit(1)
			}
			fn = strings.TrimSuffix(f.Name(), path.Ext(f.Name()))
			//fmt.Printf("Loaded %s:\n%s", fn, ttf)
			UserFontMetrics[fn] = ttf
			//ttf = append(ttf, strings.TrimSuffix(f.Name(), path.Ext(f.Name())))
		}
	}
}

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
	if IsCoreFont(fontName) {
		return CoreFontMetrics[fontName].FBox
	}
	llx := UserFontMetrics[fontName].LLx
	lly := UserFontMetrics[fontName].LLy
	urx := UserFontMetrics[fontName].URx
	ury := UserFontMetrics[fontName].URy
	return types.NewRectangle(llx, lly, urx, ury)
}

// CharWidth returns the character width for a char and font in glyph space units.
func CharWidth(fontName string, c int) int {

	if IsCoreFont(fontName) {

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

	// UserFont

	ttf := UserFontMetrics[fontName]
	if WinAnsiGlyphMap[c] == ".notdef" {
		return int(ttf.GlyphWidths[0])
	}

	pos, ok := ttf.Chars[uint16(c)]
	if !ok {
		fmt.Printf("Character %s missing\n", WinAnsiGlyphMap[c])
		return int(ttf.GlyphWidths[0])
	}

	return int(ttf.GlyphWidths[pos])
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
	for i := 0; i < len(text); i++ {
		w := CharWidth(fontName, int(text[i]))
		width += userSpaceUnits(float64(w), fontSize)
	}
	return width
}

// FontSize returns the needed font size (aka. font scaling factor) in points
// for rendering a given text string using a given font name with a given user space width.
func FontSize(text, fontName string, width float64) int {
	var i int
	for j := 0; j < len(text); j++ {
		i += CharWidth(fontName, int(text[j]))
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

// IsCoreFont returns true for the 14 PDF standard fonts.
func IsCoreFont(fontName string) bool {
	_, ok := CoreFontMetrics[fontName]
	return ok
}

// CoreFontNames returns a list of the 14 PDF standard fonts.
func CoreFontNames() []string {
	ss := []string{}
	for fname := range CoreFontMetrics {
		ss = append(ss, fname)
	}
	return ss
}

// IsUserFont returns true for installed TrueType fonts.
func IsUserFont(fontName string) bool {
	_, ok := UserFontMetrics[fontName]
	return ok
}

// UserFontNames return a list of all installed TrueType fonts.
func UserFontNames() []string {
	ss := []string{}
	for fname := range UserFontMetrics {
		ss = append(ss, fname)
	}
	return ss
}

// IsSupportedFont returns true for core fonts or user installed fonts.
func IsSupportedFont(fontName string) bool {
	return IsCoreFont(fontName) || IsUserFont(fontName)
}

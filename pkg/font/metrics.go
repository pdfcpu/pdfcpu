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

package font

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/corefont/metrics"
	"github.com/pdfcpu/pdfcpu/pkg/types"
)

// TTFLight represents a TrueType font w/o font file.
type TTFLight struct {
	PostscriptName     string            // name: NameID 6
	Protected          bool              // OS/2: fsType
	UnitsPerEm         int               // head: unitsPerEm
	Ascent             int               // OS/2: sTypoAscender
	Descent            int               // OS/2: sTypoDescender
	CapHeight          int               // OS/2: sCapHeight
	FirstChar          uint16            // OS/2: fsFirstCharIndex
	LastChar           uint16            // OS/2: fsLastCharIndex
	UnicodeRange       [4]uint32         // OS/2: Unicode Character Range
	LLx, LLy, URx, URy float64           // head: xMin, yMin, xMax, yMax (fontbox)
	ItalicAngle        float64           // post: italicAngle
	FixedPitch         bool              // post: isFixedPitch
	Bold               bool              // OS/2: usWeightClass == 7
	HorMetricsCount    int               // hhea: numOfLongHorMetrics
	GlyphCount         int               // maxp: numGlyphs
	GlyphWidths        []int             // hmtx: fd.HorMetricsCount.advanceWidth
	Chars              map[uint32]uint32 // cmap
}

func (fd TTFLight) String() string {
	return fmt.Sprintf(`
 PostscriptName = %s
      Protected = %t
     UnitsPerEm = %d
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
		fd.UnitsPerEm,
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

func (fd TTFLight) supportsUnicodeBlock(bit int) bool {
	i := fd.UnicodeRange[bit/32]
	i >>= uint32(bit) % 32
	return i&1 > 0
}

func (fd TTFLight) IsCJK() bool {
	// 4E00-9FFF	CJK Unified Ideographs
	return fd.supportsUnicodeBlock(59)
}

// UserFontMetrics represents font metrics for user installed TrueType fonts.
var UserFontMetrics = map[string]TTFLight{}

func load(fileName string, fd *TTFLight) error {
	//fmt.Printf("reading gob from: %s\n", fileName)
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	return dec.Decode(fd)
}

// Read reads in the font file bytes from gob
func Read(fileName string) ([]byte, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	fn := filepath.Join(dir, fileName+".gob")
	//fmt.Printf("reading in fontFile from %s\n", fn)
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

// Dir returns the path where pdfcpu stores font info for embedding.
func Dir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	fontDir := filepath.Join(userConfigDir, "pdfcpu", "fonts")
	return fontDir, os.MkdirAll(fontDir, os.ModePerm)
}

func init() {
	//fmt.Println("metrics.init begin")
	dir, err := Dir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed access to font dir: %v\n", err)
		os.Exit(1)
	}
	//fmt.Printf("fontDir = %s\n", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed access to fonts dir: %v\n", err)
		os.Exit(1)
	}
	for _, f := range files {
		if isSupportedFontFile(f.Name()) {
			ttf := TTFLight{}
			fn := filepath.Join(dir, f.Name())
			if err := load(fn, &ttf); err != nil {
				fmt.Fprintf(os.Stderr, "Failed access to %s:  %v\n", f.Name(), err)
				os.Exit(1)
			}
			fn = strings.TrimSuffix(f.Name(), path.Ext(f.Name()))
			//fmt.Printf("loading %s.ttf...\n", fn)
			//fmt.Printf("Loaded %s:\n%s", fn, ttf)
			UserFontMetrics[fn] = ttf
		}
	}
	//fmt.Println()
	//fmt.Println("metrics.init end")
}

// BoundingBox returns the font bounding box for a given font as specified in the corresponding AFM file.
func BoundingBox(fontName string) *types.Rectangle {
	if IsCoreFont(fontName) {
		return metrics.CoreFontMetrics[fontName].FBox
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
		return metrics.CoreFontCharWidth(fontName, c)
	}
	ttf := UserFontMetrics[fontName]
	if metrics.WinAnsiGlyphMap[c] == ".notdef" {
		//fmt.Printf("Character %d missing in WinAnsiGlyphMap\n", uint16(c))
		return int(ttf.GlyphWidths[0])
	}
	pos, ok := ttf.Chars[uint32(c)]
	if !ok {
		//fmt.Printf("Character %s (%04x) missing\n", metrics.WinAnsiGlyphMap[c], uint16(c))
		return int(ttf.GlyphWidths[0])
	}
	return int(ttf.GlyphWidths[pos])
}

// UserSpaceUnits transforms glyphSpaceUnits into userspace units.
func UserSpaceUnits(glyphSpaceUnits float64, fontScalingFactor int) float64 {
	return glyphSpaceUnits / 1000 * float64(fontScalingFactor)
}

// GlyphSpaceUnits transforms userSpaceUnits into glyphspace Units.
func GlyphSpaceUnits(userSpaceUnits float64, fontScalingFactor int) float64 {
	return userSpaceUnits * 1000 / float64(fontScalingFactor)
}

func fontScalingFactor(glyphSpaceUnits, userSpaceUnits float64) int {
	return int(userSpaceUnits / glyphSpaceUnits * 1000)
}

// Descent returns fontname's descent in userspace units corresponding to fontSize.
func Descent(fontName string, fontSize int) float64 {
	fbb := BoundingBox(fontName)
	return UserSpaceUnits(-fbb.LL.Y, fontSize)
}

// Ascent returns fontname's ascent in userspace units corresponding to fontSize.
func Ascent(fontName string, fontSize int) float64 {
	fbb := BoundingBox(fontName)
	return UserSpaceUnits(fbb.Height()+fbb.LL.Y, fontSize)
}

// LineHeight returns fontname's line height in userspace units corresponding to fontSize.
func LineHeight(fontName string, fontSize int) float64 {
	fbb := BoundingBox(fontName)
	return UserSpaceUnits(fbb.Height(), fontSize)
}

// TextWidth represents the width in user space units for a given text string, font name and font size.
func TextWidth(text, fontName string, fontSize int) float64 {
	var width float64
	j := len(text)
	for i := 0; i < j; i++ {
		c := text[i]
		w := CharWidth(fontName, int(c))
		width += UserSpaceUnits(float64(w), fontSize)
	}
	//fmt.Printf("TextWidth:%.2f\n", width)
	return width
}

// Size returns the needed font size (aka. font scaling factor) in points
// for rendering a given text string using a given font name with a given user space width.
func Size(text, fontName string, width float64) int {
	var i int
	for j := 0; j < len(text); j++ {
		i += CharWidth(fontName, int(text[j]))
	}
	//fmt.Printf("FontSize:%d\n", fontScalingFactor(float64(i), width))
	return fontScalingFactor(float64(i), width)
}

// UserSpaceFontBBox returns the font box for given font name and font size in user space coordinates.
func UserSpaceFontBBox(fontName string, fontSize int) *types.Rectangle {
	fontBBox := BoundingBox(fontName)
	llx := UserSpaceUnits(fontBBox.LL.X, fontSize)
	lly := UserSpaceUnits(fontBBox.LL.Y, fontSize)
	urx := UserSpaceUnits(fontBBox.UR.X, fontSize)
	ury := UserSpaceUnits(fontBBox.UR.Y, fontSize)
	return types.NewRectangle(llx, lly, urx, ury)
}

// IsCoreFont returns true for the 14 PDF standard fonts.
func IsCoreFont(fontName string) bool {
	_, ok := metrics.CoreFontMetrics[fontName]
	return ok
}

// CoreFontNames returns a list of the 14 PDF standard fonts.
func CoreFontNames() []string {
	ss := []string{}
	for fontName := range metrics.CoreFontMetrics {
		ss = append(ss, fontName)
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
	for fName, ttf := range UserFontMetrics {
		s := fName + " (" + strconv.Itoa(ttf.GlyphCount) + " glyphs)"
		ss = append(ss, s)
	}
	return ss
}

// SupportedFont returns true for core fonts or user installed fonts.
func SupportedFont(fontName string) bool {
	return IsCoreFont(fontName) || IsUserFont(fontName)
}

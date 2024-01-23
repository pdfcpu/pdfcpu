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
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/pdfcpu/pdfcpu/internal/corefont/metrics"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/pkg/errors"
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
	Chars              map[uint32]uint16 // cmap: Unicode character to glyph index
	ToUnicode          map[uint16]uint32 // map glyph index to unicode character
	Planes             map[int]bool      // used Unicode planes
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
	 GlyphCount = %d
len(GlyphWidths) = %d`,
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
		len(fd.GlyphWidths),
	)
}

func (fd TTFLight) supportsUnicodeBlock(bit int) bool {
	i := fd.UnicodeRange[bit/32]
	i >>= uint32(bit) % 32
	return i&1 > 0
}

func (fd TTFLight) supportsUnicodeBlocks(bits []int) bool {
	// return true if we have support for the first or one of the following unicodeBlocks.
	ok := fd.supportsUnicodeBlock(bits[0])
	if ok || len(bits) == 1 {
		return ok
	}
	for i := range bits[1:] {
		if fd.supportsUnicodeBlock(i) {
			return true
		}
	}
	return false
}

func (fd TTFLight) unicodeRangeBits(id string) []int {
	// Map iso15924 script codes (=id) to corresponding unicode blocks.
	// Returns a slice of relevant unicodeRangeBits.
	//
	// This mapping is incomplete as we only cover unicode blocks of the most popular scripts.
	// Please go to https://github.com/pdfcpu/pdfcpu/issues/new/choose for an extension request.
	//
	//  0 Basic Latin						0000-007F
	//  1 Latin-1 Supplement				0080-00FF
	//  2 Latin Extended-A					0100-017F
	//  3 Latin Extended-B					0180-024F
	//  7 Greek								0370-03FF
	//  9 Cyrillic							0400-04FF
	// 10 Armenian							0530-058F
	// 11 Hebrew							0590-05FF
	// 13 Arabic							0600-06FF
	// 15 Devanagari						0900-097F
	// 16 Bengali							0980-09FF
	// 24 Thai								0E00-0E7F
	// 28 Hangul Jamo						1100-11FF
	// 48 CJK Symbols And Punctuation		3000-303F
	// 49 Hiragana							3040-309F
	// 50 Katakana							30A0-30FF
	// 52 Hangul Compatibility Jamo			3130-318F
	// 61 CJK Strokes						31C0-31EF
	// 54 Enclosed CJK Letters And Months	3200-32FF
	// 55 CJK Compatibility					3300-33FF
	// 59 CJK Unified Ideographs			4E00-9FFF
	// 56 Hangul Syllables					AC00-D7AF

	var a []int
	switch id {
	case "LATN": // Latin
		a = append(a, 0, 1, 2, 3)
	case "GREK": // Greek
		a = append(a, 7)
	case "CYRL": // Cyrillic
		a = append(a, 9)
	case "ARMN": // Armenian
		a = append(a, 10)
	case "HEBR": // Hebrew
		a = append(a, 11)
	case "ARAB": // Arabic
		a = append(a, 13)
	case "DEVA": // Devanagari
		a = append(a, 15)
	case "BENG": // Bengali
		a = append(a, 16)
	case "THAI": // Thai
		a = append(a, 24)
	case "HIRA": // Hiragana
		a = append(a, 49)
	case "KANA": // Katakana
		a = append(a, 50)
	case "JPAN": // Japanese
		a = append(a, 59, 49, 50)
	case "KORE", "HANG": // Korean, Hangul
		a = append(a, 59, 28, 52, 56)
	case "HANS", "HANT": // Han Simplified, Han Traditional
		a = append(a, 59)
	}

	return a
}

// SupportsScript returns true if ttf supports the unicodeblocks identified by iso15924 id.
func (fd TTFLight) SupportsScript(id string) (bool, error) {

	if len(id) != 4 {
		return false, errors.New("\"script\" must be a iso15924 code (length = 4")
	}

	bits := fd.unicodeRangeBits(id)
	if bits == nil {
		return false, errors.New("\"script\" must be one of: ARAB, ARMN, CYRL, GREK, HANG, HANS, HANT, HEBR, HIRA, LATN, JPAN, KANA, KORE, THAI")
	}

	return fd.supportsUnicodeBlocks(bits), nil
}

// UserFontDir is the location for installed TTF or OTF font files.
var UserFontDir string

// UserFontMetrics represents font metrics for TTF or OTF font files installed into UserFontDir.
var UserFontMetrics = map[string]TTFLight{}
var UserFontMetricsLock = &sync.RWMutex{}

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
	fn := filepath.Join(UserFontDir, fileName+".gob")
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

// LoadUserFonts loads any installed TTF or OTF font files.
func LoadUserFonts() error {
	//fmt.Printf("loading userFonts from %s\n", UserFontDir)
	files, err := os.ReadDir(UserFontDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if !isSupportedFontFile(f.Name()) {
			continue
		}
		ttf := TTFLight{}
		fn := filepath.Join(UserFontDir, f.Name())
		if err := load(fn, &ttf); err != nil {
			return err
		}
		fn = strings.TrimSuffix(f.Name(), path.Ext(f.Name()))
		//fmt.Printf("loading %s.ttf...\n", fn)
		//fmt.Printf("Loaded %s:\n%s", fn, ttf)
		UserFontMetricsLock.Lock()
		UserFontMetrics[fn] = ttf
		UserFontMetricsLock.Unlock()
	}
	return nil
}

// BoundingBox returns the font bounding box for a given font as specified in the corresponding AFM file.
func BoundingBox(fontName string) *types.Rectangle {
	if IsCoreFont(fontName) {
		return metrics.CoreFontMetrics[fontName].FBox
	}
	UserFontMetricsLock.RLock()
	defer UserFontMetricsLock.RUnlock()
	llx := UserFontMetrics[fontName].LLx
	lly := UserFontMetrics[fontName].LLy
	urx := UserFontMetrics[fontName].URx
	ury := UserFontMetrics[fontName].URy
	return types.NewRectangle(llx, lly, urx, ury)
}

// CharWidth returns the character width for a char and font in glyph space units.
func CharWidth(fontName string, r rune) int {
	if IsCoreFont(fontName) {
		return metrics.CoreFontCharWidth(fontName, int(r))
	}
	UserFontMetricsLock.RLock()
	defer UserFontMetricsLock.RUnlock()
	ttf, ok := UserFontMetrics[fontName]
	if !ok {
		fmt.Fprintf(os.Stderr, "pdfcpu: user font not loaded: %s\n", fontName)
		os.Exit(1)
	}

	pos, ok := ttf.Chars[uint32(r)]
	if !ok {
		pos = 0
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
	return int(math.Round(userSpaceUnits / glyphSpaceUnits * 1000))
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

func glyphSpaceWidth(text, fontName string) int {
	var w int
	if IsCoreFont(fontName) {
		for i := 0; i < len(text); i++ {
			c := text[i]
			w += CharWidth(fontName, rune(c))
		}
		return w
	}
	for _, r := range text {
		w += CharWidth(fontName, r)
	}
	return w
}

// TextWidth represents the width in user space units for a given text string, font name and font size.
func TextWidth(text, fontName string, fontSize int) float64 {
	w := glyphSpaceWidth(text, fontName)
	return UserSpaceUnits(float64(w), fontSize)
}

// Size returns the needed font size (aka. font scaling factor) in points
// for rendering a given text string using a given font name with a given user space width.
func Size(text, fontName string, width float64) int {
	w := glyphSpaceWidth(text, fontName)
	return fontScalingFactor(float64(w), width)
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

// IsCoreFont returns true for the 14 PDF standard Type 1 	fonts.
func IsCoreFont(fontName string) bool {
	_, ok := metrics.CoreFontMetrics[fontName]
	return ok
}

// CoreFontNames returns a list of the 14 PDF standard Type 1 fonts.
func CoreFontNames() []string {
	ss := []string{}
	for fontName := range metrics.CoreFontMetrics {
		ss = append(ss, fontName)
	}
	return ss
}

// IsUserFont returns true for installed TrueType fonts.
func IsUserFont(fontName string) bool {
	UserFontMetricsLock.RLock()
	defer UserFontMetricsLock.RUnlock()
	_, ok := UserFontMetrics[fontName]
	return ok
}

// UserFontNames return a list of all installed TrueType fonts.
func UserFontNames() []string {
	ss := []string{}
	UserFontMetricsLock.RLock()
	defer UserFontMetricsLock.RUnlock()
	for fontName := range UserFontMetrics {
		ss = append(ss, fontName)
	}
	return ss
}

// UserFontNamesVerbose return a list of all installed TrueType fonts including glyph count.
func UserFontNamesVerbose() []string {
	ss := []string{}
	UserFontMetricsLock.RLock()
	defer UserFontMetricsLock.RUnlock()
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

func (fd TTFLight) Gids() []int {
	gids := make([]int, 0, len(fd.Chars))
	for _, g := range fd.Chars {
		gids = append(gids, int(g))
	}
	return gids
}

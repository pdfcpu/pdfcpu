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
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/pdfcpu/pdfcpu/internal/corefont/metrics"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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

func validatePostScriptName(name string) error {
	if name == "" {
		return invalidFontData("missing PostScript name")
	}
	if len(name) > 63 {
		return invalidFontData("PostScript name length %d exceeds 63", len(name))
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c <= 0x20 || c >= 0x7f || strings.ContainsRune("()<>[]{}/%", rune(c)) {
			return invalidFontData("PostScript name contains invalid byte %#x at offset %d", c, i)
		}
	}
	return nil
}

func validateMetricCounts(fd TTFLight) error {
	if fd.UnitsPerEm < 16 || fd.UnitsPerEm > 16384 {
		return invalidFontData("units per em %d outside 16..16384", fd.UnitsPerEm)
	}
	if fd.GlyphCount <= 0 || fd.GlyphCount > int(^uint16(0)) {
		return invalidFontData("glyph count %d outside 1..%d", fd.GlyphCount, ^uint16(0))
	}
	if fd.HorMetricsCount <= 0 || fd.HorMetricsCount > fd.GlyphCount {
		return invalidFontData("horizontal metrics count %d outside 1..%d", fd.HorMetricsCount, fd.GlyphCount)
	}
	if len(fd.GlyphWidths) != fd.GlyphCount {
		return invalidFontData("glyph width count %d, expected %d", len(fd.GlyphWidths), fd.GlyphCount)
	}
	for gid, width := range fd.GlyphWidths {
		if width < 0 || width > int(^uint16(0)) {
			return invalidFontData("glyph ID %d width %d outside 0..%d", gid, width, ^uint16(0))
		}
	}
	return nil
}

func validateFontBoundingBox(fd TTFLight) error {
	for name, value := range map[string]float64{"LLx": fd.LLx, "LLy": fd.LLy, "URx": fd.URx, "URy": fd.URy} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return invalidFontData("font bounding box %s is not finite", name)
		}
	}
	if fd.LLx >= fd.URx || fd.LLy >= fd.URy {
		return invalidFontData("invalid font bounding box (%.2f, %.2f, %.2f, %.2f)", fd.LLx, fd.LLy, fd.URx, fd.URy)
	}
	return nil
}

func validUnicodeScalar(r uint32) bool {
	return r <= 0x10FFFF && (r < 0xD800 || r > 0xDFFF)
}

func validateUnicodeMaps(fd TTFLight) error {
	if fd.FirstChar > fd.LastChar {
		return invalidFontData("first character %d exceeds last character %d", fd.FirstChar, fd.LastChar)
	}
	if fd.Chars == nil {
		return invalidFontData("missing character map")
	}
	for char, gid := range fd.Chars {
		if !validUnicodeScalar(char) {
			return invalidFontData("character %#x is not a Unicode scalar value", char)
		}
		if int(gid) >= fd.GlyphCount {
			return invalidFontData("character U+%04X maps to glyph ID %d outside 0..%d", char, gid, fd.GlyphCount-1)
		}
	}
	if fd.ToUnicode == nil {
		return invalidFontData("missing ToUnicode map")
	}
	for gid, char := range fd.ToUnicode {
		if int(gid) >= fd.GlyphCount {
			return invalidFontData("ToUnicode glyph ID %d outside 0..%d", gid, fd.GlyphCount-1)
		}
		if !validUnicodeScalar(char) {
			return invalidFontData("ToUnicode glyph ID %d maps to non-scalar value %#x", gid, char)
		}
	}
	return nil
}

func validateUnicodePlanes(fd TTFLight) error {
	if fd.Planes == nil {
		return invalidFontData("missing Unicode planes map")
	}
	if len(fd.Planes) == 0 {
		return invalidFontData("empty Unicode planes map")
	}
	for plane, used := range fd.Planes {
		if plane < 0 || plane > 16 {
			return invalidFontData("Unicode plane %d outside 0..16", plane)
		}
		if !used {
			return invalidFontData("Unicode plane %d is marked unused", plane)
		}
	}
	for char := range fd.Chars {
		if !fd.Planes[int(char>>16)] {
			return invalidFontData("character U+%04X has no Unicode plane entry", char)
		}
	}
	for _, char := range fd.ToUnicode {
		if !fd.Planes[int(char>>16)] {
			return invalidFontData("ToUnicode value U+%04X has no Unicode plane entry", char)
		}
	}
	return nil
}

// ValidateTTFLight validates the semantic invariants required for font embedding.
func ValidateTTFLight(fd TTFLight) error {
	if err := validatePostScriptName(fd.PostscriptName); err != nil {
		return err
	}
	if err := validateMetricCounts(fd); err != nil {
		return err
	}
	if err := validateFontBoundingBox(fd); err != nil {
		return err
	}
	if err := validateUnicodeMaps(fd); err != nil {
		return err
	}
	return validateUnicodePlanes(fd)
}

// String returns the string value of fd.
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
	for _, bit := range bits[1:] {
		if fd.supportsUnicodeBlock(bit) {
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

var userFontMetrics = map[string]TTFLight{}
var userFontMetricsLock = &sync.RWMutex{}

// Lazy loading synchronization
var loadUserFontsOnce sync.Once
var loadUserFontsMutex sync.Mutex
var loadUserFontsErr error

func openInstalledGob(fileName string) (f *os.File, err error) {
	opened, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("open installed font %s: %w", fileName, err)
	}
	defer func() {
		if err != nil {
			if closeErr := opened.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close installed font %s: %w", fileName, closeErr))
			}
		}
	}()
	fi, err := opened.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat installed font %s: %w", fileName, err)
	}
	if fi.Size() > maxInstalledFontSize {
		return nil, invalidFontData("installed font %s: size %d exceeds limit %d", fileName, fi.Size(), maxInstalledFontSize)
	}
	return opened, nil
}

func load(fileName string, fd *TTFLight) (err error) {
	//fmt.Printf("reading gob from: %s\n", fileName)
	f, err := openInstalledGob(fileName)
	if err != nil {
		return fmt.Errorf("load font metrics %s: %w", fileName, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close font metrics %s: %w", fileName, closeErr))
		}
	}()
	dec := gob.NewDecoder(f)
	if err := dec.Decode(fd); err != nil {
		return fmt.Errorf("decode font metrics %s: %w: %w", fileName, ErrInvalidFontData, err)
	}
	if err := ValidateTTFLight(*fd); err != nil {
		return fmt.Errorf("validate font metrics %s: %w", fileName, err)
	}
	return nil
}

// Read reads the embedded font bytes from an installed font representation.
func Read(fileName string) (bb []byte, err error) {
	fileName, err = sanitize.Path(fileName)
	if err != nil {
		return nil, fmt.Errorf("sanitize font name: %w", err)
	}
	fn := filepath.Join(UserFontDir, fileName+".gob")
	f, err := openInstalledGob(fn)
	if err != nil {
		return nil, fmt.Errorf("read installed font %s: %w", fn, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close installed font %s: %w", fn, closeErr))
		}
	}()
	fd := ttf{}
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&fd); err != nil {
		return nil, fmt.Errorf("decode installed font %s: %w: %w", fn, ErrInvalidFontData, err)
	}
	if err := validateDecodedTTF(fd); err != nil {
		return nil, fmt.Errorf("validate installed font %s: %w", fn, err)
	}
	return fd.FontFile, nil
}

func isSupportedFontFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".gob")
}

// doLoadUserFonts performs the actual font loading logic.
// This is called exactly once by LoadUserFonts via sync.Once.
func doLoadUserFonts() error {
	if UserFontDir == "" {
		userFontMetricsLock.Lock()
		clear(userFontMetrics)
		userFontMetricsLock.Unlock()
		return nil
	}

	//fmt.Printf("*** loading userFonts from %s ***\n", UserFontDir)

	files, err := os.ReadDir(UserFontDir)
	if err != nil {
		return fmt.Errorf("read user font directory %s: %w", UserFontDir, err)
	}

	loadedMetrics := map[string]TTFLight{}
	for _, f := range files {
		if !isSupportedFontFile(f.Name()) {
			continue
		}
		ttf := TTFLight{}
		fn := filepath.Join(UserFontDir, f.Name())
		if err := load(fn, &ttf); err != nil {
			return fmt.Errorf("load user font %s: %w", f.Name(), err)
		}
		fn = strings.TrimSuffix(f.Name(), path.Ext(f.Name()))
		//fmt.Printf("loading %s.ttf...\n", fn)
		//fmt.Printf("Loaded %s:\n%s", fn, ttf)
		loadedMetrics[fn] = ttf
	}
	userFontMetricsLock.Lock()
	clear(userFontMetrics)
	for fn, ttf := range loadedMetrics {
		userFontMetrics[fn] = ttf
	}
	userFontMetricsLock.Unlock()
	return nil
}

// LoadUserFonts loads any installed TTF or OTF font files.
// This function is idempotent - it can be called multiple times safely.
// The actual loading happens exactly once, protected by sync.Once.
func LoadUserFonts() error {
	loadUserFontsMutex.Lock()
	defer loadUserFontsMutex.Unlock()
	loadUserFontsOnce.Do(func() {
		loadUserFontsErr = doLoadUserFonts()
	})
	return loadUserFontsErr
}

// ReloadUserFonts reloads installed user fonts after the font directory has changed.
func ReloadUserFonts() error {
	loadUserFontsMutex.Lock()
	defer loadUserFontsMutex.Unlock()
	loadUserFontsErr = doLoadUserFonts()
	loadUserFontsOnce.Do(func() {})
	return loadUserFontsErr
}

// BoundingBox returns the font bounding box for a given font.
func BoundingBox(fontName string) (*types.Rectangle, error) {
	if IsCoreFont(fontName) {
		return metrics.CoreFontMetrics[fontName].FBox, nil
	}
	ttf, ok, err := userFont(fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return nil, fmt.Errorf("font %s: metrics not found: %w", fontName, ErrUnknownFont)
	}
	return types.NewRectangle(ttf.LLx, ttf.LLy, ttf.URx, ttf.URy), nil
}

// CharWidth returns the character width for a char and font in glyph space units.
func CharWidth(fontName string, r rune) (int, error) {
	if IsCoreFont(fontName) {
		return metrics.CoreFontCharWidth(fontName, int(r)), nil
	}
	ttf, ok, err := userFont(fontName)
	if err != nil {
		return 0, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return 0, fmt.Errorf("font %s: metrics not found: %w", fontName, ErrUnknownFont)
	}
	pos, ok := ttf.Chars[uint32(r)]
	if !ok {
		pos = 0
	}
	if int(pos) >= len(ttf.GlyphWidths) {
		return 0, fmt.Errorf("font %s: character U+%04X maps to glyph ID %d outside width table", fontName, r, pos)
	}
	return ttf.GlyphWidths[pos], nil
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

// Descent returns fontName's descent in user-space units for fontSize.
func Descent(fontName string, fontSize int) (float64, error) {
	fbb, err := BoundingBox(fontName)
	if err != nil {
		return 0, err
	}
	return UserSpaceUnits(-fbb.LL.Y, fontSize), nil
}

// Ascent returns fontName's ascent in user-space units for fontSize.
func Ascent(fontName string, fontSize int) (float64, error) {
	fbb, err := BoundingBox(fontName)
	if err != nil {
		return 0, err
	}
	return UserSpaceUnits(fbb.Height()+fbb.LL.Y, fontSize), nil
}

// LineHeight returns fontName's line height in user-space units for fontSize.
func LineHeight(fontName string, fontSize int) (float64, error) {
	fbb, err := BoundingBox(fontName)
	if err != nil {
		return 0, err
	}
	return UserSpaceUnits(fbb.Height(), fontSize), nil
}

func glyphSpaceWidth(text, fontName string) (int, error) {
	var w int
	if IsCoreFont(fontName) {
		for i := 0; i < len(text); i++ {
			c := text[i]
			cw, err := CharWidth(fontName, rune(c))
			if err != nil {
				return 0, err
			}
			w += cw
		}
		return w, nil
	}
	for _, r := range text {
		cw, err := CharWidth(fontName, r)
		if err != nil {
			return 0, err
		}
		w += cw
	}
	return w, nil
}

// TextWidth returns the width in user-space units for text.
func TextWidth(text, fontName string, fontSize int) (float64, error) {
	w, err := glyphSpaceWidth(text, fontName)
	if err != nil {
		return 0, err
	}
	return UserSpaceUnits(float64(w), fontSize), nil
}

// Size returns the font size needed to fit text into width.
func Size(text, fontName string, width float64) (int, error) {
	w, err := glyphSpaceWidth(text, fontName)
	if err != nil {
		return 0, err
	}
	return fontScalingFactor(float64(w), width), nil
}

// SizeForLineHeight returns the needed font size in points
// for rendering using a given font name fitting into given line height lh.
func SizeForLineHeight(fontName string, lh float64) (int, error) {
	fbb, err := BoundingBox(fontName)
	if err != nil {
		return 0, err
	}
	return int(math.Round(lh / (fbb.Height() / 1000))), nil
}

// UserSpaceFontBBox returns the font box for given font name and font size in user space coordinates.
func UserSpaceFontBBox(fontName string, fontSize int) (*types.Rectangle, error) {
	fontBBox, err := BoundingBox(fontName)
	if err != nil {
		return nil, err
	}
	llx := UserSpaceUnits(fontBBox.LL.X, fontSize)
	lly := UserSpaceUnits(fontBBox.LL.Y, fontSize)
	urx := UserSpaceUnits(fontBBox.UR.X, fontSize)
	ury := UserSpaceUnits(fontBBox.UR.Y, fontSize)
	return types.NewRectangle(llx, lly, urx, ury), nil
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
func IsUserFont(fontName string) (bool, error) {
	if IsCoreFont(fontName) {
		return false, nil
	}
	if err := LoadUserFonts(); err != nil {
		return false, err
	}
	userFontMetricsLock.RLock()
	defer userFontMetricsLock.RUnlock()
	_, ok := userFontMetrics[fontName]
	return ok, nil
}

func cloneTTFLight(ttf TTFLight) TTFLight {
	ttf.GlyphWidths = slices.Clone(ttf.GlyphWidths)
	ttf.Chars = maps.Clone(ttf.Chars)
	ttf.ToUnicode = maps.Clone(ttf.ToUnicode)
	ttf.Planes = maps.Clone(ttf.Planes)
	return ttf
}

func userFont(fontName string) (TTFLight, bool, error) {
	if err := LoadUserFonts(); err != nil {
		return TTFLight{}, false, err
	}
	userFontMetricsLock.RLock()
	defer userFontMetricsLock.RUnlock()
	ttf, ok := userFontMetrics[fontName]
	return ttf, ok, nil
}

// UserFont returns a detached copy of the metrics for an installed TrueType font.
func UserFont(fontName string) (TTFLight, bool, error) {
	ttf, ok, err := userFont(fontName)
	if err != nil || !ok {
		return TTFLight{}, ok, err
	}
	return cloneTTFLight(ttf), true, nil
}

// UserFontNames returns a list of all installed TrueType fonts.
func UserFontNames() ([]string, error) {
	if err := LoadUserFonts(); err != nil {
		return nil, err
	}
	ss := []string{}
	userFontMetricsLock.RLock()
	defer userFontMetricsLock.RUnlock()
	for fontName := range userFontMetrics {
		ss = append(ss, fontName)
	}
	return ss, nil
}

// UserFontNamesVerbose returns installed TrueType fonts with glyph counts.
func UserFontNamesVerbose() ([]string, error) {
	if err := LoadUserFonts(); err != nil {
		return nil, err
	}
	ss := []string{}
	userFontMetricsLock.RLock()
	defer userFontMetricsLock.RUnlock()
	for fName, ttf := range userFontMetrics {
		s := fName + " (" + strconv.Itoa(ttf.GlyphCount) + " glyphs)"
		ss = append(ss, s)
	}
	return ss, nil
}

// SupportedFont returns true for core fonts or installed user fonts.
func SupportedFont(fontName string) (bool, error) {
	if IsCoreFont(fontName) {
		return true, nil
	}
	return IsUserFont(fontName)
}

// Gids returns glyph ids for s using fontName.
func (fd TTFLight) Gids() []int {
	gids := make([]int, 0, len(fd.Chars))
	for _, g := range fd.Chars {
		gids = append(gids, int(g))
	}
	return gids
}

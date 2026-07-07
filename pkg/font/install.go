/*
Copyright 2019 The pdfcpu Authors.

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

// Package font provides support for TrueType fonts.
package font

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/pdfcpu/pdfcpu/internal/fileutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
)

const (
	sfntVersionTrueType      = "\x00\x01\x00\x00"
	sfntVersionTrueTypeApple = "true"
	sfntVersionCFF           = "OTTO"
	ttfHeadMagicNumber       = 0x5F0F3CF5
	ttcTag                   = "ttcf"
	maxFontFileSize          = 256 << 20
	maxFontTableSize         = 128 << 20
	maxInstalledFontSize     = 512 << 20
	maxFontTableCount        = 1024
	maxFontCollectionFonts   = 256
)

var (
	// ErrMissingFontDir signals a missing destination font directory.
	ErrMissingFontDir = errors.New("missing font directory")

	// ErrMissingFontName signals a missing font name.
	ErrMissingFontName = errors.New("missing font name")

	// ErrMissingFontData signals missing font data.
	ErrMissingFontData = errors.New("missing font data")

	// ErrInvalidFontData signals malformed or inconsistent font data.
	ErrInvalidFontData = errors.New("invalid font data")

	// ErrUnsupportedFontFormat signals a valid but unsupported font format.
	ErrUnsupportedFontFormat = errors.New("unsupported font format")

	// ErrUnknownFont signals an unavailable font.
	ErrUnknownFont = errors.New("unknown font")

	// ErrDuplicatePostScriptName signals conflicting PostScript output names.
	ErrDuplicatePostScriptName = errors.New("duplicate PostScript name")
)

// InstallResult identifies an installed PostScript font name and optional collection member.
type InstallResult struct {
	PostScriptName string
	Member         int
}

// InstallReport describes successfully installed fonts and non-fatal cleanup failures.
// A non-nil warning means the fonts were published successfully but temporary or
// backup artifacts may remain.
type InstallReport struct {
	Fonts    []InstallResult
	Warnings []error
}

type ttf struct {
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
	FontFile           []byte
}

type ttfComparable struct {
	PostscriptName  string
	Protected       bool
	UnitsPerEm      int
	Ascent          int
	Descent         int
	CapHeight       int
	FirstChar       uint16
	LastChar        uint16
	UnicodeRange    [4]uint32
	LLx             float64
	LLy             float64
	URx             float64
	URy             float64
	ItalicAngle     float64
	FixedPitch      bool
	Bold            bool
	HorMetricsCount int
	GlyphCount      int
}

func comparableTTF(fd ttf) ttfComparable {
	return ttfComparable{
		PostscriptName:  fd.PostscriptName,
		Protected:       fd.Protected,
		UnitsPerEm:      fd.UnitsPerEm,
		Ascent:          fd.Ascent,
		Descent:         fd.Descent,
		CapHeight:       fd.CapHeight,
		FirstChar:       fd.FirstChar,
		LastChar:        fd.LastChar,
		UnicodeRange:    fd.UnicodeRange,
		LLx:             fd.LLx,
		LLy:             fd.LLy,
		URx:             fd.URx,
		URy:             fd.URy,
		ItalicAngle:     fd.ItalicAngle,
		FixedPitch:      fd.FixedPitch,
		Bold:            fd.Bold,
		HorMetricsCount: fd.HorMetricsCount,
		GlyphCount:      fd.GlyphCount,
	}
}

// String returns the string value of fd.
func (fd ttf) String() string {
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

func ttfEqual(fd1, fd2 ttf) bool {
	return comparableTTF(fd1) == comparableTTF(fd2) &&
		slices.Equal(fd1.GlyphWidths, fd2.GlyphWidths) &&
		maps.Equal(fd1.Chars, fd2.Chars) &&
		maps.Equal(fd1.ToUnicode, fd2.ToUnicode) &&
		maps.Equal(fd1.Planes, fd2.Planes) &&
		slices.Equal(fd1.FontFile, fd2.FontFile)
}

func (fd ttf) toPDFGlyphSpace(i int) int {
	return i * 1000 / fd.UnitsPerEm
}

func requireUnitsPerEm(fd *ttf, tag string) error {
	if fd.UnitsPerEm <= 0 {
		return invalidFontData("%s table: missing unitsPerEm", tag)
	}
	return nil
}

type myUint32 []uint32

// Len returns the number of uint32 values.
func (f myUint32) Len() int {
	return len(f)
}

// Less reports whether value i sorts before value j.
func (f myUint32) Less(i, j int) bool {
	return f[i] < f[j]
}

// Swap swaps values i and j.
func (f myUint32) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// PrintChars prints the font characters.
func (fd ttf) PrintChars() string {
	var min = uint16(0xFFFF)
	var max uint16
	var sb strings.Builder
	sb.WriteByte(0x0a)

	keys := make(myUint32, 0, len(fd.Chars))
	for k := range fd.Chars {
		keys = append(keys, k)
	}
	sort.Sort(keys)

	for _, c := range keys {
		g := fd.Chars[c]
		if g > max {
			max = g
		}
		if g < min {
			min = g
		}
		sb.WriteString(fmt.Sprintf("#%x -> #%x(%d)\n", c, g, g))
	}
	if log.CLIEnabled() {
		log.CLI.Printf("using glyphs[%08x,%08x] [%d,%d]\n", min, max, min, max)
		log.CLI.Printf("using glyphs #%x - #%x (%d-%d)\n", min, max, min, max)
	}
	return sb.String()
}

type table struct {
	chksum uint32
	off    uint32
	size   uint32
	padded uint32
	data   []byte
}

func (t table) uint16(off int) uint16 {
	return binary.BigEndian.Uint16(t.data[off:])
}

func (t table) int16(off int) int16 {
	return int16(t.uint16(off))
}

func (t table) uint32(off int) uint32 {
	return binary.BigEndian.Uint32(t.data[off:])
}

func (t table) fixed32(off int) float64 {
	return float64(int32(t.uint32(off))) / 65536.0
}

func (t table) logicalLength(tag string) (int, error) {
	if uint64(t.size) > uint64(len(t.data)) {
		return 0, invalidFontData("%s table: size %d exceeds data length %d", tag, t.size, len(t.data))
	}
	return int(t.size), nil
}

func (t table) requireSize(tag string, size int) error {
	length, err := t.logicalLength(tag)
	if err != nil {
		return err
	}
	if size < 0 || length < size {
		return invalidFontData("%s table: expected at least %d bytes, got %d", tag, size, length)
	}
	return nil
}

func (t table) slice(tag string, off, size int) ([]byte, error) {
	length, err := t.logicalLength(tag)
	if err != nil {
		return nil, err
	}
	if off < 0 || size < 0 || off > length-size {
		return nil, invalidFontData("%s table: range %d..%d exceeds length %d", tag, off, off+size, length)
	}
	return t.data[off : off+size], nil
}

func (t table) parseFontHeaderTable(fd *ttf) error {
	// table "head"
	if err := t.requireSize("head", 44); err != nil {
		return err
	}
	magic := t.uint32(12)
	if magic != ttfHeadMagicNumber {
		return invalidFontData("head table: wrong magic number %#x", magic)
	}

	unitsPerEm := t.uint16(18)
	if unitsPerEm == 0 {
		return invalidFontData("head table: unitsPerEm must be greater than zero")
	}
	//fmt.Printf("unitsPerEm: %d\n", unitsPerEm)
	fd.UnitsPerEm = int(unitsPerEm)

	llx := t.int16(36)
	//fmt.Printf("llx: %d\n", llx)
	fd.LLx = float64(fd.toPDFGlyphSpace(int(llx)))

	lly := t.int16(38)
	//fmt.Printf("lly: %d\n", lly)
	fd.LLy = float64(fd.toPDFGlyphSpace(int(lly)))

	urx := t.int16(40)
	//fmt.Printf("urx: %d\n", urx)
	fd.URx = float64(fd.toPDFGlyphSpace(int(urx)))

	ury := t.int16(42)
	//fmt.Printf("ury: %d\n", ury)
	fd.URy = float64(fd.toPDFGlyphSpace(int(ury)))

	return nil
}

func uint16ToBigEndianBytes(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

func uint32ToBigEndianBytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

func utf16BEToString(bb []byte) string {
	buf := make([]uint16, len(bb)/2)
	for i := range buf {
		buf[i] = binary.BigEndian.Uint16(bb[2*i:])
	}
	return string(utf16.Decode(buf))
}

func (t table) parsePostScriptTable(fd *ttf) error {
	// table "post"
	if err := t.requireSize("post", 16); err != nil {
		return err
	}
	italicAngle := t.fixed32(4)
	//fmt.Printf("italicAngle: %2.2f\n", italicAngle)
	fd.ItalicAngle = italicAngle

	isFixedPitch := t.uint32(12)
	//fmt.Printf("isFixedPitch: %t\n", isFixedPitch != 0)
	fd.FixedPitch = isFixedPitch != 0

	return nil
}

// func printUnicodeRange(off int, r uint32) {
// 	for i := 0; i < 32; i++ {
// 		if r&1 > 0 {
// 			fmt.Printf("bit %d: on\n", off+i)
// 		}
// 		r >>= 1
// 	}
// }

func (t table) parseWindowsMetricsTable(fd *ttf) error {
	// table "OS/2"
	if err := t.requireSize("OS/2", 72); err != nil {
		return err
	}
	if err := requireUnitsPerEm(fd, "OS/2"); err != nil {
		return err
	}
	version := t.uint16(0)
	if version >= 2 {
		if err := t.requireSize("OS/2", 90); err != nil {
			return err
		}
	}
	fsType := t.uint16(8)
	fd.Protected = fsType&2 > 0
	//fmt.Printf("protected: %t\n", fd.Protected)

	uniCodeRange1 := t.uint32(42)
	//fmt.Printf("uniCodeRange1: %032b\n", uniCodeRange1)
	fd.UnicodeRange[0] = uniCodeRange1

	uniCodeRange2 := t.uint32(46)
	//fmt.Printf("uniCodeRange2: %032b\n", uniCodeRange2)
	fd.UnicodeRange[1] = uniCodeRange2

	uniCodeRange3 := t.uint32(50)
	//fmt.Printf("uniCodeRange3: %032b\n", uniCodeRange3)
	fd.UnicodeRange[2] = uniCodeRange3

	uniCodeRange4 := t.uint32(54)
	//fmt.Printf("uniCodeRange4: %032b\n", uniCodeRange4)
	fd.UnicodeRange[3] = uniCodeRange4

	// printUnicodeRange(0, uniCodeRange1)
	// printUnicodeRange(32, uniCodeRange2)
	// printUnicodeRange(64, uniCodeRange3)
	// printUnicodeRange(96, uniCodeRange4)

	sTypoAscender := t.int16(68)
	fd.Ascent = fd.toPDFGlyphSpace(int(sTypoAscender))

	sTypoDescender := t.int16(70)
	fd.Descent = fd.toPDFGlyphSpace(int(sTypoDescender))

	// sCapHeight: This field was defined in version 2 of the OS/2 table.
	// sCapHeight = int16(0)
	if version >= 2 {
		sCapHeight := t.int16(88)
		fd.CapHeight = fd.toPDFGlyphSpace(int(sCapHeight))
	} else {
		// TODO the value may be set equal to the top of the unscaled and unhinted glyph bounding box
		// of the glyph encoded at U+0048 (LATIN CAPITAL LETTER H).
		fd.CapHeight = fd.Ascent
	}

	fsSelection := t.uint16(62)
	fd.Bold = fsSelection&0x40 > 0

	fsFirstCharIndex := t.uint16(64)
	fd.FirstChar = fsFirstCharIndex

	fsLastCharIndex := t.uint16(66)
	fd.LastChar = fsLastCharIndex

	return nil
}

func (t table) parseNamingTable(fd *ttf) error {
	// table "name"
	if err := t.requireSize("name", 6); err != nil {
		return err
	}
	count := int(t.uint16(2))
	stringOffset := int(t.uint16(4))
	baseOff := 6
	if err := t.requireSize("name", baseOff+count*12); err != nil {
		return fmt.Errorf("name records: %w", err)
	}
	for i := range count {
		recOff := baseOff + i*12
		pf := t.uint16(recOff)
		enc := t.uint16(recOff + 2)
		lang := t.uint16(recOff + 4)
		nameID := t.uint16(recOff + 6)
		length := int(t.uint16(recOff + 8))
		off := int(t.uint16(recOff + 10))
		if nameID != 6 {
			continue
		}
		s, err := t.slice("name", stringOffset+off, length)
		if err != nil {
			return fmt.Errorf("PostScript name record %d: %w", i, err)
		}
		if pf == 3 && enc == 1 && lang == 0x0409 {
			if len(s)%2 != 0 {
				return invalidFontData("name table: PostScript name record %d has odd UTF-16 length %d", i, len(s))
			}
			fd.PostscriptName = utf16BEToString(s)
			return nil
		}
		if pf == 1 && enc == 0 && lang == 0 {
			fd.PostscriptName = string(s)
			return nil
		}
	}

	return invalidFontData("name table: unable to identify PostScript name")
}

func (t table) parseHorizontalHeaderTable(fd *ttf) error {
	// table "hhea"
	if err := t.requireSize("hhea", 36); err != nil {
		return err
	}
	if err := requireUnitsPerEm(fd, "hhea"); err != nil {
		return err
	}
	ascent := t.int16(4)
	//fmt.Printf("ascent: %d\n", ascent)
	if fd.Ascent == 0 {
		fd.Ascent = fd.toPDFGlyphSpace(int(ascent))
	}

	descent := t.int16(6)
	//fmt.Printf("descent: %d\n", descent)
	if fd.Descent == 0 {
		fd.Descent = fd.toPDFGlyphSpace(int(descent))
	}

	//lineGap := t.int16(8)
	//fmt.Printf("lineGap: %d\n", lineGap)

	//advanceWidthMax := t.uint16(10)
	//fmt.Printf("advanceWidthMax: %d\n", advanceWidthMax)

	//minLeftSideBearing := t.int16(12)
	//fmt.Printf("minLeftSideBearing: %d\n", minLeftSideBearing)

	//minRightSideBearing := t.int16(14)
	//fmt.Printf("minRightSideBearing: %d\n", minRightSideBearing)

	//xMaxExtent := t.int16(16)
	//fmt.Printf("xMaxExtent: %d\n", xMaxExtent)

	numOfLongHorMetrics := t.uint16(34)
	if numOfLongHorMetrics == 0 {
		return invalidFontData("hhea table: numOfLongHorMetrics must be greater than zero")
	}
	//fmt.Printf("numOfLongHorMetrics: %d\n", numOfLongHorMetrics)
	fd.HorMetricsCount = int(numOfLongHorMetrics)

	return nil
}

func (t table) parseMaximumProfile(fd *ttf) error {
	// table "maxp"
	if err := t.requireSize("maxp", 6); err != nil {
		return err
	}
	numGlyphs := t.uint16(4)
	if numGlyphs == 0 {
		return invalidFontData("maxp table: numGlyphs must be greater than zero")
	}
	fd.GlyphCount = int(numGlyphs)
	return nil
}

func (t table) parseHorizontalMetricsTable(fd *ttf) error {
	// table "hmtx"
	if err := requireUnitsPerEm(fd, "hmtx"); err != nil {
		return err
	}
	if fd.GlyphCount <= 0 {
		return invalidFontData("hmtx table: missing glyph count")
	}
	if fd.HorMetricsCount <= 0 || fd.HorMetricsCount > fd.GlyphCount {
		return invalidFontData("hmtx table: horizontal metrics count %d outside 1..%d", fd.HorMetricsCount, fd.GlyphCount)
	}
	requiredSize := fd.HorMetricsCount*4 + (fd.GlyphCount-fd.HorMetricsCount)*2
	if err := t.requireSize("hmtx", requiredSize); err != nil {
		return err
	}
	fd.GlyphWidths = make([]int, fd.GlyphCount)

	for i := 0; i < int(fd.HorMetricsCount); i++ {
		fd.GlyphWidths[i] = fd.toPDFGlyphSpace(int(t.uint16(i * 4)))
	}

	for i := fd.HorMetricsCount; i < fd.GlyphCount; i++ {
		fd.GlyphWidths[i] = fd.GlyphWidths[fd.HorMetricsCount-1]
	}

	return nil
}

type cmapFormat4Layout struct {
	segCount int
	endOff   int
	startOff int
	deltaOff int
	rangeOff int
}

func prepareCMapFormat4(t table) (table, cmapFormat4Layout, error) {
	if err := t.requireSize("cmap format 4", 16); err != nil {
		return table{}, cmapFormat4Layout{}, err
	}
	if format := t.uint16(0); format != 4 {
		return table{}, cmapFormat4Layout{}, invalidFontData("cmap format 4: unexpected format %d", format)
	}
	declaredLength := int(t.uint16(2))
	if declaredLength < 16 {
		return table{}, cmapFormat4Layout{}, invalidFontData("cmap format 4: invalid length %d", declaredLength)
	}
	if err := t.requireSize("cmap format 4", declaredLength); err != nil {
		return table{}, cmapFormat4Layout{}, err
	}
	t.size = uint32(declaredLength)
	segCountX2 := t.uint16(6)
	if segCountX2 == 0 || segCountX2%2 != 0 {
		return table{}, cmapFormat4Layout{}, invalidFontData("cmap format 4: invalid segCountX2 %d", segCountX2)
	}
	segCount := int(segCountX2 / 2)
	endOff := 14
	startOff := endOff + 2*segCount + 2
	deltaOff := startOff + 2*segCount
	rangeOff := deltaOff + 2*segCount
	if err := t.requireSize("cmap format 4", rangeOff+2*segCount); err != nil {
		return table{}, cmapFormat4Layout{}, fmt.Errorf("segment arrays: %w", err)
	}
	return t, cmapFormat4Layout{segCount: segCount, endOff: endOff, startOff: startOff, deltaOff: deltaOff, rangeOff: rangeOff}, nil
}

func cmapFormat4Glyph(t table, layout cmapFormat4Layout, segment int, code uint32, index int) (uint16, error) {
	idDelta := t.uint16(layout.deltaOff + segment*2)
	idRangeOff := int(t.uint16(layout.rangeOff + segment*2))
	if idRangeOff == 0 {
		return uint16(code) + idDelta, nil
	}
	glyphOff := layout.rangeOff + segment*2 + idRangeOff + index*2
	if _, err := t.slice("cmap format 4", glyphOff, 2); err != nil {
		return 0, fmt.Errorf("segment %d glyph for code point %#x: %w", segment, code, err)
	}
	glyphID := t.uint16(glyphOff)
	if glyphID != 0 {
		glyphID += idDelta
	}
	return glyphID, nil
}

func parseCMapFormat4Segment(t table, fd *ttf, layout cmapFormat4Layout, segment int) error {
	sc := t.uint16(layout.startOff + segment*2)
	ec := t.uint16(layout.endOff + segment*2)
	startCode, endCode := uint32(sc), uint32(ec)
	if endCode < startCode {
		return invalidFontData("cmap format 4 segment %d: end code %d precedes start code %d", segment, endCode, startCode)
	}
	if fd.FirstChar == 0 {
		fd.FirstChar = sc
	}
	if fd.LastChar == 0 {
		fd.LastChar = ec
	}
	for code, index := startCode, 0; code <= endCode && code != 0xFFFF; code, index = code+1, index+1 {
		glyphID, err := cmapFormat4Glyph(t, layout, segment, code, index)
		if err != nil {
			return err
		}
		if glyphID == 0 {
			continue
		}
		if int(glyphID) >= fd.GlyphCount {
			return invalidFontData("cmap format 4: glyph ID %d for code point %#x exceeds glyph count %d", glyphID, code, fd.GlyphCount)
		}
		fd.Chars[code] = glyphID
		fd.ToUnicode[glyphID] = code
	}
	return nil
}

func (t table) parseCMapFormat4(fd *ttf) error {
	t, layout, err := prepareCMapFormat4(t)
	if err != nil {
		return err
	}
	fd.Planes[0] = true
	for segment := range layout.segCount {
		if err := parseCMapFormat4Segment(t, fd, layout, segment); err != nil {
			return err
		}
	}
	return nil
}

func prepareCMapFormat12(t table) (table, uint32, error) {
	if err := t.requireSize("cmap format 12", 16); err != nil {
		return table{}, 0, err
	}
	if format := t.uint16(0); format != 12 {
		return table{}, 0, invalidFontData("cmap format 12: unexpected format %d", format)
	}
	declaredLength := t.uint32(4)
	if declaredLength < 16 {
		return table{}, 0, invalidFontData("cmap format 12: invalid length %d", declaredLength)
	}
	if uint64(declaredLength) > uint64(^uint(0)>>1) {
		return table{}, 0, invalidFontData("cmap format 12: length %d exceeds int", declaredLength)
	}
	if err := t.requireSize("cmap format 12", int(declaredLength)); err != nil {
		return table{}, 0, err
	}
	t.size = declaredLength
	numGroups := t.uint32(12)
	requiredSize := uint64(16) + uint64(numGroups)*12
	if requiredSize > uint64(declaredLength) {
		return table{}, 0, invalidFontData("cmap format 12: %d groups require %d bytes, length is %d", numGroups, requiredSize, declaredLength)
	}
	return t, numGroups, nil
}

func cmapFormat12Group(t table, fd *ttf, group uint32) (uint32, uint32, uint32, error) {
	base := 16 + int(group)*12
	startCode := t.uint32(base)
	endCode := t.uint32(base + 4)
	if endCode < startCode {
		return 0, 0, 0, invalidFontData("cmap format 12 group %d: end code %#x precedes start code %#x", group, endCode, startCode)
	}
	if endCode > 0x10FFFF {
		return 0, 0, 0, invalidFontData("cmap format 12 group %d: end code %#x exceeds Unicode", group, endCode)
	}
	startGlyphID := t.uint32(base + 8)
	lastGlyphID := uint64(startGlyphID) + uint64(endCode-startCode)
	if lastGlyphID > uint64(^uint16(0)) || lastGlyphID >= uint64(fd.GlyphCount) {
		return 0, 0, 0, invalidFontData("cmap format 12 group %d: glyph range %d..%d exceeds glyph count %d", group, startGlyphID, lastGlyphID, fd.GlyphCount)
	}
	return startCode, endCode, startGlyphID, nil
}

func recordCMapPlanes(planes map[int]bool, startCode, endCode, prevCode uint32, first bool) {
	if first || startCode/0x10000 != prevCode/0x10000 {
		planes[int(startCode/0x10000)] = true
	}
	if startCode/0x10000 != endCode/0x10000 {
		planes[int(endCode/0x10000)] = true
	}
}

func addCMapFormat12Group(fd *ttf, startCode, endCode, startGlyphID uint32) {
	for code, glyphID := startCode, uint16(startGlyphID); code <= endCode; code, glyphID = code+1, glyphID+1 {
		fd.Chars[code] = glyphID
		fd.ToUnicode[glyphID] = code
	}
}

func (t table) parseCMapFormat12(fd *ttf) error {
	t, numGroups, err := prepareCMapFormat12(t)
	if err != nil {
		return err
	}
	var prevCode uint32
	for group := uint32(0); group < numGroups; group++ {
		startCode, endCode, startGlyphID, err := cmapFormat12Group(t, fd, group)
		if err != nil {
			return err
		}
		recordCMapPlanes(fd.Planes, startCode, endCode, prevCode, group == 0)
		prevCode = endCode
		addCMapFormat12Group(fd, startCode, endCode, startGlyphID)
	}
	return nil
}

func cmapSubtableLength(t table, record, start int, format uint16) (uint32, error) {
	fieldSize, lengthOff := 2, start+2
	if format >= 8 {
		fieldSize, lengthOff = 4, start+4
	}
	bb, err := t.slice("cmap", lengthOff, fieldSize)
	if err != nil {
		return 0, fmt.Errorf("encoding record %d format %d length: %w", record, format, err)
	}
	if fieldSize == 4 {
		return binary.BigEndian.Uint32(bb), nil
	}
	return uint32(binary.BigEndian.Uint16(bb)), nil
}

func cmapSubtable(t table, record int) (string, table, bool, error) {
	off := 4 + record*8
	platform := t.uint16(off)
	encoding := t.uint16(off + 2)
	subtableOff := t.uint32(off + 4)
	if uint64(subtableOff) > uint64(^uint(0)>>1) {
		return "", table{}, false, invalidFontData("cmap encoding record %d: offset %d exceeds int", record, subtableOff)
	}
	start := int(subtableOff)
	formatBytes, err := t.slice("cmap", start, 2)
	if err != nil {
		return "", table{}, false, fmt.Errorf("encoding record %d: %w", record, err)
	}
	format := binary.BigEndian.Uint16(formatBytes)
	if format == 14 {
		return "", table{}, true, nil
	}
	length, err := cmapSubtableLength(t, record, start, format)
	if err != nil {
		return "", table{}, false, err
	}
	if uint64(length) > uint64(^uint(0)>>1) {
		return "", table{}, false, invalidFontData("cmap encoding record %d: length %d exceeds int", record, length)
	}
	bb, err := t.slice("cmap", start, int(length))
	if err != nil {
		return "", table{}, false, fmt.Errorf("encoding record %d format %d: %w", record, format, err)
	}
	key := fmt.Sprintf("p%02d.e%02d.f%02d", platform, encoding, format)
	return key, table{off: subtableOff, size: length, padded: length, data: bb}, false, nil
}

func (t table) parseCharToGlyphMappingTable(fd *ttf) error {
	// table "cmap"
	if err := t.requireSize("cmap", 4); err != nil {
		return err
	}
	fd.Chars = map[uint32]uint16{}
	fd.ToUnicode = map[uint16]uint32{}
	fd.Planes = map[int]bool{}
	tableCount := int(t.uint16(2))
	if err := t.requireSize("cmap", 4+tableCount*8); err != nil {
		return fmt.Errorf("encoding records: %w", err)
	}
	subtables := map[string]table{}
	for record := range tableCount {
		key, subtable, skip, err := cmapSubtable(t, record)
		if err != nil {
			return err
		}
		if !skip {
			subtables[key] = subtable
		}
	}

	if t, ok := subtables["p00.e10.f12"]; ok {
		return t.parseCMapFormat12(fd)
	}
	if t, ok := subtables["p00.e04.f12"]; ok {
		return t.parseCMapFormat12(fd)
	}
	if t, ok := subtables["p03.e10.f12"]; ok {
		return t.parseCMapFormat12(fd)
	}
	if t, ok := subtables["p00.e03.f04"]; ok {
		return t.parseCMapFormat4(fd)
	}
	if t, ok := subtables["p03.e01.f04"]; ok {
		return t.parseCMapFormat4(fd)
	}

	return invalidFontData("cmap table: unsupported encoding or format")
}

func calcTableChecksum(tag string, b []byte) uint32 {
	sum := uint32(0)
	c := (len(b) + 3) / 4
	for i := range c {
		if tag == "head" && i == 2 {
			continue
		}
		sum += binary.BigEndian.Uint32(b[i*4:])
	}
	return sum
}

func getNext32BitAlignedLength(i uint32) (uint32, error) {
	n := (uint64(i) + 3) &^ 3
	if n > uint64(^uint32(0)) {
		return 0, fmt.Errorf("length %d exceeds uint32", i)
	}
	return uint32(n), nil
}

func readFontHeader(r io.ReaderAt, baseOff, size int64) ([]byte, int, error) {
	if size > maxFontFileSize {
		return nil, 0, invalidFontData("font size %d exceeds limit %d", size, maxFontFileSize)
	}
	if baseOff < 0 || baseOff > size-12 {
		return nil, 0, invalidFontData("header: invalid offset %d for size %d", baseOff, size)
	}
	header := make([]byte, 12)
	n, err := r.ReadAt(header, baseOff)
	if err != nil {
		return nil, 0, fmt.Errorf("header at offset %d: read: %w: %w", baseOff, ErrInvalidFontData, err)
	}
	if n != 12 {
		return nil, 0, invalidFontData("header at offset %d: expected 12 bytes, got %d", baseOff, n)
	}
	st := string(header[:4])
	if st == sfntVersionCFF {
		return nil, 0, fmt.Errorf("%w: OpenType CFF", ErrUnsupportedFontFormat)
	}
	if st != sfntVersionTrueType && st != sfntVersionTrueTypeApple {
		return nil, 0, invalidFontData("unrecognized font format %q", st)
	}
	tableCount := int(binary.BigEndian.Uint16(header[4:]))
	if tableCount == 0 || tableCount > maxFontTableCount {
		return nil, 0, invalidFontData("font table count %d outside 1..%d", tableCount, maxFontTableCount)
	}
	return header, tableCount, nil
}

func readTableDirectory(r io.ReaderAt, baseOff, size int64, c int) ([]byte, error) {
	dirSize := int64(c * 16)
	if baseOff+12 > size-dirSize {
		return nil, invalidFontData("table directory at offset %d: size %d exceeds font size %d", baseOff+12, dirSize, size)
	}
	b := make([]byte, c*16)
	n, err := r.ReadAt(b, baseOff+12)
	if err != nil {
		return nil, fmt.Errorf("table directory at offset %d: read: %w: %w", baseOff+12, ErrInvalidFontData, err)
	}
	if n != c*16 {
		return nil, invalidFontData("table directory at offset %d: expected %d bytes, got %d", baseOff+12, c*16, n)
	}
	return b, nil
}

func readFontTable(r io.ReaderAt, size int64, b []byte) (string, *table, error) {
	if len(b) != 16 {
		return "", nil, invalidFontData("table directory entry: expected 16 bytes, got %d", len(b))
	}
	tag := string(b[:4])
	chk := binary.BigEndian.Uint32(b[4:])
	o := binary.BigEndian.Uint32(b[8:])
	l := binary.BigEndian.Uint32(b[12:])
	if l > maxFontTableSize {
		return "", nil, invalidFontData("table %s: size %d exceeds limit %d", tag, l, maxFontTableSize)
	}
	ll, err := getNext32BitAlignedLength(l)
	if err != nil {
		return "", nil, fmt.Errorf("table %s: align size: %w: %w", tag, ErrInvalidFontData, err)
	}
	if int64(o) > size-int64(ll) {
		return "", nil, invalidFontData("table %s at offset %d: size %d exceeds font size %d", tag, o, ll, size)
	}
	data := make([]byte, ll)
	n, err := r.ReadAt(data, int64(o))
	if err != nil {
		return "", nil, fmt.Errorf("table %s at offset %d: read: %w: %w", tag, o, ErrInvalidFontData, err)
	}
	if n != int(ll) {
		return "", nil, invalidFontData("table %s at offset %d: expected %d bytes, got %d", tag, o, ll, n)
	}
	sum := calcTableChecksum(tag, data)
	if sum != chk {
		if log.CLIEnabled() {
			log.CLI.Printf("pdfcpu: fixing table<%s> checksum error; want:%d got:%d\n", tag, chk, sum)
		}
		chk = sum
	}
	return tag, &table{chksum: chk, off: o, size: l, padded: ll, data: data}, nil
}

func validateTableDirectory(b []byte, fontSize int64, tableCount int) error {
	if len(b) != tableCount*16 {
		return invalidFontData("table directory: expected %d bytes, got %d", tableCount*16, len(b))
	}
	tags := map[string]bool{}
	var totalSize uint64
	for i := range tableCount {
		entry := b[i*16 : i*16+16]
		tag := string(entry[:4])
		if tags[tag] {
			return invalidFontData("duplicate table tag %q", tag)
		}
		tags[tag] = true
		offset := binary.BigEndian.Uint32(entry[8:])
		size := binary.BigEndian.Uint32(entry[12:])
		if size > maxFontTableSize {
			return invalidFontData("table %s: size %d exceeds limit %d", tag, size, maxFontTableSize)
		}
		padded, err := getNext32BitAlignedLength(size)
		if err != nil {
			return fmt.Errorf("table %s: align size: %w: %w", tag, ErrInvalidFontData, err)
		}
		if uint64(offset)+uint64(padded) > uint64(fontSize) {
			return invalidFontData("table %s at offset %d: size %d exceeds font size %d", tag, offset, padded, fontSize)
		}
		totalSize += uint64(padded)
		if totalSize > maxFontFileSize {
			return invalidFontData("font table data size %d exceeds limit %d", totalSize, maxFontFileSize)
		}
	}
	return nil
}

func headerAndTables(r io.ReaderAt, baseOff, size int64) ([]byte, map[string]*table, error) {
	header, c, err := readFontHeader(r, baseOff, size)
	if err != nil {
		return nil, nil, err
	}
	b, err := readTableDirectory(r, baseOff, size, c)
	if err != nil {
		return nil, nil, err
	}
	if err := validateTableDirectory(b, size, c); err != nil {
		return nil, nil, err
	}
	tables := map[string]*table{}
	for j := range c {
		tag, t, err := readFontTable(r, size, b[j*16:j*16+16])
		if err != nil {
			return nil, nil, err
		}
		tables[tag] = t
	}
	return header, tables, nil
}

func parse(tags map[string]*table, tag string, fd *ttf) error {
	if fd == nil {
		return invalidFontData("%s table: missing font destination", tag)
	}
	t, found := tags[tag]
	if !found {
		// OS/2 is optional for True Type fonts.
		if tag == "OS/2" {
			return nil
		}
		return invalidFontData("missing %s table", tag)
	}
	if t == nil || t.data == nil {
		return invalidFontData("%s table: missing data", tag)
	}

	var err error

	switch tag {
	case "head":
		err = t.parseFontHeaderTable(fd)
	case "OS/2":
		err = t.parseWindowsMetricsTable(fd)
	case "post":
		err = t.parsePostScriptTable(fd)
	case "name":
		err = t.parseNamingTable(fd)
	case "hhea":
		err = t.parseHorizontalHeaderTable(fd)
	case "maxp":
		err = t.parseMaximumProfile(fd)
	case "hmtx":
		err = t.parseHorizontalMetricsTable(fd)
	case "cmap":
		err = t.parseCharToGlyphMappingTable(fd)
	default:
		return invalidFontData("unsupported table parser %q", tag)
	}

	return err
}

type gobPersistenceOperations struct {
	createTemp func(string, string) (*os.File, error)
	encode     func(*os.File, ttf) error
	sync       func(*os.File) error
	syncDir    func(string) error
	close      func(*os.File) error
	rename     func(string, string) error
	remove     func(string) error
	verify     func(string, *ttf) error
}

func defaultGobPersistenceOperations() gobPersistenceOperations {
	return gobPersistenceOperations{
		createTemp: os.CreateTemp,
		encode: func(f *os.File, fd ttf) error {
			return gob.NewEncoder(f).Encode(fd)
		},
		sync:    func(f *os.File) error { return f.Sync() },
		syncDir: fileutil.SyncDirectory,
		close:   func(f *os.File) error { return f.Close() },
		rename:  fileutil.ReplaceFile,
		remove:  fileutil.RemoveFile,
		verify:  readGob,
	}
}

func encodeGobFile(f *os.File, fileName string, fd ttf, ops gobPersistenceOperations) error {
	if err := ops.encode(f, fd); err != nil {
		return fmt.Errorf("encode %s: %w", fileName, err)
	}
	if err := ops.sync(f); err != nil {
		return fmt.Errorf("sync %s: %w", fileName, err)
	}
	return nil
}

func removeTemporaryFont(fileName string, remove func(string) error) error {
	if err := remove(fileName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove temporary font %s: %w", fileName, err)
	}
	return nil
}

func writeGobWithOperations(fileName string, fd ttf, ops gobPersistenceOperations) (err error) {
	f, err := ops.createTemp(filepath.Dir(fileName), "."+filepath.Base(fileName)+".tmp-")
	if err != nil {
		return fmt.Errorf("create temporary font for %s: %w", fileName, err)
	}
	tempName := f.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := ops.close(f); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close temporary font %s: %w", tempName, closeErr))
			}
		}
		if tempName != "" {
			err = errors.Join(err, removeTemporaryFont(tempName, ops.remove))
		}
	}()
	if err := encodeGobFile(f, tempName, fd, ops); err != nil {
		return err
	}
	closeErr := ops.close(f)
	closed = true
	if closeErr != nil {
		return fmt.Errorf("close temporary font %s: %w", tempName, closeErr)
	}
	fdNew := ttf{}
	if err := ops.verify(tempName, &fdNew); err != nil {
		return fmt.Errorf("verify temporary font: %w", err)
	}
	if !ttfEqual(fd, fdNew) {
		return errors.New("verify temporary font: representation mismatch")
	}
	if err := ops.rename(tempName, fileName); err != nil {
		return fmt.Errorf("publish installed font %s: %w", fileName, err)
	}
	tempName = ""
	if err := ops.syncDir(filepath.Dir(fileName)); err != nil {
		return fmt.Errorf("sync installed font directory %s: %w", filepath.Dir(fileName), err)
	}
	return nil
}

func writeGob(fileName string, fd ttf) error {
	return writeGobWithOperations(fileName, fd, defaultGobPersistenceOperations())
}

func readGob(fileName string, fd *ttf) (err error) {
	f, err := openInstalledGob(fileName)
	if err != nil {
		return fmt.Errorf("read installed representation %s: %w", fileName, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close %s: %w", fileName, closeErr))
		}
	}()
	dec := gob.NewDecoder(f)
	if err := dec.Decode(fd); err != nil {
		return fmt.Errorf("decode %s: %w: %w", fileName, ErrInvalidFontData, err)
	}
	if err := validateDecodedTTF(*fd); err != nil {
		return fmt.Errorf("validate %s: %w", fileName, err)
	}
	return nil
}

func (fd ttf) light() TTFLight {
	return TTFLight{
		PostscriptName:  fd.PostscriptName,
		Protected:       fd.Protected,
		UnitsPerEm:      fd.UnitsPerEm,
		Ascent:          fd.Ascent,
		Descent:         fd.Descent,
		CapHeight:       fd.CapHeight,
		FirstChar:       fd.FirstChar,
		LastChar:        fd.LastChar,
		UnicodeRange:    fd.UnicodeRange,
		LLx:             fd.LLx,
		LLy:             fd.LLy,
		URx:             fd.URx,
		URy:             fd.URy,
		ItalicAngle:     fd.ItalicAngle,
		FixedPitch:      fd.FixedPitch,
		Bold:            fd.Bold,
		HorMetricsCount: fd.HorMetricsCount,
		GlyphCount:      fd.GlyphCount,
		GlyphWidths:     fd.GlyphWidths,
		Chars:           fd.Chars,
		ToUnicode:       fd.ToUnicode,
		Planes:          fd.Planes,
	}
}

func validateDecodedFontFile(bb []byte) error {
	if len(bb) == 0 {
		return ErrMissingFontData
	}
	reader := bytes.NewReader(bb)
	_, tableCount, err := readFontHeader(reader, 0, int64(len(bb)))
	if err != nil {
		return fmt.Errorf("font file header: %w", err)
	}
	directory, err := readTableDirectory(reader, 0, int64(len(bb)), tableCount)
	if err != nil {
		return fmt.Errorf("font file table directory: %w", err)
	}
	if err := validateTableDirectory(directory, int64(len(bb)), tableCount); err != nil {
		return fmt.Errorf("font file table directory: %w", err)
	}
	return nil
}

func validateDecodedTTF(fd ttf) error {
	if err := ValidateTTFLight(fd.light()); err != nil {
		return fmt.Errorf("font metrics: %w", err)
	}
	if err := validateDecodedFontFile(fd.FontFile); err != nil {
		return fmt.Errorf("embedded font file: %w", err)
	}
	return nil
}

func validateInstallTarget(fontDir, fontName string) error {
	if strings.TrimSpace(fontDir) == "" {
		return ErrMissingFontDir
	}
	if strings.TrimSpace(fontName) == "" {
		return ErrMissingFontName
	}
	return nil
}

func validateInstallData(fontDir, fontName string, bb []byte) error {
	if err := validateInstallTarget(fontDir, fontName); err != nil {
		return err
	}
	if len(bb) == 0 {
		return ErrMissingFontData
	}
	return nil
}

func installTrueTypeRep(fontDir, fontName string, header []byte, tables map[string]*table, logInstall bool, reserve func(string) error) (string, error) {
	fd := ttf{}
	//fmt.Println(fontName)
	for _, v := range []string{"head", "OS/2", "post", "name", "hhea", "maxp", "hmtx", "cmap"} {
		if err := parse(tables, v, &fd); err != nil {
			return "", fmt.Errorf("parse %s table: %w", v, err)
		}
	}

	bb, err := createTTF(header, tables)
	if err != nil {
		return "", fmt.Errorf("create embedded font: %w", err)
	}
	fd.FontFile = bb

	fd.PostscriptName, err = sanitize.Path(fd.PostscriptName)
	if err != nil {
		return "", fmt.Errorf("sanitize PostScript name: %w: %w", ErrInvalidFontData, err)
	}
	if reserve != nil {
		if err := reserve(fd.PostscriptName); err != nil {
			return "", err
		}
	}
	if logInstall && log.CLIEnabled() {
		log.CLI.Println(fd.PostscriptName)
	}
	gobName := filepath.Join(fontDir, fd.PostscriptName+".gob")

	if err := writeGob(gobName, fd); err != nil {
		return "", fmt.Errorf("persist installed font: %w", err)
	}

	return fd.PostscriptName, nil
}

func readTrueTypeCollectionHeader(f *os.File, fn string) (int64, uint32, int64, error) {
	fi, err := f.Stat()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("font collection %s: stat: %w", fn, err)
	}
	if fi.Size() > maxFontFileSize {
		return 0, 0, 0, fmt.Errorf("font collection %s: %w", fn, invalidFontData("size %d exceeds limit %d", fi.Size(), maxFontFileSize))
	}
	b := make([]byte, 12)
	if _, err := io.ReadFull(f, b); err != nil {
		return 0, 0, 0, fmt.Errorf("font collection %s: read header: %w: %w", fn, ErrInvalidFontData, err)
	}
	if string(b[:4]) != ttcTag {
		return 0, 0, 0, fmt.Errorf("font collection %s: %w", fn, invalidFontData("invalid signature"))
	}
	version := binary.BigEndian.Uint32(b[4:])
	if version != 0x00010000 && version != 0x00020000 {
		return 0, 0, 0, fmt.Errorf("font collection %s: %w", fn, invalidFontData("unsupported TTC header version %#08x", version))
	}
	count := binary.BigEndian.Uint32(b[8:])
	offsetTableEnd := int64(12) + int64(count)*4
	if count == 0 || count > maxFontCollectionFonts || offsetTableEnd > fi.Size() {
		return 0, 0, 0, fmt.Errorf("font collection %s: %w", fn, invalidFontData("invalid font count %d", count))
	}
	return fi.Size(), count, offsetTableEnd, nil
}

func trueTypeCollectionMemberOffset(f *os.File, fn string, index uint32, size, offsetTableEnd int64) (int64, error) {
	var offsetBytes [4]byte
	if _, err := f.ReadAt(offsetBytes[:], int64(12)+int64(index)*4); err != nil {
		return 0, fmt.Errorf("font collection %s member %d: read offset: %w: %w", fn, index+1, ErrInvalidFontData, err)
	}
	off := int64(binary.BigEndian.Uint32(offsetBytes[:]))
	if off < offsetTableEnd || off > size-12 {
		return 0, fmt.Errorf("font collection %s member %d: %w", fn, index+1, invalidFontData("invalid offset %d", off))
	}
	return off, nil
}

type collectionInstallFileOperations struct {
	mkdirTemp     func(string, string) (string, error)
	lstat         func(string) (os.FileInfo, error)
	close         func(*os.File) error
	syncDir       func(string) error
	rename        func(string, string) error
	remove        func(string) error
	removeAll     func(string) error
	stageMembers  func(*os.File, string, string, int64, uint32, int64) ([]InstallResult, error)
	reportWarning func(error)
}

type committedCollectionFont struct {
	name        string
	hadOriginal bool
	committed   bool
}

func defaultCollectionInstallFileOperations() collectionInstallFileOperations {
	return collectionInstallFileOperations{
		mkdirTemp:    os.MkdirTemp,
		lstat:        os.Lstat,
		close:        func(f *os.File) error { return f.Close() },
		syncDir:      fileutil.SyncDirectory,
		rename:       fileutil.ReplaceFile,
		remove:       fileutil.RemoveFile,
		removeAll:    os.RemoveAll,
		stageMembers: installTrueTypeCollectionMembers,
		reportWarning: func(error) {
		},
	}
}

func syncCollectionDirectories(ops collectionInstallFileOperations, dirs ...string) error {
	seen := map[string]bool{}
	var err error
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if syncErr := ops.syncDir(dir); syncErr != nil {
			err = errors.Join(err, fmt.Errorf("sync directory %s: %w", dir, syncErr))
		}
	}
	return err
}

func rollbackCollectionFonts(fontDir, backupDir string, files []committedCollectionFont, ops collectionInstallFileOperations) error {
	var err error
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		target := filepath.Join(fontDir, file.name)
		if file.committed {
			if removeErr := ops.remove(target); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove committed font %s: %w", file.name, removeErr))
			}
		}
		if file.hadOriginal {
			if renameErr := ops.rename(filepath.Join(backupDir, file.name), target); renameErr != nil {
				err = errors.Join(err, fmt.Errorf("restore font %s: %w", file.name, renameErr))
			}
		}
	}
	err = errors.Join(err, syncCollectionDirectories(ops, fontDir, backupDir))
	if err != nil {
		return errors.Join(err, fmt.Errorf("font backup retained at %s", backupDir))
	}
	if removeErr := ops.removeAll(backupDir); removeErr != nil {
		return fmt.Errorf("remove font backup: %w", removeErr)
	}
	return syncCollectionDirectories(ops, fontDir)
}

func commitCollectionFonts(fontDir, stagingDir string, results []InstallResult, ops collectionInstallFileOperations) error {
	backupDir, err := ops.mkdirTemp(fontDir, ".pdfcpu-font-backup-")
	if err != nil {
		return fmt.Errorf("create font backup: %w", err)
	}
	files := make([]committedCollectionFont, 0, len(results))
	rollback := func(commitErr error) error {
		return errors.Join(commitErr, rollbackCollectionFonts(fontDir, backupDir, files, ops))
	}
	for _, result := range results {
		name := result.PostScriptName + ".gob"
		files = append(files, committedCollectionFont{name: name})
		file := &files[len(files)-1]
		target := filepath.Join(fontDir, name)
		if _, err := ops.lstat(target); err == nil {
			if err := ops.rename(target, filepath.Join(backupDir, name)); err != nil {
				return rollback(fmt.Errorf("backup font %s: %w", name, err))
			}
			file.hadOriginal = true
			if err := syncCollectionDirectories(ops, fontDir, backupDir); err != nil {
				return rollback(fmt.Errorf("backup font %s: %w", name, err))
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return rollback(fmt.Errorf("inspect font %s: %w", name, err))
		}
		if err := ops.rename(filepath.Join(stagingDir, name), target); err != nil {
			return rollback(fmt.Errorf("commit font %s: %w", name, err))
		}
		file.committed = true
		if err := syncCollectionDirectories(ops, stagingDir, fontDir); err != nil {
			return rollback(fmt.Errorf("commit font %s: %w", name, err))
		}
	}
	if err := ops.removeAll(backupDir); err != nil {
		ops.reportWarning(fmt.Errorf("remove font backup: %w", err))
	} else if err := syncCollectionDirectories(ops, fontDir); err != nil {
		ops.reportWarning(fmt.Errorf("sync after removing font backup: %w", err))
	}
	return nil
}

func reserveCollectionPostScriptName(members map[string]int, postScriptName string, member int) error {
	if firstMember, found := members[postScriptName]; found {
		return fmt.Errorf("%w %s: member %d conflicts with member %d", ErrDuplicatePostScriptName, postScriptName, member, firstMember)
	}
	members[postScriptName] = member
	return nil
}

func installTrueTypeCollectionMembers(f *os.File, fontDir, fn string, size int64, count uint32, offsetTableEnd int64) ([]InstallResult, error) {
	offsets := map[int64]bool{}
	members := map[string]int{}
	results := make([]InstallResult, 0, count)
	for i := uint32(0); i < count; i++ {
		off, err := trueTypeCollectionMemberOffset(f, fn, i, size, offsetTableEnd)
		if err != nil {
			return nil, err
		}
		if offsets[off] {
			return nil, fmt.Errorf("font collection %s member %d: %w", fn, i+1, invalidFontData("duplicate offset %d", off))
		}
		offsets[off] = true
		header, tables, err := headerAndTables(f, off, size)
		if err != nil {
			return nil, fmt.Errorf("font collection %s member %d: parse tables: %w", fn, i+1, err)
		}
		member := int(i) + 1
		reserve := func(postScriptName string) error {
			return reserveCollectionPostScriptName(members, postScriptName, member)
		}
		postScriptName, err := installTrueTypeRep(fontDir, fn, header, tables, false, reserve)
		if err != nil {
			return nil, fmt.Errorf("font collection %s member %d: install: %w", fn, member, err)
		}
		results = append(results, InstallResult{PostScriptName: postScriptName, Member: member})
	}
	return results, nil
}

// InstallTrueTypeCollection transactionally installs all fonts contained in a TrueType collection.
// No collection member remains installed when parsing, staging, or commit fails.
// Cleanup failures after publication are returned in InstallReport.Warnings.
func InstallTrueTypeCollection(fontDir, fn string) (InstallReport, error) {
	return InstallTrueTypeCollectionResults(fontDir, fn)
}

func installTrueTypeCollectionResults(fontDir, fn string, ops collectionInstallFileOperations) (report InstallReport, err error) {
	if err := validateInstallTarget(fontDir, fn); err != nil {
		return InstallReport{}, fmt.Errorf("install TrueType collection: %w", err)
	}
	f, err := os.Open(fn)
	if err != nil {
		return InstallReport{}, fmt.Errorf("font collection %s: open: %w", fn, err)
	}
	ops.reportWarning = func(warning error) {
		report.Warnings = append(report.Warnings, warning)
	}
	closed := false
	installed := false
	defer func() {
		if !closed {
			if closeErr := ops.close(f); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("font collection %s: close: %w", fn, closeErr))
			}
		}
	}()
	size, count, offsetTableEnd, err := readTrueTypeCollectionHeader(f, fn)
	if err != nil {
		return InstallReport{}, err
	}
	stagingDir, err := ops.mkdirTemp(fontDir, ".pdfcpu-ttc-install-")
	if err != nil {
		return InstallReport{}, fmt.Errorf("font collection %s: create staging directory: %w", fn, err)
	}
	defer func() {
		if cleanupErr := ops.removeAll(stagingDir); cleanupErr != nil {
			cleanupErr = fmt.Errorf("font collection %s: remove staging directory: %w", fn, cleanupErr)
			if installed {
				ops.reportWarning(cleanupErr)
			} else {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	results, err := ops.stageMembers(f, stagingDir, fn, size, count, offsetTableEnd)
	if err != nil {
		return InstallReport{}, err
	}
	closeErr := ops.close(f)
	closed = true
	if closeErr != nil {
		return InstallReport{}, fmt.Errorf("font collection %s: close before commit: %w", fn, closeErr)
	}
	if err := commitCollectionFonts(fontDir, stagingDir, results, ops); err != nil {
		return InstallReport{}, fmt.Errorf("font collection %s: commit: %w", fn, err)
	}
	installed = true
	report.Fonts = results
	if log.CLIEnabled() {
		for _, result := range results {
			log.CLI.Println(result.PostScriptName)
		}
	}
	return report, nil
}

// InstallTrueTypeCollectionResults transactionally installs a TrueType collection and reports every member.
// No collection member remains installed when parsing, staging, or commit fails.
// Cleanup failures after publication are returned in InstallReport.Warnings.
func InstallTrueTypeCollectionResults(fontDir, fn string) (InstallReport, error) {
	return installTrueTypeCollectionResults(fontDir, fn, defaultCollectionInstallFileOperations())
}

// InstallTrueTypeFont saves an internal representation of TrueType font fontName to the pdfcpu config dir.
func InstallTrueTypeFont(fontDir, fontName string) (InstallReport, error) {
	return InstallTrueTypeFontResult(fontDir, fontName)
}

type trueTypeFontInstallOperations struct {
	close           func(*os.File) error
	headerAndTables func(io.ReaderAt, int64, int64) ([]byte, map[string]*table, error)
	installRep      func(string, string, []byte, map[string]*table, bool, func(string) error) (string, error)
}

func defaultTrueTypeFontInstallOperations() trueTypeFontInstallOperations {
	return trueTypeFontInstallOperations{
		close:           func(f *os.File) error { return f.Close() },
		headerAndTables: headerAndTables,
		installRep:      installTrueTypeRep,
	}
}

// InstallTrueTypeFontResult saves a TrueType font and reports its installed PostScript name.
func InstallTrueTypeFontResult(fontDir, fontName string) (InstallReport, error) {
	return installTrueTypeFontResult(fontDir, fontName, defaultTrueTypeFontInstallOperations())
}

func installTrueTypeFontResult(fontDir, fontName string, ops trueTypeFontInstallOperations) (report InstallReport, err error) {
	if err := validateInstallTarget(fontDir, fontName); err != nil {
		return InstallReport{}, fmt.Errorf("install TrueType font: %w", err)
	}
	f, err := os.Open(fontName)
	if err != nil {
		return InstallReport{}, fmt.Errorf("font %s: open: %w", fontName, err)
	}
	closed := false
	defer func() {
		if !closed {
			if closeErr := ops.close(f); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("font %s: close: %w", fontName, closeErr))
			}
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return InstallReport{}, fmt.Errorf("font %s: stat: %w", fontName, err)
	}
	header, tables, err := ops.headerAndTables(f, 0, fi.Size())
	if err != nil {
		return InstallReport{}, fmt.Errorf("font %s: parse tables: %w", fontName, err)
	}
	closeErr := ops.close(f)
	closed = true
	if closeErr != nil {
		return InstallReport{}, fmt.Errorf("font %s: close before install: %w", fontName, closeErr)
	}
	postScriptName, err := ops.installRep(fontDir, fontName, header, tables, true, nil)
	if err != nil {
		return InstallReport{}, fmt.Errorf("font %s: install: %w", fontName, err)
	}
	return InstallReport{Fonts: []InstallResult{{PostScriptName: postScriptName}}}, nil
}

// InstallFontFromBytes saves an internal representation of TrueType font fontName to the pdfcpu config dir.
func InstallFontFromBytes(fontDir, fontName string, bb []byte) error {
	if err := validateInstallData(fontDir, fontName, bb); err != nil {
		return fmt.Errorf("install font from bytes: %w", err)
	}
	return installFontFromBytes(fontDir, fontName, bb, true)
}

// InstallFontFromBytesQuiet saves an internal representation of TrueType font fontName without logging.
func InstallFontFromBytesQuiet(fontDir, fontName string, bb []byte) error {
	if err := validateInstallData(fontDir, fontName, bb); err != nil {
		return fmt.Errorf("install font from bytes quietly: %w", err)
	}
	return installFontFromBytes(fontDir, fontName, bb, false)
}

func installFontFromBytes(fontDir, fontName string, bb []byte, logInstall bool) error {
	rd := bytes.NewReader(bb)
	header, tables, err := headerAndTables(rd, 0, int64(len(bb)))
	if err != nil {
		return fmt.Errorf("font %s: parse tables: %w", fontName, err)
	}
	if _, err := installTrueTypeRep(fontDir, fontName, header, tables, logInstall, nil); err != nil {
		return fmt.Errorf("font %s: install: %w", fontName, err)
	}
	return nil
}

func ttfTables(tableCount int, bb []byte) (map[string]*table, error) {
	if len(bb) < 12 {
		return nil, invalidFontData("font header: expected at least 12 bytes, got %d", len(bb))
	}
	if len(bb) > maxFontFileSize {
		return nil, invalidFontData("font size %d exceeds limit %d", len(bb), maxFontFileSize)
	}
	if tableCount <= 0 || tableCount > maxFontTableCount || tableCount > (len(bb)-12)/16 {
		return nil, invalidFontData("table count %d exceeds font size %d", tableCount, len(bb))
	}
	tables := map[string]*table{}
	b := bb[12:]
	var totalSize uint64
	for j := range tableCount {
		off := j * 16
		b1 := b[off : off+16]
		tag := string(b1[:4])
		if _, found := tables[tag]; found {
			return nil, invalidFontData("duplicate table tag %q", tag)
		}
		chksum := binary.BigEndian.Uint32(b1[4:])
		o := binary.BigEndian.Uint32(b1[8:])
		l := binary.BigEndian.Uint32(b1[12:])
		if l > maxFontTableSize {
			return nil, invalidFontData("table %s: size %d exceeds limit %d", tag, l, maxFontTableSize)
		}
		ll, err := getNext32BitAlignedLength(l)
		if err != nil {
			return nil, fmt.Errorf("table %s: align size: %w: %w", tag, err, ErrInvalidFontData)
		}
		if uint64(o)+uint64(ll) > uint64(len(bb)) {
			return nil, invalidFontData("table %s at offset %d: size %d exceeds font size %d", tag, o, ll, len(bb))
		}
		totalSize += uint64(ll)
		if totalSize > maxFontFileSize {
			return nil, invalidFontData("font table data size %d exceeds limit %d", totalSize, maxFontFileSize)
		}
		start, end := int(o), int(uint64(o)+uint64(ll))
		t := append([]byte(nil), bb[start:end]...)
		tables[tag] = &table{chksum: chksum, off: o, size: l, padded: ll, data: t}
	}
	return tables, nil
}

const (
	compoundArgWords       = 0x0001
	compoundScale          = 0x0008
	compoundMoreComponents = 0x0020
	compoundXYScale        = 0x0040
	compoundTwoByTwo       = 0x0080
	compoundInstructions   = 0x0100
	maxCompoundGlyphDepth  = 64
)

type subsetGlyphTables struct {
	loca          *table
	glyf          *table
	numGlyphs     int
	locaFormat    int
	locaEntrySize int
}

func tableUint16At(t *table, off int) (uint16, error) {
	if t == nil {
		return 0, invalidFontData("missing table")
	}
	if off < 0 || off > len(t.data)-2 {
		return 0, invalidFontData("offset %d: need 2 bytes, table length %d", off, len(t.data))
	}
	return binary.BigEndian.Uint16(t.data[off:]), nil
}

func tableUint32At(t *table, off int) (uint32, error) {
	if t == nil {
		return 0, invalidFontData("missing table")
	}
	if off < 0 || off > len(t.data)-4 {
		return 0, invalidFontData("offset %d: need 4 bytes, table length %d", off, len(t.data))
	}
	return binary.BigEndian.Uint32(t.data[off:]), nil
}

func requiredGlyphTable(tables map[string]*table, tag string, minSize int) (*table, error) {
	t, ok := tables[tag]
	if !ok || t == nil {
		return nil, invalidFontData("missing %s table", tag)
	}
	if uint64(t.size) < uint64(minSize) || len(t.data) < minSize {
		return nil, invalidFontData("%s table: expected at least %d bytes, got %d", tag, minSize, t.size)
	}
	return t, nil
}

func subsetGlyphTableInfo(tables map[string]*table) (subsetGlyphTables, error) {
	head, err := requiredGlyphTable(tables, "head", 52)
	if err != nil {
		return subsetGlyphTables{}, err
	}
	maxp, err := requiredGlyphTable(tables, "maxp", 6)
	if err != nil {
		return subsetGlyphTables{}, err
	}
	loca, err := requiredGlyphTable(tables, "loca", 2)
	if err != nil {
		return subsetGlyphTables{}, err
	}
	glyf, err := requiredGlyphTable(tables, "glyf", 0)
	if err != nil {
		return subsetGlyphTables{}, err
	}

	format, err := tableUint16At(head, 50)
	if err != nil {
		return subsetGlyphTables{}, fmt.Errorf("head table indexToLocFormat: %w", err)
	}
	if format > 1 {
		return subsetGlyphTables{}, invalidFontData("head table indexToLocFormat: expected 0 or 1, got %d", format)
	}
	numGlyphs, err := tableUint16At(maxp, 4)
	if err != nil {
		return subsetGlyphTables{}, fmt.Errorf("maxp table numGlyphs: %w", err)
	}
	if numGlyphs == 0 {
		return subsetGlyphTables{}, invalidFontData("maxp table numGlyphs: expected at least 1")
	}
	entrySize := 2
	if format == 1 {
		entrySize = 4
	}
	requiredLocaSize := (int(numGlyphs) + 1) * entrySize
	if uint64(loca.size) < uint64(requiredLocaSize) || len(loca.data) < requiredLocaSize {
		return subsetGlyphTables{}, invalidFontData("loca table: expected at least %d bytes for %d glyphs, got %d", requiredLocaSize, numGlyphs, loca.size)
	}
	if uint64(glyf.size) > uint64(len(glyf.data)) {
		return subsetGlyphTables{}, invalidFontData("glyf table: size %d exceeds data length %d", glyf.size, len(glyf.data))
	}
	return subsetGlyphTables{loca: loca, glyf: glyf, numGlyphs: int(numGlyphs), locaFormat: int(format), locaEntrySize: entrySize}, nil
}

func locaOffset(info subsetGlyphTables, index int) (int, error) {
	if index < 0 || index > info.numGlyphs {
		return 0, invalidFontData("loca index %d outside 0..%d", index, info.numGlyphs)
	}
	off := index * info.locaEntrySize
	if info.locaEntrySize == 2 {
		v, err := tableUint16At(info.loca, off)
		if err != nil {
			return 0, fmt.Errorf("loca index %d: %w", index, err)
		}
		return 2 * int(v), nil
	}
	v, err := tableUint32At(info.loca, off)
	if err != nil {
		return 0, fmt.Errorf("loca index %d: %w", index, err)
	}
	if uint64(v) > uint64(^uint(0)>>1) {
		return 0, invalidFontData("loca index %d: offset %d exceeds int", index, v)
	}
	return int(v), nil
}

func glyphRange(info subsetGlyphTables, gid int) (int, int, error) {
	if gid < 0 || gid >= info.numGlyphs {
		return 0, 0, invalidFontData("glyph ID %d outside 0..%d", gid, info.numGlyphs-1)
	}
	from, err := locaOffset(info, gid)
	if err != nil {
		return 0, 0, err
	}
	thru, err := locaOffset(info, gid+1)
	if err != nil {
		return 0, 0, err
	}
	if thru < from {
		return 0, 0, invalidFontData("glyph %d: descending offsets %d-%d", gid, from, thru)
	}
	if uint64(thru) > uint64(info.glyf.size) {
		return 0, 0, invalidFontData("glyph %d: offset %d exceeds glyf size %d", gid, thru, info.glyf.size)
	}
	return from, thru, nil
}

func glyphData(info subsetGlyphTables, gid int) ([]byte, error) {
	from, thru, err := glyphRange(info, gid)
	if err != nil {
		return nil, err
	}
	return info.glyf.data[from:thru], nil
}

func encodeGlyfOffset(off uint64, indexToLocFormat int) ([]byte, error) {
	switch indexToLocFormat {
	case 0:
		if off%2 != 0 {
			return nil, invalidFontData("short loca offset %d is not 2-byte aligned", off)
		}
		if off/2 > uint64(^uint16(0)) {
			return nil, invalidFontData("short loca offset %d exceeds uint16 encoding", off)
		}
		return uint16ToBigEndianBytes(uint16(off / 2)), nil
	case 1:
		if off > uint64(^uint32(0)) {
			return nil, invalidFontData("long loca offset %d exceeds uint32", off)
		}
		return uint32ToBigEndianBytes(uint32(off)), nil
	default:
		return nil, invalidFontData("loca format: expected 0 or 1, got %d", indexToLocFormat)
	}
}

func writeGlyfOffset(buf *bytes.Buffer, off uint64, indexToLocFormat int) error {
	bb, err := encodeGlyfOffset(off, indexToLocFormat)
	if err != nil {
		return err
	}
	if _, err := buf.Write(bb); err != nil {
		return fmt.Errorf("write loca offset %d: %w", off, err)
	}
	return nil
}

func nextGlyfOffset(off uint64, glyphLength, indexToLocFormat int) (uint64, error) {
	if glyphLength < 0 {
		return 0, invalidFontData("negative glyph length %d", glyphLength)
	}
	next := off + uint64(glyphLength)
	if next < off {
		return 0, invalidFontData("glyf offset %d plus length %d overflows uint64", off, glyphLength)
	}
	if _, err := encodeGlyfOffset(next, indexToLocFormat); err != nil {
		return 0, err
	}
	return next, nil
}

func pad(bb []byte) []byte {
	i := len(bb) % 4
	if i == 0 {
		return bb
	}
	for j := 0; j < 4-i; j++ {
		bb = append(bb, 0x00)
	}
	return bb
}

func rebuiltTableData(tag string, bb []byte) ([]byte, uint32, uint32, error) {
	if uint64(len(bb)) > uint64(^uint32(0)) {
		return nil, 0, 0, invalidFontData("%s table length %d exceeds uint32", tag, len(bb))
	}
	padding := (4 - len(bb)%4) % 4
	if len(bb) > int(^uint(0)>>1)-padding {
		return nil, 0, 0, invalidFontData("%s padded table length overflows int", tag)
	}
	paddedLength := len(bb) + padding
	if uint64(paddedLength) > uint64(^uint32(0)) {
		return nil, 0, 0, invalidFontData("%s padded table length %d exceeds uint32", tag, paddedLength)
	}
	data := append(bb, make([]byte, padding)...)
	return data, uint32(len(bb)), uint32(paddedLength), nil
}

func compoundTransformSize(flags uint16) (int, error) {
	size, count := 0, 0
	if flags&compoundScale != 0 {
		size, count = 2, count+1
	}
	if flags&compoundXYScale != 0 {
		size, count = 4, count+1
	}
	if flags&compoundTwoByTwo != 0 {
		size, count = 8, count+1
	}
	if count > 1 {
		return 0, invalidFontData("compound glyph: conflicting transform flags %#x", flags)
	}
	return size, nil
}

func compoundComponent(bb []byte, off int) (uint16, uint16, int, error) {
	if off < 0 || off > len(bb)-4 {
		return 0, 0, 0, invalidFontData("component at offset %d: need 4 bytes, glyph length %d", off, len(bb))
	}
	flags := binary.BigEndian.Uint16(bb[off:])
	gid := binary.BigEndian.Uint16(bb[off+2:])
	if flags&compoundInstructions != 0 && flags&compoundMoreComponents != 0 {
		return 0, 0, 0, invalidFontData("component at offset %d: instructions flag before final component", off)
	}
	argSize := 2
	if flags&compoundArgWords != 0 {
		argSize = 4
	}
	transformSize, err := compoundTransformSize(flags)
	if err != nil {
		return 0, 0, 0, err
	}
	next := off + 4 + argSize + transformSize
	if next > len(bb) {
		return 0, 0, 0, invalidFontData("component at offset %d: record length %d exceeds glyph length %d", off, next-off, len(bb))
	}
	return flags, gid, next, nil
}

func validateCompoundInstructions(bb []byte, off int, flags uint16) error {
	if flags&compoundInstructions == 0 {
		return nil
	}
	if off < 0 || off > len(bb)-2 {
		return invalidFontData("compound instructions at offset %d: missing length", off)
	}
	n := int(binary.BigEndian.Uint16(bb[off:]))
	if off+2+n > len(bb) {
		return invalidFontData("compound instructions at offset %d: length %d exceeds glyph length %d", off, n, len(bb))
	}
	return nil
}

func compoundGlyph(bb []byte) (bool, error) {
	if len(bb) == 0 {
		return false, nil
	}
	if len(bb) < 10 {
		return false, invalidFontData("glyph header: expected at least 10 bytes, got %d", len(bb))
	}
	return int16(binary.BigEndian.Uint16(bb)) < 0, nil
}

func resolveCompoundGlyph(gid int, bb []byte, usedGIDs map[uint16]bool, info subsetGlyphTables, depth int) error {
	if depth >= maxCompoundGlyphDepth {
		return invalidFontData("glyph %d: compound nesting exceeds %d", gid, maxCompoundGlyphDepth)
	}
	isCompound, err := compoundGlyph(bb)
	if err != nil {
		return fmt.Errorf("glyph %d: %w", gid, err)
	}
	if !isCompound {
		return invalidFontData("glyph %d: expected compound glyph", gid)
	}
	for off := 10; ; {
		flags, componentGID, next, err := compoundComponent(bb, off)
		if err != nil {
			return fmt.Errorf("glyph %d: %w", gid, err)
		}
		off = next
		if !usedGIDs[componentGID] {
			componentData, err := glyphData(info, int(componentGID))
			if err != nil {
				return fmt.Errorf("glyph %d component %d: %w", gid, componentGID, err)
			}
			usedGIDs[componentGID] = true
			componentCompound, err := compoundGlyph(componentData)
			if err != nil {
				return fmt.Errorf("glyph %d component %d: %w", gid, componentGID, err)
			}
			if componentCompound {
				if err := resolveCompoundGlyph(int(componentGID), componentData, usedGIDs, info, depth+1); err != nil {
					return err
				}
			}
		}
		if flags&compoundMoreComponents == 0 {
			if err := validateCompoundInstructions(bb, off, flags); err != nil {
				return fmt.Errorf("glyph %d: %w", gid, err)
			}
			return nil
		}
	}
}

func sortedUsedGlyphIDs(usedGIDs map[uint16]bool) []int {
	gids := make([]int, 0, len(usedGIDs)+1)
	gids = append(gids, 0)
	for gid, used := range usedGIDs {
		if used && gid != 0 {
			gids = append(gids, int(gid))
		}
	}
	sort.Ints(gids)
	return gids
}

func resolveCompoundGlyphs(usedGIDs map[uint16]bool, info subsetGlyphTables) error {
	for _, gid := range sortedUsedGlyphIDs(usedGIDs) {
		bb, err := glyphData(info, gid)
		if err != nil {
			return err
		}
		isCompound, err := compoundGlyph(bb)
		if err != nil {
			return fmt.Errorf("glyph %d: %w", gid, err)
		}
		if isCompound {
			if err := resolveCompoundGlyph(gid, bb, usedGIDs, info, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func glyfAndLoca(tables map[string]*table, usedGIDs map[uint16]bool) error {
	info, err := subsetGlyphTableInfo(tables)
	if err != nil {
		return fmt.Errorf("validate glyph tables: %w", err)
	}
	if err := resolveCompoundGlyphs(usedGIDs, info); err != nil {
		return fmt.Errorf("resolve compound glyphs: %w", err)
	}
	gids := sortedUsedGlyphIDs(usedGIDs)

	glyfBytes := []byte{}
	var buf bytes.Buffer
	var off uint64
	firstPendingGID := 0

	for _, gid := range gids {
		offFrom, offThru, err := glyphRange(info, gid)
		if err != nil {
			return fmt.Errorf("copy glyph %d: %w", gid, err)
		}
		if offThru != offFrom {
			// We have a glyph outline.
			for i := 0; i < gid-firstPendingGID; i++ {
				if err := writeGlyfOffset(&buf, off, info.locaFormat); err != nil {
					return fmt.Errorf("rebuild loca entry %d: %w", firstPendingGID+i, err)
				}
			}
			nextOff, err := nextGlyfOffset(off, offThru-offFrom, info.locaFormat)
			if err != nil {
				return fmt.Errorf("rebuild glyph %d offset: %w", gid, err)
			}
			glyfBytes = append(glyfBytes, info.glyf.data[offFrom:offThru]...)
			if err := writeGlyfOffset(&buf, off, info.locaFormat); err != nil {
				return fmt.Errorf("rebuild loca entry %d: %w", gid, err)
			}
			off = nextOff
			firstPendingGID = gid + 1
		}
	}
	for i := 0; i <= info.numGlyphs-firstPendingGID; i++ {
		if err := writeGlyfOffset(&buf, off, info.locaFormat); err != nil {
			return fmt.Errorf("rebuild loca entry %d: %w", firstPendingGID+i, err)
		}
	}

	locaData, locaSize, locaPadded, err := rebuiltTableData("loca", buf.Bytes())
	if err != nil {
		return fmt.Errorf("rebuild loca table: %w", err)
	}
	glyfData, glyfSize, glyfPadded, err := rebuiltTableData("glyf", glyfBytes)
	if err != nil {
		return fmt.Errorf("rebuild glyf table: %w", err)
	}

	info.loca.size = locaSize
	info.loca.data = locaData
	info.loca.padded = locaPadded
	info.glyf.size = glyfSize
	info.glyf.data = glyfData
	info.glyf.padded = glyfPadded

	return nil
}

type ttfOutputTable struct {
	tag      string
	table    *table
	checksum uint32
	offset   uint32
}

func invalidFontData(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ErrInvalidFontData)
}

func validateSFNTHeader(header []byte) error {
	if len(header) != 12 {
		return invalidFontData("font header: expected 12 bytes, got %d", len(header))
	}
	version := string(header[:4])
	if version != sfntVersionTrueType && version != sfntVersionTrueTypeApple {
		return invalidFontData("font header: unsupported version %q", version)
	}
	return nil
}

func validateTTFTable(tag string, t *table) (uint32, error) {
	if len(tag) != 4 {
		return 0, invalidFontData("table tag %q: expected 4 bytes", tag)
	}
	if t == nil {
		return 0, invalidFontData("table %s: missing table data", tag)
	}
	if t.size > maxFontTableSize {
		return 0, invalidFontData("table %s: size %d exceeds limit %d", tag, t.size, maxFontTableSize)
	}
	padded, err := getNext32BitAlignedLength(t.size)
	if err != nil {
		return 0, fmt.Errorf("table %s: align size: %w: %w", tag, err, ErrInvalidFontData)
	}
	if t.padded != padded {
		return 0, invalidFontData("table %s: padded size %d, expected %d", tag, t.padded, padded)
	}
	if uint64(len(t.data)) != uint64(t.padded) {
		return 0, invalidFontData("table %s: data length %d, expected %d", tag, len(t.data), t.padded)
	}
	if uint64(t.size) > uint64(len(t.data)) {
		return 0, invalidFontData("table %s: size %d exceeds data length %d", tag, t.size, len(t.data))
	}
	return padded, nil
}

func nextTTFOffset(offset uint64, size uint32) (uint64, error) {
	next := offset + uint64(size)
	if next < offset || next > uint64(^uint32(0)) {
		return 0, invalidFontData("output offset %d plus size %d exceeds uint32", offset, size)
	}
	return next, nil
}

func prepareTTFOutput(header []byte, tables map[string]*table) ([]ttfOutputTable, int, error) {
	if err := validateSFNTHeader(header); err != nil {
		return nil, 0, err
	}
	if len(tables) == 0 {
		return nil, 0, invalidFontData("font tables: missing tables")
	}
	if len(tables) > maxFontTableCount {
		return nil, 0, invalidFontData("font table count %d exceeds limit %d", len(tables), maxFontTableCount)
	}
	if got := int(binary.BigEndian.Uint16(header[4:])); got != len(tables) {
		return nil, 0, invalidFontData("font header: table count %d, expected %d", got, len(tables))
	}

	tags := make([]string, 0, len(tables))
	for tag := range tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	offset := uint64(len(header)) + uint64(len(tags))*16
	if offset > uint64(^uint32(0)) {
		return nil, 0, invalidFontData("table directory size %d exceeds uint32", offset)
	}
	outputTables := make([]ttfOutputTable, 0, len(tags))
	for _, tag := range tags {
		t := tables[tag]
		padded, err := validateTTFTable(tag, t)
		if err != nil {
			return nil, 0, err
		}
		checksum := t.chksum
		if tag == "loca" || tag == "glyf" {
			checksum = calcTableChecksum(tag, t.data)
		}
		outputTables = append(outputTables, ttfOutputTable{tag: tag, table: t, checksum: checksum, offset: uint32(offset)})
		offset, err = nextTTFOffset(offset, padded)
		if err != nil {
			return nil, 0, fmt.Errorf("table %s: output range: %w", tag, err)
		}
	}
	if offset > uint64(^uint(0)>>1) {
		return nil, 0, invalidFontData("font output size %d exceeds int", offset)
	}
	if offset > maxFontFileSize {
		return nil, 0, invalidFontData("font output size %d exceeds limit %d", offset, maxFontFileSize)
	}
	return outputTables, int(offset), nil
}

func createTTF(header []byte, tables map[string]*table) ([]byte, error) {
	outputTables, size, err := prepareTTFOutput(header, tables)
	if err != nil {
		return nil, err
	}
	bb := make([]byte, size)
	copy(bb, header)
	dirOffset := len(header)
	for _, outputTable := range outputTables {
		copy(bb[dirOffset:], outputTable.tag)
		binary.BigEndian.PutUint32(bb[dirOffset+4:], outputTable.checksum)
		binary.BigEndian.PutUint32(bb[dirOffset+8:], outputTable.offset)
		binary.BigEndian.PutUint32(bb[dirOffset+12:], outputTable.table.size)
		copy(bb[int(outputTable.offset):], outputTable.table.data)
		dirOffset += 16
	}
	for _, outputTable := range outputTables {
		outputTable.table.off = outputTable.offset
		if outputTable.tag == "loca" || outputTable.tag == "glyf" {
			outputTable.table.chksum = outputTable.checksum
		}
	}
	return bb, nil
}

// Subset creates a new font file based on usedGIDs.
func Subset(fontName string, usedGIDs map[uint16]bool) ([]byte, error) {
	if strings.TrimSpace(fontName) == "" {
		return nil, fmt.Errorf("subset font: %w", ErrMissingFontName)
	}
	if usedGIDs == nil {
		usedGIDs = map[uint16]bool{}
	} else {
		usedGIDs = maps.Clone(usedGIDs)
	}
	bb, err := Read(fontName)
	if err != nil {
		return nil, fmt.Errorf("subset font %s: read installed font: %w", fontName, err)
	}
	if len(bb) < 12 {
		return nil, fmt.Errorf("subset font %s: parse header: %w", fontName, invalidFontData("expected at least 12 bytes, got %d", len(bb)))
	}
	header := bb[:12]
	if err := validateSFNTHeader(header); err != nil {
		return nil, fmt.Errorf("subset font %s: parse header: %w", fontName, err)
	}
	tableCount := int(binary.BigEndian.Uint16(header[4:]))
	tables, err := ttfTables(tableCount, bb)
	if err != nil {
		return nil, fmt.Errorf("subset font %s: parse tables: %w", fontName, err)
	}
	if err := glyfAndLoca(tables, usedGIDs); err != nil {
		return nil, fmt.Errorf("subset font %s: subset glyphs: %w", fontName, err)
	}
	bb, err = createTTF(header, tables)
	if err != nil {
		return nil, fmt.Errorf("subset font %s: rebuild font: %w", fontName, err)
	}
	return bb, nil
}

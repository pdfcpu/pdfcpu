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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

const (
	sfntVersionTrueType      = "\x00\x01\x00\x00"
	sfntVersionTrueTypeApple = "true"
	sfntVersionCFF           = "OTTO"
	ttfHeadMagicNumber       = 0x5F0F3CF5
	ttcTag                   = "ttcf"
)

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

func (fd ttf) toPDFGlyphSpace(i int) int {
	return i * 1000 / fd.UnitsPerEm
}

type myUint32 []uint32

func (f myUint32) Len() int {
	return len(f)
}

func (f myUint32) Less(i, j int) bool {
	return f[i] < f[j]
}

func (f myUint32) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

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
	fmt.Printf("using glyphs[%08x,%08x] [%d,%d]\n", min, max, min, max)
	fmt.Printf("using glyphs #%x - #%x (%d-%d)\n", min, max, min, max)
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
	return float64(t.uint32(off)) / 65536.0
}

func (t table) parseFontHeaderTable(fd *ttf) error {
	// table "head"
	magic := t.uint32(12)
	if magic != ttfHeadMagicNumber {
		return fmt.Errorf("parseHead: wrong magic number")
	}

	unitsPerEm := t.uint16(18)
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
	for i := 0; i < len(buf); i++ {
		buf[i] = binary.BigEndian.Uint16(bb[2*i:])
	}
	return string(utf16.Decode(buf))
}

func (t table) parsePostScriptTable(fd *ttf) error {
	// table "post"
	italicAngle := t.fixed32(4)
	//fmt.Printf("italicAngle: %2.2f\n", italicAngle)
	fd.ItalicAngle = italicAngle

	isFixedPitch := t.uint16(16)
	//fmt.Printf("isFixedPitch: %t\n", isFixedPitch != 0)
	fd.FixedPitch = isFixedPitch != 0

	return nil
}

func printUnicodeRange(off int, r uint32) {
	for i := 0; i < 64; i++ {
		if r&1 > 0 {
			fmt.Printf("bit %d: on\n", off+i)
		}
		r >>= 1
	}
}

func (t table) parseWindowsMetricsTable(fd *ttf) error {
	// table "OS/2"
	version := t.uint16(0)
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
	sCapHeight := int16(0)
	if version >= 2 {
		sCapHeight = t.int16(88)
	}
	fd.CapHeight = fd.toPDFGlyphSpace(int(sCapHeight))

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
	count := int(t.uint16(2))
	stringOffset := t.uint16(4)
	var nameID uint16
	baseOff := 6
	for i := 0; i < count; i++ {
		recOff := baseOff + i*12
		pf := t.uint16(recOff)
		enc := t.uint16(recOff + 2)
		lang := t.uint16(recOff + 4)
		nameID = t.uint16(recOff + 6)
		l := t.uint16(recOff + 8)
		o := t.uint16(recOff + 10)
		soff := stringOffset + o
		s := t.data[soff : soff+l]
		if nameID == 6 {
			if pf == 3 && enc == 1 && lang == 0x0409 {
				fd.PostscriptName = utf16BEToString(s)
				return nil
			}
			if pf == 1 && enc == 0 && lang == 0 {
				fd.PostscriptName = string(s)
				return nil
			}
		}
	}

	return errors.New("pdfcpu: unable to identify postscript name")
}

func (t table) parseHorizontalHeaderTable(fd *ttf) error {
	// table "hhea"
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

	lineGap := t.int16(8)
	//fmt.Printf("lineGap: %d\n", lineGap)
	if fd.CapHeight == 0 {
		fd.CapHeight = fd.toPDFGlyphSpace(int(lineGap))
	}

	//advanceWidthMax := t.uint16(10)
	//fmt.Printf("advanceWidthMax: %d\n", advanceWidthMax)

	//minLeftSideBearing := t.int16(12)
	//fmt.Printf("minLeftSideBearing: %d\n", minLeftSideBearing)

	//minRightSideBearing := t.int16(14)
	//fmt.Printf("minRightSideBearing: %d\n", minRightSideBearing)

	//xMaxExtent := t.int16(16)
	//fmt.Printf("xMaxExtent: %d\n", xMaxExtent)

	numOfLongHorMetrics := t.uint16(34)
	//fmt.Printf("numOfLongHorMetrics: %d\n", numOfLongHorMetrics)
	fd.HorMetricsCount = int(numOfLongHorMetrics)

	return nil
}

func (t table) parseMaximumProfile(fd *ttf) error {
	// table "maxp"
	numGlyphs := t.uint16(4)
	fd.GlyphCount = int(numGlyphs)
	return nil
}

func (t table) parseHorizontalMetricsTable(fd *ttf) error {
	// table "hmtx"
	fd.GlyphWidths = make([]int, fd.GlyphCount)

	for i := 0; i < int(fd.HorMetricsCount); i++ {
		fd.GlyphWidths[i] = fd.toPDFGlyphSpace(int(t.uint16(i * 4)))
	}

	for i := fd.HorMetricsCount; i < fd.GlyphCount; i++ {
		fd.GlyphWidths[i] = fd.GlyphWidths[fd.HorMetricsCount-1]
	}

	return nil
}

func (t table) parseCMapFormat4(fd *ttf) error {
	fd.Planes[0] = true
	segCount := int(t.uint16(6) / 2)
	endOff := 14
	startOff := endOff + 2*segCount + 2
	deltaOff := startOff + 2*segCount
	rangeOff := deltaOff + 2*segCount

	count := 0
	for i := 0; i < segCount; i++ {
		sc := t.uint16(startOff + i*2)
		startCode := uint32(sc)
		if fd.FirstChar == 0 {
			fd.FirstChar = sc
		}
		ec := t.uint16(endOff + i*2)
		endCode := uint32(ec)
		if fd.LastChar == 0 {
			fd.LastChar = ec
		}
		idDelta := uint32(t.uint16(deltaOff + i*2))
		idRangeOff := int(t.uint16(rangeOff + i*2))
		var v uint16
		for c, j := startCode, 0; c <= endCode && c != 0xFFFF; c++ {
			if idRangeOff > 0 {
				v = t.uint16(rangeOff + i*2 + idRangeOff + j*2)
			} else {
				v = uint16(c + idDelta)
			}
			if gi := v; gi > 0 {
				fd.Chars[c] = gi
				fd.ToUnicode[gi] = c
				count++
			}
			j++
		}
	}
	return nil
}

func (t table) parseCMapFormat12(fd *ttf) error {
	numGroups := int(t.uint32(12))
	off := 16
	count := 0
	var (
		lowestStartCode uint32
		prevCode        uint32
	)
	for i := 0; i < numGroups; i++ {
		base := off + i*12
		startCode := t.uint32(base)
		if lowestStartCode == 0 {
			lowestStartCode = startCode
			fd.Planes[int(lowestStartCode/0x10000)] = true
		}
		if startCode/0x10000 != prevCode/0x10000 {
			fd.Planes[int(startCode/0x10000)] = true
		}
		endCode := t.uint32(base + 4)
		if startCode != endCode {
			if startCode/0x10000 != endCode/0x10000 {
				fd.Planes[int(endCode/0x10000)] = true
			}
		}
		prevCode = endCode
		startGlyphID := uint16(t.uint32(base + 8))
		for c, gi := startCode, startGlyphID; c <= endCode; c++ {
			fd.Chars[c] = gi
			fd.ToUnicode[gi] = c
			gi++
			count++
		}
	}
	return nil
}

func (t table) parseCharToGlyphMappingTable(fd *ttf) error {
	// table "cmap"

	fd.Chars = map[uint32]uint16{}
	fd.ToUnicode = map[uint16]uint32{}
	fd.Planes = map[int]bool{}
	tableCount := t.uint16(2)
	baseOff := 4
	var pf, enc, f uint16
	m := map[string]table{}

	for i := 0; i < int(tableCount); i++ {
		off := baseOff + i*8
		pf = t.uint16(off)
		enc = t.uint16(off + 2)
		o := t.uint32(off + 4)
		f = t.uint16(int(o))
		l := uint32(t.uint16(int(o) + 2))
		if f >= 8 {
			l = t.uint32(int(o) + 4)
		}
		b := t.data[o : o+l]
		t1 := table{off: o, size: uint32(l), data: b}
		k := fmt.Sprintf("p%02d.e%02d.f%02d", pf, enc, f)
		m[k] = t1
	}

	if t, ok := m["p00.e10.f12"]; ok {
		return t.parseCMapFormat12(fd)
	}
	if t, ok := m["p00.e04.f12"]; ok {
		return t.parseCMapFormat12(fd)
	}
	if t, ok := m["p03.e10.f12"]; ok {
		return t.parseCMapFormat12(fd)
	}
	if t, ok := m["p00.e03.f04"]; ok {
		return t.parseCMapFormat4(fd)
	}
	if t, ok := m["p03.e01.f04"]; ok {
		return t.parseCMapFormat4(fd)
	}

	return fmt.Errorf("pdfcpu: unsupported cmap table")
}

func calcTableChecksum(tag string, b []byte) uint32 {
	sum := uint32(0)
	c := (len(b) + 3) / 4
	for i := 0; i < c; i++ {
		if tag == "head" && i == 2 {
			continue
		}
		sum += binary.BigEndian.Uint32(b[i*4:])
	}
	return sum
}

func getNext32BitAlignedLength(i uint32) uint32 {
	if i%4 > 0 {
		return i + (4 - i%4)
	}
	return i
}

func headerAndTables(fn string, r io.ReaderAt, baseOff int64) ([]byte, map[string]*table, error) {
	header := make([]byte, 12)
	n, err := r.ReadAt(header, baseOff)
	if err != nil {
		return nil, nil, err
	}
	if n != 12 {
		return nil, nil, fmt.Errorf("pdfcpu: corrupt ttf file: %s", fn)
	}

	st := string(header[:4])

	if st == sfntVersionCFF {
		return nil, nil, fmt.Errorf("pdfcpu: %s is based on OpenType CFF and unsupported at the moment :(", fn)
	}

	if st != sfntVersionTrueType && st != sfntVersionTrueTypeApple {
		return nil, nil, fmt.Errorf("pdfcpu: unrecognized font format: %s", fn)
	}

	c := int(binary.BigEndian.Uint16(header[4:]))

	b := make([]byte, c*16)
	n, err = r.ReadAt(b, baseOff+12)
	if err != nil {
		return nil, nil, err
	}
	if n != c*16 {
		return nil, nil, fmt.Errorf("pdfcpu: corrupt ttf file: %s", fn)
	}

	byteCount := uint32(12)
	tables := map[string]*table{}

	for j := 0; j < c; j++ {
		off := j * 16
		b1 := b[off : off+16]
		tag := string(b1[:4])
		chk := binary.BigEndian.Uint32(b1[4:])
		o := binary.BigEndian.Uint32(b1[8:])
		l := binary.BigEndian.Uint32(b1[12:])
		ll := getNext32BitAlignedLength(l)
		byteCount += ll
		t := make([]byte, ll)
		n, err = r.ReadAt(t, int64(o))
		if err != nil {
			return nil, nil, err
		}
		if n != int(ll) {
			return nil, nil, fmt.Errorf("pdfcpu: corrupt table: %s", tag)
		}
		sum := calcTableChecksum(tag, t)
		if sum != chk {
			fmt.Printf("pdfcpu: fixing table<%s> checksum error; want:%d got:%d\n", tag, chk, sum)
			chk = sum
		}
		tables[tag] = &table{chksum: chk, off: o, size: l, padded: ll, data: t}
	}

	return header, tables, nil
}

func parse(tags map[string]*table, tag string, fd *ttf) error {
	t, found := tags[tag]
	if !found {
		// OS/2 is optional for True Type fonts.
		if tag == "OS/2" {
			return nil
		}
		return fmt.Errorf("pdfcpu: tag: %s unavailable", tag)
	}
	if t.data == nil {
		return fmt.Errorf("pdfcpu: tag: %s no data", tag)
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
	}

	return err
}

func writeGob(fileName string, fd ttf) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return enc.Encode(fd)
}

func readGob(fileName string, fd *ttf) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	return dec.Decode(fd)
}

func installTrueTypeRep(fontDir, fontName string, header []byte, tables map[string]*table) error {
	fd := ttf{}
	for _, v := range []string{"head", "OS/2", "post", "name", "hhea", "maxp", "hmtx", "cmap"} {
		if err := parse(tables, v, &fd); err != nil {
			return err
		}
	}

	bb, err := createTTF(header, tables)
	if err != nil {
		return err
	}
	fd.FontFile = bb

	log.CLI.Println(fd.PostscriptName)
	gobName := filepath.Join(fontDir, fd.PostscriptName+".gob")

	// Write the populated ttf struct as gob.
	if err := writeGob(gobName, fd); err != nil {
		return err
	}

	// Read gob and double check integrity.
	fdNew := ttf{}
	if err := readGob(gobName, &fdNew); err != nil {
		return err
	}

	if !reflect.DeepEqual(fd, fdNew) {
		return errors.Errorf("pdfcpu: %s can't be installed", fontName)
	}

	return nil
}

// InstallTrueTypeCollection saves an internal representation of all fonts
// contained in a TrueType collection to the pdfcpu config dir.
func InstallTrueTypeCollection(fontDir, fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	b := make([]byte, 12)
	n, err := f.Read(b)
	if err != nil {
		return err
	}
	if n != 12 {
		return fmt.Errorf("pdfcpu: corrupt ttc file: %s", fn)
	}

	if string(b[:4]) != ttcTag {
		return fmt.Errorf("pdfcpu: corrupt ttc file: %s", fn)
	}

	c := int(binary.BigEndian.Uint32(b[8:]))

	b = make([]byte, c*4)
	n, err = f.ReadAt(b, 12)
	if err != nil {
		return err
	}
	if n != c*4 {
		return fmt.Errorf("pdfcpu: corrupt ttc file: %s", fn)
	}

	// Process contained fonts.
	for i := 0; i < c; i++ {
		off := int64(binary.BigEndian.Uint32(b[i*4:]))
		header, tables, err := headerAndTables(fn, f, off)
		if err != nil {
			return err
		}
		if err := installTrueTypeRep(fontDir, fn, header, tables); err != nil {
			return err
		}
	}

	return nil
}

// InstallTrueTypeFont saves an internal representation of TrueType font fontName to the pdfcpu config dir.
func InstallTrueTypeFont(fontDir, fontName string) error {
	f, err := os.Open(fontName)
	if err != nil {
		return err
	}
	defer f.Close()

	header, tables, err := headerAndTables(fontName, f, 0)
	if err != nil {
		return err
	}
	return installTrueTypeRep(fontDir, fontName, header, tables)
}

func ttfTables(tableCount int, bb []byte) (map[string]*table, error) {
	tables := map[string]*table{}
	b := bb[12:]
	for j := 0; j < tableCount; j++ {
		off := j * 16
		b1 := b[off : off+16]
		tag := string(b1[:4])
		chksum := binary.BigEndian.Uint32(b1[4:])
		o := binary.BigEndian.Uint32(b1[8:])
		l := binary.BigEndian.Uint32(b1[12:])
		ll := getNext32BitAlignedLength(l)
		t := append([]byte(nil), bb[o:o+ll]...)
		tables[tag] = &table{chksum: chksum, off: o, size: l, padded: ll, data: t}
	}
	return tables, nil
}

func glyfOffset(loca *table, gid, indexToLocFormat int) int {
	if indexToLocFormat == 0 {
		// short offsets
		return 2 * int(loca.uint16(2*gid))
	}
	// 1 .. long offsets
	return int(loca.uint32(4 * gid))
}

func writeGlyfOffset(buf *bytes.Buffer, off, indexToLocFormat int) {
	var bb []byte
	if indexToLocFormat == 0 {
		// 0 .. short offsets
		bb = uint16ToBigEndianBytes(uint16(off / 2))
	} else {
		// 1 .. long offsets
		bb = uint32ToBigEndianBytes(uint32(off))
	}
	buf.Write(bb)
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

func glyphOffsets(gid int, locaFull, glyfsFull *table, numGlyphs, indexToLocFormat int) (int, int) {
	offFrom := glyfOffset(locaFull, gid, indexToLocFormat)
	var offThru int
	if gid == numGlyphs {
		offThru = int(glyfsFull.padded)
	} else {
		offThru = glyfOffset(locaFull, gid+1, indexToLocFormat)
	}
	return offFrom, offThru
}

func resolveCompoundGlyph(fontName string, bb []byte, usedGIDs map[uint16]bool,
	locaFull, glyfsFull *table, numGlyphs, indexToLocFormat int) error {
	last := false
	for off := 10; !last; {
		flags := binary.BigEndian.Uint16(bb[off:])
		last = flags&0x20 == 0
		wordArgs := flags&0x01 > 0

		gid := binary.BigEndian.Uint16(bb[off+2:])

		// Position behind arguments.
		off += 6
		if wordArgs {
			off += 2
		}

		// Position behind transform.
		if flags&0x08 > 0 {
			off += 2
		} else if flags&0x40 > 0 {
			off += 4
		} else if flags&0x80 > 0 {
			off += 8
		}

		if _, ok := usedGIDs[gid]; ok {
			// duplicate
			continue
		}

		offFrom, offThru := glyphOffsets(int(gid), locaFull, glyfsFull, numGlyphs, indexToLocFormat)
		if offThru < offFrom {
			return errors.Errorf("pdfcpu: illegal glyfOffset for font: %s", fontName)
		}
		if offFrom == offThru {
			// not available
			continue
		}

		usedGIDs[gid] = true

		cbb := glyfsFull.data[offFrom:offThru]
		if cbb[0]&0x80 == 0 {
			// simple
			continue
		}

		if err := resolveCompoundGlyph(fontName, cbb, usedGIDs, locaFull, glyfsFull, numGlyphs, indexToLocFormat); err != nil {
			return err
		}
	}
	return nil
}

func resolveCompoundGlyphs(fontName string, usedGIDs map[uint16]bool, locaFull, glyfsFull *table, numGlyphs, indexToLocFormat int) error {
	gids := make([]uint16, len(usedGIDs))
	for k := range usedGIDs {
		gids = append(gids, k)
	}
	for _, gid := range gids {
		offFrom, offThru := glyphOffsets(int(gid), locaFull, glyfsFull, numGlyphs, indexToLocFormat)
		if offThru < offFrom {
			return errors.Errorf("pdfcpu: illegal glyfOffset for font: %s", fontName)
		}
		if offFrom == offThru {
			continue
		}
		bb := glyfsFull.data[offFrom:offThru]
		if bb[0]&0x80 == 0 {
			// simple
			continue
		}
		if err := resolveCompoundGlyph(fontName, bb, usedGIDs, locaFull, glyfsFull, numGlyphs, indexToLocFormat); err != nil {
			return err
		}
	}
	return nil
}

func glyfAndLoca(fontName string, tables map[string]*table, usedGIDs map[uint16]bool) error {
	head, ok := tables["head"]
	if !ok {
		return errors.Errorf("pdfcpu: missing \"head\" table for font: %s", fontName)
	}

	maxp, ok := tables["maxp"]
	if !ok {
		return errors.Errorf("pdfcpu: missing \"maxp\" table for font: %s", fontName)
	}

	glyfsFull, ok := tables["glyf"]
	if !ok {
		return errors.Errorf("pdfcpu: missing \"glyf\" table for font: %s", fontName)
	}

	locaFull, ok := tables["loca"]
	if !ok {
		return errors.Errorf("pdfcpu: missing \"loca\" table for font: %s", fontName)
	}

	indexToLocFormat := int(head.uint16(50))
	// 0 .. short offsets
	// 1 .. long offsets
	numGlyphs := int(maxp.uint16(4))

	if err := resolveCompoundGlyphs(fontName, usedGIDs, locaFull, glyfsFull, numGlyphs, indexToLocFormat); err != nil {
		return err
	}

	gids := make([]int, 0, len(usedGIDs)+1)
	gids = append(gids, 0)
	for gid := range usedGIDs {
		gids = append(gids, int(gid))
	}
	sort.Ints(gids)

	glyfBytes := []byte{}
	var buf bytes.Buffer
	off := 0
	firstPendingGID := 0

	for _, gid := range gids {
		offFrom, offThru := glyphOffsets(gid, locaFull, glyfsFull, numGlyphs, indexToLocFormat)
		if offThru < offFrom {
			return errors.Errorf("pdfcpu: illegal glyfOffset for font: %s", fontName)
		}
		if offThru != offFrom {
			// We have a glyph outline.
			for i := 0; i < gid-firstPendingGID; i++ {
				writeGlyfOffset(&buf, off, indexToLocFormat)
			}
			glyfBytes = append(glyfBytes, glyfsFull.data[offFrom:offThru]...)
			writeGlyfOffset(&buf, off, indexToLocFormat)
			off += offThru - offFrom
			firstPendingGID = gid + 1
		}
	}
	for i := 0; i <= numGlyphs-firstPendingGID; i++ {
		writeGlyfOffset(&buf, off, indexToLocFormat)
	}

	bb := buf.Bytes()
	locaFull.size = uint32(len(bb))
	locaFull.data = pad(bb)
	locaFull.padded = uint32(len(locaFull.data))

	glyfsFull.size = uint32(len(glyfBytes))
	glyfsFull.data = pad(glyfBytes)
	glyfsFull.padded = uint32(len(glyfsFull.data))

	return nil
}

func createTTF(header []byte, tables map[string]*table) ([]byte, error) {
	tags := []string{}
	for t := range tables {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	buf := bytes.NewBuffer(header)
	off := uint32(len(header) + len(tables)*16)
	o := off
	for _, tag := range tags {
		t := tables[tag]
		if _, err := buf.WriteString(tag); err != nil {
			return nil, err
		}
		if tag == "loca" || tag == "glyf" {
			t.chksum = calcTableChecksum(tag, t.data)
		}
		if _, err := buf.Write(uint32ToBigEndianBytes(t.chksum)); err != nil {
			return nil, err
		}
		t.off = o
		if _, err := buf.Write(uint32ToBigEndianBytes(t.off)); err != nil {
			return nil, err
		}
		if _, err := buf.Write(uint32ToBigEndianBytes(t.size)); err != nil {
			return nil, err
		}
		o += t.padded
	}

	for _, tag := range tags {
		t := tables[tag]
		n, err := buf.Write(t.data)
		if err != nil {
			return nil, err
		}
		if n != len(t.data) || n != int(t.padded) {
			return nil, errors.Errorf("pdfcpu: unable to write %s data\n", tag)
		}
	}

	return buf.Bytes(), nil
}

// Subset creates a new font file based on usedGIDs.
func Subset(fontName string, usedGIDs map[uint16]bool) ([]byte, error) {
	bb, err := Read(fontName)
	if err != nil {
		return nil, err
	}

	header := bb[:12]
	tableCount := int(binary.BigEndian.Uint16(header[4:]))
	tables, err := ttfTables(tableCount, bb)
	if err != nil {
		return nil, err
	}

	if err := glyfAndLoca(fontName, tables, usedGIDs); err != nil {
		return nil, err
	}

	return createTTF(header, tables)
}

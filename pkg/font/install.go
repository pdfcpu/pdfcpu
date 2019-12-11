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
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

const scalerType = "\x00\x01\x00\x00"

type ttf struct {
	PostscriptName     string            // name: NameID 6
	Protected          bool              // OS/2: fsType
	UnitsPerEm         int               // head: unitsPerEm
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
	GlyphWidths        []int             // hmtx: fd.HorMetricsCount.advanceWidth
	Chars              map[uint16]uint16 // cmap
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

func (fd ttf) PrintChars() string {

	var min = uint16(0xFFFF)
	var max uint16

	var sb strings.Builder
	sb.WriteByte(0x0a)
	for c, g := range fd.Chars {
		if g > max {
			max = g
		}
		if g < min {
			min = g
		}
		sb.WriteString(fmt.Sprintf("%04x:%04x(%d)\n", c, g, g))
	}
	fmt.Printf("using glyphs[%04x,%04x] [%d,%d]\n", min, max, min, max)
	return sb.String()
}

type table struct {
	off  uint32
	size uint32
	data []byte
}

func (t table) uint16(off int) uint16 {
	return binary.BigEndian.Uint16(t.data[off : off+2])
}

func (t table) int16(off int) int16 {
	return int16(t.uint16(off))
}

func (t table) uint32(off int) uint32 {
	return binary.BigEndian.Uint32(t.data[off : off+4])
}

func (t table) fixed32(off int) float64 {
	return float64(t.uint32(off)) / 65536.0
}

func (t table) parseFontHeaderTable(fd *ttf) error {

	magic := t.uint32(12)
	//fmt.Printf("magic: %0X\n", magic)
	if magic != 0x5F0F3CF5 {
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

func (t table) parsePostScriptTable(fd *ttf) error {

	italicAngle := t.fixed32(4)
	//fmt.Printf("italicAngle: %2.2f\n", italicAngle)
	fd.ItalicAngle = italicAngle

	isFixedPitch := t.uint16(16)
	//fmt.Printf("isFixedPitch: %t\n", isFixedPitch != 0)
	fd.FixedPitch = isFixedPitch != 0

	return nil
}

func (t table) parseWindowsMetricsTable(fd *ttf) error {
	version := t.uint16(0)
	//fmt.Printf("version: %016b\n", version)

	fsType := t.uint16(8)
	//fmt.Printf("fsType: %016b\n", fsType)
	fd.Protected = fsType&2 > 0
	//fmt.Printf("protected: %t\n", fd.Protected)

	sTypoAscender := t.int16(68)
	//fmt.Printf("sTypoAscender: %d\n", sTypoAscender)
	fd.Ascent = fd.toPDFGlyphSpace(int(sTypoAscender))

	sTypoDescender := t.int16(70)
	//fmt.Printf("sTypoDescender: %d\n", sTypoDescender)
	fd.Descent = fd.toPDFGlyphSpace(int(sTypoDescender))

	// sCapHeight: This field was defined in version 2 of the OS/2 table.
	sCapHeight := int16(0)
	if version >= 2 {
		sCapHeight = t.int16(88)
	}
	//fmt.Printf("sCapHeight: %d\n", sCapHeight)
	fd.CapHeight = fd.toPDFGlyphSpace(int(sCapHeight))

	fsSelection := t.uint16(62)
	//fmt.Printf("fsSelection: %02x\n", fsSelection)
	fd.Bold = fsSelection&0x40 > 0
	//fmt.Printf("bold: %t\n", fd.Bold)

	fsFirstCharIndex := t.uint16(64)
	//fmt.Printf("fsFirstCharIndex: %d\n", fsFirstCharIndex)
	fd.FirstChar = fsFirstCharIndex

	fsLastCharIndex := t.uint16(66)
	//fmt.Printf("fsLastCharIndex: %d\n", fsLastCharIndex)
	fd.LastChar = fsLastCharIndex

	return nil
}

func (t table) parseNamingTable(fd *ttf) error {

	count := int(t.uint16(2))
	stringOffset := t.uint16(4)
	nameID := uint16(0)
	baseOff := 6
	for i := 0; i < count; i++ {
		recOff := baseOff + i*12
		pf := t.uint16(recOff) // Mac pf:1 enc:0 lang:0(english) Win: pf:3 enc:1 lang:x0409 (english)
		enc := t.uint16(recOff + 2)
		lang := t.uint16(recOff + 4)
		nameID = t.uint16(recOff + 6)
		l := t.uint16(recOff + 8)
		o := t.uint16(recOff + 10)

		soff := stringOffset + o
		s := t.data[soff : soff+l]
		//fmt.Printf("pf:%0x enc:%0x lang:%0x nameID:%0x length:%d off:%0x <%s>\n", pf, enc, lang, nameID, l, o, s)
		if nameID == 6 && (pf == 1 && enc == 0 && lang == 0) || (pf == 3 && enc == 1 && lang == 0x0409) {
			fd.PostscriptName = string(s)
			break
		}
	}

	return nil
}

func (t table) parseHorizontalHeaderTable(fd *ttf) error {

	//ascent := t.int16(4)
	//fmt.Printf("ascent: %d\n", ascent)

	//descent := t.int16(6)
	//fmt.Printf("descent: %d\n", descent)

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
	//fmt.Printf("numOfLongHorMetrics: %d\n", numOfLongHorMetrics)
	fd.HorMetricsCount = int(numOfLongHorMetrics)

	return nil
}

func (t table) parseMaximumProfile(fd *ttf) error {

	numGlyphs := t.uint16(4)
	//fmt.Printf("numGlyphs: %d\n", numGlyphs)
	fd.GlyphCount = int(numGlyphs)

	return nil
}

func (t table) parseHorizontalMetricsTable(fd *ttf) error {

	fd.GlyphWidths = make([]int, fd.GlyphCount)

	for i := 0; i < int(fd.HorMetricsCount); i++ {
		fd.GlyphWidths[i] = fd.toPDFGlyphSpace(int(t.uint16(i * 4)))
	}

	for i := fd.HorMetricsCount; i < fd.GlyphCount; i++ {
		fd.GlyphWidths[i] = fd.GlyphWidths[fd.HorMetricsCount-1]
	}

	return nil
}

func (t table) parseWinUnicodeBMPCharToGlyphMappingTable(fd *ttf) error {

	//fmt.Printf("dump:\n%s", hex.Dump(t.data))
	segCount := int(t.uint16(6) / 2)
	endOff := 14
	startOff := endOff + 2*segCount + 2
	deltaOff := startOff + 2*segCount
	rangeOff := deltaOff + 2*segCount
	//fmt.Printf("segCount:%d endC:%04x startC:%04x deltaOff:%04x rangeOff:%04x\n", segCount, endOff, startOff, deltaOff, rangeOff)

	for i := 0; i < segCount; i++ {
		startCode := t.uint16(startOff + i*2)
		endCode := t.uint16(endOff + i*2)
		idDelta := t.uint16(deltaOff + i*2)
		idRangeOff := int(t.uint16(rangeOff + i*2))
		//fmt.Printf("Segment %02d: startc:%04x endc:%04x delta:%04x(%d) rangeOff:%04x(%d)\n", i, startCode, endCode, idDelta, idDelta, idRangeOff, idRangeOff)
		v := uint16(0)
		for c, j := startCode, 0; c <= endCode && c != 0xFFFF; c++ {
			if idRangeOff > 0 {
				v = t.uint16(rangeOff + i*2 + idRangeOff + j*2)
			} else {
				v = c + idDelta
			}
			if gi := uint32(v) % uint32(65536); gi > 0 {
				fd.Chars[c] = uint16(gi)
			}
			j++
		}
	}

	return nil
}

func (t table) parseCharToGlyphMappingTable(fd *ttf) error {

	// Note: For symbolic fonts the 'cmap' and 'name' tables must use platform ID 3 (Microsoft) and encoding ID 0.

	fd.Chars = map[uint16]uint16{}

	tableCount := t.uint16(2)
	//fmt.Printf("glyphMappingTables: %d\n", tableCount)
	baseOff := 4
	var pf, enc, f uint16
	for i := 0; i < int(tableCount); i++ {
		off := baseOff + i*8
		pf = t.uint16(off)
		enc = t.uint16(off + 2)
		o := t.uint32(off + 4)
		f = t.uint16(int(o))
		l := t.uint16(int(o) + 2)
		//fmt.Printf("platformID:%d enc:%d o:%04x format:%d length:%d\n", pf, enc, o, f, l)
		if pf == 3 && enc == 1 && f == 4 {
			// We are interested in the standard character-to-glyph-index mapping table
			// for the Windows platform for fonts that support Unicode BMP characters.
			//
			// Many of the cmap formats are either obsolete or were designed to meet
			// anticipated needs which never materialized. Modern font generation tools
			// need not be able to write general-purpose cmaps in formats other than 4, 6, and 12.
			//
			// Format 4 is a two-byte encoding format.
			// It should be used when the character codes for a font fall into several contiguous ranges,
			// possibly with holes in some or all of the ranges. That is, some of the codes in a range
			// may not be associated with glyphs in the font.
			if f == 4 {
				b := t.data[o : o+uint32(l)]
				t1 := table{off: o, size: uint32(l), data: b}
				return t1.parseWinUnicodeBMPCharToGlyphMappingTable(fd)
			}

			// Format 6 is used to map 16-bit, 2-byte, characters to glyph indexes.
			// It is sometimes called the trimmed table mapping. It should be used when character codes
			// for a font fall into a single contiguous range. This results in what is termed a dense mapping.

			// Format 12 is a bit like format 4, in that it defines segments for sparse representation in a 4-byte character space.
			// It is required for Unicode fonts covering characters above U+FFFF on Windows.
			// It is the most useful of the cmap formats with 32-bit support.
		}
	}

	return fmt.Errorf("missing WinUnicodeBMPCharToGlyphMappingTable")
}

func calcTableChecksum(tag string, b []byte) uint32 {
	sum := uint32(0)
	c := (len(b) + 3) / 4
	for i := 0; i < c; i++ {
		if tag == "head" && i == 2 {
			continue
		}
		sum += binary.BigEndian.Uint32(b[i*4 : (i+1)*4])
	}
	return sum
}

func getNext32BitAlignedLength(i uint32) uint32 {
	if i%4 > 0 {
		return i + (4 - i%4)
	}
	return i
}

func parseFontDir(name string) (map[string]table, error) {

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b := make([]byte, 6)
	n, err := f.Read(b)
	if err != nil {
		return nil, err
	}
	if n != 6 {
		return nil, fmt.Errorf("corrupt file")
	}
	//fmt.Printf("read %d bytes\n%s\n", n, hex.Dump(b))

	if (string(b[:4])) != scalerType {
		return nil, fmt.Errorf("corrupt file")
	}

	c := int(binary.BigEndian.Uint16(b[4:]))
	//fmt.Printf("we have %d tables\n", c)

	b = make([]byte, c*16)
	n, err = f.ReadAt(b, 12)
	if err != nil {
		return nil, err
	}
	if n != c*16 {
		return nil, fmt.Errorf("corrupt file")
	}

	tables := map[string]table{}
	tags := []string{}

	for j := 0; j < c; j++ {
		off := j * 16
		b1 := b[off : off+16]
		//fmt.Printf("t%02d: %s", j, hex.Dump(b1))
		tag := string(b1[:4])
		chk := binary.BigEndian.Uint32(b1[4:8])
		o := binary.BigEndian.Uint32(b1[8:12])
		l := binary.BigEndian.Uint32(b1[12:])
		//fmt.Printf("tag: %s chk:%d o:%d l:%d\n\n", tag, chk, o, l)
		ll := getNext32BitAlignedLength(l)
		t := make([]byte, ll)
		n, err = f.ReadAt(t, int64(o))
		if err != nil {
			return nil, err
		}
		if n != int(ll) {
			return nil, fmt.Errorf("corrupt table")
		}

		tables[tag] = table{off: o, size: ll, data: t}

		//fmt.Printf("table <%s>:\n%s\n", tag, hex.Dump(t))
		sum := calcTableChecksum(tag, t)
		if sum != chk {
			return nil, fmt.Errorf("table<%s> checksum error", tag)
		}

		tags = append(tags, tag)
	}

	//fmt.Println(tags)

	return tables, nil
}

func parse(tags map[string]table, tag string, fd *ttf) error {
	t, found := tags[tag]
	if !found {
		return fmt.Errorf("tag: %s unavailable", tag)
	}
	if t.data == nil {
		return fmt.Errorf("tag: %s no data", tag)
	}
	//fmt.Printf("table <%s>:\n%s\n", tag, hex.Dump(t.data))

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
	//fmt.Printf("writing gob to: %s\n", fileName)
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	return enc.Encode(fd)
}

func readGob(fileName string, fd *ttf) error {
	//fmt.Printf("reading gob from: %s\n", fileName)
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	return dec.Decode(fd)
}

// InstallTrueTypeFont compiles font attributes needed to build a font descriptor.
func InstallTrueTypeFont(fontDir, fontName string) error {

	tags, err := parseFontDir(fontName)
	if err != nil {
		return err
	}

	fd := ttf{}

	for _, v := range []string{"head", "OS/2", "post", "name", "hhea", "maxp", "hmtx", "cmap"} {
		if err := parse(tags, v, &fd); err != nil {
			return err
		}
	}

	fd.FontFile, err = ioutil.ReadFile(fontName)
	if err != nil {
		return err
	}

	fn := filepath.Base(fontName)
	fn = strings.TrimSuffix(fn, filepath.Ext(fn))
	gobName := filepath.Join(fontDir, fn+".gob")

	// Write gob.
	//fmt.Printf("Write %s:\n", fd)
	if err := writeGob(gobName, fd); err != nil {
		return err
	}

	// Read gob and double check integrity.
	fdNew := ttf{}
	if err := readGob(gobName, &fdNew); err != nil {
		return err
	}
	//fmt.Printf("Read %s:\n", fdNew)

	if !reflect.DeepEqual(fd, fdNew) {
		return errors.Errorf("%s can't be installed", fontName)
	}

	return nil
}

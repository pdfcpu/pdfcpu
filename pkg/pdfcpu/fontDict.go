/*
Copyright 2020 The pdfcpu Authors.

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

package pdfcpu

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pkg/errors"
)

func flateEncodedStreamIndRef(xRefTable *XRefTable, data []byte) (*IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(data)
	sd.InsertInt("Length1", len(data))
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return xRefTable.IndRefForNewObject(*sd)
}

func ttfFontFile(xRefTable *XRefTable, ttf font.TTFLight, fontName string) (*IndirectRef, error) {
	bb, err := font.Read(fontName)
	if err != nil {
		return nil, err
	}
	return flateEncodedStreamIndRef(xRefTable, bb)
}

func ttfSubFontFile(xRefTable *XRefTable, ttf font.TTFLight, fontName string, indRef *IndirectRef) (*IndirectRef, error) {
	bb, err := font.Subset(fontName, ttf.UsedGIDs)
	if err != nil {
		return nil, err
	}
	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, bb)
	}
	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(StreamDict)
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	entry.Object = sd
	return indRef, nil
}

func coreFontDict(xRefTable *XRefTable, coreFontName string) (*IndirectRef, error) {
	d := NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", coreFontName)
	if coreFontName != "Symbol" && coreFontName != "ZapfDingbats" {
		d.InsertName("Encoding", "WinAnsiEncoding")
	}
	return xRefTable.IndRefForNewObject(d)
}

func CIDSet(xRefTable *XRefTable, ttf font.TTFLight, indRef *IndirectRef) (*IndirectRef, error) {
	bb := make([]byte, ttf.GlyphCount/8+1)
	for gid := range ttf.UsedGIDs {
		bb[gid/8] |= 1 << (7 - (gid % 8))
	}
	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, bb)
	}
	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(StreamDict)
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	entry.Object = sd
	return indRef, nil
}

func ttfFontDescriptorFlags(ttf font.TTFLight) uint32 {
	// Bits:
	// 1 FixedPitch
	// 2 Serif
	// 3 Symbolic
	// 4 Script/cursive
	// 6 Nonsymbolic
	// 7 Italic
	// 17 AllCap

	flags := uint32(0)

	// Bit 1
	//fmt.Printf("fixedPitch: %t\n", ttf.FixedPitch)
	if ttf.FixedPitch {
		flags |= 0x01
	}

	// Bit 6 Set for non symbolic
	// Note: Symbolic fonts are unsupported.
	flags |= 0x20

	// Bit 7
	//fmt.Printf("italicAngle: %f\n", ttf.ItalicAngle)
	if ttf.ItalicAngle != 0 {
		flags |= 0x40
	}

	//fmt.Printf("flags: %08x\n", flags)

	return flags
}

// CIDFontDescriptor returns a font descriptor describing the CIDFont’s default metrics other than its glyph widths.
func CIDFontDescriptor(xRefTable *XRefTable, ttf font.TTFLight, fontName, baseFontName string, subFont bool) (*IndirectRef, error) {

	var (
		fontFile *IndirectRef
		err      error
	)
	if subFont {
		fontFile, err = ttfSubFontFile(xRefTable, ttf, fontName, nil)
	} else {
		fontFile, err = ttfFontFile(xRefTable, ttf, fontName)
	}
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":        Name("FontDescriptor"),
			"FontName":    Name(baseFontName),
			"Flags":       Integer(ttfFontDescriptorFlags(ttf)),
			"FontBBox":    NewNumberArray(ttf.LLx, ttf.LLy, ttf.URx, ttf.URy),
			"ItalicAngle": Float(ttf.ItalicAngle),
			"Ascent":      Integer(ttf.Ascent),
			"Descent":     Integer(ttf.Descent),
			//"Leading": // The spacing between baselines of consecutive lines of text.
			"CapHeight": Integer(ttf.CapHeight),
			"StemV":     Integer(70), // Irrelevant for embedded files.
			"FontFile2": *fontFile,
		},
	)

	if subFont {
		// (Optional)
		// A stream identifying which CIDs are present in the CIDFont file. If this entry is present,
		// the CIDFont shall contain only a subset of the glyphs in the character collection defined by the CIDSystemInfo dictionary.
		// If it is absent, the only indication of a CIDFont subset shall be the subset tag in the FontName entry (see 9.6.4, "Font Subsets").
		// The stream’s data shall be organized as a table of bits indexed by CID.
		// The bits shall be stored in bytes with the high-order bit first. Each bit shall correspond to a CID.
		// The most significant bit of the first byte shall correspond to CID 0, the next bit to CID 1, and so on.
		cidSetIndRef, err := CIDSet(xRefTable, ttf, nil)
		if err != nil {
			return nil, err
		}
		d["CIDSet"] = *cidSetIndRef
	}

	return xRefTable.IndRefForNewObject(d)
}

func usedCIDWidths(ttf font.TTFLight) Array {
	gids := make([]int, 0, len(ttf.UsedGIDs))
	for gid := range ttf.UsedGIDs {
		gids = append(gids, int(gid))
	}
	sort.Ints(gids)

	a := Array{}
	for _, gid := range gids {
		a = append(a, Integer(gid), Array{Integer(ttf.GlyphWidths[gid])})
	}
	return a
}

func allCIDWidths(ttf font.TTFLight) Array {
	a := Array{}
	var g0, w0, wLast int
	start, equalWidths := true, true
	for gid, w := range ttf.GlyphWidths {
		fmt.Printf("gid:%d w:%d\n", gid, w)
		if start {
			g0 = gid
			w0 = w
			wLast = w
			start = false
			continue
		}
		if w == wLast {
			if equalWidths {
				continue
			}
			// non-contiguous width block
			a = append(a, Integer(g0))
			a1 := Array{}
			for i := g0; i < gid-1; i++ {
				a1 = append(a1, Integer(ttf.GlyphWidths[i]))
			}
			a = append(a, a1)
			g0 = gid - 1
			w0 = w
			equalWidths = true
			continue
		}
		wLast = w
		if !equalWidths {
			continue
		}
		if gid-g0 == 1 {
			equalWidths = false
			continue
		}
		// contiguous width block
		a = append(a, Integer(g0), Integer(gid-1), Integer(w0))
		g0 = gid
		w0 = w
	}
	if equalWidths {
		// last contiguous width block
		a = append(a, Integer(g0), Integer(len(ttf.GlyphWidths)-1), Integer(w0))
	} else {
		// last non-contiguous width block
		a = append(a, Integer(g0))
		a1 := Array{}
		for i := g0; i < len(ttf.GlyphWidths); i++ {
			a1 = append(a1, Integer(ttf.GlyphWidths[i]))
		}
		a = append(a, a1)
	}

	return a
}

// CIDWidths returns the value for W in a CIDFontDict.
func CIDWidths(xRefTable *XRefTable, ttf font.TTFLight, subFont bool, indRef *IndirectRef) (*IndirectRef, error) {

	// 108 [750]            108:750
	// 120 [400 325 500]	120:400 121:325 122:500
	// 7080 8032 1000	    7080-8032: 1000

	var a Array
	if subFont {
		a = usedCIDWidths(ttf)
	} else {
		a = allCIDWidths(ttf)
	}

	if indRef == nil {
		return xRefTable.IndRefForNewObject(a)
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	entry.Object = a

	return indRef, nil
}

// CIDFontDict returns the descendant font dict for Type0 fonts.
func CIDFontDict(xRefTable *XRefTable, ttf font.TTFLight, fontName, baseFontName string, subFont bool) (*IndirectRef, error) {
	fdIndRef, err := CIDFontDescriptor(xRefTable, ttf, fontName, baseFontName, subFont)
	if err != nil {
		return nil, err
	}

	wIndRef, err := CIDWidths(xRefTable, ttf, subFont, nil)
	if err != nil {
		return nil, err
	}

	d := Dict(
		map[string]Object{
			"Type":     Name("Font"),
			"Subtype":  Name("CIDFontType2"),
			"BaseFont": Name(baseFontName),
			"CIDSystemInfo": Dict(
				map[string]Object{
					"Ordering":   StringLiteral("Identity"),
					"Registry":   StringLiteral("Adobe"),
					"Supplement": Integer(0),
				},
			),
			"FontDescriptor": *fdIndRef,

			// (Optional)
			// The default width for glyphs in the CIDFont (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: 1000 (defined in user units).
			"DW": Integer(1000),

			// (Optional)
			// A description of the widths for the glyphs in the CIDFont.
			// The array’s elements have a variable format that can specify individual widths for consecutive CIDs
			// or one width for a range of CIDs (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: none (the DW value shall be used for all glyphs).
			"W": *wIndRef,

			// (Optional; applies only to CIDFonts used for vertical writing)
			// An array of two numbers specifying the default metrics for vertical writing (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: [880 −1000].
			// "DW2":             Integer(1000),

			// (Optional; applies only to CIDFonts used for vertical writing)
			// A description of the metrics for vertical writing for the glyphs in the CIDFont (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: none (the DW2 value shall be used for all glyphs).
			// "W2": nil

			// (Optional; Type 2 CIDFonts only)
			// A specification of the mapping from CIDs to glyph indices.
			// maps CIDs to the glyph indices for the appropriate glyph descriptions in that font program.
			// if stream: the glyph index for a particular CID value c shall be a 2-byte value stored in bytes 2 × c and 2 × c + 1,
			// where the first byte shall be the high-order byte.))
			"CIDToGIDMap": Name("Identity"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func bf(b *bytes.Buffer, ttf font.TTFLight) {
	gids := make([]int, 0, len(ttf.UsedGIDs))
	for gid := range ttf.UsedGIDs {
		gids = append(gids, int(gid))
	}
	sort.Ints(gids)

	c := 100
	if len(gids) < 100 {
		c = len(gids)
	}
	fmt.Fprintf(b, "%d beginbfchar\n", c)
	for i := 0; i < len(gids); i++ {
		fmt.Fprintf(b, "<%04X> <", gids[i])
		u := ttf.ToUnicode[uint16(gids[i])]
		s := utf16.Encode([]rune{rune(u)})
		for _, v := range s {
			fmt.Fprintf(b, "%04X", v)
		}
		fmt.Fprintf(b, ">\n")
		if i > 0 && i%100 == 0 {
			b.WriteString("endbfchar\n")
			if len(gids)-i < 100 {
				c = len(gids) - i
			}
			fmt.Fprintf(b, "%d beginbfchar\n", c)
		}
	}
	b.WriteString("endbfchar\n")
}

// toUnicodeCMap returns a stream dict containing a CMap file that maps character codes to Unicode values (see 9.10).
func toUnicodeCMap(xRefTable *XRefTable, ttf font.TTFLight, subFont bool, indRef *IndirectRef) (*IndirectRef, error) {

	// n beginbfchar
	// srcCode dstString
	// <003A>  <0037>                                            : 003a:0037
	// <3A51>  <D840DC3E>                                        : 3a51:d840dc3e
	// ...
	// endbfchar

	// n beginbfrange
	// srcCode1 srcCode2 dstString
	// <0000>   <005E>   <0020>                                  : 0000:0020 0001:0021 0002:0022 ...
	// <005F>   <0061>   [<00660066> <00660069> <00660066006C>]  : 005F:00660066 0060:00660069 0061:00660066006C
	// endbfrange

	pro := `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo <<
	/Registry (Adobe)
	/Ordering (UCS)
	/Supplement 0
>> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
`

	r := `1 begincodespacerange
<0000> <FFFF>
endcodespacerange
`

	epi := `endcmap
CMapName currentdict /CMap defineresource pop
end
end`

	var b bytes.Buffer
	b.WriteString(pro)
	b.WriteString(r)
	bf(&b, ttf)
	b.WriteString(epi)

	bb := b.Bytes()

	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, bb)
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(StreamDict)
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	entry.Object = sd

	return indRef, nil
}

var (
	errCorruptCMap     = errors.New("pdfcpu: corrupt CMap")
	errCorruptFontDict = errors.New("pdfcpu: corrupt fontDict")
)

func usedGIDsFromCMap(cMap string) ([]uint16, error) {
	gids := []uint16{}
	i := strings.Index(cMap, "endcodespacerange")
	if i < 0 {
		return nil, errCorruptCMap
	}
	scanner := bufio.NewScanner(strings.NewReader(cMap[i+len("endcodespacerange")+1:]))

	// scanLine: %d beginbfchar
	scanner.Scan()
	s := scanner.Text()

	var lastBlock bool

	for {
		ss := strings.Split(s, " ")
		i, err := strconv.Atoi(ss[0])
		if err != nil {
			return nil, errCorruptCMap
		}

		lastBlock = i < 100

		// scan i lines: <0046>********* => record usedGid(<0046>)
		for j := 0; j < i; j++ {
			scanner.Scan()
			s1 := scanner.Text()
			if s1[0] != '<' {
				return nil, errCorruptCMap
			}
			bb, err := hex.DecodeString(s1[1:5])
			if err != nil {
				return nil, errCorruptCMap
			}
			gid := binary.BigEndian.Uint16(bb)
			gids = append(gids, gid)
		}

		// scanLine: endbfchar
		scanner.Scan()
		if scanner.Text() != "endbfchar" {
			return nil, errCorruptCMap
		}

		// scanLine: endcmap => done, or %d beginbfchar
		scanner.Scan()
		s = scanner.Text()
		if s == "endcmap" {
			break
		}
		if lastBlock {
			return nil, errCorruptCMap
		}
	}

	return gids, nil
}

func IndRefsForUserfontUpdate(xRefTable *XRefTable, d Dict, font *FontResource) error {

	// ToUnicode *IndirectRef d["ToUnicode"]
	font.ToUnicode = d.IndirectRefEntry("ToUnicode")
	if font.ToUnicode == nil {
		return errCorruptFontDict
	}

	o, found := d.Find("DescendantFonts")
	if !found {
		return errCorruptFontDict
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return err
	}

	if len(a) != 1 {
		return errCorruptFontDict
	}

	df, err := xRefTable.DereferenceDict(a[0])
	if err != nil {
		return err
	}

	// W *IndirectRef d["DescendantFonts"][0]["W"]
	font.W = df.IndirectRefEntry("W")
	if font.W == nil {
		return errCorruptFontDict
	}

	o, found = df.Find("FontDescriptor")
	if !found {
		return errCorruptFontDict
	}

	fd, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	// CIDSet *IndirectRef d["DescendantFonts"][0]["FontDescriptor"]["CIDSet"]
	font.CIDSet = fd.IndirectRefEntry("CIDSet")
	if font.CIDSet == nil {
		return errCorruptFontDict
	}

	// FontFile *IndirectRef d["DescendantFonts"][0]["FontDescriptor"]["FontFile2"]
	font.FontFile = fd.IndirectRefEntry("FontFile2")
	if font.FontFile == nil {
		return errCorruptFontDict
	}

	return nil
}

func usedGIDsFromCMapIndRef(xRefTable *XRefTable, cmapIndRef IndirectRef) ([]uint16, error) {
	sd, _, err := xRefTable.DereferenceStreamDict(cmapIndRef)
	if err != nil {
		return nil, err
	}
	if err := sd.Decode(); err != nil {
		return nil, err
	}
	return usedGIDsFromCMap(string(sd.Content))
}

func subFontPrefix() string {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var r *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	bb := make([]byte, 6)
	for i := range bb {
		bb[i] = s[r.Intn(len(s))]
	}
	return string(bb)
}

func type0FontDict(xRefTable *XRefTable, fontName string, subFont bool, indRef *IndirectRef) (*IndirectRef, error) {
	// Combines a CIDFont and a CMap to produce a font whose glyphs may be accessed
	// by means of variable-length character codes in a string to be shown.

	ttf, ok := font.UserFontMetrics[fontName]
	if !ok {
		return nil, errors.Errorf("pdfcpu: font %s not available", fontName)
	}

	baseFontName := fontName
	if subFont {
		baseFontName = subFontPrefix() + "+" + fontName
	}

	descendentFontIndRef, err := CIDFontDict(xRefTable, ttf, fontName, baseFontName, subFont)
	if err != nil {
		return nil, err
	}

	d := NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type0")
	d.InsertName("BaseFont", baseFontName)
	d.InsertName("Encoding", "Identity-H")
	d.Insert("DescendantFonts", Array{*descendentFontIndRef})

	if subFont {

		toUnicodeIndRef, err := toUnicodeCMap(xRefTable, ttf, true, nil)
		if err != nil {
			return nil, err
		}
		d.Insert("ToUnicode", *toUnicodeIndRef)
	}
	// TODO else write full toUniCodeCmap.

	// Reset used glyph ids.
	ttf.UsedGIDs = map[uint16]bool{}

	if indRef == nil {
		return xRefTable.IndRefForNewObject(d)
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	entry.Object = d

	return indRef, nil
}

func EnsureFontDict(xRefTable *XRefTable, fontName string, subDict bool, indRef *IndirectRef) (*IndirectRef, error) {
	if font.IsCoreFont(fontName) {
		if indRef != nil {
			return indRef, nil
		}
		return coreFontDict(xRefTable, fontName)
	}
	return type0FontDict(xRefTable, fontName, subDict, indRef)
}

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
	"bytes"
	"fmt"
	"math/rand"
	"sort"
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

func ttfSubFontFile(xRefTable *XRefTable, ttf font.TTFLight, fontName string) (*IndirectRef, error) {
	bb, err := font.Subset(fontName, ttf.UsedGIDs)
	if err != nil {
		return nil, err
	}
	return flateEncodedStreamIndRef(xRefTable, bb)
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

func cidSet(xRefTable *XRefTable, ttf font.TTFLight) (*IndirectRef, error) {
	bb := make([]byte, ttf.GlyphCount/8+1)
	for gid := range ttf.UsedGIDs {
		bb[gid/8] |= 1 << (7 - (gid % 8))
	}
	return flateEncodedStreamIndRef(xRefTable, bb)
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

// CIDFontDescriptor represents a font descriptor describing
// the CIDFont’s default metrics other than its glyph widths.
func CIDFontDescriptor(xRefTable *XRefTable, ttf font.TTFLight, fontName, baseFontName string) (*IndirectRef, error) {
	//fontFile, err := ttfFontFile(xRefTable, ttf, fontName)
	fontFile, err := ttfSubFontFile(xRefTable, ttf, fontName)
	if err != nil {
		return nil, err
	}

	cidSetIndRef, err := cidSet(xRefTable, ttf)
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

			// (Optional) A dictionary containing entries that describe the style of the glyphs in the font (see 9.8.3.2, "Style").
			//"Style": Dict(map[string]Object{}),

			// (Optional) A name specifying the language of the font, which may be used for encodings
			// where the language is not implied by the encoding itself.
			//"Lang": Name(""),

			// (Optional) A dictionary whose keys identify a class of glyphs in a CIDFont.
			// Each value shall be a dictionary containing entries that shall override the
			// corresponding values in the main font descriptor dictionary for that class of glyphs (see 9.8.3.3, "FD").
			//"FD": Dict(map[string]Object{}),

			// (Optional)
			// A stream identifying which CIDs are present in the CIDFont file. If this entry is present,
			// the CIDFont shall contain only a subset of the glyphs in the character collection defined by the CIDSystemInfo dictionary.
			// If it is absent, the only indication of a CIDFont subset shall be the subset tag in the FontName entry (see 9.6.4, "Font Subsets").
			// The stream’s data shall be organized as a table of bits indexed by CID.
			// The bits shall be stored in bytes with the high-order bit first. Each bit shall correspond to a CID.
			// The most significant bit of the first byte shall correspond to CID 0, the next bit to CID 1, and so on.
			"CIDSet": *cidSetIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

// CIDWidths returns the value for W in a CIDFontDict.
func CIDWidths(ttf font.TTFLight) Array {
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

// CIDFontDict returns the descendant font dict for Type0 fonts.
func CIDFontDict(xRefTable *XRefTable, ttf font.TTFLight, fontName, baseFontName string) (*IndirectRef, error) {
	fdIndRef, err := CIDFontDescriptor(xRefTable, ttf, fontName, baseFontName)
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
			"W": CIDWidths(ttf),

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
func toUnicodeCMap(xRefTable *XRefTable, ttf font.TTFLight, fontName string) (*IndirectRef, error) {
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

	return flateEncodedStreamIndRef(xRefTable, b.Bytes())
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

func type0FontDict(xRefTable *XRefTable, fontName string) (*IndirectRef, error) {
	// Combines a CIDFont and a CMap to produce a font whose glyphs may be accessed
	// by means of variable-length character codes in a string to be shown.
	ttf, ok := font.UserFontMetrics[fontName]
	if !ok {
		return nil, errors.Errorf("pdfcpu: font %s not available", fontName)
	}

	baseFontName := subFontPrefix() + "-" + fontName

	descendentFontIndRef, err := CIDFontDict(xRefTable, ttf, fontName, baseFontName)
	if err != nil {
		return nil, err
	}

	toUnicodeIndRef, err := toUnicodeCMap(xRefTable, ttf, fontName)
	if err != nil {
		return nil, err
	}

	// Reset used glyph ids.
	ttf.UsedGIDs = map[uint16]bool{}

	d := NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type0")
	d.InsertName("BaseFont", baseFontName)
	d.InsertName("Encoding", "Identity-H")
	d.Insert("DescendantFonts", Array{*descendentFontIndRef})
	d.Insert("ToUnicode", *toUnicodeIndRef)

	return xRefTable.IndRefForNewObject(d)
}

func createFontDict(xRefTable *XRefTable, fontName string) (*IndirectRef, error) {
	if font.IsCoreFont(fontName) {
		return coreFontDict(xRefTable, fontName)
	}
	return type0FontDict(xRefTable, fontName)
}

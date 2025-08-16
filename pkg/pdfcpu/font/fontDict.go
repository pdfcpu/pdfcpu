/*
Copyright 2022 The pdfcpu Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type cjk struct {
	encoding   string
	ordering   string
	supplement int
}

// Mapping of supported ISO-15924 font script code keys to corresponding encoding and CIDSystemInfo.
var cjkParms = map[string]cjk{
	// C
	"HANS": {"UniGB-UTF16-H", "GB1", 5},
	"HANT": {"UniCNS-UTF16-H", "CNS1", 7},
	// J
	"HIRA": {"UniJIS-UTF16-H", "Japan1", 7},
	"KANA": {"UniJIS-UTF16-H", "Japan1", 7},
	"JPAN": {"UniJIS-UTF16-H", "Japan1", 7},
	// K
	"HANG": {"UniKS-UTF16-H", "Korea1", 1},
	"KORE": {"UniKS-UTF16-H", "Korea1", 1},
	//"HANG": {"UniKS-UTF16-H", "KR", 9},
	//"KORE": {"UniKS-UTF16-H", "KR", 9},
}

func SupportedScript(s string) bool {
	return types.MemberOf(s, []string{"HANS", "HANT", "HIRA", "KANA", "JPAN", "HANG", "KORE"})
}

// CJKEncodings returns true for supported encodings.
func CJKEncoding(s string) bool {
	return types.MemberOf(s, []string{"UniGB-UTF16-H", "UniCNS-UTF16-H", "UniJIS-UTF16-H", "UniKS-UTF16-H"})
}

func ScriptForEncoding(enc string) string {
	for k, v := range cjkParms {
		if v.encoding == enc {
			return k
		}
	}
	return ""
}

func fontDescriptorIndRefs(fd types.Dict, lang string, font *model.FontResource) error {
	if lang != "" {
		if s := fd.NameEntry("Lang"); s != nil {
			if strings.ToLower(*s) != lang {
				return ErrCorruptFontDict
			}
		}
	}

	font.CIDSet = fd.IndirectRefEntry("CIDSet")
	if font.CIDSet == nil {
		return ErrCorruptFontDict
	}

	font.FontFile = fd.IndirectRefEntry("FontFile2")
	if font.FontFile == nil {
		return ErrCorruptFontDict
	}

	return nil
}

// IndRefsForUserfontUpdate detects used indirect references for a possible user font update.
func IndRefsForUserfontUpdate(xRefTable *model.XRefTable, d types.Dict, lang string, font *model.FontResource) error {
	if enc := d.NameEntry("Encoding"); enc == nil || *enc != "Identity-H" {
		return ErrCorruptFontDict
	}

	// TODO some indRefs may be direct objs => don't reuse userFont.

	font.ToUnicode = d.IndirectRefEntry("ToUnicode")
	if font.ToUnicode == nil {
		return ErrCorruptFontDict
	}

	o, found := d.Find("DescendantFonts")
	if !found {
		return ErrCorruptFontDict
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return err
	}

	if len(a) != 1 {
		return ErrCorruptFontDict
	}

	df, err := xRefTable.DereferenceDict(a[0])
	if err != nil {
		return err
	}

	font.W = df.IndirectRefEntry("W")
	if font.W == nil {
		return ErrCorruptFontDict
	}

	o, found = df.Find("FontDescriptor")
	if !found {
		return ErrCorruptFontDict
	}

	fd, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	return fontDescriptorIndRefs(fd, lang, font)
}

func flateEncodedStreamIndRef(xRefTable *model.XRefTable, data []byte) (*types.IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(data)
	sd.InsertInt("Length1", len(data))
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return xRefTable.IndRefForNewObject(*sd)
}

func ttfFontFile(xRefTable *model.XRefTable, fontName string) (*types.IndirectRef, error) {
	bb, err := font.Read(fontName)
	if err != nil {
		return nil, err
	}
	return flateEncodedStreamIndRef(xRefTable, bb)
}

func ttfSubFontFile(xRefTable *model.XRefTable, fontName string, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	bb, err := font.Subset(fontName, xRefTable.UsedGIDs[fontName])
	if err != nil {
		return nil, err
	}
	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, bb)
	}
	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(types.StreamDict)
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	entry.Object = sd
	return indRef, nil
}

func PDFDocEncoding(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	arr := types.Array{
		types.Integer(24),
		types.Name("breve"), types.Name("caron"), types.Name("circumflex"), types.Name("dotaccent"),
		types.Name("hungarumlaut"), types.Name("ogonek"), types.Name("ring"), types.Name("tilde"),
		types.Integer(39),
		types.Name("quotesingle"),
		types.Integer(96),
		types.Name("grave"),
		types.Integer(128),
		types.Name("bullet"), types.Name("dagger"), types.Name("daggerdbl"), types.Name("ellipsis"), types.Name("emdash"), types.Name("endash"),
		types.Name("florin"), types.Name("fraction"), types.Name("guilsinglleft"), types.Name("guilsinglright"), types.Name("minus"), types.Name("perthousand"),
		types.Name("quotedblbase"), types.Name("quotedblleft"), types.Name("quotedblright"), types.Name("quoteleft"), types.Name("quoteright"), types.Name("quotesinglbase"),
		types.Name("trademark"), types.Name("fi"), types.Name("fl"), types.Name("Lslash"), types.Name("OE"), types.Name("Scaron"), types.Name("Ydieresis"),
		types.Name("Zcaron"), types.Name("dotlessi"), types.Name("lslash"), types.Name("oe"), types.Name("scaron"), types.Name("zcaron"),
		types.Integer(160),
		types.Name("Euro"),
		types.Integer(164),
		types.Name("currency"),
		types.Integer(166),
		types.Name("brokenbar"), types.Integer(168), types.Name("dieresis"), types.Name("copyright"), types.Name("ordfeminine"),
		types.Integer(172),
		types.Name("logicalnot"), types.Name(".notdef"), types.Name("registered"), types.Name("macron"), types.Name("degree"),
		types.Name("plusminus"), types.Name("twosuperior"), types.Name("threesuperior"), types.Name("acute"), types.Name("mu"),
		types.Integer(183),
		types.Name("periodcentered"), types.Name("cedilla"), types.Name("onesuperior"), types.Name("ordmasculine"),
		types.Integer(188),
		types.Name("onequarter"), types.Name("onehalf"), types.Name("threequarters"),
		types.Integer(192),
		types.Name("Agrave"), types.Name("Aacute"), types.Name("Acircumflex"), types.Name("Atilde"), types.Name("Adieresis"), types.Name("Aring"), types.Name("AE"),
		types.Name("Ccedilla"), types.Name("Egrave"), types.Name("Eacute"), types.Name("Ecircumflex"), types.Name("Edieresis"), types.Name("Igrave"), types.Name("Iacute"),
		types.Name("Icircumflex"), types.Name("Idieresis"), types.Name("Eth"), types.Name("Ntilde"), types.Name("Ograve"), types.Name("Oacute"), types.Name("Ocircumflex"),
		types.Name("Otilde"), types.Name("Odieresis"), types.Name("multiply"), types.Name("Oslash"), types.Name("Ugrave"), types.Name("Uacute"), types.Name("Ucircumflex"),
		types.Name("Udieresis"), types.Name("Yacute"), types.Name("Thorn"), types.Name("germandbls"), types.Name("agrave"), types.Name("aacute"), types.Name("acircumflex"),
		types.Name("atilde"), types.Name("adieresis"), types.Name("aring"), types.Name("ae"), types.Name("ccedilla"), types.Name("egrave"), types.Name("eacute"), types.Name("ecircumflex"),
		types.Name("edieresis"), types.Name("igrave"), types.Name("iacute"), types.Name("icircumflex"), types.Name("idieresis"), types.Name("eth"), types.Name("ntilde"),
		types.Name("ograve"), types.Name("oacute"), types.Name("ocircumflex"), types.Name("otilde"), types.Name("odieresis"), types.Name("divide"), types.Name("oslash"),
		types.Name("ugrave"), types.Name("uacute"), types.Name("ucircumflex"), types.Name("udieresis"), types.Name("yacute"), types.Name("thorn"), types.Name("ydieresis"),
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":        types.Name("Encoding"),
			"Differences": arr,
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func coreFontDict(xRefTable *model.XRefTable, coreFontName string) (*types.IndirectRef, error) {
	d := types.NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", coreFontName)
	if coreFontName != "Symbol" && coreFontName != "ZapfDingbats" {
		d.InsertName("Encoding", "WinAnsiEncoding")
	}
	return xRefTable.IndRefForNewObject(d)
}

// CIDSet computes a CIDSet for used glyphs and updates or returns a new object.
func CIDSet(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	bb := make([]byte, ttf.GlyphCount/8+1)
	usedGIDs, ok := xRefTable.UsedGIDs[fontName]
	if ok {
		for gid := range usedGIDs {
			bb[gid/8] |= 1 << (7 - (gid % 8))
		}
	}
	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, bb)
	}
	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(types.StreamDict)
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

// CIDFontFile returns a TrueType font file or subfont file for fontName.
func CIDFontFile(xRefTable *model.XRefTable, fontName string, subFont bool) (*types.IndirectRef, error) {
	if subFont {
		return ttfSubFontFile(xRefTable, fontName, nil)
	}
	return ttfFontFile(xRefTable, fontName)
}

// CIDFontDescriptor returns a font descriptor describing the CIDFont’s default metrics other than its glyph widths.
func CIDFontDescriptor(xRefTable *model.XRefTable, ttf font.TTFLight, fontName, baseFontName, fontLang string, embed bool) (*types.IndirectRef, error) {
	var (
		fontFile *types.IndirectRef
		err      error
	)

	d := types.Dict(
		map[string]types.Object{
			"Type":        types.Name("FontDescriptor"),
			"FontName":    types.Name(baseFontName),
			"Flags":       types.Integer(ttfFontDescriptorFlags(ttf)),
			"FontBBox":    types.NewNumberArray(ttf.LLx, ttf.LLy, ttf.URx, ttf.URy),
			"ItalicAngle": types.Float(ttf.ItalicAngle),
			"Ascent":      types.Integer(ttf.Ascent),
			"Descent":     types.Integer(ttf.Descent),
			"CapHeight":   types.Integer(ttf.CapHeight),
			"StemV":       types.Integer(70), // Irrelevant for embedded files.
		},
	)

	if embed {
		fontFile, err = CIDFontFile(xRefTable, fontName, true)
		if err != nil {
			return nil, err
		}
		d["FontFile2"] = *fontFile
	}

	if embed {
		// (Optional)
		// A stream identifying which CIDs are present in the CIDFont file. If this entry is present,
		// the CIDFont shall contain only a subset of the glyphs in the character collection defined by the CIDSystemInfo dictionary.
		// If it is absent, the only indication of a CIDFont subset shall be the subset tag in the FontName entry (see 9.6.4, "Font Subsets").
		// The stream’s data shall be organized as a table of bits indexed by CID.
		// The bits shall be stored in bytes with the high-order bit first. Each bit shall correspond to a CID.
		// The most significant bit of the first byte shall correspond to CID 0, the next bit to CID 1, and so on.
		cidSetIndRef, err := CIDSet(xRefTable, ttf, fontName, nil)
		if err != nil {
			return nil, err
		}
		d["CIDSet"] = *cidSetIndRef
	}

	if fontLang != "" {
		d["Lang"] = types.Name(fontLang)
	}

	return xRefTable.IndRefForNewObject(d)
}

// FontDescriptor returns a TrueType font descriptor describing font’s default metrics other than its glyph widths.
func NewFontDescriptor(xRefTable *model.XRefTable, ttf font.TTFLight, fontName, fontLang string) (*types.IndirectRef, error) {
	fontFile, err := ttfFontFile(xRefTable, fontName)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Ascent":      types.Integer(ttf.Ascent),
			"CapHeight":   types.Integer(ttf.CapHeight),
			"Descent":     types.Integer(ttf.Descent),
			"Flags":       types.Integer(ttfFontDescriptorFlags(ttf)),
			"FontBBox":    types.NewNumberArray(ttf.LLx, ttf.LLy, ttf.URx, ttf.URy),
			"FontFamily":  types.StringLiteral(fontName),
			"FontFile2":   *fontFile,
			"FontName":    types.Name(fontName),
			"ItalicAngle": types.Float(ttf.ItalicAngle),
			"StemV":       types.Integer(70), // Irrelevant for embedded files.
			"Type":        types.Name("FontDescriptor"),
		},
	)

	if fontLang != "" {
		d["Lang"] = types.Name(fontLang)
	}

	return xRefTable.IndRefForNewObject(d)
}

func wArr(ttf font.TTFLight, from, thru int) types.Array {
	a := types.Array{}
	for i := from; i <= thru; i++ {
		a = append(a, types.Integer(ttf.GlyphWidths[i]))
	}
	return a
}

func prepGids(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, used bool) []int {
	gids := ttf.GlyphWidths
	if used {
		usedGIDs, ok := xRefTable.UsedGIDs[fontName]
		if ok {
			gids = make([]int, 0, len(usedGIDs))
			for gid := range usedGIDs {
				gids = append(gids, int(gid))
			}
			sort.Ints(gids)
		}
	}
	return gids
}

func handleEqualWidths(w, w0, wl, g, g0, gl *int, a *types.Array, skip, equalWidths *bool) {
	if *w == 1000 || *w != *wl || *g-*gl > 1 {
		// cutoff or switch to non-contiguous width block
		*a = append(*a, types.Integer(*g0), types.Integer(*gl), types.Integer(*w0)) // write last contiguous width block
		if *w == 1000 {
			// cutoff via default
			*skip = true
		} else {
			*g0, *w0 = *g, *w
			*gl, *wl = *g0, *w0
		}
		*equalWidths = false
	} else {
		// Remain in contiguous width block
		*gl = *g
	}
}

func finalizeWidths(ttf font.TTFLight, w0, g0, gl int, skip, equalWidths bool, a *types.Array) {
	if !skip {
		if equalWidths {
			// write last contiguous width block
			*a = append(*a, types.Integer(g0), types.Integer(gl), types.Integer(w0))
		} else {
			// write last non-contiguous width block
			*a = append(*a, types.Integer(g0))
			a1 := wArr(ttf, g0, gl)
			*a = append(*a, a1)
		}
	}
}

func calcWidthArray(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, used bool) types.Array {
	gids := prepGids(xRefTable, ttf, fontName, used)
	a := types.Array{}
	var g0, w0, gl, wl int
	start, equalWidths, skip := true, false, false

	for g, w := range gids {
		if used {
			g = w
			w = ttf.GlyphWidths[g]
		}

		if start {
			start = false
			if w == 1000 {
				skip = true
				continue
			}
			g0, w0 = g, w
			gl, wl = g0, w0
			continue
		}

		if skip {
			if w != 1000 {
				g0, w0 = g, w
				gl, wl = g0, w0
				skip, equalWidths = false, false
			}
			continue
		}

		if equalWidths {
			handleEqualWidths(&w, &w0, &wl, &g, &g0, &gl, &a, &skip, &equalWidths)
			continue
		}

		// Non-contiguous

		if w == 1000 {
			// cutoff via default
			a = append(a, types.Integer(g0)) // write non-contiguous width block
			a1 := wArr(ttf, g0, gl)
			a = append(a, a1)
			skip = true
			continue
		}

		if g-gl > 1 {
			// cutoff via gap for subsets only.
			a = append(a, types.Integer(g0)) // write non-contiguous width block
			a1 := wArr(ttf, g0, gl)
			a = append(a, a1)
			g0, w0 = g, w
			gl, wl = g0, w0
			continue
		}

		if w == wl {
			if g-g0 > 1 {
				// switch from non equalW to equalW
				a = append(a, types.Integer(g0)) // write non-contiguous width block
				tru := gl - 1
				if tru < g0 {
					tru = g0
				}
				a1 := wArr(ttf, g0, tru)
				a = append(a, a1)
				g0, w0 = gl, wl
			}
			// just started.
			// switch to contiguous width
			equalWidths = true
			gl = g
			continue
		}

		// Remain in non-contiguous width block
		gl, wl = g, w
	}

	finalizeWidths(ttf, w0, g0, gl, skip, equalWidths, &a)

	return a
}

// CIDWidths returns the value for W in a CIDFontDict.
func CIDWidths(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, subFont bool, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	a := calcWidthArray(xRefTable, ttf, fontName, subFont)
	if len(a) == 0 {
		return nil, nil
	}

	if indRef == nil {
		return xRefTable.IndRefForNewObject(a)
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	entry.Object = a

	return indRef, nil
}

// Widths returns the value for Widths in a TrueType FontDict.
func Widths(xRefTable *model.XRefTable, ttf font.TTFLight, first, last int) (*types.IndirectRef, error) {
	a := types.Array{}
	for i := first; i < last; i++ {
		pos, ok := ttf.Chars[uint32(i)]
		if !ok {
			pos = 0 // should be the "invalid char"
		}
		a = append(a, types.Integer(ttf.GlyphWidths[pos]))
	}
	return xRefTable.IndRefForNewObject(a)
}

func bf(b *bytes.Buffer, ttf font.TTFLight, usedGIDs map[uint16]bool, subFont bool) {
	var gids []int
	if subFont {
		gids = make([]int, 0, len(usedGIDs))
		for gid := range usedGIDs {
			gids = append(gids, int(gid))
		}
	} else {
		gids = ttf.Gids()
	}
	sort.Ints(gids)

	c := 100
	if len(gids) < 100 {
		c = len(gids)
	}
	l := c

	fmt.Fprintf(b, "%d beginbfchar\n", c)
	j := 1
	for i := 0; i < l; i++ {
		gid := gids[i]
		fmt.Fprintf(b, "<%04X> <", gid)
		u := ttf.ToUnicode[uint16(gid)]
		s := utf16.Encode([]rune{rune(u)})
		for _, v := range s {
			fmt.Fprintf(b, "%04X", v)
		}
		fmt.Fprintf(b, ">\n")
		if j%100 == 0 {
			b.WriteString("endbfchar\n")
			if l-i < 100 {
				c = l - i
			}
			fmt.Fprintf(b, "%d beginbfchar\n", c)
		}
		j++
	}
	b.WriteString("endbfchar\n")
}

// toUnicodeCMap returns a stream dict containing a CMap file that maps character codes to Unicode values (see 9.10).
func toUnicodeCMap(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, subFont bool, indRef *types.IndirectRef) (*types.IndirectRef, error) {
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
	usedGIDs := xRefTable.UsedGIDs[fontName]
	if usedGIDs == nil {
		usedGIDs = map[uint16]bool{}
	}
	bf(&b, ttf, usedGIDs, subFont)
	b.WriteString(epi)

	bb := b.Bytes()

	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, bb)
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(types.StreamDict)
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
	ErrCorruptFontDict = errors.New("pdfcpu: corrupt fontDict")
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

		// scan i lines:
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

// UpdateUserfont updates the fontdict for fontName via supplied font resource.
func UpdateUserfont(xRefTable *model.XRefTable, fontName string, f model.FontResource) error {
	font.UserFontMetricsLock.RLock()
	ttf, ok := font.UserFontMetrics[fontName]
	font.UserFontMetricsLock.RUnlock()

	if !ok {
		return errors.Errorf("pdfcpu: userfont %s not available", fontName)
	}

	if err := usedGIDsFromCMapIndRef(xRefTable, fontName, *f.ToUnicode); err != nil {
		return err
	}

	if _, err := toUnicodeCMap(xRefTable, ttf, fontName, true, f.ToUnicode); err != nil {
		return err
	}

	if _, err := ttfSubFontFile(xRefTable, fontName, f.FontFile); err != nil {
		return err
	}

	if _, err := CIDWidths(xRefTable, ttf, fontName, true, f.W); err != nil {
		return err
	}

	if _, err := CIDSet(xRefTable, ttf, fontName, f.CIDSet); err != nil {
		return err
	}

	return nil
}

func usedGIDsFromCMapIndRef(xRefTable *model.XRefTable, fontName string, cmapIndRef types.IndirectRef) error {
	sd, _, err := xRefTable.DereferenceStreamDict(cmapIndRef)
	if err != nil {
		return err
	}
	if err := sd.Decode(); err != nil {
		return err
	}
	gids, err := usedGIDsFromCMap(string(sd.Content))
	if err != nil {
		return err
	}
	m, ok := xRefTable.UsedGIDs[fontName]
	if !ok {
		m = map[uint16]bool{}
		xRefTable.UsedGIDs[fontName] = m
	}
	for _, gid := range gids {
		m[gid] = true
	}
	return nil
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

// CIDFontDict returns the descendant font dict with special encoding for Type0 fonts.
func CIDFontDict(xRefTable *model.XRefTable, ttf font.TTFLight, fontName, baseFontName, lang string, parms *cjk) (*types.IndirectRef, error) {
	fdIndRef, err := CIDFontDescriptor(xRefTable, ttf, fontName, baseFontName, lang, parms == nil)
	if err != nil {
		return nil, err
	}

	ordering := "Identity"
	if parms != nil {
		ordering = parms.ordering
	}

	supplement := 0
	if parms != nil {
		supplement = parms.supplement
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":     types.Name("Font"),
			"Subtype":  types.Name("CIDFontType2"),
			"BaseFont": types.Name(baseFontName),
			"CIDSystemInfo": types.Dict(
				map[string]types.Object{
					"Ordering":   types.StringLiteral(ordering),
					"Registry":   types.StringLiteral("Adobe"),
					"Supplement": types.Integer(supplement),
				},
			),
			"FontDescriptor": *fdIndRef,

			// (Optional)
			// The default width for glyphs in the CIDFont (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: 1000 (defined in user units).
			// "DW": types.Integer(1000),

			// (Optional)
			// A description of the widths for the glyphs in the CIDFont.
			// The array’s elements have a variable format that can specify individual widths for consecutive CIDs
			// or one width for a range of CIDs (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: none (the DW value shall be used for all glyphs).
			//"W": *wIndRef,

			// (Optional; applies only to CIDFonts used for vertical writing)
			// An array of two numbers specifying the default metrics for vertical writing (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: [880 −1000].
			// "DW2":             Integer(1000),

			// (Optional; applies only to CIDFonts used for vertical writing)
			// A description of the metrics for vertical writing for the glyphs in the CIDFont (see 9.7.4.3, "Glyph Metrics in CIDFonts").
			// Default value: none (the DW2 value shall be used for all glyphs).
			// "W2": nil,
		},
	)

	// (Optional; Type 2 CIDFonts only)
	// A specification of the mapping from CIDs to glyph indices.
	// maps CIDs to the glyph indices for the appropriate glyph descriptions in that font program.
	// if stream: the glyph index for a particular CID value c shall be a 2-byte value stored in bytes 2 × c and 2 × c + 1,
	// where the first byte shall be the high-order byte.))
	if ordering == "Identity" {
		d["CIDToGIDMap"] = types.Name("Identity")
	}

	if parms == nil {
		wIndRef, err := CIDWidths(xRefTable, ttf, fontName, parms == nil, nil)
		if err != nil {
			return nil, err
		}
		if wIndRef != nil {
			d["W"] = *wIndRef
		}
	}

	return xRefTable.IndRefForNewObject(d)
}

func type0FontDict(xRefTable *model.XRefTable, fontName, lang, script string, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	font.UserFontMetricsLock.RLock()
	ttf, ok := font.UserFontMetrics[fontName]
	font.UserFontMetricsLock.RUnlock()
	if !ok {
		return nil, errors.Errorf("pdfcpu: font %s not available", fontName)
	}

	subFont := script == ""

	// For consecutive pages or if no AP present using this font.
	if indRef != nil && subFont && !xRefTable.HasUsedGIDs(fontName) {
		if obj, _ := xRefTable.Dereference(*indRef); obj != nil {
			return indRef, nil
		}
	}

	baseFontName := fontName
	if subFont {
		baseFontName = subFontPrefix() + "+" + fontName
	}

	var parms *cjk
	if p, ok := cjkParms[script]; ok {
		parms = &p
	}

	encoding := "Identity-H"
	if parms != nil {
		encoding = parms.encoding
	}

	descendentFontIndRef, err := CIDFontDict(xRefTable, ttf, fontName, baseFontName, lang, parms)
	if err != nil {
		return nil, err
	}

	d := types.NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type0")
	d.InsertName("BaseFont", baseFontName)
	d.InsertName("Name", fontName)
	d.InsertName("Encoding", encoding)
	d.Insert("DescendantFonts", types.Array{*descendentFontIndRef})

	if subFont {
		toUnicodeIndRef, err := toUnicodeCMap(xRefTable, ttf, fontName, subFont, nil)
		if err != nil {
			return nil, err
		}
		d.Insert("ToUnicode", *toUnicodeIndRef)
	}

	if subFont {
		// Reset used glyph ids.
		delete(xRefTable.UsedGIDs, fontName)
	}

	if indRef == nil {
		return xRefTable.IndRefForNewObject(d)
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	entry.Object = d

	return indRef, nil
}

func trueTypeFontDict(xRefTable *model.XRefTable, fontName, fontLang string) (*types.IndirectRef, error) {
	font.UserFontMetricsLock.RLock()
	ttf, ok := font.UserFontMetrics[fontName]
	font.UserFontMetricsLock.RUnlock()
	if !ok {
		return nil, errors.Errorf("pdfcpu: font %s not available", fontName)
	}

	first, last := 0, 255
	wIndRef, err := Widths(xRefTable, ttf, first, last)
	if err != nil {
		return nil, err
	}

	fdIndRef, err := NewFontDescriptor(xRefTable, ttf, fontName, fontLang)
	if err != nil {
		return nil, err
	}

	d := types.NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "TrueType")
	d.InsertName("BaseFont", fontName)
	d.InsertName("Name", fontName)
	d.InsertName("Encoding", "WinAnsiEncoding")
	d.InsertInt("FirstChar", first)
	d.InsertInt("LastChar", last)
	d.Insert("Widths", *wIndRef)
	d.Insert("FontDescriptor", *fdIndRef)

	return xRefTable.IndRefForNewObject(d)
}

// CJK returns true if script and lang imply a CJK font.
func CJK(script, lang string) bool {
	if script != "" {
		_, ok := cjkParms[script]
		return ok
	}
	return types.MemberOf(lang, []string{"ja", "ko", "zh"})
}

// RTL returns true if lang implies a right-to-left script.
func RTL(lang string) bool {
	return types.MemberOf(lang, []string{"ar", "fa", "he", "ur"})
}

// EnsureFontDict ensures a font dict for fontName, lang, script.
func EnsureFontDict(xRefTable *model.XRefTable, fontName, lang, script string, field bool, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	if font.IsCoreFont(fontName) {
		if indRef != nil {
			return indRef, nil
		}
		return coreFontDict(xRefTable, fontName)
	}
	if field && (script == "" || !CJK(script, lang)) {
		return trueTypeFontDict(xRefTable, fontName, lang)
	}
	return type0FontDict(xRefTable, fontName, lang, script, indRef)
}

// FontResources returns a font resource dict for a font map.
func FontResources(xRefTable *model.XRefTable, fm model.FontMap) (types.Dict, error) {
	d := types.Dict{}

	for fontName, font := range fm {
		ir, err := EnsureFontDict(xRefTable, fontName, "", "", false, nil)
		if err != nil {
			return nil, err
		}
		d.Insert(font.Res.ID, *ir)
	}

	return d, nil
}

// Name evaluates the font name for a given font dict.
func Name(xRefTable *model.XRefTable, fontDict types.Dict, objNumber int) (prefix, fontName string, err error) {
	var found bool
	var o types.Object

	if *fontDict.Subtype() != "Type3" {

		o, found = fontDict.Find("BaseFont")
		if !found {
			o, found = fontDict.Find("Name")
			if !found {
				return "", "", errors.New("pdfcpu: fontName: missing fontDict entries \"BaseFont\" and \"Name\"")
			}
		}

	} else {

		// Type3 fonts only have Name in V1.0 else use generic name.

		o, found = fontDict.Find("Name")
		if !found {
			return "", fmt.Sprintf("Type3_%d", objNumber), nil
		}

	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return "", "", err
	}

	baseFont, ok := o.(types.Name)
	if !ok {
		return "", "", errors.New("pdfcpu: fontName: corrupt fontDict entry BaseFont")
	}

	n := string(baseFont)

	// Isolate Postscript prefix.
	var p string

	i := strings.Index(n, "+")

	if i > 0 {
		p = n[:i]
		n = n[i+1:]
	}

	return p, n, nil
}

// Lang detects the optional language indicator in a font dict.
func Lang(xRefTable *model.XRefTable, fontDict types.Dict) (string, error) {
	o, found := fontDict.Find("FontDescriptor")
	if found {
		fd, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return "", err
		}
		var s string
		n := fd.NameEntry("Lang")
		if n != nil {
			s = *n
		}
		return s, nil
	}

	o, found = fontDict.Find("DescendantFonts")
	if !found {
		return "", ErrCorruptFontDict
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return "", err
	}

	if len(arr) != 1 {
		return "", ErrCorruptFontDict
	}

	d1, err := xRefTable.DereferenceDict(arr[0])
	if err != nil {
		return "", err
	}
	o, found = d1.Find("FontDescriptor")
	if found {
		fd, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return "", err
		}
		var s string
		n := fd.NameEntry("Lang")
		if n != nil {
			s = *n
		}
		return s, nil
	}

	return "", nil
}

func trivialFontDescriptor(xRefTable *model.XRefTable, fontDict types.Dict, objNr int) (types.Dict, error) {
	o, ok := fontDict.Find("FontDescriptor")
	if !ok {
		return nil, nil
	}

	// fontDescriptor directly available.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	if d == nil {
		return nil, errors.Errorf("pdfcpu: trivialFontDescriptor: FontDescriptor is null for font object %d\n", objNr)
	}

	if d.Type() != nil && *d.Type() != "FontDescriptor" {
		return nil, errors.Errorf("pdfcpu: trivialFontDescriptor: FontDescriptor dict incorrect dict type for font object %d\n", objNr)
	}

	return d, nil
}

// FontDescriptor gets the font descriptor for this font.
func FontDescriptor(xRefTable *model.XRefTable, fontDict types.Dict, objNr int) (types.Dict, error) {
	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptor begin")
	}

	d, err := trivialFontDescriptor(xRefTable, fontDict, objNr)
	if err != nil {
		return nil, err
	}
	if d != nil {
		return d, nil
	}

	// Try to access a fontDescriptor in a Descendent font for Type0 fonts.

	o, ok := fontDict.Find("DescendantFonts")
	if !ok {
		//logErrorOptimize.Printf("FontDescriptor: Neither FontDescriptor nor DescendantFonts for font object %d\n", objectNumber)
		return nil, nil
	}

	// A descendant font is contained in an array of size 1.

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: DescendantFonts: IndirectRef or Array with length 1 expected for font object %d\n", objNr)
	}
	if len(a) != 1 {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: DescendantFonts Array length <> 1 %v\n", a)
	}

	// dict is the fontDict of the descendant font.
	d, err = xRefTable.DereferenceDict(a[0])
	if err != nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: No descendant font dict for %v\n", a)
	}
	if d == nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: descendant font dict is null for %v\n", a)
	}

	if *d.Type() != "Font" {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: font dict with incorrect dict type for %v\n", d)
	}

	o, ok = d.Find("FontDescriptor")
	if !ok {
		log.Optimize.Printf("fontDescriptor: descendant font not embedded %s\n", d)
		return nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: No FontDescriptor dict for font object %d\n", objNr)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptor end")
	}

	return d, nil
}

func Embedded(xRefTable *model.XRefTable, fontDict types.Dict, objNr int) (bool, error) {
	fd, err := FontDescriptor(xRefTable, fontDict, objNr)
	if err != nil {
		return false, err
	}
	if _, ok := fd.Find("FontFile"); ok {
		return true, nil
	}
	if _, ok := fd.Find("FontFile2"); ok {
		return true, nil
	}
	if _, ok := fd.Find("FontFile3"); ok {
		return true, nil
	}
	return false, nil
}

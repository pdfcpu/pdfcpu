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
	"errors"
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

// SupportedScript returns true if s is a supported script.
func SupportedScript(s string) bool {
	return types.MemberOf(s, []string{"HANS", "HANT", "HIRA", "KANA", "JPAN", "HANG", "KORE"})
}

// CJKEncoding returns true for supported encodings.
func CJKEncoding(s string) bool {
	return types.MemberOf(s, []string{"UniGB-UTF16-H", "UniCNS-UTF16-H", "UniJIS-UTF16-H", "UniKS-UTF16-H"})
}

// ScriptForEncoding returns the script for encoding.
func ScriptForEncoding(enc string) string {
	for k, v := range cjkParms {
		if v.encoding == enc {
			return k
		}
	}
	return ""
}

func fontDescriptorIndRefs(fd types.Dict, lang, fontName string, font *model.FontResource) error {
	phase := "inspect user font references"
	if lang != "" {
		if s := fd.NameEntry("Lang"); s != nil {
			if strings.ToLower(*s) != lang {
				return fmt.Errorf("font %s: inspect font descriptor references: %w: language mismatch", fontName, ErrCorruptFontDict)
			}
		}
	}

	font.CIDSet = fd.IndirectRefEntry("CIDSet")
	if font.CIDSet == nil {
		return missingUserfontReference(fontName, phase, "CIDSet")
	}

	font.FontFile = fd.IndirectRefEntry("FontFile2")
	if font.FontFile == nil {
		return missingUserfontReference(fontName, phase, "FontFile2")
	}

	return nil
}

func fontDictName(d types.Dict) string {
	if name := d.NameEntry("Name"); name != nil {
		return *name
	}
	if name := d.NameEntry("BaseFont"); name != nil {
		return *name
	}
	return "<unknown>"
}

// IndRefsForUserfontUpdate detects used indirect references for a possible user font update.
func IndRefsForUserfontUpdate(xRefTable *model.XRefTable, d types.Dict, lang string, font *model.FontResource) error {
	fontName := fontDictName(d)
	phase := "inspect user font references"
	if err := requireFontXRef(xRefTable, fontName, phase); err != nil {
		return err
	}
	if font == nil {
		return fmt.Errorf("%s: %w: missing font resource destination", fontPhase(fontName, phase), ErrCorruptFontDict)
	}
	if enc := d.NameEntry("Encoding"); enc == nil || *enc != "Identity-H" {
		return fmt.Errorf("%s: %w: invalid encoding", fontPhase(fontName, phase), ErrCorruptFontDict)
	}

	// TODO some indRefs may be direct objs => don't reuse userFont.

	font.ToUnicode = d.IndirectRefEntry("ToUnicode")
	if font.ToUnicode == nil {
		return fmt.Errorf("%s: %w: missing ToUnicode reference", fontPhase(fontName, phase), ErrCorruptFontDict)
	}

	o, found := d.Find("DescendantFonts")
	if !found {
		return fmt.Errorf("%s: %w: missing descendant fonts", fontPhase(fontName, phase), ErrCorruptFontDict)
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("%s: dereference descendant fonts: %w", fontPhase(fontName, phase), err)
	}

	if len(a) != 1 {
		return fmt.Errorf("%s: %w: expected one descendant font", fontPhase(fontName, phase), ErrCorruptFontDict)
	}

	df, err := xRefTable.DereferenceDict(a[0])
	if err != nil {
		return fmt.Errorf("%s: dereference descendant font dictionary: %w", fontPhase(fontName, phase), err)
	}

	font.W = df.IndirectRefEntry("W")
	if font.W == nil {
		return missingUserfontReference(fontName, phase, "W")
	}

	o, found = df.Find("FontDescriptor")
	if !found {
		return fmt.Errorf("%s: %w: missing font descriptor", fontPhase(fontName, phase), ErrCorruptFontDict)
	}

	fd, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("%s: dereference font descriptor: %w", fontPhase(fontName, phase), err)
	}

	if err := fontDescriptorIndRefs(fd, lang, fontName, font); err != nil {
		return err
	}
	return nil
}

func fontPhase(fontName, phase string) string {
	if fontName == "" {
		return "font: " + phase
	}
	return fmt.Sprintf("font %s: %s", fontName, phase)
}

func missingUserfontReference(fontName, phase, name string) error {
	return fmt.Errorf("%s: %w: missing %s reference", fontPhase(fontName, phase), ErrCorruptFontDict, name)
}

func requireFontXRef(xRefTable *model.XRefTable, fontName, phase string) error {
	if xRefTable == nil {
		return fmt.Errorf("%s: %w", fontPhase(fontName, phase), model.ErrMissingXRefTable)
	}
	return nil
}

func insertFontObject(xRefTable *model.XRefTable, fontName, phase string, obj types.Object) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, fontName, phase); err != nil {
		return nil, err
	}
	indRef, err := xRefTable.IndRefForNewObject(obj)
	if err != nil {
		return nil, fmt.Errorf("%s: insert object: %w", fontPhase(fontName, phase), err)
	}
	return indRef, nil
}

func ttfFontName(ttf font.TTFLight, fontName string) string {
	if fontName != "" {
		return fontName
	}
	if ttf.PostscriptName != "" {
		return ttf.PostscriptName
	}
	return "<unknown>"
}

func validateEmbeddingMetrics(ttf font.TTFLight, fontName, phase string) error {
	fontName = ttfFontName(ttf, fontName)
	if err := font.ValidateTTFLight(ttf); err != nil {
		return fmt.Errorf("%s: %w", fontPhase(fontName, phase), err)
	}
	return nil
}

func flateEncodedStreamIndRef(xRefTable *model.XRefTable, fontName, phase string, data []byte) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, fontName, phase); err != nil {
		return nil, err
	}
	sd, err := xRefTable.NewStreamDictForBuf(data)
	if err != nil {
		return nil, fmt.Errorf("%s: create stream dictionary: %w", fontPhase(fontName, phase), err)
	}
	sd.InsertInt("Length1", len(data))
	if err := sd.Encode(); err != nil {
		return nil, fmt.Errorf("%s: encode stream: %w", fontPhase(fontName, phase), err)
	}
	return insertFontObject(xRefTable, fontName, phase, *sd)
}

func ttfFontFile(xRefTable *model.XRefTable, fontName string) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, fontName, "embed font file"); err != nil {
		return nil, err
	}
	bb, err := font.Read(fontName)
	if err != nil {
		return nil, fmt.Errorf("embed font %s: read installed font: %w", fontName, err)
	}
	indRef, err := flateEncodedStreamIndRef(xRefTable, fontName, "embed font file", bb)
	if err != nil {
		return nil, err
	}
	return indRef, nil
}

type referencedFontObjectType uint8

const (
	fontStreamObject referencedFontObjectType = iota
	fontArrayObject
	fontDictObject
)

func referencedFontObject(xRefTable *model.XRefTable, fontName, phase string, indRef *types.IndirectRef, expected referencedFontObjectType) (*model.XRefTableEntry, types.Object, error) {
	if xRefTable == nil {
		return nil, nil, fmt.Errorf("font %s: %s: %w: %w", fontName, phase, ErrCorruptFontDict, model.ErrMissingXRefTable)
	}
	if indRef == nil {
		return nil, nil, fmt.Errorf("font %s: %s: %w: missing indirect reference", fontName, phase, ErrCorruptFontDict)
	}
	entry, found := xRefTable.FindTableEntryForIndRef(indRef)
	if !found || entry == nil {
		return nil, nil, fmt.Errorf("font %s: %s: %w: missing xref entry for %v", fontName, phase, ErrCorruptFontDict, indRef)
	}
	valid := false
	expectedName := ""
	switch expected {
	case fontStreamObject:
		_, valid = entry.Object.(types.StreamDict)
		expectedName = "stream dictionary"
	case fontArrayObject:
		_, valid = entry.Object.(types.Array)
		expectedName = "array"
	case fontDictObject:
		_, valid = entry.Object.(types.Dict)
		expectedName = "dictionary"
	}
	if !valid {
		return nil, nil, fmt.Errorf("font %s: %s: %w: xref entry for %v has type %T, expected %s", fontName, phase, ErrCorruptFontDict, indRef, entry.Object, expectedName)
	}
	return entry, entry.Object, nil
}

func ttfSubFontFile(xRefTable *model.XRefTable, fontName string, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	if xRefTable == nil {
		return nil, fmt.Errorf("font %s: update subset stream: %w: %w", fontName, ErrCorruptFontDict, model.ErrMissingXRefTable)
	}
	var entry *model.XRefTableEntry
	var sd types.StreamDict
	if indRef != nil {
		var obj types.Object
		var err error
		entry, obj, err = referencedFontObject(xRefTable, fontName, "update subset stream", indRef, fontStreamObject)
		if err != nil {
			return nil, err
		}
		sd = obj.(types.StreamDict)
	}
	bb, err := font.Subset(fontName, xRefTable.UsedGIDs[fontName])
	if err != nil {
		return nil, fmt.Errorf("embed subset font: %w", err)
	}
	if indRef == nil {
		indRef, err := flateEncodedStreamIndRef(xRefTable, fontName, "embed subset stream", bb)
		if err != nil {
			return nil, err
		}
		return indRef, nil
	}
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, fmt.Errorf("embed subset font %s: encode stream: %w", fontName, err)
	}
	entry.Object = sd
	return indRef, nil
}

// PDFDocEncoding returns an indirect reference to a new PDF doc encoding dict.
func PDFDocEncoding(xRefTable *model.XRefTable) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, "", "create PDFDoc encoding"); err != nil {
		return nil, err
	}
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

	return insertFontObject(xRefTable, "", "create PDFDoc encoding", d)
}

// CoreFontDict returns an indirect reference to a Type 1 font.
func CoreFontDict(xRefTable *model.XRefTable, coreFontName string) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, coreFontName, "create core font dictionary"); err != nil {
		return nil, err
	}
	d := types.NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", coreFontName)
	if coreFontName != "Symbol" && coreFontName != "ZapfDingbats" {
		d.InsertName("Encoding", "WinAnsiEncoding")
	}
	return insertFontObject(xRefTable, coreFontName, "create core font dictionary", d)
}

// CIDSet computes a CIDSet for used glyphs and updates or returns a new object.
func CIDSet(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	if xRefTable == nil {
		return nil, fmt.Errorf("font %s: update CID set: %w: %w", fontName, ErrCorruptFontDict, model.ErrMissingXRefTable)
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "update CID set"); err != nil {
		return nil, err
	}
	usedGIDs, ok := xRefTable.UsedGIDs[fontName]
	if ok {
		for gid := range usedGIDs {
			if int(gid) >= ttf.GlyphCount {
				return nil, fmt.Errorf("font %s: update CID set: %w: glyph ID %d outside 0..%d", fontName, ErrCorruptFontDict, gid, ttf.GlyphCount-1)
			}
		}
	}
	bb := make([]byte, ttf.GlyphCount/8+1)
	if ok {
		for gid := range usedGIDs {
			bb[gid/8] |= 1 << (7 - (gid % 8))
		}
	}
	if indRef == nil {
		return flateEncodedStreamIndRef(xRefTable, fontName, "create CID set", bb)
	}
	entry, obj, err := referencedFontObject(xRefTable, fontName, "update CID set", indRef, fontStreamObject)
	if err != nil {
		return nil, err
	}
	sd := obj.(types.StreamDict)
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, fmt.Errorf("font %s: update CID set: encode stream: %w", fontName, err)
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
	if err := requireFontXRef(xRefTable, fontName, "create CID font file"); err != nil {
		return nil, err
	}
	if subFont {
		indRef, err := ttfSubFontFile(xRefTable, fontName, nil)
		if err != nil {
			return nil, fmt.Errorf("font %s: create CID font file: subset: %w", fontName, err)
		}
		return indRef, nil
	}
	indRef, err := ttfFontFile(xRefTable, fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: create CID font file: embed: %w", fontName, err)
	}
	return indRef, nil
}

// CIDFontDescriptor returns a font descriptor describing the CIDFont’s default metrics other than its glyph widths.
func CIDFontDescriptor(xRefTable *model.XRefTable, ttf font.TTFLight, fontName, baseFontName, fontLang string, embed bool) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, fontName, "create CID font descriptor"); err != nil {
		return nil, err
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "create CID font descriptor"); err != nil {
		return nil, err
	}
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
			return nil, fmt.Errorf("font %s: create CID font descriptor: font file: %w", fontName, err)
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
			return nil, fmt.Errorf("font %s: create CID font descriptor: CID set: %w", fontName, err)
		}
		d["CIDSet"] = *cidSetIndRef
	}

	if fontLang != "" {
		d["Lang"] = types.Name(fontLang)
	}

	return insertFontObject(xRefTable, fontName, "create CID font descriptor", d)
}

// NewFontDescriptor returns a TrueType font descriptor describing font’s default metrics other than its glyph widths.
func NewFontDescriptor(xRefTable *model.XRefTable, ttf font.TTFLight, fontName, fontLang string) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, fontName, "create TrueType font descriptor"); err != nil {
		return nil, err
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "create TrueType font descriptor"); err != nil {
		return nil, err
	}
	fontFile, err := ttfFontFile(xRefTable, fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: create TrueType font descriptor: font file: %w", fontName, err)
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

	return insertFontObject(xRefTable, fontName, "create TrueType font descriptor", d)
}

func wArr(ttf font.TTFLight, from, thru int) types.Array {
	a := types.Array{}
	for i := from; i <= thru; i++ {
		a = append(a, types.Integer(ttf.GlyphWidths[i]))
	}
	return a
}

func prepGids(xRefTable *model.XRefTable, ttf font.TTFLight, fontName string, used bool) ([]int, bool) {
	gids := ttf.GlyphWidths
	if used {
		usedGIDs, ok := xRefTable.UsedGIDs[fontName]
		if ok {
			gids = make([]int, 0, len(usedGIDs))
			for gid := range usedGIDs {
				gids = append(gids, int(gid))
			}
			sort.Ints(gids)
			return gids, true
		}
	}
	return gids, false
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
	gids, ok := prepGids(xRefTable, ttf, fontName, used)
	a := types.Array{}
	var g0, w0, gl, wl int
	start, equalWidths, skip := true, false, false

	for g, w := range gids {
		if ok {
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
				tru := max(gl-1, g0)
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
	if xRefTable == nil {
		return nil, fmt.Errorf("font %s: update CID widths: %w: %w", fontName, ErrCorruptFontDict, model.ErrMissingXRefTable)
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "update CID widths"); err != nil {
		return nil, err
	}
	if subFont {
		for gid := range xRefTable.UsedGIDs[fontName] {
			if int(gid) >= ttf.GlyphCount || int(gid) >= len(ttf.GlyphWidths) {
				return nil, fmt.Errorf("font %s: update CID widths: %w: glyph ID %d outside available widths", fontName, ErrCorruptFontDict, gid)
			}
		}
	}
	a := calcWidthArray(xRefTable, ttf, fontName, subFont)
	if len(a) == 0 {
		return nil, nil
	}

	if indRef == nil {
		return insertFontObject(xRefTable, fontName, "create CID widths", a)
	}

	entry, _, err := referencedFontObject(xRefTable, fontName, "update CID widths", indRef, fontArrayObject)
	if err != nil {
		return nil, err
	}
	entry.Object = a

	return indRef, nil
}

// Widths returns the value for Widths in a TrueType FontDict.
func Widths(xRefTable *model.XRefTable, ttf font.TTFLight, first, last int) (*types.IndirectRef, error) {
	fontName := ttfFontName(ttf, "")
	if err := requireFontXRef(xRefTable, fontName, "create TrueType widths"); err != nil {
		return nil, err
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "create TrueType widths"); err != nil {
		return nil, err
	}
	a := types.Array{}
	for i := first; i < last; i++ {
		pos, ok := ttf.Chars[uint32(i)]
		if !ok {
			pos = 0 // should be the "invalid char"
		}
		a = append(a, types.Integer(ttf.GlyphWidths[pos]))
	}
	return insertFontObject(xRefTable, fontName, "create TrueType widths", a)
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

	if xRefTable == nil {
		return nil, fmt.Errorf("font %s: update ToUnicode map: %w: %w", fontName, ErrCorruptFontDict, model.ErrMissingXRefTable)
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "update ToUnicode map"); err != nil {
		return nil, err
	}
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
		return flateEncodedStreamIndRef(xRefTable, fontName, "create ToUnicode map", bb)
	}

	entry, obj, err := referencedFontObject(xRefTable, fontName, "update ToUnicode map", indRef, fontStreamObject)
	if err != nil {
		return nil, err
	}
	sd := obj.(types.StreamDict)
	sd.Content = bb
	sd.InsertInt("Length1", len(bb))
	if err := sd.Encode(); err != nil {
		return nil, fmt.Errorf("font %s: update ToUnicode map: encode stream: %w", fontName, err)
	}
	entry.Object = sd

	return indRef, nil
}

var (
	errCorruptCMap = errors.New("corrupt CMap")

	// ErrCorruptFontDict signals a malformed font dictionary.
	ErrCorruptFontDict = errors.New("corrupt fontDict")
)

const cmapCodeSpaceEnd = "endcodespacerange"

func scanCMapLine(scanner *bufio.Scanner, phase string) (string, error) {
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("%w: %s: %w", errCorruptCMap, phase, err)
	}
	return "", fmt.Errorf("%w: %s: unexpected EOF", errCorruptCMap, phase)
}

func cmapBFCharCount(line string) (int, error) {
	tokens := strings.Fields(line)
	if len(tokens) != 2 || tokens[1] != "beginbfchar" {
		return 0, fmt.Errorf("%w: invalid beginbfchar header", errCorruptCMap)
	}
	count, err := strconv.Atoi(tokens[0])
	if err != nil || count < 0 {
		return 0, fmt.Errorf("%w: invalid beginbfchar count", errCorruptCMap)
	}
	return count, nil
}

func cmapGID(line string) (uint16, error) {
	tokens := strings.Fields(line)
	if len(tokens) != 2 {
		return 0, fmt.Errorf("%w: invalid bfchar token count", errCorruptCMap)
	}
	src := tokens[0]
	if len(src) != 6 || src[0] != '<' || src[len(src)-1] != '>' {
		return 0, fmt.Errorf("%w: invalid bfchar source", errCorruptCMap)
	}
	bb, err := hex.DecodeString(src[1 : len(src)-1])
	if err != nil || len(bb) != 2 {
		return 0, fmt.Errorf("%w: invalid bfchar source code", errCorruptCMap)
	}
	dst := tokens[1]
	if len(dst) < 2 || dst[0] != '<' || dst[len(dst)-1] != '>' || (len(dst)-2)%2 != 0 {
		return 0, fmt.Errorf("%w: invalid bfchar destination", errCorruptCMap)
	}
	if _, err := hex.DecodeString(dst[1 : len(dst)-1]); err != nil {
		return 0, fmt.Errorf("%w: invalid bfchar destination code", errCorruptCMap)
	}
	return binary.BigEndian.Uint16(bb), nil
}

func cmapSingleToken(line, want string) error {
	tokens := strings.Fields(line)
	if len(tokens) != 1 || tokens[0] != want {
		return fmt.Errorf("%w: expected %s", errCorruptCMap, want)
	}
	return nil
}

func usedGIDsFromCMap(cMap string) ([]uint16, error) {
	i := strings.Index(cMap, cmapCodeSpaceEnd)
	if i < 0 {
		return nil, errCorruptCMap
	}
	start := i + len(cmapCodeSpaceEnd)
	remainder := strings.TrimLeft(cMap[start:], "\r\n")
	scanner := bufio.NewScanner(strings.NewReader(remainder))
	header, err := scanCMapLine(scanner, "read beginbfchar header")
	if err != nil {
		return nil, err
	}
	gids := []uint16{}
	for {
		count, err := cmapBFCharCount(header)
		if err != nil {
			return nil, err
		}
		for i := 0; i < count; i++ {
			line, err := scanCMapLine(scanner, "read bfchar mapping")
			if err != nil {
				return nil, err
			}
			gid, err := cmapGID(line)
			if err != nil {
				return nil, err
			}
			gids = append(gids, gid)
		}
		line, err := scanCMapLine(scanner, "read endbfchar")
		if err != nil {
			return nil, err
		}
		if err := cmapSingleToken(line, "endbfchar"); err != nil {
			return nil, err
		}
		header, err = scanCMapLine(scanner, "read next CMap block")
		if err != nil {
			return nil, err
		}
		if cmapSingleToken(header, "endcmap") == nil {
			return gids, nil
		}
		if count < 100 {
			return nil, fmt.Errorf("%w: unexpected bfchar block", errCorruptCMap)
		}
	}
}

func validateUserfontUpdateReferences(fontName string, f model.FontResource) error {
	required := []struct {
		name   string
		indRef *types.IndirectRef
	}{
		{"ToUnicode", f.ToUnicode},
		{"FontFile", f.FontFile},
		{"W", f.W},
		{"CIDSet", f.CIDSet},
	}
	for _, ref := range required {
		if ref.indRef == nil {
			return missingUserfontReference(fontName, "update user font", ref.name)
		}
	}
	return nil
}

// UpdateUserfont updates the fontdict for fontName via supplied font resource.
func UpdateUserfont(xRefTable *model.XRefTable, fontName string, f model.FontResource) error {
	if err := requireFontXRef(xRefTable, fontName, "update user font"); err != nil {
		return err
	}
	ttf, ok, err := font.UserFont(fontName)
	if err != nil {
		return fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return fmt.Errorf("userfont %s not available: %w", fontName, font.ErrUnknownFont)
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "update user font"); err != nil {
		return err
	}
	if err := validateUserfontUpdateReferences(fontName, f); err != nil {
		return err
	}

	if err := usedGIDsFromCMapIndRef(xRefTable, fontName, *f.ToUnicode); err != nil {
		return fmt.Errorf("font %s: read used glyphs: %w", fontName, err)
	}

	if _, err := toUnicodeCMap(xRefTable, ttf, fontName, true, f.ToUnicode); err != nil {
		return fmt.Errorf("font %s: update ToUnicode CMap: %w", fontName, err)
	}

	if _, err := ttfSubFontFile(xRefTable, fontName, f.FontFile); err != nil {
		return fmt.Errorf("font %s: update font file: %w", fontName, err)
	}

	if _, err := CIDWidths(xRefTable, ttf, fontName, true, f.W); err != nil {
		return fmt.Errorf("font %s: update CID widths: %w", fontName, err)
	}

	if _, err := CIDSet(xRefTable, ttf, fontName, f.CIDSet); err != nil {
		return fmt.Errorf("font %s: update CID set: %w", fontName, err)
	}

	return nil
}

// UpdateUserfonts updates referenced fonts.
func UpdateUserfonts(xRefTable *model.XRefTable, fonts map[string]types.IndirectRef) error {
	if xRefTable == nil {
		return model.ErrMissingXRefTable
	}
	fontNames := make([]string, 0, len(fonts))
	for fontName := range fonts {
		fontNames = append(fontNames, fontName)
	}
	sort.Strings(fontNames)
	for _, fName := range fontNames {
		indRef := fonts[fName]

		if len(xRefTable.UsedGIDs[fName]) == 0 {
			continue
		}

		fDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return fmt.Errorf("font %s: dereference font dictionary: %w", fName, err)
		}
		if fDict == nil {
			return fmt.Errorf("font %s: %w: missing font dictionary", fName, ErrCorruptFontDict)
		}

		fr := model.FontResource{}
		if err := IndRefsForUserfontUpdate(xRefTable, fDict, "", &fr); err != nil {
			if errors.Is(err, ErrCorruptFontDict) {
				return fmt.Errorf("font %s: inspect font dictionary: %w", fName, err)
			}
			return fmt.Errorf("font %s: inspect font dictionary: %w: %w", fName, ErrCorruptFontDict, err)
		}

		if err := UpdateUserfont(xRefTable, fName, fr); err != nil {
			return fmt.Errorf("finalize font %s: %w", fName, err)
		}
	}

	return nil
}

func usedGIDsFromCMapIndRef(xRefTable *model.XRefTable, fontName string, cmapIndRef types.IndirectRef) error {
	sd, _, err := xRefTable.DereferenceStreamDict(cmapIndRef)
	if err != nil {
		return fmt.Errorf("dereference ToUnicode CMap: %w", err)
	}
	if sd == nil {
		return errors.New("missing ToUnicode CMap stream")
	}
	if err := sd.Decode(); err != nil {
		return fmt.Errorf("decode ToUnicode CMap: %w", err)
	}
	gids, err := usedGIDsFromCMap(string(sd.Content))
	if err != nil {
		return fmt.Errorf("parse ToUnicode CMap: %w", err)
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
	if err := requireFontXRef(xRefTable, fontName, "create CID font dictionary"); err != nil {
		return nil, err
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "create CID font dictionary"); err != nil {
		return nil, err
	}
	fdIndRef, err := CIDFontDescriptor(xRefTable, ttf, fontName, baseFontName, lang, parms == nil)
	if err != nil {
		return nil, fmt.Errorf("font %s: create CID font dictionary: descriptor: %w", fontName, err)
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
		wIndRef, err := CIDWidths(xRefTable, ttf, fontName, true, nil)
		if err != nil {
			return nil, fmt.Errorf("font %s: create CID font dictionary: widths: %w", fontName, err)
		}
		if wIndRef != nil {
			d["W"] = *wIndRef
		}
	}

	return insertFontObject(xRefTable, fontName, "create CID font dictionary", d)
}

func type0FontDictData(xRefTable *model.XRefTable, ttf font.TTFLight, fontName, lang, script string) (types.Dict, bool, error) {
	subFont := script == ""
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
		return nil, false, fmt.Errorf("font %s: create Type0 font dictionary: descendant CID font: %w", fontName, err)
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
			return nil, false, fmt.Errorf("font %s: create Type0 font dictionary: ToUnicode map: %w", fontName, err)
		}
		d.Insert("ToUnicode", *toUnicodeIndRef)
	}
	return d, subFont, nil
}

func type0FontDict(xRefTable *model.XRefTable, fontName, lang, script string, indRef *types.IndirectRef) (*types.IndirectRef, error) {
	if xRefTable == nil {
		return nil, fmt.Errorf("font %s: update Type0 font: %w: %w", fontName, ErrCorruptFontDict, model.ErrMissingXRefTable)
	}
	var updateEntry *model.XRefTableEntry
	if indRef != nil {
		var err error
		updateEntry, _, err = referencedFontObject(xRefTable, fontName, "update Type0 font", indRef, fontDictObject)
		if err != nil {
			return nil, err
		}
	}
	ttf, ok, err := font.UserFont(fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return nil, fmt.Errorf("font %s not available: %w", fontName, font.ErrUnknownFont)
	}
	if indRef != nil && script == "" && !xRefTable.HasUsedGIDs(fontName) {
		return indRef, nil
	}
	d, subFont, err := type0FontDictData(xRefTable, ttf, fontName, lang, script)
	if err != nil {
		return nil, fmt.Errorf("font %s: update Type0 font: build dictionary: %w", fontName, err)
	}

	if subFont {
		// Reset used glyph ids.
		delete(xRefTable.UsedGIDs, fontName)
	}

	if indRef == nil {
		return insertFontObject(xRefTable, fontName, "create Type0 font dictionary", d)
	}

	updateEntry.Object = d

	return indRef, nil
}

func trueTypeFontDict(xRefTable *model.XRefTable, fontName, fontLang string) (*types.IndirectRef, error) {
	if err := requireFontXRef(xRefTable, fontName, "create TrueType font dictionary"); err != nil {
		return nil, err
	}
	ttf, ok, err := font.UserFont(fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return nil, fmt.Errorf("font %s not available: %w", fontName, font.ErrUnknownFont)
	}
	if err := validateEmbeddingMetrics(ttf, fontName, "create TrueType font dictionary"); err != nil {
		return nil, err
	}

	first, last := 0, 255
	wIndRef, err := Widths(xRefTable, ttf, first, last)
	if err != nil {
		return nil, fmt.Errorf("font %s: create TrueType font dictionary: widths: %w", fontName, err)
	}

	fdIndRef, err := NewFontDescriptor(xRefTable, ttf, fontName, fontLang)
	if err != nil {
		return nil, fmt.Errorf("font %s: create TrueType font dictionary: descriptor: %w", fontName, err)
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

	return insertFontObject(xRefTable, fontName, "create TrueType font dictionary", d)
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
	if err := requireFontXRef(xRefTable, fontName, "ensure font dictionary"); err != nil {
		return nil, err
	}
	// TODO Reuse fontDict
	if font.IsCoreFont(fontName) {
		if indRef != nil {
			return indRef, nil
		}
		ir, err := CoreFontDict(xRefTable, fontName)
		if err != nil {
			return nil, fmt.Errorf("font %s: ensure font dictionary: core font: %w", fontName, err)
		}
		return ir, nil
	}
	if field && (script == "" || !CJK(script, lang)) {
		ir, err := trueTypeFontDict(xRefTable, fontName, lang)
		if err != nil {
			return nil, fmt.Errorf("font %s: ensure font dictionary: TrueType font: %w", fontName, err)
		}
		return ir, nil
	}
	ir, err := type0FontDict(xRefTable, fontName, lang, script, indRef)
	if err != nil {
		return nil, fmt.Errorf("font %s: ensure font dictionary: Type0 font: %w", fontName, err)
	}
	return ir, nil
}

// FontResources returns a font resource dict for a font map.
func FontResources(xRefTable *model.XRefTable, fm model.FontMap) (types.Dict, error) {
	if err := requireFontXRef(xRefTable, "", "create font resources"); err != nil {
		return nil, err
	}
	d := types.Dict{}

	for fontName, font := range fm {
		ir, err := EnsureFontDict(xRefTable, fontName, "", "", false, nil)
		if err != nil {
			return nil, fmt.Errorf("font %s: create font resources: resource %s: %w", fontName, font.Res.ID, err)
		}
		d.Insert(font.Res.ID, *ir)
	}

	return d, nil
}

// Name evaluates the font name for a given font dict.
func Name(xRefTable *model.XRefTable, fontDict types.Dict, objNumber int) (prefix, fontName string, err error) {
	var found bool
	var o types.Object

	subtype := fontDict.Subtype()
	if subtype == nil || len(*subtype) == 0 {
		return "", "", errors.New("fontName: missing fontDict entry \"Subtype\"")
	}

	if *subtype != "Type3" {

		o, found = fontDict.Find("BaseFont")
		if !found {
			o, found = fontDict.Find("Name")
			if !found {
				return "", "", errors.New("fontName: missing fontDict entries \"BaseFont\" and \"Name\"")
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
		return "", "", errors.New("fontName: corrupt fontDict entry BaseFont")
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
		return nil, fmt.Errorf("trivialFontDescriptor: FontDescriptor is null for font object %d", objNr)
	}

	if d.Type() != nil && *d.Type() != "FontDescriptor" {
		return nil, fmt.Errorf("trivialFontDescriptor: FontDescriptor dict incorrect dict type for font object %d", objNr)
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
		return nil, fmt.Errorf("fontDescriptor: DescendantFonts: IndirectRef or Array with length 1 expected for font object %d", objNr)
	}
	if len(a) != 1 {
		return nil, fmt.Errorf("fontDescriptor: DescendantFonts Array length <> 1 %v", a)
	}

	// dict is the fontDict of the descendant font.
	d, err = xRefTable.DereferenceDict(a[0])
	if err != nil {
		return nil, fmt.Errorf("fontDescriptor: No descendant font dict for %v", a)
	}
	if d == nil {
		return nil, fmt.Errorf("fontDescriptor: descendant font dict is null for %v", a)
	}

	dictType := d.Type()
	if dictType == nil {
		return nil, fmt.Errorf("fontDescriptor: descendant font dict missing Type for font object %d", objNr)
	}
	if *dictType != "Font" {
		return nil, fmt.Errorf("fontDescriptor: font dict with incorrect dict type for %v", d)
	}

	o, ok = d.Find("FontDescriptor")
	if !ok {
		log.Optimize.Printf("fontDescriptor: descendant font not embedded %s\n", d)
		return nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, fmt.Errorf("fontDescriptor: No FontDescriptor dict for font object %d", objNr)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptor end")
	}

	return d, nil
}

// Embedded returns true if the font represented by fontDict is embedded.
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

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

package validate

import (
	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateStandardType1Font(s string) bool {

	return pdf.MemberOf(s, []string{"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats"})
}

func validateFontFile3SubType(sd *pdf.StreamDict, fontType string) error {

	// Hint about used font program.
	dictSubType := sd.Subtype()

	if dictSubType == nil {
		return errors.New("pdfcpu: validateFontFile3SubType: missing Subtype")
	}

	switch fontType {
	case "Type1":
		if *dictSubType != "Type1C" && *dictSubType != "OpenType" {
			return errors.Errorf("pdfcpu: validateFontFile3SubType: Type1: unexpected Subtype %s", *dictSubType)
		}

	case "MMType1":
		if *dictSubType != "Type1C" {
			return errors.Errorf("pdfcpu: validateFontFile3SubType: MMType1: unexpected Subtype %s", *dictSubType)
		}

	case "CIDFontType0":
		if *dictSubType != "CIDFontType0C" && *dictSubType != "OpenType" {
			return errors.Errorf("pdfcpu: validateFontFile3SubType: CIDFontType0: unexpected Subtype %s", *dictSubType)
		}

	case "CIDFontType2", "TrueType":
		if *dictSubType != "OpenType" {
			return errors.Errorf("pdfcpu: validateFontFile3SubType: %s: unexpected Subtype %s", fontType, *dictSubType)
		}
	}

	return nil
}

func validateFontFile(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, fontType string, required bool, sinceVersion pdf.Version) error {

	sd, err := validateStreamDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	// Process font file stream dict entries.

	// SubType
	if entryName == "FontFile3" {
		err = validateFontFile3SubType(sd, fontType)
		if err != nil {
			return err
		}

	}

	dName := "fontFileStreamDict"
	compactFontFormat := entryName == "FontFile3"

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dName, "Length1", (fontType == "Type1" || fontType == "TrueType") && !compactFontFormat, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dName, "Length2", fontType == "Type1" && !compactFontFormat, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dName, "Length3", fontType == "Type1" && !compactFontFormat, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Metadata, stream, optional, since 1.4
	return validateMetadata(xRefTable, sd.Dict, OPTIONAL, pdf.V14)
}

func validateFontDescriptorType(xRefTable *pdf.XRefTable, d pdf.Dict) (err error) {

	dictType := d.Type()

	if dictType == nil {

		if xRefTable.ValidationMode == pdf.ValidationRelaxed {
			log.Validate.Println("validateFontDescriptor: missing entry \"Type\"")
		} else {
			return errors.New("pdfcpu: validateFontDescriptor: missing entry \"Type\"")
		}

	}

	if dictType != nil && *dictType != "FontDescriptor" && *dictType != "Font" {
		return errors.New("pdfcpu: validateFontDescriptor: corrupt font descriptor dict")
	}

	return nil
}

func validateFontDescriptorPart1(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, fontDictType string) error {

	err := validateFontDescriptorType(xRefTable, d)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "FontName", REQUIRED, pdf.V10, nil)
	if err != nil {
		_, err = validateStringEntry(xRefTable, d, dictName, "FontName", REQUIRED, pdf.V10, nil)
		if err != nil {
			return err
		}
	}

	sinceVersion := pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "FontFamily", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	sinceVersion = pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "FontStretch", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	sinceVersion = pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateNumberEntry(xRefTable, d, dictName, "FontWeight", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, d, dictName, "Flags", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, d, dictName, "FontBBox", fontDictType != "Type3", pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "ItalicAngle", REQUIRED, pdf.V10, nil)

	return err
}

func validateFontDescriptorPart2(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, fontDictType string) error {

	_, err := validateNumberEntry(xRefTable, d, dictName, "Ascent", fontDictType != "Type3", pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Descent", fontDictType != "Type3", pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Leading", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "CapHeight", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "XHeight", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	required := fontDictType != "Type3"
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = false
	}
	_, err = validateNumberEntry(xRefTable, d, dictName, "StemV", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "StemH", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "AvgWidth", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "MaxWidth", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "MissingWidth", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	err = validateFontDescriptorFontFile(xRefTable, d, dictName, fontDictType)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, dictName, "CharSet", OPTIONAL, pdf.V11, nil)

	return err
}

func validateFontDescriptorFontFile(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, fontDictType string) (err error) {

	switch fontDictType {

	case "Type1", "MMType1":

		err = validateFontFile(xRefTable, d, dictName, "FontFile", fontDictType, OPTIONAL, pdf.V10)
		if err != nil {
			return err
		}

		err = validateFontFile(xRefTable, d, dictName, "FontFile3", fontDictType, OPTIONAL, pdf.V12)

	case "TrueType", "CIDFontType2":
		err = validateFontFile(xRefTable, d, dictName, "FontFile2", fontDictType, OPTIONAL, pdf.V11)

	case "CIDFontType0":
		err = validateFontFile(xRefTable, d, dictName, "FontFile3", fontDictType, OPTIONAL, pdf.V13)

	case "Type3": // No fontfile.

	default:
		return errors.Errorf("pdfcpu: unknown fontDictType: %s\n", fontDictType)

	}

	return err
}

func validateFontDescriptor(xRefTable *pdf.XRefTable, d pdf.Dict, fontDictName string, fontDictType string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, fontDictName, "FontDescriptor", required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName := "fdDict"

	// Process font descriptor dict

	err = validateFontDescriptorPart1(xRefTable, d1, dictName, fontDictType)
	if err != nil {
		return err
	}

	err = validateFontDescriptorPart2(xRefTable, d1, dictName, fontDictType)
	if err != nil {
		return err
	}

	if fontDictType == "CIDFontType0" || fontDictType == "CIDFontType2" {

		validateStyleDict := func(d pdf.Dict) bool {

			// see 9.8.3.2

			if d.Len() != 1 {
				return false
			}

			_, found := d.Find("Panose")

			return found
		}

		// Style, optional, dict
		_, err = validateDictEntry(xRefTable, d1, dictName, "Style", OPTIONAL, pdf.V10, validateStyleDict)
		if err != nil {
			return err
		}

		// Lang, optional, name
		sinceVersion := pdf.V15
		if xRefTable.ValidationMode == pdf.ValidationRelaxed {
			sinceVersion = pdf.V14
		}
		_, err = validateNameEntry(xRefTable, d1, dictName, "Lang", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}

		// FD, optional, dict
		_, err = validateDictEntry(xRefTable, d1, dictName, "FD", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return err
		}

		// CIDSet, optional, stream
		_, err = validateStreamDictEntry(xRefTable, d1, dictName, "CIDSet", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateFontEncoding(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool) error {

	entryName := "Encoding"

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, pdf.V10)
	if err != nil || o == nil {
		return err
	}

	encodings := []string{"MacRomanEncoding", "MacExpertEncoding", "WinAnsiEncoding"}
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		encodings = append(encodings, "StandardEncoding", "SymbolSetEncoding")
	}

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		validateFontEncodingName := func(s string) bool {
			return pdf.MemberOf(s, encodings)
		}
		if !validateFontEncodingName(s) {
			return errors.Errorf("validateFontEncoding: invalid Encoding name: %s\n", s)
		}

	case pdf.Dict:
		// no further processing

	default:
		return errors.Errorf("validateFontEncoding: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateTrueTypeFontDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 9.6.3
	dictName := "trueTypeFontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	_, err := validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FirstChar, required, integer
	required := REQUIRED
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, d, dictName, "FirstChar", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	// LastChar, required, integer
	required = REQUIRED
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, d, dictName, "LastChar", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Widths, array of numbers.
	required = REQUIRED
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Widths", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required, dictionary
	required = REQUIRED
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = OPTIONAL
	}
	err = validateFontDescriptor(xRefTable, d, dictName, "TrueType", required, pdf.V10)
	if err != nil {
		return err
	}

	// Encoding, optional, name or dict
	err = validateFontEncoding(xRefTable, d, dictName, OPTIONAL)
	if err != nil {
		return err
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, pdf.V12, nil)

	return err
}

func validateCIDToGIDMap(xRefTable *pdf.XRefTable, o pdf.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if s != "Identity" {
			return errors.Errorf("pdfcpu: validateCIDToGIDMap: invalid name: %s - must be \"Identity\"\n", s)
		}

	case pdf.StreamDict:
		// no further processing

	default:
		return errors.New("pdfcpu: validateCIDToGIDMap: corrupt entry")

	}

	return nil
}

func validateCIDFontGlyphWidths(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil || o == nil {
			return err
		}

		switch o.(type) {

		case pdf.Integer:
			// no further processing.

		case pdf.Float:
			// no further processing

		case pdf.Array:
			_, err = validateNumberArray(xRefTable, o)
			if err != nil {
				return err
			}

		default:
			return errors.Errorf("validateCIDFontGlyphWidths: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	return nil
}

func validateCIDFontDictEntryCIDSystemInfo(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "CIDSystemInfo", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateCIDSystemInfoDict(xRefTable, d1)

	}

	return err
}

func validateCIDFontDictEntryCIDToGIDMap(xRefTable *pdf.XRefTable, d pdf.Dict, isCIDFontType2 bool) error {

	if o, found := d.Find("CIDToGIDMap"); found {

		if xRefTable.ValidationMode == pdf.ValidationStrict && !isCIDFontType2 {
			return errors.New("pdfcpu: validateCIDFontDict: entry CIDToGIDMap not allowed - must be CIDFontType2")
		}

		err := validateCIDToGIDMap(xRefTable, o)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateCIDFontDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 9.7.4

	dictName := "CIDFontDict"

	// Type, required, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Font" })
	if err != nil {
		return err
	}

	var isCIDFontType2 bool
	var fontType string

	// Subtype, required, name
	subType, err := validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, pdf.V10, func(s string) bool { return s == "CIDFontType0" || s == "CIDFontType2" })
	if err != nil {
		return err
	}

	isCIDFontType2 = *subType == "CIDFontType2"
	fontType = subType.Value()

	// BaseFont, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// CIDSystemInfo, required, dict
	err = validateCIDFontDictEntryCIDSystemInfo(xRefTable, d, "CIDFontDict")
	if err != nil {
		return err
	}

	// FontDescriptor, required, dict
	err = validateFontDescriptor(xRefTable, d, dictName, fontType, REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// DW, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "DW", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// W, optional, array
	err = validateCIDFontGlyphWidths(xRefTable, d, dictName, "W", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// DW2, optional, array
	// An array of two numbers specifying the default metrics for vertical writing.
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "DW2", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// W2, optional, array
	err = validateCIDFontGlyphWidths(xRefTable, d, dictName, "W2", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// CIDToGIDMap, stream or (name /Identity)
	// optional, Type 2 CIDFonts with embedded associated TrueType font program only.
	return validateCIDFontDictEntryCIDToGIDMap(xRefTable, d, isCIDFontType2)
}

func validateDescendantFonts(xRefTable *pdf.XRefTable, d pdf.Dict, fontDictName string, required bool) error {

	// A one-element array holding a CID font dictionary.

	a, err := validateArrayEntry(xRefTable, d, fontDictName, "DescendantFonts", required, pdf.V10, func(a pdf.Array) bool { return len(a) == 1 })
	if err != nil || a == nil {
		return err
	}

	d1, err := xRefTable.DereferenceDict(a[0])
	if err != nil {
		return err
	}

	if d1 == nil {
		if required {
			return errors.Errorf("validateDescendantFonts: dict=%s required descendant font dict missing.\n", fontDictName)
		}
		return nil
	}

	return validateCIDFontDict(xRefTable, d1)
}

func validateType0FontDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "type0FontDict"

	// BaseFont, required, name
	_, err := validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Encoding, required,  name or CMap stream dict
	err = validateType0FontEncoding(xRefTable, d, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// DescendantFonts: one-element array specifying the CIDFont dictionary that is the descendant of this Type 0 font, required.
	err = validateDescendantFonts(xRefTable, d, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// ToUnicode, optional, CMap stream dict
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, pdf.V12, nil)

	return err
}

func validateType1FontDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 9.6.2

	dictName := "type1FontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	fontName, err := validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	fn := (*fontName).Value()
	required := xRefTable.Version() >= pdf.V15 || !validateStandardType1Font(fn)
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = !validateStandardType1Font(fn) && fn != "Arial"
	}
	// FirstChar,  required except for standard 14 fonts. since 1.5 always required, integer
	fc, err := validateIntegerEntry(xRefTable, d, dictName, "FirstChar", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	if !required && fc != nil {
		// For the standard 14 fonts, the entries FirstChar, LastChar, Widths and FontDescriptor shall either all be present or all be absent.
		if xRefTable.ValidationMode == pdf.ValidationStrict {
			required = true
		} else {
			// relaxed: do nothing
		}
	}

	// LastChar, required except for standard 14 fonts. since 1.5 always required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "LastChar", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Widths, required except for standard 14 fonts. since 1.5 always required, array of numbers
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Widths", required, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required since version 1.5; required unless standard font for version < 1.5, dict
	err = validateFontDescriptor(xRefTable, d, dictName, "Type1", required, pdf.V10)
	if err != nil {
		return err
	}

	// Encoding, optional, name or dict
	err = validateFontEncoding(xRefTable, d, dictName, OPTIONAL)
	if err != nil {
		return err
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, pdf.V12, nil)

	return err
}

func validateCharProcsDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "CharProcs", required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	for _, v := range d1 {

		_, _, err = xRefTable.DereferenceStreamDict(v)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateUseCMapEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	entryName := "UseCMap"

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		// no further processing

	case pdf.StreamDict:
		err = validateCMapStreamDict(xRefTable, &o)
		if err != nil {
			return err
		}

	default:
		return errors.Errorf("validateUseCMapEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateCIDSystemInfoDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "CIDSystemInfoDict"

	// Registry, required, ASCII string
	_, err := validateStringEntry(xRefTable, d, dictName, "Registry", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Ordering, required, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "Ordering", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Supplement, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Supplement", REQUIRED, pdf.V10, nil)

	return err
}

func validateCMapStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict) error {

	// See table 120

	dictName := "CMapStreamDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "CMap" })
	if err != nil {
		return err
	}

	// CMapName, required, name
	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "CMapName", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// CIDFontType0SystemInfo, required, dict
	d, err := validateDictEntry(xRefTable, sd.Dict, dictName, "CIDSystemInfo", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateCIDSystemInfoDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// WMode, optional, integer, 0 or 1
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "WMode", OPTIONAL, pdf.V10, func(i int) bool { return i == 0 || i == 1 })
	if err != nil {
		return err
	}

	// UseCMap, name or cmap stream dict, optional.
	// If present, the referencing CMap shall specify only
	// the character mappings that differ from the referenced CMap.
	return validateUseCMapEntry(xRefTable, sd.Dict, dictName, OPTIONAL, pdf.V10)
}

func validateType0FontEncoding(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool) error {

	entryName := "Encoding"

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, pdf.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		// no further processing

	case pdf.StreamDict:
		err = validateCMapStreamDict(xRefTable, &o)

	default:
		err = errors.Errorf("validateType0FontEncoding: dict=%s corrupt entry \"Encoding\"\n", dictName)

	}

	return err
}

func validateType3FontDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 9.6.5

	dictName := "type3FontDict"

	// Name, name, obsolet and should not be used.

	// FontBBox, required, rectangle
	_, err := validateRectangleEntry(xRefTable, d, dictName, "FontBBox", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FontMatrix, required, number array
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "FontMatrix", REQUIRED, pdf.V10, func(a pdf.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	// CharProcs, required, dict
	err = validateCharProcsDict(xRefTable, d, dictName, REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// Encoding, required, name or dict
	err = validateFontEncoding(xRefTable, d, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// FirstChar, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "FirstChar", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// LastChar, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "LastChar", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Widths, required, array of number
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Widths", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required since version 1.5 for tagged PDF documents, dict
	sinceVersion := pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	err = validateFontDescriptor(xRefTable, d, dictName, "Type3", xRefTable.Tagged, sinceVersion)
	if err != nil {
		return err
	}

	// Resources, optional, dict, since V1.2
	d1, err := validateDictEntry(xRefTable, d, dictName, "Resources", OPTIONAL, pdf.V12, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		_, err := validateResourceDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, pdf.V12, nil)

	return err
}

func validateFontDict(xRefTable *pdf.XRefTable, o pdf.Object) (err error) {

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		if len(d) == 0 {
			return nil
		}
	}

	if d.Type() == nil || *d.Type() != "Font" {
		return errors.New("pdfcpu: validateFontDict: corrupt font dict")
	}

	subtype := d.Subtype()
	if subtype == nil {
		return errors.New("pdfcpu: validateFontDict: missing Subtype")
	}

	switch *subtype {

	case "TrueType":
		err = validateTrueTypeFontDict(xRefTable, d)

	case "Type0":
		err = validateType0FontDict(xRefTable, d)

	case "Type1":
		err = validateType1FontDict(xRefTable, d)

	case "MMType1":
		err = validateType1FontDict(xRefTable, d)

	case "Type3":
		err = validateType3FontDict(xRefTable, d)

	default:
		return errors.Errorf("pdfcpu: validateFontDict: unknown Subtype: %s\n", *subtype)

	}

	return err
}

func validateFontResourceDict(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("fontResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Iterate over font resource dict
	for _, obj := range d {

		// Process fontDict
		err = validateFontDict(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	return nil
}

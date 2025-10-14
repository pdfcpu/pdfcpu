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
	"fmt"
	"strings"

	"github.com/mechiko/pdfcpu/pkg/log"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/font"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/model"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var ErrMissingFont = errors.New("pdfcpu: missing font dict")

func validateStandardType1Font(s string) bool {

	return types.MemberOf(s, []string{"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats"})
}

func validateFontFile3SubType(sd *types.StreamDict, fontType string, relaxed bool) error {

	// Hint about used font program.
	dictSubType := sd.Subtype()

	if dictSubType == nil {
		return errors.New("pdfcpu: validateFontFile3SubType: missing Subtype")
	}

	switch fontType {
	case "Type1":
		if *dictSubType != "Type1C" && *dictSubType != "OpenType" {
			if !relaxed {
				return errors.Errorf("pdfcpu: validateFontFile3SubType: Type1: unexpected Subtype %s", *dictSubType)
			}
			model.ShowSkipped(fmt.Sprintf("validateFontFile3SubType: Type1: unexpected Subtype %s", *dictSubType))
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

func validateFontFile(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, fontType string, required bool, sinceVersion model.Version) error {

	sd, err := validateStreamDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	// Process font file stream dict entries.

	// SubType
	if entryName == "FontFile3" {
		err = validateFontFile3SubType(sd, fontType, xRefTable.ValidationMode == model.ValidationRelaxed)
		if err != nil {
			return err
		}

	}

	dName := "fontFileStreamDict"
	compactFontFormat := entryName == "FontFile3"

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dName, "Length1", (fontType == "Type1" || fontType == "TrueType") && !compactFontFormat, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dName, "Length2", fontType == "Type1" && !compactFontFormat, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dName, "Length3", fontType == "Type1" && !compactFontFormat, model.V10, nil)
	if err != nil {
		return err
	}

	// Metadata, stream, optional, since 1.4
	return validateMetadata(xRefTable, sd.Dict, OPTIONAL, model.V14)
}

func validateFontDescriptorType(xRefTable *model.XRefTable, d types.Dict) (err error) {

	dictType := d.Type()

	if dictType == nil {

		if xRefTable.ValidationMode == model.ValidationRelaxed {
			if log.ValidateEnabled() {
				log.Validate.Println("validateFontDescriptor: missing entry \"Type\"")
			}
		} else {
			return errors.New("pdfcpu: validateFontDescriptor: missing entry \"Type\"")
		}

	}

	if dictType != nil && *dictType != "FontDescriptor" && *dictType != "Font" {
		return errors.New("pdfcpu: validateFontDescriptor: corrupt font descriptor dict")
	}

	return nil
}

func validateFontDescriptorFontName(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	required := true
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err := validateNameEntry(xRefTable, d, dictName, "FontName", required, model.V10, nil)
	if err != nil {
		if _, err = validateStringEntry(xRefTable, d, dictName, "FontName", required, model.V10, nil); err != nil {
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				model.ShowDigestedSpecViolationError(xRefTable, err)
				return nil
			}
		}
	}
	return err
}

func validateFontDescriptorFontFamily(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	required := true
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err := validateNameEntry(xRefTable, d, dictName, "FontFamily", required, sinceVersion, nil)
	if err != nil {
		if _, err = validateStringEntry(xRefTable, d, dictName, "FontFamily", required, sinceVersion, nil); err != nil {
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				model.ShowDigestedSpecViolationError(xRefTable, err)
				return nil
			}
		}
	}
	return err
}

func validateFontDescriptorFontStretch(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err := validateNameEntry(xRefTable, d, dictName, "FontStretch", OPTIONAL, sinceVersion, nil)
	return err
}

func validateFontDescriptorFontWeight(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	_, err := validateNumberEntry(xRefTable, d, dictName, "FontWeight", OPTIONAL, sinceVersion, nil)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			validateFontWeight := func(s string) bool {
				return types.MemberOf(s, []string{"Regular", "Bold", "Italic"})
			}
			_, err = validateNameEntry(xRefTable, d, dictName, "FontWeight", OPTIONAL, sinceVersion, validateFontWeight)
		}
	}
	return err
}

func validateFontDescriptorFontFlags(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	_, err := validateIntegerEntry(xRefTable, d, dictName, "Flags", REQUIRED, model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped("missing font descriptor \"Flags\"")
			return nil
		}
	}
	return err
}

func validateFontDescriptorFontBox(xRefTable *model.XRefTable, d types.Dict, dictName, fontDictType string) error {
	_, err := validateRectangleEntry(xRefTable, d, dictName, "FontBBox", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped("missing font descriptor \"FontBBox\"")
			return nil
		}
	}
	return err
}

func validateFontDescriptorItalicAngle(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	required := true
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err := validateNumberEntry(xRefTable, d, dictName, "ItalicAngle", required, model.V10, nil)
	return err
}

func validateFontDescriptorPart1(xRefTable *model.XRefTable, d types.Dict, dictName, fontDictType string) error {
	if err := validateFontDescriptorType(xRefTable, d); err != nil {
		return err
	}

	if err := validateFontDescriptorFontName(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateFontDescriptorFontFamily(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateFontDescriptorFontStretch(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateFontDescriptorFontWeight(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateFontDescriptorFontFlags(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateFontDescriptorFontBox(xRefTable, d, dictName, fontDictType); err != nil {
		return err
	}

	if err := validateFontDescriptorItalicAngle(xRefTable, d, dictName); err != nil {
		return err
	}

	return nil
}

func validateFontDescriptorPart2(xRefTable *model.XRefTable, d types.Dict, dictName, fontDictType string) error {

	_, err := validateNumberEntry(xRefTable, d, dictName, "Ascent", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		err = nil
		model.ShowSkipped("missing font descriptor \"Ascent\"")
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Descent", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		err = nil
		model.ShowSkipped("missing font descriptor \"Descent\"")
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Leading", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "CapHeight", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "XHeight", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "StemV", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		err = nil
		model.ShowSkipped("missing font descriptor \"StemV\"")
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "StemH", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "AvgWidth", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "MaxWidth", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "MissingWidth", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	err = validateFontDescriptorFontFile(xRefTable, d, dictName, fontDictType)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, dictName, "CharSet", OPTIONAL, model.V11, nil)

	return err
}

func validateFontDescriptorFontFile(xRefTable *model.XRefTable, d types.Dict, dictName, fontDictType string) (err error) {

	switch fontDictType {

	case "Type1", "MMType1":

		err = validateFontFile(xRefTable, d, dictName, "FontFile", fontDictType, OPTIONAL, model.V10)
		if err != nil {
			return err
		}

		err = validateFontFile(xRefTable, d, dictName, "FontFile3", fontDictType, OPTIONAL, model.V12)

	case "TrueType", "CIDFontType2":
		err = validateFontFile(xRefTable, d, dictName, "FontFile2", fontDictType, OPTIONAL, model.V11)

	case "CIDFontType0":
		err = validateFontFile(xRefTable, d, dictName, "FontFile3", fontDictType, OPTIONAL, model.V13)

	case "Type3": // No fontfile.

	default:
		return errors.Errorf("pdfcpu: unknown fontDictType: %s\n", fontDictType)

	}

	return err
}

func validateFontDescriptor(xRefTable *model.XRefTable, d types.Dict, fontDictName string, fontDictType string, required bool, sinceVersion model.Version) error {

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

		validateStyleDict := func(d types.Dict) bool {

			// see 9.8.3.2

			if d.Len() != 1 {
				return false
			}

			_, found := d.Find("Panose")

			return found
		}

		// Style, optional, dict
		_, err = validateDictEntry(xRefTable, d1, dictName, "Style", OPTIONAL, model.V10, validateStyleDict)
		if err != nil {
			return err
		}

		// Lang, optional, name
		sinceVersion := model.V15
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			sinceVersion = model.V13
		}
		_, err = validateNameEntry(xRefTable, d1, dictName, "Lang", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}

		// FD, optional, dict
		_, err = validateDictEntry(xRefTable, d1, dictName, "FD", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}

		// CIDSet, optional, stream
		_, err = validateStreamDictEntry(xRefTable, d1, dictName, "CIDSet", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateFontEncoding(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool) error {

	entryName := "Encoding"

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		return err
	}

	encodings := []string{"MacRomanEncoding", "MacExpertEncoding", "WinAnsiEncoding"}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		encodings = append(encodings, "FontSpecific", "StandardEncoding", "SymbolSetEncoding", "PDFDocEncoding")
	}

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		validateFontEncodingName := func(s string) bool {
			return types.MemberOf(s, encodings)
		}
		if !validateFontEncodingName(s) {
			return errors.Errorf("validateFontEncoding: invalid Encoding name: %s\n", s)
		}

	case types.Dict:
		// no further processing

	default:
		return errors.Errorf("validateFontEncoding: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateTrueTypeFontDict(xRefTable *model.XRefTable, d types.Dict) (string, error) {

	// see 9.6.3
	dictName := "trueTypeFontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	bf, err := validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, model.V10, nil)
	if err != nil {
		return "", err
	}
	fontName := ""
	if bf != nil {
		fontName = bf.String()
	}

	// FirstChar, required, integer
	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	if _, err = validateIntegerEntry(xRefTable, d, dictName, "FirstChar", required, model.V10, nil); err != nil {
		return "", err
	}

	// LastChar, required, integer
	required = REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	if _, err = validateIntegerEntry(xRefTable, d, dictName, "LastChar", required, model.V10, nil); err != nil {
		return "", err
	}

	// Widths, array of numbers.
	required = REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	if _, err = validateNumberArrayEntry(xRefTable, d, dictName, "Widths", required, model.V10, nil); err != nil {
		return "", err
	}

	// FontDescriptor, required, dictionary
	required = REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	if err = validateFontDescriptor(xRefTable, d, dictName, "TrueType", required, model.V10); err != nil {
		return "", err
	}

	// Encoding, optional, name or dict
	if err = validateFontEncoding(xRefTable, d, dictName, OPTIONAL); err != nil {
		return "", err
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, model.V12, nil)

	return fontName, err
}

func validateCIDToGIDMap(xRefTable *model.XRefTable, o types.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		if s != "Identity" {
			return errors.Errorf("pdfcpu: validateCIDToGIDMap: invalid name: %s - must be \"Identity\"\n", s)
		}

	case types.StreamDict:
		// no further processing

	default:
		return errors.New("pdfcpu: validateCIDToGIDMap: corrupt entry")

	}

	return nil
}

func validateCIDFontGlyphWidths(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {

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

		case types.Integer:
			// no further processing.

		case types.Float:
			// no further processing

		case types.Array:
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

func validateCIDFontDictEntryCIDSystemInfo(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "CIDSystemInfo", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateCIDSystemInfoDict(xRefTable, d1)

	}

	return err
}

func validateCIDFontDictEntryCIDToGIDMap(xRefTable *model.XRefTable, d types.Dict, isCIDFontType2 bool) error {

	if o, found := d.Find("CIDToGIDMap"); found {

		if xRefTable.ValidationMode == model.ValidationStrict && !isCIDFontType2 {
			return errors.New("pdfcpu: validateCIDFontDict: entry CIDToGIDMap not allowed - must be CIDFontType2")
		}

		err := validateCIDToGIDMap(xRefTable, o)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateCIDFontDict(xRefTable *model.XRefTable, d types.Dict) error {

	// see 9.7.4

	dictName := "CIDFontDict"

	// Type, required, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", REQUIRED, model.V10, func(s string) bool { return s == "Font" })
	if err != nil {
		return err
	}

	var isCIDFontType2 bool
	var fontType string

	// Subtype, required, name
	subType, err := validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "CIDFontType0" || s == "CIDFontType2" })
	if err != nil {
		return err
	}

	isCIDFontType2 = *subType == "CIDFontType2"
	fontType = subType.Value()

	// BaseFont, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// CIDSystemInfo, required, dict
	err = validateCIDFontDictEntryCIDSystemInfo(xRefTable, d, "CIDFontDict")
	if err != nil {
		return err
	}

	// FontDescriptor, required, dict
	err = validateFontDescriptor(xRefTable, d, dictName, fontType, REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// DW, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "DW", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// W, optional, array
	err = validateCIDFontGlyphWidths(xRefTable, d, dictName, "W", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// DW2, optional, array
	// An array of two numbers specifying the default metrics for vertical writing.
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "DW2", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// W2, optional, array
	err = validateCIDFontGlyphWidths(xRefTable, d, dictName, "W2", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// CIDToGIDMap, stream or (name /Identity)
	// optional, Type 2 CIDFonts with embedded associated TrueType font program only.
	return validateCIDFontDictEntryCIDToGIDMap(xRefTable, d, isCIDFontType2)
}

func validateDescendantFonts(xRefTable *model.XRefTable, d types.Dict, fontDictName string, required bool) error {

	// A one-element array holding a CID font dictionary.

	a, err := validateArrayEntry(xRefTable, d, fontDictName, "DescendantFonts", required, model.V10, func(a types.Array) bool { return len(a) == 1 })
	if err != nil || a == nil {
		return err
	}

	if len(a) != 1 {
		return font.ErrCorruptFontDict
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

func validateType0FontDict(xRefTable *model.XRefTable, d types.Dict) (string, error) {

	dictName := "type0FontDict"

	// BaseFont, required, name
	bf, err := validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, model.V10, nil)
	if err != nil {
		return "", err
	}

	fontName := ""
	if bf != nil {
		fontName = bf.String()
	}

	// Encoding, required,  name or CMap stream dict
	if err = validateType0FontEncoding(xRefTable, d, dictName, REQUIRED); err != nil {
		return "", err
	}

	// DescendantFonts: one-element array specifying the CIDFont dictionary that is the descendant of this Type 0 font, required.
	if err = validateDescendantFonts(xRefTable, d, dictName, REQUIRED); err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			err = ErrMissingFont
		}
		return fontName, err
	}

	// ToUnicode, optional, CMap stream dict
	sinceVersion := model.V12
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, sinceVersion, nil)
	if err != nil && xRefTable.ValidationMode == model.ValidationRelaxed {
		_, err = validateNameEntry(xRefTable, d, dictName, "ToUnicode", REQUIRED, sinceVersion, func(s string) bool { return s == "Identity-H" })
	}

	return fontName, err
}

func validateType1FontDict(xRefTable *model.XRefTable, d types.Dict) (string, error) {

	// see 9.6.2

	dictName := "type1FontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	bf, err := validateNameEntry(xRefTable, d, dictName, "BaseFont", REQUIRED, model.V10, nil)
	if err != nil {
		return "", err
	}

	fontName := bf.String()
	required := xRefTable.Version() >= model.V17 || !validateStandardType1Font(fontName)
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	// FirstChar,  required except for standard 14 fonts. since 2.0 always required, integer
	fc, err := validateIntegerEntry(xRefTable, d, dictName, "FirstChar", required, model.V10, nil)
	if err != nil {
		return "", err
	}

	if !required && fc != nil {
		// For the standard 14 fonts, the entries FirstChar, LastChar, Widths and FontDescriptor shall either all be present or all be absent.
		if xRefTable.ValidationMode == model.ValidationStrict {
			required = true
		}
	}

	// LastChar, required except for standard 14 fonts. since 2.0 always required, integer
	if _, err = validateIntegerEntry(xRefTable, d, dictName, "LastChar", required, model.V10, nil); err != nil {
		return "", err
	}

	// Widths, required except for standard 14 fonts. since 2.0 always required, array of numbers
	if _, err = validateNumberArrayEntry(xRefTable, d, dictName, "Widths", required, model.V10, nil); err != nil {
		return "", err
	}

	// FontDescriptor, required since version 2.0; required unless standard font for version <= 1.7, dict
	if err = validateFontDescriptor(xRefTable, d, dictName, "Type1", required, model.V10); err != nil {
		return "", err
	}

	// Encoding, optional, name or dict
	if err = validateFontEncoding(xRefTable, d, dictName, OPTIONAL); err != nil {
		return "", err
	}

	// ToUnicode, optional, stream
	sinceVersion := model.V12
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, sinceVersion, nil)

	return fontName, err
}

func validateCharProcsDict(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	d1, err := validateDictEntry(xRefTable, d, dictName, "CharProcs", required, sinceVersion, nil)
	if d1 == nil {
		return nil
	}
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		if !strings.Contains(err.Error(), "invalid type") {
			return err
		}
		model.ShowDigestedSpecViolation("\"CharProcs\" with invalid type")
		return nil
	}

	for _, v := range d1 {

		_, _, err = xRefTable.DereferenceStreamDict(v)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateUseCMapEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	entryName := "UseCMap"

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		// no further processing

	case types.StreamDict:
		err = validateCMapStreamDict(xRefTable, &o)
		if err != nil {
			return err
		}

	default:
		return errors.Errorf("validateUseCMapEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateCIDSystemInfoDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "CIDSystemInfoDict"

	// Registry, required, ASCII string
	_, err := validateStringEntry(xRefTable, d, dictName, "Registry", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Ordering, required, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "Ordering", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Supplement, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Supplement", REQUIRED, model.V10, nil)

	return err
}

func validateCMapStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {

	// See table 120

	dictName := "CMapStreamDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "CMap" })
	if err != nil {
		return err
	}

	// CMapName, required, name
	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "CMapName", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// CIDFontType0SystemInfo, required, dict
	d, err := validateDictEntry(xRefTable, sd.Dict, dictName, "CIDSystemInfo", REQUIRED, model.V10, nil)
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
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "WMode", OPTIONAL, model.V10, func(i int) bool { return i == 0 || i == 1 })
	if err != nil {
		return err
	}

	// UseCMap, name or cmap stream dict, optional.
	// If present, the referencing CMap shall specify only
	// the character mappings that differ from the referenced CMap.
	return validateUseCMapEntry(xRefTable, sd.Dict, dictName, OPTIONAL, model.V10)
}

func validateType0FontEncoding(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool) error {

	entryName := "Encoding"

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		// no further processing

	case types.StreamDict:
		err = validateCMapStreamDict(xRefTable, &o)

	default:
		err = errors.Errorf("validateType0FontEncoding: dict=%s corrupt entry \"Encoding\"\n", dictName)

	}

	return err
}

func validateType3FontDict(xRefTable *model.XRefTable, d types.Dict) error {

	// see 9.6.5

	dictName := "type3FontDict"

	// Name, name, obsolet and should not be used.

	// FontBBox, required, rectangle
	_, err := validateRectangleEntry(xRefTable, d, dictName, "FontBBox", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// FontMatrix, required, number array
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "FontMatrix", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	// CharProcs, required, dict
	err = validateCharProcsDict(xRefTable, d, dictName, REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Encoding, required, name or dict
	err = validateFontEncoding(xRefTable, d, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// FirstChar, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "FirstChar", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// LastChar, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "LastChar", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Widths, required, array of number
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Widths", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required since version 1.5 for tagged PDF documents, dict
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	err = validateFontDescriptor(xRefTable, d, dictName, "Type3", xRefTable.Tagged, sinceVersion)
	if err != nil {
		return err
	}

	// Resources, optional, dict, since V1.2
	d1, err := validateDictEntry(xRefTable, d, dictName, "Resources", OPTIONAL, model.V12, nil)
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
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "ToUnicode", OPTIONAL, model.V12, nil)

	return err
}

func _validateFontDict(xRefTable *model.XRefTable, d types.Dict, isIndRef bool, indRef types.IndirectRef) (fontName string, err error) {
	subtype := d.Subtype()
	if subtype == nil {
		return "", errors.New("pdfcpu: validateFontDict: missing Subtype")
	}

	switch *subtype {

	case "TrueType":
		fontName, err = validateTrueTypeFontDict(xRefTable, d)

	case "Type0":
		fontName, err = validateType0FontDict(xRefTable, d)

	case "Type1", "Type1C":
		fontName, err = validateType1FontDict(xRefTable, d)

	case "MMType1":
		return validateType1FontDict(xRefTable, d)

	case "Type3":
		err = validateType3FontDict(xRefTable, d)

	default:
		return "", errors.Errorf("pdfcpu: validateFontDict: unknown Subtype: %s", *subtype)

	}

	if isIndRef {
		if err1 := xRefTable.SetValid(indRef); err1 != nil {
			return "", err1
		}
	}

	return fontName, err
}

func validateFontDict(xRefTable *model.XRefTable, isIndRef bool, indRef types.IndirectRef) (string, error) {

	if isIndRef {

		ok, err := xRefTable.IsValid(indRef)
		if err != nil {
			return "", ErrMissingFont
		}
		if ok {
			return "", nil
		}

		if ok, err := xRefTable.IsBeingValidated(indRef); err != nil || ok {
			return "", err
		}

		if err := xRefTable.SetBeingValidated(indRef); err != nil {
			return "", err
		}
	}

	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil || d == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			err = ErrMissingFont
		}
		return "", err
	}

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		if len(d) == 0 {
			return "", nil
		}
	}

	if d.Type() == nil || *d.Type() != "Font" {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return "", errors.New("pdfcpu: validateFontDict: corrupt font dict")
		}
		model.ShowDigestedSpecViolation("missing fontDict entry \"Type\"")
	}

	return _validateFontDict(xRefTable, d, isIndRef, indRef)
}

func fixFontObjNr(m1 map[string]string, m2 map[string]types.IndirectRef, d types.Dict) {
	for k, v := range m1 {
		if v != "" {
			indRef, ok := m2[v]
			if ok {
				model.ShowRepaired(fmt.Sprintf("font %s mapped to objNr %d", k, indRef.ObjectNumber))
				d[k] = indRef
				continue
			}
		}
		d[k] = nil
	}
}

func validateFontResourceDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("fontResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// fontid, fontname
	m1 := map[string]string{}

	// fontname, objNr
	m2 := map[string]types.IndirectRef{}

	var defFontName string

	// Iterate over font resource dict
	for id, obj := range d {

		indRef, indRefOk := obj.(types.IndirectRef)

		// Process fontDict
		fn, err := validateFontDict(xRefTable, indRefOk, indRef)
		if err != nil {
			if err == ErrMissingFont {
				if xRefTable.ValidationMode == model.ValidationRelaxed {
					err = nil
					model.ShowSkipped(fmt.Sprintf("missing font: %s %s", id, fn))
					m1[id] = fn
					continue
				}
			}
			return err
		}
		if xRefTable.ValidationMode == model.ValidationRelaxed && indRefOk {
			m2[fn] = indRef
			if defFontName == "" {
				defFontName = fn
			}
		}
	}

	if len(m1) > 0 && xRefTable.ValidationMode == model.ValidationRelaxed {
		fixFontObjNr(m1, m2, d)
	}

	return nil
}

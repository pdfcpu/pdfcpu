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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ErrMissingFont signals a missing required font dictionary.
var ErrMissingFont = errors.New("missing font dict")

func validateStandardType1Font(s string) bool {
	return types.MemberOf(s, []string{"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats"})
}

func fontDictStringEntry(d types.Dict, key string) (string, bool) {
	o, found := d.Find(key)
	if !found {
		return "", false
	}

	sl, ok := o.(types.StringLiteral)
	if !ok {
		return "", false
	}

	s, err := types.StringLiteralToString(sl)
	return s, err == nil
}

func repairStringType1FontDict(xRefTable *model.XRefTable, d types.Dict) {
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		return
	}

	dictType, typeOK := fontDictStringEntry(d, "Type")
	subtype, subtypeOK := fontDictStringEntry(d, "Subtype")
	baseFont, baseFontOK := fontDictStringEntry(d, "BaseFont")
	if !typeOK || !subtypeOK || !baseFontOK || dictType != "Font" || subtype != "Type1" ||
		!validateStandardType1Font(baseFont) {
		return
	}

	d.Update("Type", types.Name(dictType))
	d.Update("Subtype", types.Name(subtype))
	d.Update("BaseFont", types.Name(baseFont))
	model.ShowRepaired("font dictionary string entries Type, Subtype and BaseFont converted to names")
}

func repairSelfReferentialFontToUnicode(
	xRefTable *model.XRefTable,
	d types.Dict,
	isIndRef bool,
	indRef types.IndirectRef,
) {
	if xRefTable.ValidationMode != model.ValidationRelaxed || !isIndRef {
		return
	}

	ir := d.IndirectRefEntry("ToUnicode")
	if ir == nil || *ir != indRef {
		return
	}

	d.Delete("ToUnicode")
	model.ShowRepaired("self-referential font ToUnicode entry removed")
}

func validateFontFile3SubType(xRefTable *model.XRefTable, sd *types.StreamDict, fontType string) error {
	// Hint about used font program.
	st, _, err := xRefTable.DereferenceNameEntry(sd.Dict, "Subtype")
	if err != nil {
		return fmt.Errorf("font file stream Subtype: %w", err)
	}
	if st == nil {
		return errors.New("missing Subtype")
	}
	s := st.Value()

	switch fontType {
	case "Type1":
		if s != "Type1C" && s != "OpenType" {
			if xRefTable.ValidationMode != model.ValidationRelaxed {
				return fmt.Errorf("Type1: unexpected Subtype %s", s)
			}
			model.ShowSkipped(fmt.Sprintf("validateFontFile3SubType: Type1: unexpected Subtype %s", s))
		}

	case "MMType1":
		if s != "Type1C" {
			return fmt.Errorf("MMType1: unexpected Subtype %s", s)
		}

	case "CIDFontType0":
		if s != "CIDFontType0C" && s != "OpenType" {
			return fmt.Errorf("CIDFontType0: unexpected Subtype %s", s)
		}

	case "CIDFontType2", "TrueType":
		if s != "OpenType" {
			return fmt.Errorf("%s: unexpected Subtype %s", fontType, s)
		}
	}

	return nil
}

func validateFontFile(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, fontType string, required bool, sinceVersion model.Version) error {
	fontFileObjNr := validationEntryObjectNumber(0, d, entryName)
	sd, err := validateStreamDictEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	// Process font file stream dict entries.

	// SubType
	if entryName == "FontFile3" {
		err = validateFontFile3SubType(xRefTable, sd, fontType)
		if err != nil {
			return model.WithValidationErrorObject(err, fontFileObjNr)
		}

	}

	dName := "fontFileStreamDict"
	compactFontFormat := entryName == "FontFile3"

	_, err = validateIntegerEntry(xRefTable, sd.Dict, fontFileObjNr, dName, "Length1", (fontType == "Type1" || fontType == "TrueType") && !compactFontFormat, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, fontFileObjNr, dName, "Length2", fontType == "Type1" && !compactFontFormat, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, fontFileObjNr, dName, "Length3", fontType == "Type1" && !compactFontFormat, model.V10, nil)
	if err != nil {
		return err
	}

	// Metadata, stream, optional, since 1.4
	err = validateMetadata(xRefTable, sd.Dict, OPTIONAL, model.V14)
	return model.WithValidationErrorObject(err, validationEntryObjectNumber(fontFileObjNr, sd.Dict, "Metadata"))
}

func validateFontDescriptorType(xRefTable *model.XRefTable, d types.Dict) (err error) {
	dictType, _, err := xRefTable.DereferenceNameEntry(d, "Type")
	if err != nil {
		return fmt.Errorf("font descriptor Type: %w", err)
	}

	if dictType == nil {

		if xRefTable.ValidationMode == model.ValidationRelaxed {
			if log.ValidateEnabled() {
				log.Validate.Println("validateFontDescriptor: missing entry \"Type\"")
			}
		} else {
			return errors.New("missing entry \"Type\"")
		}

	}

	if dictType != nil && dictType.Value() != "FontDescriptor" && dictType.Value() != "Font" {
		return errors.New("corrupt font descriptor dict")
	}

	return nil
}

func validateFontDescriptorFontName(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	required := true
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "FontName", required, model.V10, nil)
	if err != nil {
		if _, err = validateStringEntry(xRefTable, d, 0, dictName, "FontName", required, model.V10, nil); err != nil {
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				model.ShowDigestedSpecViolationError(err)
				return nil
			}
		}
	}
	return err
}

func validateFontDescriptorFontFamily(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	required := false
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "FontFamily", required, sinceVersion, nil)
	if err != nil {
		if _, err = validateStringEntry(xRefTable, d, 0, dictName, "FontFamily", required, sinceVersion, nil); err != nil {
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				model.ShowDigestedSpecViolationError(err)
				return nil
			}
		}
	}
	return err
}

func validateFontDescriptorFontStretch(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "FontStretch", OPTIONAL, sinceVersion, nil)
	return err
}

func validateFontDescriptorFontWeight(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	_, err := validateNumberEntry(xRefTable, d, 0, dictName, "FontWeight", OPTIONAL, sinceVersion, nil)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			validateFontWeight := func(s string) bool {
				return types.MemberOf(s, []string{"Regular", "Bold", "Italic"})
			}
			_, err = validateNameEntry(xRefTable, d, 0, dictName, "FontWeight", OPTIONAL, sinceVersion, validateFontWeight)
		}
	}
	return err
}

func validateFontDescriptorFontFlags(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	_, err := validateIntegerEntry(xRefTable, d, 0, dictName, "Flags", REQUIRED, model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped("missing font descriptor \"Flags\"")
			return nil
		}
	}
	return err
}

func validateFontDescriptorFontBox(xRefTable *model.XRefTable, d types.Dict, dictName, fontDictType string) error {
	_, err := validateRectangleEntry(xRefTable, d, 0, dictName, "FontBBox", fontDictType != "Type3", model.V10, nil)
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
	_, err := validateNumberEntry(xRefTable, d, 0, dictName, "ItalicAngle", required, model.V10, nil)
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
	_, err := validateNumberEntry(xRefTable, d, 0, dictName, "Ascent", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		err = nil
		model.ShowSkipped("missing font descriptor \"Ascent\"")
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Descent", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		err = nil
		model.ShowSkipped("missing font descriptor \"Descent\"")
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Leading", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "CapHeight", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "XHeight", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "StemV", fontDictType != "Type3", model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return err
		}
		err = nil
		model.ShowSkipped("missing font descriptor \"StemV\"")
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "StemH", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "AvgWidth", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "MaxWidth", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "MissingWidth", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	err = validateFontDescriptorFontFile(xRefTable, d, dictName, fontDictType)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, 0, dictName, "CharSet", OPTIONAL, model.V11, nil)

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
		return fmt.Errorf("unknown fontDictType: %s", fontDictType)

	}

	return err
}

func validateFontDescriptor(xRefTable *model.XRefTable, d types.Dict, fontDictName string, fontDictType string, required bool, sinceVersion model.Version) (err error) {
	descriptorObjNr := validationEntryObjectNumber(0, d, "FontDescriptor")
	defer func() {
		err = model.WithValidationErrorObject(err, descriptorObjNr)
	}()

	d1, err := validateDictEntry(xRefTable, d, 0, fontDictName, "FontDescriptor", required, sinceVersion, nil)
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
		_, err = validateDictEntry(xRefTable, d1, descriptorObjNr, dictName, "Style", OPTIONAL, model.V10, validateStyleDict)
		if err != nil {
			return err
		}

		// Lang, optional, name
		sinceVersion := model.V15
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			sinceVersion = model.V13
		}
		_, err = validateNameEntry(xRefTable, d1, descriptorObjNr, dictName, "Lang", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}

		// FD, optional, dict
		_, err = validateDictEntry(xRefTable, d1, descriptorObjNr, dictName, "FD", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}

		// CIDSet, optional, stream
		_, err = validateStreamDictEntry(xRefTable, d1, descriptorObjNr, dictName, "CIDSet", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func isZapfDingbatsFont(xRefTable *model.XRefTable, d types.Dict) (bool, error) {
	subtype, _, err := xRefTable.DereferenceNameEntry(d, "Subtype")
	if err != nil {
		return false, err
	}
	if subtype == nil || subtype.Value() != "Type1" {
		return false, nil
	}

	baseFont, _, err := xRefTable.DereferenceNameEntry(d, "BaseFont")
	if err != nil {
		return false, err
	}
	return baseFont != nil && baseFont.Value() == "ZapfDingbats", nil
}

func validateFontEncoding(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool) (err error) {
	entryName := "Encoding"
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	encodings := []string{"MacRomanEncoding", "MacExpertEncoding", "WinAnsiEncoding"}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		encodings = append(encodings,
			"FontSpecific", "StandardEncoding", "SymbolEncoding", "SymbolSetEncoding", "PDFDocEncoding")
		zapfDingbats, err := isZapfDingbatsFont(xRefTable, d)
		if err != nil {
			return fmt.Errorf("%s: classify ZapfDingbats font: %w", dictName, err)
		}
		if zapfDingbats {
			encodings = append(encodings, "ZapfDingbatsEncoding")
		}
	}

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		validateFontEncodingName := func(s string) bool {
			return types.MemberOf(s, encodings)
		}
		if !validateFontEncodingName(s) {
			return fmt.Errorf("%s.%s: invalid encoding name %q", dictName, entryName, s)
		}

	case types.Dict:
		// no further processing

	default:
		return fmt.Errorf("%s.%s: expected name or encoding dictionary, got %T", dictName, entryName, o)

	}

	return nil
}

func validateTrueTypeFontDict(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	// see 9.6.3
	dictName := "trueTypeFontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	bf, err := validateNameEntry(xRefTable, d, 0, dictName, "BaseFont", REQUIRED, model.V10, nil)
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
	if _, err = validateIntegerEntry(xRefTable, d, 0, dictName, "FirstChar", required, model.V10, nil); err != nil {
		return "", err
	}

	// LastChar, required, integer
	required = REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	if _, err = validateIntegerEntry(xRefTable, d, 0, dictName, "LastChar", required, model.V10, nil); err != nil {
		return "", err
	}

	// Widths, array of numbers.
	required = REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	if _, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Widths", required, model.V10, nil); err != nil {
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
	_, err = validateStreamDictEntry(xRefTable, d, 0, dictName, "ToUnicode", OPTIONAL, model.V12, nil)

	return fontName, err
}

func validateCIDToGIDMap(xRefTable *model.XRefTable, o types.Object) error {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("CIDToGIDMap: dereference: %w", err)
	}
	if o == nil {
		return errors.New("CIDToGIDMap: missing object")
	}

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		if s != "Identity" {
			return fmt.Errorf("CIDToGIDMap: invalid name %q, must be \"Identity\"", s)
		}

	case types.StreamDict:
		// no further processing

	default:
		return fmt.Errorf("CIDToGIDMap: expected Identity name or stream dict, got %T", o)

	}

	return nil
}

func validateCIDFontGlyphWidths(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	entryObjNr := validationEntryObjectNumber(0, d, entryName)
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	for i, raw := range a {
		objNr := validationObjectNumber(entryObjNr, raw)

		o, err := xRefTable.Dereference(raw)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: dereference: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		if o == nil {
			continue
		}

		switch o.(type) {

		case types.Integer:
			// no further processing.

		case types.Float:
			// no further processing

		case types.Array:
			_, err = validateNumberArray(xRefTable, o, objNr)
			if err != nil {
				return fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
			}

		default:
			err = fmt.Errorf("%s.%s[%d]: expected integer, float or number array, got %T", dictName, entryName, i, o)
			return model.WithValidationErrorObject(err, objNr)
		}

	}

	return nil
}

func validateCIDFontDictEntryCIDSystemInfo(xRefTable *model.XRefTable, d types.Dict, dictName string) (err error) {
	objNr := validationEntryObjectNumber(0, d, "CIDSystemInfo")
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "CIDSystemInfo", REQUIRED, model.V10, nil)
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
			return errors.New("CIDFontDict.CIDToGIDMap: not allowed unless Subtype is CIDFontType2")
		}

		err := validateCIDToGIDMap(xRefTable, o)
		if err != nil {
			return fmt.Errorf("CIDFontDict.CIDToGIDMap: %w", err)
		}

	}

	return nil
}

func validateCIDFontDict(xRefTable *model.XRefTable, d types.Dict) error {
	// see 9.7.4

	dictName := "CIDFontDict"

	// Type, required, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", REQUIRED, model.V10, func(s string) bool { return s == "Font" })
	if err != nil {
		return err
	}

	var isCIDFontType2 bool
	var fontType string

	// Subtype, required, name
	subType, err := validateNameEntry(xRefTable, d, 0, dictName, "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "CIDFontType0" || s == "CIDFontType2" })
	if err != nil {
		return err
	}

	isCIDFontType2 = *subType == "CIDFontType2"
	fontType = subType.Value()

	// BaseFont, required, name
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "BaseFont", REQUIRED, model.V10, nil)
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
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "DW", OPTIONAL, model.V10, nil)
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
	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "DW2", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
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

	a, err := validateArrayEntry(xRefTable, d, 0, fontDictName, "DescendantFonts", required, model.V10, func(a types.Array) bool { return len(a) == 1 })
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("%s.DescendantFonts: %w", fontDictName, err)
		}
		return nil
	}

	if len(a) != 1 {
		return fmt.Errorf("%s.DescendantFonts: expected one descendant font, got %d: %w", fontDictName, len(a), font.ErrCorruptFontDict)
	}

	descendantObjNr := validationObjectNumber(0, a[0])
	d1, err := xRefTable.DereferenceDict(a[0])
	if err != nil {
		err = fmt.Errorf("%s: dereference dict: %w", objectContext(fontDictName+".DescendantFonts[0]", a[0]), err)
		return model.WithValidationErrorObject(err, descendantObjNr)
	}

	if d1 == nil {
		if required {
			err = fmt.Errorf("%s: missing required descendant font dict", objectContext(fontDictName+".DescendantFonts[0]", a[0]))
			return model.WithValidationErrorObject(err, descendantObjNr)
		}
		return nil
	}

	if err := validateCIDFontDict(xRefTable, d1); err != nil {
		err = fmt.Errorf("%s: %w", objectContext(fontDictName+".DescendantFonts[0]", a[0]), err)
		return model.WithValidationErrorObject(err, descendantObjNr)
	}
	return nil
}

func validateType0FontDict(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	dictName := "type0FontDict"

	// BaseFont, required, name
	bf, err := validateNameEntry(xRefTable, d, 0, dictName, "BaseFont", REQUIRED, model.V10, nil)
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
	_, err = validateStreamDictEntry(xRefTable, d, 0, dictName, "ToUnicode", OPTIONAL, sinceVersion, nil)
	if err != nil && xRefTable.ValidationMode == model.ValidationRelaxed {
		_, err = validateNameEntry(xRefTable, d, 0, dictName, "ToUnicode", REQUIRED, sinceVersion, func(s string) bool { return s == "Identity-H" })
	}

	return fontName, err
}

func validateType1FontDict(xRefTable *model.XRefTable, d types.Dict) (string, error) {
	// see 9.6.2

	dictName := "type1FontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	bf, err := validateNameEntry(xRefTable, d, 0, dictName, "BaseFont", REQUIRED, model.V10, nil)
	if err != nil {
		return "", err
	}

	fontName := bf.String()
	required := xRefTable.Version() >= model.V17 || !validateStandardType1Font(fontName)
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	// FirstChar,  required except for standard 14 fonts. since 2.0 always required, integer
	fc, err := validateIntegerEntry(xRefTable, d, 0, dictName, "FirstChar", required, model.V10, nil)
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
	if _, err = validateIntegerEntry(xRefTable, d, 0, dictName, "LastChar", required, model.V10, nil); err != nil {
		return "", err
	}

	// Widths, required except for standard 14 fonts. since 2.0 always required, array of numbers
	if _, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Widths", required, model.V10, nil); err != nil {
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
	_, err = validateStreamDictEntry(xRefTable, d, 0, dictName, "ToUnicode", OPTIONAL, sinceVersion, nil)

	return fontName, err
}

func validateCharProcsDict(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, "CharProcs")
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "CharProcs", required, sinceVersion, nil)
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

	for _, key := range slices.Sorted(maps.Keys(d1)) {
		v := d1[key]

		_, _, err = xRefTable.DereferenceStreamDict(v)
		if err != nil {
			err = fmt.Errorf("CharProcs entry %s: %w", key, err)
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, v))
		}

	}

	return nil
}

func validateUseCMapEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) (err error) {
	entryName := "UseCMap"
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
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
		return fmt.Errorf("dict=%s corrupt entry \"%s\"", dictName, entryName)

	}

	return nil
}

func validateCIDSystemInfoDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "CIDSystemInfoDict"

	// Registry, required, ASCII string
	_, err := validateStringEntry(xRefTable, d, 0, dictName, "Registry", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Ordering, required, ASCII string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "Ordering", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Supplement, required, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "Supplement", REQUIRED, model.V10, nil)

	return err
}

func validateCMapStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	// See table 120

	dictName := "CMapStreamDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, sd.Dict, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "CMap" })
	if err != nil {
		return err
	}

	// CMapName, required, name
	_, err = validateNameEntry(xRefTable, sd.Dict, 0, dictName, "CMapName", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// CIDFontType0SystemInfo, required, dict
	d, err := validateDictEntry(xRefTable, sd.Dict, 0, dictName, "CIDSystemInfo", REQUIRED, model.V10, nil)
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
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "WMode", OPTIONAL, model.V10, func(i int) bool { return i == 0 || i == 1 })
	if err != nil {
		return err
	}

	// UseCMap, name or cmap stream dict, optional.
	// If present, the referencing CMap shall specify only
	// the character mappings that differ from the referenced CMap.
	return validateUseCMapEntry(xRefTable, sd.Dict, dictName, OPTIONAL, model.V10)
}

func validateType0FontEncoding(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool) (err error) {
	entryName := "Encoding"
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		// no further processing

	case types.StreamDict:
		err = validateCMapStreamDict(xRefTable, &o)

	default:
		err = fmt.Errorf("dict=%s corrupt entry \"Encoding\"", dictName)

	}

	return err
}

func validateType3FontDict(xRefTable *model.XRefTable, d types.Dict) error {
	// see 9.6.5

	dictName := "type3FontDict"

	// Name, name, obsolet and should not be used.

	// FontBBox, required, rectangle
	_, err := validateRectangleEntry(xRefTable, d, 0, dictName, "FontBBox", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// FontMatrix, required, number array
	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "FontMatrix", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 6 })
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
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "FirstChar", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// LastChar, required, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "LastChar", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Widths, required, array of number
	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Widths", REQUIRED, model.V10, nil)
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
	sinceVersion = model.V12
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "Resources", OPTIONAL, sinceVersion, nil)
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
	_, err = validateStreamDictEntry(xRefTable, d, 0, dictName, "ToUnicode", OPTIONAL, model.V12, nil)

	return err
}

func _validateFontDict(xRefTable *model.XRefTable, d types.Dict, isIndRef bool, indRef types.IndirectRef) (fontName string, err error) {
	repairStringType1FontDict(xRefTable, d)
	repairSelfReferentialFontToUnicode(xRefTable, d, isIndRef, indRef)

	subtype, _, err := xRefTable.DereferenceNameEntry(d, "Subtype")
	if err != nil {
		return "", fmt.Errorf("font dict Subtype: %w", err)
	}
	if subtype == nil {
		if isIndRef {
			return "", errors.New("font: missing Subtype")
		}
		return "", errors.New("font dict: missing Subtype")
	}

	switch subtype.Value() {

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
		return "", fmt.Errorf("font dict: unknown Subtype %q", subtype.Value())

	}

	if isIndRef {
		if err1 := xRefTable.SetValid(indRef); err1 != nil {
			return "", fmt.Errorf("font: mark valid: %w", err1)
		}
	}

	if err != nil {
		return fontName, fmt.Errorf("font dict Subtype %s: %w", subtype.Value(), err)
	}
	return fontName, nil
}

func checkFontIndRefValidationState(xRefTable *model.XRefTable, indRef types.IndirectRef) (bool, error) {
	ok, err := xRefTable.IsValid(indRef)
	if err != nil {
		return false, fmt.Errorf("font: check valid: %w: %w", err, ErrMissingFont)
	}
	if ok {
		return true, nil
	}

	if ok, err := xRefTable.IsBeingValidated(indRef); err != nil || ok {
		if err != nil {
			return false, fmt.Errorf("font: check being validated: %w", err)
		}
		return true, nil
	}

	if err := xRefTable.SetBeingValidated(indRef); err != nil {
		return false, fmt.Errorf("font: mark being validated: %w", err)
	}

	return false, nil
}

func dereferenceFontDict(xRefTable *model.XRefTable, indRef types.IndirectRef) (types.Dict, error) {
	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return nil, fmt.Errorf("font: dereference dict: %w: %w", err, ErrMissingFont)
		}
		return nil, fmt.Errorf("font: dereference dict: %w", err)
	}
	if d == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return nil, fmt.Errorf("font: missing dict: %w", ErrMissingFont)
		}
		return nil, errors.New("font: missing dict")
	}
	return d, nil
}

func validateFontDict(xRefTable *model.XRefTable, isIndRef bool, indRef types.IndirectRef) (string, error) {
	if isIndRef {
		done, err := checkFontIndRefValidationState(xRefTable, indRef)
		if err != nil || done {
			return "", err
		}
	}

	d, err := dereferenceFontDict(xRefTable, indRef)
	if err != nil {
		return "", err
	}

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		if len(d) == 0 {
			return "", nil
		}
	}
	repairStringType1FontDict(xRefTable, d)

	typ, _, err := xRefTable.DereferenceNameEntry(d, "Type")
	if err != nil {
		return "", fmt.Errorf("font Type: %w", err)
	}
	if typ == nil || typ.Value() != "Font" {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return "", errors.New("font: expected Type Font")
		}
		model.ShowDigestedSpecViolation("missing fontDict entry \"Type\"")
	}

	return _validateFontDict(xRefTable, d, isIndRef, indRef)
}

func validateFontObject(xRefTable *model.XRefTable, obj types.Object) (string, bool, types.IndirectRef, error) {
	indRef, ok := obj.(types.IndirectRef)
	if ok {
		fontName, err := validateFontDict(xRefTable, true, indRef)
		return fontName, true, indRef, err
	}

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return "", false, types.IndirectRef{}, fmt.Errorf("font resource: dereference direct font dict: %w", err)
	}
	if d == nil {
		return "", false, types.IndirectRef{}, ErrMissingFont
	}

	fontName, err := _validateFontDict(xRefTable, d, false, types.IndirectRef{})
	return fontName, false, types.IndirectRef{}, err
}

func fixFontObjNr(m1 map[string]string, m2 map[string]types.IndirectRef, d types.Dict) {
	for _, k := range slices.Sorted(maps.Keys(m1)) {
		v := m1[k]
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

func isEncodingDict(xRefTable *model.XRefTable, o types.Object) bool {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return false
	}
	t, _, err := xRefTable.DereferenceNameEntry(d, "Type")
	return err == nil && t != nil && t.Value() == "Encoding"
}

func isMisplacedEncodingResourceDict(xRefTable *model.XRefTable, id string, o types.Object) bool {
	if id != "Encoding" {
		return false
	}
	d, err := xRefTable.DereferenceDict(o)
	if err != nil || len(d) == 0 {
		return false
	}
	for _, o := range d {
		if !isEncodingDict(xRefTable, o) {
			return false
		}
	}
	return true
}

func validateFontResourceDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	resourceObjNr := validationObjectNumber(0, o)

	// Version check
	err := xRefTable.ValidateVersion("fontResourceDict", sinceVersion)
	if err != nil {
		err = fmt.Errorf("Font resource dict: version: %w", err)
		return model.WithValidationErrorObject(err, resourceObjNr)
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		err = fmt.Errorf("Font resource dict: dereference dict: %w", err)
		return model.WithValidationErrorObject(err, resourceObjNr)
	}
	if d == nil {
		err = errors.New("Font resource dict: missing dict")
		return model.WithValidationErrorObject(err, resourceObjNr)
	}

	// fontid, fontname
	m1 := map[string]string{}

	// fontname, objNr
	m2 := map[string]types.IndirectRef{}

	var defFontName string

	// Iterate over font resource dict
	for _, id := range slices.Sorted(maps.Keys(d)) {
		obj := d[id]
		if xRefTable.ValidationMode == model.ValidationRelaxed && isMisplacedEncodingResourceDict(xRefTable, id, obj) {
			d.Delete(id)
			model.ShowMsg("removed misplaced Encoding dictionary from Font resources")
			continue
		}
		fontObjNr := validationObjectNumber(resourceObjNr, obj)

		// Process fontDict
		fn, indRefOk, indRef, err := validateFontObject(xRefTable, obj)
		if err != nil {
			if errors.Is(err, ErrMissingFont) {
				if xRefTable.ValidationMode == model.ValidationRelaxed {
					err = nil
					model.ShowSkipped(fmt.Sprintf("missing font: %s %s", id, fn))
					m1[id] = fn
					continue
				}
			}
			err = fmt.Errorf("Font resource %s: %w", id, err)
			return model.WithValidationErrorObject(err, fontObjNr)
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

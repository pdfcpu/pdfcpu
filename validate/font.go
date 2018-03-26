package validate

import (
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateStandardType1Font(s string) bool {

	return memberOf(s, []string{"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats"})
}

func validateFontFile3SubType(sd *types.PDFStreamDict, fontType string) error {

	dictSubType := sd.Subtype()

	if fontType == "Type1" || fontType == "MMType1" {
		if dictSubType == nil || *dictSubType != "Type1C" {
			return errors.New("validateFontFile3SubType: FontFile3 missing Subtype \"Type1C\"")
		}
	}

	if fontType == "CIDFontType0" {
		if dictSubType == nil || *dictSubType != "CIDFontType0C" {
			return errors.New("validateFontFile3SubType: FontFile3 missing Subtype \"CIDFontType0C\"")
		}
	}

	if fontType == "OpenType" {
		if dictSubType == nil || *dictSubType != "OpenType" {
			return errors.New("validateFontFile3SubType: FontFile3 missing Subtype \"OpenType\"")
		}
	}

	return nil
}

func validateFontFile(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, fontType string, required bool, sinceVersion types.PDFVersion) error {

	streamDict, err := validateStreamDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || streamDict == nil {
		return err
	}

	// Process font file stream dict entries.

	// SubType, required if referenced from FontFile3.
	if entryName == "FontFile3" {

		err = validateFontFile3SubType(streamDict, fontType)
		if err != nil {
			return err
		}

	}

	dName := "fontFileStreamDict"
	compactFontFormat := entryName == "FontFile3"

	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dName, "Length1", (fontType == "Type1" || fontType == "TrueType") && !compactFontFormat, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dName, "Length2", fontType == "Type1" && !compactFontFormat, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dName, "Length3", fontType == "Type1" && !compactFontFormat, types.V10, nil)
	if err != nil {
		return err
	}

	// Metadata, stream, optional, since 1.4
	return validateMetadata(xRefTable, &streamDict.PDFDict, OPTIONAL, types.V14)
}

func validateFontDescriptorType(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	dictType := dict.Type()

	if dictType == nil {

		if xRefTable.ValidationMode == types.ValidationRelaxed {
			log.Debug.Println("validateFontDescriptor: missing entry \"Type\"")
		} else {
			return errors.New("validateFontDescriptor: missing entry \"Type\"")
		}

	}

	if dictType != nil && *dictType != "FontDescriptor" {
		return errors.New("writeFontDescriptor: corrupt font descriptor dict")
	}

	return
}

func validateFontDescriptorPart1(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, fontDictType string) error {

	err := validateFontDescriptorType(xRefTable, dict)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "FontName", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	sinceVersion := types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "FontFamily", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	sinceVersion = types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "FontStretch", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	sinceVersion = types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberEntry(xRefTable, dict, dictName, "FontWeight", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "FontBBox", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "ItalicAngle", REQUIRED, types.V10, nil)

	return err
}

func validateFontDescriptorPart2(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, fontDictType string) error {

	_, err := validateNumberEntry(xRefTable, dict, dictName, "Ascent", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Descent", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Leading", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "CapHeight", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "XHeight", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "StemV", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "StemH", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "AvgWidth", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "MaxWidth", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "MissingWidth", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	err = validateFontDescriptorFontFile(xRefTable, dict, dictName, fontDictType)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "CharSet", OPTIONAL, types.V11, nil)

	return err
}

func validateFontDescriptorFontFile(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, fontDictType string) (err error) {

	switch fontDictType {

	case "Type1", "MMType1":

		err = validateFontFile(xRefTable, dict, dictName, "FontFile", fontDictType, OPTIONAL, types.V10)
		if err != nil {
			return err
		}

		err = validateFontFile(xRefTable, dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V12)

	case "TrueType", "CIDFontType2":
		err = validateFontFile(xRefTable, dict, dictName, "FontFile2", fontDictType, OPTIONAL, types.V11)

	case "CIDFontType0":
		err = validateFontFile(xRefTable, dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V13)

	case "OpenType":
		err = validateFontFile(xRefTable, dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V16)

	default:
		return errors.Errorf("unknown fontDictType: %s\n", fontDictType)

	}

	return err
}

func validateFontDescriptor(xRefTable *types.XRefTable, fontDict *types.PDFDict, fontDictName string, fontDictType string, required bool, sinceVersion types.PDFVersion) error {

	dict, err := validateDictEntry(xRefTable, fontDict, fontDictName, "FontDescriptor", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName := "fdDict"

	// Process font descriptor dict

	err = validateFontDescriptorPart1(xRefTable, dict, dictName, fontDictType)
	if err != nil {
		return err
	}

	err = validateFontDescriptorPart2(xRefTable, dict, dictName, fontDictType)
	if err != nil {
		return err
	}

	if fontDictType == "CIDFontType0" || fontDictType == "CIDFontType2" {

		validateStyleDict := func(dict types.PDFDict) bool {

			// see 9.8.3.2

			if dict.Len() != 1 {
				return false
			}

			_, found := dict.Find("Panose")

			return found
		}

		// Style, optional, dict
		_, err = validateDictEntry(xRefTable, dict, dictName, "Style", OPTIONAL, types.V10, validateStyleDict)
		if err != nil {
			return err
		}

		// Lang, optional, name
		_, err = validateNameEntry(xRefTable, dict, dictName, "Lang", OPTIONAL, types.V15, nil)
		if err != nil {
			return err
		}

		// FD, optional, dict
		_, err = validateDictEntry(xRefTable, dict, dictName, "FD", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}

		// CIDSet, optional, stream
		_, err = validateStreamDictEntry(xRefTable, dict, dictName, "CIDSet", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateFontEncoding(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool) error {

	entryName := "Encoding"

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, types.V10)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		validateFontEncodingName := func(s string) bool {
			return memberOf(s, []string{"MacRomanEncoding", "MacExpertEncoding", "WinAnsiEncoding"})
		}
		if !validateFontEncodingName(s) {
			return errors.Errorf("validateFontEncoding: invalid Encoding name: %s\n", s)
		}

	case types.PDFDict:
		// no further processing

	default:
		return errors.Errorf("validateFontEncoding: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateTrueTypeFontDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 9.6.3
	dictName := "trueTypeFontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// FirstChar, required, integer
	required := REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "FirstChar", required, types.V10, nil)
	if err != nil {
		return err
	}

	// LastChar, required, integer
	required = REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LastChar", required, types.V10, nil)
	if err != nil {
		return err
	}

	// Widths, array of numbers.
	required = REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Widths", required, types.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required, dictionary
	required = REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	err = validateFontDescriptor(xRefTable, dict, dictName, "TrueType", required, types.V10)
	if err != nil {
		return err
	}

	// Encoding, optional, name or dict
	err = validateFontEncoding(xRefTable, dict, dictName, OPTIONAL)
	if err != nil {
		return err
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)

	return err
}

func validateCIDToGIDMap(xRefTable *types.XRefTable, obj types.PDFObject) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			return errors.Errorf("validateCIDToGIDMap: invalid name: %s - must be \"Identity\"\n", s)
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		return errors.New("validateCIDToGIDMap: corrupt entry")

	}

	return nil
}

func validateCIDFontGlyphWidths(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil || obj == nil {
			return err
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing

		case types.PDFArray:
			_, err = validateNumberArray(xRefTable, obj)
			if err != nil {
				return err
			}

		default:
			return errors.Errorf("validateCIDFontGlyphWidths: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	return nil
}

func validateCIDFontDictEntryCIDSystemInfo(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	var d *types.PDFDict

	d, err := validateDictEntry(xRefTable, dict, dictName, "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateCIDSystemInfoDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateCIDFontDictEntryCIDToGIDMap(xRefTable *types.XRefTable, dict *types.PDFDict, isCIDFontType2 bool) error {

	if obj, found := dict.Find("CIDToGIDMap"); found {

		if !isCIDFontType2 {
			return errors.New("validateCIDFontDict: entry CIDToGIDMap not allowed - must be CIDFontType2")
		}

		err := validateCIDToGIDMap(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateCIDFontDict(xRefTable *types.XRefTable, fontDict *types.PDFDict) error {

	// see 9.7.4

	dictName := "CIDFontDict"

	// Type, required, name
	_, err := validateNameEntry(xRefTable, fontDict, dictName, "Type", REQUIRED, types.V10, func(s string) bool { return s == "Font" })
	if err != nil {
		return err
	}

	var isCIDFontType2 bool
	var fontType string

	// Subtype, required, name
	subType, err := validateNameEntry(xRefTable, fontDict, dictName, "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "CIDFontType0" || s == "CIDFontType2" })
	if err != nil {
		return err
	}

	isCIDFontType2 = true
	fontType = subType.Value()

	// BaseFont, required, name
	_, err = validateNameEntry(xRefTable, fontDict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// CIDSystemInfo, required, dict
	err = validateCIDFontDictEntryCIDSystemInfo(xRefTable, fontDict, "CIDFontDict")
	if err != nil {
		return err
	}

	// FontDescriptor, required, dict
	err = validateFontDescriptor(xRefTable, fontDict, dictName, fontType, REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// DW, optional, integer
	_, err = validateIntegerEntry(xRefTable, fontDict, dictName, "DW", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// W, optional, array
	err = validateCIDFontGlyphWidths(xRefTable, fontDict, dictName, "W", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// DW2, optional, array
	// An array of two numbers specifying the default metrics for vertical writing.
	_, err = validateNumberArrayEntry(xRefTable, fontDict, dictName, "DW2", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	// W2, optional, array
	err = validateCIDFontGlyphWidths(xRefTable, fontDict, dictName, "W2", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// CIDToGIDMap, stream or (name /Identity)
	// optional, Type 2 CIDFonts with embedded associated TrueType font program only.
	return validateCIDFontDictEntryCIDToGIDMap(xRefTable, fontDict, isCIDFontType2)
}

func validateDescendantFonts(xRefTable *types.XRefTable, fontDict *types.PDFDict, fontDictName string, required bool) error {

	// A one-element array holding a CID font dictionary.

	arr, err := validateArrayEntry(xRefTable, fontDict, fontDictName, "DescendantFonts", required, types.V10, func(arr types.PDFArray) bool { return len(arr) == 1 })
	if err != nil || arr == nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict((*arr)[0])
	if err != nil {
		return err
	}

	if dict == nil {
		if required {
			return errors.Errorf("validateDescendantFonts: dict=%s required descendant font dict missing.\n", fontDictName)
		}
		return nil
	}

	return validateCIDFontDict(xRefTable, dict)
}

func validateType0FontDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "type0FontDict"

	// BaseFont, required, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// Encoding, required,  name or CMap stream dict
	err = validateType0FontEncoding(xRefTable, dict, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// DescendantFonts: one-element array specifying the CIDFont dictionary that is the descendant of this Type 0 font, required.
	err = validateDescendantFonts(xRefTable, dict, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// ToUnicode, optional, CMap stream dict
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}

	//if streamDict, ok := writeStreamDictEntry(source, dest, dict, "type0FontDict", "ToUnicode", OPTIONAL, V12, nil); ok {
	//	writeCMapStreamDict(source, dest, *streamDict)
	//}

	return nil
}

func validateType1FontDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 9.6.2

	dictName := "type1FontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, required, name
	fontName, err := validateNameEntry(xRefTable, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	required := xRefTable.Version() >= types.V15 || !validateStandardType1Font((*fontName).String())
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = !validateStandardType1Font((*fontName).String())
	}
	// FirstChar,  required except for standard 14 fonts. since 1.5 always required, integer
	fc, err := validateIntegerEntry(xRefTable, dict, dictName, "FirstChar", required, types.V10, nil)
	if err != nil {
		return err
	}

	if !required && fc != nil {
		// For the standard 14 fonts, the entries FirstChar, LastChar, Widths and FontDescriptor shall either all be present or all be absent.
		if xRefTable.ValidationMode == types.ValidationStrict {
			required = true
		} else {
			// relaxed: do nothing
		}
	}

	// LastChar, required except for standard 14 fonts. since 1.5 always required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LastChar", required, types.V10, nil)
	if err != nil {
		return err
	}

	// Widths, required except for standard 14 fonts. since 1.5 always required, array of numbers
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Widths", required, types.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required since version 1.5; required unless standard font for version < 1.5, dict
	err = validateFontDescriptor(xRefTable, dict, dictName, "Type1", required, types.V10)
	if err != nil {
		return err
	}

	// Encoding, optional, name or dict
	err = validateFontEncoding(xRefTable, dict, dictName, OPTIONAL)
	if err != nil {
		return err
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)

	return err
}

func validateCharProcsDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, "CharProcs", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	for _, v := range d.Dict {

		_, err = xRefTable.DereferenceStreamDict(v)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateUseCMapEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	entryName := "UseCMap"

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStreamDict:
		err = validateCMapStreamDict(xRefTable, &obj)
		if err != nil {
			return err
		}

	default:
		return errors.Errorf("validateUseCMapEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateCIDSystemInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "CIDSystemInfoDict"

	// Registry, required, ASCII string
	_, err := validateStringEntry(xRefTable, dict, dictName, "Registry", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// Ordering, required, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Ordering", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// Supplement, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Supplement", REQUIRED, types.V10, nil)

	return err
}

func validateCMapStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) error {

	// See table 120

	dictName := "CMapStreamDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, &streamDict.PDFDict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "CMap" })
	if err != nil {
		return err
	}

	// CMapName, required, name
	_, err = validateNameEntry(xRefTable, &streamDict.PDFDict, dictName, "CMapName", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// CIDFontType0SystemInfo, required, dict
	dict, err := validateDictEntry(xRefTable, &streamDict.PDFDict, dictName, "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	if dict != nil {
		err = validateCIDSystemInfoDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	// WMode, optional, integer, 0 or 1
	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dictName, "WMode", OPTIONAL, types.V10, func(i int) bool { return i == 0 || i == 1 })
	if err != nil {
		return err
	}

	// UseCMap, name or cmap stream dict, optional.
	// If present, the referencing CMap shall specify only
	// the character mappings that differ from the referenced CMap.
	return validateUseCMapEntry(xRefTable, &streamDict.PDFDict, dictName, OPTIONAL, types.V10)
}

func validateType0FontEncoding(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool) error {

	entryName := "Encoding"

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, types.V10)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStreamDict:
		err = validateCMapStreamDict(xRefTable, &obj)

	default:
		err = errors.Errorf("validateType0FontEncoding: dict=%s corrupt entry \"Encoding\"\n", dictName)

	}

	return err
}

func validateType3FontDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 9.6.5

	dictName := "type3FontDict"

	// Name, name, obsolet and should not be used.

	// FontBBox, required, rectangle
	_, err := validateRectangleEntry(xRefTable, dict, dictName, "FontBBox", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// FontMatrix, required, number array
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "FontMatrix", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	// CharProcs, required, dict
	err = validateCharProcsDict(xRefTable, dict, dictName, REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// Encoding, required, name or dict
	err = validateFontEncoding(xRefTable, dict, dictName, REQUIRED)
	if err != nil {
		return err
	}

	// FirstChar, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "FirstChar", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// LastChar, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LastChar", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// Widths, required, array of number
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Widths", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// FontDescriptor, required since version 1.5 for tagged PDF documents, dict
	if xRefTable.Tagged {
		err = validateFontDescriptor(xRefTable, dict, dictName, "Type3", REQUIRED, types.V15)
		if err != nil {
			return err
		}
	}

	// Resources, optional, dict, since V1.2
	d, err := validateDictEntry(xRefTable, dict, dictName, "Resources", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}
	if d != nil {
		_, err := validateResourceDict(xRefTable, *d)
		if err != nil {
			return err
		}
	}

	// ToUnicode, optional, stream
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)

	return err
}

func validateFontDict(xRefTable *types.XRefTable, obj types.PDFObject) (err error) {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	if dict.Type() == nil || *dict.Type() != "Font" {
		return errors.New("validateFontDict: corrupt font dict")
	}

	subtype := dict.Subtype()
	if subtype == nil {
		return errors.New("validateFontDict: missing Subtype")
	}

	switch *subtype {

	case "TrueType":
		err = validateTrueTypeFontDict(xRefTable, dict)

	case "Type0":
		err = validateType0FontDict(xRefTable, dict)

	case "Type1":
		err = validateType1FontDict(xRefTable, dict)

	case "MMType1":
		err = validateType1FontDict(xRefTable, dict)

	case "Type3":
		err = validateType3FontDict(xRefTable, dict)

	default:
		return errors.Errorf("validateFontDict: unknown Subtype: %s\n", *subtype)

	}

	return err
}

func validateFontResourceDict(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// Version check
	err := xRefTable.ValidateVersion("fontResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Iterate over font resource dict
	for _, obj := range dict.Dict {

		// Process fontDict
		err = validateFontDict(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	return nil
}

package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateFontFile3SubType(sd *types.PDFStreamDict, fontType string) (err error) {

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

	return
}

func validateFontFile(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, fontType string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validateFontFile begin: dictName=%s entryName=%s fontType=%s ***\n", dictName, entryName, fontType)

	streamDict, err := validateStreamDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if streamDict == nil {
		// optional and nil or already written
		logInfoValidate.Printf("validateFontFile end: is nil. dictName=%s entryName=%s fontType=%s\n", dictName, entryName, fontType)
		return
	}

	// Process font file stream dict entries.

	// SubType, required if referenced from FontFile3.
	if entryName == "FontFile3" {

		err = validateFontFile3SubType(streamDict, fontType)
		if err != nil {
			return
		}

	}

	dName := "fontFileStreamDict"
	compactFontFormat := entryName == "FontFile3"

	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dName, "Length1", (fontType == "Type1" || fontType == "TrueType") && !compactFontFormat, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dName, "Length2", fontType == "Type1" && !compactFontFormat, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dName, "Length3", fontType == "Type1" && !compactFontFormat, types.V10, nil)
	if err != nil {
		return
	}

	// Metadata, stream, optional, since 1.4
	err = validateMetadata(xRefTable, &streamDict.PDFDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateFontFile end: dictName=%s entryName=%s fontType=%s ***\n", dictName, entryName, fontType)

	return
}

func validateFontDescriptorType(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	dictType := dict.Type()

	if dictType == nil {

		if xRefTable.ValidationMode == types.ValidationRelaxed {
			logDebugValidate.Println("validateFontDescriptor: missing entry \"Type\"")
		} else {
			return errors.New("validateFontDescriptor: missing entry \"Type\"")
		}

	}

	if dictType != nil && *dictType != "FontDescriptor" {
		return errors.New("writeFontDescriptor: corrupt font descriptor dict")
	}

	return
}

func validateFontDescriptorPart1(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, fontDictType string) (err error) {

	err = validateFontDescriptorType(xRefTable, dict)
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "FontName", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	sinceVersion := types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "FontFamily", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	sinceVersion = types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "FontStretch", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	sinceVersion = types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberEntry(xRefTable, dict, dictName, "FontWeight", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "FontBBox", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "ItalicAngle", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	return
}

func validateFontDescriptorPart2(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, fontDictType string) (err error) {

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Ascent", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Descent", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Leading", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "CapHeight", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "XHeight", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "StemV", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "StemH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "AvgWidth", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "MaxWidth", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "MissingWidth", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	err = validateFontDescriptorFontFile(xRefTable, dict, dictName, fontDictType)
	if err != nil {
		return
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "CharSet", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	return
}

func validateFontDescriptorFontFile(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, fontDictType string) (err error) {

	if fontDictType == "Type1" || fontDictType == "MMType1" {

		err = validateFontFile(xRefTable, dict, dictName, "FontFile", fontDictType, OPTIONAL, types.V10)
		if err != nil {
			return
		}

		err = validateFontFile(xRefTable, dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V12)
		if err != nil {
			return
		}

	}

	if fontDictType == "TrueType" || fontDictType == "CIDFontType2" {
		err = validateFontFile(xRefTable, dict, dictName, "FontFile2", fontDictType, OPTIONAL, types.V11)
		if err != nil {
			return
		}
	}

	if fontDictType == "CIDFontType0" {
		err = validateFontFile(xRefTable, dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V13)
		if err != nil {
			return
		}
	}

	if fontDictType == "OpenType" {
		err = validateFontFile(xRefTable, dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V16)
		if err != nil {
			return
		}
	}

	return
}

func validateFontDescriptor(xRefTable *types.XRefTable, fontDict *types.PDFDict, fontDictName string, fontDictType string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validateFontDescriptor begin: dictName=%s referrer=%s***\n", fontDictName, fontDictType)

	dict, err := validateDictEntry(xRefTable, fontDict, fontDictName, "FontDescriptor", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		// optional and nil or already written
		logInfoValidate.Println("validateFontDescriptor end")
		return
	}

	dictName := "fdDict"

	// Process font descriptor dict

	err = validateFontDescriptorPart1(xRefTable, dict, dictName, fontDictType)
	if err != nil {
		return
	}

	err = validateFontDescriptorPart2(xRefTable, dict, dictName, fontDictType)
	if err != nil {
		return
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

		// Style, dict, optional
		_, err = validateDictEntry(xRefTable, dict, dictName, "Style", OPTIONAL, types.V10, validateStyleDict)
		if err != nil {
			return
		}

		// Lang, name, optional
		_, err = validateNameEntry(xRefTable, dict, dictName, "Lang", OPTIONAL, types.V15, nil)
		if err != nil {
			return
		}

		// FD, dict, optional
		_, err = validateDictEntry(xRefTable, dict, dictName, "FD", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}

		// CIDSet, stream, optional
		_, err = validateStreamDictEntry(xRefTable, dict, dictName, "CIDSet", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateFontDescriptor end ***")

	return
}

func validateFontEncoding(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool) (err error) {

	logInfoValidate.Printf("*** validateFontEncoding begin: dictName=%s ***", dictName)

	entryName := "Encoding"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateFontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFontEncoding end: \"%s\" is nil.\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateFontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFontEncoding end: entry \"%s\" is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validateFontEncodingName(s) {
			err = errors.Errorf("validateFontEncoding: invalid Encoding name: %s\n", s)
		}

	case types.PDFDict:
		// no further processing

	default:
		err = errors.Errorf("validateFontEncoding: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Printf("*** validateFontEncoding end: \"%s\" is nil. ***", entryName)

	return
}

func validateTrueTypeFontDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 9.6.3

	logInfoValidate.Println("*** validateTrueTypeFontDict begin ***")

	dictName := "trueTypeFontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, name, required
	_, err = validateNameEntry(xRefTable, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// FirstChar, integer, required.
	required := REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "FirstChar", required, types.V10, nil)
	if err != nil {
		return
	}

	// LastChar, integer, required.
	required = REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LastChar", required, types.V10, nil)
	if err != nil {
		return
	}

	// Widths, array of numbers.
	required = REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Widths", required, types.V10, nil)
	if err != nil {
		return
	}

	// FontDescriptor, dictionary, required
	required = REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	err = validateFontDescriptor(xRefTable, dict, dictName, "TrueType", required, types.V10)
	if err != nil {
		return
	}

	// Encoding: name or dict, optional
	err = validateFontEncoding(xRefTable, dict, dictName, OPTIONAL)
	if err != nil {
		return
	}

	// ToUnicode, stream, optional
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTrueTypeFontDict end ***")

	return
}

func validateCIDToGIDMap(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateCIDToGIDMap begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			err = errors.Errorf("validateCIDToGIDMap: invalid name: %s - must be \"Identity\"\n", s)
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		err = errors.New("validateCIDToGIDMap: corrupt entry")

	}

	logInfoValidate.Println("*** validateCIDToGIDMap end ***")

	return
}

func validateCIDFontGlyphWidths(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateCIDFontGlyphWidths begin ***")

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || arr == nil {
		return
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil || obj == nil {
			return
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing

		case types.PDFArray:
			_, err = validateNumberArray(xRefTable, obj)
			if err != nil {
				return
			}

		default:
			return errors.Errorf("validateCIDFontGlyphWidths: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	logInfoValidate.Println("*** validateCIDFontGlyphWidths end ***")

	return
}

func validateCIDFontDictEntryCIDSystemInfo(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) (err error) {

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateCIDSystemInfoDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	return
}

func validateCIDFontDictEntryCIDToGIDMap(xRefTable *types.XRefTable, dict *types.PDFDict, isCIDFontType2 bool) (err error) {

	if obj, found := dict.Find("CIDToGIDMap"); found {

		if !isCIDFontType2 {
			return errors.New("validateCIDFontDict: entry CIDToGIDMap not allowed - must be CIDFontType2")
		}

		err = validateCIDToGIDMap(xRefTable, obj)
		if err != nil {
			return
		}

	}

	return
}

func validateCIDFontDict(xRefTable *types.XRefTable, fontDict *types.PDFDict) (err error) {

	// see 9.7.4

	logInfoValidate.Println("*** validateCIDFontDict begin ***")

	// Type, name, required
	if fontDict.Type() == nil || *fontDict.Type() != "Font" {
		return errors.New("validateCIDFontDict: missing descendant font type")
	}

	var isCIDFontType2 bool
	var fontType string

	// Subtype, name, required
	subType := fontDict.Subtype()
	if subType == nil || (*subType != "CIDFontType0" && *subType != "CIDFontType2") {
		return errors.New("validateCIDFontDict: missing descendant font subtype")
	}

	isCIDFontType2 = true
	fontType = *subType

	dictName := "CIDFontDict"

	// BaseFont, name, required
	_, err = validateNameEntry(xRefTable, fontDict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// CIDSystemInfo, dict, required
	err = validateCIDFontDictEntryCIDSystemInfo(xRefTable, fontDict, "CIDFontDict")
	if err != nil {
		return
	}

	// FontDescriptor, dictionary, required
	err = validateFontDescriptor(xRefTable, fontDict, dictName, fontType, REQUIRED, types.V10)
	if err != nil {
		return
	}

	// DW, integer, optional
	_, err = validateIntegerEntry(xRefTable, fontDict, dictName, "DW", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// W, array, optional
	err = validateCIDFontGlyphWidths(xRefTable, fontDict, dictName, "W", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// DW2, array, optional
	// An array of two numbers specifying the default metrics for vertical writing.
	_, err = validateNumberArrayEntry(xRefTable, fontDict, dictName, "DW2", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	// W2, array, optional
	err = validateCIDFontGlyphWidths(xRefTable, fontDict, dictName, "W2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// CIDToGIDMap, stream or (name /Identity)
	// optional, Type 2 CIDFonts with embedded associated TrueType font program only.
	err = validateCIDFontDictEntryCIDToGIDMap(xRefTable, fontDict, isCIDFontType2)
	if err != nil {
		return nil
	}

	logInfoValidate.Printf("*** validateCIDFontDict end ***")

	return
}

func validateDescendantFonts(xRefTable *types.XRefTable, fontDict *types.PDFDict, fontDictName string, required bool) (err error) {

	logInfoValidate.Printf("*** validateDescendantFonts begin ***")

	// A one-element array holding a CID font dictionary.

	arr, err := validateArrayEntry(xRefTable, fontDict, fontDictName, "DescendantFonts", required, types.V10, func(arr types.PDFArray) bool { return len(arr) == 1 })
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validateDescendantFonts end")
		return
	}

	dict, err := xRefTable.DereferenceDict((*arr)[0])
	if err != nil {
		return
	}

	if dict == nil {
		if required {
			return errors.Errorf("validateDescendantFonts: dict=%s required descendant font dict missing.\n", fontDictName)
		}
		return
	}

	err = validateCIDFontDict(xRefTable, dict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateDescendantFonts end ***")

	return
}

func validateType0FontDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateType0FontDict begin ***")

	dictName := "type0FontDict"

	// BaseFont, name, required
	_, err = validateNameEntry(xRefTable, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Encoding: name or CMap stream dict, required
	err = validateType0FontEncoding(xRefTable, dict, dictName, REQUIRED)
	if err != nil {
		return
	}

	// DescendantFonts: one-element array specifying the CIDFont dictionary that is the descendant of this Type 0 font, required.
	err = validateDescendantFonts(xRefTable, dict, dictName, REQUIRED)
	if err != nil {
		return
	}

	// ToUnicode, CMap stream dict, optional
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	//if streamDict, ok := writeStreamDictEntry(source, dest, dict, "type0FontDict", "ToUnicode", OPTIONAL, V12, nil); ok {
	//	writeCMapStreamDict(source, dest, *streamDict)
	//}

	logInfoValidate.Println("*** validateType0FontDict end ***")

	return
}

func validateType1FontDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 9.6.2

	logInfoValidate.Println("*** validateType1FontDict begin ***")

	dictName := "type1FontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, name, required
	fontName, err := validateNameEntry(xRefTable, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	required := xRefTable.Version() >= types.V15 || !validateStandardType1Font((*fontName).String())
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = !validateStandardType1Font((*fontName).String())
	}
	// FirstChar, integer, required except for standard 14 fonts. since 1.5 always required.
	fc, err := validateIntegerEntry(xRefTable, dict, dictName, "FirstChar", required, types.V10, nil)
	if err != nil {
		return
	}

	if !required && fc != nil {
		// For the standard 14 fonts, the entries FirstChar, LastChar, Widths and FontDescriptor shall either all be present or all be absent.
		if xRefTable.ValidationMode == types.ValidationStrict {
			required = true
		} else {
			// relaxed: do nothing
		}
	}

	// LastChar, integer, required except for standard 14 fonts. since 1.5 always required.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LastChar", required, types.V10, nil)
	if err != nil {
		return
	}

	// Widths, array of numbers, required except for standard 14 fonts. since 1.5 always required.
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Widths", required, types.V10, nil)
	if err != nil {
		return
	}

	// FontDescriptor, dictionary
	// required since version 1.5; required unless standard font for version < 1.5
	err = validateFontDescriptor(xRefTable, dict, dictName, "Type1", required, types.V10)
	if err != nil {
		return
	}

	// Encoding: name or dict, optional
	err = validateFontEncoding(xRefTable, dict, dictName, OPTIONAL)
	if err != nil {
		return
	}

	// ToUnicode, stream, optional
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType1FontDict end ***")

	return
}

func validateCharProcsDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateCharProcsDict begin ***")

	d, err := validateDictEntry(xRefTable, dict, dictName, "CharProcs", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if d == nil {
		// optional and nil or already written
		logInfoValidate.Println("validateCharProcsDict end")
		return
	}

	for _, v := range d.Dict {

		_, err = xRefTable.DereferenceStreamDict(v)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateCharProcsDict end ***")

	return
}

func validateUseCMapEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool) (err error) {

	logInfoValidate.Println("*** validateUseCMapEntry begin ***")

	entryName := "UseCMap"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateUseCMapEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateUseCMapEntry end: \"%s\" is nil.", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateUseCMapEntry: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateUseCMapEntry end: entry \"%s\" is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStreamDict:
		err = validateCMapStreamDict(xRefTable, &obj)
		if err != nil {
			return
		}

	default:
		err = errors.Errorf("validateUseCMapEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateUseCMapEntry end ***")

	return
}

func validateCIDSystemInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateCIDSystemInfoDict begin ***")

	dictName := "CIDSystemInfoDict"

	// Registry, ASCII string, required
	_, err = validateStringEntry(xRefTable, dict, dictName, "Registry", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Ordering, ASCII string, required
	_, err = validateStringEntry(xRefTable, dict, dictName, "Ordering", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Supplement, integer, required
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Supplement", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateCIDSystemInfoDict end ***")

	return
}

func validateCMapStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) (err error) {

	// See table 120

	logInfoValidate.Println("*** validateCMapStreamDict begin ***")

	// Type, name, required
	if streamDict.Type() == nil || *streamDict.Type() != "CMap" {
		err = errors.Errorf("validateCMapStreamDict: missing required type \"CMap\"")
		return
	}

	dictName := "CMapStreamDict"

	// CMapName, name, required
	_, err = validateNameEntry(xRefTable, &streamDict.PDFDict, dictName, "CMapName", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// CIDFontType0SystemInfo, dict, required
	dict, err := validateDictEntry(xRefTable, &streamDict.PDFDict, dictName, "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if dict != nil {
		err = validateCIDSystemInfoDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	// WMode, integer, optional, 0 or 1
	_, err = validateIntegerEntry(xRefTable, &streamDict.PDFDict, dictName, "WMode", OPTIONAL, types.V10, func(i int) bool { return i == 0 || i == 1 })
	if err != nil {
		return
	}

	// UseCMap, name or cmap stream dict, optional.
	// If present, the referencing CMap shall specify only
	// the character mappings that differ from the referenced CMap.
	err = validateUseCMapEntry(xRefTable, &streamDict.PDFDict, dictName, OPTIONAL)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateCMapStreamDict end ***")

	return
}

func validateType0FontEncoding(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool) (err error) {

	logInfoValidate.Printf("*** validateType0FontEncoding begin: dictName=%s ***", dictName)

	entryName := "Encoding"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateType0FontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateType0FontEncoding end: \"%s\" is nil.\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateType0FontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateType0FontEncoding end: entry \"%s\" is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStreamDict:
		err = validateCMapStreamDict(xRefTable, &obj)
		if err != nil {
			return
		}

	default:
		err = errors.Errorf("validateType0FontEncoding: dict=%s corrupt entry \"Encoding\"\n", dictName)

	}

	logInfoValidate.Println("*** validateType0FontEncoding end ***")

	return
}

func validateType3FontDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 9.6.5

	logInfoValidate.Println("*** validateType3FontDict begin ***")

	dictName := "type3FontDict"

	// Name, name, obsolet and should not be used.

	// FontBBox, rectangle, required
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "FontBBox", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// FontMatrix, number array, required
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "FontMatrix", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return
	}

	// CharProcs, dict, required
	err = validateCharProcsDict(xRefTable, dict, dictName, REQUIRED, types.V10)
	if err != nil {
		return
	}

	// Encoding: name or dict, required
	err = validateFontEncoding(xRefTable, dict, dictName, REQUIRED)
	if err != nil {
		return
	}

	// FirstChar, integer, required.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "FirstChar", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// LastChar, integer, required.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LastChar", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Widths, array of number.
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Widths", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// FontDescriptor, dictionary
	// required since version 1.5 for tagged PDF documents.
	if xRefTable.Tagged {
		err = validateFontDescriptor(xRefTable, dict, dictName, "Type3", REQUIRED, types.V15)
		if err != nil {
			return
		}
	}

	// Resources, dict, optional, since V1.2
	var d *types.PDFDict
	d, err = validateDictEntry(xRefTable, dict, dictName, "Resources", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}
	if d != nil {
		_, err := validateResourceDict(xRefTable, *d)
		if err != nil {
			return err
		}
	}

	// ToUnicode, stream, optional
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType3FontDict end ***")

	return
}

func validateFontDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateFontDict begin ***")

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
		err = errors.Errorf("validateFontDict: unknown Subtype: %s\n", *subtype)

	}

	logInfoValidate.Println("*** validateFontDict end ***")

	return
}

func validateFontResourceDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateFontResourceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateFontResourceDict end: font resource dict is nil.")
		return
	}

	// Iterate over font resource dict
	for _, obj := range dict.Dict {

		dict, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if dict == nil {
			logInfoValidate.Println("validateFontResourceDict end: font dict is nil.")
			continue
		}

		// Process fontDict
		err = validateFontDict(xRefTable, dict)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Println("*** validateFontResourceDict end ***")

	return
}

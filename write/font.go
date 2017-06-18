package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeFontFile(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, fontType string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeFontFile begin: dictName=%s entryName=%s fontType=%s offset=%d ***\n", dictName, entryName, fontType, ctx.Write.Offset)

	streamDict, written, err := writeStreamDictEntry(ctx, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || streamDict == nil {
		// optional and nil or already written
		logInfoWriter.Printf("writeFontFile end: is nil. dictName=%s entryName=%s fontType=%s offset=%d\n", dictName, entryName, fontType, ctx.Write.Offset)
		return
	}

	// Process font file stream dict entries.

	// SubType, required if referenced from FontFile3.
	// TODO since 1.2
	if entryName == "FontFile3" {

		dictSubType := streamDict.Subtype()

		if fontType == "Type1" || fontType == "MMType1" {
			if dictSubType == nil || *dictSubType != "Type1C" {
				return errors.New("writeFontFile: FontFile3 missing Subtype \"Type1C\"")
			}
		}

		if fontType == "CIDFontType0" {
			if dictSubType == nil || *dictSubType != "CIDFontType0C" {
				return errors.New("writeFontFile: FontFile3 missing Subtype \"CIDFontType0C\"")
			}
		}

		if fontType == "OpenType" {
			if dictSubType == nil || *dictSubType != "OpenType" {
				return errors.New("writeFontFile: FontFile3 missing Subtype \"OpenType\"")
			}
		}

	}

	dName := "fontFileStreamDict"
	compactFontFormat := entryName == "FontFile3"

	_, _, err = writeIntegerEntry(ctx, streamDict.PDFDict, dName, "Length1", (fontType == "Type1" || fontType == "TrueType") && !compactFontFormat, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, streamDict.PDFDict, dName, "Length2", fontType == "Type1" && !compactFontFormat, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, streamDict.PDFDict, dName, "Length3", fontType == "Type1" && !compactFontFormat, types.V10, nil)
	if err != nil {
		return
	}

	// Metadata, stream, optional, since 1.4
	_, err = writeMetadata(ctx, streamDict.PDFDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeFontFile end: dictName=%s entryName=%s fontType=%s offset=%d ***\n", dictName, entryName, fontType, ctx.Write.Offset)

	return
}

func writeFontEncoding(ctx *types.PDFContext, dict types.PDFDict, dictName string, required bool) (err error) {

	logInfoWriter.Printf("*** writeFontEncoding begin: dictName=%s offset=%d ***\n", dictName, ctx.Write.Offset)

	entryName := "Encoding"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeFontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFontEncoding end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeFontEncoding end: entry \"%s\" already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeFontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFontEncoding end: entry \"%s\" is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validate.FontEncodingName(s) {
			err = errors.Errorf("writeFontEncoding: invalid Encoding name: %s\n", s)
		}

	case types.PDFDict:
		// no further processing

	default:
		err = errors.Errorf("writeFontEncoding: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeFontEncoding end: \"%s\" is nil, offset=%d ***\n", entryName, ctx.Write.Offset)

	return
}

func writeUseCMapEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, required bool) (err error) {

	logInfoWriter.Printf("*** writeUseCMapEntry begin: offset=%d ***\n", ctx.Write.Offset)

	entryName := "UseCMap"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeUseCMapEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeUseCMapEntry end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeUseCMapEntry end: entry \"%s\" already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeUseCMapEntry: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeUseCMapEntry end: entry \"%s\" is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStreamDict:
		err = writeCMapStreamDict(ctx, obj)
		if err != nil {
			return
		}

	default:
		err = errors.Errorf("writeUseCMapEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeUseCMap end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCIDSystemInfoDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeCIDSystemInfoDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "CIDSystemInfoDict"

	// Registry, ASCII string, required
	_, _, err = writeStringEntry(ctx, dict, dictName, "Registry", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Ordering, ASCII string, required
	_, _, err = writeStringEntry(ctx, dict, dictName, "Ordering", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Supplement, integer, required
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "Supplement", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeCIDSystemInfoDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCMapStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict) (err error) {

	logInfoWriter.Printf("*** writeCMapStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Type, name, required
	if streamDict.Type() == nil || *streamDict.Type() != "CMap" {
		err = errors.Errorf("writeCMapStreamDict: missing required type \"CMap\"")
		return
	}

	dictName := "CMapStreamDict"

	// CMapName, name, required

	_, _, err = writeNameEntry(ctx, streamDict.PDFDict, dictName, "CMapName", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// CIDFontType0SystemInfo, dict, required
	dict, written, err := writeDictEntry(ctx, streamDict.PDFDict, dictName, "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if !written && dict != nil {
		err = writeCIDSystemInfoDict(ctx, *dict)
		if err != nil {
			return
		}
	}

	_, _, err = writeDictEntry(ctx, streamDict.PDFDict, dictName, "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// TODO writeCIDSystemInfoDict

	// WMode, integer, optional, 0 or 1
	_, _, err = writeIntegerEntry(ctx, streamDict.PDFDict, dictName, "WMode", OPTIONAL, types.V10, func(i int) bool { return i == 0 || i == 1 })
	if err != nil {
		return
	}

	// UseCMap, name or cmap stream dict, optional.
	// If present, the referencing CMap shall specify only
	// the character mappings that differ from the referenced CMap.
	err = writeUseCMapEntry(ctx, streamDict.PDFDict, dictName, OPTIONAL)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeCMapStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType0FontEncoding(ctx *types.PDFContext, dict types.PDFDict, dictName string, required bool) (err error) {

	logInfoWriter.Printf("*** writeType0FontEncoding begin: dictName=%s offset=%d ***\n", dictName, ctx.Write.Offset)

	entryName := "Encoding"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeType0FontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeType0FontEncoding end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeType0FontEncoding end: entry \"%s\" already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeType0FontEncoding: dict=%s required entry \"%s\" missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeType0FontEncoding end: entry \"%s\" is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStreamDict:
		err = writeCMapStreamDict(ctx, obj)
		if err != nil {
			return
		}

	default:
		err = errors.Errorf("writeType0FontEncoding: dict=%s corrupt entry \"Encoding\"\n", dictName)

	}

	logInfoWriter.Printf("*** writeType0FontEncoding end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFontDescriptor(ctx *types.PDFContext, fontDict types.PDFDict, fontDictName string, fontDictType string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeFontDescriptor begin: dictName=%s referrer=%s offset=%d ***\n", fontDictName, fontDictType, ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, fontDict, fontDictName, "FontDescriptor", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || dict == nil {
		// optional and nil or already written
		logInfoWriter.Printf("writeFontDescriptor end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Process font descriptor dict

	dictType := dict.Type()
	// relaxed: can be nil

	if !ctx.Valid && dictType == nil {
		logErrorWriter.Printf("writeFontDescriptor: missing entry \"Type\"\n")
	}
	if dictType != nil && *dictType != "FontDescriptor" {
		return errors.New("writeFontDescriptor: corrupt font descriptor dict")
	}

	dictName := "fdDict"

	_, _, err = writeNameEntry(ctx, *dict, dictName, "FontName", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// spec: since 1.5, relaxed since 1.3
	_, _, err = writeStringEntry(ctx, *dict, dictName, "FontFamily", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// spec: since 1.5, relaxed since 1.3
	_, _, err = writeNameEntry(ctx, *dict, dictName, "FontStretch", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// spec: since 1.5, relaxed since 1.3
	_, _, err = writeNumberEntry(ctx, *dict, dictName, "FontWeight", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, *dict, dictName, "Flags", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeRectangleEntry(ctx, *dict, dictName, "FontBBox", fontDictType != "Type3", types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "ItalicAngle", REQUIRED, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "Ascent", fontDictType != "Type3", types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "Descent", fontDictType != "Type3", types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "Leading", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "CapHeight", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "XHeight", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "StemV", fontDictType != "Type3", types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "StemH", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "AvgWidth", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "MaxWidth", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "MissingWidth", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	if fontDictType == "Type1" || fontDictType == "MMType1" {

		err = writeFontFile(ctx, *dict, dictName, "FontFile", fontDictType, OPTIONAL, types.V10)
		if err != nil {
			return
		}

		err = writeFontFile(ctx, *dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V12)
		if err != nil {
			return
		}

	}

	if fontDictType == "TrueType" || fontDictType == "CIDFontType2" {
		err = writeFontFile(ctx, *dict, dictName, "FontFile2", fontDictType, OPTIONAL, types.V11)
		if err != nil {
			return
		}
	}

	if fontDictType == "CIDFontType0" {
		err = writeFontFile(ctx, *dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V13)
		if err != nil {
			return
		}
	}

	if fontDictType == "OpenType" {
		err = writeFontFile(ctx, *dict, dictName, "FontFile3", fontDictType, OPTIONAL, types.V16)
		if err != nil {
			return
		}
	}

	_, _, err = writeStringEntry(ctx, *dict, dictName, "CharSet", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	if fontDictType == "CIDFontType0" || fontDictType == "CIDFontType2" {

		// Style, dict, optional
		_, _, err = writeDictEntry(ctx, *dict, dictName, "Style", OPTIONAL, types.V10, validate.StyleDict)
		if err != nil {
			return
		}

		// Lang, name, optional
		_, _, err = writeNameEntry(ctx, *dict, dictName, "Lang", OPTIONAL, types.V15, nil)
		if err != nil {
			return
		}

		// FD, dict, optional
		_, _, err = writeDictEntry(ctx, *dict, dictName, "FD", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}

		// CIDSet, stream, optional
		_, _, err = writeStreamDictEntry(ctx, *dict, dictName, "CIDSet", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeFontDescriptor end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDescendantFonts(ctx *types.PDFContext, fontDict types.PDFDict, fontDictName string, required bool) (err error) {

	logInfoWriter.Printf("*** writeDescendantFonts begin: offset=%d ***\n", ctx.Write.Offset)

	// A one-element array holding a CID font dictionary.

	arr, written, err := writeArrayEntry(ctx, fontDict, fontDictName, "DescendantFonts", required, types.V10, func(arr types.PDFArray) bool { return len(arr) == 1 })
	if err != nil {
		return
	}

	if written || arr == nil {
		logInfoWriter.Printf("writeDescendantFonts end: offset=%d\n", ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, (*arr)[0])
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeDescendantFonts end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeDescendantFonts: dict=%s required descendant font dict missing.\n", fontDictName)
		}
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.Errorf("writeDescendantFonts: dict=%s corrupt descendant font dict\n", fontDictName)
	}

	err = writeCIDFontDict(ctx, dict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeDescendantFonts end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCIDToGIDMap(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeCIDToGIDMap begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			err = errors.Errorf("writeCIDToGIDMap: invalid name: %s - must be \"Identity\"\n", s)
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		err = errors.New("writeCIDToGIDMap: corrupt entry")

	}

	logInfoWriter.Printf("*** writeCIDToGIDMap end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCIDFontGlyphWidths(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeCIDFontGlyphWidths begin: offset=%d ***\n", ctx.Write.Offset)

	arr, written, err := writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || arr == nil {
		return
	}

	for i, obj := range *arr {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing

		case types.PDFArray:
			_, _, err = writeNumberArray(ctx, obj)
			if err != nil {
				return
			}

		default:
			return errors.Errorf("writeCIDFontGlyphWidths: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	logInfoWriter.Printf("*** writeCIDFontGlyphWidths end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCIDFontDict(ctx *types.PDFContext, fontDict types.PDFDict) (err error) {

	// see 9.7.4

	logInfoWriter.Printf("*** writeCIDFontDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Type, name, required
	if fontDict.Type() == nil || *fontDict.Type() != "Font" {
		return errors.New("writeCIDFontDict: missing descendant font type")
	}

	var isCIDFontType2 bool
	var fontType string

	// Subtype, name, required
	subType := fontDict.Subtype()
	if subType == nil || (*subType != "CIDFontType0" && *subType != "CIDFontType2") {
		return errors.New("writeCIDFontDict: missing descendant font subtype")
	}

	isCIDFontType2 = true
	fontType = *subType

	dictName := "CIDFontDict"

	// BaseFont, name, required
	// TODO Validate
	_, _, err = writeNameEntry(ctx, fontDict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// CIDSystemInfo, dict, required
	dict, written, err := writeDictEntry(ctx, fontDict, "CIDFontDict", "CIDSystemInfo", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if !written && dict != nil {
		err = writeCIDSystemInfoDict(ctx, *dict)
		if err != nil {
			return
		}
	}

	// FontDescriptor, dictionary, required
	err = writeFontDescriptor(ctx, fontDict, dictName, fontType, REQUIRED, types.V10)
	if err != nil {
		return
	}

	// DW, integer, optional
	_, _, err = writeIntegerEntry(ctx, fontDict, dictName, "DW", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// W, array, optional
	err = writeCIDFontGlyphWidths(ctx, fontDict, dictName, "W", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// DW2, array, optional
	// An array of two numbers specifying the default metrics for vertical writing.
	_, _, err = writeNumberArrayEntry(ctx, fontDict, dictName, "DW2", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	// W2, array, optional
	err = writeCIDFontGlyphWidths(ctx, fontDict, dictName, "W2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// CIDToGIDMap, stream or (name /Identity)
	// optional, Type 2 CIDFonts with embedded associated TrueType font program only.
	if pdfObject, found := fontDict.Find("CIDToGIDMap"); found {
		if !isCIDFontType2 {
			return errors.New("writeCIDFontDict: entry CIDToGIDMap not allowed - must be CIDFontType2")
		}
		err = writeCIDToGIDMap(ctx, pdfObject)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeCIDFontDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType0FontDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeType0FontDict begin: offset=%d *** \n", ctx.Write.Offset)

	dictName := "type0FontDict"

	// BaseFont, name, required
	_, _, err = writeNameEntry(ctx, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Encoding: name or CMap stream dict, required
	err = writeType0FontEncoding(ctx, dict, dictName, REQUIRED)
	if err != nil {
		return
	}

	// DescendantFonts: one-element array specifying the CIDFont dictionary that is the descendant of this Type 0 font, required.
	err = writeDescendantFonts(ctx, dict, dictName, REQUIRED)
	if err != nil {
		return
	}

	// ToUnicode, CMap stream dict, optional
	_, _, err = writeStreamDictEntry(ctx, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	//if streamDict, ok := writeStreamDictEntry(ctx, dict, "type0FontDict", "ToUnicode", OPTIONAL, V12, nil); ok {
	//	writeCMapStreamDict(ctx, *streamDict)
	//}

	logInfoWriter.Printf("*** writeType0FontDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType1FontDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 9.6.2

	logInfoWriter.Printf("*** writeType1FontDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type1FontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, name, required
	fontName, _, err := writeNameEntry(ctx, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// TODO check ok..
	//required := source.XRefTable.Version() >= V15 || !isStandardType1Font((*fontName).String())
	// relaxed version:
	required := !validate.StandardType1Font((*fontName).String())

	// FirstChar, integer, required except for standard 14 fonts. since 1.5 always required.
	fc, _, err := writeIntegerEntry(ctx, dict, dictName, "FirstChar", required, types.V10, nil)
	if err != nil {
		return
	}

	if !required && fc != nil {
		// For the standard 14 fonts, the entries FirstChar, LastChar, Widths and FontDescriptor shall either all be present or all be absent.
		// relaxed: do nothing
		// required = true
	}

	// LastChar, integer, required except for standard 14 fonts. since 1.5 always required.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "LastChar", required, types.V10, nil)
	if err != nil {
		return
	}

	// Widths, array of numbers, required except for standard 14 fonts. since 1.5 always required.
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Widths", required, types.V10, nil)
	if err != nil {
		return
	}

	// FontDescriptor, dictionary
	// required since version 1.5; required unless standard font for version < 1.5
	// relaxed = optional
	err = writeFontDescriptor(ctx, dict, dictName, "Type1", required, types.V10)
	if err != nil {
		return
	}

	// Encoding: name or dict, optional
	err = writeFontEncoding(ctx, dict, dictName, OPTIONAL)
	if err != nil {
		return
	}

	// ToUnicode, stream, optional
	_, _, err = writeStreamDictEntry(ctx, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType1FontDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeTrueTypeFontDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 9.6.3

	logInfoWriter.Printf("*** writeTrueTypeFontDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "trueTypeFontDict"

	// Name, name, obsolet and should not be used.

	// BaseFont, name, required
	_, _, err = writeNameEntry(ctx, dict, dictName, "BaseFont", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}
	// TODO check ok..

	// FirstChar, integer, required.
	// Relaxed: OPTIONAL
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "FirstChar", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// LastChar, integer, required.
	// Relaxed: OPTIONAL
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "LastChar", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Widths, array of numbers.
	// Relaxed: OPTIONAL
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Widths", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// FontDescriptor, dictionary, required
	// Relaxed: OPTIONAL
	err = writeFontDescriptor(ctx, dict, dictName, "TrueType", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Encoding: name or dict, optional
	err = writeFontEncoding(ctx, dict, dictName, OPTIONAL)
	if err != nil {
		return
	}

	// ToUnicode, stream, optional
	_, _, err = writeStreamDictEntry(ctx, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeTrueTypeFontDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCharProcsDict(ctx *types.PDFContext, dict types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeCharProcsDict begin: offset=%d ***\n", ctx.Write.Offset)

	d, written, err := writeDictEntry(ctx, dict, dictName, "CharProcs", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || d == nil {
		// optional and nil or already written
		logInfoWriter.Printf("writeCharProcsDict end: offset=%d\n", ctx.Write.Offset)
		return
	}

	var obj interface{}

	for key, value := range d.Dict {

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			return errors.Errorf("writeCharProcsDict: key:%s, value must be indirect reference", key)
		}

		obj, written, err = writeIndRef(ctx, indRef)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		if _, ok := obj.(types.PDFStreamDict); !ok {
			return errors.New("writeCharProcsDict: corrupt stream dict")
		}

	}

	logInfoWriter.Printf("*** writeCharProcsDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType3FontDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 9.6.5

	logInfoWriter.Printf("*** writeType3FontDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type3FontDict"

	// Name, name, obsolet and should not be used.

	// FontBBox, rectangle, required
	_, _, err = writeRectangleEntry(ctx, dict, dictName, "FontBBox", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// FontMatrix, array, required
	// TODO validate array of six numbers. see 9.2.4.
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "FontMatrix", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return
	}

	// CharProcs, dict, required
	err = writeCharProcsDict(ctx, dict, dictName, REQUIRED, types.V10)
	if err != nil {
		return
	}

	// Encoding: name or dict, required
	err = writeFontEncoding(ctx, dict, dictName, REQUIRED)
	if err != nil {
		return
	}

	// FirstChar, integer, required.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "FirstChar", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// LastChar, integer, required.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "LastChar", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Widths, array of number.
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Widths", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// FontDescriptor, dictionary
	// required since version 1.5 for tagged PDF documents.
	if ctx.XRefTable.Tagged {
		err = writeFontDescriptor(ctx, dict, dictName, "Type3", REQUIRED, types.V15)
		if err != nil {
			return
		}
	}

	// Resources, dict, optional, TODO since 1.2
	if obj, ok := dict.Find("Resources"); ok {
		_, err := writeResourceDict(ctx, obj)
		if err != nil {
			return err
		}
	}

	// ToUnicode, stream, optional
	_, _, err = writeStreamDictEntry(ctx, dict, dictName, "ToUnicode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType3FontDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFontDict(ctx *types.PDFContext, fontDict types.PDFDict) (err error) {

	if fontDict.Type() == nil || *fontDict.Type() != "Font" {
		return errors.New("writeFontDict: corrupt font dict")
	}

	subtype := fontDict.Subtype()
	if subtype == nil {
		return errors.New("writeFontDict: missing Subtype")
	}

	switch *subtype {

	case "TrueType":
		err = writeTrueTypeFontDict(ctx, fontDict)

	case "Type0":
		err = writeType0FontDict(ctx, fontDict)

	case "Type1":
		err = writeType1FontDict(ctx, fontDict)

	case "MMType1": // TODO Test
		err = writeType1FontDict(ctx, fontDict)

	case "Type3":
		err = writeType3FontDict(ctx, fontDict)

	default:
		err = errors.Errorf("writeFontDict: unknown Subtype: %s\n", *subtype)

	}

	return
}

func writeFontResourceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeFontResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeFontResourceDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeFontResourceDict end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeFontResourceDict: corrupt dict")
	}

	// Iterate over font resource dict
	for _, obj := range dict.Dict {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written {
			logInfoWriter.Printf("writeFontResourceDict end: resource object already written. offset=%d\n", ctx.Write.Offset)
			continue
		}

		if obj == nil {
			logInfoWriter.Printf("writeFontResourceDict end: resource object is nil. offset=%d\n", ctx.Write.Offset)
			continue
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			return errors.New("writeFontResourceDict: corrupt font dict")
		}

		// Process fontDict
		err = writeFontDict(ctx, dict)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeFontResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func writeTilingPatternDict(ctx *types.PDFContext, streamDict types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeTilingPatternDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeTilingPatternDict: unsupported in version %s.\n", ctx.VersionString())
	}

	dict := streamDict.PDFDict
	dictName := "tilingPatternDict"

	_, _, err = writeNameEntry(ctx, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeRectangleEntry(ctx, dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, dict, dictName, "XStep", REQUIRED, sinceVersion) // TODO validate != 0
	if err != nil {
		return
	}
	_, _, err = writeNumberEntry(ctx, dict, dictName, "YStep", REQUIRED, sinceVersion) // TODO validate != 0
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Matrix", OPTIONAL, sinceVersion, nil) // TODO len(arr) == 6
	if err != nil {
		return
	}

	obj, ok := streamDict.Find("Resources")
	if !ok {
		return errors.New("writeTilingPatternDict: missing required entry Resources")
	}

	_, err = writeResourceDict(ctx, obj)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeTilingPatternDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeShadingPatternDict(ctx *types.PDFContext, dict types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeShadingPatternDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeShadingPatternDict: unsupported in version %s.\n", ctx.VersionString())
	}

	dictName := "shadingPatternDict"

	_, _, err = writeNameEntry(ctx, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Matrix", OPTIONAL, sinceVersion, nil) // TODO len(arr) == 6
	if err != nil {
		return
	}

	d, written, err := writeDictEntry(ctx, dict, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeExtGStateDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// Shading: required, dict or stream dict.
	obj, ok := dict.Find("Shading")
	if !ok {
		return errors.Errorf("writeShadingPatternDict: missing required entry \"Shading\".")
	}

	err = writeShading(ctx, obj)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeShadingPatternDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePattern(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writePattern begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePattern end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writePattern end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writeShadingPatternDict(ctx, obj, types.V13)

	case types.PDFStreamDict:
		err = writeTilingPatternDict(ctx, obj, types.V10)

	default:
		err = errors.New("writePattern: corrupt obj typ, must be dict or stream dict")

	}

	logInfoWriter.Printf("*** writePattern end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePatternResourceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	// see 8.7 Patterns

	logInfoWriter.Printf("*** writePatternResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePatternResourceDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writePatternResourceDict end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writePatternResourceDict: corrupt dict")
	}

	// Iterate over pattern resource dictionary
	for key, obj := range dict.Dict {

		logInfoWriter.Printf("writePatternResourceDict: processing entry: %s\n", key)

		// Process pattern
		err = writePattern(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writePatternResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

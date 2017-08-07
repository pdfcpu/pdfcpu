package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateTilingPatternDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateTilingPatternDict begin ***")

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateTilingPatternDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dict := streamDict.PDFDict
	dictName := "tilingPatternDict"

	_, err = validateNameEntry(xRefTable, &dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateRectangleEntry(xRefTable, &dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, &dict, dictName, "XStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return
	}
	_, err = validateNumberEntry(xRefTable, &dict, dictName, "YStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 6 })
	if err != nil {
		return
	}

	obj, ok := streamDict.Find("Resources")
	if !ok {
		return errors.New("validateTilingPatternDict: missing required entry Resources")
	}

	_, err = validateResourceDict(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTilingPatternDict end ***")

	return
}

func validateShadingPatternDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateShadingPatternDict begin ***")

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateShadingPatternDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "shadingPatternDict"

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 6 })
	if err != nil {
		return
	}

	d, err := validateDictEntry(xRefTable, dict, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateExtGStateDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// Shading: required, dict or stream dict.
	obj, ok := dict.Find("Shading")
	if !ok {
		return errors.Errorf("validateShadingPatternDict: missing required entry \"Shading\".")
	}

	err = validateShading(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateShadingPatternDict end ***")

	return
}

func validatePattern(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validatePattern begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validatePattern end: object is nil.")
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateShadingPatternDict(xRefTable, &obj, types.V13)

	case types.PDFStreamDict:
		err = validateTilingPatternDict(xRefTable, &obj, types.V10)

	default:
		err = errors.New("validatePattern: corrupt obj typ, must be dict or stream dict")

	}

	logInfoValidate.Println("*** validatePattern end ***")

	return
}

func validatePatternResourceDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// see 8.7 Patterns

	logInfoValidate.Println("*** validatePatternResourceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validatePatternResourceDict end: object is nil.")
		return
	}

	// Iterate over pattern resource dictionary
	for key, obj := range dict.Dict {

		logInfoValidate.Printf("validatePatternResourceDict: processing entry: %s\n", key)

		// Process pattern
		err = validatePattern(xRefTable, obj)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validatePatternResourceDict end ***")

	return
}

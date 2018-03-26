package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateTilingPatternDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, sinceVersion types.PDFVersion) error {

	dictName := "tilingPatternDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	dict := streamDict.PDFDict

	_, err = validateNameEntry(xRefTable, &dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, &dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, &dict, dictName, "XStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return err
	}
	_, err = validateNumberEntry(xRefTable, &dict, dictName, "YStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	obj, ok := streamDict.Find("Resources")
	if !ok {
		return errors.New("validateTilingPatternDict: missing required entry Resources")
	}

	_, err = validateResourceDict(xRefTable, obj)

	return err
}

func validateShadingPatternDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "shadingPatternDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	d, err := validateDictEntry(xRefTable, dict, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateExtGStateDict(xRefTable, *d)
		if err != nil {
			return err
		}
	}

	// Shading: required, dict or stream dict.
	obj, ok := dict.Find("Shading")
	if !ok {
		return errors.Errorf("validateShadingPatternDict: missing required entry \"Shading\".")
	}

	return validateShading(xRefTable, obj)
}

func validatePattern(xRefTable *types.XRefTable, obj types.PDFObject) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateShadingPatternDict(xRefTable, &obj, types.V13)

	case types.PDFStreamDict:
		err = validateTilingPatternDict(xRefTable, &obj, types.V10)

	default:
		err = errors.New("validatePattern: corrupt obj typ, must be dict or stream dict")

	}

	return err
}

func validatePatternResourceDict(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 8.7 Patterns

	// Version check
	err := xRefTable.ValidateVersion("PatternResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Iterate over pattern resource dictionary
	for _, obj := range dict.Dict {

		// Process pattern
		err = validatePattern(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

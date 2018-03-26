package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdfcpu/types"
)

// see 7.10 Functions

func validateExponentialInterpolationFunctionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "exponentialInterpolationFunctionDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, types.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Range", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C0", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C1", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "N", REQUIRED, types.V13, nil)

	return err
}

func validateStitchingFunctionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "stitchingFunctionDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, types.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Range", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateFunctionArrayEntry(xRefTable, dict, dictName, "Functions", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Bounds", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Encode", REQUIRED, types.V13, nil)

	return err
}

func validateSampledFunctionStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict) error {

	dictName := "sampledFunctionStreamDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, types.V12)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Domain", REQUIRED, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Range", REQUIRED, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerArrayEntry(xRefTable, &dict.PDFDict, dictName, "Size", REQUIRED, types.V12, nil)
	if err != nil {
		return err
	}

	validate := func(i int) bool { return intMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32}) }
	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "BitsPerSample", REQUIRED, types.V12, validate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Order", OPTIONAL, types.V12, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Encode", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Decode", OPTIONAL, types.V12, nil)

	return err
}

func validatePostScriptCalculatorFunctionStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict) error {

	dictName := "postScriptCalculatorFunctionStreamDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, types.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Range", REQUIRED, types.V13, nil)

	return err
}

func processFunctionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	funcType, err := validateIntegerEntry(xRefTable, dict, "functionDict", "FunctionType", REQUIRED, types.V10, func(i int) bool { return i == 2 || i == 3 })
	if err != nil {
		return err
	}

	switch *funcType {

	case 2:
		err = validateExponentialInterpolationFunctionDict(xRefTable, dict)

	case 3:
		err = validateStitchingFunctionDict(xRefTable, dict)

	}

	return err
}

func processFunctionStreamDict(xRefTable *types.XRefTable, sd *types.PDFStreamDict) error {

	funcType, err := validateIntegerEntry(xRefTable, &sd.PDFDict, "functionDict", "FunctionType", REQUIRED, types.V10, func(i int) bool { return i == 0 || i == 4 })
	if err != nil {
		return err
	}

	switch *funcType {
	case 0:
		err = validateSampledFunctionStreamDict(xRefTable, sd)

	case 4:
		err = validatePostScriptCalculatorFunctionStreamDict(xRefTable, sd)

	}

	return err
}

func processFunction(xRefTable *types.XRefTable, obj types.PDFObject) (err error) {

	// Function dict: dict or stream dict with required entry "FunctionType" (integer):
	// 0: Sampled function (stream dict)
	// 2: Exponential interpolation function (dict)
	// 3: Stitching function (dict)
	// 4: PostScript calculator function (stream dict), since V1.3

	switch obj := obj.(type) {

	case types.PDFDict:

		// process function  2,3
		err = processFunctionDict(xRefTable, &obj)

	case types.PDFStreamDict:

		// process function  0,4
		err = processFunctionStreamDict(xRefTable, &obj)

	default:
		return errors.New("processFunction: obj must be dict or stream dict")
	}

	return err
}

func validateFunction(xRefTable *types.XRefTable, obj types.PDFObject) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		return errors.New("writeFunction: missing object")
	}

	return processFunction(xRefTable, obj)
}

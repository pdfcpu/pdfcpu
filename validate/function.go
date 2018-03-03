package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdfcpu/types"
)

// see 7.10 Functions

func validateExponentialInterpolationFunctionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateExponentialInterpolationFunctionDict begin ***")

	// Version check
	if xRefTable.Version() < types.V13 {
		return errors.Errorf("validateExponentialInterpolationFunctionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	_, err := validateNumberArrayEntry(xRefTable, dict, "functionDict", "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, "functionDict", "Range", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, "functionDict", "C0", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, "functionDict", "C1", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, "functionDict", "N", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateExponentialInterpolationFunctionDict end ***")

	return nil
}

func validateStitchingFunctionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Printf("validateStitchingFunctionDict begin ***")

	// Version check
	if xRefTable.Version() < types.V13 {
		return errors.Errorf("validateStitchingFunctionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	_, err := validateNumberArrayEntry(xRefTable, dict, "functionDict", "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, "functionDict", "Range", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateFunctionArrayEntry(xRefTable, dict, "functionDict", "Functions", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, "functionDict", "Bounds", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, "functionDict", "Encode", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateStitchingFunctionDict end ***")

	return nil
}

func validateSampledFunctionStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict) error {

	logInfoValidate.Printf("*** validateSampledFunctionStreamDict begin ***")

	// Version check
	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateSampledFunctionStreamDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	_, err := validateNumberArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Domain", REQUIRED, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Range", REQUIRED, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Size", REQUIRED, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, "functionDict", "BitsPerSample", REQUIRED, types.V12, validateBitsPerSample)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, "functionDict", "Order", OPTIONAL, types.V12, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Encode", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Decode", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateSampledFunctionStreamDict end ***")

	return nil
}

func validatePostScriptCalculatorFunctionStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict) error {

	logInfoValidate.Println("*** validatePostScriptCalculatorFunctionStreamDict begin ***")

	// Version check
	if xRefTable.Version() < types.V13 {
		return errors.Errorf("validatePostScriptCalculatorFunctionStreamDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	_, err := validateNumberArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, "functionDict", "Range", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validatePostScriptCalculatorFunctionStreamDict end ***")

	return nil
}

func processFunctionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	var funcType *types.PDFInteger

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

	var funcType *types.PDFInteger
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

func processFunction(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Printf("*** processFunction begin ***")

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

	logInfoValidate.Printf("*** processFunction end ***")

	return err
}

func validateFunction(xRefTable *types.XRefTable, obj interface{}) error {

	logInfoValidate.Printf("*** writeFunction begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}

	if obj == nil {
		return errors.New("writeFunction: missing object")
	}

	err = processFunction(xRefTable, obj)

	logInfoValidate.Printf("*** writeFunction end ***")

	return err
}

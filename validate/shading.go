package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateShadingDictCommonEntries(xRefTable *types.XRefTable, dict *types.PDFDict) (shadType int, err error) {

	logInfoValidate.Printf("validateShadingDictCommonEntries begin ***")

	dictName := "shadingDictCommonEntries"

	shadingType, err := validateIntegerEntry(xRefTable, dict, dictName, "ShadingType", REQUIRED, types.V10, func(i int) bool { return i >= 1 && i <= 7 })
	if err != nil {
		return 0, err
	}

	err = validateColorSpaceEntry(xRefTable, dict, dictName, "ColorSpace", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return 0, err
	}

	_, err = validateArrayEntry(xRefTable, dict, dictName, "Background", OPTIONAL, types.V10, nil)
	if err != nil {
		return 0, err
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "BBox", OPTIONAL, types.V10, nil)
	if err != nil {
		return 0, err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AntiAlias", OPTIONAL, types.V10, nil)
	if err != nil {
		return 0, err
	}

	logInfoValidate.Println("*** validateShadingDictCommonEntries end ***")

	return shadingType.Value(), nil
}

func validateFunctionBasedShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateFunctionBasedShadingDict begin ***")

	dictName := "functionBasedShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateFunctionBasedShadingDict end ***")

	return nil
}

func validateAxialShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateAxialShadingDict begin ***")

	dictName := "axialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Coords", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	_, err = validateBooleanArrayEntry(xRefTable, dict, dictName, "Extend", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateAxialShadingDict end ***")

	return nil
}

func validateRadialShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateRadialShadingDict begin ***")

	dictName := "radialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Coords", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	_, err = validateBooleanArrayEntry(xRefTable, dict, dictName, "Extend", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateRadialShadingDict end ***")

	return nil
}

func validateShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// Shading 1-3

	logInfoValidate.Println("*** validateShadingDict begin ***")

	shadingType, err := validateShadingDictCommonEntries(xRefTable, dict)
	if err != nil {
		return err
	}

	switch shadingType {
	case 1:
		err = validateFunctionBasedShadingDict(xRefTable, dict)

	case 2:
		err = validateAxialShadingDict(xRefTable, dict)

	case 3:
		err = validateRadialShadingDict(xRefTable, dict)

	default:
		return errors.Errorf("validateShadingDict: unexpected shadingType: %d\n", shadingType)
	}

	logInfoValidate.Println("*** validateShadingDict end ***")

	return err
}

func validateFreeFormGouroudShadedTriangleMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateFreeFormGouroudShadedTriangleMeshesDict begin ***")

	dictName := "freeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, types.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateFreeFormGouroudShadedTriangleMeshesDict end ***")

	return nil
}

func validateLatticeFormGouraudShadedTriangleMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateLatticeFormGouraudShadedTriangleMeshesDict begin ***")

	dictName := "latticeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "VerticesPerRow", REQUIRED, types.V10, func(i int) bool { return i >= 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateLatticeFormGouraudShadedTriangleMeshesDict end ***")

	return nil
}

func validateCoonsPatchMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateCoonsPatchMeshesDict begin: ***")

	dictName := "coonsPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	validateBitsPerFlag := func(i int) bool { return i >= 0 && i <= 3 }
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		validateBitsPerFlag = func(i int) bool { return i >= 0 && i <= 8 }
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, types.V10, validateBitsPerFlag)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateCoonsPatchMeshesDict end ***")

	return nil
}

func validateTensorProductPatchMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateTensorProductPatchMeshesDict begin ***")

	dictName := "tensorProductPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	validateBitsPerFlag := func(i int) bool { return i >= 0 && i <= 3 }
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		validateBitsPerFlag = func(i int) bool { return i >= 0 && i <= 8 }
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, types.V10, validateBitsPerFlag)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateTensorProductPatchMeshesDict end ***")

	return nil
}

func validateShadingStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) error {

	// Shading 4-7

	logInfoValidate.Println("*** validateShadingStreamDict begin ***")

	dict := streamDict.PDFDict

	shadingType, err := validateShadingDictCommonEntries(xRefTable, &dict)
	if err != nil {
		return err
	}

	switch shadingType {

	case 4:
		err = validateFreeFormGouroudShadedTriangleMeshesDict(xRefTable, &dict)

	case 5:
		err = validateLatticeFormGouraudShadedTriangleMeshesDict(xRefTable, &dict)

	case 6:
		err = validateCoonsPatchMeshesDict(xRefTable, &dict)

	case 7:
		err = validateTensorProductPatchMeshesDict(xRefTable, &dict)

	default:
		return errors.Errorf("validateShadingStreamDict: unexpected shadingType: %d\n", shadingType)
	}

	logInfoValidate.Println("*** validateShadingStreamDict end ***")

	return err
}

func validateShading(xRefTable *types.XRefTable, obj interface{}) error {

	// see 8.7.4.3 Shading Dictionaries

	logInfoValidate.Println("*** validateShading begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Println("validateShading end: object is nil.")
		return nil
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateShadingDict(xRefTable, &obj)

	case types.PDFStreamDict:
		err = validateShadingStreamDict(xRefTable, &obj)

	default:
		return errors.New("validateShading: corrupt obj typ, must be dict or stream dict")

	}

	logInfoValidate.Println("*** validateShading end ***")

	return err
}

func validateShadingResourceDict(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 8.7.4.3 Shading Dictionaries

	logInfoValidate.Println("*** validateShadingResourceDict begin ***")

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateShadingResourceDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if dict == nil {
		logInfoValidate.Println("*** validateShadingResourceDict end: object is nil. ***")
		return nil
	}

	// Iterate over shading resource dictionary
	for key, obj := range dict.Dict {

		logInfoValidate.Printf("validateShadingResourceDict: processing entry: %s\n", key)

		// Process shading
		err = validateShading(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateShadingResourceDict end ***")

	return nil
}

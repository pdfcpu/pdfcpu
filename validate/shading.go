package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateBitsPerComponent(i int) bool {
	return intMemberOf(i, []int{1, 2, 4, 8, 12, 16})
}

func validateBitsPerCoordinate(i int) bool {
	return intMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32})
}

func validateShadingDictCommonEntries(xRefTable *types.XRefTable, dict *types.PDFDict) (shadType int, err error) {

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

	return shadingType.Value(), err
}

func validateFunctionBasedShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "functionBasedShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, types.V10)
}

func validateAxialShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return err
}

func validateRadialShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return err
}

func validateShadingDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// Shading 1-3

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

	return err
}

func validateFreeFormGouroudShadedTriangleMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
}

func validateLatticeFormGouraudShadedTriangleMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
}

func validateCoonsPatchMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
}

func validateTensorProductPatchMeshesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, types.V10)
}

func validateShadingStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) error {

	// Shading 4-7

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

	return err
}

func validateShading(xRefTable *types.XRefTable, obj types.PDFObject) error {

	// see 8.7.4.3 Shading Dictionaries

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateShadingDict(xRefTable, &obj)

	case types.PDFStreamDict:
		err = validateShadingStreamDict(xRefTable, &obj)

	default:
		return errors.New("validateShading: corrupt obj typ, must be dict or stream dict")

	}

	return err
}

func validateShadingResourceDict(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 8.7.4.3 Shading Dictionaries

	// Version check
	err := xRefTable.ValidateVersion("shadingResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Iterate over shading resource dictionary
	for _, obj := range dict.Dict {

		// Process shading
		err = validateShading(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

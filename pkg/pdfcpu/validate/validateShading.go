/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateBitsPerComponent(i int) bool {
	return pdf.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16})
}

func validateBitsPerCoordinate(i int) bool {
	return pdf.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32})
}

func validateShadingDictCommonEntries(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) (shadType int, err error) {

	dictName := "shadingDictCommonEntries"

	shadingType, err := validateIntegerEntry(xRefTable, dict, dictName, "ShadingType", REQUIRED, pdf.V10, func(i int) bool { return i >= 1 && i <= 7 })
	if err != nil {
		return 0, err
	}

	err = validateColorSpaceEntry(xRefTable, dict, dictName, "ColorSpace", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return 0, err
	}

	_, err = validateArrayEntry(xRefTable, dict, dictName, "Background", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return 0, err
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "BBox", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return 0, err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AntiAlias", OPTIONAL, pdf.V10, nil)

	return shadingType.Value(), err
}

func validateFunctionBasedShadingDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "functionBasedShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 4 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, pdf.V10)
}

func validateAxialShadingDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "axialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Coords", REQUIRED, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 4 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	_, err = validateBooleanArrayEntry(xRefTable, dict, dictName, "Extend", OPTIONAL, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 2 })

	return err
}

func validateRadialShadingDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "radialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Coords", REQUIRED, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	_, err = validateBooleanArrayEntry(xRefTable, dict, dictName, "Extend", OPTIONAL, pdf.V10, func(arr pdf.Array) bool { return len(arr) == 2 })

	return err
}

func validateShadingDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

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

func validateFreeFormGouroudShadedTriangleMeshesDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "freeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, pdf.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, pdf.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, pdf.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, pdf.V10)
}

func validateLatticeFormGouraudShadedTriangleMeshesDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "latticeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, pdf.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, pdf.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "VerticesPerRow", REQUIRED, pdf.V10, func(i int) bool { return i >= 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, pdf.V10)
}

func validateCoonsPatchMeshesDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "coonsPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, pdf.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, pdf.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	validateBitsPerFlag := func(i int) bool { return i >= 0 && i <= 3 }
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		validateBitsPerFlag = func(i int) bool { return i >= 0 && i <= 8 }
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, pdf.V10, validateBitsPerFlag)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, pdf.V10)
}

func validateTensorProductPatchMeshesDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "tensorProductPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, pdf.V10, validateBitsPerCoordinate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, pdf.V10, validateBitsPerComponent)
	if err != nil {
		return err
	}

	validateBitsPerFlag := func(i int) bool { return i >= 0 && i <= 3 }
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		validateBitsPerFlag = func(i int) bool { return i >= 0 && i <= 8 }
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, pdf.V10, validateBitsPerFlag)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	return validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, pdf.V10)
}

func validateShadingStreamDict(xRefTable *pdf.XRefTable, streamDict *pdf.StreamDict) error {

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

func validateShading(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	// see 8.7.4.3 Shading Dictionaries

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.PDFDict:
		err = validateShadingDict(xRefTable, &obj)

	case pdf.StreamDict:
		err = validateShadingStreamDict(xRefTable, &obj)

	default:
		return errors.New("validateShading: corrupt obj typ, must be dict or stream dict")

	}

	return err
}

func validateShadingResourceDict(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

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

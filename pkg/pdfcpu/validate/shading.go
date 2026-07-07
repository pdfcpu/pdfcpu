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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateBitsPerComponent(i int) bool {
	return types.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16})
}

func validateBitsPerCoordinate(i int) bool {
	return types.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32})
}

func validateBitsPerFlag(i int) bool {
	return types.IntMemberOf(i, []int{2, 4, 8})
}

func validateShadingDictCommonEntries(xRefTable *model.XRefTable, dict types.Dict) (shadType int, err error) {
	dictName := "shadingDictCommonEntries"

	shadingType, err := validateIntegerEntry(xRefTable, dict, dictName, "ShadingType", REQUIRED, model.V10, func(i int) bool { return i >= 1 && i <= 7 })
	if err != nil {
		return 0, fmt.Errorf("%s.ShadingType: %w", dictName, err)
	}

	err = validateColorSpaceEntry(xRefTable, dict, dictName, "ColorSpace", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return 0, fmt.Errorf("%s.ColorSpace: %w", dictName, err)
	}

	_, err = validateArrayEntry(xRefTable, dict, dictName, "Background", OPTIONAL, model.V10, nil)
	if err != nil {
		return 0, fmt.Errorf("%s.Background: %w", dictName, err)
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "BBox", OPTIONAL, model.V10, nil)
	if err != nil {
		return 0, fmt.Errorf("%s.BBox: %w", dictName, err)
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AntiAlias", OPTIONAL, model.V10, nil)
	if err != nil {
		return 0, fmt.Errorf("%s.AntiAlias: %w", dictName, err)
	}

	return shadingType.Value(), nil
}

func validateFunctionBasedShadingDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "functionBasedShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateAxialShadingDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "axialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Coords", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return fmt.Errorf("%s.Coords: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, model.V10)
	if err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}

	_, err = validateBooleanArrayEntry(xRefTable, dict, dictName, "Extend", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Extend: %w", dictName, err)
	}

	return nil
}

func validateRadialShadingDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "radialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Coords", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Coords: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", REQUIRED, model.V10)
	if err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}

	_, err = validateBooleanArrayEntry(xRefTable, dict, dictName, "Extend", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Extend: %w", dictName, err)
	}

	return nil
}

func validateShadingDict(xRefTable *model.XRefTable, dict types.Dict) error {
	// Shading 1-3

	shadingType, err := validateShadingDictCommonEntries(xRefTable, dict)
	if err != nil {
		return fmt.Errorf("shading dict: %w", err)
	}

	switch shadingType {
	case 1:
		err = validateFunctionBasedShadingDict(xRefTable, dict)

	case 2:
		err = validateAxialShadingDict(xRefTable, dict)

	case 3:
		err = validateRadialShadingDict(xRefTable, dict)

	default:
		return fmt.Errorf("unexpected shadingType: %d", shadingType)
	}

	if err != nil {
		return fmt.Errorf("shading dict type %d: %w", shadingType, err)
	}
	return nil
}

func validateFreeFormGouroudShadedTriangleMeshesDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "freeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateLatticeFormGouraudShadedTriangleMeshesDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "latticeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "VerticesPerRow", REQUIRED, model.V10, func(i int) bool { return i >= 2 })
	if err != nil {
		return fmt.Errorf("%s.VerticesPerRow: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateCoonsPatchMeshesDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "coonsPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateTensorProductPatchMeshesDict(xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "tensorProductPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(xRefTable, dict, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateShadingStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	// Shading 2, 4-7

	dict := sd.Dict

	shadingType, err := validateShadingDictCommonEntries(xRefTable, dict)
	if err != nil {
		return fmt.Errorf("shading stream dict: %w", err)
	}

	switch shadingType {

	case 2:
		err = validateAxialShadingDict(xRefTable, dict)

	case 4:
		err = validateFreeFormGouroudShadedTriangleMeshesDict(xRefTable, dict)

	case 5:
		err = validateLatticeFormGouraudShadedTriangleMeshesDict(xRefTable, dict)

	case 6:
		err = validateCoonsPatchMeshesDict(xRefTable, dict)

	case 7:
		err = validateTensorProductPatchMeshesDict(xRefTable, dict)

	default:
		return fmt.Errorf("unexpected shadingType: %d", shadingType)
	}

	if err != nil {
		return fmt.Errorf("shading stream dict type %d: %w", shadingType, err)
	}
	return nil
}

func validateShading(xRefTable *model.XRefTable, obj types.Object) error {
	// see 8.7.4.3 Shading Dictionaries

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		if err != nil {
			return fmt.Errorf("shading: dereference: %w", err)
		}
		return nil
	}

	switch obj := obj.(type) {

	case types.Dict:
		if err = validateShadingDict(xRefTable, obj); err != nil {
			return fmt.Errorf("shading dictionary: %w", err)
		}

	case types.StreamDict:
		if err = validateShadingStreamDict(xRefTable, &obj); err != nil {
			return fmt.Errorf("shading stream: %w", err)
		}

	default:
		return fmt.Errorf("shading: expected dict or stream dict, got %T", obj)

	}

	return nil
}

func validateShadingResourceDict(xRefTable *model.XRefTable, obj types.Object, sinceVersion model.Version) error {
	// see 8.7.4.3 Shading Dictionaries

	// Version check
	err := xRefTable.ValidateVersion("shadingResourceDict", sinceVersion)
	if err != nil {
		return fmt.Errorf("shadingResourceDict: %w", err)
	}

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil || d == nil {
		if err != nil {
			return fmt.Errorf("shadingResourceDict: dereference: %w", err)
		}
		return nil
	}

	// Iterate over shading resource dictionary
	for name, obj := range d {
		// Process shading
		err = validateShading(xRefTable, obj)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("shadingResourceDict.%s", name), obj), err)
		}
	}

	return nil
}

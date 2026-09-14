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
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
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

func validateShadingDictCommonEntries(c context.Context, xRefTable *model.XRefTable, dict types.Dict) (shadType int, err error) {
	dictName := "shadingDictCommonEntries"

	shadingType, err := validateIntegerEntry(xRefTable, dict, 0, dictName, "ShadingType", REQUIRED, model.V10, func(i int) bool { return i >= 1 && i <= 7 })
	if err != nil {
		return 0, fmt.Errorf("%s.ShadingType: %w", dictName, err)
	}

	err = validateColorSpaceEntry(c, xRefTable, dict, 0, dictName, "ColorSpace", OPTIONAL, colorSpaceNoPattern)
	if err != nil {
		return 0, fmt.Errorf("%s.ColorSpace: %w", dictName, err)
	}

	_, err = validateArrayEntry(xRefTable, dict, 0, dictName, "Background", OPTIONAL, model.V10, nil)
	if err != nil {
		return 0, fmt.Errorf("%s.Background: %w", dictName, err)
	}

	_, err = validateRectangleEntry(xRefTable, dict, 0, dictName, "BBox", OPTIONAL, model.V10, nil)
	if err != nil {
		return 0, fmt.Errorf("%s.BBox: %w", dictName, err)
	}

	_, err = validateBooleanEntry(xRefTable, dict, 0, dictName, "AntiAlias", OPTIONAL, model.V10, nil)
	if err != nil {
		return 0, fmt.Errorf("%s.AntiAlias: %w", dictName, err)
	}

	return shadingType.Value(), nil
}

func validateFunctionBasedShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "functionBasedShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Matrix", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", REQUIRED, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateAxialShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr int) error {
	dictName := "axialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Coords", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return fmt.Errorf("%s.Coords: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", REQUIRED, model.V10)
	if err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}

	_, err = validateBooleanArrayEntry(
		xRefTable, dict, ownerObjNr, dictName, "Extend", OPTIONAL, model.V10,
		func(a types.Array) bool { return len(a) == 2 },
	)
	if err != nil {
		return fmt.Errorf("%s.Extend: %w", dictName, err)
	}

	return nil
}

func validateRadialShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr int) error {
	dictName := "radialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Coords", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Coords: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", REQUIRED, model.V10)
	if err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}

	_, err = validateBooleanArrayEntry(
		xRefTable, dict, ownerObjNr, dictName, "Extend", OPTIONAL, model.V10,
		func(a types.Array) bool { return len(a) == 2 },
	)
	if err != nil {
		return fmt.Errorf("%s.Extend: %w", dictName, err)
	}

	return nil
}

func validateShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr int) error {
	// Shading 1-3

	shadingType, err := validateShadingDictCommonEntries(c, xRefTable, dict)
	if err != nil {
		return fmt.Errorf("shading dict: %w", err)
	}

	switch shadingType {
	case 1:
		err = validateFunctionBasedShadingDict(c, xRefTable, dict)

	case 2:
		err = validateAxialShadingDict(c, xRefTable, dict, ownerObjNr)

	case 3:
		err = validateRadialShadingDict(c, xRefTable, dict, ownerObjNr)

	default:
		return fmt.Errorf("unexpected shadingType: %d", shadingType)
	}

	if err != nil {
		return fmt.Errorf("shading dict type %d: %w", shadingType, err)
	}
	return nil
}

func validateFreeFormGouroudShadedTriangleMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "freeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateLatticeFormGouraudShadedTriangleMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "latticeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "VerticesPerRow", REQUIRED, model.V10, func(i int) bool { return i >= 2 })
	if err != nil {
		return fmt.Errorf("%s.VerticesPerRow: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateCoonsPatchMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "coonsPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateTensorProductPatchMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict) error {
	dictName := "tensorProductPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, 0, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, 0, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	if err = validateFunctionOrArrayOfFunctionsEntry(c, xRefTable, dict, 0, dictName, "Function", OPTIONAL, model.V10); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateShadingStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, ownerObjNr int) error {
	// Shading 2, 4-7

	dict := sd.Dict

	shadingType, err := validateShadingDictCommonEntries(c, xRefTable, dict)
	if err != nil {
		return fmt.Errorf("shading stream dict: %w", err)
	}

	switch shadingType {

	case 2:
		err = validateAxialShadingDict(c, xRefTable, dict, ownerObjNr)

	case 4:
		err = validateFreeFormGouroudShadedTriangleMeshesDict(c, xRefTable, dict)

	case 5:
		err = validateLatticeFormGouraudShadedTriangleMeshesDict(c, xRefTable, dict)

	case 6:
		err = validateCoonsPatchMeshesDict(c, xRefTable, dict)

	case 7:
		err = validateTensorProductPatchMeshesDict(c, xRefTable, dict)

	default:
		return fmt.Errorf("unexpected shadingType: %d", shadingType)
	}

	if err != nil {
		return fmt.Errorf("shading stream dict type %d: %w", shadingType, err)
	}
	return nil
}

func validateShading(c context.Context, xRefTable *model.XRefTable, obj types.Object) (err error) {
	// see 8.7.4.3 Shading Dictionaries
	objNr := validationObjectNumber(0, obj)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		if err != nil {
			return fmt.Errorf("shading: dereference: %w", err)
		}
		return nil
	}

	switch obj := obj.(type) {

	case types.Dict:
		if err = validateShadingDict(c, xRefTable, obj, objNr); err != nil {
			return fmt.Errorf("shading dictionary: %w", err)
		}

	case types.StreamDict:
		if err = validateShadingStreamDict(c, xRefTable, &obj, objNr); err != nil {
			return fmt.Errorf("shading stream: %w", err)
		}

	default:
		return fmt.Errorf("shading: expected dict or stream dict, got %T", obj)

	}

	return nil
}

func validateShadingResourceDict(c context.Context, xRefTable *model.XRefTable, obj types.Object, sinceVersion model.Version) (err error) {
	// see 8.7.4.3 Shading Dictionaries
	objNr := validationObjectNumber(0, obj)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	version := sinceVersion
	if xRefTable.ValidationMode == model.ValidationRelaxed && xRefTable.Version() < sinceVersion {
		version = model.V12
	}

	// Version check
	err = xRefTable.ValidateVersion("shadingResourceDict", version)
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
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		obj := d[name]
		// Process shading
		err = validateShading(c, xRefTable, obj)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("shadingResourceDict.%s", name), obj), err)
		}
	}

	if version < sinceVersion {
		showDigestedVersionViolation(xRefTable, "shadingResourceDict")
	}

	return nil
}

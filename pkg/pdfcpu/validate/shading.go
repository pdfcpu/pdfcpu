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

type shadingInfo struct {
	kind       int
	components int
}

func validateShadingDictCommonEntries(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr int) (info shadingInfo, err error) {
	dictName := "shadingDictCommonEntries"

	shadingType, err := validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "ShadingType", REQUIRED, model.V10, func(i int) bool { return i >= 1 && i <= 7 })
	if err != nil {
		return info, fmt.Errorf("%s.ShadingType: %w", dictName, err)
	}

	err = validateColorSpaceEntry(c, xRefTable, dict, ownerObjNr, dictName, "ColorSpace", OPTIONAL, colorSpaceNoPattern)
	if err != nil {
		return info, fmt.Errorf("%s.ColorSpace: %w", dictName, err)
	}

	info.components, err = colorSpaceComponents(xRefTable, dict["ColorSpace"], ownerObjNr)
	if err != nil {
		return info, fmt.Errorf("%s.ColorSpace: %w", dictName, err)
	}
	err = validateComponentArrayEntry(c, xRefTable, dict, ownerObjNr, dictName, "Background", info.components)
	if err != nil {
		return info, fmt.Errorf("%s.Background: %w", dictName, err)
	}

	_, err = validateRectangleEntry(xRefTable, dict, ownerObjNr, dictName, "BBox", OPTIONAL, model.V10, nil)
	if err != nil {
		return info, fmt.Errorf("%s.BBox: %w", dictName, err)
	}

	_, err = validateBooleanEntry(xRefTable, dict, ownerObjNr, dictName, "AntiAlias", OPTIONAL, model.V10, nil)
	if err != nil {
		return info, fmt.Errorf("%s.AntiAlias: %w", dictName, err)
	}

	info.kind = shadingType.Value()
	return info, nil
}

func validateFunctionBasedShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "functionBasedShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, ownerObjNr, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, ownerObjNr, dictName, "Matrix", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	if _, err = validateShadingFunctionEntry(c, xRefTable, dict, ownerObjNr, dictName, REQUIRED, components); err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	return nil
}

func validateAxialShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "axialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, ownerObjNr, dictName, "Coords", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return fmt.Errorf("%s.Coords: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, ownerObjNr, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateShadingFunctionEntry(c, xRefTable, dict, ownerObjNr, dictName, REQUIRED, components)
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

func validateRadialShadingDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "radialShadingDict"

	_, err := validateNumberArrayEntry(xRefTable, dict, ownerObjNr, dictName, "Coords", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Coords: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, ownerObjNr, dictName, "Domain", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateShadingFunctionEntry(c, xRefTable, dict, ownerObjNr, dictName, REQUIRED, components)
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

	info, err := validateShadingDictCommonEntries(c, xRefTable, dict, ownerObjNr)
	if err != nil {
		return fmt.Errorf("shading dict: %w", err)
	}

	switch info.kind {
	case 1:
		err = validateFunctionBasedShadingDict(c, xRefTable, dict, ownerObjNr, info.components)

	case 2:
		err = validateAxialShadingDict(c, xRefTable, dict, ownerObjNr, info.components)

	case 3:
		err = validateRadialShadingDict(c, xRefTable, dict, ownerObjNr, info.components)

	default:
		return fmt.Errorf("unexpected shadingType: %d", info.kind)
	}

	if err != nil {
		return fmt.Errorf("shading dict type %d: %w", info.kind, err)
	}
	return nil
}

func validateFreeFormGouroudShadedTriangleMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "freeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	return validateMeshShadingArrays(c, xRefTable, dict, ownerObjNr, dictName, components)
}

func validateLatticeFormGouraudShadedTriangleMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "latticeFormGouraudShadedTriangleMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "VerticesPerRow", REQUIRED, model.V10, func(i int) bool { return i >= 2 })
	if err != nil {
		return fmt.Errorf("%s.VerticesPerRow: %w", dictName, err)
	}

	return validateMeshShadingArrays(c, xRefTable, dict, ownerObjNr, dictName, components)
}

func validateCoonsPatchMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "coonsPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	return validateMeshShadingArrays(c, xRefTable, dict, ownerObjNr, dictName, components)
}

func validateTensorProductPatchMeshesDict(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr, components int) error {
	dictName := "tensorProductPatchMeshesDict"

	_, err := validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerCoordinate", REQUIRED, model.V10, validateBitsPerCoordinate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerCoordinate: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerComponent", REQUIRED, model.V10, validateBitsPerComponent)
	if err != nil {
		return fmt.Errorf("%s.BitsPerComponent: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, dict, ownerObjNr, dictName, "BitsPerFlag", REQUIRED, model.V10, validateBitsPerFlag)
	if err != nil {
		return fmt.Errorf("%s.BitsPerFlag: %w", dictName, err)
	}

	return validateMeshShadingArrays(c, xRefTable, dict, ownerObjNr, dictName, components)
}

func validateShadingStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, ownerObjNr int) error {
	// Shading 2, 4-7

	dict := sd.Dict

	info, err := validateShadingDictCommonEntries(c, xRefTable, dict, ownerObjNr)
	if err != nil {
		return fmt.Errorf("shading stream dict: %w", err)
	}

	switch info.kind {

	case 2:
		err = validateAxialShadingDict(c, xRefTable, dict, ownerObjNr, info.components)

	case 4:
		err = validateFreeFormGouroudShadedTriangleMeshesDict(c, xRefTable, dict, ownerObjNr, info.components)

	case 5:
		err = validateLatticeFormGouraudShadedTriangleMeshesDict(c, xRefTable, dict, ownerObjNr, info.components)

	case 6:
		err = validateCoonsPatchMeshesDict(c, xRefTable, dict, ownerObjNr, info.components)

	case 7:
		err = validateTensorProductPatchMeshesDict(c, xRefTable, dict, ownerObjNr, info.components)

	default:
		return fmt.Errorf("unexpected shadingType: %d", info.kind)
	}

	if err != nil {
		return fmt.Errorf("shading stream dict type %d: %w", info.kind, err)
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

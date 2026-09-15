/*
Copyright 2026 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateNumericArrayValues(c context.Context, x *model.XRefTable, a types.Array, objNr int, dictName, entryName string) error {
	for i, o := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if _, err := x.DereferenceNumber(o); err != nil {
			err = fmt.Errorf("%s.%s[%d]: expected number: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, o))
		}
	}
	return nil
}

func validateComponentArrayEntry(c context.Context, x *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, components int) error {
	a, err := validateArrayEntry(x, d, ownerObjNr, dictName, entryName, OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	if components > 0 {
		if err = validateArrayExactLength(a, objNr, dictName, entryName, components); err != nil {
			return err
		}
	}
	return validateNumericArrayValues(c, x, a, objNr, dictName, entryName)
}

func validateShadingFunctionEntry(c context.Context, x *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, required bool, components int) (present bool, err error) {
	objNr := validationEntryObjectNumber(ownerObjNr, d, "Function")
	defer func() { err = model.WithValidationErrorObject(err, objNr) }()
	if err = contextutil.Check(c); err != nil {
		return false, err
	}
	o, err := validateEntry(x, d, ownerObjNr, dictName, "Function", required, model.V10)
	if err != nil || o == nil {
		return false, err
	}
	a, array := o.(types.Array)
	if array {
		if len(a) == 0 {
			return true, arrayCardinalityError(dictName, "Function", objNr, 0, "one or more functions")
		}
		if components > 0 {
			if err = validateArrayExactLength(a, objNr, dictName, "Function", components); err != nil {
				return true, err
			}
		}
	}
	if err = validateFunctionObjects(newFunctionTraversal(c, x), d["Function"], o, objNr); err != nil {
		return true, err
	}
	if array {
		t := singleOutputFunctions{c: c, x: x, checked: map[int]bool{}}
		for i, f := range a {
			if err = t.validate(f, objNr, 0); err != nil {
				return true, fmt.Errorf("%s.Function[%d]: %w", dictName, i, err)
			}
		}
	}
	return true, nil
}

// singleOutputFunctions checks already validated function graphs without re-expanding shared subfunctions.
type singleOutputFunctions struct {
	c       context.Context
	x       *model.XRefTable
	checked map[int]bool
}

func (t *singleOutputFunctions) validate(o types.Object, ownerObjNr, depth int) (err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() { err = model.WithValidationErrorObject(err, objNr) }()
	if err = contextutil.Check(t.c); err != nil {
		return err
	}
	if err = t.x.CheckRecursionDepth("shading function outputs", depth); err != nil {
		return err
	}
	identity := functionObjectIdentity(o)
	if identity > 0 && t.checked[identity] {
		return nil
	}
	o, err = t.x.Dereference(o)
	if err != nil {
		return err
	}
	var d types.Dict
	switch o := o.(type) {
	case types.Dict:
		d = o
	case types.StreamDict:
		d = o.Dict
	default:
		return fmt.Errorf("expected function dictionary or stream, got %T", o)
	}
	if err = t.validateDict(d, objNr, depth); err != nil {
		return err
	}
	if identity > 0 {
		t.checked[identity] = true
	}
	return nil
}

func (t *singleOutputFunctions) validateDict(d types.Dict, objNr, depth int) error {
	ft, err := t.x.DereferenceInteger(d["FunctionType"])
	if err != nil {
		return err
	}
	if ft == nil {
		return fmt.Errorf("missing FunctionType")
	}
	if *ft == 2 {
		a, err := t.x.DereferenceArray(d["C0"])
		if err != nil {
			return err
		}
		if a == nil {
			return nil
		}
		return validateArrayExactLength(a, validationEntryObjectNumber(objNr, d, "C0"), "shadingFunction", "C0", 1)
	}
	a, err := t.x.DereferenceArray(d["Range"])
	if err != nil {
		return err
	}
	if *ft != 3 || len(a) > 0 {
		if err = validateArrayExactLength(a, validationEntryObjectNumber(objNr, d, "Range"), "shadingFunction", "Range", 2); err != nil {
			return err
		}
	}
	if *ft == 3 {
		return t.validateChildren(d, objNr, depth)
	}
	return nil
}

func (t *singleOutputFunctions) validateChildren(d types.Dict, objNr, depth int) error {
	a, err := t.x.DereferenceArray(d["Functions"])
	if err != nil {
		return err
	}
	owner := validationEntryObjectNumber(objNr, d, "Functions")
	for i, f := range a {
		if err = t.validate(f, owner, depth+1); err != nil {
			return fmt.Errorf("Functions[%d]: %w", i, err)
		}
	}
	return nil
}

func validateMeshShadingArrays(c context.Context, x *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, components int) error {
	a, err := validateArrayEntry(x, d, ownerObjNr, dictName, "Decode", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}
	present, err := validateShadingFunctionEntry(c, x, d, ownerObjNr, dictName, OPTIONAL, components)
	if err != nil {
		return fmt.Errorf("%s.Function: %w", dictName, err)
	}
	if present {
		components = 1
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, "Decode")
	// Without a known colour space, still require the coordinate pairs and at least one colour pair.
	if components == 0 {
		err = validateArrayPairs(a, objNr, dictName, "Decode", 3)
	} else if len(a) < 4 || (len(a)-4)%2 != 0 || (len(a)-4)/2 != components {
		expected := fmt.Sprintf("4 coordinate values and two values per component (%d components)", components)
		err = arrayCardinalityError(dictName, "Decode", objNr, len(a), expected)
	}
	if err != nil {
		return err
	}
	return validateNumericArrayValues(c, x, a, objNr, dictName, "Decode")
}

func validateSoftMaskBackdrop(c context.Context, x *model.XRefTable, d types.Dict, sd *types.StreamDict, ownerObjNr, groupObjNr int, luminosity bool) error {
	a, err := validateArrayEntry(x, d, ownerObjNr, "softMaskDict", "BC", OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, "BC")
	// BC is consulted only for luminosity; alpha groups can inherit their colour space.
	if luminosity {
		// The stream-entry helper omits groups already validated through another soft mask.
		if sd == nil {
			sd, err = validateStreamDictForObject(x, d["G"], groupObjNr)
			if err != nil {
				return fmt.Errorf("softMaskDict.G: %w", err)
			}
		}
		components, err := softMaskBackdropComponents(x, sd, groupObjNr)
		if err != nil {
			return fmt.Errorf("softMaskDict.BC: %w", err)
		}
		if err = validateArrayExactLength(a, objNr, "softMaskDict", "BC", components); err != nil {
			return err
		}
	}
	return validateNumericArrayValues(c, x, a, objNr, "softMaskDict", "BC")
}

func softMaskBackdropComponents(x *model.XRefTable, sd *types.StreamDict, groupObjNr int) (components int, err error) {
	defer func() { err = model.WithValidationErrorObject(err, groupObjNr) }()
	if sd == nil {
		return 0, fmt.Errorf("missing G transparency group")
	}
	group, err := x.DereferenceDict(sd.Dict["Group"])
	groupObjNr = validationEntryObjectNumber(groupObjNr, sd.Dict, "Group")
	if err != nil {
		return 0, err
	}
	if group == nil {
		return 0, fmt.Errorf("G.Group: missing transparency group attributes")
	}
	components, err = colorSpaceComponents(x, group["CS"], groupObjNr)
	if err != nil {
		return 0, err
	}
	if components == 0 {
		return 0, fmt.Errorf("G.Group.CS: colour space required for luminosity backdrop")
	}
	return components, nil
}

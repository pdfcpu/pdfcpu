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
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// see 8.4.5 Graphics State Parameter Dictionaries

func validateBlendMode(s string) bool {
	// see 11.3.5; table 136

	return types.MemberOf(s, []string{"None", "Normal", "Compatible", "Multiply", "Mult", "Screen", "Overlay", "Darken", "Lighten",
		"ColorDodge", "ColorBurn", "HardLight", "SoftLight", "Difference", "Exclusion",
		"Hue", "Saturation", "Color", "Luminosity"})
}

func validateLineDashPatternEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, func(a types.Array) bool { return len(a) == 2 })
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	// We are dealing with integers which may be represented by Integer or Float objects.

	_, err = validateNumberArray(xRefTable, a[0], objNr)
	if err != nil {
		err = fmt.Errorf("%s.%s[0]: %w", dictName, entryName, err)
		return model.WithValidationErrorObject(err, validationObjectNumber(objNr, a[0]))
	}

	_, err = validateNumberForObject(xRefTable, a[1], objNr)
	if err != nil {
		err = fmt.Errorf("%s.%s[1]: %w", dictName, entryName, err)
		return model.WithValidationErrorObject(err, validationObjectNumber(objNr, a[1]))
	}

	return nil
}

func validateFunctionOrNameEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validName func(string) bool) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	switch o := o.(type) {

	case types.Name:
		if !validName(o.Value()) {
			return fmt.Errorf("%s.%s: invalid name %q", dictName, entryName, o.Value())
		}

	case types.Dict:
		if err = validateFunction(c, xRefTable, functionEntryObject(d, entryName, o), 0); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	case types.StreamDict:
		if err = validateFunction(c, xRefTable, functionEntryObject(d, entryName, o), 0); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		return fmt.Errorf("%s.%s: expected function or name, got %T", dictName, entryName, o)

	}

	return nil
}

func functionEntryObject(d types.Dict, entryName string, resolved types.Object) types.Object {
	o := d[entryName]
	if functionObjectIdentity(o) > 0 {
		return o
	}
	return resolved
}

func validateBGEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return validateFunctionOrNameEntry(c, xRefTable, d, dictName, entryName, required, sinceVersion, func(string) bool { return false })
	}
	return validateFunctionOrNameEntry(c, xRefTable, d, dictName, entryName, required, sinceVersion, func(s string) bool { return s == "Identity" })
}

func validateBG2Entry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	return validateFunctionOrNameEntry(c, xRefTable, d, dictName, entryName, required, sinceVersion, func(s string) bool { return s == "Default" })
}

func validateUCREntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return validateFunctionOrNameEntry(c, xRefTable, d, dictName, entryName, required, sinceVersion, func(string) bool { return false })
	}
	return validateFunctionOrNameEntry(c, xRefTable, d, dictName, entryName, required, sinceVersion, func(s string) bool { return s == "Identity" })
}

func validateUCR2Entry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	return validateFunctionOrNameEntry(c, xRefTable, d, dictName, entryName, required, sinceVersion, func(s string) bool { return s == "Default" })
}

func validateTransferFunction(c context.Context, xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()
	if err := contextutil.Check(c); err != nil {
		return err
	}
	rawObject := o
	if o, err = xRefTable.Dereference(o); err != nil {
		return fmt.Errorf("transfer function: dereference: %w", err)
	}
	t := newFunctionTraversal(c, xRefTable)

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		if s != "Identity" {
			return fmt.Errorf("transfer function: invalid name %q", s)
		}

	case types.Array:

		if len(o) != 4 {
			return fmt.Errorf("transfer function array: invalid length %d, expected 4", len(o))
		}

		for i, o := range o {
			objNr := validationObjectNumber(ownerObjNr, o)
			resolved, err := xRefTable.Dereference(o)
			if err != nil {
				err = fmt.Errorf("transfer function array[%d]: dereference: %w", i, err)
				return model.WithValidationErrorObject(err, objNr)
			}
			if resolved == nil {
				continue
			}

			err = t.validateFunction(o, objNr, 0)
			if err != nil {
				err = fmt.Errorf("transfer function array[%d]: %w", i, err)
				return model.WithValidationErrorObject(err, objNr)
			}

		}

	case types.Dict:
		err = t.validateFunction(rawObject, ownerObjNr, 0)

	case types.StreamDict:
		err = t.validateFunction(rawObject, ownerObjNr, 0)

	default:
		return fmt.Errorf("transfer function: expected function, name or function array, got %T", o)

	}

	return err
}

func validateTransferFunctionEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	if err := validateTransferFunction(c, xRefTable, d[entryName], objNr); err != nil {
		return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
	}
	return nil
}

func validateTR(c context.Context, xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()
	if err := contextutil.Check(c); err != nil {
		return err
	}
	rawObject := o
	if o, err = xRefTable.Dereference(o); err != nil {
		return fmt.Errorf("TR: dereference: %w", err)
	}
	t := newFunctionTraversal(c, xRefTable)

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		if s != "Identity" {
			return fmt.Errorf("TR: invalid name %q", s)
		}

	case types.Array:

		if len(o) != 4 {
			return fmt.Errorf("TR array: invalid length %d, expected 4", len(o))
		}

		for i, o := range o {
			objNr := validationObjectNumber(ownerObjNr, o)
			resolved, err := xRefTable.Dereference(o)
			if err != nil {
				err = fmt.Errorf("TR array[%d]: dereference: %w", i, err)
				return model.WithValidationErrorObject(err, objNr)
			}

			if resolved == nil {
				continue
			}

			if name, ok := resolved.(types.Name); ok {
				s := name.Value()
				if s != "Identity" {
					err = fmt.Errorf("TR array[%d]: invalid name %q", i, s)
					return model.WithValidationErrorObject(err, objNr)
				}
				continue
			}

			err = t.validateFunction(o, objNr, 0)
			if err != nil {
				err = fmt.Errorf("TR array[%d]: %w", i, err)
				return model.WithValidationErrorObject(err, objNr)
			}

		}

	case types.Dict:
		err = t.validateFunction(rawObject, ownerObjNr, 0)

	case types.StreamDict:
		err = t.validateFunction(rawObject, ownerObjNr, 0)

	default:
		return fmt.Errorf("TR: expected function, name or function array, got %T", o)

	}

	return err
}

func validateTREntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	if err := validateTR(c, xRefTable, d[entryName], objNr); err != nil {
		return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
	}
	return nil
}

func validateTR2Name(name types.Name) error {
	s := name.Value()
	if s != "Identity" && s != "Default" {
		return fmt.Errorf("TR2: invalid name %q", s)
	}
	return nil
}

func validateTR2(c context.Context, xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()
	if err := contextutil.Check(c); err != nil {
		return err
	}
	rawObject := o
	if o, err = xRefTable.Dereference(o); err != nil {
		return fmt.Errorf("TR2: dereference: %w", err)
	}
	t := newFunctionTraversal(c, xRefTable)

	switch o := o.(type) {

	case types.Name:
		if err = validateTR2Name(o); err != nil {
			return err
		}

	case types.Array:

		if len(o) != 4 {
			return fmt.Errorf("TR2 array: invalid length %d, expected 4", len(o))
		}

		for i, o := range o {
			objNr := validationObjectNumber(ownerObjNr, o)
			resolved, err := xRefTable.Dereference(o)
			if err != nil {
				err = fmt.Errorf("TR2 array[%d]: dereference: %w", i, err)
				return model.WithValidationErrorObject(err, objNr)
			}

			if resolved == nil {
				continue
			}

			if name, ok := resolved.(types.Name); ok {
				if err = validateTR2Name(name); err != nil {
					err = fmt.Errorf("TR2 array[%d]: %w", i, err)
					return model.WithValidationErrorObject(err, objNr)
				}
				continue
			}

			err = t.validateFunction(o, objNr, 0)
			if err != nil {
				err = fmt.Errorf("TR2 array[%d]: %w", i, err)
				return model.WithValidationErrorObject(err, objNr)
			}

		}

	case types.Dict:
		err = t.validateFunction(rawObject, ownerObjNr, 0)

	case types.StreamDict:
		err = t.validateFunction(rawObject, ownerObjNr, 0)

	default:
		return fmt.Errorf("TR2: expected function, name or function array, got %T", o)

	}

	return err
}

func validateTR2Entry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	if err := validateTR2(c, xRefTable, d[entryName], objNr); err != nil {
		return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
	}
	return nil
}

func validateSpotFunctionEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		validateSpotFunctionName := func(s string) bool {
			return types.MemberOf(s, []string{
				"SimpleDot", "InvertedSimpleDot", "DoubleDot", "InvertedDoubleDot", "CosineDot",
				"Double", "InvertedDouble", "Line", "LineX", "LineY", "Round", "Ellipse", "EllipseA",
				"InvertedEllipseA", "EllipseB", "EllipseC", "InvertedEllipseC", "Square", "Cross", "Rhomboid"})
		}
		s := o.Value()
		if !validateSpotFunctionName(s) {
			return fmt.Errorf("%s.%s: invalid spot function name %q", dictName, entryName, s)
		}

	case types.Dict:
		if err = validateFunction(c, xRefTable, functionEntryObject(d, entryName, o), 0); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	case types.StreamDict:
		if err = validateFunction(c, xRefTable, functionEntryObject(d, entryName, o), 0); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		return fmt.Errorf("%s.%s: expected spot function name or function, got %T", dictName, entryName, o)

	}

	return err
}

func validateType1HalftoneDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "type1HalftoneDict"

	// HalftoneName, optional, string
	_, err := validateStringEntry(xRefTable, d, 0, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Frequency, required, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Frequency", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Angle, required, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Angle", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SpotFunction, required, function or name
	err = validateSpotFunctionEntry(c, xRefTable, d, dictName, "SpotFunction", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// TransferFunction, optional, function
	err = validateTransferFunctionEntry(c, xRefTable, d, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "AccurateScreens", OPTIONAL, sinceVersion, nil)

	return err
}

type halftoneTraversal struct {
	c         context.Context
	xRefTable *model.XRefTable
	ancestors map[int]bool
	validated map[int]int
}

func newHalftoneTraversal(c context.Context, xRefTable *model.XRefTable) *halftoneTraversal {
	return &halftoneTraversal{
		c:         c,
		xRefTable: xRefTable,
		ancestors: map[int]bool{},
		validated: map[int]int{},
	}
}

func halftoneObjectIdentity(o types.Object) int {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0
	}
	return ir.ObjectNumber.Value()
}

func (t *halftoneTraversal) enter(objNr int) error {
	if objNr <= 0 {
		return nil
	}
	if t.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, model.ErrHalftoneCycle)
	}
	t.ancestors[objNr] = true
	return nil
}

func (t *halftoneTraversal) leave(objNr int) {
	if objNr > 0 {
		delete(t.ancestors, objNr)
	}
}

func (t *halftoneTraversal) alreadyValidated(objNr, depth int) bool {
	if objNr <= 0 {
		return false
	}
	validatedDepth, ok := t.validated[objNr]
	return ok && depth <= validatedDepth
}

func (t *halftoneTraversal) markValidated(objNr, depth int) {
	if objNr <= 0 {
		return
	}
	if previous, ok := t.validated[objNr]; !ok || depth > previous {
		t.validated[objNr] = depth
	}
}

func (t *halftoneTraversal) validateType5HalftoneDict(d types.Dict, sinceVersion model.Version, depth int) error {
	dictName := "type5HalftoneDict"

	_, err := validateStringEntry(t.xRefTable, d, 0, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	for _, component := range []string{"Gray", "Red", "Green", "Blue", "Cyan", "Magenta", "Yellow", "Black"} {
		err = t.validateEntry(d, dictName, component, OPTIONAL, sinceVersion, depth+1)
		if err != nil {
			return err
		}
	}

	return t.validateEntry(d, dictName, "Default", REQUIRED, sinceVersion, depth+1)
}

func validateType6HalftoneStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, sinceVersion model.Version) error {
	dictName := "type6HalftoneDict"

	_, err := validateStringEntry(xRefTable, sd.Dict, 0, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(c, xRefTable, sd.Dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateType10HalftoneStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, sinceVersion model.Version) error {
	dictName := "type10HalftoneDict"

	_, err := validateStringEntry(xRefTable, sd.Dict, 0, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Xsquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Ysquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(c, xRefTable, sd.Dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateType16HalftoneStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, sinceVersion model.Version) error {
	dictName := "type16HalftoneDict"

	_, err := validateStringEntry(xRefTable, sd.Dict, 0, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Width2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Height2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(c, xRefTable, sd.Dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func (t *halftoneTraversal) validateDict(d types.Dict, sinceVersion model.Version, depth int) error {
	dictName := "halfToneDict"
	xRefTable := t.xRefTable

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Halftone" })
	if err != nil {
		return err
	}

	// HalftoneType, required, integer
	halftoneType, err := validateIntegerEntry(xRefTable, d, 0, dictName, "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	switch *halftoneType {

	case 1:
		err = validateType1HalftoneDict(t.c, xRefTable, d, sinceVersion)

	case 5:
		err = t.validateType5HalftoneDict(d, sinceVersion, depth)

	default:
		err = fmt.Errorf("unknown halftoneTyp: %d", *halftoneType)

	}

	return err
}

func validateHalfToneStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, sinceVersion model.Version) error {
	dictName := "writeHalfToneStreamDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, sd.Dict, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Halftone" })
	if err != nil {
		return err
	}

	// HalftoneType, required, integer
	halftoneType, err := validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil || halftoneType == nil {
		return err
	}

	switch *halftoneType {

	case 6:
		err = validateType6HalftoneStreamDict(c, xRefTable, sd, sinceVersion)

	case 10:
		err = validateType10HalftoneStreamDict(c, xRefTable, sd, sinceVersion)

	case 16:
		err = validateType16HalftoneStreamDict(c, xRefTable, sd, sinceVersion)

	default:
		err = fmt.Errorf("unknown halftoneTyp: %d", *halftoneType)

	}

	return err
}

func (t *halftoneTraversal) validateObject(o types.Object, dictName, entryName string, sinceVersion model.Version, depth int) error {
	switch o := o.(type) {
	case types.Name:
		if o.Value() != "Default" {
			return fmt.Errorf("%s.%s: invalid halftone name %q", dictName, entryName, o.Value())
		}

	case types.Dict:
		if err := t.validateDict(o, sinceVersion, depth); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	case types.StreamDict:
		if err := validateHalfToneStreamDict(t.c, t.xRefTable, &o, sinceVersion); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		return fmt.Errorf("%s.%s: expected halftone dict, stream dict or Default name, got %T", dictName, entryName, o)
	}
	return nil
}

func (t *halftoneTraversal) validateEntry(d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, depth int) (err error) {
	// See 10.5
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()
	if err := contextutil.Check(t.c); err != nil {
		return err
	}

	rawObject, found := d.Find(entryName)
	if !found || rawObject == nil {
		_, err = validateEntry(t.xRefTable, d, 0, dictName, entryName, required, sinceVersion)
		return err
	}
	if err := t.xRefTable.CheckRecursionDepth("halftone graph", depth); err != nil {
		return err
	}

	halftoneObjNr := halftoneObjectIdentity(rawObject)
	if err := t.enter(halftoneObjNr); err != nil {
		return err
	}
	defer t.leave(halftoneObjNr)
	if t.alreadyValidated(halftoneObjNr, depth) {
		return nil
	}

	o, err := validateEntry(t.xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}
	if err = t.validateObject(o, dictName, entryName, sinceVersion, depth); err != nil {
		return err
	}
	t.markValidated(halftoneObjNr, depth)
	return nil
}

func validateHalfToneEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	return newHalftoneTraversal(c, xRefTable).validateEntry(d, dictName, entryName, required, sinceVersion, 0)
}

func validateBlendModeEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		_, err = xRefTable.DereferenceName(o, sinceVersion, validateBlendMode)
		if err != nil {
			return err
		}

	case types.Array:
		for i, o := range o {
			itemObjNr := validationObjectNumber(objNr, o)
			_, err = xRefTable.DereferenceName(o, sinceVersion, validateBlendMode)
			if err != nil {
				err = fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
				return model.WithValidationErrorObject(err, itemObjNr)
			}
		}

	default:
		return fmt.Errorf("%s.%s: expected name or array, got %T", dictName, entryName, o)

	}

	return nil
}

func validateSoftMaskTransferFunctionEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		if s != "Identity" {
			return fmt.Errorf("%s.%s: invalid transfer function name %q", dictName, entryName, s)
		}

	case types.Dict:
		if err = validateFunction(c, xRefTable, functionEntryObject(d, entryName, o), 0); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	case types.StreamDict:
		if err = validateFunction(c, xRefTable, functionEntryObject(d, entryName, o), 0); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		return fmt.Errorf("%s.%s: expected function or Identity name, got %T", dictName, entryName, o)

	}

	return err
}

func validateSoftMaskDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	// see 11.6.5.2

	dictName := "softMaskDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Mask" })
	if err != nil {
		return err
	}

	// S, name, required
	subtype, err := validateNameEntry(xRefTable, d, 0, dictName, "S", REQUIRED, model.V10, func(s string) bool { return s == "Alpha" || s == "Luminosity" })
	if err != nil {
		return err
	}

	// G, stream, required
	// A transparency group XObject (see “Transparency Group XObjects”)
	// to be used as the source of alpha or colour values for deriving the mask.
	rawGroup := d["G"]
	groupObjNr := validationObjectNumber(ownerObjNr, rawGroup)
	sd, err := validateStreamDictEntry(xRefTable, d, ownerObjNr, dictName, "G", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	if sd != nil {
		err = validateXObjectStreamDictContents(c, xRefTable, sd, groupObjNr)
		if err != nil {
			return model.WithValidationErrorObject(err, groupObjNr)
		}
	}

	// TR (Optional) function or name
	// A function object (see “Functions”) specifying the transfer function
	// to be used in deriving the mask values.
	err = validateSoftMaskTransferFunctionEntry(c, xRefTable, d, dictName, "TR", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// BC, number array, optional
	// Array of component values specifying the colour to be used
	// as the backdrop against which to composite the transparency group XObject G.
	err = validateSoftMaskBackdrop(c, xRefTable, d, sd, ownerObjNr, groupObjNr, subtype != nil && *subtype == "Luminosity")

	return err
}

func validateSoftMaskEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	// see 11.3.7.2 Source Shape and Opacity
	// see 11.6.4.3 Mask Shape and Opacity
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		s := o.Value()
		if !validateBlendMode(s) {
			return fmt.Errorf("%s.%s: invalid soft mask name %q", dictName, entryName, s)
		}

	case types.Dict:
		if err = validateSoftMaskDict(c, xRefTable, o, objNr); err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		err = fmt.Errorf("%s.%s: expected name or soft mask dict, got %T", dictName, entryName, o)

	}

	return err
}

func validateExtGStateFont(x *model.XRefTable, d types.Dict, dictName string) error {
	a, err := validateArrayEntry(x, d, 0, dictName, "Font", OPTIONAL, model.V13, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(0, d, "Font")
	if err := validateArrayExactLength(a, objNr, dictName, "Font", 2); err != nil {
		return err
	}
	if err := validateExtGStateFontReference(x, a[0], dictName); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}
	if _, err := x.DereferenceNumber(a[1]); err != nil {
		err = fmt.Errorf("%s.Font[1]: expected font size number: %w", dictName, err)
		return model.WithValidationErrorObject(err, validationObjectNumber(objNr, a[1]))
	}
	return nil
}

func validateExtGStateFontReference(x *model.XRefTable, o types.Object, dictName string) error {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return fmt.Errorf("%s.Font[0]: expected indirect font reference", dictName)
	}
	d, err := x.DereferenceDict(ir)
	if err == nil && d == nil {
		err = errors.New("missing font dictionary")
	}
	if err == nil {
		_, err = validateNameEntry(x, d, ir.ObjectNumber.Value(), "fontDict", "Type", REQUIRED, model.V10,
			func(s string) bool { return s == "Font" })
	}
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("%s.Font[0]: %w", dictName, err), ir.ObjectNumber.Value())
	}
	return nil
}

func validateExtGStateDictPart1(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// LW, number, optional, since V1.3
	_, err := validateNumberEntry(xRefTable, d, 0, dictName, "LW", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// LC, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "LC", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// LJ, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "LJ", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// ML, number, optional, since V1.3
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "ML", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = validateLineDashPatternEntry(xRefTable, d, dictName, "D", OPTIONAL, model.V13)
	if err != nil {
		return err
	}

	// RI, name, optional, since V1.3
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "RI", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// OP, boolean, optional,
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "OP", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// op, boolean, optional, since V1.3
	sinceVersion := model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "op", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// OPM, integer, optional, since V1.3
	sinceVersion = model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	opm, err := validateIntegerEntry(xRefTable, d, 0, dictName, "OPM", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if opm != nil && xRefTable.ValidationMode == model.ValidationRelaxed && xRefTable.Version() < model.V13 {
		showDigestedVersionViolation(xRefTable, "dict="+dictName+" entry=OPM")
	}

	// Font, array, optional, since V1.3
	return validateExtGStateFont(xRefTable, d, dictName)
}

func validateExtGStateDictPart2(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// BG, function, optional, black-generation function, see 10.3.4
	err := validateBGEntry(c, xRefTable, d, dictName, "BG", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// BG2, function or name(/Default), optional, since V1.3
	err = validateBG2Entry(c, xRefTable, d, dictName, "BG2", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// UCR, function, optional, undercolor-removal function, see 10.3.4
	err = validateUCREntry(c, xRefTable, d, dictName, "UCR", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// UCR2, function or name(/Default), optional, since V1.3
	err = validateUCR2Entry(c, xRefTable, d, dictName, "UCR2", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// TR, function, array of 4 functions or name(/Identity), optional, see 10.4 transfer functions
	err = validateTREntry(c, xRefTable, d, dictName, "TR", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// TR2, function, array of 4 functions or name(/Identity,/Default), optional, since V1.3
	err = validateTR2Entry(c, xRefTable, d, dictName, "TR2", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// HT, dict, stream or name, optional
	// half tone dictionary or stream or /Default, see 10.5
	sinceVersion := model.V12
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	err = validateHalfToneEntry(c, xRefTable, d, dictName, "HT", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// FL, number, optional, since V1.3, flatness tolerance, see 10.6.2
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "FL", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// SM, number, optional, since V1.3, smoothness tolerance
	sinceVersion = model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "SM", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SA, boolean, optional, see 10.6.5 Automatic Stroke Adjustment
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "SA", OPTIONAL, model.V10, nil)

	return err
}

func validateExtGStateDictPart3(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// BM, name or array, optional, since V1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	err := validateBlendModeEntry(xRefTable, d, dictName, "BM", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// SMask, dict or name, optional, since V1.4
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	err = validateSoftMaskEntry(c, xRefTable, d, dictName, "SMask", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// CA, number, optional, since V1.4, current stroking alpha constant, see 11.3.7.2 and 11.6.4.4
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	ca, err := validateNumberEntry(xRefTable, d, 0, dictName, "CA", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if ca != nil && xRefTable.ValidationMode == model.ValidationRelaxed && xRefTable.Version() < model.V14 {
		showDigestedVersionViolation(xRefTable, "dict="+dictName+" entry=CA")
	}

	// ca, number, optional, since V1.4, same as CA but for nonstroking operations.
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "ca", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// AIS, alpha source flag "alpha is shape", boolean, optional, since V1.4
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "AIS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// TK, boolean, optional, since V1.4, text knockout flag.
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "TK", OPTIONAL, sinceVersion, nil)

	return err
}

func validateExtGStateDict(c context.Context, xRefTable *model.XRefTable, o types.Object) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("ExtGState: dereference dict: %w", err)
	}
	if d == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return nil
		}
		return fmt.Errorf("ExtGState: missing dict")
	}

	dictName := "extGStateDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "ExtGState" })
	if err != nil {
		return fmt.Errorf("ExtGState.Type: %w", err)
	}

	err = validateExtGStateDictPart1(xRefTable, d, dictName)
	if err != nil {
		return fmt.Errorf("ExtGState graphics state parameters: %w", err)
	}

	err = validateExtGStateDictPart2(c, xRefTable, d, dictName)
	if err != nil {
		return fmt.Errorf("ExtGState transfer and halftone parameters: %w", err)
	}

	err = validateExtGStateDictPart3(c, xRefTable, d, dictName)
	if err != nil {
		return fmt.Errorf("ExtGState transparency parameters: %w", err)
	}

	// Check for AAPL extensions.
	o, _, err = d.Entry(dictName, "AAPL:AA", OPTIONAL)
	if err != nil {
		return fmt.Errorf("ExtGState.AAPL:AA: %w", err)
	}
	if o != nil {
		xRefTable.CustomExtensions = true
	}

	return nil
}

func validateExtGStateResourceDict(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("ExtGState resource dict: dereference dict: %w", err)
	}
	if d == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return nil
		}
		return fmt.Errorf("ExtGState resource dict: missing dict")
	}

	// Version check
	err = xRefTable.ValidateVersion("ExtGStateResourceDict", sinceVersion)
	if err != nil {
		return fmt.Errorf("ExtGState resource dict: version: %w", err)
	}

	// Iterate over extGState resource dictionary
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		o := d[name]
		// Process extGStateDict
		err = validateExtGStateDict(c, xRefTable, o)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("ExtGState resource %s", name), o), err)
		}

	}

	return nil
}

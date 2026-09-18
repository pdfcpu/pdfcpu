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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// see 7.10 Functions

func validateFunctionDimension(a types.Array, d types.Dict, ownerObjNr int, dictName, entryName string) (int, error) {
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	if err := validateArrayPairs(a, objNr, dictName, entryName, 1); err != nil {
		return 0, err
	}
	return len(a) / 2, nil
}

func validateFunctionArrayLength(a types.Array, d types.Dict, ownerObjNr int, dictName, entryName string, length int, relationship string) error {
	if len(a) == length {
		return nil
	}
	expected := fmt.Sprintf("%d, %s", length, relationship)
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	return arrayCardinalityError(dictName, entryName, objNr, len(a), expected)
}

type exponentialFunctionArrays struct {
	domain     types.Array
	rangeArray types.Array
	c0         types.Array
	c1         types.Array
}

func validateExponentialFunctionArrays(xRefTable *model.XRefTable, d types.Dict, dictName string) (*exponentialFunctionArrays, error) {
	a := &exponentialFunctionArrays{}
	var err error
	if a.domain, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Domain", REQUIRED, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Domain: %w", dictName, err)
	}
	if a.rangeArray, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Range", OPTIONAL, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Range: %w", dictName, err)
	}
	if a.c0, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "C0", OPTIONAL, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.C0: %w", dictName, err)
	}
	if a.c1, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "C1", OPTIONAL, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.C1: %w", dictName, err)
	}
	return a, nil
}

func validateExponentialOutputDimension(a *exponentialFunctionArrays, d types.Dict, dictName string) (int, error) {
	c0Length := 1
	if a.c0 != nil {
		c0Length = len(a.c0)
		if c0Length == 0 {
			return 0, arrayCardinalityError(dictName, "C0", validationEntryObjectNumber(0, d, "C0"), 0,
				"one or more output values")
		}
	}
	c1Length := 1
	if a.c1 != nil {
		c1Length = len(a.c1)
		if c1Length == 0 {
			return 0, arrayCardinalityError(dictName, "C1", validationEntryObjectNumber(0, d, "C1"), 0,
				"one or more output values")
		}
	}
	if c1Length != c0Length {
		expected := fmt.Sprintf("%d, matching C0 output dimension", c0Length)
		return 0, arrayCardinalityError(dictName, "C1", validationEntryObjectNumber(0, d, "C1"), c1Length, expected)
	}
	return c0Length, nil
}

func validateExponentialFunctionCardinality(a *exponentialFunctionArrays, d types.Dict, dictName string) error {
	if err := validateFunctionArrayLength(a.domain, d, 0, dictName, "Domain", 2, "one input dimension"); err != nil {
		return err
	}
	n, err := validateExponentialOutputDimension(a, d, dictName)
	if err != nil {
		return err
	}
	if a.rangeArray == nil {
		return nil
	}
	return validateFunctionArrayLength(a.rangeArray, d, 0, dictName, "Range", 2*n, "twice the output dimension")
}

func validateExponentialInterpolationFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "exponentialInterpolationFunctionDict"
	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	a, err := validateExponentialFunctionArrays(xRefTable, d, dictName)
	if err != nil {
		return err
	}
	if err = validateExponentialFunctionCardinality(a, d, dictName); err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "N", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.N: %w", dictName, err)
	}

	return nil
}

type stitchingFunctionArrays struct {
	domain     types.Array
	rangeArray types.Array
	functions  types.Array
	bounds     types.Array
	encode     types.Array
}

func (t *functionTraversal) validateStitchingFunctionArrays(d types.Dict, ownerObjNr int, dictName string) (*stitchingFunctionArrays, error) {
	a := &stitchingFunctionArrays{}
	var err error
	if a.domain, err = validateNumberArrayEntry(t.xRefTable, d, ownerObjNr, dictName, "Domain", REQUIRED, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Domain: %w", dictName, err)
	}
	if a.rangeArray, err = validateNumberArrayEntry(t.xRefTable, d, ownerObjNr, dictName, "Range", OPTIONAL, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Range: %w", dictName, err)
	}
	if a.functions, err = validateArrayEntry(t.xRefTable, d, ownerObjNr, dictName, "Functions", REQUIRED, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Functions: %w", dictName, err)
	}
	if a.bounds, err = validateNumberArrayEntry(t.xRefTable, d, ownerObjNr, dictName, "Bounds", REQUIRED, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Bounds: %w", dictName, err)
	}
	if a.encode, err = validateNumberArrayEntry(t.xRefTable, d, ownerObjNr, dictName, "Encode", REQUIRED, model.V13, nil); err != nil {
		return nil, fmt.Errorf("%s.Encode: %w", dictName, err)
	}
	return a, nil
}

func validateStitchingFunctionCardinality(xRefTable *model.XRefTable, a *stitchingFunctionArrays, d types.Dict, ownerObjNr int, dictName string) ([]error, error) {
	if err := validateFunctionArrayLength(a.domain, d, ownerObjNr, dictName, "Domain", 2, "one input dimension"); err != nil {
		return nil, err
	}
	var strictFailures []error
	if a.rangeArray != nil {
		if _, err := validateFunctionDimension(a.rangeArray, d, ownerObjNr, dictName, "Range"); err != nil {
			if xRefTable.ValidationMode != model.ValidationRelaxed || len(a.rangeArray) != 0 {
				return nil, err
			}
			strictFailures = append(strictFailures, err)
		}
	}
	k := len(a.functions)
	if k == 0 {
		return nil, arrayCardinalityError(dictName, "Functions", validationEntryObjectNumber(ownerObjNr, d, "Functions"), 0,
			"one or more subfunctions")
	}
	boundCount := len(a.bounds)
	boundCountExpected := k - 1
	if boundCount != boundCountExpected {
		err := validateFunctionArrayLength(a.bounds, d, ownerObjNr, dictName, "Bounds", boundCountExpected,
			"one fewer than Functions")
		// Older FOP versions omitted the final bound for repeated terminal gradient stops.
		if xRefTable.ValidationMode != model.ValidationRelaxed || boundCount != boundCountExpected-1 {
			return nil, err
		}
		strictFailures = append(strictFailures, err)
	}
	if err := validateFunctionArrayLength(a.encode, d, ownerObjNr, dictName, "Encode", 2*k, "twice the Functions count"); err != nil {
		return nil, err
	}
	return strictFailures, nil
}

func (t *functionTraversal) validateStitchingSubfunctions(a *stitchingFunctionArrays, d types.Dict, ownerObjNr, depth int) error {
	objNr := validationEntryObjectNumber(ownerObjNr, d, "Functions")
	for _, o := range a.functions {
		if err := t.validateFunction(o, objNr, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (t *functionTraversal) validateStitchingFunctionDict(d types.Dict, ownerObjNr, depth int) error {
	xRefTable := t.xRefTable
	dictName := "stitchingFunctionDict"
	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	a, err := t.validateStitchingFunctionArrays(d, ownerObjNr, dictName)
	if err != nil {
		return err
	}
	strictFailures, err := validateStitchingFunctionCardinality(xRefTable, a, d, ownerObjNr, dictName)
	if err != nil {
		return err
	}
	if err = t.validateStitchingSubfunctions(a, d, ownerObjNr, depth); err != nil {
		return err
	}
	for _, strictFailure := range strictFailures {
		xRefTable.AddValidationNotice(model.NewValidationNotice(
			model.NoticePhaseValidate,
			model.NoticeDigested,
			strictFailure.Error(),
			strictFailure,
		))
	}
	return nil
}

type sampledFunctionArrays struct {
	domain     types.Array
	rangeArray types.Array
	size       types.Array
	encode     types.Array
	decode     types.Array
}

func validateSampledFunctionArrays(xRefTable *model.XRefTable, d types.Dict, dictName string, version model.Version) (*sampledFunctionArrays, error) {
	a := &sampledFunctionArrays{}
	var err error
	if a.domain, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Domain", REQUIRED, version, nil); err != nil {
		return nil, fmt.Errorf("%s.Domain: %w", dictName, err)
	}
	if a.rangeArray, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Range", REQUIRED, version, nil); err != nil {
		return nil, fmt.Errorf("%s.Range: %w", dictName, err)
	}
	if a.size, err = validateIntegerArrayEntry(xRefTable, d, 0, dictName, "Size", REQUIRED, version, nil); err != nil {
		return nil, fmt.Errorf("%s.Size: %w", dictName, err)
	}
	if a.encode, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Encode", OPTIONAL, version, nil); err != nil {
		return nil, fmt.Errorf("%s.Encode: %w", dictName, err)
	}
	if a.decode, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Decode", OPTIONAL, version, nil); err != nil {
		return nil, fmt.Errorf("%s.Decode: %w", dictName, err)
	}
	return a, nil
}

func validateSampledFunctionCardinality(a *sampledFunctionArrays, d types.Dict, dictName string) error {
	m, err := validateFunctionDimension(a.domain, d, 0, dictName, "Domain")
	if err != nil {
		return err
	}
	n, err := validateFunctionDimension(a.rangeArray, d, 0, dictName, "Range")
	if err != nil {
		return err
	}
	if err = validateFunctionArrayLength(a.size, d, 0, dictName, "Size", m, "matching Domain input dimension"); err != nil {
		return err
	}
	if a.encode != nil {
		if err = validateFunctionArrayLength(a.encode, d, 0, dictName, "Encode", 2*m, "twice the input dimension"); err != nil {
			return err
		}
	}
	if a.decode == nil {
		return nil
	}
	return validateFunctionArrayLength(a.decode, d, 0, dictName, "Decode", 2*n, "twice the output dimension")
}

func validateSampledFunctionStreamDictVersion(xRefTable *model.XRefTable, sd *types.StreamDict, version model.Version) error {
	dictName := "sampledFunctionStreamDict"
	err := xRefTable.ValidateVersion(dictName, version)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	a, err := validateSampledFunctionArrays(xRefTable, sd.Dict, dictName, version)
	if err != nil {
		return err
	}
	if err = validateSampledFunctionCardinality(a, sd.Dict, dictName); err != nil {
		return err
	}

	validate := func(i int) bool { return types.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32}) }
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "BitsPerSample", REQUIRED, version, validate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerSample: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Order", OPTIONAL, version, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return fmt.Errorf("%s.Order: %w", dictName, err)
	}

	return nil
}

func validateSampledFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	version := model.V12
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		version = model.V11
	}
	err := validateSampledFunctionStreamDictVersion(xRefTable, sd, version)
	if err != nil {
		return err
	}
	if xRefTable.Version() < model.V12 {
		showDigestedVersionViolation(xRefTable, "sampledFunctionStreamDict")
	}
	return nil
}

func validatePostScriptCalculatorFunctionStreamDictVersion(xRefTable *model.XRefTable, sd *types.StreamDict, version model.Version) error {
	dictName := "postScriptCalculatorFunctionStreamDict"
	err := xRefTable.ValidateVersion(dictName, version)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	domain, err := validateNumberArrayEntry(xRefTable, sd.Dict, 0, dictName, "Domain", REQUIRED, version, nil)
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	rangeArray, err := validateNumberArrayEntry(xRefTable, sd.Dict, 0, dictName, "Range", REQUIRED, version, nil)
	if err != nil {
		return fmt.Errorf("%s.Range: %w", dictName, err)
	}
	if _, err = validateFunctionDimension(domain, sd.Dict, 0, dictName, "Domain"); err != nil {
		return err
	}
	_, err = validateFunctionDimension(rangeArray, sd.Dict, 0, dictName, "Range")
	return err
}

func validatePostScriptCalculatorFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	version := model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		version = model.V12
	}
	err := validatePostScriptCalculatorFunctionStreamDictVersion(xRefTable, sd, version)
	if err != nil {
		return err
	}
	if xRefTable.Version() < model.V13 {
		showDigestedVersionViolation(xRefTable, "postScriptCalculatorFunctionStreamDict")
	}
	return nil
}

type functionTraversal struct {
	c         context.Context
	xRefTable *model.XRefTable
	ancestors map[int]bool
	validated map[int]int
}

func newFunctionTraversal(c context.Context, xRefTable *model.XRefTable) *functionTraversal {
	return &functionTraversal{
		c:         c,
		xRefTable: xRefTable,
		ancestors: map[int]bool{},
		validated: map[int]int{},
	}
}

func functionObjectIdentity(o types.Object) int {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0
	}
	return ir.ObjectNumber.Value()
}

func (t *functionTraversal) enter(objNr int) error {
	if objNr <= 0 {
		return nil
	}
	if t.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, model.ErrFunctionCycle)
	}
	t.ancestors[objNr] = true
	return nil
}

func (t *functionTraversal) leave(objNr int) {
	if objNr > 0 {
		delete(t.ancestors, objNr)
	}
}

func (t *functionTraversal) alreadyValidated(objNr, depth int) bool {
	if objNr <= 0 {
		return false
	}
	validatedDepth, ok := t.validated[objNr]
	return ok && depth <= validatedDepth
}

func (t *functionTraversal) markValidated(objNr, depth int) {
	if objNr <= 0 {
		return
	}
	if previous, ok := t.validated[objNr]; !ok || depth > previous {
		t.validated[objNr] = depth
	}
}

func (t *functionTraversal) processFunctionDict(d types.Dict, ownerObjNr, depth int) error {
	xRefTable := t.xRefTable
	funcType, err := validateIntegerEntry(xRefTable, d, 0, "functionDict", "FunctionType", REQUIRED, model.V10, func(i int) bool { return i == 2 || i == 3 })
	if err != nil {
		return fmt.Errorf("function dictionary: FunctionType: %w", err)
	}

	switch *funcType {

	case 2:
		if err = validateExponentialInterpolationFunctionDict(xRefTable, d); err != nil {
			return fmt.Errorf("exponential interpolation function: %w", err)
		}

	case 3:
		if err = t.validateStitchingFunctionDict(d, ownerObjNr, depth); err != nil {
			return fmt.Errorf("stitching function: %w", err)
		}

	}

	return nil
}

func processFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	funcType, err := validateIntegerEntry(xRefTable, sd.Dict, 0, "functionDict", "FunctionType", REQUIRED, model.V10, func(i int) bool { return i == 0 || i == 4 })
	if err != nil {
		return fmt.Errorf("function stream dictionary: FunctionType: %w", err)
	}

	switch *funcType {
	case 0:
		if err = validateSampledFunctionStreamDict(xRefTable, sd); err != nil {
			return fmt.Errorf("sampled function: %w", err)
		}

	case 4:
		if err = validatePostScriptCalculatorFunctionStreamDict(xRefTable, sd); err != nil {
			return fmt.Errorf("PostScript calculator function: %w", err)
		}

	}

	return nil
}

func (t *functionTraversal) processFunction(o types.Object, ownerObjNr, depth int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	// Function dict: dict or stream dict with required entry "FunctionType" (integer):
	// 0: Sampled function (stream dict)
	// 2: Exponential interpolation function (dict)
	// 3: Stitching function (dict)
	// 4: PostScript calculator function (stream dict), since V1.3

	switch o := o.(type) {

	case types.Dict:

		// process function  2,3
		err = t.processFunctionDict(o, ownerObjNr, depth)

	case types.StreamDict:

		// process function  0,4
		err = processFunctionStreamDict(t.xRefTable, &o)

	default:
		return fmt.Errorf("function object: expected dict or stream dict, got %T", o)
	}

	return err
}

func (t *functionTraversal) validateFunction(o types.Object, ownerObjNr, depth int) (err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	functionObjNr := functionObjectIdentity(o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()
	if err := contextutil.Check(t.c); err != nil {
		return err
	}
	if err := t.xRefTable.CheckRecursionDepth("function graph", depth); err != nil {
		return err
	}
	if err := t.enter(functionObjNr); err != nil {
		return err
	}
	defer t.leave(functionObjNr)
	if t.alreadyValidated(functionObjNr, depth) {
		return nil
	}

	o, err = t.xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("function: dereference: %w", err)
	}
	if o == nil {
		return errors.New("function: missing object")
	}

	if err = t.processFunction(o, objNr, depth); err != nil {
		return err
	}
	t.markValidated(functionObjNr, depth)
	return nil
}

func validateFunction(c context.Context, xRefTable *model.XRefTable, o types.Object, ownerObjNr int) error {
	return newFunctionTraversal(c, xRefTable).validateFunction(o, ownerObjNr, 0)
}

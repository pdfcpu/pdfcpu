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
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// see 7.10 Functions

func validateExponentialInterpolationFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "exponentialInterpolationFunctionDict"
	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Domain", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Range", OPTIONAL, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Range: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C0", OPTIONAL, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.C0: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C1", OPTIONAL, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.C1: %w", dictName, err)
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "N", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.N: %w", dictName, err)
	}

	return nil
}

func validateStitchingFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "stitchingFunctionDict"
	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Domain", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Range", OPTIONAL, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Range: %w", dictName, err)
	}

	_, err = validateFunctionArrayEntry(xRefTable, d, dictName, "Functions", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Functions: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Bounds", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Bounds: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Encode", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Encode: %w", dictName, err)
	}

	return nil
}

func validateSampledFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	dictName := "sampledFunctionStreamDict"
	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V12)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Domain", REQUIRED, model.V12, nil)
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Range", REQUIRED, model.V12, nil)
	if err != nil {
		return fmt.Errorf("%s.Range: %w", dictName, err)
	}

	_, err = validateIntegerArrayEntry(xRefTable, sd.Dict, dictName, "Size", REQUIRED, model.V12, nil)
	if err != nil {
		return fmt.Errorf("%s.Size: %w", dictName, err)
	}

	validate := func(i int) bool { return types.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32}) }
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "BitsPerSample", REQUIRED, model.V12, validate)
	if err != nil {
		return fmt.Errorf("%s.BitsPerSample: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Order", OPTIONAL, model.V12, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return fmt.Errorf("%s.Order: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Encode", OPTIONAL, model.V12, nil)
	if err != nil {
		return fmt.Errorf("%s.Encode: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Decode", OPTIONAL, model.V12, nil)
	if err != nil {
		return fmt.Errorf("%s.Decode: %w", dictName, err)
	}

	return nil
}

func validatePostScriptCalculatorFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	dictName := "postScriptCalculatorFunctionStreamDict"
	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Domain", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Domain: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Range", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.Range: %w", dictName, err)
	}

	return nil
}

func processFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {
	funcType, err := validateIntegerEntry(xRefTable, d, "functionDict", "FunctionType", REQUIRED, model.V10, func(i int) bool { return i == 2 || i == 3 })
	if err != nil {
		return fmt.Errorf("function dictionary: FunctionType: %w", err)
	}

	switch *funcType {

	case 2:
		if err = validateExponentialInterpolationFunctionDict(xRefTable, d); err != nil {
			return fmt.Errorf("exponential interpolation function: %w", err)
		}

	case 3:
		if err = validateStitchingFunctionDict(xRefTable, d); err != nil {
			return fmt.Errorf("stitching function: %w", err)
		}

	}

	return nil
}

func processFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	funcType, err := validateIntegerEntry(xRefTable, sd.Dict, "functionDict", "FunctionType", REQUIRED, model.V10, func(i int) bool { return i == 0 || i == 4 })
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

func processFunction(xRefTable *model.XRefTable, o types.Object) (err error) {
	// Function dict: dict or stream dict with required entry "FunctionType" (integer):
	// 0: Sampled function (stream dict)
	// 2: Exponential interpolation function (dict)
	// 3: Stitching function (dict)
	// 4: PostScript calculator function (stream dict), since V1.3

	switch o := o.(type) {

	case types.Dict:

		// process function  2,3
		err = processFunctionDict(xRefTable, o)

	case types.StreamDict:

		// process function  0,4
		err = processFunctionStreamDict(xRefTable, &o)

	default:
		return fmt.Errorf("function object: expected dict or stream dict, got %T", o)
	}

	return err
}

func validateFunction(xRefTable *model.XRefTable, o types.Object) error {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("function: dereference: %w", err)
	}
	if o == nil {
		return errors.New("function: missing object")
	}

	return processFunction(xRefTable, o)
}

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// see 7.10 Functions

func validateExponentialInterpolationFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "exponentialInterpolationFunctionDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Domain", REQUIRED, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Range", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C0", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C1", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "N", REQUIRED, model.V13, nil)

	return err
}

func validateStitchingFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "stitchingFunctionDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Domain", REQUIRED, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Range", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateFunctionArrayEntry(xRefTable, d, dictName, "Functions", REQUIRED, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Bounds", REQUIRED, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Encode", REQUIRED, model.V13, nil)

	return err
}

func validateSampledFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {

	dictName := "sampledFunctionStreamDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V12)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Domain", REQUIRED, model.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Range", REQUIRED, model.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerArrayEntry(xRefTable, sd.Dict, dictName, "Size", REQUIRED, model.V12, nil)
	if err != nil {
		return err
	}

	validate := func(i int) bool { return types.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32}) }
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "BitsPerSample", REQUIRED, model.V12, validate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Order", OPTIONAL, model.V12, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Encode", OPTIONAL, model.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Decode", OPTIONAL, model.V12, nil)

	return err
}

func validatePostScriptCalculatorFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {

	dictName := "postScriptCalculatorFunctionStreamDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, model.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Domain", REQUIRED, model.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Range", REQUIRED, model.V13, nil)

	return err
}

func processFunctionDict(xRefTable *model.XRefTable, d types.Dict) error {

	funcType, err := validateIntegerEntry(xRefTable, d, "functionDict", "FunctionType", REQUIRED, model.V10, func(i int) bool { return i == 2 || i == 3 })
	if err != nil {
		return err
	}

	switch *funcType {

	case 2:
		err = validateExponentialInterpolationFunctionDict(xRefTable, d)

	case 3:
		err = validateStitchingFunctionDict(xRefTable, d)

	}

	return err
}

func processFunctionStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {

	funcType, err := validateIntegerEntry(xRefTable, sd.Dict, "functionDict", "FunctionType", REQUIRED, model.V10, func(i int) bool { return i == 0 || i == 4 })
	if err != nil {
		return err
	}

	switch *funcType {
	case 0:
		err = validateSampledFunctionStreamDict(xRefTable, sd)

	case 4:
		err = validatePostScriptCalculatorFunctionStreamDict(xRefTable, sd)

	}

	return err
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
		return errors.New("pdfcpu: processFunction: obj must be dict or stream dict")
	}

	return err
}

func validateFunction(xRefTable *model.XRefTable, o types.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		return errors.New("pdfcpu: validateFunction: missing object")
	}

	return processFunction(xRefTable, o)
}

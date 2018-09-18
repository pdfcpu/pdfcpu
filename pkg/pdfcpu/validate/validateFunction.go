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

// see 7.10 Functions

func validateExponentialInterpolationFunctionDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "exponentialInterpolationFunctionDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, pdf.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Range", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C0", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C1", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "N", REQUIRED, pdf.V13, nil)

	return err
}

func validateStitchingFunctionDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "stitchingFunctionDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, pdf.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Domain", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Range", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateFunctionArrayEntry(xRefTable, dict, dictName, "Functions", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Bounds", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Encode", REQUIRED, pdf.V13, nil)

	return err
}

func validateSampledFunctionStreamDict(xRefTable *pdf.XRefTable, dict *pdf.StreamDict) error {

	dictName := "sampledFunctionStreamDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, pdf.V12)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Domain", REQUIRED, pdf.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Range", REQUIRED, pdf.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerArrayEntry(xRefTable, &dict.PDFDict, dictName, "Size", REQUIRED, pdf.V12, nil)
	if err != nil {
		return err
	}

	validate := func(i int) bool { return pdf.IntMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32}) }
	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "BitsPerSample", REQUIRED, pdf.V12, validate)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Order", OPTIONAL, pdf.V12, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Encode", OPTIONAL, pdf.V12, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Decode", OPTIONAL, pdf.V12, nil)

	return err
}

func validatePostScriptCalculatorFunctionStreamDict(xRefTable *pdf.XRefTable, dict *pdf.StreamDict) error {

	dictName := "postScriptCalculatorFunctionStreamDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, pdf.V13)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Domain", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict.PDFDict, dictName, "Range", REQUIRED, pdf.V13, nil)

	return err
}

func processFunctionDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	funcType, err := validateIntegerEntry(xRefTable, dict, "functionDict", "FunctionType", REQUIRED, pdf.V10, func(i int) bool { return i == 2 || i == 3 })
	if err != nil {
		return err
	}

	switch *funcType {

	case 2:
		err = validateExponentialInterpolationFunctionDict(xRefTable, dict)

	case 3:
		err = validateStitchingFunctionDict(xRefTable, dict)

	}

	return err
}

func processFunctionStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict) error {

	funcType, err := validateIntegerEntry(xRefTable, &sd.PDFDict, "functionDict", "FunctionType", REQUIRED, pdf.V10, func(i int) bool { return i == 0 || i == 4 })
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

func processFunction(xRefTable *pdf.XRefTable, obj pdf.Object) (err error) {

	// Function dict: dict or stream dict with required entry "FunctionType" (integer):
	// 0: Sampled function (stream dict)
	// 2: Exponential interpolation function (dict)
	// 3: Stitching function (dict)
	// 4: PostScript calculator function (stream dict), since V1.3

	switch obj := obj.(type) {

	case pdf.PDFDict:

		// process function  2,3
		err = processFunctionDict(xRefTable, &obj)

	case pdf.StreamDict:

		// process function  0,4
		err = processFunctionStreamDict(xRefTable, &obj)

	default:
		return errors.New("processFunction: obj must be dict or stream dict")
	}

	return err
}

func validateFunction(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		return errors.New("writeFunction: missing object")
	}

	return processFunction(xRefTable, obj)
}

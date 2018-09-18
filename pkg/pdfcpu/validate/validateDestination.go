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

func validateDestinationArrayFirstElement(xRefTable *pdf.XRefTable, arr *pdf.Array) (pdf.Object, error) {

	obj, err := xRefTable.Dereference((*arr)[0])
	if err != nil || obj == nil {
		return nil, err
	}

	switch obj := obj.(type) {

	case pdf.Integer: // no further processing

	case pdf.PDFDict:
		if obj.Type() == nil || *obj.Type() != "Page" {
			err = errors.New("validateDestinationArrayFirstElement: first element refers to invalid destination page dict")
		}

	default:
		err = errors.Errorf("validateDestinationArrayFirstElement: first element must be a pageDict indRef or an integer: %v", obj)
	}

	return obj, err
}

func validateDestinationArrayLength(arr pdf.Array) bool {
	l := len(arr)
	return l == 2 || l == 3 || l == 5 || l == 6
}

func validateDestinationArray(xRefTable *pdf.XRefTable, arr *pdf.Array) error {

	// Validate first element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.

	obj, err := validateDestinationArrayFirstElement(xRefTable, arr)
	if err != nil || obj == nil {
		return err
	}

	if !validateDestinationArrayLength(*arr) {
		return errors.New("validateDestinationArray: invalid length")
	}

	// Validate rest of array elements.

	name, ok := (*arr)[1].(pdf.Name)
	if !ok {
		return errors.New("validateDestinationArray: second element must be a name")
	}

	var nameErr bool

	switch len(*arr) {

	case 2:
		if xRefTable.ValidationMode == pdf.ValidationRelaxed {
			nameErr = !pdf.MemberOf(name.Value(), []string{"Fit", "FitB", "FitH"})
		} else {
			nameErr = !pdf.MemberOf(name.Value(), []string{"Fit", "FitB"})
		}

	case 3:
		nameErr = name.Value() != "FitH" && name.Value() != "FitV" && name.Value() != "FitBH"

	case 5:
		nameErr = name.Value() != "XYZ"

	case 6:
		nameErr = name.Value() != "FitR"

	default:
		return errors.Errorf("validateDestinationArray: array length %d not allowed: %v", len(*arr), arr)
	}

	if nameErr {
		return errors.New("validateDestinationArray: arr[1] corrupt")
	}

	return nil
}

func validateDestinationDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	// D, required, array
	arr, err := validateArrayEntry(xRefTable, dict, "DestinationDict", "D", REQUIRED, pdf.V10, nil)
	if err != nil || arr == nil {
		return err
	}

	return validateDestinationArray(xRefTable, arr)
}

func validateDestination(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Name:
		// no further processing.

	case pdf.StringLiteral:
		// no further processing.

	case pdf.PDFDict:
		err = validateDestinationDict(xRefTable, &obj)

	case pdf.Array:
		err = validateDestinationArray(xRefTable, &obj)

	default:
		err = errors.New("validateDestination: unsupported PDF object")

	}

	return err
}

func validateDestinationEntry(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 12.3.2

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil {
		return err
	}

	return validateDestination(xRefTable, obj)
}

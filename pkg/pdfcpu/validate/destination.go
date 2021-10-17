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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateDestinationArrayFirstElement(xRefTable *pdf.XRefTable, a pdf.Array) (pdf.Object, error) {

	o, err := xRefTable.Dereference(a[0])
	if err != nil || o == nil {
		return nil, err
	}

	switch o := o.(type) {

	case pdf.Integer, pdf.Name: // no further processing

	case pdf.Dict:
		if o.Type() == nil || (*o.Type() != "Page" && *o.Type() != "Pages") {
			err = errors.New("pdfcpu: validateDestinationArrayFirstElement: first element refers to invalid destination page dict" + *o.Type())
		}

	default:
		err = errors.Errorf("pdfcpu: validateDestinationArrayFirstElement: first element must be a pageDict indRef or an integer: %v (%T)", o, o)
	}

	return o, err
}

func validateDestinationArrayLength(a pdf.Array) bool {
	l := len(a)
	return l == 2 || l == 3 || l == 5 || l == 6 || l == 4 // 4 = hack! see below
}

func validateDestinationArray(xRefTable *pdf.XRefTable, a pdf.Array) error {

	// Validate first element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.

	o, err := validateDestinationArrayFirstElement(xRefTable, a)
	if err != nil || o == nil {
		return err
	}

	if !validateDestinationArrayLength(a) {
		return errors.Errorf("pdfcpu: validateDestinationArray: invalid length: %d", len(a))
	}

	// NOTE if len == 4 we possible have a missing first element, which should be an indRef to the dest page.
	// TODO Investigate.
	i := 1
	// if len(a) == 4 {
	// 	i = 0
	// }

	// Validate rest of array elements.

	name, ok := a[i].(pdf.Name)
	if !ok {
		return errors.Errorf("pdfcpu: validateDestinationArray: second element must be a name %v (%d)", a[i], i)
	}

	var nameErr bool

	switch len(a) {

	case 2:
		if xRefTable.ValidationMode == pdf.ValidationRelaxed {
			nameErr = !pdf.MemberOf(name.Value(), []string{"Fit", "FitB", "FitH"})
		} else {
			nameErr = !pdf.MemberOf(name.Value(), []string{"Fit", "FitB"})
		}

	case 3:
		nameErr = name.Value() != "FitH" && name.Value() != "FitV" && name.Value() != "FitBH"

	case 4:
		// TODO Cleanup
		// hack for i381 - possibly zoom == null or 0
		// eg. [(886 0 R) XYZ 53 303]
		nameErr = name.Value() != "XYZ"

	case 5:
		nameErr = name.Value() != "XYZ"

	case 6:
		nameErr = name.Value() != "FitR"

	default:
		return errors.Errorf("validateDestinationArray: array length %d not allowed: %v", len(a), a)
	}

	if nameErr {
		return errors.New("pdfcpu: validateDestinationArray: arr[1] corrupt")
	}

	return nil
}

func validateDestinationDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// D, required, array
	a, err := validateArrayEntry(xRefTable, d, "DestinationDict", "D", REQUIRED, pdf.V10, nil)
	if err != nil || a == nil {
		return err
	}

	return validateDestinationArray(xRefTable, a)
}

func validateDestination(xRefTable *pdf.XRefTable, o pdf.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		// no further processing.

	case pdf.StringLiteral:
		// no further processing.

	case pdf.HexLiteral:
		// no further processing.

	case pdf.Dict:
		err = validateDestinationDict(xRefTable, o)

	case pdf.Array:
		err = validateDestinationArray(xRefTable, o)

	default:
		err = errors.New("pdfcpu: validateDestination: unsupported PDF object")

	}

	return err
}

func validateDestinationEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 12.3.2

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil {
		return err
	}

	return validateDestination(xRefTable, o)
}

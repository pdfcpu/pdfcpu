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

func validateDestinationArrayFirstElement(xRefTable *model.XRefTable, a types.Array) (types.Object, error) {

	o, err := xRefTable.Dereference(a[0])
	if err != nil || o == nil {
		return nil, err
	}

	switch o := o.(type) {

	case types.Integer, types.Name: // no further processing

	case types.Dict:
		if o.Type() == nil || (o.Type() != nil && (*o.Type() != "Page" && *o.Type() != "Pages")) {
			err = errors.Errorf("pdfcpu: validateDestinationArrayFirstElement: must be a pageDict indRef or an integer: %v (%T)", o, o)
		}

	default:
		err = errors.Errorf("pdfcpu: validateDestinationArrayFirstElement: must be a pageDict indRef or an integer: %v (%T)", o, o)
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			err = nil
		}
	}

	return o, err
}

func validateDestinationArrayLength(a types.Array) bool {
	return len(a) >= 2 && len(a) <= 6
}

func validateDestinationArray(xRefTable *model.XRefTable, a types.Array) error {
	if !validateDestinationArrayLength(a) {
		return errors.Errorf("pdfcpu: validateDestinationArray: invalid length: %d", len(a))
	}

	// Validate first element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.
	o, err := validateDestinationArrayFirstElement(xRefTable, a)
	if err != nil || o == nil {
		return err
	}

	name, ok := a[1].(types.Name)
	if !ok {
		return errors.Errorf("pdfcpu: validateDestinationArray: second element must be a name %v", a[1])
	}

	switch name {
	case "Fit":
	case "FitB":
		if len(a) > 2 {
			return errors.Errorf("pdfcpu: validateDestinationArray: %s - invalid length: %d", name, len(a))
		}
	case "FitH":
	case "FitV":
	case "FitBH":
	case "FitBV":
		if len(a) > 3 {
			return errors.Errorf("pdfcpu: validateDestinationArray: %s - invalid length: %d", name, len(a))
		}
	case "XYZ":
		if len(a) > 5 {
			return errors.Errorf("pdfcpu: validateDestinationArray: %s - invalid length: %d", name, len(a))
		}
	case "FitR":
		if len(a) > 6 {
			return errors.Errorf("pdfcpu: validateDestinationArray: %s - invalid length: %d", name, len(a))
		}
	default:
		return errors.Errorf("pdfcpu: validateDestinationArray     j- invalid mode: %s", name)
	}

	return nil
}

func validateDestinationDict(xRefTable *model.XRefTable, d types.Dict) error {

	// D, required, array
	a, err := validateArrayEntry(xRefTable, d, "DestinationDict", "D", REQUIRED, model.V10, nil)
	if err != nil || a == nil {
		return err
	}

	return validateDestinationArray(xRefTable, a)
}

func validateDestination(xRefTable *model.XRefTable, o types.Object, forAction bool) (string, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return "", err
	}

	switch o := o.(type) {

	case types.Name:
		return o.Value(), nil

	case types.StringLiteral:
		return types.StringLiteralToString(o)

	case types.HexLiteral:
		return types.HexLiteralToString(o)

	case types.Dict:
		if forAction {
			return "", errors.New("pdfcpu: validateDestination: unsupported PDF object")
		}
		err = validateDestinationDict(xRefTable, o)

	case types.Array:
		err = validateDestinationArray(xRefTable, o)

	default:
		err = errors.New("pdfcpu: validateDestination: unsupported PDF object")

	}

	return "", err
}

func validateActionDestinationEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {

	// see 12.3.2

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil {
		return err
	}

	name, err := validateDestination(xRefTable, o, true)
	if err != nil {
		return err
	}

	if len(name) > 0 && xRefTable.IsMerging() {
		nm := xRefTable.NameRef("Dests")
		nm.Add(name, d)
	}

	return nil
}

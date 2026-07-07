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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func indirectRefObjectNumber(o types.Object) (int, bool) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0, false
	}
	return ir.ObjectNumber.Value(), true
}

func dictEntryContext(dictName, entryName string, o types.Object) string {
	context := dictName + "." + entryName
	if objNr, ok := indirectRefObjectNumber(o); ok {
		context = fmt.Sprintf("%s obj#%d", context, objNr)
	}
	return context
}

func objectContext(context string, o types.Object) string {
	if objNr, ok := indirectRefObjectNumber(o); ok {
		return fmt.Sprintf("%s obj#%d", context, objNr)
	}
	return context
}

func validateDestinationArrayFirstElement(xRefTable *model.XRefTable, a types.Array) (types.Object, error) {
	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		return nil, fmt.Errorf("destination array[0]: dereference page: %w", err)
	}
	if o == nil {
		return nil, nil
	}

	s := "destination array: first element is not a page dict: " + a.String()

	switch o := o.(type) {

	case types.Dict:
		if o.Type() == nil || (o.Type() != nil && (*o.Type() != "Page" && *o.Type() != "Pages")) {
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				model.ShowDigestedSpecViolation(s)
				return nil, nil
			}
			dictType := "<missing>"
			if o.Type() != nil {
				dictType = *o.Type()
			}
			err = fmt.Errorf("destination array[0]: expected page dict, got dict type %q", dictType)
		}

	default:
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowDigestedSpecViolation(s)
			return nil, nil
		}
		err = fmt.Errorf("destination array[0]: expected page dict, got %T", o)
	}

	return o, err
}

func validateDestinationArrayLength(a types.Array) bool {
	return len(a) >= 2 && len(a) <= 6
}

func validateDestType(a types.Array, destType types.Name) error {
	switch destType {
	case "Fit":
	case "FitB":
		if len(a) > 2 {
			return fmt.Errorf("destination array mode %s: invalid length %d", destType, len(a))
		}
	case "FitH":
	case "FitV":
	case "FitBH":
	case "FitBV":
		if len(a) > 3 {
			return fmt.Errorf("destination array mode %s: invalid length %d", destType, len(a))
		}
	case "XYZ":
		if len(a) > 5 {
			return fmt.Errorf("destination array mode %s: invalid length %d", destType, len(a))
		}
	case "FitR":
		if len(a) > 6 {
			return fmt.Errorf("destination array mode %s: invalid length %d", destType, len(a))
		}
	default:
		return fmt.Errorf("destination array mode: invalid mode %q", destType)
	}

	return nil
}

func validateDestinationArray(xRefTable *model.XRefTable, a types.Array) error {
	if !validateDestinationArrayLength(a) {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("destination array: invalid length %d", len(a))
		}
		return nil
	}

	// Validate first element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.
	o, err := validateDestinationArrayFirstElement(xRefTable, a)
	if err != nil || o == nil {
		return err
	}

	name, ok := a[1].(types.Name)
	if !ok {
		return fmt.Errorf("destination array[1]: expected name, got %T", a[1])
	}

	return validateDestType(a, name)
}

func validateDestinationDict(xRefTable *model.XRefTable, d types.Dict) error {
	// D, required, array
	o, _ := d.Find("D")
	a, err := validateArrayEntry(xRefTable, d, "DestinationDict", "D", REQUIRED, model.V10, nil)
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext("destination dictionary", "D", o), err)
		}
		return nil
	}

	if err := validateDestinationArray(xRefTable, a); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext("destination dictionary", "D", o), err)
	}

	return nil
}

func validateDestination(xRefTable *model.XRefTable, o types.Object, forAction bool) (string, error) {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", fmt.Errorf("destination: dereference: %w", err)
	}
	if o == nil {
		return "", nil
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
			return "", fmt.Errorf("destination: action destination cannot be dict")
		}
		err = validateDestinationDict(xRefTable, o)

	case types.Array:
		err = validateDestinationArray(xRefTable, o)

	default:
		err = fmt.Errorf("destination: unsupported object type %T", o)

	}

	return "", err
}

func validateActionDestinationEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	// see 12.3.2

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, d[entryName]), err)
	}

	name, err := validateDestination(xRefTable, o, true)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, d[entryName]), err)
	}

	if len(name) > 0 && xRefTable.IsMerging() {
		nm := xRefTable.NameRef("Dests")
		nm.Add(name, d)
	}

	return nil
}

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
	"github.com/pkg/errors"
)

func validatePageLabelDict(xRefTable *model.XRefTable, o types.Object) error {

	// see 12.4.2 Page Labels

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictName := "pageLabelDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "PageLabel" })
	if err != nil {
		return err
	}

	// Optional name entry S
	// The numbering style that shall be used for the numeric portion of each page label.
	validate := func(s string) bool { return types.MemberOf(s, []string{"D", "R", "r", "A", "a"}) }
	_, err = validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}

	// Optional string entry P
	// Label prefix for page labels in this range.
	_, err = validateStringEntry(xRefTable, d, dictName, "P", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Optional integer entry St
	// The value of the numeric portion for the first page label in the range.
	_, err = validateIntegerEntry(xRefTable, d, dictName, "St", OPTIONAL, model.V10, func(i int) bool { return i >= 1 })

	return err
}

func validateNumberTreeDictNumsEntry(xRefTable *model.XRefTable, d types.Dict, name string) (firstKey, lastKey int, err error) {

	// Nums: array of the form [key1 value1 key2 value2 ... key n value n]
	o, found := d.Find("Nums")
	if !found {
		return 0, 0, errors.New("pdfcpu: validateNumberTreeDictNumsEntry: missing \"Kids\" or \"Nums\" entry")
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return 0, 0, err
	}
	if a == nil {
		return 0, 0, errors.New("pdfcpu: validateNumberTreeDictNumsEntry: missing \"Nums\" array")
	}

	// arr length needs to be even because of contained key value pairs.
	if len(a)%2 == 1 {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return 0, 0, errors.Errorf("pdfcpu: validateNumberTreeDictNumsEntry: Nums array entry length needs to be even, length=%d\n", len(a))
		}
		model.ShowDigestedSpecViolation("number tree \"Num\" entry array length needs to be even")
		model.ShowSkipped("invalid number tree")
		return 0, 0, nil
	}

	// every other entry is a value
	// value = indRef to an array of indRefs of structElemDicts
	// or
	// value = indRef of structElementDict.

	for i, o := range a {

		if i%2 == 0 {

			o, err = xRefTable.Dereference(o)
			if err != nil {
				return 0, 0, err
			}

			i, ok := o.(types.Integer)
			if !ok {
				return 0, 0, errors.Errorf("pdfcpu: validateNumberTreeDictNumsEntry: corrupt key <%v>\n", o)
			}

			if firstKey == 0 {
				firstKey = i.Value()
			}

			lastKey = i.Value()

			continue
		}

		switch name {

		case "PageLabel":
			err = validatePageLabelDict(xRefTable, o)
			if err != nil {
				return 0, 0, err
			}

		case "StructTree":
			err = validateStructTreeRootDictEntryK(xRefTable, o)
			if err != nil {
				return 0, 0, err
			}
		}

	}

	return firstKey, lastKey, nil
}

func validateNumberTreeDictLimitsEntry(xRefTable *model.XRefTable, d types.Dict, firstKey, lastKey int) error {

	a, err := validateIntegerArrayEntry(xRefTable, d, "numberTreeDict", "Limits", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	fk := 0
	if a[0] != nil {
		fk = a[0].(types.Integer).Value()
	}

	lk := 0
	if a[1] != nil {
		lk = a[1].(types.Integer).Value()
	}

	if firstKey < fk || lastKey > lk {
		msg := fmt.Sprintf("validateNumberTreeDictLimitsEntry: invalid leaf node: firstKey(%d vs. %d) lastKey(%d vs. %d)", firstKey, fk, lastKey, lk)
		if xRefTable.ValidationMode == model.ValidationStrict {
			return errors.Errorf("pdfcpu: %s\n", msg)
		}
		model.ShowDigestedSpecViolation(msg)
	}

	return nil
}

func validateNumberTree(xRefTable *model.XRefTable, name string, d types.Dict, root bool) (firstKey, lastKey int, err error) {

	// A node has "Kids" or "Nums" entry.

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if o, found := d.Find("Kids"); found {

		a, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return 0, 0, err
		}
		if a == nil {
			return 0, 0, errors.New("pdfcpu: validateNumberTree: missing \"Kids\" array")
		}

		for _, o := range a {

			d1, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return 0, 0, err
			}

			var fk int
			fk, lastKey, err = validateNumberTree(xRefTable, name, d1, false)
			if err != nil {
				return 0, 0, err
			}
			if firstKey == 0 {
				firstKey = fk
			}
		}

	} else {

		// Leaf node
		firstKey, lastKey, err = validateNumberTreeDictNumsEntry(xRefTable, d, name)
		if err != nil {
			return 0, 0, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNumberTreeDictLimitsEntry(xRefTable, d, firstKey, lastKey)
		if err != nil {
			return 0, 0, err
		}

	}

	return firstKey, lastKey, nil
}

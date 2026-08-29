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

func validatePageLabelDict(xRefTable *model.XRefTable, o types.Object) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// see 12.4.2 Page Labels

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("PageLabel number tree value: dereference dict: %w", err)
	}
	if d == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped("missing PageLabel number tree value dict")
			return nil
		}
		return errors.New("PageLabel number tree value: missing dict")
	}

	dictName := "pageLabelDict"

	// Type, optional, name
	_, err = validateNameEntry(
		xRefTable, d, objNr, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "PageLabel" },
	)
	if err != nil {
		return err
	}

	// Optional name entry S
	// The numbering style that shall be used for the numeric portion of each page label.
	validate := func(s string) bool { return types.MemberOf(s, []string{"D", "R", "r", "A", "a"}) }
	_, err = validateNameEntry(xRefTable, d, objNr, dictName, "S", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}

	// Optional string entry P
	// Label prefix for page labels in this range.
	_, err = validateStringEntry(xRefTable, d, objNr, dictName, "P", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Optional integer entry St
	// The value of the numeric portion for the first page label in the range.
	_, err = validateIntegerEntry(
		xRefTable, d, objNr, dictName, "St", OPTIONAL, model.V10, func(i int) bool { return i >= 1 },
	)

	return err
}

func validateNumberTreeKey(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, name string) (key int, valid bool, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return 0, false, fmt.Errorf("number tree %s key: dereference: %w", name, err)
	}

	i, ok := o.(types.Integer)
	if ok {
		return i.Value(), true, nil
	}

	err = fmt.Errorf("number tree %s key: expected integer, got %T", name, o)
	if name != "StructTree" {
		return 0, false, err
	}
	if err = handleInvalidStructTreeObject(xRefTable, err); err != nil {
		return 0, false, err
	}
	return 0, false, nil
}

func validateNumberTreeDictNumsEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, name string, useIDs bool) (firstKey, lastKey int, err error) {
	// Nums: array of the form [key1 value1 key2 value2 ... key n value n]
	o, found := d.Find("Nums")
	if !found {
		err = fmt.Errorf("number tree %s: missing Kids or Nums", name)
		return 0, 0, model.WithValidationErrorObject(err, ownerObjNr)
	}
	numsObjNr := validationObjectNumber(ownerObjNr, o)

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		err = fmt.Errorf("number tree %s Nums: dereference array: %w", name, err)
		return 0, 0, model.WithValidationErrorObject(err, numsObjNr)
	}
	if a == nil {
		err = fmt.Errorf("number tree %s: missing Nums array", name)
		return 0, 0, model.WithValidationErrorObject(err, numsObjNr)
	}

	// arr length needs to be even because of contained key value pairs.
	if len(a)%2 == 1 {
		if xRefTable.ValidationMode == model.ValidationStrict {
			err = fmt.Errorf("number tree %s Nums: odd entry count %d", name, len(a))
			return 0, 0, model.WithValidationErrorObject(err, numsObjNr)
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
			var key int
			var valid bool
			key, valid, err = validateNumberTreeKey(xRefTable, o, numsObjNr, name)
			if err != nil {
				return 0, 0, fmt.Errorf("number tree %s Nums[%d]: %w", name, i, err)
			}
			if !valid {
				continue
			}

			if firstKey == 0 {
				firstKey = key
			}

			lastKey = key

			continue
		}

		switch name {

		case "PageLabel":
			err = validatePageLabelDict(xRefTable, o)
			if err != nil {
				err = fmt.Errorf("number tree %s key %d: %w", name, lastKey, err)
				return 0, 0, model.WithValidationErrorObject(err, validationObjectNumber(numsObjNr, o))
			}

		case "StructTree":
			err = validateStructTreeRootDictEntryK(xRefTable, o, useIDs)
			if err != nil {
				err = fmt.Errorf("number tree %s key %d: %w", name, lastKey, err)
				return 0, 0, model.WithValidationErrorObject(err, validationObjectNumber(numsObjNr, o))
			}
		}

	}

	return firstKey, lastKey, nil
}

func validateNumberTreeDictLimitsEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr, firstKey, lastKey int) error {
	a, err := validateIntegerArrayEntry(
		xRefTable, d, ownerObjNr, "numberTreeDict", "Limits", REQUIRED, model.V10,
		func(a types.Array) bool { return len(a) == 2 },
	)
	if err != nil {
		return err
	}

	i, err := xRefTable.DereferenceInteger(a[0])
	if err != nil {
		err = fmt.Errorf("number tree Limits[0]: %w", err)
		return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, a[0]))
	}
	fk := 0
	if i != nil {
		fk = i.Value()
	}

	i, err = xRefTable.DereferenceInteger(a[1])
	if err != nil {
		err = fmt.Errorf("number tree Limits[1]: %w", err)
		return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, a[1]))
	}
	lk := 0
	if i != nil {
		lk = i.Value()
	}

	if firstKey < fk || lastKey > lk {
		msg := fmt.Sprintf("number tree leaf limits: first key %d, minimum %d; last key %d, maximum %d", firstKey, fk, lastKey, lk)
		if xRefTable.ValidationMode == model.ValidationStrict {
			return model.WithValidationErrorObject(errors.New(msg), ownerObjNr)
		}
		model.ShowDigestedSpecViolation(msg)
	}

	return nil
}

func validateNumberTree(xRefTable *model.XRefTable, name string, d types.Dict, ownerObjNr int, root, useIDs bool) (firstKey, lastKey int, err error) {
	return validateNumberTreeDepth(xRefTable, name, d, ownerObjNr, root, useIDs, 0)
}

func numberTreeKidContext(name string, o types.Object, i int) string {
	if ir, ok := o.(types.IndirectRef); ok {
		return fmt.Sprintf("number tree %s Kids[%d] obj#%d", name, i, ir.ObjectNumber.Value())
	}
	return fmt.Sprintf("number tree %s Kids[%d]", name, i)
}

func validateNumberTreeDepth(xRefTable *model.XRefTable, name string, d types.Dict, ownerObjNr int, root, useIDs bool, depth int) (firstKey, lastKey int, err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	if err := xRefTable.CheckRecursionDepth(fmt.Sprintf("number tree %s", name), depth); err != nil {
		return 0, 0, err
	}

	// A node has "Kids" or "Nums" entry.

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if o, found := d.Find("Kids"); found {
		kidsObjNr := validationObjectNumber(ownerObjNr, o)

		a, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return 0, 0, fmt.Errorf("number tree %s Kids: dereference array: %w", name, err)
		}
		if a == nil {
			return 0, 0, fmt.Errorf("number tree %s: missing Kids array", name)
		}

		for i, o := range a {
			kidObjNr := validationObjectNumber(kidsObjNr, o)

			d1, err := xRefTable.DereferenceDict(o)
			if err != nil {
				err = fmt.Errorf("%s: dereference dict: %w", numberTreeKidContext(name, o, i), err)
				return 0, 0, model.WithValidationErrorObject(err, kidObjNr)
			}
			if d1 == nil {
				err = fmt.Errorf("%s: missing dict", numberTreeKidContext(name, o, i))
				return 0, 0, model.WithValidationErrorObject(err, kidObjNr)
			}

			var fk int
			fk, lastKey, err = validateNumberTreeDepth(xRefTable, name, d1, kidObjNr, false, useIDs, depth+1)
			if err != nil {
				return 0, 0, model.WrapRecursionError(numberTreeKidContext(name, o, i), err)
			}
			if firstKey == 0 {
				firstKey = fk
			}
		}

	} else {

		// Leaf node
		firstKey, lastKey, err = validateNumberTreeDictNumsEntry(xRefTable, d, ownerObjNr, name, useIDs)
		if err != nil {
			return 0, 0, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNumberTreeDictLimitsEntry(xRefTable, d, ownerObjNr, firstKey, lastKey)
		if err != nil {
			return 0, 0, fmt.Errorf("number tree %s Limits: %w", name, err)
		}

	}

	return firstKey, lastKey, nil
}

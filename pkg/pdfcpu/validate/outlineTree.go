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

func validateOutlineItemDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "outlineItemDict"

	// Title, required, text string
	_, err := validateStringEntry(xRefTable, d, dictName, "Title", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// fmt.Printf("Title: %s\n", *title)

	// Parent, required, dict indRef
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "Parent", REQUIRED, model.V10)
	if err != nil {
		return err
	}
	_, err = xRefTable.DereferenceDict(*ir)
	if err != nil {
		return err
	}

	// // Count, optional, int
	// _, err = validateIntegerEntry(xRefTable, d, dictName, "Count", OPTIONAL, model.V10, nil)
	// if err != nil {
	// 	return err
	// }

	// SE, optional, dict indRef, since V1.3
	ir, err = validateIndRefEntry(xRefTable, d, dictName, "SE", OPTIONAL, model.V13)
	if err != nil {
		return err
	}
	if ir != nil {
		_, err = xRefTable.DereferenceDict(*ir)
		if err != nil {
			return err
		}
	}

	// C, optional, array of 3 numbers, since V1.4
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, model.V14, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// F, optional integer, since V1.4
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V14, nil)
	if err != nil {
		return err
	}

	// Optional A or Dest, since V1.1
	return validateActionOrDestination(xRefTable, d, dictName, model.V11)
}

func handleOutlineItemDict(xRefTable *model.XRefTable, ir types.IndirectRef, objNumber int) (types.Dict, error) {
	d, err := xRefTable.DereferenceDict(ir)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errors.Errorf("validateOutlineTree: object #%d is nil.", objNumber)
	}

	if err = validateOutlineItemDict(xRefTable, d); err != nil {
		return nil, err
	}

	return d, nil
}

func leaf(firstChild, lastChild *types.IndirectRef, objNumber, validationMode int) (bool, error) {
	if firstChild == nil {
		if lastChild == nil {
			// Leaf
			return true, nil
		}
		if validationMode == model.ValidationStrict {
			return false, errors.Errorf("pdfcpu: validateOutlineTree: missing \"First\" at obj#%d", objNumber)
		}
	}
	if lastChild == nil && validationMode == model.ValidationStrict {
		return false, errors.Errorf("pdfcpu: validateOutlineTree: missing \"Last\" at obj#%d", objNumber)
	}
	if firstChild != nil && firstChild.ObjectNumber.Value() == objNumber &&
		lastChild != nil && lastChild.ObjectNumber.Value() == objNumber {
		// Degenerated leaf = node pointing to itself.
		if validationMode == model.ValidationStrict {
			return false, errors.Errorf("pdfcpu: validateOutlineTree: corrupted at obj#%d", objNumber)
		}
		return true, nil
	}
	return false, nil
}

func evalOutlineCount(c, visc int, count, total, visible *int) error {

	if visc == 0 {
		if count == nil || *count == 0 {
			return errors.New("pdfcpu: validateOutlineTree: non-empty outline item dict needs \"Count\" <> 0")
		}
		if *count != c && *count != -c {
			return errors.Errorf("pdfcpu: validateOutlineTree: non-empty outline item dict got \"Count\" %d, want %d or %d", *count, c, -c)
		}
		if *count == c {
			*total += c
		}
	}

	if visc > 0 {
		if count == nil || *count != c+visc {
			return errors.Errorf("pdfcpu: validateOutlineTree: non-empty outline item dict got \"Count\" %d, want %d", *count, c+visc)
		}
		*total += c
		*visible += visc
	}

	return nil
}

func validateOutlineTree(xRefTable *model.XRefTable, first, last *types.IndirectRef) (int, int, error) {

	var (
		d         types.Dict
		objNumber int
		total     int
		visible   int
		err       error
	)

	m := map[int]bool{}

	// Process linked list of outline items.
	for ir := first; ir != nil; ir = d.IndirectRefEntry("Next") {

		objNumber = ir.ObjectNumber.Value()

		if m[objNumber] {
			return 0, 0, errors.New("pdfcpu: validateOutlineTree: circular outline items")
		}
		m[objNumber] = true

		total++

		d, err = handleOutlineItemDict(xRefTable, *ir, objNumber)
		if err != nil {
			return 0, 0, err
		}

		count := d.IntEntry("Count")

		firstChild := d.IndirectRefEntry("First")
		lastChild := d.IndirectRefEntry("Last")

		ok, err := leaf(firstChild, lastChild, objNumber, xRefTable.ValidationMode)
		if err != nil {
			return 0, 0, err
		}
		if ok {
			if count != nil && *count != 0 {
				return 0, 0, errors.New("pdfcpu: validateOutlineTree: empty outline item dict \"Count\" must be 0")
			}
			continue
		}

		c, visc, err := validateOutlineTree(xRefTable, firstChild, lastChild)
		if err != nil {
			return 0, 0, err
		}

		if err := evalOutlineCount(c, visc, count, &total, &visible); err != nil {
			return 0, 0, err
		}

	}

	if xRefTable.ValidationMode == model.ValidationStrict && objNumber != last.ObjectNumber.Value() {
		return 0, 0, errors.Errorf("pdfcpu: validateOutlineTree: corrupted child list %d <> %d\n", objNumber, last.ObjectNumber)
	}

	return total, visible, nil
}

func validateVisibleOutlineCount(xRefTable *model.XRefTable, total, visible int, count *int) error {

	if count == nil {
		return errors.Errorf("pdfcpu: validateOutlines: corrupted, root \"Count\" is nil, expected to be %d", total+visible)
	}
	if xRefTable.ValidationMode == model.ValidationStrict && *count != total+visible {
		return errors.Errorf("pdfcpu: validateOutlines: corrupted, root \"Count\" = %d, expected to be %d", *count, total+visible)
	}
	if xRefTable.ValidationMode == model.ValidationRelaxed && *count != total+visible && *count != -total-visible {
		return errors.Errorf("pdfcpu: validateOutlines: corrupted, root \"Count\" = %d, expected to be %d", *count, total+visible)
	}

	return nil
}

func validateInvisibleOutlineCount(xRefTable *model.XRefTable, total, visible int, count *int) error {

	if count != nil {
		if xRefTable.ValidationMode == model.ValidationStrict && *count == 0 {
			return errors.New("pdfcpu: validateOutlines: corrupted, root \"Count\" shall be omitted if there are no open outline items")
		}
		if xRefTable.ValidationMode == model.ValidationStrict && *count != total && *count != -total {
			return errors.Errorf("pdfcpu: validateOutlines: corrupted, root \"Count\" = %d, expected to be %d", *count, total)
		}
	}

	return nil
}

func validateOutlineCount(xRefTable *model.XRefTable, total, visible int, count *int) error {

	if visible == 0 {
		return validateInvisibleOutlineCount(xRefTable, total, visible, count)
	}

	if visible > 0 {
		return validateVisibleOutlineCount(xRefTable, total, visible, count)
	}

	return nil
}

func validateOutlines(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {

	// => 12.3.3 Document Outline

	ir, err := validateIndRefEntry(xRefTable, rootDict, "rootDict", "Outlines", required, sinceVersion)
	if err != nil || ir == nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil || d == nil {
		return err
	}

	xRefTable.Outlines = d

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, "outlineDict", "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Outlines" || s == "Outline" })
	if err != nil {
		return err
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return errors.New("pdfcpu: validateOutlines: corrupted, root missing \"First\"")
		}
		// empty outlines
		return nil
	}
	if last == nil {
		return errors.New("pdfcpu: validateOutlines: corrupted, root missing \"Last\"")
	}

	count := d.IntEntry("Count")
	if xRefTable.ValidationMode == model.ValidationStrict && count != nil && *count < 0 {
		return errors.New("pdfcpu: validateOutlines: corrupted, root \"Count\" can't be negativ")
	}

	total, visible, err := validateOutlineTree(xRefTable, first, last)
	if err != nil {
		return err
	}

	if err := validateOutlineCount(xRefTable, total, visible, count); err != nil {
		return err
	}

	return nil
}

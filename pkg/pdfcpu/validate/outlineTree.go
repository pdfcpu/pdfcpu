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

var ErrBookmarksRepair = errors.New("pdfcpu: bookmarks repair failed")

func validateOutlineItemDictTitle(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	_, err := validateStringEntry(xRefTable, d, dictName, "Title", REQUIRED, model.V10, nil)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return err
		}
		if _, err := validateNameEntry(xRefTable, d, dictName, "Title", REQUIRED, model.V10, nil); err != nil {
			return err
		}
	}
	return nil
}

func validateOutlineItemDictParent(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "Parent", required, model.V10)
	if err != nil {
		return err
	}
	if ir != nil {
		if _, err = xRefTable.DereferenceDict(*ir); err != nil {
			return err
		}
	}
	return nil
}

func validateOutlineItemDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "outlineItemDict"

	// Title, required, text string
	if err := validateOutlineItemDictTitle(xRefTable, d, dictName); err != nil {
		return err
	}

	// Parent, required, dict indRef
	if err := validateOutlineItemDictParent(xRefTable, d, dictName); err != nil {
		return err
	}

	// SE, optional, dict indRef, since V1.3
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "SE", OPTIONAL, model.V13)
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
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	if _, err = validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 }); err != nil {
		return err
	}

	// F, optional integer, since V1.4
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	if _, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, sinceVersion, nil); err != nil {
		return err
	}

	// Optional A or Dest, since V1.1
	destName, err := validateActionOrDestination(xRefTable, d, dictName, model.V11)
	if err != nil {
		return err
	}
	if destName != "" {
		if _, err = xRefTable.DereferenceDestArray(destName); err != nil && xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowDigestedSpecViolation("outlineDict with unresolved destination")
			return nil
		}
	}

	return err
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
			return false, errors.Errorf("pdfcpu: validateOutlineTree: invalid at obj#%d", objNumber)
		}
		return true, nil
	}
	return false, nil
}

func evalOutlineCount(xRefTable *model.XRefTable, c, visc int, count int, total, visible *int) error {
	if visc == 0 {
		if count == 0 {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return errors.New("pdfcpu: validateOutlineTree: non-empty outline item dict needs \"Count\" <> 0")
			}
			count = c
		}
		if count != c && count != -c {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return errors.Errorf("pdfcpu: validateOutlineTree: non-empty outline item dict got \"Count\" %d, want %d or %d", count, c, -c)
			}
			count = c
		}
		if count == c {
			*total += c
		}
	}

	if visc > 0 {
		if count != c+visc {
			return errors.Errorf("pdfcpu: validateOutlineTree: non-empty outline item dict got \"Count\" %d, want %d", count, c+visc)
		}
		*total += c
		*visible += visc
	}

	return nil
}

func validateOutlineTree(xRefTable *model.XRefTable, first, last *types.IndirectRef, m map[int]bool, fixed *bool) (int, int, error) {
	var (
		d       types.Dict
		objNr   int
		total   int
		visible int
		err     error
	)

	// Process linked list of outline items.
	for ir := first; ir != nil; ir = d.IndirectRefEntry("Next") {
		objNr = ir.ObjectNumber.Value()
		total++

		d, err = handleOutlineItemDict(xRefTable, *ir, objNr)
		if err != nil {
			return 0, 0, err
		}

		var count int
		if c := d.IntEntry("Count"); c != nil {
			count = *c
		}

		firstChild := d.IndirectRefEntry("First")
		lastChild := d.IndirectRefEntry("Last")

		ok, err := leaf(firstChild, lastChild, objNr, xRefTable.ValidationMode)
		if err != nil {
			return 0, 0, err
		}
		if ok {
			if count != 0 {
				if xRefTable.ValidationMode == model.ValidationStrict {
					return 0, 0, errors.New("pdfcpu: validateOutlineTree: empty outline item dict \"Count\" must be 0")
				}
			}
			continue
		}

		if err := scanAndFixOutlineItems(xRefTable, firstChild, lastChild, m, fixed); err != nil {
			return 0, 0, err
		}

		c, visc, err := validateOutlineTree(xRefTable, firstChild, lastChild, m, fixed)
		if err != nil {
			return 0, 0, err
		}

		if err := evalOutlineCount(xRefTable, c, visc, count, &total, &visible); err != nil {
			return 0, 0, err
		}

	}

	if xRefTable.ValidationMode == model.ValidationStrict && objNr != last.ObjectNumber.Value() {
		return 0, 0, errors.Errorf("pdfcpu: validateOutlineTree: invalid child list %d <> %d\n", objNr, last.ObjectNumber)
	}

	return total, visible, nil
}

func validateVisibleOutlineCount(xRefTable *model.XRefTable, total, visible int, count *int) error {
	if count == nil {
		return errors.Errorf("pdfcpu: validateOutlines: invalid, root \"Count\" is nil, expected to be %d", total+visible)
	}
	if xRefTable.ValidationMode == model.ValidationStrict && *count != total+visible {
		return errors.Errorf("pdfcpu: validateOutlines: invalid, root \"Count\" = %d, expected to be %d", *count, total+visible)
	}
	if xRefTable.ValidationMode == model.ValidationRelaxed && *count != total+visible && *count != -total-visible {
		return errors.Errorf("pdfcpu: validateOutlines: invalid, root \"Count\" = %d, expected to be %d", *count, total+visible)
	}

	return nil
}

func validateInvisibleOutlineCount(xRefTable *model.XRefTable, total int, count *int) error {
	if count != nil {
		if xRefTable.ValidationMode == model.ValidationStrict && *count == 0 {
			return errors.New("pdfcpu: validateOutlines: invalid, root \"Count\" shall be omitted if there are no open outline items")
		}
		if xRefTable.ValidationMode == model.ValidationStrict && *count != total && *count != -total {
			return errors.Errorf("pdfcpu: validateOutlines: invalid, root \"Count\" = %d, expected to be %d", *count, total)
		}
	}

	return nil
}

func validateOutlineCount(xRefTable *model.XRefTable, total, visible int, count *int) error {
	if visible == 0 {
		return validateInvisibleOutlineCount(xRefTable, total, count)
	}

	if visible > 0 {
		return validateVisibleOutlineCount(xRefTable, total, visible, count)
	}

	return nil
}

func firstOfRemainder(xRefTable *model.XRefTable, last *types.IndirectRef, duplObjNr, oneBeforeDuplObj int) (int, types.Dict, error) {
	// Starting with the last node, go back until we hit duplObjNr or oneBeforeDuplObj
	for ir := last; ir != nil; {
		objNr := ir.ObjectNumber.Value()
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return 0, nil, err
		}
		if len(d) == 0 {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return 0, nil, errors.New("pdfcpu: validateOutlines: corrupt outline items detected")
			}
		}
		irPrev := d.IndirectRefEntry("Prev")
		if irPrev == nil {
			break
		}
		prevObjNr := irPrev.ObjectNumber.Value()
		if prevObjNr == duplObjNr {
			d["Prev"] = *types.NewIndirectRef(oneBeforeDuplObj, 0)
			return objNr, d, nil
		}
		if prevObjNr == oneBeforeDuplObj {
			return objNr, d, nil
		}
		ir = irPrev
	}

	return 0, nil, nil
}

func removeDuplFirst(xRefTable *model.XRefTable, first, last *types.IndirectRef, duplObjNr, oneBeforeDuplObj int) error {
	nextObjNr, nextDict, err := firstOfRemainder(xRefTable, last, duplObjNr, oneBeforeDuplObj)
	if err != nil {
		return err
	}
	if nextObjNr == 0 {
		return ErrBookmarksRepair
	}
	delete(nextDict, "Prev")
	first.ObjectNumber = types.Integer(oneBeforeDuplObj)
	return nil
}

func handleCircular(xRefTable *model.XRefTable, dict types.Dict, first *types.IndirectRef, fixed *bool) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return errors.New("pdfcpu: validateOutlines: circular outline items detected")
	}
	dict["Prev"] = *first
	delete(dict, "Next")
	*fixed = true
	return nil
}

func handleCorruptDict(xRefTable *model.XRefTable) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return errors.New("pdfcpu: validateOutlines: corrupt outline items detected")
	}
	return ErrBookmarksRepair
}

func handleDuplicate(
	xRefTable *model.XRefTable,
	ir, first, last *types.IndirectRef,
	prevDict types.Dict,
	objNr, prevObjNr int) error {

	if ir == first {
		return removeDuplFirst(xRefTable, first, last, objNr, prevObjNr)
	}

	if ir == last {
		delete(prevDict, "Next")
		last.ObjectNumber = types.Integer(prevObjNr)
		return nil
	}

	nextObjNr, _, _ := firstOfRemainder(xRefTable, last, objNr, prevObjNr)
	if nextObjNr == 0 {
		return ErrBookmarksRepair
	}

	nextRef := prevDict.IndirectRefEntry("Next")
	if nextRef == nil {
		return ErrBookmarksRepair
	}

	prevDict["Next"] = *types.NewIndirectRef(nextObjNr, 0)

	return nil
}

func scanAndFixOutlineItems(xRefTable *model.XRefTable, first, last *types.IndirectRef, seen map[int]bool, fixed *bool) error {
	visited := map[int]bool{}
	var prevDict types.Dict
	var prevObjNr int

	for ir := first; ir != nil; {
		objNr := ir.ObjectNumber.Value()

		if visited[objNr] {
			return handleCircular(xRefTable, prevDict, first, fixed)
		}
		visited[objNr] = true

		dict, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return err
		}
		if len(dict) == 0 {
			return handleCorruptDict(xRefTable)
		}

		if ir == first && dict["Prev"] != nil {
			*fixed = true
			if xRefTable.ValidationMode == model.ValidationStrict {
				return errors.New("pdfcpu: validateOutlines: corrupt outline items detected")
			}
			delete(dict, "Prev")
		}

		if seen[objNr] {
			*fixed = true
			return handleDuplicate(xRefTable, ir, first, last, prevDict, objNr, prevObjNr)
		}

		seen[objNr] = true
		prevDict = dict
		prevObjNr = objNr
		ir = dict.IndirectRefEntry("Next")
	}

	return nil
}

func removeOutlines(xRefTable *model.XRefTable, rootDict types.Dict) {
	xRefTable.Outlines = nil
	delete(rootDict, "Outlines")
}

func validateOutlinesGeneral(xRefTable *model.XRefTable, rootDict types.Dict) (*types.IndirectRef, *types.IndirectRef, *int, error) {
	d := xRefTable.Outlines

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, "outlineDict", "Type", OPTIONAL, model.V10, func(s string) bool {
		return s == "Outlines" || (xRefTable.ValidationMode == model.ValidationRelaxed && (s == "Outline" || s == "BMoutlines"))
	})
	if err != nil {
		return nil, nil, nil, err
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return nil, nil, nil, errors.New("pdfcpu: validateOutlines: invalid, root missing \"First\"")
		}
		removeOutlines(xRefTable, rootDict)
		return nil, nil, nil, nil
	}
	if last == nil && xRefTable.ValidationMode == model.ValidationStrict {
		return nil, nil, nil, errors.New("pdfcpu: validateOutlines: invalid, root missing \"Last\"")
	}

	count := d.IntEntry("Count")
	if xRefTable.ValidationMode == model.ValidationStrict && count != nil && *count < 0 {
		return nil, nil, nil, errors.New("pdfcpu: validateOutlines: invalid, root \"Count\" can't be negative")
	}

	return first, last, count, nil
}

func handleCorruptOutlineItems(xRefTable *model.XRefTable, rootDict types.Dict) {
	model.ShowMsg("validateOutlines: corrupt outline items detected")
	removeOutlines(xRefTable, rootDict)
	model.ShowSkipped("bookmarks")
}

func scanAndFixOutlines(xRefTable *model.XRefTable, rootDict types.Dict, first, last *types.IndirectRef, count *int) error {

	m := map[int]bool{}
	var fixed bool

	err := scanAndFixOutlineItems(xRefTable, first, last, m, &fixed)
	if err != nil {
		if err == ErrBookmarksRepair && xRefTable.ValidationMode == model.ValidationRelaxed {
			handleCorruptOutlineItems(xRefTable, rootDict)
			return nil
		}
		return err
	}

	total, visible, err := validateOutlineTree(xRefTable, first, last, m, &fixed)
	if err != nil {
		if err == ErrBookmarksRepair && xRefTable.ValidationMode == model.ValidationRelaxed {
			handleCorruptOutlineItems(xRefTable, rootDict)
			return nil
		}
		return err
	}

	if err := validateOutlineCount(xRefTable, total, visible, count); err != nil {
		return err
	}

	if fixed {
		model.ShowRepaired("bookmarks")
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
	if err != nil {
		return err
	}

	if d == nil {
		removeOutlines(xRefTable, rootDict)
		return nil
	}

	xRefTable.Outlines = d

	first, last, count, err := validateOutlinesGeneral(xRefTable, rootDict)
	if err != nil {
		return err
	}
	if first == nil && last == nil {
		return nil
	}

	return scanAndFixOutlines(xRefTable, rootDict, first, last, count)
}

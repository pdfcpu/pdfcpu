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

func validateOutlineItemDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "outlineItemDict"

	// Title, required, text string
	_, err := validateStringEntry(xRefTable, d, dictName, "Title", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// fmt.Printf("Title: %s\n", *title)

	// Parent, required, dict indRef
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "Parent", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}
	_, err = xRefTable.DereferenceDict(*ir)
	if err != nil {
		return err
	}

	// Count, optional, int
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Count", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// SE, optional, dict indRef, since V1.3
	ir, err = validateIndRefEntry(xRefTable, d, dictName, "SE", OPTIONAL, pdf.V13)
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
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, pdf.V14, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// F, optional integer, since V1.4
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V14, nil)
	if err != nil {
		return err
	}

	// Optional A or Dest, since V1.1
	return validateActionOrDestination(xRefTable, d, dictName, pdf.V11)
}

func validateOutlineTree(xRefTable *pdf.XRefTable, first, last *pdf.IndirectRef) error {

	var (
		d         pdf.Dict
		objNumber int
		err       error
	)

	// Process linked list of outline items.
	for ir := first; ir != nil; ir = d.IndirectRefEntry("Next") {

		objNumber = ir.ObjectNumber.Value()

		// outline item dict
		d, err = xRefTable.DereferenceDict(*ir)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("validateOutlineTree: object #%d is nil.", objNumber)
		}

		err = validateOutlineItemDict(xRefTable, d)
		if err != nil {
			return err
		}

		firstChild := d.IndirectRefEntry("First")
		lastChild := d.IndirectRefEntry("Last")

		if firstChild == nil && lastChild == nil {
			// Leaf
			continue
		}

		if firstChild != nil && (xRefTable.ValidationMode == pdf.ValidationRelaxed ||
			xRefTable.ValidationMode == pdf.ValidationStrict && lastChild != nil) {
			// Recurse into subtree.
			err = validateOutlineTree(xRefTable, firstChild, lastChild)
			if err != nil {
				return err
			}
			continue
		}

		return errors.New("pdfcpu: validateOutlineTree: corrupted, needs both first and last or neither for a leaf")

	}

	if xRefTable.ValidationMode == pdf.ValidationStrict && objNumber != last.ObjectNumber.Value() {
		return errors.Errorf("pdfcpu: validateOutlineTree: corrupted child list %d <> %d\n", objNumber, last.ObjectNumber)
	}

	return nil
}

func validateOutlines(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.3.3 Document Outline

	ir, err := validateIndRefEntry(xRefTable, rootDict, "rootDict", "Outlines", OPTIONAL, sinceVersion)
	if err != nil || ir == nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil || d == nil {
		return err
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, "outlineDict", "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "Outlines" || s == "Outline" })
	if err != nil {
		return err
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return errors.New("pdfcpu: validateOutlines: corrupted, root needs both first and last")
		}
		// leaf
		return nil
	}

	if xRefTable.ValidationMode == pdf.ValidationStrict && last == nil {
		return errors.New("pdfcpu: validateOutlines: corrupted, root needs both first and last")
	}

	return validateOutlineTree(xRefTable, first, last)
}

package pdfcpu

import (
	"github.com/pkg/errors"
)

func validateOutlineItemDict(xRefTable *XRefTable, dict *PDFDict) error {

	dictName := "outlineItemDict"

	var indRef *PDFIndirectRef

	// Title, required, text string
	_, err := validateStringEntry(xRefTable, dict, dictName, "Title", REQUIRED, V10, nil)
	if err != nil {
		return err
	}

	// Parent, required, dict indRef
	indRef, err = validateIndRefEntry(xRefTable, dict, dictName, "Parent", REQUIRED, V10)
	if err != nil {
		return err
	}
	_, err = xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return err
	}

	// Count, optional, int
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Count", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// SE, optional, dict indRef, since V1.3
	indRef, err = validateIndRefEntry(xRefTable, dict, dictName, "SE", OPTIONAL, V13)
	if err != nil {
		return err
	}
	if indRef != nil {
		_, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}
	}

	// C, optional, array of 3 numbers, since V1.4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, V14, func(a PDFArray) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// F, optional integer, since V1.4
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", OPTIONAL, V14, nil)
	if err != nil {
		return err
	}

	// Optional A or Dest, since V1.1
	return validateActionOrDestination(xRefTable, dict, dictName, V11)
}

func validateOutlineTree(xRefTable *XRefTable, first, last *PDFIndirectRef) error {

	var (
		dict      *PDFDict
		objNumber int
		err       error
	)

	// Process linked list of outline items.
	for indRef := first; indRef != nil; indRef = dict.IndirectRefEntry("Next") {

		objNumber = indRef.ObjectNumber.Value()

		// outline item dict
		dict, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}
		if dict == nil {
			return errors.Errorf("validateOutlineTree: object #%d is nil.", objNumber)
		}

		err = validateOutlineItemDict(xRefTable, dict)
		if err != nil {
			return err
		}

		firstChild := dict.IndirectRefEntry("First")
		lastChild := dict.IndirectRefEntry("Last")

		if firstChild == nil && lastChild == nil {
			// Leaf
			continue
		}

		if firstChild != nil && lastChild != nil {
			// Recurse into subtree.
			err = validateOutlineTree(xRefTable, firstChild, lastChild)
			if err != nil {
				return err
			}
			continue
		}

		return errors.New("validateOutlineTree: corrupted, needs both first and last or neither for a leaf")

	}

	// Relaxed validation
	if objNumber != last.ObjectNumber.Value() && xRefTable.ValidationMode == ValidationStrict {
		return errors.Errorf("validateOutlineTree: corrupted child list %d <> %d\n", objNumber, last.ObjectNumber)
	}

	return nil
}

func validateOutlines(xRefTable *XRefTable, rootDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// => 12.3.3 Document Outline

	indRef, err := validateIndRefEntry(xRefTable, rootDict, "rootDict", "Outlines", OPTIONAL, sinceVersion)
	if err != nil || indRef == nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return err
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, "outlineDict", "Type", OPTIONAL, V10, func(s string) bool { return s == "Outlines" })
	if err != nil {
		return err
	}

	first := dict.IndirectRefEntry("First")
	last := dict.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return errors.New("validateOutlines: corrupted, root needs both first and last")
		}
		// leaf
		return nil
	}

	if last == nil {
		return errors.New("validateOutlines: corrupted, root needs both first and last")
	}

	return validateOutlineTree(xRefTable, first, last)
}

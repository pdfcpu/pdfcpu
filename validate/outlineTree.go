package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateOutlineItemDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("validateOutlineItemDict begin")

	dictName := "outlineItemDict"

	var indRef *types.PDFIndirectRef

	// Title, required, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Title", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Parent, required, dict indRef
	indRef, err = validateIndRefEntry(xRefTable, dict, dictName, "Parent", REQUIRED, types.V10)
	if err != nil {
		return
	}
	_, err = xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return
	}

	// Count, optional, int
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Count", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// SE, optional, dict indRef, since V1.3
	indRef, err = validateIndRefEntry(xRefTable, dict, dictName, "SE", OPTIONAL, types.V13)
	if err != nil {
		return
	}
	if indRef != nil {
		_, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return
		}
	}

	// C, optional, array of 3 numbers, since V1.4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V14, func(a types.PDFArray) bool { return len(a) == 3 })
	if err != nil {
		return
	}

	// F, optional integer, since V1.4
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	var d *types.PDFDict

	// A, optional if no Dest entry present, dict, since V1.1
	d, err = validateDictEntry(xRefTable, dict, dictName, "A", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateActionDict(xRefTable, *d)
		if err != nil {
			return
		}
		logInfoValidate.Printf("validateOutlineItemDict end")
		return
	}

	// Dest, optional if no A entry present, name, byte string, array
	if dest, found := dict.Find("Dest"); found {
		err = validateDestination(xRefTable, dest)
		if err != nil {
			return
		}
	}

	logInfoValidate.Printf("validateOutlineItemDict end")

	return
}

func validateOutlineTree(xRefTable *types.XRefTable, first, last *types.PDFIndirectRef) (err error) {

	logInfoValidate.Printf("*** validateOutlineTree(%d,%d) begin ***\n", first.ObjectNumber, last.ObjectNumber)

	var (
		dict      *types.PDFDict
		objNumber int
	)

	// Process linked list of outline items.
	for indRef := first; indRef != nil; indRef = dict.IndirectRefEntry("Next") {

		objNumber = indRef.ObjectNumber.Value()

		// outline item dict
		dict, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return
		}
		if dict == nil {
			return errors.Errorf("validateOutlineTree: object #%d is nil.", objNumber)
		}

		logInfoValidate.Printf("validateOutlineTree: Next object #%d\n", objNumber)

		err = validateOutlineItemDict(xRefTable, dict)
		if err != nil {
			return
		}

		firstChild := dict.IndirectRefEntry("First")
		lastChild := dict.IndirectRefEntry("Last")

		if firstChild == nil && lastChild == nil {
			// leaf
			continue
		}

		if firstChild != nil && lastChild != nil {
			// subtree, recurse.
			err = validateOutlineTree(xRefTable, firstChild, lastChild)
			if err != nil {
				return
			}
			continue
		}

		return errors.New("validateOutlineTree: corrupted, needs both first and last or neither for a leaf")

	}

	// Relaxed validation
	if objNumber != last.ObjectNumber.Value() && xRefTable.ValidationMode == types.ValidationStrict {
		logErrorValidate.Printf("validateOutlineTree: corrupted child list %d <> %d\n", objNumber, last.ObjectNumber)
	}

	logInfoValidate.Printf("*** validateOutlineTree(%d,%d) end ***\n", first.ObjectNumber, last.ObjectNumber)

	return
}

func validateOutlines(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.3 Document Outline

	logInfoValidate.Println("*** validateOutlines begin ***")

	var (
		indRef *types.PDFIndirectRef
		dict   *types.PDFDict
	)

	indRef, err = validateIndRefEntry(xRefTable, rootDict, "rootDict", "Outlines", OPTIONAL, sinceVersion)
	if err != nil || indRef == nil {
		return
	}

	dict, err = xRefTable.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, "outlineDict", "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Outlines" })
	if err != nil {
		return
	}

	first := dict.IndirectRefEntry("First")
	last := dict.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return errors.New("validateOutlines: corrupted, root needs both first and last")
		}
		// leaf
		logInfoValidate.Printf("validateOutlines end: obj#%d\n", indRef.ObjectNumber)
		return
	}

	if last == nil {
		return errors.New("validateOutlines: corrupted, root needs both first and last")
	}

	err = validateOutlineTree(xRefTable, first, last)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateOutlines end: obj#%d ***", indRef.ObjectNumber)

	return
}

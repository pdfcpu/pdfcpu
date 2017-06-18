package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

// TODO implement, parentCheck, BUG
func validateOutlineTree(xRefTable *types.XRefTable, first *types.PDFIndirectRef, last *types.PDFIndirectRef) (err error) {

	logInfoValidate.Printf("*** validateOutlineTree(%d,%d) begin ***\n", first.ObjectNumber, last.ObjectNumber)

	var dict *types.PDFDict
	var objNumber int

	for indRef := first; indRef != nil; indRef = dict.IndirectRefEntry("Next") {

		objNumber = indRef.ObjectNumber.Value()

		dict, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return
		}
		if dict == nil {
			return errors.Errorf("validateOutlineTree: object #%d is nil.", objNumber)
		}

		logInfoValidate.Printf("validateOutlineTree: Next object #%d\n", objNumber)

		// optional Dest or A entry
		// Dest: name, byte string, or array
		//       The destination that shall be displayed when this item is activated.

		// A: dictionary
		//    The action that shall be performed when this item is activated.

		if obj, found := dict.Find("A"); found {
			if _, found = dict.Find("Dest"); found {
				return errors.New("validateOutlineTree: only Dest or A allowed")
			}
			err = validateActionDict(xRefTable, obj)
			if err != nil {
				return
			}
		}

		if obj, found := dict.Find("Dest"); found {
			if _, found = dict.Find("A"); found {
				return errors.New("validateOutlineTree: only Dest or A allowed")
			}
			// TODO BUG
			err = validateDestination(xRefTable, obj)
			if err != nil {
				return
			}
		}

		// Title: required, PdfStringliteral or PDFHexLiteral.
		_, err = validateStringEntry(xRefTable, dict, "outlineItemDict", "Title", REQUIRED, types.V10, nil)
		if err != nil {
			return
		}

		/*if indRef := dict.IndirectRefEntry("Title"); indRef != nil {

			oNumber := int(indRef.ObjectNumber)
			gNumber := int(indRef.GenerationNumber)

			if dest.HasWriteOffset(oNumber) {
				logInfoWriter.Printf("*** writeOutlineTree: title object #%d already written. ***\n", oNumber)
			} else {
				if str, ok := xRefTable.Dereference(*indRef).(PDFStringLiteral); ok {
					logInfoWriter.Printf("writeOutlineTree: object #%d gets writeoffset: %d\n", oNumber, dest.Offset)
					writePDFStringLiteralObject(dest, oNumber, gNumber, str)
					logDebugWriter.Printf("writeOutlineTree: new offset after str = %d\n", dest.Offset)
				} else if hexstr, ok := xRefTable.Dereference(*indRef).(PDFHexLiteral); ok {
					logInfoWriter.Printf("writeOutlineTree: object #%d gets writeoffset: %d\n", oNumber, dest.Offset)
					writePDFHexLiteralObject(dest, oNumber, gNumber, hexstr)
					logDebugWriter.Printf("writeOutlineTree: new offset after hexstr = %d\n", dest.Offset)
				} else {
					log.Fatalf("writeOutlineTree: indRef for Title must be PDFStringLiteral or PDFHexLiteral.")
				}
			}

		}
		*/

		// TODO SE, optional, since 1.3 indRef, structure element

		// TODO C, optional, since 1.4, array of three numbers in the range 0.0 to 1.0

		// F, optional, since 1.4, integer
		_, err = validateIntegerEntry(xRefTable, dict, "outlineItemDict", "F", OPTIONAL, types.V14, nil)
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

	if objNumber != last.ObjectNumber.Value() && xRefTable.ValidationMode == types.ValidationStrict {
		logErrorValidate.Printf("validateOutlineTree: corrupted child list %d <> %d\n", objNumber, last.ObjectNumber)
	}

	logInfoValidate.Printf("*** validateOutlineTree(%d,%d) end ***\n", first.ObjectNumber, last.ObjectNumber)

	return
}

func validateOutlines(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.3 Document Outline

	logInfoValidate.Println("*** validateOutlines begin ***")

	indRef := rootDict.IndirectRefEntry("Outlines")
	if indRef == nil {
		if required {
			err = errors.Errorf("validateOutlines: required entry \"Outlines\" missing")
			return
		}
		logInfoValidate.Println("validateOutlines end: object is nil.")
		return
	}

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateOutlines end: object is nil.")
		return
	}

	// dict, ok := obj.(types.PDFDict)
	// if !ok {
	// 	return errors.Errorf("validateOutlines: corrupt outlines dict obj#%d\n", indRef.ObjectNumber)
	// 	// Ignore and continue if file has invalid Outline
	// 	//logErrorWriter.Println("writeOutlines: ignoring corrupt Outlines dict.")
	// 	//return
	// }

	// optional Type entry must be Outline

	first := dict.IndirectRefEntry("First")
	last := dict.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return errors.New("validateOutlines: corrupted, root needs both first and last")
		}
		// leaf
		//log.Fatalln("writeOutlines: root = leaf, doesn't make much sense. ")
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

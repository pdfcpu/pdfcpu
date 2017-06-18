package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

// TODO implement, parentCheck, BUG
func writeOutlineTree(ctx *types.PDFContext, first *types.PDFIndirectRef, last *types.PDFIndirectRef) (err error) {

	logInfoWriter.Printf("*** writeOutlineTree(%d,%d) begin ***\n", first.ObjectNumber, last.ObjectNumber)

	var dict *types.PDFDict
	var objNumber int

	for indRef := first; indRef != nil; indRef = dict.IndirectRefEntry("Next") {

		objNumber = int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("writeOutlineTree: object #%d already written\n", objNumber)
			continue
		}

		dict, err = ctx.DereferenceDict(*indRef)
		if err != nil {
			return
		}
		if dict == nil {
			return errors.Errorf("writeOutlineTree: object #%d is nil.", objNumber)
		}

		logInfoWriter.Printf("writeOutlineTree: Next object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

		err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
		if err != nil {
			return err
		}

		logDebugWriter.Printf("writeOutlineTree: new offset after dict = %d\n", ctx.Write.Offset)

		// optional Dest or A entry
		// Dest: name, byte string, or array
		//       The destination that shall be displayed when this item is activated.

		// A: dictionary
		//    The action that shall be performed when this item is activated.

		if pdfObject, found := dict.Find("A"); found {
			if _, found = dict.Find("Dest"); found {
				return errors.New("writeOutlineTree: only Dest or A allowed")
			}
			err = writeActionDict(ctx, pdfObject)
			if err != nil {
				return
			}
		}

		if pdfObject, found := dict.Find("Dest"); found {
			if _, found = dict.Find("A"); found {
				return errors.New("writeOutlineTree: only Dest or A allowed")
			}
			// TODO BUG
			err = writeDestination(ctx, pdfObject)
			if err != nil {
				return
			}
		}

		// Title: required, PdfStringliteral or PDFHexLiteral.
		_, _, err = writeStringEntry(ctx, *dict, "outlineItemDict", "Title", REQUIRED, types.V10, nil)
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
		_, _, err = writeIntegerEntry(ctx, *dict, "outlineItemDict", "F", OPTIONAL, types.V14, nil)
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
			err = writeOutlineTree(ctx, firstChild, lastChild)
			if err != nil {
				return
			}
			continue
		}

		return errors.New("writeOutlineTree: corrupted, needs both first and last or neither for a leaf")

	}

	if objNumber != int(last.ObjectNumber) {
		logErrorWriter.Printf("writeOutlineTree: corrupted child list %d <> %d\n", objNumber, last.ObjectNumber)
	}

	logInfoWriter.Printf("*** writeOutlineTree(%d,%d) end ***\n", first.ObjectNumber, last.ObjectNumber)

	return
}

func writeOutlines(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.3 Document Outline

	logInfoWriter.Printf("*** writeOutlines begin: offset=%d ***\n", ctx.Write.Offset)

	indRef := rootDict.IndirectRefEntry("Outlines")
	if indRef == nil {
		if required {
			err = errors.Errorf("writeOutlines: required entry \"Outlines\" missing")
			return
		}
		logInfoWriter.Printf("writeOutlines end: object is nil.\n")
		return
	}

	obj, written, err := writeIndRef(ctx, *indRef)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeOutlines end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeOutlines end: object is nil.\n")
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.Errorf("writeOutlines: corrupt outlines dict obj#%d\n", indRef.ObjectNumber)
		// Ignore and continue if file has invalid Outline
		//logErrorWriter.Println("writeOutlines: ignoring corrupt Outlines dict.")
		//return
	}

	// optional Type entry must be Outline

	first := dict.IndirectRefEntry("First")
	last := dict.IndirectRefEntry("Last")

	if first == nil {
		if last != nil {
			return errors.New("writeOutlines: corrupted, root needs both first and last")
		}
		// leaf
		//log.Fatalln("writeOutlines: root = leaf, doesn't make much sense. ")
		logInfoWriter.Printf("writeOutlines end: obj#%d offset=%d\n", indRef.ObjectNumber, ctx.Write.Offset)
		return
	}

	if last == nil {
		return errors.New("writeOutlines: corrupted, root needs both first and last")
	}

	err = writeOutlineTree(ctx, first, last)
	if err != nil {
		return
	}

	ctx.Stats.AddRootAttr(types.RootOutlines)

	logInfoWriter.Printf("*** writeOutlines end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeBorderStyleDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeBorderStyleDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDict(ctx, obj)
	if err != nil {
		return err
	}

	if written {
		logInfoWriter.Printf("writeBorderStyleDict end: already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeBorderStyleDict end: is nil.\n")
		return
	}

	logDebugWriter.Printf("writeBorderStyleDict: new offset after dict = %d\n", ctx.Write.Offset)

	if dict.Type() != nil && *dict.Type() != "Border" {
		return errors.New("writeBorderStyleDict: corrupt entry \"Type\"")
	}

	// Dash array, optional
	if indRef := dict.IndirectRefEntry("D"); indRef != nil {
		return errors.New("writeBorderStyleDict: *** unsupported entry \"D\" ***")
	}

	// Border effect dict, optional
	if _, found := dict.Find("BE"); found {
		return errors.New("writeBorderStyleDict: *** unsupported entry \"BE\" ***")
	}

	logInfoWriter.Printf("*** writeBorderStyleDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDictLink(ctx *types.PDFContext, dict *types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictLink begin: offset=%d ***\n", ctx.Write.Offset)

	//xRefTable := dest.XRefTable

	// optional Dest or A entry

	// A: dictionary
	//    The action that shall be performed when this item is activated.
	if obj, found := dict.Find("A"); found {

		if _, found = dict.Find("Dest"); found {
			return errors.New("writeAnnotationDictLink: only Dest or A allowed")
		}

		err = writeActionDict(ctx, obj)
		if err != nil {
			return
		}

	}

	// Dest: name, byte string, or array
	//       The destination that shall be displayed when this item is activated.
	if obj, found := dict.Find("Dest"); found {

		if _, found = dict.Find("A"); found {
			return errors.New("writeAnnotationDictLink: only Dest or A allowed")
		}

		err = writeDestination(ctx, obj)
		if err != nil {
			return
		}

	}

	if _, found := dict.Find("PA"); found {
		return errors.New("writeAnnotationDictLink: unsupported entry \"PA\"")
	}

	if _, found := dict.Find("QuadPoints"); found {
		return errors.New("writeAnnotationDictLink: unsupported entry \"QuadPoints\"")
	}

	if obj, found := dict.Find("BS"); found {
		err = writeBorderStyleDict(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeAnnotationDictLink end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDictPopup(ctx *types.PDFContext, dict *types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictPopup begin: offset=%d ***\n", ctx.Write.Offset)

	//return errors.New("*** writeAnnotationDictPopUp not supported ***")

	// xRefTable := dest.XRefTable

	// Parent, dict
	// An indirect reference to the widget annotation’s parent field.
	// for terminal fields: parent field must already be written.
	// if indRef := dict.IndirectRefEntry("Parent"); indRef != nil {
	// 	objNumber := int(indRef.ObjectNumber)
	// 	if !ctx.Write.HasWriteOffset(objNumber) {
	// 		return errors.Errorf("*** writeAnnotationDictPopup: unknown parent field obj#:%d\n", objNumber)
	// 	}
	// }

	logInfoWriter.Printf("*** writeAnnotationDictPopup end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDictFreeTextAAPLAKExtras(ctx *types.PDFContext, indRef *types.PDFIndirectRef) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictFreeTextAAPL_AKExtras begin: offset=%d ***\n", ctx.Write.Offset)

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		return errors.Errorf("writeAnnotationDictFreeTextAAPL_AKExtras end: object #%d already written, offset=%d\n", objNumber, ctx.Write.Offset)
	}

	aaplDict, err := ctx.DereferenceDict(*indRef)
	if err != nil || aaplDict == nil {
		return errors.New("writeAnnotationDictFreeTextAAPL_AKExtras: corrupt AAPL:AKExtras dict")
	}

	logInfoWriter.Printf("writeAnnotationDictFreeTextAAPL_AKExtras: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	err = writePDFDictObject(ctx, objNumber, genNumber, *aaplDict)
	if err != nil {
		return
	}

	logDebugWriter.Printf("writeAnnotationDictFreeTextAAPL_AKExtras: new offset after AAPL:AKExtras dict = %d\n", ctx.Write.Offset)

	if indRef = aaplDict.IndirectRefEntry("AAPL:AKPDFAnnotationDictionary"); indRef == nil {
		return errors.New("writeAnnotationDictFreeTextAAPL_AKExtras: corrupt AAPL:AKExtras dict, missing entry \"AAPL:AKPDFAnnotationDictionary\"")
	}

	objNumber = int(indRef.ObjectNumber)
	genNumber = int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logInfoWriter.Printf("writeAnnotationDictFreeTextAAPL_AKExtras end: object #%d already written, offset=%d\n", objNumber, ctx.Write.Offset)
		return
	}

	dict, err := ctx.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return errors.New("writeAnnotationDictFreeTextAAPL_AKExtras: corrupt AAPL:AKPDFAnnotationDictionary dict")
	}

	logInfoWriter.Printf("writeAnnotationDictFreeTextAAPL_AKExtras: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
	if err != nil {
		return err
	}

	logDebugWriter.Printf("writeAnnotationDictFreeTextAAPL_AKExtras: new offset after AAPL:AKPDFAnnotationDictionary = %d\n", ctx.Write.Offset)

	err = writeAnnotationDict(ctx, *dict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeAnnotationDictFreeTextAAPL_AKExtras end: offset=%d ***\n", ctx.Write.Offset)

	return nil
}

func writeAnnotationDictFreeText(ctx *types.PDFContext, dict *types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictFreeText begin: offset=%d ***\n", ctx.Write.Offset)

	if pdfObject, found := dict.Find("BS"); found {
		err = writeBorderStyleDict(ctx, pdfObject)
		if err != nil {
			return
		}
	}

	if indRef := dict.IndirectRefEntry("AAPL:AKExtras"); indRef != nil {
		err = writeAnnotationDictFreeTextAAPLAKExtras(ctx, indRef)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeAnnotationDictFreeText end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDictStampAAPLAKExtras(ctx *types.PDFContext, indRef *types.PDFIndirectRef) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictStampAAPL_AKExtras begin: offset=%d ***\n", ctx.Write.Offset)

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logInfoWriter.Printf("writeAnnotationDictStampAAPL_AKExtras end: object #%d already written, offset=%d\n", objNumber, ctx.Write.Offset)
		return
	}

	aaplDict, err := ctx.DereferenceDict(*indRef)
	if err != nil || aaplDict == nil {
		return errors.New("writeAnnotationDictStampAAPL_AKExtras: corrupt AAPL:AKExtras dict")
	}

	logInfoWriter.Printf("writeAnnotationDictStampAAPL_AKExtras: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	err = writePDFDictObject(ctx, objNumber, genNumber, *aaplDict)
	if err != nil {
		return err
	}

	logDebugWriter.Printf("writeAnnotationDictStampAAPL_AKExtras: new offset after AAPL:AKExtras dict = %d\n", ctx.Write.Offset)

	if indRef = aaplDict.IndirectRefEntry("AAPL:AKPDFAnnotationDictionary"); indRef == nil {
		return errors.New("writeAnnotationDictStampAAPL_AKExtras: corrupt AAPL:AKExtras dict, missing entry \"AAPL:AKPDFAnnotationDictionary\"")
	}

	objNumber = int(indRef.ObjectNumber)
	genNumber = int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logInfoWriter.Printf("writeAnnotationDictStampAAPL_AKExtras end: object #%d already written, offset=%d\n", objNumber, ctx.Write.Offset)
		return
	}

	dict, err := ctx.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return errors.New("writeAnnotationDictStampAAPL_AKExtras: corrupt AAPL:AKPDFAnnotationDictionary dict")
	}

	logInfoWriter.Printf("writeAnnotationDictStampAAPL_AKExtras: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
	if err != nil {
		return err
	}

	logDebugWriter.Printf("writeAnnotationDictStampAAPL_AKExtras: new offset after AAPL:AKPDFAnnotationDictionary = %d\n", ctx.Write.Offset)

	err = writeAnnotationDict(ctx, *dict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeAnnotationDictStampAAPL_AKExtras end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDictStamp(ctx *types.PDFContext, dict *types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictStamp begin: offset=%d ***\n", ctx.Write.Offset)

	if indRef := dict.IndirectRefEntry("AAPL:AKExtras"); indRef != nil {
		err = writeAnnotationDictStampAAPLAKExtras(ctx, indRef)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeAnnotationDictStamp end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDictWidget(ctx *types.PDFContext, dict *types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDictWidget begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "widgetAnnotDict"

	// H, optional, name
	_, _, err = writeNameEntry(ctx, *dict, dictName, "H", OPTIONAL, types.V10, validate.AnnotationHighlightingMode)
	if err != nil {
		return
	}

	// MK, optional, dict
	// An appearance characteristics dictionary that shall be used in constructing
	// a dynamic appearance stream specifying the annotation‚Äôs visual presentation on the page.dict
	d, _, err := writeDictEntry(ctx, *dict, dictName, "MK", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		// TODO
		//err = validateAppearanceCharacteristicsDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// A, optional, dict, since V1.1
	// An action that shall be performed when the annotation is activated.
	d, _, err = writeDictEntry(ctx, *dict, dictName, "A", OPTIONAL, types.V11, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = writeActionDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// AA, optional, dict, since V1.2
	// An additional-actions dictionary defining the annotation’s behaviour in response to various trigger events.
	_, err = writeAdditionalActions(ctx, dict, "annotDict", "AA", OPTIONAL, types.V12, "fieldOrAnnot")
	if err != nil {
		return
	}

	// BS, optional, dict, since V1.2
	// A border style dictionary specifying the width and dash pattern
	// that shall be used in drawing the annotation‚Äôs border.
	d, _, err = writeDictEntry(ctx, *dict, dictName, "BC", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = writeBorderStyleDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// Parent, dict, required if one of multiple children in a field.
	// An indirect reference to the widget annotation‚Äôs parent field.
	// for terminal fields: parent field must already be written.
	_, _, err = writeIndRefEntry(ctx, *dict, dictName, "Parent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeAnnotationDictWidget begin: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAnnotationDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAnnotationDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "annotDict"
	var subtype *types.PDFName

	// Type, optional, name
	_, _, err = writeNameEntry(ctx, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Annot" })
	if err != nil {
		return
	}

	// Subtype, required, name
	subtype, _, err = writeNameEntry(ctx, dict, dictName, "Subtype", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// Rect, required, rectangle
	_, _, err = writeRectangleEntry(ctx, dict, dictName, "Rect", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Contents, optional, text string
	_, _, err = writeStringEntry(ctx, dict, dictName, "Contents", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P, optional, indRef of page dict
	var indRef *types.PDFIndirectRef
	indRef, _, err = writeIndRefEntry(ctx, dict, dictName, "P", OPTIONAL, types.V10)
	if err != nil {
		return
	}
	if indRef != nil {
		// check if this indRef points to a pageDict.
		var d *types.PDFDict
		d, err = ctx.DereferenceDict(*indRef)
		if err != nil {
			return
		}
		if d == nil {
			err = errors.Errorf("writeAnnotationDict: entry \"P\" (obj#%d) is nil", indRef.ObjectNumber)
		}
		_, _, err = writeNameEntry(ctx, *d, "pageDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Page" })
		if err != nil {
			return
		}

		if d == nil || d.Type() == nil || *d.Type() != "Page" {
			err = errors.Errorf("writeAnnotationDict: entry \"P\" (obj#%d) not a pageDict", indRef.ObjectNumber)
			return
		}
	}

	//  P, optional, corresponding page object
	//  All pages written at this point.
	if indRef := dict.IndirectRefEntry("P"); indRef != nil {
		objNumber := int(indRef.ObjectNumber)
		if !ctx.Write.HasWriteOffset(objNumber) {
			return errors.Errorf("writeAnnotationDict: Unknown page object#:%d\n", objNumber)
		}
	}

	// NM, optional, text string, since V1.4
	_, _, err = writeStringEntry(ctx, dict, dictName, "NM", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	// M, optional, date string in any format, since V1.1
	_, _, err = writeStringEntry(ctx, dict, dictName, "M", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// F, optional integer, since V1.1, annotation flags
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "F", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// AP, optional, appearance dict, since V1.2
	d, _, err := writeDictEntry(ctx, dict, dictName, "AP", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = writeAppearanceDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// AS, optional, name, since V1.2
	_, _, err = writeNameEntry(ctx, dict, dictName, "AS", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// Border, optional, array of numbers
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Border", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 3 || len(a) == 4 })
	if err != nil {
		return
	}

	// C, optional array, of numbers, since V1.1
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "C", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// StructParent, optional, integer, since V1.3
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// TODO
	// OC, optional, content group or optional content membership dictionary
	// specifying the optional content properties for the annotation, since V1.3
	//err = validateOptionalContent(xRefTable, dict, dictName, "OC", OPTIONAL, types.V13)
	//if err != nil {
	//	return
	//}

	if _, ok := dict.Find("OC"); ok {
		return errors.New("writeAnnotationDict: unsupported entry OC")
	}

	switch *subtype {

	case "Text":
		// passthough - entry Popup not part of standard.

	case "Link":
		err = writeAnnotationDictLink(ctx, &dict)

	case "FreeText":
		err = writeAnnotationDictFreeText(ctx, &dict)

	case "Line":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Square":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Circle":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Polygon":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "PolyLine":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Highlight":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Underline":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Squiggly":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "StrikeOut":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Stamp":
		err = writeAnnotationDictStamp(ctx, &dict)

	case "Caret":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Ink":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Popup":
		err = writeAnnotationDictPopup(ctx, &dict)

	case "FileAttachment":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Sound":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Movie":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Widget":
		err = writeAnnotationDictWidget(ctx, &dict)

	case "Screen":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "PrinterMark":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "TrapNet":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Watermark":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "3D":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Redact":
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	default:
		return errors.Errorf("writeAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	}

	if err == nil {
		logInfoWriter.Printf("*** writeAnnotationDict end: offset=%d ***\n", ctx.Write.Offset)
	}

	return
}

func writePageAnnotations(ctx *types.PDFContext, pdfObject interface{}) (written bool, err error) {

	logInfoWriter.Printf("*** writePageAnnotations begin: offset=%d ***\n", ctx.Write.Offset)

	var arr types.PDFArray

	if indRef, ok := pdfObject.(types.PDFIndirectRef); ok {

		objNumber := indRef.ObjectNumber.Value()
		genNumber := indRef.GenerationNumber.Value()

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("*** writePageAnnotations end: object #%d already written. offset=%d ***\n", objNumber, ctx.Write.Offset)
			return true, nil
		}

		arrp, err := ctx.DereferenceArray(indRef)
		if err != nil || arrp == nil {
			return false, errors.New("writePageAnnotations: corrupt array of annots dicts")
		}

		arr = *arrp

		logInfoWriter.Printf("writePageAnnotations: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

		err = writePDFArrayObject(ctx, objNumber, genNumber, arr)
		if err != nil {
			return false, err
		}

		logDebugWriter.Printf("writePageAnnotations: new offset after arr = %d\n", ctx.Write.Offset)

	} else {
		arr, ok = pdfObject.(types.PDFArray)
		if !ok {
			return false, errors.New("writePageAnnotations: corrupt array of annots dicts")
		}
	}

	// array of indrefs to annotation dicts.
	var annotsDict types.PDFDict

	for _, v := range arr {

		if indRef, ok := v.(types.PDFIndirectRef); ok {

			objNumber := indRef.ObjectNumber.Value()
			genNumber := indRef.GenerationNumber.Value()

			if ctx.Write.HasWriteOffset(objNumber) {
				logInfoWriter.Printf("writePageAnnotations: object #%d already written.\n", objNumber)
				continue
			}

			annotsDictp, err := ctx.DereferenceDict(indRef)
			if err != nil || annotsDictp == nil {
				return false, errors.New("writePageAnnotations: corrupted annotation dict")
			}

			annotsDict = *annotsDictp

			logInfoWriter.Printf("writePageAnnotations: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

			err = writePDFDictObject(ctx, objNumber, genNumber, annotsDict)
			if err != nil {
				return false, err
			}

			logDebugWriter.Printf("writePageAnnotations: new offset after annotsDict = %d\n", ctx.Write.Offset)

		} else if annotsDict, ok = v.(types.PDFDict); !ok {
			return false, errors.New("writePageAnnotations: corrupted array of indrefs")
		}

		err = writeAnnotationDict(ctx, annotsDict)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writePageAnnotations end: offset=%d ***\n", ctx.Write.Offset)

	return false, nil
}

func processPageAnnotations(ctx *types.PDFContext, objNumber, genNumber int, pagesDict types.PDFDict) (written bool, err error) {

	logPages.Printf("*** processPageAnnotations begin: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	// Annotations
	if pdfObject, ok := pagesDict.Find("Annots"); ok {
		written, err = writePageAnnotations(ctx, pdfObject)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** processPageAnnotations end: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	return
}

func writePagesAnnotations(ctx *types.PDFContext, indRef types.PDFIndirectRef) (written bool, err error) {

	logInfoWriter.Printf("*** writePagesAnnotations begin: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	obj, err := ctx.Dereference(indRef)
	if err != nil {
		logInfoWriter.Printf("writePagesAnnotations end: obj#%d s nil\n", objNumber)
		return
	}

	pagesDict, ok := obj.(types.PDFDict)
	if !ok {
		return false, errors.New("writePagesAnnotations: corrupt pages dict")
	}

	// Get number of pages of this PDF file.
	count, ok := pagesDict.Find("Count")
	if !ok {
		return false, errors.New("writePagesAnnotations: missing \"Count\"")
	}

	pageCount := int(count.(types.PDFInteger))
	logInfoWriter.Printf("writePagesAnnotations: This page node has %d pages\n", pageCount)

	// Iterate over page tree.
	kidsArray := pagesDict.PDFArrayEntry("Kids")

	for _, v := range *kidsArray {

		// Dereference next page node dict.
		indRef, ok := v.(types.PDFIndirectRef)
		if !ok {
			return false, errors.New("writePagesAnnotations: corrupt page node dict")
		}

		logInfoWriter.Printf("writePagesAnnotations: PageNode: %s\n", indRef)

		objNumber = int(indRef.ObjectNumber)
		genNumber = int(indRef.GenerationNumber)

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return false, errors.New("writePagesAnnotations: corrupt pageNodeDict")
		}

		if pageNodeDict == nil {
			return false, errors.New("writePagesAnnotations: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if *dictType == "Pages" {
			// Recurse over pagetree
			smthWritten, err := writePagesAnnotations(ctx, indRef)
			if err != nil {
				return false, err
			}
			if !written {
				written = smthWritten
			}
			continue
		}

		if *dictType != "Page" {
			return false, errors.Errorf("writePagesAnnotations: expected dict type: %s\n", *dictType)
		}

		// Write page dict.
		smthWritten, err := processPageAnnotations(ctx, objNumber, genNumber, *pageNodeDict)
		if err != nil {
			return false, err
		}
		if !written {
			written = smthWritten
		}
	}

	logInfoWriter.Printf("*** writePagesAnnotations end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

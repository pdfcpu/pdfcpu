package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateBorderStyleDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateBorderStyleDict begin ***")

	dict, err := validateDict(xRefTable, obj)
	if err != nil {
		return err
	}

	if dict == nil {
		logInfoValidate.Println("validateBorderStyleDict end: is nil.")
		return
	}

	if dict.Type() != nil && *dict.Type() != "Border" {
		return errors.New("validateBorderStyleDict: corrupt entry \"Type\"")
	}

	// Dash array, optional
	if indRef := dict.IndirectRefEntry("D"); indRef != nil {
		return errors.New("validateBorderStyleDict: *** unsupported entry \"D\" ***")
	}

	// Border effect dict, optional
	if _, found := dict.Find("BE"); found {
		return errors.New("validateBorderStyleDict: *** unsupported entry \"BE\" ***")
	}

	logInfoValidate.Println("*** validateBorderStyleDict end ***")

	return
}

func validateAnnotationDictLink(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAnnotationDictLink begin ***")

	//xRefTable := source.XRefTable

	// optional Dest or A entry

	// A: dictionary
	//    The action that shall be performed when this item is activated.
	if obj, found := dict.Find("A"); found {

		if _, found = dict.Find("Dest"); found {
			return errors.New("validateAnnotationDictLink: only Dest or A allowed")
		}

		err = validateActionDict(xRefTable, obj)
		if err != nil {
			return
		}

	}

	// Dest: name, byte string, or array
	//       The destination that shall be displayed when this item is activated.
	if obj, found := dict.Find("Dest"); found {

		if _, found = dict.Find("A"); found {
			return errors.New("validateAnnotationDictLink: only Dest or A allowed")
		}

		err = validateDestination(xRefTable, obj)
		if err != nil {
			return
		}

	}

	if _, found := dict.Find("PA"); found {
		return errors.New("validateAnnotationDictLink: unsupported entry \"PA\"")
	}

	if _, found := dict.Find("QuadPoints"); found {
		return errors.New("validateAnnotationDictLink: unsupported entry \"QuadPoints\"")
	}

	if obj, found := dict.Find("BS"); found {
		err = validateBorderStyleDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAnnotationDictLink end ***")

	return
}

func validateAnnotationDictPopup(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAnnotationDictPopup begin ***")

	// xRefTable := source.XRefTable

	// Parent, dict
	// An indirect reference to the widget annotation’s parent field.
	// for terminal fields: parent field must already be written.
	// if indRef := dict.IndirectRefEntry("Parent"); indRef != nil {
	// 	objNumber := int(indRef.ObjectNumber)
	// 	if !dest.HasWriteOffset(objNumber) {
	// 		return errors.Errorf("*** validateAnnotationDictPopup: unknown parent field obj#:%d\n", objNumber)
	// 	}
	// }

	logInfoValidate.Println("*** validateAnnotationDictPopup end ***")

	return
}

func validateAnnotationDictFreeTextAAPLAKExtras(xRefTable *types.XRefTable, indRef *types.PDFIndirectRef) (err error) {

	logInfoValidate.Println("*** writeAnnotativalidateAnnotationDictFreeTextAAPLAKExtrasonDictFreeTextAAPL_AKExtras begin ***")

	aaplDict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || aaplDict == nil {
		return errors.New("validateAnnotationDictFreeTextAAPLAKExtras: corrupt AAPL:AKExtras dict")
	}

	if indRef = aaplDict.IndirectRefEntry("AAPL:AKPDFAnnotationDictionary"); indRef == nil {
		return errors.New("validateAnnotationDictFreeTextAAPLAKExtras: corrupt AAPL:AKExtras dict, missing entry \"AAPL:AKPDFAnnotationDictionary\"")
	}

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return errors.New("validateAnnotationDictFreeTextAAPLAKExtras: corrupt AAPL:AKPDFAnnotationDictionary dict")
	}

	err = validateAnnotationDict(xRefTable, dict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictFreeTextAAPLAKExtras end ***")

	return nil
}

func validateAnnotationDictFreeText(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAnnotationDictFreeText begin ***")

	if obj, found := dict.Find("BS"); found {
		err = validateBorderStyleDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if indRef := dict.IndirectRefEntry("AAPL:AKExtras"); indRef != nil {
		err = validateAnnotationDictFreeTextAAPLAKExtras(xRefTable, indRef)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAnnotationDictFreeText end ***")

	return
}

func validateAnnotationDictStampAAPLAKExtras(xRefTable *types.XRefTable, indRef *types.PDFIndirectRef) (err error) {

	logInfoValidate.Println("*** validateAnnotationDictStampAAPLAKExtras begin ***")

	aaplDict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || aaplDict == nil {
		return errors.New("validateAnnotationDictStampAAPLAKExtras: corrupt AAPL:AKExtras dict")
	}

	if indRef = aaplDict.IndirectRefEntry("AAPL:AKPDFAnnotationDictionary"); indRef == nil {
		return errors.New("validateAnnotationDictStampAAPLAKExtras: corrupt AAPL:AKExtras dict, missing entry \"AAPL:AKPDFAnnotationDictionary\"")
	}

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return errors.New("validateAnnotationDictStampAAPLAKExtras: corrupt AAPL:AKPDFAnnotationDictionary dict")
	}

	err = validateAnnotationDict(xRefTable, dict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictStampAAPLAKExtras end ***")

	return
}

func validateAnnotationDictStamp(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAnnotationDictStamp begin ***")

	if indRef := dict.IndirectRefEntry("AAPL:AKExtras"); indRef != nil {
		err = validateAnnotationDictStampAAPLAKExtras(xRefTable, indRef)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAnnotationDictStamp end ***")

	return
}

func validateAnnotationDictWidget(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** validateAnnotationDictWidget begin ***")

	// MK, optional, dict
	// An appearance characteristics dictionary that shall be used in constructing
	// a dynamic appearance stream specifying the annotation’s visual presentation on the page.dict
	if indRef := dict.IndirectRefEntry("MK"); indRef != nil {

		dict, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}

		if dict == nil {
			err = errors.New("validateAnnotationDictWidget: corrupt MK dict")
			return
		}

	}

	// A, optional, dict
	// An action that shall be performed when the annotation is activated.
	if indRef := dict.IndirectRefEntry("A"); indRef != nil {

		dict, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}

		if dict == nil {
			err = errors.New("validateAnnotationDictWidget: corrupt A action dict")
			return
		}

	}

	// AA, optional, dict
	// An additional-actions dictionary defining the annotation’s behaviour in response to various trigger events.
	if _, ok := dict.Find("AA"); ok {
		//log.Fatalln("writeAnnotationDictWidget: unsupported entry \"AA\"")
	}

	// BS, optional, dict
	// A border style dictionary specifying the width and dash pattern
	// that shall be used in drawing the annotation’s border.
	if obj, found := dict.Find("BS"); found {
		err = validateBorderStyleDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// Parent, dict, required if one of multiple children in a field.
	// An indirect reference to the widget annotation’s parent field.
	// for terminal fields: parent field must already be written.
	if indRef := dict.IndirectRefEntry("Parent"); indRef != nil {
		//objNumber := int(indRef.ObjectNumber)
	}

	logInfoValidate.Println("*** validateAnnotationDictWidget end ***")

	return
}

// TODO implement
func validateAnnotationDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAnnotationDict begin ***")

	// handle annotation types

	if dict.Type() != nil && *dict.Type() != "Annot" {
		return errors.New("validateAnnotationDict: corrupt Annot dict type")
	}

	subtype := dict.Subtype()
	if subtype == nil {
		return errors.New("validateAnnotationDict: missing Annot dict subtype")
	}

	//  P, optional, corresponding page object
	//  All pages written at this point.
	if indRef := dict.IndirectRefEntry("P"); indRef != nil {
		//objNumber := indRef.ObjectNumber.Value()
	}

	// appearance stream, optional
	if obj, ok := dict.Find("AP"); ok {
		err = validateAppearanceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// optional content group or optional content membership dictionary
	// specifying the optional content properties for the annotation.
	if _, ok := dict.Find("OC"); ok {
		return errors.New("validateAnnotationDict: unsupported entry OC")
	}

	switch *subtype {

	case "Text":
		// passthough - entry Popup not part of standard.

	case "Link":
		err = validateAnnotationDictLink(xRefTable, dict)

	case "FreeText":
		err = validateAnnotationDictFreeText(xRefTable, dict)

	case "Line":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Square":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Circle":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Polygon":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "PolyLine":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Highlight":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Underline":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Squiggly":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "StrikeOut":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Stamp":
		err = validateAnnotationDictStamp(xRefTable, dict)

	case "Caret":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Ink":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Popup":
		err = validateAnnotationDictPopup(xRefTable, dict)

	case "FileAttachment":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Sound":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Movie":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Widget":
		err = validateAnnotationDictWidget(xRefTable, dict)

	case "Screen":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "PrinterMark":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "TrapNet":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Watermark":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "3D":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	case "Redact":
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	default:
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	}

	// AP
	// P
	// V
	// Dest a13
	// A -> Action mit URI

	if err == nil {
		logInfoValidate.Println("*** validateAnnotationDict end ***")
	}

	return
}

func validatePageAnnotations(xRefTable *types.XRefTable, pdfObject interface{}) (err error) {

	logInfoValidate.Println("*** validatePageAnnotations begin ***")

	var arr types.PDFArray

	if indRef, ok := pdfObject.(types.PDFIndirectRef); ok {

		arrp, err := xRefTable.DereferenceArray(indRef)
		if err != nil || arrp == nil {
			return errors.New("validatePageAnnotations: corrupt array of annots dicts")
		}

		arr = *arrp

	} else {
		arr, ok = pdfObject.(types.PDFArray)
		if !ok {
			return errors.New("validatePageAnnotations: corrupt array of annots dicts")
		}
	}

	// array of indrefs to annotation dicts.
	var annotsDict types.PDFDict

	for _, v := range arr {

		if indRef, ok := v.(types.PDFIndirectRef); ok {

			annotsDictp, err := xRefTable.DereferenceDict(indRef)
			if err != nil || annotsDictp == nil {
				return errors.New("validatePageAnnotations: corrupted annotation dict")
			}

			annotsDict = *annotsDictp

		} else if annotsDict, ok = v.(types.PDFDict); !ok {
			return errors.New("validatePageAnnotations: corrupted array of indrefs")
		}

		err = validateAnnotationDict(xRefTable, &annotsDict)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validatePageAnnotations end ***")

	return
}

func processPageAnnotations(xRefTable *types.XRefTable, objNumber, genNumber int, pagesDict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** processPageAnnotations begin: obj#%d ***\n", objNumber)

	// Annotations
	if pdfObject, ok := pagesDict.Find("Annots"); ok {
		err = validatePageAnnotations(xRefTable, pdfObject)
		if err != nil {
			return
		}
	}

	logInfoValidate.Printf("*** processPageAnnotations end: obj#%d ***\n", objNumber)

	return
}

func validatePagesAnnotations(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validatePagesAnnotations begin ***")

	// Get number of pages of this PDF file.
	pageCount := dict.IntEntry("Count")
	if pageCount == nil {
		return errors.New("validatePagesAnnotations: missing \"Count\"")
	}

	logInfoValidate.Printf("validatePagesAnnotations: This page node has %d pages\n", *pageCount)

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")

	for _, v := range *kidsArray {

		if v == nil {
			logDebugValidate.Println("validatePagesAnnotations: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := v.(types.PDFIndirectRef)
		if !ok {
			return errors.New("validatePagesAnnotations: corrupt page node dict")
		}

		logInfoValidate.Printf("validatePagesAnnotations: PageNode: %s\n", indRef)

		objNumber := indRef.ObjectNumber.Value()
		genNumber := indRef.GenerationNumber.Value()

		pageNodeDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return err
			//return errors.New("validatePagesAnnotations: corrupt pageNodeDict")
		}

		if pageNodeDict == nil {
			return errors.New("validatePagesAnnotations: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return errors.New("validatePagesAnnotations: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			err = validatePagesAnnotations(xRefTable, pageNodeDict)
			if err != nil {
				return err
			}

		case "Page":
			err = processPageAnnotations(xRefTable, objNumber, genNumber, pageNodeDict)
			if err != nil {
				return err
			}

		default:
			return errors.Errorf("validatePagesAnnotations: expected dict type: %s\n", *dictType)

		}

	}

	logInfoValidate.Println("*** validatePagesAnnotations end ***")

	return
}

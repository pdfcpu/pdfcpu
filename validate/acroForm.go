package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateSignatureDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateSignatureDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateSignatureDict end: object is nil.")
		return
	}

	// process signature dict fields.

	if dict.Type() != nil && *dict.Type() != "Sig" {
		return errors.New("validateSignatureDict: type must be \"Sig\"")
	}

	logInfoValidate.Println("*** validateSignatureDict end ***")

	return
}

func validateAppearanceSubDict(xRefTable *types.XRefTable, subDict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAppearanceSubDict begin ***")

	// dict of stream objects.
	for _, obj := range subDict.Dict {

		sd, err := xRefTable.DereferenceStreamDict(obj)
		if err != nil {
			return err
		}

		if sd == nil {
			continue
		}

		err = validateXObjectStreamDict(xRefTable, sd)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Println("*** validateAppearanceSubDict end ***")

	return
}

func validateAppearanceDictEntry(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// stream or dict
	// single appearance stream or subdict

	logInfoValidate.Println("*** validateAppearanceDictEntry begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateAppearanceDictEntry end")
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateAppearanceSubDict(xRefTable, &obj)

	case types.PDFStreamDict:
		err = validateXObjectStreamDict(xRefTable, &obj)

	default:
		err = errors.New("validateAppearanceDictEntry: unsupported PDF object")

	}

	logInfoValidate.Println("*** validateAppearanceDictEntry end ***")

	return
}

func validateAppearanceDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// see 12.5.5 Appearance Streams

	logInfoValidate.Println("*** validateAppearanceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateAppearanceDict end")
		return
	}

	obj, ok := dict.Find("N")
	if !ok {
		if xRefTable.ValidationMode == types.ValidationStrict {
			return errors.New("validateAppearanceDict: missing required entry \"N\"")
		}
	} else {
		err = validateAppearanceDictEntry(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// Rollover Appearance
	if obj, ok = dict.Find("R"); ok {
		err = validateAppearanceDictEntry(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// Down Appearance
	if obj, ok = dict.Find("D"); ok {
		err = validateAppearanceDictEntry(xRefTable, obj)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAppearanceDict end ***")

	return
}

func validateAcroFieldDictEntries(xRefTable *types.XRefTable, dict *types.PDFDict, terminalNode bool, inFieldType *types.PDFName) (outFieldType *types.PDFName, err error) {

	logInfoValidate.Println("*** validateAcroFieldDictEntries begin ***")

	dictName := "acroFieldDict"

	// FT: name, Btn,Tx,Ch,Sig
	_, err = validateNameEntry(xRefTable, dict, dictName, "FT", terminalNode && inFieldType == nil, types.V10, validateAcroFieldType)
	if err != nil {
		return
	}

	logInfoValidate.Printf("validateAcroFieldDictEntries, inFieldType=%v outFieldType=%v", inFieldType, outFieldType)

	// Parent, required if this is a child in the field hierarchy.
	_, err = validateIndRefEntry(xRefTable, dict, dictName, "Parent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// T, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// TU, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, dict, dictName, "TU", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// TM, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, dict, dictName, "TM", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// Ff, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Ff", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// V, optional, various
	err = validateAnyEntry(xRefTable, dict, dictName, "V", OPTIONAL)
	if err != nil {
		return
	}

	// DV, optional, various
	err = validateAnyEntry(xRefTable, dict, dictName, "DV", OPTIONAL)
	if err != nil {
		return
	}

	// AA, optional, dict, since V1.2
	err = validateAdditionalActions(xRefTable, dict, "acroFieldDict", "AA", OPTIONAL, types.V14, "fieldOrAnnot")
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAcroFieldDictEntries end ***")

	return
}

func validateAcroFieldDict(xRefTable *types.XRefTable, indRef *types.PDFIndirectRef, inFieldType *types.PDFName) (err error) {

	objNr := int(indRef.ObjectNumber)

	logInfoValidate.Printf("*** validateAcroFieldDict begin: obj#:%d ***\n", objNr)

	var (
		dict         *types.PDFDict
		xInFieldType *types.PDFName
	)

	dict, err = xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateAcroFieldDict: is nil")
		return nil
	}

	if pdfObject, ok := dict.Find("Kids"); ok {

		// dict represents a non terminal field.
		if dict.Subtype() != nil && *dict.Subtype() == "Widget" {
			return errors.New("validateAcroFieldDict: non terminal field can not be widget annotation")
		}

		// Write field entries.
		xInFieldType, err = validateAcroFieldDictEntries(xRefTable, dict, false, inFieldType)
		if err != nil {
			return err
		}

		// Recurse over kids.
		var arr *types.PDFArray
		arr, err = xRefTable.DereferenceArray(pdfObject)
		if err != nil || arr == nil {
			return err
		}

		for _, value := range *arr {

			indRef, ok := value.(types.PDFIndirectRef)
			if !ok {
				return errors.New("validateAcroFieldDict: corrupt kids array: entries must be indirect reference")
			}

			err = validateAcroFieldDict(xRefTable, &indRef, xInFieldType)
			if err != nil {
				return err
			}

		}

		logInfoValidate.Printf("*** validateAcroFieldDict end: obj#:%d ***", indRef.ObjectNumber)

		return

	}

	// dict represents a terminal field and must have Subtype "Widget"
	validateNameEntry(xRefTable, dict, "acroFieldDict", "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "Widget" })
	if err != nil {
		return
	}

	// Write field entries.
	_, err = validateAcroFieldDictEntries(xRefTable, dict, true, inFieldType)
	if err != nil {
		return
	}

	// Validate widget annotation - Validation of AA redundant because of merged acrofield with widget annotation.
	err = validateAnnotationDict(xRefTable, dict)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateAcroFieldDict end: obj#:%d ***", indRef.ObjectNumber)

	return
}

func validateAcroFormFields(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateAcroFormFields begin ***")

	var arr *types.PDFArray

	arr, err = xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validateAcroFormFields end: is nil.")
		return
	}

	for _, value := range *arr {

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			return errors.New("validateAcroFormFields: corrupt form field array entry")
		}

		err = validateAcroFieldDict(xRefTable, &indRef, nil)
		if err != nil {
			return
		}

	}

	logInfoValidate.Printf("*** validateAcroFormFields end ***")

	return
}

func validateAcroFormEntryCO(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.3 Trigger Events
	// Array of indRefs to field dicts with calculation actions, since V1.3

	logInfoValidate.Println("*** validateAcroFormEntryCO begin ***")

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateAcroFormEntryCO: unsupported in version %s.\n", xRefTable.VersionString())
	}

	var (
		arr  *types.PDFArray
		dict *types.PDFDict
	)

	arr, err = xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if arr == nil {
		logDebugValidate.Println("validateAcroFormEntryCO: end, is nil")
		return
	}

	for _, obj := range *arr {

		dict, err = xRefTable.DereferenceDict(obj)
		if err != nil || dict == nil {
			return
		}

		err = validateAnnotationDict(xRefTable, dict)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateAcroFormEntryCO end ***")

	return
}

func validateAcroFormEntryXFA(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.8

	logInfoValidate.Println("*** validateAcroFormEntryXFA begin ***")

	// streamDict or array of text,streamDict pairs

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateAcroFormEntryXFA end")
		return
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		// no further processing

	case types.PDFArray:

		i := 0

		for _, v := range obj {

			if v == nil {
				return errors.New("validateAcroFormEntryXFA: array entry is nil")
			}

			var o interface{}

			o, err = xRefTable.Dereference(v)
			if err != nil {
				return
			}

			if i%2 == 0 {

				_, ok := o.(types.PDFStringLiteral)
				if !ok {
					return errors.New("validateAcroFormEntryXFA: even array must be a string")
				}

			} else {

				_, ok := o.(types.PDFStreamDict)
				if !ok {
					return errors.New("validateAcroFormEntryXFA: odd array entry must be a streamDict")
				}

			}

			i++
		}

	default:
		return errors.New("validateAcroFormEntryXFA: needs to be streamDict or array")
	}

	logInfoValidate.Println("*** validateAcroFormEntryXFA end ***")

	return
}

func validateAcroForm(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.7.2 Interactive Form Dictionary

	logInfoValidate.Println("*** validateAcroForm begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "AcroForm", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateAcroForm end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateAcroForm: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Fields, required, array of indirect references
	obj, ok := dict.Find("Fields")
	if !ok {
		return errors.New("validateAcroForm: missing required entry \"Fields\"")
	}

	err = validateAcroFormFields(xRefTable, obj)
	if err != nil {
		return
	}

	dictName := "acroFormDict"

	// NeedAppearances: optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NeedAppearances", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// SigFlags: optional, since 1.3, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "SigFlags", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// CO: array
	if obj, ok := dict.Find("CO"); ok {
		err = validateAcroFormEntryCO(xRefTable, obj, types.V13)
		if err != nil {
			return
		}
	}

	// DR, optional, resource dict
	if obj, ok := dict.Find("DR"); ok {
		_, err = validateResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// DA: optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "DA", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Q: optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Q", OPTIONAL, types.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return
	}

	// XFA: optional, since 1.5, stream or array
	if obj, ok := dict.Find("XFA"); ok {
		err = validateAcroFormEntryXFA(xRefTable, obj, types.V15)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAcroForm end ***")

	return
}

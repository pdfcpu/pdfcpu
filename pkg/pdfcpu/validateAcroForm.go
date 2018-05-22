package pdfcpu

import (
	"github.com/pkg/errors"
)

func validateSignatureDict(xRefTable *XRefTable, obj PDFObject) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, "signatureDict", "Type", OPTIONAL, V10, func(s string) bool { return s == "Sig" })

	// process signature dict fields.

	return err
}

func validateAppearanceSubDict(xRefTable *XRefTable, subDict *PDFDict) error {

	// dict of xobjects
	for _, obj := range subDict.Dict {

		err := validateXObjectStreamDict(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateAppearanceDictEntry(xRefTable *XRefTable, obj PDFObject) error {

	// stream or dict
	// single appearance stream or subdict

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case PDFDict:
		err = validateAppearanceSubDict(xRefTable, &obj)

	case PDFStreamDict:
		err = validateXObjectStreamDict(xRefTable, obj)

	default:
		err = errors.New("validateAppearanceDictEntry: unsupported PDF object")

	}

	return err
}

func validateAppearanceDict(xRefTable *XRefTable, obj PDFObject) error {

	// see 12.5.5 Appearance Streams

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Normal Appearance
	obj, ok := dict.Find("N")
	if !ok {
		if xRefTable.ValidationMode == ValidationStrict {
			return errors.New("validateAppearanceDict: missing required entry \"N\"")
		}
	} else {
		err = validateAppearanceDictEntry(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	// Rollover Appearance
	if obj, ok = dict.Find("R"); ok {
		err = validateAppearanceDictEntry(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	// Down Appearance
	if obj, ok = dict.Find("D"); ok {
		err = validateAppearanceDictEntry(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateAcroFieldDictEntries(xRefTable *XRefTable, dict *PDFDict, terminalNode bool, inFieldType *PDFName) (outFieldType *PDFName, err error) {

	dictName := "acroFieldDict"

	// FT: name, Btn,Tx,Ch,Sig
	validate := func(s string) bool { return memberOf(s, []string{"Btn", "Tx", "Ch", "Sig"}) }
	fieldType, err := validateNameEntry(xRefTable, dict, dictName, "FT", terminalNode && inFieldType == nil, V10, validate)
	if err != nil {
		return nil, err
	}

	if fieldType != nil {
		outFieldType = fieldType
	}

	// Parent, required if this is a child in the field hierarchy.
	_, err = validateIndRefEntry(xRefTable, dict, dictName, "Parent", OPTIONAL, V10)
	if err != nil {
		return nil, err
	}

	// T, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, V10, nil)
	if err != nil {
		return nil, err
	}

	// TU, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, dict, dictName, "TU", OPTIONAL, V13, nil)
	if err != nil {
		return nil, err
	}

	// TM, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, dict, dictName, "TM", OPTIONAL, V13, nil)
	if err != nil {
		return nil, err
	}

	// Ff, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Ff", OPTIONAL, V10, nil)
	if err != nil {
		return nil, err
	}

	// V, optional, various
	_, err = validateEntry(xRefTable, dict, dictName, "V", OPTIONAL, V10)
	if err != nil {
		return nil, err
	}

	// DV, optional, various
	_, err = validateEntry(xRefTable, dict, dictName, "DV", OPTIONAL, V10)
	if err != nil {
		return nil, err
	}

	// AA, optional, dict, since V1.2
	err = validateAdditionalActions(xRefTable, dict, dictName, "AA", OPTIONAL, V14, "fieldOrAnnot")
	if err != nil {
		return nil, err
	}

	return outFieldType, nil
}

func validateAcroFieldDict(xRefTable *XRefTable, indRef *PDFIndirectRef, inFieldType *PDFName) error {

	dict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return err
	}

	if pdfObject, ok := dict.Find("Kids"); ok {

		// dict represents a non terminal field.
		if dict.Subtype() != nil && *dict.Subtype() == "Widget" {
			return errors.New("validateAcroFieldDict: non terminal field can not be widget annotation")
		}

		// Write field entries.
		var xInFieldType *PDFName
		xInFieldType, err = validateAcroFieldDictEntries(xRefTable, dict, false, inFieldType)
		if err != nil {
			return err
		}

		// Recurse over kids.
		var arr *PDFArray
		arr, err = xRefTable.DereferenceArray(pdfObject)
		if err != nil || arr == nil {
			return err
		}

		for _, value := range *arr {

			indRef, ok := value.(PDFIndirectRef)
			if !ok {
				return errors.New("validateAcroFieldDict: corrupt kids array: entries must be indirect reference")
			}

			err = validateAcroFieldDict(xRefTable, &indRef, xInFieldType)
			if err != nil {
				return err
			}

		}

		return nil
	}

	// dict represents a terminal field and must have Subtype "Widget"
	_, err = validateNameEntry(xRefTable, dict, "acroFieldDict", "Subtype", REQUIRED, V10, func(s string) bool { return s == "Widget" })
	if err != nil {
		return err
	}

	// Validate field dict entries.
	_, err = validateAcroFieldDictEntries(xRefTable, dict, true, inFieldType)
	if err != nil {
		return err
	}

	// Validate widget annotation - Validation of AA redundant because of merged acrofield with widget annotation.
	_, err = validateAnnotationDict(xRefTable, dict)

	return err
}

func validateAcroFormFields(xRefTable *XRefTable, obj PDFObject) error {

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil || arr == nil {
		return err
	}

	for _, value := range *arr {

		indRef, ok := value.(PDFIndirectRef)
		if !ok {
			return errors.New("validateAcroFormFields: corrupt form field array entry")
		}

		err = validateAcroFieldDict(xRefTable, &indRef, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateAcroFormCO(xRefTable *XRefTable, obj PDFObject, sinceVersion PDFVersion) error {

	// see 12.6.3 Trigger Events
	// Array of indRefs to field dicts with calculation actions, since V1.3

	// Version check
	err := xRefTable.ValidateVersion("AcroFormCO", sinceVersion)
	if err != nil {
		return err
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil || arr == nil {
		return err
	}

	for _, obj := range *arr {

		dict, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		_, err = validateAnnotationDict(xRefTable, dict)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateAcroFormXFA(xRefTable *XRefTable, dict *PDFDict, sinceVersion PDFVersion) error {

	// see 12.7.8

	obj, ok := dict.Find("XFA")
	if !ok {
		return nil
	}

	// streamDict or array of text,streamDict pairs

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case PDFStreamDict:
		// no further processing

	case PDFArray:

		i := 0

		for _, v := range obj {

			if v == nil {
				return errors.New("validateAcroFormXFA: array entry is nil")
			}

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return err
			}

			if i%2 == 0 {

				_, ok := o.(PDFStringLiteral)
				if !ok {
					return errors.New("validateAcroFormXFA: even array must be a string")
				}

			} else {

				_, ok := o.(PDFStreamDict)
				if !ok {
					return errors.New("validateAcroFormXFA: odd array entry must be a streamDict")
				}

			}

			i++
		}

	default:
		return errors.New("validateAcroFormXFA: needs to be streamDict or array")
	}

	return xRefTable.ValidateVersion("AcroFormXFA", sinceVersion)
}

func validateQ(i int) bool { return i >= 0 && i <= 2 }

func validateAcroFormEntryCO(xRefTable *XRefTable, dict *PDFDict, sinceVersion PDFVersion) error {

	obj, ok := dict.Find("CO")
	if !ok {
		return nil
	}

	return validateAcroFormCO(xRefTable, obj, sinceVersion)
}

func validateAcroFormEntryDR(xRefTable *XRefTable, dict *PDFDict) error {

	obj, ok := dict.Find("DR")
	if !ok {
		return nil
	}

	_, err := validateResourceDict(xRefTable, obj)

	return err
}

func validateAcroForm(xRefTable *XRefTable, rootDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// => 12.7.2 Interactive Form Dictionary

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "AcroForm", OPTIONAL, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	// Version check
	err = xRefTable.ValidateVersion("AcroForm", sinceVersion)
	if err != nil {
		return err
	}

	// Fields, required, array of indirect references
	obj, ok := dict.Find("Fields")
	if !ok {
		return errors.New("validateAcroForm: missing required entry \"Fields\"")
	}

	err = validateAcroFormFields(xRefTable, obj)
	if err != nil {
		return err
	}

	dictName := "acroFormDict"

	// NeedAppearances: optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NeedAppearances", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// SigFlags: optional, since 1.3, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "SigFlags", OPTIONAL, V13, nil)
	if err != nil {
		return err
	}

	// CO: arra
	err = validateAcroFormEntryCO(xRefTable, dict, V13)
	if err != nil {
		return err
	}

	// DR, optional, resource dict
	err = validateAcroFormEntryDR(xRefTable, dict)
	if err != nil {
		return err
	}

	// DA: optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "DA", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// Q: optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Q", OPTIONAL, V10, validateQ)
	if err != nil {
		return err
	}

	// XFA: optional, since 1.5, stream or array
	return validateAcroFormXFA(xRefTable, dict, sinceVersion)
}

package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateSignatureDict(xRefTable *types.XRefTable, obj types.PDFObject) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, "signatureDict", "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Sig" })

	// process signature dict fields.

	return err
}

func validateAppearanceSubDict(xRefTable *types.XRefTable, subDict *types.PDFDict) error {

	// dict of xobjects
	for _, obj := range subDict.Dict {

		err := validateXObjectStreamDict(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateAppearanceDictEntry(xRefTable *types.XRefTable, obj types.PDFObject) error {

	// stream or dict
	// single appearance stream or subdict

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateAppearanceSubDict(xRefTable, &obj)

	case types.PDFStreamDict:
		err = validateXObjectStreamDict(xRefTable, obj)

	default:
		err = errors.New("validateAppearanceDictEntry: unsupported PDF object")

	}

	return err
}

func validateAppearanceDict(xRefTable *types.XRefTable, obj types.PDFObject) error {

	// see 12.5.5 Appearance Streams

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Normal Appearance
	obj, ok := dict.Find("N")
	if !ok {
		if xRefTable.ValidationMode == types.ValidationStrict {
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

func validateAcroFieldDictEntries(xRefTable *types.XRefTable, dict *types.PDFDict, terminalNode bool, inFieldType *types.PDFName) (outFieldType *types.PDFName, err error) {

	dictName := "acroFieldDict"

	// FT: name, Btn,Tx,Ch,Sig
	validate := func(s string) bool { return memberOf(s, []string{"Btn", "Tx", "Ch", "Sig"}) }
	fieldType, err := validateNameEntry(xRefTable, dict, dictName, "FT", terminalNode && inFieldType == nil, types.V10, validate)
	if err != nil {
		return nil, err
	}

	if fieldType != nil {
		outFieldType = fieldType
	}

	// Parent, required if this is a child in the field hierarchy.
	_, err = validateIndRefEntry(xRefTable, dict, dictName, "Parent", OPTIONAL, types.V10)
	if err != nil {
		return nil, err
	}

	// T, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return nil, err
	}

	// TU, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, dict, dictName, "TU", OPTIONAL, types.V13, nil)
	if err != nil {
		return nil, err
	}

	// TM, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, dict, dictName, "TM", OPTIONAL, types.V13, nil)
	if err != nil {
		return nil, err
	}

	// Ff, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Ff", OPTIONAL, types.V10, nil)
	if err != nil {
		return nil, err
	}

	// V, optional, various
	_, err = validateEntry(xRefTable, dict, dictName, "V", OPTIONAL, types.V10)
	if err != nil {
		return nil, err
	}

	// DV, optional, various
	_, err = validateEntry(xRefTable, dict, dictName, "DV", OPTIONAL, types.V10)
	if err != nil {
		return nil, err
	}

	// AA, optional, dict, since V1.2
	err = validateAdditionalActions(xRefTable, dict, dictName, "AA", OPTIONAL, types.V14, "fieldOrAnnot")
	if err != nil {
		return nil, err
	}

	return outFieldType, nil
}

func validateAcroFieldDict(xRefTable *types.XRefTable, indRef *types.PDFIndirectRef, inFieldType *types.PDFName) error {

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
		var xInFieldType *types.PDFName
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

		return nil
	}

	// dict represents a terminal field and must have Subtype "Widget"
	_, err = validateNameEntry(xRefTable, dict, "acroFieldDict", "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "Widget" })
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

func validateAcroFormFields(xRefTable *types.XRefTable, obj types.PDFObject) error {

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil || arr == nil {
		return err
	}

	for _, value := range *arr {

		indRef, ok := value.(types.PDFIndirectRef)
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

func validateAcroFormCO(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

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

func validateAcroFormXFA(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

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

	case types.PDFStreamDict:
		// no further processing

	case types.PDFArray:

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

				_, ok := o.(types.PDFStringLiteral)
				if !ok {
					return errors.New("validateAcroFormXFA: even array must be a string")
				}

			} else {

				_, ok := o.(types.PDFStreamDict)
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

func validateAcroFormEntryCO(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	obj, ok := dict.Find("CO")
	if !ok {
		return nil
	}

	return validateAcroFormCO(xRefTable, obj, sinceVersion)
}

func validateAcroFormEntryDR(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	obj, ok := dict.Find("DR")
	if !ok {
		return nil
	}

	_, err := validateResourceDict(xRefTable, obj)

	return err
}

func validateAcroForm(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

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
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NeedAppearances", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// SigFlags: optional, since 1.3, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "SigFlags", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// CO: arra
	err = validateAcroFormEntryCO(xRefTable, dict, types.V13)
	if err != nil {
		return err
	}

	// DR, optional, resource dict
	err = validateAcroFormEntryDR(xRefTable, dict)
	if err != nil {
		return err
	}

	// DA: optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "DA", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Q: optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Q", OPTIONAL, types.V10, validateQ)
	if err != nil {
		return err
	}

	// XFA: optional, since 1.5, stream or array
	return validateAcroFormXFA(xRefTable, dict, sinceVersion)
}

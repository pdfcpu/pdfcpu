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

	// FT: Btn,Tx,Ch,Sig
	fieldType := dict.PDFNameEntry("FT")
	if terminalNode && fieldType == nil && inFieldType == nil {
		return nil, errors.New("writeAcroFieldDictEntries: missing field type")
	}

	logInfoValidate.Printf("validateAcroFieldDictEntries moving on, inFieldType=%v outFieldType=%v", inFieldType, outFieldType)

	if fieldType != nil {

		outFieldType = fieldType
		logInfoValidate.Printf("validateAcroFieldDictEntries moving on 2, inFieldType=%v outFieldType=%v", inFieldType, outFieldType)

		switch *fieldType {

		case "Btn": // Button field

			//	<DA, (/ZaDb 0 Tf 0 g)>
			//	<FT, Btn>
			//	<Ff, 49152>
			//	<Kids, [(257 0 R) (256 0 R) (255 0 R)]>
			//	<T, (Art)>

		case "Tx": // Text field

		case "Ch": // Choice field
			return nil, errors.New("validateAcroFieldDictEntries: \"Ch\" not supported")

		case "Sig": // Signature field

			if _, ok := dict.Find("Lock"); ok {
				return nil, errors.New("validateAcroFieldDictEntries: \"Lock\" not supported")
			}

			if _, ok := dict.Find("SV"); ok {
				return nil, errors.New("validateAcroFieldDictEntries: \"SV\" not supported")
			}

			// V, optional, signature dictionary containing the signature and specifying various attributes of the signature field.
			if obj, ok := dict.Find("V"); ok {
				err = validateSignatureDict(xRefTable, obj)
				if err != nil {
					return
				}
			}

			// DV, optional, defaultvalue, like V
			if obj, ok := dict.Find("DV"); ok {
				err = validateSignatureDict(xRefTable, obj)
				if err != nil {
					return
				}
			}

			if obj, ok := dict.Find("AP"); ok {
				err = validateAppearanceDict(xRefTable, obj)
				if err != nil {
					return
				}
			}

		default:
			return nil, errors.Errorf("validateAcroFieldDictEntries: unknown fieldType:%s\n", fieldType)
		}
	}

	_, err = validateIndRefEntry(xRefTable, dict, "acroFieldDict", "Parent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// T, optional, text string

	// TU, optional, text string, since V1.3

	// TM, optional, text string, since V1.3

	// Ff, optional, integer

	// V, optional, various

	// DV, optional, various

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

	dict, err := xRefTable.DereferenceDict(*indRef)
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
		xinFieldType, err := validateAcroFieldDictEntries(xRefTable, dict, false, inFieldType)
		if err != nil {
			return err
		}

		// Recurse over kids.
		arr, err := xRefTable.DereferenceArray(pdfObject)
		if err != nil {
			return err
		}

		if arr == nil {
			return nil
		}

		for _, value := range *arr {

			indRef, ok := value.(types.PDFIndirectRef)
			if !ok {
				return errors.New("validateAcroFieldDict: corrupt kids array: entries must be indirect reference")
			}

			err = validateAcroFieldDict(xRefTable, &indRef, xinFieldType)
			if err != nil {
				return err
			}

		}

	} else {

		// dict represents a terminal field.

		if dict.Subtype() == nil || *dict.Subtype() != "Widget" {
			return errors.New("validateAcroFieldDict: terminal field must be widget annotation")
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
	}

	logInfoValidate.Printf("*** validateAcroFieldDict end: obj#:%d ***", indRef.ObjectNumber)

	return
}

func validateAcroFormFields(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateAcroFormFields begin ***")

	arr, err := xRefTable.DereferenceArray(obj)
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
	// since V1.3
	// Array of indRefs to field dicts with calculation actions.
	// => 12.6.3 Trigger Events

	logInfoValidate.Println("*** validateAcroFormEntryCO begin ***")

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateAcroFormEntryCO: unsupported in version %s.\n", xRefTable.VersionString())
	}

	var (
		arr  *types.PDFArray
		dict *types.PDFDict
	)

	arr, err = validateArray(xRefTable, obj)
	if err != nil {
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

// TODO implement
func validateAcroFormEntryXFA(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateAcroFormEntryXFA begin ***")

	err = errors.New("*** validateAcroFormEntryXFA unsupported ***")

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
	// TODO since 1.3, required if any fields in the document have additional-actions dictionaries containing a C entry.ObjectStream
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

	// TODO XFA: optional, since 1.5, stream or array
	if obj, ok := dict.Find("XFA"); ok {
		err = validateAcroFormEntryXFA(xRefTable, obj, types.V15)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAcroForm end ***")

	return
}

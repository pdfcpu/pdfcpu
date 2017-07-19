package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateGoToActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.2

	logInfoValidate.Println("*** validateGoToActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoToActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = validateDestinationEntry(xRefTable, dict, "go-to action dict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGoToActionDict end ***")

	return
}

func validateGoToRActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// Remote go-to action.
	// see 12.6.4.3

	logInfoValidate.Println("*** validateGoToRActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoToRActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "remote go-to action dict"

	err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V11)
	if err != nil {
		return
	}

	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGoToRActionDict end ***")

	return
}

// TODO implement
func validateGoToEActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// Embedded go-to action.
	// see 12.6.4.4

	logInfoValidate.Println("*** validateGoToEActionDict begin ***")

	err = errors.New("validateGoToEActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoToEActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateGoToEActionDict end ***")

	return
}

// TODO implement
func validateLaunchActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.5

	logInfoValidate.Println("*** validateLaunchActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateLaunchActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** writeLaunchActionDict: unsupported action type ***")

	logInfoValidate.Println("*** validateLaunchActionDict end ***")

	return
}

// TODO implement
func validateThreadActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	//see 12.6.4.6

	logInfoValidate.Println("*** validateThreadActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateThreadActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validateThreadActionDict: unsupported action type ***")

	logInfoValidate.Println("*** validateThreadActionDict end ***")

	return
}

func validateURIActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.7

	logInfoValidate.Println("*** validateURIActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateURIActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "uriActionDict"

	_, err = validateStringEntry(xRefTable, dict, dictName, "URI", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "IsMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateURIActionDict end ***")

	return
}

// TODO implement
func validateSoundActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.8

	logInfoValidate.Println("*** validateSoundActionDict begin ***")

	err = errors.New("validateSoundActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSoundActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateSoundActionDict end ***")

	return
}

// TODO implement
func validateMovieActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.9

	logInfoValidate.Println("*** validateMovieActionDict begin ***")

	err = errors.New("validateMovieActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateMovieActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateMovieActionDict end ***")

	return
}

func validateHideActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.10

	logInfoValidate.Println("*** validateHideActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateHideActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// T, required, dict, text string or array
	obj, found := dict.Find("T")
	if !found || obj == nil {
		return errors.New("validateHideActionDict: missing required entry \"T\"")
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateHideActionDict end: is nil")
		return
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		// Ensure UTF16 correctness.
		_, err = types.StringLiteralToString(obj.Value())
		if err != nil {
			return
		}

	case types.PDFDict:
		// annotDict,  Check for required name Subtype
		_, err = validateNameEntry(xRefTable, &obj, "annotDict", "Subtype", REQUIRED, types.V10, nil)
		if err != nil {
			return err
		}

	case types.PDFArray:
		// mixed array of annotationDict indRefs and strings
		for _, v := range obj {
			o, err := xRefTable.Dereference(v)
			if err != nil {
				return err
			}
			if o == nil {
				continue
			}

			switch o := o.(type) {

			case types.PDFStringLiteral:
				// Ensure UTF16 correctness.
				_, err = types.StringLiteralToString(o.Value())
				if err != nil {
					return err
				}

			case types.PDFDict:
				// annotDict,  Check for required name Subtype
				_, err = validateNameEntry(xRefTable, &o, "annotDict", "Subtype", REQUIRED, types.V10, nil)
				if err != nil {
					return err
				}
			}
		}

	default:
		err = errors.Errorf("validateHideActionDict: invalid entry \"T\"")

	}

	// H, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, "hideActionDict", "H", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateHideActionDict end ***")

	return
}

// TODO implement
func validateNamedActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.11

	logInfoValidate.Println("*** validateNamedActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateNamedActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	_, err = validateNameEntry(xRefTable, dict, "namedActionDict", "N", REQUIRED, types.V10, validateNamedAction)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateNamedActionDict end ***")

	return
}

func validateSubmitFormActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.5.2

	logInfoValidate.Println("*** validateSubmitFormActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSubmitFormActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "submitFormActionDict"

	// F, required, string

	//<F, <<
	//	<F, (/servlet/Forms2Web?func=LocalStorage)>
	//	<FS, URL>
	//>>>
	_, err = validateDictEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Fields, optional, array
	// TODO Each element of the array shall be either an indirect reference to a field dictionary or (PDF 1.3) a text string representing the fully qualified name of a field.
	// Elements of both kinds may be mixed in the same array.
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateSubmitFormActionDict end ***")

	return
}

func validateResetFormActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.5.3

	logInfoValidate.Println("*** validateResetFormActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateResetFormActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "resetFormActionDict"

	// Fields, optional, array
	// TODO Each element of the array shall be either an indirect reference to a field dictionary or (PDF 1.3) a text string representing the fully qualified name of a field.
	// Elements of both kinds may be mixed in the same array.
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateResetFormActionDict end ***")

	return
}

// TODO implement
func validateImportDataActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.5.4

	logInfoValidate.Println("*** validateImportDataActionDict begin ***")

	err = errors.New("validateImportDataActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateImportDataActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateImportDataActionDict end ***")

	return
}

func validateJavaScriptActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.16

	logInfoValidate.Println("*** validateJavaScriptActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateJavaScriptActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// S: name, required, action Type
	_, err = validateNameEntry(xRefTable, dict, "javaScriptActionDict", "S", REQUIRED, types.V10, func(s string) bool { return s == "JavaScript" })
	if err != nil {
		return
	}

	obj, found := dict.Find("JS")
	if !found || obj == nil {
		return errors.New("validateJavaScriptActionDict: required entry \"JS\" missing")
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateJavaScriptActionDict end: is nil")
		return
	}

	switch s := obj.(type) {

	case types.PDFStringLiteral:
		// Ensure UTF16 correctness.
		_, err = types.StringLiteralToString(s.Value())
		if err != nil {
			return
		}

	case types.PDFHexLiteral:
		// Ensure UTF16 correctness.
		_, err = types.HexLiteralToString(s.Value())
		if err != nil {
			return
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		err = errors.Errorf("validateJavaScriptActionDict: invalid type\n")

	}

	logInfoValidate.Println("*** validateJavaScriptActionDict end ***")

	return
}

// TODO implement
func validateSetOCGStateActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.12

	logInfoValidate.Println("*** validateSetOCGStateActionDict begin ***")

	err = errors.New("validateSetOCGStateActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSetOCGStateActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateSetOCGStateActionDict end ***")

	return
}

// TODO implement
func validateRenditionActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.13

	logInfoValidate.Println("*** validateRenditionActionDict begin ***")

	err = errors.New("validateRenditionActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateRenditionActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateRenditionActionDict end ***")

	return
}

// TODO implement
func validateTransActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.14

	logInfoValidate.Println("*** validateTransActionDict begin ***")

	err = errors.New("validateTransActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateTransActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateTransActionDict end ***")

	return
}

// TODO implement
func validateGoTo3DViewActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.15

	logInfoValidate.Println("*** validateGoTo3DViewActionDict begin ***")

	err = errors.New("validateGoTo3DViewActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoTo3DViewActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateGoTo3DViewActionDict end ***")

	return
}

func validateActionDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateActionDict begin ***")

	dictName := "actionDict"

	var dict *types.PDFDict

	dict, err = xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Action" })
	if err != nil {
		return
	}

	// S, required, name, action Type
	var s *types.PDFName

	s, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if s == nil {
		logInfoValidate.Println("validateActionDict end: \"S\" is nil")
		return
	}

	switch *s {

	case "GoTo":
		err = validateGoToActionDict(xRefTable, dict, types.V10)

	case "GoToR":
		err = validateGoToRActionDict(xRefTable, dict, types.V10)

	case "GoToE":
		err = validateGoToEActionDict(xRefTable, dict, types.V16)

	case "Launch":
		err = validateLaunchActionDict(xRefTable, dict, types.V10)

	case "Thread":
		err = validateThreadActionDict(xRefTable, dict, types.V10)

	case "URI":
		err = validateURIActionDict(xRefTable, dict, types.V10)

	case "Sound":
		err = validateSoundActionDict(xRefTable, dict, types.V12)

	case "Movie":
		err = validateMovieActionDict(xRefTable, dict, types.V12)

	case "Hide":
		err = validateHideActionDict(xRefTable, dict, types.V12)

	case "Named":
		err = validateNamedActionDict(xRefTable, dict, types.V12)

	case "SubmitForm":
		err = validateSubmitFormActionDict(xRefTable, dict, types.V12)

	case "ResetForm":
		err = validateResetFormActionDict(xRefTable, dict, types.V12)

	case "ImportData":
		err = validateImportDataActionDict(xRefTable, dict, types.V12)

	case "JavaScript":
		err = validateJavaScriptActionDict(xRefTable, dict, types.V13)

	case "SetOCGState":
		err = validateSetOCGStateActionDict(xRefTable, dict, types.V15)

	case "Rendition":
		err = validateRenditionActionDict(xRefTable, dict, types.V15)

	case "Trans":
		err = validateTransActionDict(xRefTable, dict, types.V15)

	case "GoTo3DView":
		err = validateGoTo3DViewActionDict(xRefTable, dict, types.V16)

	default:
		err = errors.Errorf("validateActionDict: unsupported action type: %s\n", *s)

	}

	if err != nil {
		return
	}

	if obj, ok := dict.Find("Next"); ok {

		// either optional action dict
		dict, err = xRefTable.DereferenceDict(obj)
		if err == nil {

			err = validateActionDict(xRefTable, *dict)
			if err != nil {
				return
			}
			logInfoValidate.Println("*** validateActionDict end ***")
			return
		}

		// or optional array of action dicts
		var arr *types.PDFArray

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return
		}

		for _, v := range *arr {
			err = validateActionDict(xRefTable, v)
			if err != nil {
				return
			}
		}

	}

	logInfoValidate.Println("*** validateActionDict end ***")

	return
}

func validateOpenAction(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.2 Destinations, 12.6 Actions

	// A value specifying a destination that shall be displayed
	// or an action that shall be performed when the document is opened.
	// The value shall be either an array defining a destination (see 12.3.2, "Destinations")
	// or an action dictionary representing an action (12.6, "Actions").
	//
	// If this entry is absent, the document shall be opened
	// to the top of the first page at the default magnification factor.

	logInfoValidate.Println("*** validateOpenAction begin ***")

	obj, found := rootDict.Find("OpenAction")
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateOpenAction: required entry \"OpenAction\" missing")
			return
		}
		logInfoValidate.Println("validateOpenAction end: optional entry \"OpenAction\" not found or nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateOpenAction: unsupported in version %s.\n", xRefTable.VersionString())
	}

	if _, err = xRefTable.DereferenceDict(obj); err == nil {

		err = validateActionDict(xRefTable, obj)
		if err != nil {
			return
		}

		logInfoValidate.Println("*** validateOpenAction end ***")
		return
	}

	if _, err = xRefTable.DereferenceArray(obj); err == nil {

		err = validateDestination(xRefTable, obj)
		if err != nil {
			return
		}

		logInfoValidate.Println("*** validateOpenAction end ***")
		return
	}

	return errors.New("validateOpenAction: must be dict or array")
}

func validateAdditionalActions(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion, source string) (err error) {

	logInfoValidate.Println("*** validateAdditionalActions begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateAdditionalActions: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateAdditionalActions end: dict=%s optional entry %s not found or nil\n", dictName, entryName)
		return
	}

	d, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.Errorf("validateAdditionalActions: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAdditionalActions: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	for k, v := range d.Dict {

		if !validateAdditionalAction(k, source) {
			err = errors.Errorf("validateAdditionalActions: action %s not allowed for source %s", k, source)
			return
		}

		err = validateActionDict(xRefTable, v)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateAdditionalActions begin ***")

	return
}

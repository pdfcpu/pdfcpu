package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateGoToActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.2

	logInfoValidate.Println("*** validateGoToActionDict begin ***")

	err = validateDestinationEntry(xRefTable, dict, "go-to action dict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGoToActionDict end ***")

	return
}

func validateGoToRActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// Remote go-to action.
	// see 12.6.4.3

	logInfoValidate.Println("*** validateGoToRActionDict begin ***")

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
func validateGoToEActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// Embedded go-to action.
	// see 12.6.4.4

	logInfoValidate.Println("*** validateGoToEActionDict begin ***")

	err = errors.New("validateGoToEActionDict: unsupported action type")

	if xRefTable.Version() < types.V16 {
		return errors.Errorf("validateGoToEActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateGoToEActionDict end ***")

	return
}

// TODO implement
func validateLaunchActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.5

	logInfoValidate.Println("*** validateLaunchActionDict begin ***")

	err = errors.New("*** writeLaunchActionDict: unsupported action type ***")

	logInfoValidate.Println("*** validateLaunchActionDict end ***")

	return
}

// TODO implement
func validateThreadActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	//see 12.6.4.6

	logInfoValidate.Println("*** validateThreadActionDict begin ***")

	err = errors.New("*** validateThreadActionDict: unsupported action type ***")

	logInfoValidate.Println("*** validateThreadActionDict end ***")

	return
}

func validateURIActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.7

	logInfoValidate.Println("*** validateURIActionDict begin ***")

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
func validateSoundActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.8

	logInfoValidate.Println("*** validateSoundActionDict begin ***")

	err = errors.New("validateSoundActionDict: unsupported action type")

	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateSoundActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateSoundActionDict end ***")

	return
}

// TODO implement
func validateMovieActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.9

	logInfoValidate.Println("*** validateMovieActionDict begin ***")

	err = errors.New("validateMovieActionDict: unsupported action type")

	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateMovieActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateMovieActionDict end ***")

	return
}

// TODO implement
func validateHideActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.10

	logInfoValidate.Println("*** validateHideActionDict begin ***")

	err = errors.New("validateHideActionDict: unsupported action type")

	if xRefTable.Version() < types.V12 {
		err = errors.Errorf("validateHideActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateHideActionDict end ***")

	return
}

// TODO implement
func validateNamedActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.11

	logInfoValidate.Println("*** validateNamedActionDict begin ***")

	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateNamedActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	_, err = validateNameEntry(xRefTable, dict, "namedActionDict", "N", REQUIRED, types.V10, validateNamedAction)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateNamedActionDict end ***")

	return
}

// TODO implement
func validateSubmitFormActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.7.5.2

	logInfoValidate.Println("*** validateSubmitFormActionDict begin ***")

	err = errors.New("validateSubmitFormActionDict: unsupported action type")

	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateSubmitFormActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateSubmitFormActionDict end ***")

	return
}

// TODO implement
func validateResetFormActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.7.5.3

	logInfoValidate.Println("*** validateResetFormActionDict begin ***")

	err = errors.New("validateResetFormActionDict: unsupported action type")

	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateResetFormActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateResetFormActionDict end ***")

	return
}

// TODO implement
func validateImportDataActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.7.5.4

	logInfoValidate.Println("*** validateImportDataActionDict begin ***")

	err = errors.New("validateImportDataActionDict: unsupported action type")

	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validateImportDataActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateImportDataActionDict end ***")

	return
}

// TODO implement
// TODO JS = text string
func validateJavaScriptActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {
	// see 12.6.4.16

	logInfoValidate.Println("*** validateJavaScriptActionDict begin ***")

	err = errors.New("validateJavaScriptActionDict: unsupported action type")

	if xRefTable.Version() < types.V13 {
		return errors.Errorf("validateJavaScriptActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateJavaScriptActionDict end ***")

	return
}

// TODO implement
func validateSetOCGStateActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.12

	logInfoValidate.Println("*** validateSetOCGStateActionDict begin ***")

	err = errors.New("validateSetOCGStateActionDict: unsupported action type")

	if xRefTable.Version() < types.V15 {
		return errors.Errorf("validateSetOCGStateActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateSetOCGStateActionDict end ***")

	return
}

// TODO implement
func validateRenditionActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.13

	logInfoValidate.Println("*** validateRenditionActionDict begin ***")

	err = errors.New("validateRenditionActionDict: unsupported action type")

	if xRefTable.Version() < types.V15 {
		return errors.Errorf("validateRenditionActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateRenditionActionDict end ***")

	return
}

// TODO implement
func validateTransActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.14

	logInfoValidate.Println("*** validateTransActionDict begin ***")

	err = errors.New("validateTransActionDict: unsupported action type")

	if xRefTable.Version() < types.V15 {
		return errors.Errorf("validateTransActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateTransActionDict end ***")

	return
}

// TODO implement
func validateGoTo3DViewActionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 12.6.4.15

	logInfoValidate.Println("*** validateGoTo3DViewActionDict begin ***")

	err = errors.New("validateGoTo3DViewActionDict: unsupported action type")

	if xRefTable.Version() < types.V16 {
		return errors.Errorf("validateGoTo3DViewActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	logInfoValidate.Println("*** validateGoTo3DViewActionDict end ***")

	return
}

func validateActionDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateActionDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateActionDict end: object already written")
		return
	}

	// Process action dict.

	if dict.Type() != nil && *dict.Type() != "Action" {
		return errors.New("validateActionDict: corrupt entry \"Type\"")
	}

	// S: name, required, action Type
	s, err := validateNameEntry(xRefTable, dict, "actionDict", "S", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if s == nil {
		logInfoValidate.Println("validateActionDict end: \"S\" is nil")
		return
	}

	switch *s {

	case "GoTo":
		err = validateGoToActionDict(xRefTable, dict)

	case "GoToR":
		err = validateGoToRActionDict(xRefTable, dict)

	case "GoToE":
		err = validateGoToEActionDict(xRefTable, dict)

	case "Launch":
		err = validateLaunchActionDict(xRefTable, dict)

	case "Thread":
		err = validateThreadActionDict(xRefTable, dict)

	case "URI":
		err = validateURIActionDict(xRefTable, dict)

	case "Sound":
		err = validateSoundActionDict(xRefTable, dict)

	case "Movie":
		err = validateMovieActionDict(xRefTable, dict)

	case "Hide":
		err = validateHideActionDict(xRefTable, dict)

	case "Named":
		err = validateNamedActionDict(xRefTable, dict)

	case "SubmitForm":
		err = validateSubmitFormActionDict(xRefTable, dict)

	case "ResetForm":
		err = validateResetFormActionDict(xRefTable, dict)

	case "ImportData":
		err = validateImportDataActionDict(xRefTable, dict)

	case "JavaScript":
		err = validateJavaScriptActionDict(xRefTable, dict)

	case "SetOCGState":
		err = validateSetOCGStateActionDict(xRefTable, dict)

	case "Rendition":
		err = validateRenditionActionDict(xRefTable, dict)

	case "Trans":
		err = validateTransActionDict(xRefTable, dict)

	case "GoTo3DView":
		err = validateGoTo3DViewActionDict(xRefTable, dict)

	default:
		err = errors.Errorf("validateActionDict: unsupported action type: %s\n", *s)

	}

	if err != nil {
		return
	}

	if _, ok := dict.Find("Next"); ok {
		return errors.New("validateActionDict: unsupported entry \"Next\"")
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

package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateGoToActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.2 Go-To Actions

	logInfoValidate.Println("*** validateGoToActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoToActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, "gotoActionDict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGoToActionDict end ***")

	return
}

func validateGoToRActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.3 Remote Go-To Actions

	logInfoValidate.Println("*** validateGoToRActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoToRActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "gotoRActionDict"

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V11)
	if err != nil {
		return
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGoToRActionDict end ***")

	return
}

func validateTargetDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// table 202

	logInfoValidate.Println("*** validateTargetDictEntry begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "targetDict"

	// R, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "R", REQUIRED, types.V10, func(s string) bool { return s == "P" || s == "C" })
	if err != nil {
		return
	}

	// N, optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P, optional, integer or byte string
	err = validateIntOrStringEntry(xRefTable, d, dictName, "P", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// A, optional, integer or text string
	err = validateIntOrStringEntry(xRefTable, d, dictName, "A", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// T, optional, target dict
	err = validateTargetDictEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTargetDictEntry end ***")

	return
}

func validateGoToEActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.4 Embedded Go-To Actions

	logInfoValidate.Println("*** validateGoToEActionDict begin ***")

	err = errors.New("validateGoToEActionDict: unsupported action type")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoToEActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := " gotoEActionDict"

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	// T, required unless entry F is present, target dict
	err = validateTargetDictEntry(xRefTable, dict, dictName, "T", f == nil, types.V10)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGoToEActionDict end ***")

	return
}

func validateWinDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 204

	logInfoValidate.Println("*** validateWinDict begin ***")

	dictName := "winDict"

	// F, required, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// D, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// O, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateWinDict end ***")

	return
}

func validateLaunchActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.5

	logInfoValidate.Println("*** validateLaunchActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateLaunchActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "launchActionDict"

	// F, optional, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// Win, optional, dict
	winDict, err := validateDictEntry(xRefTable, dict, dictName, "Win", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if winDict != nil {
		err = validateWinDict(xRefTable, winDict)
		if err != nil {
			return
		}
	}

	// Mac, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "Mac", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Unix, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "Unix", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateLaunchActionDict end ***")

	return
}

func validateThreadActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	//see 12.6.4.6

	logInfoValidate.Println("*** validateThreadActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateThreadActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "threadActionDict"

	// F, optional, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// B, optional, integer or bead dict
	err = validateIntOrDictEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V10)
	if err != nil {
		return
	}

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

	// URI, required, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "URI", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// IsMap, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "IsMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateURIActionDict end ***")

	return
}

func validateSoundDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateSoundDictEntry begin ***")

	var sd *types.PDFStreamDict

	sd, err = validateStreamDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return
	}

	dictName = "soundDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, &sd.PDFDict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Sound" })
	if err != nil {
		return
	}

	// R, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "R", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// C, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// B, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// E, optional, name
	validateSampleDataEncoding := func(s string) bool {
		return memberOf(s, []string{"Raw", "Signed", "muLaw", "Alaw"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "E", OPTIONAL, types.V10, validateSampleDataEncoding)

	logInfoValidate.Println("*** validateSoundDictEntry end ***")

	return
}

func validateSoundActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.8

	logInfoValidate.Println("*** validateSoundActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSoundActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "soundActionDict"

	// Sound, required, stream dict
	err = validateSoundDictEntry(xRefTable, dict, dictName, "Sound", REQUIRED, types.V10)
	if err != nil {
		return
	}

	// Volume, optional, number: -1.0 .. +1.0
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Volume", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Synchronous", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Repeat", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Mix, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Mix", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateSoundActionDict end ***")

	return
}

func validateMovieStartOrDurationEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool) (err error) {

	logInfoValidate.Println("*** validateMovieStartOrDurationEntry begin ***")

	obj, ok := dict.Find(entryName)
	if !ok {
		if required {
			return errors.Errorf("validateMovieStartOrDurationEntry: required entry \"%s\" missing", entryName)
		}
		return
	}

	if obj == nil {
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	switch o := obj.(type) {

	case types.PDFInteger, types.PDFStringLiteral:
		// no further processing

	case types.PDFArray:
		if len(o) != 2 {
			return errors.New("validateMovieStartOrDurationEntry: array length <> 2")
		}
	}

	logInfoValidate.Println("*** validateMovieStartOrDurationEntry end ***")

	return
}

func validateMovieActivationDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateMovieActivationDict begin ***")

	dictName := "movieActivationDict"

	// Start, optional
	err = validateMovieStartOrDurationEntry(xRefTable, dict, dictName, "Start", OPTIONAL)
	if err != nil {
		return
	}

	// Duration, optional
	err = validateMovieStartOrDurationEntry(xRefTable, dict, dictName, "Duration", OPTIONAL)
	if err != nil {
		return
	}

	// Rate, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Rate", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Volume, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Volume", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// ShowControls, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "ShowControls", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Mode, optional, name
	validatePlayMode := func(s string) bool {
		return memberOf(s, []string{"Once", "Open", "Repeat", "Palindrome"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Mode", OPTIONAL, types.V10, validatePlayMode)
	if err != nil {
		return
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Synchronous", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// FWScale, optional, array of 2 positive integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "FWScale", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// FWPosition, optional, array of 2 numbers [0.0 .. 1.0]
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "FWPosition", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMovieActivationDict end ***")

	return
}

func validateMovieActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.9

	logInfoValidate.Println("*** validateMovieActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateMovieActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "movieActionDict"

	// is a movie activation dict
	err = validateMovieActivationDict(xRefTable, dict)
	if err != nil {
		return
	}

	// Needs either Annotation or T entry but not both.

	// T, optional, text string
	t, err := validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Annotation, optional, indRef of movie annotation dict
	var (
		indRef *types.PDFIndirectRef
		d      *types.PDFDict
	)

	indRef, err = validateIndRefEntry(xRefTable, dict, dictName, "Annotation", OPTIONAL, types.V10)
	if err != nil {
		return
	}
	if (indRef != nil && t != nil) || (indRef == nil && t == nil) {
		return errors.New("validateMovieActionDict: needs either T or Annotation entry")
	}

	if indRef != nil {
		d, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return
		}
		_, err = validateNameEntry(xRefTable, d, "annotDict", "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "Movie" })
		if err != nil {
			return
		}
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

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return
	}

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	var arr *types.PDFArray
	arr, err = validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if arr != nil {
		for _, v := range *arr {
			switch v.(type) {
			case types.PDFStringLiteral, types.PDFIndirectRef:
				// no further processing

			default:
				return errors.New("validateSubmitFormActionDict: unknown Fields entry")
			}
		}
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
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	var arr *types.PDFArray
	arr, err = validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if arr != nil {
		for _, v := range *arr {
			switch v.(type) {
			case types.PDFStringLiteral, types.PDFIndirectRef:
				// no further processing

			default:
				return errors.New("validateResetFormActionDict: unknown Fields entry")
			}
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateResetFormActionDict end ***")

	return
}

func validateImportDataActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.5.4

	logInfoValidate.Println("*** validateImportDataActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateImportDataActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, "importDataActionDict", "F", OPTIONAL, types.V11)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateImportDataActionDict end ***")

	return
}

func validateJavaScript(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool) (err error) {

	logInfoValidate.Println("*** validateJavaScript begin ***")

	obj, found := dict.Find("JS")
	if !found || obj == nil {
		if required {
			return errors.New("validateJavaScript: required entry \"JS\" missing")
		}
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateJavaScript end: is nil")
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
		err = errors.Errorf("validateJavaScript: invalid type\n")

	}

	logInfoValidate.Println("*** validateJavaScript end ***")

	return
}

func validateJavaScriptActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.16

	logInfoValidate.Println("*** validateJavaScriptActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateJavaScriptActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// JS, required, text string or stream
	err = validateJavaScript(xRefTable, dict, "JavaScriptActionDict", "JS", REQUIRED)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateJavaScriptActionDict end ***")

	return
}

func validateSetOCGStateActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.12

	logInfoValidate.Println("*** validateSetOCGStateActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSetOCGStateActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "setOCGStateActionDict"

	// State, required, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "State", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// PreserveRB, optinal, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "PreserveRB", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateSetOCGStateActionDict end ***")

	return
}

func validateRenditionActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.13

	logInfoValidate.Println("*** validateRenditionActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateRenditionActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "rendActionDict"

	// OP or JS need to be present.

	// OP, integer
	var op *types.PDFInteger
	op, err = validateIntegerEntry(xRefTable, dict, dictName, "OP", OPTIONAL, types.V10, func(i int) bool { return 0 <= i && i <= 4 })
	if err != nil {
		return
	}

	// JS, text string or stream
	err = validateJavaScript(xRefTable, dict, dictName, "JS", op == nil)
	if err != nil {
		return
	}

	var d *types.PDFDict

	// R, required for OP 0 and 4, rendition object dict
	required := func(op *types.PDFInteger) bool {
		if op == nil {
			return false
		}
		v := op.Value()
		return v == 0 || v == 4
	}(op)

	d, err = validateDictEntry(xRefTable, dict, dictName, "R", required, types.V10, nil)
	// if d != nil {
	// 	err = validateRenditionDict(xRefTable, d)
	// 	if err != nil {
	// 		return
	// 	}
	// }

	// AN, required for any OP 0..4, screen annotation dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "AN", op != nil, types.V10, nil)
	if d != nil {
		_, err = validateNameEntry(xRefTable, dict, dictName, "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "Screen" })
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateRenditionActionDict end ***")

	return
}

func validateTransActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.14

	logInfoValidate.Println("*** validateTransActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateTransActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Trans, required, transitionDict
	var d *types.PDFDict
	d, err = validateDictEntry(xRefTable, dict, "transActionDict", "Trans", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	err = validateTransitionDict(xRefTable, d)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTransActionDict end ***")

	return
}

func validateGoTo3DViewActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.6.4.15

	logInfoValidate.Println("*** validateGoTo3DViewActionDict begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateGoTo3DViewActionDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dictName := "goto#dViewActionDict"

	// TA, required, target annotation

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, "TA", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	err = validateAnnotationDict(xRefTable, d)
	if err != nil {
		return
	}

	// V, required, the view to use: 3DViewDict or integer or text string or name
	// TODO Elaborate validation.
	err = validateAnyEntry(xRefTable, dict, dictName, "V", REQUIRED)
	if err != nil {
		return
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

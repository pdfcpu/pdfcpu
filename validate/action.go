package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateGoToActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.2 Go-To Actions

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	return validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
}

func validateGoToRActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.3 Remote Go-To Actions

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, types.V12, nil)

	return err
}

func validateTargetDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	// table 202

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "targetDict"

	// R, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "R", REQUIRED, types.V10, func(s string) bool { return s == "P" || s == "C" })
	if err != nil {
		return err
	}

	// N, optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, integer or byte string
	err = validateIntOrStringEntry(xRefTable, d, dictName, "P", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// A, optional, integer or text string
	err = validateIntOrStringEntry(xRefTable, d, dictName, "A", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// T, optional, target dict
	return validateTargetDictEntry(xRefTable, d, dictName, "T", OPTIONAL, types.V10)
}

func validateGoToEActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.4 Embedded Go-To Actions

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}

	// T, required unless entry F is present, target dict
	return validateTargetDictEntry(xRefTable, dict, dictName, "T", f == nil, types.V10)
}

func validateWinDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see table 204

	dictName := "winDict"

	// F, required, byte string
	_, err := validateStringEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// D, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// O, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)

	return err
}

func validateLaunchActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.5

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// F, optional, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)
	if err != nil {
		return err
	}

	// Win, optional, dict
	winDict, err := validateDictEntry(xRefTable, dict, dictName, "Win", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if winDict != nil {
		err = validateWinDict(xRefTable, winDict)
	}

	// Mac, optional, undefined dict

	// Unix, optional, undefined dict

	return err
}

func validateDestinationThreadEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	// The destination thread (table 205)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateDestinationThreadEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		return nil
	}

	switch obj.(type) {

	case types.PDFDict, types.PDFStringLiteral, types.PDFInteger:
		// an indRef to a thread dictionary
		// or an index of the thread within the roots Threads array
		// or the title of the thread as specified in its thread info dict

	default:
		return errors.Errorf("validateDestinationThreadEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	return xRefTable.ValidateVersion(dictName, sinceVersion)
}

func validateDestinationBeadEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	// The bead in the destination thread (table 205)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateDestinationBeadEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		return nil
	}

	switch obj.(type) {

	case types.PDFDict, types.PDFInteger:
		// an indRef to a bead dictionary of a thread in the current file
		// or an index of the thread within its thread

	default:
		return errors.Errorf("validateDestinationBeadEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	return xRefTable.ValidateVersion(dictName, sinceVersion)
}

func validateThreadActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	//see 12.6.4.6

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// F, optional, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)
	if err != nil {
		return err
	}

	// D, required, indRef to thread dict, integer or text string.
	err = validateDestinationThreadEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// B, optional, indRef to bead dict or integer.
	return validateDestinationBeadEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V10)
}

func validateURIActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.7

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// URI, required, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "URI", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// IsMap, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "IsMap", OPTIONAL, types.V10, nil)

	return err
}

func validateSoundDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	sd, err := validateStreamDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	dictName = "soundDict"
	dict = &sd.PDFDict

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Sound" })
	if err != nil {
		return err
	}

	// R, required, number - sampling rate
	_, err = validateNumberEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// C, required, integer - # of sound channels
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// B, required, integer - bits per sample value per channel
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// E, optional, name - encoding format
	validateSampleDataEncoding := func(s string) bool {
		return memberOf(s, []string{"Raw", "Signed", "muLaw", "ALaw"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "E", OPTIONAL, types.V10, validateSampleDataEncoding)

	return err
}

func validateSoundActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.8

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// Sound, required, stream dict
	err = validateSoundDictEntry(xRefTable, dict, dictName, "Sound", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// Volume, optional, number: -1.0 .. +1.0
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Volume", OPTIONAL, types.V10, func(f float64) bool { return -1.0 <= f && f <= 1.0 })
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Synchronous", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Repeat", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Mix, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Mix", OPTIONAL, types.V10, nil)

	return err
}

func validateMovieStartOrDurationEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool) error {

	logInfoValidate.Println("*** validateMovieStartOrDurationEntry begin ***")

	obj, ok := dict.Find(entryName)
	if !ok {
		if required {
			return errors.Errorf("validateMovieStartOrDurationEntry: required entry \"%s\" missing", entryName)
		}
		return nil
	}
	if obj == nil {
		return nil
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}

	switch o := obj.(type) {

	case types.PDFInteger, types.PDFStringLiteral:
		// no further processing

	case types.PDFArray:
		if len(o) != 2 {
			return errors.New("validateMovieStartOrDurationEntry: array length <> 2")
		}
	}

	return nil
}

func validateMovieActivationDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "movieActivationDict"

	// Start, optional
	err := validateMovieStartOrDurationEntry(xRefTable, dict, dictName, "Start", OPTIONAL)
	if err != nil {
		return err
	}

	// Duration, optional
	err = validateMovieStartOrDurationEntry(xRefTable, dict, dictName, "Duration", OPTIONAL)
	if err != nil {
		return err
	}

	// Rate, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Rate", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Volume, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Volume", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// ShowControls, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "ShowControls", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Mode, optional, name
	validatePlayMode := func(s string) bool {
		return memberOf(s, []string{"Once", "Open", "Repeat", "Palindrome"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Mode", OPTIONAL, types.V10, validatePlayMode)
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Synchronous", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// FWScale, optional, array of 2 positive integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "FWScale", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// FWPosition, optional, array of 2 numbers [0.0 .. 1.0]
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "FWPosition", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })

	return err
}

func validateMovieActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.9

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// is a movie activation dict
	err = validateMovieActivationDict(xRefTable, dict)
	if err != nil {
		return err
	}

	// Needs either Annotation or T entry but not both.

	// T, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err == nil {
		return nil
	}

	// Annotation, indRef of movie annotation dict
	indRef, err := validateIndRefEntry(xRefTable, dict, dictName, "Annotation", REQUIRED, types.V10)
	if err != nil || indRef == nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || d == nil {
		return errors.New("validateMovieActionDict: missing required entry \"T\" or \"Annotation\"")
	}

	_, err = validateNameEntry(xRefTable, d, "annotDict", "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "Movie" })

	return err
}

func validateHideActionDictEntryT(xRefTable *types.XRefTable, obj interface{}) error {

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		// Ensure UTF16 correctness.
		_, err := types.StringLiteralToString(obj.Value())
		if err != nil {
			return err
		}

	case types.PDFDict:
		// annotDict,  Check for required name Subtype
		_, err := validateNameEntry(xRefTable, &obj, "annotDict", "Subtype", REQUIRED, types.V10, nil)
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
		return errors.Errorf("validateHideActionDict: invalid entry \"T\"")

	}

	return nil
}

func validateHideActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.10

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// T, required, dict, text string or array
	obj, found := dict.Find("T")
	if !found || obj == nil {
		return errors.New("validateHideActionDict: missing required entry \"T\"")
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	err = validateHideActionDictEntryT(xRefTable, obj)
	if err != nil {
		return err
	}

	// H, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V10, nil)

	return err
}

func validateNamedActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.11

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "N", REQUIRED, types.V10, validateNamedAction)

	return err
}

func validateSubmitFormActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.7.5.2

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// F, required, URL specification
	_, err = validateURLSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	arr, err := validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
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

	return err
}

func validateResetFormActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.7.5.3

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	arr, err := validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
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

	return err
}

func validateImportDataActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.7.5.4

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11)

	return err
}

func validateJavaScript(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool) error {

	logInfoValidate.Println("*** validateJavaScript begin ***")

	obj, found := dict.Find("JS")
	if !found || obj == nil {
		if required {
			return errors.New("validateJavaScript: required entry \"JS\" missing")
		}
		return nil
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch s := obj.(type) {

	case types.PDFStringLiteral:
		// Ensure UTF16 correctness.
		_, err = types.StringLiteralToString(s.Value())
		if err != nil {
			return err
		}

	case types.PDFHexLiteral:
		// Ensure UTF16 correctness.
		_, err = types.HexLiteralToString(s.Value())
		if err != nil {
			return err
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		return errors.Errorf("validateJavaScript: invalid type\n")

	}

	return nil
}

func validateJavaScriptActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.16

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// JS, required, text string or stream
	return validateJavaScript(xRefTable, dict, dictName, "JS", REQUIRED)
}

func validateSetOCGStateActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.12

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// State, required, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "State", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// PreserveRB, optinal, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "PreserveRB", OPTIONAL, types.V10, nil)

	return err
}

func validateRenditionActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.13

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// OP or JS need to be present.

	// OP, integer
	op, err := validateIntegerEntry(xRefTable, dict, dictName, "OP", OPTIONAL, sinceVersion, func(i int) bool { return 0 <= i && i <= 4 })
	if err != nil {
		return err
	}

	// JS, text string or stream
	err = validateJavaScript(xRefTable, dict, dictName, "JS", op == nil)
	if err != nil {
		return err
	}

	// R, required for OP 0 and 4, rendition object dict
	required := func(op *types.PDFInteger) bool {
		if op == nil {
			return false
		}
		v := op.Value()
		return v == 0 || v == 4
	}(op)

	d, err := validateDictEntry(xRefTable, dict, dictName, "R", required, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateRenditionDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	// AN, required for any OP 0..4, indRef of screen annotation dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "AN", op != nil, types.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, types.V10, func(s string) bool { return s == "Screen" })
		if err != nil {
			return err
		}
	}

	return nil
}

func validateTransActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.14

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// Trans, required, transitionDict
	d, err := validateDictEntry(xRefTable, dict, dictName, "Trans", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	return validateTransitionDict(xRefTable, d)
}

func validateGoTo3DViewActionDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error {

	// see 12.6.4.15

	logInfoValidate.Println("*** validateGoTo3DViewActionDict begin ***")

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	// TA, required, target annotation
	d, err := validateDictEntry(xRefTable, dict, dictName, "TA", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateAnnotationDict(xRefTable, d)
	if err != nil {
		return err
	}

	// V, required, the view to use: 3DViewDict or integer or text string or name
	// TODO Validation.
	_, err = validateEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion)

	return err
}

func validateActionDictCore(xRefTable *types.XRefTable, n *types.PDFName, dict *types.PDFDict) error {

	for k, v := range map[string]struct {
		validate     func(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) error
		sinceVersion types.PDFVersion
	}{
		"GoTo":        {validateGoToActionDict, types.V10},
		"GoToR":       {validateGoToRActionDict, types.V10},
		"GoToE":       {validateGoToEActionDict, types.V16},
		"Launch":      {validateLaunchActionDict, types.V10},
		"Thread":      {validateThreadActionDict, types.V10},
		"URI":         {validateURIActionDict, types.V10},
		"Sound":       {validateSoundActionDict, types.V12},
		"Movie":       {validateMovieActionDict, types.V12},
		"Hide":        {validateHideActionDict, types.V12},
		"Named":       {validateNamedActionDict, types.V12},
		"SubmitForm":  {validateSubmitFormActionDict, types.V10},
		"ResetForm":   {validateResetFormActionDict, types.V12},
		"ImportData":  {validateImportDataActionDict, types.V12},
		"JavaScript":  {validateJavaScriptActionDict, types.V13},
		"SetOCGState": {validateSetOCGStateActionDict, types.V15},
		"Rendition":   {validateRenditionActionDict, types.V15},
		"Trans":       {validateTransActionDict, types.V15},
		"GoTo3DView":  {validateGoTo3DViewActionDict, types.V16},
	} {
		if n.Value() == k {
			return v.validate(xRefTable, dict, k, v.sinceVersion)
		}
	}

	return errors.Errorf("validateActionDictCore: unsupported action type: %s\n", *n)
}

func validateActionDict(xRefTable *types.XRefTable, obj interface{}) error {

	dictName := "actionDict"

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Action" })
	if err != nil {
		return err
	}

	// S, required, name, action Type
	s, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	err = validateActionDictCore(xRefTable, s, dict)
	if err != nil {
		return err
	}

	if obj, ok := dict.Find("Next"); ok {

		// either optional action dict
		dict, err = xRefTable.DereferenceDict(obj)
		if err == nil {
			err = validateActionDict(xRefTable, *dict)
			if err != nil {
				return err
			}
			logInfoValidate.Println("*** validateActionDict end ***")
			return nil
		}

		// or optional array of action dicts
		arr, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		for _, v := range *arr {
			err = validateActionDict(xRefTable, v)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateRootAdditionalActions(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	return validateAdditionalActions(xRefTable, rootDict, "rootDict", "AA", required, sinceVersion, "root")
}

func validateAdditionalActions(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion, source string) error {

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateAdditionalActions: dict=%s required entry=%s missing", dictName, entryName)
		}
		logInfoValidate.Printf("validateAdditionalActions end: dict=%s optional entry %s not found or nil\n", dictName, entryName)
		return nil
	}

	d, ok := obj.(types.PDFDict)
	if !ok {
		return errors.Errorf("validateAdditionalActions: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateAdditionalActions: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
	}

	for k, v := range d.Dict {

		if !validateAdditionalAction(k, source) {
			return errors.Errorf("validateAdditionalActions: action %s not allowed for source %s", k, source)
		}

		err := validateActionDict(xRefTable, v)
		if err != nil {
			return err
		}

	}

	return nil
}

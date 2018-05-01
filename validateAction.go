package pdfcpu

import (
	"github.com/pkg/errors"
)

func validateGoToActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.2 Go-To Actions

	// D, required, name, byte string or array
	return validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, V10)
}

func validateGoToRActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.3 Remote Go-To Actions

	// F, required, file specification
	_, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, V10)
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, V12, nil)

	return err
}

func validateTargetDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

	// table 202

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "targetDict"

	// R, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "R", REQUIRED, V10, func(s string) bool { return s == "P" || s == "C" })
	if err != nil {
		return err
	}

	// N, optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// P, optional, integer or byte string
	err = validateIntOrStringEntry(xRefTable, d, dictName, "P", OPTIONAL, V10)
	if err != nil {
		return err
	}

	// A, optional, integer or text string
	err = validateIntOrStringEntry(xRefTable, d, dictName, "A", OPTIONAL, V10)
	if err != nil {
		return err
	}

	// T, optional, target dict
	return validateTargetDictEntry(xRefTable, d, dictName, "T", OPTIONAL, V10)
}

func validateGoToEActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.4 Embedded Go-To Actions

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, dict, dictName, "D", REQUIRED, V10)
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "NewWindow", OPTIONAL, V12, nil)
	if err != nil {
		return err
	}

	// T, required unless entry F is present, target dict
	return validateTargetDictEntry(xRefTable, dict, dictName, "T", f == nil, V10)
}

func validateWinDict(xRefTable *XRefTable, dict *PDFDict) error {

	// see table 204

	dictName := "winDict"

	// F, required, byte string
	_, err := validateStringEntry(xRefTable, dict, dictName, "F", REQUIRED, V10, nil)
	if err != nil {
		return err
	}

	// D, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "D", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// O, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "O", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// P, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "P", OPTIONAL, V10, nil)

	return err
}

func validateLaunchActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.5

	// F, optional, file specification
	_, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, V11)
	if err != nil {
		return err
	}

	// Win, optional, dict
	winDict, err := validateDictEntry(xRefTable, dict, dictName, "Win", OPTIONAL, V10, nil)
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

func validateDestinationThreadEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

	// The destination thread (table 205)

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj.(type) {

	case PDFDict, PDFStringLiteral, PDFInteger:
		// an indRef to a thread dictionary
		// or an index of the thread within the roots Threads array
		// or the title of the thread as specified in its thread info dict

	default:
		return errors.Errorf("validateDestinationThreadEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	return nil
}

func validateDestinationBeadEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

	// The bead in the destination thread (table 205)

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj.(type) {

	case PDFDict, PDFInteger:
		// an indRef to a bead dictionary of a thread in the current file
		// or an index of the thread within its thread

	default:
		return errors.Errorf("validateDestinationBeadEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	return nil
}

func validateThreadActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	//see 12.6.4.6

	// F, optional, file specification
	_, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, V11)
	if err != nil {
		return err
	}

	// D, required, indRef to thread dict, integer or text string.
	err = validateDestinationThreadEntry(xRefTable, dict, dictName, "D", REQUIRED, V10)
	if err != nil {
		return err
	}

	// B, optional, indRef to bead dict or integer.
	return validateDestinationBeadEntry(xRefTable, dict, dictName, "B", OPTIONAL, V10)
}

func validateURIActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.7

	// URI, required, string
	_, err := validateStringEntry(xRefTable, dict, dictName, "URI", REQUIRED, V10, nil)
	if err != nil {
		return err
	}

	// IsMap, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "IsMap", OPTIONAL, V10, nil)

	return err
}

func validateSoundDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

	sd, err := validateStreamDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	dictName = "soundDict"
	dict = &sd.PDFDict

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, V10, func(s string) bool { return s == "Sound" })
	if err != nil {
		return err
	}

	// R, required, number - sampling rate
	_, err = validateNumberEntry(xRefTable, dict, dictName, "R", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// C, required, integer - # of sound channels
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "C", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// B, required, integer - bits per sample value per channel
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "B", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// E, optional, name - encoding format
	validateSampleDataEncoding := func(s string) bool {
		return memberOf(s, []string{"Raw", "Signed", "muLaw", "ALaw"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "E", OPTIONAL, V10, validateSampleDataEncoding)

	return err
}

func validateSoundActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.8

	// Sound, required, stream dict
	err := validateSoundDictEntry(xRefTable, dict, dictName, "Sound", REQUIRED, V10)
	if err != nil {
		return err
	}

	// Volume, optional, number: -1.0 .. +1.0
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Volume", OPTIONAL, V10, func(f float64) bool { return -1.0 <= f && f <= 1.0 })
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Synchronous", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Repeat", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// Mix, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Mix", OPTIONAL, V10, nil)

	return err
}

func validateMovieStartOrDurationEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch o := obj.(type) {

	case PDFInteger, PDFStringLiteral:
		// no further processing

	case PDFArray:
		if len(o) != 2 {
			return errors.New("validateMovieStartOrDurationEntry: array length <> 2")
		}
	}

	return nil
}

func validateMovieActivationDict(xRefTable *XRefTable, dict *PDFDict) error {

	dictName := "movieActivationDict"

	// Start, optional
	err := validateMovieStartOrDurationEntry(xRefTable, dict, dictName, "Start", OPTIONAL, V10)
	if err != nil {
		return err
	}

	// Duration, optional
	err = validateMovieStartOrDurationEntry(xRefTable, dict, dictName, "Duration", OPTIONAL, V10)
	if err != nil {
		return err
	}

	// Rate, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Rate", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// Volume, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Volume", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// ShowControls, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "ShowControls", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// Mode, optional, name
	validatePlayMode := func(s string) bool {
		return memberOf(s, []string{"Once", "Open", "Repeat", "Palindrome"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Mode", OPTIONAL, V10, validatePlayMode)
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Synchronous", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	// FWScale, optional, array of 2 positive integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "FWScale", OPTIONAL, V10, func(a PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// FWPosition, optional, array of 2 numbers [0.0 .. 1.0]
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "FWPosition", OPTIONAL, V10, func(a PDFArray) bool { return len(a) == 2 })

	return err
}

func validateMovieActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.9

	// is a movie activation dict
	err := validateMovieActivationDict(xRefTable, dict)
	if err != nil {
		return err
	}

	// Needs either Annotation or T entry but not both.

	// T, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, V10, nil)
	if err == nil {
		return nil
	}

	// Annotation, indRef of movie annotation dict
	indRef, err := validateIndRefEntry(xRefTable, dict, dictName, "Annotation", REQUIRED, V10)
	if err != nil || indRef == nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(*indRef)
	if err != nil || d == nil {
		return errors.New("validateMovieActionDict: missing required entry \"T\" or \"Annotation\"")
	}

	_, err = validateNameEntry(xRefTable, d, "annotDict", "Subtype", REQUIRED, V10, func(s string) bool { return s == "Movie" })

	return err
}

func validateHideActionDictEntryT(xRefTable *XRefTable, obj PDFObject) error {

	switch obj := obj.(type) {

	case PDFStringLiteral:
		// Ensure UTF16 correctness.
		_, err := StringLiteralToString(obj.Value())
		if err != nil {
			return err
		}

	case PDFDict:
		// annotDict,  Check for required name Subtype
		_, err := validateNameEntry(xRefTable, &obj, "annotDict", "Subtype", REQUIRED, V10, nil)
		if err != nil {
			return err
		}

	case PDFArray:
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

			case PDFStringLiteral:
				// Ensure UTF16 correctness.
				_, err = StringLiteralToString(o.Value())
				if err != nil {
					return err
				}

			case PDFDict:
				// annotDict,  Check for required name Subtype
				_, err = validateNameEntry(xRefTable, &o, "annotDict", "Subtype", REQUIRED, V10, nil)
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

func validateHideActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.10

	// T, required, dict, text string or array
	obj, found := dict.Find("T")
	if !found || obj == nil {
		return errors.New("validateHideActionDict: missing required entry \"T\"")
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	err = validateHideActionDictEntryT(xRefTable, obj)
	if err != nil {
		return err
	}

	// H, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "H", OPTIONAL, V10, nil)

	return err
}

func validateNamedActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.11

	validate := func(s string) bool {

		if memberOf(s, []string{"NextPage", "PrevPage", "FirstPage", "Lastpage"}) {
			return true
		}

		// Some known non standard named actions
		if memberOf(s, []string{"GoToPage", "GoBack", "GoForward", "Find"}) {
			return true
		}

		return false
	}

	_, err := validateNameEntry(xRefTable, dict, dictName, "N", REQUIRED, V10, validate)

	return err
}

func validateSubmitFormActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.7.5.2

	// F, required, URL specification
	_, err := validateURLSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, V10)
	if err != nil {
		return err
	}

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	arr, err := validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	if arr != nil {
		for _, v := range *arr {
			switch v.(type) {
			case PDFStringLiteral, PDFIndirectRef:
				// no further processing

			default:
				return errors.New("validateSubmitFormActionDict: unknown Fields entry")
			}
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", OPTIONAL, V10, nil)

	return err
}

func validateResetFormActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.7.5.3

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	arr, err := validateArrayEntry(xRefTable, dict, dictName, "Fields", OPTIONAL, V10, nil)
	if err != nil {
		return err
	}

	if arr != nil {
		for _, v := range *arr {
			switch v.(type) {
			case PDFStringLiteral, PDFIndirectRef:
				// no further processing

			default:
				return errors.New("validateResetFormActionDict: unknown Fields entry")
			}
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Flags", OPTIONAL, V10, nil)

	return err
}

func validateImportDataActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.7.5.4

	// F, required, file specification
	_, err := validateFileSpecEntry(xRefTable, dict, dictName, "F", OPTIONAL, V11)

	return err
}

func validateJavaScript(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, V13)
	if err != nil || obj == nil {
		return err
	}

	switch s := obj.(type) {

	case PDFStringLiteral:
		// Ensure UTF16 correctness.
		_, err = StringLiteralToString(s.Value())

	case PDFHexLiteral:
		// Ensure UTF16 correctness.
		_, err = HexLiteralToString(s.Value())

	case PDFStreamDict:
		// no further processing

	default:
		err = errors.Errorf("validateJavaScript: invalid type\n")

	}

	return err
}

func validateJavaScriptActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.16

	// JS, required, text string or stream
	return validateJavaScript(xRefTable, dict, dictName, "JS", REQUIRED)
}

func validateSetOCGStateActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.12

	// State, required, array
	_, err := validateArrayEntry(xRefTable, dict, dictName, "State", REQUIRED, V10, nil)
	if err != nil {
		return err
	}

	// PreserveRB, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "PreserveRB", OPTIONAL, V10, nil)

	return err
}

func validateRenditionActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.13

	// OP or JS need to be present.

	// OP, integer
	op, err := validateIntegerEntry(xRefTable, dict, dictName, "OP", OPTIONAL, V15, func(i int) bool { return 0 <= i && i <= 4 })
	if err != nil {
		return err
	}

	// JS, text string or stream
	err = validateJavaScript(xRefTable, dict, dictName, "JS", op == nil)
	if err != nil {
		return err
	}

	// R, required for OP 0 and 4, rendition object dict
	required := func(op *PDFInteger) bool {
		if op == nil {
			return false
		}
		v := op.Value()
		return v == 0 || v == 4
	}(op)

	d, err := validateDictEntry(xRefTable, dict, dictName, "R", required, V15, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateRenditionDict(xRefTable, d, V15)
		if err != nil {
			return err
		}
	}

	// AN, required for any OP 0..4, indRef of screen annotation dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "AN", op != nil, V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, V10, func(s string) bool { return s == "Screen" })
		if err != nil {
			return err
		}
	}

	return nil
}

func validateTransActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.14

	// Trans, required, transitionDict
	d, err := validateDictEntry(xRefTable, dict, dictName, "Trans", REQUIRED, V10, nil)
	if err != nil {
		return err
	}

	return validateTransitionDict(xRefTable, d)
}

func validateGoTo3DViewActionDict(xRefTable *XRefTable, dict *PDFDict, dictName string) error {

	// see 12.6.4.15

	// TA, required, target annotation
	d, err := validateDictEntry(xRefTable, dict, dictName, "TA", REQUIRED, V16, nil)
	if err != nil {
		return err
	}

	_, err = validateAnnotationDict(xRefTable, d)
	if err != nil {
		return err
	}

	// V, required, the view to use: 3DViewDict or integer or text string or name
	// TODO Validation.
	_, err = validateEntry(xRefTable, dict, dictName, "V", REQUIRED, V16)

	return err
}

func validateActionDictCore(xRefTable *XRefTable, n *PDFName, dict *PDFDict) error {

	for k, v := range map[string]struct {
		validate     func(xRefTable *XRefTable, dict *PDFDict, dictName string) error
		sinceVersion PDFVersion
	}{
		"GoTo":        {validateGoToActionDict, V10},
		"GoToR":       {validateGoToRActionDict, V10},
		"GoToE":       {validateGoToEActionDict, V16},
		"Launch":      {validateLaunchActionDict, V10},
		"Thread":      {validateThreadActionDict, V10},
		"URI":         {validateURIActionDict, V10},
		"Sound":       {validateSoundActionDict, V12},
		"Movie":       {validateMovieActionDict, V12},
		"Hide":        {validateHideActionDict, V12},
		"Named":       {validateNamedActionDict, V12},
		"SubmitForm":  {validateSubmitFormActionDict, V10},
		"ResetForm":   {validateResetFormActionDict, V12},
		"ImportData":  {validateImportDataActionDict, V12},
		"JavaScript":  {validateJavaScriptActionDict, V13},
		"SetOCGState": {validateSetOCGStateActionDict, V15},
		"Rendition":   {validateRenditionActionDict, V15},
		"Trans":       {validateTransActionDict, V15},
		"GoTo3DView":  {validateGoTo3DViewActionDict, V16},
	} {
		if n.Value() == k {

			err := xRefTable.ValidateVersion(k, v.sinceVersion)
			if err != nil {
				return err
			}

			return v.validate(xRefTable, dict, k)
		}
	}

	return errors.Errorf("validateActionDictCore: unsupported action type: %s\n", *n)
}

func validateActionDict(xRefTable *XRefTable, dict *PDFDict) error {

	dictName := "actionDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, V10, func(s string) bool { return s == "Action" })
	if err != nil {
		return err
	}

	// S, required, name, action Type
	s, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, V10, nil)
	if err != nil {
		return err
	}

	err = validateActionDictCore(xRefTable, s, dict)
	if err != nil {
		return err
	}

	if obj, ok := dict.Find("Next"); ok {

		// either optional action dict
		d, err := xRefTable.DereferenceDict(obj)
		if err == nil {
			err = validateActionDict(xRefTable, d)
			if err != nil {
				return err
			}
			return nil
		}

		// or optional array of action dicts
		arr, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		for _, v := range *arr {

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			if d == nil {
				continue
			}

			err = validateActionDict(xRefTable, d)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateRootAdditionalActions(xRefTable *XRefTable, rootDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	return validateAdditionalActions(xRefTable, rootDict, "rootDict", "AA", required, sinceVersion, "root")
}

func validateAdditionalActions(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, source string) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	validateAdditionalAction := func(s, source string) bool {

		switch source {

		case "root":
			if memberOf(s, []string{"WC", "WS", "DS", "WP", "DP"}) {
				return true
			}

		case "page":
			if memberOf(s, []string{"O", "C"}) {
				return true
			}

		case "fieldOrAnnot":
			// A terminal acro field may be merged with a widget annotation.
			fieldOptions := []string{"K", "F", "V", "C"}
			annotOptions := []string{"E", "X", "D", "U", "Fo", "Bl", "PO", "PC", "PV", "Pl"}
			options := append(fieldOptions, annotOptions...)
			if memberOf(s, options) {
				return true
			}

		}

		return false
	}

	for k, v := range d.Dict {

		if !validateAdditionalAction(k, source) {
			return errors.Errorf("validateAdditionalActions: action %s not allowed for source %s", k, source)
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateActionDict(xRefTable, d)
		if err != nil {
			return err
		}

	}

	return nil
}

/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	"strings"

	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateGoToActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.2 Go-To Actions
	required := REQUIRED
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = OPTIONAL
	}

	// D, required, name, byte string or array
	return validateDestinationEntry(xRefTable, d, dictName, "D", required, pdf.V10)
}

func validateGoToRActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.3 Remote Go-To Actions

	// F, required, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, d, dictName, "D", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, d, dictName, "NewWindow", OPTIONAL, pdf.V12, nil)

	return err
}

func validateTargetDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// table 202

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "targetDict"

	// R, required, name
	_, err = validateNameEntry(xRefTable, d1, dictName, "R", REQUIRED, pdf.V10, func(s string) bool { return s == "P" || s == "C" })
	if err != nil {
		return err
	}

	// N, optional, byte string
	_, err = validateStringEntry(xRefTable, d1, dictName, "N", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, integer or byte string
	err = validateIntOrStringEntry(xRefTable, d1, dictName, "P", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// A, optional, integer or text string
	err = validateIntOrStringEntry(xRefTable, d1, dictName, "A", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// T, optional, target dict
	return validateTargetDictEntry(xRefTable, d1, dictName, "T", OPTIONAL, pdf.V10)
}

func validateGoToEActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.4 Embedded Go-To Actions

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateDestinationEntry(xRefTable, d, dictName, "D", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, d, dictName, "NewWindow", OPTIONAL, pdf.V12, nil)
	if err != nil {
		return err
	}

	// T, required unless entry F is present, target dict
	return validateTargetDictEntry(xRefTable, d, dictName, "T", f == nil, pdf.V10)
}

func validateWinDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see table 204

	dictName := "winDict"

	// F, required, byte string
	_, err := validateStringEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// D, optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "D", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// O, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "O", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "P", OPTIONAL, pdf.V10, nil)

	return err
}

func validateLaunchActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.5

	// F, optional, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V11)
	if err != nil {
		return err
	}

	// Win, optional, dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "Win", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateWinDict(xRefTable, d1)
	}

	// Mac, optional, undefined dict

	// Unix, optional, undefined dict

	return err
}

func validateDestinationThreadEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// The destination thread (table 205)

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case pdf.Dict, pdf.StringLiteral, pdf.Integer:
		// an indRef to a thread dictionary
		// or an index of the thread within the roots Threads array
		// or the title of the thread as specified in its thread info dict

	default:
		return errors.Errorf("validateDestinationThreadEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	return nil
}

func validateDestinationBeadEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// The bead in the destination thread (table 205)

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case pdf.Dict, pdf.Integer:
		// an indRef to a bead dictionary of a thread in the current file
		// or an index of the thread within its thread

	default:
		return errors.Errorf("validateDestinationBeadEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	return nil
}

func validateThreadActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	//see 12.6.4.6

	// F, optional, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V11)
	if err != nil {
		return err
	}

	// D, required, indRef to thread dict, integer or text string.
	err = validateDestinationThreadEntry(xRefTable, d, dictName, "D", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// B, optional, indRef to bead dict or integer.
	return validateDestinationBeadEntry(xRefTable, d, dictName, "B", OPTIONAL, pdf.V10)
}

func hasURIForChecking(xRefTable *pdf.XRefTable, s string) bool {
	for _, links := range xRefTable.URIs {
		for uri := range links {
			if uri == s {
				return true
			}
		}
	}
	return false
}

func validateURIActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.7

	// URI, required, string
	uri, err := validateStringEntry(xRefTable, d, dictName, "URI", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Record URIs for link checking.
	if xRefTable.ValidateLinks && uri != nil &&
		strings.HasPrefix(*uri, "http") && !hasURIForChecking(xRefTable, *uri) {
		if len(xRefTable.URIs[xRefTable.CurPage]) == 0 {
			xRefTable.URIs[xRefTable.CurPage] = map[string]string{}
		}
		xRefTable.URIs[xRefTable.CurPage][*uri] = ""
	}

	// IsMap, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "IsMap", OPTIONAL, pdf.V10, nil)

	return err
}

func validateSoundDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	sd, err := validateStreamDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	dictName = "soundDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "Sound" })
	if err != nil {
		return err
	}

	// R, required, number - sampling rate
	_, err = validateNumberEntry(xRefTable, sd.Dict, dictName, "R", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// C, required, integer - # of sound channels
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "C", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// B, required, integer - bits per sample value per channel
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "B", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// E, optional, name - encoding format
	validateSampleDataEncoding := func(s string) bool {
		return pdf.MemberOf(s, []string{"Raw", "Signed", "muLaw", "ALaw"})
	}
	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "E", OPTIONAL, pdf.V10, validateSampleDataEncoding)

	return err
}

func validateSoundActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.8

	// Sound, required, stream dict
	err := validateSoundDictEntry(xRefTable, d, dictName, "Sound", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// Volume, optional, number: -1.0 .. +1.0
	_, err = validateNumberEntry(xRefTable, d, dictName, "Volume", OPTIONAL, pdf.V10, func(f float64) bool { return -1.0 <= f && f <= 1.0 })
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Synchronous", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Repeat", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Mix, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Mix", OPTIONAL, pdf.V10, nil)

	return err
}

func validateMovieStartOrDurationEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Integer, pdf.StringLiteral:
		// no further processing

	case pdf.Array:
		if len(o) != 2 {
			return errors.New("pdfcpu: validateMovieStartOrDurationEntry: array length <> 2")
		}
	}

	return nil
}

func validateMovieActivationDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "movieActivationDict"

	// Start, optional
	err := validateMovieStartOrDurationEntry(xRefTable, d, dictName, "Start", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// Duration, optional
	err = validateMovieStartOrDurationEntry(xRefTable, d, dictName, "Duration", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// Rate, optional, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "Rate", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Volume, optional, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "Volume", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// ShowControls, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "ShowControls", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Mode, optional, name
	validatePlayMode := func(s string) bool {
		return pdf.MemberOf(s, []string{"Once", "Open", "Repeat", "Palindrome"})
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "Mode", OPTIONAL, pdf.V10, validatePlayMode)
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Synchronous", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FWScale, optional, array of 2 positive integers
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "FWScale", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// FWPosition, optional, array of 2 numbers [0.0 .. 1.0]
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "FWPosition", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })

	return err
}

func validateMovieActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.9

	// is a movie activation dict
	err := validateMovieActivationDict(xRefTable, d)
	if err != nil {
		return err
	}

	// Needs either Annotation or T entry but not both.

	// T, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, pdf.V10, nil)
	if err == nil {
		return nil
	}

	// Annotation, indRef of movie annotation dict
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "Annotation", REQUIRED, pdf.V10)
	if err != nil || ir == nil {
		return err
	}

	d, err = xRefTable.DereferenceDict(*ir)
	if err != nil || d == nil {
		return errors.New("pdfcpu: validateMovieActionDict: missing required entry \"T\" or \"Annotation\"")
	}

	_, err = validateNameEntry(xRefTable, d, "annotDict", "Subtype", REQUIRED, pdf.V10, func(s string) bool { return s == "Movie" })

	return err
}

func validateHideActionDictEntryT(xRefTable *pdf.XRefTable, o pdf.Object) error {

	switch o := o.(type) {

	case pdf.StringLiteral:
		// Ensure UTF16 correctness.
		_, err := pdf.StringLiteralToString(o)
		if err != nil {
			return err
		}

	case pdf.Dict:
		// annotDict,  Check for required name Subtype
		_, err := validateNameEntry(xRefTable, o, "annotDict", "Subtype", REQUIRED, pdf.V10, nil)
		if err != nil {
			return err
		}

	case pdf.Array:
		// mixed array of annotationDict indRefs and strings
		for _, v := range o {

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			switch o := o.(type) {

			case pdf.StringLiteral:
				// Ensure UTF16 correctness.
				_, err = pdf.StringLiteralToString(o)
				if err != nil {
					return err
				}

			case pdf.Dict:
				// annotDict,  Check for required name Subtype
				_, err = validateNameEntry(xRefTable, o, "annotDict", "Subtype", REQUIRED, pdf.V10, nil)
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

func validateHideActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.10

	// T, required, dict, text string or array
	o, found := d.Find("T")
	if !found || o == nil {
		return errors.New("pdfcpu: validateHideActionDict: missing required entry \"T\"")
	}

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	err = validateHideActionDictEntryT(xRefTable, o)
	if err != nil {
		return err
	}

	// H, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "H", OPTIONAL, pdf.V10, nil)

	return err
}

func validateNamedActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.11

	validate := func(s string) bool {

		if pdf.MemberOf(s, []string{"NextPage", "PrevPage", "FirstPage", "Lastpage"}) {
			return true
		}

		// Some known non standard named actions
		if pdf.MemberOf(s, []string{"GoToPage", "GoBack", "GoForward", "Find", "Print", "Quit", "FullScreen"}) {
			return true
		}

		return false
	}

	_, err := validateNameEntry(xRefTable, d, dictName, "N", REQUIRED, pdf.V10, validate)

	return err
}

func validateSubmitFormActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.7.5.2

	// F, required, URL specification
	_, err := validateURLSpecEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	a, err := validateArrayEntry(xRefTable, d, dictName, "Fields", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	if a != nil {
		for _, v := range a {
			switch v.(type) {
			case pdf.StringLiteral, pdf.IndirectRef:
				// no further processing

			default:
				return errors.New("pdfcpu: validateSubmitFormActionDict: unknown Fields entry")
			}
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Flags", OPTIONAL, pdf.V10, nil)

	return err
}

func validateResetFormActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.7.5.3

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	a, err := validateArrayEntry(xRefTable, d, dictName, "Fields", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	if a != nil {
		for _, v := range a {
			switch v.(type) {
			case pdf.StringLiteral, pdf.IndirectRef:
				// no further processing

			default:
				return errors.New("pdfcpu: validateResetFormActionDict: unknown Fields entry")
			}
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Flags", OPTIONAL, pdf.V10, nil)

	return err
}

func validateImportDataActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.7.5.4

	// F, required, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V11)

	return err
}

func validateJavaScript(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, pdf.V13)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.StringLiteral:
		// Ensure UTF16 correctness.
		_, err = pdf.StringLiteralToString(o)

	case pdf.HexLiteral:
		// Ensure UTF16 correctness.
		_, err = pdf.HexLiteralToString(o)

	case pdf.StreamDict:
		// no further processing

	default:
		err = errors.Errorf("validateJavaScript: invalid type\n")

	}

	return err
}

func validateJavaScriptActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.16

	// JS, required, text string or stream
	return validateJavaScript(xRefTable, d, dictName, "JS", REQUIRED)
}

func validateSetOCGStateActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.12

	// State, required, array
	_, err := validateArrayEntry(xRefTable, d, dictName, "State", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// PreserveRB, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "PreserveRB", OPTIONAL, pdf.V10, nil)

	return err
}

func validateRenditionActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.13

	// OP or JS need to be present.

	// OP, integer
	op, err := validateIntegerEntry(xRefTable, d, dictName, "OP", OPTIONAL, pdf.V15, func(i int) bool { return 0 <= i && i <= 4 })
	if err != nil {
		return err
	}

	// JS, text string or stream
	err = validateJavaScript(xRefTable, d, dictName, "JS", op == nil)
	if err != nil {
		return err
	}

	// R, required for OP 0 and 4, rendition object dict
	required := func(op *pdf.Integer) bool {
		if op == nil {
			return false
		}
		v := op.Value()
		return v == 0 || v == 4
	}(op)

	d1, err := validateDictEntry(xRefTable, d, dictName, "R", required, pdf.V15, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateRenditionDict(xRefTable, d1, pdf.V15)
		if err != nil {
			return err
		}
	}

	// AN, required for any OP 0..4, indRef of screen annotation dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "AN", op != nil, pdf.V10, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		_, err = validateNameEntry(xRefTable, d1, dictName, "Subtype", REQUIRED, pdf.V10, func(s string) bool { return s == "Screen" })
		if err != nil {
			return err
		}
	}

	return nil
}

func validateTransActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.14

	// Trans, required, transitionDict
	d1, err := validateDictEntry(xRefTable, d, dictName, "Trans", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	return validateTransitionDict(xRefTable, d1)
}

func validateGoTo3DViewActionDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.6.4.15

	// TA, required, target annotation
	d1, err := validateDictEntry(xRefTable, d, dictName, "TA", REQUIRED, pdf.V16, nil)
	if err != nil {
		return err
	}

	_, err = validateAnnotationDict(xRefTable, d1)
	if err != nil {
		return err
	}

	// V, required, the view to use: 3DViewDict or integer or text string or name
	// TODO Validation.
	_, err = validateEntry(xRefTable, d, dictName, "V", REQUIRED, pdf.V16)

	return err
}

func validateActionDictCore(xRefTable *pdf.XRefTable, n *pdf.Name, d pdf.Dict) error {

	for k, v := range map[string]struct {
		validate     func(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error
		sinceVersion pdf.Version
	}{
		"GoTo":        {validateGoToActionDict, pdf.V10},
		"GoToR":       {validateGoToRActionDict, pdf.V10},
		"GoToE":       {validateGoToEActionDict, pdf.V16},
		"Launch":      {validateLaunchActionDict, pdf.V10},
		"Thread":      {validateThreadActionDict, pdf.V10},
		"URI":         {validateURIActionDict, pdf.V10},
		"Sound":       {validateSoundActionDict, pdf.V12},
		"Movie":       {validateMovieActionDict, pdf.V12},
		"Hide":        {validateHideActionDict, pdf.V12},
		"Named":       {validateNamedActionDict, pdf.V12},
		"SubmitForm":  {validateSubmitFormActionDict, pdf.V10},
		"ResetForm":   {validateResetFormActionDict, pdf.V12},
		"ImportData":  {validateImportDataActionDict, pdf.V12},
		"JavaScript":  {validateJavaScriptActionDict, pdf.V13},
		"SetOCGState": {validateSetOCGStateActionDict, pdf.V15},
		"Rendition":   {validateRenditionActionDict, pdf.V15},
		"Trans":       {validateTransActionDict, pdf.V15},
		"GoTo3DView":  {validateGoTo3DViewActionDict, pdf.V16},
	} {
		if n.Value() == k {

			err := xRefTable.ValidateVersion(k, v.sinceVersion)
			if err != nil {
				return err
			}

			return v.validate(xRefTable, d, k)
		}
	}

	return errors.Errorf("validateActionDictCore: unsupported action type: %s\n", *n)
}

func validateActionDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "actionDict"

	// Type, optional, name
	allowedTypes := []string{"Action"}
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		allowedTypes = []string{"A", "Action"}
	}
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return pdf.MemberOf(s, allowedTypes) })
	if err != nil {
		return err
	}

	// S, required, name, action Type
	s, err := validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	err = validateActionDictCore(xRefTable, s, d)
	if err != nil {
		return err
	}

	if o, ok := d.Find("Next"); ok {

		// either optional action dict
		d, err := xRefTable.DereferenceDict(o)
		if err == nil {
			return validateActionDict(xRefTable, d)
		}

		// or optional array of action dicts
		a, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return err
		}

		for _, v := range a {

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

func validateRootAdditionalActions(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	return validateAdditionalActions(xRefTable, rootDict, "rootDict", "AA", required, sinceVersion, "root")
}

func validateAdditionalActions(xRefTable *pdf.XRefTable, dict pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, source string) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	validateAdditionalAction := func(s, source string) bool {

		switch source {

		case "root":
			if pdf.MemberOf(s, []string{"WC", "WS", "DS", "WP", "DP"}) {
				return true
			}

		case "page":
			if pdf.MemberOf(s, []string{"O", "C"}) {
				return true
			}

		case "fieldOrAnnot":
			// A terminal acro field may be merged with a widget annotation.
			fieldOptions := []string{"K", "F", "V", "C"}
			annotOptions := []string{"E", "X", "D", "U", "Fo", "Bl", "PO", "PC", "PV", "Pl"}
			options := append(fieldOptions, annotOptions...)
			if pdf.MemberOf(s, options) {
				return true
			}

		}

		return false
	}

	for k, v := range d {

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

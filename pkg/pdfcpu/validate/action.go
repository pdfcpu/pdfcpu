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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateGoToActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.2 Go-To Actions
	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}

	// D, required, name, byte string or array
	return validateActionDestinationEntry(xRefTable, d, dictName, "D", required, model.V10)
}

func validateGoToRActionDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	// see 12.6.4.3 Remote Go-To Actions

	// F, required, file specification
	f, err := validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateRemoteActionDestinationEntry(xRefTable, d, ownerObjNr, dictName, "D")
	if err != nil {
		return err
	}

	// NewWindow, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "NewWindow", OPTIONAL, model.V12, nil)
	if err != nil {
		return err
	}
	collectFileSpecificationTarget(xRefTable, f, linkTargetFile, linkSourceRemoteGoTo)
	return nil
}

type targetTraversal map[int]bool

func (v targetTraversal) enter(objNr int) error {
	if objNr <= 0 {
		return nil
	}
	if v[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, model.ErrTargetCycle)
	}
	v[objNr] = true
	return nil
}

func (v targetTraversal) leave(objNr int) {
	if objNr > 0 {
		delete(v, objNr)
	}
}

func validateTargetDictEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	return validateTargetDictEntryDepth(c, xRefTable, d, dictName, entryName, required, sinceVersion, 0, targetTraversal{})
}

func validateTargetDictEntryDepth(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, depth int, visit targetTraversal) (err error) {
	// table 202
	parentDictName := dictName
	rawEntry := d[entryName]
	targetObjNr := validationObjectNumber(0, rawEntry)
	defer func() {
		err = model.WithValidationErrorObject(err, targetObjNr)
	}()

	if err := contextutil.Check(c); err != nil {
		return err
	}

	d1, err := validateDictEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
		}
		return nil
	}
	if err := xRefTable.CheckRecursionDepth("embedded target chain", depth); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}
	if err := visit.enter(targetObjNr); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}
	defer visit.leave(targetObjNr)

	dictName = "targetDict"

	// R, required, name
	_, err = validateNameEntry(xRefTable, d1, 0, dictName, "R", REQUIRED, model.V10, func(s string) bool { return s == "P" || s == "C" })
	if err != nil {
		return fmt.Errorf("%s: R: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}

	// N, optional, byte string
	_, err = validateStringEntry(xRefTable, d1, 0, dictName, "N", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s: N: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}

	// P, optional, integer or byte string
	err = validateIntOrStringEntry(xRefTable, d1, targetObjNr, dictName, "P", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s: P: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}

	// A, optional, integer or text string
	err = validateIntOrStringEntry(xRefTable, d1, targetObjNr, dictName, "A", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s: A: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}

	// T, optional, target dict
	if err := validateTargetDictEntryDepth(c, xRefTable, d1, dictName, "T", OPTIONAL, model.V10, depth+1, visit); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(parentDictName, entryName, rawEntry), err)
	}

	return nil
}

func validateGoToEActionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	// see 12.6.4.4 Embedded Go-To Actions

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V11)
	if err != nil {
		return err
	}

	// D, required, name, byte string or array
	err = validateRemoteActionDestinationEntry(xRefTable, d, ownerObjNr, dictName, "D")
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return err
		}
		err = validateRemoteActionDestinationEntry(xRefTable, d, ownerObjNr, dictName, "Dest")
		if err != nil && xRefTable.ValidationMode == model.ValidationRelaxed {
			err = nil
			model.ShowSkipped("GotoEAction: missing \"D\"")
		} else {
			d["D"] = d["Dest"]
			delete(d, "Dest")
			model.ShowRepaired("GotoEAction destination")
		}
	}

	// NewWindow, optional, boolean, since V1.2
	sinceVersion := model.V12
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "NewWindow", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// T, required unless entry F is present, target dict
	if err := validateTargetDictEntry(c, xRefTable, d, dictName, "T", f == nil, model.V10); err != nil {
		return err
	}
	collectFileSpecificationTarget(xRefTable, f, linkTargetFile, linkSourceEmbeddedGoTo)
	return nil
}

func validateWinDict(xRefTable *model.XRefTable, d types.Dict) error {
	// see table 204

	dictName := "winDict"

	// F, required, byte string
	_, err := validateStringEntry(xRefTable, d, 0, dictName, "F", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// D, optional, byte string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "D", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// O, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "O", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, byte string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "P", OPTIONAL, model.V10, nil)

	return err
}

func validateLaunchActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.5

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V11)
	if err != nil {
		return err
	}

	// Win, optional, dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "Win", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateWinDict(xRefTable, d1)
		if err != nil {
			return err
		}
		collectFileSpecificationTarget(xRefTable, d1["F"], linkTargetExecutable, linkSourceLaunch)
	}

	// Mac, optional, undefined dict

	// Unix, optional, undefined dict

	collectFileSpecificationTarget(xRefTable, f, linkTargetExecutable, linkSourceLaunch)
	return nil
}

func validateDestinationThreadEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	// The destination thread (table 205)

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case types.Dict, types.StringLiteral, types.Integer:
		// an indRef to a thread dictionary
		// or an index of the thread within the roots Threads array
		// or the title of the thread as specified in its thread info dict

	default:
		return fmt.Errorf("%s.%s: expected thread dict, string or integer, got %T", dictName, entryName, o)
	}

	return nil
}

func validateDestinationBeadEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	// The bead in the destination thread (table 205)

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case types.Dict, types.Integer:
		// an indRef to a bead dictionary of a thread in the current file
		// or an index of the thread within its thread

	default:
		return fmt.Errorf("%s.%s: expected bead dict or integer, got %T", dictName, entryName, o)
	}

	return nil
}

func validateThreadActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	//see 12.6.4.6

	// F, optional, file specification
	f, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V11)
	if err != nil {
		return err
	}

	// D, required, indRef to thread dict, integer or text string.
	err = validateDestinationThreadEntry(xRefTable, d, dictName, "D", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// B, optional, indRef to bead dict or integer.
	if err = validateDestinationBeadEntry(xRefTable, d, dictName, "B", OPTIONAL, model.V10); err != nil {
		return err
	}
	if !isEmbeddedFileSpecification(xRefTable, f) {
		collectFileSpecificationTarget(xRefTable, f, linkTargetFile, linkSourceThread)
	}
	return nil
}

func validateURIActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.7

	// URI, required, string
	uri, err := validateStringEntry(xRefTable, d, 0, dictName, "URI", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	if uri != nil {
		collectLinkTarget(xRefTable, *uri, linkTargetURI, linkSourceURIAction)
	}

	// IsMap, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "IsMap", OPTIONAL, model.V10, nil)

	return err
}

func soundStreamDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) (*types.StreamDict, error) {
	sd, err := validateStreamDictEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil || sd != nil {
		return sd, err
	}
	o, found := d.Find(entryName)
	if !found || o == nil {
		return nil, nil
	}
	return validateStreamDictForObject(xRefTable, o, 0)
}

func validateSoundDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	sd, err := soundStreamDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || sd == nil {
		return err
	}

	dictName = "soundDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, sd.Dict, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Sound" })
	if err != nil {
		return err
	}

	// F, optional, file specification for self-describing external sound data.
	f, err := validateFileSpecEntry(xRefTable, sd.Dict, dictName, "F", OPTIONAL, model.V12)
	if err != nil {
		return err
	}
	if f != nil && !isEmbeddedFileSpecification(xRefTable, f) {
		collectFileSpecificationTarget(xRefTable, f, linkTargetFile, linkSourceSound)
	}

	// R, required without F, number - sampling rate
	_, err = validateNumberEntry(xRefTable, sd.Dict, 0, dictName, "R", f == nil, model.V10, func(f float64) bool {
		return f > 0
	})
	if err != nil {
		return err
	}

	// C, optional, integer - number of sound channels
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "C", OPTIONAL, model.V10, func(i int) bool {
		return i > 0
	})
	if err != nil {
		return err
	}

	// B, optional, integer - bits per sample value per channel
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "B", OPTIONAL, model.V10, func(i int) bool {
		return i > 0
	})
	if err != nil {
		return err
	}

	// E, optional, name - encoding format
	validateSampleDataEncoding := func(s string) bool {
		return types.MemberOf(s, []string{"Raw", "Signed", "muLaw", "ALaw"})
	}
	if _, err = validateNameEntry(
		xRefTable, sd.Dict, 0, dictName, "E", OPTIONAL, model.V10, validateSampleDataEncoding,
	); err != nil {
		return err
	}

	// CO, optional, name - sound compression format
	_, err = validateNameEntry(xRefTable, sd.Dict, 0, dictName, "CO", OPTIONAL, model.V10, nil)

	return err
}

func validateSoundActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.8

	// Sound, required, stream dict
	err := validateSoundDictEntry(xRefTable, d, dictName, "Sound", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Volume, optional, number: -1.0 .. +1.0
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Volume", OPTIONAL, model.V10, func(f float64) bool { return -1.0 <= f && f <= 1.0 })
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Synchronous", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Repeat", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Mix, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Mix", OPTIONAL, model.V10, nil)

	return err
}

func validateMovieStartOrDurationEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	a, ok := o.(types.Array)
	if !ok {
		return validateMovieTimeValue(xRefTable, o, 0, dictName+"."+entryName)
	}
	if len(a) != 2 {
		return fmt.Errorf("%s.%s: expected array length 2, got %d", dictName, entryName, len(a))
	}
	if err := validateMovieTimeValue(xRefTable, a[0], 0, dictName+"."+entryName+"[0]"); err != nil {
		return err
	}
	_, err = validateIntegerForObject(xRefTable, a[1], 0, func(i int) bool { return i > 0 })
	return err
}

func validateMovieTimeValue(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, context string) error {
	objNr := validationObjectNumber(ownerObjNr, o)
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("%s: dereference: %w", context, err), objNr)
	}
	switch o := o.(type) {
	case types.Integer:
		if o.Value() >= 0 {
			return nil
		}
	case types.StringLiteral:
		bb, err := types.Unescape(o.Value())
		if err != nil {
			return model.WithValidationErrorObject(fmt.Errorf("%s: decode string: %w", context, err), objNr)
		}
		if len(bb) == 8 {
			return nil
		}
	case types.HexLiteral:
		bb, err := o.Bytes()
		if err != nil {
			return model.WithValidationErrorObject(fmt.Errorf("%s: decode string: %w", context, err), objNr)
		}
		if len(bb) == 8 {
			return nil
		}
	default:
		return model.WithValidationErrorObject(
			fmt.Errorf("%s: expected non-negative integer or 8-byte string", context), objNr,
		)
	}
	return model.WithValidationErrorObject(fmt.Errorf("%s: invalid time value", context), objNr)
}

func validateMoviePositiveIntegerArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string) error {
	a, err := validateIntegerArrayEntry(
		xRefTable, d, 0, dictName, entryName, OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 },
	)
	if err != nil {
		return err
	}
	for _, o := range a {
		if _, err := validateIntegerForObject(xRefTable, o, 0, func(i int) bool { return i > 0 }); err != nil {
			return err
		}
	}
	return nil
}

func validateMovieUnitIntervalArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string) error {
	a, err := validateNumberArrayEntry(
		xRefTable, d, 0, dictName, entryName, OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 },
	)
	if err != nil {
		return err
	}
	for _, o := range a {
		n, err := validateNumberForObject(xRefTable, o, 0)
		if err != nil {
			return err
		}
		var f float64
		switch n := n.(type) {
		case types.Integer:
			f = float64(n.Value())
		case types.Float:
			f = n.Value()
		}
		if f < 0 || f > 1 {
			return fmt.Errorf("dict=%s entry=%s invalid dict entry: %g", dictName, entryName, f)
		}
	}
	return nil
}

func validateMovieActivationDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "movieActivationDict"

	// Start, optional
	err := validateMovieStartOrDurationEntry(xRefTable, d, dictName, "Start", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// Duration, optional
	err = validateMovieStartOrDurationEntry(xRefTable, d, dictName, "Duration", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// Rate, optional, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Rate", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Volume, optional, number: -1.0 .. +1.0
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Volume", OPTIONAL, model.V10, func(f float64) bool {
		return -1 <= f && f <= 1
	})
	if err != nil {
		return err
	}

	// ShowControls, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "ShowControls", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Mode, optional, name
	validatePlayMode := func(s string) bool {
		return types.MemberOf(s, []string{"Once", "Open", "Repeat", "Palindrome"})
	}
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Mode", OPTIONAL, model.V10, validatePlayMode)
	if err != nil {
		return err
	}

	// Synchronous, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Synchronous", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// FWScale, optional, array of 2 positive integers
	err = validateMoviePositiveIntegerArrayEntry(xRefTable, d, dictName, "FWScale")
	if err != nil {
		return err
	}

	// FWPosition, optional, array of 2 numbers [0.0 .. 1.0]
	err = validateMovieUnitIntervalArrayEntry(xRefTable, d, dictName, "FWPosition")

	return err
}

func validateMovieActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.9

	// is a movie activation dict
	err := validateMovieActivationDict(xRefTable, d)
	if err != nil {
		return err
	}

	// Exactly one of T or Annotation is required.
	_, hasTitle := d.Find("T")
	_, hasAnnotation := d.Find("Annotation")
	if hasTitle == hasAnnotation {
		return errors.New("movie action: exactly one of \"T\" and \"Annotation\" is required")
	}

	if hasTitle {
		_, err = validateStringEntry(xRefTable, d, 0, dictName, "T", REQUIRED, model.V10, nil)
		if err != nil {
			return err
		}
	} else {
		err = validateMovieActionAnnotation(xRefTable, d, dictName)
		if err != nil {
			return err
		}
	}

	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Operation", OPTIONAL, model.V10, func(s string) bool {
		return types.MemberOf(s, []string{"Play", "Stop", "Pause", "Resume"})
	})
	return err
}

func validateMovieActionAnnotation(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	ir, err := validateIndRefEntry(xRefTable, d, 0, dictName, "Annotation", REQUIRED, model.V10)
	if err != nil || ir == nil {
		return err
	}

	annotationObjNr := ir.ObjectNumber.Value()
	d, err = xRefTable.DereferenceDict(*ir)
	if err != nil {
		err = fmt.Errorf("%s.Annotation: dereference movie annotation dict: %w", dictName, err)
		return model.WithValidationErrorObject(err, annotationObjNr)
	}
	if d == nil {
		err = errors.New("movie action: missing required entry \"T\" or \"Annotation\"")
		return model.WithValidationErrorObject(err, annotationObjNr)
	}

	_, err = validateNameEntry(
		xRefTable, d, annotationObjNr, "annotDict", "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "Movie" },
	)
	return model.WithValidationErrorObject(err, annotationObjNr)
}

func validateHideActionDictEntryT(xRefTable *model.XRefTable, o types.Object) error {
	switch o := o.(type) {

	case types.StringLiteral, types.HexLiteral:
		// Ensure UTF16 correctness.
		_, err := model.Text(o)
		if err != nil {
			return err
		}

	case types.Dict:
		// annotDict,  Check for required name Subtype
		_, err := validateNameEntry(xRefTable, o, 0, "annotDict", "Subtype", REQUIRED, model.V10, nil)
		if err != nil {
			return err
		}

	case types.Array:
		// mixed array of annotationDict indRefs and strings
		for i, v := range o {

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return fmt.Errorf("Hide.T[%d]: dereference: %w", i, err)
			}

			if o == nil {
				continue
			}

			switch o := o.(type) {

			case types.StringLiteral, types.HexLiteral:
				// Ensure UTF16 correctness.
				_, err = model.Text(o)
				if err != nil {
					return fmt.Errorf("Hide.T[%d]: string: %w", i, err)
				}

			case types.Dict:
				// annotDict,  Check for required name Subtype
				_, err = validateNameEntry(xRefTable, o, 0, "annotDict", "Subtype", REQUIRED, model.V10, nil)
				if err != nil {
					return fmt.Errorf("Hide.T[%d]: annotation dict: %w", i, err)
				}
			default:
				return fmt.Errorf("Hide.T[%d]: expected string or annotation dict, got %T", i, o)
			}
		}

	default:
		return fmt.Errorf("Hide.T: expected string, annotation dict or array, got %T", o)

	}

	return nil
}

func validateHideActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.10

	// T, required, dict, text string or array
	o, found := d.Find("T")
	if !found || o == nil {
		return errors.New("Hide action: missing required entry \"T\"")
	}

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("Hide.T: dereference: %w", err)
	}
	if o == nil {
		return errors.New("Hide.T: missing object")
	}

	err = validateHideActionDictEntryT(xRefTable, o)
	if err != nil {
		return err
	}

	// H, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "H", OPTIONAL, model.V10, nil)

	return err
}

func validateNamedActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.11

	standard := func(s string) bool {
		return types.MemberOf(s, []string{"NextPage", "PrevPage", "FirstPage", "LastPage"})
	}

	nonStandard := func(s string) bool {
		// Some known nonstandard named actions.
		return types.MemberOf(s, []string{
			"AcroSrch:Query", "AcroSrch:Results", "Find", "FindAgain", "FindAgainDoc", "FindPrevious",
			"FindPreviousDoc", "FullScreen", "GoBack", "GoBackDoc", "GoForward", "GoToPage", "Print", "Quit",
			"SaveAs", "ShowHideBookmarks", "FitPage", "FitWidth", "Close", "CropPages",
			"ZoomViewIn", "ZoomViewOut",
		})
	}

	n, err := validateNameEntry(xRefTable, d, 0, dictName, "N", REQUIRED, model.V10, func(s string) bool { return standard(s) || nonStandard(s) })
	if err != nil {
		return err
	}
	if n != nil && xRefTable.ValidationMode == model.ValidationRelaxed && nonStandard(n.Value()) {
		model.ShowDigestedSpecViolation(fmt.Sprintf("dict=%s entry=N invalid dict entry: %s", dictName, n.Value()))
	}
	return nil
}

func validateSubmitFormActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.7.5.2

	// F, required, URL specification
	f, err := validateURLSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, "Fields", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	for i, v := range a {
		objNr := validationObjectNumber(validationEntryObjectNumber(0, d, "Fields"), v)
		if _, ok := v.(types.IndirectRef); !ok {
			switch v.(type) {
			case types.StringLiteral, types.HexLiteral:
				continue
			default:
				err := fmt.Errorf("SubmitForm.Fields[%d]: expected string or indirect reference, got %T", i, v)
				return model.WithValidationErrorObject(err, objNr)
			}
		}
		o, err := xRefTable.Dereference(v)
		if err != nil {
			err = fmt.Errorf("SubmitForm.Fields[%d]: dereference: %w", i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		switch o.(type) {
		case types.StringLiteral, types.HexLiteral, types.Dict:
			// no further processing.
		default:
			err := fmt.Errorf("SubmitForm.Fields[%d]: expected string or field dictionary, got %T", i, o)
			return model.WithValidationErrorObject(err, objNr)
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "Flags", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	collectFileSpecificationTarget(xRefTable, f, linkTargetURI, linkSourceSubmitForm)
	return nil
}

func validateResetFormActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.7.5.3

	// Fields, optional, array
	// Each element of the array shall be either an indirect reference to a field dictionary
	// or (PDF 1.3) a text string representing the fully qualified name of a field.
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, "Fields", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	for i, v := range a {
		objNr := validationObjectNumber(validationEntryObjectNumber(0, d, "Fields"), v)
		if _, ok := v.(types.IndirectRef); !ok {
			switch v.(type) {
			case types.StringLiteral, types.HexLiteral:
				continue
			default:
				err := fmt.Errorf("ResetForm.Fields[%d]: expected string or indirect reference, got %T", i, v)
				return model.WithValidationErrorObject(err, objNr)
			}
		}
		o, err := xRefTable.Dereference(v)
		if err != nil {
			err = fmt.Errorf("ResetForm.Fields[%d]: dereference: %w", i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		switch o.(type) {
		case types.StringLiteral, types.HexLiteral, types.Dict:
			// no further processing.
		default:
			err := fmt.Errorf("ResetForm.Fields[%d]: expected string or field dictionary, got %T", i, o)
			return model.WithValidationErrorObject(err, objNr)
		}
	}

	// Flags, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "Flags", OPTIONAL, model.V10, nil)

	return err
}

func validateImportDataActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.7.5.4

	// F, required, file specification
	f, err := validateFileSpecEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V11)
	if err != nil {
		return err
	}
	collectFileSpecificationTarget(xRefTable, f, linkTargetFile, linkSourceImportData)
	return nil
}

func validateJavaScript(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool) error {
	sinceVersion := model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}

	relaxedRequired := required && xRefTable.ValidationMode == model.ValidationRelaxed
	if relaxedRequired {
		required = OPTIONAL
	}

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if o == nil && err == nil && relaxedRequired {
			model.ShowSkipped(dictName + `Action: missing "` + entryName + `"`)
		}
		return err
	}

	switch o := o.(type) {

	case types.StringLiteral:
		// Ensure UTF16 correctness.
		_, err = types.StringLiteralToString(o)

	case types.HexLiteral:
		// Ensure UTF16 correctness.
		_, err = types.HexLiteralToString(o)

	case types.StreamDict:
		// no further processing

	default:
		err = fmt.Errorf("%s.%s: expected string, hex literal or stream dict, got %T", dictName, entryName, o)

	}

	return err
}

func validateJavaScriptActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.16

	// JS, required, text string or stream
	return validateJavaScript(xRefTable, d, dictName, "JS", REQUIRED)
}

func validateSetOCGStateGroup(xRefTable *model.XRefTable, o types.Object, objNr, index int, dictName string) error {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		err = fmt.Errorf("%s.State[%d]: dereference optional content group: %w", dictName, index, err)
		return model.WithValidationErrorObject(err, objNr)
	}
	d, ok := o.(types.Dict)
	if !ok {
		err = fmt.Errorf("%s.State[%d]: expected optional content group dictionary", dictName, index)
		return model.WithValidationErrorObject(err, objNr)
	}
	if err = validateOptionalContentGroupDict(xRefTable, d, objNr, model.V15); err != nil {
		return fmt.Errorf("%s.State[%d]: %w", dictName, index, err)
	}
	return nil
}

func validateSetOCGStateArray(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	const entryName = "State"
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}
	objNr := validationEntryObjectNumber(0, d, entryName)
	if len(a) == 0 {
		return arrayCardinalityError(dictName, entryName, objNr, 0, "one or more operator and OCG sequences")
	}
	operatorSeen, operandSeen := false, false
	for i, raw := range a {
		entryObjNr := validationObjectNumber(objNr, raw)
		o, err := xRefTable.Dereference(raw)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: dereference: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		if n, ok := o.(types.Name); ok {
			if !types.MemberOf(n.Value(), []string{"ON", "OFF", "Toggle"}) {
				err = fmt.Errorf("%s.%s[%d]: invalid operator %s", dictName, entryName, i, n.Value())
				return model.WithValidationErrorObject(err, entryObjNr)
			}
			if operatorSeen && !operandSeen {
				err = fmt.Errorf("%s.%s[%d]: preceding operator has no OCG operands", dictName, entryName, i)
				return model.WithValidationErrorObject(err, entryObjNr)
			}
			operatorSeen, operandSeen = true, false
			continue
		}
		if !operatorSeen {
			err = fmt.Errorf("%s.%s[%d]: expected ON, OFF, or Toggle operator", dictName, entryName, i)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		if err = validateSetOCGStateGroup(xRefTable, raw, entryObjNr, i, dictName); err != nil {
			return err
		}
		operandSeen = true
	}
	if !operandSeen {
		err = fmt.Errorf("%s.%s[%d]: operator has no OCG operands", dictName, entryName, len(a)-1)
		return model.WithValidationErrorObject(err, objNr)
	}
	return nil
}

func validateSetOCGStateActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.12

	// State, required, array
	err := validateSetOCGStateArray(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	// PreserveRB, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "PreserveRB", OPTIONAL, model.V10, nil)

	return err
}

func validateRenditionScreenPage(c context.Context, xRefTable *model.XRefTable, annotationRef, pageRef types.IndirectRef) error {
	pageObjNr := pageRef.ObjectNumber.Value()
	pageDict, err := xRefTable.DereferenceDict(pageRef)
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("Rendition.AN.P: dereference page: %w", err), pageObjNr)
	}
	if pageDict == nil {
		return model.WithValidationErrorObject(errors.New("Rendition.AN.P: page is null"), pageObjNr)
	}
	if _, err = validateNameEntry(
		xRefTable, pageDict, pageObjNr, "pageDict", "Type", REQUIRED, model.V10, func(s string) bool { return s == "Page" },
	); err != nil {
		return err
	}
	pageNumber, err := xRefTable.PageNumber(c, pageObjNr)
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("Rendition.AN.P: locate page: %w", err), pageObjNr)
	}
	if pageNumber == 0 {
		return model.WithValidationErrorObject(errors.New("Rendition.AN.P: page is not in the page tree"), pageObjNr)
	}
	annots, err := validateArrayEntry(xRefTable, pageDict, pageObjNr, "pageDict", "Annots", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}
	for _, o := range annots {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if ir, ok := o.(types.IndirectRef); ok && ir == annotationRef {
			return nil
		}
	}
	return model.WithValidationErrorObject(
		errors.New("Rendition.AN: annotation is not in the page Annots array"), pageObjNr,
	)
}

func validateRenditionScreenTarget(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, required bool) error {
	annotationRef, err := validateIndRefEntry(xRefTable, d, ownerObjNr, dictName, "AN", required, model.V10)
	if err != nil || annotationRef == nil {
		return err
	}
	annotationObjNr := annotationRef.ObjectNumber.Value()
	annotationDict, err := xRefTable.DereferenceDict(*annotationRef)
	if err != nil {
		return model.WithValidationErrorObject(
			fmt.Errorf("Rendition.AN: dereference Screen annotation: %w", err), annotationObjNr,
		)
	}
	if annotationDict == nil {
		return model.WithValidationErrorObject(errors.New("Rendition.AN: Screen annotation is null"), annotationObjNr)
	}
	if err := validateRenditionScreenAnnotationStructure(c, xRefTable, annotationDict, annotationObjNr); err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("Rendition.AN: %w", err), annotationObjNr)
	}
	pageRef, err := validateIndRefEntry(xRefTable, annotationDict, annotationObjNr, "annotDict", "P", REQUIRED, model.V10)
	if err != nil {
		return err
	}
	if pageRef == nil {
		return model.WithValidationErrorObject(errors.New("Rendition.AN: missing Screen annotation P entry"), annotationObjNr)
	}
	return validateRenditionScreenPage(c, xRefTable, *annotationRef, *pageRef)
}

func validateRenditionActionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	// see 12.6.4.13

	// OP or JS need to be present.

	// OP, integer
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	op, err := validateIntegerEntry(xRefTable, d, 0, dictName, "OP", OPTIONAL, sinceVersion, func(i int) bool { return 0 <= i && i <= 4 })
	if err != nil {
		return err
	}

	// JS, text string or stream
	err = validateJavaScript(xRefTable, d, dictName, "JS", op == nil)
	if err != nil {
		return err
	}

	// R, required for OP 0 and 4, rendition object dict
	required := func(op *types.Integer) bool {
		if op == nil {
			return false
		}
		v := op.Value()
		return v == 0 || v == 4
	}(op)

	sinceVersion = model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	rawR := d["R"]
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "R", required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, "R", rawR), err)
	}
	if d1 != nil {
		err = validateRenditionDict(
			c, xRefTable, d1, validationObjectNumber(0, rawR), sinceVersion,
		)
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(dictName, "R", rawR), err)
		}
	}

	// AN, required for any OP 0..4, indRef of screen annotation dict
	return validateRenditionScreenTarget(c, xRefTable, d, ownerObjNr, dictName, op != nil)
}

func validateTransActionDict(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// see 12.6.4.14

	// Trans, required, transitionDict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "Trans", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	return validateTransitionDict(xRefTable, d1, 0)
}

func validateGoTo3DViewActionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	// see 12.6.4.15

	// TA, required, target annotation
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "TA", REQUIRED, model.V16, nil)
	if err != nil {
		return err
	}

	taObjNr := validationEntryObjectNumber(ownerObjNr, d, "TA")
	_, err = validateAnnotationDict(c, xRefTable, d1, taObjNr)
	if err != nil {
		return err
	}

	// V, required, the view to use: 3DViewDict or integer or text string or name
	// TODO Validation.
	_, err = validateEntry(xRefTable, d, 0, dictName, "V", REQUIRED, model.V16)

	return err
}

func validateActionDictCore(c context.Context, xRefTable *model.XRefTable, n *types.Name, d types.Dict, ownerObjNr int) error {
	for k, v := range map[string]struct {
		validate            func(xRefTable *model.XRefTable, d types.Dict, dictName string) error
		sinceVersion        model.Version
		sinceVersionRelaxed model.Version
	}{
		"GoTo": {validateGoToActionDict, model.V10, model.V10},
		"GoToR": {func(x *model.XRefTable, d types.Dict, name string) error {
			return validateGoToRActionDict(x, d, ownerObjNr, name)
		}, model.V10, model.V10},
		"GoToE": {func(x *model.XRefTable, d types.Dict, name string) error {
			return validateGoToEActionDict(c, x, d, ownerObjNr, name)
		}, model.V16, model.V11},
		"Launch":      {validateLaunchActionDict, model.V10, model.V10},
		"Thread":      {validateThreadActionDict, model.V10, model.V10},
		"URI":         {validateURIActionDict, model.V10, model.V10},
		"Sound":       {validateSoundActionDict, model.V12, model.V12},
		"Movie":       {validateMovieActionDict, model.V12, model.V12},
		"Hide":        {validateHideActionDict, model.V12, model.V12},
		"Named":       {validateNamedActionDict, model.V12, model.V12},
		"SubmitForm":  {validateSubmitFormActionDict, model.V10, model.V10},
		"ResetForm":   {validateResetFormActionDict, model.V12, model.V12},
		"ImportData":  {validateImportDataActionDict, model.V12, model.V12},
		"JavaScript":  {validateJavaScriptActionDict, model.V13, model.V12},
		"SetOCGState": {validateSetOCGStateActionDict, model.V15, model.V15},
		"Rendition": {func(x *model.XRefTable, d types.Dict, name string) error {
			return validateRenditionActionDict(c, x, d, ownerObjNr, name)
		}, model.V15, model.V14},
		"Trans": {validateTransActionDict, model.V15, model.V15},
		"GoTo3DView": {func(x *model.XRefTable, d types.Dict, name string) error {
			return validateGoTo3DViewActionDict(c, x, d, ownerObjNr, name)
		}, model.V16, model.V16},
	} {
		if n.Value() == k {

			sinceVersion := v.sinceVersion
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				sinceVersion = v.sinceVersionRelaxed
			}

			err := xRefTable.ValidateVersion(k, sinceVersion)
			if err != nil {
				return fmt.Errorf("action %s: version: %w", k, err)
			}

			if err := v.validate(xRefTable, d, k); err != nil {
				return fmt.Errorf("action %s: %w", k, err)
			}
			return nil
		}
	}

	return fmt.Errorf("action %s: unsupported action type %q", n.Value(), n.Value())
}

type activeContentOrigin struct {
	owner      activeContentOwner
	source     activeContentSource
	trigger    string
	ownerObjNr int
}

func defaultActiveContentOrigin() activeContentOrigin {
	return activeContentOrigin{
		owner:  activeContentOwnerAction,
		source: activeContentSourceAction,
	}
}

func activeContentPageNumber(xRefTable *model.XRefTable, owner activeContentOwner) int {
	switch owner {
	case activeContentOwnerDocument, activeContentOwnerNameTree, activeContentOwnerOutline:
		return 0
	default:
		return xRefTable.CurPage
	}
}

func hasJavaScriptPayload(xRefTable *model.XRefTable, d types.Dict) bool {
	o, found := d.Find("JS")
	if !found {
		return false
	}
	o, err := xRefTable.Dereference(o)
	return err == nil && o != nil
}

func hasRenditionOperation(xRefTable *model.XRefTable, d types.Dict) bool {
	i, found, err := xRefTable.DereferenceIntegerEntry(d, "OP")
	return err == nil && found && i != nil
}

func addActionActiveContent(xRefTable *model.XRefTable, kind activeContentKind, source activeContentSource, o types.Object, depth int, origin activeContentOrigin) {
	addActiveContentNotice(xRefTable, activeContentFinding{
		kind:       kind,
		owner:      origin.owner,
		source:     source,
		trigger:    origin.trigger,
		pageNr:     activeContentPageNumber(xRefTable, origin.owner),
		objNr:      validationObjectNumber(origin.ownerObjNr, o),
		ownerObjNr: origin.ownerObjNr,
		depth:      depth,
	})
}

func collectActionActiveContent(xRefTable *model.XRefTable, d types.Dict, o types.Object, depth int, origin activeContentOrigin) {
	n, found, err := xRefTable.DereferenceNameEntry(d, "S")
	if err != nil || !found || n == nil {
		return
	}
	switch n.Value() {
	case "JavaScript":
		if hasJavaScriptPayload(xRefTable, d) {
			addActionActiveContent(xRefTable, activeContentJavaScript, origin.source, o, depth, origin)
		}
	case "Rendition":
		if hasJavaScriptPayload(xRefTable, d) {
			addActionActiveContent(
				xRefTable, activeContentJavaScript, activeContentSourceRenditionAction, o, depth, origin,
			)
		}
		if hasRenditionOperation(xRefTable, d) {
			addActionActiveContent(xRefTable, activeContentRendition, origin.source, o, depth, origin)
		}
	case "Sound":
		addActionActiveContent(xRefTable, activeContentSound, origin.source, o, depth, origin)
	case "Movie":
		addActionActiveContent(xRefTable, activeContentMovie, origin.source, o, depth, origin)
	}
}

func validateActionDictObject(c context.Context, xRefTable *model.XRefTable, d types.Dict, o types.Object, context string) error {
	return validateActionDictObjectWithOrigin(c, xRefTable, d, o, context, defaultActiveContentOrigin())
}

func validateActionDictObjectWithOrigin(c context.Context, xRefTable *model.XRefTable, d types.Dict, o types.Object, context string, origin activeContentOrigin) error {
	return validateActionDictObjectDepth(c, xRefTable, d, o, context, 0, model.NewActionVisit(), origin)
}

func validateActionDictObjectDepth(c context.Context, xRefTable *model.XRefTable, d types.Dict, o types.Object, context string, depth int, visit *model.ActionVisit, origin activeContentOrigin) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := xRefTable.CheckRecursionDepth("action chain", depth); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}

	if err := visit.Enter(objNr); err != nil {
		return fmt.Errorf("%s: %w", objectContext(context, o), err)
	}
	defer visit.Leave(objNr)

	// A deeper path needs revalidation to preserve descendant depth checks.
	if visit.AlreadyValidated(objNr, depth) {
		return nil
	}

	if err := validateActionDict(c, xRefTable, d, objNr, depth, visit, origin); err != nil {
		return model.WrapRecursionError(objectContext(context, o), err)
	}
	collectActionActiveContent(xRefTable, d, o, depth, origin)
	visit.MarkValidated(objNr, depth)
	return nil
}

func validateNextAction(c context.Context, xRefTable *model.XRefTable, o types.Object, depth int, visit *model.ActionVisit, origin activeContentOrigin) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d, err := xRefTable.DereferenceDict(o)
	if err == nil {
		if d == nil {
			return nil
		}
		if err := validateActionDictObjectDepth(c, xRefTable, d, o, "action Next", depth+1, visit, origin); err != nil {
			return err
		}
		return nil
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("action Next: expected action dict or array: %w", err)
	}
	if a == nil {
		return nil
	}

	for i, v := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		nextObjNr := validationObjectNumber(objNr, v)
		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("action Next[%d]: dereference dict: %w", i, err)
			return model.WithValidationErrorObject(err, nextObjNr)
		}
		if d == nil {
			continue
		}
		if err := validateActionDictObjectDepth(c, xRefTable, d, v, fmt.Sprintf("action Next[%d]", i), depth+1, visit, origin); err != nil {
			return err
		}
	}

	return nil
}

func validateActionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr, depth int, visit *model.ActionVisit, origin activeContentOrigin) error {
	dictName := "actionDict"

	// Type, optional, name
	allowedTypes := []string{"Action"}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		allowedTypes = []string{"A", "Action"}
	}
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return types.MemberOf(s, allowedTypes) })
	if err != nil {
		return fmt.Errorf("action dictionary Type: %w", err)
	}

	// S, required, name, action Type
	required := REQUIRED
	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	if relaxed {
		required = OPTIONAL
	}
	s, err := validateNameEntry(xRefTable, d, 0, dictName, "S", required, model.V10, nil)
	if err != nil {
		return fmt.Errorf("action dictionary S: %w", err)
	}

	if s != nil {
		err = validateActionDictCore(c, xRefTable, s, d, ownerObjNr)
		if err != nil {
			return err
		}
	}

	if o, ok := d.Find("Next"); ok {
		if err := validateNextAction(c, xRefTable, o, depth, visit, origin); err != nil {
			return err
		}
	}

	if s == nil && relaxed {
		model.ShowDigestedSpecViolation("dict=" + dictName + " required entry=S missing")
	}

	return nil
}

func validateRootAdditionalActions(c context.Context, xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	return validateAdditionalActionsWithOwner(
		c, xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "AA", required, sinceVersion, "root",
	)
}

func additionalActionOwner(source, trigger string) activeContentOwner {
	switch source {
	case "root":
		return activeContentOwnerDocument
	case "page":
		return activeContentOwnerPage
	case "fieldOrAnnot":
		if types.MemberOf(trigger, []string{"K", "F", "V", "C"}) {
			return activeContentOwnerFormField
		}
		return activeContentOwnerAnnotation
	default:
		return activeContentOwnerAction
	}
}

func validateAdditionalActions(c context.Context, xRefTable *model.XRefTable, dict types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, source string) (err error) {
	return validateAdditionalActionsWithOwner(c, xRefTable, dict, 0, dictName, entryName, required, sinceVersion, source)
}

func validateAdditionalActionsWithOwner(c context.Context, xRefTable *model.XRefTable, dict types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, source string) (err error) {
	actionsObjNr := validationEntryObjectNumber(ownerObjNr, dict, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, actionsObjNr)
	}()

	d, err := validateDictEntry(xRefTable, dict, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	validateAdditionalAction := func(s, source string) bool {

		switch source {

		case "root":
			if types.MemberOf(s, []string{"WC", "WS", "DS", "WP", "DP"}) {
				return true
			}

		case "page":
			if types.MemberOf(s, []string{"O", "C"}) {
				return true
			}

		case "fieldOrAnnot":
			// A terminal form field may be merged with a widget annotation.
			fieldOptions := []string{"K", "F", "V", "C"}
			annotOptions := []string{"E", "X", "D", "U", "Fo", "Bl", "PO", "PC", "PV", "PI"}
			options := append(fieldOptions, annotOptions...)
			if types.MemberOf(s, options) {
				return true
			}

		}

		return false
	}

	for _, k := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		v := d[k]
		actionObjNr := validationObjectNumber(actionsObjNr, v)

		if !validateAdditionalAction(k, source) {
			return fmt.Errorf("additional action %s.%s: action %s not allowed for source %s", dictName, entryName, k, source)
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("additional action %s.%s.%s: dereference dict: %w", dictName, entryName, k, err)
			return model.WithValidationErrorObject(err, actionObjNr)
		}

		if d == nil {
			continue
		}

		origin := activeContentOrigin{
			owner:      additionalActionOwner(source, k),
			source:     activeContentSourceAdditionalAction,
			trigger:    k,
			ownerObjNr: ownerObjNr,
		}
		err = validateActionDictObjectWithOrigin(
			c, xRefTable, d, v, fmt.Sprintf("additional action %s.%s.%s", dictName, entryName, k), origin,
		)
		if err != nil {
			return err
		}

	}

	return nil
}

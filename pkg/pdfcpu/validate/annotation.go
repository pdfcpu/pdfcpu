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
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateAAPLAKExtrasDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// No documentation for this PDF-Extension - purely speculative implementation.

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "AAPLAKExtrasDict"

	// AAPL:AKAnnotationObject, string
	_, err = validateStringEntry(xRefTable, d1, dictName, "AAPL:AKAnnotationObject", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// AAPL:AKPDFAnnotationDictionary, annotationDict
	ad, err := validateDictEntry(xRefTable, d1, dictName, "AAPL:AKPDFAnnotationDictionary", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateAnnotationDict(xRefTable, ad)
	if err != nil {
		return err
	}

	return nil
}

func validateBorderEffectDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// see 12.5.4

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "borderEffectDict"

	// S, optional, name, S or C
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, model.V10, func(s string) bool { return s == "S" || s == "C" })
	if err != nil {
		return err
	}

	// I, optional, number in the range 0 to 2
	_, err = validateNumberEntry(xRefTable, d1, dictName, "I", OPTIONAL, model.V10, func(f float64) bool { return 0 <= f && f <= 2 }) // validation missing
	if err != nil {
		return err
	}

	return nil
}

func validateBorderStyleDict(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// see 12.5.4

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "borderStyleDict"

	// Type, optional, name, "Border"
	_, err = validateNameEntry(xRefTable, d1, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Border" })
	if err != nil {
		return err
	}

	// W, optional, number, border width in points
	_, err = validateNumberEntry(xRefTable, d1, dictName, "W", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, name, border style
	validate := func(s string) bool { return types.MemberOf(s, []string{"S", "D", "B", "I", "U", "A"}) }
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}

	// D, optional, dash array
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "D", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) <= 2 })

	return err
}

func validateIconFitDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// see table 247

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "iconFitDict"

	// SW, optional, name, A,B,S,N
	validate := func(s string) bool { return types.MemberOf(s, []string{"A", "B", "S", "N"}) }
	_, err = validateNameEntry(xRefTable, d1, dictName, "SW", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}

	// S, optional, name, A,P
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, model.V10, func(s string) bool { return s == "A" || s == "P" })
	if err != nil {
		return err
	}

	// A,optional, array of 2 numbers between 0.0 and 1.0
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "A", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// FB, optional, bool, since V1.5
	_, err = validateBooleanEntry(xRefTable, d1, dictName, "FB", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateAppearanceCharacteristicsDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// see 12.5.6.19

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "appCharDict"

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, d1, dictName, "R", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// BC, optional, array of numbers, len=0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "BC", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// BG, optional, array of numbers between 0.0 and 0.1, len=0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "BG", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// CA, optional, text string
	_, err = validateStringEntry(xRefTable, d1, dictName, "CA", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// RC, optional, text string
	_, err = validateStringEntry(xRefTable, d1, dictName, "RC", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// AC, optional, text string
	_, err = validateStringEntry(xRefTable, d1, dictName, "AC", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// I, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, d1, dictName, "I", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// RI, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, d1, dictName, "RI", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// IX, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, d1, dictName, "IX", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// IF, optional, icon fit dict,
	err = validateIconFitDictEntry(xRefTable, d1, dictName, "IF", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// TP, optional, integer 0..6
	_, err = validateIntegerEntry(xRefTable, d1, dictName, "TP", OPTIONAL, model.V10, func(i int) bool { return 0 <= i && i <= 6 })

	return err
}

func validateAnnotationDictText(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.4

	// Open, optional, boolean
	_, err := validateBooleanEntry(xRefTable, d, dictName, "Open", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// State, optional, text string, since V1.5
	state, err := validateStringEntry(xRefTable, d, dictName, "State", OPTIONAL, model.V15, nil)
	if err != nil {
		return err
	}

	// StateModel, text string, since V1.5
	validate := func(s string) bool { return types.MemberOf(s, []string{"Marked", "Review"}) }
	stateModel, err := validateStringEntry(xRefTable, d, dictName, "StateModel", state != nil, model.V15, validate)
	if err != nil {
		return err
	}

	if state == nil {
		if stateModel != nil {
			return errors.Errorf("pdfcpu: validateAnnotationDictText: dict=%s missing state for statemodel=%s", dictName, *stateModel)
		}
		return nil
	}

	// Ensure that the state/model combo is valid.
	validStates := []string{"Accepted", "Rejected", "Cancelled", "Completed", "None"} // stateModel "Review"
	if *stateModel == "Marked" {
		validStates = []string{"Marked", "Unmarked"}
	}
	if !types.MemberOf(*state, validStates) {
		return errors.Errorf("pdfcpu: validateAnnotationDictText: dict=%s invalid state=%s for state model=%s", dictName, *state, *stateModel)
	}

	return nil
}

func validateActionOrDestination(xRefTable *model.XRefTable, d types.Dict, dictName string, sinceVersion model.Version) error {

	// The action that shall be performed when this item is activated.
	d1, err := validateDictEntry(xRefTable, d, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		return validateActionDict(xRefTable, d1)
	}

	// A destination that shall be displayed when this item is activated.
	obj, err := validateEntry(xRefTable, d, dictName, "Dest", OPTIONAL, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	name, err := validateDestination(xRefTable, obj, false)
	if err != nil {
		return err
	}

	if len(name) > 0 && xRefTable.IsMerging() {
		nm := xRefTable.NameRef("Dests")
		nm.Add(name, d)
	}

	return nil
}

func validateURIActionDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "URIActionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d1, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Action" })
	if err != nil {
		return err
	}

	// S, required, name, action Type
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", REQUIRED, model.V10, func(s string) bool { return s == "URI" })
	if err != nil {
		return err
	}

	return validateURIActionDict(xRefTable, d1, dictName)
}

func validateAnnotationDictLink(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.5

	// A or Dest, required either or
	err := validateActionOrDestination(xRefTable, d, dictName, model.V11)
	if err != nil {
		return err
	}

	// H, optional, name, since V1.2
	_, err = validateNameEntry(xRefTable, d, dictName, "H", OPTIONAL, model.V12, nil)
	if err != nil {
		return err
	}

	// PA, optional, URI action dict, since V1.3
	err = validateURIActionDictEntry(xRefTable, d, dictName, "PA", OPTIONAL, model.V13)
	if err != nil {
		return err
	}

	// QuadPoints, optional, number array, len= a multiple of 8, since V1.6
	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "QuadPoints", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a)%8 == 0 })
	if err != nil {
		return err
	}

	// BS, optional, border style dict, since V1.6
	return validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, sinceVersion)
}

func validateAnnotationDictFreeTextPart1(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// DA, required, string
	_, err := validateStringEntry(xRefTable, d, dictName, "DA", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Q, optional, integer, since V1.4, 0,1,2
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Q", OPTIONAL, sinceVersion, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// RC, optional, text string or text stream, since V1.5
	sinceVersion = model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	err = validateStringOrStreamEntry(xRefTable, d, dictName, "RC", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// DS, optional, text string, since V1.5
	_, err = validateStringEntry(xRefTable, d, dictName, "DS", OPTIONAL, model.V15, nil)
	if err != nil {
		return err
	}

	// CL, optional, number array, since V1.6, len: 4 or 6
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "CL", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 4 || len(a) == 6 })

	return err
}

func validateAnnotationDictFreeTextPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// IT, optional, name, since V1.6
	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"FreeText", "FreeTextCallout", "FreeTextTypeWriter", "FreeTextTypewriter"})
	}
	_, err := validateNameEntry(xRefTable, d, dictName, "IT", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	// BE, optional, border effect dict, since V1.6
	err = validateBorderEffectDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, model.V15)
	if err != nil {
		return err
	}

	// RD, optional, rectangle, since V1.6
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	_, err = validateRectangleEntry(xRefTable, d, dictName, "RD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BS, optional, border style dict, since V1.6
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	err = validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// LE, optional, name, since V1.6
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "LE", OPTIONAL, sinceVersion, nil)

	return err
}

func validateAnnotationDictFreeText(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.6

	err := validateAnnotationDictFreeTextPart1(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	return validateAnnotationDictFreeTextPart2(xRefTable, d, dictName)
}

func validateEntryMeasure(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "Measure", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateMeasureDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateCP(s string) bool { return s == "Inline" || s == "Top" }

func validateAnnotationDictLine(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.7

	// L, required, array of numbers, len:4
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "L", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 4 })
	if err != nil {
		return err
	}

	// BS, optional, border style dict
	err = validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// LE, optional, name array, since V1.4, len:2
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateNameArrayEntry(xRefTable, d, dictName, "LE", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// IC, optional, number array, since V1.4, len:0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "IC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// LLE, optional, number, since V1.6, >0
	lle, err := validateNumberEntry(xRefTable, d, dictName, "LLE", OPTIONAL, model.V16, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// LL, required if LLE present, number, since V1.6
	_, err = validateNumberEntry(xRefTable, d, dictName, "LL", lle != nil, model.V16, nil)
	if err != nil {
		return err
	}

	// Cap, optional, bool, since V1.6
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Cap", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}

	// IT, optional, name, since V1.6
	_, err = validateNameEntry(xRefTable, d, dictName, "IT", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}

	// LLO, optionl, number, since V1.7, >0
	_, err = validateNumberEntry(xRefTable, d, dictName, "LLO", OPTIONAL, model.V17, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// CP, optional, name, since V1.7
	_, err = validateNameEntry(xRefTable, d, dictName, "CP", OPTIONAL, model.V17, validateCP)
	if err != nil {
		return err
	}

	// Measure, optional, measure dict, since V1.7
	err = validateEntryMeasure(xRefTable, d, dictName, OPTIONAL, model.V17)
	if err != nil {
		return err
	}

	// CO, optional, number array, since V1.7, len=2
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "CO", OPTIONAL, model.V17, func(a types.Array) bool { return len(a) == 2 })

	return err
}

func validateAnnotationDictCircleOrSquare(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.8

	// BS, optional, border style dict
	err := validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// IC, optional, array, since V1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "IC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BE, optional, border effect dict, since V1.5
	err = validateBorderEffectDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, model.V15)
	if err != nil {
		return err
	}

	// RD, optional, rectangle, since V1.5
	_, err = validateRectangleEntry(xRefTable, d, dictName, "RD", OPTIONAL, model.V15, nil)

	return err
}

func validateEntryIT(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	// IT, optional, name, since V1.6
	validateIntent := func(s string) bool {

		if xRefTable.Version() == model.V16 {
			return s == "PolygonCloud"
		}

		if xRefTable.Version() == model.V17 {
			if types.MemberOf(s, []string{"PolygonCloud", "PolyLineDimension", "PolygonDimension"}) {
				return true
			}
		}

		return false

	}

	_, err := validateNameEntry(xRefTable, d, dictName, "IT", required, sinceVersion, validateIntent)

	return err
}

func validateAnnotationDictPolyLine(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.9

	// Vertices, required, array of numbers
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "Vertices", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// LE, optional, array of 2 names, meaningful only for polyline annotations.
	if dictName == "PolyLine" {
		_, err = validateNameArrayEntry(xRefTable, d, dictName, "LE", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
		if err != nil {
			return err
		}
	}

	// BS, optional, border style dict
	err = validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// IC, optional, array of numbers [0.0 .. 1.0], len:1,3,4
	ensureArrayLength := func(a types.Array, lengths ...int) bool {
		for _, length := range lengths {
			if len(a) == length {
				return true
			}
		}
		return false
	}
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "IC", OPTIONAL, model.V14, func(a types.Array) bool { return ensureArrayLength(a, 1, 3, 4) })
	if err != nil {
		return err
	}

	// BE, optional, border effect dict, meaningful only for polygon annotations
	if dictName == "Polygon" {
		err = validateBorderEffectDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, model.V10)
		if err != nil {
			return err
		}
	}

	return validateEntryIT(xRefTable, d, dictName, OPTIONAL, model.V16)
}

func validateTextMarkupAnnotation(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.10

	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	// QuadPoints, required, number array, len: a multiple of 8
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "QuadPoints", required, model.V10, func(a types.Array) bool { return len(a)%8 == 0 })

	return err
}

func validateAnnotationDictStamp(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.12

	// Name, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, model.V10, nil)

	return err
}

func validateAnnotationDictCaret(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.11

	// RD, optional, rectangle, since V1.5
	_, err := validateRectangleEntry(xRefTable, d, dictName, "RD", OPTIONAL, model.V15, nil)
	if err != nil {
		return err
	}

	// Sy, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Sy", OPTIONAL, model.V10, func(s string) bool { return s == "P" || s == "None" })

	return err
}

func validateAnnotationDictInk(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.13

	// InkList, required, array of stroked path arrays
	_, err := validateArrayArrayEntry(xRefTable, d, dictName, "InkList", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// BS, optional, border style dict
	return validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, model.V10)
}

func validateAnnotationDictPopup(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.14

	// Parent, optional, dict indRef
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "Parent", OPTIONAL, model.V10)
	if err != nil {
		return err
	}
	if ir != nil {
		d1, err := xRefTable.DereferenceDict(*ir)
		if err != nil || d1 == nil {
			return err
		}
	}

	// Open, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Open", OPTIONAL, model.V10, nil)

	return err
}

func validateAnnotationDictFileAttachment(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.15

	// FS, required, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "FS", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, model.V10, nil)

	return err
}

func validateAnnotationDictSound(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.16

	// Sound, required, stream dict
	err := validateSoundDictEntry(xRefTable, d, dictName, "Sound", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, model.V10, nil)

	return err
}

func validateMovieDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "movieDict"

	// F, required, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Aspect, optional, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "Ascpect", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// Rotate, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Rotate", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Poster, optional boolean or stream
	return validateBooleanOrStreamEntry(xRefTable, d, dictName, "Poster", OPTIONAL, model.V10)
}

func validateAnnotationDictMovie(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.17 Movie Annotations
	// 13.4 Movies
	// The features described in this sub-clause are obsolescent and their use is no longer recommended.
	// They are superseded by the general multimedia framework described in 13.2, “Multimedia.”

	// T, optional, text string
	_, err := validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Movie, required, movie dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "Movie", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	err = validateMovieDict(xRefTable, d1)
	if err != nil {
		return err
	}

	// A, optional, boolean or movie activation dict
	o, found := d.Find("A")

	if found {

		o, err = xRefTable.Dereference(o)
		if err != nil {
			return err
		}

		if o != nil {
			switch o := o.(type) {
			case types.Boolean:
				// no further processing

			case types.Dict:
				err = validateMovieActivationDict(xRefTable, o)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func validateAnnotationDictWidget(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.19

	// H, optional, name
	validate := func(s string) bool { return types.MemberOf(s, []string{"N", "I", "O", "P", "T", "A"}) }
	_, err := validateNameEntry(xRefTable, d, dictName, "H", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}

	// MK, optional, dict
	// An appearance characteristics dictionary that shall be used in constructing
	// a dynamic appearance stream specifying the annotation’s visual presentation on the page.dict
	err = validateAppearanceCharacteristicsDictEntry(xRefTable, d, dictName, "MK", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// A, optional, dict, since V1.1
	// An action that shall be performed when the annotation is activated.
	d1, err := validateDictEntry(xRefTable, d, dictName, "A", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateActionDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	// AA, optional, dict, since V1.2
	// An additional-actions dictionary defining the annotation’s behaviour in response to various trigger events.
	err = validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, model.V12, "fieldOrAnnot")
	if err != nil {
		return err
	}

	// BS, optional, border style dict, since V1.2
	// A border style dictionary specifying the width and dash pattern
	// that shall be used in drawing the annotation’s border.
	validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, model.V12)
	if err != nil {
		return err
	}

	// Parent, dict, required if one of multiple children in a field.
	// An indirect reference to the widget annotation’s parent field.
	_, err = validateIndRefEntry(xRefTable, d, dictName, "Parent", OPTIONAL, model.V10)

	return err
}

func validateAnnotationDictScreen(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.18

	// T, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// MK, optional, appearance characteristics dict
	err = validateAppearanceCharacteristicsDictEntry(xRefTable, d, dictName, "MK", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// A, optional, action dict, since V1.0
	d1, err := validateDictEntry(xRefTable, d, dictName, "A", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateActionDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	// AA, optional, additional-actions dict, since V1.2
	return validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, model.V12, "fieldOrAnnot")
}

func validateAnnotationDictPrinterMark(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.20

	// MN, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "MN", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// F, required integer, since V1.1, annotation flags
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", REQUIRED, model.V11, nil)
	if err != nil {
		return err
	}

	// AP, required, appearance dict, since V1.2
	return validateAppearDictEntry(xRefTable, d, dictName, REQUIRED, model.V12)
}

func validateAnnotationDictTrapNet(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.21

	// LastModified, optional, date
	_, err := validateDateEntry(xRefTable, d, dictName, "LastModified", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// Version, optional, array
	_, err = validateArrayEntry(xRefTable, d, dictName, "Version", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// AnnotStates, optional, array of names
	_, err = validateNameArrayEntry(xRefTable, d, dictName, "AnnotStates", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// FontFauxing, optional, font dict array
	validateFontDictArray := func(a types.Array) bool {

		var retValue bool

		for _, v := range a {

			if v == nil {
				continue
			}

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return false
			}

			if d == nil {
				continue
			}

			if d.Type() == nil || *d.Type() != "Font" {
				return false
			}

			retValue = true

		}

		return retValue
	}

	_, err = validateArrayEntry(xRefTable, d, dictName, "FontFauxing", OPTIONAL, model.V10, validateFontDictArray)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", REQUIRED, model.V11, nil)

	return err
}

func validateAnnotationDictWatermark(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.22

	// FixedPrint, optional, dict

	validateFixedPrintDict := func(d types.Dict) bool {

		dictName := "fixedPrintDict"

		// Type, required, name
		_, err := validateNameEntry(xRefTable, d, dictName, "Type", REQUIRED, model.V10, func(s string) bool { return s == "FixedPrint" })
		if err != nil {
			return false
		}

		// Matrix, optional, integer array, length = 6
		_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "Matrix", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 6 })
		if err != nil {
			return false
		}

		// H, optional, number
		_, err = validateNumberEntry(xRefTable, d, dictName, "H", OPTIONAL, model.V10, nil)
		if err != nil {
			return false
		}

		// V, optional, number
		_, err = validateNumberEntry(xRefTable, d, dictName, "V", OPTIONAL, model.V10, nil)
		return err == nil
	}

	_, err := validateDictEntry(xRefTable, d, dictName, "FixedPrint", OPTIONAL, model.V10, validateFixedPrintDict)

	return err
}

func validateAnnotationDict3D(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 13.6.2

	// AP with entry N, required

	// 3DD, required, 3D stream or 3D reference dict
	err := validateStreamDictOrDictEntry(xRefTable, d, dictName, "3DD", REQUIRED, model.V16)
	if err != nil {
		return err
	}

	// 3DV, optional, various
	_, err = validateEntry(xRefTable, d, dictName, "3DV", OPTIONAL, model.V16)
	if err != nil {
		return err
	}

	// 3DA, optional, activation dict
	_, err = validateDictEntry(xRefTable, d, dictName, "3DA", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}

	// 3DI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "3DI", OPTIONAL, model.V16, nil)

	return err
}

func validateEntryIC(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	// IC, optional, number array, length:3 [0.0 .. 1.0]
	validateICArray := func(a types.Array) bool {

		if len(a) != 3 {
			return false
		}

		for _, v := range a {

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return false
			}

			switch o := o.(type) {
			case types.Integer:
				if o < 0 || o > 1 {
					return false
				}

			case types.Float:
				if o < 0.0 || o > 1.0 {
					return false
				}
			}
		}

		return true
	}

	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "IC", required, sinceVersion, validateICArray)

	return err
}

func validateAnnotationDictRedact(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// see 12.5.6.23

	// QuadPoints, optional, len: a multiple of 8
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "QuadPoints", OPTIONAL, model.V10, func(a types.Array) bool { return len(a)%8 == 0 })
	if err != nil {
		return err
	}

	// IC, optional, number array, length:3 [0.0 .. 1.0]
	err = validateEntryIC(xRefTable, d, dictName, OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// RO, optional, stream
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "RO", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// OverlayText, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "OverlayText", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Repeat", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// DA, required, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "DA", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// Q, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Q", OPTIONAL, model.V10, nil)

	return err
}

func validateRichMediaAnnotation(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// TODO See extension level 3.
	return nil
}

func validateExDataDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "ExData"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "ExData" })
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "Markup3D" })

	return err
}

func validatePopupEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {

		_, err = validateNameEntry(xRefTable, d1, dictName, "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "Popup" })
		if err != nil {
			return err
		}

		_, err = validateAnnotationDict(xRefTable, d1)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateIRTEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		_, err = validateAnnotationDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateMarkupAnnotationPart1(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// T, optional, text string, since V1.1
	if _, err := validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, model.V11, nil); err != nil {
		return err
	}

	// Popup, optional, dict, since V1.3
	if err := validatePopupEntry(xRefTable, d, dictName, "Popup", OPTIONAL, model.V13); err != nil {
		return err
	}

	// CA, optional, number, since V1.4
	if _, err := validateNumberEntry(xRefTable, d, dictName, "CA", OPTIONAL, model.V14, nil); err != nil {
		return err
	}

	// RC, optional, text string or stream, since V1.5
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	if err := validateStringOrStreamEntry(xRefTable, d, dictName, "RC", OPTIONAL, sinceVersion); err != nil {
		return err
	}

	// CreationDate, optional, date, since V1.5
	sinceVersion = model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	if _, err := validateDateEntry(xRefTable, d, dictName, "CreationDate", OPTIONAL, sinceVersion); err != nil {
		return err
	}

	return nil
}

func validateMarkupAnnotationPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// IRT, optional, (in reply to) dict, since V1.5
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	if err := validateIRTEntry(xRefTable, d, dictName, "IRT", OPTIONAL, sinceVersion); err != nil {
		return err
	}

	// Subj, optional, text string, since V1.5
	sinceVersion = model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	if _, err := validateStringEntry(xRefTable, d, dictName, "Subj", OPTIONAL, sinceVersion, nil); err != nil {
		return err
	}

	// RT, optional, name, since V1.6
	validate := func(s string) bool { return s == "R" || s == "Group" }
	if _, err := validateNameEntry(xRefTable, d, dictName, "RT", OPTIONAL, model.V16, validate); err != nil {
		return err
	}

	// IT, optional, name, since V1.6
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	if _, err := validateNameEntry(xRefTable, d, dictName, "IT", OPTIONAL, sinceVersion, nil); err != nil {
		return err
	}

	// ExData, optional, dict, since V1.7
	d1, err := validateDictEntry(xRefTable, d, dictName, "ExData", OPTIONAL, model.V17, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		if err := validateExDataDict(xRefTable, d1); err != nil {
			return err
		}
	}

	return nil
}

func validateMarkupAnnotation(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "markupAnnot"

	if err := validateMarkupAnnotationPart1(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateMarkupAnnotationPart2(xRefTable, d, dictName); err != nil {
		return err
	}

	return nil
}

func validateEntryP(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	ir, err := validateIndRefEntry(xRefTable, d, dictName, "P", required, sinceVersion)
	if err != nil || ir == nil {
		return err
	}

	// check if this indRef points to a pageDict.

	d1, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return err
	}

	if d1 == nil {
		d.Delete("P")
		return nil
	}

	_, err = validateNameEntry(xRefTable, d1, "pageDict", "Type", REQUIRED, model.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateAppearDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "AP", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateAppearanceDict(xRefTable, d1)
	}

	return err
}

func validateBorderArray(xRefTable *model.XRefTable, a types.Array) bool {
	if len(a) == 0 {
		return true
	}

	if xRefTable.Version() == model.V10 {
		return len(a) == 3
	}

	if !(len(a) == 3 || len(a) == 4) {
		return false
	}

	for i := 0; i < len(a); i++ {

		if i == 3 {
			// validate dash pattern array
			// len must be 0,1,2,3 numbers (dont'allow only 0s)
			a1, ok := a[i].(types.Array)
			if !ok {
				return xRefTable.ValidationMode == model.ValidationRelaxed
			}

			if len(a1) == 0 {
				return true
			}

			if len(a1) > 3 {
				return false
			}

			all0 := true
			for j := 0; j < len(a1); j++ {
				o, err := xRefTable.Dereference(a1[j])
				if err != nil || o == nil {
					return false
				}

				var f float64

				switch o := o.(type) {
				case types.Integer:
					f = float64(o.Value())
				case types.Float:
					f = o.Value()
				default:
					return false
				}

				if f < 0 {
					return false
				}

				if f != 0 {
					all0 = false
					break
				}

			}
			if all0 {
				return false
			}

			continue
		}

		o, err := xRefTable.Dereference(a[i])
		if err != nil || o == nil {
			return false
		}

		var f float64

		switch o := o.(type) {
		case types.Integer:
			f = float64(o.Value())
		case types.Float:
			f = o.Value()
		default:
			return false
		}

		if f < 0 {
			return false
		}
	}

	return true
}

func validateAnnotationDictGeneralPart1(xRefTable *model.XRefTable, d types.Dict, dictName string) (*types.Name, error) {
	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Annot" })
	if err != nil {
		return nil, err
	}

	// Subtype, required, name
	subtype, err := validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, model.V10, nil)
	if err != nil {
		return nil, err
	}

	// Rect, required, rectangle
	_, err = validateRectangleEntry(xRefTable, d, dictName, "Rect", REQUIRED, model.V10, nil)
	if err != nil {
		return nil, err
	}

	// Contents, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Contents", OPTIONAL, model.V10, nil)
	if err != nil {
		return nil, err
	}

	// P, optional, indRef of page dict
	err = validateEntryP(xRefTable, d, dictName, OPTIONAL, model.V10)
	if err != nil {
		return nil, err
	}

	// NM, optional, text string, since V1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "NM", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return nil, err
	}

	return subtype, nil
}

func validateAnnotationDictGeneralPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// M, optional, date string in any format, since V1.1
	if _, err := validateStringEntry(xRefTable, d, dictName, "M", OPTIONAL, model.V11, nil); err != nil {
		return err
	}

	// F, optional integer, since V1.1, annotation flags
	if _, err := validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V11, nil); err != nil {
		return err
	}

	// AP, optional, appearance dict, since V1.2
	if err := validateAppearDictEntry(xRefTable, d, dictName, OPTIONAL, model.V12); err != nil {
		return err
	}

	// AS, optional, name, since V1.2
	if _, err := validateNameEntry(xRefTable, d, dictName, "AS", OPTIONAL, model.V11, nil); err != nil {
		return err
	}

	// Border, optional, array of numbers
	obj, found := d.Find("BS")
	if !found || obj == nil || xRefTable.Version() < model.V12 {
		a, err := validateArrayEntry(xRefTable, d, dictName, "Border", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}
		if !validateBorderArray(xRefTable, a) {
			return errors.Errorf("invalid border array: %s", a)
		}
	}

	// C, optional array, of numbers, since V1.1
	if _, err := validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, model.V11, nil); err != nil {
		return err
	}

	// StructParent, optional, integer, since V1.3
	if _, err := validateIntegerEntry(xRefTable, d, dictName, "StructParent", OPTIONAL, model.V13, nil); err != nil {
		return err
	}

	return nil
}

func validateAnnotationDictGeneral(xRefTable *model.XRefTable, d types.Dict, dictName string) (*types.Name, error) {
	subType, err := validateAnnotationDictGeneralPart1(xRefTable, d, dictName)
	if err != nil {
		return nil, err
	}

	return subType, validateAnnotationDictGeneralPart2(xRefTable, d, dictName)
}

func validateAnnotationDictConcrete(xRefTable *model.XRefTable, d types.Dict, dictName string, subtype types.Name) error {

	// OC, optional, content group dict or content membership dict, since V1.5
	// Specifying the optional content properties for the annotation.
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	if err := validateOptionalContent(xRefTable, d, dictName, "OC", OPTIONAL, sinceVersion); err != nil {
		return err
	}

	// see table 169

	for k, v := range map[string]struct {
		validate     func(xRefTable *model.XRefTable, d types.Dict, dictName string) error
		sinceVersion model.Version
		markup       bool
	}{
		"Text":           {validateAnnotationDictText, model.V10, true},
		"Link":           {validateAnnotationDictLink, model.V10, false},
		"FreeText":       {validateAnnotationDictFreeText, model.V12, true}, // model.V13
		"Line":           {validateAnnotationDictLine, model.V13, true},
		"Polygon":        {validateAnnotationDictPolyLine, model.V15, true},
		"PolyLine":       {validateAnnotationDictPolyLine, model.V15, true},
		"Highlight":      {validateTextMarkupAnnotation, model.V13, true},
		"Underline":      {validateTextMarkupAnnotation, model.V13, true},
		"Squiggly":       {validateTextMarkupAnnotation, model.V14, true},
		"StrikeOut":      {validateTextMarkupAnnotation, model.V13, true},
		"Square":         {validateAnnotationDictCircleOrSquare, model.V13, true},
		"Circle":         {validateAnnotationDictCircleOrSquare, model.V13, true},
		"Stamp":          {validateAnnotationDictStamp, model.V13, true},
		"Caret":          {validateAnnotationDictCaret, model.V15, true},
		"Ink":            {validateAnnotationDictInk, model.V13, true},
		"Popup":          {validateAnnotationDictPopup, model.V12, false}, // model.V13
		"FileAttachment": {validateAnnotationDictFileAttachment, model.V13, true},
		"Sound":          {validateAnnotationDictSound, model.V12, true},
		"Movie":          {validateAnnotationDictMovie, model.V12, false},
		"Widget":         {validateAnnotationDictWidget, model.V12, false},
		"Screen":         {validateAnnotationDictScreen, model.V15, false},
		"PrinterMark":    {validateAnnotationDictPrinterMark, model.V14, false},
		"TrapNet":        {validateAnnotationDictTrapNet, model.V13, false},
		"Watermark":      {validateAnnotationDictWatermark, model.V16, false},
		"3D":             {validateAnnotationDict3D, model.V16, false},
		"Redact":         {validateAnnotationDictRedact, model.V17, true},
		"RichMedia":      {validateRichMediaAnnotation, model.V17, false},
	} {
		if subtype.Value() == k {

			err := xRefTable.ValidateVersion(k, v.sinceVersion)
			if err != nil {
				return err
			}

			if v.markup {
				err := validateMarkupAnnotation(xRefTable, d)
				if err != nil {
					return err
				}
			}

			return v.validate(xRefTable, d, k)
		}
	}

	return errors.Errorf("validateAnnotationDictConcrete: unsupported annotation subtype:%s\n", subtype)
}

func validateAnnotationDictSpecial(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// AAPL:AKExtras
	// No documentation for this PDF-Extension - this is a speculative implementation.
	return validateAAPLAKExtrasDictEntry(xRefTable, d, dictName, "AAPL:AKExtras", OPTIONAL, model.V10)
}

func validateAnnotationDict(xRefTable *model.XRefTable, d types.Dict) (isTrapNet bool, err error) {

	dictName := "annotDict"

	subtype, err := validateAnnotationDictGeneral(xRefTable, d, dictName)
	if err != nil {
		return false, err
	}

	err = validateAnnotationDictConcrete(xRefTable, d, dictName, *subtype)
	if err != nil {
		return false, err
	}

	err = validateAnnotationDictSpecial(xRefTable, d, dictName)
	if err != nil {
		return false, err
	}

	return *subtype == "TrapNet", nil
}

func validatePageAnnotations(xRefTable *model.XRefTable, d types.Dict) error {

	a, err := validateArrayEntry(xRefTable, d, "pageDict", "Annots", OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return err
	}

	if len(a) == 0 {
		return nil
	}

	// array of indrefs to annotation dicts.
	var annotsDict types.Dict

	// an optional TrapNetAnnotation has to be the final entry in this list.
	hasTrapNet := false

	pgAnnots := model.PgAnnots{}
	xRefTable.PageAnnots[xRefTable.CurPage] = pgAnnots

	for i, v := range a {

		if hasTrapNet {
			return errors.New("pdfcpu: validatePageAnnotations: corrupted page annotation list, \"TrapNet\" has to be the last entry")
		}

		var (
			ok, hasIndRef bool
			indRef        types.IndirectRef
		)

		if indRef, ok = v.(types.IndirectRef); ok {
			hasIndRef = true
			if log.ValidateEnabled() {
				log.Validate.Printf("processing annotDict %d\n", indRef.ObjectNumber)
			}
			annotsDict, err = xRefTable.DereferenceDict(indRef)
			if err != nil {
				return err
			}
			if annotsDict == nil {
				return errors.New("pdfcpu: validatePageAnnotations: corrupted annotation dict")
			}
		} else if annotsDict, ok = v.(types.Dict); !ok {
			return errors.New("pdfcpu: validatePageAnnotations: corrupted array of indrefs")
		}

		hasTrapNet, err = validateAnnotationDict(xRefTable, annotsDict)
		if err != nil {
			return err
		}

		// Collect annotations.
		ann, err := pdfcpu.Annotation(xRefTable, annotsDict)
		if err != nil {
			return err
		}

		annots, ok1 := pgAnnots[ann.Type()]
		if !ok1 {
			annots = model.Annot{}
			annots.IndRefs = &[]types.IndirectRef{}
			annots.Map = model.AnnotMap{}
			pgAnnots[ann.Type()] = annots
		}

		objNr := -i
		if hasIndRef {
			objNr = indRef.ObjectNumber.Value()
			*(annots.IndRefs) = append(*(annots.IndRefs), indRef)
		}
		annots.Map[objNr] = ann
	}

	return nil
}

func validatePagesAnnotations(xRefTable *model.XRefTable, d types.Dict, curPage int) (int, error) {

	// Get number of pages of this PDF file.
	pageCount := d.IntEntry("Count")
	if pageCount == nil {
		return curPage, errors.New("pdfcpu: validatePagesAnnotations: missing \"Count\"")
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validatePagesAnnotations: This page node has %d pages\n", *pageCount)
	}

	// Iterate over page tree.
	kidsArray := d.ArrayEntry("Kids")

	for _, v := range kidsArray {

		if v == nil {
			if log.ValidateEnabled() {
				log.Validate.Println("validatePagesAnnotations: kid is nil")
			}
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return curPage, err
		}
		if d == nil {
			return curPage, errors.New("pdfcpu: validatePagesAnnotations: pageNodeDict is null")
		}
		dictType := d.Type()
		if dictType == nil {
			return curPage, errors.New("pdfcpu: validatePagesAnnotations: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			curPage, err = validatePagesAnnotations(xRefTable, d, curPage)
			if err != nil {
				return curPage, err
			}

		case "Page":
			curPage++
			xRefTable.CurPage = curPage
			err = validatePageAnnotations(xRefTable, d)
			if err != nil {
				return curPage, err
			}

		default:
			return curPage, errors.Errorf("validatePagesAnnotations: expected dict type: %s\n", *dictType)

		}

	}

	return curPage, nil
}

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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateAAPLAKExtrasDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

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

func validateBorderEffectDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 12.5.4

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "borderEffectDict"

	// S, optional, name, S or C
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, pdf.V10, func(s string) bool { return s == "S" || s == "C" })
	if err != nil {
		return err
	}

	// I, optional, number in the range 0 to 2
	_, err = validateNumberEntry(xRefTable, d1, dictName, "I", OPTIONAL, pdf.V10, func(f float64) bool { return 0 <= f && f <= 2 }) // validation missing
	if err != nil {
		return err
	}

	return nil
}

func validateBorderStyleDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 12.5.4

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "borderStyleDict"

	// Type, optional, name, "Border"
	_, err = validateNameEntry(xRefTable, d1, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "Border" })
	if err != nil {
		return err
	}

	// W, optional, number, border width in points
	_, err = validateNumberEntry(xRefTable, d1, dictName, "W", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, name, border style
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"S", "D", "B", "I", "U", "A"}) }
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, pdf.V10, validate)
	if err != nil {
		return err
	}

	// D, optional, dash array
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "D", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) <= 2 })

	return err
}

func validateIconFitDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see table 247

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "iconFitDict"

	// SW, optional, name, A,B,S,N
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"A", "B", "S", "N"}) }
	_, err = validateNameEntry(xRefTable, d1, dictName, "SW", OPTIONAL, pdf.V10, validate)
	if err != nil {
		return err
	}

	// S, optional, name, A,P
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, pdf.V10, func(s string) bool { return s == "A" || s == "P" })
	if err != nil {
		return err
	}

	// A,optional, array of 2 numbers between 0.0 and 1.0
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "A", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FB, optional, bool, since V1.5
	_, err = validateBooleanEntry(xRefTable, d1, dictName, "FB", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateAppearanceCharacteristicsDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 12.5.6.19

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "appCharDict"

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, d1, dictName, "R", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// BC, optional, array of numbers, len=0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "BC", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// BG, optional, array of numbers between 0.0 and 0.1, len=0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "BG", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// CA, optional, text string
	_, err = validateStringEntry(xRefTable, d1, dictName, "CA", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// RC, optional, text string
	_, err = validateStringEntry(xRefTable, d1, dictName, "RC", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// AC, optional, text string
	_, err = validateStringEntry(xRefTable, d1, dictName, "AC", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// I, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, d1, dictName, "I", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// RI, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, d1, dictName, "RI", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// IX, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, d1, dictName, "IX", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// IF, optional, icon fit dict,
	err = validateIconFitDictEntry(xRefTable, d1, dictName, "IF", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// TP, optional, integer 0..6
	_, err = validateIntegerEntry(xRefTable, d1, dictName, "TP", OPTIONAL, pdf.V10, func(i int) bool { return 0 <= i && i <= 6 })

	return err
}

func validateAnnotationDictText(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.4

	// Open, optional, boolean
	_, err := validateBooleanEntry(xRefTable, d, dictName, "Open", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// State, optional, text string, since V1.5
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"None", "Unmarked"}) }
	state, err := validateStringEntry(xRefTable, d, dictName, "State", OPTIONAL, pdf.V15, validate)
	if err != nil {
		return err
	}

	// StateModel, text string, since V1.5
	validate = func(s string) bool { return pdf.MemberOf(s, []string{"Marked", "Review"}) }
	_, err = validateStringEntry(xRefTable, d, dictName, "StateModel", state != nil, pdf.V15, validate)

	return err
}

func validateActionOrDestination(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, sinceVersion pdf.Version) error {

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

	return validateDestination(xRefTable, obj)
}

func validateURIActionDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "URIActionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d1, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "Action" })
	if err != nil {
		return err
	}

	// S, required, name, action Type
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", REQUIRED, pdf.V10, func(s string) bool { return s == "URI" })
	if err != nil {
		return err
	}

	return validateURIActionDict(xRefTable, d1, dictName)
}

func validateAnnotationDictLink(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.5

	// A or Dest, required either or
	err := validateActionOrDestination(xRefTable, d, dictName, pdf.V11)
	if err != nil {
		return err
	}

	// H, optional, name, since V1.2
	_, err = validateNameEntry(xRefTable, d, dictName, "H", OPTIONAL, pdf.V12, nil)
	if err != nil {
		return err
	}

	// PA, optional, URI action dict, since V1.3
	err = validateURIActionDictEntry(xRefTable, d, dictName, "PA", OPTIONAL, pdf.V13)
	if err != nil {
		return err
	}

	// QuadPoints, optional, number array, len= a multiple of 8, since V1.6
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "QuadPoints", OPTIONAL, pdf.V16, func(a pdf.Array) bool { return len(a)%8 == 0 })
	if err != nil {
		return err
	}

	// BS, optional, border style dict, since V1.6
	sinceVersion := pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}

	return validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, sinceVersion)
}

func validateAnnotationDictFreeTextPart1(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, sinceVersion pdf.Version) error {

	// DA, required, string
	_, err := validateStringEntry(xRefTable, d, dictName, "DA", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Q, optional, integer, since V1.4, 0,1,2
	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Q", OPTIONAL, sinceVersion, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// RC, optional, text string or text stream, since V1.5
	sinceVersion = pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	err = validateStringOrStreamEntry(xRefTable, d, dictName, "RC", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// DS, optional, text string, since V1.5
	_, err = validateStringEntry(xRefTable, d, dictName, "DS", OPTIONAL, pdf.V15, nil)
	if err != nil {
		return err
	}

	// CL, optional, number array, since V1.6, len: 4 or 6
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "CL", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 4 || len(a) == 6 })

	return err
}

func validateAnnotationDictFreeTextPart2(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, sinceVersion pdf.Version) error {

	// IT, optional, name, since V1.6
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	validate := func(s string) bool {
		return pdf.MemberOf(s, []string{"FreeText", "FreeTextCallout", "FreeTextTypeWriter", "FreeTextTypewriter"})
	}
	_, err := validateNameEntry(xRefTable, d, dictName, "IT", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	// BE, optional, border effect dict, since V1.6
	err = validateBorderEffectDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, pdf.V15)
	if err != nil {
		return err
	}

	// RD, optional, rectangle, since V1.6
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	_, err = validateRectangleEntry(xRefTable, d, dictName, "RD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BS, optional, border style dict, since V1.6
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V12
	}
	err = validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// LE, optional, name, since V1.6
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "LE", OPTIONAL, sinceVersion, nil)

	return err
}

func validateAnnotationDictFreeText(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.6

	err := validateAnnotationDictFreeTextPart1(xRefTable, d, dictName, pdf.V12) //pdf.V13
	if err != nil {
		return err
	}

	return validateAnnotationDictFreeTextPart2(xRefTable, d, dictName, pdf.V12) //pdf.V13
}

func validateEntryMeasure(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

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

func validateAnnotationDictLine(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.7

	// L, required, array of numbers, len:4
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "L", REQUIRED, pdf.V10, func(a pdf.Array) bool { return len(a) == 4 })
	if err != nil {
		return err
	}

	// BS, optional, border style dict
	err = validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// LE, optional, name array, since V1.4, len:2
	sinceVersion := pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateNameArrayEntry(xRefTable, d, dictName, "LE", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// IC, optional, number array, since V1.4, len:0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "IC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// LLE, optional, number, since V1.6, >0
	lle, err := validateNumberEntry(xRefTable, d, dictName, "LLE", OPTIONAL, pdf.V16, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// LL, required if LLE present, number, since V1.6
	_, err = validateNumberEntry(xRefTable, d, dictName, "LL", lle != nil, pdf.V16, nil)
	if err != nil {
		return err
	}

	// Cap, optional, bool, since V1.6
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Cap", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	// IT, optional, name, since V1.6
	_, err = validateNameEntry(xRefTable, d, dictName, "IT", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	// LLO, optionl, number, since V1.7, >0
	_, err = validateNumberEntry(xRefTable, d, dictName, "LLO", OPTIONAL, pdf.V17, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// CP, optional, name, since V1.7
	_, err = validateNameEntry(xRefTable, d, dictName, "CP", OPTIONAL, pdf.V17, validateCP)
	if err != nil {
		return err
	}

	// Measure, optional, measure dict, since V1.7
	err = validateEntryMeasure(xRefTable, d, dictName, OPTIONAL, pdf.V17)
	if err != nil {
		return err
	}

	// CO, optional, number array, since V1.7, len=2
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "CO", OPTIONAL, pdf.V17, func(a pdf.Array) bool { return len(a) == 2 })

	return err
}

func validateAnnotationDictCircleOrSquare(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.8

	// BS, optional, border style dict
	err := validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// IC, optional, array, since V1.4
	sinceVersion := pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "IC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BE, optional, border effect dict, since V1.5
	err = validateBorderEffectDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, pdf.V15)
	if err != nil {
		return err
	}

	// RD, optional, rectangle, since V1.5
	_, err = validateRectangleEntry(xRefTable, d, dictName, "RD", OPTIONAL, pdf.V15, nil)

	return err
}

func validateEntryIT(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	// IT, optional, name, since V1.6
	validateIntent := func(s string) bool {

		if xRefTable.Version() == pdf.V16 {
			return s == "PolygonCloud"
		}

		if xRefTable.Version() == pdf.V17 {
			if pdf.MemberOf(s, []string{"PolygonCloud", "PolyLineDimension", "PolygonDimension"}) {
				return true
			}
		}

		return false

	}

	_, err := validateNameEntry(xRefTable, d, dictName, "IT", required, sinceVersion, validateIntent)

	return err
}

func validateAnnotationDictPolyLine(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.9

	// Vertices, required, array of numbers
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "Vertices", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// LE, optional, array of 2 names, meaningful only for polyline annotations.
	if dictName == "PolyLine" {
		_, err = validateNameArrayEntry(xRefTable, d, dictName, "LE", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })
		if err != nil {
			return err
		}
	}

	// BS, optional, border style dict
	err = validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// IC, optional, array of numbers [0.0 .. 1.0], len:1,3,4
	ensureArrayLength := func(a pdf.Array, lengths ...int) bool {
		for _, length := range lengths {
			if len(a) == length {
				return true
			}
		}
		return false
	}
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "IC", OPTIONAL, pdf.V14, func(a pdf.Array) bool { return ensureArrayLength(a, 1, 3, 4) })
	if err != nil {
		return err
	}

	// BE, optional, border effect dict, meaningful only for polygon annotations
	if dictName == "Polygon" {
		err = validateBorderEffectDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, pdf.V10)
		if err != nil {
			return err
		}
	}

	return validateEntryIT(xRefTable, d, dictName, OPTIONAL, pdf.V16)
}

func validateTextMarkupAnnotation(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.10

	required := REQUIRED
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		required = OPTIONAL
	}
	// QuadPoints, required, number array, len: a multiple of 8
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "QuadPoints", required, pdf.V10, func(a pdf.Array) bool { return len(a)%8 == 0 })

	return err
}

func validateAnnotationDictStamp(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.12

	// Name, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, pdf.V10, nil)

	return err
}

func validateAnnotationDictCaret(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.11

	// RD, optional, rectangle, since V1.5
	_, err := validateRectangleEntry(xRefTable, d, dictName, "RD", OPTIONAL, pdf.V15, nil)
	if err != nil {
		return err
	}

	// Sy, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Sy", OPTIONAL, pdf.V10, func(s string) bool { return s == "P" || s == "None" })

	return err
}

func validateAnnotationDictInk(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.13

	// InkList, required, array of stroked path arrays
	_, err := validateArrayArrayEntry(xRefTable, d, dictName, "InkList", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// BS, optional, border style dict
	return validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, pdf.V10)
}

func validateAnnotationDictPopup(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.14

	// Parent, optional, dict indRef
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "Parent", OPTIONAL, pdf.V10)
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
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Open", OPTIONAL, pdf.V10, nil)

	return err
}

func validateAnnotationDictFileAttachment(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.15

	// FS, required, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "FS", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, pdf.V10, nil)

	return err
}

func validateAnnotationDictSound(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.16

	// Sound, required, stream dict
	err := validateSoundDictEntry(xRefTable, d, dictName, "Sound", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", OPTIONAL, pdf.V10, nil)

	return err
}

func validateMovieDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "movieDict"

	// F, required, file specification
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// Aspect, optional, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "Ascpect", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// Rotate, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Rotate", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Poster, optional boolean or stream
	return validateBooleanOrStreamEntry(xRefTable, d, dictName, "Poster", OPTIONAL, pdf.V10)
}

func validateAnnotationDictMovie(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.17 Movie Annotations
	// 13.4 Movies
	// The features described in this sub-clause are obsolescent and their use is no longer recommended.
	// They are superseded by the general multimedia framework described in 13.2, “Multimedia.”

	// T, optional, text string
	_, err := validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Movie, required, movie dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "Movie", REQUIRED, pdf.V10, nil)
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
			case pdf.Boolean:
				// no further processing

			case pdf.Dict:
				err = validateMovieActivationDict(xRefTable, o)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func validateAnnotationDictWidget(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.19

	// H, optional, name
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"N", "I", "O", "P", "T", "A"}) }
	_, err := validateNameEntry(xRefTable, d, dictName, "H", OPTIONAL, pdf.V10, validate)
	if err != nil {
		return err
	}

	// MK, optional, dict
	// An appearance characteristics dictionary that shall be used in constructing
	// a dynamic appearance stream specifying the annotation’s visual presentation on the page.dict
	err = validateAppearanceCharacteristicsDictEntry(xRefTable, d, dictName, "MK", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// A, optional, dict, since V1.1
	// An action that shall be performed when the annotation is activated.
	d1, err := validateDictEntry(xRefTable, d, dictName, "A", OPTIONAL, pdf.V11, nil)
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
	err = validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, pdf.V12, "fieldOrAnnot")
	if err != nil {
		return err
	}

	// BS, optional, border style dict, since V1.2
	// A border style dictionary specifying the width and dash pattern
	// that shall be used in drawing the annotation’s border.
	validateBorderStyleDict(xRefTable, d, dictName, "BS", OPTIONAL, pdf.V12)
	if err != nil {
		return err
	}

	// Parent, dict, required if one of multiple children in a field.
	// An indirect reference to the widget annotation’s parent field.
	_, err = validateIndRefEntry(xRefTable, d, dictName, "Parent", OPTIONAL, pdf.V10)

	return err
}

func validateAnnotationDictScreen(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.18

	// T, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "T", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// MK, optional, appearance characteristics dict
	err = validateAppearanceCharacteristicsDictEntry(xRefTable, d, dictName, "MK", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// A, optional, action dict, since V1.0
	d1, err := validateDictEntry(xRefTable, d, dictName, "A", OPTIONAL, pdf.V10, nil)
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
	return validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, pdf.V12, "fieldOrAnnot")
}

func validateAnnotationDictPrinterMark(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.20

	// MN, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "MN", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// F, required integer, since V1.1, annotation flags
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V11, nil)
	if err != nil {
		return err
	}

	// AP, required, appearance dict, since V1.2
	return validateAppearDictEntry(xRefTable, d, dictName, REQUIRED, pdf.V12)
}

func validateAnnotationDictTrapNet(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.21

	// LastModified, optional, date
	_, err := validateDateEntry(xRefTable, d, dictName, "LastModified", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// Version, optional, array
	_, err = validateArrayEntry(xRefTable, d, dictName, "Version", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// AnnotStates, optional, array of names
	_, err = validateNameArrayEntry(xRefTable, d, dictName, "AnnotStates", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// FontFauxing, optional, font dict array
	validateFontDictArray := func(a pdf.Array) bool {

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

	_, err = validateArrayEntry(xRefTable, d, dictName, "FontFauxing", OPTIONAL, pdf.V10, validateFontDictArray)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V11, nil)

	return err
}

func validateAnnotationDictWatermark(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.22

	// FixedPrint, optional, dict

	validateFixedPrintDict := func(d pdf.Dict) bool {

		dictName := "fixedPrintDict"

		// Type, required, name
		_, err := validateNameEntry(xRefTable, d, dictName, "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "FixedPrint" })
		if err != nil {
			return false
		}

		// Matrix, optional, integer array, length = 6
		_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "Matrix", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a) == 6 })
		if err != nil {
			return false
		}

		// H, optional, number
		_, err = validateNumberEntry(xRefTable, d, dictName, "H", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return false
		}

		// V, optional, number
		_, err = validateNumberEntry(xRefTable, d, dictName, "V", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return false
		}

		return true
	}

	_, err := validateDictEntry(xRefTable, d, dictName, "FixedPrint", OPTIONAL, pdf.V10, validateFixedPrintDict)

	return err
}

func validateAnnotationDict3D(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 13.6.2

	// AP with entry N, required

	// 3DD, required, 3D stream or 3D reference dict
	err := validateStreamDictOrDictEntry(xRefTable, d, dictName, "3DD", REQUIRED, pdf.V16)
	if err != nil {
		return err
	}

	// 3DV, optional, various
	_, err = validateEntry(xRefTable, d, dictName, "3DV", OPTIONAL, pdf.V16)
	if err != nil {
		return err
	}

	// 3DA, optional, activation dict
	_, err = validateDictEntry(xRefTable, d, dictName, "3DA", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	// 3DI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "3DI", OPTIONAL, pdf.V16, nil)

	return err
}

func validateEntryIC(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	// IC, optional, number array, length:3 [0.0 .. 1.0]
	validateICArray := func(a pdf.Array) bool {

		if len(a) != 3 {
			return false
		}

		for _, v := range a {

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return false
			}

			switch o := o.(type) {
			case pdf.Integer:
				if o < 0 || o > 1 {
					return false
				}

			case pdf.Float:
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

func validateAnnotationDictRedact(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// see 12.5.6.23

	// QuadPoints, optional, len: a multiple of 8
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "QuadPoints", OPTIONAL, pdf.V10, func(a pdf.Array) bool { return len(a)%8 == 0 })
	if err != nil {
		return err
	}

	// IC, optional, number array, length:3 [0.0 .. 1.0]
	err = validateEntryIC(xRefTable, d, dictName, OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// RO, optional, stream
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "RO", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// OverlayText, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "OverlayText", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Repeat, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Repeat", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// DA, required, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "DA", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Q, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Q", OPTIONAL, pdf.V10, nil)

	return err
}

func validateRichMediaAnnotation(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {
	// TODO See extension level 3.
	return nil
}

func validateExDataDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "ExData"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "ExData" })
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, pdf.V10, func(s string) bool { return s == "Markup3D" })

	return err
}

func validatePopupEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V12
	}
	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {

		_, err = validateNameEntry(xRefTable, d1, dictName, "Subtype", REQUIRED, pdf.V10, func(s string) bool { return s == "Popup" })
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

func validateIRTEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

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

func validateMarkupAnnotation(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "markupAnnot"

	// T, optional, text string, since V1.1
	_, err := validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return err
	}

	// Popup, optional, dict, since V1.3
	err = validatePopupEntry(xRefTable, d, dictName, "Popup", OPTIONAL, pdf.V13)
	if err != nil {
		return err
	}

	// CA, optional, number, since V1.4
	_, err = validateNumberEntry(xRefTable, d, dictName, "CA", OPTIONAL, pdf.V14, nil)
	if err != nil {
		return err
	}

	// RC, optional, text string or stream, since V1.5
	err = validateStringOrStreamEntry(xRefTable, d, dictName, "RC", OPTIONAL, pdf.V15)
	if err != nil {
		return err
	}

	// CreationDate, optional, date, since V1.5
	_, err = validateDateEntry(xRefTable, d, dictName, "CreationDate", OPTIONAL, pdf.V15)
	if err != nil {
		return err
	}

	// IRT, optional, (in reply to) dict, since V1.5
	err = validateIRTEntry(xRefTable, d, dictName, "IRT", OPTIONAL, pdf.V15)
	if err != nil {
		return err
	}

	// Subj, optional, text string, since V1.5
	sinceVersion := pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "Subj", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RT, optional, name, since V1.6
	validate := func(s string) bool { return s == "R" || s == "Group" }
	_, err = validateNameEntry(xRefTable, d, dictName, "RT", OPTIONAL, pdf.V16, validate)
	if err != nil {
		return err
	}

	// IT, optional, name, since V1.6
	_, err = validateNameEntry(xRefTable, d, dictName, "IT", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	// ExData, optional, dict, since V1.7
	d1, err := validateDictEntry(xRefTable, d, dictName, "ExData", OPTIONAL, pdf.V17, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateExDataDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateEntryP(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

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
		return errors.Errorf("validateEntryP: entry \"P\" (obj#%d) is nil", ir.ObjectNumber)
	}

	_, err = validateNameEntry(xRefTable, d1, "pageDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateAppearDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "AP", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateAppearanceDict(xRefTable, d1)
	}

	return err
}

func validateBorderArrayLength(a pdf.Array) bool {
	return len(a) == 0 || len(a) == 3 || len(a) == 4
}

func validateAnnotationDictGeneral(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) (*pdf.Name, error) {

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "Annot" })
	if err != nil {
		return nil, err
	}

	// Subtype, required, name
	subtype, err := validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, pdf.V10, nil)
	if err != nil {
		return nil, err
	}

	// Rect, required, rectangle
	_, err = validateRectangleEntry(xRefTable, d, dictName, "Rect", REQUIRED, pdf.V10, nil)
	if err != nil {
		return nil, err
	}

	// Contents, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Contents", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return nil, err
	}

	// P, optional, indRef of page dict
	err = validateEntryP(xRefTable, d, dictName, OPTIONAL, pdf.V10)
	if err != nil {
		return nil, err
	}

	// NM, optional, text string, since V1.4
	_, err = validateStringEntry(xRefTable, d, dictName, "NM", OPTIONAL, pdf.V14, nil)
	if err != nil {
		return nil, err
	}

	// M, optional, date string in any format, since V1.1
	_, err = validateStringEntry(xRefTable, d, dictName, "M", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return nil, err
	}

	// F, optional integer, since V1.1, annotation flags
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return nil, err
	}

	// AP, optional, appearance dict, since V1.2
	err = validateAppearDictEntry(xRefTable, d, dictName, OPTIONAL, pdf.V12)
	if err != nil {
		return nil, err
	}

	// AS, optional, name, since V1.2
	_, err = validateNameEntry(xRefTable, d, dictName, "AS", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return nil, err
	}

	// Border, optional, array of numbers
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Border", OPTIONAL, pdf.V10, validateBorderArrayLength)
	if err != nil {
		return nil, err
	}

	// C, optional array, of numbers, since V1.1
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return nil, err
	}

	// StructParent, optional, integer, since V1.3
	_, err = validateIntegerEntry(xRefTable, d, dictName, "StructParent", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return nil, err
	}

	return subtype, nil
}

func validateAnnotationDictConcrete(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, subtype pdf.Name) error {

	// OC, optional, content group dict or content membership dict, since V1.5
	// Specifying the optional content properties for the annotation.
	sinceVersion := pdf.V15
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	if err := validateOptionalContent(xRefTable, d, dictName, "OC", OPTIONAL, sinceVersion); err != nil {
		return err
	}

	// see table 169

	for k, v := range map[string]struct {
		validate     func(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error
		sinceVersion pdf.Version
		markup       bool
	}{
		"Text":           {validateAnnotationDictText, pdf.V10, true},
		"Link":           {validateAnnotationDictLink, pdf.V10, false},
		"FreeText":       {validateAnnotationDictFreeText, pdf.V12, true}, // pdf.V13
		"Line":           {validateAnnotationDictLine, pdf.V13, true},
		"Polygon":        {validateAnnotationDictPolyLine, pdf.V15, true},
		"PolyLine":       {validateAnnotationDictPolyLine, pdf.V15, true},
		"Highlight":      {validateTextMarkupAnnotation, pdf.V13, true},
		"Underline":      {validateTextMarkupAnnotation, pdf.V13, true},
		"Squiggly":       {validateTextMarkupAnnotation, pdf.V14, true},
		"StrikeOut":      {validateTextMarkupAnnotation, pdf.V13, true},
		"Square":         {validateAnnotationDictCircleOrSquare, pdf.V13, true},
		"Circle":         {validateAnnotationDictCircleOrSquare, pdf.V13, true},
		"Stamp":          {validateAnnotationDictStamp, pdf.V13, true},
		"Caret":          {validateAnnotationDictCaret, pdf.V15, true},
		"Ink":            {validateAnnotationDictInk, pdf.V13, true},
		"Popup":          {validateAnnotationDictPopup, pdf.V12, false}, // pdf.V13
		"FileAttachment": {validateAnnotationDictFileAttachment, pdf.V13, true},
		"Sound":          {validateAnnotationDictSound, pdf.V12, true},
		"Movie":          {validateAnnotationDictMovie, pdf.V12, false},
		"Widget":         {validateAnnotationDictWidget, pdf.V12, false},
		"Screen":         {validateAnnotationDictScreen, pdf.V15, false},
		"PrinterMark":    {validateAnnotationDictPrinterMark, pdf.V14, false},
		"TrapNet":        {validateAnnotationDictTrapNet, pdf.V13, false},
		"Watermark":      {validateAnnotationDictWatermark, pdf.V16, false},
		"3D":             {validateAnnotationDict3D, pdf.V16, false},
		"Redact":         {validateAnnotationDictRedact, pdf.V17, true},
		"RichMedia":      {validateRichMediaAnnotation, pdf.V17, false},
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

func validateAnnotationDictSpecial(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// AAPL:AKExtras
	// No documentation for this PDF-Extension - this is a speculative implementation.
	return validateAAPLAKExtrasDictEntry(xRefTable, d, dictName, "AAPL:AKExtras", OPTIONAL, pdf.V10)
}

func validateAnnotationDict(xRefTable *pdf.XRefTable, d pdf.Dict) (isTrapNet bool, err error) {

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

func validatePageAnnotations(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	a, err := validateArrayEntry(xRefTable, d, "pageDict", "Annots", OPTIONAL, pdf.V10, nil)
	if err != nil || a == nil {
		return err
	}

	// array of indrefs to annotation dicts.
	var annotsDict pdf.Dict

	// an optional TrapNetAnnotation has to be the final entry in this list.
	hasTrapNet := false

	var i int

	if len(a) == 0 {
		return nil
	}

	pgAnnots := pdf.PgAnnots{}
	xRefTable.PageAnnots[xRefTable.CurPage] = pgAnnots

	for _, v := range a {

		if hasTrapNet {
			return errors.New("pdfcpu: validatePageAnnotations: corrupted page annotation list, \"TrapNet\" has to be the last entry")
		}

		var (
			ok, hasIndRef bool
			ir            pdf.IndirectRef
		)

		if ir, ok = v.(pdf.IndirectRef); ok {
			hasIndRef = true
			log.Validate.Printf("processing annotDict %d\n", ir.ObjectNumber)
			annotsDict, err = xRefTable.DereferenceDict(ir)
			if err != nil || annotsDict == nil {
				return errors.New("pdfcpu: validatePageAnnotations: corrupted annotation dict")
			}
		} else if annotsDict, ok = v.(pdf.Dict); !ok {
			return errors.New("pdfcpu: validatePageAnnotations: corrupted array of indrefs")
		}

		hasTrapNet, err = validateAnnotationDict(xRefTable, annotsDict)
		if err != nil {
			return err
		}

		// Collect annotations.
		ann, err := xRefTable.Annotation(annotsDict)
		if err != nil {
			return err
		}

		annots, ok1 := pgAnnots[ann.Type()]
		if !ok1 {
			annots = pdf.AnnotMap{}
			pgAnnots[ann.Type()] = annots
		}

		var k string
		if hasIndRef {
			k = ir.ObjectNumber.String()
		} else {
			k = fmt.Sprintf("?%d", i)
			i++
		}
		annots[k] = ann
	}

	return nil
}

func validatePagesAnnotations(xRefTable *pdf.XRefTable, d pdf.Dict, curPage int) (int, error) {

	// Get number of pages of this PDF file.
	pageCount := d.IntEntry("Count")
	if pageCount == nil {
		return curPage, errors.New("pdfcpu: validatePagesAnnotations: missing \"Count\"")
	}

	log.Validate.Printf("validatePagesAnnotations: This page node has %d pages\n", *pageCount)

	// Iterate over page tree.
	kidsArray := d.ArrayEntry("Kids")

	for _, v := range kidsArray {

		if v == nil {
			log.Validate.Println("validatePagesAnnotations: kid is nil")
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

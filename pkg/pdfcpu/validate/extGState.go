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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// see 8.4.5 Graphics State Parameter Dictionaries

func validateBlendMode(s string) bool {

	// see 11.3.5; table 136

	return pdf.MemberOf(s, []string{"None", "Normal", "Compatible", "Multiply", "Screen", "Overlay", "Darken", "Lighten",
		"ColorDodge", "ColorBurn", "HardLight", "SoftLight", "Difference", "Exclusion",
		"Hue", "Saturation", "Color", "Luminosity"})
}

func validateLineDashPatternEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil || a == nil {
		return err
	}

	_, err = validateIntegerArray(xRefTable, a[0])
	if err != nil {
		return err
	}

	_, err = validateInteger(xRefTable, a[1], nil)

	return err
}

func validateBG2Entry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if s != "Default" {
			err = errors.New("pdfcpu: validateBG2Entry: corrupt name")
		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		err = errors.Errorf("pdfcpu: validateBG2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateUCR2Entry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if s != "Default" {
			err = errors.New("pdfcpu: writeUCR2Entry: corrupt name")
		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		err = errors.Errorf("pdfcpu: validateUCR2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateTransferFunction(xRefTable *pdf.XRefTable, o pdf.Object) (err error) {

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if s != "Identity" {
			return errors.New("pdfcpu: validateTransferFunction: corrupt name")
		}

	case pdf.Array:

		if len(o) != 4 {
			return errors.New("pdfcpu: validateTransferFunction: corrupt function array")
		}

		for _, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}
			if o == nil {
				continue
			}

			err = processFunction(xRefTable, o)
			if err != nil {
				return err
			}

		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		return errors.Errorf("validateTransferFunction: corrupt entry: %v\n", o)

	}

	return err
}

func validateTransferFunctionEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	return validateTransferFunction(xRefTable, o)
}

func validateTR(xRefTable *pdf.XRefTable, o pdf.Object) (err error) {

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if s != "Identity" {
			return errors.Errorf("pdfcpu: validateTR: corrupt name\n")
		}

	case pdf.Array:

		if len(o) != 4 {
			return errors.New("pdfcpu: validateTR: corrupt function array")
		}

		for _, o := range o {

			o, err = xRefTable.Dereference(o)
			if err != nil {
				return
			}

			if o == nil {
				continue
			}

			if o, ok := o.(pdf.Name); ok {
				s := o.Value()
				if s != "Identity" {
					return errors.Errorf("pdfcpu: validateTR: corrupt name\n")
				}
				continue
			}

			err = processFunction(xRefTable, o)
			if err != nil {
				return
			}

		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		return errors.Errorf("validateTR: corrupt entry %v\n", o)

	}

	return err
}

func validateTREntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	return validateTR(xRefTable, o)
}

func validateTR2Name(xRefTable *pdf.XRefTable, name pdf.Name) error {
	s := name.Value()
	if s != "Identity" && s != "Default" {
		return errors.Errorf("pdfcpu: validateTR2: corrupt name\n")
	}
	return nil
}

func validateTR2(xRefTable *pdf.XRefTable, o pdf.Object) (err error) {

	switch o := o.(type) {

	case pdf.Name:
		if err = validateTR2Name(xRefTable, o); err != nil {
			return err
		}

	case pdf.Array:

		if len(o) != 4 {
			return errors.New("pdfcpu: validateTR2: corrupt function array")
		}

		for _, o := range o {

			o, err = xRefTable.Dereference(o)
			if err != nil {
				return
			}

			if o == nil {
				continue
			}

			if o, ok := o.(pdf.Name); ok {
				if err = validateTR2Name(xRefTable, o); err != nil {
					return err
				}
				continue
			}

			err = processFunction(xRefTable, o)
			if err != nil {
				return
			}

		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		return errors.Errorf("validateTR2: corrupt entry %v\n", o)

	}

	return err
}

func validateTR2Entry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	return validateTR2(xRefTable, o)
}

func validateSpotFunctionEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		validateSpotFunctionName := func(s string) bool {
			return pdf.MemberOf(s, []string{
				"SimpleDot", "InvertedSimpleDot", "DoubleDot", "InvertedDoubleDot", "CosineDot",
				"Double", "InvertedDouble", "Line", "LineX", "LineY", "Round", "Ellipse", "EllipseA",
				"InvertedEllipseA", "EllipseB", "EllipseC", "InvertedEllipseC", "Square", "Cross", "Rhomboid"})
		}
		s := o.Value()
		if !validateSpotFunctionName(s) {
			return errors.Errorf("validateSpotFunctionEntry: corrupt name\n")
		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		return errors.Errorf("validateSpotFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateType1HalftoneDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "type1HalftoneDict"

	// HalftoneName, optional, string
	_, err := validateStringEntry(xRefTable, d, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Frequency, required, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "Frequency", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Angle, required, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "Angle", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SpotFunction, required, function or name
	err = validateSpotFunctionEntry(xRefTable, d, dictName, "SpotFunction", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// TransferFunction, optional, function
	err = validateTransferFunctionEntry(xRefTable, d, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, dictName, "AccurateScreens", OPTIONAL, sinceVersion, nil)

	return err
}

func validateType5HalftoneDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "type5HalftoneDict"

	_, err := validateStringEntry(xRefTable, d, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	for _, c := range []string{"Gray", "Red", "Green", "Blue", "Cyan", "Magenta", "Yellow", "Black"} {
		err = validateHalfToneEntry(xRefTable, d, dictName, c, OPTIONAL, sinceVersion)
		if err != nil {
			return err
		}
	}

	return validateHalfToneEntry(xRefTable, d, dictName, "Default", REQUIRED, sinceVersion)
}

func validateType6HalftoneStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict, sinceVersion pdf.Version) error {

	dictName := "type6HalftoneDict"

	_, err := validateStringEntry(xRefTable, sd.Dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(xRefTable, sd.Dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateType10HalftoneStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict, sinceVersion pdf.Version) error {

	dictName := "type10HalftoneDict"

	_, err := validateStringEntry(xRefTable, sd.Dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Xsquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Ysquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(xRefTable, sd.Dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateType16HalftoneStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict, sinceVersion pdf.Version) error {

	dictName := "type16HalftoneDict"

	_, err := validateStringEntry(xRefTable, sd.Dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Width2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Height2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(xRefTable, sd.Dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateHalfToneDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "halfToneDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Halftone" })
	if err != nil {
		return err
	}

	// HalftoneType, required, integer
	halftoneType, err := validateIntegerEntry(xRefTable, d, dictName, "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	switch *halftoneType {

	case 1:
		err = validateType1HalftoneDict(xRefTable, d, sinceVersion)

	case 5:
		err = validateType5HalftoneDict(xRefTable, d, sinceVersion)

	default:
		err = errors.Errorf("validateHalfToneDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	return err
}

func validateHalfToneStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict, sinceVersion pdf.Version) error {

	dictName := "writeHalfToneStreamDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Halftone" })
	if err != nil {
		return err
	}

	// HalftoneType, required, integer
	halftoneType, err := validateIntegerEntry(xRefTable, sd.Dict, dictName, "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil || halftoneType == nil {
		return err
	}

	switch *halftoneType {

	case 6:
		err = validateType6HalftoneStreamDict(xRefTable, sd, sinceVersion)

	case 10:
		err = validateType10HalftoneStreamDict(xRefTable, sd, sinceVersion)

	case 16:
		err = validateType16HalftoneStreamDict(xRefTable, sd, sinceVersion)

	default:
		err = errors.Errorf("validateHalfToneStreamDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	return err
}

func validateHalfToneEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) (err error) {

	// See 10.5

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		if o.Value() != "Default" {
			return errors.Errorf("pdfcpu: validateHalfToneEntry: undefined name: %s\n", o)
		}

	case pdf.Dict:
		err = validateHalfToneDict(xRefTable, o, sinceVersion)

	case pdf.StreamDict:
		err = validateHalfToneStreamDict(xRefTable, &o, sinceVersion)

	default:
		err = errors.New("pdfcpu: validateHalfToneEntry: corrupt (stream)dict")
	}

	return err
}

func validateBlendModeEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		_, err = xRefTable.DereferenceName(o, sinceVersion, validateBlendMode)
		if err != nil {
			return err
		}

	case pdf.Array:
		for _, o := range o {
			_, err = xRefTable.DereferenceName(o, sinceVersion, validateBlendMode)
			if err != nil {
				return err
			}
		}

	default:
		return errors.Errorf("validateBlendModeEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateSoftMaskTransferFunctionEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if s != "Identity" {
			return errors.New("pdfcpu: validateSoftMaskTransferFunctionEntry: corrupt name")
		}

	case pdf.Dict:
		err = processFunction(xRefTable, o)

	case pdf.StreamDict:
		err = processFunction(xRefTable, o)

	default:
		return errors.Errorf("pdfcpu: validateSoftMaskTransferFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateSoftMaskDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 11.6.5.2

	dictName := "softMaskDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "Mask" })
	if err != nil {
		return err
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, pdf.V10, func(s string) bool { return s == "Alpha" || s == "Luminosity" })
	if err != nil {
		return err
	}

	// G, stream, required
	// A transparency group XObject (see “Transparency Group XObjects”)
	// to be used as the source of alpha or colour values for deriving the mask.
	sd, err := validateStreamDictEntry(xRefTable, d, dictName, "G", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	if sd != nil {
		err = validateXObjectStreamDict(xRefTable, *sd)
		if err != nil {
			return err
		}
	}

	// TR (Optional) function or name
	// A function object (see “Functions”) specifying the transfer function
	// to be used in deriving the mask values.
	err = validateSoftMaskTransferFunctionEntry(xRefTable, d, dictName, "TR", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// BC, number array, optional
	// Array of component values specifying the colour to be used
	// as the backdrop against which to composite the transparency group XObject G.
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BC", OPTIONAL, pdf.V10, nil)

	return err
}

func validateSoftMaskEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 11.3.7.2 Source Shape and Opacity
	// see 11.6.4.3 Mask Shape and Opacity

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		s := o.Value()
		if !validateBlendMode(s) {
			return errors.Errorf("pdfcpu: validateSoftMaskEntry: invalid soft mask: %s\n", s)
		}

	case pdf.Dict:
		err = validateSoftMaskDict(xRefTable, o)

	default:
		err = errors.Errorf("pdfcpu: validateSoftMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateExtGStateDictPart1(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// LW, number, optional, since V1.3
	_, err := validateNumberEntry(xRefTable, d, dictName, "LW", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// LC, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, d, dictName, "LC", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// LJ, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, d, dictName, "LJ", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// ML, number, optional, since V1.3
	_, err = validateNumberEntry(xRefTable, d, dictName, "ML", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = validateLineDashPatternEntry(xRefTable, d, dictName, "D", OPTIONAL, pdf.V13)
	if err != nil {
		return err
	}

	// RI, name, optional, since V1.3
	_, err = validateNameEntry(xRefTable, d, dictName, "RI", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// OP, boolean, optional,
	_, err = validateBooleanEntry(xRefTable, d, dictName, "OP", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// op, boolean, optional, since V1.3
	_, err = validateBooleanEntry(xRefTable, d, dictName, "op", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// OPM, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, d, dictName, "OPM", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// Font, array, optional, since V1.3
	_, err = validateArrayEntry(xRefTable, d, dictName, "Font", OPTIONAL, pdf.V13, nil)

	return err
}

func validateExtGStateDictPart2(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// BG, function, optional, black-generation function, see 10.3.4
	err := validateFunctionEntry(xRefTable, d, dictName, "BG", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// BG2, function or name(/Default), optional, since V1.3
	err = validateBG2Entry(xRefTable, d, dictName, "BG2", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// UCR, function, optional, undercolor-removal function, see 10.3.4
	err = validateFunctionEntry(xRefTable, d, dictName, "UCR", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// UCR2, function or name(/Default), optional, since V1.3
	err = validateUCR2Entry(xRefTable, d, dictName, "UCR2", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// TR, function, array of 4 functions or name(/Identity), optional, see 10.4 transfer functions
	err = validateTREntry(xRefTable, d, dictName, "TR", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// TR2, function, array of 4 functions or name(/Identity,/Default), optional, since V1.3
	err = validateTR2Entry(xRefTable, d, dictName, "TR2", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// HT, dict, stream or name, optional
	// half tone dictionary or stream or /Default, see 10.5
	err = validateHalfToneEntry(xRefTable, d, dictName, "HT", OPTIONAL, pdf.V12)
	if err != nil {
		return err
	}

	// FL, number, optional, since V1.3, flatness tolerance, see 10.6.2
	_, err = validateNumberEntry(xRefTable, d, dictName, "FL", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// SM, number, optional, since V1.3, smoothness tolerance
	_, err = validateNumberEntry(xRefTable, d, dictName, "SM", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// SA, boolean, optional, see 10.6.5 Automatic Stroke Adjustment
	_, err = validateBooleanEntry(xRefTable, d, dictName, "SA", OPTIONAL, pdf.V10, nil)

	return err
}

func validateExtGStateDictPart3(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// BM, name or array, optional, since V1.4
	sinceVersion := pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	err := validateBlendModeEntry(xRefTable, d, dictName, "BM", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// SMask, dict or name, optional, since V1.4
	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	err = validateSoftMaskEntry(xRefTable, d, dictName, "SMask", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// CA, number, optional, since V1.4, current stroking alpha constant, see 11.3.7.2 and 11.6.4.4
	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateNumberEntry(xRefTable, d, dictName, "CA", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// ca, number, optional, since V1.4, same as CA but for nonstroking operations.
	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateNumberEntry(xRefTable, d, dictName, "ca", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// AIS, alpha source flag "alpha is shape", boolean, optional, since V1.4
	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateBooleanEntry(xRefTable, d, dictName, "AIS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// TK, boolean, optional, since V1.4, text knockout flag.
	_, err = validateBooleanEntry(xRefTable, d, dictName, "TK", OPTIONAL, pdf.V14, nil)

	return err
}

func validateExtGStateDict(xRefTable *pdf.XRefTable, o pdf.Object) error {

	// 8.4.5 Graphics State Parameter Dictionaries

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictName := "extGStateDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "ExtGState" })
	if err != nil {
		return err
	}

	err = validateExtGStateDictPart1(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	err = validateExtGStateDictPart2(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	return validateExtGStateDictPart3(xRefTable, d, dictName)
}

func validateExtGStateResourceDict(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Version check
	err = xRefTable.ValidateVersion("ExtGStateResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	// Iterate over extGState resource dictionary
	for _, o := range d {

		// Process extGStateDict
		err = validateExtGStateDict(xRefTable, o)
		if err != nil {
			return err
		}

	}

	return nil
}

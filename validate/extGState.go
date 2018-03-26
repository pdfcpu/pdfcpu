package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// see 8.4.5 Graphics State Parameter Dictionaries

func validateBlendMode(s string) bool {

	// see 11.3.5; table 136

	return memberOf(s, []string{"None", "Normal", "Compatible", "Multiply", "Screen", "Overlay", "Darken", "Lighten",
		"ColorDodge", "ColorBurn", "HardLight", "SoftLight", "Difference", "Exclusion",
		"Hue", "Saturation", "Color", "Luminosity"})
}

func validateLineDashPatternEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil || arr == nil {
		return err
	}

	a := *arr

	_, err = validateIntegerArray(xRefTable, a[0])
	if err != nil {
		return err
	}

	_, err = validateInteger(xRefTable, a[1], nil)

	return err
}

func validateBG2Entry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Default" {
			err = errors.New("writeBG2Entry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		err = errors.Errorf("validateBG2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateUCR2Entry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Default" {
			err = errors.New("writeUCR2Entry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		err = errors.Errorf("validateUCR2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateTransferFunction(xRefTable *types.XRefTable, obj types.PDFObject) (err error) {

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			return errors.New("validateTransferFunction: corrupt name")
		}

	case types.PDFArray:

		if len(obj) != 4 {
			return errors.New("validateTransferFunction: corrupt function array")
		}

		for _, obj := range obj {

			obj, err := xRefTable.Dereference(obj)
			if err != nil {
				return err
			}
			if obj == nil {
				continue
			}

			err = processFunction(xRefTable, obj)
			if err != nil {
				return err
			}

		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		return errors.Errorf("validateTransferFunction: corrupt entry: %v\n", obj)

	}

	return err
}

func validateTransferFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	return validateTransferFunction(xRefTable, obj)
}

func validateTR2(xRefTable *types.XRefTable, obj types.PDFObject) (err error) {

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" && s != "Default" {
			return errors.Errorf("validateTR2: corrupt name\n")
		}

	case types.PDFArray:

		if len(obj) != 4 {
			return errors.New("validateTR2: corrupt function array")
		}

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			err = processFunction(xRefTable, obj)
			if err != nil {
				return
			}

		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		return errors.Errorf("validateTR2: corrupt entry %v\n", obj)

	}

	return err
}

func validateTR2Entry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	return validateTR2(xRefTable, obj)
}

func validateSpotFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		validateSpotFunctionName := func(s string) bool {
			return memberOf(s, []string{
				"SimpleDot", "InvertedSimpleDot", "DoubleDot", "InvertedDoubleDot", "CosineDot",
				"Double", "InvertedDouble", "Line", "LineX", "LineY"})
		}
		s := obj.String()
		if !validateSpotFunctionName(s) {
			return errors.Errorf("validateSpotFunctionEntry: corrupt name\n")
		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		return errors.Errorf("validateSpotFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateType1HalftoneDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "type1HalftoneDict"

	// HalftoneName, optional, string
	_, err := validateStringEntry(xRefTable, dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Frequency, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Frequency", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Angle, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Angle", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SpotFunction, required, function
	err = validateSpotFunctionEntry(xRefTable, dict, dictName, "Spotfunction", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// TransferFunction, optional, function
	err = validateTransferFunctionEntry(xRefTable, dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AccurateScreens", OPTIONAL, sinceVersion, nil)

	return err
}

func validateType5HalftoneDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "type5HalftoneDict"

	_, err := validateStringEntry(xRefTable, dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Gray", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Red", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Green", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Blue", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Cyan", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Magenta", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Yellow", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Black", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	return validateHalfToneEntry(xRefTable, dict, dictName, "Default", REQUIRED, sinceVersion)
}

func validateType6HalftoneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) error {

	dictName := "type6HalftoneDict"

	_, err := validateStringEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(xRefTable, &dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateType10HalftoneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) error {

	dictName := "type10HalftoneDict"

	_, err := validateStringEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Xsquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Ysquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(xRefTable, &dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateType16HalftoneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) error {

	dictName := "type16HalftoneDict"

	_, err := validateStringEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Width2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Height2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return validateTransferFunctionEntry(xRefTable, &dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
}

func validateHalfToneDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "halfToneDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Halftone" })
	if err != nil {
		return err
	}

	// HalftoneType, required, integer
	halftoneType, err := validateIntegerEntry(xRefTable, dict, dictName, "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	switch *halftoneType {

	case 1:
		err = validateType1HalftoneDict(xRefTable, dict, sinceVersion)

	case 5:
		err = validateType5HalftoneDict(xRefTable, dict, sinceVersion)

	default:
		err = errors.Errorf("validateHalfToneDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	return err
}

func validateHalfToneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) error {

	dictName := "writeHalfToneStreamDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, &dict.PDFDict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Halftone" })
	if err != nil {
		return err
	}

	// HalftoneType, required, integer
	halftoneType, err := validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil || halftoneType == nil {
		return err
	}

	switch *halftoneType {

	case 6:
		err = validateType6HalftoneStreamDict(xRefTable, dict, sinceVersion)

	case 10:
		err = validateType10HalftoneStreamDict(xRefTable, dict, sinceVersion)

	case 16:
		err = validateType16HalftoneStreamDict(xRefTable, dict, sinceVersion)

	default:
		err = errors.Errorf("validateHalfToneStreamDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	return err
}

func validateHalfToneEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// See 10.5

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if obj.String() != "Default" {
			return errors.Errorf("validateHalfToneEntry: undefined name: %s\n", obj.String())
		}

	case types.PDFDict:
		err = validateHalfToneDict(xRefTable, &obj, sinceVersion)

	case types.PDFStreamDict:
		err = validateHalfToneStreamDict(xRefTable, &obj, sinceVersion)

	default:
		err = errors.New("validateHalfToneEntry: corrupt (stream)dict")
	}

	return err
}

func validateBlendModeEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		_, err = xRefTable.DereferenceName(obj, sinceVersion, validateBlendMode)
		if err != nil {
			return err
		}

	case types.PDFArray:
		for _, obj := range obj {
			_, err = xRefTable.DereferenceName(obj, sinceVersion, validateBlendMode)
			if err != nil {
				return err
			}
		}

	default:
		return errors.Errorf("validateBlendModeEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateSoftMaskTransferFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			return errors.New("validateSoftMaskTransferFunctionEntry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		return errors.Errorf("validateSoftMaskTransferFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateSoftMaskDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 11.6.5.2

	dictName := "softMaskDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Mask" })
	if err != nil {
		return err
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "Alpha" || s == "Luminosity" })
	if err != nil {
		return err
	}

	// G, stream, required
	// A transparency group XObject (see “Transparency Group XObjects”)
	// to be used as the source of alpha or colour values for deriving the mask.
	streamDict, err := validateStreamDictEntry(xRefTable, dict, dictName, "G", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	if streamDict != nil {
		err = validateXObjectStreamDict(xRefTable, *streamDict)
		if err != nil {
			return err
		}
	}

	// TR (Optional) function or name
	// A function object (see “Functions”) specifying the transfer function
	// to be used in deriving the mask values.
	err = validateSoftMaskTransferFunctionEntry(xRefTable, dict, dictName, "TR", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// BC, number array, optional
	// Array of component values specifying the colour to be used
	// as the backdrop against which to composite the transparency group XObject G.
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BC", OPTIONAL, types.V10, nil)

	return err
}

func validateSoftMaskEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	// see 11.3.7.2 Source Shape and Opacity
	// see 11.6.4.3 Mask Shape and Opacity

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validateBlendMode(s) {
			return errors.Errorf("validateSoftMaskEntry: invalid soft mask: %s\n", s)
		}

	case types.PDFDict:
		err = validateSoftMaskDict(xRefTable, &obj)

	default:
		err = errors.Errorf("validateSoftMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateExtGStateDictPart1(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// LW, number, optional, since V1.3
	_, err := validateNumberEntry(xRefTable, dict, dictName, "LW", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// LC, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LC", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// LJ, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LJ", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// ML, number, optional, since V1.3
	_, err = validateNumberEntry(xRefTable, dict, dictName, "ML", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = validateLineDashPatternEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	// RI, name, optional, since V1.3
	_, err = validateNameEntry(xRefTable, dict, dictName, "RI", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// OP, boolean, optional,
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "OP", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// op, boolean, optional, since V1.3
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "op", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// OPM, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "OPM", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// Font, array, optional, since V1.3
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Font", OPTIONAL, types.V13, nil)

	return err
}

func validateExtGStateDictPart2(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// BG, function, optional, black-generation function, see 10.3.4
	err := validateFunctionEntry(xRefTable, dict, dictName, "BG", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// BG2, function or name(/Default), optional, since V1.3
	err = validateBG2Entry(xRefTable, dict, dictName, "BG2", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// UCR, function, optional, undercolor-removal function, see 10.3.4
	err = validateFunctionEntry(xRefTable, dict, dictName, "UCR", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// UCR2, function or name(/Default), optional, since V1.3
	err = validateUCR2Entry(xRefTable, dict, dictName, "UCR2", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// TR, function, array of 4 functions or name(/Identity), optional, see 10.4 transfer functions
	err = validateTransferFunctionEntry(xRefTable, dict, dictName, "TR", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// TR2, function, array of 4 functions or name(/Identity,/Default), optional, since V1.3
	err = validateTR2Entry(xRefTable, dict, dictName, "TR2", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// HT, dict, stream or name, optional
	// half tone dictionary or stream or /Default, see 10.5
	err = validateHalfToneEntry(xRefTable, dict, dictName, "HT", OPTIONAL, types.V12)
	if err != nil {
		return err
	}

	// FL, number, optional, since V1.3, flatness tolerance, see 10.6.2
	_, err = validateNumberEntry(xRefTable, dict, dictName, "FL", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// SM, number, optional, since V1.3, smoothness tolerance
	_, err = validateNumberEntry(xRefTable, dict, dictName, "SM", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// SA, boolean, optional, see 10.6.5 Automatic Stroke Adjustment
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "SA", OPTIONAL, types.V10, nil)

	return err
}

func validateExtGStateDictPart3(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// BM, name or array, optional, since V1.4
	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	err := validateBlendModeEntry(xRefTable, dict, dictName, "BM", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// SMask, dict or name, optional, since V1.4
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	err = validateSoftMaskEntry(xRefTable, dict, dictName, "SMask", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// CA, number, optional, since V1.4, current stroking alpha constant, see 11.3.7.2 and 11.6.4.4
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberEntry(xRefTable, dict, dictName, "CA", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// ca, number, optional, since V1.4, same as CA but for nonstroking operations.
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberEntry(xRefTable, dict, dictName, "ca", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// AIS, alpha source flag "alpha is shape", boolean, optional, since V1.4
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AIS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// TK, boolean, optional, since V1.4, text knockout flag.
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "TK", OPTIONAL, types.V14, nil)

	return err
}

func validateExtGStateDict(xRefTable *types.XRefTable, obj types.PDFObject) error {

	// 8.4.5 Graphics State Parameter Dictionaries

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	dictName := "extGStateDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "ExtGState" })
	if err != nil {
		return err
	}

	err = validateExtGStateDictPart1(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	err = validateExtGStateDictPart2(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	return validateExtGStateDictPart3(xRefTable, dict, dictName)
}

func validateExtGStateResourceDict(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Version check
	err = xRefTable.ValidateVersion("ExtGStateResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	// Iterate over extGState resource dictionary
	for _, obj := range dict.Dict {

		// Process extGStateDict
		err = validateExtGStateDict(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	return nil
}

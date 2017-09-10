package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateLineDashPatternEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validateLineDashPatternEntry begin: entry=%s ***\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	if arr == nil {
		// optional and nil or already written
		logInfoValidate.Println("validateLineDashPatternEntry end")
		return
	}

	a := *arr

	// dash array (user space units)
	array, ok := a[0].(types.PDFArray)
	if !ok {
		return errors.Errorf("validateLineDashPatternEntry: dict=%s entry \"%s\" corrupt dash array: %v", dictName, entryName, a)
	}

	_, err = validateIntegerArray(xRefTable, array)
	if err != nil {
		return
	}

	_, err = validateInteger(xRefTable, a[1], nil)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateLineDashPatternEntry end: entry=%s ***\n", entryName)

	return
}

func validateBG2Entry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateBG2Entry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateBG2Entry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateBG2Entry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateBG2Entry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateBG2Entry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateBG2Entry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
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

	logInfoValidate.Println("*** validateBG2Entry end ***")

	return
}

func validateUCR2Entry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateUCR2Entry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateUCR2Entry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateUCR2Entry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateUCR2Entry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateUCR2Entry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateUCR2Entry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
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

	logInfoValidate.Println("*** validateUCR2Entry end ***")

	return
}

func validateTransferFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateTransferFunctionEntry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateTransferFunctionEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateTransferFunctionEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateTransferFunctionEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateTransferFunctionEntry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateTransferFunctionEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			err = errors.New("validateTransferFunctionEntry: corrupt name")
		}

	case types.PDFArray:

		if len(obj) != 4 {
			return errors.New("validateTransferFunctionEntry: corrupt function array")
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
		err = errors.Errorf("validateTransferFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Printf("*** validateTransferFunctionEntry end ***")

	return
}

func validateTR2Entry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateTR2Entry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateTR2Entry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateTR2Entry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateTR2Entry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateTR2Entry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateTR2Entry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" && s != "Default" {
			err = errors.Errorf("validateTR2Entry: corrupt name\n")
		}

	case types.PDFArray:

		if len(obj) != 4 {
			return errors.New("validateTR2Entry: corrupt function array")
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
		err = errors.Errorf("validateTR2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Printf("*** validateTR2Entry end ***")

	return
}

func validateSpotFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateSpotFunctionEntry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateSpotFunctionEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateSpotFunctionEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateSpotFunctionEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateSpotFunctionEntry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSpotFunctionEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validateSpotFunctionName(s) {
			err = errors.Errorf("validateSpotFunctionEntry: corrupt name\n")
		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)

	default:
		err = errors.Errorf("validateSpotFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateSpotFunctionEntry end ***")

	return
}

func validateType1HalftoneDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateType1HalftoneDict begin ***")

	dictName := "type1HalftoneDict"

	_, err = validateStringEntry(xRefTable, dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Frequency", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Angle", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateSpotFunctionEntry(xRefTable, dict, dictName, "Spotfunction", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	err = validateTransferFunctionEntry(xRefTable, dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AccurateScreens", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType1HalftoneDict end ***")

	return
}

func validateType5HalftoneDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validateType5HalftoneDict begin ***")

	dictName := "type5HalftoneDict"

	_, err = validateStringEntry(xRefTable, dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Gray", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Red", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Green", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Blue", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Cyan", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Magenta", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Yellow", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Black", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = validateHalfToneEntry(xRefTable, dict, dictName, "Default", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType5HalftoneDict end ***")

	return
}

func validateType6HalftoneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateType6HalftoneStreamDict begin ***")

	dictName := "type6HalftoneDict"

	_, err = validateStringEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateTransferFunctionEntry(xRefTable, &dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType6HalftoneStreamDict end ***")

	return
}

func validateType10HalftoneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateType10HalftoneStreamDict begin ***")

	dictName := "type10HalftoneDict"

	_, err = validateStringEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Xsquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Ysquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateTransferFunctionEntry(xRefTable, &dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType10HalftoneStreamDict end ***")

	return
}

func validateType16HalftoneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateType16HalftoneStreamDict begin ***")

	dictName := "type16HalftoneDict"

	_, err = validateStringEntry(xRefTable, &dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Width2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, &dict.PDFDict, dictName, "Height2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateTransferFunctionEntry(xRefTable, &dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateType16HalftoneStreamDict end ***")

	return
}

func validateHalfToneDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validateHalfToneDict begin ***")

	Type, err := validateNameEntry(xRefTable, dict, "halfToneDict", "Type", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if Type != nil && *Type != "Halftone" {
		return errors.Errorf("validateHalfToneDict: unknown \"Type\": %s\n", *Type)
	}

	halftoneType, err := validateIntegerEntry(xRefTable, dict, "halfToneDict", "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	switch *halftoneType {

	case 1:
		err = validateType1HalftoneDict(xRefTable, dict, sinceVersion)

	case 5:
		err = validateType5HalftoneDict(xRefTable, dict, sinceVersion)

	default:
		err = errors.Errorf("validateHalfToneDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	logInfoValidate.Printf("*** validateHalfToneDict end ***")

	return
}

func validateHalfToneStreamDict(xRefTable *types.XRefTable, dict *types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateHalfToneStreamDict begin ***")

	Type, err := validateNameEntry(xRefTable, &dict.PDFDict, "writeHalfToneStreamDict", "Type", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if Type != nil && *Type != "Halftone" {
		return errors.Errorf("validateHalfToneStreamDict: unknown \"Type\": %s\n", *Type)
	}

	halftoneType, err := validateIntegerEntry(xRefTable, &dict.PDFDict, "writeHalfToneStreamDict", "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil || halftoneType == nil {
		return
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

	logInfoValidate.Printf("*** validateHalfToneStreamDict end ***")

	return
}

func validateHalfToneEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// See 10.5

	logInfoValidate.Println("*** validateHalfToneEntry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateHalfToneEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateHalfToneEntry end: entry %s is nil\n", entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateHalfToneEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if obj.String() != "Default" {
			err = errors.Errorf("validateHalfToneEntry: undefined name: %s\n", obj.String())
		}

	case types.PDFDict:
		err = validateHalfToneDict(xRefTable, &obj, sinceVersion)

	case types.PDFStreamDict:
		err = validateHalfToneStreamDict(xRefTable, &obj, sinceVersion)

	default:
		err = errors.New("validateHalfToneEntry: corrupt (stream)dict")
	}

	logInfoValidate.Println("*** validateHalfToneEntry end ***")

	return
}

func validateBlendModeEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateBlendModeEntry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateBlendModeEntry: dict=%s required entry \"%s\" missing.", entryName, dictName)
		}
		logInfoValidate.Printf("validateBlendModeEntry end: \"%s\" is nil.\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateBlendModeEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		// already written
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateBlendModeEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validateBlendMode(s) {
			return errors.Errorf("validateBlendModeEntry: invalid blend mode: %s\n", s)
		}

	case types.PDFArray:
		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			name, ok := obj.(types.PDFName)
			if !ok {
				return errors.Errorf("validateBlendModeEntry: corrupt blend mode array entry\n")
			}

			s := name.String()
			if !validateBlendMode(s) {
				return errors.Errorf("validateBlendModeEntry: invalid blend mode array entry: %s\n", s)
			}
		}

	default:
		return errors.Errorf("validateBlendModeEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateBlendModeEntry end ***")

	return
}

func validateSoftMaskTransferFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateSoftMaskTransferFunctionEntry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateSoftMaskTransferFunctionEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateSoftMaskTransferFunctionEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateSoftMaskTransferFunctionEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		// already written
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSoftMaskTransferFunctionEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			return errors.New("validateSoftMaskTransferFunctionEntry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(xRefTable, obj)
		if err != nil {
			return
		}

	case types.PDFStreamDict:
		err = processFunction(xRefTable, obj)
		if err != nil {
			return
		}

	default:
		return errors.Errorf("validateSoftMaskTransferFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateSoftMaskTransferFunctionEntry end ***")

	return
}

func validateSoftMaskDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 11.6.5.2

	logInfoValidate.Println("*** validateSoftMaskDict begin ***")

	dictName := "softMaskDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Mask" })
	if err != nil {
		return err
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "Alpha" || s == "Luminosity" })
	if err != nil {
		return
	}

	// G, stream, required
	// A transparency group XObject (see “Transparency Group XObjects”)
	// to be used as the source of alpha or colour values for deriving the mask.
	streamDict, err := validateStreamDictEntry(xRefTable, dict, dictName, "G", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if streamDict != nil {
		err = validateXObjectStreamDict(xRefTable, streamDict)
		if err != nil {
			return
		}
	}

	// TR (Optional) function or name
	// A function object (see “Functions”) specifying the transfer function
	// to be used in deriving the mask values.
	err = validateSoftMaskTransferFunctionEntry(xRefTable, dict, dictName, "TR", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// BC, number array, optional
	// Array of component values specifying the colour to be used
	// as the backdrop against which to composite the transparency group XObject G.
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BC", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateSoftMaskDict end ***")

	return
}

func validateSoftMaskEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 11.3.7.2 Source Shape and Opacity
	// see 11.6.4.3 Mask Shape and Opacity

	logInfoValidate.Println("*** validateSoftMaskEntry begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateSoftMaskEntry: dict=%s required entry \"%s\" missing.", entryName, dictName)
		}
		logInfoValidate.Printf("validateSoftMaskEntry end: \"%s\" is nil.\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateSoftMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateSoftMaskEntry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSoftMaskEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validateBlendMode(s) {
			err = errors.Errorf("validateSoftMaskEntry: invalid soft mask: %s\n", s)
		}

	case types.PDFDict:
		err = validateSoftMaskDict(xRefTable, &obj)

	default:
		err = errors.Errorf("validateSoftMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateSoftMaskEntry end ***")

	return
}

func validateExtGStateDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// 8.4.5 Graphics State Parameter Dictionaries

	logInfoValidate.Println("*** validateExtGStateDict begin ***")

	if dict.Type() != nil && *dict.Type() != "ExtGState" {
		return errors.New("writeExtGStateDict: corrupt dict type")
	}

	dictName := "extGStateDict"

	// LW, number, optional, since V1.3
	_, err = validateNumberEntry(xRefTable, dict, dictName, "LW", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// LC, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LC", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// LJ, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "LJ", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// ML, number, optional, since V1.3
	_, err = validateNumberEntry(xRefTable, dict, dictName, "ML", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = validateLineDashPatternEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// RI, name, optional, since V1.3
	_, err = validateNameEntry(xRefTable, dict, dictName, "RI", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OP, boolean, optional,
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "OP", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// op, boolean, optional, since V1.3
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "op", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OPM, integer, optional, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "OPM", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// Font, array, optional, since V1.3
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Font", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// BG, function, optional, black-generation function, see 10.3.4
	err = validateFunctionEntry(xRefTable, dict, dictName, "BG", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// BG2, function or name(/Default), optional, since V1.3
	err = validateBG2Entry(xRefTable, dict, dictName, "BG2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// UCR, function, optional, undercolor-removal function, see 10.3.4
	err = validateFunctionEntry(xRefTable, dict, dictName, "UCR", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// UCR2, function or name(/Default), optional, since V1.3
	err = validateUCR2Entry(xRefTable, dict, dictName, "UCR2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// TR, function, array of 4 functions or name(/Identity), optional, see 10.4 transfer functions
	err = validateTransferFunctionEntry(xRefTable, dict, dictName, "TR", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// TR2, function, array of 4 functions or name(/Identity,/Default), optional, since V1.3
	err = validateTR2Entry(xRefTable, dict, dictName, "TR2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// HT, dict, stream or name, optional
	// half tone dictionary or stream or /Default, see 10.5
	err = validateHalfToneEntry(xRefTable, dict, dictName, "HT", OPTIONAL, types.V12)
	if err != nil {
		return
	}

	// FL, number, optional, since V1.3, flatness tolerance, see 10.6.2
	_, err = validateNumberEntry(xRefTable, dict, dictName, "FL", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// SM, number, optional, since V1.3, smoothness tolerance
	_, err = validateNumberEntry(xRefTable, dict, dictName, "SM", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// SA, boolean, optional, see 10.6.5 Automatic Stroke Adjustment
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "SA", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// BM, name or array, optional, since V1.4
	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	err = validateBlendModeEntry(xRefTable, dict, dictName, "BM", OPTIONAL, sinceVersion)
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
		return
	}

	// CA, number, optional, since V1.4, current stroking alpha constant, see 11.3.7.2 and 11.6.4.4
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberEntry(xRefTable, dict, dictName, "CA", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// ca, number, optional, since V1.4, same as CA but for nonstroking operations.
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberEntry(xRefTable, dict, dictName, "ca", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// AIS, alpha source flag "alpha is shape", boolean, optional, since V1.4
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "AIS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// TK, boolean, optional, since V1.4, text knockout flag.
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "TK", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateExtGStateDict end ***")

	return
}

func validateExtGStateResourceDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateExtGStateResourceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("*** validateExtGStateResourceDict end: object is nil. ***")
		return
	}

	// Iterate over extGState resource dictionary
	for _, obj := range dict.Dict {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			logInfoValidate.Println("*** validateExtGStateResourceDict end: resource object is nil. ***")
			continue
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			return errors.New("validateExtGStateResourceDict end: corrupt extGState dict")
		}

		// Process extGStateDict
		err = validateExtGStateDict(xRefTable, &dict)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateExtGStateResourceDict end ***")

	return
}

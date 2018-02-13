package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateCalGrayColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateCalGrayColorSpace begin ***")

	if len(*arr) != 2 {
		return errors.Errorf("validateCalGrayColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateCalGrayColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dict, err := xRefTable.DereferenceDict((*arr)[1])
	if err != nil {
		return err
	}
	if dict == nil {
		logInfoValidate.Println("validateCalGrayColorSpace end")
		return nil
	}

	dictName := "calGrayCSDict"

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Gamma", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateCalGrayColorSpace end ***")

	return nil
}

func validateCalRGBColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateCalRGBColorSpace begin ***")

	if len(*arr) != 2 {
		return errors.Errorf("validateCalRGBColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateCalRGBColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dict, err := xRefTable.DereferenceDict((*arr)[1])
	if err != nil {
		return err
	}
	if dict == nil {
		logInfoValidate.Println("validateCalRGBColorSpace end")
		return nil
	}

	dictName := "calRGBCSDict"

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Gamma", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 9 })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateCalRGBColorSpace end ***")

	return nil
}

func validateLabColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateLabColorSpace begin ***")

	if len(*arr) != 2 {
		return errors.Errorf("validateLabColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateLabColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	dict, err := xRefTable.DereferenceDict((*arr)[1])
	if err != nil {
		return err
	}
	if dict == nil {
		logInfoValidate.Println("validateLabColorSpace end")
		return nil
	}

	dictName := "labCSDict"

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Range", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateLabColorSpace end ***")

	return nil
}

func validateICCBasedColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	// see 8.6.5.5

	logInfoValidate.Printf("*** validateICCBasedColorSpace begin ***")

	if len(*arr) != 2 {
		return errors.Errorf("validateICCBasedColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateICCBasedColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	sd, err := validateStreamDict(xRefTable, (*arr)[1])
	if err != nil || sd == nil {
		return err
	}

	dict := sd.PDFDict
	dictName := "ICCBasedColorSpace"

	N, err := validateIntegerEntry(xRefTable, &dict, dictName, "N", REQUIRED, sinceVersion, validateICCBasedColorSpaceEntryN)
	if err != nil {
		return err
	}

	err = validateColorSpaceEntry(xRefTable, &dict, dictName, "Alternate", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Range", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 2*N.Value() })
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	err = validateMetadata(xRefTable, &dict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateICCBasedColorSpace end ***")

	return nil
}

func validateIndexedColorSpaceLookuptable(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateIndexedColorSpaceLookuptable begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Printf("validateIndexedColorSpaceLookuptable end: is nil\n")
		return nil
	}

	switch obj.(type) {

	case types.PDFStringLiteral:
		if xRefTable.Version() < types.V12 {
			err = errors.Errorf("validateIndexedColorSpaceLookuptable: string literal unsupported in version %s.\n", xRefTable.VersionString())
		}

	case types.PDFHexLiteral:
		if xRefTable.Version() < types.V12 {
			err = errors.Errorf("validateIndexedColorSpaceLookuptable: hex literal unsupported in version %s.\n", xRefTable.VersionString())
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		err = errors.Errorf("validateIndexedColorSpaceLookuptable: invalid type\n")

	}

	logInfoValidate.Println("*** validateIndexedColorSpaceLookuptable end ***")

	return err
}

func validateIndexedColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	// see 8.6.6.3

	logInfoValidate.Printf("validateIndexedColorSpace begin ***")

	if len(*arr) != 4 {
		return errors.Errorf("validateIndexedColorSpace: invalid array length %d (expected 4) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateIndexedColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// arr[1] base: basecolorspace
	err := validateColorSpace(xRefTable, (*arr)[1], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[2] hival: 0 <= int <= 255
	_, err = validateInteger(xRefTable, (*arr)[2], func(i int) bool { return i >= 0 && i <= 255 })
	if err != nil {
		return err
	}

	// arr[3] lookup: stream or byte string, since V1.2
	err = validateIndexedColorSpaceLookuptable(xRefTable, (*arr)[3], sinceVersion)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateIndexedColorSpace end ***")

	return nil
}

func validatePatternColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	logInfoValidate.Printf("*** validatePatternColorSpace begin ***")

	if len(*arr) < 1 || len(*arr) > 2 {
		return errors.Errorf("validatePatternColorSpace: invalid array length %d (expected 1 or 2) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validatePatternColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// 8.7.3.3: arr[1]: name of underlying color space, any cs except PatternCS
	if len(*arr) == 2 {
		err := validateColorSpace(xRefTable, (*arr)[1], ExcludePatternCS)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Printf("*** validatePatternColorSpace end ***")

	return nil
}

func validateSeparationColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	// see 8.6.6.4

	logInfoValidate.Println("*** validateSeparationColorSpace begin ***")

	if len(*arr) != 4 {
		return errors.Errorf("validateSeparationColorSpace: invalid array length %d (expected 4) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSeparationColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// arr[1]: colorant name, arbitrary
	_, err := validateName(xRefTable, (*arr)[1], nil)
	if err != nil {
		return err
	}

	// arr[2]: alternate space
	err = validateColorSpace(xRefTable, (*arr)[2], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[3]: tintTransform, function
	err = validateFunction(xRefTable, (*arr)[3])
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateSeparationColorSpace end ***")

	return nil
}

func validateDeviceNColorSpaceColorantsDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Printf("*** validateDeviceNColorSpaceColorantsDict begin ***")

	for _, obj := range dict.Dict {

		arr, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		if arr != nil {
			err = validateSeparationColorSpace(xRefTable, arr, types.V12)
			if err != nil {
				return err
			}
		}

	}

	logInfoValidate.Printf("*** validateDeviceNColorSpaceColorantsDict end ***")

	return nil
}

func validateDeviceNColorSpaceProcessDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Printf("validateDeviceNColorSpaceProcessDict begin ***")

	dictName := "DeviceNCSProcessDict"

	err := validateColorSpaceEntry(xRefTable, dict, dictName, "ColorSpace", REQUIRED, true)
	if err != nil {
		return err
	}

	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "Components", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateDeviceNColorSpaceProcessDict end ***")

	return nil
}

func validateDeviceNColorSpaceSoliditiesDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateDeviceNColorSpaceSoliditiesDict begin ***")

	for _, obj := range dict.Dict {
		_, err := validateFloat(xRefTable, obj, func(f float64) bool { return f >= 0.0 && f <= 1.0 })
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateDeviceNColorSpaceSoliditiesDict end ***")

	return nil
}

func validateDeviceNColorSpaceDotGainDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateDeviceNColorSpaceDotGainDict begin ***")

	for _, obj := range dict.Dict {
		err := validateFunction(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateDeviceNColorSpaceDotGainDict end ***")

	return nil
}

func validateDeviceNColorSpaceMixingHintsDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateDeviceNColorSpaceMixingHintsDict ***")

	dictName := "deviceNCSMixingHintsDict"

	d, err := validateDictEntry(xRefTable, dict, dictName, "Solidities", OPTIONAL, types.V11, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateDeviceNColorSpaceSoliditiesDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "PrintingOrder", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	d, err = validateDictEntry(xRefTable, dict, dictName, "DotGain", OPTIONAL, types.V11, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateDeviceNColorSpaceDotGainDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateDeviceNColorSpaceMixingHintsDict end ***")

	return nil
}

func validateDeviceNColorSpaceAttributesDict(xRefTable *types.XRefTable, obj interface{}) error {

	logInfoValidate.Println("*** validateDeviceNColorSpaceAttributesDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	dictName := "deviceNCSAttributesDict"

	_, err = validateNameEntry(xRefTable, dict, dictName, "SubType", OPTIONAL, types.V16, func(s string) bool { return s == "DeviceN" || s == "NChannel" })
	if err != nil {
		return err
	}

	d, err := validateDictEntry(xRefTable, dict, dictName, "Colorants", OPTIONAL, types.V11, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateDeviceNColorSpaceColorantsDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	sinceVersion := types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	d, err = validateDictEntry(xRefTable, dict, dictName, "Process", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateDeviceNColorSpaceProcessDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	d, err = validateDictEntry(xRefTable, dict, dictName, "MixingHints", OPTIONAL, types.V16, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateDeviceNColorSpaceMixingHintsDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateDeviceNColorSpaceAttributesDict end ***")

	return nil
}

func validateDeviceNColorSpace(xRefTable *types.XRefTable, arr *types.PDFArray, sinceVersion types.PDFVersion) error {

	// see 8.6.6.5

	logInfoValidate.Println("*** validateDeviceNColorSpace begin ***")

	if len(*arr) < 4 || len(*arr) > 5 {
		return errors.Errorf("writeDeviceNColorSpace: invalid array length %d (expected 4 or 5) \n.", len(*arr))
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("writeDeviceNColorSpace: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// arr[1]: array of names specifying the individual color components
	// length subject to implementation limit.
	_, err := validateNameArray(xRefTable, (*arr)[1])
	if err != nil {
		return err
	}

	// arr[2]: alternate space
	err = validateColorSpace(xRefTable, (*arr)[2], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[3]: tintTransform, function
	err = validateFunction(xRefTable, (*arr)[3])
	if err != nil {
		return err
	}

	// arr[4]: color space attributes dict, optional
	if len(*arr) == 5 {
		err = validateDeviceNColorSpaceAttributesDict(xRefTable, (*arr)[4])
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateDeviceNColorSpace end ***")

	return nil
}

func validateCSArray(xRefTable *types.XRefTable, arr *types.PDFArray, csName string) error {

	// see 8.6 Color Spaces

	switch csName {

	// CIE-based
	case "CalGray":
		return validateCalGrayColorSpace(xRefTable, arr, types.V11)

	case "CalRGB":
		return validateCalRGBColorSpace(xRefTable, arr, types.V11)

	case "Lab":
		return validateLabColorSpace(xRefTable, arr, types.V11)

	case "ICCBased":
		return validateICCBasedColorSpace(xRefTable, arr, types.V13)

	// Special
	case "Indexed":
		return validateIndexedColorSpace(xRefTable, arr, types.V11)

	case "Pattern":
		return validatePatternColorSpace(xRefTable, arr, types.V12)

	case "Separation":
		return validateSeparationColorSpace(xRefTable, arr, types.V12)

	case "DeviceN":
		return validateDeviceNColorSpace(xRefTable, arr, types.V13)

	default:
		return errors.Errorf("validateColorSpaceArray: undefined color space: %s\n", csName)
	}

}

func validateColorSpaceArraySubset(xRefTable *types.XRefTable, arr *types.PDFArray, cs []string) error {

	csName, ok := (*arr)[0].(types.PDFName)
	if !ok {
		return errors.New("validateColorSpaceArraySubset: corrupt Colorspace array")
	}

	for _, v := range cs {
		if csName.Value() == v {
			return validateCSArray(xRefTable, arr, v)
		}
	}

	return errors.Errorf("validateColorSpaceArraySubset: invalid color space: %s\n", csName)
}

func validateColorSpaceArray(xRefTable *types.XRefTable, arr *types.PDFArray, excludePatternCS bool) (err error) {

	// see 8.6 Color Spaces

	logInfoValidate.Println("*** validateColorSpaceArray begin ***")

	name, ok := (*arr)[0].(types.PDFName)
	if !ok {
		return errors.New("validateColorSpaceArray: corrupt Colorspace array")
	}

	switch name {

	// CIE-based
	case "CalGray":
		err = validateCalGrayColorSpace(xRefTable, arr, types.V11)

	case "CalRGB":
		err = validateCalRGBColorSpace(xRefTable, arr, types.V11)

	case "Lab":
		err = validateLabColorSpace(xRefTable, arr, types.V11)

	case "ICCBased":
		err = validateICCBasedColorSpace(xRefTable, arr, types.V13)

	// Special
	case "Indexed":
		err = validateIndexedColorSpace(xRefTable, arr, types.V11)

	case "Pattern":
		if excludePatternCS {
			return errors.New("validateColorSpaceArray: Pattern color space not allowed")
		}
		err = validatePatternColorSpace(xRefTable, arr, types.V12)

	case "Separation":
		err = validateSeparationColorSpace(xRefTable, arr, types.V12)

	case "DeviceN":
		err = validateDeviceNColorSpace(xRefTable, arr, types.V13)

	default:
		err = errors.Errorf("validateColorSpaceArray: undefined color space: %s\n", name)
	}

	if err == nil {
		logInfoValidate.Println("*** validateColorSpaceArray end ***")
	}

	return err
}

func validateColorSpace(xRefTable *types.XRefTable, obj interface{}, excludePatternCS bool) error {

	logInfoValidate.Printf("*** validateColorSpace begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Println("validateColorSpace end: resource object is nil")
		return nil
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if ok := validateDeviceColorSpaceName(obj.String()) || validateSpecialColorSpaceName(obj.String()); !ok {
			err = errors.Errorf("validateColorSpace: invalid device color space name: %v\n", obj)
		}

	case types.PDFArray:
		err = validateColorSpaceArray(xRefTable, &obj, excludePatternCS)

	default:
		err = errors.New("validateColorSpace: corrupt obj typ, must be Name or Array")

	}

	logInfoValidate.Println("*** validateColorSpace end ***")

	return err
}

func validateColorSpaceEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, excludePatternCS bool) error {

	logInfoValidate.Printf("*** validateColorSpaceEntry begin: dictName=%s ***\n", dictName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateColorSpaceEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateColorSpaceEntry end: \"%s\" is nil.\n", entryName)
		return nil
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateColorSpaceEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateColorSpaceEntry end: dictName=%s\n", dictName)
		return nil
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if ok := validateDeviceColorSpaceName(obj.String()); !ok {
			err = errors.Errorf("validateColorSpaceEntry: Name:%s\n", obj.String())
		}

	case types.PDFArray:
		err = validateColorSpaceArray(xRefTable, &obj, excludePatternCS)

	default:
		err = errors.Errorf("validateColorSpaceEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Printf("*** validateColorSpaceEntry end: dictName=%s ***\n", dictName)

	return err
}

func validateColorSpaceResourceDict(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 8.6 Color Spaces

	logInfoValidate.Println("*** validateColorSpaceResourceDict begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Println("validateColorSpaceResourceDict end: object is nil.")
		return nil
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("validateColorSpaceResourceDict: corrupt dict")
	}

	// Iterate over colorspace resource dictionary
	for key, obj := range dict.Dict {

		logInfoValidate.Printf("validateColorSpaceResourceDict: processing entry: %s\n", key)

		// Process colorspace
		err = validateColorSpace(xRefTable, obj, IncludePatternCS)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateColorSpaceResourceDict end ***")

	return nil
}

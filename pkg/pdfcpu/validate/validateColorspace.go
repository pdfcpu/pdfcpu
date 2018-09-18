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
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateDeviceColorSpaceName(s string) bool {
	return pdf.MemberOf(s, []string{pdf.DeviceGrayCS, pdf.DeviceRGBCS, pdf.DeviceCMYKCS})
}

func validateCalGrayColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	dictName := "calGrayCSDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) != 2 {
		return errors.Errorf("validateCalGrayColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	dict, err := xRefTable.DereferenceDict((*arr)[1])
	if err != nil || dict == nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, dict, dictName, "Gamma", OPTIONAL, sinceVersion, nil)

	return err
}

func validateCalRGBColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	dictName := "calRGBCSDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) != 2 {
		return errors.Errorf("validateCalRGBColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	dict, err := xRefTable.DereferenceDict((*arr)[1])
	if err != nil || dict == nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Gamma", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 9 })

	return err
}

func validateLabColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	dictName := "labCSDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) != 2 {
		return errors.Errorf("validateLabColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	dict, err := xRefTable.DereferenceDict((*arr)[1])
	if err != nil || dict == nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Range", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 4 })

	return err
}

func validateICCBasedColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	// see 8.6.5.5

	dictName := "ICCBasedColorSpace"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) != 2 {
		return errors.Errorf("validateICCBasedColorSpace: invalid array length %d (expected 2) \n.", len(*arr))
	}

	sd, err := validateStreamDict(xRefTable, (*arr)[1])
	if err != nil || sd == nil {
		return err
	}

	dict := sd.PDFDict

	validate := func(i int) bool { return pdf.IntMemberOf(i, []int{1, 3, 4}) }
	N, err := validateIntegerEntry(xRefTable, &dict, dictName, "N", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	err = validateColorSpaceEntry(xRefTable, &dict, dictName, "Alternate", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Range", OPTIONAL, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 2*N.Value() })
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	err = validateMetadata(xRefTable, &dict, OPTIONAL, pdf.V14)

	return err
}

func validateIndexedColorSpaceLookuptable(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj.(type) {

	case pdf.StringLiteral, pdf.HexLiteral:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", pdf.V12)

	case pdf.StreamDict:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", sinceVersion)

	default:
		err = errors.Errorf("validateIndexedColorSpaceLookuptable: invalid type\n")

	}

	return err
}

func validateIndexedColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	// see 8.6.6.3

	err := xRefTable.ValidateVersion("IndexedColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) != 4 {
		return errors.Errorf("validateIndexedColorSpace: invalid array length %d (expected 4) \n.", len(*arr))
	}

	// arr[1] base: base colorspace
	err = validateColorSpace(xRefTable, (*arr)[1], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[2] hival: 0 <= int <= 255
	_, err = validateInteger(xRefTable, (*arr)[2], func(i int) bool { return i >= 0 && i <= 255 })
	if err != nil {
		return err
	}

	// arr[3] lookup: stream since V1.2 or byte string
	return validateIndexedColorSpaceLookuptable(xRefTable, (*arr)[3], sinceVersion)
}

func validatePatternColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	err := xRefTable.ValidateVersion("PatternColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) < 1 || len(*arr) > 2 {
		return errors.Errorf("validatePatternColorSpace: invalid array length %d (expected 1 or 2) \n.", len(*arr))
	}

	// 8.7.3.3: arr[1]: name of underlying color space, any cs except PatternCS
	if len(*arr) == 2 {
		err := validateColorSpace(xRefTable, (*arr)[1], ExcludePatternCS)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSeparationColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	// see 8.6.6.4

	err := xRefTable.ValidateVersion("SeparationColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) != 4 {
		return errors.Errorf("validateSeparationColorSpace: invalid array length %d (expected 4) \n.", len(*arr))
	}

	// arr[1]: colorant name, arbitrary
	_, err = validateName(xRefTable, (*arr)[1], nil)
	if err != nil {
		return err
	}

	// arr[2]: alternate space
	err = validateColorSpace(xRefTable, (*arr)[2], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[3]: tintTransform, function
	return validateFunction(xRefTable, (*arr)[3])
}

func validateDeviceNColorSpaceColorantsDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	for _, obj := range dict.Dict {

		arr, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		if arr != nil {
			err = validateSeparationColorSpace(xRefTable, arr, pdf.V12)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateDeviceNColorSpaceProcessDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "DeviceNCSProcessDict"

	err := validateColorSpaceEntry(xRefTable, dict, dictName, "ColorSpace", REQUIRED, true)
	if err != nil {
		return err
	}

	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "Components", REQUIRED, pdf.V10, nil)

	return err
}

func validateDeviceNColorSpaceSoliditiesDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	for _, obj := range dict.Dict {
		_, err := validateFloat(xRefTable, obj, func(f float64) bool { return f >= 0.0 && f <= 1.0 })
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceDotGainDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	for _, obj := range dict.Dict {
		err := validateFunction(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceMixingHintsDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "deviceNCSMixingHintsDict"

	d, err := validateDictEntry(xRefTable, dict, dictName, "Solidities", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateDeviceNColorSpaceSoliditiesDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "PrintingOrder", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	d, err = validateDictEntry(xRefTable, dict, dictName, "DotGain", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateDeviceNColorSpaceDotGainDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceAttributesDict(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	dictName := "deviceNCSAttributesDict"

	_, err = validateNameEntry(xRefTable, dict, dictName, "SubType", OPTIONAL, pdf.V16, func(s string) bool { return s == "DeviceN" || s == "NChannel" })
	if err != nil {
		return err
	}

	d, err := validateDictEntry(xRefTable, dict, dictName, "Colorants", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateDeviceNColorSpaceColorantsDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	sinceVersion := pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
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

	d, err = validateDictEntry(xRefTable, dict, dictName, "MixingHints", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateDeviceNColorSpaceMixingHintsDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpace(xRefTable *pdf.XRefTable, arr *pdf.Array, sinceVersion pdf.Version) error {

	// see 8.6.6.5

	err := xRefTable.ValidateVersion("DeviceNColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(*arr) < 4 || len(*arr) > 5 {
		return errors.Errorf("writeDeviceNColorSpace: invalid array length %d (expected 4 or 5) \n.", len(*arr))
	}

	// arr[1]: array of names specifying the individual color components
	// length subject to implementation limit.
	_, err = validateNameArray(xRefTable, (*arr)[1])
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

	return nil
}

func validateCSArray(xRefTable *pdf.XRefTable, arr *pdf.Array, csName string) error {

	// see 8.6 Color Spaces

	switch csName {

	// CIE-based
	case pdf.CalGrayCS:
		return validateCalGrayColorSpace(xRefTable, arr, pdf.V11)

	case pdf.CalRGBCS:
		return validateCalRGBColorSpace(xRefTable, arr, pdf.V11)

	case pdf.LabCS:
		return validateLabColorSpace(xRefTable, arr, pdf.V11)

	case pdf.ICCBasedCS:
		return validateICCBasedColorSpace(xRefTable, arr, pdf.V13)

	// Special
	case pdf.IndexedCS:
		return validateIndexedColorSpace(xRefTable, arr, pdf.V11)

	case pdf.PatternCS:
		return validatePatternColorSpace(xRefTable, arr, pdf.V12)

	case pdf.SeparationCS:
		return validateSeparationColorSpace(xRefTable, arr, pdf.V12)

	case pdf.DeviceNCS:
		return validateDeviceNColorSpace(xRefTable, arr, pdf.V13)

	default:
		return errors.Errorf("validateColorSpaceArray: undefined color space: %s\n", csName)
	}

}

func validateColorSpaceArraySubset(xRefTable *pdf.XRefTable, arr *pdf.Array, cs []string) error {

	csName, ok := (*arr)[0].(pdf.Name)
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

func validateColorSpaceArray(xRefTable *pdf.XRefTable, arr *pdf.Array, excludePatternCS bool) (err error) {

	// see 8.6 Color Spaces

	name, ok := (*arr)[0].(pdf.Name)
	if !ok {
		return errors.New("validateColorSpaceArray: corrupt Colorspace array")
	}

	switch name {

	// CIE-based
	case pdf.CalGrayCS:
		err = validateCalGrayColorSpace(xRefTable, arr, pdf.V11)

	case pdf.CalRGBCS:
		err = validateCalRGBColorSpace(xRefTable, arr, pdf.V11)

	case pdf.LabCS:
		err = validateLabColorSpace(xRefTable, arr, pdf.V11)

	case pdf.ICCBasedCS:
		err = validateICCBasedColorSpace(xRefTable, arr, pdf.V13)

	// Special
	case pdf.IndexedCS:
		err = validateIndexedColorSpace(xRefTable, arr, pdf.V11)

	case pdf.PatternCS:
		if excludePatternCS {
			return errors.New("validateColorSpaceArray: Pattern color space not allowed")
		}
		err = validatePatternColorSpace(xRefTable, arr, pdf.V12)

	case pdf.SeparationCS:
		err = validateSeparationColorSpace(xRefTable, arr, pdf.V12)

	case pdf.DeviceNCS:
		err = validateDeviceNColorSpace(xRefTable, arr, pdf.V13)

	default:
		err = errors.Errorf("validateColorSpaceArray: undefined color space: %s\n", name)
	}

	return err
}

func validateColorSpace(xRefTable *pdf.XRefTable, obj pdf.Object, excludePatternCS bool) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Name:
		validateSpecialColorSpaceName := func(s string) bool { return pdf.MemberOf(s, []string{"Pattern"}) }
		if ok := validateDeviceColorSpaceName(obj.String()) || validateSpecialColorSpaceName(obj.String()); !ok {
			err = errors.Errorf("validateColorSpace: invalid device color space name: %v\n", obj)
		}

	case pdf.Array:
		err = validateColorSpaceArray(xRefTable, &obj, excludePatternCS)

	default:
		err = errors.New("validateColorSpace: corrupt obj typ, must be Name or Array")

	}

	return err
}

func validateColorSpaceEntry(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName string, entryName string, required bool, excludePatternCS bool) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, pdf.V10)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Name:
		if ok := validateDeviceColorSpaceName(obj.String()); !ok {
			err = errors.Errorf("validateColorSpaceEntry: Name:%s\n", obj.String())
		}

	case pdf.Array:
		err = validateColorSpaceArray(xRefTable, &obj, excludePatternCS)

	default:
		err = errors.Errorf("validateColorSpaceEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateColorSpaceResourceDict(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 8.6 Color Spaces

	// Version check
	err := xRefTable.ValidateVersion("ColorSpaceResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Iterate over colorspace resource dictionary
	for _, obj := range dict.Dict {

		// Process colorspace
		err = validateColorSpace(xRefTable, obj, IncludePatternCS)
		if err != nil {
			return err
		}
	}

	return nil
}

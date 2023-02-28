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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateDeviceColorSpaceName(s string) bool {
	return types.MemberOf(s, []string{model.DeviceGrayCS, model.DeviceRGBCS, model.DeviceCMYKCS})
}

func validateAllColorSpaceNamesExceptPattern(s string) bool {
	return types.MemberOf(s, []string{model.DeviceGrayCS, model.DeviceRGBCS, model.DeviceCMYKCS, model.CalGrayCS, model.CalRGBCS, model.LabCS, model.ICCBasedCS, model.IndexedCS, model.SeparationCS, model.DeviceNCS})
}

func validateCalGrayColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	dictName := "calGrayCSDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return errors.Errorf("validateCalGrayColorSpace: invalid array length %d (expected 2) \n.", len(a))
	}

	d, err := xRefTable.DereferenceDict(a[1])
	if err != nil || d == nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Gamma", OPTIONAL, sinceVersion, nil)

	return err
}

func validateCalRGBColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	dictName := "calRGBCSDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return errors.Errorf("validateCalRGBColorSpace: invalid array length %d (expected 2) \n.", len(a))
	}

	d, err := xRefTable.DereferenceDict(a[1])
	if err != nil || d == nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Gamma", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 9 })

	return err
}

func validateLabColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	dictName := "labCSDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return errors.Errorf("validateLabColorSpace: invalid array length %d (expected 2) \n.", len(a))
	}

	d, err := xRefTable.DereferenceDict(a[1])
	if err != nil || d == nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Range", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 4 })

	return err
}

func validateAlternateColorSpaceEntryForICC(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, excludePatternCS bool) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		if ok := validateAllColorSpaceNamesExceptPattern(o.Value()); !ok {
			err = errors.Errorf("pdfcpu: validateAlternateColorSpaceEntryForICC: invalid Name:%s\n", o.Value())
		}

	case types.Array:
		err = validateColorSpaceArray(xRefTable, o, excludePatternCS)

	default:
		err = errors.Errorf("pdfcpu: validateAlternateColorSpaceEntryForICC: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateICCBasedColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	// see 8.6.5.5

	dictName := "ICCBasedColorSpace"

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return errors.Errorf("validateICCBasedColorSpace: invalid array length %d (expected 2) \n.", len(a))
	}

	valid, err := xRefTable.IsValid(a[1].(types.IndirectRef))
	if err != nil {
		return err
	}
	if valid {
		return nil
	}

	sd, err := validateStreamDict(xRefTable, a[1])
	if err != nil || sd == nil {
		return err
	}
	if err := xRefTable.SetValid(a[1].(types.IndirectRef)); err != nil {
		return err
	}

	validate := func(i int) bool { return types.IntMemberOf(i, []int{1, 3, 4}) }
	N, err := validateIntegerEntry(xRefTable, sd.Dict, dictName, "N", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	err = validateAlternateColorSpaceEntryForICC(xRefTable, sd.Dict, dictName, "Alternate", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Range", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 2*N.Value() })
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	return validateMetadata(xRefTable, sd.Dict, OPTIONAL, model.V14)
}

func validateIndexedColorSpaceLookuptable(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case types.StringLiteral, types.HexLiteral:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", model.V12)

	case types.StreamDict:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", sinceVersion)

	default:
		err = errors.Errorf("validateIndexedColorSpaceLookuptable: invalid type\n")

	}

	return err
}

func validateIndexedColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	// see 8.6.6.3

	err := xRefTable.ValidateVersion("IndexedColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 4 {
		return errors.Errorf("validateIndexedColorSpace: invalid array length %d (expected 4) \n.", len(a))
	}

	// arr[1] base: base colorspace
	err = validateColorSpace(xRefTable, a[1], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[2] hival: 0 <= int <= 255
	_, err = validateInteger(xRefTable, a[2], func(i int) bool { return i >= 0 && i <= 255 })
	if err != nil {
		return err
	}

	// arr[3] lookup: stream since V1.2 or byte string
	return validateIndexedColorSpaceLookuptable(xRefTable, a[3], sinceVersion)
}

func validatePatternColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	err := xRefTable.ValidateVersion("PatternColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) < 1 || len(a) > 2 {
		return errors.Errorf("validatePatternColorSpace: invalid array length %d (expected 1 or 2) \n.", len(a))
	}

	// 8.7.3.3: arr[1]: name of underlying color space, any cs except PatternCS
	if len(a) == 2 {
		err := validateColorSpace(xRefTable, a[1], ExcludePatternCS)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSeparationColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	// see 8.6.6.4

	err := xRefTable.ValidateVersion("SeparationColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 4 {
		return errors.Errorf("validateSeparationColorSpace: invalid array length %d (expected 4) \n.", len(a))
	}

	// arr[1]: colorant name, arbitrary
	_, err = validateName(xRefTable, a[1], nil)
	if err != nil {
		return err
	}

	// arr[2]: alternate space
	err = validateColorSpace(xRefTable, a[2], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[3]: tintTransform, function
	return validateFunction(xRefTable, a[3])
}

func validateDeviceNColorSpaceColorantsDict(xRefTable *model.XRefTable, d types.Dict) error {

	for _, obj := range d {

		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		if a != nil {
			err = validateSeparationColorSpace(xRefTable, a, model.V12)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateDeviceNColorSpaceProcessDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "DeviceNCSProcessDict"

	err := validateColorSpaceEntry(xRefTable, d, dictName, "ColorSpace", REQUIRED, true)
	if err != nil {
		return err
	}

	_, err = validateNameArrayEntry(xRefTable, d, dictName, "Components", REQUIRED, model.V10, nil)

	return err
}

func validateDeviceNColorSpaceSoliditiesDict(xRefTable *model.XRefTable, d types.Dict) error {

	for _, obj := range d {
		_, err := validateFloat(xRefTable, obj, func(f float64) bool { return f >= 0.0 && f <= 1.0 })
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceDotGainDict(xRefTable *model.XRefTable, d types.Dict) error {

	for _, obj := range d {
		err := validateFunction(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceMixingHintsDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "deviceNCSMixingHintsDict"

	d1, err := validateDictEntry(xRefTable, d, dictName, "Solidities", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateDeviceNColorSpaceSoliditiesDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	_, err = validateNameArrayEntry(xRefTable, d, dictName, "PrintingOrder", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	d1, err = validateDictEntry(xRefTable, d, dictName, "DotGain", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateDeviceNColorSpaceDotGainDict(xRefTable, d1)
	}

	return err
}

func validateDeviceNColorSpaceAttributesDict(xRefTable *model.XRefTable, o types.Object) error {

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictName := "deviceNCSAttributesDict"

	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool { return s == "DeviceN" || s == "NChannel" })
	if err != nil {
		return err
	}

	d1, err := validateDictEntry(xRefTable, d, dictName, "Colorants", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateDeviceNColorSpaceColorantsDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	d1, err = validateDictEntry(xRefTable, d, dictName, "Process", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateDeviceNColorSpaceProcessDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	d1, err = validateDictEntry(xRefTable, d, dictName, "MixingHints", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateDeviceNColorSpaceMixingHintsDict(xRefTable, d1)
	}

	return err
}

func validateDeviceNColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {

	// see 8.6.6.5

	err := xRefTable.ValidateVersion("DeviceNColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) < 4 || len(a) > 5 {
		return errors.Errorf("writeDeviceNColorSpace: invalid array length %d (expected 4 or 5) \n.", len(a))
	}

	// arr[1]: array of names specifying the individual color components
	// length subject to implementation limit.
	_, err = validateNameArray(xRefTable, a[1])
	if err != nil {
		return err
	}

	// arr[2]: alternate space
	err = validateColorSpace(xRefTable, a[2], ExcludePatternCS)
	if err != nil {
		return err
	}

	// arr[3]: tintTransform, function
	err = validateFunction(xRefTable, a[3])
	if err != nil {
		return err
	}

	// arr[4]: color space attributes dict, optional
	if len(a) == 5 {
		err = validateDeviceNColorSpaceAttributesDict(xRefTable, a[4])
	}

	return err
}

func validateCSArray(xRefTable *model.XRefTable, a types.Array, csName string) error {

	// see 8.6 Color Spaces

	switch csName {

	// CIE-based
	case model.CalGrayCS:
		return validateCalGrayColorSpace(xRefTable, a, model.V11)

	case model.CalRGBCS:
		return validateCalRGBColorSpace(xRefTable, a, model.V11)

	case model.LabCS:
		return validateLabColorSpace(xRefTable, a, model.V11)

	case model.ICCBasedCS:
		return validateICCBasedColorSpace(xRefTable, a, model.V13)

	// Special
	case model.IndexedCS:
		return validateIndexedColorSpace(xRefTable, a, model.V11)

	case model.PatternCS:
		return validatePatternColorSpace(xRefTable, a, model.V12)

	case model.SeparationCS:
		return validateSeparationColorSpace(xRefTable, a, model.V12)

	case model.DeviceNCS:
		return validateDeviceNColorSpace(xRefTable, a, model.V13)

	default:
		return errors.Errorf("validateColorSpaceArray: undefined color space: %s\n", csName)
	}

}

func validateColorSpaceArraySubset(xRefTable *model.XRefTable, a types.Array, cs []string) error {

	csName, ok := a[0].(types.Name)
	if !ok {
		return errors.New("pdfcpu: validateColorSpaceArraySubset: corrupt Colorspace array")
	}

	for _, v := range cs {
		if csName.Value() == v {
			return validateCSArray(xRefTable, a, v)
		}
	}

	return errors.Errorf("pdfcpu: validateColorSpaceArraySubset: invalid color space: %s\n", csName)
}

func validateColorSpaceArray(xRefTable *model.XRefTable, a types.Array, excludePatternCS bool) (err error) {

	// see 8.6 Color Spaces

	name, ok := a[0].(types.Name)
	if !ok {
		return errors.New("pdfcpu: validateColorSpaceArray: corrupt Colorspace array")
	}

	switch name {

	// CIE-based
	case model.CalGrayCS:
		err = validateCalGrayColorSpace(xRefTable, a, model.V11)

	case model.CalRGBCS:
		err = validateCalRGBColorSpace(xRefTable, a, model.V11)

	case model.LabCS:
		err = validateLabColorSpace(xRefTable, a, model.V11)

	case model.ICCBasedCS:
		err = validateICCBasedColorSpace(xRefTable, a, model.V13)

	// Special
	case model.IndexedCS:
		err = validateIndexedColorSpace(xRefTable, a, model.V11)

	case model.PatternCS:
		if excludePatternCS {
			return errors.New("pdfcpu: validateColorSpaceArray: Pattern color space not allowed")
		}
		err = validatePatternColorSpace(xRefTable, a, model.V12)

	case model.SeparationCS:
		err = validateSeparationColorSpace(xRefTable, a, model.V12)

	case model.DeviceNCS:
		err = validateDeviceNColorSpace(xRefTable, a, model.V13)

	// Relaxed validation:
	case model.DeviceRGBCS:

	default:
		err = errors.Errorf("pdfcpu: validateColorSpaceArray: undefined color space: %s\n", name)
	}

	return err
}

func validateColorSpace(xRefTable *model.XRefTable, o types.Object, excludePatternCS bool) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		validateSpecialColorSpaceName := func(s string) bool { return types.MemberOf(s, []string{"Pattern"}) }
		if ok := validateDeviceColorSpaceName(o.Value()) || validateSpecialColorSpaceName(o.Value()); !ok {
			err = errors.Errorf("validateColorSpace: invalid device color space name: %v\n", o)
		}

	case types.Array:
		err = validateColorSpaceArray(xRefTable, o, excludePatternCS)

	default:
		err = errors.New("pdfcpu: validateColorSpace: corrupt obj typ, must be Name or Array")

	}

	return err
}

func validateColorSpaceEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, excludePatternCS bool) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		if ok := validateDeviceColorSpaceName(o.Value()); !ok {
			err = errors.Errorf("pdfcpu: validateColorSpaceEntry: Name:%s\n", o.Value())
		}

	case types.Array:
		err = validateColorSpaceArray(xRefTable, o, excludePatternCS)

	default:
		err = errors.Errorf("pdfcpu: validateColorSpaceEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateColorSpaceResourceDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 8.6 Color Spaces

	// Version check
	err := xRefTable.ValidateVersion("ColorSpaceResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Iterate over colorspace resource dictionary
	for _, o := range d {

		// Process colorspace
		err = validateColorSpace(xRefTable, o, IncludePatternCS)
		if err != nil {
			return err
		}

	}

	return nil
}

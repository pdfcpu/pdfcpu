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

func validateDeviceColorSpaceName(s string) bool {
	return pdf.MemberOf(s, []string{pdf.DeviceGrayCS, pdf.DeviceRGBCS, pdf.DeviceCMYKCS})
}

func validateCalGrayColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Gamma", OPTIONAL, sinceVersion, nil)

	return err
}

func validateCalRGBColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Gamma", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Matrix", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 9 })

	return err
}

func validateLabColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Range", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 4 })

	return err
}

func validateICCBasedColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

	// see 8.6.5.5

	dictName := "ICCBasedColorSpace"

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V12
	}
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return errors.Errorf("validateICCBasedColorSpace: invalid array length %d (expected 2) \n.", len(a))
	}

	sd, err := validateStreamDict(xRefTable, a[1])
	if err != nil || sd == nil {
		return err
	}

	validate := func(i int) bool { return pdf.IntMemberOf(i, []int{1, 3, 4}) }
	N, err := validateIntegerEntry(xRefTable, sd.Dict, dictName, "N", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	err = validateColorSpaceEntry(xRefTable, sd.Dict, dictName, "Alternate", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Range", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 2*N.Value() })
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	return validateMetadata(xRefTable, sd.Dict, OPTIONAL, pdf.V14)
}

func validateIndexedColorSpaceLookuptable(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case pdf.StringLiteral, pdf.HexLiteral:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", pdf.V12)

	case pdf.StreamDict:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", sinceVersion)

	default:
		err = errors.Errorf("validateIndexedColorSpaceLookuptable: invalid type\n")

	}

	return err
}

func validateIndexedColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

func validatePatternColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

func validateSeparationColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

func validateDeviceNColorSpaceColorantsDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	for _, obj := range d {

		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		if a != nil {
			err = validateSeparationColorSpace(xRefTable, a, pdf.V12)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validateDeviceNColorSpaceProcessDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "DeviceNCSProcessDict"

	err := validateColorSpaceEntry(xRefTable, d, dictName, "ColorSpace", REQUIRED, true)
	if err != nil {
		return err
	}

	_, err = validateNameArrayEntry(xRefTable, d, dictName, "Components", REQUIRED, pdf.V10, nil)

	return err
}

func validateDeviceNColorSpaceSoliditiesDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	for _, obj := range d {
		_, err := validateFloat(xRefTable, obj, func(f float64) bool { return f >= 0.0 && f <= 1.0 })
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceDotGainDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	for _, obj := range d {
		err := validateFunction(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDeviceNColorSpaceMixingHintsDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "deviceNCSMixingHintsDict"

	d1, err := validateDictEntry(xRefTable, d, dictName, "Solidities", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateDeviceNColorSpaceSoliditiesDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	_, err = validateNameArrayEntry(xRefTable, d, dictName, "PrintingOrder", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	d1, err = validateDictEntry(xRefTable, d, dictName, "DotGain", OPTIONAL, pdf.V11, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateDeviceNColorSpaceDotGainDict(xRefTable, d1)
	}

	return err
}

func validateDeviceNColorSpaceAttributesDict(xRefTable *pdf.XRefTable, o pdf.Object) error {

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictName := "deviceNCSAttributesDict"

	sinceVersion := pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool { return s == "DeviceN" || s == "NChannel" })
	if err != nil {
		return err
	}

	d1, err := validateDictEntry(xRefTable, d, dictName, "Colorants", OPTIONAL, pdf.V11, nil)
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

	d1, err = validateDictEntry(xRefTable, d, dictName, "MixingHints", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateDeviceNColorSpaceMixingHintsDict(xRefTable, d1)
	}

	return err
}

func validateDeviceNColorSpace(xRefTable *pdf.XRefTable, a pdf.Array, sinceVersion pdf.Version) error {

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

func validateCSArray(xRefTable *pdf.XRefTable, a pdf.Array, csName string) error {

	// see 8.6 Color Spaces

	switch csName {

	// CIE-based
	case pdf.CalGrayCS:
		return validateCalGrayColorSpace(xRefTable, a, pdf.V11)

	case pdf.CalRGBCS:
		return validateCalRGBColorSpace(xRefTable, a, pdf.V11)

	case pdf.LabCS:
		return validateLabColorSpace(xRefTable, a, pdf.V11)

	case pdf.ICCBasedCS:
		return validateICCBasedColorSpace(xRefTable, a, pdf.V13)

	// Special
	case pdf.IndexedCS:
		return validateIndexedColorSpace(xRefTable, a, pdf.V11)

	case pdf.PatternCS:
		return validatePatternColorSpace(xRefTable, a, pdf.V12)

	case pdf.SeparationCS:
		return validateSeparationColorSpace(xRefTable, a, pdf.V12)

	case pdf.DeviceNCS:
		return validateDeviceNColorSpace(xRefTable, a, pdf.V13)

	default:
		return errors.Errorf("validateColorSpaceArray: undefined color space: %s\n", csName)
	}

}

func validateColorSpaceArraySubset(xRefTable *pdf.XRefTable, a pdf.Array, cs []string) error {

	csName, ok := a[0].(pdf.Name)
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

func validateColorSpaceArray(xRefTable *pdf.XRefTable, a pdf.Array, excludePatternCS bool) (err error) {

	// see 8.6 Color Spaces

	name, ok := a[0].(pdf.Name)
	if !ok {
		return errors.New("pdfcpu: validateColorSpaceArray: corrupt Colorspace array")
	}

	switch name {

	// CIE-based
	case pdf.CalGrayCS:
		err = validateCalGrayColorSpace(xRefTable, a, pdf.V11)

	case pdf.CalRGBCS:
		err = validateCalRGBColorSpace(xRefTable, a, pdf.V11)

	case pdf.LabCS:
		err = validateLabColorSpace(xRefTable, a, pdf.V11)

	case pdf.ICCBasedCS:
		err = validateICCBasedColorSpace(xRefTable, a, pdf.V13)

	// Special
	case pdf.IndexedCS:
		err = validateIndexedColorSpace(xRefTable, a, pdf.V11)

	case pdf.PatternCS:
		if excludePatternCS {
			return errors.New("pdfcpu: validateColorSpaceArray: Pattern color space not allowed")
		}
		err = validatePatternColorSpace(xRefTable, a, pdf.V12)

	case pdf.SeparationCS:
		err = validateSeparationColorSpace(xRefTable, a, pdf.V12)

	case pdf.DeviceNCS:
		err = validateDeviceNColorSpace(xRefTable, a, pdf.V13)

	default:
		err = errors.Errorf("pdfcpu: validateColorSpaceArray: undefined color space: %s\n", name)
	}

	return err
}

func validateColorSpace(xRefTable *pdf.XRefTable, o pdf.Object, excludePatternCS bool) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		validateSpecialColorSpaceName := func(s string) bool { return pdf.MemberOf(s, []string{"Pattern"}) }
		if ok := validateDeviceColorSpaceName(o.Value()) || validateSpecialColorSpaceName(o.Value()); !ok {
			err = errors.Errorf("validateColorSpace: invalid device color space name: %v\n", o)
		}

	case pdf.Array:
		err = validateColorSpaceArray(xRefTable, o, excludePatternCS)

	default:
		err = errors.New("pdfcpu: validateColorSpace: corrupt obj typ, must be Name or Array")

	}

	return err
}

func validateColorSpaceEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, excludePatternCS bool) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, pdf.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		if ok := validateDeviceColorSpaceName(o.Value()); !ok {
			err = errors.Errorf("pdfcpu: validateColorSpaceEntry: Name:%s\n", o.Value())
		}

	case pdf.Array:
		err = validateColorSpaceArray(xRefTable, o, excludePatternCS)

	default:
		err = errors.Errorf("pdfcpu: validateColorSpaceEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return err
}

func validateColorSpaceResourceDict(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

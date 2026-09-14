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

var errMissingColorSpaceObject = errors.New("color space: missing object")

type colorSpaceRole int

const (
	colorSpaceAny colorSpaceRole = iota
	colorSpaceNoPattern
	colorSpaceIndexedBase
	colorSpaceDeviceOrCIE
	colorSpaceSeparation
)

func (r colorSpaceRole) allows(name types.Name) bool {
	switch r {
	case colorSpaceAny:
		return true
	case colorSpaceNoPattern:
		return name != model.PatternCS
	case colorSpaceIndexedBase:
		return name != model.PatternCS && name != model.IndexedCS
	case colorSpaceDeviceOrCIE:
		return types.MemberOf(name.Value(), []string{
			model.DeviceGrayCS, model.DeviceRGBCS, model.DeviceCMYKCS,
			model.CalGrayCS, model.CalRGBCS, model.LabCS, model.ICCBasedCS,
		})
	case colorSpaceSeparation:
		return name == model.SeparationCS
	}
	return false
}

func (r colorSpaceRole) description() string {
	switch r {
	case colorSpaceNoPattern:
		return "non-Pattern color space"
	case colorSpaceIndexedBase:
		return "Indexed base color space"
	case colorSpaceDeviceOrCIE:
		return "device or CIE-based color space"
	case colorSpaceSeparation:
		return "Separation color space"
	}
	return "color space"
}

type colorSpaceTraversal struct {
	c         context.Context
	xRefTable *model.XRefTable
	ancestors map[int]bool
}

func newColorSpaceTraversal(c context.Context, xRefTable *model.XRefTable) *colorSpaceTraversal {
	return &colorSpaceTraversal{c: c, xRefTable: xRefTable, ancestors: map[int]bool{}}
}

func (t *colorSpaceTraversal) enter(o types.Object) (int, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0, nil
	}
	objNr := ir.ObjectNumber.Value()
	if objNr <= 0 {
		return 0, nil
	}
	if t.ancestors[objNr] {
		return 0, fmt.Errorf("obj#%d: %w", objNr, model.ErrColorSpaceCycle)
	}
	t.ancestors[objNr] = true
	return objNr, nil
}

func (t *colorSpaceTraversal) leave(objNr int) {
	if objNr != 0 {
		delete(t.ancestors, objNr)
	}
}

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
		return fmt.Errorf("CalGray color space: invalid array length %d, expected 2", len(a))
	}

	d, err := xRefTable.DereferenceDict(a[1])
	if err != nil {
		return fmt.Errorf("CalGray color space parameters: dereference dict: %w", err)
	}
	if d == nil {
		return errors.New("CalGray color space parameters: missing dict")
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Gamma", OPTIONAL, sinceVersion, nil)

	return err
}

func validateCalRGBColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {
	dictName := "calRGBCSDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return fmt.Errorf("CalRGB color space: invalid array length %d, expected 2", len(a))
	}

	d, err := xRefTable.DereferenceDict(a[1])
	if err != nil {
		return fmt.Errorf("CalRGB color space parameters: dereference dict: %w", err)
	}
	if d == nil {
		return errors.New("CalRGB color space parameters: missing dict")
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Gamma", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 9 })

	return err
}

func validateLabColorSpace(xRefTable *model.XRefTable, a types.Array, sinceVersion model.Version) error {
	dictName := "labCSDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return fmt.Errorf("Lab color space: invalid array length %d, expected 2", len(a))
	}

	d, err := xRefTable.DereferenceDict(a[1])
	if err != nil {
		return fmt.Errorf("Lab color space parameters: dereference dict: %w", err)
	}
	if d == nil {
		return errors.New("Lab color space parameters: missing dict")
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "WhitePoint", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Range", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 4 })

	return err
}

func (t *colorSpaceTraversal) validateAlternateColorSpaceEntryForICC(d types.Dict, ownerObjNr int, dictName string, entryName string, required bool, depth int) error {
	xRefTable := t.xRefTable
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	rawEntry := d[entryName]
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		if ok := validateAllColorSpaceNamesExceptPattern(o.Value()); !ok {
			err = fmt.Errorf("%s.%s: invalid alternate color space name %q", dictName, entryName, o.Value())
		}

	case types.Array:
		if err = t.validateColorSpaceDepth(rawEntry, objNr, colorSpaceNoPattern, depth+1); err != nil {
			err = fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		err = fmt.Errorf("%s.%s: expected name or color space array, got %T", dictName, entryName, o)

	}

	return model.WithValidationErrorObject(err, objNr)
}

func (t *colorSpaceTraversal) validateICCBasedColorSpace(a types.Array, sinceVersion model.Version, depth int) (err error) {
	// see 8.6.5.5

	xRefTable := t.xRefTable
	dictName := "ICCBasedColorSpace"

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	err = xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 2 {
		return fmt.Errorf("ICCBased color space: invalid array length %d, expected 2", len(a))
	}

	ir, ok := a[1].(types.IndirectRef)
	if !ok {
		return fmt.Errorf("ICCBased color space profile: expected indirect reference, got %T", a[1])
	}
	profileObjNr := ir.ObjectNumber.Value()
	defer func() {
		err = model.WithValidationErrorObject(err, profileObjNr)
	}()

	valid, err := xRefTable.IsValid(ir)
	if err != nil {
		return fmt.Errorf("ICCBased color space profile: check valid: %w", err)
	}
	if valid {
		return nil
	}

	sd, err := validateStreamDictForObject(xRefTable, a[1], profileObjNr)
	if err != nil {
		return fmt.Errorf("ICCBased color space profile: %w", err)
	}
	if sd == nil {
		return errors.New("ICCBased color space profile: missing stream dict")
	}
	if err := xRefTable.SetValid(ir); err != nil {
		return fmt.Errorf("ICCBased color space profile: mark valid: %w", err)
	}

	validate := func(i int) bool { return types.IntMemberOf(i, []int{1, 3, 4}) }
	N, err := validateIntegerEntry(xRefTable, sd.Dict, profileObjNr, dictName, "N", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	err = t.validateAlternateColorSpaceEntryForICC(sd.Dict, profileObjNr, dictName, "Alternate", OPTIONAL, depth)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, profileObjNr, dictName, "Range", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 2*N.Value() })
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	err = validateMetadata(xRefTable, sd.Dict, OPTIONAL, model.V14)
	return model.WithValidationErrorObject(err, validationEntryObjectNumber(profileObjNr, sd.Dict, "Metadata"))
}

func validateIndexedColorSpaceLookuptable(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, sinceVersion model.Version) (err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("Indexed color space lookup table: dereference: %w", err)
	}
	if o == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return nil
		}
		return errors.New("Indexed color space lookup table: missing object")
	}

	switch o.(type) {

	case types.StringLiteral, types.HexLiteral:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", model.V12)

	case types.StreamDict:
		err = xRefTable.ValidateVersion("IndexedColorSpaceLookuptable", sinceVersion)

	default:
		err = fmt.Errorf("Indexed color space lookup table: expected string, hex literal or stream dict, got %T", o)

	}

	return err
}

func (t *colorSpaceTraversal) validateIndexedColorSpace(a types.Array, ownerObjNr int, sinceVersion model.Version, depth int) error {
	// see 8.6.6.3

	xRefTable := t.xRefTable
	err := xRefTable.ValidateVersion("IndexedColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 4 {
		return fmt.Errorf("Indexed color space: invalid array length %d, expected 4", len(a))
	}

	// arr[1] base: base colorspace
	err = t.validateColorSpaceDepth(a[1], ownerObjNr, colorSpaceIndexedBase, depth+1)
	if err != nil {
		return fmt.Errorf("Indexed color space base: %w", err)
	}

	// arr[2] hival: 0 <= int <= 255
	hivalObjNr := validationObjectNumber(ownerObjNr, a[2])
	_, err = validateIntegerForObject(xRefTable, a[2], ownerObjNr, func(i int) bool { return i >= 0 && i <= 255 })
	if err != nil {
		err = fmt.Errorf("Indexed color space hival: %w", err)
		return model.WithValidationErrorObject(err, hivalObjNr)
	}

	// arr[3] lookup: stream since V1.2 or byte string
	if err := validateIndexedColorSpaceLookuptable(xRefTable, a[3], ownerObjNr, sinceVersion); err != nil {
		return fmt.Errorf("Indexed color space lookup: %w", err)
	}
	return nil
}

func (t *colorSpaceTraversal) validatePatternColorSpace(a types.Array, ownerObjNr int, sinceVersion model.Version, depth int) error {
	xRefTable := t.xRefTable
	err := xRefTable.ValidateVersion("PatternColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) < 1 || len(a) > 2 {
		return fmt.Errorf("Pattern color space: invalid array length %d, expected 1 or 2", len(a))
	}

	// 8.7.3.3: arr[1]: name of underlying color space, any cs except PatternCS
	if len(a) == 2 {
		err := t.validateColorSpaceDepth(a[1], ownerObjNr, colorSpaceNoPattern, depth+1)
		if err != nil {
			return fmt.Errorf("Pattern color space underlying color space: %w", err)
		}
	}

	return nil
}

func (t *colorSpaceTraversal) validateSeparationColorSpace(a types.Array, ownerObjNr int, sinceVersion model.Version, depth int) error {
	// see 8.6.6.4

	xRefTable := t.xRefTable
	err := xRefTable.ValidateVersion("SeparationColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) != 4 {
		return fmt.Errorf("Separation color space: invalid array length %d, expected 4", len(a))
	}

	// arr[1]: colorant name, arbitrary
	colorantObjNr := validationObjectNumber(ownerObjNr, a[1])
	_, err = validateNameForObject(xRefTable, a[1], ownerObjNr, nil)
	if err != nil {
		err = fmt.Errorf("Separation color space colorant name: %w", err)
		return model.WithValidationErrorObject(err, colorantObjNr)
	}

	// arr[2]: alternate space
	err = t.validateColorSpaceDepth(a[2], ownerObjNr, colorSpaceDeviceOrCIE, depth+1)
	if err != nil {
		return fmt.Errorf("Separation color space alternate color space: %w", err)
	}

	// arr[3]: tintTransform, function
	if err := validateFunction(t.c, xRefTable, a[3], ownerObjNr); err != nil {
		return fmt.Errorf("Separation color space tint transform: %w", err)
	}
	return nil
}

func (t *colorSpaceTraversal) validateDeviceNColorSpaceColorantsDict(d types.Dict, ownerObjNr, depth int) error {
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(t.c); err != nil {
			return err
		}
		obj := d[name]
		objNr := validationObjectNumber(ownerObjNr, obj)

		if err := t.validateColorSpaceDepth(obj, ownerObjNr, colorSpaceSeparation, depth+1); err != nil {
			return model.WithValidationErrorObject(fmt.Errorf("DeviceN colorants %s: %w", name, err), objNr)
		}
	}

	return nil
}

func (t *colorSpaceTraversal) validateDeviceNColorSpaceProcessDict(d types.Dict, ownerObjNr, depth int) error {
	dictName := "DeviceNCSProcessDict"

	err := t.validateColorSpaceEntryDepth(d, ownerObjNr, dictName, "ColorSpace", REQUIRED, colorSpaceDeviceOrCIE, depth+1)
	if err != nil {
		return err
	}

	_, err = validateNameArrayEntry(t.xRefTable, d, ownerObjNr, dictName, "Components", REQUIRED, model.V10, nil)

	return err
}

func (t *colorSpaceTraversal) validateDeviceNColorSpaceSoliditiesDict(d types.Dict, ownerObjNr int) error {
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(t.c); err != nil {
			return err
		}
		obj := d[name]
		objNr := validationObjectNumber(ownerObjNr, obj)
		_, err := validateFloatForObject(
			t.xRefTable, obj, ownerObjNr, func(f float64) bool { return f >= 0.0 && f <= 1.0 },
		)
		if err != nil {
			err = fmt.Errorf("DeviceN solidities %s: %w", name, err)
			return model.WithValidationErrorObject(err, objNr)
		}
	}

	return nil
}

func (t *colorSpaceTraversal) validateDeviceNColorSpaceDotGainDict(d types.Dict, ownerObjNr int) error {
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(t.c); err != nil {
			return err
		}
		obj := d[name]
		err := validateFunction(t.c, t.xRefTable, obj, ownerObjNr)
		if err != nil {
			return fmt.Errorf("DeviceN dot gain %s: %w", name, err)
		}
	}

	return nil
}

func (t *colorSpaceTraversal) validateDeviceNColorSpaceMixingHintsDict(d types.Dict, ownerObjNr int) error {
	dictName := "deviceNCSMixingHintsDict"
	xRefTable := t.xRefTable

	rawSolidities := d["Solidities"]
	soliditiesObjNr := validationObjectNumber(ownerObjNr, rawSolidities)
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "Solidities", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = t.validateDeviceNColorSpaceSoliditiesDict(d1, soliditiesObjNr)
		if err != nil {
			return err
		}
	}

	_, err = validateNameArrayEntry(xRefTable, d, ownerObjNr, dictName, "PrintingOrder", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	rawDotGain := d["DotGain"]
	dotGainObjNr := validationObjectNumber(ownerObjNr, rawDotGain)
	d1, err = validateDictEntry(xRefTable, d, ownerObjNr, dictName, "DotGain", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = t.validateDeviceNColorSpaceDotGainDict(d1, dotGainObjNr)
	}

	return err
}

func (t *colorSpaceTraversal) validateDeviceNColorSpaceAttributesDict(o types.Object, ownerObjNr, depth int) (err error) {
	xRefTable := t.xRefTable
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("DeviceN color space attributes: dereference dict: %w", err)
	}
	if d == nil {
		return errors.New("DeviceN color space attributes: missing dict")
	}

	dictName := "deviceNCSAttributesDict"

	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}

	_, err = validateNameEntry(xRefTable, d, objNr, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool { return s == "DeviceN" || s == "NChannel" })
	if err != nil {
		return err
	}

	rawColorants := d["Colorants"]
	colorantsObjNr := validationObjectNumber(objNr, rawColorants)
	d1, err := validateDictEntry(xRefTable, d, objNr, dictName, "Colorants", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = t.validateDeviceNColorSpaceColorantsDict(d1, colorantsObjNr, depth)
		if err != nil {
			return err
		}
	}

	rawProcess := d["Process"]
	processObjNr := validationObjectNumber(objNr, rawProcess)
	d1, err = validateDictEntry(xRefTable, d, objNr, dictName, "Process", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = t.validateDeviceNColorSpaceProcessDict(d1, processObjNr, depth)
		if err != nil {
			return err
		}
	}

	rawMixingHints := d["MixingHints"]
	mixingHintsObjNr := validationObjectNumber(objNr, rawMixingHints)
	d1, err = validateDictEntry(xRefTable, d, objNr, dictName, "MixingHints", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = t.validateDeviceNColorSpaceMixingHintsDict(d1, mixingHintsObjNr)
	}

	return err
}

func (t *colorSpaceTraversal) validateDeviceNColorSpace(a types.Array, ownerObjNr int, sinceVersion model.Version, depth int) error {
	// see 8.6.6.5

	xRefTable := t.xRefTable
	err := xRefTable.ValidateVersion("DeviceNColorSpace", sinceVersion)
	if err != nil {
		return err
	}

	if len(a) < 4 || len(a) > 5 {
		return fmt.Errorf("DeviceN color space: invalid array length %d, expected 4 or 5", len(a))
	}

	// arr[1]: array of names specifying the individual color components
	// length subject to implementation limit.
	_, err = validateNameArray(xRefTable, a[1], ownerObjNr)
	if err != nil {
		return fmt.Errorf("DeviceN color space component names: %w", err)
	}

	// arr[2]: alternate space
	err = t.validateColorSpaceDepth(a[2], ownerObjNr, colorSpaceDeviceOrCIE, depth+1)
	if err != nil {
		return fmt.Errorf("DeviceN color space alternate color space: %w", err)
	}

	// arr[3]: tintTransform, function
	err = validateFunction(t.c, xRefTable, a[3], ownerObjNr)
	if err != nil {
		return fmt.Errorf("DeviceN color space tint transform: %w", err)
	}

	// arr[4]: color space attributes dict, optional
	if len(a) == 5 {
		if err = t.validateDeviceNColorSpaceAttributesDict(a[4], ownerObjNr, depth); err != nil {
			return fmt.Errorf("DeviceN color space attributes: %w", err)
		}
	}

	return nil
}

func (t *colorSpaceTraversal) validateCSArray(a types.Array, ownerObjNr int, csName string, depth int) error {
	xRefTable := t.xRefTable
	switch csName {

	// CIE-based
	case model.CalGrayCS:
		return validateCalGrayColorSpace(xRefTable, a, model.V11)

	case model.CalRGBCS:
		return validateCalRGBColorSpace(xRefTable, a, model.V11)

	case model.LabCS:
		return validateLabColorSpace(xRefTable, a, model.V11)

	case model.ICCBasedCS:
		return t.validateICCBasedColorSpace(a, model.V13, depth)

	// Special
	case model.IndexedCS:
		return t.validateIndexedColorSpace(a, ownerObjNr, model.V11, depth)

	case model.PatternCS:
		return t.validatePatternColorSpace(a, ownerObjNr, model.V12, depth)

	case model.SeparationCS:
		return t.validateSeparationColorSpace(a, ownerObjNr, model.V12, depth)

	case model.DeviceNCS:
		return t.validateDeviceNColorSpace(a, ownerObjNr, model.V13, depth)

	default:
		return fmt.Errorf("color space array: undefined color space %q", csName)
	}

}

func colorSpaceArrayName(xRefTable *model.XRefTable, a types.Array, ownerObjNr int) (types.Name, error) {
	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		return "", model.WithValidationErrorObject(
			fmt.Errorf("color space array[0]: %w", err), validationObjectNumber(ownerObjNr, a[0]),
		)
	}
	name, ok := o.(types.Name)
	if !ok {
		return "", model.WithValidationErrorObject(
			fmt.Errorf("color space array[0]: expected name, got %T", o), validationObjectNumber(ownerObjNr, a[0]),
		)
	}
	return name, nil
}

func (t *colorSpaceTraversal) validateColorSpaceArraySubset(a types.Array, ownerObjNr int, cs []string) error {
	if len(a) == 0 {
		return errors.New("color space array: empty")
	}
	csName, err := colorSpaceArrayName(t.xRefTable, a, ownerObjNr)
	if err != nil {
		return err
	}

	for _, v := range cs {
		if csName.Value() == v {
			if err := t.validateCSArray(a, ownerObjNr, v, 0); err != nil {
				return fmt.Errorf("color space %s: %w", csName.Value(), err)
			}
			return nil
		}
	}

	return fmt.Errorf("color space array: invalid color space %q", csName.Value())
}

func (t *colorSpaceTraversal) validateColorSpaceArray(a types.Array, ownerObjNr int, role colorSpaceRole, depth int) (err error) {
	if len(a) == 0 {
		return errors.New("color space array: empty")
	}
	name, err := colorSpaceArrayName(t.xRefTable, a, ownerObjNr)
	if err != nil {
		return err
	}
	if !role.allows(name) {
		return fmt.Errorf("color space %s: not allowed as %s", name.Value(), role.description())
	}

	xRefTable := t.xRefTable

	switch name {

	// CIE-based
	case model.CalGrayCS:
		err = validateCalGrayColorSpace(xRefTable, a, model.V11)

	case model.CalRGBCS:
		err = validateCalRGBColorSpace(xRefTable, a, model.V11)

	case model.LabCS:
		err = validateLabColorSpace(xRefTable, a, model.V11)

	case model.ICCBasedCS:
		err = t.validateICCBasedColorSpace(a, model.V13, depth)

	// Special
	case model.IndexedCS:
		err = t.validateIndexedColorSpace(a, ownerObjNr, model.V11, depth)

	case model.PatternCS:
		err = t.validatePatternColorSpace(a, ownerObjNr, model.V12, depth)

	case model.SeparationCS:
		err = t.validateSeparationColorSpace(a, ownerObjNr, model.V12, depth)

	case model.DeviceNCS:
		err = t.validateDeviceNColorSpace(a, ownerObjNr, model.V13, depth)

	case model.DeviceGrayCS, model.DeviceRGBCS, model.DeviceCMYKCS:
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			err = fmt.Errorf("color space array: device color space %q must be a name object", name.Value())
		}

	default:
		err = fmt.Errorf("color space array: undefined color space %q", name.Value())
	}

	if err != nil {
		return fmt.Errorf("color space %s: %w", name.Value(), err)
	}
	return nil
}

func (t *colorSpaceTraversal) validateColorSpaceDepth(o types.Object, ownerObjNr int, role colorSpaceRole, depth int) (err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if err := contextutil.Check(t.c); err != nil {
		return err
	}
	if err := t.xRefTable.CheckRecursionDepth("colour-space graph", depth); err != nil {
		return err
	}
	ancestor, err := t.enter(o)
	if err != nil {
		return err
	}
	defer t.leave(ancestor)

	o, err = t.xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("color space: dereference: %w", err)
	}
	if o == nil {
		return errMissingColorSpaceObject
	}

	switch o := o.(type) {

	case types.Name:
		if !role.allows(o) {
			return fmt.Errorf("color space %s: not allowed as %s", o.Value(), role.description())
		}
		validateSpecialColorSpaceName := func(s string) bool { return types.MemberOf(s, []string{"Pattern"}) }
		if ok := validateDeviceColorSpaceName(o.Value()) || validateSpecialColorSpaceName(o.Value()); !ok {
			err = fmt.Errorf("color space name: invalid device color space name %q", o.Value())
		}

	case types.Array:
		err = t.validateColorSpaceArray(o, objNr, role, depth)

	default:
		if t.xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("color space: expected name or array, got %T", o)
		}
		model.ShowSkipped(fmt.Sprintf("invalid color space type: %s", o))
	}

	return err
}

func validateColorSpaceArraySubset(c context.Context, xRefTable *model.XRefTable, a types.Array, ownerObjNr int, cs []string) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := xRefTable.CheckRecursionDepth("colour-space graph", 0); err != nil {
		return err
	}
	return newColorSpaceTraversal(c, xRefTable).validateColorSpaceArraySubset(a, ownerObjNr, cs)
}

func (t *colorSpaceTraversal) validateColorSpaceEntryDepth(d types.Dict, ownerObjNr int, dictName, entryName string, required bool, role colorSpaceRole, depth int) error {
	xRefTable := t.xRefTable
	if err := contextutil.Check(t.c); err != nil {
		return err
	}
	if err := xRefTable.CheckRecursionDepth("colour-space graph", depth); err != nil {
		return err
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	rawEntry := d[entryName]
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, model.V10)
	if err != nil || o == nil {
		if err != nil {
			err = fmt.Errorf("%s.%s: %w", dictName, entryName, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		return nil
	}

	switch o := o.(type) {

	case types.Name:
		if !role.allows(o) {
			err = fmt.Errorf("%s.%s: color space %s: not allowed as %s", dictName, entryName, o.Value(), role.description())
			return model.WithValidationErrorObject(err, objNr)
		}
		if ok := validateDeviceColorSpaceName(o.Value()); !ok {
			if xRefTable.ValidationMode == model.ValidationStrict {
				err = fmt.Errorf("%s.%s: invalid device color space name %q", dictName, entryName, o.Value())
				return model.WithValidationErrorObject(err, objNr)
			}
			model.ShowSkipped(fmt.Sprintf("invalid colorSpaceEntry: %s", o.Value()))
		}

	case types.Array:
		if err = t.validateColorSpaceDepth(rawEntry, ownerObjNr, role, depth); err != nil {
			err = fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	default:
		err = fmt.Errorf("%s.%s: expected name or color space array, got %T", dictName, entryName, o)

	}

	return model.WithValidationErrorObject(err, objNr)
}

func validateColorSpaceEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, role colorSpaceRole) error {
	return newColorSpaceTraversal(c, xRefTable).validateColorSpaceEntryDepth(d, ownerObjNr, dictName, entryName, required, role, 0)
}

func validateColorSpaceResourceDict(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	resourceObjNr := validationObjectNumber(0, o)
	// see 8.6 Color Spaces

	// Version check
	err := xRefTable.ValidateVersion("ColorSpaceResourceDict", sinceVersion)
	if err != nil {
		err = fmt.Errorf("ColorSpace resource dict: version: %w", err)
		return model.WithValidationErrorObject(err, resourceObjNr)
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		err = fmt.Errorf("ColorSpace resource dict: dereference dict: %w", err)
		return model.WithValidationErrorObject(err, resourceObjNr)
	}
	if d == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return nil
		}
		err = errors.New("ColorSpace resource dict: missing dict")
		return model.WithValidationErrorObject(err, resourceObjNr)
	}

	t := newColorSpaceTraversal(c, xRefTable)

	// Iterate over colorspace resource dictionary
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		o := d[name]
		colorSpaceObjNr := validationObjectNumber(resourceObjNr, o)
		// Process colorspace
		err = t.validateColorSpaceDepth(o, resourceObjNr, colorSpaceAny, 0)
		if err != nil {
			if xRefTable.ValidationMode == model.ValidationRelaxed && errors.Is(err, errMissingColorSpaceObject) {
				d.Delete(name)
				msg := fmt.Sprintf("removed missing ColorSpace resource %s", name)
				if ir, ok := o.(types.IndirectRef); ok {
					msg += fmt.Sprintf(" (object #%d)", ir.ObjectNumber)
				}
				model.ShowMsg(msg)
				continue
			}
			err = fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("ColorSpace resource %s", name), o), err)
			return model.WithValidationErrorObject(err, colorSpaceObjNr)
		}

	}

	return nil
}

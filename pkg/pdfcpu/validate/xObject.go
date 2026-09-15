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
	"strconv"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	isAlternateImageStreamDict   = true
	isNoAlternateImageStreamDict = false
)

func validateReferenceDictPageEntry(xRefTable *model.XRefTable, o types.Object) error {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("reference dict Page: dereference: %w", err)
	}
	if o == nil {
		return nil
	}

	switch o.(type) {

	case types.Integer, types.StringLiteral, types.HexLiteral:
		// no further processing

	default:
		return fmt.Errorf("reference dict Page: expected integer or text string, got %T", o)

	}

	return nil
}

func validateReferenceDict(xRefTable *model.XRefTable, d types.Dict) error {
	// see 8.10.4 Reference XObjects

	dictName := "refDict"

	// F, file spec, required
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return fmt.Errorf("%s.F: %w", dictName, err)
	}

	// Page, integer or text string, required
	o, ok := d.Find("Page")
	if !ok {
		return errors.New("refDict.Page: missing required entry")
	}

	err = validateReferenceDictPageEntry(xRefTable, o)
	if err != nil {
		return fmt.Errorf("%s.Page: %w", dictName, err)
	}

	// ID, string array, optional
	_, err = validateStringArrayEntry(xRefTable, d, 0, dictName, "ID", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })

	if err != nil {
		return fmt.Errorf("%s.ID: %w", dictName, err)
	}
	return nil
}

func validateOPIDictV13Part1(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return err
	}

	// Version, required, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Version", REQUIRED, model.V10, func(f float64) bool { return f == 1.3 })
	if err != nil {
		return err
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// ID, optional, byte string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "ID", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Comments, optional, text string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "Comments", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Size, required, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, d, 0, dictName, "Size", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// CropRect, required, array of integers, len 4
	_, err = validateRectangleEntry(xRefTable, d, 0, dictName, "CropRect", REQUIRED, model.V10, nil)

	if err != nil {
		return err
	}

	// CropFixed, optional, array of numbers, len 4
	_, err = validateRectangleEntry(xRefTable, d, 0, dictName, "CropFixed", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Position, required, array of numbers, len 8
	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Position", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 8 })

	return err
}

func validateOPIIntegerArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string) (types.Array, []int, error) {
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return nil, nil, err
	}
	objNr := validationEntryObjectNumber(0, d, entryName)
	values := make([]int, len(a))
	for i, raw := range a {
		entryObjNr := validationObjectNumber(objNr, raw)
		o, err := xRefTable.Dereference(raw)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: dereference integer: %w", dictName, entryName, i, err)
			return nil, nil, model.WithValidationErrorObject(err, entryObjNr)
		}
		integer, ok := o.(types.Integer)
		if !ok {
			err = fmt.Errorf("%s.%s[%d]: expected integer", dictName, entryName, i)
			return nil, nil, model.WithValidationErrorObject(err, entryObjNr)
		}
		values[i] = integer.Value()
	}
	return a, values, nil
}

func validateOPIImageType(xRefTable *model.XRefTable, d types.Dict, dictName string) (*int, error) {
	a, values, err := validateOPIIntegerArrayEntry(xRefTable, d, dictName, "ImageType")
	if err != nil || a == nil {
		return nil, err
	}
	objNr := validationEntryObjectNumber(0, d, "ImageType")
	if err = validateArrayExactLength(a, objNr, dictName, "ImageType", 2); err != nil {
		return nil, err
	}
	for i, value := range values {
		if value <= 0 {
			err = fmt.Errorf("%s.ImageType[%d]: invalid value %d, expected a positive integer", dictName, i, value)
			return nil, model.WithValidationErrorObject(err, validationObjectNumber(objNr, a[i]))
		}
	}
	return &values[1], nil
}

func validateOPIGrayMapLength(a types.Array, objNr, bitsPerSample int, dictName string) error {
	expected := fmt.Sprintf("2^%d values, matching ImageType bits per sample", bitsPerSample)
	if bitsPerSample < strconv.IntSize {
		length := 1 << uint(bitsPerSample)
		if len(a) == length {
			return nil
		}
		expected = fmt.Sprintf("%d (2^%d), matching ImageType bits per sample", length, bitsPerSample)
	}
	return arrayCardinalityError(dictName, "GrayMap", objNr, len(a), expected)
}

func validateOPIGrayMap(xRefTable *model.XRefTable, d types.Dict, dictName string, bitsPerSample *int) error {
	a, values, err := validateOPIIntegerArrayEntry(xRefTable, d, dictName, "GrayMap")
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(0, d, "GrayMap")
	if bitsPerSample == nil {
		err = fmt.Errorf("%s.GrayMap: ImageType is required to determine the expected length", dictName)
		return model.WithValidationErrorObject(err, objNr)
	}
	if err = validateOPIGrayMapLength(a, objNr, *bitsPerSample, dictName); err != nil {
		return err
	}
	for i, value := range values {
		if value < 0 || value > 65535 {
			err = fmt.Errorf("%s.GrayMap[%d]: invalid value %d, expected 0 through 65535", dictName, i, value)
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, a[i]))
		}
	}
	return nil
}

func validateOPITags(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	const entryName = "Tags"
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(0, d, entryName)
	if err = validateArrayPairs(a, objNr, dictName, entryName, 0); err != nil {
		return err
	}
	for i := 0; i < len(a); i += 2 {
		tagObjNr := validationObjectNumber(objNr, a[i])
		o, err := xRefTable.Dereference(a[i])
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: dereference tag number: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, tagObjNr)
		}
		if _, ok := o.(types.Integer); !ok {
			err = fmt.Errorf("%s.%s[%d]: expected TIFF tag number integer", dictName, entryName, i)
			return model.WithValidationErrorObject(err, tagObjNr)
		}
		textObjNr := validationObjectNumber(objNr, a[i+1])
		o, err = xRefTable.Dereference(a[i+1])
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: dereference tag text: %w", dictName, entryName, i+1, err)
			return model.WithValidationErrorObject(err, textObjNr)
		}
		if _, err = types.StringOrHexLiteral(o); err != nil {
			err = fmt.Errorf("%s.%s[%d]: expected TIFF tag text string", dictName, entryName, i+1)
			return model.WithValidationErrorObject(err, textObjNr)
		}
	}
	return nil
}

func validateOPIDictV13Part2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// Resolution, optional, array of numbers, len 2
	_, err := validateNumberArrayEntry(xRefTable, d, 0, dictName, "Resolution", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// ColorType, optional, name
	validateColorType := func(s string) bool {
		return types.MemberOf(s, []string{"Process", "Spot", "Separation"}) ||
			(s == "Intrinsic" && xRefTable.ValidationMode == model.ValidationRelaxed)
	}
	colorType, err := validateNameEntry(xRefTable, d, 0, dictName, "ColorType", OPTIONAL, model.V10, validateColorType)
	if err != nil {
		return err
	}

	// Color, optional, array, len 5
	_, err = validateArrayEntry(xRefTable, d, 0, dictName, "Color", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 5 })
	if err != nil {
		return err
	}

	// Tint, optional, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Tint", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Overprint, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Overprint", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// ImageType, optional, array of integers, len 2
	bitsPerSample, err := validateOPIImageType(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	// GrayMap, optional, array of integers
	err = validateOPIGrayMap(xRefTable, d, dictName, bitsPerSample)
	if err != nil {
		return err
	}

	// Transparency, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Transparency", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Tags, optional, array
	if err = validateOPITags(xRefTable, d, dictName); err != nil {
		return err
	}

	if colorType != nil && colorType.Value() == "Intrinsic" && xRefTable.ValidationMode == model.ValidationRelaxed {
		model.ShowDigestedSpecViolation("dict=" + dictName + " entry=ColorType invalid dict entry: Intrinsic")
	}

	return nil
}

func validateOPIDictV13(xRefTable *model.XRefTable, d types.Dict) error {
	// 14.11.7 Open Prepresse interface (OPI)

	dictName := "opiDictV13"

	err := validateOPIDictV13Part1(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	return validateOPIDictV13Part2(xRefTable, d, dictName)
}

func validateOPIInksArray(xRefTable *model.XRefTable, a types.Array, objNr int, dictName string) error {
	if len(a) < 3 || len(a)%2 == 0 {
		return arrayCardinalityError(dictName, "Inks", objNr, len(a),
			"/monochrome followed by one or more colourant string and tint pairs")
	}
	markerObjNr := validationObjectNumber(objNr, a[0])
	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		err = fmt.Errorf("%s.Inks[0]: dereference marker: %w", dictName, err)
		return model.WithValidationErrorObject(err, markerObjNr)
	}
	marker, ok := o.(types.Name)
	if !ok || marker.Value() != "monochrome" {
		err = fmt.Errorf("%s.Inks[0]: expected /monochrome marker", dictName)
		return model.WithValidationErrorObject(err, markerObjNr)
	}
	for i := 1; i < len(a); i += 2 {
		nameObjNr := validationObjectNumber(objNr, a[i])
		o, err = xRefTable.Dereference(a[i])
		if err != nil {
			err = fmt.Errorf("%s.Inks[%d]: dereference colourant name: %w", dictName, i, err)
			return model.WithValidationErrorObject(err, nameObjNr)
		}
		if _, err = types.StringOrHexLiteral(o); err != nil {
			err = fmt.Errorf("%s.Inks[%d]: expected colourant name string", dictName, i)
			return model.WithValidationErrorObject(err, nameObjNr)
		}
		tintObjNr := validationObjectNumber(objNr, a[i+1])
		tint, err := xRefTable.DereferenceNumber(a[i+1])
		if err != nil {
			err = fmt.Errorf("%s.Inks[%d]: expected tint number: %w", dictName, i+1, err)
			return model.WithValidationErrorObject(err, tintObjNr)
		}
		if tint < 0 || tint > 1 {
			err = fmt.Errorf("%s.Inks[%d]: invalid tint %g, expected 0 through 1", dictName, i+1, tint)
			return model.WithValidationErrorObject(err, tintObjNr)
		}
	}
	return nil
}

func validateOPIDictInks(xRefTable *model.XRefTable, raw types.Object, dictName string) error {
	objNr := validationObjectNumber(0, raw)
	o, err := xRefTable.Dereference(raw)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	switch o := o.(type) {
	case types.Name:
		if colorant := o.Value(); colorant != "full_color" && colorant != "registration" {
			err = fmt.Errorf("%s.Inks: invalid colourant name %s", dictName, colorant)
			return model.WithValidationErrorObject(err, objNr)
		}
	case types.Array:
		return validateOPIInksArray(xRefTable, o, objNr, dictName)
	default:
		err = fmt.Errorf("%s.Inks: expected name or array, got %T", dictName, o)
		return model.WithValidationErrorObject(err, objNr)
	}
	return nil
}

func validateOPIDictV20(xRefTable *model.XRefTable, d types.Dict) error {
	// 14.11.7 Open Prepresse interface (OPI)

	dictName := "opiDictV20"

	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "Version", REQUIRED, model.V10, func(f float64) bool { return f == 2.0 })
	if err != nil {
		return err
	}

	_, err = validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, 0, dictName, "MainImage", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	err = validateOPITags(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "Size", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, d, 0, dictName, "CropRect", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "Overprint", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	if o, found := d.Find("Inks"); found {
		err = validateOPIDictInks(xRefTable, o, dictName)
		if err != nil {
			return err
		}
	}

	_, err = validateIntegerArrayEntry(xRefTable, d, 0, dictName, "IncludedImageDimensions", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "IncludedImageQuality", OPTIONAL, model.V10, func(i int) bool { return i >= 1 && i <= 3 })

	return err
}

func validateOPIVersionDict(xRefTable *model.XRefTable, d types.Dict) error {
	// 14.11.7 Open Prepresse interface (OPI)

	if d.Len() != 1 {
		return errors.New("must have exactly one entry keyed 1.3 or 2.0")
	}

	validateOPIVersion := func(s string) bool { return types.MemberOf(s, []string{"1.3", "2.0"}) }

	for opiVersion, obj := range d {

		if !validateOPIVersion(opiVersion) {
			return errors.New("invalid OPI version")
		}

		d, err := xRefTable.DereferenceDict(obj)
		if err != nil || d == nil {
			return err
		}

		if opiVersion == "1.3" {
			err = validateOPIDictV13(xRefTable, d)
		} else {
			err = validateOPIDictV20(xRefTable, d)
		}

		if err != nil {
			return err
		}

	}

	return nil
}

func validateMaskStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict) error {
	t, _, err := xRefTable.DereferenceNameEntry(sd.Dict, "Type")
	if err != nil {
		return fmt.Errorf("mask stream dict Type: %w", err)
	}
	if t != nil && t.Value() != "XObject" {
		return fmt.Errorf("mask stream dict Type: expected XObject, got %q", t.Value())
	}

	subtype, _, err := xRefTable.DereferenceNameEntry(sd.Dict, "Subtype")
	if err != nil {
		return fmt.Errorf("mask stream dict Subtype: %w", err)
	}
	if subtype == nil || subtype.Value() != "Image" {
		return errors.New("mask stream dict Subtype: expected Image")
	}

	if err := validateImageStreamDict(c, xRefTable, sd, isNoAlternateImageStreamDict); err != nil {
		return fmt.Errorf("mask image stream dict: %w", err)
	}
	return nil
}

func validateMaskEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, components int, bits *types.Integer) (err error) {
	// stream ("explicit masking", another Image XObject) or array of colors ("color key masking")
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	switch o := o.(type) {

	case types.StreamDict:
		err = validateMaskStreamDict(c, xRefTable, &o)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}

	case types.Array:
		components = relaxedIndexedMaskComponents(xRefTable, d, o, components)
		return validateColorKeyMask(c, xRefTable, o, objNr, dictName, components, bits)

	default:
		return fmt.Errorf("%s.%s: expected image stream dict or color key array, got %T", dictName, entryName, o)

	}

	return nil
}

func validateAlternateImageStreamDicts(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (err error) {
	arrayObjNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, arrayObjNr)
	}()

	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
	}
	if a == nil {
		if required {
			return fmt.Errorf("%s.%s: missing required entry", dictName, entryName)
		}
		return nil
	}

	for i, o := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		objNr := validationObjectNumber(arrayObjNr, o)

		sd, err := validateStreamDictForObject(xRefTable, o, arrayObjNr)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}

		if sd == nil {
			continue
		}

		err = validateImageStreamDict(c, xRefTable, sd, isAlternateImageStreamDict)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
	}

	return nil
}

func validateImageStreamDictPart1(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, dictName string) (isImageMask bool, err error) {
	// Width, integer, required
	required := true
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Width", required, model.V10, nil)
	if err != nil {
		return false, err
	}

	// Height, integer, required
	required = true
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "Height", required, model.V10, nil)
	if err != nil {
		return false, err
	}

	// ImageMask, boolean, optional
	imageMask, err := validateBooleanEntry(xRefTable, sd.Dict, 0, dictName, "ImageMask", OPTIONAL, model.V10, nil)
	if err != nil {
		return false, err
	}

	isImageMask = (imageMask != nil) && *imageMask

	// ColorSpace, name or array, required unless used filter is JPXDecode; not allowed for imagemasks.
	if !isImageMask {

		required = REQUIRED
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			required = OPTIONAL
		}

		if sd.HasSoleFilterNamed(filter.JPX) {
			required = OPTIONAL
		}

		if sd.HasSoleFilterNamed(filter.CCITTFax) && xRefTable.ValidationMode == model.ValidationRelaxed {
			required = OPTIONAL
		}

		err = validateColorSpaceEntry(c, xRefTable, sd.Dict, 0, dictName, "ColorSpace", required, colorSpaceNoPattern)
		if err != nil {
			return false, err
		}

	}

	return isImageMask, nil
}

func validateImageStreamDictPart2(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, dictName string, isImageMask, isAlternate bool) error {
	// BitsPerComponent, integer
	required := REQUIRED
	if sd.HasSoleFilterNamed(filter.JPX) || isImageMask || xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}

	// For imageMasks BitsPerComponent must be 1.
	var validateBPC func(i int) bool
	if isImageMask {
		validateBPC = func(i int) bool {
			return i == 1
		}
	}
	bpc, err := validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "BitsPerComponent", required, model.V10, validateBPC)
	if err != nil {
		return err
	}

	// Note 8.6.5.8: If a PDF processor does not recognise the specified name, it shall use the RelativeColorimetric
	// intent by default.
	_, err = validateNameEntry(xRefTable, sd.Dict, 0, dictName, "Intent", OPTIONAL, model.V11, nil)
	if err != nil {
		return err
	}

	if err = validateImageArrays(c, xRefTable, sd, dictName, isImageMask, bpc); err != nil {
		return err
	}

	// Interpolate, boolean, optional
	_, err = validateBooleanEntry(xRefTable, sd.Dict, 0, dictName, "Interpolate", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Alternates, array, optional, since V1.3
	if !isAlternate {
		err = validateAlternateImageStreamDicts(c, xRefTable, sd.Dict, dictName, "Alternates", OPTIONAL, model.V13)
	}

	return err
}

type imageTraversalContextKey struct{}

type imageTraversal struct {
	xRefTable *model.XRefTable
	depth     int
}

func imageTraversalFromContext(c context.Context, xRefTable *model.XRefTable) (context.Context, *imageTraversal) {
	if c != nil {
		if t, ok := c.Value(imageTraversalContextKey{}).(*imageTraversal); ok {
			return c, t
		}
	}
	t := &imageTraversal{xRefTable: xRefTable}
	if c == nil {
		return nil, t
	}
	return context.WithValue(c, imageTraversalContextKey{}, t), t
}

func validateImageStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, isAlternate bool) error {
	c, traversal := imageTraversalFromContext(c, xRefTable)
	return traversal.validate(c, sd, isAlternate)
}

func (t *imageTraversal) validate(c context.Context, sd *types.StreamDict, isAlternate bool) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	depth := t.depth + 1
	if err := t.xRefTable.CheckRecursionDepth("image graph", depth); err != nil {
		return err
	}
	t.depth = depth
	defer func() {
		t.depth--
	}()
	return validateImageStreamDictBody(c, t.xRefTable, sd, isAlternate)
}

func validateImageStreamDictBody(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, isAlternate bool) error {
	dictName := "imageStreamDict"
	var isImageMask bool

	isImageMask, err := validateImageStreamDictPart1(c, xRefTable, sd, dictName)
	if err != nil {
		return err
	}

	err = validateImageStreamDictPart2(c, xRefTable, sd, dictName, isImageMask, isAlternate)
	if err != nil {
		return err
	}

	// SMask, stream, optional, since V1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V12
	}
	sMaskObjNr := validationEntryObjectNumber(0, sd.Dict, "SMask")
	sd1, err := validateStreamDictEntry(xRefTable, sd.Dict, 0, dictName, "SMask", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if sd1 != nil {
		err = validateImageStreamDict(c, xRefTable, sd1, isNoAlternateImageStreamDict)
		if err != nil {
			return model.WithValidationErrorObject(err, sMaskObjNr)
		}
	}

	// SMaskInData, integer, optional
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "SMaskInData", OPTIONAL, model.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return err
	}

	// Name, name, required for V10
	// Shall no longer be used.
	// _, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Name", xRefTable.Version() == model.V10, model.V10, nil)
	// if err != nil {
	// 	return err
	// }

	// StructParent, integer, optional
	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "StructParent", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// ID, byte string, optional, since V1.3
	_, err = validateStringEntry(xRefTable, sd.Dict, 0, dictName, "ID", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// OPI, dict, optional since V1.2
	err = validateEntryOPI(xRefTable, sd.Dict, dictName, "OPI", OPTIONAL, model.V12)
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	err = validateMetadata(xRefTable, sd.Dict, OPTIONAL, model.V14)
	if err != nil {
		return err
	}

	// OC, dict, optional since V1.5
	return validateOptionalContent(xRefTable, sd.Dict, dictName, "OC", OPTIONAL, model.V15)
}

func validateFormStreamDictPart1(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, dictName string) error {
	var err error
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		_, err = validateNumberEntry(xRefTable, sd.Dict, 0, dictName, "FormType", OPTIONAL, model.V10, func(f float64) bool { return f == 1. })
	} else {
		_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "FormType", OPTIONAL, model.V10, func(i int) bool { return i == 1 })
	}
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, sd.Dict, 0, dictName, "BBox", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, 0, dictName, "Matrix", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	// Resources, dict, optional, since V1.2
	if o, ok := sd.Find("Resources"); ok {
		_, err = validateResourceDict(c, xRefTable, o)
		if err != nil {
			return err
		}
	}

	// Group, dict, optional, since V1.4
	err = validateGroupEntry(c, xRefTable, sd.Dict, 0, dictName, OPTIONAL, model.V14)
	if err != nil {
		return err
	}

	// Ref, dict, optional, since V1.4
	d, err := validateDictEntry(xRefTable, sd.Dict, 0, dictName, "Ref", OPTIONAL, model.V14, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateReferenceDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// Metadata, stream, optional, since V1.4
	return validateMetadata(xRefTable, sd.Dict, OPTIONAL, model.V14)
}

func validateEntryOPI(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) (err error) {
	objNr := validationEntryObjectNumber(0, d, entryName)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d1, err := validateDictEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateOPIVersionDict(xRefTable, d1)
	}

	return err
}

func validateFormStreamPieceInfo(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		return nil
	}

	hasPieceInfo, err := validatePieceInfo(xRefTable, d, 0, dictName, "PieceInfo", OPTIONAL, model.V13)
	if err != nil {
		return err
	}

	// LastModified, date, required if PieceInfo present, since V1.3
	lm, err := validateDateEntry(xRefTable, d, 0, dictName, "LastModified", OPTIONAL, model.V13)
	if err != nil {
		return err
	}
	if hasPieceInfo && lm == nil {
		return errors.New("missing \"LastModified\" (required by \"PieceInfo\")")
	}

	return nil
}

func validateFormStreamDictPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// PieceInfo, dict, optional, since V1.3
	if err := validateFormStreamPieceInfo(xRefTable, d, dictName); err != nil {
		return err
	}

	// StructParent, integer
	sp, err := validateIntegerEntry(xRefTable, d, 0, dictName, "StructParent", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// StructParents, integer
	sps, err := validateIntegerEntry(xRefTable, d, 0, dictName, "StructParents", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}
	if sp != nil && sps != nil {
		return errors.New("only \"StructParent\" or \"StructParents\" allowed")
	}

	// OPI, dict, optional, since V1.2
	err = validateEntryOPI(xRefTable, d, dictName, "OPI", OPTIONAL, model.V12)
	if err != nil {
		return err
	}

	// OC, optional, content group dict or content membership dict, since V1.5
	// Specifying the optional content properties for the annotation.
	sinceVersion := model.V15
	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	if relaxed {
		sinceVersion = model.V12
	}
	err = validateOptionalContent(xRefTable, d, dictName, "OC", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}
	if d["OC"] != nil && relaxed && xRefTable.Version() < model.V15 {
		showDigestedVersionViolation(xRefTable, "dict="+dictName+" entry=OC")
	}

	// Name, name, optional (required in 1.0)
	required := xRefTable.Version() == model.V10
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Name", required, model.V10, nil)

	return err
}

func validateFormStreamDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict) error {
	// 8.10 Form XObjects

	dictName := "formStreamDict"

	err := validateFormStreamDictPart1(c, xRefTable, sd, dictName)
	if err != nil {
		return err
	}

	return validateFormStreamDictPart2(xRefTable, sd.Dict, dictName)
}

func validateXObjectType(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	ss := []string{"XObject"}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		ss = append(ss, "Xobject")
	}

	n, err := validateNameEntry(xRefTable, sd.Dict, 0, "xObjectStreamDict", "Type", OPTIONAL, model.V10, func(s string) bool { return types.MemberOf(s, ss) })
	if err != nil {
		return fmt.Errorf("xObjectStreamDict.Type: %w", err)
	}

	// Repair "Xobject" to "XObject".
	if n != nil && *n == "Xobject" {
		sd.Dict["Type"] = types.Name("XObject")
	}

	return nil
}

func validateXObjectStreamDictMissingSubtype(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict) error {
	_, found := sd.Find("BBox")
	if found {
		if err := validateFormStreamDict(c, xRefTable, sd); err != nil {
			return fmt.Errorf("xObject form stream dict: %w", err)
		}
		sd.Dict["Subtype"] = types.Name("Form")
		model.ShowRepaired("XObject stream dict missing Subtype inferred as Form")
		return nil
	}

	_, hasWidth := sd.Find("Width")
	_, hasHeight := sd.Find("Height")
	if !hasWidth || !hasHeight {
		return nil
	}

	if err := validateImageStreamDict(c, xRefTable, sd, isNoAlternateImageStreamDict); err != nil {
		return fmt.Errorf("xObject image stream dict: %w", err)
	}
	sd.Dict["Subtype"] = types.Name("Image")
	model.ShowRepaired("XObject stream dict missing Subtype inferred as Image")
	return nil
}

func validateXObjectStreamDictSubtype(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, subtype types.Name) error {
	var err error

	switch subtype {
	case "Form":
		err = validateFormStreamDict(c, xRefTable, sd)
	case "Image":
		err = validateImageStreamDict(c, xRefTable, sd, isNoAlternateImageStreamDict)
	case "PS":
		err = errors.New("PostScript XObjects should not be used")
	default:
		return fmt.Errorf("xObjectStreamDict.Subtype: unknown subtype %q", subtype)
	}

	if err != nil {
		return fmt.Errorf("xObject %s stream dict: %w", subtype, err)
	}
	return nil
}

func validateXObjectStreamDict(c context.Context, xRefTable *model.XRefTable, o types.Object) (err error) {
	// see 8.8 External Objects

	if o == nil {
		return nil
	}
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// Dereference stream dict and ensure it is validated exactly once in order
	// to handle XObjects(forms) with recursive structures like produced by Microsoft.
	sd, valid, err := xRefTable.DereferenceStreamDict(o)
	if valid {
		return nil
	}
	if err != nil {
		return fmt.Errorf("xObject stream dict: dereference stream dict: %w", err)
	}
	if sd == nil {
		return nil
	}
	return validateXObjectStreamDictContents(c, xRefTable, sd)
}

func validateXObjectStreamDictContents(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict) error {
	dictName := "xObjectStreamDict"

	if err := validateXObjectType(xRefTable, sd); err != nil {
		return err
	}

	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	subtype, err := validateNameEntry(xRefTable, sd.Dict, 0, dictName, "Subtype", required, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.Subtype: %w", dictName, err)
	}

	if subtype == nil || len(*subtype) == 0 {
		return validateXObjectStreamDictMissingSubtype(c, xRefTable, sd)
	}

	return validateXObjectStreamDictSubtype(c, xRefTable, sd, *subtype)
}

func validateGroupAttributesDict(c context.Context, xRefTable *model.XRefTable, o types.Object) (err error) {
	// see 11.6.6 Transparency Group XObjects
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("group attributes dict: dereference dict: %w", err)
	}
	if d == nil {
		return nil
	}

	dictName := "groupAttributesDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, d, objNr, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Group" })
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, d, objNr, dictName, "S", REQUIRED, model.V10, func(s string) bool { return s == "Transparency" })
	if err != nil {
		return fmt.Errorf("%s.S: %w", dictName, err)
	}

	// CS, colorSpace, optional
	err = validateColorSpaceEntry(c, xRefTable, d, objNr, dictName, "CS", OPTIONAL, colorSpaceNoPattern)
	if err != nil {
		return fmt.Errorf("%s.CS: %w", dictName, err)
	}

	// I, boolean, optional
	_, err = validateBooleanEntry(xRefTable, d, objNr, dictName, "I", OPTIONAL, model.V10, nil)

	if err != nil {
		return fmt.Errorf("%s.I: %w", dictName, err)
	}
	return nil
}

func validateXObjectResourceDict(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	resourceObjNr := validationObjectNumber(0, o)

	// Version check
	err := xRefTable.ValidateVersion("XObjectResourceDict", sinceVersion)
	if err != nil {
		err = fmt.Errorf("XObject resource dict: version: %w", err)
		return model.WithValidationErrorObject(err, resourceObjNr)
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		err = fmt.Errorf("XObject resource dict: dereference dict: %w", err)
		return model.WithValidationErrorObject(err, resourceObjNr)
	}
	if d == nil {
		return nil
	}

	//fmt.Printf("XObjResDict:\n%s\n", d)

	// Iterate over XObject resource dictionary
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		o := d[name]
		xObjectObjNr := validationObjectNumber(resourceObjNr, o)
		// Process XObject dict
		err = validateXObjectStreamDict(c, xRefTable, o)
		if err != nil {
			err = fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("XObject resource %s", name), o), err)
			return model.WithValidationErrorObject(err, xObjectObjNr)
		}
	}

	return nil
}

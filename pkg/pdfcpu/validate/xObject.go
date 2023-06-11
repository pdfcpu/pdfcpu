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
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

const (

	// ExcludePatternCS ...
	ExcludePatternCS = true

	// IncludePatternCS ...
	IncludePatternCS = false

	isAlternateImageStreamDict   = true
	isNoAlternateImageStreamDict = false
)

func validateReferenceDictPageEntry(xRefTable *model.XRefTable, o types.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o.(type) {

	case types.Integer, types.StringLiteral, types.HexLiteral:
		// no further processing

	default:
		return errors.New("pdfcpu: validateReferenceDictPageEntry: corrupt type")

	}

	return nil
}

func validateReferenceDict(xRefTable *model.XRefTable, d types.Dict) error {

	// see 8.10.4 Reference XObjects

	dictName := "refDict"

	// F, file spec, required
	_, err := validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// Page, integer or text string, required
	o, ok := d.Find("Page")
	if !ok {
		return errors.New("pdfcpu: validateReferenceDict: missing required entry \"Page\"")
	}

	err = validateReferenceDictPageEntry(xRefTable, o)
	if err != nil {
		return err
	}

	// ID, string array, optional
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "ID", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })

	return err
}

func validateOPIDictV13Part1(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return err
	}

	// Version, required, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "Version", REQUIRED, model.V10, func(f float64) bool { return f == 1.3 })
	if err != nil {
		return err
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// ID, optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "ID", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Comments, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Comments", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Size, required, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "Size", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// CropRect, required, array of integers, len 4
	_, err = validateRectangleEntry(xRefTable, d, dictName, "CropRect", REQUIRED, model.V10, nil)

	if err != nil {
		return err
	}

	// CropFixed, optional, array of numbers, len 4
	_, err = validateRectangleEntry(xRefTable, d, dictName, "CropFixed", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Position, required, array of numbers, len 8
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Position", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 8 })

	return err
}

func validateOPIDictV13Part2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// Resolution, optional, array of numbers, len 2
	_, err := validateNumberArrayEntry(xRefTable, d, dictName, "Resolution", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// ColorType, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "ColorType", OPTIONAL, model.V10, func(s string) bool { return s == "Process" || s == "Spot" || s == "Separation" })
	if err != nil {
		return err
	}

	// Color, optional, array, len 5
	_, err = validateArrayEntry(xRefTable, d, dictName, "Color", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 5 })
	if err != nil {
		return err
	}

	// Tint, optional, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "Tint", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Overprint, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Overprint", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// ImageType, optional, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "ImageType", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// GrayMap, optional, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "GrayMap", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Transparency, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "Transparency", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Tags, optional, array
	_, err = validateArrayEntry(xRefTable, d, dictName, "Tags", OPTIONAL, model.V10, nil)

	return err
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

func validateOPIDictInks(xRefTable *model.XRefTable, o types.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Name:
		if colorant := o.Value(); colorant != "full_color" && colorant != "registration" {
			return errors.New("pdfcpu: validateOPIDictInks: corrupt colorant name")
		}

	case types.Array:
		// no further processing

	default:
		return errors.New("pdfcpu: validateOPIDictInks: corrupt type")

	}

	return nil
}

func validateOPIDictV20(xRefTable *model.XRefTable, d types.Dict) error {

	// 14.11.7 Open Prepresse interface (OPI)

	dictName := "opiDictV20"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, d, dictName, "Version", REQUIRED, model.V10, func(f float64) bool { return f == 2.0 })
	if err != nil {
		return err
	}

	_, err = validateFileSpecEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, dictName, "MainImage", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateArrayEntry(xRefTable, d, dictName, "Tags", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Size", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, d, dictName, "CropRect", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, dictName, "Overprint", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	if o, found := d.Find("Inks"); found {
		err = validateOPIDictInks(xRefTable, o)
		if err != nil {
			return err
		}
	}

	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "IncludedImageDimensions", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, d, dictName, "IncludedImageQuality", OPTIONAL, model.V10, func(i int) bool { return i >= 1 && i <= 3 })

	return err
}

func validateOPIVersionDict(xRefTable *model.XRefTable, d types.Dict) error {

	// 14.11.7 Open Prepresse interface (OPI)

	if d.Len() != 1 {
		return errors.New("pdfcpu: validateOPIVersionDict: must have exactly one entry keyed 1.3 or 2.0")
	}

	validateOPIVersion := func(s string) bool { return types.MemberOf(s, []string{"1.3", "2.0"}) }

	for opiVersion, obj := range d {

		if !validateOPIVersion(opiVersion) {
			return errors.New("pdfcpu: validateOPIVersionDict: invalid OPI version")
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

func validateMaskStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {

	if sd.Type() != nil && *sd.Type() != "XObject" {
		return errors.New("pdfcpu: validateMaskStreamDict: corrupt imageStreamDict type")
	}

	if sd.Subtype() == nil || *sd.Subtype() != "Image" {
		return errors.New("pdfcpu: validateMaskStreamDict: corrupt imageStreamDict subtype")
	}

	return validateImageStreamDict(xRefTable, sd, isNoAlternateImageStreamDict)
}

func validateMaskEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// stream ("explicit masking", another Image XObject) or array of colors ("color key masking")

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.StreamDict:
		err = validateMaskStreamDict(xRefTable, &o)
		if err != nil {
			return err
		}

	case types.Array:
		// no further processing

	default:

		return errors.Errorf("pdfcpu: validateMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	return nil
}

func validateAlternateImageStreamDicts(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) error {

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}
	if a == nil {
		if required {
			return errors.Errorf("pdfcpu: validateAlternateImageStreamDicts: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		return nil
	}

	for _, o := range a {

		sd, err := validateStreamDict(xRefTable, o)
		if err != nil {
			return err
		}

		if sd == nil {
			continue
		}

		err = validateImageStreamDict(xRefTable, sd, isAlternateImageStreamDict)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateImageStreamDictPart1(xRefTable *model.XRefTable, sd *types.StreamDict, dictName string) (isImageMask bool, err error) {

	// Width, integer, required
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Width", REQUIRED, model.V10, nil)
	if err != nil {
		return false, err
	}

	// Height, integer, required
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "Height", REQUIRED, model.V10, nil)
	if err != nil {
		return false, err
	}

	// ImageMask, boolean, optional
	imageMask, err := validateBooleanEntry(xRefTable, sd.Dict, dictName, "ImageMask", OPTIONAL, model.V10, nil)
	if err != nil {
		return false, err
	}

	isImageMask = (imageMask != nil) && *imageMask == true

	// ColorSpace, name or array, required unless used filter is JPXDecode; not allowed for imagemasks.
	if !isImageMask {

		required := REQUIRED

		if sd.HasSoleFilterNamed(filter.JPX) {
			required = OPTIONAL
		}

		if sd.HasSoleFilterNamed(filter.CCITTFax) && xRefTable.ValidationMode == model.ValidationRelaxed {
			required = OPTIONAL
		}

		err = validateColorSpaceEntry(xRefTable, sd.Dict, dictName, "ColorSpace", required, ExcludePatternCS)
		if err != nil {
			return false, err
		}

	}

	return isImageMask, nil
}

func validateImageStreamDictPart2(xRefTable *model.XRefTable, sd *types.StreamDict, dictName string, isImageMask, isAlternate bool) error {

	// BitsPerComponent, integer
	required := REQUIRED
	if sd.HasSoleFilterNamed(filter.JPX) || isImageMask {
		required = OPTIONAL
	}
	// For imageMasks BitsPerComponent must be 1.
	var validateBPC func(i int) bool
	if isImageMask {
		validateBPC = func(i int) bool {
			return i == 1
		}
	}
	_, err := validateIntegerEntry(xRefTable, sd.Dict, dictName, "BitsPerComponent", required, model.V10, validateBPC)
	if err != nil {
		return err
	}

	// Intent, name, optional, since V1.0
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"AbsoluteColorimetric", "RelativeColorimetric", "Saturation", "Perceptual"})
	}
	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Intent", OPTIONAL, model.V11, validate)
	if err != nil {
		return err
	}

	// Mask, stream or array, optional since V1.3; not allowed for image masks.
	if !isImageMask {
		err = validateMaskEntry(xRefTable, sd.Dict, dictName, "Mask", OPTIONAL, model.V13)
		if err != nil {
			return err
		}
	}

	// Decode, array, optional
	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Decode", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Interpolate, boolean, optional
	_, err = validateBooleanEntry(xRefTable, sd.Dict, dictName, "Interpolate", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Alternates, array, optional, since V1.3
	if !isAlternate {
		err = validateAlternateImageStreamDicts(xRefTable, sd.Dict, dictName, "Alternates", OPTIONAL, model.V13)
	}

	return err
}

func validateImageStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict, isAlternate bool) error {
	dictName := "imageStreamDict"
	var isImageMask bool

	isImageMask, err := validateImageStreamDictPart1(xRefTable, sd, dictName)
	if err != nil {
		return err
	}

	err = validateImageStreamDictPart2(xRefTable, sd, dictName, isImageMask, isAlternate)
	if err != nil {
		return err
	}

	// SMask, stream, optional, since V1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	sd1, err := validateStreamDictEntry(xRefTable, sd.Dict, dictName, "SMask", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if sd1 != nil {
		err = validateImageStreamDict(xRefTable, sd1, isNoAlternateImageStreamDict)
		if err != nil {
			return err
		}
	}

	// SMaskInData, integer, optional
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "SMaskInData", OPTIONAL, model.V10, func(i int) bool { return i >= 0 && i <= 2 })
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
	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "StructParent", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// ID, byte string, optional, since V1.3
	_, err = validateStringEntry(xRefTable, sd.Dict, dictName, "ID", OPTIONAL, model.V13, nil)
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
	return validateEntryOC(xRefTable, sd.Dict, dictName, "OC", OPTIONAL, model.V15)
}

func validateFormStreamDictPart1(xRefTable *model.XRefTable, sd *types.StreamDict, dictName string) error {
	var err error
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		_, err = validateNumberEntry(xRefTable, sd.Dict, dictName, "FormType", OPTIONAL, model.V10, func(f float64) bool { return f == 1. })
	} else {
		_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "FormType", OPTIONAL, model.V10, func(i int) bool { return i == 1 })
	}
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, sd.Dict, dictName, "BBox", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Matrix", OPTIONAL, model.V10, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	// Resources, dict, optional, since V1.2
	if o, ok := sd.Find("Resources"); ok {
		_, err = validateResourceDict(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// Group, dict, optional, since V1.4
	err = validatePageEntryGroup(xRefTable, sd.Dict, OPTIONAL, model.V14)
	if err != nil {
		return err
	}

	// Ref, dict, optional, since V1.4
	d, err := validateDictEntry(xRefTable, sd.Dict, dictName, "Ref", OPTIONAL, model.V14, nil)
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

func validateEntryOC(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateOptionalContentGroupDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateEntryOPI(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateOPIVersionDict(xRefTable, d1)
	}

	return err
}

func validateFormStreamDictPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {

	// PieceInfo, dict, optional, since V1.3
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		hasPieceInfo, err := validatePieceInfo(xRefTable, d, dictName, "PieceInfo", OPTIONAL, model.V13)
		if err != nil {
			return err
		}

		// LastModified, date, required if PieceInfo present, since V1.3
		lm, err := validateDateEntry(xRefTable, d, dictName, "LastModified", OPTIONAL, model.V13)
		if err != nil {
			return err
		}

		if hasPieceInfo && lm == nil {
			err = errors.New("pdfcpu: validateFormStreamDictPart2: missing \"LastModified\" (required by \"PieceInfo\")")
			return err
		}
	}

	// StructParent, integer
	sp, err := validateIntegerEntry(xRefTable, d, dictName, "StructParent", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// StructParents, integer
	sps, err := validateIntegerEntry(xRefTable, d, dictName, "StructParents", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}
	if sp != nil && sps != nil {
		return errors.New("pdfcpu: validateFormStreamDictPart2: only \"StructParent\" or \"StructParents\" allowed")
	}

	// OPI, dict, optional, since V1.2
	err = validateEntryOPI(xRefTable, d, dictName, "OPI", OPTIONAL, model.V12)
	if err != nil {
		return err
	}

	// OC, optional, content group dict or content membership dict, since V1.5
	// Specifying the optional content properties for the annotation.
	sinceVersion := model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	err = validateOptionalContent(xRefTable, d, dictName, "OC", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// Name, name, optional (required in 1.0)
	required := xRefTable.Version() == model.V10
	_, err = validateNameEntry(xRefTable, d, dictName, "Name", required, model.V10, nil)

	return err
}

func validateFormStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {

	// 8.10 Form XObjects

	dictName := "formStreamDict"

	err := validateFormStreamDictPart1(xRefTable, sd, dictName)
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

	n, err := validateNameEntry(xRefTable, sd.Dict, "xObjectStreamDict", "Type", OPTIONAL, model.V10, func(s string) bool { return types.MemberOf(s, ss) })
	if err != nil {
		return err
	}

	// Repair "Xobject" to "XObject".
	if n != nil && *n == "Xobject" {
		sd.Dict["Type"] = types.Name("XObject")
	}

	return nil
}

func validateXObjectStreamDict(xRefTable *model.XRefTable, o types.Object) error {

	// see 8.8 External Objects

	// Dereference stream dict and ensure it is validated exactly once in order
	// to handle XObjects(forms) with recursive structures like produced by Microsoft.
	sd, valid, err := xRefTable.DereferenceStreamDict(o)
	if valid {
		return nil
	}
	if err != nil || sd == nil {
		return err
	}

	dictName := "xObjectStreamDict"

	if err := validateXObjectType(xRefTable, sd); err != nil {
		return err
	}

	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	subtype, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Subtype", required, model.V10, nil)
	if err != nil {
		return err
	}

	if subtype == nil {
		// relaxed
		_, found := sd.Find("BBox")
		if found {
			return validateFormStreamDict(xRefTable, sd)
		}

		// Relaxed for page Thumb
		return validateImageStreamDict(xRefTable, sd, isNoAlternateImageStreamDict)
	}

	switch *subtype {

	case "Form":
		err = validateFormStreamDict(xRefTable, sd)

	case "Image":
		err = validateImageStreamDict(xRefTable, sd, isNoAlternateImageStreamDict)

	case "PS":
		err = errors.Errorf("pdfcpu: validateXObjectStreamDict: PostScript XObjects should not be used")

	default:
		return errors.Errorf("pdfcpu: validateXObjectStreamDict: unknown Subtype: %s\n", *subtype)

	}

	return err
}

func validateGroupAttributesDict(xRefTable *model.XRefTable, o types.Object) error {

	// see 11.6.6 Transparency Group XObjects

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictName := "groupAttributesDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Group" })
	if err != nil {
		return err
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, model.V10, func(s string) bool { return s == "Transparency" })
	if err != nil {
		return err
	}

	// CS, colorSpace, optional
	err = validateColorSpaceEntry(xRefTable, d, dictName, "CS", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return err
	}

	// I, boolean, optional
	_, err = validateBooleanEntry(xRefTable, d, dictName, "I", OPTIONAL, model.V10, nil)

	return err
}

func validateXObjectResourceDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("XObjectResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	//fmt.Printf("XObjResDict:\n%s\n", d)

	// Iterate over XObject resource dictionary
	for _, o := range d {

		// Process XObject dict
		err = validateXObjectStreamDict(xRefTable, o)
		if err != nil {
			return err
		}
	}

	return nil
}

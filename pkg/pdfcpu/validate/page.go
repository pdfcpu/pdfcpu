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
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type resourceDictValidator struct {
	name         string
	sinceVersion model.Version
}

var resourceDictValidators = []resourceDictValidator{
	{"ExtGState", model.V10},
	{"Font", model.V10},
	{"XObject", model.V10},
	{"Properties", model.V10},
	{"ColorSpace", model.V10},
	{"Pattern", model.V10},
	{"Shading", model.V13},
}

func validateResourceCategory(
	xRefTable *model.XRefTable,
	name string,
	o types.Object,
	sinceVersion model.Version,
) error {
	switch name {
	case "ExtGState":
		return validateExtGStateResourceDict(xRefTable, o, sinceVersion)
	case "Font":
		return validateFontResourceDict(xRefTable, o, sinceVersion)
	case "XObject":
		return validateXObjectResourceDict(xRefTable, o, sinceVersion)
	case "Properties":
		return validatePropertiesResourceDict(xRefTable, o, sinceVersion)
	case "ColorSpace":
		return validateColorSpaceResourceDict(xRefTable, o, sinceVersion)
	case "Pattern":
		return validatePatternResourceDict(xRefTable, o, sinceVersion)
	case "Shading":
		return validateShadingResourceDict(xRefTable, o, sinceVersion)
	}
	return nil
}

func validateResourceDict(xRefTable *model.XRefTable, o types.Object) (hasResources bool, err error) {
	resourceObjNr := validationObjectNumber(0, o)
	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return false, model.WithValidationErrorObject(err, resourceObjNr)
	}

	for _, v := range resourceDictValidators {
		if o, ok := d.Find(v.name); ok {
			categoryObjNr := validationObjectNumber(resourceObjNr, o)
			err = validateResourceCategory(xRefTable, v.name, o, v.sinceVersion)
			if err != nil {
				return false, model.WithValidationErrorObject(err, categoryObjNr)
			}
		}
	}

	allowedResDictKeys := []string{"ExtGState", "Font", "XObject", "Properties", "ColorSpace", "Pattern", "ProcSet", "Shading"}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		allowedResDictKeys = append(allowedResDictKeys, "Encoding")
		allowedResDictKeys = append(allowedResDictKeys, "ProcSets")
	}

	// Note: Beginning with PDF V1.4 the "ProcSet" feature is considered to be obsolete!

	for k := range d {
		if !types.MemberOf(k, allowedResDictKeys) {
			d.Delete(k)
		}
	}

	return true, nil
}

func validateContents(obj types.Object, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (hasContents bool, err error) {
	switch obj := obj.(type) {

	case types.StreamDict:
		// no further processing.
		hasContents = true

	case types.Array:
		// process array of content stream dicts.

		for _, obj := range obj {
			objNr := validationObjectNumber(ownerObjNr, obj)
			o1, _, err := xRefTable.DereferenceStreamDict(obj)
			if err != nil {
				return false, model.WithValidationErrorObject(err, objNr)
			}

			if o1 == nil {
				continue
			}

			hasContents = true

		}

		if hasContents {
			break
		}

		if xRefTable.ValidationMode == model.ValidationStrict {
			err := errors.New("page contents: empty content array")
			return false, model.WithValidationErrorObject(err, ownerObjNr)
		}

		// Digest empty array.
		d.Delete("Contents")
		model.ShowRepaired("page dict \"Contents\"")

	case types.StringLiteral:

		s := strings.TrimSpace(obj.Value())

		if len(s) > 0 || xRefTable.ValidationMode == model.ValidationStrict {
			err := fmt.Errorf("page contents: expected stream dict or array, got %T", obj)
			return false, model.WithValidationErrorObject(err, ownerObjNr)
		}

		// Digest empty string literal.
		d.Delete("Contents")
		model.ShowRepaired("page dict \"Contents\"")

	case types.Dict:

		if len(obj) > 0 || xRefTable.ValidationMode == model.ValidationStrict {
			err := fmt.Errorf("page contents: expected stream dict or array, got %T", obj)
			return false, model.WithValidationErrorObject(err, ownerObjNr)
		}

		// Digest empty dict.
		d.Delete("Contents")
		model.ShowRepaired("page dict \"Contents\"")

	default:
		err := fmt.Errorf("page contents: expected stream dict or array, got %T", obj)
		return false, model.WithValidationErrorObject(err, ownerObjNr)
	}

	return hasContents, nil
}

func validatePageContents(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (hasContents bool, err error) {
	o, found := d.Find("Contents")
	if !found {
		return false, err
	}

	contentsObjNr := validationObjectNumber(ownerObjNr, o)
	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return false, model.WithValidationErrorObject(err, contentsObjNr)
	}

	return validateContents(o, xRefTable, d, contentsObjNr)
}

func validatePageResources(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	if o, found := d.Find("Resources"); found {
		_, err := validateResourceDict(xRefTable, o)
		return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
	}

	return nil
}

func validatePageEntryMediaBox(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) (types.Array, error) {
	return validateRectangleEntry(xRefTable, d, ownerObjNr, "pageDict", "MediaBox", required, sinceVersion, nil)
}

func validatePageEntryCropBox(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateRectangleEntry(xRefTable, d, ownerObjNr, "pagesDict", "CropBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryBleedBox(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateRectangleEntry(xRefTable, d, ownerObjNr, "pagesDict", "BleedBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryTrimBox(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateRectangleEntry(xRefTable, d, ownerObjNr, "pagesDict", "TrimBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryArtBox(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateRectangleEntry(xRefTable, d, ownerObjNr, "pagesDict", "ArtBox", required, sinceVersion, nil)

	return err
}

func validateBoxStyleDictEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, entryName string, required bool, sinceVersion model.Version) error {
	boxStyleObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "boxStyleDict"

	// C, number array with 3 elements, optional
	_, err = validateNumberArrayEntry(xRefTable, d1, boxStyleObjNr, dictName, "C", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// W, number, optional
	_, err = validateNumberEntry(xRefTable, d1, boxStyleObjNr, dictName, "W", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// S, name, optional
	validate := func(s string) bool { return types.MemberOf(s, []string{"S", "D"}) }
	_, err = validateNameEntry(xRefTable, d1, boxStyleObjNr, dictName, "S", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	// D, array, optional, since V1.3, dashArray
	_, err = validateIntegerArrayEntry(xRefTable, d1, boxStyleObjNr, dictName, "D", OPTIONAL, sinceVersion, nil)

	return err
}

func validatePageBoxColorInfo(xRefTable *model.XRefTable, pageDict types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// box color information dict
	// see 14.11.2.2

	dictName := "pageDict"

	boxColorInfoObjNr := validationEntryObjectNumber(ownerObjNr, pageDict, "BoxColorInfo")
	d, err := validateDictEntry(xRefTable, pageDict, ownerObjNr, dictName, "BoxColorInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "boxColorInfoDict"

	err = validateBoxStyleDictEntry(xRefTable, d, boxColorInfoObjNr, dictName, "CropBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateBoxStyleDictEntry(xRefTable, d, boxColorInfoObjNr, dictName, "BleedBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateBoxStyleDictEntry(xRefTable, d, boxColorInfoObjNr, dictName, "TrimBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	return validateBoxStyleDictEntry(xRefTable, d, boxColorInfoObjNr, dictName, "ArtBox", OPTIONAL, sinceVersion)
}

func validatePageEntryRotate(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	validate := func(i int) bool { return i%90 == 0 }
	_, err := validateIntegerEntry(xRefTable, d, ownerObjNr, "pagesDict", "Rotate", required, sinceVersion, validate)

	return err
}

func validateGroupEntry(
	xRefTable *model.XRefTable,
	d types.Dict,
	ownerObjNr int,
	dictName string,
	required bool,
	sinceVersion model.Version,
) error {
	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	strictSinceVersion := sinceVersion
	if relaxed {
		sinceVersion = model.V12
	}

	groupObjNr := validationEntryObjectNumber(ownerObjNr, d, "Group")
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "Group", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateGroupAttributesDict(xRefTable, d1)
		err = model.WithValidationErrorObject(err, groupObjNr)
		if err == nil && relaxed && xRefTable.Version() < strictSinceVersion {
			showDigestedVersionViolation(xRefTable, "dict="+dictName+" entry=Group")
		}
	}

	return err
}

func validatePageEntryGroup(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	return validateGroupEntry(xRefTable, d, ownerObjNr, "pageDict", required, sinceVersion)
}

func validatePageEntryThumb(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	thumbObjNr := validationEntryObjectNumber(ownerObjNr, d, "Thumb")
	sd, err := validateStreamDictEntry(xRefTable, d, ownerObjNr, "pagesDict", "Thumb", required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	if err := validateXObjectStreamDict(xRefTable, *sd); err != nil {
		return model.WithValidationErrorObject(err, thumbObjNr)
	}

	indRef := d.IndirectRefEntry("Thumb")
	xRefTable.PageThumbs[xRefTable.CurPage] = *indRef
	//fmt.Printf("adding thumb page:%d obj#:%d\n", xRefTable.CurPage, indRef.ObjectNumber.Value())

	return nil
}

func validatePageEntryB(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// Note: Only makes sense if "Threads" entry in document root and bead dicts present.

	_, err := validateIndRefArrayEntry(xRefTable, d, 0, "pagesDict", "B", required, sinceVersion, nil)

	return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "B"))
}

func validatePageEntryDur(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateNumberEntry(xRefTable, d, ownerObjNr, "pagesDict", "Dur", required, sinceVersion, nil)

	return err
}

func validateTransitionDictEntryDi(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	o, found := d.Find("Di")
	if !found {
		return nil
	}

	objNr := validationObjectNumber(ownerObjNr, o)
	if ir, ok := o.(types.IndirectRef); ok {
		entry, found := xRefTable.FindTableEntryForIndRef(&ir)
		if !found || entry == nil || entry.Free {
			err := errors.New("transition dict: entry Di missing indirect target")
			return model.WithValidationErrorObject(err, objNr)
		}
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}
	if o == nil {
		return nil
	}

	validNumber := func(f float64) bool {
		switch f {
		case 0, 90, 180, 270, 315:
			return true
		}
		return false
	}

	switch o := o.(type) {

	case types.Integer:
		if !validNumber(float64(o.Value())) {
			err := errors.New("transition dict: entry Di number value undefined")
			return model.WithValidationErrorObject(err, objNr)
		}

	case types.Float:
		if !validNumber(o.Value()) {
			err := errors.New("transition dict: entry Di number value undefined")
			return model.WithValidationErrorObject(err, objNr)
		}

	case types.Name:
		if o.Value() != "None" {
			err := errors.New("transition dict: entry Di name value undefined")
			return model.WithValidationErrorObject(err, objNr)
		}

	default:
		err := fmt.Errorf("transition dict: entry Di invalid type %T", o)
		return model.WithValidationErrorObject(err, objNr)
	}

	return nil
}

func validateTransitionDictEntryM(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, transStyle *types.Name) error {
	// see 12.4.4
	validateTransitionDirectionOfMotion := func(s string) bool { return types.MemberOf(s, []string{"I", "O"}) }

	validateM := func(s string) bool {
		return validateTransitionDirectionOfMotion(s) &&
			(transStyle != nil && (*transStyle == "Split" || *transStyle == "Box" || *transStyle == "Fly"))
	}

	_, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "M", OPTIONAL, model.V10, validateM)

	return err
}

func repairTransitionStyle(xRefTable *model.XRefTable, d types.Dict) error {
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		return nil
	}

	s, _, err := xRefTable.DereferenceNameEntry(d, "S")
	if err != nil {
		return err
	}
	if s == nil || s.Value() != "Replace" {
		return nil
	}

	d.Update("S", types.Name("R"))
	model.ShowRepaired("transition style Replace converted to R")
	return nil
}

func validateTransitionDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	dictName := "transitionDict"
	if err := repairTransitionStyle(xRefTable, d); err != nil {
		return fmt.Errorf("%s.S: %w", dictName, err)
	}

	// S, name, optional

	styles := []string{"Split", "Blinds", "Box", "Wipe", "Dissolve", "Glitter", "R"}
	if xRefTable.Version() >= model.V15 {
		styles = append(styles, "Fly", "Push", "Cover", "Uncover", "Fade")
	}
	validate := func(s string) bool { return types.MemberOf(s, styles) }
	transStyle, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "S", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}

	// D, optional, number > 0
	_, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "D", OPTIONAL, model.V10, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// Dm, optional, name, see 12.4.4
	validateTransitionDimension := func(s string) bool { return types.MemberOf(s, []string{"H", "V"}) }

	validateDm := func(s string) bool {
		return validateTransitionDimension(s) && (transStyle != nil && (*transStyle == "Split" || *transStyle == "Blinds"))
	}
	_, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Dm", OPTIONAL, model.V10, validateDm)
	if err != nil {
		return err
	}

	// M, optional, name
	err = validateTransitionDictEntryM(xRefTable, d, ownerObjNr, dictName, transStyle)
	if err != nil {
		return err
	}

	// Di, optional, number or name
	err = validateTransitionDictEntryDi(xRefTable, d, ownerObjNr)
	if err != nil {
		return err
	}

	// SS, optional, number, since V1.5
	if transStyle != nil && *transStyle == "Fly" {
		_, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "SS", OPTIONAL, model.V15, nil)
		if err != nil {
			return err
		}
	}

	// B, optional, boolean, since V1.5
	validateB := func(b bool) bool { return transStyle != nil && *transStyle == "Fly" }
	_, err = validateBooleanEntry(xRefTable, d, ownerObjNr, dictName, "B", OPTIONAL, model.V15, validateB)

	return err
}

func validatePageEntryTrans(xRefTable *model.XRefTable, pageDict types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	transitionObjNr := validationEntryObjectNumber(ownerObjNr, pageDict, "Trans")
	d, err := validateDictEntry(xRefTable, pageDict, ownerObjNr, "pagesDict", "Trans", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return validateTransitionDict(xRefTable, d, transitionObjNr)
}

func validatePageEntryStructParents(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateIntegerEntry(xRefTable, d, ownerObjNr, "pagesDict", "StructParents", required, sinceVersion, nil)

	return err
}

func validatePageEntryID(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	_, err := validateStringEntry(xRefTable, d, ownerObjNr, "pagesDict", "ID", required, sinceVersion, nil)

	return err
}

func validatePageEntryPZ(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// Preferred zoom factor, number

	_, err := validateNumberEntry(xRefTable, d, ownerObjNr, "pagesDict", "PZ", required, sinceVersion, nil)

	return err
}

func validatePageEntrySeparationInfo(xRefTable *model.XRefTable, pagesDict types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// see 14.11.4

	separationObjNr := validationEntryObjectNumber(ownerObjNr, pagesDict, "SeparationInfo")
	d, err := validateDictEntry(xRefTable, pagesDict, ownerObjNr, "pagesDict", "SeparationInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName := "separationDict"

	_, err = validateIndRefArrayEntry(xRefTable, d, 0, dictName, "Pages", REQUIRED, sinceVersion, nil)
	if err != nil {
		return model.WithValidationErrorObject(err, validationEntryObjectNumber(separationObjNr, d, "Pages"))
	}

	err = validateNameOrStringEntry(
		xRefTable, d, separationObjNr, dictName, "DeviceColorant", required, sinceVersion,
	)
	if err != nil {
		return err
	}

	a, err := validateArrayEntry(xRefTable, d, separationObjNr, dictName, "ColorSpace", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if a != nil {
		colorSpaceObjNr := validationEntryObjectNumber(separationObjNr, d, "ColorSpace")
		err = validateColorSpaceArraySubset(
			xRefTable, a, colorSpaceObjNr, []string{"Separation", "DeviceN"},
		)
		err = model.WithValidationErrorObject(err, colorSpaceObjNr)
	}

	return err
}

func validatePageEntryTabs(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	validateTabs := func(s string) bool { return types.MemberOf(s, []string{"R", "C", "S", "A", "W"}) }

	_, err := validateNameEntry(xRefTable, d, ownerObjNr, "pagesDict", "Tabs", required, sinceVersion, validateTabs)

	if err != nil && xRefTable.ValidationMode == model.ValidationRelaxed {
		_, err = validateStringEntry(xRefTable, d, ownerObjNr, "pagesDict", "Tabs", required, sinceVersion, validateTabs)
	}

	return err
}

func validatePageEntryTemplateInstantiated(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// see 12.7.6

	_, err := validateNameEntry(xRefTable, d, ownerObjNr, "pagesDict", "TemplateInstantiated", required, sinceVersion, nil)

	return err
}

// TODO implement
func validatePageEntryPresSteps(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// see 12.4.4.2

	presStepsObjNr := validationEntryObjectNumber(ownerObjNr, d, "PresSteps")
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, "pagesDict", "PresSteps", required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	err = errors.New("presentation steps dict: missing supported entry NA")
	return model.WithValidationErrorObject(err, presStepsObjNr)
}

func validatePageEntryUserUnit(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// UserUnit, optional, positive number, since V1.6
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err := validateNumberEntry(xRefTable, d, ownerObjNr, "pagesDict", "UserUnit", required, sinceVersion, func(f float64) bool { return f > 0 })

	return err
}

func validateNumberFormatDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "numberFormatDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "NumberFormat" })
	if err != nil {
		return err
	}

	// U, text string, required
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "U", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, number, required
	_, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "C", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, name, optional
	format, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "F", OPTIONAL, sinceVersion, func(s string) bool {
		return types.MemberOf(s, []string{"D", "F", "R", "T"})
	})
	if err != nil {
		return err
	}

	// D, integer, optional
	precision, err := validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "D", OPTIONAL, sinceVersion, func(i int) bool {
		return i > 0
	})
	if err != nil {
		return err
	}
	if precision != nil && (format == nil || format.Value() == "D") && precision.Value()%10 != 0 {
		err := fmt.Errorf("%s.D: decimal precision must be a multiple of 10", dictName)
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	// FD, bool, optional
	_, err = validateBooleanEntry(xRefTable, d, ownerObjNr, dictName, "FD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RT, text string, optional
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "RT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RD, text string, optional
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "RD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// PS, text string, optional
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "PS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SS, text string, optional
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "SS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// O, name, optional
	_, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "O", OPTIONAL, sinceVersion, func(s string) bool {
		return types.MemberOf(s, []string{"S", "P"})
	})

	return err
}

func validateNumberFormatArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	arrayObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	a, err := validateArrayEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, func(a types.Array) bool {
		return len(a) > 0
	})
	if err != nil || a == nil {
		return err
	}

	for i, v := range a {
		objNr := validationObjectNumber(arrayObjNr, v)
		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		if d == nil {
			err = fmt.Errorf("%s.%s[%d]: missing number format dict", dictName, entryName, i)
			return model.WithValidationErrorObject(err, objNr)
		}
		err = validateNumberFormatDict(xRefTable, d, objNr, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
		}
	}

	return nil
}

func validateRectilinearMeasureDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "rectilinearMeasureDict"

	// R, text string, required, scale ratio
	_, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// X, number format array, required, for measurement of change along the x axis and, if Y is not present,
	// along the y axis as well.
	err = validateNumberFormatArrayEntry(xRefTable, d, ownerObjNr, dictName, "X", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// Y, number format array, required when the x and y scales have different units or conversion factors.
	err = validateNumberFormatArrayEntry(xRefTable, d, ownerObjNr, dictName, "Y", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// D, number format array, required, for measurement of distance in any direction.
	err = validateNumberFormatArrayEntry(xRefTable, d, ownerObjNr, dictName, "D", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// A, number format array, required, for measurement of area.
	err = validateNumberFormatArrayEntry(xRefTable, d, ownerObjNr, dictName, "A", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// T, number format array, optional, for measurement of angles.
	err = validateNumberFormatArrayEntry(xRefTable, d, ownerObjNr, dictName, "T", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// S, number format array, optional, for fmeasurement of the slope of a line.
	err = validateNumberFormatArrayEntry(xRefTable, d, ownerObjNr, dictName, "S", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// O, number array, optional, array of two numbers that shall specify the origin of the measurement coordinate system
	// in default user space coordinates.
	_, err = validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "O", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// CYX, number, optional, a factor that shall be used to convert the largest units along the y axis to the largest
	// units along the x axis.
	_, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "CYX", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateMeasureCoordinateSystemDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "measureCoordinateSystemDict"

	validateType := func(s string) bool { return types.MemberOf(s, []string{"GEOGCS", "PROJCS"}) }
	csType, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", REQUIRED, sinceVersion, validateType)
	if err != nil {
		return err
	}

	epsg, err := validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "EPSG", OPTIONAL, sinceVersion, func(i int) bool {
		return i > 0
	})
	if err != nil {
		return err
	}

	isASCII := func(s string) bool {
		for i := 0; i < len(s); i++ {
			if s[i] > 0x7f {
				return false
			}
		}
		return true
	}
	wkt, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "WKT", OPTIONAL, sinceVersion, isASCII)
	if err != nil {
		return err
	}
	if wkt != nil && len(*wkt) == 0 {
		err = errors.New("measure coordinate system dict: WKT must not be empty")
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	if epsg == nil && wkt == nil {
		err = errors.New("measure coordinate system dict: one of EPSG or WKT required")
		return model.WithValidationErrorObject(err, ownerObjNr)
	}
	if xRefTable.Version() == model.V20 && csType.Value() == "GEOGCS" && epsg != nil && wkt != nil {
		err = errors.New("measure coordinate system dict: EPSG and WKT are mutually exclusive")
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	return nil
}

func validateMeasureCoordinateSystemEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, entryName string, required bool, sinceVersion model.Version) error {
	coordinateSystemObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, "geospatialMeasureDict", entryName, required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("geospatialMeasureDict.%s: %w", entryName, err)
	}
	if d1 == nil {
		return nil
	}

	if err = validateMeasureCoordinateSystemDict(xRefTable, d1, coordinateSystemObjNr, sinceVersion); err != nil {
		return fmt.Errorf("geospatialMeasureDict.%s: %w", entryName, err)
	}

	return nil
}

func validateMeasureUnitSquareArray(xRefTable *model.XRefTable, a types.Array, ownerObjNr int, dictName, entryName string) error {
	min, max := 0.0, 1.0
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		const tolerance = 1e-5
		min, max = -tolerance, 1+tolerance
	}

	for i, o := range a {
		objNr := validationObjectNumber(ownerObjNr, o)
		f, err := xRefTable.DereferenceNumber(o)
		if err != nil {
			err = fmt.Errorf("dict=%s entry=%s index=%d: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		if f < min || f > max {
			err = fmt.Errorf("dict=%s entry=%s invalid value at index %d: %g", dictName, entryName, i, f)
			return model.WithValidationErrorObject(err, objNr)
		}
	}

	return nil
}

func validateMeasureDisplayUnits(xRefTable *model.XRefTable, a types.Array, ownerObjNr int) error {
	validUnits := [][]string{
		{"M", "KM", "FT", "USFT", "MI", "NM"},
		{"SQM", "HA", "SQKM", "SQFT", "A", "SQMI"},
		{"DEG", "GRD"},
	}

	for i, o := range a {
		objNr := validationObjectNumber(ownerObjNr, o)
		o, err := xRefTable.Dereference(o)
		if err != nil {
			err = fmt.Errorf("geospatialMeasureDict.PDU index=%d: %w", i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		name, ok := o.(types.Name)
		if !ok {
			err = fmt.Errorf("geospatialMeasureDict.PDU invalid type at index %d: %T", i, o)
			return model.WithValidationErrorObject(err, objNr)
		}
		if !types.MemberOf(name.Value(), validUnits[i]) {
			err = fmt.Errorf("geospatialMeasureDict.PDU invalid value at index %d: %s", i, o)
			return model.WithValidationErrorObject(err, objNr)
		}
	}

	return nil
}

func validateGeospatialMeasureDictPart1(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "geospatialMeasureDict"

	bounds, err := validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "Bounds", OPTIONAL, sinceVersion, func(a types.Array) bool {
		return len(a) >= 6 && len(a)%2 == 0
	})
	if err != nil {
		return err
	}
	if bounds != nil {
		if err = validateMeasureUnitSquareArray(xRefTable, bounds, validationEntryObjectNumber(ownerObjNr, d, "Bounds"), dictName, "Bounds"); err != nil {
			return err
		}
	}

	if err = validateMeasureCoordinateSystemEntry(xRefTable, d, ownerObjNr, "GCS", REQUIRED, sinceVersion); err != nil {
		return err
	}
	if err = validateMeasureCoordinateSystemEntry(xRefTable, d, ownerObjNr, "DCS", OPTIONAL, sinceVersion); err != nil {
		return err
	}

	pdu, err := validateNameArrayEntry(xRefTable, d, ownerObjNr, dictName, "PDU", OPTIONAL, sinceVersion, func(a types.Array) bool {
		return len(a) == 3
	})
	if err != nil {
		return err
	}
	if pdu != nil {
		if err = validateMeasureDisplayUnits(xRefTable, pdu, validationEntryObjectNumber(ownerObjNr, d, "PDU")); err != nil {
			return err
		}
	}

	return nil
}

func validateGeospatialMeasureDictPart2(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "geospatialMeasureDict"
	validatePairs := func(a types.Array) bool { return len(a) >= 4 && len(a)%2 == 0 }

	gpts, err := validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "GPTS", REQUIRED, sinceVersion, validatePairs)
	if err != nil {
		return err
	}
	lpts, err := validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "LPTS", OPTIONAL, sinceVersion, validatePairs)
	if err != nil {
		return err
	}
	if lpts != nil {
		if len(lpts) != len(gpts) {
			err = fmt.Errorf("%s: LPTS and GPTS array lengths differ", dictName)
			return model.WithValidationErrorObject(err, ownerObjNr)
		}
		if err = validateMeasureUnitSquareArray(xRefTable, lpts, validationEntryObjectNumber(ownerObjNr, d, "LPTS"), dictName, "LPTS"); err != nil {
			return err
		}
	}

	pcsmVersion := model.V20
	if xRefTable.ValidationMode == model.ValidationRelaxed && xRefTable.Version() == model.V17 {
		pcsmVersion = model.V17
	}
	pcsm, err := validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "PCSM", OPTIONAL, pcsmVersion, func(a types.Array) bool {
		return len(a) == 12
	})
	if err != nil {
		return err
	}
	if pcsm != nil && pcsmVersion < model.V20 {
		showDigestedVersionViolation(xRefTable, "dict="+dictName+" entry=PCSM")
	}

	return nil
}

func validateGeospatialMeasureDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	if err := validateGeospatialMeasureDictPart1(xRefTable, d, ownerObjNr, sinceVersion); err != nil {
		return err
	}

	return validateGeospatialMeasureDictPart2(xRefTable, d, ownerObjNr, sinceVersion)
}

func validateMeasureDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "measureDict"

	_, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool {
		return s == "Measure"
	})
	if err != nil {
		return err
	}

	subtype, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Subtype", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if subtype == nil || subtype.Value() == "RL" {
		return validateRectilinearMeasureDict(xRefTable, d, ownerObjNr, sinceVersion)
	}
	if subtype.Value() == "GEO" {
		return validateGeospatialMeasureDict(xRefTable, d, ownerObjNr, sinceVersion)
	}

	// Other coordinate-system subtypes are permitted but have unknown schemas.
	return nil
}

func validateViewportDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "viewportDict"

	_, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Viewport" })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, d, ownerObjNr, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Name", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Measure, optional, dict
	measureObjNr := validationEntryObjectNumber(ownerObjNr, d, "Measure")
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "Measure", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateMeasureDict(xRefTable, d1, measureObjNr, sinceVersion)
	}

	return err
}

func validatePageEntryVP(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	// see table 260
	viewportArrayObjNr := validationEntryObjectNumber(ownerObjNr, d, "VP")
	a, err := validateArrayEntry(xRefTable, d, ownerObjNr, "pagesDict", "VP", required, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}

	for _, v := range a {

		if v == nil {
			continue
		}

		viewportObjNr := validationObjectNumber(viewportArrayObjNr, v)
		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return model.WithValidationErrorObject(err, viewportObjNr)
		}

		if d == nil {
			continue
		}

		err = validateViewportDict(xRefTable, d, viewportObjNr, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func handlePieceInfo(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	sinceVersion := model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}

	hasPieceInfo, err := validatePieceInfo(xRefTable, d, 0, dictName, "PieceInfo", OPTIONAL, sinceVersion)
	if err != nil {
		return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "PieceInfo"))
	}

	// LastModified
	lm, err := validateDateEntry(xRefTable, d, ownerObjNr, dictName, "LastModified", OPTIONAL, model.V13)
	if err != nil {
		return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "LastModified"))
	}

	if hasPieceInfo && lm == nil && xRefTable.ValidationMode == model.ValidationStrict {
		err = errors.New("page dict: missing \"LastModified\" (required by \"PieceInfo\")")
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	return nil
}

func validatePageDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, hasMediaBox bool) (types.Array, error) {
	dictName := "pageDict"

	if ir := d.IndirectRefEntry("Parent"); ir == nil {
		err := errors.New("page dict: missing parent")
		return nil, model.WithValidationErrorObject(err, ownerObjNr)
	}

	// Contents
	_, err := validatePageContents(xRefTable, d, ownerObjNr)
	if err != nil {
		return nil, err
	}

	// Resources
	err = validatePageResources(xRefTable, d, ownerObjNr)
	if err != nil {
		return nil, err
	}

	// MediaBox
	mediaBoxArr, err := validatePageEntryMediaBox(xRefTable, d, ownerObjNr, !hasMediaBox, model.V10)
	if err != nil {
		return nil, err
	}

	// PieceInfo
	if err := handlePieceInfo(xRefTable, d, ownerObjNr, dictName); err != nil {
		return nil, err
	}

	// AA
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	err = validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, sinceVersion, "page")
	if err != nil {
		return nil, model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "AA"))
	}

	type v struct {
		validate            func(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) (err error)
		required            bool
		sinceVersion        model.Version
		sinceVersionRelaxed model.Version
	}

	for _, f := range []v{
		{validatePageEntryCropBox, OPTIONAL, model.V10, model.V10},
		{validatePageEntryBleedBox, OPTIONAL, model.V13, model.V12},
		{validatePageEntryTrimBox, OPTIONAL, model.V13, model.V10},
		{validatePageEntryArtBox, OPTIONAL, model.V13, model.V12},
		{validatePageBoxColorInfo, OPTIONAL, model.V14, model.V14},
		{validatePageEntryRotate, OPTIONAL, model.V10, model.V10},
		{validatePageEntryGroup, OPTIONAL, model.V14, model.V14},
		{validatePageEntryThumb, OPTIONAL, model.V10, model.V10},
		{validatePageEntryB, OPTIONAL, model.V11, model.V11},
		{validatePageEntryDur, OPTIONAL, model.V11, model.V11},
		{validatePageEntryTrans, OPTIONAL, model.V11, model.V11},
		{validatePageMetadata, OPTIONAL, model.V14, model.V14},
		{validatePageEntryStructParents, OPTIONAL, model.V10, model.V10},
		{validatePageEntryID, OPTIONAL, model.V13, model.V13},
		{validatePageEntryPZ, OPTIONAL, model.V13, model.V13},
		{validatePageEntrySeparationInfo, OPTIONAL, model.V13, model.V13},
		{validatePageEntryTabs, OPTIONAL, model.V15, model.V12},
		{validatePageEntryTemplateInstantiated, OPTIONAL, model.V15, model.V15},
		{validatePageEntryPresSteps, OPTIONAL, model.V15, model.V15},
		{validatePageEntryUserUnit, OPTIONAL, model.V16, model.V16},
		{validatePageEntryVP, OPTIONAL, model.V16, model.V14},
	} {
		sinceVersion := f.sinceVersion
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			sinceVersion = f.sinceVersionRelaxed
		}
		err = f.validate(xRefTable, d, ownerObjNr, f.required, sinceVersion)
		if err != nil {
			return nil, err
		}
	}

	return mediaBoxArr, nil
}

func validatePageMetadata(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, required bool, sinceVersion model.Version) error {
	err := validateMetadata(xRefTable, d, required, sinceVersion)
	return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "Metadata"))
}

func validatePagesDictGeneralEntries(xRefTable *model.XRefTable, d types.Dict, objNr int) (hasResources bool, mediaBoxArr types.Array, err error) {
	hasResources, err = validateResources(xRefTable, d)
	if err != nil {
		resourcesObjNr := validationEntryObjectNumber(objNr, d, "Resources")
		context := "page tree: node"
		if resourcesObjNr != objNr {
			context = fmt.Sprintf("page tree: node obj#%d", objNr)
		}
		err = fmt.Errorf("%s Resources: %w", context, err)
		return false, nil, model.WithValidationErrorObject(err, resourcesObjNr)
	}

	// MediaBox: optional, rectangle
	mediaBoxArr, err = validatePageEntryMediaBox(xRefTable, d, objNr, OPTIONAL, model.V10)
	if err != nil {
		return false, nil, pageTreeNodeEntryError(err, objNr, "MediaBox")
	}

	// CropBox: optional, rectangle
	err = validatePageEntryCropBox(xRefTable, d, objNr, OPTIONAL, model.V10)
	if err != nil {
		return false, nil, pageTreeNodeEntryError(err, objNr, "CropBox")
	}

	// Rotate:  optional, integer
	err = validatePageEntryRotate(xRefTable, d, objNr, OPTIONAL, model.V10)
	if err != nil {
		return false, nil, pageTreeNodeEntryError(err, objNr, "Rotate")
	}

	return hasResources, mediaBoxArr, nil
}

func dictTypeForPageNodeDict(xRefTable *model.XRefTable, d types.Dict, objNr int) (string, error) {
	if d == nil {
		err := errors.New("page tree: node is null")
		return "", model.WithValidationErrorObject(err, objNr)
	}

	dictType, _, err := xRefTable.DereferenceNameEntry(d, "Type")
	if err != nil {
		err = fmt.Errorf("page tree: node Type: %w", err)
		return "", model.WithValidationErrorObject(err, objNr)
	}
	if dictType == nil {
		err := errors.New("page tree: node missing Type")
		return "", model.WithValidationErrorObject(err, objNr)
	}

	return dictType.Value(), nil
}

func validateResources(xRefTable *model.XRefTable, d types.Dict) (hasResources bool, err error) {
	// Resources: optional, dict
	o, ok := d.Find("Resources")
	if !ok {
		return false, nil
	}

	return validateResourceDict(xRefTable, o)
}

func pagesDictKids(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (types.Array, error) {
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		return d.ArrayEntry("Kids"), nil
	}
	o, found := d.Find("Kids")
	if !found {
		return nil, nil
	}
	kids, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
	}
	return kids, nil
}

func validateParent(pageNodeDict types.Dict, childObjNr, parentObjNr int) error {
	parentIndRef := pageNodeDict.IndirectRefEntry("Parent")
	if parentIndRef == nil {
		return fmt.Errorf("page tree: node obj#%d: missing parent node, expected obj#%d", childObjNr, parentObjNr)
	}
	if parentIndRef.ObjectNumber.Value() != parentObjNr {
		return fmt.Errorf("page tree: node obj#%d: corrupt parent node, expected obj#%d, got obj#%d", childObjNr, parentObjNr, parentIndRef.ObjectNumber.Value())
	}
	return nil
}

func validatePageTreeParentLink(
	xRefTable *model.XRefTable,
	pageNodeDict types.Dict,
	childObjNr,
	parentObjNr int,
) (specViolation, err error) {
	err = validateParent(pageNodeDict, childObjNr, parentObjNr)
	if err == nil {
		return nil, nil
	}
	if xRefTable.ValidationMode == model.ValidationStrict || pageNodeDict.IndirectRefEntry("Parent") == nil {
		return nil, model.WithValidationErrorObject(err, childObjNr)
	}
	return model.WithValidationErrorObject(err, childObjNr), nil
}

func showDigestedPageTreeParentViolation(xRefTable *model.XRefTable, err error) {
	if err != nil {
		model.ShowDigestedSpecViolationError(err)
	}
}

func detectPageNodeDict(xRefTable *model.XRefTable, indRef types.IndirectRef, objNr, parentObjNr int, mediaBoxArr types.Array, pageNr int) (types.Dict, error) {
	pageNodeDict, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		err = fmt.Errorf("page tree: kid: dereference: %w", err)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if len(pageNodeDict) > 0 {
		return pageNodeDict, nil
	}

	if xRefTable.ValidationMode == model.ValidationStrict {
		err = fmt.Errorf("page tree: corrupt page %d", pageNr)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	var mediaBox *types.Rectangle
	if len(mediaBoxArr) > 0 {
		mediaBox, err = xRefTable.RectForArray(mediaBoxArr)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, parentObjNr)
		}
	}

	if _, err := xRefTable.EmptyPage(types.NewIndirectRef(parentObjNr, 0), mediaBox, objNr); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	model.ShowRepaired(fmt.Sprintf("corrupt page %d with blank page", pageNr))

	pageNodeDict, err = xRefTable.DereferenceDict(indRef)
	if err != nil {
		err = fmt.Errorf("page tree: repaired kid obj#%d: dereference: %w", objNr, err)
		return nil, model.WithValidationErrorObject(err, objNr)
	}
	return pageNodeDict, nil
}

func pageTreePageError(err error, objNr int) error {
	context := "page tree: page"
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) && validationErr.ObjectNumber() != objNr {
		context = fmt.Sprintf("page tree: page obj#%d", objNr)
	}
	err = fmt.Errorf("%s: %w", context, err)
	return model.WithValidationErrorObject(err, objNr)
}

func pageTreeKidError(err error, objNr int) error {
	context := "page tree: kid"
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) && validationErr.ObjectNumber() != objNr {
		context = fmt.Sprintf("page tree: kid obj#%d", objNr)
	}
	err = model.WrapRecursionError(context, err)
	return model.WithValidationErrorObject(err, objNr)
}

func pageTreeNodeEntryError(err error, objNr int, entryName string) error {
	context := "page tree: node"
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) && validationErr.ObjectNumber() != objNr {
		context = fmt.Sprintf("page tree: node obj#%d", objNr)
	}
	err = fmt.Errorf("%s %s: %w", context, entryName, err)
	return model.WithValidationErrorObject(err, objNr)
}

func processPagesKids(c context.Context, xRefTable *model.XRefTable, kids types.Array, parentObjNr int, hasResources bool, mediaBoxArr types.Array, curPage *int, depth int, visit *model.PageTreeVisit) (types.Array, error) {
	var a types.Array

	for i, o := range kids {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		ir, ok := o.(types.IndirectRef)
		if !ok {
			err := fmt.Errorf("page tree: parent obj#%d kid[%d]: expected indirect reference, got %T", parentObjNr, i, o)
			return nil, model.WithValidationErrorObject(err, parentObjNr)
		}

		objNr := ir.ObjectNumber.Value()
		if objNr == 0 {
			continue
		}

		pageNodeDict, err := detectPageNodeDict(xRefTable, ir, objNr, parentObjNr, mediaBoxArr, *curPage+1)
		if err != nil {
			return nil, err
		}

		a = append(a, ir)

		parentViolation, err := validatePageTreeParentLink(xRefTable, pageNodeDict, objNr, parentObjNr)
		if err != nil {
			return nil, err
		}

		dictType, err := dictTypeForPageNodeDict(xRefTable, pageNodeDict, objNr)
		if err != nil {
			return nil, err
		}

		switch dictType {

		case "Pages":
			if err = validatePagesDictDepth(c, xRefTable, pageNodeDict, objNr, hasResources, mediaBoxArr, curPage, depth+1, visit); err != nil {
				return nil, pageTreeKidError(err, objNr)
			}

		case "Page":
			*curPage++
			xRefTable.CurPage = *curPage
			dMediaBoxArr, err := validatePageDict(xRefTable, pageNodeDict, objNr, len(mediaBoxArr) > 0)
			if err != nil {
				return nil, pageTreePageError(err, objNr)
			}
			if len(mediaBoxArr) == 0 {
				mediaBoxArr = dMediaBoxArr
			}
			if err := xRefTable.SetValid(ir); err != nil {
				err = fmt.Errorf("page tree: page obj#%d: mark valid: %w", objNr, err)
				return nil, model.WithValidationErrorObject(err, objNr)
			}

		default:
			err := fmt.Errorf("page tree: node unexpected dict type: %s", dictType)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

		showDigestedPageTreeParentViolation(xRefTable, parentViolation)
	}

	return a, nil
}

func validatePagesDictDepth(c context.Context, xRefTable *model.XRefTable, d types.Dict, objNr int, hasResources bool, mediaBoxArr types.Array, curPage *int, depth int, visit *model.PageTreeVisit) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := xRefTable.CheckRecursionDepth("page tree", depth); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}
	if err := visit.Enter(objNr); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}
	defer visit.Leave(objNr)

	dHasResources, dMediaBoxArr, err := validatePagesDictGeneralEntries(xRefTable, d, objNr)
	if err != nil {
		return err
	}

	if dHasResources {
		hasResources = true
	}

	if len(dMediaBoxArr) > 0 {
		mediaBoxArr = dMediaBoxArr
	}

	kids, err := pagesDictKids(xRefTable, d, objNr)
	if err != nil {
		return fmt.Errorf("page tree: dereference \"Kids\" entry: %w", err)
	}
	if kids == nil {
		err = errors.New("page tree: corrupt \"Kids\" entry")
		return model.WithValidationErrorObject(err, objNr)
	}

	if len(kids) == 0 {
		return nil
	}

	d["Kids"], err = processPagesKids(c, xRefTable, kids, objNr, hasResources, mediaBoxArr, curPage, depth, visit)

	return err
}

func validatePagesDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, objNr int, hasResources bool, mediaBoxArr types.Array, curPage *int) error {
	return validatePagesDictDepth(
		c, xRefTable, d, objNr, hasResources, mediaBoxArr, curPage, 0, model.NewPageTreeVisit(),
	)
}

func repairPagesDict(c context.Context, xRefTable *model.XRefTable, obj types.Object, rootDict types.Dict, ownerObjNr int) (types.Dict, int, error) {
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		err = fmt.Errorf("page tree repair: dereference page root: %w", err)
		return nil, 0, model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, obj))
	}

	if d == nil {
		err = errors.New("page tree repair: cannot dereference page node")
		return nil, 0, model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, obj))
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		err = fmt.Errorf("page tree repair: create page root reference: %w", err)
		return nil, 0, model.WithValidationErrorObject(err, ownerObjNr)
	}

	rootDict["Pages"] = *indRef

	objNr := indRef.ObjectNumber.Value()

	// Patch kids.parents

	kids, err := pagesDictKids(xRefTable, d, objNr)
	if err != nil {
		return nil, 0, fmt.Errorf("page tree repair: dereference \"Kids\" entry: %w", err)
	}
	if kids == nil {
		return nil, 0, errors.New("page tree repair: corrupt \"Kids\" entry")
	}

	for i := range kids {
		if err := contextutil.Check(c); err != nil {
			return nil, 0, err
		}

		o := kids[i]

		if o == nil {
			continue
		}

		ir, ok := o.(types.IndirectRef)
		if !ok {
			return nil, 0, fmt.Errorf("page tree repair: kid[%d]: expected indirect reference, got %T", i, o)
		}

		if log.ValidateEnabled() {
			log.Validate.Printf("repairPagesDict: PageNode: %s\n", ir)
		}

		objNumber := ir.ObjectNumber.Value()
		if objNumber == 0 {
			continue
		}

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			err = fmt.Errorf("page tree repair: kid obj#%d: dereference: %w", objNumber, err)
			return nil, 0, model.WithValidationErrorObject(err, objNumber)
		}
		if d == nil {
			err = fmt.Errorf("page tree repair: corrupt page node obj#%d", objNumber)
			return nil, 0, model.WithValidationErrorObject(err, objNumber)
		}

		d["Parent"] = *indRef
	}

	return d, objNr, nil
}

func validateOrRepairPageTreeCount(xRefTable *model.XRefTable, pageRoot types.Dict, objNr, actualCount int) error {
	declaredCount := xRefTable.PageCount
	if actualCount == declaredCount {
		return nil
	}
	if xRefTable.ValidationMode == model.ValidationStrict {
		err := fmt.Errorf("page tree: counted %d pages, expected %d", actualCount, declaredCount)
		return model.WithValidationErrorObject(err, objNr)
	}
	pageRoot["Count"] = types.Integer(actualCount)
	xRefTable.PageCount = actualCount
	model.ShowRepaired(fmt.Sprintf("page tree root obj#%d Count from %d to %d", objNr, declaredCount, actualCount))
	return nil
}

func validatePages(c context.Context, xRefTable *model.XRefTable, rootDict types.Dict) (types.Dict, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	rootObjNr := validationRootObjectNumber(xRefTable)
	obj, found := rootDict.Find("Pages")
	if !found {
		err := errors.New("page tree root: missing \"Pages\"")
		return nil, model.WithValidationErrorObject(err, rootObjNr)
	}

	var (
		objNr    int
		pageRoot types.Dict
		err      error
	)

	ir, ok := obj.(types.IndirectRef)
	if !ok {
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			err = errors.New("page tree root: entry \"Pages\" must be an indirect reference")
			return nil, model.WithValidationErrorObject(err, rootObjNr)
		}
		pageRoot, objNr, err = repairPagesDict(c, xRefTable, obj, rootDict, rootObjNr)
		if err != nil {
			return nil, err
		}
		model.ShowRepaired("missing \"Pages\" indirect reference")
	}

	if ok {
		objNr = ir.ObjectNumber.Value()

		pageRoot, err = xRefTable.DereferenceDict(obj)
		if err != nil {
			err = fmt.Errorf("page tree root: dereference: %w", err)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

		if pageRoot == nil {
			err = errors.New("page tree root: cannot dereference page node")
			return nil, model.WithValidationErrorObject(err, objNr)
		}
	}

	obj, found = pageRoot.Find("Count")
	if !found {
		err = errors.New("page tree root: missing \"Count\"")
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	countObjNr := validationObjectNumber(objNr, obj)
	countContext := "page tree root"
	if countObjNr != objNr {
		countContext = fmt.Sprintf("page tree root: obj#%d", objNr)
	}
	i, err := xRefTable.DereferenceInteger(obj)
	if err != nil {
		err = fmt.Errorf("%s: corrupt \"Count\": %w", countContext, err)
		return nil, model.WithValidationErrorObject(err, countObjNr)
	}
	if i == nil {
		err = fmt.Errorf("%s: corrupt \"Count\"", countContext)
		return nil, model.WithValidationErrorObject(err, countObjNr)
	}

	xRefTable.PageCount = i.Value()

	pc := 0
	err = validatePagesDict(c, xRefTable, pageRoot, objNr, false, nil, &pc)
	if err != nil {
		return nil, err
	}

	if err = validateOrRepairPageTreeCount(xRefTable, pageRoot, objNr, pc); err != nil {
		return nil, err
	}

	return pageRoot, err
}

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
	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateResourceDict(xRefTable *pdf.XRefTable, o pdf.Object) (hasResources bool, err error) {

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return false, err
	}

	for k, v := range map[string]struct {
		validate     func(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error
		sinceVersion pdf.Version
	}{
		"ExtGState":  {validateExtGStateResourceDict, pdf.V10},
		"Font":       {validateFontResourceDict, pdf.V10},
		"XObject":    {validateXObjectResourceDict, pdf.V10},
		"Properties": {validatePropertiesResourceDict, pdf.V10},
		"ColorSpace": {validateColorSpaceResourceDict, pdf.V10},
		"Pattern":    {validatePatternResourceDict, pdf.V10},
		"Shading":    {validateShadingResourceDict, pdf.V13},
	} {
		if o, ok := d.Find(k); ok {
			err = v.validate(xRefTable, o, v.sinceVersion)
			if err != nil {
				return false, err
			}
		}
	}

	allowedResDictKeys := []string{"ExtGState", "Font", "XObject", "Properties", "ColorSpace", "Pattern", "ProcSet", "Shading"}
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		allowedResDictKeys = append(allowedResDictKeys, "Encoding")
		allowedResDictKeys = append(allowedResDictKeys, "ProcSets")
	}

	// Note: Beginning with PDF V1.4 the "ProcSet" feature is considered to be obsolete!

	for k := range d {
		if !pdf.MemberOf(k, allowedResDictKeys) {
			d.Delete(k)
		}
	}

	return true, nil
}

func validatePageContents(xRefTable *pdf.XRefTable, d pdf.Dict) (hasContents bool, err error) {

	o, found := d.Find("Contents")
	if !found {
		return false, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return false, err
	}

	switch o := o.(type) {

	case pdf.StreamDict:
		// no further processing.
		hasContents = true

	case pdf.Array:
		// process array of content stream dicts.

		for _, o := range o {
			o, _, err = xRefTable.DereferenceStreamDict(o)
			if err != nil {
				return false, err
			}

			if o == nil {
				continue
			}

			hasContents = true

		}

	default:
		return false, errors.Errorf("validatePageContents: page content must be stream dict or array")
	}

	return hasContents, nil
}

func validatePageResources(xRefTable *pdf.XRefTable, d pdf.Dict, hasResources, hasContents bool) error {

	if o, found := d.Find("Resources"); found {
		_, err := validateResourceDict(xRefTable, o)
		return err
	}

	// TODO Check if contents need resources (#169)
	// if !hasResources && hasContents {
	// 	return errors.New("pdfcpu: validatePageResources: missing required entry \"Resources\" - should be inherited")
	// }

	return nil
}

func validatePageEntryMediaBox(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) (hasMediaBox bool, err error) {

	o, err := validateRectangleEntry(xRefTable, d, "pageDict", "MediaBox", required, sinceVersion, nil)
	if err != nil {
		return false, err
	}
	if o != nil {
		hasMediaBox = true
	}

	return hasMediaBox, nil
}

func validatePageEntryCropBox(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateRectangleEntry(xRefTable, d, "pagesDict", "CropBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryBleedBox(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateRectangleEntry(xRefTable, d, "pagesDict", "BleedBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryTrimBox(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateRectangleEntry(xRefTable, d, "pagesDict", "TrimBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryArtBox(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateRectangleEntry(xRefTable, d, "pagesDict", "ArtBox", required, sinceVersion, nil)

	return err
}

func validateBoxStyleDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "boxStyleDict"

	// C, number array with 3 elements, optional
	_, err = validateNumberArrayEntry(xRefTable, d1, dictName, "C", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// W, number, optional
	_, err = validateNumberEntry(xRefTable, d1, dictName, "W", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// S, name, optional
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"S", "D"}) }
	_, err = validateNameEntry(xRefTable, d1, dictName, "S", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	// D, array, optional, since V1.3, dashArray
	_, err = validateIntegerArrayEntry(xRefTable, d1, dictName, "D", OPTIONAL, sinceVersion, nil)

	return err
}

func validatePageBoxColorInfo(xRefTable *pdf.XRefTable, pageDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// box color information dict
	// see 14.11.2.2

	dictName := "pageDict"

	d, err := validateDictEntry(xRefTable, pageDict, dictName, "BoxColorInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "boxColorInfoDict"

	err = validateBoxStyleDictEntry(xRefTable, d, dictName, "CropBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateBoxStyleDictEntry(xRefTable, d, dictName, "BleedBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	err = validateBoxStyleDictEntry(xRefTable, d, dictName, "TrimBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	return validateBoxStyleDictEntry(xRefTable, d, dictName, "ArtBox", OPTIONAL, sinceVersion)
}

func validatePageEntryRotate(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	validate := func(i int) bool { return i%90 == 0 }
	_, err := validateIntegerEntry(xRefTable, d, "pagesDict", "Rotate", required, sinceVersion, validate)

	return err
}

func validatePageEntryGroup(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}

	d1, err := validateDictEntry(xRefTable, d, "pageDict", "Group", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateGroupAttributesDict(xRefTable, d1)
	}

	return err
}

func validatePageEntryThumb(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	sd, err := validateStreamDictEntry(xRefTable, d, "pagesDict", "Thumb", required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	if err := validateXObjectStreamDict(xRefTable, *sd); err != nil {
		return err
	}

	indRef := d.IndirectRefEntry("Thumb")
	xRefTable.PageThumbs[xRefTable.CurPage] = *indRef
	//fmt.Printf("adding thumb page:%d obj#:%d\n", xRefTable.CurPage, indRef.ObjectNumber.Value())

	return nil
}

func validatePageEntryB(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// Note: Only makes sense if "Threads" entry in document root and bead dicts present.

	_, err := validateIndRefArrayEntry(xRefTable, d, "pagesDict", "B", required, sinceVersion, nil)

	return err
}

func validatePageEntryDur(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNumberEntry(xRefTable, d, "pagesDict", "Dur", required, sinceVersion, nil)

	return err
}

func validateTransitionDictEntryDi(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	o, found := d.Find("Di")
	if !found {
		return nil
	}

	switch o := o.(type) {

	case pdf.Integer:
		validate := func(i int) bool { return pdf.IntMemberOf(i, []int{0, 90, 180, 270, 315}) }
		if !validate(o.Value()) {
			return errors.New("pdfcpu: validateTransitionDict: entry Di int value undefined")
		}

	case pdf.Name:
		if o.Value() != "None" {
			return errors.New("pdfcpu: validateTransitionDict: entry Di name value undefined")
		}
	}

	return nil
}

func validateTransitionDictEntryM(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, transStyle *pdf.Name) error {

	// see 12.4.4
	validateTransitionDirectionOfMotion := func(s string) bool { return pdf.MemberOf(s, []string{"I", "O"}) }

	validateM := func(s string) bool {
		return validateTransitionDirectionOfMotion(s) &&
			(transStyle != nil && (*transStyle == "Split" || *transStyle == "Box" || *transStyle == "Fly"))
	}

	_, err := validateNameEntry(xRefTable, d, dictName, "M", OPTIONAL, pdf.V10, validateM)

	return err
}

func validateTransitionDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "transitionDict"

	// S, name, optional

	validateTransitionStyle := func(s string) bool {
		return pdf.MemberOf(s, []string{"Split", "Blinds", "Box", "Wipe", "Dissolve", "Glitter", "R"})
	}

	validate := validateTransitionStyle

	if xRefTable.Version() >= pdf.V15 {
		validate = func(s string) bool {

			if validateTransitionStyle(s) {
				return true
			}

			return pdf.MemberOf(s, []string{"Fly", "Push", "Cover", "Uncover", "Fade"})
		}
	}
	transStyle, err := validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, pdf.V10, validate)
	if err != nil {
		return err
	}

	// D, optional, number > 0
	_, err = validateNumberEntry(xRefTable, d, dictName, "D", OPTIONAL, pdf.V10, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// Dm, optional, name, see 12.4.4
	validateTransitionDimension := func(s string) bool { return pdf.MemberOf(s, []string{"H", "V"}) }

	validateDm := func(s string) bool {
		return validateTransitionDimension(s) && (transStyle != nil && (*transStyle == "Split" || *transStyle == "Blinds"))
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "Dm", OPTIONAL, pdf.V10, validateDm)
	if err != nil {
		return err
	}

	// M, optional, name
	err = validateTransitionDictEntryM(xRefTable, d, dictName, transStyle)
	if err != nil {
		return err
	}

	// Di, optional, number or name
	err = validateTransitionDictEntryDi(xRefTable, d)
	if err != nil {
		return err
	}

	// SS, optional, number, since V1.5
	if transStyle != nil && *transStyle == "Fly" {
		_, err = validateNumberEntry(xRefTable, d, dictName, "SS", OPTIONAL, pdf.V15, nil)
		if err != nil {
			return err
		}
	}

	// B, optional, boolean, since V1.5
	validateB := func(b bool) bool { return transStyle != nil && *transStyle == "Fly" }
	_, err = validateBooleanEntry(xRefTable, d, dictName, "B", OPTIONAL, pdf.V15, validateB)

	return err
}

func validatePageEntryTrans(xRefTable *pdf.XRefTable, pageDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	d, err := validateDictEntry(xRefTable, pageDict, "pagesDict", "Trans", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return validateTransitionDict(xRefTable, d)
}

func validatePageEntryStructParents(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateIntegerEntry(xRefTable, d, "pagesDict", "StructParents", required, sinceVersion, nil)

	return err
}

func validatePageEntryID(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateStringEntry(xRefTable, d, "pagesDict", "ID", required, sinceVersion, nil)

	return err
}

func validatePageEntryPZ(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// Preferred zoom factor, number

	_, err := validateNumberEntry(xRefTable, d, "pagesDict", "PZ", required, sinceVersion, nil)

	return err
}

func validatePageEntrySeparationInfo(xRefTable *pdf.XRefTable, pagesDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// see 14.11.4

	d, err := validateDictEntry(xRefTable, pagesDict, "pagesDict", "SeparationInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName := "separationDict"

	_, err = validateIndRefArrayEntry(xRefTable, d, dictName, "Pages", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	err = validateNameOrStringEntry(xRefTable, d, dictName, "DeviceColorant", required, sinceVersion)
	if err != nil {
		return err
	}

	a, err := validateArrayEntry(xRefTable, d, dictName, "ColorSpace", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if a != nil {
		err = validateColorSpaceArraySubset(xRefTable, a, []string{"Separation", "DeviceN"})
	}

	return err
}

func validatePageEntryTabs(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	validateTabs := func(s string) bool { return pdf.MemberOf(s, []string{"R", "C", "S", "A", "W"}) }

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	_, err := validateNameEntry(xRefTable, d, "pagesDict", "Tabs", required, sinceVersion, validateTabs)

	return err
}

func validatePageEntryTemplateInstantiated(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// see 12.7.6

	_, err := validateNameEntry(xRefTable, d, "pagesDict", "TemplateInstantiated", required, sinceVersion, nil)

	return err
}

// TODO implement
func validatePageEntryPresSteps(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// see 12.4.4.2

	d1, err := validateDictEntry(xRefTable, d, "pagesDict", "PresSteps", required, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	return errors.New("pdfcpu: validatePageEntryPresSteps: not supported")
}

func validatePageEntryUserUnit(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// UserUnit, optional, positive number, since V1.6
	_, err := validateNumberEntry(xRefTable, d, "pagesDict", "UserUnit", required, sinceVersion, func(f float64) bool { return f > 0 })

	return err
}

func validateNumberFormatDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "numberFormatDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "NumberFormat" })
	if err != nil {
		return err
	}

	// U, text string, required
	_, err = validateStringEntry(xRefTable, d, dictName, "U", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, number, required
	_, err = validateNumberEntry(xRefTable, d, dictName, "C", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, name, optional
	_, err = validateNameEntry(xRefTable, d, dictName, "F", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, integer, optional
	_, err = validateIntegerEntry(xRefTable, d, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// FD, bool, optional
	_, err = validateBooleanEntry(xRefTable, d, dictName, "FD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RT, text string, optional
	_, err = validateStringEntry(xRefTable, d, dictName, "RT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RD, text string, optional
	_, err = validateStringEntry(xRefTable, d, dictName, "RD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// PS, text string, optional
	_, err = validateStringEntry(xRefTable, d, dictName, "PS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SS, text string, optional
	_, err = validateStringEntry(xRefTable, d, dictName, "SS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// O, name, optional
	_, err = validateNameEntry(xRefTable, d, dictName, "O", OPTIONAL, sinceVersion, nil)

	return err
}

func validateNumberFormatArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}

	for _, v := range a {

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateNumberFormatDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateMeasureDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "measureDict"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Measure" })
	if err != nil {
		return err
	}

	// PDF 1.6 defines only a single type of coordinate system, a rectilinear coordinate system,
	// that shall be specified by the value RL for the Subtype entry.
	coordSys, err := validateNameEntry(xRefTable, d, dictName, "Subtype", OPTIONAL, sinceVersion, nil)
	if err != nil || coordSys == nil {
		return err
	}

	if *coordSys != "RL" {
		if xRefTable.Version() > sinceVersion {
			// unknown coord system
			return nil
		}
		return errors.Errorf("validateMeasureDict dict=%s entry=%s invalid dict entry: %s", dictName, "Subtype", coordSys.Value())
	}

	// R, text string, required, scale ratio
	_, err = validateStringEntry(xRefTable, d, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// X, number format array, required, for measurement of change along the x axis and, if Y is not present, along the y axis as well.
	err = validateNumberFormatArrayEntry(xRefTable, d, dictName, "X", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// Y, number format array, required when the x and y scales have different units or conversion factors.
	err = validateNumberFormatArrayEntry(xRefTable, d, dictName, "Y", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// D, number format array, required, for measurement of distance in any direction.
	err = validateNumberFormatArrayEntry(xRefTable, d, dictName, "D", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// A, number format array, required, for measurement of area.
	err = validateNumberFormatArrayEntry(xRefTable, d, dictName, "A", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// T, number format array, optional, for measurement of angles.
	err = validateNumberFormatArrayEntry(xRefTable, d, dictName, "T", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// S, number format array, optional, for fmeasurement of the slope of a line.
	err = validateNumberFormatArrayEntry(xRefTable, d, dictName, "S", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// O, number array, optional, array of two numbers that shall specify the origin of the measurement coordinate system in default user space coordinates.
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "O", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// CYX, number, optional, a factor that shall be used to convert the largest units along the y axis to the largest units along the x axis.
	_, err = validateNumberEntry(xRefTable, d, dictName, "CYX", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateViewportDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "viewportDict"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Viewport" })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, d, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, d, dictName, "Name", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Measure, optional, dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "Measure", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateMeasureDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validatePageEntryVP(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// see table 260

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V15
	}
	a, err := validateArrayEntry(xRefTable, d, "pagesDict", "VP", required, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}

	for _, v := range a {

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateViewportDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validatePageDict(xRefTable *pdf.XRefTable, d pdf.Dict, objNumber, genNumber int, hasResources, hasMediaBox bool) error {

	dictName := "pageDict"

	if ir := d.IndirectRefEntry("Parent"); ir == nil {
		return errors.New("pdfcpu: validatePageDict: missing parent")
	}

	// Contents
	hasContents, err := validatePageContents(xRefTable, d)
	if err != nil {
		return err
	}

	// Resources
	err = validatePageResources(xRefTable, d, hasResources, hasContents)
	if err != nil {
		return err
	}

	// MediaBox
	_, err = validatePageEntryMediaBox(xRefTable, d, !hasMediaBox, pdf.V10)
	if err != nil {
		return err
	}

	// PieceInfo
	sinceVersion := pdf.V13
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V10
	}
	hasPieceInfo, err := validatePieceInfo(xRefTable, d, dictName, "PieceInfo", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// LastModified
	lm, err := validateDateEntry(xRefTable, d, dictName, "LastModified", OPTIONAL, pdf.V13)
	if err != nil {
		return err
	}

	if hasPieceInfo && lm == nil && xRefTable.ValidationMode == pdf.ValidationStrict {
		return errors.New("pdfcpu: validatePageDict: missing \"LastModified\" (required by \"PieceInfo\")")
	}

	// AA
	err = validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, pdf.V14, "page")
	if err != nil {
		return err
	}

	type v struct {
		validate     func(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) (err error)
		required     bool
		sinceVersion pdf.Version
	}

	for _, f := range []v{
		{validatePageEntryCropBox, OPTIONAL, pdf.V10},
		{validatePageEntryBleedBox, OPTIONAL, pdf.V13},
		{validatePageEntryTrimBox, OPTIONAL, pdf.V13},
		{validatePageEntryArtBox, OPTIONAL, pdf.V13},
		{validatePageBoxColorInfo, OPTIONAL, pdf.V14},
		{validatePageEntryRotate, OPTIONAL, pdf.V10},
		{validatePageEntryGroup, OPTIONAL, pdf.V14},
		{validatePageEntryThumb, OPTIONAL, pdf.V10},
		{validatePageEntryB, OPTIONAL, pdf.V11},
		{validatePageEntryDur, OPTIONAL, pdf.V11},
		{validatePageEntryTrans, OPTIONAL, pdf.V11},
		{validateMetadata, OPTIONAL, pdf.V14},
		{validatePageEntryStructParents, OPTIONAL, pdf.V10},
		{validatePageEntryID, OPTIONAL, pdf.V13},
		{validatePageEntryPZ, OPTIONAL, pdf.V13},
		{validatePageEntrySeparationInfo, OPTIONAL, pdf.V13},
		{validatePageEntryTabs, OPTIONAL, pdf.V15},
		{validatePageEntryTemplateInstantiated, OPTIONAL, pdf.V15},
		{validatePageEntryPresSteps, OPTIONAL, pdf.V15},
		{validatePageEntryUserUnit, OPTIONAL, pdf.V16},
		{validatePageEntryVP, OPTIONAL, pdf.V16},
	} {
		err = f.validate(xRefTable, d, f.required, f.sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePagesDictGeneralEntries(xRefTable *pdf.XRefTable, d pdf.Dict) (hasResources, hasMediaBox bool, err error) {

	hasResources, err = validateResources(xRefTable, d)
	if err != nil {
		return false, false, err
	}

	// MediaBox: optional, rectangle
	hasMediaBox, err = validatePageEntryMediaBox(xRefTable, d, OPTIONAL, pdf.V10)
	if err != nil {
		return false, false, err
	}

	// CropBox: optional, rectangle
	err = validatePageEntryCropBox(xRefTable, d, OPTIONAL, pdf.V10)
	if err != nil {
		return false, false, err
	}

	// Rotate:  optional, integer
	err = validatePageEntryRotate(xRefTable, d, OPTIONAL, pdf.V10)
	if err != nil {
		return false, false, err
	}

	return hasResources, hasMediaBox, nil
}

func dictTypeForPageNodeDict(d pdf.Dict) (string, error) {

	if d == nil {
		return "", errors.New("pdfcpu: dictTypeForPageNodeDict: pageNodeDict is null")
	}

	dictType := d.Type()
	if dictType == nil {
		return "", errors.New("pdfcpu: dictTypeForPageNodeDict: missing pageNodeDict type")
	}

	return *dictType, nil
}

func validateResources(xRefTable *pdf.XRefTable, d pdf.Dict) (hasResources bool, err error) {

	// Get number of pages of this PDF file.
	pageCount := d.IntEntry("Count")
	if pageCount == nil {
		return false, errors.New("pdfcpu: validateResources: missing \"Count\"")
	}

	// TODO not ideal - overall pageCount is only set during validation!
	if xRefTable.PageCount == 0 {
		xRefTable.PageCount = *pageCount
	}

	log.Validate.Printf("validateResources: This page node has %d pages\n", *pageCount)

	// Resources: optional, dict
	o, ok := d.Find("Resources")
	if !ok {
		return false, nil
	}

	return validateResourceDict(xRefTable, o)
}

func validatePagesDict(xRefTable *pdf.XRefTable, d pdf.Dict, objNr, genNumber int, hasResources, hasMediaBox bool, curPage int) (int, error) {

	// Resources and Mediabox are inherited.
	dHasResources, dHasMediaBox, err := validatePagesDictGeneralEntries(xRefTable, d)
	if err != nil {
		return curPage, err
	}

	if dHasResources {
		hasResources = true
	}

	if dHasMediaBox {
		hasMediaBox = true
	}

	// Iterate over page tree.

	var kidsArray pdf.Array

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		o, found := d.Find("Kids")
		if !found {
			return curPage, errors.New("pdfcpu: validatePagesDict: corrupt \"Kids\" entry")
		}
		kidsArray, err = xRefTable.DereferenceArray(o)
		if err != nil {
			return curPage, errors.New("pdfcpu: validatePagesDict: corrupt \"Kids\" entry")
		}
	} else {
		kidsArray = d.ArrayEntry("Kids")
	}

	if kidsArray == nil {
		return curPage, errors.New("pdfcpu: validatePagesDict: corrupt \"Kids\" entry")
	}

	for _, o := range kidsArray {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(pdf.IndirectRef)
		if !ok {
			return curPage, errors.New("pdfcpu: validatePagesDict: missing indirect reference for kid")
		}

		log.Validate.Printf("validatePagesDict: PageNode: %s\n", ir)

		objNumber := ir.ObjectNumber.Value()
		genNumber := ir.GenerationNumber.Value()

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return curPage, err
		}

		// Validate this kid's parent.
		parentIndRef := pageNodeDict.IndirectRefEntry("Parent")
		if parentIndRef.ObjectNumber.Value() != objNr {
			return curPage, errors.New("pdfcpu: validatePagesDict: corrupt parent node")
		}

		dictType, err := dictTypeForPageNodeDict(pageNodeDict)
		if err != nil {
			return curPage, err
		}

		switch dictType {

		case "Pages":
			// Recurse over pagetree
			curPage, err = validatePagesDict(xRefTable, pageNodeDict, objNumber, genNumber, hasResources, hasMediaBox, curPage)

		case "Page":
			curPage++
			xRefTable.CurPage = curPage
			err = validatePageDict(xRefTable, pageNodeDict, objNumber, genNumber, hasResources, hasMediaBox)

		default:
			return curPage, errors.Errorf("pdfcpu: validatePagesDict: Unexpected dict type: %s", dictType)

		}

		if err != nil {
			return curPage, err
		}

	}

	return curPage, nil
}

func validatePages(xRefTable *pdf.XRefTable, rootDict pdf.Dict) (pdf.Dict, error) {

	// Ensure indirect reference entry "Pages".

	ir := rootDict.IndirectRefEntry("Pages")
	if ir == nil {
		return nil, errors.New("pdfcpu: validatePages: missing indirect obj for pages dict")
	}

	objNumber := ir.ObjectNumber.Value()
	genNumber := ir.GenerationNumber.Value()

	// Dereference root of page node tree.
	rootPageNodeDict, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return nil, err
	}

	if rootPageNodeDict == nil {
		return nil, errors.New("pdfcpu: validatePagesDict: cannot dereference pageNodeDict")
	}

	// Process page node tree.
	_, err = validatePagesDict(xRefTable, rootPageNodeDict, objNumber, genNumber, false, false, 0)
	if err != nil {
		return nil, err
	}

	return rootPageNodeDict, nil
}

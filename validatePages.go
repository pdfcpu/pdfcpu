package pdfcpu

import (
	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

func validateResourceDict(xRefTable *XRefTable, obj PDFObject) (hasResources bool, err error) {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return false, err
	}

	for k, v := range map[string]struct {
		validate     func(xRefTable *XRefTable, obj PDFObject, sinceVersion PDFVersion) error
		sinceVersion PDFVersion
	}{
		"ExtGState":  {validateExtGStateResourceDict, V10},
		"Font":       {validateFontResourceDict, V10},
		"XObject":    {validateXObjectResourceDict, V10},
		"Properties": {validatePropertiesResourceDict, V10},
		"ColorSpace": {validateColorSpaceResourceDict, V10},
		"Pattern":    {validatePatternResourceDict, V10},
		"Shading":    {validateShadingResourceDict, V13},
	} {
		if obj, ok := dict.Find(k); ok {
			err = v.validate(xRefTable, obj, v.sinceVersion)
			if err != nil {
				return false, err
			}
		}
	}

	// Beginning with PDF V1.4 this feature is considered to be obsolete.
	//_, err = validateNameArrayEntry(xRefTable, dict, "resourceDict", "ProcSet", OPTIONAL, V10, validateProcedureSetName)
	//if err != nil {
	//	return false, nil
	//}

	return true, nil
}

func validatePageContents(xRefTable *XRefTable, dict *PDFDict) (hasContents bool, err error) {

	obj, found := dict.Find("Contents")
	if !found {
		return false, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return false, err
	}

	switch obj := obj.(type) {

	case PDFStreamDict:
		// no further processing.
		hasContents = true

	case PDFArray:
		// process array of content stream dicts.

		for _, obj := range obj {

			obj, err = xRefTable.DereferenceStreamDict(obj)
			if err != nil {
				return false, err
			}

			if obj == nil {
				continue
			}

			hasContents = true

		}

	default:
		return false, errors.Errorf("validatePageContents: page content must be stream dict or array")
	}

	return hasContents, nil
}

func validatePageResources(xRefTable *XRefTable, dict *PDFDict, hasResources, hasContents bool) error {

	if obj, found := dict.Find("Resources"); found {
		_, err := validateResourceDict(xRefTable, obj)
		return err
	}

	if !hasResources && hasContents {
		return errors.New("validatePageResources: missing required entry \"Resources\" - should be inheritated")
	}

	return nil
}

func validatePageEntryMediaBox(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) (hasMediaBox bool, err error) {

	obj, err := validateRectangleEntry(xRefTable, dict, "pageDict", "MediaBox", required, sinceVersion, nil)
	if err != nil {
		return false, err
	}
	if obj != nil {
		hasMediaBox = true
	}

	return hasMediaBox, nil
}

func validatePageEntryCropBox(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateRectangleEntry(xRefTable, dict, "pagesDict", "CropBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryBleedBox(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateRectangleEntry(xRefTable, dict, "pagesDict", "BleedBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryTrimBox(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateRectangleEntry(xRefTable, dict, "pagesDict", "TrimBox", required, sinceVersion, nil)

	return err
}

func validatePageEntryArtBox(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateRectangleEntry(xRefTable, dict, "pagesDict", "ArtBox", required, sinceVersion, nil)

	return err
}

func validateBoxStyleDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName string, entryName string, required bool, sinceVersion PDFVersion) (*PDFDict, error) {

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return nil, err
	}

	dictName = "boxStyleDict"

	// C, number array with 3 elements, optional
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, sinceVersion, func(arr PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return nil, err
	}

	// W, number, optional
	_, err = validateNumberEntry(xRefTable, d, dictName, "W", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return nil, err
	}

	// S, name, optional
	validate := func(s string) bool { return memberOf(s, []string{"S", "D"}) }
	_, err = validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return nil, err
	}

	// D, array, optional, since V1.3, dashArray
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "D", OPTIONAL, sinceVersion, nil)

	return d, err
}

func validatePageBoxColorInfo(xRefTable *XRefTable, pageDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// box color information dict
	// see 14.11.2.2

	dictName := "pageDict"

	dict, err := validateDictEntry(xRefTable, pageDict, dictName, "BoxColorInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName = "boxColorInfoDict"

	_, err = validateBoxStyleDictEntry(xRefTable, dict, dictName, "CropBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, dictName, "BleedBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, dictName, "TrimBox", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, dictName, "ArtBox", OPTIONAL, sinceVersion)

	return err
}

func validatePageEntryRotate(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	validate := func(i int) bool { return intMemberOf(i, []int{0, 90, 180, 270}) }
	_, err := validateIntegerEntry(xRefTable, dict, "pagesDict", "Rotate", required, sinceVersion, validate)

	return err
}

func validatePageEntryGroup(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	d, err := validateDictEntry(xRefTable, dict, "pageDict", "Group", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateGroupAttributesDict(xRefTable, *d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePageEntryThumb(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	sd, err := validateStreamDictEntry(xRefTable, dict, "pagesDict", "Thumb", required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	return validateXObjectStreamDict(xRefTable, *sd)
}

func validatePageEntryB(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// Note: Only makes sense if "Threads" entry in document root and bead dicts present.

	_, err := validateIndRefArrayEntry(xRefTable, dict, "pagesDict", "B", required, sinceVersion, nil)

	return err
}

func validatePageEntryDur(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateNumberEntry(xRefTable, dict, "pagesDict", "Dur", required, sinceVersion, nil)

	return err
}

func validateTransitionDictEntryDi(xRefTable *XRefTable, dict *PDFDict) error {

	obj, found := dict.Find("Di")
	if !found {
		return nil
	}

	switch obj := obj.(type) {

	case PDFInteger:
		validate := func(i int) bool { return intMemberOf(i, []int{0, 90, 180, 270, 315}) }
		if !validate(obj.Value()) {
			return errors.New("validateTransitionDict: entry Di int value undefined")
		}

	case PDFName:
		if obj.Value() != "None" {
			return errors.New("validateTransitionDict: entry Di name value undefined")
		}
	}

	return nil
}

func validateTransitionDictEntryM(xRefTable *XRefTable, dict *PDFDict, dictName string, transStyle *PDFName) error {

	// see 12.4.4
	validateTransitionDirectionOfMotion := func(s string) bool { return memberOf(s, []string{"I", "O"}) }

	validateM := func(s string) bool {
		return validateTransitionDirectionOfMotion(s) &&
			(transStyle != nil && (*transStyle == "Split" || *transStyle == "Box" || *transStyle == "Fly"))
	}

	_, err := validateNameEntry(xRefTable, dict, dictName, "M", OPTIONAL, V10, validateM)

	return err
}

func validateTransitionDict(xRefTable *XRefTable, dict *PDFDict) error {

	dictName := "transitionDict"

	// S, name, optional

	validateTransitionStyle := func(s string) bool {
		return memberOf(s, []string{"Split", "Blinds", "Box", "Wipe", "Dissolve", "Glitter", "R"})
	}

	validate := validateTransitionStyle

	if xRefTable.Version() >= V15 {
		validate = func(s string) bool {

			if validateTransitionStyle(s) {
				return true
			}

			return memberOf(s, []string{"Fly", "Push", "Cover", "Uncover", "Fade"})
		}
	}
	transStyle, err := validateNameEntry(xRefTable, dict, dictName, "S", OPTIONAL, V10, validate)
	if err != nil {
		return err
	}

	// D, optional, number > 0
	_, err = validateNumberEntry(xRefTable, dict, dictName, "D", OPTIONAL, V10, func(f float64) bool { return f > 0 })
	if err != nil {
		return err
	}

	// Dm, optional, name, see 12.4.4
	validateTransitionDimension := func(s string) bool { return memberOf(s, []string{"H", "V"}) }

	validateDm := func(s string) bool {
		return validateTransitionDimension(s) && (transStyle != nil && (*transStyle == "Split" || *transStyle == "Blinds"))
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Dm", OPTIONAL, V10, validateDm)
	if err != nil {
		return err
	}

	// M, optional, name
	err = validateTransitionDictEntryM(xRefTable, dict, dictName, transStyle)
	if err != nil {
		return err
	}

	// Di, optional, number or name
	err = validateTransitionDictEntryDi(xRefTable, dict)
	if err != nil {
		return err
	}

	// SS, optional, number, since V1.5
	if transStyle != nil && *transStyle == "Fly" {
		_, err = validateNumberEntry(xRefTable, dict, dictName, "SS", OPTIONAL, V15, nil)
		if err != nil {
			return err
		}
	}

	// B, optional, boolean, since V1.5
	validateB := func(b bool) bool { return transStyle != nil && *transStyle == "Fly" }
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "B", OPTIONAL, V15, validateB)

	return err
}

func validatePageEntryTrans(xRefTable *XRefTable, pageDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	dict, err := validateDictEntry(xRefTable, pageDict, "pagesDict", "Trans", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return validateTransitionDict(xRefTable, dict)
}

func validatePageEntryStructParents(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateIntegerEntry(xRefTable, dict, "pagesDict", "StructParents", required, sinceVersion, nil)

	return err
}

func validatePageEntryID(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	_, err := validateStringEntry(xRefTable, dict, "pagesDict", "ID", required, sinceVersion, nil)

	return err
}

func validatePageEntryPZ(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// Preferred zoom factor, number

	_, err := validateNumberEntry(xRefTable, dict, "pagesDict", "PZ", required, sinceVersion, nil)

	return err
}

func validatePageEntrySeparationInfo(xRefTable *XRefTable, pagesDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// see 14.11.4

	dict, err := validateDictEntry(xRefTable, pagesDict, "pagesDict", "SeparationInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName := "separationDict"

	_, err = validateIndRefArrayEntry(xRefTable, dict, dictName, "Pages", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	err = validateNameOrStringEntry(xRefTable, dict, dictName, "DeviceColorant", required, sinceVersion)
	if err != nil {
		return err
	}

	arr, err := validateArrayEntry(xRefTable, dict, dictName, "ColorSpace", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if arr != nil {
		err = validateColorSpaceArraySubset(xRefTable, arr, []string{"Separation", "DeviceN"})
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePageEntryTabs(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// Include out of spec entry "W"
	validateTabs := func(s string) bool { return memberOf(s, []string{"R", "C", "S", "W"}) }

	_, err := validateNameEntry(xRefTable, dict, "pagesDict", "Tabs", required, sinceVersion, validateTabs)

	return err
}

func validatePageEntryTemplateInstantiated(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// see 12.7.6

	_, err := validateNameEntry(xRefTable, dict, "pagesDict", "TemplateInstantiated", required, sinceVersion, nil)

	return err
}

// TODO implement
func validatePageEntryPresSteps(xRefTable *XRefTable, pagesDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// see 12.4.4.2

	dict, err := validateDictEntry(xRefTable, pagesDict, "pagesDict", "PresSteps", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return errors.New("*** validatePageEntryPresSteps: not supported ***")
}

func validatePageEntryUserUnit(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// UserUnit, optional, positive number, since V1.6
	_, err := validateNumberEntry(xRefTable, dict, "pagesDict", "UserUnit", required, sinceVersion, func(f float64) bool { return f > 0 })

	return err
}

func validateNumberFormatDict(xRefTable *XRefTable, dict *PDFDict, sinceVersion PDFVersion) error {

	dictName := "numberFormatDict"

	// Type, name, optional
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "NumberFormat" })
	if err != nil {
		return err
	}

	// U, text string, required
	_, err = validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, number, required
	_, err = validateNumberEntry(xRefTable, dict, dictName, "C", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, name, optional
	_, err = validateNameEntry(xRefTable, dict, dictName, "F", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, integer, optional
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// FD, bool, optional
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "FD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RT, text string, optional
	_, err = validateStringEntry(xRefTable, dict, dictName, "RT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RD, text string, optional
	_, err = validateStringEntry(xRefTable, dict, dictName, "RD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// PS, text string, optional
	_, err = validateStringEntry(xRefTable, dict, dictName, "PS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// SS, text string, optional
	_, err = validateStringEntry(xRefTable, dict, dictName, "SS", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// O, name, optional
	_, err = validateNameEntry(xRefTable, dict, dictName, "O", OPTIONAL, sinceVersion, nil)

	return err
}

func validateNumberFormatArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	for _, v := range *arr {

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

func validateMeasureDict(xRefTable *XRefTable, dict *PDFDict, sinceVersion PDFVersion) error {

	dictName := "measureDict"

	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Measure" })
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool { return s == "RL" })
	if err != nil {
		return err
	}

	// R, text string, required, scale ratio
	_, err = validateStringEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// X, number format array, required, for measurement of change along the x axis and, if Y is not present, along the y axis as well.
	err = validateNumberFormatArrayEntry(xRefTable, dict, dictName, "X", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// Y, number format array, required when the x and y scales have different units or conversion factors.
	err = validateNumberFormatArrayEntry(xRefTable, dict, dictName, "Y", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// D, number format array, required, for measurement of distance in any direction.
	err = validateNumberFormatArrayEntry(xRefTable, dict, dictName, "D", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// A, number format array, required, for measurement of area.
	err = validateNumberFormatArrayEntry(xRefTable, dict, dictName, "A", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// T, number format array, optional, for measurement of angles.
	err = validateNumberFormatArrayEntry(xRefTable, dict, dictName, "T", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// S, number format array, optional, for fmeasurement of the slope of a line.
	err = validateNumberFormatArrayEntry(xRefTable, dict, dictName, "S", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// O, number array, optional, array of two numbers that shall specify the origin of the measurement coordinate system in default user space coordinates.
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "O", OPTIONAL, sinceVersion, func(a PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// CYX, number, optional, a factor that shall be used to convert the largest units along the y axis to the largest units along the x axis.
	_, err = validateNumberEntry(xRefTable, dict, dictName, "CYX", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateViewportDict(xRefTable *XRefTable, dict *PDFDict, sinceVersion PDFVersion) error {

	dictName := "viewportDict"

	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Viewport" })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "Name", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Measure, optional, dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "Measure", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateMeasureDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePageEntryVP(xRefTable *XRefTable, pagesDict *PDFDict, required bool, sinceVersion PDFVersion) error {

	// see table 260

	arr, err := validateArrayEntry(xRefTable, pagesDict, "pagesDict", "VP", required, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	var dict *PDFDict

	for _, v := range *arr {

		if v == nil {
			continue
		}

		dict, err = xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		err = validateViewportDict(xRefTable, dict, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validatePageDict(xRefTable *XRefTable, pageDict *PDFDict, objNumber, genNumber int, hasResources, hasMediaBox bool) error {

	dictName := "pageDict"

	if indref := pageDict.IndirectRefEntry("Parent"); indref == nil {
		return errors.New("validatePageDict: missing parent")
	}

	// Contents
	hasContents, err := validatePageContents(xRefTable, pageDict)
	if err != nil {
		return err
	}

	// Resources
	err = validatePageResources(xRefTable, pageDict, hasResources, hasContents)
	if err != nil {
		return err
	}

	// MediaBox
	_, err = validatePageEntryMediaBox(xRefTable, pageDict, !hasMediaBox, V10)
	if err != nil {
		return err
	}

	// PieceInfo
	sinceVersion := V13
	if xRefTable.ValidationMode == ValidationRelaxed {
		sinceVersion = V10
	}
	hasPieceInfo, err := validatePieceInfo(xRefTable, pageDict, dictName, "PieceInfo", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// LastModified
	lm, err := validateDateEntry(xRefTable, pageDict, dictName, "LastModified", OPTIONAL, V13)
	if err != nil {
		return err
	}

	if hasPieceInfo && lm == nil && xRefTable.ValidationMode == ValidationStrict {
		return errors.New("validatePageDict: missing \"LastModified\" (required by \"PieceInfo\")")
	}

	// AA
	err = validateAdditionalActions(xRefTable, pageDict, dictName, "AA", OPTIONAL, V14, "page")
	if err != nil {
		return err
	}

	type v struct {
		validate     func(xRefTable *XRefTable, dict *PDFDict, required bool, sinceVersion PDFVersion) (err error)
		required     bool
		sinceVersion PDFVersion
	}

	for _, f := range []v{
		{validatePageEntryCropBox, OPTIONAL, V10},
		{validatePageEntryBleedBox, OPTIONAL, V13},
		{validatePageEntryTrimBox, OPTIONAL, V13},
		{validatePageEntryArtBox, OPTIONAL, V13},
		{validatePageBoxColorInfo, OPTIONAL, V14},
		{validatePageEntryRotate, OPTIONAL, V10},
		{validatePageEntryGroup, OPTIONAL, V14},
		{validatePageEntryThumb, OPTIONAL, V10},
		{validatePageEntryB, OPTIONAL, V11},
		{validatePageEntryDur, OPTIONAL, V11},
		{validatePageEntryTrans, OPTIONAL, V11},
		{validateMetadata, OPTIONAL, V14},
		{validatePageEntryStructParents, OPTIONAL, V10},
		{validatePageEntryID, OPTIONAL, V13},
		{validatePageEntryPZ, OPTIONAL, V13},
		{validatePageEntrySeparationInfo, OPTIONAL, V13},
		{validatePageEntryTabs, OPTIONAL, V15},
		{validatePageEntryTemplateInstantiated, OPTIONAL, V15},
		{validatePageEntryPresSteps, OPTIONAL, V15},
		{validatePageEntryUserUnit, OPTIONAL, V16},
		{validatePageEntryVP, OPTIONAL, V16},
	} {
		err = f.validate(xRefTable, pageDict, f.required, f.sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePagesDictGeneralEntries(xRefTable *XRefTable, dict *PDFDict) (hasResources, hasMediaBox bool, err error) {

	hasResources, err = validateResources(xRefTable, dict)
	if err != nil {
		return false, false, err
	}

	// MediaBox: optional, rectangle
	hasMediaBox, err = validatePageEntryMediaBox(xRefTable, dict, OPTIONAL, V10)
	if err != nil {
		return false, false, err
	}

	// CropBox: optional, rectangle
	err = validatePageEntryCropBox(xRefTable, dict, OPTIONAL, V10)
	if err != nil {
		return false, false, err
	}

	// Rotate:  optional, integer
	err = validatePageEntryRotate(xRefTable, dict, OPTIONAL, V10)
	if err != nil {
		return false, false, err
	}

	return hasResources, hasMediaBox, nil
}

func dictTypeForPageNodeDict(pageNodeDict *PDFDict) (string, error) {

	if pageNodeDict == nil {
		return "", errors.New("dictTypeForPageNodeDict: pageNodeDict is null")
	}

	dictType := pageNodeDict.Type()
	if dictType == nil {
		return "", errors.New("dictTypeForPageNodeDict: missing pageNodeDict type")
	}

	return *dictType, nil
}

func validateResources(xRefTable *XRefTable, dict *PDFDict) (hasResources bool, err error) {

	// Get number of pages of this PDF file.
	pageCount := dict.IntEntry("Count")
	if pageCount == nil {
		return false, errors.New("validateResources: missing \"Count\"")
	}

	// TODO not ideal - overall pageCount is only set during validation!
	if xRefTable.PageCount == 0 {
		xRefTable.PageCount = *pageCount
	}

	log.Debug.Printf("validateResources: This page node has %d pages\n", *pageCount)

	// Resources: optional, dict
	obj, ok := dict.Find("Resources")
	if !ok {
		return false, nil
	}

	return validateResourceDict(xRefTable, obj)
}

func validatePagesDict(xRefTable *XRefTable, dict *PDFDict, objNumber, genNumber int, hasResources, hasMediaBox bool) error {

	// Resources and Mediabox are inheritated.
	//var dHasResources, dHasMediaBox bool
	dHasResources, dHasMediaBox, err := validatePagesDictGeneralEntries(xRefTable, dict)
	if err != nil {
		return err
	}

	if dHasResources {
		hasResources = true
	}

	if dHasMediaBox {
		hasMediaBox = true
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("validatePagesDict: corrupt \"Kids\" entry")
	}

	for _, obj := range *kidsArray {

		if obj == nil {
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(PDFIndirectRef)
		if !ok {
			return errors.New("validatePagesDict: missing indirect reference for kid")
		}

		log.Debug.Printf("validatePagesDict: PageNode: %s\n", indRef)

		objNumber := indRef.ObjectNumber.Value()
		genNumber := indRef.GenerationNumber.Value()

		var pageNodeDict *PDFDict
		pageNodeDict, err = xRefTable.DereferenceDict(indRef)
		if err != nil {
			return err
		}

		dictType, err := dictTypeForPageNodeDict(pageNodeDict)
		if err != nil {
			return err
		}

		switch dictType {

		case "Pages":
			// Recurse over pagetree
			err = validatePagesDict(xRefTable, pageNodeDict, objNumber, genNumber, hasResources, hasMediaBox)
			if err != nil {
				return err
			}

		case "Page":
			err = validatePageDict(xRefTable, pageNodeDict, objNumber, genNumber, hasResources, hasMediaBox)
			if err != nil {
				return err
			}

		default:
			return errors.Errorf("validatePagesDict: Unexpected dict type: %s", dictType)

		}

	}

	return nil
}

func validatePages(xRefTable *XRefTable, rootDict *PDFDict) (*PDFDict, error) {

	// Ensure indirect reference entry "Pages".

	indRef := rootDict.IndirectRefEntry("Pages")
	if indRef == nil {
		return nil, errors.New("validatePages: missing indirect obj for pages dict")
	}

	objNumber := indRef.ObjectNumber.Value()
	genNumber := indRef.GenerationNumber.Value()

	// Dereference root of page node tree.
	rootPageNodeDict, err := xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return nil, err
	}

	if rootPageNodeDict == nil {
		return nil, errors.New("validatePagesDict: cannot dereference pageNodeDict")
	}

	// Process page node tree.
	err = validatePagesDict(xRefTable, rootPageNodeDict, objNumber, genNumber, false, false)
	if err != nil {
		return nil, err
	}

	return rootPageNodeDict, nil
}

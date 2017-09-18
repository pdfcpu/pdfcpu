package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateResourceDict(xRefTable *types.XRefTable, obj interface{}) (hasResources bool, err error) {

	logInfoValidate.Println("*** validateResourceDict begin: ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Printf("validateResourceDict end: object  is nil.\n")
		return
	}

	if obj, ok := dict.Find("ExtGState"); ok {
		err = validateExtGStateResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Font"); ok {
		err = validateFontResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("XObject"); ok {
		err = validateXObjectResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Properties"); ok {
		err = validatePropertiesResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("ColorSpace"); ok {
		err = validateColorSpaceResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Pattern"); ok {
		err = validatePatternResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Shading"); ok {
		err = validateShadingResourceDict(xRefTable, obj, types.V13)
		if err != nil {
			return
		}
	}

	// Beginning with PDF V1.4 this feature is considered to be obsolete.
	//_, _, err = validateNameArrayEntry(xRefTable, dict, "resourceDict", "ProcSet", OPTIONAL, types.V10, validateProcedureSetName)
	if err != nil {
		return
	}

	hasResources = true

	logInfoValidate.Println("*** validateResourceDict end ***")

	return
}

func validatePageContents(xRefTable *types.XRefTable, dict *types.PDFDict) (hasContents bool, err error) {

	logInfoValidate.Println("*** validatePageContents begin ***")

	obj, found := dict.Find("Contents")
	if !found {
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		// no further processing.
		hasContents = true

	case types.PDFArray:
		// process array of content stream dicts.

		logInfoValidate.Printf("validatePageContents: writing content array\n")

		for _, obj := range obj {

			obj, err = xRefTable.DereferenceStreamDict(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			hasContents = true

		}

	default:
		err = errors.Errorf("validatePageContents: page content must be stream dict or array")
		return
	}

	logInfoValidate.Println("*** validatePageContents end ***")

	return
}

func validatePageResources(xRefTable *types.XRefTable, dict *types.PDFDict, hasResources, hasContents bool) (err error) {

	logInfoValidate.Println("*** validatePageResources begin ***")

	if obj, found := dict.Find("Resources"); found {

		_, err := validateResourceDict(xRefTable, obj)
		if err != nil {
			return err
		}

		// Leave here where called from writePageDict.
		return nil
	}

	if !hasResources && hasContents {
		err = errors.New("validatePageResources: missing required entry \"Resources\" - should be inheritated")
	}

	logInfoValidate.Println("*** validatePageResources end ***")

	return
}

func validatePageEntryMediaBox(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (hasMediaBox bool, err error) {

	logInfoValidate.Println("*** validatePageEntryMediaBox begin ***")

	obj, err := validateRectangleEntry(xRefTable, dict, "pagesDict", "MediaBox", required, sinceVersion, nil)
	if err != nil {
		return
	}
	if obj != nil {
		hasMediaBox = true
	}

	logInfoValidate.Printf("*** validatePageEntryMediaBox end: hasMediaBox=%v ***\n", hasMediaBox)

	return
}

func validatePageEntryCropBox(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryCropBox begin ***")

	_, err = validateRectangleEntry(xRefTable, dict, "pagesDict", "CropBox", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryCropBox end ***")

	return
}

func validatePageEntryBleedBox(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryBleedBox begin ***")

	_, err = validateRectangleEntry(xRefTable, dict, "pagesDict", "BleedBox", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryBleedBox end ***")

	return
}

func validatePageEntryTrimBox(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryTrimBox begin ***")

	_, err = validateRectangleEntry(xRefTable, dict, "pagesDict", "TrimBox", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryTrimBox end ***")

	return
}

func validatePageEntryArtBox(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryArtBox begin ***")

	_, err = validateRectangleEntry(xRefTable, dict, "pagesDict", "ArtBox", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryArtBox end ***")

	return
}

func validateBoxStyleDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (d *types.PDFDict, err error) {

	logInfoValidate.Println("*** writeBoxStyleDictEntry begin ***")

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if d == nil {
		logInfoValidate.Printf("writeBoxStyleDictEntry end: is nil.\n")
		return
	}

	dictName = "boxStyleDict"

	// C, number array with 3 elements, optional
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	// W, number, optional
	_, err = validateNumberEntry(xRefTable, d, dictName, "W", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// S, name, optional
	_, err = validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, sinceVersion, validateGuideLineStyle)
	if err != nil {
		return
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = validateLineDashPatternEntry(xRefTable, d, dictName, "D", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** writeBoxStyleDictEntry end ***")

	return
}

func validatePageBoxColorInfo(xRefTable *types.XRefTable, pageDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// box color information dict
	// see 14.11.2.2

	logInfoValidate.Println("*** validatePageBoxColorInfo begin ***")

	var dict *types.PDFDict

	dict, err = validateDictEntry(xRefTable, pageDict, "pageDict", "BoxColorInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, "boxColorInfoDict", "CropBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, "boxColorInfoDict", "BleedBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, "boxColorInfoDict", "TrimBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	_, err = validateBoxStyleDictEntry(xRefTable, dict, "boxColorInfoDict", "ArtBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageBoxColorInfo end ***")

	return
}

func validatePageEntryRotate(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** writePageEntryRotate begin ***")

	_, err = validateIntegerEntry(xRefTable, dict, "pagesDict", "Rotate", required, sinceVersion, validateRotate)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** writePageEntryRotate end ***")

	return
}

func validatePageEntryGroup(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryGroup begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, "pageDict", "Group", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateGroupAttributesDict(xRefTable, *d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validatePageEntryGroup end ***")

	return
}

func validatePageEntryThumb(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryThumb begin ***")

	streamDict, err := validateStreamDictEntry(xRefTable, dict, "pagesDict", "Thumb", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if streamDict == nil {
		logInfoValidate.Println("validatePageEntryThumb end: is nil.")
		return
	}

	err = validateXObjectStreamDict(xRefTable, streamDict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryThumb end ***")

	return
}

func validatePageEntryB(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// Note: Only makes sense if "Threads" entry in document root and bead dicts present.

	logInfoValidate.Println("*** validatePageEntryB begin ***")

	arr, err := validateIndRefArrayEntry(xRefTable, dict, "pagesDict", "B", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validatePageEntryB end: is nil.")
		return
	}

	logInfoValidate.Println("*** validatePageEntryB end ***")

	return
}

func validatePageEntryDur(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validatePageEntryDur begin ***")

	_, err = validateNumberEntry(xRefTable, dict, "pagesDict", "Dur", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryDur end ***")

	return
}

func validateTransitionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateTransitionDict begin ***")

	dictName := "transitionDict"

	// S, name, optional
	validate := validateTransitionStyle
	if xRefTable.Version() >= types.V15 {
		validate = validateTransitionStyleV15
	}
	transStyle, err := validateNameEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, validate)
	if err != nil {
		return
	}

	// D, optional, number > 0
	_, err = validateNumberEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V10, func(f float64) bool { return f > 0 })
	if err != nil {
		return
	}

	// Dm, optional, name
	validateDm := func(s string) bool {
		return validateTransitionDimension(s) && (transStyle != nil && (*transStyle == "Split" || *transStyle == "Blinds"))
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Dm", OPTIONAL, types.V10, validateDm)
	if err != nil {
		return
	}

	// M, optional, name
	validateM := func(s string) bool {
		return validateTransitionDirectionOfMotion(s) &&
			(transStyle != nil && (*transStyle == "Split" || *transStyle == "Box" || *transStyle == "Fly"))
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, validateM)
	if err != nil {
		return
	}

	// Di, optional, number or name
	obj, found := dict.Find("Di")
	if found {
		switch obj := obj.(type) {
		case types.PDFInteger:
			if !validateDi(obj.Value()) {
				return errors.New("validateTransitionDict: entry Di int value undefined")
			}

		case types.PDFName:
			if obj.Value() != "None" {
				return errors.New("validateTransitionDict: entry Di name value undefined")
			}
		}
	}

	// SS, optional, number, since V1.5
	if transStyle != nil && *transStyle == "Fly" {
		_, err = validateNumberEntry(xRefTable, dict, dictName, "SS", OPTIONAL, types.V15, nil)
		if err != nil {
			return
		}
	}

	// B, optional, boolean, since V1.5
	validateB := func(b bool) bool { return transStyle != nil && *transStyle == "Fly" }
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V15, validateB)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTransitionDict end ***")

	return
}

func validatePageEntryTrans(xRefTable *types.XRefTable, pageDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryTrans begin ***")

	dict, err := validateDictEntry(xRefTable, pageDict, "pagesDict", "Trans", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validatePageEntryTrans end: is nil.")
		return
	}

	err = validateTransitionDict(xRefTable, dict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryTrans begin ***")

	return
}

func validatePageEntryStructParents(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validatePageEntryStructParents begin ***")

	_, err = validateIntegerEntry(xRefTable, dict, "pagesDict", "StructParents", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryStructParents end ***")

	return
}

func validatePageEntryID(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryID begin ***")

	_, err = validateStringEntry(xRefTable, dict, "pagesDict", "ID", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryID end ***")

	return
}

func validatePageEntryPZ(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// Preferred zoom factor, number

	logInfoValidate.Println("*** validatePageEntryPZ begin ***")

	_, err = validateNumberEntry(xRefTable, dict, "pagesDict", "PZ", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryPZ end ***")

	return
}

func validateDeviceColorantEntry(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateDeviceColorantEntry begin ***")

	obj, found := dict.Find("DeviceColorant")
	if !found || obj == nil {
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateDeviceColorantEntry end: is nil.")
		return
	}

	switch obj.(type) {

	case types.PDFName, types.PDFStringLiteral:
		// no further processing

	default:
		err = errors.Errorf("validateDeviceColorantEntry: must be name or string.")
		return
	}

	logInfoValidate.Println("*** validateDeviceColorantEntry end ***")

	return
}

func validatePageEntrySeparationInfo(xRefTable *types.XRefTable, pagesDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 14.11.4

	logInfoValidate.Printf("*** writePageEntrySeparationInfo begin ***")

	dict, err := validateDictEntry(xRefTable, pagesDict, "pagesDict", "SeparationInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Printf("writePageEntrySeparationInfo end: is nil.\n")
		return
	}

	_, err = validateIndRefArrayEntry(xRefTable, dict, "separationDict", "Pages", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateDeviceColorantEntry(xRefTable, dict, OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	arr, err := validateArrayEntry(xRefTable, dict, "separationDict", "ColorSpace", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}
	if arr != nil {
		err = validateColorSpaceArray(xRefTable, arr, ExcludePatternCS)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** writePageEntrySeparationInfo end ***")

	return
}

func validatePageEntryTabs(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** writePageEntryTabs begin ***")

	// Include out of spec entry "W"
	validateTabs := func(s string) bool { return memberOf(s, []string{"R", "C", "S", "W"}) }

	_, err = validateNameEntry(xRefTable, dict, "pagesDict", "Tabs", required, sinceVersion, validateTabs)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** writePageEntryTabs end ***")

	return
}

func validatePageEntryTemplateInstantiated(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.6

	logInfoValidate.Println("*** validatePageEntryTemplateInstantiated begin ***")

	_, err = validateNameEntry(xRefTable, dict, "pagesDict", "TemplateInstantiated", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryTemplateInstantiated end ***")

	return
}

// TODO implement
func validatePageEntryPresSteps(xRefTable *types.XRefTable, pagesDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 12.4.4.2

	logInfoValidate.Println("*** validatePageEntryPresSteps begin ***")

	dict, err := validateDictEntry(xRefTable, pagesDict, "pagesDict", "PresSteps", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validatePageEntryPresSteps end: is nil.")
		return
	}

	err = errors.New("*** validatePageEntryPresSteps: not supported ***")

	logInfoValidate.Println("*** validatePageEntryPresSteps end ***")

	return
}

func validatePageEntryUserUnit(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageEntryUserUnit begin ***")

	// UserUnit, optional, positive number, since V1.6
	_, err = validateNumberEntry(xRefTable, dict, "pagesDict", "UserUnit", required, sinceVersion, func(f float64) bool { return f > 0 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageEntryUserUnit end ***")

	return
}

func validateMeasureDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateMeasureDict begin ***")

	dictName := "measureDict"

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Measure" })
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Subtype", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// TODO validate rectilinear measure dict (= default SubType or SubType "RL")

	logInfoValidate.Println("*** validateMeasureDict end ***")

	return
}

func validateViewportDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateViewportDict begin ***")

	dictName := "viewportDict"

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Viewport" })
	if err != nil {
		return
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "BBox", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "Name", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Measure, optional, dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "Measure", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	err = validateMeasureDict(xRefTable, d)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateViewportDict end ***")

	return
}

func validatePageEntryVP(xRefTable *types.XRefTable, pagesDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// see table 260

	logInfoValidate.Println("*** validatePageEntryVP begin ***")

	arr, err := validateArrayEntry(xRefTable, pagesDict, "pagesDict", "VP", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validatePageEntryVP end: is nil.")
		return
	}

	var dict *types.PDFDict

	for _, v := range *arr {

		if v == nil {
			continue
		}

		dict, err = xRefTable.DereferenceDict(v)
		if err != nil {
			return
		}

		if dict == nil {
			continue
		}

		err = validateViewportDict(xRefTable, dict)
		if err != nil {
			return
		}

	}

	logInfoValidate.Printf("*** validatePageEntryVP end ***")

	return
}

func validatePageDict(xRefTable *types.XRefTable, pageDict *types.PDFDict, objNumber, genNumber int, hasResources, hasMediaBox bool) (err error) {

	logInfoValidate.Printf("*** validatePageDict begin: hasResources=%v hasMediaBox=%v obj#%d ***\n", hasResources, hasMediaBox, objNumber)

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
	_, err = validatePageEntryMediaBox(xRefTable, pageDict, !hasMediaBox, types.V10)
	if err != nil {
		return
	}

	// CropBox
	err = validatePageEntryCropBox(xRefTable, pageDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// BleedBox
	err = validatePageEntryBleedBox(xRefTable, pageDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// TrimBox
	err = validatePageEntryTrimBox(xRefTable, pageDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// ArtBox
	err = validatePageEntryArtBox(xRefTable, pageDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// BoxColorInfo
	err = validatePageBoxColorInfo(xRefTable, pageDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// PieceInfo
	sinceVersion := types.V13
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V10
	}
	hasPieceInfo, err := validatePieceInfo(xRefTable, pageDict, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// LastModified
	lm, err := validateDateEntry(xRefTable, pageDict, "pageDict", "LastModified", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	if hasPieceInfo && lm == nil && xRefTable.ValidationMode == types.ValidationStrict {
		return errors.New("writePageDict: missing \"LastModified\" (required by \"PieceInfo\")")
	}

	// Rotate
	err = validatePageEntryRotate(xRefTable, pageDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Group
	err = validatePageEntryGroup(xRefTable, pageDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// Annotations
	// delay until processing of AcroForms.
	// see ...

	// Thumb
	err = validatePageEntryThumb(xRefTable, pageDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// B
	err = validatePageEntryB(xRefTable, pageDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// Dur
	err = validatePageEntryDur(xRefTable, pageDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// Trans
	err = validatePageEntryTrans(xRefTable, pageDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// AA
	err = validateAdditionalActions(xRefTable, pageDict, "pageDict", "AA", OPTIONAL, types.V14, "page")
	if err != nil {
		return
	}

	// Metadata
	err = validateMetadata(xRefTable, pageDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// StructParents
	err = validatePageEntryStructParents(xRefTable, pageDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// ID
	err = validatePageEntryID(xRefTable, pageDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// PZ
	err = validatePageEntryPZ(xRefTable, pageDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// SeparationInfo
	err = validatePageEntrySeparationInfo(xRefTable, pageDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// Tabs
	err = validatePageEntryTabs(xRefTable, pageDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// TemplateInstantiateds
	err = validatePageEntryTemplateInstantiated(xRefTable, pageDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// PresSteps
	err = validatePageEntryPresSteps(xRefTable, pageDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// UserUnit
	err = validatePageEntryUserUnit(xRefTable, pageDict, OPTIONAL, types.V16)
	if err != nil {
		return
	}

	// VP
	err = validatePageEntryVP(xRefTable, pageDict, OPTIONAL, types.V16)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validatePageDict end: obj#%d ***\n", objNumber)

	return
}

func validatePagesDict(xRefTable *types.XRefTable, dict *types.PDFDict, objNumber, genNumber int, hasResources, hasMediaBox bool) (err error) {

	logInfoValidate.Printf("*** validatePagesDict begin: hasResources=%v hasMediaBox=%v obj#%d ***\n", hasResources, hasMediaBox, objNumber)

	// Get number of pages of this PDF file.
	pageCount := dict.IntEntry("Count")
	if pageCount == nil {
		return errors.New("validatePagesDict: missing \"Count\"")
	}

	// TODO not ideal - overall pageCount is only set during validation!
	if xRefTable.PageCount == 0 {
		xRefTable.PageCount = *pageCount
	}

	logInfoValidate.Printf("validatePagesDict: This page node has %d pages\n", *pageCount)

	// Resources: optional, dict
	if obj, ok := dict.Find("Resources"); ok {
		hasResources, err = validateResourceDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// MediaBox: optional, rectangle
	hasPageNodeMediaBox, err := validatePageEntryMediaBox(xRefTable, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	if hasPageNodeMediaBox {
		hasMediaBox = true
	}

	// CropBox: optional, rectangle
	err = validatePageEntryCropBox(xRefTable, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Rotate:  optional, integer
	err = validatePageEntryRotate(xRefTable, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("validatePagesDict: corrupt \"Kids\" entry")
	}

	for _, obj := range *kidsArray {

		if obj == nil {
			logDebugValidate.Println("validatePagesDict: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return errors.New("validatePagesDict: missing indirect reference for kid")
		}

		logInfoValidate.Printf("validatePagesDict: PageNode: %s\n", indRef)

		objNumber := indRef.ObjectNumber.Value()
		genNumber := indRef.GenerationNumber.Value()

		pageNodeDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return err
			//return errors.New("validatePagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return errors.New("validatePagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return errors.New("validatePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

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
			return errors.Errorf("validatePagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	logInfoValidate.Printf("*** validatePagesDict end: obj#%d ***\n", objNumber)

	return
}

func validatePages(xRefTable *types.XRefTable, rootDict *types.PDFDict) (rootPageNodeDict *types.PDFDict, err error) {

	logInfoValidate.Println("*** validatePages begin: ***")

	// Ensure indirect reference entry "Pages".
	indRef := rootDict.IndirectRefEntry("Pages")
	if indRef == nil {
		err = errors.New("validatePages: missing indirect obj for pages dict")
		return
	}

	objNumber := indRef.ObjectNumber.Value()
	genNumber := indRef.GenerationNumber.Value()

	// Dereference root of page node tree.
	rootPageNodeDict, err = xRefTable.DereferenceDict(*indRef)
	if err != nil {
		return
		//return errors.New("validatePagesDict: cannot dereference pageNodeDict")
	}

	if rootPageNodeDict == nil {
		err = errors.New("validatePagesDict: cannot dereference pageNodeDict")
		return
	}

	// Process page node tree.
	err = validatePagesDict(xRefTable, rootPageNodeDict, objNumber, genNumber, false, false)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePages end: ***")

	return
}

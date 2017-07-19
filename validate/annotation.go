package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateAAPLAKExtrasDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (dictp *types.PDFDict, err error) {

	// No documentation for this PDF-Extension - purely speculative implementation.

	logInfoValidate.Println("*** validateAAPLAKExtrasDictEntry begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "AAPLAKExtrasDict"

	// AAPL:AKAnnotationObject, string
	_, err = validateStringEntry(xRefTable, d, dictName, "AAPL:AKAnnotationObject", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// AAPL:AKPDFAnnotationDictionary, annotationDict
	d, err = validateDictEntry(xRefTable, d, dictName, "AAPL:AKPDFAnnotationDictionary", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = validateAnnotationDict(xRefTable, d)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAAPLAKExtrasDictEntry end ***")

	return
}

func validateBorderEffectDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (dictp *types.PDFDict, err error) {

	// see 12.5.4

	logInfoValidate.Println("*** validateBorderEffectDictEntry begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "borderEffectDict"

	// S, optional, name, S or C
	_, err = validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, types.V10, func(s string) bool { return s == "S" || s == "C" })
	if err != nil {
		return
	}

	// I, optional, number in the range 0 to 2
	_, err = validateNumberEntry(xRefTable, d, dictName, "I", OPTIONAL, types.V10, nil) // validation missing
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateBorderEffectDictEntry end ***")

	return
}

func validateBorderStyleDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (dictp *types.PDFDict, err error) {

	// see 12.5.4

	logInfoValidate.Println("*** validateBorderStyleDict begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "borderStyleDict"

	// Type, optional, name, "Border"
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Border" })
	if err != nil {
		return
	}

	// W, optional, number, border width in points
	_, err = validateNumberEntry(xRefTable, d, dictName, "W", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// S, optional, name, border style
	_, err = validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, types.V10, validateBorderStyle)
	if err != nil {
		return
	}

	// D, optional, dash array
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) <= 2 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateBorderStyleDict end ***")

	return
}

func validateIconFitDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (dictp *types.PDFDict, err error) {

	logInfoValidate.Println("*** validateIconFitDictEntry begin ***")

	// see table 247

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "iconFitDict"

	// SW, optional, name, A,B,S,N
	_, err = validateNameEntry(xRefTable, dict, dictName, "SW", OPTIONAL, types.V10, validateIconFitDict)
	if err != nil {
		return
	}

	// S, optional, name, A,P
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, func(s string) bool { return s == "A" || s == "P" })
	if err != nil {
		return
	}

	// A,optional, array of 2 numbers between 0.0 and 1.0
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "A", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// FB, optional, bool, since V1.5
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "FB", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateIconFitDictEntry end ***")

	return
}

func validateAppearanceCharacteristicsDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (dictp *types.PDFDict, err error) {

	// see 12.5.6.19

	logInfoValidate.Println("*** validateAppearanceCharacteristicsDictEntry begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "appCharDict"

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// BC, optional, array of numbers, len=0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BC", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// BG, optional, array of numbers between 0.0 and 0.1, len=0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "BG", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// CA, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CA", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// RC, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "RC", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// AC, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "AC", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// I, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "I", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// RI, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "RI", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// IX, optional, stream dict
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "IX", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// IF, optional, icon fit dict,
	_, err = validateIconFitDictEntry(xRefTable, dict, dictName, "IF", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// TP, optional, integer 0..6
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "TP", OPTIONAL, types.V10, func(i int) bool { return 0 <= i && i <= 6 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAppearanceCharacteristicsDictEntry end ***")

	return
}

func validateAnnotationDictText(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.4

	logInfoValidate.Println("*** validateAnnotationDictText begin ***")

	dictName := "annotText"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictText: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// Open, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Open", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Name", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// State, optional, text string, since V1.5
	state, err := validateStringEntry(xRefTable, dict, dictName, "State", OPTIONAL, types.V15, validateAnnotationState)
	if err != nil {
		return
	}

	// StateModel, text string, since V1.5
	_, err = validateStringEntry(xRefTable, dict, dictName, "StateModel", state != nil, types.V15, validateAnnotationStateModel)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictText end ***")

	return
}

func validateOptionalActionOrDestination(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) (err error) {

	logInfoValidate.Println("*** validateOptionalActionOrDestination begin ***")

	var a *types.PDFDict

	// Validate optional action dict
	// The action that shall be performed when this item is activated.
	a, err = validateDictEntry(xRefTable, dict, dictName, "A", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	if a != nil {
		err = validateActionDict(xRefTable, *a)
		return
	}

	// Validate optional destination
	// The destination that shall be displayed when this item is activated.
	d, found := dict.Find("Dest")
	if !found {
		return
	}

	err = validateDestination(xRefTable, d)

	logInfoValidate.Println("*** validateOptionalActionOrDestination end ***")

	return
}

func validateURIActionDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {
	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "URIActionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Action" })
	if err != nil {
		return
	}

	// S, required, name, action Type
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "URI" })
	if err != nil {
		return
	}

	err = validateURIActionDict(xRefTable, d, types.V10)
	if err != nil {
		return
	}

	return
}

func validateAnnotationDictLink(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.5

	logInfoValidate.Println("*** validateAnnotationDictLink begin ***")

	dictName := "annotLink"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictLink: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// A or D, optional
	err = validateOptionalActionOrDestination(xRefTable, dict, dictName)
	if err != nil {
		return
	}

	// H, optional, name, since V1.2
	_, err = validateNameEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	// PA, optional, URI action dict, since V1.3
	err = validateURIActionDictEntry(xRefTable, dict, dictName, "PA", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// QuadPoints, optional, number array, len=8, since V1.6
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "QuadPoints", OPTIONAL, types.V16, func(a types.PDFArray) bool { return len(a) == 8 })
	if err != nil {
		return
	}

	// BS, optional, border style dict, since V1.6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	_, err = validateBorderStyleDict(xRefTable, dict, dictName, "BS", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictLink end ***")

	return
}

func validateAnnotationDictFreeText(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.6

	logInfoValidate.Println("*** validateAnnotationDictFreeText begin ***")

	dictName := "annotFreeText"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictFreeText: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// DA, required, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "DA", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Q, optional, integer, since V1.4, 0,1,2
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Q", OPTIONAL, sinceVersion, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return
	}

	// RC, optional, text string or text stream, since V1.5
	sinceVersion = types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	err = validateStringOrStreamEntry(xRefTable, dict, dictName, "RC", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	// DS, optional, text string, since V1.5
	_, err = validateStringEntry(xRefTable, dict, dictName, "DS", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	// CL, optional, number array, since V1.6, len: 4 or 6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "CL", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 4 || len(a) == 6 })
	if err != nil {
		return
	}

	// IT, optional, name, since V1.6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "IT", OPTIONAL, sinceVersion, validateIntentOfFreeTextAnnotation)
	if err != nil {
		return
	}

	// BE, optional, border effect dict, since V1.6
	_, err = validateBorderEffectDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// RD, optional, rectangle, since V1.6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "RD", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// BS, optional, border style dict, since V1.6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateBorderStyleDict(xRefTable, dict, dictName, "BS", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	// LE, optional, name, since V1.6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "LE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictFreeText end ***")

	return
}

func validateAnnotationDictLine(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.7

	logInfoValidate.Println("*** validateAnnotationDictLine begin ***")

	dictName := "annotLine"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictLine: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// L, required, array of numbers, len:4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "L", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 4 })
	if err != nil {
		return
	}

	// BS, optional, border style dict
	_, err = validateBorderStyleDict(xRefTable, dict, dictName, "BS", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// LE, optional, name array, since V1.4, len:2
	// TODO validate line ending styles
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "LE", OPTIONAL, sinceVersion, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// IC, optional, number array, since V1.4, len:0,1,3,4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "IC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// LLE, optional, number, since V1.6, >0
	lle, err := validateNumberEntry(xRefTable, dict, dictName, "LLE", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	// LL, required if LLE present, number, since V1.6
	_, err = validateNumberEntry(xRefTable, dict, dictName, "LL", lle != nil, types.V16, nil)
	if err != nil {
		return
	}

	// Cap, optional, bool, since V1.6
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Cap", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	// IT, optional, name, since V1.6
	_, err = validateNameEntry(xRefTable, dict, dictName, "IT", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	// LLO, optionl, number, since V1.7, >0
	_, err = validateNumberEntry(xRefTable, dict, dictName, "LLO", OPTIONAL, types.V17, nil)
	if err != nil {
		return
	}

	// CP, optional, name, since V1.7
	_, err = validateNameEntry(xRefTable, dict, dictName, "CP", OPTIONAL, types.V17, func(s string) bool { return s == "Inline" || s == "Top" })
	if err != nil {
		return
	}

	// Measure, optional, measure dict, since V1.7
	d, err := validateDictEntry(xRefTable, dict, dictName, "Measure", OPTIONAL, types.V17, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = errors.New("validateAnnotationDictLine: unsupported entry \"Measure\"")
		return
	}

	// CO, optional, number array, since V1.7, len=2
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "CO", OPTIONAL, types.V17, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictLine end ***")

	return
}

func validateAnnotationDictCircleOrSquare(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.8

	logInfoValidate.Println("*** validateAnnotationDictCircleOrSquare begin ***")

	dictName := "annotCircleOrSquare"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictCircleOrSquare: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// BS, optional, border style dict
	_, err = validateBorderStyleDict(xRefTable, dict, dictName, "BS", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// IC, optional, array, since V1.4
	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "IC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// BE, optional, border effect dict, since V1.5
	_, err = validateBorderEffectDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// RD, optional, rectangle, since V1.5
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "RD", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictCircleOrSquare end ***")

	return
}

func validateAnnotationDictPolygon(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.9

	return errors.New("validateAnnotationDictPolygon: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictPolygon begin ***")

	// dictName := "annotPolygon"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictPolygon: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictPolygon end ***")

	// return
}

func validateAnnotationDictPolyLine(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.9

	return errors.New("validateAnnotationDictPolyLine: unsupported")

	// 	logInfoValidate.Println("*** validateAnnotationDictPolyLine begin ***")

	// 	dictName := "annotPolyLine"

	// 	// Version check
	// 	if xRefTable.Version() < sinceVersion {
	// 		err = errors.Errorf("validateAnnotationDictText: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 		return
	// 	}

	// 	logInfoValidate.Println("*** validateAnnotationDictPolyLine end ***")

	// 	return
}

func validateTextMarkupAnnotation(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.10

	logInfoValidate.Println("*** validateTextMarkupAnnotation begin ***")

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateTextMarkupAnnotation: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// QuadPoints, required, number array, len:8
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "QuadPoints", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 8 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTextMarkupAnnotation end ***")

	return
}

func validateAnnotationDictStrikeOut(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.10

	return errors.New("validateAnnotationDictStrikeOut: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictStrikeOut begin ***")

	// dictName := "annotStrikeOut"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictStrikeOut: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictStrikeOut end ***")

	// return
}

func validateAnnotationDictStamp(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.12

	logInfoValidate.Println("*** validateAnnotationDictStamp begin ***")

	dictName := "annotStamp"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictStamp: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// Name, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Name", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictStamp end ***")

	return
}

func validateAnnotationDictCaret(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.11

	return errors.New("validateAnnotationDictCaret: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictCaret begin ***")

	// dictName := "annotCaret"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictCaret: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictCaret end ***")

	// return
}

func validateAnnotationDictInk(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.13

	return errors.New("validateAnnotationDictInk: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictInk begin ***")

	// dictName := "annotInk"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictInk: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictInk end ***")

	// return
}

func validateAnnotationDictPopup(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.14

	logInfoValidate.Println("*** validateAnnotationDictPopup begin ***")

	dictName := "annotPopup"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictPopup: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// Parent, optional, dict indRef
	indRef, err := validateIndRefEntry(xRefTable, dict, dictName, "Parent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	dict, err = xRefTable.DereferenceDict(*indRef)
	if err != nil || dict == nil {
		return
	}

	err = validateAnnotationDict(xRefTable, dict)
	if err != nil {
		return
	}

	// Open, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Open", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictPopup end ***")

	return
}

func validateAnnotationDictFileAttachment(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.15

	return errors.New("validateAnnotationDictFileAttachment: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictFileAttachment begin ***")

	// dictName := "fileAttachment"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictFileAttachment: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictFileAttachment end ***")

	// return
}

func validateAnnotationDictSound(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.16
	return errors.New("validateAnnotationDictSound: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictSound begin ***")

	// dictName := "annotSound"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictSound: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictSound end ***")

	// return
}

func validateAnnotationDictMovie(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.17

	return errors.New("validateAnnotationDictMovie: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictMovie begin ***")

	// dictName := "annotMovie"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictText: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictMovie end ***")

	// return
}

func validateAnnotationDictWidget(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.19

	logInfoValidate.Printf("*** validateAnnotationDictWidget begin ***")

	dictName := "widgetAnnotDict"

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAnnotationDictWidget: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
		return
	}

	// H, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V10, validateAnnotationHighlightingMode)
	if err != nil {
		return
	}

	// MK, optional, dict
	// An appearance characteristics dictionary that shall be used in constructing
	// a dynamic appearance stream specifying the annotation’s visual presentation on the page.dict
	_, err = validateAppearanceCharacteristicsDictEntry(xRefTable, dict, dictName, "MK", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// A, optional, dict, since V1.1
	// An action that shall be performed when the annotation is activated.
	d, err := validateDictEntry(xRefTable, dict, dictName, "A", OPTIONAL, types.V11, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateActionDict(xRefTable, *d)
		if err != nil {
			return
		}
	}

	// AA, optional, dict, since V1.2
	// An additional-actions dictionary defining the annotation’s behaviour in response to various trigger events.
	err = validateAdditionalActions(xRefTable, dict, dictName, "AA", OPTIONAL, types.V12, "fieldOrAnnot")
	if err != nil {
		return
	}

	// BS, optional, border style dict, since V1.2
	// A border style dictionary specifying the width and dash pattern
	// that shall be used in drawing the annotation’s border.
	validateBorderStyleDict(xRefTable, dict, dictName, "BS", OPTIONAL, types.V12)
	if err != nil {
		return
	}

	// Parent, dict, required if one of multiple children in a field.
	// An indirect reference to the widget annotation’s parent field.
	_, err = validateIndRefEntry(xRefTable, dict, dictName, "Parent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateAnnotationDictWidget end ***")

	return
}

func validateAnnotationDictScreen(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.18

	return errors.New("validateAnnotationDictScreen: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictScreen begin ***")

	// dictName := "annotScreen"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictScreen: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictScreen end ***")

	// return
}

func validateAnnotationDictPrinterMark(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.20

	return errors.New("validateAnnotationDictPrinterMark: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictPrinterMark begin ***")

	// dictName := "annotPrinterMark"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictPrinterMark: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictPrinterMark end ***")

	// return
}

func validateAnnotationDictTrapNet(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.21

	return errors.New("validateAnnotationDictTrapNet: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictTrapNet begin ***")

	// dictName := "annotTrapNet"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictTrapNet: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictTrapNet end ***")

	// return
}

func validateAnnotationDictWatermark(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.22

	return errors.New("validateAnnotationDictWatermark: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictWatermark begin ***")

	// dictName := "annotWatermark"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictWatermark: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictWatermark end ***")

	// return
}

func validateAnnotationDict3D(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 13.6.2
	return errors.New("validateAnnotationDict3D: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDict3D begin ***")

	// dictName := "annot3D"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDict3D: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDict3D end ***")

	// return
}

func validateAnnotationDictRedact(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	// see 12.5.6.23

	return errors.New("validateAnnotationDictRedact: unsupported")

	// logInfoValidate.Println("*** validateAnnotationDictRedact begin ***")

	// dictName := "annotRedact"

	// // Version check
	// if xRefTable.Version() < sinceVersion {
	// 	err = errors.Errorf("validateAnnotationDictRedact: dict=%s unsupported in version %s", dictName, xRefTable.VersionString())
	// 	return
	// }

	// logInfoValidate.Println("*** validateAnnotationDictRedact end ***")

	// return
}

func validateOptionalContent(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateOptionalContent begin ***")
	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	t, err := validateNameEntry(xRefTable, d, "optionalContent", "Type", REQUIRED, types.V10, func(s string) bool { return s == "OCG" || s == "OCMD" })
	if err != nil {
		return
	}

	switch *t {
	case "OCG":
		err = validateOptionalContentGroupDict(xRefTable, d)
	case "OCMD":
		err = validateOptionalContentMembershipDict(xRefTable, d)
	}

	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateOptionalContent end ***")

	return
}

func validateAnnotationDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateAnnotationDict begin ***")

	dictName := "annotDict"
	var subtype *types.PDFName

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Annot" })
	if err != nil {
		return
	}

	// Subtype, required, name
	subtype, err = validateNameEntry(xRefTable, dict, dictName, "Subtype", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// Rect, required, rectangle
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "Rect", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Contents, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Contents", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P, optional, indRef of page dict
	var indRef *types.PDFIndirectRef
	indRef, err = validateIndRefEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10)
	if err != nil {
		return
	}
	if indRef != nil {
		// check if this indRef points to a pageDict.
		var d *types.PDFDict
		d, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return
		}
		if d == nil {
			err = errors.Errorf("validateAnnotationDict: entry \"P\" (obj#%d) is nil", indRef.ObjectNumber)
		}
		_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Page" })
		if err != nil {
			return
		}

		if d == nil || d.Type() == nil || *d.Type() != "Page" {
			err = errors.Errorf("validateAnnotationDict: entry \"P\" (obj#%d) not a pageDict", indRef.ObjectNumber)
			return
		}
	}

	// NM, optional, text string, since V1.4
	_, err = validateStringEntry(xRefTable, dict, dictName, "NM", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	// M, optional, date string in any format, since V1.1
	_, err = validateStringEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// F, optional integer, since V1.1, annotation flags
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// AP, optional, appearance dict, since V1.2
	d, err := validateDictEntry(xRefTable, dict, dictName, "AP", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateAppearanceDict(xRefTable, *d)
		if err != nil {
			return
		}
	}

	// AS, optional, name, since V1.2
	_, err = validateNameEntry(xRefTable, dict, dictName, "AS", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// Border, optional, array of numbers
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Border", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 3 || len(a) == 4 })
	if err != nil {
		return
	}

	// C, optional array, of numbers, since V1.1
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	// StructParent, optional, integer, since V1.3
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OC, optional, content group dict or content membership dict, since V1.3
	// Specifying the optional content properties for the annotation.
	err = validateOptionalContent(xRefTable, dict, dictName, "OC", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// AAPL:AKExtras
	// No documentation for this PDF-Extension - this is a speculative implementation.
	_, err = validateAAPLAKExtrasDictEntry(xRefTable, dict, dictName, "AAPL:AKExtras", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	switch *subtype {

	case "Text":
		err = validateAnnotationDictText(xRefTable, dict, types.V10)

	case "Link":
		err = validateAnnotationDictLink(xRefTable, dict, types.V10)

	case "FreeText":
		err = validateAnnotationDictFreeText(xRefTable, dict, types.V13)

	case "Line":
		err = validateAnnotationDictLine(xRefTable, dict, types.V13) // unsupported

	case "Square", "Circle":
		err = validateAnnotationDictCircleOrSquare(xRefTable, dict, types.V13)

	case "Polygon":
		err = validateAnnotationDictPolygon(xRefTable, dict, types.V15)

	case "PolyLine":
		err = validateAnnotationDictPolyLine(xRefTable, dict, types.V15) // unsupported

	case "Highlight":
		err = validateTextMarkupAnnotation(xRefTable, dict, "annotHighlight", types.V13)

	case "Underline":
		err = validateTextMarkupAnnotation(xRefTable, dict, "annotUnderline", types.V13)

	case "Squiggly":
		err = validateTextMarkupAnnotation(xRefTable, dict, "annotSquiggly", types.V14)

	case "StrikeOut":
		err = validateTextMarkupAnnotation(xRefTable, dict, "annotStrikeout", types.V13)

	case "Stamp":
		err = validateAnnotationDictStamp(xRefTable, dict, types.V13)

	case "Caret":
		err = validateAnnotationDictCaret(xRefTable, dict, types.V15) // unsupported

	case "Ink":
		err = validateAnnotationDictInk(xRefTable, dict, types.V13) // unsupported

	case "Popup":
		err = validateAnnotationDictPopup(xRefTable, dict, types.V13)

	case "FileAttachment":
		err = validateAnnotationDictFileAttachment(xRefTable, dict, types.V13) // unsupported

	case "Sound":
		err = validateAnnotationDictSound(xRefTable, dict, types.V12) // unsupported

	case "Movie":
		err = validateAnnotationDictMovie(xRefTable, dict, types.V12) // unsupported

	case "Widget":
		err = validateAnnotationDictWidget(xRefTable, dict, types.V12)

	case "Screen":
		err = validateAnnotationDictScreen(xRefTable, dict, types.V15) /// unsupported

	case "PrinterMark":
		err = validateAnnotationDictPrinterMark(xRefTable, dict, types.V14) // unsupported

	case "TrapNet":
		err = validateAnnotationDictTrapNet(xRefTable, dict, types.V13) // unsupported

	case "Watermark":
		err = validateAnnotationDictWatermark(xRefTable, dict, types.V16) // unsupported

	case "3D":
		err = validateAnnotationDict3D(xRefTable, dict, types.V16) // unsupported

	case "Redact":
		err = validateAnnotationDictRedact(xRefTable, dict, types.V17) // unsupported

	default:
		return errors.Errorf("validateAnnotationDict: unsupported annotation subtype:%s\n", *subtype)

	}

	if err == nil {
		logInfoValidate.Println("*** validateAnnotationDict end ***")
	}

	return
}

func validatePageAnnotations(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validatePageAnnotations begin ***")

	var arr *types.PDFArray

	arr, err = validateArrayEntry(xRefTable, dict, "pageDict", "Annots", OPTIONAL, types.V10, nil)
	if err != nil || arr == nil {
		logInfoValidate.Println("*** validatePageAnnotations end, no annotaions found ***")
		return
	}

	// array of indrefs to annotation dicts.
	var annotsDict types.PDFDict

	for _, v := range *arr {

		if indRef, ok := v.(types.PDFIndirectRef); ok {

			annotsDictp, err := xRefTable.DereferenceDict(indRef)
			if err != nil || annotsDictp == nil {
				return errors.New("validatePageAnnotations: corrupted annotation dict")
			}

			annotsDict = *annotsDictp

		} else if annotsDict, ok = v.(types.PDFDict); !ok {
			return errors.New("validatePageAnnotations: corrupted array of indrefs")
		}

		err = validateAnnotationDict(xRefTable, &annotsDict)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validatePageAnnotations end ***")

	return
}

func validatePagesAnnotations(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validatePagesAnnotations begin ***")

	// Get number of pages of this PDF file.
	pageCount := dict.IntEntry("Count")
	if pageCount == nil {
		return errors.New("validatePagesAnnotations: missing \"Count\"")
	}

	logInfoValidate.Printf("validatePagesAnnotations: This page node has %d pages\n", *pageCount)

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")

	for _, v := range *kidsArray {

		if v == nil {
			logDebugValidate.Println("validatePagesAnnotations: kid is nil")
			continue
		}

		var d *types.PDFDict

		d, err = xRefTable.DereferenceDict(v)
		if err != nil {
			return
		}

		if d == nil {
			return errors.New("validatePagesAnnotations: pageNodeDict is null")
		}

		dictType := d.Type()
		if dictType == nil {
			return errors.New("validatePagesAnnotations: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			err = validatePagesAnnotations(xRefTable, d)
			if err != nil {
				return
			}

		case "Page":
			err = validatePageAnnotations(xRefTable, d)
			if err != nil {
				return
			}

		default:
			return errors.Errorf("validatePagesAnnotations: expected dict type: %s\n", *dictType)

		}

	}

	logInfoValidate.Println("*** validatePagesAnnotations end ***")

	return
}

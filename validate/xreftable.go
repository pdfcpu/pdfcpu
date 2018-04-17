package validate

import (
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// XRefTable validates a PDF cross reference table obeying the validation mode.
func XRefTable(xRefTable *types.XRefTable) error {

	log.Info.Println("validating")
	log.Debug.Println("*** validateXRefTable begin ***")

	// Validate root object(aka the document catalog) and page tree.
	err := validateRootObject(xRefTable)
	if err != nil {
		return err
	}

	// Validate document information dictionary.
	err = validateDocumentInfoObject(xRefTable)
	if err != nil {
		return err
	}

	// Validate offspec additional streams as declared in pdf trailer.
	err = validateAdditionalStreams(xRefTable)
	if err != nil {
		return err
	}

	xRefTable.Valid = true

	log.Debug.Println("*** validateXRefTable end ***")

	return nil

}

func validateRootVersion(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "Version", OPTIONAL, types.V14, nil)

	return err
}

func validateExtensions(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 7.12 Extensions Dictionary

	_, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Extensions", required, sinceVersion, nil)

	// No validation due to lack of documentation.

	return err
}

func validatePageLabels(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// optional since PDF 1.3
	// => 7.9.7 Number Trees, 12.4.2 Page Labels

	// PDFDict or indirect ref to PDFDict

	indRef := rootDict.IndirectRefEntry("PageLabels")
	if indRef == nil {
		if required {
			return errors.Errorf("validatePageLabels: required entry \"PageLabels\" missing")
		}
		return nil
	}

	dictName := "PageLabels"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	_, _, err = validateNumberTree(xRefTable, "PageLabel", *indRef, true)

	return err
}

func validateNames(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 7.7.4 Name Dictionary

	/*
		<Kids, [(86 0 R)]>

		86:
		<Limits, [(F1) (P.9)]>
		<Names, [(F1) (87 0 R) (F2) ...

	*/

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Names", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	validateNameTreeName := func(s string) bool {
		return memberOf(s, []string{"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
			"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"})
	}

	for treeName, value := range dict.Dict {

		if ok := validateNameTreeName(treeName); !ok {
			return errors.Errorf("validateNames: unknown name tree name: %s\n", treeName)
		}

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			return errors.New("validateNames: name tree must be indirect ref")
		}

		_, _, tree, err := validateNameTree(xRefTable, treeName, indRef, true)
		if err != nil {
			return err
		}

		if tree != nil {
			xRefTable.Names[treeName] = tree
		}

	}

	return nil
}

func validateNamedDestinations(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.3.2.3 Named Destinations

	// indRef or dict with destination array values.

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Dests", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	for _, value := range dict.Dict {
		err = validateDestination(xRefTable, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateViewerPreferences(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.2 Viewer Preferences

	dictName := "rootDict"

	dict, err := validateDictEntry(xRefTable, rootDict, dictName, "ViewerPreferences", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName = "ViewerPreferences"

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HideToolbar", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HideMenubar", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HideWindowUI", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "FitWindow", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "CenterWindow", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V10
	}
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "DisplayDocTitle", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	validate := func(s string) bool { return memberOf(s, []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "NonFullScreenPageMode", OPTIONAL, types.V10, validate)
	if err != nil {
		return err
	}

	validate = func(s string) bool { return memberOf(s, []string{"L2R", "R2L"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "Direction", OPTIONAL, types.V13, validate)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "ViewArea", OPTIONAL, types.V14, nil)

	return err
}

func validatePageLayout(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageLayout", required, sinceVersion, nil)

	return err
}

func validatePageMode(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageMode", required, sinceVersion, nil)

	return err
}

func validateOpenAction(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.3.2 Destinations, 12.6 Actions

	// A value specifying a destination that shall be displayed
	// or an action that shall be performed when the document is opened.
	// The value shall be either an array defining a destination (see 12.3.2, "Destinations")
	// or an action dictionary representing an action (12.6, "Actions").
	//
	// If this entry is absent, the document shall be opened
	// to the top of the first page at the default magnification factor.

	obj, err := validateEntry(xRefTable, rootDict, "rootDict", "OpenAction", required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateActionDict(xRefTable, &obj)

	case types.PDFArray:
		err = validateDestinationArray(xRefTable, &obj)

	default:
		err = errors.New("validateOpenAction: unexpected object")
	}

	return err
}

func validateURI(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.6.4.7 URI Actions

	// URI dict with one optional entry Base, ASCII string

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "URI", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	// Base, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, "URIdict", "Base", OPTIONAL, types.V10, nil)

	return err
}

func validateRootMetadata(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	return validateMetadata(xRefTable, rootDict, required, sinceVersion)
}

func validateMetadata(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 14.3 Metadata
	// In general, any PDF stream or dictionary may have metadata attached to it
	// as long as the stream or dictionary represents an actual information resource,
	// as opposed to serving as an implementation artifact.
	// Some PDF constructs are considered implementational, and hence may not have associated metadata.

	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}

	streamDict, err := validateStreamDictEntry(xRefTable, dict, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil || streamDict == nil {
		return err
	}

	dictName := "metaDataDict"

	_, err = validateNameEntry(xRefTable, &streamDict.PDFDict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Metadata" })
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, &streamDict.PDFDict, dictName, "SubType", OPTIONAL, sinceVersion, func(s string) bool { return s == "XML" })

	return err
}

func validateMarkInfo(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 14.7 Logical Structure

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "MarkInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	var isTaggedPDF bool

	dictName := "markInfoDict"

	// Marked, optional, boolean
	marked, err := validateBooleanEntry(xRefTable, dict, dictName, "Marked", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if marked != nil {
		isTaggedPDF = marked.Value()
	}

	// Suspects: optional, since V1.6, boolean
	suspects, err := validateBooleanEntry(xRefTable, dict, dictName, "Suspects", OPTIONAL, types.V16, nil)
	if err != nil {
		return err
	}

	if suspects != nil && suspects.Value() {
		isTaggedPDF = false
	}

	xRefTable.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "UserProperties", OPTIONAL, types.V16, nil)

	return err
}

func validateLang(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	_, err := validateStringEntry(xRefTable, rootDict, "rootDict", "Lang", required, sinceVersion, nil)

	return err
}

func validateCaptureCommandDictArray(xRefTable *types.XRefTable, arr *types.PDFArray) error {

	for _, v := range *arr {

		dict, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		err = validateCaptureCommandDict(xRefTable, dict)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateWebCaptureInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "webCaptureInfoDict"

	// V, required, since V1.3, number
	_, err := validateNumberEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V13, nil)
	if err != nil {
		return err
	}

	// C, optional, since V1.3, array of web capture command dict indRefs
	arr, err := validateIndRefArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	if arr != nil {
		err = validateCaptureCommandDictArray(xRefTable, arr)
	}

	return err
}

func validateSpiderInfo(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// 14.10.2 Web Capture Information Dictionary

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "SpiderInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return validateWebCaptureInfoDict(xRefTable, dict)
}

func validateOutputIntentDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "outputIntentDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "OutputIntent" })
	if err != nil {
		return err
	}

	// S: required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// OutputCondition, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "OutputCondition", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// OutputConditionIdentifier, required, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "OutputConditionIdentifier", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// RegistryName, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "RegistryName", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Info, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Info", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// DestOutputProfile, optional, streamDict
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "DestOutputProfile", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateOutputIntents(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 14.11.5 Output Intents

	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}

	arr, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "OutputIntents", required, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	for _, v := range *arr {

		dict, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		err = validateOutputIntentDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePieceDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "pieceDict"

	for _, obj := range dict.Dict {

		dict, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		_, err = validateDateEntry(xRefTable, dict, dictName, "LastModified", REQUIRED, types.V10)
		if err != nil {
			return err
		}

		_, err = validateEntry(xRefTable, dict, dictName, "Private", OPTIONAL, types.V10)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRootPieceInfo(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	_, err := validatePieceInfo(xRefTable, rootDict, "rootDict", "PieceInfo", required, sinceVersion)

	return err
}

func validatePieceInfo(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (hasPieceInfo bool, err error) {

	// 14.5 Page-Piece Dictionaries

	pieceDict, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || pieceDict == nil {
		return false, err
	}

	err = validatePieceDict(xRefTable, pieceDict)
	if err != nil {
		return false, err
	}

	return hasPieceInfo, nil
}

// TODO implement
func validatePermissions(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.8.4 Permissions

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Permissions", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return errors.New("*** validatePermissions: not supported ***")
}

// TODO implement
func validateLegal(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.8.5 Legal Content Attestations

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Legal", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return errors.New("*** validateLegal: not supported ***")
}

func validateRequirementDict(xRefTable *types.XRefTable, dict *types.PDFDict, sinceVersion types.PDFVersion) error {

	dictName := "requirementDict"

	// Type, optional, name,
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Requirement" })
	if err != nil {
		return err
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "EnableJavaScripts" })
	if err != nil {
		return err
	}

	// The RH entry (requirement handler dicts) shall not be used in PDF 1.7.

	return nil
}

func validateRequirements(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.10 Document Requirements

	arr, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "Requirements", required, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	for _, obj := range *arr {

		d, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateRequirementDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateCollectionFieldDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "colFlddict"

	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "CollectionField" })
	if err != nil {
		return err
	}

	// Subtype, required name
	validateCollectionFieldSubtype := func(s string) bool {
		return memberOf(s, []string{"S", "D", "N", "F", "Desc", "ModDate", "CreationDate", "Size"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Subtype", REQUIRED, types.V10, validateCollectionFieldSubtype)
	if err != nil {
		return err
	}

	// N, required text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// O, optional integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// V, optional boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "V", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// E, optional boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "E", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateCollectionSchemaDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	for k, v := range dict.Dict {

		if k == "Type" {

			var n types.PDFName
			n, err := xRefTable.DereferenceName(v, types.V10, nil)
			if err != nil {
				return err
			}

			if n != "CollectionSchema" {
				return errors.New("validateCollectionSchemaDict: invalid entry \"Type\"")
			}

			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateCollectionFieldDict(xRefTable, d)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateCollectionSortDict(xRefTable *types.XRefTable, d *types.PDFDict) error {

	dictName := "colSortDict"

	// S, required name or array of names.
	err := validateNameOrArrayOfNameEntry(xRefTable, d, dictName, "S", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// A, optional boolean or array of booleans.
	err = validateBooleanOrArrayOfBooleanEntry(xRefTable, d, dictName, "A", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	return nil
}

func validateInitialView(s string) bool { return s == "D" || s == "T" || s == "H" }

func validateCollection(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// => 12.3.5 Collections

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Collection", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName := "Collection"

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Collection" })
	if err != nil {
		return err
	}

	// Schema, optional dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "Schema", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateCollectionSchemaDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// D, optional string
	_, err = validateStringEntry(xRefTable, dict, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// View, optional name
	_, err = validateNameEntry(xRefTable, dict, dictName, "View", OPTIONAL, sinceVersion, validateInitialView)
	if err != nil {
		return err
	}

	// Sort, optional dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "Sort", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateCollectionSortDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateNeedsRendering(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	_, err := validateBooleanEntry(xRefTable, rootDict, "rootDict", "NeedsRendering", required, sinceVersion, nil)

	return err
}

func validateRootObject(xRefTable *types.XRefTable) error {

	log.Debug.Println("*** validateRootObject begin ***")

	// => 7.7.2 Document Catalog

	// Entry               opt  since       type            info
	// ------------------------------------------------------------------------------------
	// Type                 n               string          "Catalog"
	// Version              y   1.4         name            overrules header version if later
	// Extensions           y   ISO 32000   dict            => 7.12 Extensions Dictionary
	// Pages                n   -           (dict)          => 7.7.3 Page Tree
	// PageLabels           y   1.3         number tree     => 7.9.7 Number Trees, 12.4.2 Page Labels
	// Names                y   1.2         dict            => 7.7.4 Name Dictionary
	// Dests                y   only 1.1    (dict)          => 12.3.2.3 Named Destinations
	// ViewerPreferences    y   1.2         dict            => 12.2 Viewer Preferences
	// PageLayout           y   -           name            /SinglePage, /OneColumn etc.
	// PageMode             y   -           name            /UseNone, /FullScreen etc.
	// Outlines             y   -           (dict)          => 12.3.3 Document Outline
	// Threads              y   1.1         (array)         => 12.4.3 Articles
	// OpenAction           y   1.1         array or dict   => 12.3.2 Destinations, 12.6 Actions
	// AA                   y   1.4         dict            => 12.6.3 Trigger Events
	// URI                  y   1.1         dict            => 12.6.4.7 URI Actions
	// AcroForm             y   1.2         dict            => 12.7.2 Interactive Form Dictionary
	// Metadata             y   1.4         (stream)        => 14.3.2 Metadata Streams
	// StructTreeRoot       y   1.3         dict            => 14.7.2 Structure Hierarchy
	// Markinfo             y   1.4         dict            => 14.7 Logical Structure
	// Lang                 y   1.4         string
	// SpiderInfo           y   1.3         dict            => 14.10.2 Web Capture Information Dictionary
	// OutputIntents        y   1.4         array           => 14.11.5 Output Intents
	// PieceInfo            y   1.4         dict            => 14.5 Page-Piece Dictionaries
	// OCProperties         y   1.5         dict            => 8.11.4 Configuring Optional Content
	// Perms                y   1.5         dict            => 12.8.4 Permissions
	// Legal                y   1.5         dict            => 12.8.5 Legal Content Attestations
	// Requirements         y   1.7         array           => 12.10 Document Requirements
	// Collection           y   1.7         dict            => 12.3.5 Collections
	// NeedsRendering       y   1.7         boolean         => XML Forms Architecture (XFA) Spec.

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	// Type
	_, err = validateNameEntry(xRefTable, rootDict, "rootDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Catalog" })
	if err != nil {
		return err
	}

	// Pages
	rootPageNodeDict, err := validatePages(xRefTable, rootDict)
	if err != nil {
		return err
	}

	for _, f := range []struct {
		validate     func(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error)
		required     bool
		sinceVersion types.PDFVersion
	}{
		{validateRootVersion, OPTIONAL, types.V14},
		{validateExtensions, OPTIONAL, types.V10},
		{validatePageLabels, OPTIONAL, types.V13},
		{validateNames, OPTIONAL, types.V12},
		{validateNamedDestinations, OPTIONAL, types.V11},
		{validateViewerPreferences, OPTIONAL, types.V12},
		{validatePageLayout, OPTIONAL, types.V10},
		{validatePageMode, OPTIONAL, types.V10},
		{validateOutlines, OPTIONAL, types.V10},
		{validateThreads, OPTIONAL, types.V11},
		{validateOpenAction, OPTIONAL, types.V11},
		{validateRootAdditionalActions, OPTIONAL, types.V14},
		{validateURI, OPTIONAL, types.V11},
		{validateAcroForm, OPTIONAL, types.V12},
		{validateRootMetadata, OPTIONAL, types.V14},
		{validateStructTree, OPTIONAL, types.V13},
		{validateMarkInfo, OPTIONAL, types.V14},
		{validateLang, OPTIONAL, types.V10},
		{validateSpiderInfo, OPTIONAL, types.V13},
		{validateOutputIntents, OPTIONAL, types.V14},
		{validateRootPieceInfo, OPTIONAL, types.V14},
		{validateOCProperties, OPTIONAL, types.V15},
		{validatePermissions, OPTIONAL, types.V15},
		{validateLegal, OPTIONAL, types.V17},
		{validateRequirements, OPTIONAL, types.V17},
		{validateCollection, OPTIONAL, types.V17},
		{validateNeedsRendering, OPTIONAL, types.V17},
	} {
		err = f.validate(xRefTable, rootDict, f.required, f.sinceVersion)
		if err != nil {
			return err
		}
	}

	// Validate remainder of annotations after AcroForm validation only.
	err = validatePagesAnnotations(xRefTable, rootPageNodeDict)

	log.Debug.Println("*** validateRootObject end ***")

	return err
}

func validateAdditionalStreams(xRefTable *types.XRefTable) error {

	// Out of spec scope.
	return nil
}

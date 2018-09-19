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

// Package validate implements validation against PDF 32000-1:2008.
package validate

import (
	"github.com/hhrutter/pdfcpu/pkg/log"
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// XRefTable validates a PDF cross reference table obeying the validation mode.
func XRefTable(xRefTable *pdf.XRefTable) error {

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

func validateRootVersion(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "Version", OPTIONAL, pdf.V14, nil)

	return err
}

func validateExtensions(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 7.12 Extensions Dictionary

	_, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Extensions", required, sinceVersion, nil)

	// No validation due to lack of documentation.

	return err
}

func validatePageLabels(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// optional since PDF 1.3
	// => 7.9.7 Number Trees, 12.4.2 Page Labels

	// Dict or indirect ref to Dict

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

func validateNames(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

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
		return pdf.MemberOf(s, []string{"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
			"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"})
	}

	for treeName, value := range *dict {

		if ok := validateNameTreeName(treeName); !ok {
			return errors.Errorf("validateNames: unknown name tree name: %s\n", treeName)
		}

		indRef, ok := value.(pdf.IndirectRef)
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

func validateNamedDestinations(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.3.2.3 Named Destinations

	// indRef or dict with destination array values.

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Dests", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	for _, value := range *dict {
		err = validateDestination(xRefTable, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateViewerPreferences(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.2 Viewer Preferences

	dictName := "rootDict"

	dict, err := validateDictEntry(xRefTable, rootDict, dictName, "ViewerPreferences", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName = "ViewerPreferences"

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HideToolbar", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HideMenubar", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HideWindowUI", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "FitWindow", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "CenterWindow", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V10
	}
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "DisplayDocTitle", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	validate := func(s string) bool { return pdf.MemberOf(s, []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "NonFullScreenPageMode", OPTIONAL, pdf.V10, validate)
	if err != nil {
		return err
	}

	validate = func(s string) bool { return pdf.MemberOf(s, []string{"L2R", "R2L"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "Direction", OPTIONAL, pdf.V13, validate)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "ViewArea", OPTIONAL, pdf.V14, nil)

	return err
}

func validatePageLayout(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageLayout", required, sinceVersion, nil)

	return err
}

func validatePageMode(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageMode", required, sinceVersion, nil)

	return err
}

func validateOpenAction(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

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

	case pdf.Dict:
		err = validateActionDict(xRefTable, &obj)

	case pdf.Array:
		err = validateDestinationArray(xRefTable, &obj)

	default:
		err = errors.New("validateOpenAction: unexpected object")
	}

	return err
}

func validateURI(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.6.4.7 URI Actions

	// URI dict with one optional entry Base, ASCII string

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "URI", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	// Base, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, "URIdict", "Base", OPTIONAL, pdf.V10, nil)

	return err
}

func validateRootMetadata(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	return validateMetadata(xRefTable, rootDict, required, sinceVersion)
}

func validateMetadata(xRefTable *pdf.XRefTable, dict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 14.3 Metadata
	// In general, any PDF stream or dictionary may have metadata attached to it
	// as long as the stream or dictionary represents an actual information resource,
	// as opposed to serving as an implementation artifact.
	// Some PDF constructs are considered implementational, and hence may not have associated metadata.

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}

	streamDict, err := validateStreamDictEntry(xRefTable, dict, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil || streamDict == nil {
		return err
	}

	dictName := "metaDataDict"

	_, err = validateNameEntry(xRefTable, &streamDict.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Metadata" })
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, &streamDict.Dict, dictName, "SubType", OPTIONAL, sinceVersion, func(s string) bool { return s == "XML" })

	return err
}

func validateMarkInfo(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 14.7 Logical Structure

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "MarkInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	var isTaggedPDF bool

	dictName := "markInfoDict"

	// Marked, optional, boolean
	marked, err := validateBooleanEntry(xRefTable, dict, dictName, "Marked", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}
	if marked != nil {
		isTaggedPDF = marked.Value()
	}

	// Suspects: optional, since V1.6, boolean
	suspects, err := validateBooleanEntry(xRefTable, dict, dictName, "Suspects", OPTIONAL, pdf.V16, nil)
	if err != nil {
		return err
	}

	if suspects != nil && suspects.Value() {
		isTaggedPDF = false
	}

	xRefTable.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "UserProperties", OPTIONAL, pdf.V16, nil)

	return err
}

func validateLang(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateStringEntry(xRefTable, rootDict, "rootDict", "Lang", required, sinceVersion, nil)

	return err
}

func validateCaptureCommandDictArray(xRefTable *pdf.XRefTable, arr *pdf.Array) error {

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

func validateWebCaptureInfoDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "webCaptureInfoDict"

	// V, required, since V1.3, number
	_, err := validateNumberEntry(xRefTable, dict, dictName, "V", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	// C, optional, since V1.3, array of web capture command dict indRefs
	arr, err := validateIndRefArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	if arr != nil {
		err = validateCaptureCommandDictArray(xRefTable, arr)
	}

	return err
}

func validateSpiderInfo(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// 14.10.2 Web Capture Information Dictionary

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "SpiderInfo", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return validateWebCaptureInfoDict(xRefTable, dict)
}

func validateOutputIntentDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "outputIntentDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "OutputIntent" })
	if err != nil {
		return err
	}

	// S: required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// OutputCondition, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "OutputCondition", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// OutputConditionIdentifier, required, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "OutputConditionIdentifier", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// RegistryName, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "RegistryName", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Info, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Info", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// DestOutputProfile, optional, streamDict
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "DestOutputProfile", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateOutputIntents(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 14.11.5 Output Intents

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
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

func validatePieceDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "pieceDict"

	for _, obj := range *dict {

		dict, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		_, err = validateDateEntry(xRefTable, dict, dictName, "LastModified", REQUIRED, pdf.V10)
		if err != nil {
			return err
		}

		_, err = validateEntry(xRefTable, dict, dictName, "Private", OPTIONAL, pdf.V10)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRootPieceInfo(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validatePieceInfo(xRefTable, rootDict, "rootDict", "PieceInfo", required, sinceVersion)

	return err
}

func validatePieceInfo(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (hasPieceInfo bool, err error) {

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
func validatePermissions(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.8.4 Permissions

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Permissions", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return errors.New("*** validatePermissions: not supported ***")
}

// TODO implement
func validateLegal(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.8.5 Legal Content Attestations

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Legal", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	return errors.New("*** validateLegal: not supported ***")
}

func validateRequirementDict(xRefTable *pdf.XRefTable, dict *pdf.Dict, sinceVersion pdf.Version) error {

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

func validateRequirements(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

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

func validateCollectionFieldDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "colFlddict"

	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "CollectionField" })
	if err != nil {
		return err
	}

	// Subtype, required name
	validateCollectionFieldSubtype := func(s string) bool {
		return pdf.MemberOf(s, []string{"S", "D", "N", "F", "Desc", "ModDate", "CreationDate", "Size"})
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Subtype", REQUIRED, pdf.V10, validateCollectionFieldSubtype)
	if err != nil {
		return err
	}

	// N, required text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// O, optional integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "O", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// V, optional boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "V", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// E, optional boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "E", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateCollectionSchemaDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	for k, v := range *dict {

		if k == "Type" {

			var n pdf.Name
			n, err := xRefTable.DereferenceName(v, pdf.V10, nil)
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

func validateCollectionSortDict(xRefTable *pdf.XRefTable, d *pdf.Dict) error {

	dictName := "colSortDict"

	// S, required name or array of names.
	err := validateNameOrArrayOfNameEntry(xRefTable, d, dictName, "S", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// A, optional boolean or array of booleans.
	err = validateBooleanOrArrayOfBooleanEntry(xRefTable, d, dictName, "A", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	return nil
}

func validateInitialView(s string) bool { return s == "D" || s == "T" || s == "H" }

func validateCollection(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

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

func validateNeedsRendering(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateBooleanEntry(xRefTable, rootDict, "rootDict", "NeedsRendering", required, sinceVersion, nil)

	return err
}

func validateRootObject(xRefTable *pdf.XRefTable) error {

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
	_, err = validateNameEntry(xRefTable, rootDict, "rootDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Catalog" })
	if err != nil {
		return err
	}

	// Pages
	rootPageNodeDict, err := validatePages(xRefTable, rootDict)
	if err != nil {
		return err
	}

	for _, f := range []struct {
		validate     func(xRefTable *pdf.XRefTable, rootDict *pdf.Dict, required bool, sinceVersion pdf.Version) (err error)
		required     bool
		sinceVersion pdf.Version
	}{
		{validateRootVersion, OPTIONAL, pdf.V14},
		{validateExtensions, OPTIONAL, pdf.V10},
		{validatePageLabels, OPTIONAL, pdf.V13},
		{validateNames, OPTIONAL, pdf.V12},
		{validateNamedDestinations, OPTIONAL, pdf.V11},
		{validateViewerPreferences, OPTIONAL, pdf.V12},
		{validatePageLayout, OPTIONAL, pdf.V10},
		{validatePageMode, OPTIONAL, pdf.V10},
		{validateOutlines, OPTIONAL, pdf.V10},
		{validateThreads, OPTIONAL, pdf.V11},
		{validateOpenAction, OPTIONAL, pdf.V11},
		{validateRootAdditionalActions, OPTIONAL, pdf.V14},
		{validateURI, OPTIONAL, pdf.V11},
		{validateAcroForm, OPTIONAL, pdf.V12},
		{validateRootMetadata, OPTIONAL, pdf.V14},
		{validateStructTree, OPTIONAL, pdf.V13},
		{validateMarkInfo, OPTIONAL, pdf.V14},
		{validateLang, OPTIONAL, pdf.V10},
		{validateSpiderInfo, OPTIONAL, pdf.V13},
		{validateOutputIntents, OPTIONAL, pdf.V14},
		{validateRootPieceInfo, OPTIONAL, pdf.V14},
		{validateOCProperties, OPTIONAL, pdf.V15},
		{validatePermissions, OPTIONAL, pdf.V15},
		{validateLegal, OPTIONAL, pdf.V17},
		{validateRequirements, OPTIONAL, pdf.V17},
		{validateCollection, OPTIONAL, pdf.V17},
		{validateNeedsRendering, OPTIONAL, pdf.V17},
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

func validateAdditionalStreams(xRefTable *pdf.XRefTable) error {

	// Out of spec scope.
	return nil
}

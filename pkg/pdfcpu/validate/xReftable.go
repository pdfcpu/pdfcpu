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
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// XRefTable validates a PDF cross reference table obeying the validation mode.
func XRefTable(xRefTable *pdf.XRefTable) error {

	log.Info.Println("validating")
	log.Validate.Println("*** validateXRefTable begin ***")

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

	log.Validate.Println("*** validateXRefTable end ***")

	return nil
}

func validateRootVersion(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "Version", OPTIONAL, sinceVersion, nil)

	return err
}

func validateExtensions(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 7.12 Extensions Dictionary

	_, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Extensions", required, sinceVersion, nil)

	// No validation due to lack of documentation.

	return err
}

func validatePageLabels(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// optional since PDF 1.3
	// => 7.9.7 Number Trees, 12.4.2 Page Labels

	// Dict or indirect ref to Dict

	ir := rootDict.IndirectRefEntry("PageLabels")
	if ir == nil {
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

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return err
	}

	_, _, err = validateNumberTree(xRefTable, "PageLabel", d, true)

	return err
}

func validateNames(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 7.7.4 Name Dictionary

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Names", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	validateNameTreeName := func(s string) bool {
		return pdf.MemberOf(s, []string{"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
			"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"})
	}

	for treeName, value := range d {

		if ok := validateNameTreeName(treeName); !ok {
			return errors.Errorf("validateNames: unknown name tree name: %s\n", treeName)
		}

		d, err := xRefTable.DereferenceDict(value)
		if err != nil {
			return err
		}
		if d == nil || len(d) == 0 {
			continue
		}

		_, _, tree, err := validateNameTree(xRefTable, treeName, d, true)
		if err != nil {
			return err
		}

		// Internalize this name tree.
		// If no validation takes place, name trees have to be internalized via xRefTable.LocateNameTree
		// TODO Move this out of validation into Read.
		if tree != nil {
			xRefTable.Names[treeName] = tree
		}

	}

	return nil
}

func validateNamedDestinations(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.3.2.3 Named Destinations

	// indRef or dict with destination array values.

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Dests", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	for _, o := range d {
		err = validateDestination(xRefTable, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateViewerPreferences(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.2 Viewer Preferences

	dictName := "rootDict"

	d, err := validateDictEntry(xRefTable, rootDict, dictName, "ViewerPreferences", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "ViewerPreferences"

	_, err = validateBooleanEntry(xRefTable, d, dictName, "HideToolbar", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, dictName, "HideMenubar", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, dictName, "HideWindowUI", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, dictName, "FitWindow", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, d, dictName, "CenterWindow", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	sinceVersion = pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V10
	}
	_, err = validateBooleanEntry(xRefTable, d, dictName, "DisplayDocTitle", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	validate := func(s string) bool { return pdf.MemberOf(s, []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"}) }
	_, err = validateNameEntry(xRefTable, d, dictName, "NonFullScreenPageMode", OPTIONAL, pdf.V10, validate)
	if err != nil {
		return err
	}

	validate = func(s string) bool { return pdf.MemberOf(s, []string{"L2R", "R2L"}) }
	_, err = validateNameEntry(xRefTable, d, dictName, "Direction", OPTIONAL, pdf.V13, validate)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "ViewArea", OPTIONAL, pdf.V14, nil)

	return err
}

func validatePageLayout(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageLayout", required, sinceVersion, nil)

	return err
}

func validatePageMode(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageMode", required, sinceVersion, nil)

	return err
}

func validateOpenAction(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.3.2 Destinations, 12.6 Actions

	// A value specifying a destination that shall be displayed
	// or an action that shall be performed when the document is opened.
	// The value shall be either an array defining a destination (see 12.3.2, "Destinations")
	// or an action dictionary representing an action (12.6, "Actions").
	//
	// If this entry is absent, the document shall be opened
	// to the top of the first page at the default magnification factor.

	o, err := validateEntry(xRefTable, rootDict, "rootDict", "OpenAction", required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Dict:
		err = validateActionDict(xRefTable, o)

	case pdf.Array:
		err = validateDestinationArray(xRefTable, o)

	default:
		err = errors.New("pdfcpu: validateOpenAction: unexpected object")
	}

	return err
}

func validateURI(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.6.4.7 URI Actions

	// URI dict with one optional entry Base, ASCII string

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "URI", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	// Base, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, "URIdict", "Base", OPTIONAL, pdf.V10, nil)

	return err
}

func validateRootMetadata(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	return validateMetadata(xRefTable, rootDict, required, sinceVersion)
}

func validateMetadata(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 14.3 Metadata
	// In general, any PDF stream or dictionary may have metadata attached to it
	// as long as the stream or dictionary represents an actual information resource,
	// as opposed to serving as an implementation artifact.
	// Some PDF constructs are considered implementational, and hence may not have associated metadata.

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}

	sd, err := validateStreamDictEntry(xRefTable, d, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil || sd == nil {
		return err
	}

	dictName := "metaDataDict"

	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Metadata" })
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool { return s == "XML" })

	return err
}

func validateMarkInfo(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 14.7 Logical Structure

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "MarkInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	var isTaggedPDF bool

	dictName := "markInfoDict"

	// Marked, optional, boolean
	marked, err := validateBooleanEntry(xRefTable, d, dictName, "Marked", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}
	if marked != nil {
		isTaggedPDF = marked.Value()
	}

	// Suspects: optional, since V1.6, boolean
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V15
	}
	suspects, err := validateBooleanEntry(xRefTable, d, dictName, "Suspects", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if suspects != nil && suspects.Value() {
		isTaggedPDF = false
	}

	xRefTable.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "UserProperties", OPTIONAL, pdf.V16, nil)

	return err
}

func validateLang(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateStringEntry(xRefTable, rootDict, "rootDict", "Lang", required, sinceVersion, nil)

	return err
}

func validateCaptureCommandDictArray(xRefTable *pdf.XRefTable, a pdf.Array) error {

	for _, o := range a {

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateCaptureCommandDict(xRefTable, d)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateWebCaptureInfoDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "webCaptureInfoDict"

	// V, required, since V1.3, number
	_, err := validateNumberEntry(xRefTable, d, dictName, "V", REQUIRED, pdf.V13, nil)
	if err != nil {
		return err
	}

	// C, optional, since V1.3, array of web capture command dict indRefs
	a, err := validateIndRefArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	if a != nil {
		err = validateCaptureCommandDictArray(xRefTable, a)
	}

	return err
}

func validateSpiderInfo(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// 14.10.2 Web Capture Information Dictionary

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "SpiderInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureInfoDict(xRefTable, d)
}

func validateOutputIntentDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "outputIntentDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "OutputIntent" })
	if err != nil {
		return err
	}

	// S: required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// OutputCondition, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "OutputCondition", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// OutputConditionIdentifier, required, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "OutputConditionIdentifier", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// RegistryName, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "RegistryName", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Info, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Info", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// DestOutputProfile, optional, streamDict
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "DestOutputProfile", OPTIONAL, pdf.V10, nil)

	return err
}

func validateOutputIntents(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 14.11.5 Output Intents

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}

	a, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "OutputIntents", required, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}

	for _, o := range a {

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateOutputIntentDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validatePieceDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "pieceDict"

	for _, o := range d {

		d1, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d1 == nil {
			continue
		}

		required := REQUIRED
		if xRefTable.ValidationMode == pdf.ValidationRelaxed {
			required = OPTIONAL
		}
		_, err = validateDateEntry(xRefTable, d1, dictName, "LastModified", required, pdf.V10)
		if err != nil {
			return err
		}

		_, err = validateEntry(xRefTable, d1, dictName, "Private", OPTIONAL, pdf.V10)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRootPieceInfo(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validatePieceInfo(xRefTable, rootDict, "rootDict", "PieceInfo", required, sinceVersion)

	return err
}

func validatePieceInfo(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (hasPieceInfo bool, err error) {

	// 14.5 Page-Piece Dictionaries

	pieceDict, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || pieceDict == nil {
		return false, err
	}

	err = validatePieceDict(xRefTable, pieceDict)

	return hasPieceInfo, err
}

// TODO implement
func validatePermissions(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.8.4 Permissions

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Permissions", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return errors.New("pdfcpu: validatePermissions: not supported")
}

// TODO implement
func validateLegal(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.8.5 Legal Content Attestations

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Legal", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return errors.New("pdfcpu: validateLegal: not supported")
}

func validateRequirementDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "requirementDict"

	// Type, optional, name,
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Requirement" })
	if err != nil {
		return err
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "EnableJavaScripts" })
	if err != nil {
		return err
	}

	// The RH entry (requirement handler dicts) shall not be used in PDF 1.7.

	return nil
}

func validateRequirements(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.10 Document Requirements

	a, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "Requirements", required, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}

	for _, o := range a {

		d, err := xRefTable.DereferenceDict(o)
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

func validateCollectionFieldDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "colFlddict"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "CollectionField" })
	if err != nil {
		return err
	}

	// Subtype, required name
	validateCollectionFieldSubtype := func(s string) bool {
		return pdf.MemberOf(s, []string{"S", "D", "N", "F", "Desc", "ModDate", "CreationDate", "Size"})
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, pdf.V10, validateCollectionFieldSubtype)
	if err != nil {
		return err
	}

	// N, required text string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// O, optional integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "O", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// V, optional boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "V", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// E, optional boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "E", OPTIONAL, pdf.V10, nil)

	return err
}

func validateCollectionSchemaDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	for k, v := range d {

		if k == "Type" {

			var n pdf.Name
			n, err := xRefTable.DereferenceName(v, pdf.V10, nil)
			if err != nil {
				return err
			}

			if n != "CollectionSchema" {
				return errors.New("pdfcpu: validateCollectionSchemaDict: invalid entry \"Type\"")
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

func validateCollectionSortDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "colSortDict"

	// S, required name or array of names.
	err := validateNameOrArrayOfNameEntry(xRefTable, d, dictName, "S", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// A, optional boolean or array of booleans.
	err = validateBooleanOrArrayOfBooleanEntry(xRefTable, d, dictName, "A", OPTIONAL, pdf.V10)

	return err
}

func validateInitialView(s string) bool { return s == "D" || s == "T" || s == "H" }

func validateCollection(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.3.5 Collections

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Collection", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName := "Collection"

	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Collection" })
	if err != nil {
		return err
	}

	// Schema, optional dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "Schema", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateCollectionSchemaDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	// D, optional string
	_, err = validateStringEntry(xRefTable, d, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// View, optional name
	_, err = validateNameEntry(xRefTable, d, dictName, "View", OPTIONAL, sinceVersion, validateInitialView)
	if err != nil {
		return err
	}

	// Sort, optional dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "Sort", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateCollectionSortDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateNeedsRendering(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	_, err := validateBooleanEntry(xRefTable, rootDict, "rootDict", "NeedsRendering", required, sinceVersion, nil)

	return err
}

func logURIError(xRefTable *pdf.XRefTable, pages []int) {
	fmt.Println()
	for _, page := range pages {
		for uri, resp := range xRefTable.URIs[page] {
			if resp != "" {
				var s string
				switch resp {
				case "i":
					s = "invalid url"
				case "s":
					s = "severe error"
				default:
					s = fmt.Sprintf("status=%s", resp)
				}
				log.CLI.Printf("Page %d: %s %s\n", page, uri, s)
			}
		}
	}
}

func checkForBrokenLinks(xRefTable *pdf.XRefTable) error {
	var httpErr bool
	log.CLI.Println("validating URIs..")

	pages := []int{}
	for i := range xRefTable.URIs {
		pages = append(pages, i)
	}
	sort.Ints(pages)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	for _, page := range pages {
		for uri := range xRefTable.URIs[page] {
			if log.IsCLILoggerEnabled() {
				fmt.Printf(".")
			}
			_, err := url.ParseRequestURI(uri)
			if err != nil {
				httpErr = true
				xRefTable.URIs[page][uri] = "i"
				continue
			}
			res, err := client.Get(uri)
			if err != nil {
				httpErr = true
				xRefTable.URIs[page][uri] = "s"
				continue
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				httpErr = true
				xRefTable.URIs[page][uri] = strconv.Itoa(res.StatusCode)
				continue
			}
		}
	}

	if log.IsCLILoggerEnabled() {
		logURIError(xRefTable, pages)
	}

	if httpErr {
		return errors.New("broken links detected")
	}

	return nil
}

func validateRootObject(xRefTable *pdf.XRefTable) error {

	log.Validate.Println("*** validateRootObject begin ***")

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

	d, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	// Type
	_, err = validateNameEntry(xRefTable, d, "rootDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Catalog" })
	if err != nil {
		return err
	}

	// Pages
	rootPageNodeDict, err := validatePages(xRefTable, d)
	if err != nil {
		return err
	}

	for _, f := range []struct {
		validate     func(xRefTable *pdf.XRefTable, d pdf.Dict, required bool, sinceVersion pdf.Version) (err error)
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
		if !f.required && xRefTable.Version() < f.sinceVersion {
			// Ignore optional fields if currentVersion < sinceVersion
			// This is really a workaround for explicitly extending relaxed validation.
			continue
		}
		err = f.validate(xRefTable, d, f.required, f.sinceVersion)
		if err != nil {
			return err
		}
	}

	// Validate remainder of annotations after AcroForm validation only.
	_, err = validatePagesAnnotations(xRefTable, rootPageNodeDict, 0)

	if xRefTable.ValidateLinks && len(xRefTable.URIs) > 0 {
		err = checkForBrokenLinks(xRefTable)
	}

	if err == nil {
		log.Validate.Println("*** validateRootObject end ***")
	}

	return err
}

func validateAdditionalStreams(xRefTable *pdf.XRefTable) error {

	// Out of spec scope.
	return nil
}

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
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// XRefTable validates a PDF cross reference table obeying the validation mode.
func XRefTable(ctx *model.Context) error {
	if log.InfoEnabled() {
		log.Info.Println("validating")
	}
	if log.ValidateEnabled() {
		log.Validate.Println("*** validateXRefTable begin ***")
	}

	xRefTable := ctx.XRefTable

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	if err := validateRootVersion(xRefTable, rootDict, OPTIONAL, model.V14); err != nil {
		return err
	}

	metaDataAuthoritative, err := metaDataModifiedAfterInfoDict(xRefTable)
	if err != nil {
		return err
	}

	if metaDataAuthoritative {
		// if both info dict and catalog metadata present and metadata modification date after infodict modification date
		// validate document information dictionary before catalog metadata.
		err := validateDocumentInfoObject(xRefTable)
		if err != nil {
			return err
		}
	}

	// Validate root object(aka the document catalog) and page tree.
	err = validateRootObject(ctx, rootDict)
	if err != nil {
		return err
	}

	if !metaDataAuthoritative {
		// Validate document information dictionary after catalog metadata.
		err = validateDocumentInfoObject(xRefTable)
		if err != nil {
			return err
		}
	}

	// Validate offspec additional streams as declared in pdf trailer.
	// err = validateAdditionalStreams(xRefTable)
	// if err != nil {
	// 	return err
	// }

	xRefTable.Valid = true

	if xRefTable.CustomExtensions && log.CLIEnabled() {
		log.CLI.Println("Note: custom extensions will not be validated.")
	}

	if log.ValidateEnabled() {
		log.Validate.Println("*** validateXRefTable end ***")
	}

	return nil
}

func fixInfoDict(xRefTable *model.XRefTable, rootDict types.Dict) error {
	indRef := rootDict.IndirectRefEntry("Metadata")
	ok, err := model.EqualObjects(*indRef, *xRefTable.Info, xRefTable, nil)
	if err != nil {
		return err
	}
	if ok {
		// infoDict indRef falsely points to meta data.
		xRefTable.Info = nil
	}
	return nil
}

func metaDataModifiedAfterInfoDict(xRefTable *model.XRefTable) (bool, error) {
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return false, err
	}

	xmpMeta, err := catalogMetaData(xRefTable, rootDict, OPTIONAL, model.V14)
	if err != nil {
		return false, err
	}

	if xmpMeta != nil {
		xRefTable.CatalogXMPMeta = xmpMeta
		if xRefTable.Info != nil {
			if err := fixInfoDict(xRefTable, rootDict); err != nil {
				return false, err
			}
		}
	}

	if !(xmpMeta != nil && xRefTable.Info != nil) {
		return false, nil
	}

	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil {
		return false, err
	}
	if d == nil {
		return true, nil
	}

	modDate, ok := d["ModDate"]
	if !ok {
		return true, nil
	}

	modTimestampInfoDict, err := timeOfDateObject(xRefTable, modDate, model.V10)
	if err != nil {
		return false, err
	}
	if modTimestampInfoDict == nil {
		return true, nil
	}

	modTimestampMetaData := time.Time(xmpMeta.RDF.Description.ModDate)
	if modTimestampMetaData.IsZero() {
		//  xmlns:xap='http://ns.adobe.com/xap/1.0/ ...xap:ModifyDate='2006-06-05T21:58:13-05:00'></rdf:Description>
		//fmt.Println("metadata modificationDate is zero -> older than infodict")
		return false, nil
	}

	//fmt.Printf("infoDict: %s metaData: %s\n", modTimestampInfoDict, modTimestampMetaData)

	if *modTimestampInfoDict == modTimestampMetaData {
		return false, nil
	}

	infoDictOlderThanMetaDict := (*modTimestampInfoDict).Before(modTimestampMetaData)

	return infoDictOlderThanMetaDict, nil
}

func setRootVersion(xRefTable *model.XRefTable, s string) error {

	rootVersion, err := model.PDFVersion(s)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return errors.Wrapf(err, "identifyRootVersion: unknown PDF Root version: %s\n", s)
		}
		rootVersion, err = model.PDFVersionRelaxed(s)
		if err != nil {
			return errors.Wrapf(err, "identifyRootVersion: unknown PDF Root version: %s\n", s)
		}
	}

	xRefTable.RootVersion = &rootVersion

	// since V1.4 the header version may be overridden by a Version entry in the catalog.
	if *xRefTable.HeaderVersion < model.V14 {
		if log.InfoEnabled() {
			log.Info.Printf("identifyRootVersion: PDF version is %s - will ignore root version: %s\n", xRefTable.HeaderVersion, s)
		}
	}

	return nil
}

func validateRootVersion(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// Locate a possible Version entry (since V1.4) in the catalog
	// and record this as rootVersion (as opposed to headerVersion).

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	n, err := validateNameEntry(xRefTable, rootDict, "rootDict", "Version", required, sinceVersion, nil)
	if err == nil {
		if n != nil {
			// Validate version and save corresponding constant to xRefTable.
			rootVersionStr := n.Value()
			if err := setRootVersion(xRefTable, rootVersionStr); err != nil {
				return err
			}
		}
		return nil
	}

	if xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}

	f, err := validateNumberEntryToFloat(xRefTable, rootDict, "rootDict", "Version", OPTIONAL, sinceVersion, nil)
	if err != nil || f == 0 {
		return errors.New("invalid catalog version")
	}

	rootVersionStr := strconv.FormatFloat(f, 'f', 1, 64)
	if err := setRootVersion(xRefTable, rootVersionStr); err != nil {
		return err
	}

	model.ShowDigestedSpecViolation("catalog version with unexpected number type")

	return nil
}

func validateExtensions(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 7.12 Extensions Dictionary

	_, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Extensions", required, sinceVersion, nil)

	// No validation due to lack of documentation.

	return err
}

func validatePageLabels(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
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

	_, _, err = validateNumberTree(xRefTable, "PageLabel", d, true, false)

	return err
}

func validateNames(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 7.7.4 Name Dictionary

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Names", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	validateNameTreeName := func(s string) bool {
		return types.MemberOf(s, []string{"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
			"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"})
	}

	d1 := types.Dict{}

	for treeName, value := range d {

		if ok := validateNameTreeName(treeName); !ok {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return errors.Errorf("validateNames: unknown name tree name: %s\n", treeName)
			}
			continue
		}

		if xRefTable.Names[treeName] != nil {
			// Already internalized.
			continue
		}

		d, err := xRefTable.DereferenceDict(value)
		if err != nil {
			return err
		}
		if len(d) == 0 {
			continue
		}

		_, _, tree, err := validateNameTree(xRefTable, treeName, d, true)
		if err != nil {
			return err
		}

		if tree != nil && tree.Kmin != "" && tree.Kmax != "" {
			// Internalize.
			xRefTable.Names[treeName] = tree
			d1.Insert(treeName, value)
		}

	}

	delete(rootDict, "Names")
	if len(d1) > 0 {
		rootDict["Names"] = d1
	}

	return nil
}

func validateNamedDestinations(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) (err error) {
	// => 12.3.2.3 Named Destinations

	// indRef or dict with destination array values.

	xRefTable.Dests, err = validateDictEntry(xRefTable, rootDict, "rootDict", "Dests", required, sinceVersion, nil)
	if err != nil || xRefTable.Dests == nil {
		return err
	}

	for _, o := range xRefTable.Dests {
		if _, err = validateDestination(xRefTable, o, false); err != nil {
			return err
		}
	}

	return nil
}

func pageLayoutValidator(v model.Version) func(s string) bool {
	// "UseNone", "Continuous", "oneside", "useoutlines" is out of spec.
	layouts := []string{"SinglePage", "OneColumn", "TwoColumnLeft", "TwoColumnRight", "UseNone", "Continuous", "oneside", "TwoPageRight", "useoutlines"}
	if v >= model.V15 {
		layouts = append(layouts, "TwoPageLeft", "TwoPageRight")
	}
	validate := func(s string) bool {
		return types.MemberOf(s, layouts)
	}
	return validate
}

func validatePageLayout(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	n, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageLayout", required, sinceVersion, pageLayoutValidator(xRefTable.Version()))
	if err != nil {
		return err
	}

	if n != nil {
		xRefTable.PageLayout = model.PageLayoutFor(n.String())
	}

	return nil
}

func pageModeValidator(v model.Version) func(s string) bool {
	// "None", "none", "UserNone" are out of spec.
	modes := []string{"UseNone", "UseOutlines", "UseThumbs", "FullScreen", "None", "none", "UserNone"}
	if v >= model.V14 {
		modes = append(modes, "UseOC")
	}
	if v >= model.V16 {
		modes = append(modes, "UseAttachments")
	}
	return func(s string) bool { return types.MemberOf(s, modes) }
}

func validatePageMode(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	n, err := validateNameEntry(xRefTable, rootDict, "rootDict", "PageMode", required, sinceVersion, pageModeValidator(xRefTable.Version()))
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict || n == nil {
			return err
		}
		// Relax validation of "UseAttachments" before PDF v1.6.
		if *n != "UseAttachments" {
			return err
		}
	}

	if n != nil {
		xRefTable.PageMode = model.PageModeFor(n.String())
	}

	return nil
}

func validateOpenAction(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
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

	case types.Dict:
		err = validateActionDict(xRefTable, o)

	case types.Array:
		err = validateDestinationArray(xRefTable, o)

	default:
		err = errors.New("pdfcpu: validateOpenAction: unexpected object")
	}

	return err
}

func validateURI(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.6.4.7 URI Actions

	// URI dict with one optional entry Base, ASCII string

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "URI", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	// Base, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, "URIdict", "Base", OPTIONAL, model.V10, nil)

	return err
}

func validateMarkInfo(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.7 Logical Structure

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "MarkInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	var isTaggedPDF bool

	dictName := "markInfoDict"

	// Marked, optional, boolean
	marked, err := validateBooleanEntry(xRefTable, d, dictName, "Marked", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if marked != nil {
		isTaggedPDF = *marked
	}

	// Suspects: optional, since V1.6, boolean
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	suspects, err := validateBooleanEntry(xRefTable, d, dictName, "Suspects", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if suspects != nil && *suspects {
		isTaggedPDF = false
	}

	xRefTable.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "UserProperties", OPTIONAL, model.V16, nil)

	return err
}

func validateLang(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	_, err := validateStringEntry(xRefTable, rootDict, "rootDict", "Lang", required, sinceVersion, nil)
	return err
}

func validateCaptureCommandDictArray(xRefTable *model.XRefTable, a types.Array) error {
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

func validateWebCaptureInfoDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "webCaptureInfoDict"

	// V, required, since V1.3, number
	_, err := validateNumberEntry(xRefTable, d, dictName, "V", REQUIRED, model.V13, nil)
	if err != nil {
		return err
	}

	// C, optional, since V1.3, array of web capture command dict indRefs
	a, err := validateIndRefArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	if a != nil {
		err = validateCaptureCommandDictArray(xRefTable, a)
	}

	return err
}

func validateSpiderInfo(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// 14.10.2 Web Capture Information Dictionary

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "SpiderInfo", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureInfoDict(xRefTable, d)
}

func validateOutputIntentDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "outputIntentDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "OutputIntent" })
	if err != nil {
		return err
	}

	// S: required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// OutputCondition, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "OutputCondition", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// OutputConditionIdentifier, required, text string
	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "OutputConditionIdentifier", required, model.V10, nil)
	if err != nil {
		return err
	}

	// RegistryName, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "RegistryName", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Info, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Info", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// DestOutputProfile, optional, streamDict
	_, err = validateStreamDictEntry(xRefTable, d, dictName, "DestOutputProfile", OPTIONAL, model.V10, nil)

	return err
}

func validateOutputIntents(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.11.5 Output Intents

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
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

func validatePieceDict(xRefTable *model.XRefTable, d types.Dict) error {
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
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			required = OPTIONAL
		}
		_, err = validateDateEntry(xRefTable, d1, dictName, "LastModified", required, model.V10)
		if err != nil {
			return err
		}

		_, err = validateEntry(xRefTable, d1, dictName, "Private", OPTIONAL, model.V10)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRootPieceInfo(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		return nil
	}

	_, err := validatePieceInfo(xRefTable, rootDict, "rootDict", "PieceInfo", required, sinceVersion)

	return err
}

func validatePieceInfo(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) (hasPieceInfo bool, err error) {
	// 14.5 Page-Piece Dictionaries

	pieceDict, err := validateDictEntry(xRefTable, d, dictName, entryName, required, sinceVersion, nil)
	if err != nil || pieceDict == nil {
		return false, err
	}

	err = validatePieceDict(xRefTable, pieceDict)

	return hasPieceInfo, err
}

func validatePermissions(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.8.4 Permissions

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Perms", required, sinceVersion, nil)
	if err != nil {
		return err
	}
	if len(d) == 0 {
		return nil
	}

	i := 0

	if indRef := d.IndirectRefEntry("DocMDP"); indRef != nil {
		d1, err := xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}
		if len(d1) > 0 {
			xRefTable.CertifiedSigObjNr = indRef.ObjectNumber.Value()
			i++
		}
	}

	d1, err := validateDictEntry(xRefTable, d, "permDict", "UR3", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if len(d1) == 0 {
		return nil
	}

	xRefTable.URSignature = d1
	i++

	if i == 0 {
		return errors.New("pdfcpu: validatePermissions: unsupported permissions detected")
	}

	return nil
}

// TODO implement
func validateLegal(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.8.5 Legal Content Attestations

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Legal", required, sinceVersion, nil)
	if err != nil || len(d) == 0 {
		return err
	}

	return errors.New("pdfcpu: \"Legal\" not supported")
}

func validateRequirementDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
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

func validateRequirements(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
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

func validateCollectionFieldDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "colFlddict"

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "CollectionField" })
	if err != nil {
		return err
	}

	// Subtype, required name
	subTypes := []string{"S", "D", "N", "F", "Desc", "ModDate", "CreationDate", "Size"}

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		// See i659.pdf
		subTypes = append(subTypes, "AFRelationship")
		subTypes = append(subTypes, "CompressedSize")
	}

	validateCollectionFieldSubtype := func(s string) bool {
		return types.MemberOf(s, subTypes)
	}
	_, err = validateNameEntry(xRefTable, d, dictName, "Subtype", REQUIRED, model.V10, validateCollectionFieldSubtype)
	if err != nil {
		return err
	}

	// N, required text string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// O, optional integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "O", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// V, optional boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "V", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// E, optional boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "E", OPTIONAL, model.V10, nil)

	return err
}

func validateCollectionSchemaDict(xRefTable *model.XRefTable, d types.Dict) error {
	for k, v := range d {

		if k == "Type" {

			var n types.Name
			n, err := xRefTable.DereferenceName(v, model.V10, nil)
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

func validateCollectionSortDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "colSortDict"

	// S, required name or array of names.
	err := validateNameOrArrayOfNameEntry(xRefTable, d, dictName, "S", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// A, optional boolean or array of booleans.
	err = validateBooleanOrArrayOfBooleanEntry(xRefTable, d, dictName, "A", OPTIONAL, model.V10)

	return err
}

func validateInitialView(s string) bool { return s == "D" || s == "T" || s == "H" || s == "C" }

func validateCollection(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
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

func validateNeedsRendering(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	_, err := validateBooleanEntry(xRefTable, rootDict, "rootDict", "NeedsRendering", required, sinceVersion, nil)
	return err
}

func validateDSS(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.8.4.3 Document Security Store

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "DSS", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	xRefTable.DSS = d

	return nil
}

func validateAF(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.13 Associated Files

	a, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "AF", required, sinceVersion, nil)
	if err != nil || len(a) == 0 {
		return err
	}

	return errors.New("pdfcpu: PDF2.0 \"AF\" not supported")
}

func validateDPartRoot(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.12 Document Parts

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "DPartRoot", required, sinceVersion, nil)
	if err != nil || len(d) == 0 {
		return err
	}

	return errors.New("pdfcpu: PDF2.0 \"DPartRoot\" not supported")
}

func logURIError(xRefTable *model.XRefTable, pages []int) {
	if log.CLIEnabled() {
		log.CLI.Println()
	}
	for _, page := range pages {
		for uri, resp := range xRefTable.URIs[page] {
			if resp != "" {
				var s string
				switch resp {
				case "i":
					s = "invalid url"
				case "s":
					s = "severe error"
				case "t":
					s = "timeout"
				default:
					s = fmt.Sprintf("status=%s", resp)
				}
				if log.CLIEnabled() {
					log.CLI.Printf("Page %d: %s - %s\n", page, uri, s)
				}
			}
		}
	}
}

func checkLinks(xRefTable *model.XRefTable, client http.Client, pages []int) bool {
	var httpErr bool
	for _, page := range pages {
		for uri := range xRefTable.URIs[page] {
			if log.CLIEnabled() {
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
				if e, ok := err.(net.Error); ok && e.Timeout() {
					xRefTable.URIs[page][uri] = "t"
				} else {
					xRefTable.URIs[page][uri] = "s"
				}
				httpErr = true
				continue
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				httpErr = true
				xRefTable.URIs[page][uri] = strconv.Itoa(res.StatusCode)
				continue
			}
		}
	}
	return httpErr
}

func checkForBrokenLinks(ctx *model.Context) error {
	if !ctx.XRefTable.ValidateLinks {
		return nil
	}
	if len(ctx.URIs) > 0 {
		if ctx.Offline {
			if log.CLIEnabled() {
				log.CLI.Printf("pdfcpu is offline, can't validate Links")
			}
			return nil
		}
	}

	if log.CLIEnabled() {
		log.CLI.Println("validating URIs..")
	}

	xRefTable := ctx.XRefTable

	pages := []int{}
	for i := range xRefTable.URIs {
		pages = append(pages, i)
	}
	sort.Ints(pages)

	client := http.Client{
		Timeout: time.Duration(ctx.Timeout) * time.Second,
	}

	httpErr := checkLinks(xRefTable, client, pages)

	if log.CLIEnabled() {
		logURIError(xRefTable, pages)
	}

	if httpErr {
		return errors.New("broken links detected")
	}

	return nil
}

func validateRootObject(ctx *model.Context, rootDict types.Dict) error {
	if log.ValidateEnabled() {
		log.Validate.Println("*** validateRootObject begin ***")
	}

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

	// DSS					y	2.0			dict			=> 12.8.4.3 Document Security Store	TODO
	// AF					y	2.0			array of dicts	=> 14.3 Associated Files			TODO
	// DPartRoot			y	2.0			dict			=> 14.12 Document parts				TODO

	xRefTable := ctx.XRefTable

	// Type
	required := true
	if ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err := validateNameEntry(xRefTable, rootDict, "rootDict", "Type", required, model.V10, func(s string) bool { return s == "Catalog" })
	if err != nil {
		return err
	}

	// Pages
	rootPageNodeDict, err := validatePages(xRefTable, rootDict)
	if err != nil {
		return err
	}

	for _, f := range []struct {
		validate     func(xRefTable *model.XRefTable, d types.Dict, required bool, sinceVersion model.Version) (err error)
		required     bool
		sinceVersion model.Version
	}{
		//{validateRootVersion, OPTIONAL, model.V14}, Note: moved up
		{validateExtensions, OPTIONAL, model.V10},
		{validatePageLabels, OPTIONAL, model.V13},
		{validateNames, OPTIONAL, model.V11}, //model.V12},
		{validateNamedDestinations, OPTIONAL, model.V11},
		{validateViewerPreferences, OPTIONAL, model.V12},
		{validatePageLayout, OPTIONAL, model.V10},
		{validatePageMode, OPTIONAL, model.V10},
		{validateOutlines, OPTIONAL, model.V10},
		{validateThreads, OPTIONAL, model.V11},
		{validateOpenAction, OPTIONAL, model.V11},
		{validateRootAdditionalActions, OPTIONAL, model.V14},
		{validateURI, OPTIONAL, model.V11},
		{validateForm, OPTIONAL, model.V12},
		{validateRootMetadata, OPTIONAL, model.V14},
		{validateStructTree, OPTIONAL, model.V13},
		{validateMarkInfo, OPTIONAL, model.V14},
		{validateLang, OPTIONAL, model.V10},
		{validateSpiderInfo, OPTIONAL, model.V13},
		{validateOutputIntents, OPTIONAL, model.V14},
		{validateRootPieceInfo, OPTIONAL, model.V14},
		{validateOCProperties, OPTIONAL, model.V15},
		{validatePermissions, OPTIONAL, model.V15},
		{validateLegal, OPTIONAL, model.V17},
		{validateRequirements, OPTIONAL, model.V17},
		{validateCollection, OPTIONAL, model.V17},
		{validateNeedsRendering, OPTIONAL, model.V17},
		{validateDSS, OPTIONAL, model.V17},
		{validateAF, OPTIONAL, model.V20},
		{validateDPartRoot, OPTIONAL, model.V20},
	} {
		if !f.required && xRefTable.Version() < f.sinceVersion {
			// Ignore optional fields if currentVersion < sinceVersion
			// This is really a workaround for explicitly extending relaxed validation.
			continue
		}
		err = f.validate(xRefTable, rootDict, f.required, f.sinceVersion)
		if err != nil {
			return err
		}
	}

	// Validate remainder of annotations after AcroForm validation only.
	if _, err = validatePagesAnnotations(xRefTable, rootPageNodeDict, 0); err != nil {
		return err
	}

	// Validate form fields against page annotations.
	if xRefTable.Form != nil {
		if err := validateFormFieldsAgainstPageAnnotations(xRefTable); err != nil {
			return err
		}
	}

	// Validate links.
	if err = checkForBrokenLinks(ctx); err == nil {
		if log.ValidateEnabled() {
			log.Validate.Println("*** validateRootObject end ***")
		}
	}

	return err
}

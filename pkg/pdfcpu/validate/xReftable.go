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
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateXRefTableContext(ctx *model.Context) error {
	if ctx == nil {
		return model.ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return model.ErrMissingXRefTable
	}
	return nil
}

// XRefTable validates a PDF cross reference table obeying the validation mode.
func XRefTable(ctx *model.Context) error {
	if log.InfoEnabled() {
		log.Info.Println("validating")
	}
	if log.ValidateEnabled() {
		log.Validate.Println("*** validateXRefTable begin ***")
	}
	if err := validateXRefTableContext(ctx); err != nil {
		return err
	}

	xRefTable := ctx.XRefTable

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		err = fmt.Errorf("load catalog: %w", err)
		return model.WithValidationErrorObject(err, validationRootObjectNumber(xRefTable))
	}

	if err := validateRootVersion(xRefTable, rootDict, OPTIONAL, model.V14); err != nil {
		err = fmt.Errorf("catalog version: %w", err)
		return model.WithValidationErrorObject(err, validationRootObjectNumber(xRefTable))
	}

	metaDataAuthoritative, err := metaDataModifiedAfterInfoDict(xRefTable)
	if err != nil {
		return fmt.Errorf("metadata/info order: %w", err)
	}

	if metaDataAuthoritative {
		// if both info dict and catalog metadata present and metadata modification date after infodict modification date
		// validate document information dictionary before catalog metadata.
		err := validateDocumentInfoObject(xRefTable)
		if err != nil {
			return fmt.Errorf("document info: %w", err)
		}
	}

	// Validate root object(aka the document catalog) and page tree.
	err = validateRootObject(ctx, rootDict)
	if err != nil {
		return fmt.Errorf("catalog: %w", err)
	}

	if !metaDataAuthoritative {
		// Validate document information dictionary after catalog metadata.
		err = validateDocumentInfoObject(xRefTable)
		if err != nil {
			return fmt.Errorf("document info: %w", err)
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

	infoObjNr := xRefTable.Info.ObjectNumber.Value()
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil {
		return false, model.WithValidationErrorObject(err, infoObjNr)
	}
	if d == nil {
		return true, nil
	}

	modDate, ok := d["ModDate"]
	if !ok {
		return true, nil
	}

	modTimestampInfoDict, err := timeOfDateObject(xRefTable, modDate, infoObjNr, model.V10)
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

	if (*modTimestampInfoDict).Equal(modTimestampMetaData) {
		return false, nil
	}

	infoDictOlderThanMetaDict := (*modTimestampInfoDict).Before(modTimestampMetaData)

	return infoDictOlderThanMetaDict, nil
}

func setRootVersion(xRefTable *model.XRefTable, s string) error {
	rootVersion, err := model.PDFVersion(s)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("unknown root version %s: %w", s, err)
		}
		rootVersion, err = model.PDFVersionRelaxed(s)
		if err != nil {
			return fmt.Errorf("unknown root version %s: %w", s, err)
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

func validateRootVersion(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) (err error) {
	rootObjNr := validationRootObjectNumber(xRefTable)
	defer func() {
		err = model.WithValidationErrorObject(err, rootObjNr)
	}()

	// Locate a possible Version entry (since V1.4) in the catalog
	// and record this as rootVersion (as opposed to headerVersion).

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}
	n, err := validateNameEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Version", required, sinceVersion, nil,
	)
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
	if err != nil {
		return err
	}
	if f == 0 {
		err = errors.New("invalid version")
		return model.WithValidationErrorObject(err, validationEntryObjectNumber(rootObjNr, rootDict, "Version"))
	}

	rootVersionStr := strconv.FormatFloat(f, 'f', 1, 64)
	if err := setRootVersion(xRefTable, rootVersionStr); err != nil {
		return err
	}

	model.ShowDigestedSpecViolation("catalog version with unexpected number type")

	return nil
}

func validateExtensionsDirectObject(xRefTable *model.XRefTable, o types.Object, path string, depth int) error {
	if err := xRefTable.CheckRecursionDepth("Extensions dictionary", depth); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	switch o := o.(type) {
	case types.IndirectRef:
		return fmt.Errorf("%s: indirect reference not permitted", path)

	case types.Array:
		for i, o := range o {
			path := fmt.Sprintf("%s array index %d", path, i)
			if err := validateExtensionsDirectObject(xRefTable, o, path, depth+1); err != nil {
				return err
			}
		}

	case types.Dict:
		keys := make([]string, 0, len(o))
		for key := range o {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			path := fmt.Sprintf("%s key %s", path, key)
			if err := validateExtensionsDirectObject(xRefTable, o[key], path, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateExtensions(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 7.12 Extensions Dictionary
	rootObjNr := validationRootObjectNumber(xRefTable)
	o, _, err := rootDict.Entry("rootDict", "Extensions", required)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, rootObjNr)
	}

	if _, ok := o.(types.IndirectRef); ok {
		err := fmt.Errorf("catalog obj#%d entry Extensions: value must be direct", rootObjNr)
		return model.WithValidationErrorObject(err, rootObjNr)
	}

	if err := xRefTable.ValidateVersion("dict=rootDict entry=Extensions", sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, rootObjNr)
	}

	d, ok := o.(types.Dict)
	if !ok {
		err := fmt.Errorf("dict=rootDict entry=Extensions invalid type %T", o)
		return model.WithValidationErrorObject(err, rootObjNr)
	}

	path := fmt.Sprintf("catalog obj#%d entry Extensions", rootObjNr)
	if err := validateExtensionsDirectObject(xRefTable, d, path, 0); err != nil {
		return model.WithValidationErrorObject(err, rootObjNr)
	}

	return nil
}

func validatePageLabels(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// optional since PDF 1.3
	// => 7.9.7 Number Trees, 12.4.2 Page Labels

	// Dict or indirect ref to Dict

	ir := rootDict.IndirectRefEntry("PageLabels")
	if ir == nil {
		if required {
			err := errors.New("dict=rootDict required entry=PageLabels missing")
			return model.WithValidationErrorObject(err, validationRootObjectNumber(xRefTable))
		}
		return nil
	}

	dictName := "PageLabels"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}

	_, _, err = validateNumberTree(xRefTable, "PageLabel", d, ir.ObjectNumber.Value(), true, false)

	return err
}

func validateNames(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 7.7.4 Name Dictionary

	rootObjNr := validationRootObjectNumber(xRefTable)
	namesObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "Names")
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Names", required, sinceVersion, nil,
	)
	if err != nil || d == nil {
		return err
	}

	validateNameTreeName := func(s string) bool {
		return types.MemberOf(s, []string{"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
			"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"})
	}

	d1 := types.Dict{}

	for _, treeName := range slices.Sorted(maps.Keys(d)) {
		value := d[treeName]
		treeObjNr := validationObjectNumber(namesObjNr, value)

		if ok := validateNameTreeName(treeName); !ok {
			if xRefTable.ValidationMode == model.ValidationStrict {
				err = fmt.Errorf("name tree: unknown name %s", treeName)
				return model.WithValidationErrorObject(err, namesObjNr)
			}
			continue
		}

		if xRefTable.Names[treeName] != nil {
			// Already internalized.
			continue
		}

		d, err := xRefTable.DereferenceDict(value)
		if err != nil {
			return model.WithValidationErrorObject(err, treeObjNr)
		}
		if len(d) == 0 {
			continue
		}

		_, _, tree, err := validateNameTree(xRefTable, treeName, d, treeObjNr, true)
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

	rootObjNr := validationRootObjectNumber(xRefTable)
	destsObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "Dests")
	xRefTable.Dests, err = validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Dests", required, sinceVersion, nil,
	)
	if err != nil || xRefTable.Dests == nil {
		return err
	}

	for _, key := range slices.Sorted(maps.Keys(xRefTable.Dests)) {
		o := xRefTable.Dests[key]
		if _, err = validateDestination(xRefTable, o, destsObjNr, false); err != nil {
			err = fmt.Errorf("named destination %s: %w", key, err)
			return model.WithValidationErrorObject(err, validationObjectNumber(destsObjNr, o))
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
	n, err := validateNameEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "PageLayout", required, sinceVersion,
		pageLayoutValidator(xRefTable.Version()),
	)
	if err != nil {
		return err
	}

	if n != nil {
		xRefTable.PageLayout = model.PageLayoutFor(n.String())
	}

	return nil
}

func nonStandardPageMode(s string) bool {
	return types.MemberOf(s, []string{"None", "none", "UserNone", `"None"`})
}

func pageModeValidator(v model.Version, relaxed bool) func(s string) bool {
	modes := []string{"UseNone", "UseOutlines", "UseThumbs", "FullScreen"}
	if v >= model.V14 {
		modes = append(modes, "UseOC")
	}
	if v >= model.V16 {
		modes = append(modes, "UseAttachments")
	}
	return func(s string) bool {
		return types.MemberOf(s, modes) || (relaxed && nonStandardPageMode(s))
	}
}

func validatePageMode(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	validate := pageModeValidator(xRefTable.Version(), relaxed)
	n, err := validateNameEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "PageMode", required, sinceVersion,
		validate,
	)
	if err != nil {
		if !relaxed || n == nil {
			return err
		}
		// Relax validation of "UseAttachments" before PDF v1.6.
		if *n != "UseAttachments" {
			return err
		}
	}

	if n != nil {
		pageMode := n.String()
		if nonStandardPageMode(pageMode) {
			xRefTable.PageMode = model.PageModeFor("UseNone")
			model.ShowDigestedSpecViolation("dict=rootDict entry=PageMode invalid dict entry: " + pageMode)
			return nil
		}
		xRefTable.PageMode = model.PageModeFor(pageMode)
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

	rawOpenAction, _ := rootDict.Find("OpenAction")
	o, err := validateEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "OpenAction", required, sinceVersion,
	)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Dict:
		err = validateActionDictObject(xRefTable, o, rawOpenAction, "rootDict.OpenAction")

	case types.Array:
		err = validateDestinationArray(
			xRefTable, o, validationObjectNumber(validationRootObjectNumber(xRefTable), rawOpenAction),
		)
		if err != nil {
			err = fmt.Errorf("rootDict.OpenAction: %w", err)
		}

	default:
		err = fmt.Errorf("rootDict entry=OpenAction expected dict or array, got %T", o)
	}

	return model.WithValidationErrorObject(
		err, validationObjectNumber(validationRootObjectNumber(xRefTable), rawOpenAction),
	)
}

func validateURI(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.6.4.7 URI Actions

	// URI dict with one optional entry Base, ASCII string

	rootObjNr := validationRootObjectNumber(xRefTable)
	uriObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "URI")
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "URI", required, sinceVersion, nil,
	)
	if err != nil || d == nil {
		return err
	}

	// Base, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, uriObjNr, "URIdict", "Base", OPTIONAL, model.V10, nil)

	return err
}

func validateMarkInfo(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.7 Logical Structure

	rootObjNr := validationRootObjectNumber(xRefTable)
	markInfoObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "MarkInfo")
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "MarkInfo", required, sinceVersion, nil,
	)
	if err != nil || d == nil {
		return err
	}

	var isTaggedPDF bool

	dictName := "markInfoDict"

	// Marked, optional, boolean
	marked, err := validateBooleanEntry(
		xRefTable, d, markInfoObjNr, dictName, "Marked", OPTIONAL, model.V10, nil,
	)
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
	suspects, err := validateBooleanEntry(
		xRefTable, d, markInfoObjNr, dictName, "Suspects", OPTIONAL, sinceVersion, nil,
	)
	if err != nil {
		return err
	}

	if suspects != nil && *suspects {
		isTaggedPDF = false
	}

	xRefTable.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, err = validateBooleanEntry(
		xRefTable, d, markInfoObjNr, dictName, "UserProperties", OPTIONAL, model.V16, nil,
	)

	return err
}

func validateLang(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	_, err := validateStringEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "Lang", required, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("rootDict.Lang: %w", err)
	}
	return nil
}

func validateCaptureCommandDictArray(xRefTable *model.XRefTable, a types.Array, ownerObjNr int) error {
	for i, o := range a {
		commandObjNr := validationObjectNumber(ownerObjNr, o)

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference capture command dict: %w", objectContext(fmt.Sprintf("webCaptureInfoDict.C[%d]", i), o), err)
			return model.WithValidationErrorObject(err, commandObjNr)
		}

		if d == nil {
			continue
		}

		err = validateCaptureCommandDict(xRefTable, d, 0)
		if err != nil {
			err = fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("webCaptureInfoDict.C[%d]", i), o), err)
			return model.WithValidationErrorObject(err, commandObjNr)
		}

	}

	return nil
}

func validateWebCaptureInfoDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "webCaptureInfoDict"

	// V, required, since V1.3, number
	_, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "V", REQUIRED, model.V13, nil)
	if err != nil {
		return fmt.Errorf("%s.V: %w", dictName, err)
	}

	// C, optional, since V1.3, array of web capture command dict indRefs
	a, err := validateIndRefArrayEntry(xRefTable, d, 0, dictName, "C", OPTIONAL, model.V13, nil)
	if err != nil {
		err = fmt.Errorf("%s.C: %w", dictName, err)
		return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "C"))
	}

	if a != nil {
		err = validateCaptureCommandDictArray(
			xRefTable, a, validationEntryObjectNumber(ownerObjNr, d, "C"),
		)
	}

	return err
}

func validateSpiderInfo(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// 14.10.2 Web Capture Information Dictionary

	rawEntry := rootDict["SpiderInfo"]
	rootObjNr := validationRootObjectNumber(xRefTable)
	spiderInfoObjNr := validationObjectNumber(rootObjNr, rawEntry)
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "SpiderInfo", required, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext("rootDict", "SpiderInfo", rawEntry), err)
	}
	if d == nil {
		return nil
	}

	if err := validateWebCaptureInfoDict(xRefTable, d, spiderInfoObjNr); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext("rootDict", "SpiderInfo", rawEntry), err)
	}
	return nil
}

func validateOutputIntentDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "outputIntentDict"

	// Type, optional, name
	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, model.V10,
		func(s string) bool { return s == "OutputIntent" },
	)
	if err != nil {
		return err
	}

	// S: required, name
	required := REQUIRED
	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	if relaxed {
		required = OPTIONAL
	}
	s, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "S", required, model.V10, nil)
	if err != nil {
		return err
	}

	// OutputCondition, optional, text string
	_, err = validateStringEntry(
		xRefTable, d, ownerObjNr, dictName, "OutputCondition", OPTIONAL, model.V10, nil,
	)
	if err != nil {
		return err
	}

	// OutputConditionIdentifier, required, text string
	required = REQUIRED
	if relaxed {
		required = OPTIONAL
	}
	_, err = validateStringEntry(
		xRefTable, d, ownerObjNr, dictName, "OutputConditionIdentifier", required, model.V10, nil,
	)
	if err != nil {
		return err
	}

	// RegistryName, optional, text string
	_, err = validateStringEntry(
		xRefTable, d, ownerObjNr, dictName, "RegistryName", OPTIONAL, model.V10, nil,
	)
	if err != nil {
		return err
	}

	// Info, optional, text string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Info", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// DestOutputProfile, optional, streamDict
	if _, err = validateStreamDictEntry(
		xRefTable, d, ownerObjNr, dictName, "DestOutputProfile", OPTIONAL, model.V10, nil,
	); err != nil {
		return err
	}

	if s == nil && relaxed {
		model.ShowDigestedSpecViolation("dict=" + dictName + " required entry=S missing")
	}

	return nil
}

func validateOutputIntents(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.11.5 Output Intents

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}

	rootObjNr := validationRootObjectNumber(xRefTable)
	outputIntentsObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "OutputIntents")
	a, err := validateArrayEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "OutputIntents", required, sinceVersion, nil,
	)
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("rootDict.OutputIntents: %w", err)
		}
		return nil
	}

	for i, o := range a {
		outputIntentObjNr := validationObjectNumber(outputIntentsObjNr, o)

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference output intent dict: %w", objectContext(fmt.Sprintf("rootDict.OutputIntents[%d]", i), o), err)
			return model.WithValidationErrorObject(err, outputIntentObjNr)
		}

		if d == nil {
			continue
		}

		err = validateOutputIntentDict(xRefTable, d, outputIntentObjNr)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("rootDict.OutputIntents[%d]", i), o), err)
		}
	}

	return nil
}

func validatePieceDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "pieceDict"

	for _, name := range slices.Sorted(maps.Keys(d)) {
		o := d[name]
		pieceObjNr := validationObjectNumber(ownerObjNr, o)

		d1, err := xRefTable.DereferenceDict(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference dict: %w", objectContext(fmt.Sprintf("%s.%s", dictName, name), o), err)
			return model.WithValidationErrorObject(err, pieceObjNr)
		}

		if d1 == nil {
			continue
		}

		required := REQUIRED
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			required = OPTIONAL
		}
		_, err = validateDateEntry(xRefTable, d1, pieceObjNr, dictName, "LastModified", required, model.V10)
		if err != nil {
			err = fmt.Errorf("%s: LastModified: %w", objectContext(fmt.Sprintf("%s.%s", dictName, name), o), err)
			return model.WithValidationErrorObject(
				err, validationEntryObjectNumber(pieceObjNr, d1, "LastModified"),
			)
		}

		_, err = validateEntry(xRefTable, d1, pieceObjNr, dictName, "Private", OPTIONAL, model.V10)
		if err != nil {
			return fmt.Errorf("%s: Private: %w", objectContext(fmt.Sprintf("%s.%s", dictName, name), o), err)
		}

	}

	return nil
}

func validateRootPieceInfo(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		return nil
	}

	_, err := validatePieceInfo(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "PieceInfo", required, sinceVersion,
	)

	return err
}

func validatePieceInfo(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (hasPieceInfo bool, err error) {
	// 14.5 Page-Piece Dictionaries

	rawEntry := d[entryName]
	pieceInfoObjNr := validationObjectNumber(ownerObjNr, rawEntry)
	defer func() {
		err = model.WithValidationErrorObject(err, pieceInfoObjNr)
	}()

	pieceDict, err := validateDictEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, nil,
	)
	if err != nil {
		return false, fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}
	if pieceDict == nil {
		return false, nil
	}

	err = validatePieceDict(xRefTable, pieceDict, pieceInfoObjNr)
	if err != nil {
		return true, fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}

	return true, nil
}

func validatePermissions(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.8.4 Permissions

	rawEntry := rootDict["Perms"]
	rootObjNr := validationRootObjectNumber(xRefTable)
	permsObjNr := validationObjectNumber(rootObjNr, rawEntry)
	context := dictEntryContext("rootDict", "Perms", rawEntry)
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Perms", required, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	if len(d) == 0 {
		return nil
	}
	permsIncrement := indirectObjectIncrement(xRefTable, rawEntry, 0)

	i := 0

	if indRef := d.IndirectRefEntry("DocMDP"); indRef != nil {
		docMDPObjNr := indRef.ObjectNumber.Value()
		d1, err := xRefTable.DereferenceDict(*indRef)
		if err != nil {
			err = fmt.Errorf("%s: permDict.DocMDP obj#%d: dereference dict: %w", context, docMDPObjNr, err)
			return model.WithValidationErrorObject(err, docMDPObjNr)
		}
		if len(d1) > 0 {
			xRefTable.CertifiedSigObjNr = indRef.ObjectNumber.Value()
			i++
		}
	}

	rawUR3 := d["UR3"]
	d1, err := validateDictEntry(
		xRefTable, d, permsObjNr, "permDict", "UR3", OPTIONAL, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: permDict.UR3: %w", context, err)
	}
	if len(d1) == 0 {
		return nil
	}

	xRefTable.URSignature = d1
	xRefTable.URSignatureIncrement = indirectObjectIncrement(xRefTable, rawUR3, permsIncrement)
	i++

	if i == 0 {
		err = fmt.Errorf("%s: permDict: unsupported entries", context)
		return model.WithValidationErrorObject(err, permsObjNr)
	}

	return nil
}

func indirectObjectIncrement(
	xRefTable *model.XRefTable,
	obj types.Object,
	fallback int,
) int {
	indRef, ok := obj.(types.IndirectRef)
	if !ok {
		return fallback
	}
	entry, found := xRefTable.FindTableEntryForIndRef(&indRef)
	if !found {
		return fallback
	}
	return entry.Incr
}

// TODO implement
func validateLegal(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.8.5 Legal Content Attestations

	rawEntry := rootDict["Legal"]
	rootObjNr := validationRootObjectNumber(xRefTable)
	legalObjNr := validationObjectNumber(rootObjNr, rawEntry)
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Legal", required, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext("rootDict", "Legal", rawEntry), err)
	}
	if len(d) == 0 {
		return nil
	}

	return model.WithValidationErrorObject(errors.New("rootDict.Legal: not supported"), legalObjNr)
}

func validateRequirementDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "requirementDict"

	// Type, optional, name,
	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, sinceVersion,
		func(s string) bool { return s == "Requirement" },
	)
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	// S, required, name
	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "S", REQUIRED, sinceVersion,
		func(s string) bool { return s == "EnableJavaScripts" },
	)
	if err != nil {
		return fmt.Errorf("%s.S: %w", dictName, err)
	}

	// The RH entry (requirement handler dicts) shall not be used in PDF 1.7.

	return nil
}

func validateRequirements(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.10 Document Requirements

	rootObjNr := validationRootObjectNumber(xRefTable)
	requirementsObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "Requirements")
	a, err := validateArrayEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Requirements", required, sinceVersion, nil,
	)
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("rootDict.Requirements: %w", err)
		}
		return nil
	}

	for i, o := range a {
		requirementObjNr := validationObjectNumber(requirementsObjNr, o)

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference requirement dict: %w", objectContext(fmt.Sprintf("rootDict.Requirements[%d]", i), o), err)
			return model.WithValidationErrorObject(err, requirementObjNr)
		}

		if d == nil {
			continue
		}

		err = validateRequirementDict(xRefTable, d, requirementObjNr, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("rootDict.Requirements[%d]", i), o), err)
		}

	}

	return nil
}

func validateCollectionFieldDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "colFlddict"

	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, model.V10,
		func(s string) bool { return s == "CollectionField" },
	)
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
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
	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "Subtype", REQUIRED, model.V10, validateCollectionFieldSubtype,
	)
	if err != nil {
		return fmt.Errorf("%s.Subtype: %w", dictName, err)
	}

	// N, required text string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "N", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.N: %w", dictName, err)
	}

	// O, optional integer
	_, err = validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "O", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.O: %w", dictName, err)
	}

	// V, optional boolean
	_, err = validateBooleanEntry(xRefTable, d, ownerObjNr, dictName, "V", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.V: %w", dictName, err)
	}

	// E, optional boolean
	_, err = validateBooleanEntry(xRefTable, d, ownerObjNr, dictName, "E", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.E: %w", dictName, err)
	}

	return nil
}

func validateCollectionSchemaDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	for _, k := range slices.Sorted(maps.Keys(d)) {
		v := d[k]
		entryObjNr := validationObjectNumber(ownerObjNr, v)

		if k == "Type" {

			var n types.Name
			n, err := xRefTable.DereferenceName(v, model.V10, nil)
			if err != nil {
				err = fmt.Errorf("Collection.Schema.Type: dereference name: %w", err)
				return model.WithValidationErrorObject(err, entryObjNr)
			}

			if n != "CollectionSchema" {
				return model.WithValidationErrorObject(
					errors.New("Collection.Schema.Type: invalid value"), entryObjNr,
				)
			}

			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s: dereference collection field dict: %w", objectContext(fmt.Sprintf("Collection.Schema.%s", k), v), err)
			return model.WithValidationErrorObject(err, entryObjNr)
		}

		if d == nil {
			continue
		}

		err = validateCollectionFieldDict(xRefTable, d, entryObjNr)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("Collection.Schema.%s", k), v), err)
		}

	}

	return nil
}

func validateCollectionSortDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "colSortDict"

	// S, required name or array of names.
	err = validateNameOrArrayOfNameEntry(xRefTable, d, ownerObjNr, dictName, "S", REQUIRED, model.V10)
	if err != nil {
		err = fmt.Errorf("%s.S: %w", dictName, err)
		return err
	}

	// A, optional boolean or array of booleans.
	err = validateBooleanOrArrayOfBooleanEntry(xRefTable, d, ownerObjNr, dictName, "A", OPTIONAL, model.V10)
	if err != nil {
		err = fmt.Errorf("%s.A: %w", dictName, err)
		return err
	}

	return nil
}

func validateInitialView(s string) bool { return s == "D" || s == "T" || s == "H" || s == "C" }

func validateCollection(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.3.5 Collections

	rawEntry := rootDict["Collection"]
	rootObjNr := validationRootObjectNumber(xRefTable)
	collectionObjNr := validationObjectNumber(rootObjNr, rawEntry)
	context := dictEntryContext("rootDict", "Collection", rawEntry)
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Collection", required, sinceVersion, nil,
	)
	if err != nil || d == nil {
		if err != nil {
			return fmt.Errorf("%s: %w", context, err)
		}
		return nil
	}

	dictName := "Collection"

	_, err = validateNameEntry(
		xRefTable, d, collectionObjNr, dictName, "Type", OPTIONAL, sinceVersion,
		func(s string) bool { return s == "Collection" },
	)
	if err != nil {
		return fmt.Errorf("%s: %s.Type: %w", context, dictName, err)
	}

	// Schema, optional dict
	schemaObjNr := validationEntryObjectNumber(collectionObjNr, d, "Schema")
	d1, err := validateDictEntry(
		xRefTable, d, collectionObjNr, dictName, "Schema", OPTIONAL, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %s.Schema: %w", context, dictName, err)
	}
	if d1 != nil {
		err = validateCollectionSchemaDict(xRefTable, d1, schemaObjNr)
		if err != nil {
			return fmt.Errorf("%s: %s.Schema: %w", context, dictName, err)
		}
	}

	// D, optional string
	_, err = validateStringEntry(
		xRefTable, d, collectionObjNr, dictName, "D", OPTIONAL, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %s.D: %w", context, dictName, err)
	}

	// View, optional name
	_, err = validateNameEntry(
		xRefTable, d, collectionObjNr, dictName, "View", OPTIONAL, sinceVersion, validateInitialView,
	)
	if err != nil {
		return fmt.Errorf("%s: %s.View: %w", context, dictName, err)
	}

	// Sort, optional dict
	sortObjNr := validationEntryObjectNumber(collectionObjNr, d, "Sort")
	d1, err = validateDictEntry(
		xRefTable, d, collectionObjNr, dictName, "Sort", OPTIONAL, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s: %s.Sort: %w", context, dictName, err)
	}
	if d1 != nil {
		err = validateCollectionSortDict(xRefTable, d1, sortObjNr)
		if err != nil {
			return fmt.Errorf("%s: %s.Sort: %w", context, dictName, err)
		}
	}

	return nil
}

func validateNeedsRendering(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	_, err := validateBooleanEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "NeedsRendering", required, sinceVersion, nil,
	)
	return err
}

func validateDSS(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.8.4.3 Document Security Store

	d, err := validateDictEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "DSS", required, sinceVersion, nil,
	)
	if err != nil || d == nil {
		return err
	}

	xRefTable.DSS = d

	return nil
}

func validateAF(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.13 Associated Files

	rootObjNr := validationRootObjectNumber(xRefTable)
	afObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "AF")
	a, err := validateArrayEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "AF", required, sinceVersion, nil,
	)
	if err != nil || len(a) == 0 {
		return err
	}

	return model.WithValidationErrorObject(errors.New("PDF 2.0 associated files not supported"), afObjNr)
}

func validateDPartRoot(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.12 Document Parts

	rootObjNr := validationRootObjectNumber(xRefTable)
	dPartRootObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "DPartRoot")
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "DPartRoot", required, sinceVersion, nil,
	)
	if err != nil || len(d) == 0 {
		return err
	}

	return model.WithValidationErrorObject(errors.New("PDF 2.0 document parts not supported"), dPartRootObjNr)
}

func logURIError(xRefTable *model.XRefTable, pages []int) {
	if log.CLIEnabled() {
		log.CLI.Println()
	}
	for _, page := range pages {
		for _, uri := range slices.Sorted(maps.Keys(xRefTable.URIs[page])) {
			resp := xRefTable.URIs[page][uri]
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
		for _, uri := range slices.Sorted(maps.Keys(xRefTable.URIs[page])) {
			if log.CLIEnabled() {
				log.CLI.Print(".")
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

func validateRootObject(ctx *model.Context, rootDict types.Dict) (err error) {
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
	rootObjNr := validationRootObjectNumber(xRefTable)
	defer func() {
		err = model.WithValidationErrorObject(err, rootObjNr)
	}()

	// Type
	required := true
	if ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
		required = false
	}
	_, err = validateNameEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "Type", required, model.V10,
		func(s string) bool { return s == "Catalog" },
	)
	if err != nil {
		return err
	}

	// Pages
	rootPageNodeDict, err := validatePages(xRefTable, rootDict)
	if err != nil {
		return fmt.Errorf("pages: %w", err)
	}

	for _, f := range []struct {
		name         string
		validate     func(xRefTable *model.XRefTable, d types.Dict, required bool, sinceVersion model.Version) (err error)
		required     bool
		sinceVersion model.Version
	}{
		//{validateRootVersion, OPTIONAL, model.V14}, Note: moved up
		{"Extensions", validateExtensions, OPTIONAL, model.V17},
		{"PageLabels", validatePageLabels, OPTIONAL, model.V13},
		{"Names", validateNames, OPTIONAL, model.V11}, //model.V12},
		{"Dests", validateNamedDestinations, OPTIONAL, model.V11},
		{"ViewerPreferences", validateViewerPreferences, OPTIONAL, model.V12},
		{"PageLayout", validatePageLayout, OPTIONAL, model.V10},
		{"PageMode", validatePageMode, OPTIONAL, model.V10},
		{"Outlines", validateOutlines, OPTIONAL, model.V10},
		{"Threads", validateThreads, OPTIONAL, model.V11},
		{"OpenAction", validateOpenAction, OPTIONAL, model.V11},
		{"AA", validateRootAdditionalActions, OPTIONAL, model.V14},
		{"URI", validateURI, OPTIONAL, model.V11},
		{"AcroForm", validateForm, OPTIONAL, model.V12},
		{"Metadata", validateRootMetadata, OPTIONAL, model.V14},
		{"StructTreeRoot", validateStructTree, OPTIONAL, model.V13},
		{"MarkInfo", validateMarkInfo, OPTIONAL, model.V14},
		{"Lang", validateLang, OPTIONAL, model.V10},
		{"SpiderInfo", validateSpiderInfo, OPTIONAL, model.V13},
		{"OutputIntents", validateOutputIntents, OPTIONAL, model.V14},
		{"PieceInfo", validateRootPieceInfo, OPTIONAL, model.V14},
		{"OCProperties", validateOCProperties, OPTIONAL, model.V15},
		{"Perms", validatePermissions, OPTIONAL, model.V15},
		{"Legal", validateLegal, OPTIONAL, model.V17},
		{"Requirements", validateRequirements, OPTIONAL, model.V17},
		{"Collection", validateCollection, OPTIONAL, model.V17},
		{"NeedsRendering", validateNeedsRendering, OPTIONAL, model.V17},
		{"DSS", validateDSS, OPTIONAL, model.V17},
		{"AF", validateAF, OPTIONAL, model.V20},
		{"DPartRoot", validateDPartRoot, OPTIONAL, model.V20},
	} {
		if !f.required && xRefTable.Version() < f.sinceVersion {
			// Ignore optional fields if currentVersion < sinceVersion
			// This is really a workaround for explicitly extending relaxed validation.
			continue
		}
		err = f.validate(xRefTable, rootDict, f.required, f.sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s: %w", f.name, err)
			return model.WithValidationErrorObject(
				err, validationEntryObjectNumber(rootObjNr, rootDict, f.name),
			)
		}
	}

	// Validate remainder of annotations after AcroForm validation only.
	if _, err = validatePagesAnnotations(xRefTable, rootPageNodeDict, 0); err != nil {
		return fmt.Errorf("page annotations: %w", err)
	}

	// Validate form fields against page annotations.
	if xRefTable.Form != nil {
		if err := validateFormFieldsAgainstPageAnnotations(xRefTable); err != nil {
			return fmt.Errorf("form fields/page annotations: %w", err)
		}
	}

	// Validate links.
	if err = checkForBrokenLinks(ctx); err == nil {
		if log.ValidateEnabled() {
			log.Validate.Println("*** validateRootObject end ***")
		}
	}

	if err != nil {
		return fmt.Errorf("uri link check: %w", err)
	}

	return nil
}

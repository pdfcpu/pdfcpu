package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdflib/types"
)

func validateDestsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateDestsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateDestsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	err = validateDestination(xRefTable, obj)

	logInfoValidate.Println("*** validateDestsNameTreeValue: end ***")

	return
}

func validateAPNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("validateAPNameTreeValue: begin")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAPNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	err = validateAppearanceDict(xRefTable, obj)

	logInfoValidate.Println("validateAPNameTreeValue: end")

	return
}

func validateJavaScriptNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateJavaScriptNameTreeValue: begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	// Javascript Action:
	err = validateJavaScriptActionDict(xRefTable, dict, sinceVersion)

	logInfoValidate.Println("*** validateJavaScriptNameTreeValue: end ***")

	return
}

func validatePagesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.6

	logInfoValidate.Println("*** validatePagesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validatePagesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a page dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Page" })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePagesNameTreeValue: end ***")

	return
}

func validateTemplatesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 12.7.6

	logInfoValidate.Printf("*** validateTemplatesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateTemplatesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a template dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "templateDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Template" })
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateTemplatesNameTreeValue: end ***")

	return
}

func validateURLAliasDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** validateURLAliasDict: begin ***")

	dictName := "urlAliasDict"

	// U, required, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateURLAliasDict: end ***")

	return
}

func validateCommandInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** validateCommandInfoDict: begin ***")

	dictName := "cmdInfoDict"

	// URL, required, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "URL", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "L", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CT", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateCommandInfoDict: end ***")

	return
}

func validateSourceInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** validateSourceInfoDict: begin ***")

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	obj, found := dict.Find("AU")
	if !found || obj == nil {
		return errors.New("validateSourceInfoDict: missing required entry \"AU\"")
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return errors.New("validateSourceInfoDict: corrupt entry \"AU\"")
	}

	if obj == nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral, types.PDFHexLiteral:
		// no further processing

	case types.PDFDict:
		err = validateURLAliasDict(xRefTable, &obj)
		if err != nil {
			return
		}

	default:
		return errors.New("validateSourceInfoDict: entry \"AU\" must be string or dict")

	}

	// TS, optional, date
	_, err = validateDateEntry(xRefTable, dict, dictName, "TS", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// E, optional, date
	_, err = validateDateEntry(xRefTable, dict, dictName, "E", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// S, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return
	}

	// C, optional, indRef of command dict
	indRef, err := validateIndRefEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	if indRef != nil {

		var d *types.PDFDict

		d, err = xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return
		}

		err = validateCommandInfoDict(xRefTable, d)
		if err != nil {
			return
		}

	}

	logInfoValidate.Printf("*** validateSourceInfoDict: end ***")

	return
}

func validateEntrySI(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 14.10.5, table 355, source information dictionary

	logInfoValidate.Printf("*** validateEntrySI: begin ***")

	obj, found := dict.Find("SI")
	if !found {
		if required {
			return errors.New("")
		}
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		return errors.New("validateEntrySI: obj is nil")
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateSourceInfoDict(xRefTable, &obj)
		if err != nil {
			return
		}

	case types.PDFArray:

		var d *types.PDFDict

		for _, v := range obj {

			if v == nil {
				continue
			}

			d, err = xRefTable.DereferenceDict(v)
			if err != nil {
				return
			}

			err = validateSourceInfoDict(xRefTable, d)
			if err != nil {
				return
			}

		}

	}

	logInfoValidate.Printf("*** validateEntrySI: end ***")

	return
}

func validateWebCaptureContentSetDict(XRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 14.10.4

	logInfoValidate.Printf("*** validateWebCaptureContentSetDict: begin ***")

	dictName := "webCaptureContentSetDict"

	// Type, optional, name
	_, err = validateNameEntry(XRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "SpiderContentSet" })
	if err != nil {
		return
	}

	// S, required, name
	s, err := validateNameEntry(XRefTable, dict, dictName, "Type", REQUIRED, types.V10, func(s string) bool { return s == "SPS" || s == "SIS" })
	if err != nil {
		return
	}

	// ID, required, byte string
	_, err = validateStringEntry(XRefTable, dict, dictName, "ID", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// O, required, array of indirect references.
	_, err = validateIndRefArrayEntry(XRefTable, dict, dictName, "O", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// SI, required, source info dict or array of source info dicts
	err = validateEntrySI(XRefTable, dict, REQUIRED, types.V10)
	if err != nil {
		return
	}

	// CT, optional, string
	_, err = validateStringEntry(XRefTable, dict, dictName, "CT", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// TS, optional, date
	_, err = validateDateEntry(XRefTable, dict, dictName, "TS", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// spider page set
	if *s == "SPS" {

		// T, optional, string
		_, err = validateStringEntry(XRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}

		// TID, optional, byte string
		_, err = validateStringEntry(XRefTable, dict, dictName, "TID", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}
	}

	// spider image set
	if *s == "SIS" {

		// R, required, integer or array of integers
		err = validateIntegerOrArrayOfIntegerEntry(XRefTable, dict, dictName, "R", REQUIRED, types.V10)
		if err != nil {
			return
		}

	}

	logInfoValidate.Printf("*** validateWebCaptureContentSetDict: end ***")

	return
}

func validateIDSNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 14.10.4

	logInfoValidate.Printf("*** validateIDSNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIDSNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if d == nil {
		return errors.New("validateIDSNameTreeValue: value is nil")
	}

	err = validateWebCaptureContentSetDict(xRefTable, d)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateIDSTreeValue: end ***")

	return
}

func validateURLSNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 14.10.4

	logInfoValidate.Println("*** validateURLSNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateURLSNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if d == nil {
		return errors.New("validateURLSNameTreeValue: value is nil")
	}

	err = validateWebCaptureContentSetDict(xRefTable, d)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateURLSNameTreeValue: end ***")

	return
}

func validateEmbeddedFilesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 7.11.4

	logInfoValidate.Printf("*** validateEmbeddedFilesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateEmbeddedFilesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a file specification for an embedded file stream.

	if obj == nil {
		logInfoValidate.Println("validateEmbeddedFilesNameTreeValue: is nil.")
		return
	}

	_, err = validateFileSpecification(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateEmbeddedFilesNameTreeValue: end ***")

	return
}

func validateSlideShowDict(XRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.5, table 297

	logInfoValidate.Printf("*** validateSlideShowDict: begin ***")

	dictName := "slideShowDict"

	// Type, required, name, since V1.4
	_, err = validateNameEntry(XRefTable, dict, dictName, "Type", REQUIRED, types.V14, func(s string) bool { return s == "SlideShow" })
	if err != nil {
		return
	}

	// Subtype, required, name, since V1.4
	_, err = validateNameEntry(XRefTable, dict, dictName, "Subtype", REQUIRED, types.V14, func(s string) bool { return s == "Embedded" })
	if err != nil {
		return
	}

	// Resources, required, name tree, since V1.4
	// Note: This is really an array of (string,indRef) pairs.
	_, err = validateArrayEntry(XRefTable, dict, dictName, "Resources", REQUIRED, types.V14, nil)
	if err != nil {
		return
	}

	// StartResource, required, byte string, since V1.4
	_, err = validateStringEntry(XRefTable, dict, dictName, "StartResource", REQUIRED, types.V14, nil)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateSlideShowDict: end ***")

	return
}

func validateAlternatePresentationsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 13.5

	logInfoValidate.Println("*** validateAlternatePresentationsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateAlternatePresentationsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a slide show dict.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict != nil {
		err = validateSlideShowDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateAlternatePresentationsNameTreeValue: end ***")

	return
}

func validateRenditionsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 13.2.3

	logInfoValidate.Println("*** validateRenditionsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateRenditionsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a rendition object.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict != nil {
		err = validateRenditionDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateRenditionsNameTreeValue: end ***")

	return
}

func validateIDTreeValue(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateIDTreeValue: begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateIDTreeValue: is nil.")
		return
	}

	dictType := dict.Type()
	if dictType == nil || *dictType == "StructElem" {
		err = validateStructElementDict(xRefTable, dict)
		if err != nil {
			return
		}
	} else {
		return errors.Errorf("validateIDTreeValue: invalid dictType %s (should be \"StructElem\")\n", *dictType)
	}

	logInfoValidate.Println("*** validateIDTreeValue: end ***")

	return
}

func validateNameTreeDictNamesEntry(xRefTable *types.XRefTable, dict *types.PDFDict, name string) (err error) {

	logInfoValidate.Printf("*** validateNameTreeDictNamesEntry begin: name:%s ***\n", name)

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	obj, found := dict.Find("Names")
	if !found {
		return errors.Errorf("validateNameTreeDictNamesEntry: missing \"Kids\" or \"Names\" entry.")
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if arr == nil {
		return errors.Errorf("validateNameTreeDictNamesEntry: missing \"Names\" array.")
	}

	logInfoValidate.Println("validateNameTreeDictNamesEntry: \"Nums\": now writing value objects")

	// arr length needs to be even because of contained key value pairs.
	if len(*arr)%2 == 1 {
		return errors.Errorf("validateNameTreeDictNamesEntry: Names array entry length needs to be even, length=%d\n", len(*arr))
	}

	for i, obj := range *arr {

		if i%2 == 0 {
			continue
		}

		logDebugValidate.Printf("validateNameTreeDictNamesEntry: Nums array value: %v\n", obj)

		switch name {

		case "Dests":
			err = validateDestsNameTreeValue(xRefTable, obj, types.V12)

		case "AP":
			err = validateAPNameTreeValue(xRefTable, obj, types.V13)

		case "JavaScript":
			err = validateJavaScriptNameTreeValue(xRefTable, obj, types.V13)

		case "Pages":
			err = validatePagesNameTreeValue(xRefTable, obj, types.V13)

		case "Templates":
			err = validateTemplatesNameTreeValue(xRefTable, obj, types.V13)

		case "IDS":
			err = validateIDSNameTreeValue(xRefTable, obj, types.V13)

		case "URLS":
			err = validateURLSNameTreeValue(xRefTable, obj, types.V13)

		case "EmbeddedFiles":
			err = validateEmbeddedFilesNameTreeValue(xRefTable, obj, types.V14)

		case "AlternatePresentations":
			err = validateAlternatePresentationsNameTreeValue(xRefTable, obj, types.V14)

		case "Renditions":
			err = validateRenditionsNameTreeValue(xRefTable, obj, types.V15)

		case "IDTree":
			// for structure tree root
			err = validateIDTreeValue(xRefTable, obj)

		default:
			err = errors.Errorf("validateNameTreeDictNamesEntry: unknown dict name: %s", name)

		}

	}

	logInfoValidate.Println("*** validateNameTreeDictNamesEntry end ***")

	return
}

func validateNameTreeDictLimitsEntry(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateNameTreeDictLimitsEntry begin ***")

	// An array of two integers, that shall specify the
	// numerically least and greatest keys included in the "Nums" array.

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if arr == nil {
		return errors.New("validateNameTreeDictLimitsEntry: missing \"Limits\" array")
	}

	if len(*arr) != 2 {
		return errors.New("validateNameTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	if _, ok := (*arr)[0].(types.PDFStringLiteral); !ok {
		return errors.New("validateNameTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	if _, ok := (*arr)[1].(types.PDFStringLiteral); !ok {
		return errors.New("validateNameTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	logInfoValidate.Println("*** validateNameTreeDictLimitsEntry end ***")

	return
}

func validateNameTree(xRefTable *types.XRefTable, name string, indRef types.PDFIndirectRef, root bool) (err error) {

	// see 7.7.4

	logInfoValidate.Printf("*** validateNameTree: %s ***\n", name)

	dict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || dict == nil {
		return
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		arr, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}

		if arr == nil {
			return errors.New("validateNameTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			logInfoValidate.Printf("validateNameTree: processing kid: %v\n", obj)

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return errors.New("validateNameTree: corrupt kid, should be indirect reference")
			}

			err = validateNameTree(xRefTable, name, kid, false)
			if err != nil {
				return err
			}
		}

		logInfoValidate.Printf("validateNameTree end: %s\n", name)

		return nil
	}

	err = validateNameTreeDictNamesEntry(xRefTable, dict, name)
	if err != nil {
		return
	}

	if !root {

		_, err = validateStringArrayEntry(xRefTable, dict, "nameTreeDict", "Limits", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateNameTree end ***")

	return
}

package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdfcpu/types"
)

func validateDestsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateDestsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateDestsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err := validateDestination(xRefTable, obj)

	logInfoValidate.Println("*** validateDestsNameTreeValue: end ***")

	return err
}

func validateAPNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("validateAPNameTreeValue: begin")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateAPNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err := validateAppearanceDict(xRefTable, obj)

	logInfoValidate.Println("validateAPNameTreeValue: end")

	return err
}

func validateJavaScriptNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateJavaScriptNameTreeValue: begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	// Javascript Action:
	err = validateJavaScriptActionDict(xRefTable, dict, "JavaScript", sinceVersion)

	logInfoValidate.Println("*** validateJavaScriptNameTreeValue: end ***")

	return err
}

func validatePagesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 12.7.6

	logInfoValidate.Println("*** validatePagesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validatePagesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a page dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Page" })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validatePagesNameTreeValue: end ***")

	return nil
}

func validateTemplatesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 12.7.6

	logInfoValidate.Printf("*** validateTemplatesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateTemplatesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a template dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "templateDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Template" })
	if err != nil {
		return err
	}

	logInfoValidate.Printf("*** validateTemplatesNameTreeValue: end ***")

	return nil
}

func validateURLAliasDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Printf("*** validateURLAliasDict: begin ***")

	dictName := "urlAliasDict"

	// U, required, ASCII string
	_, err := validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Printf("*** validateURLAliasDict: end ***")

	return nil
}

func validateCommandSettingsDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 14.10.5.4

	dictName := "cmdSettingsDict"

	// G, optional, dict
	_, err := validateDictEntry(xRefTable, dict, dictName, "G", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateCaptureCommandDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Printf("*** validateCaptureCommandDict: begin ***")

	dictName := "captureCommandDict"

	// URL, required, string
	_, err := validateStringEntry(xRefTable, dict, dictName, "URL", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "L", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CT", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, command settings dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, nil)
	if d != nil {
		err = validateCommandSettingsDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Printf("*** validateCaptureCommandDict: end ***")

	return nil
}

func validateSourceInfoDictEntryAU(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	obj, found := dict.Find("AU")
	if !found || obj == nil {
		return errors.New("validateSourceInfoDict: missing required entry \"AU\"")
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral, types.PDFHexLiteral:
		// no further processing

	case types.PDFDict:
		err = validateURLAliasDict(xRefTable, &obj)
		if err != nil {
			return err
		}

	default:
		return errors.New("validateSourceInfoDict: entry \"AU\" must be string or dict")

	}

	return nil
}

func validateSourceInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Printf("*** validateSourceInfoDict: begin ***")

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	err := validateSourceInfoDictEntryAU(xRefTable, dict)
	if err != nil {
		return err
	}

	// E, optional, date
	_, err = validateDateEntry(xRefTable, dict, dictName, "E", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// S, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// C, optional, indRef of command dict
	indRef, err := validateIndRefEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	if indRef != nil {

		d, err := xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}

		err = validateCaptureCommandDict(xRefTable, d)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Printf("*** validateSourceInfoDict: end ***")

	return nil
}

func validateEntrySI(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) error {

	// see 14.10.5, table 355, source information dictionary

	logInfoValidate.Printf("*** validateEntrySI: begin ***")

	obj, found := dict.Find("SI")
	if !found {
		if required {
			return errors.New("")
		}
		return nil
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		return errors.New("validateEntrySI: obj is nil")
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateSourceInfoDict(xRefTable, &obj)
		if err != nil {
			return err
		}

	case types.PDFArray:

		for _, v := range obj {

			if v == nil {
				continue
			}

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			err = validateSourceInfoDict(xRefTable, d)
			if err != nil {
				return err
			}

		}

	}

	logInfoValidate.Printf("*** validateEntrySI: end ***")

	return nil
}

func validateWebCaptureContentSetDict(XRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 14.10.4

	logInfoValidate.Printf("*** validateWebCaptureContentSetDict: begin ***")

	dictName := "webCaptureContentSetDict"

	// Type, optional, name
	_, err := validateNameEntry(XRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "SpiderContentSet" })
	if err != nil {
		return err
	}

	// S, required, name
	s, err := validateNameEntry(XRefTable, dict, dictName, "Type", REQUIRED, types.V10, func(s string) bool { return s == "SPS" || s == "SIS" })
	if err != nil {
		return err
	}

	// ID, required, byte string
	_, err = validateStringEntry(XRefTable, dict, dictName, "ID", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// O, required, array of indirect references.
	_, err = validateIndRefArrayEntry(XRefTable, dict, dictName, "O", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// SI, required, source info dict or array of source info dicts
	err = validateEntrySI(XRefTable, dict, REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// CT, optional, string
	_, err = validateStringEntry(XRefTable, dict, dictName, "CT", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// TS, optional, date
	_, err = validateDateEntry(XRefTable, dict, dictName, "TS", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// spider page set
	if *s == "SPS" {

		// T, optional, string
		_, err = validateStringEntry(XRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}

		// TID, optional, byte string
		_, err = validateStringEntry(XRefTable, dict, dictName, "TID", OPTIONAL, types.V10, nil)
		if err != nil {
			return err
		}
	}

	// spider image set
	if *s == "SIS" {

		// R, required, integer or array of integers
		err = validateIntegerOrArrayOfIntegerEntry(XRefTable, dict, dictName, "R", REQUIRED, types.V10)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Printf("*** validateWebCaptureContentSetDict: end ***")

	return nil
}

func validateIDSNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 14.10.4

	logInfoValidate.Printf("*** validateIDSNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateIDSNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("validateIDSNameTreeValue: value is nil")
	}

	err = validateWebCaptureContentSetDict(xRefTable, d)
	if err != nil {
		return err
	}

	logInfoValidate.Printf("*** validateIDSTreeValue: end ***")

	return nil
}

func validateURLSNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 14.10.4

	logInfoValidate.Println("*** validateURLSNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateURLSNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("validateURLSNameTreeValue: value is nil")
	}

	err = validateWebCaptureContentSetDict(xRefTable, d)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateURLSNameTreeValue: end ***")

	return nil
}

func validateEmbeddedFilesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 7.11.4

	logInfoValidate.Printf("*** validateEmbeddedFilesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateEmbeddedFilesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a file specification for an embedded file stream.

	if obj == nil {
		logInfoValidate.Println("validateEmbeddedFilesNameTreeValue: is nil.")
		return nil
	}

	_, err := validateFileSpecification(xRefTable, obj)
	if err != nil {
		return err
	}

	logInfoValidate.Printf("*** validateEmbeddedFilesNameTreeValue: end ***")

	return nil
}

func validateSlideShowDict(XRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.5, table 297

	logInfoValidate.Printf("*** validateSlideShowDict: begin ***")

	dictName := "slideShowDict"

	// Type, required, name, since V1.4
	_, err := validateNameEntry(XRefTable, dict, dictName, "Type", REQUIRED, types.V14, func(s string) bool { return s == "SlideShow" })
	if err != nil {
		return err
	}

	// Subtype, required, name, since V1.4
	_, err = validateNameEntry(XRefTable, dict, dictName, "Subtype", REQUIRED, types.V14, func(s string) bool { return s == "Embedded" })
	if err != nil {
		return err
	}

	// Resources, required, name tree, since V1.4
	// Note: This is really an array of (string,indRef) pairs.
	_, err = validateArrayEntry(XRefTable, dict, dictName, "Resources", REQUIRED, types.V14, nil)
	if err != nil {
		return err
	}

	// StartResource, required, byte string, since V1.4
	_, err = validateStringEntry(XRefTable, dict, dictName, "StartResource", REQUIRED, types.V14, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Printf("*** validateSlideShowDict: end ***")

	return nil
}

func validateAlternatePresentationsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 13.5

	logInfoValidate.Println("*** validateAlternatePresentationsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateAlternatePresentationsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a slide show dict.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if dict != nil {
		err = validateSlideShowDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateAlternatePresentationsNameTreeValue: end ***")

	return nil
}

func validateRenditionsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	// see 13.2.3

	logInfoValidate.Println("*** validateRenditionsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateRenditionsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Value is a rendition object.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if dict != nil {
		err = validateRenditionDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateRenditionsNameTreeValue: end ***")

	return nil
}

func validateIDTreeValue(xRefTable *types.XRefTable, obj interface{}) error {

	logInfoValidate.Println("*** validateIDTreeValue: begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if dict == nil {
		logInfoValidate.Println("validateIDTreeValue: is nil.")
		return nil
	}

	dictType := dict.Type()
	if dictType == nil || *dictType == "StructElem" {
		err = validateStructElementDict(xRefTable, dict)
		if err != nil {
			return err
		}
	} else {
		return errors.Errorf("validateIDTreeValue: invalid dictType %s (should be \"StructElem\")\n", *dictType)
	}

	logInfoValidate.Println("*** validateIDTreeValue: end ***")

	return nil
}

func validateNameTreeByName(name string, xRefTable *types.XRefTable, obj interface{}) (err error) {

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
		return errors.Errorf("validateNameTreeDictNamesEntry: unknown dict name: %s", name)

	}

	return err
}

func validateNameTreeDictNamesEntry(xRefTable *types.XRefTable, dict *types.PDFDict, name string) (firstKey, lastKey string, err error) {

	logInfoValidate.Printf("*** validateNameTreeDictNamesEntry begin: name:%s ***\n", name)

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	obj, found := dict.Find("Names")
	if !found {
		return "", "", errors.Errorf("validateNameTreeDictNamesEntry: missing \"Kids\" or \"Names\" entry.")
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return "", "", err
	}
	if arr == nil {
		return "", "", errors.Errorf("validateNameTreeDictNamesEntry: missing \"Names\" array.")
	}

	logInfoValidate.Println("validateNameTreeDictNamesEntry: \"Nums\": now writing value objects")

	// arr length needs to be even because of contained key value pairs.
	if len(*arr)%2 == 1 {
		return "", "", errors.Errorf("validateNameTreeDictNamesEntry: Names array entry length needs to be even, length=%d\n", len(*arr))
	}

	for i, obj := range *arr {

		if i%2 == 0 {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return "", "", err
			}

			s, ok := obj.(types.PDFStringLiteral)
			if !ok {
				return "", "", errors.Errorf("validateNameTreeDictNamesEntry: corrupt key <%v>\n", obj)
			}

			if firstKey == "" {
				firstKey = s.Value()
			}

			lastKey = s.Value()

			continue
		}

		logDebugValidate.Printf("validateNameTreeDictNamesEntry: Nums array value: %v\n", obj)

		err = validateNameTreeByName(name, xRefTable, obj)
		if err != nil {
			return "", "", err
		}

	}

	logInfoValidate.Println("*** validateNameTreeDictNamesEntry end ***")

	return firstKey, lastKey, nil
}

func validateNameTreeDictLimitsEntry(xRefTable *types.XRefTable, dict *types.PDFDict, firstKey, lastKey string) error {

	var arr *types.PDFArray

	arr, err := validateStringArrayEntry(xRefTable, dict, "nameTreeDict", "Limits", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	fk, _ := (*arr)[0].(types.PDFStringLiteral)
	lk, _ := (*arr)[1].(types.PDFStringLiteral)

	if firstKey != fk.Value() || lastKey != lk.Value() {
		return errors.Errorf("validateNameTreeDictLimitsEntry: leaf node corrupted\n")
	}

	return nil
}

func validateNameTree(xRefTable *types.XRefTable, name string, indRef types.PDFIndirectRef, root bool) (firstKey, lastKey string, err error) {

	// see 7.7.4

	logInfoValidate.Printf("*** validateNameTree: %s obj#%d***\n", name, indRef.ObjectNumber)

	// A node has "Kids" or "Names" entry.

	var dict *types.PDFDict

	dict, err = xRefTable.DereferenceDict(indRef)
	if err != nil || dict == nil {
		return
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		// Intermediate node

		var arr *types.PDFArray

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return "", "", err
		}

		if arr == nil {
			return "", "", errors.New("validateNameTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			logInfoValidate.Printf("validateNameTree: processing kid: %v\n", obj)

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return "", "", errors.New("validateNameTree: corrupt kid, should be indirect reference")
			}

			var fk string
			fk, lastKey, err = validateNameTree(xRefTable, name, kid, false)
			if err != nil {
				return "", "", err
			}
			if firstKey == "" {
				firstKey = fk
			}
		}

	} else {

		// Leaf node
		firstKey, lastKey, err = validateNameTreeDictNamesEntry(xRefTable, dict, name)
		if err != nil {
			return "", "", err
		}
	}

	if !root {

		err = validateNameTreeDictLimitsEntry(xRefTable, dict, firstKey, lastKey)
		if err != nil {
			return "", "", err
		}

	}

	logInfoValidate.Println("*** validateNameTree end ***")

	return firstKey, lastKey, nil
}

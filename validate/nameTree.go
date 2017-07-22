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

// TODO implement
func validateIDSNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 14.10.4

	logInfoValidate.Printf("*** validateIDSNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIDSNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a web capture content set.

	err = errors.New("*** validateIDSNAmeTreeValue: unsupported ***")

	logInfoValidate.Printf("*** validateIDSTreeValue: end ***")

	return
}

// TODO implement
func validateURLSNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 14.10.4

	logInfoValidate.Println("*** validateURLSNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateURLSNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a web capture content set.

	err = errors.New("*** validateURLSNameTreeValue: unsupported ***")

	logInfoValidate.Println("*** validateURLSNameTreeValue: end ***")

	return
}

// TODO implement
func validateEmbeddedFilesNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 7.11.4

	logInfoValidate.Printf("*** validateEmbeddedFilesNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateEmbeddedFilesNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a file specification.

	err = errors.New("*** validateEmbeddedFilesNameTreeValue: unsupported ***")

	logInfoValidate.Printf("*** validateEmbeddedFilesNameTreeValue: end ***")

	return
}

// TODO implement
func validateSlideShowDict(XRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	return errors.New("*** validateSlideShowDict: unsupported ***")
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

// TODO implement
func validateRenditionsNameTreeValue(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 13.2.3

	logInfoValidate.Println("*** validateRenditionsNameTreeValue: begin ***")

	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateRenditionsNameTreeValue: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	// Value is a rendition object.

	err = errors.New("*** validateEmbeddedFilesNameTreeValue: unsupported ***")

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

	// Names: array of the form [key1 value1 key2 value2 ... keyn valuen]
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
			err = errors.Errorf("validateNameTreeDictNamesEntry: unknow dict name: %s", name)

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

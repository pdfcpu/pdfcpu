package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdfcpu/types"
)

func validateDestsNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	return validateDestination(xRefTable, obj)
}

func validateAPNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	return validateAppearanceDict(xRefTable, obj)
}

func validateJavaScriptNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	// Javascript Action:
	return validateJavaScriptActionDict(xRefTable, dict, "JavaScript")
}

func validatePagesNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 12.7.6

	// Value is a page dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateTemplatesNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 12.7.6

	// Value is a template dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "templateDict", "Type", REQUIRED, types.V10, func(s string) bool { return s == "Template" })

	return err
}

func validateURLAliasDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "urlAliasDict"

	// U, required, ASCII string
	_, err := validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)

	return err
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

	return err
}

func validateCaptureCommandDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

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

	return nil
}

func validateSourceInfoDictEntryAU(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
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

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	err := validateSourceInfoDictEntryAU(xRefTable, dict, dictName, "AU", REQUIRED, types.V10)
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

	return nil
}

func validateEntrySI(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) error {

	// see 14.10.5, table 355, source information dictionary

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
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

	return nil
}

func validateWebCaptureContentSetDict(XRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 14.10.4

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
	err = validateEntrySI(XRefTable, dict, dictName, "SI", REQUIRED, types.V10)
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

	return nil
}

func validateIDSNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 14.10.4

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateURLSNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 14.10.4

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateEmbeddedFilesNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 7.11.4

	// Value is a file specification for an embedded file stream.

	if obj == nil {
		return nil
	}

	_, err := validateFileSpecification(xRefTable, obj)

	return err
}

func validateSlideShowDict(XRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 13.5, table 297

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

	return err
}

func validateAlternatePresentationsNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 13.5

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

	return nil
}

func validateRenditionsNameTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// see 13.2.3

	// Value is a rendition object.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if dict != nil {
		err = validateRenditionDict(xRefTable, dict, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateIDTreeValue(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
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

	return nil
}

func validateNameTreeValue(name string, xRefTable *types.XRefTable, obj types.PDFObject) (err error) {

	for k, v := range map[string]struct {
		validate     func(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error
		sinceVersion types.PDFVersion
	}{
		"Dests":                  {validateDestsNameTreeValue, types.V12},
		"AP":                     {validateAPNameTreeValue, types.V13},
		"JavaScript":             {validateJavaScriptNameTreeValue, types.V13},
		"Pages":                  {validatePagesNameTreeValue, types.V13},
		"Templates":              {validateTemplatesNameTreeValue, types.V13},
		"IDS":                    {validateIDSNameTreeValue, types.V13},
		"URLS":                   {validateURLSNameTreeValue, types.V13},
		"EmbeddedFiles":          {validateEmbeddedFilesNameTreeValue, types.V14},
		"AlternatePresentations": {validateAlternatePresentationsNameTreeValue, types.V14},
		"Renditions":             {validateRenditionsNameTreeValue, types.V15},
		"IDTree":                 {validateIDTreeValue, types.V13},
	} {
		if name == k {

			err := xRefTable.ValidateVersion(name, v.sinceVersion)
			if err != nil {
				return err
			}

			return v.validate(xRefTable, obj, v.sinceVersion)

		}
	}

	return errors.Errorf("validateNameTreeDictNamesEntry: unknown dict name: %s", name)
}

func validateNameTreeDictNamesEntry(xRefTable *types.XRefTable, dict *types.PDFDict, name string, node *types.Node) (firstKey, lastKey string, err error) {

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

	// arr length needs to be even because of contained key value pairs.
	if len(*arr)%2 == 1 {
		return "", "", errors.Errorf("validateNameTreeDictNamesEntry: Names array entry length needs to be even, length=%d\n", len(*arr))
	}

	var key string
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

			key = s.Value()

			if firstKey == "" {
				firstKey = key
			}

			lastKey = key

			continue
		}

		err = validateNameTreeValue(name, xRefTable, obj)
		if err != nil {
			return "", "", err
		}

		node.AddToLeaf(key, obj)
	}

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

func validateNameTree(xRefTable *types.XRefTable, name string, indRef types.PDFIndirectRef, root bool) (string, string, *types.Node, error) {

	// see 7.7.4

	// A node has "Kids" or "Names" entry.

	node := &types.Node{IndRef: &indRef}
	var kmin, kmax string

	var dict *types.PDFDict

	dict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || dict == nil {
		return "", "", nil, err
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		// Intermediate node

		var arr *types.PDFArray

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return "", "", nil, err
		}

		if arr == nil {
			return "", "", nil, errors.New("validateNameTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return "", "", nil, errors.New("validateNameTree: corrupt kid, should be indirect reference")
			}

			var kminKid string
			var kidNode *types.Node
			kminKid, kmax, kidNode, err = validateNameTree(xRefTable, name, kid, false)
			if err != nil {
				return "", "", nil, err
			}
			if kmin == "" {
				kmin = kminKid
			}

			node.Kids = append(node.Kids, kidNode)
		}

	} else {

		// Leaf node
		kmin, kmax, err = validateNameTreeDictNamesEntry(xRefTable, dict, name, node)
		if err != nil {
			return "", "", nil, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNameTreeDictLimitsEntry(xRefTable, dict, kmin, kmax)
		if err != nil {
			return "", "", nil, err
		}
	}

	// We track limits for all nodes internally.
	node.Kmin = kmin
	node.Kmax = kmax

	return kmin, kmax, node, nil
}

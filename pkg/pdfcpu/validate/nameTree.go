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

package validate

import (
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateDestsNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	return validateDestination(xRefTable, obj)
}

func validateAPNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	return validateAppearanceDict(xRefTable, obj)
}

func validateJavaScriptNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	// Javascript Action:
	return validateJavaScriptActionDict(xRefTable, dict, "JavaScript")
}

func validatePagesNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 12.7.6

	// Value is a page dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateTemplatesNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 12.7.6

	// Value is a template dict.

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "templateDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Template" })

	return err
}

func validateURLAliasDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "urlAliasDict"

	// U, required, ASCII string
	_, err := validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "C", OPTIONAL, pdf.V10, nil)

	return err
}

func validateCommandSettingsDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	// see 14.10.5.4

	dictName := "cmdSettingsDict"

	// G, optional, dict
	_, err := validateDictEntry(xRefTable, dict, dictName, "G", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "C", OPTIONAL, pdf.V10, nil)

	return err
}

func validateCaptureCommandDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "captureCommandDict"

	// URL, required, string
	_, err := validateStringEntry(xRefTable, dict, dictName, "URL", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "L", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, dict, dictName, "P", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CT", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "H", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, command settings dict
	d, err := validateDictEntry(xRefTable, dict, dictName, "S", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateCommandSettingsDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSourceInfoDictEntryAU(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.StringLiteral, pdf.HexLiteral:
		// no further processing

	case pdf.Dict:
		err = validateURLAliasDict(xRefTable, &obj)
		if err != nil {
			return err
		}

	default:
		return errors.New("validateSourceInfoDict: entry \"AU\" must be string or dict")

	}

	return nil
}

func validateSourceInfoDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	err := validateSourceInfoDictEntryAU(xRefTable, dict, dictName, "AU", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// E, optional, date
	_, err = validateDateEntry(xRefTable, dict, dictName, "E", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// S, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "S", OPTIONAL, pdf.V10, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// C, optional, indRef of command dict
	indRef, err := validateIndRefEntry(xRefTable, dict, dictName, "C", OPTIONAL, pdf.V10)
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

func validateEntrySI(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 14.10.5, table 355, source information dictionary

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Dict:
		err = validateSourceInfoDict(xRefTable, &obj)
		if err != nil {
			return err
		}

	case pdf.Array:

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

func validateWebCaptureContentSetDict(XRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	// see 14.10.4

	dictName := "webCaptureContentSetDict"

	// Type, optional, name
	_, err := validateNameEntry(XRefTable, dict, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "SpiderContentSet" })
	if err != nil {
		return err
	}

	// S, required, name
	s, err := validateNameEntry(XRefTable, dict, dictName, "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "SPS" || s == "SIS" })
	if err != nil {
		return err
	}

	// ID, required, byte string
	_, err = validateStringEntry(XRefTable, dict, dictName, "ID", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// O, required, array of indirect references.
	_, err = validateIndRefArrayEntry(XRefTable, dict, dictName, "O", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// SI, required, source info dict or array of source info dicts
	err = validateEntrySI(XRefTable, dict, dictName, "SI", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// CT, optional, string
	_, err = validateStringEntry(XRefTable, dict, dictName, "CT", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// TS, optional, date
	_, err = validateDateEntry(XRefTable, dict, dictName, "TS", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// spider page set
	if *s == "SPS" {

		// T, optional, string
		_, err = validateStringEntry(XRefTable, dict, dictName, "T", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return err
		}

		// TID, optional, byte string
		_, err = validateStringEntry(XRefTable, dict, dictName, "TID", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return err
		}
	}

	// spider image set
	if *s == "SIS" {

		// R, required, integer or array of integers
		err = validateIntegerOrArrayOfIntegerEntry(XRefTable, dict, dictName, "R", REQUIRED, pdf.V10)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateIDSNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 14.10.4

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateURLSNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 14.10.4

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateEmbeddedFilesNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 7.11.4

	// Value is a file specification for an embedded file stream.

	if obj == nil {
		return nil
	}

	_, err := validateFileSpecification(xRefTable, obj)

	return err
}

func validateSlideShowDict(XRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	// see 13.5, table 297

	dictName := "slideShowDict"

	// Type, required, name, since V1.4
	_, err := validateNameEntry(XRefTable, dict, dictName, "Type", REQUIRED, pdf.V14, func(s string) bool { return s == "SlideShow" })
	if err != nil {
		return err
	}

	// Subtype, required, name, since V1.4
	_, err = validateNameEntry(XRefTable, dict, dictName, "Subtype", REQUIRED, pdf.V14, func(s string) bool { return s == "Embedded" })
	if err != nil {
		return err
	}

	// Resources, required, name tree, since V1.4
	// Note: This is really an array of (string,indRef) pairs.
	_, err = validateArrayEntry(XRefTable, dict, dictName, "Resources", REQUIRED, pdf.V14, nil)
	if err != nil {
		return err
	}

	// StartResource, required, byte string, since V1.4
	_, err = validateStringEntry(XRefTable, dict, dictName, "StartResource", REQUIRED, pdf.V14, nil)

	return err
}

func validateAlternatePresentationsNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

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

func validateRenditionsNameTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

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

func validateIDTreeValue(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

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

func validateNameTreeValue(name string, xRefTable *pdf.XRefTable, obj pdf.Object) (err error) {

	for k, v := range map[string]struct {
		validate     func(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error
		sinceVersion pdf.Version
	}{
		"Dests":                  {validateDestsNameTreeValue, pdf.V12},
		"AP":                     {validateAPNameTreeValue, pdf.V13},
		"JavaScript":             {validateJavaScriptNameTreeValue, pdf.V13},
		"Pages":                  {validatePagesNameTreeValue, pdf.V13},
		"Templates":              {validateTemplatesNameTreeValue, pdf.V13},
		"IDS":                    {validateIDSNameTreeValue, pdf.V13},
		"URLS":                   {validateURLSNameTreeValue, pdf.V13},
		"EmbeddedFiles":          {validateEmbeddedFilesNameTreeValue, pdf.V14},
		"AlternatePresentations": {validateAlternatePresentationsNameTreeValue, pdf.V14},
		"Renditions":             {validateRenditionsNameTreeValue, pdf.V15},
		"IDTree":                 {validateIDTreeValue, pdf.V13},
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

func validateNameTreeDictNamesEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, name string, node *pdf.Node) (firstKey, lastKey string, err error) {

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

			s, ok := obj.(pdf.StringLiteral)
			if !ok {
				s, ok := obj.(pdf.HexLiteral)
				if !ok {
					return "", "", errors.Errorf("validateNameTreeDictNamesEntry: corrupt key <%v>\n", obj)
				}
				key = s.Value()
			} else {
				key = s.Value()
			}

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

func validateNameTreeDictLimitsEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, firstKey, lastKey string) error {

	var arr *pdf.Array

	arr, err := validateStringArrayEntry(xRefTable, dict, "nameTreeDict", "Limits", REQUIRED, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	var fkv, lkv string

	fk, ok := (*arr)[0].(pdf.StringLiteral)
	if !ok {
		fk, _ := (*arr)[0].(pdf.HexLiteral)
		fkv = fk.Value()
	} else {
		fkv = fk.Value()
	}

	lk, ok := (*arr)[1].(pdf.StringLiteral)
	if !ok {
		lk, _ := (*arr)[1].(pdf.HexLiteral)
		lkv = lk.Value()
	} else {
		lkv = lk.Value()
	}

	if firstKey != fkv || lastKey != lkv {
		return errors.Errorf("validateNameTreeDictLimitsEntry: leaf node corrupted\n")
	}

	return nil
}

func validateNameTree(xRefTable *pdf.XRefTable, name string, indRef pdf.IndirectRef, root bool) (string, string, *pdf.Node, error) {

	// see 7.7.4

	// A node has "Kids" or "Names" entry.

	node := &pdf.Node{IndRef: &indRef}
	var kmin, kmax string

	var dict *pdf.Dict

	dict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || dict == nil {
		return "", "", nil, err
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		// Intermediate node

		var arr *pdf.Array

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return "", "", nil, err
		}

		if arr == nil {
			return "", "", nil, errors.New("validateNameTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			kid, ok := obj.(pdf.IndirectRef)
			if !ok {
				return "", "", nil, errors.New("validateNameTree: corrupt kid, should be indirect reference")
			}

			var kminKid string
			var kidNode *pdf.Node
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

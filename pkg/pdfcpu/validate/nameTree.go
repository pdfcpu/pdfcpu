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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateDestsNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("DestsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateDestination(xRefTable, o, false)
	return err
}

func validateAPNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("APNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	return validateXObjectStreamDict(xRefTable, o)
}

func validateJavaScriptNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("JavaScriptNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	// Javascript Action:
	return validateJavaScriptActionDict(xRefTable, d, "JavaScript")
}

func validatePagesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 12.7.6

	// Version check
	err := xRefTable.ValidateVersion("PagesNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	// Value is a page dict.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.New("pdfcpu: validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, model.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateTemplatesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 12.7.6

	// Version check
	err := xRefTable.ValidateVersion("TemplatesNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	// Value is a template dict.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.New("pdfcpu: validatePagesNameTreeValue: value is nil")
	}

	_, err = validateNameEntry(xRefTable, d, "templateDict", "Type", REQUIRED, model.V10, func(s string) bool { return s == "Template" })

	return err
}

func validateURLAliasDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "urlAliasDict"

	// U, required, ASCII string
	_, err := validateStringEntry(xRefTable, d, dictName, "U", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, model.V10, nil)

	return err
}

func validateCommandSettingsDict(xRefTable *model.XRefTable, d types.Dict) error {

	// see 14.10.5.4

	dictName := "cmdSettingsDict"

	// G, optional, dict
	_, err := validateDictEntry(xRefTable, d, dictName, "G", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "C", OPTIONAL, model.V10, nil)

	return err
}

func validateCaptureCommandDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "captureCommandDict"

	// URL, required, string
	_, err := validateStringEntry(xRefTable, d, dictName, "URL", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "L", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, d, dictName, "P", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "CT", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, d, dictName, "H", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, command settings dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "S", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateCommandSettingsDict(xRefTable, d1)
	}

	return err
}

func validateSourceInfoDictEntryAU(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.StringLiteral, types.HexLiteral:
		// no further processing

	case types.Dict:
		err = validateURLAliasDict(xRefTable, o)
		if err != nil {
			return err
		}

	default:
		return errors.New("pdfcpu: validateSourceInfoDict: entry \"AU\" must be string or dict")

	}

	return nil
}

func validateSourceInfoDict(xRefTable *model.XRefTable, d types.Dict) error {

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	err := validateSourceInfoDictEntryAU(xRefTable, d, dictName, "AU", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// E, optional, date
	_, err = validateDateEntry(xRefTable, d, dictName, "E", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// S, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "S", OPTIONAL, model.V10, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// C, optional, indRef of command dict
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "C", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	if ir != nil {

		d1, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return err
		}

		return validateCaptureCommandDict(xRefTable, d1)

	}

	return nil
}

func validateEntrySI(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

	// see 14.10.5, table 355, source information dictionary

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Dict:
		err = validateSourceInfoDict(xRefTable, o)
		if err != nil {
			return err
		}

	case types.Array:

		for _, v := range o {

			if v == nil {
				continue
			}

			d1, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			err = validateSourceInfoDict(xRefTable, d1)
			if err != nil {
				return err
			}

		}

	}

	return nil
}

func validateWebCaptureContentSetDict(XRefTable *model.XRefTable, d types.Dict) error {

	// see 14.10.4

	dictName := "webCaptureContentSetDict"

	// Type, optional, name
	_, err := validateNameEntry(XRefTable, d, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "SpiderContentSet" })
	if err != nil {
		return err
	}

	// S, required, name
	s, err := validateNameEntry(XRefTable, d, dictName, "S", REQUIRED, model.V10, func(s string) bool { return s == "SPS" || s == "SIS" })
	if err != nil {
		return err
	}

	// ID, required, byte string
	_, err = validateStringEntry(XRefTable, d, dictName, "ID", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// O, required, array of indirect references.
	_, err = validateIndRefArrayEntry(XRefTable, d, dictName, "O", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// SI, required, source info dict or array of source info dicts
	err = validateEntrySI(XRefTable, d, dictName, "SI", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// CT, optional, string
	_, err = validateStringEntry(XRefTable, d, dictName, "CT", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// TS, optional, date
	_, err = validateDateEntry(XRefTable, d, dictName, "TS", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// spider page set
	if *s == "SPS" {

		// T, optional, string
		_, err = validateStringEntry(XRefTable, d, dictName, "T", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}

		// TID, optional, byte string
		_, err = validateStringEntry(XRefTable, d, dictName, "TID", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}
	}

	// spider image set
	if *s == "SIS" {

		// R, required, integer or array of integers
		err = validateIntegerOrArrayOfIntegerEntry(XRefTable, d, dictName, "R", REQUIRED, model.V10)

	}

	return err
}

func validateIDSNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 14.10.4

	// Version check
	err := xRefTable.ValidateVersion("IDSNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateURLSNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 14.10.4

	// Version check
	err := xRefTable.ValidateVersion("URLSNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	// Value is a web capture content set.
	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateEmbeddedFilesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 7.11.4

	// Value is a file specification for an embedded file stream.

	// Version check
	err := xRefTable.ValidateVersion("EmbeddedFilesNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	if o == nil {
		return nil
	}

	_, err = validateFileSpecification(xRefTable, o)

	return err
}

func validateSlideShowDict(XRefTable *model.XRefTable, d types.Dict) error {

	// see 13.5, table 297

	dictName := "slideShowDict"

	// Type, required, name, since V1.4
	_, err := validateNameEntry(XRefTable, d, dictName, "Type", REQUIRED, model.V14, func(s string) bool { return s == "SlideShow" })
	if err != nil {
		return err
	}

	// Subtype, required, name, since V1.4
	_, err = validateNameEntry(XRefTable, d, dictName, "Subtype", REQUIRED, model.V14, func(s string) bool { return s == "Embedded" })
	if err != nil {
		return err
	}

	// Resources, required, name tree, since V1.4
	// Note: This is really an array of (string,indRef) pairs.
	_, err = validateArrayEntry(XRefTable, d, dictName, "Resources", REQUIRED, model.V14, nil)
	if err != nil {
		return err
	}

	// StartResource, required, byte string, since V1.4
	_, err = validateStringEntry(XRefTable, d, dictName, "StartResource", REQUIRED, model.V14, nil)

	return err
}

func validateAlternatePresentationsNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 13.5

	// Value is a slide show dict.

	// Version check
	err := xRefTable.ValidateVersion("AlternatePresentationsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateSlideShowDict(xRefTable, d)
	}

	return err
}

func validateRenditionsNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 13.2.3

	// Value is a rendition object.

	// Version check
	err := xRefTable.ValidateVersion("RenditionsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateRenditionDict(xRefTable, d, sinceVersion)
	}

	return err
}

func validateIDTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("IDTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictType := d.Type()
	if dictType == nil || *dictType == "StructElem" {
		err = validateStructElementDict(xRefTable, d)
		if err != nil {
			return err
		}
	} else {
		return errors.Errorf("pdfcpu: validateIDTreeValue: invalid dictType %s (should be \"StructElem\")\n", *dictType)
	}

	return nil
}

func validateNameTreeValue(name string, xRefTable *model.XRefTable, o types.Object) (err error) {

	// The values associated with the keys may be objects of any type.
	// Stream objects shall be specified by indirect object references.
	// Dictionary, array, and string objects should be specified by indirect object references.
	// Other PDF objects (nulls, numbers, booleans, and names) should be specified as direct objects.

	for k, v := range map[string]struct {
		validate     func(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error
		sinceVersion model.Version
	}{
		"Dests":                  {validateDestsNameTreeValue, model.V12},
		"AP":                     {validateAPNameTreeValue, model.V13},
		"JavaScript":             {validateJavaScriptNameTreeValue, model.V13},
		"Pages":                  {validatePagesNameTreeValue, model.V13},
		"Templates":              {validateTemplatesNameTreeValue, model.V13},
		"IDS":                    {validateIDSNameTreeValue, model.V13},
		"URLS":                   {validateURLSNameTreeValue, model.V13},
		"EmbeddedFiles":          {validateEmbeddedFilesNameTreeValue, model.V14},
		"AlternatePresentations": {validateAlternatePresentationsNameTreeValue, model.V14},
		"Renditions":             {validateRenditionsNameTreeValue, model.V15},
		"IDTree":                 {validateIDTreeValue, model.V13},
	} {
		if name == k {
			return v.validate(xRefTable, o, v.sinceVersion)
		}
	}

	return errors.Errorf("pdfcpu: validateNameTreeDictNamesEntry: unknown dict name: %s", name)
}

func validateNameTreeDictNamesEntry(xRefTable *model.XRefTable, d types.Dict, name string, node *model.Node) (string, string, error) {

	//fmt.Printf("validateNameTreeDictNamesEntry begin %s\n", d)

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	o, found := d.Find("Names")
	if !found {
		return "", "", errors.Errorf("pdfcpu: validateNameTreeDictNamesEntry: missing \"Kids\" or \"Names\" entry.")
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return "", "", err
	}
	if a == nil {
		return "", "", errors.Errorf("pdfcpu: validateNameTreeDictNamesEntry: missing \"Names\" array.")
	}

	// arr length needs to be even because of contained key value pairs.
	if len(a)%2 == 1 {
		return "", "", errors.Errorf("pdfcpu: validateNameTreeDictNamesEntry: Names array entry length needs to be even, length=%d\n", len(a))
	}

	var key, firstKey, lastKey string

	for i := 0; i < len(a); i++ {
		o := a[i]

		if i%2 == 0 {

			// TODO Do we really need to process indRefs here?
			o, err = xRefTable.Dereference(o)
			if err != nil {
				return "", "", err
			}

			k, err := types.StringOrHexLiteral(o)
			if err != nil {
				return "", "", err
			}

			key = *k

			if firstKey == "" {
				firstKey = key
			}

			lastKey = key

			continue
		}

		err = validateNameTreeValue(name, xRefTable, o)
		if err != nil {
			return "", "", err
		}

		node.AppendToNames(key, o)

	}

	return firstKey, lastKey, nil
}

func validateNameTreeDictLimitsEntry(xRefTable *model.XRefTable, d types.Dict, firstKey, lastKey string) error {

	a, err := validateStringArrayEntry(xRefTable, d, "nameTreeDict", "Limits", REQUIRED, model.V10, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	var fkv, lkv string

	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		return err
	}

	s, err := types.StringOrHexLiteral(o)
	if err != nil {
		return err
	}
	fkv = *s

	if o, err = xRefTable.Dereference(a[1]); err != nil {
		return err
	}

	s, err = types.StringOrHexLiteral(o)
	if err != nil {
		return err
	}
	lkv = *s

	if firstKey < fkv || lastKey > lkv {
		return errors.Errorf("pdfcpu: validateNameTreeDictLimitsEntry: leaf node corrupted (firstKey: %s vs %s) (lastKey: %s vs %s)\n", firstKey, fkv, lastKey, lkv)
	}

	return nil
}

func validateNameTree(xRefTable *model.XRefTable, name string, d types.Dict, root bool) (string, string, *model.Node, error) {

	//fmt.Printf("validateNameTree begin %s\n", d)

	// see 7.7.4

	// A node has "Kids" or "Names" entry.

	//fmt.Printf("validateNameTree %s\n", name)

	node := &model.Node{D: d}
	var kmin, kmax string
	var err error

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if o, found := d.Find("Kids"); found {

		// Intermediate node

		a, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return "", "", nil, err
		}

		if a == nil {
			return "", "", nil, errors.New("pdfcpu: validateNameTree: missing \"Kids\" array")
		}

		for _, o := range a {

			d, err := xRefTable.DereferenceDict(o)
			if err != nil {
				return "", "", nil, err
			}

			var kminKid string
			var kidNode *model.Node
			kminKid, kmax, kidNode, err = validateNameTree(xRefTable, name, d, false)
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
		kmin, kmax, err = validateNameTreeDictNamesEntry(xRefTable, d, name, node)
		if err != nil {
			return "", "", nil, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNameTreeDictLimitsEntry(xRefTable, d, kmin, kmax)
		if err != nil {
			return "", "", nil, err
		}
	}

	// We track limits for all nodes internally.
	node.Kmin = kmin
	node.Kmax = kmax

	//fmt.Println("validateNameTree end")

	return kmin, kmax, node, nil
}

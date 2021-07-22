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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateDestsNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("DestsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	return validateDestination(xRefTable, o)
}

func validateAPNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("APNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	return validateXObjectStreamDict(xRefTable, o)
}

func validateJavaScriptNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validatePagesNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

	_, err = validateNameEntry(xRefTable, d, "pageDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateTemplatesNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

	_, err = validateNameEntry(xRefTable, d, "templateDict", "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "Template" })

	return err
}

func validateURLAliasDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "urlAliasDict"

	// U, required, ASCII string
	_, err := validateStringEntry(xRefTable, d, dictName, "U", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "C", OPTIONAL, pdf.V10, nil)

	return err
}

func validateCommandSettingsDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 14.10.5.4

	dictName := "cmdSettingsDict"

	// G, optional, dict
	_, err := validateDictEntry(xRefTable, d, dictName, "G", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "C", OPTIONAL, pdf.V10, nil)

	return err
}

func validateCaptureCommandDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "captureCommandDict"

	// URL, required, string
	_, err := validateStringEntry(xRefTable, d, dictName, "URL", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "L", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, d, dictName, "P", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "CT", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, d, dictName, "H", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// S, optional, command settings dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "S", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateCommandSettingsDict(xRefTable, d1)
	}

	return err
}

func validateSourceInfoDictEntryAU(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.StringLiteral, pdf.HexLiteral:
		// no further processing

	case pdf.Dict:
		err = validateURLAliasDict(xRefTable, o)
		if err != nil {
			return err
		}

	default:
		return errors.New("pdfcpu: validateSourceInfoDict: entry \"AU\" must be string or dict")

	}

	return nil
}

func validateSourceInfoDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	err := validateSourceInfoDictEntryAU(xRefTable, d, dictName, "AU", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// E, optional, date
	_, err = validateDateEntry(xRefTable, d, dictName, "E", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// S, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "S", OPTIONAL, pdf.V10, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// C, optional, indRef of command dict
	ir, err := validateIndRefEntry(xRefTable, d, dictName, "C", OPTIONAL, pdf.V10)
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

func validateEntrySI(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 14.10.5, table 355, source information dictionary

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Dict:
		err = validateSourceInfoDict(xRefTable, o)
		if err != nil {
			return err
		}

	case pdf.Array:

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

func validateWebCaptureContentSetDict(XRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 14.10.4

	dictName := "webCaptureContentSetDict"

	// Type, optional, name
	_, err := validateNameEntry(XRefTable, d, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "SpiderContentSet" })
	if err != nil {
		return err
	}

	// S, required, name
	s, err := validateNameEntry(XRefTable, d, dictName, "Type", REQUIRED, pdf.V10, func(s string) bool { return s == "SPS" || s == "SIS" })
	if err != nil {
		return err
	}

	// ID, required, byte string
	_, err = validateStringEntry(XRefTable, d, dictName, "ID", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// O, required, array of indirect references.
	_, err = validateIndRefArrayEntry(XRefTable, d, dictName, "O", REQUIRED, pdf.V10, nil)
	if err != nil {
		return err
	}

	// SI, required, source info dict or array of source info dicts
	err = validateEntrySI(XRefTable, d, dictName, "SI", REQUIRED, pdf.V10)
	if err != nil {
		return err
	}

	// CT, optional, string
	_, err = validateStringEntry(XRefTable, d, dictName, "CT", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// TS, optional, date
	_, err = validateDateEntry(XRefTable, d, dictName, "TS", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// spider page set
	if *s == "SPS" {

		// T, optional, string
		_, err = validateStringEntry(XRefTable, d, dictName, "T", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return err
		}

		// TID, optional, byte string
		_, err = validateStringEntry(XRefTable, d, dictName, "TID", OPTIONAL, pdf.V10, nil)
		if err != nil {
			return err
		}
	}

	// spider image set
	if *s == "SIS" {

		// R, required, integer or array of integers
		err = validateIntegerOrArrayOfIntegerEntry(XRefTable, d, dictName, "R", REQUIRED, pdf.V10)

	}

	return err
}

func validateIDSNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validateURLSNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validateEmbeddedFilesNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validateSlideShowDict(XRefTable *pdf.XRefTable, d pdf.Dict) error {

	// see 13.5, table 297

	dictName := "slideShowDict"

	// Type, required, name, since V1.4
	_, err := validateNameEntry(XRefTable, d, dictName, "Type", REQUIRED, pdf.V14, func(s string) bool { return s == "SlideShow" })
	if err != nil {
		return err
	}

	// Subtype, required, name, since V1.4
	_, err = validateNameEntry(XRefTable, d, dictName, "Subtype", REQUIRED, pdf.V14, func(s string) bool { return s == "Embedded" })
	if err != nil {
		return err
	}

	// Resources, required, name tree, since V1.4
	// Note: This is really an array of (string,indRef) pairs.
	_, err = validateArrayEntry(XRefTable, d, dictName, "Resources", REQUIRED, pdf.V14, nil)
	if err != nil {
		return err
	}

	// StartResource, required, byte string, since V1.4
	_, err = validateStringEntry(XRefTable, d, dictName, "StartResource", REQUIRED, pdf.V14, nil)

	return err
}

func validateAlternatePresentationsNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validateRenditionsNameTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validateIDTreeValue(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

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

func validateNameTreeValue(name string, xRefTable *pdf.XRefTable, o pdf.Object) (err error) {

	// TODO
	// The values associated with the keys may be objects of any type.
	// Stream objects shall be specified by indirect object references.
	// Dictionary, array, and string objects should be specified by indirect object references.
	// Other PDF objects (nulls, numbers, booleans, and names) should be specified as direct objects.

	for k, v := range map[string]struct {
		validate     func(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error
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
			return v.validate(xRefTable, o, v.sinceVersion)
		}
	}

	return errors.Errorf("pdfcpu: validateNameTreeDictNamesEntry: unknown dict name: %s", name)
}

func validateNameTreeDictNamesEntry(xRefTable *pdf.XRefTable, d pdf.Dict, name string, node *pdf.Node) (firstKey, lastKey string, err error) {

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

	var key string
	for i, o := range a {

		if i%2 == 0 {

			o, err = xRefTable.Dereference(o)
			if err != nil {
				return "", "", err
			}

			s, ok := o.(pdf.StringLiteral)
			if !ok {
				s, ok := o.(pdf.HexLiteral)
				if !ok {
					return "", "", errors.Errorf("pdfcpu: validateNameTreeDictNamesEntry: corrupt key <%v>\n", o)
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

		err = validateNameTreeValue(name, xRefTable, o)
		if err != nil {
			return "", "", err
		}

		node.AddToLeaf(key, o)
	}

	return firstKey, lastKey, nil
}

func validateNameTreeDictLimitsEntry(xRefTable *pdf.XRefTable, d pdf.Dict, firstKey, lastKey string) error {

	a, err := validateStringArrayEntry(xRefTable, d, "nameTreeDict", "Limits", REQUIRED, pdf.V10, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	//fmt.Printf("validateNameTreeDictLimitsEntry: firstKey=%s lastKey=%s limits:%v\n", firstKey, lastKey, a)

	var fkv, lkv string

	fk, ok := a[0].(pdf.StringLiteral)
	if !ok {
		fk, _ := a[0].(pdf.HexLiteral)
		//bb, _ := fk.Bytes()
		//fmt.Printf("fk: %v %s\n", bb, string(bb))
		fkv = fk.Value()
	} else {
		fkv = fk.Value()
	}

	lk, ok := a[1].(pdf.StringLiteral)
	if !ok {
		lk, _ := a[1].(pdf.HexLiteral)
		//bb, _ := lk.Bytes()
		//fmt.Printf("lk: %v %s\n", bb, string(bb))
		lkv = lk.Value()
	} else {
		lkv = lk.Value()
	}

	if firstKey < fkv || lastKey > lkv {
		return errors.Errorf("pdfcpu: validateNameTreeDictLimitsEntry: leaf node corrupted (firstKey: %s vs %s) (lastKey: %s vs %s)\n", firstKey, fkv, lastKey, lkv)
	}

	return nil
}

func validateNameTree(xRefTable *pdf.XRefTable, name string, d pdf.Dict, root bool) (string, string, *pdf.Node, error) {

	// see 7.7.4

	// A node has "Kids" or "Names" entry.

	//fmt.Printf("validateNameTree %s\n", name)

	node := &pdf.Node{D: d}
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
			var kidNode *pdf.Node
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

	return kmin, kmax, node, nil
}

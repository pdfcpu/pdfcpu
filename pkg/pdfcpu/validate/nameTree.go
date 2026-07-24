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
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var errMissingNameTreeKidsOrNames = errors.New("missing Kids or Names")

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
		return fmt.Errorf("JavaScript name tree value: dereference dict: %w", err)
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
		return fmt.Errorf("Pages name tree value: dereference page dict: %w", err)
	}

	if d == nil {
		return errors.New("Pages name tree value: missing page dict")
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
		return fmt.Errorf("Templates name tree value: dereference template dict: %w", err)
	}
	if d == nil {
		return errors.New("Templates name tree value: missing template dict")
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
		return fmt.Errorf("%s.URL: %w", dictName, err)
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "L", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.L: %w", dictName, err)
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "F", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.F: %w", dictName, err)
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, d, dictName, "P", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.P: %w", dictName, err)
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "CT", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.CT: %w", dictName, err)
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, d, dictName, "H", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.H: %w", dictName, err)
	}

	// S, optional, command settings dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "S", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.S: %w", dictName, err)
	}
	if d1 != nil {
		err = validateCommandSettingsDict(xRefTable, d1)
		if err != nil {
			return fmt.Errorf("%s.S: %w", dictName, err)
		}
	}

	return nil
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
		return fmt.Errorf("dict=%s entry=%s expected string or dict, got %T", dictName, entryName, o)

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

		for i, v := range o {

			if v == nil {
				continue
			}

			d1, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return fmt.Errorf("dict=%s entry=%s[%d]: dereference dict: %w", dictName, entryName, i, err)
			}

			err = validateSourceInfoDict(xRefTable, d1)
			if err != nil {
				return fmt.Errorf("dict=%s entry=%s[%d]: %w", dictName, entryName, i, err)
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
	if err != nil {
		return fmt.Errorf("IDS name tree value: dereference content set dict: %w", err)
	}
	if d == nil {
		return errors.New("IDS name tree value: missing content set dict")
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
	if err != nil {
		return fmt.Errorf("URLS name tree value: dereference content set dict: %w", err)
	}
	if d == nil {
		return errors.New("URLS name tree value: missing content set dict")
	}

	return validateWebCaptureContentSetDict(xRefTable, d)
}

func validateEmbeddedFilesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 7.11.4

	// Value is a file specification for an embedded file stream.

	// Version check
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
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
		return fmt.Errorf("AlternatePresentations name tree value: dereference slide show dict: %w", err)
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
		return fmt.Errorf("Renditions name tree value: dereference rendition dict: %w", err)
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
	if err != nil {
		return fmt.Errorf("IDTree value: dereference structure element dict: %w", err)
	}
	if d == nil {
		return errors.New("IDTree value: missing structure element dict")
	}

	dictType := d.Type()
	if dictType == nil || *dictType == "StructElem" {
		err = validateStructElementDict(xRefTable, d, true)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("IDTree value: unexpected dict Type %s, expected StructElem", *dictType)
	}

	return nil
}

func validateNameTreeValue(name string, xRefTable *model.XRefTable, o types.Object) (err error) {
	// The values associated with the keys may be objects of any type.
	// Stream objects shall be specified by indirect object references.
	// Dictionary, array, and string objects should be specified by indirect object references.
	// Other PDF objects (nulls, numbers, booleans, and names) should be specified as direct objects.

	for k, v := range map[string]struct {
		validate            func(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error
		sinceVersion        model.Version
		sinceVersionRelaxed model.Version
	}{
		"Dests":                  {validateDestsNameTreeValue, model.V12, model.V12},
		"AP":                     {validateAPNameTreeValue, model.V13, model.V13},
		"JavaScript":             {validateJavaScriptNameTreeValue, model.V13, model.V13},
		"Pages":                  {validatePagesNameTreeValue, model.V13, model.V13},
		"Templates":              {validateTemplatesNameTreeValue, model.V13, model.V13},
		"IDS":                    {validateIDSNameTreeValue, model.V13, model.V13},
		"URLS":                   {validateURLSNameTreeValue, model.V13, model.V13},
		"EmbeddedFiles":          {validateEmbeddedFilesNameTreeValue, model.V14, model.V11},
		"AlternatePresentations": {validateAlternatePresentationsNameTreeValue, model.V14, model.V14},
		"Renditions":             {validateRenditionsNameTreeValue, model.V15, model.V15},
		"IDTree":                 {validateIDTreeValue, model.V13, model.V13},
	} {
		if name == k {
			sinceVersion := v.sinceVersion
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				sinceVersion = v.sinceVersionRelaxed
			}
			return v.validate(xRefTable, o, sinceVersion)
		}
	}

	return fmt.Errorf("name tree %s: unknown tree name", name)
}

func validateNameTreeDictNamesEntry(xRefTable *model.XRefTable, d types.Dict, name string, node *model.Node) (string, string, error) {
	//fmt.Printf("validateNameTreeDictNamesEntry begin %s\n", d)

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	o, found := d.Find("Names")
	if !found {
		return "", "", fmt.Errorf("name tree %s: %w", name, errMissingNameTreeKidsOrNames)
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return "", "", fmt.Errorf("name tree %s Names: dereference array: %w", name, err)
	}
	if a == nil {
		return "", "", fmt.Errorf("name tree %s: missing Names array", name)
	}

	// arr length needs to be even because of contained key value pairs.
	entries := len(a)
	if entries%2 == 1 {
		if xRefTable.ValidationMode != model.ValidationRelaxed || name != "JavaScript" {
			return "", "", fmt.Errorf("name tree %s Names: odd entry count %d", name, len(a))
		}
		entries--
	}

	var key, firstKey, lastKey string

	for i := 0; i < entries; i++ {
		o := a[i]

		if i%2 == 0 {

			// TODO Do we really need to process indRefs here?
			o, err = xRefTable.Dereference(o)
			if err != nil {
				return "", "", fmt.Errorf("name tree %s Names[%d]: dereference key: %w", name, i, err)
			}

			k, err := types.StringOrHexLiteral(o)
			if err != nil {
				return "", "", fmt.Errorf("name tree %s Names[%d]: expected string key: %w", name, i, err)
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
			return "", "", fmt.Errorf("name tree %s key %q: %w", name, key, err)
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

	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		return fmt.Errorf("name tree Limits[0]: dereference: %w", err)
	}
	s, err := types.StringOrHexLiteral(o)
	if err != nil {
		return fmt.Errorf("name tree Limits[0]: expected string: %w", err)
	}
	fkv := *s

	o, err = xRefTable.Dereference(a[1])
	if err != nil {
		return fmt.Errorf("name tree Limits[1]: dereference: %w", err)
	}
	s, err = types.StringOrHexLiteral(o)
	if err != nil {
		return fmt.Errorf("name tree Limits[1]: expected string: %w", err)
	}
	lkv := *s

	if xRefTable.ValidationMode == model.ValidationRelaxed {

		if fkv != firstKey && xRefTable.ValidationMode == model.ValidationRelaxed {
			fkv = firstKey
			a[0] = types.StringLiteral(fkv)
		}

		if lkv != lastKey && xRefTable.ValidationMode == model.ValidationRelaxed {
			lkv = lastKey
			a[1] = types.StringLiteral(lkv)
		}

	}

	if firstKey != fkv || lastKey != lkv {
		return fmt.Errorf("name tree leaf limits: first key %s, expected %s; last key %s, expected %s", fkv, firstKey, lkv, lastKey)
	}

	return nil
}

func validateNameTree(xRefTable *model.XRefTable, name string, d types.Dict, root bool) (string, string, *model.Node, error) {
	return validateNameTreeDepth(xRefTable, name, d, root, 0)
}

func nameTreeKidContext(name string, o types.Object, i int) string {
	if ir, ok := o.(types.IndirectRef); ok {
		return fmt.Sprintf("name tree %s Kids[%d] obj#%d", name, i, ir.ObjectNumber.Value())
	}
	return fmt.Sprintf("name tree %s Kids[%d]", name, i)
}

func validateNameTreeKids(
	xRefTable *model.XRefTable,
	name string,
	a types.Array,
	node *model.Node,
	depth int,
	specViolations *[]error,
) (string, string, error) {
	var kmin, kmax string

	for i, o := range a {

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			return "", "", fmt.Errorf("%s: dereference dict: %w", nameTreeKidContext(name, o, i), err)
		}
		if d == nil {
			return "", "", fmt.Errorf("%s: missing dict", nameTreeKidContext(name, o, i))
		}

		kminKid, kmaxKid, kidNode, err := validateNameTreeDepthWithViolations(
			xRefTable,
			name,
			d,
			false,
			depth+1,
			specViolations,
		)
		if err != nil {
			err = fmt.Errorf("%s: %w", nameTreeKidContext(name, o, i), err)
			if xRefTable.ValidationMode == model.ValidationStrict {
				return "", "", err
			}
			if errors.Is(err, errMissingNameTreeKidsOrNames) {
				*specViolations = append(*specViolations, err)
			}
			continue
		}
		kmax = kmaxKid
		if kmin == "" {
			kmin = kminKid
		}

		node.Kids = append(node.Kids, kidNode)
	}

	return kmin, kmax, nil
}

func validateNameTreeDepth(xRefTable *model.XRefTable, name string, d types.Dict, root bool, depth int) (string, string, *model.Node, error) {
	var specViolations []error
	kmin, kmax, node, err := validateNameTreeDepthWithViolations(
		xRefTable,
		name,
		d,
		root,
		depth,
		&specViolations,
	)
	if err == nil {
		showDigestedSpecViolations(xRefTable, specViolations)
	}
	return kmin, kmax, node, err
}

func validateNameTreeDepthWithViolations(
	xRefTable *model.XRefTable,
	name string,
	d types.Dict,
	root bool,
	depth int,
	specViolations *[]error,
) (string, string, *model.Node, error) {
	if err := xRefTable.CheckRecursionDepth("name tree", depth); err != nil {
		return "", "", nil, err
	}

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
			return "", "", nil, fmt.Errorf("name tree %s Kids: dereference array: %w", name, err)
		}
		if a == nil {
			return "", "", nil, fmt.Errorf("name tree %s: missing Kids array", name)
		}

		if len(a) == 0 {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return "", "", nil, fmt.Errorf("name tree %s: empty Kids array", name)
			}
			return "", "", nil, nil
		}

		kmin, kmax, err = validateNameTreeKids(xRefTable, name, a, node, depth, specViolations)
		if err != nil {
			return "", "", nil, err
		}
	} else {

		// Leaf node
		kmin, kmax, err = validateNameTreeDictNamesEntry(xRefTable, d, name, node)
		if err != nil {
			if root &&
				xRefTable.ValidationMode == model.ValidationRelaxed &&
				errors.Is(err, errMissingNameTreeKidsOrNames) {
				*specViolations = append(*specViolations, err)
				return "", "", node, nil
			}
			return "", "", nil, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNameTreeDictLimitsEntry(xRefTable, d, kmin, kmax)
		if err != nil {
			return "", "", nil, fmt.Errorf("name tree %s Limits: %w", name, err)
		}
	}

	// We track limits for all nodes internally.
	node.Kmin = kmin
	node.Kmax = kmax

	//fmt.Println("validateNameTree end")

	return kmin, kmax, node, nil
}

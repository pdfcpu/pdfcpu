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

func validateMarkedContentReferenceDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	var err error

	// Pg: optional, indirect reference
	// Page object representing a page on which the graphics object in the marked-content sequence shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err = processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return err
		}
	}

	// Stm: optional, indirect reference
	// The content stream containing the marked-content sequence.
	if ir := d.IndirectRefEntry("Stm"); ir != nil {
		_, err = xRefTable.Dereference(ir)
		if err != nil {
			return err
		}
	}

	// StmOwn: optional, indirect reference
	// The PDF object owning the stream identified by Stems annotation to which an appearance stream belongs.
	if ir := d.IndirectRefEntry("StmOwn"); ir != nil {
		_, err = xRefTable.Dereference(ir)
		if err != nil {
			return err
		}
	}

	// MCID: required, integer
	// The marked-content identifier of the marked-content sequence within its content stream.

	if d.IntEntry("MCID") == nil {
		err = errors.Errorf("pdfcpu: validateMarkedContentReferenceDict: missing entry \"MCID\".")
	}

	// if o, found := d.Find("MCID"); !found {
	// 	// TODO FIX!
	// } else {
	// 	o, err := xRefTable.Dereference(o)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if o == nil {
	// 		return errors.Errorf("validateMarkedContentReferenceDict: missing entry \"MCID\".")
	// 	}
	// }

	return err
}

func validateObjectReferenceDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err := processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return err
		}
	}

	// Obj: required, indirect reference
	ir := d.IndirectRefEntry("Obj")
	if xRefTable.ValidationMode == pdf.ValidationStrict && ir == nil {
		return errors.New("pdfcpu: validateObjectReferenceDict: missing required entry \"Obj\"")
	}

	if ir == nil {
		return nil
	}

	obj, err := xRefTable.Dereference(*ir)
	if err != nil {
		return err
	}

	if obj == nil {
		return errors.New("pdfcpu: validateObjectReferenceDict: missing required entry \"Obj\"")
	}

	return nil
}

func validateStructElementKArrayElement(xRefTable *pdf.XRefTable, o pdf.Object) error {
	switch o := o.(type) {

	case pdf.Integer:
		return nil

	case pdf.Dict:

		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			return validateStructElementDict(xRefTable, o)
		}

		if *dictType == "MCR" {
			return validateMarkedContentReferenceDict(xRefTable, o)
		}

		if *dictType == "OBJR" {
			return validateObjectReferenceDict(xRefTable, o)
		}

		return errors.Errorf("validateStructElementKArrayElement: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

	}

	return errors.New("validateStructElementKArrayElement: unsupported PDF object")
}

func validateStructElementDictEntryKArray(xRefTable *pdf.XRefTable, a pdf.Array) error {
	for _, o := range a {

		// Avoid recursion.
		ir, ok := o.(pdf.IndirectRef)
		if ok {
			valid, err := xRefTable.IsValid(ir)
			if err != nil {
				return err
			}
			if valid {
				continue
			}
			if err := xRefTable.SetValid(ir); err != nil {
				return err
			}
		}

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		if err := validateStructElementKArrayElement(xRefTable, o); err != nil {
			return err
		}

	}

	return nil
}

func validateStructElementDictEntryK(xRefTable *pdf.XRefTable, o pdf.Object) error {

	// K: optional, the children of this structure element
	//
	// struct element dict
	// marked content reference dict
	// object reference dict
	// marked content id int
	// array of all above

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Integer:

	case pdf.Dict:
		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDict(xRefTable, o)
			if err != nil {
				return err
			}
			break
		}

		if *dictType == "MCR" {
			err = validateMarkedContentReferenceDict(xRefTable, o)
			if err != nil {
				return err
			}
			break
		}

		if *dictType == "OBJR" {
			err = validateObjectReferenceDict(xRefTable, o)
			if err != nil {
				return err
			}
			break
		}

		return errors.Errorf("pdfcpu: validateStructElementDictEntryK: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

	case pdf.Array:

		err = validateStructElementDictEntryKArray(xRefTable, o)
		if err != nil {
			return err
		}

	default:
		return errors.New("pdfcpu: validateStructElementDictEntryK: unsupported PDF object")

	}

	return nil
}

func processStructElementDictPgEntry(xRefTable *pdf.XRefTable, ir pdf.IndirectRef) error {

	// is this object a known page object?

	o, err := xRefTable.Dereference(ir)
	if err != nil {
		return errors.Errorf("pdfcpu: processStructElementDictPgEntry: Pg obj:#%d gen:%d unknown\n", ir.ObjectNumber, ir.GenerationNumber)
	}

	//logInfoWriter.Printf("known object for Pg: %v %s\n", obj, obj)

	if xRefTable.ValidationMode == pdf.ValidationRelaxed && o == nil {
		return nil
	}

	pageDict, ok := o.(pdf.Dict)
	if !ok {
		return errors.Errorf("pdfcpu: processStructElementDictPgEntry: Pg object corrupt dict: %s\n", o)
	}

	if t := pageDict.Type(); t == nil || *t != "Page" {
		return errors.Errorf("pdfcpu: processStructElementDictPgEntry: Pg object no pageDict: %s\n", pageDict)
	}

	return nil
}

func validateStructElementDictEntryA(xRefTable *pdf.XRefTable, o pdf.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Dict: // No further processing.

	case pdf.StreamDict: // No further processing.

	case pdf.Array:

		for _, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			switch o.(type) {

			case pdf.Integer:
				// Each array element may be followed by a revision number (int).sort

			case pdf.Dict:
				// No further processing.

			case pdf.StreamDict:
				// No further processing.

			default:
				return errors.Errorf("pdfcpu: validateStructElementDictEntryA: unsupported PDF object: %v\n.", o)
			}
		}

	default:
		return errors.Errorf("pdfcpu: validateStructElementDictEntryA: unsupported PDF object: %v\n.", o)

	}

	return nil
}

func validateStructElementDictEntryC(xRefTable *pdf.XRefTable, o pdf.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		// No further processing.

	case pdf.Array:

		for _, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			switch o.(type) {

			case pdf.Name:
				// No further processing.

			case pdf.Integer:
				// Each array element may be followed by a revision number.

			default:
				return errors.New("pdfcpu: validateStructElementDictEntryC: unsupported PDF object")

			}
		}

	default:
		return errors.New("pdfcpu: validateStructElementDictEntryC: unsupported PDF object")

	}

	return nil
}

func validateStructElementDictPart1(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// S: structure type, required, name, see 14.7.3 and Annex E.
	_, err := validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// P: immediate parent, required, indirect reference
	ir := d.IndirectRefEntry("P")
	if xRefTable.ValidationMode != pdf.ValidationRelaxed {
		if ir == nil {
			return errors.Errorf("pdfcpu: validateStructElementDict: missing entry P: %s\n", d)
		}

		// Check if parent structure element exists.
		if _, ok := xRefTable.FindTableEntryForIndRef(ir); !ok {
			return errors.Errorf("pdfcpu: validateStructElementDict: unknown parent: %v\n", ir)
		}
	}

	// ID: optional, byte string
	_, err = validateStringEntry(xRefTable, d, dictName, "ID", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err = processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return err
		}
	}

	// K: optional, the children of this structure element.
	if o, found := d.Find("K"); found {
		err = validateStructElementDictEntryK(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// A: optional, attribute objects: dict or stream dict or array of these.
	if o, ok := d.Find("A"); ok {
		err = validateStructElementDictEntryA(xRefTable, o)
	}

	return err
}

func validateStructElementDictPart2(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string) error {

	// C: optional, name or array
	if o, ok := d.Find("C"); ok {
		err := validateStructElementDictEntryC(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// R: optional, integer >= 0
	_, err := validateIntegerEntry(xRefTable, d, dictName, "R", OPTIONAL, pdf.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// T: optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Lang: optional, text string, since 1.4
	sinceVersion := pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "Lang", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Alt: optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Alt", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// E: optional, text sttring, since 1.5
	_, err = validateStringEntry(xRefTable, d, dictName, "E", OPTIONAL, pdf.V15, nil)
	if err != nil {
		return err
	}

	// ActualText: optional, text string, since 1.4
	_, err = validateStringEntry(xRefTable, d, dictName, "ActualText", OPTIONAL, pdf.V14, nil)

	return err
}

func validateStructElementDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	// See table 323

	dictName := "StructElementDict"

	err := validateStructElementDictPart1(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	return validateStructElementDictPart2(xRefTable, d, dictName)
}

func validateStructTreeRootDictEntryKArray(xRefTable *pdf.XRefTable, a pdf.Array) error {

	for _, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		switch o := o.(type) {

		case pdf.Dict:

			dictType := o.Type()

			if dictType == nil || *dictType == "StructElem" {
				err = validateStructElementDict(xRefTable, o)
				if err != nil {
					return err
				}
				break
			}

			return errors.Errorf("pdfcpu: validateStructTreeRootDictEntryKArray: invalid dictType %s (should be \"StructElem\")\n", *dictType)

		default:
			return errors.New("pdfcpu: validateStructTreeRootDictEntryKArray: unsupported PDF object")

		}
	}

	return nil
}

func validateStructTreeRootDictEntryK(xRefTable *pdf.XRefTable, o pdf.Object) error {

	// The immediate child or children of the structure tree root in the structure hierarchy.
	// The value may be either a dictionary representing a single structure element or an array of such dictionaries.

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Dict:

		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDict(xRefTable, o)
			if err != nil {
				return err
			}
			break
		}

		return errors.Errorf("validateStructTreeRootDictEntryK: invalid dictType %s (should be \"StructElem\")\n", *dictType)

	case pdf.Array:

		err = validateStructTreeRootDictEntryKArray(xRefTable, o)
		if err != nil {
			return err
		}

	default:
		return errors.New("pdfcpu: validateStructTreeRootDictEntryK: unsupported PDF object")

	}

	return nil
}

func processStructTreeClassMapDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	for _, o := range d {

		// Process dict or array of dicts.

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		switch o := o.(type) {

		case pdf.Dict:
			// no further processing.

		case pdf.Array:

			for _, o := range o {

				_, err = xRefTable.DereferenceDict(o)
				if err != nil {
					return err
				}

			}

		default:
			return errors.New("pdfcpu: processStructTreeClassMapDict: unsupported PDF object")

		}

	}

	return nil
}

func validateStructTreeRootDictEntryParentTree(xRefTable *pdf.XRefTable, ir *pdf.IndirectRef) error {

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {

		// Accept empty dict
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return err
		}
		if d == nil || d.Len() == 0 {
			return nil
		}
	}

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return err
	}

	_, _, err = validateNumberTree(xRefTable, "StructTree", d, true)
	return err
}

func validateStructTreeRootDict(xRefTable *pdf.XRefTable, d pdf.Dict) error {

	dictName := "StructTreeRootDict"

	// required entry Type: name:StructTreeRoot
	if d.Type() == nil || *d.Type() != "StructTreeRoot" {
		return errors.New("pdfcpu: validateStructTreeRootDict: missing type")
	}

	// Optional entry K: struct element dict or array of struct element dicts
	if o, found := d.Find("K"); found {
		err := validateStructTreeRootDictEntryK(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// Optional entry IDTree: name tree, key=elementId value=struct element dict
	// A name tree that maps element identifiers to the structure elements they denote.
	ir := d.IndirectRefEntry("IDTree")
	if ir != nil {
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return err
		}
		_, _, _, err = validateNameTree(xRefTable, "IDTree", d, true)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTree: number tree, value=indRef of struct element dict or array of struct element dicts
	// A number tree used in finding the structure elements to which content items belong.
	if ir = d.IndirectRefEntry("ParentTree"); ir != nil {
		err := validateStructTreeRootDictEntryParentTree(xRefTable, ir)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTreeNextKey: integer
	_, err := validateIntegerEntry(xRefTable, d, dictName, "ParentTreeNextKey", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry RoleMap: dict
	// A dictionary that shall map the names of structure used in the document
	// to their approximate equivalents in the set of standard structure
	_, err = validateDictEntry(xRefTable, d, dictName, "RoleMap", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry ClassMap: dict
	// A dictionary that shall map name objects designating attribute classes
	// to the corresponding attribute objects or arrays of attribute objects.
	d1, err := validateDictEntry(xRefTable, d, dictName, "ClassMap", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = processStructTreeClassMapDict(xRefTable, d1)
	}

	return err
}

func validateStructTree(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// 14.7.2 Structure Hierarchy

	d, err := validateDictEntry(xRefTable, rootDict, "RootDict", "StructTreeRoot", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return validateStructTreeRootDict(xRefTable, d)
}

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

func validateMarkedContentReferenceDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	// Pg: optional, indirect reference
	// Page object representing a page on which the graphics object in the marked-content sequence shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		err := processStructElementDictPgEntry(xRefTable, *indRef)
		if err != nil {
			return err
		}
	}

	// Stm: optional, indirect reference
	// The content stream containing the marked-content sequence.
	if indRef := dict.IndirectRefEntry("Stm"); indRef != nil {
		_, err := xRefTable.Dereference(indRef)
		if err != nil {
			return err
		}
	}

	// StmOwn: optional, indirect reference
	// The PDF object owning the stream identified by Stems annotation to which an appearance stream belongs.
	if indRef := dict.IndirectRefEntry("StmOwn"); indRef != nil {
		_, err := xRefTable.Dereference(indRef)
		if err != nil {
			return err
		}
	}

	// MCID: required, integer
	// The marked-content identifier of the marked-content sequence within its content stream.
	if obj, found := dict.Find("MCID"); !found {
	} else {
		obj, err := xRefTable.Dereference(obj)
		if err != nil {
			return err
		}

		if obj == nil {
			return errors.Errorf("validateMarkedContentReferenceDict: missing entry \"MCID\".")
		}
	}

	return nil
}

func validateObjectReferenceDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		err := processStructElementDictPgEntry(xRefTable, *indRef)
		if err != nil {
			return err
		}
	}

	// Obj: required, indirect reference
	indRef := dict.IndirectRefEntry("Obj")
	if indRef == nil {
		return errors.New("validateObjectReferenceDict: missing required entry \"Obj\"")
	}

	obj, err := xRefTable.Dereference(*indRef)
	if err != nil {
		return err
	}

	if obj == nil {
		// object is nil
		return errors.New("validateObjectReferenceDict: missing required entry \"Obj\"")
	}

	return nil
}

func validateStructElementDictEntryKArray(xRefTable *pdf.XRefTable, arr *pdf.Array) error {

	for _, obj := range *arr {

		obj, err := xRefTable.Dereference(obj)
		if err != nil {
			return err
		}

		if obj == nil {
			continue
		}

		switch obj := obj.(type) {

		case pdf.Integer:

		case pdf.PDFDict:

			dictType := obj.Type()

			if dictType == nil || *dictType == "StructElem" {
				err = validateStructElementDict(xRefTable, &obj)
				if err != nil {
					return err
				}
				break
			}

			if *dictType == "MCR" {
				err = validateMarkedContentReferenceDict(xRefTable, &obj)
				if err != nil {
					return err
				}
				break
			}

			if *dictType == "OBJR" {
				err = validateObjectReferenceDict(xRefTable, &obj)
				if err != nil {
					return err
				}
				break
			}

			return errors.Errorf("validateStructElementDictEntryKArray: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

		default:
			return errors.New("validateStructElementDictEntryKArray: unsupported PDF object")

		}
	}

	return nil
}

func validateStructElementDictEntryK(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	// K: optional, the children of this structure element
	//
	// struct element dict
	// marked content reference dict
	// object reference dict
	// marked content id int
	// array of all above

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Integer:

	case pdf.PDFDict:

		dictType := obj.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDict(xRefTable, &obj)
			if err != nil {
				return err
			}
			break
		}

		if *dictType == "MCR" {
			err = validateMarkedContentReferenceDict(xRefTable, &obj)
			if err != nil {
				return err
			}
			break
		}

		if *dictType == "OBJR" {
			err = validateObjectReferenceDict(xRefTable, &obj)
			if err != nil {
				return err
			}
			break
		}

		return errors.Errorf("validateStructElementDictEntryK: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

	case pdf.Array:

		err = validateStructElementDictEntryKArray(xRefTable, &obj)
		if err != nil {
			return err
		}

	default:
		return errors.New("validateStructElementDictEntryK: unsupported PDF object")

	}

	return nil
}

func processStructElementDictPgEntry(xRefTable *pdf.XRefTable, indRef pdf.IndirectRef) error {

	// is this object a known page object?

	obj, err := xRefTable.Dereference(indRef)
	if err != nil {
		return errors.Errorf("processStructElementDictPgEntry: Pg obj:#%d gen:%d unknown\n", indRef.ObjectNumber, indRef.GenerationNumber)
	}

	//logInfoWriter.Printf("known object for Pg: %v %s\n", obj, obj)

	pageDict, ok := obj.(pdf.PDFDict)
	if !ok {
		return errors.Errorf("processStructElementDictPgEntry: Pg object corrupt dict: %s\n", obj)
	}

	if t := pageDict.Type(); t == nil || *t != "Page" {
		return errors.Errorf("processStructElementDictPgEntry: Pg object no pageDict: %s\n", pageDict)
	}

	return nil
}

func validateStructElementDictEntryA(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.PDFDict: // No further processing.

	case pdf.StreamDict: // No further processing.

	case pdf.Array:

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			switch obj.(type) {

			case pdf.Integer:
				// Each array element may be followed by a revision number (int).sort

			case pdf.PDFDict:
				// No further processing.

			case pdf.StreamDict:
				// No further processing.

			default:
				return errors.Errorf("validateStructElementDictEntryA: unsupported PDF object: %v\n.", obj)
			}
		}

	default:
		return errors.Errorf("validateStructElementDictEntryA: unsupported PDF object: %v\n.", obj)

	}

	return nil
}

func validateStructElementDictEntryC(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Name:
		// No further processing.

	case pdf.Array:

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			switch obj.(type) {

			case pdf.Name:
				// No further processing.

			case pdf.Integer:
				// Each array element may be followed by a revision number.

			default:
				return errors.New("validateStructElementDictEntryC: unsupported PDF object")

			}
		}

	default:
		return errors.New("validateStructElementDictEntryC: unsupported PDF object")

	}

	return nil
}

func validateStructElementDictPart1(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName string) error {

	// S: structure type, required, name, see 14.7.3 and Annex E.
	_, err := validateNameEntry(xRefTable, dict, dictName, "S", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// pl: immediate parent, required, indirect reference
	indRef := dict.IndirectRefEntry("P")
	if indRef == nil {
		return errors.Errorf("validateStructElementDict: missing entry P: %s\n", dict)
	}

	// Check if parent structure element exists.
	if _, ok := xRefTable.FindTableEntryForIndRef(indRef); !ok {
		return errors.Errorf("validateStructElementDict: unknown parent: %v\n", indRef)
	}

	// ID: optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "ID", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		err = processStructElementDictPgEntry(xRefTable, *indRef)
		if err != nil {
			return err
		}
	}

	// K: optional, the children of this structure element.
	if obj, found := dict.Find("K"); found {
		err = validateStructElementDictEntryK(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	// A: optional, attribute objects: dict or stream dict or array of these.
	if obj, ok := dict.Find("A"); ok {
		err = validateStructElementDictEntryA(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateStructElementDictPart2(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName string) error {

	// C: optional, name or array
	if obj, ok := dict.Find("C"); ok {
		err := validateStructElementDictEntryC(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	// R: optional, integer >= 0
	_, err := validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, pdf.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// T: optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Lang: optional, text string, since 1.4
	sinceVersion := pdf.V14
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V13
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "Lang", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Alt: optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// E: optional, text sttring, since 1.5
	_, err = validateStringEntry(xRefTable, dict, dictName, "E", OPTIONAL, pdf.V15, nil)
	if err != nil {
		return err
	}

	// ActualText: optional, text string, since 1.4
	_, err = validateStringEntry(xRefTable, dict, dictName, "ActualText", OPTIONAL, pdf.V14, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateStructElementDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	// See table 323

	dictName := "StructElementDict"

	err := validateStructElementDictPart1(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	err = validateStructElementDictPart2(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	return nil
}

func validateStructTreeRootDictEntryKArray(xRefTable *pdf.XRefTable, arr *pdf.Array) error {

	for _, obj := range *arr {

		obj, err := xRefTable.Dereference(obj)
		if err != nil {
			return err
		}

		if obj == nil {
			continue
		}

		switch obj := obj.(type) {

		case pdf.PDFDict:

			dictType := obj.Type()

			if dictType == nil || *dictType == "StructElem" {
				err = validateStructElementDict(xRefTable, &obj)
				if err != nil {
					return err
				}
				break
			}

			return errors.Errorf("validateStructTreeRootDictEntryKArray: invalid dictType %s (should be \"StructElem\")\n", *dictType)

		default:
			return errors.New("validateStructTreeRootDictEntryKArray: unsupported PDF object")

		}
	}

	return nil
}

func validateStructTreeRootDictEntryK(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	// The immediate child or children of the structure tree root in the structure hierarchy.
	// The value may be either a dictionary representing a single structure element or an array of such dictionaries.

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.PDFDict:

		dictType := obj.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDict(xRefTable, &obj)
			if err != nil {
				return err
			}
			break
		}

		return errors.Errorf("validateStructTreeRootDictEntryK: invalid dictType %s (should be \"StructElem\")\n", *dictType)

	case pdf.Array:

		err = validateStructTreeRootDictEntryKArray(xRefTable, &obj)
		if err != nil {
			return err
		}

	default:
		return errors.New("validateStructTreeRootDictEntryK: unsupported PDF object")

	}

	return nil
}

func processStructTreeClassMapDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	for _, obj := range dict.Dict {

		// Process dict or array of dicts.

		obj, err := xRefTable.Dereference(obj)
		if err != nil {
			return err
		}

		if obj == nil {
			continue
		}

		switch obj := obj.(type) {

		case pdf.PDFDict:
			// no further processing.

		case pdf.Array:

			for _, obj := range obj {

				_, err = xRefTable.DereferenceDict(obj)
				if err != nil {
					return err
				}

			}

		default:
			return errors.New("processStructTreeClassMapDict: unsupported PDF object")

		}

	}

	return nil
}

func validateStructTreeRootDictEntryParentTree(xRefTable *pdf.XRefTable, indRef *pdf.IndirectRef) error {

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {

		// Accept empty dict

		d, err := xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return err
		}

		if d == nil || len(d.Dict) == 0 {
			return errors.New("validateStructTreeRootDict: corrupt entry \"ParentTree\"")
		}

	} else {

		_, _, err := validateNumberTree(xRefTable, "StructTree", *indRef, true)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateStructTreeRootDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict) error {

	dictName := "StructTreeRootDict"

	// required entry Type: name:StructTreeRoot
	if dict.Type() == nil || *dict.Type() != "StructTreeRoot" {
		return errors.New("writeStructTreeRootDict: missing type")
	}

	// Optional entry K: struct element dict or array of struct element dicts
	if obj, found := dict.Find("K"); found {
		err := validateStructTreeRootDictEntryK(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	// Optional entry IDTree: name tree, key=elementId value=struct element dict
	// A name tree that maps element identifiers to the structure elements they denote.
	indRef := dict.IndirectRefEntry("IDTree")
	if indRef != nil {
		_, _, _, err := validateNameTree(xRefTable, "IDTree", *indRef, true)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTree: number tree, value=indRef of struct element dict or array of struct element dicts
	// A number tree used in finding the structure elements to which content items belong.
	if indRef = dict.IndirectRefEntry("ParentTree"); indRef != nil {
		err := validateStructTreeRootDictEntryParentTree(xRefTable, indRef)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTreeNextKey: integer
	_, err := validateIntegerEntry(xRefTable, dict, dictName, "ParentTreeNextKey", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry RoleMap: dict
	// A dictionary that shall map the names of structure used in the document
	// to their approximate equivalents in the set of standard structure
	_, err = validateDictEntry(xRefTable, dict, dictName, "RoleMap", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry ClassMap: dict
	// A dictionary that shall map name objects designating attribute classes
	// to the corresponding attribute objects or arrays of attribute objects.
	d, err := validateDictEntry(xRefTable, dict, dictName, "ClassMap", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = processStructTreeClassMapDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateStructTree(xRefTable *pdf.XRefTable, rootDict *pdf.PDFDict, required bool, sinceVersion pdf.Version) error {

	// 14.7.2 Structure Hierarchy

	dict, err := validateDictEntry(xRefTable, rootDict, "RootDict", "StructTreeRoot", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	err = validateStructTreeRootDict(xRefTable, dict)
	if err != nil {
		return err
	}

	return nil
}

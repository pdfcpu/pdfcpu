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
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var errUnsupportedPDFObject = errors.New("unsupported PDF object")

func validateMarkedContentReferenceDict(xRefTable *model.XRefTable, d types.Dict) error {
	var err error

	// Pg: optional, indirect reference
	// Page object representing a page on which the graphics object in the marked-content sequence shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err = processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return fmt.Errorf("marked content reference Pg: %w", err)
		}
	}

	// Stm: optional, indirect reference
	// The content stream containing the marked-content sequence.
	if ir := d.IndirectRefEntry("Stm"); ir != nil {
		_, err = xRefTable.Dereference(ir)
		if err != nil {
			return fmt.Errorf("marked content reference Stm: dereference: %w", err)
		}
	}

	// StmOwn: optional, indirect reference
	// The PDF object owning the stream identified by Stems annotation to which an appearance stream belongs.
	if ir := d.IndirectRefEntry("StmOwn"); ir != nil {
		_, err = xRefTable.Dereference(ir)
		if err != nil {
			return fmt.Errorf("marked content reference StmOwn: dereference: %w", err)
		}
	}

	// MCID: required, integer
	// The marked-content identifier of the marked-content sequence within its content stream.

	if d.IntEntry("MCID") == nil {
		err = errors.New("marked content reference: missing MCID")
	}

	return err
}

func validateObjectReferenceDict(xRefTable *model.XRefTable, d types.Dict) error {
	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err := processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return fmt.Errorf("object reference Pg: %w", err)
		}
	}

	// Obj: required, indirect reference
	ir := d.IndirectRefEntry("Obj")
	if xRefTable.ValidationMode == model.ValidationStrict && ir == nil {
		return errors.New("object reference: missing Obj")
	}

	if ir == nil {
		model.ShowSkipped(`objectReferenceDict: entry "Obj"`)
		return nil
	}

	obj, err := xRefTable.Dereference(*ir)
	if err != nil {
		return fmt.Errorf("object reference Obj obj#%d: dereference: %w", ir.ObjectNumber.Value(), err)
	}

	if obj == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf("objectReferenceDict: missing obj#%s", ir.ObjectNumber))
			return nil
		}
		return fmt.Errorf("object reference Obj obj#%d: missing object", ir.ObjectNumber.Value())
	}

	return nil
}

func validateStructElementKArrayElement(xRefTable *model.XRefTable, o types.Object, useIDs bool, depth int) error {
	switch o := o.(type) {

	case types.Integer:
		return nil

	case types.Dict:

		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			return validateStructElementDictDepth(xRefTable, o, useIDs, depth+1)
		}

		if *dictType == "MCR" {
			return validateMarkedContentReferenceDict(xRefTable, o)
		}

		if *dictType == "OBJR" {
			return validateObjectReferenceDict(xRefTable, o)
		}

		return fmt.Errorf("structure element K: unexpected dict Type %s, expected StructElem, OBJR or MCR", *dictType)

	}

	return fmt.Errorf("structure element K: %w: %T", errUnsupportedPDFObject, o)
}

func validateStructElementDictEntryKArray(xRefTable *model.XRefTable, a types.Array, useIDs bool, depth int) error {
	for i, o := range a {

		// Avoid recursion.
		ir, ok := o.(types.IndirectRef)
		if ok {
			valid, err := xRefTable.IsValid(ir)
			if err != nil {
				return fmt.Errorf("structure element K[%d] obj#%d: check valid: %w", i, ir.ObjectNumber.Value(), err)
			}
			if valid {
				continue
			}
			if err := xRefTable.SetValid(ir); err != nil {
				return fmt.Errorf("structure element K[%d] obj#%d: mark valid: %w", i, ir.ObjectNumber.Value(), err)
			}
		}

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return fmt.Errorf("%s: dereference: %w", objectContext(fmt.Sprintf("structure element K[%d]", i), a[i]), err)
		}

		if o == nil {
			continue
		}

		if err := validateStructElementKArrayElement(xRefTable, o, useIDs, depth); err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("structure element K[%d]", i), a[i]), err)
		}

	}

	return nil
}

func validateStructElementDictEntryK(xRefTable *model.XRefTable, o types.Object, useIDs bool, depth int) error {
	// K: optional, the children of this structure element
	//
	// struct element dict
	// marked content reference dict
	// object reference dict
	// marked content id int
	// array of all above

	rawObject := o
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("%s: dereference: %w", objectContext("structure element K", rawObject), err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Integer:

	case types.Dict:
		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDictDepth(xRefTable, o, useIDs, depth+1)
			if err != nil {
				return fmt.Errorf("%s: %w", objectContext("structure element K", rawObject), err)
			}
			break
		}

		if *dictType == "MCR" {
			err = validateMarkedContentReferenceDict(xRefTable, o)
			if err != nil {
				return fmt.Errorf("%s: %w", objectContext("structure element K", rawObject), err)
			}
			break
		}

		if *dictType == "OBJR" {
			err = validateObjectReferenceDict(xRefTable, o)
			if err != nil {
				return fmt.Errorf("%s: %w", objectContext("structure element K", rawObject), err)
			}
			break
		}

		return fmt.Errorf("%s: unexpected dict Type %s, expected StructElem, OBJR or MCR", objectContext("structure element K", rawObject), *dictType)

	case types.Array:

		err = validateStructElementDictEntryKArray(xRefTable, o, useIDs, depth)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("%s: %w: %T", objectContext("structure element K", rawObject), errUnsupportedPDFObject, o)

	}

	return nil
}

func processStructElementDictPgEntry(xRefTable *model.XRefTable, ir types.IndirectRef) error {
	// is this object a known page object?

	o, err := xRefTable.Dereference(ir)
	if err != nil {
		return fmt.Errorf("page obj#%d: dereference: %w", ir.ObjectNumber.Value(), err)
	}

	//logInfoWriter.Printf("known object for Pg: %v %s\n", obj, obj)

	if xRefTable.ValidationMode == model.ValidationRelaxed && o == nil {
		return nil
	}

	pageDict, ok := o.(types.Dict)
	if !ok {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf("invalid structElementDict Pg entry, objNr: %d ", ir.ObjectNumber))
			return nil
		}
		return fmt.Errorf("page obj#%d: expected page dict, got %T", ir.ObjectNumber.Value(), o)
	}

	if t := pageDict.Type(); t == nil || *t != "Page" {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf("invalid structElementDict Pg entry, objNr: %d ", ir.ObjectNumber))
			return nil
		}
		return fmt.Errorf("page obj#%d: expected Type Page", ir.ObjectNumber.Value())
	}

	return nil
}

func validateStructElementDictEntryA(xRefTable *model.XRefTable, o types.Object) error {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("structure element A: dereference: %w", err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Dict: // No further processing.

	case types.StreamDict: // No further processing.

	case types.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return fmt.Errorf("structure element A[%d]: dereference: %w", i, err)
			}

			if o == nil {
				continue
			}

			switch o.(type) {

			case types.Integer:
				// Each array element may be followed by a revision number (int).sort

			case types.Dict:
				// No further processing.

			case types.StreamDict:
				// No further processing.

			default:
				return fmt.Errorf("structure element A[%d]: %w: %T", i, errUnsupportedPDFObject, o)
			}
		}

	default:
		return fmt.Errorf("structure element A: %w: %T", errUnsupportedPDFObject, o)

	}

	return nil
}

func validateStructElementDictEntryC(xRefTable *model.XRefTable, o types.Object) error {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("structure element C: dereference: %w", err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Name:
		// No further processing.

	case types.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return fmt.Errorf("structure element C[%d]: dereference: %w", i, err)
			}

			if o == nil {
				continue
			}

			switch o.(type) {

			case types.Name:
				// No further processing.

			case types.Integer:
				// Each array element may be followed by a revision number.

			default:
				return fmt.Errorf("structure element C[%d]: %w: %T", i, errUnsupportedPDFObject, o)

			}
		}

	default:
		return fmt.Errorf("structure element C: %w: %T", errUnsupportedPDFObject, o)

	}

	return nil
}

func validateStructElementDictEntryP(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// P: immediate parent, required, indirect reference
	ir := d.IndirectRefEntry("P")
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		if ir == nil {
			logMissingRequiredEntry(dictName, "P", d)
			return missingRequiredEntryError(xRefTable, dictName, "P", "add parent structure element reference or validate in relaxed mode")
		}

		// Check if parent structure element exists.
		if _, ok := xRefTable.FindTableEntryForIndRef(ir); !ok {
			return fmt.Errorf("structure element parent obj#%d: unknown", ir.ObjectNumber.Value())
		}
	}

	return nil
}

func validateStructElementDictEntryPg(xRefTable *model.XRefTable, d types.Dict) error {
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err := processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return fmt.Errorf("structure element Pg: %w", err)
		}
	}
	return nil
}

func validateStructElementDictEntryS(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	_, err := validateNameEntry(xRefTable, d, dictName, "S", OPTIONAL, model.V10, nil)
	if err == nil {
		return nil
	}
	if xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}

	i, err := validateIntegerEntry(xRefTable, d, dictName, "S", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if i != nil {
		d["S"] = types.Name(strconv.Itoa((*i).Value()))
	}
	return nil
}

func validateStructElementDictPart1(xRefTable *model.XRefTable, d types.Dict, dictName string, useIDs bool, depth int) error {
	// S: structure type, required, name, see 14.7.3 and Annex E.
	if err := validateStructElementDictEntryS(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateStructElementDictEntryP(xRefTable, d, dictName); err != nil {
		return err
	}

	if useIDs {
		// ID: optional, byte string
		_, err := validateStringEntry(xRefTable, d, dictName, "ID", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}
	}

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if err := validateStructElementDictEntryPg(xRefTable, d); err != nil {
		return err
	}

	// K: optional, the children of this structure element.
	if o, found := d.Find("K"); found {
		if err := validateStructElementDictEntryK(xRefTable, o, useIDs, depth); err != nil {
			return err
		}
	}

	// A: optional, attribute objects: dict or stream dict or array of these.
	if o, ok := d.Find("A"); ok {
		if err := validateStructElementDictEntryA(xRefTable, o); err != nil {
			return err
		}
	}

	return nil
}

func validateStructElementDictPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// C: optional, name or array
	if o, ok := d.Find("C"); ok {
		err := validateStructElementDictEntryC(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// R: optional, integer >= 0
	_, err := validateIntegerEntry(xRefTable, d, dictName, "R", OPTIONAL, model.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// T: optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Lang: optional, text string, since 1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "Lang", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Alt: optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "Alt", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// E: optional, text string, since 1.5
	sinceVersion = model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "E", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// ActualText: optional, text string, since 1.4
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateStringEntry(xRefTable, d, dictName, "ActualText", OPTIONAL, sinceVersion, nil)

	return err
}

func validateStructElementDict(xRefTable *model.XRefTable, d types.Dict, useIDs bool) error {
	return validateStructElementDictDepth(xRefTable, d, useIDs, 0)
}

func validateStructElementDictDepth(xRefTable *model.XRefTable, d types.Dict, useIDs bool, depth int) error {
	if err := xRefTable.CheckRecursionDepth("structure tree", depth); err != nil {
		return err
	}

	// See table 323

	dictName := "StructElementDict"

	err := validateStructElementDictPart1(xRefTable, d, dictName, useIDs, depth)
	if err != nil {
		return err
	}

	return validateStructElementDictPart2(xRefTable, d, dictName)
}

func handleInvalidStructTreeObject(xRefTable *model.XRefTable, err error) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}
	model.ShowDigestedSpecViolationError(xRefTable, err)
	return nil
}

func validateStructTreeRootDictEntryKArray(xRefTable *model.XRefTable, a types.Array, useIDs bool) error {
	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return fmt.Errorf("%s: dereference: %w", objectContext(fmt.Sprintf("structure tree root K[%d]", i), a[i]), err)
		}

		if o == nil {
			continue
		}

		switch o := o.(type) {

		case types.Dict:

			dictType := o.Type()

			if dictType == nil || *dictType == "StructElem" {
				err = validateStructElementDictDepth(xRefTable, o, useIDs, 1)
				if err != nil {
					return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("structure tree root K[%d]", i), a[i]), err)
				}
				break
			}

			err := fmt.Errorf("%s: unexpected dict Type %s, expected StructElem", objectContext(fmt.Sprintf("structure tree root K[%d]", i), a[i]), *dictType)
			if err = handleInvalidStructTreeObject(xRefTable, err); err != nil {
				return err
			}

		default:
			err := fmt.Errorf("%s: %w: %T", objectContext(fmt.Sprintf("structure tree root K[%d]", i), a[i]), errUnsupportedPDFObject, o)
			if err = handleInvalidStructTreeObject(xRefTable, err); err != nil {
				return err
			}

		}
	}

	return nil
}

func validateStructTreeRootDictEntryK(xRefTable *model.XRefTable, o types.Object, useIDs bool) error {
	// The immediate child or children of the structure tree root in the structure hierarchy.
	// The value may be either a dictionary representing a single structure element or an array of such dictionaries.

	rawObject := o
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("%s: dereference: %w", objectContext("structure tree root K", rawObject), err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Dict:

		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDictDepth(xRefTable, o, useIDs, 1)
			if err != nil {
				return fmt.Errorf("%s: %w", objectContext("structure tree root K", rawObject), err)
			}
			break
		}

		err := fmt.Errorf("%s: unexpected dict Type %s, expected StructElem", objectContext("structure tree root K", rawObject), *dictType)
		return handleInvalidStructTreeObject(xRefTable, err)

	case types.Array:

		err = validateStructTreeRootDictEntryKArray(xRefTable, o, useIDs)
		if err != nil {
			return err
		}

	default:
		err := fmt.Errorf("%s: %w: %T", objectContext("structure tree root K", rawObject), errUnsupportedPDFObject, o)
		return handleInvalidStructTreeObject(xRefTable, err)

	}

	return nil
}

func processStructTreeClassMapDict(xRefTable *model.XRefTable, d types.Dict) error {
	for name, o := range d {

		// Process dict or array of dicts.

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return fmt.Errorf("structure tree ClassMap %s: dereference: %w", name, err)
		}

		if o == nil {
			continue
		}

		switch o := o.(type) {

		case types.Dict:
			// no further processing.

		case types.Array:

			for i, o := range o {

				_, err = xRefTable.DereferenceDict(o)
				if err != nil {
					return fmt.Errorf("structure tree ClassMap %s[%d]: dereference dict: %w", name, i, err)
				}

			}

		default:
			return fmt.Errorf("structure tree ClassMap %s: %w: %T", name, errUnsupportedPDFObject, o)

		}

	}

	return nil
}

func validateStructTreeRootDictEntryParentTree(xRefTable *model.XRefTable, ir *types.IndirectRef, useIDs bool) error {
	if xRefTable.ValidationMode == model.ValidationRelaxed {

		// Accept empty dict
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return fmt.Errorf("structure tree ParentTree obj#%d: dereference: %w", ir.ObjectNumber.Value(), err)
		}
		if d == nil || d.Len() == 0 {
			return nil
		}
	}

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return fmt.Errorf("structure tree ParentTree obj#%d: dereference: %w", ir.ObjectNumber.Value(), err)
	}

	_, _, err = validateNumberTree(xRefTable, "StructTree", d, true, useIDs)
	return err
}

func validateStructTreeRootDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "StructTreeRootDict"

	// required entry Type: name:StructTreeRoot
	if d.Type() == nil || *d.Type() != "StructTreeRoot" {
		return errors.New("structure tree root: missing Type StructTreeRoot")
	}

	useIDs := false

	// Optional entry IDTree: name tree, key=elementId value=struct element dict
	// A name tree that maps element identifiers to the structure elements they denote.
	ir := d.IndirectRefEntry("IDTree")
	if ir != nil {
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return fmt.Errorf("structure tree IDTree obj#%d: dereference: %w", ir.ObjectNumber.Value(), err)
		}
		if len(d) > 0 {
			_, _, _, err = validateNameTree(xRefTable, "IDTree", d, true)
			if err != nil {
				return fmt.Errorf("structure tree IDTree: %w", err)
			}
			useIDs = true
		}
	}

	// Optional entry K: struct element dict or array of struct element dicts
	if o, found := d.Find("K"); found {
		err := validateStructTreeRootDictEntryK(xRefTable, o, useIDs)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTree: number tree, value=indRef of struct element dict or array of struct element dicts
	// A number tree used in finding the structure elements to which content items belong.
	if ir = d.IndirectRefEntry("ParentTree"); ir != nil {
		err := validateStructTreeRootDictEntryParentTree(xRefTable, ir, useIDs)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTreeNextKey: integer
	_, err := validateIntegerEntry(xRefTable, d, dictName, "ParentTreeNextKey", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry RoleMap: dict
	// A dictionary that shall map the names of structure used in the document
	// to their approximate equivalents in the set of standard structure
	_, err = validateDictEntry(xRefTable, d, dictName, "RoleMap", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry ClassMap: dict
	// A dictionary that shall map name objects designating attribute classes
	// to the corresponding attribute objects or arrays of attribute objects.
	d1, err := validateDictEntry(xRefTable, d, dictName, "ClassMap", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = processStructTreeClassMapDict(xRefTable, d1)
	}

	return err
}

func validateStructTree(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// 14.7.2 Structure Hierarchy

	d, err := validateDictEntry(xRefTable, rootDict, "RootDict", "StructTreeRoot", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	return validateStructTreeRootDict(xRefTable, d)
}

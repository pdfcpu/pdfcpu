package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateMarkedContentReferenceDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateMarkedContentReferenceDict begin ***")

	// Pg: optional, indirect reference
	// Page object representing a page on which the graphics object in the marked-content sequence shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		logInfoValidate.Println("validateMarkedContentReferenceDict: found entry \"Pg\".")
		err = processStructElementDictPgEntry(xRefTable, *indRef)
		if err != nil {
			return
		}
	}

	// Stm: optional, indirect reference
	// The content stream containing the marked-content sequence.
	if indRef := dict.IndirectRefEntry("Stm"); indRef != nil {
		logInfoValidate.Println("validateMarkedContentReferenceDict: found entry \"Stm\".")
		_, err = xRefTable.Dereference(indRef)
		if err != nil {
			return
		}
	}

	// StmOwn: optional, indirect reference
	// The PDF object owning the stream identified by Stems annotation to which an appearance stream belongs.
	if indRef := dict.IndirectRefEntry("StmOwn"); indRef != nil {
		logInfoValidate.Println("validateMarkedContentReferenceDict: found entry \"StmOwn\".")
		_, err = xRefTable.Dereference(indRef)
		if err != nil {
			return
		}
	}

	// MCID: required, integer
	// The marked-content identifier of the marked-content sequence within its content stream.
	if obj, found := dict.Find("MCID"); !found {
		logInfoValidate.Println("validateMarkedContentReferenceDict: missing entry \"MCID\".")
	} else {
		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			logInfoValidate.Println("validateMarkedContentReferenceDict: missing entry \"MCID\".")
		}
	}

	logInfoValidate.Println("*** validateMarkedContentReferenceDict end ***")

	return
}

func validateObjectReferenceDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateObjectReferenceDict begin ***")

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		logInfoValidate.Println("validateObjectReferenceDict: found entry Pg")
		err = processStructElementDictPgEntry(xRefTable, *indRef)
		if err != nil {
			return
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

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateAnnotationDict(xRefTable, &obj)

	case types.PDFStreamDict:
		err = validateXObjectStreamDict(xRefTable, &obj)

	default:
		err = errors.Errorf("validateObjectReferenceDict: unsupported PDF object: %v", obj)

	}

	logInfoValidate.Println("*** validateObjectReferenceDict end ***")

	return
}

func validateStructElementDictEntryK(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateStructElementDictEntryK begin ***")

	// K: optional, the children of this structure element
	//
	// struct element dict
	// marked content reference dict
	// object reference dict
	// marked content id int
	// array of all above

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateStructElementDictEntryK: obj is nil, end")
		return
	}

	switch obj := obj.(type) {

	case types.PDFInteger:

	case types.PDFDict:

		dictType := obj.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDict(xRefTable, &obj)
			if err != nil {
				return
			}
			break
		}

		if *dictType == "MCR" {
			err = validateMarkedContentReferenceDict(xRefTable, &obj)
			if err != nil {
				return
			}
			break
		}

		if *dictType == "OBJR" {
			err = validateObjectReferenceDict(xRefTable, &obj)
			break
		}

		return errors.Errorf("validateStructElementDictEntryK: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

	case types.PDFArray:

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			switch obj := obj.(type) {

			case types.PDFInteger:

			case types.PDFDict:

				dictType := obj.Type()

				if dictType == nil || *dictType == "StructElem" {
					err = validateStructElementDict(xRefTable, &obj)
					if err != nil {
						return
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

			default:
				return errors.New("validateStructElementDictEntryK: unsupported PDF object")

			}
		}

	default:
		return errors.New("validateStructElementDictEntryK: unsupported PDF object")

	}

	logInfoValidate.Println("*** validateStructElementDictEntryK end ***")

	return
}

func processStructElementDictPgEntry(xRefTable *types.XRefTable, indRef types.PDFIndirectRef) (err error) {

	logInfoValidate.Println("*** processStructElementDictPgEntry begin ***")

	// is this object a known page object?

	obj, err := xRefTable.Dereference(indRef)
	if err != nil {
		return errors.Errorf("processStructElementDictPgEntry: Pg obj:#%d gen:%d unknown\n", indRef.ObjectNumber, indRef.GenerationNumber)
	}

	//logInfoWriter.Printf("known object for Pg: %v %s\n", obj, obj)

	pageDict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.Errorf("processStructElementDictPgEntry: Pg object corrupt dict: %s\n", obj)
	}

	if t := pageDict.Type(); t == nil || *t != "Page" { // or "Pages" ?
		return errors.Errorf("processStructElementDictPgEntry: Pg object no pageDict: %s\n", pageDict)
	}

	logInfoValidate.Println("*** processStructElementDictPgEntry end ***")

	return
}

func validateStructElementDictEntryA(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateStructElementDictEntryA begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateStructElementDictEntryA: obj is nil, end")
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict: // No further processing.

	case types.PDFStreamDict: // No further processing.

	case types.PDFArray:

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			switch obj.(type) {

			case types.PDFInteger: // TODO: Each array element may be followed by a revision number (int).sort

			case types.PDFDict: // No further processing.

			case types.PDFStreamDict: // No further processing.

			default:
				return errors.Errorf("validateStructElementDictEntryA: unsupported PDF object: %v\n.", obj)
			}
		}

	default:
		return errors.Errorf("validateStructElementDictEntryA: unsupported PDF object: %v\n.", obj)

	}

	logInfoValidate.Println("*** validateStructElementDictEntryA end ***")

	return
}

func validateStructElementDictEntryC(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// TODO content element unclear!

	logInfoValidate.Println("*** validateStructElementDictEntryC begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateStructElementDictEntryC: obj is nil, end")
		return
	}

	switch obj := obj.(type) {

	case types.PDFName: // No further processing.

	case types.PDFArray:

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			switch obj.(type) {

			case types.PDFName: // No further processing.

			case types.PDFInteger: // TODO: Each array element may be followed by a revision number (int).sort

			default:
				return errors.New("validateStructElementDictEntryC: unsupported PDF object")

			}
		}

	default:
		return errors.New("validateStructElementDictEntryC: unsupported PDF object")

	}

	logInfoValidate.Println("*** validateStructElementDictEntryC end ***")

	return
}

func validateStructElementDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateStructElementDict begin ***")

	dictName := "StructElementDict"

	// S: structure type, required, name, see 14.7.3 and Annex E.
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P: immediate parent, required, indirect reference
	// should already written.
	indRef := dict.IndirectRefEntry("P")
	if indRef == nil {
		return errors.Errorf("validateStructElementDict: missing entry P: %s\n", dict)
	}

	// Check if parent structure element exists.
	if _, ok := xRefTable.FindTableEntryForIndRef(indRef); !ok {
		return errors.Errorf("validateStructElementDict: unknown parent: %v\n", indRef)
	}

	// ID: optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "ID", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		logInfoValidate.Println("validateStructElementDict: found entry Pg")
		err = processStructElementDictPgEntry(xRefTable, *indRef)
		if err != nil {
			return
		}
	}

	// K: optional, the children of this structure element.
	if obj, found := dict.Find("K"); found {
		err = validateStructElementDictEntryK(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// A: optional, attribute objects: dict or stream dict or array of these.
	if obj, ok := dict.Find("A"); ok {
		err = validateStructElementDictEntryA(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// C: optional, name or array
	if obj, ok := dict.Find("C"); ok {
		err = validateStructElementDictEntryC(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// R: optional, integer >= 0
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return
	}

	// T: optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Lang: optional, text string, since 1.4
	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "Lang", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	// Alt: optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// E: optional, text sttring, since 1.5
	_, err = validateStringEntry(xRefTable, dict, dictName, "E", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	// ActualText: optional, text string, since 1.4
	_, err = validateStringEntry(xRefTable, dict, dictName, "ActualText", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateStructElementDict: end ***")

	return
}

// TODO implement OBJR
func validateStructTreeRootDictEntryK(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// The immediate child or children of the structure tree root in the structure hierarchy.
	// The value may be either a dictionary representing a single structure element or an array of such dictionaries.

	logInfoValidate.Printf("*** validateStructTreeRootDictEntryK: begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Printf("validateStructTreeRootDictEntryK: is nil, end")
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:

		dictType := obj.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDict(xRefTable, &obj)
			if err != nil {
				return
			}
			break
		}

		//if *dictType == "OBJR" {
		//	writeObjectReferenceDict(source, dest, o)
		//	break
		//}

		return errors.Errorf("validateStructTreeRootDictEntryK: invalid dictType %s (should be \"StructElem\")\n", *dictType)

	case types.PDFArray:

		for _, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			switch obj := obj.(type) {

			case types.PDFDict:

				dictType := obj.Type()

				if dictType == nil || *dictType == "StructElem" {
					err = validateStructElementDict(xRefTable, &obj)
					if err != nil {
						return
					}
					break
				}

				//if *dictType == "OBJR" {
				//	writeObjectReferenceDict(source, dest, o)
				//	break
				//}

				return errors.Errorf("validateStructTreeRootDictEntryK: invalid dictType %s (should be \"StructElem\")\n", *dictType)

			default:
				return errors.New("validateStructTreeRootDictEntryK: unsupported PDF object")

			}
		}

	default:
		return errors.New("validateStructTreeRootDictEntryK: unsupported PDF object")

	}

	logInfoValidate.Println("*** validateStructTreeRootDictEntryK: end ***")

	return
}

func processStructTreeClassMapDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** processStructTreeClassMapDict: begin ***")

	for _, obj := range dict.Dict {

		// Process dict or array of dicts.

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			logInfoValidate.Println("processStructTreeClassMapDict: end")
			continue
		}

		switch obj := obj.(type) {

		case types.PDFDict:
			// no further processing.

		case types.PDFArray:

			for _, obj := range obj {

				_, err = xRefTable.DereferenceDict(obj)
				if err != nil {
					return
				}

			}

		default:
			return errors.New("processStructTreeClassMapDict: unsupported PDF object")

		}

	}

	logInfoValidate.Println("*** processStructTreeClassMapDict: begin ***")

	return
}

func validateStructTreeRootDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateStructTreeRootDict: begin ***")

	dictName := "StructTreeRootDict"

	// required entry Type: name:StructTreeRoot
	if dict.Type() == nil || *dict.Type() != "StructTreeRoot" {
		return errors.New("writeStructTreeRootDict: missing type")
	}

	// Optional entry K: struct element dict or array of struct element dicts
	if obj, found := dict.Find("K"); found {
		err = validateStructTreeRootDictEntryK(xRefTable, obj)
		if err != nil {
			return
		}
	}

	// Optional entry IDTree: name tree, key=elementId value=struct element dict
	// A name tree that maps element identifiers to the structure elements they denote.
	// TODO Required if structure elements have element identifiers.
	indRef := dict.IndirectRefEntry("IDTree")
	if indRef != nil {
		logInfoValidate.Println("validateStructTreeRootDict: writing IDTree.")
		err = validateNameTree(xRefTable, "IDTree", *indRef, true)
		if err != nil {
			return
		}
	}

	// Optional entry ParentTree: number tree, value=indRef of struct element dict or array of struct element dicts
	// A number tree used in finding the structure elements to which content items belong.
	// TODO Required if any structure element contains content items.
	if indRef = dict.IndirectRefEntry("ParentTree"); indRef != nil {
		if xRefTable.ValidationMode == types.ValidationRelaxed {
			// Accept empty dict
			var d *types.PDFDict
			d, err = xRefTable.DereferenceDict(*indRef)
			if err != nil {
				return
			}
			if d == nil || len(d.Dict) == 0 {
				return errors.New("validateStructTreeRootDict: corrupt entry \"ParentTree\"")
			}
		} else {
			err = validateNumberTree(xRefTable, "StructTree", *indRef, true)
			if err != nil {
				return
			}
		}
	}

	// Optional entry ParentTreeNextKey: integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "ParentTreeNextKey", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Optional entry RoleMap: dict
	// A dictionary that shall map the names of structure used in the document
	// to their approximate equivalents in the set of standard structure
	_, err = validateDictEntry(xRefTable, dict, dictName, "RoleMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Optional entry ClassMap: dict
	// A dictionary that shall map name objects designating attribute classes
	// to the corresponding attribute objects or arrays of attribute objects.
	d, err := validateDictEntry(xRefTable, dict, dictName, "ClassMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = processStructTreeClassMapDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateStructTreeRootDict: end ***")

	return
}

func validateStructTree(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// 14.7.2 Structure Hierarchy

	logInfoValidate.Printf("*** validateStructTree: begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "RootDict", "StructTreeRoot", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logDebugValidate.Println("validateStructTree: dict is nil.")
		return
	}

	err = validateStructTreeRootDict(xRefTable, dict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateStructTree: end ***")

	return
}

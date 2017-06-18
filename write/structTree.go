package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func writeMarkedContentReferenceDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeMarkedContentReferenceDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Pg: optional, indirect reference
	// Page object representing a page on which the graphics object in the marked-content sequence shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		logInfoWriter.Println("writeMarkedContentReferenceDict: found entry \"Pg\".")
		err = processStructElementDictPgEntry(ctx, *indRef)
		if err != nil {
			return
		}
	}

	// Stm: optional, indirect reference
	// The content stream containing the marked-content sequence.
	if indRef := dict.IndirectRefEntry("Stm"); indRef != nil {
		logInfoWriter.Println("writeMarkedContentReferenceDict: found entry \"Stm\".")
		_, _, err = writeIndRef(ctx, *indRef)
		if err != nil {
			return
		}
	}

	// StmOwn: optional, indirect reference
	// The PDF object owning the stream identified by Stems annotation to which an appearance stream belongs.
	if indRef := dict.IndirectRefEntry("StmOwn"); indRef != nil {
		logInfoWriter.Println("writeMarkedContentReferenceDict: found entry \"StmOwn\".")
		_, _, err = writeIndRef(ctx, *indRef)
		if err != nil {
			return
		}
	}

	// MCID: required, integer
	// The marked-content identifier of the marked-content sequence within its content stream.
	if obj, found := dict.Find("MCID"); !found {
		logInfoWriter.Println("writeMarkedContentReferenceDict: missing entry \"MCID\".")
	} else {
		obj, written, err := writeObject(ctx, obj)
		if err != nil {
			return err
		}
		if obj == nil && written {
			logInfoWriter.Println("writeMarkedContentReferenceDict: missing entry \"MCID\".")
		}
	}

	logInfoWriter.Printf("*** writeMarkedContentReferenceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeObjectReferenceDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeObjectReferenceDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		logInfoWriter.Println("writeObjectReferenceDict: found entry Pg")
		err = processStructElementDictPgEntry(ctx, *indRef)
		if err != nil {
			return
		}
	}

	// Obj: required, indirect reference
	indRef := dict.IndirectRefEntry("Obj")
	if indRef == nil {
		return errors.New("writeObjectReferenceDict: missing required entry \"Obj\"")
	}

	obj, written, err := writeIndRef(ctx, *indRef)
	if err != nil {
		return err
	}

	if written {
		// object already written
		//return errors.Errorf("writeObjectReferenceDict end: offset=%d\n", dest.Offset)
		return
	}

	if obj == nil {
		// object is nil
		return errors.New("writeObjectReferenceDict: missing required entry \"Obj\"")
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writeAnnotationDict(ctx, obj)

	case types.PDFStreamDict:
		err = writeXObjectStreamDict(ctx, obj)

	default:
		err = errors.New("writeObjectReferenceDict: unsupported PDF object")

	}

	logInfoWriter.Printf("*** writeObjectReferenceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStructElementDictEntryK(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeStructElementDictEntryK begin: offset=%d ***\n", ctx.Write.Offset)

	// K: optional, the children of this structure element
	//
	// struct element dict
	// marked content reference dict
	// object reference dict
	// marked content id int
	// array of all above

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeStructElementDictEntryK: already written, end offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeStructElementDictEntryK: obj is nil, end offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFInteger:

	case types.PDFDict:

		dictType := obj.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = writeStructElementDict(ctx, obj)
			if err != nil {
				return
			}
			break
		}

		if *dictType == "MCR" {
			err = writeMarkedContentReferenceDict(ctx, obj)
			if err != nil {
				return
			}
			break
		}

		if *dictType == "OBJR" {
			err = writeObjectReferenceDict(ctx, obj)
			break
		}

		return errors.Errorf("writeStructElementDictEntryK: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

	case types.PDFArray:

		for _, obj := range obj {

			obj, written, err = writeObject(ctx, obj)
			if err != nil {
				return err
			}

			if written || obj == nil {
				continue
			}

			switch obj := obj.(type) {

			case types.PDFInteger:

			case types.PDFDict:

				dictType := obj.Type()

				if dictType == nil || *dictType == "StructElem" {
					err = writeStructElementDict(ctx, obj)
					if err != nil {
						return
					}
					break
				}

				if *dictType == "MCR" {
					err = writeMarkedContentReferenceDict(ctx, obj)
					if err != nil {
						return err
					}
					break
				}

				if *dictType == "OBJR" {
					err = writeObjectReferenceDict(ctx, obj)
					if err != nil {
						return err
					}
					break
				}

				return errors.Errorf("writeStructElementDictEntryK: invalid dictType %s (should be \"StructElem\" or \"OBJR\" or \"MCR\")\n", *dictType)

			default:
				return errors.New("writeStructElementDictEntryK: unsupported PDF object")

			}
		}

	default:
		return errors.New("writeStructElementDictEntryK: unsupported PDF object")

	}

	logInfoWriter.Printf("*** writeStructElementDictEntryK end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func processStructElementDictPgEntry(ctx *types.PDFContext, indRef types.PDFIndirectRef) (err error) {

	logInfoWriter.Printf("*** processStructElementDictPgEntry begin: offset=%d ***\n", ctx.Write.Offset)

	// is this object a known page object?

	obj, err := ctx.Dereference(indRef)
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

	if !ctx.Write.HasWriteOffset(int(indRef.ObjectNumber)) {
		return errors.New("processStructElementDictPgEntry: Pg object not written")
	}

	logInfoWriter.Printf("*** processStructElementDictPgEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStructElementDictEntryA(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeStructElementDictEntryA begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeStructElementDictEntryA: already written, end offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeStructElementDictEntryA: obj is nil, end offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict: // No further processing.

	case types.PDFStreamDict: // No further processing.

	case types.PDFArray:

		for _, obj := range obj {

			obj, written, err = writeObject(ctx, obj)
			if err != nil {
				return err
			}

			if written || obj == nil {
				continue
			}

			switch obj.(type) {

			case types.PDFInteger: // TODO: Each array element may be followed by a revision number (int).sort

			case types.PDFDict: // No further processing.

			case types.PDFStreamDict: // No further processing.

			default:
				return errors.Errorf("writeStructElementDictEntryA: unsupported PDF object: %v\n.", obj)
			}
		}

	default:
		return errors.Errorf("writeStructElementDictEntryA: unsupported PDF object: %v\n.", obj)

	}

	logInfoWriter.Printf("*** writeStructElementDictEntryA end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStructElementDictEntryC(ctx *types.PDFContext, obj interface{}) (err error) {

	// TODO content element unclear!

	logInfoWriter.Printf("*** writeStructElementDictEntryC begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeStructElementDictEntryC: already written, end offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeStructElementDictEntryC: obj is nil, end offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName: // No further processing.

	case types.PDFArray:

		for _, obj := range obj {

			obj, written, err := writeObject(ctx, obj)
			if err != nil {
				return err
			}

			if written || obj == nil {
				continue
			}

			switch obj.(type) {

			case types.PDFName: // No further processing.

			case types.PDFInteger: // TODO: Each array element may be followed by a revision number (int).sort

			default:
				return errors.New("writeStructElementDictEntryC: unsupported PDF object")

			}
		}

	default:
		return errors.New("writeStructElementDictEntryC: unsupported PDF object")

	}

	logInfoWriter.Printf("*** writeStructElementDictEntryC end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStructElementDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeStructElementDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "StructElementDict"

	// S: structure type, required, name, see 14.7.3 and Annex E.
	_, _, err = writeNameEntry(ctx, dict, dictName, "S", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// P: immediate parent, required, indirect reference
	// should already written.
	indRef := dict.IndirectRefEntry("P")
	if indRef == nil {
		return errors.Errorf("writeStructElementDict: missing entry P: %s\n", dict)
	}

	// Check if parent structure element exists.
	if _, ok := ctx.FindTableEntryForIndRef(indRef); !ok {
		return errors.Errorf("writeStructElementDict: unknown parent: %v\n", indRef)
	}

	// Check if parent structure element already written.
	if !ctx.Write.HasWriteOffset(int(indRef.ObjectNumber)) {
		return errors.Errorf("writeStructElementDict: parent not written: %v\n", indRef)
	}

	// ID: optional, byte string
	_, _, err = writeStringEntry(ctx, dict, dictName, "ID", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if indRef := dict.IndirectRefEntry("Pg"); indRef != nil {
		logInfoWriter.Println("writeStructElementDict: found entry Pg")
		err = processStructElementDictPgEntry(ctx, *indRef)
		if err != nil {
			return
		}
	}

	// K: optional, the children of this structure element.
	if obj, found := dict.Find("K"); found {
		err = writeStructElementDictEntryK(ctx, obj)
		if err != nil {
			return
		}
	}

	// A: optional, attribute objects: dict or stream dict or array of these.
	if obj, ok := dict.Find("A"); ok {
		err = writeStructElementDictEntryA(ctx, obj)
		if err != nil {
			return
		}
	}

	// C: optional, name or array
	if obj, ok := dict.Find("C"); ok {
		err = writeStructElementDictEntryC(ctx, obj)
		if err != nil {
			return
		}
	}

	// R: optional, integer >= 0
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "R", OPTIONAL, types.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return
	}

	// T: optional, text string
	_, _, err = writeStringEntry(ctx, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Lang: optional, text string, since 1.4
	_, _, err = writeStringEntry(ctx, dict, dictName, "Lang", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	// Alt: optional, text string
	_, _, err = writeStringEntry(ctx, dict, dictName, "Alt", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// E: optional, text sttring, since 1.5
	_, _, err = writeStringEntry(ctx, dict, dictName, "E", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	// ActualText: optional, text string, since 1.4
	_, _, err = writeStringEntry(ctx, dict, dictName, "ActualText", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeStructElementDict: end offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement OBJR
func writeStructTreeRootDictEntryK(ctx *types.PDFContext, obj interface{}) (err error) {

	// The immediate child or children of the structure tree root in the structure hierarchy.
	// The value may be either a dictionary representing a single structure element or an array of such dictionaries.

	logInfoWriter.Printf("*** writeStructTreeRootDictEntryK: begin offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeStructTreeRootDictEntryK: already written, end offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeStructTreeRootDictEntryK: is nil, end offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:

		dictType := obj.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = writeStructElementDict(ctx, obj)
			if err != nil {
				return
			}
			break
		}

		//if *dictType == "OBJR" {
		//	writeObjectReferenceDict(source, dest, o)
		//	break
		//}

		return errors.Errorf("writeStructTreeRootDictEntryK: invalid dictType %s (should be \"StructElem\")\n", *dictType)

	case types.PDFArray:

		for _, obj := range obj {

			obj, _, _ := writeObject(ctx, obj)
			if obj == nil {
				continue
			}

			switch obj := obj.(type) {

			case types.PDFDict:

				dictType := obj.Type()

				if dictType == nil || *dictType == "StructElem" {
					err = writeStructElementDict(ctx, obj)
					if err != nil {
						return
					}
					break
				}

				//if *dictType == "OBJR" {
				//	writeObjectReferenceDict(source, dest, o)
				//	break
				//}

				return errors.Errorf("writeStructTreeRootDictEntryK: invalid dictType %s (should be \"StructElem\")\n", *dictType)

			default:
				return errors.New("writeStructTreeRootDictEntryK: unsupported PDF object")

			}
		}

	default:
		return errors.New("writeStructTreeRootDictEntryK: unsupported PDF object")

	}

	logInfoWriter.Printf("*** writeStructTreeRootDictEntryK: end offset=%d ***\n", ctx.Write.Offset)

	return
}

func processStructTreeClassMapDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** processStructTreeClassMapDict: begin offset=%d ***\n", ctx.Write.Offset)

	for _, obj := range dict.Dict {

		// Process dict or array of dicts.

		obj, written, err := writeObject(ctx, obj)
		if err != nil {
			return err
		}

		if written || obj == nil {
			logInfoWriter.Printf("processStructTreeClassMapDict: end offset=%d\n", ctx.Write.Offset)
			continue
		}

		switch obj := obj.(type) {

		case types.PDFDict:
			// no further processing.

		case types.PDFArray:

			for _, obj := range obj {

				obj, written, err := writeObject(ctx, obj)
				if err != nil {
					return err
				}

				if written || obj == nil {
					continue
				}

				if _, ok := obj.(types.PDFDict); !ok {
					return errors.New("processStructTreeClassMapDict: unsupported PDF object")
				}

			}

		default:
			return errors.New("processStructTreeClassMapDict: unsupported PDF object")

		}

	}

	logInfoWriter.Printf("*** processStructTreeClassMapDict: begin offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStructTreeRootDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeStructTreeRootDict: begin offset=%d ***\n", ctx.Write.Offset)

	// required entry Type: name:StructTreeRoot
	if dict.Type() == nil || *dict.Type() != "StructTreeRoot" {
		return errors.New("writeStructTreeRootDict: missing type")
	}

	// Optional entry K: struct element dict or array of struct element dicts
	if obj, found := dict.Find("K"); found {
		err = writeStructTreeRootDictEntryK(ctx, obj)
		if err != nil {
			return
		}
	}

	// Optional entry IDTree: name tree, key=elementId value=struct element dict
	// A name tree that maps element identifiers to the structure elements they denote.
	// TODO Required if structure elements have element identifiers.
	indRef := dict.IndirectRefEntry("IDTree")
	if indRef != nil {
		logInfoWriter.Printf("writeStructTree: writing IDTree\n")
		err = writeNameTree(ctx, "IDTree", *indRef, true)
		if err != nil {
			return
		}
	}

	// Optional entry ParentTree: number tree, value=indRef of struct element dict or array of struct element dicts
	// A number tree used in finding the structure elements to which content items belong.
	// TODO Required if any structure element contains content items.
	if indRef = dict.IndirectRefEntry("ParentTree"); indRef != nil {
		err = writeNumberTree(ctx, "StructTree", *indRef, true)
		if err != nil {
			return
		}
	}

	// Optional entry ParentTreeNextKey: integer
	_, _, err = writeIntegerEntry(ctx, dict, "StructTreeRootDict", "ParentTreeNextKey", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Optional entry RoleMap: dict
	// A dictionary that shall map the names of structure used in the document
	// to their approximate equivalents in the set of standard structure
	_, written, err := writeDictEntry(ctx, dict, "StructTreeRootDict", "RoleMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Optional entry ClassMap: dict
	// A dictionary that shall map name objects designating attribute classes
	// to the corresponding attribute objects or arrays of attribute objects.
	d, written, err := writeDictEntry(ctx, dict, "StructTreeRootDict", "ClassMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = processStructTreeClassMapDict(ctx, *d)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeStructTreeRootDict: end offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStructTree(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// 14.7.2 Structure Hierarchy

	logDebugWriter.Printf("*** writeStructTree: begin offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "RootDict", "StructTreeRoot", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logDebugWriter.Printf("writeStructTree: dict already written, end offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logDebugWriter.Printf("writeStructTree: dict is nil, end offset=%d\n", ctx.Write.Offset)
		return
	}

	err = writeStructTreeRootDict(ctx, *dict)
	if err != nil {
		return
	}

	logDebugWriter.Printf("*** writeStructTree: end offset=%d ***\n", ctx.Write.Offset)

	ctx.Stats.AddRootAttr(types.RootStructTreeRoot)

	return
}

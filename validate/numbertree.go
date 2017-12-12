package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdfcpu/types"
)

func validatePageLabelDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// see 12.4.2 Page Labels

	logInfoValidate.Println("*** validatePageLabelDict: begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validatePageLabelDict: end, obj is nil")
		return
	}

	if dict.Type() != nil && *dict.Type() != "PageLabel" {
		return errors.New("validatePageLabelDict: wrong type")
	}

	// Optional name entry S
	// The numbering style that shall be used for the numeric portion of each page label.
	_, err = validateNameEntry(xRefTable, dict, " pageLabelDict", "S", OPTIONAL, types.V10, validatePageLabelDictEntryS)
	if err != nil {
		return
	}

	// Optional string entry P
	// Label prefix for page labels in this range.
	_, err = validateStringEntry(xRefTable, dict, "pageLabelDict", "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Optional integer entry St
	// The value of the numeric portion for the first page label in the range.
	_, err = validateIntegerEntry(xRefTable, dict, "pageLabelDict", "St", OPTIONAL, types.V10, func(i int) bool { return i >= 1 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageLabelDict: end ***")

	return
}

func validateNumberTreeDictNumsEntry(xRefTable *types.XRefTable, dict *types.PDFDict, name string) (firstKey, lastKey int, err error) {

	logInfoValidate.Println("*** validateNumberTreeDictNumsEntry begin ***")

	// Nums: array of the form [key1 value1 key2 value2 ... key n value n]
	obj, found := dict.Find("Nums")
	if !found {
		return 0, 0, errors.New("writeNumberTreeDictNumsEntry: missing \"Kids\" or \"Nums\" entry")
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if arr == nil {
		return 0, 0, errors.New("validateNumberTreeDictNumsEntry: missing \"Nums\" array")
	}

	logInfoValidate.Println("validateNumberTreeDictNumsEntry: \"Nums\": now writing value objects")

	// arr length needs to be even because of contained key value pairs.
	if len(*arr)%2 == 1 {
		return 0, 0, errors.Errorf("validateNumberTreeDictNumsEntry: Nums array entry length needs to be even, length=%d\n", len(*arr))
	}

	// every other entry is a value
	// value = indRef to an array of indRefs of structElemDicts
	// or
	// value = indRef of structElementDict.

	for i, obj := range *arr {

		if i%2 == 0 {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			i, ok := obj.(types.PDFInteger)
			if !ok {
				return 0, 0, errors.Errorf("validateNumberTreeDictNumsEntry: corrupt key <%v>\n", obj)
			}

			if firstKey == 0 {
				firstKey = i.Value()
			}

			lastKey = i.Value()

			continue
		}

		logDebugValidate.Printf("validateNumberTreeDictNumsEntry: Nums array value: %v\n", obj)

		switch name {

		case "PageLabel":
			err = validatePageLabelDict(xRefTable, obj)
			if err != nil {
				return
			}

		case "StructTree":
			err = validateStructTreeRootDictEntryK(xRefTable, obj)
			if err != nil {
				return
			}
		}

	}

	logInfoValidate.Printf("*** validateNumberTreeDictNumsEntry end ***")

	return
}

func validateNumberTree(xRefTable *types.XRefTable, name string, indRef types.PDFIndirectRef, root bool) (firstKey, lastKey int, err error) {

	logInfoValidate.Printf("*** validateNumberTree: %s, rootObj#:%d ***\n", name, indRef.ObjectNumber)

	// A node has "Kids" or "Nums" entry.

	var dict *types.PDFDict

	dict, err = xRefTable.DereferenceDict(indRef)
	if err != nil || dict == nil {
		return
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		var arr *types.PDFArray

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return
		}

		if arr == nil {
			return 0, 0, errors.New("validateNumberTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			logInfoValidate.Printf("validateNumberTree: processing kid: %v\n", obj)

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return 0, 0, errors.New("validateNumberTree: corrupt kid, should be indirect reference")
			}

			var fk int
			fk, lastKey, err = validateNumberTree(xRefTable, name, kid, false)
			if err != nil {
				return
			}
			if firstKey == 0 {
				firstKey = fk
			}
		}

		if !root {
			arr, err = validateIntegerArrayEntry(xRefTable, dict, "numberTreeDict", "Limits", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
			if err != nil {
				return
			}
			fk, _ := (*arr)[0].(types.PDFInteger)
			lk, _ := (*arr)[1].(types.PDFInteger)
			if firstKey != fk.Value() || lastKey != lk.Value() {
				err = errors.Errorf("validateNumberTree: %s intermediate node corrupted: %d %d %d %d\n", name, firstKey, fk.Value(), lastKey, lk.Value())
			}
		}

		logInfoValidate.Printf("validateNumberTree end: %s\n", name)

		return
	}

	// Leaf node

	firstKey, lastKey, err = validateNumberTreeDictNumsEntry(xRefTable, dict, name)
	if err != nil {
		return
	}

	if !root {
		var arr *types.PDFArray
		arr, err = validateIntegerArrayEntry(xRefTable, dict, "numberTreeDict", "Limits", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
		if err != nil {
			return
		}
		fk, _ := (*arr)[0].(types.PDFInteger)
		lk, _ := (*arr)[1].(types.PDFInteger)
		if firstKey != fk.Value() || lastKey != lk.Value() {
			err = errors.Errorf("validateNumberTree: leaf node corrupted\n")
		}
	}

	logInfoValidate.Printf("*** validateNumberTree end: %s ***\n", name)

	return
}

package validate

import (
	"github.com/pkg/errors"

	"github.com/hhrutter/pdfcpu/types"
)

func validatePageLabelDict(xRefTable *types.XRefTable, obj types.PDFObject) error {

	// see 12.4.2 Page Labels

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	dictName := "pageLabelDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "PageLabel" })
	if err != nil {
		return err
	}

	// Optional name entry S
	// The numbering style that shall be used for the numeric portion of each page label.
	validate := func(s string) bool { return memberOf(s, []string{"D", "R", "r", "A", "a"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, validate)
	if err != nil {
		return err
	}

	// Optional string entry P
	// Label prefix for page labels in this range.
	_, err = validateStringEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Optional integer entry St
	// The value of the numeric portion for the first page label in the range.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "St", OPTIONAL, types.V10, func(i int) bool { return i >= 1 })

	return err
}

func validateNumberTreeDictNumsEntry(xRefTable *types.XRefTable, dict *types.PDFDict, name string) (firstKey, lastKey int, err error) {

	// Nums: array of the form [key1 value1 key2 value2 ... key n value n]
	obj, found := dict.Find("Nums")
	if !found {
		return 0, 0, errors.New("writeNumberTreeDictNumsEntry: missing \"Kids\" or \"Nums\" entry")
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return 0, 0, err
	}
	if arr == nil {
		return 0, 0, errors.New("validateNumberTreeDictNumsEntry: missing \"Nums\" array")
	}

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
				return 0, 0, err
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

		switch name {

		case "PageLabel":
			err = validatePageLabelDict(xRefTable, obj)
			if err != nil {
				return 0, 0, err
			}

		case "StructTree":
			err = validateStructTreeRootDictEntryK(xRefTable, obj)
			if err != nil {
				return 0, 0, err
			}
		}

	}

	return firstKey, lastKey, nil
}

func validateNumberTreeDictLimitsEntry(xRefTable *types.XRefTable, dict *types.PDFDict, firstKey, lastKey int) error {

	arr, err := validateIntegerArrayEntry(xRefTable, dict, "numberTreeDict", "Limits", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	fk, _ := (*arr)[0].(types.PDFInteger)
	lk, _ := (*arr)[1].(types.PDFInteger)

	if firstKey != fk.Value() || lastKey != lk.Value() {
		return errors.Errorf("validateNumberTreeDictLimitsEntry: leaf node corrupted\n")
	}

	return nil
}

func validateNumberTree(xRefTable *types.XRefTable, name string, indRef types.PDFIndirectRef, root bool) (firstKey, lastKey int, err error) {

	// A node has "Kids" or "Nums" entry.

	dict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || dict == nil {
		return 0, 0, err
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		var arr *types.PDFArray

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return 0, 0, err
		}
		if arr == nil {
			return 0, 0, errors.New("validateNumberTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return 0, 0, errors.New("validateNumberTree: corrupt kid, should be indirect reference")
			}

			var fk int
			fk, lastKey, err = validateNumberTree(xRefTable, name, kid, false)
			if err != nil {
				return 0, 0, err
			}
			if firstKey == 0 {
				firstKey = fk
			}
		}

	} else {

		// Leaf node
		firstKey, lastKey, err = validateNumberTreeDictNumsEntry(xRefTable, dict, name)
		if err != nil {
			return 0, 0, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNumberTreeDictLimitsEntry(xRefTable, dict, firstKey, lastKey)
		if err != nil {
			return 0, 0, err
		}

	}

	return firstKey, lastKey, nil
}

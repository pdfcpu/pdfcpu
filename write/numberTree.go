package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writePageLabelDict(ctx *types.PDFContext, obj interface{}) (err error) {

	// see 12.4.2 Page Labels

	logInfoWriter.Printf("*** writePageLabelDict: begin offset:%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageLabelDict: already written end offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writePageLabelDict: obj is nil end offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writePageLabelDict: corrupt dict")
	}

	if dict.Type() != nil && *dict.Type() != "PageLabel" {
		return errors.New("writePageLabelDict: wrong type")
	}

	// Optional name entry S
	// The numbering style that shall be used for the numeric portion of each page label.
	_, _, err = writeNameEntry(ctx, dict, " pageLabelDict", "S", OPTIONAL, types.V10, validate.PageLabelDictEntryS)
	if err != nil {
		return
	}

	// Optional string entry P
	// Label prefix for page labels in this range.
	_, _, err = writeStringEntry(ctx, dict, "pageLabelDict", "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Optional integer entry St
	// The value of the numeric portion for the first page label in the range.
	_, _, err = writeIntegerEntry(ctx, dict, "pageLabelDict", "St", OPTIONAL, types.V10, func(i int) bool { return i >= 1 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writePageLabelDict: end offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeNumberTreeDictNumsEntry(ctx *types.PDFContext, dict types.PDFDict, name string) (err error) {

	logInfoWriter.Printf("*** writeNumberTreeDictNumsEntry begin: offset:%d ***\n", ctx.Write.Offset)

	// Nums: array of the form [key1 value1 key2 value2 ... keyn valuen]
	// key: int
	// value: indRef of structElemDict or array of indRefs of structElemDicts.
	obj, found := dict.Find("Nums")
	if !found {
		return errors.New("writeNumberTreeDictNumsEntry: missing \"Kids\" or \"Nums\" entry")
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNumberTreeDictNumsEntry end: offset:%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		return errors.New("writeNumberTreeDictNumsEntry: missing \"Nums\" array")
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		return errors.New("writeNumberTreeDictNumsEntry: corrupt array entry \"Nums\"")
	}

	logInfoWriter.Printf("writeNumberTreeDictNumsEntry: \"Nums\": now writing value objects")

	// arr length needs to be even because of contained key value pairs.
	if len(arr)%2 == 1 {
		return errors.Errorf("writeNumberTreeDictNumsEntry: Nums array entry length needs to be even, length=%d\n", len(arr))
	}

	// every other entry is a value
	// value = indRef to an array of indRefs of structElemDicts
	// or
	// value = indRef of structElementDict.

	for i, obj := range arr {

		if i%2 == 0 {
			continue
		}

		logDebugWriter.Printf("writeNumberTreeDictNumsEntry: Nums array value: %v\n", obj)

		switch name {

		case "PageLabel":
			err = writePageLabelDict(ctx, obj)
			if err != nil {
				return
			}

		case "StructTree":
			err = writeStructTreeRootDictEntryK(ctx, obj)
			if err != nil {
				return
			}
		}

	}

	logInfoWriter.Printf("*** writeNumberTreeDictNumsEntry end: offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeNumberTreeDictLimitsEntry(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeNumberTreeDictLimitsEntry begin: offset:%d ***\n", ctx.Write.Offset)

	// An array of two integers, that shall specify the
	// numerically least and greatest keys included in the "Nums" array.

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNumberTreeDictLimitsEntry end: offset:%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		return errors.New("writeNumberTreeDictLimitsEntry: missing \"Limits\" array")
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		return errors.New("writeNumberTreeDictLimitsEntry: corrupt array entry \"Limits\"")
	}

	if len(arr) != 2 {
		return errors.New("writeNumberTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	// TODO process indref PDFInteger?

	if _, ok := arr[0].(types.PDFInteger); !ok {
		return errors.New("writeNumberTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	if _, ok := arr[1].(types.PDFInteger); !ok {
		return errors.New("writeNumberTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	logInfoWriter.Printf("*** writeNumberTreeDictLimitsEntry end: offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeNumberTree(ctx *types.PDFContext, name string, indRef types.PDFIndirectRef, root bool) (err error) {

	logInfoWriter.Printf("*** writeNumberTree: %s, rootObj#:%d at offset:%d ***\n", name, indRef.ObjectNumber, ctx.Write.Offset)

	// A node has "Kids" or "Nums" entry.

	obj, written, err := writeIndRef(ctx, indRef)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNumberTree end: %s, offset:%d\n", name, ctx.Write.Offset)
		return
	}

	if obj == nil {
		return errors.New("writeNumberTree: missing dict")
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeNumberTree: corrupt dict")
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written {
			logInfoWriter.Printf("writeNumberTree end: already written, %s, offset:%d\n", name, ctx.Write.Offset)
			return
		}

		if obj == nil {
			return errors.New("writeNumberTree: missing \"Kids\" array")
		}

		arr, ok := obj.(types.PDFArray)
		if !ok {
			return errors.New("writeNumberTree: corrupt array entry \"Kids\"")
		}

		for _, obj := range arr {

			logInfoWriter.Printf("writeNumberTree: processing kid: %v\n", obj)

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return errors.New("writeNumberTree: corrupt kid, should be indirect reference")
			}

			err = writeNumberTree(ctx, name, kid, false)
			if err != nil {
				return
			}
		}

		logInfoWriter.Printf("writeNumberTree end: %s, offset:%d\n", name, ctx.Write.Offset)

		return
	}

	err = writeNumberTreeDictNumsEntry(ctx, dict, name)
	if err != nil {
		return
	}

	if !root {

		obj, found := dict.Find("Limits")
		if !found {
			return errors.New("writeNumberTree: missing \"Limits\" entry")
		}

		err = writeNumberTreeDictLimitsEntry(ctx, obj)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeNumberTree end: %s, offset:%d ***\n", name, ctx.Write.Offset)

	return
}

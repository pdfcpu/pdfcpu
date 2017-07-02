package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func writeDestsNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeDestsNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = writeDestination(ctx, obj)
	logInfoWriter.Printf("*** writeDestsNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeAPNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("writeAPNameTreeValue: begin offset:%d\n", ctx.Write.Offset)
	err = errors.New("*** writeAPNameTreeValue: unsupported ***")
	logInfoWriter.Printf("writeAPNameTreeValue: end offset:%d\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeJavaScriptNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeJavaScriptNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeJavaScriptNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeJavaScriptNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writePagesNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writePagesNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writePagesNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writePagesNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeTemplatesNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeTemplatesNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeTemplatesNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeTemplatesNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeIDSTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeIDSTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeIDSTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeIDSTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeURLSNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeURLSNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeURLSNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeURLSNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeEmbeddedFilesNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeEmbeddedFilesNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeEmbeddedFilesNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeEmbeddedFilesNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeAlternatePresentationsNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeAlternatePresentationsNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeAlternatePresentationsNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeAlternatePresentationsNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeRenditionsNameTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {
	logInfoWriter.Printf("*** writeRenditionsNameTreeValue: begin offset:%d ***\n", ctx.Write.Offset)
	err = errors.New("*** writeRenditionsNameTreeValue: unsupported ***")
	logInfoWriter.Printf("*** writeRenditionsNameTreeValue: end offset:%d ***\n", ctx.Write.Offset)
	return
}

// TODO OBJR
func writeIDTreeValue(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeIDTreeValue: begin offset:%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeIDTreeValue: already written, end offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeIDTreeValue: is nil, end offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeIDTreeValue: invalid type, should be dict")
	}

	dictType := dict.Type()
	if dictType == nil || *dictType == "StructElem" {
		err = writeStructElementDict(ctx, dict)
		if err != nil {
			return
		}
	} else {
		return errors.Errorf("writeIDTreeValue: invalid dictType %s (should be \"StructElem\")\n", *dictType)
	}

	//if *dictType == "OBJR" {
	//    writeObjectReferenceDict(ctx, dict)
	//    break
	//}

	logInfoWriter.Printf("*** writeIDTreeValue: end offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeNameTreeDictNamesEntry(ctx *types.PDFContext, dict types.PDFDict, name string) (err error) {

	logInfoWriter.Printf("*** writeNameTreeDictNamesEntry begin: name:%s offset:%d ***\n", name, ctx.Write.Offset)

	// Names: array of the form [key1 value1 key2 value2 ... keyn valuen]
	obj, found := dict.Find("Names")
	if !found {
		return errors.Errorf("writeNameTreeDictNamesEntry: missing \"Kids\" or \"Names\" entry.")
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNameTreeDictNamesEntry end: offset:%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		return errors.Errorf("writeNameTreeDictNamesEntry: missing \"Names\" array.")
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		return errors.Errorf("writeNameTreeDictNamesEntry: corrupt array entry \"Names\".")
	}

	logInfoWriter.Printf("writeNameTreeDictNamesEntry: \"Nums\": now writing value objects")

	// arr length needs to be even because of contained key value pairs.
	if len(arr)%2 == 1 {
		return errors.Errorf("writeNameTreeDictNamesEntry: Names array entry length needs to be even, length=%d\n", len(arr))
	}

	for i, obj := range arr {

		if i%2 == 0 {
			continue
		}

		logDebugWriter.Printf("writeNameTreeDictNamesEntry: Nums array value: %v\n", obj)

		switch name {

		case "Dests":
			err = writeDestsNameTreeValue(ctx, obj)

		case "AP":
			err = writeAPNameTreeValue(ctx, obj)

		case "JavaScript":
			err = writeJavaScriptNameTreeValue(ctx, obj)

		case "Pages":
			err = writePagesNameTreeValue(ctx, obj)

		case "Templates":
			err = writeTemplatesNameTreeValue(ctx, obj)

		case "IDS":
			err = writeIDSTreeValue(ctx, obj)

		case "URLS":
			err = writeURLSNameTreeValue(ctx, obj)

		case "EmbeddedFiles":
			err = writeEmbeddedFilesNameTreeValue(ctx, obj)

		case "AlternatePresentations":
			err = writeAlternatePresentationsNameTreeValue(ctx, obj)

		case "Renditions":
			err = writeRenditionsNameTreeValue(ctx, obj)

		case "IDTree":
			err = writeIDTreeValue(ctx, obj)

		default:
			err = errors.Errorf("writeNameTreeDictNamesEntry: unknow dict name: %s", name)

		}

	}

	logInfoWriter.Printf("*** writeNameTreeDictNamesEntry end: offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeNameTreeDictLimitsEntry(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeNameTreeDictLimitsEntry begin: offset:%d ***\n", ctx.Write.Offset)

	// An array of two integers, that shall specify the
	// numerically least and greatest keys included in the "Nums" array.

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNameTreeDictLimitsEntry end: already written, offset:%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		return errors.New("writeNameTreeDictLimitsEntry: missing \"Limits\" array")
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		return errors.New("writeNameTreeDictLimitsEntry: corrupt array entry \"Limits\"")
	}

	if len(arr) != 2 {
		return errors.New("writeNameTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	// TODO process indref PDFStringLiteral?

	if _, ok := arr[0].(types.PDFStringLiteral); !ok {
		return errors.New("writeNameTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	if _, ok := arr[1].(types.PDFStringLiteral); !ok {
		return errors.New("writeNameTreeDictLimitsEntry: corrupt array entry \"Limits\" expected to contain 2 integers")
	}

	logInfoWriter.Printf("*** writeNameTreeDictLimitsEntry end: offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeNameTree(ctx *types.PDFContext, name string, indRef types.PDFIndirectRef, root bool) (err error) {

	logInfoWriter.Printf("*** writeNameTree: %s, at offset:%d ***\n", name, ctx.Write.Offset)

	// Rootnode has "Kids" or "Names" entry.
	// Kids: array of indirect references to the immediate children of this node.
	// Names: array of the form [key1 value1 key2 value2 ... keyn valuen]
	// key: string
	// value: indRef or the object associated with the key.
	// if Kids present then recurse

	obj, written, err := writeIndRef(ctx, indRef)
	if err != nil {
		return err
	}

	if written || obj == nil {
		logInfoWriter.Printf("writeNameTree end: %s, offset:%d\n", name, ctx.Write.Offset)
		return nil
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeNameTree: corrupt dict")
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		obj, _, err = writeObject(ctx, obj)
		if obj == nil {
			if !ok {
				return errors.New("writeNameTree: missing \"Kids\" array")
			}

			logInfoWriter.Printf("writeNameTree end: %s, offset:%d\n", name, ctx.Write.Offset)
			return nil
		}

		arr, ok := obj.(types.PDFArray)
		if !ok {
			return errors.New("writeNameTree: corrupt array entry \"Kids\"")
		}

		for _, obj := range arr {

			logInfoWriter.Printf("writeNameTree: processing kid: %v\n", obj)

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return errors.New("writeNameTree: corrupt kid, should be indirect reference")
			}

			err = writeNameTree(ctx, name, kid, false)
			if err != nil {
				return
			}
		}

		logInfoWriter.Printf("writeNameTree end: %s, offset:%d\n", name, ctx.Write.Offset)
		return nil
	}

	err = writeNameTreeDictNamesEntry(ctx, dict, name)
	if err != nil {
		return
	}

	if !root {

		obj, found := dict.Find("Limits")
		if !found {
			return errors.New("writeNameTree: missing \"Limits\" entry")
		}

		err = writeNameTreeDictLimitsEntry(ctx, obj)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeNameTree end: offset=%d ***\n", ctx.Write.Offset)

	return
}

package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

// TODO actually write something!
func writeDestinationArray(ctx *types.PDFContext, arr types.PDFArray) (written bool, err error) {

	logInfoWriter.Printf("*** writeDestinationArray: begin offset:%d ***\n", ctx.Write.Offset)

	// Relaxed: Allow nil for Page for all actions, (according to spec only for remote Goto actions nil allowed)
	//if arr[0] == nil {
	//	return false, errors.New("writeDestinationArray end: arr[0] (PageIndRef) is null")
	//}

	indRef, ok := arr[0].(types.PDFIndirectRef)
	if !ok {
		// TODO: log.Fatalln("writeDestinationArray: destination array[0] not an indirect ref.")
		logInfoWriter.Println("writeDestinationArray end: arr[0] is no indRef")
		return
	}

	pageDict, err := ctx.DereferenceDict(indRef)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page" {
		return false, errors.Errorf("writeDestinationArray: invalid destination page dict. for obj#%d", indRef.ObjectNumber)
	}

	pageObjectNumber := int(indRef.ObjectNumber)
	if !ctx.Write.HasWriteOffset(pageObjectNumber) {
		return false, errors.New("writeDestinationArray: first element must be indRef to existing pageDict")
	}

	written = true

	logInfoWriter.Printf("*** writeDestinationArray: end offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeDestinationDict(ctx *types.PDFContext, dict types.PDFDict) (written bool, err error) {

	logInfoWriter.Printf("*** writeDestinationDict: begin offset:%d ***\n", ctx.Write.Offset)

	arr, written, err := writeArrayEntry(ctx, dict, "DestinationDict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeDestinationDict: arr already written end offset:%d\n", ctx.Write.Offset)
		return
	}

	if arr == nil {
		logInfoWriter.Printf("writeDestinationDict: arr is nil end offset:%d\n", ctx.Write.Offset)
		return
	}

	written, err = writeDestinationArray(ctx, *arr)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeDestinationDict: end offset:%d ***\n", ctx.Write.Offset)

	return
}

func writeDestination(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeDestination: begin offset:%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeDestination: already written, end offset:%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeDestination: is nil, end offset:%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing.
		//ok = true

	case types.PDFStringLiteral:
		// no further processing.
		//ok = true

	case types.PDFDict:
		_, err = writeDestinationDict(ctx, obj)

	case types.PDFArray:
		_, err = writeDestinationArray(ctx, obj)

	default:
		err = errors.New("writeDestination: unsupported PDF object")

	}

	if err == nil {
		logInfoWriter.Printf("*** writeDestination: end offset:%d ***\n", ctx.Write.Offset)
	}

	return
}

func writeDestinationEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(interface{}) bool) (written bool, err error) {

	logInfoWriter.Printf("*** writeDestinationEntry begin: entry=%s offset=%d ***\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeDestinationEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeDestinationEntry end: entry %s is nil\n", entryName)
		return
	}

	err = writeDestination(ctx, obj)
	if err != nil {
		return
	}

	written = true

	logInfoWriter.Printf("*** writeDestinationEntry end: entry=%s offset=%d ***\n", entryName, ctx.Write.Offset)

	return
}

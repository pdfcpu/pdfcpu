package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeExponentialInterpolationFunctionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeExponentialInterpolationFunctionDict begin:  offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < types.V13 {
		return errors.Errorf("writeExponentialInterpolationFunctionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "Range", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "C0", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "C1", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, dict, "functionDict", "N", REQUIRED, types.V13)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeExponentialInterpolationFunctionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeStitchingFunctionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeStitchingFunctionDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < types.V13 {
		return errors.Errorf("writeStitchingFunctionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "Range", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeFunctionArrayEntry(ctx, dict, "functionDict", "Functions", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "Bounds", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, "functionDict", "Encode", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeStitchingFunctionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeSampledFunctionStreamDict(ctx *types.PDFContext, dict types.PDFStreamDict) (err error) {

	logInfoWriter.Printf("*** writeSampledFunctionStreamDict begin:  offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < types.V12 {
		return errors.Errorf("writeSampledFunctionStreamDict: unsupported in version %s.\n", ctx.VersionString())
	}

	_, _, err = writeNumberArrayEntry(ctx, dict.PDFDict, "functionDict", "Domain", REQUIRED, types.V12, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict.PDFDict, "functionDict", "Range", REQUIRED, types.V12, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerArrayEntry(ctx, dict.PDFDict, "functionDict", "Size", REQUIRED, types.V12, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, "functionDict", "BitsPerSample", REQUIRED, types.V12, validate.BitsPerSample)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, "functionDict", "Order", OPTIONAL, types.V12, func(i int) bool { return i == 1 || i == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict.PDFDict, "functionDict", "Encode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict.PDFDict, "functionDict", "Decode", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeSampledFunctionStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePostScriptCalculatorFunctionStreamDict(ctx *types.PDFContext, dict types.PDFStreamDict) (err error) {

	logInfoWriter.Printf("*** writePostScriptCalculatorFunctionStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < types.V13 {
		return errors.Errorf("writePostScriptCalculatorFunctionStreamDict: unsupported in version %s.\n", ctx.VersionString())
	}

	_, _, err = writeNumberArrayEntry(ctx, dict.PDFDict, "functionDict", "Domain", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict.PDFDict, "functionDict", "Range", REQUIRED, types.V13, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writePostScriptCalculatorFunctionStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func processFunction(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** processFunction begin: offset=%d ***\n", ctx.Write.Offset)

	// Function dict: dict or stream dict with required entry "FunctionType" (integer):
	// 0: Sampled function (stream dict)
	// 2: Exponential interpolation function (dict)
	// 3: Stitching function (dict)
	// 4: PostScript calculator function (stream dict), since V1.3

	var funcType *types.PDFInteger

	switch obj := obj.(type) {

	case types.PDFDict:

		// process function  2,3

		funcType, _, err = writeIntegerEntry(ctx, obj, "functionDict", "FunctionType", REQUIRED, types.V10, func(i int) bool { return i == 2 || i == 3 })
		if err != nil {
			return
		}

		switch *funcType {
		case 2:
			err = writeExponentialInterpolationFunctionDict(ctx, obj)
			if err != nil {
				return
			}

		case 3:
			err = writeStitchingFunctionDict(ctx, obj)
			if err != nil {
				return
			}

		}

	case types.PDFStreamDict:

		// process function  0,4

		funcType, _, err = writeIntegerEntry(ctx, obj.PDFDict, "functionDict", "FunctionType", REQUIRED, types.V10, func(i int) bool { return i == 0 || i == 4 })
		if err != nil {
			return
		}

		switch *funcType {
		case 0:
			err = writeSampledFunctionStreamDict(ctx, obj)
			if err != nil {
				return
			}

		case 4:
			err = writePostScriptCalculatorFunctionStreamDict(ctx, obj)
			if err != nil {
				return
			}

		}

	default:
		err = errors.New("processFunction: obj must be dict or stream dict")
	}

	logInfoWriter.Printf("*** processFunction end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFunction(ctx *types.PDFContext, obj interface{}) (written bool, err error) {

	logInfoWriter.Printf("*** writeFunction begin: offset=%d ***\n", ctx.Write.Offset)

	object, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeFunction: missing object")
		return
	}

	err = processFunction(ctx, object)
	if err != nil {
		return
	}

	written = true

	logInfoWriter.Printf("*** writeFunction end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFunctionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeFunctionDict begin: offset=%d ***\n", ctx.Write.Offset)

	funcArr := dict.PDFArrayEntry("Functions")
	if funcArr == nil {
		logInfoWriter.Println("writeFunctionDict: no \"Functions\" array")
		return
	}

	for _, value := range *funcArr {

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writeFunctionDict: missing indirect reference for entry \"Function\"")
		}

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("writeFunctionDict: object #%d already written.\n", objNumber)
			continue
		}

		if dict, err := ctx.DereferenceDict(indRef); err == nil && dict != nil {

			logInfoWriter.Printf("writeFunctionDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

			err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
			if err != nil {
				return err
			}

			logDebugWriter.Printf("writeFunctionDict: new offset after function dict = %d\n", ctx.Write.Offset)

			err = writeFunctionDict(ctx, *dict)
			if err != nil {
				return err
			}

		}

		if streamDict, err := ctx.DereferenceStreamDict(indRef); err == nil && streamDict != nil {

			logInfoWriter.Printf("writeFunctionDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

			err = writePDFStreamDictObject(ctx, objNumber, genNumber, *streamDict)
			if err != nil {
				return err
			}

			logDebugWriter.Printf("writeFunctionDict: new offset after function stream dict = %d\n", ctx.Write.Offset)

			err = writeFunctionStreamDict(ctx, *streamDict)
			if err != nil {
				return err
			}

		} else {
			return errors.Errorf("writeFunctionDict: object #%d neither dict nor stream dict or null", objNumber)
		}

	}

	logInfoWriter.Printf("*** writeFunctionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFunctionStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict) (err error) {

	logInfoWriter.Printf("*** writeFunctionStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	funcArr := streamDict.PDFArrayEntry("Functions")
	if funcArr == nil {
		logInfoWriter.Println("writeFunctionStreamDict: no \"Functions\" array")
		return
	}

	for _, value := range *funcArr {

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writeFunctionStreamDict: missing indirect reference for entry \"Function\"")
		}

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("writeFunctionStreamDict: object #%d already written.\n", objNumber)
			continue
		}

		if dict, err := ctx.DereferenceDict(indRef); err == nil && dict != nil {

			logInfoWriter.Printf("writeFunctionStreamDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

			err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
			if err != nil {
				return err
			}

			logDebugWriter.Printf("writeFunctionStreamDict: new offset after function dict = %d\n", ctx.Write.Offset)

			err = writeFunctionDict(ctx, *dict)
			if err != nil {
				return err
			}

		}

		if streamDict, err := ctx.DereferenceStreamDict(indRef); err == nil && streamDict != nil {

			logInfoWriter.Printf("writeFunctionStreamDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

			err = writePDFStreamDictObject(ctx, objNumber, genNumber, *streamDict)
			if err != nil {
				return err
			}

			logDebugWriter.Printf("writeFunctionStreamDict: new offset after function stream dict = %d\n", ctx.Write.Offset)

			err = writeFunctionStreamDict(ctx, *streamDict)
			if err != nil {
				return nil
			}
		}

	}

	logInfoWriter.Printf("*** writeFunctionStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

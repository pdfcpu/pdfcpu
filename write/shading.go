package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeFunctionBasedShadingDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeFunctionBasedShadingDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "functionBasedShadingDict"

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Matrix", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", REQUIRED, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeFunctionBasedShadingDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAxialShadingDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAxialShadingDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "axialShadingDict"

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Coords", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", REQUIRED, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	_, _, err = writeBooleanArrayEntry(ctx, dict, dictName, "Extend", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeAxialShadingDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeRadialShadingDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeRadialShadingDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "radialShadingDict"

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Coords", REQUIRED, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Domain", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", REQUIRED, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	_, _, err = writeBooleanArrayEntry(ctx, dict, dictName, "Extend", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeRadialShadingDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeShadingDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// Shading 1-3

	logInfoWriter.Printf("*** writeShadingDict begin: offset=%d ***\n", ctx.Write.Offset)

	shadingType, err := writeShadingDictCommonEntries(ctx, dict)
	if err != nil {
		return
	}

	switch shadingType {
	case 1:
		err = writeFunctionBasedShadingDict(ctx, dict)

	case 2:
		err = writeAxialShadingDict(ctx, dict)

	case 3:
		err = writeRadialShadingDict(ctx, dict)

	default:
		err = errors.Errorf("writeShadingDict: unexpected shadingType: %d\n", shadingType)
	}

	logInfoWriter.Printf("*** writeShadingDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeShadingDictCommonEntries(ctx *types.PDFContext, dict types.PDFDict) (i int, err error) {

	logInfoWriter.Printf("*** writeShadingDictCommonEntries begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "shadingDictCommonEntries"

	shadingType, _, err := writeIntegerEntry(ctx, dict, dictName, "ShadingType", REQUIRED, types.V10, func(i int) bool { return i >= 1 && i <= 7 })
	if err != nil {
		return
	}

	err = writeColorSpaceEntry(ctx, dict, dictName, "ColorSpace", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return
	}

	_, _, err = writeArrayEntry(ctx, dict, dictName, "Background", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeRectangleEntry(ctx, dict, dictName, "BBox", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, dict, dictName, "AntiAlias", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	i = shadingType.Value()

	logInfoWriter.Printf("*** writeShadingDictCommonEntries end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFreeFormGouroudShadedTriangleMeshesDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeFreeFormGouroudShadedTriangleMeshesDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "freeFormGouraudShadedTriangleMeshesDict"

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validate.BitsPerCoordinate)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validate.BitsPerComponent)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerFlag", REQUIRED, types.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", OPTIONAL, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeFreeFormGouroudShadedTriangleMeshesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeLatticeFormGouraudShadedTriangleMeshesDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeLatticeFormGouraudShadedTriangleMeshesDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "latticeFormGouraudShadedTriangleMeshesDict"

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validate.BitsPerCoordinate)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validate.BitsPerComponent)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "VerticesPerRow", REQUIRED, types.V10, func(i int) bool { return i >= 2 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", OPTIONAL, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeLatticeFormGouraudShadedTriangleMeshesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCoonsPatchMeshesDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeCoonsPatchMeshesDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "coonsPatchMeshesDict"

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validate.BitsPerCoordinate)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validate.BitsPerComponent)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerFlag", REQUIRED, types.V10, func(i int) bool { return i >= 0 && i <= 8 }) // TODO relaxed from 3 to 8
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", OPTIONAL, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeCoonsPatchMeshesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeTensorProductPatchMeshesDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeTensorProductPatchMeshesDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "tensorProductPatchMeshesDict"

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerCoordinate", REQUIRED, types.V10, validate.BitsPerCoordinate)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, validate.BitsPerComponent)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerFlag", REQUIRED, types.V10, func(i int) bool { return i >= 0 && i <= 8 }) // TODO relaxed from 3 to 8
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Decode", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = writeFunctionEntry(ctx, dict, dictName, "Function", OPTIONAL, types.V10) // TODO or array of functions
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeTensorProductPatchMeshesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeShadingStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict) (err error) {

	// Shading 4-7

	logInfoWriter.Printf("*** writeShadingStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict := streamDict.PDFDict

	shadingType, err := writeShadingDictCommonEntries(ctx, dict)
	if err != nil {
		return
	}

	switch shadingType {

	case 4:
		err = writeFreeFormGouroudShadedTriangleMeshesDict(ctx, dict)

	case 5:
		err = writeLatticeFormGouraudShadedTriangleMeshesDict(ctx, dict)

	case 6:
		err = writeCoonsPatchMeshesDict(ctx, dict)

	case 7:
		err = writeTensorProductPatchMeshesDict(ctx, dict)

	default:
		err = errors.Errorf("writeShadingStreamDict: unexpected shadingType: %d\n", shadingType)
	}

	logInfoWriter.Printf("*** writeShadingStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeShading(ctx *types.PDFContext, obj interface{}) (err error) {

	// see 8.7.4.3 Shading Dictionaries

	logInfoWriter.Printf("*** writeShading begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeShading end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeShading end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writeShadingDict(ctx, obj)

	case types.PDFStreamDict:
		err = writeShadingStreamDict(ctx, obj)

	default:
		err = errors.New("writeShading: corrupt obj typ, must be dict or stream dict")

	}

	logInfoWriter.Printf("*** writeShading end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeShadingResourceDict(ctx *types.PDFContext, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	// see 8.7.4.3 Shading Dictionaries

	logInfoWriter.Printf("*** writeShadingResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeShadingResourceDict: unsupported in version %s.\n", ctx.VersionString())
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("*** writeShadingResourceDict end: object already written. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("*** writeShadingResourceDict end: object is nil. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeShadingResourceDict: corrupt dict")
	}

	// Iterate over shading resource dictionary
	for key, obj := range dict.Dict {

		logInfoWriter.Printf("writeShadingResourceDict: processing entry: %s\n", key)

		// Process shading
		err = writeShading(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeShadingResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

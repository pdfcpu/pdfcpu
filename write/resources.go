package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeResourceDict(ctx *types.PDFContext, obj interface{}) (written bool, err error) {

	logInfoWriter.Printf("*** writeResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeResourceDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeResourceDict end: object  is nil.\n")
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.New("writeResourceDict: corrupt resources dict")
		return
	}

	if obj, ok := dict.Find("ExtGState"); ok {
		err = writeExtGStateResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Font"); ok {
		err = writeFontResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("XObject"); ok {
		err = writeXObjectResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Properties"); ok {
		err = writePropertiesResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("ColorSpace"); ok {
		err = writeColorSpaceResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Pattern"); ok {
		err = writePatternResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	if obj, ok := dict.Find("Shading"); ok {
		err = writeShadingResourceDict(ctx, obj, types.V13)
		if err != nil {
			return
		}
	}

	// Beginning with V1.4 this feature is considered to be obsolete.
	_, _, err = writeNameArrayEntry(ctx, dict, "resourceDict", "ProcSet", OPTIONAL, types.V10, validate.ProcedureSetName)
	if err != nil {
		return
	}

	written = true

	logInfoWriter.Printf("*** writeResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

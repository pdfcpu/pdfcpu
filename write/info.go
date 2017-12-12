package write

import (
	"fmt"
	"strings"
	"time"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/hhrutter/pdfcpu/validate"
	"github.com/pkg/errors"
)

func date() (string, error) {

	now := time.Now()
	_, tz := now.Zone()

	dateStr := fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		tz/60/60, tz/60%60)

	if !validate.Date(dateStr) {
		return "", errors.Errorf("date: invalid dateString: %s\n", dateStr)
	}

	return dateStr, nil
}

func textString(ctx *types.PDFContext, obj interface{}) (string, error) {

	var s string
	var err error

	obj, err = ctx.Dereference(obj)
	if err != nil {
		return s, err
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		s, err = types.StringLiteralToString(obj.Value())
		if err != nil {
			return s, err
		}

	case types.PDFHexLiteral:
		s, err = types.HexLiteralToString(obj.Value())
		if err != nil {
			return s, err
		}

	default:
		return s, errors.New("writeInfoObject: corrupt \"Producer\"")
	}

	// Return a csv safe string.
	return strings.Replace(s, ";", ",", -1), nil
}

// Write the document info object for this PDF file.
// Add pdfcpu as Producer with proper creation date and mod date.
func writeDocumentInfoDict(ctx *types.PDFContext) (err error) {

	// => 14.3.3 Document Information Dictionary

	// Optional:
	// Title                -
	// Author               -
	// Subject              -
	// Keywords             -
	// Creator              -
	// Producer		        modified by pdfcpu
	// CreationDate	        modified by pdfcpu
	// ModDate		        modified by pdfcpu
	// Trapped              -

	logInfoWriter.Printf("*** writeDocumentInfoDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Document info object is optional.
	if ctx.Info == nil {
		logInfoWriter.Printf("writeDocumentInfoObject end: No info object present, offset=%d\n", ctx.Write.Offset)
		// Note: We could generate an info object from scratch in this scenario.
		return
	}

	logInfoWriter.Printf("writeDocumentInfoObject: %s\n", *ctx.Info)

	obj := *ctx.Info

	dict, err := ctx.DereferenceDict(obj)
	if err != nil || dict == nil {
		return
	}

	for key, value := range dict.Dict {

		switch key {

		case "Title":
			logDebugWriter.Println("found Title")

		case "Author":
			logDebugWriter.Println("found Author")

			if indRef, ok := value.(types.PDFIndirectRef); ok {
				value, err = ctx.Dereference(indRef)
				if err != nil {
					return
				}
			}

			ctx.Author, err = textString(ctx, value)
			if err != nil {
				return
			}

		case "Subject":
			logDebugWriter.Println("found Subject")

		case "Keywords":
			logDebugWriter.Println("found Keywords")

		case "Creator":
			logDebugWriter.Println("found Creator")

			if indRef, ok := value.(types.PDFIndirectRef); ok {
				value, err = ctx.Dereference(indRef)
				if err != nil {
					return
				}
			}

			ctx.Creator, err = textString(ctx, value)
			if err != nil {
				return
			}

		case "Producer":
			logDebugWriter.Println("found Producer")

			if indRef, ok := value.(types.PDFIndirectRef); ok {
				value, err = ctx.Dereference(indRef)
				if err != nil {
					return
				}
				// Do not write indRef, will be modified by pdfcpu.
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

			ctx.Producer, err = textString(ctx, value)
			if err != nil {
				return
			}

		case "CreationDate":
			logDebugWriter.Println("found CreationDate")

			if indRef, ok := value.(types.PDFIndirectRef); ok {
				// Do not write indRef, will be modified by pdfcpu.
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

		case "ModDate":
			logDebugWriter.Println("found ModDate")

			if indRef, ok := value.(types.PDFIndirectRef); ok {
				// Do not write indRef, will be modified by pdfcpu.
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

		case "Trapped":
			logDebugWriter.Println("found Trapped")

		default:
			logInfoWriter.Printf("writeInfoObject: found out of spec entry %s %v\n", key, value)

		}
	}

	// These are the modifications for the info dict of this PDF file:

	var dateStr string

	dateStr, err = date()
	if err != nil {
		return
	}

	dict.Update("CreationDate", types.PDFStringLiteral(dateStr))
	dict.Update("ModDate", types.PDFStringLiteral(dateStr))
	dict.Update("Producer", types.PDFStringLiteral(types.PDFCPULongVersion))

	_, _, err = writeDeepObject(ctx, obj)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeDocumentInfoDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

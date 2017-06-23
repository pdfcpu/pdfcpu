package write

import (
	"fmt"
	"strings"
	"time"

	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

// The main info dict gets a modified Producer and Creation/ModDates.
func writeDocumentInfoDict(ctx *types.PDFContext, obj interface{}, main bool) (hasModDate bool, err error) {

	logInfoWriter.Printf("*** writeDocumentInfoInfoDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict, err := ctx.DereferenceDict(obj)
	if err != nil || dict == nil {
		return
	}

	var s *string

	for key, value := range dict.Dict {

		switch key {

		case "Title":
			logDebugWriter.Println("found Title")
			_, _, err = writeTextString(ctx, value, nil)
			if err != nil {
				return
			}

		case "Author":
			logDebugWriter.Println("found Author")
			s, _, err = writeTextString(ctx, value, nil)
			if err != nil {
				return
			}
			if main {
				ctx.Author = strings.Replace(*s, ";", ",", -1)
			}

		case "Subject":
			logDebugWriter.Println("found Subject")
			_, _, err = writeTextString(ctx, value, nil)
			if err != nil {
				return
			}

		case "Keywords":
			logDebugWriter.Println("found Keywords")
			_, _, err = writeString(ctx, value, nil)
			if err != nil {
				return
			}

		case "Creator":
			logDebugWriter.Println("found Creator")
			s, _, err = writeTextString(ctx, value, nil)
			if err != nil {
				return
			}
			if main {
				ctx.Creator = strings.Replace(*s, ";", ",", -1)
			}

		case "Producer":
			logDebugWriter.Println("found Producer")
			if !main {
				_, _, err = writeTextString(ctx, value, nil)
				if err != nil {
					return
				}
				continue
			}

			indRef, ok := value.(types.PDFIndirectRef)
			if ok {
				value, _ = ctx.Dereference(indRef)
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

			var s string

			switch obj := value.(type) {

			case types.PDFStringLiteral:
				s, err = types.StringLiteralToString(obj.Value())
				if err != nil {
					return
				}

			case types.PDFHexLiteral:
				s, err = types.HexLiteralToString(obj.Value())
				if err != nil {
					return
				}

			default:
				return false, errors.New("writeInfoObject: corrupt \"Producer\"")
			}

			ctx.Producer = strings.Replace(s, ";", ",", -1)

		case "CreationDate":
			logDebugWriter.Println("found CreationDate")
			if !main {
				_, _, err = writeDate(ctx, value)
				if err != nil {
					return
				}
				continue
			}

			// Do not write indRef, will be modified by pdflib.
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

		case "ModDate":
			logDebugWriter.Println("found ModDate")

			hasModDate = true

			if !main {
				_, _, err = writeDate(ctx, value)
				if err != nil {
					return
				}
				continue
			}

			// Do not write indRef, will be modified by pdflib.
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

		case "Trapped":
			logDebugWriter.Println("found Trapped")
			_, _, err = writeName(ctx, value, nil)
			if err != nil {
				return
			}

		default:
			logInfoWriter.Printf("writeInfoObject: found out of spec entry %s %v\n", key, value)
			_, _, err = writeObject(ctx, value)
			if err != nil {
				return
			}

		}
	}

	if main {

		// These are the modifications for the main document info dict of this PDF file.

		now := time.Now()
		_, tz := now.Zone()

		dateStr := fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
			now.Year(), now.Month(), now.Day(),
			now.Hour(), now.Minute(), now.Second(),
			tz/60/60, tz/60%60)

		if !validate.Date(dateStr) {
			return false, errors.Errorf("writeInfoObect: invalid date: %s\n", dateStr)
		}

		// TODO insert CreationDate, ModDate and Producer if missing.
		dict.Update("CreationDate", types.PDFStringLiteral(dateStr))
		dict.Update("ModDate", types.PDFStringLiteral(dateStr))
		dict.Update("Producer", types.PDFStringLiteral("golang pdflib"))
	}

	logInfoWriter.Printf("writeInfoObject: gets writeoffset: %d\n", ctx.Write.Offset)

	_, _, err = writeObject(ctx, obj)
	if err != nil {
		return false, err
	}

	return
}

// Write the document info object for this PDF file.
// Add pdflib as Producer with proper creation date and mod date.
func writeDocumentInfoObject(ctx *types.PDFContext) (err error) {

	// => 14.3.3 Document Information Dictionary

	// Optional:
	// Title                -
	// Author               -
	// Subject              -
	// Keywords             -
	// Creator              -
	// Producer		        modified by pdflib
	// CreationDate	        modified by pdflib
	// ModDate		        modified by pdflib
	// Trapped              -

	logInfoWriter.Printf("*** writeDocumentInfoObject begin: offset=%d ***\n", ctx.Write.Offset)

	// Document info object is optional.
	if ctx.Info == nil {
		logInfoWriter.Printf("writeDocumentInfoObject end: No info object present, offset=%d\n", ctx.Write.Offset)
		// TODO Generate info object from scratch.
		return
	}

	logInfoWriter.Printf("writeDocumentInfoObject: %s\n", *ctx.Info)

	_, err = writeDocumentInfoDict(ctx, *ctx.Info, true)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeDocumentInfoObject end: offset=%d ***\n", ctx.Write.Offset)

	return
}

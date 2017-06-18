package write

import (
	"fmt"
	"strings"
	"time"

	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeInfoObject(ctx *types.PDFContext) (err error) {

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

	logInfoWriter.Printf("*** writeInfoObject begin: offset=%d ***\n", ctx.Write.Offset)

	xRefTable := ctx.XRefTable

	// Document info object is optional.
	if xRefTable.Info == nil {
		logInfoWriter.Printf("writeInfoObject end: No info object present, offset=%d\n", ctx.Write.Offset)
		// TODO Generate info object from scratch.
		return
	}

	logInfoWriter.Printf("writeInfoObject: %s\n", *xRefTable.Info)
	info := *xRefTable.Info
	objNumber := int(info.ObjectNumber)
	genNumber := int(info.GenerationNumber)

	infoDict, err := xRefTable.DereferenceDict(info)
	if err != nil {
		return errors.New("writeInfoObject: corrupt info dict")
	}

	// Document info object is optional.
	if infoDict == nil {
		// TODO Generate pdflib info object.
		return
	}

	var s *string

	for key, value := range infoDict.Dict {

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
			xRefTable.Author = strings.Replace(*s, ";", ",", -1)

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
			xRefTable.Creator = strings.Replace(*s, ";", ",", -1)

		case "Producer":
			// Do not write indRef, will be modified by pdflib.
			logDebugWriter.Println("found Producer")

			indRef, ok := value.(types.PDFIndirectRef)
			if ok {
				value, _ = xRefTable.Dereference(indRef)
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
				return errors.New("writeInfoObject: corrupt \"Producer\"")
			}

			xRefTable.Producer = strings.Replace(s, ";", ",", -1)

		case "CreationDate":
			// Do not write indRef, will be modified by pdflib.
			logDebugWriter.Println("found CreationDate")
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

		case "ModDate":
			// Do not write indRef, will be modified by pdflib.
			logDebugWriter.Println("found ModDate")
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
			// Relaxed allow any object, should be text string.
			//_, _, err = writeTextString(source, dest, value, nil)
			_, _, err = writeObject(ctx, value)
			if err != nil {
				return
			}

		}
	}

	now := time.Now()
	_, tz := now.Zone()

	dateStr := fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(),
		tz/60/60, tz/60%60)

	if !validate.Date(dateStr) {
		return errors.Errorf("writeInfoObect: invalid date: %s\n", dateStr)
	}

	infoDict.Update("CreationDate", types.PDFStringLiteral(dateStr))
	infoDict.Update("ModDate", types.PDFStringLiteral(dateStr))
	infoDict.Update("Producer", types.PDFStringLiteral("golang pdflib"))

	logInfoWriter.Printf("writeInfoObject: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	err = writePDFDictObject(ctx, objNumber, genNumber, *infoDict)
	if err != nil {
		return err
	}

	logDebugWriter.Printf("writeInfoObject: new offset after infoDict = %d\n", ctx.Write.Offset)

	logInfoWriter.Printf("*** writeInfoObject end: offset=%d ***\n", ctx.Write.Offset)

	return
}

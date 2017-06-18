package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeGoToActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.2

	logInfoWriter.Printf("*** writeGoToActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	_, err = writeDestinationEntry(ctx, dict, "go-to action dict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeGoToActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeGoToRActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// Remote go-to action.
	// see 12.6.4.3

	logInfoWriter.Printf("*** writeGoToRActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "remote go-to action dict"

	_, err = writeFileSpecEntry(ctx, dict, dictName, "F", REQUIRED, types.V11)
	if err != nil {
		return
	}

	_, err = writeDestinationEntry(ctx, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, dict, dictName, "NewWindow", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeGoToRActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeGoToEActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// Embedded go-to action.
	// see 12.6.4.4

	logInfoWriter.Printf("*** writeGoToEActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeGoToEActionDict: unsupported action type")

	if ctx.Version() < types.V16 {
		return errors.Errorf("writeGoToEActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeGoToEActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeLaunchActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.5

	logInfoWriter.Printf("*** writeLaunchActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeLaunchActionDict: unsupported action type ***")

	logInfoWriter.Printf("*** writeLaunchActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeThreadActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	//see 12.6.4.6

	logInfoWriter.Printf("*** writeThreadActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeThreadActionDict: unsupported action type ***")

	logInfoWriter.Printf("*** writeThreadActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeURIActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.7

	logInfoWriter.Printf("*** writeURIActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "uriActionDict"

	_, _, err = writeStringEntry(ctx, dict, dictName, "URI", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, dict, dictName, "IsMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeURIActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeSoundActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.8

	logInfoWriter.Printf("*** writeSoundActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeSoundActionDict: unsupported action type")

	if ctx.Version() < types.V12 {
		return errors.Errorf("writeSoundActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeSoundActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeMovieActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.9

	logInfoWriter.Printf("*** writeMovieActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeMovieActionDict: unsupported action type")

	if ctx.Version() < types.V12 {
		return errors.Errorf("writeMovieActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeMovieActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeHideActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.10

	logInfoWriter.Printf("*** writeHideActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeHideActionDict: unsupported action type")

	if ctx.Version() < types.V12 {
		err = errors.Errorf("writeHideActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeHideActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeNamedActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.11

	logInfoWriter.Printf("*** writeNamedActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	if ctx.Version() < types.V12 {
		return errors.Errorf("writeNamedActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	_, _, err = writeNameEntry(ctx, dict, "namedActionDict", "N", REQUIRED, types.V10, validate.NamedAction)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeNamedActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeSubmitFormActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.7.5.2

	logInfoWriter.Printf("*** writeSubmitFormActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeSubmitFormActionDict: unsupported action type")

	if ctx.Version() < types.V12 {
		return errors.Errorf("writeSubmitFormActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeSubmitFormActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeResetFormActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.7.5.3

	logInfoWriter.Printf("*** writeResetFormActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeResetFormActionDict: unsupported action type")

	if ctx.Version() < types.V12 {
		return errors.Errorf("writeResetFormActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeResetFormActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeImportDataActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.7.5.4

	logInfoWriter.Printf("*** writeImportDataActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeImportDataActionDict: unsupported action type")

	if ctx.Version() < types.V12 {
		return errors.Errorf("writeImportDataActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeImportDataActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
// TODO JS = text string
func writeJavaScriptActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {
	// see 12.6.4.16

	logInfoWriter.Printf("*** writeJavaScriptActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeJavaScriptActionDict: unsupported action type")

	if ctx.Version() < types.V13 {
		return errors.Errorf("writeJavaScriptActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeJavaScriptActionDict end: offset=%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeSetOCGStateActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.12

	logInfoWriter.Printf("*** writeSetOCGStateActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeSetOCGStateActionDict: unsupported action type")

	if ctx.Version() < types.V15 {
		return errors.Errorf("writeSetOCGStateActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeSetOCGStateActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeRenditionActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.13

	logInfoWriter.Printf("*** writeRenditionActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeRenditionActionDict: unsupported action type")

	if ctx.Version() < types.V15 {
		return errors.Errorf("writeRenditionActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeRenditionActionDict end: offset=%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeTransActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.14

	logInfoWriter.Printf("*** writeTransActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeTransActionDict: unsupported action type")

	if ctx.Version() < types.V15 {
		return errors.Errorf("writeTransActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeTransActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeGoTo3DViewActionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 12.6.4.15

	logInfoWriter.Printf("*** writeGoTo3DViewActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("writeGoTo3DViewActionDict: unsupported action type")

	if ctx.Version() < types.V16 {
		return errors.Errorf("writeGoTo3DViewActionDict: unsupported in version %s.\n", ctx.VersionString())
	}

	logInfoWriter.Printf("*** writeGoTo3DViewActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeActionDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeActionDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeActionDict end: object is nil.\n")
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeActionDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeActionDict: not a PDFDict")
	}

	// Process action dict.

	if dict.Type() != nil && *dict.Type() != "Action" {
		return errors.New("writeActionDict: corrupt entry \"Type\"")
	}

	// S: name, required, action Type
	s, written, err := writeNameEntry(ctx, dict, "actionDict", "S", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if written || s == nil {
		logInfoWriter.Printf("writeActionDict end: \"S\" is nil or already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	switch *s {

	case "GoTo":
		err = writeGoToActionDict(ctx, dict)

	case "GoToR":
		err = writeGoToRActionDict(ctx, dict)

	case "GoToE":
		err = writeGoToEActionDict(ctx, dict)

	case "Launch":
		err = writeLaunchActionDict(ctx, dict)

	case "Thread":
		err = writeThreadActionDict(ctx, dict)

	case "URI":
		err = writeURIActionDict(ctx, dict)

	case "Sound":
		err = writeSoundActionDict(ctx, dict)

	case "Movie":
		err = writeMovieActionDict(ctx, dict)

	case "Hide":
		err = writeHideActionDict(ctx, dict)

	case "Named":
		err = writeNamedActionDict(ctx, dict)

	case "SubmitForm":
		err = writeSubmitFormActionDict(ctx, dict)

	case "ResetForm":
		err = writeResetFormActionDict(ctx, dict)

	case "ImportData":
		err = writeImportDataActionDict(ctx, dict)

	case "JavaScript":
		err = writeJavaScriptActionDict(ctx, dict)

	case "SetOCGState":
		err = writeSetOCGStateActionDict(ctx, dict)

	case "Rendition":
		err = writeRenditionActionDict(ctx, dict)

	case "Trans":
		err = writeTransActionDict(ctx, dict)

	case "GoTo3DView":
		err = writeGoTo3DViewActionDict(ctx, dict)

	default:
		err = errors.Errorf("writeActionDict: unsupported action type: %s\n", *s)

	}

	if err != nil {
		return
	}

	if _, ok := dict.Find("Next"); ok {
		return errors.New("writeActionDict: unsupported entry \"Next\"")
	}

	logInfoWriter.Printf("*** writeActionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOpenAction(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.2 Destinations, 12.6 Actions

	// A value specifying a destination that shall be displayed
	// or an action that shall be performed when the document is opened.
	// The value shall be either an array defining a destination (see 12.3.2, "Destinations")
	// or an action dictionary representing an action (12.6, "Actions").
	//
	// If this entry is absent, the document shall be opened
	// to the top of the first page at the default magnification factor.

	logInfoWriter.Printf("*** writeOpenAction begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := rootDict.Find("OpenAction")
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeOpenAction: required entry \"OpenAction\" missing")
			return
		}
		logInfoWriter.Printf("writeOpenAction end: optional entry \"OpenAction\" not found or nil\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeOpenAction: unsupported in version %s.\n", ctx.VersionString())
	}

	if _, err = ctx.DereferenceDict(obj); err == nil {
		err = writeActionDict(ctx, obj)
		if err != nil {
			return
		}
		logInfoWriter.Printf("*** writeOpenAction end: offset=%d ***\n", ctx.Write.Offset)
		ctx.Stats.AddRootAttr(types.RootOpenAction)
		return
	}

	if _, err = ctx.DereferenceArray(obj); err == nil {
		err = writeDestination(ctx, obj)
		if err != nil {
			return
		}
		logInfoWriter.Printf("*** writeOpenAction end: offset=%d ***\n", ctx.Write.Offset)
		ctx.Stats.AddRootAttr(types.RootOpenAction)
		return
	}

	return errors.New("writeOpenAction: must be dict or array")
}

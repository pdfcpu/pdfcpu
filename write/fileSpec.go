package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func processEmbeddedFileStreamMacParameterDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** processEmbeddedFileStreamMacParameterDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "embeddedFileStreamMacParameterDict"

	// Subtype, optional integer
	// The embedded file's file type integer encoded according to Mac OS conventions.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "Subtype", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Creator, optional integer
	// The embedded file's creator signature integer encoded according to Mac OS conventions.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "Creator", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// ResFork, optional stream dict
	// The binary contents of the embedded file's resource fork.
	_, _, err = writeStreamDictEntry(ctx, dict, dictName, "ResFork", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** processEmbeddedFileStreamMacParameterDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeEmbeddedFileStreamParameterDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeEmbeddedFileStreamParameterDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeEmbeddedFileStreamParameterDict end: offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.New("writeEmbeddedFileStreamParameterDict: corrupt dict")
		return
	}

	dictName := "embeddedFileStreamParameterDict"

	// Size, optional integer
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "Size", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// CreationDate, optional date
	_, _, err = writeDateEntry(ctx, dict, dictName, "CreationDate", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// ModDate, optional date
	_, _, err = writeDateEntry(ctx, dict, dictName, "ModDate", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Mac, optional dict
	macDict, written, err := writeDictEntry(ctx, dict, dictName, "Mac", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeEmbeddedFileStreamParameterDict end: offset=%d\n", ctx.Write.Offset)
		return
	}

	err = processEmbeddedFileStreamMacParameterDict(ctx, *macDict)
	if err != nil {
		return
	}

	// CheckSum, optional string
	_, _, err = writeStringEntry(ctx, dict, dictName, "CheckSum", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeEmbeddedFileStreamParameterDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func processEmbeddedFileStreamDict(ctx *types.PDFContext, sd types.PDFStreamDict) (err error) {

	logInfoWriter.Printf("*** processEmbeddedFileStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	if sd.Type() != nil && *sd.Type() != "EmbeddedFile" {
		return errors.Errorf("processEmbeddedFileStreamDict: invalid type: %s\n", *sd.Type())
	}

	// TODO Subtype, optional name, see Annex E, RFC 2046 (MIME)

	// Params, optional dict
	// parameter dict containing additional file-specific information.
	if obj, found := sd.PDFDict.Find("Params"); found && obj != nil {
		err = writeEmbeddedFileStreamParameterDict(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** processEmbeddedFileStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func processFileSpecDictEntriesEFAndRF(ctx *types.PDFContext, efDict, rfDict *types.PDFDict) (err error) {

	// EF only or EF and RF

	logInfoWriter.Printf("*** processFileSpecDictEntriesEFAndRF begin: offset=%d ***\n", ctx.Write.Offset)

	if efDict == nil {
		return errors.Errorf("processFileSpecEntriesEFAndRF: missing required efDict.")
	}

	var written bool
	var obj interface{}

	for k, obj := range (*efDict).Dict {

		if !(k == "F" || k == "UF" || k == "DOS" || k == "Mac" || k == "Unix") {
			return errors.Errorf("processFileSpecEntriesEFAndRF: invalid key: %s", k)
		}

		// value must be embedded file stream dict
		// see 7.11.4
		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written {
			continue
		}

		if obj == nil {
			return errors.Errorf("processFileSpecEntriesEFAndRF: efDict entry=%s missing embedded file stream dict", k)
		}

		sd, ok := obj.(types.PDFStreamDict)
		if !ok {
			return errors.Errorf("processFileSpecEntriesEFAndRF: efDict entry=%s not an embedded file stream dict", k)
		}

		err = processEmbeddedFileStreamDict(ctx, sd)
		if err != nil {
			return
		}

		continue

	}

	if rfDict != nil {

		for k, val := range (*rfDict).Dict {

			if _, ok := efDict.Find(k); !ok {
				return errors.Errorf("processFileSpecEntriesEFAndRF: rfDict entry=%s missing corresponding efDict entry\n", k)
			}

			// value must be related files array.
			// see 7.11.4.2
			obj, written, err = writeObject(ctx, val)
			if err != nil {
				return
			}

			if written || obj == nil {
				continue
			}

			// TODO:
			// array length must be even
			// odd entries must be strings
			// even entries must be ind ref of embedded file stream

		}

	}

	logInfoWriter.Printf("*** processFileSpecDictEntriesEFAndRF end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func processFileSpecDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** processFileSpecDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "fileSpecDict"

	// TODO validation, see 7.11.3
	_, _, err = writeNameEntry(ctx, dict, dictName, "FS", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// DOS, byte string, optional, obsolescent.
	_, dosFound := dict.Find("DOS")

	// Mac, byte string, optional, obsolescent.
	_, macFound := dict.Find("Mac")

	// Unix, byte string, optional, obsolescent.
	_, unixFound := dict.Find("Unix")

	// TODO if $FS=URL, process text string for F
	_, _, err = writeStringEntry(ctx, dict, dictName, "F", !dosFound && !macFound && !unixFound, types.V10, validate.FileSpecStringOrURLString)
	if err != nil {
		return
	}

	// TODO process text string
	_, _, err = writeStringEntry(ctx, dict, dictName, "UF", OPTIONAL, types.V17, validate.FileSpecString)
	if err != nil {
		return
	}

	_, _, err = writeStringArrayEntry(ctx, dict, dictName, "ID", OPTIONAL, types.V11, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, dict, dictName, "V", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	// RF, dict of related files arrays, optional, since 1.3
	rfDict, _, err := writeDictEntry(ctx, dict, dictName, "RF", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// EF, dict of embedded file streams, required if RF is present, since 1.3
	efDict, _, err := writeDictEntry(ctx, dict, dictName, "EF", rfDict != nil, types.V13, nil)
	if err != nil {
		return
	}

	// if EF present, Type "FileSpec" is required
	if efDict != nil {

		if dict.Type() == nil || *dict.Type() != "Filespec" {
			return errors.New("processFileSpecDict: missing type: FileSpec")
		}

		err = processFileSpecDictEntriesEFAndRF(ctx, efDict, rfDict)
		if err != nil {
			return
		}

	}

	_, _, err = writeStringEntry(ctx, dict, dictName, "Desc", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	// TODO shall be indirect ref, collection item dict.
	// see 7.11.6
	d, _, err := writeDictEntry(ctx, dict, dictName, "CI", OPTIONAL, types.V17, nil)
	if err != nil {
		return
	}

	if d != nil {
		// TODO
		return errors.New("processFileSpecDict: unsupported entry CI")
	}

	logInfoWriter.Printf("*** processFileSpecDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeFileSpecEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (written bool, err error) {

	logInfoWriter.Printf("*** writeFileSpecEntry begin: entry=%s offset=%d ***\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeFileSpecEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFileSpecEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeFileSpecEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeFileSpecEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFileSpecEntry end: entry %s is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		s := obj.Value()
		if !validate.FileSpecString(s) {
			err = errors.Errorf("writeFileSpecEntry: dict=%s entry=%s invalid file spec string: %s", dictName, entryName, s)
			return
		}

	case types.PDFHexLiteral:
		s := obj.Value()
		if !validate.FileSpecString(s) {
			err = errors.Errorf("writeFileSpecEntry: dict=%s entry=%s invalid file spec string: %s", dictName, entryName, s)
			return
		}

	case types.PDFDict:
		err = processFileSpecDict(ctx, obj)
		if err != nil {
			return
		}

	default:
		err = errors.Errorf("writeFileSpecEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	logInfoWriter.Printf("*** writeFileSpecEntry end: entry=%s offset=%d ***\n", entryName, ctx.Write.Offset)

	return
}

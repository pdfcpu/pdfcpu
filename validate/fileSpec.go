package validate

import (
	"github.com/hhrutter/pdflib/types"

	"github.com/pkg/errors"
)

func processEmbeddedFileStreamMacParameterDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** processEmbeddedFileStreamMacParameterDict begin ***")

	dictName := "embeddedFileStreamMacParameterDict"

	// Subtype, optional integer
	// The embedded file's file type integer encoded according to Mac OS conventions.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Subtype", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Creator, optional integer
	// The embedded file's creator signature integer encoded according to Mac OS conventions.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Creator", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// ResFork, optional stream dict
	// The binary contents of the embedded file's resource fork.
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ResFork", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** processEmbeddedFileStreamMacParameterDict end ***")

	return
}

func validateEmbeddedFileStreamParameterDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateEmbeddedFileStreamParameterDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateEmbeddedFileStreamParameterDict end")
		return
	}

	dictName := "embeddedFileStreamParameterDict"

	// Size, optional integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Size", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// CreationDate, optional date
	_, err = validateDateEntry(xRefTable, dict, dictName, "CreationDate", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// ModDate, optional date
	_, err = validateDateEntry(xRefTable, dict, dictName, "ModDate", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Mac, optional dict
	macDict, err := validateDictEntry(xRefTable, dict, dictName, "Mac", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateEmbeddedFileStreamParameterDict end")
		return
	}

	err = processEmbeddedFileStreamMacParameterDict(xRefTable, macDict)
	if err != nil {
		return
	}

	// CheckSum, optional string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CheckSum", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateEmbeddedFileStreamParameterDict end ***")

	return
}

// TODO implement
func processEmbeddedFileStreamDict(xRefTable *types.XRefTable, sd *types.PDFStreamDict) (err error) {

	logInfoValidate.Println("*** processEmbeddedFileStreamDict begin ***")

	if sd.Type() != nil && *sd.Type() != "EmbeddedFile" {
		return errors.Errorf("processEmbeddedFileStreamDict: invalid type: %s\n", *sd.Type())
	}

	// TODO Subtype, optional name, see Annex E, RFC 2046 (MIME)

	// Params, optional dict
	// parameter dict containing additional file-specific information.
	if obj, found := sd.PDFDict.Find("Params"); found && obj != nil {
		err = validateEmbeddedFileStreamParameterDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** processEmbeddedFileStreamDict end ***")

	return
}

// TODO implement
func processFileSpecDictEntriesEFAndRF(xRefTable *types.XRefTable, efDict, rfDict *types.PDFDict) (err error) {

	// EF only or EF and RF

	logInfoValidate.Println("*** processFileSpecDictEntriesEFAndRF begin ***")

	if efDict == nil {
		return errors.Errorf("processFileSpecEntriesEFAndRF: missing required efDict.")
	}

	var obj interface{}

	for k, obj := range (*efDict).Dict {

		if !(k == "F" || k == "UF" || k == "DOS" || k == "Mac" || k == "Unix") {
			return errors.Errorf("processFileSpecEntriesEFAndRF: invalid key: %s", k)
		}

		// value must be embedded file stream dict
		// see 7.11.4
		sd, err := validateStreamDict(xRefTable, obj)
		if err != nil {
			return err
		}

		err = processEmbeddedFileStreamDict(xRefTable, sd)
		if err != nil {
			return err
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
			obj, err = xRefTable.Dereference(val)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			// TODO:
			// array length must be even
			// odd entries must be strings
			// even entries must be ind ref of embedded file stream

		}

	}

	logInfoValidate.Println("*** processFileSpecDictEntriesEFAndRF end ***")

	return
}

// TODO implement
func processFileSpecDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** processFileSpecDict begin ***")

	dictName := "fileSpecDict"

	// TODO validation, see 7.11.3
	_, err = validateNameEntry(xRefTable, dict, dictName, "FS", OPTIONAL, types.V10, nil)
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
	_, err = validateStringEntry(xRefTable, dict, dictName, "F", !dosFound && !macFound && !unixFound, types.V10, validateFileSpecStringOrURLString)
	if err != nil {
		return
	}

	// TODO process text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "UF", OPTIONAL, types.V17, validateFileSpecString)
	if err != nil {
		return
	}

	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "ID", OPTIONAL, types.V11, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "V", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	// RF, dict of related files arrays, optional, since 1.3
	rfDict, err := validateDictEntry(xRefTable, dict, dictName, "RF", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// EF, dict of embedded file streams, required if RF is present, since 1.3
	efDict, err := validateDictEntry(xRefTable, dict, dictName, "EF", rfDict != nil, types.V13, nil)
	if err != nil {
		return
	}

	// if EF present, Type "FileSpec" is required
	if efDict != nil {

		if dict.Type() == nil || *dict.Type() != "Filespec" {
			return errors.New("processFileSpecDict: missing type: FileSpec")
		}

		err = processFileSpecDictEntriesEFAndRF(xRefTable, efDict, rfDict)
		if err != nil {
			return
		}

	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "Desc", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	// TODO shall be indirect ref, collection item dict.
	// see 7.11.6
	d, err := validateDictEntry(xRefTable, dict, dictName, "CI", OPTIONAL, types.V17, nil)
	if err != nil {
		return
	}

	if d != nil {
		// TODO
		return errors.New("processFileSpecDict: unsupported entry CI")
	}

	logInfoValidate.Println("*** processFileSpecDict end ***")

	return
}

func validateFileSpecEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("*** validateFileSpecEntry begin: entry=%s ***\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateFileSpecEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFileSpecEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateFileSpecEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFileSpecEntry end: entry %s is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		s := obj.Value()
		if !validateFileSpecString(s) {
			err = errors.Errorf("validateFileSpecEntry: dict=%s entry=%s invalid file spec string: %s", dictName, entryName, s)
			return
		}

	case types.PDFHexLiteral:
		s := obj.Value()
		if !validateFileSpecString(s) {
			err = errors.Errorf("validateFileSpecEntry: dict=%s entry=%s invalid file spec string: %s", dictName, entryName, s)
			return
		}

	case types.PDFDict:
		err = processFileSpecDict(xRefTable, &obj)
		if err != nil {
			return
		}

	default:
		err = errors.Errorf("validateFileSpecEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	logInfoValidate.Printf("*** validateFileSpecEntry end: entry=%s ***\n", entryName)

	return
}

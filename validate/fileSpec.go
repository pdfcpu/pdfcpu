package validate

import (
	"net/url"

	"github.com/hhrutter/pdfcpu/types"

	"github.com/pkg/errors"
)

// See 7.11.4

func validateFileSpecString(s string) bool {

	// see 7.11.2
	// The standard format for representing a simple file specification in string form divides the string into component substrings
	// separated by the SOLIDUS character (2Fh) (/). The SOLIDUS is a generic component separator that shall be mapped to the appropriate
	// platform-specific separator when generating a platform-dependent file name. Any of the components may be empty.
	// If a component contains one or more literal SOLIDI, each shall be preceded by a REVERSE SOLIDUS (5Ch) (\), which in turn shall be
	// preceded by another REVERSE SOLIDUS to indicate that it is part of the string and not an escape character.
	//
	// EXAMPLE ( in\\/out )
	// represents the file name in/out

	// I have not seen an instance of a single file spec string that actually complies with this definition and uses
	// the double reverse solidi in front of the solidus, because of that we simply
	return true
}

func validateURLString(s string) bool {

	// RFC1738 compliant URL, see 7.11.5

	_, err := url.ParseRequestURI(s)

	return err == nil
}

func validateEmbeddedFileStreamMacParameterDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "embeddedFileStreamMacParameterDict"

	// Subtype, optional integer
	// The embedded file's file type integer encoded according to Mac OS conventions.
	_, err := validateIntegerEntry(xRefTable, dict, dictName, "Subtype", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Creator, optional integer
	// The embedded file's creator signature integer encoded according to Mac OS conventions.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Creator", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// ResFork, optional stream dict
	// The binary contents of the embedded file's resource fork.
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ResFork", OPTIONAL, types.V10, nil)

	return err
}

func validateEmbeddedFileStreamParameterDict(xRefTable *types.XRefTable, obj types.PDFObject) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || obj == nil {
		return err
	}

	dictName := "embeddedFileStreamParmDict"

	// Size, optional integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Size", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// CreationDate, optional date
	_, err = validateDateEntry(xRefTable, dict, dictName, "CreationDate", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// ModDate, optional date
	_, err = validateDateEntry(xRefTable, dict, dictName, "ModDate", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	// Mac, optional dict
	macDict, err := validateDictEntry(xRefTable, dict, dictName, "Mac", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}
	if macDict != nil {
		err = validateEmbeddedFileStreamMacParameterDict(xRefTable, macDict)
		if err != nil {
			return err
		}
	}

	// CheckSum, optional string
	_, err = validateStringEntry(xRefTable, dict, dictName, "CheckSum", OPTIONAL, types.V10, nil)

	return err
}

func validateEmbeddedFileStreamDict(xRefTable *types.XRefTable, sd *types.PDFStreamDict) error {

	dictName := "embeddedFileStreamDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, &sd.PDFDict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "EmbeddedFile" })
	if err != nil {
		return err
	}

	// Subtype, optional, name
	_, err = validateNameEntry(xRefTable, &sd.PDFDict, dictName, "Subtype", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Params, optional, dict
	// parameter dict containing additional file-specific information.
	if obj, found := sd.PDFDict.Find("Params"); found && obj != nil {
		err = validateEmbeddedFileStreamParameterDict(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateFileSpecDictEntriesEFAndRFKeys(k string) bool {
	return k == "F" || k == "UF" || k == "DOS" || k == "Mac" || k == "Unix"
}

func validateFileSpecDictEntryEFDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	for k, obj := range (*dict).Dict {

		if !validateFileSpecDictEntriesEFAndRFKeys(k) {
			return errors.Errorf("validateFileSpecEntriesEFAndRF: invalid key: %s", k)
		}

		// value must be embedded file stream dict
		// see 7.11.4
		sd, err := validateStreamDict(xRefTable, obj)
		if err != nil {
			return err
		}
		if sd == nil {
			continue
		}

		err = validateEmbeddedFileStreamDict(xRefTable, sd)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRFDictFilesArray(xRefTable *types.XRefTable, arr *types.PDFArray) error {

	if len(*arr)%2 > 0 {
		return errors.New("validateRFDictFilesArray: rfDict array corrupt")
	}

	for k, v := range *arr {

		if v == nil {
			return errors.New("validateRFDictFilesArray: rfDict, array entry nil")
		}

		obj, err := xRefTable.Dereference(v)
		if err != nil {
			return err
		}

		if obj == nil {
			return errors.New("validateRFDictFilesArray: rfDict, array entry nil")
		}

		if k%2 > 0 {

			_, ok := obj.(types.PDFStringLiteral)
			if !ok {
				return errors.New("validateRFDictFilesArray: rfDict, array entry corrupt")
			}

		} else {

			// value must be embedded file stream dict
			// see 7.11.4
			sd, err := validateStreamDict(xRefTable, obj)
			if err != nil {
				return err
			}

			err = validateEmbeddedFileStreamDict(xRefTable, sd)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func validateFileSpecDictEntriesEFAndRF(xRefTable *types.XRefTable, efDict, rfDict *types.PDFDict) error {

	// EF only or EF and RF

	if efDict == nil {
		return errors.Errorf("validateFileSpecEntriesEFAndRF: missing required efDict.")
	}

	err := validateFileSpecDictEntryEFDict(xRefTable, efDict)
	if err != nil {
		return err
	}

	if rfDict != nil {

		for k, val := range (*rfDict).Dict {

			if _, ok := efDict.Find(k); !ok {
				return errors.Errorf("validateFileSpecEntriesEFAndRF: rfDict entry=%s missing corresponding efDict entry\n", k)
			}

			// value must be related files array.
			// see 7.11.4.2
			arr, err := xRefTable.DereferenceArray(val)
			if err != nil {
				return err
			}

			if arr == nil {
				continue
			}

			err = validateRFDictFilesArray(xRefTable, arr)
			if err != nil {
				return err
			}

		}

	}

	return nil
}

func validateFileSpecDictType(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	if dict.Type() == nil || (*dict.Type() != "Filespec" && (xRefTable.ValidationMode == types.ValidationRelaxed && *dict.Type() != "F")) {
		return errors.New("validateFileSpecDictType: missing type: FileSpec")
	}

	return nil
}

func requiredF(dosFound, macFound, unixFound bool) bool {
	return !dosFound && !macFound && !unixFound
}

func validateFileSpecDictEFAndRF(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// RF, optional, dict of related files arrays, since V1.3
	rfDict, err := validateDictEntry(xRefTable, dict, dictName, "RF", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// EF, required if RF present, dict of embedded file streams, since 1.3
	efDict, err := validateDictEntry(xRefTable, dict, dictName, "EF", rfDict != nil, types.V13, nil)
	if err != nil {
		return err
	}

	// Type, required if EF present, name
	validate := func(s string) bool {
		return s == "Filespec" || (xRefTable.ValidationMode == types.ValidationRelaxed && s == "F")
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", efDict != nil, types.V10, validate)
	if err != nil {
		return err
	}

	// if EF present, Type "FileSpec" is required
	if efDict != nil {

		err = validateFileSpecDictType(xRefTable, dict)
		if err != nil {
			return err
		}

		err = validateFileSpecDictEntriesEFAndRF(xRefTable, efDict, rfDict)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateFileSpecDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	dictName := "fileSpecDict"

	// FS, optional, name
	fsName, err := validateNameEntry(xRefTable, dict, dictName, "FS", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// DOS, byte string, optional, obsolescent.
	_, dosFound := dict.Find("DOS")

	// Mac, byte string, optional, obsolescent.
	_, macFound := dict.Find("Mac")

	// Unix, byte string, optional, obsolescent.
	_, unixFound := dict.Find("Unix")

	// F, file spec string
	validate := validateFileSpecString
	if fsName != nil && fsName.Value() == "URL" {
		validate = validateURLString
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "F", requiredF(dosFound, macFound, unixFound), types.V10, validate)
	if err != nil {
		return err
	}

	// UF, optional, text string
	sinceVersion := types.V17
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "UF", OPTIONAL, sinceVersion, validateFileSpecString)
	if err != nil {
		return err
	}

	// ID, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "ID", OPTIONAL, types.V11, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	// V, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "V", OPTIONAL, types.V12, nil)
	if err != nil {
		return err
	}

	err = validateFileSpecDictEFAndRF(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	// Desc, optional, text string, since V1.6
	sinceVersion = types.V16
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V10
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "Desc", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// CI, optional, collection item dict, since V1.7
	_, err = validateDictEntry(xRefTable, dict, dictName, "CI", OPTIONAL, types.V17, nil)

	return err
}

func validateFileSpecification(xRefTable *types.XRefTable, obj types.PDFObject) (types.PDFObject, error) {

	// See 7.11.4

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		s := obj.Value()
		if !validateFileSpecString(s) {
			return nil, errors.Errorf("validateFileSpecification: invalid file spec string: %s", s)
		}

	case types.PDFHexLiteral:
		s := obj.Value()
		if !validateFileSpecString(s) {
			return nil, errors.Errorf("validateFileSpecification: invalid file spec string: %s", s)
		}

	case types.PDFDict:
		err = validateFileSpecDict(xRefTable, &obj)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.Errorf("validateFileSpecification: invalid type")

	}

	return obj, nil
}

func validateURLSpecification(xRefTable *types.XRefTable, obj types.PDFObject) (types.PDFObject, error) {

	// See 7.11.4

	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return nil, err
	}

	if d == nil {
		return nil, errors.New("validateURLSpecification: missing dict")
	}

	dictName := "urlSpec"

	// FS, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "FS", REQUIRED, types.V10, func(s string) bool { return s == "URL" })
	if err != nil {
		return nil, err
	}

	// F, required, string, URL (Internet RFC 1738)
	_, err = validateStringEntry(xRefTable, d, dictName, "F", REQUIRED, types.V10, validateURLString)

	return obj, err
}

func validateFileSpecEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (types.PDFObject, error) {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return nil, err
	}

	err = xRefTable.ValidateVersion("fileSpec", sinceVersion)
	if err != nil {
		return nil, err
	}

	return validateFileSpecification(xRefTable, obj)
}

func validateURLSpecEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (types.PDFObject, error) {

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return nil, err
	}

	err = xRefTable.ValidateVersion("URLSpec", sinceVersion)
	if err != nil {
		return nil, err
	}

	return validateURLSpecification(xRefTable, obj)
}

func validateFileSpecificationOrFormObject(xRefTable *types.XRefTable, obj types.PDFObject) error {

	sd, ok := obj.(types.PDFStreamDict)
	if ok {
		return validateFormStreamDict(xRefTable, &sd)
	}

	_, err := validateFileSpecification(xRefTable, obj)

	return err
}

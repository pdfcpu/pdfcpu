/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	"net/url"

	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
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

func validateEmbeddedFileStreamMacParameterDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "embeddedFileStreamMacParameterDict"

	// Subtype, optional integer
	// The embedded file's file type integer encoded according to Mac OS conventions.
	_, err := validateIntegerEntry(xRefTable, dict, dictName, "Subtype", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Creator, optional integer
	// The embedded file's creator signature integer encoded according to Mac OS conventions.
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Creator", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// ResFork, optional stream dict
	// The binary contents of the embedded file's resource fork.
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "ResFork", OPTIONAL, pdf.V10, nil)

	return err
}

func validateEmbeddedFileStreamParameterDict(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || obj == nil {
		return err
	}

	dictName := "embeddedFileStreamParmDict"

	// Size, optional integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "Size", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// CreationDate, optional date
	_, err = validateDateEntry(xRefTable, dict, dictName, "CreationDate", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// ModDate, optional date
	_, err = validateDateEntry(xRefTable, dict, dictName, "ModDate", OPTIONAL, pdf.V10)
	if err != nil {
		return err
	}

	// Mac, optional dict
	macDict, err := validateDictEntry(xRefTable, dict, dictName, "Mac", OPTIONAL, pdf.V10, nil)
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
	_, err = validateStringEntry(xRefTable, dict, dictName, "CheckSum", OPTIONAL, pdf.V10, nil)

	return err
}

func validateEmbeddedFileStreamDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict) error {

	dictName := "embeddedFileStreamDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, &sd.Dict, dictName, "Type", OPTIONAL, pdf.V10, func(s string) bool { return s == "EmbeddedFile" })
	if err != nil {
		return err
	}

	// Subtype, optional, name
	_, err = validateNameEntry(xRefTable, &sd.Dict, dictName, "Subtype", OPTIONAL, pdf.V10, nil)
	if err != nil {
		return err
	}

	// Params, optional, dict
	// parameter dict containing additional file-specific information.
	if obj, found := sd.Dict.Find("Params"); found && obj != nil {
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

func validateFileSpecDictEntryEFDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	for k, obj := range *dict {

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

func validateRFDictFilesArray(xRefTable *pdf.XRefTable, arr *pdf.Array) error {

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

			_, ok := obj.(pdf.StringLiteral)
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

func validateFileSpecDictEntriesEFAndRF(xRefTable *pdf.XRefTable, efDict, rfDict *pdf.Dict) error {

	// EF only or EF and RF

	if efDict == nil {
		return errors.Errorf("validateFileSpecEntriesEFAndRF: missing required efDict.")
	}

	err := validateFileSpecDictEntryEFDict(xRefTable, efDict)
	if err != nil {
		return err
	}

	if rfDict != nil {

		for k, val := range *rfDict {

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

func validateFileSpecDictType(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	if dict.Type() == nil || (*dict.Type() != "Filespec" && (xRefTable.ValidationMode == pdf.ValidationRelaxed && *dict.Type() != "F")) {
		return errors.New("validateFileSpecDictType: missing type: FileSpec")
	}

	return nil
}

func requiredF(dosFound, macFound, unixFound bool) bool {
	return !dosFound && !macFound && !unixFound
}

func validateFileSpecDictEFAndRF(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName string) error {

	// RF, optional, dict of related files arrays, since V1.3
	rfDict, err := validateDictEntry(xRefTable, dict, dictName, "RF", OPTIONAL, pdf.V13, nil)
	if err != nil {
		return err
	}

	// EF, required if RF present, dict of embedded file streams, since 1.3
	efDict, err := validateDictEntry(xRefTable, dict, dictName, "EF", rfDict != nil, pdf.V13, nil)
	if err != nil {
		return err
	}

	// Type, required if EF present, name
	validate := func(s string) bool {
		return s == "Filespec" || (xRefTable.ValidationMode == pdf.ValidationRelaxed && s == "F")
	}
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", efDict != nil, pdf.V10, validate)
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

func validateFileSpecDict(xRefTable *pdf.XRefTable, dict *pdf.Dict) error {

	dictName := "fileSpecDict"

	// FS, optional, name
	fsName, err := validateNameEntry(xRefTable, dict, dictName, "FS", OPTIONAL, pdf.V10, nil)
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

	_, err = validateStringEntry(xRefTable, dict, dictName, "F", requiredF(dosFound, macFound, unixFound), pdf.V10, validate)
	if err != nil {
		return err
	}

	// UF, optional, text string
	sinceVersion := pdf.V17
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "UF", OPTIONAL, sinceVersion, validateFileSpecString)
	if err != nil {
		return err
	}

	// ID, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "ID", OPTIONAL, pdf.V11, func(arr pdf.Array) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	// V, optional, boolean, since V1.2
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "V", OPTIONAL, pdf.V12, nil)
	if err != nil {
		return err
	}

	err = validateFileSpecDictEFAndRF(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	// Desc, optional, text string, since V1.6
	sinceVersion = pdf.V16
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V10
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "Desc", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// CI, optional, collection item dict, since V1.7
	_, err = validateDictEntry(xRefTable, dict, dictName, "CI", OPTIONAL, pdf.V17, nil)

	return err
}

func validateFileSpecification(xRefTable *pdf.XRefTable, obj pdf.Object) (pdf.Object, error) {

	// See 7.11.4

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}

	switch obj := obj.(type) {

	case pdf.StringLiteral:
		s := obj.Value()
		if !validateFileSpecString(s) {
			return nil, errors.Errorf("validateFileSpecification: invalid file spec string: %s", s)
		}

	case pdf.HexLiteral:
		s := obj.Value()
		if !validateFileSpecString(s) {
			return nil, errors.Errorf("validateFileSpecification: invalid file spec string: %s", s)
		}

	case pdf.Dict:
		err = validateFileSpecDict(xRefTable, &obj)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.Errorf("validateFileSpecification: invalid type")

	}

	return obj, nil
}

func validateURLSpecification(xRefTable *pdf.XRefTable, obj pdf.Object) (pdf.Object, error) {

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
	_, err = validateNameEntry(xRefTable, d, dictName, "FS", REQUIRED, pdf.V10, func(s string) bool { return s == "URL" })
	if err != nil {
		return nil, err
	}

	// F, required, string, URL (Internet RFC 1738)
	_, err = validateStringEntry(xRefTable, d, dictName, "F", REQUIRED, pdf.V10, validateURLString)

	return obj, err
}

func validateFileSpecEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) (pdf.Object, error) {

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

func validateURLSpecEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName string, entryName string, required bool, sinceVersion pdf.Version) (pdf.Object, error) {

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

func validateFileSpecificationOrFormObject(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	sd, ok := obj.(pdf.StreamDict)
	if ok {
		return validateFormStreamDict(xRefTable, &sd)
	}

	_, err := validateFileSpecification(xRefTable, obj)

	return err
}

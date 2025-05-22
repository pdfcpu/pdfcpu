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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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

func validateEmbeddedFileStreamMacParameterDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "embeddedFileStreamMacParameterDict"

	// Subtype, optional integer
	// The embedded file's file type integer encoded according to Mac OS conventions.
	if _, err := validateIntegerEntry(xRefTable, d, dictName, "Subtype", OPTIONAL, model.V10, nil); err != nil {
		return err
	}

	// Creator, optional integer
	// The embedded file's creator signature integer encoded according to Mac OS conventions.
	if _, err := validateIntegerEntry(xRefTable, d, dictName, "Creator", OPTIONAL, model.V10, nil); err != nil {
		return err
	}

	// ResFork, optional stream dict
	// The binary contents of the embedded file's resource fork.
	if _, err := validateStreamDictEntry(xRefTable, d, dictName, "ResFork", OPTIONAL, model.V10, nil); err != nil {
		return err
	}

	return nil
}

func validateEmbeddedFileStreamParameterDict(xRefTable *model.XRefTable, o types.Object) error {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	dictName := "embeddedFileStreamParmDict"

	// Size, optional integer
	if _, err = validateIntegerEntry(xRefTable, d, dictName, "Size", OPTIONAL, model.V10, nil); err != nil {
		return err
	}

	// CreationDate, optional date
	if _, err = validateDateEntry(xRefTable, d, dictName, "CreationDate", OPTIONAL, model.V10); err != nil {
		return err
	}

	// ModDate, optional date
	if _, err = validateDateEntry(xRefTable, d, dictName, "ModDate", OPTIONAL, model.V10); err != nil {
		return err
	}

	// Mac, optional dict
	macDict, err := validateDictEntry(xRefTable, d, dictName, "Mac", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if macDict != nil {
		if err = validateEmbeddedFileStreamMacParameterDict(xRefTable, macDict); err != nil {
			return err
		}
	}

	// CheckSum, optional string
	_, err = validateStringEntry(xRefTable, d, dictName, "CheckSum", OPTIONAL, model.V10, nil)

	return err
}

func validateEmbeddedFileStreamDict(xRefTable *model.XRefTable, sd *types.StreamDict) error {
	dictName := "embeddedFileStreamDict"

	// Type, optional, name
	if _, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "EmbeddedFile" }); err != nil {
		return err
	}

	// Subtype, optional, name
	if _, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Subtype", OPTIONAL, model.V10, nil); err != nil {
		return err
	}

	// Params, optional, dict
	// parameter dict containing additional file-specific information.
	if o, found := sd.Dict.Find("Params"); found && o != nil {
		if err := validateEmbeddedFileStreamParameterDict(xRefTable, o); err != nil {
			return err
		}
	}

	return nil
}

func validateFileSpecDictEntriesEFAndRFKeys(k string) bool {
	return k == "F" || k == "UF" || k == "DOS" || k == "Mac" || k == "Unix" || k == "Subtype"
}

func validateFileSpecDictEntryEFDict(xRefTable *model.XRefTable, d types.Dict) error {
	for k, obj := range d {

		if !validateFileSpecDictEntriesEFAndRFKeys(k) {
			return errors.Errorf("validateFileSpecEntriesEFAndRF: invalid key: %s", k)
		}

		if k == "F" || k == "UF" {
			// value must be embedded file stream dict
			// see 7.11.4
			sd, err := validateStreamDict(xRefTable, obj)
			if err != nil {
				return err
			}
			if sd == nil {
				continue
			}

			if err = validateEmbeddedFileStreamDict(xRefTable, sd); err != nil {
				return err
			}
		}

	}

	return nil
}

func validateRFDictFilesArray(xRefTable *model.XRefTable, a types.Array) error {
	if len(a)%2 > 0 {
		return errors.New("pdfcpu: validateRFDictFilesArray: rfDict array corrupt")
	}

	for k, v := range a {

		if v == nil {
			return errors.New("pdfcpu: validateRFDictFilesArray: rfDict, array entry nil")
		}

		o, err := xRefTable.Dereference(v)
		if err != nil {
			return err
		}

		if o == nil {
			return errors.New("pdfcpu: validateRFDictFilesArray: rfDict, array entry nil")
		}

		if k%2 > 0 {

			_, ok := o.(types.StringLiteral)
			if !ok {
				return errors.New("pdfcpu: validateRFDictFilesArray: rfDict, array entry corrupt")
			}

		} else {

			// value must be embedded file stream dict
			// see 7.11.4
			sd, err := validateStreamDict(xRefTable, o)
			if err != nil {
				return err
			}

			if err = validateEmbeddedFileStreamDict(xRefTable, sd); err != nil {
				return err
			}

		}
	}

	return nil
}

func validateFileSpecDictEntriesEFAndRF(xRefTable *model.XRefTable, efDict, rfDict types.Dict) error {
	// EF only or EF and RF

	if efDict == nil {
		return errors.Errorf("pdfcpu: validateFileSpecEntriesEFAndRF: missing required efDict.")
	}

	if err := validateFileSpecDictEntryEFDict(xRefTable, efDict); err != nil {
		return err
	}

	for k, val := range rfDict {

		if _, ok := efDict.Find(k); !ok {
			return errors.Errorf("pdfcpu: validateFileSpecEntriesEFAndRF: rfDict entry=%s missing corresponding efDict entry\n", k)
		}

		// value must be related files array.
		// see 7.11.4.2
		a, err := xRefTable.DereferenceArray(val)
		if err != nil {
			return err
		}

		if a == nil {
			continue
		}

		if err = validateRFDictFilesArray(xRefTable, a); err != nil {
			return err
		}

	}

	return nil
}

func requiredF(dosFound, macFound, unixFound bool) bool {
	return !dosFound && !macFound && !unixFound
}

func validateFileSpecDictEFAndRF(xRefTable *model.XRefTable, d types.Dict, dictName string, hasEP bool) error {
	// RF, optional, dict of related files arrays, since V1.3
	rfDict, err := validateDictEntry(xRefTable, d, dictName, "RF", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}

	// EF, required if RF present, dict of embedded file streams, since 1.3
	efDict, err := validateDictEntry(xRefTable, d, dictName, "EF", rfDict != nil, model.V13, nil)
	if err != nil {
		return err
	}

	// Type, required if EF, EP or RF present, name
	validate := func(s string) bool {
		return s == "Filespec" || (xRefTable.ValidationMode == model.ValidationRelaxed && s == "F")
	}
	required := rfDict != nil || efDict != nil || hasEP
	if _, err = validateNameEntry(xRefTable, d, dictName, "Type", required, model.V10, validate); err != nil {
		return err
	}

	if efDict != nil {
		err = validateFileSpecDictEntriesEFAndRF(xRefTable, efDict, rfDict)
	}

	return err
}

func validateFileSpecDictPart1(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// FS, optional, name
	fsName, err := validateNameEntry(xRefTable, d, dictName, "FS", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// UF, optional, text string
	sinceVersion := model.V17
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	uf, err := validateStringEntry(xRefTable, d, dictName, "UF", OPTIONAL, sinceVersion, validateFileSpecString)
	if err != nil {
		return err
	}

	// DOS, byte string, optional, obsolescent.
	_, dosFound := d.Find("DOS")

	// Mac, byte string, optional, obsolescent.
	_, macFound := d.Find("Mac")

	// Unix, byte string, optional, obsolescent.
	_, unixFound := d.Find("Unix")

	// F, file spec string
	validate := validateFileSpecString
	if fsName != nil && fsName.Value() == "URL" {
		validate = validateURLString
	}

	required := requiredF(dosFound, macFound, unixFound)
	if xRefTable.ValidationMode == model.ValidationRelaxed && uf != nil {
		required = OPTIONAL
	}
	if _, err = validateStringEntry(xRefTable, d, dictName, "F", required, model.V10, validate); err != nil {
		return err
	}

	return nil
}

func validateFileSpecDictPart2(xRefTable *model.XRefTable, d types.Dict, dictName string) error {
	// ID, optional, array of strings
	if _, err := validateStringArrayEntry(xRefTable, d, dictName, "ID", OPTIONAL, model.V11, func(a types.Array) bool { return len(a) == 2 }); err != nil {
		return err
	}

	// V, optional, boolean, since V1.2
	if _, err := validateBooleanEntry(xRefTable, d, dictName, "V", OPTIONAL, model.V12, nil); err != nil {
		return err
	}

	// EP, optional, encrypted payload dict, since V2.0
	epDict, err := validateDictEntry(xRefTable, d, dictName, "EP", OPTIONAL, model.V20, nil)
	if err != nil {
		return err
	}
	if err = validateFileSpecDictEFAndRF(xRefTable, d, dictName, len(epDict) > 0); err != nil {
		return err
	}

	// Desc, optional, text string, since V1.6
	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}
	if _, err = validateStringEntry(xRefTable, d, dictName, "Desc", OPTIONAL, sinceVersion, nil); err != nil {
		return err
	}

	// CI, optional, collection item dict, since V1.7
	if _, err = validateDictEntry(xRefTable, d, dictName, "CI", OPTIONAL, model.V17, nil); err != nil {
		return err
	}

	// Thumb, optional, thumbnail image, since V2.0
	sinceVersion = model.V20
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V17
	}
	if _, err := validateStreamDictEntry(xRefTable, d, dictName, "Thumb", OPTIONAL, sinceVersion, nil); err != nil {
		return err
	}

	// AFRelationship, optional, associated file semantics, since V2.0
	validateAFRelationship := func(s string) bool {
		return types.MemberOf(s, []string{"Source", "Data", "Alternative", "Supplement", "EncryptedPayload", "FormData", "Schema", "Unspecified"})
	}
	sinceVersion = model.V20
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	if _, err := validateNameEntry(xRefTable, d, dictName, "AFRelationship", OPTIONAL, sinceVersion, validateAFRelationship); err != nil {
		return err
	}

	return nil
}

func validateFileSpecDict(xRefTable *model.XRefTable, d types.Dict) error {
	// See 7.11.3

	dictName := "fileSpecDict"

	if err := validateFileSpecDictPart1(xRefTable, d, dictName); err != nil {
		return err
	}

	if err := validateFileSpecDictPart2(xRefTable, d, dictName); err != nil {
		return err
	}

	return nil
}

func validateFileSpecification(xRefTable *model.XRefTable, o types.Object) (types.Object, error) {
	// See 7.11

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	switch o := o.(type) {

	case types.StringLiteral, types.HexLiteral:
		s := o.(interface{ Value() string }).Value()
		if !validateFileSpecString(s) {
			return nil, errors.Errorf("pdfcpu: validateFileSpecification: invalid file spec string: %s", s)
		}

	case types.Dict:
		if err = validateFileSpecDict(xRefTable, o); err != nil {
			return nil, err
		}

	default:
		return nil, errors.Errorf("pdfcpu: validateFileSpecification: invalid type")

	}

	return o, nil
}

func validateURLSpecification(xRefTable *model.XRefTable, o types.Object) (types.Object, error) {
	// See 7.11.4

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	if d == nil {
		return nil, errors.New("pdfcpu: validateURLSpecification: missing dict")
	}

	dictName := "urlSpec"

	// FS, required, name
	if _, err = validateNameEntry(xRefTable, d, dictName, "FS", REQUIRED, model.V10, func(s string) bool { return s == "URL" }); err != nil {
		return nil, err
	}

	// F, required, string, URL (Internet RFC 1738)
	_, err = validateStringEntry(xRefTable, d, dictName, "F", REQUIRED, model.V10, validateURLString)

	return o, err
}

func validateFileSpecEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (types.Object, error) {
	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return nil, err
	}

	if err = xRefTable.ValidateVersion("fileSpec", sinceVersion); err != nil {
		return nil, err
	}

	return validateFileSpecification(xRefTable, o)
}

func validateURLSpecEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, entryName string, required bool, sinceVersion model.Version) (types.Object, error) {
	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return nil, err
	}

	if err = xRefTable.ValidateVersion("URLSpec", sinceVersion); err != nil {
		return nil, err
	}

	return validateURLSpecification(xRefTable, o)
}

func validateFileSpecificationOrFormObject(xRefTable *model.XRefTable, obj types.Object) error {
	sd, ok := obj.(types.StreamDict)
	if ok {
		return validateFormStreamDict(xRefTable, &sd)
	}

	_, err := validateFileSpecification(xRefTable, obj)

	return err
}

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
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateTilingPatternDict(xRefTable *pdf.XRefTable, streamDict *pdf.StreamDict, sinceVersion pdf.Version) error {

	dictName := "tilingPatternDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	dict := streamDict.Dict

	_, err = validateNameEntry(xRefTable, &dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, &dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, &dict, dictName, "XStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return err
	}
	_, err = validateNumberEntry(xRefTable, &dict, dictName, "YStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	obj, ok := streamDict.Find("Resources")
	if !ok {
		return errors.New("validateTilingPatternDict: missing required entry Resources")
	}

	_, err = validateResourceDict(xRefTable, obj)

	return err
}

func validateShadingPatternDict(xRefTable *pdf.XRefTable, dict *pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "shadingPatternDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	d, err := validateDictEntry(xRefTable, dict, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateExtGStateDict(xRefTable, *d)
		if err != nil {
			return err
		}
	}

	// Shading: required, dict or stream dict.
	obj, ok := dict.Find("Shading")
	if !ok {
		return errors.Errorf("validateShadingPatternDict: missing required entry \"Shading\".")
	}

	return validateShading(xRefTable, obj)
}

func validatePattern(xRefTable *pdf.XRefTable, obj pdf.Object) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Dict:
		err = validateShadingPatternDict(xRefTable, &obj, pdf.V13)

	case pdf.StreamDict:
		err = validateTilingPatternDict(xRefTable, &obj, pdf.V10)

	default:
		err = errors.New("validatePattern: corrupt obj typ, must be dict or stream dict")

	}

	return err
}

func validatePatternResourceDict(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	// see 8.7 Patterns

	// Version check
	err := xRefTable.ValidateVersion("PatternResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Iterate over pattern resource dictionary
	for _, obj := range *dict {

		// Process pattern
		err = validatePattern(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

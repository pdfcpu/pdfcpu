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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateTilingPatternDict(xRefTable *pdf.XRefTable, sd *pdf.StreamDict, sinceVersion pdf.Version) error {

	dictName := "tilingPatternDict"

	// Version check
	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, sd.Dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, sd.Dict, dictName, "XStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return err
	}

	_, err = validateNumberEntry(xRefTable, sd.Dict, dictName, "YStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	o, ok := sd.Find("Resources")
	if !ok {
		return errors.New("pdfcpu: validateTilingPatternDict: missing required entry Resources")
	}

	_, err = validateResourceDict(xRefTable, o)

	return err
}

func validateShadingPatternDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "shadingPatternDict"

	err := xRefTable.ValidateVersion(dictName, sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, d, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Matrix", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 6 })
	if err != nil {
		return err
	}

	d1, err := validateDictEntry(xRefTable, d, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateExtGStateDict(xRefTable, d1)
		if err != nil {
			return err
		}
	}

	// Shading: required, dict or stream dict.
	o, ok := d.Find("Shading")
	if !ok {
		return errors.Errorf("pdfcpu: validateShadingPatternDict: missing required entry \"Shading\".")
	}

	return validateShading(xRefTable, o)
}

func validatePattern(xRefTable *pdf.XRefTable, o pdf.Object) error {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Dict:
		err = validateShadingPatternDict(xRefTable, o, pdf.V13)

	case pdf.StreamDict:
		err = validateTilingPatternDict(xRefTable, &o, pdf.V10)

	default:
		err = errors.New("pdfcpu: validatePattern: corrupt obj typ, must be dict or stream dict")

	}

	return err
}

func validatePatternResourceDict(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	// see 8.7 Patterns

	// Version check
	err := xRefTable.ValidateVersion("PatternResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Iterate over pattern resource dictionary
	for _, o := range d {

		// Process pattern
		err = validatePattern(xRefTable, o)
		if err != nil {
			return err
		}

	}

	return nil
}

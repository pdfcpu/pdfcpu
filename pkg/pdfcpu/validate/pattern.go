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
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateTilingPatternDict(xRefTable *model.XRefTable, sd *types.StreamDict, sinceVersion model.Version) error {
	dictName := "tilingPatternDict"

	if err := xRefTable.ValidateVersion(dictName, sinceVersion); err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return fmt.Errorf("%s.PatternType: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.PaintType: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.TilingType: %w", dictName, err)
	}

	_, err = validateRectangleEntry(xRefTable, sd.Dict, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.BBox: %w", dictName, err)
	}

	_, err = validateNumberEntry(xRefTable, sd.Dict, dictName, "XStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return fmt.Errorf("%s.XStep: %w", dictName, err)
	}

	_, err = validateNumberEntry(xRefTable, sd.Dict, dictName, "YStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return fmt.Errorf("%s.YStep: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	o, ok := sd.Find("Resources")
	if !ok {
		return fmt.Errorf("%s.Resources: missing required entry", dictName)
	}

	_, err = validateResourceDict(xRefTable, o)
	if err != nil {
		return fmt.Errorf("%s.Resources: %w", dictName, err)
	}
	return nil
}

func validateShadingPatternDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "shadingPatternDict"

	if err := xRefTable.ValidateVersion(dictName, sinceVersion); err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, d, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return fmt.Errorf("%s.PatternType: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	d1, err := validateDictEntry(xRefTable, d, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.ExtGState: %w", dictName, err)
	}

	if d1 != nil {
		err = validateExtGStateDict(xRefTable, d1)
		if err != nil {
			return fmt.Errorf("%s.ExtGState: %w", dictName, err)
		}
	}

	// Shading: required, dict or stream dict.
	o, ok := d.Find("Shading")
	if !ok {
		return fmt.Errorf("%s.Shading: missing required entry", dictName)
	}

	if err := validateShading(xRefTable, o); err != nil {
		return fmt.Errorf("%s.Shading: %w", dictName, err)
	}
	return nil
}

func validatePattern(xRefTable *model.XRefTable, o types.Object) error {
	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("pattern: dereference: %w", err)
		}
		return nil
	}

	switch o := o.(type) {

	case types.StreamDict:
		if err = validateTilingPatternDict(xRefTable, &o, model.V10); err != nil {
			return fmt.Errorf("tiling pattern: %w", err)
		}

	case types.Dict:
		if err = validateShadingPatternDict(xRefTable, o, model.V13); err != nil {
			return fmt.Errorf("shading pattern: %w", err)
		}

	default:
		err = errors.New("corrupt obj type, must be dict or stream dict")

	}

	return err
}

func validatePatternResourceDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 8.7 Patterns

	// Version check
	if err := xRefTable.ValidateVersion("PatternResourceDict", sinceVersion); err != nil {
		return fmt.Errorf("patternResourceDict: %w", err)
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		if err != nil {
			return fmt.Errorf("patternResourceDict: dereference: %w", err)
		}
		return nil
	}

	// Iterate over pattern resource dictionary
	for name, o := range d {
		// Process pattern
		if err = validatePattern(xRefTable, o); err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("patternResourceDict.%s", name), o), err)
		}

	}

	return nil
}

/*
Copyright 2026 The pdfcpu Authors.

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
	"fmt"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func arrayCardinalityError(dictName, entryName string, objNr, actual int, expected string) error {
	err := fmt.Errorf("%s.%s: invalid array length %d, expected %s", dictName, entryName, actual, expected)
	return model.WithValidationErrorObject(err, objNr)
}

func validateArrayExactLength(a types.Array, objNr int, dictName, entryName string, length int) error {
	if len(a) == length {
		return nil
	}
	return arrayCardinalityError(dictName, entryName, objNr, len(a), strconv.Itoa(length))
}

func validateArrayAllowedLengths(a types.Array, objNr int, dictName, entryName string, lengths ...int) error {
	for _, length := range lengths {
		if len(a) == length {
			return nil
		}
	}
	return arrayCardinalityError(dictName, entryName, objNr, len(a), fmt.Sprintf("one of %v", lengths))
}

// validateArrayPairs checks pair completion and a caller-specified minimum without inspecting element types.
func validateArrayPairs(a types.Array, objNr int, dictName, entryName string, minPairs int) error {
	if len(a)%2 == 0 && len(a)/2 >= minPairs {
		return nil
	}
	expected := fmt.Sprintf("complete pairs, minimum pair count %d", minPairs)
	return arrayCardinalityError(dictName, entryName, objNr, len(a), expected)
}

func validateMultiLanguageTextEntry(x *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, sinceVersion model.Version) error {
	a, err := validateStringArrayEntry(x, d, ownerObjNr, dictName, entryName, OPTIONAL, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	if err = validateArrayPairs(a, objNr, dictName, entryName, 1); err != nil {
		return err
	}
	seen := map[string]bool{}
	for i := 0; i < len(a); i += 2 {
		entryObjNr := validationObjectNumber(objNr, a[i])
		o, err := x.Dereference(a[i])
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: dereference language identifier: %w", dictName, entryName, i, err)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		language, err := types.StringOrHexLiteral(o)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: expected language identifier string", dictName, entryName, i)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		key := strings.ToLower(*language)
		if seen[key] {
			err = fmt.Errorf("%s.%s[%d]: duplicate language identifier %q", dictName, entryName, i, *language)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		seen[key] = true
	}
	return nil
}

func validateUnitIntervalArrayEntry(x *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, sinceVersion model.Version, lengths ...int) error {
	a, err := validateArrayEntry(x, d, ownerObjNr, dictName, entryName, OPTIONAL, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	if len(lengths) == 1 {
		err = validateArrayExactLength(a, objNr, dictName, entryName, lengths[0])
	} else {
		err = validateArrayAllowedLengths(a, objNr, dictName, entryName, lengths...)
	}
	if err != nil {
		return err
	}
	for i, o := range a {
		n, err := x.DereferenceNumber(o)
		if err != nil {
			err = fmt.Errorf("%s.%s[%d]: expected number: %w", dictName, entryName, i, err)
		} else if x.ValidationMode == model.ValidationStrict && !(n >= 0 && n <= 1) {
			err = fmt.Errorf("%s.%s[%d]: invalid value %g, expected 0 through 1", dictName, entryName, i, n)
		}
		if err != nil {
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, o))
		}
	}
	return nil
}

func validateColorArrayEntry(x *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, sinceVersion model.Version) error {
	return validateUnitIntervalArrayEntry(x, d, ownerObjNr, dictName, entryName, sinceVersion, 0, 1, 3, 4)
}

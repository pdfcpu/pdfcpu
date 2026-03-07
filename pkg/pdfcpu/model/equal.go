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

package model

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func appendPair(pairs []int, a, b int) []int {
	if a > b {
		a, b = b, a
	}
	return append(pairs, a, b)
}

func containsPair(pairs []int, a, b int) bool {
	if a > b {
		a, b = b, a
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i] == a && pairs[i+1] == b {
			return true
		}
	}
	return false
}

// EqualObjects returns true if two objects are equal in the context of xrefTable.
// An object and an indirect reference to it are treated as equal.
// Objects may be object trees.
func EqualObjects(o1, o2 types.Object, xRefTable *XRefTable, pairs []int) (ok bool, err error) {
	ir1, ok := o1.(types.IndirectRef)
	if ok {
		ir2, ok := o2.(types.IndirectRef)
		if ok {
			if ir1 == ir2 {
				return true, nil
			}
			objNr1, objNr2 := ir1.ObjectNumber.Value(), ir2.ObjectNumber.Value()
			if len(pairs) > 0 {
				if containsPair(pairs, objNr1, objNr2) {
					return true, nil
				}
			} else {
				pairs = make([]int, 0, 6)
			}
			pairs = appendPair(pairs, objNr1, objNr2)
		}
	}

	o1, err = xRefTable.Dereference(o1)
	if err != nil {
		return false, err
	}

	o2, err = xRefTable.Dereference(o2)
	if err != nil {
		return false, err
	}

	if o1 == nil {
		return o2 != nil, nil
	}

	o1Type := fmt.Sprintf("%T", o1)
	o2Type := fmt.Sprintf("%T", o2)
	if o1Type != o2Type {
		return false, nil
	}

	switch o1.(type) {

	case types.Name, types.StringLiteral, types.HexLiteral,
		types.Integer, types.Float, types.Boolean:
		ok = o1 == o2

	case types.Dict:
		ok, err = equalDicts(o1.(types.Dict), o2.(types.Dict), xRefTable, pairs)

	case types.StreamDict:
		sd1 := o1.(types.StreamDict)
		sd2 := o2.(types.StreamDict)
		ok, err = equalStreamDicts(&sd1, &sd2, xRefTable, pairs)

	case types.Array:
		ok, err = equalArrays(o1.(types.Array), o2.(types.Array), xRefTable, pairs)

	default:
		err = errors.Errorf("equalObjects: unhandled compare for type %s\n", o1Type)
	}

	return ok, err
}

func equalArrays(a1, a2 types.Array, xRefTable *XRefTable, pairs []int) (bool, error) {
	if len(a1) != len(a2) {
		return false, nil
	}

	for i, o1 := range a1 {
		ok, err := EqualObjects(o1, a2[i], xRefTable, pairs)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// equalStreamDicts returns true if two stream dicts are equal and contain the same bytes.
func equalStreamDicts(sd1, sd2 *types.StreamDict, xRefTable *XRefTable, pairs []int) (bool, error) {
	ok, err := equalDicts(sd1.Dict, sd2.Dict, xRefTable, pairs)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil
	}

	if sd1.Raw == nil || sd2 == nil {
		return false, errors.New("pdfcpu: EqualStreamDicts: stream dict not loaded")
	}

	return bytes.Equal(sd1.Raw, sd2.Raw), nil
}

func equalFontNames(v1, v2 types.Object, xRefTable *XRefTable) (bool, error) {
	v1, err := xRefTable.Dereference(v1)
	if err != nil {
		return false, err
	}

	bf1, ok := v1.(types.Name)
	if !ok {
		return false, errors.Errorf("equalFontNames: type cast problem")
	}

	v2, err = xRefTable.Dereference(v2)
	if err != nil {
		return false, err
	}

	bf2, ok := v2.(types.Name)
	if !ok {
		return false, errors.Errorf("equalFontNames: type cast problem")
	}

	// Ignore fontname prefix
	i := strings.Index(string(bf1), "+")
	if i > 0 {
		bf1 = bf1[i+1:]
	}
	i = strings.Index(string(bf2), "+")
	if i > 0 {
		bf2 = bf2[i+1:]
	}

	return bf1 == bf2, nil
}

func equalDicts(d1, d2 types.Dict, xRefTable *XRefTable, pairs []int) (bool, error) {
	if d1.Len() != d2.Len() {
		return false, nil
	}

	t1, t2 := d1.Type(), d2.Type()
	fontDicts := (t1 != nil && *t1 == "Font") && (t2 != nil && *t2 == "Font")

	for key, v1 := range d1 {

		v2, found := d2[key]
		if !found {
			return false, nil
		}

		// Special treatment for font dicts
		if fontDicts && (key == "BaseFont" || key == "FontName" || key == "Name") {
			ok, err := equalFontNames(v1, v2, xRefTable)
			if err != nil {
				return false, err
			}

			if !ok {
				return false, nil
			}

			continue
		}

		ok, err := EqualObjects(v1, v2, xRefTable, pairs)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

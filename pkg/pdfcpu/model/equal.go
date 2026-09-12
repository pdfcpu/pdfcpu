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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func appendPair(pairs []int, a, b int) []int {
	if a > b {
		a, b = b, a
	}
	return append(pairs, a, b)
}

func containsPair(c context.Context, pairs []int, a, b int) (bool, error) {
	if a > b {
		a, b = b, a
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		if pairs[i] == a && pairs[i+1] == b {
			return true, nil
		}
	}
	return false, nil
}

// EqualObjects returns true if two objects are equal in the context of xrefTable.
// An object and an indirect reference to it are treated as equal.
// Objects may be object trees. Comparison stops when c is canceled or its deadline expires.
func EqualObjects(c context.Context, o1, o2 types.Object, xRefTable *XRefTable, pairs []int) (ok bool, err error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	pairs, ok, err = equalObjectReferences(c, o1, o2, pairs)
	if err != nil || ok {
		return ok, err
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
		ok, err = equalDicts(c, o1.(types.Dict), o2.(types.Dict), xRefTable, pairs)

	case types.StreamDict:
		sd1 := o1.(types.StreamDict)
		sd2 := o2.(types.StreamDict)
		ok, err = equalStreamDicts(c, &sd1, &sd2, xRefTable, pairs)

	case types.Array:
		ok, err = equalArrays(c, o1.(types.Array), o2.(types.Array), xRefTable, pairs)

	default:
		err = fmt.Errorf("unhandled compare for type %s", o1Type)
	}

	return ok, err
}

func equalObjectReferences(c context.Context, o1, o2 types.Object, pairs []int) ([]int, bool, error) {
	ir1, ok1 := o1.(types.IndirectRef)
	ir2, ok2 := o2.(types.IndirectRef)
	if !ok1 || !ok2 {
		return pairs, false, nil
	}
	if ir1 == ir2 {
		return pairs, true, nil
	}
	a, b := ir1.ObjectNumber.Value(), ir2.ObjectNumber.Value()
	found, err := containsPair(c, pairs, a, b)
	if err != nil || found {
		return pairs, found, err
	}
	return appendPair(pairs, a, b), false, nil
}

func equalArrays(c context.Context, a1, a2 types.Array, xRefTable *XRefTable, pairs []int) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if len(a1) != len(a2) {
		return false, nil
	}

	for i, o1 := range a1 {
		ok, err := EqualObjects(c, o1, a2[i], xRefTable, pairs)
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
func equalStreamDicts(c context.Context, sd1, sd2 *types.StreamDict, xRefTable *XRefTable, pairs []int) (bool, error) {
	ok, err := equalDicts(c, sd1.Dict, sd2.Dict, xRefTable, pairs)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil
	}

	if sd1.Raw == nil || sd2 == nil {
		return false, errors.New("stream dict not loaded")
	}

	return equalStreamBytes(c, sd1.Raw, sd2.Raw)
}

func equalStreamBytes(c context.Context, a, b []byte) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if len(a) != len(b) {
		return false, nil
	}
	const chunkSize = 64 * 1024
	for start := 0; start < len(a); start += chunkSize {
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		end := start + min(chunkSize, len(a)-start)
		if !bytes.Equal(a[start:end], b[start:end]) {
			return false, nil
		}
	}
	return true, contextutil.Check(c)
}

func equalFontNames(v1, v2 types.Object, xRefTable *XRefTable) (bool, error) {
	v1, err := xRefTable.Dereference(v1)
	if err != nil {
		return false, err
	}

	bf1, ok := v1.(types.Name)
	if !ok {
		return false, fmt.Errorf("type cast problem")
	}

	v2, err = xRefTable.Dereference(v2)
	if err != nil {
		return false, err
	}

	bf2, ok := v2.(types.Name)
	if !ok {
		return false, fmt.Errorf("type cast problem")
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

func fontDictPair(d1, d2 types.Dict, xRefTable *XRefTable) (bool, error) {
	t1, _, err := xRefTable.DereferenceNameEntry(d1, "Type")
	if err != nil {
		return false, fmt.Errorf("first dict Type: %w", err)
	}
	t2, _, err := xRefTable.DereferenceNameEntry(d2, "Type")
	if err != nil {
		return false, fmt.Errorf("second dict Type: %w", err)
	}

	return t1 != nil && t1.Value() == "Font" && t2 != nil && t2.Value() == "Font", nil
}

func fontNameEntry(fontDicts bool, key string) bool {
	if !fontDicts {
		return false
	}
	return key == "BaseFont" || key == "FontName" || key == "Name"
}

func equalDicts(c context.Context, d1, d2 types.Dict, xRefTable *XRefTable, pairs []int) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if d1.Len() != d2.Len() {
		return false, nil
	}

	fontDicts, err := fontDictPair(d1, d2, xRefTable)
	if err != nil {
		return false, err
	}

	for _, key := range slices.Sorted(maps.Keys(d1)) {
		if err := contextutil.Check(c); err != nil {
			return false, err
		}
		v1 := d1[key]

		v2, found := d2[key]
		if !found {
			return false, nil
		}

		// Special treatment for font dicts
		if fontNameEntry(fontDicts, key) {
			ok, err := equalFontNames(v1, v2, xRefTable)
			if err != nil {
				return false, fmt.Errorf("dict entry %s: %w", key, err)
			}

			if !ok {
				return false, nil
			}

			continue
		}

		ok, err := EqualObjects(c, v1, v2, xRefTable, pairs)
		if err != nil {
			return false, fmt.Errorf("dict entry %s: %w", key, err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

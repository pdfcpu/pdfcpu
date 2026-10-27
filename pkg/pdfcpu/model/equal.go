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
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var errObjectComparisonLimit = errors.New("object comparison limit exceeded")

type objectPair struct {
	a int
	b int
}

func newObjectPair(a, b int) objectPair {
	if a > b {
		a, b = b, a
	}
	return objectPair{a, b}
}

type objectComparator struct {
	xRefTable *XRefTable
	pairs     map[objectPair]struct{}
	pairCount int
	pairLimit int
}

func newObjectComparator(c context.Context, xRefTable *XRefTable, pairs []int) (objectComparator, error) {
	pairLimit := 1
	if xRefTable != nil && len(xRefTable.Table) > pairLimit {
		// Equal objects with isomorphic reference graphs need at most one pair per object.
		// Larger comparisons are treated as unequal.
		pairLimit = len(xRefTable.Table)
	}
	oc := objectComparator{
		xRefTable: xRefTable,
		pairLimit: pairLimit,
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		if err := contextutil.Check(c); err != nil {
			return objectComparator{}, err
		}
		if oc.pairs == nil {
			oc.pairs = map[objectPair]struct{}{}
		}
		oc.pairs[newObjectPair(pairs[i], pairs[i+1])] = struct{}{}
	}
	return oc, nil
}

// EqualObjects returns true if two objects are equal in the context of xrefTable.
// An object and an indirect reference to it are treated as equal.
// Objects may be object trees.
// Exceeding the configured recursion depth or the indirect-reference pair budget, which equals the xref table size,
// returns false without an error.
// Canceling c or reaching its deadline returns an error.
// The pairs argument contains indirect-object-number pairs already being compared by the caller.
func EqualObjects(c context.Context, o1, o2 types.Object, xRefTable *XRefTable, pairs []int) (bool, error) {
	oc, err := newObjectComparator(c, xRefTable, pairs)
	if err != nil {
		return false, err
	}

	ok, err := oc.equalObjects(c, o1, o2, 0)
	if errors.Is(err, errObjectComparisonLimit) {
		return false, nil
	}

	return ok, err
}

func (oc *objectComparator) equalObjects(c context.Context, o1, o2 types.Object, depth int) (ok bool, err error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}

	maxDepth := oc.maxRecursionDepth()
	if err := CheckRecursionDepth("object comparison", depth, maxDepth); err != nil {
		if log.OptimizeEnabled() {
			log.Optimize.Printf(
				"EqualObjects: recursion depth %d exceeds limit %d; treating objects as unequal\n", depth, maxDepth)
		}
		return false, errObjectComparisonLimit
	}

	ok, err = oc.equalObjectReferences(o1, o2)
	if err != nil || ok {
		return ok, err
	}

	o1, err = oc.xRefTable.Dereference(o1)
	if err != nil {
		return false, err
	}

	o2, err = oc.xRefTable.Dereference(o2)
	if err != nil {
		return false, err
	}

	if o1 == nil {
		return o2 == nil, nil
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
		ok, err = oc.equalDicts(c, o1.(types.Dict), o2.(types.Dict), depth)

	case types.StreamDict:
		sd1 := o1.(types.StreamDict)
		sd2 := o2.(types.StreamDict)
		ok, err = oc.equalStreamDicts(c, &sd1, &sd2, depth)

	case types.Array:
		ok, err = oc.equalArrays(c, o1.(types.Array), o2.(types.Array), depth)

	default:
		err = fmt.Errorf("unhandled compare for type %s", o1Type)
	}

	return ok, err
}

func (oc *objectComparator) maxRecursionDepth() int {
	if oc.xRefTable == nil {
		return DefaultResourceLimits().MaxRecursionDepth
	}
	return oc.xRefTable.MaxRecursionDepth()
}

func (oc *objectComparator) equalObjectReferences(o1, o2 types.Object) (bool, error) {
	ir1, ok1 := o1.(types.IndirectRef)
	ir2, ok2 := o2.(types.IndirectRef)
	if !ok1 || !ok2 {
		return false, nil
	}

	if ir1 == ir2 {
		return true, nil
	}

	pair := newObjectPair(ir1.ObjectNumber.Value(), ir2.ObjectNumber.Value())
	if _, found := oc.pairs[pair]; found {
		return true, nil
	}

	if oc.pairCount >= oc.pairLimit {
		if log.OptimizeEnabled() {
			log.Optimize.Printf(
				"EqualObjects: indirect-reference pair budget %d reached; treating objects as unequal\n", oc.pairLimit)
		}
		return false, errObjectComparisonLimit
	}

	if oc.pairs == nil {
		oc.pairs = map[objectPair]struct{}{}
	}
	oc.pairs[pair] = struct{}{}
	oc.pairCount++

	return false, nil
}

func (oc *objectComparator) equalArrays(c context.Context, a1, a2 types.Array, depth int) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}

	if len(a1) != len(a2) {
		return false, nil
	}

	for i, o1 := range a1 {
		ok, err := oc.equalObjects(c, o1, a2[i], depth+1)
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
func (oc *objectComparator) equalStreamDicts(c context.Context, sd1, sd2 *types.StreamDict, depth int) (bool, error) {
	ok, err := oc.equalDicts(c, sd1.Dict, sd2.Dict, depth)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, nil
	}

	if sd1.Raw == nil || sd2.Raw == nil {
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

func (oc *objectComparator) equalDicts(c context.Context, d1, d2 types.Dict, depth int) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}

	if d1.Len() != d2.Len() {
		return false, nil
	}

	fontDicts, err := fontDictPair(d1, d2, oc.xRefTable)
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
			ok, err := equalFontNames(v1, v2, oc.xRefTable)
			if err != nil {
				return false, fmt.Errorf("dict entry %s: %w", key, err)
			}

			if !ok {
				return false, nil
			}

			continue
		}

		ok, err := oc.equalObjects(c, v1, v2, depth+1)
		if err != nil {
			return false, fmt.Errorf("dict entry %s: %w", key, err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

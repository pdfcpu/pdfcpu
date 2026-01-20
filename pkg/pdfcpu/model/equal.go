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
	"strings"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// comparisonKey is used to track object pairs being compared to detect cycles.
// Uses object numbers for indirect refs, pointer addresses for direct objects.
type comparisonKey struct {
	id1 uintptr
	id2 uintptr
}

// getObjectID returns a unique identifier for an object for cycle detection.
// For indirect refs, uses object number. For direct objects, uses pointer address.
func getObjectID(o types.Object, xRefTable *XRefTable) uintptr {
	if ir, ok := o.(types.IndirectRef); ok {
		// Use object number for indirect refs
		return uintptr(ir.ObjectNumber)
	}
	// For direct objects, use pointer address
	return uintptr(unsafe.Pointer(&o))
}

// EqualObjects returns true if two objects are equal in the context of given xrefTable.
// Some object and an indirect reference to it are treated as equal.
// Objects may in fact be object trees.
func EqualObjects(o1, o2 types.Object, xRefTable *XRefTable) (ok bool, err error) {
	visited := make(map[comparisonKey]bool)
	return equalObjectsWithVisited(o1, o2, xRefTable, visited)
}

// equalObjectsWithVisited is the internal implementation with cycle detection.
func equalObjectsWithVisited(o1, o2 types.Object, xRefTable *XRefTable, visited map[comparisonKey]bool) (ok bool, err error) {

	//log.Debug.Printf("equalObjects: comparing %T with %T \n", o1, o2)

	// Check for cycle before dereferencing
	id1 := getObjectID(o1, xRefTable)
	id2 := getObjectID(o2, xRefTable)
	key := comparisonKey{id1: id1, id2: id2}
	keyRev := comparisonKey{id1: id2, id2: id1}

	// If we're already comparing this pair, assume equal to break cycle
	if visited[key] || visited[keyRev] {
		return true, nil
	}

	ir1, ok := o1.(types.IndirectRef)
	if ok {
		ir2, ok := o2.(types.IndirectRef)
		if ok && ir1 == ir2 {
			return true, nil
		}
	}

	// Mark as being compared before dereferencing
	visited[key] = true
	defer delete(visited, key)

	o1Deref, err := xRefTable.Dereference(o1)
	if err != nil {
		return false, err
	}

	o2Deref, err := xRefTable.Dereference(o2)
	if err != nil {
		return false, err
	}

	if o1Deref == nil {
		return o2Deref == nil, nil
	}

	// Use type switch to check types match and handle comparison in one pass.
	// This avoids fmt.Sprintf("%T", ...) which can cause stack overflow with circular refs.
	switch v1 := o1Deref.(type) {
	case types.Name, types.StringLiteral, types.HexLiteral,
		types.Integer, types.Float, types.Boolean:
		// For primitive types, check o2 is same type and compare values
		switch o2Deref.(type) {
		case types.Name, types.StringLiteral, types.HexLiteral,
			types.Integer, types.Float, types.Boolean:
			ok = v1 == o2Deref
		default:
			return false, nil // Different types
		}

	case types.Dict:
		d2, isDict := o2Deref.(types.Dict)
		if !isDict {
			return false, nil // Different types
		}
		ok, err = equalDicts(v1, d2, xRefTable, visited)

	case types.StreamDict:
		sd2, isStreamDict := o2Deref.(types.StreamDict)
		if !isStreamDict {
			return false, nil // Different types
		}
		ok, err = equalStreamDictsWithVisited(&v1, &sd2, xRefTable, visited)

	case types.Array:
		a2, isArray := o2Deref.(types.Array)
		if !isArray {
			return false, nil // Different types
		}
		ok, err = equalArrays(v1, a2, xRefTable, visited)

	default:
		err = errors.Errorf("equalObjects: unhandled compare for type %T\n", o1Deref)
	}

	return ok, err
}

func equalArrays(a1, a2 types.Array, xRefTable *XRefTable, visited map[comparisonKey]bool) (bool, error) {

	if len(a1) != len(a2) {
		return false, nil
	}

	for i, o1 := range a1 {

		ok, err := equalObjectsWithVisited(o1, a2[i], xRefTable, visited)
		if err != nil {
			return false, err
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// EqualStreamDicts returns true if two stream dicts are equal and contain the same bytes.
func EqualStreamDicts(sd1, sd2 *types.StreamDict, xRefTable *XRefTable) (bool, error) {
	visited := make(map[comparisonKey]bool)
	return equalStreamDictsWithVisited(sd1, sd2, xRefTable, visited)
}

func equalStreamDictsWithVisited(sd1, sd2 *types.StreamDict, xRefTable *XRefTable, visited map[comparisonKey]bool) (bool, error) {

	ok, err := equalDicts(sd1.Dict, sd2.Dict, xRefTable, visited)
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

	//log.Debug.Printf("equalFontNames: bf1=%s fb2=%s\n", bf1, bf2)

	return bf1 == bf2, nil
}

func equalDicts(d1, d2 types.Dict, xRefTable *XRefTable, visited map[comparisonKey]bool) (bool, error) {

	//log.Debug.Printf("equalDicts: %v\n%v\n", d1, d2)

	if d1.Len() != d2.Len() {
		return false, nil
	}

	t1, t2 := d1.Type(), d2.Type()
	fontDicts := (t1 != nil && *t1 == "Font") && (t2 != nil && *t2 == "Font")

	for key, v1 := range d1 {

		v2, found := d2[key]
		if !found {
			//log.Debug.Printf("equalDict: return false, key=%s\n", key)
			return false, nil
		}

		// Special treatment for font dicts
		if fontDicts && (key == "BaseFont" || key == "FontName" || key == "Name") {

			ok, err := equalFontNames(v1, v2, xRefTable)
			if err != nil {
				//log.Debug.Printf("equalDict: return2 false, key=%s v1=%v\nv2=%v\n", key, v1, v2)
				return false, err
			}

			if !ok {
				//log.Debug.Printf("equalDict: return3 false, key=%s v1=%v\nv2=%v\n", key, v1, v2)
				return false, nil
			}

			continue
		}

		ok, err := equalObjectsWithVisited(v1, v2, xRefTable, visited)
		if err != nil {
			//log.Debug.Printf("equalDict: return4 false, key=%s v1=%v\nv2=%v\n%v\n", key, v1, v2, err)
			return false, err
		}

		if !ok {
			//log.Debug.Printf("equalDict: return5 false, key=%s v1=%v\nv2=%v\n", key, v1, v2)
			return false, nil
		}

	}

	//log.Debug.Println("equalDict: return true")

	return true, nil
}

// EqualFontDicts returns true, if two font dicts are equal.
func EqualFontDicts(fd1, fd2 types.Dict, xRefTable *XRefTable) (bool, error) {
	visited := make(map[comparisonKey]bool)
	return equalFontDictsWithVisited(fd1, fd2, xRefTable, visited)
}

func equalFontDictsWithVisited(fd1, fd2 types.Dict, xRefTable *XRefTable, visited map[comparisonKey]bool) (bool, error) {

	//log.Debug.Printf("EqualFontDicts: %v\n%v\n", fd1, fd2)

	if fd1 == nil {
		return fd2 == nil, nil
	}

	if fd2 == nil {
		return false, nil
	}

	ok, err := equalDicts(fd1, fd2, xRefTable, visited)
	if err != nil {
		return false, err
	}

	return ok, nil
}

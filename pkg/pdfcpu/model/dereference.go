/*
Copyright 2021 The pdfcpu Authors.

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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func (xRefTable *XRefTable) indRefToObject(ir *types.IndirectRef) (types.Object, error) {
	if ir == nil {
		return nil, errors.New("pdfcpu: indRefToObject: input argument is nil")
	}

	// 7.3.10
	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	entry, found := xRefTable.FindTableEntryForIndRef(ir)
	if !found || entry.Free {
		return nil, nil
	}

	xRefTable.CurObj = int(ir.ObjectNumber)

	// return dereferenced object
	return entry.Object, nil
}

// Dereference resolves an indirect object and returns the resulting PDF object.
func (xRefTable *XRefTable) Dereference(o types.Object) (types.Object, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		// Nothing do dereference.
		return o, nil
	}

	return xRefTable.indRefToObject(&ir)
}

// DereferenceBoolean resolves and validates a boolean object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceBoolean(o types.Object, sinceVersion Version) (*types.Boolean, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	b, ok := o.(types.Boolean)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceBoolean: wrong type <%v>", o)
	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceBoolean", sinceVersion); err != nil {
		return nil, err
	}

	return &b, nil
}

// DereferenceInteger resolves and validates an integer object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceInteger(o types.Object) (*types.Integer, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	i, ok := o.(types.Integer)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceInteger: wrong type <%v>", o)
	}

	return &i, nil
}

// DereferenceNumber resolves a number object, which may be an indirect reference and returns a float64.
func (xRefTable *XRefTable) DereferenceNumber(o types.Object) (float64, error) {

	var (
		f   float64
		err error
	)

	o, _ = xRefTable.Dereference(o)

	switch o := o.(type) {

	case types.Integer:
		f = float64(o.Value())

	case types.Float:
		f = o.Value()

	default:
		err = errors.Errorf("pdfcpu: dereferenceNumber: wrong type <%v>", o)

	}

	return f, err
}

// DereferenceName resolves and validates a name object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceName(o types.Object, sinceVersion Version, validate func(string) bool) (n types.Name, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return n, err
	}

	n, ok := o.(types.Name)
	if !ok {
		return n, errors.Errorf("pdfcpu: dereferenceName: wrong type <%v>", o)
	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceName", sinceVersion); err != nil {
		return n, err
	}

	// Validation
	if validate != nil && !validate(n.Value()) {
		return n, errors.Errorf("pdfcpu: dereferenceName: invalid <%s>", n.Value())
	}

	return n, nil
}

// DereferenceStringLiteral resolves and validates a string literal object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStringLiteral(o types.Object, sinceVersion Version, validate func(string) bool) (s types.StringLiteral, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return s, err
	}

	s, ok := o.(types.StringLiteral)
	if !ok {
		return s, errors.Errorf("pdfcpu: dereferenceStringLiteral: wrong type <%v>", o)
	}

	// Ensure UTF16 correctness.
	s1, err := types.StringLiteralToString(s)
	if err != nil {
		return s, err
	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceStringLiteral", sinceVersion); err != nil {
		return s, err
	}

	// Validation
	if validate != nil && !validate(s1) {
		return s, errors.Errorf("pdfcpu: dereferenceStringLiteral: invalid <%s>", s1)
	}

	return s, nil
}

// DereferenceStringOrHexLiteral resolves and validates a string or hex literal object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStringOrHexLiteral(obj types.Object, sinceVersion Version, validate func(string) bool) (s string, err error) {

	o, err := xRefTable.Dereference(obj)
	if err != nil || o == nil {
		return "", err
	}

	switch str := o.(type) {

	case types.StringLiteral:
		// Ensure UTF16 correctness.
		if s, err = types.StringLiteralToString(str); err != nil {
			return "", err
		}

	case types.HexLiteral:
		// Ensure UTF16 correctness.
		if s, err = types.HexLiteralToString(str); err != nil {
			return "", err
		}

	default:
		return "", errors.Errorf("pdfcpu: dereferenceStringOrHexLiteral: wrong type %T", obj)

	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceStringOrHexLiteral", sinceVersion); err != nil {
		return "", err
	}

	// Validation
	if validate != nil && !validate(s) {
		return "", errors.Errorf("pdfcpu: dereferenceStringOrHexLiteral: invalid <%s>", s)
	}

	return s, nil
}

// Text returns a string based representation for String and Hexliterals.
func Text(o types.Object) (string, error) {
	switch obj := o.(type) {
	case types.StringLiteral:
		return types.StringLiteralToString(obj)
	case types.HexLiteral:
		return types.HexLiteralToString(obj)
	default:
		return "", errors.Errorf("pdfcpu: text: corrupt -  %v\n", obj)
	}
}

// DereferenceText resolves and validates a string or hex literal object to a string.
func (xRefTable *XRefTable) DereferenceText(o types.Object) (string, error) {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", err
	}
	return Text(o)
}

func CSVSafeString(s string) string {
	return strings.Replace(s, ";", ",", -1)
}

// DereferenceCSVSafeText resolves and validates a string or hex literal object to a string.
func (xRefTable *XRefTable) DereferenceCSVSafeText(o types.Object) (string, error) {
	s, err := xRefTable.DereferenceText(o)
	if err != nil {
		return "", err
	}
	return CSVSafeString(s), nil
}

// DereferenceArray resolves and validates an array object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceArray(o types.Object) (types.Array, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	a, ok := o.(types.Array)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceArray: wrong type %T <%v>", o, o)
	}

	return a, nil
}

// DereferenceDict resolves and validates a dictionary object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceDict(o types.Object) (types.Dict, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(types.Dict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceDict: wrong type %T <%v>", o, o)
	}

	return d, nil
}

func (xRefTable *XRefTable) dereferenceDestArray(o types.Object) (types.Array, error) {
	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}
	switch o := o.(type) {
	case types.Array:
		return o, nil
	case types.Dict:
		o1, err := xRefTable.DereferenceDictEntry(o, "D")
		if err != nil {
			return nil, err
		}
		arr, ok := o1.(types.Array)
		if !ok {
			errors.Errorf("pdfcpu: corrupted dest array:\n%s\n", o)
		}
		return arr, nil
	}

	return nil, errors.Errorf("pdfcpu: corrupted dest array:\n%s\n", o)
}

// DereferenceDestArray resolves the destination for key.
func (xRefTable *XRefTable) DereferenceDestArray(key string) (types.Array, error) {
	o, ok := xRefTable.Names["Dests"].Value(key)
	if !ok {
		return nil, errors.Errorf("pdfcpu: corrupted named destination for: %s", key)
	}
	return xRefTable.dereferenceDestArray(o)
}

// DereferenceDictEntry returns a dereferenced dict entry.
func (xRefTable *XRefTable) DereferenceDictEntry(d types.Dict, key string) (types.Object, error) {
	o, found := d.Find(key)
	if !found || o == nil {
		return nil, errors.Errorf("pdfcpu: dict=%s entry=%s missing.", d, key)
	}
	return xRefTable.Dereference(o)
}

// DereferenceStringEntryBytes returns the bytes of a string entry of d.
func (xRefTable *XRefTable) DereferenceStringEntryBytes(d types.Dict, key string) ([]byte, error) {
	o, found := d.Find(key)
	if !found || o == nil {
		return nil, nil
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, nil
	}

	switch o := o.(type) {
	case types.StringLiteral:
		bb, err := types.Unescape(o.Value(), false)
		if err != nil {
			return nil, err
		}
		return bb, nil

	case types.HexLiteral:
		return o.Bytes()

	}

	return nil, errors.Errorf("pdfcpu: DereferenceStringEntryBytes dict=%s entry=%s, wrong type %T <%v>", d, key, o, o)
}

func (xRefTable *XRefTable) DestName(obj types.Object) (string, error) {
	dest, err := xRefTable.Dereference(obj)
	if err != nil {
		return "", err
	}

	var s string

	switch d := dest.(type) {
	case types.Name:
		s = d.Value()
	case types.StringLiteral:
		s, err = types.StringLiteralToString(d)
	case types.HexLiteral:
		s, err = types.HexLiteralToString(d)
	}

	return s, err
}

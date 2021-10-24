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

package pdfcpu

import "github.com/pkg/errors"

// indRefToObject dereferences an indirect object from the xRefTable and returns the result.
func (xRefTable *XRefTable) indRefToObject(ir *IndirectRef) (Object, error) {
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
func (xRefTable *XRefTable) Dereference(o Object) (Object, error) {
	ir, ok := o.(IndirectRef)
	if !ok {
		// Nothing do dereference.
		return o, nil
	}

	return xRefTable.indRefToObject(&ir)
}

// DereferenceBoolean resolves and validates a boolean object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceBoolean(o Object, sinceVersion Version) (*Boolean, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	b, ok := o.(Boolean)
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
func (xRefTable *XRefTable) DereferenceInteger(o Object) (*Integer, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	i, ok := o.(Integer)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceInteger: wrong type <%v>", o)
	}

	return &i, nil
}

// DereferenceNumber resolves a number object, which may be an indirect reference and returns a float64.
func (xRefTable *XRefTable) DereferenceNumber(o Object) (float64, error) {

	var (
		f   float64
		err error
	)

	o, _ = xRefTable.Dereference(o)

	switch o := o.(type) {

	case Integer:
		f = float64(o.Value())

	case Float:
		f = o.Value()

	default:
		err = errors.Errorf("pdfcpu: dereferenceNumber: wrong type <%v>", o)

	}

	return f, err
}

// DereferenceName resolves and validates a name object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceName(o Object, sinceVersion Version, validate func(string) bool) (n Name, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return n, err
	}

	n, ok := o.(Name)
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
func (xRefTable *XRefTable) DereferenceStringLiteral(o Object, sinceVersion Version, validate func(string) bool) (s StringLiteral, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return s, err
	}

	s, ok := o.(StringLiteral)
	if !ok {
		return s, errors.Errorf("pdfcpu: dereferenceStringLiteral: wrong type <%v>", o)
	}

	// Ensure UTF16 correctness.
	s1, err := StringLiteralToString(s)
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
func (xRefTable *XRefTable) DereferenceStringOrHexLiteral(obj Object, sinceVersion Version, validate func(string) bool) (s string, err error) {

	o, err := xRefTable.Dereference(obj)
	if err != nil || o == nil {
		return "", err
	}

	switch str := o.(type) {

	case StringLiteral:
		// Ensure UTF16 correctness.
		if s, err = StringLiteralToString(str); err != nil {
			return "", err
		}

	case HexLiteral:
		// Ensure UTF16 correctness.
		if s, err = HexLiteralToString(str); err != nil {
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
func Text(o Object) (string, error) {
	switch obj := o.(type) {
	case StringLiteral:
		return StringLiteralToString(obj)
	case HexLiteral:
		return HexLiteralToString(obj)
	default:
		return "", errors.Errorf("pdfcpu: text: corrupt -  %v\n", obj)
	}
}

// DereferenceText resolves and validates a string or hex literal object to a string.
func (xRefTable *XRefTable) DereferenceText(o Object) (string, error) {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", err
	}
	return Text(o)
}

// DereferenceCSVSafeText resolves and validates a string or hex literal object to a string.
func (xRefTable *XRefTable) DereferenceCSVSafeText(o Object) (string, error) {
	s, err := xRefTable.DereferenceText(o)
	if err != nil {
		return "", err
	}
	return csvSafeString(s), nil
}

// DereferenceArray resolves and validates an array object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceArray(o Object) (Array, error) {

	// TODO Cleanup responsibilities!
	// Fix relict from Destination validation.

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	a, ok := o.(Array)
	if ok {
		return a, nil
	}

	d, ok := o.(Dict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceArray: dest of wrong type <%v>", o)
	}

	return d["D"].(Array), nil
}

// DereferenceDict resolves and validates a dictionary object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceDict(o Object) (Dict, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(Dict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceDict: wrong type %T <%v>", o, o)
	}

	return d, nil
}

// DereferenceDictEntry returns a dereferenced dict entry.
func (xRefTable *XRefTable) DereferenceDictEntry(d Dict, key string) (Object, error) {
	o, found := d.Find(key)
	if !found || o == nil {
		return nil, errors.Errorf("pdfcpu: dict=%s entry=%s missing.", d, key)
	}
	return xRefTable.Dereference(o)
}

// DereferenceStringEntryBytes returns the bytes of a string entry of d.
func (xRefTable *XRefTable) DereferenceStringEntryBytes(d Dict, key string) ([]byte, error) {
	o, found := d.Find(key)
	if !found || o == nil {
		return nil, nil
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, nil
	}

	switch o := o.(type) {
	case StringLiteral:
		bb, err := Unescape(o.Value())
		if err != nil {
			return nil, err
		}
		return bb, nil

	case HexLiteral:
		return o.Bytes()

	}

	return nil, errors.Errorf("pdfcpu: DereferenceStringEntryBytes dict=%s entry=%s, wrong type %T <%v>", d, key, o, o)
}

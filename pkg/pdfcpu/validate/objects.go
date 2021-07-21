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
	"fmt"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

func validateEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (pdf.Object, error) {

	o, found := d.Find(entryName)
	if !found || o == nil {
		if required {
			return nil, errors.Errorf("dict=%s required entry=%s missing (obj#%d).", dictName, entryName, xRefTable.CurObj)
		}
		return nil, nil
	}

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		if required {
			return nil, errors.Errorf("dict=%s required entry=%s missing (obj#%d).", dictName, entryName, xRefTable.CurObj)
		}
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s (obj#%d)", dictName, entryName, xRefTable.CurObj), sinceVersion)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func validateArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateArrayEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("validateArrayEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateArrayEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	a, ok := o.(pdf.Array)
	if !ok {
		return nil, errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid type %T", dictName, entryName, o)
	}

	// Validation
	if validate != nil && !validate(a) {
		return nil, errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateBooleanEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(bool) bool) (*pdf.Boolean, error) {

	log.Validate.Printf("validateBooleanEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("validateBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
		}
		log.Validate.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	b, ok := o.(pdf.Boolean)
	if !ok {
		return nil, errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(b.Value()) {
		return nil, errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid name dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateBooleanEntry end: entry=%s\n", entryName)

	return &b, nil
}

func validateBooleanArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateBooleanArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}
		if o == nil {
			continue
		}

		_, ok := o.(pdf.Boolean)
		if !ok {
			return nil, errors.Errorf("validateBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Validate.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateDateObject(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) (string, error) {
	s, err := xRefTable.DereferenceStringOrHexLiteral(o, sinceVersion, nil)
	if err != nil {
		return "", err
	}
	if s == "" {
		return s, nil
	}

	t, ok := pdf.DateTime(s, xRefTable.ValidationMode == pdf.ValidationRelaxed)
	if !ok {
		return "", errors.Errorf("pdfcpu: validateDateObject: <%s> invalid date", s)
	}

	return pdf.DateString(t), nil
}

func validateDateEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (*time.Time, error) {

	log.Validate.Printf("validateDateEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	s, err := xRefTable.DereferenceStringOrHexLiteral(o, sinceVersion, nil)
	if err != nil {
		return nil, err
	}
	if s == "" {
		if required {
			return nil, errors.Errorf("validateDateEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateDateEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	time, ok := pdf.DateTime(s, xRefTable.ValidationMode == pdf.ValidationRelaxed)
	if !ok {
		return nil, errors.Errorf("pdfcpu: validateDateEntry: <%s> invalid date", s)
	}

	log.Validate.Printf("validateDateEntry end: entry=%s\n", entryName)

	return &time, nil
}

func validateDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Dict) bool) (pdf.Dict, error) {

	log.Validate.Printf("validateDictEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("validateDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateDictEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	d, ok := o.(pdf.Dict)
	if !ok {
		return nil, errors.Errorf("validateDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(d) {
		return nil, errors.Errorf("validateDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateDictEntry end: entry=%s\n", entryName)

	return d, nil
}

func validateFloat(xRefTable *pdf.XRefTable, o pdf.Object, validate func(float64) bool) (*pdf.Float, error) {

	log.Validate.Println("validateFloat begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateFloat: missing object")
	}

	f, ok := o.(pdf.Float)
	if !ok {
		return nil, errors.New("pdfcpu: validateFloat: invalid type")
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		return nil, errors.Errorf("pdfcpu: validateFloat: invalid float: %s\n", f)
	}

	log.Validate.Println("validateFloat end")

	return &f, nil
}

func validateFloatEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(float64) bool) (*pdf.Float, error) {

	log.Validate.Printf("validateFloatEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("pdfcpu: validateFloatEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateFloatEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	f, ok := o.(pdf.Float)
	if !ok {
		return nil, errors.Errorf("pdfcpu: validateFloatEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		return nil, errors.Errorf("pdfcpu: validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateFloatEntry end: entry=%s\n", entryName)

	return &f, nil
}

func validateFunctionEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateFunctionEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	err = validateFunction(xRefTable, o)
	if err != nil {
		return err
	}

	log.Validate.Printf("validateFunctionEntry end: entry=%s\n", entryName)

	return nil
}

func validateFunctionArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateFunctionArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for _, o := range a {
		err = validateFunction(xRefTable, o)
		if err != nil {
			return nil, err
		}
	}

	log.Validate.Printf("validateFunctionArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateFunctionOrArrayOfFunctionsEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateFunctionOrArrayOfFunctionsEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateFunctionOrArrayOfFunctionsEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateFunctionOrArrayOfFunctionsEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	switch o := o.(type) {

	case pdf.Array:

		for _, o := range o {

			if o == nil {
				continue
			}

			err = validateFunction(xRefTable, o)
			if err != nil {
				return err
			}

		}

	default:
		err = validateFunction(xRefTable, o)
		if err != nil {
			return err
		}

	}

	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	log.Validate.Printf("validateFunctionOrArrayOfFunctionsEntry end: entry=%s\n", entryName)

	return nil
}

func validateIndRefEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (*pdf.IndirectRef, error) {

	log.Validate.Printf("validateIndRefEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	ir, ok := o.(pdf.IndirectRef)
	if !ok {
		return nil, errors.Errorf("pdfcpu: validateIndRefEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	log.Validate.Printf("validateIndRefEntry end: entry=%s\n", entryName)

	return &ir, nil
}

func validateIndRefArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {
		_, ok := o.(pdf.IndirectRef)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateIndRefArrayEntry: invalid type at index %d\n", i)
		}
	}

	log.Validate.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)

	return a, nil
}

func validateInteger(xRefTable *pdf.XRefTable, o pdf.Object, validate func(int) bool) (*pdf.Integer, error) {

	log.Validate.Println("validateInteger begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("pdfcpu: validateInteger: missing object")
	}

	i, ok := o.(pdf.Integer)
	if !ok {
		return nil, errors.New("pdfcpu: validateInteger: invalid type")
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		return nil, errors.Errorf("pdfcpu: validateInteger: invalid integer: %s\n", i)
	}

	log.Validate.Println("validateInteger end")

	return &i, nil
}

func validateIntegerEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(int) bool) (*pdf.Integer, error) {

	log.Validate.Printf("validateIntegerEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("pdfcpu: validateIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateIntegerEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	i, ok := o.(pdf.Integer)
	if !ok {
		return nil, errors.Errorf("pdfcpu: validateIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		return nil, errors.Errorf("pdfcpu: validateIntegerEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateIntegerEntry end: entry=%s\n", entryName)

	return &i, nil
}

func validateIntegerArray(xRefTable *pdf.XRefTable, o pdf.Object) (pdf.Array, error) {

	log.Validate.Println("validateIntegerArray begin")

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		switch o.(type) {

		case pdf.Integer:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateIntegerArray: invalid type at index %d\n", i)
		}

	}

	log.Validate.Println("validateIntegerArray end")

	return a, nil
}

func validateIntegerArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateIntegerArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		_, ok := o.(pdf.Integer)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Validate.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateName(xRefTable *pdf.XRefTable, o pdf.Object, validate func(string) bool) (*pdf.Name, error) {

	log.Validate.Println("validateName begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateName: missing object")
	}

	name, ok := o.(pdf.Name)
	if !ok {
		return nil, errors.New("pdfcpu: validateName: invalid type")
	}

	// Validation
	if validate != nil && !validate(name.Value()) {
		return nil, errors.Errorf("pdfcpu: validateName: invalid name: %s\n", name)
	}

	log.Validate.Println("validateName end")

	return &name, nil
}

func validateNameEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(string) bool) (*pdf.Name, error) {

	log.Validate.Printf("validateNameEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("pdfcpu: validateNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateNameEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	name, ok := o.(pdf.Name)
	if !ok {
		return nil, errors.Errorf("pdfcpu: validateNameEntry: dict=%s entry=%s invalid type %T", dictName, entryName, o)
	}

	// Validation
	v := name.Value()
	if validate != nil && (required || len(v) > 0) && !validate(v) {
		return nil, errors.Errorf("pdfcpu: validateNameEntry: dict=%s entry=%s invalid dict entry: %s", dictName, entryName, name.Value())
	}

	log.Validate.Printf("validateNameEntry end: entry=%s\n", entryName)

	return &name, nil
}

func validateNameArray(xRefTable *pdf.XRefTable, o pdf.Object) (pdf.Array, error) {

	log.Validate.Println("validateNameArray begin")

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		_, ok := o.(pdf.Name)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateNameArray: invalid type at index %d\n", i)
		}

	}

	log.Validate.Println("validateNameArray end")

	return a, nil
}

func validateNameArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(a pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateNameArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		_, ok := o.(pdf.Name)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Validate.Printf("validateNameArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateNumber(xRefTable *pdf.XRefTable, o pdf.Object) (pdf.Object, error) {

	log.Validate.Println("validateNumber begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateNumber: missing object")
	}

	switch o.(type) {

	case pdf.Integer:
		// no further processing.

	case pdf.Float:
		// no further processing.

	default:
		return nil, errors.New("pdfcpu: validateNumber: invalid type")

	}

	log.Validate.Println("validateNumber end ")

	return o, nil
}

func validateNumberEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(f float64) bool) (pdf.Object, error) {

	log.Validate.Printf("validateNumberEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	o, err = validateNumber(xRefTable, o)
	if err != nil {
		return nil, err
	}

	var f float64

	// Validation
	switch o := o.(type) {

	case pdf.Integer:
		f = float64(o.Value())

	case pdf.Float:
		f = o.Value()
	}

	if validate != nil && !validate(f) {
		return nil, errors.Errorf("pdfcpu: validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateNumberEntry end: entry=%s\n", entryName)

	return o, nil
}

func validateNumberArray(xRefTable *pdf.XRefTable, o pdf.Object) (pdf.Array, error) {

	log.Validate.Println("validateNumberArray begin")

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		switch o.(type) {

		case pdf.Integer:
			// no further processing.

		case pdf.Float:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateNumberArray: invalid type at index %d\n", i)
		}

	}

	log.Validate.Println("validateNumberArray end")

	return a, err
}

func validateNumberArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateNumberArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		switch o.(type) {

		case pdf.Integer:
			// no further processing.

		case pdf.Float:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateNumberArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Validate.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateRectangleEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateRectangleEntry begin: entry=%s\n", entryName)

	a, err := validateNumberArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, func(a pdf.Array) bool { return len(a) == 4 })
	if err != nil || a == nil {
		return nil, err
	}

	if validate != nil && !validate(a) {
		return nil, errors.Errorf("pdfcpu: validateRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
	}

	log.Validate.Printf("validateRectangleEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateStreamDict(xRefTable *pdf.XRefTable, o pdf.Object) (*pdf.StreamDict, error) {

	log.Validate.Println("validateStreamDict begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateStreamDict: missing object")
	}

	sd, ok := o.(pdf.StreamDict)
	if !ok {
		return nil, errors.New("pdfcpu: validateStreamDict: invalid type")
	}

	log.Validate.Println("validateStreamDict endobj")

	return &sd, nil
}

func validateStreamDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.StreamDict) bool) (*pdf.StreamDict, error) {

	log.Validate.Printf("validateStreamDictEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	sd, valid, err := xRefTable.DereferenceStreamDict(o)
	if valid {
		return nil, nil
	}
	if err != nil || sd == nil {
		return nil, err
	}

	// o, err = xRefTable.Dereference(o)
	// if err != nil {
	// 	return nil, err
	// }
	if sd == nil {
		if required {
			return nil, errors.Errorf("pdfcpu: validateStreamDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateStreamDictEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	// sd, ok := o.(pdf.StreamDict)
	// if !ok {
	// 	return nil, errors.Errorf("pdfcpu: validateStreamDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	// }

	// Validation
	if validate != nil && !validate(*sd) {
		return nil, errors.Errorf("pdfcpu: validateStreamDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateStreamDictEntry end: entry=%s\n", entryName)

	return sd, nil
}

func validateString(xRefTable *pdf.XRefTable, o pdf.Object, validate func(string) bool) (string, error) {

	//log.Validate.Println("validateString begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", err
	}
	if o == nil {
		return "", errors.New("pdfcpu: validateString: missing object")
	}

	var s string

	switch o := o.(type) {

	case pdf.StringLiteral:
		s, err = pdf.StringLiteralToString(o)

	case pdf.HexLiteral:
		s, err = pdf.HexLiteralToString(o)

	default:
		err = errors.New("pdfcpu: validateString: invalid type")
	}

	if err != nil {
		return s, err
	}

	// Validation
	if validate != nil && !validate(s) {
		return "", errors.Errorf("pdfcpu: validateString: %s invalid", s)
	}

	//log.Validate.Println("validateString end")

	return s, nil
}

func validateStringEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(string) bool) (*string, error) {

	log.Validate.Printf("validateStringEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		if required {
			return nil, errors.Errorf("pdfcpu: validateStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateStringEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	var s string

	switch o := o.(type) {

	case pdf.StringLiteral:
		s, err = pdf.StringLiteralToString(o)

	case pdf.HexLiteral:
		s, err = pdf.HexLiteralToString(o)

	default:
		err = errors.Errorf("pdfcpu: validateStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	if err != nil {
		return nil, err
	}

	// Validation
	if validate != nil && (required || len(s) > 0) && !validate(s) {
		return nil, errors.Errorf("pdfcpu: validateStringEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateStringEntry end: entry=%s\n", entryName)

	return &s, nil
}

func validateStringArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateStringArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		switch o.(type) {

		case pdf.StringLiteral:
			// no further processing.

		case pdf.HexLiteral:
			// no further processing

		default:
			return nil, errors.Errorf("pdfcpu: validateStringArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Validate.Printf("validateStringArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateArrayArrayEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (pdf.Array, error) {

	log.Validate.Printf("validateArrayArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, err
		}

		if o == nil {
			continue
		}

		switch o.(type) {

		case pdf.Array:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateArrayArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Validate.Printf("validateArrayArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateStringOrStreamEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateStringOrStreamEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateStringOrStreamEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateStringOrStreamEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o.(type) {

	case pdf.StringLiteral, pdf.HexLiteral, pdf.StreamDict:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateStringOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateStringOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateNameOrStringEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateNameOrStringEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateNameOrStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateNameOrStringEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o.(type) {

	case pdf.StringLiteral, pdf.Name:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateNameOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateNameOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrStringEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateIntOrStringEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateIntOrStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateIntOrStringEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o.(type) {

	case pdf.StringLiteral, pdf.HexLiteral, pdf.Integer:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateIntOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateIntOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateIntOrDictEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateIntOrDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateIntOrDictEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o.(type) {

	case pdf.Integer, pdf.Dict:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateIntOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateIntOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateBooleanOrStreamEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateBooleanOrStreamEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateBooleanOrStreamEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateBooleanOrStreamEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o.(type) {

	case pdf.Boolean, pdf.StreamDict:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateBooleanOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateBooleanOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateStreamDictOrDictEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateStreamDictOrDictEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateStreamDictOrDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateStreamDictOrDictEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o.(type) {

	case pdf.StreamDict:
		// TODO validate 3D stream dict

	case pdf.Dict:
		// TODO validate 3D reference dict

	default:
		return errors.Errorf("pdfcpu: validateStreamDictOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateStreamDictOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntegerOrArrayOfIntegerEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateIntegerOrArrayOfIntegerEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateIntegerOrArrayOfIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateIntegerOrArrayOfIntegerEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Integer:
		// no further processing

	case pdf.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			_, ok := o.(pdf.Integer)
			if !ok {
				return errors.Errorf("pdfcpu: validateIntegerOrArrayOfIntegerEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			}

		}

	default:
		return errors.Errorf("pdfcpu: validateIntegerOrArrayOfIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateIntegerOrArrayOfIntegerEntry end: entry=%s\n", entryName)

	return nil
}

func validateNameOrArrayOfNameEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateNameOrArrayOfNameEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateNameOrArrayOfNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateNameOrArrayOfNameEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Name:
		// no further processing

	case pdf.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			_, ok := o.(pdf.Name)
			if !ok {
				err = errors.Errorf("pdfcpu: validateNameOrArrayOfNameEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
				return err
			}

		}

	default:
		return errors.Errorf("pdfcpu: validateNameOrArrayOfNameEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateNameOrArrayOfNameEntry end: entry=%s\n", entryName)

	return nil
}

func validateBooleanOrArrayOfBooleanEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Validate.Printf("validateBooleanOrArrayOfBooleanEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return err
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return err
	}
	if o == nil {
		if required {
			return errors.Errorf("pdfcpu: validateBooleanOrArrayOfBooleanEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Validate.Printf("validateBooleanOrArrayOfBooleanEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch o := o.(type) {

	case pdf.Boolean:
		// no further processing

	case pdf.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			_, ok := o.(pdf.Boolean)
			if !ok {
				return errors.Errorf("pdfcpu: validateBooleanOrArrayOfBooleanEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			}

		}

	default:
		return errors.Errorf("pdfcpu: validateBooleanOrArrayOfBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateBooleanOrArrayOfBooleanEntry end: entry=%s\n", entryName)

	return nil
}

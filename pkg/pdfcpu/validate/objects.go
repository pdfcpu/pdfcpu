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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

func validateEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) (types.Object, error) {

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

func validateArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

	a, ok := o.(types.Array)
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

func validateBooleanEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(bool) bool) (*types.Boolean, error) {

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

	b, ok := o.(types.Boolean)
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

func validateBooleanArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

		_, ok := o.(types.Boolean)
		if !ok {
			return nil, errors.Errorf("validateBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Validate.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateDateObject(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) (string, error) {
	s, err := xRefTable.DereferenceStringOrHexLiteral(o, sinceVersion, nil)
	if err != nil {
		return "", err
	}
	if s == "" {
		return s, nil
	}

	t, ok := types.DateTime(s, xRefTable.ValidationMode == model.ValidationRelaxed)
	if !ok {
		return "", errors.Errorf("pdfcpu: validateDateObject: <%s> invalid date", s)
	}

	return types.DateString(t), nil
}

func validateDateEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) (*time.Time, error) {

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

	time, ok := types.DateTime(s, xRefTable.ValidationMode == model.ValidationRelaxed)
	if !ok {
		return nil, errors.Errorf("pdfcpu: validateDateEntry: <%s> invalid date", s)
	}

	log.Validate.Printf("validateDateEntry end: entry=%s\n", entryName)

	return &time, nil
}

func validateDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Dict) bool) (types.Dict, error) {

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

	d, ok := o.(types.Dict)
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

func validateFloat(xRefTable *model.XRefTable, o types.Object, validate func(float64) bool) (*types.Float, error) {

	log.Validate.Println("validateFloat begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateFloat: missing object")
	}

	f, ok := o.(types.Float)
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

func validateFunctionArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

func validateFunctionOrArrayOfFunctionsEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.Array:

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

func validateIndRefEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) (*types.IndirectRef, error) {

	log.Validate.Printf("validateIndRefEntry begin: entry=%s\n", entryName)

	o, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, err
	}

	ir, ok := o.(types.IndirectRef)
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

func validateIndRefArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

	log.Validate.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)

	a, err := validateArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}

	for i, o := range a {
		_, ok := o.(types.IndirectRef)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateIndRefArrayEntry: invalid type at index %d\n", i)
		}
	}

	log.Validate.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)

	return a, nil
}

func validateInteger(xRefTable *model.XRefTable, o types.Object, validate func(int) bool) (*types.Integer, error) {

	log.Validate.Println("validateInteger begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("pdfcpu: validateInteger: missing object")
	}

	i, ok := o.(types.Integer)
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

func validateIntegerEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(int) bool) (*types.Integer, error) {

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

	i, ok := o.(types.Integer)
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

func validateIntegerArray(xRefTable *model.XRefTable, o types.Object) (types.Array, error) {

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

		case types.Integer:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateIntegerArray: invalid type at index %d\n", i)
		}

	}

	log.Validate.Println("validateIntegerArray end")

	return a, nil
}

func validateIntegerArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

		_, ok := o.(types.Integer)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Validate.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateName(xRefTable *model.XRefTable, o types.Object, validate func(string) bool) (*types.Name, error) {

	log.Validate.Println("validateName begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateName: missing object")
	}

	name, ok := o.(types.Name)
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

func validateNameEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(string) bool) (*types.Name, error) {

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

	name, ok := o.(types.Name)
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

func validateNameArray(xRefTable *model.XRefTable, o types.Object) (types.Array, error) {

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

		_, ok := o.(types.Name)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateNameArray: invalid type at index %d\n", i)
		}

	}

	log.Validate.Println("validateNameArray end")

	return a, nil
}

func validateNameArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(a types.Array) bool) (types.Array, error) {

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

		_, ok := o.(types.Name)
		if !ok {
			return nil, errors.Errorf("pdfcpu: validateNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Validate.Printf("validateNameArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateNumber(xRefTable *model.XRefTable, o types.Object) (types.Object, error) {

	log.Validate.Println("validateNumber begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateNumber: missing object")
	}

	switch o.(type) {

	case types.Integer:
		// no further processing.

	case types.Float:
		// no further processing.

	default:
		return nil, errors.New("pdfcpu: validateNumber: invalid type")

	}

	log.Validate.Println("validateNumber end ")

	return o, nil
}

func validateNumberEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(f float64) bool) (types.Object, error) {

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

	case types.Integer:
		f = float64(o.Value())

	case types.Float:
		f = o.Value()
	}

	if validate != nil && !validate(f) {
		return nil, errors.Errorf("pdfcpu: validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Validate.Printf("validateNumberEntry end: entry=%s\n", entryName)

	return o, nil
}

func validateNumberArray(xRefTable *model.XRefTable, o types.Object) (types.Array, error) {

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

		case types.Integer:
			// no further processing.

		case types.Float:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateNumberArray: invalid type at index %d\n", i)
		}

	}

	log.Validate.Println("validateNumberArray end")

	return a, err
}

func validateNumberArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

		case types.Integer:
			// no further processing.

		case types.Float:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateNumberArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Validate.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateRectangleEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

	log.Validate.Printf("validateRectangleEntry begin: entry=%s\n", entryName)

	a, err := validateNumberArrayEntry(xRefTable, d, dictName, entryName, required, sinceVersion, func(a types.Array) bool { return len(a) == 4 })
	if err != nil || a == nil {
		return nil, err
	}

	if validate != nil && !validate(a) {
		return nil, errors.Errorf("pdfcpu: validateRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
	}

	log.Validate.Printf("validateRectangleEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateStreamDict(xRefTable *model.XRefTable, o types.Object) (*types.StreamDict, error) {

	log.Validate.Println("validateStreamDict begin")

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("pdfcpu: validateStreamDict: missing object")
	}

	sd, ok := o.(types.StreamDict)
	if !ok {
		return nil, errors.New("pdfcpu: validateStreamDict: invalid type")
	}

	log.Validate.Println("validateStreamDict endobj")

	return &sd, nil
}

func validateStreamDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.StreamDict) bool) (*types.StreamDict, error) {

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

func validateStringEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(string) bool) (*string, error) {

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

	case types.StringLiteral:
		s, err = types.StringLiteralToString(o)

	case types.HexLiteral:
		s, err = types.HexLiteralToString(o)

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

func validateStringArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

		case types.StringLiteral:
			// no further processing.

		case types.HexLiteral:
			// no further processing

		default:
			return nil, errors.Errorf("pdfcpu: validateStringArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Validate.Printf("validateStringArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateArrayArrayEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {

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

		case types.Array:
			// no further processing.

		default:
			return nil, errors.Errorf("pdfcpu: validateArrayArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Validate.Printf("validateArrayArrayEntry end: entry=%s\n", entryName)

	return a, nil
}

func validateStringOrStreamEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.StringLiteral, types.HexLiteral, types.StreamDict:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateStringOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateStringOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateNameOrStringEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.StringLiteral, types.Name:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateNameOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateNameOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrStringEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.StringLiteral, types.HexLiteral, types.Integer:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateIntOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateIntOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateBooleanOrStreamEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.Boolean, types.StreamDict:
		// no further processing

	default:
		return errors.Errorf("pdfcpu: validateBooleanOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateBooleanOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateStreamDictOrDictEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.StreamDict:
		// TODO validate 3D stream dict

	case types.Dict:
		// TODO validate 3D reference dict

	default:
		return errors.Errorf("pdfcpu: validateStreamDictOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Validate.Printf("validateStreamDictOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntegerOrArrayOfIntegerEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.Integer:
		// no further processing

	case types.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			_, ok := o.(types.Integer)
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

func validateNameOrArrayOfNameEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.Name:
		// no further processing

	case types.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			_, ok := o.(types.Name)
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

func validateBooleanOrArrayOfBooleanEntry(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {

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

	case types.Boolean:
		// no further processing

	case types.Array:

		for i, o := range o {

			o, err := xRefTable.Dereference(o)
			if err != nil {
				return err
			}

			if o == nil {
				continue
			}

			_, ok := o.(types.Boolean)
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

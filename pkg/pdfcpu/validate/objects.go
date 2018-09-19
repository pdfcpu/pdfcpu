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

	"github.com/hhrutter/pdfcpu/pkg/log"
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

func validateEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (pdf.Object, error) {

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return nil, errors.Errorf("dict=%s required entry=%s missing.", dictName, entryName)
		}
		return nil, nil
	}

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		if required {
			return nil, errors.Errorf("dict=%s required entry=%s missing.", dictName, entryName)
		}
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func validateArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateArrayEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateArrayEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateArrayEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	arr, ok := obj.(pdf.Array)
	if !ok {
		return nil, errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(arr) {
		return nil, errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateArrayEntry end: entry=%s\n", entryName)

	return &arr, nil
}

func validateBooleanEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(bool) bool) (*pdf.Boolean, error) {

	log.Debug.Printf("validateBooleanEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
		}
		log.Debug.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	b, ok := obj.(pdf.Boolean)
	if !ok {
		return nil, errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(b.Value()) {
		return nil, errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid name dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateBooleanEntry end: entry=%s\n", entryName)

	return &b, nil
}

func validateBooleanArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateBooleanArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err := xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}
		if obj == nil {
			continue
		}

		_, ok := obj.(pdf.Boolean)
		if !ok {
			return nil, errors.Errorf("validateBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Debug.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateDateObject(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) (pdf.StringLiteral, error) {
	return xRefTable.DereferenceStringLiteral(obj, sinceVersion, validateDate)
}

func validateDateEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (*pdf.StringLiteral, error) {

	log.Debug.Printf("validateDateEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateDateEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateDateEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	date, ok := obj.(pdf.StringLiteral)
	if !ok {
		return nil, errors.Errorf("validateDateEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if ok := validateDate(date.Value()); !ok {
		return nil, errors.Errorf("validateDateEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateDateEntry end: entry=%s\n", entryName)

	return &date, nil
}

func validateDictEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Dict) bool) (*pdf.Dict, error) {

	log.Debug.Printf("validateDictEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateDictEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	d, ok := obj.(pdf.Dict)
	if !ok {
		return nil, errors.Errorf("validateDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(d) {
		return nil, errors.Errorf("validateDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateDictEntry end: entry=%s\n", entryName)

	return &d, nil
}

func validateFloat(xRefTable *pdf.XRefTable, obj pdf.Object, validate func(float64) bool) (*pdf.Float, error) {

	log.Debug.Println("validateFloat begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateFloat: missing object")
	}

	f, ok := obj.(pdf.Float)
	if !ok {
		return nil, errors.New("validateFloat: invalid type")
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		return nil, errors.Errorf("validateFloat: invalid float: %s\n", f)
	}

	log.Debug.Println("validateFloat end")

	return &f, nil
}

func validateFloatEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(float64) bool) (*pdf.Float, error) {

	log.Debug.Printf("validateFloatEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateFloatEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateFloatEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	f, ok := obj.(pdf.Float)
	if !ok {
		return nil, errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		return nil, errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateFloatEntry end: entry=%s\n", entryName)

	return &f, nil
}

func validateFunctionEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateFunctionEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	err = validateFunction(xRefTable, obj)
	if err != nil {
		return err
	}

	log.Debug.Printf("validateFunctionEntry end: entry=%s\n", entryName)

	return nil
}

func validateFunctionArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateFunctionArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for _, obj := range *arr {
		err = validateFunction(xRefTable, obj)
		if err != nil {
			return nil, err
		}
	}

	log.Debug.Printf("validateFunctionArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateFunctionOrArrayOfFunctionsEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateFunctionOrArrayOfFunctionsEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateFunctionOrArrayOfFunctionsEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateFunctionOrArrayOfFunctionsEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	switch obj := obj.(type) {

	case pdf.Array:

		for _, obj := range obj {

			if obj == nil {
				continue
			}

			err = validateFunction(xRefTable, obj)
			if err != nil {
				return err
			}

		}

	default:
		err = validateFunction(xRefTable, obj)
		if err != nil {
			return err
		}

	}

	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	log.Debug.Printf("validateFunctionOrArrayOfFunctionsEntry end: entry=%s\n", entryName)

	return nil
}

func validateIndRefEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) (*pdf.IndirectRef, error) {

	log.Debug.Printf("validateIndRefEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	indRef, ok := obj.(pdf.IndirectRef)
	if !ok {
		return nil, errors.Errorf("validateIndRefEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	log.Debug.Printf("validateIndRefEntry end: entry=%s\n", entryName)

	return &indRef, nil
}

func validateIndRefArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {
		_, ok := obj.(pdf.IndirectRef)
		if !ok {
			return nil, errors.Errorf("validateIndRefArrayEntry: invalid type at index %d\n", i)
		}
	}

	log.Debug.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)

	return arr, nil
}

func validateInteger(xRefTable *pdf.XRefTable, obj pdf.Object, validate func(int) bool) (*pdf.Integer, error) {

	log.Debug.Println("validateInteger begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, errors.New("validateInteger: missing object")
	}

	i, ok := obj.(pdf.Integer)
	if !ok {
		return nil, errors.New("validateInteger: invalid type")
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		return nil, errors.Errorf("validateInteger: invalid integer: %s\n", i)
	}

	log.Debug.Println("validateInteger end")

	return &i, nil
}

func validateIntegerEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(int) bool) (*pdf.Integer, error) {

	log.Debug.Printf("validateIntegerEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateIntegerEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	i, ok := obj.(pdf.Integer)
	if !ok {
		return nil, errors.Errorf("validateIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		return nil, errors.Errorf("validateIntegerEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateIntegerEntry end: entry=%s\n", entryName)

	return &i, nil
}

func validateIntegerArray(xRefTable *pdf.XRefTable, obj pdf.Object) (*pdf.Array, error) {

	log.Debug.Println("validateIntegerArray begin")

	a, err := xRefTable.DereferenceArray(obj)
	if err != nil || a == nil {
		return nil, err
	}

	for i, obj := range *a {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case pdf.Integer:
			// no further processing.

		default:
			return nil, errors.Errorf("validateIntegerArray: invalid type at index %d\n", i)
		}

	}

	log.Debug.Println("validateIntegerArray end")

	return a, nil
}

func validateIntegerArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateIntegerArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(pdf.Integer)
		if !ok {
			return nil, errors.Errorf("validateIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Debug.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateName(xRefTable *pdf.XRefTable, obj pdf.Object, validate func(string) bool) (*pdf.Name, error) {

	log.Debug.Println("validateName begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateName: missing object")
	}

	name, ok := obj.(pdf.Name)
	if !ok {
		return nil, errors.New("validateName: invalid type")
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		return nil, errors.Errorf("validateName: invalid name: %s\n", name)
	}

	log.Debug.Println("validateName end")

	return &name, nil
}

func validateNameEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(string) bool) (*pdf.Name, error) {

	log.Debug.Printf("validateNameEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateNameEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	name, ok := obj.(pdf.Name)
	if !ok {
		return nil, errors.Errorf("validateNameEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		return nil, errors.Errorf("validateNameEntry: dict=%s entry=%s invalid dict entry: %s", dictName, entryName, name.String())
	}

	log.Debug.Printf("validateNameEntry end: entry=%s\n", entryName)

	return &name, nil
}

func validateNameArray(xRefTable *pdf.XRefTable, obj pdf.Object) (*pdf.Array, error) {

	log.Debug.Println("validateNameArray begin")

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(pdf.Name)
		if !ok {
			return nil, errors.Errorf("validateNameArray: invalid type at index %d\n", i)
		}

	}

	log.Debug.Println("validateNameArray end")

	return arr, nil
}

func validateNameArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(a pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateNameArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(pdf.Name)
		if !ok {
			return nil, errors.Errorf("validateNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Debug.Printf("validateNameArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateNumber(xRefTable *pdf.XRefTable, obj pdf.Object) (pdf.Object, error) {

	log.Debug.Println("validateNumber begin")

	n, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errors.New("validateNumber: missing object")
	}

	switch n.(type) {

	case pdf.Integer:
		// no further processing.

	case pdf.Float:
		// no further processing.

	default:
		return nil, errors.New("validateNumber: invalid type")

	}

	log.Debug.Println("validateNumber end ")

	return n, nil
}

func validateNumberEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(f float64) bool) (pdf.Object, error) {

	log.Debug.Printf("validateNumberEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	obj, err = validateNumber(xRefTable, obj)
	if err != nil {
		return nil, err
	}

	var f float64

	// Validation
	switch o := obj.(type) {

	case pdf.Integer:
		f = float64(o.Value())

	case pdf.Float:
		f = o.Value()
	}

	if validate != nil && !validate(f) {
		return nil, errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateNumberEntry end: entry=%s\n", entryName)

	return obj, nil
}

func validateNumberArray(xRefTable *pdf.XRefTable, obj pdf.Object) (*pdf.Array, error) {

	log.Debug.Println("validateNumberArray begin")

	arrp, err := xRefTable.DereferenceArray(obj)
	if err != nil || arrp == nil {
		return nil, err
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case pdf.Integer:
			// no further processing.

		case pdf.Float:
			// no further processing.

		default:
			return nil, errors.Errorf("validateNumberArray: invalid type at index %d\n", i)
		}

	}

	log.Debug.Println("validateNumberArray end")

	return nil, err
}

func validateNumberArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateNumberArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case pdf.Integer:
			// no further processing.

		case pdf.Float:
			// no further processing.

		default:
			return nil, errors.Errorf("validateNumberArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Debug.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateRectangleEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateRectangleEntry begin: entry=%s\n", entryName)

	arr, err := validateNumberArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, func(arr pdf.Array) bool { return len(arr) == 4 })
	if err != nil || arr == nil {
		return nil, err
	}

	if validate != nil && !validate(*arr) {
		return nil, errors.Errorf("validateRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
	}

	log.Debug.Printf("validateRectangleEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateStreamDict(xRefTable *pdf.XRefTable, obj pdf.Object) (*pdf.StreamDict, error) {

	log.Debug.Println("validateStreamDict begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateStreamDict: missing object")
	}

	sd, ok := obj.(pdf.StreamDict)
	if !ok {
		return nil, errors.New("validateStreamDict: invalid type")
	}

	log.Debug.Println("validateStreamDict endobj")

	return &sd, nil
}

func validateStreamDictEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.StreamDict) bool) (*pdf.StreamDict, error) {

	log.Debug.Printf("validateStreamDictEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateStreamDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateStreamDictEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	sd, ok := obj.(pdf.StreamDict)
	if !ok {
		return nil, errors.Errorf("validateStreamDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(sd) {
		return nil, errors.Errorf("validateStreamDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateStreamDictEntry end: entry=%s\n", entryName)

	return &sd, nil
}

func validateString(xRefTable *pdf.XRefTable, obj pdf.Object, validate func(string) bool) (*string, error) {

	//log.Debug.Println("validateString begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateString: missing object")
	}

	var s string

	switch obj := obj.(type) {

	case pdf.StringLiteral:
		s = obj.Value()

	case pdf.HexLiteral:
		s = obj.Value()

	default:
		return nil, errors.New("validateString: invalid type")
	}

	// Validation
	if validate != nil && !validate(s) {
		return nil, errors.Errorf("validateString: %s invalid", s)
	}

	//log.Debug.Println("validateString end")

	return &s, nil
}

func validateStringEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(string) bool) (*string, error) {

	log.Debug.Printf("validateStringEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		if required {
			return nil, errors.Errorf("validateStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateStringEntry end: optional entry %s is nil\n", entryName)
		return nil, nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return nil, err
	}

	var s string

	switch obj := obj.(type) {

	case pdf.StringLiteral:
		s = obj.Value()

	case pdf.HexLiteral:
		s = obj.Value()

	default:
		return nil, errors.Errorf("validateStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	// Validation
	if validate != nil && !validate(s) {
		return nil, errors.Errorf("validateStringEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateStringEntry end: entry=%s\n", entryName)

	return &s, nil
}

func validateStringArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateStringArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case pdf.StringLiteral:
			// no further processing.

		case pdf.HexLiteral:
			// no further processing

		default:
			return nil, errors.Errorf("validateStringArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Debug.Printf("validateStringArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateArrayArrayEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version, validate func(pdf.Array) bool) (*pdf.Array, error) {

	log.Debug.Printf("validateArrayArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return nil, err
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case pdf.Array:
			// no further processing.

		default:
			return nil, errors.Errorf("validateArrayArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Debug.Printf("validateArrayArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateStringOrStreamEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateStringOrStreamEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateStringOrStreamEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateStringOrStreamEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj.(type) {

	case pdf.StringLiteral, pdf.HexLiteral, pdf.StreamDict:
		// no further processing

	default:
		return errors.Errorf("validateStringOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateStringOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateNameOrStringEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateNameOrStringEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateNameOrStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateNameOrStringEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj.(type) {

	case pdf.StringLiteral, pdf.Name:
		// no further processing

	default:
		return errors.Errorf("validateNameOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateNameOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrStringEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateIntOrStringEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateIntOrStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateIntOrStringEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj.(type) {

	case pdf.StringLiteral, pdf.HexLiteral, pdf.Integer:
		// no further processing

	default:
		return errors.Errorf("validateIntOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateIntOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrDictEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateIntOrDictEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateIntOrDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateIntOrDictEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj.(type) {

	case pdf.Integer, pdf.Dict:
		// no further processing

	default:
		return errors.Errorf("validateIntOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateIntOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateBooleanOrStreamEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateBooleanOrStreamEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateBooleanOrStreamEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateBooleanOrStreamEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj.(type) {

	case pdf.Boolean, pdf.StreamDict:
		// no further processing

	default:
		return errors.Errorf("validateBooleanOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateBooleanOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateStreamDictOrDictEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateStreamDictOrDictEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateStreamDictOrDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateStreamDictOrDictEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj.(type) {

	case pdf.StreamDict:
		// TODO validate 3D stream dict

	case pdf.Dict:
		// TODO validate 3D reference dict

	default:
		return errors.Errorf("validateStreamDictOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateStreamDictOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntegerOrArrayOfIntegerEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateIntegerOrArrayOfIntegerEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateIntegerOrArrayOfIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateIntegerOrArrayOfIntegerEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Integer:
		// no further processing

	case pdf.Array:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(pdf.Integer)
			if !ok {
				return errors.Errorf("validateIntegerOrArrayOfIntegerEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			}

		}

	default:
		return errors.Errorf("validateIntegerOrArrayOfIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateIntegerOrArrayOfIntegerEntry end: entry=%s\n", entryName)

	return nil
}

func validateNameOrArrayOfNameEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateNameOrArrayOfNameEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateNameOrArrayOfNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateNameOrArrayOfNameEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Name:
		// no further processing

	case pdf.Array:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(pdf.Name)
			if !ok {
				err = errors.Errorf("validateNameOrArrayOfNameEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
				return err
			}

		}

	default:
		return errors.Errorf("validateNameOrArrayOfNameEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateNameOrArrayOfNameEntry end: entry=%s\n", entryName)

	return nil
}

func validateBooleanOrArrayOfBooleanEntry(xRefTable *pdf.XRefTable, dict *pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	log.Debug.Printf("validateBooleanOrArrayOfBooleanEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateBooleanOrArrayOfBooleanEntry: dict=%s required entry=%s is nil", dictName, entryName)
		}
		log.Debug.Printf("validateBooleanOrArrayOfBooleanEntry end: optional entry %s is nil\n", entryName)
		return nil
	}

	// Version check
	err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion)
	if err != nil {
		return err
	}

	switch obj := obj.(type) {

	case pdf.Boolean:
		// no further processing

	case pdf.Array:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(pdf.Boolean)
			if !ok {
				return errors.Errorf("validateBooleanOrArrayOfBooleanEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			}

		}

	default:
		return errors.Errorf("validateBooleanOrArrayOfBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateBooleanOrArrayOfBooleanEntry end: entry=%s\n", entryName)

	return nil
}

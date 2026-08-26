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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

func validateEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (types.Object, error) {
	o, found := d.Find(entryName)
	if !found || o == nil {
		if required {
			err := missingRequiredEntryError(dictName, entryName, "")
			return nil, model.WithValidationErrorObject(err, ownerObjNr)
		}
		return nil, nil
	}

	objNr := validationObjectNumber(ownerObjNr, o)
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := missingRequiredEntryError(dictName, entryName, "")
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	return o, nil
}

func missingRequiredEntryError(dictName, entryName, hint string) error {
	msg := fmt.Sprintf("dict=%s required entry=%s missing", dictName, entryName)
	if hint != "" {
		msg += "; repair: " + hint
	}
	return errors.New(msg)
}

func showDigestedVersionViolation(xRefTable *model.XRefTable, element string) {
	model.ShowDigestedSpecViolation(fmt.Sprintf("%s: unsupported in version %s", element, xRefTable.VersionString()))
}

func logMissingRequiredEntry(dictName, entryName string, d types.Dict) {
	if log.ValidateEnabled() {
		log.Validate.Printf("dict=%s required entry=%s missing: %s\n", dictName, entryName, d)
	}
}

func validateArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateArrayEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateArrayEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	a, ok := o.(types.Array)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid type %T", dictName, entryName, o)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	if validate != nil && !validate(a) {
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validationObjectNumber(ownerObjNr int, o types.Object) int {
	if ir, ok := o.(types.IndirectRef); ok {
		return ir.ObjectNumber.Value()
	}
	return ownerObjNr
}

func validationEntryObjectNumber(ownerObjNr int, d types.Dict, entryName string) int {
	o, _ := d.Find(entryName)
	return validationObjectNumber(ownerObjNr, o)
}

func validationRootObjectNumber(xRefTable *model.XRefTable) int {
	if xRefTable.Root == nil {
		return 0
	}
	return xRefTable.Root.ObjectNumber.Value()
}

func validateBooleanEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(bool) bool) (*bool, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s missing", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	b, ok := o.(types.Boolean)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	if validate != nil && !validate(b.Value()) {
		err := fmt.Errorf("dict=%s entry=%s invalid name dict entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanEntry end: entry=%s\n", entryName)
	}

	flag := b.Value()
	return &flag, nil
}

func validateFlexBooleanEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (*bool, error) {
	flag, err := validateBooleanEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, nil)
	if err == nil {
		return flag, nil
	}
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		return nil, err
	}
	n, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion,
		func(s string) bool {
			return types.MemberOf(strings.ToLower(s), []string{"false", "true"})
		},
	)
	if err != nil || n == nil {
		return nil, err
	}

	b := strings.ToLower(n.Value()) == "true"
	flag = &b

	return flag, nil
}

func validateBooleanArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanArrayEntry begin: entry=%s\n", entryName)
	}

	entryObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	a, err := validateArrayEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate,
	)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, entryObjNr)
	}

	for i, raw := range a {
		objNr := validationObjectNumber(entryObjNr, raw)
		o, err := xRefTable.Dereference(raw)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if o == nil {
			continue
		}

		if _, ok := o.(types.Boolean); !ok {
			err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func timeOfDateObject(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, sinceVersion model.Version) (t *time.Time, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	s, err := xRefTable.DereferenceStringOrHexLiteral(o, sinceVersion, nil)
	if err != nil {
		return nil, err
	}

	if s == "" {
		return nil, nil
	}

	t1, ok := types.DateTime(s, xRefTable.ValidationMode == model.ValidationRelaxed)
	if !ok {
		return nil, fmt.Errorf("date object: <%s> invalid date", s)
	}

	return &t1, nil
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
		return "", fmt.Errorf("date object: <%s> invalid date", s)
	}

	return types.DateString(t), nil
}

func validateDateEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (*time.Time, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateDateEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	s, err := xRefTable.DereferenceStringOrHexLiteral(o, model.V10, nil)
	if err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if s == "" {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateDateEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	time, ok := types.DateTime(s, xRefTable.ValidationMode == model.ValidationRelaxed)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid date <%s>", dictName, entryName, s)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateDateEntry end: entry=%s\n", entryName)
	}

	return &time, nil
}

func validateDictEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Dict) bool) (types.Dict, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateDictEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateDictEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	d, ok := o.(types.Dict)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	if validate != nil && len(d) > 0 && !validate(d) {
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateDictEntry end: entry=%s\n", entryName)
	}

	return d, nil
}

func validateFloatForObject(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, validate func(float64) bool) (result *types.Float, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if log.ValidateEnabled() {
		log.Validate.Println("validateFloat begin")
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("float object: missing object")
	}

	f, ok := o.(types.Float)
	if !ok {
		return nil, fmt.Errorf("float object: invalid type %T", o)
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		return nil, fmt.Errorf("invalid float: %s", f)
	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateFloat end")
	}

	return &f, nil
}

func validateFunctionArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateFunctionArrayEntry begin: entry=%s\n", entryName)
	}

	entryObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	a, err := validateArrayEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate,
	)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, entryObjNr)
	}

	for _, o := range a {
		if err = validateFunction(xRefTable, o, entryObjNr); err != nil {
			return nil, err
		}
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateFunctionArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateFunctionOrArrayOfFunctionsEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateFunctionOrArrayOfFunctionsEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	entryObjNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, entryObjNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, entryObjNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateFunctionOrArrayOfFunctionsEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	switch o := o.(type) {

	case types.Array:

		for _, o := range o {

			if o == nil {
				continue
			}

			if err = validateFunction(xRefTable, o, entryObjNr); err != nil {
				return err
			}

		}

	default:
		if err = processFunction(xRefTable, o, entryObjNr); err != nil {
			return err
		}

	}

	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, entryObjNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateFunctionOrArrayOfFunctionsEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateIndRefEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (*types.IndirectRef, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateIndRefEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, ownerObjNr)
	}

	ir, ok := o.(types.IndirectRef)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, ownerObjNr)
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, ownerObjNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateIndRefEntry end: entry=%s\n", entryName)
	}

	return &ir, nil
}

func validateIndRefArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)
	}

	arrayObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	a, err := validateArrayEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate,
	)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, arrayObjNr)
	}

	for i, o := range a {
		if o == nil {
			continue
		}
		if _, ok := o.(types.IndirectRef); !ok {
			err := fmt.Errorf("indirect reference array: invalid type at index %d", i)
			return nil, model.WithValidationErrorObject(err, arrayObjNr)
		}
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)
	}

	return a, nil
}

func validateIntegerForObject(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, validate func(int) bool) (integer *types.Integer, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if log.ValidateEnabled() {
		log.Validate.Println("validateInteger begin")
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("integer object: missing object")
	}

	i, ok := o.(types.Integer)
	if !ok {
		return nil, fmt.Errorf("integer object: invalid type %T", o)
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		return nil, fmt.Errorf("invalid integer: %s", i)
	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateInteger end")
	}

	return &i, nil
}

func validateIntegerEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(int) bool) (*types.Integer, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntegerEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateIntegerEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	i, ok := o.(types.Integer)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntegerEntry end: entry=%s\n", entryName)
	}

	return &i, nil
}

func validateIntegerArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntegerArrayEntry begin: entry=%s\n", entryName)
	}

	rawEntry, _ := d.Find(entryName)
	entryObjNr := validationObjectNumber(ownerObjNr, rawEntry)
	a, err := validateArrayEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, entryObjNr)
	}

	for i, o := range a {
		objNr := validationObjectNumber(entryObjNr, o)

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
		}

		if o == nil {
			continue
		}

		if _, ok := o.(types.Integer); !ok {
			err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateNameForObject(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, validate func(string) bool) (result *types.Name, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if log.ValidateEnabled() {
		log.Validate.Println("validateName begin")
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("name object: missing object")
	}

	name, ok := o.(types.Name)
	if !ok {
		return nil, fmt.Errorf("name object: invalid type %T", o)
	}

	// Validation
	if validate != nil && !validate(name.Value()) {
		return nil, fmt.Errorf("invalid name: %s", name)
	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateName end")
	}

	return &name, nil
}

func validateNameEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(string) bool) (*types.Name, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateNameEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	name, ok := o.(types.Name)
	if !ok {
		err := fmt.Errorf("dict=%s entry=%s invalid type %T", dictName, entryName, o)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	v := name.Value()
	if validate != nil && (required || len(v) > 0) && !validate(v) {
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry: %s", dictName, entryName, v)
		return &name, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameEntry end: entry=%s\n", entryName)
	}

	return &name, nil
}

func validateNameArray(xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Println("validateNameArray begin")
	}

	arrayObjNr := validationObjectNumber(ownerObjNr, o)
	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, arrayObjNr)
	}

	for i, raw := range a {
		objNr := validationObjectNumber(arrayObjNr, raw)

		o, err := xRefTable.Dereference(raw)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
		}

		if o == nil {
			continue
		}

		if _, ok := o.(types.Name); !ok {
			err := fmt.Errorf("name array: invalid type at index %d", i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateNameArray end")
	}

	return a, nil
}

func validateNameArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(a types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameArrayEntry begin: entry=%s\n", entryName)
	}

	rawEntry, _ := d.Find(entryName)
	entryObjNr := validationObjectNumber(ownerObjNr, rawEntry)
	a, err := validateArrayEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, entryObjNr)
	}

	for i, o := range a {
		objNr := validationObjectNumber(entryObjNr, o)

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
		}

		if o == nil {
			continue
		}

		if _, ok := o.(types.Name); !ok {
			err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateNumberForObject(xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (number types.Object, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if log.ValidateEnabled() {
		log.Validate.Println("validateNumber begin")
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("number object: missing object")
	}

	switch o.(type) {

	case types.Integer:
		// no further processing.

	case types.Float:
		// no further processing.

	default:
		return nil, fmt.Errorf("number object: invalid type %T", o)

	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateNumber end ")
	}

	return o, nil
}

func validateNumberEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(f float64) bool) (types.Object, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateNumberEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = validateNumberForObject(xRefTable, o, ownerObjNr); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
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
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry: %g", dictName, entryName, f)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateNumberEntry end: entry=%s\n", entryName)
	}

	return o, nil
}

func validateNumberEntryToFloat(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, validate func(f float64) bool) (float64, error) {
	obj, err := validateNumberEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return 0, err
	}

	f := 0.0

	switch o := obj.(type) {
	case types.Integer:
		f = float64(o.Value())
	case types.Float:
		f = o.Value()
	}

	return f, nil
}

func validateNumberArray(xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Println("validateNumberArray begin")
	}

	arrayObjNr := validationObjectNumber(ownerObjNr, o)
	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, arrayObjNr)
	}

	for i, raw := range a {
		objNr := validationObjectNumber(arrayObjNr, raw)

		o, err := xRefTable.Dereference(raw)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
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
			err := fmt.Errorf("number array: invalid type at index %d", i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateNumberArray end")
	}

	return a, err
}

func validateNumberArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateNumberArrayEntry begin: entry=%s\n", entryName)
	}

	entryObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	a, err := validateArrayEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, entryObjNr)
	}

	for i, o := range a {
		objNr := validationObjectNumber(entryObjNr, o)

		o, err := xRefTable.Dereference(o)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
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
			err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateRectangleEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateRectangleEntry begin: entry=%s\n", entryName)
	}

	a, err := validateNumberArrayEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, func(a types.Array) bool { return len(a) == 4 },
	)
	if err != nil || a == nil {
		return nil, err
	}

	if validate != nil && !validate(a) {
		err := fmt.Errorf("dict=%s entry=%s invalid rectangle entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, entryName))
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateRectangleEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateStreamDictForObject(xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (streamDict *types.StreamDict, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if log.ValidateEnabled() {
		log.Validate.Println("validateStreamDict begin")
	}

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, errors.New("stream dict: missing object")
	}

	sd, ok := o.(types.StreamDict)
	if !ok {
		return nil, fmt.Errorf("stream dict: invalid type %T", o)
	}

	if log.ValidateEnabled() {
		log.Validate.Println("validateStreamDict endobj")
	}

	return &sd, nil
}

func validateStreamDictEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.StreamDict) bool) (*types.StreamDict, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateStreamDictEntry begin: entry=%s\n", entryName)
	}

	o, found, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}
	if o == nil {
		if !found {
			return nil, nil
		}
		if xRefTable.ValidationMode == model.ValidationStrict {
			err := fmt.Errorf("dict=%s optional entry=%s is corrupt", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		delete(d, entryName)
		model.ShowRepaired("root dict \"Metadata\"")
	}

	sd, valid, err := xRefTable.DereferenceStreamDict(o)
	if valid {
		return nil, nil
	}

	if err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if sd == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateStreamDictEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	if validate != nil && !validate(*sd) {
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateStreamDictEntry end: entry=%s\n", entryName)
	}

	return sd, nil
}

func decodeString(o types.Object, dictName, entryName string) (s string, err error) {
	switch o := o.(type) {
	case types.StringLiteral:
		s, err = types.StringLiteralToString(o)
	case types.HexLiteral:
		s, err = types.HexLiteralToString(o)
	default:
		err = fmt.Errorf("dict=%s entry=%s invalid type %T", dictName, entryName, o)
	}
	return s, err
}

func validateStringEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(string) bool) (*string, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateStringEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateStringEntry end: optional entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	// Version check
	if err = xRefTable.ValidateVersion(fmt.Sprintf("dict=%s entry=%s", dictName, entryName), sinceVersion); err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	s, err := decodeString(o, dictName, entryName)
	if err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	// Validation
	if validate != nil && (required || len(s) > 0) && !validate(s) {
		err := fmt.Errorf("dict=%s entry=%s invalid dict entry", dictName, entryName)
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateStringEntry end: entry=%s\n", entryName)
	}

	return &s, nil
}

func validateStringArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateStringArrayEntry begin: entry=%s\n", entryName)
	}

	a, err := validateArrayEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate)
	if err != nil || a == nil {
		return nil, err
	}
	arrayObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)

	for i, o := range a {
		context := objectContext(fmt.Sprintf("dict=%s entry=%s[%d]", dictName, entryName, i), o)
		objNr := validationObjectNumber(arrayObjNr, o)

		o, err := xRefTable.Dereference(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference: %w", context, err)
			return nil, model.WithValidationErrorObject(err, objNr)
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
			err = fmt.Errorf(
				"%s: invalid type %T, expected types.StringLiteral or types.HexLiteral",
				context,
				o,
			)
			return nil, model.WithValidationErrorObject(err, objNr)
		}

	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateStringArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateArrayArrayEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version, validate func(types.Array) bool) (types.Array, error) {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateArrayArrayEntry begin: entry=%s\n", entryName)
	}

	entryObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	a, err := validateArrayEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, validate,
	)
	if err != nil || a == nil {
		return nil, model.WithValidationErrorObject(err, entryObjNr)
	}

	for i, raw := range a {
		objNr := validationObjectNumber(entryObjNr, raw)
		o, err := xRefTable.Dereference(raw)
		if err != nil {
			return nil, model.WithValidationErrorObject(err, objNr)
		}

		if o == nil {
			continue
		}

		if _, ok := o.(types.Array); !ok {
			err := fmt.Errorf("array array: invalid type at index %d", i)
			return nil, model.WithValidationErrorObject(err, objNr)
		}
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateArrayArrayEntry end: entry=%s\n", entryName)
	}

	return a, nil
}

func validateStringOrStreamEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateStringOrStreamEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateStringOrStreamEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	switch o.(type) {

	case types.StringLiteral, types.HexLiteral, types.StreamDict:
		// no further processing

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateStringOrStreamEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateNameOrStringEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameOrStringEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateNameOrStringEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	switch o.(type) {

	case types.StringLiteral, types.Name:
		// no further processing

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameOrStringEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateIntOrStringEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntOrStringEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateIntOrStringEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	switch o.(type) {

	case types.StringLiteral, types.HexLiteral, types.Integer:
		// no further processing

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntOrStringEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateBooleanOrStreamEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanOrStreamEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateBooleanOrStreamEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	switch o.(type) {

	case types.Boolean, types.StreamDict:
		// no further processing

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanOrStreamEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateStreamDictOrDictEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateStreamDictOrDictEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateStreamDictOrDictEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	switch o.(type) {

	case types.StreamDict:
		// TODO validate 3D stream dict

	case types.Dict:
		// TODO validate 3D reference dict

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, objNr)
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateStreamDictOrDictEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateIntegerOrArrayOfInteger(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, dictName, entryName string) error {
	switch o := o.(type) {

	case types.Integer:
		// no further processing

	case types.Array:

		for i, raw := range o {
			objNr := validationObjectNumber(ownerObjNr, raw)
			o, err := xRefTable.Dereference(raw)
			if err != nil {
				return model.WithValidationErrorObject(err, objNr)
			}

			if o == nil {
				continue
			}

			if _, ok := o.(types.Integer); !ok {
				err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
				return model.WithValidationErrorObject(err, objNr)
			}

		}

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	return nil
}

func validateIntegerOrArrayOfIntegerEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntegerOrArrayOfIntegerEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateIntegerOrArrayOfIntegerEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if err := validateIntegerOrArrayOfInteger(xRefTable, o, objNr, dictName, entryName); err != nil {
		return err
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateIntegerOrArrayOfIntegerEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateNameOrArrayOfName(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, dictName, entryName string) error {
	switch o := o.(type) {

	case types.Name:
		// no further processing

	case types.Array:

		for i, raw := range o {
			objNr := validationObjectNumber(ownerObjNr, raw)
			o, err := xRefTable.Dereference(raw)
			if err != nil {
				return model.WithValidationErrorObject(err, objNr)
			}

			if o == nil {
				continue
			}

			if _, ok := o.(types.Name); !ok {
				err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
				return model.WithValidationErrorObject(err, objNr)
			}

		}

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	return nil
}

func validateNameOrArrayOfNameEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameOrArrayOfNameEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateNameOrArrayOfNameEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if err := validateNameOrArrayOfName(xRefTable, o, objNr, dictName, entryName); err != nil {
		return err
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateNameOrArrayOfNameEntry end: entry=%s\n", entryName)
	}

	return nil
}

func validateBooleanOrArrayOfBoolean(xRefTable *model.XRefTable, o types.Object, ownerObjNr int, dictName, entryName string) error {
	switch o := o.(type) {

	case types.Boolean:
		// no further processing

	case types.Array:

		for i, raw := range o {
			objNr := validationObjectNumber(ownerObjNr, raw)
			o, err := xRefTable.Dereference(raw)
			if err != nil {
				return model.WithValidationErrorObject(err, objNr)
			}

			if o == nil {
				continue
			}

			if _, ok := o.(types.Boolean); !ok {
				err := fmt.Errorf("dict=%s entry=%s invalid type at index %d", dictName, entryName, i)
				return model.WithValidationErrorObject(err, objNr)
			}

		}

	default:
		err := fmt.Errorf("dict=%s entry=%s invalid type", dictName, entryName)
		return model.WithValidationErrorObject(err, ownerObjNr)
	}

	return nil
}

func validateBooleanOrArrayOfBooleanEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanOrArrayOfBooleanEntry begin: entry=%s\n", entryName)
	}

	o, _, err := d.Entry(dictName, entryName, required)
	objNr := validationObjectNumber(ownerObjNr, o)
	if err != nil || o == nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o, err = xRefTable.Dereference(o); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if o == nil {
		if required {
			err := fmt.Errorf("dict=%s required entry=%s is nil", dictName, entryName)
			return model.WithValidationErrorObject(err, objNr)
		}
		if log.ValidateEnabled() {
			log.Validate.Printf("validateBooleanOrArrayOfBooleanEntry end: optional entry %s is nil\n", entryName)
		}
		return nil
	}

	// Version check
	if err = xRefTable.ValidateVersion("dict="+dictName+" entry="+entryName, sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}

	if err := validateBooleanOrArrayOfBoolean(xRefTable, o, objNr, dictName, entryName); err != nil {
		return err
	}

	if log.ValidateEnabled() {
		log.Validate.Printf("validateBooleanOrArrayOfBooleanEntry end: entry=%s\n", entryName)
	}

	return nil
}

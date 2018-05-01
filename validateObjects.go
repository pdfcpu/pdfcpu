package pdfcpu

import (
	"fmt"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

func validateEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) (PDFObject, error) {

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

func validateArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

	arr, ok := obj.(PDFArray)
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

func validateBooleanEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(bool) bool) (*PDFBoolean, error) {

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

	b, ok := obj.(PDFBoolean)
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

func validateBooleanArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

		_, ok := obj.(PDFBoolean)
		if !ok {
			return nil, errors.Errorf("validateBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Debug.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateDateObject(xRefTable *XRefTable, obj PDFObject, sinceVersion PDFVersion) (PDFStringLiteral, error) {
	return xRefTable.DereferenceStringLiteral(obj, sinceVersion, validateDate)
}

func validateDateEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) (*PDFStringLiteral, error) {

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

	date, ok := obj.(PDFStringLiteral)
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

func validateDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFDict) bool) (*PDFDict, error) {

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

	d, ok := obj.(PDFDict)
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

func validateFloat(xRefTable *XRefTable, obj PDFObject, validate func(float64) bool) (*PDFFloat, error) {

	log.Debug.Println("validateFloat begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateFloat: missing object")
	}

	f, ok := obj.(PDFFloat)
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

func validateFloatEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(float64) bool) (*PDFFloat, error) {

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

	f, ok := obj.(PDFFloat)
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

func validateFunctionEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

func validateFunctionArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

func validateFunctionOrArrayOfFunctionsEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFArray:

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

func validateIndRefEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) (*PDFIndirectRef, error) {

	log.Debug.Printf("validateIndRefEntry begin: entry=%s\n", entryName)

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return nil, err
	}

	indRef, ok := obj.(PDFIndirectRef)
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

func validateIndRefArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

	log.Debug.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arr == nil {
		return nil, err
	}

	for i, obj := range *arr {
		_, ok := obj.(PDFIndirectRef)
		if !ok {
			return nil, errors.Errorf("validateIndRefArrayEntry: invalid type at index %d\n", i)
		}
	}

	log.Debug.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)

	return arr, nil
}

func validateInteger(xRefTable *XRefTable, obj PDFObject, validate func(int) bool) (*PDFInteger, error) {

	log.Debug.Println("validateInteger begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, errors.New("validateInteger: missing object")
	}

	i, ok := obj.(PDFInteger)
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

func validateIntegerEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(int) bool) (*PDFInteger, error) {

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

	i, ok := obj.(PDFInteger)
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

func validateIntegerArray(xRefTable *XRefTable, obj PDFObject) (*PDFArray, error) {

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

		case PDFInteger:
			// no further processing.

		default:
			return nil, errors.Errorf("validateIntegerArray: invalid type at index %d\n", i)
		}

	}

	log.Debug.Println("validateIntegerArray end")

	return a, nil
}

func validateIntegerArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

		_, ok := obj.(PDFInteger)
		if !ok {
			return nil, errors.Errorf("validateIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Debug.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateName(xRefTable *XRefTable, obj PDFObject, validate func(string) bool) (*PDFName, error) {

	log.Debug.Println("validateName begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateName: missing object")
	}

	name, ok := obj.(PDFName)
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

func validateNameEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(string) bool) (*PDFName, error) {

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

	name, ok := obj.(PDFName)
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

func validateNameArray(xRefTable *XRefTable, obj PDFObject) (*PDFArray, error) {

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

		_, ok := obj.(PDFName)
		if !ok {
			return nil, errors.Errorf("validateNameArray: invalid type at index %d\n", i)
		}

	}

	log.Debug.Println("validateNameArray end")

	return arr, nil
}

func validateNameArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(a PDFArray) bool) (*PDFArray, error) {

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

		_, ok := obj.(PDFName)
		if !ok {
			return nil, errors.Errorf("validateNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
		}

	}

	log.Debug.Printf("validateNameArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateNumber(xRefTable *XRefTable, obj PDFObject) (PDFObject, error) {

	log.Debug.Println("validateNumber begin")

	n, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errors.New("validateNumber: missing object")
	}

	switch n.(type) {

	case PDFInteger:
		// no further processing.

	case PDFFloat:
		// no further processing.

	default:
		return nil, errors.New("validateNumber: invalid type")

	}

	log.Debug.Println("validateNumber end ")

	return n, nil
}

func validateNumberEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(f float64) bool) (PDFObject, error) {

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

	case PDFInteger:
		f = float64(o.Value())

	case PDFFloat:
		f = o.Value()
	}

	if validate != nil && !validate(f) {
		return nil, errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	log.Debug.Printf("validateNumberEntry end: entry=%s\n", entryName)

	return obj, nil
}

func validateNumberArray(xRefTable *XRefTable, obj PDFObject) (*PDFArray, error) {

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

		case PDFInteger:
			// no further processing.

		case PDFFloat:
			// no further processing.

		default:
			return nil, errors.Errorf("validateNumberArray: invalid type at index %d\n", i)
		}

	}

	log.Debug.Println("validateNumberArray end")

	return nil, err
}

func validateNumberArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

		case PDFInteger:
			// no further processing.

		case PDFFloat:
			// no further processing.

		default:
			return nil, errors.Errorf("validateNumberArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Debug.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateRectangleEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

	log.Debug.Printf("validateRectangleEntry begin: entry=%s\n", entryName)

	arr, err := validateNumberArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, func(arr PDFArray) bool { return len(arr) == 4 })
	if err != nil || arr == nil {
		return nil, err
	}

	if validate != nil && !validate(*arr) {
		return nil, errors.Errorf("validateRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
	}

	log.Debug.Printf("validateRectangleEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateStreamDict(xRefTable *XRefTable, obj PDFObject) (*PDFStreamDict, error) {

	log.Debug.Println("validateStreamDict begin")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("validateStreamDict: missing object")
	}

	sd, ok := obj.(PDFStreamDict)
	if !ok {
		return nil, errors.New("validateStreamDict: invalid type")
	}

	log.Debug.Println("validateStreamDict endobj")

	return &sd, nil
}

func validateStreamDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFStreamDict) bool) (*PDFStreamDict, error) {

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

	sd, ok := obj.(PDFStreamDict)
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

func validateString(xRefTable *XRefTable, obj PDFObject, validate func(string) bool) (*string, error) {

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

	case PDFStringLiteral:
		s = obj.Value()

	case PDFHexLiteral:
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

func validateStringEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(string) bool) (*string, error) {

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

	case PDFStringLiteral:
		s = obj.Value()

	case PDFHexLiteral:
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

func validateStringArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

		case PDFStringLiteral:
			// no further processing.

		case PDFHexLiteral:
			// no further processing

		default:
			return nil, errors.Errorf("validateStringArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Debug.Printf("validateStringArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateArrayArrayEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion, validate func(PDFArray) bool) (*PDFArray, error) {

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

		case PDFArray:
			// no further processing.

		default:
			return nil, errors.Errorf("validateArrayArrayEntry: invalid type at index %d\n", i)
		}

	}

	log.Debug.Printf("validateArrayArrayEntry end: entry=%s\n", entryName)

	return arr, nil
}

func validateStringOrStreamEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFStringLiteral, PDFHexLiteral, PDFStreamDict:
		// no further processing

	default:
		return errors.Errorf("validateStringOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateStringOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateNameOrStringEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFStringLiteral, PDFName:
		// no further processing

	default:
		return errors.Errorf("validateNameOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateNameOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrStringEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFStringLiteral, PDFHexLiteral, PDFInteger:
		// no further processing

	default:
		return errors.Errorf("validateIntOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateIntOrStringEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntOrDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFInteger, PDFDict:
		// no further processing

	default:
		return errors.Errorf("validateIntOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateIntOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateBooleanOrStreamEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFBoolean, PDFStreamDict:
		// no further processing

	default:
		return errors.Errorf("validateBooleanOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateBooleanOrStreamEntry end: entry=%s\n", entryName)

	return nil
}

func validateStreamDictOrDictEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFStreamDict:
		// TODO validate 3D stream dict

	case PDFDict:
		// TODO validate 3D reference dict

	default:
		return errors.Errorf("validateStreamDictOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
	}

	log.Debug.Printf("validateStreamDictOrDictEntry end: entry=%s\n", entryName)

	return nil
}

func validateIntegerOrArrayOfIntegerEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFInteger:
		// no further processing

	case PDFArray:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(PDFInteger)
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

func validateNameOrArrayOfNameEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFName:
		// no further processing

	case PDFArray:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(PDFName)
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

func validateBooleanOrArrayOfBooleanEntry(xRefTable *XRefTable, dict *PDFDict, dictName, entryName string, required bool, sinceVersion PDFVersion) error {

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

	case PDFBoolean:
		// no further processing

	case PDFArray:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return err
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(PDFBoolean)
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

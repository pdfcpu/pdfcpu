package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

func validateAnyEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool) (err error) {

	logInfoValidate.Printf("writeAnyEntry begin: entry=%s\n", entryName)

	entry, found := dict.Find(entryName)
	if !found || entry == nil {
		if required {
			err = errors.Errorf("writeAnyEntry: missing required entry: %s", entryName)
			return
		}
		logInfoValidate.Printf("writeAnyEntry end: entry %s not found or nil\n", entryName)
		return
	}

	indRef, ok := entry.(types.PDFIndirectRef)
	if !ok {
		logInfoValidate.Println("writeAnyEntry end")
		return
	}

	objNumber := indRef.ObjectNumber.Value()

	obj, err := xRefTable.Dereference(indRef)
	if err != nil {
		return errors.Wrapf(err, "writeAnyEntry: unable to dereference object #%d", objNumber)
	}

	if obj == nil {
		return errors.Errorf("writeAnyEntry end: entry %s is nil", entryName)
	}

	switch obj.(type) {

	case types.PDFDict:
	case types.PDFStreamDict:
	case types.PDFArray:
	case types.PDFInteger:
	case types.PDFFloat:
	case types.PDFStringLiteral:
	case types.PDFHexLiteral:
	case types.PDFBoolean:
	case types.PDFName:

	default:
		err = errors.Errorf("writeAnyEntry: unsupported entry: %s", entryName)

	}

	logInfoValidate.Println("writeAnyEntry end")

	return
}

func validateArray(xRefTable *types.XRefTable, obj interface{}) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateArray begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateArray: missing object")
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		err = errors.New("validateArray: invalid type")
		return
	}

	arrp = &arr

	logInfoValidate.Println("validateArray end")

	return
}

func validateArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateArrayEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateArrayEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateArrayEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateArrayEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateArrayEntry end: optional entry %s is nil\n", entryName)
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		err = errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateArrayEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(arr) {
		err = errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	arrp = &arr

	logInfoValidate.Printf("validateArrayEntry end: entry=%s\n", entryName)

	return
}

func validateBooleanEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(bool) bool) (boolp *types.PDFBoolean, err error) {

	logInfoValidate.Printf("validateBooleanEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		return
	}

	b, ok := obj.(types.PDFBoolean)
	if !ok {
		err = errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateBooleanEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(b.Value()) {
		err = errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid name dict entry", dictName, entryName)
		return
	}

	boolp = &b

	logInfoValidate.Printf("validateBooleanEntry end: entry=%s\n", entryName)

	return
}

func validateBooleanArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateBooleanArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFBoolean)
		if !ok {
			err = errors.Errorf("validateBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoValidate.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)

	return
}

func validateDateObject(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (s types.PDFStringLiteral, err error) {
	return xRefTable.DereferenceStringLiteral(obj, sinceVersion, validateDate)
}

func validateDateEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (s *types.PDFStringLiteral, err error) {

	logInfoValidate.Printf("validateDateEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateDateEntry: missing required entry: %s", entryName)
			return
		}
		logInfoValidate.Printf("validateDateEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateDateEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDateEntry end: optional entry %s is nil\n", entryName)
		return
	}

	date, ok := obj.(types.PDFStringLiteral)
	if !ok {
		err = errors.Errorf("validateDateEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateDateEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if ok := validateDate(date.Value()); !ok {
		err = errors.Errorf("validateDateEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	s = &date

	logInfoValidate.Printf("validateDateEntry end: entry=%s\n", entryName)

	return
}

func validateDict(xRefTable *types.XRefTable, obj interface{}) (dictp *types.PDFDict, err error) {

	logInfoValidate.Println("validateDict begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateDict: missing object")
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.New("validateDict: invalid type")
		return
	}

	dictp = &dict

	logInfoValidate.Println("validateDict end")

	return
}

func validateDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFDict) bool) (dictp *types.PDFDict, err error) {

	logInfoValidate.Printf("validateDictEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDictEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	d, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.Errorf("validateDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(d) {
		err = errors.Errorf("validateDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	dictp = &d

	logInfoValidate.Printf("validateDictEntry end: entry=%s\n", entryName)

	return
}

func validateFloat(xRefTable *types.XRefTable, obj interface{}, validate func(float64) bool) (fp *types.PDFFloat, err error) {

	logInfoValidate.Println("validateFloat begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateFloat: missing object")
		return
	}

	f, ok := obj.(types.PDFFloat)
	if !ok {
		err = errors.New("validateFloat: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		err = errors.Errorf("validateFloat: invalid float: %s\n", f)
		return
	}

	fp = &f

	logInfoValidate.Println("validateFloat end")

	return
}

func validateFloatEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(float64) bool) (fp *types.PDFFloat, err error) {

	logInfoValidate.Printf("validateFloatEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateFloatEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFloatEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateFloatEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFloatEntry end: optional entry %s is nil\n", entryName)
		return
	}

	f, ok := obj.(types.PDFFloat)
	if !ok {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	fp = &f

	logInfoValidate.Printf("validateFloatEntry end: entry=%s\n", entryName)

	return
}

func validateFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateFunctionEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateFunctionEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFunctionEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateFunctionEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	err = validateFunction(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Printf("validateFunctionEntry end: entry=%s\n", entryName)

	return
}

func validateFunctionArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateFunctionArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for _, obj := range *arrp {
		err = validateFunction(xRefTable, obj)
		if err != nil {
			return
		}
	}

	logInfoValidate.Printf("validateFunctionArrayEntry end: entry=%s\n", entryName)

	return
}

func validateIndRefEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (indRefp *types.PDFIndirectRef, err error) {

	logInfoValidate.Printf("validateIndRefEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIndRefEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIndRefEntry end: entry %s is nil\n", entryName)
		return
	}

	indRef, ok := obj.(types.PDFIndirectRef)
	if !ok {
		err = errors.Errorf("validateIndRefEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIndRefEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	indRefp = &indRef

	logInfoValidate.Printf("validateIndRefEntry end: entry=%s\n", entryName)

	return
}

func validateIndRefArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		_, ok := obj.(types.PDFIndirectRef)
		if !ok {
			err = errors.Errorf("validateIndRefArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)

	return
}

func validateInteger(xRefTable *types.XRefTable, obj interface{}, validate func(int) bool) (ip *types.PDFInteger, err error) {

	logInfoValidate.Println("validateInteger begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateInteger: missing object")
		return
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		err = errors.New("validateInteger: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err = errors.Errorf("validateInteger: invalid integer: %s\n", i)
		return
	}

	ip = &i

	logInfoValidate.Println("validateInteger end")

	return
}

func validateIntegerEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(int) bool) (ip *types.PDFInteger, err error) {

	logInfoValidate.Printf("validateIntegerEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIntegerEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntegerEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntegerEntry end: optional entry %s is nil\n", entryName)
		return
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		err = errors.Errorf("validateIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIntegerEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err = errors.Errorf("validateIntegerEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	ip = &i

	logInfoValidate.Printf("validateIntegerEntry end: entry=%s\n", entryName)

	return
}

func validateIntegerArray(xRefTable *types.XRefTable, arr types.PDFArray) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateIntegerArray begin")

	arrp, err = validateArray(xRefTable, arr)
	if err != nil {
		return
	}

	if arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		default:
			err = errors.Errorf("validateIntegerArray: invalid type at index %d\n", i)
		}

	}

	logInfoValidate.Println("validateIntegerArray end")

	return
}

func validateIntegerArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateIntegerArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFInteger)
		if !ok {
			err = errors.Errorf("validateIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoValidate.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)

	return
}

func validateName(xRefTable *types.XRefTable, obj interface{}, validate func(string) bool) (namep *types.PDFName, err error) {

	// TODO written irrelevant?

	logInfoValidate.Println("validateName begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateName: missing object")
		return
	}

	name, ok := obj.(types.PDFName)
	if !ok {
		err = errors.New("validateName: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		err = errors.Errorf("validateName: invalid name: %s\n", name)
		return
	}

	namep = &name

	logInfoValidate.Println("validateName end")

	return
}

func validateNameEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (namep *types.PDFName, err error) {

	logInfoValidate.Printf("validateNameEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateNameEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateNameEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateNameEntry end: optional entry %s is nil\n", entryName)
		return
	}

	name, ok := obj.(types.PDFName)
	if !ok {
		err = errors.Errorf("validateNameEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateNameEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		err = errors.Errorf("validateNameEntry: dict=%s entry=%s invalid dict entry: %s", dictName, entryName, name.String())
		return
	}

	namep = &name

	logInfoValidate.Printf("validateNameEntry end: entry=%s\n", entryName)

	return
}

func validateNameArray(xRefTable *types.XRefTable, obj interface{}) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateNameArray begin")

	arrp, err = validateArray(xRefTable, obj)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFName)
		if !ok {
			err = errors.Errorf("validateNameArray: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Println("validateNameArray end")

	return
}

func validateNameArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(a types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateNameArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFName)
		if !ok {
			err = errors.Errorf("validateNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoValidate.Printf("validateNameArrayEntry end: entry=%s\n", entryName)

	return
}

func validateNumber(xRefTable *types.XRefTable, obj interface{}) (n interface{}, err error) {

	logInfoValidate.Println("validateNumber begin")

	n, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if n == nil {
		err = errors.New("validateNumber: missing object")
		return
	}

	switch n.(type) {

	case types.PDFInteger:
		// no further processing.

	case types.PDFFloat:
		// no further processing.

	default:
		err = errors.New("validateNumber: invalid type")

	}

	logInfoValidate.Println("validateNumber end ")

	return
}

func validateNumberEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(interface{}) bool) (obj interface{}, err error) {

	logInfoValidate.Printf("validateNumberEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateNumberEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateNumberEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateNumberEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	obj, err = validateNumber(xRefTable, obj)
	if err != nil {
		return
	}

	// Validation
	// TODO Would be nice if we could always validate against a float here.
	if validate != nil && !validate(obj) {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	logInfoValidate.Printf("validateNumberEntry end: entry=%s\n", entryName)

	return
}

func validateNumberArray(xRefTable *types.XRefTable, obj interface{}) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateNumberArray begin")

	arrp, err = validateArray(xRefTable, obj)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing.

		default:
			err = errors.Errorf("validateNumberArray: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Println("validateNumberArray end")

	return
}

func validateNumberArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateNumberArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing.

		default:
			err = errors.Errorf("validateNumberArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)

	return
}

func validateRectangleEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateRectangleEntry begin: entry=%s\n", entryName)

	arrp, err = validateNumberArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return
	}

	if arrp == nil {
		return
	}

	if validate != nil && !validate(*arrp) {
		err = errors.Errorf("validateRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
		return
	}

	logInfoValidate.Printf("validateRectangleEntry end: entry=%s\n", entryName)

	return
}

func validateStreamDict(xRefTable *types.XRefTable, obj interface{}) (streamDictp *types.PDFStreamDict, err error) {

	logInfoValidate.Println("validateStreamDict begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateStreamDict: missing object")
		return
	}

	streamDict, ok := obj.(types.PDFStreamDict)
	if !ok {
		err = errors.New("validateStreamDict: invalid type")
		return
	}

	streamDictp = &streamDict

	logInfoValidate.Println("validateStreamDict endobj")

	return
}

func validateStreamDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFStreamDict) bool) (sdp *types.PDFStreamDict, err error) {

	logInfoValidate.Printf("validateStreamDictEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateStreamDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStreamDictEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateStreamDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStreamDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	sd, ok := obj.(types.PDFStreamDict)
	if !ok {
		err = errors.Errorf("validateStreamDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateStreamDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(sd) {
		err = errors.Errorf("validateStreamDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	sdp = &sd

	logInfoValidate.Printf("validateStreamDictEntry end: entry=%s\n", entryName)

	return
}

func validateString(xRefTable *types.XRefTable, obj interface{}, validate func(string) bool) (s *string, err error) {

	logInfoValidate.Println("validateString begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateString: missing object")
		return
	}

	var str string

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		str = obj.Value()

	case types.PDFHexLiteral:
		str = obj.Value()

	default:
		err = errors.New("validateString: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(str) {
		err = errors.Errorf("validateString: %s invalid", str)
		return
	}

	s = &str

	logInfoValidate.Println("validateString end")

	return
}

func validateStringEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (s *string, err error) {

	logInfoValidate.Printf("validateStringEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateStringEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStringEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStringEntry end: optional entry %s is nil\n", entryName)
		return
	}

	var str string

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		str = obj.Value()

	case types.PDFHexLiteral:
		str = obj.Value()

	default:
		err = errors.Errorf("validateStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateStringEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(str) {
		err = errors.Errorf("validateStringEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	s = &str

	logInfoValidate.Printf("validateStringEntry end: entry=%s\n", entryName)

	return
}

func validateStringArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateStringArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFStringLiteral:
			// no further processing.

		case types.PDFHexLiteral:
			// no further processing

		default:
			err = errors.Errorf("validateStringArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateStringArrayEntry end: entry=%s\n", entryName)

	return
}

func validateArrayArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateArrayArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFArray:
			// no further processing.

		default:
			err = errors.Errorf("validateArrayArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateArrayArrayEntry end: entry=%s\n", entryName)

	return
}

func validateStringOrStreamEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateStringOrStreamEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateStringOrStreamEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStringOrStreamEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateStringOrStreamEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStringOrStreamEntry end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj.(type) {

	case types.PDFStringLiteral, types.PDFHexLiteral, types.PDFStreamDict:
		// no further processing

	default:
		err = errors.Errorf("validateStringOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateStringOrStreamEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	logInfoValidate.Printf("validateStringOrStreamEntry end: entry=%s\n", entryName)

	return
}

func validateIntOrStringEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateIntOrStringEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIntOrStringEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntOrStringEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateIntOrStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntOrStringEntry end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj.(type) {

	case types.PDFStringLiteral, types.PDFHexLiteral, types.PDFInteger:
		// no further processing

	default:
		err = errors.Errorf("validateIntOrStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIntOrStringEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	logInfoValidate.Printf("validateIntOrStringEntry end: entry=%s\n", entryName)

	return
}

func validateIntOrDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateIntOrDictEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIntOrDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntOrDictEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateIntOrDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntOrDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj.(type) {

	case types.PDFInteger, types.PDFDict:
		// no further processing

	default:
		err = errors.Errorf("validateIntOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIntOrDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	logInfoValidate.Printf("validateIntOrDictEntry end: entry=%s\n", entryName)

	return
}

func validateBooleanOrStreamEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateBooleanOrStreamEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateBooleanOrStreamEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateBooleanOrStreamEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateBooleanOrStreamEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateBooleanOrStreamEntry end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj.(type) {

	case types.PDFBoolean, types.PDFStreamDict:
		// no further processing

	default:
		err = errors.Errorf("validateBooleanOrStreamEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateBooleanOrStreamEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	logInfoValidate.Printf("validateBooleanOrStreamEntry end: entry=%s\n", entryName)

	return
}

func validateStreamDictOrDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateStreamDictOrDictEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateStreamDictOrDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStreamDictOrDictEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateStreamDictOrDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStreamDictOrDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj.(type) {

	case types.PDFStreamDict, types.PDFDict:
		// no further processing

	default:
		err = errors.Errorf("validateStreamDictOrDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateStreamDictOrDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	logInfoValidate.Printf("validateStreamDictOrDictEntry end: entry=%s\n", entryName)

	return
}

func validateIntegerOrArrayOfInteger(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateIntegerOrArrayOfInteger begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIntegerOrArrayOfInteger: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntegerOrArrayOfInteger end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateIntegerOrArrayOfInteger: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntegerOrArrayOfInteger end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFStringLiteral, types.PDFHexLiteral:
		// no further processing

	case types.PDFArray:

		for i, obj := range obj {

			obj, err = xRefTable.Dereference(obj)
			if err != nil {
				return
			}

			if obj == nil {
				continue
			}

			_, ok := obj.(types.PDFInteger)
			if !ok {
				err = errors.Errorf("validateIntegerOrArrayOfInteger: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
				return
			}

		}

	default:
		err = errors.Errorf("validateIntegerOrArrayOfInteger: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIntegerOrArrayOfInteger: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	logInfoValidate.Printf("validateIntegerOrArrayOfInteger end: entry=%s\n", entryName)

	return
}

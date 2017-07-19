package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateOptionalContentGroupIntent(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 8.11.2.1

	logInfoValidate.Println("*** validateOptionalContentGroupIntent begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateOptionalContentGroupIntent: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateOptionalContentGroupIntent end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateOptionalContentGroupIntent: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateOptionalContentGroupIntent end: optional entry %s is nil\n", entryName)
		return
	}

	validate := func(s string) bool {
		return s == "View" || s == "Design"
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if !validate(obj.Value()) {
			err = errors.Errorf("validateOptionalContentGroupIntent: invalid intent: %s", obj.Value())
			return
		}

	case types.PDFArray:

		for i, v := range obj {

			if v == nil {
				continue
			}

			n, ok := v.(types.PDFName)
			if !ok {
				err = errors.Errorf("validateOptionalContentGroupIntent: invalid type at index %d\n", i)
				return
			}

			if !validate(n.Value()) {
				err = errors.Errorf("validateOptionalContentGroupIntent: invalid intent: %s", n.Value())
				return
			}
		}

	default:
		err = errors.New("validateOptionalContentGroupIntent: invalid type")
		return
	}

	logInfoValidate.Println("*** validateOptionalContentGroupIntent end ***")

	return
}

// TODO implement
func validateOptionalContentGroupUsageDict(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 8.11.4.4

	logInfoValidate.Println("*** validateOptionalContentGroupUsageDict begin ***")

	var d *types.PDFDict

	d, err = validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return
	}

	dictName = "OCUsageDict"

	err = errors.New("*** unsupported entry OCG usage dict ***")

	logInfoValidate.Println("*** validateOptionalContentGroupUsageDict end ***")

	return
}

func validateOptionalContentGroupDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 8.11 Optional Content

	logInfoValidate.Println("*** validateOptionalContentGroupDict begin ***")

	dictName := "optionalContentGroupDict"

	// Type, required, name, OCG
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", REQUIRED, types.V10, func(s string) bool { return s == "OCG" })
	if err != nil {
		return
	}

	// Name, required, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Name", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Intent, optional, name or array
	err = validateOptionalContentGroupIntent(xRefTable, dict, dictName, "Intent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Usage, optional, usage dict
	err = validateOptionalContentGroupUsageDict(xRefTable, dict, dictName, "Usage", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateOptionalContentGroupDict end ***")

	return
}

func validateOCGs(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool) (err error) {

	// see 8.11.2.2

	logInfoValidate.Println("*** validateOCGs begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateOCGs: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateOCGs end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateOCGs: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateOCGs end: optional entry %s is nil\n", entryName)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = validateOptionalContentGroupDict(xRefTable, &obj)
		if err != nil {
			return
		}

	case types.PDFArray:

		for i, v := range obj {

			if v == nil {
				continue
			}

			d, ok := v.(types.PDFDict)
			if !ok {
				err = errors.Errorf("validateOCGs: invalid type at index %d\n", i)
				return
			}

			err = validateOptionalContentGroupDict(xRefTable, &d)
			if err != nil {
				return
			}
		}

	default:
		err = errors.New("validateOCGs: invalid type")
		return
	}

	logInfoValidate.Println("*** validateOCGs end ***")

	return
}

func validateOptionalContentMembershipDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 8.11.2.2

	logInfoValidate.Println("*** validateOptionalContentMembershipDict begin ***")

	dictName := "OCMDict"

	// OCGs, optional, dict or array
	err = validateOCGs(xRefTable, dict, dictName, "OCGs", OPTIONAL)
	if err != nil {
		return
	}

	// P, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, validateVisibilityPolicy)
	if err != nil {
		return
	}

	// VE, optional, array, since V1.6
	_, err = validateArrayEntry(xRefTable, dict, dictName, "VE", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateOptionalContentMembershipDict end ***")

	return
}

// TODO implement
func validateOptionalContentGroupArray(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateOptionalContentGroupArray begin ***")

	logInfoValidate.Println("*** validateOptionalContentGroupArray end ***")

	return
}

func validateOptContentConfigDictIntentEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateOptContentConfigDictIntentEntry begin ***")

	obj, found := dict.Find(dictEntry)
	if !found || obj == nil {
		return
	}

	obj, err = xRefTable.DereferenceDict(obj)
	if err != nil || obj == nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if !validateOptContentConfigDictIntent(obj.String()) {
			err = errors.Errorf("validateOptContentConfigDictIntentEntry: invalid entry")
		}

	case types.PDFArray:
		for _, obj := range obj {
			_, err := validateName(xRefTable, obj, validateOptContentConfigDictIntent)
			if err != nil {
				return err
			}
		}

	default:
		err = errors.Errorf("validateOptContentConfigDictIntentEntry: must be stream dict or array")
		return
	}

	logInfoValidate.Println("*** validateOptContentConfigDictIntentEntry end ***")

	return
}

// TODO implement
func validateUsageApplicationDictArray(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateUsageApplicationDictArray begin ***")

	logInfoValidate.Println("*** validateUsageApplicationDictArray end ***")

	return
}

// TODO implement
func validateOptContentConfigDictOrderEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateOptContentConfigDictOrderEntry begin ***")

	logInfoValidate.Println("*** validateOptContentConfigDictOrderEntry end ***")

	return
}

// TODO implement
func validateRBGroupsEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateRBGroupsEntry begin ***")

	logInfoValidate.Println("*** validateRBGroupsEntry end ***")

	return
}

func validateOptionalContentConfigurationDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateOptionalContentConfigurationDict begin ***")

	dictName := "optContentConfigDict"

	_, err = validateStringEntry(xRefTable, dict, dictName, "Name", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "Creator", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	baseState, err := validateNameEntry(xRefTable, dict, dictName, "BaseState", OPTIONAL, types.V10, validateBaseState)
	if err != nil {
		return
	}

	if baseState != nil {

		if baseState.String() != "ON" {
			err = validateOptionalContentGroupArray(xRefTable, dict, dictName, "ON", OPTIONAL, types.V10)
			if err != nil {
				return err
			}
		}

		if baseState.String() != "OFF" {
			err = validateOptionalContentGroupArray(xRefTable, dict, dictName, "OFF", OPTIONAL, types.V10)
			if err != nil {
				return err
			}
		}

	}

	err = validateOptContentConfigDictIntentEntry(xRefTable, dict, dictName, "Intent", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	err = validateUsageApplicationDictArray(xRefTable, dict, dictName, "AS", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	err = validateOptContentConfigDictOrderEntry(xRefTable, dict, dictName, "Order", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "ListMode", OPTIONAL, types.V10, validateListMode)
	if err != nil {
		return
	}

	err = validateRBGroupsEntry(xRefTable, dict, dictName, "RBGroups", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	err = validateOptionalContentGroupArray(xRefTable, dict, dictName, "Locked", OPTIONAL, types.V16)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateOptionalContentConfigurationDict end ***")

	return
}

func validateOCProperties(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// aka optional content properties dict.

	// => 8.11.4 Configuring Optional Content

	logInfoValidate.Println("*** validateOCProperties begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "OCProperties", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateOCProperties end: object is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateOCProperties: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// "OCGs" required array of already written indRefs
	_, err = validateIndRefArrayEntry(xRefTable, dict, "optContPropsDict", "OCGs", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// "D" required dict, default viewing optional content configuration dict.
	d, err := validateDictEntry(xRefTable, dict, "optContPropsDict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}
	if d == nil {
		err = validateOptionalContentConfigurationDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// "Configs" optional array of alternate optional content configuration dicts.
	arr, err := validateArrayEntry(xRefTable, dict, "optContPropsDict", "Configs", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if arr != nil {
		for _, value := range *arr {
			d, err = validateDict(xRefTable, value)
			if err != nil {
				return
			}
			err = validateOptionalContentConfigurationDict(xRefTable, d)
			if err != nil {
				return
			}
		}
	}

	logInfoValidate.Println("*** validateOCProperties end ***")

	return
}

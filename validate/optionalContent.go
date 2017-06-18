package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

// TODO implement
func validateOptionalContentGroupArray(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateOptionalContentGroupArray begin ***")

	logInfoValidate.Println("*** validateOptionalContentGroupArray end ***")

	return
}

func validateOptContentConfigDictIntentEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

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
func validateUsageApplicationDictArray(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateUsageApplicationDictArray begin ***")

	logInfoValidate.Println("*** validateUsageApplicationDictArray end ***")

	return
}

// TODO implement
func validateOptContentConfigDictOrderEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateOptContentConfigDictOrderEntry begin ***")

	logInfoValidate.Println("*** validateOptContentConfigDictOrderEntry end ***")

	return
}

// TODO implement
func validateRBGroupsEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

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

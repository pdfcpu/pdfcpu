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
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateOptionalContentGroupIntent(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 8.11.2.1

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil || obj == nil {
		return err
	}

	validate := func(s string) bool {
		return s == "View" || s == "Design" || s == "All"
	}

	switch obj := obj.(type) {

	case pdf.Name:
		if !validate(obj.Value()) {
			return errors.Errorf("validateOptionalContentGroupIntent: invalid intent: %s", obj.Value())
		}

	case pdf.Array:

		for i, v := range obj {

			if v == nil {
				continue
			}

			n, ok := v.(pdf.Name)
			if !ok {
				return errors.Errorf("validateOptionalContentGroupIntent: invalid type at index %d\n", i)
			}

			if !validate(n.Value()) {
				return errors.Errorf("validateOptionalContentGroupIntent: invalid intent: %s", n.Value())
			}
		}

	default:
		return errors.New("validateOptionalContentGroupIntent: invalid type")
	}

	return nil
}

func validateOptionalContentGroupUsageDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	// see 8.11.4.4

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	dictName = "OCUsageDict"

	// CreatorInfo, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "CreatorInfo", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Language, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "Language", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Export, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "Export", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Zoom, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "Zoom", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Print, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "Print", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// View, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "View", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// User, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "User", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// PageElement, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "PageElement", OPTIONAL, sinceVersion, nil)

	return err
}

func validateOptionalContentGroupDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, sinceVersion pdf.Version) error {

	// see 8.11 Optional Content

	dictName := "optionalContentGroupDict"

	// Type, required, name, OCG
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", REQUIRED, sinceVersion, func(s string) bool { return s == "OCG" })
	if err != nil {
		return err
	}

	// Name, required, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Name", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Intent, optional, name or array
	err = validateOptionalContentGroupIntent(xRefTable, dict, dictName, "Intent", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// Usage, optional, usage dict
	return validateOptionalContentGroupUsageDict(xRefTable, dict, dictName, "Usage", OPTIONAL, sinceVersion)
}

func validateOptionalContentGroupArray(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName, dictEntry string, sinceVersion pdf.Version) error {

	arr, err := validateArrayEntry(xRefTable, dict, dictName, dictEntry, OPTIONAL, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	for _, v := range *arr {

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateOptionalContentGroupDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateOCGs(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName, entryName string, sinceVersion pdf.Version) error {

	// see 8.11.2.2

	obj, err := dict.Entry(dictName, entryName, OPTIONAL)
	if err != nil || obj == nil {
		return err
	}

	// Version check
	err = xRefTable.ValidateVersion("OCGs", sinceVersion)
	if err != nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	d, ok := obj.(pdf.PDFDict)
	if ok {
		return validateOptionalContentGroupDict(xRefTable, &d, sinceVersion)
	}

	return validateOptionalContentGroupArray(xRefTable, dict, dictName, entryName, sinceVersion)
}

func validateOptionalContentMembershipDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, sinceVersion pdf.Version) error {

	// see 8.11.2.2

	dictName := "OCMDict"

	// OCGs, optional, dict or array
	err := validateOCGs(xRefTable, dict, dictName, "OCGs", sinceVersion)
	if err != nil {
		return err
	}

	// P, optional, name
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"AllOn", "AnyOn", "AnyOff", "AllOff"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "P", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	// VE, optional, array, since V1.6
	_, err = validateArrayEntry(xRefTable, dict, dictName, "VE", OPTIONAL, pdf.V16, nil)

	return err
}

func validateOptionalContent(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	validate := func(s string) bool { return s == "OCG" || s == "OCMD" }
	t, err := validateNameEntry(xRefTable, d, "optionalContent", "Type", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	if *t == "OCG" {
		return validateOptionalContentGroupDict(xRefTable, d, sinceVersion)
	}

	return validateOptionalContentMembershipDict(xRefTable, d, sinceVersion)
}

func validateUsageApplicationDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, sinceVersion pdf.Version) error {

	dictName := "usageAppDict"

	// Event, required, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Event", REQUIRED, sinceVersion, func(s string) bool { return s == "View" || s == "Print" || s == "Export" })
	if err != nil {
		return err
	}

	// OCGs, optional, array of content groups
	err = validateOptionalContentGroupArray(xRefTable, dict, dictName, "OCGs", sinceVersion)
	if err != nil {
		return err
	}

	// Category, required, array of names
	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "Category", REQUIRED, sinceVersion, nil)

	return err
}

func validateUsageApplicationDictArray(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName, dictEntry string, required bool, sinceVersion pdf.Version) error {

	arr, err := validateArrayEntry(xRefTable, dict, dictName, dictEntry, required, sinceVersion, nil)
	if err != nil || arr == nil {
		return err
	}

	for _, v := range *arr {

		if v == nil {
			continue
		}

		var d *pdf.PDFDict

		d, err = xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateUsageApplicationDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateOptionalContentConfigurationDict(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, sinceVersion pdf.Version) error {

	dictName := "optContentConfigDict"

	// Name, optional, string
	_, err := validateStringEntry(xRefTable, dict, dictName, "Name", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Creator, optional, string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Creator", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BaseState, optional, name
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"ON", "OFF", "UNCHANGED"}) }
	baseState, err := validateNameEntry(xRefTable, dict, dictName, "BaseState", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	if baseState != nil {

		if baseState.String() != "ON" {
			// ON, optional, content group array
			err = validateOptionalContentGroupArray(xRefTable, dict, dictName, "ON", sinceVersion)
			if err != nil {
				return err
			}
		}

		if baseState.String() != "OFF" {
			// OFF, optional, content group array
			err = validateOptionalContentGroupArray(xRefTable, dict, dictName, "OFF", sinceVersion)
			if err != nil {
				return err
			}
		}

	}

	// Intent, optional, name or array
	err = validateOptionalContentGroupIntent(xRefTable, dict, dictName, "Intent", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// AS, optional, usage application dicts array
	err = validateUsageApplicationDictArray(xRefTable, dict, dictName, "AS", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// Order, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Order", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// ListMode, optional, name
	validate = func(s string) bool { return pdf.MemberOf(s, []string{"AllPages", "VisiblePages"}) }
	_, err = validateNameEntry(xRefTable, dict, dictName, "ListMode", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}

	// RBGroups, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "RBGroups", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Locked, optional, array
	return validateOptionalContentGroupArray(xRefTable, dict, dictName, "Locked", pdf.V16)
}

func validateOCProperties(xRefTable *pdf.XRefTable, rootDict *pdf.PDFDict, required bool, sinceVersion pdf.Version) error {

	// aka optional content properties dict.

	// => 8.11.4 Configuring Optional Content

	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		sinceVersion = pdf.V14
	}

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "OCProperties", required, sinceVersion, nil)
	if err != nil || dict == nil {
		return err
	}

	dictName := "optContentPropertiesDict"

	// "OCGs" required array of already written indRefs
	r := true
	if xRefTable.ValidationMode == pdf.ValidationRelaxed {
		r = false
	}
	_, err = validateIndRefArrayEntry(xRefTable, dict, dictName, "OCGs", r, sinceVersion, nil)
	if err != nil {
		return err
	}

	// "D" required dict, default viewing optional content configuration dict.
	d, err := validateDictEntry(xRefTable, dict, dictName, "D", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}
	err = validateOptionalContentConfigurationDict(xRefTable, d, sinceVersion)
	if err != nil {
		return err
	}

	// "Configs" optional array of alternate optional content configuration dicts.
	arr, err := validateArrayEntry(xRefTable, dict, dictName, "Configs", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if arr != nil {
		for _, value := range *arr {

			d, err = xRefTable.DereferenceDict(value)
			if err != nil {
				return err
			}

			if d == nil {
				continue
			}

			err = validateOptionalContentConfigurationDict(xRefTable, d, sinceVersion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

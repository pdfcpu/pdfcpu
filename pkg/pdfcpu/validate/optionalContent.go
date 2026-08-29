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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateOptionalContentGroupIntent(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (err error) {
	// see 8.11.2.1

	entryObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	validate := func(s string) bool {
		return s == "View" || s == "Design" || s == "All"
	}

	switch o := o.(type) {

	case types.Name:
		if !validate(o.Value()) {
			err = fmt.Errorf("%s.%s: invalid intent: %s", dictName, entryName, o.Value())
			return model.WithValidationErrorObject(err, entryObjNr)
		}

	case types.Array:

		for i, v := range o {
			intentObjNr := validationObjectNumber(entryObjNr, v)

			if v == nil {
				continue
			}

			o, err := xRefTable.Dereference(v)
			if err != nil {
				err = fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i, err)
				return model.WithValidationErrorObject(err, intentObjNr)
			}
			n, ok := o.(types.Name)
			if !ok {
				err = fmt.Errorf("%s.%s[%d]: invalid type", dictName, entryName, i)
				return model.WithValidationErrorObject(err, intentObjNr)
			}

			if !validate(n.Value()) {
				err = fmt.Errorf("%s.%s[%d]: invalid intent: %s", dictName, entryName, i, n.Value())
				return model.WithValidationErrorObject(err, intentObjNr)
			}
		}

	default:
		err = fmt.Errorf("%s.%s: %w", dictName, entryName, errors.New("invalid type"))
		return model.WithValidationErrorObject(err, entryObjNr)
	}

	return nil
}

func validateOptionalContentGroupUsageDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) (err error) {
	// see 8.11.4.4

	usageObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	d1, err := validateDictEntry(
		xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion, nil,
	)
	if err != nil || d1 == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	dictName = "OCUsageDict"
	defer func() {
		err = model.WithValidationErrorObject(err, usageObjNr)
	}()

	// CreatorInfo, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "CreatorInfo", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.CreatorInfo: %w", dictName, err)
	}

	// Language, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "Language", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Language: %w", dictName, err)
	}

	// Export, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "Export", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Export: %w", dictName, err)
	}

	// Zoom, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "Zoom", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Zoom: %w", dictName, err)
	}

	// Print, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "Print", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Print: %w", dictName, err)
	}

	// View, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "View", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.View: %w", dictName, err)
	}

	// User, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "User", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.User: %w", dictName, err)
	}

	// PageElement, optional, dict
	_, err = validateDictEntry(xRefTable, d1, usageObjNr, dictName, "PageElement", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.PageElement: %w", dictName, err)
	}
	return nil
}

func validateOptionalContentGroupDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	// see 8.11 Optional Content

	dictName := "optionalContentGroupDict"

	// Type, required, name, OCG
	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "Type", REQUIRED, sinceVersion, func(s string) bool { return s == "OCG" },
	)
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	// Name, required, text string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Name", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Name: %w", dictName, err)
	}

	// Intent, optional, name or array
	err = validateOptionalContentGroupIntent(
		xRefTable, d, ownerObjNr, dictName, "Intent", OPTIONAL, sinceVersion,
	)
	if err != nil {
		return fmt.Errorf("%s.Intent: %w", dictName, err)
	}

	// Usage, optional, usage dict
	return validateOptionalContentGroupUsageDict(
		xRefTable, d, ownerObjNr, dictName, "Usage", OPTIONAL, sinceVersion,
	)
}

func validateOptionalContentGroupArray(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, dictEntry string, sinceVersion model.Version) error {
	arrayObjNr := validationEntryObjectNumber(ownerObjNr, d, dictEntry)
	a, err := validateArrayEntry(
		xRefTable, d, ownerObjNr, dictName, dictEntry, OPTIONAL, sinceVersion, nil,
	)
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, dictEntry, err)
		}
		return nil
	}

	for i, v := range a {

		if v == nil {
			continue
		}

		groupObjNr := validationObjectNumber(arrayObjNr, v)
		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s: dereference optional content group dict: %w", objectContext(fmt.Sprintf("%s.%s[%d]", dictName, dictEntry, i), v), err)
			return model.WithValidationErrorObject(err, groupObjNr)
		}

		if d == nil {
			continue
		}

		err = validateOptionalContentGroupDict(xRefTable, d, groupObjNr, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("%s.%s[%d]", dictName, dictEntry, i), v), err)
		}

	}

	return nil
}

func validateOCGs(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, sinceVersion model.Version) error {
	// see 8.11.2.2

	o, _, err := d.Entry(dictName, entryName, OPTIONAL)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		}
		return nil
	}

	rawEntry := o
	entryObjNr := validationObjectNumber(ownerObjNr, rawEntry)

	// Version check
	err = xRefTable.ValidateVersion("OCGs", sinceVersion)
	if err != nil {
		err = fmt.Errorf("%s.%s: %w", dictName, entryName, err)
		return model.WithValidationErrorObject(err, entryObjNr)
	}

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		if err != nil {
			err = fmt.Errorf("%s: dereference: %w", objectContext(dictEntryContext(dictName, entryName, rawEntry), rawEntry), err)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		return nil
	}

	d1, ok := o.(types.Dict)
	if ok {
		if err := validateOptionalContentGroupDict(xRefTable, d1, entryObjNr, sinceVersion); err != nil {
			return fmt.Errorf("%s: %w", objectContext(dictEntryContext(dictName, entryName, rawEntry), rawEntry), err)
		}
		return nil
	}

	return validateOptionalContentGroupArray(xRefTable, d, ownerObjNr, dictName, entryName, sinceVersion)
}

func validateOptionalContentMembershipDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	// see 8.11.2.2

	dictName := "OCMDict"

	// OCGs, optional, dict or array
	err = validateOCGs(xRefTable, d, ownerObjNr, dictName, "OCGs", sinceVersion)
	if err != nil {
		return fmt.Errorf("%s.OCGs: %w", dictName, err)
	}

	// P, optional, name
	validate := func(s string) bool { return types.MemberOf(s, []string{"AllOn", "AnyOn", "AnyOff", "AllOff"}) }
	_, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "P", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return fmt.Errorf("%s.P: %w", dictName, err)
	}

	// VE, optional, array, since V1.6
	_, err = validateArrayEntry(xRefTable, d, ownerObjNr, dictName, "VE", OPTIONAL, model.V16, nil)
	if err != nil {
		return fmt.Errorf("%s.VE: %w", dictName, err)
	}
	return nil
}

func validateOptionalContent(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	rawEntry := d[entryName]
	entryObjNr := validationObjectNumber(0, rawEntry)
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil || d1 == nil {
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
		}
		return nil
	}

	validate := func(s string) bool { return s == "OCG" || s == "OCMD" }
	t, err := validateNameEntry(
		xRefTable, d1, entryObjNr, "optionalContent", "Type", REQUIRED, sinceVersion, validate,
	)
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}

	if *t == "OCG" {
		if err := validateOptionalContentGroupDict(xRefTable, d1, entryObjNr, sinceVersion); err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
		}
		return nil
	}

	if err := validateOptionalContentMembershipDict(xRefTable, d1, entryObjNr, sinceVersion); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}
	return nil
}

func validateUsageApplicationDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "usageAppDict"

	// Event, required, name
	_, err = validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "Event", REQUIRED, sinceVersion,
		func(s string) bool { return s == "View" || s == "Print" || s == "Export" },
	)
	if err != nil {
		return fmt.Errorf("%s.Event: %w", dictName, err)
	}

	// OCGs, optional, array of content groups
	err = validateOptionalContentGroupArray(xRefTable, d, ownerObjNr, dictName, "OCGs", sinceVersion)
	if err != nil {
		return fmt.Errorf("%s.OCGs: %w", dictName, err)
	}

	// Category, required, array of names
	_, err = validateNameArrayEntry(
		xRefTable, d, ownerObjNr, dictName, "Category", REQUIRED, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s.Category: %w", dictName, err)
	}
	return nil
}

func validateUsageApplicationDictArray(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, dictEntry string, required bool, sinceVersion model.Version) error {
	arrayObjNr := validationEntryObjectNumber(ownerObjNr, d, dictEntry)
	a, err := validateArrayEntry(
		xRefTable, d, ownerObjNr, dictName, dictEntry, required, sinceVersion, nil,
	)
	if err != nil || a == nil {
		if err != nil {
			return fmt.Errorf("%s.%s: %w", dictName, dictEntry, err)
		}
		return nil
	}

	for i, v := range a {

		if v == nil {
			continue
		}

		usageObjNr := validationObjectNumber(arrayObjNr, v)
		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s: dereference usage application dict: %w", objectContext(fmt.Sprintf("%s.%s[%d]", dictName, dictEntry, i), v), err)
			return model.WithValidationErrorObject(err, usageObjNr)
		}

		if d == nil {
			continue
		}

		err = validateUsageApplicationDict(xRefTable, d, usageObjNr, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("%s.%s[%d]", dictName, dictEntry, i), v), err)
		}

	}

	return nil
}

func validateOptionalContentConfigBaseState(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, sinceVersion model.Version) error {
	// BaseState, optional, name
	validate := func(s string) bool { return types.MemberOf(s, []string{"ON", "OFF", "UNCHANGED"}) }
	baseState, err := validateNameEntry(
		xRefTable, d, ownerObjNr, dictName, "BaseState", OPTIONAL, sinceVersion, validate,
	)
	if err != nil {
		return fmt.Errorf("%s.BaseState: %w", dictName, err)
	}

	if baseState != nil {

		if baseState.Value() != "ON" {
			// ON, optional, content group array
			err = validateOptionalContentGroupArray(xRefTable, d, ownerObjNr, dictName, "ON", sinceVersion)
			if err != nil {
				return fmt.Errorf("%s.ON: %w", dictName, err)
			}
		}

		if baseState.Value() != "OFF" {
			// OFF, optional, content group array
			err = validateOptionalContentGroupArray(xRefTable, d, ownerObjNr, dictName, "OFF", sinceVersion)
			if err != nil {
				return fmt.Errorf("%s.OFF: %w", dictName, err)
			}
		}

	}

	return nil
}

func validateOptionalContentConfigurationDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "optContentConfigDict"

	// Name, optional, string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Name", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Name: %w", dictName, err)
	}

	// Creator, optional, string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Creator", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Creator: %w", dictName, err)
	}

	if err := validateOptionalContentConfigBaseState(xRefTable, d, ownerObjNr, dictName, sinceVersion); err != nil {
		return err
	}

	// Intent, optional, name or array
	err = validateOptionalContentGroupIntent(
		xRefTable, d, ownerObjNr, dictName, "Intent", OPTIONAL, sinceVersion,
	)
	if err != nil {
		return fmt.Errorf("%s.Intent: %w", dictName, err)
	}

	// AS, optional, usage application dicts array
	err = validateUsageApplicationDictArray(
		xRefTable, d, ownerObjNr, dictName, "AS", OPTIONAL, sinceVersion,
	)
	if err != nil {
		return fmt.Errorf("%s.AS: %w", dictName, err)
	}

	// Order, optional, array
	_, err = validateArrayEntry(xRefTable, d, ownerObjNr, dictName, "Order", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.Order: %w", dictName, err)
	}

	// ListMode, optional, name
	validate := func(s string) bool { return types.MemberOf(s, []string{"AllPages", "VisiblePages"}) }
	_, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "ListMode", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return fmt.Errorf("%s.ListMode: %w", dictName, err)
	}

	// RBGroups, optional, array
	_, err = validateArrayEntry(xRefTable, d, ownerObjNr, dictName, "RBGroups", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.RBGroups: %w", dictName, err)
	}

	// Locked, optional, array
	sinceVersion = model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V15
	}
	if err := validateOptionalContentGroupArray(xRefTable, d, ownerObjNr, dictName, "Locked", sinceVersion); err != nil {
		return fmt.Errorf("%s.Locked: %w", dictName, err)
	}
	return nil
}

func validateOCPropertiesD(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, sinceVersion model.Version) error {
	required := REQUIRED
	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	if relaxed {
		required = OPTIONAL
	}

	dObjNr := validationEntryObjectNumber(ownerObjNr, d, "D")
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "D", required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.D: %w", dictName, err)
	}
	if d1 == nil {
		if relaxed {
			model.ShowDigestedSpecViolation("dict=" + dictName + " required entry=D missing")
		}
		return nil
	}
	if err = validateOptionalContentConfigurationDict(xRefTable, d1, dObjNr, sinceVersion); err != nil {
		return fmt.Errorf("%s.D: %w", dictName, err)
	}
	return nil
}

// validateOCProperties validates the optional content properties dictionary described in 8.11.4.
func validateOCProperties(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}

	rootObjNr := validationRootObjectNumber(xRefTable)
	ocPropertiesObjNr := validationEntryObjectNumber(rootObjNr, rootDict, "OCProperties")
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "rootDict", "OCProperties", required, sinceVersion, nil,
	)
	if err != nil || len(d) == 0 {
		if err != nil {
			return fmt.Errorf("rootDict.OCProperties: %w", err)
		}
		return nil
	}

	dictName := "optContentPropertiesDict"

	// "OCGs" required array of already written indRefs
	r := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		r = OPTIONAL
	}
	_, err = validateIndRefArrayEntry(xRefTable, d, 0, dictName, "OCGs", r, sinceVersion, nil)
	if err != nil {
		err = fmt.Errorf("%s.OCGs: %w", dictName, err)
		return model.WithValidationErrorObject(
			err, validationEntryObjectNumber(ocPropertiesObjNr, d, "OCGs"),
		)
	}

	// "D" required dict, default viewing optional content configuration dict.
	if err = validateOCPropertiesD(xRefTable, d, ocPropertiesObjNr, dictName, sinceVersion); err != nil {
		return err
	}

	// "Configs" optional array of alternate optional content configuration dicts.
	a, err := validateArrayEntry(
		xRefTable, d, ocPropertiesObjNr, dictName, "Configs", OPTIONAL, sinceVersion, nil,
	)
	if err != nil {
		return fmt.Errorf("%s.Configs: %w", dictName, err)
	}
	configsObjNr := validationEntryObjectNumber(ocPropertiesObjNr, d, "Configs")
	for i, o := range a {
		configObjNr := validationObjectNumber(configsObjNr, o)

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference optional content configuration dict: %w", objectContext(fmt.Sprintf("%s.Configs[%d]", dictName, i), o), err)
			return model.WithValidationErrorObject(err, configObjNr)
		}

		if d == nil {
			continue
		}

		err = validateOptionalContentConfigurationDict(xRefTable, d, configObjNr, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("%s.Configs[%d]", dictName, i), o), err)
		}

	}

	return nil
}

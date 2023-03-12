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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// func validateSignatureDict(xRefTable *model.XRefTable, o pdf.Object) error {
//
// 	d, err := xRefTable.DereferenceDict(o)
// 	if err != nil || d == nil {
// 		return err
// 	}
//
// 	// Type, optional, name
// 	_, err = validateNameEntry(xRefTable, d, "signatureDict", "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Sig" })
//
// 	// process signature dict fields.
//
// 	return err
// }

func validateAppearanceSubDict(xRefTable *model.XRefTable, d types.Dict) error {

	// dict of xobjects
	for _, o := range d {

		if xRefTable.ValidationMode == model.ValidationRelaxed {
			if d, ok := o.(types.Dict); ok && len(d) == 0 {
				continue
			}
		}

		err := validateXObjectStreamDict(xRefTable, o)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateAppearanceDictEntry(xRefTable *model.XRefTable, o types.Object) error {

	// stream or dict
	// single appearance stream or subdict

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Dict:
		err = validateAppearanceSubDict(xRefTable, o)

	case types.StreamDict:
		err = validateXObjectStreamDict(xRefTable, o)

	default:
		err = errors.New("pdfcpu: validateAppearanceDictEntry: unsupported PDF object")

	}

	return err
}

func validateAppearanceDict(xRefTable *model.XRefTable, o types.Object) error {

	// see 12.5.5 Appearance Streams

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Normal Appearance
	o, ok := d.Find("N")
	if !ok {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return errors.New("pdfcpu: validateAppearanceDict: missing required entry \"N\"")
		}
	} else {
		err = validateAppearanceDictEntry(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// Rollover Appearance
	if o, ok = d.Find("R"); ok {
		err = validateAppearanceDictEntry(xRefTable, o)
		if err != nil {
			return err
		}
	}

	// Down Appearance
	if o, ok = d.Find("D"); ok {
		err = validateAppearanceDictEntry(xRefTable, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateAcroFieldDictEntries(xRefTable *model.XRefTable, d types.Dict, terminalNode bool, inFieldType *types.Name) (outFieldType *types.Name, err error) {

	dictName := "acroFieldDict"

	// FT: name, Btn,Tx,Ch,Sig
	validate := func(s string) bool { return types.MemberOf(s, []string{"Btn", "Tx", "Ch", "Sig"}) }
	fieldType, err := validateNameEntry(xRefTable, d, dictName, "FT", terminalNode && inFieldType == nil, model.V10, validate)
	if err != nil {
		return nil, err
	}

	if fieldType != nil {
		outFieldType = fieldType
	}

	// Parent, required if this is a child in the field hierarchy.
	_, err = validateIndRefEntry(xRefTable, d, dictName, "Parent", OPTIONAL, model.V10)
	if err != nil {
		return nil, err
	}

	// T, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return nil, err
	}

	// TU, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, d, dictName, "TU", OPTIONAL, model.V13, nil)
	if err != nil {
		return nil, err
	}

	// TM, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, d, dictName, "TM", OPTIONAL, model.V13, nil)
	if err != nil {
		return nil, err
	}

	// Ff, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Ff", OPTIONAL, model.V10, nil)
	if err != nil {
		return nil, err
	}

	// V, optional, various
	_, err = validateEntry(xRefTable, d, dictName, "V", OPTIONAL, model.V10)
	if err != nil {
		return nil, err
	}

	// DV, optional, various
	_, err = validateEntry(xRefTable, d, dictName, "DV", OPTIONAL, model.V10)
	if err != nil {
		return nil, err
	}

	// AA, optional, dict, since V1.2
	err = validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, model.V12, "fieldOrAnnot")
	if err != nil {
		return nil, err
	}

	return outFieldType, nil
}

func validateAcroFieldParts(xRefTable *model.XRefTable, d types.Dict, inFieldType *types.Name) error {
	// dict represents a terminal field and must have Subtype "Widget"
	if _, err := validateNameEntry(xRefTable, d, "acroFieldDict", "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "Widget" }); err != nil {
		return err
	}

	// Validate field dict entries.
	if _, err := validateAcroFieldDictEntries(xRefTable, d, true, inFieldType); err != nil {
		return err
	}

	// Validate widget annotation - Validation of AA redundant because of merged acrofield with widget annotation.
	_, err := validateAnnotationDict(xRefTable, d)
	return err
}

func validateAcroFieldKid(xRefTable *model.XRefTable, d types.Dict, o types.Object, inFieldType *types.Name) error {
	var err error
	// dict represents a non terminal field.
	if d.Subtype() != nil && *d.Subtype() == "Widget" {
		return errors.New("pdfcpu: validateAcroFieldKid: non terminal field can not be widget annotation")
	}

	// Validate field entries.
	var xInFieldType *types.Name
	if xInFieldType, err = validateAcroFieldDictEntries(xRefTable, d, false, inFieldType); err != nil {
		return err
	}

	// Recurse over kids.
	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return err
	}

	for _, value := range a {
		ir, ok := value.(types.IndirectRef)
		if !ok {
			return errors.New("pdfcpu: validateAcroFieldKid: corrupt kids array: entries must be indirect reference")
		}
		valid, err := xRefTable.IsValid(ir)
		if err != nil {
			return err
		}

		if !valid {
			if err = validateAcroFieldDict(xRefTable, ir, xInFieldType); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateAcroFieldDict(xRefTable *model.XRefTable, ir types.IndirectRef, inFieldType *types.Name) error {
	d, err := xRefTable.DereferenceDict(ir)
	if err != nil || d == nil {
		return err
	}

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		if len(d) == 0 {
			return nil
		}
	}

	if err := xRefTable.SetValid(ir); err != nil {
		return err
	}

	if o, ok := d.Find("Kids"); ok {
		return validateAcroFieldKid(xRefTable, d, o, inFieldType)
	}

	return validateAcroFieldParts(xRefTable, d, inFieldType)
}

func validateAcroFormFields(xRefTable *model.XRefTable, o types.Object) error {

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || len(a) == 0 {
		return err
	}

	for _, value := range a {

		ir, ok := value.(types.IndirectRef)
		if !ok {
			return errors.New("pdfcpu: validateAcroFormFields: corrupt form field array entry")
		}

		valid, err := xRefTable.IsValid(ir)
		if err != nil {
			return err
		}

		if !valid {
			if err = validateAcroFieldDict(xRefTable, ir, nil); err != nil {
				return err
			}
		}

	}

	return nil
}

func validateAcroFormCO(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {

	// see 12.6.3 Trigger Events
	// Array of indRefs to field dicts with calculation actions, since V1.3

	// Version check
	err := xRefTable.ValidateVersion("AcroFormCO", sinceVersion)
	if err != nil {
		return err
	}

	return validateAcroFormFields(xRefTable, o)
}

func validateAcroFormXFA(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {

	// see 12.7.8

	o, ok := d.Find("XFA")
	if !ok {
		return nil
	}

	// streamDict or array of text,streamDict pairs

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.StreamDict:
		// no further processing

	case types.Array:

		i := 0

		for _, v := range o {

			if v == nil {
				return errors.New("pdfcpu: validateAcroFormXFA: array entry is nil")
			}

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return err
			}

			if i%2 == 0 {

				_, ok := o.(types.StringLiteral)
				if !ok {
					return errors.New("pdfcpu: validateAcroFormXFA: even array must be a string")
				}

			} else {

				_, ok := o.(types.StreamDict)
				if !ok {
					return errors.New("pdfcpu: validateAcroFormXFA: odd array entry must be a streamDict")
				}

			}

			i++
		}

	default:
		return errors.New("pdfcpu: validateAcroFormXFA: needs to be streamDict or array")
	}

	return xRefTable.ValidateVersion("AcroFormXFA", sinceVersion)
}

func validateQ(i int) bool { return i >= 0 && i <= 2 }

func validateAcroFormEntryCO(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {

	o, ok := d.Find("CO")
	if !ok {
		return nil
	}

	return validateAcroFormCO(xRefTable, o, sinceVersion)
}

func validateAcroFormEntryDR(xRefTable *model.XRefTable, d types.Dict) error {

	o, ok := d.Find("DR")
	if !ok {
		return nil
	}

	_, err := validateResourceDict(xRefTable, o)

	return err
}

func validateAcroForm(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {

	// => 12.7.2 Interactive Form Dictionary

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "AcroForm", OPTIONAL, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	// Version check
	err = xRefTable.ValidateVersion("AcroForm", sinceVersion)
	if err != nil {
		return err
	}

	// Fields, required, array of indirect references
	o, ok := d.Find("Fields")
	if !ok {
		rootDict.Delete("AcroForm")
		return nil
	}

	xRefTable.AcroForm = d

	err = validateAcroFormFields(xRefTable, o)
	if err != nil {
		return err
	}

	dictName := "acroFormDict"

	// NeedAppearances: optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "NeedAppearances", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// SigFlags: optional, since 1.3, integer
	sf, err := validateIntegerEntry(xRefTable, d, dictName, "SigFlags", OPTIONAL, model.V13, nil)
	if err != nil {
		return err
	}
	if sf != nil {
		i := sf.Value()
		xRefTable.SignatureExist = i&1 > 0
		xRefTable.AppendOnly = i&2 > 0
	}

	// CO: arra
	err = validateAcroFormEntryCO(xRefTable, d, model.V13)
	if err != nil {
		return err
	}

	// DR, optional, resource dict
	err = validateAcroFormEntryDR(xRefTable, d)
	if err != nil {
		return err
	}

	// DA: optional, string
	_, err = validateStringEntry(xRefTable, d, dictName, "DA", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Q: optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Q", OPTIONAL, model.V10, validateQ)
	if err != nil {
		return err
	}

	// XFA: optional, since 1.5, stream or array
	return validateAcroFormXFA(xRefTable, d, sinceVersion)
}

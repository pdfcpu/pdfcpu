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
	"strconv"
	"strings"

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

func validateDA(s string) bool {
	// A sequence of valid page-content graphics or text state operators.
	// At a minimum, the string shall include a Tf (text font) operator along with its two operands, font and size.
	da := strings.Fields(s)
	for i := 0; i < len(da); i++ {
		if da[i] == "Tf" {
			if i < 2 {
				return false
			}
			if da[i-2][0] != '/' {
				return false
			}
			fontID := da[i-2][1:]
			if len(fontID) == 0 {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-1], 64); err != nil {
				return false
			}
			continue
		}
		if da[i] == "rg" {
			if i < 3 {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-3], 32); err != nil {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-2], 32); err != nil {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-1], 32); err != nil {
				return false
			}
		}
		if da[i] == "g" {
			if i < 1 {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-1], 32); err != nil {
				return false
			}
		}
	}

	return true
}

func validateDARelaxed(s string) bool {
	// A sequence of valid page-content graphics or text state operators.
	// At a minimum, the string shall include a Tf (text font) operator along with its two operands, font and size.
	da := strings.Fields(s)
	for i := 0; i < len(da); i++ {
		if da[i] == "Tf" {
			if i < 2 {
				return false
			}
			if da[i-2][0] != '/' {
				return false
			}
			//fontID := da[i-2][1:]
			// if len(fontID) == 0 {
			// 	return false
			// }
			if _, err := strconv.ParseFloat(da[i-1], 64); err != nil {
				return false
			}
			continue
		}
		if da[i] == "rg" {
			if i < 3 {
				return false
			}
			if _, err := strconv.ParseFloat(strings.TrimPrefix(da[i-3], "["), 32); err != nil {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-2], 32); err != nil {
				return false
			}
			if _, err := strconv.ParseFloat(strings.TrimSuffix(da[i-1], "]"), 32); err != nil {
				return false
			}
		}
		if da[i] == "g" {
			if i < 1 {
				return false
			}
			if _, err := strconv.ParseFloat(da[i-1], 32); err != nil {
				return false
			}
		}
	}

	return true
}

func validateFormFieldDA(xRefTable *model.XRefTable, d types.Dict, dictName string, terminalNode bool, outFieldType *types.Name, requiresDA bool) (bool, error) {
	validate := validateDA
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		validate = validateDARelaxed
	}

	if outFieldType == nil || (*outFieldType).Value() == "Tx" {
		//if (*outFieldType).Value() == "Tx" {
		da, err := validateStringEntry(xRefTable, d, dictName, "DA", requiresDA, model.V10, validate)
		if err != nil {
			if !terminalNode && requiresDA {
				err = nil
			}
			return false, err
		}
		if xRefTable.ValidationMode == model.ValidationRelaxed && da != nil {
			// Repair DA
			d["DA"] = types.StringLiteral(*da)
		}

		return da != nil && *da != "", nil
	}

	return false, nil
}

func cacheSig(xRefTable *model.XRefTable, d types.Dict, dictName string, form bool, objNr, incr int) error {
	fieldType := d.NameEntry("FT")
	if fieldType == nil || *fieldType != "Sig" {
		return nil
	}

	sig := &model.Signature{Type: model.SigTypePage, ObjNr: objNr, Signed: d["V"] != nil, PageNr: xRefTable.CurPage}
	if form {
		sig.Type = model.SigTypeForm
	}

	var dts bool

	if indRef := d.IndirectRefEntry("V"); indRef != nil {
		sigDict, err := xRefTable.DereferenceDict(*indRef)
		if err != nil {
			return nil
		}
		if typ := sigDict.Type(); typ != nil {
			if *typ == "DocTimeStamp" {
				sig.Type = model.SigTypeDTS
				dts = true
			}
		}
	}

	// Rect is only required for terminal fields (fields without Kids).
	// Non-terminal fields (fields with Kids array) don't need Rect on the parent.
	// Per PDF spec ISO 32000-1:2008, Section 12.7.3.1 (Field Dictionaries):
	// - Terminal fields can be merged with widget annotations and must have Rect
	// - Non-terminal fields are parents in the field hierarchy and Rect is optional
	// Per PDF spec ISO 32000-1:2008, Section 12.5.6.19 (Widget Annotations):
	// - Widget annotations (terminal fields) require a Rect entry
	required := REQUIRED
	if _, hasKids := d.Find("Kids"); hasKids {
		required = OPTIONAL
	}

	arr, err := validateRectangleEntry(xRefTable, d, dictName, "Rect", required, model.V10, nil)
	if err != nil {
		return err
	}

	var r *types.Rectangle
	if arr != nil {
		r = types.RectForArray(arr)
		sig.Visible = r.Visible() && !dts
	}

	if _, ok := xRefTable.Signatures[incr]; !ok {
		xRefTable.Signatures[incr] = map[int]model.Signature{}
	}
	if sig1, ok := xRefTable.Signatures[incr][sig.ObjNr]; !ok {
		xRefTable.Signatures[incr][sig.ObjNr] = *sig
	} else {
		sig1.PageNr = xRefTable.CurPage
		xRefTable.Signatures[incr][sig.ObjNr] = sig1
	}

	return nil
}

func isTextField(ft *types.Name) bool {
	return ft != nil && *ft == "Tx"
}

func validateV(xRefTable *model.XRefTable, objNr, incr int, d types.Dict, dictName string, terminalNode, textField, oneKid bool) error {
	_, err := validateEntry(xRefTable, d, dictName, "V", OPTIONAL, model.V10)
	if err != nil {
		return err
	}
	// Ignore kids if V is present
	// if textField && v != nil && !terminalNode && !oneKid {
	// 	return errors.New("\"V\" not allowed in non terminal text fields with more than one kid")
	// }
	if err := cacheSig(xRefTable, d, dictName, true, objNr, incr); err != nil {
		return err
	}
	return nil
}

func validateDV(xRefTable *model.XRefTable, d types.Dict, dictName string, terminalNode, textField, oneKid bool) error {
	_, err := validateEntry(xRefTable, d, dictName, "DV", OPTIONAL, model.V10)
	if err != nil {
		return err
	}
	// Ignore kids if DV is present.
	// if textField && dv != nil && !terminalNode && !oneKid {
	// 	return errors.New("\"DV\" not allowed in non terminal text fields with more than one kid")
	// }
	return nil
}

func validateFormFieldDictEntries(xRefTable *model.XRefTable, objNr, incr int, d types.Dict, terminalNode, oneKid bool, inFieldType *types.Name, requiresDA bool) (outFieldType *types.Name, hasDA bool, err error) {

	dictName := "formFieldDict"

	// FT: name, Btn,Tx,Ch,Sig
	validate := func(s string) bool { return types.MemberOf(s, []string{"Btn", "Tx", "Ch", "Sig"}) }
	fieldType, err := validateNameEntry(xRefTable, d, dictName, "FT", terminalNode && inFieldType == nil, model.V10, validate)
	if err != nil {
		return nil, false, err
	}

	outFieldType = inFieldType
	if fieldType != nil {
		outFieldType = fieldType
	}

	textField := isTextField(outFieldType)

	// Parent, required if this is a child in the field hierarchy.
	_, err = validateIndRefEntry(xRefTable, d, dictName, "Parent", OPTIONAL, model.V10)
	if err != nil {
		return nil, false, err
	}

	// T, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return nil, false, err
	}

	// TU, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, d, dictName, "TU", OPTIONAL, model.V13, nil)
	if err != nil {
		return nil, false, err
	}

	// TM, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, d, dictName, "TM", OPTIONAL, model.V13, nil)
	if err != nil {
		return nil, false, err
	}

	// Ff, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Ff", OPTIONAL, model.V10, nil)
	if err != nil {
		return nil, false, err
	}

	// V, optional, various
	if err := validateV(xRefTable, objNr, incr, d, dictName, terminalNode, textField, oneKid); err != nil {
		return nil, false, err
	}

	// DV, optional, various
	if err := validateDV(xRefTable, d, dictName, terminalNode, textField, oneKid); err != nil {
		return nil, false, err
	}

	// AA, optional, dict, since V1.2
	err = validateAdditionalActions(xRefTable, d, dictName, "AA", OPTIONAL, model.V12, "fieldOrAnnot")
	if err != nil {
		return nil, false, err
	}

	// DA, required for text fields, since ?
	// The default appearance string containing a sequence of valid page-content graphics or text state operators that define such properties as the field’s text size and colour.
	hasDA, err = validateFormFieldDA(xRefTable, d, dictName, terminalNode, outFieldType, requiresDA)

	return outFieldType, hasDA, err
}

func validateFormFieldParts(xRefTable *model.XRefTable, objNr, incr int, d types.Dict, inFieldType *types.Name, requiresDA bool) error {
	// dict represents a terminal field and must have Subtype "Widget"
	if _, err := validateNameEntry(xRefTable, d, "formFieldDict", "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "Widget" }); err != nil {
		d["Subtype"] = types.Name("Widget")
	}

	// Validate field dict entries.
	if _, _, err := validateFormFieldDictEntries(xRefTable, objNr, incr, d, true, false, inFieldType, requiresDA); err != nil {
		return err
	}

	// Validate widget annotation - Validation of AA redundant because of merged acrofield with widget annotation.
	_, err := validateAnnotationDict(xRefTable, d)
	return err
}

func validateFormFieldKids(xRefTable *model.XRefTable, objNr, incr int, d types.Dict, o types.Object, inFieldType *types.Name, requiresDA bool) error {
	var err error
	// dict represents a non terminal field.
	if d.Subtype() != nil && *d.Subtype() == "Widget" {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return errors.New("pdfcpu: validateFormFieldKids: non terminal field can not be widget annotation")
		}
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return err
	}

	// Validate field entries.
	var xInFieldType *types.Name
	var hasDA bool
	if xInFieldType, hasDA, err = validateFormFieldDictEntries(xRefTable, objNr, incr, d, false, len(a) == 1, inFieldType, requiresDA); err != nil {
		return err
	}
	if requiresDA && hasDA {
		requiresDA = false
	}

	if len(a) == 0 {
		return nil
	}

	// Recurse over kids.
	for _, value := range a {
		ir, ok := value.(types.IndirectRef)
		if !ok {
			return errors.New("pdfcpu: validateFormFieldKids: corrupt kids array: entries must be indirect reference")
		}
		valid, err := xRefTable.IsValid(ir)
		if err != nil {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return err
			}
			model.ShowSkipped(fmt.Sprintf("missing form field kid obj #%s", ir.ObjectNumber.String()))
			valid = true
		}

		if !valid {
			if err = validateFormFieldDict(xRefTable, ir, xInFieldType, requiresDA); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateFormFieldDict(xRefTable *model.XRefTable, ir types.IndirectRef, inFieldType *types.Name, requiresDA bool) error {
	d, incr, err := xRefTable.DereferenceDictWithIncr(ir)
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

	objNr := ir.ObjectNumber.Value()

	if o, ok := d.Find("Kids"); ok {
		return validateFormFieldKids(xRefTable, objNr, incr, d, o, inFieldType, requiresDA)
	}

	return validateFormFieldParts(xRefTable, objNr, incr, d, inFieldType, requiresDA)
}

func validateFormFields(xRefTable *model.XRefTable, arr types.Array, requiresDA bool) error {

	for _, value := range arr {

		ir, ok := value.(types.IndirectRef)
		if !ok {
			return errors.New("pdfcpu: validateFormFields: corrupt form field array entry")
		}

		valid, err := xRefTable.IsValid(ir)
		if err != nil {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return err
			}
			model.ShowSkipped(fmt.Sprintf("missing form field obj #%s", ir.ObjectNumber.String()))
			valid = true
		}

		if !valid {
			if err = validateFormFieldDict(xRefTable, ir, nil, requiresDA); err != nil {
				return err
			}
		}

	}

	return nil
}

func validateFormCO(xRefTable *model.XRefTable, arr types.Array, sinceVersion model.Version, requiresDA bool) error {

	// see 12.6.3 Trigger Events
	// Array of indRefs to field dicts with calculation actions, since V1.3

	// Version check
	err := xRefTable.ValidateVersion("AcroFormCO", sinceVersion)
	if err != nil {
		return err
	}

	return validateFormFields(xRefTable, arr, requiresDA)
}

func validateFormXFA(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {

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
				return errors.New("pdfcpu: validateFormXFA: array entry is nil")
			}

			o, err := xRefTable.Dereference(v)
			if err != nil {
				return err
			}

			if i%2 == 0 {

				_, ok := o.(types.StringLiteral)
				if !ok {
					return errors.New("pdfcpu: validateFormXFA: even array must be a string")
				}

			} else {

				_, ok := o.(types.StreamDict)
				if !ok {
					return errors.New("pdfcpu: validateFormXFA: odd array entry must be a streamDict")
				}

			}

			i++
		}

	default:
		return errors.New("pdfcpu: validateFormXFA: needs to be streamDict or array")
	}

	return xRefTable.ValidateVersion("AcroFormXFA", sinceVersion)
}

func validateQ(i int) bool { return i >= 0 && i <= 2 }

func validateFormEntryCO(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version, requiresDA bool) error {

	o, ok := d.Find("CO")
	if !ok {
		return nil
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil || len(arr) == 0 {
		return err
	}

	return validateFormCO(xRefTable, arr, sinceVersion, requiresDA)
}

func validateFormEntryDR(xRefTable *model.XRefTable, d types.Dict) error {

	o, ok := d.Find("DR")
	if !ok {
		return nil
	}

	_, err := validateResourceDict(xRefTable, o)

	return err
}

func validateFormEntries(xRefTable *model.XRefTable, d types.Dict, dictName string, requiresDA bool, sinceVersion model.Version) error {
	// NeedAppearances: optional, boolean
	_, err := validateBooleanEntry(xRefTable, d, dictName, "NeedAppearances", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// SigFlags: optional, since 1.3, integer
	sinceV := model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceV = model.V12
	}
	sf, err := validateIntegerEntry(xRefTable, d, dictName, "SigFlags", OPTIONAL, sinceV, nil)
	if err != nil {
		return err
	}
	if sf != nil {
		i := sf.Value()
		xRefTable.SignatureExist = i&1 > 0
		xRefTable.AppendOnly = i&2 > 0
	}

	// CO: array
	err = validateFormEntryCO(xRefTable, d, model.V13, requiresDA)
	if err != nil {
		return err
	}

	// DR, optional, resource dict
	err = validateFormEntryDR(xRefTable, d)
	if err != nil {
		return err
	}

	// Q: optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "Q", OPTIONAL, model.V10, validateQ)
	if err != nil {
		return err
	}

	// XFA: optional, since 1.5, stream or array
	return validateFormXFA(xRefTable, d, sinceVersion)
}

func validateForm(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {

	// => 12.7.2 Interactive Form Dictionary

	d, err := validateDictEntry(xRefTable, rootDict, "rootDict", "AcroForm", OPTIONAL, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	// Version check
	if err = xRefTable.ValidateVersion("AcroForm", sinceVersion); err != nil {
		return err
	}

	// Fields, required, array of indirect references
	o, ok := d.Find("Fields")
	if !ok {
		// Fix empty AcroForm dict.
		rootDict.Delete("AcroForm")
		return nil
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arr) == 0 {
		// Fix empty AcroForm dict.
		rootDict.Delete("AcroForm")
		return nil
	}

	xRefTable.Form = d

	dictName := "acroFormDict"

	// DA: optional, string
	validate := validateDA
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		validate = validateDARelaxed
	}
	da, err := validateStringEntry(xRefTable, d, dictName, "DA", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}
	if xRefTable.ValidationMode == model.ValidationRelaxed && da != nil {
		// Repair
		d["DA"] = types.StringLiteral(*da)
	}

	requiresDA := da == nil || len(*da) == 0

	err = validateFormFields(xRefTable, arr, requiresDA)
	if err != nil {
		return err
	}

	return validateFormEntries(xRefTable, d, dictName, requiresDA, sinceVersion)
}

func locateAnnForAPAndRect(d types.Dict, r *types.Rectangle, pageAnnots map[int]model.PgAnnots) *types.IndirectRef {
	if indRef1 := d.IndirectRefEntry("AP"); indRef1 != nil {
		apObjNr := indRef1.ObjectNumber.Value()
		for _, m := range pageAnnots {
			annots, ok := m[model.AnnWidget]
			if ok {
				for objNr, annRend := range annots.Map {
					if objNr > 0 {
						if annRend.RectString() == r.ShortString() && annRend.APObjNrInt() == apObjNr {
							return types.NewIndirectRef(objNr, 0)
						}
					}
				}
			}
		}
	}
	return nil
}

func pageAnnotIndRefForAcroField(xRefTable *model.XRefTable, indRef types.IndirectRef) (*types.IndirectRef, error) {

	// indRef should be part of a page annotation dict.

	for _, m := range xRefTable.PageAnnots {
		annots, ok := m[model.AnnWidget]
		if ok {
			for _, ir := range *annots.IndRefs {
				if ir == indRef {
					return &ir, nil
				}
			}
		}
	}

	// form field is duplicated, retrieve corresponding page annotation for Rect, AP

	d, err := xRefTable.DereferenceDict(indRef)
	if err != nil {
		return nil, err
	}

	arr, err := xRefTable.DereferenceArray(d["Rect"])
	if err != nil {
		return nil, err
	}
	if arr == nil {
		// Assumption: There are kids and the kids are allright.
		return &indRef, nil
	}

	r, err := xRefTable.RectForArray(arr)
	if err != nil {
		return nil, err
	}

	// Possible orphan sig field dicts.
	if ft := d.NameEntry("FT"); ft != nil && *ft == "Sig" {
		// Signature Field
		if _, ok := d.Find("V"); !ok {
			// without linked sig dict (unsigned)
			return &indRef, nil
		}
		// signed but invisible
		if !r.Visible() {
			return &indRef, nil
		}
	}

	if indRef := locateAnnForAPAndRect(d, r, xRefTable.PageAnnots); indRef != nil {
		return indRef, nil
	}

	return &indRef, nil
	//return nil, errors.Errorf("pdfcpu: can't repair form field: %d\n", indRef.ObjectNumber.Value())
}

func fixFormFieldsArray(xRefTable *model.XRefTable, arr types.Array) (types.Array, error) {
	arr1 := types.Array{}
	for _, obj := range arr {
		indRef, err := pageAnnotIndRefForAcroField(xRefTable, obj.(types.IndirectRef))
		if err != nil {
			return nil, err
		}
		arr1 = append(arr1, *indRef)
	}
	return arr1, nil
}

func validateFormFieldsAgainstPageAnnotations(xRefTable *model.XRefTable) error {
	o, found := xRefTable.Form.Find("Fields")
	if !found {
		return nil
	}

	indRef, ok := o.(types.IndirectRef)
	if !ok {
		arr, ok := o.(types.Array)
		if !ok {
			return errors.New("pdfcpu: invalid array object")
		}
		arr, err := fixFormFieldsArray(xRefTable, arr)
		if err != nil {
			return err
		}
		indRef, err := xRefTable.IndRefForNewObject(arr)
		if err != nil {
			return err
		}
		xRefTable.Form["Fields"] = *indRef
		return nil
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return err
	}
	arr, err = fixFormFieldsArray(xRefTable, arr)
	if err != nil {
		return err
	}
	entry, _ := xRefTable.FindTableEntryForIndRef(&indRef)
	entry.Object = arr

	return nil
}

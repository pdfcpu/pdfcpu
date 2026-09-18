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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// func validateSignatureDict(xRefTable *model.XRefTable, o pdf.Object) error {
//
// 	d, err := xRefTable.DereferenceDict(o)
// 	if err != nil || d == nil {
// 		return err
// 	}
//
// 	// Type, optional, name
// 	_, err = validateNameEntry(
// 		xRefTable, d, "signatureDict", "Type", OPTIONAL, model.V10, func(s string) bool { return s == "Sig" },
// 	)
//
// 	// process signature dict fields.
//
// 	return err
// }

func validateAppearanceSubDict(c context.Context, xRefTable *model.XRefTable, d types.Dict) error {
	// dict of xobjects
	for _, key := range slices.Sorted(maps.Keys(d)) {
		o := d[key]

		if xRefTable.ValidationMode == model.ValidationRelaxed {
			if d, ok := o.(types.Dict); ok && len(d) == 0 {
				continue
			}
		}

		err := validateXObjectStreamDict(c, xRefTable, o)
		if err != nil {
			return fmt.Errorf("appearance subdict entry %s: %w", key, err)
		}

	}

	return nil
}

func validateAppearanceDictEntry(c context.Context, xRefTable *model.XRefTable, o types.Object) error {
	// stream or dict
	// single appearance stream or subdict

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Dict:
		err = validateAppearanceSubDict(c, xRefTable, o)

	case types.StreamDict:
		err = validateXObjectStreamDict(c, xRefTable, o)

	default:
		err = errUnsupportedPDFObject

	}

	return err
}

func validateAppearanceEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, entryName string) error {
	o, ok := d.Find(entryName)
	if !ok {
		return nil
	}

	err := validateAppearanceDictEntry(c, xRefTable, o)
	if err == nil || xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}

	d.Delete(entryName)
	model.ShowSkipped(fmt.Sprintf("corrupt appearance %s: %v", entryName, err))
	return nil
}

func validateAppearanceDict(c context.Context, xRefTable *model.XRefTable, o types.Object) error {
	// see 12.5.5 Appearance Streams

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Normal Appearance
	_, ok := d.Find("N")
	if !ok {
		if xRefTable.ValidationMode == model.ValidationStrict {
			logMissingRequiredEntry("appearanceDict", "N", d)
			return missingRequiredEntryError("appearanceDict", "N", "add normal appearance stream/subdict or validate in relaxed mode")
		}
	} else if err = validateAppearanceEntry(c, xRefTable, d, "N"); err != nil {
		return err
	}

	// Rollover Appearance
	if err = validateAppearanceEntry(c, xRefTable, d, "R"); err != nil {
		return err
	}

	// Down Appearance
	return validateAppearanceEntry(c, xRefTable, d, "D")
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
		required := requiresDA
		if terminalNode && outFieldType == nil && xRefTable.ValidationMode == model.ValidationRelaxed {
			required = OPTIONAL
		}
		da, err := validateStringEntry(xRefTable, d, 0, dictName, "DA", required, model.V10, validate)
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

func detectRectArray(xRefTable *model.XRefTable, d types.Dict, dictName string) (types.Array, error) {
	obj, ok := d.Find("Kids")
	if !ok {
		// terminal field
		return validateRectangleEntry(xRefTable, d, 0, dictName, "Rect", REQUIRED, model.V10, nil)
	}

	// non terminal field
	kids, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return nil, fmt.Errorf("form field Kids: dereference array: %w", err)
	}
	if len(kids) == 0 {
		return nil, errors.New("form field Kids: empty array")
	}

	d1, err := xRefTable.DereferenceDict(kids[0])
	if err != nil {
		return nil, fmt.Errorf("form field Kids[0]: dereference dict: %w", err)
	}

	return validateRectangleEntry(xRefTable, d1, 0, dictName, "Rect", REQUIRED, model.V10, nil)
}

func cacheSig(xRefTable *model.XRefTable, d types.Dict, dictName string, form bool, objNr, incr int) error {
	ft, _, err := xRefTable.DereferenceNameEntry(d, "FT")
	if err != nil {
		return fmt.Errorf("%s.FT: %w", dictName, err)
	}
	if ft == nil || ft.Value() != "Sig" {
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
		// The signature dictionary determines the revision, even when its field was updated later.
		incr = indirectObjectIncrement(xRefTable, *indRef, incr)
		typ, _, err := xRefTable.DereferenceNameEntry(sigDict, "Type")
		if err != nil {
			return fmt.Errorf("signature dict Type: %w", err)
		}
		if typ != nil && typ.Value() == "DocTimeStamp" {
			sig.Type = model.SigTypeDTS
			dts = true
		}
	}

	arr, err := detectRectArray(xRefTable, d, dictName)
	if err != nil {
		return err
	}

	r, err := xRefTable.RectForArray(arr)
	if err != nil {
		return fmt.Errorf("%s.Rect: %w", dictName, err)
	}
	sig.Visible = r.Visible() && !dts

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
	_, err := validateEntry(xRefTable, d, 0, dictName, "V", OPTIONAL, model.V10)
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
	_, err := validateEntry(xRefTable, d, 0, dictName, "DV", OPTIONAL, model.V10)
	if err != nil {
		return err
	}
	// Ignore kids if DV is present.
	// if textField && dv != nil && !terminalNode && !oneKid {
	// 	return errors.New("\"DV\" not allowed in non terminal text fields with more than one kid")
	// }
	return nil
}

func validateFormFieldType(xRefTable *model.XRefTable) func(string) bool {
	return func(s string) bool {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			return true
		}
		return types.MemberOf(s, []string{"Btn", "Tx", "Ch", "Sig"})
	}
}

func validateFormFieldDictEntries(c context.Context, xRefTable *model.XRefTable, objNr, incr int, d types.Dict, terminalNode, oneKid bool, inFieldType *types.Name, requiresDA bool) (outFieldType *types.Name, hasDA bool, err error) {
	dictName := "formFieldDict"

	// FT: name, Btn,Tx,Ch,Sig
	required := terminalNode && inFieldType == nil
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}
	fieldType, err := validateNameEntry(xRefTable, d, 0, dictName, "FT", required, model.V10, validateFormFieldType(xRefTable))
	if err != nil {
		return nil, false, err
	}

	outFieldType = inFieldType
	if fieldType != nil {
		outFieldType = fieldType
	}

	textField := isTextField(outFieldType)

	// Parent, required if this is a child in the field hierarchy.
	_, err = validateIndRefEntry(xRefTable, d, 0, dictName, "Parent", OPTIONAL, model.V10)
	if err != nil {
		return nil, false, err
	}

	// T, optional, text string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return nil, false, err
	}

	// TU, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "TU", OPTIONAL, model.V13, nil)
	if err != nil {
		return nil, false, err
	}

	// TM, optional, text string, since V1.3
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "TM", OPTIONAL, model.V13, nil)
	if err != nil {
		return nil, false, err
	}

	// Ff, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "Ff", OPTIONAL, model.V10, nil)
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
	err = validateAdditionalActions(c, xRefTable, d, dictName, "AA", OPTIONAL, model.V12, "fieldOrAnnot")
	if err != nil {
		return nil, false, err
	}

	// DA, required for text fields, since ?
	// The default appearance string contains valid page-content graphics or text-state operators
	// that define properties such as the field's text size and colour.
	hasDA, err = validateFormFieldDA(xRefTable, d, dictName, terminalNode, outFieldType, requiresDA)

	return outFieldType, hasDA, err
}

func validateFormFieldParts(c context.Context, xRefTable *model.XRefTable, objNr, incr int, d types.Dict, inFieldType *types.Name, requiresDA bool) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// dict represents a terminal field and must have Subtype "Widget"
	if _, err := validateNameEntry(xRefTable, d, 0, "formFieldDict", "Subtype", REQUIRED, model.V10, func(s string) bool { return s == "Widget" }); err != nil {
		d["Subtype"] = types.Name("Widget")
	}

	// Validate field dict entries.
	fieldType, _, err := validateFormFieldDictEntries(c, xRefTable, objNr, incr, d, true, false, inFieldType, requiresDA)
	if err != nil {
		return err
	}
	if fieldType == nil {
		return errors.New("form field: missing effective field type")
	}

	// Validate widget annotation - Validation of AA redundant because of merged acrofield with widget annotation.
	if _, err = validateAnnotationDict(c, xRefTable, d, objNr); err != nil {
		return err
	}

	return nil
}

func formFieldCycleError(err error) error {
	if errors.Is(err, model.ErrFormFieldCycle) {
		return model.ErrFormFieldCycle
	}
	return err
}

func formFieldKidsDereferenceError(err error, fieldObjNr, kidsObjNr int) error {
	context := "form field"
	if kidsObjNr != fieldObjNr {
		context = fmt.Sprintf("form field obj#%d", fieldObjNr)
	}
	err = fmt.Errorf("%s: dereference Kids array: %w", context, err)
	return model.WithValidationErrorObject(err, kidsObjNr)
}

func formFieldKidsElementError(err error, fieldObjNr, kidsObjNr, index int) error {
	context := "form field"
	if kidsObjNr != fieldObjNr {
		context = fmt.Sprintf("form field obj#%d", fieldObjNr)
	}
	err = fmt.Errorf("%s Kids[%d]: %w", context, index, err)
	return model.WithValidationErrorObject(err, kidsObjNr)
}

func validateNonTerminalFieldSubtype(xRefTable *model.XRefTable, d types.Dict) error {
	st, _, err := xRefTable.DereferenceNameEntry(d, "Subtype")
	if err != nil {
		return fmt.Errorf("form field Subtype: %w", err)
	}
	if st != nil && st.Value() == "Widget" && xRefTable.ValidationMode == model.ValidationStrict {
		return errors.New("form field: non-terminal field cannot be widget annotation")
	}
	return nil
}

func validateFormFieldKids(c context.Context, xRefTable *model.XRefTable, objNr, incr int, d types.Dict, o types.Object, inFieldType *types.Name, requiresDA bool, depth int, visit *model.FormFieldVisit, specViolations *[]error) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// dict represents a non terminal field.
	if err := validateNonTerminalFieldSubtype(xRefTable, d); err != nil {
		return err
	}

	kidsObjNr := validationObjectNumber(objNr, o)
	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return formFieldKidsDereferenceError(err, objNr, kidsObjNr)
	}

	// Validate field entries.
	var xInFieldType *types.Name
	var hasDA bool
	if xInFieldType, hasDA, err = validateFormFieldDictEntries(c, xRefTable, objNr, incr, d, false, len(a) == 1, inFieldType, requiresDA); err != nil {
		return err
	}
	if requiresDA && hasDA {
		requiresDA = false
	}

	if len(a) == 0 {
		return nil
	}

	// Recurse over kids.
	for i, value := range a {
		ir, ok := value.(types.IndirectRef)
		if !ok {
			err = fmt.Errorf("expected indirect reference, got %T", value)
			return formFieldKidsElementError(err, objNr, kidsObjNr, i)
		}
		kidObjNr := ir.ObjectNumber.Value()
		if err := visit.Check(ir.ObjectNumber.Value()); err != nil {
			err = formFieldCycleError(err)
			err = fmt.Errorf("form field obj#%d Kids[%d] obj#%d: %w", objNr, i, kidObjNr, err)
			return model.WithValidationErrorObject(err, kidObjNr)
		}
		valid, err := xRefTable.IsValid(ir)
		if err != nil {
			if xRefTable.ValidationMode == model.ValidationStrict {
				err = fmt.Errorf("form field obj#%d Kids[%d] obj#%d: check valid: %w", objNr, i, kidObjNr, err)
				return model.WithValidationErrorObject(err, kidObjNr)
			}
			err = fmt.Errorf("form field obj#%d Kids[%d] obj#%d: check valid: %w", objNr, i, kidObjNr, err)
			*specViolations = append(*specViolations, err)
			valid = true
		}

		if !valid {
			if err = validateFormFieldDictDepth(
				c,
				xRefTable,
				ir,
				xInFieldType,
				requiresDA,
				depth+1,
				visit,
				specViolations,
			); err != nil {
				context := fmt.Sprintf("form field obj#%d Kids[%d] obj#%d", objNr, i, kidObjNr)
				err = model.WrapRecursionError(context, err)
				return model.WithValidationErrorObject(err, kidObjNr)
			}
		}
	}

	return nil
}

func validateFormFieldDictDepth(c context.Context, xRefTable *model.XRefTable, ir types.IndirectRef, inFieldType *types.Name, requiresDA bool, depth int, visit *model.FormFieldVisit, specViolations *[]error) (err error) {
	objNr := ir.ObjectNumber.Value()
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if err := xRefTable.CheckRecursionDepth("form field tree", depth); err != nil {
		return err
	}
	if err := visit.Enter(objNr); err != nil {
		return fmt.Errorf("form field obj#%d: %w", objNr, formFieldCycleError(err))
	}
	defer visit.Leave(objNr)

	d, incr, err := xRefTable.DereferenceDictWithIncr(ir)
	if err != nil {
		return fmt.Errorf("form field: dereference dict: %w", err)
	}
	if d == nil {
		err = errors.New("form field: missing dict")
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			*specViolations = append(*specViolations, model.WithValidationErrorObject(err, objNr))
			return nil
		}
		return err
	}

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		if len(d) == 0 {
			return nil
		}
	}

	if err := xRefTable.SetValid(ir); err != nil {
		return fmt.Errorf("form field obj#%d: mark valid: %w", objNr, err)
	}

	if o, ok := d.Find("Kids"); ok {
		return validateFormFieldKids(
			c,
			xRefTable,
			objNr,
			incr,
			d,
			o,
			inFieldType,
			requiresDA,
			depth,
			visit,
			specViolations,
		)
	}

	return validateFormFieldParts(c, xRefTable, objNr, incr, d, inFieldType, requiresDA)
}

func acroFormFieldError(err error, index, fieldObjNr int) error {
	context := fmt.Sprintf("Fields[%d]", index)
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) && validationErr.ObjectNumber() != fieldObjNr {
		context = fmt.Sprintf("%s obj#%d", context, fieldObjNr)
	}
	err = fmt.Errorf("%s: %w", context, err)
	return model.WithValidationErrorObject(err, fieldObjNr)
}

func nonWidgetAnnotation(xRefTable *model.XRefTable, d types.Dict) bool {
	t, _, err := xRefTable.DereferenceNameEntry(d, "Type")
	if err != nil || t == nil || t.Value() != "Annot" {
		return false
	}
	st, _, err := xRefTable.DereferenceNameEntry(d, "Subtype")
	if err != nil || st == nil || st.Value() == "Widget" {
		return false
	}
	ft, _, err := xRefTable.DereferenceNameEntry(d, "FT")
	if err != nil || ft != nil {
		return false
	}
	_, hasKids := d.Find("Kids")
	return !hasKids
}

func removeNonWidgetAnnotationsFromFormFields(xRefTable *model.XRefTable, arr types.Array) (types.Array, bool) {
	cleaned := types.Array{}
	removed := false

	for _, value := range arr {
		ir, ok := value.(types.IndirectRef)
		if !ok {
			cleaned = append(cleaned, value)
			continue
		}

		d, err := xRefTable.DereferenceDict(ir)
		if err != nil || d == nil || !nonWidgetAnnotation(xRefTable, d) {
			cleaned = append(cleaned, value)
			continue
		}

		model.ShowMsg(fmt.Sprintf("removed non-widget annotation from AcroForm Fields (object #%d)", ir.ObjectNumber))
		removed = true
	}

	return cleaned, removed
}

func validateFormFields(c context.Context, xRefTable *model.XRefTable, arr types.Array, ownerObjNr int, requiresDA bool) error {
	var specViolations []error

	for i, value := range arr {

		ir, ok := value.(types.IndirectRef)
		if !ok {
			err := fmt.Errorf("Fields[%d]: expected indirect reference, got %T", i, value)
			return model.WithValidationErrorObject(err, ownerObjNr)
		}
		fieldObjNr := ir.ObjectNumber.Value()

		valid, err := xRefTable.IsValid(ir)
		if err != nil {
			if xRefTable.ValidationMode == model.ValidationStrict {
				err = fmt.Errorf("Fields[%d] obj#%d: check valid: %w", i, fieldObjNr, err)
				return model.WithValidationErrorObject(err, fieldObjNr)
			}
			err = fmt.Errorf("Fields[%d] obj#%d: check valid: %w", i, fieldObjNr, err)
			specViolations = append(specViolations, err)
			valid = true
		}

		if !valid {
			if err = validateFormFieldDictDepth(
				c,
				xRefTable,
				ir,
				nil,
				requiresDA,
				0,
				model.NewFormFieldVisit(),
				&specViolations,
			); err != nil {
				return acroFormFieldError(err, i, fieldObjNr)
			}
		}

	}

	showDigestedSpecViolations(specViolations)
	return nil
}

func validateFormCO(c context.Context, xRefTable *model.XRefTable, arr types.Array, ownerObjNr int, sinceVersion model.Version, requiresDA bool) error {
	// see 12.6.3 Trigger Events
	// Array of indRefs to field dicts with calculation actions, since V1.3

	// Version check
	err := xRefTable.ValidateVersion("AcroFormCO", sinceVersion)
	if err != nil {
		return err
	}

	return validateFormFields(c, xRefTable, arr, ownerObjNr, requiresDA)
}

func validateFormXFAArray(xRefTable *model.XRefTable, a types.Array, objNr int) error {
	// see 12.7.8
	if err := validateArrayPairs(a, objNr, "AcroForm", "XFA", 1); err != nil {
		return err
	}
	for i, v := range a {
		entryObjNr := validationObjectNumber(objNr, v)
		if v == nil {
			err := fmt.Errorf("AcroForm XFA[%d]: missing entry", i)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		o, err := xRefTable.Dereference(v)
		if err != nil {
			err = fmt.Errorf("AcroForm XFA[%d]: dereference: %w", i, err)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
		if i%2 == 0 {
			if _, err := types.StringOrHexLiteral(o); err != nil {
				err = fmt.Errorf("AcroForm XFA[%d]: expected string", i)
				return model.WithValidationErrorObject(err, entryObjNr)
			}
			continue
		}
		if _, ok := o.(types.StreamDict); !ok {
			err = fmt.Errorf("AcroForm XFA[%d]: expected stream dict", i)
			return model.WithValidationErrorObject(err, entryObjNr)
		}
	}
	return nil
}

func validateFormXFA(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see 12.7.8
	rawObject, ok := d.Find("XFA")
	if !ok {
		return nil
	}
	objNr := validationObjectNumber(0, rawObject)
	o, err := xRefTable.Dereference(rawObject)
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("AcroForm XFA: dereference: %w", err), objNr)
	}
	if o == nil {
		return model.WithValidationErrorObject(errors.New("AcroForm XFA: missing object"), objNr)
	}
	switch o := o.(type) {
	case types.StreamDict:
		// no further processing
	case types.Array:
		if err = validateFormXFAArray(xRefTable, o, objNr); err != nil {
			return err
		}
	default:
		return model.WithValidationErrorObject(fmt.Errorf("AcroForm XFA: expected stream dict or array, got %T", o), objNr)
	}
	return xRefTable.ValidateVersion("AcroFormXFA", sinceVersion)
}

func validateQ(i int) bool { return i >= 0 && i <= 2 }

func validateFormEntryCO(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version, requiresDA bool) error {
	o, ok := d.Find("CO")
	if !ok {
		return nil
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil || len(arr) == 0 {
		return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
	}

	coObjNr := validationObjectNumber(ownerObjNr, o)
	err = validateFormCO(c, xRefTable, arr, coObjNr, sinceVersion, requiresDA)
	return model.WithValidationErrorObject(err, coObjNr)
}

func validateFormEntryDR(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	o, ok := d.Find("DR")
	if !ok {
		return nil
	}

	_, err := validateResourceDict(c, xRefTable, o)

	return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
}

func validateFormEntries(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, requiresDA bool, sinceVersion model.Version) error {
	// NeedAppearances: optional, boolean
	_, err := validateBooleanEntry(
		xRefTable, d, ownerObjNr, dictName, "NeedAppearances", OPTIONAL, model.V10, nil,
	)
	if err != nil {
		return err
	}

	// SigFlags: optional, since 1.3, integer
	sinceV := model.V13
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceV = model.V12
	}
	sf, err := validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "SigFlags", OPTIONAL, sinceV, nil)
	if err != nil {
		return err
	}
	if sf != nil {
		i := sf.Value()
		xRefTable.SignatureExist = i&1 > 0
		xRefTable.AppendOnly = i&2 > 0
	}

	// CO: array
	err = validateFormEntryCO(c, xRefTable, d, ownerObjNr, model.V13, requiresDA)
	if err != nil {
		return err
	}

	// DR, optional, resource dict
	err = validateFormEntryDR(c, xRefTable, d, ownerObjNr)
	if err != nil {
		return err
	}

	// Q: optional, integer
	_, err = validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "Q", OPTIONAL, model.V10, validateQ)
	if err != nil {
		return err
	}

	// XFA: optional, since 1.5, stream or array
	err = validateFormXFA(xRefTable, d, sinceVersion)
	return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "XFA"))
}

func handleSelfReferentialAcroForm(xRefTable *model.XRefTable, rootDict types.Dict) (bool, error) {
	if ir := rootDict.IndirectRefEntry("AcroForm"); ir != nil && xRefTable.Root != nil && *ir == *xRefTable.Root {
		const msg = "AcroForm references root catalog"
		if xRefTable.ValidationMode == model.ValidationStrict {
			return true, errors.New(msg)
		}
		model.ShowDigestedSpecViolation(msg)
		rootDict.Delete("AcroForm")
		return true, nil
	}
	return false, nil
}

func acroFormFieldsArray(
	xRefTable *model.XRefTable,
	d types.Dict,
	o types.Object,
	formObjNr int,
) (types.Array, error) {
	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		err = fmt.Errorf("Fields: dereference array: %w", err)
		return nil, model.WithValidationErrorObject(err, validationObjectNumber(formObjNr, o))
	}
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		return arr, nil
	}

	arr, removed := removeNonWidgetAnnotationsFromFormFields(xRefTable, arr)
	if removed {
		d["Fields"] = arr
	}
	return arr, nil
}

func validateFormContext(c context.Context, xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.7.2 Interactive Form Dictionary

	handled, err := handleSelfReferentialAcroForm(xRefTable, rootDict)
	if handled {
		return err
	}

	rawForm, _ := rootDict.Find("AcroForm")
	formObjNr := 0
	if xRefTable.Root != nil {
		formObjNr = xRefTable.Root.ObjectNumber.Value()
	}
	formObjNr = validationObjectNumber(formObjNr, rawForm)

	d, err := validateDictEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), "rootDict", "AcroForm", OPTIONAL, sinceVersion, nil,
	)
	if err != nil || d == nil {
		return err
	}

	// Version check
	if err = xRefTable.ValidateVersion("AcroForm", sinceVersion); err != nil {
		return model.WithValidationErrorObject(err, formObjNr)
	}

	// Fields, required, array of indirect references
	o, ok := d.Find("Fields")
	if !ok {
		// Fix empty AcroForm dict.
		rootDict.Delete("AcroForm")
		return nil
	}

	arr, err := acroFormFieldsArray(xRefTable, d, o, formObjNr)
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
	da, err := validateStringEntry(xRefTable, d, formObjNr, dictName, "DA", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}
	if xRefTable.ValidationMode == model.ValidationRelaxed && da != nil {
		// Repair
		d["DA"] = types.StringLiteral(*da)
	}

	requiresDA := da == nil || len(*da) == 0

	err = validateFormHierarchyAndFields(c, xRefTable, arr, validationObjectNumber(formObjNr, o), requiresDA)
	if err != nil {
		return err
	}

	return validateFormEntries(c, xRefTable, d, formObjNr, dictName, requiresDA, sinceVersion)
}

func locateAnnForAPAndRect(d types.Dict, r *types.Rectangle, pageAnnots map[int]model.PgAnnots) *types.IndirectRef {
	indRef := d.IndirectRefEntry("AP")
	if indRef == nil {
		return nil
	}

	apObjNr := indRef.ObjectNumber.Value()
	rect := r.ShortString()
	pageNr, objNr := 0, 0
	for page, m := range pageAnnots {
		annots, ok := m[model.AnnWidget]
		if !ok {
			continue
		}
		for candidateObjNr, annRend := range annots.Map {
			if candidateObjNr <= 0 || annRend.RectString() != rect || annRend.APObjNrInt() != apObjNr {
				continue
			}
			if objNr == 0 || page < pageNr || page == pageNr && candidateObjNr < objNr {
				pageNr, objNr = page, candidateObjNr
			}
		}
	}
	if objNr == 0 {
		return nil
	}
	return types.NewIndirectRef(objNr, 0)
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
		return nil, fmt.Errorf("form field obj#%d: dereference page annotation candidate: %w", indRef.ObjectNumber.Value(), err)
	}

	arr, err := xRefTable.DereferenceArray(d["Rect"])
	if err != nil {
		return nil, fmt.Errorf("form field obj#%d Rect: dereference array: %w", indRef.ObjectNumber.Value(), err)
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
	ft, _, err := xRefTable.DereferenceNameEntry(d, "FT")
	if err != nil {
		return nil, fmt.Errorf("form field obj#%d FT: %w", indRef.ObjectNumber.Value(), err)
	}
	if ft != nil && ft.Value() == "Sig" {
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
}

func fixFormFieldsArray(xRefTable *model.XRefTable, arr types.Array) (types.Array, error) {
	arr1 := types.Array{}
	for i, obj := range arr {
		ir, ok := obj.(types.IndirectRef)
		if !ok {
			return nil, fmt.Errorf("AcroForm Fields[%d]: expected indirect reference, got %T", i, obj)
		}
		indRef, err := pageAnnotIndRefForAcroField(xRefTable, ir)
		if err != nil {
			return nil, fmt.Errorf("AcroForm Fields[%d]: resolve page annotation: %w", i, err)
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
			return fmt.Errorf("AcroForm Fields: expected array or indirect reference, got %T", o)
		}
		arr, err := fixFormFieldsArray(xRefTable, arr)
		if err != nil {
			return err
		}
		indRef, err := xRefTable.IndRefForNewObject(arr)
		if err != nil {
			return fmt.Errorf("AcroForm Fields: create repaired fields array reference: %w", err)
		}
		xRefTable.Form["Fields"] = *indRef
		return nil
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return fmt.Errorf("AcroForm Fields: dereference array: %w", err)
	}
	arr, err = fixFormFieldsArray(xRefTable, arr)
	if err != nil {
		return err
	}
	entry, _ := xRefTable.FindTableEntryForIndRef(&indRef)
	entry.Object = arr

	return nil
}

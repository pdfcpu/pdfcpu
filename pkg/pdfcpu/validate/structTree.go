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
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var errUnsupportedPDFObject = errors.New("unsupported PDF object")

type invalidStructElementKError struct {
	err error
}

func (e *invalidStructElementKError) Error() string {
	return e.err.Error()
}

func handleInvalidStructElementK(
	xRefTable *model.XRefTable,
	err error,
	specViolations *[]error,
) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}
	*specViolations = append(*specViolations, err)
	return nil
}

func showDigestedSpecViolations(specViolations []error) {
	for _, err := range specViolations {
		model.ShowDigestedSpecViolationError(err)
	}
}

func validateMarkedContentReferenceDict(xRefTable *model.XRefTable, d types.Dict) error {
	var err error

	// Pg: optional, indirect reference
	// Page object representing a page on which the graphics object in the marked-content sequence shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err = processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return fmt.Errorf("marked content reference Pg: %w", err)
		}
	}

	// Stm: optional, indirect reference
	// The content stream containing the marked-content sequence.
	if ir := d.IndirectRefEntry("Stm"); ir != nil {
		_, err = xRefTable.Dereference(*ir)
		if err != nil {
			err = fmt.Errorf("marked content reference Stm: dereference: %w", err)
			return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
		}
	}

	// StmOwn: optional, indirect reference
	// The PDF object owning the stream identified by Stems annotation to which an appearance stream belongs.
	if ir := d.IndirectRefEntry("StmOwn"); ir != nil {
		_, err = xRefTable.Dereference(*ir)
		if err != nil {
			err = fmt.Errorf("marked content reference StmOwn: dereference: %w", err)
			return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
		}
	}

	// MCID: required, integer
	// The marked-content identifier of the marked-content sequence within its content stream.

	obj, ok := d.Find("MCID")
	if !ok {
		return errors.New("marked content reference: missing MCID")
	}

	if _, err = xRefTable.DereferenceInteger(obj); err != nil {
		return fmt.Errorf("marked content reference MCID: dereference: %w", err)
	}

	return nil
}

func validateObjectReferenceDict(xRefTable *model.XRefTable, d types.Dict) error {
	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err := processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return fmt.Errorf("object reference Pg: %w", err)
		}
	}

	// Obj: required, indirect reference
	ir := d.IndirectRefEntry("Obj")
	if xRefTable.ValidationMode == model.ValidationStrict && ir == nil {
		return errors.New("object reference: missing Obj")
	}

	if ir == nil {
		model.ShowSkipped(`objectReferenceDict: entry "Obj"`)
		return nil
	}

	objNr := ir.ObjectNumber.Value()
	obj, err := xRefTable.Dereference(*ir)
	if err != nil {
		err = fmt.Errorf("object reference Obj: dereference: %w", err)
		return model.WithValidationErrorObject(err, objNr)
	}

	if obj == nil {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf("objectReferenceDict: missing obj#%s", ir.ObjectNumber))
			return nil
		}
		err = errors.New("object reference Obj: missing object")
		return model.WithValidationErrorObject(err, objNr)
	}

	return nil
}

func enterStructureTreeObject(visit *model.StructureTreeVisit, o types.Object) (int, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0, nil
	}

	objNr := ir.ObjectNumber.Value()
	if err := visit.Enter(objNr); err != nil {
		return objNr, err
	}

	return objNr, nil
}

func validateStructElementKArrayElement(
	xRefTable *model.XRefTable,
	o types.Object,
	ownerObjNr int,
	useIDs bool,
	depth int,
	visit *model.StructureTreeVisit,
) error {
	switch o := o.(type) {

	case types.Integer:
		return nil

	case types.Dict:

		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			return validateStructElementDictDepth(xRefTable, o, ownerObjNr, useIDs, depth+1, visit)
		}

		if *dictType == "MCR" {
			return validateMarkedContentReferenceDict(xRefTable, o)
		}

		if *dictType == "OBJR" {
			return validateObjectReferenceDict(xRefTable, o)
		}

		err := fmt.Errorf("unexpected dict Type %s, expected StructElem, OBJR or MCR", *dictType)
		return &invalidStructElementKError{err}

	}

	return fmt.Errorf("%w: %T", errUnsupportedPDFObject, o)
}

func validateStructElementDictEntryKArrayElement(
	xRefTable *model.XRefTable,
	rawObject types.Object,
	index int,
	useIDs bool,
	depth int,
	visit *model.StructureTreeVisit,
	specViolations *[]error,
) (err error) {
	context := objectContext(fmt.Sprintf("structure element K[%d]", index), rawObject)
	ownerObjNr := validationObjectNumber(0, rawObject)
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	objNr, err := enterStructureTreeObject(visit, rawObject)
	if err != nil {
		err = fmt.Errorf("%s: %w", context, err)
		if xRefTable.ValidationMode == model.ValidationRelaxed && errors.Is(err, model.ErrStructureTreeCycle) {
			*specViolations = append(*specViolations, err)
			return nil
		}
		return err
	}
	defer visit.Leave(objNr)

	o, err := xRefTable.Dereference(rawObject)
	if err != nil {
		return fmt.Errorf("%s: dereference: %w", context, err)
	}
	if o == nil {
		return nil
	}

	if err := validateStructElementKArrayElement(xRefTable, o, ownerObjNr, useIDs, depth, visit); err != nil {
		err = model.WrapRecursionError(context, err)
		var invalidK *invalidStructElementKError
		if errors.As(err, &invalidK) {
			return handleInvalidStructElementK(xRefTable, err, specViolations)
		}
		return err
	}

	return nil
}

func validateStructElementDictEntryKArrayDepth(
	xRefTable *model.XRefTable,
	a types.Array,
	useIDs bool,
	depth int,
	visit *model.StructureTreeVisit,
	specViolations *[]error,
) (err error) {
	for i, o := range a {
		if err := validateStructElementDictEntryKArrayElement(
			xRefTable,
			o,
			i,
			useIDs,
			depth,
			visit,
			specViolations,
		); err != nil {
			return err
		}
	}

	return nil
}

func validateStructElementDictEntryKArray(xRefTable *model.XRefTable, a types.Array, useIDs bool, depth int) error {
	var specViolations []error
	err := validateStructElementDictEntryKArrayDepth(
		xRefTable,
		a,
		useIDs,
		depth,
		model.NewStructureTreeVisit(),
		&specViolations,
	)
	if err == nil {
		showDigestedSpecViolations(specViolations)
	}
	return err
}

func validateStructElementDictEntryKDepth(
	xRefTable *model.XRefTable,
	rawObject types.Object,
	useIDs bool,
	depth int,
	visit *model.StructureTreeVisit,
	specViolations *[]error,
) (err error) {
	// K: optional, the children of this structure element
	//
	// struct element dict
	// marked content reference dict
	// object reference dict
	// marked content id int
	// array of all above

	context := objectContext("structure element K", rawObject)
	ownerObjNr := validationObjectNumber(0, rawObject)
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	objNr, err := enterStructureTreeObject(visit, rawObject)
	if err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	defer visit.Leave(objNr)

	o, err := xRefTable.Dereference(rawObject)
	if err != nil {
		return fmt.Errorf("%s: dereference: %w", context, err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Integer:

	case types.Dict:
		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDictDepth(xRefTable, o, ownerObjNr, useIDs, depth+1, visit)
			if err != nil {
				return model.WrapRecursionError(context, err)
			}
			break
		}

		if *dictType == "MCR" {
			err = validateMarkedContentReferenceDict(xRefTable, o)
			if err != nil {
				return fmt.Errorf("%s: %w", context, err)
			}
			break
		}

		if *dictType == "OBJR" {
			err = validateObjectReferenceDict(xRefTable, o)
			if err != nil {
				return fmt.Errorf("%s: %w", context, err)
			}
			break
		}

		err := fmt.Errorf("%s: unexpected dict Type %s, expected StructElem, OBJR or MCR", context, *dictType)
		return handleInvalidStructElementK(xRefTable, err, specViolations)

	case types.Array:

		err = validateStructElementDictEntryKArrayDepth(xRefTable, o, useIDs, depth, visit, specViolations)
		if err != nil {
			return err
		}

	default:
		err := fmt.Errorf("%s: %w: %T", context, errUnsupportedPDFObject, o)
		return err

	}

	return nil
}

func validateStructElementDictEntryK(xRefTable *model.XRefTable, o types.Object, useIDs bool, depth int) error {
	var specViolations []error
	err := validateStructElementDictEntryKDepth(
		xRefTable,
		o,
		useIDs,
		depth,
		model.NewStructureTreeVisit(),
		&specViolations,
	)
	if err == nil {
		showDigestedSpecViolations(specViolations)
	}
	return err
}

func processStructElementDictPgEntry(xRefTable *model.XRefTable, ir types.IndirectRef) (err error) {
	objNr := ir.ObjectNumber.Value()
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// is this object a known page object?

	o, err := xRefTable.Dereference(ir)
	if err != nil {
		return fmt.Errorf("page: dereference: %w", err)
	}

	//logInfoWriter.Printf("known object for Pg: %v %s\n", obj, obj)

	if xRefTable.ValidationMode == model.ValidationRelaxed && o == nil {
		return nil
	}

	pageDict, ok := o.(types.Dict)
	if !ok {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf("invalid structElementDict Pg entry, objNr: %d ", ir.ObjectNumber))
			return nil
		}
		return fmt.Errorf("page: expected page dict, got %T", o)
	}

	if t := pageDict.Type(); t == nil || *t != "Page" {
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf("invalid structElementDict Pg entry, objNr: %d ", ir.ObjectNumber))
			return nil
		}
		return errors.New("page: expected Type Page")
	}

	return nil
}

func validateStructElementDictEntryA(xRefTable *model.XRefTable, o types.Object) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("structure element A: dereference: %w", err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Dict: // No further processing.

	case types.StreamDict: // No further processing.

	case types.Array:

		for i, o := range o {
			entryObjNr := validationObjectNumber(objNr, o)

			o, err := xRefTable.Dereference(o)
			if err != nil {
				err = fmt.Errorf("structure element A[%d]: dereference: %w", i, err)
				return model.WithValidationErrorObject(err, entryObjNr)
			}

			if o == nil {
				continue
			}

			switch o.(type) {

			case types.Integer:
				// Each array element may be followed by a revision number (int).sort

			case types.Dict:
				// No further processing.

			case types.StreamDict:
				// No further processing.

			default:
				err = fmt.Errorf("structure element A[%d]: %w: %T", i, errUnsupportedPDFObject, o)
				return model.WithValidationErrorObject(err, entryObjNr)
			}
		}

	default:
		return fmt.Errorf("structure element A: %w: %T", errUnsupportedPDFObject, o)

	}

	return nil
}

func validateStructElementDictEntryC(xRefTable *model.XRefTable, o types.Object) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("structure element C: dereference: %w", err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Name:
		// No further processing.

	case types.Array:

		for i, o := range o {
			entryObjNr := validationObjectNumber(objNr, o)

			o, err := xRefTable.Dereference(o)
			if err != nil {
				err = fmt.Errorf("structure element C[%d]: dereference: %w", i, err)
				return model.WithValidationErrorObject(err, entryObjNr)
			}

			if o == nil {
				continue
			}

			switch o.(type) {

			case types.Name:
				// No further processing.

			case types.Integer:
				// Each array element may be followed by a revision number.

			default:
				err = fmt.Errorf("structure element C[%d]: %w: %T", i, errUnsupportedPDFObject, o)
				return model.WithValidationErrorObject(err, entryObjNr)

			}
		}

	default:
		return fmt.Errorf("structure element C: %w: %T", errUnsupportedPDFObject, o)

	}

	return nil
}

func validateStructElementDictEntryP(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	// P: immediate parent, required, indirect reference
	ir := d.IndirectRefEntry("P")
	if xRefTable.ValidationMode != model.ValidationRelaxed {
		if ir == nil {
			logMissingRequiredEntry(dictName, "P", d)
			err := missingRequiredEntryError(dictName, "P", "add parent structure element reference or validate in relaxed mode")
			return model.WithValidationErrorObject(err, ownerObjNr)
		}

		// Check if parent structure element exists.
		if _, ok := xRefTable.FindTableEntryForIndRef(ir); !ok {
			err := errors.New("structure element parent: unknown")
			return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
		}
	}

	return nil
}

func validateStructElementDictEntryPg(xRefTable *model.XRefTable, d types.Dict) error {
	if ir := d.IndirectRefEntry("Pg"); ir != nil {
		err := processStructElementDictPgEntry(xRefTable, *ir)
		if err != nil {
			return fmt.Errorf("structure element Pg: %w", err)
		}
	}
	return nil
}

func validateStructElementDictEntryS(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	_, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "S", OPTIONAL, model.V10, nil)
	if err == nil {
		return nil
	}
	if xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}

	i, err := validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "S", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}
	if i != nil {
		d["S"] = types.Name(strconv.Itoa((*i).Value()))
	}
	return nil
}

func validateStructElementDictPart1(
	xRefTable *model.XRefTable,
	d types.Dict,
	ownerObjNr int,
	dictName string,
	useIDs bool,
	depth int,
	visit *model.StructureTreeVisit,
	specViolations *[]error,
) error {
	// S: structure type, required, name, see 14.7.3 and Annex E.
	if err := validateStructElementDictEntryS(xRefTable, d, ownerObjNr, dictName); err != nil {
		return err
	}

	if err := validateStructElementDictEntryP(xRefTable, d, ownerObjNr, dictName); err != nil {
		return err
	}

	if useIDs {
		// ID: optional, byte string
		_, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "ID", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}
	}

	// Pg: optional, indirect reference
	// Page object representing a page on which some or all of the content items designated by the K entry shall be rendered.
	if err := validateStructElementDictEntryPg(xRefTable, d); err != nil {
		return err
	}

	// K: optional, the children of this structure element.
	if o, found := d.Find("K"); found {
		if err := validateStructElementDictEntryKDepth(
			xRefTable,
			o,
			useIDs,
			depth,
			visit,
			specViolations,
		); err != nil {
			return err
		}
	}

	// A: optional, attribute objects: dict or stream dict or array of these.
	if o, ok := d.Find("A"); ok {
		if err := validateStructElementDictEntryA(xRefTable, o); err != nil {
			return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
		}
	}

	return nil
}

func validateStructElementLang(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, sinceVersion model.Version) (bool, error) {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		if o, ok := d.Find("Lang"); ok {
			langObjNr := validationObjectNumber(ownerObjNr, o)
			o, err := xRefTable.Dereference(o)
			if err != nil {
				return false, model.WithValidationErrorObject(err, langObjNr)
			}
			if _, ok := o.(types.Name); ok {
				err = xRefTable.ValidateVersion("dict="+dictName+" entry=Lang", sinceVersion)
				return err == nil, err
			}
		}
	}

	_, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Lang", OPTIONAL, sinceVersion, nil)

	return false, err
}

func validateStructElementDictPart2(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) (bool, error) {
	// C: optional, name or array
	if o, ok := d.Find("C"); ok {
		err := validateStructElementDictEntryC(xRefTable, o)
		if err != nil {
			return false, model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
		}
	}

	// R: optional, integer >= 0
	_, err := validateIntegerEntry(
		xRefTable, d, ownerObjNr, dictName, "R", OPTIONAL, model.V10, func(i int) bool { return i >= 0 },
	)
	if err != nil {
		return false, err
	}

	// T: optional, text string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "T", OPTIONAL, model.V10, nil)
	if err != nil {
		return false, err
	}

	// Lang: optional, text string, since 1.4
	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	langName, err := validateStructElementLang(xRefTable, d, ownerObjNr, dictName, sinceVersion)
	if err != nil {
		return false, err
	}

	// Alt: optional, text string
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Alt", OPTIONAL, model.V10, nil)
	if err != nil {
		return false, err
	}

	// E: optional, text string, since 1.5
	sinceVersion = model.V15
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "E", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return false, err
	}

	// ActualText: optional, text string, since 1.4
	sinceVersion = model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	_, err = validateStringEntry(xRefTable, d, ownerObjNr, dictName, "ActualText", OPTIONAL, sinceVersion, nil)

	return langName, err
}

func validateStructElementDict(xRefTable *model.XRefTable, d types.Dict, useIDs bool) error {
	return validateStructElementDictDepth(xRefTable, d, 0, useIDs, 0, model.NewStructureTreeVisit())
}

func validateStructElementDictDepth(
	xRefTable *model.XRefTable,
	d types.Dict,
	ownerObjNr int,
	useIDs bool,
	depth int,
	visit *model.StructureTreeVisit,
) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	if err := xRefTable.CheckRecursionDepth("structure tree", depth); err != nil {
		return err
	}

	// See table 323

	dictName := "StructElementDict"

	var specViolations []error
	err = validateStructElementDictPart1(
		xRefTable, d, ownerObjNr, dictName, useIDs, depth, visit, &specViolations,
	)
	if err != nil {
		return err
	}

	langName, err := validateStructElementDictPart2(xRefTable, d, ownerObjNr, dictName)
	if err != nil {
		return err
	}

	if langName {
		model.ShowDigestedSpecViolation("dict=" + dictName + " entry=Lang invalid type types.Name")
	}
	showDigestedSpecViolations(specViolations)

	return nil
}

func handleInvalidStructTreeObject(xRefTable *model.XRefTable, err error) error {
	if xRefTable.ValidationMode == model.ValidationStrict {
		return err
	}
	model.ShowDigestedSpecViolationError(err)
	return nil
}

func validateStructTreeRootDictEntryKArrayElement(
	xRefTable *model.XRefTable,
	rawObject types.Object,
	index int,
	useIDs bool,
	visit *model.StructureTreeVisit,
) (err error) {
	context := objectContext(fmt.Sprintf("structure tree root K[%d]", index), rawObject)
	ownerObjNr := validationObjectNumber(0, rawObject)
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	objNr, err := enterStructureTreeObject(visit, rawObject)
	if err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	defer visit.Leave(objNr)

	o, err := xRefTable.Dereference(rawObject)
	if err != nil {
		return fmt.Errorf("%s: dereference: %w", context, err)
	}
	if o == nil {
		return nil
	}

	d, ok := o.(types.Dict)
	if !ok {
		err := fmt.Errorf("%s: %w: %T", context, errUnsupportedPDFObject, o)
		return handleInvalidStructTreeObject(xRefTable, err)
	}

	dictType := d.Type()
	if dictType == nil || *dictType == "StructElem" {
		if err := validateStructElementDictDepth(xRefTable, d, ownerObjNr, useIDs, 1, visit); err != nil {
			return fmt.Errorf("%s: %w", context, err)
		}
		return nil
	}

	err = fmt.Errorf("%s: unexpected dict Type %s, expected StructElem", context, *dictType)

	return handleInvalidStructTreeObject(xRefTable, err)
}

func validateStructTreeRootDictEntryKArrayDepth(
	xRefTable *model.XRefTable,
	a types.Array,
	useIDs bool,
	visit *model.StructureTreeVisit,
) (err error) {
	for i, o := range a {
		if err := validateStructTreeRootDictEntryKArrayElement(xRefTable, o, i, useIDs, visit); err != nil {
			return err
		}
	}

	return nil
}

func validateStructTreeRootDictEntryKArray(xRefTable *model.XRefTable, a types.Array, useIDs bool) error {
	return validateStructTreeRootDictEntryKArrayDepth(xRefTable, a, useIDs, model.NewStructureTreeVisit())
}

func validateStructTreeRootDictEntryKDepth(
	xRefTable *model.XRefTable,
	rawObject types.Object,
	useIDs bool,
	visit *model.StructureTreeVisit,
) (err error) {
	// The immediate child or children of the structure tree root in the structure hierarchy.
	// The value may be either a dictionary representing a single structure element or an array of such dictionaries.

	context := objectContext("structure tree root K", rawObject)
	ownerObjNr := validationObjectNumber(0, rawObject)
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	objNr, err := enterStructureTreeObject(visit, rawObject)
	if err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	defer visit.Leave(objNr)

	o, err := xRefTable.Dereference(rawObject)
	if err != nil {
		return fmt.Errorf("%s: dereference: %w", context, err)
	}
	if o == nil {
		return nil
	}

	switch o := o.(type) {

	case types.Dict:

		dictType := o.Type()

		if dictType == nil || *dictType == "StructElem" {
			err = validateStructElementDictDepth(xRefTable, o, ownerObjNr, useIDs, 1, visit)
			if err != nil {
				return fmt.Errorf("%s: %w", context, err)
			}
			break
		}

		err := fmt.Errorf("%s: unexpected dict Type %s, expected StructElem", context, *dictType)
		return handleInvalidStructTreeObject(xRefTable, err)

	case types.Array:

		err = validateStructTreeRootDictEntryKArrayDepth(xRefTable, o, useIDs, visit)
		if err != nil {
			return err
		}

	default:
		err := fmt.Errorf("%s: %w: %T", context, errUnsupportedPDFObject, o)
		return handleInvalidStructTreeObject(xRefTable, err)

	}

	return nil
}

func validateStructTreeRootDictEntryK(xRefTable *model.XRefTable, o types.Object, useIDs bool) error {
	return validateStructTreeRootDictEntryKDepth(xRefTable, o, useIDs, model.NewStructureTreeVisit())
}

func processStructTreeClassMapDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	for name, o := range d {
		entryObjNr := validationObjectNumber(ownerObjNr, o)

		// Process dict or array of dicts.

		o, err := xRefTable.Dereference(o)
		if err != nil {
			err = fmt.Errorf("structure tree ClassMap %s: dereference: %w", name, err)
			return model.WithValidationErrorObject(err, entryObjNr)
		}

		if o == nil {
			continue
		}

		switch o := o.(type) {

		case types.Dict:
			// no further processing.

		case types.Array:

			for i, o := range o {
				arrayEntryObjNr := validationObjectNumber(entryObjNr, o)

				_, err = xRefTable.DereferenceDict(o)
				if err != nil {
					err = fmt.Errorf("structure tree ClassMap %s[%d]: dereference dict: %w", name, i, err)
					return model.WithValidationErrorObject(err, arrayEntryObjNr)
				}

			}

		default:
			err = fmt.Errorf("structure tree ClassMap %s: %w: %T", name, errUnsupportedPDFObject, o)
			return model.WithValidationErrorObject(err, entryObjNr)

		}

	}

	return nil
}

func validateStructTreeRootDictEntryParentTree(xRefTable *model.XRefTable, ir *types.IndirectRef, useIDs bool) error {
	objNr := ir.ObjectNumber.Value()
	if xRefTable.ValidationMode == model.ValidationRelaxed {

		// Accept empty dict
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			err = fmt.Errorf("structure tree ParentTree: dereference: %w", err)
			return model.WithValidationErrorObject(err, objNr)
		}
		if d == nil || d.Len() == 0 {
			return nil
		}
	}

	d, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		err = fmt.Errorf("structure tree ParentTree: dereference: %w", err)
		return model.WithValidationErrorObject(err, objNr)
	}

	_, _, err = validateNumberTree(xRefTable, "StructTree", d, objNr, true, useIDs)
	return err
}

func validateStructTreeRootDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "StructTreeRootDict"

	// required entry Type: name:StructTreeRoot
	if d.Type() == nil || *d.Type() != "StructTreeRoot" {
		return errors.New("structure tree root: missing Type StructTreeRoot")
	}

	useIDs := false

	// Optional entry IDTree: name tree, key=elementId value=struct element dict
	// A name tree that maps element identifiers to the structure elements they denote.
	ir := d.IndirectRefEntry("IDTree")
	if ir != nil {
		idTreeObjNr := ir.ObjectNumber.Value()
		d, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			err = fmt.Errorf("structure tree IDTree: dereference: %w", err)
			return model.WithValidationErrorObject(err, idTreeObjNr)
		}
		if len(d) > 0 {
			_, _, _, err = validateNameTree(xRefTable, "IDTree", d, idTreeObjNr, true)
			if err != nil {
				return fmt.Errorf("structure tree IDTree: %w", err)
			}
			useIDs = true
		}
	}

	// Optional entry K: struct element dict or array of struct element dicts
	if o, found := d.Find("K"); found {
		err := validateStructTreeRootDictEntryK(xRefTable, o, useIDs)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTree: number tree, value=indRef of struct element dict or array of struct element dicts
	// A number tree used in finding the structure elements to which content items belong.
	if ir = d.IndirectRefEntry("ParentTree"); ir != nil {
		err := validateStructTreeRootDictEntryParentTree(xRefTable, ir, useIDs)
		if err != nil {
			return err
		}
	}

	// Optional entry ParentTreeNextKey: integer
	_, err = validateIntegerEntry(
		xRefTable, d, ownerObjNr, dictName, "ParentTreeNextKey", OPTIONAL, model.V10, nil,
	)
	if err != nil {
		return err
	}

	// Optional entry RoleMap: dict
	// A dictionary that shall map the names of structure used in the document
	// to their approximate equivalents in the set of standard structure
	_, err = validateDictEntry(xRefTable, d, ownerObjNr, dictName, "RoleMap", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// Optional entry ClassMap: dict
	// A dictionary that shall map name objects designating attribute classes
	// to the corresponding attribute objects or arrays of attribute objects.
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "ClassMap", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		classMapObjNr := validationEntryObjectNumber(ownerObjNr, d, "ClassMap")
		err = processStructTreeClassMapDict(xRefTable, d1, classMapObjNr)
	}

	return err
}

func validateStructTree(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// 14.7.2 Structure Hierarchy

	rootObjNr := validationRootObjectNumber(xRefTable)
	d, err := validateDictEntry(
		xRefTable, rootDict, rootObjNr, "RootDict", "StructTreeRoot", required, sinceVersion, nil,
	)
	if err != nil || d == nil {
		return err
	}

	return validateStructTreeRootDict(
		xRefTable, d, validationEntryObjectNumber(rootObjNr, rootDict, "StructTreeRoot"),
	)
}

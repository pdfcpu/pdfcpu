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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateBeadPageEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, sinceVersion model.Version) (bool, error) {
	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}

	ir, err := validateIndRefEntry(xRefTable, d, 0, dictName, "P", required, sinceVersion)
	if err != nil || ir == nil {
		return ir == nil, err
	}

	pageDict, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return false, model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}
	if pageDict == nil {
		return false, model.WithValidationErrorObject(errors.New("missing page dict"), ir.ObjectNumber.Value())
	}

	_, err = validateNameEntry(xRefTable, pageDict, 0, "pageDict", "Type", REQUIRED, model.V10, func(s string) bool {
		return s == "Page"
	})

	return false, model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
}

func threadObjectContext(err error, role string, objNumber int) string {
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) && validationErr.ObjectNumber() != objNumber {
		return fmt.Sprintf("%s obj#%d", role, objNumber)
	}
	return role
}

func validateEntryV(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version, pBeadIndRef *types.IndirectRef, objNumber int) error {
	previousBeadIndRef, err := validateIndRefEntry(xRefTable, d, 0, dictName, "V", required, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s.V: %w", dictName, err)
	}

	if *previousBeadIndRef != *pBeadIndRef {
		return fmt.Errorf("bead obj#%d: invalid V backpointer", objNumber)
	}

	return nil
}

func enterBead(c context.Context, visit *model.BeadVisit, beadIndRef *types.IndirectRef) (int, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	objNumber := beadIndRef.ObjectNumber.Value()
	if err := visit.Enter(objNumber); err != nil {
		return 0, err
	}
	return objNumber, nil
}

func validateBeadDict(c context.Context, xRefTable *model.XRefTable, beadIndRef, threadIndRef, pBeadIndRef, lBeadIndRef *types.IndirectRef, visit *model.BeadVisit) error {
	dictName := "beadDict"
	sinceVersion := model.V10

	for {
		objNumber := beadIndRef.ObjectNumber.Value()
		_, err := enterBead(c, visit, beadIndRef)
		if err != nil {
			return model.WithValidationErrorObject(err, objNumber)
		}

		d, err := xRefTable.DereferenceDict(*beadIndRef)
		if err != nil {
			err = fmt.Errorf("bead: dereference dict: %w", err)
			return model.WithValidationErrorObject(err, objNumber)
		}
		if d == nil {
			return model.WithValidationErrorObject(errors.New("bead: missing dict"), objNumber)
		}

		// Validate optional entry Type, must be "Bead".
		_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
		if err != nil {
			err = fmt.Errorf("%s Type: %w", threadObjectContext(err, "bead", objNumber), err)
			return model.WithValidationErrorObject(err, objNumber)
		}

		// Validate entry T, must refer to threadDict.
		indRefT, err := validateIndRefEntry(xRefTable, d, 0, dictName, "T", OPTIONAL, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s T: %w", threadObjectContext(err, "bead", objNumber), err)
			return model.WithValidationErrorObject(err, objNumber)
		}
		if indRefT != nil && *indRefT != *threadIndRef {
			err = fmt.Errorf("bead obj#%d: invalid T backpointer to thread dict", objNumber)
			return model.WithValidationErrorObject(err, objNumber)
		}

		// Validate required entry R, must be rectangle.
		_, err = validateRectangleEntry(xRefTable, d, 0, dictName, "R", REQUIRED, sinceVersion, nil)
		if err != nil {
			err = fmt.Errorf("%s R: %w", threadObjectContext(err, "bead", objNumber), err)
			return model.WithValidationErrorObject(err, objNumber)
		}

		// Validate required entry P, must be indRef to pageDict.
		missingP, err := validateBeadPageEntry(xRefTable, d, dictName, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s P: %w", threadObjectContext(err, "bead", objNumber), err)
			return model.WithValidationErrorObject(err, objNumber)
		}

		// Validate required entry V, must refer to previous bead.
		err = validateEntryV(xRefTable, d, dictName, REQUIRED, sinceVersion, pBeadIndRef, objNumber)
		if err != nil {
			return model.WithValidationErrorObject(err, objNumber)
		}

		// Validate required entry N, must refer to last bead.
		nBeadIndRef, err := validateIndRefEntry(xRefTable, d, 0, dictName, "N", REQUIRED, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s N: %w", threadObjectContext(err, "bead", objNumber), err)
			return model.WithValidationErrorObject(err, objNumber)
		}

		if missingP {
			model.ShowDigestedSpecViolation("dict=" + dictName + " required entry=P missing")
		}
		if *nBeadIndRef == *lBeadIndRef {
			return nil
		}

		pBeadIndRef = beadIndRef
		beadIndRef = nBeadIndRef
	}
}

func soleBeadDict(beadIndRef, pBeadIndRef, nBeadIndRef *types.IndirectRef) bool {
	// if N and V reference this bead dict, must be the first and only one.
	return *pBeadIndRef == *nBeadIndRef && *beadIndRef == *pBeadIndRef
}

func validateBeadChainIntegrity(beadIndRef, pBeadIndRef, nBeadIndRef *types.IndirectRef) bool {
	return *pBeadIndRef != *beadIndRef && *nBeadIndRef != *beadIndRef
}

func validateFirstBeadDict(c context.Context, xRefTable *model.XRefTable, beadIndRef, threadIndRef *types.IndirectRef) (err error) {
	dictName := "firstBeadDict"
	sinceVersion := model.V10
	objNumber := beadIndRef.ObjectNumber.Value()
	defer func() {
		err = model.WithValidationErrorObject(err, objNumber)
	}()
	if err := contextutil.Check(c); err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(*beadIndRef)
	if err != nil {
		return fmt.Errorf("first bead: dereference dict: %w", err)
	}

	if d == nil {
		return errors.New("first bead: missing dict")
	}

	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return fmt.Errorf("%s Type: %w", threadObjectContext(err, "first bead", objNumber), err)
	}

	indRefT, err := validateIndRefEntry(xRefTable, d, 0, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s T: %w", threadObjectContext(err, "first bead", objNumber), err)
	}

	if *indRefT != *threadIndRef {
		return fmt.Errorf("first bead obj#%d: invalid T backpointer to thread dict", objNumber)
	}

	_, err = validateRectangleEntry(xRefTable, d, 0, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s R: %w", threadObjectContext(err, "first bead", objNumber), err)
	}

	missingP, err := validateBeadPageEntry(xRefTable, d, dictName, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s P: %w", threadObjectContext(err, "first bead", objNumber), err)
	}

	pBeadIndRef, err := validateIndRefEntry(xRefTable, d, 0, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s V: %w", threadObjectContext(err, "first bead", objNumber), err)
	}

	nBeadIndRef, err := validateIndRefEntry(xRefTable, d, 0, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s N: %w", threadObjectContext(err, "first bead", objNumber), err)
	}

	if !soleBeadDict(beadIndRef, pBeadIndRef, nBeadIndRef) {
		if !validateBeadChainIntegrity(beadIndRef, pBeadIndRef, nBeadIndRef) {
			return fmt.Errorf("first bead obj#%d: corrupt bead chain", objNumber)
		}
		if err = validateBeadDict(c, xRefTable, nBeadIndRef, threadIndRef, beadIndRef, pBeadIndRef, model.NewBeadVisit()); err != nil {
			return fmt.Errorf("first bead obj#%d next: %w", objNumber, err)
		}
	}
	if missingP {
		model.ShowDigestedSpecViolation("dict=" + dictName + " required entry=P missing")
	}
	return nil
}

func validateThreadDict(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) (err error) {
	dictName := "threadDict"
	var specViolations []error
	if err := contextutil.Check(c); err != nil {
		return err
	}

	threadIndRef, ok := o.(types.IndirectRef)
	if !ok {
		return fmt.Errorf("threadDict: expected indirect ref, got %T", o)
	}

	objNumber := threadIndRef.ObjectNumber.Value()
	defer func() {
		err = model.WithValidationErrorObject(err, objNumber)
	}()

	d, err := xRefTable.DereferenceDict(threadIndRef)
	if err != nil {
		return fmt.Errorf("thread: dereference dict: %w", err)
	}
	if d == nil {
		return errors.New("thread: missing dict")
	}

	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return fmt.Errorf("%s Type: %w", threadObjectContext(err, "thread", objNumber), err)
	}

	// Validate optional thread information dict entry.
	o, found := d.Find("I")
	if found && o != nil {
		_, specViolations, err = validateDocumentInfoDict(xRefTable, o)
		if err != nil {
			return fmt.Errorf("%s I: %w", threadObjectContext(err, "thread", objNumber), err)
		}
	}

	fBeadIndRef, err := validateIndRefEntry(xRefTable, d, 0, dictName, "F", OPTIONAL, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s F: %w", threadObjectContext(err, "thread", objNumber), err)
	}
	if fBeadIndRef == nil {
		msg := "thread F: missing required indirect entry"
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return errors.New(msg)
		}
		specViolations = append(
			specViolations,
			model.WithValidationErrorObject(errors.New(msg), objNumber),
		)
		showDigestedSpecViolations(specViolations)
		return nil
	}

	// Validate the list of beads starting with the first bead dict.
	if err = validateFirstBeadDict(c, xRefTable, fBeadIndRef, &threadIndRef); err != nil {
		return fmt.Errorf("thread obj#%d first bead: %w", objNumber, err)
	}
	showDigestedSpecViolations(specViolations)
	return nil
}

func validateThreads(c context.Context, xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.4.3 Articles
	if err := contextutil.Check(c); err != nil {
		return err
	}

	ir := rootDict.IndirectRefEntry("Threads")
	if ir == nil {
		if required {
			err := errors.New("rootDict.Threads: missing required entry")
			return model.WithValidationErrorObject(err, validationRootObjectNumber(xRefTable))
		}
		return nil
	}

	a, err := xRefTable.DereferenceArray(*ir)
	if err != nil {
		err = fmt.Errorf("rootDict.Threads: dereference array: %w", err)
		return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}
	if a == nil {
		return nil
	}

	err = xRefTable.ValidateVersion("threads", sinceVersion)
	if err != nil {
		err = fmt.Errorf("rootDict.Threads: %w", err)
		return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}

	for i, o := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}

		if o == nil {
			continue
		}

		err = validateThreadDict(c, xRefTable, o, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("rootDict.Threads[%d]", i), o), err)
			return model.WithValidationErrorObject(err, validationObjectNumber(ir.ObjectNumber.Value(), o))
		}

	}

	return nil
}

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

func validateBeadPageEntry(xRefTable *model.XRefTable, d types.Dict, dictName string, sinceVersion model.Version) (bool, error) {
	required := REQUIRED
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		required = OPTIONAL
	}

	ir, err := validateIndRefEntry(xRefTable, d, dictName, "P", required, sinceVersion)
	if err != nil || ir == nil {
		return ir == nil, err
	}

	pageDict, err := xRefTable.DereferenceDict(*ir)
	if err != nil {
		return false, err
	}
	if pageDict == nil {
		return false, errors.New("missing page dict")
	}

	_, err = validateNameEntry(xRefTable, pageDict, "pageDict", "Type", REQUIRED, model.V10, func(s string) bool {
		return s == "Page"
	})

	return false, err
}

func validateEntryV(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version, pBeadIndRef *types.IndirectRef, objNumber int) error {
	previousBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "V", required, sinceVersion)
	if err != nil {
		return fmt.Errorf("%s.V: %w", dictName, err)
	}

	if *previousBeadIndRef != *pBeadIndRef {
		return fmt.Errorf("bead obj#%d: invalid V backpointer", objNumber)
	}

	return nil
}

func enterBead(visit *model.BeadVisit, beadIndRef *types.IndirectRef) (int, error) {
	objNumber := beadIndRef.ObjectNumber.Value()
	if err := visit.Enter(objNumber); err != nil {
		return 0, err
	}
	return objNumber, nil
}

func validateBeadDict(
	xRefTable *model.XRefTable,
	beadIndRef,
	threadIndRef,
	pBeadIndRef,
	lBeadIndRef *types.IndirectRef,
	visit *model.BeadVisit,
) error {
	dictName := "beadDict"
	sinceVersion := model.V10

	for {
		objNumber, err := enterBead(visit, beadIndRef)
		if err != nil {
			return err
		}

		d, err := xRefTable.DereferenceDict(*beadIndRef)
		if err != nil {
			return fmt.Errorf("bead obj#%d: dereference dict: %w", objNumber, err)
		}
		if d == nil {
			return fmt.Errorf("bead obj#%d: missing dict", objNumber)
		}

		// Validate optional entry Type, must be "Bead".
		_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
		if err != nil {
			return fmt.Errorf("bead obj#%d Type: %w", objNumber, err)
		}

		// Validate entry T, must refer to threadDict.
		indRefT, err := validateIndRefEntry(xRefTable, d, dictName, "T", OPTIONAL, sinceVersion)
		if err != nil {
			return fmt.Errorf("bead obj#%d T: %w", objNumber, err)
		}
		if indRefT != nil && *indRefT != *threadIndRef {
			return fmt.Errorf("bead obj#%d: invalid T backpointer to thread dict", objNumber)
		}

		// Validate required entry R, must be rectangle.
		_, err = validateRectangleEntry(xRefTable, d, dictName, "R", REQUIRED, sinceVersion, nil)
		if err != nil {
			return fmt.Errorf("bead obj#%d R: %w", objNumber, err)
		}

		// Validate required entry P, must be indRef to pageDict.
		missingP, err := validateBeadPageEntry(xRefTable, d, dictName, sinceVersion)
		if err != nil {
			return fmt.Errorf("bead obj#%d P: %w", objNumber, err)
		}

		// Validate required entry V, must refer to previous bead.
		err = validateEntryV(xRefTable, d, dictName, REQUIRED, sinceVersion, pBeadIndRef, objNumber)
		if err != nil {
			return err
		}

		// Validate required entry N, must refer to last bead.
		nBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "N", REQUIRED, sinceVersion)
		if err != nil {
			return fmt.Errorf("bead obj#%d N: %w", objNumber, err)
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

func validateFirstBeadDict(xRefTable *model.XRefTable, beadIndRef, threadIndRef *types.IndirectRef) error {
	dictName := "firstBeadDict"
	sinceVersion := model.V10
	objNumber := beadIndRef.ObjectNumber.Value()

	d, err := xRefTable.DereferenceDict(*beadIndRef)
	if err != nil {
		return fmt.Errorf("first bead obj#%d: dereference dict: %w", objNumber, err)
	}

	if d == nil {
		return fmt.Errorf("first bead obj#%d: missing dict", objNumber)
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return fmt.Errorf("first bead obj#%d Type: %w", objNumber, err)
	}

	indRefT, err := validateIndRefEntry(xRefTable, d, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return fmt.Errorf("first bead obj#%d T: %w", objNumber, err)
	}

	if *indRefT != *threadIndRef {
		return fmt.Errorf("first bead obj#%d: invalid T backpointer to thread dict", objNumber)
	}

	_, err = validateRectangleEntry(xRefTable, d, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("first bead obj#%d R: %w", objNumber, err)
	}

	missingP, err := validateBeadPageEntry(xRefTable, d, dictName, sinceVersion)
	if err != nil {
		return fmt.Errorf("first bead obj#%d P: %w", objNumber, err)
	}

	pBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return fmt.Errorf("first bead obj#%d V: %w", objNumber, err)
	}

	nBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return fmt.Errorf("first bead obj#%d N: %w", objNumber, err)
	}

	if !soleBeadDict(beadIndRef, pBeadIndRef, nBeadIndRef) {
		if !validateBeadChainIntegrity(beadIndRef, pBeadIndRef, nBeadIndRef) {
			return fmt.Errorf("first bead obj#%d: corrupt bead chain", objNumber)
		}
		if err = validateBeadDict(xRefTable, nBeadIndRef, threadIndRef, beadIndRef, pBeadIndRef, model.NewBeadVisit()); err != nil {
			return fmt.Errorf("first bead obj#%d next: %w", objNumber, err)
		}
	}
	if missingP {
		model.ShowDigestedSpecViolation("dict=" + dictName + " required entry=P missing")
	}
	return nil
}

func validateThreadDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	dictName := "threadDict"
	var specViolations []error

	threadIndRef, ok := o.(types.IndirectRef)
	if !ok {
		return fmt.Errorf("threadDict: expected indirect ref, got %T", o)
	}

	objNumber := threadIndRef.ObjectNumber.Value()

	d, err := xRefTable.DereferenceDict(threadIndRef)
	if err != nil {
		return fmt.Errorf("thread obj#%d: dereference dict: %w", objNumber, err)
	}
	if d == nil {
		return fmt.Errorf("thread obj#%d: missing dict", objNumber)
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return fmt.Errorf("thread obj#%d Type: %w", objNumber, err)
	}

	// Validate optional thread information dict entry.
	o, found := d.Find("I")
	if found && o != nil {
		_, specViolations, err = validateDocumentInfoDict(xRefTable, o)
		if err != nil {
			return fmt.Errorf("thread obj#%d I: %w", objNumber, err)
		}
	}

	fBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "F", OPTIONAL, sinceVersion)
	if err != nil {
		return fmt.Errorf("thread obj#%d F: %w", objNumber, err)
	}
	if fBeadIndRef == nil {
		msg := fmt.Sprintf("thread obj#%d F: missing required indirect entry", objNumber)
		if xRefTable.ValidationMode != model.ValidationRelaxed {
			return errors.New(msg)
		}
		showDigestedSpecViolations(xRefTable, specViolations)
		model.ShowDigestedSpecViolation(msg)
		return nil
	}

	// Validate the list of beads starting with the first bead dict.
	if err = validateFirstBeadDict(xRefTable, fBeadIndRef, &threadIndRef); err != nil {
		return fmt.Errorf("thread obj#%d first bead: %w", objNumber, err)
	}
	showDigestedSpecViolations(xRefTable, specViolations)
	return nil
}

func validateThreads(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.4.3 Articles

	ir := rootDict.IndirectRefEntry("Threads")
	if ir == nil {
		if required {
			return errors.New("rootDict.Threads: missing required entry")
		}
		return nil
	}

	a, err := xRefTable.DereferenceArray(*ir)
	if err != nil {
		return fmt.Errorf("rootDict.Threads: dereference array: %w", err)
	}
	if a == nil {
		return nil
	}

	err = xRefTable.ValidateVersion("threads", sinceVersion)
	if err != nil {
		return fmt.Errorf("rootDict.Threads: %w", err)
	}

	for i, o := range a {

		if o == nil {
			continue
		}

		err = validateThreadDict(xRefTable, o, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("rootDict.Threads[%d]", i), o), err)
		}

	}

	return nil
}

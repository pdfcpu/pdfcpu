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

func validateEntryV(xRefTable *pdf.XRefTable, dict *pdf.PDFDict, dictName string, required bool, sinceVersion pdf.Version, pBeadIndRef *pdf.IndirectRef, objNumber int) error {

	previousBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "V", required, sinceVersion)
	if err != nil {
		return err
	}

	if !previousBeadIndRef.Equals(*pBeadIndRef) {
		return errors.Errorf("validateEntryV: obj#%d invalid entry V, corrupt previous Bead indirect reference", objNumber)
	}

	return nil
}

func validateBeadDict(xRefTable *pdf.XRefTable, beadIndRef, threadIndRef, pBeadIndRef, lBeadIndRef *pdf.IndirectRef) error {

	objNumber := beadIndRef.ObjectNumber.Value()

	dictName := "beadDict"
	sinceVersion := pdf.V10

	dict, err := xRefTable.DereferenceDict(*beadIndRef)
	if err != nil {
		return err
	}
	if dict == nil {
		return errors.Errorf("validateBeadDict: obj#%d missing dict", objNumber)
	}

	// Validate optional entry Type, must be "Bead".
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return err
	}

	// Validate entry T, must refer to threadDict.
	indRefT, err := validateIndRefEntry(xRefTable, dict, dictName, "T", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}
	if indRefT != nil && !indRefT.Equals(*threadIndRef) {
		return errors.Errorf("validateBeadDict: obj#%d invalid entry T (backpointer to ThreadDict)", objNumber)
	}

	// Validate required entry R, must be rectangle.
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Validate required entry P, must be indRef to pageDict.
	err = validateEntryP(xRefTable, dict, dictName, REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// Validate required entry V, must refer to previous bead.
	err = validateEntryV(xRefTable, dict, dictName, REQUIRED, sinceVersion, pBeadIndRef, objNumber)
	if err != nil {
		return err
	}

	// Validate required entry N, must refer to last bead.
	nBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// Recurse until next bead equals last bead.
	if !nBeadIndRef.Equals(*lBeadIndRef) {
		err = validateBeadDict(xRefTable, nBeadIndRef, threadIndRef, beadIndRef, lBeadIndRef)
		if err != nil {
			return err
		}
	}

	return nil
}

func soleBeadDict(beadIndRef, pBeadIndRef, nBeadIndRef *pdf.IndirectRef) bool {
	// if N and V reference this bead dict, must be the first and only one.
	return pBeadIndRef.Equals(*nBeadIndRef) && beadIndRef.Equals(*pBeadIndRef)
}

func validateBeadChainIntegrity(beadIndRef, pBeadIndRef, nBeadIndRef *pdf.IndirectRef) bool {
	return !pBeadIndRef.Equals(*beadIndRef) && !nBeadIndRef.Equals(*beadIndRef)
}

func validateFirstBeadDict(xRefTable *pdf.XRefTable, beadIndRef, threadIndRef *pdf.IndirectRef) error {

	dictName := "firstBeadDict"
	sinceVersion := pdf.V10

	dict, err := xRefTable.DereferenceDict(*beadIndRef)
	if err != nil {
		return err
	}

	if dict == nil {
		return errors.New("validateFirstBeadDict: missing dict")
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return err
	}

	indRefT, err := validateIndRefEntry(xRefTable, dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	if !indRefT.Equals(*threadIndRef) {
		return errors.New("validateFirstBeadDict: invalid entry T (backpointer to ThreadDict)")
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	err = validateEntryP(xRefTable, dict, dictName, REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	pBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	nBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	if soleBeadDict(beadIndRef, pBeadIndRef, nBeadIndRef) {
		return nil
	}

	if !validateBeadChainIntegrity(beadIndRef, pBeadIndRef, nBeadIndRef) {
		return errors.New("validateFirstBeadDict: corrupt chain of beads")
	}

	return validateBeadDict(xRefTable, nBeadIndRef, threadIndRef, beadIndRef, pBeadIndRef)
}

func validateThreadDict(xRefTable *pdf.XRefTable, obj pdf.Object, sinceVersion pdf.Version) error {

	dictName := "threadDict"

	threadIndRef, ok := obj.(pdf.IndirectRef)
	if !ok {
		return errors.New("validateThreadDict: not an indirect ref")
	}

	objNumber := threadIndRef.ObjectNumber.Value()

	dict, err := xRefTable.DereferenceDict(threadIndRef)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return err
	}

	// Validate optional thread information dict entry.
	obj, found := dict.Find("I")
	if found && obj != nil {
		_, err = validateDocumentInfoDict(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	fBeadIndRef := dict.IndirectRefEntry("F")
	if fBeadIndRef == nil {
		return errors.Errorf("validateThreadDict: obj#%d required indirect entry \"F\" missing", objNumber)
	}

	// Validate the list of beads starting with the first bead dict.
	return validateFirstBeadDict(xRefTable, fBeadIndRef, &threadIndRef)
}

func validateThreads(xRefTable *pdf.XRefTable, rootDict *pdf.PDFDict, required bool, sinceVersion pdf.Version) error {

	// => 12.4.3 Articles

	indRef := rootDict.IndirectRefEntry("Threads")
	if indRef == nil {
		if required {
			return errors.New("validateThreads: required entry \"Threads\" missing")
		}
		return nil
	}

	arr, err := xRefTable.DereferenceArray(*indRef)
	if err != nil {
		return err
	}
	if arr == nil {
		return nil
	}

	err = xRefTable.ValidateVersion("threads", sinceVersion)
	if err != nil {
		return err
	}

	for _, obj := range *arr {

		if obj == nil {
			continue
		}

		err = validateThreadDict(xRefTable, obj, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

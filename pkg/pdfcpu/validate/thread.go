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
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validateEntryV(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version, pBeadIndRef *pdf.IndirectRef, objNumber int) error {

	previousBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "V", required, sinceVersion)
	if err != nil {
		return err
	}

	if !previousBeadIndRef.Equals(*pBeadIndRef) {
		return errors.Errorf("pdfcpu: validateEntryV: obj#%d invalid entry V, corrupt previous Bead indirect reference", objNumber)
	}

	return nil
}

func validateBeadDict(xRefTable *pdf.XRefTable, beadIndRef, threadIndRef, pBeadIndRef, lBeadIndRef *pdf.IndirectRef) error {

	objNumber := beadIndRef.ObjectNumber.Value()

	dictName := "beadDict"
	sinceVersion := pdf.V10

	d, err := xRefTable.DereferenceDict(*beadIndRef)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.Errorf("pdfcpu: validateBeadDict: obj#%d missing dict", objNumber)
	}

	// Validate optional entry Type, must be "Bead".
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return err
	}

	// Validate entry T, must refer to threadDict.
	indRefT, err := validateIndRefEntry(xRefTable, d, dictName, "T", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}
	if indRefT != nil && !indRefT.Equals(*threadIndRef) {
		return errors.Errorf("pdfcpu: validateBeadDict: obj#%d invalid entry T (backpointer to ThreadDict)", objNumber)
	}

	// Validate required entry R, must be rectangle.
	_, err = validateRectangleEntry(xRefTable, d, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// Validate required entry P, must be indRef to pageDict.
	err = validateEntryP(xRefTable, d, dictName, REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// Validate required entry V, must refer to previous bead.
	err = validateEntryV(xRefTable, d, dictName, REQUIRED, sinceVersion, pBeadIndRef, objNumber)
	if err != nil {
		return err
	}

	// Validate required entry N, must refer to last bead.
	nBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "N", REQUIRED, sinceVersion)
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

	d, err := xRefTable.DereferenceDict(*beadIndRef)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.New("pdfcpu: validateFirstBeadDict: missing dict")
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return err
	}

	indRefT, err := validateIndRefEntry(xRefTable, d, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	if !indRefT.Equals(*threadIndRef) {
		return errors.New("pdfcpu: validateFirstBeadDict: invalid entry T (backpointer to ThreadDict)")
	}

	_, err = validateRectangleEntry(xRefTable, d, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	err = validateEntryP(xRefTable, d, dictName, REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	pBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	nBeadIndRef, err := validateIndRefEntry(xRefTable, d, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	if soleBeadDict(beadIndRef, pBeadIndRef, nBeadIndRef) {
		return nil
	}

	if !validateBeadChainIntegrity(beadIndRef, pBeadIndRef, nBeadIndRef) {
		return errors.New("pdfcpu: validateFirstBeadDict: corrupt chain of beads")
	}

	return validateBeadDict(xRefTable, nBeadIndRef, threadIndRef, beadIndRef, pBeadIndRef)
}

func validateThreadDict(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	dictName := "threadDict"

	threadIndRef, ok := o.(pdf.IndirectRef)
	if !ok {
		return errors.New("pdfcpu: validateThreadDict: not an indirect ref")
	}

	objNumber := threadIndRef.ObjectNumber.Value()

	d, err := xRefTable.DereferenceDict(threadIndRef)
	if err != nil {
		return err
	}

	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return err
	}

	// Validate optional thread information dict entry.
	o, found := d.Find("I")
	if found && o != nil {
		_, err = validateDocumentInfoDict(xRefTable, o)
		if err != nil {
			return err
		}
	}

	fBeadIndRef := d.IndirectRefEntry("F")
	if fBeadIndRef == nil {
		return errors.Errorf("pdfcpu: validateThreadDict: obj#%d required indirect entry \"F\" missing", objNumber)
	}

	// Validate the list of beads starting with the first bead dict.
	return validateFirstBeadDict(xRefTable, fBeadIndRef, &threadIndRef)
}

func validateThreads(xRefTable *pdf.XRefTable, rootDict pdf.Dict, required bool, sinceVersion pdf.Version) error {

	// => 12.4.3 Articles

	ir := rootDict.IndirectRefEntry("Threads")
	if ir == nil {
		if required {
			return errors.New("pdfcpu: validateThreads: required entry \"Threads\" missing")
		}
		return nil
	}

	a, err := xRefTable.DereferenceArray(*ir)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}

	err = xRefTable.ValidateVersion("threads", sinceVersion)
	if err != nil {
		return err
	}

	for _, o := range a {

		if o == nil {
			continue
		}

		err = validateThreadDict(xRefTable, o, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

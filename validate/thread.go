package validate

import (
	"github.com/EndFirstCorp/pdflib/types"
	"github.com/pkg/errors"
)

func validateBeadDict(xRefTable *types.XRefTable, indRefBeadDict, indRefThreadDict, indRefPreviousBead, indRefLastBead types.PDFIndirectRef) (err error) {

	objNumber := indRefBeadDict.ObjectNumber.Value()

	logInfoValidate.Printf("*** validateBeadDict begin: objectNumber=%d ***", objNumber)

	dictName := "beadDict"
	sinceVersion := types.V10

	dict, err := xRefTable.DereferenceDict(indRefBeadDict)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.Errorf("validateBeadDict: obj#%d missing dict", objNumber)
		return
	}

	// Validate optional entry Type, must be "Bead".
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return
	}

	// Validate entry T, must refer to threadDict.
	indRefT, err := validateIndRefEntry(xRefTable, dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !indRefT.Equals(indRefThreadDict) {
		err = errors.Errorf("validateBeadDict: obj#%d invalid entry T (backpointer to ThreadDict)", objNumber)
		return
	}

	// Validate required entry R, must be rectangle.
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	// Validate required entry P, must be indRef to pageDict.
	pageDict, err := validateDictEntry(xRefTable, dict, dictName, "P", REQUIRED, sinceVersion, nil)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page" {
		return errors.Errorf("validateBeadDict: obj#%d invalid entry P, no page dict", objNumber)
	}

	// Validate required entry V, must refer to previous bead.
	previousBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !previousBeadIndRef.Equals(indRefPreviousBead) {
		err = errors.Errorf("validateBeadDict: obj#%d invalid entry V, corrupt previous Bead indirect reference", objNumber)
		return
	}

	// Validate required entry N, must refer to last bead.
	nextBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	// Recurse until next bead equals last bead.
	if !nextBeadIndRef.Equals(indRefLastBead) {
		err = validateBeadDict(xRefTable, *nextBeadIndRef, indRefThreadDict, indRefBeadDict, indRefLastBead)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Printf("*** validateBeadDict end: objectNumber=%d ***", objNumber)

	return
}

func validateFirstBeadDict(xRefTable *types.XRefTable, indRefBeadDict, indRefThreadDict types.PDFIndirectRef) (err error) {

	logInfoValidate.Printf("*** validateFirstBeadDict begin beadDictObj#%d threadDictObj#%d ***",
		indRefBeadDict.ObjectNumber.Value(), indRefThreadDict.ObjectNumber.Value())

	dictName := "firstBeadDict"
	sinceVersion := types.V10

	dict, err := xRefTable.DereferenceDict(indRefBeadDict)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.New("validateFirstBeadDict: missing dict")
		return
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return
	}

	indRefT, err := validateIndRefEntry(xRefTable, dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !indRefT.Equals(indRefThreadDict) {
		err = errors.New("validateFirstBeadDict: invalid entry T (backpointer to ThreadDict)")
		return
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	pageDict, err := validateDictEntry(xRefTable, dict, dictName, "P", REQUIRED, sinceVersion, nil)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page" {
		return errors.New("validateFirstBeadDict: invalid page dict")
	}

	previousBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	nextBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	// if N and V reference same bead dict, must be the first and only one.
	if previousBeadIndRef.Equals(*nextBeadIndRef) {
		if !indRefBeadDict.Equals(*previousBeadIndRef) {
			err = errors.New("validateFirstBeadDict: corrupt chain of beads")
			return
		}
		logInfoValidate.Println("*** validateFirstBeadDict end single bead ***")
		return
	}

	err = validateBeadDict(xRefTable, *nextBeadIndRef, indRefThreadDict, indRefBeadDict, *previousBeadIndRef)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateFirstBeadDict end ***")

	return
}

func validateThreadDict(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateThreadDict begin ***")

	dictName := "threadDict"

	indRefThreadDict, ok := obj.(types.PDFIndirectRef)
	if !ok {
		err = errors.New("validateThreadDict: not an indirect ref")
		return
	}

	objNumber := indRefThreadDict.ObjectNumber.Value()

	dict, err := xRefTable.DereferenceDict(indRefThreadDict)
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return
	}

	// Validate optional thread information dict entry.
	obj, found := dict.Find("I")
	if found && obj != nil {
		_, err = validateDocumentInfoDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	firstBeadDict := dict.IndirectRefEntry("F")
	if firstBeadDict == nil {
		err = errors.Errorf("validateThreadDict: obj#%d required indirect entry \"F\" missing", objNumber)
		return
	}

	// Validate the list of beads starting with the first bead dict.
	err = validateFirstBeadDict(xRefTable, *firstBeadDict, indRefThreadDict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateThreadDict end ***")

	return
}

func validateThreads(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.4.3 Articles

	logInfoValidate.Println("*** validateThreads begin ***")

	indRef := rootDict.IndirectRefEntry("Threads")
	if indRef == nil {
		if required {
			err = errors.New("validateThreads: required entry \"Threads\" missing")
			return
		}
		logInfoValidate.Println("validateThreads end: object is nil.")
		return
	}

	arr, err := xRefTable.DereferenceArray(*indRef)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validateThreads end: object is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateThreads: unsupported in version %s", xRefTable.VersionString())
	}

	for _, obj := range *arr {

		if obj == nil {
			continue
		}

		err = validateThreadDict(xRefTable, obj, sinceVersion)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateThreads end ***")

	return
}

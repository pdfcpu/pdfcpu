package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateCreationDate(xRefTable *types.XRefTable, o interface{}) (err error) {

	if xRefTable.ValidationMode == types.ValidationRelaxed {
		_, err = validateString(xRefTable, o, nil)
	} else {
		_, err = validateDateObject(xRefTable, o, types.V10)
	}

	return err
}

func handleDefault(xRefTable *types.XRefTable, o interface{}) (err error) {

	if xRefTable.ValidationMode == types.ValidationStrict {
		_, err = xRefTable.DereferenceStringOrHexLiteral(o, types.V10, nil)
	} else {
		_, err = xRefTable.Dereference(o)
	}

	return err
}

func validateDocumentInfoDict(xRefTable *types.XRefTable, obj interface{}) (hasModDate bool, err error) {

	// Document info object is optional.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return false, err
	}

	for k, v := range dict.Dict {

		switch k {

		// text string, opt, since V1.1
		case "Title":
			_, err = xRefTable.DereferenceStringOrHexLiteral(v, types.V11, nil)

		// text string, optional
		case "Author":
			_, err = xRefTable.DereferenceStringOrHexLiteral(v, types.V10, nil)

		// text string, optional, since V1.1
		case "Subject":
			_, err = xRefTable.DereferenceStringOrHexLiteral(v, types.V11, nil)

		// text string, optional, since V1.1
		case "Keywords":
			_, err = xRefTable.DereferenceStringOrHexLiteral(v, types.V11, nil)

		// text string, optional
		case "Creator":
			_, err = xRefTable.DereferenceStringOrHexLiteral(v, types.V10, nil)

		// text string, optional
		case "Producer":
			_, err = xRefTable.DereferenceStringOrHexLiteral(v, types.V10, nil)

		// date, optional
		case "CreationDate":
			err = validateCreationDate(xRefTable, v)

		// date, required if PieceInfo is present in document catalog.
		case "ModDate":
			hasModDate = true
			_, err = validateDateObject(xRefTable, v, types.V10)

		// name, optional, since V1.3
		case "Trapped":
			_, err = xRefTable.DereferenceName(v, types.V13, validateDocInfoDictTrapped)

		// text string, optional
		default:
			err = handleDefault(xRefTable, v)

		}

		if err != nil {
			return false, err
		}

	}

	return hasModDate, nil
}

func validateDocumentInfoObject(xRefTable *types.XRefTable) error {

	logInfoValidate.Println("*** validateDocumentInfoObject begin ***")

	// Document info object is optional.
	if xRefTable.Info == nil {
		return nil
	}

	hasModDate, err := validateDocumentInfoDict(xRefTable, *xRefTable.Info)
	if err != nil {
		return err
	}

	hasPieceInfo, err := xRefTable.CatalogHasPieceInfo()
	if err != nil {
		return err
	}

	if hasPieceInfo && !hasModDate {
		return errors.Errorf("validateDocumentInfoObject: missing required entry \"ModDate\"")
	}

	logInfoValidate.Println("*** validateDocumentInfoObject end ***")

	return nil
}

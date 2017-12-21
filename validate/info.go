package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateCreationDate(xRefTable *types.XRefTable, o interface{}) (err error) {

	if xRefTable.ValidationMode == types.ValidationRelaxed {
		_, err = validateString(xRefTable, o, nil)
		if err != nil {
			return
		}
	} else {
		_, err = validateDateObject(xRefTable, o, types.V10)
		if err != nil {
			return
		}
	}

	return
}

func handleDefault(xRefTable *types.XRefTable, o interface{}) (err error) {

	if xRefTable.ValidationMode == types.ValidationStrict {
		_, err = xRefTable.DereferenceStringOrHexLiteral(o, types.V10, nil)
	} else {
		_, err = xRefTable.Dereference(o)
	}

	return
}

func validateDocumentInfoDict(xRefTable *types.XRefTable, obj interface{}) (hasModDate bool, err error) {

	// Document info object is optional.

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return
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
			return
		}

	}

	return
}

func validateDocumentInfoObject(xRefTable *types.XRefTable) (err error) {

	logInfoValidate.Println("*** validateDocumentInfoObject begin ***")

	// Document info object is optional.
	if xRefTable.Info == nil {
		return
	}

	hasModDate, err := validateDocumentInfoDict(xRefTable, *xRefTable.Info)
	if err != nil {
		return
	}

	hasPieceInfo, err := xRefTable.CatalogHasPieceInfo()
	if err != nil {
		return
	}

	if hasPieceInfo && !hasModDate {
		err = errors.Errorf("validateDocumentInfoObject: missing required entry \"ModDate\"")
		return
	}

	logInfoValidate.Println("*** validateDocumentInfoObject end ***")

	return
}

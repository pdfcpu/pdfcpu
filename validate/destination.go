package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateDestinationArrayFirstElement(xRefTable *types.XRefTable, arr *types.PDFArray) (types.PDFObject, error) {

	obj, err := xRefTable.Dereference((*arr)[0])
	if err != nil || obj == nil {
		return nil, err
	}

	switch obj := obj.(type) {

	case types.PDFInteger: // no further processing

	case types.PDFDict:
		if obj.Type() == nil || *obj.Type() != "Page" {
			err = errors.New("validateDestinationArrayFirstElement: first element refers to invalid destination page dict")
		}

	default:
		err = errors.Errorf("validateDestinationArrayFirstElement: first element must be a pageDict indRef or an integer: %v", obj)
	}

	return obj, err
}

func validateDestinationArrayLength(arr types.PDFArray) bool {
	l := len(arr)
	return l == 2 || l == 3 || l == 5 || l == 6
}

func validateDestinationArray(xRefTable *types.XRefTable, arr *types.PDFArray) error {

	// Validate first element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.

	obj, err := validateDestinationArrayFirstElement(xRefTable, arr)
	if err != nil || obj == nil {
		return err
	}

	if !validateDestinationArrayLength(*arr) {
		return errors.New("validateDestinationArray: invalid length")
	}

	// Validate rest of array elements.

	name, ok := (*arr)[1].(types.PDFName)
	if !ok {
		return errors.New("validateDestinationArray: second element must be a name")
	}

	var nameErr bool

	switch len(*arr) {

	case 2:
		if xRefTable.ValidationMode == types.ValidationRelaxed {
			nameErr = !memberOf(name.Value(), []string{"Fit", "FitB", "FitH"})
		} else {
			nameErr = !memberOf(name.Value(), []string{"Fit", "FitB"})
		}

	case 3:
		nameErr = name.Value() != "FitH" && name.Value() != "FitV" && name.Value() != "FitBH"

	case 5:
		nameErr = name.Value() != "XYZ"

	case 6:
		nameErr = name.Value() != "FitR"

	default:
		return errors.Errorf("validateDestinationArray: array length %d not allowed: %v", len(*arr), arr)
	}

	if nameErr {
		return errors.New("validateDestinationArray: arr[1] corrupt")
	}

	return nil
}

func validateDestinationDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// D, required, array
	arr, err := validateArrayEntry(xRefTable, dict, "DestinationDict", "D", REQUIRED, types.V10, nil)
	if err != nil || arr == nil {
		return err
	}

	return validateDestinationArray(xRefTable, arr)
}

func validateDestination(xRefTable *types.XRefTable, obj types.PDFObject) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case types.PDFName:
		// no further processing.

	case types.PDFStringLiteral:
		// no further processing.

	case types.PDFDict:
		err = validateDestinationDict(xRefTable, &obj)

	case types.PDFArray:
		err = validateDestinationArray(xRefTable, &obj)

	default:
		err = errors.New("validateDestination: unsupported PDF object")

	}

	return err
}

func validateDestinationEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	// see 12.3.2

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil {
		return err
	}

	return validateDestination(xRefTable, obj)
}

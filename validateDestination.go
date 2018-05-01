package pdfcpu

import (
	"github.com/pkg/errors"
)

func validateDestinationArrayFirstElement(xRefTable *XRefTable, arr *PDFArray) (PDFObject, error) {

	obj, err := xRefTable.Dereference((*arr)[0])
	if err != nil || obj == nil {
		return nil, err
	}

	switch obj := obj.(type) {

	case PDFInteger: // no further processing

	case PDFDict:
		if obj.Type() == nil || *obj.Type() != "Page" {
			err = errors.New("validateDestinationArrayFirstElement: first element refers to invalid destination page dict")
		}

	default:
		err = errors.Errorf("validateDestinationArrayFirstElement: first element must be a pageDict indRef or an integer: %v", obj)
	}

	return obj, err
}

func validateDestinationArrayLength(arr PDFArray) bool {
	l := len(arr)
	return l == 2 || l == 3 || l == 5 || l == 6
}

func validateDestinationArray(xRefTable *XRefTable, arr *PDFArray) error {

	// Validate first element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.

	obj, err := validateDestinationArrayFirstElement(xRefTable, arr)
	if err != nil || obj == nil {
		return err
	}

	if !validateDestinationArrayLength(*arr) {
		return errors.New("validateDestinationArray: invalid length")
	}

	// Validate rest of array elements.

	name, ok := (*arr)[1].(PDFName)
	if !ok {
		return errors.New("validateDestinationArray: second element must be a name")
	}

	var nameErr bool

	switch len(*arr) {

	case 2:
		if xRefTable.ValidationMode == ValidationRelaxed {
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

func validateDestinationDict(xRefTable *XRefTable, dict *PDFDict) error {

	// D, required, array
	arr, err := validateArrayEntry(xRefTable, dict, "DestinationDict", "D", REQUIRED, V10, nil)
	if err != nil || arr == nil {
		return err
	}

	return validateDestinationArray(xRefTable, arr)
}

func validateDestination(xRefTable *XRefTable, obj PDFObject) error {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}

	switch obj := obj.(type) {

	case PDFName:
		// no further processing.

	case PDFStringLiteral:
		// no further processing.

	case PDFDict:
		err = validateDestinationDict(xRefTable, &obj)

	case PDFArray:
		err = validateDestinationArray(xRefTable, &obj)

	default:
		err = errors.New("validateDestination: unsupported PDF object")

	}

	return err
}

func validateDestinationEntry(xRefTable *XRefTable, dict *PDFDict, dictName string, entryName string, required bool, sinceVersion PDFVersion) error {

	// see 12.3.2

	obj, err := validateEntry(xRefTable, dict, dictName, entryName, required, sinceVersion)
	if err != nil {
		return err
	}

	return validateDestination(xRefTable, obj)
}

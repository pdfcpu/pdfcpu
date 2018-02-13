package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validateDestinationArrayFirstElement(xRefTable *types.XRefTable, arr *types.PDFArray) (interface{}, error) {

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

	logInfoValidate.Println("*** validateDestinationArray: begin ***")

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

	logInfoValidate.Println("*** validateDestinationArray: end ***")

	return nil
}

func validateDestinationDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	logInfoValidate.Println("*** validateDestinationDict: begin ***")

	arr, err := validateArrayEntry(xRefTable, dict, "DestinationDict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}
	if arr == nil {
		logInfoValidate.Println("validateDestinationDict: arr is nil end")
		return nil
	}

	err = validateDestinationArray(xRefTable, arr)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateDestinationDict: end ***")

	return nil
}

func validateDestination(xRefTable *types.XRefTable, obj interface{}) error {

	logInfoValidate.Println("*** validateDestination: begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Println("validateDestination: is nil, end")
		return nil
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

	if err == nil {
		logInfoValidate.Println("*** validateDestination: end ***")
	}

	return err
}

func validateDestinationEntry(
	xRefTable *types.XRefTable,
	dict *types.PDFDict,
	dictName string,
	entryName string,
	required bool,
	sinceVersion types.PDFVersion,
	validate func(interface{}) bool) error {

	// see 12.3.2

	logInfoValidate.Printf("*** validateDestinationEntry begin: entry=%s ***\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateDestinationEntry: dict=%s required entry=%s missing", dictName, entryName)
		}
		logInfoValidate.Printf("validateDestinationEntry end: entry %s is nil\n", entryName)
		return nil
	}

	err := validateDestination(xRefTable, obj)
	if err != nil {
		return err
	}

	logInfoValidate.Printf("*** validateDestinationEntry end: entry=%s ***\n", entryName)

	return nil
}

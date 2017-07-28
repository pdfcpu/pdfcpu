package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validateDestinationArray(xRefTable *types.XRefTable, arr *types.PDFArray) error {

	logInfoValidate.Println("*** validateDestinationArray: begin ***")

	// First element: indRef of page dict or pageNumber(int) of remote doc for remote Go-to Action or nil.
	obj, err := xRefTable.Dereference((*arr)[0])
	if err != nil {
		return err
	}

	if obj == nil {
		return nil
	}

	switch obj := obj.(type) {

	case types.PDFInteger: // no further processing

	case types.PDFDict:
		if obj.Type() == nil || *obj.Type() != "Page" {
			return errors.New("validateDestinationArray: first element refers to invalid destination page dict")
		}

	default:
		return errors.Errorf("validateDestinationArray: first element must be a pageDict indRef or an integer: %v", obj)
	}

	switch len(*arr) {

	case 2:
		name, ok := (*arr)[1].(types.PDFName)
		if !ok {
			return errors.New("validateDestinationArray: second element must be a name")
		}
		if xRefTable.ValidationMode == types.ValidationRelaxed {
			if !memberOf(name.Value(), []string{"Fit", "FitB", "FitH"}) {
				return errors.New("validateDestinationArray: arr[1] corrupt")
			}
		} else {
			if !memberOf(name.Value(), []string{"Fit", "FitB"}) {
				return errors.New("validateDestinationArray: arr[1] corrupt")
			}
		}

	case 3:
		name, ok := (*arr)[1].(types.PDFName)
		if !ok {
			return errors.New("validateDestinationArray: arr[1] must be a name")
		}
		if name.Value() != "FitH" && name.Value() != "FitV" && name.Value() != "FitBH" {
			return errors.New("validateDestinationArray: arr[1] corrupt")
		}

	case 5:
		name, ok := (*arr)[1].(types.PDFName)
		if !ok {
			return errors.New("validateDestinationArray: arr[1] must be a name")
		}
		if name.Value() != "XYZ" {
			return errors.New("validateDestinationArray: arr[1] corrupt")
		}

	case 6:

		name, ok := (*arr)[1].(types.PDFName)
		if !ok {
			return errors.New("validateDestinationArray: arr[1] must be a name")
		}
		if name.Value() != "FitR" {
			return errors.New("validateDestinationArray: arr[1] corrupt")
		}

	default:
		return errors.Errorf("validateDestinationArray: array length %d not alowed: %v", len(*arr), arr)
	}

	logInfoValidate.Println("*** validateDestinationArray: end ***")

	return nil
}

func validateDestinationDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateDestinationDict: begin ***")

	arr, err := validateArrayEntry(xRefTable, dict, "DestinationDict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validateDestinationDict: arr is nil end")
		return
	}

	err = validateDestinationArray(xRefTable, arr)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateDestinationDict: end ***")

	return
}

func validateDestination(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateDestination: begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateDestination: is nil, end")
		return
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

	return
}

func validateDestinationEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(interface{}) bool) (err error) {

	// see 12.3.2

	logInfoValidate.Printf("*** validateDestinationEntry begin: entry=%s ***\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateDestinationEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDestinationEntry end: entry %s is nil\n", entryName)
		return
	}

	err = validateDestination(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Printf("*** validateDestinationEntry end: entry=%s ***\n", entryName)

	return
}

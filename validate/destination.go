package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

// TODO actually write something!
func validateDestinationArray(xRefTable *types.XRefTable, arr *types.PDFArray) (err error) {

	logInfoValidate.Println("*** validateDestinationArray: begin ***")

	// Relaxed: Allow nil for Page for all actions, (according to spec only for remote Goto actions nil allowed)
	//if arr[0] == nil {
	//	return false, errors.New("writeDestinationArray end: arr[0] (PageIndRef) is null")
	//}

	indRef, ok := (*arr)[0].(types.PDFIndirectRef)
	if !ok {
		// TODO: log.Fatalln("writeDestinationArray: destination array[0] not an indirect ref.")
		logInfoValidate.Println("validateDestinationArray end: arr[0] is no indRef")
		return
	}

	pageDict, err := xRefTable.DereferenceDict(indRef)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page" {
		return errors.Errorf("validateDestinationArray: invalid destination page dict. for obj#%d", indRef.ObjectNumber)
	}

	logInfoValidate.Println("*** validateDestinationArray: end ***")

	return
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
		//ok = true

	case types.PDFStringLiteral:
		// no further processing.
		//ok = true

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

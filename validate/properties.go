package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validatePropertiesDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 14.6.2
	// a dictionary containing private information meaningful to the conforming writer creating marked content.

	// anything possible +

	// empty dict ok
	// Optional Metadata entry ok
	// Optional Contents entry ok
	// Optional Resources entry ok

	// Optional content group /OCG see 8.11.2
	// Optional content membership dict. /OCMD see 8.11.2.2
	// Optional MCID integer entry
	// Optional Alt since 1.5 see 14.9.3
	// Optional ActualText since 1.5 see 14.9.4
	// Optional E see since 1.4 14.9.5
	// Optional Lang string RFC 3066 see 14.9.2

	logInfoValidate.Println("*** validatePropertiesDict begin ***")

	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	err = validateMetadata(xRefTable, dict, OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	for key, val := range dict.Dict {

		logInfoValidate.Printf("validatePropertiesDict: key=%s val=%v\n", key, val)

		switch key {

		case "Metadata":
			logInfoValidate.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			// see above

		case "Contents":
			logInfoValidate.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			//indRef, ok := val.(PDFIndirectRef)
			//if !ok {
			//	log.Fatalf("writePropertiesDict: currupt entry \"%s\"\n", key)
			//}
			_, err = validateStreamDict(xRefTable, val)
			if err != nil {
				return
			}

		case "Resources":
			logInfoValidate.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			_, err = validateResourceDict(xRefTable, val)
			if err != nil {
				return
			}

		case "OCG":
			return errors.Errorf("validatePropertiesDict: recognized unsupported key \"%s\"\n", key)

		case "OCMD":
			return errors.Errorf("validatePropertiesDict: recognized unsupported key \"%s\"\n", key)

		//case "MCID": -> default
		//case "Alt": -> default
		//case "ActualText": -> default
		//case "E": -> default
		//case "Lang": -> default

		default:
			logInfoValidate.Printf("validatePropertiesDict: processing unrecognized key \"%s\"\n", key)
			_, err = xRefTable.Dereference(val)
			if err != nil {
				return
			}
		}

	}

	logInfoValidate.Println("*** validatePropertiesDict end ***")

	return
}

func validatePropertiesResourceDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validatePropertiesResourceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validatePropertiesResourceDict end: object is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < types.V12 {
		return errors.Errorf("validatePropertiesResourceDict: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// Iterate over properties resource dict
	for _, obj := range dict.Dict {

		dict, err = xRefTable.DereferenceDict(obj)
		if err != nil {
			return
		}

		if dict == nil {
			logInfoValidate.Println("validatePropertiesResourceDict end: resource object is nil.")
			continue
		}

		// Process propDict
		err = validatePropertiesDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validatePropertiesResourceDict end ***")

	return
}

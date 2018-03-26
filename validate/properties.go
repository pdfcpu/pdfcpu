package validate

import (
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func validatePropertiesDict(xRefTable *types.XRefTable, obj types.PDFObject) error {

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

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || obj == nil {
		return err
	}

	err = validateMetadata(xRefTable, dict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	for key, val := range dict.Dict {

		log.Debug.Printf("validatePropertiesDict: key=%s val=%v\n", key, val)

		switch key {

		case "Metadata":
			log.Debug.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			// see above

		case "Contents":
			log.Debug.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			_, err = validateStreamDict(xRefTable, val)
			if err != nil {
				return err
			}

		case "Resources":
			log.Debug.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			_, err = validateResourceDict(xRefTable, val)
			if err != nil {
				return err
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
			log.Debug.Printf("validatePropertiesDict: processing unrecognized key \"%s\"\n", key)
			_, err = xRefTable.Dereference(val)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validatePropertiesResourceDict(xRefTable *types.XRefTable, obj types.PDFObject, sinceVersion types.PDFVersion) error {

	// Version check
	err := xRefTable.ValidateVersion("PropertiesResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	// Iterate over properties resource dict
	for _, obj := range dict.Dict {

		// Process propDict
		err = validatePropertiesDict(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

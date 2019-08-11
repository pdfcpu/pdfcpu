/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

func validatePropertiesDict(xRefTable *pdf.XRefTable, o pdf.Object) error {

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

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	err = validateMetadata(xRefTable, d, OPTIONAL, pdf.V14)
	if err != nil {
		return err
	}

	for key, val := range d {

		log.Validate.Printf("validatePropertiesDict: key=%s val=%v\n", key, val)

		switch key {

		case "Metadata":
			log.Validate.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			// see above

		case "Contents":
			log.Validate.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
			_, err = validateStreamDict(xRefTable, val)
			if err != nil {
				return err
			}

		case "Resources":
			log.Validate.Printf("validatePropertiesDict: recognized key \"%s\"\n", key)
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
			log.Validate.Printf("validatePropertiesDict: processing unrecognized key \"%s\"\n", key)
			_, err = xRefTable.Dereference(val)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func validatePropertiesResourceDict(xRefTable *pdf.XRefTable, o pdf.Object, sinceVersion pdf.Version) error {

	// Version check
	err := xRefTable.ValidateVersion("PropertiesResourceDict", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Iterate over properties resource dict
	for _, o := range d {

		// Process propDict
		err = validatePropertiesDict(xRefTable, o)
		if err != nil {
			return err
		}
	}

	return nil
}

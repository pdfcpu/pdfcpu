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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func validatePropertiesDict(xRefTable *model.XRefTable, o types.Object) error {
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

	logProp := func(qual, k string, v types.Object) {
		if log.ValidateEnabled() {
			log.Validate.Printf("validatePropertiesDict: %s key=%s val=%v\n", qual, k, v)
		}
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	if err = validateMetadata(xRefTable, d, OPTIONAL, model.V14); err != nil {
		return err
	}

	for key, val := range d {

		switch key {

		case "Metadata":
			logProp("known", key, val)

		case "Contents":
			logProp("known", key, val)
			if _, err = validateStreamDict(xRefTable, val); err != nil {
				return err
			}

		case "Resources":
			logProp("known", key, val)
			if _, err = validateResourceDict(xRefTable, val); err != nil {
				return err
			}

		case "OCG":
			logProp("unsupported", key, val)
			return errors.Errorf("validatePropertiesDict: unsupported key \"%s\"\n", key)

		case "OCMD":
			logProp("unsupported", key, val)
			return errors.Errorf("validatePropertiesDict: unsupported key \"%s\"\n", key)

		//case "MCID": -> default
		//case "Alt": -> default
		//case "ActualText": -> default
		//case "E": -> default
		//case "Lang": -> default

		default:
			logProp("unknown", key, val)
			if _, err = xRefTable.Dereference(val); err != nil {
				return err
			}
		}

	}

	return nil
}

func validatePropertiesResourceDict(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	if err := xRefTable.ValidateVersion("PropertiesResourceDict", sinceVersion); err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	// Iterate over properties resource dict
	for _, o := range d {
		if err = validatePropertiesDict(xRefTable, o); err != nil {
			return err
		}
	}

	return nil
}

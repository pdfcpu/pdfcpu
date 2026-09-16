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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type propertiesTraversalContextKey struct{}

type propertiesTraversal struct {
	ancestors map[int]bool
}

func propertiesTraversalFromContext(c context.Context) (context.Context, *propertiesTraversal) {
	if c != nil {
		if t, ok := c.Value(propertiesTraversalContextKey{}).(*propertiesTraversal); ok {
			return c, t
		}
	}
	t := &propertiesTraversal{ancestors: map[int]bool{}}
	if c == nil {
		return nil, t
	}
	return context.WithValue(c, propertiesTraversalContextKey{}, t), t
}

func propertiesObjectIdentity(o types.Object) int {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0
	}
	return ir.ObjectNumber.Value()
}

func validatePropertiesDict(c context.Context, xRefTable *model.XRefTable, o types.Object) error {
	c, traversal := propertiesTraversalFromContext(c)
	return traversal.validate(c, xRefTable, o)
}

func (t *propertiesTraversal) enter(o types.Object) (int, bool) {
	objNr := propertiesObjectIdentity(o)
	if objNr <= 0 {
		return 0, true
	}
	if t.ancestors[objNr] {
		return objNr, false
	}
	t.ancestors[objNr] = true
	return objNr, true
}

func (t *propertiesTraversal) validate(c context.Context, xRefTable *model.XRefTable, o types.Object) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()
	if err := contextutil.Check(c); err != nil {
		return err
	}

	propertyObjNr, entered := t.enter(o)
	if !entered {
		return nil
	}
	if propertyObjNr > 0 {
		defer delete(t.ancestors, propertyObjNr)
	}

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

	d, streamDict, err := dereferencePropertiesDict(xRefTable, o)
	if err != nil || d == nil {
		if err != nil {
			return fmt.Errorf("%s: dereference: %w", objectContext("propertiesDict", o), err)
		}
		return nil
	}

	if err = validateMetadata(xRefTable, d, OPTIONAL, model.V14); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext("propertiesDict", "Metadata", d["Metadata"]), err)
	}
	if err = validatePropertiesDictEntries(c, xRefTable, d, objNr); err != nil {
		return err
	}
	if streamDict {
		err := errors.New("property list stream dictionary accepted in relaxed mode")
		model.ShowDigestedSpecViolationError(model.WithValidationErrorObject(err, objNr))
	}
	return nil
}

func dereferencePropertiesDict(xRefTable *model.XRefTable, o types.Object) (types.Dict, bool, error) {
	resolvedObject, err := xRefTable.Dereference(o)
	if err != nil || resolvedObject == nil {
		return nil, false, err
	}
	if d, ok := resolvedObject.(types.Dict); ok {
		return d, false, nil
	}
	if d, ok := resolvedObject.(types.StreamDict); ok && xRefTable.ValidationMode == model.ValidationRelaxed {
		return d.Dict, true, nil
	}
	_, err = xRefTable.DereferenceDict(o)
	return nil, false, err
}

func validatePropertiesDictEntries(c context.Context, xRefTable *model.XRefTable, d types.Dict, objNr int) (err error) {
	logProp := func(qual, k string, v types.Object) {
		if log.ValidateEnabled() {
			log.Validate.Printf("validatePropertiesDict: %s key=%s val=%v\n", qual, k, v)
		}
	}

	for _, key := range slices.Sorted(maps.Keys(d)) {
		val := d[key]

		switch key {

		case "Metadata":
			logProp("known", key, val)

		case "Contents":
			logProp("known", key, val)
			if err = validateStringOrStreamEntry(
				xRefTable, d, objNr, "propertiesDict", "Contents", OPTIONAL, model.V10,
			); err != nil {
				return fmt.Errorf("%s: %w", dictEntryContext("propertiesDict", "Contents", val), err)
			}

		case "Resources":
			logProp("known", key, val)
			if _, err = validateResourceDict(c, xRefTable, val); err != nil {
				return fmt.Errorf("%s: %w", dictEntryContext("propertiesDict", "Resources", val), err)
			}

		case "OCG":
			logProp("unsupported", key, val)
			return fmt.Errorf("%s: unsupported key", dictEntryContext("propertiesDict", key, val))

		case "OCMD":
			logProp("unsupported", key, val)
			return fmt.Errorf("%s: unsupported key", dictEntryContext("propertiesDict", key, val))

		//case "MCID": -> default
		//case "Alt": -> default
		//case "ActualText": -> default
		//case "E": -> default
		//case "Lang": -> default

		default:
			logProp("unknown", key, val)
			if _, err = xRefTable.Dereference(val); err != nil {
				err = fmt.Errorf("%s: dereference: %w", dictEntryContext("propertiesDict", key, val), err)
				return model.WithValidationErrorObject(err, validationObjectNumber(objNr, val))
			}
		}

	}

	return nil
}

func validatePropertiesResourceDict(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	if err := xRefTable.ValidateVersion("PropertiesResourceDict", sinceVersion); err != nil {
		return fmt.Errorf("propertiesResourceDict: %w", err)
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		if err != nil {
			return fmt.Errorf("%s: dereference: %w", objectContext("propertiesResourceDict", o), err)
		}
		return nil
	}
	c, _ = propertiesTraversalFromContext(c)

	// Iterate over properties resource dict
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		o := d[name]
		if err = validatePropertiesDict(c, xRefTable, o); err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("propertiesResourceDict.%s", name), o), err)
		}
	}

	return nil
}

/*
Copyright 2023 The pdfcpu Authors.

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

package pdfcpu

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// formFieldSelection tracks source widget annotations selected during page migration.
type formFieldSelection struct {
	seen    map[int]bool
	widgets []types.IndirectRef
}

func newFormFieldSelection() *formFieldSelection {
	return &formFieldSelection{seen: map[int]bool{}}
}

func (s *formFieldSelection) addWidget(indRef types.IndirectRef) {
	objNr := indRef.ObjectNumber.Value()
	if s.seen[objNr] {
		return
	}
	s.seen[objNr] = true
	s.widgets = append(s.widgets, indRef)
}

func migrateIndRef(ir *types.IndirectRef, ctxSource, ctxDest *model.Context, migrated map[int]int) (types.Object, error) {
	o, err := ctxSource.Dereference(*ir)
	if err != nil {
		return nil, err
	}

	if o != nil {
		o = o.Clone()
	}

	objNrNew, err := ctxDest.InsertObject(o)
	if err != nil {
		return nil, err
	}

	objNr := ir.ObjectNumber.Value()
	migrated[objNr] = objNrNew
	ir.ObjectNumber = types.Integer(objNrNew)
	return o, nil
}

func migrateObject(o types.Object, ctxSource, ctxDest *model.Context, migrated map[int]int) (types.Object, error) {
	var err error
	switch o := o.(type) {
	case types.IndirectRef:
		objNr := o.ObjectNumber.Value()
		if migrated[objNr] > 0 {
			o.ObjectNumber = types.Integer(migrated[objNr])
			return o, nil
		}
		o1, err := migrateIndRef(&o, ctxSource, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		if _, err := migrateObject(o1, ctxSource, ctxDest, migrated); err != nil {
			return nil, err
		}
		return o, nil

	case types.Dict:
		for k, v := range o {
			if o[k], err = migrateObject(v, ctxSource, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
		return o, nil

	case types.StreamDict:
		for k, v := range o.Dict {
			if o.Dict[k], err = migrateObject(v, ctxSource, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
		return o, nil

	case types.Array:
		for k, v := range o {
			if o[k], err = migrateObject(v, ctxSource, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
		return o, nil
	}

	return o, nil
}

func migrateAnnotEntry(
	o types.Object,
	index int,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, types.Dict, *types.IndirectRef, bool, error) {
	indRef, ok := o.(types.IndirectRef)
	if !ok {
		d, ok := o.(types.Dict)
		if !ok {
			return nil, nil, nil, false, fmt.Errorf("annotation entry %d: wrong type %T", index, o)
		}
		return o, d, nil, false, nil
	}

	sourceIndRef := indRef
	objNr := indRef.ObjectNumber.Value()
	if migrated[objNr] > 0 {
		indRef.ObjectNumber = types.Integer(migrated[objNr])
		return indRef, nil, nil, true, nil
	}
	o, err := migrateIndRef(&indRef, ctxSrc, ctxDest, migrated)
	if err != nil {
		return nil, nil, nil, false, err
	}
	d, ok := o.(types.Dict)
	if !ok {
		return nil, nil, nil, false, fmt.Errorf("annotation obj#%d: wrong type %T", objNr, o)
	}
	return indRef, d, &sourceIndRef, false, nil
}

func migrateAnnots(
	o types.Object,
	pageIndRef types.IndirectRef,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
	selection *formFieldSelection,
) (types.Object, error) {
	arr, ok := o.(types.Array)
	if !ok {
		return nil, fmt.Errorf("annotations: wrong type %T", o)
	}
	for i, v := range arr {
		o, d, sourceIndRef, done, err := migrateAnnotEntry(v, i, ctxSrc, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		arr[i] = o
		if done {
			continue
		}
		subtype, _, err := ctxSrc.DereferenceNameEntry(d, "Subtype")
		if err != nil {
			return nil, fmt.Errorf("annotation entry %d Subtype: %w", i, err)
		}
		isWidget := subtype != nil && subtype.Value() == "Widget"
		if isWidget && sourceIndRef != nil {
			selection.addWidget(*sourceIndRef)
		}
		for k, v := range d {
			if k == "P" {
				d["P"] = pageIndRef
				continue
			}
			if k == "Parent" && isWidget {
				d.Delete("Parent")
				continue
			}
			o1, err := migrateObject(v, ctxSrc, ctxDest, migrated)
			if err != nil {
				return nil, err
			}
			d[k] = o1
		}
	}

	return arr, nil
}

func migratePageDict(
	d types.Dict,
	pageIndRef types.IndirectRef,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
	selection *formFieldSelection,
) error {
	if err := requireContextWithXRefTable(ctxSrc); err != nil {
		return fmt.Errorf("source context: %w", err)
	}
	if err := requireContextWithXRefTable(ctxDest); err != nil {
		return fmt.Errorf("destination context: %w", err)
	}
	if migrated == nil {
		return fmt.Errorf("missing migration map")
	}
	if selection == nil {
		return fmt.Errorf("missing form field selection")
	}

	var err error
	for k, v := range d {
		if k == "Parent" {
			continue
		}
		if k == "Annots" {
			o, ok := d[k].(types.IndirectRef)
			if ok {
				objNr := o.ObjectNumber.Value()
				if migrated[objNr] > 0 {
					o.ObjectNumber = types.Integer(migrated[objNr])
					d[k] = o
					continue
				}
				v, err = migrateIndRef(&o, ctxSrc, ctxDest, migrated)
				if err != nil {
					return fmt.Errorf("page dict entry %s: migrate annotation reference: %w", k, err)
				}
				d[k] = o
				if _, err = migrateAnnots(v, pageIndRef, ctxSrc, ctxDest, migrated, selection); err != nil {
					return fmt.Errorf("page dict entry %s: migrate annotations: %w", k, err)
				}
				continue
			}
			if d[k], err = migrateAnnots(v, pageIndRef, ctxSrc, ctxDest, migrated, selection); err != nil {
				return fmt.Errorf("page dict entry %s: migrate annotations: %w", k, err)
			}
			continue
		}
		if d[k], err = migrateObject(v, ctxSrc, ctxDest, migrated); err != nil {
			return fmt.Errorf("page dict entry %s: migrate object: %w", k, err)
		}
	}
	return nil
}

type formFieldMigration struct {
	ctxSrc, ctxDest *model.Context
	migrated        map[int]int
	fields          map[int]types.IndirectRef
	sources         map[int]types.IndirectRef
	roots           map[int]bool
}

func migratedFieldRef(indRef types.IndirectRef, migrated map[int]int) *types.IndirectRef {
	objNr := migrated[indRef.ObjectNumber.Value()]
	if objNr == 0 {
		return nil
	}
	indRef.ObjectNumber = types.Integer(objNr)
	return &indRef
}

func (fm *formFieldMigration) cloneField(indRef types.IndirectRef, dSrc types.Dict) (types.IndirectRef, error) {
	dDest := dSrc.Clone().(types.Dict)
	dDest.Delete("Kids")
	dDest.Delete("Parent")

	objNr, err := fm.ctxDest.InsertObject(dDest)
	if err != nil {
		return types.IndirectRef{}, err
	}
	fm.migrated[indRef.ObjectNumber.Value()] = objNr
	indRef.ObjectNumber = types.Integer(objNr)

	for k, v := range dDest {
		dDest[k], err = migrateObject(v, fm.ctxSrc, fm.ctxDest, fm.migrated)
		if err != nil {
			return types.IndirectRef{}, fmt.Errorf("field entry %s: %w", k, err)
		}
	}

	return indRef, nil
}

func (fm *formFieldMigration) selectField(indRef types.IndirectRef) (types.IndirectRef, error) {
	objNr := indRef.ObjectNumber.Value()
	fm.sources[objNr] = indRef
	if destIndRef, ok := fm.fields[objNr]; ok {
		return destIndRef, nil
	}
	if destIndRef := migratedFieldRef(indRef, fm.migrated); destIndRef != nil {
		fm.fields[objNr] = *destIndRef
		return *destIndRef, nil
	}

	dSrc, err := fm.ctxSrc.DereferenceDict(indRef)
	if err != nil {
		return types.IndirectRef{}, err
	}
	if dSrc == nil {
		return types.IndirectRef{}, fmt.Errorf("missing form field dict obj#%d", objNr)
	}
	destIndRef, err := fm.cloneField(indRef, dSrc)
	if err != nil {
		return types.IndirectRef{}, err
	}
	fm.fields[objNr] = destIndRef
	return destIndRef, nil
}

func (fm *formFieldMigration) selectWidget(indRef types.IndirectRef) error {
	seen := map[int]bool{}
	for {
		objNr := indRef.ObjectNumber.Value()
		if seen[objNr] {
			return fmt.Errorf("form field parent cycle at obj#%d", objNr)
		}
		seen[objNr] = true
		if _, err := fm.selectField(indRef); err != nil {
			return fmt.Errorf("select form field obj#%d: %w", objNr, err)
		}

		dSrc, err := fm.ctxSrc.DereferenceDict(indRef)
		if err != nil {
			return fmt.Errorf("dereference form field obj#%d: %w", objNr, err)
		}
		parentIndRef := dSrc.IndirectRefEntry("Parent")
		if parentIndRef == nil {
			fm.roots[objNr] = true
			return nil
		}
		indRef = *parentIndRef
	}
}

func (fm *formFieldMigration) rebuildField(objNr int, destIndRef types.IndirectRef) error {
	srcIndRef := fm.sources[objNr]
	dSrc, err := fm.ctxSrc.DereferenceDict(srcIndRef)
	if err != nil {
		return err
	}
	if dSrc == nil {
		return fmt.Errorf("missing source form field dict")
	}
	dDest, err := fm.ctxDest.DereferenceDict(destIndRef)
	if err != nil {
		return err
	}
	if dDest == nil {
		return fmt.Errorf("missing destination form field dict")
	}
	dDest.Delete("Kids")
	dDest.Delete("Parent")

	if parentIndRef := dSrc.IndirectRefEntry("Parent"); parentIndRef != nil {
		if parentDest, ok := fm.fields[parentIndRef.ObjectNumber.Value()]; ok {
			dDest["Parent"] = parentDest
		}
	}

	o, ok := dSrc.Find("Kids")
	if !ok {
		return nil
	}
	kidsSrc, err := fm.ctxSrc.DereferenceArray(o)
	if err != nil {
		return err
	}
	kidsDest := types.Array{}
	for i, o := range kidsSrc {
		kidIndRef, ok := o.(types.IndirectRef)
		if !ok {
			return fmt.Errorf("form field obj#%d Kids[%d]: wrong type %T", objNr, i, o)
		}
		if kidDest, ok := fm.fields[kidIndRef.ObjectNumber.Value()]; ok {
			kidsDest = append(kidsDest, kidDest)
		}
	}
	if len(kidsDest) > 0 {
		dDest["Kids"] = kidsDest
	}
	return nil
}

func migrateFields(
	fieldsSrc, fieldsDest *types.Array,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
	selection *formFieldSelection,
) error {
	fm := formFieldMigration{
		ctxSrc:   ctxSrc,
		ctxDest:  ctxDest,
		migrated: migrated,
		fields:   map[int]types.IndirectRef{},
		sources:  map[int]types.IndirectRef{},
		roots:    map[int]bool{},
	}
	for _, indRef := range selection.widgets {
		if err := fm.selectWidget(indRef); err != nil {
			return err
		}
	}
	for objNr, destIndRef := range fm.fields {
		if err := fm.rebuildField(objNr, destIndRef); err != nil {
			return fmt.Errorf("rebuild form field obj#%d: %w", objNr, err)
		}
	}

	*fieldsDest = types.Array{}
	for _, o := range *fieldsSrc {
		indRef, ok := o.(types.IndirectRef)
		if !ok || !fm.roots[indRef.ObjectNumber.Value()] {
			continue
		}
		*fieldsDest = append(*fieldsDest, fm.fields[indRef.ObjectNumber.Value()])
	}
	return nil
}

func migrateFormDict(d types.Dict, fields types.Array, ctxSrc, ctxDest *model.Context, migrated map[int]int) error {
	var err error
	for k, v := range d {
		if k == "Fields" {
			d[k] = fields
			continue
		}
		if d[k], err = migrateObject(v, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}
	}
	return nil
}

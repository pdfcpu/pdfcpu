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
	"context"
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
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

func migrateIndRef(
	c context.Context,
	ir *types.IndirectRef,
	ctxSource, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	o, err := ctxSource.Dereference(*ir)
	if err != nil {
		return nil, err
	}

	if o != nil {
		o = o.Clone()
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	objNrNew, err := ctxDest.InsertObject(o)
	if err != nil {
		return nil, err
	}
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	objNr := ir.ObjectNumber.Value()
	migrated[objNr] = objNrNew
	ir.ObjectNumber = types.Integer(objNrNew)
	return o, nil
}

func migrateDict(
	c context.Context,
	d types.Dict,
	ctxSource, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, error) {
	for k, v := range d {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		o, err := migrateObject(c, v, ctxSource, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		d[k] = o
	}
	return d, contextutil.Check(c)
}

func migrateStreamDict(
	c context.Context,
	sd types.StreamDict,
	ctxSource, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, error) {
	o, err := migrateDict(c, sd.Dict, ctxSource, ctxDest, migrated)
	if err != nil {
		return nil, err
	}
	sd.Dict = o.(types.Dict)
	return sd, contextutil.Check(c)
}

func migrateArray(
	c context.Context,
	a types.Array,
	ctxSource, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, error) {
	for i, v := range a {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		o, err := migrateObject(c, v, ctxSource, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		a[i] = o
	}
	return a, contextutil.Check(c)
}

func migrateObject(
	c context.Context,
	o types.Object,
	ctxSource, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	switch o := o.(type) {
	case types.IndirectRef:
		objNr := o.ObjectNumber.Value()
		if migrated[objNr] > 0 {
			o.ObjectNumber = types.Integer(migrated[objNr])
			return o, nil
		}
		o1, err := migrateIndRef(c, &o, ctxSource, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		if _, err := migrateObject(c, o1, ctxSource, ctxDest, migrated); err != nil {
			return nil, err
		}
		return o, nil

	case types.Dict:
		return migrateDict(c, o, ctxSource, ctxDest, migrated)

	case types.StreamDict:
		return migrateStreamDict(c, o, ctxSource, ctxDest, migrated)

	case types.Array:
		return migrateArray(c, o, ctxSource, ctxDest, migrated)
	}

	return o, contextutil.Check(c)
}

func migrateAnnotEntry(
	c context.Context,
	o types.Object,
	index int,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
) (types.Object, types.Dict, *types.IndirectRef, bool, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, nil, nil, false, err
	}
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
	o, err := migrateIndRef(c, &indRef, ctxSrc, ctxDest, migrated)
	if err != nil {
		return nil, nil, nil, false, err
	}
	d, ok := o.(types.Dict)
	if !ok {
		return nil, nil, nil, false, fmt.Errorf("annotation obj#%d: wrong type %T", objNr, o)
	}
	return indRef, d, &sourceIndRef, false, nil
}

func migrateAnnotationDict(
	c context.Context,
	d types.Dict,
	pageIndRef types.IndirectRef,
	isWidget bool,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
) error {
	for k, v := range d {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if k == "P" {
			d["P"] = pageIndRef
			continue
		}
		if k == "Parent" && isWidget {
			d.Delete("Parent")
			continue
		}
		o, err := migrateObject(c, v, ctxSrc, ctxDest, migrated)
		if err != nil {
			return err
		}
		d[k] = o
	}
	return contextutil.Check(c)
}

func migrateAnnots(
	c context.Context,
	o types.Object,
	pageIndRef types.IndirectRef,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
	selection *formFieldSelection,
) (types.Object, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	arr, ok := o.(types.Array)
	if !ok {
		return nil, fmt.Errorf("annotations: wrong type %T", o)
	}
	for i, v := range arr {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		o, d, sourceIndRef, done, err := migrateAnnotEntry(c, v, i, ctxSrc, ctxDest, migrated)
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
		if err := migrateAnnotationDict(
			c, d, pageIndRef, isWidget, ctxSrc, ctxDest, migrated,
		); err != nil {
			return nil, err
		}
	}

	return arr, contextutil.Check(c)
}

func validatePageMigration(
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
	return nil
}

func migratePageDict(
	c context.Context,
	d types.Dict,
	pageIndRef types.IndirectRef,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
	selection *formFieldSelection,
) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := validatePageMigration(ctxSrc, ctxDest, migrated, selection); err != nil {
		return err
	}

	var err error
	for k, v := range d {
		if err := contextutil.Check(c); err != nil {
			return err
		}
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
				v, err = migrateIndRef(c, &o, ctxSrc, ctxDest, migrated)
				if err != nil {
					return fmt.Errorf("page dict entry %s: migrate annotation reference: %w", k, err)
				}
				d[k] = o
				if _, err = migrateAnnots(c, v, pageIndRef, ctxSrc, ctxDest, migrated, selection); err != nil {
					return fmt.Errorf("page dict entry %s: migrate annotations: %w", k, err)
				}
				continue
			}
			if d[k], err = migrateAnnots(c, v, pageIndRef, ctxSrc, ctxDest, migrated, selection); err != nil {
				return fmt.Errorf("page dict entry %s: migrate annotations: %w", k, err)
			}
			continue
		}
		if d[k], err = migrateObject(c, v, ctxSrc, ctxDest, migrated); err != nil {
			return fmt.Errorf("page dict entry %s: migrate object: %w", k, err)
		}
	}
	return contextutil.Check(c)
}

type formFieldMigration struct {
	c               context.Context
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
	if err := contextutil.Check(fm.c); err != nil {
		return types.IndirectRef{}, err
	}
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
		dDest[k], err = migrateObject(fm.c, v, fm.ctxSrc, fm.ctxDest, fm.migrated)
		if err != nil {
			return types.IndirectRef{}, fmt.Errorf("field entry %s: %w", k, err)
		}
	}

	return indRef, nil
}

func (fm *formFieldMigration) selectField(indRef types.IndirectRef) (types.IndirectRef, error) {
	if err := contextutil.Check(fm.c); err != nil {
		return types.IndirectRef{}, err
	}
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
		if err := contextutil.Check(fm.c); err != nil {
			return err
		}
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
	if err := contextutil.Check(fm.c); err != nil {
		return err
	}
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
		if err := contextutil.Check(fm.c); err != nil {
			return err
		}
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
	c context.Context,
	fieldsSrc, fieldsDest *types.Array,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
	selection *formFieldSelection,
) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	fm := formFieldMigration{
		c:        c,
		ctxSrc:   ctxSrc,
		ctxDest:  ctxDest,
		migrated: migrated,
		fields:   map[int]types.IndirectRef{},
		sources:  map[int]types.IndirectRef{},
		roots:    map[int]bool{},
	}
	for _, indRef := range selection.widgets {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := fm.selectWidget(indRef); err != nil {
			return err
		}
	}
	for objNr, destIndRef := range fm.fields {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := fm.rebuildField(objNr, destIndRef); err != nil {
			return fmt.Errorf("rebuild form field obj#%d: %w", objNr, err)
		}
	}

	*fieldsDest = types.Array{}
	for _, o := range *fieldsSrc {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		indRef, ok := o.(types.IndirectRef)
		if !ok || !fm.roots[indRef.ObjectNumber.Value()] {
			continue
		}
		*fieldsDest = append(*fieldsDest, fm.fields[indRef.ObjectNumber.Value()])
	}
	return contextutil.Check(c)
}

func migrateFormDict(
	c context.Context,
	d types.Dict,
	fields types.Array,
	ctxSrc, ctxDest *model.Context,
	migrated map[int]int,
) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	var err error
	for k, v := range d {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if k == "Fields" {
			d[k] = fields
			continue
		}
		if d[k], err = migrateObject(c, v, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}
	}
	return contextutil.Check(c)
}

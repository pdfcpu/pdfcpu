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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

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

func migrateAnnots(o types.Object, pageIndRef types.IndirectRef, ctxSrc, ctxDest *model.Context, migrated map[int]int) (types.Object, error) {
	arr, err := ctxSrc.DereferenceArray(o)
	if err != nil {
		return nil, err
	}

	for i, v := range arr {
		o := v.(types.IndirectRef)
		objNr := o.ObjectNumber.Value()
		if migrated[objNr] > 0 {
			o.ObjectNumber = types.Integer(migrated[objNr])
			arr[i] = o
			continue
		}
		o1, err := migrateIndRef(&o, ctxSrc, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		arr[i] = o
		d := o1.(types.Dict)
		for k, v := range d {
			if k == "P" {
				d["P"] = pageIndRef
				continue
			}
			if k == "Parent" {
				pDict, err := ctxSrc.DereferenceDict(v)
				if err != nil {
					return nil, err
				}
				ft := pDict.NameEntry("FT")
				if ft == nil || *ft != "Btn" {
					d.Delete("Parent")
					continue
				}
				pDict.Delete("Parent")
			}
			if d[k], err = migrateObject(v, ctxSrc, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
	}

	return arr, nil
}

func migratePageDict(d types.Dict, pageIndRef types.IndirectRef, ctxSrc, ctxDest *model.Context, migrated map[int]int) error {
	var err error
	for k, v := range d {
		if k == "Parent" {
			continue
		}
		if k == "Annots" {
			if d[k], err = migrateAnnots(v, pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
				return err
			}
			continue
		}
		if d[k], err = migrateObject(v, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}
	}
	return nil
}

func migrateFields(d types.Dict, fieldsSrc, fieldsDest *types.Array, ctxSrc, ctxDest *model.Context, migrated map[int]int) error {
	o, _ := d.Find("Annots")
	annots, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return err
	}
	for _, v := range annots {
		indRef := v.(types.IndirectRef)
		d, err := ctxDest.DereferenceDict(indRef)
		if err != nil {
			return err
		}
		if pIndRef := d.IndirectRefEntry("Parent"); pIndRef != nil {
			indRef = *pIndRef
		}
		var found bool
		for _, v := range *fieldsDest {
			if v.(types.IndirectRef) == indRef {
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, v := range *fieldsSrc {
			ir := v.(types.IndirectRef)
			objNr := ir.ObjectNumber.Value()
			if migrated[objNr] == indRef.ObjectNumber.Value() {
				*fieldsDest = append(*fieldsDest, indRef)
				break
			}
			d, err := ctxSrc.DereferenceDict(ir)
			if err != nil {
				return err
			}
			o, ok := d.Find("Kids")
			if !ok {
				continue
			}
			kids, err := ctxSrc.DereferenceArray(o)
			if err != nil {
				return err
			}
			if ok, err = detectMigratedAnnot(ctxSrc, &indRef, kids, migrated); err != nil {
				return err
			}
			if ok {
				*fieldsDest = append(*fieldsDest, indRef)
			}
		}
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

func detectMigratedAnnot(ctxSrc *model.Context, indRef *types.IndirectRef, kids types.Array, migrated map[int]int) (bool, error) {
	for _, v := range kids {
		ir := v.(types.IndirectRef)
		objNr := ir.ObjectNumber.Value()
		if migrated[objNr] == indRef.ObjectNumber.Value() {
			indRef.ObjectNumber = types.Integer(migrated[objNr])
			return true, nil
		}

		d, err := ctxSrc.DereferenceDict(ir)
		if err != nil {
			return false, err
		}
		o, ok := d.Find("Kids")
		if !ok {
			continue
		}
		kids, err := ctxSrc.DereferenceArray(o)
		if err != nil {
			return false, err
		}
		if ok, err = detectMigratedAnnot(ctxSrc, indRef, kids, migrated); err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

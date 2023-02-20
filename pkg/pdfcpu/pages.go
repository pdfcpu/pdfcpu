/*
Copyright 2022 The pdfcpu Authors.

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
	"github.com/pkg/errors"
)

// AddPages adds pages and corresponding resources from ctxSrc to ctxDest.
func AddPages(ctxSrc, ctxDest *model.Context, pageNrs []int, usePgCache bool) error {

	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		return err
	}

	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		return err
	}

	pageCache := map[int]*types.IndirectRef{}
	migrated := map[int]int{}
	fields := types.Array{}

	for _, i := range pageNrs {

		if usePgCache {
			if indRef, ok := pageCache[i]; ok {
				if err := model.AppendPageTree(indRef, 1, pagesDict); err != nil {
					return err
				}
				continue
			}
		}

		d, _, inhPAttrs, err := ctxSrc.PageDict(i, false)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("pdfcpu: unknown page number: %d\n", i)
		}

		//log.Write.Printf("AddPages:\n%s\n", inhPAttrs.resources)

		//fmt.Printf("migrresDict bef: \n%s", d)

		d = d.Clone().(types.Dict)
		d["Resources"] = inhPAttrs.Resources.Clone()
		d["Parent"] = *pagesIndRef

		// Handle inherited page attributes.
		d["MediaBox"] = inhPAttrs.MediaBox.Array()
		if inhPAttrs.Rotate%360 > 0 {
			d["Rotate"] = types.Integer(inhPAttrs.Rotate)
		}

		pageIndRef, err := ctxDest.IndRefForNewObject(d)
		if err != nil {
			return err
		}

		// Migrate external page dict into ctxDest.
		if err := migratePageDict(d, *pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}

		if d["Annots"] != nil && ctxSrc.AcroForm != nil {
			// Record form fields
			o, _ := ctxSrc.AcroForm.Find("Fields")
			fieldsSrc, err := ctxSrc.DereferenceArray(o)
			if err != nil {
				return err
			}
			o, _ = d.Find("Annots")
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
				for _, v := range fields {
					ir := v.(types.IndirectRef)
					//if ir.Equals(indRef) {
					if ir == indRef {
						found = true
						break
					}
				}
				if found {
					continue
				}
				for _, v := range fieldsSrc {
					ir := v.(types.IndirectRef)
					objNr := ir.ObjectNumber.Value()
					if migrated[objNr] == indRef.ObjectNumber.Value() {
						fields = append(fields, indRef)
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
						fields = append(fields, indRef)
					}
				}
			}
		}

		if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
			return err
		}

		if usePgCache {
			pageCache[i] = pageIndRef
		}
	}

	if ctxSrc.AcroForm != nil && len(fields) > 0 {
		d := ctxSrc.AcroForm.Clone().(types.Dict)
		if err := migrateFormDict(d, fields, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}
		ctxDest.RootDict["AcroForm"] = d
	}

	return nil
}

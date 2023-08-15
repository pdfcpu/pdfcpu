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

func addPages(
	ctxSrc, ctxDest *model.Context,
	pageNrs []int,
	usePgCache bool,
	pagesIndRef types.IndirectRef,
	pagesDict types.Dict,
	fieldsSrc, fieldsDest types.Array,
	migrated map[int]int) error {

	pageCache := map[int]*types.IndirectRef{}

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

		d = d.Clone().(types.Dict)
		d["Resources"] = inhPAttrs.Resources.Clone()
		d["Parent"] = pagesIndRef
		d["MediaBox"] = inhPAttrs.MediaBox.Array()
		if inhPAttrs.Rotate%360 > 0 {
			d["Rotate"] = types.Integer(inhPAttrs.Rotate)
		}

		pageIndRef, err := ctxDest.IndRefForNewObject(d)
		if err != nil {
			return err
		}

		if err := migratePageDict(d, *pageIndRef, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}

		if d["Annots"] != nil && len(fieldsSrc) > 0 {
			if err := migrateFields(d, &fieldsSrc, &fieldsDest, ctxSrc, ctxDest, migrated); err != nil {
				return err
			}
		}

		if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
			return err
		}

		if usePgCache {
			pageCache[i] = pageIndRef
		}
	}

	return nil
}

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

	fieldsSrc, fieldsDest := types.Array{}, types.Array{}

	if ctxSrc.Form != nil {
		o, _ := ctxSrc.Form.Find("Fields")
		fieldsSrc, err = ctxSrc.DereferenceArray(o)
		if err != nil {
			return err
		}
	}

	migrated := map[int]int{}

	if err := addPages(ctxSrc, ctxDest, pageNrs, usePgCache, *pagesIndRef, pagesDict, fieldsSrc, fieldsDest, migrated); err != nil {
		return err
	}

	if ctxSrc.Form != nil && len(fieldsDest) > 0 {
		d := ctxSrc.Form.Clone().(types.Dict)
		if err := migrateFormDict(d, fieldsDest, ctxSrc, ctxDest, migrated); err != nil {
			return err
		}
		ctxDest.RootDict["AcroForm"] = d
	}

	return nil
}

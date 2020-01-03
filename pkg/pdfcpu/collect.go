/*
Copyright 2020 The pdfcpu Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// CollectPages creates a new PDF Context for a custom PDF page sequence of the PDF represented by ctx.
func CollectPages(ctx *Context, collectedPages []int) (*Context, error) {

	log.Debug.Printf("CollectPages %v\n", collectedPages)

	ctxDest, err := CreateContextWithXRefTable(nil, PaperSize["A4"])
	if err != nil {
		return nil, err
	}

	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		return nil, err
	}

	// This is the page tree root.
	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, err
	}

	pageCache := map[int]*IndirectRef{}
	migrated := map[int]int{}

	for _, i := range collectedPages {

		indRef, ok := pageCache[i]
		if ok {
			if err := AppendPageTree(indRef, 1, pagesDict); err != nil {
				return nil, err
			}
			continue
		}

		// Move page i and resources into new context.

		d, inhPAttrs, err := ctx.PageDict(i)
		if err != nil {
			return nil, err
		}
		if d == nil {
			return nil, errors.Errorf("pdfcpu: unknown page number: %d\n", i)
		}

		// Migrate external page dict into ctxDest.
		for k, v := range d {
			if k == "Parent" {
				d["Parent"] = *pagesIndRef
				continue
			}
			if v, err = migrateObject(ctx, ctxDest, migrated, v); err != nil {
				return nil, err
			}
			d[k] = v
		}

		// Handle inherited page attributes.
		d["MediaBox"] = inhPAttrs.mediaBox.Array()
		if inhPAttrs.rotate%360 > 0 {
			d["Rotate"] = Integer(inhPAttrs.rotate)
		}

		if indRef, err = ctxDest.IndRefForNewObject(d); err != nil {
			return nil, err
		}

		if err := AppendPageTree(indRef, 1, pagesDict); err != nil {
			return nil, err
		}

		pageCache[i] = indRef
	}

	return ctxDest, nil
}

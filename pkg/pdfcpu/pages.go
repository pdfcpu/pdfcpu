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
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pkg/errors"
)

// PageContent returns the content in PDF syntax for page dict d.
func (xRefTable *XRefTable) PageContent(d Dict) ([]byte, error) {

	o, _ := d.Find("Contents")

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	bb := []byte{}

	switch o := o.(type) {

	case StreamDict:
		// no further processing.
		err := o.Decode()
		if err == filter.ErrUnsupportedFilter {
			return nil, errors.New("pdfcpu: unsupported filter: unable to decode content")
		}
		if err != nil {
			return nil, err
		}

		bb = append(bb, o.Content...)

	case Array:
		// process array of content stream dicts.
		for _, o := range o {
			if o == nil {
				continue
			}
			o, _, err := xRefTable.DereferenceStreamDict(o)
			if err != nil {
				return nil, err
			}
			if o == nil {
				continue
			}
			err = o.Decode()
			if err == filter.ErrUnsupportedFilter {
				return nil, errors.New("pdfcpu: unsupported filter: unable to decode content")
			}
			if err != nil {
				return nil, err
			}
			bb = append(bb, o.Content...)
		}

	default:
		return nil, errors.Errorf("pdfcpu: page content must be stream dict or array")
	}

	if len(bb) == 0 {
		return nil, errNoContent
	}

	return bb, nil
}

func migratePageDict(d Dict, ctx, ctxDest *Context, migrated map[int]int) error {
	var err error
	for k, v := range d {
		if k == "Parent" {
			continue
		}
		if d[k], err = migrateObject(v, ctx, ctxDest, migrated); err != nil {
			return err
		}
	}
	return nil
}

// AddPages adds pages and corresponding resources from otherXRefTable to xRefTable.
func AddPages(ctx, ctxDest *Context, pages []int, usePgCache bool) error {

	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		return err
	}

	// This is the page tree root.
	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		return err
	}

	pageCache := map[int]*IndirectRef{}
	migrated := map[int]int{}

	for _, i := range pages {

		if usePgCache {
			if indRef, ok := pageCache[i]; ok {
				if err := AppendPageTree(indRef, 1, pagesDict); err != nil {
					return err
				}
				continue
			}
		}

		// Move page i and required resources into new context.

		consolidateRes := false
		// TODO consolidate via optimize flag
		d, _, inhPAttrs, err := ctx.PageDict(i, consolidateRes)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("pdfcpu: unknown page number: %d\n", i)
		}
		//log.Write.Printf("AddPages:\n%s\n", inhPAttrs.resources)

		//fmt.Printf("migrresDict bef: \n%s", d)

		d = d.Clone().(Dict)

		d["Resources"] = inhPAttrs.Resources
		d["Parent"] = *pagesIndRef

		// Migrate external page dict into ctxDest.
		if err := migratePageDict(d, ctx, ctxDest, migrated); err != nil {
			return err
		}

		// Handle inherited page attributes.
		d["MediaBox"] = inhPAttrs.MediaBox.Array()
		if inhPAttrs.Rotate%360 > 0 {
			d["Rotate"] = Integer(inhPAttrs.Rotate)
		}

		indRef, err := ctxDest.IndRefForNewObject(d)
		if err != nil {
			return err
		}

		if err := AppendPageTree(indRef, 1, pagesDict); err != nil {
			return err
		}

		if usePgCache {
			pageCache[i] = indRef
		}
	}

	return nil
}

/*
Copyright 2026 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type renditionPageIndexKey struct{}

// renditionPageIndex belongs to one root validation pass, including page actions and annotations.
// Page membership comes from the existing page walk; forward references are checked when that walk finishes.
type renditionPageIndex struct {
	pages       types.IntSet
	pending     []int
	complete    bool
	annotations map[int]map[types.IndirectRef]bool
}

func withRenditionPageIndex(c context.Context) (context.Context, *renditionPageIndex) {
	index := &renditionPageIndex{
		pages:       types.IntSet{},
		annotations: map[int]map[types.IndirectRef]bool{},
	}
	return context.WithValue(c, renditionPageIndexKey{}, index), index
}

func renditionPageIndexFromContext(c context.Context) *renditionPageIndex {
	if c == nil {
		return nil
	}
	index, _ := c.Value(renditionPageIndexKey{}).(*renditionPageIndex)
	return index
}

func recordRenditionPage(c context.Context, objNr int) {
	if index := renditionPageIndexFromContext(c); index != nil {
		index.pages[objNr] = true
	}
}

func renditionPageNotInTree(objNr int) error {
	return model.WithValidationErrorObject(errors.New("Rendition.AN.P: page is not in the page tree"), objNr)
}

func (index *renditionPageIndex) finish(c context.Context) error {
	for _, objNr := range index.pending {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if !index.pages[objNr] {
			return renditionPageNotInTree(objNr)
		}
	}
	index.pending = nil
	index.complete = true
	return nil
}

func validateRenditionPageMembership(c context.Context, xRefTable *model.XRefTable, objNr int) error {
	// see 12.6.4.14
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if index := renditionPageIndexFromContext(c); index != nil {
		if index.pages[objNr] {
			return nil
		}
		if !index.complete {
			index.pending = append(index.pending, objNr)
			return nil
		}
		return renditionPageNotInTree(objNr)
	}
	// Individual action validation has no enclosing page walk.
	pageNumber, err := xRefTable.PageNumber(c, objNr)
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("Rendition.AN.P: locate page: %w", err), objNr)
	}
	if pageNumber == 0 {
		return renditionPageNotInTree(objNr)
	}
	return nil
}

func renditionPageAnnotations(c context.Context, xRefTable *model.XRefTable, pageDict types.Dict, objNr int) (map[types.IndirectRef]bool, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	index := renditionPageIndexFromContext(c)
	if index != nil && index.annotations[objNr] != nil {
		return index.annotations[objNr], nil
	}
	annots, err := validateArrayEntry(xRefTable, pageDict, objNr, "pageDict", "Annots", REQUIRED, model.V10, nil)
	if err != nil {
		return nil, err
	}
	refs := map[types.IndirectRef]bool{}
	for _, o := range annots {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		if ir, ok := o.(types.IndirectRef); ok {
			refs[ir] = true
		}
	}
	if index != nil {
		index.annotations[objNr] = refs
	}
	return refs, nil
}

func clearRenditionPageAnnotations(c context.Context, objNr int) {
	if index := renditionPageIndexFromContext(c); index != nil {
		delete(index.annotations, objNr)
	}
}

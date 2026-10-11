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

type formActionPagesKey struct{}

// formActionPages is built lazily and belongs to one AcroForm validation pass.
type formActionPages struct {
	pages   map[types.IndirectRef]int
	widgets map[int]int
}

func withFormActionPages(c context.Context) context.Context {
	return context.WithValue(c, formActionPagesKey{}, &formActionPages{})
}

func (p *formActionPages) collectPage(c context.Context, x *model.XRefTable, ref types.IndirectRef, d types.Dict) error {
	page := len(p.pages) + 1
	p.pages[ref] = page
	annots, err := x.DereferenceArray(d["Annots"])
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("form actions: page Annots: %w", err), ref.ObjectNumber.Value())
	}
	for _, value := range annots {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if ir, ok := value.(types.IndirectRef); ok {
			if p.widgets[ir.ObjectNumber.Value()] == 0 {
				p.widgets[ir.ObjectNumber.Value()] = page
			}
		}
	}
	return nil
}

func (p *formActionPages) walk(c context.Context, x *model.XRefTable, ref types.IndirectRef, depth int, visit *model.PageTreeVisit) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := visit.Enter(ref.ObjectNumber.Value()); err != nil {
		return err
	}
	defer visit.Leave(ref.ObjectNumber.Value())
	d, err := x.DereferenceDict(ref)
	if err != nil {
		return model.WithValidationErrorObject(fmt.Errorf("form actions: page dictionary: %w", err), ref.ObjectNumber.Value())
	}
	if d == nil {
		return model.WithValidationErrorObject(errors.New("form actions: missing page dictionary"), ref.ObjectNumber.Value())
	}
	// The page validator treats Catalog.Pages as the root without requiring its Type entry.
	name := "Pages"
	if depth > 0 {
		name, err = dictTypeForPageNodeDict(x, d, ref.ObjectNumber.Value())
		if err != nil {
			return err
		}
	}
	if name == "Page" {
		return p.collectPage(c, x, ref, d)
	}
	if name != "Pages" {
		return errors.New("form actions: invalid page tree node type")
	}
	// Match the page-tree validator: only Pages nodes contribute to the depth limit.
	if err := x.CheckRecursionDepth("form action page tree", depth); err != nil {
		return err
	}
	kids, err := x.DereferenceArray(d["Kids"])
	if err != nil {
		return err
	}
	for _, value := range kids {
		ir, ok := value.(types.IndirectRef)
		if !ok {
			return errors.New("form actions: page tree kid is not an indirect reference")
		}
		if err := p.walk(c, x, ir, depth+1, visit); err != nil {
			return err
		}
	}
	return nil
}

func (p *formActionPages) build(c context.Context, x *model.XRefTable) error {
	if x.Root == nil && x.RootDict == nil {
		return nil
	}
	root, err := x.Pages()
	if err != nil || root == nil {
		return err
	}
	next := &formActionPages{pages: map[types.IndirectRef]int{}, widgets: map[int]int{}}
	if err := next.walk(c, x, *root, 0, model.NewPageTreeVisit()); err != nil {
		return err
	}
	*p = *next
	return nil
}

func formActionPage(c context.Context, x *model.XRefTable, objNr int, d types.Dict) (int, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	p, _ := c.Value(formActionPagesKey{}).(*formActionPages)
	if p == nil {
		p = &formActionPages{}
	}
	if p.pages == nil {
		if err := p.build(c, x); err != nil {
			return 0, err
		}
	}
	if page := p.widgets[objNr]; page != 0 {
		return page, nil
	}
	if ref, ok := d["P"].(types.IndirectRef); ok {
		return p.pages[ref], nil
	}
	return 0, nil
}

func validateFormActionsOnPage(c context.Context, x *model.XRefTable, objNr int, d types.Dict, terminal bool, validate func() error) error {
	if terminal && d["A"] == nil && d["AA"] == nil {
		return validate()
	}
	page := 0
	if terminal {
		var err error
		page, err = formActionPage(c, x, objNr, d)
		if err != nil {
			return err
		}
	}
	previous := x.CurPage
	x.CurPage = page
	defer func() { x.CurPage = previous }()
	return validate()
}

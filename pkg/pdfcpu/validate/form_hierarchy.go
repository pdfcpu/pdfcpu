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
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateFormHierarchyAndFields(c context.Context, x *model.XRefTable, fields types.Array, owner int, requiresDA bool) error {
	visit := &treeVisit{ancestors: map[int]bool{}, seen: map[int]bool{},
		cycle: model.ErrFormFieldCycle, duplicate: model.ErrFormFieldDuplicate}
	if err := validateFormHierarchy(c, x, fields, owner, "Fields", 0, visit); err != nil {
		return err
	}
	return validateFormFields(c, x, fields, owner, requiresDA)
}

func validateFormHierarchy(c context.Context, x *model.XRefTable, fields types.Array, owner int, entry string, depth int, visit *treeVisit) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	for i, value := range fields {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		ir, ok := value.(types.IndirectRef)
		if !ok {
			// Semantic validation retains responsibility for malformed entry types.
			continue
		}
		if err := validateFormHierarchyNode(c, x, ir, depth, visit); err != nil {
			label := fmt.Sprintf("form field obj#%d %s[%d] obj#%d", owner, entry, i, ir.ObjectNumber.Value())
			return model.WithValidationErrorObject(model.WrapRecursionError(label, err), ir.ObjectNumber.Value())
		}
	}
	return nil
}

func validateFormHierarchyNode(c context.Context, x *model.XRefTable, ir types.IndirectRef, depth int, visit *treeVisit) error {
	if err := checkValidationTree(c, x, "form field tree", depth); err != nil {
		return err
	}
	n, err := visit.enter(ir)
	if err != nil {
		return err
	}
	defer visit.leave(n)
	d, err := x.DereferenceDict(ir)
	if err != nil || d == nil {
		// Keep existing strict errors and relaxed recovery in semantic validation.
		return nil
	}
	raw, found := d.Find("Kids")
	if !found {
		return nil
	}
	kids, err := x.DereferenceArray(raw)
	if err != nil {
		return nil
	}
	// Do not follow Parent, page Annots or CO: they refer to nodes rather than owning children.
	return validateFormHierarchy(c, x, kids, n, "Kids", depth+1, visit)
}

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

type structureTraversal struct {
	*model.StructureTreeVisit
	ctx context.Context
}

func newStructureTraversal(c context.Context) *structureTraversal {
	return &structureTraversal{StructureTreeVisit: model.NewStructureTreeVisit(), ctx: c}
}

func validateStructureHierarchy(c context.Context, x *model.XRefTable, root types.Dict) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := validateStructTreeRootType(x, root); err != nil {
		return err
	}
	// Index values are references, not additional ownership paths.
	return structureHierarchyObject(c, x, root["K"], 1, newStructureTraversal(c), map[int]bool{}, true, "structure tree root K")
}

func structureHierarchyObject(c context.Context, x *model.XRefTable, raw types.Object, depth int, visit *structureTraversal, seen map[int]bool, allowArray bool, label string) (err error) {
	owner := validationObjectNumber(0, raw)
	defer func() {
		err = model.WrapRecursionError(objectContext(label, raw), err)
		err = model.WithValidationErrorObject(err, owner)
	}()
	if err = contextutil.Check(c); err != nil {
		return err
	}
	n, err := enterStructureTreeObject(visit, raw)
	if err != nil {
		return err
	}
	defer visit.Leave(n)
	o, err := x.Dereference(raw)
	if err != nil {
		return err
	}
	switch v := o.(type) {
	case types.Dict:
		return structureHierarchyDict(c, x, v, n, depth, visit, seen)
	case types.Array:
		// Nested arrays are invalid K children; leave their handling to semantic validation.
		if !allowArray {
			return nil
		}
		for i, child := range v {
			if err := structureHierarchyObject(c, x, child, depth, visit, seen, false, fmt.Sprintf("%s[%d]", label, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func structureHierarchyDict(c context.Context, x *model.XRefTable, d types.Dict, n, depth int, visit *structureTraversal, seen map[int]bool) error {
	typ, _, err := x.DereferenceNameEntry(d, "Type")
	if err != nil {
		return err
	}
	if typ != nil && typ.Value() != "StructElem" {
		return nil
	}
	if err := checkValidationTree(c, x, "structure tree", depth); err != nil {
		return err
	}
	if n != 0 {
		if seen[n] {
			err := fmt.Errorf("obj#%d: %w", n, model.ErrStructureTreeDuplicate)
			if x.ValidationMode == model.ValidationStrict {
				return err
			}
			model.ShowDigestedSpecViolationError(err)
			return nil
		}
		seen[n] = true
	}
	raw, found := d.Find("K")
	if !found {
		return nil
	}
	return structureHierarchyObject(c, x, raw, depth+1, visit, seen, true, "structure element K")
}

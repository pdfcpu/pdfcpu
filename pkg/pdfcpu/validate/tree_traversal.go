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
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type treeVisit struct {
	ancestors map[int]bool
	seen      map[int]bool
	cycle     error
	duplicate error
}

func newTreeVisit(nameTree bool, root ...types.Object) *treeVisit {
	v := &treeVisit{ancestors: map[int]bool{}, seen: map[int]bool{},
		cycle: model.ErrNumberTreeCycle, duplicate: model.ErrNumberTreeDuplicate}
	if nameTree {
		v.cycle, v.duplicate = model.ErrNameTreeCycle, model.ErrNameTreeDuplicate
	}
	if len(root) > 0 {
		if ir, ok := root[0].(types.IndirectRef); ok {
			n := ir.ObjectNumber.Value()
			v.ancestors[n], v.seen[n] = true, true
		}
	}
	return v
}

func treeTraversal(visits []*treeVisit, nameTree bool) *treeVisit {
	if len(visits) > 0 {
		return visits[0]
	}
	return newTreeVisit(nameTree)
}

func (v *treeVisit) enter(o types.Object) (int, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0, nil
	}
	n := ir.ObjectNumber.Value()
	if v.ancestors[n] {
		return 0, model.WithValidationErrorObject(fmt.Errorf("obj#%d: %w", n, v.cycle), n)
	}
	if v.seen[n] {
		return 0, model.WithValidationErrorObject(fmt.Errorf("obj#%d: %w", n, v.duplicate), n)
	}
	v.ancestors[n], v.seen[n] = true, true
	return n, nil
}

func (v *treeVisit) leave(n int) {
	delete(v.ancestors, n)
}

func fatalTreeError(err error) bool {
	return errors.Is(err, model.ErrMaxRecursionDepthExceeded) ||
		errors.Is(err, model.ErrNameTreeCycle) || errors.Is(err, model.ErrNameTreeDuplicate) ||
		errors.Is(err, model.ErrNumberTreeCycle) || errors.Is(err, model.ErrNumberTreeDuplicate)
}

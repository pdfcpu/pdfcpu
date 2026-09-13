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

package model

import (
	"errors"
	"fmt"
)

// ErrMaxRecursionDepthExceeded signals excessive parser or object graph nesting.
var ErrMaxRecursionDepthExceeded = errors.New("max recursion depth exceeded")

type recursionDepthError struct {
	name  string
	depth int
	limit int
}

func (e *recursionDepthError) Error() string {
	return fmt.Sprintf("%s: max recursion depth exceeded (depth %d, limit %d)", e.name, e.depth, e.limit)
}

func (e *recursionDepthError) Unwrap() error {
	return ErrMaxRecursionDepthExceeded
}

// ErrPageTreeCycle signals a page tree node cycle.
var ErrPageTreeCycle = errors.New("circular page tree")

// ErrPageTreeDuplicate signals a page tree node reachable from multiple parents.
var ErrPageTreeDuplicate = errors.New("duplicate page tree node")

// ErrFormFieldCycle signals a form field tree cycle.
var ErrFormFieldCycle = errors.New("circular form field tree")

// ErrStructureTreeCycle signals a structure tree cycle.
var ErrStructureTreeCycle = errors.New("circular structure tree")

// ErrActionCycle signals a circular action chain.
var ErrActionCycle = errors.New("circular action chain")

// ErrBeadCycle signals a circular bead chain that does not terminate correctly.
var ErrBeadCycle = errors.New("circular bead chain")

// ErrNameTreeCycle signals an indirect child cycle in a name tree.
var ErrNameTreeCycle = errors.New("circular name tree")

// ErrNameTreeDuplicate signals a repeated indirect name-tree node.
var ErrNameTreeDuplicate = errors.New("duplicate name tree node")

// ErrNumberTreeCycle signals an indirect child cycle in a number tree.
var ErrNumberTreeCycle = errors.New("circular number tree")

// ErrNumberTreeDuplicate signals a repeated indirect number-tree node.
var ErrNumberTreeDuplicate = errors.New("duplicate number tree node")

// MaxRecursionDepth returns the configured recursion depth limit.
func (xRefTable *XRefTable) MaxRecursionDepth() int {
	if xRefTable == nil || xRefTable.Conf == nil || xRefTable.Conf.Limits.MaxRecursionDepth <= 0 {
		return DefaultResourceLimits().MaxRecursionDepth
	}
	return xRefTable.Conf.Limits.MaxRecursionDepth
}

// CheckRecursionDepth rejects recursion levels beyond the configured limit.
func (xRefTable *XRefTable) CheckRecursionDepth(name string, depth int) error {
	return CheckRecursionDepth(name, depth, xRefTable.MaxRecursionDepth())
}

// CheckRecursionDepth rejects recursion levels beyond maxDepth.
func CheckRecursionDepth(name string, depth, maxDepth int) error {
	if maxDepth <= 0 {
		maxDepth = DefaultResourceLimits().MaxRecursionDepth
	}
	if depth > maxDepth {
		return &recursionDepthError{name: name, depth: depth, limit: maxDepth}
	}
	return nil
}

// WrapRecursionError adds context to err without expanding a recursion-limit error during stack unwinding.
func WrapRecursionError(context string, err error) error {
	if err == nil || errors.Is(err, ErrMaxRecursionDepthExceeded) {
		return err
	}
	return fmt.Errorf("%s: %w", context, err)
}

// PageTreeVisit tracks page tree traversal state.
type PageTreeVisit struct {
	ancestors map[int]bool
	seen      map[int]bool
}

// NewPageTreeVisit returns a page tree traversal state.
func NewPageTreeVisit() *PageTreeVisit {
	return &PageTreeVisit{
		ancestors: map[int]bool{},
		seen:      map[int]bool{},
	}
}

// Enter rejects page tree cycles and duplicate page tree nodes.
func (v *PageTreeVisit) Enter(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrPageTreeCycle)
	}
	if v.seen[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrPageTreeDuplicate)
	}
	v.ancestors[objNr] = true
	v.seen[objNr] = true
	return nil
}

// Leave leaves the current page tree node.
func (v *PageTreeVisit) Leave(objNr int) {
	if v == nil || objNr == 0 {
		return
	}
	delete(v.ancestors, objNr)
}

// FormFieldVisit tracks form field ancestor traversal state.
type FormFieldVisit struct {
	ancestors map[int]bool
}

// NewFormFieldVisit returns a form field traversal state.
func NewFormFieldVisit() *FormFieldVisit {
	return &FormFieldVisit{
		ancestors: map[int]bool{},
	}
}

// Enter rejects form field ancestor cycles.
func (v *FormFieldVisit) Enter(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrFormFieldCycle)
	}
	v.ancestors[objNr] = true
	return nil
}

// Check rejects form field ancestor cycles without entering objNr.
func (v *FormFieldVisit) Check(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrFormFieldCycle)
	}
	return nil
}

// Leave leaves the current form field node.
func (v *FormFieldVisit) Leave(objNr int) {
	if v == nil || objNr == 0 {
		return
	}
	delete(v.ancestors, objNr)
}

// StructureTreeVisit tracks structure tree ancestor traversal state.
type StructureTreeVisit struct {
	ancestors map[int]bool
}

// NewStructureTreeVisit returns a structure tree traversal state.
func NewStructureTreeVisit() *StructureTreeVisit {
	return &StructureTreeVisit{
		ancestors: map[int]bool{},
	}
}

// Enter rejects structure tree ancestor cycles.
func (v *StructureTreeVisit) Enter(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return ErrStructureTreeCycle
	}
	v.ancestors[objNr] = true
	return nil
}

// Leave leaves the current structure tree node.
func (v *StructureTreeVisit) Leave(objNr int) {
	if v == nil || objNr == 0 {
		return
	}
	delete(v.ancestors, objNr)
}

// ActionVisit tracks active ancestors and completed subtrees within one action-chain traversal.
type ActionVisit struct {
	ancestors map[int]bool
	validated map[int]int
}

// NewActionVisit returns an action-chain traversal state.
func NewActionVisit() *ActionVisit {
	return &ActionVisit{
		ancestors: map[int]bool{},
	}
}

// Enter rejects action-chain cycles.
func (v *ActionVisit) Enter(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrActionCycle)
	}
	v.ancestors[objNr] = true
	return nil
}

// Leave leaves the current action-chain node.
func (v *ActionVisit) Leave(objNr int) {
	if v == nil || objNr == 0 {
		return
	}
	delete(v.ancestors, objNr)
}

// AlreadyValidated reports whether a completed action subtree covers the requested starting depth.
func (v *ActionVisit) AlreadyValidated(objNr, depth int) bool {
	if v == nil || objNr == 0 {
		return false
	}
	validatedDepth, ok := v.validated[objNr]
	return ok && depth <= validatedDepth
}

// MarkValidated records successful action subtree validation at the requested starting depth.
func (v *ActionVisit) MarkValidated(objNr, depth int) {
	if v == nil || objNr == 0 {
		return
	}
	if v.validated == nil {
		v.validated = map[int]int{}
	}
	if previous, ok := v.validated[objNr]; !ok || depth > previous {
		v.validated[objNr] = depth
	}
}

// BeadVisit tracks bead-chain ancestor traversal state.
type BeadVisit struct {
	ancestors map[int]bool
}

// NewBeadVisit returns a bead-chain traversal state.
func NewBeadVisit() *BeadVisit {
	return &BeadVisit{
		ancestors: map[int]bool{},
	}
}

// Enter rejects bead-chain cycles.
func (v *BeadVisit) Enter(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrBeadCycle)
	}
	v.ancestors[objNr] = true
	return nil
}

// Leave leaves the current bead-chain node.
func (v *BeadVisit) Leave(objNr int) {
	if v == nil || objNr == 0 {
		return
	}
	delete(v.ancestors, objNr)
}

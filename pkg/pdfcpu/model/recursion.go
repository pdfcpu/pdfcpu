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
		return fmt.Errorf("%s depth %d exceeds limit %d: %w", name, depth, maxDepth, ErrMaxRecursionDepthExceeded)
	}
	return nil
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

// ActionVisit tracks action-chain ancestor traversal state.
type ActionVisit struct {
	ancestors map[int]bool
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

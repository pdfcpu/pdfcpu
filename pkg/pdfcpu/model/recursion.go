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

// ErrActionCycle signals a circular action chain.
var ErrActionCycle = errors.New("circular action chain")

// ErrAnnotationIRTCycle signals a circular annotation reply chain.
var ErrAnnotationIRTCycle = errors.New("circular annotation IRT chain")

// ErrBeadCycle signals a circular bead chain that does not terminate correctly.
var ErrBeadCycle = errors.New("circular bead chain")

// ErrCMapCycle signals a circular CMap chain.
var ErrCMapCycle = errors.New("circular CMap chain")

// ErrColorSpaceCycle signals a circular colour-space graph.
var ErrColorSpaceCycle = errors.New("circular colour-space graph")

// ErrFormFieldCycle signals a form field tree cycle.
var ErrFormFieldCycle = errors.New("circular form field tree")

// ErrFormFieldDuplicate signals a repeated field or widget in the Fields/Kids hierarchy.
var ErrFormFieldDuplicate = errors.New("duplicate form field node")

// ErrFunctionCycle signals a circular function graph.
var ErrFunctionCycle = errors.New("circular function graph")

// ErrHalftoneCycle signals a circular halftone graph.
var ErrHalftoneCycle = errors.New("circular halftone graph")

// ErrMaxRecursionDepthExceeded signals excessive parser or object graph nesting.
var ErrMaxRecursionDepthExceeded = errors.New("max recursion depth exceeded")

// ErrMediaClipCycle signals a circular MediaClip chain.
var ErrMediaClipCycle = errors.New("circular MediaClip chain")

// ErrNameTreeCycle signals an indirect child cycle in a name tree.
var ErrNameTreeCycle = errors.New("circular name tree")

// ErrNameTreeDuplicate signals a repeated indirect name-tree node.
var ErrNameTreeDuplicate = errors.New("duplicate name tree node")

// ErrNumberTreeCycle signals an indirect child cycle in a number tree.
var ErrNumberTreeCycle = errors.New("circular number tree")

// ErrNumberTreeDuplicate signals a repeated indirect number-tree node.
var ErrNumberTreeDuplicate = errors.New("duplicate number tree node")

// ErrPageTreeCycle signals a page tree node cycle.
var ErrPageTreeCycle = errors.New("circular page tree")

// ErrPageTreeDuplicate signals a page tree node reachable from multiple parents.
var ErrPageTreeDuplicate = errors.New("duplicate page tree node")

// ErrPatternCycle signals a circular Pattern graph.
var ErrPatternCycle = errors.New("circular Pattern graph")

// ErrRenditionCycle signals a circular selector-Rendition graph.
var ErrRenditionCycle = errors.New("circular rendition graph")

// ErrStructureTreeCycle signals a structure tree cycle.
var ErrStructureTreeCycle = errors.New("circular structure tree")

// ErrStructureTreeDuplicate signals repeated structure-element ownership in the K hierarchy.
var ErrStructureTreeDuplicate = errors.New("duplicate structure tree node")

// ErrTargetCycle signals a circular embedded target chain.
var ErrTargetCycle = errors.New("circular embedded target chain")

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
	validated map[int]int
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

// AlreadyValidated reports whether a completed structure subtree covers the requested starting depth.
func (v *StructureTreeVisit) AlreadyValidated(objNr, depth int) bool {
	if v == nil || objNr == 0 {
		return false
	}
	validatedDepth, ok := v.validated[objNr]
	return ok && depth <= validatedDepth
}

// MarkValidated records successful structure subtree validation at the requested starting depth.
func (v *StructureTreeVisit) MarkValidated(objNr, depth int) {
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

// RenditionVisit tracks active ancestors and completed subtrees within one selector-Rendition traversal.
type RenditionVisit struct {
	ancestors map[int]bool
	validated map[int]int
}

// NewRenditionVisit returns a selector-Rendition traversal state.
func NewRenditionVisit() *RenditionVisit {
	return &RenditionVisit{
		ancestors: map[int]bool{},
	}
}

// Enter rejects selector-Rendition cycles.
func (v *RenditionVisit) Enter(objNr int) error {
	if v == nil || objNr == 0 {
		return nil
	}
	if v.ancestors[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrRenditionCycle)
	}
	v.ancestors[objNr] = true
	return nil
}

// Leave leaves the current selector-Rendition node.
func (v *RenditionVisit) Leave(objNr int) {
	if v == nil || objNr == 0 {
		return
	}
	delete(v.ancestors, objNr)
}

// AlreadyValidated reports whether a completed rendition subtree covers the requested starting depth.
func (v *RenditionVisit) AlreadyValidated(objNr, depth int) bool {
	if v == nil || objNr == 0 {
		return false
	}
	validatedDepth, ok := v.validated[objNr]
	return ok && depth <= validatedDepth
}

// MarkValidated records successful rendition subtree validation at the requested starting depth.
func (v *RenditionVisit) MarkValidated(objNr, depth int) {
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

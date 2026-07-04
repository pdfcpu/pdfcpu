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
var ErrMaxRecursionDepthExceeded = errors.New("pdfcpu: max recursion depth exceeded")

// ErrPageTreeCycle signals a page tree node cycle.
var ErrPageTreeCycle = errors.New("pdfcpu: circular page tree")

// ErrPageTreeDuplicate signals a page tree node reachable from multiple parents.
var ErrPageTreeDuplicate = errors.New("pdfcpu: duplicate page tree node")

// ErrFormFieldCycle signals a form field tree cycle.
var ErrFormFieldCycle = errors.New("pdfcpu: circular form field tree")

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

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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type renditionContext struct {
	context.Context
	cancel   context.CancelFunc
	cancelAt int
	checks   int
}

// Err returns the wrapped context error and advances the deterministic test check counter.
func (c *renditionContext) Err() error {
	c.checks++
	if c.cancel != nil && c.checks == c.cancelAt {
		c.cancel()
	}
	return c.Context.Err()
}

func selectorRendition(r types.Object) types.Dict {
	return types.Dict{
		"Type": types.Name("Rendition"),
		"S":    types.Name("SR"),
		"R":    r,
	}
}

func renditionXRefTable(maxDepth, mode int, objects map[int]types.Object) *model.XRefTable {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = maxDepth
	table := map[int]*model.XRefTableEntry{}
	for objNr, o := range objects {
		table[objNr] = model.NewXRefTableEntryGen0(o)
	}
	v := model.V17
	return &model.XRefTable{
		Table:          table,
		Conf:           conf,
		HeaderVersion:  &v,
		ValidationMode: mode,
	}
}

// TestValidateSelectorRenditionCancellation verifies cancellation stops selector-array traversal before dereference.
func TestValidateSelectorRenditionCancellation(t *testing.T) {
	ir := func(n int) types.IndirectRef { return *types.NewIndirectRef(n, 0) }
	objects := map[int]types.Object{
		1: selectorRendition(types.Array{ir(2), types.Integer(1), ir(3)}),
		2: selectorRendition(types.Array{}),
		3: selectorRendition(types.Array{}),
	}
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &renditionContext{Context: base, cancel: cancel, cancelAt: 4}
	err := validateRenditionDict(c, renditionXRefTable(100, model.ValidationStrict, objects), objects[1].(types.Dict), 1, model.V15)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if c.checks != 4 {
		t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
	}
}

// TestValidateSelectorRenditionRejectsCycleAndDepth verifies both recursive traversal bounds.
func TestValidateSelectorRenditionRejectsCycleAndDepth(t *testing.T) {
	ir := func(n int) types.IndirectRef { return *types.NewIndirectRef(n, 0) }
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		cycle := map[int]types.Object{1: selectorRendition(types.Array{ir(1)})}
		err := validateRenditionDict(t.Context(), renditionXRefTable(100, mode, cycle), cycle[1].(types.Dict), 1, model.V15)
		if !errors.Is(err, model.ErrRenditionCycle) {
			t.Fatalf("mode %d: got %v, want ErrRenditionCycle", mode, err)
		}

		chain := map[int]types.Object{
			1: selectorRendition(types.Array{ir(2)}),
			2: selectorRendition(types.Array{ir(3)}),
			3: selectorRendition(types.Array{}),
		}
		err = validateRenditionDict(t.Context(), renditionXRefTable(1, mode, chain), chain[1].(types.Dict), 1, model.V15)
		if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want ErrMaxRecursionDepthExceeded", mode, err)
		}
	}
}

// TestValidateSelectorRenditionCachesSharedSubtrees verifies shared indirect successors are validated once.
func TestValidateSelectorRenditionCachesSharedSubtrees(t *testing.T) {
	ir := func(n int) types.IndirectRef { return *types.NewIndirectRef(n, 0) }
	objects := map[int]types.Object{13: selectorRendition(types.Array{})}
	for objNr := 12; objNr >= 1; objNr-- {
		objects[objNr] = selectorRendition(types.Array{ir(objNr + 1), ir(objNr + 1)})
	}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		c := &renditionContext{Context: t.Context()}
		if err := validateRenditionDict(c, renditionXRefTable(100, mode, objects), objects[1].(types.Dict), 1, model.V15); err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		if c.checks != 49 {
			t.Fatalf("mode %d: context checks: got %d, want 49", mode, c.checks)
		}
	}
}

// TestValidateSelectorRenditionChecksSharedSubtreeAtDeeperDepth verifies cached shallower paths cannot hide
// depth errors.
func TestValidateSelectorRenditionChecksSharedSubtreeAtDeeperDepth(t *testing.T) {
	ir := func(n int) types.IndirectRef { return *types.NewIndirectRef(n, 0) }
	objects := map[int]types.Object{
		1: selectorRendition(types.Array{ir(2), ir(4)}),
		2: selectorRendition(types.Array{ir(3)}),
		3: selectorRendition(types.Array{}),
		4: selectorRendition(types.Array{ir(5)}),
		5: selectorRendition(types.Array{ir(2)}),
	}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		err := validateRenditionDict(t.Context(), renditionXRefTable(3, mode, objects), objects[1].(types.Dict), 1, model.V15)
		if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want ErrMaxRecursionDepthExceeded", mode, err)
		}
	}
	objects[1] = selectorRendition(types.Array{ir(4), ir(2)})
	if err := validateRenditionDict(t.Context(), renditionXRefTable(4, model.ValidationStrict, objects), objects[1].(types.Dict), 1, model.V15); err != nil {
		t.Fatal(err)
	}
}

// TestValidateSelectorRenditionKeepsDirectDictionariesDistinct verifies direct array members are never conflated.
func TestValidateSelectorRenditionKeepsDirectDictionariesDistinct(t *testing.T) {
	ir := func(n int) types.IndirectRef { return *types.NewIndirectRef(n, 0) }
	objects := map[int]types.Object{
		1: selectorRendition(ir(2)),
		2: types.Array{
			selectorRendition(types.Array{}),
			types.Dict{"Type": types.Name("Rendition"), "R": types.Array{}},
		},
	}
	err := validateRenditionDict(t.Context(), renditionXRefTable(100, model.ValidationStrict, objects), objects[1].(types.Dict), 1, model.V15)
	if err == nil {
		t.Fatal("invalid second direct rendition was skipped")
	}
}

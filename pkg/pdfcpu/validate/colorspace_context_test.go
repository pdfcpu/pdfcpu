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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type colorSpaceCancelContext struct {
	context.Context
	cancel   context.CancelFunc
	cancelAt int
	checks   int
}

// Err returns the wrapped context error and advances the deterministic test check counter.
func (c *colorSpaceCancelContext) Err() error {
	c.checks++
	if c.checks == c.cancelAt {
		c.cancel()
	}
	return c.Context.Err()
}

func colorSpaceXRefTable(maxDepth, mode int, objects map[int]types.Object) *model.XRefTable {
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

func colorSpaceFunction() types.Dict {
	return types.Dict{
		"FunctionType": types.Integer(2),
		"Domain":       types.Array{types.Integer(0), types.Integer(1)},
		"C0":           types.Array{types.Integer(0), types.Integer(0), types.Integer(0)},
		"C1":           types.Array{types.Integer(1), types.Integer(1), types.Integer(1)},
		"N":            types.Integer(1),
	}
}

func nestedColorSpace() types.Array {
	return types.Array{
		types.Name(model.PatternCS),
		types.Array{
			types.Name(model.IndexedCS),
			types.Array{
				types.Name(model.SeparationCS),
				types.Name("Spot"),
				types.Name(model.DeviceRGBCS),
				colorSpaceFunction(),
			},
			types.Integer(1),
			types.StringLiteral("lookup"),
		},
	}
}

// TestValidateColorSpaceCancellation verifies cancellation stops sorted resource traversal before dereference.
func TestValidateColorSpaceCancellation(t *testing.T) {
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := &colorSpaceCancelContext{Context: base, cancel: cancel, cancelAt: 3}
	d := types.Dict{
		"A": types.Name(model.DeviceRGBCS),
		"B": *types.NewIndirectRef(99, 0),
	}
	err := validateColorSpaceResourceDict(c, colorSpaceXRefTable(100, model.ValidationStrict, nil), d, model.V10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if c.checks != 3 {
		t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
	}
}

// TestValidateColorSpaceRejectsCycleAndDepth verifies both recursive traversal bounds in both validation modes.
func TestValidateColorSpaceRejectsCycleAndDepth(t *testing.T) {
	ir := *types.NewIndirectRef(1, 0)
	cycle := types.Array{types.Name(model.IndexedCS), ir, types.Integer(1), types.StringLiteral("lookup")}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		xRefTable := colorSpaceXRefTable(100, mode, map[int]types.Object{1: cycle})
		err := newColorSpaceTraversal(t.Context(), xRefTable).validateColorSpaceDepth(ir, 0, colorSpaceAny, 0)
		if !errors.Is(err, model.ErrColorSpaceCycle) {
			t.Fatalf("mode %d: got %v, want ErrColorSpaceCycle", mode, err)
		}

		xRefTable = colorSpaceXRefTable(2, mode, nil)
		err = newColorSpaceTraversal(t.Context(), xRefTable).validateColorSpaceDepth(nestedColorSpace(), 0, colorSpaceAny, 0)
		if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want ErrMaxRecursionDepthExceeded", mode, err)
		}
	}
}

// TestValidateColorSpaceEnforcesRecursiveRoles verifies semantic restrictions apply before recursive descent.
func TestValidateColorSpaceEnforcesRecursiveRoles(t *testing.T) {
	indexed := types.Array{
		types.Name(model.IndexedCS),
		types.Array{types.Name(model.IndexedCS), types.Name(model.DeviceRGBCS), types.Integer(1), types.StringLiteral("x")},
		types.Integer(1),
		types.StringLiteral("lookup"),
	}
	separation := types.Array{
		types.Name(model.SeparationCS), types.Name("Spot"),
		types.Array{types.Name(model.SeparationCS), types.Name("Alt"), types.Name(model.DeviceRGBCS), colorSpaceFunction()},
		colorSpaceFunction(),
	}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		xRefTable := colorSpaceXRefTable(100, mode, nil)
		err := newColorSpaceTraversal(t.Context(), xRefTable).validateColorSpaceDepth(indexed, 0, colorSpaceAny, 0)
		if err == nil || !strings.Contains(err.Error(), "not allowed as Indexed base color space") {
			t.Fatalf("mode %d: Indexed base: %v", mode, err)
		}
		err = newColorSpaceTraversal(t.Context(), xRefTable).validateColorSpaceDepth(separation, 0, colorSpaceAny, 0)
		if err == nil || !strings.Contains(err.Error(), "not allowed as device or CIE-based color space") {
			t.Fatalf("mode %d: Separation alternate: %v", mode, err)
		}
	}
}

// TestValidateColorSpaceAllowsConformingNestingAndSharing verifies legal nesting and shared references remain valid.
func TestValidateColorSpaceAllowsConformingNestingAndSharing(t *testing.T) {
	ir := *types.NewIndirectRef(1, 0)
	shared := types.Array{types.Name(model.IndexedCS), types.Name(model.DeviceRGBCS), types.Integer(1), types.StringLiteral("x")}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		xRefTable := colorSpaceXRefTable(100, mode, map[int]types.Object{1: shared})
		err := newColorSpaceTraversal(t.Context(), xRefTable).validateColorSpaceDepth(nestedColorSpace(), 0, colorSpaceAny, 0)
		if err != nil {
			t.Fatalf("mode %d: conforming nesting: %v", mode, err)
		}
		if err := validateColorSpaceResourceDict(t.Context(), xRefTable, types.Dict{"A": ir, "B": ir}, model.V10); err != nil {
			t.Fatalf("mode %d: shared reference: %v", mode, err)
		}
	}
}

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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func namedAction(next types.Object) types.Dict {
	d := types.Dict{
		"S": types.Name("Named"),
		"N": types.Name("FirstPage"),
	}
	if next != nil {
		d["Next"] = next
	}
	return d
}

func embeddedTarget(next types.Object) types.Dict {
	d := types.Dict{"R": types.Name("P")}
	if next != nil {
		d["T"] = next
	}
	return d
}

func embeddedGoToAction(target types.Object) types.Dict {
	return types.Dict{
		"S": types.Name("GoToE"),
		"D": types.StringLiteral("destination"),
		"T": target,
	}
}

func requireActionValidationObject(t *testing.T, err error, want int) {
	t.Helper()
	var validationErr *model.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("got %T, want *model.ValidationError: %v", err, err)
	}
	if got := validationErr.ObjectNumber(); got != want {
		t.Fatalf("got validation object %d, want %d", got, want)
	}
}

func actionXRefTable(maxDepth int, dicts map[int]types.Dict) *model.XRefTable {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = maxDepth

	table := map[int]*model.XRefTableEntry{}
	for objNr, d := range dicts {
		table[objNr] = model.NewXRefTableEntryGen0(d)
	}

	v := model.V17
	return &model.XRefTable{
		Table:          table,
		Conf:           conf,
		HeaderVersion:  &v,
		ValidationMode: model.ValidationStrict,
	}
}

// TestValidateActionDictRejectsRecursionDepth verifies action validation respects recursion limits.
func TestValidateActionDictRejectsRecursionDepth(t *testing.T) {
	ir1 := *types.NewIndirectRef(1, 0)
	ir2 := *types.NewIndirectRef(2, 0)
	ir3 := *types.NewIndirectRef(3, 0)
	dicts := map[int]types.Dict{
		1: namedAction(ir2),
		2: namedAction(ir3),
		3: namedAction(nil),
	}

	err := validateActionDictObject(t.Context(), actionXRefTable(1, dicts), dicts[1], ir1, "test action")
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestValidateActionDictRejectsCycles verifies action validation rejects indirect-reference cycles.
func TestValidateActionDictRejectsCycles(t *testing.T) {
	ir1 := *types.NewIndirectRef(1, 0)
	ir2 := *types.NewIndirectRef(2, 0)

	for _, tt := range []struct {
		name  string
		dicts map[int]types.Dict
	}{
		{
			name: "self reference",
			dicts: map[int]types.Dict{
				1: namedAction(ir1),
			},
		},
		{
			name: "two object cycle",
			dicts: map[int]types.Dict{
				1: namedAction(ir2),
				2: namedAction(ir1),
			},
		},
		{
			name: "array cycle",
			dicts: map[int]types.Dict{
				1: namedAction(types.Array{ir2}),
				2: namedAction(ir1),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateActionDictObject(t.Context(), actionXRefTable(100, tt.dicts), tt.dicts[1], ir1, "test action")
			if !errors.Is(err, model.ErrActionCycle) {
				t.Fatalf("got %v, want ErrActionCycle", err)
			}
		})
	}
}

// TestValidateActionDictAllowsSharedSuccessor verifies shared actions do not count as cycles.
func TestValidateActionDictAllowsSharedSuccessor(t *testing.T) {
	ir1 := *types.NewIndirectRef(1, 0)
	ir2 := *types.NewIndirectRef(2, 0)
	ir3 := *types.NewIndirectRef(3, 0)
	ir4 := *types.NewIndirectRef(4, 0)
	dicts := map[int]types.Dict{
		1: namedAction(types.Array{ir2, ir3}),
		2: namedAction(ir4),
		3: namedAction(ir4),
		4: namedAction(nil),
	}

	if err := validateActionDictObject(t.Context(), actionXRefTable(100, dicts), dicts[1], ir1, "test action"); err != nil {
		t.Fatal(err)
	}
}

// TestValidateEmbeddedTargetRejectsRecursionDepth verifies target validation respects recursion limits.
func TestValidateEmbeddedTargetRejectsRecursionDepth(t *testing.T) {
	ir5 := *types.NewIndirectRef(5, 0)
	ir6 := *types.NewIndirectRef(6, 0)
	ir7 := *types.NewIndirectRef(7, 0)
	dicts := map[int]types.Dict{
		5: embeddedTarget(ir6),
		6: embeddedTarget(ir7),
		7: embeddedTarget(nil),
	}

	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		xRefTable := actionXRefTable(1, dicts)
		xRefTable.ValidationMode = mode
		err := validateGoToEActionDict(t.Context(), xRefTable, embeddedGoToAction(ir5), 0, "GoToE")
		if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want ErrMaxRecursionDepthExceeded", mode, err)
		}
		requireActionValidationObject(t, err, 7)
	}
}

// TestValidateEmbeddedTargetRejectsCycles verifies target validation rejects active indirect-reference cycles.
func TestValidateEmbeddedTargetRejectsCycles(t *testing.T) {
	ir5 := *types.NewIndirectRef(5, 0)
	ir6 := *types.NewIndirectRef(6, 0)

	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		for _, tt := range []struct {
			name  string
			dicts map[int]types.Dict
		}{
			{name: "self reference", dicts: map[int]types.Dict{5: embeddedTarget(ir5)}},
			{name: "two object cycle", dicts: map[int]types.Dict{5: embeddedTarget(ir6), 6: embeddedTarget(ir5)}},
		} {
			t.Run(fmt.Sprintf("mode_%d/%s", mode, tt.name), func(t *testing.T) {
				xRefTable := actionXRefTable(100, tt.dicts)
				xRefTable.ValidationMode = mode
				err := validateGoToEActionDict(t.Context(), xRefTable, embeddedGoToAction(ir5), 0, "GoToE")
				if !errors.Is(err, model.ErrTargetCycle) {
					t.Fatalf("got %v, want ErrTargetCycle", err)
				}
				requireActionValidationObject(t, err, 5)
			})
		}
	}
}

// TestValidateEmbeddedTargetAllowsReuse verifies target state does not leak across actions.
func TestValidateEmbeddedTargetAllowsReuse(t *testing.T) {
	ir5 := *types.NewIndirectRef(5, 0)
	xRefTable := actionXRefTable(100, map[int]types.Dict{5: embeddedTarget(nil)})
	action := embeddedGoToAction(ir5)

	for i := 0; i < 2; i++ {
		if err := validateGoToEActionDict(t.Context(), xRefTable, action, 0, "GoToE"); err != nil {
			t.Fatal(err)
		}
	}
}

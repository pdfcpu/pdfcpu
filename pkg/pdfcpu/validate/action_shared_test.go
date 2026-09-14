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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func sharedActionDict(next types.Object) types.Dict {
	d := types.Dict{
		"S": types.Name("Named"),
		"N": types.Name("FirstPage"),
	}
	if next != nil {
		d["Next"] = next
	}
	return d
}

func sharedActionXRefTable(maxDepth int, dicts map[int]types.Dict) *model.XRefTable {
	conf := model.NewStatelessConfiguration()
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

// TestSharedActionRegistersDestinationOnce verifies shared graphs do not multiply merge references.
func TestSharedActionRegistersDestinationOnce(t *testing.T) {
	dicts := map[int]types.Dict{13: {"S": types.Name("GoTo"), "D": types.StringLiteral("target")}}
	for i := 12; i >= 1; i-- {
		ir := *types.NewIndirectRef(i+1, 0)
		dicts[i] = sharedActionDict(types.Array{ir, ir})
	}
	x := sharedActionXRefTable(100, dicts)
	x.Conf.Cmd = model.MERGECREATE
	x.NameRefs = map[string]model.NameMap{}
	if err := validateActionDictObject(t.Context(), x, dicts[1], *types.NewIndirectRef(1, 0), "shared action"); err != nil {
		t.Fatal(err)
	}
	if got := len(x.NameRefs["Dests"]["target"]); got != 1 {
		t.Fatalf("destination registrations: got %d, want 1", got)
	}
}

// TestSharedActionChecksDeeperPath verifies a cached success cannot bypass a descendant depth failure.
func TestSharedActionChecksDeeperPath(t *testing.T) {
	ir := func(n int) types.IndirectRef { return *types.NewIndirectRef(n, 0) }
	dicts := map[int]types.Dict{
		1: sharedActionDict(types.Array{ir(2), ir(4)}), 2: sharedActionDict(ir(3)),
		3: sharedActionDict(nil), 4: sharedActionDict(ir(5)), 5: sharedActionDict(ir(2)),
	}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		x := sharedActionXRefTable(3, dicts)
		x.ValidationMode = mode
		err := validateActionDictObject(t.Context(), x, dicts[1], ir(1), "shared action")
		if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want depth error", mode, err)
		}
	}
	dicts[1]["Next"] = types.Array{ir(4), ir(2)}
	if err := validateActionDictObject(t.Context(), sharedActionXRefTable(4, dicts), dicts[1], ir(1), "shared action"); err != nil {
		t.Fatal(err)
	}
}

func validateSharedAction(t *testing.T, d types.Dict) *model.XRefTable {
	t.Helper()
	ir := *types.NewIndirectRef(2, 0)
	dicts := map[int]types.Dict{1: sharedActionDict(types.Array{ir, ir}), 2: d}
	x := sharedActionXRefTable(100, dicts)
	x.ValidationMode = model.ValidationRelaxed
	x.URIs = map[int]map[string]string{}
	x.CurPage = 7
	x.ValidateLinks = true
	if err := validateActionDictObject(t.Context(), x, dicts[1], *types.NewIndirectRef(1, 0), "shared action"); err != nil {
		t.Fatal(err)
	}
	return x
}

// TestSharedActionPreservesRepair verifies the first visit repairs a shared action.
func TestSharedActionPreservesRepair(t *testing.T) {
	d := types.Dict{"S": types.Name("GoToE"), "F": types.StringLiteral("other.pdf"), "Dest": types.StringLiteral("target")}
	validateSharedAction(t, d)
	if got := d["D"]; got != types.StringLiteral("target") {
		t.Fatalf("repaired destination: %v", got)
	}
	if _, ok := d["Dest"]; ok {
		t.Fatal("obsolete Dest entry retained")
	}
}

// TestSharedActionPreservesURI verifies a shared URI is collected at its first page.
func TestSharedActionPreservesURI(t *testing.T) {
	uri := "https://example.invalid/shared"
	x := validateSharedAction(t, types.Dict{"S": types.Name("URI"), "URI": types.StringLiteral(uri)})
	if _, ok := x.URIs[7][uri]; !ok || len(x.URIs[7]) != 1 {
		t.Fatalf("collected URIs: %v", x.URIs)
	}
}

// TestDirectActionsRemainDistinct verifies dictionaries without indirect identities are not conflated.
func TestDirectActionsRemainDistinct(t *testing.T) {
	d := sharedActionDict(types.Array{
		types.Dict{"S": types.Name("GoTo"), "D": types.StringLiteral("first")},
		types.Dict{"S": types.Name("GoTo"), "D": types.StringLiteral("second")},
	})
	x := sharedActionXRefTable(100, map[int]types.Dict{1: d})
	x.Conf.Cmd = model.MERGECREATE
	x.NameRefs = map[string]model.NameMap{}
	if err := validateActionDictObject(t.Context(), x, d, *types.NewIndirectRef(1, 0), "direct actions"); err != nil {
		t.Fatal(err)
	}
	if len(x.NameRefs["Dests"]["first"]) != 1 || len(x.NameRefs["Dests"]["second"]) != 1 {
		t.Fatalf("destination registrations: %v", x.NameRefs)
	}
}

// TestActionValidationStartsFresh verifies a later root traversal sees changed dictionaries.
func TestActionValidationStartsFresh(t *testing.T) {
	d := sharedActionDict(nil)
	x := sharedActionXRefTable(100, map[int]types.Dict{1: d})
	ir := *types.NewIndirectRef(1, 0)
	if err := validateActionDictObject(t.Context(), x, d, ir, "first traversal"); err != nil {
		t.Fatal(err)
	}
	d["S"] = types.Name("InvalidAction")
	if err := validateActionDictObject(t.Context(), x, d, ir, "second traversal"); err == nil {
		t.Fatal("second traversal reused stale validation")
	}
}

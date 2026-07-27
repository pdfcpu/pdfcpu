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

func beadDict(thread, previous, next types.IndirectRef) types.Dict {
	return types.Dict{
		"T": thread,
		"R": types.Array{
			types.Integer(0),
			types.Integer(0),
			types.Integer(1),
			types.Integer(1),
		},
		"V": previous,
		"N": next,
	}
}

func beadXRefTable(maxDepth int, dicts map[int]types.Dict) *model.XRefTable {
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
		ValidationMode: model.ValidationRelaxed,
	}
}

// TestValidateBeadDictAllowsLongChains verifies bead validation does not treat flat lists as recursive nesting.
func TestValidateBeadDictAllowsLongChains(t *testing.T) {
	const beadCount = 102

	thread := *types.NewIndirectRef(10, 0)
	first := *types.NewIndirectRef(1, 0)
	dicts := map[int]types.Dict{}
	for objNr := 1; objNr <= beadCount; objNr++ {
		previous := objNr - 1
		if objNr == 1 {
			previous = beadCount
		}
		next := objNr + 1
		if objNr == beadCount {
			next = 1
		}
		dicts[objNr] = beadDict(
			thread,
			*types.NewIndirectRef(previous, 0),
			*types.NewIndirectRef(next, 0),
		)
	}

	if err := validateFirstBeadDict(beadXRefTable(1, dicts), &first, &thread); err != nil {
		t.Fatal(err)
	}
}

// TestValidateBeadDictRejectsCycles verifies bead validation rejects malformed descendant cycles.
func TestValidateBeadDictRejectsCycles(t *testing.T) {
	thread := *types.NewIndirectRef(10, 0)
	first := *types.NewIndirectRef(1, 0)
	second := *types.NewIndirectRef(2, 0)
	third := *types.NewIndirectRef(3, 0)
	last := *types.NewIndirectRef(4, 0)

	for _, tt := range []struct {
		name  string
		dicts map[int]types.Dict
	}{
		{
			name: "self reference",
			dicts: map[int]types.Dict{
				1: beadDict(thread, last, second),
				2: beadDict(thread, first, second),
			},
		},
		{
			name: "multi-bead cycle",
			dicts: map[int]types.Dict{
				1: beadDict(thread, last, second),
				2: beadDict(thread, first, third),
				3: beadDict(thread, second, second),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFirstBeadDict(beadXRefTable(100, tt.dicts), &first, &thread)
			if !errors.Is(err, model.ErrBeadCycle) {
				t.Fatalf("got %v, want ErrBeadCycle", err)
			}
		})
	}
}

// TestValidateBeadDictAllowsValidCircularChains verifies valid bead-list closure does not count as a cycle.
func TestValidateBeadDictAllowsValidCircularChains(t *testing.T) {
	thread := *types.NewIndirectRef(10, 0)
	first := *types.NewIndirectRef(1, 0)
	second := *types.NewIndirectRef(2, 0)

	for _, tt := range []struct {
		name  string
		dicts map[int]types.Dict
	}{
		{
			name: "sole bead",
			dicts: map[int]types.Dict{
				1: beadDict(thread, first, first),
			},
		},
		{
			name: "two beads",
			dicts: map[int]types.Dict{
				1: beadDict(thread, second, second),
				2: beadDict(thread, first, first),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFirstBeadDict(beadXRefTable(100, tt.dicts), &first, &thread)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"strings"
	"testing"
)

type structureCancelContext struct {
	context.Context
	cancel context.CancelFunc
	checks int
}

func (c *structureCancelContext) Err() error {
	c.checks++
	if c.checks == 4 {
		c.cancel()
	}
	return c.Context.Err()
}

// TestStructureTraversalCancellation interrupts semantic descent through each structure entry point.
func TestStructureTraversalCancellation(t *testing.T) {
	for _, entry := range []string{"K", "IDTree", "ParentTree", "hierarchy"} {
		t.Run(entry, func(t *testing.T) {
			x, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
			if err != nil {
				t.Fatal(err)
			}
			version := model.V17
			x.HeaderVersion = &version
			root := types.Dict{"Type": types.Name("StructTreeRoot"), "K": *types.NewIndirectRef(5, 0)}
			x.Table[4] = model.NewXRefTableEntryGen0(root)
			for n := 5; n < 20; n++ {
				d := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Div"), "P": *types.NewIndirectRef(n-1, 0), "ID": types.StringLiteral("item")}
				if n < 19 {
					d["K"] = types.Array{*types.NewIndirectRef(n+1, 0)}
				}
				x.Table[n] = model.NewXRefTableEntryGen0(d)
			}
			x.Table[20] = model.NewXRefTableEntryGen0(types.Dict{"Nums": types.Array{types.Integer(0), *types.NewIndirectRef(5, 0)}})
			base, cancel := context.WithCancel(t.Context())
			defer cancel()
			c := &structureCancelContext{Context: base, cancel: cancel}
			switch entry {
			case "K":
				err = validateStructTreeRootKContext(c, x.XRefTable, root["K"], false)
			case "IDTree":
				err = validateNameTreeValueContext(c, "IDTree", x.XRefTable, root["K"], 4)
			case "ParentTree":
				err = validateStructTreeRootDictEntryParentTree(c, x.XRefTable, types.NewIndirectRef(20, 0), false)
			case "hierarchy":
				err = validateStructureHierarchy(c, x.XRefTable, root)
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, want cancellation", err)
			}
			if c.checks != 4 {
				t.Fatalf("continued traversal after cancellation: %d checks", c.checks)
			}
		})
	}
}

// TestStructureHierarchyContentReferences keeps repeated content references outside ownership tracking.
func TestStructureHierarchyContentReferences(t *testing.T) {
	for _, typ := range []string{"MCR", "OBJR"} {
		t.Run(typ, func(t *testing.T) {
			x, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
			if err != nil {
				t.Fatal(err)
			}
			ref := *types.NewIndirectRef(5, 0)
			x.Table[5] = model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name(typ)})
			root := types.Dict{"Type": types.Name("StructTreeRoot"), "K": types.Dict{"S": types.Name("Div"), "K": types.Array{ref, ref}}}
			if err := validateStructureHierarchy(t.Context(), x.XRefTable, root); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestStructureHierarchyNestedArray leaves invalid nested arrays to semantic validation without walking them.
func TestStructureHierarchyNestedArray(t *testing.T) {
	x, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ref := *types.NewIndirectRef(5, 0)
	x.Table[5] = model.NewXRefTableEntryGen0(types.Dict{"S": types.Name("Div")})
	root := types.Dict{"Type": types.Name("StructTreeRoot"), "K": types.Array{types.Array{ref, ref}}}
	if err := validateStructureHierarchy(t.Context(), x.XRefTable, root); err != nil {
		t.Fatal(err)
	}
}

// TestStructureHierarchyDuplicateModes preserves relaxed compatibility for repeated structure children.
func TestStructureHierarchyDuplicateModes(t *testing.T) {
	for _, tt := range []struct {
		name string
		mode int
		want error
	}{
		{"strict", model.ValidationStrict, model.ErrStructureTreeDuplicate},
		{"relaxed", model.ValidationRelaxed, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			x, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
			if err != nil {
				t.Fatal(err)
			}
			x.XRefTable.ValidationMode = tt.mode
			ref := *types.NewIndirectRef(5, 0)
			x.Table[5] = model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name("StructElem"), "S": types.Name("TD")})
			root := types.Dict{"Type": types.Name("StructTreeRoot"), "K": types.Array{ref, ref}}
			err = validateStructureHierarchy(t.Context(), x.XRefTable, root)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

type structureCountingContext struct {
	context.Context
	checks int
}

func (c *structureCountingContext) Err() error {
	c.checks++
	return c.Context.Err()
}

// TestStructureSemanticTraversalCachesSharedSubtrees verifies linear validation of repeated structure subtrees.
func TestStructureSemanticTraversalCachesSharedSubtrees(t *testing.T) {
	x, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	x.XRefTable.ValidationMode = model.ValidationRelaxed
	version := model.V17
	x.HeaderVersion = &version
	for n := 5; n <= 13; n++ {
		d := types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Div")}
		if n < 13 {
			ref := *types.NewIndirectRef(n+1, 0)
			d["K"] = types.Array{ref, ref}
		}
		x.Table[n] = model.NewXRefTableEntryGen0(d)
	}
	c := &structureCountingContext{Context: t.Context()}
	if err := validateStructTreeRootKContext(c, x.XRefTable, *types.NewIndirectRef(5, 0), false); err != nil {
		t.Fatal(err)
	}
	if c.checks > 128 {
		t.Fatalf("shared subtree traversal performed %d context checks", c.checks)
	}
}

// TestStructureSemanticTraversalRechecksDeeperReuse preserves depth enforcement after a shallow cached validation.
func TestStructureSemanticTraversalRechecksDeeperReuse(t *testing.T) {
	x, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	x.XRefTable.ValidationMode = model.ValidationRelaxed
	version := model.V17
	x.HeaderVersion = &version
	x.Configuration.Limits.MaxRecursionDepth = 2
	shared := *types.NewIndirectRef(5, 0)
	x.Table[5] = model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Div")})
	x.Table[6] = model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Div"), "K": *types.NewIndirectRef(7, 0)})
	x.Table[7] = model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name("StructElem"), "S": types.Name("Div"), "K": shared})
	err = validateStructTreeRootKContext(t.Context(), x.XRefTable, types.Array{shared, *types.NewIndirectRef(6, 0)}, false)
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want depth error", err)
	}
}

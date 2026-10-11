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
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func treeRangeTestContext(t *testing.T, version model.Version, mode int) *model.XRefTable {
	t.Helper()
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.HeaderVersion = &version
	ctx.XRefTable.ValidationMode = mode
	return ctx.XRefTable
}

func treeRangeTestRoot(x *model.XRefTable, names bool, layout string) (types.Dict, types.Object) {
	entry := "Nums"
	keys := types.Array{types.Integer(0), types.Integer(1), types.Integer(5), types.Integer(6)}
	value := types.Dict{"S": types.Name("D")}
	if names {
		entry = "Names"
		keys = types.Array{types.StringLiteral(""), types.StringLiteral("a"), types.StringLiteral("b"), types.StringLiteral("c")}
		value = types.Dict{"S": types.Name("JavaScript"), "JS": types.StringLiteral("1;")}
	}
	leaf := types.Dict{entry: types.Array{keys[0], value, keys[1], value}}
	if layout == "root leaf" {
		return leaf, keys[1]
	}
	leaf["Limits"] = types.Array{keys[0], keys[1]}
	x.Table[10] = model.NewXRefTableEntryGen0(leaf)
	root := types.Dict{"Kids": types.Array{*types.NewIndirectRef(10, 0)}}
	if layout == "bad leaf" {
		leaf["Limits"] = types.Array{keys[1], keys[1]}
	}
	if layout == "leaf" || layout == "bad leaf" {
		return root, keys[1]
	}
	x.Table[11] = model.NewXRefTableEntryGen0(types.Dict{
		entry:    types.Array{keys[2], value, keys[3], value},
		"Limits": types.Array{keys[2], keys[3]},
	})
	root["Kids"] = types.Array{*types.NewIndirectRef(10, 0), *types.NewIndirectRef(11, 0)}
	if layout == "kids" {
		return root, keys[3]
	}
	root["Limits"] = types.Array{keys[0], keys[3]}
	if layout == "bad branch" {
		root["Limits"] = types.Array{keys[2], keys[3]}
	}
	x.Table[12] = model.NewXRefTableEntryGen0(root)
	return types.Dict{"Kids": types.Array{*types.NewIndirectRef(12, 0)}}, keys[3]
}

// TestTreeFirstKeyRanges preserves valid zero-valued keys in leaf and intermediate ranges.
func TestTreeFirstKeyRanges(t *testing.T) {
	for _, version := range []model.Version{model.V17, model.V20} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, names := range []bool{false, true} {
				for _, layout := range []string{"root leaf", "leaf", "kids", "nested", "bad leaf", "bad branch"} {
					t.Run(fmt.Sprintf("%s/mode=%d/names=%t/%s", version, mode, names, layout), func(t *testing.T) {
						x := treeRangeTestContext(t, version, mode)
						root, last := treeRangeTestRoot(x, names, layout)
						wantErr := mode == model.ValidationStrict && strings.HasPrefix(layout, "bad")
						if names {
							checkNameTreeFirstKey(t, x, root, string(last.(types.StringLiteral)), wantErr)
						} else {
							checkNumberTreeFirstKey(t, x, root, int(last.(types.Integer)), wantErr)
						}
					})
				}
			}
		}
	}
}

func checkNumberTreeFirstKey(t *testing.T, x *model.XRefTable, root types.Dict, wantLast int, wantErr bool) {
	t.Helper()
	first, last, err := validateNumberTree(t.Context(), x, "PageLabel", root, 0, true, false)
	if wantErr {
		if err == nil || !strings.Contains(err.Error(), "limits") {
			t.Fatalf("got %v, want limits error", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if first != 0 || last != wantLast {
		t.Fatalf("range = [%d %d], want [0 %d]", first, last, wantLast)
	}
}

func checkNameTreeFirstKey(t *testing.T, x *model.XRefTable, root types.Dict, wantLast string, wantErr bool) {
	t.Helper()
	first, last, node, err := validateNameTree(t.Context(), x, "JavaScript", root, 0, true)
	if wantErr {
		if err == nil || !strings.Contains(err.Error(), "limits") {
			t.Fatalf("got %v, want limits error", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if first != "" || last != wantLast || node.Kmin != "" || node.Kmax != wantLast {
		t.Fatalf("range = [%q %q], node range = [%q %q], want [\"\" %q]", first, last, node.Kmin, node.Kmax, wantLast)
	}
	if _, found, err := node.Value(t.Context(), ""); err != nil || !found {
		t.Fatalf("empty key lookup: found=%t, error=%v", found, err)
	}
}

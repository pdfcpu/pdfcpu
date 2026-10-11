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

// TestTreeKeyOrder checks strict ordering and relaxed compatibility using PDF string bytes.
func TestTreeKeyOrder(t *testing.T) {
	for _, version := range []model.Version{model.V17, model.V20} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, tt := range []struct {
				name      string
				names     bool
				keys      types.Array
				strictErr bool
			}{
				{"numbers sorted", false, types.NewIntegerArray(0, 2, 10), false},
				{"numbers descending", false, types.NewIntegerArray(7, 2), true},
				{"numbers later descending", false, types.NewIntegerArray(0, 10, 2), true},
				{"numbers duplicate zero", false, types.NewIntegerArray(0, 0), true},
				{"numbers duplicate nonzero", false, types.NewIntegerArray(2, 2), true},
				{"names sorted", true, types.Array{types.StringLiteral(""), types.StringLiteral("a"), types.StringLiteral("aa")}, false},
				{"names descending", true, types.Array{types.StringLiteral("z"), types.StringLiteral("a")}, true},
				{"names shorter prefix after longer", true, types.Array{types.StringLiteral("aa"), types.StringLiteral("a")}, true},
				{"names empty after nonempty", true, types.Array{types.StringLiteral("a"), types.StringLiteral("")}, true},
				{"names duplicate compatibility", true, types.Array{types.StringLiteral("a"), types.StringLiteral("a")}, false},
				{"names mixed representation", true, types.Array{types.StringLiteral("a"), types.HexLiteral("62")}, false},
				{"names PDFDocEncoding sorted", true, types.Array{types.HexLiteral("80"), types.HexLiteral("81")}, false},
				{"names PDFDocEncoding descending", true, types.Array{types.HexLiteral("81"), types.HexLiteral("80")}, true},
				{"names octal sorted", true, types.Array{types.StringLiteral(`\200`), types.StringLiteral(`\201`)}, false},
				{"names UTF16 sorted", true, types.Array{types.HexLiteral("FEFFD800DC00"), types.HexLiteral("FEFFE000")}, false},
				{"names UTF16 descending", true, types.Array{types.HexLiteral("FEFFE000"), types.HexLiteral("FEFFD800DC00")}, true},
			} {
				t.Run(fmt.Sprintf("%s/mode=%d/%s", version, mode, tt.name), func(t *testing.T) {
					x := treeRangeTestContext(t, version, mode)
					err := validateTreeOrderTestKeys(t, x, tt.names, tt.keys)
					wantErr := mode == model.ValidationStrict && tt.strictErr
					if !wantErr {
						if err != nil {
							t.Fatal(err)
						}
						return
					}
					if err == nil || !strings.Contains(err.Error(), "order after") {
						t.Fatalf("got %v, want key order error", err)
					}
				})
			}
		}
	}
}

func validateTreeOrderTestKeys(t *testing.T, x *model.XRefTable, names bool, keys types.Array) error {
	t.Helper()
	entry := "Nums"
	value := types.Dict{"S": types.Name("D")}
	if names {
		entry = "Names"
		value = types.Dict{"S": types.Name("JavaScript"), "JS": types.StringLiteral("1;")}
	}
	a := types.Array{}
	for _, key := range keys {
		a = append(a, key, value)
	}
	root := types.Dict{entry: a}
	if names {
		_, _, _, err := validateNameTree(t.Context(), x, "JavaScript", root, 0, true)
		return err
	}
	_, _, err := validateNumberTree(t.Context(), x, "PageLabel", root, 0, true, false)
	return err
}

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

func encodedValidationNameTree(t *testing.T, x *model.XRefTable, layout string) types.Dict {
	t.Helper()
	keys := types.Array{types.HexLiteral("FEFFD800DC00"), types.HexLiteral("FEFFE000")}
	value := types.Dict{"S": types.Name("JavaScript"), "JS": types.StringLiteral("1;")}
	names := types.Array{keys[0], value, keys[1], value}
	if layout == "indirect" {
		x.Table[12] = model.NewXRefTableEntryGen0(names)
		x.Table[13] = model.NewXRefTableEntryGen0(keys[0])
		names[0] = *types.NewIndirectRef(13, 0)
		return types.Dict{"Names": *types.NewIndirectRef(12, 0)}
	}
	if layout == "leaf" {
		return types.Dict{"Names": names}
	}
	kids := types.Array{}
	for i, key := range keys {
		leaf := types.Dict{"Names": types.Array{key, value}, "Limits": types.Array{key, key}}
		x.Table[10+i] = model.NewXRefTableEntryGen0(leaf)
		kids = append(kids, *types.NewIndirectRef(10+i, 0))
	}
	return types.Dict{"Kids": kids}
}

func checkValidatedEncodedKeys(t *testing.T, x *model.XRefTable, root types.Dict) {
	t.Helper()
	first, last, node, err := validateNameTree(t.Context(), x, "JavaScript", root, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	textKeys := []string{"\U00010000", "\uE000"}
	if first != types.EncodeUTF16String(textKeys[0]) || last != types.EncodeUTF16String(textKeys[1]) {
		t.Fatalf("validation byte range=[%x %x]", first, last)
	}
	if node.Kmin != textKeys[0] || node.Kmax != textKeys[1] {
		t.Fatalf("public text limits=[%q %q]", node.Kmin, node.Kmax)
	}
	checkDecodedNameTreeLookups(t, node, textKeys)
	x.Names["JavaScript"] = node
	x.RootDict = types.Dict{"Names": types.Dict{"JavaScript": root}}
	size := 20
	x.Size = &size
	x.Table[0] = model.NewFreeHeadXRefTableEntry()
	if err := x.BindNameTrees(); err != nil {
		t.Fatal(err)
	}
	first2, last2, reread, err := validateNameTree(t.Context(), x, "JavaScript", node.D, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if first2 != first || last2 != last {
		t.Fatalf("key bytes changed: [%x %x]", first2, last2)
	}
	checkDecodedNameTreeLookups(t, reread, textKeys)
}

// TestNameTreeEncodedKeysRoundTrip checks original byte ordering and decoded lookup in both input versions.
func TestNameTreeEncodedKeysRoundTrip(t *testing.T) {
	for _, version := range []model.Version{model.V17, model.V20} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			for _, layout := range []string{"leaf", "kids", "indirect"} {
				t.Run(fmt.Sprintf("%s/mode=%d/%s", version, mode, layout), func(t *testing.T) {
					x := treeRangeTestContext(t, version, mode)
					checkValidatedEncodedKeys(t, x, encodedValidationNameTree(t, x, layout))
				})
			}
		}
	}
}

// TestNameTreeLimitsCompareBytes distinguishes keys with equal display text but different PDF bytes.
func TestNameTreeLimitsCompareBytes(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		x := treeRangeTestContext(t, model.V17, mode)
		value := types.Dict{"S": types.Name("JavaScript"), "JS": types.StringLiteral("1;")}
		leaf := types.Dict{
			"Names":  types.Array{types.HexLiteral("FEFF0061"), value},
			"Limits": types.Array{types.StringLiteral("a"), types.StringLiteral("a")},
		}
		x.Table[10] = model.NewXRefTableEntryGen0(leaf)
		root := types.Dict{"Kids": types.Array{*types.NewIndirectRef(10, 0)}}
		_, _, node, err := validateNameTree(t.Context(), x, "JavaScript", root, 0, true)
		if mode == model.ValidationStrict {
			if err == nil || !strings.Contains(err.Error(), "limits") {
				t.Fatalf("got %v, want limits error", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			if _, found, err := node.Value(t.Context(), "a"); err != nil || !found {
				t.Fatalf("found=%t error=%v", found, err)
			}
		}
	}
}

func checkDecodedNameTreeLookups(t *testing.T, n *model.Node, keys []string) {
	t.Helper()
	for _, key := range keys {
		if _, found, err := n.Value(t.Context(), key); err != nil || !found {
			t.Fatalf("lookup %q: found=%t error=%v", key, found, err)
		}
	}
}

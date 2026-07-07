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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestValidateJavaScriptNameTreeDanglingNameRelaxed(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("SomeRandomName")}}
	node := &model.Node{D: d}

	if _, _, err := validateNameTreeDictNamesEntry(testXRef(t, model.ValidationRelaxed), d, "JavaScript", node); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if len(node.Names) != 0 {
		t.Fatalf("relaxed validation preserved dangling name: %d", len(node.Names))
	}
}

func TestValidateJavaScriptNameTreeDanglingNameStrict(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("SomeRandomName")}}

	if _, _, err := validateNameTreeDictNamesEntry(testXRef(t, model.ValidationStrict), d, "JavaScript", &model.Node{D: d}); err == nil {
		t.Fatal("strict validation accepted a dangling JavaScript name")
	}
}

func TestValidateNameTreeDanglingNameRelaxed(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("SomeRandomName")}}

	if _, _, err := validateNameTreeDictNamesEntry(testXRef(t, model.ValidationRelaxed), d, "Dests", &model.Node{D: d}); err == nil {
		t.Fatal("relaxed validation accepted a dangling Dests name")
	}
}

func TestValidateNameTreeDictNamesEntryReportsOddNamesContext(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("OnlyKey")}}

	_, _, err := validateNameTreeDictNamesEntry(testXRef(t, model.ValidationStrict), d, "Dests", &model.Node{D: d})
	requireErrContains(t, err, "name tree Dests Names: odd entry count 1")
}

func TestValidateNameTreeDictNamesEntryReportsValueContext(t *testing.T) {
	d := types.Dict{
		"Names": types.Array{
			types.StringLiteral("PageOne"),
			types.Integer(1),
		},
	}

	_, _, err := validateNameTreeDictNamesEntry(testXRef(t, model.ValidationStrict), d, "Pages", &model.Node{D: d})
	requireErrChainContains(t, err, "name tree Pages key \"PageOne\"", "Pages name tree value")
}

func TestValidateNameTreeDepthReportsMissingKidsArray(t *testing.T) {
	d := types.Dict{"Kids": nil}

	_, _, _, err := validateNameTreeDepth(testXRef(t, model.ValidationStrict), "Dests", d, true, 0)
	requireErrContains(t, err, "name tree Dests: missing Kids array")
}

func TestValidateNameTreeDepthReportsKidObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(nil)

	_, _, _, err := validateNameTreeDepth(xRefTable, "Dests", types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, true, 0)
	requireErrChainContains(t, err, "name tree Dests Kids[0] obj#2", "missing dict")
}

func TestValidateNameTreeDepthReportsNestedKidContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(3, 0)},
	})
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Dict{})

	_, _, _, err := validateNameTreeDepth(xRefTable, "Dests", types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, true, 0)
	requireErrChainContains(t, err, "Kids[0] obj#2", "Kids[0] obj#3", "missing Kids or Names")
}

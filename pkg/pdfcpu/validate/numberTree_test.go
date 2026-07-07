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

func TestValidateStructTreeNumberTreeInvalidValue(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Integer(0), types.Integer(1)}}

	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationRelaxed), d, "StructTree", false); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationStrict), d, "StructTree", false); err == nil {
		t.Fatal("strict validation accepted an invalid StructTree value")
	}
}

func TestValidateStructTreeNumberTreeInvalidKey(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Array{}, types.Array{}}}

	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationRelaxed), d, "StructTree", false); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationStrict), d, "StructTree", false); err == nil {
		t.Fatal("strict validation accepted an invalid StructTree key")
	}
	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationRelaxed), d, "PageLabel", false); err == nil {
		t.Fatal("relaxed validation accepted an invalid PageLabel key")
	}
}

func TestValidateStructTreeNumberTreeInvalidArrayValue(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Integer(0), types.Array{types.Integer(1)}}}

	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationRelaxed), d, "StructTree", false); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationStrict), d, "StructTree", false); err == nil {
		t.Fatal("strict validation accepted an invalid StructTree array value")
	}
}

func TestValidateNumberTreeDictNumsEntryReportsOddNumsContext(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Integer(0)}}

	_, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationStrict), d, "PageLabel", false)
	requireErrContains(t, err, "number tree PageLabel Nums: odd entry count 1")
}

func TestValidateNumberTreeDictNumsEntryReportsValueContext(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Integer(0), types.Integer(1)}}

	_, _, err := validateNumberTreeDictNumsEntry(testXRef(t, model.ValidationStrict), d, "PageLabel", false)
	requireErrChainContains(t, err, "number tree PageLabel key 0", "PageLabel number tree value")
}

func TestValidatePageLabelNumberTreeSkipsMissingDictRelaxed(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	ir := *types.NewIndirectRef(114, 0)
	xRefTable.Table[114] = model.NewXRefTableEntryGen0(nil)
	d := types.Dict{"Nums": types.Array{types.Integer(0), ir}}

	if _, _, err := validateNumberTreeDictNumsEntry(xRefTable, d, "PageLabel", false); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePageLabelNumberTreeRejectsMissingDictStrict(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	ir := *types.NewIndirectRef(114, 0)
	xRefTable.Table[114] = model.NewXRefTableEntryGen0(nil)
	d := types.Dict{"Nums": types.Array{types.Integer(0), ir}}

	_, _, err := validateNumberTreeDictNumsEntry(xRefTable, d, "PageLabel", false)
	if err == nil {
		t.Fatal("expected missing PageLabel dict error")
	}
	if got := err.Error(); got != "number tree PageLabel key 0: PageLabel number tree value: missing dict" {
		t.Fatalf("got %q, want number tree PageLabel key 0: PageLabel number tree value: missing dict", got)
	}
}

func TestValidateNumberTreeDepthReportsMissingKidsArray(t *testing.T) {
	d := types.Dict{"Kids": nil}

	_, _, err := validateNumberTreeDepth(testXRef(t, model.ValidationStrict), "PageLabel", d, true, false, 0)
	requireErrContains(t, err, "number tree PageLabel: missing Kids array")
}

func TestValidateNumberTreeDepthReportsKidObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(nil)

	_, _, err := validateNumberTreeDepth(xRefTable, "PageLabel", types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, true, false, 0)
	requireErrChainContains(t, err, "number tree PageLabel Kids[0] obj#2", "missing dict")
}

func TestValidateNumberTreeDepthReportsNestedKidContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(3, 0)},
	})
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Dict{})

	_, _, err := validateNumberTreeDepth(xRefTable, "PageLabel", types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, true, false, 0)
	requireErrChainContains(t, err, "Kids[0] obj#2", "Kids[0] obj#3", "missing Kids or Nums")
}

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

func TestValidateStructTreeRootDictReportsMissingTypeContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validateStructTreeRootDict(xRefTable, types.Dict{})
	requireErrContains(t, err, "structure tree root: missing Type StructTreeRoot")
}

func TestProcessStructElementDictPgEntryReportsPageContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := processStructElementDictPgEntry(xRefTable, *types.NewIndirectRef(2, 0))
	requireErrContains(t, err, "page obj#2: expected page dict")
}

func TestValidateStructTreeRootDictEntryKArrayReportsIndexContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validateStructTreeRootDictEntryKArray(xRefTable, types.Array{types.Integer(1)}, false)
	requireErrContains(t, err, "structure tree root K[0]: unsupported PDF object")
}

func TestValidateStructElementKArrayReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Boolean(true))

	err := validateStructElementDictEntryKArray(xRefTable, types.Array{*types.NewIndirectRef(2, 0)}, false, 0)
	requireErrChainContains(t, err, "structure element K[0] obj#2", "unsupported PDF object")
}

func TestValidateStructElementKArraySkipsSelfReference(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	selfRef := *types.NewIndirectRef(2, 0)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S": types.Name("Div"),
		"K": types.Array{selfRef},
	})

	err := validateStructElementDictEntryKArray(xRefTable, types.Array{selfRef}, false, 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateStructTreeRootKArrayReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Boolean(true))

	err := validateStructTreeRootDictEntryKArray(xRefTable, types.Array{*types.NewIndirectRef(2, 0)}, false)
	requireErrChainContains(t, err, "structure tree root K[0] obj#2", "unsupported PDF object")
}

func TestValidateStructTreeRootKReportsNestedObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S": types.Name("Div"),
		"K": *types.NewIndirectRef(3, 0),
	})
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Boolean(true))

	err := validateStructTreeRootDictEntryK(xRefTable, *types.NewIndirectRef(2, 0), false)
	requireErrChainContains(t, err, "structure tree root K obj#2", "structure element K obj#3", "unsupported PDF object")
}

func TestValidateStructElementReportsPageReferenceContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[1] = model.NewXRefTableEntryGen0(types.Dict{"Type": types.Name("StructElem")})
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateStructElementDict(xRefTable, types.Dict{
		"S":  types.Name("Div"),
		"P":  *types.NewIndirectRef(1, 0),
		"Pg": *types.NewIndirectRef(2, 0),
	}, false)
	requireErrChainContains(t, err, "structure element Pg", "page obj#2", "expected page dict")
}

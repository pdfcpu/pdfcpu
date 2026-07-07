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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestValidatePagesDictRejectsRecursionDepth verifies page validation respects recursion limits.
func TestValidatePagesDictRejectsRecursionDepth(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	pageCount := 0
	err = validatePagesDictDepth(ctx.XRefTable, nil, 1, false, nil, &pageCount, ctx.XRefTable.MaxRecursionDepth()+1, model.NewPageTreeVisit())
	requireErrIs(t, err, model.ErrMaxRecursionDepthExceeded)
}

// TestValidatePagesPreservesCountDereferenceError verifies page root Count
// validation preserves the underlying dereference failure.
func TestValidatePagesPreservesCountDereferenceError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	osd := types.NewObjectStreamDict()
	lazy := types.NewLazyObjectStreamObject(osd, 1, -1, nil)
	xRefTable.Table[1] = model.NewXRefTableEntryGen0(types.Dict{
		"Count": *types.NewIndirectRef(2, 0),
		"Kids":  types.Array{},
	})
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(lazy)

	_, err := validatePages(xRefTable, types.Dict{"Pages": *types.NewIndirectRef(1, 0)})
	requireErrChainContains(t, err, "page tree root", "obj#1", "corrupt \"Count\"", "unexpected EOF")
	requireErrNotContains(t, err, "wrong type")
}

func TestValidatePagesDictPreservesKidsDereferenceError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	osd := types.NewObjectStreamDict()
	lazy := types.NewLazyObjectStreamObject(osd, 1, -1, nil)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(lazy)

	pageCount := 0
	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{"Kids": *types.NewIndirectRef(2, 0)},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrChainContains(t, err, "page tree", "dereference \"Kids\" entry", "unexpected EOF")
	requireErrNotContains(t, err, "corrupt \"Kids\"")
}

func TestValidatePagesDictReportsGeneralEntryNodeContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	pageCount := 0

	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{
			"MediaBox": types.Integer(1),
			"Kids":     types.Array{},
		},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrChainContains(t, err, "page tree: node obj#1", "MediaBox", "dict=pageDict entry=MediaBox")
}

func TestProcessPagesKidsWrapsKidDereferenceError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	osd := types.NewObjectStreamDict()
	lazy := types.NewLazyObjectStreamObject(osd, 1, -1, nil)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(lazy)

	pageCount := 0
	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{"Kids": types.Array{*types.NewIndirectRef(2, 0)}},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrContains(t, err, "page tree: kid obj#2: dereference")
}

func TestProcessPagesKidsWrapsPageDictValidationError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type":     types.Name("Page"),
		"Parent":   *types.NewIndirectRef(1, 0),
		"MediaBox": types.Integer(1),
	})
	pageCount := 0

	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{"Kids": types.Array{*types.NewIndirectRef(2, 0)}},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrChainContains(t, err, "page tree: page obj#2", "dict=pageDict entry=MediaBox")
}

func TestProcessPagesKidsReportsNonIndirectKidContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	pageCount := 0

	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{"Kids": types.Array{types.Integer(7)}},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrContains(t, err, "parent obj#1 kid[0]: expected indirect reference")
}

func TestProcessPagesKidsReportsCorruptParentContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type":   types.Name("Page"),
		"Parent": *types.NewIndirectRef(99, 0),
	})
	pageCount := 0

	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{"Kids": types.Array{*types.NewIndirectRef(2, 0)}},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrContains(t, err, "page tree: node obj#2: corrupt parent node, expected obj#1, got obj#99")
}

func TestProcessPagesKidsReportsMissingNodeTypeContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Parent": *types.NewIndirectRef(1, 0),
	})
	pageCount := 0

	err := validatePagesDictDepth(
		xRefTable,
		types.Dict{"Kids": types.Array{*types.NewIndirectRef(2, 0)}},
		1,
		false,
		nil,
		&pageCount,
		0,
		model.NewPageTreeVisit(),
	)
	requireErrContains(t, err, "page tree: node obj#2: missing Type")
}

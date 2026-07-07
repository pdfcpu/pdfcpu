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

func TestXRefTableRejectsMissingInput(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		wantErr error
	}{
		{
			name:    "missing context",
			wantErr: model.ErrMissingPDFContext,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{},
			wantErr: model.ErrMissingXRefTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := XRefTable(tt.ctx)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestXRefTableWrapsCatalogError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	ctx := &model.Context{
		Configuration: xRefTable.Conf,
		XRefTable:     xRefTable,
	}

	err := XRefTable(ctx)
	requireErrContains(t, err, "load catalog: missing root dict")
}

func TestValidateRootObjectWrapsPageTreeError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v
	ctx := &model.Context{
		Configuration: xRefTable.Conf,
		XRefTable:     xRefTable,
	}

	err := validateRootObject(ctx, types.Dict{"Type": types.Name("Catalog")})
	requireErrContains(t, err, "catalog Pages: page tree root: missing \"Pages\"")
}

func TestValidateRootObjectWrapsCatalogEntryError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v

	pageTreeRef := *types.NewIndirectRef(1, 0)
	xRefTable.Table[1] = model.NewXRefTableEntryGen0(types.Dict{
		"Type":  types.Name("Pages"),
		"Count": types.Integer(0),
		"Kids":  types.Array{},
	})

	ctx := &model.Context{
		Configuration: xRefTable.Conf,
		XRefTable:     xRefTable,
	}

	err := validateRootObject(ctx, types.Dict{
		"Type":  types.Name("Catalog"),
		"Pages": pageTreeRef,
		"Names": types.Dict{"Unexpected": types.Dict{}},
	})
	requireErrContains(t, err, "catalog Names: name tree: unknown name Unexpected")
}

func TestXRefTablePreservesPageTreeCycleSentinel(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v

	pageTreeRef := *types.NewIndirectRef(1, 0)
	xRefTable.RootDict = types.Dict{
		"Type":  types.Name("Catalog"),
		"Pages": pageTreeRef,
	}
	xRefTable.Table[1] = model.NewXRefTableEntryGen0(types.Dict{
		"Type":   types.Name("Pages"),
		"Count":  types.Integer(1),
		"Kids":   types.Array{pageTreeRef},
		"Parent": pageTreeRef,
	})

	ctx := &model.Context{
		Configuration: xRefTable.Conf,
		XRefTable:     xRefTable,
	}

	err := XRefTable(ctx)
	requireErrIs(t, err, model.ErrPageTreeCycle)
	requireErrChainContains(t, err, "document catalog", "catalog Pages", "circular page tree")
}

func TestValidateSpiderInfoReportsRootContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validateSpiderInfo(
		xRefTable,
		types.Dict{"SpiderInfo": types.Dict{}},
		REQUIRED,
		model.V13,
	)
	requireErrChainContains(t, err, "rootDict.SpiderInfo", "webCaptureInfoDict.V")
}

func TestValidateWebCaptureInfoReportsCaptureCommandContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	captureCommandRef := *types.NewIndirectRef(2, 0)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	err := validateWebCaptureInfoDict(xRefTable, types.Dict{
		"V": types.Float(1),
		"C": types.Array{captureCommandRef},
	})
	requireErrChainContains(t, err, "webCaptureInfoDict.C[0] obj#2", "captureCommandDict.URL")
}

func TestValidatePieceInfoReportsEntryContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	_, err := validatePieceInfo(
		xRefTable,
		types.Dict{"PieceInfo": types.Dict{"App": types.Dict{}}},
		"rootDict",
		"PieceInfo",
		REQUIRED,
		model.V14,
	)
	requireErrChainContains(t, err, "rootDict.PieceInfo", "pieceDict.App", "LastModified")
}

func TestValidatePieceInfoReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	pieceRef := *types.NewIndirectRef(2, 0)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	_, err := validatePieceInfo(
		xRefTable,
		types.Dict{"PieceInfo": types.Dict{"App": pieceRef}},
		"rootDict",
		"PieceInfo",
		REQUIRED,
		model.V14,
	)
	requireErrChainContains(t, err, "rootDict.PieceInfo", "pieceDict.App obj#2", "LastModified")
}

func TestValidateLangReportsRootContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validateLang(
		xRefTable,
		types.Dict{"Lang": types.Integer(1)},
		OPTIONAL,
		model.V10,
	)
	requireErrContains(t, err, "rootDict.Lang")
}

func TestValidatePermissionsReportsRootContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validatePermissions(
		xRefTable,
		types.Dict{"Perms": types.Dict{"UR3": types.Integer(1)}},
		OPTIONAL,
		model.V15,
	)
	requireErrChainContains(t, err, "rootDict.Perms", "permDict.UR3")
}

func TestValidateRequirementsReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	requirementRef := *types.NewIndirectRef(2, 0)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	err := validateRequirements(
		xRefTable,
		types.Dict{"Requirements": types.Array{requirementRef}},
		OPTIONAL,
		model.V17,
	)
	requireErrChainContains(t, err, "rootDict.Requirements[0] obj#2", "requirementDict.S")
}

func TestValidateCollectionReportsSchemaContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	fieldRef := *types.NewIndirectRef(2, 0)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	err := validateCollection(
		xRefTable,
		types.Dict{
			"Collection": types.Dict{
				"Schema": types.Dict{"Title": fieldRef},
			},
		},
		OPTIONAL,
		model.V17,
	)
	requireErrChainContains(t, err, "rootDict.Collection", "Collection.Schema.Title obj#2", "colFlddict.Subtype")
}

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

func TestValidatePropertiesDictPaginationContents(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	v := model.V17
	ctx.XRefTable.HeaderVersion = &v

	d := types.Dict{
		"Contents": types.StringLiteral("Pagination header text"),
		"Subtype":  types.Name("Header"),
		"Type":     types.Name("Pagination"),
	}

	for _, tt := range []struct {
		name string
		mode int
	}{
		{"strict", model.ValidationStrict},
		{"relaxed", model.ValidationRelaxed},
	} {
		ctx.XRefTable.ValidationMode = tt.mode
		if err := validatePropertiesDict(ctx.XRefTable, d); err != nil {
			t.Fatalf("validate properties dict mode=%s: %v", tt.name, err)
		}
	}
}

func TestValidatePropertiesDictReportsUnsupportedKeyContext(t *testing.T) {
	err := validatePropertiesDict(testXRef(t, model.ValidationStrict), types.Dict{
		"OCG": types.Dict{},
	})
	requireErrContains(t, err, "propertiesDict.OCG")
}

func TestValidatePropertiesDictReportsContentsContext(t *testing.T) {
	err := validatePropertiesDict(testXRef(t, model.ValidationStrict), types.Dict{
		"Contents": types.Integer(1),
	})
	requireErrContains(t, err, "propertiesDict.Contents")
}

func TestValidatePropertiesResourceDictReportsNameContext(t *testing.T) {
	err := validatePropertiesResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"P1": types.Dict{"OCMD": types.Dict{}},
	}, model.V10)
	requireErrContains(t, err, "propertiesResourceDict.P1")
}

func TestValidatePropertiesDictReportsEntryObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	err := validatePropertiesDict(xRefTable, types.Dict{
		"OCG": *types.NewIndirectRef(2, 0),
	})
	requireErrContains(t, err, "propertiesDict.OCG obj#2")
}

func TestValidatePropertiesResourceDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"OCMD": types.Dict{},
	})

	err := validatePropertiesResourceDict(xRefTable, types.Dict{
		"P1": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "propertiesResourceDict.P1 obj#2", "propertiesDict.OCMD")
}

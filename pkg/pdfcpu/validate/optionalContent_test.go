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

func TestValidateOCPropertiesMissingD(t *testing.T) {
	rootDict := types.Dict{
		"OCProperties": types.Dict{
			"OCGs": types.Array{},
		},
	}

	for _, tt := range []struct {
		name    string
		mode    int
		wantErr bool
	}{
		{"strict", model.ValidationStrict, true},
		{"relaxed", model.ValidationRelaxed, false},
	} {
		ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
		if err != nil {
			t.Fatal(err)
		}

		v := model.V15
		ctx.XRefTable.HeaderVersion = &v
		ctx.XRefTable.ValidationMode = tt.mode

		err = validateOCProperties(ctx.XRefTable, rootDict, OPTIONAL, model.V15)
		if gotErr := err != nil; gotErr != tt.wantErr {
			t.Fatalf("validate OCProperties missing D mode=%s err=%v", tt.name, err)
		}
	}
}

func TestValidateOptionalContentGroupIntentReportsArrayIndexContext(t *testing.T) {
	err := validateOptionalContentGroupIntent(testXRef(t, model.ValidationStrict), types.Dict{
		"Intent": types.Array{types.Integer(1)},
	}, "optionalContentGroupDict", "Intent", REQUIRED, model.V15)
	requireErrContains(t, err, "optionalContentGroupDict.Intent[0]")
}

func TestValidateOptionalContentGroupArrayReportsIndexContext(t *testing.T) {
	err := validateOptionalContentGroupArray(testXRef(t, model.ValidationStrict), types.Dict{
		"OCGs": types.Array{types.Integer(1)},
	}, "optContentConfigDict", "OCGs", model.V15)
	requireErrContains(t, err, "optContentConfigDict.OCGs[0]")
}

func TestValidateUsageApplicationDictArrayReportsIndexContext(t *testing.T) {
	err := validateUsageApplicationDictArray(testXRef(t, model.ValidationStrict), types.Dict{
		"AS": types.Array{types.Integer(1)},
	}, "optContentConfigDict", "AS", REQUIRED, model.V15)
	requireErrContains(t, err, "optContentConfigDict.AS[0]")
}

func TestValidateOCPropertiesReportsConfigsIndexContext(t *testing.T) {
	err := validateOCProperties(testXRef(t, model.ValidationStrict), types.Dict{
		"OCProperties": types.Dict{
			"OCGs":    types.Array{},
			"D":       types.Dict{},
			"Configs": types.Array{types.Integer(1)},
		},
	}, OPTIONAL, model.V15)
	requireErrContains(t, err, "optContentPropertiesDict.Configs[0]")
}

func TestValidateOptionalContentGroupArrayReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("OCG"),
	})

	err := validateOptionalContentGroupArray(xRefTable, types.Dict{
		"OCGs": types.Array{*types.NewIndirectRef(2, 0)},
	}, "optContentConfigDict", "OCGs", model.V15)
	requireErrChainContains(t, err, "optContentConfigDict.OCGs[0] obj#2", "optionalContentGroupDict.Name")
}

func TestValidateUsageApplicationDictArrayReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Event": types.Name("View"),
	})

	err := validateUsageApplicationDictArray(xRefTable, types.Dict{
		"AS": types.Array{*types.NewIndirectRef(2, 0)},
	}, "optContentConfigDict", "AS", REQUIRED, model.V15)
	requireErrChainContains(t, err, "optContentConfigDict.AS[0] obj#2", "usageAppDict.Category")
}

func TestValidateOCPropertiesReportsConfigsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"BaseState": types.Name("Bogus"),
	})

	err := validateOCProperties(xRefTable, types.Dict{
		"OCProperties": types.Dict{
			"OCGs":    types.Array{},
			"D":       types.Dict{},
			"Configs": types.Array{*types.NewIndirectRef(2, 0)},
		},
	}, OPTIONAL, model.V15)
	requireErrChainContains(t, err, "optContentPropertiesDict.Configs[0] obj#2", "optContentConfigDict.BaseState")
}

func TestValidateOptionalContentReportsEntryObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("OCG"),
	})

	err := validateOptionalContent(xRefTable, types.Dict{
		"OC": *types.NewIndirectRef(2, 0),
	}, "annotDict", "OC", REQUIRED, model.V15)
	requireErrChainContains(t, err, "annotDict.OC obj#2", "optionalContentGroupDict.Name")
}

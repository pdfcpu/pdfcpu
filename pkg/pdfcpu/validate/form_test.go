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

// TestValidateFormFieldDictRejectsCycle verifies form field validation rejects cycles.
func TestValidateFormFieldDictRejectsCycle(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	ir := *types.NewIndirectRef(1, 0)
	if _, err := ctx.IndRefForObject(1, types.Dict{
		"Kids": types.Array{ir},
	}); err != nil {
		t.Fatal(err)
	}

	err = validateFormFieldDict(ctx.XRefTable, ir, nil, false)
	requireErrIs(t, err, model.ErrFormFieldCycle)
	requireErrChainContains(t, err, "form field obj#1 Kids[0] obj#1", "circular form field tree")
}

func TestValidateFormFieldDictAllowsPrivateFieldTypeRelaxed(t *testing.T) {
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

		ctx.XRefTable.ValidationMode = tt.mode
		v := model.V10
		ctx.XRefTable.HeaderVersion = &v

		_, _, err = validateFormFieldDictEntries(ctx.XRefTable, 0, 0, types.Dict{
			"FT": types.Name("KSI"),
		}, true, false, nil, false)
		if gotErr := err != nil; gotErr != tt.wantErr {
			t.Fatalf("validate private field type mode=%s err=%v", tt.name, err)
		}
	}
}

func TestDetectRectArrayRejectsEmptyKids(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	_, err := detectRectArray(xRefTable, types.Dict{
		"Kids": types.Array{},
	}, "formFieldDict")
	requireErrContains(t, err, "form field Kids: empty array")
}

func TestValidateFormFieldsReportsArrayEntryContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validateFormFields(xRefTable, types.Array{types.Integer(1)}, false)
	requireErrContains(t, err, "AcroForm Fields[0]: expected indirect reference")
}

func TestValidateFormFieldsSkipsMissingDictRelaxed(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	ir := *types.NewIndirectRef(114, 0)
	xRefTable.Table[114] = model.NewXRefTableEntryGen0(nil)

	if err := validateFormFields(xRefTable, types.Array{ir}, false); err != nil {
		t.Fatal(err)
	}
}

func TestValidateFormFieldsRejectsMissingDictStrict(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	ir := *types.NewIndirectRef(114, 0)
	xRefTable.Table[114] = model.NewXRefTableEntryGen0(nil)

	err := validateFormFields(xRefTable, types.Array{ir}, false)
	requireErrChainContains(t, err, "AcroForm Fields[0] obj#114", "form field obj#114: missing dict")
}

func TestValidateFormFieldKidsWrapsChildValidationError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[1] = model.NewXRefTableEntryGen0(types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	})
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(nil)

	err := validateFormFieldDict(xRefTable, *types.NewIndirectRef(1, 0), nil, false)
	requireErrChainContains(t, err, "form field obj#1 Kids[0] obj#2", "form field obj#2: missing dict")
}

func TestValidateFormXFAReportsArrayEntryContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v

	err := validateFormXFA(xRefTable, types.Dict{
		"XFA": types.Array{types.Integer(1)},
	}, model.V15)
	requireErrContains(t, err, "AcroForm XFA[0]: expected string")
}

func selfReferentialAcroForm(t *testing.T) (*model.XRefTable, types.Dict) {
	t.Helper()

	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	ir := types.NewIndirectRef(1, 0)
	rootDict := types.Dict{
		"Type":     types.Name("Catalog"),
		"AcroForm": *ir,
		"Fields":   types.Array{*types.NewIndirectRef(4, 0)},
	}
	if _, err := ctx.IndRefForObject(1, rootDict); err != nil {
		t.Fatal(err)
	}
	ctx.Root = ir
	ctx.RootDict = rootDict

	return ctx.XRefTable, rootDict
}

func TestValidateFormRejectsSelfReferentialAcroFormStrict(t *testing.T) {
	xRefTable, rootDict := selfReferentialAcroForm(t)
	xRefTable.ValidationMode = model.ValidationStrict

	err := validateForm(xRefTable, rootDict, OPTIONAL, model.V12)
	requireErrContains(t, err, "AcroForm references root catalog")
}

func TestValidateFormRepairsSelfReferentialAcroFormRelaxed(t *testing.T) {
	xRefTable, rootDict := selfReferentialAcroForm(t)
	xRefTable.ValidationMode = model.ValidationRelaxed

	if err := validateForm(xRefTable, rootDict, OPTIONAL, model.V12); err != nil {
		t.Fatal(err)
	}
	if _, found := rootDict.Find("AcroForm"); found {
		t.Fatal("self-referential AcroForm not removed")
	}
	if xRefTable.Form != nil {
		t.Fatal("self-referential AcroForm registered as form")
	}
}

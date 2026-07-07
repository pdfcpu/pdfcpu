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

func TestValidateActionDictReportsUnsupportedActionType(t *testing.T) {
	err := validateActionDict(testXRef(t, model.ValidationStrict), types.Dict{
		"S": types.Name("Bogus"),
	})
	if err == nil {
		t.Fatal("expected unsupported action type error")
	}
	if got := err.Error(); got != `action Bogus: unsupported action type "Bogus"` {
		t.Fatalf("got %q, want unsupported action type", got)
	}
}

func TestValidateActionDictReportsNextArrayIndexContext(t *testing.T) {
	err := validateActionDict(testXRef(t, model.ValidationStrict), types.Dict{
		"S":   types.Name("URI"),
		"URI": types.StringLiteral("https://pdfcpu.io"),
		"Next": types.Array{
			types.Dict{
				"S":  types.Name("JavaScript"),
				"JS": types.Integer(1),
			},
		},
	})
	requireErrChainContains(t, err, "action Next[0]", "JavaScript.JS")
}

func TestValidateActionDictReportsNextArrayObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.Integer(1),
	})

	err := validateActionDict(xRefTable, types.Dict{
		"S":    types.Name("URI"),
		"URI":  types.StringLiteral("https://pdfcpu.io"),
		"Next": types.Array{*types.NewIndirectRef(2, 0)},
	})
	requireErrChainContains(t, err, "action Next[0] obj#2", "action JavaScript", "JavaScript.JS")
}

func TestValidateActionDictSkipsMissingNextAction(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	ir := *types.NewIndirectRef(3, 0)
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(nil)

	if err := validateActionDict(xRefTable, types.Dict{
		"S":    types.Name("URI"),
		"URI":  types.StringLiteral("https://pdfcpu.io"),
		"Next": ir,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAdditionalActionsReportsKeyContext(t *testing.T) {
	err := validateAdditionalActions(
		testXRef(t, model.ValidationStrict),
		types.Dict{"AA": types.Dict{"BAD": types.Dict{}}},
		"rootDict",
		"AA",
		REQUIRED,
		model.V10,
		"root",
	)
	requireErrContains(t, err, "additional action rootDict.AA")
}

func TestValidateAdditionalActionsReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.Integer(1),
	})

	err := validateAdditionalActions(
		xRefTable,
		types.Dict{"AA": types.Dict{"WC": *types.NewIndirectRef(2, 0)}},
		"rootDict",
		"AA",
		REQUIRED,
		model.V10,
		"root",
	)
	requireErrChainContains(t, err, "additional action rootDict.AA.WC obj#2", "action JavaScript", "JavaScript.JS")
}

func TestValidateOpenActionReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"S":  types.Name("JavaScript"),
		"JS": types.Integer(1),
	})

	err := validateOpenAction(
		xRefTable,
		types.Dict{"OpenAction": *types.NewIndirectRef(2, 0)},
		REQUIRED,
		model.V10,
	)
	requireErrChainContains(t, err, "rootDict.OpenAction obj#2", "action JavaScript", "JavaScript.JS")
}

func TestValidateTargetDictEntryReportsEntryContext(t *testing.T) {
	err := validateTargetDictEntry(
		testXRef(t, model.ValidationStrict),
		types.Dict{"T": types.Dict{"R": types.Name("Invalid")}},
		"GoToE",
		"T",
		REQUIRED,
		model.V10,
	)
	requireErrChainContains(t, err, "GoToE.T", "R", "dict=targetDict entry=R")
}

func TestValidateTargetDictEntryReportsNestedContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"R": types.Name("Invalid"),
	})

	err := validateTargetDictEntry(
		xRefTable,
		types.Dict{
			"T": types.Dict{
				"R": types.Name("P"),
				"T": *types.NewIndirectRef(2, 0),
			},
		},
		"GoToE",
		"T",
		REQUIRED,
		model.V10,
	)
	requireErrChainContains(t, err, "GoToE.T", "targetDict.T obj#2", "R", "dict=targetDict entry=R")
}

func TestValidateGoToEActionReportsTargetContext(t *testing.T) {
	err := validateActionDict(testXRef(t, model.ValidationStrict), types.Dict{
		"S": types.Name("GoToE"),
		"D": types.StringLiteral("chapter1"),
		"T": types.Dict{"R": types.Name("Invalid")},
	})
	requireErrChainContains(t, err, "action GoToE", "GoToE.T", "R", "dict=targetDict entry=R")
}

func TestValidateHideActionDictEntryTReportsArrayIndexContext(t *testing.T) {
	err := validateHideActionDictEntryT(testXRef(t, model.ValidationStrict), types.Array{types.Integer(1)})
	requireErrContains(t, err, "Hide.T[0]")
}

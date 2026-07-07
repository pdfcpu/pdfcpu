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

func TestValidateColorSpaceArrayRejectsEmptyArray(t *testing.T) {
	err := validateColorSpaceArray(testXRef(t, model.ValidationStrict), types.Array{}, IncludePatternCS)
	if err == nil {
		t.Fatal("expected empty color space array error")
	}
	if got := err.Error(); got != "color space array: empty" {
		t.Fatalf("got %q, want color space array: empty", got)
	}
}

func TestValidateColorSpaceArrayReportsNameContext(t *testing.T) {
	err := validateColorSpaceArray(testXRef(t, model.ValidationStrict), types.Array{types.Integer(1)}, IncludePatternCS)
	requireErrContains(t, err, "color space array[0]: expected name")
}

func TestValidateColorSpaceReportsUnsupportedObjectContextStrict(t *testing.T) {
	err := validateColorSpace(testXRef(t, model.ValidationStrict), types.Integer(1), IncludePatternCS)
	if err == nil {
		t.Fatal("expected unsupported color space object error")
	}
	if got := err.Error(); got != "color space: expected name or array, got types.Integer" {
		t.Fatalf("got %q, want color space object context", got)
	}
}

func TestValidateColorSpaceEntryReportsEntryContext(t *testing.T) {
	err := validateColorSpaceEntry(
		testXRef(t, model.ValidationStrict),
		types.Dict{"CS": types.Integer(1)},
		"resourceDict",
		"CS",
		REQUIRED,
		IncludePatternCS,
	)
	requireErrContains(t, err, "resourceDict.CS")
}

func TestValidateColorSpaceResourceDictReportsResourceContext(t *testing.T) {
	err := validateColorSpaceResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"Cs1": types.Integer(1),
	}, model.V10)
	requireErrContains(t, err, "ColorSpace resource Cs1")
}

func TestValidateColorSpaceResourceDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateColorSpaceResourceDict(xRefTable, types.Dict{
		"Cs1": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "ColorSpace resource Cs1 obj#2", "color space: expected name or array")
}

func TestValidateColorSpaceResourceDictIgnoresNilResourceDict(t *testing.T) {
	err := validateColorSpaceResourceDict(testXRef(t, model.ValidationRelaxed), nil, model.V10)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateColorSpaceResourceDictRejectsNilResourceDictStrict(t *testing.T) {
	err := validateColorSpaceResourceDict(testXRef(t, model.ValidationStrict), nil, model.V10)
	if err == nil {
		t.Fatal("expected missing ColorSpace resource dict error")
	}
	if got := err.Error(); got != "ColorSpace resource dict: missing dict" {
		t.Fatalf("got %q, want ColorSpace resource dict: missing dict", got)
	}
}

func TestValidateIndexedColorSpaceLookupTableIgnoresNilObjectRelaxed(t *testing.T) {
	err := validateIndexedColorSpaceLookuptable(testXRef(t, model.ValidationRelaxed), nil, model.V12)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateIndexedColorSpaceLookupTableRejectsNilObjectStrict(t *testing.T) {
	err := validateIndexedColorSpaceLookuptable(testXRef(t, model.ValidationStrict), nil, model.V12)
	if err == nil {
		t.Fatal("expected missing Indexed color space lookup table error")
	}
	if got := err.Error(); got != "Indexed color space lookup table: missing object" {
		t.Fatalf("got %q, want Indexed color space lookup table: missing object", got)
	}
}

func TestValidateICCBasedColorSpaceRejectsNonIndirectProfile(t *testing.T) {
	err := validateICCBasedColorSpace(testXRef(t, model.ValidationStrict), types.Array{
		types.Name(model.ICCBasedCS),
		types.Dict{},
	}, model.V13)
	requireErrContains(t, err, "ICCBased color space profile: expected indirect reference")
}

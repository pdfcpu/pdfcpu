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

func TestValidateTransferFunctionReportsArrayLengthContext(t *testing.T) {
	err := validateTransferFunction(testXRef(t, model.ValidationStrict), types.Array{types.Name("Identity")})
	if err == nil {
		t.Fatal("expected transfer function array length error")
	}
	if got := err.Error(); got != "transfer function array: invalid length 1, expected 4" {
		t.Fatalf("got %q, want transfer function array length context", got)
	}
}

func TestValidateTRReportsArrayIndexContext(t *testing.T) {
	err := validateTREntry(testXRef(t, model.ValidationStrict), types.Dict{
		"TR": types.Array{
			types.Name("Identity"),
			types.Name("Bogus"),
			types.Name("Identity"),
			types.Name("Identity"),
		},
	}, "extGStateDict", "TR", REQUIRED, model.V10)
	requireErrContains(t, err, "extGStateDict.TR: TR array[1]")
}

func TestValidateBlendModeEntryReportsArrayIndexContext(t *testing.T) {
	err := validateBlendModeEntry(testXRef(t, model.ValidationStrict), types.Dict{
		"BM": types.Array{types.Name("Normal"), types.Name("Bogus")},
	}, "extGStateDict", "BM", REQUIRED, model.V14)
	requireErrContains(t, err, "extGStateDict.BM[1]")
}

func TestValidateExtGStateResourceDictReportsResourceContext(t *testing.T) {
	err := validateExtGStateResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"GS1": types.Integer(1),
	}, model.V10)
	requireErrContains(t, err, "ExtGState resource GS1")
}

func TestValidateExtGStateResourceDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateExtGStateResourceDict(xRefTable, types.Dict{
		"GS1": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "ExtGState resource GS1 obj#2", "ExtGState: dereference dict")
}

func TestValidateBGEntryReportsEntryContextStrict(t *testing.T) {
	err := validateBGEntry(testXRef(t, model.ValidationStrict), types.Dict{
		"BG": types.Name("Identity"),
	}, "extGStateDict", "BG", REQUIRED, model.V10)
	if err == nil {
		t.Fatal("expected invalid BG name error in strict mode")
	}
	if got := err.Error(); got != `extGStateDict.BG: invalid name "Identity"` {
		t.Fatalf("got %q, want extGStateDict.BG context", got)
	}
}

func TestValidateExtGStateResourceDictIgnoresNilResource(t *testing.T) {
	err := validateExtGStateResourceDict(testXRef(t, model.ValidationRelaxed), types.Dict{
		"s6": nil,
	}, model.V10)
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateExtGStateResourceDictRejectsNilResourceStrict(t *testing.T) {
	err := validateExtGStateResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"s6": nil,
	}, model.V10)
	requireErrContains(t, err, "ExtGState resource s6: ExtGState: missing dict")
}

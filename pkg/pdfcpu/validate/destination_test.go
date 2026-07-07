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

func TestDestinationArrayFirstElementStreamDict(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	ir := *types.NewIndirectRef(1, 0)
	sd := types.StreamDict{
		Dict: types.Dict{
			"Subtype": types.Name("Image"),
			"Type":    types.Name("XObject"),
		},
	}
	if _, err := ctx.IndRefForObject(1, sd); err != nil {
		t.Fatal(err)
	}

	dest := types.Array{ir, types.Name("FitH"), types.Integer(3507)}

	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	if err := validateDestinationArray(ctx.XRefTable, dest); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}

	ctx.XRefTable.ValidationMode = model.ValidationStrict
	if err := validateDestinationArray(ctx.XRefTable, dest); err == nil {
		t.Fatal("strict validation accepted a stream as destination page")
	}
}

func TestValidateDestinationArrayReportsLengthContext(t *testing.T) {
	err := validateDestinationArray(testXRef(t, model.ValidationStrict), types.Array{types.Integer(1)})
	if err == nil {
		t.Fatal("expected invalid destination array length error")
	}
	if got := err.Error(); got != "destination array: invalid length 1" {
		t.Fatalf("got %q, want destination array length context", got)
	}
}

func TestValidateDestinationArrayReportsModeContext(t *testing.T) {
	err := validateDestinationArray(testXRef(t, model.ValidationStrict), types.Array{
		types.Dict{"Type": types.Name("Page")},
		types.Name("Bogus"),
	})
	if err == nil {
		t.Fatal("expected invalid destination mode error")
	}
	if got := err.Error(); got != `destination array mode: invalid mode "Bogus"` {
		t.Fatalf("got %q, want destination mode context", got)
	}
}

func TestValidateDestinationArrayReportsSecondElementContext(t *testing.T) {
	err := validateDestinationArray(testXRef(t, model.ValidationStrict), types.Array{
		types.Dict{"Type": types.Name("Page")},
		types.Integer(1),
	})
	requireErrContains(t, err, "destination array[1]: expected name")
}

func TestValidateActionDestinationEntryReportsEntryContext(t *testing.T) {
	err := validateActionDestinationEntry(
		testXRef(t, model.ValidationStrict),
		types.Dict{"D": types.Integer(1)},
		"GoTo",
		"D",
		REQUIRED,
		model.V10,
	)
	requireErrContains(t, err, "GoTo.D: destination: unsupported object type")
}

func TestValidateActionDestinationEntryReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateActionDestinationEntry(
		xRefTable,
		types.Dict{"D": *types.NewIndirectRef(2, 0)},
		"GoTo",
		"D",
		REQUIRED,
		model.V10,
	)
	requireErrChainContains(t, err, "GoTo.D obj#2", "destination: unsupported object type")
}

func TestValidateDestinationDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Array{
		types.Dict{"Type": types.Name("Page")},
		types.Integer(1),
	})

	err := validateDestinationDict(xRefTable, types.Dict{"D": *types.NewIndirectRef(2, 0)})
	requireErrChainContains(t, err, "destination dictionary.D obj#2", "destination array[1]")
}

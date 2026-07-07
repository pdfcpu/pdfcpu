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

func TestValidateFontEncodingReportsEntryContext(t *testing.T) {
	err := validateFontEncoding(testXRefVersion(t, model.ValidationStrict, model.V16), types.Dict{
		"Encoding": types.Integer(1),
	}, "type1FontDict", REQUIRED)
	requireErrContains(t, err, "type1FontDict.Encoding")
}

func TestValidateCIDToGIDMapReportsTypeContext(t *testing.T) {
	err := validateCIDToGIDMap(testXRefVersion(t, model.ValidationStrict, model.V16), types.Integer(1))
	requireErrContains(t, err, "CIDToGIDMap: expected Identity name or stream dict")
}

func TestValidateDescendantFontsReportsIndexContext(t *testing.T) {
	err := validateDescendantFonts(testXRefVersion(t, model.ValidationStrict, model.V16), types.Dict{
		"DescendantFonts": types.Array{types.Integer(1)},
	}, "type0FontDict", REQUIRED)
	requireErrContains(t, err, "type0FontDict.DescendantFonts[0]")
}

func TestValidateDescendantFontsReportsObjectContext(t *testing.T) {
	xRefTable := testXRefVersion(t, model.ValidationStrict, model.V16)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(nil)

	err := validateDescendantFonts(xRefTable, types.Dict{
		"DescendantFonts": types.Array{*types.NewIndirectRef(2, 0)},
	}, "type0FontDict", REQUIRED)
	requireErrChainContains(t, err, "type0FontDict.DescendantFonts[0] obj#2", "missing required descendant font dict")
}

func TestValidateFontResourceDictReportsResourceContext(t *testing.T) {
	err := validateFontResourceDict(testXRefVersion(t, model.ValidationStrict, model.V16), types.Dict{
		"F1": types.Integer(1),
	}, model.V10)
	requireErrContains(t, err, "Font resource F1")
}

func TestValidateFontResourceDictReportsIndirectFontObjectContext(t *testing.T) {
	xRefTable := testXRefVersion(t, model.ValidationStrict, model.V16)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Font"),
	})

	err := validateFontResourceDict(xRefTable, types.Dict{
		"F1": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "Font resource F1", "font obj#2", "missing Subtype")
}

func TestValidateFontResourceDictAcceptsDirectFontDict(t *testing.T) {
	err := validateFontResourceDict(testXRefVersion(t, model.ValidationStrict, model.V16), types.Dict{
		"F1": types.Dict{
			"Type":     types.Name("Font"),
			"Subtype":  types.Name("Type1"),
			"BaseFont": types.Name("Helvetica"),
		},
	}, model.V10)
	if err != nil {
		t.Fatal(err)
	}
}

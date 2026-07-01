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

func newNumberTreeTestXRef(t *testing.T, mode int) *model.XRefTable {
	t.Helper()
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = mode
	return ctx.XRefTable
}

func TestValidateStructTreeNumberTreeInvalidValue(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Integer(0), types.Integer(1)}}

	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationRelaxed), d, "StructTree", false); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationStrict), d, "StructTree", false); err == nil {
		t.Fatal("strict validation accepted an invalid StructTree value")
	}
}

func TestValidateStructTreeNumberTreeInvalidKey(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Array{}, types.Array{}}}

	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationRelaxed), d, "StructTree", false); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationStrict), d, "StructTree", false); err == nil {
		t.Fatal("strict validation accepted an invalid StructTree key")
	}
	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationRelaxed), d, "PageLabel", false); err == nil {
		t.Fatal("relaxed validation accepted an invalid PageLabel key")
	}
}

func TestValidateStructTreeNumberTreeInvalidArrayValue(t *testing.T) {
	d := types.Dict{"Nums": types.Array{types.Integer(0), types.Array{types.Integer(1)}}}

	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationRelaxed), d, "StructTree", false); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if _, _, err := validateNumberTreeDictNumsEntry(newNumberTreeTestXRef(t, model.ValidationStrict), d, "StructTree", false); err == nil {
		t.Fatal("strict validation accepted an invalid StructTree array value")
	}
}

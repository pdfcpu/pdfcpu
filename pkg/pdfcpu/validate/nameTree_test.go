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

func newNameTreeTestXRef(t *testing.T, mode int) *model.XRefTable {
	t.Helper()

	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = mode

	return ctx.XRefTable
}

func TestValidateJavaScriptNameTreeDanglingNameRelaxed(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("SomeRandomName")}}
	node := &model.Node{D: d}

	if _, _, err := validateNameTreeDictNamesEntry(newNameTreeTestXRef(t, model.ValidationRelaxed), d, "JavaScript", node); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}
	if len(node.Names) != 0 {
		t.Fatalf("relaxed validation preserved dangling name: %d", len(node.Names))
	}
}

func TestValidateJavaScriptNameTreeDanglingNameStrict(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("SomeRandomName")}}

	if _, _, err := validateNameTreeDictNamesEntry(newNameTreeTestXRef(t, model.ValidationStrict), d, "JavaScript", &model.Node{D: d}); err == nil {
		t.Fatal("strict validation accepted a dangling JavaScript name")
	}
}

func TestValidateNameTreeDanglingNameRelaxed(t *testing.T) {
	d := types.Dict{"Names": types.Array{types.StringLiteral("SomeRandomName")}}

	if _, _, err := validateNameTreeDictNamesEntry(newNameTreeTestXRef(t, model.ValidationRelaxed), d, "Dests", &model.Node{D: d}); err == nil {
		t.Fatal("relaxed validation accepted a dangling Dests name")
	}
}

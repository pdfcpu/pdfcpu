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

func TestValidateAnnotationsArrayReportsIndirectReferenceRequirement(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	_, err := validateAnnotationsArray(xRefTable, types.Array{types.Dict{"Subtype": types.Name("Text")}})
	requireErrContains(t, err, "page annotation array: expected indirect references")
}

func TestValidateAnnotationsArrayWrapsAnnotationDereferenceError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	osd := types.NewObjectStreamDict()
	lazy := types.NewLazyObjectStreamObject(osd, 1, -1, nil)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(lazy)

	_, err := validateAnnotationsArray(xRefTable, types.Array{*types.NewIndirectRef(2, 0)})
	requireErrChainContains(t, err, "page annotation obj#2", "dereference", "unexpected EOF")
}

func TestValidateAnnotationsArrayWrapsAnnotationValidationError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Subtype": types.Name("Text"),
	})

	_, err := validateAnnotationsArray(xRefTable, types.Array{*types.NewIndirectRef(2, 0)})
	requireErrChainContains(t, err, "page annotation obj#2", "validate", "dict=annotDict required entry=Rect")
}

func TestValidatePagesAnnotationsReportsMissingPageNodeTypeContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	_, err := validatePagesAnnotations(xRefTable, types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, 0)
	requireErrChainContains(t, err, "page tree annotation walk: kid obj#2", "missing page node Type")
}

func TestValidatePagesAnnotationsWrapsPageAnnotationErrorWithPageNumber(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type":   types.Name("Page"),
		"Annots": types.Array{types.Dict{"Subtype": types.Name("Text")}},
	})

	_, err := validatePagesAnnotations(xRefTable, types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, 0)
	requireErrChainContains(t, err, "page annotations: page 1", "page annotation array: expected indirect references")
}

func TestValidatePagesAnnotationsWrapsNestedPageTreeContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Pages"),
		"Kids": types.Array{
			*types.NewIndirectRef(3, 0),
		},
	})
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Dict{})

	_, err := validatePagesAnnotations(xRefTable, types.Dict{
		"Kids": types.Array{*types.NewIndirectRef(2, 0)},
	}, 0)
	requireErrChainContains(t, err, "kid obj#2", "kid obj#3", "missing page node Type")
}

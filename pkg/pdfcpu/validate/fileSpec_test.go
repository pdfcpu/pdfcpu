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

func TestValidateFileSpecificationReportsInvalidTypeContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	_, err := validateFileSpecification(xRefTable, types.Integer(1))
	requireErrContains(t, err, "file specification: invalid type")
}

func TestValidateURLSpecificationWrapsDereferenceError(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	_, err := validateURLSpecification(xRefTable, types.Integer(1))
	requireErrContains(t, err, "URL specification: dereference dict")
}

func TestValidateRFDictFilesArrayReportsElementContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	err := validateRFDictFilesArray(xRefTable, types.Array{types.StringLiteral("not-a-stream"), types.StringLiteral("description")})
	requireErrContains(t, err, "related files array[0]: embedded file stream")
}

func TestValidateFileSpecEntryReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Filespec"),
	})

	_, err := validateFileSpecEntry(
		xRefTable,
		types.Dict{"F": *types.NewIndirectRef(2, 0)},
		"GoToR",
		"F",
		REQUIRED,
		model.V11,
	)
	requireErrChainContains(t, err, "GoToR.F obj#2", "file specification obj#2 dict", "required entry=F")
}

func TestValidateActionFileSpecEntryPreservesActionContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Filespec"),
	})

	err := validateActionDict(xRefTable, types.Dict{
		"S": types.Name("GoToR"),
		"F": *types.NewIndirectRef(2, 0),
	})
	requireErrChainContains(t, err, "action GoToR", "GoToR.F obj#2", "file specification obj#2 dict")
}

func TestValidateURLSpecEntryReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{
		"FS": types.Name("URL"),
	})

	_, err := validateURLSpecEntry(
		xRefTable,
		types.Dict{"F": *types.NewIndirectRef(2, 0)},
		"SubmitForm",
		"F",
		REQUIRED,
		model.V10,
	)
	requireErrChainContains(t, err, "SubmitForm.F obj#2", "URL specification", "required entry=F")
}

func TestValidateFileSpecEFDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateFileSpecDict(xRefTable, types.Dict{
		"Type": types.Name("Filespec"),
		"F":    types.StringLiteral("attachment.txt"),
		"EF": types.Dict{
			"F": *types.NewIndirectRef(2, 0),
		},
	})
	requireErrChainContains(t, err, "fileSpec.EF.F obj#2", "stream dict: invalid type")
}

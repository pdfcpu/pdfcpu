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

func TestValidateDocInfoDictEntryReportsKeyContext(t *testing.T) {
	_, err := validateDocInfoDictEntry(testXRef(t, model.ValidationStrict), "Title", types.Integer(1))
	requireErrContains(t, err, `document info entry "Title"`)
}

func TestValidateDocumentInfoObjectReportsDereferenceContextStrict(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	ir := types.NewIndirectRef(3, 0)
	xRefTable.Info = ir
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateDocumentInfoObject(xRefTable)
	requireErrContains(t, err, "document info: dereference dict")
}

func TestValidateDocumentInfoObjectSkipsInvalidInfoDictRelaxed(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	ir := types.NewIndirectRef(3, 0)
	xRefTable.Info = ir
	xRefTable.Table[3] = model.NewXRefTableEntryGen0(types.Integer(1))

	if err := validateDocumentInfoObject(xRefTable); err != nil {
		t.Fatal(err)
	}
	if xRefTable.Info != nil {
		t.Fatal("invalid document info object not skipped")
	}
}

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

func TestValidateFlexBooleanEntryAcceptsNameInRelaxedMode(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationRelaxed)
	v := model.V17
	xRefTable.HeaderVersion = &v
	d := types.Dict{"DisplayDocTitle": types.Name("true")}

	b, err := validateFlexBooleanEntry(xRefTable, d, "viewerPreferences", "DisplayDocTitle", OPTIONAL, model.V14)
	if err != nil {
		t.Fatal(err)
	}
	if b == nil || !*b {
		t.Fatalf("expected true, got %v", b)
	}
}

func TestValidateFlexBooleanEntryRejectsNameInStrictMode(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v
	d := types.Dict{"DisplayDocTitle": types.Name("true")}

	if _, err := validateFlexBooleanEntry(xRefTable, d, "viewerPreferences", "DisplayDocTitle", OPTIONAL, model.V14); err == nil {
		t.Fatal("strict validation accepted name boolean")
	}
}

func TestValidateFloatReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)

	_, err := validateFloat(xRefTable, types.Integer(1), nil)
	requireErrContains(t, err, "float object: invalid type")
}

func TestValidateDateEntryReportsDictEntryContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	v := model.V17
	xRefTable.HeaderVersion = &v
	d := types.Dict{"ModDate": types.StringLiteral("not-a-date")}

	_, err := validateDateEntry(xRefTable, d, "infoDict", "ModDate", REQUIRED, model.V10)
	requireErrContains(t, err, "dict=infoDict entry=ModDate invalid date")
}

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

func TestValidateViewerPreferencesReportsRootEntryContextStrict(t *testing.T) {
	err := validateViewerPreferences(
		testXRefVersion(t, model.ValidationStrict, model.V20),
		types.Dict{"ViewerPreferences": types.Integer(1)},
		OPTIONAL,
		model.V12,
	)
	requireErrContains(t, err, "rootDict.ViewerPreferences")
}

func TestValidateViewerPreferencesReportsRelaxedArrayIndexContext(t *testing.T) {
	err := validateViewerPreferences(
		testXRefVersion(t, model.ValidationRelaxed, model.V20),
		types.Dict{"ViewerPreferences": types.Array{types.Integer(1)}},
		OPTIONAL,
		model.V12,
	)
	requireErrContains(t, err, "rootDict.ViewerPreferences[0]")
}

func TestValidateViewerPreferencesReportsPrintPageRangeContext(t *testing.T) {
	err := validateViewerPreferences(
		testXRefVersion(t, model.ValidationStrict, model.V20),
		types.Dict{
			"ViewerPreferences": types.Dict{
				"PrintPageRange": types.Array{types.Integer(2), types.Integer(1)},
			},
		},
		OPTIONAL,
		model.V12,
	)
	requireErrContains(t, err, "ViewerPreferences.PrintPageRange")
}

func TestValidateViewerPreferencesReportsEnforcePrintScalingConflict(t *testing.T) {
	err := validateViewerPreferences(
		testXRefVersion(t, model.ValidationStrict, model.V20),
		types.Dict{
			"ViewerPreferences": types.Dict{
				"PrintScaling": types.Name("AppDefault"),
				"Enforce":      types.Array{types.Name("PrintScaling")},
			},
		},
		OPTIONAL,
		model.V12,
	)
	if err == nil {
		t.Fatal("expected Enforce PrintScaling conflict")
	}
	if got := err.Error(); got != `ViewerPreferences.Enforce: PrintScaling requires PrintScaling != "AppDefault"` {
		t.Fatalf("got %q, want Enforce PrintScaling conflict context", got)
	}
}

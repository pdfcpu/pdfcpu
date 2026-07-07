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

func TestValidateShadingReportsUnsupportedObjectContext(t *testing.T) {
	err := validateShading(testXRef(t, model.ValidationStrict), types.Integer(1))
	if err == nil {
		t.Fatal("expected unsupported shading object error")
	}
	if got := err.Error(); got != "shading: expected dict or stream dict, got types.Integer" {
		t.Fatalf("got %q, want shading object context", got)
	}
}

func TestValidateShadingDictReportsFunctionContext(t *testing.T) {
	err := validateShadingDict(testXRef(t, model.ValidationStrict), types.Dict{
		"ShadingType": types.Integer(1),
	})
	requireErrContains(t, err, "shading dict type 1: functionBasedShadingDict.Function")
}

func TestValidateShadingStreamDictReportsMeshEntryContext(t *testing.T) {
	err := validateShadingStreamDict(testXRef(t, model.ValidationStrict), &types.StreamDict{
		Dict: types.Dict{
			"ShadingType": types.Integer(4),
		},
	})
	requireErrContains(t, err, "shading stream dict type 4: freeFormGouraudShadedTriangleMeshesDict.BitsPerCoordinate")
}

func TestValidateShadingResourceDictReportsResourceNameContext(t *testing.T) {
	err := validateShadingResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"Sh1": types.Integer(1),
	}, model.V13)
	requireErrContains(t, err, "shadingResourceDict.Sh1")
}

func TestValidateShadingResourceDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateShadingResourceDict(xRefTable, types.Dict{
		"Sh1": *types.NewIndirectRef(2, 0),
	}, model.V13)
	requireErrChainContains(t, err, "shadingResourceDict.Sh1 obj#2", "shading: expected dict or stream dict")
}

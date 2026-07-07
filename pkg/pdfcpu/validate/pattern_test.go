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

func TestValidateTilingPatternDictReportsResourcesContext(t *testing.T) {
	err := validateTilingPatternDict(testXRef(t, model.ValidationStrict), &types.StreamDict{
		Dict: types.Dict{
			"PatternType": types.Integer(1),
			"PaintType":   types.Integer(1),
			"TilingType":  types.Integer(1),
			"BBox":        types.Array{types.Integer(0), types.Integer(0), types.Integer(1), types.Integer(1)},
			"XStep":       types.Integer(1),
			"YStep":       types.Integer(1),
		},
	}, model.V10)
	requireErrContains(t, err, "tilingPatternDict.Resources")
}

func TestValidateShadingPatternDictReportsShadingContext(t *testing.T) {
	err := validateShadingPatternDict(testXRef(t, model.ValidationStrict), types.Dict{
		"PatternType": types.Integer(2),
	}, model.V13)
	requireErrContains(t, err, "shadingPatternDict.Shading")
}

func TestValidatePatternResourceDictReportsPatternNameContext(t *testing.T) {
	err := validatePatternResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"P1": types.Integer(1),
	}, model.V10)
	requireErrContains(t, err, "patternResourceDict.P1")
}

func TestValidatePatternResourceDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validatePatternResourceDict(xRefTable, types.Dict{
		"P1": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "patternResourceDict.P1 obj#2", "corrupt obj type")
}

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

// TestChoiceFieldDAValidation checks the DA requirement and inheritance for text and choice fields.
func TestChoiceFieldDAValidation(t *testing.T) {
	tests := []struct {
		name       string
		fieldType  types.Name
		version    model.Version
		mode       int
		terminal   bool
		requiresDA bool
		da         types.Object
		wantErr    bool
		wantDA     bool
	}{
		{"strict text without DA", "Tx", model.V17, model.ValidationStrict, true, true, nil, true, false},
		{"strict choice without DA in PDF 1.2", "Ch", model.V12, model.ValidationStrict, true, true, nil, true, false},
		{"strict choice without DA in PDF 1.7", "Ch", model.V17, model.ValidationStrict, true, true, nil, true, false},
		{"strict choice without DA in PDF 2.0", "Ch", model.V20, model.ValidationStrict, true, true, nil, true, false},
		{"strict choice with DA", "Ch", model.V17, model.ValidationStrict, true, true, types.StringLiteral("/F1 12 Tf"), false, true},
		{"strict choice with effective default", "Ch", model.V17, model.ValidationStrict, true, false, nil, false, false},
		{"relaxed choice without DA", "Ch", model.V17, model.ValidationRelaxed, true, true, nil, false, false},
		{"relaxed choice with DA", "Ch", model.V17, model.ValidationRelaxed, true, true, types.StringLiteral("/F1 12 Tf"), false, true},
		{"relaxed choice with invalid DA", "Ch", model.V17, model.ValidationRelaxed, true, true, types.Integer(42), true, false},
		{"relaxed parent choice with DA", "Ch", model.V17, model.ValidationRelaxed, false, true, types.StringLiteral("/F1 12 Tf"), false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
			if err != nil {
				t.Fatal(err)
			}
			v := tc.version
			x := ctx.XRefTable
			x.HeaderVersion = &v
			x.ValidationMode = tc.mode
			d := types.Dict{
				"FT": tc.fieldType,
				"T":  types.StringLiteral("field"),
			}
			if tc.fieldType == "Ch" {
				d["Opt"] = types.Array{types.StringLiteral("option")}
			}
			if tc.da != nil {
				d["DA"] = tc.da
			}
			_, hasDA, err := validateFormFieldDictEntries(t.Context(), x, 0, 0, d, tc.terminal, false, nil, tc.requiresDA)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validation error: got %v, want error %t", err, tc.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "DA") {
				t.Fatalf("validation error: got %v, want DA error", err)
			}
			if err == nil && hasDA != tc.wantDA {
				t.Fatalf("hasDA: got %t, want %t", hasDA, tc.wantDA)
			}
		})
	}
}

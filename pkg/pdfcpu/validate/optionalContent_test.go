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

func TestValidateOCPropertiesMissingD(t *testing.T) {
	rootDict := types.Dict{
		"OCProperties": types.Dict{
			"OCGs": types.Array{},
		},
	}

	for _, tt := range []struct {
		name    string
		mode    int
		wantErr bool
	}{
		{"strict", model.ValidationStrict, true},
		{"relaxed", model.ValidationRelaxed, false},
	} {
		ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
		if err != nil {
			t.Fatal(err)
		}

		v := model.V15
		ctx.XRefTable.HeaderVersion = &v
		ctx.XRefTable.ValidationMode = tt.mode

		err = validateOCProperties(ctx.XRefTable, rootDict, OPTIONAL, model.V15)
		if gotErr := err != nil; gotErr != tt.wantErr {
			t.Fatalf("validate OCProperties missing D mode=%s err=%v", tt.name, err)
		}
	}
}

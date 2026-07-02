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

func TestDestinationArrayFirstElementStreamDict(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	ir := *types.NewIndirectRef(1, 0)
	sd := types.StreamDict{
		Dict: types.Dict{
			"Subtype": types.Name("Image"),
			"Type":    types.Name("XObject"),
		},
	}
	if _, err := ctx.IndRefForObject(1, sd); err != nil {
		t.Fatal(err)
	}

	dest := types.Array{ir, types.Name("FitH"), types.Integer(3507)}

	ctx.XRefTable.ValidationMode = model.ValidationRelaxed
	if err := validateDestinationArray(ctx.XRefTable, dest); err != nil {
		t.Fatalf("relaxed validation: %v", err)
	}

	ctx.XRefTable.ValidationMode = model.ValidationStrict
	if err := validateDestinationArray(ctx.XRefTable, dest); err == nil {
		t.Fatal("strict validation accepted a stream as destination page")
	}
}

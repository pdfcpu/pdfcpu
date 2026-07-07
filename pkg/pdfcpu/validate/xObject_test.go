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

func TestValidateXObjectResourceDictReportsResourceNameContext(t *testing.T) {
	err := validateXObjectResourceDict(testXRef(t, model.ValidationStrict), types.Dict{
		"Im1": types.Integer(1),
	}, model.V10)
	requireErrContains(t, err, "XObject resource Im1")
}

func TestValidateXObjectResourceDictReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Integer(1))

	err := validateXObjectResourceDict(xRefTable, types.Dict{
		"Im1": *types.NewIndirectRef(2, 0),
	}, model.V10)
	requireErrChainContains(t, err, "XObject resource Im1 obj#2", "xObject stream dict: dereference stream dict")
}

func TestValidateXObjectStreamDictReportsSubtypeContext(t *testing.T) {
	err := validateXObjectStreamDict(testXRef(t, model.ValidationStrict), types.StreamDict{
		Dict: types.Dict{
			"Type":    types.Name("XObject"),
			"Subtype": types.Name("Bogus"),
		},
	})
	if err == nil {
		t.Fatal("expected unknown XObject subtype error")
	}
	if got := err.Error(); got != `xObjectStreamDict.Subtype: unknown subtype "Bogus"` {
		t.Fatalf("got %q, want XObject subtype context", got)
	}
}

func TestValidateMaskEntryReportsEntryContext(t *testing.T) {
	err := validateMaskEntry(testXRef(t, model.ValidationStrict), types.Dict{
		"Mask": types.Integer(1),
	}, "imageStreamDict", "Mask", OPTIONAL, model.V13)
	requireErrContains(t, err, "imageStreamDict.Mask")
}

func TestValidateXObjectTypeRepairsRelaxedXobject(t *testing.T) {
	sd := types.StreamDict{
		Dict: types.Dict{
			"Type": types.Name("Xobject"),
		},
	}
	if err := validateXObjectType(testXRef(t, model.ValidationRelaxed), &sd); err != nil {
		t.Fatal(err)
	}
	if got := sd.Dict.NameEntry("Type"); got == nil || *got != "XObject" {
		t.Fatalf("got Type %v, want XObject", got)
	}
}

func TestValidateFormXObjectRequiresLastModifiedWithPieceInfo(t *testing.T) {
	err := validateFormStreamDict(testXRef(t, model.ValidationStrict), &types.StreamDict{
		Dict: types.Dict{
			"BBox": types.Array{
				types.Integer(0),
				types.Integer(0),
				types.Integer(100),
				types.Integer(100),
			},
			"PieceInfo": types.Dict{
				"App": types.Dict{
					"LastModified": types.StringLiteral("D:2017"),
				},
			},
		},
	})
	requireErrContains(t, err, `missing "LastModified" (required by "PieceInfo")`)
}

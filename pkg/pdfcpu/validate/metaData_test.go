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

func TestValidateMetadataStreamReportsMetadataEntryContext(t *testing.T) {
	_, err := validateMetadataStream(testXRef(t, model.ValidationStrict), types.Dict{
		"Metadata": types.Integer(1),
	}, REQUIRED, model.V14)
	requireErrContains(t, err, "dict.Metadata")
}

func TestValidateMetadataStreamReportsTypeContext(t *testing.T) {
	_, err := validateMetadataStream(testXRef(t, model.ValidationStrict), types.Dict{
		"Metadata": types.StreamDict{
			Dict: types.Dict{
				"Type": types.Name("Bogus"),
			},
		},
	}, REQUIRED, model.V14)
	requireErrContains(t, err, "metaDataDict.Type")
}

func TestValidateMetadataStreamReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.StreamDict{
		Dict: types.Dict{
			"Type": types.Name("Bogus"),
		},
	})

	_, err := validateMetadataStream(xRefTable, types.Dict{
		"Metadata": *types.NewIndirectRef(2, 0),
	}, REQUIRED, model.V14)
	requireErrChainContains(t, err, "dict.Metadata obj#2", "metaDataDict.Type")
}

func TestValidateOutputIntentsReportsObjectContext(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.Dict{})

	err := validateOutputIntents(xRefTable, types.Dict{
		"OutputIntents": types.Array{*types.NewIndirectRef(2, 0)},
	}, REQUIRED, model.V14)
	requireErrChainContains(t, err, "rootDict.OutputIntents[0] obj#2", "outputIntentDict")
}

func TestValidateRootMetadataPopulatesDocumentInfo(t *testing.T) {
	xRefTable := testXRef(t, model.ValidationStrict)
	xRefTable.CatalogXMPMeta = &model.XMPMeta{
		RDF: model.RDF{
			Description: model.Description{
				Title: model.Title{
					Alt: model.Alt{Entries: []string{"Title"}},
				},
				Author: model.Creator{
					Seq: model.Seq{Entries: []string{"Author"}},
				},
				Subject: model.Desc{
					Alt: model.Alt{Entries: []string{"Subject"}},
				},
				Creator:  "Creator",
				Producer: "Producer",
				Keywords: "alpha, beta;gamma",
			},
		},
	}

	if err := validateRootMetadata(xRefTable, types.Dict{}, OPTIONAL, model.V14); err != nil {
		t.Fatal(err)
	}
	if xRefTable.Title != "Title" || xRefTable.Author != "Author" || xRefTable.Subject != "Subject" {
		t.Fatalf("unexpected document info title=%q author=%q subject=%q", xRefTable.Title, xRefTable.Author, xRefTable.Subject)
	}
	if xRefTable.Creator != "Creator" || xRefTable.Producer != "Producer" {
		t.Fatalf("unexpected document info creator=%q producer=%q", xRefTable.Creator, xRefTable.Producer)
	}
	for _, keyword := range []string{"alpha", "beta", "gamma"} {
		if !xRefTable.KeywordList[keyword] {
			t.Fatalf("missing keyword %q", keyword)
		}
	}
}

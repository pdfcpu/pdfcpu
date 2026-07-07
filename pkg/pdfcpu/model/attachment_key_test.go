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

package model

import (
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func canonicalAttachmentKeyContext() *Context {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	version := V17
	xRefTable.HeaderVersion = &version
	xRefTable.Valid = true

	const key = "canonical-id"
	sd := types.StreamDict{
		Dict: types.NewDict(),
		Raw:  []byte("attachment content"),
	}
	fileSpec := types.Dict{
		"UF":   types.StringLiteral("report.txt"),
		"Desc": types.StringLiteral("quarterly report"),
		"EF":   types.Dict{"F": sd},
	}
	xRefTable.Names["EmbeddedFiles"] = &Node{
		Kmin:  key,
		Kmax:  key,
		Names: []entry{{k: key, v: fileSpec}},
	}

	return &Context{
		Configuration: xRefTable.Conf,
		XRefTable:     xRefTable,
	}
}

func TestExtractAttachmentsPreservesCanonicalKeyForAllLookups(t *testing.T) {
	for _, tt := range []struct {
		name   string
		lookup string
	}{
		{name: "ID", lookup: "canonical-id"},
		{name: "filename", lookup: "report.txt"},
		{name: "description", lookup: "quarterly report"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			aa, err := canonicalAttachmentKeyContext().ExtractAttachments([]string{tt.lookup})
			if err != nil {
				t.Fatal(err)
			}
			if len(aa) != 1 {
				t.Fatalf("expected one attachment, got %d", len(aa))
			}
			if aa[0].ID != "canonical-id" {
				t.Fatalf("expected canonical attachment key, got %q", aa[0].ID)
			}
		})
	}
}

func TestListAttachmentsSkipsEmbeddedStreamDecoding(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	version := V17
	xRefTable.HeaderVersion = &version
	xRefTable.Valid = true
	sd := types.StreamDict{
		Dict:           types.NewDict(),
		Raw:            []byte("not a flate stream"),
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	probe := sd
	if err := decodeFileSpecStreamDict(&probe); err == nil {
		t.Fatal("expected intentionally undecodable embedded stream")
	}
	fileSpec := types.Dict{
		"UF":   types.StringLiteral("report.txt"),
		"Desc": types.StringLiteral("quarterly report"),
		"EF":   types.Dict{"F": sd},
	}
	xRefTable.Names["EmbeddedFiles"] = &Node{
		Kmin:  "canonical-id",
		Kmax:  "canonical-id",
		Names: []entry{{k: "canonical-id", v: fileSpec}},
	}
	ctx := &Context{
		Configuration: xRefTable.Conf,
		XRefTable:     xRefTable,
	}

	aa, err := ctx.ListAttachments()

	if err != nil {
		t.Fatal(err)
	}
	if len(aa) != 1 {
		t.Fatalf("expected one attachment, got %d", len(aa))
	}
	if aa[0].ID != "canonical-id" || aa[0].FileName != "report.txt" || aa[0].Desc != "quarterly report" {
		t.Fatalf("unexpected attachment metadata: %+v", aa[0])
	}
}

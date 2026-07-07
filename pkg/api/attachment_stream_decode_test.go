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

package api

import (
	"bytes"
	"compress/zlib"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func pdfWithUndecodableAttachment(t *testing.T, id string) []byte {
	t.Helper()
	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.LocateNameTree("EmbeddedFiles", true); err != nil {
		t.Fatal(err)
	}

	raw := []byte("not a flate stream")
	streamLength := int64(len(raw))
	sd := types.StreamDict{
		Dict: types.Dict{
			"Filter": types.Name(filter.Flate),
			"Length": types.Integer(streamLength),
		},
		Raw:            raw,
		StreamLength:   &streamLength,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate}},
	}
	streamRef, err := ctx.IndRefForNewObject(sd)
	if err != nil {
		t.Fatal(err)
	}
	fileSpec := types.Dict{
		"Type": types.Name("Filespec"),
		"F":    types.StringLiteral(id),
		"UF":   types.StringLiteral(id),
		"EF":   types.Dict{"F": *streamRef},
	}
	fileSpecRef, err := ctx.IndRefForNewObject(fileSpec)
	if err != nil {
		t.Fatal(err)
	}
	nameRefs := model.NameMap{id: []types.Dict{fileSpec}}
	if err := ctx.Names["EmbeddedFiles"].Add(ctx.XRefTable, id, *fileSpecRef, nameRefs, []string{"F", "UF"}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := WriteContext(ctx, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractAttachmentsReportsUndecodableEmbeddedStream(t *testing.T) {
	const id = "broken.bin"
	pdf := pdfWithUndecodableAttachment(t, id)

	_, err := ExtractAttachmentsRaw(bytes.NewReader(pdf), "", []string{id}, nil)

	if !errors.Is(err, zlib.ErrHeader) {
		t.Fatalf("expected %v, got %v", zlib.ErrHeader, err)
	}
	for _, want := range []string{
		"extract attachments",
		`file spec "broken.bin"`,
		"decode stream",
		`stream filter[0] "FlateDecode": decode`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

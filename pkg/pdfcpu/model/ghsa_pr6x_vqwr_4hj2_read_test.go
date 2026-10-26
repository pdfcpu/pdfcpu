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

package model_test

// GHSA-pr6x-vqwr-4hj2: streams decoded while reading must be charged to the document's decoded-stream budget, each
// aliased xref entry separately, except where the decoded content is the raw stream data.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-pr6x-vqwr-4hj2

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ghsaPr6xStreamPDF writes a one-page document whose objects 4.. all refer to one stream object with the given filters.
func ghsaPr6xStreamPDF(t *testing.T, copies int, filters ...string) []byte {
	t.Helper()
	sd := types.StreamDict{Dict: types.NewDict(), Content: bytes.Repeat([]byte("q Q\n"), 16<<10)}
	for _, name := range filters {
		sd.FilterPipeline = append(sd.FilterPipeline, types.PDFFilter{Name: name})
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	dict := fmt.Sprintf("<</Length %d>>", len(sd.Raw))
	if len(filters) > 0 {
		dict = fmt.Sprintf("<</Length %d/Filter/%s>>", len(sd.Raw), filters[0])
	}

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	var offs []int
	for i, body := range []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Kids[3 0 R]/Count 1>>",
		"<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]>>",
	} {
		offs = append(offs, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	streamOff := b.Len()
	fmt.Fprintf(&b, "4 0 obj\n%s\nstream\n%s\nendstream\nendobj\n", dict, sd.Raw)
	for range copies {
		offs = append(offs, streamOff)
	}

	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offs)+1)
	for _, off := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(offs)+1, xref)
	return b.Bytes()
}

func ghsaPr6xReadContext(t *testing.T, data []byte, eager bool) *model.Context {
	t.Helper()
	conf := model.NewDefaultConfiguration()
	conf.DecodeAllStreams = eager
	ctx, err := pdfcpu.Read(context.Background(), bytes.NewReader(data), conf)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func ghsaPr6xStream(t *testing.T, ctx *model.Context, objNr int) types.StreamDict {
	t.Helper()
	sd, ok := ctx.Table[objNr].Object.(types.StreamDict)
	if !ok {
		t.Fatalf("object %d: got %T, want stream dict", objNr, ctx.Table[objNr].Object)
	}
	return sd
}

func TestGHSA_pr6x_vqwr_4hj2_EagerReadChargesEachAlias(t *testing.T) {
	data := ghsaPr6xStreamPDF(t, 2, filter.Flate)

	ctx := ghsaPr6xReadContext(t, data, true)
	first, second := ghsaPr6xStream(t, ctx, 4), ghsaPr6xStream(t, ctx, 5)
	if len(first.Content) == 0 || len(second.Content) == 0 || &first.Content[0] == &second.Content[0] {
		t.Fatal("want two separately decoded buffers")
	}
	want := int64(cap(first.Content) + cap(second.Content))
	if got := model.DecodedStreamBudgetUsed(ctx.XRefTable); got != want {
		t.Fatalf("eager read charged %d bytes, want the decoded capacity %d", got, want)
	}

	ctx = ghsaPr6xReadContext(t, data, false)
	if got := model.DecodedStreamBudgetUsed(ctx.XRefTable); got != 0 {
		t.Fatalf("read without eager decoding charged %d bytes", got)
	}
}

func TestGHSA_pr6x_vqwr_4hj2_EagerReadDoesNotChargeRawContent(t *testing.T) {
	ctx := ghsaPr6xReadContext(t, ghsaPr6xStreamPDF(t, 2), true)
	sd := ghsaPr6xStream(t, ctx, 4)
	if len(sd.Content) == 0 || &sd.Content[0] != &sd.Raw[0] {
		t.Fatal("want unfiltered content to share the raw stream data")
	}
	if got := model.DecodedStreamBudgetUsed(ctx.XRefTable); got != 0 {
		t.Fatalf("unfiltered streams charged %d bytes", got)
	}
}

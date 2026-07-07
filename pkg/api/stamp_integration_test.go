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
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const watermarkArtifactMarker = "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC"

func twoPageStampInput(t *testing.T) []byte {
	t.Helper()
	ctx, err := pdfcpu.CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	pagesRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesRef)
	if err != nil {
		t.Fatal(err)
	}
	for pageNr := 1; pageNr <= 2; pageNr++ {
		pageRef, err := ctx.EmptyPage(pagesRef, types.RectForFormat("A4"), 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := model.AppendPageTree(pageRef, pageNr, pagesDict); err != nil {
			t.Fatal(err)
		}
	}
	ctx.PageCount = 2
	var buf bytes.Buffer
	if err := WriteContext(ctx, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pageHasWatermarkArtifact(t *testing.T, ctx *model.Context, pageNr int) bool {
	t.Helper()
	d, _, _, err := ctx.PageDict(pageNr, false)
	if err != nil {
		t.Fatal(err)
	}
	content, err := ctx.PageContent(d, pageNr)
	if errors.Is(err, model.ErrNoContent) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Contains(content, []byte(watermarkArtifactMarker))
}

func TestRemoveWatermarkFromOnePagePreservesOtherPage(t *testing.T) {
	input := twoPageStampInput(t)
	wm := stampTestWatermark(t, false)
	var stamped bytes.Buffer
	if err := AddWatermarks(bytes.NewReader(input), &stamped, []string{"1-2"}, wm, nil); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := RemoveWatermarks(bytes.NewReader(stamped.Bytes()), &output, []string{"1"}, nil); err != nil {
		t.Fatal(err)
	}

	ctx, err := ReadContext(bytes.NewReader(output.Bytes()), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateContext(ctx); err != nil {
		t.Fatal(err)
	}
	if pageHasWatermarkArtifact(t, ctx, 1) {
		t.Fatal("expected page 1 watermark to be removed")
	}
	if !pageHasWatermarkArtifact(t, ctx, 2) {
		t.Fatal("expected page 2 to remain watermarked")
	}
}

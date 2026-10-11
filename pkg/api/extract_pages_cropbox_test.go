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
	"context"
	"io"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type extractionCropBoxCase struct {
	name         string
	rootCrop     *types.Rectangle
	ancestorCrop *types.Rectangle
	pageCrop     *types.Rectangle
	indirect     bool
	want         *types.Rectangle
}

func setExtractionCropBox(t *testing.T, ctx *model.Context, d types.Dict, crop *types.Rectangle, indirect bool) {
	t.Helper()
	if crop == nil {
		return
	}
	var obj types.Object = crop.Array()
	if indirect {
		ref, err := ctx.IndRefForNewObject(obj)
		if err != nil {
			t.Fatal(err)
		}
		obj = *ref
	}
	d["CropBox"] = obj
}

func extractionCropBoxAncestor(t *testing.T, ctx *model.Context, root types.IndirectRef, rootDict types.Dict, crop *types.Rectangle) (types.IndirectRef, types.Dict) {
	t.Helper()
	d := types.Dict{
		"Type":    types.Name("Pages"),
		"Parent":  root,
		"Count":   types.Integer(0),
		"Kids":    types.Array{},
		"CropBox": crop.Array(),
	}
	ref, err := ctx.IndRefForNewObject(d)
	if err != nil {
		t.Fatal(err)
	}
	rootDict["Kids"] = types.Array{*ref}
	return *ref, d
}

func extractionCropBoxPDF(t *testing.T, tc extractionCropBoxCase) []byte {
	t.Helper()
	ctx, err := pdfcpu.CreateContextWithXRefTable(model.NewStatelessConfiguration(), &types.Dim{Width: 600, Height: 800})
	if err != nil {
		t.Fatal(err)
	}
	root, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	rootDict, err := ctx.DereferenceDict(*root)
	if err != nil {
		t.Fatal(err)
	}
	setExtractionCropBox(t, ctx, rootDict, tc.rootCrop, tc.indirect)
	parent, parentDict := *root, rootDict
	if tc.ancestorCrop != nil {
		parent, parentDict = extractionCropBoxAncestor(t, ctx, *root, rootDict, tc.ancestorCrop)
	}
	for range 2 {
		page, err := ctx.EmptyPage(&parent, types.RectForDim(600, 800), 0)
		if err != nil {
			t.Fatal(err)
		}
		d, err := ctx.DereferenceDict(*page)
		if err != nil {
			t.Fatal(err)
		}
		d.Delete("MediaBox")
		setExtractionCropBox(t, ctx, d, tc.pageCrop, tc.indirect)
		if err := model.AppendPageTree(page, 1, parentDict); err != nil {
			t.Fatal(err)
		}
	}
	rootDict["Count"] = types.Integer(2)
	ctx.PageCount = 2
	var out bytes.Buffer
	if err := WriteContext(t.Context(), ctx, &out); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func assertExtractionCropBox(t *testing.T, data []byte, pageCount int, want *types.Rectangle) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.Offline = true
	ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(data), conf)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.PageCount != pageCount {
		t.Fatalf("page count = %d, want %d", ctx.PageCount, pageCount)
	}
	for pageNr := 1; pageNr <= pageCount; pageNr++ {
		d, _, attrs, err := ctx.PageDict(t.Context(), pageNr, false)
		if err != nil {
			t.Fatal(err)
		}
		if attrs.MediaBox == nil || !attrs.MediaBox.Equals(*types.RectForDim(600, 800)) {
			t.Errorf("page %d: MediaBox = %v, want [0 0 600 800]", pageNr, attrs.MediaBox)
		}
		if want == nil {
			if attrs.CropBox != nil {
				t.Errorf("page %d: CropBox = %v, want MediaBox default", pageNr, attrs.CropBox)
			}
			continue
		}
		if attrs.CropBox == nil || !attrs.CropBox.Equals(*want) {
			t.Errorf("page %d: CropBox = %v, want %v", pageNr, attrs.CropBox, want)
		}
		if _, found := d.Find("CropBox"); !found {
			t.Errorf("page %d: missing explicit CropBox after extraction", pageNr)
		}
	}
}

func testSplitCropBox(t *testing.T, input []byte, conf *model.Configuration, want *types.Rectangle) {
	t.Helper()
	spans, err := SplitRaw(t.Context(), bytes.NewReader(input), 1, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("span count = %d, want 2", len(spans))
	}
	for _, span := range spans {
		data, err := io.ReadAll(span.Reader)
		if err != nil {
			t.Fatal(err)
		}
		assertExtractionCropBox(t, data, 1, want)
	}
}

func testExtractCropBox(t *testing.T, input []byte, conf *model.Configuration, usePgCache bool, want *types.Rectangle) {
	t.Helper()
	src, err := ReadAndValidate(t.Context(), bytes.NewReader(input), conf)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := pdfcpu.ExtractPages(t.Context(), src, []int{1, 1}, usePgCache)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := WriteContext(t.Context(), dst, &out); err != nil {
		t.Fatal(err)
	}
	assertExtractionCropBox(t, out.Bytes(), 2, want)
}

func testPageOperationCropBoxes(t *testing.T, input []byte, conf *model.Configuration, want *types.Rectangle) {
	t.Helper()
	operations := []struct {
		name      string
		run       func(context.Context, io.ReadSeeker, io.Writer, []string, *model.Configuration) error
		selection []string
		pageCount int
	}{
		{"trim", Trim, []string{"1"}, 1},
		{"collect", Collect, []string{"1", "1"}, 2},
		{"remove", RemovePages, []string{"2"}, 1},
	}
	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := op.run(t.Context(), bytes.NewReader(input), &out, op.selection, conf); err != nil {
				t.Fatal(err)
			}
			assertExtractionCropBox(t, out.Bytes(), op.pageCount, want)
		})
	}
}

// TestPageExtractionPreservesCropBox verifies page operations preserve effective cropping when reparenting pages.
func TestPageExtractionPreservesCropBox(t *testing.T) {
	inherited := types.NewRectangle(50, 50, 550, 750)
	ancestor := types.NewRectangle(70, 70, 530, 730)
	local := types.NewRectangle(80, 80, 520, 720)
	cases := []extractionCropBoxCase{
		{name: "root_inheritance", rootCrop: inherited, want: inherited},
		{name: "nearest_ancestor", rootCrop: inherited, ancestorCrop: ancestor, want: ancestor},
		{name: "local_override", rootCrop: inherited, pageCrop: local, want: local},
		{name: "inherited_indirect", rootCrop: inherited, indirect: true, want: inherited},
		{name: "local_indirect", pageCrop: local, indirect: true, want: local},
		{name: "media_box_default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := extractionCropBoxPDF(t, tc)
			conf := model.NewStatelessConfiguration()
			conf.Offline = true
			t.Run("split", func(t *testing.T) {
				testSplitCropBox(t, input, conf, tc.want)
			})
			testPageOperationCropBoxes(t, input, conf, tc.want)
			t.Run("extract", func(t *testing.T) {
				testExtractCropBox(t, input, conf, false, tc.want)
			})
			t.Run("extract_cached", func(t *testing.T) {
				testExtractCropBox(t, input, conf, true, tc.want)
			})
		})
	}
}

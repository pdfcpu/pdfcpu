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

package pdfcpu

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func testOptimizeContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	ctx.Optimize = &model.OptimizationContext{
		FontObjects:          map[int]*model.FontObject{},
		FormFontObjects:      map[int]*model.FontObject{},
		Fonts:                map[string][]int{},
		DuplicateFonts:       map[int]types.Dict{},
		DuplicateFontObjs:    types.IntSet{},
		ImageObjects:         map[int]*model.ImageObject{},
		DuplicateImages:      map[int]*model.DuplicateImageObject{},
		DuplicateImageObjs:   types.IntSet{},
		DuplicateInfoObjects: types.IntSet{},
		ContentStreamCache:   map[int]*types.StreamDict{},
		FormStreamCache:      map[int]*types.StreamDict{},
		Cache:                map[int]bool{},
	}
	return ctx
}

func addOptimizeTestPage(t *testing.T, ctx *model.Context) types.Dict {
	t.Helper()

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pageIndRef, err := ctx.EmptyPage(pagesIndRef, types.RectForFormat("A4"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
		t.Fatal(err)
	}
	ctx.PageCount = 1

	pageDict, _, _, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatal(err)
	}
	return pageDict
}

func TestOptimizeResourceDictsErrorIncludesPageContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.PageCount = 1
	ctx.RootDict["Pages"] = *types.NewIndirectRef(999, 0)

	err := optimizeResourceDicts(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1: resource dict") {
		t.Fatalf("expected page resource context, got %q", err.Error())
	}
}

func TestEnsureDirectWidthForXObjsErrorIncludesImageContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.Optimize.PageImages = []types.IntSet{{7: true}}
	ctx.Optimize.ImageObjects[7] = &model.ImageObject{
		ImageDict: &types.StreamDict{
			Dict: types.Dict{
				"Width": *types.NewIndirectRef(999, 0),
			},
		},
	}

	err := ensureDirectWidthForXObjs(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "image obj#7: resolve width") {
		t.Fatalf("expected image width context, got %q", err.Error())
	}
}

func TestOptimizeXRefTableResourceErrorIncludesDeepPageContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.Conf.OptimizeResourceDicts = false
	pageDict := addOptimizeTestPage(t, ctx)
	sd, err := ctx.NewStreamDictForBuf(nil)
	if err != nil {
		t.Fatal(err)
	}
	sd.InsertName("Subtype", "Pattern")
	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Resources"] = types.Dict{
		"XObject": types.Dict{
			"Im1": *ir,
		},
	}

	err = OptimizeXRefTable(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		"optimize fonts and images",
		"page tree",
		"page 1 obj#",
		"optimize resources",
		"XObject",
		"XObject resource Im1 obj#",
		"dereference XObject",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestOptimizeXRefTableContentErrorIncludesDeepPageContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.Conf.OptimizeResourceDicts = false
	ctx.OptimizeDuplicateContentStreams = true
	pageDict := addOptimizeTestPage(t, ctx)
	pageDict["Contents"] = types.Name("broken")

	err := OptimizeXRefTable(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{
		"optimize fonts and images",
		"page tree",
		"page 1 obj#",
		"optimize content",
		"remove empty content streams",
		"corrupt page content array",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestCacheFormFontsErrorIncludesFormFontObjectContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	objNr, err := ctx.InsertObject(types.Dict{
		"Type":    types.Name("Font"),
		"Subtype": types.Name("Type1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx.Form = types.Dict{
		"DR": types.Dict{
			"Font": types.Dict{
				"F1": *types.NewIndirectRef(objNr, 0),
			},
		},
	}

	err = CacheFormFonts(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	want := fmt.Sprintf("form font obj#%d: parse font name", objNr)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected form font context, got %q", err.Error())
	}
}

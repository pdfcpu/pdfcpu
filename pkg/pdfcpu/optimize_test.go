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
		FormResourceCache:    map[int]types.IntSet{},
		Cache:                map[int]bool{},
	}
	return ctx
}

func TestFormResourcesVisitedPerPage(t *testing.T) {
	ctx := testOptimizeContext(t)
	if formResourcesVisited(ctx, 0, 7) {
		t.Fatal("form resources unexpectedly visited")
	}
	if !formResourcesVisited(ctx, 0, 7) {
		t.Fatal("form resources not cached for page")
	}
	if formResourcesVisited(ctx, 1, 7) {
		t.Fatal("form resources cache leaked across pages")
	}
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

func TestOptimizeResourceDictsConsolidatesInheritedResources(t *testing.T) {
	ctx := testOptimizeContext(t)
	pagesIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	pagesDict["Resources"] = types.Dict{
		"Font": types.Dict{"F0": types.Name("root")},
	}

	pageDicts := make([]types.Dict, 2)
	for i := range pageDicts {
		pageIndRef, err := ctx.EmptyPage(pagesIndRef, types.RectForFormat("A4"), 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := model.AppendPageTree(pageIndRef, 1, pagesDict); err != nil {
			t.Fatal(err)
		}
		pageDicts[i], err = ctx.DereferenceDict(*pageIndRef)
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx.PageCount = len(pageDicts)
	pageDicts[0]["Resources"] = types.Dict{
		"XObject": types.Dict{"X0": types.Name("pageOne")},
	}

	if err := optimizeResourceDicts(ctx); err != nil {
		t.Fatal(err)
	}

	pageOneResources := pageDicts[0].DictEntry("Resources")
	if pageOneResources == nil || pageOneResources.DictEntry("Font") == nil || pageOneResources.DictEntry("XObject") == nil {
		t.Fatalf("page one resources not consolidated: %v", pageOneResources)
	}
	pageTwoResources := pageDicts[1].DictEntry("Resources")
	if pageTwoResources == nil || pageTwoResources.DictEntry("Font") == nil {
		t.Fatalf("page two inherited resources missing: %v", pageTwoResources)
	}
	if pageTwoResources.DictEntry("XObject") != nil {
		t.Fatalf("page one resources leaked into page two: %v", pageTwoResources)
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

func TestHandleDuplicateImageObjectUsesStreamHashes(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.Optimize.PageImages = []types.IntSet{{}}

	register := func(objNr int, raw []byte) {
		t.Helper()
		sd := &types.StreamDict{
			Dict: types.Dict{
				"Length": types.Integer(len(raw)),
			},
			Raw: raw,
		}
		originalObjNr, alreadyDuplicate, err := handleDuplicateImageObject(ctx, sd, "Im", objNr, 0)
		if err != nil {
			t.Fatal(err)
		}
		if originalObjNr != nil || alreadyDuplicate {
			t.Fatalf("image obj#%d unexpectedly classified as duplicate", objNr)
		}
		ctx.Optimize.ImageObjects[objNr] = &model.ImageObject{
			ResourceNames: map[int]string{0: "Im"},
			ImageDict:     sd,
		}
	}

	register(1, []byte("first"))
	register(2, []byte("other"))

	duplicate := &types.StreamDict{
		Dict: types.Dict{
			"Length": types.Integer(5),
		},
		Raw: []byte("first"),
	}
	originalObjNr, alreadyDuplicate, err := handleDuplicateImageObject(ctx, duplicate, "Im3", 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if originalObjNr == nil || *originalObjNr != 1 || alreadyDuplicate {
		t.Fatalf("duplicate image classified as original: objNr=%v alreadyDuplicate=%t", originalObjNr, alreadyDuplicate)
	}
	if len(ctx.Optimize.ImageObjectHashes) != 2 {
		t.Fatalf("expected two image stream hash buckets, got %d", len(ctx.Optimize.ImageObjectHashes))
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

func TestOptimizeXRefTableSkipsUnclassifiableXObject(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.Conf.OptimizeResourceDicts = false
	pageDict := addOptimizeTestPage(t, ctx)
	sd, err := ctx.NewStreamDictForBuf([]byte("BT (Hello World!) Tj ET"))
	if err != nil {
		t.Fatal(err)
	}
	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		t.Fatal(err)
	}
	pageDict["Resources"] = types.Dict{
		"XObject": types.Dict{
			"Im0": *ir,
		},
	}

	if err := OptimizeXRefTable(ctx); err != nil {
		t.Fatal(err)
	}
	if subtype := sd.Subtype(); subtype != nil {
		t.Fatalf("unexpected inferred Subtype %q", *subtype)
	}
	if len(ctx.Optimize.PageImages) != 1 || len(ctx.Optimize.PageImages[0]) != 0 {
		t.Fatalf("unclassifiable XObject registered as image: %v", ctx.Optimize.PageImages)
	}
}

func TestOptimizeSMaskResourcesSkipsUnclassifiableXObject(t *testing.T) {
	ctx := testOptimizeContext(t)
	sd, err := ctx.NewStreamDictForBuf(nil)
	if err != nil {
		t.Fatal(err)
	}
	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		t.Fatal(err)
	}
	sMask := types.Dict{
		"G": *ir,
	}

	if err := optimizeSMaskResources(sMask, nil, "", ctx, types.Dict{}, 0, types.IntSet{}, 1); err != nil {
		t.Fatal(err)
	}
	if len(ctx.Optimize.ImageObjects) != 0 {
		t.Fatalf("unclassifiable SMask XObject registered as image: %v", ctx.Optimize.ImageObjects)
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

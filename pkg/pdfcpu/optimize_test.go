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
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func optimizeTestStream(dict types.Dict, raw string) *types.StreamDict {
	d := types.Dict{"Length": types.Integer(len(raw))}
	maps.Copy(d, dict)
	length := int64(len(raw))
	return &types.StreamDict{Dict: d, Raw: []byte(raw), StreamLength: &length}
}

// TestContentStreamDeduplicationChecksDictionary verifies bytes and their decoding parameters both match.
func TestContentStreamDeduplicationChecksDictionary(t *testing.T) {
	filter := types.Dict{"Filter": types.Name("ASCIIHexDecode")}
	predictor := func(value types.Object) types.Dict {
		return types.Dict{
			"Filter": types.Array{types.Name("FlateDecode")}, "DecodeParms": types.Array{value},
		}
	}
	nullParams := predictor(nil)
	customParams := predictor(types.Dict{"Predictor": types.Integer(2)})
	tests := []struct {
		name          string
		first, second types.Dict
		raw           string
		duplicate     bool
	}{
		{"equal", nil, nil, "00>", true},
		{"different bytes", nil, nil, "01>", false},
		{"different filter", nil, filter, "00>", false},
		{"equal filter", filter, filter, "00>", true},
		{"different parameters", predictor(types.Dict{"Predictor": types.Integer(1)}), customParams, "00>", false},
		{"equal parameters", customParams, customParams, "00>", true},
		{"null versus parameters", customParams, nullParams, "00>", false},
		{"parameters versus null", nullParams, customParams, "00>", false},
		{"equal null parameters", nullParams, nullParams, "00>", true},
		{"different metadata", types.Dict{"Test": types.Integer(1)}, types.Dict{"Test": types.Integer(2)}, "00>", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testOptimizeContext(t)
			ctx.Optimize.ContentStreamCache[100] = optimizeTestStream(tc.first, "00>")
			ir, err := optimizeContentStreamUsage(t.Context(), ctx, optimizeTestStream(tc.second, tc.raw), 101)
			if err != nil {
				t.Fatal(err)
			}
			if (ir != nil) != tc.duplicate {
				t.Fatalf("duplicate reference = %v, want duplicate: %t", ir, tc.duplicate)
			}
		})
	}
}

func dedupGenerationReference(t *testing.T, ctx *model.Context, kind string, sd *types.StreamDict) (*types.IndirectRef, error) {
	t.Helper()
	resource := types.Dict{"R": *types.NewIndirectRef(101, 9)}
	var err error
	switch kind {
	case "content":
		ctx.Optimize.ContentStreamCache[100] = sd
		return optimizeContentStreamUsage(t.Context(), ctx, sd, 101)
	case "form":
		ctx.Optimize.FormStreamCache[100] = sd
		return optimizeXObjectForm(t.Context(), ctx, sd, 101)
	case "font":
		font := types.Dict{"Type": types.Name("Font"), "Subtype": types.Name("Type1"), "BaseFont": types.Name("Helvetica")}
		ctx.Table[100].Object, ctx.Table[101].Object = font, font.Clone()
		ctx.Optimize.Fonts["Helvetica"] = []int{100}
		ctx.Optimize.FontObjects[100] = &model.FontObject{FontDict: font}
		ctx.Optimize.PageFonts = []types.IntSet{{}}
		err = optimizeFontResourcesDict(t.Context(), ctx, resource, 0, "")
	case "image":
		ctx.Optimize.PageImages = []types.IntSet{{}}
		ctx.Optimize.ImageObjects[100] = &model.ImageObject{ImageDict: sd, ResourceNames: map[int]string{}}
		err = optimizeXObjectImage(t.Context(), ctx, sd, "", "R", resource, 101, 0, 3, ctx.Optimize.PageImages[0])
	case "font repair":
		resource["R"] = nil
		ctx.Optimize.Fonts["Arial"] = []int{100}
		ctx.Optimize.CorruptFontResDicts = []types.Dict{resource}
		err = fixCorruptFontResDicts(ctx)
	default:
		t.Fatalf("unknown resource kind: %s", kind)
	}
	return resource.IndirectRefEntry("R"), err
}

// TestDeduplicationPreservesGeneration covers all replacement-reference sites, including corrupt-font repair.
func TestDeduplicationPreservesGeneration(t *testing.T) {
	for _, generation := range []int{0, 5} {
		for _, kind := range []string{"content", "form", "font", "image", "font repair"} {
			t.Run(fmt.Sprintf("%s/generation%d", kind, generation), func(t *testing.T) {
				ctx := testOptimizeContext(t)
				sd := optimizeTestStream(nil, "q Q")
				ctx.Table[100] = model.NewXRefTableEntryGen0(*sd)
				ctx.Table[100].Generation = &generation
				ctx.Table[100].RefCount = 1
				ctx.Table[101] = model.NewXRefTableEntryGen0(*sd)
				duplicateGeneration := 9
				ctx.Table[101].Generation = &duplicateGeneration
				ir, err := dedupGenerationReference(t, ctx, kind, sd)
				if err != nil {
					t.Fatal(err)
				}
				want := types.NewIndirectRef(100, generation)
				if ir == nil || *ir != *want {
					t.Fatalf("reference = %v, want %v", ir, want)
				}
				if kind != "font repair" && ctx.Table[100].RefCount != 2 {
					t.Fatalf("retained reference count = %d, want 2", ctx.Table[100].RefCount)
				}
			})
		}
	}
}

// TestContentStreamComparisonPreservesErrors checks cancellation and dictionary comparison errors reach the caller.
func TestContentStreamComparisonPreservesErrors(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprintf("canceled=%t", canceled), func(t *testing.T) {
			ctx := testOptimizeContext(t)
			// Type is an invalid integer so EqualObjects returns its dictionary error.
			sd := optimizeTestStream(types.Dict{"Type": types.Integer(1)}, "q Q")
			ctx.Optimize.ContentStreamCache[100] = sd
			c, cancel := context.WithCancel(t.Context())
			defer cancel()
			if canceled {
				cancel()
			}
			_, err := optimizeContentStreamUsage(c, ctx, sd, 101)
			if err == nil || !strings.Contains(err.Error(), "compare content stream obj#101 with obj#100") {
				t.Fatalf("missing comparison context: %v", err)
			}
			if canceled && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation cause lost: %v", err)
			}
		})
	}
}

func TestOptimizeXRefTableRejectsNilContext(t *testing.T) {
	if err := OptimizeXRefTable(nil, nil); !errors.Is(err, model.ErrMissingContext) {
		t.Fatalf("got %v, want model.ErrMissingContext", err)
	}
}

func TestOptimizeXRefTableReturnsCancellation(t *testing.T) {
	c, cancel := context.WithCancel(t.Context())
	cancel()

	if err := OptimizeXRefTable(c, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

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

	pageDict, _, _, err := ctx.PageDict(t.Context(), 1, false)
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
		originalObjNr, alreadyDuplicate, err := handleDuplicateImageObject(t.Context(), ctx, sd, "Im", objNr, 0)
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
	originalObjNr, alreadyDuplicate, err := handleDuplicateImageObject(t.Context(), ctx, duplicate, "Im3", 3, 0)
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

	err = OptimizeXRefTable(t.Context(), ctx)
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

	if err := OptimizeXRefTable(t.Context(), ctx); err != nil {
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

	if err := optimizeSMaskResources(t.Context(), sMask, nil, "", ctx, types.Dict{}, 0, types.IntSet{}, 1); err != nil {
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

	err := OptimizeXRefTable(t.Context(), ctx)
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

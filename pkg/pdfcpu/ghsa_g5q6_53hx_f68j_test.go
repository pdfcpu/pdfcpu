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

// GHSA-g5q6-53hx-f68j prevents attacker-controlled poster page dimensions from creating unbounded tile grids.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-g5q6-53hx-f68j

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestGHSA_g5q6_53hx_f68j_PosterTileLimit(t *testing.T) {
	dim := &types.Dim{Width: 1, Height: 1}
	hor, vert, err := createPosterCuts(t.Context(), types.RectForWidthAndHeight(0, 0, 100, 10), dim)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(hor) * len(vert); got != maxPosterTiles {
		t.Fatalf("got %d poster tiles, want %d", got, maxPosterTiles)
	}

	_, _, err = createPosterCuts(t.Context(), types.RectForWidthAndHeight(0, 0, 101, 10), dim)
	if err == nil || !strings.Contains(err.Error(), "poster tile count exceeds limit") {
		t.Fatalf("expected poster tile limit error, got %v", err)
	}

	_, _, err = createPosterCuts(t.Context(), types.RectForWidthAndHeight(0, 0, 1_000_000, 1), dim)
	if err == nil || !strings.Contains(err.Error(), "poster tile count exceeds limit") {
		t.Fatalf("expected poster axis limit error, got %v", err)
	}
}

func TestGHSA_g5q6_53hx_f68j_PosterRejectsInvalidDimensions(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height float64
		tile          types.Dim
	}{
		// A zero tile size used to append cut points forever.
		{"zero tile", 100, 100, types.Dim{Width: 0, Height: 1}},
		{"negative tile", 100, 100, types.Dim{Width: 1, Height: -1}},
		{"NaN tile", 100, 100, types.Dim{Width: math.NaN(), Height: 1}},
		{"Inf tile", 100, 100, types.Dim{Width: 1, Height: math.Inf(1)}},
		{"zero box", 0, 100, types.Dim{Width: 1, Height: 1}},
		{"NaN box", 100, math.NaN(), types.Dim{Width: 1, Height: 1}},
		{"Inf box", math.Inf(1), 100, types.Dim{Width: 1, Height: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			box := types.RectForWidthAndHeight(0, 0, tc.width, tc.height)
			_, _, err := createPosterCuts(t.Context(), box, &tc.tile)
			if err == nil || !strings.Contains(err.Error(), "poster dimensions must be finite and > 0") {
				t.Fatalf("got %v, want invalid poster dimensions error", err)
			}
		})
	}
}

func TestGHSA_g5q6_53hx_f68j_PosterPageRejectsBeforeMutation(t *testing.T) {
	ctx, d := cutBlankPageContext(t)
	mediaBox := d["MediaBox"].String()
	cropBox := d["CropBox"].String()
	rotate := d["Rotate"].String()

	_, err := PosterPage(t.Context(), ctx, 1, &model.Cut{
		Scale: 1, PageDim: &types.Dim{Width: 1, Height: 1}, UserDim: true,
	})
	if err == nil || !strings.Contains(err.Error(), "poster tile count exceeds limit") {
		t.Fatalf("expected poster tile limit error, got %v", err)
	}
	if got := d["MediaBox"].String(); got != mediaBox {
		t.Fatalf("source MediaBox changed: got %s, want %s", got, mediaBox)
	}
	if got := d["CropBox"].String(); got != cropBox {
		t.Fatalf("source CropBox changed: got %s, want %s", got, cropBox)
	}
	if got := d["Rotate"].String(); got != rotate {
		t.Fatalf("source Rotate changed: got %s, want %s", got, rotate)
	}
}

const ghsaG5q6Content = "0 0 m 400 500 l S"

// ghsaG5q6EntryPoints cut a page into margin tiles through each public tiling entry point.
var ghsaG5q6EntryPoints = []struct {
	name string
	run  func(context.Context, *model.Context) (*model.Context, error)
}{
	{"poster", func(c context.Context, ctx *model.Context) (*model.Context, error) {
		return PosterPage(c, ctx, 1, &model.Cut{
			Scale: 1, PageDim: &types.Dim{Width: 100, Height: 100}, UserDim: true, Margin: 5,
		})
	}},
	{"cut", func(c context.Context, ctx *model.Context) (*model.Context, error) {
		return CutPage(c, ctx, 1, &model.Cut{Hor: []float64{0, .5}, Vert: []float64{0, .5}, Margin: 5})
	}},
	{"ndown", func(c context.Context, ctx *model.Context) (*model.Context, error) {
		return NDownPage(c, ctx, 1, 4, &model.Cut{Margin: 5})
	}},
}

// ghsaG5q6MarginTiles runs an entry point on a page with content and resources and returns the tile pages.
func ghsaG5q6MarginTiles(t *testing.T, run func(context.Context, *model.Context) (*model.Context, error)) (*model.Context, []types.Dict) {
	t.Helper()
	ctx, d := cutBlankPageContext(t)
	sd, err := ctx.NewStreamDictForBuf([]byte(ghsaG5q6Content))
	if err != nil {
		t.Fatal(err)
	}
	if err := sd.Encode(); err != nil {
		t.Fatal(err)
	}
	indRef, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		t.Fatal(err)
	}
	d["Contents"] = *indRef
	d["Resources"] = types.Dict{"ProcSet": types.Array{types.Name("PDF")}}

	ctxDest, err := run(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	pagesIndRef, err := ctxDest.Pages()
	if err != nil {
		t.Fatal(err)
	}
	pagesDict, err := ctxDest.DereferenceDict(*pagesIndRef)
	if err != nil {
		t.Fatal(err)
	}
	// The first page is the cut outline.
	var tiles []types.Dict
	for _, kid := range pagesDict.ArrayEntry("Kids")[1:] {
		td, err := ctxDest.DereferenceDict(kid)
		if err != nil {
			t.Fatal(err)
		}
		tiles = append(tiles, td)
	}
	if len(tiles) < 2 {
		t.Fatalf("got %d tiles, want several", len(tiles))
	}
	return ctxDest, tiles
}

func TestGHSA_g5q6_53hx_f68j_MarginTilesShareContent(t *testing.T) {
	for _, ep := range ghsaG5q6EntryPoints {
		t.Run(ep.name, func(t *testing.T) {
			ctxDest, tiles := ghsaG5q6MarginTiles(t, ep.run)
			shared := map[int]bool{}
			for i, td := range tiles {
				contents := td.ArrayEntry("Contents")
				if len(contents) != 3 {
					t.Fatalf("tile %d: got Contents %v, want transform, content and margin", i+1, td["Contents"])
				}
				indRef, ok := contents[1].(types.IndirectRef)
				if !ok {
					t.Fatalf("tile %d: page content is not shared by reference: %v", i+1, contents[1])
				}
				shared[indRef.ObjectNumber.Value()] = true

				bb, err := ctxDest.PageContent(td, i+2)
				if err != nil {
					t.Fatal(err)
				}
				if got := strings.Count(string(bb), ghsaG5q6Content); got != 1 {
					t.Fatalf("tile %d: page content occurs %d times, want 1", i+1, got)
				}
			}
			if len(shared) != 1 {
				t.Fatalf("tiles reference %d page content streams, want 1", len(shared))
			}
		})
	}
}

func TestGHSA_g5q6_53hx_f68j_TilesShareResources(t *testing.T) {
	for _, ep := range ghsaG5q6EntryPoints {
		t.Run(ep.name, func(t *testing.T) {
			ctxDest, tiles := ghsaG5q6MarginTiles(t, ep.run)
			shared := map[int]bool{}
			for i, td := range tiles {
				indRef, ok := td["Resources"].(types.IndirectRef)
				if !ok {
					t.Fatalf("tile %d: resources are not shared by reference: %v", i+1, td["Resources"])
				}
				shared[indRef.ObjectNumber.Value()] = true

				resources, err := ctxDest.DereferenceDict(indRef)
				if err != nil {
					t.Fatal(err)
				}
				if resources.ArrayEntry("ProcSet") == nil {
					t.Fatalf("tile %d: resources lost ProcSet: %v", i+1, resources)
				}
			}
			if len(shared) != 1 {
				t.Fatalf("tiles reference %d resource dictionaries, want 1", len(shared))
			}
		})
	}
}

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
	"reflect"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func cutBlankPageContext(t *testing.T) (*model.Context, types.Dict) {
	t.Helper()
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
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

	d, _, _, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatal(err)
	}
	d.Delete("Contents")
	d["Annots"] = types.Array{}
	d["CropBox"] = types.RectForWidthAndHeight(10, 20, 400, 500).Array()
	d["Rotate"] = types.Integer(90)
	return ctx, d
}

// TestGeneratedCutsPreserveConfiguration verifies poster and n-down use operation-local cut points.
func TestGeneratedCutsPreserveConfiguration(t *testing.T) {
	tests := []struct {
		name string
		cut  *model.Cut
		run  func(*model.Context, *model.Cut) error
	}{
		{name: "ndown", cut: &model.Cut{Hor: []float64{0.75}, Vert: []float64{0.25}}, run: func(ctx *model.Context, cut *model.Cut) error {
			_, err := NDownPage(ctx, 1, 2, cut)
			return err
		}},
		{name: "poster", cut: &model.Cut{
			Hor: []float64{0.75}, Vert: []float64{0.25}, Scale: 1,
			PageDim: &types.Dim{Width: 100, Height: 100}, UserDim: true,
		}, run: func(ctx *model.Context, cut *model.Cut) error {
			_, err := PosterPage(ctx, 1, cut)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := cutBlankPageContext(t)
			wantHor := append([]float64(nil), tt.cut.Hor...)
			wantVert := append([]float64(nil), tt.cut.Vert...)
			if err := tt.run(ctx, tt.cut); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.cut.Hor, wantHor) || !reflect.DeepEqual(tt.cut.Vert, wantVert) {
				t.Fatalf("cut points changed: got (%v, %v), want (%v, %v)", tt.cut.Hor, tt.cut.Vert, wantHor, wantVert)
			}
		})
	}
}

// TestCutMarginValidationUsesResolvedTileDimensions verifies per-tile validation and location context.
func TestCutMarginValidationUsesResolvedTileDimensions(t *testing.T) {
	ctx, _ := cutBlankPageContext(t)
	cut := &model.Cut{Hor: []float64{0}, Vert: []float64{0, 0.8}, Margin: 60}
	_, err := CutPage(ctx, 1, cut)
	if err == nil {
		t.Fatal("expected margin error")
	}
	for _, want := range []string{"page 1", "row 1", "column 2", "tile dimensions"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q context, got %q", want, err.Error())
		}
	}
}

// TestCutPageOperationsAcceptBlankPagesAndPreserveSource verifies blank-page and source-mutation regressions.
func TestCutPageOperationsAcceptBlankPagesAndPreserveSource(t *testing.T) {
	tests := []struct {
		name string
		run  func(*model.Context) (*model.Context, error)
	}{
		{name: "cut", run: func(ctx *model.Context) (*model.Context, error) {
			return CutPage(ctx, 1, &model.Cut{Hor: []float64{0, 0.5}, Vert: []float64{0}})
		}},
		{name: "ndown", run: func(ctx *model.Context) (*model.Context, error) {
			return NDownPage(ctx, 1, 2, &model.Cut{})
		}},
		{name: "poster", run: func(ctx *model.Context) (*model.Context, error) {
			return PosterPage(ctx, 1, &model.Cut{Scale: 1, PageDim: &types.Dim{Width: 100, Height: 100}, UserDim: true})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, d := cutBlankPageContext(t)
			cropBox := d["CropBox"].String()
			rotate := d["Rotate"].String()

			ctxDest, err := tt.run(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if ctxDest == nil {
				t.Fatal("expected destination context")
			}
			if _, found := d.Find("Contents"); found {
				t.Fatal("source page content was mutated")
			}
			if _, found := d.Find("Annots"); !found {
				t.Fatal("source page annotations were removed")
			}
			if got := d["CropBox"].String(); got != cropBox {
				t.Fatalf("source CropBox changed: got %s, want %s", got, cropBox)
			}
			if got := d["Rotate"].String(); got != rotate {
				t.Fatalf("source Rotate changed: got %s, want %s", got, rotate)
			}
		})
	}
}

// TestCutPageOperationsIncludePreparationContext verifies exported operation-boundary context.
func TestCutPageOperationsIncludePreparationContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(*model.Context) error
	}{
		{name: "cut", run: func(ctx *model.Context) error {
			_, err := CutPage(ctx, 1, &model.Cut{Hor: []float64{0, 0.5}, Vert: []float64{0}})
			return err
		}},
		{name: "ndown", run: func(ctx *model.Context) error {
			_, err := NDownPage(ctx, 1, 2, &model.Cut{})
			return err
		}},
		{name: "poster", run: func(ctx *model.Context) error {
			_, err := PosterPage(ctx, 1, &model.Cut{Scale: 1, PageDim: &types.Dim{Width: 100, Height: 100}, UserDim: true})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := cutBlankPageContext(t)
			ctx.RootDict["Pages"] = *types.NewIndirectRef(999, 0)
			err := tt.run(ctx)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "prepare page: source page dictionary") {
				t.Fatalf("expected preparation context, got %q", err.Error())
			}
		})
	}
}

// TestCutPageContentErrorIncludesTransformContext verifies deep content-stream context.
func TestCutPageContentErrorIncludesTransformContext(t *testing.T) {
	ctx, d := cutBlankPageContext(t)
	d["Contents"] = types.Integer(7)

	_, err := CutPage(ctx, 1, &model.Cut{Hor: []float64{0, 0.5}, Vert: []float64{0}})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"transform page content", "read page content"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q context, got %q", want, err.Error())
		}
	}
}

// TestCreateNDownCutsListsAllSupportedValues verifies the n-down error contract.
func TestCreateNDownCutsListsAllSupportedValues(t *testing.T) {
	_, _, err := createNDownCuts(5, types.RectForFormat("A4"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "2, 3, 4, 6, 8, 9, 12, 16") {
		t.Fatalf("expected complete supported-value list, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "16s") {
		t.Fatalf("unexpected stale n-down suffix in %q", err.Error())
	}
}

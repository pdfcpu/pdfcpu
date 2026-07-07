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
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestResizePageNumbers verifies sorted selected-page expansion.
func TestResizePageNumbers(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		selected types.IntSet
		want     []int
	}{
		{name: "all pages", count: 3, want: []int{1, 2, 3}},
		{name: "selected pages", count: 3, selected: types.IntSet{3: true, 2: false, 1: true}, want: []int{1, 3}},
		{name: "none selected", count: 3, selected: types.IntSet{1: false}, want: []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resizePageNumbers(tt.count, tt.selected); !slices.Equal(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func resizeTestRectangle(t *testing.T, ctx *model.Context, o types.Object) *types.Rectangle {
	t.Helper()
	a, err := ctx.DereferenceArray(o)
	if err != nil {
		t.Fatal(err)
	}
	r, err := ctx.RectForArray(a)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestResizeBlankPageUpdatesGeometryAndAnnotations verifies blank pages are fully resized.
func TestResizeBlankPageUpdatesGeometryAndAnnotations(t *testing.T) {
	ctx := annotationTestContext(t)
	d, _, attrs, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatal(err)
	}
	wantWidth := attrs.MediaBox.Width() / 2
	wantHeight := attrs.MediaBox.Height() / 2
	d.Delete("Contents")
	d["CropBox"] = attrs.MediaBox.Array()
	d["Rotate"] = types.Integer(0)
	ann := types.Dict{"Rect": types.Array{types.Integer(10), types.Integer(20), types.Integer(30), types.Integer(40)}}
	d["Annots"] = types.Array{ann}

	if err := Resize(ctx, types.IntSet{1: true}, &model.Resize{Scale: 0.5}); err != nil {
		t.Fatal(err)
	}
	mediaBox := resizeTestRectangle(t, ctx, d["MediaBox"])
	if math.Abs(mediaBox.Width()-wantWidth) > 0.001 || math.Abs(mediaBox.Height()-wantHeight) > 0.001 {
		t.Fatalf("expected resized blank page %.3fx%.3f, got %.3fx%.3f", wantWidth, wantHeight, mediaBox.Width(), mediaBox.Height())
	}
	r := resizeTestRectangle(t, ctx, ann["Rect"])
	if r.LL.X != 5 || r.LL.Y != 10 || r.UR.X != 15 || r.UR.Y != 20 {
		t.Fatalf("expected transformed blank-page annotation, got %s", r)
	}
	if _, found := d.Find("CropBox"); found {
		t.Fatal("expected blank-page CropBox removal")
	}
	if _, found := d.Find("Rotate"); found {
		t.Fatal("expected blank-page Rotate removal")
	}
}

// TestParseResizeConfigPreservesExplicitOrientation verifies explicit P/L suffixes enforce orientation.
func TestParseResizeConfigPreservesExplicitOrientation(t *testing.T) {
	for _, form := range []string{"A4P", "LedgerL"} {
		res, err := ParseResizeConfig("form:"+form, types.POINTS)
		if err != nil {
			t.Fatal(err)
		}
		if !res.EnforceOrientation() {
			t.Fatalf("expected form:%s to enforce orientation", form)
		}
	}
	res, err := ParseResizeConfig("form:A4", types.POINTS)
	if err != nil {
		t.Fatal(err)
	}
	if res.EnforceOrientation() {
		t.Fatal("expected unsuffixed form:A4 to preserve source orientation")
	}
}

// TestResizeProcessesSelectedPagesInOrder verifies deterministic page failure ordering.
func TestResizeProcessesSelectedPagesInOrder(t *testing.T) {
	ctx := annotationTestContext(t)
	selectedPages := types.IntSet{0: true, 2: true}
	for range 200 {
		err := Resize(ctx, selectedPages, &model.Resize{Scale: 0.5})
		if err == nil || !strings.HasPrefix(err.Error(), "page 0:") {
			t.Fatalf("expected lowest selected page first, got %v", err)
		}
	}
}

// TestResizeAnnotationErrorIncludesObjectIdentity verifies indirect annotation object context.
func TestResizeAnnotationErrorIncludesObjectIdentity(t *testing.T) {
	ctx := annotationTestContext(t)
	for objNr := 77; objNr <= 82; objNr++ {
		ctx.Table[objNr] = model.NewXRefTableEntryGen0(types.Integer(1))
	}
	ctx.Table[81] = model.NewXRefTableEntryGen0(types.Array{types.Integer(1)})
	ctx.Table[82] = model.NewXRefTableEntryGen0(types.Array{
		types.Name("bad"), types.Integer(0), types.Integer(1), types.Integer(1),
		types.Integer(2), types.Integer(2), types.Integer(3), types.Integer(3),
	})
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "annotation", run: func() error {
			return resizePageAnnotations(ctx, types.Dict{"Annots": types.Array{*types.NewIndirectRef(77, 0)}}, matrix.IdentMatrix)
		}, want: "annotation 1 obj#77"},
		{name: "Annots", run: func() error {
			return resizePageAnnotations(ctx, types.Dict{"Annots": *types.NewIndirectRef(78, 0)}, matrix.IdentMatrix)
		}, want: "Annots obj#78"},
		{name: "Rect", run: func() error {
			return resizeAnnotationRect(ctx, types.Dict{"Rect": *types.NewIndirectRef(79, 0)}, matrix.IdentMatrix)
		}, want: "annotation Rect obj#79"},
		{name: "QuadPoints", run: func() error {
			return resizeAnnotationQuadPoints(ctx, types.Dict{"QuadPoints": *types.NewIndirectRef(80, 0)}, matrix.IdentMatrix)
		}, want: "annotation QuadPoints obj#80"},
		{name: "Rect length", run: func() error {
			return resizeAnnotationRect(ctx, types.Dict{"Rect": *types.NewIndirectRef(81, 0)}, matrix.IdentMatrix)
		}, want: "annotation Rect obj#81: invalid length 1"},
		{name: "QuadPoints coordinate", run: func() error {
			return resizeAnnotationQuadPoints(ctx, types.Dict{"QuadPoints": *types.NewIndirectRef(82, 0)}, matrix.IdentMatrix)
		}, want: "annotation QuadPoints obj#82[0]: dereference x coordinate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestParseResizeConfigReportsParameterContext verifies parameter error context.
func TestParseResizeConfigReportsParameterContext(t *testing.T) {
	tests := []struct {
		config string
		param  string
	}{
		{config: "border:maybe", param: "border"},
		{config: "sc:nope", param: "sc"},
		{config: "dim:nope", param: "dim"},
	}
	for _, tt := range tests {
		_, err := ParseResizeConfig(tt.config, types.POINTS)
		want := fmt.Sprintf("resize parameter %q", tt.param)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("ParseResizeConfig(%q): expected %q, got %v", tt.config, want, err)
		}
	}
}

// TestParseResizeConfigRejectsScaleAndDimensions verifies conflicting resize modes are rejected by the parser.
func TestParseResizeConfigRejectsScaleAndDimensions(t *testing.T) {
	for _, config := range []string{
		"sc:.5, dim:100 100",
		"dim:100 100, sc:.5",
		"sc:.5, form:A4",
	} {
		_, err := ParseResizeConfig(config, types.POINTS)
		if err == nil || !strings.Contains(err.Error(), "conflicts") {
			t.Errorf("ParseResizeConfig(%q): expected conflicting resize mode error, got %v", config, err)
		}
	}
}

// TestParseResizeConfigRejectsMalformedClauses verifies positional context for malformed parser input.
func TestParseResizeConfigRejectsMalformedClauses(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   []string
	}{
		{name: "malformed first clause", config: "sc=.5, border:on", want: []string{"resize configuration clause 1", `missing ":" separator`}},
		{name: "malformed later clause", config: "sc:.5, border", want: []string{"resize configuration clause 2", `missing ":" separator`}},
		{name: "empty parameter name", config: ":.5", want: []string{"resize configuration clause 1", "missing parameter name"}},
		{name: "empty parameter value", config: "sc:", want: []string{"resize configuration clause 1", "missing parameter value"}},
		{name: "too many separators", config: "sc:.5:extra", want: []string{"resize configuration clause 1", `resize parameter "sc"`, `too many ":" separators`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseResizeConfig(tt.config, types.POINTS)
			if err == nil {
				t.Fatal("expected parser error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q in %q", want, err)
				}
			}
		})
	}
}

// TestPrepTransformHonorsEnforcedOrientation verifies destination orientation is preserved when enforced.
func TestPrepTransformHonorsEnforcedOrientation(t *testing.T) {
	src := types.RectForDim(200, 100)
	dest := types.RectForDim(100, 200)
	prepTransform(src, dest, true)
	if dest.Width() != 100 || dest.Height() != 200 {
		t.Fatalf("expected enforced portrait destination, got %.0fx%.0f", dest.Width(), dest.Height())
	}

	dest = types.RectForDim(100, 200)
	prepTransform(src, dest, false)
	if dest.Width() != 200 || dest.Height() != 100 {
		t.Fatalf("expected destination orientation adjustment, got %.0fx%.0f", dest.Width(), dest.Height())
	}
}

// TestResizeAnnotationEntryErrorContext verifies semantic annotation entry context.
func TestResizeAnnotationEntryErrorContext(t *testing.T) {
	ctx := annotationTestContext(t)
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "Rect array", run: func() error {
			return resizeAnnotationRect(ctx, types.Dict{"Rect": types.Integer(1)}, matrix.IdentMatrix)
		}, want: "annotation Rect: dereference array"},
		{name: "Rect length", run: func() error {
			return resizeAnnotationRect(ctx, types.Dict{"Rect": types.Array{types.Integer(1)}}, matrix.IdentMatrix)
		}, want: "annotation Rect: invalid length 1"},
		{name: "Rect coordinate", run: func() error {
			return resizeAnnotationRect(ctx, types.Dict{"Rect": types.Array{types.Name("bad"), types.Integer(0), types.Integer(1), types.Integer(1)}}, matrix.IdentMatrix)
		}, want: "annotation Rect: resolve rectangle"},
		{name: "QuadPoints array", run: func() error {
			return resizeAnnotationQuadPoints(ctx, types.Dict{"QuadPoints": types.Integer(1)}, matrix.IdentMatrix)
		}, want: "annotation QuadPoints: dereference array"},
		{name: "QuadPoints length", run: func() error {
			return resizeAnnotationQuadPoints(ctx, types.Dict{"QuadPoints": types.Array{types.Integer(1)}}, matrix.IdentMatrix)
		}, want: "annotation QuadPoints: invalid length 1"},
		{name: "QuadPoints coordinate", run: func() error {
			a := types.Array{types.Name("bad"), types.Integer(0), types.Integer(1), types.Integer(1), types.Integer(2), types.Integer(2), types.Integer(3), types.Integer(3)}
			return resizeAnnotationQuadPoints(ctx, types.Dict{"QuadPoints": a}, matrix.IdentMatrix)
		}, want: "annotation QuadPoints[0]: dereference x coordinate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestResizePageAnnotationsErrorContext verifies annotation indexing and entry context.
func TestResizePageAnnotationsErrorContext(t *testing.T) {
	ctx := annotationTestContext(t)
	d := types.Dict{"Annots": types.Array{types.Dict{}, types.Integer(1)}}
	err := resizePageAnnotations(ctx, d, matrix.IdentMatrix)
	if err == nil || !strings.Contains(err.Error(), "annotation 2: dereference dictionary") {
		t.Fatalf("expected annotation index context, got %v", err)
	}

	d = types.Dict{"Annots": types.Array{types.Dict{"Rect": types.Array{types.Integer(1)}}}}
	err = resizePageAnnotations(ctx, d, matrix.IdentMatrix)
	if err == nil || !strings.Contains(err.Error(), "annotation 1: annotation Rect: invalid length 1") {
		t.Fatalf("expected annotation entry context, got %v", err)
	}
}

// TestResizePageErrorContext verifies page and content operation context.
func TestResizePageErrorContext(t *testing.T) {
	ctx := annotationTestContext(t)
	err := Resize(ctx, types.IntSet{99: true}, &model.Resize{Scale: 0.5})
	if err == nil || !strings.Contains(err.Error(), "page 99: page dictionary") {
		t.Fatalf("expected page dictionary context, got %v", err)
	}

	ctx = annotationTestContext(t)
	d := annotationTestPageDict(t, ctx)
	d["Contents"] = types.Integer(1)
	err = Resize(ctx, types.IntSet{1: true}, &model.Resize{Scale: 0.5})
	if err == nil || !strings.Contains(err.Error(), "page 1: read page content") {
		t.Fatalf("expected page content context, got %v", err)
	}

	ctx = annotationTestContext(t)
	d = annotationTestPageDict(t, ctx)
	d["Annots"] = types.Integer(1)
	err = Resize(ctx, types.IntSet{1: true}, &model.Resize{Scale: 0.5})
	if err == nil || !strings.Contains(err.Error(), "page 1: resize annotations: Annots: dereference array") {
		t.Fatalf("expected page annotation context, got %v", err)
	}
}

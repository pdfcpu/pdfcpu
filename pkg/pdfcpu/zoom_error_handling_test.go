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
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestZoomPageNumbers verifies sorted selected-page expansion.
func TestZoomPageNumbers(t *testing.T) {
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
			if got := zoomPageNumbers(tt.count, tt.selected); !slices.Equal(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

// TestZoomProcessesSelectedPagesInOrder verifies deterministic page failure ordering.
func TestZoomProcessesSelectedPagesInOrder(t *testing.T) {
	ctx := annotationTestContext(t)
	selectedPages := types.IntSet{0: true, 2: true}
	for range 200 {
		err := Zoom(ctx, selectedPages, &model.Zoom{Factor: 0.5})
		if err == nil || !strings.HasPrefix(err.Error(), "page 0:") {
			t.Fatalf("expected lowest selected page first, got %v", err)
		}
	}
}

// TestZoomPageErrorContext verifies page and content operation context.
func TestZoomPageErrorContext(t *testing.T) {
	ctx := annotationTestContext(t)
	err := Zoom(ctx, types.IntSet{99: true}, &model.Zoom{Factor: 0.5})
	if err == nil || !strings.Contains(err.Error(), "page 99: page dictionary") {
		t.Fatalf("expected page dictionary context, got %v", err)
	}

	ctx = annotationTestContext(t)
	d := annotationTestPageDict(t, ctx)
	d["Contents"] = types.Integer(1)
	err = Zoom(ctx, types.IntSet{1: true}, &model.Zoom{Factor: 0.5})
	if err == nil || !strings.Contains(err.Error(), "page 1: read page content") {
		t.Fatalf("expected page content context, got %v", err)
	}

	ctx = annotationTestContext(t)
	err = Zoom(ctx, types.IntSet{1: true}, &model.Zoom{HMargin: 10000})
	if err == nil || !strings.Contains(err.Error(), "page 1: derive factor and margins") {
		t.Fatalf("expected margin context, got %v", err)
	}
}

// TestZoomBlankPageCreatesContentForDecorations verifies blank-page decoration behavior.
func TestZoomBlankPageCreatesContentForDecorations(t *testing.T) {
	bgColor := color.LightGray
	tests := []struct {
		name        string
		zoom        *model.Zoom
		wantContent bool
	}{
		{name: "plain", zoom: &model.Zoom{Factor: 0.5}},
		{name: "border", zoom: &model.Zoom{Factor: 0.5, Border: true}, wantContent: true},
		{name: "background", zoom: &model.Zoom{Factor: 0.5, BgColor: &bgColor}, wantContent: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := annotationTestContext(t)
			d := annotationTestPageDict(t, ctx)
			d.Delete("Contents")
			if err := Zoom(ctx, types.IntSet{1: true}, tt.zoom); err != nil {
				t.Fatal(err)
			}
			contents, found := d.Find("Contents")
			if found != tt.wantContent {
				t.Fatalf("content stream present=%t, want %t", found, tt.wantContent)
			}
			if !found {
				return
			}
			sd, _, err := ctx.DereferenceStreamDict(contents)
			if err != nil {
				t.Fatal(err)
			}
			if sd == nil {
				t.Fatal("missing content stream")
			}
			if err := sd.Decode(); err != nil {
				t.Fatal(err)
			}
			if len(sd.Content) == 0 {
				t.Fatal("empty content stream")
			}
		})
	}
}

// TestZoomDecoratedBlankPageHonorsCropBoxOrigin verifies decoration coordinates use the crop-box origin.
func TestZoomDecoratedBlankPageHonorsCropBoxOrigin(t *testing.T) {
	ctx := annotationTestContext(t)
	d := annotationTestPageDict(t, ctx)
	d.Delete("Contents")
	d["CropBox"] = types.NewRectangle(10, 20, 210, 120).Array()
	bgColor := color.LightGray
	zoom := &model.Zoom{Factor: 0.5, Border: true, BgColor: &bgColor}
	if err := Zoom(ctx, types.IntSet{1: true}, zoom); err != nil {
		t.Fatal(err)
	}
	contents, found := d.Find("Contents")
	if !found {
		t.Fatal("missing content stream")
	}
	sd, _, err := ctx.DereferenceStreamDict(contents)
	if err != nil {
		t.Fatal(err)
	}
	if sd == nil {
		t.Fatal("missing content stream dictionary")
	}
	if err := sd.Decode(); err != nil {
		t.Fatal(err)
	}
	content := string(sd.Content)
	for _, want := range []string{
		"10.00 20.00 200.00 25.00 re B",
		"10.00 95.00 200.00 25.00 re B",
		"10.00 45.00 50.00 50.00 re B",
		"160.00 45.00 50.00 50.00 re B",
		"60.00 45.00 100.00 50.00 re s",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in %q", want, content)
		}
	}
}

// TestParseZoomConfigReportsParameterContext verifies clause and parameter context.
func TestParseZoomConfigReportsParameterContext(t *testing.T) {
	tests := []struct {
		config string
		param  string
	}{
		{config: "border:maybe", param: "border"},
		{config: "factor:nope", param: "factor"},
		{config: "bgcolor:nope", param: "bgcolor"},
	}
	for _, tt := range tests {
		_, err := ParseZoomConfig(tt.config, types.POINTS)
		for _, want := range []string{"zoom configuration clause 1", `zoom parameter "` + tt.param + `"`} {
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("ParseZoomConfig(%q): expected %q, got %v", tt.config, want, err)
			}
		}
	}
}

// TestParseZoomConfigPreservesNumericParseErrors verifies context and NumError preservation.
func TestParseZoomConfigPreservesNumericParseErrors(t *testing.T) {
	tests := []struct {
		name   string
		config string
		param  string
		detail string
	}{
		{name: "factor", config: "factor:nope", param: "factor", detail: "parse float value"},
		{name: "horizontal margin", config: "hmargin:nope", param: "hmargin", detail: "parse numeric value"},
		{name: "vertical margin", config: "vmargin:nope", param: "vmargin", detail: "parse numeric value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseZoomConfig(tt.config, types.POINTS)
			for _, want := range []string{"zoom configuration clause 1", `zoom parameter "` + tt.param + `"`, tt.detail} {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q in %v", want, err)
				}
			}
			var numErr *strconv.NumError
			if !errors.As(err, &numErr) {
				t.Fatalf("expected *strconv.NumError, got %v", err)
			}
			if numErr.Func != "ParseFloat" || numErr.Num != "nope" {
				t.Fatalf("unexpected numeric parse error: %+v", numErr)
			}
		})
	}
}

// TestParseZoomConfigRejectsMalformedClauses verifies indexed malformed-clause context.
func TestParseZoomConfigRejectsMalformedClauses(t *testing.T) {
	tests := []struct {
		name   string
		config string
		want   []string
	}{
		{name: "malformed first clause", config: "factor=.5, border:on", want: []string{"zoom configuration clause 1", `missing ":" separator`}},
		{name: "malformed later clause", config: "factor:.5, border", want: []string{"zoom configuration clause 2", `missing ":" separator`}},
		{name: "empty parameter name", config: ":.5", want: []string{"zoom configuration clause 1", "missing parameter name"}},
		{name: "empty parameter value", config: "factor:", want: []string{"zoom configuration clause 1", "missing parameter value"}},
		{name: "too many separators", config: "factor:.5:extra", want: []string{"zoom configuration clause 1", `zoom parameter "factor"`, `too many ":" separators`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseZoomConfig(tt.config, types.POINTS)
			for _, want := range tt.want {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Errorf("expected %q in %v", want, err)
				}
			}
		})
	}
}

// TestParseZoomConfigRejectsConflictingModes verifies factor and margin conflicts.
func TestParseZoomConfigRejectsConflictingModes(t *testing.T) {
	_, err := ParseZoomConfig("factor:.5, hmargin:10", types.POINTS)
	if err == nil || !strings.Contains(err.Error(), "factor conflicts with margins") {
		t.Fatalf("expected conflict context, got %v", err)
	}
}

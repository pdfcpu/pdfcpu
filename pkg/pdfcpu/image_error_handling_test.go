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
	"bytes"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func imageOperationPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func imageOperationTarget(t *testing.T, ctx *model.Context, objNr, width, height int) types.IndirectRef {
	t.Helper()
	sd := types.StreamDict{Dict: types.Dict{
		"Type":    types.Name("XObject"),
		"Subtype": types.Name("Image"),
		"Width":   types.Integer(width),
		"Height":  types.Integer(height),
	}}
	if _, err := ctx.IndRefForObject(objNr, sd); err != nil {
		t.Fatal(err)
	}
	ctx.Optimize.ImageObjects[objNr] = &model.ImageObject{ImageDict: &sd}
	return *types.NewIndirectRef(objNr, 0)
}

func imageOperationPage(t *testing.T) (*model.Context, types.Dict) {
	t.Helper()
	ctx := testOptimizeContext(t)
	return ctx, addOptimizeTestPage(t, ctx)
}

func setInheritedImageResource(t *testing.T, ctx *model.Context, pageDict types.Dict, id string, ref types.IndirectRef) {
	t.Helper()
	parentRef := pageDict.IndirectRefEntry("Parent")
	if parentRef == nil {
		t.Fatal("missing page parent")
	}
	parent, err := ctx.DereferenceDict(*parentRef)
	if err != nil {
		t.Fatal(err)
	}
	pageDict.Delete("Resources")
	parent["Resources"] = types.Dict{"XObject": types.Dict{id: ref}}
}

// TestImagesAddsPageContext verifies image-list failures identify the selected page.
func TestImagesAddsPageContext(t *testing.T) {
	ctx := &model.Context{
		XRefTable: &model.XRefTable{
			PageCount:  1,
			PageThumbs: map[int]types.IndirectRef{1: *types.NewIndirectRef(7, 0)},
		},
		Optimize: &model.OptimizationContext{},
	}
	_, _, err := Images(ctx, types.IntSet{1: true})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1: thumbnail obj#7") {
		t.Fatalf("expected page image context, got %q", err)
	}
	if got := strings.Count(err.Error(), "page 1:"); got != 1 {
		t.Fatalf("expected page context once, got %d in %q", got, err)
	}
}

// TestUpdateImagesByObjNrAddsConstructionContext verifies image decode causes remain discoverable.
func TestUpdateImagesByObjNrAddsConstructionContext(t *testing.T) {
	ctx := testOptimizeContext(t)
	err := UpdateImagesByObjNr(ctx, bytes.NewReader(nil), 7)
	if !errors.Is(err, image.ErrFormat) {
		t.Fatalf("expected %v, got %v", image.ErrFormat, err)
	}
	for _, want := range []string{"image obj#7: create replacement", "decode image configuration"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

// TestValidateImageDimensionsRejectsMalformedOptimizationEntries verifies nil-sensitive image metadata.
func TestValidateImageDimensionsRejectsMalformedOptimizationEntries(t *testing.T) {
	ctx := testOptimizeContext(t)
	tests := []struct {
		name string
		obj  *model.ImageObject
		want string
	}{
		{name: "missing entry", want: "image obj#7: missing optimization entry"},
		{name: "missing dictionary", obj: &model.ImageObject{}, want: "image obj#7: missing image dictionary"},
		{
			name: "missing dimensions",
			obj:  &model.ImageObject{ImageDict: &types.StreamDict{Dict: types.Dict{}}},
			want: "image obj#7: missing width or height",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.obj == nil {
				delete(ctx.Optimize.ImageObjects, 7)
			} else {
				ctx.Optimize.ImageObjects[7] = tt.obj
			}
			err := validateImageDimensions(ctx, 7, 1, 1)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestUpdateImagesByObjNrReportsMissingXRefEntry verifies target lookup context.
func TestUpdateImagesByObjNrReportsMissingXRefEntry(t *testing.T) {
	ctx := testOptimizeContext(t)
	ctx.Optimize.ImageObjects[7] = &model.ImageObject{ImageDict: &types.StreamDict{Dict: types.Dict{
		"Width":  types.Integer(1),
		"Height": types.Integer(1),
	}}}
	err := UpdateImagesByObjNr(ctx, bytes.NewReader(imageOperationPNG(t, 1, 1)), 7)
	if err == nil || !strings.Contains(err.Error(), "image obj#7: missing xref entry") {
		t.Fatalf("expected xref context, got %v", err)
	}
}

// TestUpdateImagesByPageNrAndIdRejectsMalformedResource verifies malformed XObjects return errors instead of panicking.
func TestUpdateImagesByPageNrAndIdRejectsMalformedResource(t *testing.T) {
	ctx, pageDict := imageOperationPage(t)
	pageDict["Resources"] = types.Dict{"XObject": types.Dict{"Im0": types.Name("broken")}}

	err := UpdateImagesByPageNrAndId(ctx, bytes.NewReader(nil), 1, "Im0")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page 1 resource Im0: expected indirect reference, got types.Name") {
		t.Fatalf("expected malformed resource context, got %q", err)
	}
}

// TestUpdateImagesByPageNrAndIdAddsDereferenceContext verifies XObject dictionary failures identify the page resource.
func TestUpdateImagesByPageNrAndIdAddsDereferenceContext(t *testing.T) {
	ctx, pageDict := imageOperationPage(t)
	pageDict["Resources"] = types.Dict{"XObject": *types.NewIndirectRef(999, 0)}

	err := UpdateImagesByPageNrAndId(ctx, bytes.NewReader(nil), 1, "Im0")
	if err == nil || !strings.Contains(err.Error(), "page 1 resource Im0: missing XObject dictionary for obj#999") {
		t.Fatalf("expected XObject identity context, got %v", err)
	}
}

// TestUpdateImagesByPageNrAndIdPreservesPageSentinel verifies page lookup causes remain discoverable.
func TestUpdateImagesByPageNrAndIdPreservesPageSentinel(t *testing.T) {
	ctx := testOptimizeContext(t)
	err := UpdateImagesByPageNrAndId(ctx, bytes.NewReader(nil), 1, "Im0")
	if !errors.Is(err, model.ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
	}
	if !strings.Contains(err.Error(), "page 1 resource Im0: resolve page dictionary") {
		t.Fatalf("expected page dictionary context, got %q", err)
	}
}

// TestUpdateImagesByPageNrAndIdReplacesDirectResource verifies direct XObject replacement.
func TestUpdateImagesByPageNrAndIdReplacesDirectResource(t *testing.T) {
	ctx, pageDict := imageOperationPage(t)
	targetRef := imageOperationTarget(t, ctx, 7, 1, 1)
	xObjects := types.Dict{"Im0": targetRef}
	pageDict["Resources"] = types.Dict{"XObject": xObjects}

	if err := UpdateImagesByPageNrAndId(ctx, bytes.NewReader(imageOperationPNG(t, 1, 1)), 1, "Im0"); err != nil {
		t.Fatal(err)
	}
	got := xObjects.IndirectRefEntry("Im0")
	if got == nil || got.ObjectNumber.Value() == targetRef.ObjectNumber.Value() {
		t.Fatalf("expected replacement reference, got %v", got)
	}
}

// TestUpdateImagesByPageNrAndIdIgnoresShadowedInheritedResources verifies direct page resources own lookup.
func TestUpdateImagesByPageNrAndIdIgnoresShadowedInheritedResources(t *testing.T) {
	ctx, pageDict := imageOperationPage(t)
	targetRef := imageOperationTarget(t, ctx, 7, 1, 1)
	xObjects := types.Dict{"Im0": targetRef}
	pageDict["Resources"] = types.Dict{"XObject": xObjects}

	parentRef := pageDict.IndirectRefEntry("Parent")
	if parentRef == nil {
		t.Fatal("missing page parent")
	}
	parent, err := ctx.DereferenceDict(*parentRef)
	if err != nil {
		t.Fatal(err)
	}
	parent["Resources"] = types.Dict{"XObject": types.Name("broken")}

	if err := UpdateImagesByPageNrAndId(ctx, bytes.NewReader(imageOperationPNG(t, 1, 1)), 1, "Im0"); err != nil {
		t.Fatal(err)
	}
}

// TestUpdateImagesByPageNrAndIdValidatesInheritedDimensions verifies inherited resources use the target image dimensions.
func TestUpdateImagesByPageNrAndIdValidatesInheritedDimensions(t *testing.T) {
	ctx, pageDict := imageOperationPage(t)
	targetRef := imageOperationTarget(t, ctx, 7, 2, 2)
	setInheritedImageResource(t, ctx, pageDict, "Im0", targetRef)

	err := UpdateImagesByPageNrAndId(ctx, bytes.NewReader(imageOperationPNG(t, 1, 1)), 1, "Im0")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page 1 resource Im0", "image obj#7: replacement dimensions 1x1, want 2x2"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err)
		}
	}
}

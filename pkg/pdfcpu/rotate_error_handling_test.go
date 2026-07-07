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
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestComposePageRotationAvoidsIntegerOverflow verifies bounded rotation arithmetic.
func TestComposePageRotationAvoidsIntegerOverflow(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	delta := maxInt - maxInt%90
	want := 270 + delta%360
	if want >= 360 {
		want -= 360
	}

	got := composePageRotation(270, delta)
	if got != want {
		t.Fatalf("expected %d, got %d", want, got)
	}
	if got%90 != 0 {
		t.Fatalf("expected a multiple of 90, got %d", got)
	}
}

// TestRotatePagesComposesRotationValues verifies page dictionary rotation updates.
func TestRotatePagesComposesRotationValues(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	largePositive := maxInt - maxInt%90
	largeNegative := minInt - minInt%90
	tests := []struct {
		name    string
		current int
		delta   int
	}{
		{name: "positive", current: 270, delta: 90},
		{name: "negative", current: 270, delta: -90},
		{name: "zero", current: 90},
		{name: "large positive", current: 270, delta: largePositive},
		{name: "large negative", current: 270, delta: largeNegative},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := annotationTestContext(t)
			d := annotationTestPageDict(t, ctx)
			d["Rotate"] = types.Integer(tt.current)
			if err := RotatePages(ctx, types.IntSet{1: true}, tt.delta); err != nil {
				t.Fatal(err)
			}
			got := d.IntEntry("Rotate")
			if got == nil {
				t.Fatal("expected page rotation")
			}
			want := ((tt.current/90 + tt.delta/90) % 4) * 90
			if want < 0 {
				want += 360
			}
			if *got != want {
				t.Fatalf("expected %d, got %d", want, *got)
			}
			if *got%90 != 0 {
				t.Fatalf("expected a multiple of 90, got %d", *got)
			}
		})
	}
}

// TestRotatePagesErrorIncludesPageContext verifies page dictionary error context.
func TestRotatePagesErrorIncludesPageContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{PageCount: 1}}
	err := RotatePages(ctx, types.IntSet{1: true}, 90)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page 1", "page dict"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestRotatePagesPreservesPageNotFound verifies page-not-found sentinel preservation.
func TestRotatePagesPreservesPageNotFound(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{PageCount: 1}}
	err := RotatePages(ctx, types.IntSet{2: true}, 90)
	if !errors.Is(err, model.ErrPageNotFound) {
		t.Fatalf("expected %v, got %v", model.ErrPageNotFound, err)
	}
	if !strings.Contains(err.Error(), "page 2: page dict") {
		t.Fatalf("expected page dictionary context, got %q", err.Error())
	}
}

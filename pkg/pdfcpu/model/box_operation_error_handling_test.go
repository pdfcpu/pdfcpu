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

package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestBoxOperationsIncludePageContext(t *testing.T) {
	ctx := &Context{XRefTable: &XRefTable{}}
	pages := types.IntSet{1: true}
	pb := &PageBoundaries{}
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "add", run: func() error { return ctx.AddPageBoundaries(pages, pb) }},
		{name: "remove", run: func() error { return ctx.RemovePageBoundaries(pages, pb) }},
		{name: "crop", run: func() error { return ctx.Crop(pages, &Box{}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.Is(err, ErrPageNotFound) {
				t.Fatalf("expected %v, got %v", ErrPageNotFound, err)
			}
			if !strings.Contains(err.Error(), "page 1: page dictionary") {
				t.Fatalf("expected page dictionary context, got %q", err.Error())
			}
		})
	}
}

func TestPageBoundaryEntryContext(t *testing.T) {
	ctx := &Context{XRefTable: &XRefTable{}}
	_, err := ctx.pageBoundary(types.Dict{"TrimBox": types.Integer(1)}, "TrimBox")
	if err == nil || !strings.Contains(err.Error(), "TrimBox: dereference array") {
		t.Fatalf("expected TrimBox dereference context, got %v", err)
	}

	_, err = ctx.resolvePageBoundary(types.Dict{"CropBox": types.Integer(1)}, "CropBox")
	if err == nil || !strings.Contains(err.Error(), "CropBox: dereference array") {
		t.Fatalf("expected CropBox dereference context, got %v", err)
	}
}

func TestPageBoundariesIncludePageTreeAndPageContext(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	pageRef, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type":    types.Name("Page"),
		"TrimBox": types.Integer(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pageRef}
	xRefTable.PageCount = 1

	_, err = xRefTable.PageBoundaries(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page tree", "page 1", "TrimBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

// TestPageBoundariesRejectsNilPageTreeKid verifies contextual nil child handling.
func TestPageBoundariesRejectsNilPageTreeKid(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	pagesRef, err := xRefTable.IndRefForObject(2, types.Dict{
		"Type":     types.Name("Pages"),
		"Count":    types.Integer(1),
		"Kids":     types.Array{nil},
		"MediaBox": types.RectForFormat("A4").Array(),
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
	xRefTable.PageCount = 1

	_, err = xRefTable.PageBoundaries(nil)
	if err == nil || !strings.Contains(err.Error(), "page tree obj#2: kid 1: nil object") {
		t.Fatalf("expected nil child context, got %v", err)
	}
}

func pageBoundariesWithKid(t *testing.T, kid func(*testing.T, *XRefTable) types.Object) *XRefTable {
	t.Helper()
	xRefTable := newXRefTable(NewDefaultConfiguration())
	pagesRef, err := xRefTable.IndRefForObject(2, types.Dict{
		"Type":     types.Name("Pages"),
		"Count":    types.Integer(1),
		"Kids":     types.Array{kid(t, xRefTable)},
		"MediaBox": types.RectForFormat("A4").Array(),
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
	xRefTable.PageCount = 1
	return xRefTable
}

// TestPageBoundariesRejectsInvalidPageTreeKids verifies contextual page-tree child errors.
func TestPageBoundariesRejectsInvalidPageTreeKids(t *testing.T) {
	tests := []struct {
		name string
		kid  func(*testing.T, *XRefTable) types.Object
		want string
	}{
		{
			name: "direct object",
			kid:  func(*testing.T, *XRefTable) types.Object { return types.Integer(1) },
			want: "page tree obj#2: kid 1: expected indirect reference",
		},
		{
			name: "missing dict",
			kid:  func(*testing.T, *XRefTable) types.Object { return *types.NewIndirectRef(7, 0) },
			want: "page tree obj#2: kid 1 obj#7: missing dict",
		},
		{
			name: "missing Type",
			kid: func(t *testing.T, xRefTable *XRefTable) types.Object {
				indRef, err := xRefTable.IndRefForObject(1, types.Dict{})
				if err != nil {
					t.Fatal(err)
				}
				return *indRef
			},
			want: "page tree obj#2: kid 1 obj#1: missing Type",
		},
		{
			name: "unsupported Type",
			kid: func(t *testing.T, xRefTable *XRefTable) types.Object {
				indRef, err := xRefTable.IndRefForObject(1, types.Dict{"Type": types.Name("Other")})
				if err != nil {
					t.Fatal(err)
				}
				return *indRef
			},
			want: `page tree obj#2: kid 1 obj#1: unsupported Type "Other"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := pageBoundariesWithKid(t, tt.kid)
			_, err := xRefTable.PageBoundaries(nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestPageBoundariesDistinguishesKidsByNodeType verifies Page and Pages Kids semantics.
func TestPageBoundariesDistinguishesKidsByNodeType(t *testing.T) {
	tests := []struct {
		name        string
		kids        types.Object
		includeKids bool
		want        string
	}{
		{name: "missing", want: "page tree obj#2: missing Kids"},
		{name: "null", includeKids: true, want: "page tree obj#2: missing Kids"},
		{
			name:        "wrong type",
			kids:        types.Integer(1),
			includeKids: true,
			want:        "page tree obj#2: Kids: expected array, got types.Integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := newXRefTable(NewDefaultConfiguration())
			d := types.Dict{
				"Type":     types.Name("Pages"),
				"Count":    types.Integer(1),
				"MediaBox": types.RectForFormat("A4").Array(),
			}
			if tt.includeKids {
				d["Kids"] = tt.kids
			}
			pagesRef, err := xRefTable.IndRefForObject(2, d)
			if err != nil {
				t.Fatal(err)
			}
			xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
			xRefTable.PageCount = 1

			_, err = xRefTable.PageBoundaries(nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestPageBoundariesAllowsPageWithoutKids verifies leaf Page handling.
func TestPageBoundariesAllowsPageWithoutKids(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	pageRef, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type":     types.Name("Page"),
		"MediaBox": types.RectForFormat("A4").Array(),
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pageRef}
	xRefTable.PageCount = 1

	pb, err := xRefTable.PageBoundaries(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pb) != 1 || pb[0].MediaBox() == nil {
		t.Fatalf("expected one page with a MediaBox, got %v", pb)
	}
}

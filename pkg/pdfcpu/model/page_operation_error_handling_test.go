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

func emptyPageTestXRefTable() *XRefTable {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	size := 1
	xRefTable.Size = &size
	xRefTable.Table[0] = NewFreeHeadXRefTableEntry()
	return xRefTable
}

func pointFreeListAtMissingObject(xRefTable *XRefTable, objNr int) {
	offset := int64(objNr)
	xRefTable.Table[0].Offset = &offset
}

func addSingleFreeObject(xRefTable *XRefTable, objNr, nextObjNr int) {
	pointFreeListAtMissingObject(xRefTable, objNr)
	entry := NewFreeHeadXRefTableEntry()
	next := int64(nextObjNr)
	entry.Offset = &next
	xRefTable.Table[objNr] = entry
}

// TestEmptyPageRejectsNilParent verifies the construction boundary does not panic.
func TestEmptyPageRejectsNilParent(t *testing.T) {
	_, err := emptyPageTestXRefTable().EmptyPage(nil, types.RectForFormat("A4"), 0)
	if err == nil || !strings.Contains(err.Error(), "empty page: missing parent") {
		t.Fatalf("expected missing parent context, got %v", err)
	}
}

// TestEmptyPageObjectCreationErrorsIncludePhase verifies wrapped object-storage failures.
func TestEmptyPageObjectCreationErrorsIncludePhase(t *testing.T) {
	parent := types.NewIndirectRef(10, 0)
	tests := []struct {
		name  string
		setup func(*XRefTable)
		want  string
	}{
		{
			name: "content object",
			setup: func(xRefTable *XRefTable) {
				pointFreeListAtMissingObject(xRefTable, 999)
			},
			want: "empty page: create content object",
		},
		{
			name: "page object",
			setup: func(xRefTable *XRefTable) {
				addSingleFreeObject(xRefTable, 1, 999)
			},
			want: "empty page: create page object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := emptyPageTestXRefTable()
			tt.setup(xRefTable)
			_, err := xRefTable.EmptyPage(parent, types.RectForFormat("A4"), 0)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			cause := errors.Unwrap(err)
			if cause == nil || !strings.Contains(cause.Error(), "no entry for obj #999") {
				t.Fatalf("expected discoverable object-creation cause, got %v", cause)
			}
		})
	}
}

// TestInsertBlankPagesRetainsPageCreationContext verifies insertion owns the selected page identity.
func TestInsertBlankPagesRetainsPageCreationContext(t *testing.T) {
	xRefTable := emptyPageTestXRefTable()
	pageRef, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type":     types.Name("Page"),
		"Parent":   *types.NewIndirectRef(2, 0),
		"MediaBox": types.RectForFormat("A4").Array(),
	})
	if err != nil {
		t.Fatal(err)
	}
	pagesRef, err := xRefTable.IndRefForObject(2, types.Dict{
		"Type":     types.Name("Pages"),
		"Count":    types.Integer(1),
		"Kids":     types.Array{*pageRef},
		"MediaBox": types.RectForFormat("A4").Array(),
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
	xRefTable.PageCount = 1
	pointFreeListAtMissingObject(xRefTable, 999)

	err = xRefTable.InsertBlankPages(types.IntSet{1: true}, nil, false)
	for _, want := range []string{"page 1: create blank page", "empty page: create content object"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q, got %v", want, err)
		}
	}
}

// TestPageMediaBoxUsesDomainContext verifies MediaBox-specific helper errors.
func TestPageMediaBoxUsesDomainContext(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	tests := []struct {
		name string
		d    types.Dict
		want string
	}{
		{
			name: "missing",
			d:    types.Dict{},
			want: "missing MediaBox",
		},
		{
			name: "dereference array",
			d:    types.Dict{"MediaBox": types.Integer(1)},
			want: "MediaBox: dereference array",
		},
		{
			name: "rectangle",
			d:    types.Dict{"MediaBox": types.Array{types.Integer(0)}},
			want: "MediaBox: rectangle: expected 4 elements, got 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := xRefTable.pageMediaBox(tt.d)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), "pageMediaBox") {
				t.Fatalf("unexpected implementation name in %q", err)
			}
		})
	}
}

// TestInsertBlankPagesRejectsMissingPageTree verifies missing page-tree handling.
func TestInsertBlankPagesRejectsMissingPageTree(t *testing.T) {
	tests := []struct {
		name      string
		xRefTable *XRefTable
		want      string
	}{
		{
			name:      "missing catalog",
			xRefTable: newXRefTable(NewDefaultConfiguration()),
			want:      "pages root: missing root dict",
		},
		{
			name: "missing pages root",
			xRefTable: func() *XRefTable {
				xRefTable := newXRefTable(NewDefaultConfiguration())
				xRefTable.RootDict = types.Dict{}
				return xRefTable
			}(),
			want: "missing pages root",
		},
		{
			name: "unresolved pages root",
			xRefTable: func() *XRefTable {
				xRefTable := newXRefTable(NewDefaultConfiguration())
				xRefTable.RootDict = types.Dict{"Pages": *types.NewIndirectRef(7, 0)}
				return xRefTable
			}(),
			want: "page tree: page tree obj#7: missing dict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.xRefTable.InsertBlankPages(types.IntSet{1: true}, nil, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func insertBlankPagesTestXRefTable(t *testing.T, kid types.Object) *XRefTable {
	t.Helper()
	xRefTable := newXRefTable(NewDefaultConfiguration())
	pagesRef, err := xRefTable.IndRefForObject(2, types.Dict{
		"Type":     types.Name("Pages"),
		"Count":    types.Integer(1),
		"Kids":     types.Array{kid},
		"MediaBox": types.RectForFormat("A4").Array(),
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
	xRefTable.PageCount = 1
	return xRefTable
}

// TestInsertBlankPagesRejectsInvalidKidsEntry verifies page-tree Kids invariants.
func TestInsertBlankPagesRejectsInvalidKidsEntry(t *testing.T) {
	tests := []struct {
		name    string
		kids    types.Object
		include bool
		want    string
	}{
		{
			name: "missing",
			want: "page tree obj#2: missing Kids",
		},
		{
			name:    "wrong type",
			kids:    types.Integer(1),
			include: true,
			want:    "page tree obj#2: Kids: expected array, got types.Integer",
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
			if tt.include {
				d["Kids"] = tt.kids
			}
			pagesRef, err := xRefTable.IndRefForObject(2, d)
			if err != nil {
				t.Fatal(err)
			}
			xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
			xRefTable.PageCount = 1

			err = xRefTable.InsertBlankPages(types.IntSet{1: true}, nil, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestInsertBlankPagesRejectsInvalidPageTreeKids verifies contextual child errors.
func TestInsertBlankPagesRejectsInvalidPageTreeKids(t *testing.T) {
	t.Run("nil object", func(t *testing.T) {
		xRefTable := insertBlankPagesTestXRefTable(t, nil)
		err := xRefTable.InsertBlankPages(types.IntSet{1: true}, nil, false)
		if err == nil || !strings.Contains(err.Error(), "page tree obj#2: kid 1: nil object") {
			t.Fatalf("expected nil child context, got %v", err)
		}
	})

	t.Run("direct object", func(t *testing.T) {
		xRefTable := insertBlankPagesTestXRefTable(t, types.Integer(1))
		err := xRefTable.InsertBlankPages(types.IntSet{1: true}, nil, false)
		for _, want := range []string{"page tree obj#2: kid 1", "expected indirect reference"} {
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q, got %v", want, err)
			}
		}
	})

	t.Run("missing type", func(t *testing.T) {
		xRefTable := newXRefTable(NewDefaultConfiguration())
		kidRef, err := xRefTable.IndRefForObject(1, types.Dict{})
		if err != nil {
			t.Fatal(err)
		}
		pagesRef, err := xRefTable.IndRefForObject(2, types.Dict{
			"Type":     types.Name("Pages"),
			"Count":    types.Integer(1),
			"Kids":     types.Array{*kidRef},
			"MediaBox": types.RectForFormat("A4").Array(),
		})
		if err != nil {
			t.Fatal(err)
		}
		xRefTable.RootDict = types.Dict{"Pages": *pagesRef}
		xRefTable.PageCount = 1

		err = xRefTable.InsertBlankPages(types.IntSet{1: true}, nil, false)
		if err == nil || !strings.Contains(err.Error(), "page tree kid obj#1: missing Type") {
			t.Fatalf("expected missing Type context, got %v", err)
		}
	})
}

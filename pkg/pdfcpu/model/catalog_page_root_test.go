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
	"context"
	"errors"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestCatalogContract verifies required catalog resolution and non-nil success results.
func TestCatalogContract(t *testing.T) {
	tests := []struct {
		name    string
		root    *types.IndirectRef
		entry   *XRefTableEntry
		cached  types.Dict
		wantErr string
	}{
		{name: "MissingRoot", wantErr: "missing root dict"},
		{name: "UnresolvedRoot", root: types.NewIndirectRef(1, 0), wantErr: "missing root dict"},
		{name: "NullObject", root: types.NewIndirectRef(1, 0), entry: NewXRefTableEntryGen0(nil), wantErr: "missing root dict"},
		{name: "NilDictionary", root: types.NewIndirectRef(1, 0), entry: NewXRefTableEntryGen0(types.Dict(nil)), wantErr: "missing root dict"},
		{name: "WrongType", root: types.NewIndirectRef(1, 0), entry: NewXRefTableEntryGen0(types.Integer(1)), wantErr: "corrupt root dict"},
		{name: "Dictionary", root: types.NewIndirectRef(1, 0), entry: NewXRefTableEntryGen0(types.Dict{})},
		{name: "CachedDictionary", cached: types.Dict{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newXRefTable(NewStatelessConfiguration())
			x.Root, x.RootDict = tt.root, tt.cached
			if tt.entry != nil {
				x.Table[1] = tt.entry
			}
			d, err := x.Catalog()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr || d != nil {
					t.Fatalf("Catalog: got (%v, %v), want (nil, %s)", d, err, tt.wantErr)
				}
				return
			}
			if err != nil || d == nil {
				t.Fatalf("Catalog: got (%v, %v), want non-nil dictionary without error", d, err)
			}
		})
	}
}

// TestParseRootVersion verifies optional version lookup preserves root resolution behavior.
func TestParseRootVersion(t *testing.T) {
	tests := []struct {
		name    string
		root    *types.IndirectRef
		object  types.Object
		cached  types.Dict
		want    string
		wantErr string
	}{
		{name: "MissingRoot", wantErr: "missing root dict"},
		{name: "UnresolvedRoot", root: types.NewIndirectRef(1, 0)},
		{name: "NullRoot", root: types.NewIndirectRef(1, 0), object: types.Dict(nil)},
		{name: "WrongRootType", root: types.NewIndirectRef(1, 0), object: types.Integer(1), wantErr: "corrupt root dict"},
		{name: "NoOverride", root: types.NewIndirectRef(1, 0), object: types.Dict{}},
		{name: "Override", root: types.NewIndirectRef(1, 0), object: types.Dict{"Version": types.Name("1.7")}, want: "1.7"},
		{name: "CachedOverride", cached: types.Dict{"Version": types.Name("1.6")}, want: "1.6"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newXRefTable(NewStatelessConfiguration())
			x.Root = tt.root
			x.RootDict = tt.cached
			if tt.object != nil {
				x.Table[1] = NewXRefTableEntryGen0(tt.object)
			}
			got, err := x.ParseRootVersion()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("got error %v, want %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if got != nil {
					t.Fatalf("got version %q, want no override", *got)
				}
				return
			}
			if got == nil || *got != tt.want {
				t.Fatalf("got version %v, want %q", got, tt.want)
			}
		})
	}
}

// TestPageRootErrors verifies accessors and page operations reject malformed required roots.
func TestPageRootErrors(t *testing.T) {
	tests := []struct {
		name    string
		root    *types.IndirectRef
		catalog types.Dict
		want    string
	}{
		{name: "MissingCatalog", want: "missing root dict"},
		{name: "UnresolvedCatalog", root: types.NewIndirectRef(1, 0), want: "missing root dict"},
		{name: "MissingPages", catalog: types.Dict{}, want: "missing pages root"},
		{name: "NullPages", catalog: types.Dict{"Pages": nil}, want: "corrupt pages root"},
		{name: "IntegerPages", catalog: types.Dict{"Pages": types.Integer(1)}, want: "corrupt pages root"},
		{name: "DirectPages", catalog: types.Dict{"Pages": types.Dict{}}, want: "corrupt pages root"},
		{name: "ArrayPages", catalog: types.Dict{"Pages": types.Array{}}, want: "corrupt pages root"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := newXRefTable(NewStatelessConfiguration())
			x.Root, x.RootDict = tt.root, tt.catalog
			ref, err := x.Pages()
			if ref != nil || err == nil || err.Error() != tt.want {
				t.Fatalf("Pages: got (%v, %v), want (nil, %s)", ref, err, tt.want)
			}
			if err := x.EnsurePageCount(); err == nil || err.Error() != tt.want {
				t.Fatalf("EnsurePageCount: got %v, want %s", err, tt.want)
			}
			n, err := x.PageNumber(t.Context(), 2)
			if n != 0 || err == nil || err.Error() != tt.want {
				t.Fatalf("PageNumber: got (%d, %v), want (0, %s)", n, err, tt.want)
			}
		})
	}
}

// TestPageRootValidReference verifies a valid page tree supports both page-count and page-number access.
func TestPageRootValidReference(t *testing.T) {
	x := newXRefTable(NewStatelessConfiguration())
	ref := *types.NewIndirectRef(1, 0)
	x.RootDict = types.Dict{"Pages": ref}
	x.Table[1] = NewXRefTableEntryGen0(types.Dict{
		"Type": types.Name("Pages"), "Count": types.Integer(1), "Kids": types.Array{*types.NewIndirectRef(2, 0)},
	})
	x.Table[2] = NewXRefTableEntryGen0(types.Dict{"Type": types.Name("Page"), "Parent": ref})
	got, err := x.Pages()
	if err != nil || got == nil || *got != ref {
		t.Fatalf("Pages: got (%v, %v), want %v without error", got, err, ref)
	}
	if err := x.EnsurePageCount(); err != nil || x.PageCount != 1 {
		t.Fatalf("EnsurePageCount: got (%d, %v), want (1, nil)", x.PageCount, err)
	}
	if n, err := x.PageNumber(t.Context(), 2); err != nil || n != 1 {
		t.Fatalf("PageNumber: got (%d, %v), want (1, nil)", n, err)
	}
}

// TestCatalogAccessPreservesDecodeError verifies required access and optional lookup preserve decoding errors.
func TestCatalogAccessPreservesDecodeError(t *testing.T) {
	want := errors.New("catalog decode failed")
	x := newXRefTable(NewStatelessConfiguration())
	x.Root = types.NewIndirectRef(1, 0)
	osd := &types.ObjectStreamDict{StreamDict: types.StreamDict{Content: []byte("catalog")}}
	object := types.NewLazyObjectStreamObject(osd, 0, -1, func(context.Context, string) (types.Object, error) {
		return nil, want
	})
	x.Table[1] = NewXRefTableEntryGen0(object)
	if _, err := x.Catalog(); err != want {
		t.Fatalf("Catalog: got %v, want original error %v", err, want)
	}
	if _, err := x.Pages(); err != want {
		t.Fatalf("Pages: got %v, want original error %v", err, want)
	}
	if _, err := x.PageNumber(t.Context(), 2); err != want {
		t.Fatalf("PageNumber: got %v, want original error %v", err, want)
	}
	if _, err := x.ParseRootVersion(); err != want {
		t.Fatalf("ParseRootVersion: got %v, want original error %v", err, want)
	}
}

// TestUnresolvedReferenceRemainsNull verifies required catalog checks do not alter generic null semantics.
func TestUnresolvedReferenceRemainsNull(t *testing.T) {
	x := newXRefTable(NewStatelessConfiguration())
	x.Root = types.NewIndirectRef(7, 0)
	if _, err := x.Catalog(); err == nil {
		t.Fatal("Catalog accepted an unresolved root")
	}
	o, _, err := x.indRefToObject(x.Root, true)
	if o != nil || err != nil {
		t.Fatalf("indRefToObject: got (%v, %v), want (nil, nil)", o, err)
	}
	o, err = x.Dereference(*x.Root)
	if o != nil || err != nil {
		t.Fatalf("Dereference: got (%v, %v), want (nil, nil)", o, err)
	}
}

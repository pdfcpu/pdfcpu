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

// TestExtractPagesRejectsInvalidInput verifies stable extraction errors.
func TestExtractPagesRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *model.Context
		pageNrs []int
		wantErr error
	}{
		{
			name:    "missing context",
			pageNrs: []int{1},
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "missing page numbers",
			ctx:     &model.Context{XRefTable: &model.XRefTable{}},
			wantErr: ErrMissingPageNumbers,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{},
			pageNrs: []int{1},
			wantErr: ErrMissingXRefTable,
		},
		{
			name:    "page number below lower bound",
			ctx:     &model.Context{XRefTable: &model.XRefTable{PageCount: 2}},
			pageNrs: []int{0},
			wantErr: ErrInvalidPageNumber,
		},
		{
			name:    "page number above upper bound",
			ctx:     &model.Context{XRefTable: &model.XRefTable{PageCount: 2}},
			pageNrs: []int{3},
			wantErr: ErrInvalidPageNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractPages(tt.ctx, tt.pageNrs, false)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestExtractPagesAddPagesErrorsIncludePageContext(t *testing.T) {
	ctx := &model.Context{XRefTable: &model.XRefTable{PageCount: 1}}

	_, err := ExtractPages(ctx, []int{1}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "extract pages [1]") {
		t.Fatalf("expected extract pages context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "add pages") {
		t.Fatalf("expected add pages context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "page 1") {
		t.Fatalf("expected page context, got %q", err.Error())
	}
}

func TestAddPagesRejectsMissingContexts(t *testing.T) {
	tests := []struct {
		name string
		src  *model.Context
		dest *model.Context
		want string
	}{
		{
			name: "missing source",
			dest: &model.Context{XRefTable: &model.XRefTable{}},
			want: "add pages: source context",
		},
		{
			name: "missing destination",
			src:  &model.Context{XRefTable: &model.XRefTable{}},
			want: "add pages: destination context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddPages(tt.src, tt.dest, []int{1}, false)
			if !errors.Is(err, ErrMissingPDFContext) {
				t.Fatalf("expected %v, got %v", ErrMissingPDFContext, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestAddPagesRejectsMissingDestinationPageTree(t *testing.T) {
	src, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	dest, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	dest.RootDict.Delete("Pages")

	err = AddPages(src, dest, []int{1}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "add pages: missing destination page tree") {
		t.Fatalf("expected missing destination page tree context, got %q", err.Error())
	}
}

func TestAddPagesInternalGuardsPreventPanics(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "missing page tree dict",
			fn: func() error {
				return addPages(ctx, ctx, []int{1}, false, *types.NewIndirectRef(1, 0), nil, &types.Array{}, &types.Array{}, map[int]int{})
			},
			want: "missing destination page tree dict",
		},
		{
			name: "missing source fields",
			fn: func() error {
				return addPages(ctx, ctx, []int{1}, false, *types.NewIndirectRef(1, 0), types.Dict{}, nil, &types.Array{}, map[int]int{})
			},
			want: "missing source form fields",
		},
		{
			name: "missing migration map",
			fn: func() error {
				return addPages(ctx, ctx, []int{1}, false, *types.NewIndirectRef(1, 0), types.Dict{}, &types.Array{}, &types.Array{}, nil)
			},
			want: "missing migration map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected error, got panic: %v", r)
				}
			}()

			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestMigratePageDictErrorsIncludeEntryContext(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	d := types.Dict{
		"Annots": types.Integer(1),
	}

	err = migratePageDict(d, *types.NewIndirectRef(1, 0), ctx, ctx, map[int]int{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "page dict entry Annots") {
		t.Fatalf("expected page dict entry context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "migrate annotations") {
		t.Fatalf("expected migrate annotations context, got %q", err.Error())
	}
}

func TestMigrateNamedDestsErrorsIncludeKeyContext(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}

	n := &model.Node{}
	n.AppendToNames("badDest", types.Integer(1))

	err = migrateNamedDests(ctx, n, map[int]int{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "named destination \"badDest\"") {
		t.Fatalf("expected named destination key context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "process named destinations") {
		t.Fatalf("expected process named destinations context, got %q", err.Error())
	}
}

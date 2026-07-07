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
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func nestedBookmarks(depth int) []Bookmark {
	bms := []Bookmark{{Title: "bookmark", PageFrom: 1}}
	if depth > 0 {
		bms[0].Kids = nestedBookmarks(depth - 1)
	}
	return bms
}

// TestBookmarkListRejectsRecursionDepth verifies bookmark listing respects recursion limits.
func TestBookmarkListRejectsRecursionDepth(t *testing.T) {
	_, err := bookmarkList(nestedBookmarks(2), 0, 1)
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestCreateOutlineItemDictRejectsRecursionDepth verifies outline creation respects recursion limits.
func TestCreateOutlineItemDictRejectsRecursionDepth(t *testing.T) {
	_, _, _, _, err := createOutlineItemDictDepth(nil, nestedBookmarks(0), nil, nil, model.DefaultResourceLimits().MaxRecursionDepth+1)
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestCreateOutlineItemDictRejectsInvalidBookmark verifies outline creation classifies invalid bookmarks.
func TestCreateOutlineItemDictRejectsInvalidBookmark(t *testing.T) {
	parentPageNr := 2
	bms := []Bookmark{{Title: "bookmark", PageFrom: 1}}

	_, _, _, _, err := createOutlineItemDictDepth(nil, bms, nil, &parentPageNr, 0)
	if !errors.Is(err, ErrInvalidBookmark) {
		t.Fatalf("got %v, want ErrInvalidBookmark", err)
	}
}

// TestPageNrFromDestinationReturnsStrictDestinationError verifies strict validation propagates destination errors.
func TestPageNrFromDestinationReturnsStrictDestinationError(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationStrict

	if _, err = PageNrFromDestination(ctx, types.Name("missing")); err == nil {
		t.Fatal("expected destination error")
	}
}

// TestPageNrFromDestinationIgnoresRelaxedDestinationError verifies relaxed validation keeps missing destinations non-fatal.
func TestPageNrFromDestinationIgnoresRelaxedDestinationError(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationRelaxed

	pageNr, err := PageNrFromDestination(ctx, types.Name("missing"))
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	if pageNr != 0 {
		t.Fatalf("got page number %d, want 0", pageNr)
	}
}

func TestPageNrFromDestinationRejectsInvalidInput(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		ctx     *model.Context
		dest    types.Object
		wantErr error
	}{
		{
			name:    "missing context",
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "missing xref table",
			ctx:     &model.Context{},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "empty destination array",
			ctx:  ctx,
			dest: types.Array{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PageNrFromDestination(tt.ctx, tt.dest)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "empty destination array") {
				t.Fatalf("got %v, want empty destination array error", err)
			}
		})
	}
}

// TestAddBookmarksRejectsExistingBookmarks verifies add bookmarks classifies existing bookmarks.
func TestAddBookmarksRejectsExistingBookmarks(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.RootDict = types.Dict{"Outlines": *types.NewIndirectRef(1, 0)}

	err = AddBookmarks(ctx, nestedBookmarks(0), false)
	if !errors.Is(err, ErrExistingBookmarks) {
		t.Fatalf("got %v, want ErrExistingBookmarks", err)
	}
}

func TestBookmarkOperationsRejectInvalidBoundaryInput(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		fn      func() error
		wantErr error
	}{
		{
			name: "bookmarks missing context",
			fn: func() error {
				_, err := Bookmarks(nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "bookmarks missing xref table",
			fn: func() error {
				_, err := Bookmarks(&model.Context{})
				return err
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "bookmark list missing context",
			fn: func() error {
				_, err := BookmarkList(nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "export bookmarks missing context",
			fn: func() error {
				_, err := ExportBookmarks(nil, "")
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "export bookmarks JSON missing context",
			fn: func() error {
				_, err := ExportBookmarksJSON(nil, "", nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "remove bookmarks missing context",
			fn: func() error {
				_, err := RemoveBookmarks(nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "add bookmarks missing context",
			fn: func() error {
				return AddBookmarks(nil, nestedBookmarks(0), false)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "add bookmarks empty input",
			fn: func() error {
				return AddBookmarks(ctx, nil, false)
			},
			wantErr: ErrInvalidBookmark,
		},
		{
			name: "import bookmarks missing context",
			fn: func() error {
				_, err := ImportBookmarks(nil, strings.NewReader("{}"), false)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestImportBookmarksRejectsMissingAndEmptyJSONInput(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ImportBookmarks(ctx, nil, false); err == nil || !strings.Contains(err.Error(), "missing bookmark JSON reader") {
		t.Fatalf("got %v, want missing bookmark JSON reader", err)
	}

	_, err = ImportBookmarks(ctx, strings.NewReader(`{"bookmarks":[]}`), false)
	if !errors.Is(err, ErrInvalidBookmark) {
		t.Fatalf("got %v, want ErrInvalidBookmark", err)
	}
}

// TestParseBookmarksFromJSONClassifiesInvalidJSON verifies bookmark JSON failures are branchable.
func TestParseBookmarksFromJSONClassifiesInvalidJSON(t *testing.T) {
	tests := []struct {
		name         string
		bb           []byte
		wantTypeFail bool
	}{
		{
			name: "invalid encoding",
			bb:   []byte("{"),
		},
		{
			name:         "invalid field type",
			bb:           []byte(`{"bookmarks":[{"title":1}]}`),
			wantTypeFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseBookmarksFromJSON(tt.bb)
			if !errors.Is(err, ErrInvalidBookmarkJSON) {
				t.Fatalf("got %v, want ErrInvalidBookmarkJSON", err)
			}

			var typeErr *json.UnmarshalTypeError
			if got := errors.As(err, &typeErr); got != tt.wantTypeFail {
				t.Fatalf("got type error match %t, want %t", got, tt.wantTypeFail)
			}
		})
	}
}

func cyclicBookmarkContext(t *testing.T) *model.Context {
	t.Helper()

	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}

	dest := types.Array{types.Integer(1)}
	d := types.Dict{
		"Title": types.StringLiteral("bookmark"),
		"Dest":  dest,
		"Next":  *types.NewIndirectRef(1, 0),
	}
	if _, err := ctx.IndRefForObject(1, d); err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestBookmarksForOutlineItemRejectsCycle verifies outline traversal rejects cycles.
func TestBookmarksForOutlineItemRejectsCycle(t *testing.T) {
	ctx := cyclicBookmarkContext(t)

	_, err := BookmarksForOutlineItem(ctx, types.NewIndirectRef(1, 0), nil)
	if !errors.Is(err, ErrCircularBookmarks) {
		t.Fatalf("got %v, want ErrCircularBookmarks", err)
	}
}

// TestRemoveNamedDestsRejectsCycle verifies named destination removal rejects cycles.
func TestRemoveNamedDestsRejectsCycle(t *testing.T) {
	ctx := cyclicBookmarkContext(t)

	err := removeNamedDests(ctx, types.NewIndirectRef(1, 0), 0, map[int]bool{})
	if !errors.Is(err, ErrCircularBookmarks) {
		t.Fatalf("got %v, want ErrCircularBookmarks", err)
	}
}

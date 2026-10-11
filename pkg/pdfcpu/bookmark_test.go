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
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	_, err := bookmarkList(t.Context(), nestedBookmarks(2), 0, 1)
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestCreateOutlineItemDictRejectsRecursionDepth verifies outline creation respects recursion limits.
func TestCreateOutlineItemDictRejectsRecursionDepth(t *testing.T) {
	_, _, _, _, err := createOutlineItemDictDepth(t.Context(), nil, nestedBookmarks(0), nil, nil, model.DefaultResourceLimits().MaxRecursionDepth+1)
	if !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestCreateOutlineItemDictRejectsInvalidBookmark verifies outline creation classifies invalid bookmarks.
func TestCreateOutlineItemDictRejectsInvalidBookmark(t *testing.T) {
	parentPageNr := 2
	bms := []Bookmark{{Title: "bookmark", PageFrom: 1}}

	_, _, _, _, err := createOutlineItemDictDepth(t.Context(), nil, bms, nil, &parentPageNr, 0)
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

	if _, err = PageNrFromDestination(t.Context(), ctx, types.Name("missing")); err == nil {
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

	pageNr, err := PageNrFromDestination(t.Context(), ctx, types.Name("missing"))
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	if pageNr != 0 {
		t.Fatalf("got page number %d, want 0", pageNr)
	}
}

func TestPageNrFromDestinationResolvesIndirectInteger(t *testing.T) {
	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.Table[7] = model.NewXRefTableEntryGen0(types.Integer(4))
	indRef := *types.NewIndirectRef(7, 0)
	dest := types.Array{indRef, types.Name("Fit")}

	pageNr, err := PageNrFromDestination(t.Context(), ctx, dest)
	if err != nil {
		t.Fatal(err)
	}
	if pageNr != 4 {
		t.Fatalf("page number = %d, want 4", pageNr)
	}
	if dest[0] != indRef {
		t.Fatalf("destination was normalized in place: %v", dest)
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
			_, err := PageNrFromDestination(t.Context(), tt.ctx, tt.dest)
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

	err = AddBookmarks(t.Context(), ctx, nestedBookmarks(0), false)
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
				_, err := Bookmarks(t.Context(), nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "bookmarks missing xref table",
			fn: func() error {
				_, err := Bookmarks(t.Context(), &model.Context{})
				return err
			},
			wantErr: ErrMissingXRefTable,
		},
		{
			name: "bookmark list missing context",
			fn: func() error {
				_, err := BookmarkList(t.Context(), nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "export bookmarks missing context",
			fn: func() error {
				_, err := ExportBookmarks(t.Context(), nil, "")
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "export bookmarks JSON missing context",
			fn: func() error {
				_, err := ExportBookmarksJSON(t.Context(), nil, "", nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "remove bookmarks missing context",
			fn: func() error {
				_, err := RemoveBookmarks(t.Context(), nil)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "add bookmarks missing context",
			fn: func() error {
				return AddBookmarks(t.Context(), nil, nestedBookmarks(0), false)
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "add bookmarks empty input",
			fn: func() error {
				return AddBookmarks(t.Context(), ctx, nil, false)
			},
			wantErr: ErrInvalidBookmark,
		},
		{
			name: "import bookmarks missing context",
			fn: func() error {
				_, err := ImportBookmarks(t.Context(), nil, strings.NewReader("{}"), false)
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

	if _, err := ImportBookmarks(t.Context(), ctx, nil, false); err == nil || !strings.Contains(err.Error(), "missing bookmark JSON reader") {
		t.Fatalf("got %v, want missing bookmark JSON reader", err)
	}

	_, err = ImportBookmarks(t.Context(), ctx, strings.NewReader(`{"bookmarks":[]}`), false)
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

	_, err := BookmarksForOutlineItem(t.Context(), ctx, types.NewIndirectRef(1, 0), nil)
	if !errors.Is(err, ErrCircularBookmarks) {
		t.Fatalf("got %v, want ErrCircularBookmarks", err)
	}
}

// TestRemoveBookmarksRejectsCycle verifies bookmark removal rejects cycles.
func TestRemoveBookmarksRejectsCycle(t *testing.T) {
	ctx := cyclicBookmarkContext(t)
	ctx.Outlines = types.Dict{"First": *types.NewIndirectRef(1, 0)}

	_, err := RemoveBookmarks(t.Context(), ctx)
	if !errors.Is(err, ErrCircularBookmarks) {
		t.Fatalf("got %v, want ErrCircularBookmarks", err)
	}
}

func bookmarkDestinationTestContext(t *testing.T, legacy, link bool) (*model.Context, types.Dict) {
	t.Helper()
	ctx, err := CreateContextWithXRefTable(model.NewStatelessConfiguration(), types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	appendExtractionTestPage(t, ctx)
	appendExtractionTestPage(t, ctx)
	if err := AddBookmarks(t.Context(), ctx, []Bookmark{{Title: "shared", PageFrom: 1}}, false); err != nil {
		t.Fatal(err)
	}
	ctx.Outlines, err = ctx.DereferenceDict(ctx.RootDict["Outlines"])
	if err != nil {
		t.Fatal(err)
	}
	var name types.Object = types.StringLiteral("shared")
	if legacy {
		dest, _, err := ctx.Names["Dests"].Value(t.Context(), "shared")
		if err != nil {
			t.Fatal(err)
		}
		ctx.Dests = types.Dict{"shared": dest}
		ctx.RootDict["Dests"] = ctx.Dests
		delete(ctx.Names, "Dests")
		delete(ctx.RootDict, "Names")
		name = types.Name("shared")
		d, err := ctx.DereferenceDict(*ctx.Outlines.IndirectRefEntry("First"))
		if err != nil {
			t.Fatal(err)
		}
		d["Dest"] = name
	}
	annotation := types.Dict{"Type": types.Name("Annot"), "Subtype": types.Name("Link"), "Dest": name}
	if link {
		page, _, _, err := ctx.PageDict(t.Context(), 1, false)
		if err != nil {
			t.Fatal(err)
		}
		page["Annots"] = types.Array{annotation}
	}
	return ctx, annotation
}

// TestBookmarkChangesPreserveNamedDestinations checks removal and replacement retain document-level targets.
func TestBookmarkChangesPreserveNamedDestinations(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		for _, link := range []bool{false, true} {
			for _, replace := range []bool{false, true} {
				t.Run(fmt.Sprintf("legacy=%t/link=%t/replace=%t", legacy, link, replace), func(t *testing.T) {
					ctx, annotation := bookmarkDestinationTestContext(t, legacy, link)
					ctx.XRefTable.ValidationMode = model.ValidationStrict
					if replace {
						if err := AddBookmarks(t.Context(), ctx, []Bookmark{{Title: "shared", PageFrom: 2}}, true); err != nil {
							t.Fatal(err)
						}
					} else if removed, err := RemoveBookmarks(t.Context(), ctx); err != nil || !removed {
						t.Fatalf("remove: %t, %v", removed, err)
					}
					page, err := PageNrFromDestination(t.Context(), ctx, annotation["Dest"])
					if err != nil || page != 1 {
						t.Fatalf("original target: page %d, %v; want page 1", page, err)
					}
					checkBookmarkChange(t, ctx, replace)
				})
			}
		}
	}
}

func checkBookmarkChange(t *testing.T, ctx *model.Context, replace bool) {
	t.Helper()
	bms, err := Bookmarks(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if replace {
		if len(bms) != 1 || bms[0].PageFrom != 2 {
			t.Fatalf("replacement bookmarks: %v", bms)
		}
	} else if len(bms) != 0 || ctx.RootDict["Outlines"] != nil {
		t.Fatal("outlines remain after removal")
	}
}

// TestBookmarkReplacementAvoidsLegacyNameCollision checks generated names against both destination stores.
func TestBookmarkReplacementAvoidsLegacyNameCollision(t *testing.T) {
	ctx, _ := bookmarkDestinationTestContext(t, false, true)
	dest, _, err := ctx.Names["Dests"].Value(t.Context(), "shared")
	if err != nil {
		t.Fatal(err)
	}
	ctx.Dests = types.Dict{"shared\x01": dest}
	ctx.RootDict["Dests"] = ctx.Dests
	if err := AddBookmarks(t.Context(), ctx, []Bookmark{{Title: "shared", PageFrom: 2}}, true); err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	for _, name := range []types.Object{types.StringLiteral("shared"), types.Name("shared\x01")} {
		page, err := PageNrFromDestination(t.Context(), ctx, name)
		if err != nil || page != 1 {
			t.Fatalf("original target %v: page %d, %v; want page 1", name, page, err)
		}
	}
	checkBookmarkChange(t, ctx, true)
}

// TestRemoveBookmarksTraversalChecks retains cycle, depth, and cancellation checks without changing destinations.
func TestRemoveBookmarksTraversalChecks(t *testing.T) {
	for _, tc := range []struct {
		name string
		want error
	}{
		{"child cycle", ErrCircularBookmarks},
		{"depth", model.ErrMaxRecursionDepthExceeded},
		{"canceled", context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, annotation := bookmarkDestinationTestContext(t, false, true)
			first := ctx.Outlines.IndirectRefEntry("First")
			d, err := ctx.DereferenceDict(*first)
			if err != nil {
				t.Fatal(err)
			}
			c := t.Context()
			switch tc.name {
			case "child cycle":
				d["First"] = *first
			case "depth":
				ctx.Configuration.Limits.MaxRecursionDepth = 1
				for range 2 {
					child := types.Dict{"Dest": types.StringLiteral("shared")}
					ir, err := ctx.IndRefForNewObject(child)
					if err != nil {
						t.Fatal(err)
					}
					d["First"] = *ir
					d = child
				}
			case "canceled":
				var cancel context.CancelFunc
				c, cancel = context.WithCancel(c)
				cancel()
			}
			if _, err := RemoveBookmarks(c, ctx); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			page, err := PageNrFromDestination(t.Context(), ctx, annotation["Dest"])
			if err != nil || page != 1 {
				t.Fatalf("destination changed on failure: page %d, %v", page, err)
			}
		})
	}
}

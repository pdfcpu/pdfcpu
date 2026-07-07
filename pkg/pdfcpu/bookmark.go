/*
	Copyright 2020 The pdfcpu Authors.

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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var (
	// ErrNoBookmarks signals that a PDF has no bookmarks to process.
	ErrNoBookmarks = errors.New("no bookmarks available")

	// ErrInvalidBookmark signals an invalid bookmark tree.
	ErrInvalidBookmark = errors.New("invalid bookmark")

	// ErrInvalidBookmarkJSON signals malformed bookmark JSON data.
	ErrInvalidBookmarkJSON = errors.New("invalid bookmark JSON")

	// ErrExistingBookmarks signals that adding bookmarks would conflict with existing bookmarks.
	ErrExistingBookmarks = errors.New("existing bookmarks")

	// ErrCircularBookmarks signals a circular outline item list.
	ErrCircularBookmarks = errors.New("circular outline item list")

	errMissingBookmarkJSONReader = errors.New("missing bookmark JSON reader")
)

func validateBookmarkContext(ctx *model.Context) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}
	return nil
}

// Header contains metadata written to exported bookmark JSON.
type Header struct {
	Source   string   `json:"source,omitempty"`
	Version  string   `json:"version"`
	Creation string   `json:"creation"`
	ID       []string `json:"id,omitempty"`
	Title    string   `json:"title,omitempty"`
	Author   string   `json:"author,omitempty"`
	Creator  string   `json:"creator,omitempty"`
	Producer string   `json:"producer,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	Keywords string   `json:"keywords,omitempty"`
}

// Bookmark represents a PDF bookmark backed by an outline item.
type Bookmark struct {
	Title    string             `json:"title"`
	PageFrom int                `json:"page"`
	PageThru int                `json:"-"` // for extraction only; >= pageFrom and reaches until before pageFrom of the next bookmark.
	Bold     bool               `json:"bold,omitempty"`
	Italic   bool               `json:"italic,omitempty"`
	Color    *color.SimpleColor `json:"color,omitempty"`
	Kids     []Bookmark         `json:"kids,omitempty"`
	Parent   *Bookmark          `json:"-"`
}

// BookmarkTree contains exported bookmark metadata and the bookmark hierarchy.
type BookmarkTree struct {
	Header    Header     `json:"header"`
	Bookmarks []Bookmark `json:"bookmarks"`
}

func header(xRefTable *model.XRefTable, source string) Header {
	h := Header{}
	h.Source = filepath.Base(source)
	h.Version = "pdfcpu " + model.VersionStr
	h.Creation = time.Now().Format("2006-01-02 15:04:05 MST")
	h.ID = []string{}
	h.Title = xRefTable.Title
	h.Author = xRefTable.Author
	h.Creator = xRefTable.Creator
	h.Producer = xRefTable.Producer
	h.Subject = xRefTable.Subject
	h.Keywords = xRefTable.Keywords
	return h
}

// Style returns an int corresponding to the bookmark style.
func (bm Bookmark) Style() int {
	var i int
	if bm.Bold { // bit 1
		i += 2
	}
	if bm.Italic { // bit 0
		i += 1
	}
	return i
}

func positionToFirstBookmark(ctx *model.Context) (*types.IndirectRef, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return nil, err
	}

	d := ctx.Outlines
	if d == nil {
		return nil, ErrNoBookmarks
	}
	return d.IndirectRefEntry("First"), nil
}

func outlineItemTitle(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 32 {
			sb.WriteByte(b)
		}
	}
	return sb.String()
}

func destArray(ctx *model.Context, dest types.Object) (types.Array, error) {
	switch dest := dest.(type) {
	case types.Name:
		return ctx.DereferenceDestArray(dest.Value())
	case types.StringLiteral:
		s, err := types.StringLiteralToString(dest)
		if err != nil {
			return nil, err
		}
		return ctx.DereferenceDestArray(s)
	case types.HexLiteral:
		s, err := types.HexLiteralToString(dest)
		if err != nil {
			return nil, err
		}
		return ctx.DereferenceDestArray(s)
	case types.Array:
		return dest, nil
	}
	return nil, fmt.Errorf("unable to resolve destination array %v", dest)
}

// PageNrFromDestination returns the page number of a destination.
func PageNrFromDestination(ctx *model.Context, dest types.Object) (int, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return 0, err
	}

	arr, err := destArray(ctx, dest)
	if err != nil {
		if ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
			return 0, nil
		}
		return 0, fmt.Errorf("resolve destination page: %w", err)
	}
	if len(arr) == 0 {
		return 0, fmt.Errorf("resolve destination page: empty destination array %v", dest)
	}

	if i, ok := arr[0].(types.Integer); ok {
		return i.Value(), nil
	}

	if ir, ok := arr[0].(types.IndirectRef); ok {
		return ctx.PageNumber(ir.ObjectNumber.Value())
	}

	return 0, fmt.Errorf("resolve destination page: unable to extract page number from %v", dest)
}

func title(ctx *model.Context, d types.Dict) (string, error) {
	obj, err := ctx.Dereference(d["Title"])
	if err != nil {
		return "", err
	}

	s, err := model.Text(obj)
	if err != nil {
		if ctx.XRefTable.ValidationMode == model.ValidationStrict {
			return "", err
		}
		return "", nil
	}

	return outlineItemTitle(s), nil
}

func bookmark(d types.Dict, title string, pageFrom int, parent *Bookmark) Bookmark {
	bm := Bookmark{
		Title:    title,
		PageFrom: pageFrom,
		Parent:   parent,
		Bold:     false,
		Italic:   false,
	}

	if arr := d.ArrayEntry("C"); len(arr) == 3 {
		col := color.NewSimpleColorForArray(arr)
		bm.Color = &col
	}

	if f := d.IntEntry("F"); f != nil {
		bm.Bold = *f&0x02 > 0
		bm.Italic = *f&0x01 > 0
	}

	return bm
}

func checkBookmarkRecursionDepth(ctx *model.Context, name string, depth int) error {
	if ctx == nil || ctx.XRefTable == nil {
		return model.CheckRecursionDepth(name, depth, 0)
	}
	return ctx.XRefTable.CheckRecursionDepth(name, depth)
}

func checkBookmarkCycle(ir *types.IndirectRef, visited map[int]bool) error {
	objNr := ir.ObjectNumber.Value()
	if visited[objNr] {
		return fmt.Errorf("obj#%d: %w", objNr, ErrCircularBookmarks)
	}
	visited[objNr] = true
	return nil
}

func outlineItemDict(ctx *model.Context, ir *types.IndirectRef, visited map[int]bool) (types.Dict, error) {
	if err := checkBookmarkCycle(ir, visited); err != nil {
		return nil, err
	}

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return nil, fmt.Errorf("outline item %s: dereference dict: %w", *ir, err)
	}
	return d, nil
}

func bookmarksForOutlineItem(ctx *model.Context, item *types.IndirectRef, parent *Bookmark, depth int, visited map[int]bool) ([]Bookmark, error) {
	if err := checkBookmarkRecursionDepth(ctx, "outline item", depth); err != nil {
		return nil, fmt.Errorf("outline item depth %d: %w", depth, err)
	}

	bms := []Bookmark{}

	var (
		d   types.Dict
		err error
	)

	// Process outline items.
	for ir := item; ir != nil; ir = d.IndirectRefEntry("Next") {

		if d, err = outlineItemDict(ctx, ir, visited); err != nil {
			return nil, err
		}

		title, err := title(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("outline item %s title: %w", *ir, err)
		}

		if title == "" {
			continue
		}

		dest, ok, err := outlineItemDestination(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("outline item %s action: %w", *ir, err)
		}
		if !ok {
			continue
		}

		obj, err := ctx.Dereference(dest)
		if err != nil {
			return nil, fmt.Errorf("outline item %s destination: %w", *ir, err)
		}

		pageFrom, err := PageNrFromDestination(ctx, obj)
		if err != nil {
			return nil, fmt.Errorf("outline item %s destination page: %w", *ir, err)
		}

		if len(bms) > 0 {
			if pageFrom > bms[len(bms)-1].PageFrom {
				bms[len(bms)-1].PageThru = pageFrom - 1
			} else {
				bms[len(bms)-1].PageThru = bms[len(bms)-1].PageFrom
			}
		}

		bm := bookmark(d, title, pageFrom, parent)

		first := d["First"]
		if first != nil {
			indRef, ok := first.(types.IndirectRef)
			if !ok {
				return nil, fmt.Errorf("outline item %s first kid: expected indirect reference, got %T", *ir, first)
			}
			kids, err := bookmarksForOutlineItem(ctx, &indRef, &bm, depth+1, visited)
			if err != nil {
				return nil, fmt.Errorf("outline item %s kids: %w", *ir, err)
			}
			bm.Kids = kids
		}

		bms = append(bms, bm)
	}

	return bms, nil
}

func outlineItemDestination(ctx *model.Context, d types.Dict) (types.Object, bool, error) {
	dest, found := d["Dest"]
	if found {
		return dest, true, nil
	}

	act, found := d["A"]
	if !found {
		return nil, false, nil
	}

	act, err := ctx.Dereference(act)
	if err != nil {
		return nil, false, err
	}

	actionDict, ok := act.(types.Dict)
	if !ok {
		return nil, false, nil
	}

	actType := actionDict["S"]
	if actType == nil || actType.String() != "GoTo" {
		return nil, false, nil
	}

	dest, found = actionDict["D"]
	return dest, found, nil
}

// BookmarksForOutlineItem returns the bookmarks tree for an outline item.
func BookmarksForOutlineItem(ctx *model.Context, item *types.IndirectRef, parent *Bookmark) ([]Bookmark, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrNoBookmarks
	}

	return bookmarksForOutlineItem(ctx, item, parent, 0, map[int]bool{})
}

// Bookmarks returns all bookmark information in ctx recursively.
func Bookmarks(ctx *model.Context) ([]Bookmark, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return nil, err
	}

	if err := ctx.LocateNameTree("Dests", false); err != nil {
		return nil, fmt.Errorf("locate destinations name tree: %w", err)
	}

	first, err := positionToFirstBookmark(ctx)
	if err != nil {
		if !errors.Is(err, ErrNoBookmarks) {
			return nil, err
		}
		return []Bookmark{}, nil
	}
	if first == nil {
		return []Bookmark{}, nil
	}

	bms, err := BookmarksForOutlineItem(ctx, first, nil)
	if err != nil {
		return nil, fmt.Errorf("read bookmark tree: %w", err)
	}
	return bms, nil
}

func bookmarkList(bms []Bookmark, level, maxDepth int) ([]string, error) {
	if err := model.CheckRecursionDepth("bookmark list", level, maxDepth); err != nil {
		return nil, fmt.Errorf("bookmark list level %d: %w", level, err)
	}

	pre := strings.Repeat("    ", level)
	ss := []string{}
	for _, bm := range bms {
		ss = append(ss, pre+bm.Title)
		if len(bm.Kids) > 0 {
			ss1, err := bookmarkList(bm.Kids, level+1, maxDepth)
			if err != nil {
				return nil, fmt.Errorf("bookmark %q kids: %w", bm.Title, err)
			}
			ss = append(ss, ss1...)
		}
	}
	return ss, nil
}

// BookmarkList returns a formatted bookmark list for ctx.
func BookmarkList(ctx *model.Context) ([]string, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return nil, err
	}

	bms, err := Bookmarks(ctx)
	if err != nil {
		return nil, err
	}

	if len(bms) == 0 {
		return []string{"no bookmarks available"}, nil
	}

	return bookmarkList(bms, 0, ctx.XRefTable.MaxRecursionDepth())
}

// ExportBookmarks returns the bookmark tree for ctx.
func ExportBookmarks(ctx *model.Context, source string) (*BookmarkTree, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return nil, err
	}

	bms, err := Bookmarks(ctx)
	if err != nil {
		return nil, err
	}
	if len(bms) == 0 {
		return nil, nil
	}

	bmTree := BookmarkTree{}
	bmTree.Header = header(ctx.XRefTable, source)
	bmTree.Bookmarks = bms

	return &bmTree, nil
}

// ExportBookmarksJSON writes the bookmark tree for ctx as JSON.
func ExportBookmarksJSON(ctx *model.Context, source string, w io.Writer) (bool, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return false, err
	}
	if w == nil {
		return false, errors.New("missing bookmark JSON writer")
	}

	bookmarkTree, err := ExportBookmarks(ctx, source)
	if err != nil || bookmarkTree == nil {
		return false, err
	}

	bb, err := json.MarshalIndent(bookmarkTree, "", "\t")
	if err != nil {
		return false, fmt.Errorf("encode bookmark JSON: %w", err)
	}

	if _, err = w.Write(bb); err != nil {
		return false, fmt.Errorf("write bookmark JSON: %w", err)
	}

	return true, nil
}

func bmDict(ctx *model.Context, bm Bookmark, parent types.IndirectRef) (types.Dict, error) {
	_, pageIndRef, _, err := ctx.PageDict(bm.PageFrom, false)
	if err != nil {
		return nil, fmt.Errorf("bookmark %q page %d: page dict: %w", bm.Title, bm.PageFrom, err)
	}

	arr := types.Array{*pageIndRef, types.Name("Fit")}
	ir, err := ctx.IndRefForNewObject(arr)
	if err != nil {
		return nil, fmt.Errorf("bookmark %q page %d: create destination: %w", bm.Title, bm.PageFrom, err)
	}

	var o types.Object = *ir

	s, err := types.EscapedUTF16String(bm.Title)
	if err != nil {
		return nil, fmt.Errorf("bookmark %q: encode title: %w", bm.Title, err)
	}

	d := types.Dict(map[string]types.Object{
		"Dest":   types.NewHexLiteral([]byte(bm.Title)),
		"Title":  types.StringLiteral(*s),
		"Parent": parent},
	)

	m := model.NameMap{bm.Title: []types.Dict{d}}
	if err := ctx.Names["Dests"].Add(ctx.XRefTable, bm.Title, o, m, []string{"D", "Dest"}); err != nil {
		return nil, fmt.Errorf("bookmark %q: add named destination: %w", bm.Title, err)
	}

	if bm.Color != nil {
		d["C"] = types.Array{types.Float(bm.Color.R), types.Float(bm.Color.G), types.Float(bm.Color.B)}
	}

	if style := bm.Style(); style > 0 {
		d["F"] = types.Integer(style)
	}

	return d, nil
}

func invalidBookmark(i int, bm Bookmark, bms []Bookmark, parentPageNr *int) bool {
	if i == 0 && parentPageNr != nil && bm.PageFrom < *parentPageNr {
		return true
	}

	return i > 0 && bm.PageFrom < bms[i-1].PageFrom
}

func createOutlineItemDictDepth(ctx *model.Context, bms []Bookmark, parent *types.IndirectRef, parentPageNr *int, depth int) (*types.IndirectRef, *types.IndirectRef, int, int, error) {
	if err := checkBookmarkRecursionDepth(ctx, "bookmark import", depth); err != nil {
		return nil, nil, 0, 0, fmt.Errorf("bookmark import depth %d: %w", depth, err)
	}

	var (
		first   *types.IndirectRef
		irPrev  *types.IndirectRef
		dPrev   types.Dict
		total   int
		visible int
	)

	for i, bm := range bms {

		if invalidBookmark(i, bm, bms, parentPageNr) {
			return nil, nil, 0, 0, fmt.Errorf("bookmark %d page %d: %w", i, bm.PageFrom, ErrInvalidBookmark)
		}

		total++

		d, err := bmDict(ctx, bm, *parent)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("bookmark %d: %w", i, err)
		}

		ir, err := ctx.IndRefForNewObject(d)
		if err != nil {
			return nil, nil, 0, 0, fmt.Errorf("bookmark %d %q: create outline item: %w", i, bm.Title, err)
		}

		if first == nil {
			first = ir
		}

		if len(bm.Kids) > 0 {

			first, last, c, visc, err := createOutlineItemDictDepth(ctx, bm.Kids, ir, &bm.PageFrom, depth+1)
			if err != nil {
				return nil, nil, 0, 0, fmt.Errorf("bookmark %d %q kids: %w", i, bm.Title, err)
			}

			d["First"] = *first
			d["Last"] = *last

			if visc == 0 {
				d["Count"] = types.Integer(c)
				total += c
			}

			if visc > 0 {
				d["Count"] = types.Integer(c + visc)
				total += c
				visible += visc
			}

		}

		if irPrev != nil {
			d["Prev"] = *irPrev
			dPrev["Next"] = *ir
		}

		dPrev = d
		irPrev = ir

	}

	return first, irPrev, total, visible, nil
}

func createOutlineItemDict(ctx *model.Context, bms []Bookmark, parent *types.IndirectRef, parentPageNr *int) (*types.IndirectRef, *types.IndirectRef, int, int, error) {
	return createOutlineItemDictDepth(ctx, bms, parent, parentPageNr, 0)
}

func cleanupDestinations(ctx *model.Context, dNamesEmpty bool) error {
	if dNamesEmpty {
		delete(ctx.Names, "Dests")
		if err := ctx.RemoveNameTree("Dests"); err != nil {
			return fmt.Errorf("remove destinations name tree: %w", err)
		}
	}

	if ctx.Dests != nil && len(ctx.Dests) == 0 {
		delete(ctx.RootDict, "Dests")
	}

	return nil
}

func removeDest(ctx *model.Context, name string) (bool, bool, error) {
	var (
		dNamesEmpty, ok bool
		err             error
	)
	if dNames := ctx.Names["Dests"]; dNames != nil {
		// Remove destName from dest nametree.
		dNamesEmpty, ok, err = dNames.Remove(ctx.XRefTable, name)
		if err != nil {
			return false, false, fmt.Errorf("remove destination %q from name tree: %w", name, err)
		}
	}

	if !ok {
		if ctx.Dests != nil {
			// Remove destName from named destinations.
			ok = ctx.Dests.Delete(name) != nil
		}
	}

	return dNamesEmpty, ok, err
}

func removeNamedDestForOutlineItem(ctx *model.Context, d types.Dict, ir *types.IndirectRef, depth int, visited map[int]bool) (bool, bool, error) {
	dest, destFound, err := outlineItemDestination(ctx, d)
	if err != nil {
		return false, false, fmt.Errorf("bookmark destination %s action: %w", *ir, err)
	}
	if !destFound {
		return false, false, nil
	}

	s, err := ctx.DestName(dest)
	if err != nil {
		return false, false, fmt.Errorf("bookmark destination %s: resolve destination name: %w", *ir, err)
	}

	if len(s) == 0 {
		return false, false, nil
	}

	dNamesEmpty, ok, err := removeDest(ctx, s)
	if err != nil {
		return false, false, fmt.Errorf("bookmark destination %s: %w", *ir, err)
	}
	if !ok {
		if log.DebugEnabled() {
			log.Debug.Printf("unable to remove bookmark destination name: %s\n", s)
		}
	}

	first := d["First"]
	if first != nil {
		indRef, ok := first.(types.IndirectRef)
		if !ok {
			return false, false, fmt.Errorf("bookmark destination %s first kid: expected indirect reference, got %T", *ir, first)
		}
		if err := removeNamedDests(ctx, &indRef, depth+1, visited); err != nil {
			return false, false, fmt.Errorf("bookmark destination %s kids: %w", *ir, err)
		}
	}

	return dNamesEmpty, true, nil
}

func removeNamedDests(ctx *model.Context, item *types.IndirectRef, depth int, visited map[int]bool) error {
	if err := checkBookmarkRecursionDepth(ctx, "bookmark destinations", depth); err != nil {
		return fmt.Errorf("bookmark destinations depth %d: %w", depth, err)
	}

	var (
		d           types.Dict
		err         error
		dNamesEmpty bool
	)
	for ir := item; ir != nil; ir = d.IndirectRefEntry("Next") {

		if err := checkBookmarkCycle(ir, visited); err != nil {
			return fmt.Errorf("bookmark destination %s: %w", *ir, err)
		}

		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return fmt.Errorf("bookmark destination %s: dereference outline item: %w", *ir, err)
		}

		dNamesEmpty1, dNamesChanged, err := removeNamedDestForOutlineItem(ctx, d, ir, depth, visited)
		if err != nil {
			return err
		}
		if !dNamesChanged {
			continue
		}
		dNamesEmpty = dNamesEmpty1
	}

	if err := cleanupDestinations(ctx, dNamesEmpty); err != nil {
		return fmt.Errorf("cleanup bookmark destinations: %w", err)
	}
	return nil
}

// RemoveBookmarks erases all bookmarks from ctx.
func RemoveBookmarks(ctx *model.Context) (bool, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return false, err
	}

	first, err := positionToFirstBookmark(ctx)
	if err != nil {
		if !errors.Is(err, ErrNoBookmarks) {
			return false, err
		}
		return false, nil
	}
	if first == nil {
		return false, nil
	}

	if err := removeNamedDests(ctx, first, 0, map[int]bool{}); err != nil {
		return false, err
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return false, fmt.Errorf("catalog: %w", err)
	}

	rootDict["Outlines"] = nil

	return true, nil
}

// AddBookmarks adds bookmarks to ctx.
func AddBookmarks(ctx *model.Context, bms []Bookmark, replace bool) error {
	if err := validateBookmarkContext(ctx); err != nil {
		return err
	}
	if len(bms) == 0 {
		return ErrInvalidBookmark
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return fmt.Errorf("catalog: %w", err)
	}

	if !replace {
		if _, ok := rootDict.Find("Outlines"); ok {
			return ErrExistingBookmarks
		}
	}

	if _, err = RemoveBookmarks(ctx); err != nil {
		return fmt.Errorf("remove existing bookmarks: %w", err)
	}

	if err := ctx.LocateNameTree("Dests", true); err != nil {
		return fmt.Errorf("locate destinations name tree: %w", err)
	}

	outlinesDict := types.Dict(map[string]types.Object{"Type": types.Name("Outlines")})
	outlinesir, err := ctx.IndRefForNewObject(outlinesDict)
	if err != nil {
		return fmt.Errorf("create outlines dict: %w", err)
	}

	first, last, total, visible, err := createOutlineItemDict(ctx, bms, outlinesir, nil)
	if err != nil {
		return fmt.Errorf("create outline items: %w", err)
	}
	if first == nil || last == nil {
		return ErrInvalidBookmark
	}

	outlinesDict["First"] = *first
	outlinesDict["Last"] = *last
	outlinesDict["Count"] = types.Integer(total + visible)

	rootDict["Outlines"] = *outlinesir

	return nil
}

func addBookmarkTree(ctx *model.Context, bmTree *BookmarkTree, replace bool) error {
	if bmTree == nil {
		return ErrInvalidBookmarkJSON
	}
	return AddBookmarks(ctx, bmTree.Bookmarks, replace)
}

func parseBookmarksFromJSON(bb []byte) (*BookmarkTree, error) {
	if !json.Valid(bb) {
		return nil, fmt.Errorf("%w: invalid JSON encoding", ErrInvalidBookmarkJSON)
	}

	bmTree := &BookmarkTree{}

	if err := json.Unmarshal(bb, bmTree); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBookmarkJSON, err)
	}

	return bmTree, nil
}

// ImportBookmarks creates/replaces bookmarks in ctx as provided by rd.
func ImportBookmarks(ctx *model.Context, rd io.Reader, replace bool) (bool, error) {
	if err := validateBookmarkContext(ctx); err != nil {
		return false, err
	}
	if rd == nil {
		return false, errMissingBookmarkJSONReader
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return false, fmt.Errorf("read bookmark JSON: %w", err)
	}

	bmTree, err := parseBookmarksFromJSON(buf.Bytes())
	if err != nil {
		return false, fmt.Errorf("parse bookmark JSON: %w", err)
	}

	err = addBookmarkTree(ctx, bmTree, replace)
	if err != nil {
		if errors.Is(err, ErrExistingBookmarks) {
			return false, nil
		}
		return true, err
	}

	return true, nil
}

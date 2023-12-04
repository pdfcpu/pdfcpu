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
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var (
	errNoBookmarks        = errors.New("pdfcpu: no bookmarks available")
	errCorruptedBookmarks = errors.New("pdfcpu: corrupt bookmark")
	errExistingBookmarks  = errors.New("pdfcpu: existing bookmarks")
)

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

// Bookmark represents an outline item tree.
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

func positionToFirstBookmark(ctx *model.Context) (types.Dict, *types.IndirectRef, error) {

	// Position to first bookmark on top most level with more than 1 bookmarks.
	// Default to top most single bookmark level.

	d := ctx.Outlines
	if d == nil {
		return nil, nil, errNoBookmarks
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	var err error

	for first != nil && last != nil && *first == *last {
		if d, err = ctx.DereferenceDict(*first); err != nil {
			return nil, nil, err
		}
		first = d.IndirectRefEntry("First")
		last = d.IndirectRefEntry("Last")
	}

	return d, first, nil
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

// PageObjFromDestinationArray returns an IndirectRef of the destinations page.
func PageObjFromDestination(ctx *model.Context, dest types.Object) (*types.IndirectRef, error) {
	var (
		err error
		ir  types.IndirectRef
		arr types.Array
	)
	switch dest := dest.(type) {
	case types.Name:
		arr, err = ctx.DereferenceDestArray(dest.Value())
		if err == nil {
			ir = arr[0].(types.IndirectRef)
		}
	case types.StringLiteral:
		s, err := types.StringLiteralToString(dest)
		if err != nil {
			return nil, err
		}
		arr, err = ctx.DereferenceDestArray(s)
		if err == nil {
			ir = arr[0].(types.IndirectRef)
		}
	case types.HexLiteral:
		s, err := types.HexLiteralToString(dest)
		if err != nil {
			return nil, err
		}
		arr, err = ctx.DereferenceDestArray(s)
		if err == nil {
			ir = arr[0].(types.IndirectRef)
		}
	case types.Array:
		if dest[0] != nil {
			ir = dest[0].(types.IndirectRef)
		}
	}
	return &ir, err
}

// BookmarksForOutlineItem returns the bookmarks tree for an outline item.
func BookmarksForOutlineItem(ctx *model.Context, item *types.IndirectRef, parent *Bookmark) ([]Bookmark, error) {
	bms := []Bookmark{}

	var (
		d   types.Dict
		err error
	)

	// Process outline items.
	for ir := item; ir != nil; ir = d.IndirectRefEntry("Next") {

		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return nil, err
		}

		s, _ := model.Text(d["Title"])
		title := outlineItemTitle(s)

		// Retrieve page number out of a destination via "Dest" or "Goto Action".
		dest, destFound := d["Dest"]
		if !destFound {
			act, actFound := d["A"]
			if !actFound {
				continue
			}
			act, _ = ctx.Dereference(act)
			actType := act.(types.Dict)["S"]
			if actType.String() != "GoTo" {
				continue
			}
			dest = act.(types.Dict)["D"]
		}

		obj, err := ctx.Dereference(dest)
		if err != nil {
			return nil, err
		}

		ir, err := PageObjFromDestination(ctx, obj)
		if err != nil {
			return nil, err
		}
		if ir == nil {
			continue
		}

		pageFrom, err := ctx.PageNumber(ir.ObjectNumber.Value())
		if err != nil {
			return nil, err
		}

		if len(bms) > 0 {
			if pageFrom > bms[len(bms)-1].PageFrom {
				bms[len(bms)-1].PageThru = pageFrom - 1
			} else {
				bms[len(bms)-1].PageThru = bms[len(bms)-1].PageFrom
			}
		}

		newBookmark := Bookmark{
			Title:    title,
			PageFrom: pageFrom,
			Parent:   parent,
			Bold:     false,
			Italic:   false,
		}

		if arr := d.ArrayEntry("C"); len(arr) == 3 {
			col := color.NewSimpleColorForArray(arr)
			newBookmark.Color = &col
		}

		if f := d.IntEntry("F"); f != nil {
			newBookmark.Bold = *f&0x02 > 0
			newBookmark.Italic = *f&0x01 > 0
		}

		first := d["First"]
		if first != nil {
			indRef := first.(types.IndirectRef)
			kids, _ := BookmarksForOutlineItem(ctx, &indRef, &newBookmark)
			newBookmark.Kids = kids
		}

		bms = append(bms, newBookmark)
	}

	return bms, nil
}

// Bookmarks returns all ctx bookmark information recursively.
func Bookmarks(ctx *model.Context) ([]Bookmark, error) {

	if err := ctx.LocateNameTree("Dests", false); err != nil {
		return nil, err
	}

	_, first, err := positionToFirstBookmark(ctx)
	if err != nil {
		if err != errNoBookmarks {
			return nil, err
		}
		return nil, nil
	}

	return BookmarksForOutlineItem(ctx, first, nil)
}

func bookmarkList(bms []Bookmark, level int) ([]string, error) {
	pre := strings.Repeat("    ", level)
	ss := []string{}
	for _, bm := range bms {
		ss = append(ss, pre+bm.Title)
		if len(bm.Kids) > 0 {
			ss1, err := bookmarkList(bm.Kids, level+1)
			if err != nil {
				return nil, err
			}
			ss = append(ss, ss1...)
		}
	}
	return ss, nil
}

func BookmarkList(ctx *model.Context) ([]string, error) {

	bms, err := Bookmarks(ctx)
	if err != nil {
		return nil, err
	}

	if bms == nil {
		return []string{"no bookmarks available"}, nil
	}

	return bookmarkList(bms, 0)
}

func ExportBookmarks(ctx *model.Context, source string) (*BookmarkTree, error) {
	bms, err := Bookmarks(ctx)
	if err != nil {
		return nil, err
	}
	if bms == nil {
		return nil, nil
	}

	bmTree := BookmarkTree{}
	bmTree.Header = header(ctx.XRefTable, source)
	bmTree.Bookmarks = bms

	return &bmTree, nil
}

func ExportBookmarksJSON(ctx *model.Context, source string, w io.Writer) (bool, error) {
	bookmarkTree, err := ExportBookmarks(ctx, source)
	if err != nil || bookmarkTree == nil {
		return false, err
	}

	bb, err := json.MarshalIndent(bookmarkTree, "", "\t")
	if err != nil {
		return false, err
	}

	_, err = w.Write(bb)

	return true, err
}

func bmDict(ctx *model.Context, bm Bookmark, parent types.IndirectRef) (types.Dict, error) {

	_, pageIndRef, _, err := ctx.PageDict(bm.PageFrom, false)
	if err != nil {
		return nil, err
	}

	arr := types.Array{*pageIndRef, types.Name("Fit")}
	ir, err := ctx.IndRefForNewObject(arr)
	if err != nil {
		return nil, err
	}

	var o types.Object = *ir

	s, err := types.EscapeUTF16String(bm.Title)
	if err != nil {
		return nil, err
	}

	d := types.Dict(map[string]types.Object{
		"Dest":   types.NewHexLiteral([]byte(bm.Title)),
		"Title":  types.StringLiteral(*s),
		"Parent": parent},
	)

	m := model.NameMap{bm.Title: []types.Dict{d}}
	if err := ctx.Names["Dests"].Add(ctx.XRefTable, bm.Title, o, m, []string{"D", "Dest"}); err != nil {
		return nil, err
	}

	if bm.Color != nil {
		d["C"] = types.Array{types.Float(bm.Color.R), types.Float(bm.Color.G), types.Float(bm.Color.B)}
	}

	if style := bm.Style(); style > 0 {
		d["F"] = types.Integer(style)
	}

	return d, nil
}

func createOutlineItemDict(ctx *model.Context, bms []Bookmark, parent *types.IndirectRef, parentPageNr *int) (*types.IndirectRef, *types.IndirectRef, int, int, error) {
	var (
		first   *types.IndirectRef
		irPrev  *types.IndirectRef
		dPrev   types.Dict
		total   int
		visible int
	)

	for i, bm := range bms {

		if i == 0 && parentPageNr != nil && bm.PageFrom < *parentPageNr {
			return nil, nil, 0, 0, errCorruptedBookmarks
		}

		if i > 0 && bm.PageFrom < bms[i-1].PageFrom {
			return nil, nil, 0, 0, errCorruptedBookmarks
		}

		total++

		d, err := bmDict(ctx, bm, *parent)
		if err != nil {
			return nil, nil, 0, 0, err
		}

		ir, err := ctx.IndRefForNewObject(d)
		if err != nil {
			return nil, nil, 0, 0, err
		}

		if first == nil {
			first = ir
		}

		if len(bm.Kids) > 0 {

			first, last, c, visc, err := createOutlineItemDict(ctx, bm.Kids, ir, &bm.PageFrom)
			if err != nil {
				return nil, nil, 0, 0, err
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

func removeNamedDests(ctx *model.Context, item *types.IndirectRef) error {
	var (
		d         types.Dict
		err       error
		empty, ok bool
	)
	for ir := item; ir != nil; ir = d.IndirectRefEntry("Next") {

		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return err
		}

		dest, destFound := d["Dest"]
		if !destFound {
			act, actFound := d["A"]
			if !actFound {
				continue
			}
			act, _ = ctx.Dereference(act)
			actType := act.(types.Dict)["S"]
			if actType.String() != "GoTo" {
				continue
			}
			dest = act.(types.Dict)["D"]
		}

		s, err := ctx.DestName(dest)
		if err != nil {
			return err
		}

		if len(s) == 0 {
			continue
		}

		// Remove destName from dest nametree.
		// TODO also try to remove from any existing root.Dests
		empty, ok, err = ctx.Names["Dests"].Remove(ctx.XRefTable, s)
		if err != nil {
			return err
		}
		if !ok {
			if log.DebugEnabled() {
				log.Debug.Println("removeNamedDests: unable to remove dest name: " + s)
			}
		}

		first := d["First"]
		if first != nil {
			indRef := first.(types.IndirectRef)
			if err := removeNamedDests(ctx, &indRef); err != nil {
				return err
			}
		}
	}

	if empty {
		delete(ctx.Names, "Dests")
		if err := ctx.RemoveNameTree("Dests"); err != nil {
			return err
		}
	}

	return nil
}

// RemoveBookmarks erases all outlines from ctx.
func RemoveBookmarks(ctx *model.Context) (bool, error) {
	_, first, err := positionToFirstBookmark(ctx)
	if err != nil {
		if err != errNoBookmarks {
			return false, err
		}
		return false, nil
	}
	if err := removeNamedDests(ctx, first); err != nil {
		return false, err
	}

	rootDict, err := ctx.Catalog()
	if err != nil {
		return false, err
	}

	rootDict["Outlines"] = nil

	return true, nil
}

// AddBookmarks adds bms to ctx.
func AddBookmarks(ctx *model.Context, bms []Bookmark, replace bool) error {

	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	if !replace {
		if _, ok := rootDict.Find("Outlines"); ok {
			return errExistingBookmarks
		}
	}

	if _, err = RemoveBookmarks(ctx); err != nil {
		return err
	}

	if err := ctx.LocateNameTree("Dests", true); err != nil {
		return err
	}

	outlinesDict := types.Dict(map[string]types.Object{"Type": types.Name("Outlines")})
	outlinesir, err := ctx.IndRefForNewObject(outlinesDict)
	if err != nil {
		return err
	}

	first, last, total, visible, err := createOutlineItemDict(ctx, bms, outlinesir, nil)
	if err != nil {
		return err
	}

	outlinesDict["First"] = *first
	outlinesDict["Last"] = *last
	outlinesDict["Count"] = types.Integer(total + visible)

	rootDict["Outlines"] = *outlinesir

	return nil
}

func addBookmarkTree(ctx *model.Context, bmTree *BookmarkTree, replace bool) error {
	return AddBookmarks(ctx, bmTree.Bookmarks, replace)
}

func parseBookmarksFromJSON(ctx *model.Context, bb []byte) (*BookmarkTree, error) {

	if !json.Valid(bb) {
		return nil, errors.Errorf("pdfcpu: invalid JSON encoding detected.")
	}

	bmTree := &BookmarkTree{}

	if err := json.Unmarshal(bb, bmTree); err != nil {
		return nil, err
	}

	return bmTree, nil
}

// ImportBookmarks creates/replaces outlines in ctx as provided by rd.
func ImportBookmarks(ctx *model.Context, rd io.Reader, replace bool) (bool, error) {

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, rd); err != nil {
		return false, err
	}

	bmTree, err := parseBookmarksFromJSON(ctx, buf.Bytes())
	if err != nil {
		return false, err
	}

	err = addBookmarkTree(ctx, bmTree, replace)
	if err != nil {
		if err == errExistingBookmarks {
			return false, nil
		}
		return true, err
	}

	return true, nil
}

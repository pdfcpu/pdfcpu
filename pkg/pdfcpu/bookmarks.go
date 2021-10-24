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
	"strings"

	"github.com/pkg/errors"
)

var (
	errNoBookmarks        = errors.New("pdfcpu: no bookmarks available")
	errCorruptedDests     = errors.New("pdfcpu: corrupted named destination")
	errCorruptedBookmarks = errors.New("pdfcpu: corrupt bookmark")
	errExistingBookmarks  = errors.New("pdfcpu: existing bookmarks")
)

// Bookmark represents an outline item tree.
type Bookmark struct {
	Title    string
	PageFrom int
	PageThru int // for extraction only; >= pageFrom and reaches until before pageFrom of the next bookmark.
	Bold     bool
	Italic   bool
	Color    *SimpleColor
	Children []Bookmark
	Parent   *Bookmark
}

// Style returns an int corresponding to the bookmark style.
func (bm Bookmark) Style() int {
	var i int
	if bm.Bold {
		i += 2
	}
	if bm.Italic {
		i += 1
	}
	return i
}

func (ctx *Context) dereferenceDestinationArray(key string) (Array, error) {
	o, ok := ctx.Names["Dests"].Value(key)
	if !ok {
		return nil, errCorruptedDests
	}
	return ctx.DereferenceArray(o)
}

func (ctx *Context) positionToOutlineTreeLevel1() (Dict, *IndirectRef, error) {
	// Load Dests nametree.
	if err := ctx.LocateNameTree("Dests", false); err != nil {
		return nil, nil, err
	}

	ir, err := ctx.Outlines()
	if err != nil {
		return nil, nil, err
	}
	if ir == nil {
		return nil, nil, errNoBookmarks
	}

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return nil, nil, err
	}
	if d == nil {
		return nil, nil, errNoBookmarks
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	// We consider Bookmarks at level 1 or 2 only.
	for *first == *last {
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

// PageObjFromDestinationArray return an IndirectRef for this destinations page object.
func (ctx *Context) PageObjFromDestinationArray(dest Object) (*IndirectRef, error) {
	var (
		err error
		ir  IndirectRef
		arr Array
	)
	switch dest := dest.(type) {
	case Name:
		arr, err = ctx.dereferenceDestinationArray(dest.Value())
		if err == nil {
			ir = arr[0].(IndirectRef)
		}
	case StringLiteral:
		arr, err = ctx.dereferenceDestinationArray(dest.Value())
		if err == nil {
			ir = arr[0].(IndirectRef)
		}
	case HexLiteral:
		arr, err = ctx.dereferenceDestinationArray(dest.Value())
		if err == nil {
			ir = arr[0].(IndirectRef)
		}
	case Array:
		if dest[0] != nil {
			ir = dest[0].(IndirectRef)
		} else {
			// Skipping bookmarks that don't point to anything.
		}
	}
	return &ir, err
}

// BookmarksForOutlineItem returns the bookmarks tree for an outline item.
func (ctx *Context) BookmarksForOutlineItem(item *IndirectRef, parent *Bookmark) ([]Bookmark, error) {
	bms := []Bookmark{}

	d, err := ctx.DereferenceDict(*item)
	if err != nil {
		return nil, err
	}

	// Process outline items.
	for ir := item; ir != nil; ir = d.IndirectRefEntry("Next") {

		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return nil, err
		}

		s, _ := Text(d["Title"])
		title := outlineItemTitle(s)

		// Retrieve page number out of a destination via "Dest" or "Goto Action".
		dest, destFound := d["Dest"]
		if !destFound {
			act, actFound := d["A"]
			if !actFound {
				continue
			}
			act, _ = ctx.Dereference(act)
			actType, _ := act.(Dict)["S"]
			if actType.String() != "GoTo" {
				continue
			}
			dest, _ = act.(Dict)["D"]
		}

		dest, _ = ctx.Dereference(dest)

		ir, err := ctx.PageObjFromDestinationArray(dest)
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
		}

		first, _ := d["First"]
		if first != nil {
			indRef := first.(IndirectRef)
			children, _ := ctx.BookmarksForOutlineItem(&indRef, &newBookmark)
			newBookmark.Children = children
		}

		bms = append(bms, newBookmark)
	}

	return bms, nil
}

// BookmarksForOutline returns all ctx bookmark information recursively.
func (ctx *Context) BookmarksForOutline() ([]Bookmark, error) {
	_, first, err := ctx.positionToOutlineTreeLevel1()
	if err != nil {
		return nil, err
	}

	return ctx.BookmarksForOutlineItem(first, nil)
}

func createOutlineItemDict(ctx *Context, bms []Bookmark, parent *IndirectRef, parentPageNr *int) (*IndirectRef, *IndirectRef, int, error) {
	var (
		first  *IndirectRef
		irPrev *IndirectRef
		dPrev  Dict
		count  int
	)

	for i, bm := range bms {

		if i == 0 && parentPageNr != nil && bm.PageFrom < *parentPageNr {
			return nil, nil, 0, errCorruptedBookmarks
		}

		if i > 0 && bm.PageFrom < bms[i-1].PageFrom {
			return nil, nil, 0, errCorruptedBookmarks
		}

		_, pageIndRef, _, err := ctx.PageDict(bm.PageFrom, false)
		if err != nil {
			return nil, nil, 0, err
		}

		s, err := Escape(EncodeUTF16String(bm.Title))
		if err != nil {
			return nil, nil, 0, err
		}

		d := Dict(map[string]Object{
			"Dest":   Array{*pageIndRef, Name("Fit")},
			"Title":  StringLiteral(*s),
			"Parent": *parent},
		)

		if bm.Color != nil {
			d["C"] = Array{Float(bm.Color.R), Float(bm.Color.G), Float(bm.Color.B)}
		}

		if style := bm.Style(); style > 0 {
			d["F"] = Integer(style)
		}

		ir, err := ctx.IndRefForNewObject(d)
		if err != nil {
			return nil, nil, 0, err
		}

		if first == nil {
			first = ir
		}

		if bm.Children != nil {
			first, last, c, err := createOutlineItemDict(ctx, bm.Children, ir, &bm.PageFrom)
			if err != nil {
				return nil, nil, 0, err
			}
			d["First"] = *first
			d["Last"] = *last

			d["Count"] = Integer(c + 1)
			count += c + 1
		} else {
			count++
		}

		if irPrev != nil {
			d["Prev"] = *irPrev
			dPrev["Next"] = *ir
		}

		dPrev = d
		irPrev = ir

	}

	return first, irPrev, count, nil
}

// AddBookmarks adds bms to ctx.
func (ctx *Context) AddBookmarks(bms []Bookmark) error {

	rootDict, err := ctx.Catalog()
	if err != nil {
		return err
	}

	if _, ok := rootDict.Find("Outlines"); ok {
		return errExistingBookmarks
	}

	outlinesDict := Dict(map[string]Object{"Type": Name("Outlines")})
	outlinesir, err := ctx.IndRefForNewObject(outlinesDict)
	if err != nil {
		return err
	}

	first, last, count, err := createOutlineItemDict(ctx, bms, outlinesir, nil)
	if err != nil {
		return err
	}

	outlinesDict["First"] = *first
	outlinesDict["Last"] = *last
	outlinesDict["Count"] = Integer(count)

	rootDict["Outlines"] = *outlinesir

	return nil
}

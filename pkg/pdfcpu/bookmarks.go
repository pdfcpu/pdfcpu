/*
Copyright 2020 The pdf Authors.

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
	errNoBookmarks    = errors.New("pdfcpu: no bookmarks available")
	errCorruptedDests = errors.New("pdfcpu: corrupted named destination")
)

// Bookmark represents an outline item at some level including page span info.
type Bookmark struct {
	Title    string
	PageFrom int
	PageThru int // >= pageFrom and reaches until before pageFrom of the next bookmark.
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

func (ctx *Context) dereferenceDestPageNumber(dest Object) (IndirectRef, error) {
	var ir IndirectRef
	switch dest := dest.(type) {
	case Name:
		arr, err := ctx.dereferenceDestinationArray(dest.Value())
		if err != nil {
			return ir, err
		}
		ir = arr[0].(IndirectRef)
	case StringLiteral:
		arr, err := ctx.dereferenceDestinationArray(dest.Value())
		if err != nil {
			return ir, err
		}
		ir = arr[0].(IndirectRef)
	case HexLiteral:
		arr, err := ctx.dereferenceDestinationArray(dest.Value())
		if err != nil {
			return ir, err
		}
		ir = arr[0].(IndirectRef)
	case Array:
		ir = dest[0].(IndirectRef)
	}
	return ir, nil
}

// BookmarksForOutlineLevel1 returns bookmarks incliuding page span info.
func (ctx *Context) BookmarksForOutlineLevel1() ([]Bookmark, error) {
	d, first, err := ctx.positionToOutlineTreeLevel1()
	if err != nil {
		return nil, err
	}

	bms := []Bookmark{}

	// Process outline items.
	for ir := first; ir != nil; ir = d.IndirectRefEntry("Next") {

		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return nil, err
		}

		s, _ := Text(d["Title"])
		var sb strings.Builder
		for i := 0; i < len(s); i++ {
			b := s[i]
			if b >= 32 {
				if b == 32 {
					b = '_'
				}
				sb.WriteByte(b)
			}
		}
		title := sb.String()

		dest, found := d["Dest"]
		if !found {
			return nil, errNoBookmarks
		}

		dest, _ = ctx.Dereference(dest)

		ir, err := ctx.dereferenceDestPageNumber(dest)
		if err != nil {
			return nil, err
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
		bms = append(bms, Bookmark{Title: title, PageFrom: pageFrom})
	}

	return bms, nil
}

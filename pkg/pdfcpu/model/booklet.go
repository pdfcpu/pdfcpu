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

package model

import (
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type BookletType int

// These are the types of booklet layouts.
const (
	Booklet BookletType = iota
	BookletAdvanced
	BookletPerfectBound
)

// String returns the string value of b.
func (b BookletType) String() string {
	switch b {
	case Booklet:
		return "booklet"
	case BookletAdvanced:
		return "booklet advanced"
	case BookletPerfectBound:
		return "booklet perfect bound"
	}
	return ""
}

type BookletBinding int

const (
	LongEdge BookletBinding = iota
	ShortEdge
)

// String returns the string value of b.
func (b BookletBinding) String() string {
	switch b {
	case ShortEdge:
		return "short-edge"
	case LongEdge:
		return "long-edge"
	}
	return ""
}

type BookletPage struct {
	Number int
	Rotate bool
}

func drawGuideLineLabel(xRefTable *XRefTable, w io.Writer, x, y float64, s string, mb *types.Rectangle, fm FontMap, rot int) error {
	fontName := "Helvetica"
	td := TextDescriptor{
		FontName:  fontName,
		FontKey:   fm.EnsureKey(fontName),
		FontSize:  9,
		Scale:     1.0,
		ScaleAbs:  true,
		StrokeCol: color.Black,
		FillCol:   color.Black,
		X:         x,
		Y:         y,
		Rotation:  float64(rot),
		Text:      s,
	}
	if _, err := WriteMultiLine(xRefTable, w, mb, nil, td); err != nil {
		return fmt.Errorf("render guide label %q: %w", s, err)
	}
	return nil
}

func drawScissors(xRefTable *XRefTable, w io.Writer, isVerticalCut bool, horzCutYpos float64, mb *types.Rectangle, fm FontMap) error {
	x := 0.
	y := horzCutYpos - 4
	rot := 0.
	if isVerticalCut {
		// TODO: if we ever have multiple vertical cuts, would need to change this.
		x = mb.Width()/2 - 12
		y = 12
		rot = 90
	}
	fontName := "ZapfDingbats"
	td := TextDescriptor{
		FontName:  fontName,
		FontKey:   fm.EnsureKey(fontName),
		FontSize:  12,
		Scale:     1.0,
		ScaleAbs:  true,
		StrokeCol: color.Black,
		FillCol:   color.Black,
		X:         x,
		Y:         y,
		Rotation:  rot,
		Text:      string([]byte{byte(34)}),
	}
	if _, err := WriteMultiLine(xRefTable, w, mb, nil, td); err != nil {
		return fmt.Errorf("render scissors: %w", err)
	}
	return nil
}

type cutOrFold int

const (
	none cutOrFold = iota
	cut
	fold
)

// String returns a string representation of c.
func (c cutOrFold) String(nup *NUp) string {
	if c == cut {
		if nup.BookletType == BookletAdvanced {
			return "Fold & Cut here"
		}
		return "Cut here"
	}
	if c == fold {
		return "Fold here"
	}
	return ""
}

func getCutFolds(nup *NUp) (horizontal cutOrFold, vertical cutOrFold) {
	var getCutOrFold = func(nup *NUp) (cutOrFold, cutOrFold) {
		switch nup.N() {
		case 2:
			return fold, none
		case 4:
			if nup.BookletBinding == LongEdge {
				return cut, fold
			} else {
				return fold, cut
			}
		case 6:
			// Really, it has two horizontal cuts.
			return cut, fold
		case 8:
			if nup.BookletBinding == LongEdge {
				// Also has cuts in the center row & column.
				return cut, cut
			} else {
				// short edge has the fold in the center col. cut on each row
				return cut, fold
			}
		}
		return none, none
	}
	horizontal, vertical = getCutOrFold(nup)
	if nup.BookletType == BookletPerfectBound {
		// All folds turn into cuts for perfect binding.
		if horizontal == fold {
			horizontal = cut
		}
		if vertical == fold {
			vertical = cut
		}
	}
	if nup.N() == 4 && nup.PageDim.Landscape() {
		// The logic above is for a portrait sheet, so swap the outputs.
		return vertical, horizontal
	}
	return horizontal, vertical
}

func drawGuideHorizontal(xRefTable *XRefTable, w io.Writer, y, width float64, cutOrFold cutOrFold, nup *NUp, mb *types.Rectangle, fm FontMap) error {
	fmt.Fprint(w, "[3] 0 d ")
	draw.SetLineWidth(w, 0)
	draw.SetStrokeColor(w, color.Gray)
	draw.DrawLineSimple(w, 0, y, width, y)
	if err := drawGuideLineLabel(xRefTable, w, width-46, y+2, cutOrFold.String(nup), mb, fm, 0); err != nil {
		return err
	}
	if cutOrFold == cut {
		return drawScissors(xRefTable, w, false, y, mb, fm)
	}
	return nil
}

func drawGuideVertical(xRefTable *XRefTable, w io.Writer, x, height float64, cutOrFold cutOrFold, nup *NUp, mb *types.Rectangle, fm FontMap) error {
	fmt.Fprint(w, "[3] 0 d ")
	draw.SetLineWidth(w, 0)
	draw.SetStrokeColor(w, color.Gray)
	draw.DrawLineSimple(w, x, 0, x, height)
	if err := drawGuideLineLabel(xRefTable, w, x-23, height-32, cutOrFold.String(nup), mb, fm, 90); err != nil {
		return err
	}
	if cutOrFold == cut {
		return drawScissors(xRefTable, w, true, height/2, mb, fm)
	}
	return nil
}

// DrawBookletGuides draws guides and reports rendering failures.
func DrawBookletGuides(xRefTable *XRefTable, nup *NUp, w io.Writer) (FontMap, error) {
	width := nup.PageDim.Width
	height := nup.PageDim.Height
	var fm FontMap = FontMap{}
	mb := types.RectForDim(width, height)

	horz, vert := getCutFolds(nup)
	if horz != none {
		switch nup.N() {
		case 2, 4:
			if err := drawGuideHorizontal(xRefTable, w, height/2, width, horz, nup, mb, fm); err != nil {
				return nil, err
			}
		case 6:
			// 6up: two cuts
			if err := drawGuideHorizontal(xRefTable, w, height*1/3, width, horz, nup, mb, fm); err != nil {
				return nil, err
			}
			if err := drawGuideHorizontal(xRefTable, w, height*2/3, width, horz, nup, mb, fm); err != nil {
				return nil, err
			}
		case 8:
			if nup.BookletBinding == LongEdge {
				// 8up: middle cut and 1/4,3/4 folds
				for _, guide := range []struct {
					y    float64
					kind cutOrFold
				}{
					{height / 2, cut},
					{height * 1 / 4, fold},
					{height * 3 / 4, fold},
				} {
					if err := drawGuideHorizontal(xRefTable, w, guide.y, width, guide.kind, nup, mb, fm); err != nil {
						return nil, err
					}
				}
			} else {
				// short edge: cuts on rows
				for i := 1; i < 4; i++ {
					if err := drawGuideHorizontal(xRefTable, w, height*float64(i)/4, width, cut, nup, mb, fm); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	if vert != none {
		if err := drawGuideVertical(xRefTable, w, width/2, height, vert, nup, mb, fm); err != nil {
			return nil, err
		}
	}
	return fm, nil
}

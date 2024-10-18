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

func drawGuideLineLabel(w io.Writer, x, y float64, s string, mb *types.Rectangle, fm FontMap, rot int) {
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
	WriteMultiLine(nil, w, mb, nil, td)
}

func drawScissors(w io.Writer, isVerticalCut bool, horzCutYpos float64, mb *types.Rectangle, fm FontMap) {
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
	WriteMultiLine(nil, w, mb, nil, td)
}

type cutOrFold int

const (
	none cutOrFold = iota
	cut
	fold
)

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

func drawGuideHorizontal(w io.Writer, y, width float64, cutOrFold cutOrFold, nup *NUp, mb *types.Rectangle, fm FontMap) {
	fmt.Fprint(w, "[3] 0 d ")
	draw.SetLineWidth(w, 0)
	draw.SetStrokeColor(w, color.Gray)
	draw.DrawLineSimple(w, 0, y, width, y)
	drawGuideLineLabel(w, width-46, y+2, cutOrFold.String(nup), mb, fm, 0)
	if cutOrFold == cut {
		drawScissors(w, false, y, mb, fm)
	}
}

func drawGuideVertical(w io.Writer, x, height float64, cutOrFold cutOrFold, nup *NUp, mb *types.Rectangle, fm FontMap) {
	fmt.Fprint(w, "[3] 0 d ")
	draw.SetLineWidth(w, 0)
	draw.SetStrokeColor(w, color.Gray)
	draw.DrawLineSimple(w, x, 0, x, height)
	drawGuideLineLabel(w, x-23, height-32, cutOrFold.String(nup), mb, fm, 90)
	if cutOrFold == cut {
		drawScissors(w, true, height/2, mb, fm)
	}
}

// DrawBookletGuides draws guides according to corresponding nup value.
func DrawBookletGuides(nup *NUp, w io.Writer) FontMap {
	width := nup.PageDim.Width
	height := nup.PageDim.Height
	var fm FontMap = FontMap{}
	mb := types.RectForDim(width, height)

	horz, vert := getCutFolds(nup)
	if horz != none {
		switch nup.N() {
		case 2, 4:
			drawGuideHorizontal(w, height/2, width, horz, nup, mb, fm)
		case 6:
			// 6up: two cuts
			drawGuideHorizontal(w, height*1/3, width, horz, nup, mb, fm)
			drawGuideHorizontal(w, height*2/3, width, horz, nup, mb, fm)
		case 8:
			if nup.BookletBinding == LongEdge {
				// 8up: middle cut and 1/4,3/4 folds
				drawGuideHorizontal(w, height/2, width, cut, nup, mb, fm)
				drawGuideHorizontal(w, height*1/4, width, fold, nup, mb, fm)
				drawGuideHorizontal(w, height*3/4, width, fold, nup, mb, fm)
			} else {
				// short edge: cuts on rows
				for i := 1; i < 4; i++ {
					drawGuideHorizontal(w, height*float64(i)/4, width, cut, nup, mb, fm)
				}
			}
		}
	}
	if vert != none {
		drawGuideVertical(w, width/2, height, vert, nup, mb, fm)
	}
	return fm
}

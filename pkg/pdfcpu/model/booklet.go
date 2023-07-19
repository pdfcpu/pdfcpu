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

func drawScissors(w io.Writer, mb *types.Rectangle, fm FontMap) {
	fontName := "ZapfDingbats"
	td := TextDescriptor{
		FontName:  fontName,
		FontKey:   fm.EnsureKey(fontName),
		FontSize:  12,
		Scale:     1.0,
		ScaleAbs:  true,
		StrokeCol: color.Black,
		FillCol:   color.Black,
		X:         0,
		Y:         mb.Height()/2 - 4,
		Text:      string([]byte{byte(34)}),
	}
	WriteMultiLine(nil, w, mb, nil, td)
}

// DrawBookletGuides draws guides according to corresponding nup value.
func DrawBookletGuides(nup *NUp, w io.Writer) FontMap {
	width := nup.PageDim.Width
	height := nup.PageDim.Height
	var fm FontMap = FontMap{}
	mb := types.RectForDim(nup.PageDim.Width, nup.PageDim.Height)

	draw.SetLineWidth(w, 0)
	draw.SetStrokeColor(w, color.Gray)

	switch nup.N() {
	case 2:
		// Draw horizontal folding line.
		fmt.Fprint(w, "[3] 0 d ")
		draw.DrawLineSimple(w, 0, height/2, width, height/2)
		drawGuideLineLabel(w, 1, height/2+2, "Fold here", mb, fm, 0)
	case 4:
		// Draw vertical folding line.
		fmt.Fprint(w, "[3] 0 d ")
		draw.DrawLineSimple(w, width/2, 0, width/2, height)
		drawGuideLineLabel(w, width/2-23, 20, "Fold here", mb, fm, 90)

		// Draw horizontal cutting line.
		fmt.Fprint(w, "[3] 0 d ")
		draw.DrawLineSimple(w, 0, height/2, width, height/2)
		drawGuideLineLabel(w, width, height/2+2, "Fold & Cut here", mb, fm, 0)

		// Draw scissors over cutting line.
		drawScissors(w, mb, fm)
	}

	return fm
}

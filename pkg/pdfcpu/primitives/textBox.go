/*
	Copyright 2021 The pdfcpu Authors.

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

package primitives

import (
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type TextBox struct {
	pdf      *PDF
	content  *Content
	Name     string
	Value    string     // text, content
	Position [2]float64 `json:"pos"` // x,y
	x, y     float64
	Dx, Dy   float64
	Anchor   string
	anchor   pdfcpu.Anchor
	anchored bool
	Width    float64

	Font    *FormFont
	Margin  *Margin // applied to content box
	Border  *Border
	Padding *Padding // applied to TextDescriptor marginx

	BackgroundColor string `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor
	Alignment       string `json:"align"` // "Left", "Center", "Right"
	horAlign        pdfcpu.HAlignment
	RTL             bool
	Rotation        float64 `json:"rot"`
	Hide            bool
}

func (tb *TextBox) validate() error {

	tb.x = tb.Position[0]
	tb.y = tb.Position[1]

	if tb.Name == "$" {
		return errors.New("pdfcpu: invalid text reference $")
	}

	if tb.Anchor != "" {
		if tb.Position[0] != 0 || tb.Position[1] != 0 {
			return errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
		}
		a, err := pdfcpu.ParseAnchor(tb.Anchor)
		if err != nil {
			return err
		}
		tb.anchor = a
		tb.anchored = true
	}

	if tb.Font != nil {
		tb.Font.pdf = tb.pdf
		if err := tb.Font.validate(); err != nil {
			return err
		}
	} else if !strings.HasPrefix(tb.Name, "$") {
		return errors.New("pdfcpu: textbox missing font definition")
	}

	if tb.Margin != nil {
		if err := tb.Margin.validate(); err != nil {
			return err
		}
	}

	if tb.Border != nil {
		tb.Border.pdf = tb.pdf
		if err := tb.Border.validate(); err != nil {
			return err
		}
	}

	if tb.Padding != nil {
		if err := tb.Padding.validate(); err != nil {
			return err
		}
	}

	if tb.BackgroundColor != "" {
		sc, err := tb.pdf.parseColor(tb.BackgroundColor)
		if err != nil {
			return err
		}
		tb.bgCol = sc
	}

	tb.horAlign = pdfcpu.AlignLeft
	if tb.Alignment != "" {
		ha, err := pdfcpu.ParseHorAlignment(tb.Alignment)
		if err != nil {
			return err
		}
		tb.horAlign = ha
	}

	return nil
}

func (tb *TextBox) font(name string) *FormFont {
	return tb.content.namedFont(name)
}

func (tb *TextBox) margin(name string) *Margin {
	return tb.content.namedMargin(name)
}

func (tb *TextBox) border(name string) *Border {
	return tb.content.namedBorder(name)
}

func (tb *TextBox) padding(name string) *Padding {
	return tb.content.namedPadding(name)
}

func (tb *TextBox) mergeIn(tb0 *TextBox) {

	if !tb.anchored && tb.x == 0 && tb.y == 0 {
		tb.x = tb0.x
		tb.y = tb0.y
		tb.anchor = tb0.anchor
		tb.anchored = tb0.anchored
	}

	if tb.Dx == 0 {
		tb.Dx = tb0.Dx
	}
	if tb.Dy == 0 {
		tb.Dy = tb0.Dy
	}

	if tb.Width == 0 {
		tb.Width = tb0.Width
	}

	if tb.Margin == nil {
		tb.Margin = tb0.Margin
	}

	if tb.Border == nil {
		tb.Border = tb0.Border
	}

	if tb.Padding == nil {
		tb.Padding = tb0.Padding
	}

	if tb.Font == nil {
		tb.Font = tb0.Font
	}

	if tb.horAlign == pdfcpu.AlignLeft {
		tb.horAlign = tb0.horAlign
	}

	if tb.bgCol == nil {
		tb.bgCol = tb0.bgCol
	}

	if tb.Rotation == 0 {
		tb.Rotation = tb0.Rotation
	}

	if !tb.Hide {
		tb.Hide = tb0.Hide
	}
}

func (tb *TextBox) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap) error {
	pdf := tb.content.page.pdf
	f := tb.Font
	if f.Name[0] == '$' {
		// use named font
		fName := f.Name[1:]
		f0 := tb.font(fName)
		if f0 == nil {
			return errors.Errorf("pdfcpu: unknown font name %s", fName)
		}
		f.Name = f0.Name
		if f.Size == 0 {
			f.Size = f0.Size
		}
		if f.col == nil {
			f.col = f0.col
		}
	}
	if f.col == nil {
		f.col = &pdfcpu.Black
	}

	fontName := f.Name
	fontSize := f.Size
	col := f.col

	t, _ := pdfcpu.ResolveWMTextString(tb.Value, pdf.TimestampFormat, pageNr, pdf.pageCount())

	id, err := tb.pdf.idForFontName(fontName, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	dx, dy := pdfcpu.NormalizeOffset(tb.Dx, tb.Dy, pdf.origin)

	td := pdfcpu.TextDescriptor{
		Text:     t,
		Dx:       dx,
		Dy:       dy,
		HAlign:   tb.horAlign,
		VAlign:   pdfcpu.AlignBottom,
		FontName: fontName,
		FontKey:  id,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		Rotation: tb.Rotation,
		RTL:      tb.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if tb.bgCol != nil {
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *tb.bgCol
	}

	if tb.Border != nil {
		b := tb.Border
		if b.Name != "" && b.Name[0] == '$' {
			// Use named border
			bName := b.Name[1:]
			b0 := tb.border(bName)
			if b0 == nil {
				return errors.Errorf("pdfcpu: unknown named border %s", bName)
			}
			b.mergeIn(b0)
		}
		if b.Width >= 0 {
			td.BorderWidth = float64(b.Width)
			if b.col != nil {
				td.BorderCol = *b.col
				td.ShowBorder = true
			}
			td.BorderStyle = b.style
		}
	}

	if tb.Padding != nil {
		p := tb.Padding
		if p.Name != "" && p.Name[0] == '$' {
			// use named padding
			pName := p.Name[1:]
			p0 := tb.padding(pName)
			if p0 == nil {
				return errors.Errorf("pdfcpu: unknown named padding %s", pName)
			}
			p.mergeIn(p0)
		}

		// TODO TextDescriptor margin is actually a padding.
		if p.Width > 0 {
			td.MTop = p.Width
			td.MRight = p.Width
			td.MBot = p.Width
			td.MLeft = p.Width
		} else {
			td.MTop = p.Top
			td.MRight = p.Right
			td.MBot = p.Bottom
			td.MLeft = p.Left
		}
	}

	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if tb.Margin != nil {
		m := tb.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := tb.margin(mName)
			if m0 == nil {
				return errors.Errorf("pdfcpu: unknown named margin %s", mName)
			}
			m.mergeIn(m0)
		}

		if m.Width > 0 {
			mTop = m.Width
			mRight = m.Width
			mBottom = m.Width
			mLeft = m.Width
		} else {
			mTop = m.Top
			mRight = m.Right
			mBottom = m.Bottom
			mLeft = m.Left
		}
	}

	cBox := tb.content.Box()
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	if tb.anchored {
		pdfcpu.WriteMultiLineAnchored(p.Buf, r, nil, td, tb.anchor)
		return nil
	}

	td.X, td.Y = pdfcpu.NormalizeCoord(tb.x, tb.y, tb.content.Box(), pdf.origin, false)

	if td.X == -1 {
		// Center horizontally
		td.X = cBox.Center().X - r.LL.X
	} else if td.X > 0 {
		td.X -= mLeft
		if td.X < 0 {
			td.X = 0
		}
	}

	if td.Y == -1 {
		// Center vertically
		td.Y = cBox.Center().Y - r.LL.Y
		td.VAlign = pdfcpu.AlignMiddle
	} else if td.Y > 0 {
		td.Y -= mBottom
		if td.Y < 0 {
			td.Y = 0
		}
		r.LL.Y += td.BorderWidth
	}

	pdfcpu.WriteColumn(p.Buf, r, nil, td, float64(tb.Width))

	return nil
}

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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/format"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// TextBox represents a form text input field including a positioned label.
type TextBox struct {
	pdf      *PDF
	content  *Content
	Name     string
	Value    string     // text, content
	Position [2]float64 `json:"pos"` // x,y
	x, y     float64
	Dx, Dy   float64
	Anchor   string
	anchor   types.Anchor
	anchored bool
	Width    float64

	Font    *FormFont
	Margin  *Margin // applied to content box
	Border  *Border
	Padding *Padding // applied to TextDescriptor marginx

	BackgroundColor string `json:"bgCol"`
	bgCol           *color.SimpleColor
	Alignment       string `json:"align"` // "Left", "Center", "Right"
	horAlign        types.HAlignment
	RTL             bool
	Rotation        float64 `json:"rot"`
	Hide            bool
}

func (tb *TextBox) validateAnchor() error {
	if tb.Anchor != "" {
		if tb.Position[0] != 0 || tb.Position[1] != 0 {
			return errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
		}
		a, err := types.ParseAnchor(tb.Anchor)
		if err != nil {
			return err
		}
		tb.anchor = a
		tb.anchored = true
	}
	return nil
}

func (tb *TextBox) validateFont() error {
	if tb.Font != nil {
		tb.Font.pdf = tb.pdf
		if err := tb.Font.validate(); err != nil {
			return err
		}
	} else if !strings.HasPrefix(tb.Name, "$") {
		return errors.New("pdfcpu: textbox missing font definition")
	}
	return nil
}

func (tb *TextBox) validateMargin() error {
	if tb.Margin == nil {
		return nil
	}
	return tb.Margin.validate()
}

func (tb *TextBox) validateBorder() error {
	if tb.Border != nil {
		tb.Border.pdf = tb.pdf
		if err := tb.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (tb *TextBox) validatePadding() error {
	if tb.Padding == nil {
		return nil
	}
	return tb.Padding.validate()
}

func (tb *TextBox) validateBackgroundColor() error {
	if tb.BackgroundColor != "" {
		sc, err := tb.pdf.parseColor(tb.BackgroundColor)
		if err != nil {
			return err
		}
		tb.bgCol = sc
	}
	return nil
}

func (tb *TextBox) validateHorAlign() error {
	tb.horAlign = types.AlignLeft
	if tb.Alignment != "" {
		ha, err := types.ParseHorAlignment(tb.Alignment)
		if err != nil {
			return err
		}
		tb.horAlign = ha
	}
	return nil
}

func (tb *TextBox) validate() error {

	tb.x = tb.Position[0]
	tb.y = tb.Position[1]

	if tb.Name == "$" {
		return errors.New("pdfcpu: invalid text reference $")
	}

	if err := tb.validateAnchor(); err != nil {
		return err
	}

	if err := tb.validateFont(); err != nil {
		return err
	}

	if err := tb.validateMargin(); err != nil {
		return err
	}

	if err := tb.validateBorder(); err != nil {
		return err
	}

	if err := tb.validatePadding(); err != nil {
		return err
	}

	if err := tb.validateBackgroundColor(); err != nil {
		return err
	}

	return tb.validateHorAlign()
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

func (tb *TextBox) mergeInPos(tb0 *TextBox) {

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
}

func (tb *TextBox) mergeIn(tb0 *TextBox) {

	tb.mergeInPos(tb0)

	if tb.Value == "" {
		tb.Value = tb0.Value
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

	if tb.horAlign == types.AlignLeft {
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

func (tb *TextBox) calcFont() error {
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
		if f.Lang == "" {
			f.Lang = f0.Lang
		}
		if f.Script == "" {
			f.Script = f0.Script
		}
	}
	if f.col == nil {
		f.col = &color.Black
	}
	return nil
}

func tdMargin(p *Padding, td *model.TextDescriptor) {
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

func (tb *TextBox) prepareTextDescriptor(p *model.Page, pageNr int, fonts model.FontMap) (*model.TextDescriptor, error) {

	pdf := tb.pdf
	f := tb.Font
	fontName := f.Name
	fontLang := f.Lang
	fontSize := f.Size
	col := f.col

	t, _ := format.Text(tb.Value, pdf.TimestampFormat, pageNr, pdf.pageCount())

	id, err := tb.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
	if err != nil {
		return nil, err
	}

	dx, dy := types.NormalizeOffset(tb.Dx, tb.Dy, pdf.origin)

	td := model.TextDescriptor{
		Text:     t,
		Dx:       dx,
		Dy:       dy,
		HAlign:   tb.horAlign,
		VAlign:   types.AlignBottom,
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

	bgCol := tb.bgCol
	if bgCol == nil {
		bgCol = tb.content.bgCol
	}
	if bgCol != nil {
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *bgCol
	}

	if tb.Border != nil {
		b := tb.Border
		if b.Name != "" && b.Name[0] == '$' {
			// Use named border
			bName := b.Name[1:]
			b0 := tb.border(bName)
			if b0 == nil {
				return nil, errors.Errorf("pdfcpu: unknown named border %s", bName)
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
				return nil, errors.Errorf("pdfcpu: unknown named padding %s", pName)
			}
			p.mergeIn(p0)
		}
		tdMargin(p, &td)
	}

	return &td, nil
}

func (tb *TextBox) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if tb.Margin != nil {
		m := tb.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := tb.margin(mName)
			if m0 == nil {
				return mTop, mRight, mBottom, mLeft, errors.Errorf("pdfcpu: unknown named margin %s", mName)
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
	return mTop, mRight, mBottom, mLeft, nil
}

func (tb *TextBox) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	pdf := tb.pdf

	if err := tb.calcFont(); err != nil {
		return err
	}

	td, err := tb.prepareTextDescriptor(p, pageNr, fonts)
	if err != nil {
		return err
	}

	mTop, mRight, mBottom, mLeft, err := tb.calcMargin()
	if err != nil {
		return err
	}

	cBox := tb.content.Box()
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	if pdf.Debug {
		td.ShowPosition = true
		td.HairCross = true
		td.ShowLineBB = true
	}

	if tb.anchored {
		model.WriteMultiLineAnchored(tb.pdf.XRefTable, p.Buf, r, nil, *td, tb.anchor)
		return nil
	}

	td.X, td.Y = types.NormalizeCoord(tb.x, tb.y, tb.content.Box(), pdf.origin, false)

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
		td.VAlign = types.AlignMiddle
	} else if td.Y > 0 {
		td.Y -= mBottom
		if td.Y < 0 {
			td.Y = 0
		}
		r.LL.Y += td.BorderWidth
	}

	model.WriteColumn(tb.pdf.XRefTable, p.Buf, r, nil, *td, float64(tb.Width))

	return nil
}

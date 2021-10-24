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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type HorizontalBand struct {
	pdf             *PDF
	Left            string
	Center          string
	Right           string
	position        pdfcpu.Anchor // topcenter, center, bottomcenter
	Height          float64
	Dx, Dy          int
	BackgroundColor string `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor
	Font            *FormFont
	From            int
	Thru            int
	Border          bool
}

func (hb *HorizontalBand) validate() error {

	pdf := hb.pdf

	if hb.BackgroundColor != "" {
		sc, err := pdf.parseColor(hb.BackgroundColor)
		if err != nil {
			return err
		}
		hb.bgCol = sc
	}

	if hb.Font != nil {
		hb.Font.pdf = pdf
		if err := hb.Font.validate(); err != nil {
			return err
		}
	}

	if hb.Height <= 0 {
		return errors.Errorf("pdfcpu: missing header/footer height")
	}

	return nil
}

func (hb *HorizontalBand) renderAnchoredImageBox(
	imageName string,
	r *pdfcpu.Rectangle,
	a pdfcpu.Anchor,
	p *pdfcpu.Page,
	pageNr int,
	images pdfcpu.ImageMap) error {

	ib := hb.pdf.ImageBoxPool[imageName]
	if ib == nil {
		return errors.Errorf("pdfcpu: HorizontalBand - unable to resolve $%s", imageName)
	}

	if ib.Margin != nil && ib.Margin.Name != "" {
		return errors.Errorf("pdfcpu: HorizontalBand - unsupported named margin %s", ib.Margin.Name)
	}

	if ib.Border != nil && ib.Border.Name != "" {
		return errors.Errorf("pdfcpu: HorizontalBand - unsupported named border %s", ib.Border.Name)
	}

	if ib.Padding != nil && ib.Padding.Name != "" {
		return errors.Errorf("pdfcpu: HorizontalBand - unsupported named padding %s", ib.Padding.Name)
	}

	// push state
	anchor, anchored, dest := ib.anchor, ib.anchored, ib.dest

	ib.anchor, ib.anchored, ib.dest = a, true, r

	if err := ib.render(p, pageNr, images); err != nil {
		return err
	}

	// pop state
	ib.anchor, ib.anchored, ib.dest = anchor, anchored, dest

	return nil
}

func (hb *HorizontalBand) renderAnchoredTextBox(
	text string,
	r *pdfcpu.Rectangle,
	a pdfcpu.Anchor,
	p *pdfcpu.Page,
	pageNr int,
	fonts pdfcpu.FontMap) error {

	pdf := hb.pdf
	font := hb.Font
	bgCol := hb.bgCol

	fontName := font.Name
	fontSize := font.Size
	col := font.col
	t, _ := pdfcpu.ResolveWMTextString(text, pdf.TimestampFormat, pageNr, pdf.pageCount())

	id, err := pdf.idForFontName(fontName, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := pdfcpu.TextDescriptor{
		Text:     t,
		FontName: fontName,
		FontKey:  id,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		//RTL:      tb.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if bgCol != nil {
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *bgCol
	}

	pdfcpu.WriteMultiLineAnchored(p.Buf, r, nil, td, a)

	return nil
}

func (hb *HorizontalBand) renderComponent(
	content string,
	a pdfcpu.Anchor,
	r *pdfcpu.Rectangle,
	p *pdfcpu.Page,
	pageNr int,
	fonts pdfcpu.FontMap,
	images pdfcpu.ImageMap) error {

	if content[0] == '$' {
		return hb.renderAnchoredImageBox(content[1:], r, a, p, pageNr, images)
	}

	return hb.renderAnchoredTextBox(content, r, a, p, pageNr, fonts)
}

func (hb *HorizontalBand) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap, images pdfcpu.ImageMap, top bool) error {

	if pageNr < hb.From || (hb.Thru > 0 && pageNr > hb.Thru) {
		return nil
	}

	left := pdfcpu.BottomLeft
	center := pdfcpu.BottomCenter
	right := pdfcpu.BottomRight
	if top {
		left = pdfcpu.Left
		center = pdfcpu.Center
		right = pdfcpu.Right
	}

	if hb.Font.Name[0] == '$' {
		if err := hb.pdf.calcFont(hb.Font); err != nil {
			return err
		}
	}

	llx := p.CropBox.LL.X + float64(hb.Dx)
	lly := p.CropBox.LL.Y + float64(hb.Dy)
	if top {
		lly = p.CropBox.UR.Y - float64(hb.Dy) - hb.Height
	}
	w := p.CropBox.Width() - float64(2*hb.Dx)
	h := hb.Height
	r := pdfcpu.RectForWidthAndHeight(llx, lly, w, h)

	if hb.Left != "" {
		if err := hb.renderComponent(hb.Left, left, r, p, pageNr, fonts, images); err != nil {
			return err
		}
	}

	if hb.Center != "" {
		if err := hb.renderComponent(hb.Center, center, r, p, pageNr, fonts, images); err != nil {
			return err
		}
	}

	if hb.Right != "" {
		if err := hb.renderComponent(hb.Right, right, r, p, pageNr, fonts, images); err != nil {
			return err
		}
	}

	if hb.Border {
		pdfcpu.DrawRect(p.Buf, r, 0, &pdfcpu.Black, nil)
	}

	return nil
}

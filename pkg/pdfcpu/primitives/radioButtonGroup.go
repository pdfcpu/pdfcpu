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
	"bytes"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type RadioButtonGroup struct {
	pdf         *PDF
	content     *Content
	Label       *TextFieldLabel
	ID          string
	Value       string     // checked button
	Position    [2]float64 `json:"pos"` // x,y
	x, y        float64
	Width       float64
	boundingBox *pdfcpu.Rectangle
	Orientation string
	hor         bool
	Dx, Dy      float64
	Margin      *Margin // applied to content box
	Buttons     *Buttons
	Hide        bool
	//Border          *Border
	//BackgroundColor string `json:"bgCol"`
	//bgCol           *SimpleColor
}

func (rbg *RadioButtonGroup) font(name string) *FormFont {
	return rbg.content.namedFont(name)
}

func (rbg *RadioButtonGroup) margin(name string) *Margin {
	return rbg.content.namedMargin(name)
}

func (rbg *RadioButtonGroup) validate() error {

	if rbg.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if rbg.pdf.FieldIDs[rbg.ID] {
		return errors.Errorf("pdfcpu: duplicate form field: %s", rbg.ID)
	}
	rbg.pdf.FieldIDs[rbg.ID] = true

	if rbg.Position[0] < 0 || rbg.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", rbg.ID)
	}
	rbg.x, rbg.y = rbg.Position[0], rbg.Position[1]

	rbg.hor = true
	if rbg.Orientation != "" {
		o, err := pdfcpu.ParseRadioButtonOrientation(rbg.Orientation)
		if err != nil {
			return err
		}
		rbg.hor = o == pdfcpu.Horizontal
	}

	if rbg.Width <= 0 {
		return errors.Errorf("pdfcpu: field: %s width <= 0", rbg.ID)
	}

	if rbg.Margin != nil {
		if err := rbg.Margin.validate(); err != nil {
			return err
		}
	}

	if rbg.Label != nil {
		rbg.Label.pdf = rbg.pdf
		if err := rbg.Label.validate(); err != nil {
			return err
		}
	}

	if rbg.Buttons == nil {
		return errors.New("pdfcpu: radiobutton missing buttons")
	}

	rbg.Buttons.pdf = rbg.pdf
	return rbg.Buttons.validate()
}

func (rbg *RadioButtonGroup) buttonLabelPosition(i int, maxWidth, firstWidth float64) (float64, float64) {
	rbw := rbg.Width
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}
		var x float64
		if rbg.Buttons.Label.horAlign == pdfcpu.AlignLeft {
			x = rbg.boundingBox.LL.X + float64(i)*(rbw+w) + rbw
		}
		if rbg.Buttons.Label.horAlign == pdfcpu.AlignRight {
			x = rbg.boundingBox.LL.X + firstWidth
			if i > 0 {
				x += float64(i) * (rbw + w)
			}
			//x -= 3
		}
		return x, rbg.boundingBox.LL.Y
	}

	if maxWidth > w {
		w = maxWidth
	}
	dx := rbw
	if rbg.Buttons.Label.horAlign == pdfcpu.AlignRight {
		dx = w
	}
	dy := float64(i) * (rbw + g)
	return rbg.boundingBox.LL.X + dx, rbg.boundingBox.LL.Y - dy
}

func (rbg *RadioButtonGroup) renderButtonLabels(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap) error {
	l := rbg.Buttons.Label

	fontName := l.Font.Name
	fontSize := l.Font.Size
	col := l.Font.col

	id, err := rbg.pdf.idForFontName(fontName, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := pdfcpu.TextDescriptor{
		FontName: fontName,
		FontKey:  id,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		td.ShowTextBB = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	w, firstw := rbg.Buttons.maxLabelWidth(rbg.hor)

	td.HAlign, td.VAlign = l.horAlign, pdfcpu.AlignBottom

	for i, v := range rbg.Buttons.Values {
		td.Text = v
		td.X, td.Y = rbg.buttonLabelPosition(i, w, firstw)
		pdfcpu.WriteColumn(p.Buf, p.MediaBox, nil, td, 0)
	}

	return nil
}

func (rbg *RadioButtonGroup) buttonGroupBB() *pdfcpu.Rectangle {
	maxWidth, lastWidth := rbg.Buttons.maxLabelWidth(rbg.hor)
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)

	rbSize := rbg.Width
	rbCount := float64(len(rbg.Buttons.Values))

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}

		width := (rbCount-1)*(w+rbSize) + rbSize + lastWidth
		// if rbg.Buttons.Label.horAlign == AlignRight {
		// 	width += 3
		// }
		return pdfcpu.RectForWidthAndHeight(rbg.boundingBox.LL.X, rbg.boundingBox.LL.Y, width, rbSize)
	}

	if maxWidth > w {
		w = maxWidth
	}
	y := rbg.boundingBox.LL.Y - (rbCount-1)*(rbSize+g) // g is better smth derived from fontsize
	h := rbSize + (rbCount-1)*(rbSize+g)

	return pdfcpu.RectForWidthAndHeight(rbg.boundingBox.LL.X, y, w+rbSize, h)
}

func labelPosition(
	relPos pdfcpu.RelPosition,
	horAlign pdfcpu.HAlignment,
	boundingBox *pdfcpu.Rectangle,
	labelHeight, w, g float64, multiline bool) (float64, float64) {

	var x, y float64

	switch relPos {

	case pdfcpu.RelPosLeft:
		x = boundingBox.LL.X - g
		if horAlign == pdfcpu.AlignLeft {
			x -= w
			if x < 0 {
				x = 0
			}
		}
		if multiline {
			y = boundingBox.UR.Y - labelHeight
		} else {
			y = boundingBox.LL.Y
		}

	case pdfcpu.RelPosRight:
		x = boundingBox.UR.X + g
		if horAlign == pdfcpu.AlignRight {
			x += w
		}
		if multiline {
			y = boundingBox.UR.Y - labelHeight
		} else {
			y = boundingBox.LL.Y
		}

	case pdfcpu.RelPosTop:
		y = boundingBox.UR.Y + g
		x = boundingBox.LL.X
		if horAlign == pdfcpu.AlignRight {
			x += boundingBox.Width()
		} else if horAlign == pdfcpu.AlignCenter {
			x += boundingBox.Width() / 2
		}

	case pdfcpu.RelPosBottom:
		y = boundingBox.LL.Y - g - labelHeight
		x = boundingBox.LL.X
		if horAlign == pdfcpu.AlignRight {
			x += boundingBox.Width()
		} else if horAlign == pdfcpu.AlignCenter {
			x += boundingBox.Width() / 2
		}

	}

	return x, y
}

func (rbg *RadioButtonGroup) renderLabels(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap) error {

	rbg.renderButtonLabels(p, pageNr, fonts)

	// Main label:
	if rbg.Label == nil {
		return nil
	}

	l := rbg.Label
	v := "Default"
	if l.Value != "" {
		v = l.Value
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	fontName := l.Font.Name
	fontSize := l.Font.Size
	col := l.Font.col

	id, err := rbg.pdf.idForFontName(fontName, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := pdfcpu.TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  id,
		FontSize: fontSize,
		HAlign:   l.horAlign,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		td.ShowTextBB = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	bb := pdfcpu.WriteMultiLine(new(bytes.Buffer), pdfcpu.RectForFormat("A4"), nil, td)
	buttonGroupBB := rbg.buttonGroupBB()
	td.X, td.Y = labelPosition(l.relPos, l.horAlign, buttonGroupBB, bb.Height(), w, g, !rbg.hor)
	td.VAlign = pdfcpu.AlignBottom

	pdfcpu.WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func (rbg *RadioButtonGroup) rect(i, maxWidth, firstWidth float64) *pdfcpu.Rectangle {
	rbw := rbg.Width
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}
		var x float64
		if rbg.Buttons.Label.horAlign == pdfcpu.AlignLeft {
			x = rbg.boundingBox.LL.X + i*(rbw+w)
		}
		if rbg.Buttons.Label.horAlign == pdfcpu.AlignRight {
			x = rbg.boundingBox.LL.X + firstWidth // ??
			if i > 0 {
				x += i * (rbw + w)
			}
		}
		// TODO Increase width to enlarge focusArea.
		return pdfcpu.RectForWidthAndHeight(x, rbg.boundingBox.LL.Y, rbw, rbw)
	}
	if maxWidth > w {
		w = maxWidth
	}
	dx := 0.
	if rbg.Buttons.Label.horAlign == pdfcpu.AlignRight {
		dx = w
	}
	dy := i * (rbw + g)
	return pdfcpu.RectForWidthAndHeight(rbg.boundingBox.LL.X+dx, rbg.boundingBox.LL.Y-dy, rbw, rbw)
}

func (rbg *RadioButtonGroup) irDOff() (*pdfcpu.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[w]
	if found && ap.irDOff != nil {
		return ap.irDOff, nil
	}

	f := .5523
	r := w / 2
	r1 := r - .5

	buf := fmt.Sprintf("0.5g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	buf += fmt.Sprintf("f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	buf += "s Q "

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, w, w))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.CheckBoxAPs[w] = ap
	}
	ap.irDOff = ir

	return ir, nil
}

func (rbg *RadioButtonGroup) irDYes() (*pdfcpu.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[w]
	if found && ap.irDYes != nil {
		return ap.irDYes, nil
	}

	f := .5523
	r := w / 2
	r1 := r - .5
	r2 := r / 2

	buf := fmt.Sprintf("0.5 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	buf += fmt.Sprintf("f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	buf += fmt.Sprintf("s Q 0 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r2)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r2, f*r2, f*r2, r2, 0., r2)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r2, r2, -r2, f*r2, -r2, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r2, -f*r2, -f*r2, -r2, .0, -r2)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r2, -r2, r2, -f*r2, r2, 0.)
	buf += "f Q "

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, w, w))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.CheckBoxAPs[w] = ap
	}
	ap.irDOff = ir

	return ir, nil
}

func (rbg *RadioButtonGroup) irNOff() (*pdfcpu.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[w]
	if found && ap.irNOff != nil {
		return ap.irNOff, nil
	}

	f := .5523
	r := w / 2
	r1 := r - .5

	buf := fmt.Sprintf("1 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	buf += fmt.Sprintf("f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	buf += "s Q "

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, w, w))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.CheckBoxAPs[w] = ap
	}
	ap.irDOff = ir

	return ir, nil
}

func (rbg *RadioButtonGroup) irNYes() (*pdfcpu.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[w]
	if found && ap.irNYes != nil {
		return ap.irNYes, nil
	}

	f := .5523
	r := w / 2
	r1 := r - .5
	r2 := r / 2

	buf := fmt.Sprintf("1 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	buf += fmt.Sprintf("f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	buf += fmt.Sprintf("s Q 0 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", r, r, r2)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", r2, f*r2, f*r2, r2, 0., r2)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r2, r2, -r2, f*r2, -r2, 0.)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", -r2, -f*r2, -f*r2, -r2, .0, -r2)
	buf += fmt.Sprintf("%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r2, -r2, r2, -f*r2, r2, 0.)
	buf += "f Q "

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, w, w))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.CheckBoxAPs[w] = ap
	}
	ap.irDOff = ir

	return ir, nil
}

func (rbg *RadioButtonGroup) appearanceIndRefs() (
	*pdfcpu.IndirectRef, *pdfcpu.IndirectRef, *pdfcpu.IndirectRef, *pdfcpu.IndirectRef, error) {

	irDOff, err := rbg.irDOff()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irDYes, err := rbg.irDYes()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNOff, err := rbg.irNOff()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNYes, err := rbg.irNYes()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return irDOff, irDYes, irNOff, irNYes, nil
}

func (rbg *RadioButtonGroup) renderRadioButtonFields(p *pdfcpu.Page, parent pdfcpu.IndirectRef) (pdfcpu.Array, error) {

	irDOff, irDYes, irNOff, irNYes, err := rbg.appearanceIndRefs()
	if err != nil {
		return nil, err
	}

	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if rbg.Margin != nil {
		m := rbg.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := rbg.margin(mName)
			if m0 == nil {
				return nil, errors.Errorf("pdfcpu: unknown named margin %s", mName)
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

	cBox := rbg.content.Box()
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	pdf := rbg.pdf
	x, y := pdfcpu.NormalizeCoord(rbg.x, rbg.y, cBox, pdf.origin, false)

	if x == -1 {
		// Center horizontally
		x = cBox.Center().X - r.LL.X
	} else if x > 0 {
		x -= mLeft
		if x < 0 {
			x = 0
		}
	}

	if y == -1 {
		// Center vertically
		y = cBox.Center().Y - r.LL.Y
	} else if y > 0 {
		y -= mBottom
		if y < 0 {
			y = 0
		}
	}

	if x >= 0 {
		x = r.LL.X + x
	}
	if y >= 0 {
		y = r.LL.Y + y
	}

	// Position text horizontally centered for x < 0.
	if x < 0 {
		x = r.LL.X + r.Width()/2
	}

	// Position text vertically centered for y < 0.
	if y < 0 {
		y = r.LL.Y + r.Height()/2
	}

	// Apply offset.
	x += rbg.Dx
	y += rbg.Dy

	rbg.boundingBox = pdfcpu.RectForWidthAndHeight(x, y, rbg.Width, rbg.Width)

	kids := pdfcpu.Array{}

	maxw, firstw := rbg.Buttons.maxLabelWidth(rbg.hor)

	for i, v := range rbg.Buttons.Values {

		r := rbg.rect(float64(i), maxw, firstw)

		d := pdfcpu.Dict(map[string]pdfcpu.Object{
			"Type":    pdfcpu.Name("Annot"),
			"Subtype": pdfcpu.Name("Widget"),
			"F":       pdfcpu.Integer(pdfcpu.AnnPrint),
			"Parent":  parent,
			"AS":      pdfcpu.Name("Off"), // Preselect "Off" or buttonVal
			"Rect":    r.Array(),
			//"T":       id, // required
			//"TU":      id, // Acrobat Reader Hover over field
			"AP": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					"D": pdfcpu.Dict(
						map[string]pdfcpu.Object{
							"Off": *irDOff,
							v:     *irDYes,
						},
					),
					"N": pdfcpu.Dict(
						map[string]pdfcpu.Object{
							"Off": *irNOff,
							v:     *irNYes,
						},
					),
				},
			),
			"BS": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					"S": pdfcpu.Name("I"),
					"W": pdfcpu.Integer(1),
				},
			),
		})

		if v == rbg.Value {
			d["AS"] = pdfcpu.Name(v) // not on MacPreview!
		}

		ir, err := rbg.pdf.XRefTable.IndRefForNewObject(d)
		if err != nil {
			return nil, err
		}

		kids = append(kids, *ir)

		p.Annots = append(p.Annots, d)
		p.AnnotIndRefs = append(p.AnnotIndRefs, *ir)
	}

	return kids, nil
}

func (rbg *RadioButtonGroup) labelFont() (*FormFont, error) {
	// Use inherited named font "label".
	f := rbg.font("label")
	if f == nil {
		return nil, errors.Errorf("pdfcpu: missing named font \"label\"")
	}
	if f.col == nil {
		f.col = &pdfcpu.Black
	}
	return f, nil
}

func (rbg *RadioButtonGroup) calcFont(f *FormFont) error {
	// Use named font.
	fName := f.Name[1:]
	f0 := rbg.font(fName)
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
	if f.col == nil {
		f.col = &pdfcpu.Black
	}
	return nil
}

func (rbg *RadioButtonGroup) prepareFonts() error {

	if rbg.Buttons.Label.Font == nil {
		f, err := rbg.labelFont()
		if err != nil {
			return err
		}
		rbg.Buttons.Label.Font = f
	} else {
		if err := rbg.calcFont(rbg.Buttons.Label.Font); err != nil {
			return err
		}
	}

	if rbg.Label == nil {
		return nil
	}

	if rbg.Label.Font == nil {
		f, err := rbg.labelFont()
		if err != nil {
			return err
		}
		rbg.Label.Font = f
	} else {
		if err := rbg.calcFont(rbg.Label.Font); err != nil {
			return err
		}
	}

	return nil
}

func (rbg *RadioButtonGroup) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap, fields *pdfcpu.Array) error {

	id := pdfcpu.StringLiteral(pdfcpu.EncodeUTF16String(rbg.ID))

	d := pdfcpu.Dict(
		map[string]pdfcpu.Object{
			"FT": pdfcpu.Name("Btn"),
			"Ff": pdfcpu.Integer(FieldNoToggleToOff + FieldRadio),
			"T":  id,                 // required
			"TU": id,                 // Acrobat Reader Hover over field
			"V":  pdfcpu.Name("Off"), // -> extract set radio button, set for preselection
			"DV": pdfcpu.Name("Off"),
		},
	)

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	if err := rbg.prepareFonts(); err != nil {
		return err
	}

	kids, err := rbg.renderRadioButtonFields(p, *ir)
	if err != nil {
		return err
	}

	if err := rbg.renderLabels(p, pageNr, fonts); err != nil {
		return err
	}

	d["Kids"] = kids

	*fields = append(*fields, *ir)

	return nil
}

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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Note:
// Mac Preview is unable to save modified radio buttons:
// The form field holding the kid terminal fields for each button does not get the current value assigned to V.
// Instead Preview sets V in the widget annotation that corresponds to the selected radio button.

// RadioButtonGroup represents a set of radio buttons including positioned labels.
type RadioButtonGroup struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Tip             string
	Value           string // checked button
	Default         string
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	boundingBox     *types.Rectangle
	Orientation     string
	hor             bool
	Dx, Dy          float64
	Margin          *Margin // applied to content box
	BackgroundColor string  `json:"bgCol"`
	bgCol           *color.SimpleColor
	Buttons         *Buttons
	RTL             bool
	Tab             int
	Locked          bool
	Debug           bool
	Hide            bool
}

func (rbg *RadioButtonGroup) Rtl() bool {
	if rbg.Buttons == nil {
		return false
	}
	return rbg.Buttons.Rtl()
}

func (rbg *RadioButtonGroup) validateID() error {
	if rbg.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if rbg.pdf.DuplicateField(rbg.ID) {
		return errors.Errorf("pdfcpu: duplicate form field: %s", rbg.ID)
	}
	rbg.pdf.FieldIDs[rbg.ID] = true
	return nil
}

func (rbg *RadioButtonGroup) validatePosition() error {
	if rbg.Position[0] < 0 || rbg.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", rbg.ID)
	}
	rbg.x, rbg.y = rbg.Position[0], rbg.Position[1]
	return nil
}

func parseRadioButtonOrientation(s string) (types.Orientation, error) {
	var o types.Orientation
	switch strings.ToLower(s) {
	case "h", "hor", "horizontal":
		o = types.Horizontal
	case "v", "vert", "vertical":
		o = types.Vertical
	default:
		return o, errors.Errorf("pdfcpu: unknown radiobutton orientation (hor, vert): %s", s)
	}
	return o, nil
}

func (rbg *RadioButtonGroup) validateOrientation() error {
	rbg.hor = true
	if rbg.Orientation != "" {
		o, err := parseRadioButtonOrientation(rbg.Orientation)
		if err != nil {
			return err
		}
		rbg.hor = o == types.Horizontal
	}
	return nil
}

func (rbg *RadioButtonGroup) validateWidth() error {
	if rbg.Width <= 0 {
		return errors.Errorf("pdfcpu: field: %s width <= 0", rbg.ID)
	}
	return nil
}

func (rbg *RadioButtonGroup) validateMargin() error {
	if rbg.Margin != nil {
		if err := rbg.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (rbg *RadioButtonGroup) validateLabel() error {
	if rbg.Label != nil {
		rbg.Label.pdf = rbg.pdf
		if err := rbg.Label.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (rbg *RadioButtonGroup) validateButtonsDefaultAndValue() error {
	if rbg.Buttons == nil {
		return errors.New("pdfcpu: radiobuttongroup missing buttons")
	}
	rbg.Buttons.pdf = rbg.pdf
	return rbg.Buttons.validate(rbg.Default, rbg.Value)
}

func (rbg *RadioButtonGroup) validateTab() error {
	if rbg.Tab < 0 {
		return errors.Errorf("pdfcpu: field: %s negative tab value", rbg.ID)
	}
	if rbg.Tab == 0 {
		return nil
	}
	page := rbg.content.page
	if page.Tabs == nil {
		page.Tabs = types.IntSet{}
	} else {
		if page.Tabs[rbg.Tab] {
			return errors.Errorf("pdfcpu: field: %s duplicate tab value %d", rbg.ID, rbg.Tab)
		}
	}
	page.Tabs[rbg.Tab] = true
	return nil
}

func (rbg *RadioButtonGroup) validate() error {

	if err := rbg.validateID(); err != nil {
		return err
	}

	if err := rbg.validatePosition(); err != nil {
		return err
	}

	if err := rbg.validateOrientation(); err != nil {
		return err
	}

	if err := rbg.validateWidth(); err != nil {
		return err
	}

	if err := rbg.validateMargin(); err != nil {
		return err
	}

	if err := rbg.validateLabel(); err != nil {
		return err
	}

	if err := rbg.validateButtonsDefaultAndValue(); err != nil {
		return err
	}

	return rbg.validateTab()
}

func (rbg *RadioButtonGroup) calcFont() error {

	if rbg.Label != nil {
		f, err := rbg.content.calcLabelFont(rbg.Label.Font)
		if err != nil {
			return err
		}
		rbg.Label.Font = f
	}

	if rbg.Buttons.Label != nil {
		f, err := rbg.content.calcLabelFont(rbg.Buttons.Label.Font)
		if err != nil {
			return err
		}
		rbg.Buttons.Label.Font = f
	}

	return nil
}

func (rbg *RadioButtonGroup) margin(name string) *Margin {
	return rbg.content.namedMargin(name)
}

func (rbg *RadioButtonGroup) prepareMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBot, mLeft := 0., 0., 0., 0.

	if rbg.Margin != nil {

		m := rbg.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := rbg.margin(mName)
			if m0 == nil {
				return mTop, mRight, mBot, mLeft, errors.Errorf("pdfcpu: unknown named margin %s", mName)
			}
			m.mergeIn(m0)
		}

		if m.Width > 0 {
			mTop = m.Width
			mRight = m.Width
			mBot = m.Width
			mLeft = m.Width
		} else {
			mTop = m.Top
			mRight = m.Right
			mBot = m.Bottom
			mLeft = m.Left
		}
	}

	return mTop, mRight, mBot, mLeft, nil
}

func (rbg *RadioButtonGroup) buttonLabelPosition(i int) (float64, float64) {
	rbw := rbg.Width
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)
	bg := float64(rbg.Buttons.Gap)
	maxWidth := rbg.Buttons.maxWidth

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}
		var x float64
		if rbg.Buttons.Label.HorAlign == types.AlignLeft {
			x = rbg.boundingBox.LL.X + rbw + bg + float64(i)*(rbw+w)
		}
		if rbg.Buttons.Label.HorAlign == types.AlignRight {
			x = rbg.boundingBox.LL.X + rbg.Buttons.widths[0]
			if i > 0 {
				x += float64(i) * (rbw + w)
			}
		}
		return x, rbg.boundingBox.LL.Y + rbg.boundingBox.Height()/2
	}

	if maxWidth > w {
		w = maxWidth
	}
	var dx float64
	if rbg.Buttons.Label.HorAlign == types.AlignLeft {
		dx += rbw + bg
	}
	if rbg.Buttons.Label.HorAlign == types.AlignRight {
		dx += w
	}
	dy := float64(i) * (rbw + g)
	return rbg.boundingBox.LL.X + dx, rbg.boundingBox.LL.Y - dy
}

func (rbg *RadioButtonGroup) renderButtonLabels(p *model.Page, pageNr int, fonts model.FontMap) error {
	l := rbg.Buttons.Label

	fontName := l.Font.Name
	fontLang := l.Font.Lang
	fontSize := l.Font.Size
	col := l.Font.col

	id, err := rbg.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := model.TextDescriptor{
		FontName: fontName,
		FontKey:  id,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL,
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.BgCol != nil {
		td.ShowTextBB = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.BgCol
	}

	td.HAlign, td.VAlign = l.HorAlign, types.AlignBottom

	for i, v := range rbg.Buttons.Values {
		td.Text = v
		td.X, td.Y = rbg.buttonLabelPosition(i)
		if rbg.hor {
			td.VAlign = types.AlignMiddle
		}
		model.WriteColumn(rbg.pdf.XRefTable, p.Buf, p.MediaBox, nil, td, 0)
	}

	return nil
}

func (rbg *RadioButtonGroup) buttonGroupBB() *types.Rectangle {
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)
	bg := float64(rbg.Buttons.Gap)
	maxWidth := rbg.Buttons.maxWidth

	rbSize := rbg.Width
	rbCount := float64(len(rbg.Buttons.Values))

	if rbg.hor {
		if maxWidth+g > w {
			w = maxWidth + g
		}
		width := (rbCount-1)*(w+rbSize+bg) + rbSize
		if rbg.Buttons.Label.HorAlign == types.AlignRight {
			width += rbg.Buttons.widths[0]
		}
		if rbg.Buttons.Label.HorAlign == types.AlignLeft {
			width += rbg.Buttons.widths[len(rbg.Buttons.widths)-1]
		}
		return types.RectForWidthAndHeight(rbg.boundingBox.LL.X, rbg.boundingBox.LL.Y, width, rbSize)
	}

	if maxWidth > w {
		w = maxWidth
	}
	y := rbg.boundingBox.LL.Y - (rbCount-1)*(rbSize+g)
	h := rbSize + (rbCount-1)*(rbSize+g)

	return types.RectForWidthAndHeight(rbg.boundingBox.LL.X, y, w+rbSize+bg, h)
}

func labelPos(
	relPos types.RelPosition,
	horAlign types.HAlignment,
	boundingBox *types.Rectangle,
	labelHeight, w, g float64, multiline bool) (float64, float64) {

	var x, y float64

	switch relPos {

	case types.RelPosLeft:
		x = boundingBox.LL.X - g
		if horAlign == types.AlignLeft {
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

	case types.RelPosRight:
		x = boundingBox.UR.X + g
		if horAlign == types.AlignRight {
			x += w
		}
		if multiline {
			y = boundingBox.UR.Y - labelHeight
		} else {
			y = boundingBox.LL.Y
		}

	case types.RelPosTop:
		y = boundingBox.UR.Y + g
		x = boundingBox.LL.X
		if horAlign == types.AlignRight {
			x += boundingBox.Width()
		} else if horAlign == types.AlignCenter {
			x += boundingBox.Width() / 2
		}

	case types.RelPosBottom:
		y = boundingBox.LL.Y - g - labelHeight
		x = boundingBox.LL.X
		if horAlign == types.AlignRight {
			x += boundingBox.Width()
		} else if horAlign == types.AlignCenter {
			x += boundingBox.Width() / 2
		}

	}

	return x, y
}

func (rbg *RadioButtonGroup) rect(i int) *types.Rectangle {
	rbw := rbg.Width
	g := float64(rbg.Buttons.Label.Gap)
	w := float64(rbg.Buttons.Label.Width)
	bg := float64(rbg.Buttons.Gap)

	if rbg.hor {
		if rbg.Buttons.maxWidth+g > w {
			w = rbg.Buttons.maxWidth + g
		}
		var x float64
		if rbg.Buttons.Label.HorAlign == types.AlignLeft {
			x = rbg.boundingBox.LL.X + float64(i)*(rbw+w)
		}
		if rbg.Buttons.Label.HorAlign == types.AlignRight {
			x = rbg.boundingBox.LL.X + rbg.Buttons.widths[0] + bg
			if i > 0 {
				x += float64(i) * (rbw + w)
			}
		}
		return types.RectForWidthAndHeight(x, rbg.boundingBox.LL.Y, rbw, rbw)
	}
	dx := 0.
	if rbg.Buttons.Label.HorAlign == types.AlignRight {
		if rbg.Buttons.maxWidth > w {
			w = rbg.Buttons.maxWidth
		}
		dx = w + bg
	}
	dy := float64(i) * (rbw + g)
	return types.RectForWidthAndHeight(rbg.boundingBox.LL.X+dx, rbg.boundingBox.LL.Y-dy, rbw, rbw)
}

func (rbg *RadioButtonGroup) irDOff(asWidth float64, flip bool) (*types.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[asWidth]
	if found {
		if !flip && ap.irDOffL != nil {
			return ap.irDOffL, nil
		}
		if flip && ap.irDOffR != nil {
			return ap.irDOffR, nil
		}
	}

	f := .5523
	r := w / 2
	r1 := r - .5
	dx := r
	if flip {
		dx = asWidth - r
	}

	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "q 0.5 g 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	fmt.Fprintf(buf, "f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	fmt.Fprint(buf, "s Q ")

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, asWidth, w))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.RadioBtnAPs[asWidth] = ap
	}
	if !flip {
		ap.irDOffL = ir
	}
	if flip {
		ap.irDOffR = ir
	}

	return ir, nil
}

func (rbg *RadioButtonGroup) irDYes(asWidth float64, flip bool) (*types.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[asWidth]
	if found {
		if !flip && ap.irDYesL != nil {
			return ap.irDYesL, nil
		}
		if flip && ap.irDYesR != nil {
			return ap.irDYesR, nil
		}
	}

	f := .5523
	r := w / 2
	r1 := r - .5
	r2 := r / 2
	dx := r
	if flip {
		dx = asWidth - r
	}

	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "q 0.5 g 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	fmt.Fprintf(buf, "f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	fmt.Fprintf(buf, "s Q 0 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r2)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r2, f*r2, f*r2, r2, 0., r2)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r2, r2, -r2, f*r2, -r2, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r2, -f*r2, -f*r2, -r2, .0, -r2)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r2, -r2, r2, -f*r2, r2, 0.)
	fmt.Fprint(buf, "f Q ")

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, asWidth, w))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.RadioBtnAPs[asWidth] = ap
	}
	if !flip {
		ap.irDYesL = ir
	}
	if flip {
		ap.irDYesR = ir
	}

	return ir, nil
}

func (rbg *RadioButtonGroup) irNOff(asWidth float64, flip bool, bgCol *color.SimpleColor) (*types.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[asWidth]
	if found {
		if !flip && ap.irNOffL != nil {
			return ap.irNOffL, nil
		}
		if flip && ap.irNOffR != nil {
			return ap.irNOffR, nil
		}
	}

	f := .5523
	r := w / 2
	r1 := r - .5
	dx := r
	if flip {
		dx = asWidth - r
	}

	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "q ")
	if bgCol != nil {
		fmt.Fprintf(buf, "%.2f %.2f %.2f rg ", bgCol.R, bgCol.G, bgCol.B)
	} else {
		fmt.Fprint(buf, "1 g ")
	}

	fmt.Fprintf(buf, "1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	fmt.Fprintf(buf, "f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	fmt.Fprint(buf, "s Q ")

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, asWidth, w))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.RadioBtnAPs[asWidth] = ap
	}
	if !flip {
		ap.irNOffL = ir
	}
	if flip {
		ap.irNOffR = ir
	}

	return ir, nil
}

func (rbg *RadioButtonGroup) irNYes(asWidth float64, flip bool, bgCol *color.SimpleColor) (*types.IndirectRef, error) {

	w := rbg.Width

	ap, found := rbg.pdf.RadioBtnAPs[asWidth]
	if found {
		if !flip && ap.irNYesL != nil {
			return ap.irNYesL, nil
		}
		if flip && ap.irNYesR != nil {
			return ap.irNYesR, nil
		}
	}

	f := .5523
	r := w / 2
	r1 := r - .5
	r2 := r / 2
	dx := r
	if flip {
		dx = asWidth - r
	}

	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "q ")
	if bgCol != nil {
		fmt.Fprintf(buf, "%.2f %.2f %.2f rg ", bgCol.R, bgCol.G, bgCol.B)
	} else {
		fmt.Fprint(buf, "1 g ")
	}

	fmt.Fprintf(buf, "1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r, f*r, f*r, r, 0., r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r, r, -r, f*r, -r, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r, -f*r, -f*r, -r, .0, -r)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r, -r, r, -f*r, r, 0.)
	fmt.Fprintf(buf, "f Q q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r1, f*r1, f*r1, r1, 0., r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r1, r1, -r1, f*r1, -r1, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r1, -f*r1, -f*r1, -r1, .0, -r1)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r1, -r1, r1, -f*r1, r1, 0.)
	fmt.Fprintf(buf, "s Q 0 g q 1 0 0 1 %.2f %.2f cm %.2f 0 m ", dx, r, r2)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", r2, f*r2, f*r2, r2, 0., r2)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -f*r2, r2, -r2, f*r2, -r2, 0.)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", -r2, -f*r2, -f*r2, -r2, .0, -r2)
	fmt.Fprintf(buf, "%.3f %.3f %.3f %.3f %.3f %.3f c ", f*r2, -r2, r2, -f*r2, r2, 0.)
	fmt.Fprint(buf, "f Q ")

	sd, err := rbg.pdf.XRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, asWidth, w))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		rbg.pdf.RadioBtnAPs[asWidth] = ap
	}
	if !flip {
		ap.irNYesL = ir
	}
	if flip {
		ap.irNYesR = ir
	}

	return ir, nil
}

func (rbg *RadioButtonGroup) appearanceIndRefs(flip bool, bgCol *color.SimpleColor) (
	*types.IndirectRef, *types.IndirectRef, *types.IndirectRef, *types.IndirectRef, error) {

	w := rbg.Width

	irDOff, err := rbg.irDOff(w, flip)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irDYes, err := rbg.irDYes(w, flip)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNOff, err := rbg.irNOff(w, flip, bgCol)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNYes, err := rbg.irNYes(w, flip, bgCol)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return irDOff, irDYes, irNOff, irNYes, nil
}

func (rbg *RadioButtonGroup) prepareButtonDict(r *types.Rectangle, v string, parent types.IndirectRef, irDOff, irDYes, irNOff, irNYes *types.IndirectRef) (*types.IndirectRef, types.Dict, error) {

	/*	Note: Mac Preview seems to have a problem saving radio buttons.
		1) Once saved in Mac Preview selected radio buttons don't get rendered in Mac Preview whereas Adobe Reader renders them w/o problem.
		2) Preselected radio buttons remain sticky after saving across Mac Preview and Adobe Reader.
	*/

	s := types.EncodeName(v)

	as := types.Name("Off")

	v1 := rbg.Default
	if rbg.Value != "" {
		v1 = rbg.Value
	}
	if v == v1 {
		as = types.Name(s)
	}

	d := types.Dict(map[string]types.Object{
		"Type":    types.Name("Annot"),
		"Subtype": types.Name("Widget"),
		"F":       types.Integer(model.AnnPrint),
		"Parent":  parent,
		"AS":      as,
		"Rect":    r.Array(),
		"AP": types.Dict(
			map[string]types.Object{
				"D": types.Dict(
					map[string]types.Object{
						"Off": *irDOff,
						s:     *irDYes,
					},
				),
				"N": types.Dict(
					map[string]types.Object{
						"Off": *irNOff,
						s:     *irNYes,
					},
				),
			},
		),
		"BS": types.Dict(
			map[string]types.Object{
				"S": types.Name("I"),
				"W": types.Integer(1),
			},
		),
	})

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(d)

	return ir, d, err
}

func (rbg *RadioButtonGroup) renderRadioButtonFields(p *model.Page, parent types.IndirectRef) (types.Array, error) {
	flip := rbg.Buttons.Label.HorAlign == types.AlignRight
	kids := types.Array{}

	bgCol := rbg.bgCol
	if bgCol == nil {
		bgCol = rbg.content.page.bgCol
		if bgCol == nil {
			bgCol = rbg.pdf.bgCol
		}
	}

	for i := 0; i < len(rbg.Buttons.Values); i++ {

		irDOff, irDYes, irNOff, irNYes, err := rbg.appearanceIndRefs(flip, bgCol)
		if err != nil {
			return nil, err
		}

		r := rbg.rect(i)
		v := rbg.Buttons.Values[i]

		ir, _, err := rbg.prepareButtonDict(r, v, parent, irDOff, irDYes, irNOff, irNYes)
		if err != nil {
			return nil, err
		}

		kids = append(kids, *ir)
	}

	return kids, nil
}

func (rbg *RadioButtonGroup) bbox() *types.Rectangle {
	if rbg.Label == nil {
		return rbg.Buttons.boundingBox.Clone()
	}

	l := rbg.Label
	var r *types.Rectangle
	x := l.td.X

	switch l.td.HAlign {
	case types.AlignCenter:
		x -= float64(l.Width) / 2
	case types.AlignRight:
		x -= float64(l.Width)
	}

	y := l.td.Y
	if rbg.hor {
		y -= rbg.boundingBox.Height() / 2
	}
	r = types.RectForWidthAndHeight(x, y, float64(l.Width), l.height)

	return model.CalcBoundingBoxForRects(rbg.Buttons.boundingBox, r)
}

func (rbg *RadioButtonGroup) prepareRectLL(mTop, mRight, mBottom, mLeft float64) (float64, float64) {
	return rbg.content.calcPosition(rbg.x, rbg.y, rbg.Dx, rbg.Dy, mTop, mRight, mBottom, mLeft)
}

func (rbg *RadioButtonGroup) prepLabel(p *model.Page, pageNr int, fonts model.FontMap) error {

	if rbg.Label == nil {
		return nil
	}

	l := rbg.Label
	v := l.Value
	w := float64(l.Width)
	g := float64(l.Gap)

	f := l.Font
	fontName, fontLang, col := f.Name, f.Lang, f.col

	id, err := rbg.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := model.TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  id,
		FontSize: f.Size,
		HAlign:   l.HorAlign,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL,
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.BgCol != nil {
		td.ShowTextBB = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.BgCol
	}

	bb := model.WriteMultiLine(rbg.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	l.height = bb.Height()
	if bb.Width() > w {
		w = bb.Width()
		l.Width = int(bb.Width())
	}
	td.X, td.Y = labelPos(l.relPos, l.HorAlign, rbg.buttonGroupBB(), l.height, w, g, !rbg.hor)
	td.VAlign = types.AlignBottom
	if rbg.hor {
		td.Y += rbg.boundingBox.Height() / 2
		td.VAlign = types.AlignMiddle
	}

	l.td = &td

	return nil
}

func (rbg *RadioButtonGroup) prepForRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := rbg.calcFont(); err != nil {
		return err
	}

	rbg.Buttons.calcLabelWidths(rbg.hor)

	mTop, mRight, mBottom, mLeft, err := rbg.prepareMargin()
	if err != nil {
		return err
	}

	x, y := rbg.prepareRectLL(mTop, mRight, mBottom, mLeft)

	rbg.boundingBox = types.RectForWidthAndHeight(x, y, rbg.Width, rbg.Width)

	rbg.Buttons.boundingBox = rbg.buttonGroupBB()

	return rbg.prepLabel(p, pageNr, fonts)
}

func (rbg *RadioButtonGroup) prepareDict(p *model.Page, pageNr int, fonts model.FontMap) (*types.IndirectRef, types.Array, error) {

	rbg.renderButtonLabels(p, pageNr, fonts)

	id, err := types.EscapeUTF16String(rbg.ID)
	if err != nil {
		return nil, nil, err
	}

	ff := FieldNoToggleToOff + FieldRadio
	if rbg.Locked {
		// Note: unsupported in Mac Preview
		ff += FieldReadOnly
	}

	d := types.Dict(
		map[string]types.Object{
			"FT": types.Name("Btn"),
			"Ff": types.Integer(ff),
			"T":  types.StringLiteral(*id),
		},
	)

	v := types.Name("Off")
	if rbg.Value != "" {
		s := types.EncodeName(rbg.Value)
		v = types.Name(s)
	}

	if rbg.Default != "" {
		s := types.EncodeName(rbg.Default)
		d["DV"] = types.Name(s)
		if rbg.Value == "" {
			v = types.Name(s)
		}
	}

	d["V"] = v

	if rbg.Tip != "" {
		tu, err := types.EscapeUTF16String(rbg.Tip)
		if err != nil {
			return nil, nil, err
		}
		d["TU"] = types.StringLiteral(*tu)
	}

	ir, err := rbg.pdf.XRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, nil, err
	}

	kids, err := rbg.renderRadioButtonFields(p, *ir)
	if err != nil {
		return nil, nil, err
	}

	d["Kids"] = kids

	return ir, kids, nil
}

func (rbg *RadioButtonGroup) doRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	ir, kids, err := rbg.prepareDict(p, pageNr, fonts)
	if err != nil {
		return err
	}

	ann := model.FieldAnnotation{IndRef: ir, Kids: kids}
	if rbg.Tab > 0 {
		p.AnnotTabs[rbg.Tab] = ann
	} else {
		p.Annots = append(p.Annots, ann)
	}

	if rbg.Label != nil {
		model.WriteColumn(rbg.pdf.XRefTable, p.Buf, p.MediaBox, nil, *rbg.Label.td, 0)
	}

	if rbg.Debug || rbg.pdf.Debug {
		rbg.pdf.highlightPos(p.Buf, rbg.boundingBox.LL.X, rbg.boundingBox.LL.Y, rbg.content.Box())
	}

	return nil
}

func (rbg *RadioButtonGroup) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := rbg.prepForRender(p, pageNr, fonts); err != nil {
		return err
	}

	return rbg.doRender(p, pageNr, fonts)
}

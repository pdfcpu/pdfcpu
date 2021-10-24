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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type CheckBox struct {
	pdf         *PDF
	content     *Content
	Label       *TextFieldLabel
	ID          string
	Value       bool       // checked state
	Position    [2]float64 `json:"pos"` // x,y
	x, y        float64
	Width       float64
	Dx, Dy      float64
	boundingBox *pdfcpu.Rectangle
	Margin      *Margin // applied to content box
	Hide        bool
}

type AP struct {
	irDOff, irDYes *pdfcpu.IndirectRef
	irNOff, irNYes *pdfcpu.IndirectRef
}

func (cb *CheckBox) validate() error {

	if cb.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if cb.pdf.FieldIDs[cb.ID] {
		return errors.Errorf("pdfcpu: duplicate form field: %s", cb.ID)
	}
	cb.pdf.FieldIDs[cb.ID] = true

	if cb.Position[0] < 0 || cb.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", cb.ID)
	}
	cb.x, cb.y = cb.Position[0], cb.Position[1]

	if cb.Width <= 0 {
		return errors.Errorf("pdfcpu: field: %s width <= 0", cb.ID)
	}

	if cb.Margin != nil {
		if err := cb.Margin.validate(); err != nil {
			return err
		}
	}

	if cb.Label != nil {
		cb.Label.pdf = cb.pdf
		if err := cb.Label.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (cb *CheckBox) font(name string) *FormFont {
	return cb.content.namedFont(name)
}

func (cb *CheckBox) margin(name string) *Margin {
	return cb.content.namedMargin(name)
}

func (cb *CheckBox) labelPos(w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := cb.boundingBox, cb.Label.horAlign

	switch cb.Label.relPos {

	case pdfcpu.RelPosLeft:
		x = bb.LL.X - g
		if horAlign == pdfcpu.AlignLeft {
			x -= w
		}
		y = bb.LL.Y

	case pdfcpu.RelPosRight:
		x = bb.UR.X + g
		if horAlign == pdfcpu.AlignRight {
			x += w
		}
		y = bb.LL.Y

	case pdfcpu.RelPosTop:
		y = bb.UR.Y + g
		x = bb.LL.X
		if horAlign == pdfcpu.AlignRight {
			x += bb.Width()
		} else if horAlign == pdfcpu.AlignCenter {
			x += bb.Width() / 2
		}

	case pdfcpu.RelPosBottom:
		y = bb.LL.Y - g - bb.Height()
		x = bb.LL.X
		if horAlign == pdfcpu.AlignRight {
			x += bb.Width()
		} else if horAlign == pdfcpu.AlignCenter {
			x += bb.Width() / 2
		}
	}

	return x, y
}

func (cb *CheckBox) renderLabel(r *pdfcpu.Rectangle, p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap) error {

	if cb.Label == nil {
		return nil
	}

	l := cb.Label
	v := "Default"
	if l.Value != "" {
		v = l.Value
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	var f *FormFont

	if l.Font != nil {
		f = l.Font
		if f.Name[0] == '$' {
			// use named font
			fName := f.Name[1:]
			f0 := cb.font(fName)
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
	} else {
		// Use inherited named font "label".
		f = cb.font("label")
		if f == nil {
			return errors.Errorf("pdfcpu: missing named font \"label\"")
		}
	}

	if f.col == nil {
		f.col = &pdfcpu.Black
	}

	fontName := f.Name
	fontSize := f.Size
	col := f.col

	id, err := cb.pdf.idForFontName(fontName, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := pdfcpu.TextDescriptor{
		Text:     v,
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
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	td.X, td.Y = cb.labelPos(w, g)

	td.HAlign, td.VAlign = l.horAlign, pdfcpu.AlignBottom

	pdfcpu.WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func (cb *CheckBox) ensureZapfDingbats(fonts pdfcpu.FontMap) (*pdfcpu.IndirectRef, error) {
	pdf := cb.pdf
	fontName := "ZapfDingbats"
	font, ok := fonts[fontName]
	if ok {
		if font.Res.IndRef != nil {
			return font.Res.IndRef, nil
		}
		ir, err := pdfcpu.EnsureFontDict(pdf.XRefTable, fontName, false, nil)
		if err != nil {
			return nil, err
		}
		font.Res.IndRef = ir
		fonts[fontName] = font
		return ir, nil
	}

	var (
		indRef *pdfcpu.IndirectRef
		err    error
	)

	if pdf.Optimize != nil {

		for objNr, fo := range pdf.Optimize.FormFontObjects {
			//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
			if fontName == fo.FontName {
				//fmt.Println("Match!")
				indRef = pdfcpu.NewIndirectRef(objNr, 0)
				break
			}
		}

		if indRef == nil {
			for objNr, fo := range pdf.Optimize.FontObjects {
				//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
				if fontName == fo.FontName {
					//fmt.Println("Match!")
					indRef = pdfcpu.NewIndirectRef(objNr, 0)
					break
				}
			}
		}
	}

	if indRef == nil {
		indRef, err = pdfcpu.EnsureFontDict(pdf.XRefTable, fontName, false, nil)
		if err != nil {
			return nil, err
		}
	}

	font.Res = pdfcpu.Resource{IndRef: indRef}
	fonts[fontName] = font
	return indRef, nil
}

func (cb *CheckBox) irNOff() (*pdfcpu.IndirectRef, error) {

	ap, found := cb.pdf.CheckBoxAPs[cb.Width]
	if found && ap.irNOff != nil {
		return ap.irNOff, nil
	}

	buf := fmt.Sprintf("q 1 g 0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re s Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	sd, err := cb.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := cb.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		cb.pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irNOff = ir

	return ir, nil
}

func (cb *CheckBox) irNYes(fonts pdfcpu.FontMap) (*pdfcpu.IndirectRef, error) {

	ap, found := cb.pdf.CheckBoxAPs[cb.Width]
	if found && ap.irNYes != nil {
		return ap.irNYes, nil
	}

	s, x, y := 14.532/18, 2.853/18, 4.081/18
	buf := fmt.Sprintf("q 1 g 0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re s Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	buf += fmt.Sprintf("q 1 1 %.1f %.1f re W n BT /F0 %f Tf %f %f Td (4) Tj ET Q ", cb.Width-2, cb.Width-2, s*cb.Width, x*cb.Width, y*cb.Width)
	sd, err := cb.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	ir, err := cb.ensureZapfDingbats(fonts)
	if err != nil {
		return nil, err
	}

	d := pdfcpu.Dict(
		map[string]pdfcpu.Object{
			"Font": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					"F0": *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err = cb.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		cb.pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irNYes = ir

	return ir, nil
}

func (cb *CheckBox) irDOff() (*pdfcpu.IndirectRef, error) {

	ap, found := cb.pdf.CheckBoxAPs[cb.Width]
	if found && ap.irDOff != nil {
		return ap.irDOff, nil
	}

	buf := fmt.Sprintf("q 0.75293 g 0 0  %.1f %.1f re f 0.5 0.5 %.1f %.1f re se Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	sd, err := cb.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := cb.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		cb.pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irDOff = ir

	return ir, nil
}

func (cb *CheckBox) irDYes(fonts pdfcpu.FontMap) (*pdfcpu.IndirectRef, error) {

	ap, found := cb.pdf.CheckBoxAPs[cb.Width]
	if found && ap.irDYes != nil {
		return ap.irDYes, nil
	}

	s, x, y := 14.532/18, 2.853/18, 4.081/18
	buf := fmt.Sprintf("q 0.75293 g 0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re se Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	buf += fmt.Sprintf("q 1 1 %.1f %.1f re W n BT /F0 %f Tf %f %f Td (4) Tj ET Q ", cb.Width-2, cb.Width-2, s*cb.Width, x*cb.Width, y*cb.Width)
	sd, _ := cb.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	ir, err := cb.ensureZapfDingbats(fonts)
	if err != nil {
		return nil, err
	}

	d := pdfcpu.Dict(
		map[string]pdfcpu.Object{
			"Font": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					"F0": *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err = cb.pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		cb.pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irDYes = ir

	return ir, nil
}

func (cb *CheckBox) appearanceIndRefs(fonts pdfcpu.FontMap) (
	*pdfcpu.IndirectRef, *pdfcpu.IndirectRef, *pdfcpu.IndirectRef, *pdfcpu.IndirectRef, error) {

	irDOff, err := cb.irDOff()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irDYes, err := cb.irDYes(fonts)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNOff, err := cb.irNOff()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNYes, err := cb.irNYes(fonts)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return irDOff, irDYes, irNOff, irNYes, nil
}

func (cb *CheckBox) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap) error {

	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if cb.Margin != nil {
		m := cb.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := cb.margin(mName)
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

	cBox := cb.content.Box()
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	pdf := cb.pdf
	x, y := pdfcpu.NormalizeCoord(cb.x, cb.y, cBox, pdf.origin, false)

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
	x += cb.Dx
	y += cb.Dy

	cb.boundingBox = pdfcpu.RectForWidthAndHeight(x, y, cb.Width, cb.Width)

	id := pdfcpu.StringLiteral(pdfcpu.EncodeUTF16String(cb.ID))

	v := "Off"
	if cb.Value {
		v = "Yes"
	}

	irDOff, irDYes, irNOff, irNYes, err := cb.appearanceIndRefs(fonts)
	if err != nil {
		return err
	}

	d := pdfcpu.Dict(
		map[string]pdfcpu.Object{
			"Type":    pdfcpu.Name("Annot"),
			"Subtype": pdfcpu.Name("Widget"),
			"FT":      pdfcpu.Name("Btn"),
			"Rect":    cb.boundingBox.Array(),
			"F":       pdfcpu.Integer(pdfcpu.AnnPrint),
			"T":       id,
			"TU":      id,
			"V":       pdfcpu.Name(v), // -> extractValue: Off or Yes
			"AS":      pdfcpu.Name(v),
			"AP": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					"D": pdfcpu.Dict(
						map[string]pdfcpu.Object{
							"Off": *irDOff,
							"Yes": *irDYes,
						},
					),
					"N": pdfcpu.Dict(
						map[string]pdfcpu.Object{
							"Off": *irNOff,
							"Yes": *irNYes,
						},
					),
				},
			),
			"MK": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					"BC": pdfcpu.NewNumberArray(0.0),
					"BG": pdfcpu.NewNumberArray(0.0),
				},
			),
		},
	)

	p.Annots = append(p.Annots, d)

	return cb.renderLabel(r, p, pageNr, fonts)
}

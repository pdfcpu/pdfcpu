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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// CheckBox represents a form checkbox including a positioned label.
type CheckBox struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Tip             string
	Value           bool // checked state
	Default         bool
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Dx, Dy          float64
	boundingBox     *types.Rectangle
	Margin          *Margin // applied to content box
	BackgroundColor string  `json:"bgCol"`
	bgCol           *color.SimpleColor
	Tab             int
	Locked          bool
	Debug           bool
	Hide            bool
}

type AP struct {
	irDOffL, irDYesL *types.IndirectRef
	irNOffL, irNYesL *types.IndirectRef
	irDOffR, irDYesR *types.IndirectRef
	irNOffR, irNYesR *types.IndirectRef
}

func (cb *CheckBox) validateID() error {
	if cb.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if cb.pdf.DuplicateField(cb.ID) {
		return errors.Errorf("pdfcpu: duplicate form field: %s", cb.ID)
	}
	cb.pdf.FieldIDs[cb.ID] = true
	return nil
}

func (cb *CheckBox) validatePosition() error {
	if cb.Position[0] < 0 || cb.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", cb.ID)
	}
	cb.x, cb.y = cb.Position[0], cb.Position[1]
	return nil
}

func (cb *CheckBox) validateMargin() error {
	if cb.Margin != nil {
		if err := cb.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cb *CheckBox) validateWidth() error {
	if cb.Width <= 0 {
		return errors.Errorf("pdfcpu: field: %s width <= 0", cb.ID)
	}
	return nil
}

func (cb *CheckBox) validateLabel() error {
	if cb.Label != nil {
		cb.Label.pdf = cb.pdf
		if err := cb.Label.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cb *CheckBox) validateTab() error {
	if cb.Tab < 0 {
		return errors.Errorf("pdfcpu: field: %s negative tab value", cb.ID)
	}
	if cb.Tab == 0 {
		return nil
	}
	page := cb.content.page
	if page.Tabs == nil {
		page.Tabs = types.IntSet{}
	} else {
		if page.Tabs[cb.Tab] {
			return errors.Errorf("pdfcpu: field: %s duplicate tab value %d", cb.ID, cb.Tab)
		}
	}
	page.Tabs[cb.Tab] = true
	return nil
}

func (cb *CheckBox) validate() error {

	if err := cb.validateID(); err != nil {
		return err
	}

	if err := cb.validatePosition(); err != nil {
		return err
	}

	if err := cb.validateWidth(); err != nil {
		return err
	}

	if err := cb.validateMargin(); err != nil {
		return err
	}

	if err := cb.validateLabel(); err != nil {
		return err
	}

	return cb.validateTab()
}

func (cb *CheckBox) margin(name string) *Margin {
	return cb.content.namedMargin(name)
}

func (cb *CheckBox) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if cb.Margin != nil {
		m := cb.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := cb.margin(mName)
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

func (cb *CheckBox) labelPos(labelHeight, w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := cb.boundingBox, cb.Label.HorAlign

	switch cb.Label.relPos {

	case types.RelPosLeft:
		x = bb.LL.X - g
		if horAlign == types.AlignLeft {
			x -= w
			if x < 0 {
				x = 0
			}
		}
		y = bb.LL.Y

	case types.RelPosRight:
		x = bb.UR.X + g
		if horAlign == types.AlignRight {
			x += w
		}
		y = bb.LL.Y

	case types.RelPosTop:
		y = bb.UR.Y + g
		x = bb.LL.X
		if horAlign == types.AlignRight {
			x += bb.Width()
		} else if horAlign == types.AlignCenter {
			x += bb.Width() / 2
		}

	case types.RelPosBottom:
		y = bb.LL.Y - g - labelHeight
		x = bb.LL.X
		if horAlign == types.AlignRight {
			x += bb.Width()
		} else if horAlign == types.AlignCenter {
			x += bb.Width() / 2
		}
	}

	return x, y
}

func (cb *CheckBox) ensureZapfDingbats(fonts model.FontMap) (*types.IndirectRef, error) {
	// TODO Refactor
	pdf := cb.pdf
	fontName := "ZapfDingbats"
	font, ok := fonts[fontName]
	if ok {
		if font.Res.IndRef != nil {
			return font.Res.IndRef, nil
		}
		ir, err := pdffont.EnsureFontDict(pdf.XRefTable, fontName, "", "", false, false, nil)
		if err != nil {
			return nil, err
		}
		font.Res.IndRef = ir
		fonts[fontName] = font
		return ir, nil
	}

	var (
		indRef *types.IndirectRef
		err    error
	)

	if pdf.Update() {

		for objNr, fo := range pdf.Optimize.FormFontObjects {
			//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
			if fontName == fo.FontName {
				//fmt.Println("Match!")
				indRef = types.NewIndirectRef(objNr, 0)
				break
			}
		}

		if indRef == nil {
			for objNr, fo := range pdf.Optimize.FontObjects {
				//fmt.Printf("searching for %s - obj:%d fontName:%s prefix:%s\n", fontName, objNr, fo.FontName, fo.Prefix)
				if fontName == fo.FontName {
					//fmt.Println("Match!")
					indRef = types.NewIndirectRef(objNr, 0)
					break
				}
			}
		}
	}

	if indRef == nil {
		indRef, err = pdffont.EnsureFontDict(pdf.XRefTable, fontName, "", "", false, false, nil)
		if err != nil {
			return nil, err
		}
	}

	font.Res = model.Resource{IndRef: indRef}

	fonts[fontName] = font

	return indRef, nil
}

func (cb *CheckBox) calcFont() error {

	if cb.Label != nil {
		f, err := cb.content.calcLabelFont(cb.Label.Font)
		if err != nil {
			return err
		}
		cb.Label.Font = f
	}

	return nil
}

func (cb *CheckBox) irNOff(bgCol *color.SimpleColor) (*types.IndirectRef, error) {

	pdf := cb.pdf

	ap, found := pdf.CheckBoxAPs[cb.Width]
	if found && ap.irNOffL != nil {
		return ap.irNOffL, nil
	}

	buf := new(bytes.Buffer)

	fmt.Fprint(buf, "q ")
	if bgCol != nil {
		fmt.Fprintf(buf, "%.2f %.2f %.2f rg ", bgCol.R, bgCol.G, bgCol.B)
	} else {
		fmt.Fprint(buf, "1 g ")
	}

	fmt.Fprintf(buf, "0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re s Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)

	sd, err := pdf.XRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irNOffL = ir

	return ir, nil
}

func (cb *CheckBox) irNYes(fonts model.FontMap, bgCol *color.SimpleColor) (*types.IndirectRef, error) {

	pdf := cb.pdf

	ap, found := pdf.CheckBoxAPs[cb.Width]
	if found && ap.irNYesL != nil {
		return ap.irNYesL, nil
	}

	buf := new(bytes.Buffer)

	fmt.Fprint(buf, "q ")
	if bgCol != nil {
		fmt.Fprintf(buf, "%.2f %.2f %.2f rg ", bgCol.R, bgCol.G, bgCol.B)
	} else {
		fmt.Fprint(buf, "1 g ")
	}

	s, x, y := 14.532/18, 2.853/18, 4.081/18
	fmt.Fprintf(buf, "0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re s Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	fmt.Fprintf(buf, "q 1 1 %.1f %.1f re W n BT /F0 %f Tf %f %f Td (4) Tj ET Q ", cb.Width-2, cb.Width-2, s*cb.Width, x*cb.Width, y*cb.Width)
	sd, err := pdf.XRefTable.NewStreamDictForBuf(buf.Bytes())
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	ir, err := cb.ensureZapfDingbats(fonts)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					"F0": *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err = pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irNYesL = ir

	return ir, nil
}

func (cb *CheckBox) irDOff(bgCol *color.SimpleColor) (*types.IndirectRef, error) {

	pdf := cb.pdf

	ap, found := cb.pdf.CheckBoxAPs[cb.Width]
	if found && ap.irDOffL != nil {
		return ap.irDOffL, nil
	}

	buf := fmt.Sprintf("q 0.75293 g 0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re se Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	sd, err := pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err := pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irDOffL = ir

	return ir, nil
}

func (cb *CheckBox) irDYes(fonts model.FontMap, bgCol *color.SimpleColor) (*types.IndirectRef, error) {

	pdf := cb.pdf

	ap, found := pdf.CheckBoxAPs[cb.Width]
	if found && ap.irDYesL != nil {
		return ap.irDYesL, nil
	}

	s, x, y := 14.532/18, 2.853/18, 4.081/18
	buf := fmt.Sprintf("q 0.75293 g 0 0 %.1f %.1f re f 0.5 0.5 %.1f %.1f re se Q ", cb.Width, cb.Width, cb.Width-1, cb.Width-1)
	buf += fmt.Sprintf("q 1 1 %.1f %.1f re W n BT /F0 %f Tf %f %f Td (4) Tj ET Q ", cb.Width-2, cb.Width-2, s*cb.Width, x*cb.Width, y*cb.Width)
	sd, _ := cb.pdf.XRefTable.NewStreamDictForBuf([]byte(buf))
	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, cb.Width, cb.Width))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	ir, err := cb.ensureZapfDingbats(fonts)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					"F0": *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	ir, err = pdf.XRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	if !found {
		ap = &AP{}
		pdf.CheckBoxAPs[cb.Width] = ap
	}
	ap.irDYesL = ir

	return ir, nil
}

func (cb *CheckBox) appearanceIndRefs(fonts model.FontMap, bgCol *color.SimpleColor) (
	*types.IndirectRef, *types.IndirectRef, *types.IndirectRef, *types.IndirectRef, error) {

	irDOff, err := cb.irDOff(bgCol)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irDYes, err := cb.irDYes(fonts, bgCol)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNOff, err := cb.irNOff(bgCol)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	irNYes, err := cb.irNYes(fonts, bgCol)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return irDOff, irDYes, irNOff, irNYes, nil
}

func (cb *CheckBox) prepareDict(fonts model.FontMap) (types.Dict, error) {

	id, err := types.EscapeUTF16String(cb.ID)
	if err != nil {
		return nil, err
	}

	v := "Off"
	if cb.Value {
		v = "Yes"
	}

	dv := "Off"
	if cb.Default {
		dv = "Yes"
		if !cb.Value {
			v = "Yes"
		}
	}

	bgCol := cb.bgCol
	if bgCol == nil {
		bgCol = cb.content.page.bgCol
		if bgCol == nil {
			bgCol = cb.pdf.bgCol
		}
	}

	irDOff, irDYes, irNOff, irNYes, err := cb.appearanceIndRefs(fonts, bgCol)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"FT":      types.Name("Btn"),
			"Rect":    cb.boundingBox.Array(),
			"F":       types.Integer(model.AnnPrint),
			"T":       types.StringLiteral(*id),
			"V":       types.Name(v), // -> extractValue: Off or Yes
			"DV":      types.Name(dv),
			"AS":      types.Name(v),
			"AP": types.Dict(
				map[string]types.Object{
					"D": types.Dict(
						map[string]types.Object{
							"Off": *irDOff,
							"Yes": *irDYes,
						},
					),
					"N": types.Dict(
						map[string]types.Object{
							"Off": *irNOff,
							"Yes": *irNYes,
						},
					),
				},
			),
		},
	)

	if cb.Tip != "" {
		tu, err := types.EscapeUTF16String(cb.Tip)
		if err != nil {
			return nil, err
		}
		d["TU"] = types.StringLiteral(*tu)
	}

	if bgCol != nil {
		appCharDict := types.Dict{}
		if bgCol != nil {
			appCharDict["BG"] = bgCol.Array()
		}
		d["MK"] = appCharDict
	}

	if cb.Locked {
		d["Ff"] = types.Integer(FieldReadOnly)
	}

	return d, nil
}

func (cb *CheckBox) bbox() *types.Rectangle {
	if cb.Label == nil {
		return cb.boundingBox.Clone()
	}

	l := cb.Label
	var r *types.Rectangle
	x := l.td.X

	switch l.td.HAlign {
	case types.AlignCenter:
		x -= float64(l.Width) / 2
	case types.AlignRight:
		x -= float64(l.Width)
	}

	y := l.td.Y
	if l.relPos == types.RelPosLeft || l.relPos == types.RelPosRight {
		y -= cb.boundingBox.Height() / 2
	}
	r = types.RectForWidthAndHeight(x, y, float64(l.Width), l.height)

	return model.CalcBoundingBoxForRects(cb.boundingBox, r)
}

func (cb *CheckBox) prepareRectLL(mTop, mRight, mBottom, mLeft float64) (float64, float64) {
	return cb.content.calcPosition(cb.x, cb.y, cb.Dx, cb.Dy, mTop, mRight, mBottom, mLeft)
}

func (cb *CheckBox) prepLabel(p *model.Page, pageNr int, fonts model.FontMap) error {

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

	f := l.Font
	fontName, fontLang, col := f.Name, f.Lang, f.col

	id, err := cb.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := model.TextDescriptor{
		Text:     v,
		FontName: fontName,
		FontKey:  id,
		FontSize: f.Size,
		Scale:    1.,
		ScaleAbs: true,
		RTL:      l.RTL,
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.BgCol != nil {
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.BgCol
	}

	bb := model.WriteMultiLine(cb.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	l.height = bb.Height()
	if bb.Width() > w {
		w = bb.Width()
		l.Width = int(bb.Width())
	}

	td.X, td.Y = cb.labelPos(l.height, w, g)
	td.HAlign, td.VAlign = l.HorAlign, types.AlignBottom

	if l.relPos == types.RelPosLeft || l.relPos == types.RelPosRight {
		td.Y += cb.boundingBox.Height() / 2
		td.VAlign = types.AlignMiddle
	}

	l.td = &td

	return nil
}

func (cb *CheckBox) prepForRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	mTop, mRight, mBottom, mLeft, err := cb.calcMargin()
	if err != nil {
		return err
	}

	x, y := cb.prepareRectLL(mTop, mRight, mBottom, mLeft)

	if err := cb.calcFont(); err != nil {
		return err
	}

	cb.boundingBox = types.RectForWidthAndHeight(x, y, cb.Width, cb.Width)

	return cb.prepLabel(p, pageNr, fonts)
}

func (cb *CheckBox) doRender(p *model.Page, fonts model.FontMap) error {

	d, err := cb.prepareDict(fonts)
	if err != nil {
		return err
	}

	ann := model.FieldAnnotation{Dict: d}
	if cb.Tab > 0 {
		p.AnnotTabs[cb.Tab] = ann
	} else {
		p.Annots = append(p.Annots, ann)
	}

	if cb.Label != nil {
		model.WriteColumn(cb.pdf.XRefTable, p.Buf, p.MediaBox, nil, *cb.Label.td, 0)
	}

	if cb.Debug || cb.pdf.Debug {
		cb.pdf.highlightPos(p.Buf, cb.boundingBox.LL.X, cb.boundingBox.LL.Y, cb.content.Box())
	}

	return nil
}

func (cb *CheckBox) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := cb.prepForRender(p, pageNr, fonts); err != nil {
		return err
	}

	return cb.doRender(p, fonts)
}

func CalcCheckBoxASNames(d types.Dict) (types.Name, types.Name) {
	apDict := d.DictEntry("AP")
	d1 := apDict.DictEntry("D")
	if d1 == nil {
		d1 = apDict.DictEntry("N")
	}
	offName, yesName := "Off", "Yes"
	for k := range d1 {
		if k != "Off" {
			yesName = k
		}
	}
	return types.Name(offName), types.Name(yesName)
}

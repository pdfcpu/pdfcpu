/*
	Copyright 2022 The pdfcpu Authors.

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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ComboBox represents a specific choice form field including a positioned label.
type ComboBox struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Tip             string
	Default         string
	Value           string
	Options         []string
	Position        [2]float64 `json:"pos"`
	x, y            float64
	Width           float64
	Dx, Dy          float64
	BoundingBox     *types.Rectangle `json:"-"`
	Edit            bool
	Font            *FormFont
	FontID          string `json:"-"`
	Margin          *Margin
	Border          *Border
	BackgroundColor string             `json:"bgCol"`
	BgCol           *color.SimpleColor `json:"-"`
	Alignment       string             `json:"align"` // "Left", "Center", "Right"
	HorAlign        types.HAlignment   `json:"-"`
	RTL             bool
	Tab             int
	Locked          bool
	Debug           bool
	Hide            bool
}

func (cb *ComboBox) SetFontID(s string) {
	cb.FontID = s
}

func (cb *ComboBox) validateID() error {
	if cb.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if cb.pdf.DuplicateField(cb.ID) {
		return errors.Errorf("pdfcpu: duplicate form field: %s", cb.ID)
	}
	cb.pdf.FieldIDs[cb.ID] = true
	return nil
}

func (cb *ComboBox) validatePosition() error {
	if cb.Position[0] < 0 || cb.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", cb.ID)
	}
	cb.x, cb.y = cb.Position[0], cb.Position[1]
	return nil
}

func (cb *ComboBox) validateWidth() error {
	if cb.Width == 0 {
		return errors.Errorf("pdfcpu: field: %s width == 0", cb.ID)
	}
	return nil
}

func (cb *ComboBox) validateOptionsValueAndDefault() error {
	if len(cb.Options) == 0 {
		return errors.Errorf("pdfcpu: field: %s missing options", cb.ID)
	}

	if len(cb.Value) > 0 && !types.MemberOf(cb.Value, cb.Options) {
		return errors.Errorf("pdfcpu: field: %s invalid value: %s", cb.ID, cb.Value)
	}

	if len(cb.Default) > 0 && !types.MemberOf(cb.Default, cb.Options) {
		return errors.Errorf("pdfcpu: field: %s invalid default: %s", cb.ID, cb.Default)
	}

	return nil
}

func (cb *ComboBox) validateFont() error {
	if cb.Font != nil {
		cb.Font.pdf = cb.pdf
		if err := cb.Font.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cb *ComboBox) validateMargin() error {
	if cb.Margin != nil {
		if err := cb.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cb *ComboBox) validateBorder() error {
	if cb.Border != nil {
		cb.Border.pdf = cb.pdf
		if err := cb.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cb *ComboBox) validateBackgroundColor() error {
	if cb.BackgroundColor != "" {
		sc, err := cb.pdf.parseColor(cb.BackgroundColor)
		if err != nil {
			return err
		}
		cb.BgCol = sc
	}
	return nil
}

func (cb *ComboBox) validateHorAlign() error {
	cb.HorAlign = types.AlignLeft
	if cb.Alignment != "" {
		ha, err := types.ParseHorAlignment(cb.Alignment)
		if err != nil {
			return err
		}
		cb.HorAlign = ha
	}
	return nil
}

func (cb *ComboBox) validateLabel() error {
	if cb.Label != nil {
		cb.Label.pdf = cb.pdf
		if err := cb.Label.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cb *ComboBox) validateTab() error {
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

func (cb *ComboBox) validate() error {

	if err := cb.validateID(); err != nil {
		return err
	}

	if err := cb.validatePosition(); err != nil {
		return err
	}

	if err := cb.validateWidth(); err != nil {
		return err
	}

	if err := cb.validateOptionsValueAndDefault(); err != nil {
		return err
	}

	if err := cb.validateFont(); err != nil {
		return err
	}

	if err := cb.validateMargin(); err != nil {
		return err
	}

	if err := cb.validateBorder(); err != nil {
		return err
	}

	if err := cb.validateBackgroundColor(); err != nil {
		return err
	}

	if err := cb.validateHorAlign(); err != nil {
		return err
	}

	if err := cb.validateLabel(); err != nil {
		return err
	}

	return cb.validateTab()
}

// NewComboBox creates a new combobox for d.
func NewComboBox(xRefTable *model.XRefTable, d types.Dict, opts []string) (*ComboBox, error) {
	cb := &ComboBox{}

	bb, _ := types.RectForArray(d.ArrayEntry("Rect"))
	cb.BoundingBox = types.RectForDim(bb.Width(), bb.Height())

	var f FormFont

	s := d.StringEntry("DA")
	if s == nil {
		return nil, errors.New("pdfcpu: combobox missing \"DA\"")
	}
	da := strings.Split(*s, " ")

	var fontID string

	for i := 0; i < len(da); i++ {
		if da[i] == "Tf" {
			fontID = da[i-2][1:]
			cb.SetFontID(fontID)
			f.Size, _ = strconv.Atoi(da[i-1])
			continue
		}
		if da[i] == "rg" {
			r, _ := strconv.ParseFloat(da[i-3], 32)
			g, _ := strconv.ParseFloat(da[i-2], 32)
			b, _ := strconv.ParseFloat(da[i-1], 32)
			f.SetCol(color.SimpleColor{R: float32(r), G: float32(g), B: float32(b)})
		}
	}
	cb.Font = &f

	cb.Options = opts

	// cb.horAlign
	q := d.IntEntry("Q")
	cb.HorAlign = types.AlignLeft
	if q != nil {
		cb.HorAlign = types.HAlignment(*q)
	}

	// cb.bgCol, cb.boCol
	boCol := color.Black

	o, err := xRefTable.DereferenceDictEntry(d, "MK")
	if err != nil {
		return nil, err
	}
	if o != nil {
		d1, _ := o.(types.Dict)
		if len(d1) > 0 {

			if arr := d1.ArrayEntry("BG"); arr != nil {
				bgCol := (color.NewSimpleColorForArray(arr))
				cb.BgCol = &bgCol
			}

			if arr := d1.ArrayEntry("BC"); arr != nil {
				boCol = (color.NewSimpleColorForArray(arr))
			}
		}
	}

	// cb.Border
	boWidth := 0
	if arr := d.ArrayEntry("Border"); arr != nil {
		// 0, 1 ??
		bw, ok := arr[2].(types.Integer)
		if ok {
			boWidth = bw.Value()
		} else {
			boWidth = int(arr[2].(types.Float).Value())
		}
	}

	var b Border
	if boWidth > 0 {
		b.Width = boWidth
		b.SetCol(boCol)
	}
	cb.Border = &b

	return cb, nil
}

func (cb *ComboBox) calcFont() error {
	f, err := cb.content.calcInputFont(cb.Font)
	if err != nil {
		return err
	}
	cb.Font = f

	if cb.Label != nil {
		f, err = cb.content.calcLabelFont(cb.Label.Font)
		if err != nil {
			return err
		}
		cb.Label.Font = f
	}

	return nil
}

func (cb *ComboBox) margin(name string) *Margin {
	return cb.content.namedMargin(name)
}

func (cb *ComboBox) calcMargin() (float64, float64, float64, float64, error) {
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

func (cb *ComboBox) labelPos(labelHeight, w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := cb.BoundingBox, cb.Label.HorAlign

	switch cb.Label.relPos {

	case types.RelPosLeft, types.RelPosBottom:
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

	}

	return x, y
}

func (cb *ComboBox) renderN(xRefTable *model.XRefTable) ([]byte, error) {
	w, h := cb.BoundingBox.Width(), cb.BoundingBox.Height()
	bgCol := cb.BgCol
	boWidth, boCol := cb.calcBorder()
	buf := new(bytes.Buffer)

	if bgCol != nil || boCol != nil {
		fmt.Fprint(buf, "q ")
		if bgCol != nil {
			fmt.Fprintf(buf, "%.2f %.2f %.2f rg 0 0 %.2f %.2f re f ", bgCol.R, bgCol.G, bgCol.B, w, h)
		}
		if boCol != nil {
			fmt.Fprintf(buf, "%.2f %.2f %.2f RG %.2f w %.2f %.2f %.2f %.2f re s ",
				boCol.R, boCol.G, boCol.B, boWidth, boWidth/2, boWidth/2, w-boWidth, h-boWidth)
		}
		fmt.Fprint(buf, "Q ")
	}

	fmt.Fprint(buf, "/Tx BMC q ")
	fmt.Fprintf(buf, "1 1 %.2f %.2f re W n ", w-2, h-2)

	f := cb.Font

	v := cb.Default
	if cb.Value != "" {
		v = cb.Value
	}

	//cjk := fo.CJK(f.Script, f.Lang)
	if font.IsCoreFont(f.Name) && utf8.ValidString(v) {
		v = model.DecodeUTF8ToByte(v)
	}
	lineBB := model.CalcBoundingBox(v, 0, 0, f.Name, f.Size)
	s := model.PrepBytes(xRefTable, v, f.Name, false, cb.RTL)
	x := 2 * boWidth
	if x == 0 {
		x = 2
	}
	switch cb.HorAlign {
	case types.AlignCenter:
		x = w/2 - lineBB.Width()/2
	case types.AlignRight:
		x = w - lineBB.Width() - 2
	}

	y := (cb.BoundingBox.Height()-font.LineHeight(f.Name, f.Size))/2 + font.Descent(f.Name, f.Size)

	fmt.Fprintf(buf, "BT /%s %d Tf ", cb.FontID, f.Size)
	fmt.Fprintf(buf, "%.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td (%s) Tj ET ",
		f.col.R, f.col.G, f.col.B,
		f.col.R, f.col.G, f.col.B, x, y, s)

	fmt.Fprint(buf, "Q EMC ")

	if boCol != nil && boWidth > 0 {
		fmt.Fprintf(buf, "q %.2f %.2f %.2f RG %.2f w %.2f %.2f %.2f %.2f re s Q ",
			boCol.R, boCol.G, boCol.B, boWidth-1, boWidth/2, boWidth/2, w-boWidth, h-boWidth)
	}

	return buf.Bytes(), nil
}

// ComboBoxN creates a new appearance dict needed to render locked combo boxes.
// This is needed because Adobe Reader provides the appearance dict for unlocked fields only.
func ComboBoxN(xRefTable *model.XRefTable, cb *ComboBox, ir types.IndirectRef) (*types.IndirectRef, error) {

	bb, err := cb.renderN(xRefTable)
	if err != nil {
		return nil, err
	}

	sd, err := xRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, cb.BoundingBox.Width(), cb.BoundingBox.Height()))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					cb.FontID: ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func (cb *ComboBox) calcBorder() (boWidth float64, boCol *color.SimpleColor) {
	if cb.Border == nil {
		return 0, nil
	}
	return cb.Border.calc()
}

func (cb *ComboBox) prepareDict(fonts model.FontMap) (types.Dict, error) {
	pdf := cb.pdf

	id, err := types.EscapeUTF16String(cb.ID)
	if err != nil {
		return nil, err
	}

	opt := types.Array{}
	for _, s := range cb.Options {
		s, err := types.EscapeUTF16String(s)
		if err != nil {
			return nil, err
		}
		opt = append(opt, types.StringLiteral(*s))
	}

	ff := FieldFlags(0)
	ff += FieldCombo
	if cb.Edit {
		// Note: unsupported in Mac Preview
		ff += FieldEdit + FieldDoNotSpellCheck
	}
	if cb.Locked {
		// Note: unsupported in Mac Preview
		ff += FieldReadOnly
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"FT":      types.Name("Ch"),
			"Rect":    cb.BoundingBox.Array(),
			"F":       types.Integer(model.AnnPrint),
			"Ff":      types.Integer(ff),
			"Opt":     opt,
			"Q":       types.Integer(cb.HorAlign),
			"T":       types.StringLiteral(*id),
		},
	)

	if cb.Tip != "" {
		tu, err := types.EscapeUTF16String(cb.Tip)
		if err != nil {
			return nil, err
		}
		d["TU"] = types.StringLiteral(*tu)
	}

	bgCol := cb.BgCol
	if bgCol == nil {
		bgCol = cb.content.page.bgCol
		if bgCol == nil {
			bgCol = cb.pdf.bgCol
		}
	}
	cb.BgCol = bgCol

	boWidth, boCol := cb.calcBorder()

	if bgCol != nil || boCol != nil {
		appCharDict := types.Dict{}
		if bgCol != nil {
			appCharDict["BG"] = bgCol.Array()
		}
		if boCol != nil && cb.Border.Width > 0 {
			appCharDict["BC"] = boCol.Array()
		}
		d["MK"] = appCharDict
	}

	if boWidth > 0 {
		d["Border"] = types.NewNumberArray(0, 0, boWidth)
	}

	v := cb.Value
	if cb.Default != "" {
		s, err := types.EscapeUTF16String(cb.Default)
		if err != nil {
			return nil, err
		}
		d["DV"] = types.StringLiteral(*s)
		if v == "" {
			v = cb.Default
		}
	}

	ind := types.Array{}
	for i, o := range cb.Options {
		if o == v {
			ind = append(ind, types.Integer(i))
			break
		}
	}
	s, err := types.EscapeUTF16String(v)
	if err != nil {
		return nil, err
	}
	d["V"] = types.StringLiteral(*s)
	d["I"] = ind

	if pdf.InheritedDA != "" {
		d["DA"] = types.StringLiteral(pdf.InheritedDA)
	}

	f := cb.Font
	fCol := f.col

	fontID, err := pdf.ensureFormFont(f)
	if err != nil {
		return d, err
	}
	cb.FontID = fontID

	da := fmt.Sprintf("/%s %d Tf %.2f %.2f %.2f rg", fontID, f.Size, fCol.R, fCol.G, fCol.B)
	// Note: Mac Preview does not honour inherited "DA"
	d["DA"] = types.StringLiteral(da)

	return d, nil
}

func (cb *ComboBox) bbox() *types.Rectangle {
	if cb.Label == nil {
		return cb.BoundingBox.Clone()
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

	r = types.RectForWidthAndHeight(x, l.td.Y, float64(l.Width), l.height)

	return model.CalcBoundingBoxForRects(cb.BoundingBox, r)
}

func (cb *ComboBox) prepareRectLL(mTop, mRight, mBottom, mLeft float64) (float64, float64) {
	return cb.content.calcPosition(cb.x, cb.y, cb.Dx, cb.Dy, mTop, mRight, mBottom, mLeft)
}

func (cb *ComboBox) prepLabel(p *model.Page, pageNr int, fonts model.FontMap) error {

	if cb.Label == nil {
		return nil
	}

	l := cb.Label
	v := l.Value
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
	l.height = bb.Height() + 10

	// Weird heuristic for vertical alignment with label
	if f.Size >= 24 {
		td.MTop, td.MBot = 6, 4
	} else if f.Size >= 12 {
		td.MTop, td.MBot = 5, 5
	} else {
		td.MTop, td.MBot = 6, 4
	}

	if bb.Width() > w {
		w = bb.Width()
		l.Width = int(bb.Width())
	}

	td.X, td.Y = cb.labelPos(l.height, w, g)
	td.HAlign, td.VAlign = l.HorAlign, types.AlignBottom

	l.td = &td

	return nil
}

func (cb *ComboBox) prepForRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	mTop, mRight, mBottom, mLeft, err := cb.calcMargin()
	if err != nil {
		return err
	}

	x, y := cb.prepareRectLL(mTop, mRight, mBottom, mLeft)

	if err := cb.calcFont(); err != nil {
		return err
	}

	td := model.TextDescriptor{
		Text:     "Xy",
		FontName: cb.Font.Name,
		FontSize: cb.Font.Size,
		Scale:    1.,
		ScaleAbs: true,
	}

	bb := model.WriteMultiLine(cb.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)

	if cb.Width < 0 {
		// Extend width to maxWidth.
		r := cb.content.Box().CroppedCopy(0)
		r.LL.X += mLeft
		r.LL.Y += mBottom
		r.UR.X -= mRight
		r.UR.Y -= mTop
		cb.Width = r.Width() - cb.x

	}

	cb.BoundingBox = types.RectForWidthAndHeight(x, y, cb.Width, bb.Height()+10)

	return cb.prepLabel(p, pageNr, fonts)
}

func (cb *ComboBox) doRender(p *model.Page, fonts model.FontMap) error {

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
		cb.pdf.highlightPos(p.Buf, cb.BoundingBox.LL.X, cb.BoundingBox.LL.Y, cb.content.Box())
	}

	return nil
}

func (cb *ComboBox) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := cb.prepForRender(p, pageNr, fonts); err != nil {
		return err
	}

	return cb.doRender(p, fonts)
}

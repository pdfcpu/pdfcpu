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
	"io"

	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/format"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type TextField struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Tip             string
	Value           string
	Default         string
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Height          float64
	Dx, Dy          float64
	BoundingBox     *types.Rectangle `json:"-"`
	Multiline       bool
	Font            *FormFont
	fontID          string
	Margin          *Margin // applied to content box
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

func (tf *TextField) SetFontID(s string) {
	tf.fontID = s
}

func (tf *TextField) validateID() error {
	if tf.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if tf.pdf.DuplicateField(tf.ID) {
		return errors.Errorf("pdfcpu: duplicate form field: %s", tf.ID)
	}
	tf.pdf.FieldIDs[tf.ID] = true
	return nil
}

func (tf *TextField) validatePosition() error {
	if tf.Position[0] < 0 || tf.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", tf.ID)
	}
	tf.x, tf.y = tf.Position[0], tf.Position[1]
	return nil
}

func (tf *TextField) validateWidth() error {
	if tf.Width == 0 {
		return errors.Errorf("pdfcpu: field: %s width == 0", tf.ID)
	}
	return nil
}

func (tf *TextField) validateHeight() error {
	if tf.Height < 0 {
		return errors.Errorf("pdfcpu: field: %s height < 0", tf.ID)
	}
	return nil
}

func (tf *TextField) validateFont() error {
	if tf.Font != nil {
		tf.Font.pdf = tf.pdf
		if err := tf.Font.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (tf *TextField) validateMargin() error {
	if tf.Margin != nil {
		if err := tf.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (tf *TextField) validateBorder() error {
	if tf.Border != nil {
		tf.Border.pdf = tf.pdf
		if err := tf.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (tf *TextField) validateBackgroundColor() error {
	if tf.BackgroundColor != "" {
		sc, err := tf.pdf.parseColor(tf.BackgroundColor)
		if err != nil {
			return err
		}
		tf.BgCol = sc
	}
	return nil
}

func (tf *TextField) validateHorAlign() error {
	tf.HorAlign = types.AlignLeft
	if tf.Alignment != "" {
		ha, err := types.ParseHorAlignment(tf.Alignment)
		if err != nil {
			return err
		}
		tf.HorAlign = ha
	}
	return nil
}

func (tf *TextField) validateLabel() error {
	if tf.Label != nil {
		tf.Label.pdf = tf.pdf
		if err := tf.Label.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (tf *TextField) validateTab() error {
	if tf.Tab < 0 {
		return errors.Errorf("pdfcpu: field: %s negative tab value", tf.ID)
	}
	if tf.Tab == 0 {
		return nil
	}
	page := tf.content.page
	if page.Tabs == nil {
		page.Tabs = types.IntSet{}
	} else {
		if page.Tabs[tf.Tab] {
			return errors.Errorf("pdfcpu: field: %s duplicate tab value %d", tf.ID, tf.Tab)
		}
	}
	page.Tabs[tf.Tab] = true
	return nil
}

func (tf *TextField) validate() error {

	if err := tf.validateID(); err != nil {
		return err
	}

	if err := tf.validatePosition(); err != nil {
		return err
	}

	if err := tf.validateWidth(); err != nil {
		return err
	}

	if err := tf.validateHeight(); err != nil {
		return err
	}

	if err := tf.validateFont(); err != nil {
		return err
	}

	if err := tf.validateMargin(); err != nil {
		return err
	}

	if err := tf.validateBorder(); err != nil {
		return err
	}

	if err := tf.validateBackgroundColor(); err != nil {
		return err
	}

	if err := tf.validateHorAlign(); err != nil {
		return err
	}

	if err := tf.validateLabel(); err != nil {
		return err
	}

	return tf.validateTab()
}

func (tf *TextField) calcFontFromDA(ctx *model.Context, d types.Dict, fonts map[string]types.IndirectRef) (*types.IndirectRef, error) {

	s := d.StringEntry("DA")
	if s == nil {
		s = ctx.Form.StringEntry("DA")
		if s == nil {
			return nil, errors.New("pdfcpu: textfield missing \"DA\"")
		}
	}

	fontID, f, err := fontFromDA(*s)
	if err != nil {
		return nil, err
	}

	tf.Font, tf.fontID = &f, fontID

	id, name, lang, fontIndRef, err := extractFormFontDetails(ctx, tf.fontID, fonts)
	if err != nil {
		return nil, err
	}
	if fontIndRef == nil {
		return nil, errors.New("pdfcpu: unable to detect indirect reference for font")
	}

	tf.fontID = id
	tf.Font.Name = name
	tf.Font.Lang = lang
	tf.RTL = pdffont.RTL(lang)

	return fontIndRef, nil
}

func (tf *TextField) calcFont() error {
	f, err := tf.content.calcInputFont(tf.Font)
	if err != nil {
		return err
	}
	tf.Font = f

	if tf.Label != nil {
		f, err = tf.content.calcLabelFont(tf.Label.Font)
		if err != nil {
			return err
		}
		tf.Label.Font = f
	}

	return nil
}

func (tf *TextField) margin(name string) *Margin {
	return tf.content.namedMargin(name)
}

func (tf *TextField) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if tf.Margin != nil {
		m := tf.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := tf.margin(mName)
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

func (tf *TextField) labelPos(labelHeight, w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := tf.BoundingBox, tf.Label.HorAlign

	switch tf.Label.relPos {

	case types.RelPosLeft:
		x = bb.LL.X - g
		if horAlign == types.AlignLeft {
			x -= w
			if x < 0 {
				x = 0
			}
		}
		if tf.Multiline {
			y = bb.UR.Y - labelHeight
		} else {
			y = bb.LL.Y
		}

	case types.RelPosRight:
		x = bb.UR.X + g
		if horAlign == types.AlignRight {
			x += w
		}
		if tf.Multiline {
			y = bb.UR.Y - labelHeight
		} else {
			y = bb.LL.Y
		}

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

func (tf *TextField) renderBackground(w io.Writer, bgCol, boCol *color.SimpleColor, boWidth, width, height float64) {
	if bgCol != nil || (boCol != nil && boWidth > 0) {
		fmt.Fprint(w, "q ")
		if bgCol != nil {
			fmt.Fprintf(w, "%.2f %.2f %.2f rg 0 0 %.2f %.2f re f ", bgCol.R, bgCol.G, bgCol.B, width, height)
		}
		if boCol != nil && boWidth > 0 {
			fmt.Fprintf(w, "%.2f %.2f %.2f RG %.2f w %.2f %.2f %.2f %.2f re s ",
				boCol.R, boCol.G, boCol.B, boWidth, boWidth/2, boWidth/2, width-boWidth, height-boWidth)
		}
		fmt.Fprint(w, "Q ")
	}
}

func (tf *TextField) renderN(xRefTable *model.XRefTable) ([]byte, error) {

	w, h := tf.BoundingBox.Width(), tf.BoundingBox.Height()
	bgCol := tf.BgCol
	boWidth, boCol := tf.calcBorder()
	buf := new(bytes.Buffer)

	tf.renderBackground(buf, bgCol, boCol, boWidth, w, h)

	f := tf.Font

	s := tf.Value
	if s == "" {
		s = tf.Default
	}

	if font.IsCoreFont(f.Name) && utf8.ValidString(s) {
		s = model.DecodeUTF8ToByte(s)
	}
	lines := model.SplitMultilineStr(s)

	fmt.Fprint(buf, "/Tx BMC ")

	lh := font.LineHeight(f.Name, f.Size)
	y := (tf.BoundingBox.Height()-font.LineHeight(f.Name, f.Size))/2 + font.Descent(f.Name, f.Size)
	if tf.Multiline {
		y = tf.BoundingBox.Height() - font.LineHeight(f.Name, f.Size)
	}

	if len(lines) > 0 {
		fmt.Fprintf(buf, "q 1 1 %.1f %.1f re W n ", w-2, h-2)
	}

	cjk := pdffont.CJK(f.Script, f.Lang)

	for i := 0; i < len(lines); i++ {
		s := lines[i]
		lineBB := model.CalcBoundingBox(s, 0, 0, f.Name, f.Size)
		s = model.PrepBytes(xRefTable, s, f.Name, cjk, f.RTL()) //tf.RTL)
		x := 2 * boWidth
		if x == 0 {
			x = 2
		}
		switch tf.HorAlign {
		case types.AlignCenter:
			x = w/2 - lineBB.Width()/2
		case types.AlignRight:
			x = w - lineBB.Width() - 2
		}
		fmt.Fprint(buf, "BT ")
		if i == 0 {
			fmt.Fprintf(buf, "/%s %d Tf %.2f %.2f %.2f RG %.2f %.2f %.2f rg ",
				tf.fontID, f.Size,
				f.col.R, f.col.G, f.col.B,
				f.col.R, f.col.G, f.col.B)
		}
		fmt.Fprintf(buf, "%.2f %.2f Td (%s) Tj ET ", x, y, s)
		y -= lh
	}

	if len(lines) > 0 {
		fmt.Fprint(buf, "Q ")
	}

	fmt.Fprint(buf, "EMC ")

	if boCol != nil && boWidth > 0 {
		fmt.Fprintf(buf, "q %.2f %.2f %.2f RG %.2f w %.2f %.2f %.2f %.2f re s Q ",
			boCol.R, boCol.G, boCol.B, boWidth-1, boWidth/2, boWidth/2, w-boWidth, h-boWidth)
	}

	return buf.Bytes(), nil
}

func (tf *TextField) RefreshN(xRefTable *model.XRefTable, indRef *types.IndirectRef) error {

	bb, err := tf.renderN(xRefTable)
	if err != nil {
		return err
	}

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)
	sd, _ := entry.Object.(types.StreamDict)

	sd.Content = bb
	if err := sd.Encode(); err != nil {
		return err
	}

	entry.Object = sd

	return nil
}

func (tf *TextField) irN(fonts model.FontMap) (*types.IndirectRef, error) {

	bb, err := tf.renderN(tf.pdf.XRefTable)
	if err != nil {
		return nil, err
	}

	sd, err := tf.pdf.XRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, tf.BoundingBox.Width(), tf.BoundingBox.Height()))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	f := tf.Font

	fName := f.Name
	if pdffont.CJK(tf.Font.Script, tf.Font.Lang) {
		fName = "cjk:" + fName
	}

	ir, err := tf.pdf.ensureFont(tf.fontID, fName, tf.Font.Lang, fonts)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					tf.fontID: *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return tf.pdf.XRefTable.IndRefForNewObject(*sd)
}

func (tf *TextField) calcBorder() (boWidth float64, boCol *color.SimpleColor) {
	if tf.Border == nil {
		return 0, nil
	}
	return tf.Border.calc()
}

func (tf *TextField) prepareFF() FieldFlags {
	ff := FieldDoNotSpellCheck
	if tf.Multiline {
		// If FieldMultiline set, the field may contain multiple lines of text;
		// if clear, the field’s text shall be restricted to a single line.
		// Adobe Reader ok, Mac Preview nope
		ff += FieldMultiline
	} else {
		// If FieldDoNotScroll set, the field shall not scroll (horizontally for single-line fields, vertically for multiple-line fields)
		// to accommodate more text than fits within its annotation rectangle.
		// Once the field is full, no further text shall be accepted for interactive form filling;
		// for non- interactive form filling, the filler should take care
		// not to add more character than will visibly fit in the defined area.
		// Adobe Reader ok, Mac Preview nope :(
		ff += FieldDoNotScroll
	}

	if tf.Locked {
		ff += FieldReadOnly
	}

	return ff
}

func (tf *TextField) handleBorderAndMK(d types.Dict) {
	bgCol := tf.BgCol
	if bgCol == nil {
		bgCol = tf.content.page.bgCol
		if bgCol == nil {
			bgCol = tf.pdf.bgCol
		}
	}
	tf.BgCol = bgCol

	boWidth, boCol := tf.calcBorder()

	if bgCol != nil || boCol != nil {
		appCharDict := types.Dict{}
		if bgCol != nil {
			appCharDict["BG"] = bgCol.Array()
		}
		if boCol != nil && tf.Border.Width > 0 {
			appCharDict["BC"] = boCol.Array()
		}
		d["MK"] = appCharDict
	}

	if boWidth > 0 {
		d["Border"] = types.NewNumberArray(0, 0, boWidth)
	}
}

func (tf *TextField) prepareDict(fonts model.FontMap) (types.Dict, error) {
	pdf := tf.pdf

	id, err := types.EscapeUTF16String(tf.ID)
	if err != nil {
		return nil, err
	}

	ff := tf.prepareFF()

	d := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"FT":      types.Name("Tx"),
			"Rect":    tf.BoundingBox.Array(),
			"F":       types.Integer(model.AnnPrint),
			"Ff":      types.Integer(ff),
			"Q":       types.Integer(tf.HorAlign),
			"T":       types.StringLiteral(*id),
		},
	)

	if tf.Tip != "" {
		tu, err := types.EscapeUTF16String(tf.Tip)
		if err != nil {
			return nil, err
		}
		d["TU"] = types.StringLiteral(*tu)
	}

	tf.handleBorderAndMK(d)

	if tf.Value != "" {
		s, err := types.EscapeUTF16String(tf.Value)
		if err != nil {
			return nil, err
		}
		d["V"] = types.StringLiteral(*s)
	}

	if tf.Default != "" {
		s, err := types.EscapeUTF16String(tf.Default)
		if err != nil {
			return nil, err
		}
		d["DV"] = types.StringLiteral(*s)
		if tf.Value == "" {
			d["V"] = types.StringLiteral(*s)
		}
	}

	if pdf.InheritedDA != "" {
		d["DA"] = types.StringLiteral(pdf.InheritedDA)
	}

	f := tf.Font
	fCol := f.col

	fontID, err := pdf.ensureFormFont(f)
	if err != nil {
		return d, err
	}
	tf.fontID = fontID

	da := fmt.Sprintf("/%s %d Tf %.2f %.2f %.2f rg", fontID, f.Size, fCol.R, fCol.G, fCol.B)
	// Note: Mac Preview does not honour inherited "DA"
	d["DA"] = types.StringLiteral(da)

	irN, err := tf.irN(fonts)
	if err != nil {
		return nil, err
	}

	d["AP"] = types.Dict(map[string]types.Object{"N": *irN})

	return d, nil
}

func (tf *TextField) bbox() *types.Rectangle {
	if tf.Label == nil {
		return tf.BoundingBox.Clone()
	}

	l := tf.Label
	var r *types.Rectangle
	x := l.td.X

	switch l.td.HAlign {
	case types.AlignCenter:
		x -= float64(l.Width) / 2
	case types.AlignRight:
		x -= float64(l.Width)
	}

	r = types.RectForWidthAndHeight(x, l.td.Y, float64(l.Width), l.height)

	return model.CalcBoundingBoxForRects(tf.BoundingBox, r)
}

func (tf *TextField) prepareRectLL(mTop, mRight, mBottom, mLeft float64) (float64, float64) {
	return tf.content.calcPosition(tf.x, tf.y, tf.Dx, tf.Dy, mTop, mRight, mBottom, mLeft)
}

func (tf *TextField) prepLabel(p *model.Page, pageNr int, fonts model.FontMap) error {

	if tf.Label == nil {
		return nil
	}

	l := tf.Label
	pdf := tf.pdf

	t := "Default"
	if l.Value != "" {
		t, _ = format.Text(l.Value, pdf.TimestampFormat, pageNr, pdf.pageCount())
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	f := l.Font
	fontName, fontLang, col := f.Name, f.Lang, f.col

	id, err := tf.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td := model.TextDescriptor{
		Text:     t,
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

	bb := model.WriteMultiLine(tf.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	l.height = bb.Height()
	if bb.Width() > w {
		w = bb.Width()
		l.Width = int(bb.Width())
	}

	td.X, td.Y = tf.labelPos(l.height, w, g)

	if !tf.Multiline &&
		(bb.Height() < tf.BoundingBox.Height()) &&
		(l.relPos == types.RelPosLeft || l.relPos == types.RelPosRight) {
		td.MBot = (tf.BoundingBox.Height() - bb.Height()) / 2
		td.MTop = td.MBot
	}

	td.HAlign, td.VAlign = l.HorAlign, types.AlignBottom

	l.td = &td

	return nil
}

func (tf *TextField) prepForRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	mTop, mRight, mBottom, mLeft, err := tf.calcMargin()
	if err != nil {
		return err
	}

	x, y := tf.prepareRectLL(mTop, mRight, mBottom, mLeft)

	if err := tf.calcFont(); err != nil {
		return err
	}

	var boWidth int
	if tf.Border != nil {
		if tf.Border.col != nil {
			boWidth = tf.Border.Width
		}
	}

	h := float64(tf.Font.Size)*1.2 + 2*float64(boWidth)

	if tf.Multiline {
		if tf.Height == 0 {
			return errors.Errorf("pdfcpu: field: %s height == 0", tf.ID)
		}
		h = tf.Height
	}

	if tf.Width < 0 {
		// Extend width to maxWidth.
		if tf.HorAlign == types.AlignLeft || tf.HorAlign == types.AlignCenter {
			r := tf.content.Box().CroppedCopy(0)
			r.LL.X += mLeft
			r.LL.Y += mBottom
			r.UR.X -= mRight
			r.UR.Y -= mTop
			tf.Width = r.Width() - tf.x
		}
	}

	tf.BoundingBox = types.RectForWidthAndHeight(x, y, tf.Width, h)

	return tf.prepLabel(p, pageNr, fonts)
}

func (tf *TextField) doRender(p *model.Page, fonts model.FontMap) error {

	d, err := tf.prepareDict(fonts)
	if err != nil {
		return err
	}

	ann := model.FieldAnnotation{Dict: d}
	if tf.Tab > 0 {
		p.AnnotTabs[tf.Tab] = ann
	} else {
		p.Annots = append(p.Annots, ann)
	}

	if tf.Label != nil {
		model.WriteColumn(tf.pdf.XRefTable, p.Buf, p.MediaBox, nil, *tf.Label.td, 0)
	}

	if tf.Debug || tf.pdf.Debug {
		tf.pdf.highlightPos(p.Buf, tf.BoundingBox.LL.X, tf.BoundingBox.LL.Y, tf.content.Box())
	}

	return nil
}

func (tf *TextField) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := tf.prepForRender(p, pageNr, fonts); err != nil {
		return err
	}

	return tf.doRender(p, fonts)
}

func calcColsFromMK(ctx *model.Context, d types.Dict) (*color.SimpleColor, *color.SimpleColor, error) {

	var bgCol, boCol *color.SimpleColor

	if o, found := d.Find("MK"); found {
		d1, err := ctx.DereferenceDict(o)
		if err != nil {
			return nil, nil, err
		}
		if len(d1) > 0 {
			if arr := d1.ArrayEntry("BG"); len(arr) == 3 {
				sc := color.NewSimpleColorForArray(arr)
				bgCol = &sc
			}
			if arr := d1.ArrayEntry("BC"); len(arr) == 3 {
				sc := color.NewSimpleColorForArray(arr)
				boCol = &sc
			}
		}
	}

	return bgCol, boCol, nil
}

func calcBorderWidth(d types.Dict) int {
	w := 0
	if arr := d.ArrayEntry("Border"); len(arr) == 3 {
		// 0, 1 ??
		bw, ok := arr[2].(types.Integer)
		if ok {
			w = bw.Value()
		} else {
			w = int(arr[2].(types.Float).Value())
		}
	}
	return w
}

// NewTextField returns a new text field for d.
func NewTextField(
	ctx *model.Context,
	d types.Dict,
	v string,
	multiLine bool,
	fonts map[string]types.IndirectRef) (*TextField, *types.IndirectRef, error) {

	tf := &TextField{Value: v, Multiline: multiLine}

	bb, err := types.RectForArray(d.ArrayEntry("Rect"))
	if err != nil {
		return nil, nil, err
	}

	tf.BoundingBox = types.RectForDim(bb.Width(), bb.Height())

	fontIndRef, err := tf.calcFontFromDA(ctx, d, fonts)
	if err != nil {
		return nil, nil, err
	}

	tf.HorAlign = types.AlignLeft
	if q := d.IntEntry("Q"); q != nil {
		tf.HorAlign = types.HAlignment(*q)
	}

	bgCol, boCol, err := calcColsFromMK(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	tf.BgCol = bgCol

	var b Border
	boWidth := calcBorderWidth(d)
	if boWidth > 0 {
		b.Width = boWidth
		b.col = boCol
	}
	tf.Border = &b

	return tf, fontIndRef, nil
}

func renderTextFieldAP(ctx *model.Context, d types.Dict, v string, multiLine bool, fonts map[string]types.IndirectRef) error {

	tf, fontIndRef, err := NewTextField(ctx, d, v, multiLine, fonts)
	if err != nil {
		return err
	}

	bb, err := tf.renderN(ctx.XRefTable)
	if err != nil {
		return err
	}

	irN, err := NewForm(ctx.XRefTable, bb, tf.fontID, fontIndRef, tf.BoundingBox)
	if err != nil {
		return err
	}

	d["AP"] = types.Dict(map[string]types.Object{"N": *irN})

	return nil
}

func refreshTextFieldAP(ctx *model.Context, d types.Dict, v string, multiLine bool, fonts map[string]types.IndirectRef, irN *types.IndirectRef) error {

	tf, _, err := NewTextField(ctx, d, v, multiLine, fonts)
	if err != nil {
		return err
	}

	bb, err := tf.renderN(ctx.XRefTable)
	if err != nil {
		return err
	}

	return UpdateForm(ctx.XRefTable, bb, irN)
}

func EnsureTextFieldAP(ctx *model.Context, d types.Dict, v string, multiLine bool, fonts map[string]types.IndirectRef) error {

	apd := d.DictEntry("AP")
	if apd == nil {
		return renderTextFieldAP(ctx, d, v, multiLine, fonts)
	}

	irN := apd.IndirectRefEntry("N")
	if irN == nil {
		return nil
	}

	return refreshTextFieldAP(ctx, d, v, multiLine, fonts, irN)
}

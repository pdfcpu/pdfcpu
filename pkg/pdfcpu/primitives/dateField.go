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
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/format"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Note: Mac Preview does not support validating date fields.

// DateField is a form field accepting date strings according to DateFormat including a positioned label.
type DateField struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Tip             string
	Value           string
	Default         string
	DateFormat      string `json:"format"`
	dateFormat      *DateFormat
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Dx, Dy          float64
	BoundingBox     *types.Rectangle `json:"-"`
	Font            *FormFont
	fontID          string
	Margin          *Margin // applied to content box
	Border          *Border
	BackgroundColor string             `json:"bgCol"`
	BgCol           *color.SimpleColor `json:"-"`
	Alignment       string             `json:"align"` // "Left", "Center", "Right"
	HorAlign        types.HAlignment   `json:"-"`
	Tab             int
	Locked          bool
	Debug           bool
	Hide            bool
}

func (df *DateField) SetFontID(s string) {
	df.fontID = s
}

func (df *DateField) validateID() error {
	if df.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if df.pdf.DuplicateField(df.ID) {
		return errors.Errorf("pdfcpu: duplicate form field: %s", df.ID)
	}
	df.pdf.FieldIDs[df.ID] = true
	return nil
}

func (df *DateField) validatePosition() error {
	if df.Position[0] < 0 || df.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", df.ID)
	}
	df.x, df.y = df.Position[0], df.Position[1]
	return nil
}

func (df *DateField) validateWidth() error {
	if df.Width <= 0 {
		return errors.Errorf("pdfcpu: field: %s width <= 0", df.ID)
	}
	return nil
}

func (df *DateField) validateFont() error {
	if df.Font != nil {
		df.Font.pdf = df.pdf
		if err := df.Font.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (df *DateField) validateMargin() error {
	if df.Margin != nil {
		if err := df.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (df *DateField) validateBorder() error {
	if df.Border != nil {
		df.Border.pdf = df.pdf
		if err := df.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (df *DateField) validateBackgroundColor() error {
	if df.BackgroundColor != "" {
		sc, err := df.pdf.parseColor(df.BackgroundColor)
		if err != nil {
			return err
		}
		df.BgCol = sc
	}
	return nil
}

func (df *DateField) validateHorAlign() error {
	df.HorAlign = types.AlignLeft
	if df.Alignment != "" {
		ha, err := types.ParseHorAlignment(df.Alignment)
		if err != nil {
			return err
		}
		df.HorAlign = ha
	}
	return nil
}

func (df *DateField) validateLabel() error {
	if df.Label != nil {
		df.Label.pdf = df.pdf
		if err := df.Label.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (df *DateField) validateDateFormat() error {
	if len(df.DateFormat) == 0 {
		return nil
	}
	dFormat, err := DateFormatForFmtExt(df.DateFormat)
	if err != nil {
		return err
	}
	df.dateFormat = dFormat
	return nil
}

func (df *DateField) validateDefault() error {
	if df.Default == "" {
		return nil
	}
	if df.dateFormat != nil {
		if err := df.dateFormat.validate(df.Default); err != nil {
			return errors.Errorf("pdfcpu: field: %s date format failure, \"%s\" incompatible with  \"%s\"", df.ID, df.Default, df.dateFormat.Ext)
		}
		return nil
	}
	dFormat, err := DateFormatForDate(df.Default)
	if err != nil {
		return err
	}
	df.dateFormat = dFormat
	return nil
}

func (df *DateField) validateValue() error {
	if df.Value == "" {
		return nil
	}
	if df.dateFormat != nil {
		if err := df.dateFormat.validate(df.Value); err != nil {
			return errors.Errorf("pdfcpu: field: %s date format failure, \"%s\" incompatible with  \"%s\"", df.ID, df.Value, df.dateFormat.Ext)
		}
		return nil
	}
	dFormat, err := DateFormatForDate(df.Value)
	if err != nil {
		return err
	}
	df.dateFormat = dFormat
	return nil
}

func (df *DateField) validateTab() error {
	if df.Tab < 0 {
		return errors.Errorf("pdfcpu: field: %s negative tab value", df.ID)
	}
	if df.Tab == 0 {
		return nil
	}
	page := df.content.page
	if page.Tabs == nil {
		page.Tabs = types.IntSet{}
	} else {
		if page.Tabs[df.Tab] {
			return errors.Errorf("pdfcpu: field: %s duplicate tab value %d", df.ID, df.Tab)
		}
	}
	page.Tabs[df.Tab] = true
	return nil
}

func (df *DateField) validate() error {

	if err := df.validateID(); err != nil {
		return err
	}

	if err := df.validatePosition(); err != nil {
		return err
	}

	if err := df.validateWidth(); err != nil {
		return err
	}

	if err := df.validateFont(); err != nil {
		return err
	}

	if err := df.validateMargin(); err != nil {
		return err
	}

	if err := df.validateBorder(); err != nil {
		return err
	}

	if err := df.validateBackgroundColor(); err != nil {
		return err
	}

	if err := df.validateHorAlign(); err != nil {
		return err
	}

	if err := df.validateLabel(); err != nil {
		return err
	}

	if err := df.validateDateFormat(); err != nil {
		return err
	}

	if err := df.validateDefault(); err != nil {
		return err
	}

	if err := df.validateValue(); err != nil {
		return err
	}

	if df.dateFormat == nil {
		dFormat, err := DateFormatForFmtInt(df.pdf.DateFormat)
		if err != nil {
			return err
		}
		df.dateFormat = dFormat
	}

	return df.validateTab()
}

func (df *DateField) calcFontFromDA(ctx *model.Context, d types.Dict, fonts map[string]types.IndirectRef) (*types.IndirectRef, error) {

	s := d.StringEntry("DA")
	if s == nil {
		s = ctx.Form.StringEntry("DA")
		if s == nil {
			return nil, errors.New("pdfcpu: datefield missing \"DA\"")
		}
	}

	fontID, f, err := fontFromDA(*s)
	if err != nil {
		return nil, err
	}

	df.Font, df.fontID = &f, fontID

	id, name, lang, fontIndRef, err := extractFormFontDetails(ctx, df.fontID, fonts)
	if err != nil {
		return nil, err
	}
	if fontIndRef == nil {
		return nil, errors.New("pdfcpu: unable to detect indirect reference for font")
	}

	df.fontID = id
	df.Font.Name = name
	df.Font.Lang = lang
	//df.RTL = pdffont.RTL(lang)

	return fontIndRef, nil
}

func (df *DateField) calcFont() error {
	f, err := df.content.calcInputFont(df.Font)
	if err != nil {
		return err
	}
	df.Font = f

	if df.Label != nil {
		f, err = df.content.calcLabelFont(df.Label.Font)
		if err != nil {
			return err
		}
		df.Label.Font = f
	}

	return nil
}

func (df *DateField) margin(name string) *Margin {
	return df.content.namedMargin(name)
}

func (df *DateField) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if df.Margin != nil {
		m := df.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := df.margin(mName)
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

func (df *DateField) labelPos(labelHeight, w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := df.BoundingBox, df.Label.HorAlign

	switch df.Label.relPos {

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

func (tf *DateField) renderBackground(w io.Writer, bgCol, boCol *color.SimpleColor, boWidth, width, height float64) {
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

func (df *DateField) renderN(xRefTable *model.XRefTable) ([]byte, error) {

	w, h := df.BoundingBox.Width(), df.BoundingBox.Height()
	bgCol := df.BgCol
	boWidth, boCol := df.calcBorder()
	buf := new(bytes.Buffer)

	df.renderBackground(buf, bgCol, boCol, boWidth, w, h)

	fmt.Fprint(buf, "/Tx BMC q ")
	fmt.Fprintf(buf, "1 1 %.1f %.1f re W n ", w-2, h-2)

	v := ""
	if df.dateFormat != nil {
		v = df.dateFormat.Ext
	}
	if len(df.Default) > 0 {
		v = df.Default
	}
	if len(df.Value) > 0 {
		v = df.Value
	}

	f := df.Font
	//cjk := fo.CJK(f.Script, f.Lang)
	lineBB := model.CalcBoundingBox(v, 0, 0, f.Name, f.Size)
	s := model.PrepBytes(xRefTable, v, f.Name, false, false)
	x := 2 * boWidth
	if x == 0 {
		x = 2
	}
	switch df.HorAlign {
	case types.AlignCenter:
		x = w/2 - lineBB.Width()/2
	case types.AlignRight:
		x = w - lineBB.Width() - 2
	}

	y := (df.BoundingBox.Height()-font.LineHeight(f.Name, f.Size))/2 + font.Descent(f.Name, f.Size)

	fmt.Fprintf(buf, "BT /%s %d Tf ", df.fontID, f.Size)
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

// RefreshN updates the normal appearance referred to by indRef according to df.
func (df *DateField) RefreshN(xRefTable *model.XRefTable, indRef *types.IndirectRef) error {

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)

	bb, err := df.renderN(xRefTable)
	if err != nil {
		return err
	}

	sd, _ := entry.Object.(types.StreamDict)

	sd.Content = bb
	if err := sd.Encode(); err != nil {
		return err
	}

	entry.Object = sd

	return nil
}

func (df *DateField) irN(fonts model.FontMap) (*types.IndirectRef, error) {

	bb, err := df.renderN(df.pdf.XRefTable)
	if err != nil {
		return nil, err
	}

	sd, err := df.pdf.XRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, df.BoundingBox.Width(), df.BoundingBox.Height()))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	// f := df.Font

	// fName := f.Name
	// if fo.CJK(df.Font.Script, df.Font.Lang) {
	// 	fName = "cjk:" + fName
	// }

	ir, err := df.pdf.ensureFont(df.fontID, df.Font.Name, df.Font.Lang, fonts)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					df.fontID: *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return df.pdf.XRefTable.IndRefForNewObject(*sd)
}

func (df *DateField) calcBorder() (boWidth float64, boCol *color.SimpleColor) {
	if df.Border == nil {
		return 0, nil
	}
	return df.Border.calc()
}

func (df *DateField) prepareFF() FieldFlags {
	ff := FieldDoNotSpellCheck
	ff += FieldDoNotScroll
	if df.Locked {
		ff += FieldReadOnly
	}
	return ff
}

func (df *DateField) handleBorderAndMK(d types.Dict) {
	bgCol := df.BgCol
	if bgCol == nil {
		bgCol = df.content.page.bgCol
		if bgCol == nil {
			bgCol = df.pdf.bgCol
		}
	}
	df.BgCol = bgCol

	boWidth, boCol := df.calcBorder()

	if bgCol != nil || boCol != nil {
		appCharDict := types.Dict{}
		if bgCol != nil {
			appCharDict["BG"] = bgCol.Array()
		}
		if boCol != nil && df.Border.Width > 0 {
			appCharDict["BC"] = boCol.Array()
		}
		d["MK"] = appCharDict
	}

	if boWidth > 0 {
		d["Border"] = types.NewNumberArray(0, 0, boWidth)
	}
}

func (df *DateField) prepareDict(fonts model.FontMap) (types.Dict, error) {
	pdf := df.pdf

	id, err := types.EscapeUTF16String(df.ID)
	if err != nil {
		return nil, err
	}

	ff := df.prepareFF()

	format, err := types.Escape(fmt.Sprintf("AFDate_FormatEx(\"%s\");", df.dateFormat.Ext))
	if err != nil {
		return nil, err
	}

	keystroke, err := types.Escape(fmt.Sprintf("AFDate_KeystrokeEx(\"%s\");", df.dateFormat.Ext))
	if err != nil {
		return nil, err
	}

	aa := types.Dict(
		map[string]types.Object{
			"F": types.Dict(
				map[string]types.Object{
					"JS": types.StringLiteral(*format),
					"S":  types.Name("JavaScript"),
				},
			),
			"K": types.Dict(
				map[string]types.Object{
					"JS": types.StringLiteral(*keystroke),
					"S":  types.Name("JavaScript"),
				},
			),
		},
	)

	tu := types.StringLiteral(df.dateFormat.Ext)
	if df.Tip != "" {
		tu = types.StringLiteral(types.EncodeUTF16String(df.Tip))
	}

	d := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"FT":      types.Name("Tx"),
			"Rect":    df.BoundingBox.Array(),
			"F":       types.Integer(model.AnnPrint),
			"Ff":      types.Integer(ff),
			"T":       types.StringLiteral(*id),
			"Q":       types.Integer(df.HorAlign),
			"TU":      tu,
			"AA":      aa,
		},
	)

	df.handleBorderAndMK(d)

	if df.Value != "" {
		s, err := types.EscapeUTF16String(df.Value)
		if err != nil {
			return nil, err
		}
		d["V"] = types.StringLiteral(*s)
	}

	if df.Default != "" {
		s, err := types.EscapeUTF16String(df.Default)
		if err != nil {
			return nil, err
		}
		d["DV"] = types.StringLiteral(*s)
		if df.Value == "" {
			d["V"] = types.StringLiteral(*s)
		}
	}

	if pdf.InheritedDA != "" {
		d["DA"] = types.StringLiteral(pdf.InheritedDA)
	}

	f := df.Font
	fCol := f.col

	fontID, err := pdf.ensureFormFont(f)
	if err != nil {
		return d, err
	}
	df.fontID = fontID

	da := fmt.Sprintf("/%s %d Tf %.2f %.2f %.2f rg", fontID, f.Size, fCol.R, fCol.G, fCol.B)
	// Note: Mac Preview does not honour inherited "DA"
	d["DA"] = types.StringLiteral(da)

	irN, err := df.irN(fonts)
	if err != nil {
		return nil, err
	}

	d["AP"] = types.Dict(map[string]types.Object{"N": *irN})

	return d, nil
}

func (df *DateField) bbox() *types.Rectangle {
	if df.Label == nil {
		return df.BoundingBox.Clone()
	}

	l := df.Label
	var r *types.Rectangle
	x := l.td.X

	switch l.td.HAlign {
	case types.AlignCenter:
		x -= float64(l.Width) / 2
	case types.AlignRight:
		x -= float64(l.Width)
	}

	r = types.RectForWidthAndHeight(x, l.td.Y, float64(l.Width), l.height)

	return model.CalcBoundingBoxForRects(df.BoundingBox, r)
}

func (df *DateField) prepareRectLL(mTop, mRight, mBottom, mLeft float64) (float64, float64) {
	return df.content.calcPosition(df.x, df.y, df.Dx, df.Dy, mTop, mRight, mBottom, mLeft)
}

func (df *DateField) prepLabel(p *model.Page, pageNr int, fonts model.FontMap) error {

	if df.Label == nil {
		return nil
	}

	l := df.Label
	pdf := df.pdf

	t := "Default"
	if l.Value != "" {
		t, _ = format.Text(l.Value, pdf.TimestampFormat, pageNr, pdf.pageCount())
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	f := l.Font
	fontName, fontLang, col := f.Name, f.Lang, f.col

	id, err := df.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
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

	bb := model.WriteMultiLine(df.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	l.height = bb.Height()
	if bb.Width() > w {
		w = bb.Width()
		l.Width = int(bb.Width())
	}

	td.X, td.Y = df.labelPos(l.height, w, g)

	if bb.Height() < df.BoundingBox.Height() &&
		(l.relPos == types.RelPosLeft || l.relPos == types.RelPosRight) {
		td.MBot = (df.BoundingBox.Height() - bb.Height()) / 2
		td.MTop = td.MBot
	}

	td.HAlign, td.VAlign = l.HorAlign, types.AlignBottom

	l.td = &td

	return nil
}

func (df *DateField) prepForRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	mTop, mRight, mBottom, mLeft, err := df.calcMargin()
	if err != nil {
		return err
	}

	x, y := df.prepareRectLL(mTop, mRight, mBottom, mLeft)

	if err := df.calcFont(); err != nil {
		return err
	}

	var boWidth int
	if df.Border != nil {
		if df.Border.col != nil {
			boWidth = df.Border.Width
		}
	}

	h := float64(df.Font.Size)*1.2 + 2*float64(boWidth)

	df.BoundingBox = types.RectForWidthAndHeight(x, y, df.Width, h)

	return df.prepLabel(p, pageNr, fonts)
}

func (df *DateField) doRender(p *model.Page, fonts model.FontMap) error {

	d, err := df.prepareDict(fonts)
	if err != nil {
		return err
	}

	ann := model.FieldAnnotation{Dict: d}
	if df.Tab > 0 {
		p.AnnotTabs[df.Tab] = ann
	} else {
		p.Annots = append(p.Annots, ann)
	}

	if df.Label != nil {
		model.WriteColumn(df.pdf.XRefTable, p.Buf, p.MediaBox, nil, *df.Label.td, 0)
	}

	if df.Debug || df.pdf.Debug {
		df.pdf.highlightPos(p.Buf, df.BoundingBox.LL.X, df.BoundingBox.LL.Y, df.content.Box())
	}

	return nil
}

func (df *DateField) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := df.prepForRender(p, pageNr, fonts); err != nil {
		return err
	}

	return df.doRender(p, fonts)
}

// NewDateField returns a new date field for d.
func NewDateField(
	ctx *model.Context,
	d types.Dict,
	v string,
	fonts map[string]types.IndirectRef) (*DateField, *types.IndirectRef, error) {

	df := &DateField{Value: v}

	bb, err := types.RectForArray(d.ArrayEntry("Rect"))
	if err != nil {
		return nil, nil, err
	}

	df.BoundingBox = types.RectForDim(bb.Width(), bb.Height())

	fontIndRef, err := df.calcFontFromDA(ctx, d, fonts)
	if err != nil {
		return nil, nil, err
	}

	df.HorAlign = types.AlignLeft
	if q := d.IntEntry("Q"); q != nil {
		df.HorAlign = types.HAlignment(*q)
	}

	bgCol, boCol, err := calcColsFromMK(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	df.BgCol = bgCol

	var b Border
	boWidth := calcBorderWidth(d)
	if boWidth > 0 {
		b.Width = boWidth
		b.col = boCol
	}
	df.Border = &b

	return df, fontIndRef, nil
}

func renderDateFieldAP(ctx *model.Context, d types.Dict, v string, fonts map[string]types.IndirectRef) error {

	df, fontIndRef, err := NewDateField(ctx, d, v, fonts)
	if err != nil {
		return err
	}

	bb, err := df.renderN(ctx.XRefTable)
	if err != nil {
		return err
	}

	irN, err := NewForm(ctx.XRefTable, bb, df.fontID, fontIndRef, df.BoundingBox)
	if err != nil {
		return err
	}

	d["AP"] = types.Dict(map[string]types.Object{"N": *irN})

	return nil
}

func refreshDateFieldAP(ctx *model.Context, d types.Dict, v string, fonts map[string]types.IndirectRef, irN *types.IndirectRef) error {

	df, _, err := NewDateField(ctx, d, v, fonts)
	if err != nil {
		return err
	}

	bb, err := df.renderN(ctx.XRefTable)
	if err != nil {
		return err
	}

	return UpdateForm(ctx.XRefTable, bb, irN)
}

func EnsureDateFieldAP(ctx *model.Context, d types.Dict, v string, fonts map[string]types.IndirectRef) error {

	apd := d.DictEntry("AP")
	if apd == nil {
		return renderDateFieldAP(ctx, d, v, fonts)
	}

	irN := apd.IndirectRefEntry("N")
	if irN == nil {
		return nil
	}

	return refreshDateFieldAP(ctx, d, v, fonts, irN)
}

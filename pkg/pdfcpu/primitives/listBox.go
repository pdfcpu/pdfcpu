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
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ListBox represents a specific choice form field including a positioned label.
type ListBox struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Tip             string
	Default         string
	Defaults        []string
	Value           string
	Values          []string
	Ind             types.Array `json:"-"`
	Options         []string
	Position        [2]float64 `json:"pos"`
	x, y            float64
	Width           float64
	Height          float64
	Dx, Dy          float64
	BoundingBox     *types.Rectangle `json:"-"`
	Multi           bool             `json:"multi"`
	Font            *FormFont
	fontID          string
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

func (lb *ListBox) SetFontID(s string) {
	lb.fontID = s
}

func (lb *ListBox) validateDefault() error {
	if len(lb.Options) == 0 {
		return errors.Errorf("pdfcpu: field: %s missing options", lb.ID)
	}

	dv := []string{}
	if lb.Default != "" {
		dv = append(dv, lb.Default)
	}
	for _, v := range lb.Defaults {
		if !types.MemberOf(v, dv) {
			dv = append(dv, v)
		}
	}
	if len(dv) == 0 {
		return nil
	}

	for _, v := range dv {
		if !types.MemberOf(v, lb.Options) {
			return errors.Errorf("pdfcpu: field: %s invalid default: %s", lb.ID, v)
		}
	}

	if !lb.Multi && len(dv) > 1 {
		return errors.Errorf("pdfcpu: field %s only 1 value allowed", lb.ID)
	}

	lb.Defaults = dv

	return nil
}

func (lb *ListBox) validateValue() error {
	if len(lb.Options) == 0 {
		return errors.Errorf("pdfcpu: field: %s missing options", lb.ID)
	}

	vv := []string{}
	if lb.Value != "" {
		vv = append(vv, lb.Value)
	}
	for _, v1 := range lb.Values {
		if !types.MemberOf(v1, vv) {
			vv = append(vv, v1)
		}
	}
	if len(vv) == 0 {
		return nil
	}

	for _, v := range vv {
		if !types.MemberOf(v, lb.Options) {
			return errors.Errorf("pdfcpu: field: %s invalid default: %s", lb.ID, v)
		}
	}

	if !lb.Multi && len(vv) > 1 {
		return errors.Errorf("pdfcpu: field %s only 1 value allowed", lb.ID)
	}

	lb.Values = vv

	return nil
}

func (lb *ListBox) validateID() error {
	if lb.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if lb.pdf.DuplicateField(lb.ID) {
		return errors.Errorf("pdfcpu: duplicate form field: %s", lb.ID)
	}
	lb.pdf.FieldIDs[lb.ID] = true
	return nil
}

func (lb *ListBox) validatePosition() error {
	if lb.Position[0] < 0 || lb.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", lb.ID)
	}
	lb.x, lb.y = lb.Position[0], lb.Position[1]
	return nil
}

func (lb *ListBox) validateWidth() error {
	if lb.Width == 0 {
		return errors.Errorf("pdfcpu: field: %s width == 0", lb.ID)
	}
	return nil
}

func (lb *ListBox) validateHeight() error {
	if lb.Height < 0 {
		return errors.Errorf("pdfcpu: field: %s height < 0", lb.ID)
	}
	return nil
}

func (lb *ListBox) validateFont() error {
	if lb.Font != nil {
		lb.Font.pdf = lb.pdf
		if err := lb.Font.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBox) validateMargin() error {
	if lb.Margin != nil {
		if err := lb.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBox) validateBorder() error {
	if lb.Border != nil {
		lb.Border.pdf = lb.pdf
		if err := lb.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBox) validateBackgroundColor() error {
	if lb.BackgroundColor != "" {
		sc, err := lb.pdf.parseColor(lb.BackgroundColor)
		if err != nil {
			return err
		}
		lb.BgCol = sc
	}
	return nil
}

func (lb *ListBox) validateHorAlign() error {
	lb.HorAlign = types.AlignLeft
	if lb.Alignment != "" {
		ha, err := types.ParseHorAlignment(lb.Alignment)
		if err != nil {
			return err
		}
		lb.HorAlign = ha
	}
	return nil
}

func (lb *ListBox) validateLabel() error {
	if lb.Label != nil {
		lb.Label.pdf = lb.pdf
		if err := lb.Label.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *ListBox) validateTab() error {
	if lb.Tab < 0 {
		return errors.Errorf("pdfcpu: field: %s negative tab value", lb.ID)
	}
	if lb.Tab == 0 {
		return nil
	}
	page := lb.content.page
	if page.Tabs == nil {
		page.Tabs = types.IntSet{}
	} else {
		if page.Tabs[lb.Tab] {
			return errors.Errorf("pdfcpu: field: %s duplicate tab value %d", lb.ID, lb.Tab)
		}
	}
	page.Tabs[lb.Tab] = true
	return nil
}

func (lb *ListBox) validate() error {

	if err := lb.validateID(); err != nil {
		return err
	}

	if err := lb.validatePosition(); err != nil {
		return err
	}

	if err := lb.validateWidth(); err != nil {
		return err
	}

	if err := lb.validateHeight(); err != nil {
		return err
	}

	if err := lb.validateDefault(); err != nil {
		return err
	}

	if err := lb.validateValue(); err != nil {
		return err
	}

	if err := lb.validateFont(); err != nil {
		return err
	}

	if err := lb.validateMargin(); err != nil {
		return err
	}

	if err := lb.validateBorder(); err != nil {
		return err
	}

	if err := lb.validateBackgroundColor(); err != nil {
		return err
	}

	if err := lb.validateHorAlign(); err != nil {
		return err
	}

	if err := lb.validateLabel(); err != nil {
		return err
	}

	return lb.validateTab()
}

func (lb *ListBox) calcFontFromDA(ctx *model.Context, d types.Dict, fonts map[string]types.IndirectRef) (*types.IndirectRef, error) {

	s := d.StringEntry("DA")
	if s == nil {
		s = ctx.Form.StringEntry("DA")
		if s == nil {
			return nil, errors.New("pdfcpu: listbox missing \"DA\"")
		}
	}

	fontID, f, err := fontFromDA(*s)
	if err != nil {
		return nil, err
	}

	lb.Font, lb.fontID = &f, fontID

	id, name, lang, fontIndRef, err := extractFormFontDetails(ctx, lb.fontID, fonts)
	if err != nil {
		return nil, err
	}
	if fontIndRef == nil {
		return nil, errors.New("pdfcpu: unable to detect indirect reference for font")
	}

	lb.fontID = id
	lb.Font.Name = name
	lb.Font.Lang = lang
	lb.RTL = pdffont.RTL(lang)

	return fontIndRef, nil
}

func (lb *ListBox) calcFont() error {
	f, err := lb.content.calcInputFont(lb.Font)
	if err != nil {
		return err
	}
	lb.Font = f

	if lb.Label != nil {
		f, err = lb.content.calcLabelFont(lb.Label.Font)
		if err != nil {
			return err
		}
		lb.Label.Font = f
	}

	return nil
}

func (lb *ListBox) margin(name string) *Margin {
	return lb.content.namedMargin(name)
}

func (lb *ListBox) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if lb.Margin != nil {
		m := lb.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := lb.margin(mName)
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

func (lb *ListBox) labelPos(labelHeight, w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := lb.BoundingBox, lb.Label.HorAlign

	switch lb.Label.relPos {

	case types.RelPosLeft:
		x = bb.LL.X - g
		if horAlign == types.AlignLeft {
			x -= w
			if x < 0 {
				x = 0
			}
		}
		y = bb.UR.Y - labelHeight

	case types.RelPosRight:
		x = bb.UR.X + g
		if horAlign == types.AlignRight {
			x += w
		}
		y = bb.UR.Y - labelHeight

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

func selectItem(w io.Writer, i int, width, height float64, fontName string, fontSize int, boWidth float64, col color.SimpleColor) {
	lh := font.LineHeight(fontName, fontSize)
	fmt.Fprintf(w, "%.2f %.2f %.2f rg 1 %.2f %.2f %.2f re f ",
		col.R, col.G, col.B,
		height-boWidth-float64(i+1)*lh, width-2, lh)
}

func (lb *ListBox) renderN(xRefTable *model.XRefTable) ([]byte, error) {
	w, h := lb.BoundingBox.Width(), lb.BoundingBox.Height()
	bgCol := lb.BgCol
	boWidth, boCol := lb.calcBorder()
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

	f, ind := lb.Font, lb.Ind
	selCol := color.SimpleColor{R: 0.600006, G: 0.756866, B: 0.854904}
	for i := 0; i < len(ind); i++ {
		j := ind[i].(types.Integer).Value()
		selectItem(buf, j, w, h, f.Name, f.Size, boWidth, selCol)
	}

	x := 2 * boWidth
	if x == 0 {
		x = 2
	}
	h0 := h + font.Descent(f.Name, f.Size) - boWidth
	lh := font.LineHeight(f.Name, f.Size)

	opts := lb.Options
	for i := 0; i < len(opts); i++ {
		s := opts[i]
		if font.IsCoreFont(f.Name) && utf8.ValidString(s) {
			s = model.DecodeUTF8ToByte(s)
		}
		lineBB := model.CalcBoundingBox(s, 0, 0, f.Name, f.Size)
		s = model.PrepBytes(xRefTable, s, f.Name, false, lb.RTL)
		x := 2 * boWidth
		if x == 0 {
			x = 2
		}
		switch lb.HorAlign {
		case types.AlignCenter:
			x = w/2 - lineBB.Width()/2
		case types.AlignRight:
			x = w - lineBB.Width() - 2
		}
		fmt.Fprint(buf, "BT ")
		if i == 0 {
			fmt.Fprintf(buf, "/%s %d Tf %.2f %.2f %.2f RG %.2f %.2f %.2f rg ",
				lb.fontID, f.Size,
				f.col.R, f.col.G, f.col.B,
				f.col.R, f.col.G, f.col.B)
		}
		fmt.Fprintf(buf, "%.2f %.2f Td (%s) Tj ET ", x, h0-float64(i+1)*lh, s)
	}

	fmt.Fprint(buf, "Q EMC ")

	if boCol != nil {
		fmt.Fprintf(buf, "q %.2f %.2f %.2f RG %.2f w %.2f %.2f %.2f %.2f re s Q ",
			boCol.R, boCol.G, boCol.B, boWidth, boWidth/2, boWidth/2, w-boWidth, h-boWidth)
	}

	return buf.Bytes(), nil
}

func (lb *ListBox) irN(fonts model.FontMap) (*types.IndirectRef, error) {
	pdf := lb.pdf

	bb, err := lb.renderN(lb.pdf.XRefTable)
	if err != nil {
		return nil, err
	}

	sd, err := pdf.XRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, lb.BoundingBox.Width(), lb.BoundingBox.Height()))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	ir, err := pdf.ensureFont(lb.fontID, lb.Font.Name, lb.Font.Lang, fonts)
	if err != nil {
		return nil, err
	}

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					lb.fontID: *ir,
				},
			),
		},
	)

	sd.Insert("Resources", d)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	return pdf.XRefTable.IndRefForNewObject(*sd)
}

func (lb *ListBox) calcBorder() (boWidth float64, boCol *color.SimpleColor) {
	if lb.Border == nil {
		return 0, nil
	}
	return lb.Border.calc()
}

func (lb *ListBox) prepareFF() FieldFlags {
	ff := FieldFlags(0)
	if lb.Multi {
		// Note: unsupported in Mac Preview
		ff += FieldMultiselect
	}
	if lb.Locked {
		ff += FieldReadOnly
	}
	return ff
}

func (lb *ListBox) handleBorderAndMK(d types.Dict) {
	bgCol := lb.BgCol
	if bgCol == nil {
		bgCol = lb.content.page.bgCol
		if bgCol == nil {
			bgCol = lb.pdf.bgCol
		}
	}
	lb.BgCol = bgCol

	boWidth, boCol := lb.calcBorder()

	if bgCol != nil || boCol != nil {
		appCharDict := types.Dict{}
		if bgCol != nil {
			appCharDict["BG"] = bgCol.Array()
		}
		if boCol != nil && boWidth > 0 {
			appCharDict["BC"] = boCol.Array()
		}
		d["MK"] = appCharDict
	}

	if boWidth > 0 {
		d["Border"] = types.NewNumberArray(0, 0, boWidth)
	}
}

func (lb *ListBox) handleVAndDV(d types.Dict) error {
	vv := lb.Values
	if len(vv) == 0 {
		vv = lb.Defaults
	}
	ind := types.Array{}
	arr := types.Array{}
	for _, v := range vv {
		for i, o := range lb.Options {
			if o == v {
				ind = append(ind, types.Integer(i))
			}
		}
		s, err := types.EscapeUTF16String(v)
		if err != nil {
			return err
		}
		arr = append(arr, types.StringLiteral(*s))
	}
	if len(arr) == 1 {
		d["V"] = arr[0]
		d["I"] = ind
		lb.Ind = ind
	}
	if len(arr) > 1 {
		d["V"] = arr
		d["I"] = ind
		lb.Ind = ind
	}

	arr = types.Array{}
	for _, v := range lb.Defaults {
		s, err := types.EscapeUTF16String(v)
		if err != nil {
			return err
		}
		arr = append(arr, types.StringLiteral(*s))
	}
	if len(arr) == 1 {
		d["DV"] = arr[0]
	}
	if len(arr) > 1 {
		d["DV"] = arr
	}

	return nil
}

func (lb *ListBox) prepareDict(fonts model.FontMap) (types.Dict, error) {
	pdf := lb.pdf

	id, err := types.EscapeUTF16String(lb.ID)
	if err != nil {
		return nil, err
	}

	opt := types.Array{}
	for _, s := range lb.Options {
		s1, err := types.EscapeUTF16String(s)
		if err != nil {
			return nil, err
		}
		opt = append(opt, types.StringLiteral(*s1))
	}

	ff := lb.prepareFF()

	d := types.Dict(
		map[string]types.Object{
			"Type":    types.Name("Annot"),
			"Subtype": types.Name("Widget"),
			"FT":      types.Name("Ch"),
			"Rect":    lb.BoundingBox.Array(),
			"F":       types.Integer(model.AnnPrint),
			"Ff":      types.Integer(ff),
			"Opt":     opt,
			"Q":       types.Integer(lb.HorAlign),
			"T":       types.StringLiteral(*id),
		},
	)

	if lb.Tip != "" {
		tu, err := types.EscapeUTF16String(lb.Tip)
		if err != nil {
			return nil, err
		}
		d["TU"] = types.StringLiteral(*tu)
	}

	lb.handleBorderAndMK(d)

	if err := lb.handleVAndDV(d); err != nil {
		return nil, err
	}

	if pdf.InheritedDA != "" {
		d["DA"] = types.StringLiteral(pdf.InheritedDA)
	}

	f := lb.Font
	fCol := f.col

	fontID, err := pdf.ensureFormFont(f)
	if err != nil {
		return nil, err
	}
	lb.fontID = fontID

	da := fmt.Sprintf("/%s %d Tf %.2f %.2f %.2f rg", fontID, f.Size, fCol.R, fCol.G, fCol.B)
	// Note: Mac Preview does not honour inherited "DA"
	d["DA"] = types.StringLiteral(da)

	irN, err := lb.irN(fonts)
	if err != nil {
		return nil, err
	}

	d["AP"] = types.Dict(map[string]types.Object{"N": *irN})

	return d, nil
}

func (lb *ListBox) bbox() *types.Rectangle {
	if lb.Label == nil {
		return lb.BoundingBox.Clone()
	}

	l := lb.Label
	var r *types.Rectangle
	x := l.td.X

	switch l.td.HAlign {
	case types.AlignCenter:
		x -= float64(l.Width) / 2
	case types.AlignRight:
		x -= float64(l.Width)
	}

	r = types.RectForWidthAndHeight(x, l.td.Y, float64(l.Width), l.height)

	return model.CalcBoundingBoxForRects(lb.BoundingBox, r)
}

func (lb *ListBox) prepareRectLL(mTop, mRight, mBottom, mLeft float64) (float64, float64) {
	return lb.content.calcPosition(lb.x, lb.y, lb.Dx, lb.Dy, mTop, mRight, mBottom, mLeft)
}

func (lb *ListBox) prepLabel(p *model.Page, pageNr int, fonts model.FontMap) error {

	if lb.Label == nil {
		return nil
	}

	l := lb.Label
	v := l.Value
	w := float64(l.Width)
	g := float64(l.Gap)

	f := l.Font
	fontName, fontLang, col := f.Name, f.Lang, f.col

	id, err := lb.pdf.idForFontName(fontName, fontLang, p.Fm, fonts, pageNr)
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

	bb := model.WriteMultiLine(lb.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
	l.height = bb.Height()
	if bb.Width() > w {
		w = bb.Width()
		l.Width = int(bb.Width())
	}

	td.X, td.Y = lb.labelPos(l.height, w, g)
	td.HAlign, td.VAlign = l.HorAlign, types.AlignBottom

	l.td = &td

	return nil
}

func (lb *ListBox) prepForRender(p *model.Page, pageNr int, fonts model.FontMap) error {

	mTop, mRight, mBottom, mLeft, err := lb.calcMargin()
	if err != nil {
		return err
	}

	x, y := lb.prepareRectLL(mTop, mRight, mBottom, mLeft)

	if err := lb.calcFont(); err != nil {
		return err
	}

	if lb.Width < 0 {
		// Extend width to maxWidth.
		r := lb.content.Box().CroppedCopy(0)
		r.LL.X += mLeft
		r.LL.Y += mBottom
		r.UR.X -= mRight
		r.UR.Y -= mTop
		lb.Width = r.Width() - lb.x

	}

	lb.BoundingBox = types.RectForWidthAndHeight(x, y, lb.Width, lb.Height)

	return lb.prepLabel(p, pageNr, fonts)
}

func (lb *ListBox) doRender(p *model.Page, fonts model.FontMap) error {

	d, err := lb.prepareDict(fonts)
	if err != nil {
		return err
	}

	ann := model.FieldAnnotation{Dict: d}
	if lb.Tab > 0 {
		p.AnnotTabs[lb.Tab] = ann
	} else {
		p.Annots = append(p.Annots, ann)
	}

	if lb.Label != nil {
		model.WriteColumn(lb.pdf.XRefTable, p.Buf, p.MediaBox, nil, *lb.Label.td, 0)
	}

	if lb.Debug || lb.pdf.Debug {
		lb.pdf.highlightPos(p.Buf, lb.BoundingBox.LL.X, lb.BoundingBox.LL.Y, lb.content.Box())
	}

	return nil
}

func (lb *ListBox) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := lb.prepForRender(p, pageNr, fonts); err != nil {
		return err
	}

	return lb.doRender(p, fonts)
}

// NewListBox creates a new listbox for d.
func NewListBox(
	ctx *model.Context,
	d types.Dict,
	opts []string,
	ind types.Array,
	fonts map[string]types.IndirectRef) (*ListBox, *types.IndirectRef, error) {

	lb := &ListBox{Options: opts, Ind: ind}

	bb, err := types.RectForArray(d.ArrayEntry("Rect"))
	if err != nil {
		return nil, nil, err
	}

	lb.BoundingBox = types.RectForDim(bb.Width(), bb.Height())

	fontIndRef, err := lb.calcFontFromDA(ctx, d, fonts)
	if err != nil {
		return nil, nil, err
	}

	lb.HorAlign = types.AlignLeft
	if q := d.IntEntry("Q"); q != nil {
		lb.HorAlign = types.HAlignment(*q)
	}

	bgCol, boCol, err := calcColsFromMK(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	lb.BgCol = bgCol

	var b Border
	boWidth := calcBorderWidth(d)
	if boWidth > 0 {
		b.Width = boWidth
		b.col = boCol
	}
	lb.Border = &b

	return lb, fontIndRef, nil
}

func NewForm(
	xRefTable *model.XRefTable,
	bb []byte,
	fontID string,
	fontIndRef *types.IndirectRef,
	boundingBox *types.Rectangle) (*types.IndirectRef, error) {

	sd, err := xRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", types.NewNumberArray(0, 0, boundingBox.Width(), boundingBox.Height()))
	sd.Insert("Matrix", types.NewNumberArray(1, 0, 0, 1, 0, 0))

	d := types.Dict(
		map[string]types.Object{
			"Font": types.Dict(
				map[string]types.Object{
					fontID: *fontIndRef,
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

func UpdateForm(xRefTable *model.XRefTable, bb []byte, indRef *types.IndirectRef) error {

	entry, _ := xRefTable.FindTableEntryForIndRef(indRef)

	sd := entry.Object.(types.StreamDict)

	sd.Content = bb
	if err := sd.Encode(); err != nil {
		return err
	}

	entry.Object = sd

	return nil
}

func renderListBoxAP(ctx *model.Context, d types.Dict, opts []string, ind types.Array, fonts map[string]types.IndirectRef) error {

	lb, fontIndRef, err := NewListBox(ctx, d, opts, ind, fonts)
	if err != nil {
		return err
	}

	bb, err := lb.renderN(ctx.XRefTable)
	if err != nil {
		return err
	}

	irN, err := NewForm(ctx.XRefTable, bb, lb.fontID, fontIndRef, lb.BoundingBox)
	if err != nil {
		return err
	}

	d["AP"] = types.Dict(map[string]types.Object{"N": *irN})

	return nil
}

func refreshListBoxAP(ctx *model.Context, d types.Dict, opts []string, ind types.Array, fonts map[string]types.IndirectRef, irN *types.IndirectRef) error {

	lb, _, err := NewListBox(ctx, d, opts, ind, fonts)
	if err != nil {
		return err
	}

	bb, err := lb.renderN(ctx.XRefTable)
	if err != nil {
		return err
	}

	return UpdateForm(ctx.XRefTable, bb, irN)
}

func EnsureListBoxAP(ctx *model.Context, d types.Dict, opts []string, ind types.Array, fonts map[string]types.IndirectRef) error {

	apd := d.DictEntry("AP")
	if apd == nil {
		return renderListBoxAP(ctx, d, opts, ind, fonts)
	}

	irN := apd.IndirectRefEntry("N")
	if irN == nil {
		return nil
	}

	return refreshListBoxAP(ctx, d, opts, ind, fonts, irN)
}

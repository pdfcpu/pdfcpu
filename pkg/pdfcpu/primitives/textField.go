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
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// FieldFlags represents the PDF form field flags.
type FieldFlags int

const ( // See table 221 et.al.
	FieldReadOnly FieldFlags = 1 << iota
	FieldRequired
	FieldNoExport
	UnusedFlag4
	UnusedFlag5
	UnusedFlag6
	UnusedFlag7
	UnusedFlag8
	UnusedFlag9
	UnusedFlag10
	UnusedFlag11
	UnusedFlag12
	FieldMultiline
	FieldPassword
	FieldNoToggleToOff
	FieldRadio
	FieldPushbutton
	FieldCombo
	FieldEdit
	FieldSort
	FieldFileSelect
	FieldMultiselect
	FieldDoNotSpellCheck
	FieldDoNotScroll
	FieldComb
	FieldRichTextAndRadiosInUnison
	FieldCommitOnSelChange
)

type TextField struct {
	pdf             *PDF
	content         *Content
	Label           *TextFieldLabel
	ID              string
	Value           string
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Width           float64
	Height          float64
	Dx, Dy          float64
	boundingBox     *pdfcpu.Rectangle
	Multiline       bool
	Font            *FormFont // optional
	Margin          *Margin   // applied to content box
	Border          *Border
	BackgroundColor string `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor
	Alignment       string `json:"align"` // "Left", "Center", "Right"
	horAlign        pdfcpu.HAlignment
	RTL             bool
	Rotation        float64 `json:"rot"`
	Hide            bool
}

func (tf *TextField) validate() error {

	if tf.ID == "" {
		return errors.New("pdfcpu: missing field id")
	}
	if tf.pdf.FieldIDs[tf.ID] {
		return errors.Errorf("pdfcpu: duplicate form field: %s", tf.ID)
	}
	tf.pdf.FieldIDs[tf.ID] = true

	if tf.Position[0] < 0 || tf.Position[1] < 0 {
		return errors.Errorf("pdfcpu: field: %s pos value < 0", tf.ID)
	}
	tf.x, tf.y = tf.Position[0], tf.Position[1]

	if tf.Width <= 0 {
		return errors.Errorf("pdfcpu: field: %s width <= 0", tf.ID)
	}

	if tf.Height < 0 {
		return errors.Errorf("pdfcpu: field: %s height < 0", tf.ID)
	}

	if tf.Font != nil {
		tf.Font.pdf = tf.pdf
		if err := tf.Font.validate(); err != nil {
			return err
		}
	}

	if tf.Margin != nil {
		if err := tf.Margin.validate(); err != nil {
			return err
		}
	}

	if tf.Border != nil {
		tf.Border.pdf = tf.pdf
		if err := tf.Border.validate(); err != nil {
			return err
		}
	}

	if tf.BackgroundColor != "" {
		sc, err := tf.pdf.parseColor(tf.BackgroundColor)
		if err != nil {
			return err
		}
		tf.bgCol = sc
	}

	tf.horAlign = pdfcpu.AlignLeft
	if tf.Alignment != "" {
		ha, err := pdfcpu.ParseHorAlignment(tf.Alignment)
		if err != nil {
			return err
		}
		tf.horAlign = ha
	}

	if tf.Label != nil {
		tf.Label.pdf = tf.pdf
		if err := tf.Label.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (tf *TextField) font(name string) *FormFont {
	return tf.content.namedFont(name)
}

func (tf *TextField) margin(name string) *Margin {
	return tf.content.namedMargin(name)
}

func (tf *TextField) labelPos(labelHeight, w, g float64) (float64, float64) {

	var x, y float64
	bb, horAlign := tf.boundingBox, tf.Label.horAlign

	switch tf.Label.relPos {

	case pdfcpu.RelPosLeft:
		x = bb.LL.X - g
		if horAlign == pdfcpu.AlignLeft {
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

	case pdfcpu.RelPosRight:
		x = bb.UR.X + g
		if horAlign == pdfcpu.AlignRight {
			x += w
		}
		if tf.Multiline {
			y = bb.UR.Y - labelHeight
		} else {
			y = bb.LL.Y
		}

	case pdfcpu.RelPosTop:
		y = bb.UR.Y + g
		x = bb.LL.X
		if horAlign == pdfcpu.AlignRight {
			x += bb.Width()
		} else if horAlign == pdfcpu.AlignCenter {
			x += bb.Width() / 2
		}

	case pdfcpu.RelPosBottom:
		y = bb.LL.Y - g - labelHeight
		x = bb.LL.X
		if horAlign == pdfcpu.AlignRight {
			x += bb.Width()
		} else if horAlign == pdfcpu.AlignCenter {
			x += bb.Width() / 2
		}

	}

	return x, y
}

func (tf *TextField) renderLabel(r *pdfcpu.Rectangle, p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap, fontSize int) error {

	if tf.Label == nil {
		return nil
	}

	l := tf.Label
	pdf := tf.pdf

	t := "Default"
	if l.Value != "" {
		t, _ = pdfcpu.ResolveWMTextString(l.Value, pdf.TimestampFormat, pageNr, pdf.pageCount())
	}

	w := float64(l.Width)
	g := float64(l.Gap)

	var f *FormFont

	if l.Font != nil {
		f = l.Font
		if f.Name[0] == '$' {
			// use named font
			fName := f.Name[1:]
			f0 := tf.font(fName)
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
		f = tf.font("label")
		if f == nil {
			return errors.Errorf("pdfcpu: missing named font \"label\"")
		}
	}

	if f.col == nil {
		f.col = &pdfcpu.Black
	}

	fontName := f.Name

	// Enforce input fontSize.
	//fontSize := f.Size
	col := f.col

	id, err := tf.pdf.idForFontName(fontName, p.Fm, fonts, pageNr)
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
		RTL:      l.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if l.bgCol != nil {
		//td.ShowBorder = true
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *l.bgCol
	}

	// TODO Ensure bb.Height = tf.boundingBox.Height() with td.Height =  tf.boundingBox.Height() ???
	bb := pdfcpu.WriteMultiLine(new(bytes.Buffer), pdfcpu.RectForFormat("A4"), nil, td)

	td.X, td.Y = tf.labelPos(bb.Height(), w, g)

	if !tf.Multiline && bb.Height() < tf.boundingBox.Height() {
		td.MBot = (tf.boundingBox.Height() - bb.Height()) / 2
		td.MTop = td.MBot
	}

	td.HAlign, td.VAlign = l.horAlign, pdfcpu.AlignBottom

	pdfcpu.WriteColumn(p.Buf, p.MediaBox, nil, td, 0)

	return nil
}

func (tf *TextField) ensureFont(fontID, fontName string, fonts pdfcpu.FontMap) (*pdfcpu.IndirectRef, error) {
	pdf := tf.pdf
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

func (tf *TextField) renderText(lines []string, da, fontName string, fontSize int) []byte {

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "/Tx BMC q 1 1 %.1f %.1f re W n ", tf.boundingBox.Width()-2, tf.boundingBox.Height()-2)

	lh := font.LineHeight(fontName, fontSize)
	y := (tf.boundingBox.Height()-font.LineHeight(fontName, fontSize))/2 + font.Descent(fontName, fontSize)
	if len(lines) > 1 {
		y = tf.boundingBox.Height() - font.LineHeight(fontName, fontSize)
	}

	for _, s := range lines {
		lineBB := pdfcpu.CalcBoundingBox(s, 0, 0, fontName, fontSize)
		w := tf.boundingBox.Width()
		// Apply horizontal alignment.
		x := 2.
		switch tf.horAlign {
		case pdfcpu.AlignCenter:
			x = w/2 - lineBB.Width()/2
		case pdfcpu.AlignRight:
			x = w - lineBB.Width() - 2
		}
		s = pdfcpu.PrepBytes(s, fontName, tf.RTL)
		fmt.Fprintf(buf, "BT %s %.2f %.2f Td (%s) Tj ET ", da, x, y, s)
		y -= lh
	}
	fmt.Fprint(buf, "Q EMC ")
	return buf.Bytes()
}

func (tf *TextField) irN(fontID, fontName string, fontSize int, col *pdfcpu.SimpleColor, da string, fonts pdfcpu.FontMap) (*pdfcpu.IndirectRef, error) {

	s := tf.Value
	if font.IsCoreFont(fontName) && utf8.ValidString(s) {
		s = pdfcpu.DecodeUTF8ToByte(s)
	}

	lines := pdfcpu.SplitMultilineStr(s)

	// TODO if no multifield then ignore \n = leave right now
	bb := tf.renderText(lines, da, fontName, fontSize)

	sd, err := tf.pdf.XRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "XObject")
	sd.InsertName("Subtype", "Form")
	sd.InsertInt("FormType", 1)
	sd.Insert("BBox", pdfcpu.NewNumberArray(0, 0, tf.boundingBox.Width(), tf.boundingBox.Height()))
	sd.Insert("Matrix", pdfcpu.NewNumberArray(1, 0, 0, 1, 0, 0))

	ir, err := tf.ensureFont(fontID, fontName, fonts)
	if err != nil {
		return nil, err
	}

	d := pdfcpu.Dict(
		map[string]pdfcpu.Object{
			"Font": pdfcpu.Dict(
				map[string]pdfcpu.Object{
					fontID: *ir,
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

func (tf *TextField) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap) error {

	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if tf.Margin != nil {
		m := tf.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := tf.margin(mName)
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

	cBox := tf.content.Box()
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	pdf := tf.pdf
	x, y := pdfcpu.NormalizeCoord(tf.x, tf.y, cBox, pdf.origin, false)

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
	x += tf.Dx
	y += tf.Dy

	var f *FormFont

	if tf.Font != nil {
		f = tf.Font
		if f.Name[0] == '$' {
			// use named font
			fName := f.Name[1:]
			f0 := tf.font(fName)
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
		// Use inherited named font "input".
		f = tf.font("input")
		if f == nil {
			return errors.Errorf("pdfcpu: missing named font \"input\"")
		}
	}

	var (
		fontName string
		fontSize int
	)

	if f != nil {

		if f.col == nil {
			f.col = &pdfcpu.Black
		}

		fontName = f.Name
		fontSize = f.Size
	}

	h := float64(fontSize) * 1.2
	if tf.Multiline {
		if tf.Height == 0 {
			return errors.Errorf("pdfcpu: field: %s height == 0", tf.ID)
		}
		h = tf.Height
	}
	tf.boundingBox = pdfcpu.RectForWidthAndHeight(x, y, tf.Width, h)

	id := pdfcpu.StringLiteral(pdfcpu.EncodeUTF16String(tf.ID))

	ff := FieldDoNotSpellCheck
	if tf.Multiline {
		// If set, the field may contain multiple lines of text;
		// if clear, the field’s text shall be restricted to a single line.
		// Adobe Reader ok, Mac Preview nope
		ff += FieldMultiline
	} else {
		// If set, the field shall not scroll (horizontally for single-line fields, vertically for multiple-line fields)
		// to accommodate more text than fits within its annotation rectangle.
		// Once the field is full, no further text shall be accepted for interactive form filling;
		// for non- interactive form filling, the filler should take care
		// not to add more character than will visibly fit in the defined area.
		// Adobe Reader ok, Mac Preview nope
		ff += FieldDoNotScroll
	}

	d := pdfcpu.Dict(
		map[string]pdfcpu.Object{
			"Type":    pdfcpu.Name("Annot"),
			"Subtype": pdfcpu.Name("Widget"),
			"FT":      pdfcpu.Name("Tx"),
			"Rect":    tf.boundingBox.Array(),
			"F":       pdfcpu.Integer(pdfcpu.AnnPrint),
			"Ff":      pdfcpu.Integer(ff),
			"Q":       pdfcpu.Integer(tf.horAlign), // Adjustment: (0:L) 1:C 2:R
			"T":       id,
			"TU":      id,
		},
	)

	if tf.bgCol != nil || tf.Border != nil {
		appCharDict := pdfcpu.Dict{}
		if tf.bgCol != nil {
			appCharDict["BG"] = tf.bgCol.Array()
		}
		if tf.Border != nil && tf.Border.col != nil && tf.Border.Width > 0 {
			appCharDict["BC"] = tf.Border.col.Array()
		}
		d["MK"] = appCharDict
	}

	if tf.Border != nil && tf.Border.Width > 0 {
		d["Border"] = pdfcpu.NewIntegerArray(0, 0, tf.Border.Width)
	}

	if tf.Value != "" {
		sl := pdfcpu.StringLiteral(pdfcpu.EncodeUTF16String(tf.Value))
		d["DV"] = sl
		d["V"] = sl
	}

	if pdf.InheritedDA != "" {
		d["DA"] = pdfcpu.StringLiteral(pdf.InheritedDA)
	}

	if f != nil {

		col := f.col

		fontID, err := pdf.ensureFormFont(fontName)
		if err != nil {
			return err
		}

		da := fmt.Sprintf("/%s %d Tf %.2f %.2f %.2f rg", fontID, fontSize, col.R, col.G, col.B)
		// Note: Mac Preview does not honor inherited "DA"
		d["DA"] = pdfcpu.StringLiteral(da)

		if tf.Value != "" {
			irN, err := tf.irN(fontID, fontName, fontSize, col, da, fonts)
			if err != nil {
				return err
			}
			d["AP"] = pdfcpu.Dict(map[string]pdfcpu.Object{"N": *irN})
		}
	}

	p.Annots = append(p.Annots, d)

	return tf.renderLabel(r, p, pageNr, fonts, fontSize)
}

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

type Content struct {
	parent          *Content
	page            *PDFPage
	BackgroundColor string               `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor  // page background color
	Fonts           map[string]*FormFont // named fonts
	Margins         map[string]*Margin   // named margins
	Borders         map[string]*Border   // named borders
	Paddings        map[string]*Padding  // named paddings
	Margin          *Margin              // content margin
	Border          *Border              // content border
	Padding         *Padding             // content padding
	Regions         *Regions
	mediaBox        *pdfcpu.Rectangle
	borderRect      *pdfcpu.Rectangle
	box             *pdfcpu.Rectangle
	Guides          []*Guide              // hor/vert guidelines for layout
	Bars            []*Bar                `json:"bar"`
	SimpleBoxes     []*SimpleBox          `json:"box"`
	SimpleBoxPool   map[string]*SimpleBox `json:"boxes"`
	TextBoxes       []*TextBox            `json:"text"`
	TextBoxPool     map[string]*TextBox   `json:"texts"`
	ImageBoxes      []*ImageBox           `json:"image"`
	ImageBoxPool    map[string]*ImageBox  `json:"images"`
	Tables          []*Table              `json:"table"`
	TablePool       map[string]*Table     `json:"tables"`
	// Form elements
	TextFields        []*TextField        `json:"textfield"`        // input text fields with optional label
	CheckBoxes        []*CheckBox         `json:"checkbox"`         // input checkboxes with optional label
	RadioButtonGroups []*RadioButtonGroup `json:"radiobuttongroup"` // input radiobutton groups with optional label
}

func (c *Content) validate() error {
	pdf := c.page.pdf
	if c.BackgroundColor != "" {
		sc, err := pdf.parseColor(c.BackgroundColor)
		if err != nil {
			return err
		}
		c.bgCol = sc
	}

	for _, g := range c.Guides {
		g.validate()
	}

	if c.Border != nil {
		if len(c.Borders) > 0 {
			return errors.New("pdfcpu: Please supply either content \"border\" or \"borders\"")
		}
		c.Border.pdf = pdf
		if err := c.Border.validate(); err != nil {
			return err
		}
		c.Borders = map[string]*Border{}
		c.Borders["border"] = c.Border
	}
	for _, b := range c.Borders {
		b.pdf = pdf
		if err := b.validate(); err != nil {
			return err
		}
	}

	if c.Margin != nil {
		if len(c.Margins) > 0 {
			return errors.New("pdfcpu: Please supply either page \"margin\" or \"margins\"")
		}
		if err := c.Margin.validate(); err != nil {
			return err
		}
		c.Margins = map[string]*Margin{}
		c.Margins["margin"] = c.Margin
	}
	for _, m := range c.Margins {
		if err := m.validate(); err != nil {
			return err
		}
	}

	if c.Padding != nil {
		if len(c.Paddings) > 0 {
			return errors.New("pdfcpu: Please supply either page \"padding\" or \"paddings\"")
		}
		if err := c.Padding.validate(); err != nil {
			return err
		}
		c.Paddings = map[string]*Padding{}
		c.Paddings["padding"] = c.Padding
	}
	for _, p := range c.Paddings {
		if err := p.validate(); err != nil {
			return err
		}
	}

	if c.Regions != nil {
		s := "must be defined within region content"
		if len(c.SimpleBoxPool) > 0 {
			return errors.Errorf("pdfcpu: \"boxes\" %s", s)
		}
		if len(c.SimpleBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"box\" %s", s)
		}
		if len(c.TextBoxPool) > 0 {
			return errors.Errorf("pdfcpu: \"texts\" %s", s)
		}
		if len(c.TextBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"text\" %s", s)
		}
		if len(c.ImageBoxPool) > 0 {
			return errors.Errorf("pdfcpu: \"images\" %s", s)
		}
		if len(c.ImageBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"image\" %s", s)
		}
		if len(c.TablePool) > 0 {
			return errors.Errorf("pdfcpu: \"tables\" %s", s)
		}
		if len(c.Tables) > 0 {
			return errors.Errorf("pdfcpu: \"table\" %s", s)
		}
		if len(c.TextFields) > 0 {
			return errors.Errorf("pdfcpu: \"textfield\" %s", s)
		}
		if len(c.CheckBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"checkbox\" %s", s)
		}
		if len(c.RadioButtonGroups) > 0 {
			return errors.Errorf("pdfcpu: \"checkbox\" %s", s)
		}
		c.Regions.page = c.page
		c.Regions.parent = c
		return c.Regions.validate()
	}

	// bars
	for _, b := range c.Bars {
		b.pdf = pdf
		b.content = c
		if err := b.validate(); err != nil {
			return err
		}
	}

	// boxes
	for _, sb := range c.SimpleBoxPool {
		sb.pdf = pdf
		sb.content = c
		if err := sb.validate(); err != nil {
			return err
		}
	}
	for _, sb := range c.SimpleBoxes {
		sb.pdf = pdf
		sb.content = c
		if err := sb.validate(); err != nil {
			return err
		}
	}

	// text
	for _, tb := range c.TextBoxPool {
		tb.pdf = pdf
		tb.content = c
		if err := tb.validate(); err != nil {
			return err
		}
	}
	for _, tb := range c.TextBoxes {
		tb.pdf = pdf
		tb.content = c
		if err := tb.validate(); err != nil {
			return err
		}
	}

	// images
	for _, ib := range c.ImageBoxPool {
		ib.pdf = pdf
		ib.content = c
		if err := ib.validate(); err != nil {
			return err
		}
	}
	for _, ib := range c.ImageBoxes {
		ib.pdf = pdf
		ib.content = c
		if err := ib.validate(); err != nil {
			return err
		}
	}

	// tables
	for _, t := range c.TablePool {
		t.pdf = pdf
		t.content = c
		if err := t.validate(); err != nil {
			return err
		}
	}
	for _, t := range c.Tables {
		t.pdf = pdf
		t.content = c
		if err := t.validate(); err != nil {
			return err
		}
	}

	// text fields
	if len(c.TextFields) > 0 {
		// Reject form fields for existing Acroform.
		if pdf.HasForm {
			return errors.New("pdfcpu: appending form fields to existing form unsupported")
		}
		for _, tf := range c.TextFields {
			tf.pdf = pdf
			tf.content = c
			if err := tf.validate(); err != nil {
				return err
			}
		}
	}

	// check boxes
	if len(c.CheckBoxes) > 0 {
		// Reject form fields for existing Acroform.
		if pdf.HasForm {
			return errors.New("pdfcpu: appending form fields to existing form unsupported")
		}
		for _, cb := range c.CheckBoxes {
			cb.pdf = pdf
			cb.content = c
			if err := cb.validate(); err != nil {
				return err
			}
		}
	}

	// radio button groups
	if len(c.RadioButtonGroups) > 0 {
		// Reject form fields for existing Acroform.
		if pdf.HasForm {
			return errors.New("pdfcpu: appending form fields to existing form unsupported")
		}
		for _, rbg := range c.RadioButtonGroups {
			rbg.pdf = pdf
			rbg.content = c
			if err := rbg.validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Content) namedFont(id string) *FormFont {
	f := c.Fonts[id]
	if f != nil {
		return f
	}
	if c.parent != nil {
		return c.parent.namedFont(id)
	}
	return c.page.namedFont(id)
}

func (c *Content) namedMargin(id string) *Margin {
	m := c.Margins[id]
	if m != nil {
		return m
	}
	if c.parent != nil {
		return c.parent.namedMargin(id)
	}
	return c.page.namedMargin(id)
}

func (c *Content) margin() *Margin {
	return c.namedMargin("margin")
}

func (c *Content) namedBorder(id string) *Border {
	b := c.Borders[id]
	if b != nil {
		return b
	}
	if c.parent != nil {
		return c.parent.namedBorder(id)
	}
	return c.page.namedBorder(id)
}

func (c *Content) border() *Border {
	return c.namedBorder("border")
}

func (c *Content) namedPadding(id string) *Padding {
	p := c.Paddings[id]
	if p != nil {
		return p
	}
	if c.parent != nil {
		return c.parent.namedPadding(id)
	}
	return c.page.namedPadding(id)
}

func (c *Content) namedSimpleBox(id string) *SimpleBox {
	sb := c.SimpleBoxPool[id]
	if sb != nil {
		return sb
	}
	if c.parent != nil {
		return c.parent.namedSimpleBox(id)
	}
	return c.page.namedSimpleBox(id)
}

func (c *Content) namedImageBox(id string) *ImageBox {
	ib := c.ImageBoxPool[id]
	if ib != nil {
		return ib
	}
	if c.parent != nil {
		return c.parent.namedImageBox(id)
	}
	return c.page.namedImageBox(id)
}

func (c *Content) namedTextBox(id string) *TextBox {
	tb := c.TextBoxPool[id]
	if tb != nil {
		return tb
	}
	if c.parent != nil {
		return c.parent.namedTextBox(id)
	}
	return c.page.namedTextBox(id)
}

func (c *Content) namedTable(id string) *Table {
	t := c.TablePool[id]
	if t != nil {
		return t
	}
	if c.parent != nil {
		return c.parent.namedTable(id)
	}
	return c.page.namedTable(id)
}

func (c *Content) padding() *Padding {
	return c.namedPadding("padding")
}

func (c *Content) calcFont(ff map[string]*FormFont) {

	fff := map[string]*FormFont{}
	for id, f0 := range ff {
		fff[id] = f0
		f1 := c.Fonts[id]
		if f1 != nil {
			f1.mergeIn(f0)
			fff[id] = f1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcFont(fff)
			c.Regions.Right.calcFont(fff)
		} else {
			c.Regions.Top.calcFont(fff)
			c.Regions.Bottom.calcFont(fff)
		}
	}
}

func (c *Content) calcBorder(bb map[string]*Border) {

	bbb := map[string]*Border{}
	for id, b0 := range bb {
		bbb[id] = b0
		b1 := c.Borders[id]
		if b1 != nil {
			b1.mergeIn(b0)
			bbb[id] = b1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcBorder(bbb)
			c.Regions.Right.calcBorder(bb)
		} else {
			c.Regions.Top.calcBorder(bbb)
			c.Regions.Bottom.calcBorder(bbb)
		}
	}
}

func (c *Content) calcMargin(mm map[string]*Margin) {

	mmm := map[string]*Margin{}
	for id, m0 := range mm {
		mmm[id] = m0
		m1 := c.Margins[id]
		if m1 != nil {
			m1.mergeIn(m0)
			mmm[id] = m1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcMargin(mmm)
			c.Regions.Right.calcMargin(mmm)
		} else {
			c.Regions.Top.calcMargin(mmm)
			c.Regions.Bottom.calcMargin(mmm)
		}
	}
}

func (c *Content) calcPadding(pp map[string]*Padding) {

	ppp := map[string]*Padding{}
	for id, p0 := range pp {
		ppp[id] = p0
		p1 := c.Paddings[id]
		if p1 != nil {
			p1.mergeIn(p0)
			ppp[id] = p1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcPadding(ppp)
			c.Regions.Right.calcPadding(ppp)
		} else {
			c.Regions.Top.calcPadding(ppp)
			c.Regions.Bottom.calcPadding(ppp)
		}
	}
}

func (c *Content) calcSimpleBoxes(bb map[string]*SimpleBox) {

	bbb := map[string]*SimpleBox{}
	for id, sb0 := range bb {
		bbb[id] = sb0
		sb1 := c.SimpleBoxPool[id]
		if sb1 != nil {
			sb1.mergeIn(sb0)
			bbb[id] = sb1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcSimpleBoxes(bbb)
			c.Regions.Right.calcSimpleBoxes(bbb)
		} else {
			c.Regions.Top.calcSimpleBoxes(bbb)
			c.Regions.Bottom.calcSimpleBoxes(bbb)
		}
	}
}

func (c *Content) calcTextBoxes(bb map[string]*TextBox) {

	bbb := map[string]*TextBox{}
	for id, tb0 := range bb {
		bbb[id] = tb0
		tb1 := c.TextBoxPool[id]
		if tb1 != nil {
			tb1.mergeIn(tb0)
			bbb[id] = tb1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcTextBoxes(bbb)
			c.Regions.Right.calcTextBoxes(bbb)
		} else {
			c.Regions.Top.calcTextBoxes(bbb)
			c.Regions.Bottom.calcTextBoxes(bbb)
		}
	}
}

func (c *Content) calcImageBoxes(bb map[string]*ImageBox) {

	bbb := map[string]*ImageBox{}
	for id, ib0 := range bb {
		bbb[id] = ib0
		ib1 := c.ImageBoxPool[id]
		if ib1 != nil {
			ib1.mergeIn(ib0)
			bbb[id] = ib1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcImageBoxes(bbb)
			c.Regions.Right.calcImageBoxes(bbb)
		} else {
			c.Regions.Top.calcImageBoxes(bbb)
			c.Regions.Bottom.calcImageBoxes(bbb)
		}
	}
}

func (c *Content) calcTables(bb map[string]*Table) {

	bbb := map[string]*Table{}
	for id, t0 := range bb {
		bbb[id] = t0
		t1 := c.TablePool[id]
		if t1 != nil {
			t1.mergeIn(t0)
			bbb[id] = t1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcTables(bbb)
			c.Regions.Right.calcTables(bbb)
		} else {
			c.Regions.Top.calcTables(bbb)
			c.Regions.Bottom.calcTables(bbb)
		}
	}
}

func (c *Content) BorderRect() *pdfcpu.Rectangle {

	if c.borderRect == nil {

		mLeft, mRight, mTop, mBottom, borderWidth := 0., 0., 0., 0., 0.

		m := c.margin()
		if m != nil {
			mTop = m.Top
			mRight = m.Right
			mBottom = m.Bottom
			mLeft = m.Left
		}

		b := c.border()
		if b != nil && b.col != nil && b.Width >= 0 {
			borderWidth = float64(b.Width)
		}

		c.borderRect = pdfcpu.RectForWidthAndHeight(
			c.mediaBox.LL.X+mLeft+borderWidth/2,
			c.mediaBox.LL.Y+mBottom+borderWidth/2,
			c.mediaBox.Width()-mLeft-mRight-borderWidth,
			c.mediaBox.Height()-mTop-mBottom-borderWidth)

	}

	return c.borderRect
}

func (c *Content) Box() *pdfcpu.Rectangle {

	if c.box == nil {

		var mTop, mRight, mBottom, mLeft float64
		var pTop, pRight, pBottom, pLeft float64
		var borderWidth float64

		m := c.margin()
		if m != nil {
			if m.Width > 0 {
				mTop, mRight, mBottom, mLeft = m.Width, m.Width, m.Width, m.Width
			} else {
				mTop = m.Top
				mRight = m.Right
				mBottom = m.Bottom
				mLeft = m.Left
			}
		}

		b := c.border()
		if b != nil && b.col != nil && b.Width >= 0 {
			borderWidth = float64(b.Width)
		}

		p := c.padding()
		if p != nil {
			if p.Width > 0 {
				pTop, pRight, pBottom, pLeft = p.Width, p.Width, p.Width, p.Width
			} else {
				pTop = p.Top
				pRight = p.Right
				pBottom = p.Bottom
				pLeft = p.Left
			}
		}

		llx := c.mediaBox.LL.X + mLeft + borderWidth + pLeft
		lly := c.mediaBox.LL.Y + mBottom + borderWidth + pBottom
		w := c.mediaBox.Width() - mLeft - mRight - 2*borderWidth - pLeft - pRight
		h := c.mediaBox.Height() - mTop - mBottom - 2*borderWidth - pTop - pBottom
		c.box = pdfcpu.RectForWidthAndHeight(llx, lly, w, h)
	}

	return c.box
}

func (c *Content) render(p *pdfcpu.Page, pageNr int, fonts pdfcpu.FontMap, images pdfcpu.ImageMap, fields *pdfcpu.Array) error {

	if c.Regions != nil {
		c.Regions.mediaBox = c.mediaBox
		c.Regions.page = c.page
		return c.Regions.render(p, pageNr, fonts, images, fields)
	}

	pdf := c.page.pdf

	// Render background
	if c.bgCol != nil {
		pdfcpu.FillRectNoBorder(p.Buf, c.BorderRect(), *c.bgCol)
	}

	// Render border
	b := c.border()
	if b != nil && b.col != nil && b.Width >= 0 {
		pdfcpu.DrawRect(p.Buf, c.BorderRect(), float64(b.Width), b.col, &b.style)
	}

	// Render bars
	for _, b := range c.Bars {
		if b.Hide {
			continue
		}
		if err := b.render(p); err != nil {
			return err
		}
	}

	// Render boxes
	for _, sb := range c.SimpleBoxes {
		if sb.Hide {
			continue
		}
		if sb.Name != "" && sb.Name[0] == '$' {
			// Use named simplebox
			sbName := sb.Name[1:]
			sb0 := c.namedSimpleBox(sbName)
			if sb0 == nil {
				return errors.Errorf("pdfcpu: unknown named box %s", sbName)
			}
			sb.mergeIn(sb0)
		}
		if err := sb.render(p); err != nil {
			return err
		}
	}

	// Render text
	for _, tb := range c.TextBoxes {
		if tb.Hide {
			continue
		}
		if tb.Name != "" && tb.Name[0] == '$' {
			// Use named textbox
			tbName := tb.Name[1:]
			tb0 := c.namedTextBox(tbName)
			if tb0 == nil {
				return errors.Errorf("pdfcpu: unknown named text %s", tbName)
			}
			tb.mergeIn(tb0)
		}
		if err := tb.render(p, pageNr, fonts); err != nil {
			return err
		}
	}

	// Render images
	for _, ib := range c.ImageBoxes {
		if ib.Hide {
			continue
		}
		if ib.Name != "" && ib.Name[0] == '$' {
			// Use named imagebox
			ibName := ib.Name[1:]
			ib0 := c.namedImageBox(ibName)
			if ib0 == nil {
				return errors.Errorf("pdfcpu: unknown named text %s", ibName)
			}
			ib.mergeIn(ib0)
		}
		if err := ib.render(p, pageNr, images); err != nil {
			return err
		}
	}

	// Render tables
	for _, t := range c.Tables {
		if t.Hide {
			continue
		}
		if t.Name != "" && t.Name[0] == '$' {
			// Use named imagebox
			tName := t.Name[1:]
			t0 := c.namedTable(tName)
			if t0 == nil {
				return errors.Errorf("pdfcpu: unknown named text %s", tName)
			}
			t.mergeIn(t0)
		}
		if err := t.render(p, pageNr, fonts); err != nil {
			return err
		}
	}

	// Render textfields
	for _, tf := range c.TextFields {
		if tf.Hide {
			continue
		}
		if err := tf.render(p, pageNr, fonts); err != nil {
			return err
		}
	}

	// Render checkboxes
	for _, cb := range c.CheckBoxes {
		if cb.Hide {
			continue
		}
		if err := cb.render(p, pageNr, fonts); err != nil {
			return err
		}
	}

	// Render radio button groups
	for _, rbg := range c.RadioButtonGroups {
		if rbg.Hide {
			continue
		}
		if err := rbg.render(p, pageNr, fonts, fields); err != nil {
			return err
		}
	}

	// Render mediaBox & contentBox
	if pdf.ContentBox {
		pdfcpu.DrawRect(p.Buf, c.mediaBox, 0, &pdfcpu.Green, nil)
		pdfcpu.DrawRect(p.Buf, c.Box(), 0, &pdfcpu.Red, nil)
	}

	// Render guides
	if pdf.Guides {
		for _, g := range c.Guides {
			g.render(p.Buf, c.Box(), pdf)
		}
	}

	return nil
}

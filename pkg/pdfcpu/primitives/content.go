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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Content represents page content.
type Content struct {
	parent          *Content
	page            *PDFPage
	BackgroundColor string               `json:"bgCol"`
	bgCol           *color.SimpleColor   // page background color
	Fonts           map[string]*FormFont // named fonts
	Margins         map[string]*Margin   // named margins
	Borders         map[string]*Border   // named borders
	Paddings        map[string]*Padding  // named paddings
	Margin          *Margin              // content margin
	Border          *Border              // content border
	Padding         *Padding             // content padding
	Regions         *Regions
	mediaBox        *types.Rectangle
	borderRect      *types.Rectangle
	box             *types.Rectangle
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
	TextFields        []*TextField           `json:"textfield"`        // input text fields with optional label
	DateFields        []*DateField           `json:"datefield"`        // input date fields with optional label
	CheckBoxes        []*CheckBox            `json:"checkbox"`         // input checkboxes with optional label
	RadioButtonGroups []*RadioButtonGroup    `json:"radiobuttongroup"` // input radiobutton groups with optional label
	ComboBoxes        []*ComboBox            `json:"combobox"`
	ListBoxes         []*ListBox             `json:"listbox"`
	FieldGroups       []*FieldGroup          `json:"fieldgroup"` // rectangular container holding form elements
	FieldGroupPool    map[string]*FieldGroup `json:"fieldgroups"`
}

func (c *Content) validateBackgroundColor() error {
	if c.BackgroundColor != "" {
		sc, err := c.page.pdf.parseColor(c.BackgroundColor)
		if err != nil {
			return err
		}
		c.bgCol = sc
	}
	return nil
}

func (c *Content) validateBorders() error {
	pdf := c.page.pdf
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
	return nil
}

func (c *Content) validateMargins() error {
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
	return nil
}

func (c *Content) validatePaddings() error {
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
	return nil
}

func (c *Content) validatePrimitives(s string) error {
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
	return nil
}

func (c *Content) validateFormPrimitives(s string) error {
	if len(c.FieldGroupPool) > 0 {
		return errors.Errorf("pdfcpu: \"fieldgroups\" %s", s)
	}
	if len(c.FieldGroups) > 0 {
		return errors.Errorf("pdfcpu: \"fieldgroup\" %s", s)
	}
	if len(c.TextFields) > 0 {
		return errors.Errorf("pdfcpu: \"textfield\" %s", s)
	}
	if len(c.DateFields) > 0 {
		return errors.Errorf("pdfcpu: \"datefield\" %s", s)
	}
	if len(c.CheckBoxes) > 0 {
		return errors.Errorf("pdfcpu: \"checkbox\" %s", s)
	}
	if len(c.RadioButtonGroups) > 0 {
		return errors.Errorf("pdfcpu: \"radiobuttongroup\" %s", s)
	}
	if len(c.ComboBoxes) > 0 {
		return errors.Errorf("pdfcpu: \"combobox\" %s", s)
	}
	if len(c.ListBoxes) > 0 {
		return errors.Errorf("pdfcpu: \"listbox\" %s", s)
	}
	return nil
}

func (c *Content) validateRegions() error {
	s := "must be defined within region content"
	if err := c.validatePrimitives(s); err != nil {
		return err
	}
	if err := c.validateFormPrimitives(s); err != nil {
		return err
	}
	c.Regions.page = c.page
	c.Regions.parent = c
	return c.Regions.validate()
}

func (c *Content) validateBars() error {
	// bars
	for _, b := range c.Bars {
		b.pdf = c.page.pdf
		b.content = c
		if err := b.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) validateSimpleBoxPool() error {
	// boxes
	for _, sb := range c.SimpleBoxPool {
		sb.pdf = c.page.pdf
		sb.content = c
		if err := sb.validate(); err != nil {
			return err
		}
	}
	for _, sb := range c.SimpleBoxes {
		sb.pdf = c.page.pdf
		sb.content = c
		if err := sb.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) validateTextBoxPool() error {
	// text
	for _, tb := range c.TextBoxPool {
		tb.pdf = c.page.pdf
		tb.content = c
		if err := tb.validate(); err != nil {
			return err
		}
	}
	for _, tb := range c.TextBoxes {
		tb.pdf = c.page.pdf
		tb.content = c
		if err := tb.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) validateImageBoxPool() error {
	// images
	for _, ib := range c.ImageBoxPool {
		ib.pdf = c.page.pdf
		ib.content = c
		if err := ib.validate(); err != nil {
			return err
		}
	}
	for _, ib := range c.ImageBoxes {
		ib.pdf = c.page.pdf
		ib.content = c
		if err := ib.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) validateTablePool() error {
	// tables
	for _, t := range c.TablePool {
		t.pdf = c.page.pdf
		t.content = c
		if err := t.validate(); err != nil {
			return err
		}
	}
	for _, t := range c.Tables {
		t.pdf = c.page.pdf
		t.content = c
		if err := t.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) validateFieldGroupPool() error {
	// textfield groups
	for _, fg := range c.FieldGroupPool {
		fg.pdf = c.page.pdf
		fg.content = c
		if err := fg.validate(); err != nil {
			return err
		}
	}
	return c.validateFieldGroups()
}

func (c *Content) validatePools() error {
	if err := c.validateSimpleBoxPool(); err != nil {
		return err
	}
	if err := c.validateTextBoxPool(); err != nil {
		return err
	}
	if err := c.validateImageBoxPool(); err != nil {
		return err
	}
	if err := c.validateTablePool(); err != nil {
		return err
	}
	return c.validateFieldGroupPool()
}

func (c *Content) validateTextFields() error {
	pdf := c.page.pdf
	if len(c.TextFields) > 0 {
		for _, tf := range c.TextFields {
			tf.pdf = pdf
			tf.content = c
			if err := tf.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) validateDateFields() error {
	pdf := c.page.pdf
	if len(c.DateFields) > 0 {
		for _, df := range c.DateFields {
			df.pdf = pdf
			df.content = c
			if err := df.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) validateFieldGroups() error {
	pdf := c.page.pdf
	if len(c.FieldGroups) > 0 {
		for _, fg := range c.FieldGroups {
			fg.pdf = pdf
			fg.content = c
			if err := fg.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) validateCheckBoxes() error {
	pdf := c.page.pdf
	if len(c.CheckBoxes) > 0 {
		for _, cb := range c.CheckBoxes {
			cb.pdf = pdf
			cb.content = c
			if err := cb.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) validateRadioButtonGroups() error {
	pdf := c.page.pdf
	if len(c.RadioButtonGroups) > 0 {
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

func (c *Content) validateComboBoxes() error {
	pdf := c.page.pdf
	if len(c.ComboBoxes) > 0 {
		for _, cb := range c.ComboBoxes {
			cb.pdf = pdf
			cb.content = c
			if err := cb.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) validateListBoxes() error {
	pdf := c.page.pdf
	if len(c.ListBoxes) > 0 {
		for _, lb := range c.ListBoxes {
			lb.pdf = pdf
			lb.content = c
			if err := lb.validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) validate() error {

	if err := c.validateBackgroundColor(); err != nil {
		return err
	}

	for _, g := range c.Guides {
		g.validate()
	}

	if err := c.validateBorders(); err != nil {
		return err
	}

	if err := c.validateMargins(); err != nil {
		return err
	}

	if err := c.validatePaddings(); err != nil {
		return err
	}

	if c.Regions != nil {
		return c.validateRegions()
	}

	if err := c.validateBars(); err != nil {
		return err
	}

	if err := c.validatePools(); err != nil {
		return err
	}

	if err := c.validateTextFields(); err != nil {
		return err
	}

	if err := c.validateDateFields(); err != nil {
		return err
	}

	if err := c.validateCheckBoxes(); err != nil {
		return err
	}

	if err := c.validateRadioButtonGroups(); err != nil {
		return err
	}

	if err := c.validateComboBoxes(); err != nil {
		return err
	}

	return c.validateListBoxes()
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

func (c *Content) padding() *Padding {
	return c.namedPadding("padding")
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

func (c *Content) namedFieldGroup(id string) *FieldGroup {
	fg := c.FieldGroupPool[id]
	if fg != nil {
		return fg
	}
	if c.parent != nil {
		return c.parent.namedFieldGroup(id)
	}
	return c.page.namedFieldGroup(id)
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

func (c *Content) mergeIn(fName string, f *FormFont) error {
	f0 := c.namedFont(fName)
	if f0 == nil {
		return errors.Errorf("pdfcpu: missing named font \"input\"")
	}
	f.Name = f0.Name
	if f.Size == 0 {
		f.Size = f0.Size
	}
	if f.col == nil {
		f.col = f0.col
	}
	if f.Lang == "" {
		f.Lang = f0.Lang
	}
	if f.Script == "" {
		f.Script = f0.Script
	}
	return nil
}

func (c *Content) calcInputFont(f *FormFont) (*FormFont, error) {

	if f != nil {
		if f.Name == "" {
			// Inherited named font "input".
			if err := c.mergeIn("input", f); err != nil {
				return nil, err
			}
		}
		if f.Name != "" && f.Name[0] == '$' {
			// Inherit some other named font.
			fName := f.Name[1:]
			if err := c.mergeIn(fName, f); err != nil {
				return nil, err
			}
		}
	} else {
		// Use inherited named font "input".
		f = c.namedFont("input")
		if f == nil {
			return nil, errors.Errorf("pdfcpu: missing named font \"input\"")
		}
	}

	if f.col == nil {
		f.col = &color.Black
	}

	return f, nil
}

func (c *Content) calcLabelFont(f *FormFont) (*FormFont, error) {

	if f != nil {
		var f0 *FormFont
		if f.Name == "" {
			// Use inherited named font "label".
			f0 = c.namedFont("label")
			if f0 == nil {
				return nil, errors.Errorf("pdfcpu: missing named font \"label\"")
			}
			f.Name = f0.Name
			if f.Size == 0 {
				f.Size = f0.Size
			}
			if f.col == nil {
				f.col = f0.col
			}
			if f.Script == "" {
				f.Script = f0.Script
			}
		}
		if f.Name != "" && f.Name[0] == '$' {
			// use named font
			fName := f.Name[1:]
			f0 := c.namedFont(fName)
			if f0 == nil {
				return nil, errors.Errorf("pdfcpu: unknown font name %s", fName)
			}
			f.Name = f0.Name
			if f.Size == 0 {
				f.Size = f0.Size
			}
			if f.col == nil {
				f.col = f0.col
			}
			if f.Script == "" {
				f.Script = f0.Script
			}
		}
	} else {
		// Use inherited named font "label".
		f = c.namedFont("label")
		if f == nil {
			return nil, errors.Errorf("pdfcpu: missing named font \"label\"")
		}
	}

	if f.col == nil {
		f.col = &color.Black
	}

	return f, nil
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

func (c *Content) calcFieldGroups(bb map[string]*FieldGroup) {

	bbb := map[string]*FieldGroup{}
	for id, fg0 := range bb {
		bbb[id] = fg0
		fg1 := c.FieldGroupPool[id]
		if fg1 != nil {
			fg1.mergeIn(fg0)
			bbb[id] = fg1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcFieldGroups(bbb)
			c.Regions.Right.calcFieldGroups(bbb)
		} else {
			c.Regions.Top.calcFieldGroups(bbb)
			c.Regions.Bottom.calcFieldGroups(bbb)
		}
	}
}

// BorderRect returns the border rect for c.
func (c *Content) BorderRect() *types.Rectangle {

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

		c.borderRect = types.RectForWidthAndHeight(
			c.mediaBox.LL.X+mLeft+borderWidth/2,
			c.mediaBox.LL.Y+mBottom+borderWidth/2,
			c.mediaBox.Width()-mLeft-mRight-borderWidth,
			c.mediaBox.Height()-mTop-mBottom-borderWidth)

	}

	return c.borderRect
}

func (c *Content) Box() *types.Rectangle {

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
		c.box = types.RectForWidthAndHeight(llx, lly, w, h)
	}

	return c.box
}

func (c *Content) calcPosition(x, y, dx, dy, mTop, mRight, mBottom, mLeft float64) (float64, float64) {

	cBox := c.Box()

	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	pdf := c.page.pdf
	x, y = types.NormalizeCoord(x, y, cBox, pdf.origin, false)

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
	x += dx
	y += dy

	return x, y

}

func (c *Content) renderBars(p *model.Page) error {
	for _, b := range c.Bars {
		if b.Hide {
			continue
		}
		if err := b.render(p); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderSimpleBoxes(p *model.Page) error {
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
	return nil
}

func (c *Content) renderTextBoxes(p *model.Page, pageNr int, fonts model.FontMap) error {
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
	return nil
}

func (c *Content) renderImageBoxes(p *model.Page, pageNr int, images model.ImageMap) error {
	for _, ib := range c.ImageBoxes {
		if ib.Hide {
			continue
		}
		if ib.Name != "" && ib.Name[0] == '$' {
			// Use named imagebox
			ibName := ib.Name[1:]
			ib0 := c.namedImageBox(ibName)
			if ib0 == nil {
				return errors.Errorf("pdfcpu: unknown named image %s", ibName)
			}
			ib.mergeIn(ib0)
		}
		if err := ib.render(p, pageNr, images); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderTables(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, t := range c.Tables {
		if t.Hide {
			continue
		}
		if t.Name != "" && t.Name[0] == '$' {
			// Use named table
			tName := t.Name[1:]
			t0 := c.namedTable(tName)
			if t0 == nil {
				return errors.Errorf("pdfcpu: unknown named table %s", tName)
			}
			t.mergeIn(t0)
		}
		if err := t.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderTextFields(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, tf := range c.TextFields {
		if tf.Hide {
			continue
		}
		if err := tf.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderDateFields(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, df := range c.DateFields {
		if df.Hide {
			continue
		}
		if err := df.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderCheckBoxes(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, cb := range c.CheckBoxes {
		if cb.Hide {
			continue
		}
		if err := cb.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderRadioButtonGroups(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, rbg := range c.RadioButtonGroups {
		if rbg.Hide {
			continue
		}
		if err := rbg.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderComboBoxes(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, cb := range c.ComboBoxes {
		if cb.Hide {
			continue
		}
		if err := cb.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderListBoxes(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, lb := range c.ListBoxes {
		if lb.Hide {
			continue
		}
		if err := lb.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderFieldGroups(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, fg := range c.FieldGroups {
		if fg.Hide {
			continue
		}
		if fg.Name != "" && fg.Name[0] == '$' {
			// Use named field group
			fgName := fg.Name[1:]
			fg0 := c.namedFieldGroup(fgName)
			if fg0 == nil {
				return errors.Errorf("pdfcpu: unknown named field group %s", fgName)
			}
			fg.mergeIn(fg0)
		}
		if err := fg.render(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (c *Content) renderBoxesAndGuides(p *model.Page) {
	pdf := c.page.pdf

	// Render mediaBox & contentBox
	if pdf.ContentBox {
		draw.DrawRect(p.Buf, c.mediaBox, 0, &color.Green, nil)
		draw.DrawRect(p.Buf, c.Box(), 0, &color.Red, nil)
	}

	// Render guides
	if pdf.Guides {
		for _, g := range c.Guides {
			g.render(p.Buf, c.Box(), pdf)
		}
	}
}

func (c *Content) renderPrimitives(p *model.Page, pageNr int, fonts model.FontMap, images model.ImageMap) error {
	if err := c.renderBars(p); err != nil {
		return err
	}

	if err := c.renderSimpleBoxes(p); err != nil {
		return err
	}

	if err := c.renderTextBoxes(p, pageNr, fonts); err != nil {
		return err
	}

	if err := c.renderImageBoxes(p, pageNr, images); err != nil {
		return err
	}

	return c.renderTables(p, pageNr, fonts)
}

func (c *Content) renderFormPrimitives(p *model.Page, pageNr int, fonts model.FontMap) error {
	if err := c.renderTextFields(p, pageNr, fonts); err != nil {
		return err
	}

	if err := c.renderDateFields(p, pageNr, fonts); err != nil {
		return err
	}

	if err := c.renderCheckBoxes(p, pageNr, fonts); err != nil {
		return err
	}

	if err := c.renderRadioButtonGroups(p, pageNr, fonts); err != nil {
		return err
	}

	if err := c.renderComboBoxes(p, pageNr, fonts); err != nil {
		return err
	}

	if err := c.renderListBoxes(p, pageNr, fonts); err != nil {
		return err
	}

	return c.renderFieldGroups(p, pageNr, fonts)
}

func (c *Content) render(p *model.Page, pageNr int, fonts model.FontMap, images model.ImageMap) error {

	if c.Regions != nil {
		c.Regions.mediaBox = c.mediaBox
		c.Regions.page = c.page
		return c.Regions.render(p, pageNr, fonts, images)
	}

	// Render background
	if c.bgCol != nil {
		draw.FillRectNoBorder(p.Buf, c.BorderRect(), *c.bgCol)
	}

	// Render border
	b := c.border()
	if b != nil && b.col != nil && b.Width >= 0 {
		draw.DrawRect(p.Buf, c.BorderRect(), float64(b.Width), b.col, &b.style)
	}

	if err := c.renderPrimitives(p, pageNr, fonts, images); err != nil {
		return err
	}

	if err := c.renderFormPrimitives(p, pageNr, fonts); err != nil {
		return err
	}

	c.renderBoxesAndGuides(p)

	return nil
}

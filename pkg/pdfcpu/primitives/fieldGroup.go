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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// FieldGroup is a container for fields.
type FieldGroup struct {
	pdf               *PDF
	content           *Content
	Name              string
	Value             string
	Border            *Border
	Padding           *Padding
	BackgroundColor   string `json:"bgCol"`
	bgCol             *color.SimpleColor
	TextFields        []*TextField        `json:"textfield"`        // text fields with optional label
	DateFields        []*DateField        `json:"datefield"`        // date fields with optional label
	CheckBoxes        []*CheckBox         `json:"checkbox"`         // checkboxes with optional label
	RadioButtonGroups []*RadioButtonGroup `json:"radiobuttongroup"` // radiobutton groups with optional label
	ComboBoxes        []*ComboBox         `json:"combobox"`         // comboboxes with optional label
	ListBoxes         []*ListBox          `json:"listbox"`          // listboxes with optional label
	Hide              bool
}

func (fg *FieldGroup) validateBorder() error {
	if fg.Border != nil {
		fg.Border.pdf = fg.pdf
		if err := fg.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) validatePadding() error {
	if fg.Padding != nil {
		if err := fg.Padding.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) validateBackgroundColor() error {
	if fg.BackgroundColor != "" {
		sc, err := fg.pdf.parseColor(fg.BackgroundColor)
		if err != nil {
			return err
		}
		fg.bgCol = sc
	}
	return nil
}

func (fg *FieldGroup) validateBorderPaddingBgCol() error {
	if err := fg.validateBorder(); err != nil {
		return err
	}

	if err := fg.validatePadding(); err != nil {
		return err
	}

	return fg.validateBackgroundColor()
}

func (fg *FieldGroup) validate() error {

	if err := fg.validateBorderPaddingBgCol(); err != nil {
		return err
	}

	for _, tf := range fg.TextFields {
		tf.pdf = fg.pdf
		tf.content = fg.content
		if err := tf.validate(); err != nil {
			return err
		}
	}

	for _, df := range fg.DateFields {
		df.pdf = fg.pdf
		df.content = fg.content
		if err := df.validate(); err != nil {
			return err
		}
	}

	for _, cb := range fg.CheckBoxes {
		cb.pdf = fg.pdf
		cb.content = fg.content
		if err := cb.validate(); err != nil {
			return err
		}
	}

	for _, rbg := range fg.RadioButtonGroups {
		rbg.pdf = fg.pdf
		rbg.content = fg.content
		if err := rbg.validate(); err != nil {
			return err
		}
	}

	for _, cb := range fg.ComboBoxes {
		cb.pdf = fg.pdf
		cb.content = fg.content
		if err := cb.validate(); err != nil {
			return err
		}
	}

	for _, lb := range fg.ListBoxes {
		lb.pdf = fg.pdf
		lb.content = fg.content
		if err := lb.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (fg *FieldGroup) mergeIn(fg0 *FieldGroup) {
	if fg.Border == nil {
		fg.Border = fg0.Border
	}

	if fg.Padding == nil {
		fg.Padding = fg0.Padding
	}

	if fg.bgCol == nil {
		fg.bgCol = fg0.bgCol
	}

	if !fg.Hide {
		fg.Hide = fg0.Hide
	}
}

func (fg *FieldGroup) calcBBoxFromTextFields(bbox **types.Rectangle, p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, tf := range fg.TextFields {
		if err := tf.prepForRender(p, pageNr, fonts); err != nil {
			return err
		}
		*bbox = model.CalcBoundingBoxForRects(*bbox, tf.bbox())
	}
	return nil
}

func (fg *FieldGroup) calcBBoxFromDateFields(bbox **types.Rectangle, p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, df := range fg.DateFields {
		if err := df.prepForRender(p, pageNr, fonts); err != nil {
			return err
		}
		*bbox = model.CalcBoundingBoxForRects(*bbox, df.bbox())
	}
	return nil
}

func (fg *FieldGroup) calcBBoxFromCheckBoxes(bbox **types.Rectangle, p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, cb := range fg.CheckBoxes {
		if err := cb.prepForRender(p, pageNr, fonts); err != nil {
			return err
		}
		*bbox = model.CalcBoundingBoxForRects(*bbox, cb.bbox())
	}
	return nil
}

func (fg *FieldGroup) calcBBoxFromRadioButtonGroups(bbox **types.Rectangle, p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, rbg := range fg.RadioButtonGroups {
		if err := rbg.prepForRender(p, pageNr, fonts); err != nil {
			return err
		}
		*bbox = model.CalcBoundingBoxForRects(*bbox, rbg.bbox())
	}
	return nil
}

func (fg *FieldGroup) calcBBoxFromComboBoxes(bbox **types.Rectangle, p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, cb := range fg.ComboBoxes {
		if err := cb.prepForRender(p, pageNr, fonts); err != nil {
			return err
		}
		*bbox = model.CalcBoundingBoxForRects(*bbox, cb.bbox())
	}
	return nil
}

func (fg *FieldGroup) calcBBoxFromListBoxes(bbox **types.Rectangle, p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, lb := range fg.ListBoxes {
		if err := lb.prepForRender(p, pageNr, fonts); err != nil {
			return err
		}
		*bbox = model.CalcBoundingBoxForRects(*bbox, lb.bbox())
	}
	return nil
}

func (fg *FieldGroup) calcBBox(p *model.Page, pageNr int, fonts model.FontMap) (*types.Rectangle, error) {
	var bbox *types.Rectangle

	if err := fg.calcBBoxFromTextFields(&bbox, p, pageNr, fonts); err != nil {
		return nil, err
	}

	if err := fg.calcBBoxFromDateFields(&bbox, p, pageNr, fonts); err != nil {
		return nil, err
	}

	if err := fg.calcBBoxFromCheckBoxes(&bbox, p, pageNr, fonts); err != nil {
		return nil, err
	}

	if err := fg.calcBBoxFromRadioButtonGroups(&bbox, p, pageNr, fonts); err != nil {
		return nil, err
	}

	if err := fg.calcBBoxFromComboBoxes(&bbox, p, pageNr, fonts); err != nil {
		return nil, err
	}

	if err := fg.calcBBoxFromListBoxes(&bbox, p, pageNr, fonts); err != nil {
		return nil, err
	}

	return bbox, nil
}

func (fg *FieldGroup) renderBBox(bbox *types.Rectangle, p *model.Page) error {
	r := fg.content.Box()
	var bw float64
	if fg.Border != nil {
		bw = float64(fg.Border.Width)
	}

	var pTop, pRight, pBottom, pLeft float64
	if fg.Padding != nil {
		p := fg.Padding
		pTop, pRight, pBottom, pLeft = p.Top, p.Right, p.Bottom, p.Left
	}

	x := bbox.LL.X - r.LL.X - bw - pLeft
	w := bbox.Width() + 2*bw + pLeft + pRight
	if x < 0 {
		w += x
		x = 0
	}

	y := bbox.LL.Y - r.LL.Y - bw - pBottom
	h := bbox.Height() + 2*bw + pBottom + pTop
	if y < 0 {
		h += y
		y = 0
	}

	sb := SimpleBox{
		x:       x,
		y:       y,
		Width:   w,
		Height:  h,
		fillCol: fg.bgCol,
		Border:  fg.Border,
	}

	sb.pdf = fg.pdf
	sb.content = fg.content

	return sb.render(p)
}

func (fg *FieldGroup) renderTextFields(p *model.Page, fonts model.FontMap) error {
	for _, tf := range fg.TextFields {
		if tf.Hide {
			continue
		}
		if err := tf.doRender(p, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) renderDateFields(p *model.Page, fonts model.FontMap) error {
	for _, df := range fg.DateFields {
		if df.Hide {
			continue
		}
		if err := df.doRender(p, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) renderCheckBoxes(p *model.Page, fonts model.FontMap) error {
	for _, cb := range fg.CheckBoxes {
		if cb.Hide {
			continue
		}
		if err := cb.doRender(p, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) renderRadioButtonGroups(p *model.Page, pageNr int, fonts model.FontMap) error {
	for _, rbg := range fg.RadioButtonGroups {
		if rbg.Hide {
			continue
		}
		if err := rbg.doRender(p, pageNr, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) renderComboBoxes(p *model.Page, fonts model.FontMap) error {
	for _, cb := range fg.ComboBoxes {
		if cb.Hide {
			continue
		}
		if err := cb.doRender(p, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) renderListBoxes(p *model.Page, fonts model.FontMap) error {
	for _, cb := range fg.ListBoxes {
		if cb.Hide {
			continue
		}
		if err := cb.doRender(p, fonts); err != nil {
			return err
		}
	}
	return nil
}

func (fg *FieldGroup) renderFields(p *model.Page, pageNr int, fonts model.FontMap) error {
	if err := fg.renderTextFields(p, fonts); err != nil {
		return err
	}
	if err := fg.renderDateFields(p, fonts); err != nil {
		return err
	}
	if err := fg.renderCheckBoxes(p, fonts); err != nil {
		return err
	}
	if err := fg.renderRadioButtonGroups(p, pageNr, fonts); err != nil {
		return err
	}
	if err := fg.renderComboBoxes(p, fonts); err != nil {
		return err
	}
	return fg.renderListBoxes(p, fonts)
}

func (fg *FieldGroup) render(p *model.Page, pageNr int, fonts model.FontMap) error {
	bbox, err := fg.calcBBox(p, pageNr, fonts)
	if err != nil {
		return err
	}

	// Render simpleBox containing all fields of this group.
	if err := fg.renderBBox(bbox, p); err != nil {
		return err
	}

	if err := fg.renderFields(p, pageNr, fonts); err != nil {
		return err
	}

	if fg.pdf.Debug {
		fg.pdf.highlightPos(p.Buf, bbox.LL.X, bbox.LL.Y, fg.content.Box())
	}

	return nil
}

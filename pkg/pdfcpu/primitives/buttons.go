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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type Buttons struct {
	pdf         *PDF
	Values      []string
	Label       *TextFieldLabel
	Gap         int // horizontal space between radio button and its value
	widths      []float64
	maxWidth    float64
	boundingBox *types.Rectangle
}

func (b *Buttons) Rtl() bool {
	if b.Label == nil {
		return false
	}
	return b.Label.RTL
}

func (b *Buttons) validate(defValue, value string) error {

	if len(b.Values) < 2 {
		return errors.New("pdfcpu: radiobuttongroups.buttons missing values")
	}

	if defValue != "" {
		if !types.MemberOf(defValue, b.Values) {
			return errors.Errorf("pdfcpu: radiobuttongroups invalid default: %s", defValue)
		}
	}

	if value != "" {
		if !types.MemberOf(value, b.Values) {
			return errors.Errorf("pdfcpu: radiobuttongroups invalid value: %s", value)
		}
	}

	if b.Label == nil {
		return errors.New("pdfcpu: radiobuttongroups.buttons: missing label")
	}

	b.Label.pdf = b.pdf
	if err := b.Label.validate(); err != nil {
		return err
	}

	pos := b.Label.relPos
	if pos == types.RelPosTop || pos == types.RelPosBottom {
		return errors.New("pdfcpu: radiobuttongroups.buttons.label: pos must be left or right")
	}

	b.Label.HorAlign = types.AlignLeft
	if pos == types.RelPosLeft {
		// A radio button label on the left side of a radio button is right aligned.
		b.Label.HorAlign = types.AlignRight
	}

	if b.Gap <= 0 {
		b.Gap = 3
	}

	return nil
}

func (b *Buttons) calcLeftAlignedHorLabelWidths(td model.TextDescriptor) {
	var maxw float64
	for i := 0; i < len(b.Values); i++ {
		td.Text = b.Values[i]
		bb := model.WriteMultiLine(b.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
		// Leave last label width as is.
		if i == len(b.Values)-1 {
			b.maxWidth = maxw
			for i := range b.widths {
				b.widths[i] = maxw
			}
			if bb.Width() > maxw {
				b.widths[i] = bb.Width()
			}
			return
		}
		if bb.Width() > maxw {
			maxw = bb.Width()
		}
	}
}

func (b *Buttons) calcRightAlignedHorLabelWidths(td model.TextDescriptor) {
	var maxw float64
	for i := 0; i < len(b.Values); i++ {
		td.Text = b.Values[i]
		bb := model.WriteMultiLine(b.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
		// Leave first label width as is.
		if i == 0 {
			b.widths[0] = bb.Width()
			continue
		}
		if bb.Width() > maxw {
			maxw = bb.Width()
		}
	}
	b.maxWidth = maxw
	if b.widths[0] < maxw {
		b.widths[0] = maxw
	}
	for i := 1; i < len(b.Values); i++ {
		b.widths[i] = maxw
	}
}

func (b *Buttons) calcHorLabelWidths(td model.TextDescriptor) {
	if b.Label.HorAlign == types.AlignLeft {
		b.calcLeftAlignedHorLabelWidths(td)
		return
	}
	b.calcRightAlignedHorLabelWidths(td)
}

func (b *Buttons) calcVerLabelWidths(td model.TextDescriptor) {
	var maxw float64
	for _, v := range b.Values {
		td.Text = v
		bb := model.WriteMultiLine(b.pdf.XRefTable, new(bytes.Buffer), types.RectForFormat("A4"), nil, td)
		if bb.Width() > maxw {
			maxw = bb.Width()
		}
	}
	for i := range b.widths {
		b.widths[i] = maxw
	}
	b.maxWidth = maxw
}

func (b *Buttons) calcLabelWidths(hor bool) {
	b.widths = make([]float64, len(b.Values))
	td := model.TextDescriptor{
		FontName: b.Label.Font.Name,
		FontSize: b.Label.Font.Size,
		RTL:      b.Label.RTL,
		Scale:    1.,
		ScaleAbs: true,
	}
	if hor {
		b.calcHorLabelWidths(td)
		return
	}
	b.calcVerLabelWidths(td)
}

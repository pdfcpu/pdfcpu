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

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type Buttons struct {
	pdf    *PDF
	Values []string
	Label  *TextFieldLabel
}

func (b *Buttons) validate() error {

	if len(b.Values) < 2 {
		return errors.New("pdfcpu: radiobuttongroups.buttons missing values")
	}

	if b.Label == nil {
		return errors.New("pdfcpu: radiobuttongroups.buttons: missing label")
	}

	b.Label.pdf = b.pdf
	if err := b.Label.validate(); err != nil {
		return err
	}

	pos := b.Label.relPos
	if pos == pdfcpu.RelPosTop || pos == pdfcpu.RelPosBottom {
		return errors.New("pdfcpu: radiobuttongroups.buttons.label: pos must be left or right")
	}

	b.Label.horAlign = pdfcpu.AlignLeft
	if pos == pdfcpu.RelPosLeft {
		// A radio button label on the left side of a radio button is right aligned.
		b.Label.horAlign = pdfcpu.AlignRight
	}

	return nil
}

func (b *Buttons) maxLabelWidth(hor bool) (float64, float64) {
	maxw, lastw := 0.0, 0.0
	fontName := b.Label.Font.Name
	fontSize := b.Label.Font.Size
	for i, v := range b.Values {
		td := pdfcpu.TextDescriptor{
			Text:     v,
			FontName: fontName,
			FontSize: fontSize,
			Scale:    1.,
			ScaleAbs: true,
		}
		bb := pdfcpu.WriteMultiLine(new(bytes.Buffer), pdfcpu.RectForFormat("A4"), nil, td)
		if hor {
			if b.Label.horAlign == pdfcpu.AlignLeft {
				// Leave last label width as is.
				if i == len(b.Values)-1 {
					lastw = maxw
					if bb.Width() > maxw {
						lastw = bb.Width()
					}
					continue
				}
			}
			if b.Label.horAlign == pdfcpu.AlignRight {
				// Leave first label width as is.
				if i == 0 {
					lastw = bb.Width()
					continue
				}
			}
		}
		if bb.Width() > maxw {
			maxw = bb.Width()
		}
	}
	if b.Label.horAlign == pdfcpu.AlignRight {
		// This is actually the width of the first (left most) label in this case.
		if lastw < maxw {
			lastw = maxw
		}
	}
	return maxw, lastw
}

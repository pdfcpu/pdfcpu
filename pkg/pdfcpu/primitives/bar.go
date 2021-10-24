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

type Bar struct {
	pdf     *PDF
	content *Content
	X, Y    float64 // either or determines orientation.
	Width   int
	Color   string `json:"col"`
	col     *pdfcpu.SimpleColor
	Style   string
	style   pdfcpu.LineJoinStyle
	Hide    bool
}

func (b *Bar) validate() error {

	if b.X != 0 && b.Y != 0 || b.X < 0 || b.Y < 0 {
		return errors.Errorf("pdfcpu: bar: supply positive values for either x (vertical bar) or y (horizontal)")
	}

	if b.Color != "" {
		sc, err := b.pdf.parseColor(b.Color)
		if err != nil {
			return err
		}
		b.col = sc
	}

	b.style = pdfcpu.LJMiter
	if b.Style != "" {
		switch b.Style {
		case "miter":
			b.style = pdfcpu.LJMiter
		case "round":
			b.style = pdfcpu.LJRound
		case "bevel":
			b.style = pdfcpu.LJBevel
		default:
			return errors.Errorf("pdfcpu: invalid bar style: %s (should be \"miter\", \"round\" or \"bevel\")", b.Style)
		}
	}

	return nil
}

func (b *Bar) render(p *pdfcpu.Page) error {

	if b.col == nil {
		return nil
	}

	cBox := b.content.Box()

	var px, py, qx, qy float64

	if b.X > 0 {
		// Vertical bar
		px, py = b.X, 0
		qx, qy = px, cBox.Height()
	} else {
		// Horizontal bar
		px, py = 0, b.Y
		qx, qy = cBox.Width(), py
	}

	px, py = pdfcpu.NormalizeCoord(px, py, cBox, b.pdf.origin, true)
	qx, qy = pdfcpu.NormalizeCoord(qx, qy, cBox, b.pdf.origin, true)

	pdfcpu.DrawLine(p.Buf, px, py, qx, qy, float64(b.Width), b.col, &b.style)

	return nil
}

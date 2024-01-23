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

// Bar represents a horizontal or vertical bar used by content.
type Bar struct {
	pdf     *PDF
	content *Content
	X, Y    float64 // either or determines orientation.
	Width   int
	Color   string `json:"col"`
	col     *color.SimpleColor
	Style   string
	style   types.LineJoinStyle
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

	b.style = types.LJMiter
	if b.Style != "" {
		switch b.Style {
		case "miter":
			b.style = types.LJMiter
		case "round":
			b.style = types.LJRound
		case "bevel":
			b.style = types.LJBevel
		default:
			return errors.Errorf("pdfcpu: invalid bar style: %s (should be \"miter\", \"round\" or \"bevel\")", b.Style)
		}
	}

	return nil
}

func (b *Bar) render(p *model.Page) error {

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

	px, py = types.NormalizeCoord(px, py, cBox, b.pdf.origin, true)
	qx, qy = types.NormalizeCoord(qx, qy, cBox, b.pdf.origin, true)

	draw.DrawLine(p.Buf, px, py, qx, qy, float64(b.Width), b.col, &b.style)

	return nil
}

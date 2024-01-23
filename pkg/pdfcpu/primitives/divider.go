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

// Divider is a positioned separator between two regions from p to q.
type Divider struct {
	pdf   *PDF
	Pos   float64     `json:"at"` // fraction 0..1
	p, q  types.Point // Endpoints
	Width int         // 1..10
	Color string      `json:"col"`
	col   *color.SimpleColor
}

func (d *Divider) validate() error {
	if d.Pos <= 0 || d.Pos >= 1 {
		return errors.Errorf("pdfcpu: div at(%.1f) needs to be between 0 and 1", d.Pos)
	}
	if d.Width < 0 || d.Width > 10 {
		return errors.Errorf("pdfcpu: div width(%d) needs to be between 0 and 10", d.Width)
	}
	if d.Color != "" {
		sc, err := d.pdf.parseColor(d.Color)
		if err != nil {
			return err
		}
		d.col = sc
	}
	return nil
}

func (d *Divider) render(p *model.Page) error {

	if d.col == nil {
		return nil
	}

	draw.DrawLine(p.Buf, d.p.X, d.p.Y, d.q.X, d.q.Y, float64(d.Width), d.col, nil)

	return nil
}

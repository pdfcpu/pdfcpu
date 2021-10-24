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

type Divider struct {
	pdf   *PDF
	At    float64
	p, q  pdfcpu.Point
	Width int
	Color string `json:"col"`
	col   *pdfcpu.SimpleColor
}

func (d *Divider) validate() error {
	if d.At <= 0 || d.At >= 1 {
		return errors.Errorf("pdfcpu: div at(%.1f) needs to be between 0 and 1", d.At)
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

func (d *Divider) render(p *pdfcpu.Page) error {

	if d.col == nil {
		return nil
	}

	pdfcpu.DrawLine(p.Buf, d.p.X, d.p.Y, d.q.X, d.q.Y, float64(d.Width), d.col, nil)

	return nil
}

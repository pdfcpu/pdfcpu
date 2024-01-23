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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type Border struct {
	pdf   *PDF
	Name  string
	Width int
	Color string `json:"col"`
	col   *color.SimpleColor
	Style string
	style types.LineJoinStyle
}

func (b *Border) validate() error {

	if b.Name == "$" {
		return errors.New("pdfcpu: invalid border reference $")
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
			return errors.Errorf("pdfcpu: invalid border style: %s (should be \"miter\", \"round\" or \"bevel\")", b.Style)
		}
	}

	return nil
}

func (b *Border) mergeIn(b0 *Border) {
	if b.Width == 0 {
		b.Width = b0.Width
	}
	if b.col == nil {
		b.col = b0.col
	}
	if b.style == types.LJMiter {
		b.style = b0.style
	}
}

// func (b *Border) SetCol(c color.SimpleColor) {
// 	b.col = &c
// }

func (b Border) calc() (boWidth float64, boCol *color.SimpleColor) {
	if b.col == nil {
		return 0, &color.Black
	}
	return float64(b.Width), b.col
}

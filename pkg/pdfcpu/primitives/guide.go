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
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Guide represents horizontal and vertical lines at (x,y) for layout purposes.
type Guide struct {
	Position [2]float64 `json:"pos"` // x,y
	x, y     float64
}

func (g *Guide) validate() {
	g.x = g.Position[0]
	g.y = g.Position[1]
}

func (g *Guide) render(w io.Writer, r *types.Rectangle, pdf *PDF) {
	x, y := types.NormalizeCoord(g.x, g.y, r, pdf.origin, true)
	if g.x < 0 {
		x = 0
	}
	if g.y < 0 {
		y = 0
	}
	draw.DrawHairCross(w, x, y, r)
}

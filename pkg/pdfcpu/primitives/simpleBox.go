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
	"fmt"
	"math"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// SimpleBox is a positioned rectangular region within content.
type SimpleBox struct {
	pdf       *PDF
	content   *Content
	Name      string
	Position  [2]float64 `json:"pos"` // x,y
	x, y      float64
	Dx, Dy    float64
	Anchor    string
	anchor    types.Anchor
	anchored  bool
	Width     float64
	Height    float64
	Margin    *Margin
	Border    *Border
	FillColor string `json:"fillCol"`
	fillCol   *color.SimpleColor
	Rotation  float64 `json:"rot"`
	Hide      bool
}

func (sb *SimpleBox) validate() error {

	sb.x = sb.Position[0]
	sb.y = sb.Position[1]

	if sb.Name == "$" {
		return errors.New("pdfcpu: invalid box reference $")
	}

	if sb.Anchor != "" {
		if sb.Position[0] != 0 || sb.Position[1] != 0 {
			return errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
		}
		a, err := types.ParseAnchor(sb.Anchor)
		if err != nil {
			return err
		}
		sb.anchor = a
		sb.anchored = true
	}

	if sb.Margin != nil {
		if err := sb.Margin.validate(); err != nil {
			return err
		}
	}

	if sb.Border != nil {
		sb.Border.pdf = sb.pdf
		if err := sb.Border.validate(); err != nil {
			return err
		}
	}

	if sb.FillColor != "" {
		sc, err := sb.pdf.parseColor(sb.FillColor)
		if err != nil {
			return err
		}
		sb.fillCol = sc
	}

	return nil
}

func (sb *SimpleBox) margin(name string) *Margin {
	return sb.content.namedMargin(name)
}

func (sb *SimpleBox) border(name string) *Border {
	return sb.content.namedBorder(name)
}

func (sb *SimpleBox) mergeIn(sb0 *SimpleBox) {
	if sb.Width == 0 {
		sb.Width = sb0.Width
	}
	if sb.Height == 0 {
		sb.Height = sb0.Height
	}
	if !sb.anchored && sb.x == 0 && sb.y == 0 {
		sb.x = sb0.x
		sb.y = sb0.y
		sb.anchor = sb0.anchor
		sb.anchored = sb0.anchored
	}

	if sb.Dx == 0 {
		sb.Dx = sb0.Dx
	}
	if sb.Dy == 0 {
		sb.Dy = sb0.Dy
	}

	if sb.Margin == nil {
		sb.Margin = sb0.Margin
	}

	if sb.Border == nil {
		sb.Border = sb0.Border
	}

	if sb.fillCol == nil {
		sb.fillCol = sb0.fillCol
	}

	if sb.Rotation == 0 {
		sb.Rotation = sb0.Rotation
	}

	if !sb.Hide {
		sb.Hide = sb0.Hide
	}
}

func (sb *SimpleBox) calcBorder() (float64, *color.SimpleColor, types.LineJoinStyle, error) {
	bWidth := 0.
	var bCol *color.SimpleColor
	bStyle := types.LJMiter
	if sb.Border != nil {
		b := sb.Border
		if b.Name != "" && b.Name[0] == '$' {
			// Use named border
			bName := b.Name[1:]
			b0 := sb.border(bName)
			if b0 == nil {
				return bWidth, bCol, bStyle, errors.Errorf("pdfcpu: unknown named border %s", bName)
			}
			b.mergeIn(b0)
		}
		if b.Width >= 0 {
			bWidth = float64(b.Width)
			if b.col != nil {
				bCol = b.col
			}
			bStyle = b.style
		}
	}
	return bWidth, bCol, bStyle, nil
}

func (sb *SimpleBox) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if sb.Margin != nil {
		m := sb.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := sb.margin(mName)
			if m0 == nil {
				return mTop, mRight, mBottom, mLeft, errors.Errorf("pdfcpu: unknown named margin %s", mName)
			}
			m.mergeIn(m0)
		}
		if m.Width > 0 {
			mTop = m.Width
			mRight = m.Width
			mBottom = m.Width
			mLeft = m.Width
		} else {
			mTop = m.Top
			mRight = m.Right
			mBottom = m.Bottom
			mLeft = m.Left
		}
	}
	return mTop, mRight, mBottom, mLeft, nil
}

func (sb *SimpleBox) calcTransform(mLeft, mBottom, mRight, mTop, bWidth float64) (matrix.Matrix, *types.Rectangle) {
	pdf := sb.content.page.pdf
	cBox := sb.content.Box()
	r := sb.content.Box().CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	var x, y float64
	if sb.anchored {
		x, y = types.AnchorPosition(sb.anchor, r, sb.Width, sb.Height)
	} else {
		x, y = types.NormalizeCoord(sb.x, sb.y, cBox, pdf.origin, false)
		if y < 0 {
			y = cBox.Center().Y - sb.Height/2 - r.LL.Y
		} else if y > 0 {
			y -= mBottom
		}
		if x < 0 {
			x = cBox.Center().X - sb.Width/2 - r.LL.X
		} else if x > 0 {
			x -= mLeft
		}
	}

	dx, dy := types.NormalizeOffset(sb.Dx, sb.Dy, pdf.origin)
	x += r.LL.X + dx
	y += r.LL.Y + dy

	if x < r.LL.X {
		x = r.LL.X
	} else if x > r.UR.X-sb.Width {
		x = r.UR.X - sb.Width
	}

	if y < r.LL.Y {
		y = r.LL.Y
	} else if y > r.UR.Y-sb.Height {
		y = r.UR.Y - sb.Height
	}

	r = types.RectForWidthAndHeight(x, y, sb.Width, sb.Height)
	r.LL.X += bWidth / 2
	r.LL.Y += bWidth / 2
	r.UR.X -= bWidth / 2
	r.UR.Y -= bWidth / 2

	sin := math.Sin(float64(sb.Rotation) * float64(matrix.DegToRad))
	cos := math.Cos(float64(sb.Rotation) * float64(matrix.DegToRad))

	dx = r.LL.X
	dy = r.LL.Y
	r.Translate(-r.LL.X, -r.LL.Y)

	dx += r.Width()/2 + sin*(r.Height()/2) - cos*r.Width()/2
	dy += r.Height()/2 - cos*(r.Height()/2) - sin*r.Width()/2

	m := matrix.CalcTransformMatrix(1, 1, sin, cos, dx, dy)

	return m, r
}

func (sb *SimpleBox) render(p *model.Page) error {

	bWidth, bCol, bStyle, err := sb.calcBorder()
	if err != nil {
		return err
	}

	mTop, mRight, mBottom, mLeft, err := sb.calcMargin()
	if err != nil {
		return err
	}

	m, r := sb.calcTransform(mTop, mRight, mBottom, mLeft, bWidth)

	fmt.Fprintf(p.Buf, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	if sb.fillCol != nil {
		draw.FillRect(p.Buf, r, bWidth, bCol, *sb.fillCol, &bStyle)
		if sb.pdf.Debug {
			draw.DrawCircle(p.Buf, r.LL.X, r.LL.Y, 5, color.Black, &color.Red)
		}
		fmt.Fprint(p.Buf, "Q ")
		return nil
	}

	draw.DrawRect(p.Buf, r, bWidth, bCol, &bStyle)
	if sb.pdf.Debug {
		draw.DrawCircle(p.Buf, r.LL.X, r.LL.Y, 5, color.Black, &color.Red)
	}
	fmt.Fprint(p.Buf, "Q ")
	return nil
}

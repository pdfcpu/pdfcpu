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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/format"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

type TableHeader struct {
	Values          []string
	ColAnchors      []string
	colAnchors      []types.Anchor
	BackgroundColor string `json:"bgCol"`
	bgCol           *color.SimpleColor
	Font            *FormFont // defaults to table font
	RTL             bool
}

func (th *TableHeader) validate(pdf *PDF, cols int) error {

	if th.Values == nil || len(th.Values) != cols {
		return errors.Errorf("pdfcpu: wants %d table header values", cols)
	}

	if len(th.ColAnchors) > 0 {
		th.colAnchors = make([]types.Anchor, cols)
		if len(th.ColAnchors) != cols {
			return errors.New("pdfcpu: table header colAnchors must be specified for each column.")
		}
		for i, s := range th.ColAnchors {
			a, err := types.ParseAnchor(s)
			if err != nil {
				return err
			}
			th.colAnchors[i] = a
		}
	}

	if th.Font != nil {
		th.Font.pdf = pdf
		if err := th.Font.validate(); err != nil {
			return err
		}
	}

	if th.BackgroundColor != "" {
		sc, err := pdf.parseColor(th.BackgroundColor)
		if err != nil {
			return err
		}
		th.bgCol = sc
	}
	return nil
}

// Table represents a positioned fillable data grid including a header row.
type Table struct {
	pdf             *PDF
	content         *Content
	Name            string
	Values          [][]string
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Dx, Dy          float64
	Anchor          string
	anchor          types.Anchor
	anchored        bool
	Width           float64 // if < 1 then fraction of content width
	Rows, Cols      int
	ColWidths       []int // optional column width percentages
	ColAnchors      []string
	colAnchors      []types.Anchor
	LineHeight      int `json:"lheight"`
	Font            *FormFont
	Margin          *Margin
	Border          *Border
	Padding         *Padding
	BackgroundColor string `json:"bgCol"`
	OddColor        string `json:"oddCol"`
	EvenColor       string `json:"evenCol"`
	bgCol           *color.SimpleColor
	oddCol          *color.SimpleColor
	evenCol         *color.SimpleColor
	RTL             bool
	Rotation        float64 `json:"rot"`
	Grid            bool
	Hide            bool
	Header          *TableHeader
}

func (t *Table) Height() float64 {
	i := t.Rows
	if t.Header != nil {
		i++
	}
	return float64(i) * float64(t.LineHeight)
}

func (t *Table) validateAnchor() error {
	if t.Anchor != "" {
		if t.Position[0] != 0 || t.Position[1] != 0 {
			return errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
		}
		a, err := types.ParseAnchor(t.Anchor)
		if err != nil {
			return err
		}
		t.anchor = a
		t.anchored = true
	}
	return nil
}

func (t *Table) validateColWidths() error {
	// Missing colWidths results in uniform grid.
	if len(t.ColWidths) > 0 {
		if len(t.ColWidths) != t.Cols {
			return errors.New("pdfcpu: colWidths must be specified for each column.")
		}
		total := 0
		for _, w := range t.ColWidths {
			if w <= 0 || w >= 100 {
				return errors.New("pdfcpu: colWidths 0 < wi < 1")
			}
			total += w
		}
		if total != 100 {
			return errors.New("pdfcpu: colWidths % total must be 100.")
		}
	}
	return nil
}

func (t *Table) validateColAnchors() error {
	t.colAnchors = make([]types.Anchor, t.Cols)
	for i := range t.colAnchors {
		t.colAnchors[i] = types.Center
	}
	if len(t.ColAnchors) > 0 {
		if len(t.ColAnchors) != t.Cols {
			return errors.New("pdfcpu: colAnchors must be specified for each column.")
		}
		for i, s := range t.ColAnchors {
			a, err := types.ParseAnchor(s)
			if err != nil {
				return err
			}
			t.colAnchors[i] = a
		}
	}
	return nil
}

func (t *Table) validateValues() error {
	if t.Values != nil {
		if len(t.Values) > t.Rows {
			return errors.Errorf("pdfcpu: values for more than %d rows", t.Rows)
		}
		for _, vv := range t.Values {
			if len(vv) > t.Cols {
				return errors.Errorf("pdfcpu: values for more than %d cols", t.Cols)
			}
		}
	}
	return nil
}

func (t *Table) validateFont() error {
	if t.Font != nil {
		t.Font.pdf = t.pdf
		if err := t.Font.validate(); err != nil {
			return err
		}
	} else if !strings.HasPrefix(t.Name, "$") {
		return errors.New("pdfcpu: table missing font definition")
	}
	return nil
}

func (t *Table) validateMargin() error {
	if t.Margin != nil {
		if err := t.Margin.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Table) validateBorder() error {
	if t.Border != nil {
		t.Border.pdf = t.pdf
		if err := t.Border.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Table) validatePadding() error {
	if t.Padding != nil {
		if err := t.Padding.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Table) validateBackgroundColor() error {
	if t.BackgroundColor != "" {
		sc, err := t.pdf.parseColor(t.BackgroundColor)
		if err != nil {
			return err
		}
		t.bgCol = sc
	}
	return nil
}

func (t *Table) validateOddColor() error {
	if t.OddColor != "" {
		sc, err := t.pdf.parseColor(t.OddColor)
		if err != nil {
			return err
		}
		t.oddCol = sc
	}
	return nil
}

func (t *Table) validateEvenColor() error {
	if t.EvenColor != "" {
		sc, err := t.pdf.parseColor(t.EvenColor)
		if err != nil {
			return err
		}
		t.evenCol = sc
	}
	return nil
}

func (t *Table) validateColors() error {
	if err := t.validateBackgroundColor(); err != nil {
		return err
	}
	if err := t.validateOddColor(); err != nil {
		return err
	}
	return t.validateEvenColor()
}

func (t *Table) validate() error {

	t.x = t.Position[0]
	t.y = t.Position[1]

	if t.Name == "$" {
		return errors.New("pdfcpu: invalid table reference $")
	}

	if err := t.validateAnchor(); err != nil {
		return nil
	}

	// TODO validate width against content box width

	if t.Rows < 1 {
		return errors.New("pdfcpu: table \"rows\" missing.")
	}
	if t.Cols < 1 {
		return errors.New("pdfcpu: table \"cols\" missing.")
	}

	if t.Header != nil {
		if err := t.Header.validate(t.pdf, t.Cols); err != nil {
			return err
		}
	}

	if err := t.validateColWidths(); err != nil {
		return err
	}

	if err := t.validateColAnchors(); err != nil {
		return err
	}

	if t.LineHeight <= 0 {
		return errors.New("pdfcpu: line height \"lheight\" missing.")
	}

	if err := t.validateValues(); err != nil {
		return err
	}

	if err := t.validateFont(); err != nil {
		return err
	}

	if err := t.validateMargin(); err != nil {
		return err
	}

	if err := t.validateBorder(); err != nil {
		return err
	}

	if err := t.validatePadding(); err != nil {
		return err
	}

	return t.validateColors()
}

func (t *Table) font(name string) *FormFont {
	return t.content.namedFont(name)
}

func (t *Table) margin(name string) *Margin {
	return t.content.namedMargin(name)
}

func (t *Table) border(name string) *Border {
	return t.content.namedBorder(name)
}

func (t *Table) padding(name string) *Padding {
	return t.content.namedPadding(name)
}

func (t *Table) mergeInAnchor(t0 *Table) {
	if !t.anchored && t.x == 0 && t.y == 0 {
		t.x = t0.x
		t.y = t0.y
		t.anchor = t0.anchor
		t.anchored = t0.anchored
	}
}

func (t *Table) mergeIn(t0 *Table) {

	t.mergeInAnchor(t0)

	if t.Dx == 0 {
		t.Dx = t0.Dx
	}
	if t.Dy == 0 {
		t.Dy = t0.Dy
	}

	if t.Width == 0 {
		t.Width = t0.Width
	}

	if t.Margin == nil {
		t.Margin = t0.Margin
	}

	if t.Border == nil {
		t.Border = t0.Border
	}

	if t.Padding == nil {
		t.Padding = t0.Padding
	}

	if t.Font == nil {
		t.Font = t0.Font
	}

	if t.bgCol == nil {
		t.bgCol = t0.bgCol
	}

	if t.oddCol == nil {
		t.oddCol = t0.oddCol
	}

	if t.evenCol == nil {
		t.evenCol = t0.evenCol
	}

	if t.Rotation == 0 {
		t.Rotation = t0.Rotation
	}

	if !t.Hide {
		t.Hide = t0.Hide
	}
}

func (t *Table) calcFont() error {
	f := t.Font
	if f.Name[0] == '$' {
		// use named font
		fName := f.Name[1:]
		f0 := t.font(fName)
		if f0 == nil {
			return errors.Errorf("pdfcpu: unknown font name %s", fName)
		}
		f.Name = f0.Name
		if f.Size == 0 {
			f.Size = f0.Size
		}
		if f.col == nil {
			f.col = f0.col
		}
		if f.Lang == "" {
			f.Lang = f0.Lang
		}
		if f.Script == "" {
			f.Script = f0.Script
		}
	}
	if f.col == nil {
		f.col = &color.Black
	}
	return nil
}

func (t *Table) calcBorder() (float64, *color.SimpleColor, types.LineJoinStyle, error) {
	bWidth := 0.
	var bCol *color.SimpleColor
	bStyle := types.LJMiter
	if t.Border != nil {
		b := t.Border
		if b.Name != "" && b.Name[0] == '$' {
			// Use named border
			bName := b.Name[1:]
			b0 := t.border(bName)
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

func (t *Table) calcMargin() (float64, float64, float64, float64, error) {
	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if t.Margin != nil {
		m := t.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := t.margin(mName)
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

func (t *Table) calcTransform(mLeft, mBottom, mRight, mTop, bWidth float64) (matrix.Matrix, *types.Rectangle) {
	pdf := t.content.page.pdf
	cBox := t.content.Box()
	r := t.content.Box().CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	var x, y float64
	h := t.Height() + 2*bWidth
	if t.anchored {
		x, y = types.AnchorPosition(t.anchor, r, t.Width, h)
	} else {
		x, y = types.NormalizeCoord(t.x, t.y, r, pdf.origin, false)
		if y < 0 {
			y = cBox.Center().Y - h/2 - r.LL.Y
		} else if y > 0 {
			y -= mBottom
		}
		if x < 0 {
			x = cBox.Center().X - t.Width/2 - r.LL.X
		} else if x > 0 {
			x -= mLeft
		}
	}

	x += r.LL.X + t.Dx
	y += r.LL.Y + t.Dy

	if x < r.LL.X {
		x = r.LL.X
	} else if x > r.UR.X-t.Width {
		x = r.UR.X - t.Width
	}

	if y < r.LL.Y {
		y = r.LL.Y
	} else if y > r.UR.Y-h {
		y = r.UR.Y - h
	}

	r = types.RectForWidthAndHeight(x, y, t.Width, h)
	r.LL.X += bWidth / 2
	r.LL.Y += bWidth / 2
	r.UR.X -= bWidth / 2
	r.UR.Y -= bWidth / 2

	sin := math.Sin(float64(t.Rotation) * float64(matrix.DegToRad))
	cos := math.Cos(float64(t.Rotation) * float64(matrix.DegToRad))

	dx := r.LL.X
	dy := r.LL.Y
	r.Translate(-r.LL.X, -r.LL.Y)

	dx += t.Dx + r.Width()/2 + sin*(r.Height()/2) - cos*r.Width()/2
	dy += t.Dy + r.Height()/2 - cos*(r.Height()/2) - sin*r.Width()/2

	m := matrix.CalcTransformMatrix(1, 1, sin, cos, dx, dy)

	return m, r
}

func (t *Table) renderBackground(p *model.Page, bWidth float64, r *types.Rectangle) {
	x := r.LL.X + bWidth/2
	// Render odd,even row background.
	if t.oddCol != nil || t.evenCol != nil {
		x, w := x, t.Width-2*bWidth
		if bWidth == 0 {
			// Reduce artefacts.
			x += .5
			w -= 1
		}
		for i := 0; i < t.Rows; i++ {
			col := t.evenCol
			if i%2 > 0 {
				col = t.oddCol
			}
			if col == nil {
				continue
			}
			y := r.LL.Y + bWidth/2 + float64(i*t.LineHeight)
			r1 := types.RectForWidthAndHeight(x, y, w, float64(t.LineHeight))
			draw.FillRect(p.Buf, r1, 0, nil, *col, nil)
		}
	}

	// Render header background.
	if t.Header != nil && t.Header.bgCol != nil {
		x, w := x, t.Width-2*bWidth
		h := float64(t.LineHeight)
		if bWidth == 0 {
			// Reduce artefacts.
			x += .5
			w -= 1
			h -= .5
		}
		y := r.LL.Y + bWidth/2 + float64(t.Rows*t.LineHeight)
		col := t.Header.bgCol
		r1 := types.RectForWidthAndHeight(x, y, w, h)
		draw.FillRect(p.Buf, r1, 0, nil, *col, nil)
	}
}

func (t *Table) prepareColWidths(bWidth float64) []float64 {
	colWidths := make([]float64, t.Cols)
	w := t.Width - 2*bWidth
	if len(t.ColWidths) > 0 {
		for i := 0; i < t.Cols; i++ {
			colWidths[i] = float64(t.ColWidths[i]) / 100 * w
		}
	} else {
		colw := w / float64(t.Cols)
		for i := 0; i < t.Cols; i++ {
			colWidths[i] = colw
		}
	}
	return colWidths
}

func (t *Table) renderGrid(p *model.Page, colWidths []float64, bWidth float64, bCol *color.SimpleColor, r *types.Rectangle) {
	// Draw vertical lines.
	x := r.LL.X + bWidth/2
	for i := 1; i < t.Cols; i++ {
		x += colWidths[i-1]
		draw.DrawLine(p.Buf, x, r.LL.Y, x, r.UR.Y, 0, bCol, nil)
	}

	// Draw horizontal lines.
	maxRows := t.Rows
	if t.Header != nil {
		maxRows++
	}
	y := r.LL.Y + bWidth/2
	for i := 1; i < maxRows; i++ {
		y += float64(t.LineHeight)
		//y := r.LL.Y + bWidth/2 + float64(i*t.LineHeight)
		draw.DrawLine(p.Buf, r.LL.X, y, r.UR.X, y, 0, bCol, nil)
	}
}

func (t *Table) prepareTextDescriptor() (model.TextDescriptor, error) {
	td := model.TextDescriptor{
		Scale:    1.,
		ScaleAbs: true,
		//ShowTextBB:     true,
		//ShowBorder:     true,
		//ShowBackground: true,
		//BackgroundCol:  pdfcpu.White,
	}
	if t.Padding != nil {
		p := t.Padding
		if p.Name != "" && p.Name[0] == '$' {
			// use named padding
			pName := p.Name[1:]
			p0 := t.padding(pName)
			if p0 == nil {
				return td, errors.Errorf("pdfcpu: unknown named padding %s", pName)
			}
			p.mergeIn(p0)
		}

		if p.Width > 0 {
			td.MTop = p.Width
			td.MRight = p.Width
			td.MBot = p.Width
			td.MLeft = p.Width
		} else {
			td.MTop = p.Top
			td.MRight = p.Right
			td.MBot = p.Bottom
			td.MLeft = p.Left
		}
	}
	return td, nil
}

func (t *Table) renderValues(p *model.Page, pageNr int, fonts model.FontMap, colWidths []float64, td model.TextDescriptor, ll func(row, col int) (float64, float64)) error {
	pdf := t.pdf
	f := t.Font
	id, err := pdf.idForFontName(f.Name, f.Lang, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td.FontName = f.Name
	td.FontKey = id
	td.FontSize = f.Size
	td.RTL = t.RTL
	td.StrokeCol = *f.col
	td.FillCol = *f.col

	// Render values
	for i := 0; i < t.Rows; i++ {
		if len(t.Values) < i+1 {
			break
		}
		for j := 0; j < t.Cols; j++ {
			if len(t.Values[i]) < j+1 {
				break
			}
			s := t.Values[i][j]
			if len(strings.TrimSpace(s)) == 0 {
				continue
			}
			td.Text, _ = format.Text(s, pdf.TimestampFormat, pageNr, pdf.pageCount())
			row := i
			if t.Header != nil {
				row++
			}
			x, y := ll(row, j)
			r1 := types.RectForWidthAndHeight(x, y, colWidths[j], float64(t.LineHeight))
			bb := model.WriteMultiLineAnchored(pdf.XRefTable, p.Buf, r1, nil, td, t.colAnchors[j])
			if bb.Width() > colWidths[j] {
				return errors.Errorf("pdfcpu: table cell width overflow - reduce padding or text: %s", td.Text)
			}
			if bb.Height() > float64(t.LineHeight) {
				return errors.Errorf("pdfcpu: table cell height overflow - reduce padding or text: %s", td.Text)
			}
		}
	}
	return nil
}

func (t *Table) renderHeader(p *model.Page, pageNr int, fonts model.FontMap, colWidths []float64, td model.TextDescriptor, ll func(row, col int) (float64, float64)) error {
	h := t.Header
	f1 := *t.Font
	if h.Font != nil {
		f1 = *h.Font
	}
	if f1.Name[0] == '$' {
		// use named font
		fName := f1.Name[1:]
		f0 := t.font(fName)
		if f0 == nil {
			return errors.Errorf("pdfcpu: unknown font name %s", fName)
		}
		f1.Name = f0.Name
		f1.Script = f0.Script
		if f1.Size == 0 {
			f1.Size = f0.Size
		}
		if f1.col == nil {
			f1.col = f0.col
		}
	}
	if f1.col == nil {
		f1.col = &color.Black
	}

	id, err := t.pdf.idForFontName(f1.Name, f1.Lang, p.Fm, fonts, pageNr)
	if err != nil {
		return err
	}

	td.FontName = f1.Name
	td.FontKey = id
	td.FontSize = f1.Size
	td.RTL = h.RTL
	td.StrokeCol = *f1.col
	td.FillCol = *f1.col

	pdf := t.content.page.pdf

	for i, s := range t.Header.Values {
		if len(strings.TrimSpace(s)) == 0 {
			continue
		}
		td.Text, _ = format.Text(s, pdf.TimestampFormat, pageNr, pdf.pageCount())
		x, y := ll(0, i)
		r1 := types.RectForWidthAndHeight(x, y, colWidths[i], float64(t.LineHeight))
		a := t.colAnchors[i]
		if len(t.Header.colAnchors) > 0 {
			a = t.Header.colAnchors[i]
		}
		bb := model.WriteMultiLineAnchored(t.pdf.XRefTable, p.Buf, r1, nil, td, a)
		if bb.Width() > colWidths[i] {
			return errors.Errorf("pdfcpu: table header cell width overflow - reduce padding or text: %s", td.Text)
		}
		if bb.Height() > float64(t.LineHeight) {
			return errors.Errorf("pdfcpu: table header cell height overflow - reduce padding or text: %s", td.Text)
		}
	}
	return nil
}

func (t *Table) render(p *model.Page, pageNr int, fonts model.FontMap) error {

	if err := t.calcFont(); err != nil {
		return err
	}

	bWidth, bCol, bStyle, err := t.calcBorder()
	if err != nil {
		return err
	}

	mTop, mRight, mBottom, mLeft, err := t.calcMargin()
	if err != nil {
		return err
	}

	m, r := t.calcTransform(mTop, mRight, mBottom, mLeft, bWidth)

	fmt.Fprintf(p.Buf, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	if t.bgCol != nil {
		draw.FillRect(p.Buf, r, bWidth, bCol, *t.bgCol, &bStyle)
	}

	draw.DrawRect(p.Buf, r, bWidth, bCol, &bStyle)

	t.renderBackground(p, bWidth, r)

	colWidths := t.prepareColWidths(bWidth)

	if t.Grid {
		t.renderGrid(p, colWidths, bWidth, bCol, r)
	}

	td, err := t.prepareTextDescriptor()
	if err != nil {
		return err
	}

	ll := func(row, col int) (float64, float64) {
		var x float64
		for i := 0; i < col; i++ {
			x += colWidths[i]
		}
		y := r.UR.Y - bWidth/2 - float64((row+1)*t.LineHeight)
		return r.LL.X + bWidth/2 + x, y
	}

	if len(t.Values) > 0 {
		if err := t.renderValues(p, pageNr, fonts, colWidths, td, ll); err != nil {
			return err
		}
	}

	if t.Header != nil {
		if err := t.renderHeader(p, pageNr, fonts, colWidths, td, ll); err != nil {
			return err
		}
	}
	if t.pdf.Debug {
		draw.DrawCircle(p.Buf, r.LL.X, r.LL.Y, 5, color.Black, &color.Red)
	}

	fmt.Fprint(p.Buf, "Q ")
	return nil
}

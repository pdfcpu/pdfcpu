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

package pdfcpu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Corner represents one of four rectangle corners.
type Corner int

// The four corners of a rectangle.
const (
	LowerLeft Corner = iota
	LowerRight
	UpperLeft
	UpperRight
)

func parseOrigin(s string) (Corner, error) {
	var c Corner
	switch strings.ToLower(s) {
	case "ll", "lowerleft":
		c = LowerLeft
	case "lr", "lowerright":
		c = LowerRight
	case "ul", "upperleft":
		c = UpperLeft
	case "ur", "upperright":
		c = UpperRight
	default:
		return c, errors.Errorf("pdfcpu: unknown origin (ll, lr, ul, ur): %s", s)
	}
	return c, nil
}

func parseRegionOrientation(s string) (Orientation, error) {
	var o Orientation
	switch strings.ToLower(s) {
	case "h", "hor", "horizontal":
		o = Horizontal
	case "v", "vert", "vertical":
		o = Vertical
	default:
		return o, errors.Errorf("pdfcpu: unknown region orientation (hor, vert): %s", s)
	}
	return o, nil
}

func parseAnchor(s string) (anchor, error) {
	var a anchor
	switch strings.ToLower(s) {
	case "tl", "topleft":
		a = TopLeft
	case "tc", "topcenter":
		a = TopCenter
	case "tr", "topright":
		a = TopRight
	case "l", "left":
		a = Left
	case "c", "center":
		a = Center
	case "r", "right":
		a = Right
	case "bl", "bottomleft":
		a = BottomLeft
	case "bc", "bottomcenter":
		a = BottomCenter
	case "br", "bottomright":
		a = BottomRight
	default:
		return a, errors.Errorf("pdfcpu: unknown anchor: %s", s)
	}
	return a, nil
}

type ImageData struct {
	Payload       string // base64 encoded image data
	Format        string // jpeg, png, webp, tiff, ccitt
	Width, Height int
}
type ImageBox struct {
	pdf             *PDF
	content         *Content
	Name            string
	FileName        string `json:"file"` // path of image file name
	Data            *ImageData
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Dx, Dy          float64
	Anchor          string
	anchor          anchor
	anchored        bool
	Width           float64
	Height          float64
	Margin          *Margin
	Border          *Border
	Padding         *Padding
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Rotation        float64 `json:"rot"`
	URL             string
	Hide            bool
}

func (ib *ImageBox) validate() error {

	ib.x = ib.Position[0]
	ib.y = ib.Position[1]

	if ib.Name == "$" {
		return errors.New("pdfcpu: invalid image reference $")
	}

	if ib.FileName != "" {
		s, err := ib.pdf.fileName(ib.FileName)
		if err != nil {
			return err
		}
		ib.FileName = s
	}

	if ib.Anchor != "" {
		if ib.Position[0] != 0 || ib.Position[1] != 0 {
			return errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
		}
		a, err := parseAnchor(ib.Anchor)
		if err != nil {
			return err
		}
		ib.anchor = a
		ib.anchored = true
	}

	if ib.Margin != nil {
		if err := ib.Margin.validate(); err != nil {
			return err
		}
	}

	if ib.Border != nil {
		ib.Border.pdf = ib.pdf
		if err := ib.Border.validate(); err != nil {
			return err
		}
	}

	if ib.Padding != nil {
		if err := ib.Padding.validate(); err != nil {
			return err
		}
	}

	if ib.BackgroundColor != "" {
		sc, err := ib.pdf.parseColor(ib.BackgroundColor)
		if err != nil {
			return err
		}
		ib.bgCol = sc
	}

	return nil
}

func (ib *ImageBox) margin(name string) *Margin {
	return ib.content.namedMargin(name)
}

func (ib *ImageBox) border(name string) *Border {
	return ib.content.namedBorder(name)
}

func (ib *ImageBox) padding(name string) *Padding {
	return ib.content.namedPadding(name)
}

func (ib *ImageBox) mergeIn(ib0 *ImageBox) {

	if !ib.anchored && ib.x == 0 && ib.y == 0 {
		ib.x = ib0.x
		ib.y = ib0.y
		ib.anchor = ib0.anchor
		ib.anchored = ib0.anchored
	}

	if ib.Dx == 0 {
		ib.Dx = ib0.Dx
	}
	if ib.Dy == 0 {
		ib.Dy = ib0.Dy
	}

	if ib.Width == 0 {
		ib.Width = ib0.Width
	}

	if ib.Margin == nil {
		ib.Margin = ib0.Margin
	}

	if ib.Border == nil {
		ib.Border = ib0.Border
	}

	if ib.Padding == nil {
		ib.Padding = ib0.Padding
	}

	if ib.FileName == "" && ib.Data == nil {
		ib.FileName = ib0.FileName
		ib.Data = ib0.Data
	}

	if ib.bgCol == nil {
		ib.bgCol = ib0.bgCol
	}

	if ib.Rotation == 0 {
		ib.Rotation = ib0.Rotation
	}

	if !ib.Hide {
		ib.Hide = ib0.Hide
	}
}

func (ib *ImageBox) image(pageImages, images ImageMap) (int, int, string, error) {
	var (
		w, h   int
		id     string
		indRef *IndirectRef
	)
	img, ok := pageImages[ib.FileName]
	if ok {
		w = img.width
		h = img.height
		id = img.res.id
	} else {
		img, ok = images[ib.FileName]
		if ok {
			w = img.width
			h = img.height
			id = img.res.id
			pageImages[ib.FileName] = img
		} else {
			f, err := os.Open(ib.FileName)
			if err != nil {
				return w, h, id, err
			}
			defer f.Close()
			indRef, w, h, err = createImageResource(ib.pdf.xRefTable, f, false, false)
			if err != nil {
				return w, h, id, err
			}
			id = "Im" + strconv.Itoa(len(images))
			res := Resource{id: id, indRef: *indRef}
			img := ImageResource{res: res, width: w, height: h}
			images[ib.FileName] = img
			pageImages[ib.FileName] = img
		}
	}

	return w, h, id, nil
}

func (ib *ImageBox) render(p *Page, images ImageMap) error {
	pdf := ib.content.page.pdf
	bWidth := 0.
	var bCol *SimpleColor
	bStyle := LJMiter
	if ib.Border != nil {
		b := ib.Border
		if b.Name != "" && b.Name[0] == '$' {
			// Use named border
			bName := b.Name[1:]
			b0 := ib.border(bName)
			if b0 == nil {
				return errors.Errorf("pdfcpu: unknown named border %s", bName)
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

	mTop, mRight, mBottom, mLeft := 0., 0., 0., 0.
	if ib.Margin != nil {
		m := ib.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := ib.margin(mName)
			if m0 == nil {
				return errors.Errorf("pdfcpu: unknown named margin %s", mName)
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

	pTop, pRight, pBot, pLeft := 0., 0., 0., 0.
	if ib.Padding != nil {
		p := ib.Padding
		if p.Name != "" && p.Name[0] == '$' {
			// use named padding
			pName := p.Name[1:]
			p0 := ib.padding(pName)
			if p0 == nil {
				return errors.Errorf("pdfcpu: unknown named padding %s", pName)
			}
			p.mergeIn(p0)
		}

		if p.Width > 0 {
			pTop = p.Width
			pRight = p.Width
			pBot = p.Width
			pLeft = p.Width
		} else {
			pTop = p.Top
			pRight = p.Right
			pBot = p.Bottom
			pLeft = p.Left
		}
	}

	w, h, id, err := ib.image(p.Im, images)
	if err != nil {
		return err
	}

	if bWidth > 0 && bCol == nil {
		bWidth = 0
	}

	rSrc := RectForDim(float64(w), float64(h))

	if ib.Width == 0 && ib.Height == 0 {
		ib.Width = float64(w)
		ib.Height = float64(h)
	} else if ib.Width == 0 {
		ib.Width = rSrc.ScaledWidth(ib.Height-2*bWidth-pTop-pBot) + 2*bWidth + pLeft + pRight
	} else if ib.Height == 0 {
		ib.Height = rSrc.ScaledHeight(ib.Width-2*bWidth-pLeft-pRight) + 2*bWidth + pTop + pBot
	}

	cBox := ib.content.Box()
	r := ib.content.Box().CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBottom
	r.UR.X -= mRight
	r.UR.Y -= mTop

	var x, y float64
	if ib.anchored {
		x, y = anchorPosition(ib.anchor, r, ib.Width, ib.Height)
	} else {
		x, y = coord(ib.x, ib.y, r, pdf.origin, false)
		if y < 0 {
			y = cBox.Center().Y - ib.Height/2 - r.LL.Y
		} else if y > 0 {
			y -= mBottom
		}
		if x < 0 {
			x = cBox.Center().X - ib.Width/2 - r.LL.X
		} else if x > 0 {
			x -= mLeft
		}
	}

	x += r.LL.X + ib.Dx
	y += r.LL.Y + ib.Dy

	if x < r.LL.X {
		x = r.LL.X
	} else if x > r.UR.X-ib.Width {
		x = r.UR.X - ib.Width
	}

	if y < r.LL.Y {
		y = r.LL.Y
	} else if y > r.UR.Y-ib.Height {
		y = r.UR.Y - ib.Height
	}

	//////////////////

	r = RectForWidthAndHeight(x, y, ib.Width, ib.Height)
	r.LL.X += bWidth / 2
	r.LL.Y += bWidth / 2
	r.UR.X -= bWidth / 2
	r.UR.Y -= bWidth / 2

	if bCol == nil {
		bCol = &Black
	}

	sin := math.Sin(float64(ib.Rotation) * float64(degToRad))
	cos := math.Cos(float64(ib.Rotation) * float64(degToRad))

	dx := r.LL.X
	dy := r.LL.Y
	r.Translate(-r.LL.X, -r.LL.Y)

	dx += ib.Dx + r.Width()/2 + sin*(r.Height()/2) - cos*r.Width()/2
	dy += ib.Dy + r.Height()/2 - cos*(r.Height()/2) - sin*r.Width()/2

	m := calcTransformMatrix(1, 1, sin, cos, dx, dy)
	fmt.Fprintf(p.Buf, "q %.2f %.2f %.2f %.2f %.2f %.2f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	// Render box
	if ib.bgCol != nil {
		if bWidth == 0 {
			bCol = ib.bgCol
		}
		FillRect(p.Buf, r, bWidth, bCol, *ib.bgCol, &bStyle)
	} else {
		FillRect(p.Buf, r, bWidth, bCol, *bCol, &bStyle)
	}

	fmt.Fprint(p.Buf, "Q ")

	// Render image
	rDest := RectForWidthAndHeight(x+bWidth+pLeft, y+bWidth+pBot, ib.Width-2*bWidth-pLeft-pRight, ib.Height-2*bWidth-pTop-pBot)
	sx, sy, dx, dy, _ := bestFitRectIntoRect(rSrc, rDest, false)
	dx += rDest.LL.X
	dy += rDest.LL.Y

	dx += ib.Dx + sx/2 + sin*(sy/2) - cos*sx/2
	dy += ib.Dy + sy/2 - cos*(sy/2) - sin*sx/2

	m = calcTransformMatrix(sx, sy, sin, cos, dx, dy)
	fmt.Fprintf(p.Buf, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s Do Q ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], id)

	return nil
}

// Guide represents horizontal and vertical lines at (x,y) for layout purposes.
type Guide struct {
	Position [2]float64 `json:"pos"` // x,y
	x, y     float64
}

func (g *Guide) validate() {
	g.x = g.Position[0]
	g.y = g.Position[1]
}

func (g *Guide) render(w io.Writer, r *Rectangle, pdf *PDF) {
	x, y := coord(g.x, g.y, r, pdf.origin, true)
	if g.x == 0 {
		x = 0
	}
	if g.y == 0 {
		y = 0
	}
	DrawHairCross(w, x, y, r)
}

type HorizontalBand struct {
	pdf             *PDF
	Left            string
	Center          string
	Right           string
	position        anchor // topcenter, center, bottomcenter
	Height          float64
	Dx, Dy          int
	BackgroundColor string `json:"bgCol"`
	bgCol           *SimpleColor
	Font            *FormFont
	From            int
	Thru            int
	Border          bool
}

func (hb *HorizontalBand) validate() error {

	pdf := hb.pdf

	if hb.BackgroundColor != "" {
		sc, err := pdf.parseColor(hb.BackgroundColor)
		if err != nil {
			return err
		}
		hb.bgCol = sc
	}

	if hb.Font != nil {
		hb.Font.pdf = pdf
		if err := hb.Font.validate(); err != nil {
			return err
		}
	}

	if hb.Height <= 0 {
		return errors.Errorf("pdfcpu: missing header/footer height")
	}

	return nil
}

func (hb *HorizontalBand) render(p *Page, pageNr int, top bool) error {

	if pageNr < hb.From || (hb.Thru > 0 && pageNr > hb.Thru) {
		return nil
	}

	left := BottomLeft
	center := BottomCenter
	right := BottomRight
	if top {
		left = TopLeft
		center = TopCenter
		right = TopRight
	}

	if hb.Font.Name[0] == '$' {
		if err := hb.pdf.calcFont(hb.Font); err != nil {
			return err
		}
	}

	if hb.Left != "" {
		if err := createAnchoredTextbox(hb.Left, left, hb.Dx, hb.Dy, hb.Font, hb.bgCol, pageNr, p, hb.pdf); err != nil {
			return err
		}
	}

	if hb.Center != "" {
		if err := createAnchoredTextbox(hb.Center, center, hb.Dx, hb.Dy, hb.Font, hb.bgCol, pageNr, p, hb.pdf); err != nil {
			return err
		}
	}

	if hb.Right != "" {
		if err := createAnchoredTextbox(hb.Right, right, hb.Dx, hb.Dy, hb.Font, hb.bgCol, pageNr, p, hb.pdf); err != nil {
			return err
		}
	}

	if !hb.Border {
		return nil
	}

	llx := p.CropBox.LL.X + float64(hb.Dx)
	lly := p.CropBox.LL.Y + float64(hb.Dy)
	if top {
		lly = p.CropBox.UR.Y - float64(hb.Dy) - hb.Height
	}
	w := p.CropBox.Width() - float64(2*hb.Dx)
	h := hb.Height

	r := RectForWidthAndHeight(llx, lly, w, h)

	DrawRect(p.Buf, r, 0, &Black, nil)

	return nil
}

type Divider struct {
	pdf   *PDF
	At    float64
	p, q  Point
	Width int
	Color string `json:"col"`
	col   *SimpleColor
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

func (d *Divider) render(p *Page) error {

	if d.col == nil {
		return nil
	}

	DrawLine(p.Buf, d.p.X, d.p.Y, d.q.X, d.q.Y, float64(d.Width), d.col, nil)

	return nil
}

type Bar struct {
	pdf     *PDF
	content *Content
	X, Y    float64 // either or determines orientation.
	Width   int
	Color   string `json:"col"`
	col     *SimpleColor
	Style   string
	style   LineJoinStyle
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

	b.style = LJMiter
	if b.Style != "" {
		switch b.Style {
		case "miter":
			b.style = LJMiter
		case "round":
			b.style = LJRound
		case "bevel":
			b.style = LJBevel
		default:
			return errors.Errorf("pdfcpu: invalid bar style: %s (should be \"miter\", \"round\" or \"bevel\")", b.Style)
		}
	}

	return nil
}

func (b *Bar) render(p *Page) error {

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

	px, py = coord(px, py, cBox, b.pdf.origin, true)
	qx, qy = coord(qx, qy, cBox, b.pdf.origin, true)

	DrawLine(p.Buf, px, py, qx, qy, float64(b.Width), b.col, &b.style)

	return nil
}

type Padding struct {
	Name                     string
	Width                    float64
	Top, Right, Bottom, Left float64
}

func (p *Padding) validate() error {

	if p.Name == "$" {
		return errors.New("pdfcpu: invalid padding reference $")
	}

	if p.Width < 0 {
		if p.Top > 0 || p.Right > 0 || p.Bottom > 0 || p.Left > 0 {
			return errors.Errorf("pdfcpu: invalid padding width: %f", p.Width)
		}
	}

	if p.Width > 0 {
		p.Top, p.Right, p.Bottom, p.Left = p.Width, p.Width, p.Width, p.Width
		return nil
	}

	return nil
}

func (p *Padding) mergeIn(p0 *Padding) {
	if p.Width > 0 {
		return
	}
	if p.Width < 0 {
		p.Top, p.Right, p.Bottom, p.Left = 0, 0, 0, 0
		return
	}

	if p.Top == 0 {
		p.Top = p0.Top
	} else if p.Top < 0 {
		p.Top = 0.
	}

	if p.Right == 0 {
		p.Right = p0.Right
	} else if p.Right < 0 {
		p.Right = 0.
	}

	if p.Bottom == 0 {
		p.Bottom = p0.Bottom
	} else if p.Bottom < 0 {
		p.Bottom = 0.
	}

	if p.Left == 0 {
		p.Left = p0.Left
	} else if p.Left < 0 {
		p.Left = 0.
	}
}

type Content struct {
	parent            *Content
	page              *PDFPage
	BackgroundColor   string               `json:"bgCol"`
	bgCol             *SimpleColor         // page background color
	Fonts             map[string]*FormFont // named fonts
	Margins           map[string]*Margin   // named margins
	Borders           map[string]*Border   // named borders
	Paddings          map[string]*Padding  // named paddings
	Margin            *Margin              // content margin
	Border            *Border              // content border
	Padding           *Padding             // content padding
	Regions           *Regions
	mediaBox          *Rectangle
	borderRect        *Rectangle
	box               *Rectangle
	Bars              []*Bar                `json:"bar"`
	SimpleBoxes       []*SimpleBox          `json:"box"`
	SimpleBoxPool     map[string]*SimpleBox `json:"boxes"`
	TextBoxes         []*TextBox            `json:"text"`
	TextBoxPool       map[string]*TextBox   `json:"texts"`
	ImageBoxes        []*ImageBox           `json:"image"`
	ImageBoxPool      map[string]*ImageBox  `json:"images"`
	Guides            []*Guide              // hor/vert guidelines for layout
	TextFields        []*TextField          // input text fields with optional label
	CheckBoxes        []*CheckBox           // input checkboxes with optional label
	RadioButtonGroups []*RadioButtonGroup   // input radiobutton groups with optional label
	ListBoxes         []*ScrollableListBox  // input listboxes with optional label and multi selection
	ComboBoxes        []*ComboBox           // input comboboxes with optional label and editable
}

func (c *Content) validate() error {
	pdf := c.page.pdf
	if c.BackgroundColor != "" {
		sc, err := pdf.parseColor(c.BackgroundColor)
		if err != nil {
			return err
		}
		c.bgCol = sc
	}

	for _, g := range c.Guides {
		g.validate()
	}

	if c.Border != nil {
		if len(c.Borders) > 0 {
			return errors.New("pdfcpu: Please supply either content \"border\" or \"borders\"")
		}
		c.Border.pdf = pdf
		if err := c.Border.validate(); err != nil {
			return err
		}
		c.Borders = map[string]*Border{}
		c.Borders["border"] = c.Border
	}
	for _, b := range c.Borders {
		b.pdf = pdf
		if err := b.validate(); err != nil {
			return err
		}
	}

	if c.Margin != nil {
		if len(c.Margins) > 0 {
			return errors.New("pdfcpu: Please supply either page \"margin\" or \"margins\"")
		}
		if err := c.Margin.validate(); err != nil {
			return err
		}
		c.Margins = map[string]*Margin{}
		c.Margins["margin"] = c.Margin
	}
	for _, m := range c.Margins {
		if err := m.validate(); err != nil {
			return err
		}
	}

	if c.Padding != nil {
		if len(c.Paddings) > 0 {
			return errors.New("pdfcpu: Please supply either page \"padding\" or \"paddings\"")
		}
		if err := c.Padding.validate(); err != nil {
			return err
		}
		c.Paddings = map[string]*Padding{}
		c.Paddings["padding"] = c.Padding
	}
	for _, p := range c.Paddings {
		if err := p.validate(); err != nil {
			return err
		}
	}

	if c.Regions != nil {
		s := "must be defined within region content"
		if len(c.SimpleBoxPool) > 0 {
			return errors.Errorf("pdfcpu: \"boxes\" %s", s)
		}
		if len(c.SimpleBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"box\" %s", s)
		}
		if len(c.TextBoxPool) > 0 {
			return errors.Errorf("pdfcpu: \"texts\" %s", s)
		}
		if len(c.TextBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"text\" %s", s)
		}
		if len(c.ImageBoxes) > 0 {
			return errors.Errorf("pdfcpu: \"image\" %s", s)
		}
		c.Regions.page = c.page
		c.Regions.parent = c
		return c.Regions.validate()
	}

	// bars
	for _, b := range c.Bars {
		b.pdf = pdf
		b.content = c
		if err := b.validate(); err != nil {
			return err
		}
	}

	// boxes
	for _, sb := range c.SimpleBoxPool {
		sb.pdf = pdf
		sb.content = c
		if err := sb.validate(); err != nil {
			return err
		}
	}
	for _, sb := range c.SimpleBoxes {
		sb.pdf = pdf
		sb.content = c
		if err := sb.validate(); err != nil {
			return err
		}
	}

	// text
	for _, tb := range c.TextBoxPool {
		tb.pdf = pdf
		tb.content = c
		if err := tb.validate(); err != nil {
			return err
		}
	}
	for _, tb := range c.TextBoxes {
		tb.pdf = pdf
		tb.content = c
		if err := tb.validate(); err != nil {
			return err
		}
	}

	// images
	for _, ib := range c.ImageBoxPool {
		ib.pdf = pdf
		ib.content = c
		if err := ib.validate(); err != nil {
			return err
		}
	}
	for _, ib := range c.ImageBoxes {
		ib.pdf = pdf
		ib.content = c
		if err := ib.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Content) namedFont(id string) *FormFont {
	f := c.Fonts[id]
	if f != nil {
		return f
	}
	if c.parent != nil {
		return c.parent.namedFont(id)
	}
	return c.page.namedFont(id)
}

func (c *Content) namedMargin(id string) *Margin {
	m := c.Margins[id]
	if m != nil {
		return m
	}
	if c.parent != nil {
		return c.parent.namedMargin(id)
	}
	return c.page.namedMargin(id)
}

func (c *Content) margin() *Margin {
	return c.namedMargin("margin")
}

func (c *Content) namedBorder(id string) *Border {
	b := c.Borders[id]
	if b != nil {
		return b
	}
	if c.parent != nil {
		return c.parent.namedBorder(id)
	}
	return c.page.namedBorder(id)
}

func (c *Content) border() *Border {
	return c.namedBorder("border")
}

func (c *Content) namedPadding(id string) *Padding {
	p := c.Paddings[id]
	if p != nil {
		return p
	}
	if c.parent != nil {
		return c.parent.namedPadding(id)
	}
	return c.page.namedPadding(id)
}

func (c *Content) namedSimpleBox(id string) *SimpleBox {
	sb := c.SimpleBoxPool[id]
	if sb != nil {
		return sb
	}
	if c.parent != nil {
		return c.parent.namedSimpleBox(id)
	}
	return c.page.namedSimpleBox(id)
}

func (c *Content) namedImageBox(id string) *ImageBox {
	ib := c.ImageBoxPool[id]
	if ib != nil {
		return ib
	}
	if c.parent != nil {
		return c.parent.namedImageBox(id)
	}
	return c.page.namedImageBox(id)
}

func (c *Content) namedTextBox(id string) *TextBox {
	tb := c.TextBoxPool[id]
	if tb != nil {
		return tb
	}
	if c.parent != nil {
		return c.parent.namedTextBox(id)
	}
	return c.page.namedTextBox(id)
}

func (c *Content) padding() *Padding {
	return c.namedPadding("padding")
}

func (c *Content) calcFont(ff map[string]*FormFont) {

	fff := map[string]*FormFont{}
	for id, f0 := range ff {
		fff[id] = f0
		f1 := c.Fonts[id]
		if f1 != nil {
			f1.mergeIn(f0)
			fff[id] = f1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcFont(fff)
			c.Regions.Right.calcFont(fff)
		} else {
			c.Regions.Top.calcFont(fff)
			c.Regions.Bottom.calcFont(fff)
		}
	}
}

func (c *Content) calcBorder(bb map[string]*Border) {

	bbb := map[string]*Border{}
	for id, b0 := range bb {
		bbb[id] = b0
		b1 := c.Borders[id]
		if b1 != nil {
			b1.mergeIn(b0)
			bbb[id] = b1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcBorder(bbb)
			c.Regions.Right.calcBorder(bb)
		} else {
			c.Regions.Top.calcBorder(bbb)
			c.Regions.Bottom.calcBorder(bbb)
		}
	}
}

func (c *Content) calcMargin(mm map[string]*Margin) {

	mmm := map[string]*Margin{}
	for id, m0 := range mm {
		mmm[id] = m0
		m1 := c.Margins[id]
		if m1 != nil {
			m1.mergeIn(m0)
			mmm[id] = m1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcMargin(mmm)
			c.Regions.Right.calcMargin(mmm)
		} else {
			c.Regions.Top.calcMargin(mmm)
			c.Regions.Bottom.calcMargin(mmm)
		}
	}
}

func (c *Content) calcPadding(pp map[string]*Padding) {

	ppp := map[string]*Padding{}
	for id, p0 := range pp {
		ppp[id] = p0
		p1 := c.Paddings[id]
		if p1 != nil {
			p1.mergeIn(p0)
			ppp[id] = p1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcPadding(ppp)
			c.Regions.Right.calcPadding(ppp)
		} else {
			c.Regions.Top.calcPadding(ppp)
			c.Regions.Bottom.calcPadding(ppp)
		}
	}
}

func (c *Content) calcSimpleBoxes(bb map[string]*SimpleBox) {

	bbb := map[string]*SimpleBox{}
	for id, sb0 := range bb {
		bbb[id] = sb0
		sb1 := c.SimpleBoxPool[id]
		if sb1 != nil {
			sb1.mergeIn(sb0)
			bbb[id] = sb1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcSimpleBoxes(bbb)
			c.Regions.Right.calcSimpleBoxes(bbb)
		} else {
			c.Regions.Top.calcSimpleBoxes(bbb)
			c.Regions.Bottom.calcSimpleBoxes(bbb)
		}
	}
}

func (c *Content) calcTextBoxes(bb map[string]*TextBox) {

	bbb := map[string]*TextBox{}
	for id, tb0 := range bb {
		bbb[id] = tb0
		tb1 := c.TextBoxPool[id]
		if tb1 != nil {
			tb1.mergeIn(tb0)
			bbb[id] = tb1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcTextBoxes(bbb)
			c.Regions.Right.calcTextBoxes(bbb)
		} else {
			c.Regions.Top.calcTextBoxes(bbb)
			c.Regions.Bottom.calcTextBoxes(bbb)
		}
	}
}

func (c *Content) calcImageBoxes(bb map[string]*ImageBox) {

	bbb := map[string]*ImageBox{}
	for id, ib0 := range bb {
		bbb[id] = ib0
		ib1 := c.ImageBoxPool[id]
		if ib1 != nil {
			ib1.mergeIn(ib0)
			bbb[id] = ib1
		}
	}

	if c.Regions != nil {
		if c.Regions.horizontal {
			c.Regions.Left.calcImageBoxes(bbb)
			c.Regions.Right.calcImageBoxes(bbb)
		} else {
			c.Regions.Top.calcImageBoxes(bbb)
			c.Regions.Bottom.calcImageBoxes(bbb)
		}
	}
}

func (c *Content) BorderRect() *Rectangle {

	if c.borderRect == nil {

		mLeft, mRight, mTop, mBottom, borderWidth := 0., 0., 0., 0., 0.

		m := c.margin()
		if m != nil {
			mTop = m.Top
			mRight = m.Right
			mBottom = m.Bottom
			mLeft = m.Left
		}

		b := c.border()
		if b != nil && b.col != nil && b.Width >= 0 {
			borderWidth = float64(b.Width)
		}

		c.borderRect = RectForWidthAndHeight(
			c.mediaBox.LL.X+mLeft+borderWidth/2,
			c.mediaBox.LL.Y+mBottom+borderWidth/2,
			c.mediaBox.Width()-mLeft-mRight-borderWidth,
			c.mediaBox.Height()-mTop-mBottom-borderWidth)

	}

	return c.borderRect
}

func (c *Content) Box() *Rectangle {

	if c.box == nil {

		var mTop, mRight, mBottom, mLeft float64
		var pTop, pRight, pBottom, pLeft float64
		var borderWidth float64

		m := c.margin()
		if m != nil {
			if m.Width > 0 {
				mTop, mRight, mBottom, mLeft = m.Width, m.Width, m.Width, m.Width
			} else {
				mTop = m.Top
				mRight = m.Right
				mBottom = m.Bottom
				mLeft = m.Left
			}
		}

		b := c.border()
		if b != nil && b.col != nil && b.Width >= 0 {
			borderWidth = float64(b.Width)
		}

		p := c.padding()
		if p != nil {
			if p.Width > 0 {
				pTop, pRight, pBottom, pLeft = p.Width, p.Width, p.Width, p.Width
			} else {
				pTop = p.Top
				pRight = p.Right
				pBottom = p.Bottom
				pLeft = p.Left
			}
		}

		llx := c.mediaBox.LL.X + mLeft + borderWidth + pLeft
		lly := c.mediaBox.LL.Y + mBottom + borderWidth + pBottom
		w := c.mediaBox.Width() - mLeft - mRight - 2*borderWidth - pLeft - pRight
		h := c.mediaBox.Height() - mTop - mBottom - 2*borderWidth - pTop - pBottom
		c.box = RectForWidthAndHeight(llx, lly, w, h)
	}

	return c.box
}

func (c *Content) render(p *Page, pageNr int, fonts FontMap, images ImageMap) error {

	if c.Regions != nil {
		c.Regions.mediaBox = c.mediaBox
		c.Regions.page = c.page
		return c.Regions.render(p, pageNr, fonts, images)
	}

	pdf := c.page.pdf

	// Render background
	if c.bgCol != nil {
		FillRectNoBorder(p.Buf, c.BorderRect(), *c.bgCol)
	}

	// Render border
	b := c.border()
	if b != nil && b.col != nil && b.Width >= 0 {
		DrawRect(p.Buf, c.BorderRect(), float64(b.Width), b.col, &b.style)
	}

	// Render bars
	for _, b := range c.Bars {
		if b.Hide {
			continue
		}
		if err := b.render(p); err != nil {
			return err
		}
	}

	// Render boxes
	for _, sb := range c.SimpleBoxes {
		if sb.Hide {
			continue
		}
		if sb.Name != "" && sb.Name[0] == '$' {
			// Use named simplebox
			sbName := sb.Name[1:]
			sb0 := c.namedSimpleBox(sbName)
			if sb0 == nil {
				return errors.Errorf("pdfcpu: unknown named box %s", sbName)
			}
			sb.mergeIn(sb0)
		}
		if err := sb.render(p); err != nil {
			return err
		}
	}

	// Render text
	for _, tb := range c.TextBoxes {
		if tb.Hide {
			continue
		}
		if tb.Name != "" && tb.Name[0] == '$' {
			// Use named textbox
			tbName := tb.Name[1:]
			tb0 := c.namedTextBox(tbName)
			if tb0 == nil {
				return errors.Errorf("pdfcpu: unknown named text %s", tbName)
			}
			tb.mergeIn(tb0)
		}
		if err := tb.render(p, pageNr, fonts); err != nil {
			return err
		}
	}

	// Render images
	for _, ib := range c.ImageBoxes {
		if ib.Hide {
			continue
		}
		if ib.Name != "" && ib.Name[0] == '$' {
			// Use named imagebox
			ibName := ib.Name[1:]
			ib0 := c.namedImageBox(ibName)
			if ib0 == nil {
				return errors.Errorf("pdfcpu: unknown named text %s", ibName)
			}
			ib.mergeIn(ib0)
		}
		if err := ib.render(p, images); err != nil {
			return err
		}
	}

	// Render mediaBox & contentBox
	if pdf.ContentBox {
		DrawRect(p.Buf, c.mediaBox, 0, &Green, nil)
		DrawRect(p.Buf, c.Box(), 0, &Red, nil)
	}

	// Render guides
	if pdf.Guides {
		for _, g := range c.Guides {
			g.render(p.Buf, c.Box(), pdf)
		}
	}

	return nil
}

type Regions struct {
	page        *PDFPage
	parent      *Content
	Name        string // unique
	Orientation string `json:"orient"`
	horizontal  bool
	Divider     *Divider `json:"div"`
	Left, Right *Content // 2 horizontal regions
	Top, Bottom *Content // 2 vertical regions
	mediaBox    *Rectangle
}

func (r *Regions) validate() error {

	pdf := r.page.pdf

	// trim json string necessary?
	if r.Orientation == "" {
		return errors.Errorf("pdfcpu: region is missing orientation")
	}
	o, err := parseRegionOrientation(r.Orientation)
	if err != nil {
		return err
	}
	r.horizontal = o == Horizontal

	if r.Divider == nil {
		return errors.New("pdfcpu: region is missing divider")
	}
	r.Divider.pdf = pdf
	if err := r.Divider.validate(); err != nil {
		return err
	}

	if r.horizontal {
		if r.Left == nil {
			return errors.Errorf("pdfcpu: regions %s is missing Left", r.Name)
		}
		r.Left.page = r.page
		r.Left.parent = r.parent
		if err := r.Left.validate(); err != nil {
			return err
		}
		if r.Right == nil {
			return errors.Errorf("pdfcpu: regions %s is missing Right", r.Name)
		}
		r.Right.page = r.page
		r.Right.parent = r.parent
		return r.Right.validate()
	}

	if r.Top == nil {
		return errors.Errorf("pdfcpu: regions %s is missing Top", r.Name)
	}
	r.Top.page = r.page
	r.Top.parent = r.parent
	if err := r.Top.validate(); err != nil {
		return err
	}
	if r.Bottom == nil {
		return errors.Errorf("pdfcpu: regions %s is missing Bottom", r.Name)
	}
	r.Bottom.page = r.page
	r.Bottom.parent = r.parent
	if err := r.Bottom.validate(); err != nil {
		return err
	}

	return nil
}

func (r *Regions) render(p *Page, pageNr int, fonts FontMap, images ImageMap) error {

	if r.horizontal {

		// Calc divider.
		dx := r.mediaBox.Width() * r.Divider.At
		r.Divider.p.X, r.Divider.p.Y = coord(dx, 0, r.mediaBox, r.page.pdf.origin, true)
		r.Divider.q.X, r.Divider.q.Y = coord(dx, r.mediaBox.Height(), r.mediaBox, r.page.pdf.origin, true)

		// Render left region.
		r.Left.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Left.mediaBox.UR.X = r.Divider.p.X - float64(r.Divider.Width)/2
		r.Left.page = r.page
		if err := r.Left.render(p, pageNr, fonts, images); err != nil {
			return err
		}

		// Render right region.
		r.Right.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Right.mediaBox.LL.X = r.Divider.p.X + float64(r.Divider.Width)/2
		r.Right.page = r.page
		if err := r.Right.render(p, pageNr, fonts, images); err != nil {
			return err
		}

	} else {

		// Calc divider.
		dy := r.mediaBox.Height() * r.Divider.At
		r.Divider.p.X, r.Divider.p.Y = coord(0, dy, r.mediaBox, r.page.pdf.origin, true)
		r.Divider.q.X, r.Divider.q.Y = coord(r.mediaBox.Width(), dy, r.mediaBox, r.page.pdf.origin, true)

		// Render top region.
		r.Top.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Top.mediaBox.LL.Y = r.Divider.p.Y + float64(r.Divider.Width)/2
		r.Top.page = r.page
		if err := r.Top.render(p, pageNr, fonts, images); err != nil {
			return err
		}

		// Render bottom region.
		r.Bottom.mediaBox = r.mediaBox.CroppedCopy(0)
		r.Bottom.mediaBox.UR.Y = r.Divider.p.Y - float64(r.Divider.Width)/2
		r.Bottom.page = r.page
		if err := r.Bottom.render(p, pageNr, fonts, images); err != nil {
			return err
		}

	}

	return r.Divider.render(p)
}

type PDFPage struct {
	pdf             *PDF
	number          int                   // page number
	Paper           string                // page size
	mediaBox        *Rectangle            // page media box
	Crop            string                // page crop box
	cropBox         *Rectangle            // page crop box
	BackgroundColor string                `json:"bgCol"`
	bgCol           *SimpleColor          // page background color
	Fonts           map[string]*FormFont  // default fonts
	InputFont       *FormFont             // default font for input fields
	LabelFont       *FormFont             // default font for input field labels
	annots          Array                 // page annotations
	Guides          []*Guide              // hor/vert guidelines for layout
	Margin          *Margin               // page margin
	Border          *Border               // page border
	Padding         *Padding              // page padding
	Margins         map[string]*Margin    // page scoped named margins
	Borders         map[string]*Border    // page scoped named borders
	Paddings        map[string]*Padding   // page scoped named paddings
	SimpleBoxPool   map[string]*SimpleBox `json:"boxes"`
	TextBoxPool     map[string]*TextBox   `json:"texts"`
	ImageBoxPool    map[string]*ImageBox  `json:"images"`
	Content         *Content              // textboxes, imageboxes, form fields, h/v bars, guides
}

func (page *PDFPage) validate() error {
	pdf := page.pdf
	page.mediaBox = pdf.mediaBox
	page.cropBox = pdf.cropBox
	if page.Paper != "" {
		dim, _, err := parsePageFormat(page.Paper)
		if err != nil {
			return err
		}
		page.mediaBox = RectForDim(dim.Width, dim.Height)
		page.cropBox = page.mediaBox.CroppedCopy(0)
	}

	if page.Crop != "" {
		box, err := ParseBox(page.Crop, POINTS)
		if err != nil {
			return err
		}
		page.cropBox = applyBox("CropBox", box, nil, page.mediaBox)
	}

	// Default background color
	if page.BackgroundColor != "" {
		sc, err := page.pdf.parseColor(page.BackgroundColor)
		if err != nil {
			return err
		}
		page.bgCol = sc
	}

	// Default page fonts
	for _, f := range page.Fonts {
		f.pdf = pdf
		if err := f.validate(); err != nil {
			return err
		}
	}

	// Default font for input fields
	if page.InputFont != nil {
		page.InputFont.pdf = pdf
		if err := page.InputFont.validate(); err != nil {
			return err
		}
	}

	// Default font for input field labels
	if page.LabelFont != nil {
		page.LabelFont.pdf = pdf
		if err := page.LabelFont.validate(); err != nil {
			return err
		}
	}

	for _, g := range page.Guides {
		g.validate()
	}

	if page.Border != nil {
		if len(page.Borders) > 0 {
			return errors.New("pdfcpu: Please supply either page \"border\" or \"borders\"")
		}
		page.Border.pdf = pdf
		if err := page.Border.validate(); err != nil {
			return err
		}
		page.Borders = map[string]*Border{}
		page.Borders["border"] = page.Border
	}
	for _, b := range page.Borders {
		b.pdf = pdf
		if err := b.validate(); err != nil {
			return err
		}
	}

	if page.Margin != nil {
		if len(page.Margins) > 0 {
			return errors.New("pdfcpu: Please supply either page \"margin\" or \"margins\"")
		}
		if err := page.Margin.validate(); err != nil {
			return err
		}
		page.Margins = map[string]*Margin{}
		page.Margins["margin"] = page.Margin
	}
	for _, m := range page.Margins {
		if err := m.validate(); err != nil {
			return err
		}
	}

	if page.Padding != nil {
		if len(page.Paddings) > 0 {
			return errors.New("pdfcpu: Please supply either page \"padding\" or \"paddings\"")
		}
		if err := page.Padding.validate(); err != nil {
			return err
		}
		page.Paddings = map[string]*Padding{}
		page.Paddings["padding"] = page.Padding
	}
	for _, p := range page.Paddings {
		if err := p.validate(); err != nil {
			return err
		}
	}

	// box templates
	for _, sb := range page.SimpleBoxPool {
		sb.pdf = pdf
		if err := sb.validate(); err != nil {
			return err
		}
	}

	// text templates
	for _, tb := range page.TextBoxPool {
		tb.pdf = pdf
		if err := tb.validate(); err != nil {
			return err
		}
	}

	// image templates
	for _, ib := range page.ImageBoxPool {
		ib.pdf = pdf
		if err := ib.validate(); err != nil {
			return err
		}
	}

	if page.Content == nil {
		return errors.New("pdfcpu: Please supply page \"content\"")
	}
	page.Content.page = page
	return page.Content.validate()
}

func (page *PDFPage) namedFont(id string) *FormFont {
	f := page.Fonts[id]
	if f != nil {
		return f
	}
	return page.pdf.Fonts[id]
}

func (page *PDFPage) namedMargin(id string) *Margin {
	m := page.Margins[id]
	if m != nil {
		return m
	}
	return page.pdf.Margins[id]
}

func (page *PDFPage) namedBorder(id string) *Border {
	b := page.Borders[id]
	if b != nil {
		return b
	}
	return page.pdf.Borders[id]
}

func (page *PDFPage) namedPadding(id string) *Padding {
	p := page.Paddings[id]
	if p != nil {
		return p
	}
	return page.pdf.Paddings[id]
}

func (page *PDFPage) namedSimpleBox(id string) *SimpleBox {
	sb := page.SimpleBoxPool[id]
	if sb != nil {
		return sb
	}
	return page.pdf.SimpleBoxPool[id]
}

func (page *PDFPage) namedImageBox(id string) *ImageBox {
	tb := page.ImageBoxPool[id]
	if tb != nil {
		return tb
	}
	return page.pdf.ImageBoxPool[id]
}

func (page *PDFPage) namedTextBox(id string) *TextBox {
	tb := page.TextBoxPool[id]
	if tb != nil {
		return tb
	}
	return page.pdf.TextBoxPool[id]
}

type PDF struct {
	Paper           string               // default paper size
	mediaBox        *Rectangle           // default media box
	Crop            string               // default crop box
	cropBox         *Rectangle           // default crop box
	Origin          string               // origin of the coordinate system
	origin          Corner               // one of 4 page corners
	Guides          bool                 // render guides for layouting
	ContentBox      bool                 // render contentBox = cropBox - header - footer
	BackgroundColor string               `json:"bgCol"`
	bgCol           *SimpleColor         // default background color
	Fonts           map[string]*FormFont // default fonts
	InputFont       *FormFont            // default font for input fields
	LabelFont       *FormFont            // default font for input field labels
	fields          StringSet            // input field ids
	fonts           map[string]Resource
	Header          *HorizontalBand
	Footer          *HorizontalBand
	Pages           map[string]*PDFPage
	pages           []*PDFPage
	Margin          *Margin               // the global margin named "margin"
	Border          *Border               // the global border named "border"
	Padding         *Padding              // the global padding named "padding"
	Margins         map[string]*Margin    // global named margins
	Borders         map[string]*Border    // global named borders
	Paddings        map[string]*Padding   // global named paddings
	SimpleBoxPool   map[string]*SimpleBox `json:"boxes"`
	TextBoxPool     map[string]*TextBox   `json:"texts"`
	ImageBoxPool    map[string]*ImageBox  `json:"images"`
	Colors          map[string]string
	colors          map[string]SimpleColor
	FileNames       map[string]string `json:"files"`
	TimestampFormat string            `json:"timestamp"`
	conf            *Configuration
	xRefTable       *XRefTable
}

func (pdf *PDF) pageCount() int {
	return len(pdf.pages)
}

func (pdf *PDF) validate() error {

	// Default paper size
	defaultPaperSize := "A4"

	// Default media box
	pdf.mediaBox = RectForFormat(defaultPaperSize)
	if pdf.Paper != "" {
		dim, _, err := parsePageFormat(pdf.Paper)
		if err != nil {
			return err
		}
		pdf.mediaBox = RectForDim(dim.Width, dim.Height)
	}
	pdf.cropBox = pdf.mediaBox.CroppedCopy(0)

	if pdf.Crop != "" {
		box, err := ParseBox(pdf.Crop, POINTS)
		if err != nil {
			return err
		}
		pdf.cropBox = applyBox("CropBox", box, nil, pdf.mediaBox)
	}

	// Layout coordinate system
	pdf.origin = LowerLeft
	if pdf.Origin != "" {
		corner, err := parseOrigin(pdf.Origin)
		if err != nil {
			return err
		}
		pdf.origin = corner
	}

	// Custom colors
	pdf.colors = map[string]SimpleColor{}
	for n, c := range pdf.Colors {
		if c == "" {
			continue
		}
		sc, err := parseHexColor(c)
		if err != nil {
			return err
		}
		pdf.colors[strings.ToLower(n)] = sc
	}

	// Default background color
	if pdf.BackgroundColor != "" {
		sc, err := pdf.parseColor(pdf.BackgroundColor)
		if err != nil {
			return err
		}
		pdf.bgCol = sc
	}

	// Default fonts
	for _, f := range pdf.Fonts {
		f.pdf = pdf
		if err := f.validate(); err != nil {
			return err
		}
	}

	// Default font for input fields
	if pdf.InputFont != nil {
		pdf.InputFont.pdf = pdf
		if err := pdf.InputFont.validate(); err != nil {
			return err
		}
	}

	// Default font for input field labels
	if pdf.LabelFont != nil {
		pdf.LabelFont.pdf = pdf
		if err := pdf.LabelFont.validate(); err != nil {
			return err
		}
	}

	if pdf.Header != nil {
		if err := pdf.Header.validate(); err != nil {
			return err
		}
		pdf.Header.position = TopCenter
		pdf.Header.pdf = pdf
	}

	if pdf.Footer != nil {
		if err := pdf.Footer.validate(); err != nil {
			return err
		}
		pdf.Footer.position = BottomCenter
		pdf.Footer.pdf = pdf
	}

	if pdf.TimestampFormat == "" {
		pdf.TimestampFormat = pdf.conf.TimestampFormat
	}

	// What follows is a quirky way of turning a map of pages into a sorted slice of pages
	// including entries for pages that are missing in the map.

	var pageNrs []int

	for pageNr, p := range pdf.Pages {
		nr, err := strconv.Atoi(pageNr)
		if err != nil {
			return errors.Errorf("pdfcpu: invalid page number: %s", pageNr)
		}
		pageNrs = append(pageNrs, nr)
		p.number = nr
		p.pdf = pdf
		if err := p.validate(); err != nil {
			return err
		}
	}

	sort.Ints(pageNrs)

	pp := []*PDFPage{}

	maxPageNr := pageNrs[len(pageNrs)-1]
	for i := 1; i <= maxPageNr; i++ {
		pp = append(pp, pdf.Pages[strconv.Itoa(i)])
	}

	pdf.pages = pp

	if pdf.Border != nil {
		if len(pdf.Borders) > 0 {
			return errors.New("pdfcpu: Please supply either \"border\" or \"borders\"")
		}
		pdf.Border.pdf = pdf
		if err := pdf.Border.validate(); err != nil {
			return err
		}
		pdf.Borders = map[string]*Border{}
		pdf.Borders["border"] = pdf.Border
	}
	for _, b := range pdf.Borders {
		b.pdf = pdf
		if err := b.validate(); err != nil {
			return err
		}
	}

	if pdf.Margin != nil {
		if len(pdf.Margins) > 0 {
			return errors.New("pdfcpu: Please supply either \"margin\" or \"margins\"")
		}
		if err := pdf.Margin.validate(); err != nil {
			return err
		}
		pdf.Margins = map[string]*Margin{}
		pdf.Margins["margin"] = pdf.Margin
	}
	for _, m := range pdf.Margins {
		if err := m.validate(); err != nil {
			return err
		}
	}

	if pdf.Padding != nil {
		if len(pdf.Paddings) > 0 {
			return errors.New("pdfcpu: Please supply either \"padding\" or \"paddings\"")
		}
		if err := pdf.Padding.validate(); err != nil {
			return err
		}
		pdf.Paddings = map[string]*Padding{}
		pdf.Paddings["padding"] = pdf.Padding
	}
	for _, p := range pdf.Paddings {
		if err := p.validate(); err != nil {
			return err
		}
	}

	// box templates
	for _, sb := range pdf.SimpleBoxPool {
		sb.pdf = pdf
		if err := sb.validate(); err != nil {
			return err
		}
	}

	// text templates
	for _, tb := range pdf.TextBoxPool {
		tb.pdf = pdf
		if err := tb.validate(); err != nil {
			return err
		}
	}

	// image templates
	for _, ib := range pdf.ImageBoxPool {
		ib.pdf = pdf
		if err := ib.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (pdf *PDF) calcInheritedAttrs() {

	// Calc inherited fonts.
	for id, f0 := range pdf.Fonts {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			f1 := page.Fonts[id]
			if f1 != nil {
				f1.mergeIn(f0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		ff := map[string]*FormFont{}
		for k, v := range pdf.Fonts {
			ff[k] = v
		}
		for k, v := range page.Fonts {
			ff[k] = v
		}
		page.Content.calcFont(ff)
	}

	// Calc inherited margins.
	for id, m0 := range pdf.Margins {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			m1 := page.Margins[id]
			if m1 != nil {
				m1.mergeIn(m0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		mm := map[string]*Margin{}
		for k, v := range pdf.Margins {
			mm[k] = v
		}
		for k, v := range page.Margins {
			mm[k] = v
		}
		page.Content.calcMargin(mm)
	}

	// Calc inherited borders.
	for id, b0 := range pdf.Borders {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			b1 := page.Borders[id]
			if b1 != nil {
				b1.mergeIn(b0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		bb := map[string]*Border{}
		for k, v := range pdf.Borders {
			bb[k] = v
		}
		for k, v := range page.Borders {
			bb[k] = v
		}
		page.Content.calcBorder(bb)
	}

	// Calc inherited paddings.
	for id, p0 := range pdf.Paddings {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			p1 := page.Paddings[id]
			if p1 != nil {
				p1.mergeIn(p0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		pp := map[string]*Padding{}
		for k, v := range pdf.Paddings {
			pp[k] = v
		}
		for k, v := range page.Paddings {
			pp[k] = v
		}
		page.Content.calcPadding(pp)
	}

	// Calc inherited SimpleBoxes.
	for id, sb0 := range pdf.SimpleBoxPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			sb1 := page.SimpleBoxPool[id]
			if sb1 != nil {
				sb1.mergeIn(sb0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		bb := map[string]*SimpleBox{}
		for k, v := range pdf.SimpleBoxPool {
			bb[k] = v
		}
		for k, v := range page.SimpleBoxPool {
			bb[k] = v
		}
		page.Content.calcSimpleBoxes(bb)
	}

	// Calc inherited TextBoxes.
	for id, tb0 := range pdf.TextBoxPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			tb1 := page.TextBoxPool[id]
			if tb1 != nil {
				tb1.mergeIn(tb0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		tb := map[string]*TextBox{}
		for k, v := range pdf.TextBoxPool {
			tb[k] = v
		}
		for k, v := range page.TextBoxPool {
			tb[k] = v
		}
		page.Content.calcTextBoxes(tb)
	}

	// Calc inherited ImageBoxes.
	for id, ib0 := range pdf.ImageBoxPool {
		for _, page := range pdf.pages {
			if page == nil {
				continue
			}
			ib1 := page.ImageBoxPool[id]
			if ib1 != nil {
				ib1.mergeIn(ib0)
			}
		}
	}
	for _, page := range pdf.pages {
		if page == nil {
			continue
		}
		ib := map[string]*ImageBox{}
		for k, v := range pdf.ImageBoxPool {
			ib[k] = v
		}
		for k, v := range page.ImageBoxPool {
			ib[k] = v
		}
		page.Content.calcImageBoxes(ib)
	}

}

func (pdf *PDF) color(s string) *SimpleColor {
	sc, ok := pdf.colors[strings.ToLower(s)]
	if !ok {
		return nil
	}
	return &sc
}

func (pdf *PDF) parseColor(s string) (*SimpleColor, error) {
	sc, err := parseColor(s)
	if err == nil {
		return &sc, nil
	}
	if err != errInvalidColor || s[0] != '$' {
		return nil, err
	}
	sc1 := pdf.color(s[1:])
	if sc1 == nil {
		return nil, errInvalidColor
	}
	return sc1, nil
}

func (pdf *PDF) fileName(s string) (string, error) {
	if s[0] != '$' {
		return s, nil
	}
	fn, ok := pdf.FileNames[s[1:]]
	if !ok {
		return "", errors.Errorf("pdfcpu: invalid named filename: %s", s[1:])
	}
	return fn, nil
}

func (pdf *PDF) calcFont(f *FormFont) error {
	if f.Name[0] != '$' {
		return nil
	}
	fName := f.Name[1:]
	f0 := pdf.Fonts[fName]
	if f0 == nil {
		return errors.Errorf("pdfcpu: unknown font %s", fName)
	}
	f.Name = f0.Name
	if f.Size == 0 {
		f.Size = f0.Size
	}
	if f.col == nil {
		f.col = f0.col
	}
	return nil
}

func (pdf *PDF) renderPages() ([]Page, error) {

	pdf.calcInheritedAttrs()

	pp := []Page{}
	fontMap := FontMap{}
	imageMap := ImageMap{}

	for i, page := range pdf.pages {

		pageNr := i + 1
		mediaBox := pdf.mediaBox
		cropBox := pdf.cropBox

		// TODO Use constructor
		p := Page{
			MediaBox: mediaBox,
			CropBox:  cropBox,
			Fm:       FontMap{},
			Im:       ImageMap{},
			Buf:      new(bytes.Buffer),
		}

		if page == nil {
			// Create blank page with optional background color.
			if pdf.bgCol != nil {
				FillRectNoBorder(p.Buf, p.CropBox, *pdf.bgCol)
			}
			pp = append(pp, p)
			continue
		}

		if page.mediaBox != nil {
			mediaBox = page.mediaBox
		}
		if page.cropBox != nil {
			cropBox = page.cropBox
		}

		// Render page background.
		if page.bgCol == nil {
			page.bgCol = pdf.bgCol
		}
		if page.bgCol != nil {
			FillRectNoBorder(p.Buf, cropBox, *page.bgCol)
		}

		var headerHeight, headerDy float64
		var footerHeight, footerDy float64

		// Render page header.
		if pdf.Header != nil {
			if err := pdf.Header.render(&p, pageNr, true); err != nil {
				return nil, err
			}
			headerHeight = pdf.Header.Height
			headerDy = float64(pdf.Header.Dy)
		}

		// Render page footer.
		if pdf.Footer != nil {
			if err := pdf.Footer.render(&p, pageNr, false); err != nil {
				return nil, err
			}
			footerHeight = pdf.Footer.Height
			footerDy = float64(pdf.Footer.Dy)
		}

		// Render page content.
		r := cropBox.CroppedCopy(0)
		r.LL.Y += footerHeight + footerDy
		r.UR.Y -= headerHeight + headerDy
		page.Content.mediaBox = r
		if err := page.Content.render(&p, pageNr, fontMap, imageMap); err != nil {
			return nil, err
		}

		pp = append(pp, p)
	}

	return pp, nil
}

func (pdf *PDF) addPages(pp []Page) error {

	xRefTable := pdf.xRefTable

	pagesDict := Dict(
		map[string]Object{
			"Type":  Name("Pages"),
			"Count": Integer(len(pp)),
		},
	)

	pagesIndRef, err := xRefTable.IndRefForNewObject(pagesDict)
	if err != nil {
		return err
	}

	kids := Array{}
	fonts := map[string]Resource{}

	for _, p := range pp {

		pageIndRef, _, err := createFormPage(xRefTable, *pagesIndRef, p, fonts)
		if err != nil {
			return err
		}

		kids = append(kids, *pageIndRef)
	}

	// formDict, err := createForm(xRefTable, f, pageIndRef)
	// if err != nil {
	// 	return nil, err
	// }

	//rootDict.Insert("AcroForm", formDict)

	//pageDict.Insert("Annots", f.annots)
	pagesDict.Insert("Kids", kids)

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	rootDict.Insert("Pages", *pagesIndRef)

	return nil
}

func parseFromJSON(bb []byte, conf *Configuration) (*PDF, error) {

	if !json.Valid(bb) {
		return nil, errors.Errorf("pdfcpu: invalid JSON encoding detected.")
	}

	pdf := &PDF{
		fields: StringSet{},
		fonts:  map[string]Resource{},
		Pages:  map[string]*PDFPage{},
		conf:   conf,
	}

	if err := json.Unmarshal(bb, pdf); err != nil {
		return nil, err
	}

	if err := pdf.validate(); err != nil {
		return nil, err
	}

	return pdf, nil
}

func createAnchoredTextbox(text string, a anchor, dx, dy int, font *FormFont, bgCol *SimpleColor, pageNr int, p *Page, pdf *PDF) error {

	fontName := font.Name
	fontSize := font.Size
	col := font.col
	t, _ := resolveWMTextString(text, pdf.TimestampFormat, pageNr, pdf.pageCount())
	k := p.Fm.EnsureKey(fontName)

	if a == TopRight || a == BottomRight {
		dx = -dx
	}

	if a == TopCenter || a == BottomCenter {
		dx = 0
	}

	if a == TopLeft || a == TopCenter || a == TopRight {
		dy = -dy
	}

	td := TextDescriptor{
		Text:     t,
		Dx:       float64(dx),
		Dy:       float64(dy),
		FontName: fontName,
		FontKey:  k,
		FontSize: fontSize,
		Scale:    1.,
		ScaleAbs: true,
		//RTL:      tb.RTL, // for user fonts only!
	}

	if col != nil {
		td.StrokeCol, td.FillCol = *col, *col
	}

	if bgCol != nil {
		td.ShowBackground, td.ShowTextBB, td.BackgroundCol = true, true, *bgCol
	}

	WriteMultiLineAnchored(p.Buf, p.CropBox, nil, td, a)

	return nil
}

// CreateXRefFromJSON creates a PDF XRefTable from the JSON set bb.
func CreateXRefFromJSON(bb []byte, conf *Configuration) (*XRefTable, error) {

	pdf, err := parseFromJSON(bb, conf)
	if err != nil {
		return nil, err
	}

	xRefTable, err := createXRefTableWithRootDict()
	if err != nil {
		return nil, err
	}

	pdf.xRefTable = xRefTable

	pages, err := pdf.renderPages()
	if err != nil {
		return nil, err
	}

	err = pdf.addPages(pages)
	if err != nil {
		return nil, err
	}

	return xRefTable, nil
}

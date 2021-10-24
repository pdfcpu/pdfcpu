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
	"os"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

type ImageData struct {
	Payload       string // base64 encoded image data
	Format        string // jpeg, png, webp, tiff, ccitt
	Width, Height int
}
type ImageBox struct {
	pdf             *PDF
	content         *Content
	Name            string
	FileName        string     `json:"file"` // path of image file name
	Data            *ImageData // TODO
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Dx, Dy          float64
	dest            *pdfcpu.Rectangle
	Anchor          string
	anchor          pdfcpu.Anchor
	anchored        bool
	Width           float64
	Height          float64
	Margin          *Margin
	Border          *Border
	Padding         *Padding
	BackgroundColor string `json:"bgCol"`
	bgCol           *pdfcpu.SimpleColor
	Rotation        float64 `json:"rot"`
	Url             string
	Hide            bool
}

func (ib *ImageBox) resolveFileName(s string) (string, error) {
	if s[0] != '$' {
		return s, nil
	}

	varName := s[1:]
	if ib.content != nil {
		return ib.content.page.resolveFileName(varName)
	}

	return ib.pdf.resolveFileName(varName)
}

func (ib *ImageBox) validate() error {

	ib.x = ib.Position[0]
	ib.y = ib.Position[1]

	if ib.Name == "$" {
		return errors.New("pdfcpu: invalid image reference $")
	}

	// TODO Validate width, height inside content box

	if ib.FileName != "" {
		s, err := ib.resolveFileName(ib.FileName)
		if err != nil {
			return err
		}
		ib.FileName = s
	}

	if ib.Anchor != "" {
		if ib.Position[0] != 0 || ib.Position[1] != 0 {
			return errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
		}
		a, err := pdfcpu.ParseAnchor(ib.Anchor)
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

func (ib *ImageBox) image(pageImages, images pdfcpu.ImageMap, pageNr int) (int, int, string, error) {
	var (
		w, h   int
		id     string
		indRef *pdfcpu.IndirectRef
	)

	img, ok := pageImages[ib.FileName]
	if ok {
		return img.Width, img.Height, img.Res.ID, nil
	}

	imgResIDs := ib.pdf.XObjectResIDs[pageNr]

	img, ok = images[ib.FileName]
	if ok {
		img.Res.ID = "Im" + strconv.Itoa(len(pageImages))
		if ib.pdf.Optimize != nil {
			var id string
			for k, v := range imgResIDs {
				if v == img.Res.IndRef {
					id = k
					break
				}
			}
			if id == "" {
				id = imgResIDs.NewIDForPrefix("Im", len(pageImages))
			}
			img.Res.ID = id
		}
		pageImages[ib.FileName] = img
		return img.Width, img.Height, img.Res.ID, nil
	}

	f, err := os.Open(ib.FileName)
	if err != nil {
		return w, h, id, err
	}
	defer f.Close()

	var sd *pdfcpu.StreamDict

	if ib.pdf.Optimize != nil {

		// PDF file update
		sd, w, h, err = pdfcpu.CreateImageStreamDict(ib.pdf.XRefTable, f, false, false)
		if err != nil {
			return w, h, id, err
		}

		// For each existing image in xRefTable with matching w,h check for byte level identity.
		for objNr, io := range ib.pdf.Optimize.ImageObjects {
			d := io.ImageDict.Dict
			if w != *d.IntEntry("Width") || h != *d.IntEntry("Height") {
				continue
			}
			// compare decoded content from sd and io.ImageDict
			ok, err := pdfcpu.EqualObjects(*sd, *io.ImageDict, ib.pdf.XRefTable)
			if err != nil {
				return w, h, id, err
			}
			if ok {
				// If identical create indRef for objNr
				indRef = pdfcpu.NewIndirectRef(objNr, 0)
				break
			}
		}

		if indRef != nil {
			for k, v := range imgResIDs {
				if v == *indRef {
					id = k
					break
				}
			}
			if id == "" {
				id = imgResIDs.NewIDForPrefix("Im", len(images))
			}
		}

	}

	if indRef == nil {
		if ib.pdf.Optimize != nil {
			indRef, err = ib.pdf.XRefTable.IndRefForNewObject(*sd)
			if err != nil {
				return w, h, id, err
			}
			id = imgResIDs.NewIDForPrefix("Im", len(pageImages))
		} else {
			indRef, w, h, err = pdfcpu.CreateImageResource(ib.pdf.XRefTable, f, false, false)
			if err != nil {
				return w, h, id, err
			}
			id = "Im" + strconv.Itoa(len(pageImages))
		}
	}

	res := pdfcpu.Resource{ID: id, IndRef: indRef}
	img = pdfcpu.ImageResource{Res: res, Width: w, Height: h}
	images[ib.FileName] = img
	pageImages[ib.FileName] = img

	return w, h, id, nil
}

func (ib *ImageBox) createLink(p *pdfcpu.Page, pageNr int, r *pdfcpu.Rectangle, m pdfcpu.Matrix) {

	p1 := m.Transform(pdfcpu.Point{X: r.LL.X, Y: r.LL.Y})
	p2 := m.Transform(pdfcpu.Point{X: r.UR.X, Y: r.LL.X})
	p3 := m.Transform(pdfcpu.Point{X: r.UR.X, Y: r.UR.Y})
	p4 := m.Transform(pdfcpu.Point{X: r.LL.X, Y: r.UR.Y})

	ql := pdfcpu.QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}

	id := fmt.Sprintf("l%d%d", pageNr, len(p.LinkAnnots))
	ann := pdfcpu.NewLinkAnnotation(
		*ql.EnclosingRectangle(5.0),
		pdfcpu.QuadPoints{ql},
		ib.Url,
		id,
		pdfcpu.AnnNoZoom+pdfcpu.AnnNoRotate,
		nil)

	p.LinkAnnots = append(p.LinkAnnots, ann)
}

func (ib *ImageBox) prepareMargin() (float64, float64, float64, float64, error) {

	mTop, mRight, mBot, mLeft := 0., 0., 0., 0.

	if ib.Margin != nil {

		m := ib.Margin
		if m.Name != "" && m.Name[0] == '$' {
			// use named margin
			mName := m.Name[1:]
			m0 := ib.margin(mName)
			if m0 == nil {
				return mTop, mRight, mBot, mLeft, errors.Errorf("pdfcpu: unknown named margin %s", mName)
			}
			m.mergeIn(m0)
		}

		if m.Width > 0 {
			mTop = m.Width
			mRight = m.Width
			mBot = m.Width
			mLeft = m.Width
		} else {
			mTop = m.Top
			mRight = m.Right
			mBot = m.Bottom
			mLeft = m.Left
		}
	}

	return mTop, mRight, mBot, mLeft, nil
}

func (ib *ImageBox) prepareBorder() (float64, *pdfcpu.SimpleColor, pdfcpu.LineJoinStyle, error) {

	bWidth := 0.
	var bCol *pdfcpu.SimpleColor
	bStyle := pdfcpu.LJMiter

	if ib.Border != nil {

		b := ib.Border
		if b.Name != "" && b.Name[0] == '$' {
			// Use named border
			bName := b.Name[1:]
			b0 := ib.border(bName)
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

		if bWidth > 0 && bCol == nil {
			bWidth = 0
		}
	}

	return bWidth, bCol, bStyle, nil
}

func (ib *ImageBox) preparePadding() (float64, float64, float64, float64, error) {

	pTop, pRight, pBot, pLeft := 0., 0., 0., 0.

	if ib.Padding != nil {

		p := ib.Padding
		if p.Name != "" && p.Name[0] == '$' {
			// use named padding
			pName := p.Name[1:]
			p0 := ib.padding(pName)
			if p0 == nil {
				return pTop, pRight, pBot, pLeft, errors.Errorf("pdfcpu: unknown named padding %s", pName)
			}
			p.mergeIn(p0)
		}

		pTop, pRight, pBot, pLeft = p.Top, p.Right, p.Bottom, p.Left
		if p.Width > 0 {
			pTop, pRight, pBot, pLeft = p.Width, p.Width, p.Width, p.Width
		}

	}

	return pTop, pRight, pBot, pLeft, nil
}

func (ib *ImageBox) render(p *pdfcpu.Page, pageNr int, images pdfcpu.ImageMap) error {

	mTop, mRight, mBot, mLeft, err := ib.prepareMargin()
	if err != nil {
		return err
	}

	bWidth, bCol, bStyle, err := ib.prepareBorder()
	if err != nil {
		return err
	}

	pTop, pRight, pBot, pLeft, err := ib.preparePadding()
	if err != nil {
		return err
	}

	w, h, id, err := ib.image(p.Im, images, pageNr)
	if err != nil {
		return err
	}

	rSrc := pdfcpu.RectForDim(float64(w), float64(h))

	cBox := ib.dest
	if ib.content != nil {
		cBox = ib.content.Box()
	}
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBot
	r.UR.X -= mRight
	r.UR.Y -= mTop

	if ib.Width == 0 && ib.Height == 0 {
		if rSrc.Width() <= r.Width() && rSrc.Height() <= r.Height() {
			ib.Width = rSrc.Width()
			ib.Height = rSrc.Height()
		} else {
			ib.Height = r.Height()
			ib.Width = rSrc.ScaledWidth(ib.Height-2*bWidth-pTop-pBot) + 2*bWidth + pLeft + pRight
		}
	} else if ib.Width == 0 {
		ib.Width = rSrc.ScaledWidth(ib.Height-2*bWidth-pTop-pBot) + 2*bWidth + pLeft + pRight
	} else if ib.Height == 0 {
		ib.Height = rSrc.ScaledHeight(ib.Width-2*bWidth-pLeft-pRight) + 2*bWidth + pTop + pBot
	}

	var x, y float64
	if ib.anchored {
		x, y = pdfcpu.AnchorPosition(ib.anchor, r, ib.Width, ib.Height)
	} else {
		x, y = pdfcpu.NormalizeCoord(ib.x, ib.y, cBox, ib.pdf.origin, false)
		if y < 0 {
			y = cBox.Center().Y - ib.Height/2 - r.LL.Y
		} else if y > 0 {
			y -= mBot
		}
		if x < 0 {
			x = cBox.Center().X - ib.Width/2 - r.LL.X
		} else if x > 0 {
			x -= mLeft
		}
	}

	dx, dy := pdfcpu.NormalizeOffset(ib.Dx, ib.Dy, ib.pdf.origin)
	x += r.LL.X + dx
	y += r.LL.Y + dy

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

	r = pdfcpu.RectForWidthAndHeight(x, y, ib.Width, ib.Height)
	r.LL.X += bWidth / 2
	r.LL.Y += bWidth / 2
	r.UR.X -= bWidth / 2
	r.UR.Y -= bWidth / 2

	if bCol == nil {
		bCol = &pdfcpu.Black
	}

	sin := math.Sin(float64(ib.Rotation) * float64(pdfcpu.DegToRad))
	cos := math.Cos(float64(ib.Rotation) * float64(pdfcpu.DegToRad))

	dx = r.LL.X
	dy = r.LL.Y
	r.Translate(-r.LL.X, -r.LL.Y)

	dx += ib.Dx + r.Width()/2 + sin*(r.Height()/2) - cos*r.Width()/2
	dy += ib.Dy + r.Height()/2 - cos*(r.Height()/2) - sin*r.Width()/2

	m := pdfcpu.CalcTransformMatrix(1, 1, sin, cos, dx, dy)
	fmt.Fprintf(p.Buf, "q %.2f %.2f %.2f %.2f %.2f %.2f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	if ib.Url != "" {
		ib.createLink(p, pageNr, r, m)
	}

	// Render border
	if ib.bgCol != nil {
		if bWidth == 0 {
			bCol = ib.bgCol
		}
		pdfcpu.FillRect(p.Buf, r, bWidth, bCol, *ib.bgCol, &bStyle)
	} else if ib.Border != nil {
		pdfcpu.DrawRect(p.Buf, r, bWidth, bCol, &bStyle)
	}
	fmt.Fprint(p.Buf, "Q ")

	// Render image
	rDest := pdfcpu.RectForWidthAndHeight(x+bWidth+pLeft, y+bWidth+pBot, ib.Width-2*bWidth-pLeft-pRight, ib.Height-2*bWidth-pTop-pBot)
	sx, sy, dx, dy, _ := pdfcpu.BestFitRectIntoRect(rSrc, rDest, false)
	dx += rDest.LL.X
	dy += rDest.LL.Y

	dx += ib.Dx + sx/2 + sin*(sy/2) - cos*sx/2
	dy += ib.Dy + sy/2 - cos*(sy/2) - sin*sx/2

	m = pdfcpu.CalcTransformMatrix(sx, sy, sin, cos, dx, dy)
	fmt.Fprintf(p.Buf, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s Do Q ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], id)

	return nil
}

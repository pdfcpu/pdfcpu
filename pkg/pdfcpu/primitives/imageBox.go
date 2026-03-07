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
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ImageData represents a more direct way for providing image data for form filling scenarios.
type ImageData struct {
	Payload       string // base64 encoded image data
	Format        string // jpeg, png, webp, tiff, ccitt
	Width, Height int
}

// ImageBox is a rectangular region within content containing an image.
type ImageBox struct {
	pdf             *PDF
	content         *Content
	Name            string
	Src             string     `json:"src"` // path of image file name
	Data            *ImageData // TODO Implement
	Position        [2]float64 `json:"pos"` // x,y
	x, y            float64
	Dx, Dy          float64
	dest            *types.Rectangle
	Anchor          string
	anchor          types.Anchor
	anchored        bool
	Width           float64
	Height          float64
	Margin          *Margin
	Border          *Border
	Padding         *Padding
	BackgroundColor string `json:"bgCol"`
	bgCol           *color.SimpleColor
	Rotation        float64 `json:"rot"`
	Url             string
	Hide            bool
	PageNr          string `json:"-"`
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

func (ib *ImageBox) parseAnchor() (types.Anchor, error) {
	if ib.Position[0] != 0 || ib.Position[1] != 0 {
		var a types.Anchor
		return a, errors.New("pdfcpu: Please supply \"pos\" or \"anchor\"")
	}
	return types.ParseAnchor(ib.Anchor)
}

func (ib *ImageBox) validate() error {

	ib.x = ib.Position[0]
	ib.y = ib.Position[1]

	if ib.Name == "$" {
		return errors.New("pdfcpu: invalid image reference $")
	}

	// TODO Validate width, height inside content box

	if ib.Src != "" {
		s, err := ib.resolveFileName(ib.Src)
		if err != nil {
			return err
		}
		ib.Src = s
	}

	if ib.Anchor != "" {
		a, err := ib.parseAnchor()
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

func (ib *ImageBox) missingPosition() bool {
	return ib.x == 0 && ib.y == 0
}

func (ib *ImageBox) mergeIn(ib0 *ImageBox) {

	if !ib.anchored && ib.missingPosition() {
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

	if ib.Height == 0 {
		ib.Height = ib0.Height
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

	if ib.Src == "" && ib.Data == nil {
		ib.Src = ib0.Src
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

func (ib *ImageBox) cachedImg(img model.ImageResource, pageImages model.ImageMap, pageNr int) (int, int, string, error) {
	imgResIDs := ib.pdf.XObjectResIDs[pageNr]
	img.Res.ID = "Im" + strconv.Itoa(len(pageImages))
	if ib.pdf.Update() {
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
	pageImages[ib.Src] = img

	return img.Width, img.Height, img.Res.ID, nil
}

func (ib *ImageBox) checkForExistingImage(sd *types.StreamDict, w, h int) (*types.IndirectRef, error) {
	// For each existing image in xRefTable with matching w,h check for byte level identity.
	for objNr, io := range ib.pdf.Optimize.ImageObjects {
		d := io.ImageDict.Dict
		if w != *d.IntEntry("Width") || h != *d.IntEntry("Height") {
			continue
		}
		// compare decoded content from sd and io.ImageDict
		ok, err := model.EqualObjects(*sd, *io.ImageDict, ib.pdf.XRefTable, nil)
		if err != nil {
			return nil, err
		}
		if ok {
			// If identical create indRef for objNr
			return types.NewIndirectRef(objNr, 0), nil
		}
	}
	return nil, nil
}

func (ib *ImageBox) resource() (io.ReadCloser, error) {
	pdf := ib.pdf
	var f io.ReadCloser
	if strings.HasPrefix(ib.Src, "http") {
		if pdf.Offline {
			if log.CLIEnabled() {
				log.CLI.Printf("pdfcpu is offline, can't get %s\n", ib.Src)
			}
			return nil, nil
		}
		client := pdf.httpClient
		if client == nil {
			pdf.httpClient = &http.Client{
				Timeout: time.Duration(pdf.Timeout) * time.Second,
			}
			client = pdf.httpClient
		}
		resp, err := client.Get(ib.Src)
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				if log.CLIEnabled() {
					log.CLI.Printf("timeout: %s\n", ib.Src)
				}
			} else {
				if log.CLIEnabled() {
					log.CLI.Printf("%v: %s\n", err, ib.Src)
				}
			}
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			if log.CLIEnabled() {
				log.CLI.Printf("http status %d: %s\n", resp.StatusCode, ib.Src)
			}
			return nil, nil
		}
		f = resp.Body
	} else {
		var err error
		f, err = os.Open(ib.Src)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func (ib *ImageBox) imageResource(pageImages, images model.ImageMap, pageNr int) (*model.ImageResource, error) {

	f, err := ib.resource()
	if err != nil || f == nil {
		return nil, err
	}

	defer f.Close()

	var (
		w, h   int
		id     string
		indRef *types.IndirectRef
		sd     *types.StreamDict
	)

	pdf := ib.pdf
	imgResIDs := pdf.XObjectResIDs[pageNr]

	if ib.pdf.Update() {

		sd, w, h, err = model.CreateImageStreamDict(pdf.XRefTable, f)
		if err != nil {
			return nil, err
		}

		indRef, err := ib.checkForExistingImage(sd, w, h)
		if err != nil {
			return nil, err
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
		if pdf.Update() {
			indRef, err = pdf.XRefTable.IndRefForNewObject(*sd)
			if err != nil {
				return nil, err
			}
			id = imgResIDs.NewIDForPrefix("Im", len(pageImages))
		} else {
			indRef, w, h, err = model.CreateImageResource(pdf.XRefTable, f)
			if err != nil {
				return nil, err
			}
			id = "Im" + strconv.Itoa(len(pageImages))
		}
	}

	res := model.Resource{ID: id, IndRef: indRef}

	return &model.ImageResource{Res: res, Width: w, Height: h}, nil
}

func (ib *ImageBox) image(pageImages, images model.ImageMap, pageNr int) (int, int, string, error) {

	img, ok := pageImages[ib.Src]
	if ok {
		return img.Width, img.Height, img.Res.ID, nil
	}

	img, ok = images[ib.Src]
	if ok {
		return ib.cachedImg(img, pageImages, pageNr)
	}

	imgRes, err := ib.imageResource(pageImages, images, pageNr)
	if err != nil || imgRes == nil {
		return 0, 0, "", nil
	}

	images[ib.Src] = *imgRes
	pageImages[ib.Src] = *imgRes

	return imgRes.Width, imgRes.Height, imgRes.Res.ID, nil
}

func (ib *ImageBox) createLink(p *model.Page, pageNr int, r *types.Rectangle, m matrix.Matrix) {

	p1 := m.Transform(types.Point{X: r.LL.X, Y: r.LL.Y})
	p2 := m.Transform(types.Point{X: r.UR.X, Y: r.LL.X})
	p3 := m.Transform(types.Point{X: r.UR.X, Y: r.UR.Y})
	p4 := m.Transform(types.Point{X: r.LL.X, Y: r.UR.Y})

	ql := types.QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}

	id := fmt.Sprintf("l%d%d", pageNr, len(p.LinkAnnots))
	ann := model.NewLinkAnnotation(
		*ql.EnclosingRectangle(5.0), // rect
		0,                           // apObjNr
		"",                          // contents
		id,                          // id
		"",                          // modDate
		0,                           // f
		&color.Red,                  // borderCol
		nil,                         // dest
		ib.Url,                      // uri
		types.QuadPoints{ql},        // quad
		false,                       // border
		0,                           // borderWidth
		model.BSSolid,               // borderStyle
	)

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

func (ib *ImageBox) prepareBorder() (float64, *color.SimpleColor, types.LineJoinStyle, error) {

	bWidth := 0.
	var bCol *color.SimpleColor
	bStyle := types.LJMiter

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

func (ib *ImageBox) calcDim(rSrc, r *types.Rectangle, bWidth, pTop, pBot, pLeft, pRight float64) {
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
}

func (ib *ImageBox) calcTransform(
	mLeft, mBot, mRight, mTop,
	pLeft, pBot, pRight, pTop,
	bWidth float64, rSrc *types.Rectangle) (matrix.Matrix, float64, float64, float64, float64, *types.Rectangle) {

	cBox := ib.dest
	if ib.content != nil {
		cBox = ib.content.Box()
	}
	r := cBox.CroppedCopy(0)
	r.LL.X += mLeft
	r.LL.Y += mBot
	r.UR.X -= mRight
	r.UR.Y -= mTop

	ib.calcDim(rSrc, r, bWidth, pTop, pBot, pLeft, pRight)

	var x, y float64
	if ib.anchored {
		x, y = types.AnchorPosition(ib.anchor, r, ib.Width, ib.Height)
	} else {
		x, y = types.NormalizeCoord(ib.x, ib.y, cBox, ib.pdf.origin, false)
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

	dx, dy := types.NormalizeOffset(ib.Dx, ib.Dy, ib.pdf.origin)
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

	r = types.RectForWidthAndHeight(x, y, ib.Width, ib.Height)
	r.LL.X += bWidth / 2
	r.LL.Y += bWidth / 2
	r.UR.X -= bWidth / 2
	r.UR.Y -= bWidth / 2

	sin := math.Sin(float64(ib.Rotation) * float64(matrix.DegToRad))
	cos := math.Cos(float64(ib.Rotation) * float64(matrix.DegToRad))

	dx = r.LL.X
	dy = r.LL.Y
	r.Translate(-r.LL.X, -r.LL.Y)

	dx += r.Width()/2 + sin*(r.Height()/2) - cos*r.Width()/2
	dy += r.Height()/2 - cos*(r.Height()/2) - sin*r.Width()/2

	m := matrix.CalcTransformMatrix(1, 1, sin, cos, dx, dy)

	return m, x, y, sin, cos, r
}

func (ib *ImageBox) render(p *model.Page, pageNr int, images model.ImageMap) error {

	mTop, mRight, mBot, mLeft, err := ib.prepareMargin()
	if err != nil {
		return err
	}

	bWidth, bCol, bStyle, err := ib.prepareBorder()
	if err != nil {
		return err
	}
	if bCol == nil {
		bCol = &color.Black
	}

	pTop, pRight, pBot, pLeft, err := ib.preparePadding()
	if err != nil {
		return err
	}

	w, h, id, err := ib.image(p.Im, images, pageNr)
	if err != nil {
		return err
	}

	missingImg := w == 0 && h == 0
	if missingImg {
		w = 200
	}

	rSrc := types.RectForDim(float64(w), float64(h))

	m, x, y, sin, cos, r := ib.calcTransform(mLeft, mBot, mRight, mTop, pLeft, pBot, pRight, pTop, bWidth, rSrc)

	fmt.Fprintf(p.Buf, "q %.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	if ib.Url != "" {
		ib.createLink(p, pageNr, r, m)
	}

	// Render border
	if ib.bgCol != nil {
		if bWidth == 0 {
			bCol = ib.bgCol
		}
		draw.FillRect(p.Buf, r, bWidth, bCol, *ib.bgCol, &bStyle)
	} else if ib.Border != nil {
		draw.DrawRect(p.Buf, r, bWidth, bCol, &bStyle)
	}
	if ib.pdf.Debug {
		draw.DrawCircle(p.Buf, r.LL.X, r.LL.Y, 5, color.Black, &color.Red)
	}
	fmt.Fprint(p.Buf, "Q ")

	if !missingImg {
		// Render image
		rDest := types.RectForWidthAndHeight(x+bWidth+pLeft, y+bWidth+pBot, ib.Width-2*bWidth-pLeft-pRight, ib.Height-2*bWidth-pTop-pBot)
		sx, sy, dx, dy, _ := types.BestFitRectIntoRect(rSrc, rDest, false, false)
		dx += rDest.LL.X
		dy += rDest.LL.Y

		dx += sx/2 + sin*(sy/2) - cos*sx/2
		dy += sy/2 - cos*(sy/2) - sin*sx/2

		m = matrix.CalcTransformMatrix(sx, sy, sin, cos, dx, dy)
		fmt.Fprintf(p.Buf, "q %.5f %.5f %.5f %.5f %.5f %.5f cm /%s Do Q ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], id)
	}

	return nil
}

// RenderForFill renders ib during form filling.
func (ib *ImageBox) RenderForFill(pdf *PDF, p *model.Page, pageNr int, imageMap model.ImageMap) error {

	ib.pdf = pdf

	if err := ib.validate(); err != nil {
		return err
	}

	ib.dest = p.CropBox

	return ib.render(p, pageNr, imageMap)
}

/*
Copyright 2022 The pdfcpu Authors.

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

package model

import (
	"fmt"
	"io"
	"math"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	DegToRad = math.Pi / 180
	RadToDeg = 180 / math.Pi
)

// Rotation along one of 2 diagonals
const (
	NoDiagonal = iota
	DiagonalLLToUR
	DiagonalULToLR
)

// Watermark mode
const (
	WMText = iota
	WMImage
	WMPDF
)

type formCache map[types.Rectangle]*types.IndirectRef

type PdfResources struct {
	Content []byte
	ResDict *types.IndirectRef
	Bb      *types.Rectangle // visible region in user space
}

// Watermark represents the basic structure and command details for the commands "Stamp" and "Watermark".
type Watermark struct {
	// configuration
	Mode              int                 // WMText, WMImage or WMPDF
	TextString        string              // raw display text.
	TextLines         []string            // display multiple lines of text.
	URL               string              // overlay link annotation for stamps.
	FileName          string              // display pdf page or png image.
	Image             io.Reader           // reader for image watermark.
	Page              int                 // the page number of a PDF file. 0 means multistamp/multiwatermark.
	OnTop             bool                // if true this is a STAMP else this is a WATERMARK.
	InpUnit           types.DisplayUnit   // input display unit.
	Pos               types.Anchor        // position anchor, one of tl,tc,tr,l,c,r,bl,bc,br.
	Dx, Dy            float64             // anchor offset.
	HAlign            *types.HAlignment   // horizonal alignment for text watermarks.
	FontName          string              // supported are Adobe base fonts only. (as of now: Helvetica, Times-Roman, Courier)
	FontSize          int                 // font scaling factor.
	ScaledFontSize    int                 // font scaling factor for a specific page
	RTL               bool                // if true, render text from right to left
	Color             color.SimpleColor   // text fill color(=non stroking color) for backwards compatibility.
	FillColor         color.SimpleColor   // text fill color(=non stroking color).
	StrokeColor       color.SimpleColor   // text stroking color
	BgColor           *color.SimpleColor  // text bounding box background color
	MLeft, MRight     float64             // left and right bounding box margin
	MTop, MBot        float64             // top and bottom bounding box margin
	BorderWidth       float64             // Border width, visible if BgColor is set.
	BorderStyle       types.LineJoinStyle // Border style (bounding box corner style), visible if BgColor is set.
	BorderColor       *color.SimpleColor  // border color
	Rotation          float64             // rotation to apply in degrees. -180 <= x <= 180
	Diagonal          int                 // paint along the diagonal.
	UserRotOrDiagonal bool                // true if one of rotation or diagonal provided overriding the default.
	Opacity           float64             // opacity of the watermark. 0 <= x <= 1
	RenderMode        draw.RenderMode     // fill=0, stroke=1 fill&stroke=2
	Scale             float64             // relative scale factor: 0 <= x <= 1, absolute scale factor: 0 <= x
	ScaleEff          float64             // effective scale factor
	ScaleAbs          bool                // true for absolute scaling.
	Update            bool                // true for updating instead of adding a page watermark.

	// resources
	Ocg, ExtGState, Font, Img *types.IndirectRef

	// image or PDF watermark
	Width, Height int // image or page dimensions.
	bbPDF         *types.Rectangle

	// PDF watermark
	PdfRes map[int]PdfResources

	// page specific
	Bb      *types.Rectangle   // bounding box of the form representing this watermark.
	BbTrans types.QuadLiteral  // Transformed bounding box.
	Vp      *types.Rectangle   // page dimensions.
	PageRot int                // page rotation in effect.
	Form    *types.IndirectRef // Forms are dependent on given page dimensions.

	// house keeping
	Objs   types.IntSet // objects for which wm has been applied already.
	FCache formCache    // form cache.
}

// DefaultWatermarkConfig returns the default configuration.
func DefaultWatermarkConfig() *Watermark {
	return &Watermark{
		Page:        0,
		FontName:    "Helvetica",
		FontSize:    24,
		RTL:         false,
		Pos:         types.Center,
		Scale:       0.5,
		ScaleAbs:    false,
		Color:       color.Gray,
		StrokeColor: color.Gray,
		FillColor:   color.Gray,
		Diagonal:    DiagonalLLToUR,
		Opacity:     1.0,
		RenderMode:  draw.RMFill,
		PdfRes:      map[int]PdfResources{},
		Objs:        types.IntSet{},
		FCache:      formCache{},
		TextLines:   []string{},
	}
}

// Recycle resets all caches.
func (wm *Watermark) Recycle() {
	wm.Objs = types.IntSet{}
	wm.FCache = formCache{}
}

// IsText returns true if the watermark content is text.
func (wm Watermark) IsText() bool {
	return wm.Mode == WMText
}

// IsPDF returns true if the watermark content is PDF.
func (wm Watermark) IsPDF() bool {
	return wm.Mode == WMPDF
}

// IsImage returns true if the watermark content is an image.
func (wm Watermark) IsImage() bool {
	return wm.Mode == WMImage
}

// Typ returns the nature of wm.
func (wm Watermark) Typ() string {
	if wm.IsImage() {
		return "image"
	}
	if wm.IsPDF() {
		return "pdf"
	}
	return "text"
}

func (wm Watermark) String() string {
	var s string
	if !wm.OnTop {
		s = "not "
	}

	t := wm.TextString
	if len(t) == 0 {
		t = wm.FileName
	}

	sc := "relative"
	if wm.ScaleAbs {
		sc = "absolute"
	}

	bbox := ""
	if wm.Bb != nil {
		bbox = (*wm.Bb).String()
	}

	vp := ""
	if wm.Vp != nil {
		vp = (*wm.Vp).String()
	}

	return fmt.Sprintf("Watermark: <%s> is %son top, typ:%s\n"+
		"%s %d points\n"+
		"PDFpage#: %d\n"+
		"scaling: %.1f %s\n"+
		"color: %s\n"+
		"rotation: %.1f\n"+
		"diagonal: %d\n"+
		"opacity: %.1f\n"+
		"renderMode: %d\n"+
		"bbox:%s\n"+
		"vp:%s\n"+
		"pageRotation: %d\n",
		t, s, wm.Typ(),
		wm.FontName, wm.FontSize,
		wm.Page,
		wm.Scale, sc,
		wm.Color,
		wm.Rotation,
		wm.Diagonal,
		wm.Opacity,
		wm.RenderMode,
		bbox,
		vp,
		wm.PageRot,
	)
}

// OnTopString returns "watermark" or "stamp" whichever applies.
func (wm Watermark) OnTopString() string {
	s := "watermark"
	if wm.OnTop {
		s = "stamp"
	}
	return s
}

// MultiStamp returns true if wm is a multi stamp.
func (wm Watermark) MultiStamp() bool {
	return wm.Page == 0
}

// CalcBoundingBox returns the bounding box for wm and pageNr.
func (wm *Watermark) CalcBoundingBox(pageNr int) {
	bb := types.RectForDim(float64(wm.Width), float64(wm.Height))

	if wm.IsPDF() {
		wm.bbPDF = wm.PdfRes[wm.Page].Bb
		if wm.MultiStamp() {
			i := pageNr
			if i > len(wm.PdfRes) {
				i = len(wm.PdfRes)
			}
			wm.bbPDF = wm.PdfRes[i].Bb
		}
		wm.Width = int(wm.bbPDF.Width())
		wm.Height = int(wm.bbPDF.Height())
		bb = wm.bbPDF.CroppedCopy(0)
	}

	ar := bb.AspectRatio()

	if wm.ScaleAbs {
		w1 := wm.Scale * bb.Width()
		bb.UR.X = bb.LL.X + w1
		bb.UR.Y = bb.LL.Y + w1/ar
		wm.Bb = bb
		wm.ScaleEff = wm.Scale
		return
	}

	if ar >= 1 {
		// Landscape
		w1 := wm.Scale * wm.Vp.Width()
		bb.UR.X = bb.LL.X + w1
		bb.UR.Y = bb.LL.Y + w1/ar
		wm.ScaleEff = w1 / float64(wm.Width)
	} else {
		// Portrait
		h1 := wm.Scale * wm.Vp.Height()
		bb.UR.Y = bb.LL.Y + h1
		bb.UR.X = bb.LL.X + h1*ar
		wm.ScaleEff = h1 / float64(wm.Height)
	}

	wm.Bb = bb
}

// LowerLeftCorner returns the lower left corner for a bounding box anchored onto vp.
func LowerLeftCorner(vp *types.Rectangle, bbw, bbh float64, a types.Anchor) types.Point {

	var p types.Point
	vpw := vp.Width()
	vph := vp.Height()

	switch a {

	case types.TopLeft:
		p.X = vp.LL.X
		p.Y = vp.UR.Y - bbh

	case types.TopCenter:
		p.X = vp.LL.X + (vpw/2 - bbw/2)
		p.Y = vp.UR.Y - bbh

	case types.TopRight:
		p.X = vp.UR.X - bbw
		p.Y = vp.UR.Y - bbh

	case types.Left:
		p.X = vp.LL.X
		p.Y = vp.LL.Y + (vph/2 - bbh/2)

	case types.Center:
		p.X = vp.LL.X + (vpw/2 - bbw/2)
		p.Y = vp.LL.Y + (vph/2 - bbh/2)

	case types.Right:
		p.X = vp.UR.X - bbw
		p.Y = vp.LL.Y + (vph/2 - bbh/2)

	case types.BottomLeft:
		p.X = vp.LL.X
		p.Y = vp.LL.Y

	case types.BottomCenter:
		p.X = vp.LL.X + (vpw/2 - bbw/2)
		p.Y = vp.LL.Y

	case types.BottomRight:
		p.X = vp.UR.X - bbw
		p.Y = vp.LL.Y
	}

	return p
}

func (wm *Watermark) alignWithPageBoundariesForNegRot() (float64, float64) {
	w, h := wm.Bb.Width(), wm.Bb.Height()
	var dx, dy float64

	switch wm.Pos {

	case types.TopLeft:
		dx, dy = 0, h

	case types.TopCenter:
		dx, dy = (w-h)/2, h

	case types.TopRight:
		dx, dy = w-h, h

	case types.Left:
		dx, dy = 0, (w+h)/2

	case types.Right:
		dx, dy = w-h, (w+h)/2

	case types.BottomLeft:
		dx, dy = 0, w

	case types.BottomCenter:
		dx, dy = (w-h)/2, w

	case types.BottomRight:
		dx, dy = w-h, w
	}

	return dx, dy
}

func (wm *Watermark) alignWithPageBoundariesForPosRot() (float64, float64) {
	w, h := wm.Bb.Width(), wm.Bb.Height()
	var dx, dy float64

	switch wm.Pos {

	case types.TopLeft:
		dx, dy = h, h-w

	case types.TopCenter:
		dx, dy = (w+h)/2, h-w

	case types.TopRight:
		dx, dy = w, h-w

	case types.Left:
		dx, dy = h, (h-w)/2

	case types.Right:
		dx, dy = w, (h-w)/2

	case types.BottomLeft:
		dx, dy = h, 0

	case types.BottomCenter:
		dx, dy = (w+h)/2, 0

	case types.BottomRight:
		dx, dy = w, 0

	}

	return dx, dy
}

func (wm *Watermark) alignWithPageBoundaries() (float64, float64) {
	if wm.Rotation == 90 {
		return wm.alignWithPageBoundariesForPosRot()
	}
	// wm.Rotation == -90
	return wm.alignWithPageBoundariesForNegRot()
}

// CalcTransformMatrix return the transform matrix for a watermark.
func (wm *Watermark) CalcTransformMatrix() matrix.Matrix {
	var sin, cos float64
	r := wm.Rotation

	if wm.Diagonal != NoDiagonal {

		// Calculate the angle of the diagonal with respect of the aspect ratio of the bounding box.
		r = math.Atan(wm.Vp.Height()/wm.Vp.Width()) * float64(RadToDeg)

		if wm.Bb.AspectRatio() < 1 {
			r -= 90
		}

		if wm.Diagonal == DiagonalULToLR {
			r = -r
		}

	}

	sin = math.Sin(float64(r) * float64(DegToRad))
	cos = math.Cos(float64(r) * float64(DegToRad))

	var dx, dy float64
	if !wm.IsImage() && !wm.IsPDF() {
		dy = wm.Bb.LL.Y
	}

	ll := LowerLeftCorner(wm.Vp, wm.Bb.Width(), wm.Bb.Height(), wm.Pos)

	if wm.Pos != types.Center && (r == 90 || r == -90) {
		dx, dy = wm.alignWithPageBoundaries()
		dx = ll.X + dx + wm.Dx
		dy = ll.Y + dy + wm.Dy
	} else {
		dx = ll.X + wm.Bb.Width()/2 + wm.Dx + sin*(wm.Bb.Height()/2+dy) - cos*wm.Bb.Width()/2
		dy = ll.Y + wm.Bb.Height()/2 + wm.Dy - cos*(wm.Bb.Height()/2+dy) - sin*wm.Bb.Width()/2
	}

	return matrix.CalcTransformMatrix(1, 1, sin, cos, dx, dy)
}

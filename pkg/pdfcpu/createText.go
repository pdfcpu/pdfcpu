/*
Copyright 2020 The pdfcpu Authors.

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
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/types"
	"golang.org/x/text/unicode/norm"
)

// HAlignment represents the horizontal alignment of text.
type HAlignment int

// These are the options for horizontal aligned text.
const (
	AlignLeft HAlignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// VAlignment represents the vertical alignment of text.
type VAlignment int

// These are the options for vertical aligned text.
const (
	AlignBaseline VAlignment = iota
	AlignTop
	AlignMiddle
	AlignBottom
)

// LineJoinStyle represents the shape to be used at the corners of paths that are stroked (see 8.4.3.4)
type LineJoinStyle int

// Render mode
const (
	LJMiter LineJoinStyle = iota
	LJRound
	LJBevel
)

// TextDescriptor contains all attributes needed for rendering a text column in PDF user space.
type TextDescriptor struct {
	Text           string        // A multi line string using \n for line breaks.
	FontName       string        // Name of the core or user font to be used.
	RTL            bool          // Right to left user font.
	FontKey        string        // Resource id registered for FontName.
	FontSize       int           // Fontsize in points.
	X, Y           float64       // Position of first char's baseline.
	Dx, Dy         float64       // Horizontal and vertical offsets for X,Y.
	MTop, MBot     float64       // Top and bottom margins applied to text bounding box.
	MLeft, MRight  float64       // Left and right margins applied to text bounding box.
	MinHeight      float64       // The minimum height of this text's bounding box.
	Rotation       float64       // 0..360 degree rotation angle.
	ScaleAbs       bool          // Scaling type, true=absolute, false=relative to container dimensions.
	Scale          float64       // font scaling factor > 0 (and <= 1 for relative scaling).
	HAlign         HAlignment    // Horizontal text alignment.
	VAlign         VAlignment    // Vertical text alignment.
	RMode          RenderMode    // Text render mode
	StrokeCol      SimpleColor   // Stroke color to be used for rendering text corresponding to RMode.
	FillCol        SimpleColor   // Fill color to be used for rendering text corresponding to RMode.
	ShowTextBB     bool          // Render bounding box including BackgroundCol, border and margins.
	ShowBackground bool          // Render background of bounding box using BackgroundCol.
	BackgroundCol  SimpleColor   // Bounding box fill color.
	ShowBorder     bool          // Render border using BorderCol, BorderWidth and BorderStyle.
	BorderWidth    float64       // Border width, visibility depends on ShowBorder.
	BorderStyle    LineJoinStyle // Border style, also visible if ShowBorder is false as long as ShowBackground is true.
	BorderCol      SimpleColor   // Border color.
	ParIndent      bool          // Indent first line of paragraphs or space between paragraphs.
	ShowLineBB     bool          // Render line bounding boxes in black (for HAlign != AlignJustify only)
	ShowMargins    bool          // Render margins in light gray.
	HairCross      bool          // Draw haircross at X,Y.
}

type Resource struct {
	ID     string
	IndRef *IndirectRef
}

// FontResource represents an existing PDF font resource.
type FontResource struct {
	Res       Resource
	CIDSet    *IndirectRef
	FontFile  *IndirectRef
	ToUnicode *IndirectRef
	W         *IndirectRef
}

// FontMap maps font names to font resources.
type FontMap map[string]FontResource

// EnsureKey registers fontName with corresponding font resource id.
func (fm FontMap) EnsureKey(fontName string) string {
	// TODO userfontname prefix
	for k, v := range fm {
		if k == fontName {
			return v.Res.ID
		}
	}
	key := "F" + strconv.Itoa(len(fm))
	fm[fontName] = FontResource{Res: Resource{ID: key}}
	return key
}

// ImageResource represents an existing PDF image resource.
type ImageResource struct {
	Res    Resource
	Width  int
	Height int
}

// ImageMap maps image filenames to image resources.
type ImageMap map[string]ImageResource

// Page represents rendered page content.
type Page struct {
	MediaBox     *Rectangle
	CropBox      *Rectangle
	Fm           FontMap
	Im           ImageMap
	AnnotIndRefs Array
	Annots       []Dict
	LinkAnnots   []LinkAnnotation
	Buf          *bytes.Buffer
}

// NewPage creates a page for a mediaBox.
func NewPage(mediaBox *Rectangle) Page {
	return Page{MediaBox: mediaBox, Fm: FontMap{}, Im: ImageMap{}, Buf: new(bytes.Buffer)}
}

// NewPageWithBg creates a page for a mediaBox.
func NewPageWithBg(mediaBox *Rectangle, c SimpleColor) Page {
	p := Page{MediaBox: mediaBox, Fm: FontMap{}, Im: ImageMap{}, Buf: new(bytes.Buffer)}
	FillRectNoBorder(p.Buf, mediaBox, c)
	return p
}

func deltaAlignMiddle(fontName string, fontSize, lines int, mTop, mBot float64) float64 {
	return -font.Ascent(fontName, fontSize) + (float64(lines)*font.LineHeight(fontName, fontSize)+mTop+mBot)/2 - mTop
}

func deltaAlignTop(fontName string, fontSize int, mTop float64) float64 {
	return -font.Ascent(fontName, fontSize) - mTop
}

func deltaAlignBottom(fontName string, fontSize, lines int, mBot float64) float64 {
	return -font.Ascent(fontName, fontSize) + float64(lines)*font.LineHeight(fontName, fontSize) + mBot
}

var unicodeToCP1252 = map[rune]byte{
	0x20AC: 128, // € Euro Sign Note: Width in metrics file is not correct!
	0x201A: 130, // ‚ Single Low-9 Quotation Mark
	0x0192: 131, // ƒ Latin Small Letter F with Hook
	0x201E: 132, // „ Double Low-9 Quotation Mark
	0x2026: 133, // … Horizontal Ellipsis
	0x2020: 134, // † Dagger
	0x2021: 135, // ‡ Double Dagger
	0x02C6: 136, // ˆ Modifier Letter Circumflex Accent
	0x2030: 137, // ‰ Per Mille Sign
	0x0160: 138, // Š Latin Capital Letter S with Caron
	0x2039: 139, // ‹ Single Left-Pointing Angle Quotation Mark
	0x0152: 140, // Œ Latin Capital Ligature Oe
	0x017D: 142, // Ž Latin Capital Letter Z with Caron
	0x2018: 145, // ‘ Left Single Quotation Mark
	0x2019: 146, // ’ Right Single Quotation Mark
	0x201C: 147, // “ Left Double Quotation Mark
	0x201D: 148, // ” Right Double Quotation Mark
	0x2022: 149, // • Bullet
	0x2013: 150, // – En Dash
	0x2014: 151, // — Em Dash
	0x02DC: 152, // ˜ Small Tilde
	0x2122: 153, // ™ Trade Mark Sign Emoji
	0x0161: 154, // š Latin Small Letter S with Caron
	0x203A: 155, // › Single Right-Pointing Angle Quotation Mark
	0x0153: 156, // œ Latin Small Ligature Oe
	0x017E: 158, // ž Latin Small Letter Z with Caron
	0x0178: 159, // Ÿ Latin Capital Letter Y with Diaeresis
}

func DecodeUTF8ToByte(s string) string {
	var sb strings.Builder
	for _, r := range s {
		// Unicode => char code
		if r <= 0xFF {
			sb.WriteByte(byte(r))
			continue
		}
		if b, ok := unicodeToCP1252[r]; ok {
			sb.WriteByte(b)
			continue
		}
		sb.WriteByte(byte(0x20))
	}
	return sb.String()
}

// SetLineJoinStyle sets the line join style for stroking operations.
func SetLineJoinStyle(w io.Writer, s LineJoinStyle) {
	fmt.Fprintf(w, "%d j ", s)
}

// SetLineWidth sets line width for stroking operations.
func SetLineWidth(w io.Writer, width float64) {
	fmt.Fprintf(w, "%.2f w ", width)
}

// SetStrokeColor sets the stroke color.
func SetStrokeColor(w io.Writer, c SimpleColor) {
	fmt.Fprintf(w, "%.2f %.2f %.2f RG ", c.R, c.G, c.B)
}

// SetFillColor sets the fill color.
func SetFillColor(w io.Writer, c SimpleColor) {
	fmt.Fprintf(w, "%.2f %.2f %.2f rg ", c.R, c.G, c.B)
}

// DrawLineSimple draws the path from P to Q.
func DrawLineSimple(w io.Writer, xp, yp, xq, yq float64) {
	fmt.Fprintf(w, "%.2f %.2f m %.2f %.2f l s ", xp, yp, xq, yq)
}

// DrawLine draws the path from P to Q using lineWidth, strokeColor and style.
func DrawLine(w io.Writer, xp, yp, xq, yq float64, lineWidth float64, strokeColor *SimpleColor, style *LineJoinStyle) {
	fmt.Fprintf(w, "q ")
	SetLineWidth(w, lineWidth)
	if strokeColor != nil {
		SetStrokeColor(w, *strokeColor)
	}
	if style != nil {
		SetLineJoinStyle(w, *style)
	}
	DrawLineSimple(w, xp, yp, xq, yq)
	fmt.Fprintf(w, "Q ")
}

// DrawRectSimple strokes a rectangular path for r.
func DrawRectSimple(w io.Writer, r *Rectangle) {
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f re s ", r.LL.X, r.LL.Y, r.Width(), r.Height())
}

// DrawRect strokes a rectangular path for r using lineWidth, strokeColor and style.
func DrawRect(w io.Writer, r *Rectangle, lineWidth float64, strokeColor *SimpleColor, style *LineJoinStyle) {
	fmt.Fprintf(w, "q ")
	SetLineWidth(w, lineWidth)
	if strokeColor != nil {
		SetStrokeColor(w, *strokeColor)
	}
	if style != nil {
		SetLineJoinStyle(w, *style)
	}
	DrawRectSimple(w, r)
	fmt.Fprintf(w, "Q ")
}

// FillRect fills a rectangular path for r using lineWidth, strokeCol, fillCol and style.
func FillRect(w io.Writer, r *Rectangle, lineWidth float64, strokeCol *SimpleColor, fillCol SimpleColor, style *LineJoinStyle) {
	fmt.Fprintf(w, "q ")
	SetLineWidth(w, lineWidth)
	c := fillCol
	if strokeCol != nil {
		c = *strokeCol
	}
	SetStrokeColor(w, c)
	SetFillColor(w, fillCol)
	if style != nil {
		SetLineJoinStyle(w, *style)
	}
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f re B ", r.LL.X, r.LL.Y, r.Width(), r.Height())
	fmt.Fprintf(w, "Q ")
}

// FillRectNoBorder fills a rectangular path for r using fillCol.
func FillRectNoBorder(w io.Writer, r *Rectangle, fillCol SimpleColor) {
	fmt.Fprintf(w, "q ")
	SetStrokeColor(w, fillCol)
	SetFillColor(w, fillCol)
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f re B ", r.LL.X, r.LL.Y, r.Width(), r.Height())
	fmt.Fprintf(w, "Q ")
}

// DrawGrid draws an x * y grid on r using strokeCol and fillCol.
func DrawGrid(w io.Writer, x, y int, r *Rectangle, strokeCol SimpleColor, fillCol *SimpleColor) {

	if fillCol != nil {
		FillRectNoBorder(w, r, *fillCol)
	}

	s := r.Width() / float64(x)
	for i := 0; i <= x; i++ {
		x := r.LL.X + float64(i)*s
		DrawLine(w, x, r.LL.Y, x, r.UR.Y, 0, &strokeCol, nil)
	}

	s = r.Height() / float64(y)
	for i := 0; i <= y; i++ {
		y := r.LL.Y + float64(i)*s
		DrawLine(w, r.LL.X, y, r.UR.X, y, 0, &strokeCol, nil)
	}
}

func calcBoundingBoxForRectAndPoint(r *Rectangle, p types.Point) *Rectangle {
	llx, lly, urx, ury := r.LL.X, r.LL.Y, r.UR.X, r.UR.Y
	if p.X < r.LL.X {
		llx = p.X
	} else if p.X > r.UR.X {
		urx = p.X
	}
	if p.Y < r.LL.Y {
		lly = p.Y
	} else if p.Y > r.UR.Y {
		ury = p.Y
	}
	return Rect(llx, lly, urx, ury)
}

func calcBoundingBoxForRects(r1, r2 *Rectangle) *Rectangle {
	if r1 == nil && r2 == nil {
		return Rect(0, 0, 0, 0)
	}
	if r1 == nil {
		return r2
	}
	if r2 == nil {
		return r1
	}
	bbox := calcBoundingBoxForRectAndPoint(r1, r2.LL)
	return calcBoundingBoxForRectAndPoint(bbox, r2.UR)
}

func calcBoundingBoxForLines(lines []string, x, y float64, fontName string, fontSize int) (*Rectangle, string) {
	var (
		box      *Rectangle
		maxLine  string
		maxWidth float64
	)
	// TODO Return error if lines == nil or empty.
	for _, s := range lines {
		bbox := CalcBoundingBox(s, x, y, fontName, fontSize)
		if bbox.Width() > maxWidth {
			maxWidth = bbox.Width()
			maxLine = s
		}
		box = calcBoundingBoxForRects(box, bbox)
		y -= bbox.Height()
	}
	return box, maxLine
}

func calcBoundingBoxJ(x, y, w, h, d float64, fontName string, fontSize int) *Rectangle {
	y -= d
	return Rect(x, y, x+w, y+h)
}

func calcBoundingBoxForJLines(lines []string, x, y, w float64, fontName string, fontSize int) *Rectangle {
	var box *Rectangle
	h := font.LineHeight(fontName, fontSize)
	d := math.Ceil(font.Descent(fontName, fontSize))
	// TODO Return error if lines == nil or empty.
	for i := 0; i < len(lines); i++ {
		bbox := calcBoundingBoxJ(x, y, w, h, d, fontName, fontSize)
		box = calcBoundingBoxForRects(box, bbox)
		y -= bbox.Height()
	}
	return box
}

// DrawHairCross draw a haircross with origin x/y.
func DrawHairCross(w io.Writer, x, y float64, r *Rectangle) {
	x1, y1 := x, y
	if x == 0 {
		x1 = r.LL.X + r.Width()/2
	}
	if y == 0 {
		y1 = r.LL.Y + r.Height()/2
	}
	DrawLine(w, r.LL.X, y1, r.LL.X+r.Width(), y1, 0, &Black, nil)  // Horizontal line
	DrawLine(w, x1, r.LL.Y, x1, r.LL.Y+r.Height(), 0, &Black, nil) // Vertical line

	//DrawLine(w, r.LL.X, y1, r.LL.X+r.Width(), y1, 0, &Black, nil)  // Horizontal line
	//DrawLine(w, x1, r.LL.Y, x1, r.LL.Y+r.Height(), 0, &Black, nil) // Vertical line
}

func PrepBytes(s, fontName string, rtl bool) string {
	if font.IsUserFont(fontName) {
		ttf := font.UserFontMetrics[fontName]
		bb := []byte{}
		if rtl {
			s = reverse(s)
		}
		for _, r := range s {
			gid, ok := ttf.Chars[uint32(r)]
			if ok {
				bb = append(bb, byte((gid>>8)&0xFF))
				bb = append(bb, byte(gid&0xFF))
				ttf.UsedGIDs[gid] = true
			}
		}
		s = string(bb)
	}
	s1, _ := Escape(s)
	return *s1
}

func writeStringToBuf(w io.Writer, s string, x, y float64, td TextDescriptor) {
	s = PrepBytes(s, td.FontName, td.RTL)
	fmt.Fprintf(w, "BT 0 Tw %.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td %d Tr (%s) Tj ET ",
		td.StrokeCol.R, td.StrokeCol.G, td.StrokeCol.B, td.FillCol.R, td.FillCol.G, td.FillCol.B, x, y, td.RMode, s)
}

func setFont(w io.Writer, fontID string, fontSize float32) {
	fmt.Fprintf(w, "BT /%s %.2f Tf ET ", fontID, fontSize)
}

func CalcBoundingBox(s string, x, y float64, fontName string, fontSize int) *Rectangle {
	w := font.TextWidth(s, fontName, fontSize)
	h := font.LineHeight(fontName, fontSize)
	y -= math.Ceil(font.Descent(fontName, fontSize))
	return Rect(x, y, x+w, y+h)
}

func calcRotateTransformMatrix(rot float64, bb *Rectangle) Matrix {
	sin := math.Sin(float64(rot) * float64(DegToRad))
	cos := math.Cos(float64(rot) * float64(DegToRad))
	dx := bb.LL.X + bb.Width()/2 + sin*(bb.Height()/2) - cos*bb.Width()/2
	dy := bb.LL.Y + bb.Height()/2 - cos*(bb.Height()/2) - sin*bb.Width()/2
	return CalcTransformMatrix(1, 1, sin, cos, dx, dy)
}

func horAdjustBoundingBoxForLines(r, box *Rectangle, dx, dy float64, x, y *float64) {
	if r.UR.X-box.LL.X < box.Width() {
		dx -= box.Width() - (r.UR.X - box.LL.X)
		*x += dx
		box.Translate(dx, 0)
	} else if box.LL.X < r.LL.X {
		dx += r.LL.X - box.LL.X
		*x += dx
		box.Translate(dx, 0)
	}
	if r.UR.Y-box.LL.Y < box.Height() {
		dy -= box.Height() - (r.UR.Y - box.LL.Y)
		*y += dy
		box.Translate(0, dy)
	} else if box.LL.Y < r.LL.Y {
		dy += r.LL.Y - box.LL.Y
		*y += dy
		box.Translate(0, dy)
	}
}

func prepJustifiedLine(lines *[]string, strbuf []string, strWidth, w float64, fontSize int, fontName string, rtl bool) {
	blank := PrepBytes(" ", fontName, false)
	var sb strings.Builder
	sb.WriteString("[")
	wc := len(strbuf)
	dx := font.GlyphSpaceUnits(float64((w-strWidth))/float64(wc-1), fontSize)
	for i := 0; i < wc; i++ {
		j := i
		if rtl {
			j = wc - 1 - i
		}
		s := PrepBytes(strbuf[j], fontName, rtl)
		sb.WriteString(fmt.Sprintf(" (%s)", s))
		if i < wc-1 {
			sb.WriteString(fmt.Sprintf(" %d (%s)", -int(dx), blank))
		}
	}
	sb.WriteString(" ] TJ")
	*lines = append(*lines, sb.String())
}

func newPrepJustifiedString(
	fontName string,
	fontSize int) func(lines *[]string, s string, w float64, fontName string, fontSize *int, lastline, parIndent, rtl bool) int {

	// Not yet rendered content.
	strbuf := []string{}

	// Width of strbuf's content in user space implied by fontSize.
	var strWidth float64

	// Indent first line of paragraphs.
	var indent bool = true

	// Indentation string for first line of paragraphs.
	identPrefix := "    "

	blankWidth := font.TextWidth(" ", fontName, fontSize)

	return func(lines *[]string, s string, w float64, fontName string, fontSize *int, lastline, parIndent, rtl bool) int {

		if len(s) == 0 {
			if len(strbuf) > 0 {
				s1 := PrepBytes(strings.Join(strbuf, " "), fontName, rtl)
				if rtl {
					dx := font.GlyphSpaceUnits(w-strWidth, *fontSize)
					s = fmt.Sprintf("[ %d (%s) ] TJ ", -int(dx), s1)
				} else {
					s = fmt.Sprintf("(%s) Tj", s1)
				}
				*lines = append(*lines, s)
				strbuf = []string{}
				strWidth = 0
			}
			if lastline {
				return 0
			}
			indent = true
			if parIndent {
				return 0
			}
			return 1
		}

		linefeeds := 0
		ss := strings.Split(s, " ")
		if parIndent && len(strbuf) == 0 && indent {
			ss[0] = identPrefix + ss[0]
		}

		for _, s1 := range ss {
			s1Width := font.TextWidth(s1, fontName, *fontSize)
			bw := 0.
			if len(strbuf) > 0 {
				bw = blankWidth
			}
			if w-strWidth-(s1Width+bw) > 0 {
				strWidth += s1Width + bw
				strbuf = append(strbuf, s1)
				continue
			}
			// Ensure s1 fits into w.
			fs := font.Size(s1, fontName, w)
			if fs < *fontSize {
				*fontSize = fs
			}
			if len(strbuf) == 0 {
				prepJustifiedLine(lines, []string{s1}, s1Width, w, *fontSize, fontName, rtl)
			} else {
				// Note: Previous lines have whitespace based on bigger font size.
				prepJustifiedLine(lines, strbuf, strWidth, w, *fontSize, fontName, rtl)
				strbuf = []string{s1}
				strWidth = s1Width
			}
			linefeeds++
			indent = false
		}
		return 0
	}
}

// Prerender justified text in order to calculate bounding box height.
func preRenderJustifiedText(
	lines *[]string,
	r *Rectangle,
	x, y, width float64,
	td TextDescriptor,
	mLeft, mRight, borderWidth float64,
	fontSize *int) float64 {

	var ww float64
	if !td.ScaleAbs {
		ww = r.Width() * td.Scale
	} else {
		if width > 0 {
			ww = width * td.Scale
		} else {
			box, _ := calcBoundingBoxForLines(*lines, x, y, td.FontName, *fontSize)
			ww = box.Width() * td.Scale
		}
	}
	ww -= mLeft + mRight + 2*borderWidth
	prepJustifiedString := newPrepJustifiedString(td.FontName, *fontSize)
	l := []string{}
	for i, s := range *lines {
		linefeeds := prepJustifiedString(&l, s, ww, td.FontName, fontSize, false, td.ParIndent, td.RTL)
		for j := 0; j < linefeeds; j++ {
			l = append(l, "")
		}
		isLastLine := i == len(*lines)-1
		if isLastLine {
			prepJustifiedString(&l, "", ww, td.FontName, fontSize, true, td.ParIndent, td.RTL)
		}
	}
	*lines = l
	return ww
}

func scaleFontSize(r *Rectangle, lines []string, scaleAbs bool,
	scale, width, x, y, mLeft, mRight, borderWidth float64,
	fontName string, fontSize *int) {
	if scaleAbs {
		*fontSize = int(float64(*fontSize) * scale)
	} else {
		www := width
		if width == 0 {
			box, _ := calcBoundingBoxForLines(lines, x, y, fontName, *fontSize)
			www = box.Width() + mLeft + mRight + 2*borderWidth
		}
		*fontSize = int(r.Width() * scale * float64(*fontSize) / www)
	}
}

func horizontalWrapUp(box *Rectangle, maxLine string, hAlign HAlignment,
	x *float64, width, ww, mLeft, mRight, borderWidth float64,
	fontName string, fontSize *int) {
	switch hAlign {
	case AlignLeft:
		box.Translate(mLeft+borderWidth, 0)
		*x += mLeft + borderWidth
	case AlignJustify:
		if width > 0 {
			box.Translate(mLeft+borderWidth, 0)
			*x += mLeft + borderWidth
		} else {
			box.Translate(-ww/2, 0)
			*x -= ww / 2
		}
	case AlignRight:
		box.Translate(-box.Width()-mRight-borderWidth, 0)
		*x -= mRight + borderWidth
	case AlignCenter:
		box.Translate(-box.Width()/2, 0)
	}

	if hAlign == AlignJustify {
		box.UR.X = box.LL.X + ww + mRight + borderWidth
		box.LL.X -= mLeft + borderWidth
	} else if width > 0 {
		netWidth := width - 2*borderWidth - mLeft - mRight
		if box.Width() > netWidth {
			*fontSize = font.Size(maxLine, fontName, netWidth)
		}
		switch hAlign {
		case AlignLeft:
			box.UR.X = box.LL.X + width - mLeft - borderWidth
			box.LL.X -= mLeft + borderWidth
		case AlignRight:
			box.LL.X = box.UR.X - width
			box.Translate(mRight+borderWidth, 0)
		case AlignCenter:
			box.LL.X = box.UR.X - width
			box.Translate(box.Width()/2-(box.UR.X-*x), 0)
		}
	} else {
		box.LL.X -= mLeft + borderWidth
		box.UR.X += mRight + borderWidth
	}
}

func createBoundingBoxForColumn(r *Rectangle, x, y *float64,
	width float64,
	td TextDescriptor,
	dx, dy float64,
	mTop, mBot, mLeft, mRight float64,
	borderWidth float64,
	fontSize *int, lines *[]string) *Rectangle {

	var ww float64
	if td.HAlign == AlignJustify {
		ww = preRenderJustifiedText(lines, r, *x, *y, width, td, mLeft, mRight, borderWidth, fontSize)
	}

	if td.HAlign != AlignJustify {
		scaleFontSize(r, *lines, td.ScaleAbs, td.Scale, width, *x, *y, mLeft, mRight, borderWidth, td.FontName, fontSize)
	}

	// Apply vertical alignment.
	var dy1 float64
	switch td.VAlign {
	case AlignTop:
		dy1 = deltaAlignTop(td.FontName, *fontSize, mTop+borderWidth)
	case AlignMiddle:
		dy1 = deltaAlignMiddle(td.FontName, *fontSize, len(*lines), mTop, mBot)
	case AlignBottom:
		dy1 = deltaAlignBottom(td.FontName, *fontSize, len(*lines), mBot)
	}
	*y += math.Ceil(dy1)

	box, maxLine := calcBoundingBoxForLines(*lines, *x, *y, td.FontName, *fontSize)
	// maxLine for hAlign != AlignJustify only!
	horizontalWrapUp(box, maxLine, td.HAlign, x, width, ww, mLeft, mRight, borderWidth, td.FontName, fontSize)

	box.LL.Y -= mBot + borderWidth
	box.UR.Y += mTop + borderWidth

	if td.MinHeight > 0 && box.Height() < td.MinHeight {
		box.LL.Y = box.UR.Y - td.MinHeight
	}

	horAdjustBoundingBoxForLines(r, box, dx, dy, x, y)

	return box
}

func flushJustifiedStringToBuf(w io.Writer, s string, x, y float64, strokeCol, fillCol SimpleColor, rm RenderMode) {
	fmt.Fprintf(w, "BT 0 Tw %.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td %d Tr %s ET ",
		strokeCol.R, strokeCol.G, strokeCol.B, fillCol.R, fillCol.G, fillCol.B, x, y, rm, s)
}

func scaleXForRegion(x float64, mediaBox, region *Rectangle) float64 {
	return x / mediaBox.Width() * region.Width()
}

func scaleYForRegion(y float64, mediaBox, region *Rectangle) float64 {
	return y / mediaBox.Width() * region.Width()
}

func drawMargins(w io.Writer, c SimpleColor, colBB *Rectangle, borderWidth, mLeft, mRight, mTop, mBot float64) {
	if mLeft <= 0 && mRight <= 0 && mTop <= 0 && mBot <= 0 {
		return
	}

	var r *Rectangle

	if mBot > 0 {
		r = RectForWidthAndHeight(colBB.LL.X+borderWidth, colBB.LL.Y+borderWidth, colBB.Width()-2*borderWidth, mBot)
		FillRectNoBorder(w, r, c)
	}

	if mTop > 0 {
		r = RectForWidthAndHeight(colBB.LL.X+borderWidth, colBB.UR.Y-borderWidth-mTop, colBB.Width()-2*borderWidth, mTop)
		FillRectNoBorder(w, r, c)
	}

	if mLeft > 0 {
		r = RectForWidthAndHeight(colBB.LL.X+borderWidth, colBB.LL.Y+borderWidth+mBot, mLeft, colBB.Height()-2*borderWidth-mTop-mBot)
		FillRectNoBorder(w, r, c)
	}

	if mRight > 0 {
		r = RectForWidthAndHeight(colBB.UR.X-borderWidth-mRight, colBB.LL.Y+borderWidth+mBot, mRight, colBB.Height()-2*borderWidth-mTop-mBot)
		FillRectNoBorder(w, r, c)
	}

}

func renderBackgroundAndBorder(w io.Writer, td TextDescriptor, borderWidth float64, colBB *Rectangle) {
	r := RectForWidthAndHeight(colBB.LL.X+borderWidth/2, colBB.LL.Y+borderWidth/2, colBB.Width()-borderWidth, colBB.Height()-borderWidth)
	if td.ShowBackground {
		c := td.BackgroundCol
		if td.ShowBorder {
			c = td.BorderCol
		}
		FillRect(w, r, borderWidth, &c, td.BackgroundCol, &td.BorderStyle)
	} else if td.ShowBorder {
		DrawRect(w, r, borderWidth, &td.BorderCol, &td.BorderStyle)
	}
}

func reverse(s string) string {
	inRunes := []rune(norm.NFC.String(s))
	outRunes := make([]rune, len(inRunes))
	iMax := len(inRunes) - 1
	for i, r := range inRunes {
		outRunes[iMax-i] = r
	}
	return string(outRunes)
}

func renderText(w io.Writer, lines []string, td TextDescriptor, x, y float64, fontSize int) {
	lh := font.LineHeight(td.FontName, fontSize)
	for _, s := range lines {
		if td.HAlign != AlignJustify {
			lineBB := CalcBoundingBox(s, x, y, td.FontName, fontSize)
			// Apply horizontal alignment.
			var dx float64
			switch td.HAlign {
			case AlignCenter:
				dx = lineBB.Width() / 2
			case AlignRight:
				dx = lineBB.Width()
			}
			lineBB.Translate(-dx, 0)
			if td.ShowLineBB {
				// Draw line bounding box.
				SetStrokeColor(w, Black)
				DrawRectSimple(w, lineBB)
			}
			writeStringToBuf(w, s, x-dx, y, td)
			y -= lh
			continue
		}

		if len(s) > 0 {
			flushJustifiedStringToBuf(w, s, x, y, td.StrokeCol, td.FillCol, td.RMode)
		}
		y -= lh
	}
}

func SplitMultilineStr(s string) []string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	var lines []string
	return append(lines, fieldsFunc(s, func(c rune) bool { return c == 0x0a })...)
}

// WriteColumn writes a text column using s at position x/y using a certain font, fontsize and a desired horizontal and vertical alignment.
// Enforce a desired column width by supplying a width > 0 (especially useful for justified text).
// It returns the bounding box of this column.
func WriteColumn(w io.Writer, mediaBox, region *Rectangle, td TextDescriptor, width float64) *Rectangle {
	x, y, dx, dy := td.X, td.Y, td.Dx, td.Dy
	mTop, mBot, mLeft, mRight := td.MTop, td.MBot, td.MLeft, td.MRight
	s, fontSize, borderWidth := td.Text, td.FontSize, td.BorderWidth

	r := mediaBox
	if region != nil {
		r = region
		dx = scaleXForRegion(dx, mediaBox, r)
		dy = scaleYForRegion(dy, mediaBox, r)
		width = scaleXForRegion(width, mediaBox, r)
		fontSize = int(scaleYForRegion(float64(fontSize), mediaBox, r))
		mTop = scaleYForRegion(mTop, mediaBox, r)
		mBot = scaleYForRegion(mBot, mediaBox, r)
		mLeft = scaleXForRegion(mLeft, mediaBox, r)
		mRight = scaleXForRegion(mRight, mediaBox, r)
		borderWidth = scaleXForRegion(borderWidth, mediaBox, r)
	}

	if x >= 0 {
		x = r.LL.X + x
	}
	if y >= 0 {
		y = r.LL.Y + y
	}

	// Position text horizontally centered for x < 0.
	if x < 0 {
		x = r.LL.X + r.Width()/2
	}

	// Position text vertically centered for y < 0.
	if y < 0 {
		y = r.LL.Y + r.Height()/2
	}

	// Apply offset.
	x += dx
	y += dy

	// Cache haircross coordinates.
	x0, y0 := x, y

	if font.IsCoreFont(td.FontName) && utf8.ValidString(s) {
		s = DecodeUTF8ToByte(s)
	}

	lines := SplitMultilineStr(s)

	// s = strings.ReplaceAll(s, "\\n", "\n")
	// lines := []string{}
	// for _, l := range fieldsFunc(s, func(c rune) bool { return c == 0x0a }) {
	// 	lines = append(lines, l)
	// }

	if !td.ScaleAbs {
		if td.Scale > 1 {
			td.Scale = 1
		}
	}

	// Create bounding box and prerender content stream bytes for justified text.
	colBB := createBoundingBoxForColumn(
		r, &x, &y, width, td, dx, dy, mTop, mBot, mLeft, mRight, borderWidth, &fontSize, &lines)

	fmt.Fprint(w, "q ")

	setFont(w, td.FontKey, float32(fontSize))
	m := calcRotateTransformMatrix(td.Rotation, colBB)
	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f %.2f %.2f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	x -= colBB.LL.X
	y -= colBB.LL.Y
	colBB.Translate(-colBB.LL.X, -colBB.LL.Y)

	// Render background and border.
	if td.ShowTextBB {
		renderBackgroundAndBorder(w, td, borderWidth, colBB)
	}

	// Render margins
	if td.ShowMargins {
		drawMargins(w, LightGray, colBB, borderWidth, mLeft, mRight, mTop, mBot)
	}

	// Render text.
	renderText(w, lines, td, x, y, fontSize)

	fmt.Fprintf(w, "Q ")

	if td.HairCross {
		DrawHairCross(w, x0, y0, r)
	}

	return colBB
}

// WriteMultiLine writes s at position x/y using a certain font, fontsize and a desired horizontal and vertical alignment.
// It returns the bounding box of this text column.
func WriteMultiLine(w io.Writer, mediaBox, region *Rectangle, td TextDescriptor) *Rectangle {
	return WriteColumn(w, mediaBox, region, td, 0)
}

func anchorPosAndAlign(a Anchor, r *Rectangle) (x, y float64, hAlign HAlignment, vAlign VAlignment) {
	switch a {
	case TopLeft:
		x, y, hAlign, vAlign = 0, r.Height(), AlignLeft, AlignTop
	case TopCenter:
		x, y, hAlign, vAlign = -1, r.Height(), AlignCenter, AlignTop
	case TopRight:
		x, y, hAlign, vAlign = r.Width(), r.Height(), AlignRight, AlignTop
	case Left:
		x, y, hAlign, vAlign = 0, -1, AlignLeft, AlignMiddle
	case Center:
		x, y, hAlign, vAlign = -1, -1, AlignCenter, AlignMiddle
	case Right:
		x, y, hAlign, vAlign = r.Width(), -1, AlignRight, AlignMiddle
	case BottomLeft:
		x, y, hAlign, vAlign = 0, 0, AlignLeft, AlignMiddle
	case BottomCenter:
		x, y, hAlign, vAlign = -1, 0, AlignCenter, AlignMiddle
	case BottomRight:
		x, y, hAlign, vAlign = r.Width(), 0, AlignRight, AlignMiddle
	}
	return
}

// WriteMultiLineAnchored writes multiple lines with anchored position and returns its bounding box.
func WriteMultiLineAnchored(w io.Writer, mediaBox, region *Rectangle, td TextDescriptor, a Anchor) *Rectangle {
	r := mediaBox
	if region != nil {
		r = region
	}
	td.X, td.Y, td.HAlign, td.VAlign = anchorPosAndAlign(a, r)
	return WriteMultiLine(w, mediaBox, region, td)
}

// WriteColumnAnchored writes a justified text column with anchored position and returns its bounding box.
func WriteColumnAnchored(w io.Writer, mediaBox, region *Rectangle, td TextDescriptor, a Anchor, width float64) *Rectangle {
	r := mediaBox
	if region != nil {
		r = region
	}
	td.HAlign = AlignJustify
	td.X, td.Y, _, td.VAlign = anchorPosAndAlign(a, r)
	return WriteColumn(w, mediaBox, region, td, width)
}

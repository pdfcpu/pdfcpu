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

package model

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TextDescriptor contains all attributes needed for rendering a text column in PDF user space.
type TextDescriptor struct {
	Text           string              // A multi line string using \n for line breaks.
	FontName       string              // Name of the core or user font to be used.
	RTL            bool                // Right to left user font.
	FontKey        string              // Resource id registered for FontName.
	FontSize       int                 // Fontsize in points.
	X, Y           float64             // Position of first char's baseline.
	Dx, Dy         float64             // Horizontal and vertical offsets for X,Y.
	MTop, MBot     float64             // Top and bottom margins applied to text bounding box.
	MLeft, MRight  float64             // Left and right margins applied to text bounding box.
	MinHeight      float64             // The minimum height of this text's bounding box.
	Rotation       float64             // 0..360 degree rotation angle.
	ScaleAbs       bool                // Scaling type, true=absolute, false=relative to container dimensions.
	Scale          float64             // font scaling factor > 0 (and <= 1 for relative scaling).
	HAlign         types.HAlignment    // Horizontal text alignment.
	VAlign         types.VAlignment    // Vertical text alignment.
	RMode          draw.RenderMode     // Text render mode
	StrokeCol      color.SimpleColor   // Stroke color to be used for rendering text corresponding to RMode.
	FillCol        color.SimpleColor   // Fill color to be used for rendering text corresponding to RMode.
	ShowTextBB     bool                // Render bounding box including BackgroundCol, border and margins.
	ShowBackground bool                // Render background of bounding box using BackgroundCol.
	BackgroundCol  color.SimpleColor   // Bounding box fill color.
	ShowBorder     bool                // Render border using BorderCol, BorderWidth and BorderStyle.
	BorderWidth    float64             // Border width, visibility depends on ShowBorder.
	BorderStyle    types.LineJoinStyle // Border style, also visible if ShowBorder is false as long as ShowBackground is true.
	BorderCol      color.SimpleColor   // Border color.
	ParIndent      bool                // Indent first line of paragraphs or space between paragraphs.
	ShowLineBB     bool                // Render line bounding boxes in black (for HAlign != AlignJustify only)
	ShowMargins    bool                // Render margins in light gray.
	ShowPosition   bool                // Highlight position.
	HairCross      bool                // Draw haircross at X,Y
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

func calcBoundingBoxForRectAndPoint(r *types.Rectangle, p types.Point) *types.Rectangle {
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
	return types.NewRectangle(llx, lly, urx, ury)
}

func CalcBoundingBoxForRects(r1, r2 *types.Rectangle) *types.Rectangle {
	if r1 == nil && r2 == nil {
		return types.NewRectangle(0, 0, 0, 0)
	}
	if r1 == nil {
		return r2.Clone()
	}
	if r2 == nil {
		return r1.Clone()
	}
	bbox := calcBoundingBoxForRectAndPoint(r1, r2.LL)
	return calcBoundingBoxForRectAndPoint(bbox, r2.UR)
}

func calcBoundingBoxForLines(lines []string, x, y float64, fontName string, fontSize int) (*types.Rectangle, string) {
	var (
		box      *types.Rectangle
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
		box = CalcBoundingBoxForRects(box, bbox)
		y -= bbox.Height()
	}
	return box, maxLine
}

func PrepBytes(xRefTable *XRefTable, s, fontName string, cjk, rtl bool) string {
	if font.IsUserFont(fontName) {
		if rtl {
			s = types.Reverse(s)
		}
		bb := []byte{}
		if cjk {
			for _, r := range s {
				b := make([]byte, 2)
				binary.BigEndian.PutUint16(b, uint16(r))
				bb = append(bb, b...)
			}
		} else {
			usedGIDs, ok := xRefTable.UsedGIDs[fontName]
			if !ok {
				xRefTable.UsedGIDs[fontName] = map[uint16]bool{}
				usedGIDs = xRefTable.UsedGIDs[fontName]
			}

			font.UserFontMetricsLock.RLock()
			ttf := font.UserFontMetrics[fontName]
			font.UserFontMetricsLock.RUnlock()

			for _, r := range s {
				gid, ok := ttf.Chars[uint32(r)]
				if ok {
					b := make([]byte, 2)
					binary.BigEndian.PutUint16(b, gid)
					bb = append(bb, b...)
					usedGIDs[gid] = true
				} // else "invalid char"
			}
		}
		s = string(bb)
	}
	s1, _ := types.Escape(s)
	return *s1
}

func writeStringToBuf(xRefTable *XRefTable, w io.Writer, s string, x, y float64, td TextDescriptor) {
	s = PrepBytes(xRefTable, s, td.FontName, false, td.RTL)
	fmt.Fprintf(w, "BT 0 Tw %.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td %d Tr (%s) Tj ET ",
		td.StrokeCol.R, td.StrokeCol.G, td.StrokeCol.B, td.FillCol.R, td.FillCol.G, td.FillCol.B, x, y, td.RMode, s)
}

func setFont(w io.Writer, fontID string, fontSize float32) {
	fmt.Fprintf(w, "BT /%s %.2f Tf ET ", fontID, fontSize)
}

func CalcBoundingBox(s string, x, y float64, fontName string, fontSize int) *types.Rectangle {
	w := font.TextWidth(s, fontName, fontSize)
	h := font.LineHeight(fontName, fontSize)
	y -= math.Ceil(font.Descent(fontName, fontSize))
	return types.NewRectangle(x, y, x+w, y+h)
}

func horAdjustBoundingBoxForLines(r, box *types.Rectangle, dx, dy float64, x, y *float64) {
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

func prepJustifiedLine(xRefTable *XRefTable, lines *[]string, strbuf []string, strWidth, w float64, fontSize int, fontName string, rtl bool) {
	blank := PrepBytes(xRefTable, " ", fontName, false, false)
	var sb strings.Builder
	sb.WriteString("[")
	wc := len(strbuf)
	dx := font.GlyphSpaceUnits(float64((w-strWidth))/float64(wc-1), fontSize)
	for i := 0; i < wc; i++ {
		j := i
		if rtl {
			j = wc - 1 - i
		}
		s := PrepBytes(xRefTable, strbuf[j], fontName, false, rtl)
		sb.WriteString(fmt.Sprintf(" (%s)", s))
		if i < wc-1 {
			sb.WriteString(fmt.Sprintf(" %d (%s)", -int(dx), blank))
		}
	}
	sb.WriteString(" ] TJ")
	*lines = append(*lines, sb.String())
}

func newPrepJustifiedString(
	xRefTable *XRefTable,
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
				s1 := PrepBytes(xRefTable, strings.Join(strbuf, " "), fontName, false, rtl)
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
				prepJustifiedLine(xRefTable, lines, []string{s1}, s1Width, w, *fontSize, fontName, rtl)
			} else {
				// Note: Previous lines have whitespace based on bigger font size.
				prepJustifiedLine(xRefTable, lines, strbuf, strWidth, w, *fontSize, fontName, rtl)
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
	xRefTable *XRefTable,
	lines *[]string,
	r *types.Rectangle,
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
	prepJustifiedString := newPrepJustifiedString(xRefTable, td.FontName, *fontSize)
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

func scaleFontSize(r *types.Rectangle, lines []string, scaleAbs bool,
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

func horizontalWrapUp(box *types.Rectangle, maxLine string, hAlign types.HAlignment,
	x *float64, width, ww, mLeft, mRight, borderWidth float64,
	fontName string, fontSize *int) {
	switch hAlign {
	case types.AlignLeft:
		box.Translate(mLeft+borderWidth, 0)
		*x += mLeft + borderWidth
	case types.AlignJustify:
		box.Translate(mLeft+borderWidth, 0)
		*x += mLeft + borderWidth
	case types.AlignRight:
		box.Translate(-box.Width()-mRight-borderWidth, 0)
		*x -= mRight + borderWidth
	case types.AlignCenter:
		box.Translate(-box.Width()/2, 0)
	}

	if hAlign == types.AlignJustify {
		box.UR.X = box.LL.X + ww + mRight + borderWidth
		box.LL.X -= mLeft + borderWidth
	} else if width > 0 {
		netWidth := width - 2*borderWidth - mLeft - mRight
		if box.Width() > netWidth {
			*fontSize = font.Size(maxLine, fontName, netWidth)
		}
		switch hAlign {
		case types.AlignLeft:
			box.UR.X = box.LL.X + width - mLeft - borderWidth
			box.LL.X -= mLeft + borderWidth
		case types.AlignRight:
			box.LL.X = box.UR.X - width
			box.Translate(mRight+borderWidth, 0)
		case types.AlignCenter:
			box.LL.X = box.UR.X - width
			box.Translate(box.Width()/2-(box.UR.X-*x), 0)
		}
	} else {
		box.LL.X -= mLeft + borderWidth
		box.UR.X += mRight + borderWidth
	}
}

func createBoundingBoxForColumn(xRefTable *XRefTable, r *types.Rectangle, x, y *float64,
	width float64,
	td TextDescriptor,
	dx, dy float64,
	mTop, mBot, mLeft, mRight float64,
	borderWidth float64,
	fontSize *int, lines *[]string) *types.Rectangle {

	var ww float64
	if td.HAlign == types.AlignJustify {
		ww = preRenderJustifiedText(xRefTable, lines, r, *x, *y, width, td, mLeft, mRight, borderWidth, fontSize)
	}

	if td.HAlign != types.AlignJustify {
		scaleFontSize(r, *lines, td.ScaleAbs, td.Scale, width, *x, *y, mLeft, mRight, borderWidth, td.FontName, fontSize)
	}

	// Apply vertical alignment.
	var dy1 float64
	switch td.VAlign {
	case types.AlignTop:
		dy1 = deltaAlignTop(td.FontName, *fontSize, mTop+borderWidth)
	case types.AlignMiddle:
		dy1 = deltaAlignMiddle(td.FontName, *fontSize, len(*lines), mTop, mBot)
	case types.AlignBottom:
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

func flushJustifiedStringToBuf(w io.Writer, s string, x, y float64, strokeCol, fillCol color.SimpleColor, rm draw.RenderMode) {
	fmt.Fprintf(w, "BT 0 Tw %.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td %d Tr %s ET ",
		strokeCol.R, strokeCol.G, strokeCol.B, fillCol.R, fillCol.G, fillCol.B, x, y, rm, s)
}

func scaleXForRegion(x float64, mediaBox, region *types.Rectangle) float64 {
	return x / mediaBox.Width() * region.Width()
}

func scaleYForRegion(y float64, mediaBox, region *types.Rectangle) float64 {
	return y / mediaBox.Width() * region.Width()
}

func DrawMargins(w io.Writer, c color.SimpleColor, colBB *types.Rectangle, borderWidth, mLeft, mRight, mTop, mBot float64) {
	if mLeft <= 0 && mRight <= 0 && mTop <= 0 && mBot <= 0 {
		return
	}

	var r *types.Rectangle

	if mBot > 0 {
		r = types.RectForWidthAndHeight(colBB.LL.X+borderWidth, colBB.LL.Y+borderWidth, colBB.Width()-2*borderWidth, mBot)
		draw.FillRectNoBorder(w, r, c)
	}

	if mTop > 0 {
		r = types.RectForWidthAndHeight(colBB.LL.X+borderWidth, colBB.UR.Y-borderWidth-mTop, colBB.Width()-2*borderWidth, mTop)
		draw.FillRectNoBorder(w, r, c)
	}

	if mLeft > 0 {
		r = types.RectForWidthAndHeight(colBB.LL.X+borderWidth, colBB.LL.Y+borderWidth+mBot, mLeft, colBB.Height()-2*borderWidth-mTop-mBot)
		draw.FillRectNoBorder(w, r, c)
	}

	if mRight > 0 {
		r = types.RectForWidthAndHeight(colBB.UR.X-borderWidth-mRight, colBB.LL.Y+borderWidth+mBot, mRight, colBB.Height()-2*borderWidth-mTop-mBot)
		draw.FillRectNoBorder(w, r, c)
	}

}

func renderBackgroundAndBorder(w io.Writer, td TextDescriptor, borderWidth float64, colBB *types.Rectangle) {
	r := types.RectForWidthAndHeight(colBB.LL.X+borderWidth/2, colBB.LL.Y+borderWidth/2, colBB.Width()-borderWidth, colBB.Height()-borderWidth)
	if td.ShowBackground {
		c := td.BackgroundCol
		if td.ShowBorder {
			c = td.BorderCol
		}
		draw.FillRect(w, r, borderWidth, &c, td.BackgroundCol, &td.BorderStyle)
	} else if td.ShowBorder {
		draw.DrawRect(w, r, borderWidth, &td.BorderCol, &td.BorderStyle)
	}
}

func renderText(xRefTable *XRefTable, w io.Writer, lines []string, td TextDescriptor, x, y float64, fontSize int) {
	lh := font.LineHeight(td.FontName, fontSize)
	for _, s := range lines {
		if td.HAlign != types.AlignJustify {
			lineBB := CalcBoundingBox(s, x, y, td.FontName, fontSize)
			// Apply horizontal alignment.
			var dx float64
			switch td.HAlign {
			case types.AlignCenter:
				dx = lineBB.Width() / 2
			case types.AlignRight:
				dx = lineBB.Width()
			}
			lineBB.Translate(-dx, 0)
			if td.ShowLineBB {
				// Draw line bounding box.
				draw.SetStrokeColor(w, color.Black)
				draw.DrawRectSimple(w, lineBB)
			}
			writeStringToBuf(xRefTable, w, s, x-dx, y, td)
			y -= lh
			continue
		}

		if len(s) > 0 {
			flushJustifiedStringToBuf(w, s, x, y, td.StrokeCol, td.FillCol, td.RMode)
		}
		y -= lh
	}
}

// This is a patched version of strings.FieldsFunc that also returns empty fields.
func fieldsFunc(s string, f func(rune) bool) []string {
	// A span is used to record a slice of s of the form s[start:end].
	// The start index is inclusive and the end index is exclusive.
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	wasField := false
	fromIndex := 0
	for i, rune := range s {
		if f(rune) {
			if wasField {
				spans = append(spans, span{start: fromIndex, end: i})
				wasField = false
			} else {
				spans = append(spans, span{})
			}
		} else {
			if !wasField {
				fromIndex = i
				wasField = true
			}
		}
	}

	// Last field might end at EOF.
	if wasField {
		spans = append(spans, span{fromIndex, len(s)})
	}

	// Create strings from recorded field indices.
	a := make([]string, len(spans))
	for i, span := range spans {
		a[i] = s[span.start:span.end]
	}

	return a
}

func SplitMultilineStr(s string) []string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	var lines []string
	return append(lines, fieldsFunc(s, func(c rune) bool { return c == 0x0a })...)
}

// WriteColumn writes a text column using s at position x/y using a certain font, fontsize and a desired horizontal and vertical alignment.
// Enforce a desired column width by supplying a width > 0 (especially useful for justified text).
// It returns the bounding box of this column.
func WriteColumn(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, width float64) *types.Rectangle {
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

	if !td.ScaleAbs {
		if td.Scale > 1 {
			td.Scale = 1
		}
	}

	// Create bounding box and prerender content stream bytes for justified text.
	colBB := createBoundingBoxForColumn(xRefTable,
		r, &x, &y, width, td, dx, dy, mTop, mBot, mLeft, mRight, borderWidth, &fontSize, &lines)

	fmt.Fprint(w, "q ")

	setFont(w, td.FontKey, float32(fontSize))
	m := matrix.CalcRotateTransformMatrix(td.Rotation, colBB)
	fmt.Fprintf(w, "%.5f %.5f %.5f %.5f %.5f %.5f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	x -= colBB.LL.X
	y -= colBB.LL.Y
	colBB.Translate(-colBB.LL.X, -colBB.LL.Y)

	// Render background and border.
	if td.ShowTextBB {
		renderBackgroundAndBorder(w, td, borderWidth, colBB)
	}

	// Render margins
	if td.ShowMargins {
		DrawMargins(w, color.LightGray, colBB, borderWidth, mLeft, mRight, mTop, mBot)
	}

	// Render text.
	renderText(xRefTable, w, lines, td, x, y, fontSize)

	fmt.Fprintf(w, "Q ")

	if td.HairCross {
		draw.DrawHairCross(w, x0, y0, r)
	}

	if td.ShowPosition {
		draw.DrawCircle(w, x0, y0, 5, color.Black, &color.Red)
	}

	return colBB
}

// WriteMultiLine writes s at position x/y using a certain font, fontsize and a desired horizontal and vertical alignment.
// It returns the bounding box of this text column.
func WriteMultiLine(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor) *types.Rectangle {
	return WriteColumn(xRefTable, w, mediaBox, region, td, 0)
}

// AnchorPosAndAlign calculates position and alignment for an anchored rectangle r.
func AnchorPosAndAlign(a types.Anchor, r *types.Rectangle) (x, y float64, hAlign types.HAlignment, vAlign types.VAlignment) {
	switch a {
	case types.TopLeft:
		x, y, hAlign, vAlign = 0, r.Height(), types.AlignLeft, types.AlignTop
	case types.TopCenter:
		x, y, hAlign, vAlign = -1, r.Height(), types.AlignCenter, types.AlignTop
	case types.TopRight:
		x, y, hAlign, vAlign = r.Width(), r.Height(), types.AlignRight, types.AlignTop
	case types.Left:
		x, y, hAlign, vAlign = 0, -1, types.AlignLeft, types.AlignMiddle
	case types.Center:
		x, y, hAlign, vAlign = -1, -1, types.AlignCenter, types.AlignMiddle
	case types.Right:
		x, y, hAlign, vAlign = r.Width(), -1, types.AlignRight, types.AlignMiddle
	case types.BottomLeft:
		x, y, hAlign, vAlign = 0, 0, types.AlignLeft, types.AlignMiddle
	case types.BottomCenter:
		x, y, hAlign, vAlign = -1, 0, types.AlignCenter, types.AlignMiddle
	case types.BottomRight:
		x, y, hAlign, vAlign = r.Width(), 0, types.AlignRight, types.AlignMiddle
	}
	return
}

// WriteMultiLineAnchored writes multiple lines with anchored position and returns its bounding box.
func WriteMultiLineAnchored(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, a types.Anchor) *types.Rectangle {
	r := mediaBox
	if region != nil {
		r = region
	}
	td.X, td.Y, td.HAlign, td.VAlign = AnchorPosAndAlign(a, r)
	return WriteMultiLine(xRefTable, w, mediaBox, region, td)
}

// WriteColumnAnchored writes a justified text column with anchored position and returns its bounding box.
func WriteColumnAnchored(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, a types.Anchor, width float64) *types.Rectangle {
	r := mediaBox
	if region != nil {
		r = region
	}
	td.HAlign = types.AlignJustify
	td.X, td.Y, _, td.VAlign = AnchorPosAndAlign(a, r)
	return WriteColumn(xRefTable, w, mediaBox, region, td, width)
}

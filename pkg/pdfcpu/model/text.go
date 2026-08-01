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
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/matrix"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type textRenderWriter struct {
	io.Writer
	err error
}

func (w *textRenderWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = err
	}
	return n, err
}

// TextDescriptor contains all attributes needed for rendering a text column in PDF user space.
type TextDescriptor struct {
	Text           string              // A multi line string using \n for line breaks.
	FontName       string              // Name of the core or user font to be used.
	RTL            bool                // Right to left user font.
	Embed          bool                // Embed font.
	FontKey        string              // Resource id registered for FontName.
	FontSize       float64             // Fontsize in points.
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

func fontVerticalMetrics(fontName string, fontSize float64) (float64, float64, error) {
	bb, err := font.BoundingBox(fontName)
	if err != nil {
		return 0, 0, fmt.Errorf("font %s: vertical metrics: %w", fontName, err)
	}
	ascent := font.UserSpaceUnitsFloat(bb.Height()+bb.LL.Y, fontSize)
	lineHeight := font.UserSpaceUnitsFloat(bb.Height(), fontSize)
	return ascent, lineHeight, nil
}

func deltaAlignMiddle(fontName string, fontSize float64, lines int, mTop, mBot float64) (float64, error) {
	ascent, lineHeight, err := fontVerticalMetrics(fontName, fontSize)
	if err != nil {
		return 0, err
	}
	return -ascent + (float64(lines)*lineHeight+mTop+mBot)/2 - mTop, nil
}

func deltaAlignTop(fontName string, fontSize, mTop float64) (float64, error) {
	ascent, _, err := fontVerticalMetrics(fontName, fontSize)
	if err != nil {
		return 0, err
	}
	return -ascent - mTop, nil
}

func deltaAlignBottom(fontName string, fontSize float64, lines int, mBot float64) (float64, error) {
	ascent, lineHeight, err := fontVerticalMetrics(fontName, fontSize)
	if err != nil {
		return 0, err
	}
	return -ascent + float64(lines)*lineHeight + mBot, nil
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

// DecodeUTF8ToByte decodes utf8 to byte.
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

// CalcBoundingBoxForRects calculates bounding box for rects.
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

func calcBoundingBoxForLines(lines []string, x, y float64, fontName string, fontSize float64) (*types.Rectangle, string, error) {
	if len(lines) == 0 {
		return nil, "", errors.New("calculate text bounding box: no text lines")
	}
	var (
		box      *types.Rectangle
		maxLine  string
		maxWidth float64
	)
	for i, s := range lines {
		bbox, err := CalcBoundingBoxFloat(s, x, y, fontName, fontSize)
		if err != nil {
			return nil, "", fmt.Errorf("line %d: %w", i+1, err)
		}
		if bbox.Width() > maxWidth {
			maxWidth = bbox.Width()
			maxLine = s
		}
		box = CalcBoundingBoxForRects(box, bbox)
		y -= bbox.Height()
	}
	return box, maxLine, nil
}

func encodeUserFontRunes(s string) string {
	bb := make([]byte, 0, utf8.RuneCountInString(s)*2)
	for _, r := range s {
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(r))
		bb = append(bb, b...)
	}
	return string(bb)
}

func prepareEmbeddedUserFontBytes(xRefTable *XRefTable, s, fontName string) (string, error) {
	if xRefTable == nil {
		return "", fmt.Errorf("font %s: prepare embedded text: %w", fontName, ErrMissingXRefTable)
	}
	if xRefTable.UsedGIDs == nil {
		xRefTable.UsedGIDs = map[string]map[uint16]bool{}
	}
	usedGIDs, ok := xRefTable.UsedGIDs[fontName]
	if !ok {
		usedGIDs = map[uint16]bool{}
		xRefTable.UsedGIDs[fontName] = usedGIDs
	}
	ttf, ok, err := font.UserFont(fontName)
	if err != nil {
		return "", fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return "", fmt.Errorf("font %s: metrics not found: %w", fontName, font.ErrUnknownFont)
	}
	bb := make([]byte, 0, utf8.RuneCountInString(s)*2)
	for _, r := range s {
		gid, ok := ttf.Chars[uint32(r)]
		if !ok {
			continue
		}
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, gid)
		bb = append(bb, b...)
		usedGIDs[gid] = true
	}
	return string(bb), nil
}

func prepareUserFontBytes(xRefTable *XRefTable, s, fontName string, embed, rtl, fillFont bool) (string, error) {
	if fillFont && embed {
		return s, nil
	}
	if rtl {
		s = types.Reverse(s)
	}
	if !embed {
		return encodeUserFontRunes(s), nil
	}
	return prepareEmbeddedUserFontBytes(xRefTable, s, fontName)
}

// PrepBytes prepares bytes for s and fontName.
func PrepBytes(xRefTable *XRefTable, s, fontName string, embed, rtl, fillFont bool) (string, error) {
	if strings.TrimSpace(fontName) == "" {
		return "", fmt.Errorf("prepare text bytes: %w", font.ErrMissingFontName)
	}
	userFont, err := font.IsUserFont(fontName)
	if err != nil {
		return "", fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !userFont && !font.IsCoreFont(fontName) {
		return "", fmt.Errorf("font %s: prepare text bytes: %w", fontName, font.ErrUnknownFont)
	}
	if userFont {
		s, err = prepareUserFontBytes(xRefTable, s, fontName, embed, rtl, fillFont)
		if err != nil {
			return "", err
		}
	}
	s1, err := types.Escape(s)
	if err != nil {
		return "", fmt.Errorf("font %s: escape text: %w", fontName, err)
	}
	return *s1, nil
}

func writeStringToBuf(xRefTable *XRefTable, w io.Writer, s string, x, y float64, td TextDescriptor) error {
	s, err := PrepBytes(xRefTable, s, td.FontName, td.Embed, td.RTL, false)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "BT 0 Tw %.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td %d Tr (%s) Tj ET ",
		td.StrokeCol.R, td.StrokeCol.G, td.StrokeCol.B, td.FillCol.R, td.FillCol.G, td.FillCol.B, x, y, td.RMode, s); err != nil {
		return fmt.Errorf("write text string: %w", err)
	}
	return nil
}

func setFont(w io.Writer, fontID string, fontSize float64) {
	fmt.Fprintf(w, "BT /%s %s Tf ET ", fontID, strconv.FormatFloat(fontSize, 'f', -1, 64))
}

// CalcBoundingBox calculates a bounding box.
func CalcBoundingBox(s string, x, y float64, fontName string, fontSize int) (*types.Rectangle, error) {
	return CalcBoundingBoxFloat(s, x, y, fontName, float64(fontSize))
}

// CalcBoundingBoxFloat calculates a bounding box using a fractional font size.
func CalcBoundingBoxFloat(s string, x, y float64, fontName string, fontSize float64) (*types.Rectangle, error) {
	w, err := font.TextWidthFloat(s, fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("font %s: text width: %w", fontName, err)
	}
	fbb, err := font.BoundingBox(fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: bounding box: %w", fontName, err)
	}
	h := font.UserSpaceUnitsFloat(fbb.Height(), fontSize)
	y -= math.Ceil(font.UserSpaceUnitsFloat(-fbb.LL.Y, fontSize))
	return types.NewRectangle(x, y, x+w, y+h), nil
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

func prepJustifiedLine(xRefTable *XRefTable, lines *[]string, strbuf []string, strWidth, w, fontSize float64, fontName string, embed, rtl bool) error {
	blank, err := PrepBytes(xRefTable, " ", fontName, embed, true, false)
	if err != nil {
		return fmt.Errorf("prepare justified blank: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("[")
	wc := len(strbuf)
	var dx float64
	if wc > 1 {
		dx = font.GlyphSpaceUnitsFloat((w-strWidth)/float64(wc-1), fontSize)
	}
	for i := 0; i < wc; i++ {
		j := i
		if rtl {
			j = wc - 1 - i
		}
		s, err := PrepBytes(xRefTable, strbuf[j], fontName, embed, rtl, false)
		if err != nil {
			return fmt.Errorf("prepare justified word %d: %w", j+1, err)
		}
		sb.WriteString(fmt.Sprintf(" (%s)", s))
		if i < wc-1 {
			sb.WriteString(fmt.Sprintf(" %d (%s)", -int(dx), blank))
		}
	}
	sb.WriteString(" ] TJ")
	*lines = append(*lines, sb.String())
	return nil
}

type justifiedTextPreparer struct {
	xRefTable  *XRefTable
	strbuf     []string
	strWidth   float64
	indent     bool
	blankWidth float64
}

func newJustifiedTextPreparer(xRefTable *XRefTable, fontName string, fontSize float64) (*justifiedTextPreparer, error) {
	blankWidth, err := font.TextWidthFloat(" ", fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("font %s: justified blank width: %w", fontName, err)
	}
	return &justifiedTextPreparer{xRefTable: xRefTable, indent: true, blankWidth: blankWidth}, nil
}

func (p *justifiedTextPreparer) flush(lines *[]string, w float64, fontName string, fontSize *float64, lastline, parIndent, embed, rtl bool) (int, error) {
	if len(p.strbuf) > 0 {
		s, err := PrepBytes(p.xRefTable, strings.Join(p.strbuf, " "), fontName, embed, rtl, false)
		if err != nil {
			return 0, fmt.Errorf("prepare final justified line: %w", err)
		}
		if rtl {
			dx := font.GlyphSpaceUnitsFloat(w-p.strWidth, *fontSize)
			s = fmt.Sprintf("[ %d (%s) ] TJ ", -int(dx), s)
		} else {
			s = fmt.Sprintf("(%s) Tj", s)
		}
		*lines = append(*lines, s)
		p.strbuf = nil
		p.strWidth = 0
	}
	if lastline {
		return 0, nil
	}
	p.indent = true
	if parIndent {
		return 0, nil
	}
	return 1, nil
}

func (p *justifiedTextPreparer) add(lines *[]string, s string, w float64, fontName string, fontSize *float64, parIndent, embed, rtl bool) (int, error) {
	ss := strings.Split(s, " ")
	if parIndent && len(p.strbuf) == 0 && p.indent {
		ss[0] = "    " + ss[0]
	}
	for _, word := range ss {
		wordWidth, err := font.TextWidthFloat(word, fontName, *fontSize)
		if err != nil {
			return 0, fmt.Errorf("font %s: justified word width: %w", fontName, err)
		}
		blankWidth := 0.
		if len(p.strbuf) > 0 {
			blankWidth = p.blankWidth
		}
		if w-p.strWidth-(wordWidth+blankWidth) > 0 {
			p.strWidth += wordWidth + blankWidth
			p.strbuf = append(p.strbuf, word)
			continue
		}
		size, err := font.Size(word, fontName, w)
		if err != nil {
			return 0, fmt.Errorf("font %s: fit justified word: %w", fontName, err)
		}
		if float64(size) < *fontSize {
			*fontSize = float64(size)
		}
		if len(p.strbuf) == 0 {
			err = prepJustifiedLine(p.xRefTable, lines, []string{word}, wordWidth, w, *fontSize, fontName, embed, rtl)
		} else {
			err = prepJustifiedLine(p.xRefTable, lines, p.strbuf, p.strWidth, w, *fontSize, fontName, embed, rtl)
			if err != nil {
				return 0, err
			}
			p.strbuf = []string{word}
			p.strWidth = wordWidth
		}
		if err != nil {
			return 0, err
		}
		p.indent = false
	}
	return 0, nil
}

func (p *justifiedTextPreparer) prepare(lines *[]string, s string, w float64, fontName string, fontSize *float64, lastline, parIndent, embed, rtl bool) (int, error) {
	if len(s) == 0 {
		return p.flush(lines, w, fontName, fontSize, lastline, parIndent, embed, rtl)
	}
	return p.add(lines, s, w, fontName, fontSize, parIndent, embed, rtl)
}

// Prerender justified text in order to calculate bounding box height.
func preRenderJustifiedText(
	xRefTable *XRefTable,
	lines *[]string,
	r *types.Rectangle,
	x, y, width float64,
	td TextDescriptor,
	mLeft, mRight, borderWidth float64,
	fontSize *float64) (float64, error) {

	var ww float64
	if !td.ScaleAbs {
		ww = r.Width() * td.Scale
	} else {
		if width > 0 {
			ww = width * td.Scale
		} else {
			box, _, err := calcBoundingBoxForLines(*lines, x, y, td.FontName, *fontSize)
			if err != nil {
				return 0, err
			}
			ww = box.Width() * td.Scale
		}
	}
	ww -= mLeft + mRight + 2*borderWidth
	preparer, err := newJustifiedTextPreparer(xRefTable, td.FontName, *fontSize)
	if err != nil {
		return 0, err
	}
	l := []string{}
	for i, s := range *lines {
		linefeeds, err := preparer.prepare(&l, s, ww, td.FontName, fontSize, false, td.ParIndent, td.Embed, td.RTL)
		if err != nil {
			return 0, fmt.Errorf("justify line %d: %w", i+1, err)
		}
		for j := 0; j < linefeeds; j++ {
			l = append(l, "")
		}
		isLastLine := i == len(*lines)-1
		if isLastLine {
			if _, err := preparer.prepare(&l, "", ww, td.FontName, fontSize, true, td.ParIndent, td.Embed, td.RTL); err != nil {
				return 0, fmt.Errorf("finalize justified text: %w", err)
			}
		}
	}
	*lines = l
	return ww, nil
}

func scaleFontSize(r *types.Rectangle, lines []string, scaleAbs bool,
	scale, width, x, y, mLeft, mRight, borderWidth float64,
	fontName string, fontSize *float64) error {
	if scaleAbs {
		*fontSize *= scale
	} else {
		www := width
		if width == 0 {
			box, _, err := calcBoundingBoxForLines(lines, x, y, fontName, *fontSize)
			if err != nil {
				return err
			}
			www = box.Width() + mLeft + mRight + 2*borderWidth
		}
		*fontSize = r.Width() * scale * *fontSize / www
	}
	return nil
}

func horizontalWrapUp(box *types.Rectangle, maxLine string, hAlign types.HAlignment,
	x *float64, width, ww, mLeft, mRight, borderWidth float64,
	fontName string, fontSize *float64) error {
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
			size, err := font.Size(maxLine, fontName, netWidth)
			if err != nil {
				return fmt.Errorf("font %s: fit aligned text: %w", fontName, err)
			}
			*fontSize = float64(size)
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
	return nil
}

func createBoundingBoxForColumn(xRefTable *XRefTable, r *types.Rectangle, x, y *float64,
	width float64,
	td TextDescriptor,
	dx, dy float64,
	mTop, mBot, mLeft, mRight float64,
	borderWidth float64,
	fontSize *float64, lines *[]string) (*types.Rectangle, error) {

	var ww float64
	if td.HAlign == types.AlignJustify {
		var err error
		ww, err = preRenderJustifiedText(xRefTable, lines, r, *x, *y, width, td, mLeft, mRight, borderWidth, fontSize)
		if err != nil {
			return nil, fmt.Errorf("prepare justified text: %w", err)
		}
	}

	if td.HAlign != types.AlignJustify {
		if err := scaleFontSize(r, *lines, td.ScaleAbs, td.Scale, width, *x, *y, mLeft, mRight, borderWidth, td.FontName, fontSize); err != nil {
			return nil, fmt.Errorf("scale text: %w", err)
		}
	}

	// Apply vertical alignment.
	var dy1 float64
	var err error
	switch td.VAlign {
	case types.AlignTop:
		dy1, err = deltaAlignTop(td.FontName, *fontSize, mTop+borderWidth)
	case types.AlignMiddle:
		dy1, err = deltaAlignMiddle(td.FontName, *fontSize, len(*lines), mTop, mBot)
	case types.AlignBottom:
		dy1, err = deltaAlignBottom(td.FontName, *fontSize, len(*lines), mBot)
	}
	if err != nil {
		return nil, fmt.Errorf("align text vertically: %w", err)
	}
	*y += math.Ceil(dy1)

	box, maxLine, err := calcBoundingBoxForLines(*lines, *x, *y, td.FontName, *fontSize)
	if err != nil {
		return nil, fmt.Errorf("measure column lines: %w", err)
	}
	// maxLine for hAlign != AlignJustify only!
	if err := horizontalWrapUp(box, maxLine, td.HAlign, x, width, ww, mLeft, mRight, borderWidth, td.FontName, fontSize); err != nil {
		return nil, fmt.Errorf("align text horizontally: %w", err)
	}

	box.LL.Y -= mBot + borderWidth
	box.UR.Y += mTop + borderWidth

	if td.MinHeight > 0 && box.Height() < td.MinHeight {
		box.LL.Y = box.UR.Y - td.MinHeight
	}

	horAdjustBoundingBoxForLines(r, box, dx, dy, x, y)

	return box, nil
}

func flushJustifiedStringToBuf(w io.Writer, s string, x, y float64, strokeCol, fillCol color.SimpleColor, rm draw.RenderMode) error {
	if _, err := fmt.Fprintf(w, "BT 0 Tw %.2f %.2f %.2f RG %.2f %.2f %.2f rg %.2f %.2f Td %d Tr %s ET ",
		strokeCol.R, strokeCol.G, strokeCol.B, fillCol.R, fillCol.G, fillCol.B, x, y, rm, s); err != nil {
		return fmt.Errorf("write justified text: %w", err)
	}
	return nil
}

func scaleXForRegion(x float64, mediaBox, region *types.Rectangle) float64 {
	return x / mediaBox.Width() * region.Width()
}

func scaleYForRegion(y float64, mediaBox, region *types.Rectangle) float64 {
	return y / mediaBox.Width() * region.Width()
}

// DrawMargins draws margins.
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

func renderText(xRefTable *XRefTable, w io.Writer, lines []string, td TextDescriptor, x, y, fontSize float64) error {
	_, lh, err := fontVerticalMetrics(td.FontName, fontSize)
	if err != nil {
		return fmt.Errorf("font %s: resolve vertical metrics: %w", td.FontName, err)
	}
	for i, s := range lines {
		if td.HAlign != types.AlignJustify {
			lineBB, err := CalcBoundingBoxFloat(s, x, y, td.FontName, fontSize)
			if err != nil {
				return fmt.Errorf("line %d bounding box: %w", i+1, err)
			}
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
			if err := writeStringToBuf(xRefTable, w, s, x-dx, y, td); err != nil {
				return fmt.Errorf("line %d bytes: %w", i+1, err)
			}
			y -= lh
			continue
		}

		if len(s) > 0 {
			if err := flushJustifiedStringToBuf(w, s, x, y, td.StrokeCol, td.FillCol, td.RMode); err != nil {
				return fmt.Errorf("line %d justified text: %w", i+1, err)
			}
		}
		y -= lh
	}
	return nil
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

// SplitMultilineStr splits a  multiline string.
func SplitMultilineStr(s string) []string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	var lines []string
	return append(lines, fieldsFunc(s, func(c rune) bool { return c == 0x0a })...)
}

func textFits(candidate, fontName string, fontSize, maxWidthPoints float64) (bool, error) {
	width, err := font.TextWidthFloat(candidate, fontName, fontSize)
	if err != nil {
		return false, fmt.Errorf("font %s: wrap candidate: %w", fontName, err)
	}
	return width < maxWidthPoints, nil
}

func wrapLine(ss *[]string, line, space, word, fontName string, fontSize, maxWidthPoints float64) error {
	candidate := line + space + word
	fits, err := textFits(candidate, fontName, fontSize, maxWidthPoints)
	if err != nil {
		return err
	}
	if fits {
		*ss = append(*ss, candidate)
	} else {
		if len(line) > 0 {
			*ss = append(*ss, line)
		}
		*ss = append(*ss, word)
	}
	return nil
}

func wrapWord(ss *[]string, line, space, word, nextSpace, fontName string, fontSize, maxWidthPoints float64) (string, string, error) {
	candidate := line + space + word
	fits, err := textFits(candidate, fontName, fontSize, maxWidthPoints)
	if err != nil {
		return "", "", err
	}
	if fits {
		return candidate, nextSpace, nil
	}
	if len(line) > 0 {
		*ss = append(*ss, line)
		return word, space, nil
	}
	*ss = append(*ss, word)
	return line, "", nil
}

func wrap(lines []string, fontName string, fontSize, maxWidthPoints float64) ([]string, error) {

	var wrapState int

	const (
		beginLine = iota
		inWord
		leadingSpace
		inSpace
	)

	var ss []string

	for lineIndex, s := range lines {

		var word, space, line string

		wrapState = beginLine

		for _, c := range s {

			switch wrapState {

			case beginLine:
				if unicode.IsSpace(c) {
					line = string(c)
					wrapState = leadingSpace
				} else {
					word = string(c)
					wrapState = inWord
				}

			case leadingSpace:
				if unicode.IsSpace(c) {
					line += string(c)
				} else {
					word = string(c)
					wrapState = inWord
				}

			case inWord:
				if unicode.IsSpace(c) {
					var err error
					line, space, err = wrapWord(&ss, line, space, word, string(c), fontName, fontSize, maxWidthPoints)
					if err != nil {
						return nil, fmt.Errorf("line %d: %w", lineIndex+1, err)
					}
					wrapState = inSpace
				} else if len(word) > 0 && canBreakAfterChar(lastRune(word)) && canBreakBeforeChar(c) {
					fullCandidate := line + space + word + string(c)

					getTextWidth := func(text string) float64 {
						tx, err := font.TextWidth(text, fontName, int(fontSize))
						if err != nil {
							return 0
						}

						return tx
					}

					if getTextWidth(fullCandidate) <= maxWidthPoints {
						word += string(c)

					} else if getTextWidth(word+string(c)) <= maxWidthPoints {
						word += string(c)

					} else {
						if len(line) > 0 {
							ss = append(ss, line)
						}
						if getTextWidth(word) > maxWidthPoints {
							wrapLine(&ss, "", "", word, fontName, fontSize, maxWidthPoints)
						} else {
							ss = append(ss, word)
						}
						line = ""
						space = ""
						word = string(c)
					}
				} else {
					word += string(c)
				}

			case inSpace:
				if unicode.IsSpace(c) {
					space += string(c)
				} else {
					word = string(c)
					wrapState = inWord
				}
			}
		}

		if wrapState == inWord {
			if err := wrapLine(&ss, line, space, word, fontName, fontSize, maxWidthPoints); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineIndex+1, err)
			}
		}
	}

	return ss, nil
}

// lastRune get last rune of a string
func lastRune(s string) rune {
	r := rune(0)
	for _, c := range s {
		r = c
	}
	return r
}

// WordWrap wraps text at Unicode whitespace and reports font metric failures.
func WordWrap(s string, fontName string, fontSize int, maxWidthPoints float64) ([]string, error) {
	return WordWrapFloat(s, fontName, float64(fontSize), maxWidthPoints)
}

// WordWrapFloat wraps text at Unicode whitespace using a fractional font size.
// It reports font metric failures.
func WordWrapFloat(s string, fontName string, fontSize, maxWidthPoints float64) ([]string, error) {
	if len(s) == 0 || maxWidthPoints <= 0 {
		return []string{s}, nil
	}

	lines := SplitMultilineStr(s)

	ss, err := wrap(lines, fontName, fontSize, maxWidthPoints)
	if err != nil {
		return nil, err
	}

	if len(ss) == 0 {
		ss = append(ss, "")
	}

	return ss, nil
}

// canBreakAfterChar reports whether a line break is allowed immediately after r.
func canBreakAfterChar(r rune) bool {
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	if r >= 0x20000 && r <= 0x2A6DF {
		return true
	}
	if r >= 0x3040 && r <= 0x309F {
		return true
	}
	if r >= 0x30A0 && r <= 0x30FF {
		return true
	}
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	if r >= 0x1100 && r <= 0x11FF {
		return true
	}
	if r >= 0x3000 && r <= 0x303F {
		return !isOpeningPunct(r)
	}
	if r >= 0xFF00 && r <= 0xFFEF {
		return !isOpeningPunct(r)
	}
	return false
}

// canBreakBeforeChar reports whether a line break is allowed immediately before r.
func canBreakBeforeChar(r rune) bool {
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	if r >= 0x20000 && r <= 0x2A6DF {
		return true
	}
	if r >= 0x3040 && r <= 0x309F {
		return true
	}
	if r >= 0x30A0 && r <= 0x30FF {
		return true
	}
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	if r >= 0x1100 && r <= 0x11FF {
		return true
	}
	if r >= 0x3000 && r <= 0x303F {
		return !isClosingPunct(r)
	}
	if r >= 0xFF00 && r <= 0xFFEF {
		return !isClosingPunct(r)
	}
	return false
}

// isOpeningPunct reports whether r is an opening punctuation mark.
func isOpeningPunct(r rune) bool {
	switch r {
	case 0x3008, 0x300A, 0x300C, 0x300E, 0x3010, 0x3014, 0x3016, 0x3018, 0x301A, 0x301D:
		return true
	case 0xFF02, 0xFF07, 0xFF08, 0xFF3B, 0xFF40, 0xFF5B:
		return true
	case 0x2018, 0x201A, 0x201C, 0x201E, 0x2039, 0x00AB:
		return true
	}
	return false
}

// isClosingPunct reports whether r is a closing punctuation mark.
func isClosingPunct(r rune) bool {
	switch r {
	case 0x3001, 0x3002, 0x3009, 0x300B, 0x300D, 0x300F, 0x3011, 0x3015, 0x3017, 0x3019, 0x301B, 0x301E, 0x301F:
		return true
	case 0xFF01, 0xFF02, 0xFF07, 0xFF09, 0xFF0C, 0xFF0E, 0xFF1A, 0xFF1B, 0xFF1F, 0xFF3D, 0xFF5D:
		return true
	case 0x2013, 0x2014, 0x2019, 0x201D, 0x2026, 0x203A, 0x00BB:
		return true
	case 0x0022, 0x0027:
		return true
	}
	return false
}

func scaleTextColumnForRegion(
	mediaBox, region *types.Rectangle,
	dx, dy, width *float64,
	fontSize *float64,
	mTop, mBot, mLeft, mRight, borderWidth *float64,
) *types.Rectangle {
	if region == nil {
		return mediaBox
	}
	*dx = scaleXForRegion(*dx, mediaBox, region)
	*dy = scaleYForRegion(*dy, mediaBox, region)
	*width = scaleXForRegion(*width, mediaBox, region)
	*fontSize = scaleYForRegion(*fontSize, mediaBox, region)
	*mTop = scaleYForRegion(*mTop, mediaBox, region)
	*mBot = scaleYForRegion(*mBot, mediaBox, region)
	*mLeft = scaleXForRegion(*mLeft, mediaBox, region)
	*mRight = scaleXForRegion(*mRight, mediaBox, region)
	*borderWidth = scaleXForRegion(*borderWidth, mediaBox, region)
	return region
}

func positionTextColumn(r *types.Rectangle, x, y, dx, dy float64) (float64, float64) {
	if x >= 0 {
		x = r.LL.X + x
	} else {
		x = r.LL.X + r.Width()/2
	}
	if y >= 0 {
		y = r.LL.Y + y
	} else {
		y = r.LL.Y + r.Height()/2
	}
	return x + dx, y + dy
}

func writeColumn(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, width float64) (*types.Rectangle, error) {
	renderWriter := &textRenderWriter{Writer: w}
	w = renderWriter
	x, y, dx, dy := td.X, td.Y, td.Dx, td.Dy
	mTop, mBot, mLeft, mRight := td.MTop, td.MBot, td.MLeft, td.MRight
	s, fontSize, borderWidth := td.Text, td.FontSize, td.BorderWidth

	r := scaleTextColumnForRegion(mediaBox, region, &dx, &dy, &width, &fontSize, &mTop, &mBot, &mLeft, &mRight, &borderWidth)
	x, y = positionTextColumn(r, x, y, dx, dy)

	// Cache haircross coordinates.
	x0, y0 := x, y

	if font.IsCoreFont(td.FontName) && utf8.ValidString(s) {
		s = DecodeUTF8ToByte(s)
	}

	lines := SplitMultilineStr(s)

	if width > 0 {
		var err error
		lines, err = wrap(lines, td.FontName, fontSize, width)
		if err != nil {
			return nil, fmt.Errorf("create column bounding box: measure column lines: %w", err)
		}
	}

	if !td.ScaleAbs {
		if td.Scale > 1 {
			td.Scale = 1
		}
	}

	// Create bounding box and prerender content stream bytes for justified text.
	colBB, err := createBoundingBoxForColumn(xRefTable,
		r, &x, &y, width, td, dx, dy, mTop, mBot, mLeft, mRight, borderWidth, &fontSize, &lines)
	if err != nil {
		return nil, fmt.Errorf("font %s: create column bounding box: %w", td.FontName, err)
	}

	fmt.Fprint(w, "q ")

	setFont(w, td.FontKey, fontSize)
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
	if err := renderText(xRefTable, w, lines, td, x, y, fontSize); err != nil {
		return nil, errors.Join(fmt.Errorf("font %s: render column text: %w", td.FontName, err), renderWriter.err)
	}

	fmt.Fprintf(w, "Q ")

	if td.HairCross {
		draw.DrawHairCross(w, x0, y0, r)
	}

	if td.ShowPosition {
		draw.DrawCircle(w, x0, y0, 5, color.Black, &color.Red)
	}

	if renderWriter.err != nil {
		return nil, fmt.Errorf("font %s: write column content: %w", td.FontName, renderWriter.err)
	}
	return colBB, nil
}

// WriteColumn writes a text column and reports rendering failures.
func WriteColumn(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, width float64) (*types.Rectangle, error) {
	if xRefTable == nil {
		return nil, fmt.Errorf("render text column: %w", ErrMissingXRefTable)
	}
	if w == nil {
		return nil, errors.New("render text column: missing writer")
	}
	if mediaBox == nil {
		return nil, errors.New("render text column: missing media box")
	}
	return writeColumn(xRefTable, w, mediaBox, region, td, width)
}

// WriteMultiLine writes multiline text and reports rendering failures.
func WriteMultiLine(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor) (*types.Rectangle, error) {
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

// WriteMultiLineAnchored writes anchored multiline text and reports rendering failures.
func WriteMultiLineAnchored(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, a types.Anchor) (*types.Rectangle, error) {
	r := mediaBox
	if region != nil {
		r = region
	}
	td.X, td.Y, td.HAlign, td.VAlign = AnchorPosAndAlign(a, r)
	return WriteMultiLine(xRefTable, w, mediaBox, region, td)
}

// WriteColumnAnchored writes an anchored justified column and reports rendering failures.
func WriteColumnAnchored(xRefTable *XRefTable, w io.Writer, mediaBox, region *types.Rectangle, td TextDescriptor, a types.Anchor, width float64) (*types.Rectangle, error) {
	r := mediaBox
	if region != nil {
		r = region
	}
	td.HAlign = types.AlignJustify
	td.X, td.Y, _, td.VAlign = AnchorPosAndAlign(a, r)
	return WriteColumn(xRefTable, w, mediaBox, region, td, width)
}

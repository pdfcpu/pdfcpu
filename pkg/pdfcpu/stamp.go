/*
Copyright 2018 The pdfcpu Authors.

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
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/types"

	"github.com/pkg/errors"
)

const stampWithBBox = false

const (
	DegToRad = math.Pi / 180
	RadToDeg = 180 / math.Pi
)

// Watermark mode
const (
	WMText = iota
	WMImage
	WMPDF
)

// Rotation along one of 2 diagonals
const (
	NoDiagonal = iota
	DiagonalLLToUR
	DiagonalULToLR
)

// RenderMode represents the text rendering mode (see 9.3.6)
type RenderMode int

// Render mode
const (
	RMFill RenderMode = iota
	RMStroke
	RMFillAndStroke
)

var (
	errNoContent    = errors.New("pdfcpu: page without content")
	errNoWatermark  = errors.New("pdfcpu: no watermarks found")
	errCorruptOCGs  = errors.New("pdfcpu: OCProperties: corrupt OCGs element")
	ErrInvalidColor = errors.New("pdfcpu: invalid color constant")
)

type watermarkParamMap map[string]func(string, *Watermark) error

// Handle applies parameter completion and if successful
// parses the parameter values into import.
func (m watermarkParamMap) Handle(paramPrefix, paramValueStr string, imp *Watermark) error {
	var param string

	// Completion support
	for k := range m {
		if !strings.HasPrefix(k, paramPrefix) {
			continue
		}
		if len(param) > 0 {
			return errors.Errorf("pdfcpu: ambiguous parameter prefix \"%s\"", paramPrefix)
		}
		param = k
	}

	if param == "" {
		return errors.Errorf("pdfcpu: unknown parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, imp)
}

var wmParamMap = watermarkParamMap{
	"aligntext":       parseTextHorAlignment,
	"backgroundcolor": parseBackgroundColor,
	"bgcolor":         parseBackgroundColor,
	"border":          parseBorder,
	"color":           parseFillColor,
	"diagonal":        parseDiagonal,
	"fillcolor":       parseFillColor,
	"fontname":        parseFontName,
	"margins":         parseMargins,
	"mode":            parseRenderMode,
	"offset":          parsePositionOffsetWM,
	"opacity":         parseOpacity,
	"points":          parseFontSize,
	"position":        parsePositionAnchorWM,
	"rendermode":      parseRenderMode,
	"rtl":             parseRightToLeft,
	"rotation":        parseRotation,
	"scalefactor":     parseScaleFactorWM,
	"strokecolor":     parseStrokeColor,
	"url":             parseURL,
}

// SimpleColor is a simple rgb wrapper.
type SimpleColor struct {
	R, G, B float32 // intensities between 0 and 1.
}

func (sc SimpleColor) String() string {
	return fmt.Sprintf("r=%1.1f g=%1.1f b=%1.1f", sc.R, sc.G, sc.B)
}

func (sc SimpleColor) Array() Array {
	return NewNumberArray(float64(sc.R), float64(sc.G), float64(sc.B))
}

// NewSimpleColor returns a SimpleColor for rgb in the form 0x00RRGGBB
func NewSimpleColor(rgb uint32) SimpleColor {
	r := float32((rgb>>16)&0xFF) / 255
	g := float32((rgb>>8)&0xFF) / 255
	b := float32(rgb&0xFF) / 255
	return SimpleColor{r, g, b}
}

// Some popular colors.
var (
	Black     = SimpleColor{}
	White     = SimpleColor{R: 1, G: 1, B: 1}
	LightGray = SimpleColor{.9, .9, .9}
	Gray      = SimpleColor{.5, .5, .5}
	DarkGray  = SimpleColor{.3, .3, .3}
	Red       = SimpleColor{1, 0, 0}
	Green     = SimpleColor{0, 1, 0}
	Blue      = SimpleColor{0, 0, 1}
)

type formCache map[types.Rectangle]*IndirectRef

type pdfResources struct {
	content []byte
	resDict *IndirectRef
	bb      *Rectangle // visible region in user space
}

// Watermark represents the basic structure and command details for the commands "Stamp" and "Watermark".
type Watermark struct {
	// configuration
	Mode              int           // WMText, WMImage or WMPDF
	TextString        string        // raw display text.
	TextLines         []string      // display multiple lines of text.
	URL               string        // overlay link annotation for stamps.
	FileName          string        // display pdf page or png image.
	Image             io.Reader     // reader for image watermark.
	Page              int           // the page number of a PDF file. 0 means multistamp/multiwatermark.
	OnTop             bool          // if true this is a STAMP else this is a WATERMARK.
	InpUnit           DisplayUnit   // input display unit.
	Pos               Anchor        // position anchor, one of tl,tc,tr,l,c,r,bl,bc,br.
	Dx, Dy            int           // anchor offset.
	HAlign            *HAlignment   // horizonal alignment for text watermarks.
	FontName          string        // supported are Adobe base fonts only. (as of now: Helvetica, Times-Roman, Courier)
	FontSize          int           // font scaling factor.
	ScaledFontSize    int           // font scaling factor for a specific page
	RTL               bool          // if true, render text from right to left
	Color             SimpleColor   // text fill color(=non stroking color) for backwards compatibility.
	FillColor         SimpleColor   // text fill color(=non stroking color).
	StrokeColor       SimpleColor   // text stroking color
	BgColor           *SimpleColor  // text bounding box background color
	MLeft, MRight     int           // left and right bounding box margin
	MTop, MBot        int           // top and bottom bounding box margin
	BorderWidth       int           // Border width, visible if BgColor is set.
	BorderStyle       LineJoinStyle // Border style (bounding box corner style), visible if BgColor is set.
	BorderColor       *SimpleColor  // border color
	Rotation          float64       // rotation to apply in degrees. -180 <= x <= 180
	Diagonal          int           // paint along the diagonal.
	UserRotOrDiagonal bool          // true if one of rotation or diagonal provided overriding the default.
	Opacity           float64       // opacity of the watermark. 0 <= x <= 1
	RenderMode        RenderMode    // fill=0, stroke=1 fill&stroke=2
	Scale             float64       // relative scale factor: 0 <= x <= 1, absolute scale factor: 0 <= x
	ScaleEff          float64       // effective scale factor
	ScaleAbs          bool          // true for absolute scaling.
	Update            bool          // true for updating instead of adding a page watermark.

	// resources
	ocg, extGState, font, image *IndirectRef

	// image or PDF watermark
	width, height int // image or page dimensions.
	bbPDF         *Rectangle

	// PDF watermark
	pdfRes map[int]pdfResources

	// page specific
	bb      *Rectangle   // bounding box of the form representing this watermark.
	bbTrans QuadLiteral  // Transformed bounding box.
	vp      *Rectangle   // page dimensions.
	pageRot int          // page rotation in effect.
	form    *IndirectRef // Forms are dependent on given page dimensions.

	// house keeping
	objs   IntSet    // objects for which wm has been applied already.
	fCache formCache // form cache.
}

// DefaultWatermarkConfig returns the default configuration.
func DefaultWatermarkConfig() *Watermark {
	return &Watermark{
		Page:        0,
		FontName:    "Helvetica",
		FontSize:    24,
		RTL:         false,
		Pos:         Center,
		Scale:       0.5,
		ScaleAbs:    false,
		Color:       Gray,
		StrokeColor: Gray,
		FillColor:   Gray,
		Diagonal:    DiagonalLLToUR,
		Opacity:     1.0,
		RenderMode:  RMFill,
		pdfRes:      map[int]pdfResources{},
		objs:        IntSet{},
		fCache:      formCache{},
		TextLines:   []string{},
	}
}

func (wm Watermark) typ() string {
	if wm.isImage() {
		return "image"
	}
	if wm.isPDF() {
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
	if wm.bb != nil {
		bbox = (*wm.bb).String()
	}

	vp := ""
	if wm.vp != nil {
		vp = (*wm.vp).String()
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
		t, s, wm.typ(),
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
		wm.pageRot,
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

func (wm Watermark) multiStamp() bool {
	return wm.Page == 0
}

func (wm Watermark) calcMaxTextWidth() float64 {
	var maxWidth float64
	for _, l := range wm.TextLines {
		w := font.TextWidth(l, wm.FontName, wm.ScaledFontSize)
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth
}

func ResolveWMTextString(text, timeStampFormat string, pageNr, pageCount int) (string, bool) {
	// replace  %p with pageNr
	//			%P with pageCount
	//			%t with timestamp
	//			%v with pdfcpu version
	var (
		bb         []byte
		hasPercent bool
		unique     bool
	)
	for i := 0; i < len(text); i++ {
		if text[i] == '%' {
			if hasPercent {
				bb = append(bb, '%')
			}
			hasPercent = true
			continue
		}
		if hasPercent {
			hasPercent = false
			if text[i] == 'p' {
				bb = append(bb, strconv.Itoa(pageNr)...)
				unique = true
				continue
			}
			if text[i] == 'P' {
				bb = append(bb, strconv.Itoa(pageCount)...)
				unique = true
				continue
			}
			if text[i] == 't' {
				bb = append(bb, time.Now().Format(timeStampFormat)...)
				unique = true
				continue
			}
			if text[i] == 'v' {
				bb = append(bb, VersionStr...)
				unique = true
				continue
			}
		}
		bb = append(bb, text[i])
	}
	return string(bb), unique
}

func (wm Watermark) textDescriptor(timestampFormat string, pageNr, pageCount int) (TextDescriptor, bool) {
	t, unique := ResolveWMTextString(wm.TextString, timestampFormat, pageNr, pageCount)
	td := TextDescriptor{
		Text:           t,
		FontName:       wm.FontName,
		FontSize:       wm.FontSize,
		Scale:          wm.Scale,
		ScaleAbs:       wm.ScaleAbs,
		RMode:          wm.RenderMode,
		StrokeCol:      wm.StrokeColor,
		FillCol:        wm.FillColor,
		ShowBackground: true,
	}
	if wm.BgColor != nil {
		td.ShowTextBB = true
		td.BackgroundCol = *wm.BgColor
	}
	return td, unique
}

func parseTextHorAlignment(s string, wm *Watermark) error {
	var a HAlignment
	switch s {
	case "l", "left":
		a = AlignLeft
	case "r", "right":
		a = AlignRight
	case "c", "center":
		a = AlignCenter
	case "j", "justify":
		a = AlignJustify
	default:
		return errors.Errorf("pdfcpu: unknown horizontal alignment (l,r,c,j): %s", s)
	}

	wm.HAlign = &a

	return nil
}

func parsePositionAnchorWM(s string, wm *Watermark) error {
	a, err := parsePositionAnchor(s)
	if err != nil {
		return err
	}
	wm.Pos = a
	return nil
}

func parsePositionOffsetWM(s string, wm *Watermark) error {
	d := strings.Split(s, " ")
	if len(d) != 2 {
		return errors.Errorf("pdfcpu: illegal position offset string: need 2 numeric values, %s\n", s)
	}

	f, err := strconv.ParseFloat(d[0], 64)
	if err != nil {
		return err
	}
	wm.Dx = int(toUserSpace(f, wm.InpUnit))

	f, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return err
	}
	wm.Dy = int(toUserSpace(f, wm.InpUnit))

	return nil
}

func parseScaleFactorWM(s string, wm *Watermark) (err error) {
	wm.Scale, wm.ScaleAbs, err = parseScaleFactor(s)
	return err
}

func parseFontName(s string, wm *Watermark) error {
	if !font.SupportedFont(s) {
		return errors.Errorf("pdfcpu: %s is unsupported, please refer to \"pdfcpu fonts list\".\n", s)
	}
	wm.FontName = s
	return nil
}

func parseURL(s string, wm *Watermark) error {
	if !wm.OnTop {
		return errors.Errorf("pdfcpu: \"url\" supported for stamps only.\n")
	}
	if !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}
	if _, err := url.ParseRequestURI(s); err != nil {
		return err
	}
	wm.URL = s
	return nil
}

func parseFontSize(s string, wm *Watermark) error {
	fs, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	wm.FontSize = fs

	return nil
}

func parseScaleFactor(s string) (float64, bool, error) {
	ss := strings.Split(s, " ")
	if len(ss) > 2 {
		return 0, false, errors.Errorf("pdfcpu: invalid scale string %s: 0.0 < i <= 1.0 {rel} | 0.0 < i {abs}\n", s)
	}

	sc, err := strconv.ParseFloat(ss[0], 64)
	if err != nil {
		return 0, false, errors.Errorf("pdfcpu: scale factor must be a float value: %s\n", ss[0])
	}

	if sc <= 0 {
		return 0, false, errors.Errorf("pdfcpu: invalid scale value %.2f: 0.0 < i <= 1.0 {rel} | 0.0 < i {abs}\n", sc)
	}

	var scaleAbs bool

	if len(ss) == 1 {
		// Assume relative scaling for sc <= 1 and absolute scaling for sc > 1.
		scaleAbs = sc > 1
		return sc, scaleAbs, nil
	}

	switch ss[1] {
	case "a", "abs", "absolute":
		scaleAbs = true

	case "r", "rel", "relative":
		scaleAbs = false

	default:
		return 0, false, errors.Errorf("pdfcpu: illegal scale mode: abs|rel, %s\n", ss[1])
	}

	if !scaleAbs && sc > 1 {
		return 0, false, errors.Errorf("pdfcpu: invalid relative scale value %.2f: 0.0 < i <= 1\n", sc)
	}

	return sc, scaleAbs, nil
}

func parseRightToLeft(s string, wm *Watermark) error {
	switch strings.ToLower(s) {
	case "on", "true", "t":
		wm.RTL = true
	case "off", "false", "f":
		wm.RTL = false
	default:
		return errors.New("pdfcpu: rtl (right-to-left), please provide one of: on/off true/false t/f")
	}

	return nil
}

func ParseHexColor(hexCol string) (SimpleColor, error) {
	var sc SimpleColor
	if len(hexCol) != 7 || hexCol[0] != '#' {
		return sc, errors.Errorf("pdfcpu: invalid hex color string: #FFFFFF, %s\n", hexCol)
	}
	b, err := hex.DecodeString(hexCol[1:])
	if err != nil || len(b) != 3 {
		return sc, errors.Errorf("pdfcpu: invalid hex color string: #FFFFFF, %s\n", hexCol)
	}
	return SimpleColor{float32(b[0]) / 255, float32(b[1]) / 255, float32(b[2]) / 255}, nil
}

func parseConstantColor(s string) (SimpleColor, error) {
	var (
		sc  SimpleColor
		err error
	)
	switch strings.ToLower(s) {
	case "black":
		sc = Black
	case "darkgray":
		sc = DarkGray
	case "gray":
		sc = Gray
	case "lightgray":
		sc = LightGray
	case "white":
		sc = White
	default:
		err = ErrInvalidColor
	}
	return sc, err
}

func ParseColor(s string) (SimpleColor, error) {
	var sc SimpleColor

	cs := strings.Split(s, " ")
	if len(cs) != 1 && len(cs) != 3 {
		return sc, errors.Errorf("pdfcpu: illegal color string: 3 intensities 0.0 <= i <= 1.0 or #FFFFFF, %s\n", s)
	}

	if len(cs) == 1 {
		if len(cs[0]) == 7 && cs[0][0] == '#' {
			// #FFFFFF to uint32
			return ParseHexColor(cs[0])
		}
		return parseConstantColor(cs[0])
	}

	r, err := strconv.ParseFloat(cs[0], 32)
	if err != nil {
		return sc, errors.Errorf("red must be a float value: %s\n", cs[0])
	}
	if r < 0 || r > 1 {
		return sc, errors.New("pdfcpu: red: a color value is an intensity between 0.0 and 1.0")
	}
	sc.R = float32(r)

	g, err := strconv.ParseFloat(cs[1], 32)
	if err != nil {
		return sc, errors.Errorf("pdfcpu: green must be a float value: %s\n", cs[1])
	}
	if g < 0 || g > 1 {
		return sc, errors.New("pdfcpu: green: a color value is an intensity between 0.0 and 1.0")
	}
	sc.G = float32(g)

	b, err := strconv.ParseFloat(cs[2], 32)
	if err != nil {
		return sc, errors.Errorf("pdfcpu: blue must be a float value: %s\n", cs[2])
	}
	if b < 0 || b > 1 {
		return sc, errors.New("pdfcpu: blue: a color value is an intensity between 0.0 and 1.0")
	}
	sc.B = float32(b)

	return sc, nil
}

func parseStrokeColor(s string, wm *Watermark) error {
	c, err := ParseColor(s)
	if err != nil {
		return err
	}
	wm.StrokeColor = c
	return nil
}

func parseFillColor(s string, wm *Watermark) error {
	c, err := ParseColor(s)
	if err != nil {
		return err
	}
	wm.FillColor = c
	return nil
}

func parseBackgroundColor(s string, wm *Watermark) error {
	c, err := ParseColor(s)
	if err != nil {
		return err
	}
	wm.BgColor = &c
	return nil
}

func parseRotation(s string, wm *Watermark) error {
	if wm.UserRotOrDiagonal {
		return errors.New("pdfcpu: please specify rotation or diagonal (r or d)")
	}

	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("pdfcpu: rotation must be a float value: %s\n", s)
	}
	if r < -180 || r > 180 {
		return errors.Errorf("pdfcpu: illegal rotation: -180 <= r <= 180 degrees, %s\n", s)
	}

	wm.Rotation = r
	wm.Diagonal = NoDiagonal
	wm.UserRotOrDiagonal = true

	return nil
}

func parseDiagonal(s string, wm *Watermark) error {
	if wm.UserRotOrDiagonal {
		return errors.New("pdfcpu: please specify rotation or diagonal (r or d)")
	}

	d, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("pdfcpu: illegal diagonal value: allowed 1 or 2, %s\n", s)
	}
	if d != DiagonalLLToUR && d != DiagonalULToLR {
		return errors.New("pdfcpu: diagonal: 1..lower left to upper right, 2..upper left to lower right")
	}

	wm.Diagonal = d
	wm.Rotation = 0
	wm.UserRotOrDiagonal = true

	return nil
}

func parseOpacity(s string, wm *Watermark) error {
	o, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("pdfcpu: opacity must be a float value: %s\n", s)
	}
	if o < 0 || o > 1 {
		return errors.Errorf("pdfcpu: illegal opacity: 0.0 <= r <= 1.0, %s\n", s)
	}
	wm.Opacity = o

	return nil
}

func parseRenderMode(s string, wm *Watermark) error {
	m, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("pdfcpu: illegal render mode value: allowed 0,1,2, %s\n", s)
	}
	rm := RenderMode(m)
	if rm != RMFill && rm != RMStroke && rm != RMFillAndStroke {
		return errors.New("pdfcpu: valid rendermodes: 0..fill, 1..stroke, 2..fill&stroke")
	}
	wm.RenderMode = rm

	return nil
}

func parseMargins(s string, wm *Watermark) error {
	var err error

	m := strings.Split(s, " ")
	if len(m) == 0 || len(m) > 4 {
		return errors.Errorf("pdfcpu: margins: need 1,2,3 or 4 int values, %s\n", s)
	}

	f, err := strconv.ParseFloat(m[0], 64)
	if err != nil {
		return err
	}
	i := int(toUserSpace(f, wm.InpUnit))

	if len(m) == 1 {
		wm.MLeft = i
		wm.MRight = i
		wm.MTop = i
		wm.MBot = i
		return nil
	}

	f, err = strconv.ParseFloat(m[1], 64)
	if err != nil {
		return err
	}
	j := int(toUserSpace(f, wm.InpUnit))

	if len(m) == 2 {
		wm.MTop, wm.MBot = i, i
		wm.MLeft, wm.MRight = j, j
		return nil
	}

	f, err = strconv.ParseFloat(m[2], 64)
	if err != nil {
		return err
	}
	k := int(toUserSpace(f, wm.InpUnit))

	if len(m) == 3 {
		wm.MTop = i
		wm.MLeft, wm.MRight = j, j
		wm.MBot = k
		return nil
	}

	f, err = strconv.ParseFloat(m[3], 64)
	if err != nil {
		return err
	}
	l := int(toUserSpace(f, wm.InpUnit))

	wm.MTop = i
	wm.MRight = j
	wm.MBot = k
	wm.MLeft = l
	return nil
}

func parseBorder(s string, wm *Watermark) error {
	// w
	// w r g b
	// w #c
	// w round
	// w round r g b
	// w round #c

	var err error

	b := strings.Split(s, " ")
	if len(b) == 0 || len(b) > 5 {
		return errors.Errorf("pdfcpu: borders: need 1,2,3,4 or 5 int values, %s\n", s)
	}

	wm.BorderWidth, err = strconv.Atoi(b[0])
	if err != nil {
		return err
	}
	if wm.BorderWidth == 0 {
		return errors.New("pdfcpu: borders: need width > 0")
	}

	if len(b) == 1 {
		return nil
	}

	if strings.HasPrefix("round", b[1]) {
		wm.BorderStyle = LJRound
		if len(b) == 2 {
			return nil
		}
		c, err := ParseColor(strings.Join(b[2:], " "))
		wm.BorderColor = &c
		return err
	}

	c, err := ParseColor(strings.Join(b[1:], " "))
	wm.BorderColor = &c
	return err
}

func parseWatermarkDetails(mode int, modeParm, s string, onTop bool, u DisplayUnit) (*Watermark, error) {
	wm := DefaultWatermarkConfig()
	wm.OnTop = onTop
	wm.InpUnit = u

	ss := strings.Split(s, ",")
	if len(ss) > 0 && len(ss[0]) == 0 {
		setWatermarkType(mode, modeParm, wm)
		return wm, nil
	}

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, parseWatermarkError(onTop)
		}

		paramPrefix := strings.TrimSpace(ss1[0])
		paramValueStr := strings.TrimSpace(ss1[1])

		if err := wmParamMap.Handle(paramPrefix, paramValueStr, wm); err != nil {
			return nil, err
		}
	}

	return wm, setWatermarkType(mode, modeParm, wm)
}

// ParseTextWatermarkDetails parses a text Watermark/Stamp command string into an internal structure.
func ParseTextWatermarkDetails(text, desc string, onTop bool, u DisplayUnit) (*Watermark, error) {
	return parseWatermarkDetails(WMText, text, desc, onTop, u)
}

// ParseImageWatermarkDetails parses an image Watermark/Stamp command string into an internal structure.
func ParseImageWatermarkDetails(fileName, desc string, onTop bool, u DisplayUnit) (*Watermark, error) {
	return parseWatermarkDetails(WMImage, fileName, desc, onTop, u)
}

// ParsePDFWatermarkDetails parses a PDF Watermark/Stamp command string into an internal structure.
func ParsePDFWatermarkDetails(fileName, desc string, onTop bool, u DisplayUnit) (*Watermark, error) {
	return parseWatermarkDetails(WMPDF, fileName, desc, onTop, u)
}

func (wm Watermark) calcMinFontSize(w float64) int {
	var minSize int
	for _, l := range wm.TextLines {
		w := font.Size(l, wm.FontName, w)
		if minSize == 0.0 {
			minSize = w
		}
		if w < minSize {
			minSize = w
		}
	}
	return minSize
}

// IsText returns true if the watermark content is text.
func (wm Watermark) isText() bool {
	return wm.Mode == WMText
}

// IsPDF returns true if the watermark content is PDF.
func (wm Watermark) isPDF() bool {
	return wm.Mode == WMPDF
}

// IsImage returns true if the watermark content is an image.
func (wm Watermark) isImage() bool {
	return wm.Mode == WMImage
}

func (wm *Watermark) calcBoundingBox(pageNr int) {
	bb := RectForDim(float64(wm.width), float64(wm.height))

	if wm.isPDF() {
		wm.bbPDF = wm.pdfRes[wm.Page].bb
		if wm.multiStamp() {
			i := pageNr
			if i > len(wm.pdfRes) {
				i = len(wm.pdfRes)
			}
			wm.bbPDF = wm.pdfRes[i].bb
		}
		wm.width = int(wm.bbPDF.Width())
		wm.height = int(wm.bbPDF.Height())
		bb = wm.bbPDF.CroppedCopy(0)
	}

	ar := bb.AspectRatio()

	if wm.ScaleAbs {
		w1 := wm.Scale * bb.Width()
		bb.UR.X = bb.LL.X + w1
		bb.UR.Y = bb.LL.Y + w1/ar
		wm.bb = bb
		wm.ScaleEff = wm.Scale
		return
	}

	if ar >= 1 {
		// Landscape
		w1 := wm.Scale * wm.vp.Width()
		bb.UR.X = bb.LL.X + w1
		bb.UR.Y = bb.LL.Y + w1/ar
		wm.ScaleEff = w1 / float64(wm.width)
	} else {
		// Portrait
		h1 := wm.Scale * wm.vp.Height()
		bb.UR.Y = bb.LL.Y + h1
		bb.UR.X = bb.LL.X + h1*ar
		wm.ScaleEff = h1 / float64(wm.height)
	}

	wm.bb = bb
}

func (wm *Watermark) calcTransformMatrix() Matrix {
	var sin, cos float64
	r := wm.Rotation

	if wm.Diagonal != NoDiagonal {

		// Calculate the angle of the diagonal with respect of the aspect ratio of the bounding box.
		r = math.Atan(wm.vp.Height()/wm.vp.Width()) * float64(RadToDeg)

		if wm.bb.AspectRatio() < 1 {
			r -= 90
		}

		if wm.Diagonal == DiagonalULToLR {
			r = -r
		}

	}

	sin = math.Sin(float64(r) * float64(DegToRad))
	cos = math.Cos(float64(r) * float64(DegToRad))

	var dy float64
	if !wm.isImage() && !wm.isPDF() {
		dy = wm.bb.LL.Y
	}
	ll := lowerLeftCorner(wm.vp.Width(), wm.vp.Height(), wm.bb.Width(), wm.bb.Height(), wm.Pos)

	dx := ll.X + wm.bb.Width()/2 + float64(wm.Dx) + sin*(wm.bb.Height()/2+dy) - cos*wm.bb.Width()/2
	dy = ll.Y + wm.bb.Height()/2 + float64(wm.Dy) - cos*(wm.bb.Height()/2+dy) - sin*wm.bb.Width()/2

	return CalcTransformMatrix(1, 1, sin, cos, dx, dy)
}

func onTopString(onTop bool) string {
	e := "watermark"
	if onTop {
		e = "stamp"
	}
	return e
}

func parseWatermarkError(onTop bool) error {
	s := onTopString(onTop)
	return errors.Errorf("Invalid %s configuration string. Please consult pdfcpu help %s.\n", s, s)
}

func setTextWatermark(s string, wm *Watermark) {
	wm.TextString = s
	if font.IsCoreFont(wm.FontName) {
		bb := []byte{}
		for _, r := range s {
			// Unicode => char code
			b := byte(0x20) // better use glyph: .notdef
			if r <= 0xff {
				b = byte(r)
			}
			bb = append(bb, b)
		}
		s = string(bb)
	} else {
		bb := []byte{}
		u := utf16.Encode([]rune(s))
		for _, i := range u {
			bb = append(bb, byte((i>>8)&0xFF))
			bb = append(bb, byte(i&0xFF))
		}
		s = string(bb)
	}
	s = strings.ReplaceAll(s, "\\n", "\n")
	wm.TextLines = append(wm.TextLines, strings.FieldsFunc(s, func(c rune) bool { return c == 0x0a })...)
}

func setImageWatermark(s string, wm *Watermark) error {
	if len(s) == 0 {
		// The caller is expected to supply wm.Image
		return nil
	}
	if !ImageFileName(s) {
		return errors.New("imageFileName has to have one of these extensions: .jpg, .jpeg, .png, .tif, .tiff, .webp")
	}
	wm.FileName = s
	f, err := os.Open(wm.FileName)
	if err != nil {
		return err
	}
	defer f.Close()
	bb, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	wm.Image = bytes.NewReader(bb)
	return nil
}

func setPDFWatermark(s string, wm *Watermark) error {
	i := strings.LastIndex(s, ":")
	if i < 1 {
		// No Colon.
		if strings.ToLower(filepath.Ext(s)) != ".pdf" {
			return errors.Errorf("%s is not a PDF file", s)
		}
		wm.FileName = s
		return nil
	}
	// We have at least one Colon.
	if strings.ToLower(filepath.Ext(s)) == ".pdf" {
		// We have an absolute DOS filename.
		wm.FileName = s
		return nil
	}
	// We expect a page number on the right side of the right most Colon.
	var err error
	pageNumberStr := s[i+1:]
	wm.Page, err = strconv.Atoi(pageNumberStr)
	if err != nil {
		return errors.Errorf("illegal PDF page number: %s\n", pageNumberStr)
	}
	fileName := s[:i]
	if strings.ToLower(filepath.Ext(fileName)) != ".pdf" {
		return errors.Errorf("%s is not a PDF file", fileName)
	}
	wm.FileName = fileName
	return nil
}

func setWatermarkType(mode int, s string, wm *Watermark) (err error) {
	wm.Mode = mode
	switch wm.Mode {
	case WMText:
		setTextWatermark(s, wm)

	case WMImage:
		err = setImageWatermark(s, wm)

	case WMPDF:
		err = setPDFWatermark(s, wm)
	}
	return err
}

func migrateIndRef(ir *IndirectRef, ctxSource, ctxDest *Context, migrated map[int]int) (Object, error) {
	o, err := ctxSource.Dereference(*ir)
	if err != nil {
		return nil, err
	}

	if o != nil {
		o = o.Clone()
	}

	objNrNew, err := ctxDest.InsertObject(o)
	if err != nil {
		return nil, err
	}

	objNr := ir.ObjectNumber.Value()
	migrated[objNr] = objNrNew
	ir.ObjectNumber = Integer(objNrNew)
	return o, nil
}

func migrateObject(o Object, ctxSource, ctxDest *Context, migrated map[int]int) (Object, error) {
	var err error
	switch o := o.(type) {
	case IndirectRef:
		objNr := o.ObjectNumber.Value()
		if migrated[objNr] > 0 {
			o.ObjectNumber = Integer(migrated[objNr])
			return o, nil
		}
		o1, err := migrateIndRef(&o, ctxSource, ctxDest, migrated)
		if err != nil {
			return nil, err
		}
		if _, err := migrateObject(o1, ctxSource, ctxDest, migrated); err != nil {
			return nil, err
		}
		return o, nil

	case Dict:
		for k, v := range o {
			if o[k], err = migrateObject(v, ctxSource, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
		return o, nil

	case StreamDict:
		for k, v := range o.Dict {
			if o.Dict[k], err = migrateObject(v, ctxSource, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
		return o, nil

	case Array:
		for k, v := range o {
			if o[k], err = migrateObject(v, ctxSource, ctxDest, migrated); err != nil {
				return nil, err
			}
		}
		return o, nil
	}

	return o, nil
}

func createPDFRes(ctx, otherCtx *Context, pageNr int, migrated map[int]int, wm *Watermark) error {
	pdfRes := pdfResources{}
	xRefTable := ctx.XRefTable
	otherXRefTable := otherCtx.XRefTable

	// Locate page dict & resource dict of PDF stamp.
	consolidateRes := true
	d, _, inhPAttrs, err := otherXRefTable.PageDict(pageNr, consolidateRes)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.Errorf("pdfcpu: unknown page number: %d\n", pageNr)
	}

	// Retrieve content stream bytes of page dict.
	pdfRes.content, err = otherXRefTable.PageContent(d)
	if err != nil {
		return err
	}

	// Migrate external resource dict into ctx.
	if _, err = migrateObject(inhPAttrs.Resources, otherCtx, ctx, migrated); err != nil {
		return err
	}

	// Create an object for resource dict in xRefTable.
	if inhPAttrs.Resources != nil {
		ir, err := xRefTable.IndRefForNewObject(inhPAttrs.Resources)
		if err != nil {
			return err
		}
		pdfRes.resDict = ir
	}

	pdfRes.bb = viewPort(inhPAttrs)
	wm.pdfRes[pageNr] = pdfRes

	return nil
}

func (ctx *Context) createPDFResForWM(wm *Watermark) error {
	// Note: The stamp pdf is assumed to be valid!
	otherCtx, err := ReadFile(wm.FileName, NewDefaultConfiguration())
	if err != nil {
		return err
	}

	if err := otherCtx.EnsurePageCount(); err != nil {
		return nil
	}

	migrated := map[int]int{}

	if !wm.multiStamp() {
		if err := createPDFRes(ctx, otherCtx, wm.Page, migrated, wm); err != nil {
			return err
		}
	} else {
		j := otherCtx.PageCount
		if ctx.PageCount < otherCtx.PageCount {
			j = ctx.PageCount
		}
		for i := 1; i <= j; i++ {
			if err := createPDFRes(ctx, otherCtx, i, migrated, wm); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ctx *Context) createImageResForWM(wm *Watermark) (err error) {

	wm.image, wm.width, wm.height, err = CreateImageResource(ctx.XRefTable, wm.Image, false, false)
	return err
}

func (ctx *Context) createFontResForWM(wm *Watermark) (err error) {
	// TODO Take existing font dicts into account.
	if font.IsUserFont(wm.FontName) {
		// Dummy call in order to setup used glyphs.
		td, _ := setupTextDescriptor(wm, "", 0, 0)
		WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)
	}
	wm.font, err = EnsureFontDict(ctx.XRefTable, wm.FontName, true, nil)
	return err
}

func (ctx *Context) createResourcesForWM(wm *Watermark) error {
	if wm.isPDF() {
		return ctx.createPDFResForWM(wm)
	}
	if wm.isImage() {
		return ctx.createImageResForWM(wm)
	}
	return ctx.createFontResForWM(wm)
}

func (ctx *Context) ensureOCG(onTop bool) (*IndirectRef, error) {
	name := "Background"
	subt := "BG"
	if onTop {
		name = "Watermark"
		subt = "FG"
	}

	d := Dict(
		map[string]Object{
			"Name": StringLiteral(name),
			"Type": Name("OCG"),
			"Usage": Dict(
				map[string]Object{
					"PageElement": Dict(map[string]Object{"Subtype": Name(subt)}),
					"View":        Dict(map[string]Object{"ViewState": Name("ON")}),
					"Print":       Dict(map[string]Object{"PrintState": Name("ON")}),
					"Export":      Dict(map[string]Object{"ExportState": Name("ON")}),
				},
			),
		},
	)

	return ctx.IndRefForNewObject(d)
}

func (ctx *Context) prepareOCPropertiesInRoot(onTop bool) (*IndirectRef, error) {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, err
	}

	if o, ok := rootDict.Find("OCProperties"); ok {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return nil, err
		}

		o, found := d.Find("OCGs")
		if found {
			a, err := ctx.DereferenceArray(o)
			if err != nil {
				return nil, errCorruptOCGs
			}
			if len(a) > 0 {
				ir, ok := a[0].(IndirectRef)
				if !ok {
					return nil, errCorruptOCGs
				}
				return &ir, nil
			}
		}
	}

	ir, err := ctx.ensureOCG(onTop)
	if err != nil {
		return nil, err
	}

	optionalContentConfigDict := Dict(
		map[string]Object{
			"AS": Array{
				Dict(
					map[string]Object{
						"Category": NewNameArray("View"),
						"Event":    Name("View"),
						"OCGs":     Array{*ir},
					},
				),
				Dict(
					map[string]Object{
						"Category": NewNameArray("Print"),
						"Event":    Name("Print"),
						"OCGs":     Array{*ir},
					},
				),
				Dict(
					map[string]Object{
						"Category": NewNameArray("Export"),
						"Event":    Name("Export"),
						"OCGs":     Array{*ir},
					},
				),
			},
			"ON":       Array{*ir},
			"Order":    Array{},
			"RBGroups": Array{},
		},
	)

	d := Dict(
		map[string]Object{
			"OCGs": Array{*ir},
			"D":    optionalContentConfigDict,
		},
	)

	rootDict.Update("OCProperties", d)
	return ir, nil
}

func (ctx *Context) createFormResDict(pageNr int, wm *Watermark) (*IndirectRef, error) {
	if wm.isPDF() {
		i := wm.Page
		if wm.multiStamp() {
			maxStampPageNr := len(wm.pdfRes)
			i = pageNr
			if pageNr > maxStampPageNr {
				i = maxStampPageNr
			}
		}
		return wm.pdfRes[i].resDict, nil
	}

	if wm.isImage() {
		d := Dict(
			map[string]Object{
				"ProcSet": NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
				"XObject": Dict(map[string]Object{"Im0": *wm.image}),
			},
		)
		return ctx.IndRefForNewObject(d)
	}

	d := Dict(
		map[string]Object{
			"Font":    Dict(map[string]Object{"F1": *wm.font}),
			"ProcSet": NewNameArray("PDF", "Text", "ImageB", "ImageC", "ImageI"),
		},
	)

	return ctx.IndRefForNewObject(d)
}

func cachedForm(wm *Watermark) bool {
	return !wm.isPDF() || !wm.multiStamp()
}

func pdfFormContent(w io.Writer, pageNr int, wm *Watermark) error {
	cs := wm.pdfRes[wm.Page].content
	if wm.multiStamp() {
		maxStampPageNr := len(wm.pdfRes)
		i := pageNr
		if pageNr > maxStampPageNr {
			i = maxStampPageNr
		}
		cs = wm.pdfRes[i].content
	}
	sc := wm.Scale
	if !wm.ScaleAbs {
		sc = wm.bb.Width() / float64(wm.width)
	}

	// Scale & translate into origin

	m1 := identMatrix
	m1[0][0] = sc
	m1[1][1] = sc

	m2 := identMatrix
	m2[2][0] = -wm.bb.LL.X * wm.ScaleEff
	m2[2][1] = -wm.bb.LL.Y * wm.ScaleEff

	m := m1.multiply(m2)

	fmt.Fprintf(w, "%.2f %.2f %.2f %.2f %.2f %.2f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])

	_, err := w.Write(cs)
	return err
}

func imageFormContent(w io.Writer, wm *Watermark) {
	fmt.Fprintf(w, "q %f 0 0 %f 0 0 cm /Im0 Do Q", wm.bb.Width(), wm.bb.Height()) // TODO dont need Q
}

func formContent(w io.Writer, pageNr int, wm *Watermark) error {
	switch true {
	case wm.isPDF():
		return pdfFormContent(w, pageNr, wm)
	case wm.isImage():
		imageFormContent(w, wm)
	}
	return nil
}

func setupTextDescriptor(wm *Watermark, timestampFormat string, pageNr, pageCount int) (TextDescriptor, bool) {
	// Set horizontal alignment.
	var hAlign HAlignment
	if wm.HAlign == nil {
		// Use alignment implied by anchor.
		_, _, hAlign, _ = anchorPosAndAlign(wm.Pos, RectForDim(0, 0))
	} else {
		// Use manual alignment.
		hAlign = *wm.HAlign
	}

	// Set effective position and vertical alignment.
	x, y, _, vAlign := anchorPosAndAlign(BottomLeft, wm.vp)
	td, unique := wm.textDescriptor(timestampFormat, pageNr, pageCount)
	td.X, td.Y, td.HAlign, td.VAlign, td.FontKey = x, y, hAlign, vAlign, "F1"

	// Set right to left rendering.
	td.RTL = wm.RTL

	// Set margins.
	td.MLeft = float64(wm.MLeft)
	td.MRight = float64(wm.MRight)
	td.MTop = float64(wm.MTop)
	td.MBot = float64(wm.MBot)

	// Set border.
	td.BorderWidth = float64(wm.BorderWidth)
	td.BorderStyle = wm.BorderStyle
	if wm.BorderColor != nil {
		td.ShowBorder = true
		td.BorderCol = *wm.BorderColor
	}
	return td, unique
}

func drawBoundingBox(b bytes.Buffer, wm *Watermark, bb *Rectangle) {
	urx := bb.UR.X
	ury := bb.UR.Y
	if wm.isPDF() {
		sc := wm.Scale
		if !wm.ScaleAbs {
			sc = bb.Width() / float64(wm.width)
		}
		urx /= sc
		ury /= sc
	}
	fmt.Fprintf(&b, "[]0 d 2 w %.2f %.2f m %.2f %.2f l %.2f %.2f l %.2f %.2f l s ",
		bb.LL.X, bb.LL.Y,
		urx, bb.LL.Y,
		urx, ury,
		bb.LL.X, ury,
	)
}

func calcFormBoundingBox(w io.Writer, timestampFormat string, pageNr, pageCount int, wm *Watermark) bool {
	var unique bool
	if wm.isImage() || wm.isPDF() {
		wm.calcBoundingBox(pageNr)
	} else {
		var td TextDescriptor
		td, unique = setupTextDescriptor(wm, timestampFormat, pageNr, pageCount)
		// Render td into b and return the bounding box.
		wm.bb = WriteMultiLine(w, wm.vp, nil, td)
	}
	return unique
}

func (ctx *Context) createForm(pageNr, pageCount int, wm *Watermark, withBB bool) error {
	var b bytes.Buffer
	unique := calcFormBoundingBox(&b, ctx.Configuration.TimestampFormat, pageNr, pageCount, wm)

	// The forms bounding box is dependent on the page dimensions.
	bb := wm.bb

	if !unique && (cachedForm(wm) || pageNr > len(wm.pdfRes)) {
		// Use cached form.
		ir, ok := wm.fCache[*bb.Rectangle]
		if ok {
			wm.form = ir
			return nil
		}
	}

	if wm.isImage() || wm.isPDF() {
		if err := formContent(&b, pageNr, wm); err != nil {
			return err
		}
	}

	ir, err := ctx.createFormResDict(pageNr, wm)
	if err != nil {
		return err
	}

	bbox := bb.CroppedCopy(0)
	bbox.Translate(-bb.LL.X, -bb.LL.Y)

	// Paint bounding box
	if withBB {
		drawBoundingBox(b, wm, bbox)
	}

	sd := StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":    Name("XObject"),
				"Subtype": Name("Form"),
				"BBox":    bbox.Array(),
				"Matrix":  NewNumberArray(1, 0, 0, 1, 0, 0),
				"OC":      *wm.ocg,
			},
		),
		Content:        b.Bytes(),
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}

	if ir != nil {
		sd.Insert("Resources", *ir)
	}

	sd.InsertName("Filter", filter.Flate)

	if err = sd.Encode(); err != nil {
		return err
	}

	ir, err = ctx.IndRefForNewObject(sd)
	if err != nil {
		return err
	}

	wm.form = ir

	if cachedForm(wm) || pageNr >= len(wm.pdfRes) {
		// Cache form.
		wm.fCache[*wm.bb.Rectangle] = ir
	}

	return nil
}

func (ctx *Context) createExtGStateForStamp(opacity float64) (*IndirectRef, error) {
	d := Dict(
		map[string]Object{
			"Type": Name("ExtGState"),
			"CA":   Float(opacity),
			"ca":   Float(opacity),
		},
	)

	return ctx.IndRefForNewObject(d)
}

func (ctx *Context) insertPageResourcesForWM(pageDict Dict, wm *Watermark, gsID, xoID string) error {
	resourceDict := Dict(
		map[string]Object{
			"ExtGState": Dict(map[string]Object{gsID: *wm.extGState}),
			"XObject":   Dict(map[string]Object{xoID: *wm.form}),
		},
	)

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func (ctx *Context) updatePageResourcesForWM(resDict Dict, wm *Watermark, gsID, xoID *string) error {
	o, ok := resDict.Find("ExtGState")
	if !ok {
		resDict.Insert("ExtGState", Dict(map[string]Object{*gsID: *wm.extGState}))
	} else {
		d, _ := ctx.DereferenceDict(o)
		for i := 0; i < 1000; i++ {
			*gsID = "GS" + strconv.Itoa(i)
			if _, found := d.Find(*gsID); !found {
				break
			}
		}
		d.Insert(*gsID, *wm.extGState)
	}

	o, ok = resDict.Find("XObject")
	if !ok {
		resDict.Insert("XObject", Dict(map[string]Object{*xoID: *wm.form}))
	} else {
		d, _ := ctx.DereferenceDict(o)
		for i := 0; i < 1000; i++ {
			*xoID = "Fm" + strconv.Itoa(i)
			if _, found := d.Find(*xoID); !found {
				break
			}
		}
		d.Insert(*xoID, *wm.form)
	}

	return nil
}

func wmContent(wm *Watermark, gsID, xoID string) []byte {
	m := wm.calcTransformMatrix()
	p1 := m.Transform(Point{wm.bb.LL.X, wm.bb.LL.Y})
	p2 := m.Transform(Point{wm.bb.UR.X, wm.bb.LL.X})
	p3 := m.Transform(Point{wm.bb.UR.X, wm.bb.UR.Y})
	p4 := m.Transform(Point{wm.bb.LL.X, wm.bb.UR.Y})
	wm.bbTrans = QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}
	insertOCG := " /Artifact <</Subtype /Watermark /Type /Pagination >>BDC q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s gs /%s Do Q EMC "
	var b bytes.Buffer
	fmt.Fprintf(&b, insertOCG, m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], gsID, xoID)
	return b.Bytes()
}

func (ctx *Context) insertPageContentsForWM(pageDict Dict, wm *Watermark, gsID, xoID string) error {
	sd, _ := ctx.NewStreamDictForBuf(wmContent(wm, gsID, xoID))
	if err := sd.Encode(); err != nil {
		return err
	}

	ir, err := ctx.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *ir)

	return nil
}

func translationForPageRotation(pageRot int, w, h float64) (float64, float64) {
	var dx, dy float64

	switch pageRot {
	case 90, -270:
		dy = h
	case -90, 270:
		dx = w
	case 180, -180:
		dx, dy = w, h
	}

	return dx, dy
}

func contentBytesForPageRotation(rot int, w, h float64) []byte {
	dx, dy := translationForPageRotation(rot, w, h)
	// Note: PDF rotation is clockwise!
	m := calcRotateAndTranslateTransformMatrix(float64(-rot), dx, dy)
	var b bytes.Buffer
	fmt.Fprintf(&b, "%.2f %.2f %.2f %.2f %.2f %.2f cm ", m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])
	return b.Bytes()
}

func patchFirstContentStreamForWatermark(sd *StreamDict, gsID, xoID string, wm *Watermark, isLast bool) error {
	err := sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		log.Info.Println("unsupported filter: unable to patch content with watermark.")
		return nil
	}
	if err != nil {
		return err
	}

	wmbb := wmContent(wm, gsID, xoID)

	// stamp
	if wm.OnTop {
		bb := []byte(" q ")
		if wm.pageRot != 0 {
			bb = append(bb, contentBytesForPageRotation(wm.pageRot, wm.vp.Width(), wm.vp.Height())...)
		}
		sd.Content = append(bb, sd.Content...)
		if !isLast {
			return sd.Encode()
		}
		sd.Content = append(sd.Content, []byte(" Q ")...)
		sd.Content = append(sd.Content, wmbb...)
		return sd.Encode()
	}

	// watermark
	if wm.pageRot == 0 {
		sd.Content = append(wmbb, sd.Content...)
		return sd.Encode()
	}

	bb := append([]byte(" q "), contentBytesForPageRotation(wm.pageRot, wm.vp.Width(), wm.vp.Height())...)
	sd.Content = append(bb, sd.Content...)
	if isLast {
		sd.Content = append(sd.Content, []byte(" Q")...)
	}
	return sd.Encode()
}

func patchLastContentStreamForWatermark(sd *StreamDict, gsID, xoID string, wm *Watermark) error {
	err := sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		log.Info.Println("unsupported filter: unable to patch content with watermark.")
		return nil
	}
	if err != nil {
		return err
	}

	// stamp
	if wm.OnTop {
		sd.Content = append(sd.Content, []byte(" Q ")...)
		sd.Content = append(sd.Content, wmContent(wm, gsID, xoID)...)
		return sd.Encode()
	}

	// watermark
	if wm.pageRot != 0 {
		sd.Content = append(sd.Content, []byte(" Q")...)
		return sd.Encode()
	}

	return nil
}

func (ctx *Context) updatePageContentsForWM(obj Object, wm *Watermark, gsID, xoID string) error {
	var entry *XRefTableEntry
	var objNr int

	ir, ok := obj.(IndirectRef)
	if ok {
		objNr = ir.ObjectNumber.Value()
		if wm.objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}
		genNr := ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		obj = entry.Object
	}

	switch o := obj.(type) {

	case StreamDict:

		err := patchFirstContentStreamForWatermark(&o, gsID, xoID, wm, true)
		if err != nil {
			return err
		}

		entry.Object = o
		wm.objs[objNr] = true

	case Array:

		// Get stream dict for first array element.
		o1 := o[0]
		ir, _ := o1.(IndirectRef)
		objNr = ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(StreamDict)

		if wm.objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}

		err := patchFirstContentStreamForWatermark(&sd, gsID, xoID, wm, len(o) == 1)
		if err != nil {
			return err
		}

		entry.Object = sd
		wm.objs[objNr] = true
		if len(o) == 1 {
			return nil
		}

		// Get stream dict for last array element.
		o1 = o[len(o)-1]

		ir, _ = o1.(IndirectRef)
		objNr = ir.ObjectNumber.Value()
		if wm.objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}

		genNr = ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		sd, _ = (entry.Object).(StreamDict)

		err = patchLastContentStreamForWatermark(&sd, gsID, xoID, wm)
		if err != nil {
			return err
		}

		entry.Object = sd
		wm.objs[objNr] = true
	}

	return nil
}

func viewPort(a *InheritedPageAttrs) *Rectangle {
	visibleRegion := a.MediaBox
	if a.CropBox != nil {
		visibleRegion = a.CropBox
	}
	return visibleRegion
}

func (ctx *Context) addPageWatermark(i int, wm *Watermark) error {
	if i > ctx.PageCount {
		return errors.Errorf("pdfcpu: invalid page number: %d", i)
	}

	log.Debug.Printf("addPageWatermark page:%d\n", i)
	if wm.Update {
		log.Debug.Println("Updating")
		if _, err := ctx.removePageWatermark(i); err != nil {
			return err
		}
	}

	consolidateRes := false
	d, pageIndRef, inhPAttrs, err := ctx.PageDict(i, consolidateRes)
	if err != nil {
		return err
	}

	// Internalize page rotation into content stream.
	wm.pageRot = inhPAttrs.Rotate

	wm.vp = viewPort(inhPAttrs)

	// Reset page rotation in page dict.
	if wm.pageRot != 0 {

		if IntMemberOf(wm.pageRot, []int{+90, -90, +270, -270}) {
			w := wm.vp.Width()
			wm.vp.UR.X = wm.vp.LL.X + wm.vp.Height()
			wm.vp.UR.Y = wm.vp.LL.Y + w
		}

		d.Update("MediaBox", wm.vp.Array())
		d.Update("CropBox", wm.vp.Array())
		d.Delete("Rotate")
	}

	if err = ctx.createForm(i, ctx.PageCount, wm, stampWithBBox); err != nil {
		return err
	}

	log.Debug.Printf("\n%s\n", wm)

	gsID := "GS0"
	xoID := "Fm0"

	if inhPAttrs.Resources != nil {
		err = ctx.updatePageResourcesForWM(inhPAttrs.Resources, wm, &gsID, &xoID)
		d.Update("Resources", inhPAttrs.Resources)
	} else {
		err = ctx.insertPageResourcesForWM(d, wm, gsID, xoID)
	}
	if err != nil {
		return err
	}

	obj, found := d.Find("Contents")
	if found {
		err = ctx.updatePageContentsForWM(obj, wm, gsID, xoID)
	} else {
		err = ctx.insertPageContentsForWM(d, wm, gsID, xoID)
	}
	if err != nil {
		return err
	}

	if wm.OnTop && wm.URL != "" {

		ann := NewLinkAnnotation(
			*wm.bbTrans.EnclosingRectangle(5.0),
			QuadPoints{wm.bbTrans},
			wm.URL,
			"pdfcpu",
			AnnNoZoom+AnnNoRotate,
			nil)

		if _, err := ctx.AddAnnotation(pageIndRef, d, i, ann, false); err != nil {
			return err
		}
	}

	return nil
}

func (ctx *Context) createWMResources(
	wm *Watermark,
	pageNr int,
	fm map[string]IntSet,
	ocgIndRef, extGStateIndRef *IndirectRef,
	onTop bool, opacity float64) error {

	wm.ocg = ocgIndRef
	wm.extGState = extGStateIndRef
	wm.OnTop = onTop
	wm.Opacity = opacity

	if wm.isImage() {
		return ctx.createImageResForWM(wm)
	}

	if wm.isPDF() {
		return ctx.createPDFResForWM(wm)
	}

	// Text watermark

	if font.IsUserFont(wm.FontName) {
		// Dummy call in order to setup used glyphs.
		td, _ := setupTextDescriptor(wm, "", 0, 0)
		WriteMultiLine(new(bytes.Buffer), RectForFormat("A4"), nil, td)
	}

	pageSet, found := fm[wm.FontName]
	if !found {
		fm[wm.FontName] = IntSet{pageNr: true}
	} else {
		pageSet[pageNr] = true
	}

	return nil
}

func (ctx *Context) createResourcesForWMMap(
	m map[int]*Watermark,
	ocgIndRef, extGStateIndRef *IndirectRef,
	onTop bool,
	opacity float64) (map[string]IntSet, error) {

	fm := map[string]IntSet{}
	for pageNr, wm := range m {
		if err := ctx.createWMResources(wm, pageNr, fm, ocgIndRef, extGStateIndRef, onTop, opacity); err != nil {
			return nil, err
		}
	}

	return fm, nil
}

func (ctx *Context) createResourcesForWMSliceMap(
	m map[int][]*Watermark,
	ocgIndRef, extGStateIndRef *IndirectRef,
	onTop bool,
	opacity float64) (map[string]IntSet, error) {

	fm := map[string]IntSet{}
	for pageNr, wms := range m {
		for _, wm := range wms {
			if err := ctx.createWMResources(wm, pageNr, fm, ocgIndRef, extGStateIndRef, onTop, opacity); err != nil {
				return nil, err
			}
		}
	}

	return fm, nil
}

// AddWatermarksMap adds watermarks in m to corresponding pages.
func (ctx *Context) AddWatermarksMap(m map[int]*Watermark) error {
	var (
		onTop   bool
		opacity float64
	)
	for _, wm := range m {
		onTop = wm.OnTop
		opacity = wm.Opacity
		break
	}

	ocgIndRef, err := ctx.prepareOCPropertiesInRoot(onTop)
	if err != nil {
		return err
	}

	extGStateIndRef, err := ctx.createExtGStateForStamp(opacity)
	if err != nil {
		return err
	}

	fm, err := ctx.createResourcesForWMMap(m, ocgIndRef, extGStateIndRef, onTop, opacity)
	if err != nil {
		return err
	}

	// TODO Take existing font dicts in xref into account.
	for fontName, pageSet := range fm {
		ir, err := EnsureFontDict(ctx.XRefTable, fontName, true, nil)
		if err != nil {
			return err
		}
		for pageNr, v := range pageSet {
			if !v {
				continue
			}
			wm := m[pageNr]
			if wm.isText() && wm.FontName == fontName {
				m[pageNr].font = ir
			}
		}
	}

	for k, wm := range m {
		if err := ctx.addPageWatermark(k, wm); err != nil {
			return err
		}
	}

	ctx.EnsureVersionForWriting()
	return nil
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages.
func (ctx *Context) AddWatermarksSliceMap(m map[int][]*Watermark) error {
	var (
		onTop   bool
		opacity float64
	)
	for _, wms := range m {
		onTop = wms[0].OnTop
		opacity = wms[0].Opacity
		break
	}

	ocgIndRef, err := ctx.prepareOCPropertiesInRoot(onTop)
	if err != nil {
		return err
	}

	extGStateIndRef, err := ctx.createExtGStateForStamp(opacity)
	if err != nil {
		return err
	}

	fm, err := ctx.createResourcesForWMSliceMap(m, ocgIndRef, extGStateIndRef, onTop, opacity)
	if err != nil {
		return err
	}

	// TODO Take existing font dicts in xref into account.
	for fontName, pageSet := range fm {
		ir, err := EnsureFontDict(ctx.XRefTable, fontName, true, nil)
		if err != nil {
			return err
		}
		for pageNr, v := range pageSet {
			if !v {
				continue
			}
			for _, wm := range m[pageNr] {
				if wm.isText() && wm.FontName == fontName {
					wm.font = ir
				}
			}
		}
	}

	for k, wms := range m {
		for _, wm := range wms {
			if err := ctx.addPageWatermark(k, wm); err != nil {
				return err
			}
		}
	}

	ctx.EnsureVersionForWriting()
	return nil
}

// AddWatermarks adds watermarks to all pages selected.
func (ctx *Context) AddWatermarks(selectedPages IntSet, wm *Watermark) error {
	log.Debug.Printf("AddWatermarks wm:\n%s\n", wm)
	var err error
	if wm.ocg, err = ctx.prepareOCPropertiesInRoot(wm.OnTop); err != nil {
		return err
	}

	if err = ctx.createResourcesForWM(wm); err != nil {
		return err
	}

	if wm.extGState, err = ctx.createExtGStateForStamp(wm.Opacity); err != nil {
		return err
	}

	if selectedPages == nil || len(selectedPages) == 0 {
		selectedPages = IntSet{}
		for i := 1; i <= ctx.PageCount; i++ {
			selectedPages[i] = true
		}
	}

	for k, v := range selectedPages {
		if v {
			if err = ctx.addPageWatermark(k, wm); err != nil {
				return err
			}
		}
	}

	ctx.EnsureVersionForWriting()
	return nil
}

func (ctx *Context) removeResDictEntry(d Dict, entry string, ids []string, i int) error {
	o, ok := d.Find(entry)
	if !ok {
		return errors.Errorf("pdfcpu: page %d: corrupt resource dict", i)
	}

	d1, err := ctx.DereferenceDict(o)
	if err != nil {
		return err
	}

	for _, id := range ids {
		o, ok := d1.Find(id)
		if ok {
			err = ctx.deleteObject(o)
			if err != nil {
				return err
			}
			d1.Delete(id)
		}
	}

	if d1.Len() == 0 {
		d.Delete(entry)
	}

	return nil
}

func (ctx *Context) removeExtGStates(d Dict, ids []string, i int) error {
	return ctx.removeResDictEntry(d, "ExtGState", ids, i)
}

func (ctx *Context) removeForms(d Dict, ids []string, i int) error {
	return ctx.removeResDictEntry(d, "XObject", ids, i)
}

func removeArtifacts(sd *StreamDict, i int) (ok bool, extGStates []string, forms []string, err error) {
	err = sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		log.Info.Printf("unsupported filter: unable to patch content with watermark for page %d\n", i)
		return false, nil, nil, nil
	}
	if err != nil {
		return false, nil, nil, err
	}

	var patched bool

	// Watermarks may begin or end the content stream.

	for {
		s := string(sd.Content)
		beg := strings.Index(s, "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC")
		if beg < 0 {
			break
		}

		end := strings.Index(s[beg:], "EMC")
		if end < 0 {
			break
		}

		// Check for usage of resources.
		t := s[beg : beg+end]

		i := strings.Index(t, "/GS")
		if i > 0 {
			j := i + 3
			k := strings.Index(t[j:], " gs")
			if k > 0 {
				extGStates = append(extGStates, "GS"+t[j:j+k])
			}
		}

		i = strings.Index(t, "/Fm")
		if i > 0 {
			j := i + 3
			k := strings.Index(t[j:], " Do")
			if k > 0 {
				forms = append(forms, "Fm"+t[j:j+k])
			}
		}

		// TODO Remove whitespace until 0x0a
		sd.Content = append(sd.Content[:beg], sd.Content[beg+end+3:]...)
		patched = true
	}

	if patched {
		err = sd.Encode()
	}

	return patched, extGStates, forms, err
}

func (ctx *Context) removeArtifactsFromPage(sd *StreamDict, resDict Dict, i int) (bool, error) {
	// Remove watermark artifacts and locate id's
	// of used extGStates and forms.
	ok, extGStates, forms, err := removeArtifacts(sd, i)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// Remove obsolete extGStates from page resource dict.
	err = ctx.removeExtGStates(resDict, extGStates, i)
	if err != nil {
		return false, err
	}

	// Remove obsolete forms from page resource dict.
	return true, ctx.removeForms(resDict, forms, i)
}

func (ctx *Context) locatePageContentAndResourceDict(pageNr int) (Object, *IndirectRef, Dict, error) {
	consolidateRes := false
	d, pageDictIndRef, _, err := ctx.PageDict(pageNr, consolidateRes)
	if err != nil {
		return nil, nil, nil, err
	}

	o, found := d.Find("Resources")
	if !found {
		return nil, nil, nil, errors.Errorf("pdfcpu: page %d: no resource dict found\n", pageNr)
	}

	resDict, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, nil, nil, err
	}

	o, found = d.Find("Contents")
	if !found {
		return nil, nil, nil, errors.Errorf("pdfcpu: page %d: no page watermark found", pageNr)
	}

	return o, pageDictIndRef, resDict, nil
}

func (ctx *Context) removeArtifacts(o Object, entry *XRefTableEntry, resDict Dict, pageNr int) (bool, error) {
	found := false
	switch o := o.(type) {

	case StreamDict:
		ok, err := ctx.removeArtifactsFromPage(&o, resDict, pageNr)
		if err != nil {
			return false, err
		}
		if !found && ok {
			found = true
		}
		entry.Object = o

	case Array:
		// Get stream dict for first element.
		o1 := o[0]
		ir, _ := o1.(IndirectRef)
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(StreamDict)

		ok, err := ctx.removeArtifactsFromPage(&sd, resDict, pageNr)
		if err != nil {
			return false, err
		}
		if !found && ok {
			found = true
			entry.Object = sd
		}

		if len(o) > 1 {
			// Get stream dict for last element.
			o1 := o[len(o)-1]
			ir, _ := o1.(IndirectRef)
			objNr = ir.ObjectNumber.Value()
			genNr := ir.GenerationNumber.Value()
			entry, _ := ctx.FindTableEntry(objNr, genNr)
			sd, _ := (entry.Object).(StreamDict)

			ok, err = ctx.removeArtifactsFromPage(&sd, resDict, pageNr)
			if err != nil {
				return false, err
			}
			if !found && ok {
				found = true
				entry.Object = sd
			}
		}

	}
	return found, nil
}

func (ctx *Context) removePageWatermark(pageNr int) (bool, error) {
	o, pageDictIndRef, resDict, err := ctx.locatePageContentAndResourceDict(pageNr)
	if err != nil {
		return false, err
	}

	var entry *XRefTableEntry

	ir, ok := o.(IndirectRef)
	if ok {
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		o = entry.Object
	}

	found, err := ctx.removeArtifacts(o, entry, resDict, pageNr)
	if err != nil {
		return false, err
	}

	/*
		Supposedly the form needs a PieceInfo in order to be recognized by Acrobat like so:

			<PieceInfo, <<
				<ADBE_CompoundType, <<
					<DocSettings, (61 0 R)>
					<LastModified, (D:20190830152436+02'00')>
					<Private, Watermark>
				>>>
			>>>

	*/

	if found {
		// Remove any associated link annotations.
		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return false, err
		}
		objNr := pageDictIndRef.ObjectNumber.Value()
		if _, err = ctx.RemoveAnnotationsFromPageDict([]string{"pdfcpu"}, nil, d, objNr, pageNr, false); err != nil {
			return false, err
		}
	}

	return found, nil
}

func (ctx *Context) locateOCGs() (Array, error) {
	rootDict, err := ctx.Catalog()
	if err != nil {
		return nil, err
	}

	o, ok := rootDict.Find("OCProperties")
	if !ok {
		return nil, errNoWatermark
	}

	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	o, found := d.Find("OCGs")
	if !found {
		return nil, errNoWatermark
	}

	return ctx.DereferenceArray(o)
}

// RemoveWatermarks removes watermarks for all pages selected.
func (ctx *Context) RemoveWatermarks(selectedPages IntSet) error {
	log.Debug.Printf("RemoveWatermarks\n")

	a, err := ctx.locateOCGs()
	if err != nil {
		return err
	}

	found := false

	for _, o := range a {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		if *d.Type() != "OCG" {
			continue
		}

		n := d.StringEntry("Name")
		if n == nil {
			continue
		}

		if *n != "Background" && *n != "Watermark" {
			continue
		}

		found = true
		break
	}

	if !found {
		return errNoWatermark
	}

	var removedSmth bool

	for k, v := range selectedPages {
		if !v {
			continue
		}

		ok, err := ctx.removePageWatermark(k)
		if err != nil {
			return err
		}

		if ok {
			removedSmth = true
		}
	}

	if !removedSmth {
		return errNoWatermark
	}

	return nil
}

func detectArtifacts(sd *StreamDict) (bool, error) {
	if err := sd.Decode(); err != nil {
		return false, err
	}
	// Watermarks may begin or end the content stream.
	i := strings.Index(string(sd.Content), "/Artifact <</Subtype /Watermark /Type /Pagination >>BDC")
	return i >= 0, nil
}

func (ctx *Context) findPageWatermarks(pageDictIndRef *IndirectRef) (bool, error) {
	d, err := ctx.DereferenceDict(*pageDictIndRef)
	if err != nil {
		return false, err
	}

	o, found := d.Find("Contents")
	if !found {
		return false, errNoContent
	}

	var entry *XRefTableEntry

	ir, ok := o.(IndirectRef)
	if ok {
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ = ctx.FindTableEntry(objNr, genNr)
		o = entry.Object
	}

	switch o := o.(type) {

	case StreamDict:
		return detectArtifacts(&o)

	case Array:
		// Get stream dict for first element.
		o1 := o[0]
		ir, _ := o1.(IndirectRef)
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(StreamDict)
		ok, err := detectArtifacts(&sd)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}

		if len(o) > 1 {
			// Get stream dict for last element.
			o1 := o[len(o)-1]
			ir, _ := o1.(IndirectRef)
			objNr = ir.ObjectNumber.Value()
			genNr := ir.GenerationNumber.Value()
			entry, _ := ctx.FindTableEntry(objNr, genNr)
			sd, _ := (entry.Object).(StreamDict)
			return detectArtifacts(&sd)
		}

	}

	return false, nil
}

func (ctx *Context) detectPageTreeWatermarks(root *IndirectRef) error {
	d, err := ctx.DereferenceDict(*root)
	if err != nil {
		return err
	}

	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return nil
	}

	for _, o := range kids {

		if ctx.Watermarked {
			return nil
		}

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(IndirectRef)
		if !ok {
			return errors.Errorf("pdfcpu: detectPageTreeWatermarks: corrupt page node dict")
		}

		pageNodeDict, err := ctx.DereferenceDict(ir)
		if err != nil {
			return err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			if err := ctx.detectPageTreeWatermarks(&ir); err != nil {
				return err
			}

		case "Page":
			found, err := ctx.findPageWatermarks(&ir)
			if err != nil {
				return err
			}
			if found {
				ctx.Watermarked = true
				return nil
			}

		}
	}

	return nil
}

// DetectPageTreeWatermarks checks xRefTable's page tree for watermarks
// and records the result to xRefTable.Watermarked.
func (ctx *Context) DetectPageTreeWatermarks() error {
	root, err := ctx.Pages()
	if err != nil {
		return err
	}
	return ctx.detectPageTreeWatermarks(root)
}

// DetectWatermarks checks ctx for watermarks
// and records the result to xRefTable.Watermarked.
func (ctx *Context) DetectWatermarks() error {
	a, err := ctx.locateOCGs()
	if err != nil {
		if err == errNoWatermark {
			ctx.Watermarked = false
			return nil
		}
		return err
	}

	found := false

	for _, o := range a {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if o == nil {
			continue
		}

		if *d.Type() != "OCG" {
			continue
		}

		n := d.StringEntry("Name")
		if n == nil {
			continue
		}

		if *n != "Background" && *n != "Watermark" {
			continue
		}

		found = true
		break
	}

	if !found {
		ctx.Watermarked = false
		return nil
	}

	return ctx.DetectPageTreeWatermarks()
}

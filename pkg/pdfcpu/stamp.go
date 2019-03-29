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
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu/fonts/metrics"

	"github.com/pkg/errors"
)

const (
	degToRad = math.Pi / 180
	radToDeg = 180 / math.Pi
)

type matrix [3][3]float64

var errNoContent = errors.New("PDF page has no content")

var identMatrix = matrix{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

func (m matrix) multiply(n matrix) matrix {
	var p matrix
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				p[i][j] += m[i][k] * n[k][j]
			}
		}
	}
	return p
}

func (m matrix) String() string {
	return fmt.Sprintf("%3.2f %3.2f %3.2f\n%3.2f %3.2f %3.2f\n%3.2f %3.2f %3.2f\n",
		m[0][0], m[0][1], m[0][2],
		m[1][0], m[1][1], m[1][2],
		m[2][0], m[2][1], m[2][2])
}

type simpleColor struct {
	r, g, b float32 // intensities between 0 and 1.
}

func (sc simpleColor) String() string {
	return fmt.Sprintf("r=%1.1f g=%1.1f b=%1.1f", sc.r, sc.g, sc.b)
}

const (
	noDiagonal = iota
	diagonalLLToUR
	diagonalULToLR
)

// render mode
const (
	rmFill = iota
	rmStroke
	rmFillAndStroke
)

type formCache map[*Rectangle]*IndirectRef

// Watermark represents the basic structure and command details for the commands "Stamp" and "Watermark".
type Watermark struct {

	// configuration
	textString     string      // raw display text.
	textLines      []string    // display multiple lines of text.
	fileName       string      // display pdf page or png image.
	page           int         // the page number of a PDF file.
	onTop          bool        // if true this is a STAMP else this is a WATERMARK.
	fontName       string      // supported are Adobe base fonts only. (as of now: Helvetica, Times-Roman, Courier)
	fontSize       int         // font scaling factor.
	scaledFontSize int         // font scaling factor for a specific page
	color          simpleColor // fill color(=non stroking color).
	rotation       float64     // rotation to apply in degrees. -180 <= x <= 180
	diagonal       int         // paint along the diagonal.
	opacity        float64     // opacity the displayed text. 0 <= x <= 1
	renderMode     int         // fill=0, stroke=1 fill&stroke=2
	scale          float64     // relative scale factor. 0 <= x <= 1
	scaleAbs       bool        // true for absolute scaling.

	// resources
	ocg, extGState, font, image *IndirectRef

	// for an image or PDF watermark
	width, height int // image or page dimensions.

	// for a PDF watermark
	resDict *IndirectRef // page resource dict.
	cs      []byte       // page content stream.

	// page specific
	bb      *Rectangle   // bounding box of the form representing this watermark.
	vp      *Rectangle   // page dimensions for text alignment.
	pageRot float64      // page rotation in effect.
	form    *IndirectRef // Forms are dependent on given page dimensions.

	// house keeping
	objs   IntSet    // objects for which wm has been applied already.
	fCache formCache // form cache.
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
	if !wm.onTop {
		s = "not "
	}

	t := wm.textString
	if len(t) == 0 {
		t = wm.fileName
	}

	sc := "relative"
	if wm.scaleAbs {
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
		"pageRotation: %.1f\n",
		t, s, wm.typ(),
		wm.fontName, wm.fontSize,
		wm.page,
		wm.scale, sc,
		wm.color,
		wm.rotation,
		wm.diagonal,
		wm.opacity,
		wm.renderMode,
		bbox,
		vp,
		wm.pageRot,
	)
}

// OnTopString returns "watermark" or "stamp" whichever applies.
func (wm Watermark) OnTopString() string {
	s := "watermark"
	if wm.onTop {
		s = "stamp"
	}
	return s
}

func (wm Watermark) calcMaxTextWidth() float64 {

	var maxWidth float64

	for _, l := range wm.textLines {
		w := metrics.TextWidth(l, wm.fontName, wm.scaledFontSize)
		if w > maxWidth {
			maxWidth = w
		}
	}

	return maxWidth
}

func (wm Watermark) calcMinFontSize(w float64) int {

	var minSize int

	for _, l := range wm.textLines {
		w := metrics.FontSize(l, wm.fontName, w)
		if minSize == 0.0 {
			minSize = w
		}
		if w < minSize {
			minSize = w
		}
	}

	return minSize
}

// IsPDF returns whether the watermark content is an image or text.
func (wm Watermark) isPDF() bool {
	return len(wm.fileName) > 0 && strings.ToLower(filepath.Ext(wm.fileName)) == ".pdf"
}

// IsImage returns whether the watermark content is an image or text.
func (wm Watermark) isImage() bool {
	return len(wm.fileName) > 0 && strings.ToLower(filepath.Ext(wm.fileName)) != ".pdf"
}

func (wm *Watermark) calcBoundingBox() {

	//fmt.Println("calcBoundingBox:")

	var bb *Rectangle

	if wm.isImage() || wm.isPDF() {

		bb = RectForDim(wm.width, wm.height)
		ar := bb.AspectRatio()
		//fmt.Printf("calcBB: ar:%f scale:%f\n", ar, wm.scale)
		//fmt.Printf("vp: %s\n", wm.vp)

		if wm.scaleAbs {
			bb.UR.X = wm.scale * bb.Width()
			bb.UR.Y = bb.UR.X / ar

			wm.bb = bb
			return
		}

		if ar >= 1 {
			bb.UR.X = wm.scale * wm.vp.Width()
			bb.UR.Y = bb.UR.X / ar
			//fmt.Printf("ar>1: %s\n", bb)
		} else {
			bb.UR.Y = wm.scale * wm.vp.Height()
			bb.UR.X = bb.UR.Y * ar
			//fmt.Printf("ar<=1: %s\n", bb)
		}

		wm.bb = bb
		return
	}

	// font watermark

	var w float64
	if wm.scaleAbs {
		wm.scaledFontSize = int(float64(wm.fontSize) * wm.scale)
		w = wm.calcMaxTextWidth()
	} else {
		w = wm.scale * wm.vp.Width()
		wm.scaledFontSize = wm.calcMinFontSize(w)
	}
	h := float64(len(wm.textLines)) * float64(wm.scaledFontSize)
	wm.bb = Rect(0, 0, w, h)

	return
}

func (wm *Watermark) calcTransformMatrix() *matrix {

	var sin, cos float64
	r := wm.rotation

	if wm.diagonal != noDiagonal {

		// Calculate the angle of the diagonal with respect of the aspect ratio of the bounding box.
		r = math.Atan(wm.vp.Height()/wm.vp.Width()) * float64(radToDeg)

		if wm.bb.AspectRatio() < 1 {
			r -= 90
		}

		if wm.diagonal == diagonalULToLR {
			r = -r
		}

	}

	// Apply negative page rotation.
	r += wm.pageRot

	sin = math.Sin(float64(r) * float64(degToRad))
	cos = math.Cos(float64(r) * float64(degToRad))

	// 1) Rotate
	m1 := identMatrix
	m1[0][0] = cos
	m1[0][1] = sin
	m1[1][0] = -sin
	m1[1][1] = cos

	// 2) Translate
	m2 := identMatrix

	var dy float64
	if !wm.isImage() && !wm.isPDF() {
		dy = wm.bb.LL.Y
	}

	m2[2][0] = wm.vp.LL.X + wm.vp.Width()/2 + sin*(wm.bb.Height()/2+dy) - cos*wm.bb.Width()/2
	m2[2][1] = wm.vp.LL.Y + wm.vp.Height()/2 - cos*(wm.bb.Height()/2+dy) - sin*wm.bb.Width()/2

	m := m1.multiply(m2)
	return &m
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

func oneWatermarkOnlyError(onTop bool) error {
	s := onTopString(onTop)
	return errors.Errorf("Cannot apply %s. Only one watermark/stamp allowed.\n", s)
}

func setWatermarkType(s string, wm *Watermark) error {

	ss := strings.Split(s, ":")

	wm.textString = ss[0]

	for _, l := range strings.Split(ss[0], `\n`) {
		wm.textLines = append(wm.textLines, l)
	}

	if len(wm.textLines) > 1 {
		// Multiline text watermark.
		return nil
	}

	ext := strings.ToLower(filepath.Ext(ss[0]))
	if MemberOf(ext, []string{".jpg", ".jpeg", ".png", ".tif", ".tiff", ".pdf"}) {
		wm.fileName = wm.textString
	}

	if len(ss) > 1 {
		// Parse page number for PDF watermarks.
		var err error
		wm.page, err = strconv.Atoi(ss[1])
		if err != nil {
			return errors.Errorf("illegal page number value: %s\n", ss[1])
		}
	}

	return nil
}

func supportedWatermarkFont(fn string) bool {
	for _, s := range metrics.FontNames() {
		if fn == s {
			return true
		}
	}
	return false
}

func parseWatermarkFontSize(s string, wm *Watermark) error {

	fs, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	wm.fontSize = fs

	return nil
}

func parseScaleFactor(s string) (float64, bool, error) {

	ss := strings.Split(s, " ")
	if len(ss) > 2 {
		return 0, false, errors.Errorf("illegal scale string: 0.0 <= i <= 1.0 {abs|rel}, %s\n", s)
	}

	sc, err := strconv.ParseFloat(ss[0], 64)
	if err != nil {
		return 0, false, errors.Errorf("scale factor must be a float value: %s\n", ss[0])
	}
	if sc < 0 || sc > 1 {
		return 0, false, errors.Errorf("illegal scale factor: 0.0 <= s <= 1.0, %s\n", ss[0])
	}

	var scaleAbs bool

	if len(ss) == 2 {
		switch ss[1] {
		case "a", "abs":
			scaleAbs = true

		case "r", "rel":
			scaleAbs = false

		default:
			return 0, false, errors.Errorf("illegal scale mode: abs|rel, %s\n", ss[1])
		}
	}

	return sc, scaleAbs, nil
}

func parseWatermarkColor(s string, wm *Watermark) error {

	cs := strings.Split(s, " ")
	if len(cs) != 3 {
		return errors.Errorf("illegal color string: 3 intensities 0.0 <= i <= 1.0, %s\n", s)
	}

	r, err := strconv.ParseFloat(cs[0], 32)
	if err != nil {
		return errors.Errorf("red must be a float value: %s\n", cs[0])
	}
	if r < 0 || r > 1 {
		return errors.New("red: a color value is an intensity between 0.0 and 1.0")
	}
	wm.color.r = float32(r)

	g, err := strconv.ParseFloat(cs[1], 32)
	if err != nil {
		return errors.Errorf("green must be a float value: %s\n", cs[1])
	}
	if g < 0 || g > 1 {
		return errors.New("green: a color value is an intensity between 0.0 and 1.0")
	}
	wm.color.g = float32(g)

	b, err := strconv.ParseFloat(cs[2], 32)
	if err != nil {
		return errors.Errorf("blue must be a float value: %s\n", cs[2])
	}
	if b < 0 || b > 1 {
		return errors.New("blue: a color value is an intensity between 0.0 and 1.0")
	}
	wm.color.b = float32(b)

	return nil
}

func parseWatermarkRotation(s string, setDiag bool, wm *Watermark) error {

	if setDiag {
		return errors.New("Please specify rotation or diagonal (r or d)")
	}

	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("rotation must be a float value: %s\n", s)
	}
	if r < -180 || r > 180 {
		return errors.Errorf("illegal rotation: -180 <= r <= 180 degrees, %s\n", s)
	}

	wm.rotation = r
	wm.diagonal = noDiagonal

	return nil
}

func parseWatermarkDiagonal(s string, setRot bool, wm *Watermark) error {

	if setRot {
		return errors.New("Please specify rotation or diagonal (r or d)")
	}

	d, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("illegal diagonal value: allowed 1 or 2, %s\n", s)
	}
	if d != diagonalLLToUR && d != diagonalULToLR {
		return errors.New("diagonal: 1..lower left to upper right, 2..upper left to lower right")
	}

	wm.diagonal = d
	wm.rotation = 0

	return nil
}

func parseWatermarkOpacity(s string, wm *Watermark) error {

	o, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return errors.Errorf("opacity must be a float value: %s\n", s)
	}
	if o < 0 || o > 1 {
		return errors.Errorf("illegal opacity: 0.0 <= r <= 1.0, %s\n", s)
	}
	wm.opacity = o

	return nil
}

func parseWatermarkRenderMode(s string, wm *Watermark) error {

	m, err := strconv.Atoi(s)
	if err != nil {
		return errors.Errorf("illegal mode value: allowed 0,1,2, %s\n", s)
	}
	if m != rmFill && m != rmStroke && m != rmFillAndStroke {
		return errors.New("Valid rendermodes: 0..fill, 1..stroke, 2..fill&stroke")
	}
	wm.renderMode = m

	return nil
}

// ParseWatermarkDetails parses a Watermark/Stamp command string into an internal structure.
func ParseWatermarkDetails(s string, onTop bool) (*Watermark, error) {

	//fmt.Printf("watermark details: <%s>\n", s)

	// Set default watermark
	wm := Watermark{
		onTop:      onTop,
		page:       1,
		fontName:   "Helvetica",
		fontSize:   24,
		scale:      0.5,
		scaleAbs:   false,
		color:      simpleColor{0.5, 0.5, 0.5}, // gray
		diagonal:   diagonalLLToUR,
		opacity:    1.0,
		renderMode: rmFill,
		objs:       IntSet{},
		fCache:     formCache{},
		textLines:  []string{},
	}

	ss := strings.Split(s, ",")

	setWatermarkType(ss[0], &wm)

	if len(ss) == 1 {
		return &wm, nil
	}

	var setDiag, setRot bool

	for _, s := range ss[1:] {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, parseWatermarkError(onTop)
		}

		k := strings.TrimSpace(ss1[0])
		v := strings.TrimSpace(ss1[1])

		var err error

		switch k {
		case "f": // font name
			if !supportedWatermarkFont(v) {
				err = errors.Errorf("%s is unsupported, try one of Helvetica, Times-Roman, Courier.\n", v)
			}
			wm.fontName = v

		case "p": // font size in points
			err = parseWatermarkFontSize(v, &wm)

		case "s": // scale factor
			wm.scale, wm.scaleAbs, err = parseScaleFactor(v)

		case "c": // color
			err = parseWatermarkColor(v, &wm)

		case "r": // rotation
			err = parseWatermarkRotation(v, setDiag, &wm)
			setRot = true

		case "d": // diagonal
			err = parseWatermarkDiagonal(v, setRot, &wm)
			setDiag = true

		case "o": // opacity
			err = parseWatermarkOpacity(v, &wm)

		case "m": // render mode
			err = parseWatermarkRenderMode(v, &wm)

		default:
			err = parseWatermarkError(onTop)
		}

		if err != nil {
			return nil, err
		}
	}

	return &wm, nil
}

func createFontResForWM(xRefTable *XRefTable, wm *Watermark) error {

	d := NewDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", wm.fontName)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	wm.font = ir

	return nil
}

func contentStream(xRefTable *XRefTable, o Object) ([]byte, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return nil, err
	}

	bb := []byte{}

	switch o := o.(type) {

	case StreamDict:

		// Decode streamDict for supported filters only.
		err := decodeStream(&o)
		if err == filter.ErrUnsupportedFilter {
			return nil, errors.New("unsupported filter: unable to decode content for PDF watermark")
		}
		if err != nil {
			return nil, err
		}

		//fmt.Printf("found %d content bytes\n", len(o.Content))
		bb = o.Content

	case Array:
		for _, o := range o {

			if o == nil {
				continue
			}

			sd, err := xRefTable.DereferenceStreamDict(o)
			if err != nil {
				return nil, err
			}

			// Decode streamDict for supported filters only.
			err = decodeStream(sd)
			if err == filter.ErrUnsupportedFilter {
				return nil, errors.New("unsupported filter: unable to decode content for PDF watermark")
			}
			if err != nil {
				return nil, err
			}

			//fmt.Printf("append %d content bytes\n", len(sd.Content))
			bb = append(bb, sd.Content...)
			//fmt.Printf("len bb = %d\n", len(bb))
		}
	}

	if len(bb) == 0 {
		return nil, errNoContent
	}

	return bb, nil
}

func identifyObjNrs(ctx *Context, o Object, objNrs IntSet) error {

	switch o := o.(type) {

	case IndirectRef:

		objNrs[o.ObjectNumber.Value()] = true

		o1, err := ctx.Dereference(o)
		if err != nil {
			return err
		}

		err = identifyObjNrs(ctx, o1, objNrs)
		if err != nil {
			return err
		}

	case Dict:
		for _, v := range o {
			err := identifyObjNrs(ctx, v, objNrs)
			if err != nil {
				return err
			}
		}

	case StreamDict:
		for _, v := range o.Dict {
			err := identifyObjNrs(ctx, v, objNrs)
			if err != nil {
				return err
			}
		}

	case Array:
		for _, v := range o {
			err := identifyObjNrs(ctx, v, objNrs)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func migrateObject(ctxSource, ctxDest *Context, o Object) error {

	//fmt.Printf("migrateObject start  %s\n", o)

	// Identify involved objNrs.
	objNrs := IntSet{}
	err := identifyObjNrs(ctxSource, o, objNrs)
	if err != nil {
		return err
	}
	//fmt.Printf("objNrs = %v\n", objNrs)

	// Create a lookup table mapping objNrs from ctxSource to ctxDest.
	// Create lookup table for object numbers.
	// The first number is the successor of the last number in ctxDest.
	lookup := lookupTable(objNrs, *ctxDest.Size)
	//fmt.Printf("lookup = %v\n", lookup)

	// Patch indRefs of resourceDict.
	patchObject(o, lookup)
	//fmt.Printf("o(resDict) patched: %s\n", o)

	// Patch all involved indRefs.
	for i := range lookup {
		//fmt.Printf("before patching old obj %d\n%s\n", i, ctxSource.Table[i].Object)
		patchObject(ctxSource.Table[i].Object, lookup)
		//fmt.Printf("after patching old obj\n%s\n", ctxSource.Table[i].Object)
	}

	// Migrate xrefTableEntries.
	for k, v := range lookup {
		//fmt.Printf("dest[%d]=src[%d]\n", v, k)
		ctxDest.Table[v] = ctxSource.Table[k]
		*ctxDest.Size++
	}

	//fmt.Printf("migrateObject end  %s\n", o)

	return nil
}

func createPDFResForWM(ctx *Context, wm *Watermark) error {

	xRefTable := ctx.XRefTable

	// This PDF file is assumed to be valid.
	otherCtx, err := ReadFile(wm.fileName, NewDefaultConfiguration())
	if err != nil {
		return err
	}

	otherXRefTable := otherCtx.XRefTable

	d, inhPAttrs, err := otherXRefTable.PageDict(wm.page)
	if err != nil {
		return err
	}
	if d == nil {
		return errors.Errorf("unknown page number: %d\n", wm.page)
	}

	// Retrieve content stream bytes.

	o, found := d.Find("Contents")
	if !found {
		return errors.New("PDF page has no content")
	}

	wm.cs, err = contentStream(otherXRefTable, o)
	if err != nil {
		return err
	}

	// Migrate all objects referenced by this external resource dict into this context.
	err = migrateObject(otherCtx, ctx, inhPAttrs.resources)
	if err != nil {
		return err
	}

	//fmt.Printf("migrated resDict inhPAttrs.resources %s\n", inhPAttrs.resources)

	// Create an object for this resDict in xRefTable.
	ir, err := xRefTable.IndRefForNewObject(inhPAttrs.resources)
	if err != nil {
		return err
	}

	wm.resDict = ir

	vp := viewPort(xRefTable, inhPAttrs)
	//fmt.Printf("createPDFResForWM: vp = %s\n", vp)

	wm.width = int(vp.Width())
	wm.height = int(vp.Height())

	return nil
}

func createImageResource(xRefTable *XRefTable, fileName string) (*IndirectRef, int, int, error) {

	ext := strings.ToLower(filepath.Ext(fileName))
	var f func(xRefTable *XRefTable, fileName string) (*StreamDict, error)

	switch ext {
	case ".png":
		f = ReadPNGFile
	case ".jpg", ".jpeg":
		f = ReadJPEGFile
	case ".tif", ".tiff":
		f = ReadTIFFFile
	default:
		return nil, 0, 0, errors.Errorf("unsupported imagefile extension: %s", ext)
	}

	sd, err := f(xRefTable, fileName)
	if err != nil {
		return nil, 0, 0, err
	}

	w := *sd.IntEntry("Width")
	h := *sd.IntEntry("Height")

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, 0, 0, err
	}

	return indRef, w, h, nil
}

func createImageResForWM(xRefTable *XRefTable, wm *Watermark) (err error) {

	wm.image, wm.width, wm.height, err = createImageResource(xRefTable, wm.fileName)

	return err
}

func createResourcesForWM(ctx *Context, wm *Watermark) error {

	log.Debug.Println("createResourcesForWM begin")

	xRefTable := ctx.XRefTable

	if wm.isPDF() {
		return createPDFResForWM(ctx, wm)
	}

	if wm.isImage() {
		return createImageResForWM(xRefTable, wm)
	}

	return createFontResForWM(xRefTable, wm)
}

func createOCG(xRefTable *XRefTable, wm *Watermark) error {

	name := "Background"
	subt := "BG"
	if wm.onTop {
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

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	wm.ocg = ir

	return nil
}

func prepareOCPropertiesInRoot(rootDict Dict, wm *Watermark) error {

	optionalContentConfigDict := Dict(
		map[string]Object{
			"AS": Array{
				Dict(
					map[string]Object{
						"Category": NewNameArray("View"),
						"Event":    Name("View"),
						"OCGs":     Array{*wm.ocg},
					},
				),
				Dict(
					map[string]Object{
						"Category": NewNameArray("Print"),
						"Event":    Name("Print"),
						"OCGs":     Array{*wm.ocg},
					},
				),
				Dict(
					map[string]Object{
						"Category": NewNameArray("Export"),
						"Event":    Name("Export"),
						"OCGs":     Array{*wm.ocg},
					},
				),
			},
			"ON":       Array{*wm.ocg},
			"Order":    Array{},
			"RBGroups": Array{},
		},
	)

	d := Dict(
		map[string]Object{
			"OCGs": Array{*wm.ocg},
			"D":    optionalContentConfigDict,
		},
	)

	_, ok := rootDict.Find("OCProperties")
	if !ok {
		rootDict.Insert("OCProperties", d)
		return nil
	}

	return oneWatermarkOnlyError(wm.onTop)
}

func createFormResDict(xRefTable *XRefTable, wm *Watermark) (*IndirectRef, error) {

	if wm.isPDF() {
		//return consolidated ResDict of for PDF/PageBoxColorInfo
		return wm.resDict, nil
	}

	if wm.isImage() {

		d := Dict(
			map[string]Object{
				"ProcSet": NewNameArray("PDF", "ImageC"),
				"XObject": Dict(map[string]Object{"Im0": *wm.image}),
			},
		)

		return xRefTable.IndRefForNewObject(d)

	}

	d := Dict(
		map[string]Object{
			"Font":    Dict(map[string]Object{wm.fontName: *wm.font}),
			"ProcSet": NewNameArray("PDF", "Text"),
		},
	)

	return xRefTable.IndRefForNewObject(d)
}

func createForm(xRefTable *XRefTable, wm *Watermark, withBB bool) error {

	// The forms bounding box is dependent on the page dimensions.
	wm.calcBoundingBox()
	bb := wm.bb

	// Cache the form for every bounding box encountered.
	ir, ok := wm.fCache[bb] // *bb ??
	if ok {
		//fmt.Printf("reusing form obj#%d\n", ir.ObjectNumber)
		wm.form = ir
		return nil
	}

	var b bytes.Buffer

	if wm.isPDF() {
		fmt.Fprintf(&b, "%f 0 0 %f 0 0 cm ", bb.Width()/wm.vp.Width(), bb.Height()/wm.vp.Height())
		_, err := b.Write(wm.cs)
		if err != nil {
			return err
		}
	} else if wm.isImage() {
		fmt.Fprintf(&b, "q %f 0 0 %f 0 0 cm /Im0 Do Q", bb.Width(), bb.Height()) // TODO dont need Q
	} else {

		// 12 font points result in a vertical displacement of 9.47
		dy := -float64(wm.scaledFontSize) / 12 * 9.47

		wmForm := "0 g 0 G 0 i 0 J []0 d 0 j 1 w 10 M 0 Tc 0 Tw 100 Tz 0 TL %d Tr 0 Ts "
		fmt.Fprintf(&b, wmForm, wm.renderMode)

		j := 1
		for i := len(wm.textLines) - 1; i >= 0; i-- {

			sw := metrics.TextWidth(wm.textLines[i], wm.fontName, wm.scaledFontSize)
			dx := wm.bb.Width()/2 - sw/2

			fmt.Fprintf(&b, "BT /%s %d Tf %f %f %f rg %f %f Td (%s) Tj ET ",
				wm.fontName, wm.scaledFontSize, wm.color.r, wm.color.g, wm.color.b, dx, dy+float64(j*wm.scaledFontSize), wm.textLines[i])
			j++
		}

	}

	// Paint bounding box
	if withBB {
		fmt.Fprintf(&b, "[]0 d 2 w %f %f m %f %f l %f %f l %f %f l s ",
			bb.LL.X, bb.LL.Y,
			bb.UR.X, bb.LL.Y,
			bb.UR.X, bb.UR.Y,
			bb.LL.X, bb.UR.Y,
		)
	}

	ir, err := createFormResDict(xRefTable, wm)
	if err != nil {
		return err
	}

	sd := StreamDict{
		Dict: Dict(
			map[string]Object{
				"Type":      Name("XObject"),
				"Subtype":   Name("Form"),
				"BBox":      bb.Array(),
				"Matrix":    NewIntegerArray(1, 0, 0, 1, 0, 0),
				"OC":        *wm.ocg,
				"Resources": *ir,
			},
		),
		Content: b.Bytes(),
	}

	err = encodeStream(&sd)
	if err != nil {
		return err
	}

	ir, err = xRefTable.IndRefForNewObject(sd)
	if err != nil {
		return err
	}

	//fmt.Printf("caching form obj#%d\n", ir.ObjectNumber)
	wm.fCache[wm.bb] = ir

	wm.form = ir

	return nil
}

func createExtGStateForStamp(xRefTable *XRefTable, wm *Watermark) error {

	log.Debug.Println("createExtGStateForStamp begin")

	d := Dict(
		map[string]Object{
			"Type": Name("ExtGState"),
			"CA":   Float(wm.opacity),
			"ca":   Float(wm.opacity),
		},
	)

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	wm.extGState = ir

	return nil
}

func insertPageResourcesForWM(xRefTable *XRefTable, pageDict Dict, wm *Watermark, gsID, xoID string) error {

	resourceDict := Dict(
		map[string]Object{
			"ExtGState": Dict(map[string]Object{gsID: *wm.extGState}),
			"XObject":   Dict(map[string]Object{xoID: *wm.form}),
		},
	)

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func updatePageResourcesForWM(xRefTable *XRefTable, resDict Dict, wm *Watermark, gsID, xoID *string) error {

	o, ok := resDict.Find("ExtGState")
	if !ok {
		resDict.Insert("ExtGState", Dict(map[string]Object{*gsID: *wm.extGState}))
	} else {
		d, _ := xRefTable.DereferenceDict(o)
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
		d, _ := xRefTable.DereferenceDict(o)
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

	insertOCG := " /Artifact <</Subtype /Watermark /Type /Pagination >>BDC q %.2f %.2f %.2f %.2f %.2f %.2f cm /%s gs /%s Do Q EMC "

	var b bytes.Buffer
	fmt.Fprintf(&b, insertOCG, m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], gsID, xoID)

	return b.Bytes()
}

func insertPageContentsForWM(xRefTable *XRefTable, pageDict Dict, wm *Watermark, gsID, xoID string) error {

	sd := &StreamDict{Dict: NewDict()}

	sd.Content = wmContent(wm, gsID, xoID)

	err := encodeStream(sd)
	if err != nil {
		return err
	}

	ir, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *ir)

	return nil
}

func updatePageContentsForWM(xRefTable *XRefTable, obj Object, wm *Watermark, gsID, xoID string) error {

	var entry *XRefTableEntry
	var objNr int

	ir, ok := obj.(IndirectRef)
	if ok {
		objNr = ir.ObjectNumber.Value()
		if wm.objs[objNr] {
			// wm already applied to this content stream.
			log.Debug.Printf("wm already applied to content stream obj: %d\n", objNr)
			return nil
		}
		genNr := ir.GenerationNumber.Value()
		entry, _ = xRefTable.FindTableEntry(objNr, genNr)
		obj = entry.Object
	}

	switch o := obj.(type) {

	case StreamDict:

		//fmt.Printf("%T %T\n", &o, o)
		//fmt.Printf("Content obj#%d addr:%v\n%s\n", objNr, &o, o)
		log.Debug.Printf("patching content stream obj:%d\n", objNr)

		err := patchContentForWM(&o, gsID, xoID, wm, true)
		if err != nil {
			return err
		}

		//fmt.Printf("content after patch:\n%s\n", o)

		entry.Object = o
		wm.objs[objNr] = true

	case Array:

		// Get stream dict for first element.
		o1 := o[0]
		ir, _ := o1.(IndirectRef)
		objNr = ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ := xRefTable.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(StreamDict)

		if len(o) == 1 || !wm.onTop {

			if wm.objs[objNr] {
				// wm already applied to this content stream.
				log.Debug.Printf("wm already applied to first=last content stream obj: %d\n", objNr)
				return nil
			}

			log.Debug.Printf("patching first=last content stream obj:%d\n", objNr)
			err := patchContentForWM(&sd, gsID, xoID, wm, true)
			if err != nil {
				return err
			}
			entry.Object = sd
			wm.objs[objNr] = true
			return nil
		}

		if wm.objs[objNr] {
			// wm already applied to this content stream.
			log.Debug.Printf("wm already applied to first content stream obj: %d\n", objNr)
		} else {
			// Patch first content stream.
			log.Debug.Printf("patching first content stream obj:%d\n", objNr)
			err := patchFirstContentForWM(&sd)
			if err != nil {
				return err
			}
			entry.Object = sd
			wm.objs[objNr] = true
		}

		// Patch last content stream.
		o1 = o[len(o)-1]

		ir, _ = o1.(IndirectRef)
		objNr = ir.ObjectNumber.Value()
		if wm.objs[objNr] {
			// wm already applied to this content stream.
			log.Debug.Printf("wm already applied to last content stream obj:%d\n", objNr)
			return nil
		}

		log.Debug.Printf("patching last content stream obj:%d\n", objNr)
		genNr = ir.GenerationNumber.Value()
		entry, _ = xRefTable.FindTableEntry(objNr, genNr)
		sd, _ = (entry.Object).(StreamDict)

		err := patchContentForWM(&sd, gsID, xoID, wm, false)
		if err != nil {
			return err
		}

		entry.Object = sd
		wm.objs[objNr] = true
	}

	return nil
}

func viewPort(xRefTable *XRefTable, a *InheritedPageAttrs) *Rectangle {

	visibleRegion := a.mediaBox
	if a.cropBox != nil {
		visibleRegion = a.cropBox
	}

	return visibleRegion
}

func watermarkPage(xRefTable *XRefTable, i int, wm *Watermark) error {

	log.Debug.Printf("watermark page:%d\n", i)

	d, inhPAttrs, err := xRefTable.PageDict(i)
	if err != nil {
		return err
	}

	wm.vp = viewPort(xRefTable, inhPAttrs)
	//log.Debug.Printf("watermarkPage: vp = %s\n", wm.vp)

	err = createForm(xRefTable, wm, false)
	if err != nil {
		return err
	}

	wm.pageRot = inhPAttrs.rotate

	log.Debug.Printf("\n%s\n", wm)

	gsID := "GS0"
	xoID := "Fm0"

	if inhPAttrs.resources == nil {
		err = insertPageResourcesForWM(xRefTable, d, wm, gsID, xoID)
	} else {
		err = updatePageResourcesForWM(xRefTable, inhPAttrs.resources, wm, &gsID, &xoID)
	}
	if err != nil {
		return err
	}

	obj, found := d.Find("Contents")
	if !found {
		return insertPageContentsForWM(xRefTable, d, wm, gsID, xoID)
	}

	return updatePageContentsForWM(xRefTable, obj, wm, gsID, xoID)
}

func patchContentForWM(sd *StreamDict, gsID, xoID string, wm *Watermark, saveGState bool) error {

	// Decode streamDict for supported filters only.
	err := decodeStream(sd)
	if err == filter.ErrUnsupportedFilter {
		log.Info.Println("unsupported filter: unable to patch content with watermark.")
		return nil
	}
	if err != nil {
		return err
	}

	bb := wmContent(wm, gsID, xoID)

	if wm.onTop {
		if saveGState {
			sd.Content = append([]byte("q "), sd.Content...)
		}
		sd.Content = append(sd.Content, []byte(" Q")...)
		sd.Content = append(sd.Content, bb...)
	} else {
		sd.Content = append(bb, sd.Content...)
	}
	//fmt.Printf("patched content:\n%s\n", hex.Dump(sd.Content))

	return encodeStream(sd)
}

func patchFirstContentForWM(sd *StreamDict) error {

	err := decodeStream(sd)
	if err == filter.ErrUnsupportedFilter {
		log.Info.Println("unsupported filter: unable to patch content with watermark.")
		return nil
	}
	if err != nil {
		return err
	}

	sd.Content = append([]byte("q "), sd.Content...)

	return encodeStream(sd)
}

// AddWatermarks adds watermarks to all pages selected.
func AddWatermarks(ctx *Context, selectedPages IntSet, wm *Watermark) error {

	log.Debug.Printf("AddWatermarks wm:\n%s\n", wm)

	xRefTable := ctx.XRefTable

	err := createOCG(xRefTable, wm)
	if err != nil {
		return err
	}

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	err = prepareOCPropertiesInRoot(rootDict, wm)
	if err != nil {
		return err
	}

	err = createResourcesForWM(ctx, wm)
	if err != nil {
		return err
	}

	err = createExtGStateForStamp(xRefTable, wm)
	if err != nil {
		return err
	}

	for k, v := range selectedPages {
		if v {
			err := watermarkPage(xRefTable, k, wm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

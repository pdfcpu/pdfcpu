package pdfcpu

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/fonts/metrics"
	"github.com/hhrutter/pdfcpu/pkg/types"

	"github.com/pkg/errors"
)

const (
	degToRad = math.Pi / 180
	radToDeg = 180 / math.Pi
)

type matrix [3][3]float64

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

type formCache map[types.Rectangle]*PDFIndirectRef

// Watermark represents the basic structure and command details for the commands "Stamp" and "Watermark".
type Watermark struct {

	// configuration
	text          string      // display text
	imageFileName string      // display png image
	onTop         bool        // if true this is a STAMP else this is a WATERMARK.
	fontName      string      // supported are Adobe base fonts only. (as of now: Helvetica, Times-Roman, Courier)
	fontSize      int         // font scaling factor.
	color         simpleColor // fill color(=non stroking color).
	rotation      float64     // rotation to apply in degrees. -180 <= x <= 180
	diagonal      int         // paint along the diagonal.
	opacity       float64     // opacity the displayed text. 0 <= x <= 1
	renderMode    int         // fill=0, stroke=1 fill&stroke=2
	scale         float64     // relative scale factor. 0 <= x <= 1
	scaleAbs      bool        // true for absolute scaling

	// resources
	ocg, extGState, font, image *PDFIndirectRef
	imgWidth, imgHeight         int

	// page specific
	bb      types.Rectangle // bounding box of the form representing this watermark.
	vp      types.Rectangle // page dimensions for text alignment.
	pageRot float64         // page rotation in effect.
	form    *PDFIndirectRef // Forms are dependent on given page dimensions.

	// house keeping
	objs   IntSet    // objects for which wm has been applied already.
	fCache formCache // form cache.
}

func (wm Watermark) String() string {
	var s string
	if !wm.onTop {
		s = "not "
	}
	t := wm.text
	if len(t) == 0 {
		t = wm.imageFileName
	}
	sc := "relative"
	if wm.scaleAbs {
		sc = "absolute"
	}
	return fmt.Sprintf("Watermark: <%s> is %son top\n"+
		"%s %d points\n"+
		"scaling: %f %s\n"+
		"color: %s\n"+
		"rotation: %f\n"+
		"diagonal: %d\n"+
		"opacity: %f\n"+
		"renderMode: %d\n"+
		"bbox:%s\n"+
		"vp:%s\n"+
		"pageRotation: %f\n",
		t, s,
		wm.fontName, wm.fontSize,
		wm.scale, sc,
		wm.color,
		wm.rotation,
		wm.diagonal,
		wm.opacity,
		wm.renderMode,
		wm.bb,
		wm.vp,
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

// IsImage returns whether the watermark content is an image or text.
func (wm Watermark) IsImage() bool {
	return len(wm.imageFileName) > 0
}

func (wm *Watermark) calcBoundingBox() {

	//fmt.Println("calcBoundingBox:")

	var bb types.Rectangle

	if wm.IsImage() {
		// image watermark
		bb = types.NewRectangle(0, 0, float64(wm.imgWidth), float64(wm.imgHeight))
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
		wm.fontSize = int(float64(wm.fontSize) * wm.scale)
		w = metrics.TextWidth(wm.text, wm.fontName, wm.fontSize)
	} else {
		w = wm.scale * wm.vp.Width()
		wm.fontSize = metrics.FontSize(wm.text, wm.fontName, w)
	}
	bb = types.NewRectangle(0, -float64(wm.fontSize), w, float64(wm.fontSize)/10)

	wm.bb = bb
	return
}

func (wm *Watermark) calcTransformMatrix() *matrix {

	var sin, cos float64
	r := wm.rotation

	if wm.diagonal != noDiagonal {
		// Calculate the angle of the diagonal.
		r = math.Atan(wm.vp.Height()/wm.vp.Width()) * float64(radToDeg)
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
	if !wm.IsImage() {
		dy = wm.bb.LL.Y
	}

	m2[2][0] = wm.vp.Width()/2 + sin*(wm.bb.Height()/2+dy) - cos*wm.bb.Width()/2
	m2[2][1] = wm.vp.Height()/2 - cos*(wm.bb.Height()/2+dy) - sin*wm.bb.Width()/2

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

func setWatermarkType(s string, wm *Watermark) {
	ext := filepath.Ext(s)
	if ext == ".png" || ext == ".tif" || ext == ".tiff" {
		wm.imageFileName = s
	} else {
		wm.text = s
	}
}

func supportedWatermarkFont(fn string) bool {
	for _, s := range metrics.FontNames() {
		if fn == s {
			return true
		}
	}
	return false
}

func parseWatermarkFontSize(v string, wm *Watermark) error {

	fs, err := strconv.Atoi(v)
	if err != nil {
		return err
	}

	wm.fontSize = fs

	return nil
}

func parseWatermarkScaleFactor(v string, wm *Watermark) error {

	sc := strings.Split(v, " ")
	if len(sc) > 2 {
		return errors.Errorf("illegal scale string: 0.0 <= i <= 1.0 {abs|rel}, %s\n", v)
	}

	s, err := strconv.ParseFloat(sc[0], 64)
	if err != nil {
		return errors.Errorf("scale factor must be a float value: %s\n", v)
	}

	if s < 0 || s > 1 {
		return errors.Errorf("illegal scale factor: 0.0 <= s <= 1.0, %s\n", v)
	}

	wm.scale = s

	if len(sc) == 2 {
		switch sc[1] {
		case "a", "abs":
			wm.scaleAbs = true

		case "r", "rel":
			wm.scaleAbs = false

		default:
			return errors.Errorf("illegal scale mode: abs|rel, %s\n", v)
		}
	}

	return nil
}

func parseWatermarkColor(v string, wm *Watermark) error {

	cs := strings.Split(v, " ")
	if len(cs) != 3 {
		return errors.Errorf("illegal color string: 3 intensities 0.0 <= i <= 1.0, %s\n", v)
	}

	r, err := strconv.ParseFloat(cs[0], 32)
	if err != nil {
		return errors.Errorf("red must be a float value: %s\n", v)
	}
	if r < 0 || r > 1 {
		return errors.New("a color value is an intensity between 0.0 and 1.0")
	}
	wm.color.r = float32(r)

	g, err := strconv.ParseFloat(cs[1], 32)
	if err != nil {
		return errors.Errorf("green must be a float value: %s\n", v)
	}
	if g < 0 || g > 1 {
		return errors.New("a color value is an intensity between 0.0 and 1.0")
	}
	wm.color.g = float32(g)

	b, err := strconv.ParseFloat(cs[2], 32)
	if err != nil {
		return errors.Errorf("blue must be a float value: %s\n", v)
	}
	if b < 0 || b > 1 {
		return errors.New("a color value is an intensity between 0.0 and 1.0")
	}
	wm.color.b = float32(b)

	return nil
}

func parseWatermarkRotation(v string, setDiag bool, wm *Watermark) error {

	if setDiag {
		return errors.New("Please specify rotation or diagonal (r or d)")
	}

	r, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return errors.Errorf("rotation must be a float value: %s\n", v)
	}
	if r < -180 || r > 180 {
		return errors.Errorf("illegal rotation: -180 <= r <= 180 degrees, %s\n", v)
	}

	wm.rotation = r
	wm.diagonal = noDiagonal

	return nil
}

func parseWatermarkDiagonal(v string, setRot bool, wm *Watermark) error {

	if setRot {
		return errors.New("Please specify rotation or diagonal (r or d)")
	}

	d, err := strconv.Atoi(v)
	if err != nil {
		return errors.Errorf("illegal diagonal value: allowed 1 or 2, %s\n", v)
	}
	if d != diagonalLLToUR && d != diagonalULToLR {
		return errors.New("diagonal: 1..lower left to upper right, 2..upper left to lower right")
	}

	wm.diagonal = d
	wm.rotation = 0

	return nil
}

func parseWatermarkOpacity(v string, wm *Watermark) error {

	o, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return errors.Errorf("opacity must be a float value: %s\n", v)
	}
	if o < 0 || o > 1 {
		return errors.Errorf("illegal opacity: 0.0 <= r <= 1.0, %s\n", v)
	}
	wm.opacity = o

	return nil
}

func parseWatermarkRenderMode(v string, wm *Watermark) error {

	m, err := strconv.Atoi(v)
	if err != nil {
		return errors.Errorf("illegal mode value: allowed 0,1,2, %s\n", v)
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
	wm := &Watermark{
		onTop:      onTop,
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
	}

	ss := strings.Split(s, ",")

	setWatermarkType(ss[0], wm)

	if len(ss) == 1 {
		return wm, nil
	}

	var setDiag, setRot bool

	for _, s := range ss[1:] {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, parseWatermarkError(onTop)
		}

		k := strings.TrimSpace(ss1[0])
		v := strings.TrimSpace(ss1[1])
		//fmt.Printf("key:<%s> value<%s>\n", k, v)

		var err error

		switch k {
		case "f": // font name
			if !supportedWatermarkFont(v) {
				err = errors.Errorf("%s is unsupported, try one of Helvetica, Times-Roman, Courier.\n", v)
			}
			wm.fontName = v

		case "p": // font size in points
			err = parseWatermarkFontSize(v, wm)

		case "s": // scale factor
			err = parseWatermarkScaleFactor(v, wm)

		case "c": // color
			err = parseWatermarkColor(v, wm)

		case "r": // rotation
			err = parseWatermarkRotation(v, setDiag, wm)
			setRot = true

		case "d": // diagonal
			err = parseWatermarkDiagonal(v, setRot, wm)
			setDiag = true

		case "o": // opacity
			err = parseWatermarkOpacity(v, wm)

		case "m": // render mode
			err = parseWatermarkRenderMode(v, wm)

		default:
			err = parseWatermarkError(onTop)
		}

		if err != nil {
			return nil, err
		}
	}

	return wm, nil
}

func createFontResForWM(xRefTable *XRefTable, wm *Watermark) error {

	d := NewPDFDict()
	d.InsertName("Type", "Font")
	d.InsertName("Subtype", "Type1")
	d.InsertName("BaseFont", wm.fontName)

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	wm.font = indRef

	return nil
}

func createImageResForWM(xRefTable *XRefTable, wm *Watermark) error {

	f := ReadTIFFFile
	if filepath.Ext(wm.imageFileName) == ".png" {
		f = ReadPNGFile
	}

	sd, err := f(xRefTable, wm.imageFileName)
	if err != nil {
		return err
	}
	//fmt.Println("image loaded!")

	wm.imgWidth = *sd.IntEntry("Width")
	wm.imgHeight = *sd.IntEntry("Height")
	//fmt.Printf("w:%d h%d\n", wm.imgWidth, wm.imgHeight)

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	wm.image = indRef

	return nil
}

func createResourcesForWM(xRefTable *XRefTable, wm *Watermark) error {

	if wm.IsImage() {
		return createImageResForWM(xRefTable, wm)
	}

	return createFontResForWM(xRefTable, wm)
}

// AddWatermarks adds watermarks to all pages selected.
func AddWatermarks(xRefTable *XRefTable, selectedPages IntSet, wm *Watermark) error {

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

	err = createResourcesForWM(xRefTable, wm)
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

func createOCG(xRefTable *XRefTable, wm *Watermark) error {

	name := "Background"
	subt := "BG"
	if wm.onTop {
		name = "Watermark"
		subt = "FG"
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Name": PDFStringLiteral(name),
			"Type": PDFName("OCG"),
			"Usage": PDFDict{
				Dict: map[string]PDFObject{
					"PageElement": PDFDict{Dict: map[string]PDFObject{"Subtype": PDFName(subt)}},
					"View":        PDFDict{Dict: map[string]PDFObject{"ViewState": PDFName("ON")}},
					"Print":       PDFDict{Dict: map[string]PDFObject{"PrintState": PDFName("ON")}},
					"Export":      PDFDict{Dict: map[string]PDFObject{"ExportState": PDFName("ON")}},
				},
			},
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	wm.ocg = indRef

	return nil
}

func prepareOCPropertiesInRoot(rootDict *PDFDict, wm *Watermark) error {

	optionalContentConfigDict := PDFDict{
		Dict: map[string]PDFObject{
			"AS": PDFArray{
				PDFDict{
					Dict: map[string]PDFObject{
						"Category": NewNameArray("View"),
						"Event":    PDFName("View"),
						"OCGs":     PDFArray{*wm.ocg},
					},
				},
				PDFDict{
					Dict: map[string]PDFObject{
						"Category": NewNameArray("Print"),
						"Event":    PDFName("Print"),
						"OCGs":     PDFArray{*wm.ocg},
					},
				},
				PDFDict{
					Dict: map[string]PDFObject{
						"Category": NewNameArray("Export"),
						"Event":    PDFName("Export"),
						"OCGs":     PDFArray{*wm.ocg},
					},
				},
			},
			"ON":       PDFArray{*wm.ocg},
			"Order":    PDFArray{},
			"RBGroups": PDFArray{},
		},
	}

	d := PDFDict{
		Dict: map[string]PDFObject{
			"OCGs": PDFArray{*wm.ocg},
			"D":    optionalContentConfigDict,
		},
	}

	_, ok := rootDict.Find("OCProperties")
	if !ok {
		rootDict.Insert("OCProperties", d)
		return nil
	}

	return oneWatermarkOnlyError(wm.onTop)
}

func createFormResDict(xRefTable *XRefTable, wm *Watermark) *PDFDict {

	if wm.IsImage() {
		return &PDFDict{
			Dict: map[string]PDFObject{
				"ProcSet": NewNameArray("PDF", "ImageC"),
				"XObject": PDFDict{Dict: map[string]PDFObject{"Im0": *wm.image}},
			}}
	}

	return &PDFDict{
		Dict: map[string]PDFObject{
			"Font":    PDFDict{Dict: map[string]PDFObject{wm.fontName: *wm.font}},
			"ProcSet": NewNameArray("PDF", "Text"),
		}}
}

func createForm(xRefTable *XRefTable, wm *Watermark, withBB bool) error {

	wm.calcBoundingBox()
	bb := wm.bb

	// The forms bounding box is dependent on the page dimensions.

	indRef, ok := wm.fCache[wm.bb]
	if ok {
		//fmt.Printf("reusing form obj#%d\n", indRef.ObjectNumber)
		wm.form = indRef
		return nil
	}

	var b bytes.Buffer

	if wm.IsImage() {
		fmt.Fprintf(&b, "q %f 0 0 %f 0 0 cm /Im0 Do Q", bb.Width(), bb.Height())
	} else {
		// 12 font points result in a vertical displacement of 9.47
		dy := -float64(wm.fontSize) / 12 * 9.47
		wmForm := "0 g 0 G 0 i 0 J []0 d 0 j 1 w 10 M 0 Tc 0 Tw 100 Tz 0 TL %d Tr 0 Ts BT /%s %d Tf %f %f %f rg 0 %f Td (%s)Tj ET"
		fmt.Fprintf(&b, wmForm, wm.renderMode, wm.fontName, wm.fontSize, wm.color.r, wm.color.g, wm.color.b, dy, wm.text)
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

	sd := &PDFStreamDict{
		PDFDict: PDFDict{
			Dict: map[string]PDFObject{
				"Type":      PDFName("XObject"),
				"Subtype":   PDFName("Form"),
				"BBox":      NewRectangle(bb.LL.X, bb.LL.Y, bb.UR.X, bb.UR.Y),
				"Matrix":    NewIntegerArray(1, 0, 0, 1, 0, 0),
				"OC":        *wm.ocg,
				"Resources": *createFormResDict(xRefTable, wm),
			},
		},
		Content: b.Bytes(),
	}

	err := encodeStream(sd)
	if err != nil {
		return err
	}

	indRef, err = xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	//fmt.Printf("caching form obj#%d\n", indRef.ObjectNumber)
	wm.fCache[wm.bb] = indRef

	wm.form = indRef

	return nil
}

func createExtGStateForStamp(xRefTable *XRefTable, wm *Watermark) error {

	d := PDFDict{
		Dict: map[string]PDFObject{
			"Type": PDFName("ExtGState"),
			"CA":   PDFFloat(wm.opacity),
			"ca":   PDFFloat(wm.opacity),
		},
	}

	indRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	wm.extGState = indRef

	return nil
}

func rect(xRefTable *XRefTable, rect PDFArray) types.Rectangle {
	llx := xRefTable.DereferenceNumber(rect[0])
	lly := xRefTable.DereferenceNumber(rect[1])
	urx := xRefTable.DereferenceNumber(rect[2])
	ury := xRefTable.DereferenceNumber(rect[3])
	return types.NewRectangle(llx, lly, urx, ury)
}

func insertPageResourcesForWM(xRefTable *XRefTable, pageDict *PDFDict, wm *Watermark, gsID, xoID string) error {

	resourceDict := PDFDict{
		Dict: map[string]PDFObject{
			"ExtGState": PDFDict{Dict: map[string]PDFObject{gsID: *wm.extGState}},
			"XObject":   PDFDict{Dict: map[string]PDFObject{xoID: *wm.form}},
		},
	}

	pageDict.Insert("Resources", resourceDict)

	return nil
}

func updatePageResourcesForWM(xRefTable *XRefTable, resDict *PDFDict, wm *Watermark, gsID, xoID *string) error {

	o, ok := resDict.Find("ExtGState")
	if !ok {
		resDict.Insert("ExtGState", PDFDict{Dict: map[string]PDFObject{*gsID: *wm.extGState}})
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
	//println("extGState done")

	o, ok = resDict.Find("XObject")
	if !ok {
		resDict.Insert("XObject", PDFDict{Dict: map[string]PDFObject{*xoID: *wm.form}})
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
	//println("xObject done")

	return nil
}

func wmContent(wm *Watermark, gsID, xoID string) []byte {

	m := wm.calcTransformMatrix()

	insertOCG := " /Artifact <</Subtype /Watermark /Type /Pagination >>BDC q %f %f %f %f %f %f cm /%s gs /%s Do Q EMC "

	var b bytes.Buffer
	fmt.Fprintf(&b, insertOCG, m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1], gsID, xoID)

	return b.Bytes()
}

func insertPageContentsForWM(xRefTable *XRefTable, pageDict *PDFDict, wm *Watermark, gsID, xoID string) error {

	sd := &PDFStreamDict{PDFDict: NewPDFDict()}

	sd.Content = wmContent(wm, gsID, xoID)

	err := encodeStream(sd)
	if err != nil {
		return err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *indRef)

	return nil
}

func updatePageContentsForWM(xRefTable *XRefTable, obj PDFObject, wm *Watermark, gsID, xoID string) error {

	var entry *XRefTableEntry
	var objNr int

	indRef, ok := obj.(PDFIndirectRef)
	if ok {
		objNr = indRef.ObjectNumber.Value()
		if wm.objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}
		generationNumber := indRef.GenerationNumber.Value()
		entry, _ = xRefTable.FindTableEntry(objNr, generationNumber)
		obj = entry.Object
	}

	switch o := obj.(type) {

	case PDFStreamDict:

		//fmt.Printf("%T %T\n", &o, o)
		//fmt.Printf("Content obj#%d addr:%v\n%s\n", objNr, &o, o)

		err := patchContentForWM(&o, gsID, xoID, wm)
		if err != nil {
			return err
		}

		//fmt.Printf("content after patch:\n%s\n", o)

		entry.Object = o
		wm.objs[objNr] = true

	case PDFArray:

		var o1 PDFObject
		if wm.onTop {
			o1 = o[len(o)-1] // patch last content stream
		} else {
			o1 = o[0] // patch first content stream
		}

		indRef, _ := o1.(PDFIndirectRef)
		objNr = indRef.ObjectNumber.Value()
		if wm.objs[objNr] {
			// wm already applied to this content stream.
			return nil
		}

		generationNumber := indRef.GenerationNumber.Value()
		entry, _ := xRefTable.FindTableEntry(objNr, generationNumber)
		sd, _ := (entry.Object).(PDFStreamDict)
		err := patchContentForWM(&sd, gsID, xoID, wm)
		if err != nil {
			return err
		}

		entry.Object = sd
		wm.objs[objNr] = true
	}

	return nil
}

func watermarkPage(xRefTable *XRefTable, i int, wm *Watermark) error {

	fmt.Printf("watermark page %d\n", i)

	d, inhPAttrs, err := xRefTable.PageDict(i)
	if err != nil {
		return err
	}

	visibleRegion := inhPAttrs.mediaBox
	if inhPAttrs.cropBox != nil {
		visibleRegion = inhPAttrs.cropBox
	}
	vp := rect(xRefTable, *visibleRegion)
	if err != nil {
		return err
	}
	//fmt.Printf("vp = %f %f %f %f\n", vp.Llx, vp.Lly, vp.Urx, vp.Ury)
	wm.vp = vp

	err = createForm(xRefTable, wm, true)
	if err != nil {
		return err
	}

	//fmt.Println(wm)

	wm.pageRot = inhPAttrs.rotate
	// wm.pageRot = 0
	// if inhPAttrs.rotate != nil && *rotate != 0 {
	// 	wm.pageRot = *rotate
	// }

	// if resources != nil {
	// 	fmt.Printf("resourceDict: %s\n", *resources)
	// }
	// fmt.Printf("%s\n", *d)

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

func patchContentForWM(sd *PDFStreamDict, gsID, xoID string, wm *Watermark) error {

	// Decode streamDict for supported filters only.
	err := decodeStream(sd)
	if err == filter.ErrUnsupportedFilter {
		fmt.Println("unsupported filter")
		return nil
	}
	if err != nil {
		return err
	}

	bb := wmContent(wm, gsID, xoID)

	if wm.onTop {
		sd.Content = append(sd.Content, bb...)
	} else {
		sd.Content = append(bb, sd.Content...)
	}
	//fmt.Printf("patched content:\n%s\n", hex.Dump(sd.Content))

	// Manipulate sd.Content
	err = encodeStream(sd)
	if err != nil {
		return err
	}

	return nil

}

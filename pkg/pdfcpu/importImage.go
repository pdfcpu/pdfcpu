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
	"io"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/hhrutter/pdfcpu/pkg/types"
	"github.com/pkg/errors"
)

var errInvalidImportConfig = errors.New("Invalid import configuration string. Please consult pdfcpu help import")

// Import represents the command details for the command "ImportImage".
type Import struct {
	pageDim  dim     // page dimensions in user units.
	pageSize string  // one of A0,A1,A2,A3,A4(=default),A5,A6,A7,A8,Letter,Legal,Ledger,Tabloid,Executive,ANSIC,ANSID,ANSIE.
	pos      anchor  // position anchor, one of tl,tc,tr,l,c,r,bl,bc,br,full.
	dx, dy   int     // anchor offset.
	scale    float64 // relative scale factor. 0 <= x <= 1
	scaleAbs bool    // true for absolute scaling.
}

func (imp Import) String() string {

	sc := "relative"
	if imp.scaleAbs {
		sc = "absolute"
	}

	return fmt.Sprintf("Import config: %s %s, pos=%s, dx=%d, dy=%d, scaling: %.1f %s\n",
		imp.pageSize, imp.pageDim, imp.pos, imp.dx, imp.dy, imp.scale, sc)
}

func parsePageFormat(v string, setDim bool) (dim, string, error) {

	var d dim

	if setDim {
		return d, v, errors.New("Only one of format('f') or dimensions('d') allowed")
	}

	d, ok := PaperSize[v]
	if !ok {
		return d, v, errors.Errorf("Page format %s is unsupported.\n", v)
	}

	return d, v, nil
}

func parsePageDim(v string, setFormat bool) (dim, string, error) {

	var d dim

	if setFormat {
		return d, v, errors.New("Only one of format('f') or dimensions('d') allowed")
	}

	ss := strings.Split(v, " ")
	if len(ss) != 2 {
		return d, v, errors.Errorf("illegal dimension string: need 2 positive values, %s\n", v)
	}

	w, err := strconv.Atoi(ss[0])
	if err != nil || w <= 0 {
		return d, v, errors.Errorf("dimension X must be a positiv numeric value: %s\n", ss[0])
	}

	h, err := strconv.Atoi(ss[1])
	if err != nil || h <= 0 {
		return d, v, errors.Errorf("dimension Y must be a positiv numeric value: %s\n", ss[1])
	}

	d = dim{w, h}

	return d, "", nil
}

type anchor int

func (a anchor) String() string {

	switch a {

	case TopLeft:
		return "top left"

	case TopCenter:
		return "top center"

	case TopRight:
		return "top right"

	case Left:
		return "left"

	case Center:
		return "center"

	case Right:
		return "right"

	case BottomLeft:
		return "bottom left"

	case BottomCenter:
		return "bottom center"

	case BottomRight:
		return "bottom right"

	case Full:
		return "full"

	}

	return ""
}

// These are the defined anchors for relative positioning.
const (
	TopLeft anchor = iota
	TopCenter
	TopRight
	Left
	Center // default
	Right
	BottomLeft
	BottomCenter
	BottomRight
	Full // special case, no anchor needed, imageSize = pageSize
)

func parseAnchorPosition(s string) (anchor, error) {

	switch s {
	case "tl":
		return TopLeft, nil
	case "tc":
		return TopCenter, nil
	case "tr":
		return TopRight, nil
	case "l":
		return Left, nil
	case "c":
		return Center, nil
	case "r":
		return Right, nil
	case "bl":
		return BottomLeft, nil
	case "bc":
		return BottomCenter, nil
	case "br":
		return BottomRight, nil
	case "full":
		return Full, nil
	}

	return 0, errors.Errorf("unknown position anchor: %s", s)
}

func parsePositionOffset(s string) (int, int, error) {

	d := strings.Split(s, " ")
	if len(d) != 2 {
		return 0, 0, errors.Errorf("illegal position offset string: need 2 numeric values, %s\n", s)
	}

	dx, err := strconv.Atoi(d[0])
	if err != nil {
		return 0, 0, err
	}

	dy, err := strconv.Atoi(d[1])
	if err != nil {
		return 0, 0, err
	}

	return dx, dy, nil
}

// DefaultImportConfig returns the default configuration.
func DefaultImportConfig() *Import {
	return &Import{
		pageDim:  PaperSize["A4"],
		pageSize: "A4",
		pos:      Full,
		scale:    0.5,
	}
}

// ParseImportDetails parses a ImportImage command string into an internal structure.
func ParseImportDetails(s string) (*Import, error) {

	//fmt.Printf("impdetails: <%s>\n", s)

	if s == "" {
		return nil, nil
	}

	imp := DefaultImportConfig()

	ss := strings.Split(s, ",")

	var setDim, setFormat bool

	for _, s := range ss {

		ss1 := strings.Split(s, ":")
		if len(ss1) != 2 {
			return nil, errInvalidImportConfig
		}

		k := strings.TrimSpace(ss1[0])
		v := strings.TrimSpace(ss1[1])
		//fmt.Printf("key:<%s> value<%s>\n", k, v)

		var err error

		switch k {
		case "f": // page format
			imp.pageDim, imp.pageSize, err = parsePageFormat(v, setDim)
			setFormat = true

		case "d": // page dimensions
			imp.pageDim, imp.pageSize, err = parsePageDim(v, setFormat)
			setDim = true

		case "p": // position
			imp.pos, err = parseAnchorPosition(v)

		case "o": // offset
			imp.dx, imp.dy, err = parsePositionOffset(v)

		case "s": // scale factor
			imp.scale, imp.scaleAbs, err = parseScaleFactor(v)

		default:
			err = errInvalidImportConfig
		}

		if err != nil {
			return nil, err
		}
	}

	return imp, nil
}

// AppendPageTree appends a pagetree d1 to page tree d2.
func AppendPageTree(d1 *IndirectRef, countd1 int, d2 *Dict) error {

	a := d2.ArrayEntry("Kids")
	log.Debug.Printf("Kids before: %v\n", a)

	a = append(a, *d1)
	log.Debug.Printf("Kids after: %v\n", a)

	d2.Update("Kids", a)

	err := d2.IncrementBy("Count", countd1)
	if err != nil {
		return err
	}

	return nil
}

func lowerLeftCorner(vpw, vph, bbw, bbh float64, a anchor) types.Point {

	var p types.Point

	switch a {

	case TopLeft:
		p.X = 0
		p.Y = vph - bbh

	case TopCenter:
		p.X = vpw/2 - bbw/2
		p.Y = vph - bbh

	case TopRight:
		p.X = vpw - bbw
		p.Y = vph - bbh

	case Left:
		p.X = 0
		p.Y = vph/2 - bbh/2

	case Center:
		p.X = vpw/2 - bbw/2
		p.Y = vph/2 - bbh/2

	case Right:
		p.X = vpw - bbw
		p.Y = vph/2 - bbh/2

	case BottomLeft:
		p.X = 0
		p.Y = 0

	case BottomCenter:
		p.X = vpw/2 - bbw/2
		p.Y = 0

	case BottomRight:
		p.X = vpw - bbw
		p.Y = 0
	}

	return p
}

func importImagePDFBytes(wr io.Writer, pageDim dim, imgWidth, imgHeight float64, imp *Import) {

	vpw := float64(pageDim.w)
	vph := float64(pageDim.h)

	if imp.pos == Full {
		// The bounding box equals the page dimensions.
		bb := types.NewRectangle(0, 0, vpw, vph)
		bb.UR.X = bb.Width()
		bb.UR.Y = bb.UR.X / bb.AspectRatio()
		fmt.Fprintf(wr, "q %f 0 0 %f 0 0 cm /Im0 Do Q", bb.Width(), bb.Height())
		return
	}

	bb := types.NewRectangle(0, 0, imgWidth, imgHeight)
	ar := bb.AspectRatio()

	if imp.scaleAbs {
		bb.UR.X = imp.scale * bb.Width()
		bb.UR.Y = bb.UR.X / ar
	} else {
		if ar >= 1 {
			bb.UR.X = imp.scale * vpw
			bb.UR.Y = bb.UR.X / ar
		} else {
			bb.UR.Y = imp.scale * vph
			bb.UR.X = bb.UR.Y * ar
		}
	}

	//fmt.Printf("vp.w=%f vp.h=%f bb.w=%f bb.h=%f\n", vpw, vph, bb.Width(), bb.Height())

	m := identMatrix

	// Scale
	m[0][0] = bb.Width()
	m[1][1] = bb.Height()

	// Translate
	ll := lowerLeftCorner(vpw, vph, bb.Width(), bb.Height(), imp.pos)
	m[2][0] = ll.X + float64(imp.dx)
	m[2][1] = ll.Y + float64(imp.dy)

	fmt.Fprintf(wr, "q %.2f %.2f %.2f %.2f %.2f %.2f cm /Im0 Do Q",
		m[0][0], m[0][1], m[1][0], m[1][1], m[2][0], m[2][1])
}

// NewPageForImage creates a new page dict in xRefTable for given image filename.
func NewPageForImage(xRefTable *XRefTable, fileName string, parentIndRef *IndirectRef, imp *Import) (*IndirectRef, error) {

	// create image dict.
	imgIndRef, w, h, err := createImageResource(xRefTable, fileName)
	if err != nil {
		return nil, err
	}

	// create resource dict for XObject.
	d := Dict(
		map[string]Object{
			"ProcSet": NewNameArray("PDF", "ImageB", "ImageC", "ImageI"),
			"XObject": Dict(map[string]Object{"Im0": *imgIndRef}),
		},
	)

	resIndRef, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return nil, err
	}

	// create content stream for Im0.
	contents := &StreamDict{Dict: NewDict()}
	contents.InsertName("Filter", filter.Flate)
	contents.FilterPipeline = []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}

	dim := dim{w, h}
	if imp.pos != Full {
		dim = imp.pageDim
	}
	// mediabox = physical page dimensions
	mediaBox := NewRectangle(0, 0, float64(dim.w), float64(dim.h))

	var buf bytes.Buffer
	importImagePDFBytes(&buf, dim, float64(w), float64(h), imp)
	contents.Content = buf.Bytes()

	err = encodeStream(contents)
	if err != nil {
		return nil, err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*contents)
	if err != nil {
		return nil, err
	}

	pageDict := Dict(
		map[string]Object{
			"Type":      Name("Page"),
			"Parent":    *parentIndRef,
			"MediaBox":  mediaBox,
			"Resources": *resIndRef,
			"Contents":  *contentsIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(pageDict)
}

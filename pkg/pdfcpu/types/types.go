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

package types

import (
	"encoding/hex"
	"fmt"
	"strconv"
)

// Supported line delimiters
const (
	EolLF   = "\x0A"
	EolCR   = "\x0D"
	EolCRLF = "\x0D\x0A"
)

// FreeHeadGeneration is the predefined generation number for the head of the free list.
const FreeHeadGeneration = 65535

// ByteSize represents the various terms for storage space.
type ByteSize float64

// Storage space terms.
const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
)

func (b ByteSize) String() string {

	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.1f MB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.0f KB", b/KB)
	}

	return fmt.Sprintf("%.0f", b)
}

// IntSet is a set of integers.
type IntSet map[int]bool

// StringSet is a set of strings.
type StringSet map[string]bool

// Object defines an interface for all Objects.
type Object interface {
	fmt.Stringer
	Clone() Object
	PDFString() string
}

// Boolean represents a PDF boolean object.
type Boolean bool

// Clone returns a clone of boolean.
func (boolean Boolean) Clone() Object {
	return boolean
}

func (boolean Boolean) String() string {
	return fmt.Sprintf("%v", bool(boolean))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (boolean Boolean) PDFString() string {
	return boolean.String()
}

// Value returns a bool value for this PDF object.
func (boolean Boolean) Value() bool {
	return bool(boolean)
}

///////////////////////////////////////////////////////////////////////////////////

// Float represents a PDF float object.
type Float float64

// Clone returns a clone of f.
func (f Float) Clone() Object {
	return f
}

func (f Float) String() string {
	// Use a precision of 2 for logging readability.
	return fmt.Sprintf("%.2f", float64(f))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (f Float) PDFString() string {
	// The max precision encountered so far has been 12 (fontType3 fontmatrix components).
	return strconv.FormatFloat(f.Value(), 'f', 12, 64)
}

// Value returns a float64 value for this PDF object.
func (f Float) Value() float64 {
	return float64(f)
}

///////////////////////////////////////////////////////////////////////////////////

// Integer represents a PDF integer object.
type Integer int

// Clone returns a clone of i.
func (i Integer) Clone() Object {
	return i
}

func (i Integer) String() string {
	return strconv.Itoa(int(i))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (i Integer) PDFString() string {
	return i.String()
}

// Value returns an int value for this PDF object.
func (i Integer) Value() int {
	return int(i)
}

///////////////////////////////////////////////////////////////////////////////////

// Point represents a user space location.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func NewPoint(x, y float64) Point {
	return Point{X: x, Y: y}
}

// Translate modifies p's coordinates.
func (p *Point) Translate(dx, dy float64) {
	p.X += dx
	p.Y += dy
}

func (p Point) String() string {
	return fmt.Sprintf("(%.2f,%.2f)\n", p.X, p.Y)
}

// Rectangle represents a rectangular region in userspace.
type Rectangle struct {
	LL Point `json:"ll"`
	UR Point `json:"ur"`
}

// NewRectangle returns a new rectangle for given corner coordinates.
func NewRectangle(llx, lly, urx, ury float64) *Rectangle {
	return &Rectangle{LL: Point{llx, lly}, UR: Point{urx, ury}}
}

func decodeFloat(number Object) float64 {
	var f float64
	switch v := number.(type) {
	case Float:
		f = v.Value()
	case Integer:
		f = float64(v.Value())
	}
	return f
}

func RectForArray(arr Array) *Rectangle {
	if len(arr) != 4 {
		return nil
	}

	llx := decodeFloat(arr[0])
	lly := decodeFloat(arr[1])
	urx := decodeFloat(arr[2])
	ury := decodeFloat(arr[3])

	return NewRectangle(llx, lly, urx, ury)
}

// RectForDim returns a new rectangle for given dimensions.
func RectForDim(width, height float64) *Rectangle {
	return NewRectangle(0.0, 0.0, width, height)
}

// RectForWidthAndHeight returns a new rectangle for given dimensions.
func RectForWidthAndHeight(llx, lly, width, height float64) *Rectangle {
	return NewRectangle(llx, lly, llx+width, lly+height)
}

// RectForFormat returns a new rectangle for given format.
func RectForFormat(f string) *Rectangle {
	d := PaperSize[f]
	return RectForDim(d.Width, d.Height)
}

// Width returns the horizontal span of a rectangle in userspace.
func (r Rectangle) Width() float64 {
	return r.UR.X - r.LL.X
}

// Height returns the vertical span of a rectangle in userspace.
func (r Rectangle) Height() float64 {
	return r.UR.Y - r.LL.Y
}

func (r Rectangle) Equals(r2 Rectangle) bool {
	return r.LL == r2.LL && r.UR == r2.UR
}

// FitsWithin returns true if rectangle r fits within rectangle r2.
func (r Rectangle) FitsWithin(r2 *Rectangle) bool {
	return r.Width() <= r2.Width() && r.Height() <= r2.Height()
}

func (r Rectangle) Visible() bool {
	return r.Width() != 0 && r.Height() != 0
}

// AspectRatio returns the relation between width and height of a rectangle.
func (r Rectangle) AspectRatio() float64 {
	return r.Width() / r.Height()
}

// Landscape returns true if r is in landscape mode.
func (r Rectangle) Landscape() bool {
	return r.AspectRatio() > 1
}

// Portrait returns true if r is in portrait mode.
func (r Rectangle) Portrait() bool {
	return r.AspectRatio() < 1
}

// Contains returns true if rectangle r contains point p.
func (r Rectangle) Contains(p Point) bool {
	return p.X >= r.LL.X && p.X <= r.UR.X && p.Y >= r.LL.Y && p.Y <= r.LL.Y
}

// ScaledWidth returns the width for given height according to r's aspect ratio.
func (r Rectangle) ScaledWidth(h float64) float64 {
	return r.AspectRatio() * h
}

// ScaledHeight returns the height for given width according to r's aspect ratio.
func (r Rectangle) ScaledHeight(w float64) float64 {
	return w / r.AspectRatio()
}

// Dimensions returns r's dimensions.
func (r Rectangle) Dimensions() Dim {
	return Dim{r.Width(), r.Height()}
}

// Translate moves r by dx and dy.
func (r *Rectangle) Translate(dx, dy float64) {
	r.LL.Translate(dx, dy)
	r.UR.Translate(dx, dy)
}

// Center returns the center point of a rectangle.
func (r Rectangle) Center() Point {
	return Point{(r.UR.X - r.Width()/2), (r.UR.Y - r.Height()/2)}
}

func (r Rectangle) String() string {
	return fmt.Sprintf("(%3.2f, %3.2f, %3.2f, %3.2f) w=%.2f h=%.2f ar=%.2f", r.LL.X, r.LL.Y, r.UR.X, r.UR.Y, r.Width(), r.Height(), r.AspectRatio())
}

// ShortString returns a compact string representation for r.
func (r Rectangle) ShortString() string {
	return fmt.Sprintf("(%3.0f, %3.0f, %3.0f, %3.0f)", r.LL.X, r.LL.Y, r.UR.X, r.UR.Y)
}

// Array returns the PDF representation of a rectangle.
func (r Rectangle) Array() Array {
	return NewNumberArray(r.LL.X, r.LL.Y, r.UR.X, r.UR.Y)
}

// Clone returns a clone of r.
func (r Rectangle) Clone() *Rectangle {
	return NewRectangle(r.LL.X, r.LL.Y, r.UR.X, r.UR.Y)
}

// CroppedCopy returns a copy of r with applied margin..
func (r Rectangle) CroppedCopy(margin float64) *Rectangle {
	return NewRectangle(r.LL.X+margin, r.LL.Y+margin, r.UR.X-margin, r.UR.Y-margin)
}

// ToInches converts r to inches.
func (r Rectangle) ToInches() *Rectangle {
	return NewRectangle(r.LL.X*userSpaceToInch, r.LL.Y*userSpaceToInch, r.UR.X*userSpaceToInch, r.UR.Y*userSpaceToInch)
}

// ToCentimetres converts r to centimetres.
func (r Rectangle) ToCentimetres() *Rectangle {
	return NewRectangle(r.LL.X*userSpaceToCm, r.LL.Y*userSpaceToCm, r.UR.X*userSpaceToCm, r.UR.Y*userSpaceToCm)
}

// ToMillimetres converts r to millimetres.
func (r Rectangle) ToMillimetres() *Rectangle {
	return NewRectangle(r.LL.X*userSpaceToMm, r.LL.Y*userSpaceToMm, r.UR.X*userSpaceToMm, r.UR.Y*userSpaceToMm)
}

// ConvertToUnit converts r to unit.
func (r *Rectangle) ConvertToUnit(unit DisplayUnit) *Rectangle {
	switch unit {
	case INCHES:
		return r.ToInches()
	case CENTIMETRES:
		return r.ToCentimetres()
	case MILLIMETRES:
		return r.ToMillimetres()
	}
	return r
}

func (r Rectangle) formatToInches() string {
	return fmt.Sprintf("(%3.2f, %3.2f, %3.2f, %3.2f) w=%.2f h=%.2f ar=%.2f",
		r.LL.X*userSpaceToInch,
		r.LL.Y*userSpaceToInch,
		r.UR.X*userSpaceToInch,
		r.UR.Y*userSpaceToInch,
		r.Width()*userSpaceToInch,
		r.Height()*userSpaceToInch,
		r.AspectRatio())
}

func (r Rectangle) formatToCentimetres() string {
	return fmt.Sprintf("(%3.2f, %3.2f, %3.2f, %3.2f) w=%.2f h=%.2f ar=%.2f",
		r.LL.X*userSpaceToCm,
		r.LL.Y*userSpaceToCm,
		r.UR.X*userSpaceToCm,
		r.UR.Y*userSpaceToCm,
		r.Width()*userSpaceToCm,
		r.Height()*userSpaceToCm,
		r.AspectRatio())
}

func (r Rectangle) formatToMillimetres() string {
	return fmt.Sprintf("(%3.2f, %3.2f, %3.2f, %3.2f) w=%.2f h=%.2f ar=%.2f",
		r.LL.X*userSpaceToMm,
		r.LL.Y*userSpaceToMm,
		r.UR.X*userSpaceToMm,
		r.UR.Y*userSpaceToMm,
		r.Width()*userSpaceToMm,
		r.Height()*userSpaceToMm,
		r.AspectRatio())
}

// Format returns r's details converted into unit.
func (r Rectangle) Format(unit DisplayUnit) string {
	switch unit {
	case INCHES:
		return r.formatToInches()
	case CENTIMETRES:
		return r.formatToCentimetres()
	case MILLIMETRES:
		return r.formatToMillimetres()
	}
	return r.String()
}

///////////////////////////////////////////////////////////////////////////////////

// QuadLiteral is a polygon with four edges and four vertices.
// The four vertices are assumed to be specified in counter clockwise order.
type QuadLiteral struct {
	P1, P2, P3, P4 Point
}

func NewQuadLiteralForRect(r *Rectangle) *QuadLiteral {
	// p1 := Point{X: r.LL.X, Y: r.LL.Y}
	// p2 := Point{X: r.UR.X, Y: r.LL.Y}
	// p3 := Point{X: r.UR.X, Y: r.UR.Y}
	// p4 := Point{X: r.LL.X, Y: r.UR.Y}

	p3 := Point{X: r.LL.X, Y: r.LL.Y}
	p4 := Point{X: r.UR.X, Y: r.LL.Y}
	p2 := Point{X: r.UR.X, Y: r.UR.Y}
	p1 := Point{X: r.LL.X, Y: r.UR.Y}

	return &QuadLiteral{P1: p1, P2: p2, P3: p3, P4: p4}
}

// Array returns the PDF representation of ql.
func (ql QuadLiteral) Array() Array {
	return NewNumberArray(ql.P1.X, ql.P1.Y, ql.P2.X, ql.P2.Y, ql.P3.X, ql.P3.Y, ql.P4.X, ql.P4.Y)
}

// EnclosingRectangle calculates the rectangle enclosing ql's vertices at a distance f.
func (ql QuadLiteral) EnclosingRectangle(f float64) *Rectangle {
	xmin, xmax := ql.P1.X, ql.P1.X
	ymin, ymax := ql.P1.Y, ql.P1.Y
	for _, p := range []Point{ql.P2, ql.P3, ql.P4} {
		if p.X < xmin {
			xmin = p.X
		} else if p.X > xmax {
			xmax = p.X
		}
		if p.Y < ymin {
			ymin = p.Y
		} else if p.Y > ymax {
			ymax = p.Y
		}
	}
	return NewRectangle(xmin-f, ymin-f, xmax+f, ymax+f)
}

// QuadPoints is an array of 8 × n numbers specifying the coordinates of n quadrilaterals in default user space.
type QuadPoints []QuadLiteral

// AddQuadLiteral adds a quadliteral to qp.
func (qp *QuadPoints) AddQuadLiteral(ql QuadLiteral) {
	*qp = append(*qp, ql)
}

// Array returns the PDF representation of qp.
func (qp *QuadPoints) Array() Array {
	a := Array{}
	for _, ql := range *qp {
		a = append(a, ql.Array()...)
	}
	return a
}

///////////////////////////////////////////////////////////////////////////////////

// Name represents a PDF name object.
type Name string

// Clone returns a clone of nameObject.
func (nameObject Name) Clone() Object {
	return nameObject
}

func (nameObject Name) String() string {
	return string(nameObject)
}

// PDFString returns a string representation as found in and written to a PDF file.
func (nameObject Name) PDFString() string {
	s := " "
	if len(nameObject) > 0 {
		s = EncodeName(string(nameObject))
	}
	return fmt.Sprintf("/%s", s)
}

// Value returns a string value for this PDF object.
func (nameObject Name) Value() string {
	return nameObject.String()
}

///////////////////////////////////////////////////////////////////////////////////

// StringLiteral represents a PDF string literal object.
type StringLiteral string

// Clone returns a clone of stringLiteral.
func (stringliteral StringLiteral) Clone() Object {
	return stringliteral
}

func (stringliteral StringLiteral) String() string {
	return fmt.Sprintf("(%s)", string(stringliteral))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (stringliteral StringLiteral) PDFString() string {
	return stringliteral.String()
}

// Value returns a string value for this PDF object.
func (stringliteral StringLiteral) Value() string {
	return string(stringliteral)
}

///////////////////////////////////////////////////////////////////////////////////

// HexLiteral represents a PDF hex literal object.
type HexLiteral string

// NewHexLiteral creates a new HexLiteral for b..
func NewHexLiteral(b []byte) HexLiteral {
	return HexLiteral(hex.EncodeToString(b))
}

// Clone returns a clone of hexliteral.
func (hexliteral HexLiteral) Clone() Object {
	return hexliteral
}
func (hexliteral HexLiteral) String() string {
	return fmt.Sprintf("<%s>", string(hexliteral))
}

// PDFString returns the string representation as found in and written to a PDF file.
func (hexliteral HexLiteral) PDFString() string {
	return hexliteral.String()
}

// Value returns a string value for this PDF object.
func (hexliteral HexLiteral) Value() string {
	return string(hexliteral)
}

// Bytes returns the byte representation.
func (hexliteral HexLiteral) Bytes() ([]byte, error) {
	return hex.DecodeString(hexliteral.Value())
}

///////////////////////////////////////////////////////////////////////////////////

// IndirectRef represents a PDF indirect object.
type IndirectRef struct {
	ObjectNumber     Integer
	GenerationNumber Integer
}

// NewIndirectRef returns a new PDFIndirectRef object.
func NewIndirectRef(objectNumber, generationNumber int) *IndirectRef {
	return &IndirectRef{
		ObjectNumber:     Integer(objectNumber),
		GenerationNumber: Integer(generationNumber)}
}

// Clone returns a clone of ir.
func (ir IndirectRef) Clone() Object {
	ir2 := ir
	return ir2
}

func (ir IndirectRef) String() string {
	return fmt.Sprintf("(%s)", ir.PDFString())
}

// PDFString returns a string representation as found in and written to a PDF file.
func (ir IndirectRef) PDFString() string {
	return fmt.Sprintf("%d %d R", ir.ObjectNumber, ir.GenerationNumber)
}

/////////////////////////////////////////////////////////////////////////////////////

// DisplayUnit is the metric unit used to output paper sizes.
type DisplayUnit int

// Options for display unit in effect.
const (
	POINTS DisplayUnit = iota
	INCHES
	CENTIMETRES
	MILLIMETRES
)

const (
	userSpaceToInch = float64(1) / 72
	userSpaceToCm   = 2.54 / 72
	userSpaceToMm   = userSpaceToCm * 10

	inchToUserSpace = 1 / userSpaceToInch
	cmToUserSpace   = 1 / userSpaceToCm
	mmToUserSpace   = 1 / userSpaceToMm
)

func ToUserSpace(f float64, unit DisplayUnit) float64 {
	switch unit {
	case INCHES:
		return f * inchToUserSpace
	case CENTIMETRES:
		return f * cmToUserSpace
	case MILLIMETRES:
		return f * mmToUserSpace

	}
	return f
}

// Dim represents the dimensions of a rectangular view medium
// like a PDF page, a sheet of paper or an image grid
// in user space, inches, centimetres or millimetres.
type Dim struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ToInches converts d to inches.
func (d Dim) ToInches() Dim {
	return Dim{d.Width * userSpaceToInch, d.Height * userSpaceToInch}
}

// ToCentimetres converts d to centimetres.
func (d Dim) ToCentimetres() Dim {
	return Dim{d.Width * userSpaceToCm, d.Height * userSpaceToCm}
}

// ToMillimetres converts d to millimetres.
func (d Dim) ToMillimetres() Dim {
	return Dim{d.Width * userSpaceToMm, d.Height * userSpaceToMm}
}

// ConvertToUnit converts d to unit.
func (d Dim) ConvertToUnit(unit DisplayUnit) Dim {
	switch unit {
	case INCHES:
		return d.ToInches()
	case CENTIMETRES:
		return d.ToCentimetres()
	case MILLIMETRES:
		return d.ToMillimetres()
	}
	return d
}

// AspectRatio returns the relation between width and height.
func (d Dim) AspectRatio() float64 {
	return d.Width / d.Height
}

// Landscape returns true if d is in landscape mode.
func (d Dim) Landscape() bool {
	return d.AspectRatio() > 1
}

// Portrait returns true if d is in portrait mode.
func (d Dim) Portrait() bool {
	return d.AspectRatio() < 1
}

func (d Dim) String() string {
	return fmt.Sprintf("%fx%f", d.Width, d.Height)
}

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
	"strconv"
	"time"

	"github.com/hhrutter/pdfcpu/pkg/types"
)

// Supported line delimiters
const (
	EolLF   = "\x0A"
	EolCR   = "\x0D"
	EolCRLF = "\x0D\x0A"
)

// ReadSeekerCloser is the interface that groups the ReadSeeker and Close interfaces.
type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

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

	return fmt.Sprintf("%f Bytes", b)
}

// IntSet is a set of integers.
type IntSet map[int]bool

// StringSet is a set of strings.
type StringSet map[string]bool

// Object defines an interface for all Objects.
type Object interface {
	fmt.Stringer
	PDFString() string
}

// Boolean represents a PDF boolean object.
type Boolean bool

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

func (f Float) String() string {
	// Use a precision of 2 for logging readability.
	return fmt.Sprintf("%.2f", float64(f))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (f Float) PDFString() string {
	// The max precision encountered so far has been 11 (fontType3 fontmatrix components).
	return strconv.FormatFloat(f.Value(), 'f', 12, 64)
}

// Value returns a float64 value for this PDF object.
func (f Float) Value() float64 {
	return float64(f)
}

///////////////////////////////////////////////////////////////////////////////////

// Integer represents a PDF integer object.
type Integer int

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
	X, Y float64
}

// Rectangle represents a rectangular region in userspace.
type Rectangle struct {
	*types.Rectangle
}

func (r Rectangle) equals(r2 Rectangle) bool {
	return r.LL == r2.LL && r.UR == r2.UR
}

// FitsWithin returns true if rectangle r fits within rectangle r2.
func (r Rectangle) FitsWithin(r2 *Rectangle) bool {
	return r.Width() <= r2.Width() && r.Height() <= r2.Height()
}

// ScaledWidth returns the width for given height according to r's aspect ratio.
func (r Rectangle) ScaledWidth(h float64) float64 {
	return r.AspectRatio() * h
}

// ScaledHeight returns the height for given width according to r's aspect ratio.
func (r Rectangle) ScaledHeight(w float64) float64 {
	return w / r.AspectRatio()
}

// Array returns the PDF representation of a rectangle.
func (r Rectangle) Array() Array {
	return NewNumberArray(r.LL.X, r.LL.Y, r.UR.X, r.UR.Y)
}

// CroppedCopy returns a copy of r with applied margin..
func (r Rectangle) CroppedCopy(margin float64) *Rectangle {
	return Rect(
		r.LL.X+margin,
		r.LL.Y+margin,
		r.UR.X-margin,
		r.UR.Y-margin,
	)
}

// Rect returns a new rectangle for given lower left and upper right corners.
func Rect(llx, lly, urx, ury float64) *Rectangle {
	return &Rectangle{types.NewRectangle(llx, lly, urx, ury)}
}

// RectForArray returns a new rectangle for given Array.
func RectForArray(a Array) *Rectangle {
	return Rect(
		a[0].(Float).Value(),
		a[1].(Float).Value(),
		a[2].(Float).Value(),
		a[3].(Float).Value(),
	)
}

// RectForDim returns a new rectangle for given dimensions.
func RectForDim(width, height int) *Rectangle {
	return Rect(0.0, 0.0, float64(width), float64(height))
}

// RectForFormat returns a new rectangle for given format.
func RectForFormat(f string) *Rectangle {
	d := PaperSize[f]
	return RectForDim(d.w, d.h)
}

///////////////////////////////////////////////////////////////////////////////////

// Name represents a PDF name object.
type Name string

func (nameObject Name) String() string {
	return fmt.Sprintf("%s", string(nameObject))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (nameObject Name) PDFString() string {
	s := " "
	if len(nameObject) > 0 {
		s = string(nameObject)
	}
	return fmt.Sprintf("/%s", s)
}

// Value returns a string value for this PDF object.
func (nameObject Name) Value() string {

	s := string(nameObject)
	var b bytes.Buffer

	for i := 0; i < len(s); {
		c := s[i]
		if c != '#' {
			b.WriteByte(c)
			i++
			continue
		}

		// # detected, next 2 chars have to exist.
		// This gets checked during parsing.
		s1 := s[i+1 : i+3]
		b1, _ := hex.DecodeString(s1)
		b.WriteByte(b1[0])
		i += 3
	}

	return b.String()
}

///////////////////////////////////////////////////////////////////////////////////

// StringLiteral represents a PDF string literal object.
type StringLiteral string

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

// DateString returns a string representation of t.
func DateString(t time.Time) string {

	_, tz := t.Zone()

	return fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		tz/60/60, tz/60%60)
}

///////////////////////////////////////////////////////////////////////////////////

// HexLiteral represents a PDF hex literal object.
type HexLiteral string

// NewHexLiteral creates a new HexLiteral for b..
func NewHexLiteral(b []byte) HexLiteral {
	return HexLiteral(hex.EncodeToString(b))
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
	b, err := hex.DecodeString(hexliteral.Value())
	if err != nil {
		return nil, err
	}
	return b, err
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

func (ir IndirectRef) String() string {
	return fmt.Sprintf("(%s)", ir.PDFString())
}

// PDFString returns a string representation as found in and written to a PDF file.
func (ir IndirectRef) PDFString() string {
	return fmt.Sprintf("%d %d R", ir.ObjectNumber, ir.GenerationNumber)
}

// Equals returns true if two indirect References refer to the same object.
func (ir IndirectRef) Equals(indRef IndirectRef) bool {
	return ir.ObjectNumber == indRef.ObjectNumber &&
		ir.GenerationNumber == indRef.GenerationNumber
}

// Package types provides the PDFContext, representing an ecosystem for PDF processing.
//
// It implements the specification PDF 32000-1:2008
//
// Please refer to the spec for any documentation of PDFContext's internals.
package types

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
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

	return fmt.Sprintf("%f Bytes", b)
}

// IntSet is a set of integers.
type IntSet map[int]bool

// StringSet is a set of strings.
type StringSet map[string]bool

// PDFObject defines an interface for all PDFObjects.
type PDFObject interface {
	fmt.Stringer
	PDFString() string
}

// PDFBoolean represents a PDF boolean object.
type PDFBoolean bool

func (boolean PDFBoolean) String() string {
	return fmt.Sprintf("%v", bool(boolean))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (boolean PDFBoolean) PDFString() string {
	return boolean.String()
}

// Value returns a bool value for this PDF object.
func (boolean PDFBoolean) Value() bool {
	return bool(boolean)
}

///////////////////////////////////////////////////////////////////////////////////

// PDFFloat represents a PDF float object.
type PDFFloat float64

func (f PDFFloat) String() string {
	// strconv may be faster.
	return fmt.Sprintf("%.2f", float64(f))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (f PDFFloat) PDFString() string {
	return f.String()
}

// Value returns a float64 value for this PDF object.
func (f PDFFloat) Value() float64 {
	return float64(f)
}

///////////////////////////////////////////////////////////////////////////////////

// PDFInteger represents a PDF integer object.
type PDFInteger int

func (i PDFInteger) String() string {
	return strconv.Itoa(int(i))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (i PDFInteger) PDFString() string {
	return i.String()
}

// Value returns an int value for this PDF object.
func (i PDFInteger) Value() int {
	return int(i)
}

///////////////////////////////////////////////////////////////////////////////////

// NewRectangle creates a rectangle array
func NewRectangle(llx, lly, urx, ury float64) PDFArray {
	return NewNumberArray(llx, lly, urx, ury)
}

///////////////////////////////////////////////////////////////////////////////////

// PDFName represents a PDF name object.
type PDFName string

func (nameObject PDFName) String() string {
	return fmt.Sprintf("%s", string(nameObject))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (nameObject PDFName) PDFString() string {
	s := " "
	if len(nameObject) > 0 {
		s = string(nameObject)
	}
	return fmt.Sprintf("/%s", s)
}

// Value returns a string value for this PDF object.
func (nameObject PDFName) Value() string {
	return string(nameObject)
}

///////////////////////////////////////////////////////////////////////////////////

// PDFStringLiteral represents a PDF string literal object.
type PDFStringLiteral string

func (stringliteral PDFStringLiteral) String() string {
	return fmt.Sprintf("(%s)", string(stringliteral))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (stringliteral PDFStringLiteral) PDFString() string {
	return stringliteral.String()
}

// Value returns a string value for this PDF object.
func (stringliteral PDFStringLiteral) Value() string {
	return string(stringliteral)
}

// DateStringLiteral returns a PDFStringLiteral for time.
func DateStringLiteral(t time.Time) PDFStringLiteral {

	_, tz := t.Zone()

	dateStr := fmt.Sprintf("D:%d%02d%02d%02d%02d%02d+%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		tz/60/60, tz/60%60)

	return PDFStringLiteral(dateStr)
}

///////////////////////////////////////////////////////////////////////////////////

// PDFHexLiteral represents a PDF hex literal object.
type PDFHexLiteral string

func (hexliteral PDFHexLiteral) String() string {
	return fmt.Sprintf("<%s>", string(hexliteral))
}

// PDFString returns the string representation as found in and written to a PDF file.
func (hexliteral PDFHexLiteral) PDFString() string {
	return hexliteral.String()
}

// Value returns a string value for this PDF object.
func (hexliteral PDFHexLiteral) Value() string {
	return string(hexliteral)
}

// Bytes returns the byte representation.
func (hexliteral PDFHexLiteral) Bytes() ([]byte, error) {
	b, err := hex.DecodeString(hexliteral.Value())
	if err != nil {
		return nil, err
	}
	return b, err
}

///////////////////////////////////////////////////////////////////////////////////

// PDFIndirectRef represents a PDF indirect object.
type PDFIndirectRef struct {
	ObjectNumber     PDFInteger
	GenerationNumber PDFInteger
}

// NewPDFIndirectRef returns a new PDFIndirectRef object.
func NewPDFIndirectRef(objectNumber, generationNumber int) *PDFIndirectRef {
	return &PDFIndirectRef{
		ObjectNumber:     PDFInteger(objectNumber),
		GenerationNumber: PDFInteger(generationNumber)}
}

func (ir PDFIndirectRef) String() string {
	return fmt.Sprintf("(%s)", ir.PDFString())
}

// PDFString returns a string representation as found in and written to a PDF file.
func (ir PDFIndirectRef) PDFString() string {
	return fmt.Sprintf("%d %d R", ir.ObjectNumber, ir.GenerationNumber)
}

// Equals returns true if two indirect References refer to the same object.
func (ir PDFIndirectRef) Equals(indRef PDFIndirectRef) bool {
	return ir.ObjectNumber == indRef.ObjectNumber &&
		ir.GenerationNumber == indRef.GenerationNumber
}

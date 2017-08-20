// Package types provides the PDFContext, representing an ecosystem for PDF processing.
//
// It implements the specification PDF 32000-1:2008
//
// Please refer to the spec for any documentation of PDFContext's internals.
package types

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

// Supported line delimiters
const (
	EolLF   = "\x0A"
	EolCR   = "\x0D"
	EolCRLF = "\x0D\x0A"

	FreeHeadGeneration = 65535
)

var logDebugTypes, logInfoTypes, logErrorTypes *log.Logger

func init() {

	//logDebugTypes = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logDebugTypes = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoTypes = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorTypes = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	if verbose {
		logInfoTypes = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logInfoTypes = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
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

///////////////////////////////////////////////////////////////////////////////////

// PDFHexLiteral represents a PDF hex literal object.
type PDFHexLiteral string

func (hexliteral PDFHexLiteral) String() string {
	return fmt.Sprintf("<%s>", string(hexliteral))
}

// PDFString returns a string representation as found in and written to a PDF file.
func (hexliteral PDFHexLiteral) PDFString() string {
	return hexliteral.String()
}

// Value returns a string value for this PDF object.
func (hexliteral PDFHexLiteral) Value() string {
	return string(hexliteral)
}

///////////////////////////////////////////////////////////////////////////////////

// PDFIndirectRef represents a PDF indirect object.
type PDFIndirectRef struct {
	ObjectNumber     PDFInteger
	GenerationNumber PDFInteger
}

// NewPDFIndirectRef returns a new PDFIndirectRef object.
func NewPDFIndirectRef(objectNumber, generationNumber int) PDFIndirectRef {
	return PDFIndirectRef{
		ObjectNumber:     PDFInteger(objectNumber),
		GenerationNumber: PDFInteger(generationNumber)}
}

func (indirectRef PDFIndirectRef) String() string {
	return fmt.Sprintf("(%d %d R)", indirectRef.ObjectNumber, indirectRef.GenerationNumber)
}

// PDFString returns a string representation as found in and written to a PDF file.
func (indirectRef PDFIndirectRef) PDFString() string {
	return fmt.Sprintf("%d %d R", indirectRef.ObjectNumber, indirectRef.GenerationNumber)
}

// Equals returns true if two indirect References refer to the same object.
func (indirectRef PDFIndirectRef) Equals(indRef PDFIndirectRef) bool {
	return indirectRef.ObjectNumber == indRef.ObjectNumber &&
		indirectRef.GenerationNumber == indRef.GenerationNumber
}

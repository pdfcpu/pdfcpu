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
	"fmt"

	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

// Array represents a PDF array object.
type Array []Object

// NewStringLiteralArray returns a PDFArray with StringLiteral entries.
func NewStringLiteralArray(sVars ...string) Array {

	a := Array{}

	for _, s := range sVars {
		a = append(a, StringLiteral(s))
	}

	return a
}

// NewHexLiteralArray returns a PDFArray with HexLiteralLiteral entries.
func NewHexLiteralArray(sVars ...string) Array {

	a := Array{}

	for _, s := range sVars {
		a = append(a, NewHexLiteral([]byte(s)))
	}

	return a
}

// NewNameArray returns a PDFArray with Name entries.
func NewNameArray(sVars ...string) Array {

	a := Array{}

	for _, s := range sVars {
		a = append(a, Name(s))
	}

	return a
}

// NewNumberArray returns a PDFArray with Float entries.
func NewNumberArray(fVars ...float64) Array {

	a := Array{}

	for _, f := range fVars {
		a = append(a, Float(f))
	}

	return a
}

// NewIntegerArray returns a PDFArray with Integer entries.
func NewIntegerArray(fVars ...int) Array {

	a := Array{}

	for _, f := range fVars {
		a = append(a, Integer(f))
	}

	return a
}

// Clone returns a clone of a.
func (a Array) Clone() Object {
	a1 := Array(make([]Object, len(a)))
	for k, v := range a {
		if v != nil {
			v = v.Clone()
		}
		a1[k] = v
	}
	return a1
}

func (a Array) indentedString(level int) string {

	logstr := []string{"["}
	tabstr := strings.Repeat("\t", level)
	first := true
	sepstr := ""

	for _, entry := range a {

		if first {
			first = false
			sepstr = ""
		} else {
			sepstr = " "
		}

		switch entry := entry.(type) {
		case Dict:
			dictstr := entry.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("\n%[1]s%[2]s\n%[1]s", tabstr, dictstr))
			first = true
		case Array:
			arrstr := entry.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, arrstr))
		default:
			v := "null"
			if entry != nil {
				v = entry.String()
				if n, ok := entry.(Name); ok {
					v, _ = DecodeName(string(n))
				}
			}

			logstr = append(logstr, fmt.Sprintf("%s%v", sepstr, v))
		}
	}

	logstr = append(logstr, "]")

	return strings.Join(logstr, "")
}

func (a Array) String() string {
	return a.indentedString(1)
}

// PDFString returns a string representation as found in and written to a PDF file.
func (a Array) PDFString() string {

	logstr := []string{}
	logstr = append(logstr, "[")
	first := true
	var sepstr string

	for _, entry := range a {

		if first {
			first = false
			sepstr = ""
		} else {
			sepstr = " "
		}

		switch entry := entry.(type) {
		case nil:
			logstr = append(logstr, fmt.Sprintf("%snull", sepstr))
		case Dict:
			logstr = append(logstr, entry.PDFString())
		case Array:
			logstr = append(logstr, entry.PDFString())
		case IndirectRef:
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, entry.PDFString()))
		case Name:
			logstr = append(logstr, entry.PDFString())
		case Integer:
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, entry.PDFString()))
		case Float:
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, entry.PDFString()))
		case Boolean:
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, entry.PDFString()))
		case StringLiteral:
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, entry.PDFString()))
		case HexLiteral:
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, entry.PDFString()))
		default:
			if log.InfoEnabled() {
				log.Info.Fatalf("PDFArray.PDFString(): entry of unknown object type: %[1]T %[1]v\n", entry)
			}
		}
	}

	logstr = append(logstr, "]")

	return strings.Join(logstr, "")
}

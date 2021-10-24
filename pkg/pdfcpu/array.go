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
	"fmt"

	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

// Array represents a PDF array object.
type Array []Object

// NewStringArray returns a PDFArray with StringLiteral entries.
func NewStringArray(sVars ...string) Array {

	a := Array{}

	for _, s := range sVars {
		a = append(a, StringLiteral(s))
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

func (a Array) contains(o Object, xRefTable *XRefTable) (bool, error) {
	for _, e := range a {
		ok, err := EqualObjects(e, o, xRefTable)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
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

		if subdict, ok := entry.(Dict); ok {
			dictstr := subdict.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("\n%[1]s%[2]s\n%[1]s", tabstr, dictstr))
			first = true
			continue
		}

		if array, ok := entry.(Array); ok {
			arrstr := array.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, arrstr))
			continue
		}

		logstr = append(logstr, fmt.Sprintf("%s%v", sepstr, entry))
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

		if entry == nil {
			logstr = append(logstr, fmt.Sprintf("%snull", sepstr))
			continue
		}

		d, ok := entry.(Dict)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s", d.PDFString()))
			continue
		}

		a, ok := entry.(Array)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s", a.PDFString()))
			continue
		}

		ir, ok := entry.(IndirectRef)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, ir.PDFString()))
			continue
		}

		n, ok := entry.(Name)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s", n.PDFString()))
			continue
		}

		i, ok := entry.(Integer)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, i.PDFString()))
			continue
		}

		f, ok := entry.(Float)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, f.PDFString()))
			continue
		}

		b, ok := entry.(Boolean)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, b.PDFString()))
			continue
		}
		sl, ok := entry.(StringLiteral)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, sl.PDFString()))
			continue
		}

		hl, ok := entry.(HexLiteral)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, hl.PDFString()))
			continue
		}

		log.Info.Fatalf("PDFArray.PDFString(): entry of unknown object type: %[1]T %[1]v\n", entry)
	}

	logstr = append(logstr, "]")

	return strings.Join(logstr, "")
}

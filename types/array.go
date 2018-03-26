package types

import (
	"fmt"
	"log"
	"strings"
)

// PDFArray represents a PDF array object.
type PDFArray []PDFObject

// NewStringArray returns a PDFArray with PDFStringLiteral entries.
func NewStringArray(sVars ...string) PDFArray {

	a := PDFArray{}

	for _, s := range sVars {
		a = append(a, PDFStringLiteral(s))
	}

	return a
}

// NewNameArray returns a PDFArray with PDFName entries.
func NewNameArray(sVars ...string) PDFArray {

	a := PDFArray{}

	for _, s := range sVars {
		a = append(a, PDFName(s))
	}

	return a
}

// NewNumberArray returns a PDFArray with PDFFloat entries.
func NewNumberArray(fVars ...float64) PDFArray {

	a := PDFArray{}

	for _, f := range fVars {
		a = append(a, PDFFloat(f))
	}

	return a
}

// NewIntegerArray returns a PDFArray with PDFInteger entries.
func NewIntegerArray(fVars ...int) PDFArray {

	a := PDFArray{}

	for _, f := range fVars {
		a = append(a, PDFInteger(f))
	}

	return a
}

func (array PDFArray) indentedString(level int) string {

	logstr := []string{"["}
	tabstr := strings.Repeat("\t", level)
	first := true
	sepstr := ""

	for _, entry := range array {

		if first {
			first = false
			sepstr = ""
		} else {
			sepstr = " "
		}

		if subdict, ok := entry.(PDFDict); ok {
			dictstr := subdict.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("\n%[1]s%[2]s\n%[1]s", tabstr, dictstr))
			first = true
			continue
		}

		if array, ok := entry.(PDFArray); ok {
			arrstr := array.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, arrstr))
			continue
		}

		logstr = append(logstr, fmt.Sprintf("%s%v", sepstr, entry))
	}

	logstr = append(logstr, "]")

	return strings.Join(logstr, "")
}

func (array PDFArray) String() string {
	return array.indentedString(1)
}

// PDFString returns a string representation as found in and written to a PDF file.
func (array PDFArray) PDFString() string {

	logstr := []string{}
	logstr = append(logstr, "[")
	first := true
	var sepstr string

	for _, entry := range array {

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

		subdict, ok := entry.(PDFDict)
		if ok {
			dictStr := subdict.PDFString()
			logstr = append(logstr, fmt.Sprintf("%s", dictStr))
			continue
		}

		array, ok := entry.(PDFArray)
		if ok {
			arrstr := array.PDFString()
			logstr = append(logstr, fmt.Sprintf("%s", arrstr))
			continue
		}

		indRef, ok := entry.(PDFIndirectRef)
		if ok {
			indRefstr := indRef.PDFString()
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, indRefstr))
			continue
		}

		name, ok := entry.(PDFName)
		if ok {
			namestr := name.PDFString()
			logstr = append(logstr, fmt.Sprintf("%s", namestr))
			continue
		}

		i, ok := entry.(PDFInteger)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, i.String()))
			continue
		}

		f, ok := entry.(PDFFloat)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, f.String()))
			continue
		}

		b, ok := entry.(PDFBoolean)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, b.String()))
			continue
		}
		sl, ok := entry.(PDFStringLiteral)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, sl.String()))
			continue
		}

		hl, ok := entry.(PDFHexLiteral)
		if ok {
			logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, hl.String()))
			continue
		}

		log.Fatalf("PDFArray.PDFString(): entry of unknown object type: %[1]T %[1]v\n", entry)
	}

	logstr = append(logstr, "]")

	return strings.Join(logstr, "")
}

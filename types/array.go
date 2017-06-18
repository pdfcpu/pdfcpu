package types

import (
	"fmt"
	"log"
	"strings"
)

// PDFArray represents a PDF array object.
type PDFArray []interface{}

func (array PDFArray) string(i int, arr *PDFArray) string {

	logstr := []string{} //make([]string, 20)
	logstr = append(logstr, "[")
	tabstr := strings.Repeat("\t", i)
	first := true
	sepstr := ""

	for _, entry := range *arr {

		if first {
			first = false
			sepstr = ""
		} else {
			sepstr = " "
		}

		subdict, ok := entry.(PDFDict)

		if ok {
			dictstr := subdict.string(i+1, &subdict)
			logstr = append(logstr, fmt.Sprintf("\n%s%s\n%s", tabstr, dictstr, tabstr))
			first = true
		} else {
			array, ok := entry.(PDFArray)
			if ok {
				arrstr := array.string(i+1, &array)
				logstr = append(logstr, fmt.Sprintf("%s%s", sepstr, arrstr))
			} else {
				logstr = append(logstr, fmt.Sprintf("%s%v", sepstr, entry))
			}
		}

	}

	logstr = append(logstr, "]")
	return strings.Join(logstr, "")
}

func (array PDFArray) String() string {
	return array.string(1, &array)
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

		log.Fatalf("PDFArray.PDFString(): unknown entry: %s\n", entry)
	}

	logstr = append(logstr, "]")

	return strings.Join(logstr, "")
}

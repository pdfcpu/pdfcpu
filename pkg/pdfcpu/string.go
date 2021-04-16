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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// NewStringSet returns a new StringSet for slice.
func NewStringSet(slice []string) StringSet {
	strSet := StringSet{}
	if slice == nil {
		return strSet
	}
	for _, s := range slice {
		strSet[s] = true
	}
	return strSet
}

// Convert a 1,2 or 3 digit unescaped octal string into the corresponding byte value.
func byteForOctalString(octalBytes string) (b byte) {
	i, _ := strconv.ParseInt(octalBytes, 8, 64)
	return byte(i)
}

// Escape applies all defined escape sequences to s.
func Escape(s string) (*string, error) {

	var b bytes.Buffer

	for i := 0; i < len(s); i++ {

		c := s[i]

		switch c {
		case 0x0A:
			c = 'n'
		case 0x0D:
			c = 'r'
		case 0x09:
			c = 't'
		case 0x08:
			c = 'b'
		case 0x0C:
			c = 'f'
		case '\\', '(', ')':
		default:
			b.WriteByte(c)
			continue
		}

		b.WriteByte('\\')
		b.WriteByte(c)
	}

	s1 := b.String()

	return &s1, nil
}

func escaped(c byte) (bool, byte) {

	switch c {
	case 'n':
		c = 0x0A
	case 'r':
		c = 0x0D
	case 't':
		c = 0x09
	case 'b':
		c = 0x08
	case 'f':
		c = 0x0C
	case '(', ')':
	case '0', '1', '2', '3', '4', '5', '6', '7':
		return true, c
	}

	return false, c
}

func regularChar(c byte, esc bool) bool {
	return c != 0x5c && !esc
}

// Unescape resolves all escape sequences of s.
func Unescape(s string) ([]byte, error) {

	var esc bool
	var longEol bool
	var octalCode string
	var b bytes.Buffer

	for i := 0; i < len(s); i++ {

		c := s[i]

		if longEol {
			esc = false
			longEol = false
			// c is completing a 0x5C0D0A line break.
			if c == 0x0A {
				continue
			}
		}

		if regularChar(c, esc) {
			b.WriteByte(c)
			continue
		}

		if c == 0x5c { // '\'
			if !esc { // Start escape sequence.
				esc = true
			} else { // Escaped \
				if len(octalCode) > 0 {
					return nil, errors.Errorf("Unescape: illegal \\ in octal code sequence detected %X", octalCode)
				}
				b.WriteByte(c)
				esc = false
			}
			continue
		}

		// escaped = true && any other than \

		if len(octalCode) > 0 {
			if !strings.ContainsRune("01234567", rune(c)) {
				return nil, errors.Errorf("Unescape: illegal octal sequence detected %X", octalCode)
			}
			octalCode = octalCode + string(c)
			if len(octalCode) == 3 {
				b.WriteByte(byteForOctalString(octalCode))
				octalCode = ""
				esc = false
			}
			continue
		}

		// Ignore \eol line breaks.
		if c == 0x0A {
			esc = false
			continue
		}

		if c == 0x0D {
			longEol = true
			continue
		}

		// Relax for issue 305 and also accept "\ ".
		if !strings.ContainsRune(" nrtbf()01234567", rune(c)) {
			return nil, errors.Errorf("Unescape: illegal escape sequence \\%c detected: <%s>", c, s)
		}

		var octal bool
		octal, c = escaped(c)
		if octal {
			octalCode = octalCode + string(c)
			continue
		}

		b.WriteByte(c)
		esc = false
	}

	return b.Bytes(), nil
}

// This is a patched version of strings.FieldsFunc that also returns empty fields.
func fieldsFunc(s string, f func(rune) bool) []string {
	// A span is used to record a slice of s of the form s[start:end].
	// The start index is inclusive and the end index is exclusive.
	type span struct {
		start int
		end   int
	}
	spans := make([]span, 0, 32)

	// Find the field start and end indices.
	wasField := false
	fromIndex := 0
	for i, rune := range s {
		if f(rune) {
			if wasField {
				spans = append(spans, span{start: fromIndex, end: i})
				wasField = false
			} else {
				spans = append(spans, span{})
			}
		} else {
			if !wasField {
				fromIndex = i
				wasField = true
			}
		}
	}

	// Last field might end at EOF.
	if wasField {
		spans = append(spans, span{fromIndex, len(s)})
	}

	// Create strings from recorded field indices.
	a := make([]string, len(spans))
	for i, span := range spans {
		a[i] = s[span.start:span.end]
	}

	return a
}

// UTF8ToCP1252 converts UTF-8 to CP1252.
func UTF8ToCP1252(s string) string {
	bb := []byte{}
	for _, r := range s {
		bb = append(bb, byte(r))
	}
	return string(bb)
}

// CP1252ToUTF8 converts CP1252 to UTF-8.
func CP1252ToUTF8(s string) string {
	utf8Buf := make([]byte, utf8.UTFMax)
	bb := []byte{}
	for i := 0; i < len(s); i++ {
		n := utf8.EncodeRune(utf8Buf, rune(s[i]))
		bb = append(bb, utf8Buf[:n]...)
	}
	return string(bb)
}

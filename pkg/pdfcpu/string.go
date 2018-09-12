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
	"math"
	"strings"

	"github.com/pkg/errors"
)

// Convert a 1,2 or 3 digit unescaped octal string into the corresponding byte value.
func byteForOctalString(octalBytes []byte) (b byte) {

	var j float64

	for i := len(octalBytes) - 1; i >= 0; i-- {
		b += (octalBytes[i] - '0') * byte(math.Pow(8, j))
		j++
	}

	return
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
	var octalCode []byte
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
			octalCode = append(octalCode, c)
			if len(octalCode) == 3 {
				b.WriteByte(byteForOctalString(octalCode))
				octalCode = nil
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

		if !strings.ContainsRune("nrtbf()01234567", rune(c)) {
			return nil, errors.Errorf("Unescape: illegal escape sequence \\%c detected", c)
		}

		var octal bool
		octal, c = escaped(c)
		if octal {
			octalCode = append(octalCode, c)
			continue
		}

		b.WriteByte(c)
		esc = false
	}

	return b.Bytes(), nil
}

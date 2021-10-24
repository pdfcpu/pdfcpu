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
	"encoding/hex"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"

	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// ErrInvalidUTF16BE represents an error that gets raised for invalid UTF-16BE byte sequences.
var ErrInvalidUTF16BE = errors.New("pdfcpu: invalid UTF-16BE detected")

// IsStringUTF16BE checks a string for Big Endian byte order BOM.
func IsStringUTF16BE(s string) bool {
	s1 := fmt.Sprintf("%s", s)
	ok := strings.HasPrefix(s1, "\376\377") // 0xFE 0xFF
	return ok
}

// IsUTF16BE checks for Big Endian byte order mark and valid length.
func IsUTF16BE(b []byte) bool {
	if len(b) == 0 || len(b)%2 != 0 {
		return false
	}
	// Check BOM
	return b[0] == 0xFE && b[1] == 0xFF
}

func decodeUTF16String(b []byte) (string, error) {
	// We only accept big endian byte order.
	if !IsUTF16BE(b) {
		log.Debug.Printf("decodeUTF16String: not UTF16BE: %s\n", hex.Dump(b))
		return "", ErrInvalidUTF16BE
	}

	// Strip BOM.
	b = b[2:]

	// code points
	u16 := make([]uint16, 0, len(b))

	// Collect code points.
	for i := 0; i < len(b); {

		val := (uint16(b[i]) << 8) + uint16(b[i+1])

		if val <= 0xD7FF || val > 0xE000 && val <= 0xFFFF {
			// Basic Multilingual Plane
			u16 = append(u16, val)
			i += 2
			continue
		}

		// Ensure bytes needed in order to decode surrogate pair.
		if i+2 >= len(b) {
			return "", errors.Errorf("decodeUTF16String: corrupt UTF16BE byte length on unicode point 1: %v", b)
		}

		// Ensure high surrogate is leading in possible surrogate pair.
		if val >= 0xDC00 && val <= 0xDFFF {
			return "", errors.Errorf("decodeUTF16String: corrupt UTF16BE on unicode point 1: %v", b)
		}

		// Supplementary Planes
		u16 = append(u16, val)
		val = (uint16(b[i+2]) << 8) + uint16(b[i+3])
		if val < 0xDC00 || val > 0xDFFF {
			return "", errors.Errorf("decodeUTF16String: corrupt UTF16BE on unicode point 2: %v", b)
		}

		u16 = append(u16, val)
		i += 4
	}

	decb := []byte{}
	utf8Buf := make([]byte, utf8.UTFMax)

	for _, rune := range utf16.Decode(u16) {
		n := utf8.EncodeRune(utf8Buf, rune)
		decb = append(decb, utf8Buf[:n]...)
	}

	return string(decb), nil
}

// DecodeUTF16String decodes a UTF16BE string from a hex string.
func DecodeUTF16String(s string) (string, error) {
	return decodeUTF16String([]byte(s))
}

func EncodeUTF16String(s string) string {
	rr := utf16.Encode([]rune(s))
	bb := []byte{0xFE, 0xFF}
	for _, r := range rr {
		bb = append(bb, byte(r>>8), byte(r&0xFF))
	}
	return string(bb)
}

// StringLiteralToString returns the best possible string rep for a string literal.
func StringLiteralToString(sl StringLiteral) (string, error) {
	bb, err := Unescape(sl.Value())
	if err != nil {
		return "", err
	}

	s1 := string(bb)

	// Check for Big Endian UTF-16.
	if IsStringUTF16BE(s1) {
		return DecodeUTF16String(s1)
	}

	// if no acceptable UTF16 encoding found, ensure utf8 encoding.
	if !utf8.ValidString(s1) {
		s1 = CP1252ToUTF8(s1)
	}
	return s1, nil
}

// HexLiteralToString returns a possibly UTF16 encoded string for a hex string.
func HexLiteralToString(hl HexLiteral) (string, error) {
	// Get corresponding byte slice.
	b, err := hex.DecodeString(hl.Value())
	if err != nil {
		return "", err
	}

	// Check for Big Endian UTF-16.
	if IsUTF16BE(b) {
		return decodeUTF16String(b)
	}

	// if no acceptable UTF16 encoding found, just return decoded hexstring.
	return string(b), nil
}

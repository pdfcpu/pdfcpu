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
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/mechiko/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// ErrInvalidUTF16BE represents an error that gets raised for invalid UTF-16BE byte sequences.
var ErrInvalidUTF16BE = errors.New("pdfcpu: invalid UTF-16BE detected")

// IsStringUTF16BE checks a string for Big Endian byte order BOM.
func IsStringUTF16BE(s string) bool {
	s1 := fmt.Sprint(s)
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
	// Convert UTF-16 to UTF-8
	// We only accept big endian byte order.
	if !IsUTF16BE(b) {
		if log.DebugEnabled() {
			log.Debug.Printf("decodeUTF16String: not UTF16BE: %s\n", hex.Dump(b))
		}
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

func EscapedUTF16String(s string) (*string, error) {
	return Escape(EncodeUTF16String(s))
}

// StringLiteralToString returns the best possible string rep for a string literal.
func StringLiteralToString(sl StringLiteral) (string, error) {
	bb, err := Unescape(sl.Value())
	if err != nil {
		return "", err
	}
	if IsUTF16BE(bb) {
		return decodeUTF16String(bb)
	}
	// if no acceptable UTF16 encoding found, ensure utf8 encoding.
	bb = bytes.TrimPrefix(bb, []byte{239, 187, 191})
	s := string(bb)
	if !utf8.ValidString(s) {
		s = CP1252ToUTF8(s)
	}
	return s, nil
}

// HexLiteralToString returns a possibly UTF16 encoded string for a hex string.
func HexLiteralToString(hl HexLiteral) (string, error) {
	bb, err := hl.Bytes()
	if err != nil {
		return "", err
	}
	if IsUTF16BE(bb) {
		return decodeUTF16String(bb)
	}

	bb, err = Unescape(string(bb))
	if err != nil {
		return "", err
	}

	bb = bytes.TrimPrefix(bb, []byte{239, 187, 191})

	return string(bb), nil
}

func StringOrHexLiteral(obj Object) (*string, error) {
	if sl, ok := obj.(StringLiteral); ok {
		s, err := StringLiteralToString(sl)
		return &s, err
	}
	if hl, ok := obj.(HexLiteral); ok {
		s, err := HexLiteralToString(hl)
		return &s, err
	}
	return nil, errors.New("pdfcpu: expected StringLiteral or HexLiteral")
}

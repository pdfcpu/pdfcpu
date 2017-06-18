package types

import (
	"encoding/hex"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"

	"strings"

	"github.com/pkg/errors"
)

// IsStringUTF16BE checks for Big Endian byte order BOM in octal.
func IsStringUTF16BE(s string) bool {
	// UTF16-BE BOM disguised as octal.

	s1 := fmt.Sprintf("%s", s)

	ok := strings.HasPrefix(s1, "\376\377")

	logDebugTypes.Printf("IsStringUTF16BE: <%s> returning %v\n", s1, ok)
	logDebugTypes.Printf("\n%s", hex.Dump([]byte(s1)))

	return ok
}

// IsUTF16BE checks for Big Endian byte order mark.
func IsUTF16BE(b []byte) (ok bool, err error) {

	if len(b) == 0 {
		return
	}

	if len(b)%2 != 0 {
		err = errors.Errorf("DecodeUTF16String: UTF16 needs even number of bytes: %v\n", b)
		return
	}

	// Check BOM
	ok = b[0] == 0xFE && b[1] == 0xFF

	return
}

func decodeUTF16String(b []byte) (s string, err error) {

	logDebugTypes.Printf("decodeUTF16String: begin %v\n", b)

	// Check for Big Endian UTF-16.
	isUTF16BE, err := IsUTF16BE(b)
	if err != nil {
		return
	}

	// We only accept big endian byte order.
	if !isUTF16BE {
		err = errors.Errorf("decodeUTF16String: not UTF16BE: %v\n", b)
		return
	}

	// Strip BOM.
	b = b[2:]

	// code points
	u16 := make([]uint16, 0, len(b))

	// Collect code points.
	for i := 0; i < len(b); {

		logDebugTypes.Printf("i=%d\n", i)

		val := (uint16(b[i]) << 8) + uint16(b[i+1])

		if val <= 0xD7FF || val > 0xE000 && val <= 0xFFFF {
			// Basic Multilingual Plane
			logDebugTypes.Println("decodeUTF16String: Basic Multilingual Plane detected")
			u16 = append(u16, val)
			i += 2
			continue
		}

		// Ensure bytes needed in order to decode surrogate pair.
		if i+2 >= len(b) {
			err = errors.Errorf("decodeUTF16String: corrupt UTF16BE on unicode point 1: %v", b)
			return
		}

		// Ensure high surrogate is leading in possible surrogate pair.
		if val >= 0xDC00 && val <= 0xDFFF {
			err = errors.Errorf("decodeUTF16String: corrupt UTF16BE on unicode point 1: %v", b)
			return
		}

		// Supplementary Planes
		logDebugTypes.Println("decodeUTF16String: Supplementary Planes detected")
		u16 = append(u16, val)
		val = (uint16(b[i+2]) << 8) + uint16(b[i+3])
		if val < 0xDC00 || val > 0xDFFF {
			err = errors.Errorf("decodeUTF16String: corrupt UTF16BE on unicode point 2: %v", b)
			return
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

	s = string(decb)

	logDebugTypes.Printf("decodeUTF16String: end %s %s %s\n", s, hex.Dump(decb), hex.Dump([]byte(s)))

	return
}

// DecodeUTF16String decodes a UTF16BE string from a hex string.
func DecodeUTF16String(s string) (string, error) {

	return decodeUTF16String([]byte(s))
}

// StringLiteralToString returns the best possible string rep for a string literal.
func StringLiteralToString(str string) (s string, err error) {

	// Check for Big Endian UTF-16.
	if IsStringUTF16BE(s) {
		return DecodeUTF16String(s)
	}

	// if no acceptable UTF16 encoding found, just return str.

	return str, nil
}

// HexLiteralToString returns a possibly UTF16 encoded string for a hex string.
func HexLiteralToString(hexString string) (s string, err error) {

	// Get corresponding byte slice.
	b, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}

	// Check for Big Endian UTF-16.
	isUTF16BE, err := IsUTF16BE(b)
	if err != nil {
		return
	}

	if isUTF16BE {
		return decodeUTF16String(b)
	}

	// if no acceptable UTF16 encoding found, just return decoded hexstring.
	s = string(b)

	return
}

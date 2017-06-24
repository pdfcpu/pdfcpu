package read

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/pkg/errors"
)

var (
	logDebugUtil *log.Logger
	logInfoUtil  *log.Logger
)

func init() {

	logDebugUtil = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugUtil = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	logInfoUtil = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Dump2Bufs dumps two buffers side by side to logger.
func Dump2Bufs(buf1 []byte, buf2 []byte, lineLength int) {

	logDebugUtil.Println("dump2bufs:")

	for i := 0; i < len(buf1); i += lineLength {
		to := i + lineLength
		logInfoUtil.Printf("%4X(%4d): % X  |  % X\n", i, i, buf1[i:to], buf2[i:to])
	}
}

func positionToNextWhitespace(s string) (int, string) {

	for i, c := range s {
		if unicode.IsSpace(c) {
			return i, s[i:]
		}
	}
	return 0, s
}

// PositionToNextWhitespaceOrChar trims a string to next whitespace or one of given chars.
func positionToNextWhitespaceOrChar(s, chars string) (int, string) {

	if len(chars) == 0 {
		return positionToNextWhitespace(s)
	}

	if len(chars) > 0 {
		for i, c := range s {
			for _, m := range chars {
				if c == m || unicode.IsSpace(c) {
					return i, s[i:]
				}
			}
		}
	}
	return 0, s
}

func positionToNextEOL(s string) string {

	chars := "\x0A\x0D"

	for i, c := range s {
		for _, m := range chars {
			if c == m {
				return s[i:]
			}
		}
	}
	return ""
}

// trimLeftSpace trims leading whitespace and trailing comment.
func trimLeftSpace(s string) (outstr string, trimmedSpaces int) {

	logDebugUtil.Printf("TrimLeftSpace: begin %s\n", s)

	whitespace := func(c rune) bool { return unicode.IsSpace(c) }

	outstr = s
	for {
		// trim leading whitespace
		outstr = strings.TrimLeftFunc(outstr, whitespace)
		logDebugUtil.Printf("1 outstr: <%s>\n", outstr)
		if len(outstr) <= 1 || outstr[0] != '%' {
			break
		}
		// trim PDF comment (= '%' up to eol)
		outstr = positionToNextEOL(outstr)
		logDebugUtil.Printf("2 outstr: <%s>\n", outstr)

	}

	trimmedSpaces = len(s) - len(outstr)

	logDebugUtil.Printf("TrimLeftSpace: end %s %d\n", outstr, trimmedSpaces)

	return
}

// HexString validates and formats a hex string to be of even length.
func hexString(s string) (*string, bool) {

	logDebugUtil.Printf("HexString(%s)\n", s)

	if len(s) == 0 {
		s1 := ""
		return &s1, true
	}

	uc := strings.ToUpper(s)

	for _, c := range uc {
		logDebugUtil.Printf("checking <%c>\n", c)
		isHexChar := false
		for _, hexch := range "ABCDEF1234567890" {
			logDebugUtil.Printf("checking against <%c>\n", hexch)
			if c == hexch {
				isHexChar = true
				break
			}
		}
		if !isHexChar {
			logDebugUtil.Println("isHexStr returning false")
			return nil, false
		}
	}

	logDebugUtil.Println("isHexStr returning true")

	// If the final digit of a hexadecimal string is missing -
	// that is, if there is an odd number of digits - the final digit shall be assumed to be 0.
	if len(uc)%2 == 1 {
		uc = uc + "0"
	}

	return &uc, true
}

// balancedParenthesesPrefix returns the index of the end position of the balanced parentheses prefix of s
// or -1 if unbalanced. s has to start with '('
func balancedParenthesesPrefix(s string) int {

	var j int
	escaped := false

	for i := 0; i < len(s); i++ {

		c := s[i]

		if !escaped && c == '\\' {
			escaped = true
			continue
		}

		if escaped {
			escaped = false
			continue
		}

		if c == '(' {
			j++
		}

		if c == ')' {
			j--
		}

		if j == 0 {
			return i
		}

	}

	return -1
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

// Convert a 1,2 or 3 digit unescaped octal string into the corresponding byte value.
func byteForOctalString(octalBytes []byte) (b byte) {

	var j float64

	for i := len(octalBytes) - 1; i >= 0; i-- {
		b += (octalBytes[i] - '0') * byte(math.Pow(8, j))
		j++
	}

	logDebugUtil.Printf("getByteForOctalString: returning x%x for %v\n", b, octalBytes)

	return
}

// stringLiteral see 7.3.4.2
func stringLiteral(s string) string {

	logDebugUtil.Printf("ParseStringLiteral: begin <%s>\n", s)

	if len(s) == 0 {
		return s
	}

	var b bytes.Buffer
	var octalCode []byte

	escaped := false
	wasCR := false

	for i := 0; i < len(s); i++ {

		c := s[i]

		if !escaped {

			if c == '\\' {
				escaped = true
				wasCR = false
				octalCode = nil
				continue
			}

			// Write \x0d as \x0a.
			if c == '\x0d' {
				if !wasCR {
					wasCR = true
				}
				b.WriteByte('\x0a')
				continue
			}

			// Write \x0a as \x0a.
			// Skip, if 2nd char of eol.
			if c == '\x0a' {
				if wasCR {
					wasCR = false
				} else {
					b.WriteByte('\x0a')
				}
				continue
			}

			b.WriteByte(c)
			wasCR = false
			continue
		}

		// escaped:

		if len(octalCode) == 0 {

			if c == '\x0d' {
				if !wasCR {
					// split line by \\x0d or \\x0d\x0a.
					wasCR = true
				} else {
					// the 2nd of 2 split lines starts with \x0d.
					escaped = false
					wasCR = false
					b.WriteByte('\x0d')
				}
				continue
			}

			if c == '\x0a' {
				// split line by \\x0a or \\x0d\x0a
				escaped = false
				wasCR = false
				continue
			}

			if wasCR {
				// join lines split by \\x0d unless 2nd line starts with '\\'
				if c == '\\' {
					escaped = true
					wasCR = false
					continue
				}
				b.WriteByte(c)
				wasCR = false
				escaped = false
				continue
			}

			if containsByte("01234567", c) {
				// begin octal code escape sequence.
				logDebugUtil.Printf("ParseStringLiteral: recognized octaldigit: %d 0x%x\n", c, c)
				octalCode = append(octalCode, c)
				wasCR = false
				continue
			}

			if containsByte("nrtbf()\\", c) {
				// check against defined escape sequences.
				logDebugUtil.Printf("ParseStringLiteral: recognized escape sequence: \\%c\n", c)
				b.WriteByte('\\')
				b.WriteByte(c)
			} else {
				// Skip '\' for undefined escape sequences.
				logDebugUtil.Printf("ParseStringLiteral: skipping undefined escape sequence: \\%c\n", c)
				b.WriteByte(c)
			}

			escaped = false
			continue
		}

		// in octal code escape sequence: len(octalCode) > 0

		if containsByte("01234567", c) {

			// append to octal code escape sequence.
			logDebugUtil.Printf("ParseStringLiteral: recognized octaldigit: %d 0x%x\n", c, c)
			octalCode = append(octalCode, c)
			if len(octalCode) < 3 {
				wasCR = false
				continue
			}

			// 3 digit octal code sequence completed.
			logDebugUtil.Printf("ParseStringLiteral: recognized escaped octalCode: %s\n", octalCode)
			b.WriteByte(byteForOctalString(octalCode))
			wasCR = false
			escaped = false
			continue

		}

		// 1 or 2 digit octal code sequence completed.
		logDebugUtil.Printf("ParseStringLiteral: recognized escaped octalCode: %s\n", octalCode)
		b.WriteByte(byteForOctalString(octalCode))

		escaped = false

		if c == '\\' {
			escaped = true
			wasCR = false
			octalCode = nil
			continue
		}

		// Write \x0d as \x0a.
		if c == '\x0d' {
			if !wasCR {
				wasCR = true
			}
			b.WriteByte('\x0a')
			continue
		}

		// Write \x0a as \x0a.
		// Skip, if 2nd char of eol.
		if c == '\x0a' {
			if wasCR {
				wasCR = false
			} else {
				b.WriteByte('\x0a')
			}
			continue
		}

		b.WriteByte(c)
		wasCR = false
		octalCode = nil

	}

	logDebugUtil.Printf("ParseStringLiteral: end <%s>\n", b.String())

	return b.String()
}

// getInt interprets the content of buf as an int64.
func getInt(buf []byte) (i int64) {

	for _, b := range buf {
		i <<= 8
		i |= int64(b)
	}

	return
}

// PrefixBigEndian returns true if buf is prefixed with a UTF16 big endian byte order mark.
func PrefixBigEndian(buf []byte) bool {

	if len(buf) <= 2 {
		return false
	}
	return buf[0] == 0xFE && buf[1] == 0xFF
}

// isUTF16BE checks a hex string for Big Endian byte order mark.
func isUTF16BE(hexString string) (ok bool, err error) {

	b, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}

	return PrefixBigEndian(b), nil
}

// decodeUTF16String decodes a UTF16BE string from a hex string.
func decodeUTF16String(hexString string) (s string, err error) {

	logDebugUtil.Println("DecodeUTF16String: begin")

	isUTF16BE, err := isUTF16BE(hexString)
	if err != nil {
		return
	}

	// We only accept big endian byte order.
	if !isUTF16BE {
		err = errors.Errorf("DecodeUTF16String: not UTF16BE: %s\n", hexString)
		return
	}

	// Get a byte slice for this hexString.
	b, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}

	// code points
	u16 := make([]uint16, 0, len(b))

	// Collect code points.
	for i := 0; i < len(b); {

		logDebugUtil.Printf("i=%d\n", i)

		val := (uint16(b[i]) << 8) + uint16(b[i+1])

		if val <= 0xD7FF || val > 0xE000 && val <= 0xFFFF {
			// Basic Multilingual Plane
			logDebugUtil.Println("Basic Multilingual Plane")
			u16 = append(u16, val)
			i += 2
		} else if val >= 0xDC00 && val <= 0xDFFF {
			// Ensure high surrogate is leading in possible surrogate pair.
			err = errors.Errorf("DecodeUTF16String: corrupt UTF16BE on unicode point 1: %s", hexString)
			return
		} else {
			// Supplementary Planes
			logDebugUtil.Println("Supplementary Planes")
			u16 = append(u16, val)
			val = (uint16(b[i+2]) << 8) + uint16(b[i+3])
			if val < 0xDC00 || val > 0xDFFF {
				err = errors.Errorf("DecodeUTF16String: corrupt UTF16BE on unicode point 2: %s", hexString)
				return
			}
			u16 = append(u16, val)
			i += 4
		}

	}

	decb := []byte{}

	utf8Buf := make([]byte, utf8.UTFMax)

	for _, rune := range utf16.Decode(u16) {
		n := utf8.EncodeRune(utf8Buf, rune)
		decb = append(decb, utf8Buf[:n]...)
	}

	s = string(decb)

	logDebugUtil.Println("DecodeUTF16String: end")

	return
}

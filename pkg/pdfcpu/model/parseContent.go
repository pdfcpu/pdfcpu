/*
Copyright 2020 The pdfcpu Authors.

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

package model

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

var (
	errPageContentCorrupt  = errors.New("corrupt page content")
	errTJExpressionCorrupt = errors.New("corrupt TJ expression")
	errBIExpressionCorrupt = errors.New("corrupt BI expression")
)

// whitespaceOrEOL recognizes only PDF whitespace, including EOL markers (see 7.2.3).
func whitespaceOrEOL(c rune) bool {
	return c >= 0 && c < utf8.RuneSelf && pdfWhitespace(byte(c))
}

func whitespaceOrEOLOrClosingBracket(c rune) bool {
	return whitespaceOrEOL(c) || c == ']'
}

func skipDict(l *string) error {
	s := *l
	if !strings.HasPrefix(s, "<<") {
		return ErrDictionaryCorrupt
	}
	s = s[2:]
	depth := 1
	for len(s) > 0 {
		i := strings.IndexAny(s, "<>(%")
		if i < 0 {
			return ErrDictionaryCorrupt
		}
		s = s[i:]
		var err error
		switch s[0] {
		case '(':
			// Delimiters within strings do not affect dictionary nesting (7.3.4).
			err = skipStringLiteral(&s)
		case '%':
			s, _ = positionToNextEOL(s)
		case '<':
			if strings.HasPrefix(s, "<<") {
				depth++
				s = s[2:]
			} else {
				err = skipHexStringLiteral(&s)
			}
		case '>':
			if !strings.HasPrefix(s, ">>") {
				return ErrDictionaryCorrupt
			}
			depth--
			s = s[2:]
			if depth == 0 {
				*l = s
				return nil
			}
		}
		if err != nil {
			return ErrDictionaryCorrupt
		}
	}
	return ErrDictionaryCorrupt
}

func skipStringLiteral(l *string) error {
	// Literal strings may contain balanced, unescaped parentheses (7.3.4.2).
	s := *l
	if !strings.HasPrefix(s, "(") {
		return errStringLiteralCorrupt
	}
	i := balancedParenthesesPrefix(s)
	if i < 0 {
		return errStringLiteralCorrupt
	}
	s = s[i+1:]
	*l = s
	return nil
}

func skipHexStringLiteral(l *string) error {
	s := *l
	i := strings.Index(s, ">")
	if i < 0 {
		return errHexLiteralCorrupt
	}
	s = s[i+1:]
	*l = s
	return nil
}

func skipTJ(l *string) error {
	// Each element shall be either a string or a number.
	s := *l
	for {
		s = strings.TrimLeftFunc(s, whitespaceOrEOL)
		if len(s) == 0 {
			return errTJExpressionCorrupt
		}
		switch s[0] {
		case ']':
			s = s[1:]
			*l = s
			return nil
		case '(':
			if err := skipStringLiteral(&s); err != nil {
				return err
			}
			continue
		case '<':
			if err := skipHexStringLiteral(&s); err != nil {
				return err
			}
			continue
		}
		i, _ := positionToNextWhitespaceOrChar(s, "<(]")
		if i < 0 {
			return errTJExpressionCorrupt
		}
		s = s[i:]
	}
}

// relaxedEIEnd preserves the legacy tolerance for omitted whitespace before EI, as in issue 1080.
func relaxedEIEnd(s string) bool {
	if s == "" {
		return true
	}
	if !whitespaceOrEOL(rune(s[0])) {
		return false
	}
	s = strings.TrimLeftFunc(s, whitespaceOrEOL)
	return s == "" || s[0] == 'Q' && (len(s) == 1 || whitespaceOrEOL(rune(s[1])) || delimiter(s[1]) || s[1] == '%')
}

func inlineImageEnd(s string, i int, relaxed bool) bool {
	if !whitespaceOrEOL(rune(s[i-1])) {
		return relaxed && relaxedEIEnd(s[i+2:])
	}
	return inlineImageEndBoundary(s, i)
}

func inlineImageEndBoundary(s string, i int) bool {
	return i+2 == len(s) || whitespaceOrEOL(rune(s[i+2])) || delimiter(s[i+2]) || s[i+2] == '%'
}

func lookupASCII85EI(s string, relaxed bool) (int, error) {
	// ASCII85 image bytes may contain EI tokens; only ~> terminates encoded data (7.4.3).
	i := strings.Index(s, "~>")
	if i < 0 {
		return 0, errBIExpressionCorrupt
	}
	tail := strings.TrimLeftFunc(s[i+2:], whitespaceOrEOL)
	i = len(s) - len(tail)
	if strings.HasPrefix(tail, "EI") && inlineImageEnd(s, i, relaxed) {
		return i, nil
	}
	return 0, errBIExpressionCorrupt
}

func inlineImageDataStart(s, firstFilter string) (int, error) {
	if len(s) < 3 || !strings.HasPrefix(s, "ID") || !whitespaceOrEOL(rune(s[2])) {
		return 0, errBIExpressionCorrupt
	}
	start := 3
	if s[2] == '\r' && start < len(s) && s[start] == '\n' {
		start++
	}
	if types.MemberOf(firstFilter, []string{"ASCII85Decode", "A85", "ASCIIHexDecode", "AHx"}) {
		for start < len(s) && whitespaceOrEOL(rune(s[start])) {
			start++
		}
	}
	return start, nil
}

func lookupLengthEI(s string, length int, firstFilter string) (int, error) {
	start, err := inlineImageDataStart(s, firstFilter)
	if err != nil {
		return 0, err
	}
	if length > len(s)-start {
		return 0, errBIExpressionCorrupt
	}
	i := start + length
	for i < len(s) && whitespaceOrEOL(rune(s[i])) {
		i++
	}
	if i+2 > len(s) || s[i:i+2] != "EI" || !inlineImageEndBoundary(s, i) {
		return 0, errBIExpressionCorrupt
	}
	return i, nil
}

func lookupEI(l *string, relaxed bool, firstFilter string, length *int) (int, error) {
	s := *l
	if length != nil {
		return lookupLengthEI(s, *length, firstFilter)
	}
	if firstFilter == "ASCII85Decode" || firstFilter == "A85" {
		return lookupASCII85EI(s, relaxed)
	}
	// Any content operator may follow EI (8.9.7). Do not skip ahead to a later EI Q.
	for i := 3; i <= len(s)-2; i++ {
		if s[i:i+2] == "EI" && inlineImageEnd(s, i, relaxed) {
			return i, nil
		}
	}
	return 0, errBIExpressionCorrupt
}

// skipBIColorSpaceArray skips an Indexed colour space and its string lookup table (8.9.7).
func skipBIColorSpaceArray(s string, prn PageResourceNames) (string, error) {
	s = s[1:]
	for {
		s, _ = trimLeftSpace(s, false)
		if len(s) == 0 {
			return "", errBIExpressionCorrupt
		}
		var err error
		switch s[0] {
		case ']':
			return s[1:], nil
		case '(':
			err = skipStringLiteral(&s)
		case '<':
			err = skipHexStringLiteral(&s)
		case '/':
			s, err = skipBIColorSpaceName(s, prn)
		default:
			i, _ := positionToNextWhitespaceOrChar(s, "/[]<>()%")
			if i <= 0 {
				return "", errBIExpressionCorrupt
			}
			s = s[i:]
		}
		if err != nil {
			return "", err
		}
	}
}

func skipBIColorSpaceName(s string, prn PageResourceNames) (string, error) {
	s, _ = trimLeftSpace(s, false)
	if strings.HasPrefix(s, "[") {
		return skipBIColorSpaceArray(s, prn)
	}
	if len(s) < 2 || s[0] != '/' {
		return "", errBIExpressionCorrupt
	}
	s = s[1:]
	i, _ := positionToNextWhitespaceOrChar(s, "/[]<>()%")
	if i < 0 {
		return "", errBIExpressionCorrupt
	}
	name, err := types.DecodeName(s[:i])
	if err != nil {
		return "", errBIExpressionCorrupt
	}
	if !types.MemberOf(name, []string{"DeviceGray", "DeviceRGB", "DeviceCMYK", "Indexed", "G", "RGB", "CMYK", "I"}) {
		prn["ColorSpace"][name] = true
	}
	return s[i:], nil
}

func firstInlineImageFilter(s string) (string, error) {
	s, _ = trimLeftSpace(s, false)
	if strings.HasPrefix(s, "[") {
		s, _ = trimLeftSpace(s[1:], false)
		if strings.HasPrefix(s, "]") {
			return "", nil
		}
	}
	name, err := parseName(&s)
	if err != nil {
		return "", errBIExpressionCorrupt
	}
	return string(*name), nil
}

func inlineImageLength(s string, previous *int) (*int, error) {
	s = strings.TrimLeftFunc(s, whitespaceOrEOL)
	i, _ := positionToNextWhitespaceOrChar(s, "/[]<>()%")
	if i <= 0 {
		return nil, errBIExpressionCorrupt
	}
	n, err := strconv.Atoi(s[:i])
	if err != nil || n < 0 || previous != nil && *previous != n {
		return nil, errBIExpressionCorrupt
	}
	return &n, nil
}

type inlineImageState struct {
	firstFilter string
	length      *int
}

func skipBIEntry(s string, prn PageResourceNames, pdf20 bool, state *inlineImageState) (string, error) {
	s = s[1:]
	i, _ := positionToNextWhitespaceOrChar(s, "/")
	if i < 0 {
		return "", errBIExpressionCorrupt
	}
	switch s[:i] {
	case "L", "Length":
		if pdf20 {
			var err error
			state.length, err = inlineImageLength(s[i:], state.length)
			if err != nil {
				return "", err
			}
		}
	case "CS", "ColorSpace":
		return skipBIColorSpaceName(s[i:], prn)
	case "F", "Filter":
		var err error
		state.firstFilter, err = firstInlineImageFilter(s[i:])
		if err != nil {
			return "", err
		}
	}
	return s[i:], nil
}

func skipBI(l *string, prn PageResourceNames, relaxed, pdf20 bool) error {
	s := *l
	var state inlineImageState
	//fmt.Printf("skipBI <%s>\n", s)
	for {
		s = strings.TrimLeftFunc(s, whitespaceOrEOLOrClosingBracket)
		if len(s) == 0 {
			return errBIExpressionCorrupt
		}
		if len(s) > 2 && strings.HasPrefix(s, "ID") && whitespaceOrEOL(rune(s[2])) {
			if pdf20 && !relaxed && state.length == nil {
				return errBIExpressionCorrupt
			}
			i, err := lookupEI(&s, relaxed, state.firstFilter, state.length)
			if err != nil {
				return err
			}
			s = s[i+2:]
			break
		}
		if s[0] == '/' {
			var err error
			s, err = skipBIEntry(s, prn, pdf20, &state)
			if err != nil {
				return err
			}
			continue
		}
		i, _ := positionToNextWhitespaceOrChar(s, "/]")
		if i < 0 {
			return errBIExpressionCorrupt
		}
		s = s[i:]
	}
	*l = s
	return nil
}

func skipInlineImage(l *string, prn PageResourceNames, relaxed, pdf20 bool) (bool, error) {
	s := *l
	if !strings.HasPrefix(s, "BI") {
		return false, nil
	}
	if len(s) == 2 {
		return false, errBIExpressionCorrupt
	}
	if s[2] != '/' && !whitespaceOrEOL(rune(s[2])) {
		return false, nil
	}
	s = s[2:]
	if err := skipBI(&s, prn, relaxed, pdf20); err != nil {
		return false, err
	}
	*l = s
	return true, nil
}

func positionToNextContentToken(line *string, prn PageResourceNames, relaxed, pdf20 bool) (bool, error) {
	l := *line
	for {
		l = strings.TrimLeftFunc(l, whitespaceOrEOL)
		if len(l) == 0 {
			// whitespace or eol only
			return true, nil
		}
		if l[0] == '%' {
			// Skip comment.
			l, _ = positionToNextEOL(l)
			continue
		}

		if l[0] == '[' {
			// Skip TJ expression:
			// [()...()] TJ
			// [<>...<>] TJ
			if err := skipTJ(&l); err != nil {
				return true, err
			}
			continue
		}
		if l[0] == '(' {
			// Skip text strings as in TJ, Tj, ', " expressions
			if err := skipStringLiteral(&l); err != nil {
				return true, err
			}
			continue
		}
		if l[0] == '<' {
			// Skip hex strings as in TJ, Tj, ', " expressions
			if err := skipHexStringLiteral(&l); err != nil {
				return true, err
			}
			continue
		}
		ok, err := skipInlineImage(&l, prn, relaxed, pdf20)
		if err != nil {
			return true, err
		}
		if ok {
			continue
		}
		*line = l
		return false, nil
	}
}

func nextContentToken(pre string, line *string, prn PageResourceNames, relaxed, pdf20 bool) (string, error) {
	// A token is either a name or some chunk terminated by white space or one of /, (, [
	if noBuf(line) {
		return "", nil
	}
	l := pre + *line
	t := ""

	//log.Parse.Printf("nextContentToken: start buf= <%s>\n", *line)

	// Skip Tj, TJ and inline images.
	done, err := positionToNextContentToken(&l, prn, relaxed, pdf20)
	if err != nil {
		return t, err
	}
	if done {
		return "", nil
	}

	if l[0] == '/' {
		// Cut off at / [ ( < or white space.
		l1 := l[1:]
		i, _ := positionToNextWhitespaceOrChar(l1, "/[(<")
		if i <= 0 {
			*line = ""
			return t, errPageContentCorrupt
		}
		t = l1[:i]
		l1 = l1[i:]
		l1 = strings.TrimLeftFunc(l1, whitespaceOrEOL)
		if !strings.HasPrefix(l1, "<<") {
			// Resource dictionary keys use decoded names (7.3.5).
			t, err = types.DecodeName(t)
			if err != nil {
				return "", errPageContentCorrupt
			}
			t = "/" + t
			*line = l1
			return t, nil
		}
		if err := skipDict(&l1); err != nil {
			return t, err
		}
		*line = l1
		return t, nil
	}

	i, _ := positionToNextWhitespaceOrChar(l, "/[(<")
	if i <= 0 {
		*line = ""
		return l, nil
	}
	t = l[:i]
	l = l[i:]
	if strings.HasPrefix(l, "<<") {
		if err := skipDict(&l); err != nil {
			return t, err
		}
	}
	*line = l
	return t, nil
}

func colorSpace(s, name string, prn PageResourceNames) bool {
	if strings.HasPrefix(s, "cs") || strings.HasPrefix(s, "CS") {
		if !types.MemberOf(name, []string{"DeviceGray", "DeviceRGB", "DeviceCMYK", "Pattern"}) {
			prn["ColorSpace"][name] = true
			if log.ParseEnabled() {
				log.Parse.Printf("ColorSpace[%s]\n", name)
			}
		}
		return true
	}
	return false
}

func resourceNameAtPos1(s, name string, prn PageResourceNames) (string, bool) {
	if colorSpace(s, name, prn) {
		return s[2:], true
	}

	if strings.HasPrefix(s, "gs") {
		prn["ExtGState"][name] = true
		if log.ParseEnabled() {
			log.Parse.Printf("ExtGState[%s]\n", name)
		}
		return s[2:], true
	}

	if strings.HasPrefix(s, "Do") {
		prn["XObject"][name] = true
		if log.ParseEnabled() {
			log.Parse.Printf("XObject[%s]\n", name)
		}
		return s[2:], true
	}

	if strings.HasPrefix(s, "sh") {
		prn["Shading"][name] = true
		if log.ParseEnabled() {
			log.Parse.Printf("Shading[%s]\n", name)
		}
		return s[2:], true
	}

	if strings.HasPrefix(s, "scn") || strings.HasPrefix(s, "SCN") {
		prn["Pattern"][name] = true
		if log.ParseEnabled() {
			log.Parse.Printf("Pattern[%s]\n", name)
		}
		return s[3:], true
	}

	if strings.HasPrefix(s, "ri") || strings.HasPrefix(s, "MP") {
		return s[2:], true
	}

	if strings.HasPrefix(s, "BMC") {
		return s[3:], true
	}

	return "", false
}

func resourceNameAtPos2(s, name string, prn PageResourceNames) (string, bool) {
	switch s {
	case "Tf":
		prn["Font"][name] = true
		if log.ParseEnabled() {
			log.Parse.Printf("Font[%s]\n", name)
		}
		return "", true
	case "BDC", "DP":
		prn["Properties"][name] = true
		if log.ParseEnabled() {
			log.Parse.Printf("Properties[%s]\n", name)
		}
		return "", true
	}
	return "", false
}

func parseContent(s string, relaxed, pdf20 bool) (PageResourceNames, error) {
	var (
		pre  string
		name string
		n    bool
		ok   bool
	)
	prn := NewPageResourceNames()

	//fmt.Printf("parseContent:\n%s\n", hex.Dump([]byte(s)))

	for pos := 0; ; {
		t, err := nextContentToken(pre, &s, prn, relaxed, pdf20)
		pre = ""
		if log.ParseEnabled() {
			log.Parse.Printf("t = <%s>\n", t)
		}
		if err != nil {
			return nil, err
		}
		if t == "" {
			return prn, nil
		}

		if t[0] == '/' {
			name = t[1:]
			if n {
				pos++
			} else {
				n = true
				pos = 0
			}
			if log.ParseEnabled() {
				log.Parse.Printf("name=%s\n", name)
			}
			continue
		}

		if !n {
			if log.ParseEnabled() {
				log.Parse.Printf("skip:%s\n", t)
			}
			continue
		}

		pos++
		if pos == 1 {
			if pre, ok = resourceNameAtPos1(t, name, prn); ok {
				n = false
			}
			continue
		}
		if pos == 2 {
			if pre, ok = resourceNameAtPos2(t, name, prn); ok {
				n = false
			}
			continue
		}
		ShowSkipped("corrupt page content")
		n = false
		if log.ParseEnabled() {
			log.Parse.Printf("skip:%s\n", t)
		}
		//return nil, errPageContentCorrupt
	}
}

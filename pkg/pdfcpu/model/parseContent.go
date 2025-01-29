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
	"strings"
	"unicode"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var (
	errPageContentCorrupt  = errors.New("pdfcpu: corrupt page content")
	errTJExpressionCorrupt = errors.New("pdfcpu: corrupt TJ expression")
	errBIExpressionCorrupt = errors.New("pdfcpu: corrupt BI expression")
)

func whitespaceOrEOL(c rune) bool {
	return unicode.IsSpace(c) || c == 0x0A || c == 0x0D || c == 0x00
}

func skipDict(l *string) error {
	s := *l
	if !strings.HasPrefix(s, "<<") {
		return errDictionaryCorrupt
	}
	s = s[2:]
	j := 0
	for {
		i := strings.IndexAny(s, "<>")
		if i < 0 {
			return errDictionaryCorrupt
		}
		if s[i] == '<' {
			j++
			s = s[i+1:]
			continue
		}
		if s[i] == '>' {
			if j > 0 {
				j--
				s = s[i+1:]
				continue
			}
			// >> ?
			s = s[i:]
			if !strings.HasPrefix(s, ">>") {
				return errDictionaryCorrupt
			}
			*l = s[2:]
			break
		}
	}
	return nil
}

func skipStringLiteral(l *string) error {
	s := *l
	i := 0
	for {
		i = strings.IndexByte(s, byte(')'))
		if i <= 0 || i > 0 && s[i-1] != '\\' {
			break
		}
		k := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			k++
		}
		if k%2 == 0 {
			break
		}
		// Skip \)
		s = s[i+1:]
	}
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
		if s[0] == ']' {
			s = s[1:]
			break
		}
		if s[0] == '(' {
			if err := skipStringLiteral(&s); err != nil {
				return err
			}
		}
		if s[0] == '<' {
			if err := skipHexStringLiteral(&s); err != nil {
				return err
			}
		}
		i, _ := positionToNextWhitespaceOrChar(s, "<(]")
		if i < 0 {
			return errTJExpressionCorrupt
		}
		s = s[i:]
	}
	*l = s
	return nil
}

func skipBI(l *string, prn PageResourceNames) error {
	s := *l
	//fmt.Printf("skipBI <%s>\n", s)
	for {
		s = strings.TrimLeftFunc(s, whitespaceOrEOL)
		if strings.HasPrefix(s, "ID") && whitespaceOrEOL(rune(s[2])) {
			s = s[2:]
			i := strings.Index(s, "EI")
			if i < 0 {
				return errBIExpressionCorrupt
			}
			if i == len(s)-2 {
				break
			}
			i += 2
			if whitespaceOrEOL(rune(s[i])) {
				s = s[i+1:]
				break
			} else {
				return errBIExpressionCorrupt
			}
		}
		if len(s) == 0 {
			return errBIExpressionCorrupt
		}
		if s[0] == '/' {
			s = s[1:]
			i, _ := positionToNextWhitespaceOrChar(s, "/")
			if i < 0 {
				return errBIExpressionCorrupt
			}
			token := s[:i]
			if token == "CS" || token == "ColorSpace" {
				s = s[i:]
				s, _ = trimLeftSpace(s, false)
				s = s[1:]
				i, _ = positionToNextWhitespaceOrChar(s, "/")
				if i < 0 {
					return errBIExpressionCorrupt
				}
				name := s[:i]
				if !types.MemberOf(name, []string{"DeviceGray", "DeviceRGB", "DeviceCMYK", "Indexed", "G", "RGB", "CMYK", "I"}) {
					prn["ColorSpace"][name] = true
				}
			}
			s = s[i:]
			continue
		}
		i, _ := positionToNextWhitespaceOrChar(s, "/")
		if i < 0 {
			return errBIExpressionCorrupt
		}
		s = s[i:]
	}
	*l = s
	return nil
}

func positionToNextContentToken(line *string, prn PageResourceNames) (bool, error) {
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
		if strings.HasPrefix(l, "BI") && (l[2] == '/' || whitespaceOrEOL(rune(l[2]))) {
			// Handle inline image
			l = l[2:]
			if err := skipBI(&l, prn); err != nil {
				return true, err
			}
			continue
		}
		*line = l
		return false, nil
	}
}

func nextContentToken(pre string, line *string, prn PageResourceNames) (string, error) {
	// A token is either a name or some chunk terminated by white space or one of /, (, [
	if noBuf(line) {
		return "", nil
	}
	l := pre + *line
	t := ""

	//log.Parse.Printf("nextContentToken: start buf= <%s>\n", *line)

	// Skip Tj, TJ and inline images.
	done, err := positionToNextContentToken(&l, prn)
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

func parseContent(s string) (PageResourceNames, error) {
	var (
		pre  string
		name string
		n    bool
		ok   bool
	)
	prn := NewPageResourceNames()

	//fmt.Printf("parseContent:\n%s\n", hex.Dump([]byte(s)))

	for pos := 0; ; {
		t, err := nextContentToken(pre, &s, prn)
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
		return nil, errPageContentCorrupt
	}
}

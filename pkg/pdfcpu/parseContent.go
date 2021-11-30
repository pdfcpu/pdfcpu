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

package pdfcpu

import (
	"strings"
	"unicode"

	"github.com/pdfcpu/pdfcpu/pkg/log"
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
		if i <= 0 || i > 0 && s[i-1] != '\\' || i > 1 && s[i-2] == '\\' {
			break
		}
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
	cs := false
	for {
		s = strings.TrimLeftFunc(s, whitespaceOrEOL)
		if strings.HasPrefix(s, "EI") && whitespaceOrEOL(rune(s[2])) {
			s = s[2:]
			break
		}
		// TODO Check len(s) > 0
		if s[0] == '/' {
			s = s[1:]
			i, _ := positionToNextWhitespaceOrChar(s, "/")
			if i < 0 {
				return errBIExpressionCorrupt
			}
			n := s[:i]
			s = s[i:]
			if cs {
				if !MemberOf(n, []string{"RGB", "Gray", "CMYK", "DeviceRGB", "DeviceGray", "DeviceCMYK"}) {
					prn["ColorSpace"][n] = true
				}
				cs = false
				continue
			}
			if n == "CS" {
				cs = true
			}
			continue
		}
		i, _ := positionToNextWhitespaceOrChar(s, "/")
		if i < 0 {
			return errBIExpressionCorrupt
		}
		cs = false
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

func nextContentToken(line *string, prn PageResourceNames) (string, error) {
	// A token is either a name or some chunk terminated by white space or one of /, (, [
	if noBuf(line) {
		return "", nil
	}
	l := *line
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

func resourceNameAtPos1(s, name string, prn PageResourceNames) bool {
	switch s {
	case "cs", "CS":
		if !MemberOf(name, []string{"DeviceGray", "DeviceRGB", "DeviceCMYK", "Pattern"}) {
			prn["ColorSpace"][name] = true
			log.Parse.Printf("ColorSpace[%s]\n", name)
		}
		return true
	case "gs":
		prn["ExtGState"][name] = true
		log.Parse.Printf("ExtGState[%s]\n", name)
		return true
	case "Do":
		prn["XObject"][name] = true
		log.Parse.Printf("XObject[%s]\n", name)
		return true
	case "sh":
		prn["Shading"][name] = true
		log.Parse.Printf("Shading[%s]\n", name)
		return true
	case "scn", "SCN":
		prn["Pattern"][name] = true
		log.Parse.Printf("Pattern[%s]\n", name)
		return true
	case "ri", "BMC", "MP":
		return true
	}
	return false
}

func resourceNameAtPos2(s, name string, prn PageResourceNames) bool {
	switch s {
	case "Tf":
		prn["Font"][name] = true
		log.Parse.Printf("Font[%s]\n", name)
		return true
	case "BDC", "DP":
		prn["Properties"][name] = true
		log.Parse.Printf("Properties[%s]\n", name)
		return true
	}
	return false
}

func parseContent(s string) (PageResourceNames, error) {
	var (
		name string
		n    bool
	)
	prn := NewPageResourceNames()

	for pos := 0; ; {
		t, err := nextContentToken(&s, prn)
		log.Parse.Printf("t = <%s>\n", t)
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
			log.Parse.Printf("name=%s\n", name)
			continue
		}

		if !n {
			log.Parse.Printf("skip:%s\n", t)
			continue
		}

		pos++
		if pos == 1 {
			if resourceNameAtPos1(t, name, prn) {
				n = false
			}
			continue
		}
		if pos == 2 {
			if resourceNameAtPos2(t, name, prn) {
				n = false
			}
			continue
		}
		return nil, errPageContentCorrupt
	}
}

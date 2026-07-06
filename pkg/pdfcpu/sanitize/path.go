/*
Copyright 2026 The pdfcpu Authors.

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

package sanitize

import (
	"errors"
	"strings"
	"unicode"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func logDebugError(s string, err error) error {
	if log.DebugEnabled() {
		log.Debug.Printf("%v: %q\n", err, s)
	}
	return err
}

func pathPart(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if r < 0x20 || strings.ContainsRune(`<>:"|?*`, r) || unicode.IsControl(r) {
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
			continue
		}
		b.WriteRune(r)
		lastUnderscore = r == '_'
	}

	s = strings.Trim(b.String(), " .")
	if s == "" {
		return ""
	}

	stem := s
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	// Windows-only guardrail.
	switch strings.ToUpper(stem) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		s = "_" + s
	}

	return s
}

// Path returns a filesystem-safe, relative filename for an untrusted path.
func Path(s string) (string, error) {
	orig := s
	if strings.ContainsRune(s, 0) {
		return "", logDebugError(orig, errors.New("path contains NUL byte"))
	}

	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[1] == ':' {
		s = s[2:]
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '/'
	})

	cleanParts := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			continue
		}

		if part = pathPart(part); part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	if len(cleanParts) == 0 {
		return "", logDebugError(orig, errors.New("path contains no usable filename"))
	}

	return strings.Join(cleanParts, "_"), nil
}

// PathOr returns fallback if Path rejects s.
func PathOr(s, fallback string) string {
	s, err := Path(s)
	if err != nil {
		return fallback
	}
	return s
}

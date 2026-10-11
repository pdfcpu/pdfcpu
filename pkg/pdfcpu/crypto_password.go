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

package pdfcpu

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/xdg-go/stringprep"
	"golang.org/x/text/unicode/norm"
)

// Mapping and output checks are separate so normalization can use Unicode 3.2 rather than the current x/text version.
var passwordMappingProfile = stringprep.Profile{
	// stringprep v1.0.4 omits U+1806 from RFC 3454 table B.1.
	Mappings: append([]stringprep.Mapping{{0x1806: nil}}, stringprep.SASLprep.Mappings...),
}

// RFC 3454 permits unassigned code points in queries, but prohibits them in stored strings.
var passwordQueryProfile = stringprep.Profile{
	Prohibits: []stringprep.Set{
		stringprep.TableC1_2, stringprep.TableC2_1, stringprep.TableC2_2, stringprep.TableC3,
		stringprep.TableC4, stringprep.TableC5, stringprep.TableC6, stringprep.TableC7,
		stringprep.TableC8, stringprep.TableC9,
	},
	CheckBiDi: true,
}

var passwordStoredProfile = stringprep.Profile{Prohibits: stringprep.SASLprep.Prohibits, CheckBiDi: true}

// Unicode NormalizationCorrections.txt records these five decompositions as corrected after Unicode 3.2.
var passwordNormalizationCorrections = strings.NewReplacer(
	"\U0002f868", "\U0002136a",
	"\U0002f874", "\u5f33",
	"\U0002f91f", "\u43ab",
	"\U0002f95f", "\u7aae",
	"\U0002f9bf", "\u4d57",
)

func normalizePasswordUnicode32(input string) string {
	input = passwordNormalizationCorrections.Replace(input)
	var b strings.Builder
	start := 0
	for i, r := range input {
		if stringprep.TableA1.Contains(r) {
			// A character unassigned in Unicode 3.2 is unchanged and separates normalization segments.
			b.WriteString(normalizePasswordSegment(input[start:i]))
			b.WriteRune(r)
			start = i + utf8.RuneLen(r)
		}
	}
	b.WriteString(normalizePasswordSegment(input[start:]))
	return b.String()
}

type passwordRune struct {
	r   rune
	ccc uint8
}

func normalizePasswordSegment(input string) string {
	output := norm.NFKC.String(input)
	if !strings.ContainsRune(output, '\u034f') {
		return output
	}
	// x/text inserts stream-safe CGJs after 30 nonstarters. SASLprep maps original CGJs to nothing,
	// so any CGJ here was inserted by normalization and must not become part of the password.
	decomposed := strings.ReplaceAll(norm.NFKD.String(input), "\u034f", "")
	var rr []passwordRune
	start := 0
	for _, r := range decomposed {
		ccc := norm.NFD.PropertiesString(string(r)).CCC()
		if ccc == 0 {
			orderPasswordRunes(rr[start:])
			start = len(rr) + 1
		}
		rr = append(rr, passwordRune{r: r, ccc: ccc})
	}
	orderPasswordRunes(rr[start:])
	return composePasswordRunes(rr)
}

func orderPasswordRunes(rr []passwordRune) {
	slices.SortStableFunc(rr, func(a, b passwordRune) int { return cmp.Compare(a.ccc, b.ccc) })
}

func composePasswordRunes(rr []passwordRune) string {
	var output []rune
	starter := -1
	var lastCCC uint8
	for _, p := range rr {
		if starter >= 0 && (lastCCC == 0 || lastCCC < p.ccc) {
			pair := norm.NFC.String(string([]rune{output[starter], p.r}))
			if utf8.RuneCountInString(pair) == 1 {
				output[starter], _ = utf8.DecodeRuneInString(pair)
				continue
			}
		}
		if p.ccc == 0 {
			starter = len(output)
		}
		lastCCC = p.ccc
		output = append(output, p.r)
	}
	return string(output)
}

// processInput applies SASLprep using the Unicode 3.2 repertoire specified by RFC 4013.
func processInput(input string, stored bool) ([]byte, error) {
	if !utf8.ValidString(input) {
		return nil, errors.New("pdfcpu: SASLprep: invalid UTF-8")
	}
	mapped, err := passwordMappingProfile.Prepare(input)
	if err != nil {
		return nil, err
	}
	profile := passwordQueryProfile
	if stored {
		profile = passwordStoredProfile
	}
	output, err := profile.Prepare(normalizePasswordUnicode32(mapped))
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

// preparePassword applies SASLprep and the 127-byte limit before hashing AES-256 passwords.
// See ISO 32000-2 7.6.4.3.3, Algorithm 2.A, steps a and b; R5 uses Adobe extension 3, Algorithm 3.2a.
func preparePassword(input string, stored bool) ([]byte, error) {
	b, err := processInput(input, stored)
	if err != nil {
		return nil, err
	}
	if len(b) > 127 {
		b = b[:127]
	}
	// Hashing appends salts repeatedly; prevent those appends from changing the password's backing array.
	return slices.Clip(b), nil
}

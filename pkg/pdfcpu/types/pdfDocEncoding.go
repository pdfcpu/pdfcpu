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

package types

import "unicode"

// See ISO 32000-1:2008, Table D.2, PDFDocEncoding.
var pdfDocEncoding = map[byte]rune{
	0x18: 0x02d8,
	0x19: 0x02c7,
	0x1a: 0x02c6,
	0x1b: 0x02d9,
	0x1c: 0x02dd,
	0x1d: 0x02db,
	0x1e: 0x02da,
	0x1f: 0x02dc,
	0x80: 0x2022,
	0x81: 0x2020,
	0x82: 0x2021,
	0x83: 0x2026,
	0x84: 0x2014,
	0x85: 0x2013,
	0x86: 0x0192,
	0x87: 0x2044,
	0x88: 0x2039,
	0x89: 0x203a,
	0x8a: 0x2212,
	0x8b: 0x2030,
	0x8c: 0x201e,
	0x8d: 0x201c,
	0x8e: 0x201d,
	0x8f: 0x2018,
	0x90: 0x2019,
	0x91: 0x201a,
	0x92: 0x2122,
	0x93: 0xfb01,
	0x94: 0xfb02,
	0x95: 0x0141,
	0x96: 0x0152,
	0x97: 0x0160,
	0x98: 0x0178,
	0x99: 0x017d,
	0x9a: 0x0131,
	0x9b: 0x0142,
	0x9c: 0x0153,
	0x9d: 0x0161,
	0x9e: 0x017e,
	0xa0: 0x20ac,
}

func pdfDocEncodingRune(b byte) rune {
	if r, ok := pdfDocEncoding[b]; ok {
		return r
	}
	if b == 0x09 || b == 0x0a || b == 0x0d || b >= 0x20 && b <= 0x7e || b >= 0xa1 && b <= 0xff && b != 0xad {
		return rune(b)
	}
	return unicode.ReplacementChar
}

func decodePDFDocEncoding(b []byte) string {
	rr := make([]rune, len(b))
	for i, c := range b {
		rr[i] = pdfDocEncodingRune(c)
	}
	return string(rr)
}

func hasPDFDocEncodingControlByte(b []byte) bool {
	for _, c := range b {
		if c >= 0x18 && c <= 0x1f {
			return true
		}
	}
	return false
}

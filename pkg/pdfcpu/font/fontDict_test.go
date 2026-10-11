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

package font

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	corefont "github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestNameRejectsMissingSubtype(t *testing.T) {
	for _, subtype := range []types.Object{nil, types.Name("")} {
		fontDict := types.Dict{
			"BaseFont": types.Name("Helvetica"),
		}
		if subtype != nil {
			fontDict["Subtype"] = subtype
		}

		_, _, err := Name(nil, fontDict, 7)
		if err == nil || err.Error() != `fontName: missing fontDict entry "Subtype"` {
			t.Fatalf("got %v, want missing Subtype error", err)
		}
	}
}

func TestFontDescriptorRejectsDescendantFontMissingType(t *testing.T) {
	fontDict := types.Dict{
		"DescendantFonts": types.Array{types.Dict{}},
	}

	_, err := FontDescriptor(&model.XRefTable{}, fontDict, 7)
	if err == nil || !strings.Contains(err.Error(), "descendant font dict missing Type for font object 7") {
		t.Fatalf("got %v, want missing descendant font Type error", err)
	}
}

func usedGIDsFromCMapWithoutPanic(t *testing.T, cMap string) (gids []uint16, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("usedGIDsFromCMap panicked: %v", r)
		}
	}()
	return usedGIDsFromCMap(cMap)
}

func TestUsedGIDsFromCMap(t *testing.T) {
	cMap := "endcodespacerange\r\n2 beginbfchar\n<0001> <0041>\n<00FF> <0042>\nendbfchar\nendcmap"
	gids, err := usedGIDsFromCMap(cMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(gids) != 2 || gids[0] != 1 || gids[1] != 255 {
		t.Fatalf("expected [1 255], got %v", gids)
	}
}

func TestUsedGIDsFromCMapRejectsMalformedInputWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing codespace end", body: "1 beginbfchar"},
		{name: "header token count", body: "endcodespacerange\n1 beginbfchar extra"},
		{name: "header operator", body: "endcodespacerange\n1 beginbfrange"},
		{name: "header count", body: "endcodespacerange\nno beginbfchar"},
		{name: "negative count", body: "endcodespacerange\n-1 beginbfchar"},
		{name: "mapping token count", body: "endcodespacerange\n1 beginbfchar\n<0001>\nendbfchar\nendcmap"},
		{name: "short source", body: "endcodespacerange\n1 beginbfchar\n<01> <0041>\nendbfchar\nendcmap"},
		{name: "source boundary", body: "endcodespacerange\n1 beginbfchar\n0001> <0041>\nendbfchar\nendcmap"},
		{name: "source hex", body: "endcodespacerange\n1 beginbfchar\n<ZZZZ> <0041>\nendbfchar\nendcmap"},
		{name: "destination boundary", body: "endcodespacerange\n1 beginbfchar\n<0001> 0041>\nendbfchar\nendcmap"},
		{name: "destination hex", body: "endcodespacerange\n1 beginbfchar\n<0001> <ZZZZ>\nendbfchar\nendcmap"},
		{name: "bad endbfchar tokens", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>\nendbfchar extra\nendcmap"},
		{name: "unexpected block", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>\nendbfchar\n1 beginbfchar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := usedGIDsFromCMapWithoutPanic(t, tt.body); !errors.Is(err, errCorruptCMap) {
				t.Fatalf("expected %v, got %v", errCorruptCMap, err)
			}
		})
	}
}

func TestUsedGIDsFromCMapRejectsTruncatedInputWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "after codespace", body: "endcodespacerange"},
		{name: "after header", body: "endcodespacerange\n1 beginbfchar"},
		{name: "within mappings", body: "endcodespacerange\n2 beginbfchar\n<0001> <0041>"},
		{name: "before endbfchar", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>"},
		{name: "before endcmap", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>\nendbfchar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := usedGIDsFromCMapWithoutPanic(t, tt.body); !errors.Is(err, errCorruptCMap) {
				t.Fatalf("expected %v, got %v", errCorruptCMap, err)
			}
		})
	}
}

func TestUsedGIDsFromCMapPreservesScannerFailure(t *testing.T) {
	cMap := "endcodespacerange\n" + strings.Repeat("1", 70_000)
	_, err := usedGIDsFromCMapWithoutPanic(t, cMap)
	if !errors.Is(err, errCorruptCMap) {
		t.Fatalf("expected %v, got %v", errCorruptCMap, err)
	}
	if !strings.Contains(err.Error(), "read beginbfchar header") {
		t.Fatalf("expected scan phase in %q", err.Error())
	}
}

// TestToUnicodeBFCharBlocks preserves every mapping across block boundaries for full fonts and subsets.
func TestToUnicodeBFCharBlocks(t *testing.T) {
	for _, n := range []int{0, 1, 99, 100, 101, 199, 200, 201, 250, 300} {
		for _, subset := range []bool{false, true} {
			t.Run(fmt.Sprintf("count=%d/subset=%t", n, subset), func(t *testing.T) {
				ttf := corefont.TTFLight{Chars: map[uint32]uint16{}, ToUnicode: map[uint16]uint32{}}
				used := map[uint16]bool{}
				want := make([]string, n)
				for i := n - 1; i >= 0; i-- {
					gid := uint16(2*i + 3)
					u := uint32(0x40 + i)
					dst := fmt.Sprintf("%04X", u)
					if i == n-1 {
						u, dst = 0x1f600, "D83DDE00"
					}
					ttf.Chars[u] = gid
					ttf.ToUnicode[gid] = u
					used[gid] = true
					want[i] = fmt.Sprintf("<%04X> <%s>", gid, dst)
				}
				if subset {
					// An unused glyph in the font must not leak into the subset CMap.
					ttf.Chars[0x3000], ttf.ToUnicode[65535] = 65535, 0x3000
				} else {
					// Full-font mappings must not depend on the subset selection.
					used = nil
				}
				var b bytes.Buffer
				bf(&b, ttf, used, subset)
				checkBFCharBlocks(t, b.String(), want)
				gids, err := usedGIDsFromCMap("endcodespacerange\n" + b.String() + "endcmap\n")
				if err != nil || len(gids) != n {
					t.Fatalf("read generated CMap: got %d glyphs, error %v; want %d", len(gids), err, n)
				}
			})
		}
	}
}

func checkBFCharBlocks(t *testing.T, content string, want []string) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	mappings, blocks := 0, 0
	for len(lines) > 0 {
		header := strings.Fields(lines[0])
		if len(header) != 2 || header[1] != "beginbfchar" {
			t.Fatalf("invalid block header: %q", lines[0])
		}
		n, err := strconv.Atoi(header[0])
		if err != nil || n < 0 || n > 100 || len(lines) < n+2 {
			t.Fatalf("invalid block size: %q", lines[0])
		}
		if mappings+n > len(want) {
			t.Fatal("extra mappings")
		}
		for i, line := range lines[1 : n+1] {
			if line != want[mappings+i] {
				t.Fatalf("mapping %d: got %q, want %q", mappings+i, line, want[mappings+i])
			}
		}
		if lines[n+1] != "endbfchar" {
			t.Fatalf("invalid block end: %q", lines[n+1])
		}
		mappings += n
		blocks++
		lines = lines[n+2:]
	}
	if mappings != len(want) || blocks != max(1, (len(want)+99)/100) {
		t.Fatalf("got %d mappings in %d blocks for %d glyphs", mappings, blocks, len(want))
	}
}

// TestWidthsUsesWinAnsiGlyphs checks the glyph selected by each high WinAnsi code and its aliases.
func TestWidthsUsesWinAnsiGlyphs(t *testing.T) {
	// Annex D.2, Table D.1: codes 0x80 through 0x9f, including unused-code bullet aliases.
	high := []uint32{
		0x20ac, 0x2022, 0x201a, 0x0192, 0x201e, 0x2026, 0x2020, 0x2021,
		0x02c6, 0x2030, 0x0160, 0x2039, 0x0152, 0x2022, 0x017d, 0x2022,
		0x2022, 0x2018, 0x2019, 0x201c, 0x201d, 0x2022, 0x2013, 0x2014,
		0x02dc, 0x2122, 0x0161, 0x203a, 0x0153, 0x2022, 0x017e, 0x0178,
	}
	cases := []struct {
		code    int
		unicode uint32
	}{
		{0x41, 0x41}, {0xe4, 0xe4}, {0x7f, 0x2022}, {0xa0, 0x20}, {0xad, 0x2d},
		{0x20ac, 0x20ac}, // Preserve the existing behavior for calls outside the byte range.
	}
	for i, u := range high {
		cases = append(cases, struct {
			code    int
			unicode uint32
		}{0x80 + i, u})
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%X", tt.code), func(t *testing.T) {
			ttf := validTTFLight()
			ttf.GlyphCount, ttf.HorMetricsCount = 3, 3
			ttf.GlyphWidths = []int{500, 600, 900}
			ttf.Chars = map[uint32]uint16{uint32(tt.code): 2}
			ttf.Chars[tt.unicode] = 1
			ttf.LastChar = 0x20ac
			a := testTrueTypeWidths(t, ttf, tt.code, tt.code)
			if len(a) != 1 || a[0] != types.Integer(600) {
				t.Fatalf("widths for code %X = %v, want [600] for U+%04X", tt.code, a, tt.unicode)
			}
		})
	}
}

// TestWidthsWinAnsiRangeAndMissingGlyph preserves code positions and the missing-glyph fallback.
func TestWidthsWinAnsiRangeAndMissingGlyph(t *testing.T) {
	ttf := validTTFLight()
	ttf.Chars[0x20ac] = 1
	ttf.Chars[0x0178] = 1
	a := testTrueTypeWidths(t, ttf, 0x80, 0x9f)
	if len(a) != 32 {
		t.Fatalf("got %d widths, want 32", len(a))
	}
	for i, w := range a {
		want := types.Integer(500)
		if i == 0 || i == 31 {
			want = 600
		}
		if w != want {
			t.Fatalf("width for code %X = %v, want %v", 0x80+i, w, want)
		}
	}
	delete(ttf.Chars, 0x20ac)
	ttf.Chars[0x80] = 1 // A control-code glyph must not substitute for a missing Euro glyph.
	a = testTrueTypeWidths(t, ttf, 0x80, 0x80)
	if a[0] != types.Integer(500) {
		t.Fatalf("missing Euro width = %v, want 500", a[0])
	}
}

func testTrueTypeWidths(t *testing.T, ttf corefont.TTFLight, first, last int) types.Array {
	t.Helper()
	size := 1
	x := &model.XRefTable{
		Table: map[int]*model.XRefTableEntry{0: model.NewFreeHeadXRefTableEntry()},
		Size:  &size,
	}
	ir, err := Widths(x, ttf, first, last)
	if err != nil {
		t.Fatal(err)
	}
	a, err := x.DereferenceArray(*ir)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

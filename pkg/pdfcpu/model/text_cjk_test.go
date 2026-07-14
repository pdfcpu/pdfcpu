/*
Copyright 2025 The pdfcpu Authors.

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
	"reflect"
	"testing"
)

// CJK line-breaking rules (kinsoku shori / 避头尾):
//
// 1. Opening punctuation (（, 「, 『, (, [, {, etc.) must NOT appear at end of line.
// 2. Closing punctuation (，, 。, ）, 」, !, ?, etc.) must NOT appear at start of line.
// 3. CJK ideographs, hiragana, katakana, and hangul allow breaks both before and after.
// 4. ASCII letters and digits do NOT allow CJK-style breaks (only break at spaces).
// 5. Fullwidth opening forms must NOT end a line; fullwidth closing forms must NOT start a line.

// TestIsOpeningPunct verifies classification of opening punctuation marks
// that must NOT appear at the end of a line (kinsoku shori / 避头尾 rule 1 & 5).
// Covers: CJK bracket pairs, fullwidth opening forms, and Unicode quotation marks.
func TestIsOpeningPunct(t *testing.T) {
	testcases := []struct {
		char rune
		want bool
		desc string
	}{
		{0x3008, true, "〈 left angle bracket"},
		{0x300A, true, "《 left double angle bracket"},
		{0x300C, true, "「 left corner bracket"},
		{0x300E, true, "『 left white corner bracket"},
		{0x3010, true, "【 left lenticular bracket"},
		{0x3014, true, "〔 left tortoise shell bracket"},
		{0x3016, true, "〖 left hollow lenticular bracket"},
		{0x3018, true, "〘 left hollow tortoise shell bracket"},
		{0x301A, true, "〚 left hollow square bracket"},
		{0x301D, true, "〝 reversed double prime quotation mark"},
		{0xFF02, true, "＿ fullwidth quotation mark"},
		{0xFF07, true, "＿ fullwidth apostrophe"},
		{0xFF08, true, "（ fullwidth left paren"},
		{0xFF3B, true, "［ fullwidth left square bracket"},
		{0xFF40, true, "｀ fullwidth grave accent"},
		{0xFF5B, true, "｛ fullwidth left brace"},
		{0x2018, true, "‘ left single quotation mark"},
		{0x201A, true, "‚ single low-9 quotation mark"},
		{0x201C, true, "“ left double quotation mark"},
		{0x201E, true, "„ double low-9 quotation mark"},
		{0x2039, true, "‹ single left-pointing angle quotation mark"},
		{0x00AB, true, "« left-pointing double angle quotation mark"},
		{0x3001, false, "、 ideographic comma is NOT opening"},
		{0x3002, false, "。 ideographic period is NOT opening"},
		{0x3009, false, "〉 right angle bracket is NOT opening"},
		{0x300D, false, "」 right corner bracket is NOT opening"},
		{'天', false, "CJK ideograph is NOT opening"},
		{'A', false, "ASCII letter is NOT opening"},
	}

	for _, tc := range testcases {
		got := isOpeningPunct(tc.char)
		if got != tc.want {
			t.Errorf("isOpeningPunct(%U) [%s]: got %t, want %t", tc.char, tc.desc, got, tc.want)
		}
	}
}

// TestIsClosingPunct verifies classification of closing punctuation marks
// that must NOT appear at the start of a line (kinsoku shori / 避头尾 rule 2 & 5).
// Covers: CJK commas/periods, closing brackets, fullwidth exclamation/question,
// em/en dash, ellipsis, and Unicode closing quotation marks.
func TestIsClosingPunct(t *testing.T) {
	testcases := []struct {
		char rune
		want bool
		desc string
	}{
		{0x3001, true, "、 ideographic comma"},
		{0x3002, true, "。 ideographic period"},
		{0x3009, true, "〉 right angle bracket"},
		{0x300B, true, "》 right double angle bracket"},
		{0x300D, true, "」 right corner bracket"},
		{0x300F, true, "』 right white corner bracket"},
		{0x3011, true, "】 right lenticular bracket"},
		{0x3015, true, "〗 right hollow lenticular bracket"},
		{0x3017, true, "〙 right hollow tortoise shell bracket"},
		{0x3019, true, "〛 right hollow square bracket"},
		{0x301B, true, "〟 low right corner bracket"},
		{0x301E, true, "〞 double right reversed quotation mark"},
		{0x301F, true, "〟 low right double reversed quotation mark"},
		{0xFF01, true, "！ fullwidth exclamation"},
		{0xFF02, true, "＿ fullwidth quotation mark"},
		{0xFF07, true, "＿ fullwidth apostrophe"},
		{0xFF09, true, "） fullwidth right paren"},
		{0xFF0C, true, "， fullwidth comma"},
		{0xFF0E, true, "． fullwidth period"},
		{0xFF1A, true, "： fullwidth colon"},
		{0xFF1B, true, "； fullwidth semicolon"},
		{0xFF1F, true, "？ fullwidth question mark"},
		{0xFF3D, true, "］ fullwidth right square bracket"},
		{0xFF5D, true, "｝ fullwidth right brace"},
		{0x2013, true, "– en dash"},
		{0x2014, true, "— em dash"},
		{0x2019, true, "’ right single quotation mark"},
		{0x201D, true, "” right double quotation mark"},
		{0x2026, true, "… horizontal ellipsis"},
		{0x203A, true, "› single right-pointing angle quotation mark"},
		{0x00BB, true, "» right-pointing double angle quotation mark"},
		{0x0022, true, "\" ASCII straight double quote"},
		{0x0027, true, "' ASCII straight single quote"},
		{0x3008, false, "〈 left angle bracket is NOT closing"},
		{0x300C, false, "「 left corner bracket is NOT closing"},
		{0xFF08, false, "（ fullwidth left paren is NOT closing"},
		{'天', false, "CJK ideograph is NOT closing"},
		{'A', false, "ASCII letter is NOT closing"},
	}

	for _, tc := range testcases {
		got := isClosingPunct(tc.char)
		if got != tc.want {
			t.Errorf("isClosingPunct(%U) [%s]: got %t, want %t", tc.char, tc.desc, got, tc.want)
		}
	}
}

// TestCanBreakAfterChar verifies which characters allow a line break immediately
// after them. Covers: CJK ideographs, hiragana, katakana, hangul (rule 3),
// closing punctuation (rule 2), fullwidth closing forms (rule 5),
// and confirms ASCII letters/digits/spaces do NOT allow CJK-style breaks (rule 4).
func TestCanBreakAfterChar(t *testing.T) {
	testcases := []struct {
		char rune
		want bool
		desc string
	}{
		{'天', true, "CJK Unified Ideograph"},
		{'中', true, "CJK Unified Ideograph"},
		{'あ', true, "Hiragana"},
		{'い', true, "Hiragana"},
		{'ア', true, "Katakana"},
		{'カ', true, "Katakana"},
		{'한', true, "Hangul Syllable"},
		{'글', true, "Hangul Syllable"},
		{'，', true, "CJK comma — can end a line"},
		{'。', true, "CJK period — can end a line"},
		{'）', true, "closing paren — can end a line"},
		{'」', true, "closing corner bracket — can end a line"},
		{'（', false, "opening paren — must NOT end a line"},
		{'「', false, "opening corner bracket — must NOT end a line"},
		{'『', false, "opening white corner bracket — must NOT end a line"},
		{'【', false, "opening lenticular bracket — must NOT end a line"},
		{'Ａ', true, "Fullwidth A — can end a line"},
		{'（', false, "Fullwidth opening paren — must NOT end a line"},
		{'［', false, "Fullwidth opening bracket — must NOT end a line"},
		{'｛', false, "Fullwidth opening brace — must NOT end a line"},
		{'）', true, "Fullwidth closing paren — can end a line"},
		{'］', true, "Fullwidth closing bracket — can end a line"},
		{'A', false, "Basic Latin"},
		{'z', false, "Basic Latin"},
		{'0', false, "ASCII digit"},
		{' ', false, "Space"},
		{'\n', false, "Newline"},
	}

	for _, tc := range testcases {
		got := canBreakAfterChar(tc.char)
		if got != tc.want {
			t.Errorf("canBreakAfterChar(%U) [%s]: got %t, want %t", tc.char, tc.desc, got, tc.want)
		}
	}
}

// TestCanBreakBeforeChar verifies which characters allow a line break immediately
// before them. Covers: CJK ideographs, hiragana, katakana, hangul (rule 3),
// opening punctuation allowed at line start (rule 1), fullwidth closing forms
// that must NOT start a line (rule 5), and confirms ASCII letters/spaces do NOT
// allow CJK-style breaks (rule 4).
func TestCanBreakBeforeChar(t *testing.T) {
	testcases := []struct {
		char rune
		want bool
		desc string
	}{
		{'天', true, "CJK Unified Ideograph"},
		{'中', true, "CJK Unified Ideograph"},
		{'あ', true, "Hiragana"},
		{'ア', true, "Katakana"},
		{'한', true, "Hangul Syllable"},
		{'，', false, "CJK comma — must NOT start a line"},
		{'。', false, "CJK period — must NOT start a line"},
		{'、', false, "CJK ideographic comma — must NOT start a line"},
		{'）', false, "closing paren — must NOT start a line"},
		{'」', false, "closing corner bracket — must NOT start a line"},
		{'！', false, "CJK exclamation — must NOT start a line"},
		{'？', false, "CJK question — must NOT start a line"},
		{'（', true, "opening paren — allowed at line start"},
		{'「', true, "opening corner bracket — allowed at line start"},
		{'Ａ', true, "Fullwidth A"},
		{'（', true, "Fullwidth opening paren — allowed at line start"},
		{'）', false, "Fullwidth closing paren — must NOT start a line"},
		{'，', false, "Fullwidth comma — must NOT start a line"},
		{'．', false, "Fullwidth period — must NOT start a line"},
		{'！', false, "Fullwidth exclamation — must NOT start a line"},
		{'？', false, "Fullwidth question — must NOT start a line"},
		{'A', false, "Basic Latin"},
		{' ', false, "Space"},
	}

	for _, tc := range testcases {
		got := canBreakBeforeChar(tc.char)
		if got != tc.want {
			t.Errorf("canBreakBeforeChar(%U) [%s]: got %t, want %t", tc.char, tc.desc, got, tc.want)
		}
	}
}

// All tests verify exact line content using reflect.DeepEqual.

// TestWordWrap verifies the end-to-end word wrapping behaviour with CJK text.
// Covers: pure Chinese wrapping at character boundaries, narrow-width forced
// breaks, Japanese kanji/hiragana mix, Korean hangul with space, pure ASCII
// space-based wrapping, and mixed CJK/Latin text with spaces.
func TestWordWrapCJK(t *testing.T) {
	testcases := []struct {
		FontName       string
		FontSize       int
		MaxWidthPoints float64
		Text           string
		WantLines      []string
		Desc           string
	}{
		{"Helvetica", 12, 100,
			"天地玄黄宇宙洪荒日月盈昃辰宿列张",
			[]string{"天地玄黄", "宇宙洪荒", "日月盈昃", "辰宿列张"},
			"Chinese: pure CJK wraps into exact lines",
		},
		{"Helvetica", 12, 50,
			"天地玄黄宇宙洪荒",
			[]string{"天地", "玄黄", "宇宙", "洪荒"},
			"Chinese: narrow width forces more line breaks",
		},
		{"Helvetica", 12, 500,
			"天地玄黄",
			[]string{"天地玄黄"},
			"Chinese: short text fits in one line",
		},
		{"Helvetica", 12, 100,
			"僕生まれ育つ此郷里山",
			[]string{"僕生まれ育", "つ此郷里", "山"},
			"Japanese: Kanji/Hiragana mix should wrap",
		},
		{"Helvetica", 12, 100,
			"안녕하세요 세계",
			[]string{"안녕하세요 세계"},
			"Korean: hangs with space fits in one line",
		},
		{"Helvetica", 12, 100,
			"Hello World",
			[]string{"Hello World"},
			"ASCII: text still wraps at spaces normally",
		},
		{"Helvetica", 12, 100,
			"Hello 世界 Test 測試",
			[]string{"Hello 世界 Test", "測試"},
			"Mixed CJK/Latin: wraps at spaces, CJK breaks at boundaries",
		},
	}

	for _, tc := range testcases {
		gotLines := WordWrap(tc.Text, tc.FontName, tc.FontSize, tc.MaxWidthPoints)
		if !reflect.DeepEqual(gotLines, tc.WantLines) {
			t.Errorf("[%s]:\n  got:  %v\n  want: %v", tc.Desc, gotLines, tc.WantLines)
		}
	}
}

// All tests verify exact line content using reflect.DeepEqual.

// TestWordWrapPunctuationBoundary verifies that WordWrap respects CJK
// line-breaking rules (kinsoku shori / 避头尾) at punctuation boundaries.
// Rather than asserting exact line content (which is font-width dependent),
// this test validates the structural rules:
//   - No line starts with closing punctuation (，。）」！？ etc.)
//   - No line ends with opening punctuation （（「『【 etc.)
//
// Covers Chinese, Japanese, and Korean text with various punctuation types.
func TestWordWrapPunctuationBoundary(t *testing.T) {
	testcases := []struct {
		FontName       string
		FontSize       int
		MaxWidthPoints float64
		Text           string
		Desc           string
	}{
		{"Helvetica", 12, 100, "天地玄黄，宇宙洪荒。日月盈昃，辰宿列张。", "Chinese: closing punctuation (，。)"},
		{"Helvetica", 12, 100, "こんにちは、世界。テスト！", "Japanese: closing punctuation (、。！)"},
		{"Helvetica", 12, 100, "안녕하세요. 세계. 테스트.", "Korean: period after hangul"},
		{"Helvetica", 12, 100, "天地（玄黄）宇宙（洪荒）日月（盈昃）", "Chinese: opening paren (（) must not end line"},
		{"Helvetica", 12, 100, "テスト（カタカナ）世界（漢字）テスト", "Japanese: opening paren (（) must not end line"},
		{"Helvetica", 12, 100, "天地「玄黄」宇宙「日月」盈昃", "Chinese: opening bracket (「) must not end line"},
	}

	for _, tc := range testcases {
		lines := WordWrap(tc.Text, tc.FontName, tc.FontSize, tc.MaxWidthPoints)
		if len(lines) < 2 {
			t.Errorf("[%s]: expected wrapping into multiple lines, got %d line(s): %v", tc.Desc, len(lines), lines)
			continue
		}
		for i, line := range lines {
			if len(line) == 0 {
				continue
			}
			first := []rune(line)[0]
			last := []rune(line)[len([]rune(line))-1]
			if i > 0 && isClosingPunct(first) {
				t.Errorf("[%s]: line %d starts with closing punctuation %q: %q", tc.Desc, i, string(first), line)
			}
			if i < len(lines)-1 && isOpeningPunct(last) {
				t.Errorf("[%s]: line %d ends with opening punctuation %q: %q", tc.Desc, i, string(last), line)
			}
		}
	}
}

// TestWordWrapExplicitNewline verifies that explicit newline characters (\n)
// are honoured when wrapping CJK and mixed-script text.
// Covers: pure Chinese, mixed CJK/Latin, CJK with punctuation adjacent to \n,
// and Japanese with multiple consecutive newlines.
func TestWordWrapExplicitNewline(t *testing.T) {
	testcases := []struct {
		FontName       string
		FontSize       int
		MaxWidthPoints float64
		Text           string
		WantLines      []string
		Desc           string
	}{
		{
			"Helvetica", 12, 500,
			"天地玄黄\n宇宙洪荒",
			[]string{"天地玄黄", "宇宙洪荒"},
			"Chinese: explicit newline splits CJK text",
		},
		{
			"Helvetica", 12, 500,
			"Hello\n世界",
			[]string{"Hello", "世界"},
			"Mixed: explicit newline between ASCII and CJK",
		},
		{
			"Helvetica", 12, 500,
			"天地玄黄，\n宇宙洪荒。",
			[]string{"天地玄黄，", "宇宙洪荒。"},
			"Chinese: explicit newline with punctuation preserved",
		},
		{
			"Helvetica", 12, 500,
			"こんにちは\n世界\nHello",
			[]string{"こんにちは", "世界", "Hello"},
			"Japanese: multiple explicit newlines with mixed scripts",
		},
	}

	for _, tc := range testcases {
		gotLines := WordWrap(tc.Text, tc.FontName, tc.FontSize, tc.MaxWidthPoints)
		if !reflect.DeepEqual(gotLines, tc.WantLines) {
			t.Errorf("[%s]:\n  got:  %v\n  want: %v", tc.Desc, gotLines, tc.WantLines)
		}
	}
}

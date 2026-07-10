package model

import "testing"

// TestCanBreakAfterChar verifies which characters can be used for wrap.
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
		{'，', true, "CJK Symbols and Punctuation"},
		{'Ａ', true, "Fullwidth Forms"},
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

// TestWordWrapCJK verifies word wrapping of CJK text.
func TestWordWrapCJK(t *testing.T) {
	testcases := []struct {
		FontName       string
		FontSize       float64
		MaxWidthPoints float64
		Text           string
		WantMinLines   int
		Desc           string
	}{
		{"Helvetica", 12, 100,
			"天地玄黄宇宙洪荒日月盈昃辰宿列张",
			2,
			"CJK text without spaces should wrap into multiple lines",
		},

		{"Helvetica", 12, 50,
			"天地玄黄宇宙洪荒",
			2,
			"narrow width forces more line breaks",
		},

		{"Helvetica", 12, 500,
			"天地玄黄",
			1,
			"short CJK text fits in one line",
		},

		{"Helvetica", 12, 100,
			"僕生まれ育つ此郷里山",
			2,
			"Japanese Hiragana/Kanji mix should wrap",
		},

		{"Helvetica", 12, 100,
			"Hello World",
			1,
			"ASCII text still wraps at spaces normally",
		},

		{"Helvetica", 12, 100,
			"Hello 世界 Test 測試",
			2,
			"mixed CJK and ASCII text",
		},
	}

	for _, tc := range testcases {
		gotLines := WordWrap(tc.Text, tc.FontName, int(tc.FontSize), tc.MaxWidthPoints)
		if len(gotLines) < tc.WantMinLines {
			t.Errorf("[%s]: expected at least %d lines, got %d: %v", tc.Desc, tc.WantMinLines, len(gotLines), gotLines)
		}
	}
}

/*
Copyright 2024 The pdfcpu Authors.

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
	"testing"
)

func TestDecodeNameHexInvalid(t *testing.T) {
	testcases := []string{
		"#",
		"#A",
		"#a",
		"#G0",
		"#00",
		"Fo\x00",
	}
	for _, tc := range testcases {
		if decoded, err := decodeNameHexSequence(tc); err == nil {
			t.Errorf("expected error decoding %s, got %s", tc, decoded)
		}
	}
}

func TestDecodeNameHexValid(t *testing.T) {
	testcases := []struct {
		Input    string
		Expected string
	}{
		{"", ""},
		{"Foo", "Foo"},
		{"A#23", "A#"},
		// Examples from "7.3.5 Name Objects"
		{"Name1", "Name1"},
		{"ASomewhatLongerName", "ASomewhatLongerName"},
		{"A;Name_With-Various***Characters?", "A;Name_With-Various***Characters?"},
		{"1.2", "1.2"},
		{"$$", "$$"},
		{"@pattern", "@pattern"},
		{".notdef", ".notdef"},
		{"Lime#20Green", "Lime Green"},
		{"paired#28#29parentheses", "paired()parentheses"},
		{"The_Key_of_F#23_Minor", "The_Key_of_F#_Minor"},
		{"A#42", "AB"},
	}
	for _, tc := range testcases {
		decoded, err := decodeNameHexSequence(tc.Input)
		if err != nil {
			t.Errorf("decoding %s failed: %s", tc.Input, err)
		} else if decoded != tc.Expected {
			t.Errorf("expected %s when decoding %s, got %s", tc.Expected, tc.Input, decoded)
		}
	}
}

func TestDetectNonEscaped(t *testing.T) {
	testcases := []struct {
		input string
		want  int
	}{
		{"", -1},
		{" ( ", 1},
		{" \\( )", -1},
		{"\\(", -1},
		{"   \\(   ", -1},
		{"\\()(", 3},
		{" \\(\\((abc)", 5},
	}
	for _, tc := range testcases {
		got := detectNonEscaped(tc.input, "(")
		if tc.want != got {
			t.Errorf("%s, want: %d, got: %d", tc.input, tc.want, got)
		}
	}
}

func TestBalancedParenthesesPrefix(t *testing.T) {
	testcases := []struct {
		input string
		want  int
	}{
		// Basic cases
		{"()", 1},
		{"(abc)", 4},
		{"(a(b)c)", 6},
		{"(escaped \\) paren)", 17},
		{"(unbalanced", -1},

		// UTF-16BE cases - issue #1210
		// UTF-16BE BOM: 0xFE 0xFF
		// Chinese text "使用说明" encoded in UTF-16BE
		// 使(U+4F7F): 0x4F 0x7F
		// 用(U+7528): 0x75 0x28 <- contains 0x28 which looks like '(' but isn't
		// 说(U+8BF4): 0x8B 0xF4
		// 明(U+660E): 0x66 0x0E
		{"(\xfe\xff\x4f\x7f\x75\x28\x8b\xf4\x66\x0e)", 11},

		// UTF-16LE cases
		// UTF-16LE BOM: 0xFF 0xFE
		// Same text in UTF-16LE: 0x7F 0x4F 0x28 0x75 0xF4 0x8B 0x0E 0x66
		{"(\xff\xfe\x7f\x4f\x28\x75\xf4\x8b\x0e\x66)", 11},

		// UTF-16BE with actual parentheses (0x00 0x28 and 0x00 0x29)
		{"(\xfe\xff\x00\x28\x00\x29)", 7},

		// Mixed ASCII and UTF-16BE
		{"(\xfe\xff\x00\x48\x00\x65\x00\x6c\x00\x6c\x00\x6f)", 13}, // "Hello" in UTF-16BE
	}

	for _, tc := range testcases {
		got := balancedParenthesesPrefix(tc.input)
		if tc.want != got {
			t.Errorf("balancedParenthesesPrefix(%q), want: %d, got: %d", tc.input, tc.want, got)
		}
	}
}

func TestDetectKeywords(t *testing.T) {
	msg := "detectKeywords"

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} endobj
	//                                    streamInd                        endInd
	//                                  -1 if absent                    -1 if absent

	//s := "5 0 obj\n<</Title (xxxxendobjxxxxx)\n/Parent 4 0 R\n/Dest [3 0 R /XYZ 0 738 0]>>\nendobj\n" //78

	s := "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % comment \n>>\nendobj\n\n2 0 obj\n"
	//    0....... ..1 .........2.........3.........4.........5..... ... .6
	endInd, _, err := DetectKeywords(s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd != 59 {
		t.Errorf("%s failed: want %d, got %d", msg, 59, endInd)
	}

	// negative test
	s = "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % endobject"
	endInd, _, err = DetectKeywords(s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd > 0 {
		t.Errorf("%s failed: want %d, got %d", msg, 0, endInd)
	}

}

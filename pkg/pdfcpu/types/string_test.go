/*
Copyright 2022 The pdfcpu Authors.

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

import (
	"bytes"
	"errors"
	"testing"
)

// TestByteForOctalString verifies byte for octal string.
func TestByteForOctalString(t *testing.T) {
	tests := []struct {
		input    string
		expected byte
	}{
		{
			"001",
			0x1,
		},
		{
			"01",
			0x1,
		},
		{
			"1",
			0x1,
		},
		{
			"010",
			0x8,
		},
		{
			"020",
			0x10,
		},
		{
			"377",
			0xff,
		},
		{
			"400",
			0x00,
		},
		{
			"777",
			0xff,
		},
		{
			"",
			0x00,
		},
		{
			"8",
			0x00,
		},
		{
			"1000",
			0x00,
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := ByteForOctalString(test.input)
			if got != test.expected {
				t.Errorf("got %x; want %x", got, test.expected)
			}
		})
	}
}

func TestEscapedUTF16StringRejectsInvalidUTF8(t *testing.T) {
	_, err := EscapedUTF16String(string([]byte{0xFF}))
	if !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("expected %v, got %v", ErrInvalidUTF8, err)
	}
}

// TestUnescapeStringWithOctal verifies unescape string with octal.
func TestUnescapeStringWithOctal(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{
			"\\5",
			[]byte{0x05},
		},
		{
			"\\5a",
			[]byte{0x05, 'a'},
		},
		{
			"\\5\\5",
			[]byte{0x05, 0x05},
		},
		{
			"\\53",
			[]byte{'+'},
		},
		{
			"\\53a",
			[]byte{'+', 'a'},
		},
		{
			"\\053",
			[]byte{'+'},
		},
		{
			"\\0053",
			[]byte{0x05, '3'},
		},
		{
			"\\400",
			[]byte{0x00},
		},
		{
			"\\777",
			[]byte{0xff},
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := Unescape(test.input)
			if err != nil {
				t.Fail()
			}
			if !bytes.Equal(got, test.expected) {
				t.Errorf("got %x; want %x", got, test.expected)
			}
		})
	}
}

// TestStringLiteralToStringPDFDocEncoding verifies PDFDocEncoding fallback for non-UTF16 string literals.
func TestStringLiteralToStringPDFDocEncoding(t *testing.T) {
	tests := []struct {
		name string
		in   StringLiteral
		want string
	}{
		{
			name: "ASCII",
			in:   StringLiteral("625,50"),
			want: "625,50",
		},
		{
			name: "Euro",
			in:   StringLiteral("625,50 \xa0"),
			want: "625,50 \u20ac",
		},
		{
			name: "UTF8",
			in:   StringLiteral("caf\u00e9"),
			want: "caf\u00e9",
		},
		{
			name: "UTF16BE",
			in:   StringLiteral(EncodeUTF16String("625,50 \u20ac")),
			want: "625,50 \u20ac",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StringLiteralToString(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}
}

// TestHexLiteralToStringPDFDocEncoding verifies PDFDocEncoding fallback for non-UTF16 hex literals.
func TestHexLiteralToStringPDFDocEncoding(t *testing.T) {
	got, err := HexLiteralToString(HexLiteral("3632352c353020a0"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "625,50 \u20ac" {
		t.Fatalf("got %q; want %q", got, "625,50 \u20ac")
	}
}

// TestDecodeName verifies decode name.
func TestDecodeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"",
			"",
		},
		{
			"Size",
			"Size",
		},
		{
			"S#69#7a#65",
			"Size",
		},
		{
			"#52#6f#6f#74",
			"Root",
		},
		{
			"#4f#75t#6c#69#6e#65#73",
			"Outlines",
		},
		{
			"C#6fu#6et",
			"Count",
		},
		{
			"K#69#64s",
			"Kids",
		},
		{
			"#50a#72e#6et",
			"Parent",
		},
		{
			"#4d#65di#61#42#6f#78",
			"MediaBox",
		},
		{
			"#46#69#6c#74er",
			"Filter",
		},
		{
			"#46#6ca#74e#44#65c#6fde",
			"FlateDecode",
		},
		{
			"A#53#43#49I#48e#78D#65code",
			"ASCIIHexDecode",
		},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := DecodeName(test.input)
			if err != nil {
				t.Fail()
			}
			if got != test.expected {
				t.Errorf("got %x; want %x", got, test.expected)
			}
		})
	}
}

// TestEncodeName verifies encode name.
func TestEncodeName(t *testing.T) {
	testcases := []struct {
		Input    string
		Expected string
	}{
		{"Foo", "Foo"},
		{"A#", "A#23"},
		{"F#o", "F#23o"},
		{"A;Name_With-Various***Characters?", "A;Name_With-Various***Characters?"},
		{"1.2", "1.2"},
		{"$$", "$$"},
		{"@pattern", "@pattern"},
		{".notdef", ".notdef"},
		{"Lime Green", "Lime#20Green"},
		{"paired()parentheses", "paired#28#29parentheses"},
		{"The_Key_of_F#_Minor", "The_Key_of_F#23_Minor"},
	}
	for _, tc := range testcases {
		if encoded := EncodeName(tc.Input); encoded != tc.Expected {
			t.Errorf("expected %s for %s, got %s", tc.Expected, tc.Input, encoded)
		}
	}
}

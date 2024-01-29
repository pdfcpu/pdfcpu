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
	"testing"
)

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

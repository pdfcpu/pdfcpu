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

func TestDetectKeywords(t *testing.T) {
	msg := "detectKeywords"

	s := "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % comment \n>>\nendobj\n\n2 0 obj\n"
	//    0....... ..1 .........2.........3.........4.........5..... ... .6
	endInd, _, err := DetectKeywords(s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd != 59 {
		t.Errorf("%s failed: want %d, got %d", msg, 59, endInd)
	}

	s = "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % endobject"
	endInd, _, err = DetectKeywords(s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd > 0 {
		t.Errorf("%s failed: want %d, got %d", msg, 0, endInd)
	}

}

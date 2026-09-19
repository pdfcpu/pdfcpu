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

import "testing"

// TestDecodeUTF16StringSurrogateBoundary verifies the BMP boundary following the surrogate range.
func TestDecodeUTF16StringSurrogateBoundary(t *testing.T) {
	tests := []struct {
		name    string
		encoded []byte
		want    string
		wantErr bool
	}{
		{"ASCII", []byte{0xFE, 0xFF, 0x00, 0x41}, "A", false},
		{"LastHighSurrogate", []byte{0xFE, 0xFF, 0xDB, 0xFF}, "", true},
		{"FirstLowSurrogate", []byte{0xFE, 0xFF, 0xDC, 0x00}, "", true},
		{"LastLowSurrogate", []byte{0xFE, 0xFF, 0xDF, 0xFF}, "", true},
		{"FirstPrivateUse", []byte{0xFE, 0xFF, 0xE0, 0x00}, "\uE000", false},
		{"SecondPrivateUse", []byte{0xFE, 0xFF, 0xE0, 0x01}, "\uE001", false},
		{"ValidSurrogatePair", []byte{0xFE, 0xFF, 0xD8, 0x3D, 0xDE, 0x00}, "\U0001F600", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeUTF16String(string(tt.encoded))
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v; want error %t", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %q; want %q", got, tt.want)
			}
		})
	}
}

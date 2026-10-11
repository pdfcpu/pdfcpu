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

package filter

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"testing"
)

// TestDCTDecodeLengthUsesConfiguredLimit verifies that a JPEG cannot be decoded only up to a prefix.
func TestDCTDecodeLengthUsesConfiguredLimit(t *testing.T) {
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}

	f, err := NewFilter(DCT, nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	r, err := f.DecodeLength(bytes.NewReader(jpegData.Bytes()), 1)
	if err != nil {
		t.Fatalf("small prefix rejected: %v", err)
	}
	decoded, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) <= 1 {
		t.Fatalf("DCT returned only %d bytes, want the full decoded image", len(decoded))
	}

	f, err = NewFilter(DCT, nil, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.DecodeLength(bytes.NewReader(jpegData.Bytes()), 1); !errors.Is(err, ErrDecodeLimitExceeded) {
		t.Fatalf("got %v, want %v", err, ErrDecodeLimitExceeded)
	}
}

func encodedFilterData(t *testing.T, name string, parms map[string]int, data string) []byte {
	t.Helper()
	f, err := NewFilter(name, parms)
	if err != nil {
		t.Fatal(err)
	}
	r, err := f.Encode(bytes.NewBufferString(data))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestDecodeLengthHonorsConfiguredLimit verifies that a prefix request cannot bypass the decoded-byte limit.
func TestDecodeLengthHonorsConfiguredLimit(t *testing.T) {
	const content = "hello"
	tests := []struct {
		name   string
		filter string
		parms  map[string]int
		data   []byte
		limit  int64
		prefix int64
	}{
		{"ASCIIHex", ASCIIHex, nil, encodedFilterData(t, ASCIIHex, nil, content), 4, 5},
		{"ASCII85", ASCII85, nil, encodedFilterData(t, ASCII85, nil, content), 4, 5},
		{"RunLength", RunLength, nil, encodedFilterData(t, RunLength, nil, content), 4, 5},
		{"LZW", LZW, nil, encodedFilterData(t, LZW, nil, content), 4, 5},
		{"Flate", Flate, nil, flateTestData(t, content).Bytes(), 4, 5},
		{"Flate predictor", Flate, map[string]int{"Predictor": PredictorNone, "Columns": 3},
			flateTestData(t, string([]byte{PNGNone, 'f', 'o', 'o', PNGNone, 'b', 'a', 'r'})).Bytes(), 5, 6},
		{"CCITT", CCITTFax, map[string]int{"K": -1, "Columns": 8, "Rows": 2}, ccittTwoWhiteRows, 1, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, err := NewFilter(tc.filter, tc.parms, tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.DecodeLength(bytes.NewReader(tc.data), tc.prefix); !errors.Is(err, ErrDecodeLimitExceeded) {
				t.Fatalf("got %v, want %v", err, ErrDecodeLimitExceeded)
			}

			r, err := f.DecodeLength(bytes.NewReader(tc.data), tc.limit)
			if err != nil {
				t.Fatalf("prefix at limit: %v", err)
			}
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if int64(len(got)) != tc.limit {
				t.Fatalf("prefix at limit returned %d bytes, want %d", len(got), tc.limit)
			}
		})
	}
}

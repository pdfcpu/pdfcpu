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

package sign

import (
	"bytes"
	"math"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func byteRange(values ...types.Object) types.Array {
	return types.Array(values)
}

// TestBytesForByteRange verifies safe extraction of signature byte ranges.
func TestBytesForByteRange(t *testing.T) {
	ra := bytes.NewReader([]byte("0123456789"))
	arr := byteRange(types.Integer(0), types.Integer(3), types.Integer(7), types.Integer(3))

	got, err := bytesForByteRange(ra, arr)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("012789"); !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestBytesForByteRangeRejectsInvalidRanges verifies malformed range handling.
func TestBytesForByteRangeRejectsInvalidRanges(t *testing.T) {
	ra := bytes.NewReader([]byte("0123456789"))
	tests := []struct {
		name string
		arr  types.Array
	}{
		{"Length", byteRange(types.Integer(0), types.Integer(3))},
		{"Type", byteRange(types.Integer(0), types.Name("x"), types.Integer(7), types.Integer(3))},
		{"Negative", byteRange(types.Integer(0), types.Integer(-1), types.Integer(7), types.Integer(3))},
		{"Overlap", byteRange(types.Integer(0), types.Integer(5), types.Integer(4), types.Integer(3))},
		{"OffsetOverflow", byteRange(types.Integer(math.MaxInt), types.Integer(1), types.Integer(math.MaxInt), types.Integer(0))},
		{"OutOfBounds", byteRange(types.Integer(0), types.Integer(3), types.Integer(7), types.Integer(100))},
		{"HugeOutOfBounds", byteRange(types.Integer(0), types.Integer(0), types.Integer(0), types.Integer(math.MaxInt))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := bytesForByteRange(ra, tt.arr); err == nil {
				t.Fatal("expected invalid ByteRange error")
			}
		})
	}
}

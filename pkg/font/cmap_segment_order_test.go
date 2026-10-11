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
	"encoding/binary"
	"strings"
	"testing"
)

func cmapFormat4Segments(segs [][2]uint16) table {
	segs = append(segs, [2]uint16{0xFFFF, 0xFFFF})
	n := len(segs)
	b := make([]byte, 16+8*n)
	binary.BigEndian.PutUint16(b[0:], 4)
	binary.BigEndian.PutUint16(b[2:], uint16(len(b)))
	binary.BigEndian.PutUint16(b[6:], uint16(2*n))
	for i, seg := range segs {
		binary.BigEndian.PutUint16(b[14+2*i:], seg[1])
		binary.BigEndian.PutUint16(b[16+2*n+2*i:], seg[0])
	}
	return table{size: uint32(len(b)), data: b}
}

func cmapFormat12Groups(groups [][3]uint32) table {
	b := make([]byte, 16+12*len(groups))
	binary.BigEndian.PutUint16(b[0:], 12)
	binary.BigEndian.PutUint32(b[4:], uint32(len(b)))
	binary.BigEndian.PutUint32(b[12:], uint32(len(groups)))
	for i, group := range groups {
		binary.BigEndian.PutUint32(b[16+12*i:], group[0])
		binary.BigEndian.PutUint32(b[20+12*i:], group[1])
		binary.BigEndian.PutUint32(b[24+12*i:], group[2])
	}
	return table{size: uint32(len(b)), data: b}
}

// TestCMapFormat4SegmentOrder verifies that repeated or overlapping segments cannot multiply parsing work.
func TestCMapFormat4SegmentOrder(t *testing.T) {
	tests := []struct {
		name    string
		segs    [][2]uint16
		wantErr bool
	}{
		{"adjacent", [][2]uint16{{'A', 'Z'}, {'[', '`'}}, false},
		{"overlapping", [][2]uint16{{'A', 'Z'}, {'Y', '['}}, true},
		{"repeated", [][2]uint16{{'A', 'Z'}, {'A', 'Z'}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmapFormat4Segments(tc.segs).parseCMapFormat4(cmapTestFont(256))
			if (err != nil) != tc.wantErr {
				t.Fatalf("parse error: got %v, want error %t", err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "previous segment") {
				t.Fatalf("parse error: got %v, want segment-order error", err)
			}
		})
	}
}

// TestCMapFormat12GroupOrder verifies that groups have disjoint, ascending character ranges.
func TestCMapFormat12GroupOrder(t *testing.T) {
	tests := []struct {
		name    string
		groups  [][3]uint32
		wantErr bool
	}{
		{"adjacent", [][3]uint32{{'A', 'Z', 1}, {'[', '`', 27}}, false},
		{"overlapping", [][3]uint32{{'A', 'Z', 1}, {'Z', '`', 27}}, true},
		{"out of order", [][3]uint32{{'A', 'Z', 1}, {'0', '9', 27}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := cmapFormat12Groups(tc.groups).parseCMapFormat12(cmapTestFont(256))
			if (err != nil) != tc.wantErr {
				t.Fatalf("parse error: got %v, want error %t", err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "previous group") {
				t.Fatalf("parse error: got %v, want group-order error", err)
			}
		})
	}
}

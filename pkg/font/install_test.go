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
	"math"
	"os"
	"path/filepath"
	"testing"
)

func writeTTCFile(t *testing.T, count uint32, offsets ...uint32) string {
	t.Helper()
	b := make([]byte, 12+len(offsets)*4)
	copy(b, ttcTag)
	binary.BigEndian.PutUint32(b[4:], 0x00010000)
	binary.BigEndian.PutUint32(b[8:], count)
	for i, off := range offsets {
		binary.BigEndian.PutUint32(b[12+i*4:], off)
	}
	fn := filepath.Join(t.TempDir(), "test.ttc")
	if err := os.WriteFile(fn, b, 0600); err != nil {
		t.Fatal(err)
	}
	return fn
}

// TestInstallTrueTypeCollectionRejectsInvalidCounts verifies safe TTC count handling.
func TestInstallTrueTypeCollectionRejectsInvalidCounts(t *testing.T) {
	tests := []struct {
		name    string
		count   uint32
		offsets []uint32
	}{
		{"Zero", 0, nil},
		{"Huge", math.MaxUint32, nil},
		{"MissingOffset", 1, nil},
		{"OffsetInHeader", 1, []uint32{12}},
		{"OffsetOutOfBounds", 1, []uint32{100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := writeTTCFile(t, tt.count, tt.offsets...)
			if _, err := InstallTrueTypeCollection(t.TempDir(), fn); err == nil {
				t.Fatal("expected corrupt TTC error")
			}
		})
	}
}

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
	"io"
	"strings"
	"testing"
	"time"
)

// Two all-white 8-pixel Group 4 rows followed by EOFB.
var ccittTwoWhiteRows = []byte{0xC0, 0x04, 0x00, 0x40}

func decodeCCITTWithin(t *testing.T, parms map[string]int, data []byte) ([]byte, error) {
	t.Helper()
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		f, err := NewFilter(CCITTFax, parms)
		if err != nil {
			done <- result{err: err}
			return
		}
		r, err := f.Decode(bytes.NewReader(data))
		if err != nil {
			done <- result{err: err}
			return
		}
		b, err := io.ReadAll(r)
		done <- result{data: b, err: err}
	}()
	select {
	case got := <-done:
		return got.data, got.err
	case <-time.After(5 * time.Second):
		t.Fatalf("CCITT decode with %v did not return within 5 seconds", parms)
		return nil, nil
	}
}

// TestCCITTRowsWithoutKnownHeight verifies that omitted and nonpositive Rows decode through the end of the data.
func TestCCITTRowsWithoutKnownHeight(t *testing.T) {
	tests := []struct {
		name  string
		parms map[string]int
	}{
		{"omitted", map[string]int{"K": -1, "Columns": 8}},
		{"zero", map[string]int{"K": -1, "Columns": 8, "Rows": 0}},
		{"negative", map[string]int{"K": -1, "Columns": 8, "Rows": -1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeCCITTWithin(t, tc.parms, ccittTwoWhiteRows)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte{0xFF, 0xFF}) {
				t.Fatalf("decoded bytes: got % x, want ff ff", got)
			}
		})
	}
}

// TestCCITTRejectsNonpositiveColumns verifies that zero-width decoding cannot loop without producing output.
func TestCCITTRejectsNonpositiveColumns(t *testing.T) {
	tests := []struct {
		name  string
		parms map[string]int
	}{
		{"zero and unknown rows", map[string]int{"K": -1, "Columns": 0}},
		{"zero and many rows", map[string]int{"K": -1, "Columns": 0, "Rows": 1 << 30}},
		{"negative", map[string]int{"K": -1, "Columns": -1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeCCITTWithin(t, tc.parms, nil)
			if err == nil || !strings.Contains(err.Error(), "Columns") {
				t.Fatalf("got %v, want Columns validation error", err)
			}
		})
	}
}

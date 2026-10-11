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

package model

import (
	"strings"
	"testing"
	"time"
)

// TestDetectKeywordsLexicalBoundaries covers comments, string delimiters, and backslashes outside strings.
func TestDetectKeywordsLexicalBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantEnd    int
		wantStream int
	}{
		{"backslash before string", "<</K/a\\(endobj )>>\nendobj\n", len("<</K/a\\(endobj )>>\n"), -1},
		{"backslash before comment", "<</K/a\\% endobj\n>>\nendobj\n", len("<</K/a\\% endobj\n>>\n"), -1},
		{"endobj after string", "(abc)endobj\n2 0 obj\nstream\n", 5, 20},
		{"stream after string", "(abc)stream\nendobj\n", 12, 5},
		{"comment to EOF", "<</A 1>> % endobj stream", -1, -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			end, stream, err := DetectKeywords(t.Context(), tc.line)
			if err != nil {
				t.Fatal(err)
			}
			if end != tc.wantEnd || stream != tc.wantStream {
				t.Fatalf("got endobj=%d stream=%d, want endobj=%d stream=%d", end, stream, tc.wantEnd, tc.wantStream)
			}
		})
	}
}

// TestDetectKeywordsKeepsEndobjAfterLiteral ensures later strings cannot displace an earlier object terminator.
func TestDetectKeywordsKeepsEndobjAfterLiteral(t *testing.T) {
	line := "(abc)endobj\n2 0 obj\n<</T (def)/Length 3>>\nstream\nxyz\nendstream\nendobj\n"
	end, stream, err := DetectKeywords(t.Context(), line)
	if err != nil {
		t.Fatal(err)
	}
	if end != 5 || stream >= 0 && stream < end {
		t.Fatalf("got endobj=%d stream=%d, want first endobj at 5", end, stream)
	}
}

// TestDetectKeywordsLongInput checks that scanning many literals does not repeatedly rescan a trailing comment.
func TestDetectKeywordsLongInput(t *testing.T) {
	if testing.Short() {
		t.Skip("long-input performance regression")
	}
	run := func(n int) time.Duration {
		line := strings.Repeat("(a)", n) + strings.Repeat(" ", 4*n) + "%\nendobj\n"
		start := time.Now()
		end, _, err := DetectKeywords(t.Context(), line)
		if err != nil {
			t.Fatal(err)
		}
		if end != len(line)-len("endobj\n") {
			t.Fatalf("got endobj=%d, want %d", end, len(line)-len("endobj\n"))
		}
		return time.Since(start)
	}
	small := run(20000)
	large := run(80000)
	if large > 8*small && large > 200*time.Millisecond {
		t.Fatalf("4x input took %v vs %v", large, small)
	}
}

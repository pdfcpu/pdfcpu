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

package scan

import (
	"bufio"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
)

func TestLinesAcrossReads(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"unterminated", "abc", []string{"abc"}},
		{"crlf", "a\r\nb\r\nc", []string{"a", "b", "c"}},
		{"trailing crlf", "a\r\n", []string{"a"}},
		{"cr", "a\rb\rc", []string{"a", "b", "c"}},
		{"trailing cr", "a\r", []string{"a"}},
		{"lf", "a\nb\nc", []string{"a", "b", "c"}},
		{"trailing lf", "a\n", []string{"a"}},
		{"empty crlf line", "\r\n", []string{""}},
		{"empty cr line", "\r", []string{""}},
		{"empty lf line", "\n", []string{""}},
		{"mixed", "\r\na\rb\nc\r\n\r\nd", []string{"", "a", "b", "c", "", "d"}},
		{"consecutive cr", "a\r\rb\r\r", []string{"a", "", "b", ""}},
		{"lf then cr", "a\n\rb", []string{"a", "", "b"}},
		{"buffer boundary", strings.Repeat("a", 4095) + "\r\nb", []string{strings.Repeat("a", 4095), "b"}},
	}
	readers := []struct {
		name string
		wrap func(io.Reader) io.Reader
	}{
		{"buffered", func(r io.Reader) io.Reader { return r }},
		{"one byte", iotest.OneByteReader},
		{"half", iotest.HalfReader},
	}
	for _, tt := range tests {
		for _, reader := range readers {
			t.Run(tt.name+"/"+reader.name, func(t *testing.T) {
				s := bufio.NewScanner(reader.wrap(strings.NewReader(tt.input)))
				s.Split(Lines)
				var got []string
				for s.Scan() {
					got = append(got, s.Text())
				}
				if err := s.Err(); err != nil {
					t.Fatal(err)
				}
				if !slices.Equal(got, tt.want) {
					t.Fatalf("got %q, want %q", got, tt.want)
				}
			})
		}
	}
}

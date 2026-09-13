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

package pdfcpu

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestNextStreamOffset verifies safe stream delimiter parsing.
func TestNextStreamOffset(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name      string
		line      string
		streamInd int
		want      int
		wantErr   error
	}{
		{"LF", "stream\nabc", 0, 7, nil},
		{"CRLF", "stream   \r\nabc", 0, 11, nil},
		{"CR", "stream\r", 0, 7, nil},
		{"Offset", "xxstream\nabc", 2, 9, nil},
		{"NegativeOffset", "stream\n", -1, 0, errCorruptStreamMarker},
		{"OverflowOffset", "stream\n", maxInt, 0, errCorruptStreamMarker},
		{"WrongMarker", "xxxxxx\n", 0, 0, errCorruptStreamMarker},
		{"MissingMarker", "stream", 1, 0, errCorruptStreamMarker},
		{"Truncated", "stream", 0, 0, errTruncatedStreamMarker},
		{"TrailingSpaces", "stream   ", 0, 0, errTruncatedStreamMarker},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nextStreamOffset(tt.line, tt.streamInd)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got offset %d, want %d", got, tt.want)
			}
		})
	}
}

// TestBufferRejectsObjectBeyondLimit verifies bounded indirect object buffering.
func TestBufferRejectsObjectBeyondLimit(t *testing.T) {
	_, _, _, _, err := buffer(t.Context(), strings.NewReader(strings.Repeat("x", 32)), 16)
	if !errors.Is(err, errObjectBufferLimit) {
		t.Fatalf("got %v, want object buffer limit error", err)
	}
}

// TestBufferAcceptsMarkersAtLimit verifies markers may end at the configured limit.
func TestBufferAcceptsMarkersAtLimit(t *testing.T) {
	for _, input := range []string{
		"1 0 obj null endobj\n",
		"1 0 obj <<>>stream\n",
	} {
		t.Run(input, func(t *testing.T) {
			buf, _, _, _, err := buffer(t.Context(), strings.NewReader(input), int64(len(input)))
			if err != nil {
				t.Fatal(err)
			}
			if got := string(buf); got != input {
				t.Fatalf("got %q, want %q", got, input)
			}
		})
	}
}

// TestBufferHonorsCancellation verifies cancellation takes precedence over reading.
func TestBufferHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, _, _, err := buffer(ctx, strings.NewReader(strings.Repeat("x", 32)), 16)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context cancellation", err)
	}
}

// TestObjectHonorsBufferLimit verifies configured limits at the indirect-object reader boundary.
func TestObjectHonorsBufferLimit(t *testing.T) {
	const input = "1 0 obj (bounded object) endobj\n"
	for _, limit := range []int64{16, int64(len(input)), 0, -1} {
		conf := model.NewStatelessConfiguration()
		conf.Limits.MaxObjectBytes = limit
		ctx, err := model.NewContext(strings.NewReader(input), conf)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, _, err = object(t.Context(), ctx, 0, 1, 0)
		if limit == 16 || limit < 0 {
			if !errors.Is(err, errObjectBufferLimit) {
				t.Fatalf("limit %d: got %v, want object buffer limit", limit, err)
			}
		} else if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
}

// TestXRefStreamHonorsBufferLimit verifies xref streams cannot bypass the configured object buffer limit.
func TestXRefStreamHonorsBufferLimit(t *testing.T) {
	const input = "1 0 obj << /Type /XRef /Length 0 >> stream\n"
	conf := model.NewStatelessConfiguration()
	conf.Limits.MaxObjectBytes = 16
	ctx, err := model.NewContext(strings.NewReader(input), conf)
	if err != nil {
		t.Fatal(err)
	}
	var offset int64
	_, err = parseXRefStream(t.Context(), ctx, strings.NewReader(input), &offset, 0, 0)
	if !errors.Is(err, errObjectBufferLimit) {
		t.Fatalf("got %v, want object buffer limit", err)
	}
}

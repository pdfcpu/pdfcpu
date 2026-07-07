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
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestParsePageDimRejectsInvalidDimensions verifies finite, positive parser inputs.
func TestParsePageDimRejectsInvalidDimensions(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "zero width", value: "0 10", want: "dimension x must be a positive finite number"},
		{name: "negative width", value: "-1 10", want: "dimension x must be a positive finite number"},
		{name: "NaN width", value: "NaN 10", want: "dimension x must be a positive finite number"},
		{name: "positive infinite width", value: "+Inf 10", want: "dimension x must be a positive finite number"},
		{name: "negative infinite width", value: "-Inf 10", want: "dimension x must be a positive finite number"},
		{name: "zero height", value: "10 0", want: "dimension y must be a positive finite number"},
		{name: "negative height", value: "10 -1", want: "dimension y must be a positive finite number"},
		{name: "NaN height", value: "10 NaN", want: "dimension y must be a positive finite number"},
		{name: "positive infinite height", value: "10 +Inf", want: "dimension y must be a positive finite number"},
		{name: "negative infinite height", value: "10 -Inf", want: "dimension y must be a positive finite number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParsePageDim(tt.value, types.POINTS)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestParsePageDimRejectsNonFiniteConvertedDimensions verifies unit conversion cannot produce invalid dimensions.
func TestParsePageDimRejectsNonFiniteConvertedDimensions(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "width overflow", value: "1e308 10", want: "dimension x must resolve to a positive finite number"},
		{name: "height overflow", value: "10 1e308", want: "dimension y must resolve to a positive finite number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParsePageDim(tt.value, types.INCHES)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestParsePageDimPreservesNumericParseErrors verifies primitive conversion errors remain discoverable.
func TestParsePageDimPreservesNumericParseErrors(t *testing.T) {
	_, _, err := ParsePageDim("foo 10", types.POINTS)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("expected numeric parse error, got %v", err)
	}
	if !strings.Contains(err.Error(), `dimension x "foo"`) {
		t.Fatalf("expected parser-local dimension context, got %q", err)
	}
}

// TestParsePageDimAcceptsPositiveFiniteDimensions verifies valid parser input.
func TestParsePageDimAcceptsPositiveFiniteDimensions(t *testing.T) {
	dim, pageSize, err := ParsePageDim("10 20", types.POINTS)
	if err != nil {
		t.Fatal(err)
	}
	if pageSize != "" {
		t.Fatalf("expected no named page size, got %q", pageSize)
	}
	if dim == nil || dim.Width != 10 || dim.Height != 20 {
		t.Fatalf("expected 10x20 dimensions, got %v", dim)
	}
}

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
)

func TestPageLayoutRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		layout PageLayout
		value  int
	}{
		{name: "SinglePage", layout: PageLayoutSinglePage, value: 0},
		{name: "TwoColumnLeft", layout: PageLayoutTwoColumnLeft, value: 1},
		{name: "TwoColumnRight", layout: PageLayoutTwoColumnRight, value: 2},
		{name: "TwoPageLeft", layout: PageLayoutTwoPageLeft, value: 3},
		{name: "TwoPageRight", layout: PageLayoutTwoPageRight, value: 4},
		{name: "OneColumn", layout: PageLayoutOneColumn, value: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := int(tt.layout); got != tt.value {
				t.Fatalf("numeric value: got %d, want %d", got, tt.value)
			}

			pl := PageLayoutFor(strings.ToLower(tt.name))
			if pl == nil {
				t.Fatalf("PageLayoutFor(%q) returned nil", tt.name)
			}
			if *pl != tt.layout {
				t.Fatalf("layout: got %d, want %d", *pl, tt.layout)
			}
			if got := pl.String(); got != tt.name {
				t.Fatalf("string: got %q, want %q", got, tt.name)
			}
		})
	}
}

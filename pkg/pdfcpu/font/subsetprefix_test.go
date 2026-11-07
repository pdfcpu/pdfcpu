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
	"fmt"
	"testing"
)

func gidSet(gids ...uint16) map[uint16]bool {
	m := map[uint16]bool{}
	for _, g := range gids {
		m[g] = true
	}
	return m
}

func TestSubFontPrefixIsDeterministic(t *testing.T) {
	a := subFontPrefix("Roboto-Regular", gidSet(3, 17, 42))
	b := subFontPrefix("Roboto-Regular", gidSet(42, 3, 17))
	if a != b {
		t.Fatalf("same font and glyph set gave %q and %q", a, b)
	}
}

func TestSubFontPrefixDiffersBySubset(t *testing.T) {
	base := subFontPrefix("Roboto-Regular", gidSet(3, 17, 42))
	if got := subFontPrefix("Roboto-Regular", gidSet(3, 17, 43)); got == base {
		t.Errorf("different glyph sets share prefix %q", got)
	}
	if got := subFontPrefix("Roboto-Bold", gidSet(3, 17, 42)); got == base {
		t.Errorf("different fonts share prefix %q", got)
	}
}

func TestSubFontPrefixShape(t *testing.T) {
	for i := 0; i < 5000; i++ {
		p := subFontPrefix(fmt.Sprintf("Font%d", i), gidSet(uint16(i), uint16(i+1)))
		if len(p) != 6 {
			t.Fatalf("prefix %q: want 6 letters", p)
		}
		for _, r := range p {
			if r < 'A' || r > 'Z' {
				t.Fatalf("prefix %q: want uppercase ASCII letters only", p)
			}
		}
		// MuPDF forces hinting for font names starting with "DF"/"DLC".
		if p[0] == 'D' {
			t.Fatalf("prefix %q starts with D", p)
		}
	}
}

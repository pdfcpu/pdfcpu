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
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestNameRejectsMissingSubtype(t *testing.T) {
	for _, subtype := range []types.Object{nil, types.Name("")} {
		fontDict := types.Dict{
			"BaseFont": types.Name("Helvetica"),
		}
		if subtype != nil {
			fontDict["Subtype"] = subtype
		}

		_, _, err := Name(nil, fontDict, 7)
		if err == nil || err.Error() != `fontName: missing fontDict entry "Subtype"` {
			t.Fatalf("got %v, want missing Subtype error", err)
		}
	}
}

func TestFontDescriptorRejectsDescendantFontMissingType(t *testing.T) {
	fontDict := types.Dict{
		"DescendantFonts": types.Array{types.Dict{}},
	}

	_, err := FontDescriptor(&model.XRefTable{}, fontDict, 7)
	if err == nil || !strings.Contains(err.Error(), "descendant font dict missing Type for font object 7") {
		t.Fatalf("got %v, want missing descendant font Type error", err)
	}
}

func usedGIDsFromCMapWithoutPanic(t *testing.T, cMap string) (gids []uint16, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("usedGIDsFromCMap panicked: %v", r)
		}
	}()
	return usedGIDsFromCMap(cMap)
}

func TestUsedGIDsFromCMap(t *testing.T) {
	cMap := "endcodespacerange\r\n2 beginbfchar\n<0001> <0041>\n<00FF> <0042>\nendbfchar\nendcmap"
	gids, err := usedGIDsFromCMap(cMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(gids) != 2 || gids[0] != 1 || gids[1] != 255 {
		t.Fatalf("expected [1 255], got %v", gids)
	}
}

func TestUsedGIDsFromCMapRejectsMalformedInputWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing codespace end", body: "1 beginbfchar"},
		{name: "header token count", body: "endcodespacerange\n1 beginbfchar extra"},
		{name: "header operator", body: "endcodespacerange\n1 beginbfrange"},
		{name: "header count", body: "endcodespacerange\nno beginbfchar"},
		{name: "negative count", body: "endcodespacerange\n-1 beginbfchar"},
		{name: "mapping token count", body: "endcodespacerange\n1 beginbfchar\n<0001>\nendbfchar\nendcmap"},
		{name: "short source", body: "endcodespacerange\n1 beginbfchar\n<01> <0041>\nendbfchar\nendcmap"},
		{name: "source boundary", body: "endcodespacerange\n1 beginbfchar\n0001> <0041>\nendbfchar\nendcmap"},
		{name: "source hex", body: "endcodespacerange\n1 beginbfchar\n<ZZZZ> <0041>\nendbfchar\nendcmap"},
		{name: "destination boundary", body: "endcodespacerange\n1 beginbfchar\n<0001> 0041>\nendbfchar\nendcmap"},
		{name: "destination hex", body: "endcodespacerange\n1 beginbfchar\n<0001> <ZZZZ>\nendbfchar\nendcmap"},
		{name: "bad endbfchar tokens", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>\nendbfchar extra\nendcmap"},
		{name: "unexpected block", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>\nendbfchar\n1 beginbfchar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := usedGIDsFromCMapWithoutPanic(t, tt.body); !errors.Is(err, errCorruptCMap) {
				t.Fatalf("expected %v, got %v", errCorruptCMap, err)
			}
		})
	}
}

func TestUsedGIDsFromCMapRejectsTruncatedInputWithoutPanic(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "after codespace", body: "endcodespacerange"},
		{name: "after header", body: "endcodespacerange\n1 beginbfchar"},
		{name: "within mappings", body: "endcodespacerange\n2 beginbfchar\n<0001> <0041>"},
		{name: "before endbfchar", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>"},
		{name: "before endcmap", body: "endcodespacerange\n1 beginbfchar\n<0001> <0041>\nendbfchar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := usedGIDsFromCMapWithoutPanic(t, tt.body); !errors.Is(err, errCorruptCMap) {
				t.Fatalf("expected %v, got %v", errCorruptCMap, err)
			}
		})
	}
}

func TestUsedGIDsFromCMapPreservesScannerFailure(t *testing.T) {
	cMap := "endcodespacerange\n" + strings.Repeat("1", 70_000)
	_, err := usedGIDsFromCMapWithoutPanic(t, cMap)
	if !errors.Is(err, errCorruptCMap) {
		t.Fatalf("expected %v, got %v", errCorruptCMap, err)
	}
	if !strings.Contains(err.Error(), "read beginbfchar header") {
		t.Fatalf("expected scan phase in %q", err.Error())
	}
}

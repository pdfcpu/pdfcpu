/*
Copyright 2024 The pdfcpu Authors.

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
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestObjectParserRegularSpaceBytes preserves bytes outside the six PDF whitespace characters.
func TestObjectParserRegularSpaceBytes(t *testing.T) {
	for _, regular := range []string{"\v", "\x85", "\xa0", "\u0085", "\u00a0", "\u2003", "\u2028", "\u3000", "\xff"} {
		t.Run(fmt.Sprintf("%q", regular), func(t *testing.T) {
			name := "A" + regular + "B"
			line := "/" + name + " /Tail"
			o, err := ParseObject(t.Context(), &line, 0)
			if err != nil || o != types.Name(name) || line != " /Tail" {
				t.Fatalf("name = %v, %v; remaining = %q", o, err, line)
			}
			for _, relaxed := range []bool{false, true} {
				input := regular + "\n/Name"
				got, eol := trimLeftSpace(" \t"+input, relaxed)
				if got != input || eol {
					t.Fatalf("trim relaxed=%t: got %q, eol=%t; want %q, false", relaxed, got, eol, input)
				}
			}
			prefix := "12" + regular + "34"
			if i, rest := positionToNextWhitespace(prefix + " 5"); i != len(prefix) || rest != " 5" {
				t.Fatalf("whitespace boundary = %d, %q", i, rest)
			}
			if i, rest := positionToNextWhitespaceOrChar(prefix+"/Tail", "/"); i != len(prefix) || rest != "/Tail" {
				t.Fatalf("delimiter boundary = %d, %q", i, rest)
			}
		})
	}
}

// TestObjectParserPDFWhitespace checks all six PDF token separators.
func TestObjectParserPDFWhitespace(t *testing.T) {
	for _, ws := range []string{"\x00", "\t", "\n", "\f", "\r", " "} {
		t.Run(fmt.Sprintf("%q", ws), func(t *testing.T) {
			line := ws + "/Name" + ws + "17"
			o, err := ParseObject(t.Context(), &line, 0)
			if err != nil || o != types.Name("Name") || line != ws+"17" {
				t.Fatalf("name = %v, %v; remaining = %q", o, err, line)
			}
			o, err = ParseObject(t.Context(), &line, 0)
			if err != nil || o != types.Integer(17) || line != "" {
				t.Fatalf("integer = %v, %v; remaining = %q", o, err, line)
			}
		})
	}
}

// TestTrimLeftPDFWhitespace preserves comment handling and relaxed line-ending detection.
func TestTrimLeftPDFWhitespace(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
		eol   bool
	}{
		{"", "", false},
		{"\x00\t\f /Name", "/Name", false},
		{"\x00\t\n\f\r /Name", "/Name", true},
		{"%comment\r\n/Name", "/Name", true},
		{" %first\n\t%second\r/Name", "/Name", true},
		{"%comment without line ending", "", false},
	} {
		for _, relaxed := range []bool{false, true} {
			got, eol := trimLeftSpace(tc.input, relaxed)
			if got != tc.want || eol != (relaxed && tc.eol) {
				t.Errorf("trim %q relaxed=%t: got %q, eol=%t; want %q, eol=%t", tc.input, relaxed, got, eol, tc.want, relaxed && tc.eol)
			}
		}
	}
}

// TestParseObjectAttributesWhitespace checks PDF separators keep object and generation numbers distinct.
func TestParseObjectAttributesWhitespace(t *testing.T) {
	whitespace := []string{"\x00", "\t", "\n", "\f", "\r", " ", "\r\n", "\x00\t\n\f\r "}
	const remainder = " << /Length 0 >>\nendobj\n13 0 obj"
	for _, beforeGeneration := range whitespace {
		for _, beforeObj := range whitespace {
			for _, generation := range []int{0, 34} {
				t.Run(fmt.Sprintf("%q/%q/generation=%d", beforeGeneration, beforeObj, generation), func(t *testing.T) {
					line := fmt.Sprintf("%s12%s%d%sobj%s", beforeGeneration, beforeGeneration, generation, beforeObj, remainder)
					objNr, genNr, err := ParseObjectAttributes(&line)
					if err != nil {
						t.Fatal(err)
					}
					if objNr == nil || genNr == nil {
						t.Fatal("missing object or generation number")
					}
					if *objNr != 12 || *genNr != generation {
						t.Fatalf("got %d %d, want 12 %d", *objNr, *genNr, generation)
					}
					if line != remainder {
						t.Fatalf("remaining data = %q, want %q", line, remainder)
					}
				})
			}
		}
	}
}

// TestParseFloatRejectsNonPDFNumbers checks Go-only syntax cannot produce PDF real objects.
func TestParseFloatRejectsNonPDFNumbers(t *testing.T) {
	for _, input := range []string{
		"NaN", "nan", "NAN", "Inf", "+Inf", "-Inf", "Infinity", "-Infinity", "iNfInItY",
		"1e5", "-1E-5", "0x1p2", "0X1.FP+3", "1_000.0", "1.2.3", "1,2,3", "1.-2.-3",
		"", ".", "+.", "-", "+", "1a", "١.٥", strings.Repeat("9", 400) + ".0",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := parseFloat(input)
			if err != nil || got != nil {
				t.Fatalf("parseFloat = %v, %v; want skipped token", got, err)
			}
		})
	}
}

// TestParseFloatDecimals preserves valid decimal syntax and existing tolerated spellings.
func TestParseFloatDecimals(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  float64
	}{
		{"0", 0}, {"+17", 17}, {"-98", -98}, {"34.5", 34.5}, {"-3.62", -3.62},
		{"+123.6", 123.6}, {"4.", 4}, {"-.002", -.002}, {".5", .5}, {"00.50", .5},
		{"1,5", 1.5}, {"-2,75", -2.75}, {".-5", .5}, {"1.-5", 1.5}, {"1,-5", 1.5},
		{"0." + strings.Repeat("0", 400) + "1", 0},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseFloat(tc.input)
			f, ok := got.(types.Float)
			if err != nil || !ok || f.Value() != tc.want {
				t.Fatalf("parseFloat = %v, %v; want Float %v", got, err, tc.want)
			}
		})
	}
	got, err := parseFloat("-0.0")
	f, ok := got.(types.Float)
	if err != nil || !ok || !math.Signbit(f.Value()) {
		t.Fatalf("negative zero lost: %v, %v", got, err)
	}
}

// TestParseObjectSkipsNonPDFNumbers checks discarded tokens do not consume the next object.
func TestParseObjectSkipsNonPDFNumbers(t *testing.T) {
	for _, input := range []string{"NaN", "+Inf", "-Infinity", "1e5", "0x1p2", "1_000.0"} {
		t.Run(input, func(t *testing.T) {
			line := input + " 17"
			got, err := ParseObject(t.Context(), &line, 0)
			if err != nil || got != nil || line != " 17" {
				t.Fatalf("first object = %v, %v; remaining = %q", got, err, line)
			}
			got, err = ParseObject(t.Context(), &line, 0)
			n, ok := got.(types.Integer)
			if err != nil || !ok || n != 17 || line != "" {
				t.Fatalf("next object = %v, %v; remaining = %q", got, err, line)
			}
		})
	}
}

// TestParseHexStringWhitespace verifies whitespace does not interrupt digit pairs or final padding.
func TestParseHexStringWhitespace(t *testing.T) {
	for _, ws := range []string{"\x00", "\t", "\n", "\f", "\r", " ", "\x00\t\n\f\r "} {
		for _, tc := range []struct{ digits, want string }{
			{"41", "A"}, {"4142", "AB"}, {"414", "A@"}, {"9aB", "\x9a\xb0"}, {"", ""},
		} {
			in := "<" + ws + strings.Join(strings.Split(tc.digits, ""), ws) + ws + ">"
			checkParsedHexString(t, in, tc.want)
		}
	}
	checkParsedHexString(t, "<>", "")
	checkParsedHexString(t, "<901FA>", "\x90\x1f\xa0")
}

func checkParsedHexString(t *testing.T, in, want string) {
	t.Helper()
	line := in
	o, err := ParseObject(t.Context(), &line, 0)
	if err != nil {
		t.Fatalf("%q: %v", in, err)
	}
	h, ok := o.(types.HexLiteral)
	if !ok {
		t.Fatalf("%q: got %T, want HexLiteral", in, o)
	}
	b, err := h.Bytes()
	if err != nil || string(b) != want {
		t.Fatalf("%q: decoded %q, want %q, error %v", in, b, want, err)
	}
}

// TestHexStringRejectsInvalidCharacters verifies the hex decoder accepts only PDF whitespace and hex digits.
func TestHexStringRejectsInvalidCharacters(t *testing.T) {
	for _, s := range []string{"4\v1", "4\u00851", "4\u00a01", "4Z1", "4%1"} {
		if _, ok := hexString(s); ok {
			t.Errorf("accepted invalid hex string %q", s)
		}
	}
}

// TestDecodeNameHexInvalid verifies invalid name hex escapes are rejected.
func TestDecodeNameHexInvalid(t *testing.T) {
	testcases := []string{
		"#",
		"#A",
		"#a",
		"#G0",
		"#00",
		"Fo\x00",
	}
	for _, tc := range testcases {
		if decoded, err := decodeNameHexSequence(tc); err == nil {
			t.Errorf("expected error decoding %s, got %s", tc, decoded)
		}
	}
}

// TestParseObjectRejectsRecursionDepth verifies object parsing respects recursion limits.
func TestParseObjectRejectsRecursionDepth(t *testing.T) {
	s := "[[[1]]]"

	_, err := ParseObject(t.Context(), &s, 0, 1)
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestParseObjectBoundsRelaxedFallback verifies relaxed parsing remains bounded.
func TestParseObjectBoundsRelaxedFallback(t *testing.T) {
	s := strings.Repeat("<</Differences[24/breve/quotesingle0 obj\r", 34)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	_, err := ParseObject(ctx, &s, 0)
	if !errors.Is(err, errArrayNotTerminated) {
		t.Fatalf("got %v, want errArrayNotTerminated", err)
	}
}

// TestParseRejectsMissingContext verifies context-aware parsers reject a nil context.
func TestParseRejectsMissingContext(t *testing.T) {
	s := "1"
	if _, err := ParseObject(nil, &s, 0); !errors.Is(err, ErrMissingContext) {
		t.Fatalf("parse object: got %v, want ErrMissingContext", err)
	}
	if _, _, err := DetectKeywords(nil, s); !errors.Is(err, ErrMissingContext) {
		t.Fatalf("detect keywords: got %v, want ErrMissingContext", err)
	}
}

// TestProcessRefCountsRejectsRecursionDepth verifies ref count processing respects recursion limits.
func TestProcessRefCountsRejectsRecursionDepth(t *testing.T) {
	conf := NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = 1
	xRefTable := newXRefTable(conf)

	o := types.Array{types.Array{types.Array{types.Integer(1)}}}

	err := ProcessRefCountsWithError(xRefTable, o)
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestDereferenceNumberPreservesDereferenceError verifies numeric dereferencing
// preserves an underlying lazy object decode error.
func TestDereferenceNumberPreservesDereferenceError(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	osd := types.NewObjectStreamDict()
	lazy := types.NewLazyObjectStreamObject(osd, 1, -1, nil)
	xRefTable.Table[1] = NewXRefTableEntryGen0(lazy)

	_, err := xRefTable.DereferenceNumber(*types.NewIndirectRef(1, 0))
	if err == nil {
		t.Fatal("expected dereference error")
	}
	if strings.Contains(err.Error(), "wrong type") {
		t.Fatalf("got %v, want underlying decode error", err)
	}
}

// TestPageTreeLookupRejectsRecursionDepth verifies page tree lookup respects recursion limits.
func TestPageTreeLookupRejectsRecursionDepth(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	maxDepth := xRefTable.MaxRecursionDepth()
	ir := types.NewIndirectRef(1, 0)
	attrs := InheritedPageAttrs{}
	pageCount := 0

	_, _, err := xRefTable.processPageTreeForPageDictDepth(t.Context(), ir, &attrs, &pageCount, 1, false, maxDepth+1, NewPageTreeVisit())
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	_, err = xRefTable.processPageTreeForPageNumberDepth(t.Context(), ir, &pageCount, 1, maxDepth+1, NewPageTreeVisit())
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestPageDictRejectsUnresolvedPage verifies PageDict returns an error instead of nil page data.
func TestPageDictRejectsUnresolvedPage(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	pages, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type":  types.Name("Pages"),
		"Count": types.Integer(1),
		"Kids":  types.Array{},
	})
	if err != nil {
		t.Fatal(err)
	}
	xRefTable.RootDict = types.Dict{"Pages": *pages}
	xRefTable.PageCount = 1

	d, indRef, attrs, err := xRefTable.PageDict(t.Context(), 1, false)
	if err == nil {
		t.Fatal("expected unresolved page error")
	}
	if d != nil || indRef != nil || attrs != nil {
		t.Fatalf("got %v, %v, %v; want nil page data", d, indRef, attrs)
	}
}

// TestPageTreeMutationRejectsRecursionDepth verifies page tree mutation respects recursion limits.
func TestPageTreeMutationRejectsRecursionDepth(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	maxDepth := xRefTable.MaxRecursionDepth()
	ir := types.NewIndirectRef(1, 0)
	attrs := InheritedPageAttrs{}
	pageCount := 0

	_, err := xRefTable.insertBlankPagesDepth(t.Context(), ir, &attrs, &pageCount, nil, nil, false, maxDepth+1, NewPageTreeVisit())
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	_, err = xRefTable.insertPagesDepth(t.Context(), ir, &pageCount, nil, maxDepth+1, NewPageTreeVisit())
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

// TestPageTreeRejectsCycle verifies page tree traversal rejects cycles.
func TestPageTreeRejectsCycle(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	ir := types.NewIndirectRef(1, 0)
	pageCount := 0

	_, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type": types.Name("Pages"),
		"Kids": types.Array{*ir},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = xRefTable.processPageTreeForPageNumber(t.Context(), ir, &pageCount, 1)
	if !errors.Is(err, ErrPageTreeCycle) {
		t.Fatalf("got %v, want ErrPageTreeCycle", err)
	}
}

// TestPageTreeRejectsDuplicateNode verifies page tree traversal rejects duplicate nodes.
func TestPageTreeRejectsDuplicateNode(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	root := types.NewIndirectRef(1, 0)
	child := types.NewIndirectRef(2, 0)
	pageCount := 0

	if _, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type": types.Name("Pages"),
		"Kids": types.Array{*child, *child},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := xRefTable.IndRefForObject(2, types.Dict{
		"Type": types.Name("Pages"),
	}); err != nil {
		t.Fatal(err)
	}

	_, err := xRefTable.processPageTreeForPageNumber(t.Context(), root, &pageCount, 1)
	if !errors.Is(err, ErrPageTreeDuplicate) {
		t.Fatalf("got %v, want ErrPageTreeDuplicate", err)
	}
}

func TestPageTreeOperationsRejectChildMissingType(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	root := types.NewIndirectRef(1, 0)
	child := types.NewIndirectRef(2, 0)
	pageCount := 0

	if _, err := xRefTable.IndRefForObject(1, types.Dict{
		"Type": types.Name("Pages"),
		"Kids": types.Array{*child},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := xRefTable.IndRefForObject(2, types.Dict{}); err != nil {
		t.Fatal(err)
	}

	_, err := xRefTable.processPageTreeForPageNumber(t.Context(), root, &pageCount, 1)
	if err == nil || !strings.Contains(err.Error(), "page tree kid obj#2: missing dict type") {
		t.Fatalf("got %v, want missing page node Type error", err)
	}

	_, err = xRefTable.insertPagesDepth(t.Context(), root, &pageCount, nil, 0, NewPageTreeVisit())
	if err == nil || !strings.Contains(err.Error(), "page tree kid obj#2: missing dict type") {
		t.Fatalf("got %v, want missing page node Type error", err)
	}
}

func TestDereferencePageNodeDictRejectsMissingDict(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())

	_, err := xRefTable.DereferencePageNodeDict(*types.NewIndirectRef(7, 0))
	if err == nil || !strings.Contains(err.Error(), "missing page node dict") {
		t.Fatalf("got %v, want missing page node dict error", err)
	}
}

// TestParseXRefStreamDictRejectsSizeLimit verifies xref stream parsing enforces size limits.
func TestParseXRefStreamDictRejectsSizeLimit(t *testing.T) {
	sd := types.StreamDict{
		Dict: types.Dict{
			"Size": types.Integer(3),
			"W":    types.Array{types.Integer(1), types.Integer(1), types.Integer(1)},
		},
	}

	_, err := ParseXRefStreamDictWithLimits(&sd, ResourceLimits{
		MaxObjectCount:       2,
		MaxXRefEntries:       10,
		MaxObjectStreamCount: 1,
		MaxObjectStreamFirst: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "Size") {
		t.Fatalf("got %v, want Size limit error", err)
	}
}

func TestParseXRefStreamDictRequiresDirectSize(t *testing.T) {
	sd := types.StreamDict{Dict: types.Dict{
		"Size": *types.NewIndirectRef(7, 0),
		"W":    types.Array{types.Integer(1), types.Integer(1), types.Integer(1)},
	}}

	_, err := ParseXRefStreamDictWithLimits(&sd, DefaultResourceLimits())
	if err == nil || !strings.Contains(err.Error(), `"Size" not available`) {
		t.Fatalf("got %v, want direct Size error", err)
	}
}

// TestParseXRefStreamDictAllocatesForIndexedEntries verifies a sparse xref stream reserves capacity for its checked
// Index entry count instead of its declared object-number range.
func TestParseXRefStreamDictAllocatesForIndexedEntries(t *testing.T) {
	sd := types.StreamDict{Dict: types.Dict{
		"Size":  types.Integer(1_000_000),
		"Index": types.Array{types.Integer(0), types.Integer(4)},
		"W":     types.Array{types.Integer(1), types.Integer(1), types.Integer(1)},
	}}
	limits := DefaultResourceLimits()
	limits.MaxObjectCount = 1_000_000
	limits.MaxXRefEntries = 4

	for _, tt := range []struct {
		name  string
		parse func(*types.StreamDict, ResourceLimits) (*types.XRefStreamDict, error)
	}{
		{"strict", ParseXRefStreamDictWithLimits},
		{"relaxed", ParseXRefStreamDictRelaxedWithLimits},
	} {
		t.Run(tt.name, func(t *testing.T) {
			xsd, err := tt.parse(&sd, limits)
			if err != nil {
				t.Fatal(err)
			}
			if len(xsd.Objects) != 4 || cap(xsd.Objects) != 4 {
				t.Fatalf("got xref object length %d and capacity %d, want 4 and 4", len(xsd.Objects), cap(xsd.Objects))
			}
		})
	}
}

// TestParseXRefStreamDictRepairsIndexSizeMismatch verifies relaxed parsing repairs
// an undersized Size entry without weakening strict parsing or resource limits.
func TestParseXRefStreamDictRepairsIndexSizeMismatch(t *testing.T) {
	sd := types.StreamDict{
		Dict: types.Dict{
			"Size":  types.Integer(6),
			"Index": types.Array{types.Integer(0), types.Integer(7)},
			"W":     types.Array{types.Integer(1), types.Integer(1), types.Integer(1)},
		},
	}

	limits := DefaultResourceLimits()
	if _, err := ParseXRefStreamDictWithLimits(&sd, limits); !errors.Is(err, ErrXRefStreamIndexSizeMismatch) {
		t.Fatalf("got %v, want ErrXRefStreamIndexSizeMismatch", err)
	}

	xsd, err := ParseXRefStreamDictRelaxedWithLimits(&sd, limits)
	if err != nil {
		t.Fatalf("parse relaxed xref stream dictionary: %v", err)
	}
	if xsd.Size != 7 || len(xsd.Objects) != 7 {
		t.Fatalf("got size %d and %d objects, want 7 and 7", xsd.Size, len(xsd.Objects))
	}

	limits.MaxObjectCount = 6
	if _, err := ParseXRefStreamDictRelaxedWithLimits(&sd, limits); err == nil {
		t.Fatal("expected repaired Size to respect MaxObjectCount")
	}
}

// TestObjectStreamDictRejectsLimits verifies object stream parsing enforces resource limits.
func TestObjectStreamDictRejectsLimits(t *testing.T) {
	sd := types.StreamDict{
		Dict: types.Dict{
			"Type":  types.Name("ObjStm"),
			"N":     types.Integer(3),
			"First": types.Integer(10),
		},
	}

	_, err := ObjectStreamDictWithLimits(&sd, ResourceLimits{
		MaxObjectStreamCount: 2,
		MaxObjectStreamFirst: 20,
	})
	if err == nil || !strings.Contains(err.Error(), "N") {
		t.Fatalf("got %v, want N limit error", err)
	}

	sd.Dict["N"] = types.Integer(2)
	_, err = ObjectStreamDictWithLimits(&sd, ResourceLimits{
		MaxObjectStreamCount: 2,
		MaxObjectStreamFirst: 9,
	})
	if err == nil || !strings.Contains(err.Error(), "First") {
		t.Fatalf("got %v, want First limit error", err)
	}
}

func TestObjectStreamDictWithResolvedIntegersPreservesIndirectEntries(t *testing.T) {
	nRef := *types.NewIndirectRef(7, 0)
	firstRef := *types.NewIndirectRef(8, 0)
	sd := types.StreamDict{Dict: types.Dict{
		"Type":  types.Name("ObjStm"),
		"N":     nRef,
		"First": firstRef,
	}}
	n := types.Integer(3)
	first := types.Integer(12)

	osd, err := ObjectStreamDictWithResolvedIntegers(&sd, DefaultResourceLimits(), &n, &first)
	if err != nil {
		t.Fatal(err)
	}
	if osd.ObjCount != 3 || osd.FirstObjOffset != 12 {
		t.Fatalf("got N=%d First=%d, want N=3 First=12", osd.ObjCount, osd.FirstObjOffset)
	}
	if sd.Dict["N"] != nRef || sd.Dict["First"] != firstRef {
		t.Fatalf("object stream dictionary was normalized in place: %v", sd.Dict)
	}
}

// TestDecodeNameHexValid verifies valid name hex escapes are decoded.
func TestDecodeNameHexValid(t *testing.T) {
	testcases := []struct {
		Input    string
		Expected string
	}{
		{"", ""},
		{"Foo", "Foo"},
		{"A#23", "A#"},
		// Examples from "7.3.5 Name Objects"
		{"Name1", "Name1"},
		{"ASomewhatLongerName", "ASomewhatLongerName"},
		{"A;Name_With-Various***Characters?", "A;Name_With-Various***Characters?"},
		{"1.2", "1.2"},
		{"$$", "$$"},
		{"@pattern", "@pattern"},
		{".notdef", ".notdef"},
		{"Lime#20Green", "Lime Green"},
		{"paired#28#29parentheses", "paired()parentheses"},
		{"The_Key_of_F#23_Minor", "The_Key_of_F#_Minor"},
		{"A#42", "AB"},
	}
	for _, tc := range testcases {
		decoded, err := decodeNameHexSequence(tc.Input)
		if err != nil {
			t.Errorf("decoding %s failed: %s", tc.Input, err)
		} else if decoded != tc.Expected {
			t.Errorf("expected %s when decoding %s, got %s", tc.Expected, tc.Input, decoded)
		}
	}
}

// TestDetectKeywords verifies keyword detection.
func TestDetectKeywords(t *testing.T) {
	msg := "detectKeywords"

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} endobj
	//                                    streamInd                        endInd
	//                                  -1 if absent                    -1 if absent

	//s := "5 0 obj\n<</Title (xxxxendobjxxxxx)\n/Parent 4 0 R\n/Dest [3 0 R /XYZ 0 738 0]>>\nendobj\n" //78

	s := "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % comment \n>>\nendobj\n\n2 0 obj\n"
	//    0....... ..1 .........2.........3.........4.........5..... ... .6
	endInd, _, err := DetectKeywords(t.Context(), s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd != 59 {
		t.Errorf("%s failed: want %d, got %d", msg, 59, endInd)
	}

	// negative test
	s = "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % endobject"
	endInd, _, err = DetectKeywords(t.Context(), s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd > 0 {
		t.Errorf("%s failed: want %d, got %d", msg, 0, endInd)
	}

}

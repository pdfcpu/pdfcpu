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
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

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

// TestParseObjectContextRejectsRecursionDepth verifies object parsing respects recursion limits.
func TestParseObjectContextRejectsRecursionDepth(t *testing.T) {
	s := "[[[1]]]"

	_, err := ParseObjectContext(t.Context(), &s, 0, 1)
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}
}

func TestParseObjectContextBoundsRelaxedFallback(t *testing.T) {
	s := strings.Repeat("<</Differences[24/breve/quotesingle0 obj\r", 34)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	_, err := ParseObjectContext(ctx, &s, 0)
	if !errors.Is(err, errArrayNotTerminated) {
		t.Fatalf("got %v, want errArrayNotTerminated", err)
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

	_, _, err := xRefTable.processPageTreeForPageDictDepth(ir, &attrs, &pageCount, 1, false, maxDepth+1, NewPageTreeVisit())
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	_, err = xRefTable.processPageTreeForPageNumberDepth(ir, &pageCount, 1, maxDepth+1, NewPageTreeVisit())
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

	d, indRef, attrs, err := xRefTable.PageDict(1, false)
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

	_, err := xRefTable.insertBlankPagesDepth(ir, &attrs, &pageCount, nil, nil, false, maxDepth+1, NewPageTreeVisit())
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want ErrMaxRecursionDepthExceeded", err)
	}

	_, err = xRefTable.insertPagesDepth(ir, &pageCount, nil, maxDepth+1, NewPageTreeVisit())
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

	_, err = xRefTable.processPageTreeForPageNumber(ir, &pageCount, 1)
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

	_, err := xRefTable.processPageTreeForPageNumber(root, &pageCount, 1)
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

	_, err := xRefTable.processPageTreeForPageNumber(root, &pageCount, 1)
	if err == nil || !strings.Contains(err.Error(), "page tree kid obj#2: missing dict type") {
		t.Fatalf("got %v, want missing page node Type error", err)
	}

	_, err = xRefTable.insertPagesDepth(root, &pageCount, nil, 0, NewPageTreeVisit())
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

// TestDetectNonEscaped verifies detection of non-escaped delimiters.
func TestDetectNonEscaped(t *testing.T) {
	testcases := []struct {
		input string
		want  int
	}{
		{"", -1},
		{" ( ", 1},
		{" \\( )", -1},
		{"\\(", -1},
		{"   \\(   ", -1},
		{"\\()(", 3},
		{" \\(\\((abc)", 5},
	}
	for _, tc := range testcases {
		got := detectNonEscaped(tc.input, "(")
		if tc.want != got {
			t.Errorf("%s, want: %d, got: %d", tc.input, tc.want, got)
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
	endInd, _, err := DetectKeywords(s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd != 59 {
		t.Errorf("%s failed: want %d, got %d", msg, 59, endInd)
	}

	// negative test
	s = "1 0 obj\n<<\n /Lang (en-endobject-stream-UK%)  % endobject"
	endInd, _, err = DetectKeywords(s)
	if err != nil {
		t.Errorf("%s failed: %v", msg, err)
	}
	if endInd > 0 {
		t.Errorf("%s failed: want %d, got %d", msg, 0, endInd)
	}

}

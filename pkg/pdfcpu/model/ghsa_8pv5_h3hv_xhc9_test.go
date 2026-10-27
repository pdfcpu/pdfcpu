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

// GHSA-8pv5-h3hv-xhc9: shared acyclic reference graphs must not make object comparison take exponential time.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-8pv5-h3hv-xhc9

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func ghsa8pv5SharedChains(depth int, unequal bool) (*XRefTable, types.IndirectRef, types.IndirectRef) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	const xBase = 100
	yBase := xBase + depth + 1
	for i := range depth {
		xNext := *types.NewIndirectRef(xBase+i+1, 0)
		yNext := *types.NewIndirectRef(yBase+i+1, 0)
		xRefTable.Table[xBase+i] = NewXRefTableEntryGen0(types.Array{xNext, xNext})
		xRefTable.Table[yBase+i] = NewXRefTableEntryGen0(types.Array{yNext, yNext})
	}
	xRefTable.Table[xBase+depth] = NewXRefTableEntryGen0(types.Integer(1))
	yLeaf := types.Integer(1)
	if unequal {
		yLeaf = 2
	}
	xRefTable.Table[yBase+depth] = NewXRefTableEntryGen0(yLeaf)
	return xRefTable, *types.NewIndirectRef(xBase, 0), *types.NewIndirectRef(yBase, 0)
}

func TestGHSA_8pv5_h3hv_xhc9_SharedGraphComparisonCompletes(t *testing.T) {
	xRefTable, x, y := ghsa8pv5SharedChains(32, false)
	c, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	equal, err := EqualObjects(c, x, y, xRefTable, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatal("expected shared graphs to compare equal")
	}
}

func TestGHSA_8pv5_h3hv_xhc9_SharedGraphMismatchIsNotMemoizedAsEqual(t *testing.T) {
	xRefTable, x, y := ghsa8pv5SharedChains(16, true)

	equal, err := EqualObjects(t.Context(), x, y, xRefTable, nil)
	if err != nil {
		t.Fatal(err)
	}
	if equal {
		t.Fatal("expected unequal shared graphs")
	}
}

func TestGHSA_8pv5_h3hv_xhc9_SeededPairIsRecognized(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	xRefTable.Table[1] = NewXRefTableEntryGen0(types.Integer(1))
	xRefTable.Table[2] = NewXRefTableEntryGen0(types.Integer(2))
	x := *types.NewIndirectRef(1, 0)
	y := *types.NewIndirectRef(2, 0)

	// A seeded pair represents an active ancestor comparison, so encountering it closes a cycle.
	equal, err := EqualObjects(t.Context(), x, y, xRefTable, []int{2, 1})
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatal("expected the seeded pair to stop repeated comparison")
	}
}

func TestGHSA_8pv5_h3hv_xhc9_EqualCyclesStayEqual(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	const cycleSize = 5
	for i := range cycleSize {
		xRefTable.Table[100+i] = NewXRefTableEntryGen0(
			types.Array{*types.NewIndirectRef(100+(i+1)%cycleSize, 0)})
		xRefTable.Table[200+i] = NewXRefTableEntryGen0(
			types.Array{*types.NewIndirectRef(200+(i+1)%cycleSize, 0)})
	}
	x := *types.NewIndirectRef(100, 0)
	y := *types.NewIndirectRef(200, 0)

	equal, err := EqualObjects(t.Context(), x, y, xRefTable, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatal("expected structurally equal cycles to compare equal")
	}
}

func ghsa8pv5ProductGraph(xSize, ySize int) (*XRefTable, types.IndirectRef, types.IndirectRef) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	const xBase = 100
	yBase := xBase + xSize
	for i := range xSize {
		self := *types.NewIndirectRef(xBase+i, 0)
		next := *types.NewIndirectRef(xBase+(i+1)%xSize, 0)
		xRefTable.Table[xBase+i] = NewXRefTableEntryGen0(types.Array{next, self})
	}
	for i := range ySize {
		self := *types.NewIndirectRef(yBase+i, 0)
		next := *types.NewIndirectRef(yBase+(i+1)%ySize, 0)
		xRefTable.Table[yBase+i] = NewXRefTableEntryGen0(types.Array{self, next})
	}
	return xRefTable, *types.NewIndirectRef(xBase, 0), *types.NewIndirectRef(yBase, 0)
}

func TestGHSA_8pv5_h3hv_xhc9_ComparisonWorkIsBounded(t *testing.T) {
	xRefTable, x, y := ghsa8pv5ProductGraph(4, 5)
	var buf bytes.Buffer
	pdfcpuLog.SetOptimizeLogger(log.New(&buf, "", 0))
	defer pdfcpuLog.SetOptimizeLogger(nil)

	equal, err := EqualObjects(t.Context(), x, y, xRefTable, nil)
	if err != nil {
		t.Fatal(err)
	}
	if equal {
		t.Fatal("expected a comparison beyond the work limit to be treated as not equal")
	}
	if !strings.Contains(buf.String(), "indirect-reference pair budget") {
		t.Fatalf("missing pair-budget diagnostic: %q", buf.String())
	}
}

func TestGHSA_8pv5_h3hv_xhc9_ComparisonDepthIsBounded(t *testing.T) {
	conf := NewDefaultConfiguration()
	conf.Limits.MaxRecursionDepth = 2
	xRefTable := newXRefTable(conf)
	var buf bytes.Buffer
	pdfcpuLog.SetOptimizeLogger(log.New(&buf, "", 0))
	defer pdfcpuLog.SetOptimizeLogger(nil)
	var x, y types.Object = types.Integer(1), types.Integer(1)
	for range 3 {
		x = types.Array{x}
		y = types.Array{y}
	}

	equal, err := EqualObjects(t.Context(), x, y, xRefTable, nil)
	if err != nil {
		t.Fatal(err)
	}
	if equal {
		t.Fatal("expected a comparison beyond the depth limit to be treated as not equal")
	}
	if !strings.Contains(buf.String(), "recursion depth 3 exceeds limit 2") {
		t.Fatalf("missing recursion-depth diagnostic: %q", buf.String())
	}
}

func TestGHSA_8pv5_h3hv_xhc9_UnrelatedDepthErrorsArePreserved(t *testing.T) {
	xRefTable := newXRefTable(NewDefaultConfiguration())
	osd := types.NewObjectStreamDict()
	decodeStream := func(*types.StreamDict) error {
		return CheckRecursionDepth("fixture", 2, 1)
	}
	xRefTable.Table[1] = NewXRefTableEntryGen0(
		types.NewLazyObjectStreamObjectWithDecoder(osd, 0, -1, nil, decodeStream))
	xRefTable.Table[2] = NewXRefTableEntryGen0(types.Integer(1))

	x := *types.NewIndirectRef(1, 0)
	y := *types.NewIndirectRef(2, 0)
	_, err := EqualObjects(t.Context(), x, y, xRefTable, nil)
	if !errors.Is(err, ErrMaxRecursionDepthExceeded) {
		t.Fatalf("got %v, want underlying recursion depth error", err)
	}
}

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

package primitives

// GHSA-cphg-88q6-whwh: the rows and columns of all tables in a create template share one bound, and positioning a cell
// does not depend on its column, so a small template cannot make create render or allocate without limit.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-cphg-88q6-whwh

import (
	"math"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestGHSA_cphg_88q6_whwh_TableLineBudget(t *testing.T) {
	pdf := &PDF{}
	if err := pdf.chargeTableLines(maxTableLines-1, 1); err != nil {
		t.Fatalf("budget at the limit: %v", err)
	}
	if err := pdf.chargeTableLines(1, 1); err == nil {
		t.Fatal("budget beyond the limit was accepted")
	}

	for _, tc := range []struct{ rows, cols int }{
		{maxTableLines, 1},
		{1, maxTableLines},
		{math.MaxInt, 1},
		{1, math.MaxInt},
		{math.MaxInt, math.MaxInt},
	} {
		pdf := &PDF{}
		if err := pdf.chargeTableLines(tc.rows, tc.cols); err == nil {
			t.Fatalf("rows=%d cols=%d: accepted", tc.rows, tc.cols)
		}
		if pdf.tableLines != 0 {
			t.Fatalf("rows=%d cols=%d: rejected table charged %d lines", tc.rows, tc.cols, pdf.tableLines)
		}
	}
}

func TestGHSA_cphg_88q6_whwh_CellPositionsUseColumnOffsets(t *testing.T) {
	tbl := &Table{LineHeight: 10}
	r := types.RectForWidthAndHeight(5, 0, 60, 10)
	colOffsets := columnOffsets([]float64{10, 20, 30})
	for col, want := range []float64{5, 15, 35} {
		if x, _ := tbl.headerCellLowerLeft(r, colOffsets, 0, col); x != want {
			t.Fatalf("column %d: got x %.2f, want %.2f", col, x, want)
		}
	}

	// The widest accepted table positions every column; summing the widths per cell made this quadratic.
	colWidths := make([]float64, maxTableLines-1)
	for i := range colWidths {
		colWidths[i] = 1
	}
	start := time.Now()
	colOffsets = columnOffsets(colWidths)
	for col := range colOffsets {
		tbl.headerCellLowerLeft(r, colOffsets, 0, col)
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("positioning %d columns took %v", len(colOffsets), elapsed)
	}
}

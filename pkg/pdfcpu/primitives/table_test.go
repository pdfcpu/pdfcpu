/*
Copyright 2026 The pdf Authors.

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

import (
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestTableCellLowerLeft(t *testing.T) {
	for _, tt := range []struct {
		name    string
		table   Table
		headerY float64
		valueY  []float64
	}{
		{
			name:   "NoHeader",
			table:  Table{Rows: 2, LineHeight: 10},
			valueY: []float64{10, 0},
		},
		{
			name:    "CustomHeader",
			table:   Table{Rows: 2, LineHeight: 10, Header: &TableHeader{LineHeight: 15}},
			headerY: 20,
			valueY:  []float64{10, 0},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := types.RectForWidthAndHeight(0, 0, 20, tt.table.Height())
			colWidths := []float64{20}

			if tt.table.Header != nil {
				_, y := tt.table.headerCellLowerLeft(r, colWidths, 0, 0)
				if y != tt.headerY {
					t.Fatalf("header y: got %.2f want %.2f", y, tt.headerY)
				}
			}

			for row, wantY := range tt.valueY {
				_, y := tt.table.valueCellLowerLeft(r, colWidths, 0, row, 0)
				if y != wantY {
					t.Fatalf("value row %d y: got %.2f want %.2f", row, y, wantY)
				}
			}
		})
	}
}

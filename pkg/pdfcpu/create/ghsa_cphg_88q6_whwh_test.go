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

package create

// GHSA-cphg-88q6-whwh: table rows and columns in a create template must be bounded across the document before create
// allocates or renders per row or column, and a table with an invalid anchor must not skip that validation.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-cphg-88q6-whwh

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"
)

// ghsaCphgTable returns a table template with oddCol, evenCol and grid set, so every row and column is drawn.
func ghsaCphgTable(rows, cols int, extra string) string {
	return fmt.Sprintf(`{"pos": [50, 50], "width": 400, "rows": %d, "cols": %d, "lheight": 20%s,
		"font": {"name": "Helvetica", "size": 10, "col": "Black"}, "oddCol": "#00ff00", "evenCol": "#ff0000",
		"grid": true}`, rows, cols, extra)
}

// ghsaCphgTemplate returns a create template with one page per entry in pages, each holding the given tables.
func ghsaCphgTemplate(pages ...[]string) string {
	pp := make([]string, len(pages))
	for i, tables := range pages {
		pp[i] = fmt.Sprintf(`"%d": {"content": {"table": [%s]}}`, i+1, strings.Join(tables, ", "))
	}
	return `{"paper": "A4", "pages": {` + strings.Join(pp, ", ") + `}}`
}

func TestGHSA_cphg_88q6_whwh_CreateRejectsLargeTables(t *testing.T) {
	for _, tc := range []struct {
		name     string
		template string
		want     string
	}{
		{"rows", ghsaCphgTemplate([]string{ghsaCphgTable(100_000, 1, "")}), "at most 100000 per document"},
		{"cols", ghsaCphgTemplate([]string{ghsaCphgTable(1, 100_000, "")}), "at most 100000 per document"},
		{"max", ghsaCphgTemplate([]string{ghsaCphgTable(math.MaxInt, math.MaxInt, "")}), "at most 100000 per document"},
		{"tables", ghsaCphgTemplate([]string{ghsaCphgTable(50_000, 1, ""), ghsaCphgTable(50_000, 1, "")}),
			"at most 100000 per document"},
		{"pages", ghsaCphgTemplate([]string{ghsaCphgTable(50_000, 1, "")}, []string{ghsaCphgTable(50_000, 1, "")}),
			"at most 100000 per document"},
		// An anchor next to "pos" is invalid; validation used to stop there and render the table unchecked.
		{"anchor", ghsaCphgTemplate([]string{ghsaCphgTable(1_000_000, 4, `, "anchor": "nowhere"`)}),
			`please supply table "pos" or "anchor"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newCreateTestContext(t)
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			err := FromJSON(t.Context(), ctx, strings.NewReader(tc.template))
			runtime.ReadMemStats(&after)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error containing %q", err, tc.want)
			}
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 8<<20 {
				t.Fatalf("allocated %d bytes before rejecting the template", allocated)
			}
		})
	}
}

func TestGHSA_cphg_88q6_whwh_CreateKeepsTablesWithinBudget(t *testing.T) {
	ctx := newCreateTestContext(t)
	template := ghsaCphgTemplate([]string{ghsaCphgTable(5, 4, "")}, []string{ghsaCphgTable(1_000, 10, "")})
	if err := FromJSON(t.Context(), ctx, strings.NewReader(template)); err != nil {
		t.Fatal(err)
	}
	if ctx.PageCount != 2 {
		t.Fatalf("got %d pages, want 2", ctx.PageCount)
	}
}

func TestGHSA_cphg_88q6_whwh_RevalidationDoesNotDoubleCount(t *testing.T) {
	ctx := newCreateTestContext(t)
	pdf, err := parseFromJSON(t.Context(), ctx, []byte(ghsaCphgTemplate([]string{ghsaCphgTable(60_000, 1, "")})))
	if err != nil {
		t.Fatal(err)
	}
	if err := pdf.Validate(t.Context()); err != nil {
		t.Fatalf("second validation: %v", err)
	}
}

func TestGHSA_cphg_88q6_whwh_CreateRendersWidestAcceptedTable(t *testing.T) {
	// One row and 99,999 columns, each with a header and a row value: the largest workload the budget accepts.
	const cols = 99_999
	cells := strings.TrimSuffix(strings.Repeat(`"a",`, cols), ",")
	table := fmt.Sprintf(`{"pos": [50, 50], "width": 1e8, "rows": 1, "cols": %d, "lheight": 20,
		"font": {"name": "Helvetica", "size": 10, "col": "Black"}, "header": {"values": [%s]}, "values": [[%s]]}`,
		cols, cells, cells)
	ctx := newCreateTestContext(t)
	if err := FromJSON(t.Context(), ctx, strings.NewReader(ghsaCphgTemplate([]string{table}))); err != nil {
		t.Fatal(err)
	}
}

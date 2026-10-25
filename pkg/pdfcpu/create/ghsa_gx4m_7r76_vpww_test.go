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

// GHSA-gx4m-7r76-vpww: a page number in a create template must be bounded before blank pages are allocated and
// rendered, so a few bytes of JSON cannot select gigabytes of memory.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-gx4m-7r76-vpww

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func ghsaGx4mTemplate(pageNrs ...string) string {
	pages := make([]string, len(pageNrs))
	for i, pageNr := range pageNrs {
		pages[i] = fmt.Sprintf(`%q: {"content": {}}`, pageNr)
	}
	return `{"pages": {` + strings.Join(pages, ", ") + `}}`
}

// TestGHSA_gx4m_7r76_vpww_CreateRejectsPageNumbers verifies invalid and excessive page numbers fail before
// allocation.
func TestGHSA_gx4m_7r76_vpww_CreateRejectsPageNumbers(t *testing.T) {
	for _, pageNr := range []string{"0", "-1", "1001", "100000"} {
		t.Run(pageNr, func(t *testing.T) {
			ctx := newCreateTestContext(t)
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err := FromJSON(t.Context(), ctx, strings.NewReader(ghsaGx4mTemplate("1", pageNr)))
			runtime.ReadMemStats(&after)
			if err == nil || !strings.Contains(err.Error(), "invalid page number: "+pageNr) {
				t.Fatalf("got %v, want invalid page number", err)
			}
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 8<<20 {
				t.Fatalf("allocated %d bytes before rejecting page %s", allocated, pageNr)
			}
		})
	}
}

// TestGHSA_gx4m_7r76_vpww_CreateKeepsSparsePages verifies gaps up to the limit still become blank pages.
func TestGHSA_gx4m_7r76_vpww_CreateKeepsSparsePages(t *testing.T) {
	for _, tc := range []struct {
		pageNrs []string
		want    int
	}{
		{[]string{"1", "3"}, 3},
		{[]string{"1000"}, 1000},
	} {
		ctx := newCreateTestContext(t)
		if err := FromJSON(t.Context(), ctx, strings.NewReader(ghsaGx4mTemplate(tc.pageNrs...))); err != nil {
			t.Fatal(err)
		}
		if ctx.PageCount != tc.want {
			t.Fatalf("pages %v: got %d pages, want %d", tc.pageNrs, ctx.PageCount, tc.want)
		}
	}
}

// TestGHSA_gx4m_7r76_vpww_CreateLimitsPagesAddedToExistingDocument verifies the limit counts pages beyond the existing
// document, so pages of a larger document stay addressable.
func TestGHSA_gx4m_7r76_vpww_CreateLimitsPagesAddedToExistingDocument(t *testing.T) {
	ctx := newCreateTestContext(t)
	if err := FromJSON(t.Context(), ctx, strings.NewReader(ghsaGx4mTemplate("5"))); err != nil {
		t.Fatal(err)
	}
	if ctx.PageCount != 5 {
		t.Fatalf("got %d pages, want 5", ctx.PageCount)
	}
	err := FromJSON(t.Context(), ctx, strings.NewReader(ghsaGx4mTemplate("1006")))
	if err == nil || !strings.Contains(err.Error(), "at most 1000 pages may be added to 5 existing pages") {
		t.Fatalf("got %v, want limit error", err)
	}
	if err := FromJSON(t.Context(), ctx, strings.NewReader(ghsaGx4mTemplate("1005"))); err != nil {
		t.Fatal(err)
	}
	if ctx.PageCount != 1005 {
		t.Fatalf("got %d pages, want 1005", ctx.PageCount)
	}
}

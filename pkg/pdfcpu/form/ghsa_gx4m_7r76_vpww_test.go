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

package form

// GHSA-gx4m-7r76-vpww: a page number in form fill data must be checked against the page count before any
// per-page allocation, so a few bytes of JSON cannot select gigabytes of memory.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-gx4m-7r76-vpww

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ghsaGx4mOnePageContext(t *testing.T) *model.Context {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	var offsets []int
	for _, body := range []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>",
	} {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}
	xref := b.Len()
	b.WriteString("xref\n0 4\n0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)

	ctx, err := pdfcpu.Read(t.Context(), bytes.NewReader(b.Bytes()), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.EnsurePageCount(); err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestGHSA_gx4m_7r76_vpww_FillRejectsPageNumbersOutsideDocument verifies out-of-range page numbers fail before
// allocation, and that page numbers below 1 are no longer ignored.
func TestGHSA_gx4m_7r76_vpww_FillRejectsPageNumbersOutsideDocument(t *testing.T) {
	for _, pageNr := range []string{"0", "-1", "2", "10000000"} {
		t.Run(pageNr, func(t *testing.T) {
			ctx := ghsaGx4mOnePageContext(t)
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, err := addImages(t.Context(), ctx, map[string]*Page{pageNr: {}})
			runtime.ReadMemStats(&after)
			if err == nil || !strings.Contains(err.Error(), "page number out of range 1..1") {
				t.Fatalf("got %v, want page number out of range", err)
			}
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 8<<20 {
				t.Fatalf("allocated %d bytes before rejecting page %s", allocated, pageNr)
			}
		})
	}
}

// TestGHSA_gx4m_7r76_vpww_FillAcceptsExistingPage verifies page numbers within the document still work.
func TestGHSA_gx4m_7r76_vpww_FillAcceptsExistingPage(t *testing.T) {
	pages, err := addImages(t.Context(), ghsaGx4mOnePageContext(t), map[string]*Page{"1": {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || pages[0] == nil {
		t.Fatalf("got %v, want one page", pages)
	}
}

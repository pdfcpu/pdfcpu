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

package api

// GHSA-p2r6-9jcq-fp7v: splitting along bookmarks must check a bookmark's page span against the page count before
// listing its pages. Otherwise an outline destination with a huge page number makes split allocate one slice entry per
// page number, or panic.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-p2r6-9jcq-fp7v

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const ghsaP2r6AllocLimit = 8 << 20

// ghsaP2r6PDF writes a document with pageCount pages and one top-level outline item per destination.
func ghsaP2r6PDF(pageCount int, dests ...string) []byte {
	const firstPage = 4
	firstItem := firstPage + pageCount

	var objs []string
	kids := make([]string, pageCount)
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", firstPage+i)
	}
	objs = append(objs,
		"<</Type/Catalog/Pages 2 0 R/Outlines 3 0 R>>",
		fmt.Sprintf("<</Type/Pages/Kids[%s]/Count %d>>", strings.Join(kids, " "), pageCount),
		fmt.Sprintf("<</Type/Outlines/First %d 0 R/Last %d 0 R/Count %d>>", firstItem, firstItem+len(dests)-1, len(dests)),
	)
	for range pageCount {
		objs = append(objs, "<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Resources<<>>>>")
	}
	for i, dest := range dests {
		item := fmt.Sprintf("<</Title(b%d)/Parent 3 0 R/Dest[%s/Fit]", i+1, dest)
		if i > 0 {
			item += fmt.Sprintf("/Prev %d 0 R", firstItem+i-1)
		}
		if i < len(dests)-1 {
			item += fmt.Sprintf("/Next %d 0 R", firstItem+i+1)
		}
		objs = append(objs, item+">>")
	}

	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offs := make([]int, len(objs))
	for i, o := range objs {
		offs[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func ghsaP2r6Alloc(f func() error) (uint64, error) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	err := f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc, err
}

func TestGHSA_p2r6_9jcq_fp7v_SplitRejectsBookmarkPagesBeyondDocument(t *testing.T) {
	// math.MaxInt is the largest page number on every platform; before the fix it made the page slice length overflow.
	for _, n := range []int{10_000_000, math.MaxInt} {
		data := ghsaP2r6PDF(1, "1", fmt.Sprint(n))
		want := fmt.Sprintf("extract page span 1-%d: invalid page number: %d outside 1..1", n-1, n-1)

		t.Run(fmt.Sprintf("raw/%d", n), func(t *testing.T) {
			alloc, err := ghsaP2r6Alloc(func() error {
				_, err := SplitRaw(context.Background(), bytes.NewReader(data), 0, model.NewDefaultConfiguration())
				return err
			})
			if !errors.Is(err, pdfcpu.ErrInvalidPageNumber) || !strings.Contains(err.Error(), want) {
				t.Fatalf("got %v, want error containing %q", err, want)
			}
			if alloc > ghsaP2r6AllocLimit {
				t.Fatalf("split allocated %d bytes, want at most %d", alloc, ghsaP2r6AllocLimit)
			}
		})

		t.Run(fmt.Sprintf("files/%d", n), func(t *testing.T) {
			alloc, err := ghsaP2r6Alloc(func() error {
				return Split(context.Background(), bytes.NewReader(data), t.TempDir(), "in.pdf", 0, model.NewDefaultConfiguration())
			})
			if !errors.Is(err, pdfcpu.ErrInvalidPageNumber) || !strings.Contains(err.Error(), want) {
				t.Fatalf("got %v, want error containing %q", err, want)
			}
			if alloc > ghsaP2r6AllocLimit {
				t.Fatalf("split allocated %d bytes, want at most %d", alloc, ghsaP2r6AllocLimit)
			}
		})
	}
}

func TestGHSA_p2r6_9jcq_fp7v_SplitAlongValidBookmarks(t *testing.T) {
	data := ghsaP2r6PDF(3, "4 0 R", "6 0 R")

	spans, err := SplitRaw(context.Background(), bytes.NewReader(data), 0, model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, ps := range spans {
		got = append(got, fmt.Sprintf("%d-%d", ps.From, ps.Thru))
	}
	if strings.Join(got, ",") != "1-2,3-3" {
		t.Fatalf("got spans %v, want [1-2 3-3]", got)
	}
}

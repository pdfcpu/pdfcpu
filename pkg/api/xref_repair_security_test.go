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

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func freeHeadAtOnePDF(size int) []byte {
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	off2 := pdf.Len()
	pdf.WriteString("2 0 obj\n<< /Type /Catalog /Pages 3 0 R >>\nendobj\n")
	off3 := pdf.Len()
	pdf.WriteString("3 0 obj\n<< /Type /Pages /Count 0 /Kids [] >>\nendobj\n")
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n1 3\n0000000000 00000 f\n%010d 00000 n\n%010d 00000 n\n", off2, off3)
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 2 0 R >>\nstartxref\n%d\n%%%%EOF\n", size, xref)
	return pdf.Bytes()
}

func TestReadContextRepairsFreeHeadWithoutMissingEntries(t *testing.T) {
	for _, mode := range []struct {
		name string
		mode int
	}{
		{"strict", model.ValidationStrict},
		{"relaxed", model.ValidationRelaxed},
	} {
		t.Run(mode.name, func(t *testing.T) {
			conf := model.NewStatelessConfiguration()
			conf.ValidationMode = mode.mode
			conf.Limits.MaxObjectCount = 10
			conf.Limits.MaxXRefEntries = 3
			ctx, err := ReadContext(t.Context(), bytes.NewReader(freeHeadAtOnePDF(4)), conf)
			if err != nil {
				t.Fatal(err)
			}
			if len(ctx.Table) != 3 {
				t.Fatalf("got %d xref entries, want 3", len(ctx.Table))
			}
			for objNr := 0; objNr < 3; objNr++ {
				if ctx.Table[objNr] == nil {
					t.Fatalf("missing xref entry for object %d", objNr)
				}
			}
		})
	}
}

// TestReadContextRepairsInvalidTraditionalTrailerSize preserves strict rejection and bounds relaxed repair.
func TestReadContextRepairsInvalidTraditionalTrailerSize(t *testing.T) {
	for _, size := range []int{0, -1, 100000} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(fmt.Sprintf("size_%d/mode_%d", size, mode), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.ValidationMode = mode
				conf.Limits.MaxInputBytes = 1024
				conf.Limits.MaxObjectCount = 10
				conf.Limits.MaxXRefEntries = 3
				ctx, err := ReadContext(t.Context(), bytes.NewReader(freeHeadAtOnePDF(size)), conf)
				if mode == model.ValidationStrict {
					if err == nil || !strings.Contains(err.Error(), "trailer Size") {
						t.Fatalf("got %v, want trailer Size error", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if *ctx.Size != ctx.MaxObjNr+1 || len(ctx.Table) != 3 || *ctx.Size > 10 {
					t.Fatalf("unexpected repaired size/table: %d/%d", *ctx.Size, len(ctx.Table))
				}
				assertTrailerSizeRepairNotice(t, ctx, size)
			})
		}
	}
}

func assertTrailerSizeRepairNotice(t *testing.T, ctx *model.Context, size int) {
	t.Helper()
	count := 0
	for _, notice := range ctx.ValidationReport().Notices() {
		if strings.Contains(notice.Message, fmt.Sprintf("trailer Size %d", size)) {
			count++
			if notice.Phase != model.NoticePhaseRead || notice.Disposition != model.NoticeRepaired || notice.Cause == nil {
				t.Fatalf("unexpected repair notice: %+v", notice)
			}
		}
	}
	if count != 1 {
		t.Fatalf("got %d trailer Size repair notices, want 1", count)
	}
}

// TestTrailerSizeRepairPreservesXRefLimits rejects actual oversized xref ranges and entry counts.
func TestTrailerSizeRepairPreservesXRefLimits(t *testing.T) {
	for _, objectRange := range []bool{false, true} {
		conf := model.NewStatelessConfiguration()
		conf.Limits.MaxObjectCount = 10
		conf.Limits.MaxXRefEntries = 3
		want := "xref entry count exceeds limit"
		if objectRange {
			conf.Limits.MaxObjectCount = 3
			want = "xref subsection object range exceeds limit"
		} else {
			conf.Limits.MaxXRefEntries = 2
		}
		_, err := ReadContext(t.Context(), bytes.NewReader(freeHeadAtOnePDF(100000)), conf)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("got %v, want %s", err, want)
		}
	}
}

// TestTrailerSizeRepairReadsPreviousTrailer preserves incremental reading after discarding the newest invalid Size.
func TestTrailerSizeRepairReadsPreviousTrailer(t *testing.T) {
	for _, oldSize := range []int{4, 0} {
		pdf := freeHeadAtOnePDF(oldSize)
		previous := bytes.Index(pdf, []byte("xref\n"))
		offset := len(pdf)
		update := fmt.Sprintf("xref\n0 1\n0000000000 65535 f \ntrailer\n<< /Size -1 /Root 2 0 R /Prev %d >>\nstartxref\n%d\n%%%%EOF\n", previous, offset)
		conf := model.NewStatelessConfiguration()
		conf.Limits.MaxObjectCount = 10
		ctx, err := ReadContext(t.Context(), bytes.NewReader(append(pdf, update...)), conf)
		if err != nil {
			t.Fatal(err)
		}
		if *ctx.Size != 4 || ctx.Root.ObjectNumber.Value() != 2 {
			t.Fatalf("unexpected size/root: %d/%v", *ctx.Size, ctx.Root)
		}
		assertTrailerSizeRepairNotice(t, ctx, -1)
		if oldSize == 0 {
			assertTrailerSizeRepairNotice(t, ctx, 0)
		}
	}
}

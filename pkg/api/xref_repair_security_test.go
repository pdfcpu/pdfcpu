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

func TestReadContextRejectsOversizedTraditionalTrailerSize(t *testing.T) {
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
			conf.Limits.MaxInputBytes = 1024
			conf.Limits.MaxObjectCount = 10
			conf.Limits.MaxXRefEntries = 3
			_, err := ReadContext(t.Context(), bytes.NewReader(freeHeadAtOnePDF(100000)), conf)
			if err == nil || !strings.Contains(err.Error(), "trailer Size 100000 exceeds limit 10") {
				t.Fatalf("got %v, want trailer Size limit error", err)
			}
		})
	}
}

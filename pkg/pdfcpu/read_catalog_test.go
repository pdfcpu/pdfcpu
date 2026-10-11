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

package pdfcpu

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestReadEffectiveVersion checks catalog upgrades, lower catalog versions and reader compatibility.
func TestReadEffectiveVersion(t *testing.T) {
	for _, tc := range []struct {
		header, catalog string
		want            model.Version
	}{
		{"1.4", "", model.V14},
		{"1.3", "2.0", model.V20},
		{"1.7", "", model.V17},
		{"1.4", "2.0", model.V20},
		{"2.0", "1.7", model.V20},
	} {
		for _, reader15 := range []bool{false, true} {
			t.Run(fmt.Sprintf("header%s/catalog%s/reader15=%t", tc.header, tc.catalog, reader15), func(t *testing.T) {
				pdf := directInfoPDF20(tc.header, tc.catalog != "")
				if tc.catalog != "" {
					pdf = bytes.Replace(pdf, []byte("/Version/2.0"), []byte("/Version/"+tc.catalog), 1)
				}
				conf := model.NewDefaultConfiguration()
				conf.Reader15 = reader15
				ctx, err := Read(t.Context(), bytes.NewReader(pdf), conf)
				if !reader15 && tc.want >= model.V15 {
					if err == nil || !strings.Contains(err.Error(), "incompatible version") {
						t.Fatalf("got %v, want reader compatibility error", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if got := ctx.XRefTable.Version(); got != tc.want {
					t.Fatalf("got version %s, want %s", got, tc.want)
				}
				if got := newPDFInfo(ctx, "test.pdf").Version; got != tc.want.String() {
					t.Fatalf("reported version %s, want %s", got, tc.want)
				}
			})
		}
	}
}

func catalogEndobjPDF(omitEndobj bool) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	objects := []string{
		"<< /Type /Pages /Kids [2 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 1 0 R /MediaBox [0 0 100 100] /Resources <<>> >>",
		"<< /Type /Catalog /Pages 1 0 R >>",
	}
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\n", i+1, object)
		if i != len(objects)-1 || !omitEndobj {
			buf.WriteString("endobj\n")
		}
	}
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 3 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return buf.Bytes()
}

// TestReadCatalogEndobj verifies page-count access rejects a catalog that cannot be parsed after reading.
func TestReadCatalogEndobj(t *testing.T) {
	for _, omitEndobj := range []bool{false, true} {
		name := "ValidCatalog"
		if omitEndobj {
			name = "MissingCatalogEndobj"
		}
		t.Run(name, func(t *testing.T) {
			ctx, err := Read(t.Context(), bytes.NewReader(catalogEndobjPDF(omitEndobj)), model.NewStatelessConfiguration())
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if ctx == nil {
				t.Fatal("Read returned a nil context without an error")
			}
			defer func() {
				if p := recover(); p != nil {
					t.Fatalf("EnsurePageCount panicked: %v", p)
				}
			}()
			err = ctx.EnsurePageCount()
			if omitEndobj {
				if err == nil || err.Error() != "missing root dict" {
					t.Fatalf("EnsurePageCount: got %v, want missing root dict", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("EnsurePageCount: %v", err)
			}
			if ctx.PageCount != 1 {
				t.Fatalf("page count: got %d, want 1", ctx.PageCount)
			}
		})
	}
}

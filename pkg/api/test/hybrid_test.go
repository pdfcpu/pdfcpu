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

package test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func appendObject(buf *bytes.Buffer, objNr int, body string) int {
	offset := buf.Len()
	fmt.Fprintf(buf, "%d 0 obj\n%s\nendobj\n", objNr, body)
	return offset
}

func appendXRefStream(buf *bytes.Buffer) int {
	offset := buf.Len()
	var entry bytes.Buffer
	entry.WriteByte(1)
	entry.Write([]byte{byte(offset >> 24), byte(offset >> 16), byte(offset >> 8), byte(offset)})
	entry.Write([]byte{0, 0})
	fmt.Fprintf(buf, "4 0 obj\n<< /Type /XRef /Size 5 /Root 1 0 R /W [1 4 2] /Index [4 1] /Length %d >>\nstream\n", entry.Len())
	buf.Write(entry.Bytes())
	fmt.Fprintf(buf, "\nendstream\nendobj\n")
	return offset
}

func minimalHybridPDF() []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.5\n%\xFF\xFF\xFF\xFF\n")

	// Build a minimal hybrid-reference PDF: the visible classic xref section
	// contains regular objects and its trailer points /XRefStm at a hidden xref
	// stream. This exercises hybrid input reading without requiring a fixture.
	offsets := []int{0}
	offsets = append(offsets, appendObject(&buf, 1, "<< /Type /Catalog /Pages 2 0 R >>"))
	offsets = append(offsets, appendObject(&buf, 2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"))
	offsets = append(offsets, appendObject(&buf, 3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>"))
	xRefStreamOffset := appendXRefStream(&buf)

	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 4\n")
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size 4 /Root 1 0 R /XRefStm %d >>\nstartxref\n%d\n%%%%EOF\n", xRefStreamOffset, xrefOffset)

	return buf.Bytes()
}

func hybridConf(writeXRefStream bool) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.WriteObjectStream = writeXRefStream
	conf.WriteXRefStream = writeXRefStream
	return conf
}

func TestReadAndWatermarkHybridPDF(t *testing.T) {
	in := minimalHybridPDF()
	conf := hybridConf(true)

	ctx, err := api.ReadContext(bytes.NewReader(in), conf)
	if err != nil {
		t.Fatalf("read hybrid PDF: %v", err)
	}
	if !ctx.Read.Hybrid {
		t.Fatal("expected hybrid PDF detection")
	}
	if err := api.ValidateContext(ctx); err != nil {
		t.Fatalf("validate hybrid PDF: %v", err)
	}

	for _, tt := range []struct {
		name            string
		writeXRefStream bool
		wantXRefStream  bool
	}{
		// Hybrid input does not imply hybrid output. pdfcpu writes the selected
		// representation according to configuration and both outputs must read back.
		{"xref stream", true, true},
		{"xref section", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wm, err := api.TextWatermark("Issue1059", "pos:c", false, false, types.POINTS)
			if err != nil {
				t.Fatalf("text watermark: %v", err)
			}

			conf := hybridConf(tt.writeXRefStream)
			var out bytes.Buffer
			if err := api.AddWatermarks(bytes.NewReader(in), &out, nil, wm, conf); err != nil {
				t.Fatalf("watermark hybrid PDF: %v", err)
			}
			validateWatermarkedHybridOutput(t, out.Bytes(), tt.wantXRefStream)
		})
	}
}

func validateWatermarkedHybridOutput(t *testing.T, out []byte, wantXRefStream bool) {
	t.Helper()

	ctx, err := api.ReadContext(bytes.NewReader(out), hybridConf(true))
	if err != nil {
		t.Fatalf("read watermarked output: %v", err)
	}
	if err := api.ValidateContext(ctx); err != nil {
		t.Fatalf("validate watermarked output: %v", err)
	}
	if ctx.Read.UsingXRefStreams != wantXRefStream {
		t.Fatalf("UsingXRefStreams = %v, want %v", ctx.Read.UsingXRefStreams, wantXRefStream)
	}
}

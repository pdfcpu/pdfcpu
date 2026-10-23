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
	"os"
	"path/filepath"
	"testing"
)

func minimalPDF15(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.5\n%\xE2\xE3\xCF\xD3\n")
	offsets := []int{0}
	for i, body := range []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Resources << >> >>",
	} {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 4\n")
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	dir := t.TempDir()
	inFile := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(inFile, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return inFile
}

func TestOptimizePreserveVersionFlag(t *testing.T) {
	inFile := minimalPDF15(t)
	outFile := filepath.Join(t.TempDir(), "out.pdf")

	if _, err := runPDFCPU(t, "--preserve-version", "optimize", inFile, outFile); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-1.5\n")) {
		end := min(24, len(got))
		t.Fatalf("header: got %q, want %%PDF-1.5", got[:end])
	}
}

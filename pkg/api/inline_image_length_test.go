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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func inlineImageLengthTestPDF(headerVersion, catalogVersion string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%%PDF-%s\n%%\xFF\xFF\xFF\xFF\n", headerVersion)
	content := "q BI /W 13 /H 1 /BPC 8 /CS /G /L 13 ID x EI /Fake Do EI Q /Real Do"
	objects := []string{
		fmt.Sprintf("<< /Type /Catalog /Pages 2 0 R %s >>", catalogVersion),
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << /XObject << /Real 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Length 3 /Type /XObject /Subtype /Form /BBox [0 0 1 1] /Resources <<>> >>\nstream\nq Q\nendstream",
	}
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = appendValidationTestObject(&buf, i+1, body)
	}
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return buf.Bytes()
}

// TestOptimizePDF20InlineImageLength covers both header and catalog PDF 2.0 versions.
func TestOptimizePDF20InlineImageLength(t *testing.T) {
	for _, tc := range []struct {
		name           string
		headerVersion  string
		catalogVersion string
	}{
		{"header", "2.0", ""},
		{"catalog", "1.7", "/Version /2.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = model.ValidationStrict
			conf.Optimize = true
			input := inlineImageLengthTestPDF(tc.headerVersion, tc.catalogVersion)
			if _, err := ReadValidateAndOptimize(t.Context(), bytes.NewReader(input), conf, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

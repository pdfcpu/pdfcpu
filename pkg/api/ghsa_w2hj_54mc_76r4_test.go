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

// GHSA-w2hj-54mc-76r4: overflowing xref stream /W widths must return an error instead of panicking.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-w2hj-54mc-76r4

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ghsaW2hjXRefStreamWidthOverflowPDF() []byte {
	header := []byte("%PDF-1.5\n%\xe2\xe3\xcf\xd3\n")
	w := 2*(math.MaxInt/3) + 1
	xref := fmt.Sprintf("1 0 obj\n<</Type/XRef/Size 3/Root 2 0 R/W[%d %d %d]/Index[0 3]/Length 3>>\nstream\n", w, w, w+2)
	b := append(header, xref...)
	b = append(b, 0, 0, 0)
	return fmt.Appendf(b, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", len(header))
}

// TestGHSA_w2hj_54mc_76r4_ValidateFileRejectsXRefStreamWidthOverflow verifies malformed widths return an error instead
// of panicking.
func TestGHSA_w2hj_54mc_76r4_ValidateFileRejectsXRefStreamWidthOverflow(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "xref-stream-width-overflow.pdf")
	if err := os.WriteFile(fileName, ghsaW2hjXRefStreamWidthOverflowPDF(), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		mode int
	}{
		{"strict", model.ValidationStrict},
		{"relaxed", model.ValidationRelaxed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := model.NewStatelessConfiguration()
			conf.ValidationMode = tc.mode
			err := ValidateFile(t.Context(), fileName, conf, nil)
			if err == nil {
				t.Fatal("expected malformed PDF error")
			}
		})
	}
}

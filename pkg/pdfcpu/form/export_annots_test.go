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

package form_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func readExportFixture(t *testing.T, objs []string) *model.Context {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offs := make([]int, len(objs))
	for i, obj := range objs {
		offs[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	ctx, err := api.ReadValidateAndOptimize(t.Context(), bytes.NewReader(b.Bytes()), model.NewDefaultConfiguration(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// TestExportFormFollowsAnnotationOrder verifies that fields follow page annotation order on repeated exports.
func TestExportFormFollowsAnnotationOrder(t *testing.T) {
	const n = 12
	objs := []string{"", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", ""}
	refs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		refs = append(refs, fmt.Sprintf("%d 0 R", 4+i))
		objs = append(objs, fmt.Sprintf("<< /Type /Annot /Subtype /Widget /FT /Tx /T (f%02d) /Rect [0 %d 50 %d] /P 3 0 R /DA (/Helv 0 Tf 0 g) >>", i, i*10, i*10+8))
	}
	objs[0] = "<< /Type /Catalog /Pages 2 0 R /AcroForm << /Fields [" + strings.Join(refs, " ") + "] /DA (/Helv 0 Tf 0 g) >> >>"
	objs[2] = "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Annots [" + strings.Join(refs, " ") + "] >>"
	ctx := readExportFixture(t, objs)
	for run := 0; run < 5; run++ {
		fg, ok, err := form.ExportForm(t.Context(), ctx.XRefTable, "x.pdf")
		if err != nil {
			t.Fatal(err)
		}
		if !ok || len(fg.Forms[0].TextFields) != n {
			t.Fatalf("run %d: exported %d text fields, want %d", run, len(fg.Forms[0].TextFields), n)
		}
		for i, tf := range fg.Forms[0].TextFields {
			if want := fmt.Sprintf("f%02d", i); tf.Name != want {
				t.Fatalf("run %d: text field %d is %s, want %s", run, i, tf.Name, want)
			}
		}
	}
}

// TestExportFormSkipsDirectNonWidgetAnnotation verifies that a tolerated direct annotation does not block form export.
func TestExportFormSkipsDirectNonWidgetAnnotation(t *testing.T) {
	ctx := readExportFixture(t, []string{
		"<< /Type /Catalog /Pages 2 0 R /AcroForm << /Fields [4 0 R] /DA (/Helv 0 Tf 0 g) >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Annots [<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] >> 4 0 R] >>",
		"<< /Type /Annot /Subtype /Widget /FT /Tx /T (name) /V (x) /Rect [0 50 50 70] /P 3 0 R /DA (/Helv 0 Tf 0 g) >>",
	})
	fg, ok, err := form.ExportForm(t.Context(), ctx.XRefTable, "x.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(fg.Forms[0].TextFields) != 1 || fg.Forms[0].TextFields[0].Name != "name" {
		t.Fatalf("unexpected exported form: %+v", fg)
	}
}

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
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

func choiceHierarchyFixture(t *testing.T, flags int, merged bool) (*model.Context, types.Dict, []types.Dict) {
	t.Helper()
	value := "(a)"
	if flags&int(primitives.FieldMultiselect) != 0 {
		value = "[(a)]"
	}
	field := fmt.Sprintf("/T (field) /Parent 4 0 R /Opt [() (a) () (b)] /V %s /DV %s /DA (/Helv 12 Tf 0 g)", value, value)
	annots, kids := "6 0 R 7 0 R", "/Kids [6 0 R 7 0 R]"
	if merged {
		annots, kids = "5 0 R", "/Type /Annot /Subtype /Widget /Rect [0 0 100 60] /P 3 0 R"
	}
	ctx := readExportFixture(t, []string{
		"<< /Type /Catalog /Pages 2 0 R /AcroForm << /Fields [4 0 R] /DA (/Helv 12 Tf 0 g) >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Annots [" + annots + "] >>",
		fmt.Sprintf("<< /T (group) /FT /Ch /Ff %d /Kids [5 0 R] >>", flags),
		"<< " + field + " " + kids + " >>",
		"<< /Type /Annot /Subtype /Widget /Parent 5 0 R /Rect [0 0 100 60] /P 3 0 R >>",
		"<< /Type /Annot /Subtype /Widget /Parent 5 0 R /Rect [0 80 100 140] /P 3 0 R >>",
	})
	d, err := ctx.DereferenceDict(*types.NewIndirectRef(5, 0))
	if err != nil {
		t.Fatal(err)
	}
	widgets := []types.Dict{d}
	if !merged {
		widgets = nil
		for _, nr := range []int{6, 7} {
			w, err := ctx.DereferenceDict(*types.NewIndirectRef(nr, 0))
			if err != nil {
				t.Fatal(err)
			}
			widgets = append(widgets, w)
		}
	}
	return ctx, d, widgets
}

func assertChoiceIndex(t *testing.T, d types.Dict, want int) {
	t.Helper()
	got := d.ArrayEntry("I")
	if len(got) != 1 || got[0] != types.Integer(want) {
		t.Fatalf("I=%v, want [%d]", got, want)
	}
}

func checkChoiceListingAndExport(t *testing.T, ctx *model.Context, combo bool) {
	t.Helper()
	fs, _, err := form.FormFields(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := form.FTListBox
	if combo {
		want = form.FTComboBox
	}
	if len(fs) != 1 || fs[0].Name != "group.field" || fs[0].Typ != want {
		t.Fatalf("listed fields: %+v", fs)
	}
	fg, ok, err := form.ExportForm(t.Context(), ctx.XRefTable, "choice.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("nothing exported")
	}
	if combo {
		if len(fg.Forms[0].ComboBoxes) != 1 {
			t.Fatalf("exported combo boxes: %+v", fg.Forms[0])
		}
	} else if len(fg.Forms[0].ListBoxes) != 1 {
		t.Fatalf("exported list boxes: %+v", fg.Forms[0])
	}
}

func checkFieldLocking(t *testing.T, ctx *model.Context, d types.Dict, widgets []types.Dict, flags int) {
	t.Helper()
	if _, err := form.LockFormFields(t.Context(), ctx, nil); err != nil {
		t.Fatal(err)
	}
	ff := d.IntEntry("Ff")
	if ff == nil || *ff != flags|int(primitives.FieldReadOnly) {
		t.Fatalf("locked flags: %v", ff)
	}
	for _, w := range widgets {
		if w.DictEntry("AP") == nil {
			t.Fatal("missing widget appearance")
		}
	}
	if _, err := form.UnlockFormFields(t.Context(), ctx, nil); err != nil {
		t.Fatal(err)
	}
	ff = d.IntEntry("Ff")
	if ff == nil || *ff != flags {
		t.Fatalf("unlocked flags: %v", ff)
	}
	if flags&int(primitives.FieldCombo) != 0 {
		for _, w := range widgets {
			if w.DictEntry("AP") != nil {
				t.Fatal("unlocked combo retained appearance")
			}
		}
	}
}

// TestChoiceHierarchyOperations verifies inherited choice flags and option indices across form commands.
func TestChoiceHierarchyOperations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags int
	}{
		{"single list", 0}, {"combo", int(primitives.FieldCombo)}, {"multi list", int(primitives.FieldMultiselect)},
	} {
		for _, merged := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/merged=%t", tc.name, merged), func(t *testing.T) {
				ctx, d, widgets := choiceHierarchyFixture(t, tc.flags, merged)
				combo := tc.flags&int(primitives.FieldCombo) != 0
				checkChoiceListingAndExport(t, ctx, combo)
				data := &form.Form{ListBoxes: []*form.ListBox{{Name: "group.field", Values: []string{"b"}}}}
				if combo {
					data = &form.Form{ComboBoxes: []*form.ComboBox{{Name: "group.field", Value: "b"}}}
				}
				if _, _, err := form.FillForm(t.Context(), ctx, form.FillDetails(data, nil), nil, form.JSON); err != nil {
					t.Fatal(err)
				}
				assertChoiceIndex(t, d, 3)
				if _, err := form.ResetFormFields(t.Context(), ctx, nil); err != nil {
					t.Fatal(err)
				}
				assertChoiceIndex(t, d, 1)
				checkFieldLocking(t, ctx, d, widgets, tc.flags)
			})
		}
	}
}

// TestTextHierarchyOperations verifies ancestor-inherited type and multiline flags across form commands.
func TestTextHierarchyOperations(t *testing.T) {
	ctx := readExportFixture(t, []string{
		"<< /Type /Catalog /Pages 2 0 R /AcroForm << /Fields [4 0 R] /DA (/Helv 12 Tf 0 g) >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Annots [6 0 R] >>",
		fmt.Sprintf("<< /T (group) /FT /Tx /Ff %d /Kids [5 0 R] >>", primitives.FieldMultiline),
		"<< /T (field) /Parent 4 0 R /Kids [6 0 R] /V (old) /DV (first\\nsecond) /DA (/Helv 12 Tf 0 g) >>",
		"<< /Type /Annot /Subtype /Widget /Parent 5 0 R /Rect [0 0 100 60] /P 3 0 R >>",
	})
	fs, _, err := form.FormFields(t.Context(), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Name != "group.field" || fs[0].Typ != form.FTText {
		t.Fatalf("listed fields: %+v", fs)
	}
	fg, ok, err := form.ExportForm(t.Context(), ctx.XRefTable, "text.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || len(fg.Forms[0].TextFields) != 1 || !fg.Forms[0].TextFields[0].Multiline {
		t.Fatalf("exported fields: %+v", fg)
	}
	data := &form.Form{TextFields: []*form.TextField{{Name: "group.field", Value: "new"}}}
	if _, _, err := form.FillForm(t.Context(), ctx, form.FillDetails(data, nil), nil, form.JSON); err != nil {
		t.Fatal(err)
	}
	d, err := ctx.DereferenceDict(*types.NewIndirectRef(5, 0))
	if err != nil {
		t.Fatal(err)
	}
	checkFieldValue(t, ctx, d, "new")
	if _, err := form.ResetFormFields(t.Context(), ctx, nil); err != nil {
		t.Fatal(err)
	}
	checkFieldValue(t, ctx, d, "first\nsecond")
	w, err := ctx.DereferenceDict(*types.NewIndirectRef(6, 0))
	if err != nil {
		t.Fatal(err)
	}
	checkMultilineAppearance(t, ctx, w)
	checkFieldLocking(t, ctx, d, []types.Dict{w}, int(primitives.FieldMultiline))
}

func checkMultilineAppearance(t *testing.T, ctx *model.Context, widget types.Dict) {
	t.Helper()
	sd, _, err := ctx.DereferenceStreamDict(widget.DictEntry("AP")["N"])
	if err != nil || sd == nil {
		t.Fatalf("normal appearance: %v", err)
	}
	if got := strings.Count(string(sd.Content), " Tj "); got != 2 {
		t.Fatalf("appearance has %d text lines, want 2: %s", got, sd.Content)
	}
}

func checkFieldValue(t *testing.T, ctx *model.Context, d types.Dict, want string) {
	t.Helper()
	v, _, err := ctx.DereferenceStringEntry(d, "V")
	if err != nil {
		t.Fatal(err)
	}
	if v == nil || *v != want {
		t.Fatalf("field value: %v, want %q", v, want)
	}
}

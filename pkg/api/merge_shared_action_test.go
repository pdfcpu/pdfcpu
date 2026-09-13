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
	"io"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func sharedActionMergePDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R /OpenAction 4 0 R /Names << /Dests 5 0 R >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << >> >>",
		"<< /S /Named /N /FirstPage /Next [6 0 R 6 0 R] >>",
		"<< /Names [(target) [3 0 R /Fit]] >>",
		"<< /S /GoTo /D (target) >>",
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, body := range objects {
		offsets[i] = appendValidationTestObject(&b, i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Root 1 0 R /Size %d >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

// TestMergeSharedActionDestinations verifies conflicting names update each shared action only once.
func TestMergeSharedActionDestinations(t *testing.T) {
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		conf := model.NewStatelessConfiguration()
		conf.ValidationMode = mode
		conf.Offline = true
		data := sharedActionMergePDF()
		sources := []io.ReadSeeker{bytes.NewReader(data), bytes.NewReader(data)}
		var out bytes.Buffer
		if err := MergeRaw(t.Context(), sources, &out, false, conf); err != nil {
			t.Fatalf("mode %d: merge: %v", mode, err)
		}
		checkSharedActionMerge(t, out.Bytes())
	}
}

func checkSharedActionMerge(t *testing.T, data []byte) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationStrict
	ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(data), conf)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.PageCount != 2 {
		t.Fatalf("page count: %d", ctx.PageCount)
	}
	tree := ctx.Names["Dests"]
	if tree == nil {
		t.Fatal("missing destinations")
	}
	pages := map[int]bool{}
	keys := map[string]bool{}
	err = tree.Process(t.Context(), ctx.XRefTable, func(x *model.XRefTable, key string, value *types.Object) error {
		a, err := x.DereferenceArray(*value)
		if err != nil {
			return err
		}
		if len(a) == 0 {
			return fmt.Errorf("empty destination %q", key)
		}
		ir, ok := a[0].(types.IndirectRef)
		if !ok {
			return fmt.Errorf("destination %q has no page reference", key)
		}
		keys[key] = true
		pages[ir.ObjectNumber.Value()] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || len(pages) != 2 || !keys["target"] {
		t.Fatalf("destinations: keys=%v pages=%v", keys, pages)
	}
}

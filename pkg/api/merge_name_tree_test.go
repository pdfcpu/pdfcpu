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

func mergeNameTreePDF(prefix string, count int) []byte {
	var names bytes.Buffer
	names.WriteString("<< /Names [")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&names, "(%s%04d) [3 0 R /Fit] ", prefix, i)
	}
	names.WriteString("] >>")
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R /Names << /Dests 4 0 R >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << >> >>",
		names.String(),
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

func checkMergedNameTree(t *testing.T, ctx *model.Context, count int) {
	t.Helper()
	tree := ctx.Names["Dests"]
	if tree == nil {
		t.Fatal("missing destination tree")
	}
	keys, err := tree.KeyList(t.Context())
	if err != nil || len(keys) != count+1 {
		t.Fatalf("keys=%d error=%v", len(keys), err)
	}
	pages := map[string]int{}
	counts := map[string]int{"a": 1, "b": count}
	for _, prefix := range []string{"a", "b"} {
		for i := 0; i < counts[prefix]; i++ {
			key := fmt.Sprintf("%s%04d", prefix, i)
			value, ok, err := tree.Value(t.Context(), key)
			if err != nil || !ok {
				t.Fatalf("destination %s: %v", key, err)
			}
			a, err := ctx.DereferenceArray(value)
			if err != nil || len(a) != 2 {
				t.Fatalf("destination %s: %v %v", key, a, err)
			}
			ir, ok := a[0].(types.IndirectRef)
			if !ok {
				t.Fatalf("destination %s has no page reference", key)
			}
			objNr := ir.ObjectNumber.Value()
			if previous, ok := pages[prefix]; ok && previous != objNr {
				t.Fatalf("destination %s changed page", key)
			}
			pages[prefix] = objNr
		}
	}
	if ctx.PageCount != 2 || pages["a"] == pages["b"] {
		t.Fatalf("page mapping: %v, count=%d", pages, ctx.PageCount)
	}
}

// TestMergeSortedNameTreesPreservesDestinations verifies merged trees stay within the configured depth limit.
func TestMergeSortedNameTreesPreservesDestinations(t *testing.T) {
	const count = 204
	sources := []io.ReadSeeker{bytes.NewReader(mergeNameTreePDF("a", 1)), bytes.NewReader(mergeNameTreePDF("b", count))}
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationStrict
	conf.Offline = true
	var out bytes.Buffer
	if err := MergeRaw(t.Context(), sources, &out, false, conf); err != nil {
		t.Fatal(err)
	}
	ctx, err := ReadAndValidate(t.Context(), bytes.NewReader(out.Bytes()), conf)
	if err != nil {
		t.Fatal(err)
	}
	checkMergedNameTree(t, ctx, count)
}

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

// GHSA-2623-w998-vcr4: annotation deletion, page migration and the repair of references to free objects during
// optimization must not recurse once per object of a reference chain, because a long chain overflows the goroutine
// stack and kills the process.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-2623-w998-vcr4

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const ghsa2623Links = 20000

func ghsa2623PDF(objs map[int]string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	n := slices.Max(slices.Collect(maps.Keys(objs)))
	offsets := make([]int, n+1)
	for i := 1; i <= n; i++ {
		if body, ok := objs[i]; ok {
			offsets[i] = b.Len()
			fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i, body)
		}
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", n+1)
	for i := 1; i <= n; i++ {
		if offsets[i] == 0 {
			b.WriteString("0000000000 65535 f \n")
			continue
		}
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", n+1, xref)
	return b.Bytes()
}

func ghsa2623Objects(page string) map[int]string {
	return map[int]string{
		1: "<< /Type /Catalog /Pages 2 0 R >>",
		2: "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		3: "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << >> " + page + " >>",
		4: "<< /Length 0 >>\nstream\n\nendstream",
	}
}

// ghsa2623AnnotationChain hangs a chain of links dictionaries below a nonstandard annotation entry.
func ghsa2623AnnotationChain(links int) map[int]string {
	objs := ghsa2623Objects("/Annots [5 0 R]")
	objs[5] = "<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /Zz 6 0 R >>"
	for i := range links - 1 {
		objs[6+i] = fmt.Sprintf("<< /Zz %d 0 R >>", 7+i)
	}
	objs[5+links] = "<< >>"
	return objs
}

// ghsa2623PageChain hangs a chain of links arrays below a nonstandard page entry.
func ghsa2623PageChain(links int) map[int]string {
	objs := ghsa2623Objects("/X 5 0 R")
	for i := range links - 1 {
		objs[5+i] = fmt.Sprintf("[%d 0 R]", 6+i)
	}
	objs[4+links] = "null"
	return objs
}

// ghsa2623AnnotationRefChain is ghsa2623AnnotationChain with objects whose value is the next reference.
func ghsa2623AnnotationRefChain(links int) map[int]string {
	objs := ghsa2623AnnotationChain(links)
	for i := range links - 1 {
		objs[6+i] = fmt.Sprintf("%d 0 R", 7+i)
	}
	return objs
}

// ghsa2623PageRefChain is ghsa2623PageChain with objects whose value is the next reference.
func ghsa2623PageRefChain(links int) map[int]string {
	objs := ghsa2623PageChain(links)
	for i := range links - 1 {
		objs[5+i] = fmt.Sprintf("%d 0 R", 6+i)
	}
	return objs
}

func ghsa2623Read(t *testing.T, objs map[int]string) *model.Context {
	t.Helper()
	ctx, err := Read(t.Context(), bytes.NewReader(ghsa2623PDF(objs)), model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.EnsurePageCount(); err != nil {
		t.Fatal(err)
	}
	return ctx
}

// ghsa2623RemoveAnnotationChain removes the annotation; wantFreed checks that the last chain object was freed.
// XRefTable.DeleteObject does not follow objects whose value is a reference, so such chains are only walked.
func ghsa2623RemoveAnnotationChain(t *testing.T, objs map[int]string, wantFreed bool) {
	ctx := ghsa2623Read(t, objs)
	d, _, _, err := ctx.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := RemoveAnnotationsFromPageDict(t.Context(), ctx, nil, nil, nil, d, 3, 1, false)
	if err != nil || !ok {
		t.Fatalf("remove annotations: ok=%t err=%v", ok, err)
	}
	if entry, found := ctx.FindTableEntryLight(5 + ghsa2623Links); wantFreed && (!found || !entry.Free) {
		t.Fatal("last chain object was not freed")
	}
}

func ghsa2623ExtractPageChain(t *testing.T, objs map[int]string) {
	ctx := ghsa2623Read(t, objs)
	ctxDest, err := ExtractPages(t.Context(), ctx, []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(ctxDest.Table); got < ghsa2623Links {
		t.Fatalf("migrated objects: got %d, want at least %d", got, ghsa2623Links)
	}
}

func ghsa2623FixFreeReferencesInPageChain(t *testing.T) {
	ctx := ghsa2623Read(t, ghsa2623PageChain(ghsa2623Links))
	if err := fixReferencesToFreeObjects(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	if !ctx.Optimize.Cache[4+ghsa2623Links] {
		t.Fatal("last chain object was not visited")
	}
}

// TestGHSA_2623_w998_vcr4_LongReferenceChains isolates long chains in a subprocess with a small stack, so a
// recursive traversal fails the test instead of crashing the test binary.
func TestGHSA_2623_w998_vcr4_LongReferenceChains(t *testing.T) {
	const childEnv = "PDFCPU_GHSA_2623_CHAIN"
	switch os.Getenv(childEnv) {
	case "":
	case "annotations":
		debug.SetMaxStack(1 << 20)
		ghsa2623RemoveAnnotationChain(t, ghsa2623AnnotationChain(ghsa2623Links), true)
		return
	case "annotation-refs":
		debug.SetMaxStack(1 << 20)
		ghsa2623RemoveAnnotationChain(t, ghsa2623AnnotationRefChain(ghsa2623Links), false)
		return
	case "pages":
		debug.SetMaxStack(1 << 20)
		ghsa2623ExtractPageChain(t, ghsa2623PageChain(ghsa2623Links))
		return
	case "page-refs":
		debug.SetMaxStack(1 << 20)
		ghsa2623ExtractPageChain(t, ghsa2623PageRefChain(ghsa2623Links))
		return
	case "optimize":
		debug.SetMaxStack(1 << 20)
		ghsa2623FixFreeReferencesInPageChain(t)
		return
	default:
		t.Fatalf("unknown chain %q", os.Getenv(childEnv))
	}

	for _, chain := range []string{"annotations", "annotation-refs", "pages", "page-refs", "optimize"} {
		t.Run(chain, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestGHSA_2623_w998_vcr4_LongReferenceChains$")
			cmd.Env = append(os.Environ(), childEnv+"="+chain)
			out, err := cmd.CombinedOutput()
			if err != nil {
				if len(out) > 1000 {
					out = out[:1000]
				}
				t.Fatalf("%s chain in subprocess: %v\n%s", chain, err, out)
			}
		})
	}
}

// TestGHSA_2623_w998_vcr4_ArticleThreadMigrates verifies a valid article thread longer than the recursion limit
// is still copied completely; a depth limit would reject it.
func TestGHSA_2623_w998_vcr4_ArticleThreadMigrates(t *testing.T) {
	const beads = 500
	refs := make([]string, beads)
	for i := range refs {
		refs[i] = fmt.Sprintf("%d 0 R", 6+i)
	}
	objs := ghsa2623Objects("/B [" + strings.Join(refs, " ") + "]")
	objs[1] = "<< /Type /Catalog /Pages 2 0 R /Threads [5 0 R] >>"
	objs[5] = "<< /Type /Thread /F 6 0 R >>"
	for i := range beads {
		next, prev := 6+(i+1)%beads, 6+(i+beads-1)%beads
		objs[6+i] = fmt.Sprintf("<< /Type /Bead /T 5 0 R /N %d 0 R /V %d 0 R /P 3 0 R /R [0 0 10 10] >>", next, prev)
	}
	ctx := ghsa2623Read(t, objs)
	ctxDest, err := ExtractPages(t.Context(), ctx, []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(ctxDest.Table); got < beads {
		t.Fatalf("migrated objects: got %d, want at least %d", got, beads)
	}
}

// TestGHSA_2623_w998_vcr4_AnnotationDeletionErrorPath verifies errors still name the entries leading to them.
func TestGHSA_2623_w998_vcr4_AnnotationDeletionErrorPath(t *testing.T) {
	objs := ghsa2623Objects("/Annots [5 0 R]")
	objs[5] = "<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /A 7 0 R /Zz 6 0 R >>"
	objs[6] = "<< /K [7 0 R 99 0 R] >>"
	objs[7] = "<< >>"
	ctx := ghsa2623Read(t, objs)
	delete(ctx.Table, 99)
	err := validateAnnotationDeletionTargets(ctx, []types.Object{*types.NewIndirectRef(5, 0)}, 1)
	want := "page 1 annotation deletion target 1: entry Zz: entry K: array[1]: " +
		"page 1 annotation deletion obj#99: missing xref table entry"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %s", err, want)
	}
}

// TestGHSA_2623_w998_vcr4_FreeReferenceAtChainEnd verifies a reference to a free object at the end of a chain is
// still replaced with a reference to a null object.
func TestGHSA_2623_w998_vcr4_FreeReferenceAtChainEnd(t *testing.T) {
	objs := ghsa2623Objects("/X 5 0 R")
	objs[5] = "[6 0 R]"
	objs[6] = "[7 0 R]"
	objs[7] = "[9 0 R]" // obj 9 is a free xref entry
	objs[10] = "null"
	ctx := ghsa2623Read(t, objs)
	if entry, found := ctx.FindTableEntryLight(9); !found || !entry.Free {
		t.Fatal("test input: obj 9 must be a free xref entry")
	}
	if err := fixReferencesToFreeObjects(t.Context(), ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Optimize.NullObjNr == nil {
		t.Fatal("no null object inserted")
	}
	a, err := ctx.DereferenceArray(*types.NewIndirectRef(7, 0))
	if err != nil {
		t.Fatal(err)
	}
	ir, ok := a[0].(types.IndirectRef)
	if !ok || ir.ObjectNumber.Value() != *ctx.Optimize.NullObjNr {
		t.Fatalf("got %v, want reference to null object %d", a[0], *ctx.Optimize.NullObjNr)
	}
}

// TestGHSA_2623_w998_vcr4_MigratesReferenceValues verifies an object whose value is a reference is copied with its
// target, so the destination holds no reference into the source document.
func TestGHSA_2623_w998_vcr4_MigratesReferenceValues(t *testing.T) {
	objs := ghsa2623Objects("/X 5 0 R")
	objs[5] = "6 0 R"
	objs[6] = "7 0 R"
	objs[7] = "[1 2]"
	ctx := ghsa2623Read(t, objs)
	ctxDest, err := ExtractPages(t.Context(), ctx, []int{1}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ctxDest.EnsurePageCount(); err != nil {
		t.Fatal(err)
	}
	d, _, _, err := ctxDest.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	o := d["X"]
	for range 3 {
		ir, ok := o.(types.IndirectRef)
		if !ok {
			break
		}
		entry, found := ctxDest.FindTableEntryLight(ir.ObjectNumber.Value())
		if !found || entry.Free {
			t.Fatalf("destination has no obj#%d", ir.ObjectNumber.Value())
		}
		o = entry.Object
	}
	if a, ok := o.(types.Array); !ok || len(a) != 2 {
		t.Fatalf("got %v, want the migrated array [1 2]", o)
	}
}

// TestGHSA_2623_w998_vcr4_LongErrorPathIsBounded verifies the error for a failure at the end of a long chain names a
// bounded path instead of one entry per link.
func TestGHSA_2623_w998_vcr4_LongErrorPathIsBounded(t *testing.T) {
	const links = 20000
	objs := ghsa2623AnnotationChain(links)
	objs[5+links] = fmt.Sprintf("<< /K %d 0 R >>", 100+links) // beyond the xref table
	ctx := ghsa2623Read(t, objs)
	err := validateAnnotationDeletionTargets(ctx, []types.Object{*types.NewIndirectRef(5, 0)}, 1)
	if err == nil {
		t.Fatal("missing error")
	}
	msg := err.Error()
	if len(msg) > 4096 {
		t.Fatalf("error message has %d bytes", len(msg))
	}
	for _, want := range []string{"page 1 annotation deletion target 1: entry Zz: ", "more entries", "entry K: ",
		fmt.Sprintf("page 1 annotation deletion obj#%d: missing xref table entry", 100+links)} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q lacks %q", msg, want)
		}
	}
}

// TestGHSA_2623_w998_vcr4_ErrorPathElidesLongPaths verifies short error paths are complete and long ones keep only their
// first and last entries.
func TestGHSA_2623_w998_vcr4_ErrorPathElidesLongPaths(t *testing.T) {
	label := func(j int) string { return fmt.Sprintf("e%d", j) }
	join := func(from, to int) string {
		labels := make([]string, 0, to-from)
		for j := from; j < to; j++ {
			labels = append(labels, label(j))
		}
		return strings.Join(labels, ": ")
	}
	for _, tc := range []struct {
		n    int
		want string
	}{
		{1, "e0"},
		{32, join(0, 32)},
		{33, join(0, 16) + ": ... 1 more entries ...: " + join(17, 33)},
		{100, join(0, 16) + ": ... 68 more entries ...: " + join(84, 100)},
	} {
		if got := errorPath(tc.n, label); got != tc.want {
			t.Errorf("n=%d: got %q, want %q", tc.n, got, tc.want)
		}
	}
}

// TestGHSA_2623_w998_vcr4_FreeReferenceErrorPath verifies errors of the free-reference repair name dictionary keys,
// array indices and the indirect objects they belong to, outermost first, and keep the cause.
func TestGHSA_2623_w998_vcr4_FreeReferenceErrorPath(t *testing.T) {
	f := freeRefFix{}
	f.push(types.Dict{"A": types.Integer(0)}, 0)
	f.stack.next()
	f.push(types.Array{types.Integer(0), types.Integer(1)}, 7)
	f.stack.next()
	f.stack.next()
	f.push(types.Dict{"B": types.Integer(0)}, 9)
	i, _, _ := f.stack.next()

	cause := errors.New("cause")
	err := f.wrap(i, cause)
	if want := "A: obj#7: [1]: obj#9: B: cause"; err.Error() != want {
		t.Fatalf("got %q, want %q", err, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("cause not wrapped")
	}
}

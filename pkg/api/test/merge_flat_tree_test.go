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

// Regression test for the degenerate-page-tree merge bug.
//
// Before the page-tree-fix patch, appendSourcePageTreeToDestPageTree
// wrapped both the existing dest tree AND the source tree in a NEW
// /Pages node on every merge call, producing a left-skewed binary
// tree of depth N+1 for N inputs. Recursive PDF parsers (qpdf,
// Adobe Acrobat) overflow their stack on trees deeper than a few
// thousand levels. A real-world witness-pack merge of 2,404 single-
// page inputs produced a 2,405-deep tree that crashed qpdf with
// stack overflow and would not open in Acrobat.
//
// The fix splices source root /Pages into dest root's /Kids array
// without wrapping. This test merges N inputs and asserts the
// resulting page tree depth stays small.

package test

import (
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// pageTreeDepth recursively measures the maximum depth of a /Pages
// subtree rooted at indRef. Bails at sanityLimit to avoid hitting
// the test runner's stack overflow — if we hit that, the test fails
// with a clear "degenerate tree" message rather than a confusing
// runtime panic.
func pageTreeDepth(t *testing.T, ctx *model.Context, indRef types.IndirectRef, currentDepth int) int {
	t.Helper()

	const sanityLimit = 5000
	if currentDepth > sanityLimit {
		t.Fatalf("page tree walk exceeded sanity limit %d — tree is degenerate", sanityLimit)
	}

	d, err := ctx.XRefTable.DereferenceDict(indRef)
	if err != nil {
		t.Fatalf("DereferenceDict(%v): %v", indRef, err)
	}

	// /Page leaf: depth is currentDepth + 1 (counting this node).
	if typ := d.Type(); typ != nil && *typ == "Page" {
		return currentDepth + 1
	}

	kids, ok := d["Kids"].(types.Array)
	if !ok || len(kids) == 0 {
		return currentDepth + 1
	}

	// /Pages node — recurse into kids, return the max depth.
	maxDepth := currentDepth + 1
	for _, kid := range kids {
		kidRef, ok := kid.(types.IndirectRef)
		if !ok {
			continue
		}
		kidDepth := pageTreeDepth(t, ctx, kidRef, currentDepth+1)
		if kidDepth > maxDepth {
			maxDepth = kidDepth
		}
	}
	return maxDepth
}

func TestMergeProducesFlatPageTree(t *testing.T) {
	const (
		// 100 inputs is enough to reliably catch the bug: pre-fix this
		// would produce a 101-deep tree. We don't need to go to the
		// thousands here — the bug is linear in N, so 100 is plenty
		// of signal while keeping the test fast.
		numInputs = 100

		// Max acceptable depth: 1 (dest root /Pages) + source's own
		// depth (typically 2: /Pages → /Page for stock test PDFs) =
		// 3 in the typical case. Buffer to 10 to absorb test PDFs
		// with slightly nested page trees of their own without flag-
		// ging false positives. Anything close to numInputs means
		// the wrap-on-every-merge bug regressed.
		maxAcceptableDepth = 10
	)

	msg := "TestMergeProducesFlatPageTree"

	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	inFiles := make([]string, numInputs)
	for i := range inFiles {
		inFiles[i] = inFile
	}

	outFile := filepath.Join(outDir, "merge_flat_tree.pdf")
	if err := api.MergeCreateFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: MergeCreateFile: %v", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: ValidateFile: %v", msg, err)
	}

	ctx, err := api.ReadContextFile(outFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile: %v", msg, err)
	}

	indRefRoot, err := ctx.Pages()
	if err != nil {
		t.Fatalf("%s: ctx.Pages: %v", msg, err)
	}

	depth := pageTreeDepth(t, ctx, *indRefRoot, 0)

	if depth > maxAcceptableDepth {
		t.Fatalf(
			"%s: page tree depth %d exceeds maximum %d after merging %d inputs — "+
				"degenerate-tree regression (see appendSourcePageTreeToDestPageTree)",
			msg, depth, maxAcceptableDepth, numInputs,
		)
	}
	t.Logf("%s: merged %d inputs → page tree depth %d (well within max %d)",
		msg, numInputs, depth, maxAcceptableDepth)
}

func TestMergeProducesFlatPageTree_WithDivider(t *testing.T) {
	// Same as above but with dividerPage = true. The original code
	// wrapped dest + divider + source in the new wrapper; the fix
	// appends the divider directly to dest root's /Kids. Result
	// should still be flat.

	const (
		numInputs          = 20
		maxAcceptableDepth = 10
	)

	msg := "TestMergeProducesFlatPageTree_WithDivider"

	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	inFiles := make([]string, numInputs)
	for i := range inFiles {
		inFiles[i] = inFile
	}

	outFile := filepath.Join(outDir, "merge_flat_tree_with_divider.pdf")
	if err := api.MergeCreateFile(inFiles, outFile, true, nil); err != nil {
		t.Fatalf("%s: MergeCreateFile: %v", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: ValidateFile: %v", msg, err)
	}

	ctx, err := api.ReadContextFile(outFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile: %v", msg, err)
	}

	indRefRoot, err := ctx.Pages()
	if err != nil {
		t.Fatalf("%s: ctx.Pages: %v", msg, err)
	}

	depth := pageTreeDepth(t, ctx, *indRefRoot, 0)

	if depth > maxAcceptableDepth {
		t.Fatalf(
			"%s: page tree depth %d exceeds maximum %d (divider-page path regressed)",
			msg, depth, maxAcceptableDepth,
		)
	}
	t.Logf("%s: merged %d inputs with divider → depth %d (max %d)",
		msg, numInputs, depth, maxAcceptableDepth)
}

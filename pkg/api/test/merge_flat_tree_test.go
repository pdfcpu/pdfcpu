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

// Tests for the merge page-tree shape and inheritance contract.
//
// MergeCreate/Append calls appendSourcePageTreeToDestPageTree per
// input. These tests assert three invariants of that function:
//
//   1. Depth at dest is bounded — does not grow with the number of
//      inputs. Recursive consumers (qpdf, Adobe Acrobat) walk the
//      page tree with a bounded stack and fail on deep trees.
//   2. Inherited page attributes (/Resources, /MediaBox, /CropBox,
//      /Rotate) declared on a source's root /Pages still resolve
//      correctly at each merged leaf page.
//   3. Every leaf's /Parent chain terminates at the catalog's /Pages
//      root with no broken links.

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
		// 100 inputs gives a clear signal — a depth-bound regression
		// would be linear in N, so 100 is enough to detect without
		// scaling up further.
		numInputs = 100

		// Expected: 1 (dest root /Pages) + source's own depth
		// (typically 2: /Pages → /Page) = 3 in the typical case.
		// Buffered to 10 to absorb test PDFs with slightly nested
		// page trees without false positives. Anything approaching
		// numInputs indicates a depth-bound regression.
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
	// Same as above with dividerPage = true — divider is appended
	// directly to dest root's /Kids. Result should still be flat.

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

// rectsEqual handles nil-safe comparison of two *types.Rectangle.
func rectsEqual(a, b *types.Rectangle) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Equals(*b)
	}
}

// TestMergePreservesInheritedPageAttrs asserts that after merge each
// output page resolves the same inherited attributes (/MediaBox,
// /CropBox, /Rotate, /Resources) as its source page did pre-merge.
//
// Fixture: BuildingWebappsWithGo.pdf declares /MediaBox at the source
// root /Pages and omits it from leaf /Page dicts, so the inheritance
// walk is genuinely exercised. A regression that flattened source_root
// (dropping its inheritable attrs) would cause leaf /Page dicts to
// lose /MediaBox post-merge and this test would fail.
//
// Also asserts ctxDest.PageCount equals N * srcPages.
func TestMergePreservesInheritedPageAttrs(t *testing.T) {
	const numInputs = 3

	msg := "TestMergePreservesInheritedPageAttrs"
	inFile := filepath.Join(inDir, "BuildingWebappsWithGo.pdf")

	srcCtx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile(src): %v", msg, err)
	}

	expected := make([]*model.InheritedPageAttrs, srcCtx.PageCount+1)
	for pageNr := 1; pageNr <= srcCtx.PageCount; pageNr++ {
		_, _, attrs, err := srcCtx.XRefTable.PageDict(pageNr, false)
		if err != nil {
			t.Fatalf("%s: src.PageDict(%d): %v", msg, pageNr, err)
		}
		expected[pageNr] = attrs
	}

	// Sanity: the fixture is supposed to inherit /MediaBox from its root
	// /Pages. If that ever changes the test becomes vacuous — fail loudly
	// rather than silently passing.
	if expected[1].MediaBox == nil {
		t.Fatalf("%s: fixture %s no longer inherits /MediaBox — pick another", msg, inFile)
	}

	inFiles := make([]string, numInputs)
	for i := range inFiles {
		inFiles[i] = inFile
	}
	outFile := filepath.Join(outDir, "merge_inherited_attrs.pdf")
	if err := api.MergeCreateFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: MergeCreateFile: %v", msg, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: ValidateFile: %v", msg, err)
	}

	mergedCtx, err := api.ReadContextFile(outFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile(merged): %v", msg, err)
	}

	wantPages := numInputs * srcCtx.PageCount
	if mergedCtx.PageCount != wantPages {
		t.Fatalf("%s: merged PageCount = %d, want %d", msg, mergedCtx.PageCount, wantPages)
	}

	for pageNr := 1; pageNr <= mergedCtx.PageCount; pageNr++ {
		_, _, got, err := mergedCtx.XRefTable.PageDict(pageNr, false)
		if err != nil {
			t.Fatalf("%s: merged.PageDict(%d): %v", msg, pageNr, err)
		}
		srcPageIdx := ((pageNr - 1) % srcCtx.PageCount) + 1
		want := expected[srcPageIdx]

		if !rectsEqual(want.MediaBox, got.MediaBox) {
			t.Errorf("%s: page %d (src %d): MediaBox = %v, want %v",
				msg, pageNr, srcPageIdx, got.MediaBox, want.MediaBox)
		}
		if !rectsEqual(want.CropBox, got.CropBox) {
			t.Errorf("%s: page %d (src %d): CropBox = %v, want %v",
				msg, pageNr, srcPageIdx, got.CropBox, want.CropBox)
		}
		if got.Rotate != want.Rotate {
			t.Errorf("%s: page %d (src %d): Rotate = %d, want %d",
				msg, pageNr, srcPageIdx, got.Rotate, want.Rotate)
		}

		// Resources: shallow top-level key check. Deep equality would be
		// brittle (pdfcpu may re-emit refs differently); presence of every
		// source key in the merged resolution is what proves the inherited
		// Resources dict is still reachable.
		switch {
		case want.Resources == nil && got.Resources != nil:
			t.Errorf("%s: page %d (src %d): Resources appeared post-merge", msg, pageNr, srcPageIdx)
		case want.Resources != nil && got.Resources == nil:
			t.Errorf("%s: page %d (src %d): Resources lost post-merge", msg, pageNr, srcPageIdx)
		case want.Resources != nil && got.Resources != nil:
			for k := range want.Resources {
				if _, ok := got.Resources[k]; !ok {
					t.Errorf("%s: page %d (src %d): Resources missing key %q",
						msg, pageNr, srcPageIdx, k)
				}
			}
		}
	}

	t.Logf("%s: merged %d × %d pages, inherited attrs preserved on all %d pages",
		msg, numInputs, srcCtx.PageCount, wantPages)
}

// TestMergePageTreeParentLinks verifies the /Parent chain on the merged
// output. Every leaf /Page must walk upward via /Parent and terminate
// at the catalog's /Pages root — no broken links, no chain that loops
// or detours through an orphan node.
//
// The depth test catches degenerate shape; this catches a broken-
// pointer regression that would leave pdfcpu unable to walk back up
// the tree even though the file validates.
func TestMergePageTreeParentLinks(t *testing.T) {
	const numInputs = 5

	msg := "TestMergePageTreeParentLinks"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")

	inFiles := make([]string, numInputs)
	for i := range inFiles {
		inFiles[i] = inFile
	}
	outFile := filepath.Join(outDir, "merge_parent_links.pdf")
	if err := api.MergeCreateFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: MergeCreateFile: %v", msg, err)
	}

	ctx, err := api.ReadContextFile(outFile)
	if err != nil {
		t.Fatalf("%s: ReadContextFile: %v", msg, err)
	}

	rootRef, err := ctx.Pages()
	if err != nil {
		t.Fatalf("%s: ctx.Pages: %v", msg, err)
	}

	// Catalog /Pages root itself must have no /Parent.
	rootDict, err := ctx.XRefTable.DereferenceDict(*rootRef)
	if err != nil {
		t.Fatalf("%s: DereferenceDict(root): %v", msg, err)
	}
	if _, hasParent := rootDict["Parent"]; hasParent {
		t.Errorf("%s: catalog /Pages root obj#%d has unexpected /Parent",
			msg, rootRef.ObjectNumber)
	}

	const walkLimit = 50
	for pageNr := 1; pageNr <= ctx.PageCount; pageNr++ {
		leafRef, err := ctx.XRefTable.PageDictIndRef(pageNr)
		if err != nil {
			t.Fatalf("%s: PageDictIndRef(%d): %v", msg, pageNr, err)
		}

		cur := *leafRef
		terminated := false
		for step := 0; step <= walkLimit; step++ {
			d, err := ctx.XRefTable.DereferenceDict(cur)
			if err != nil {
				t.Fatalf("%s: page %d: DereferenceDict(obj#%d): %v",
					msg, pageNr, cur.ObjectNumber, err)
			}
			parentObj, hasParent := d["Parent"]
			if !hasParent {
				if cur.ObjectNumber != rootRef.ObjectNumber {
					t.Errorf("%s: page %d: chain terminated at obj#%d, expected catalog /Pages obj#%d",
						msg, pageNr, cur.ObjectNumber, rootRef.ObjectNumber)
				}
				terminated = true
				break
			}
			parentRef, ok := parentObj.(types.IndirectRef)
			if !ok {
				t.Fatalf("%s: page %d: /Parent on obj#%d is not an indirect ref: %T",
					msg, pageNr, cur.ObjectNumber, parentObj)
			}
			cur = parentRef
		}
		if !terminated {
			t.Fatalf("%s: page %d: /Parent chain did not terminate within %d steps",
				msg, pageNr, walkLimit)
		}
	}

	t.Logf("%s: %d pages × parent chains all terminate at catalog /Pages obj#%d",
		msg, ctx.PageCount, rootRef.ObjectNumber)
}

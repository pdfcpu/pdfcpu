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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func pageTreeDepth(t *testing.T, ctx *model.Context, indRef types.IndirectRef, currentDepth int) int {
	t.Helper()

	const sanityLimit = 5000
	if currentDepth > sanityLimit {
		t.Fatalf("page tree walk exceeded sanity limit %d", sanityLimit)
	}

	d, err := ctx.DereferenceDict(indRef)
	if err != nil {
		t.Fatalf("DereferenceDict(%v): %v", indRef, err)
	}

	if typ := d.Type(); typ != nil && *typ == "Page" {
		return currentDepth + 1
	}

	kids := d.ArrayEntry("Kids")
	if len(kids) == 0 {
		return currentDepth + 1
	}

	maxDepth := currentDepth + 1
	for _, obj := range kids {
		kidIndRef, ok := obj.(types.IndirectRef)
		if !ok {
			t.Fatalf("page tree kid is not an indirect reference: %T", obj)
		}
		depth := pageTreeDepth(t, ctx, kidIndRef, currentDepth+1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

func mergePageTreeInputs(
	t *testing.T,
	inFile, outFile string,
	inputCount int,
	dividerPage bool,
) *model.Context {
	t.Helper()

	inFiles := make([]string, inputCount)
	for i := range inFiles {
		inFiles[i] = inFile
	}

	if err := api.MergeCreateFile(t.Context(), inFiles, outFile, dividerPage, nil); err != nil {
		t.Fatalf("MergeCreateFile: %v", err)
	}
	if err := api.ValidateFile(t.Context(), outFile, conf, nil); err != nil {
		t.Fatalf("ValidateFile: %v", err)
	}

	ctx, err := api.ReadContextFile(t.Context(), outFile)
	if err != nil {
		t.Fatalf("ReadContextFile: %v", err)
	}

	return ctx
}

func assertPageTreeDepth(t *testing.T, ctx *model.Context, inputCount, maxDepth int) {
	t.Helper()

	rootIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatalf("ctx.Pages: %v", err)
	}

	if depth := pageTreeDepth(t, ctx, *rootIndRef, 0); depth > maxDepth {
		t.Fatalf("page tree depth after merging %d inputs: got %d, want <= %d", inputCount, depth, maxDepth)
	}
}

// TestMergeProducesFlatPageTree verifies that page tree depth remains bounded across repeated merges.
func TestMergeProducesFlatPageTree(t *testing.T) {
	const (
		inputCount = 100
		maxDepth   = 10
	)

	ctx := mergePageTreeInputs(
		t,
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(outDir, "merge_flat_tree.pdf"),
		inputCount,
		false,
	)
	assertPageTreeDepth(t, ctx, inputCount, maxDepth)
}

// TestMergeProducesFlatPageTree_WithDivider verifies bounded depth when inserting divider pages.
func TestMergeProducesFlatPageTree_WithDivider(t *testing.T) {
	const (
		inputCount = 20
		maxDepth   = 10
	)

	ctx := mergePageTreeInputs(
		t,
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(outDir, "merge_flat_tree_with_divider.pdf"),
		inputCount,
		true,
	)
	assertPageTreeDepth(t, ctx, inputCount, maxDepth)
}

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

func inheritedPageAttrs(ctx *model.Context) ([]*model.InheritedPageAttrs, error) {
	attrs := make([]*model.InheritedPageAttrs, ctx.PageCount+1)
	for pageNr := 1; pageNr <= ctx.PageCount; pageNr++ {
		_, _, pageAttrs, err := ctx.PageDict(pageNr, false)
		if err != nil {
			return nil, err
		}
		attrs[pageNr] = pageAttrs
	}
	return attrs, nil
}

func compareResourceKeys(
	t *testing.T,
	pageNr, srcPageNr int,
	want, got types.Dict,
) {
	t.Helper()

	switch {
	case want == nil && got != nil:
		t.Errorf("page %d (src %d): Resources appeared after merge", pageNr, srcPageNr)
	case want != nil && got == nil:
		t.Errorf("page %d (src %d): Resources lost after merge", pageNr, srcPageNr)
	case want != nil && got != nil:
		for key := range want {
			if _, ok := got[key]; !ok {
				t.Errorf("page %d (src %d): Resources missing key %q", pageNr, srcPageNr, key)
			}
		}
	}
}

func compareInheritedPageAttrs(
	t *testing.T,
	pageNr, srcPageNr int,
	want, got *model.InheritedPageAttrs,
) {
	t.Helper()

	if !rectsEqual(want.MediaBox, got.MediaBox) {
		t.Errorf("page %d (src %d): MediaBox = %v, want %v", pageNr, srcPageNr, got.MediaBox, want.MediaBox)
	}
	if !rectsEqual(want.CropBox, got.CropBox) {
		t.Errorf("page %d (src %d): CropBox = %v, want %v", pageNr, srcPageNr, got.CropBox, want.CropBox)
	}
	if got.Rotate != want.Rotate {
		t.Errorf("page %d (src %d): Rotate = %d, want %d", pageNr, srcPageNr, got.Rotate, want.Rotate)
	}
	compareResourceKeys(t, pageNr, srcPageNr, want.Resources, got.Resources)
}

// TestMergePreservesInheritedPageAttrs verifies that merged pages retain their source attributes.
func TestMergePreservesInheritedPageAttrs(t *testing.T) {
	const inputCount = 3

	inFile := filepath.Join(inDir, "BuildingWebappsWithGo.pdf")
	srcCtx, err := api.ReadContextFile(t.Context(), inFile)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := inheritedPageAttrs(srcCtx)
	if err != nil {
		t.Fatal(err)
	}
	if expected[1].MediaBox == nil {
		t.Fatalf("fixture %s no longer inherits /MediaBox", inFile)
	}

	mergedCtx := mergePageTreeInputs(
		t,
		inFile,
		filepath.Join(outDir, "merge_inherited_attrs.pdf"),
		inputCount,
		false,
	)

	wantPageCount := inputCount * srcCtx.PageCount
	if mergedCtx.PageCount != wantPageCount {
		t.Fatalf("merged page count: got %d, want %d", mergedCtx.PageCount, wantPageCount)
	}

	for pageNr := 1; pageNr <= mergedCtx.PageCount; pageNr++ {
		_, _, got, err := mergedCtx.PageDict(pageNr, false)
		if err != nil {
			t.Fatalf("PageDict(%d): %v", pageNr, err)
		}
		srcPageNr := ((pageNr - 1) % srcCtx.PageCount) + 1
		compareInheritedPageAttrs(t, pageNr, srcPageNr, expected[srcPageNr], got)
	}
}

func parentChainEndsAtRoot(
	ctx *model.Context,
	leafIndRef, rootIndRef types.IndirectRef,
) error {
	const walkLimit = 50

	current := leafIndRef
	for range walkLimit {
		d, err := ctx.DereferenceDict(current)
		if err != nil {
			return err
		}

		parent, ok := d["Parent"]
		if !ok {
			if current.ObjectNumber != rootIndRef.ObjectNumber {
				return fmt.Errorf("chain terminated at obj#%d, want obj#%d", current.ObjectNumber, rootIndRef.ObjectNumber)
			}
			return nil
		}

		parentIndRef, ok := parent.(types.IndirectRef)
		if !ok {
			return fmt.Errorf("/Parent on obj#%d is not an indirect reference: %T", current.ObjectNumber, parent)
		}
		current = parentIndRef
	}

	return fmt.Errorf("parent chain did not terminate within %d steps", walkLimit)
}

// TestMergePageTreeParentLinks verifies that every merged page links back to the catalog page root.
func TestMergePageTreeParentLinks(t *testing.T) {
	const inputCount = 5

	ctx := mergePageTreeInputs(
		t,
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(outDir, "merge_parent_links.pdf"),
		inputCount,
		false,
	)

	rootIndRef, err := ctx.Pages()
	if err != nil {
		t.Fatal(err)
	}
	root, err := ctx.DereferenceDict(*rootIndRef)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := root["Parent"]; ok {
		t.Fatalf("catalog /Pages root obj#%d has a /Parent", rootIndRef.ObjectNumber)
	}

	for pageNr := 1; pageNr <= ctx.PageCount; pageNr++ {
		leafIndRef, err := ctx.PageDictIndRef(pageNr)
		if err != nil {
			t.Fatalf("PageDictIndRef(%d): %v", pageNr, err)
		}
		if err := parentChainEndsAtRoot(ctx, *leafIndRef, *rootIndRef); err != nil {
			t.Fatalf("page %d: %v", pageNr, err)
		}
	}
}

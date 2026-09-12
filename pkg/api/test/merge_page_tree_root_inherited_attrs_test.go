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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func removePageTreeAttr(ctx *model.Context, indRef types.IndirectRef, key string) error {
	d, err := ctx.DereferenceDict(indRef)
	if err != nil {
		return err
	}

	delete(d, key)

	for _, obj := range d.ArrayEntry("Kids") {
		indRef, ok := obj.(types.IndirectRef)
		if !ok {
			return fmt.Errorf("page tree kid is not an indirect reference: %T", obj)
		}
		if err := removePageTreeAttr(ctx, indRef, key); err != nil {
			return err
		}
	}

	return nil
}

// TestMergePageTreeInheritedAttrsIsolated ensures destination page attributes do not leak into source pages.
func TestMergePageTreeInheritedAttrsIsolated(t *testing.T) {
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	ctxDest, err := api.ReadContextFile(t.Context(), inFile)
	if err != nil {
		t.Fatal(err)
	}
	ctxSrc, err := api.ReadContextFile(t.Context(), inFile)
	if err != nil {
		t.Fatal(err)
	}

	destPageCount := ctxDest.PageCount
	destRootIndRef, err := ctxDest.Pages()
	if err != nil {
		t.Fatal(err)
	}
	destRoot, err := ctxDest.DereferenceDict(*destRootIndRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := removePageTreeAttr(ctxDest, *destRootIndRef, "Rotate"); err != nil {
		t.Fatal(err)
	}
	destRoot["Rotate"] = types.Integer(90)

	srcRootIndRef, err := ctxSrc.Pages()
	if err != nil {
		t.Fatal(err)
	}
	if err := removePageTreeAttr(ctxSrc, *srcRootIndRef, "Rotate"); err != nil {
		t.Fatal(err)
	}

	if err := pdfcpu.MergeXRefTables(t.Context(), "", ctxSrc, ctxDest, false, false); err != nil {
		t.Fatal(err)
	}

	_, _, destAttrs, err := ctxDest.PageDict(t.Context(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if destAttrs.Rotate != 90 {
		t.Fatalf("destination page rotation: got %d, want 90", destAttrs.Rotate)
	}

	_, _, srcAttrs, err := ctxDest.PageDict(t.Context(), destPageCount+1, false)
	if err != nil {
		t.Fatal(err)
	}
	if srcAttrs.Rotate != 0 {
		t.Fatalf("source page rotation: got %d, want 0", srcAttrs.Rotate)
	}
}

/*
Copyright 2020 The pdf Authors.

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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestInsertRemovePages verifies insert remove pages.
func TestInsertRemovePages(t *testing.T) {
	msg := "TestInsertRemovePages"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	n1, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Insert an empty page before pages 1 and 2.
	if err := api.InsertPagesFile(inFile, outFile, []string{"-2"}, true, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	n2, err := api.PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if n2 != n1+2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1+2, n2)
	}

	// 	// Remove pages 1 and 2.
	if err := api.RemovePagesFile(outFile, "", []string{"-2"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	n2, err = api.PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n1 != n2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1, n2)
	}
}

// TestRemovePagesDropsNamedDestsForRemovedPages verifies that stale named destinations do not break kept links.
func TestRemovePagesDropsNamedDestsForRemovedPages(t *testing.T) {
	msg := "TestRemovePagesDropsNamedDestsForRemovedPages"
	inFile := filepath.Join(inDir, "Walden.pdf")

	f, err := os.Open(inFile)
	if err != nil {
		t.Fatalf("%s open: %v\n", msg, err)
	}
	defer f.Close()

	ctx, err := api.ReadValidateAndOptimize(f, model.NewDefaultConfiguration())
	if err != nil {
		t.Fatalf("%s read: %v\n", msg, err)
	}

	if err := ctx.LocateNameTree("Dests", true); err != nil {
		t.Fatalf("%s locate dests: %v\n", msg, err)
	}

	_, keptPage, _, err := ctx.PageDict(1, false)
	if err != nil {
		t.Fatalf("%s page 1: %v\n", msg, err)
	}
	_, removedPage, _, err := ctx.PageDict(ctx.PageCount, false)
	if err != nil {
		t.Fatalf("%s page %d: %v\n", msg, ctx.PageCount, err)
	}

	if err := ctx.Names["Dests"].Add(ctx.XRefTable, "kept", destArray(*keptPage), nil, nil); err != nil {
		t.Fatalf("%s add kept dest: %v\n", msg, err)
	}
	if err := ctx.Names["Dests"].Add(ctx.XRefTable, "removed", destArray(*removedPage), nil, nil); err != nil {
		t.Fatalf("%s add removed dest: %v\n", msg, err)
	}

	ctxNew, err := pdfcpu.ExtractPages(ctx, []int{1}, false)
	if err != nil {
		t.Fatalf("%s extract: %v\n", msg, err)
	}

	if _, err := ctxNew.DereferenceDestArray("kept"); err != nil {
		t.Fatalf("%s kept dest: %v\n", msg, err)
	}
	if _, ok := ctxNew.Names["Dests"].Value("removed"); ok {
		t.Fatalf("%s removed page destination still present\n", msg)
	}

	var buf bytes.Buffer
	if err := api.WriteContext(ctxNew, &buf); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}
	if err := api.Validate(bytes.NewReader(buf.Bytes()), nil); err != nil {
		t.Fatalf("%s validate: %v\n", msg, err)
	}
}

func destArray(page types.IndirectRef) types.Array {
	return types.Array{page, types.Name("Fit")}
}

// TestInsertCustomBlankPage verifies insert custom blank page.
func TestInsertCustomBlankPage(t *testing.T) {
	msg := "TestInsertCustomBlankPage"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	selectedPages := []string{"2"}

	before := false

	pageConf, err := pdfcpu.ParsePageConfiguration("f:A5L", conf.Unit)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Insert an empty A5 page in landscape mode after page 5.
	if err := api.InsertPagesFile(inFile, outFile, selectedPages, before, pageConf, conf); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	selectedPages = []string{"odd"}

	pageConf, err = pdfcpu.ParsePageConfiguration("dim:5 10", types.CENTIMETRES)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Insert an empty page with dimensions 5 x 10 cm after every odd page.
	if err := api.InsertPagesFile(inFile, outFile, selectedPages, before, pageConf, conf); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

}

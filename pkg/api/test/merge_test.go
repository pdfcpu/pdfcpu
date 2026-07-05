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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestMergeCreateNew verifies merge create new.
func TestMergeCreateNew(t *testing.T) {
	msg := "TestMergeCreate"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// outFile will be overwritten.

	// Bookmarks for the merged document will be created/preserved per default (see config.yaml)

	outFile := filepath.Join(outDir, "out.pdf")
	if err := api.MergeCreateFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Insert an empty page between merged files.
	outFile = filepath.Join(outDir, "outWithDivider.pdf")
	dividerPage := true
	if err := api.MergeCreateFile(inFiles, outFile, dividerPage, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// TestMergeCreateZipped verifies merge create zipped.
func TestMergeCreateZipped(t *testing.T) {
	msg := "TestMergeCreateZipped"

	// The actual usecase for this is the recombination of 2 PDF files representing even and odd pages of some PDF source.
	// See #716
	inFile1 := filepath.Join(inDir, "Acroforms2.pdf")
	inFile2 := filepath.Join(inDir, "adobe_errata.pdf")
	outFile := filepath.Join(outDir, "out.pdf")

	if err := api.MergeCreateZipFile(inFile1, inFile2, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// TestMergeCreatePreserveBookmarks verifies merge create preserves source bookmarks without filename wrappers.
func TestMergeCreatePreserveBookmarks(t *testing.T) {
	msg := "TestMergeCreatePreserveBookmarks"
	inFile := filepath.Join(samplesDir, "bookmarks", "bookmarkTree.pdf")
	inFiles := []string{inFile, inFile}
	outFile := filepath.Join(outDir, "outPreserveBookmarks1.pdf")
	conf := model.NewDefaultConfiguration()
	conf.MergeBookmarkMode = model.MergeBookmarkModePreserve

	if err := api.MergeCreateFile(inFiles, outFile, false, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	f, err := os.Open(outFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	defer f.Close()

	bms, err := api.Bookmarks(f, conf)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if len(bms) != 4 {
		t.Fatalf("%s: got %d top-level bookmarks, want 4", msg, len(bms))
	}
	for _, bm := range bms {
		if bm.Title == "bookmarkTree.pdf" {
			t.Fatalf("%s: found filename wrapper bookmark", msg)
		}
	}
}

// TestMergeCreateDuplicateBookmarkDestinations verifies duplicate merge bookmark titles resolve to their own pages.
func TestMergeCreateDuplicateBookmarkDestinations(t *testing.T) {
	msg := "TestMergeCreateDuplicateBookmarkDestinations"
	inFile := filepath.Join(inDir, "test.pdf")
	inFiles := []string{inFile, inFile, inFile, inFile}
	outFile := filepath.Join(outDir, "outDuplicateBookmarkDestinations.pdf")

	if err := api.MergeCreateFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	f, err := os.Open(outFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	defer f.Close()

	bms, err := api.Bookmarks(f, nil)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if len(bms) != len(inFiles) {
		t.Fatalf("%s: got %d bookmarks, want %d", msg, len(bms), len(inFiles))
	}
	for i, bm := range bms {
		if bm.Title != "test.pdf" {
			t.Fatalf("%s: bookmark %d title = %q, want test.pdf", msg, i, bm.Title)
		}
		if bm.PageFrom != i+1 {
			t.Fatalf("%s: bookmark %d page = %d, want %d", msg, i, bm.PageFrom, i+1)
		}
	}
}

func writeOrphanWidgetFieldPDF(t *testing.T, inFile, outFile string) {
	t.Helper()

	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("ReadContextFile: %v", err)
	}
	ctx.RootDict.Delete("AcroForm")
	ctx.XRefTable.Form = nil
	if err := api.WriteContextFile(ctx, outFile); err != nil {
		t.Fatalf("WriteContextFile: %v", err)
	}
}

func widgetFieldNames(t *testing.T, ctx *model.Context) []string {
	t.Helper()

	var names []string
	for _, entry := range ctx.Table {
		if entry.Free {
			continue
		}
		d, ok := entry.Object.(types.Dict)
		if !ok {
			continue
		}
		if typ := d.NameEntry("Subtype"); typ == nil || *typ != "Widget" {
			continue
		}
		name, err := d.StringOrHexLiteralEntry("T")
		if err != nil {
			t.Fatalf("StringOrHexLiteralEntry: %v", err)
		}
		if name != nil {
			names = append(names, *name)
		}
	}
	return names
}

func assertUniqueWidgetFieldNames(t *testing.T, names []string) {
	t.Helper()

	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			t.Fatalf("duplicate widget field name %q in %v", name, names)
		}
		seen[name] = true
	}
}

func containsNamespacedWidgetField(names []string) bool {
	for _, name := range names {
		if strings.Contains(name, ".") {
			return true
		}
	}
	return false
}

// TestMergeCreateRenamesOrphanWidgetFields verifies orphan widget fields do not get linked by name after merging.
func TestMergeCreateRenamesOrphanWidgetFields(t *testing.T) {
	msg := "TestMergeCreateRenamesOrphanWidgetFields"
	inFile := filepath.Join(samplesDir, "form", "primitives", "textfield.pdf")
	orphanFile := filepath.Join(outDir, "orphanWidgetFields.pdf")
	writeOrphanWidgetFieldPDF(t, inFile, orphanFile)

	outFile := filepath.Join(outDir, "outOrphanWidgetFields.pdf")
	if err := api.MergeCreateFile([]string{orphanFile, orphanFile}, outFile, false, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	ctx, err := api.ReadContextFile(outFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	names := widgetFieldNames(t, ctx)
	if len(names) < 2 {
		t.Fatalf("%s: got %d widget field names, want at least 2", msg, len(names))
	}
	assertUniqueWidgetFieldNames(t, names)

	if !containsNamespacedWidgetField(names) {
		t.Fatalf("%s: no widget field name is namespaced: %v", msg, names)
	}
}

// TestMergeAppendNew verifies merge append new.
func TestMergeAppendNew(t *testing.T) {
	msg := "TestMergeAppend"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")
	if err := copyFile(t, filepath.Join(inDir, "test.pdf"), outFile); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// If outFile already exists its content will be preserved and serves as the beginning of the merge result.

	// Bookmarks for the merged document will be created/preserved per default (see config.yaml)

	if err := api.MergeAppendFile(inFiles, outFile, false, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	anotherFile := filepath.Join(inDir, "testRot.pdf")
	err := api.MergeAppendFile([]string{anotherFile}, outFile, false, nil)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// TestMergeToBufNew verifies merge to buf new.
func TestMergeToBufNew(t *testing.T) {
	msg := "TestMergeToBuf"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	destFile := inFiles[0]
	inFiles = inFiles[1:]

	buf := &bytes.Buffer{}
	if err := api.Merge(destFile, inFiles, buf, nil, false); err != nil {
		t.Fatalf("%s: merge: %v\n", msg, err)
	}

	if err := os.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("%s: write: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// TestMergeRaw verifies merge raw.
func TestMergeRaw(t *testing.T) {
	msg := "TestMergeRaw"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	var rsc []io.ReadSeeker = make([]io.ReadSeeker, 2)

	f0, err := os.Open(inFiles[0])
	if err != nil {
		t.Fatalf("%s: open file1: %v\n", msg, err)
	}
	defer f0.Close()
	rsc[0] = f0

	f1, err := os.Open(inFiles[1])
	if err != nil {
		t.Fatalf("%s: open file2: %v\n", msg, err)
	}
	defer f1.Close()
	rsc[1] = f1

	buf := &bytes.Buffer{}
	if err := api.MergeRaw(rsc, buf, false, nil); err != nil {
		t.Fatalf("%s: merge: %v\n", msg, err)
	}

	if err := os.WriteFile(outFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("%s: write: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

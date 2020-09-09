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
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestSplitSpan1(t *testing.T) {
	msg := "TestSplitSpan1"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Create single page files of inFile in outDir.
	span := 1
	if err := api.SplitFile(inFile, outDir, span, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestSplitSpan2(t *testing.T) {
	msg := "TestSplitSpan2"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Create dual page files of inFile in outDir.
	span := 2
	if err := api.SplitFile(inFile, outDir, span, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestSplit0ByBookmark(t *testing.T) {
	msg := "TestSplit0ByBookmark"
	fileName := "5116.DCT_Filter.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Split along bookmarks.
	span := 0
	if err := api.SplitFile(inFile, outDir, span, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestSplitLowLevel(t *testing.T) {
	msg := "TestSplitLowLevel"
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	outFile := filepath.Join(outDir, "MyExtractedPageSpan.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract a page span.
	from, thru := 2, 4
	selectedPages := api.PagesForPageRange(from, thru)
	usePgCache := false
	ctxNew, err := ctx.ExtractPages(selectedPages, usePgCache)
	if err != nil {
		t.Fatalf("%s ExtractPages(%d,%d): %v\n", msg, from, thru, err)
	}

	// Here you can process this single page PDF context.

	// Write context to file.
	if err := api.WriteContextFile(ctxNew, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}
}

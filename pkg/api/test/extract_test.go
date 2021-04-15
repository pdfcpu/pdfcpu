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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func TestExtractImages(t *testing.T) {
	msg := "TestExtractImages"
	// Extract images for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		// Test writing files
		fn = filepath.Join(inDir, fn)
		if err := api.ExtractImagesFile(fn, outDir, nil, nil); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}
	// Extract images for inFile starting with page 1 into outDir.
	inFile := filepath.Join(inDir, "testImage.pdf")
	if err := api.ExtractImagesFile(inFile, outDir, []string{"1-"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractImagesLowLevel(t *testing.T) {
	msg := "TestExtractImagesLowLevel"
	fileName := "testImage.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Optimize resource usage of this context.
	if err := api.OptimizeContext(ctx); err != nil {
		t.Fatalf("%s optimizeContext: %v\n", msg, err)
	}

	// Extract images for page 1.
	i := 1
	ii, err := ctx.ExtractPageImages(i)
	if err != nil {
		t.Fatalf("%s extractPageFonts(%d): %v\n", msg, i, err)
	}

	baseFileName := strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	// Process extracted images.
	for _, img := range ii {
		fn := filepath.Join(outDir, fmt.Sprintf("%s_%d_%s.%s", baseFileName, i, img.Name, img.Type))
		if err := pdfcpu.WriteReader(fn, img); err != nil {
			t.Fatalf("%s write: %s", msg, fn)
		}
	}
}

func TestExtractFonts(t *testing.T) {
	msg := "TestExtractFonts"
	// Extract fonts for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = filepath.Join(inDir, fn)
		if err := api.ExtractFontsFile(fn, outDir, nil, nil); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}
	// Extract fonts for inFile for pages 1-3 into outDir.
	inFile := filepath.Join(inDir, "go.pdf")
	if err := api.ExtractFontsFile(inFile, outDir, []string{"1-3"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractFontsLowLevel(t *testing.T) {
	msg := "TestExtractFontsLowLevel"
	inFile := filepath.Join(inDir, "go.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Optimize resource usage of this context.
	if err := api.OptimizeContext(ctx); err != nil {
		t.Fatalf("%s optimizeContext: %v\n", msg, err)
	}

	// Extract fonts for page 1.
	i := 1
	ff, err := ctx.ExtractPageFonts(i)
	if err != nil {
		t.Fatalf("%s extractPageFonts(%d): %v\n", msg, i, err)
	}

	// Process extracted fonts.
	for _, f := range ff {
		fn := filepath.Join(outDir, fmt.Sprintf("%s.%s", f.Name, f.Type))
		if err := pdfcpu.WriteReader(fn, f); err != nil {
			t.Fatalf("%s write: %s", msg, fn)
		}
	}
}

func TestExtractPages(t *testing.T) {
	msg := "TestExtractPages"
	// Extract page #1 into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := api.ExtractPagesFile(inFile, outDir, []string{"1"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractPagesLowLevel(t *testing.T) {
	msg := "TestExtractPagesLowLevel"
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	outFile := filepath.Join(outDir, "MyExtractedAndProcessedSinglePage.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract page 1.
	i := 1
	ctxNew, err := ctx.ExtractPage(i)
	if err != nil {
		t.Fatalf("%s extractPage(%d): %v\n", msg, i, err)
	}

	// Here you can process this single page PDF context.

	// Write context to file.
	if err := api.WriteContextFile(ctxNew, outFile); err != nil {
		t.Fatalf("%s write: %v\n", msg, err)
	}
}

func TestExtractContent(t *testing.T) {
	msg := "TestExtractContent"
	// Extract content of all pages into outDir.
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	if err := api.ExtractContentFile(inFile, outDir, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractContentLowLevel(t *testing.T) {
	msg := "TestExtractContentLowLevel"
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract page content for page 2.
	i := 2
	r, err := ctx.ExtractPageContent(i)
	if err != nil {
		t.Fatalf("%s extractPageContent(%d): %v\n", msg, i, err)
	}

	// Process page content.
	bb, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("%s readAll: %v\n", msg, err)
	}
	t.Logf("Page content (PDF-syntax) for page %d:\n%s", i, string(bb))
}

func TestExtractMetadata(t *testing.T) {
	msg := "TestExtractMetadata"
	// Extract all metadata into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := api.ExtractMetadataFile(inFile, outDir, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractMetadataLowLevel(t *testing.T) {
	msg := "TestExtractMedadataLowLevel"
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")

	// Create a context.
	ctx, err := api.ReadContextFile(inFile)
	if err != nil {
		t.Fatalf("%s readContext: %v\n", msg, err)
	}

	// Extract all metadata.
	mm, err := ctx.ExtractMetadata()
	if err != nil {
		t.Fatalf("%s ExtractMetadata: %v\n", msg, err)
	}

	// Process metadata.
	for _, md := range mm {
		bb, err := ioutil.ReadAll(md)
		if err != nil {
			t.Fatalf("%s metadata readAll: %v\n", msg, err)
		}
		t.Logf("Metadata: objNr=%d parentDictObjNr=%d parentDictType=%s\n%s\n",
			md.ObjNr, md.ParentObjNr, md.ParentType, string(bb))
	}
}

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

func TestExtractImagesCommand(t *testing.T) {
	msg := "TestExtractImagesCommand"

	// Extract images for all pages into outDir.
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
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

func TestExtractFontsCommand(t *testing.T) {
	msg := "TestExtractFontsCommand"

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

func TestExtractContentCommand(t *testing.T) {
	msg := "TestExtractContentCommand"

	// Extract content of all pages into outDir.
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	if err := api.ExtractContentFile(inFile, outDir, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractPagesCommand(t *testing.T) {
	msg := "TestExtractPagesCommand"

	// Extract page #1 into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := api.ExtractPagesFile(inFile, outDir, []string{"1"}, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractMetadataCommand(t *testing.T) {
	msg := "TestExtractMetadataCommand"

	// Extract metadata into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	if err := api.ExtractMetadataFile(inFile, outDir, nil, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

/*
Copyright 2020 The pdfcpu Authors.

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

	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

func TestExtractImagesCommand(t *testing.T) {
	msg := "TestExtractImagesCommand"

	// Extract all images for each PDF file into outDir.
	cmd := cli.ExtractImagesCommand("", outDir, nil, nil)
	//for _, f := range allPDFs(t, inDir) {
	for _, f := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		inFile := filepath.Join(inDir, f)
		cmd.InFile = &inFile
		// Extract all images.
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", msg, inFile, err)
		}
	}

	// Extract all images for inFile starting with page 1 into outDir.
	inFile := filepath.Join(inDir, "testImage.pdf")
	cmd = cli.ExtractImagesCommand(inFile, outDir, []string{"1-"}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractFontsCommand(t *testing.T) {
	msg := "TestExtractFontsCommand"

	// Extract fonts for all pages for the following 3 PDF files into outDir.
	cmd := cli.ExtractFontsCommand("", outDir, nil, nil)
	for _, fn := range []string{"5116.DCT_Filter.pdf", "testImage.pdf", "go.pdf"} {
		fn = filepath.Join(inDir, fn)
		cmd.InFile = &fn
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", msg, fn, err)
		}
	}

	// Extract fonts for pages 1-3 into outDir.
	inFile := filepath.Join(inDir, "go.pdf")
	cmd = cli.ExtractFontsCommand(inFile, outDir, []string{"1-3"}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractPagesCommand(t *testing.T) {
	msg := "TestExtractPagesCommand"
	// Extract page #1 into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	cmd := cli.ExtractPagesCommand(inFile, outDir, []string{"1"}, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractContentCommand(t *testing.T) {
	msg := "TestExtractContentCommand"
	// Extract content of all pages into outDir.
	inFile := filepath.Join(inDir, "5116.DCT_Filter.pdf")
	cmd := cli.ExtractContentCommand(inFile, outDir, nil, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestExtractMetadataCommand(t *testing.T) {
	msg := "TestExtractMetadataCommand"
	// Extract metadata into outDir.
	inFile := filepath.Join(inDir, "TheGoProgrammingLanguageCh1.pdf")
	cmd := cli.ExtractMetadataCommand(inFile, outDir, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

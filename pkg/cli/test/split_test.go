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

	"github.com/mechiko/pdfcpu/pkg/cli"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/model"
)

// Split a test PDF file up into single page PDFs (using a split span of 1).
func TestSplitCommand(t *testing.T) {
	msg := "TestSplitCommand"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)
	span := 1

	conf := model.NewDefaultConfiguration()

	cmd := cli.SplitCommand(inFile, outDir, span, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s span=%d %s: %v\n", msg, span, inFile, err)
	}
}

// Split a test PDF file up into PDFs with 2 pages each (using a split span of 2).
func TestSplitBySpanCommand(t *testing.T) {
	msg := "TestSplitBySpanCommand"
	fileName := "CenterOfWhy.pdf"
	inFile := filepath.Join(inDir, fileName)
	span := 2

	cmd := cli.SplitCommand(inFile, outDir, span, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s span=%d %s: %v\n", msg, span, inFile, err)
	}
}

// Split a PDF along its defined bookmarks on level 1 or 2
func TestSplitByBookmarkCommand(t *testing.T) {
	msg := "TestSplitByBookmarkCommand"
	fileName := "5116.DCT_Filter.pdf"
	inFile := filepath.Join(inDir, fileName)

	span := 0 // This means we are going to split by bookmarks.

	cmd := cli.SplitCommand(inFile, outDir, span, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestSplitByPageNrCommand(t *testing.T) {
	msg := "TestSplitByPageNrCommand"
	fileName := "5116.DCT_Filter.pdf"
	inFile := filepath.Join(inDir, fileName)

	// Generate page section 1
	// Generate page section 2-9
	// Generate page section 10-49
	// Generate page section 50-last page

	cmd := cli.SplitByPageNrCommand(inFile, outDir, []int{2, 10, 50}, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

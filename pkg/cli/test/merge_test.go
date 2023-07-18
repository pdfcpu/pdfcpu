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

// Merge all PDFs in testdir into out/test.pdf.
func TestMergeCreateCommand(t *testing.T) {
	msg := "TestMergeCreateCommand"

	var inFiles []string
	for _, f := range allPDFs(t, inDir) {
		inFiles = append(inFiles, filepath.Join(inDir, f))
	}

	outFile := filepath.Join(outDir, "test.pdf")

	cmd := cli.MergeCreateCommand(inFiles, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeAppendCommand(t *testing.T) {
	msg := "TestMergeAppendCommand"

	var inFiles []string
	for _, f := range allPDFs(t, inDir) {
		if f == "test.pdf" {
			continue
		}
		inFiles = append(inFiles, filepath.Join(inDir, f))
	}

	outFile := filepath.Join(outDir, "test.pdf")

	if err := copyFile(t, filepath.Join(inDir, "test.pdf"), outFile); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// If outFile already exists its content will be preserved and serves as the beginning of the merge result.
	cmd := cli.MergeAppendCommand(inFiles, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

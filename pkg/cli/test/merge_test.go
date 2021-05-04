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
func TestMergeCommand(t *testing.T) {
	msg := "TestMergeCommand"

	inFiles := []string(nil)
	for _, f := range allPDFs(t, inDir) {
		inFile := filepath.Join(inDir, f)
		inFiles = append(inFiles, inFile)
	}

	outFile := filepath.Join(outDir, "test.pdf")
	cmd := cli.MergeCreateCommand(inFiles, outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	cmd = cli.ValidateCommand(outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := copyFile(t, filepath.Join(inDir, "test.pdf"), outFile); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	cmd = cli.MergeAppendCommand(inFiles, outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	cmd = cli.ValidateCommand(outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

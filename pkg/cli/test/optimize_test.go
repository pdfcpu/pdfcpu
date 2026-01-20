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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func optimizeFile(t *testing.T, fileName string, conf *model.Configuration) error {
	t.Helper()
	cmd := cli.OptimizeCommand(fileName, "", conf)
	_, err := cli.Process(cmd)
	return err
}

func testOptimizeFile(t *testing.T, inFile, outFile string) {
	t.Helper()
	msg := "testOptimizeFile"

	// Optimize inFile and write result to outFile.
	cmd := cli.OptimizeCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Optimize outFile and write result to outFile.
	cmd = cli.OptimizeCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Optimize outFile and write result to outFile.
	if err := optimizeFile(t, outFile, nil); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
}

func TestOptimizeCommand(t *testing.T) {
	for _, f := range allPDFs(t, inDir) {
		inFile := filepath.Join(inDir, f)
		outFile := filepath.Join(outDir, f)
		testOptimizeFile(t, inFile, outFile)
	}
}

// TestOptimizeCircularReference tests that the optimize CLI command handles PDFs with circular
// object references without causing a stack overflow. This test uses a minimal PDF with
// circular references in Resources dictionaries that previously caused infinite recursion
// in EqualObjects.
func TestOptimizeCircularReference(t *testing.T) {
	msg := "TestOptimizeCircularReference"
	fileName := "circular_ref_test.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// This PDF has two Form XObjects with identical stream lengths that
	// reference each other in their Resources via ProcSet, creating a cycle.
	// Without cycle detection in EqualObjects, this causes infinite recursion.
	cmd := cli.OptimizeCommand(inFile, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: optimize should handle circular references without stack overflow: %v\n", msg, err)
	}

	// Test that we can optimize it again (in-place)
	cmd = cli.OptimizeCommand(outFile, "", conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: second optimize should also succeed: %v\n", msg, err)
	}
}

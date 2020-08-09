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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func optimizeFile(t *testing.T, fileName string, conf *pdfcpu.Configuration) error {
	t.Helper()
	cmd := cli.OptimizeCommand(fileName, "", conf)
	_, err := cli.Process(cmd)
	return err
}

func testOptimizeFile(t *testing.T, inFile, outFile string) {
	t.Helper()
	msg := "testOptimizeFile"

	// Optimize inFile and write result to outFile.
	cmd := cli.OptimizeCommand(inFile, outFile, nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Optimize outFile and write result to outFile.
	cmd = cli.OptimizeCommand(outFile, "", nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	// Optimize outFile and write result to outFile.
	// Also skip validation.
	c := pdfcpu.NewDefaultConfiguration()
	c.ValidationMode = pdfcpu.ValidationNone

	if err := optimizeFile(t, outFile, c); err != nil {
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

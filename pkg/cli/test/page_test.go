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

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

func TestPagesCommand(t *testing.T) {
	msg := "TestPagesCommand"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")

	n1, err := api.PageCountFile(inFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	// Insert an empty page before pages 1 and 2.
	cmd := cli.InsertPagesCommand(inFile, outFile, []string{"-2"}, conf, "before", nil)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	n2, err := api.PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n2 != n1+2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1+2, n2)
	}

	// Remove pages 1 and 2.
	cmd = cli.RemovePagesCommand(outFile, "", []string{"-2"}, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	n2, err = api.PageCountFile(outFile)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if n1 != n2 {
		t.Fatalf("%s %s: pageCount want:%d got:%d\n", msg, inFile, n1, n2)
	}
}

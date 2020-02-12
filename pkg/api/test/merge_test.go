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

func TestMergeCreate(t *testing.T) {
	msg := "TestMergeCreate"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// outFile will be overwritten.
	if err := api.MergeCreateFile(inFiles, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

func TestMergeAppend(t *testing.T) {
	msg := "TestMergeAppend"
	inFiles := []string{
		filepath.Join(inDir, "Acroforms2.pdf"),
		filepath.Join(inDir, "adobe_errata.pdf"),
	}
	outFile := filepath.Join(outDir, "test.pdf")

	// Merge inFiles by concatenation in the order specified and write the result to outFile.
	// If outFile already exists its content will be preserved and serves as the beginning of the merge result.
	if err := api.MergeAppendFile(inFiles, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

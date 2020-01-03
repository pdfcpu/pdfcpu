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
)

func TestTrim(t *testing.T) {
	msg := "TestTrim"
	fileName := "adobe_errata.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// Create a trimmed version of inFile containing odd page numbers only.
	if err := api.TrimFile(inFile, outFile, []string{"odd"}, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Create a trimmed version of inFile containing the first two pages only.
	// If you want to modify the original file, pass an empty string for outFile.
	inFile = outFile
	if err := api.TrimFile(inFile, "", []string{"1-2"}, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

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

func TestRotate(t *testing.T) {
	msg := "TestRotate"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// Rotate all pages of inFile, clockwise by 90 degrees and write the result to outFile.
	if err := api.RotateFile(inFile, outFile, 90, nil, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Rotate the first page of inFile by 180 degrees.
	// If you want to modify the original file, pass an empty string for outFile.
	inFile = outFile
	if err := api.RotateFile(inFile, "", 180, []string{"1"}, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

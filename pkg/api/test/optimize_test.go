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

func TestOptimize(t *testing.T) {
	msg := "TestOptimize"
	fileName := "Acroforms2.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// Create an optimized version of inFile.
	if err := api.OptimizeFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Create an optimized version of inFile.
	// If you want to modify the original file, pass an empty string for outFile.
	inFile = outFile
	if err := api.OptimizeFile(inFile, "", nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

// TestOptimizeCircularReference tests that optimize handles PDFs with circular
// object references without causing a stack overflow. This test uses a minimal
// PDF with circular references in Resources dictionaries that previously caused
// infinite recursion in EqualObjects.
func TestOptimizeCircularReference(t *testing.T) {
	msg := "TestOptimizeCircularReference"
	fileName := "circular_ref_test.pdf"
	inFile := filepath.Join(inDir, fileName)
	outFile := filepath.Join(outDir, fileName)

	// This PDF has two Form XObjects with identical stream lengths that
	// reference each other in their Resources via ProcSet, creating a cycle.
	// Without cycle detection in EqualObjects, this causes infinite recursion.
	if err := api.OptimizeFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s: optimize should handle circular references without stack overflow: %v\n", msg, err)
	}

	// Test that we can optimize it again (in-place)
	if err := api.OptimizeFile(outFile, "", nil); err != nil {
		t.Fatalf("%s: second optimize should also succeed: %v\n", msg, err)
	}
}

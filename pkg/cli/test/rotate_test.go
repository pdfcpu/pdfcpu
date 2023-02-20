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

// Rotate first 2 pages clockwise by 90 degrees.
func TestRotateCommand(t *testing.T) {
	msg := "TestRotateCommand"
	inFile := filepath.Join(inDir, "Acroforms2.pdf")
	outFile := filepath.Join(outDir, "test.pdf")
	rotation := 90

	cmd := cli.RotateCommand(inFile, outFile, rotation, []string{"-2"}, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
}

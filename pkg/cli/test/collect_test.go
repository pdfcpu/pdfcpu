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

	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

// Create a custom page sequence.
func TestCollectCommand(t *testing.T) {
	msg := "TestCollectCommand"

	inFile := filepath.Join(inDir, "pike-stanford.pdf")
	outFile := filepath.Join(outDir, "myPageSequence.pdf")

	// Start with all odd pages but page 1, then append pages 8-11 and the last page.
	cmd := cli.CollectCommand(inFile, outFile, []string{"odd", "!1", "8-11", "l"}, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}

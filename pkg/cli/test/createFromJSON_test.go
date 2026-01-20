/*
Copyright 2023 The pdf Authors.

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

/**************************************************************
 * All form related processing is optimized for Adobe Reader! *
 **************************************************************/

func createPDF(t *testing.T, msg, inFile, inFileJSON, outFile string, conf *model.Configuration) {

	t.Helper()

	// inFile	inFileJSON 	outFile		action
	// ---------------------------------------------------------------
	// ""		jsonFile 	outfile		write outFile
	// inFile 	jsonFile	""			update (read and write inFile)
	// inFile 	jsonFile 	outFile		read inFile and write outFile

	if outFile == "" {
		outFile = inFile
	}

	cmd := cli.CreateCommand(inFile, inFileJSON, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, outFile, err)
	}

	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}

func TestCreateSinglePageDemoFormsViaJson(t *testing.T) {

	// Render single page demo forms for export, reset, lock, unlock and fill tests.

	inDirFormDemo := filepath.Join(inDir, "json", "form", "demoSinglePage")
	outDirFormDemo := outDir

	for _, tt := range []struct {
		msg        string
		inFileJSON string
		outFile    string
	}{

		{"TestFormDemoEN", "english.json", "english.pdf"},             // Core font (Helvetica)
		{"TestFormDemoUK", "ukrainian.json", "ukrainian.pdf"},         // User font (Roboto-Regular)
		{"TestFormDemoAR", "arabic.json", "arabic.pdf"},               // User font RTL (Roboto-Regular)
		{"TestFormDemoSC", "chineseSimple.json", "chineseSimple.pdf"}, // User font CJK (UnifontMedium)
		{"TestPersonFormDemo", "person.json", "person.pdf"},           // Person Form
	} {
		inFileJSON := filepath.Join(inDirFormDemo, tt.inFileJSON)
		outFile := filepath.Join(outDirFormDemo, tt.outFile)
		createPDF(t, tt.msg, "", inFileJSON, outFile, conf)
	}

	// For more comprehensive PDF creation tests please refer to api/test/createFromJSON_test.go
}

// TestCreateCircularReference tests that the create CLI command handles PDFs with circular
// object references without causing a stack overflow when appending content. This test uses
// a minimal PDF with circular references in Resources dictionaries that previously could
// cause infinite recursion in EqualObjects when ImageBox compares image stream dicts.
func TestCreateCircularReference(t *testing.T) {
	msg := "TestCreateCircularReference"
	fileName := "circular_ref_test.pdf"
	inFile := filepath.Join(inDir, fileName)
	
	// Use a simple JSON that creates a page with text
	inFileJSON := filepath.Join(inDir, "json", "create", "textAnchored.json")
	outFile := filepath.Join(outDir, "circular_ref_created.pdf")

	// This PDF has two Form XObjects with identical stream lengths that
	// reference each other in their Resources via ProcSet, creating a cycle.
	// Without cycle detection in EqualObjects, creating/appending to this PDF
	// could cause infinite recursion when comparing objects during ImageBox
	// image deduplication or other object comparisons.
	cmd := cli.CreateCommand(inFile, inFileJSON, outFile, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s: create should handle circular references without stack overflow: %v\n", msg, err)
	}

	// Validate the output
	if err := validateFile(t, outFile, conf); err != nil {
		t.Fatalf("%s: created PDF should be valid: %v\n", msg, err)
	}
}

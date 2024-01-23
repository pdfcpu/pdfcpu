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
)

/**************************************************************
 * All form related processing is optimized for Adobe Reader! *
 **************************************************************/

func TestListFormFields(t *testing.T) {

	msg := "TestListFormFields"
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")

	cmd := cli.ListFormFieldsCommand([]string{inFile}, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestRemoveFormFields(t *testing.T) {

	msg := "TestRemoveFormFields"
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")
	outFile := filepath.Join(outDir, "removedField.pdf")

	cmd := cli.RemoveFormFieldsCommand(inFile, outFile, []string{"dob1"}, conf)
	if _, err := cli.Process(cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

func TestResetFormFields(t *testing.T) {

	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
	}{
		{"TestResetFormCorefont", "english.pdf", "english-reset.pdf"},        // Core font (Helvetica)
		{"TestResetFormUserfont", "ukrainian.pdf", "ukrainian-reset.pdf"},    // User font (Roboto-Regular)
		{"TestFormRTL", "arabic.pdf", "arabic-reset.pdf"},                    // User font RTL (Roboto-Regular)
		{"TestResetFormCJK", "chineseSimple.pdf", "chineseSimple-reset.pdf"}, // User font CJK (UnifontMedium)
		{"TestResetPersonForm", "person.pdf", "person-reset.pdf"},            // Person Form
	} {
		inFile := filepath.Join(samplesDir, "form", "demoSinglePage", tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.ResetFormCommand(inFile, outFile, nil, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}

}

func TestLockFormFields(t *testing.T) {

	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
	}{
		{"TestLockFormEN", "english.pdf", "english-locked.pdf"},              // Core font (Helvetica)
		{"TestLockFormUK", "ukrainian.pdf", "ukrainian-locked.pdf"},          // User font (Roboto-Regular)
		{"TestLockFormRTL", "arabic.pdf", "arabic-locked.pdf"},               // User font RTL (Roboto-Regular)
		{"TestLockFormCJK", "chineseSimple.pdf", "chineseSimple-locked.pdf"}, // User font CJK (UnifontMedium)
		{"TestLockPersonForm", "person.pdf", "person-locked.pdf"},            // Person Form
	} {
		inFile := filepath.Join(samplesDir, "form", "demoSinglePage", tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.LockFormCommand(inFile, outFile, nil, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestUnlockFormFields(t *testing.T) {

	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
	}{
		{"TestUnlockFormEN", "english-locked.pdf", "english-unlocked.pdf"},              // Core font (Helvetica)
		{"TestUnlockFormUK", "ukrainian-locked.pdf", "ukrainian-unlocked.pdf"},          // User font (Roboto-Regular)
		{"TestUnlockFormRTL", "arabic-locked.pdf", "arabic-unlocked.pdf"},               // User font RTL (Roboto-Regular)
		{"TestUnlockFormCJK", "chineseSimple-locked.pdf", "chineseSimple-unlocked.pdf"}, // User font CJK (UnifontMedium)
		{"TestUnlockPersonForm", "person-locked.pdf", "person-unlocked.pdf"},            // Person Form
	} {
		inFile := filepath.Join(samplesDir, "form", "lock", tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.UnlockFormCommand(inFile, outFile, nil, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestExportForm(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")

	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
	}{
		{"TestExportFormEN", "english.pdf", "english.json"},              // Core font (Helvetica)
		{"TestExportFormUK", "ukrainian.pdf", "ukrainian.json"},          // User font (Roboto-Regular)
		{"TestExportFormRTL", "arabic.pdf", "arabic.json"},               // User font RTL (Roboto-Regular)
		{"TestExportFormCJK", "chineseSimple.pdf", "chineseSimple.json"}, // User font CJK (UnifontMedium)
		{"TestExportPersonForm", "person.pdf", "person.json"},            // Person Form
	} {
		inFile := filepath.Join(inDir, tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.ExportFormCommand(inFile, outFile, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestFillForm(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	jsonDir := filepath.Join(samplesDir, "form", "fill")

	for _, tt := range []struct {
		msg        string
		inFile     string
		inFileJSON string
		outFile    string
	}{
		{"TestFillFormEN", "english.pdf", "english.json", "english.pdf"},                    // Core font (Helvetica)
		{"TestFillFormUK", "ukrainian.pdf", "ukrainian.json", "ukrainian.pdf"},              // User font (Roboto-Regular)
		{"TestFillFormRTL", "arabic.pdf", "arabic.json", "arabic.pdf"},                      // User font RTL (Roboto-Regular)
		{"TestFillFormCJK", "chineseSimple.pdf", "chineseSimple.json", "chineseSimple.pdf"}, // User font CJK (UnifontMedium)
		{"TestFillPersonForm", "person.pdf", "person.json", "person.pdf"},                   // Person Form
	} {
		inFile := filepath.Join(inDir, tt.inFile)
		inFileJSON := filepath.Join(jsonDir, tt.inFileJSON)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.FillFormCommand(inFile, inFileJSON, outFile, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestMultiFillFormJSON(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	jsonDir := filepath.Join(samplesDir, "form", "multifill", "json")

	for _, tt := range []struct {
		msg        string
		inFile     string
		inFileJSON string
	}{
		{"TestMultiFillFormJSONEnglish", "english.pdf", "english.json"},
		{"TestMultiFillFormJSONPerson", "person.pdf", "person.json"},
	} {
		inFile := filepath.Join(inDir, tt.inFile)
		inFileJSON := filepath.Join(jsonDir, tt.inFileJSON)

		cmd := cli.MultiFillFormCommand(inFile, inFileJSON, outDir, tt.inFile, false, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestMultiFillFormJSONMerged(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	jsonDir := filepath.Join(samplesDir, "form", "multifill", "json")

	for _, tt := range []struct {
		msg        string
		inFile     string
		inFileJSON string
	}{
		{"TestMultiFillFormJSONEnglish", "english.pdf", "english.json"},
		{"TestMultiFillFormJSONPerson", "person.pdf", "person.json"},
	} {
		inFile := filepath.Join(inDir, tt.inFile)
		inFileJSON := filepath.Join(jsonDir, tt.inFileJSON)

		cmd := cli.MultiFillFormCommand(inFile, inFileJSON, outDir, tt.inFile, true, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestMultiFillFormCSV(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	csvDir := filepath.Join(samplesDir, "form", "multifill", "csv")

	for _, tt := range []struct {
		msg       string
		inFile    string
		inFileCSV string
	}{
		{"TestMultiFillFormCSVEnglish", "english.pdf", "english.csv"},
		{"TestMultiFillFormCSVPerson", "person.pdf", "person.csv"},
	} {

		inFile := filepath.Join(inDir, tt.inFile)
		inFileCSV := filepath.Join(csvDir, tt.inFileCSV)

		cmd := cli.MultiFillFormCommand(inFile, inFileCSV, outDir, tt.inFile, false, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

func TestMultiFillFormCSVMerged(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	csvDir := filepath.Join(samplesDir, "form", "multifill", "csv")

	for _, tt := range []struct {
		msg       string
		inFile    string
		inFileCSV string
	}{
		{"TestMultiFillFormCSVEnglish", "english.pdf", "english.csv"},
		{"TestMultiFillFormCSVPerson", "person.pdf", "person.csv"},
	} {

		inFile := filepath.Join(inDir, tt.inFile)
		inFileCSV := filepath.Join(csvDir, tt.inFileCSV)

		cmd := cli.MultiFillFormCommand(inFile, inFileCSV, outDir, tt.inFile, false, conf)
		if _, err := cli.Process(cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

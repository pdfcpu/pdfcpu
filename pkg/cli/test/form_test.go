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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
)

func createSinglePageDemoForm(t *testing.T, fileName string) string {
	t.Helper()
	jsonFile := strings.TrimSuffix(fileName, filepath.Ext(fileName)) + ".json"
	inFileJSON := filepath.Join(inDir, "json", "form", "demoSinglePage", jsonFile)
	outFile := filepath.Join(t.TempDir(), fileName)
	createPDF(t, "create single page demo form", "", inFileJSON, outFile, conf)
	return outFile
}

/**************************************************************
 * All form related processing is optimized for Adobe Reader! *
 **************************************************************/

// TestListFormFields verifies listing form fields.
func TestListFormFields(t *testing.T) {

	msg := "TestListFormFields"
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")

	cmd := cli.ListFormFieldsCommand([]string{inFile}, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

// TestListFormFieldsJSON verifies listing form fields as export JSON.
func TestListFormFieldsJSON(t *testing.T) {
	msg := "TestListFormFieldsJSON"
	inFile := filepath.Join(samplesDir, "form", "demoSinglePage", "english.pdf")

	cmd := cli.ListFormFieldsJSONCommand([]string{inFile}, conf)
	ss, err := cli.Dispatch(t.Context(), cmd)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
	if len(ss) != 1 {
		t.Fatalf("%s: want 1 JSON output string, got %d\n", msg, len(ss))
	}

	formGroup := form.FormGroup{}
	if err := json.Unmarshal([]byte(ss[0]), &formGroup); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if len(formGroup.Forms) != 1 {
		t.Fatalf("%s: want 1 form, got %d\n", msg, len(formGroup.Forms))
	}
	if formGroup.Header.Source != "english.pdf" {
		t.Fatalf("%s: want source english.pdf, got %s\n", msg, formGroup.Header.Source)
	}
}

// TestListFormFieldsJSONCanFill verifies JSON form list output can be fed into form fill.
func TestListFormFieldsJSONCanFill(t *testing.T) {
	msg := "TestListFormFieldsJSONCanFill"
	inFile := filepath.Join(samplesDir, "form", "demoSinglePage", "english.pdf")
	jsonFile := filepath.Join(outDir, "english-list.json")
	outFile := filepath.Join(outDir, "english-list-filled.pdf")

	cmd := cli.ListFormFieldsJSONCommand([]string{inFile}, conf)
	ss, err := cli.Dispatch(t.Context(), cmd)
	if err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}

	formGroup := form.FormGroup{}
	if err := json.Unmarshal([]byte(ss[0]), &formGroup); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if len(formGroup.Forms) == 0 || len(formGroup.Forms[0].TextFields) == 0 {
		t.Fatalf("%s: missing text fields\n", msg)
	}
	formGroup.Forms[0].TextFields[0].Value = "list json fill"

	bb, err := json.MarshalIndent(formGroup, "", "\t")
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if err := os.WriteFile(jsonFile, bb, 0644); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	cmd = cli.FillFormCommand(inFile, jsonFile, outFile, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

// TestListFormFieldsJSONMultiFile verifies JSON form list output for multiple input files.
func TestListFormFieldsJSONMultiFile(t *testing.T) {
	msg := "TestListFormFieldsJSONMultiFile"
	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	inFiles := []string{
		filepath.Join(inDir, "english.pdf"),
		filepath.Join(inDir, "person.pdf"),
	}

	cmd := cli.ListFormFieldsJSONCommand(inFiles, conf)
	ss, err := cli.Dispatch(t.Context(), cmd)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	formGroup := form.FormGroup{}
	if err := json.Unmarshal([]byte(ss[0]), &formGroup); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	if len(formGroup.Forms) != len(inFiles) {
		t.Fatalf("%s: want %d forms, got %d\n", msg, len(inFiles), len(formGroup.Forms))
	}
}

// TestRemoveFormFields verifies remove form fields.
func TestRemoveFormFields(t *testing.T) {

	msg := "TestRemoveFormFields"
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")
	outFile := filepath.Join(outDir, "removedField.pdf")

	cmd := cli.RemoveFormFieldsCommand(inFile, outFile, []string{"dob1"}, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s %s: %v\n", msg, inFile, err)
	}
}

// TestResetFormFields verifies reset form fields.
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
		inFile := createSinglePageDemoForm(t, tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.ResetFormCommand(inFile, outFile, nil, conf)
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}

}

// TestLockFormFields verifies lock form fields.
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
		inFile := createSinglePageDemoForm(t, tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.LockFormCommand(inFile, outFile, nil, conf)
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestUnlockFormFields verifies unlock form fields.
func TestUnlockFormFields(t *testing.T) {

	for _, tt := range []struct {
		msg        string
		sourceFile string
		inFile     string
		outFile    string
	}{
		{"TestUnlockFormEN", "english.pdf", "english-locked.pdf", "english-unlocked.pdf"},                    // Core font (Helvetica)
		{"TestUnlockFormUK", "ukrainian.pdf", "ukrainian-locked.pdf", "ukrainian-unlocked.pdf"},              // User font (Roboto-Regular)
		{"TestUnlockFormRTL", "arabic.pdf", "arabic-locked.pdf", "arabic-unlocked.pdf"},                      // User font RTL (Roboto-Regular)
		{"TestUnlockFormCJK", "chineseSimple.pdf", "chineseSimple-locked.pdf", "chineseSimple-unlocked.pdf"}, // User font CJK (UnifontMedium)
		{"TestUnlockPersonForm", "person.pdf", "person-locked.pdf", "person-unlocked.pdf"},                   // Person Form
	} {
		sourceFile := createSinglePageDemoForm(t, tt.sourceFile)
		inFile := filepath.Join(t.TempDir(), tt.inFile)
		cmd := cli.LockFormCommand(sourceFile, inFile, nil, conf)
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, sourceFile, err)
		}
		outFile := filepath.Join(outDir, tt.outFile)

		cmd = cli.UnlockFormCommand(inFile, outFile, nil, conf)
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestExportForm verifies export form.
func TestExportForm(t *testing.T) {

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
		inFile := createSinglePageDemoForm(t, tt.inFile)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.ExportFormCommand(inFile, outFile, conf)
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestFillForm verifies fill form.
func TestFillForm(t *testing.T) {

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
		inFile := createSinglePageDemoForm(t, tt.inFile)
		inFileJSON := filepath.Join(jsonDir, tt.inFileJSON)
		outFile := filepath.Join(outDir, tt.outFile)

		cmd := cli.FillFormCommand(inFile, inFileJSON, outFile, conf)
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestMultiFillFormJSON verifies multi fill form JSON.
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
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestMultiFillFormJSONMerged verifies multi fill form JSON merged.
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
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestMultiFillFormJSONMergedStdinStdout verifies multifill supports stdin and stdout together.
func TestMultiFillFormJSONMergedStdinStdout(t *testing.T) {
	inFile := filepath.Join(samplesDir, "form", "demoSinglePage", "english.pdf")
	inFileJSON := filepath.Join(samplesDir, "form", "multifill", "json", "english.json")

	stdin, err := os.Open(inFile)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()

	stdout, err := os.CreateTemp(t.TempDir(), "multifill-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	os.Stdin = stdin
	os.Stdout = stdout
	t.Cleanup(func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	})

	cmd := cli.MultiFillFormCommand("-", inFileJSON, "", "-", true, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("multifill stdin/stdout: %v", err)
	}

	info, err := stdout.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("expected PDF output on stdout")
	}
}

// TestMultiFillFormCSV verifies multi fill form CSV.
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
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

// TestMultiFillFormCSVMerged verifies multi fill form CSV merged.
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
		if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
			t.Fatalf("%s %s: %v\n", tt.msg, inFile, err)
		}
	}
}

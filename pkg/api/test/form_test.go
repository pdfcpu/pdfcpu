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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

/**************************************************************
 * All form related processing is optimized for Adobe Reader! *
 **************************************************************/

func listFormFieldsFile(t *testing.T, inFile string, conf *model.Configuration) ([]string, error) {
	t.Helper()

	msg := "listFormFields"

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS

	f, err := os.Open(inFile)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	defer f.Close()

	ctx, err := api.ReadValidateAndOptimize(f, conf)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	return form.ListFormFields(ctx)
}

func TestListFormFields(t *testing.T) {

	msg := "TestListFormFields"
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")

	ss, err := listFormFieldsFile(t, inFile, conf)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if len(ss) != 27 {
		t.Fatalf("%s: want 27, got %d lines\n", msg, len(ss))
	}
}

func TestRemoveFormFields(t *testing.T) {

	msg := "TestRemoveFormFields"
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")
	outFile := filepath.Join(samplesDir, "form", "remove", "removedField.pdf")

	ss, err := listFormFieldsFile(t, inFile, conf)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	want := len(ss) - 2

	if err := api.RemoveFormFieldsFile(inFile, outFile, []string{"dob1", "firstName1"}, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	ss, err = listFormFieldsFile(t, outFile, conf)
	if err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}
	got := len(ss)

	if got != want {
		t.Fail()
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
		outFile := filepath.Join(samplesDir, "form", "reset", tt.outFile)
		if err := api.ResetFormFieldsFile(inFile, outFile, nil, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
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
		outFile := filepath.Join(samplesDir, "form", "lock", tt.outFile)
		if err := api.LockFormFieldsFile(inFile, outFile, nil, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
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
		outFile := filepath.Join(samplesDir, "form", "lock", tt.outFile)
		if err := api.UnlockFormFieldsFile(inFile, outFile, nil, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

func TestExportForm(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	outDir := filepath.Join(samplesDir, "form", "export")

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
		if err := api.ExportFormFile(inFile, outFile, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

func TestFillForm(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	jsonDir := filepath.Join(samplesDir, "form", "fill")
	outDir := jsonDir

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
		if err := api.FillFormFile(inFile, inFileJSON, outFile, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

func TestMultiFillFormJSON(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	jsonDir := filepath.Join(samplesDir, "form", "multifill", "json")
	outDir := jsonDir

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
		if err := api.MultiFillFormFile(inFile, inFileJSON, outDir, inFile, false, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

func TestMultiFillFormJSONMerged(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	jsonDir := filepath.Join(samplesDir, "form", "multifill", "json")
	outDir := filepath.Join(jsonDir, "merge")

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
		if err := api.MultiFillFormFile(inFile, inFileJSON, outDir, inFile, true, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

func TestMultiFillFormCSV(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	csvDir := filepath.Join(samplesDir, "form", "multifill", "csv")
	outDir := csvDir

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
		if err := api.MultiFillFormFile(inFile, inFileCSV, outDir, inFile, false, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

func TestMultiFillFormCSVMerged(t *testing.T) {

	inDir := filepath.Join(samplesDir, "form", "demoSinglePage")
	csvDir := filepath.Join(samplesDir, "form", "multifill", "csv")
	outDir := filepath.Join(csvDir, "merge")

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
		if err := api.MultiFillFormFile(inFile, inFileCSV, outDir, inFile, true, conf); err != nil {
			t.Fatalf("%s: %v\n", tt.msg, err)
		}
	}
}

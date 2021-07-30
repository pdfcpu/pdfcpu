/*
Copyright 2021 The pdfcpu Authors.

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

func createForm(t *testing.T, msg, inDir, inFile, outDir, outFile string) {

	t.Helper()

	inFile = filepath.Join(inDir, inFile)
	outFile = filepath.Join(outDir, outFile)

	if err := api.CreateFormFile(inFile, outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}

func TestCreateForm(t *testing.T) {

	api.LoadConfiguration()

	// Install test user fonts from pkg/testdata/fonts.
	if err := api.InstallFonts(userFonts(t, filepath.Join("..", "..", "testdata", "fonts"))); err != nil {
		t.Fatalf("%s: %v\n", "TestCreateForm", err)
	}

	inDir := filepath.Join("..", "..", "testdata", "forms")
	outDir := filepath.Join("..", "..", "samples", "forms")

	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
	}{
		// {"TestTextfield", "textfield.json", "textfield.pdf"},
		// {"TestTextarea", "textarea.json", "textarea.pdf"},
		// {"TestCheckbox", "checkbox.json", "checkbox.pdf"},
		// {"TestRadiobuttonsHor", "radiobuttonsHor.json", "radiobuttonsHor.pdf"},
		// {"TestRadiobuttonsVertLeft", "radiobuttonsVertLeft.json", "radiobuttonsVertLeft.pdf"},
		// {"TestRadiobuttonsVertRight", "radiobuttonsVertRight.json", "radiobuttonsVertRight.pdf"},
		{"TestListbox", "listbox.json", "listbox.pdf"},
		{"TestCombobox", "combobox.json", "combobox.pdf"},
	} {
		createForm(t, tt.msg, inDir, tt.inFile, outDir, tt.outFile)
	}

	// if err := api.ExtractForm(outFile, "out.json", nil); err != nil {
	// 	t.Fatalf("%s: %v\n", msg, err)
	// }

	// in.json should == out.sign
}

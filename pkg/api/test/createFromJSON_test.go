/*
Copyright 2019 The pdf Authors.

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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func createText(t *testing.T, msg, inDir, jsonFile, outDir, outFile string, conf *pdfcpu.Configuration) {

	t.Helper()

	jsonFile = filepath.Join(inDir, jsonFile)
	outFile = filepath.Join(outDir, outFile)

	// jsonFile outFile ""					does not find outFile, write new Context to outFile
	// jsonFile outFile outFile				read from outFile, write to outFile
	// jsonFile "" outFile					write new Context to outFile

	if err := api.CreateFromJSONFile(jsonFile, "", outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}

func TestCreateViaJson(t *testing.T) {

	conf := api.LoadConfiguration()

	// Install test user fonts from pkg/testdata/fonts.
	if err := api.InstallFonts(userFonts(t, filepath.Join("..", "..", "testdata", "fonts"))); err != nil {
		t.Fatalf("%s: %v\n", "TestCreateForm", err)
	}

	inDir := filepath.Join("..", "..", "testdata", "json")
	outDir := filepath.Join("..", "..", "samples", "create")

	for _, tt := range []struct {
		msg     string
		inFile  string
		outFile string
	}{

		// Render basic page content.

		{"TestFonts", "fonts.json", "fonts.pdf"},
		{"TestUserFonts", "userfonts.json", "userfonts.pdf"},

		{"TestTextAnchored", "textAnchored.json", "textAnchored.pdf"},
		{"TestTextBordersAndPaddings", "textBordersAndPaddings.json", "textBordersAndPaddings.pdf"},

		{"TestImages", "images.json", "images.pdf"},
		{"TestImagesOptimized", "imagesOptimized.json", "imagesOptimized.pdf"},
		{"TestImagesDirsFiles", "imagesDirsFiles.json", "imagesDirsFiles.pdf"},

		{"TestBoxesAndColors", "boxesAndColors.json", "boxesAndColors.pdf"},
		{"TestBoxesAndMargin", "boxesAndMargin.json", "boxesAndMargin.pdf"},
		{"TestBoxesAndRotation", "boxesAndRotation.json", "boxesAndRotation.pdf"},

		{"TestTables", "tables.json", "tables.pdf"},

		{"TestRegions", "regions.json", "regions.pdf"},
		{"TestRegionsMarginBorderPadding", "regionsMarginBorderPadding.json", "regionsMarginBorderPadding.pdf"},

		// Render page content using form input components.

		{"TestTextfield", "textfield.json", "textfield.pdf"},
		{"TestTextarea", "textarea.json", "textarea.pdf"},
		{"TestCheckbox", "checkbox.json", "checkbox.pdf"},

		{"TestRadiobuttonsHor", "radiobuttonsHor.json", "radiobuttonsHor.pdf"},
		{"TestRadiobuttonsVertLeft", "radiobuttonsVertLeft.json", "radiobuttonsVertLeft.pdf"},
		{"TestRadiobuttonsVertRight", "radiobuttonsVertRight.json", "radiobuttonsVertRight.pdf"},
	} {
		createText(t, tt.msg, inDir, tt.inFile, outDir, tt.outFile, conf)
	}

}

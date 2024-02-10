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

	"github.com/pdfcpu/pdfcpu/pkg/api"
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

	if err := api.CreateFile(inFile, inFileJSON, outFile, conf); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}

func TestCreateContentPrimitivesViaJson(t *testing.T) {

	t.Helper()
	inDir := filepath.Join(inDir, "json", "create")
	outDir := filepath.Join(samplesDir, "create", "primitives")

	for _, tt := range []struct {
		msg        string
		inFileJSON string
		outFile    string
	}{
		// Render page content samples.

		// Font
		{"TestFonts", "fonts.json", "fonts.pdf"},

		// Text
		{"TestTextAnchored", "textAnchored.json", "textAnchored.pdf"},
		{"TestTextBordersAndPaddings", "textBordersAndPaddings.json", "textBordersAndPaddings.pdf"},
		{"TestTextAlignment", "textAndAlignment.json", "textAndAlignment.pdf"},

		// Image
		{"TestImages", "images.json", "images.pdf"},
		{"TestImagesOptimized", "imagesOptimized.json", "imagesOptimized.pdf"},
		{"TestImagesDirsFiles", "imagesDirsFiles.json", "imagesDirsFiles.pdf"},

		// Box
		{"TestBoxesAndColors", "boxesAndColors.json", "boxesAndColors.pdf"},
		{"TestBoxesAndMargin", "boxesAndMargin.json", "boxesAndMargin.pdf"},
		{"TestBoxesAndRotation", "boxesAndRotation.json", "boxesAndRotation.pdf"},

		// Table
		{"TestTable", "table.json", "table.pdf"},
		{"TestTableRTL", "tableRTL.json", "tableRTL.pdf"},
		{"TestTableCJK", "tableCJK.json", "tableCJK.pdf"},

		// Content Region
		{"TestRegions", "regions.json", "regions.pdf"},
		{"TestRegionsMarginBorderPadding", "regionsMargBordPadd.json", "regionsMarginBorderPadding.pdf"},
	} {
		inFileJSON := filepath.Join(inDir, tt.inFileJSON)
		outFile := filepath.Join(outDir, tt.outFile)
		createPDF(t, tt.msg, "", inFileJSON, outFile, conf)
	}

}

func TestCreateFormPrimitivesViaJson(t *testing.T) {

	inDirForm := filepath.Join(inDir, "json", "form")
	outDirForm := filepath.Join(samplesDir, "form", "primitives")

	for _, tt := range []struct {
		msg        string
		inFileJSON string
		outFile    string
	}{
		// Render form field samples.

		// Textfield
		{"TestTextfield", "textfield.json", "textfield.pdf"},
		{"TestTextfieldGroup", "textfieldGroup.json", "textfieldGroup.pdf"},
		{"TestTextfieldGroupSingle", "textfieldGroupSingle.json", "textfieldGroupSingle.pdf"},

		// Textarea
		{"TestTextarea", "textarea.json", "textarea.pdf"},
		{"TestTextareaGroup", "textareaGroup.json", "textareaGroup.pdf"},

		// Datefield
		{"TestDatefield", "datefield.json", "datefield.pdf"},
		{"TestDatefieldGroup", "datefieldGroup.json", "datefieldGroup.pdf"},

		// Checkbox
		{"TestCheckbox", "checkbox.json", "checkbox.pdf"},
		{"TestCheckboxGroup", "checkboxGroup.json", "checkboxGroup.pdf"},

		// Radio button group
		{"TestRadiobuttonsHor", "radiobuttonsHor.json", "radiobuttonsHor.pdf"},
		{"TestRadiobuttonsHorGroup", "radiobuttonsHorGroup.json", "radiobuttonsHorGroup.pdf"},
		{"TestRadiobuttonsVertLeft", "radiobuttonsVertL.json", "radiobuttonsVertL.pdf"},
		{"TestRadiobuttonsVertLeftGroup", "radiobuttonsVertLGroup.json", "radiobuttonsVertLGroup.pdf"},
		{"TestRadiobuttonsVertRight", "radiobuttonsVertR.json", "radiobuttonsVertR.pdf"},
		{"TestRadiobuttonsVertRightGroup", "radiobuttonsVertRGroup.json", "radiobuttonsVertRGroup.pdf"},

		// Combobox
		{"TestCombobox", "combobox.json", "combobox.pdf"},
		{"TestComboboxGroup", "comboboxGroup.json", "comboboxGroup.pdf"},

		// Listbox
		{"TestListbox", "listbox.json", "listbox.pdf"},
		{"TestListboxGroup", "listboxGroup.json", "listboxGroup.pdf"},
	} {
		inFileJSON := filepath.Join(inDirForm, tt.inFileJSON)
		outFile := filepath.Join(outDirForm, tt.outFile)
		createPDF(t, tt.msg, "", inFileJSON, outFile, conf)
	}

}

func TestCreateSinglePageDemoFormsViaJson(t *testing.T) {

	// Render single page demo forms for export, reset, lock, unlock and fill tests.

	inDirFormDemo := filepath.Join(inDir, "json", "form", "demoSinglePage")
	outDirFormDemo := filepath.Join(samplesDir, "form", "demoSinglePage")

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

}

func TestCreateDemoFormsViaJson(t *testing.T) {

	inDirFormDemo := filepath.Join(inDir, "json", "form", "demo")
	outDirFormDemo := filepath.Join(samplesDir, "form", "demo")

	for _, tt := range []struct {
		msg        string
		inFileJSON string
		outFile    string
	}{
		// Render demo forms.

		// For corrections please open an issue on Github.

		// Core font (Helvetica)
		{"TestFormDemoDM", "danish.json", "danish.pdf"},
		{"TestFormDemoNL", "dutch.json", "dutch.pdf"},
		{"TestFormDemoEN", "english.json", "english.pdf"},
		{"TestFormDemoFI", "finnish.json", "finnish.pdf"},
		{"TestFormDemoFR", "french.json", "french.pdf"},
		{"TestFormDemoDE", "german.json", "german.pdf"},
		{"TestFormDemoHU", "hungarian.json", "hungarian.pdf"},
		{"TestFormDemoIN", "indonesian.json", "indonesian.pdf"},
		{"TestFormDemoIC", "icelandic.json", "icelandic.pdf"},
		{"TestFormDemoIT", "italian.json", "italian.pdf"},
		{"TestFormDemoNO", "norwegian.json", "norwegian.pdf"},
		{"TestFormDemoPT", "portuguese.json", "portuguese.pdf"},
		{"TestFormDemoSK", "slovak.json", "slovak.pdf"},
		{"TestFormDemoSL", "slovenian.json", "slovenian.pdf"},
		{"TestFormDemoES", "spanish.json", "spanish.pdf"},
		{"TestFormDemoSWA", "swahili.json", "swahili.pdf"},
		{"TestFormDemoSWE", "swedish.json", "swedish.pdf"},

		// User font (Roboto-Regular)
		{"TestFormDemoBR", "belarusian.json", "belarusian.pdf"},
		{"TestFormDemoBG", "bulgarian.json", "bulgarian.pdf"},
		{"TestFormDemoCR", "croatian.json", "croatian.pdf"},
		{"TestFormDemoCZ", "czech.json", "czech.pdf"},
		{"TestFormDemoGR", "greek.json", "greek.pdf"},
		{"TestFormDemoKU", "kurdish.json", "kurdish.pdf"},
		{"TestFormDemoPO", "polish.json", "polish.pdf"},
		{"TestFormDemoRO", "romanian.json", "romanian.pdf"},
		{"TestFormDemoRU", "russian.json", "russian.pdf"},
		{"TestFormDemoTK", "turkish.json", "turkish.pdf"},
		{"TestFormDemoUK", "ukrainian.json", "ukrainian.pdf"},
		{"TestFormDemoVI", "vietnamese.json", "vietnamese.pdf"},

		// User font (UnifontMedium)
		{"TestFormDemoAR", "arabic.json", "arabic.pdf"},
		{"TestFormDemoARM", "armenian.json", "armenian.pdf"},
		{"TestFormDemoAZ", "azerbaijani.json", "azerbaijani.pdf"},
		{"TestFormDemoBA", "bangla.json", "bangla.pdf"},
		{"TestFormDemoSC", "chineseSimple.json", "chineseSimple.pdf"},
		{"TestFormDemoTC", "chineseTrad.json", "chineseTraditional.pdf"},
		{"TestFormDemoHE", "hebrew.json", "hebrew.pdf"},
		{"TestFormDemoHI", "hindi.json", "hindi.pdf"},
		{"TestFormDemoJP", "japanese.json", "japanese.pdf"},
		{"TestFormDemoKR", "korean.json", "korean.pdf"},
		{"TestFormDemoMA", "marathi.json", "marathi.pdf"},
		{"TestFormDemoPE", "persian.json", "persian.pdf"},
		{"TestFormDemoUR", "thai.json", "thai.pdf"},
		{"TestFormDemoUR", "urdu.json", "urdu.pdf"},
	} {
		inFileJSON := filepath.Join(inDirFormDemo, tt.inFileJSON)
		outFile := filepath.Join(outDirFormDemo, tt.outFile)
		createPDF(t, tt.msg, "", inFileJSON, outFile, conf)
	}

}

func TestCreateAndUpdatePageViaJson(t *testing.T) {

	// CREATE PDF, UPDATE/ADD PAGE
	// 1. Create PDF page
	// 2. Add textbox and reuse corefont/userfont/cjkfont
	// 	 a) from same page
	// 	 b) on different page

	jsonDir := filepath.Join(inDir, "json", "create", "flow")
	outDir := filepath.Join(samplesDir, "create", "flow")

	// Create PDF in outDir.
	inFileJSON1 := filepath.Join(jsonDir, "createPage.json")
	file := filepath.Join(outDir, "createAndUpdatePage.pdf")
	createPDF(t, "pass1", "", inFileJSON1, file, conf)

	// Update PDF in outDir: reuse fonts from (same) page 1
	inFileJSON2 := filepath.Join(jsonDir, "updatePage1.json")
	createPDF(t, "pass2", file, inFileJSON2, file, conf)

	// Update PDF in outDir: reuse fonts from (different) page 1
	inFileJSON3 := filepath.Join(jsonDir, "updatePage2.json")
	createPDF(t, "pass2", file, inFileJSON3, file, conf)
}

func TestReadAndUpdatePageViaJson(t *testing.T) {

	// READ PDF, UPDATE/ADD PAGE
	// 1. Read any PDF
	// 2. Add textbox and reuse corefont/userfont/cjkfont
	// 	 a) from same page
	// 	 b) on different page

	jsonDir := filepath.Join(inDir, "json", "create", "flow")
	outDir := filepath.Join(samplesDir, "create", "flow")

	// Update PDF in outDir.
	inFile := filepath.Join(inDir, "Walden.pdf")
	inFileJSON1 := filepath.Join(jsonDir, "updateAnyPage1.json")
	outFile := filepath.Join(outDir, "readAndUpdatePage.pdf")
	createPDF(t, "pass", inFile, inFileJSON1, outFile, conf)

	// Update PDF in outDir: reuse fonts from (different) page 1 and create new page
	inFileJSON2 := filepath.Join(jsonDir, "updateAnyPage2.json")
	createPDF(t, "pass2", outFile, inFileJSON2, outFile, conf)
}

func TestCreateFormAndUpdatePageViaJson(t *testing.T) {

	// CREATE FORM, UPDATE/ADD PAGE
	// 1. Create PDF form
	// 2. Add content

	jsonDir := filepath.Join(inDir, "json", "form")
	outDir := filepath.Join(samplesDir, "form", "flow")

	// Create PDF form in outDir and add content using corefont.
	inFileJSON1 := filepath.Join(jsonDir, "demo", "english.json")
	file := filepath.Join(outDir, "createFormAndUpdatePageCoreFont.pdf")
	createPDF(t, "pass1", "", inFileJSON1, file, conf)
	// Update PDF form in outDir reusing page font.
	inFileJSON2 := filepath.Join(jsonDir, "flow", "updatePageCoreFont.json")
	createPDF(t, "pass2", file, inFileJSON2, file, conf)

	// Create PDF form in outDir and add content using userfont.
	inFileJSON1 = filepath.Join(jsonDir, "demo", "ukrainian.json")
	file = filepath.Join(outDir, "createFormAndUpdatePageUserFont.pdf")
	createPDF(t, "pass1", "", inFileJSON1, file, conf)
	// Update PDF form in outDir reusing page font.
	inFileJSON2 = filepath.Join(jsonDir, "flow", "updatePageUserFont.json")
	createPDF(t, "pass2", file, inFileJSON2, file, conf)

	// Create PDF form in outDir and add content using CJK userfont.
	inFileJSON1 = filepath.Join(jsonDir, "demo", "chineseSimple.json")
	file = filepath.Join(outDir, "createFormAndUpdatePageCJKUserFont.pdf")
	createPDF(t, "pass1", "", inFileJSON1, file, conf)
	// Update PDF form in outDir reusing page font.
	inFileJSON2 = filepath.Join(jsonDir, "flow", "updatePageCJKUserFont.json")
	createPDF(t, "pass2", file, inFileJSON2, file, conf)
}

func TestReadFormAndUpdateFormViaJson(t *testing.T) {

	// READ FORM, UPDATE FORM
	// 1. Read PDF form
	// 2. Add fields

	jsonDir := filepath.Join(inDir, "json", "form", "flow")
	outDir := filepath.Join(samplesDir, "form", "flow")

	// Read demo form and update with corefont in outDir.
	inFile := filepath.Join(samplesDir, "form", "demo", "english.pdf")
	inFileJSON := filepath.Join(jsonDir, "updateFormCoreFont.json")
	outFile := filepath.Join(outDir, "readFormAndUpdateFormCoreFont.pdf")
	createPDF(t, "pass1", inFile, inFileJSON, outFile, conf)

	// Read demo form and update with userfont in outDir.
	inFile = filepath.Join(samplesDir, "form", "demo", "ukrainian.pdf")
	inFileJSON = filepath.Join(jsonDir, "updateFormUserFont.json")
	outFile = filepath.Join(outDir, "readFormAndUpdateFormUserFont.pdf")
	createPDF(t, "pass1", inFile, inFileJSON, outFile, conf)

	// Read demo form and update with CJK userfont in outDir.
	inFile = filepath.Join(samplesDir, "form", "demo", "chineseSimple.pdf")
	inFileJSON = filepath.Join(jsonDir, "updateFormCJK.json")
	outFile = filepath.Join(outDir, "readFormAndUpdateFormCJK.pdf")
	createPDF(t, "pass1", inFile, inFileJSON, outFile, conf)
}

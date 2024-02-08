/*
Copyright 2018 The pdfcpu Authors.

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

package api

import (
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func ExampleValidateFile() {

	// Use the default configuration to validate in.pdf.
	ValidateFile("in.pdf", nil)
}

func ExampleOptimizeFile() error {

	conf, err := model.NewDefaultConfiguration()
	if err != nil {
		return err
	}

	// Set passwords for encrypted files.
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"

	// Configure end of line sequence for writing.
	conf.Eol = types.EolLF

	// Create an optimized version of in.pdf and write it to out.pdf.
	OptimizeFile("in.pdf", "out.pdf", conf)

	// Create an optimized version of inFile.
	// If you want to modify the original file, pass an empty string for outFile.
	// Use nil for a default configuration.
	OptimizeFile("in.pdf", "", nil)
	return nil
}

func ExampleTrimFile() {

	// Create a trimmed version of in.pdf containing odd page numbers only.
	TrimFile("in.pdf", "outFile", []string{"odd"}, nil)

	// Create a trimmed version of in.pdf containing the first two pages only.
	// If you want to modify the original file, pass an empty string for outFile.
	TrimFile("in.pdf", "", []string{"1-2"}, nil)
}

func ExampleSplitFile() {

	// Create single page PDFs for in.pdf in outDir using the default configuration.
	SplitFile("in.pdf", "outDir", 1, nil)

	// Create dual page PDFs for in.pdf in outDir using the default configuration.
	SplitFile("in.pdf", "outDir", 2, nil)

	// Create a sequence of PDFs representing bookmark secions.
	SplitFile("in.pdf", "outDir", 0, nil)
}

func ExampleRotateFile() {

	// Rotate all pages of in.pdf, clockwise by 90 degrees and write the result to out.pdf.
	RotateFile("in.pdf", "out.pdf", 90, nil, nil)

	// Rotate the first page of in.pdf by 180 degrees.
	// If you want to modify the original file, pass an empty string as outFile.
	RotateFile("in.pdf", "", 180, []string{"1"}, nil)
}

func ExampleMergeCreateFile() {

	// Merge inFiles by concatenation in the order specified and write the result to out.pdf.
	// out.pdf will be overwritten.
	inFiles := []string{"in1.pdf", "in2.pdf"}
	MergeCreateFile(inFiles, "out.pdf", false, nil)
}

func ExampleMergeAppendFile() {

	// Merge inFiles by concatenation in the order specified and write the result to out.pdf.
	// If out.pdf already exists it will be preserved and serves as the beginning of the merge result.
	inFiles := []string{"in1.pdf", "in2.pdf"}
	MergeAppendFile(inFiles, "out.pdf", false, nil)
}

func ExampleInsertPagesFile() {

	// Insert a blank page into in.pdf before page #3.
	InsertPagesFile("in.pdf", "", []string{"3"}, true, nil)

	// Insert a blank page into in.pdf after every page.
	InsertPagesFile("in.pdf", "", nil, false, nil)
}

func ExampleRemovePagesFile() {

	// Remove pages 2 and 8 of in.pdf.
	RemovePagesFile("in.pdf", "", []string{"2", "8"}, nil)

	// Remove first 2 pages of in.pdf.
	RemovePagesFile("in.pdf", "", []string{"-2"}, nil)

	// Remove all pages >= 10 of in.pdf.
	RemovePagesFile("in.pdf", "", []string{"10-"}, nil)
}

func ExampleAddWatermarksFile() {

	// Unique abbreviations are accepted for all watermark descriptor parameters.
	// eg. sc = scalefactor or rot = rotation

	// Add a "Demo" watermark to all pages of in.pdf along the diagonal running from lower left to upper right.
	onTop := false
	update := false
	wm, _ := TextWatermark("Demo", "", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Stamp all odd pages of in.pdf in red "Confidential" in 48 point Courier
	// using a rotation angle of 45 degrees and an absolute scalefactor of 1.0.
	onTop = true
	wm, _ = TextWatermark("Confidential", "font:Courier, points:48, col: 1 0 0, rot:45, scale:1 abs, ", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", []string{"odd"}, wm, nil)

	// Add image stamps to in.pdf using absolute scaling and a negative rotation of 90 degrees.
	wm, _ = ImageWatermark("image.png", "scalefactor:.5 a, rot:-90", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Add a PDF stamp to all pages of in.pdf using the 2nd page of stamp.pdf, use absolute scaling of 0.5
	// and rotate along the 2nd diagonal running from upper left to lower right corner.
	wm, _ = PDFWatermark("stamp.pdf:2", "scale:.5 abs, diagonal:2", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)
}

func ExampleRemoveWatermarksFile() {

	// Add a "Demo" stamp to all pages of in.pdf along the diagonal running from lower left to upper right.
	onTop := true
	update := false
	wm, _ := TextWatermark("Demo", "", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Update stamp for correction:
	update = true
	wm, _ = TextWatermark("Confidential", "", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Add another watermark on top of page 1
	wm, _ = TextWatermark("Footer stamp", "c:.5 1 1, pos:bc", onTop, update, types.POINTS)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Remove watermark on page 1
	RemoveWatermarksFile("in.pdf", "", []string{"1"}, nil)

	// Remove all watermarks
	RemoveWatermarksFile("in.pdf", "", nil, nil)
}

func ExampleImportImagesFile() {

	// Convert an image into a single page of out.pdf which will be created if necessary.
	// The page dimensions will match the image dimensions.
	// If out.pdf already exists, append a new page.
	// Use the default import configuration.
	ImportImagesFile([]string{"image.png"}, "out.pdf", nil, nil)

	// Import images by creating an A3 page for each image.
	// Images are page centered with 1.0 relative scaling.
	// Import an image as a new page of the existing out.pdf.
	imp, _ := Import("form:A3, pos:c, s:1.0", types.POINTS)
	ImportImagesFile([]string{"a1.png", "a2.jpg", "a3.tiff"}, "out.pdf", imp, nil)
}

func ExampleNUpFile() {

	// 4-Up in.pdf and write result to out.pdf.
	nup, _ := PDFNUpConfig(4, "", nil)
	inFiles := []string{"in.pdf"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)

	// 9-Up a sequence of images using format Tabloid w/o borders and no margins.
	nup, _ = ImageNUpConfig(9, "f:Tabloid, b:off, m:0", nil)
	inFiles = []string{"in1.png", "in2.jpg", "in3.tiff"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)

	// TestGridFromPDF
	nup, _ = PDFGridConfig(1, 3, "f:LegalL", nil)
	inFiles = []string{"in.pdf"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)

	// TestGridFromImages
	nup, _ = ImageGridConfig(4, 2, "d:500 500, m:20, b:off", nil)
	inFiles = []string{"in1.png", "in2.jpg", "in3.tiff"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)
}

func ExampleSetPermissionsFile() error {

	// Setting all permissions for the AES-256 encrypted in.pdf.
	conf, err := model.NewAESConfiguration("upw", "opw", 256)
	if err != nil {
		return err
	}
	conf.Permissions = model.PermissionsAll
	SetPermissionsFile("in.pdf", "", conf)

	// Restricting permissions for the AES-256 encrypted in.pdf.
	conf, err = model.NewAESConfiguration("upw", "opw", 256)
	if err != nil {
		return err
	}
	conf.Permissions = model.PermissionsNone
	SetPermissionsFile("in.pdf", "", conf)
	return nil
}

func ExampleEncryptFile() error {

	// Encrypting a file using AES-256.
	conf, err := model.NewAESConfiguration("upw", "opw", 256)
	if err != nil {
		return err
	}
	EncryptFile("in.pdf", "", conf)
	return nil
}

func ExampleDecryptFile() error {

	// Decrypting an AES-256 encrypted file.
	conf, err := model.NewAESConfiguration("upw", "opw", 256)
	if err != nil {
		return err
	}
	DecryptFile("in.pdf", "", conf)
	return nil
}

func ExampleChangeUserPasswordFile() error {

	// Changing the user password for an AES-256 encrypted file.
	conf, err := model.NewAESConfiguration("upw", "opw", 256)
	if err != nil {
		return err
	}
	ChangeUserPasswordFile("in.pdf", "", "upw", "upwNew", conf)
	return nil
}

func ExampleChangeOwnerPasswordFile() error {

	// Changing the owner password for an AES-256 encrypted file.
	conf, err := model.NewAESConfiguration("upw", "opw", 256)
	if err != nil {
		return err
	}
	ChangeOwnerPasswordFile("in.pdf", "", "opw", "opwNew", conf)
	return nil
}

func ExampleAddAttachmentsFile() {

	// Attach 3 files to in.pdf.
	AddAttachmentsFile("in.pdf", "", []string{"img.jpg", "attach.pdf", "test.zip"}, false, nil)
}

func ExampleRemoveAttachmentsFile() {

	// Remove 1 attachment from in.pdf.
	RemoveAttachmentsFile("in.pdf", "", []string{"img.jpg"}, nil)

	// Remove all attachments from in.pdf
	RemoveAttachmentsFile("in.pdf", "", nil, nil)
}

func ExampleExtractAttachmentsFile() {

	// Extract 1 attachment from in.pdf into outDir.
	ExtractAttachmentsFile("in.pdf", "outDir", []string{"img.jpg"}, nil)

	// Extract all attachments from in.pdf into outDir
	ExtractAttachmentsFile("in.pdf", "outDir", nil, nil)
}

func ExampleExtractImagesFile() {

	// Extract embedded images from in.pdf into outDir.
	ExtractImagesFile("in.pdf", "outDir", nil, nil)
}

func ExampleExtractFontsFile() {

	// Extract embedded fonts for pages 1-3 from in.pdf into outDir.
	ExtractFontsFile("in.pdf", "outDir", []string{"1-3"}, nil)
}

func ExampleExtractContentFile() {

	// Extract content for all pages in PDF syntax from in.pdf into outDir.
	ExtractContentFile("in.pdf", "outDir", nil, nil)
}

func ExampleExtractPagesFile() {

	// Extract all even numbered pages from in.pdf into outDir.
	ExtractPagesFile("in.pdf", "outDir", []string{"even"}, nil)
}

func ExampleExtractMetadataFile() {

	// Extract all metadata from in.pdf into outDir.
	ExtractMetadataFile("in.pdf", "outDir", nil)
}

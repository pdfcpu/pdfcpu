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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func ExampleValidateFile() {

	// Use the default configuration to validate in.pdf.
	ValidateFile("in.pdf", nil)
}

func ExampleOptimizeFile() {

	conf := pdfcpu.NewDefaultConfiguration()

	// Set passwords for encrypted files.
	conf.UserPW = "upw"
	conf.OwnerPW = "opw"

	// Configure end of line sequence for writing.
	conf.Eol = pdfcpu.EolLF

	// Create an optimized version of in.pdf and write it to out.pdf.
	OptimizeFile("in.pdf", "out.pdf", conf)

	// Create an optimized version of inFile.
	// If you want to modify the original file, pass an empty string for outFile.
	// Use nil for a default configuration.
	OptimizeFile("in.pdf", "", nil)
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
	MergeCreateFile(inFiles, "out.pdf", nil)
}

func ExampleMergeAppendFile() {

	// Merge inFiles by concatenation in the order specified and write the result to out.pdf.
	// If out.pdf already exists it will be preserved and serves as the beginning of the merge result.
	inFiles := []string{"in1.pdf", "in2.pdf"}
	MergeAppendFile(inFiles, "out.pdf", nil)
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
	wm, _ := pdfcpu.ParseTextWatermarkDetails("Demo", "", onTop)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Stamp all odd pages of in.pdf in red "Confidential" in 48 point Courier
	// using a rotation angle of 45 degrees and an absolute scalefactor of 1.0.
	onTop = true
	wm, _ = pdfcpu.ParseTextWatermarkDetails("Confidential", "font:Courier, points:48, col: 1 0 0, rot:45, sc:1 abs, ", onTop)
	AddWatermarksFile("in.pdf", "", []string{"odd"}, wm, nil)

	// Add image stamps to in.pdf using absolute scaling and a negative rotation of 90 degrees.
	onTop = true
	wm, _ = pdfcpu.ParseImageWatermarkDetails("image.png", "scalefactor:.5 a, rot:-90", onTop)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Add a PDF stamp to all pages of in.pdf using the 2nd page of stamp.pdf, use absolute scaling of 0.5
	// and rotate along the 2nd diagonal running from upper left to lower right corner.
	onTop = true
	wm, _ = pdfcpu.ParsePDFWatermarkDetails("stamp.pdf:2", "sc:.5 abs, diagonal:2", onTop)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)
}

func ExampleRemoveWatermarksFile() {

	// Add a "Demo" stamp to all pages of in.pdf along the diagonal running from lower left to upper right.
	onTop := true
	wm, _ := pdfcpu.ParseTextWatermarkDetails("Demo", "", onTop)
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Update stamp for correction:
	wm, _ = pdfcpu.ParseTextWatermarkDetails("Confidential", "", onTop)
	wm.Update = true
	AddWatermarksFile("in.pdf", "", nil, wm, nil)

	// Add another watermark on top of page 1
	wm, _ = pdfcpu.ParseTextWatermarkDetails("Footer stamp", "c:.5 1 1, pos:bc", onTop)
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
	imp, _ := pdf.ParseImportDetails("form:A3, pos:c, s:1.0")
	ImportImagesFile([]string{"a1.png", "a2.jpg", "a3.tiff"}, "out.pdf", imp, nil)
}

func ExampleNUpFile() {

	// 4-Up in.pdf and write result to out.pdf.
	nup, _ := pdf.PDFNUpConfig(4, "")
	inFiles := []string{"in.pdf"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)

	// 9-Up a sequence of images using format Tabloid w/o borders and no margins.
	nup, _ = pdf.ImageNUpConfig(9, "f:Tabloid, b:off, m:0")
	inFiles = []string{"in1.png", "in2.jpg", "in3.tiff"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)

	// TestGridFromPDF
	nup, _ = pdf.PDFGridConfig(1, 3, "f:LegalL")
	inFiles = []string{"in.pdf"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)

	// TestGridFromImages
	nup, _ = pdf.ImageGridConfig(4, 2, "d:500 500, m:20, b:off")
	inFiles = []string{"in1.png", "in2.jpg", "in3.tiff"}
	NUpFile(inFiles, "out.pdf", nil, nup, nil)
}

func ExampleListPermissionsFile() {

	// Output the current permissions of in.pdf.
	list, _ := ListPermissionsFile("in.pdf", nil)
	for _, s := range list {
		fmt.Println(s)
	}
}

func ExampleSetPermissionsFile() {

	// Setting all permissions for the AES-256 encrypted in.pdf.
	conf := pdf.NewAESConfiguration("upw", "opw", 256)
	conf.Permissions = pdfcpu.PermissionsAll
	SetPermissionsFile("in.pdf", "", conf)

	// Restricting permissions for the AES-256 encrypted in.pdf.
	conf = pdf.NewAESConfiguration("upw", "opw", 256)
	conf.Permissions = pdfcpu.PermissionsNone
	SetPermissionsFile("in.pdf", "", conf)
}

func ExampleEncryptFile() {

	// Encrypting a file using AES-256.
	conf := pdf.NewAESConfiguration("upw", "opw", 256)
	EncryptFile("in.pdf", "", conf)
}

func ExampleDecryptFile() {

	// Decrypting an AES-256 encrypted file.
	conf := pdf.NewAESConfiguration("upw", "opw", 256)
	DecryptFile("in.pdf", "", conf)
}

func ExampleChangeUserPasswordFile() {

	// Changing the user password for an AES-256 encrypted file.
	conf := pdf.NewAESConfiguration("upw", "opw", 256)
	ChangeUserPasswordFile("in.pdf", "", "upw", "upwNew", conf)
}

func ExampleChangeOwnerPasswordFile() {

	// Changing the owner password for an AES-256 encrypted file.
	conf := pdf.NewAESConfiguration("upw", "opw", 256)
	ChangeOwnerPasswordFile("in.pdf", "", "opw", "opwNew", conf)
}

func ExampleListAttachmentsFile() {

	// Output a list of attachments of in.pdf.
	list, _ := ListAttachmentsFile("in.pdf", nil)
	for _, s := range list {
		fmt.Println(s)
	}
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
	ExtractMetadataFile("in.pdf", "outDir", nil, nil)
}

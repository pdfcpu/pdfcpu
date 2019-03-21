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

	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

func ExampleProcess() {
	// Please refer to the following examples.
}

func exampleProcessValidate() {

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Set relaxed validation mode.
	config.ValidationMode = pdfcpu.ValidationRelaxed

	_, err := Process(ValidateCommand("in.pdf", config))
	if err != nil {
		return
	}

}

func exampleProcessOptimize() {

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Generate optional stats.
	config.StatsFileName = "stats.csv"

	// Configure end of line sequence for writing.
	config.Eol = pdfcpu.EolLF

	_, err := Process(OptimizeCommand("in.pdf", "out.pdf", config))
	if err != nil {
		return
	}

}

func exampleProcessMerge() {

	// Concatenate this sequence of PDF files:
	filenamesIn := []string{"in1.pdf", "in2.pdf", "in3.pdf"}

	_, err := Process(MergeCommand(filenamesIn, "out.pdf", pdfcpu.NewDefaultConfiguration()))
	if err != nil {
		return
	}

}

func exampleProcessSplit() {

	// Split into single-page PDFs.

	config := pdfcpu.NewDefaultConfiguration()

	_, err := Process(SplitCommand("in.pdf", "outDir", 1, config))
	if err != nil {
		return
	}

}

func exampleProcessSplitWithSpan() {

	// Split into PDFs using a split span of 2
	// Each generated file has 2 pages.

	config := pdfcpu.NewDefaultConfiguration()

	_, err := Process(SplitCommand("in.pdf", "outDir", 2, config))
	if err != nil {
		return
	}

}

func exampleProcessTrim() {

	// Trim to first three pages.
	selectedPages := []string{"-3"}

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	_, err := Process(TrimCommand("in.pdf", "out.pdf", selectedPages, config))
	if err != nil {
		return
	}

}

func exampleProcessExtractPages() {

	// Extract single-page PDFs for pages 3, 4 and 5.
	selectedPages := []string{"3..5"}

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	_, err := Process(ExtractPagesCommand("in.pdf", "dirOut", selectedPages, config))
	if err != nil {
		return
	}

}

func exampleProcessExtractImages() {

	// Extract all embedded images for first 5 and last 5 pages but not for page 4.
	selectedPages := []string{"-5", "5-", "!4"}

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	_, err := Process(ExtractImagesCommand("in.pdf", "dirOut", selectedPages, config))
	if err != nil {
		return
	}

}

func exampleProcessListAttachments() {

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = opw"

	list, err := Process(ListAttachmentsCommand("in.pdf", config))
	if err != nil {
		return
	}

	// Print attachment list.
	for _, l := range list {
		fmt.Println(l)
	}

}

func exampleProcessAddAttachments() {

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	_, err := Process(AddAttachmentsCommand("in.pdf", []string{"a.csv", "b.jpg", "c.pdf"}, config))
	if err != nil {
		return
	}
}

func exampleProcessRemoveAttachments() {

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Not to be confused with the ExtractAttachmentsCommand!

	// Remove all attachments.
	_, err := Process(RemoveAttachmentsCommand("in.pdf", nil, config))
	if err != nil {
		return
	}

	// Remove specific attachments.
	_, err = Process(RemoveAttachmentsCommand("in.pdf", []string{"a.csv", "b.jpg"}, config))
	if err != nil {
		return
	}

}

func exampleProcessExtractAttachments() {

	config := pdfcpu.NewDefaultConfiguration()

	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	// Extract all attachments.
	_, err := Process(ExtractAttachmentsCommand("in.pdf", "dirOut", nil, config))
	if err != nil {
		return
	}

	// Extract specific attachments.
	_, err = Process(ExtractAttachmentsCommand("in.pdf", "dirOut", []string{"a.csv", "b.pdf"}, config))
	if err != nil {
		return
	}
}

func exampleProcessEncrypt() {

	config := pdfcpu.NewDefaultConfiguration()

	config.UserPW = "upw"
	config.OwnerPW = "opw"

	_, err := Process(EncryptCommand("in.pdf", "out.pdf", config))
	if err != nil {
		return
	}
}

func exampleProcessDecrypt() {

	config := pdfcpu.NewDefaultConfiguration()

	config.UserPW = "upw"
	config.OwnerPW = "opw"

	_, err := Process(DecryptCommand("in.pdf", "out.pdf", config))
	if err != nil {
		return
	}
}

func exampleProcessChangeUserPW() {

	config := pdfcpu.NewDefaultConfiguration()

	// Provide existing owner pw like so
	config.OwnerPW = "opw"

	pwOld := "pwOld"
	pwNew := "pwNew"

	_, err := Process(ChangeUserPWCommand("in.pdf", "out.pdf", config, &pwOld, &pwNew))
	if err != nil {
		return
	}
}

func exampleProcessChangeOwnerPW() {

	config := pdfcpu.NewDefaultConfiguration()

	// Provide existing user pw like so
	config.UserPW = "upw"

	// old and new owner pw
	pwOld := "pwOld"
	pwNew := "pwNew"

	_, err := Process(ChangeOwnerPWCommand("in.pdf", "out.pdf", config, &pwOld, &pwNew))
	if err != nil {
		return
	}
}

func exampleProcesslistPermissions() {

	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"

	list, err := Process(ListPermissionsCommand("in.pdf", config))
	if err != nil {
		return
	}

	// Print permissions list.
	for _, l := range list {
		fmt.Println(l)
	}
}

func exampleProcessAddPermissions() {

	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = "upw"
	config.OwnerPW = "opw"

	config.UserAccessPermissions = pdfcpu.PermissionsAll

	_, err := Process(AddPermissionsCommand("in.pdf", config))
	if err != nil {
		return
	}

}

func exampleProcessStamp() {

	// Stamp all but the first page.
	selectedPages := []string{"odd,!1"}
	var watermark *pdfcpu.Watermark

	config := pdfcpu.NewDefaultConfiguration()
	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	_, err := Process(AddWatermarksCommand("in.pdf", "out.pdf", selectedPages, watermark, config))
	if err != nil {
		return
	}

}

func exampleProcessWatermark() {

	// Stamp all but the first page.
	selectedPages := []string{"even"}
	var watermark *pdfcpu.Watermark

	config := pdfcpu.NewDefaultConfiguration()
	// Set optional password(s).
	//config.UserPW = "upw"
	//config.OwnerPW = "opw"

	_, err := Process(AddWatermarksCommand("in.pdf", "out.pdf", selectedPages, watermark, config))
	if err != nil {
		return
	}

}

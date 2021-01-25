/*
Copyright 2019 The pdfcpu Authors.

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

// Package cli provides pdfcpu command line processing.
package cli

import (
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// Validate inFile against ISO-32000-1:2008.
func Validate(cmd *Command) ([]string, error) {
	conf := cmd.Conf
	if conf != nil && conf.ValidationMode == pdfcpu.ValidationNone {
		return nil, errors.New("validate: mode == ValidationNone")
	}

	return nil, api.ValidateFile(*cmd.InFile, conf)
}

// Optimize inFile and write result to outFile.
func Optimize(cmd *Command) ([]string, error) {
	return nil, api.OptimizeFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// Encrypt inFile and write result to outFile.
func Encrypt(cmd *Command) ([]string, error) {
	return nil, api.EncryptFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// Decrypt inFile and write result to outFile.
func Decrypt(cmd *Command) ([]string, error) {
	return nil, api.DecryptFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// ChangeUserPassword of inFile and write result to outFile.
func ChangeUserPassword(cmd *Command) ([]string, error) {
	// md.Conf.UserPW = *cmd.PWOld
	// cmd.Conf.UserPWNew = cmd.PWNew
	return nil, api.ChangeUserPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
	// return Optimize(cmd)
}

// ChangeOwnerPassword of inFile and write result to outFile.
func ChangeOwnerPassword(cmd *Command) ([]string, error) {
	// cmd.Conf.OwnerPW = *cmd.PWOld
	// cmd.Conf.OwnerPWNew = cmd.PWNew
	return nil, api.ChangeOwnerPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
	// return Optimize(cmd)
}

// ListPermissions of inFile.
func ListPermissions(cmd *Command) ([]string, error) {
	return api.ListPermissionsFile(*cmd.InFile, cmd.Conf)
}

// SetPermissions of inFile.
func SetPermissions(cmd *Command) ([]string, error) {
	return nil, api.SetPermissionsFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// Split inFile into single page PDFs and write result files to outDir.
func Split(cmd *Command) ([]string, error) {
	return nil, api.SplitFile(*cmd.InFile, *cmd.OutDir, cmd.Span, cmd.Conf)
}

// Trim inFile and write result to outFile.
func Trim(cmd *Command) ([]string, error) {
	return nil, api.TrimFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
}

// Rotate rotates selected pages of inFile and writes result to outFile.
func Rotate(cmd *Command) ([]string, error) {
	return nil, api.RotateFile(*cmd.InFile, *cmd.OutFile, cmd.Rotation, cmd.PageSelection, cmd.Conf)
}

// AddWatermarks adds watermarks or stamps to selected pages of inFile and writes the result to outFile.
func AddWatermarks(cmd *Command) ([]string, error) {
	return nil, api.AddWatermarksFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Watermark, cmd.Conf)
}

// RemoveWatermarks remove watermarks or stamps from selected pages of inFile and writes the result to outFile.
func RemoveWatermarks(cmd *Command) ([]string, error) {
	return nil, api.RemoveWatermarksFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
}

// NUp renders selected PDF pages or image files to outFile in n-up fashion.
func NUp(cmd *Command) ([]string, error) {
	return nil, api.NUpFile(cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
}

// Booklet arranges selected PDF pages to outFile in an order and arrangement that form a small book.
func Booklet(cmd *Command) ([]string, error) {
	return nil, api.BookletFile(cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
}

// ImportImages appends PDF pages containing images to outFile which will be created if necessary.
// ImportImages turns image files into a page sequence and writes the result to outFile.
// In its simplest form this operation converts an image into a PDF.
func ImportImages(cmd *Command) ([]string, error) {
	return nil, api.ImportImagesFile(cmd.InFiles, *cmd.OutFile, cmd.Import, cmd.Conf)
}

// InsertPages inserts a blank page before or after each selected page.
func InsertPages(cmd *Command) ([]string, error) {
	before := true
	if cmd.Mode == pdfcpu.INSERTPAGESAFTER {
		before = false
	}
	return nil, api.InsertPagesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, before, cmd.Conf)
}

// RemovePages removes selected pages.
func RemovePages(cmd *Command) ([]string, error) {
	return nil, api.RemovePagesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
}

// MergeCreate merges inFiles in the order specified and writes the result to outFile.
func MergeCreate(cmd *Command) ([]string, error) {
	return nil, api.MergeCreateFile(cmd.InFiles, *cmd.OutFile, cmd.Conf)
}

// MergeAppend merges inFiles in the order specified and writes the result to outFile.
func MergeAppend(cmd *Command) ([]string, error) {
	return nil, api.MergeAppendFile(cmd.InFiles, *cmd.OutFile, cmd.Conf)
}

// ExtractImages dumps embedded image resources from inFile into outDir for selected pages.
func ExtractImages(cmd *Command) ([]string, error) {
	return nil, api.ExtractImagesFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

// ExtractFonts dumps embedded fontfiles from inFile into outDir for selected pages.
func ExtractFonts(cmd *Command) ([]string, error) {
	return nil, api.ExtractFontsFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

// ExtractPages generates single page PDF files from inFile in outDir for selected pages.
func ExtractPages(cmd *Command) ([]string, error) {
	return nil, api.ExtractPagesFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

// ExtractContent dumps "PDF source" files from inFile into outDir for selected pages.
func ExtractContent(cmd *Command) ([]string, error) {
	return nil, api.ExtractContentFile(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Conf)
}

// ExtractMetadata dumps all metadata dict entries for inFile into outDir.
func ExtractMetadata(cmd *Command) ([]string, error) {
	return nil, api.ExtractMetadataFile(*cmd.InFile, *cmd.OutDir, cmd.Conf)
}

// ListAttachments returns a list of embedded file attachments for inFile.
func ListAttachments(cmd *Command) ([]string, error) {
	return api.ListAttachmentsFile(*cmd.InFile, cmd.Conf)
}

// AddAttachments embeds inFiles into a PDF context read from inFile and writes the result to outFile.
func AddAttachments(cmd *Command) ([]string, error) {
	return nil, api.AddAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Mode == pdfcpu.ADDATTACHMENTSPORTFOLIO, cmd.Conf)
}

// RemoveAttachments deletes inFiles from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachments(cmd *Command) ([]string, error) {
	return nil, api.RemoveAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// ExtractAttachments extracts inFiles from a PDF context read from inFile and writes the result to outFile.
func ExtractAttachments(cmd *Command) ([]string, error) {
	return nil, api.ExtractAttachmentsFile(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Conf)
}

// Info gathers information about inFile and returns the result as []string.
func Info(cmd *Command) ([]string, error) {
	return api.InfoFile(*cmd.InFile, cmd.PageSelection, cmd.Conf)
}

// CreateCheatSheetsFonts creates single page PDF cheat sheets for user fonts in current dir.
func CreateCheatSheetsFonts(cmd *Command) ([]string, error) {
	return nil, api.CreateCheatSheetsUserFonts(cmd.InFiles)
}

// ListFonts gathers information about supported fonts and returns the result as []string.
func ListFonts(cmd *Command) ([]string, error) {
	return api.ListFonts()
}

// InstallFonts installs True Type fonts into the pdfcpu pconfig dir.
func InstallFonts(cmd *Command) ([]string, error) {
	return nil, api.InstallFonts(cmd.InFiles)
}

// ListKeywords returns a list of keywords for inFile.
func ListKeywords(cmd *Command) ([]string, error) {
	return api.ListKeywordsFile(*cmd.InFile, cmd.Conf)
}

// AddKeywords adds keywords to inFile's document info dict and writes the result to outFile.
func AddKeywords(cmd *Command) ([]string, error) {
	return nil, api.AddKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// RemoveKeywords deletes keywords from inFile's document info dict and writes the result to outFile.
func RemoveKeywords(cmd *Command) ([]string, error) {
	return nil, api.RemoveKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// ListProperties returns inFile's properties.
func ListProperties(cmd *Command) ([]string, error) {
	return api.ListPropertiesFile(*cmd.InFile, cmd.Conf)
}

// AddProperties adds properties to inFile's document info dict and writes the result to outFile.
func AddProperties(cmd *Command) ([]string, error) {
	return nil, api.AddPropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringMap, cmd.Conf)
}

// RemoveProperties deletes properties from inFile's document info dict and writes the result to outFile.
func RemoveProperties(cmd *Command) ([]string, error) {
	return nil, api.RemovePropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// Collect creates a custom page sequence for selected pages of inFile and writes result to outFile.
func Collect(cmd *Command) ([]string, error) {
	return nil, api.CollectFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
}

// ListBoxes returns inFile's page boundaries.
func ListBoxes(cmd *Command) ([]string, error) {
	return api.ListBoxesFile(*cmd.InFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
}

// AddBoxes adds page boundaries to inFile's page tree and writes the result to outFile.
func AddBoxes(cmd *Command) ([]string, error) {
	return nil, api.AddBoxesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
}

// RemoveBoxes deletes page boundaries from inFile's page tree and writes the result to outFile.
func RemoveBoxes(cmd *Command) ([]string, error) {
	return nil, api.RemoveBoxesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
}

// Crop adds crop boxes for selected pages of inFile and writes result to outFile.
func Crop(cmd *Command) ([]string, error) {
	return nil, api.CropFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Box, cmd.Conf)
}

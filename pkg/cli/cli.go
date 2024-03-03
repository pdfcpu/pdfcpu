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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Validate inFile against ISO-32000-1:2008.
func Validate(cmd *Command) ([]string, error) {
	return nil, api.ValidateFiles(cmd.InFiles, cmd.Conf)
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
	return nil, api.ChangeUserPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
}

// ChangeOwnerPassword of inFile and write result to outFile.
func ChangeOwnerPassword(cmd *Command) ([]string, error) {
	return nil, api.ChangeOwnerPasswordFile(*cmd.InFile, *cmd.OutFile, *cmd.PWOld, *cmd.PWNew, cmd.Conf)
}

// ListPermissions of inFile.
func ListPermissions(cmd *Command) ([]string, error) {
	return ListPermissionsFile(cmd.InFiles, cmd.Conf)
}

// SetPermissions of inFile.
func SetPermissions(cmd *Command) ([]string, error) {
	return nil, api.SetPermissionsFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// Split inFile into single page PDFs and write result files to outDir.
func Split(cmd *Command) ([]string, error) {
	return nil, api.SplitFile(*cmd.InFile, *cmd.OutDir, cmd.IntVal, cmd.Conf)
}

// Split inFile along pages and write result files to outDir.
func SplitByPageNr(cmd *Command) ([]string, error) {
	return nil, api.SplitByPageNrFile(*cmd.InFile, *cmd.OutDir, cmd.IntVals, cmd.Conf)
}

// Trim inFile and write result to outFile.
func Trim(cmd *Command) ([]string, error) {
	return nil, api.TrimFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
}

// Rotate selected pages of inFile and write result to outFile.
func Rotate(cmd *Command) ([]string, error) {
	return nil, api.RotateFile(*cmd.InFile, *cmd.OutFile, cmd.IntVal, cmd.PageSelection, cmd.Conf)
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
	if cmd.Mode == model.INSERTPAGESAFTER {
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
	return nil, api.MergeCreateFile(cmd.InFiles, *cmd.OutFile, cmd.BoolVal1, cmd.Conf)
}

// MergeCreateZip zips two inFiles in the order specified and writes the result to outFile.
func MergeCreateZip(cmd *Command) ([]string, error) {
	return nil, api.MergeCreateZipFile(cmd.InFiles[0], cmd.InFiles[1], *cmd.OutFile, cmd.Conf)
}

// MergeAppend merges inFiles in the order specified and writes the result to outFile.
func MergeAppend(cmd *Command) ([]string, error) {
	return nil, api.MergeAppendFile(cmd.InFiles, *cmd.OutFile, cmd.BoolVal1, cmd.Conf)
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
	return ListAttachmentsFile(*cmd.InFile, cmd.Conf)
}

// AddAttachments embeds inFiles into a PDF context read from inFile and writes the result to outFile.
func AddAttachments(cmd *Command) ([]string, error) {
	return nil, api.AddAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Mode == model.ADDATTACHMENTSPORTFOLIO, cmd.Conf)
}

// RemoveAttachments deletes inFiles from a PDF context read from inFile and writes the result to outFile.
func RemoveAttachments(cmd *Command) ([]string, error) {
	return nil, api.RemoveAttachmentsFile(*cmd.InFile, *cmd.OutFile, cmd.InFiles, cmd.Conf)
}

// ExtractAttachments extracts inFiles from a PDF context read from inFile and writes the result to outFile.
func ExtractAttachments(cmd *Command) ([]string, error) {
	return nil, api.ExtractAttachmentsFile(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Conf)
}

// ListInfo gathers information about inFile and returns the result as []string.
func ListInfo(cmd *Command) ([]string, error) {
	return ListInfoFiles(cmd.InFiles, cmd.PageSelection, cmd.BoolVal1, cmd.Conf)
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
	return ListKeywordsFile(*cmd.InFile, cmd.Conf)
}

// AddKeywords adds keywords to inFile's document info dict and writes the result to outFile.
func AddKeywords(cmd *Command) ([]string, error) {
	return nil, api.AddKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// RemoveKeywords deletes keywords from inFile's document info dict and writes the result to outFile.
func RemoveKeywords(cmd *Command) ([]string, error) {
	return nil, api.RemoveKeywordsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// ListProperties returns inFile's properties.
func ListProperties(cmd *Command) ([]string, error) {
	return ListPropertiesFile(*cmd.InFile, cmd.Conf)
}

// AddProperties adds properties to inFile's document info dict and writes the result to outFile.
func AddProperties(cmd *Command) ([]string, error) {
	return nil, api.AddPropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringMap, cmd.Conf)
}

// RemoveProperties deletes properties from inFile's document info dict and writes the result to outFile.
func RemoveProperties(cmd *Command) ([]string, error) {
	return nil, api.RemovePropertiesFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// Collect creates a custom page sequence for selected pages of inFile and writes result to outFile.
func Collect(cmd *Command) ([]string, error) {
	return nil, api.CollectFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
}

// ListBoxes returns inFile's page boundaries.
func ListBoxes(cmd *Command) ([]string, error) {
	return ListBoxesFile(*cmd.InFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
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

// ListAnnotations returns inFile's page annotations.
func ListAnnotations(cmd *Command) ([]string, error) {
	_, ss, err := ListAnnotationsFile(*cmd.InFile, cmd.PageSelection, cmd.Conf)
	return ss, err
}

// RemoveAnnotations deletes annotations from inFile's page tree and writes the result to outFile.
func RemoveAnnotations(cmd *Command) ([]string, error) {
	incr := false // No incremental writing on cli.
	return nil, api.RemoveAnnotationsFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.StringVals, cmd.IntVals, cmd.Conf, incr)
}

// ListImages returns inFiles embedded images.
func ListImages(cmd *Command) ([]string, error) {
	return ListImagesFile(cmd.InFiles, cmd.PageSelection, cmd.Conf)
}

// Dump known object to stdout.
func Dump(cmd *Command) ([]string, error) {
	mode := cmd.IntVals[0]
	objNr := cmd.IntVals[1]
	return nil, api.DumpObjectFile(*cmd.InFile, mode, objNr, cmd.Conf)
}

// Create renders page content corresponding to declarations found in inFileJSON and writes the result to outFile.
// If inFile is present, page content will be appended,
func Create(cmd *Command) ([]string, error) {
	return nil, api.CreateFile(*cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.Conf)
}

// ListFormFields returns inFile's form field ids.
func ListFormFields(cmd *Command) ([]string, error) {
	return ListFormFieldsFile(cmd.InFiles, cmd.Conf)
}

// RemoveFormFields removes some form fields from inFile.
func RemoveFormFields(cmd *Command) ([]string, error) {
	return nil, api.RemoveFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// LockFormFields makes some or all form fields of inFile read-only.
func LockFormFields(cmd *Command) ([]string, error) {
	return nil, api.LockFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// UnlockFormFields makes some or all form fields of inFile writeable.
func UnlockFormFields(cmd *Command) ([]string, error) {
	return nil, api.UnlockFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// ResetFormFields sets some or all form fields of inFile to the corresponding default value.
func ResetFormFields(cmd *Command) ([]string, error) {
	return nil, api.ResetFormFieldsFile(*cmd.InFile, *cmd.OutFile, cmd.StringVals, cmd.Conf)
}

// ExportFormFields returns a representation of inFile's form as outFileJSON.
func ExportFormFields(cmd *Command) ([]string, error) {
	return nil, api.ExportFormFile(*cmd.InFile, *cmd.OutFileJSON, cmd.Conf)
}

// FillFormFields fills out inFile's form using data represented by inFileJSON.
func FillFormFields(cmd *Command) ([]string, error) {
	return nil, api.FillFormFile(*cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.Conf)
}

// MultiFillFormFields fills out multiple instances of inFile's form using JSON or CSV data.
func MultiFillFormFields(cmd *Command) ([]string, error) {
	return nil, api.MultiFillFormFile(*cmd.InFile, *cmd.InFileJSON, *cmd.OutDir, *cmd.OutFile, cmd.BoolVal1, cmd.Conf)
}

// Resize selected pages and write result to outFile.
func Resize(cmd *Command) ([]string, error) {
	return nil, api.ResizeFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Resize, cmd.Conf)
}

// Create poster for selected pages and write result PDFs into outDir.
func Poster(cmd *Command) ([]string, error) {
	return nil, api.PosterFile(*cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
}

// NDown selected pages and write result PDFs into outDir.
func NDown(cmd *Command) ([]string, error) {
	return nil, api.NDownFile(*cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.IntVal, cmd.Cut, cmd.Conf)
}

// Cut selected pages and write result PDFs into outDir.
func Cut(cmd *Command) ([]string, error) {
	return nil, api.CutFile(*cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
}

// ListBookmarks returns inFile's outlines.
func ListBookmarks(cmd *Command) ([]string, error) {
	return ListBookmarksFile(*cmd.InFile, cmd.Conf)
}

// ExportBookmarks returns a representation of inFile's outlines as outFileJSON.
func ExportBookmarks(cmd *Command) ([]string, error) {
	return nil, api.ExportBookmarksFile(*cmd.InFile, *cmd.OutFileJSON, cmd.Conf)
}

// ImportBookmarks creates/replaces outlines of inFile corresponding to declarations found in inJSONFile and writes the result to outFile.
func ImportBookmarks(cmd *Command) ([]string, error) {
	return nil, api.ImportBookmarksFile(*cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.BoolVal1, cmd.Conf)
}

// RemoveBookmarks erases outlines of inFile.
func RemoveBookmarks(cmd *Command) ([]string, error) {
	return nil, api.RemoveBookmarksFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// ListPageLayout returns inFile's page layout.
func ListPageLayout(cmd *Command) ([]string, error) {
	return api.ListPageLayoutFile(*cmd.InFile, cmd.Conf)
}

// SetPageLayout sets inFile's page layout.
func SetPageLayout(cmd *Command) ([]string, error) {
	pageLayout := model.PageLayoutFor(cmd.StringVal)
	return nil, api.SetPageLayoutFile(*cmd.InFile, *cmd.OutFile, *pageLayout, cmd.Conf)
}

// ResetPageLayout resets inFile's page layout.
func ResetPageLayout(cmd *Command) ([]string, error) {
	return nil, api.ResetPageLayoutFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// ListPageMode returns inFile's page mode.
func ListPageMode(cmd *Command) ([]string, error) {
	return api.ListPageModeFile(*cmd.InFile, cmd.Conf)
}

// SetPageMode sets inFile's page mode.
func SetPageMode(cmd *Command) ([]string, error) {
	pageMode := model.PageModeFor(cmd.StringVal)
	return nil, api.SetPageModeFile(*cmd.InFile, *cmd.OutFile, *pageMode, cmd.Conf)
}

// ResetPageMode resets inFile's page mode.
func ResetPageMode(cmd *Command) ([]string, error) {
	return nil, api.ResetPageModeFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// ListViewerPreferences returns inFile's viewer preferences.
func ListViewerPreferences(cmd *Command) ([]string, error) {
	return api.ListViewerPreferencesFile(*cmd.InFile, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
}

// SetViewerPreferences sets inFile's viewer preferences.
func SetViewerPreferences(cmd *Command) ([]string, error) {
	if *cmd.InFileJSON != "" {
		return nil, api.SetViewerPreferencesFileFromJSONFile(*cmd.InFile, *cmd.OutFile, *cmd.InFileJSON, cmd.Conf)
	}
	return nil, api.SetViewerPreferencesFileFromJSONBytes(*cmd.InFile, *cmd.OutFile, []byte(cmd.StringVal), cmd.Conf)
}

// ResetViewerPreferences resets inFile's viewer preferences.
func ResetViewerPreferences(cmd *Command) ([]string, error) {
	return nil, api.ResetViewerPreferencesFile(*cmd.InFile, *cmd.OutFile, cmd.Conf)
}

// Zoom in/out of selected pages either by zoom factor or corresponding margin.
func Zoom(cmd *Command) ([]string, error) {
	return nil, api.ZoomFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Zoom, cmd.Conf)
}

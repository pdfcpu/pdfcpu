/*
Copyright 2026 The pdfcpu Authors.

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

package cli

import "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

// AddWatermarksCommand creates a new command to add watermarks to a file.
func AddWatermarksCommand(inFile, outFile string, pageSelection []string, wm *model.Watermark, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.ADDWATERMARKS
	}
	return &Command{
		Mode:          model.ADDWATERMARKS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Watermark:     wm,
		Conf:          conf}
}

// RemoveWatermarksCommand creates a new command to remove watermarks from a file.
func RemoveWatermarksCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.REMOVEWATERMARKS
	}
	return &Command{
		Mode:          model.REMOVEWATERMARKS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

func listAnnotationsCommand(inFile string, pageSelection []string, json bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTANNOTATIONS
	}
	return &Command{
		Mode:          model.LISTANNOTATIONS,
		InFile:        &inFile,
		PageSelection: pageSelection,
		BoolVal1:      json,
		Conf:          conf}
}

// ListAnnotationsCommand creates a new command to list annotations for selected pages.
func ListAnnotationsCommand(inFile string, pageSelection []string, conf *model.Configuration) *Command {
	return listAnnotationsCommand(inFile, pageSelection, false, conf)
}

// ListAnnotationsJSONCommand creates a new command to list annotations as JSON.
func ListAnnotationsJSONCommand(inFile string, pageSelection []string, conf *model.Configuration) *Command {
	return listAnnotationsCommand(inFile, pageSelection, true, conf)
}

// RemoveAnnotationsCommand creates a new command to remove annotations for selected pages.
func RemoveAnnotationsCommand(inFile, outFile string, pageSelection []string, idsAndTypes []string, objNrs []int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.REMOVEANNOTATIONS
	}
	return &Command{
		Mode:          model.REMOVEANNOTATIONS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		StringVals:    idsAndTypes,
		IntVals:       objNrs,
		Conf:          conf}
}

// ListBookmarksCommand creates a new command to list bookmarks of inFile.
func ListBookmarksCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTBOOKMARKS
	}
	return &Command{
		Mode:   model.LISTBOOKMARKS,
		InFile: &inFile,
		Conf:   conf}
}

// ExportBookmarksCommand creates a new command to export bookmarks of inFile.
func ExportBookmarksCommand(inFile, outFileJSON string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXPORTBOOKMARKS
	}
	return &Command{
		Mode:        model.EXPORTBOOKMARKS,
		InFile:      &inFile,
		OutFileJSON: &outFileJSON,
		Conf:        conf}
}

// ImportBookmarksCommand creates a new command to import bookmarks to inFile.
func ImportBookmarksCommand(inFile, inFileJSON, outFile string, replace bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.IMPORTBOOKMARKS
	}
	return &Command{
		Mode:       model.IMPORTBOOKMARKS,
		BoolVal1:   replace,
		InFile:     &inFile,
		InFileJSON: &inFileJSON,
		OutFile:    &outFile,
		Conf:       conf}
}

// RemoveBookmarksCommand creates a new command to remove all bookmarks from inFile.
func RemoveBookmarksCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.REMOVEBOOKMARKS
	}
	return &Command{
		Mode:    model.REMOVEBOOKMARKS,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ListPageLayoutCommand creates a new command to list the document page layout.
func ListPageLayoutCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTPAGELAYOUT
	}
	return &Command{
		Mode:   model.LISTPAGELAYOUT,
		InFile: &inFile,
		Conf:   conf}
}

// SetPageLayoutCommand creates a new command to set the document page layout.
func SetPageLayoutCommand(inFile, outFile, value string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.SETPAGELAYOUT
	}
	return &Command{
		Mode:      model.SETPAGELAYOUT,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringVal: value,
		Conf:      conf}
}

// ResetPageLayoutCommand creates a new command to reset the document page layout.
func ResetPageLayoutCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.RESETPAGELAYOUT
	}
	return &Command{
		Mode:    model.RESETPAGELAYOUT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ListPageModeCommand creates a new command to list the document page mode.
func ListPageModeCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTPAGEMODE
	}
	return &Command{
		Mode:   model.LISTPAGEMODE,
		InFile: &inFile,
		Conf:   conf}
}

// SetPageModeCommand creates a new command to set the document page mode.
func SetPageModeCommand(inFile, outFile, value string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.SETPAGEMODE
	}
	return &Command{
		Mode:      model.SETPAGEMODE,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringVal: value,
		Conf:      conf}
}

// ResetPageModeCommand creates a new command to reset the document page mode.
func ResetPageModeCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.RESETPAGEMODE
	}
	return &Command{
		Mode:    model.RESETPAGEMODE,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ListViewerPreferencesCommand creates a new command to list the viewer preferences.
func ListViewerPreferencesCommand(inFile string, all, json bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTVIEWERPREFERENCES
	}
	return &Command{
		Mode:     model.LISTVIEWERPREFERENCES,
		InFile:   &inFile,
		BoolVal1: all,
		BoolVal2: json,
		Conf:     conf}
}

// SetViewerPreferencesCommand creates a new command to set the viewer preferences.
func SetViewerPreferencesCommand(inFilePDF, inFileJSON, outFilePDF, stringJSON string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.SETVIEWERPREFERENCES
	}
	return &Command{
		Mode:       model.SETVIEWERPREFERENCES,
		InFile:     &inFilePDF,
		InFileJSON: &inFileJSON,
		OutFile:    &outFilePDF,
		StringVal:  stringJSON,
		Conf:       conf}
}

// ResetViewerPreferencesCommand creates a new command to reset the viewer preferences.
func ResetViewerPreferencesCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.RESETVIEWERPREFERENCES
	}
	return &Command{
		Mode:    model.RESETVIEWERPREFERENCES,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

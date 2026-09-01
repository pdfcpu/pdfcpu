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

// ValidateCommand creates a new command to validate a file.
func ValidateCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.VALIDATE
	}
	return &Command{
		Mode:    model.VALIDATE,
		InFiles: inFiles,
		Conf:    conf}
}

// OptimizeCommand creates a new command to optimize a file.
func OptimizeCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.OPTIMIZE
	}
	return &Command{
		Mode:    model.OPTIMIZE,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// InfoCommand creates a new command to output information about inFile.
func InfoCommand(inFiles []string, pageSelection []string, fonts, json bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTINFO
	}
	return &Command{
		Mode:          model.LISTINFO,
		InFiles:       inFiles,
		PageSelection: pageSelection,
		BoolVal1:      fonts,
		BoolVal2:      json,
		Conf:          conf}
}

// DumpCommand creates a new command to dump objects on stdout.
func DumpCommand(inFilePDF string, vals []int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.DUMP
	}
	return &Command{
		Mode:    model.DUMP,
		InFile:  &inFilePDF,
		IntVals: vals,
		Conf:    conf}
}

// CreateCommand creates a new command to create a PDF file.
func CreateCommand(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.CREATE
	}
	return &Command{
		Mode:       model.CREATE,
		InFile:     &inFilePDF,
		InFileJSON: &inFileJSON,
		OutFile:    &outFilePDF,
		Conf:       conf}
}

// MergeCreateCommand creates a new command to merge files.
// Outfile will be created. An existing outFile will be overwritten.
func MergeCreateCommand(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.MERGECREATE
	}
	return &Command{
		Mode:     model.MERGECREATE,
		InFiles:  inFiles,
		OutFile:  &outFile,
		BoolVal1: dividerPage,
		Conf:     conf}
}

// MergeCreateZipCommand creates a new command to zip merge 2 files.
// Outfile will be created. An existing outFile will be overwritten.
func MergeCreateZipCommand(inFiles []string, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.MERGECREATEZIP
	}
	return &Command{
		Mode:    model.MERGECREATEZIP,
		InFiles: inFiles,
		OutFile: &outFile,
		Conf:    conf}
}

// MergeAppendCommand creates a new command to merge files.
// Any existing outFile PDF content will be preserved and serves as the beginning of the merge result.
func MergeAppendCommand(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.MERGEAPPEND
	}
	return &Command{
		Mode:     model.MERGEAPPEND,
		InFiles:  inFiles,
		OutFile:  &outFile,
		BoolVal1: dividerPage,
		Conf:     conf}
}

// SplitCommand creates a new command to split a file according to span or along bookmarks.
func SplitCommand(inFile, dirNameOut string, span int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.SPLIT
	}
	return &Command{
		Mode:   model.SPLIT,
		InFile: &inFile,
		OutDir: &dirNameOut,
		IntVal: span,
		Conf:   conf}
}

// SplitByPageNrCommand creates a new command to split a file into files along given pages.
func SplitByPageNrCommand(inFile, dirNameOut string, pageNrs []int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.SPLITBYPAGENR
	}
	return &Command{
		Mode:    model.SPLITBYPAGENR,
		InFile:  &inFile,
		OutDir:  &dirNameOut,
		IntVals: pageNrs,
		Conf:    conf}
}

// TrimCommand creates a new command to trim the pages of a file.
func TrimCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.TRIM
	}
	return &Command{
		Mode:          model.TRIM,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// CollectCommand creates a new command to create a custom PDF page sequence.
func CollectCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.COLLECT
	}
	return &Command{
		Mode:          model.COLLECT,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

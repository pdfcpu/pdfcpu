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

func listFormFieldsCommand(inFiles []string, json bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LISTFORMFIELDS
	}
	return &Command{
		Mode:     model.LISTFORMFIELDS,
		InFiles:  inFiles,
		BoolVal1: json,
		Conf:     conf}
}

// ListFormFieldsCommand creates a new command to list the field ids from a PDF form.
func ListFormFieldsCommand(inFiles []string, conf *model.Configuration) *Command {
	return listFormFieldsCommand(inFiles, false, conf)
}

// ListFormFieldsJSONCommand creates a new command to list PDF form fields as export JSON.
func ListFormFieldsJSONCommand(inFiles []string, conf *model.Configuration) *Command {
	return listFormFieldsCommand(inFiles, true, conf)
}

// RemoveFormFieldsCommand creates a new command to remove fields from a PDF form.
func RemoveFormFieldsCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.REMOVEFORMFIELDS
	}
	return &Command{
		Mode:       model.REMOVEFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// LockFormCommand creates a new command to lock PDF form fields.
func LockFormCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.LOCKFORMFIELDS
	}
	return &Command{
		Mode:       model.LOCKFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// UnlockFormCommand creates a new command to unlock PDF form fields.
func UnlockFormCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.UNLOCKFORMFIELDS
	}
	return &Command{
		Mode:       model.UNLOCKFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// ResetFormCommand creates a new command to reset PDF form fields.
func ResetFormCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.RESETFORMFIELDS
	}
	return &Command{
		Mode:       model.RESETFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// ExportFormCommand creates a new command to export a PDF form.
func ExportFormCommand(inFilePDF, outFileJSON string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXPORTFORMFIELDS
	}
	return &Command{
		Mode:        model.EXPORTFORMFIELDS,
		InFile:      &inFilePDF,
		OutFileJSON: &outFileJSON,
		Conf:        conf}
}

// FillFormCommand creates a new command to fill a PDF form with data.
func FillFormCommand(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.FILLFORMFIELDS
	}
	return &Command{
		Mode:       model.FILLFORMFIELDS,
		InFile:     &inFilePDF,
		InFileJSON: &inFileJSON,
		OutFile:    &outFilePDF,
		Conf:       conf}
}

// MultiFillFormCommand creates a new command to fill multiple PDF forms with JSON or CSV data.
func MultiFillFormCommand(inFilePDF, inFileData, outDir, outFilePDF string, merge bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.MULTIFILLFORMFIELDS
	}
	return &Command{
		Mode:       model.MULTIFILLFORMFIELDS,
		InFile:     &inFilePDF,
		InFileJSON: &inFileData, // TODO Fix name clash.
		OutDir:     &outDir,
		OutFile:    &outFilePDF,
		BoolVal1:   merge,
		Conf:       conf}
}

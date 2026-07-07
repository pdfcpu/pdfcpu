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

import (
	"maps"
	"slices"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ImportImagesCommand creates a new command to import images.
func ImportImagesCommand(imageFiles []string, outFile string, imp *pdfcpu.Import, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.IMPORTIMAGES
	return &Command{
		Mode:    model.IMPORTIMAGES,
		InFiles: imageFiles,
		OutFile: &outFile,
		Import:  imp,
		Conf:    conf}
}

// ListFontsCommand returns a list of supported fonts.
func ListFontsCommand(conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFONTS
	return &Command{
		Mode: model.LISTFONTS,
		Conf: conf}
}

// InstallFontsCommand installs true type fonts for embedding.
func InstallFontsCommand(fontFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.INSTALLFONTS
	return &Command{
		Mode:    model.INSTALLFONTS,
		InFiles: fontFiles,
		Conf:    conf}
}

// CreateCheatSheetsFontsCommand creates single page PDF cheat sheets in current dir.
func CreateCheatSheetsFontsCommand(fontFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CHEATSHEETSFONTS
	return &Command{
		Mode:    model.CHEATSHEETSFONTS,
		InFiles: fontFiles,
		Conf:    conf}
}

// ListImagesCommand creates a new command to list images for selected pages.
func ListImagesCommand(inFiles []string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTIMAGES
	return &Command{
		Mode:          model.LISTIMAGES,
		InFiles:       inFiles,
		PageSelection: pageSelection,
		Conf:          conf}
}

// UpdateImagesCommand creates a new command to update images.
func UpdateImagesCommand(inFile, imageFile, outFile string, objNrOrPageNr int, id string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.UPDATEIMAGES

	return &Command{
		Mode:      model.UPDATEIMAGES,
		InFiles:   []string{inFile, imageFile},
		OutFile:   &outFile,
		IntVal:    objNrOrPageNr,
		StringVal: id,
		Conf:      conf}
}

// ListAttachmentsCommand creates a new command to list attachments.
func ListAttachmentsCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTATTACHMENTS
	return &Command{
		Mode:   model.LISTATTACHMENTS,
		InFile: &inFile,
		Conf:   conf}
}

// AddAttachmentsCommand creates a new command to add attachments.
func AddAttachmentsCommand(inFile, outFile string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDATTACHMENTS
	return &Command{
		Mode:    model.ADDATTACHMENTS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// AddAttachmentsPortfolioCommand creates a new command to add attachments to a portfolio.
func AddAttachmentsPortfolioCommand(inFile, outFile string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDATTACHMENTSPORTFOLIO
	return &Command{
		Mode:    model.ADDATTACHMENTSPORTFOLIO,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// RemoveAttachmentsCommand creates a new command to remove attachments.
func RemoveAttachmentsCommand(inFile, outFile string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEATTACHMENTS
	return &Command{
		Mode:    model.REMOVEATTACHMENTS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// ExtractAttachmentsCommand creates a new command to extract attachments.
func ExtractAttachmentsCommand(inFile string, outDir string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTATTACHMENTS
	return &Command{
		Mode:    model.EXTRACTATTACHMENTS,
		InFile:  &inFile,
		OutDir:  &outDir,
		InFiles: fileNames,
		Conf:    conf}
}

// ListKeywordsCommand creates a new command to list keywords.
func ListKeywordsCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTKEYWORDS
	return &Command{
		Mode:   model.LISTKEYWORDS,
		InFile: &inFile,
		Conf:   conf}
}

// AddKeywordsCommand creates a new command to add keywords.
func AddKeywordsCommand(inFile, outFile string, keywords []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	if outFile == "" {
		outFile = inFile
	}
	conf.Cmd = model.ADDKEYWORDS
	return &Command{
		Mode:       model.ADDKEYWORDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: slices.Clone(keywords),
		Conf:       conf}
}

// RemoveKeywordsCommand creates a new command to remove keywords.
func RemoveKeywordsCommand(inFile, outFile string, keywords []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	if outFile == "" {
		outFile = inFile
	}
	conf.Cmd = model.REMOVEKEYWORDS
	return &Command{
		Mode:       model.REMOVEKEYWORDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: slices.Clone(keywords),
		Conf:       conf}
}

// ListPropertiesCommand creates a new command to list document properties.
func ListPropertiesCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPROPERTIES
	return &Command{
		Mode:   model.LISTPROPERTIES,
		InFile: &inFile,
		Conf:   conf}
}

// AddPropertiesCommand creates a new command to add document properties.
func AddPropertiesCommand(inFile, outFile string, properties map[string]string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDPROPERTIES
	return &Command{
		Mode:      model.ADDPROPERTIES,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringMap: maps.Clone(properties),
		Conf:      conf}
}

// RemovePropertiesCommand creates a new command to remove document properties.
func RemovePropertiesCommand(inFile, outFile string, propKeys []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEPROPERTIES
	return &Command{
		Mode:       model.REMOVEPROPERTIES,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: slices.Clone(propKeys),
		Conf:       conf}
}

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

// ExtractImagesCommand creates a new command to extract embedded images.
func ExtractImagesCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXTRACTIMAGES
	}
	return &Command{
		Mode:          model.EXTRACTIMAGES,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractFontsCommand creates a new command to extract embedded fonts.
// (experimental)
func ExtractFontsCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXTRACTFONTS
	}
	return &Command{
		Mode:          model.EXTRACTFONTS,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractPagesCommand creates a new command to extract specific pages of a file.
func ExtractPagesCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXTRACTPAGES
	}
	return &Command{
		Mode:          model.EXTRACTPAGES,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractContentCommand creates a new command to extract page content streams.
func ExtractContentCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXTRACTCONTENT
	}
	return &Command{
		Mode:          model.EXTRACTCONTENT,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractMetadataCommand creates a new command to extract metadata streams.
func ExtractMetadataCommand(inFile string, outDir string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
		conf.Cmd = model.EXTRACTMETADATA
	}
	return &Command{
		Mode:   model.EXTRACTMETADATA,
		InFile: &inFile,
		OutDir: &outDir,
		Conf:   conf}
}

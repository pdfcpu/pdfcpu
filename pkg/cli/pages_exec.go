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
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// NUp renders selected PDF pages or image files to outFile in n-up fashion.
func NUp(cmd *Command) ([]string, error) {
	if cmd == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.NUp == nil {
		return nil, api.ErrMissingNUpConfiguration
	}
	if len(cmd.InFiles) == 0 {
		if cmd.NUp.ImgInputFile {
			return nil, api.ErrMissingImageInput
		}
		return nil, api.ErrMissingPDFInput
	}
	if *cmd.OutFile != "-" && cmd.InFiles[0] != "-" {
		return nil, api.NUpFile(cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
	}
	inFile := ""
	if !cmd.NUp.ImgInputFile {
		inFile = cmd.InFiles[0]
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, *cmd.OutFile, "n-up")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.NUp(rs, w, cmd.InFiles, cmd.PageSelection, cmd.NUp, cmd.Conf))
}

// Grid renders selected PDF pages or image files to outFile in a grid.
func Grid(cmd *Command) ([]string, error) {
	if cmd == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.NUp == nil {
		return nil, api.ErrMissingGridConfiguration
	}
	if len(cmd.InFiles) == 0 {
		if cmd.NUp.ImgInputFile {
			return nil, api.ErrMissingImageInput
		}
		return nil, api.ErrMissingPDFInput
	}
	if *cmd.OutFile != "-" && cmd.InFiles[0] != "-" {
		return nil, api.GridFile(cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
	}

	inFile := ""
	if !cmd.NUp.ImgInputFile {
		inFile = cmd.InFiles[0]
	}
	rs, w, finalize, err := streamInOutForOperation(inFile, *cmd.OutFile, "grid")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Grid(rs, w, cmd.InFiles, cmd.PageSelection, cmd.NUp, cmd.Conf))
}

// Booklet arranges selected PDF pages to outFile in an order and arrangement that form a small book.
func Booklet(cmd *Command) ([]string, error) {
	if cmd == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.NUp == nil {
		return nil, api.ErrMissingBookletConfiguration
	}
	if len(cmd.InFiles) == 0 {
		if cmd.NUp.ImgInputFile {
			return nil, api.ErrMissingImageInput
		}
		return nil, api.ErrMissingPDFInput
	}
	if *cmd.OutFile != "-" && cmd.InFiles[0] != "-" {
		return nil, api.BookletFile(cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
	}
	inFile := ""
	if !cmd.NUp.ImgInputFile {
		inFile = cmd.InFiles[0]
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, *cmd.OutFile, "booklet")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Booklet(rs, w, cmd.InFiles, cmd.PageSelection, cmd.NUp, cmd.Conf))
}

// Resize selected pages and write result to outFile.
func Resize(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil || *cmd.InFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.Resize == nil {
		return nil, api.ErrMissingResizeConfiguration
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ResizeFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Resize, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "resize")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Resize(rs, w, cmd.PageSelection, cmd.Resize, cmd.Conf))
}

func validateCutCommand(cmd *Command) error {
	if cmd == nil || cmd.InFile == nil || *cmd.InFile == "" {
		return api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil || cmd.OutDir == nil || *cmd.OutDir == "" {
		return api.ErrMissingPDFOutput
	}
	if cmd.Cut == nil {
		return api.ErrMissingCutConfiguration
	}
	return nil
}

// Poster creates a poster for selected pages and writes result PDFs into outDir.
func Poster(cmd *Command) ([]string, error) {
	if err := validateCutCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		outFile := *cmd.OutFile
		if outFile == "" {
			outFile = "stdin"
		}
		return withStdinReadSeeker("poster", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.Poster(rs, *cmd.OutDir, outFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
		})
	}

	return nil, api.PosterFile(*cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
}

// NDown selected pages and write result PDFs into outDir.
func NDown(cmd *Command) ([]string, error) {
	if err := validateCutCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		outFile := *cmd.OutFile
		if outFile == "" {
			outFile = "stdin"
		}
		return withStdinReadSeeker("ndown", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.NDown(rs, *cmd.OutDir, outFile, cmd.PageSelection, cmd.IntVal, cmd.Cut, cmd.Conf)
		})
	}

	return nil, api.NDownFile(*cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.IntVal, cmd.Cut, cmd.Conf)
}

// Cut selected pages and write result PDFs into outDir.
func Cut(cmd *Command) ([]string, error) {
	if err := validateCutCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		outFile := *cmd.OutFile
		if outFile == "" {
			outFile = "stdin"
		}
		return withStdinReadSeeker("cut", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.Cut(rs, *cmd.OutDir, outFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
		})
	}

	return nil, api.CutFile(*cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
}

func validateZoomCommand(cmd *Command) error {
	if cmd == nil || cmd.InFile == nil || *cmd.InFile == "" {
		return api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return api.ErrMissingPDFOutput
	}
	if cmd.Zoom == nil {
		return api.ErrMissingZoomConfiguration
	}
	return nil
}

// Zoom zooms selected pages either by factor or corresponding margin.
func Zoom(cmd *Command) ([]string, error) {
	if err := validateZoomCommand(cmd); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ZoomFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Zoom, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "zoom")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Zoom(rs, w, cmd.PageSelection, cmd.Zoom, cmd.Conf))
}

// Rotate selected pages of inFile and write result to outFile.
func Rotate(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RotateFile(*cmd.InFile, *cmd.OutFile, cmd.IntVal, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "rotate")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Rotate(rs, w, cmd.IntVal, cmd.PageSelection, cmd.Conf))
}

// InsertPages inserts a blank page before or after each selected page.
func InsertPages(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil || *cmd.InFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	before := true
	if cmd.Mode == model.INSERTPAGESAFTER {
		before = false
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.InsertPagesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, before, cmd.PageConf, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "insert pages")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.InsertPages(rs, w, cmd.PageSelection, before, cmd.PageConf, cmd.Conf))
}

// RemovePages removes selected pages.
func RemovePages(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil || *cmd.InFile == "" {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemovePagesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "remove pages")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemovePages(rs, w, cmd.PageSelection, cmd.Conf))
}

// Crop adds crop boxes for selected pages of inFile and writes result to outFile.
func Crop(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.Box == nil {
		return nil, api.ErrMissingBoxConfiguration
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.CropFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Box, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "crop")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Crop(rs, w, cmd.PageSelection, cmd.Box, cmd.Conf))
}

// ListBoxesFile returns a list of page boundaries for selected pages of inFile.
func ListBoxesFile(inFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) ([]string, error) {
	return api.ListBoxesFile(inFile, selectedPages, pb, conf)
}

// ListBoxes returns inFile's page boundaries.
func ListBoxes(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil {
		return nil, api.ErrMissingPDFInput
	}
	if *cmd.InFile == "-" {
		return withStdinReadSeeker("list boxes", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListBoxes(rs, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
		})
	}

	return ListBoxesFile(*cmd.InFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
}

// AddBoxes adds page boundaries to inFile's page tree and writes the result to outFile.
func AddBoxes(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.PageBoundaries == nil {
		return nil, api.ErrMissingPageBoundaries
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddBoxesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "add boxes")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.AddBoxes(rs, w, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf))
}

// RemoveBoxes deletes page boundaries from inFile's page tree and writes the result to outFile.
func RemoveBoxes(cmd *Command) ([]string, error) {
	if cmd == nil || cmd.InFile == nil {
		return nil, api.ErrMissingPDFInput
	}
	if cmd.OutFile == nil {
		return nil, api.ErrMissingPDFOutput
	}
	if cmd.PageBoundaries == nil {
		return nil, api.ErrMissingPageBoundaries
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveBoxesFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "remove boxes")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveBoxes(rs, w, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf))
}

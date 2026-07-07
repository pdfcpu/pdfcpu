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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// NUpCommand creates a new command to render PDFs or image files in n-up fashion.
func NUpCommand(inFiles []string, outFile string, pageSelection []string, nUp *model.NUp, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NUP
	return &Command{
		Mode:          model.NUP,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nUp,
		Conf:          conf}
}

// GridCommand creates a new command to arrange PDF pages or images in a grid.
func GridCommand(inFiles []string, outFile string, pageSelection []string, nup *model.NUp, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.GRID
	return &Command{
		Mode:          model.GRID,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nup,
		Conf:          conf}
}

// BookletCommand creates a new command to render PDFs or image files in booklet fashion.
func BookletCommand(inFiles []string, outFile string, pageSelection []string, nup *model.NUp, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.BOOKLET
	return &Command{
		Mode:          model.BOOKLET,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nup,
		Conf:          conf}
}

// ResizeCommand creates a new command to scale selected pages.
func ResizeCommand(inFile, outFile string, pageSelection []string, resize *model.Resize, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESIZE
	return &Command{
		Mode:          model.RESIZE,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Resize:        resize,
		Conf:          conf}
}

// PosterCommand creates a new command to cut and slice pages horizontally or vertically.
func PosterCommand(inFile, outDir, outFile string, pageSelection []string, cut *model.Cut, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.POSTER
	return &Command{
		Mode:          model.POSTER,
		InFile:        &inFile,
		OutDir:        &outDir,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Cut:           cut,
		Conf:          conf}
}

// NDownCommand creates a new command to cut and slice pages horizontally or vertically.
func NDownCommand(inFile, outDir, outFile string, pageSelection []string, n int, cut *model.Cut, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NDOWN
	return &Command{
		Mode:          model.NDOWN,
		InFile:        &inFile,
		OutDir:        &outDir,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		IntVal:        n,
		Cut:           cut,
		Conf:          conf}
}

// CutCommand creates a new command to cut and slice pages horizontally or vertically.
func CutCommand(inFile, outDir, outFile string, pageSelection []string, cut *model.Cut, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CUT
	return &Command{
		Mode:          model.CUT,
		InFile:        &inFile,
		OutDir:        &outDir,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Cut:           cut,
		Conf:          conf}
}

// ZoomCommand creates a new command to zoom in/out of selected pages.
func ZoomCommand(inFile, outFile string, pageSelection []string, zoom *model.Zoom, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ZOOM
	return &Command{
		Mode:          model.ZOOM,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Zoom:          zoom,
		Conf:          conf}
}

// InsertPagesCommand creates a new command to insert a blank page before or after selected pages.
func InsertPagesCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration, mode string, pageConf *pdfcpu.PageConfiguration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	cmdMode := model.INSERTPAGESBEFORE
	if mode == "after" {
		cmdMode = model.INSERTPAGESAFTER
	}
	conf.Cmd = cmdMode
	return &Command{
		Mode:          cmdMode,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		PageConf:      pageConf,
		Conf:          conf}
}

// RemovePagesCommand creates a new command to remove selected pages.
func RemovePagesCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEPAGES
	return &Command{
		Mode:          model.REMOVEPAGES,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// RotateCommand creates a new command to rotate pages.
func RotateCommand(inFile, outFile string, rotation int, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ROTATE
	return &Command{
		Mode:          model.ROTATE,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		IntVal:        rotation,
		Conf:          conf}
}

// CropCommand creates a new command to apply a cropBox to selected pages.
func CropCommand(inFile, outFile string, pageSelection []string, box *model.Box, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CROP
	return &Command{
		Mode:          model.CROP,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Box:           box,
		Conf:          conf}
}

// ListBoxesCommand creates a new command to list page boundaries for selected pages.
func ListBoxesCommand(inFile string, pageSelection []string, pb *model.PageBoundaries, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTBOXES
	return &Command{
		Mode:           model.LISTBOXES,
		InFile:         &inFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// AddBoxesCommand creates a new command to add page boundaries for selected pages.
func AddBoxesCommand(inFile, outFile string, pageSelection []string, pb *model.PageBoundaries, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDBOXES
	return &Command{
		Mode:           model.ADDBOXES,
		InFile:         &inFile,
		OutFile:        &outFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// RemoveBoxesCommand creates a new command to remove page boundaries for selected pages.
func RemoveBoxesCommand(inFile, outFile string, pageSelection []string, pb *model.PageBoundaries, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEBOXES
	return &Command{
		Mode:           model.REMOVEBOXES,
		InFile:         &inFile,
		OutFile:        &outFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

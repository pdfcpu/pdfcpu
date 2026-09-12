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
	"context"
	"io"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateNUpLikeCommand(cmd *Command, operation string, missingConfiguration error) error {
	requirements := commandRequirements{
		operation: operation,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return err
	}
	if cmd.NUp == nil {
		return commandValidationError(operation, missingConfiguration)
	}
	missingInput := api.ErrMissingPDFInput
	if cmd.NUp.ImgInputFile {
		missingInput = api.ErrMissingImageInput
	}
	err := validateCommandInputFiles(cmd.InFiles, 1, 0, missingInput)
	return commandValidationError(operation, err)
}

func validatePageInputOutputCommand(cmd *Command, operation string) error {
	return validateCommandRequirements(cmd, commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	})
}

func nUp(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateNUpLikeCommand(cmd, "n-up", api.ErrMissingNUpConfiguration); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.OutFile != "-" && cmd.InFiles[0] != "-" {
		return nil, api.NUpFile(c, cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
	}
	inFile := ""
	if !cmd.NUp.ImgInputFile {
		inFile = cmd.InFiles[0]
	}

	rs, w, finalize, err := streamInOutForOperation(c, inFile, *cmd.OutFile, "n-up")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.NUp(c, rs, w, cmd.InFiles, cmd.PageSelection, cmd.NUp, cmd.Conf))
}

func grid(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateNUpLikeCommand(cmd, "grid", api.ErrMissingGridConfiguration); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.OutFile != "-" && cmd.InFiles[0] != "-" {
		return nil, api.GridFile(c, cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
	}

	inFile := ""
	if !cmd.NUp.ImgInputFile {
		inFile = cmd.InFiles[0]
	}
	rs, w, finalize, err := streamInOutForOperation(c, inFile, *cmd.OutFile, "grid")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Grid(c, rs, w, cmd.InFiles, cmd.PageSelection, cmd.NUp, cmd.Conf))
}

func booklet(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateNUpLikeCommand(cmd, "booklet", api.ErrMissingBookletConfiguration); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.OutFile != "-" && cmd.InFiles[0] != "-" {
		return nil, api.BookletFile(c, cmd.InFiles, *cmd.OutFile, cmd.PageSelection, cmd.NUp, cmd.Conf)
	}
	inFile := ""
	if !cmd.NUp.ImgInputFile {
		inFile = cmd.InFiles[0]
	}

	rs, w, finalize, err := streamInOutForOperation(c, inFile, *cmd.OutFile, "booklet")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Booklet(c, rs, w, cmd.InFiles, cmd.PageSelection, cmd.NUp, cmd.Conf))
}

func resize(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "resize"); err != nil {
		return nil, err
	}
	if cmd.Resize == nil {
		return nil, commandValidationError("resize", api.ErrMissingResizeConfiguration)
	}
	reportCommandProgress(cmd, "resizing %s\n", *cmd.InFile)
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ResizeFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Resize, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "resize")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Resize(c, rs, w, cmd.PageSelection, cmd.Resize, cmd.Conf))
}

func validateCutCommand(cmd *Command, operation string) error {
	requirements := commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
		outDir:    commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return err
	}
	if cmd.Cut == nil {
		return commandValidationError(operation, api.ErrMissingCutConfiguration)
	}
	return nil
}

func cutInputLabel(inFile string) string {
	if inFile == "-" {
		return "stdin"
	}
	return inFile
}

func poster(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCutCommand(cmd, "poster"); err != nil {
		return nil, err
	}
	reportCommandProgress(cmd, "creating poster pages from %s into %s/ ...\n", cutInputLabel(*cmd.InFile), *cmd.OutDir)
	if *cmd.InFile == "-" {
		outFile := *cmd.OutFile
		if outFile == "" {
			outFile = "stdin"
		}
		return withStdinReadSeeker(c, "poster", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.Poster(c, rs, *cmd.OutDir, outFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
		})
	}

	return nil, api.PosterFile(
		c, *cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.Cut, cmd.Conf,
	)
}

func nDown(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCutCommand(cmd, "ndown"); err != nil {
		return nil, err
	}
	reportCommandProgress(cmd, "ndown %s into %s/ ...\n", cutInputLabel(*cmd.InFile), *cmd.OutDir)
	if *cmd.InFile == "-" {
		outFile := *cmd.OutFile
		if outFile == "" {
			outFile = "stdin"
		}
		return withStdinReadSeeker(c, "ndown", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.NDown(c, rs, *cmd.OutDir, outFile, cmd.PageSelection, cmd.IntVal, cmd.Cut, cmd.Conf)
		})
	}

	return nil, api.NDownFile(
		c, *cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.IntVal, cmd.Cut, cmd.Conf,
	)
}

func cut(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCutCommand(cmd, "cut"); err != nil {
		return nil, err
	}
	reportCommandProgress(cmd, "cutting %s into %s/ ...\n", cutInputLabel(*cmd.InFile), *cmd.OutDir)
	if *cmd.InFile == "-" {
		outFile := *cmd.OutFile
		if outFile == "" {
			outFile = "stdin"
		}
		return withStdinReadSeeker(c, "cut", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.Cut(c, rs, *cmd.OutDir, outFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
		})
	}

	return nil, api.CutFile(c, *cmd.InFile, *cmd.OutDir, *cmd.OutFile, cmd.PageSelection, cmd.Cut, cmd.Conf)
}

func validateZoomCommand(cmd *Command) error {
	if err := validatePageInputOutputCommand(cmd, "zoom"); err != nil {
		return err
	}
	if cmd.Zoom == nil {
		return commandValidationError("zoom", api.ErrMissingZoomConfiguration)
	}
	return nil
}

func zoom(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateZoomCommand(cmd); err != nil {
		return nil, err
	}
	reportCommandProgress(cmd, "zooming %s\n", *cmd.InFile)
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.ZoomFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Zoom, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "zoom")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Zoom(c, rs, w, cmd.PageSelection, cmd.Zoom, cmd.Conf))
}

func rotate(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "rotate"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RotateFile(c, *cmd.InFile, *cmd.OutFile, cmd.IntVal, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "rotate")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Rotate(c, rs, w, cmd.IntVal, cmd.PageSelection, cmd.Conf))
}

func insertPages(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "insert pages"); err != nil {
		return nil, err
	}
	before := true
	if cmd.Mode == model.INSERTPAGESAFTER {
		before = false
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.InsertPagesFile(
			c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, before, cmd.PageConf, cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "insert pages")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.InsertPages(c, rs, w, cmd.PageSelection, before, cmd.PageConf, cmd.Conf))
}

func removePages(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "remove pages"); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemovePagesFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "remove pages")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemovePages(c, rs, w, cmd.PageSelection, cmd.Conf))
}

func crop(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "crop"); err != nil {
		return nil, err
	}
	if cmd.Box == nil {
		return nil, commandValidationError("crop", api.ErrMissingBoxConfiguration)
	}
	reportCommandProgress(cmd, "cropping %s\n", *cmd.InFile)
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.CropFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Box, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "crop")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Crop(c, rs, w, cmd.PageSelection, cmd.Box, cmd.Conf))
}

func listBoxesFile(c context.Context, inFile string, selectedPages []string, pb *model.PageBoundaries, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, commandValidationError("list boxes", api.ErrMissingPDFInput)
	}
	return api.ListBoxesFile(c, inFile, selectedPages, pb, conf)
}

func pageBoundariesForPresentation(pb *model.PageBoundaries) *model.PageBoundaries {
	if pb != nil {
		return pb
	}
	pb = &model.PageBoundaries{}
	pb.SelectAll()
	return pb
}

func listBoxes(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "list boxes")
	if err != nil {
		return nil, err
	}
	reportCommandProgress(cmd, "listing %s for %s\n", pageBoundariesForPresentation(cmd.PageBoundaries), inFile)
	if inFile == "-" {
		return withStdinReadSeeker(c, "list boxes", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListBoxes(c, rs, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
		})
	}

	return listBoxesFile(c, inFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf)
}

func addBoxes(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "add boxes"); err != nil {
		return nil, err
	}
	if cmd.PageBoundaries == nil {
		return nil, commandValidationError("add boxes", api.ErrMissingPageBoundaries)
	}
	reportCommandProgress(cmd, "adding %s for %s\n", cmd.PageBoundaries, *cmd.InFile)
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddBoxesFile(
			c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "add boxes")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.AddBoxes(c, rs, w, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf))
}

func removeBoxes(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validatePageInputOutputCommand(cmd, "remove boxes"); err != nil {
		return nil, err
	}
	if cmd.PageBoundaries == nil {
		return nil, commandValidationError("remove boxes", api.ErrMissingPageBoundaries)
	}
	reportCommandProgress(cmd, "removing %s for %s\n", cmd.PageBoundaries, *cmd.InFile)
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveBoxesFile(
			c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(c, *cmd.InFile, *cmd.OutFile, "remove boxes")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveBoxes(c, rs, w, cmd.PageSelection, cmd.PageBoundaries, cmd.Conf))
}

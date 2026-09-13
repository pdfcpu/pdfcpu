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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateWatermarkCommand(cmd *Command, operation string, requireWatermark bool) error {
	requirements := commandRequirements{
		operation: operation,
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return err
	}
	if requireWatermark && cmd.Watermark == nil {
		return commandValidationError(operation, api.ErrMissingWatermarkConfiguration)
	}
	return nil
}

func addWatermarks(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateWatermarkCommand(cmd, "add watermarks", true); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddWatermarksFile(
			c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Watermark, cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "add watermarks")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.AddWatermarks(c, rs, w, cmd.PageSelection, cmd.Watermark, cmd.Conf))
}

func removeWatermarks(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateWatermarkCommand(cmd, "remove watermarks", false); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveWatermarksFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "remove watermarks")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveWatermarks(c, rs, w, cmd.PageSelection, cmd.Conf))
}

func listAnnotations(c context.Context, rs io.ReadSeeker, selectedPages []string, json bool, conf *model.Configuration) (int, []string, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, nil, err
	}
	if json {
		log.SetCLILogger(nil)
	}
	annots, err := api.Annotations(c, rs, selectedPages, conf)
	if err != nil {
		return 0, nil, err
	}
	if json {
		return pdfcpu.ListAnnotationsJSON(c, annots)
	}

	return pdfcpu.ListAnnotations(c, annots)
}

func closeListAnnotationsInput(f *os.File, err error) error {
	return errors.Join(err, closeStreamFile(f, "list annotations: close input"))
}

func listAnnotationsFile(c context.Context, inFile string, selectedPages []string, json bool, conf *model.Configuration) (count int, ss []string, err error) {
	const op = "list annotations"

	if err := contextutil.Check(c); err != nil {
		return 0, nil, err
	}
	f, err := os.Open(inFile)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}
	defer func() {
		err = closeListAnnotationsInput(f, err)
	}()

	return listAnnotations(c, f, selectedPages, json, conf)
}

// ListAnnotationsFile returns a list of page annotations of inFile and supports cancellation.
func ListAnnotationsFile(c context.Context, inFile string, selectedPages []string, conf *model.Configuration) (int, []string, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, nil, err
	}
	if inFile == "" {
		return 0, nil, commandValidationError("list annotations", api.ErrMissingPDFInput)
	}
	return listAnnotationsFile(c, inFile, selectedPages, false, conf)
}

// ListAnnotationsJSONFile returns a JSON list of page annotations of inFile and supports cancellation.
func ListAnnotationsJSONFile(c context.Context, inFile string, selectedPages []string, conf *model.Configuration) (int, []string, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, nil, err
	}
	if inFile == "" {
		return 0, nil, commandValidationError("list annotations", api.ErrMissingPDFInput)
	}
	return listAnnotationsFile(c, inFile, selectedPages, true, conf)
}

func listAnnotationsForCommand(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "list annotations")
	if err != nil {
		return nil, err
	}
	if inFile == "-" {
		return withStdinReadSeeker(c, cmd.Conf, "list annotations", func(rs io.ReadSeeker) ([]string, error) {
			_, ss, err := listAnnotations(c, rs, cmd.PageSelection, cmd.BoolVal1, cmd.Conf)
			return ss, err
		})
	}

	_, ss, err := listAnnotationsFile(c, inFile, cmd.PageSelection, cmd.BoolVal1, cmd.Conf)
	return ss, err
}

func removeAnnotations(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	requirements := commandRequirements{
		operation: "remove annotations",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		incr := false // No incremental writing on cli.
		return nil, api.RemoveAnnotationsFile(
			c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.StringVals, cmd.IntVals, cmd.Conf, incr,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "remove annotations")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveAnnotations(
		c, rs, w, cmd.PageSelection, cmd.StringVals, cmd.IntVals, cmd.Conf,
	))
}

// ListBookmarksFile returns inFile's bookmarks and supports cancellation.
func ListBookmarksFile(c context.Context, inFile string, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	return api.ListBookmarksFile(c, inFile, conf)
}

func listBookmarks(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "list bookmarks")
	if err != nil {
		return nil, err
	}
	if inFile == "-" {
		return withStdinReadSeeker(c, cmd.Conf, "list bookmarks", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListBookmarks(c, rs, cmd.Conf)
		})
	}

	return api.ListBookmarksFile(c, inFile, cmd.Conf)
}

func exportBookmarks(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	requirements := commandRequirements{
		operation:   "export bookmarks",
		inFile:      commandStringRequiredNonEmpty,
		outFileJSON: commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	inFile := *cmd.InFile
	outFileJSON := *cmd.OutFileJSON
	reportOutputPath(outFileJSON)
	if inFile != "-" && outFileJSON != "-" {
		return nil, api.ExportBookmarksFile(c, inFile, outFileJSON, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFileJSON, "export bookmarks")
	if err != nil {
		return nil, err
	}

	source := inFile
	if source == "-" {
		source = "stdin"
	}

	return nil, finalize(api.ExportBookmarksJSON(c, rs, w, source, cmd.Conf))
}

func importBookmarks(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	requirements := commandRequirements{
		operation:  "import bookmarks",
		inFile:     commandStringRequiredNonEmpty,
		inFileJSON: commandStringRequiredNonEmpty,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	inFile := *cmd.InFile
	inFileJSON := *cmd.InFileJSON
	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.ImportBookmarksFile(c, inFile, inFileJSON, outFile, cmd.BoolVal1, cmd.Conf)
	}

	f, err := os.Open(inFileJSON)
	if err != nil {
		return nil, err
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "import bookmarks")
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	opErr := api.ImportBookmarks(c, rs, f, w, cmd.BoolVal1, cmd.Conf)
	opErr = errors.Join(opErr, closeStreamFile(f, "import bookmarks: close JSON input"))
	return nil, finalize(opErr)
}

func removeBookmarks(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "remove bookmarks")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.RemoveBookmarksFile(c, inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "remove bookmarks")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveBookmarks(c, rs, w, cmd.Conf))
}

func listPageLayout(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "list page layout")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		return withStdinReadSeeker(c, cmd.Conf, "list page layout", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListPageLayout(c, rs, cmd.Conf)
		})
	}

	return api.ListPageLayoutFile(c, inFile, cmd.Conf)
}

func setPageLayout(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "set page layout")
	if err != nil {
		return nil, err
	}

	pageLayout := model.PageLayoutFor(cmd.StringVal)
	if pageLayout == nil {
		return nil, fmt.Errorf("set page layout %q: %w", cmd.StringVal, api.ErrInvalidPageLayout)
	}

	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.SetPageLayoutFile(c, inFile, outFile, *pageLayout, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "set page layout")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.SetPageLayout(c, rs, w, *pageLayout, cmd.Conf))
}

func resetPageLayout(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "reset page layout")
	if err != nil {
		return nil, err
	}

	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.ResetPageLayoutFile(c, inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "reset page layout")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.ResetPageLayout(c, rs, w, cmd.Conf))
}

func listPageMode(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "list page mode")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		return withStdinReadSeeker(c, cmd.Conf, "list page mode", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListPageMode(c, rs, cmd.Conf)
		})
	}

	return api.ListPageModeFile(c, inFile, cmd.Conf)
}

func setPageMode(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "set page mode")
	if err != nil {
		return nil, err
	}

	pageMode := model.PageModeFor(cmd.StringVal)
	if pageMode == nil {
		return nil, fmt.Errorf("set page mode %q: %w", cmd.StringVal, api.ErrInvalidPageMode)
	}

	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.SetPageModeFile(c, inFile, outFile, *pageMode, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "set page mode")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.SetPageMode(c, rs, w, *pageMode, cmd.Conf))
}

func resetPageMode(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "reset page mode")
	if err != nil {
		return nil, err
	}

	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.ResetPageModeFile(c, inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "reset page mode")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.ResetPageMode(c, rs, w, cmd.Conf))
}

func listViewerPreferences(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "list viewer preferences")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		return withStdinReadSeeker(c, cmd.Conf, "list viewer preferences", func(rs io.ReadSeeker) ([]string, error) {
			if !cmd.BoolVal2 {
				return api.ListViewerPreferences(c, rs, cmd.BoolVal1, cmd.Conf)
			}
			return api.ListViewerPreferencesJSON(c, rs, cmd.BoolVal1, cmd.Conf)
		})
	}

	return api.ListViewerPreferencesFile(c, inFile, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
}

func setViewerPreferences(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "set viewer preferences")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	jsonInput := optionalCommandString(cmd.InFileJSON)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		if jsonInput != "" {
			return nil, api.SetViewerPreferencesFileFromJSONFile(c, inFile, outFile, jsonInput, cmd.Conf)
		}
		return nil, api.SetViewerPreferencesFileFromJSONBytes(
			c, inFile, outFile, []byte(cmd.StringVal), cmd.Conf,
		)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "set viewer preferences")
	if err != nil {
		return nil, err
	}

	if jsonInput != "" {
		f, err := os.Open(jsonInput)
		if err != nil {
			return nil, finalize(fmt.Errorf("set viewer preferences: read JSON %s: %w", jsonInput, err))
		}
		err = api.SetViewerPreferencesFromJSONReader(c, rs, w, f, cmd.Conf)
		err = errors.Join(err, closeStreamFile(f, "set viewer preferences: close JSON input"))
		return nil, finalize(err)
	}

	return nil, finalize(api.SetViewerPreferencesFromJSONBytes(
		c, rs, w, []byte(cmd.StringVal), cmd.Conf,
	))
}

func resetViewerPreferences(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	inFile, err := validatedCommandInFile(cmd, "reset viewer preferences")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	reportCommandOutputPath(cmd)
	if inFile != "-" && outFile != "-" {
		return nil, api.ResetViewerPreferencesFile(c, inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, inFile, outFile, "reset viewer preferences")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.ResetViewerPreferences(c, rs, w, cmd.Conf))
}

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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func runContentStreamOperation(inFile, outFile, op string, fn func(io.ReadSeeker, io.Writer) error) error {
	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, op)
	if err != nil {
		return err
	}
	return finalize(fn(rs, w))
}

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

// AddWatermarks adds watermarks or stamps to selected pages of inFile and writes the result to outFile.
func AddWatermarks(cmd *Command) ([]string, error) {
	if err := validateWatermarkCommand(cmd, "add watermarks", true); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.AddWatermarksFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Watermark, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "add watermarks")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.AddWatermarks(rs, w, cmd.PageSelection, cmd.Watermark, cmd.Conf))
}

// RemoveWatermarks removes watermarks or stamps from selected pages of inFile and writes the result to outFile.
func RemoveWatermarks(cmd *Command) ([]string, error) {
	if err := validateWatermarkCommand(cmd, "remove watermarks", false); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.RemoveWatermarksFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "remove watermarks")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveWatermarks(rs, w, cmd.PageSelection, cmd.Conf))
}

func listAnnotations(rs io.ReadSeeker, selectedPages []string, json bool, conf *model.Configuration) (int, []string, error) {
	if json {
		log.SetCLILogger(nil)
	}
	annots, err := api.Annotations(rs, selectedPages, conf)
	if err != nil {
		return 0, nil, err
	}
	if json {
		return pdfcpu.ListAnnotationsJSON(annots)
	}

	return pdfcpu.ListAnnotations(annots)
}

func closeListAnnotationsInput(f *os.File, err error) error {
	return errors.Join(err, closeStreamFile(f, "list annotations: close input"))
}

func listAnnotationsFile(inFile string, selectedPages []string, json bool, conf *model.Configuration) (count int, ss []string, err error) {
	const op = "list annotations"

	f, err := os.Open(inFile)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: open input %s: %w", op, inFile, err)
	}
	defer func() {
		err = closeListAnnotationsInput(f, err)
	}()

	return listAnnotations(f, selectedPages, json, conf)
}

// ListAnnotationsFile returns a list of page annotations of inFile.
func ListAnnotationsFile(inFile string, selectedPages []string, conf *model.Configuration) (int, []string, error) {
	if inFile == "" {
		return 0, nil, commandValidationError("list annotations", api.ErrMissingPDFInput)
	}
	return listAnnotationsFile(inFile, selectedPages, false, conf)
}

// ListAnnotationsJSONFile returns a JSON list of page annotations of inFile.
func ListAnnotationsJSONFile(inFile string, selectedPages []string, conf *model.Configuration) (int, []string, error) {
	if inFile == "" {
		return 0, nil, commandValidationError("list annotations", api.ErrMissingPDFInput)
	}
	return listAnnotationsFile(inFile, selectedPages, true, conf)
}

// ListAnnotations returns inFile's page annotations.
func ListAnnotations(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list annotations")
	if err != nil {
		return nil, err
	}
	if inFile == "-" {
		return withStdinReadSeeker("list annotations", func(rs io.ReadSeeker) ([]string, error) {
			_, ss, err := listAnnotations(rs, cmd.PageSelection, cmd.BoolVal1, cmd.Conf)
			return ss, err
		})
	}

	_, ss, err := listAnnotationsFile(inFile, cmd.PageSelection, cmd.BoolVal1, cmd.Conf)
	return ss, err
}

// RemoveAnnotations deletes annotations from inFile's page tree and writes the result to outFile.
func RemoveAnnotations(cmd *Command) ([]string, error) {
	requirements := commandRequirements{
		operation: "remove annotations",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}
	if err := validateCommandRequirements(cmd, requirements); err != nil {
		return nil, err
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		incr := false // No incremental writing on cli.
		return nil, api.RemoveAnnotationsFile(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.StringVals, cmd.IntVals, cmd.Conf, incr)
	}

	rs, w, finalize, err := streamInOutForOperation(*cmd.InFile, *cmd.OutFile, "remove annotations")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveAnnotations(rs, w, cmd.PageSelection, cmd.StringVals, cmd.IntVals, cmd.Conf))
}

// ListBookmarksFile returns inFile's bookmarks.
// Deprecated: use api.ListBookmarksFile.
func ListBookmarksFile(inFile string, conf *model.Configuration) ([]string, error) {
	return api.ListBookmarksFile(inFile, conf)
}

// ListBookmarks returns inFile's bookmarks.
func ListBookmarks(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list bookmarks")
	if err != nil {
		return nil, err
	}
	if inFile == "-" {
		return withStdinReadSeeker("list bookmarks", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListBookmarks(rs, cmd.Conf)
		})
	}

	return api.ListBookmarksFile(inFile, cmd.Conf)
}

// ExportBookmarks exports inFile's bookmarks to outFileJSON.
func ExportBookmarks(cmd *Command) ([]string, error) {
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
	if inFile != "-" && outFileJSON != "-" {
		return nil, api.ExportBookmarksFile(inFile, outFileJSON, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, outFileJSON, "export bookmarks")
	if err != nil {
		return nil, err
	}

	source := inFile
	if source == "-" {
		source = "stdin"
	}

	return nil, finalize(api.ExportBookmarksJSON(rs, w, source, cmd.Conf))
}

// ImportBookmarks creates or replaces inFile's bookmarks using inFileJSON and writes the result to outFile.
func ImportBookmarks(cmd *Command) ([]string, error) {
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
	if inFile != "-" && outFile != "-" {
		return nil, api.ImportBookmarksFile(inFile, inFileJSON, outFile, cmd.BoolVal1, cmd.Conf)
	}

	f, err := os.Open(inFileJSON)
	if err != nil {
		return nil, err
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, "import bookmarks")
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	opErr := api.ImportBookmarks(rs, f, w, cmd.BoolVal1, cmd.Conf)
	opErr = errors.Join(opErr, closeStreamFile(f, "import bookmarks: close JSON input"))
	return nil, finalize(opErr)
}

// RemoveBookmarks removes bookmarks from inFile.
func RemoveBookmarks(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "remove bookmarks")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	if inFile != "-" && outFile != "-" {
		return nil, api.RemoveBookmarksFile(inFile, outFile, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(inFile, outFile, "remove bookmarks")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.RemoveBookmarks(rs, w, cmd.Conf))
}

// ListPageLayout returns inFile's page layout.
func ListPageLayout(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list page layout")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		return withStdinReadSeeker("list page layout", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListPageLayout(rs, cmd.Conf)
		})
	}

	return api.ListPageLayoutFile(inFile, cmd.Conf)
}

// SetPageLayout sets inFile's page layout.
func SetPageLayout(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "set page layout")
	if err != nil {
		return nil, err
	}

	pageLayout := model.PageLayoutFor(cmd.StringVal)
	if pageLayout == nil {
		return nil, fmt.Errorf("set page layout %q: %w", cmd.StringVal, api.ErrInvalidPageLayout)
	}

	outFile := optionalCommandString(cmd.OutFile)
	if inFile != "-" && outFile != "-" {
		return nil, api.SetPageLayoutFile(inFile, outFile, *pageLayout, cmd.Conf)
	}

	err = runContentStreamOperation(inFile, outFile, "set page layout", func(rs io.ReadSeeker, w io.Writer) error {
		return api.SetPageLayout(rs, w, *pageLayout, cmd.Conf)
	})
	return nil, err
}

// ResetPageLayout resets inFile's page layout.
func ResetPageLayout(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "reset page layout")
	if err != nil {
		return nil, err
	}

	outFile := optionalCommandString(cmd.OutFile)
	if inFile != "-" && outFile != "-" {
		return nil, api.ResetPageLayoutFile(inFile, outFile, cmd.Conf)
	}

	err = runContentStreamOperation(inFile, outFile, "reset page layout", func(rs io.ReadSeeker, w io.Writer) error {
		return api.ResetPageLayout(rs, w, cmd.Conf)
	})
	return nil, err
}

// ListPageMode returns inFile's page mode.
func ListPageMode(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list page mode")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		return withStdinReadSeeker("list page mode", func(rs io.ReadSeeker) ([]string, error) {
			return api.ListPageMode(rs, cmd.Conf)
		})
	}

	return api.ListPageModeFile(inFile, cmd.Conf)
}

// SetPageMode sets inFile's page mode.
func SetPageMode(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "set page mode")
	if err != nil {
		return nil, err
	}

	pageMode := model.PageModeFor(cmd.StringVal)
	if pageMode == nil {
		return nil, fmt.Errorf("set page mode %q: %w", cmd.StringVal, api.ErrInvalidPageMode)
	}

	outFile := optionalCommandString(cmd.OutFile)
	if inFile != "-" && outFile != "-" {
		return nil, api.SetPageModeFile(inFile, outFile, *pageMode, cmd.Conf)
	}

	err = runContentStreamOperation(inFile, outFile, "set page mode", func(rs io.ReadSeeker, w io.Writer) error {
		return api.SetPageMode(rs, w, *pageMode, cmd.Conf)
	})
	return nil, err
}

// ResetPageMode resets inFile's page mode.
func ResetPageMode(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "reset page mode")
	if err != nil {
		return nil, err
	}

	outFile := optionalCommandString(cmd.OutFile)
	if inFile != "-" && outFile != "-" {
		return nil, api.ResetPageModeFile(inFile, outFile, cmd.Conf)
	}

	err = runContentStreamOperation(inFile, outFile, "reset page mode", func(rs io.ReadSeeker, w io.Writer) error {
		return api.ResetPageMode(rs, w, cmd.Conf)
	})
	return nil, err
}

// ListViewerPreferences returns inFile's viewer preferences.
func ListViewerPreferences(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "list viewer preferences")
	if err != nil {
		return nil, err
	}

	if inFile == "-" {
		return withStdinReadSeeker("list viewer preferences", func(rs io.ReadSeeker) ([]string, error) {
			if !cmd.BoolVal2 {
				return api.ListViewerPreferences(rs, cmd.BoolVal1, cmd.Conf)
			}
			return api.ListViewerPreferencesJSON(rs, cmd.BoolVal1, cmd.Conf)
		})
	}

	return api.ListViewerPreferencesFile(inFile, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
}

// SetViewerPreferences sets inFile's viewer preferences.
func SetViewerPreferences(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "set viewer preferences")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	jsonInput := optionalCommandString(cmd.InFileJSON)
	if inFile != "-" && outFile != "-" {
		if jsonInput != "" {
			return nil, api.SetViewerPreferencesFileFromJSONFile(inFile, outFile, jsonInput, cmd.Conf)
		}
		return nil, api.SetViewerPreferencesFileFromJSONBytes(inFile, outFile, []byte(cmd.StringVal), cmd.Conf)
	}

	var jsonBytes []byte
	if jsonInput != "" {
		jsonBytes, err = os.ReadFile(jsonInput)
		if err != nil {
			return nil, fmt.Errorf("set viewer preferences: read JSON %s: %w", jsonInput, err)
		}
	} else {
		jsonBytes = []byte(cmd.StringVal)
	}
	err = runContentStreamOperation(inFile, outFile, "set viewer preferences", func(rs io.ReadSeeker, w io.Writer) error {
		return api.SetViewerPreferencesFromJSONBytes(rs, w, jsonBytes, cmd.Conf)
	})
	return nil, err
}

// ResetViewerPreferences resets inFile's viewer preferences.
func ResetViewerPreferences(cmd *Command) ([]string, error) {
	inFile, err := validatedCommandInFile(cmd, "reset viewer preferences")
	if err != nil {
		return nil, err
	}
	outFile := optionalCommandString(cmd.OutFile)
	if inFile != "-" && outFile != "-" {
		return nil, api.ResetViewerPreferencesFile(inFile, outFile, cmd.Conf)
	}

	err = runContentStreamOperation(inFile, outFile, "reset viewer preferences", func(rs io.ReadSeeker, w io.Writer) error {
		return api.ResetViewerPreferences(rs, w, cmd.Conf)
	})
	return nil, err
}

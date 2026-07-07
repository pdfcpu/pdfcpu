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

package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/spf13/cobra"
)

type bookmarksImportOptions struct {
	replaceBookmarks bool
}

func bookmarksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "List, import, export, remove bookmarks",
		Long:  usageLongBookmarks,
	}
	addPersistentPasswordFlags(cmd)

	importOpts := &bookmarksImportOptions{replaceBookmarks: false}
	importCmd := &cobra.Command{
		Use:   "import inFile inFileJSON [ outFile ]",
		Short: "Import bookmarks",
		Args:  cobra.RangeArgs(2, 3),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleImportBookmarksCommand(conf, args, importOpts)
		}),
	}
	importCmd.Flags().BoolVarP(&importOpts.replaceBookmarks, "replace", "r", importOpts.replaceBookmarks, "replace existing bookmarks")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List bookmarks",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListBookmarksCommand),
		},
		&cobra.Command{
			Use:   "export inFile [ outFileJSON ]",
			Short: "Export bookmarks",
			Args:  cobra.RangeArgs(1, 2),
			RunE:  wrapHandler(handleExportBookmarksCommand),
		},
		importCmd,
		&cobra.Command{
			Use:   "remove inFile [ outFile ]",
			Short: "Remove bookmarks",
			Args:  cobra.RangeArgs(1, 2),
			RunE:  wrapHandler(handleRemoveBookmarksCommand),
		},
	)

	return cmd
}

type viewerpreferencesListOptions struct {
	all  bool
	json bool
}

func pagelayoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pagelayout",
		Short: "List, set, reset page layout for opened document",
		Long:  usageLongPageLayout,
	}
	addPersistentPasswordFlags(cmd)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List page layout",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListPageLayoutCommand),
		},
		&cobra.Command{
			Use:   "set inFile value [ outFile ]",
			Short: "Set page layout",
			Args:  cobra.RangeArgs(2, 3),
			RunE:  wrapHandler(handleSetPageLayoutCommand),
		},
		&cobra.Command{
			Use:   "reset inFile [ outFile ]",
			Short: "Reset page layout",
			Args:  cobra.RangeArgs(1, 2),
			RunE:  wrapHandler(handleResetPageLayoutCommand),
		},
	)

	return cmd
}

func pagemodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pagemode",
		Short: "List, set, reset page mode for opened document",
		Long:  usageLongPageMode,
	}
	addPersistentPasswordFlags(cmd)

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile",
			Short: "List page mode",
			Args:  cobra.ExactArgs(1),
			RunE:  wrapHandler(handleListPageModeCommand),
		},
		&cobra.Command{
			Use:   "set inFile value [ outFile ]",
			Short: "Set page mode",
			Args:  cobra.RangeArgs(2, 3),
			RunE:  wrapHandler(handleSetPageModeCommand),
		},
		&cobra.Command{
			Use:   "reset inFile [ outFile ]",
			Short: "Reset page mode",
			Args:  cobra.RangeArgs(1, 2),
			RunE:  wrapHandler(handleResetPageModeCommand),
		},
	)

	return cmd
}

func viewerprefCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "viewerpref",
		Short: "List, set, reset viewer preferences",
		Long:  usageLongViewerPreferences,
	}
	addPersistentPasswordFlags(cmd)

	listOpts := &viewerpreferencesListOptions{all: false, json: false}
	list := &cobra.Command{
		Use:   "list inFile",
		Short: "List viewer preferences",
		Args:  cobra.ExactArgs(1),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleListViewerPreferencesCommand(conf, args, listOpts)
		}),
	}
	list.Flags().BoolVarP(&listOpts.all, "all", "a", listOpts.all, "output all (including default values)")
	list.Flags().BoolVarP(&listOpts.json, "json", "j", listOpts.json, "output JSON")

	cmd.AddCommand(
		list,
		&cobra.Command{
			Use:   "set inFile ( inFileJSON | JSONstring ) [ outFile ]",
			Short: "Set viewer preferences",
			Args:  cobra.RangeArgs(2, 3),
			RunE:  wrapHandler(handleSetViewerPreferencesCommand),
		},
		&cobra.Command{
			Use:   "reset inFile [ outFile ]",
			Short: "Reset viewer preferences",
			Args:  cobra.RangeArgs(1, 2),
			RunE:  wrapHandler(handleResetViewerPreferencesCommand),
		},
	)

	return cmd
}

func listSinglePDFCommand(conf *model.Configuration, args []string, command func(string, *model.Configuration) *cli.Command) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}
	return runCommand(command(inFile, conf))
}

func handleListPageLayoutCommand(conf *model.Configuration, args []string) error {
	return listSinglePDFCommand(conf, args, cli.ListPageLayoutCommand)
}

func setDocumentViewCommand(conf *model.Configuration, args []string, valid func(string) bool, invalidMsg string, command func(string, string, string, *model.Configuration) *cli.Command) error {
	v := args[1]
	if !valid(v) {
		return errors.New(invalidMsg)
	}
	inFile, outFile, err := optionalOutputPDFArgs(conf, append([]string{args[0]}, args[2:]...))
	if err != nil {
		return err
	}
	return runCommand(command(inFile, outFile, v, conf))
}

func handleSetPageLayoutCommand(conf *model.Configuration, args []string) error {
	return setDocumentViewCommand(
		conf,
		args,
		validate.DocumentPageLayout,
		"invalid page layout, use one of: SinglePage, OneColumn, TwoColumnLeft, TwoColumnRight, TwoPageLeft, TwoPageRight",
		cli.SetPageLayoutCommand,
	)
}

func resetDocumentViewCommand(conf *model.Configuration, args []string, command func(string, string, *model.Configuration) *cli.Command) error {
	inFile, outFile, err := optionalOutputPDFArgs(conf, args)
	if err != nil {
		return err
	}
	return runCommand(command(inFile, outFile, conf))
}

func handleResetPageLayoutCommand(conf *model.Configuration, args []string) error {
	return resetDocumentViewCommand(conf, args, cli.ResetPageLayoutCommand)
}

func handleListPageModeCommand(conf *model.Configuration, args []string) error {
	return listSinglePDFCommand(conf, args, cli.ListPageModeCommand)
}

func handleSetPageModeCommand(conf *model.Configuration, args []string) error {
	return setDocumentViewCommand(
		conf,
		args,
		validate.DocumentPageMode,
		"invalid page mode, use one of: UseNone, UseOutlines, UseThumbs, FullScreen, UseOC, UseAttachments",
		cli.SetPageModeCommand,
	)
}

func handleResetPageModeCommand(conf *model.Configuration, args []string) error {
	return resetDocumentViewCommand(conf, args, cli.ResetPageModeCommand)
}

func handleListViewerPreferencesCommand(conf *model.Configuration, args []string, opts *viewerpreferencesListOptions) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	if opts.json {
		log.SetCLILogger(nil)
	}

	return runCommand(cli.ListViewerPreferencesCommand(inFile, opts.all, opts.json, conf))
}

func viewerPreferenceInput(args []string) (string, string) {
	if hasJSONExtension(args[1]) {
		return args[1], ""
	}
	return "", args[1]
}

func handleSetViewerPreferencesCommand(conf *model.Configuration, args []string) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	inFileJSON, stringJSON := viewerPreferenceInput(args)
	inFile, outFile, err := optionalOutputPDFArgs(conf, append([]string{inFile}, args[2:]...))
	if err != nil {
		return err
	}
	return runCommand(cli.SetViewerPreferencesCommand(inFile, inFileJSON, outFile, stringJSON, conf))
}

func handleResetViewerPreferencesCommand(conf *model.Configuration, args []string) error {
	return resetDocumentViewCommand(conf, args, cli.ResetViewerPreferencesCommand)
}

type watermarkOptions struct {
	mode string
}

type stampOptions struct {
	mode string
}

type annotationListOptions struct {
	json bool
}

func annotationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotations",
		Short: "List, remove page annotations",
		Long:  usageLongAnnots,
	}
	addPersistentPasswordFlags(cmd)

	listOpts := &annotationListOptions{json: false}
	listCmd := &cobra.Command{
		Use:   "list inFile",
		Short: "List annotations",
		Args:  cobra.MinimumNArgs(1),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleListAnnotationsCommand(conf, args, listOpts)
		}),
	}
	addSelectedPagesFlag(listCmd)
	listCmd.Flags().BoolVarP(&listOpts.json, "json", "j", listOpts.json, "output JSON")

	removeCmd := &cobra.Command{
		Use:   "remove inFile [ outFile ] [ objNr | annotId | annotType]...",
		Short: "Remove annotations",
		Args:  cobra.MinimumNArgs(1),
		RunE:  wrapHandler(handleRemoveAnnotationsCommand),
	}
	addSelectedPagesFlag(removeCmd)

	cmd.AddCommand(listCmd, removeCmd)

	return cmd
}

func stampCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stamp",
		Short: "Add, remove, update text, image or PDF stamps for selected pages",
		Long:  usageLongStamp,
	}
	addPersistentSelectedPagesFlag(cmd)

	addOpts := &stampOptions{mode: "text"}
	addCmd := &cobra.Command{
		Use:   "add string | file description inFile [ outFile ]",
		Short: "Add stamps",
		Args:  cobra.MinimumNArgs(3),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleAddStampsCommand(conf, args, addOpts)
		}),
	}
	addCmd.Flags().StringVarP(&addOpts.mode, "mode", "m", addOpts.mode, "stamp mode: text | image | pdf")
	addUnitFlag(addCmd)

	updateOpts := &stampOptions{mode: "text"}
	updateCmd := &cobra.Command{
		Use:   "update string | file description inFile [ outFile ]",
		Short: "Update stamps",
		Args:  cobra.RangeArgs(3, 4),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleUpdateStampsCommand(conf, args, updateOpts)
		}),
	}
	updateCmd.Flags().StringVarP(&updateOpts.mode, "mode", "m", updateOpts.mode, "stamp mode: text | image | pdf")
	addUnitFlag(updateCmd)

	removeCmd := &cobra.Command{
		Use:   "remove inFile [ outFile ]",
		Short: "Remove stamps",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleRemoveStampsCommand),
	}

	cmd.AddCommand(addCmd, updateCmd, removeCmd)

	return cmd
}

func watermarkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watermark",
		Short: "Add, remove, update watermarks",
		Long:  usageLongWatermark,
	}
	addPersistentSelectedPagesFlag(cmd)

	addOpts := &watermarkOptions{mode: "text"}
	addCmd := &cobra.Command{
		Use:   "add string | file description inFile [ outFile ]",
		Short: "Add, remove, update text, image or PDF watermarks for selected pages",
		Args:  cobra.MinimumNArgs(3),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleAddWatermarksCommand(conf, args, addOpts)
		}),
	}
	addCmd.Flags().StringVarP(&addOpts.mode, "mode", "m", addOpts.mode, "watermark mode: text | image | pdf")
	addUnitFlag(addCmd)

	updateOpts := &watermarkOptions{mode: "text"}
	updateCmd := &cobra.Command{
		Use:   "update string | file description inFile [ outFile ]",
		Short: "Update watermarks",
		Args:  cobra.MinimumNArgs(3),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleUpdateWatermarksCommand(conf, args, updateOpts)
		}),
	}
	updateCmd.Flags().StringVarP(&updateOpts.mode, "mode", "m", updateOpts.mode, "watermark mode: text|image|pdf")
	addUnitFlag(updateCmd)

	removeCmd := &cobra.Command{
		Use:   "remove inFile [ outFile ]",
		Short: "Remove watermarks",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleRemoveWatermarksCommand),
	}

	cmd.AddCommand(addCmd, updateCmd, removeCmd)

	return cmd
}

func validateWatermarkMode(wmMode string) error {
	if wmMode != "text" && wmMode != "image" && wmMode != "pdf" {
		return errors.New("mode must be one of: image, pdf, text")
	}
	return nil
}

func parseWatermark(args []string, onTop bool, wmMode string, unit types.DisplayUnit) (*model.Watermark, error) {
	switch wmMode {
	case "text":
		if err := pdfcpu.ValidateWatermarkModeParam(model.WMText, args[0], onTop); err != nil {
			return nil, err
		}
		return pdfcpu.ParseTextWatermarkDetails(args[0], args[1], onTop, unit)
	case "image":
		if err := pdfcpu.ValidateWatermarkModeParam(model.WMImage, args[0], onTop); err != nil {
			return nil, err
		}
		return pdfcpu.ParseImageWatermarkDetails(args[0], args[1], onTop, unit)
	case "pdf":
		if err := pdfcpu.ValidateWatermarkModeParam(model.WMPDF, args[0], onTop); err != nil {
			return nil, err
		}
		return pdfcpu.ParsePDFWatermarkDetails(args[0], args[1], onTop, unit)
	}
	return nil, fmt.Errorf("unsupported wm type: %s", wmMode)
}

func watermarkCommand(conf *model.Configuration, args []string, onTop bool, wmMode string, update bool) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}
	if err := validateWatermarkMode(wmMode); err != nil {
		return err
	}

	wm, err := parseWatermark(args, onTop, wmMode, conf.Unit)
	if err != nil {
		return err
	}
	wm.Update = update

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	inFile, outFile, err := optionalOutputPDFArgs(conf, args[2:])
	if err != nil {
		return err
	}

	return runCommand(cli.AddWatermarksCommand(inFile, outFile, selectedPages, wm, conf))
}

func addWatermarks(conf *model.Configuration, args []string, onTop bool, wmMode string) error {
	return watermarkCommand(conf, args, onTop, wmMode, false)
}

func handleAddStampsCommand(conf *model.Configuration, args []string, opts *stampOptions) error {
	return addWatermarks(conf, args, true, opts.mode)
}

func handleAddWatermarksCommand(conf *model.Configuration, args []string, opts *watermarkOptions) error {
	return addWatermarks(conf, args, false, opts.mode)
}

func updateWatermarks(conf *model.Configuration, args []string, onTop bool, wmMode string) error {
	return watermarkCommand(conf, args, onTop, wmMode, true)
}

func handleUpdateStampsCommand(conf *model.Configuration, args []string, opts *stampOptions) error {
	return updateWatermarks(conf, args, true, opts.mode)
}

func handleUpdateWatermarksCommand(conf *model.Configuration, args []string, opts *watermarkOptions) error {
	return updateWatermarks(conf, args, false, opts.mode)
}

func removeWatermarks(conf *model.Configuration, args []string, onTop bool) error {
	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}
	inFile, outFile, err := optionalOutputPDFArgs(conf, args)
	if err != nil {
		return err
	}

	return runCommand(cli.RemoveWatermarksCommand(inFile, outFile, selectedPages, conf))
}

func handleRemoveStampsCommand(conf *model.Configuration, args []string) error {
	return removeWatermarks(conf, args, true)
}

func handleRemoveWatermarksCommand(conf *model.Configuration, args []string) error {
	return removeWatermarks(conf, args, false)
}

func handleListAnnotationsCommand(conf *model.Configuration, args []string, opts *annotationListOptions) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	if opts.json {
		return runCommand(cli.ListAnnotationsJSONCommand(inFile, selectedPages, conf))
	}
	return runCommand(cli.ListAnnotationsCommand(inFile, selectedPages, conf))
}

func annotationRemovalArgs(conf *model.Configuration, args []string) (string, string, []string, []int, error) {
	var idsAndTypes []string
	var objNrs []int

	for i, arg := range args {
		if i == 0 {
			if err := inputPDFArg(conf, arg); err != nil {
				return "", "", nil, nil, err
			}
			continue
		}
		if i == 1 {
			if hasPDFExtension(arg) || arg == "-" {
				if err := ensureOutputFileAvailable(arg); err != nil {
					return "", "", nil, nil, err
				}
				continue
			}
		}

		j, err := strconv.Atoi(arg)
		if err != nil {
			// strings args may be an id or annotType
			if err := validateNoEmptyArgs([]string{arg}, "annotation ID or type"); err != nil {
				return "", "", nil, nil, err
			}
			idsAndTypes = append(idsAndTypes, arg)
			continue
		}
		objNrs = append(objNrs, j)
	}

	return args[0], annotationOutFile(args), idsAndTypes, objNrs, nil
}

func annotationOutFile(args []string) string {
	if len(args) > 1 && (hasPDFExtension(args[1]) || args[1] == "-") {
		return args[1]
	}
	return ""
}

func handleRemoveAnnotationsCommand(conf *model.Configuration, args []string) error {
	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	inFile, outFile, idsAndTypes, objNrs, err := annotationRemovalArgs(conf, args)
	if err != nil {
		return err
	}

	return runCommand(cli.RemoveAnnotationsCommand(inFile, outFile, selectedPages, idsAndTypes, objNrs, conf))
}

func handleListBookmarksCommand(conf *model.Configuration, args []string) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	return runCommand(cli.ListBookmarksCommand(inFile, conf))
}

func handleExportBookmarksCommand(conf *model.Configuration, args []string) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	outFileJSON := "out.json"
	if len(args) == 2 {
		outFileJSON = args[1]
		if outFileJSON != "-" {
			if err := ensureJSONExtension(outFileJSON); err != nil {
				return err
			}
		}
	}
	if outFileJSON != "-" {
		if err := ensureOutputFileAvailable(outFileJSON); err != nil {
			return err
		}
	}

	return runCommand(cli.ExportBookmarksCommand(inFile, outFileJSON, conf))
}

func handleImportBookmarksCommand(conf *model.Configuration, args []string, opts *bookmarksImportOptions) error {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	inFileJSON := args[1]
	if err := ensureJSONExtension(inFileJSON); err != nil {
		return err
	}

	outFile := ""
	if inFile == "-" {
		outFile = "-"
	}
	if len(args) == 3 {
		outFile = args[2]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return err
			}
		}
		if err := ensureOutputFileAvailable(outFile); err != nil {
			return err
		}
	}

	return runCommand(cli.ImportBookmarksCommand(inFile, inFileJSON, outFile, opts.replaceBookmarks, conf))
}

func handleRemoveBookmarksCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, err := optionalOutputPDFArgs(conf, args)
	if err != nil {
		return err
	}
	return runCommand(cli.RemoveBookmarksCommand(inFile, outFile, conf))
}

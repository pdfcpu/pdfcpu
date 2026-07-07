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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/spf13/cobra"
)

type pagesInsertOptions struct {
	mode string
}

func bookletCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "booklet [ description ] outFile n inFile | imageFiles...",
		Short: "Arrange pages onto larger sheets of paper to make a booklet or zine",
		Long:  usageLongBooklet,
		Args:  cobra.MinimumNArgs(3),
		RunE:  wrapHandler(handleBookletCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func cutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cut description inFile outDir [ outFile ]",
		Short: "Custom cut pages horizontally or vertically",
		Long:  usageLongCut,
		Args:  cobra.RangeArgs(3, 4),
		RunE:  wrapHandler(handleCutCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func gridCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grid [ description ] outFile m n inFile | imageFiles...",
		Short: "Rearrange pages or images for enhanced browsing experience",
		Long:  usageLongGrid,
		Args:  cobra.MinimumNArgs(4),
		RunE:  wrapHandler(handleGridCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func ndownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ndown [ description ] n inFile outDir [ outFile ]",
		Short: "Cut selected page into n pages symmetrically",
		Long:  usageLongNDown,
		Args:  cobra.RangeArgs(3, 5),
		RunE:  wrapHandler(handleNDownCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func nupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nup [ description ] outFile n inFile | imageFiles...",
		Short: "Rearrange pages or images for reduced number of pages",
		Long:  usageLongNUp,
		Args:  cobra.MinimumNArgs(3),
		RunE:  wrapHandler(handleNUpCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func posterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poster description inFile outDir [ outFile]",
		Short: "Create poster using paper size",
		Long:  usageLongPoster,
		Args:  cobra.RangeArgs(3, 4),
		RunE:  wrapHandler(handlePosterCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func resizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resize description inFile [ outFile ]",
		Short: "Scale selected pages",
		Long:  usageLongResize,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleResizeCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func zoomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zoom description inFile [outFile]",
		Short: "Zoom in/out of selected pages",
		Long:  usageLongZoom,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleZoomCommand),
	}
	addSelectedPagesUnitPasswordFlags(cmd)

	return cmd
}

func boxesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boxes",
		Short: "List, add, remove page boundaries for selected pages",
		Long:  usageLongBoxes,
	}
	addPersistentPasswordFlags(cmd)
	addPersistentSelectedPagesFlag(cmd)

	listCmd := &cobra.Command{
		Use:   "list [ boxTypes ] inFile",
		Short: "List boxes",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleListBoxesCommand),
	}
	addUnitFlag(listCmd)

	addCmd := &cobra.Command{
		Use:   "add description inFile [ outFile ]",
		Short: "Add boxes",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleAddBoxesCommand),
	}
	addUnitFlag(addCmd)

	removeCmd := &cobra.Command{
		Use:   "remove boxTypes inFile [ outFile ]",
		Short: "Remove boxes",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleRemoveBoxesCommand),
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd)

	return cmd
}

func parseForGrid(args []string, nup *model.NUp, argInd *int) error {
	if *argInd < 0 || len(args)-*argInd < 2 {
		return errors.New("missing grid dimensions")
	}
	rows, err := strconv.Atoi(args[*argInd])
	if err != nil {
		return fmt.Errorf("parse grid rows %q: %w", args[*argInd], err)
	}
	cols, err := strconv.Atoi(args[*argInd+1])
	if err != nil {
		return fmt.Errorf("parse grid columns %q: %w", args[*argInd+1], err)
	}
	if err = api.ParseGridDefinition(rows, cols, nup); err != nil {
		return fmt.Errorf("parse grid dimensions: %w", err)
	}
	*argInd += 2
	return nil
}

func nUpValueError(nUpValues []int) error {
	ss := make([]string, len(nUpValues))
	for i, v := range nUpValues {
		ss[i] = strconv.Itoa(v)
	}
	return fmt.Errorf("n must be one of %s", strings.Join(ss, ", "))
}

func parseForNUp(args []string, nup *model.NUp, argInd *int, nUpValues []int) error {
	n, err := strconv.Atoi(args[*argInd])
	if err != nil {
		return err
	}
	if !types.IntMemberOf(n, nUpValues) {
		return nUpValueError(nUpValues)
	}
	if err = api.ParseNUpValue(n, nup); err != nil {
		return err
	}
	*argInd++
	return nil
}

func validateNUpInputFile(filenameIn string, allowStdin bool) error {
	if filenameIn == "-" && !allowStdin {
		return fmt.Errorf("inFile has to be a PDF or one or a sequence of image files: %s", filenameIn)
	}
	if filenameIn != "-" && !hasPDFExtension(filenameIn) && !model.ImageFileName(filenameIn) {
		return fmt.Errorf("inFile has to be a PDF or one or a sequence of image files: %s", filenameIn)
	}
	return nil
}

func appendNUpImageFiles(args []string, startInd int, filenamesIn []string) ([]string, error) {
	for i := startInd; i < len(args); i++ {
		arg := args[i]
		if err := ensureImageExtension(arg); err != nil {
			return nil, err
		}
		filenamesIn = append(filenamesIn, arg)
	}
	return filenamesIn, nil
}

func parseAfterNUpDetails(args []string, nup *model.NUp, argInd int, nUpValues []int, filenameOut string, allowStdin bool) ([]string, error) {
	if nup.PageGrid {
		if err := parseForGrid(args, nup, &argInd); err != nil {
			return nil, err
		}
	} else {
		if err := parseForNUp(args, nup, &argInd, nUpValues); err != nil {
			return nil, err
		}
	}
	if argInd >= len(args) {
		return nil, errors.New("missing input file")
	}

	filenameIn := args[argInd]
	if err := validateNUpInputFile(filenameIn, allowStdin); err != nil {
		return nil, err
	}

	filenamesIn := []string{filenameIn}

	if filenameIn == "-" || hasPDFExtension(filenameIn) {
		if len(args) > argInd+1 {
			return nil, errors.New("too many args")
		}
		if filenameIn != "-" && filenameIn == filenameOut {
			return nil, errors.New("inFile and outFile can't be the same.")
		}
	} else {
		nup.ImgInputFile = true
		return appendNUpImageFiles(args, argInd+1, filenamesIn)
	}

	return filenamesIn, nil
}

func nupOutFileAndArgIndex(args []string, nup *model.NUp) (string, int, error) {
	outFile := args[0]
	argInd := 1
	if outFile != "-" && !hasPDFExtension(outFile) {
		if err := api.ParseNUpDetails(args[0], nup); err != nil {
			return "", 0, err
		}
		outFile = args[1]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return "", 0, err
			}
		}
		argInd = 2
	}
	if err := ensureOutputFileAvailable(outFile); err != nil {
		return "", 0, err
	}
	return outFile, argInd, nil
}

func nupFilesAndConfig(args []string, nup *model.NUp, nUpValues []int) ([]string, string, error) {
	outFile, argInd, err := nupOutFileAndArgIndex(args, nup)
	if err != nil {
		return nil, "", err
	}
	inFiles, err := parseAfterNUpDetails(args, nup, argInd, nUpValues, outFile, true)
	if err != nil {
		return nil, "", err
	}
	return inFiles, outFile, nil
}

func handleNUpCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}

	pages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit

	inFiles, outFile, err := nupFilesAndConfig(args, nup, api.NUpValues())
	if err != nil {
		return err
	}
	return runCommand(cli.NUpCommand(inFiles, outFile, pages, nup, conf))
}

func handleGridCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return fmt.Errorf("grid: configure display unit: %w", err)
	}

	pages, err := parseSelectedPages()
	if err != nil {
		return fmt.Errorf("grid: parse page selection: %w", err)
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit
	nup.PageGrid = true

	inFiles, outFile, err := nupFilesAndConfig(args, nup, nil)
	if err != nil {
		return fmt.Errorf("grid: parse arguments: %w", err)
	}
	return runCommand(cli.GridCommand(inFiles, outFile, pages, nup, conf))
}

func handleBookletCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}

	pages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	nup := api.DefaultBookletConfig()
	nup.InpUnit = conf.Unit

	inFiles, outFile, err := nupFilesAndConfig(args, nup, api.NUpValuesForBooklets())
	if err != nil {
		return err
	}
	return runCommand(cli.BookletCommand(inFiles, outFile, pages, nup, conf))
}

func handleResizeCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingResizeConfiguration
	}
	if len(args) == 1 {
		return api.ErrMissingPDFInput
	}
	if err := configureDisplayUnit(conf); err != nil {
		return fmt.Errorf("resize: configure display unit: %w", err)
	}
	rc, err := pdfcpu.ParseResizeConfig(args[0], conf.Unit)
	if err != nil {
		return fmt.Errorf("resize: parse configuration: %w", err)
	}

	inFile, outFile, err := optionalOutputPDFArgs(conf, args[1:])
	if err != nil {
		return fmt.Errorf("resize: parse arguments: %w", err)
	}

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return fmt.Errorf("resize: parse page selection: %w", err)
	}

	return runCommand(cli.ResizeCommand(inFile, outFile, selectedPages, rc, conf))
}

func handlePosterCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingCutConfiguration
	}
	if len(args) == 1 {
		return api.ErrMissingPDFInput
	}
	if len(args) == 2 {
		return api.ErrMissingPDFOutput
	}
	if err := configureDisplayUnit(conf); err != nil {
		return fmt.Errorf("poster: configure display unit: %w", err)
	}
	// formsize(=papersize) or dimensions, optionally: scalefactor, border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForPoster(args[0], conf.Unit)
	if err != nil {
		return fmt.Errorf("poster: parse configuration: %w", err)
	}

	inFile := args[1]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("poster: check input: %w", err)
	}

	outDir := args[2]

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return fmt.Errorf("poster: parse page selection: %w", err)
	}

	var outFile string
	if len(args) == 4 {
		outFile = args[3]
	}
	if err := ensureOutputDirOrFileAvailable(outDir, outFile); err != nil {
		return fmt.Errorf("poster: check output: %w", err)
	}

	return runCommand(cli.PosterCommand(inFile, outDir, outFile, selectedPages, cut, conf))
}

func ndownArgs(args []string, unit types.DisplayUnit) (int, *model.Cut, string, string, string, error) {
	if len(args) == 0 {
		return 0, nil, "", "", "", api.ErrMissingCutConfiguration
	}
	if len(args) == 1 {
		return 0, nil, "", "", "", api.ErrMissingPDFInput
	}
	if len(args) == 2 {
		return 0, nil, "", "", "", api.ErrMissingPDFOutput
	}
	n, err := strconv.Atoi(args[0])
	if err == nil {
		cut, err := pdfcpu.ParseCutConfigForN(n, "", unit)
		if err != nil {
			return 0, nil, "", "", "", fmt.Errorf("parse configuration: %w", err)
		}
		var outFile string
		if len(args) == 4 {
			outFile = args[3]
		}
		return n, cut, args[1], args[2], outFile, nil
	}

	n, err = strconv.Atoi(args[1])
	if err != nil {
		return 0, nil, "", "", "", fmt.Errorf("parse n-down value %q: %w", args[1], err)
	}
	cut, err := pdfcpu.ParseCutConfigForN(n, args[0], unit)
	if err != nil {
		return 0, nil, "", "", "", fmt.Errorf("parse configuration: %w", err)
	}
	var outFile string
	if len(args) == 5 {
		outFile = args[4]
	}
	return n, cut, args[2], args[3], outFile, nil
}

func handleNDownCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingCutConfiguration
	}
	if len(args) == 1 {
		return api.ErrMissingPDFInput
	}
	if len(args) == 2 {
		return api.ErrMissingPDFOutput
	}
	if err := configureDisplayUnit(conf); err != nil {
		return fmt.Errorf("ndown: configure display unit: %w", err)
	}

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return fmt.Errorf("ndown: parse page selection: %w", err)
	}

	n, cut, inFile, outDir, outFile, err := ndownArgs(args, conf.Unit)
	if err != nil {
		return fmt.Errorf("ndown: parse arguments: %w", err)
	}
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("ndown: check input: %w", err)
	}
	if err := ensureOutputDirOrFileAvailable(outDir, outFile); err != nil {
		return fmt.Errorf("ndown: check output: %w", err)
	}

	return runCommand(cli.NDownCommand(inFile, outDir, outFile, selectedPages, n, cut, conf))
}

func handleCutCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingCutConfiguration
	}
	if len(args) == 1 {
		return api.ErrMissingPDFInput
	}
	if len(args) == 2 {
		return api.ErrMissingPDFOutput
	}
	if err := configureDisplayUnit(conf); err != nil {
		return fmt.Errorf("cut: configure display unit: %w", err)
	}
	// required: at least one of horizontalCut, verticalCut
	// optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfig(args[0], conf.Unit)
	if err != nil {
		return fmt.Errorf("cut: parse configuration: %w", err)
	}

	inFile := args[1]
	if err := inputPDFArg(conf, inFile); err != nil {
		return fmt.Errorf("cut: check input: %w", err)
	}

	outDir := args[2]

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return fmt.Errorf("cut: parse page selection: %w", err)
	}

	var outFile string
	if len(args) >= 4 {
		outFile = args[3]
	}
	if err := ensureOutputDirOrFileAvailable(outDir, outFile); err != nil {
		return fmt.Errorf("cut: check output: %w", err)
	}

	return runCommand(cli.CutCommand(inFile, outDir, outFile, selectedPages, cut, conf))
}

func handleZoomCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingZoomConfiguration
	}
	if len(args) == 1 {
		return api.ErrMissingPDFInput
	}
	if err := configureDisplayUnit(conf); err != nil {
		return fmt.Errorf("zoom: configure display unit: %w", err)
	}
	zc, err := pdfcpu.ParseZoomConfig(args[0], conf.Unit)
	if err != nil {
		return fmt.Errorf("zoom: parse configuration: %w", err)
	}

	inFile, outFile, err := optionalOutputPDFArgs(conf, args[1:])
	if err != nil {
		return fmt.Errorf("zoom: parse arguments: %w", err)
	}

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return fmt.Errorf("zoom: parse page selection: %w", err)
	}

	return runCommand(cli.ZoomCommand(inFile, outFile, selectedPages, zc, conf))
}

func addSelectedPagesUnitPasswordFlags(cmd *cobra.Command) {
	addSelectedPagesFlag(cmd)
	addUnitFlag(cmd)
	addPasswordFlags(cmd)
}

func cropCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crop description inFile [ outFile ]",
		Short: "Set cropbox for selected pages",
		Long:  usageLongCrop,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleCropCommand),
	}
	addSelectedPagesFlag(cmd)
	addUnitFlag(cmd)
	addPasswordFlags(cmd)

	return cmd
}

func pagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pages",
		Short: "Insert, remove selected pages",
		Long:  usageLongPages,
	}
	addPersistentPasswordFlags(cmd)

	insertOpts := &pagesInsertOptions{mode: "before"}
	insertCmd := &cobra.Command{
		Use:   "insert [ description ] inFile [ outFile ]",
		Short: "Insert pages",
		Args:  cobra.RangeArgs(1, 3),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleInsertPagesCommand(conf, args, insertOpts)
		}),
	}
	addRequiredSelectedPagesFlag(insertCmd)
	insertCmd.Flags().StringVarP(&insertOpts.mode, "mode", "m", insertOpts.mode, "insertion mode: before|after")
	addUnitFlag(insertCmd)

	removeCmd := &cobra.Command{
		Use:   "remove inFile [ outFile ]",
		Short: "Remove pages",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleRemovePagesCommand),
	}
	addRequiredSelectedPagesFlag(removeCmd)

	cmd.AddCommand(insertCmd, removeCmd)

	return cmd
}

func rotateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate inFile rotation [outFile]",
		Short: "Rotate selected pages",
		Long:  usageLongRotate,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleRotateCommand),
	}
	addSelectedPagesFlag(cmd)
	addPasswordFlags(cmd)
	return cmd
}

func insertPagesWithoutDesc(inFile string, conf *model.Configuration, pages []string, args []string, opts *pagesInsertOptions) error {
	outFile := ""
	if inFile == "-" {
		outFile = "-"
	}
	if len(args) == 2 {
		outFile = args[1]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return err
			}
		}
		if err := ensureOutputFileAvailable(outFile); err != nil {
			return err
		}
	}

	return runCommand(cli.InsertPagesCommand(inFile, outFile, pages, conf, opts.mode, nil))
}

func selectedPagesRequired() ([]string, error) {
	pages, err := parseSelectedPages()
	if err != nil {
		return nil, err
	}
	if pages == nil {
		return nil, errors.New("missing page selection")
	}
	return pages, nil
}

func validatePagesInsertMode(opts *pagesInsertOptions) error {
	if opts.mode != "" && opts.mode != "before" && opts.mode != "after" {
		return errors.New("mode must be one of: before, after")
	}
	return nil
}

func pagesInsertWithDesc(conf *model.Configuration, args []string, pages []string, opts *pagesInsertOptions) error {
	pageConf, err := pdfcpu.ParsePageConfiguration(args[0], conf.Unit)
	if err != nil {
		return err
	}
	if pageConf == nil {
		return errors.New("missing page configuration")
	}

	inFile, outFile, err := optionalOutputPDFArgs(conf, args[1:])
	if err != nil {
		return err
	}

	return runCommand(cli.InsertPagesCommand(inFile, outFile, pages, conf, opts.mode, pageConf))
}

func handleInsertPagesCommand(conf *model.Configuration, args []string, opts *pagesInsertOptions) error {
	pages, err := selectedPagesRequired()
	if err != nil {
		return err
	}
	if err := validatePagesInsertMode(opts); err != nil {
		return err
	}

	inFile := args[0]
	if hasPDFExtension(inFile) || inFile == "-" {
		return insertPagesWithoutDesc(inFile, conf, pages, args, opts)
	}

	return pagesInsertWithDesc(conf, args, pages, opts)
}

func handleRemovePagesCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, err := optionalOutputPDFArgs(conf, args)
	if err != nil {
		return err
	}

	pages, err := selectedPagesRequired()
	if err != nil {
		return err
	}

	return runCommand(cli.RemovePagesCommand(inFile, outFile, pages, conf))
}

func rotation(s string) (int, error) {
	rotation, err := strconv.Atoi(s)
	if err != nil || rotation%90 != 0 {
		return 0, fmt.Errorf("rotation must be a multiple of 90: %s", s)
	}
	return rotation, nil
}

func handleRotateCommand(conf *model.Configuration, args []string) error {
	rotation, err := rotation(args[1])
	if err != nil {
		return err
	}
	inFile, outFile, pages, err := selectedPagesPDFArgs(conf, append([]string{args[0]}, args[2:]...))
	if err != nil {
		return err
	}
	return runCommand(cli.RotateCommand(inFile, outFile, rotation, pages, conf))
}

func handleCropCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}
	box, err := api.Box(args[0], conf.Unit)
	if err != nil {
		return fmt.Errorf("crop: %w", err)
	}
	inFile, outFile, pages, err := selectedPagesPDFArgs(conf, args[1:])
	if err != nil {
		return err
	}
	return runCommand(cli.CropCommand(inFile, outFile, pages, box, conf))
}

func handleListBoxesCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}
	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}
	if len(args) == 1 {
		inFile := args[0]
		if err := inputPDFArg(conf, inFile); err != nil {
			return err
		}
		return runCommand(cli.ListBoxesCommand(inFile, selectedPages, nil, conf))
	}

	pb, err := api.PageBoundariesFromBoxList(args[0])
	if err != nil {
		return fmt.Errorf("list boxes: %w", err)
	}

	inFile := args[1]
	if err := inputPDFArg(conf, inFile); err != nil {
		return err
	}

	return runCommand(cli.ListBoxesCommand(inFile, selectedPages, pb, conf))
}

func handleAddBoxesCommand(conf *model.Configuration, args []string) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}
	pb, err := api.PageBoundaries(args[0], conf.Unit)
	if err != nil {
		return fmt.Errorf("add boxes: %w", err)
	}

	inFile, outFile, err := optionalOutputPDFArgs(conf, args[1:])
	if err != nil {
		return err
	}

	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	return runCommand(cli.AddBoxesCommand(inFile, outFile, selectedPages, pb, conf))
}

func removeBoxBoundaries(s string) (*model.PageBoundaries, error) {
	pb, err := api.PageBoundariesFromBoxList(s)
	if err != nil {
		return nil, fmt.Errorf("remove boxes: %w", err)
	}
	if pb == nil {
		pb = &model.PageBoundaries{}
	}
	return pb, nil
}

func handleRemoveBoxesCommand(conf *model.Configuration, args []string) error {
	pb, err := removeBoxBoundaries(args[0])
	if err != nil {
		return err
	}
	inFile, outFile, err := optionalOutputPDFArgs(conf, args[1:])
	if err != nil {
		return err
	}
	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	return runCommand(cli.RemoveBoxesCommand(inFile, outFile, selectedPages, pb, conf))
}

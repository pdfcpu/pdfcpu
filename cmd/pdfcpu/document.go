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
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	mode     string
	links    bool
	optimize bool
}

type optimizeCommandOptions struct {
	fileStats string
}

type infoOptions struct {
	fonts bool
	json  bool
}

type splitOptions struct {
	mode string
}

type mergeOptions struct {
	mode            string
	bookmarkMode    string
	bookmarks       bool
	dividerPage     bool
	optimize        bool
	sorted          bool
	bookmarksSet    bool
	bookmarkModeSet bool
	optimizeSet     bool
}

func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create inFileJSON [ inFile ] outFile",
		Short: "Create PDF content including forms via JSON",
		Long:  usageLongCreate,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  wrapHandler(handleCreateCommand),
	}
}

func dumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "dump a|h obj# inFile",
		Short:  "Dump object",
		Args:   cobra.ExactArgs(3),
		Hidden: true,
		RunE:   wrapHandler(handleDumpCommand),
	}
}

func infoCmd() *cobra.Command {
	opts := &infoOptions{fonts: false, json: false}
	cmd := &cobra.Command{
		Use:   "info inFile...",
		Short: "Print file info",
		Long:  usageLongInfo,
		Args:  cobra.MinimumNArgs(1),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleInfoCommand(conf, args, opts)
		}),
	}
	addSelectedPagesFlag(cmd)
	cmd.Flags().BoolVar(&opts.fonts, "fonts", opts.fonts, "include font info")
	cmd.Flags().BoolVarP(&opts.json, "json", "j", opts.json, "output JSON")
	addUnitFlag(cmd)
	addPasswordFlags(cmd)

	return cmd
}

func collectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect inFile [ outFile ]",
		Short: "Create custom sequence of selected pages",
		Long:  usageLongCollect,
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleCollectCommand),
	}
	addRequiredSelectedPagesFlag(cmd)
	addPasswordFlags(cmd)

	return cmd
}

func mergeCmd() *cobra.Command {
	opts := &mergeOptions{
		mode:         "create",
		bookmarkMode: string(model.MergeBookmarkModeWrap),
		sorted:       false,
		bookmarks:    false,
		dividerPage:  false,
		optimize:     false,
	}

	cmd := &cobra.Command{
		Use:   "merge outFile inFile...",
		Short: "Concatenate PDFs",
		Long:  usageLongMerge,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.bookmarksSet = cmd.Flags().Changed("bookmarks")
			opts.bookmarkModeSet = cmd.Flags().Changed("bookmark-mode")
			opts.optimizeSet = cmd.Flags().Changed("optimize")
			return wrapHandler(func(conf *model.Configuration, args []string) error {
				return handleMergeCommand(conf, args, opts)
			})(cmd, args)
		},
	}

	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "merge mode: create|append|zip")
	cmd.Flags().BoolVarP(&opts.sorted, "sort", "s", opts.sorted, "sort inFiles by file name")
	cmd.Flags().BoolVarP(&opts.bookmarks, "bookmarks", "b", opts.bookmarks, "create bookmarks")
	cmd.Flags().StringVar(&opts.bookmarkMode, "bookmark-mode", opts.bookmarkMode, "bookmark mode: wrap|preserve")
	cmd.Flags().BoolVarP(&opts.dividerPage, "divider", "d", opts.dividerPage, "insert blank page between merged documents")
	cmd.Flags().BoolVar(&opts.optimize, "optimize", opts.optimize, "optimize before writing")
	cmd.Flags().BoolVar(&opts.optimize, "opt", opts.optimize, "optimize before writing")
	cmd.Flags().BoolVar(&removeSignatures, "rmsig", false, "remove signatures")

	return cmd
}

func splitCmd() *cobra.Command {
	opts := &splitOptions{
		mode: "span",
	}
	cmd := &cobra.Command{
		Use:   "split inFile outDir [ span | pageNr... ]",
		Short: "Split up inFile by span or bookmark",
		Long:  usageLongSplit,
		Args:  cobra.MinimumNArgs(2),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleSplitCommand(conf, args, opts)
		}),
	}
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "split mode: span | bookmark | page")
	addPasswordFlags(cmd)
	return cmd
}

func trimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trim inFile [outFile]",
		Short: "Create trimmed version of selected pages",
		Long:  usageLongTrim,
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleTrimCommand),
	}
	addPasswordFlags(cmd)
	addRequiredSelectedPagesFlag(cmd)
	return cmd
}

func optimizeCmd() *cobra.Command {
	opts := &optimizeCommandOptions{fileStats: ""}
	cmd := &cobra.Command{
		Use:   "optimize inFile [ outFile ]",
		Short: "Optimize a PDF by getting rid of redundant page resources",
		Long:  usageLongOptimize,
		Args:  cobra.RangeArgs(1, 2),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleOptimizeCommand(conf, args, opts)
		}),
	}
	addPasswordFlags(cmd)
	cmd.Flags().BoolVar(&removeEncryption, "rmenc", false, "remove encryption")
	cmd.Flags().BoolVar(&removeSignatures, "rmsig", false, "remove signatures")
	cmd.Flags().StringVar(&opts.fileStats, "stats", opts.fileStats, "appends a stats line to a csv file")

	return cmd
}

func validateCmd() *cobra.Command {
	opts := &validateOptions{
		mode:     "relaxed",
		links:    false,
		optimize: false,
	}
	cmd := &cobra.Command{
		Use:   "validate inFile...",
		Short: "Validate PDF against PDF 32000-1:2008 (PDF 1.7) + basic PDF 2.0 validation",
		Long:  usageLongValidate,
		Args:  cobra.MinimumNArgs(1),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleValidateCommand(conf, args, opts)
		}),
	}
	addPasswordFlags(cmd)
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "validation mode: strict|relaxed")
	cmd.Flags().BoolVarP(&opts.links, "links", "l", opts.links, "check for broken links")
	cmd.Flags().BoolVar(&opts.optimize, "optimize", opts.optimize, "optimize resources")
	cmd.Flags().BoolVar(&opts.optimize, "opt", opts.optimize, "optimize resources")
	return cmd
}

func handleValidateCommand(conf *model.Configuration, args []string, opts *validateOptions) error {
	inFiles, err := collectInFiles(conf, args)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	if len(inFiles) == 0 {
		return fmt.Errorf("validate: %w", api.ErrMissingPDFInput)
	}

	switch opts.mode {
	case "strict", "s":
		conf.ValidationMode = model.ValidationStrict
	case "relaxed", "r":
		conf.ValidationMode = model.ValidationRelaxed
	case "":
		conf.ValidationMode = model.ValidationRelaxed
	default:
		return errors.New("mode must be one of: r(elaxed), s(trict)")
	}

	if opts.links {
		conf.ValidateLinks = true
	}

	conf.Optimize = opts.optimize

	return runCommand(cli.ValidateCommand(inFiles, conf))
}

func handleOptimizeCommand(conf *model.Configuration, args []string, opts *optimizeCommandOptions) error {
	inFile := args[0]
	if conf.CheckFileNameExt && inFile != "-" {
		if err := ensurePDFExtension(inFile); err != nil {
			return err
		}
	}

	outFile := inFile
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

	conf.StatsFileName = opts.fileStats
	if len(opts.fileStats) > 0 {
		fmt.Fprintf(os.Stderr, "stats will be appended to %s\n", opts.fileStats)
	}

	return runCommand(cli.OptimizeCommand(inFile, outFile, conf))
}

func infoInputFiles(conf *model.Configuration, args []string) ([]string, error) {
	var inFiles []string
	for _, arg := range args {
		files, err := infoInputFile(conf, arg)
		if err != nil {
			return nil, err
		}
		inFiles = append(inFiles, files...)
	}
	return inFiles, nil
}

func infoInputFile(conf *model.Configuration, arg string) ([]string, error) {
	if arg == "-" {
		return []string{arg}, nil
	}
	if strings.Contains(arg, "*") {
		return filepath.Glob(arg)
	}
	if conf.CheckFileNameExt {
		if err := ensurePDFExtension(arg); err != nil {
			return nil, err
		}
	}
	return []string{arg}, nil
}

func handleInfoCommand(conf *model.Configuration, args []string, opts *infoOptions) error {
	if err := configureDisplayUnit(conf); err != nil {
		return err
	}

	inFiles, err := infoInputFiles(conf, args)
	if err != nil {
		return err
	}
	selectedPages, err := parseSelectedPages()
	if err != nil {
		return err
	}

	if opts.json {
		log.SetCLILogger(nil)
	}

	return runCommand(cli.InfoCommand(inFiles, selectedPages, opts.fonts, opts.json, conf))
}

func dumpMode(mode string) []int {
	vals := []int{0, 0}
	switch strings.ToLower(mode)[0] {
	case 'a':
		vals[0] = 1
	case 'h':
		vals[0] = 2
	}
	return vals
}

func handleDumpCommand(conf *model.Configuration, args []string) error {
	vals := dumpMode(args[0])
	objNr, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid dump object number %q: %w", args[1], err)
	}
	vals[1] = objNr

	inFile := args[2]
	if err := ensurePDFExtension(inFile); err != nil {
		return err
	}

	conf.ValidationMode = model.ValidationRelaxed

	return runCommand(cli.DumpCommand(inFile, vals, conf))
}

func sortFiles(inFiles []string) {
	re := regexp.MustCompile(`\d+`)

	sort.Slice(
		inFiles,
		func(i, j int) bool {
			ssi := re.FindAllString(inFiles[i], 1)
			ssj := re.FindAllString(inFiles[j], 1)
			if len(ssi) == 0 || len(ssj) == 0 {
				return inFiles[i] <= inFiles[j]
			}
			i1, _ := strconv.Atoi(ssi[0])
			i2, _ := strconv.Atoi(ssj[0])
			return i1 < i2
		})
}

func processArgsForMerge(conf *model.Configuration, args []string, mergeMode string) ([]string, string, error) {
	inFiles := []string{}
	outFile := ""
	for i, arg := range args {
		if i == 0 {
			if arg != "-" {
				if err := ensurePDFExtension(arg); err != nil {
					return nil, "", err
				}
			}
			outFile = arg
			continue
		}
		if arg == outFile && arg != "-" {
			return nil, "", fmt.Errorf("%s may appear as inFile or outFile only", outFile)
		}
		if mergeMode != "zip" && strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				return nil, "", err
			}
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt && arg != "-" {
			if err := ensurePDFExtension(arg); err != nil {
				return nil, "", err
			}
		}
		inFiles = append(inFiles, arg)
	}
	return inFiles, outFile, nil
}

func mergeCommandVariation(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration, mergeMode string) *cli.Command {
	switch mergeMode {
	case "create":
		return cli.MergeCreateCommand(inFiles, outFile, dividerPage, conf)
	case "zip":
		return cli.MergeCreateZipCommand(inFiles, outFile, conf)
	case "append":
		return cli.MergeAppendCommand(inFiles, outFile, dividerPage, conf)
	}
	return nil
}

func mergeMode(mode string) (string, error) {
	if mode == "" {
		mode = "create"
	}
	mode = modeCompletion(mode, []string{"create", "append", "zip"})
	if mode == "" {
		return "", errors.New("mode must be one of: append, create, zip")
	}
	return mode, nil
}

func mergeBookmarkMode(mode string) (model.MergeBookmarkMode, error) {
	if mode == "" {
		mode = string(model.MergeBookmarkModeWrap)
	}
	mode = modeCompletion(mode, []string{
		string(model.MergeBookmarkModeWrap),
		string(model.MergeBookmarkModePreserve),
	})
	if mode == "" {
		return "", errors.New("bookmark-mode must be one of: preserve, wrap")
	}
	return model.MergeBookmarkMode(mode), nil
}

func validateMergeModeArgs(mode string, args []string, dividerPage bool) error {
	if mode == "zip" && len(args) != 3 {
		return errors.New("merge zip: expecting outFile inFile1 inFile2")
	}
	if mode == "zip" && dividerPage {
		fmt.Fprintf(os.Stderr, "merge zip: -d(ivider) not applicable and will be ignored\n")
	}
	return nil
}

func validateMergeFiles(mode, outFile string, inFiles []string) error {
	if mode != "create" {
		if slices.Contains(inFiles, "-") {
			return fmt.Errorf("merge %s: stdin input not supported", mode)
		}
	}
	if mode == "append" && outFile == "-" {
		return errors.New("merge append: stdout not supported")
	}
	if mode != "append" {
		return ensureOutputFileAvailable(outFile)
	}
	return nil
}

func applyMergeOptions(opts *mergeOptions, conf *model.Configuration) (*model.Configuration, error) {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	if opts.bookmarksSet {
		conf.CreateBookmarks = opts.bookmarks
	}
	if opts.bookmarkModeSet {
		if !conf.CreateBookmarks {
			return nil, errors.New("merge: --bookmark-mode requires --bookmarks")
		}
		bookmarkMode, err := mergeBookmarkMode(opts.bookmarkMode)
		if err != nil {
			return nil, err
		}
		conf.MergeBookmarkMode = bookmarkMode
	}
	if opts.optimizeSet {
		conf.OptimizeBeforeWriting = opts.optimize
	}
	return conf, nil
}

func handleMergeCommand(conf *model.Configuration, args []string, opts *mergeOptions) error {
	mode, err := mergeMode(opts.mode)
	if err != nil {
		return err
	}
	opts.mode = mode
	if err := validateMergeModeArgs(opts.mode, args, opts.dividerPage); err != nil {
		return err
	}

	inFiles, outFile, err := processArgsForMerge(conf, args, opts.mode)
	if err != nil {
		return err
	}
	if err := validateMergeFiles(opts.mode, outFile, inFiles); err != nil {
		return err
	}
	if opts.sorted {
		sortFiles(inFiles)
	}

	conf, err = applyMergeOptions(opts, conf)
	if err != nil {
		return err
	}
	cmd := mergeCommandVariation(inFiles, outFile, opts.dividerPage, conf, opts.mode)
	if cmd == nil {
		return errors.New("missing merge mode")
	}
	return runCommand(cmd)
}

func splitPageNumbers(args []string) ([]int, error) {
	if len(args) == 2 {
		return nil, errors.New("split: missing page numbers")
	}
	pageNrs := make([]int, 0, len(args)-2)
	seen := map[int]bool{}
	for i := 2; i < len(args); i++ {
		p, err := strconv.Atoi(args[i])
		if err != nil || p < 2 {
			return nil, errors.New("split: pageNr is a numeric value >= 2")
		}
		if seen[p] {
			return nil, errors.New("split: page numbers must be unique")
		}
		if len(pageNrs) > 0 && p <= pageNrs[len(pageNrs)-1] {
			return nil, errors.New("split: page numbers must be sorted ascending")
		}
		seen[p] = true
		pageNrs = append(pageNrs, p)
	}
	return pageNrs, nil
}

func handleSplitByPageNumberCommand(inFile, outDir string, args []string, conf *model.Configuration) error {
	pageNrs, err := splitPageNumbers(args)
	if err != nil {
		return err
	}
	return runCommand(cli.SplitByPageNrCommand(inFile, outDir, pageNrs, conf))
}

func splitMode(opts *splitOptions) error {
	if opts.mode == "" {
		opts.mode = "span"
	}
	opts.mode = modeCompletion(opts.mode, []string{"span", "bookmark", "page"})
	if opts.mode == "" {
		return errors.New("mode must be one of: bookmark, span, page")
	}
	return nil
}

func splitInputOutput(conf *model.Configuration, args []string) (string, string, error) {
	inFile := args[0]
	if err := inputPDFArg(conf, inFile); err != nil {
		return "", "", err
	}
	outDir := args[1]
	if err := ensureOutputDirEmpty(outDir); err != nil {
		return "", "", err
	}
	return inFile, outDir, nil
}

func splitSpan(args []string) (int, error) {
	if len(args) > 3 {
		return 0, errors.New("split: span mode accepts at most one span")
	}
	if len(args) != 3 {
		return 1, nil
	}
	span, err := strconv.Atoi(args[2])
	if err != nil || span < 1 {
		return 0, errors.New("split: span is a numeric value >= 1")
	}
	return span, nil
}

func validateSplitModeArgs(mode string, args []string) error {
	if mode == "bookmark" && len(args) > 2 {
		return errors.New("split: bookmark mode does not accept span or page numbers")
	}
	if mode == "span" && len(args) > 3 {
		return errors.New("split: span mode accepts at most one span")
	}
	return nil
}

func handleSplitCommand(conf *model.Configuration, args []string, opts *splitOptions) error {
	if err := splitMode(opts); err != nil {
		return err
	}
	if err := validateSplitModeArgs(opts.mode, args); err != nil {
		return err
	}
	inFile, outDir, err := splitInputOutput(conf, args)
	if err != nil {
		return err
	}

	if opts.mode == "page" {
		return handleSplitByPageNumberCommand(inFile, outDir, args, conf)
	}

	span := 0
	if opts.mode == "span" {
		var err error
		span, err = splitSpan(args)
		if err != nil {
			return err
		}
	}

	return runCommand(cli.SplitCommand(inFile, outDir, span, conf))
}

func selectedPagesPDFArgs(conf *model.Configuration, args []string) (string, string, []string, error) {
	inFile, outFile, err := optionalOutputPDFArgs(conf, args)
	if err != nil {
		return "", "", nil, err
	}
	pages, err := parseSelectedPages()
	if err != nil {
		return "", "", nil, err
	}
	return inFile, outFile, pages, nil
}

func handleTrimCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, pages, err := selectedPagesPDFArgs(conf, args)
	if err != nil {
		return err
	}
	return runCommand(cli.TrimCommand(inFile, outFile, pages, conf))
}

func handleCollectCommand(conf *model.Configuration, args []string) error {
	inFile, outFile, pages, err := selectedPagesPDFArgs(conf, args)
	if err != nil {
		return err
	}
	return runCommand(cli.CollectCommand(inFile, outFile, pages, conf))
}

func createArgs(args []string) (string, string, string, error) {
	inFileJSON := args[0]
	if err := ensureJSONExtension(inFileJSON); err != nil {
		return "", "", "", err
	}

	inFile, outFile := "", ""
	if len(args) == 2 {
		outFile = args[1]
	} else {
		inFile = args[1]
		if inFile != "-" {
			if err := ensurePDFExtension(inFile); err != nil {
				return "", "", "", err
			}
		}
		outFile = args[2]
	}
	if outFile != "-" {
		if err := ensurePDFExtension(outFile); err != nil {
			return "", "", "", err
		}
	}
	return inFile, inFileJSON, outFile, nil
}

func handleCreateCommand(conf *model.Configuration, args []string) error {
	inFile, inFileJSON, outFile, err := createArgs(args)
	if err != nil {
		return err
	}
	if err := ensureOutputFileAvailable(outFile); err != nil {
		return err
	}
	return runCommand(cli.CreateCommand(inFile, inFileJSON, outFile, conf))
}

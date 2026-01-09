/*
Copyright 2020 The pdfcpu Authors.

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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/pkg/errors"
)

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func hasPDFExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

func ensurePDFExtension(filename string) {
	if !hasPDFExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", filename)
		os.Exit(1)
	}
}

func hasJSONExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".json")
}

func ensureJSONExtension(filename string) {
	if !hasJSONExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".json\".\n", filename)
		os.Exit(1)
	}
}

func hasCSVExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".csv")
}

func ensureCSVExtension(filename string) {
	if !hasCSVExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".csv\".\n", filename)
		os.Exit(1)
	}
}


func printConfiguration(conf *model.Configuration, args []string) {
	fmt.Fprintf(os.Stdout, "config: %s\n", conf.Path)
	f, err := os.Open(conf.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't open %s", conf.Path)
		os.Exit(1)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		fmt.Fprintf(os.Stderr, "can't read %s", conf.Path)
		os.Exit(1)
	}

	fmt.Print(string(buf.String()))
}

func confirmed() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("(yes/no): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input. Please try again.")
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "yes":
			return true
		case "no":
			return false
		default:
			fmt.Println("Invalid input. Please type 'yes' or 'no'.")
		}
	}
}

func resetConfiguration(conf *model.Configuration, args []string) {
	fmt.Printf("Did you make a backup of %s ?\n", conf.Path)
	if confirmed() {
		fmt.Printf("Are you ready to reset your config.yml to %s ?\n", model.VersionStr)
		if confirmed() {
			fmt.Println("resetting..")
			if err := model.ResetConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "pdfcpu: config problem: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Finished - Don't forget to update config.yml with your modifications.")
		} else {
			fmt.Println("Operation canceled.")
		}
	} else {
		fmt.Println("Operation canceled.")
	}
}

func resetCertificates(conf *model.Configuration, args []string) {
	fmt.Println("Are you ready to reset your certificates to your system root certificates?")
	if confirmed() {
		fmt.Println("resetting..")
		if err := model.ResetCertificates(); err != nil {
			fmt.Fprintf(os.Stderr, "pdfcpu: config problem: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Finished")
	} else {
		fmt.Println("Operation canceled")
	}
}

func printPaperSizes(conf *model.Configuration, args []string) {
	fmt.Fprintln(os.Stderr, paperSizes)
}

func printSelectedPages(conf *model.Configuration, args []string) {
	fmt.Fprintln(os.Stderr, usagePageSelection)
}

func printVersion(conf *model.Configuration, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "pdfcpu: %s\n", version)

	if date == "?" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
					if len(commit) >= 8 {
						commit = commit[:8]
					}
				}
				if setting.Key == "vcs.time" {
					date = setting.Value
				}
			}
		}
	}

	fmt.Fprintf(os.Stdout, "commit: %s (%s)\n", commit, date)
	fmt.Fprintf(os.Stdout, "base  : %s\n", runtime.Version())
	fmt.Fprintf(os.Stdout, "config: %s\n", conf.Path)
}

func process(cmd *cli.Command) {
	out, err := cli.Process(cmd)
	if err != nil {
		if needStackTrace {
			fmt.Fprintf(os.Stderr, "Fatal: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}

	if out != nil && !quiet {
		for _, s := range out {
			fmt.Fprintln(os.Stdout, s)
		}
	}
	//os.Exit(0)
}

func getBaseDir(path string) string {
	i := strings.Index(path, "**")
	basePath := path[:i]
	basePath = filepath.Clean(basePath)
	if basePath == "" {
		return "."
	}
	return basePath
}

func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func expandWildcardsRec(s string, inFiles *[]string, conf *model.Configuration) error {
	s = filepath.Clean(s)
	wantsPdf := strings.HasSuffix(s, ".pdf")
	return filepath.WalkDir(getBaseDir(s), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ok := hasPDFExtension(path); ok {
			*inFiles = append(*inFiles, path)
			return nil
		}
		if !wantsPdf && conf.CheckFileNameExt {
			if !quiet {
				fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", path)
			}
		}
		return nil
	})
}

func expandWildcards(s string, inFiles *[]string, conf *model.Configuration) error {
	paths, err := filepath.Glob(s)
	if err != nil {
		return err
	}
	for _, path := range paths {

		if conf.CheckFileNameExt {
			if !hasPDFExtension(path) {
				if isDir, err := isDir(path); isDir && err == nil {
					continue
				}
				if !quiet {
					fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", path)
				}
				continue
			}
		}

		*inFiles = append(*inFiles, path)
	}
	return nil
}

func collectInFiles(conf *model.Configuration, args []string) []string {
	inFiles := []string{}

	for _, arg := range args {

		if strings.Contains(arg, "**") {
			// **/			skips files w/o extension "pdf"
			// **/*.pdf
			if err := expandWildcardsRec(arg, &inFiles, conf); err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
			}
			continue
		}

		if strings.Contains(arg, "*") {
			// *			skips files w/o extension "pdf"
			// *.pdf
			if err := expandWildcards(arg, &inFiles, conf); err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
			}
			continue
		}

		if conf.CheckFileNameExt {
			if !hasPDFExtension(arg) {
				if isDir, err := isDir(arg); isDir && err == nil {
					if err := expandWildcards(arg+"/*", &inFiles, conf); err != nil {
						fmt.Fprintf(os.Stderr, "%s", err)
					}
					continue
				}
				if !quiet {
					fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", arg)
				}
				continue
			}
		}

		inFiles = append(inFiles, arg)
	}

	return inFiles
}

func processValidateCommand(conf *model.Configuration, args []string, opts *validateOptions) {
	if len(args) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
	}

	inFiles := collectInFiles(conf, args)

	switch opts.mode {
	case "strict", "s":
		conf.ValidationMode = model.ValidationStrict
	case "relaxed", "r":
		conf.ValidationMode = model.ValidationRelaxed
	case "":
		conf.ValidationMode = model.ValidationRelaxed
	default:
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
	}

	if opts.links {
		conf.ValidateLinks = true
	}

	conf.Optimize = opts.optimize

	process(cli.ValidateCommand(inFiles, conf))
}

func processOptimizeCommand(conf *model.Configuration, args []string, opts *optimizeCommandOptions) {
	if len(args) == 0 || len(args) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageOptimize)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	conf.StatsFileName = opts.fileStats
	if len(opts.fileStats) > 0 {
		fmt.Fprintf(os.Stdout, "stats will be appended to %s\n", opts.fileStats)
	}

	process(cli.OptimizeCommand(inFile, outFile, conf))
}

func processSplitByPageNumberCommand(inFile, outDir string, args []string, conf *model.Configuration) {
	if len(args) == 2 {
		fmt.Fprintln(os.Stderr, "split: missing page numbers")
		os.Exit(1)
	}

	ii := types.IntSet{}
	for i := 2; i < len(args); i++ {
		p, err := strconv.Atoi(args[i])
		if err != nil || p < 2 {
			fmt.Fprintln(os.Stderr, "split: pageNr is a numeric value >= 2")
			os.Exit(1)
		}
		ii[p] = true
	}

	pageNrs := make([]int, 0, len(ii))
	for k := range ii {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	process(cli.SplitByPageNrCommand(inFile, outDir, pageNrs, conf))
}

func processSplitCommand(conf *model.Configuration, args []string, opts *splitOptions) {
	if opts.mode == "" {
		opts.mode = "span"
	}
	opts.mode = modeCompletion(opts.mode, []string{"span", "bookmark", "page"})
	if opts.mode == "" || len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := args[1]

	if opts.mode == "page" {
		processSplitByPageNumberCommand(inFile, outDir, args, conf)
		return
	}

	span := 0

	if opts.mode == "span" {
		span = 1
		var err error
		if len(args) == 3 {
			span, err = strconv.Atoi(args[2])
			if err != nil || span < 1 {
				fmt.Fprintln(os.Stderr, "split: span is a numeric value >= 1")
				os.Exit(1)
			}
		}
	}

	process(cli.SplitCommand(inFile, outDir, span, conf))
}

func sortFiles(inFiles []string) {

	// See PR #631

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

func processArgsForMerge(conf *model.Configuration, args []string, mergeMode string) ([]string, string) {
	inFiles := []string{}
	outFile := ""
	for i, arg := range args {
		if i == 0 {
			ensurePDFExtension(arg)
			outFile = arg
			continue
		}
		if arg == outFile {
			fmt.Fprintf(os.Stderr, "%s may appear as inFile or outFile only\n", outFile)
			os.Exit(1)
		}
		if mergeMode != "zip" && strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			// TODO check extension
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}
	return inFiles, outFile
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

func processMergeCommand(conf *model.Configuration, args []string, opts *mergeOptions) {
	if opts.mode == "" {
		opts.mode = "create"
	}
	opts.mode = modeCompletion(opts.mode, []string{"create", "append", "zip"})
	if opts.mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	if opts.mode == "zip" && len(args) != 3 {
		fmt.Fprintf(os.Stderr, "merge zip: expecting outFile inFile1 inFile2\n")
		os.Exit(1)
	}

	if opts.mode == "zip" && opts.dividerPage {
		fmt.Fprintf(os.Stderr, "merge zip: -d(ivider) not applicable and will be ignored\n")
	}

	inFiles, outFile := processArgsForMerge(conf, args, opts.mode)

	if opts.sorted {
		sortFiles(inFiles)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	if opts.bookmarksSet {
		conf.CreateBookmarks = opts.bookmarks
	}

	if opts.optimizeSet {
		conf.OptimizeBeforeWriting = opts.optimize
	}

	cmd := mergeCommandVariation(inFiles, outFile, opts.dividerPage, conf, opts.mode)
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	process(cmd)
}

func modeCompletion(modePrefix string, modes []string) string {
	var modeStr string
	for _, mode := range modes {
		if !strings.HasPrefix(mode, modePrefix) {
			continue
		}
		if len(modeStr) > 0 {
			return ""
		}
		modeStr = mode
	}
	return modeStr
}

func processExtractCommand(conf *model.Configuration, args []string, opts *extractOptions) {
	opts.mode = modeCompletion(opts.mode, []string{"image", "font", "page", "content", "meta"})
	if len(args) != 2 || opts.mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageExtract)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outDir := args[1]

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var cmd *cli.Command

	switch opts.mode {

	case "image":
		cmd = cli.ExtractImagesCommand(inFile, outDir, pages, conf)

	case "font":
		cmd = cli.ExtractFontsCommand(inFile, outDir, pages, conf)

	case "page":
		cmd = cli.ExtractPagesCommand(inFile, outDir, pages, conf)

	case "content":
		cmd = cli.ExtractContentCommand(inFile, outDir, pages, conf)

	case "meta":
		cmd = cli.ExtractMetadataCommand(inFile, outDir, conf)

	default:
		fmt.Fprintf(os.Stderr, "unknown extract mode: %s\n", opts.mode)
		os.Exit(1)

	}

	process(cmd)
}

func processTrimCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || len(args) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageTrim)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	process(cli.TrimCommand(inFile, outFile, pages, conf))
}

func processListAttachmentsCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAttachList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListAttachmentsCommand(inFile, conf))
}

func processAddAttachmentsCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		os.Exit(1)
	}

	var inFile string
	fileNames := []string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			fileNames = append(fileNames, matches...)
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.AddAttachmentsCommand(inFile, "", fileNames, conf))
}

func processAddAttachmentsPortfolioCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		os.Exit(1)
	}

	var inFile string
	fileNames := []string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			fileNames = append(fileNames, matches...)
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.AddAttachmentsPortfolioCommand(inFile, "", fileNames, conf))
}

func processRemoveAttachmentsCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachRemove)
		os.Exit(1)
	}

	var inFile string
	fileNames := []string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.RemoveAttachmentsCommand(inFile, "", fileNames, conf))
}

func processExtractAttachmentsCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachExtract)
		os.Exit(1)
	}

	var inFile string
	fileNames := []string{}
	var outDir string

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		if i == 1 {
			outDir = arg
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.ExtractAttachmentsCommand(inFile, outDir, fileNames, conf))
}

func processListPermissionsCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePermList)
		os.Exit(1)
	}

	inFiles := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			// TODO check extension
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	process(cli.ListPermissionsCommand(inFiles, conf))
}

func permCompletion(permPrefix string) string {
	for _, perm := range []string{"none", "print", "all"} {
		if !strings.HasPrefix(perm, permPrefix) {
			continue
		}
		return perm
	}

	return permPrefix
}

func isBinary(s string) bool {
	_, err := strconv.ParseUint(s, 2, 12)
	return err == nil
}

func isHex(s string) bool {
	if s[0] != 'x' {
		return false
	}
	s = s[1:]
	_, err := strconv.ParseUint(s, 16, 16)
	return err == nil
}

func configPerm(perm string, conf *model.Configuration) {
	if perm != "" {
		switch perm {
		case "none":
			conf.Permissions = model.PermissionsNone
		case "print":
			conf.Permissions = model.PermissionsPrint
		case "all":
			conf.Permissions = model.PermissionsAll
		default:
			var p uint64
			if perm[0] == 'x' {
				p, _ = strconv.ParseUint(perm[1:], 16, 16)
			} else {
				p, _ = strconv.ParseUint(perm, 2, 12)
			}
			conf.Permissions = model.PermissionFlags(p)
		}
	}
}

func processSetPermissionsCommand(conf *model.Configuration, args []string) {
	if perm != "" {
		perm = permCompletion(perm)
	}
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		os.Exit(1)
	}
	if perm != "" && perm != "none" && perm != "print" && perm != "all" && !isBinary(perm) && !isHex(perm) {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	configPerm(perm, conf)

	process(cli.SetPermissionsCommand(inFile, "", conf))
}

func processDecryptCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || len(args) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageDecrypt)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	process(cli.DecryptCommand(inFile, outFile, conf))
}

func validateEncryptModeFlag(opts *encryptOptions) {
	if !types.MemberOf(opts.mode, []string{"rc4", "aes", ""}) {
		fmt.Fprintf(os.Stderr, "%s\n\n", "valid modes: rc4,aes default:aes")
		os.Exit(1)
	}

	// Default to AES encryption.
	if opts.mode == "" {
		opts.mode = "aes"
	}

	if opts.key == "256" && opts.mode == "rc4" {
		opts.key = "128"
	}

	if opts.mode == "rc4" {
		if opts.key != "40" && opts.key != "128" && opts.key != "" {
			fmt.Fprintf(os.Stderr, "%s\n\n", "supported RC4 key lengths: 40,128 default:128")
			os.Exit(1)
		}
	}

	if opts.mode == "aes" {
		if opts.key != "40" && opts.key != "128" && opts.key != "256" && opts.key != "" {
			fmt.Fprintf(os.Stderr, "%s\n\n", "supported AES key lengths: 40,128,256 default:256")
			os.Exit(1)
		}
	}

}

func validateEncryptFlags(opts *encryptOptions) {
	validateEncryptModeFlag(opts)
	if opts.perm != "none" && opts.perm != "print" && opts.perm != "all" && opts.perm != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported permissions: none,print,all default:none (viewing always allowed!)")
		os.Exit(1)
	}
}

func processEncryptCommand(conf *model.Configuration, args []string, opts *encryptOptions) {
	if opts.perm != "" {
		opts.perm = permCompletion(opts.perm)
	}
	if len(args) == 0 || len(args) > 2 ||
		!(opts.perm == "none" || opts.perm == "print" || opts.perm == "all") {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

	if conf.OwnerPW == "" {
		fmt.Fprintln(os.Stderr, "missing non-empty owner password!")
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

	validateEncryptFlags(opts)
	if opts.perm != "" {
		opts.perm = permCompletion(opts.perm)
	}

	conf.EncryptUsingAES = opts.mode != "rc4"

	kl, _ := strconv.Atoi(opts.key)
	conf.EncryptKeyLength = kl

	if opts.perm == "all" {
		conf.Permissions = model.PermissionsAll
	}

	if opts.perm == "print" {
		conf.Permissions = model.PermissionsPrint
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	process(cli.EncryptCommand(inFile, outFile, conf))
}

func processChangeUserPasswordCommand(conf *model.Configuration, args []string) {
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeUserPW)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	pwOld := args[1]
	pwNew := args[2]

	process(cli.ChangeUserPWCommand(inFile, outFile, &pwOld, &pwNew, conf))
}

func processChangeOwnerPasswordCommand(conf *model.Configuration, args []string) {
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	pwOld := args[1]
	pwNew := args[2]
	if pwNew == "" {
		fmt.Fprintf(os.Stderr, "owner password cannot be empty")
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		os.Exit(1)
	}

	process(cli.ChangeOwnerPWCommand(inFile, outFile, &pwOld, &pwNew, conf))
}

func addWatermarks(conf *model.Configuration, args []string, onTop bool, wmMode string) {
	u := usageWatermarkAdd
	if onTop {
		u = usageStampAdd
	}

	if len(args) < 3 || len(args) > 4 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", u)
		os.Exit(1)
	}

	if wmMode != "text" && wmMode != "image" && wmMode != "pdf" {
		fmt.Fprintln(os.Stderr, "mode has to be one of: text, image or pdf")
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	var (
		wm  *model.Watermark
		err error
	)

	switch wmMode {
	case "text":
		wm, err = pdfcpu.ParseTextWatermarkDetails(args[0], args[1], onTop, conf.Unit)

	case "image":
		wm, err = pdfcpu.ParseImageWatermarkDetails(args[0], args[1], onTop, conf.Unit)

	case "pdf":
		wm, err = pdfcpu.ParsePDFWatermarkDetails(args[0], args[1], onTop, conf.Unit)
	default:
		err = errors.Errorf("unsupported wm type: %s\n", wmMode)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		os.Exit(1)
	}

	inFile := args[2]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 4 {
		outFile = args[3]
		ensurePDFExtension(outFile)
	}

	process(cli.AddWatermarksCommand(inFile, outFile, selectedPages, wm, conf))
}

func processAddStampsCommand(conf *model.Configuration, args []string, opts *stampOptions) {
	addWatermarks(conf, args, true, opts.mode)
}

func processAddWatermarksCommand(conf *model.Configuration, args []string, opts *watermarkOptions) {
	addWatermarks(conf, args, false, opts.mode)
}

func updateWatermarks(conf *model.Configuration, args []string, onTop bool, wmMode string) {
	u := usageWatermarkUpdate
	if onTop {
		u = usageStampUpdate
	}

	if len(args) < 3 || len(args) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		os.Exit(1)
	}

	if wmMode != "text" && wmMode != "image" && wmMode != "pdf" {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	var (
		wm  *model.Watermark
		err error
	)

	switch wmMode {
	case "text":
		wm, err = pdfcpu.ParseTextWatermarkDetails(args[0], args[1], onTop, conf.Unit)
	case "image":
		wm, err = pdfcpu.ParseImageWatermarkDetails(args[0], args[1], onTop, conf.Unit)
	case "pdf":
		wm, err = pdfcpu.ParsePDFWatermarkDetails(args[0], args[1], onTop, conf.Unit)
	default:
		err = errors.Errorf("unsupported wm type: %s\n", wmMode)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		os.Exit(1)
	}

	wm.Update = true

	inFile := args[2]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 4 {
		outFile = args[3]
		ensurePDFExtension(outFile)
	}

	process(cli.AddWatermarksCommand(inFile, outFile, selectedPages, wm, conf))
}

func processUpdateStampsCommand(conf *model.Configuration, args []string, opts *stampOptions) {
	updateWatermarks(conf, args, true, opts.mode)
}

func processUpdateWatermarksCommand(conf *model.Configuration, args []string, opts *watermarkOptions) {
	updateWatermarks(conf, args, false, opts.mode)
}

func removeWatermarks(conf *model.Configuration, args []string, onTop bool) {
	if len(args) < 1 || len(args) > 2 {
		s := usageWatermarkRemove
		if onTop {
			s = usageStampRemove
		}
		fmt.Fprintf(os.Stderr, "%s\n\n", s)
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	process(cli.RemoveWatermarksCommand(inFile, outFile, selectedPages, conf))
}

func processRemoveStampsCommand(conf *model.Configuration, args []string) {
	removeWatermarks(conf, args, true)
}

func processRemoveWatermarksCommand(conf *model.Configuration, args []string) {
	removeWatermarks(conf, args, false)
}

func ensureImageExtension(filename string) {
	if !model.ImageFileName(filename) {
		fmt.Fprintf(os.Stderr, "%s needs an image extension (.jpg, .jpeg, .png, .tif, .tiff, .webp)\n", filename)
		os.Exit(1)
	}
}

func parseArgsForImageFileNames(args []string, startInd int) []string {
	imageFileNames := []string{}
	for i := startInd; i < len(args); i++ {
		arg := args[i]
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			for _, fn := range matches {
				ensureImageExtension(fn)
				imageFileNames = append(imageFileNames, fn)
			}
			continue
		}
		ensureImageExtension(arg)
		imageFileNames = append(imageFileNames, arg)
	}
	return imageFileNames
}

func processImportImagesCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImportImages)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	var outFile string
	outFile = args[0]
	if hasPDFExtension(outFile) {
		// pdfcpu import outFile imageFile...
		imp := pdfcpu.DefaultImportConfig()
		imageFileNames := parseArgsForImageFileNames(args, 1)
		process(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))
		return
	}

	// pdfcpu import description outFile imageFile...
	imp, err := pdfcpu.ParseImportDetails(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if imp == nil {
		fmt.Fprintf(os.Stderr, "missing import description\n")
		os.Exit(1)
	}

	outFile = args[1]
	ensurePDFExtension(outFile)
	imageFileNames := parseArgsForImageFileNames(args, 2)
	process(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))
}

func processInsertPagesCommand(conf *model.Configuration, args []string, opts *pagesInsertOptions) {
	if len(args) == 0 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePagesInsert)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	// Set default to insert pages before selected pages.
	if opts.mode != "" && opts.mode != "before" && opts.mode != "after" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesInsert)
		os.Exit(1)
	}

	inFile := args[0]
	if hasPDFExtension(inFile) {
		// pdfcpu pages insert inFile [outFile]

		outFile := ""
		if len(args) == 2 {
			outFile = args[1]
			ensurePDFExtension(outFile)
		}

		process(cli.InsertPagesCommand(inFile, outFile, pages, conf, opts.mode, nil))

		return
	}

	// pdfcpu pages insert description inFile [outFile]

	pageConf, err := pdfcpu.ParsePageConfiguration(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if pageConf == nil {
		fmt.Fprintf(os.Stderr, "missing page configuration\n")
		os.Exit(1)
	}

	inFile = args[1]
	ensurePDFExtension(inFile)

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	process(cli.InsertPagesCommand(inFile, outFile, pages, conf, opts.mode, pageConf))
}

func processRemovePagesCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || len(args) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePagesRemove)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outFile := ""
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}
	if pages == nil {
		fmt.Fprintf(os.Stderr, "missing page selection\n")
		os.Exit(1)
	}

	process(cli.RemovePagesCommand(inFile, outFile, pages, conf))
}

func processRotateCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageRotate)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	rotation, err := strconv.Atoi(args[1])
	if err != nil || abs(rotation)%90 > 0 {
		fmt.Fprintf(os.Stderr, "rotation must be a multiple of 90: %s\n", args[1])
		os.Exit(1)
	}

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.RotateCommand(inFile, outFile, rotation, selectedPages, conf))
}

func parseForGrid(args []string, nup *model.NUp, argInd *int) {
	cols, err := strconv.Atoi(args[*argInd])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	rows, err := strconv.Atoi(args[*argInd + 1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	if err = pdfcpu.ParseNUpGridDefinition(cols, rows, nup); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	*argInd += 2
}

func parseForNUp(args []string, nup *model.NUp, argInd *int, nUpValues []int) {
	n, err := strconv.Atoi(args[*argInd])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	if !types.IntMemberOf(n, nUpValues) {
		ss := make([]string, len(nUpValues))
		for i, v := range nUpValues {
			ss[i] = strconv.Itoa(v)
		}
		err := errors.Errorf("pdfcpu: n must be one of %s", strings.Join(ss, ", "))
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	if err = pdfcpu.ParseNUpValue(n, nup); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	*argInd++
}

func parseAfterNUpDetails(args []string, nup *model.NUp, argInd int, nUpValues []int, filenameOut string) []string {
	if nup.PageGrid {
		parseForGrid(args, nup, &argInd)
	} else {
		parseForNUp(args, nup, &argInd, nUpValues)
	}

	filenameIn := args[argInd]
	if !hasPDFExtension(filenameIn) && !model.ImageFileName(filenameIn) {
		fmt.Fprintf(os.Stderr, "inFile has to be a PDF or one or a sequence of image files: %s\n", filenameIn)
		os.Exit(1)
	}

	filenamesIn := []string{filenameIn}

	if hasPDFExtension(filenameIn) {
		if len(args) > argInd+1 {
			usage := usageNUp
			if nup.PageGrid {
				usage = usageGrid
			}
			fmt.Fprintf(os.Stderr, "%s\n\n", usage)
			os.Exit(1)
		}
		if filenameIn == filenameOut {
			fmt.Fprintln(os.Stderr, "inFile and outFile can't be the same.")
			os.Exit(1)
		}
	} else {
		nup.ImgInputFile = true
		for i := argInd + 1; i < len(args); i++ {
			arg := args[i]
			ensureImageExtension(arg)
			filenamesIn = append(filenamesIn, arg)
		}
	}

	return filenamesIn
}

func processNUpCommand(conf *model.Configuration, args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageNUp)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit
	argInd := 1

	outFile := args[0]
	if !hasPDFExtension(outFile) {
		// pdfcpu nup description outFile n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(args[0], nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		outFile = args[1]
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu nup outFile n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles := parseAfterNUpDetails(args, nup, argInd, pdfcpu.NUpValues, outFile)
	process(cli.NUpCommand(inFiles, outFile, pages, nup, conf))
}

func processGridCommand(conf *model.Configuration, args []string) {
	if len(args) < 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageGrid)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit
	nup.PageGrid = true
	argInd := 1

	outFile := args[0]
	if !hasPDFExtension(outFile) {
		// pdfcpu grid description outFile m n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(args[0], nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		outFile = args[1]
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu grid outFile m n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles := parseAfterNUpDetails(args, nup, argInd, nil, outFile)
	process(cli.NUpCommand(inFiles, outFile, pages, nup, conf))
}

func processBookletCommand(conf *model.Configuration, args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageBooklet)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	nup := pdfcpu.DefaultBookletConfig()
	nup.InpUnit = conf.Unit
	argInd := 1

	// First argument may be outFile or description.
	outFile := args[0]
	if !hasPDFExtension(outFile) {
		// pdfcpu booklet description outFile n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(args[0], nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		outFile = args[1]
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu booklet outFile n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles := parseAfterNUpDetails(args, nup, argInd, pdfcpu.NUpValuesForBooklets, outFile)
	process(cli.BookletCommand(inFiles, outFile, pages, nup, conf))
}

func processDisplayUnit(conf *model.Configuration, args []string) {
	if !types.MemberOf(unit, []string{"", "points", "po", "inches", "in", "cm", "mm"}) {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported units: (po)ints, (in)ches, cm, mm")
		os.Exit(1)
	}

	switch unit {
	case "points", "po":
		conf.Unit = types.POINTS
	case "inches", "in":
		conf.Unit = types.INCHES
	case "cm":
		conf.Unit = types.CENTIMETRES
	case "mm":
		conf.Unit = types.MILLIMETRES
	}
}

func processInfoCommand(conf *model.Configuration, args []string, opts *infoOptions) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageInfo)
		os.Exit(1)
	}

	inFiles := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			// TODO check extension
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	if opts.json {
		log.SetCLILogger(nil)
	}

	process(cli.InfoCommand(inFiles, selectedPages, opts.fonts, opts.json, conf))
}

func processListFontsCommand(conf *model.Configuration, args []string) {
	process(cli.ListFontsCommand(conf))
}

func processInstallFontsCommand(conf *model.Configuration, args []string) {
	fileNames := []string{}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", "expecting a list of TrueType filenames (.ttf, .ttc) for installation.")
		os.Exit(1)
	}
	for _, arg := range args {
		if !types.MemberOf(filepath.Ext(arg), []string{".ttf", ".ttc"}) {
			continue
		}
		fileNames = append(fileNames, arg)
	}
	if len(fileNames) == 0 {
		fmt.Fprintln(os.Stderr, "Please supply a *.ttf or *.tcc fontname!")
		os.Exit(1)
	}
	process(cli.InstallFontsCommand(fileNames, conf))
}

func processCreateCheatSheetFontsCommand(conf *model.Configuration, args []string) {
	fileNames := []string{}
	if len(args) > 0 {
		fileNames = append(fileNames, args...)
	}
	process(cli.CreateCheatSheetsFontsCommand(fileNames, conf))
}

func processListKeywordsCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageKeywordsList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListKeywordsCommand(inFile, conf))
}

func processAddKeywordsCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageKeywordsAdd)
		os.Exit(1)
	}

	var inFile string
	keywords := []string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		keywords = append(keywords, arg)
	}

	process(cli.AddKeywordsCommand(inFile, "", keywords, conf))
}

func processRemoveKeywordsCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageKeywordsRemove)
		os.Exit(1)
	}

	var inFile string
	keywords := []string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		keywords = append(keywords, arg)
	}

	process(cli.RemoveKeywordsCommand(inFile, "", keywords, conf))
}

func processListPropertiesCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePropertiesList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPropertiesCommand(inFile, conf))
}

func processAddPropertiesCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
		os.Exit(1)
	}

	var inFile string
	properties := map[string]string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		// Ensure key value pair.
		ss := strings.SplitN(arg, "=", 2)
		if len(ss) != 2 {
			fmt.Fprintf(os.Stderr, "keyValuePair = 'key = value'\n")
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
			os.Exit(1)
		}
		k := strings.TrimSpace(ss[0])
		if !validate.DocumentProperty(k) {
			fmt.Fprintf(os.Stderr, "property name \"%s\" not allowed!\n", k)
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
			os.Exit(1)
		}
		v := strings.TrimSpace(ss[1])
		properties[k] = v
	}

	process(cli.AddPropertiesCommand(inFile, "", properties, conf))
}

func processRemovePropertiesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesRemove)
		os.Exit(1)
	}

	var inFile string
	keys := []string{}

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}

		if !validate.DocumentProperty(arg) {
			fmt.Fprintf(os.Stderr, "property name \"%s\" not allowed!\n", arg)
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesRemove)
			os.Exit(1)
		}

		keys = append(keys, arg)
	}

	process(cli.RemovePropertiesCommand(inFile, "", keys, conf))
}

func processCollectCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || len(args) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCollect)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.CollectCommand(inFile, outFile, selectedPages, conf))
}

func processListBoxesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || len(args) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesList)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 1 {
		inFile := args[0]
		if conf.CheckFileNameExt {
			ensurePDFExtension(inFile)
		}
		process(cli.ListBoxesCommand(inFile, selectedPages, nil, conf))
		return
	}

	pb, err := api.PageBoundariesFromBoxList(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box list: %v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListBoxesCommand(inFile, selectedPages, pb, conf))
}

func processAddBoxesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesAdd)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	pb, err := api.PageBoundaries(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing page boundaries: %v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.AddBoxesCommand(inFile, outFile, selectedPages, pb, conf))
}

func processRemoveBoxesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesRemove)
		os.Exit(1)
	}

	pb, err := api.PageBoundariesFromBoxList(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box list: %v\n", err)
		os.Exit(1)
	}
	if pb == nil {
		fmt.Fprintln(os.Stderr, "please supply a list of box types to be removed")
		os.Exit(1)
	}

	if pb.Media != nil {
		fmt.Fprintf(os.Stderr, "cannot remove media box\n")
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.RemoveBoxesCommand(inFile, outFile, selectedPages, pb, conf))
}

func processCropCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageCrop)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	box, err := api.Box(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box definition: %v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.CropCommand(inFile, outFile, selectedPages, box, conf))
}

func processListAnnotationsCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAnnotsList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ListAnnotationsCommand(inFile, selectedPages, conf))
}

func processRemoveAnnotationsCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAnnotsRemove)
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	inFile, outFile := "", ""

	var (
		idsAndTypes []string
		objNrs      []int
	)

	for i, arg := range args {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		if i == 1 {
			if hasPDFExtension(arg) {
				outFile = arg
				continue
			}
		}

		j, err := strconv.Atoi(arg)
		if err != nil {
			// strings args may be and id or annotType
			idsAndTypes = append(idsAndTypes, arg)
			continue
		}
		objNrs = append(objNrs, j)
	}

	process(cli.RemoveAnnotationsCommand(inFile, outFile, selectedPages, idsAndTypes, objNrs, conf))
}

func processListImagesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageImagesList)
		os.Exit(1)
	}

	inFiles := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			// TODO check extension
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ListImagesCommand(inFiles, selectedPages, conf))
}

func processExtractImagesCommand(conf *model.Configuration, args []string) {
	// See also processExtractCommand
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImagesExtract)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outDir := args[1]

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ExtractImagesCommand(inFile, outDir, pages, conf))
}

func processUpdateImagesCommand(conf *model.Configuration, args []string) {
	argCount := len(args)
	if argCount < 2 || argCount > 5 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImagesUpdate)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	imageFile := args[1]
	ensureImageExtension(imageFile)

	outFile := ""
	objNrOrPageNr := 0
	id := ""

	if argCount > 2 {
		c := 2
		if hasPDFExtension(args[2]) {
			outFile = args[2]
			c++
		}
		if argCount > c {
			i, err := strconv.Atoi(args[c])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			if i <= 0 {
				fmt.Fprintln(os.Stderr, "objNr & pageNr must be > 0")
				os.Exit(1)
			}
			objNrOrPageNr = i
			if argCount == c+2 {
				id = args[c + 1]
			}
		}
	}

	//fmt.Printf("inFile:%s imgFile:%s outFile:%s, objPageNr:%d, id:%s\n", inFile, imageFile, outFile, objNrOrPageNr, id)

	process(cli.UpdateImagesCommand(inFile, imageFile, outFile, objNrOrPageNr, id, conf))
}

func processDumpCommand(conf *model.Configuration, args []string) {
	s := "No dump for you! - One year!\n\n"
	if len(args) != 3 {
		fmt.Fprintln(os.Stderr, s)
		os.Exit(1)
	}

	vals := []int{0, 0}

	mode := strings.ToLower(args[0])

	switch mode[0] {
	case 'a':
		vals[0] = 1
	case 'h':
		vals[0] = 2
	}

	objNr, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, s)
		os.Exit(1)
	}
	vals[1] = objNr

	inFile := args[2]
	ensurePDFExtension(inFile)

	conf.ValidationMode = model.ValidationRelaxed

	process(cli.DumpCommand(inFile, vals, conf))
}

func processCreateCommand(conf *model.Configuration, args []string) {
	if len(args) <= 1 || len(args) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCreate)
		os.Exit(1)
	}

	inFileJSON := args[0]
	ensureJSONExtension(inFileJSON)

	inFile, outFile := "", ""
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	} else {
		inFile = args[1]
		ensurePDFExtension(inFile)
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	process(cli.CreateCommand(inFile, inFileJSON, outFile, conf))
}

func processListFormFieldsCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormListFields)
		os.Exit(1)
	}

	inFiles := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			// TODO check extension
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	process(cli.ListFormFieldsCommand(inFiles, conf))
}

func processRemoveFormFieldsCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormRemoveFields)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(args) == 2 {
		s := args[1]
		if hasPDFExtension(s) {
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormRemoveFields)
			os.Exit(1)
		}
		fieldIDs = append(fieldIDs, s)
	} else {
		s := args[1]
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
		for i := 2; i < len(args); i++ {
			fieldIDs = append(fieldIDs, args[i])
		}
	}

	process(cli.RemoveFormFieldsCommand(inFile, outFile, fieldIDs, conf))
}

func processLockFormCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormLock)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(args) > 1 {
		s := args[1]
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
	}

	if len(args) > 2 {
		for i := 2; i < len(args); i++ {
			fieldIDs = append(fieldIDs, args[i])
		}
	}

	process(cli.LockFormCommand(inFile, outFile, fieldIDs, conf))
}

func processUnlockFormCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormUnlock)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(args) > 1 {
		s := args[1]
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
	}

	if len(args) > 2 {
		for i := 2; i < len(args); i++ {
			fieldIDs = append(fieldIDs, args[i])
		}
	}

	process(cli.UnlockFormCommand(inFile, outFile, fieldIDs, conf))
}

func processResetFormCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormReset)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(args) > 1 {
		s := args[1]
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
	}

	if len(args) > 2 {
		for i := 2; i < len(args); i++ {
			fieldIDs = append(fieldIDs, args[i])
		}
	}

	process(cli.ResetFormCommand(inFile, outFile, fieldIDs, conf))
}

func processExportFormCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || len(args) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormExport)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	// TODO inFile.json
	outFileJSON := "out.json"
	if len(args) == 2 {
		outFileJSON = args[1]
		ensureJSONExtension(outFileJSON)
	}
	ensureJSONExtension(outFileJSON)

	process(cli.ExportFormCommand(inFile, outFileJSON, conf))
}

func processFillFormCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || len(args) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormFill)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileJSON := args[1]
	ensureJSONExtension(inFileJSON)

	outFile := inFile
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	process(cli.FillFormCommand(inFile, inFileJSON, outFile, conf))
}

func processMultiFillFormCommand(conf *model.Configuration, args []string, opts *formMultifillOptions) {
	if opts.mode == "" {
		opts.mode = "single"
	}
	opts.mode = modeCompletion(opts.mode, []string{"single", "merge"})
	if opts.mode == "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormMultiFill)
		os.Exit(1)
	}

	if len(args) < 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormMultiFill)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileData := args[1]
	if !hasJSONExtension(inFileData) && !hasCSVExtension(inFileData) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".json\" or \".csv\".\n", inFileData)
		os.Exit(1)
	}

	outDir := args[2]

	outFile := inFile
	if len(args) == 4 {
		outFile = args[3]
		ensurePDFExtension(outFile)
	}

	process(cli.MultiFillFormCommand(inFile, inFileData, outDir, outFile, opts.mode == "merge", conf))
}

func processResizeCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageResize)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	rc, err := pdfcpu.ParseResizeConfig(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ResizeCommand(inFile, outFile, selectedPages, rc, conf))
}

func processPosterCommand(conf *model.Configuration, args []string) {
	if len(args) < 3 || len(args) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n", usagePoster)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	// formsize(=papersize) or dimensions, optionally: scalefactor, border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForPoster(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := args[2]

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var outFile string
	if len(args) == 4 {
		outFile = args[3]
	}

	process(cli.PosterCommand(inFile, outDir, outFile, selectedPages, cut, conf))
}

func processNDownCommand(conf *model.Configuration, args []string) {
	if len(args) < 3 || len(args) > 5 {
		fmt.Fprintf(os.Stderr, "%s\n", usageNDown)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var inFile, outDir string

	n, err := strconv.Atoi(args[0])
	if err == nil {
		// pdfcpu ndown n inFile outDir outFile

		// Optionally: border, margin, bgcolor
		cut, err := pdfcpu.ParseCutConfigForN(n, "", conf.Unit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		inFile = args[1]
		if conf.CheckFileNameExt {
			ensurePDFExtension(inFile)
		}
		outDir = args[2]

		var outFile string
		if len(args) == 4 {
			outFile = args[3]
		}

		process(cli.NDownCommand(inFile, outDir, outFile, selectedPages, n, cut, conf))
		return
	}

	// pdfcpu ndown description n inFile outDir outFile

	n, err = strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForN(n, args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile = args[2]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outDir = args[3]

	var outFile string
	if len(args) == 5 {
		outFile = args[4]
	}

	process(cli.NDownCommand(inFile, outDir, outFile, selectedPages, n, cut, conf))
}

func processCutCommand(conf *model.Configuration, args []string) {
	if len(args) < 3 || len(args) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n", usageCut)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	// required: at least one of horizontalCut, verticalCut
	// optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfig(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := args[2]

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var outFile string
	if len(args) >= 4 {
		outFile = args[3]
	}

	process(cli.CutCommand(inFile, outDir, outFile, selectedPages, cut, conf))
}

func processListBookmarksCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListBookmarksCommand(inFile, conf))
}

func processExportBookmarksCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || len(args) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksExport)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFileJSON := "out.json"
	if len(args) == 2 {
		outFileJSON = args[1]
		ensureJSONExtension(outFileJSON)
	}

	process(cli.ExportBookmarksCommand(inFile, outFileJSON, conf))
}

func processImportBookmarksCommand(conf *model.Configuration, args []string, opts *bookmarksImportOptions) {
	if len(args) == 0 || len(args) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksImport)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileJSON := args[1]
	ensureJSONExtension(inFileJSON)

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	process(cli.ImportBookmarksCommand(inFile, inFileJSON, outFile, opts.replaceBookmarks, conf))
}

func processRemoveBookmarksCommand(conf *model.Configuration, args []string) {
	if len(args) == 0 || len(args) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksExport)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 2 {
		outFile = args[1]
		ensurePDFExtension(outFile)
	}

	process(cli.RemoveBookmarksCommand(inFile, outFile, conf))
}

func processListPageLayoutCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPageLayoutCommand(inFile, conf))
}

func processSetPageLayoutCommand(conf *model.Configuration, args []string) {
	if len(args) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutSet)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	v := args[1]

	if !validate.DocumentPageLayout(v) {
		fmt.Fprintln(os.Stderr, "invalid page layout, use one of: SinglePage, TwoColumnLeft, TwoColumnRight, TwoPageLeft, TwoPageRight")
		os.Exit(1)
	}

	process(cli.SetPageLayoutCommand(inFile, "", v, conf))
}

func processResetPageLayoutCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutReset)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetPageLayoutCommand(inFile, "", conf))
}

func processListPageModeCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPageModeCommand(inFile, conf))
}

func processSetPageModeCommand(conf *model.Configuration, args []string) {
	if len(args) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeSet)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	v := args[1]

	if !validate.DocumentPageMode(v) {
		fmt.Fprintln(os.Stderr, "invalid page mode, use one of: UseNone, UseOutlines, UseThumbs, FullScreen, UseOC, UseAttachments")
		os.Exit(1)
	}

	process(cli.SetPageModeCommand(inFile, "", v, conf))
}

func processResetPageModeCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeReset)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetPageModeCommand(inFile, "", conf))
}

func processListViewerPreferencesCommand(conf *model.Configuration, args []string, opts *viewerpreferencesListOptions) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesList)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	if opts.json {
		log.SetCLILogger(nil)
	}

	process(cli.ListViewerPreferencesCommand(inFile, opts.all, opts.json, conf))
}

func processSetViewerPreferencesCommand(conf *model.Configuration, args []string) {
	if len(args) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesSet)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileJSON, stringJSON := "", ""

	s := args[1]
	if hasJSONExtension(s) {
		inFileJSON = s
	} else {
		stringJSON = s
	}

	process(cli.SetViewerPreferencesCommand(inFile, inFileJSON, "", stringJSON, conf))
}

func processResetViewerPreferencesCommand(conf *model.Configuration, args []string) {
	if len(args) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesReset)
		os.Exit(1)
	}

	inFile := args[0]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetViewerPreferencesCommand(inFile, "", conf))
}

func processZoomCommand(conf *model.Configuration, args []string) {
	if len(args) < 2 || len(args) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageZoom)
		os.Exit(1)
	}

	processDisplayUnit(conf, args)

	zc, err := pdfcpu.ParseZoomConfig(args[0], conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile := args[1]
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(args) == 3 {
		outFile = args[2]
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ZoomCommand(inFile, outFile, selectedPages, zc, conf))
}

func processListCertificatesCommand(conf *model.Configuration, args []string, opts *certificatesListOptions) {
	if len(args) > 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCertificatesList)
		os.Exit(1)
	}
	if opts.json {
		log.SetCLILogger(nil)
	}
	process(cli.ListCertificatesCommand(opts.json, conf))
}

func processInspectCertificatesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCertificatesInspect)
		os.Exit(1)
	}
	inFiles := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			for _, inFile := range matches {
				if !isCertificateFile(inFile) {
					fmt.Fprintf(os.Stderr, "skipping %s - allowed extensions: .pem, .p7c, .cer, .crt\n", inFile)
				} else {
					inFiles = append(inFiles, inFile)
				}
			}
			continue
		}
		if !isCertificateFile(arg) {
			fmt.Fprintf(os.Stderr, "%s - allowed extensions: .pem, .p7c, .cer, .crt\n", arg)
			os.Exit(1)
		}
		inFiles = append(inFiles, arg)
	}

	process(cli.InspectCertificatesCommand(inFiles, conf))
}

func isCertificateFile(fName string) bool {
	for _, ext := range []string{".p7c", ".pem", ".cer", ".crt"} {
		if strings.HasSuffix(strings.ToLower(fName), ext) {
			return true
		}
	}
	return false
}

func processImportCertificatesCommand(conf *model.Configuration, args []string) {
	if len(args) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCertificatesImport)
		os.Exit(1)
	}
	inFiles := []string{}
	for _, arg := range args {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			for _, inFile := range matches {
				if !isCertificateFile(inFile) {
					fmt.Fprintf(os.Stderr, "skipping %s - allowed extensions: .pem, .p7c, .cer, .crt\n", inFile)
				} else {
					inFiles = append(inFiles, inFile)
				}
			}
			continue
		}
		if !isCertificateFile(arg) {
			fmt.Fprintf(os.Stderr, "%s - allowed extensions: .pem, .p7c, .cer, .crt\n", arg)
			os.Exit(1)
		}
		inFiles = append(inFiles, arg)
	}

	process(cli.ImportCertificatesCommand(inFiles, conf))
}

func processValidateSignaturesCommand(conf *model.Configuration, args []string, opts *signaturesValidateOptions) {
	if len(args) > 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSignaturesValidate)
		os.Exit(1)
	}

	inFile := args[0]

	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ValidateSignaturesCommand(inFile, opts.all, opts.full, conf))
}

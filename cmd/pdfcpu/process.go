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
	"bytes"
	"flag"
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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/pkg/errors"
)

var errInvalidBookletID = errors.New("pdfcpu: booklet: n: one of 2, 4")

func hasPDFExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

func ensurePDFExtension(filename string) error {
	if !hasPDFExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", filename)
		return fmt.Errorf("%s needs extension \".pdf\".\n", filename)
	}
	return nil
}

func hasJSONExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".json")
}

func ensureJSONExtension(filename string) error {
	if !hasJSONExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".json\".\n", filename)
		return fmt.Errorf("%s needs extension \".pdf\".\n", filename)
	}
	return nil
}

func hasCSVExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".csv")
}

func ensureCSVExtension(filename string) error {
	if !hasCSVExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".csv\".\n", filename)
		return fmt.Errorf("%s needs extension \".csv\".\n", filename)
	}
	return nil
}

func printHelp(conf *model.Configuration) error {
	switch len(flag.Args()) {

	case 0:
		fmt.Fprintln(os.Stderr, usage)

	case 1:
		s, err := cmdMap.HelpString(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		fmt.Fprintln(os.Stderr, s)

	default:
		fmt.Fprintln(os.Stderr, "usage: pdfcpu help command\n\nToo many arguments.")

	}
	return nil
}

func printConfiguration(conf *model.Configuration) error {
	fmt.Fprintf(os.Stdout, "config: %s\n", conf.Path)
	f, err := os.Open(conf.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't open %s", conf.Path)
		return fmt.Errorf("can't open %s", conf.Path)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		fmt.Fprintf(os.Stderr, "can't read %s", conf.Path)
		return fmt.Errorf("can't read %s", conf.Path)
	}

	fmt.Print(string(buf.String()))
	return nil
}

func printPaperSizes(conf *model.Configuration) error {
	fmt.Fprintln(os.Stderr, paperSizes)
	return nil
}

func printSelectedPages(conf *model.Configuration) error {
	fmt.Fprintln(os.Stderr, usagePageSelection)
	return nil
}

func printVersion(conf *model.Configuration) error {
	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		return fmt.Errorf("%s", usageVersion)
	}

	fmt.Fprintf(os.Stdout, "pdfcpu: %s\n", model.VersionStr)

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

	return nil
}

func process(cmd *cli.Command) error {
	out, err := cli.Process(cmd)
	if err != nil {
		if needStackTrace {
			fmt.Fprintf(os.Stderr, "Fatal: %+v\n", err)
			return fmt.Errorf("fatal: %+v", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return fmt.Errorf("%v", err)
		}
	}

	if out != nil && !quiet {
		for _, s := range out {
			fmt.Fprintln(os.Stdout, s)
		}
	}
	//os.Exit(0)
	return nil
}

func processValidateCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		return fmt.Errorf("%s", usageValidate)
	}

	inFiles := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				return fmt.Errorf("%s", err)
			}
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	if mode != "" && mode != "strict" && mode != "s" && mode != "relaxed" && mode != "r" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		return fmt.Errorf("%s", usageValidate)
	}

	switch mode {
	case "strict", "s":
		conf.ValidationMode = model.ValidationStrict
	case "relaxed", "r":
		conf.ValidationMode = model.ValidationRelaxed
	}

	if links {
		conf.ValidateLinks = true
	}

	process(cli.ValidateCommand(inFiles, conf))
	return nil
}

func processOptimizeCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageOptimize)
		return fmt.Errorf("%s", usageOptimize)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	conf.StatsFileName = fileStats
	if len(fileStats) > 0 {
		fmt.Fprintf(os.Stdout, "stats will be appended to %s\n", fileStats)
	}

	process(cli.OptimizeCommand(inFile, outFile, conf))
	return nil
}

func processSplitByPageNumberCommand(inFile, outDir string, conf *model.Configuration) error {
	if len(flag.Args()) == 2 {
		fmt.Fprintln(os.Stderr, "split: missing page numbers")
		return fmt.Errorf("split: missing page numbers")
	}

	ii := types.IntSet{}
	for i := 2; i < len(flag.Args()); i++ {
		p, err := strconv.Atoi(flag.Arg(i))
		if err != nil || p < 2 {
			fmt.Fprintln(os.Stderr, "split: pageNr is a numeric value >= 2")
			return fmt.Errorf("split: pageNr is a numeric value >= 2")
		}
		ii[p] = true
	}

	pageNrs := make([]int, 0, len(ii))
	for k := range ii {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)

	process(cli.SplitByPageNrCommand(inFile, outDir, pageNrs, conf))
	return nil
}

func processSplitCommand(conf *model.Configuration) error {
	if mode == "" {
		mode = "span"
	}
	mode = modeCompletion(mode, []string{"span", "bookmark", "page"})
	if mode == "" || len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		return fmt.Errorf("%s", usageSplit)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := flag.Arg(1)

	if mode == "page" {
		processSplitByPageNumberCommand(inFile, outDir, conf)
		return nil
	}

	span := 0

	if mode == "span" {
		span = 1
		var err error
		if len(flag.Args()) == 3 {
			span, err = strconv.Atoi(flag.Arg(2))
			if err != nil || span < 1 {
				fmt.Fprintln(os.Stderr, "split: span is a numeric value >= 1")
				return fmt.Errorf("split: span is a numeric value >= 1")
			}
		}
	}

	process(cli.SplitCommand(inFile, outDir, span, conf))
	return nil
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

func processArgsForMerge(conf *model.Configuration) ([]string, string, error) {
	inFiles := []string{}
	outFile := ""
	for i, arg := range flag.Args() {
		if i == 0 {
			ensurePDFExtension(arg)
			outFile = arg
			continue
		}
		if arg == outFile {
			fmt.Fprintf(os.Stderr, "%s may appear as inFile or outFile only\n", outFile)
			return nil, "", fmt.Errorf("%s may appear as inFile or outFile only\n", outFile)
		}
		if mode != "zip" && strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				return nil, "", fmt.Errorf("%s", err)
			}
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}
	return inFiles, outFile, nil
}

func processMergeCommand(conf *model.Configuration) error {
	var err error
	if mode == "" {
		mode = "create"
	}
	mode = modeCompletion(mode, []string{"create", "append", "zip"})
	if mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		return fmt.Errorf("%s", usageMerge)
	}

	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		return fmt.Errorf("%s", usageMerge)
	}

	if mode == "zip" && len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "merge zip: expecting outFile inFile1 inFile2\n")
		return fmt.Errorf("merge zip: expecting outFile inFile1 inFile2\n")
	}

	if mode == "zip" && dividerPage {
		fmt.Fprintf(os.Stderr, "merge zip: -d(ivider) not applicable and will be ignored\n")
	}

	inFiles, outFile, err := processArgsForMerge(conf)
	if err != nil {
		return err
	}

	if sorted {
		sortFiles(inFiles)
	}

	if conf == nil {
		conf, err = model.NewDefaultConfiguration()
		if err != nil {
			return err
		}
		conf.CreateBookmarks = bookmarks
	}

	conf.CreateBookmarks = bookmarks

	var cmd *cli.Command

	switch mode {

	case "create":
		cmd = cli.MergeCreateCommand(inFiles, outFile, dividerPage, conf)

	case "zip":
		cmd = cli.MergeCreateZipCommand(inFiles, outFile, conf)

	case "append":
		cmd = cli.MergeAppendCommand(inFiles, outFile, dividerPage, conf)

	}

	process(cmd)
	return nil
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

func processExtractCommand(conf *model.Configuration) error {
	mode = modeCompletion(mode, []string{"image", "font", "page", "content", "meta"})
	if len(flag.Args()) != 2 || mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageExtract)
		return fmt.Errorf("%s", usageExtract)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outDir := flag.Arg(1)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	var cmd *cli.Command

	switch mode {

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
		fmt.Fprintf(os.Stderr, "unknown extract mode: %s\n", mode)
		return fmt.Errorf("unknown extract mode: %s\n", mode)

	}

	process(cmd)
	return nil
}

func processTrimCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageTrim)
		return fmt.Errorf("%s", usageTrim)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	process(cli.TrimCommand(inFile, outFile, pages, conf))
	return nil
}

func processListAttachmentsCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAttachList)
		return fmt.Errorf("usage: %s\n", usageAttachList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListAttachmentsCommand(inFile, conf))

	return nil
}

func processAddAttachmentsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		return fmt.Errorf("usage: %s\n\n", usageAttachAdd)
	}

	var inFile string
	fileNames := []string{}

	for i, arg := range flag.Args() {
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
				return fmt.Errorf("%s", err)
			}
			fileNames = append(fileNames, matches...)
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.AddAttachmentsCommand(inFile, "", fileNames, conf))

	return nil
}

func processAddAttachmentsPortfolioCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		return fmt.Errorf("usage: %s\n\n", usageAttachAdd)
	}

	var inFile string
	fileNames := []string{}

	for i, arg := range flag.Args() {
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
				return fmt.Errorf("%s", err)
			}
			fileNames = append(fileNames, matches...)
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.AddAttachmentsPortfolioCommand(inFile, "", fileNames, conf))

	return nil
}

func processRemoveAttachmentsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachRemove)
		return fmt.Errorf("usage: %s\n\n", usageAttachRemove)
	}

	var inFile string
	fileNames := []string{}

	for i, arg := range flag.Args() {
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

	return nil
}

func processExtractAttachmentsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachExtract)
		return fmt.Errorf("usage: %s\n\n", usageAttachExtract)
	}

	var inFile string
	fileNames := []string{}
	var outDir string

	for i, arg := range flag.Args() {
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

	return nil
}

func processListPermissionsCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePermList)
		return fmt.Errorf("usage: %s\n", usagePermList)
	}

	inFiles := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	process(cli.ListPermissionsCommand(inFiles, conf))

	return nil
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

func processSetPermissionsCommand(conf *model.Configuration) error {
	if perm != "" {
		perm = permCompletion(perm)
	}
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		return fmt.Errorf("usage: %s\n\n", usagePermSet)
	}
	if perm != "" && perm != "none" && perm != "print" && perm != "all" && !isBinary(perm) && !isHex(perm) {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		return fmt.Errorf("usage: %s\n\n", usagePermSet)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	configPerm(perm, conf)

	process(cli.SetPermissionsCommand(inFile, "", conf))

	return nil
}

func processDecryptCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageDecrypt)
		return fmt.Errorf("%s", usageDecrypt)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	process(cli.DecryptCommand(inFile, outFile, conf))

	return nil
}

func validateEncryptModeFlag() error {
	if !types.MemberOf(mode, []string{"rc4", "aes", ""}) {
		fmt.Fprintf(os.Stderr, "%s\n\n", "valid modes: rc4,aes default:aes")
		return fmt.Errorf("%s", "valid modes: rc4,aes default:aes")
	}

	// Default to AES encryption.
	if mode == "" {
		mode = "aes"
	}

	if key == "256" && mode == "rc4" {
		key = "128"
	}

	if mode == "rc4" {
		if key != "40" && key != "128" && key != "" {
			fmt.Fprintf(os.Stderr, "%s\n\n", "supported RC4 key lengths: 40,128 default:128")
			return fmt.Errorf("supported RC4 key lengths: 40,128 default:128")
		}
	}

	if mode == "aes" {
		if key != "40" && key != "128" && key != "256" && key != "" {
			fmt.Fprintf(os.Stderr, "%s\n\n", "supported AES key lengths: 40,128,256 default:256")
			return fmt.Errorf("%s", "supported AES key lengths: 40,128,256 default:256")
		}
	}

	return nil

}

func validateEncryptFlags() error {
	validateEncryptModeFlag()
	if perm != "none" && perm != "print" && perm != "all" && perm != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported permissions: none,print,all default:none (viewing always allowed!)")
		return fmt.Errorf("%s", "supported permissions: none,print,all default:none (viewing always allowed!)")
	}

	return nil
}

func processEncryptCommand(conf *model.Configuration) error {
	if perm != "" {
		perm = permCompletion(perm)
	}
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 ||
		!(perm == "none" || perm == "print" || perm == "all") {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		return fmt.Errorf("%s", usageEncrypt)
	}

	if conf.OwnerPW == "" {
		fmt.Fprintln(os.Stderr, "missing non-empty owner password!")
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		return fmt.Errorf("missing non-empty owner password! %s\n\n", usageEncrypt)
	}

	validateEncryptFlags()
	if perm != "" {
		perm = permCompletion(perm)
	}

	conf.EncryptUsingAES = mode != "rc4"

	kl, _ := strconv.Atoi(key)
	conf.EncryptKeyLength = kl

	if perm == "all" {
		conf.Permissions = model.PermissionsAll
	}

	if perm == "print" {
		conf.Permissions = model.PermissionsPrint
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	process(cli.EncryptCommand(inFile, outFile, conf))

	return nil
}

func processChangeUserPasswordCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeUserPW)
		return fmt.Errorf("%s", usageChangeUserPW)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	pwOld := flag.Arg(1)
	pwNew := flag.Arg(2)

	process(cli.ChangeUserPWCommand(inFile, outFile, &pwOld, &pwNew, conf))

	return nil
}

func processChangeOwnerPasswordCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		return fmt.Errorf("%s", usageChangeOwnerPW)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := inFile
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	pwOld := flag.Arg(1)
	pwNew := flag.Arg(2)
	if pwNew == "" {
		fmt.Fprintf(os.Stderr, "owner password cannot be empty")
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		return fmt.Errorf("owner password cannot be empty %s\n\n", usageChangeOwnerPW)
	}

	process(cli.ChangeOwnerPWCommand(inFile, outFile, &pwOld, &pwNew, conf))

	return nil
}

func addWatermarks(conf *model.Configuration, onTop bool) error {
	u := usageWatermarkAdd
	if onTop {
		u = usageStampAdd
	}

	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", u)
		return fmt.Errorf("usage: %s\n\n", u)
	}

	if mode != "text" && mode != "image" && mode != "pdf" {
		fmt.Fprintln(os.Stderr, "mode has to be one of: text, image or pdf")
		return fmt.Errorf("mode has to be one of: text, image or pdf")
	}

	processDiplayUnit(conf)

	var (
		wm  *model.Watermark
		err error
	)

	switch mode {
	case "text":
		wm, err = pdfcpu.ParseTextWatermarkDetails(flag.Arg(0), flag.Arg(1), onTop, conf.Unit)

	case "image":
		wm, err = pdfcpu.ParseImageWatermarkDetails(flag.Arg(0), flag.Arg(1), onTop, conf.Unit)

	case "pdf":
		wm, err = pdfcpu.ParsePDFWatermarkDetails(flag.Arg(0), flag.Arg(1), onTop, conf.Unit)
	default:
		err = errors.Errorf("unsupported wm type: %s\n", mode)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		return fmt.Errorf("problem with flag selectedPages: %v", err)
	}

	inFile := flag.Arg(2)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 4 {
		outFile = flag.Arg(3)
		ensurePDFExtension(outFile)
	}

	process(cli.AddWatermarksCommand(inFile, outFile, selectedPages, wm, conf))

	return nil
}

func processAddStampsCommand(conf *model.Configuration) error {
	err := addWatermarks(conf, true)
	if err != nil {
		return err
	}
	return nil
}

func processAddWatermarksCommand(conf *model.Configuration) error {
	err := addWatermarks(conf, false)
	if err != nil {
		return err
	}
	return nil
}

func updateWatermarks(conf *model.Configuration, onTop bool) error {
	u := usageWatermarkUpdate
	if onTop {
		u = usageStampUpdate
	}

	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		return fmt.Errorf("%s", u)
	}

	if mode != "text" && mode != "image" && mode != "pdf" {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		return fmt.Errorf("%s", u)
	}

	processDiplayUnit(conf)

	var (
		wm  *model.Watermark
		err error
	)

	switch mode {
	case "text":
		wm, err = pdfcpu.ParseTextWatermarkDetails(flag.Arg(0), flag.Arg(1), onTop, conf.Unit)

	case "image":
		wm, err = pdfcpu.ParseImageWatermarkDetails(flag.Arg(0), flag.Arg(1), onTop, conf.Unit)

	case "pdf":
		wm, err = pdfcpu.ParsePDFWatermarkDetails(flag.Arg(0), flag.Arg(1), onTop, conf.Unit)
	default:
		err = errors.Errorf("unsupported wm type: %s\n", mode)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		return err
	}

	wm.Update = true

	inFile := flag.Arg(2)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 4 {
		outFile = flag.Arg(3)
		ensurePDFExtension(outFile)
	}

	process(cli.AddWatermarksCommand(inFile, outFile, selectedPages, wm, conf))

	return nil
}

func processUpdateStampsCommand(conf *model.Configuration) error {
	err := updateWatermarks(conf, true)
	if err != nil {
		return err
	}
	return nil
}

func processUpdateWatermarksCommand(conf *model.Configuration) error {
	err := updateWatermarks(conf, false)
	if err != nil {
		return err
	}
	return nil
}

func removeWatermarks(conf *model.Configuration, onTop bool) error {
	if len(flag.Args()) < 1 || len(flag.Args()) > 2 {
		s := usageWatermarkRemove
		if onTop {
			s = usageStampRemove
		}
		fmt.Fprintf(os.Stderr, "%s\n\n", s)
		return fmt.Errorf("%s", s)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		return fmt.Errorf("problem with flag selectedPages: %v", err)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	process(cli.RemoveWatermarksCommand(inFile, outFile, selectedPages, conf))

	return nil
}

func processRemoveStampsCommand(conf *model.Configuration) error {
	removeWatermarks(conf, true)
	return nil
}

func processRemoveWatermarksCommand(conf *model.Configuration) error {
	removeWatermarks(conf, false)
	return nil
}

func ensureImageExtension(filename string) error {
	if !model.ImageFileName(filename) {
		fmt.Fprintf(os.Stderr, "%s needs an image extension (.jpg, .jpeg, .png, .tif, .tiff, .webp)\n", filename)
		return fmt.Errorf("%s needs an image extension (.jpg, .jpeg, .png, .tif, .tiff, .webp)\n", filename)
	}

	return nil
}

func parseArgsForImageFileNames(startInd int) ([]string, error) {
	imageFileNames := []string{}
	for i := startInd; i < len(flag.Args()); i++ {
		arg := flag.Arg(i)
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return nil, fmt.Errorf("%v\n", err)
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
	return imageFileNames, nil
}

func processImportImagesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImportImages)
		return fmt.Errorf("%s", usageImportImages)
	}

	processDiplayUnit(conf)

	var outFile string
	outFile = flag.Arg(0)
	if hasPDFExtension(outFile) {
		// pdfcpu import outFile imageFile...
		imp := pdfcpu.DefaultImportConfig()
		imageFileNames, err := parseArgsForImageFileNames(1)
		if err != nil {
			return err
		}
		process(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))
	}

	// pdfcpu import description outFile imageFile...
	imp, err := pdfcpu.ParseImportDetails(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	if imp == nil {
		fmt.Fprintf(os.Stderr, "missing import description\n")
		return fmt.Errorf("missing import description\n")
	}

	outFile = flag.Arg(1)
	ensurePDFExtension(outFile)
	imageFileNames, err := parseArgsForImageFileNames(2)
	if err != nil {
		return err
	}
	process(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))

	return nil
}

func processInsertPagesCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePagesInsert)
		return fmt.Errorf("usage: %s\n\n", usagePagesInsert)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outFile := ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	// Set default to insert pages before selected pages.
	if mode != "" && mode != "before" && mode != "after" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesInsert)
		return fmt.Errorf("%s", usagePagesInsert)
	}

	process(cli.InsertPagesCommand(inFile, outFile, pages, conf, mode))

	return nil

}

func processRemovePagesCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePagesRemove)
		return fmt.Errorf("usage: %s\n\n", usagePagesRemove)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outFile := ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}
	if pages == nil {
		fmt.Fprintf(os.Stderr, "missing page selection\n")
		return fmt.Errorf("missing page selection\n")
	}

	process(cli.RemovePagesCommand(inFile, outFile, pages, conf))

	return nil
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func processRotateCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageRotate)
		return fmt.Errorf("%s", usageRotate)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	rotation, err := strconv.Atoi(flag.Arg(1))
	if err != nil || abs(rotation)%90 > 0 {
		fmt.Fprintf(os.Stderr, "rotation must be a multiple of 90: %s\n", flag.Arg(1))
		return fmt.Errorf("rotation must be a multiple of 90: %s\n", flag.Arg(1))
	}

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.RotateCommand(inFile, outFile, rotation, selectedPages, conf))

	return nil
}

func parseAfterNUpDetails(nup *model.NUp, argInd int, filenameOut string) ([]string, error) {
	if nup.PageGrid {
		cols, err := strconv.Atoi(flag.Arg(argInd))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return nil, fmt.Errorf("%s\n", err)
		}
		rows, err := strconv.Atoi(flag.Arg(argInd + 1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return nil, err
		}
		if err = pdfcpu.ParseNUpGridDefinition(cols, rows, nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return nil, err
		}
		argInd += 2
	} else {
		n, err := strconv.Atoi(flag.Arg(argInd))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return nil, err
		}
		if err = pdfcpu.ParseNUpValue(n, nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return nil, err
		}
		argInd++
	}

	filenameIn := flag.Arg(argInd)
	if !hasPDFExtension(filenameIn) && !model.ImageFileName(filenameIn) {
		fmt.Fprintf(os.Stderr, "inFile has to be a PDF or one or a sequence of image files: %s\n", filenameIn)
		return nil, fmt.Errorf("inFile has to be a PDF or one or a sequence of image files: %s\n", filenameIn)
	}

	filenamesIn := []string{filenameIn}

	if hasPDFExtension(filenameIn) {
		if len(flag.Args()) > argInd+1 {
			usage := usageNUp
			if nup.PageGrid {
				usage = usageGrid
			}
			fmt.Fprintf(os.Stderr, "%s\n\n", usage)
			return nil, fmt.Errorf("%s", usage)
		}
		if filenameIn == filenameOut {
			fmt.Fprintln(os.Stderr, "inFile and outFile can't be the same.")
			return nil, fmt.Errorf("inFile and outFile can't be the same.")
		}
	} else {
		nup.ImgInputFile = true
		for i := argInd + 1; i < len(flag.Args()); i++ {
			arg := flag.Args()[i]
			ensureImageExtension(arg)
			filenamesIn = append(filenamesIn, arg)
		}
	}

	return filenamesIn, nil
}

func processNUpCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageNUp)
		return fmt.Errorf("%s", usageNUp)
	}

	processDiplayUnit(conf)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit
	argInd := 1

	outFile := flag.Arg(0)
	if !hasPDFExtension(outFile) {
		// pdfcpu nup description outFile n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(flag.Arg(0), nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return err
		}
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu nup outFile n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles, err := parseAfterNUpDetails(nup, argInd, outFile)
	if err != nil {
		return err
	}
	process(cli.NUpCommand(inFiles, outFile, pages, nup, conf))

	return nil
}

func processGridCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageGrid)
		return fmt.Errorf("%s", usageGrid)
	}

	processDiplayUnit(conf)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit
	nup.PageGrid = true
	argInd := 1

	outFile := flag.Arg(0)
	if !hasPDFExtension(outFile) {
		// pdfcpu grid description outFile m n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(flag.Arg(0), nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return err
		}
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu grid outFile m n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles, err := parseAfterNUpDetails(nup, argInd, outFile)
	if err != nil {
		return err
	}
	process(cli.NUpCommand(inFiles, outFile, pages, nup, conf))

	return nil
}

func processBookletCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageBooklet)
		return fmt.Errorf("%s", usageBooklet)
	}

	processDiplayUnit(conf)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	nup := pdfcpu.DefaultBookletConfig()
	nup.InpUnit = conf.Unit
	argInd := 1

	// First argument may be outFile or description.
	outFile := flag.Arg(0)
	if !hasPDFExtension(outFile) {
		// pdfcpu booklet description outFile n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(flag.Arg(0), nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return err
		}
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu booklet outFile n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles, err := parseAfterNUpDetails(nup, argInd, outFile)
	if err != nil {
		return err
	}
	n := nup.Grid.Width * nup.Grid.Height
	if n != 2 && n != 4 {
		fmt.Fprintf(os.Stderr, "%s\n", errInvalidBookletID)
		return fmt.Errorf("%s\n", errInvalidBookletID)
	}
	process(cli.BookletCommand(inFiles, outFile, pages, nup, conf))

	return nil
}

func processDiplayUnit(conf *model.Configuration) error {
	if !types.MemberOf(unit, []string{"", "points", "po", "inches", "in", "cm", "mm"}) {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported units: (po)ints, (in)ches, cm, mm")
		return fmt.Errorf("supported units: (po)ints, (in)ches, cm, mm")
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

	return nil
}

func processInfoCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageInfo)
		return fmt.Errorf("%s", usageInfo)
	}

	inFiles := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				return err
			}
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
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	processDiplayUnit(conf)

	process(cli.InfoCommand(inFiles, selectedPages, json, conf))

	return nil
}

func processListFontsCommand(conf *model.Configuration) error {
	process(cli.ListFontsCommand(conf))
	return nil
}

func processInstallFontsCommand(conf *model.Configuration) error {
	fileNames := []string{}
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", "expecting a list of TrueType filenames (.ttf, .ttc) for installation.")
		return fmt.Errorf("\n\n", "expecting a list of TrueType filenames (.ttf, .ttc) for installation.")
	}
	for _, arg := range flag.Args() {
		if !types.MemberOf(filepath.Ext(arg), []string{".ttf", ".ttc"}) {
			continue
		}
		fileNames = append(fileNames, arg)
	}
	if len(fileNames) == 0 {
		fmt.Fprintln(os.Stderr, "Please supply a *.ttf or *.tcc fontname!")
		return fmt.Errorf("Please supply a *.ttf or *.tcc fontname!")
	}
	process(cli.InstallFontsCommand(fileNames, conf))

	return nil
}

func processCreateCheatSheetFontsCommand(conf *model.Configuration) error {
	fileNames := []string{}
	if len(flag.Args()) > 0 {
		fileNames = append(fileNames, flag.Args()...)
	}
	process(cli.CreateCheatSheetsFontsCommand(fileNames, conf))
	return nil
}

func processListKeywordsCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageKeywordsList)
		return fmt.Errorf("usage: %s\n", usageKeywordsList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListKeywordsCommand(inFile, conf))

	return nil
}

func processAddKeywordsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageKeywordsAdd)
		return fmt.Errorf("usage: %s\n\n", usageKeywordsAdd)
	}

	var inFile string
	keywords := []string{}

	for i, arg := range flag.Args() {
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

	return nil
}

func processRemoveKeywordsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageKeywordsRemove)
		return fmt.Errorf("usage: %s\n\n", usageKeywordsRemove)
	}

	var inFile string
	keywords := []string{}

	for i, arg := range flag.Args() {
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

	return nil
}

func processListPropertiesCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePropertiesList)
		return fmt.Errorf("usage: %s\n", usagePropertiesList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPropertiesCommand(inFile, conf))

	return nil
}

func processAddPropertiesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
		return fmt.Errorf("usage: %s\n\n", usagePropertiesAdd)
	}

	var inFile string
	properties := map[string]string{}

	for i, arg := range flag.Args() {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		// Ensure key value pair.
		ss := strings.Split(arg, "=")
		if len(ss) != 2 {
			fmt.Fprintf(os.Stderr, "keyValuePair = 'key = value'\n")
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
			return fmt.Errorf("usage: %s\n\n", usagePropertiesAdd)
		}
		k := strings.TrimSpace(ss[0])
		if !validate.DocumentProperty(k) {
			fmt.Fprintf(os.Stderr, "property name \"%s\" not allowed!\n", k)
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
			return fmt.Errorf("usage: %s\n\n", usagePropertiesAdd)
		}
		v := strings.TrimSpace(ss[1])
		properties[k] = v
	}

	process(cli.AddPropertiesCommand(inFile, "", properties, conf))

	return nil
}

func processRemovePropertiesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesRemove)
		return fmt.Errorf("usage: %s\n\n", usagePropertiesRemove)
	}

	var inFile string
	keys := []string{}

	for i, arg := range flag.Args() {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		keys = append(keys, arg)
	}

	process(cli.RemovePropertiesCommand(inFile, "", keys, conf))

	return nil
}

func processCollectCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCollect)
		return fmt.Errorf("%s", usageCollect)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.CollectCommand(inFile, outFile, selectedPages, conf))

	return nil
}

func processListBoxesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesList)
		return fmt.Errorf("usage: %s\n", usageBoxesList)
	}

	processDiplayUnit(conf)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	if len(flag.Args()) == 1 {
		inFile := flag.Arg(0)
		if conf.CheckFileNameExt {
			ensurePDFExtension(inFile)
		}
		process(cli.ListBoxesCommand(inFile, selectedPages, nil, conf))
	}

	pb, err := api.PageBoundariesFromBoxList(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box list: %v\n", err)
		return fmt.Errorf("problem parsing box list: %v\n", err)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListBoxesCommand(inFile, selectedPages, pb, conf))

	return nil
}

func processAddBoxesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesAdd)
		return fmt.Errorf("usage: %s\n", usageBoxesAdd)
	}

	processDiplayUnit(conf)

	pb, err := api.PageBoundaries(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing page boundaries: %v\n", err)
		return fmt.Errorf("problem parsing page boundaries: %v\n", err)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.AddBoxesCommand(inFile, outFile, selectedPages, pb, conf))

	return nil
}

func processRemoveBoxesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesRemove)
		return fmt.Errorf("usage: %s\n", usageBoxesRemove)
	}

	pb, err := api.PageBoundariesFromBoxList(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box list: %v\n", err)
		return fmt.Errorf("problem parsing box list: %v\n", err)
	}
	if pb == nil {
		fmt.Fprintln(os.Stderr, "please supply a list of box types to be removed")
		return fmt.Errorf("please supply a list of box types to be removed")
	}

	if pb.Media != nil {
		fmt.Fprintf(os.Stderr, "cannot remove media box\n")
		return fmt.Errorf("cannot remove media box\n")
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.RemoveBoxesCommand(inFile, outFile, selectedPages, pb, conf))

	return err
}

func processCropCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageCrop)
		return fmt.Errorf("%s\n", usageCrop)
	}

	processDiplayUnit(conf)

	box, err := api.Box(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box definition: %v\n", err)
		return fmt.Errorf("problem parsing box definition: %v\n", err)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.CropCommand(inFile, outFile, selectedPages, box, conf))

	return nil
}

func processListAnnotationsCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAnnotsList)
		return fmt.Errorf("usage: %s\n", usageAnnotsList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.ListAnnotationsCommand(inFile, selectedPages, conf))

	return nil
}

func processRemoveAnnotationsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAnnotsRemove)
		return fmt.Errorf("usage: %s\n", usageAnnotsRemove)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	inFile, outFile := "", ""

	var (
		idsAndTypes []string
		objNrs      []int
	)

	for i, arg := range flag.Args() {
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

	return nil
}

func processListImagesCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageImagesList)
		return fmt.Errorf("usage: %s\n", usageImagesList)
	}

	inFiles := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				return err
			}
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
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.ListImagesCommand(inFiles, selectedPages, conf))

	return nil
}

func processDumpCommand(conf *model.Configuration) error {
	s := "No dump for you! - One year!\n\n"
	if len(flag.Args()) != 3 {
		fmt.Fprintln(os.Stderr, s)
		return fmt.Errorf(s)
	}

	vals := []int{0, 0}

	mode := strings.ToLower(flag.Arg(0))

	switch mode[0] {
	case 'a':
		vals[0] = 1
	case 'h':
		vals[0] = 2
	}

	objNr, err := strconv.Atoi(flag.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, s)
		return fmt.Errorf(s)
	}
	vals[1] = objNr

	inFile := flag.Arg(2)
	ensurePDFExtension(inFile)

	conf.ValidationMode = model.ValidationRelaxed

	process(cli.DumpCommand(inFile, vals, conf))

	return nil
}

func processCreateCommand(conf *model.Configuration) error {
	if len(flag.Args()) <= 1 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCreate)
		return fmt.Errorf("%s", usageCreate)
	}

	inFileJSON := flag.Arg(0)
	ensureJSONExtension(inFileJSON)

	inFile, outFile := "", ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	} else {
		inFile = flag.Arg(1)
		ensurePDFExtension(inFile)
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	process(cli.CreateCommand(inFile, inFileJSON, outFile, conf))

	return nil
}

func processListFormFieldsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormListFields)
		return fmt.Errorf("usage: %s\n\n", usageFormListFields)
	}

	inFiles := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				return err
			}
			inFiles = append(inFiles, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		inFiles = append(inFiles, arg)
	}

	process(cli.ListFormFieldsCommand(inFiles, conf))

	return nil
}

func processRemoveFormFieldsCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormRemoveFields)
		return fmt.Errorf("usage: %s\n\n", usageFormRemoveFields)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(flag.Args()) == 2 {
		s := flag.Arg(1)
		if hasPDFExtension(s) {
			fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormRemoveFields)
			return fmt.Errorf("usage: %s\n\n", usageFormRemoveFields)
		}
		fieldIDs = append(fieldIDs, s)
	} else {
		s := flag.Arg(1)
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
		for i := 2; i < len(flag.Args()); i++ {
			fieldIDs = append(fieldIDs, flag.Arg(i))
		}
	}

	process(cli.RemoveFormFieldsCommand(inFile, outFile, fieldIDs, conf))

	return nil
}

func processLockFormCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormLock)
		return fmt.Errorf("usage: %s\n\n", usageFormLock)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(flag.Args()) > 1 {
		s := flag.Arg(1)
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
	}

	if len(flag.Args()) > 2 {
		for i := 2; i < len(flag.Args()); i++ {
			fieldIDs = append(fieldIDs, flag.Arg(i))
		}
	}

	process(cli.LockFormCommand(inFile, outFile, fieldIDs, conf))

	return nil
}

func processUnlockFormCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormUnlock)
		return fmt.Errorf("usage: %s\n\n", usageFormUnlock)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(flag.Args()) > 1 {
		s := flag.Arg(1)
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
	}

	if len(flag.Args()) > 2 {
		for i := 2; i < len(flag.Args()); i++ {
			fieldIDs = append(fieldIDs, flag.Arg(i))
		}
	}

	process(cli.UnlockFormCommand(inFile, outFile, fieldIDs, conf))

	return nil
}

func processResetFormCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormReset)
		return fmt.Errorf("usage: %s\n\n", usageFormReset)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	var fieldIDs []string
	outFile := inFile

	if len(flag.Args()) > 1 {
		s := flag.Arg(1)
		if hasPDFExtension(s) {
			outFile = s
		} else {
			fieldIDs = append(fieldIDs, s)
		}
	}

	if len(flag.Args()) > 2 {
		for i := 2; i < len(flag.Args()); i++ {
			fieldIDs = append(fieldIDs, flag.Arg(i))
		}
	}

	process(cli.ResetFormCommand(inFile, outFile, fieldIDs, conf))

	return nil
}

func processExportFormCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormExport)
		return fmt.Errorf("usage: %s\n\n", usageFormExport)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	// TODO inFile.json
	outFileJSON := "out.json"
	if len(flag.Args()) == 2 {
		outFileJSON = flag.Arg(1)
		ensureJSONExtension(outFileJSON)
	}
	ensureJSONExtension(outFileJSON)

	process(cli.ExportFormCommand(inFile, outFileJSON, conf))

	return nil
}

func processFillFormCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormFill)
		return fmt.Errorf("usage: %s\n\n", usageFormFill)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileJSON := flag.Arg(1)
	ensureJSONExtension(inFileJSON)

	outFile := inFile
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	process(cli.FillFormCommand(inFile, inFileJSON, outFile, conf))

	return nil
}

func processMultiFillFormCommand(conf *model.Configuration) error {
	if mode == "" {
		mode = "single"
	}
	mode = modeCompletion(mode, []string{"single", "merge"})
	if mode == "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormMultiFill)
		return fmt.Errorf("usage: %s\n\n", usageFormMultiFill)
	}

	if len(flag.Args()) < 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormMultiFill)
		return fmt.Errorf("usage: %s\n\n", usageFormMultiFill)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileData := flag.Arg(1)
	if !hasJSONExtension(inFileData) && !hasCSVExtension(inFileData) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".json\" or \".csv\".\n", inFileData)
		return fmt.Errorf("%s needs extension \".json\" or \".csv\".\n", inFileData)
	}

	outDir := flag.Arg(2)

	outFile := inFile
	if len(flag.Args()) == 4 {
		outFile = flag.Arg(3)
		ensurePDFExtension(outFile)
	}

	process(cli.MultiFillFormCommand(inFile, inFileData, outDir, outFile, mode == "merge", conf))

	return nil
}

func processResizeCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 2 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageResize)
		return fmt.Errorf("%s\n", usageResize)
	}

	processDiplayUnit(conf)

	rc, err := pdfcpu.ParseResizeConfig(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	process(cli.ResizeCommand(inFile, outFile, selectedPages, rc, conf))

	return nil
}

func processPosterCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n", usagePoster)
		return fmt.Errorf("%s\n", usagePoster)
	}

	processDiplayUnit(conf)

	// formsize(=papersize) or dimensions, optionally: scalefactor, border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForPoster(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := flag.Arg(2)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	var outFile string
	if len(flag.Args()) == 4 {
		outFile = flag.Arg(3)
	}

	process(cli.PosterCommand(inFile, outDir, outFile, selectedPages, cut, conf))

	return nil
}

func processNDownCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 3 || len(flag.Args()) > 5 {
		fmt.Fprintf(os.Stderr, "%s\n", usageNDown)
		return fmt.Errorf("%s\n", usageNDown)
	}

	processDiplayUnit(conf)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	var inFile, outDir string

	n, err := strconv.Atoi(flag.Arg(0))
	if err == nil {
		// pdfcpu ndown n inFile outDir outFile

		// Optionally: border, margin, bgcolor
		cut, err := pdfcpu.ParseCutConfigForN(n, "", conf.Unit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return fmt.Errorf("%v\n", err)
		}
		inFile = flag.Arg(1)
		if conf.CheckFileNameExt {
			ensurePDFExtension(inFile)
		}
		outDir = flag.Arg(2)

		var outFile string
		if len(flag.Args()) == 4 {
			outFile = flag.Arg(3)
		}

		process(cli.NDownCommand(inFile, outDir, outFile, selectedPages, n, cut, conf))
	}

	// pdfcpu ndown description n inFile outDir outFile

	n, err = strconv.Atoi(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	// Optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForN(n, flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}

	inFile = flag.Arg(2)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outDir = flag.Arg(3)

	var outFile string
	if len(flag.Args()) == 5 {
		outFile = flag.Arg(4)
	}

	process(cli.NDownCommand(inFile, outDir, outFile, selectedPages, n, cut, conf))

	return nil
}

func processCutCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n", usageCut)
		return fmt.Errorf("%s\n", usageCut)
	}

	processDiplayUnit(conf)

	// required: at least one of horizontalCut, verticalCut
	// optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfig(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return fmt.Errorf("%v\n", err)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := flag.Arg(2)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		return fmt.Errorf("problem with flag selectedPages: %v\n", err)
	}

	var outFile string
	if len(flag.Args()) >= 4 {
		outFile = flag.Arg(3)
	}

	process(cli.CutCommand(inFile, outDir, outFile, selectedPages, cut, conf))

	return nil
}

func processListBookmarksCommand(conf *model.Configuration) error {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksList)
		return fmt.Errorf("usage: %s\n\n", usageBookmarksList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListBookmarksCommand(inFile, conf))

	return nil
}

func processExportBookmarksCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksExport)
		return fmt.Errorf("usage: %s\n\n", usageBookmarksExport)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFileJSON := "out.json"
	if len(flag.Args()) == 2 {
		outFileJSON = flag.Arg(1)
		ensureJSONExtension(outFileJSON)
	}

	process(cli.ExportBookmarksCommand(inFile, outFileJSON, conf))

	return nil
}

func processImportBookmarksCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksImport)
		return fmt.Errorf("usage: %s\n\n", usageBookmarksImport)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileJSON := flag.Arg(1)
	ensureJSONExtension(inFileJSON)

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	process(cli.ImportBookmarksCommand(inFile, inFileJSON, outFile, replaceBookmarks, conf))

	return nil
}

func processRemoveBookmarksCommand(conf *model.Configuration) error {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksExport)
		return fmt.Errorf("usage: %s\n\n", usageBookmarksExport)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outFile := ""
	if len(flag.Args()) == 2 {
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
	}

	process(cli.RemoveBookmarksCommand(inFile, outFile, conf))

	return nil
}

func processListPageLayoutCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutList)
		return fmt.Errorf("usage: %s\n", usagePageLayoutList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPageLayoutCommand(inFile, conf))

	return nil
}

func processSetPageLayoutCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutSet)
		return fmt.Errorf("usage: %s\n", usagePageLayoutSet)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	v := flag.Arg(1)

	if !validate.DocumentPageLayout(v) {
		fmt.Fprintln(os.Stderr, "invalid page layout, use one of: SinglePage, TwoColumnLeft, TwoColumnRight, TwoPageLeft, TwoPageRight")
		return fmt.Errorf("invalid page layout, use one of: SinglePage, TwoColumnLeft, TwoColumnRight, TwoPageLeft, TwoPageRight")
	}

	process(cli.SetPageLayoutCommand(inFile, "", v, conf))

	return nil
}

func processResetPageLayoutCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutReset)
		return fmt.Errorf("usage: %s\n", usagePageLayoutReset)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetPageLayoutCommand(inFile, "", conf))

	return nil
}

func processListPageModeCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeList)
		return fmt.Errorf("usage: %s\n", usagePageModeList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPageModeCommand(inFile, conf))

	return nil
}

func processSetPageModeCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeSet)
		return fmt.Errorf("usage: %s\n", usagePageModeSet)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	v := flag.Arg(1)

	if !validate.DocumentPageMode(v) {
		fmt.Fprintln(os.Stderr, "invalid page mode, use one of: UseNone, UseOutlines, UseThumbs, FullScreen, UseOC, UseAttachments")
		return fmt.Errorf("invalid page mode, use one of: UseNone, UseOutlines, UseThumbs, FullScreen, UseOC, UseAttachments")
	}

	process(cli.SetPageModeCommand(inFile, "", v, conf))

	return nil
}

func processResetPageModeCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeReset)
		return fmt.Errorf("usage: %s\n", usagePageModeReset)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetPageModeCommand(inFile, "", conf))

	return nil
}

func processListViewerPreferencesCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesList)
		return fmt.Errorf("usage: %s\n", usageViewerPreferencesList)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListViewerPreferencesCommand(inFile, all, json, conf))

	return nil
}

func processSetViewerPreferencesCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesSet)
		return fmt.Errorf("usage: %s\n", usageViewerPreferencesSet)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileJSON, stringJSON := "", ""

	s := flag.Arg(1)
	if hasJSONExtension(s) {
		inFileJSON = s
	} else {
		stringJSON = s
	}

	process(cli.SetViewerPreferencesCommand(inFile, inFileJSON, "", stringJSON, conf))

	return nil
}

func processResetViewerPreferencesCommand(conf *model.Configuration) error {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesReset)
		return fmt.Errorf("usage: %s\n", usageViewerPreferencesReset)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetViewerPreferencesCommand(inFile, "", conf))

	return nil
}

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

func printHelp(conf *model.Configuration) {
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
}

func printConfiguration(conf *model.Configuration) {
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

func printPaperSizes(conf *model.Configuration) {
	fmt.Fprintln(os.Stderr, paperSizes)
}

func printSelectedPages(conf *model.Configuration) {
	fmt.Fprintln(os.Stderr, usagePageSelection)
}

func printVersion(conf *model.Configuration) {
	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "pdfcpu: %s\n", model.VersionStr)

	if date == "?" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
				}
				if setting.Key == "vcs.time" {
					date = setting.Value
				}
			}
		}
	}

	fmt.Fprintf(os.Stdout, "commit: %s (%s)\n", commit[:8], date)
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

func processValidateCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
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

	if mode != "" && mode != "strict" && mode != "s" && mode != "relaxed" && mode != "r" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
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
}

func processOptimizeCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageOptimize)
		os.Exit(1)
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
}

func processSplitByPageNumberCommand(inFile, outDir string, conf *model.Configuration) {
	if len(flag.Args()) == 2 {
		fmt.Fprintln(os.Stderr, "split: missing page numbers")
		os.Exit(1)
	}

	ii := types.IntSet{}
	for i := 2; i < len(flag.Args()); i++ {
		p, err := strconv.Atoi(flag.Arg(i))
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

func processSplitCommand(conf *model.Configuration) {
	if mode == "" {
		mode = "span"
	}
	mode = modeCompletion(mode, []string{"span", "bookmark", "page"})
	if mode == "" || len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := flag.Arg(1)

	if mode == "page" {
		processSplitByPageNumberCommand(inFile, outDir, conf)
		return
	}

	span := 0

	if mode == "span" {
		span = 1
		var err error
		if len(flag.Args()) == 3 {
			span, err = strconv.Atoi(flag.Arg(2))
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

func processArgsForMerge(conf *model.Configuration) ([]string, string) {
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
			os.Exit(1)
		}
		if mode != "zip" && strings.Contains(arg, "*") {
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
	return inFiles, outFile
}

func processMergeCommand(conf *model.Configuration) {
	if mode == "" {
		mode = "create"
	}
	mode = modeCompletion(mode, []string{"create", "append", "zip"})
	if mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	if mode == "zip" && len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "merge zip: expecting outFile inFile1 inFile2\n")
		os.Exit(1)
	}

	if mode == "zip" && dividerPage {
		fmt.Fprintf(os.Stderr, "merge zip: -d(ivider) not applicable and will be ignored\n")
	}

	inFiles, outFile := processArgsForMerge(conf)

	if sorted {
		sortFiles(inFiles)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
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

func processExtractCommand(conf *model.Configuration) {
	mode = modeCompletion(mode, []string{"image", "font", "page", "content", "meta"})
	if len(flag.Args()) != 2 || mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageExtract)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	outDir := flag.Arg(1)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)

	}

	process(cmd)
}

func processTrimCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageTrim)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
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
}

func processListAttachmentsCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAttachList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListAttachmentsCommand(inFile, conf))
}

func processAddAttachmentsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		os.Exit(1)
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
				os.Exit(1)
			}
			fileNames = append(fileNames, matches...)
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.AddAttachmentsCommand(inFile, "", fileNames, conf))
}

func processAddAttachmentsPortfolioCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		os.Exit(1)
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
				os.Exit(1)
			}
			fileNames = append(fileNames, matches...)
			continue
		}
		fileNames = append(fileNames, arg)
	}

	process(cli.AddAttachmentsPortfolioCommand(inFile, "", fileNames, conf))
}

func processRemoveAttachmentsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachRemove)
		os.Exit(1)
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
}

func processExtractAttachmentsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachExtract)
		os.Exit(1)
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
}

func processListPermissionsCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePermList)
		os.Exit(1)
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

func processSetPermissionsCommand(conf *model.Configuration) {
	if perm != "" {
		perm = permCompletion(perm)
	}
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		os.Exit(1)
	}
	if perm != "" && perm != "none" && perm != "print" && perm != "all" && !isBinary(perm) && !isHex(perm) {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	configPerm(perm, conf)

	process(cli.SetPermissionsCommand(inFile, "", conf))
}

func processDecryptCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageDecrypt)
		os.Exit(1)
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
}

func validateEncryptModeFlag() {
	if !types.MemberOf(mode, []string{"rc4", "aes", ""}) {
		fmt.Fprintf(os.Stderr, "%s\n\n", "valid modes: rc4,aes default:aes")
		os.Exit(1)
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
			os.Exit(1)
		}
	}

	if mode == "aes" {
		if key != "40" && key != "128" && key != "256" && key != "" {
			fmt.Fprintf(os.Stderr, "%s\n\n", "supported AES key lengths: 40,128,256 default:256")
			os.Exit(1)
		}
	}

}

func validateEncryptFlags() {
	validateEncryptModeFlag()
	if perm != "none" && perm != "print" && perm != "all" && perm != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported permissions: none,print,all default:none (viewing always allowed!)")
		os.Exit(1)
	}
}

func processEncryptCommand(conf *model.Configuration) {
	if perm != "" {
		perm = permCompletion(perm)
	}
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 ||
		!(perm == "none" || perm == "print" || perm == "all") {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

	if conf.OwnerPW == "" {
		fmt.Fprintln(os.Stderr, "missing non-empty owner password!")
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
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
}

func processChangeUserPasswordCommand(conf *model.Configuration) {
	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeUserPW)
		os.Exit(1)
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
}

func processChangeOwnerPasswordCommand(conf *model.Configuration) {
	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		os.Exit(1)
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
		os.Exit(1)
	}

	process(cli.ChangeOwnerPWCommand(inFile, outFile, &pwOld, &pwNew, conf))
}

func addWatermarks(conf *model.Configuration, onTop bool) {
	u := usageWatermarkAdd
	if onTop {
		u = usageStampAdd
	}

	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", u)
		os.Exit(1)
	}

	if mode != "text" && mode != "image" && mode != "pdf" {
		fmt.Fprintln(os.Stderr, "mode has to be one of: text, image or pdf")
		os.Exit(1)
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
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		os.Exit(1)
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
}

func processAddStampsCommand(conf *model.Configuration) {
	addWatermarks(conf, true)
}

func processAddWatermarksCommand(conf *model.Configuration) {
	addWatermarks(conf, false)
}

func updateWatermarks(conf *model.Configuration, onTop bool) {
	u := usageWatermarkUpdate
	if onTop {
		u = usageStampUpdate
	}

	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		os.Exit(1)
	}

	if mode != "text" && mode != "image" && mode != "pdf" {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		os.Exit(1)
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
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v", err)
		os.Exit(1)
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
}

func processUpdateStampsCommand(conf *model.Configuration) {
	updateWatermarks(conf, true)
}

func processUpdateWatermarksCommand(conf *model.Configuration) {
	updateWatermarks(conf, false)
}

func removeWatermarks(conf *model.Configuration, onTop bool) {
	if len(flag.Args()) < 1 || len(flag.Args()) > 2 {
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
}

func processRemoveStampsCommand(conf *model.Configuration) {
	removeWatermarks(conf, true)
}

func processRemoveWatermarksCommand(conf *model.Configuration) {
	removeWatermarks(conf, false)
}

func ensureImageExtension(filename string) {
	if !model.ImageFileName(filename) {
		fmt.Fprintf(os.Stderr, "%s needs an image extension (.jpg, .jpeg, .png, .tif, .tiff, .webp)\n", filename)
		os.Exit(1)
	}
}

func parseArgsForImageFileNames(startInd int) []string {
	imageFileNames := []string{}
	for i := startInd; i < len(flag.Args()); i++ {
		arg := flag.Arg(i)
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

func processImportImagesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImportImages)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	var outFile string
	outFile = flag.Arg(0)
	if hasPDFExtension(outFile) {
		// pdfcpu import outFile imageFile...
		imp := pdfcpu.DefaultImportConfig()
		imageFileNames := parseArgsForImageFileNames(1)
		process(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))
	}

	// pdfcpu import description outFile imageFile...
	imp, err := pdfcpu.ParseImportDetails(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if imp == nil {
		fmt.Fprintf(os.Stderr, "missing import description\n")
		os.Exit(1)
	}

	outFile = flag.Arg(1)
	ensurePDFExtension(outFile)
	imageFileNames := parseArgsForImageFileNames(2)
	process(cli.ImportImagesCommand(imageFileNames, outFile, imp, conf))
}

func processInsertPagesCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePagesInsert)
		os.Exit(1)
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
		os.Exit(1)
	}

	// Set default to insert pages before selected pages.
	if mode != "" && mode != "before" && mode != "after" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesInsert)
		os.Exit(1)
	}

	process(cli.InsertPagesCommand(inFile, outFile, pages, conf, mode))
}

func processRemovePagesCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePagesRemove)
		os.Exit(1)
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
		os.Exit(1)
	}
	if pages == nil {
		fmt.Fprintf(os.Stderr, "missing page selection\n")
		os.Exit(1)
	}

	process(cli.RemovePagesCommand(inFile, outFile, pages, conf))
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func processRotateCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageRotate)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	rotation, err := strconv.Atoi(flag.Arg(1))
	if err != nil || abs(rotation)%90 > 0 {
		fmt.Fprintf(os.Stderr, "rotation must be a multiple of 90: %s\n", flag.Arg(1))
		os.Exit(1)
	}

	outFile := ""
	if len(flag.Args()) == 3 {
		outFile = flag.Arg(2)
		ensurePDFExtension(outFile)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.RotateCommand(inFile, outFile, rotation, selectedPages, conf))
}

func parseAfterNUpDetails(nup *model.NUp, argInd int, filenameOut string) []string {
	if nup.PageGrid {
		cols, err := strconv.Atoi(flag.Arg(argInd))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		rows, err := strconv.Atoi(flag.Arg(argInd + 1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		if err = pdfcpu.ParseNUpGridDefinition(cols, rows, nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		argInd += 2
	} else {
		n, err := strconv.Atoi(flag.Arg(argInd))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		if err = pdfcpu.ParseNUpValue(n, nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		argInd++
	}

	filenameIn := flag.Arg(argInd)
	if !hasPDFExtension(filenameIn) && !model.ImageFileName(filenameIn) {
		fmt.Fprintf(os.Stderr, "inFile has to be a PDF or one or a sequence of image files: %s\n", filenameIn)
		os.Exit(1)
	}

	filenamesIn := []string{filenameIn}

	if hasPDFExtension(filenameIn) {
		if len(flag.Args()) > argInd+1 {
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
		for i := argInd + 1; i < len(flag.Args()); i++ {
			arg := flag.Args()[i]
			ensureImageExtension(arg)
			filenamesIn = append(filenamesIn, arg)
		}
	}

	return filenamesIn
}

func processNUpCommand(conf *model.Configuration) {
	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageNUp)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	nup := model.DefaultNUpConfig()
	nup.InpUnit = conf.Unit
	argInd := 1

	outFile := flag.Arg(0)
	if !hasPDFExtension(outFile) {
		// pdfcpu nup description outFile n inFile|imageFiles...
		if err = pdfcpu.ParseNUpDetails(flag.Arg(0), nup); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu nup outFile n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles := parseAfterNUpDetails(nup, argInd, outFile)
	process(cli.NUpCommand(inFiles, outFile, pages, nup, conf))
}

func processGridCommand(conf *model.Configuration) {
	if len(flag.Args()) < 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageGrid)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
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
			os.Exit(1)
		}
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu grid outFile m n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles := parseAfterNUpDetails(nup, argInd, outFile)
	process(cli.NUpCommand(inFiles, outFile, pages, nup, conf))
}

func processBookletCommand(conf *model.Configuration) {
	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageBooklet)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	pages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
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
			os.Exit(1)
		}
		outFile = flag.Arg(1)
		ensurePDFExtension(outFile)
		argInd = 2
	} // else first argument is outFile.

	// pdfcpu booklet outFile n inFile|imageFiles...
	// If no optional 'description' argument provided use default nup configuration.

	inFiles := parseAfterNUpDetails(nup, argInd, outFile)
	n := nup.Grid.Width * nup.Grid.Height
	if n != 2 && n != 4 {
		fmt.Fprintf(os.Stderr, "%s\n", errInvalidBookletID)
		os.Exit(1)
	}
	process(cli.BookletCommand(inFiles, outFile, pages, nup, conf))
}

func processDiplayUnit(conf *model.Configuration) {
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

func processInfoCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageInfo)
		os.Exit(1)
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

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	process(cli.InfoCommand(inFiles, selectedPages, json, conf))
}

func processListFontsCommand(conf *model.Configuration) {
	process(cli.ListFontsCommand(conf))
}

func processInstallFontsCommand(conf *model.Configuration) {
	fileNames := []string{}
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", "expecting a list of TrueType filenames (.ttf, .ttc) for installation.")
		os.Exit(1)
	}
	for _, arg := range flag.Args() {
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

func processCreateCheatSheetFontsCommand(conf *model.Configuration) {
	fileNames := []string{}
	if len(flag.Args()) > 0 {
		fileNames = append(fileNames, flag.Args()...)
	}
	process(cli.CreateCheatSheetsFontsCommand(fileNames, conf))
}

func processListKeywordsCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageKeywordsList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListKeywordsCommand(inFile, conf))
}

func processAddKeywordsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageKeywordsAdd)
		os.Exit(1)
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
}

func processRemoveKeywordsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageKeywordsRemove)
		os.Exit(1)
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
}

func processListPropertiesCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePropertiesList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPropertiesCommand(inFile, conf))
}

func processAddPropertiesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesAdd)
		os.Exit(1)
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

func processRemovePropertiesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePropertiesRemove)
		os.Exit(1)
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
}

func processCollectCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCollect)
		os.Exit(1)
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
		os.Exit(1)
	}

	process(cli.CollectCommand(inFile, outFile, selectedPages, conf))
}

func processListBoxesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesList)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListBoxesCommand(inFile, selectedPages, pb, conf))
}

func processAddBoxesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesAdd)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	pb, err := api.PageBoundaries(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing page boundaries: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}

	process(cli.AddBoxesCommand(inFile, outFile, selectedPages, pb, conf))
}

func processRemoveBoxesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageBoxesRemove)
		os.Exit(1)
	}

	pb, err := api.PageBoundariesFromBoxList(flag.Arg(0))
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
		os.Exit(1)
	}

	process(cli.RemoveBoxesCommand(inFile, outFile, selectedPages, pb, conf))
}

func processCropCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageCrop)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	box, err := api.Box(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem parsing box definition: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}

	process(cli.CropCommand(inFile, outFile, selectedPages, box, conf))
}

func processListAnnotationsCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAnnotsList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
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
func processRemoveAnnotationsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 {
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
}

func processListImagesCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageImagesList)
		os.Exit(1)
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

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ListImagesCommand(inFiles, selectedPages, conf))
}

func processDumpCommand(conf *model.Configuration) {
	s := "No dump for you! - One year!\n\n"
	if len(flag.Args()) != 3 {
		fmt.Fprintln(os.Stderr, s)
		os.Exit(1)
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
		os.Exit(1)
	}
	vals[1] = objNr

	inFile := flag.Arg(2)
	ensurePDFExtension(inFile)

	conf.ValidationMode = model.ValidationRelaxed

	process(cli.DumpCommand(inFile, vals, conf))
}

func processCreateCommand(conf *model.Configuration) {
	if len(flag.Args()) <= 1 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCreate)
		os.Exit(1)
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
}

func processListFormFieldsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormListFields)
		os.Exit(1)
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

	process(cli.ListFormFieldsCommand(inFiles, conf))
}

func processRemoveFormFieldsCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormRemoveFields)
		os.Exit(1)
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
			os.Exit(1)
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
}

func processLockFormCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormLock)
		os.Exit(1)
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
}

func processUnlockFormCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormUnlock)
		os.Exit(1)
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
}

func processResetFormCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormReset)
		os.Exit(1)
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
}

func processExportFormCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormExport)
		os.Exit(1)
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
}

func processFillFormCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormFill)
		os.Exit(1)
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
}

func processMultiFillFormCommand(conf *model.Configuration) {
	if mode == "" {
		mode = "single"
	}
	mode = modeCompletion(mode, []string{"single", "merge"})
	if mode == "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormMultiFill)
		os.Exit(1)
	}

	if len(flag.Args()) < 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageFormMultiFill)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	inFileData := flag.Arg(1)
	if !hasJSONExtension(inFileData) && !hasCSVExtension(inFileData) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".json\" or \".csv\".\n", inFileData)
		os.Exit(1)
	}

	outDir := flag.Arg(2)

	outFile := inFile
	if len(flag.Args()) == 4 {
		outFile = flag.Arg(3)
		ensurePDFExtension(outFile)
	}

	process(cli.MultiFillFormCommand(inFile, inFileData, outDir, outFile, mode == "merge", conf))
}

func processResizeCommand(conf *model.Configuration) {
	if len(flag.Args()) < 2 || len(flag.Args()) > 3 {
		fmt.Fprintf(os.Stderr, "%s\n", usageResize)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	rc, err := pdfcpu.ParseResizeConfig(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}

	process(cli.ResizeCommand(inFile, outFile, selectedPages, rc, conf))
}

func processPosterCommand(conf *model.Configuration) {
	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n", usagePoster)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	// formsize(=papersize) or dimensions, optionally: scalefactor, border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForPoster(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := flag.Arg(2)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var outFile string
	if len(flag.Args()) == 4 {
		outFile = flag.Arg(3)
	}

	process(cli.PosterCommand(inFile, outDir, outFile, selectedPages, cut, conf))
}

func processNDownCommand(conf *model.Configuration) {
	if len(flag.Args()) < 3 || len(flag.Args()) > 5 {
		fmt.Fprintf(os.Stderr, "%s\n", usageNDown)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var inFile, outDir string

	n, err := strconv.Atoi(flag.Arg(0))
	if err == nil {
		// pdfcpu ndown n inFile outDir outFile

		// Optionally: border, margin, bgcolor
		cut, err := pdfcpu.ParseCutConfigForN(n, "", conf.Unit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
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
		os.Exit(1)
	}

	// Optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfigForN(n, flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
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
}

func processCutCommand(conf *model.Configuration) {
	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n", usageCut)
		os.Exit(1)
	}

	processDiplayUnit(conf)

	// required: at least one of horizontalCut, verticalCut
	// optionally: border, margin, bgcolor
	cut, err := pdfcpu.ParseCutConfig(flag.Arg(0), conf.Unit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	inFile := flag.Arg(1)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	outDir := flag.Arg(2)

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	var outFile string
	if len(flag.Args()) >= 4 {
		outFile = flag.Arg(3)
	}

	process(cli.CutCommand(inFile, outDir, outFile, selectedPages, cut, conf))
}

func processListBookmarksCommand(conf *model.Configuration) {
	if len(flag.Args()) < 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListBookmarksCommand(inFile, conf))
}

func processExportBookmarksCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksExport)
		os.Exit(1)
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
}

func processImportBookmarksCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksImport)
		os.Exit(1)
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
}

func processRemoveBookmarksCommand(conf *model.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageBookmarksExport)
		os.Exit(1)
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
}

func processListPageLayoutCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPageLayoutCommand(inFile, conf))
}

func processSetPageLayoutCommand(conf *model.Configuration) {
	if len(flag.Args()) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutSet)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	v := flag.Arg(1)

	if !validate.DocumentPageLayout(v) {
		fmt.Fprintln(os.Stderr, "invalid page layout, use one of: SinglePage, TwoColumnLeft, TwoColumnRight, TwoPageLeft, TwoPageRight")
		os.Exit(1)
	}

	process(cli.SetPageLayoutCommand(inFile, "", v, conf))
}

func processResetPageLayoutCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageLayoutReset)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetPageLayoutCommand(inFile, "", conf))
}

func processListPageModeCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPageModeCommand(inFile, conf))
}

func processSetPageModeCommand(conf *model.Configuration) {
	if len(flag.Args()) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeSet)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	v := flag.Arg(1)

	if !validate.DocumentPageMode(v) {
		fmt.Fprintln(os.Stderr, "invalid page mode, use one of: UseNone, UseOutlines, UseThumbs, FullScreen, UseOC, UseAttachments")
		os.Exit(1)
	}

	process(cli.SetPageModeCommand(inFile, "", v, conf))
}

func processResetPageModeCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePageModeReset)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetPageModeCommand(inFile, "", conf))
}

func processListViewerPreferencesCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListViewerPreferencesCommand(inFile, all, json, conf))
}

func processSetViewerPreferencesCommand(conf *model.Configuration) {
	if len(flag.Args()) != 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesSet)
		os.Exit(1)
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
}

func processResetViewerPreferencesCommand(conf *model.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageViewerPreferencesReset)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ResetViewerPreferencesCommand(inFile, "", conf))
}

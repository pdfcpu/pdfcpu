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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
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

func printHelp(conf *pdfcpu.Configuration) {
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

func printConfiguration(conf *pdfcpu.Configuration) {
	fmt.Fprintf(os.Stdout, "config: %s\n", conf.Path)
	f, err := os.Open(conf.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't open %s", conf.Path)
		os.Exit(1)
	}
	defer f.Close()
	bb, err := io.ReadAll(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't read %s", conf.Path)
		os.Exit(1)
	}
	fmt.Print(string(bb))
}

func printPaperSizes(conf *pdfcpu.Configuration) {
	fmt.Fprintln(os.Stderr, paperSizes)
}

func printSelectedPages(conf *pdfcpu.Configuration) {
	fmt.Fprintln(os.Stderr, usagePageSelection)
}

func printVersion(conf *pdfcpu.Configuration) {
	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "pdfcpu: %v\n", pdfcpu.VersionStr)
	if date != "?" {
		fmt.Fprintf(os.Stdout, "build : %v\ncommit: %v\n", date, commit)
	}
	if verbose {
		fmt.Fprintf(os.Stdout, "config: %s\n", conf.Path)
	}
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
	os.Exit(0)
}
func processValidateCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) == 0 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
	}

	filesIn := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			filesIn = append(filesIn, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		filesIn = append(filesIn, arg)
	}

	if mode != "" && mode != "strict" && mode != "s" && mode != "relaxed" && mode != "r" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
	}

	switch mode {
	case "strict", "s":
		conf.ValidationMode = pdfcpu.ValidationStrict
	case "relaxed", "r":
		conf.ValidationMode = pdfcpu.ValidationRelaxed
	}

	if links {
		conf.ValidateLinks = true
	}

	process(cli.ValidateCommand(filesIn, conf))
}

func processOptimizeCommand(conf *pdfcpu.Configuration) {
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

func processSplitCommand(conf *pdfcpu.Configuration) {
	if mode == "" {
		mode = "span"
	}
	mode = extractModeCompletion(mode, []string{"span", "bookmark"})
	if mode == "" || len(flag.Args()) < 2 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
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

	outDir := flag.Arg(1)

	process(cli.SplitCommand(inFile, outDir, span, conf))
}

func processMergeCommand(conf *pdfcpu.Configuration) {
	if mode == "" {
		mode = "create"
	}
	mode = extractModeCompletion(mode, []string{"create", "append"})
	if mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	if len(flag.Args()) < 2 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	filesIn := []string{}
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
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			filesIn = append(filesIn, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		filesIn = append(filesIn, arg)
	}

	if sorted {
		sort.Strings(filesIn)
	}

	var cmd *cli.Command

	switch mode {

	case "create":
		cmd = cli.MergeCreateCommand(filesIn, outFile, conf)

	case "append":
		cmd = cli.MergeAppendCommand(filesIn, outFile, conf)
	}

	process(cmd)
}

func extractModeCompletion(modePrefix string, modes []string) string {
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

func processExtractCommand(conf *pdfcpu.Configuration) {
	mode = extractModeCompletion(mode, []string{"image", "font", "page", "content", "meta"})
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

func processTrimCommand(conf *pdfcpu.Configuration) {
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

func processListAttachmentsCommand(conf *pdfcpu.Configuration) {
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

func processAddAttachmentsCommand(conf *pdfcpu.Configuration) {
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

func processAddAttachmentsPortfolioCommand(conf *pdfcpu.Configuration) {
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

func processRemoveAttachmentsCommand(conf *pdfcpu.Configuration) {
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

func processExtractAttachmentsCommand(conf *pdfcpu.Configuration) {
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

func processListPermissionsCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePermList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	process(cli.ListPermissionsCommand(inFile, conf))
}

func permCompletion(permPrefix string) string {
	var permStr string
	for _, perm := range []string{"none", "all"} {
		if !strings.HasPrefix(perm, permPrefix) {
			continue
		}
		if len(permStr) > 0 {
			return ""
		}
		permStr = perm
	}

	return permStr
}

func processSetPermissionsCommand(conf *pdfcpu.Configuration) {
	if perm != "" {
		perm = permCompletion(perm)
	}
	if len(flag.Args()) != 1 || selectedPages != "" ||
		!(perm == "none" || perm == "all") {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermSet)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}

	if perm == "all" {
		conf.Permissions = pdfcpu.PermissionsAll
	}

	process(cli.SetPermissionsCommand(inFile, "", conf))
}

func processDecryptCommand(conf *pdfcpu.Configuration) {
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
	if !pdfcpu.MemberOf(mode, []string{"rc4", "aes", ""}) {
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
	if perm != "none" && perm != "all" && perm != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported permissions: none,all default:none (viewing always allowed!)")
		os.Exit(1)
	}
}

func processEncryptCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

	if conf.OwnerPW == "" {
		fmt.Fprintln(os.Stderr, "missing non-empty owner password!")
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

	validateEncryptFlags()

	conf.EncryptUsingAES = mode != "rc4"

	kl, _ := strconv.Atoi(key)
	conf.EncryptKeyLength = kl

	if perm == "all" {
		conf.Permissions = pdfcpu.PermissionsAll
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

func processChangeUserPasswordCommand(conf *pdfcpu.Configuration) {
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

func processChangeOwnerPasswordCommand(conf *pdfcpu.Configuration) {
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

func addWatermarks(conf *pdfcpu.Configuration, onTop bool) {
	u := usageWatermarkAdd
	if onTop {
		u = usageStampAdd
	}

	if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", u)
		os.Exit(1)
	}

	if mode != "text" && mode != "image" && mode != "pdf" {
		fmt.Fprintln(os.Stderr, "mode has to be one of: text, image or pdf")
		os.Exit(1)
	}

	processDiplayUnit(conf)

	var (
		wm  *pdfcpu.Watermark
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

func processAddStampsCommand(conf *pdfcpu.Configuration) {
	addWatermarks(conf, true)
}

func processAddWatermarksCommand(conf *pdfcpu.Configuration) {
	addWatermarks(conf, false)
}

func updateWatermarks(conf *pdfcpu.Configuration, onTop bool) {
	u := usageWatermarkAdd
	if onTop {
		u = usageStampAdd
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
		wm  *pdfcpu.Watermark
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

func processUpdateStampsCommand(conf *pdfcpu.Configuration) {
	updateWatermarks(conf, true)
}

func processUpdateWatermarksCommand(conf *pdfcpu.Configuration) {
	updateWatermarks(conf, false)
}

func removeWatermarks(conf *pdfcpu.Configuration, onTop bool) {
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

func processRemoveStampsCommand(conf *pdfcpu.Configuration) {
	removeWatermarks(conf, true)
}

func processRemoveWatermarksCommand(conf *pdfcpu.Configuration) {
	removeWatermarks(conf, false)
}

func ensureImageExtension(filename string) {
	if !pdfcpu.ImageFileName(filename) {
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

func processImportImagesCommand(conf *pdfcpu.Configuration) {
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

func processInsertPagesCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesInsert)
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

func processRemovePagesCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || selectedPages == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesRemove)
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

func processRotateCommand(conf *pdfcpu.Configuration) {
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

func parseAfterNUpDetails(nup *pdfcpu.NUp, argInd int, filenameOut string) []string {
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
	if !hasPDFExtension(filenameIn) && !pdfcpu.ImageFileName(filenameIn) {
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

func processNUpCommand(conf *pdfcpu.Configuration) {
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

	nup := pdfcpu.DefaultNUpConfig()
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

func processGridCommand(conf *pdfcpu.Configuration) {
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

	nup := pdfcpu.DefaultNUpConfig()
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

func processBookletCommand(conf *pdfcpu.Configuration) {
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

func processDiplayUnit(conf *pdfcpu.Configuration) {
	if !pdfcpu.MemberOf(unit, []string{"", "points", "po", "inches", "in", "cm", "mm"}) {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported units: (po)ints, (in)ches, cm, mm")
		os.Exit(1)
	}

	switch unit {
	case "points", "po":
		conf.Unit = pdfcpu.POINTS
	case "inches", "in":
		conf.Unit = pdfcpu.INCHES
	case "cm":
		conf.Unit = pdfcpu.CENTIMETRES
	case "mm":
		conf.Unit = pdfcpu.MILLIMETRES
	}
}

func processInfoCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageInfo)
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

	processDiplayUnit(conf)

	process(cli.InfoCommand(inFile, selectedPages, conf))
}

func processListFontsCommand(conf *pdfcpu.Configuration) {
	process(cli.ListFontsCommand(conf))
}

func processInstallFontsCommand(conf *pdfcpu.Configuration) {
	fileNames := []string{}
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", "expecting a list of TrueType filenames (.ttf, .ttc) for installation.")
		os.Exit(1)
	}
	for _, arg := range flag.Args() {
		if !pdfcpu.MemberOf(filepath.Ext(arg), []string{".ttf", ".ttc"}) {
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

func processCreateCheatSheetFontsCommand(conf *pdfcpu.Configuration) {
	fileNames := []string{}
	if len(flag.Args()) > 0 {
		fileNames = append(fileNames, flag.Args()...)
	}
	process(cli.CreateCheatSheetsFontsCommand(fileNames, conf))
}

func processListKeywordsCommand(conf *pdfcpu.Configuration) {
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

func processAddKeywordsCommand(conf *pdfcpu.Configuration) {
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

func processRemoveKeywordsCommand(conf *pdfcpu.Configuration) {
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

func processListPropertiesCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) != 1 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageKeywordsList)
		os.Exit(1)
	}

	inFile := flag.Arg(0)
	if conf.CheckFileNameExt {
		ensurePDFExtension(inFile)
	}
	process(cli.ListPropertiesCommand(inFile, conf))
}

func processAddPropertiesCommand(conf *pdfcpu.Configuration) {
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

func processRemovePropertiesCommand(conf *pdfcpu.Configuration) {
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

func processCollectCommand(conf *pdfcpu.Configuration) {
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

func processListBoxesCommand(conf *pdfcpu.Configuration) {
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

func processAddBoxesCommand(conf *pdfcpu.Configuration) {
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

func processRemoveBoxesCommand(conf *pdfcpu.Configuration) {
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

func processCropCommand(conf *pdfcpu.Configuration) {
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

func processListAnnotationsCommand(conf *pdfcpu.Configuration) {
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
func processRemoveAnnotationsCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAnnotsRemove)
		os.Exit(1)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	inFile := ""
	objNrs := []int{}

	for i, arg := range flag.Args() {
		if i == 0 {
			inFile = arg
			if conf.CheckFileNameExt {
				ensurePDFExtension(inFile)
			}
			continue
		}
		i, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "objNr has to be a positiv numeric value")
			os.Exit(1)
		}
		objNrs = append(objNrs, i)
	}

	process(cli.RemoveAnnotationsCommand(inFile, "", selectedPages, objNrs, conf))
}

func processListImagesCommand(conf *pdfcpu.Configuration) {
	filesIn := []string{}
	for _, arg := range flag.Args() {
		if strings.Contains(arg, "*") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
			filesIn = append(filesIn, matches...)
			continue
		}
		if conf.CheckFileNameExt {
			ensurePDFExtension(arg)
		}
		filesIn = append(filesIn, arg)
	}

	selectedPages, err := api.ParsePageSelection(selectedPages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag selectedPages: %v\n", err)
		os.Exit(1)
	}

	process(cli.ListImagesCommand(filesIn, selectedPages, conf))
}

func processCreateCommand(conf *pdfcpu.Configuration) {
	if len(flag.Args()) <= 1 || len(flag.Args()) > 3 || selectedPages != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageCreate)
		os.Exit(1)
	}

	var inFile string

	inJSONFile := flag.Arg(0)
	ensureJSONExtension(inJSONFile)

	outFile := flag.Arg(1)
	ensurePDFExtension(outFile)

	if len(flag.Args()) == 3 {
		inFile = outFile
		outFile = flag.Arg(2)
		if conf.CheckFileNameExt {
			ensurePDFExtension(outFile)
		}
	}

	process(cli.CreateCommand(inJSONFile, inFile, outFile, conf))
}

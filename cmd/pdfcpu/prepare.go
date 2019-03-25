/*
Copyright 2018 The pdfcpu Authors.

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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/api"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

func hasPdfExtension(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".pdf")
}

func ensurePdfExtension(filename string) {
	if !hasPdfExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", filename)
		os.Exit(1)
	}
}

func defaultFilenameOut(filename string) string {
	return filename[:len(filename)-4] + "_new.pdf"
}

func printHelp(config *pdfcpu.Configuration) *api.Command {

	switch len(flag.Args()) {

	case 0:
		fmt.Fprintln(os.Stderr, usage)

	case 1:
		fmt.Fprintln(os.Stderr, cmdMap.HelpString(flag.Arg(0)))

	default:
		fmt.Fprintln(os.Stderr, "usage: pdfcpu help command\n\nToo many arguments.")

	}

	return nil
}

func printPaperSizes(config *pdfcpu.Configuration) *api.Command {

	fmt.Fprintln(os.Stderr, paperSizes)

	return nil
}

func printVersion(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "pdfcpu version %s\n", pdfcpu.PDFCPUVersion)

	return nil
}

func prepareValidateCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 1 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	if mode != "" && mode != "strict" && mode != "s" && mode != "relaxed" && mode != "r" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageValidate)
		os.Exit(1)
	}

	switch mode {
	case "strict", "s":
		config.ValidationMode = pdfcpu.ValidationStrict
	case "relaxed", "r":
		config.ValidationMode = pdfcpu.ValidationRelaxed
	}

	return api.ValidateCommand(filenameIn, config)
}

func prepareOptimizeCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageOptimize)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := defaultFilenameOut(filenameIn)
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	config.StatsFileName = fileStats
	if len(fileStats) > 0 {
		fmt.Fprintf(os.Stdout, "stats will be appended to %s\n", fileStats)
	}

	return api.OptimizeCommand(filenameIn, filenameOut, config)
}

func prepareSplitCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 2 || len(flag.Args()) > 3 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	dirnameOut := flag.Arg(1)

	span := 1
	var err error
	if len(flag.Args()) == 3 {
		span, err = strconv.Atoi(flag.Arg(2))
		if err != nil || span < 1 {
			fmt.Fprintln(os.Stderr, "split: span is a numeric value >= 1")
			os.Exit(1)
		}
	}

	return api.SplitCommand(filenameIn, dirnameOut, span, config)
}

func prepareMergeCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 3 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageMerge)
		os.Exit(1)
	}

	var filenameOut string
	filenamesIn := []string{}
	for i, arg := range flag.Args() {
		if i == 0 {
			filenameOut = arg
			ensurePdfExtension(filenameOut)
			continue
		}
		ensurePdfExtension(arg)
		filenamesIn = append(filenamesIn, arg)
	}

	return api.MergeCommand(filenamesIn, filenameOut, config)
}

func extractModeCompletion(modePrefix string) string {

	var modeStr string

	for _, mode := range []string{"image", "font", "page", "content", "meta"} {
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

func prepareExtractCommand(config *pdfcpu.Configuration) *api.Command {

	mode = extractModeCompletion(mode)

	if len(flag.Args()) != 2 || mode == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageExtract)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	dirnameOut := flag.Arg(1)

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}

	var cmd *api.Command

	switch mode {

	case "image":
		cmd = api.ExtractImagesCommand(filenameIn, dirnameOut, pages, config)

	case "font":
		cmd = api.ExtractFontsCommand(filenameIn, dirnameOut, pages, config)

	case "page":
		cmd = api.ExtractPagesCommand(filenameIn, dirnameOut, pages, config)

	case "content":
		cmd = api.ExtractContentCommand(filenameIn, dirnameOut, pages, config)

	case "meta":
		cmd = api.ExtractMetadataCommand(filenameIn, dirnameOut, config)
	}

	return cmd
}

func prepareTrimCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageTrim)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := defaultFilenameOut(filenameIn)
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	return api.TrimCommand(filenameIn, filenameOut, pages, config)
}

func prepareListAttachmentsCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) != 1 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAttachList)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	return api.ListAttachmentsCommand(filenameIn, config)
}

func prepareAddAttachmentsCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachAdd)
		os.Exit(1)
	}

	var filenameIn string
	filenames := []string{}

	for i, arg := range flag.Args() {
		if i == 0 {
			filenameIn = arg
			ensurePdfExtension(filenameIn)
			continue
		}
		filenames = append(filenames, arg)
	}

	return api.AddAttachmentsCommand(filenameIn, filenames, config)
}

func prepareRemoveAttachmentsCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 1 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachRemove)
		os.Exit(1)
	}

	var filenameIn string
	filenames := []string{}

	for i, arg := range flag.Args() {
		if i == 0 {
			filenameIn = arg
			ensurePdfExtension(filenameIn)
			continue
		}
		filenames = append(filenames, arg)
	}

	return api.RemoveAttachmentsCommand(filenameIn, filenames, config)
}

func prepareExtractAttachmentsCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usageAttachExtract)
		os.Exit(1)
	}

	var filenameIn, dirnameOut string
	filenames := []string{}

	for i, arg := range flag.Args() {
		if i == 0 {
			filenameIn = arg
			ensurePdfExtension(filenameIn)
			continue
		}
		if i == 1 {
			dirnameOut = arg
			continue
		}
		filenames = append(filenames, arg)
	}

	return api.ExtractAttachmentsCommand(filenameIn, dirnameOut, filenames, config)
}

func prepareListPermissionsCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) != 1 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePermList)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	return api.ListPermissionsCommand(filenameIn, config)
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
		permStr = mode
	}

	return permStr
}

func prepareAddPermissionsCommand(config *pdfcpu.Configuration) *api.Command {

	if perm != "" {
		perm = permCompletion(perm)
	}

	if len(flag.Args()) != 1 || pageSelection != "" ||
		!(perm == "" || perm == "none" || perm == "all") {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermAdd)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	if perm == "all" {
		config.UserAccessPermissions = pdfcpu.PermissionsAll
	}

	return api.AddPermissionsCommand(filenameIn, config)
}

func prepareDecryptCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageDecrypt)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := filenameIn
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	return api.DecryptCommand(filenameIn, filenameOut, config)
}

func validateEncryptFlags() {

	if mode != "rc4" && mode != "aes" && mode != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "valid modes: rc4,aes default:aes")
		os.Exit(1)
	}

	if key != "40" && key != "128" && key != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported key lengths: 40,128 default:128")
		os.Exit(1)
	}

	if perm != "none" && perm != "all" && perm != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", "supported permissions: none,all default:none (viewing always allowed!)")
		os.Exit(1)
	}
}

func prepareEncryptCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

	validateEncryptFlags()

	if mode == "rc4" {
		config.EncryptUsingAES = false
	}

	if key == "40" {
		config.EncryptUsing128BitKey = false
	}

	if perm == "all" {
		config.UserAccessPermissions = pdfcpu.PermissionsAll
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)
	filenameOut := filenameIn
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	return api.EncryptCommand(filenameIn, filenameOut, config)
}

func prepareChangeUserPasswordCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeUserPW)
		os.Exit(1)
	}

	pwOld := flag.Arg(1)
	pwNew := flag.Arg(2)

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := filenameIn

	return api.ChangeUserPWCommand(filenameIn, filenameOut, config, &pwOld, &pwNew)
}

func prepareChangeOwnerPasswordCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		os.Exit(1)
	}

	pwOld := flag.Arg(1)
	pwNew := flag.Arg(2)

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := filenameIn

	return api.ChangeOwnerPWCommand(filenameIn, filenameOut, config, &pwOld, &pwNew)
}

func prepareChangePasswordCommand(config *pdfcpu.Configuration, s string) *api.Command {

	var cmd *api.Command

	switch s {

	case "changeupw":
		cmd = prepareChangeUserPasswordCommand(config)

	case "changeopw":
		cmd = prepareChangeOwnerPasswordCommand(config)

	}

	return cmd
}

func prepareWatermarksCommand(config *pdfcpu.Configuration, onTop bool) *api.Command {

	if len(flag.Args()) < 2 || len(flag.Args()) > 3 {
		s := usageWatermark
		if onTop {
			s = usageStamp
		}
		fmt.Fprintf(os.Stderr, "%s\n\n", s)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v", err)
		os.Exit(1)
	}

	//fmt.Printf("details: <%s>\n", flag.Arg(0))
	wm, err := pdfcpu.ParseWatermarkDetails(flag.Arg(0), onTop)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	filenameIn := flag.Arg(1)
	ensurePdfExtension(filenameIn)

	filenameOut := defaultFilenameOut(filenameIn)
	if len(flag.Args()) == 3 {
		filenameOut = flag.Arg(2)
		ensurePdfExtension(filenameOut)
	}

	return api.AddWatermarksCommand(filenameIn, filenameOut, pages, wm, config)
}

func prepareAddStampsCommand(config *pdfcpu.Configuration) *api.Command {
	return prepareWatermarksCommand(config, true)
}

func prepareAddWatermarksCommand(config *pdfcpu.Configuration) *api.Command {
	return prepareWatermarksCommand(config, false)
}

func hasImageExtension(filename string) bool {
	s := strings.ToLower(filepath.Ext(filename))
	return pdfcpu.MemberOf(s, []string{".jpg", ".jpeg", ".png", ".tif", ".tiff"})
}

func ensureImageExtension(filename string) {
	if !hasImageExtension(filename) {
		fmt.Fprintf(os.Stderr, "%s needs an image extension (.jpg, .jpeg, .png, .tif, .tiff)\n", filename)
		os.Exit(1)
	}
}

func prepareImportImagesCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImportImages)
		os.Exit(1)
	}

	filenameOut := flag.Arg(0)
	if hasPdfExtension(filenameOut) {
		// pdfcpu import outFile imageFile...
		// No optional 'description' argument provided.
		// We use the default import configuration.
		imp := pdfcpu.DefaultImportConfig()
		filenamesIn := []string{}
		for i := 1; i < len(flag.Args()); i++ {
			arg := flag.Arg(i)
			ensureImageExtension(arg)
			filenamesIn = append(filenamesIn, arg)
		}
		return api.ImportImagesCommand(filenamesIn, filenameOut, imp, config)
	}

	// pdfcpu import outFile imageFile
	// Possibly unexpected 'description'
	if len(flag.Args()) == 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageImportImages)
		os.Exit(1)
	}

	// pdfcpu import description outFile imageFile...
	//fmt.Printf("details: <%s>\n", flag.Arg(0))
	imp, err := pdfcpu.ParseImportDetails(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if imp == nil {
		fmt.Fprintf(os.Stderr, "missing import description\n")
		os.Exit(1)
	}

	filenameOut = flag.Arg(1)
	ensurePdfExtension(filenameOut)

	filenamesIn := []string{}
	for i := 2; i < len(flag.Args()); i++ {
		arg := flag.Args()[i]
		ensureImageExtension(arg)
		filenamesIn = append(filenamesIn, arg)
	}

	return api.ImportImagesCommand(filenamesIn, filenameOut, imp, config)
}

func prepareInsertPagesCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesInsert)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := defaultFilenameOut(filenameIn)
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}

	return api.InsertPagesCommand(filenameIn, filenameOut, pages, config)
}

func prepareRemovePagesCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usagePagesRemove)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := defaultFilenameOut(filenameIn)
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}
	if pages == nil {
		fmt.Fprintf(os.Stderr, "missing page selection\n")
		os.Exit(1)
	}

	return api.RemovePagesCommand(filenameIn, filenameOut, pages, config)
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func prepareRotateCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageRotate)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	r, err := strconv.Atoi(flag.Arg(1))
	if err != nil || abs(r)%90 > 0 {
		fmt.Fprintf(os.Stderr, "rotation must be a multiple of 90: %s\n", flag.Arg(1))
		os.Exit(1)
	}

	return api.RotateCommand(filenameIn, r, pages, config)
}

func parseAfterNUpDetails(nup *pdfcpu.NUp, argInd int, filenameOut string) []string {

	var err error

	if nup.PageGrid {
		err = pdfcpu.ParseNUpGridDefinition(flag.Arg(argInd), flag.Arg(argInd+1), nup)
		argInd += 2
	} else {
		err = pdfcpu.ParseNUpValue(flag.Arg(argInd), nup)
		argInd++
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	filenameIn := flag.Arg(argInd)
	if !hasPdfExtension(filenameIn) && !hasImageExtension(filenameIn) {
		fmt.Fprintf(os.Stderr, "Inputfile has to be a PDF or one or a sequence of image files: %s\n", filenameIn)
		os.Exit(1)
	}

	filenamesIn := []string{filenameIn}

	if hasPdfExtension(filenameIn) {
		if len(flag.Args()) > argInd+1 {
			usage := usageNUp
			if nup.PageGrid {
				usage = usageGrid
			}
			fmt.Fprintf(os.Stderr, "%s\n\n", usage)
			os.Exit(1)
		}
		if filenameIn == filenameOut {
			fmt.Fprintln(os.Stderr, "input and output pdf should be different.")
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

func prepareNUpCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageNUp)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}

	nup := pdfcpu.DefaultNUpConfig()

	filenameOut := flag.Arg(0)
	if hasPdfExtension(filenameOut) {
		// pdfcpu nup outFile n inFile|imageFiles...
		// No optional 'description' argument provided.
		// We use the default nup configuration.
		filenamesIn := parseAfterNUpDetails(nup, 1, filenameOut)
		return api.NUpCommand(filenamesIn, filenameOut, pages, nup, config)
	}

	// pdfcpu nup description outFile n inFile|imageFiles...
	//fmt.Printf("details: <%s>\n", flag.Arg(0))
	err = pdfcpu.ParseNUpDetails(flag.Arg(0), nup)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	filenameOut = flag.Arg(1)
	ensurePdfExtension(filenameOut)

	filenamesIn := parseAfterNUpDetails(nup, 2, filenameOut)
	return api.NUpCommand(filenamesIn, filenameOut, pages, nup, config)
}

func prepareGridCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) < 4 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageGrid)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "problem with flag pageSelection: %v\n", err)
		os.Exit(1)
	}

	nup := pdfcpu.DefaultNUpConfig()
	nup.PageGrid = true

	filenameOut := flag.Arg(0)
	if hasPdfExtension(filenameOut) {
		// pdfcpu grid outFile m n inFile|imageFiles...
		// No optional 'description' argument provided.
		// We use the default nup configuration.
		filenamesIn := parseAfterNUpDetails(nup, 1, filenameOut)
		return api.NUpCommand(filenamesIn, filenameOut, pages, nup, config)
	}

	// pdfcpu grid description outFile m n inFile|imageFiles...
	//fmt.Printf("details: <%s>\n", flag.Arg(0))
	err = pdfcpu.ParseNUpDetails(flag.Arg(0), nup)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	filenameOut = flag.Arg(1)
	ensurePdfExtension(filenameOut)

	filenamesIn := parseAfterNUpDetails(nup, 2, filenameOut)
	return api.NUpCommand(filenamesIn, filenameOut, pages, nup, config)
}

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
	"log"
	"os"

	"github.com/hhrutter/pdfcpu/pkg/api"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

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

	if len(flag.Args()) != 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	dirnameOut := flag.Arg(1)

	return api.SplitCommand(filenameIn, dirnameOut, config)
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

func allowedExtracMode(s string) bool {

	return mode == "image" || mode == "font" || mode == "page" || mode == "content" || mode == "meta" ||
		mode == "i" || mode == "p" || mode == "c" || mode == "m"
}

func prepareExtractCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) != 2 || mode == "" || !allowedExtracMode(mode) {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageExtract)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	dirnameOut := flag.Arg(1)

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		log.Fatalf("extract: problem with flag pageSelection: %v", err)
	}

	var cmd *api.Command

	switch mode {

	case "image", "i":
		cmd = api.ExtractImagesCommand(filenameIn, dirnameOut, pages, config)

	case "font":
		cmd = api.ExtractFontsCommand(filenameIn, dirnameOut, pages, config)

	case "page", "p":
		cmd = api.ExtractPagesCommand(filenameIn, dirnameOut, pages, config)

	case "content", "c":
		cmd = api.ExtractContentCommand(filenameIn, dirnameOut, pages, config)

	case "meta", "m":
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
		log.Fatalf("trim: problem with flag pageSelection: %v", err)
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

func prepareAttachmentCommand(config *pdfcpu.Configuration) *api.Command {

	if len(os.Args) == 2 {
		fmt.Fprintln(os.Stderr, usageAttach)
		os.Exit(1)
	}

	var cmd *api.Command

	subCmd := os.Args[2]

	switch subCmd {

	case "list":
		cmd = prepareListAttachmentsCommand(config)

	case "add":
		cmd = prepareAddAttachmentsCommand(config)

	case "remove":
		cmd = prepareRemoveAttachmentsCommand(config)

	case "extract":
		cmd = prepareExtractAttachmentsCommand(config)

	default:
		fmt.Fprintln(os.Stderr, usageAttach)
		os.Exit(1)
	}

	return cmd
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

func prepareAddPermissionsCommand(config *pdfcpu.Configuration) *api.Command {

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

func preparePermissionsCommand(config *pdfcpu.Configuration) *api.Command {

	if len(os.Args) == 2 {
		fmt.Fprintln(os.Stderr, usagePerm)
		os.Exit(1)
	}

	var cmd *api.Command

	subCmd := os.Args[2]

	switch subCmd {

	case "list":
		cmd = prepareListPermissionsCommand(config)

	case "add":
		cmd = prepareAddPermissionsCommand(config)

	default:
		fmt.Fprintln(os.Stderr, usagePerm)
		os.Exit(1)
	}

	return cmd

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

func validEncryptOptions() bool {
	return pageSelection == "" &&
		(mode == "" || mode == "rc4" || mode == "aes") &&
		(key == "" || key == "40" || key == "128") &&
		(perm == "" || perm == "none" || perm == "all")
}

func prepareEncryptCommand(config *pdfcpu.Configuration) *api.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || !validEncryptOptions() {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageEncrypt)
		os.Exit(1)
	}

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
		fmt.Fprintf(os.Stderr, "%s\n\n", usageStamp)
		os.Exit(1)
	}

	pages, err := api.ParsePageSelection(pageSelection)
	if err != nil {
		log.Fatalf("problem with flag pageSelection: %v", err)
	}

	//fmt.Printf("details: <%s>\n", flag.Arg(0))
	wm, err := pdfcpu.ParseWatermarkDetails(flag.Arg(0), onTop)
	if err != nil {
		log.Fatalf("%v", err)
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

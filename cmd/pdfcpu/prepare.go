package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hhrutter/pdfcpu"
	"github.com/hhrutter/pdfcpu/types"
)

func prepareValidateCommand(config *types.Configuration) *pdfcpu.Command {

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
		config.ValidationMode = types.ValidationStrict
	case "relaxed", "r":
		config.ValidationMode = types.ValidationRelaxed
	}

	return pdfcpu.ValidateCommand(filenameIn, config)
}

func prepareOptimizeCommand(config *types.Configuration) *pdfcpu.Command {

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

	return pdfcpu.OptimizeCommand(filenameIn, filenameOut, config)
}

func prepareSplitCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 2 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageSplit)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	dirnameOut := flag.Arg(1)

	return pdfcpu.SplitCommand(filenameIn, dirnameOut, config)
}

func prepareMergeCommand(config *types.Configuration) *pdfcpu.Command {

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

	return pdfcpu.MergeCommand(filenamesIn, filenameOut, config)
}

func prepareExtractCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 2 || mode == "" ||
		(mode != "image" && mode != "font" && mode != "page" && mode != "content") &&
			(mode != "i" && mode != "p" && mode != "c") {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageExtract)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	dirnameOut := flag.Arg(1)

	pages, err := pdfcpu.ParsePageSelection(pageSelection)
	if err != nil {
		log.Fatalf("extract: problem with flag pageSelection: %v", err)
	}

	var cmd *pdfcpu.Command

	switch mode {

	case "image", "i":
		cmd = pdfcpu.ExtractImagesCommand(filenameIn, dirnameOut, pages, config)

	case "font":
		cmd = pdfcpu.ExtractFontsCommand(filenameIn, dirnameOut, pages, config)

	case "page", "p":
		cmd = pdfcpu.ExtractPagesCommand(filenameIn, dirnameOut, pages, config)

	case "content", "c":
		cmd = pdfcpu.ExtractContentCommand(filenameIn, dirnameOut, pages, config)
	}

	return cmd
}

func prepareTrimCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection == "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageTrim)
		os.Exit(1)
	}

	pages, err := pdfcpu.ParsePageSelection(pageSelection)
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

	return pdfcpu.TrimCommand(filenameIn, filenameOut, pages, config)
}

func prepareListAttachmentsCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 1 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usageAttachList)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	return pdfcpu.ListAttachmentsCommand(filenameIn, config)
}

func prepareAddAttachmentsCommand(config *types.Configuration) *pdfcpu.Command {

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

	return pdfcpu.AddAttachmentsCommand(filenameIn, filenames, config)
}

func prepareRemoveAttachmentsCommand(config *types.Configuration) *pdfcpu.Command {

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

	return pdfcpu.RemoveAttachmentsCommand(filenameIn, filenames, config)
}

func prepareExtractAttachmentsCommand(config *types.Configuration) *pdfcpu.Command {

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

	return pdfcpu.ExtractAttachmentsCommand(filenameIn, dirnameOut, filenames, config)
}

func prepareAttachmentCommand(config *types.Configuration) *pdfcpu.Command {

	if len(os.Args) == 2 {
		fmt.Fprintln(os.Stderr, usageAttach)
		os.Exit(1)
	}

	var cmd *pdfcpu.Command

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

func prepareListPermissionsCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 1 || pageSelection != "" {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usagePermList)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	return pdfcpu.ListPermissionsCommand(filenameIn, config)
}

func prepareAddPermissionsCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 1 || pageSelection != "" ||
		!(perm == "" || perm == "none" || perm == "all") {
		fmt.Fprintf(os.Stderr, "usage: %s\n\n", usagePermAdd)
		os.Exit(1)
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	if perm == "all" {
		config.UserAccessPermissions = types.PermissionsAll
	}

	return pdfcpu.AddPermissionsCommand(filenameIn, config)
}

func preparePermissionsCommand(config *types.Configuration) *pdfcpu.Command {

	if len(os.Args) == 2 {
		fmt.Fprintln(os.Stderr, usagePerm)
		os.Exit(1)
	}

	var cmd *pdfcpu.Command

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

func prepareDecryptCommand(config *types.Configuration) *pdfcpu.Command {

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

	return pdfcpu.DecryptCommand(filenameIn, filenameOut, config)
}

func validEncryptOptions() bool {
	return pageSelection == "" &&
		(mode == "" || mode == "rc4" || mode == "aes") &&
		(key == "" || key == "40" || key == "128") &&
		(perm == "" || perm == "none" || perm == "all")
}

func prepareEncryptCommand(config *types.Configuration) *pdfcpu.Command {

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
		config.UserAccessPermissions = types.PermissionsAll
	}

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)
	filenameOut := filenameIn
	if len(flag.Args()) == 2 {
		filenameOut = flag.Arg(1)
		ensurePdfExtension(filenameOut)
	}

	return pdfcpu.EncryptCommand(filenameIn, filenameOut, config)
}

func prepareChangeUserPasswordCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeUserPW)
		os.Exit(1)
	}

	pwOld := flag.Arg(1)
	pwNew := flag.Arg(2)

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := filenameIn

	return pdfcpu.ChangeUserPWCommand(filenameIn, filenameOut, config, &pwOld, &pwNew)
}

func prepareChangeOwnerPasswordCommand(config *types.Configuration) *pdfcpu.Command {

	if len(flag.Args()) != 3 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageChangeOwnerPW)
		os.Exit(1)
	}

	pwOld := flag.Arg(1)
	pwNew := flag.Arg(2)

	filenameIn := flag.Arg(0)
	ensurePdfExtension(filenameIn)

	filenameOut := filenameIn

	return pdfcpu.ChangeOwnerPWCommand(filenameIn, filenameOut, config, &pwOld, &pwNew)
}

func prepareChangePasswordCommand(config *types.Configuration, s string) *pdfcpu.Command {

	var cmd *pdfcpu.Command

	switch s {

	case "changeupw":
		cmd = prepareChangeUserPasswordCommand(config)

	case "changeopw":
		cmd = prepareChangeOwnerPasswordCommand(config)

	}

	return cmd
}

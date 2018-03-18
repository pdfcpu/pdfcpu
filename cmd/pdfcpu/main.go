package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hhrutter/pdfcpu"
	PDFCPULog "github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
)

var (
	fileStats, mode, pageSelection string
	upw, opw, key, perm            string
	verbose                        bool

	needStackTrace = true
)

func init() {

	statsUsage := "optimize: a csv file for stats appending"
	flag.StringVar(&fileStats, "stats", "", statsUsage)
	flag.StringVar(&fileStats, "s", "", statsUsage)

	modeUsage := "validate: strict|relaxed; extract: image|font|content|page; encrypt: rc4|aes"
	flag.StringVar(&mode, "mode", "", modeUsage)
	flag.StringVar(&mode, "m", "", modeUsage)

	keyUsage := "encrypt: 40|128"
	flag.StringVar(&key, "key", "128", keyUsage)
	flag.StringVar(&key, "k", "128", keyUsage)

	permUsage := "encrypt, perm set: none|all"
	flag.StringVar(&perm, "perm", "none", permUsage)

	pageSelectionUsage := "a comma separated list of pages or page ranges, see pdfcpu help split/extract"
	flag.StringVar(&pageSelection, "pages", "", pageSelectionUsage)
	flag.StringVar(&pageSelection, "p", "", pageSelectionUsage)

	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")

	flag.StringVar(&upw, "upw", "", "user password")
	flag.StringVar(&opw, "opw", "", "owner password")

}

func ensurePdfExtension(filename string) {
	if !strings.HasSuffix(filename, ".pdf") {
		log.Fatalf("%s needs extension \".pdf\".", filename)
	}
}

func defaultFilenameOut(filename string) string {
	ensurePdfExtension(filename)
	return strings.TrimSuffix(filename, ".pdf") + "_new.pdf"
}

func version() {

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "pdfcpu version %s\n", types.PDFCPUVersion)
}

func helpString(topic string) string {

	switch topic {

	case "validate":
		return fmt.Sprintf("%s\n\n%s\n", usageValidate, usageLongValidate)

	case "optimize":
		return fmt.Sprintf("%s\n\n%s\n", usageOptimize, usageLongOptimize)

	case "split":
		return fmt.Sprintf("%s\n\n%s\n", usageSplit, usageLongSplit)

	case "merge":
		return fmt.Sprintf("%s\n\n%s\n", usageMerge, usageLongMerge)

	case "extract":
		return fmt.Sprintf("%s\n\n%s\n\n%s\n", usageExtract, usageLongExtract, usagePageSelection)

	case "trim":
		return fmt.Sprintf("%s\n\n%s\n\n%s\n", usageTrim, usageLongTrim, usagePageSelection)

	case "attach":
		return fmt.Sprintf("%s\n\n%s\n", usageAttach, usageLongAttach)

	case "perm":
		return fmt.Sprintf("%s\n\n%s\n", usagePerm, usageLongPerm)

	case "encrypt":
		return fmt.Sprintf("%s\n\n%s\n", usageEncrypt, usageLongEncrypt)

	case "decrypt":
		return fmt.Sprintf("%s\n\n%s\n", usageDecrypt, usageLongDecrypt)

	case "changeupw":
		return fmt.Sprintf("%s\n\n%s\n", usageChangeUserPW, usageLongChangeUserPW)

	case "changeopw":
		return fmt.Sprintf("%s\n\n%s\n", usageChangeOwnerPW, usageLongChangeOwnerPW)

	case "version":
		return fmt.Sprintf("%s\n\n%s\n", usageVersion, usageLongVersion)

	default:
		return fmt.Sprintf("Unknown help topic `%s`.  Run 'pdfcpu help'.\n", topic)

	}

}

func help() {

	switch len(flag.Args()) {

	case 0:
		fmt.Fprintln(os.Stderr, usage)

	case 1:
		fmt.Fprintln(os.Stderr, helpString(flag.Arg(0)))

	default:
		fmt.Fprintln(os.Stderr, "usage: pdfcpu help command\n\nToo many arguments given.")

	}

}

func setupLogging(verbose bool) {

	if verbose {

		//PDFCPULog.SetDefaultLoggers()

		PDFCPULog.SetDefaultDebugLogger()
		PDFCPULog.SetDefaultInfoLogger()
		PDFCPULog.SetDefaultStatsLogger()
	}

	needStackTrace = verbose
}

func parseFlagsAndGetCommand() (command string) {

	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	// The first argument is the pdfcpu command => start flag processing after 2nd argument.
	command = os.Args[1]

	i := 2
	// The attach command uses a subcommand and is therefore a special case => start flag processing after 3rd argument.
	if command == "attach" {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, usageAttach)
			os.Exit(1)
		}
		i = 3
	}

	if command == "perm" {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, usagePerm)
			os.Exit(1)
		}
		i = 3
	}

	// Parse commandline flags.
	flag.CommandLine.Parse(os.Args[i:])

	return
}

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
		config.SetValidationStrict()
	case "relaxed", "r":
		config.SetValidationRelaxed()
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

	var err error
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

	var err error
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
	//fmt.Println("filenameIn: " + filenameIn)

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
	//fmt.Println("filenameIn: " + filenameIn)

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

	//fmt.Printf("mode: %s\n", mode)
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

func process(cmd *pdfcpu.Command) {

	out, err := pdfcpu.Process(cmd)

	if err != nil {
		if needStackTrace {
			fmt.Fprintf(os.Stderr, "Fatal: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}

	if out != nil {
		for _, l := range out {
			fmt.Fprintln(os.Stdout, l)
		}
	}
}

func handleVersion(command string) {
	if (command == "v" || command == "version") && len(flag.Args()) == 0 {
		version()
		os.Exit(0)
	}
}

func main() {

	command := parseFlagsAndGetCommand()

	setupLogging(verbose)

	config := types.NewDefaultConfiguration()
	config.UserPW = upw
	config.OwnerPW = opw

	var cmd *pdfcpu.Command

	handleVersion(command)

	switch command {

	case "validate":
		cmd = prepareValidateCommand(config)

	case "optimize", "o":
		// Always write using 0x0A end of line sequence default.
		cmd = prepareOptimizeCommand(config)

	case "split", "s":
		cmd = prepareSplitCommand(config)

	case "merge", "m":
		cmd = prepareMergeCommand(config)

	case "extract", "ext":
		cmd = prepareExtractCommand(config)

	case "trim", "t":
		cmd = prepareTrimCommand(config)

	case "attach":
		cmd = prepareAttachmentCommand(config)

	case "decrypt", "d", "dec":
		cmd = prepareDecryptCommand(config)

	case "encrypt", "enc":
		cmd = prepareEncryptCommand(config)

	case "changeupw", "changeopw":
		cmd = prepareChangePasswordCommand(config, command)

	case "perm":
		cmd = preparePermissionsCommand(config)

	case "help", "h":
		help()
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "pdfcpu unknown subcommand \"%s\"\n", command)
		fmt.Fprintln(os.Stderr, "Run 'pdfcpu help' for usage.")
		os.Exit(1)

	}

	process(cmd)
}

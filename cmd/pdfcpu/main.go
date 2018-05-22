package main

import (
	"flag"
	"fmt"
	"log"

	"os"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/api"
	PDFCPULog "github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
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

func main() {

	command := parseFlagsAndGetCommand()

	setupLogging(verbose)

	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = upw
	config.OwnerPW = opw

	var cmd *api.Command

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

func ensurePdfExtension(filename string) {
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		log.Fatalf("%s needs extension \".pdf\".", filename)
	}
}

func defaultFilenameOut(filename string) string {
	ensurePdfExtension(filename)
	return strings.TrimSuffix(strings.ToLower(filename), ".pdf") + "_new.pdf"
}

func version() {

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "pdfcpu version %s\n", pdfcpu.PDFCPUVersion)
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
	err := flag.CommandLine.Parse(os.Args[i:])
	if err != nil {
		os.Exit(1)
	}

	return
}

func process(cmd *api.Command) {

	out, err := api.Process(cmd)

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

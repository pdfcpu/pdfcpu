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

// Package main provides the command line for interacting with pdfcpu.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/api"
	PDFCPULog "github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

var (
	fileStats, mode, pageSelection string
	upw, opw, key, perm            string
	verbose, veryVerbose           bool

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
	flag.BoolVar(&veryVerbose, "vv", false, "")

	flag.StringVar(&upw, "upw", "", "user password")
	flag.StringVar(&opw, "opw", "", "owner password")

}

func main() {

	command := parseFlagsAndGetCommand()

	setupLogging(verbose, veryVerbose)

	config := pdfcpu.NewDefaultConfiguration()
	config.UserPW = upw
	config.OwnerPW = opw

	var cmd *api.Command

	handleVersion(command)

	if command == "h" || command == "help" {
		help()
		os.Exit(1)
	}

	for k, v := range map[string]func(config *pdfcpu.Configuration) *api.Command{
		"validate":  prepareValidateCommand,
		"optimize":  prepareOptimizeCommand,
		"o":         prepareOptimizeCommand,
		"split":     prepareSplitCommand,
		"s":         prepareSplitCommand,
		"merge":     prepareMergeCommand,
		"m":         prepareMergeCommand,
		"extract":   prepareExtractCommand,
		"ext":       prepareExtractCommand,
		"trim":      prepareTrimCommand,
		"t":         prepareTrimCommand,
		"attach":    prepareAttachmentCommand,
		"decrypt":   prepareDecryptCommand,
		"d":         prepareDecryptCommand,
		"dec":       prepareDecryptCommand,
		"encrypt":   prepareEncryptCommand,
		"enc":       prepareEncryptCommand,
		"changeupw": prepareChangeUserPasswordCommand,
		"changeopw": prepareChangeOwnerPasswordCommand,
		"perm":      preparePermissionsCommand,
		"stamp":     prepareAddStampsCommand,
		"watermark": prepareAddWatermarksCommand,
	} {
		if command == k {
			cmd = v(config)
			process(cmd)
			os.Exit(0)
		}
	}

	fmt.Fprintf(os.Stderr, "pdfcpu unknown subcommand \"%s\"\n", command)
	fmt.Fprintln(os.Stderr, "Run 'pdfcpu help' for usage.")
	os.Exit(1)
}

func ensurePdfExtension(filename string) {
	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".", filename)
		os.Exit(1)
	}
}

func defaultFilenameOut(filename string) string {
	ensurePdfExtension(filename)
	return filename[:len(filename)-4] + "_new.pdf"
}

func version() {

	if len(flag.Args()) != 0 {
		fmt.Fprintf(os.Stderr, "%s\n\n", usageVersion)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "pdfcpu version %s\n", pdfcpu.PDFCPUVersion)
}

func helpString(topic string) string {

	for k, v := range map[string]struct {
		usageShort, usageLong string
		usagePageSelection    bool
	}{
		"validate":  {usageValidate, usageLongValidate, false},
		"optimize":  {usageOptimize, usageLongOptimize, false},
		"split":     {usageSplit, usageLongSplit, false},
		"merge":     {usageMerge, usageLongMerge, false},
		"extract":   {usageExtract, usageLongExtract, false},
		"trim":      {usageTrim, usageLongTrim, true},
		"attach":    {usageAttach, usageLongAttach, false},
		"perm":      {usagePerm, usageLongPerm, false},
		"encrypt":   {usageEncrypt, usageLongEncrypt, false},
		"decrypt":   {usageDecrypt, usageLongDecrypt, false},
		"changeupw": {usageChangeUserPW, usageLongChangeUserPW, false},
		"changeopw": {usageChangeOwnerPW, usageLongChangeOwnerPW, false},
		"stamp":     {usageStamp, usageLongStamp, true},
		"watermark": {usageWatermark, usageLongWatermark, true},
		"version":   {usageVersion, usageLongVersion, false},
	} {
		if topic == k {
			if v.usagePageSelection {
				return fmt.Sprintf("%s\n\n%s\n\n%s\n", v.usageShort, v.usageLong, usagePageSelection)
			}
			return fmt.Sprintf("%s\n\n%s\n", v.usageShort, v.usageLong)
		}
	}

	return fmt.Sprintf("Unknown help topic `%s`.  Run 'pdfcpu help'.\n", topic)
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

func setupLogging(verbose, veryVerbose bool) {

	needStackTrace = verbose || veryVerbose

	if verbose || veryVerbose {
		PDFCPULog.SetDefaultDebugLogger()
		PDFCPULog.SetDefaultInfoLogger()
		PDFCPULog.SetDefaultStatsLogger()
	}

	if veryVerbose {
		PDFCPULog.SetDefaultTraceLogger()
		//PDFCPULog.SetDefaultParseLogger()
		PDFCPULog.SetDefaultReadLogger()
		PDFCPULog.SetDefaultValidateLogger()
		PDFCPULog.SetDefaultOptimizeLogger()
		PDFCPULog.SetDefaultWriteLogger()
	}

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

	// The perm command uses a subcommand and is therefore a special case => start flag processing after 3rd argument.
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

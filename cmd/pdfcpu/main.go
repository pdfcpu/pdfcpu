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

	"github.com/hhrutter/pdfcpu/pkg/api"
	PDFCPULog "github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

var (
	fileStats, mode, pageSelection string
	upw, opw, key, perm            string
	verbose, veryVerbose           bool
	needStackTrace                 = true
	cmdMap                         CommandMap
)

func initFlags() {

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

func initCommandMap() {

	attachCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"list":    {prepareListAttachmentsCommand, nil, "", ""},
		"add":     {prepareAddAttachmentsCommand, nil, "", ""},
		"remove":  {prepareRemoveAttachmentsCommand, nil, "", ""},
		"extract": {prepareExtractAttachmentsCommand, nil, "", ""},
	} {
		attachCmdMap.Register(k, v)
	}

	permissionsCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"list": {prepareListPermissionsCommand, nil, "", ""},
		"add":  {prepareAddPermissionsCommand, nil, "", ""},
	} {
		permissionsCmdMap.Register(k, v)
	}

	pagesCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"insert": {prepareInsertPagesCommand, nil, "", ""},
		"remove": {prepareRemovePagesCommand, nil, "", ""},
	} {
		pagesCmdMap.Register(k, v)
	}

	cmdMap = NewCommandMap()

	for k, v := range map[string]Command{
		"attachments": {nil, attachCmdMap, usageAttach, usageLongAttach},
		"changeopw":   {prepareChangeOwnerPasswordCommand, nil, usageChangeOwnerPW, usageLongChangeUserPW},
		"changeupw":   {prepareChangeUserPasswordCommand, nil, usageChangeUserPW, usageLongChangeUserPW},
		"decrypt":     {prepareDecryptCommand, nil, usageDecrypt, usageLongDecrypt},
		"encrypt":     {prepareEncryptCommand, nil, usageEncrypt, usageLongEncrypt},
		"extract":     {prepareExtractCommand, nil, usageExtract, usageLongExtract},
		"grid":        {prepareGridCommand, nil, usageGrid, usageLongGrid},
		"help":        {printHelp, nil, "", ""},
		"import":      {prepareImportImagesCommand, nil, usageImportImages, usageLongImportImages},
		"merge":       {prepareMergeCommand, nil, usageMerge, usageLongMerge},
		"nup":         {prepareNUpCommand, nil, usageNUp, usageLongNUp},
		"n-up":        {prepareNUpCommand, nil, usageNUp, usageLongNUp},
		"optimize":    {prepareOptimizeCommand, nil, usageOptimize, usageLongOptimize},
		"pages":       {nil, pagesCmdMap, usagePages, usageLongPages},
		"paper":       {printPaperSizes, nil, usagePaper, usageLongPaper},
		"permissions": {nil, permissionsCmdMap, usagePerm, usageLongPerm},
		"rotate":      {prepareRotateCommand, nil, usageRotate, usageLongRotate},
		"split":       {prepareSplitCommand, nil, usageSplit, usageLongSplit},
		"stamp":       {prepareAddStampsCommand, nil, usageStamp, usageLongStamp},
		"trim":        {prepareTrimCommand, nil, usageTrim, usageLongTrim},
		"validate":    {prepareValidateCommand, nil, usageValidate, usageLongValidate},
		"watermark":   {prepareAddWatermarksCommand, nil, usageWatermark, usageLongWatermark},
		"version":     {printVersion, nil, usageVersion, usageLongVersion},
	} {
		cmdMap.Register(k, v)
	}
}

func initLogging(verbose, veryVerbose bool) {

	needStackTrace = verbose || veryVerbose

	PDFCPULog.SetDefaultAPILogger()

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

func init() {
	initFlags()
	initCommandMap()
}

func main() {

	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(0)
	}

	// The first argument is the pdfcpu command
	cmdStr := os.Args[1]

	config := pdfcpu.NewConfiguration(upw, opw)

	cmd, command, err := cmdMap.Handle(cmdStr, "", config)
	if err != nil {
		if len(command) > 0 {
			cmdStr = fmt.Sprintf("%s %s", command, os.Args[2])
		}
		if err == errUnknownCmd {
			fmt.Fprintf(os.Stderr, "pdfcpu unknown command \"%s\"\n", cmdStr)
		}
		if err == errAmbiguousCmd {
			fmt.Fprintf(os.Stderr, "pdfcpu ambiguous command \"%s\"\n", cmdStr)
		}
		fmt.Fprintln(os.Stderr, "Run 'pdfcpu help' for usage.")
		os.Exit(1)
	}

	if cmd != nil {
		process(cmd)
	}

	os.Exit(0)
}

func parseFlags(cmd *Command) {

	// Execute after command completion.

	i := 2

	// This command uses a subcommand and is therefore a special case => start flag processing after 3rd argument.
	if cmd.handler == nil {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, cmd.usageShort)
			os.Exit(1)
		}
		i = 3
	}

	// Parse commandline flags.
	if !flag.CommandLine.Parsed() {

		err := flag.CommandLine.Parse(os.Args[i:])
		if err != nil {
			os.Exit(1)
		}

		initLogging(verbose, veryVerbose)
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

	os.Exit(0)
}

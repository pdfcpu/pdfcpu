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

	"github.com/hhrutter/pdfcpu/pkg/cli"
	PDFCPULog "github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

var (
	fileStats, mode, selectedPages string
	upw, opw, key, perm            string
	verbose, veryVerbose           bool
	quiet                          bool
	needStackTrace                 = true
	cmdMap                         CommandMap
)

// Set by Goreleaser.
var (
	commit = "none"
	date   = "unknown"
)

func initFlags() {

	statsUsage := "optimize: a csv file for stats appending"
	flag.StringVar(&fileStats, "stats", "", statsUsage)
	flag.StringVar(&fileStats, "s", "", statsUsage)

	modeUsage := "validate: strict|relaxed; extract: image|font|content|page; encrypt: rc4|aes"
	flag.StringVar(&mode, "mode", "", modeUsage)
	flag.StringVar(&mode, "m", "", modeUsage)

	keyUsage := "encrypt: 40|128|256"
	flag.StringVar(&key, "key", "256", keyUsage)
	flag.StringVar(&key, "k", "256", keyUsage)

	permUsage := "encrypt, perm set: none|all"
	flag.StringVar(&perm, "perm", "none", permUsage)

	selectedPagesUsage := "a comma separated list of pages or page ranges, see pdfcpu help split/extract"
	flag.StringVar(&selectedPages, "pages", "", selectedPagesUsage)
	flag.StringVar(&selectedPages, "p", "", selectedPagesUsage)

	flag.BoolVar(&quiet, "quiet", false, "")
	flag.BoolVar(&quiet, "q", false, "")

	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")
	flag.BoolVar(&veryVerbose, "vv", false, "")

	flag.StringVar(&upw, "upw", "", "user password")
	flag.StringVar(&opw, "opw", "", "owner password")
}

func initCommandMap() {

	attachCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"list":    {handleListAttachmentsCommand, nil, "", ""},
		"add":     {handleAddAttachmentsCommand, nil, "", ""},
		"remove":  {handleRemoveAttachmentsCommand, nil, "", ""},
		"extract": {handleExtractAttachmentsCommand, nil, "", ""},
	} {
		attachCmdMap.Register(k, v)
	}

	permissionsCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"list": {handleListPermissionsCommand, nil, "", ""},
		"set":  {handleSetPermissionsCommand, nil, "", ""},
	} {
		permissionsCmdMap.Register(k, v)
	}

	pagesCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"insert": {handleInsertPagesCommand, nil, "", ""},
		"remove": {handleRemovePagesCommand, nil, "", ""},
	} {
		pagesCmdMap.Register(k, v)
	}

	cmdMap = NewCommandMap()

	for k, v := range map[string]Command{
		"attachments": {nil, attachCmdMap, usageAttach, usageLongAttach},
		"changeopw":   {handleChangeOwnerPasswordCommand, nil, usageChangeOwnerPW, usageLongChangeUserPW},
		"changeupw":   {handleChangeUserPasswordCommand, nil, usageChangeUserPW, usageLongChangeUserPW},
		"decrypt":     {handleDecryptCommand, nil, usageDecrypt, usageLongDecrypt},
		"encrypt":     {handleEncryptCommand, nil, usageEncrypt, usageLongEncrypt},
		"extract":     {handleExtractCommand, nil, usageExtract, usageLongExtract},
		"grid":        {handleGridCommand, nil, usageGrid, usageLongGrid},
		"help":        {printHelp, nil, "", ""},
		"info":        {handleInfoCommand, nil, usageInfo, usageLongInfo},
		"import":      {handleImportImagesCommand, nil, usageImportImages, usageLongImportImages},
		"merge":       {handleMergeCommand, nil, usageMerge, usageLongMerge},
		"nup":         {handleNUpCommand, nil, usageNUp, usageLongNUp},
		"n-up":        {handleNUpCommand, nil, usageNUp, usageLongNUp},
		"optimize":    {handleOptimizeCommand, nil, usageOptimize, usageLongOptimize},
		"pages":       {nil, pagesCmdMap, usagePages, usageLongPages},
		"paper":       {printPaperSizes, nil, usagePaper, usageLongPaper},
		"permissions": {nil, permissionsCmdMap, usagePerm, usageLongPerm},
		"rotate":      {handleRotateCommand, nil, usageRotate, usageLongRotate},
		"split":       {handleSplitCommand, nil, usageSplit, usageLongSplit},
		"stamp":       {handleAddStampsCommand, nil, usageStamp, usageLongStamp},
		"trim":        {handleTrimCommand, nil, usageTrim, usageLongTrim},
		"validate":    {handleValidateCommand, nil, usageValidate, usageLongValidate},
		"watermark":   {handleAddWatermarksCommand, nil, usageWatermark, usageLongWatermark},
		"version":     {printVersion, nil, usageVersion, usageLongVersion},
	} {
		cmdMap.Register(k, v)
	}
}

func initLogging(verbose, veryVerbose bool) {

	needStackTrace = verbose || veryVerbose

	if quiet {
		return
	}

	PDFCPULog.SetDefaultCLILogger()

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

	conf := pdfcpu.NewDefaultConfiguration()

	str, err := cmdMap.Handle(cmdStr, "", conf)
	if err != nil {
		if len(str) > 0 {
			cmdStr = fmt.Sprintf("%s %s", str, os.Args[2])
		}
		fmt.Fprintf(os.Stderr, "%v \"%s\"\n", err, cmdStr)
		fmt.Fprintln(os.Stderr, "Run 'pdfcpu help' for usage.")
		os.Exit(1)
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

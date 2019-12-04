/*
Copyright 2019 The pdfcpu Authors.

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

	PDFCPULog "github.com/pdfcpu/pdfcpu/pkg/log"
)

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

	stampCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"add":    {handleAddStampsCommand, nil, "", ""},
		"remove": {handleRemoveStampsCommand, nil, "", ""},
		"update": {handleUpdateStampsCommand, nil, "", ""},
	} {
		stampCmdMap.Register(k, v)
	}

	watermarkCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"add":    {handleAddWatermarksCommand, nil, "", ""},
		"remove": {handleRemoveWatermarksCommand, nil, "", ""},
		"update": {handleUpdateWatermarksCommand, nil, "", ""},
	} {
		watermarkCmdMap.Register(k, v)
	}

	fontsCmdMap := NewCommandMap()
	for k, v := range map[string]Command{
		"install": {handleInstallFontsCommand, nil, "", ""},
		"list":    {handleListFontsCommand, nil, "", ""},
	} {
		fontsCmdMap.Register(k, v)
	}

	cmdMap = NewCommandMap()

	for k, v := range map[string]Command{
		"attachments": {nil, attachCmdMap, usageAttach, usageLongAttach},
		"changeopw":   {handleChangeOwnerPasswordCommand, nil, usageChangeOwnerPW, usageLongChangeUserPW},
		"changeupw":   {handleChangeUserPasswordCommand, nil, usageChangeUserPW, usageLongChangeUserPW},
		"decrypt":     {handleDecryptCommand, nil, usageDecrypt, usageLongDecrypt},
		"encrypt":     {handleEncryptCommand, nil, usageEncrypt, usageLongEncrypt},
		"extract":     {handleExtractCommand, nil, usageExtract, usageLongExtract},
		"fonts":       {nil, fontsCmdMap, usageFonts, usageLongFonts},
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
		"stamp":       {nil, stampCmdMap, usageStamp, usageLongStamp},
		"trim":        {handleTrimCommand, nil, usageTrim, usageLongTrim},
		"validate":    {handleValidateCommand, nil, usageValidate, usageLongValidate},
		"watermark":   {nil, watermarkCmdMap, usageWatermark, usageLongWatermark},
		"version":     {printVersion, nil, usageVersion, usageLongVersion},
	} {
		cmdMap.Register(k, v)
	}
}

func initFlags() {

	statsUsage := "optimize: a csv file for stats appending"
	flag.StringVar(&fileStats, "stats", "", statsUsage)
	flag.StringVar(&fileStats, "s", "", statsUsage)

	modeUsage := "validate: strict|relaxed; extract: image|font|content|page|meta; encrypt: rc4|aes, stamp:text|image/pdf"
	flag.StringVar(&mode, "mode", "", modeUsage)
	flag.StringVar(&mode, "m", "", modeUsage)

	keyUsage := "encrypt: 40|128|256"
	flag.StringVar(&key, "key", "256", keyUsage)
	flag.StringVar(&key, "k", "256", keyUsage)

	permUsage := "encrypt, perm set: none|all"
	flag.StringVar(&perm, "perm", "none", permUsage)

	unitsUsage := "info: po|in|cm|mm"
	flag.StringVar(&units, "units", "po", unitsUsage)
	flag.StringVar(&units, "u", "po", unitsUsage)

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

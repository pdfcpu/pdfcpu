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

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func initCommandMap() {
	attachCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"list":    {processListAttachmentsCommand, nil, "", ""},
		"add":     {processAddAttachmentsCommand, nil, "", ""},
		"remove":  {processRemoveAttachmentsCommand, nil, "", ""},
		"extract": {processExtractAttachmentsCommand, nil, "", ""},
	} {
		attachCmdMap.register(k, v)
	}

	boxesCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListBoxesCommand, nil, "", ""},
		"add":    {processAddBoxesCommand, nil, "", ""},
		"remove": {processRemoveBoxesCommand, nil, "", ""},
	} {
		boxesCmdMap.register(k, v)
	}

	portfolioCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"list":    {processListAttachmentsCommand, nil, "", ""},
		"add":     {processAddAttachmentsPortfolioCommand, nil, "", ""},
		"remove":  {processRemoveAttachmentsCommand, nil, "", ""},
		"extract": {processExtractAttachmentsCommand, nil, "", ""},
	} {
		portfolioCmdMap.register(k, v)
	}

	permissionsCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"list": {processListPermissionsCommand, nil, "", ""},
		"set":  {processSetPermissionsCommand, nil, "", ""},
	} {
		permissionsCmdMap.register(k, v)
	}

	pagesCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"insert": {processInsertPagesCommand, nil, "", ""},
		"remove": {processRemovePagesCommand, nil, "", ""},
	} {
		pagesCmdMap.register(k, v)
	}

	stampCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"add":    {processAddStampsCommand, nil, "", ""},
		"remove": {processRemoveStampsCommand, nil, "", ""},
		"update": {processUpdateStampsCommand, nil, "", ""},
	} {
		stampCmdMap.register(k, v)
	}

	watermarkCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"add":    {processAddWatermarksCommand, nil, "", ""},
		"remove": {processRemoveWatermarksCommand, nil, "", ""},
		"update": {processUpdateWatermarksCommand, nil, "", ""},
	} {
		watermarkCmdMap.register(k, v)
	}

	fontsCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"cheatsheet": {processCreateCheatSheetFontsCommand, nil, "", ""},
		"install":    {processInstallFontsCommand, nil, "", ""},
		"list":       {processListFontsCommand, nil, "", ""},
	} {
		fontsCmdMap.register(k, v)
	}

	keywordsCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListKeywordsCommand, nil, "", ""},
		"add":    {processAddKeywordsCommand, nil, "", ""},
		"remove": {processRemoveKeywordsCommand, nil, "", ""},
	} {
		keywordsCmdMap.register(k, v)
	}

	propertiesCmdMap := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListPropertiesCommand, nil, "", ""},
		"add":    {processAddPropertiesCommand, nil, "", ""},
		"remove": {processRemovePropertiesCommand, nil, "", ""},
	} {
		propertiesCmdMap.register(k, v)
	}

	cmdMap = newCommandMap()

	for k, v := range map[string]command{
		"attachments":   {nil, attachCmdMap, usageAttach, usageLongAttach},
		"booklet":       {processBookletCommand, nil, usageBooklet, usageLongBooklet},
		"boxes":         {nil, boxesCmdMap, usageBoxes, usageLongBoxes},
		"changeopw":     {processChangeOwnerPasswordCommand, nil, usageChangeOwnerPW, usageLongChangeUserPW},
		"changeupw":     {processChangeUserPasswordCommand, nil, usageChangeUserPW, usageLongChangeUserPW},
		"collect":       {processCollectCommand, nil, usageCollect, usageLongCollect},
		"crop":          {processCropCommand, nil, usageCrop, usageLongCrop},
		"decrypt":       {processDecryptCommand, nil, usageDecrypt, usageLongDecrypt},
		"encrypt":       {processEncryptCommand, nil, usageEncrypt, usageLongEncrypt},
		"extract":       {processExtractCommand, nil, usageExtract, usageLongExtract},
		"fonts":         {nil, fontsCmdMap, usageFonts, usageLongFonts},
		"grid":          {processGridCommand, nil, usageGrid, usageLongGrid},
		"help":          {printHelp, nil, "", ""},
		"info":          {processInfoCommand, nil, usageInfo, usageLongInfo},
		"import":        {processImportImagesCommand, nil, usageImportImages, usageLongImportImages},
		"keywords":      {nil, keywordsCmdMap, usageKeywords, usageLongKeywords},
		"merge":         {processMergeCommand, nil, usageMerge, usageLongMerge},
		"nup":           {processNUpCommand, nil, usageNUp, usageLongNUp},
		"optimize":      {processOptimizeCommand, nil, usageOptimize, usageLongOptimize},
		"pages":         {nil, pagesCmdMap, usagePages, usageLongPages},
		"paper":         {printPaperSizes, nil, usagePaper, usageLongPaper},
		"permissions":   {nil, permissionsCmdMap, usagePerm, usageLongPerm},
		"portfolio":     {nil, portfolioCmdMap, usagePortfolio, usageLongPortfolio},
		"properties":    {nil, propertiesCmdMap, usageProperties, usageLongProperties},
		"rotate":        {processRotateCommand, nil, usageRotate, usageLongRotate},
		"selectedpages": {printSelectedPages, nil, usageSelectedPages, usageLongSelectedPages},
		"split":         {processSplitCommand, nil, usageSplit, usageLongSplit},
		"stamp":         {nil, stampCmdMap, usageStamp, usageLongStamp},
		"trim":          {processTrimCommand, nil, usageTrim, usageLongTrim},
		"validate":      {processValidateCommand, nil, usageValidate, usageLongValidate},
		"watermark":     {nil, watermarkCmdMap, usageWatermark, usageLongWatermark},
		"version":       {printVersion, nil, usageVersion, usageLongVersion},
	} {
		cmdMap.register(k, v)
	}
}

func initFlags() {
	statsUsage := "optimize: create a csv file for stats"
	flag.StringVar(&fileStats, "stats", "", statsUsage)

	modeUsage := "validate: strict|relaxed; extract: image|font|content|page|meta; encrypt: rc4|aes, stamp:text|image/pdf"
	flag.StringVar(&mode, "mode", "", modeUsage)
	flag.StringVar(&mode, "m", "", modeUsage)

	keyUsage := "encrypt: 40|128|256"
	flag.StringVar(&key, "key", "256", keyUsage)
	flag.StringVar(&key, "k", "256", keyUsage)

	permUsage := "encrypt, perm set: none|all"
	flag.StringVar(&perm, "perm", "none", permUsage)

	unitUsage := "info: po|in|cm|mm"
	flag.StringVar(&unit, "unit", "", unitUsage)
	flag.StringVar(&unit, "u", "", unitUsage)

	selectedPagesUsage := "a comma separated list of pages or page ranges, see pdfcpu help split/extract"
	flag.StringVar(&selectedPages, "pages", "", selectedPagesUsage)
	flag.StringVar(&selectedPages, "p", "", selectedPagesUsage)

	flag.BoolVar(&quiet, "quiet", false, "")
	flag.BoolVar(&quiet, "q", false, "")

	sortUsage := "sort files before merging"
	flag.BoolVar(&sorted, "sort", false, sortUsage)
	flag.BoolVar(&sorted, "s", false, sortUsage)

	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")
	flag.BoolVar(&veryVerbose, "vv", false, "")

	linksUsage := "check for broken links"
	flag.BoolVar(&links, "links", false, linksUsage)
	flag.BoolVar(&links, "l", false, linksUsage)

	flag.StringVar(&upw, "upw", "", "user password")
	flag.StringVar(&opw, "opw", "", "owner password")

	confUsage := "the config directory path | skip | none"
	flag.StringVar(&conf, "config", "", confUsage)
	flag.StringVar(&conf, "conf", "", confUsage)
	flag.StringVar(&conf, "c", "", confUsage)
}

func initLogging(verbose, veryVerbose bool) {
	needStackTrace = verbose || veryVerbose
	if quiet {
		return
	}

	log.SetDefaultCLILogger()

	if verbose || veryVerbose {
		log.SetDefaultDebugLogger()
		log.SetDefaultInfoLogger()
		log.SetDefaultStatsLogger()
	}

	if veryVerbose {
		log.SetDefaultTraceLogger()
		//log.SetDefaultParseLogger()
		log.SetDefaultReadLogger()
		log.SetDefaultValidateLogger()
		log.SetDefaultOptimizeLogger()
		log.SetDefaultWriteLogger()
	}
}

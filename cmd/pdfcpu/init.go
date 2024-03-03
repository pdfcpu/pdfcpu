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

func initAnnotsCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListAnnotationsCommand, nil, "", ""},
		"remove": {processRemoveAnnotationsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initAttachCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":    {processListAttachmentsCommand, nil, "", ""},
		"add":     {processAddAttachmentsCommand, nil, "", ""},
		"remove":  {processRemoveAttachmentsCommand, nil, "", ""},
		"extract": {processExtractAttachmentsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initBookmarksCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListBookmarksCommand, nil, "", ""},
		"import": {processImportBookmarksCommand, nil, "", ""},
		"export": {processExportBookmarksCommand, nil, "", ""},
		"remove": {processRemoveBookmarksCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initBoxesCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListBoxesCommand, nil, "", ""},
		"add":    {processAddBoxesCommand, nil, "", ""},
		"remove": {processRemoveBoxesCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initFontsCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"cheatsheet": {processCreateCheatSheetFontsCommand, nil, "", ""},
		"install":    {processInstallFontsCommand, nil, "", ""},
		"list":       {processListFontsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initFormCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":      {processListFormFieldsCommand, nil, "", ""},
		"remove":    {processRemoveFormFieldsCommand, nil, "", ""},
		"lock":      {processLockFormCommand, nil, "", ""},
		"unlock":    {processUnlockFormCommand, nil, "", ""},
		"reset":     {processResetFormCommand, nil, "", ""},
		"export":    {processExportFormCommand, nil, "", ""},
		"fill":      {processFillFormCommand, nil, "", ""},
		"multifill": {processMultiFillFormCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initImagesCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list": {processListImagesCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initKeywordsCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListKeywordsCommand, nil, "", ""},
		"add":    {processAddKeywordsCommand, nil, "", ""},
		"remove": {processRemoveKeywordsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initPagesCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"insert": {processInsertPagesCommand, nil, "", ""},
		"remove": {processRemovePagesCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initPermissionsCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list": {processListPermissionsCommand, nil, "", ""},
		"set":  {processSetPermissionsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initPortfolioCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":    {processListAttachmentsCommand, nil, "", ""},
		"add":     {processAddAttachmentsPortfolioCommand, nil, "", ""},
		"remove":  {processRemoveAttachmentsCommand, nil, "", ""},
		"extract": {processExtractAttachmentsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initPropertiesCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":   {processListPropertiesCommand, nil, "", ""},
		"add":    {processAddPropertiesCommand, nil, "", ""},
		"remove": {processRemovePropertiesCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initStampCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"add":    {processAddStampsCommand, nil, "", ""},
		"remove": {processRemoveStampsCommand, nil, "", ""},
		"update": {processUpdateStampsCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initWatermarkCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"add":    {processAddWatermarksCommand, nil, "", ""},
		"remove": {processRemoveWatermarksCommand, nil, "", ""},
		"update": {processUpdateWatermarksCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initPageModeCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":  {processListPageModeCommand, nil, "", ""},
		"set":   {processSetPageModeCommand, nil, "", ""},
		"reset": {processResetPageModeCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initPageLayoutCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":  {processListPageLayoutCommand, nil, "", ""},
		"set":   {processSetPageLayoutCommand, nil, "", ""},
		"reset": {processResetPageLayoutCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initViewerPreferencesCmdMap() commandMap {
	m := newCommandMap()
	for k, v := range map[string]command{
		"list":  {processListViewerPreferencesCommand, nil, "", ""},
		"set":   {processSetViewerPreferencesCommand, nil, "", ""},
		"reset": {processResetViewerPreferencesCommand, nil, "", ""},
	} {
		m.register(k, v)
	}
	return m
}

func initCommandMap() {
	annotsCmdMap := initAnnotsCmdMap()
	attachCmdMap := initAttachCmdMap()
	bookmarksCmdMap := initBookmarksCmdMap()
	boxesCmdMap := initBoxesCmdMap()
	fontsCmdMap := initFontsCmdMap()
	formCmdMap := initFormCmdMap()
	imagesCmdMap := initImagesCmdMap()
	keywordsCmdMap := initKeywordsCmdMap()
	pagesCmdMap := initPagesCmdMap()
	permissionsCmdMap := initPermissionsCmdMap()
	portfolioCmdMap := initPortfolioCmdMap()
	propertiesCmdMap := initPropertiesCmdMap()
	stampCmdMap := initStampCmdMap()
	watermarkCmdMap := initWatermarkCmdMap()
	pageModeCmdMap := initPageModeCmdMap()
	pageLayoutCmdMap := initPageLayoutCmdMap()
	viewerPrefsCmdMap := initViewerPreferencesCmdMap()

	cmdMap = newCommandMap()

	for k, v := range map[string]command{
		"annotations":   {nil, annotsCmdMap, usageAnnots, usageLongAnnots},
		"attachments":   {nil, attachCmdMap, usageAttach, usageLongAttach},
		"bookmarks":     {nil, bookmarksCmdMap, usageBookmarks, usageLongBookmarks},
		"booklet":       {processBookletCommand, nil, usageBooklet, usageLongBooklet},
		"boxes":         {nil, boxesCmdMap, usageBoxes, usageLongBoxes},
		"changeopw":     {processChangeOwnerPasswordCommand, nil, usageChangeOwnerPW, usageLongChangeOwnerPW},
		"changeupw":     {processChangeUserPasswordCommand, nil, usageChangeUserPW, usageLongChangeUserPW},
		"collect":       {processCollectCommand, nil, usageCollect, usageLongCollect},
		"config":        {printConfiguration, nil, usageConfig, usageLongConfig},
		"create":        {processCreateCommand, nil, usageCreate, usageLongCreate},
		"crop":          {processCropCommand, nil, usageCrop, usageLongCrop},
		"cut":           {processCutCommand, nil, usageCut, usageLongCut},
		"decrypt":       {processDecryptCommand, nil, usageDecrypt, usageLongDecrypt},
		"dump":          {processDumpCommand, nil, "", ""},
		"encrypt":       {processEncryptCommand, nil, usageEncrypt, usageLongEncrypt},
		"extract":       {processExtractCommand, nil, usageExtract, usageLongExtract},
		"fonts":         {nil, fontsCmdMap, usageFonts, usageLongFonts},
		"form":          {nil, formCmdMap, usageForm, usageLongForm},
		"grid":          {processGridCommand, nil, usageGrid, usageLongGrid},
		"help":          {printHelp, nil, "", ""},
		"images":        {nil, imagesCmdMap, usageImages, usageLongImages},
		"import":        {processImportImagesCommand, nil, usageImportImages, usageLongImportImages},
		"info":          {processInfoCommand, nil, usageInfo, usageLongInfo},
		"keywords":      {nil, keywordsCmdMap, usageKeywords, usageLongKeywords},
		"merge":         {processMergeCommand, nil, usageMerge, usageLongMerge},
		"ndown":         {processNDownCommand, nil, usageNDown, usageLongNDown},
		"nup":           {processNUpCommand, nil, usageNUp, usageLongNUp},
		"optimize":      {processOptimizeCommand, nil, usageOptimize, usageLongOptimize},
		"pagelayout":    {nil, pageLayoutCmdMap, usagePageLayout, usageLongPageLayout},
		"pagemode":      {nil, pageModeCmdMap, usagePageMode, usageLongPageMode},
		"pages":         {nil, pagesCmdMap, usagePages, usageLongPages},
		"paper":         {printPaperSizes, nil, usagePaper, usageLongPaper},
		"permissions":   {nil, permissionsCmdMap, usagePerm, usageLongPerm},
		"portfolio":     {nil, portfolioCmdMap, usagePortfolio, usageLongPortfolio},
		"poster":        {processPosterCommand, nil, usagePoster, usageLongPoster},
		"properties":    {nil, propertiesCmdMap, usageProperties, usageLongProperties},
		"resize":        {processResizeCommand, nil, usageResize, usageLongResize},
		"rotate":        {processRotateCommand, nil, usageRotate, usageLongRotate},
		"selectedpages": {printSelectedPages, nil, usageSelectedPages, usageLongSelectedPages},
		"split":         {processSplitCommand, nil, usageSplit, usageLongSplit},
		"stamp":         {nil, stampCmdMap, usageStamp, usageLongStamp},
		"trim":          {processTrimCommand, nil, usageTrim, usageLongTrim},
		"validate":      {processValidateCommand, nil, usageValidate, usageLongValidate},
		"watermark":     {nil, watermarkCmdMap, usageWatermark, usageLongWatermark},
		"version":       {printVersion, nil, usageVersion, usageLongVersion},
		"viewerpref":    {nil, viewerPrefsCmdMap, usageViewerPreferences, usageLongViewerPreferences},
		"zoom":          {processZoomCommand, nil, usageZoom, usageLongZoom},
	} {
		cmdMap.register(k, v)
	}
}

func initFlags() {
	flag.BoolVar(&all, "all", false, "")
	flag.BoolVar(&all, "a", false, "")

	bookmarksUsage := "create bookmarks while merging"
	flag.BoolVar(&bookmarks, "bookmarks", true, bookmarksUsage)
	flag.BoolVar(&bookmarks, "b", true, bookmarksUsage)

	confUsage := "the config directory path | skip | none"
	flag.StringVar(&conf, "config", "", confUsage)
	flag.StringVar(&conf, "conf", "", confUsage)
	flag.StringVar(&conf, "c", "", confUsage)

	dividerPageUsage := "create divider pages while merging"
	flag.BoolVar(&dividerPage, "dividerPage", false, dividerPageUsage)
	flag.BoolVar(&dividerPage, "d", false, dividerPageUsage)

	jsonUsage := "produce JSON output"
	flag.BoolVar(&json, "json", false, jsonUsage)
	flag.BoolVar(&json, "j", false, jsonUsage)

	keyUsage := "encrypt: 40|128|256"
	flag.StringVar(&key, "key", "256", keyUsage)
	flag.StringVar(&key, "k", "256", keyUsage)

	linksUsage := "check for broken links"
	flag.BoolVar(&links, "links", false, linksUsage)
	flag.BoolVar(&links, "l", false, linksUsage)

	modeUsage := "validate: strict|relaxed; extract: image|font|content|page|meta; encrypt: rc4|aes, stamp:text|image/pdf"
	flag.StringVar(&mode, "mode", "", modeUsage)
	flag.StringVar(&mode, "m", "", modeUsage)

	selectedPagesUsage := "a comma separated list of pages or page ranges, see pdfcpu selectedpages"
	flag.StringVar(&selectedPages, "pages", "", selectedPagesUsage)
	flag.StringVar(&selectedPages, "p", "", selectedPagesUsage)

	permUsage := "encrypt, perm set: none|all"
	flag.StringVar(&perm, "perm", "none", permUsage)

	flag.BoolVar(&quiet, "quiet", false, "")
	flag.BoolVar(&quiet, "q", false, "")

	replaceUsage := "replace existing bookmarks"
	flag.BoolVar(&replaceBookmarks, "replace", false, replaceUsage)
	flag.BoolVar(&replaceBookmarks, "r", false, replaceUsage)

	sortUsage := "sort files before merging"
	flag.BoolVar(&sorted, "sort", false, sortUsage)
	flag.BoolVar(&sorted, "s", false, sortUsage)

	statsUsage := "optimize: create a csv file for stats"
	flag.StringVar(&fileStats, "stats", "", statsUsage)

	unitUsage := "info: po|in|cm|mm"
	flag.StringVar(&unit, "unit", "", unitUsage)
	flag.StringVar(&unit, "u", "", unitUsage)

	flag.StringVar(&upw, "upw", "", "user password")
	flag.StringVar(&opw, "opw", "", "owner password")

	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")
	flag.BoolVar(&veryVerbose, "vv", false, "")
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

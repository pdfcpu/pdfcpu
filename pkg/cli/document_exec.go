/*
Copyright 2026 The pdfcpu Authors.

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

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strconv"
	"time"

	"encoding/json"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validationInputLabel(fn string) string {
	if fn == "-" {
		return "stdin"
	}
	return fn
}

func reportValidationProgress(w io.Writer, conf *model.Configuration, fn string) error {
	_, err := fmt.Fprintf(w, "validating(mode=%s) %s ...\n", conf.ValidationModeString(), validationInputLabel(fn))
	return err
}

func validationProgressObserver(w io.Writer, conf *model.Configuration) api.ProgressObserver {
	return func(event api.ProgressEvent) error {
		switch event.Stage {
		case api.ProgressStageReading:
			if w == nil {
				log.CLI.Printf("validating(mode=%s) %s ...\n", conf.ValidationModeString(), validationInputLabel(event.Input))
				return nil
			}
			if err := reportValidationProgress(w, conf, event.Input); err != nil {
				return fmt.Errorf("write validation progress: %w", err)
			}
		case api.ProgressStageOptimizing:
			if w == nil {
				log.CLI.Println("optimizing...")
				return nil
			}
			if _, err := fmt.Fprintln(w, "optimizing..."); err != nil {
				return fmt.Errorf("write validation progress: %w", err)
			}
		}
		return nil
	}
}

func validationNoticeText(notice model.ValidationNotice) string {
	message := notice.Message
	cause := ""
	if notice.Cause != nil && notice.Cause.Error() != message {
		cause = notice.Cause.Error()
	}
	if notice.ObjectNumber > 0 {
		message += fmt.Sprintf(" (obj#:%d)", notice.ObjectNumber)
	}
	if cause != "" {
		if message != "" {
			message += ": "
		}
		message += cause
	}
	return fmt.Sprintf("pdfcpu %s: %s", notice.Disposition, message)
}

func reportValidationNotices(w io.Writer, report model.ValidationReport) error {
	if w == nil {
		return nil
	}
	for i, notice := range report.Notices() {
		if _, err := fmt.Fprintln(w, validationNoticeText(notice)); err != nil {
			return fmt.Errorf("write validation notice %d: %w", i+1, err)
		}
	}
	return nil
}

func validateInput(c context.Context, fn string, conf *model.Configuration, noticeOutput, progressOutput io.Writer, item, total int) error {
	options := api.ProgressOptions{
		Observer: validationProgressObserver(progressOutput, conf),
		Input:    fn,
		Item:     item,
		Total:    total,
	}

	var report model.ValidationReport
	var err error
	if fn != "-" {
		report, err = api.ValidateFileWithReport(c, fn, conf, &options)
	} else {
		report, err = withStdinReadSeeker(c, conf, "validate", func(rs io.ReadSeeker) (model.ValidationReport, error) {
			return api.ValidateWithReport(c, rs, conf, &options)
		})
	}
	if noticeErr := reportValidationNotices(noticeOutput, report); noticeErr != nil {
		return errors.Join(err, noticeErr)
	}
	if err != nil {
		return err
	}
	log.CLI.Println("validation ok")
	return nil
}

func reportValidationError(w io.Writer, err error) error {
	if _, writeErr := fmt.Fprintln(w, err); writeErr != nil {
		return errors.Join(err, fmt.Errorf("write validation error: %w", writeErr))
	}
	return nil
}

func validateInputs(c context.Context, inFiles []string, conf *model.Configuration, errorOutput, noticeOutput, progressOutput io.Writer) error {
	var errs []error
	failures := 0
	for i, fn := range inFiles {
		if i > 0 {
			log.CLI.Println()
		}

		err := validateInput(c, fn, conf, noticeOutput, progressOutput, i+1, len(inFiles))
		if err == nil {
			continue
		}
		if cancelErr := c.Err(); cancelErr != nil {
			return cancelErr
		}

		if errorOutput == nil {
			errs = append(errs, fmt.Errorf("%s: %w", fn, err))
			continue
		}

		failures++
		if err := reportValidationError(errorOutput, err); err != nil {
			return err
		}
	}

	if failures > 0 {
		return fmt.Errorf("validation failed: %d of %d files invalid", failures, len(inFiles))
	}
	return errors.Join(errs...)
}

func validateCommand(c context.Context, cmd *Command) ([]string, error) {
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:     "validate",
		minInputFiles: 1,
	}); err != nil {
		return nil, err
	}
	conf := cmd.Conf
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	if len(cmd.InFiles) == 1 {
		var progressOutput io.Writer
		if cmd.BoolVal1 {
			progressOutput = cmd.ErrorOutput
		}
		return nil, validateInput(c, cmd.InFiles[0], conf, cmd.NoticeOutput, progressOutput, 1, 1)
	}

	var progressOutput io.Writer
	if cmd.BoolVal1 {
		progressOutput = cmd.ErrorOutput
	}
	return nil, validateInputs(c, cmd.InFiles, conf, cmd.ErrorOutput, cmd.NoticeOutput, progressOutput)
}

func optimizationProgressObserver(cmd *Command) api.ProgressObserver {
	if commandWritesPDFToStdout(cmd) {
		return nil
	}
	return func(event api.ProgressEvent) error {
		switch event.Stage {
		case api.ProgressStageOptimizing:
			reportCommandProgress(cmd, "optimizing...\n")
		case api.ProgressStageWriting:
			reportCommandOutputPath(cmd)
		}
		return nil
	}
}

func optimize(c context.Context, cmd *Command) ([]string, error) {
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation: "optimize",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}); err != nil {
		return nil, err
	}
	options := api.ProgressOptions{
		Observer: optimizationProgressObserver(cmd),
		Input:    *cmd.InFile,
		Item:     1,
		Total:    1,
	}
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.OptimizeFile(c, *cmd.InFile, *cmd.OutFile, cmd.Conf, &options)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "optimize")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Optimize(c, rs, w, cmd.Conf, &options))
}

func mergeStdinCount(inFiles []string) int {
	count := 0
	for _, fn := range inFiles {
		if fn == "-" {
			count++
		}
	}
	return count
}

func mergeReader(c context.Context, conf *model.Configuration, fn string, source int) (io.ReadSeeker, *os.File, *temporaryInput, error) {
	if fn == "-" {
		in, err := readSeekerFromStdin(c, conf, fmt.Sprintf("merge source %d", source))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("merge source %d: read source: %w", source, err)
		}
		return in, nil, in, nil
	}
	f, err := os.Open(fn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("merge source %d: read source: %w", source, err)
	}
	return f, f, nil, nil
}

func closeMergeInputs(files []*os.File) error {
	errs := make([]error, 0, len(files))
	for i, f := range files {
		if err := f.Close(); err != nil {
			errs = append(errs, fmt.Errorf("merge source %d: close input: %w", i+1, err))
		}
	}
	return errors.Join(errs...)
}

func mergeReaders(c context.Context, conf *model.Configuration, inFiles []string) ([]io.ReadSeeker, []*os.File, *temporaryInput, error) {
	if c == nil {
		return nil, nil, nil, ErrMissingContext
	}
	readers := make([]io.ReadSeeker, 0, len(inFiles))
	files := make([]*os.File, 0, len(inFiles))
	var temporaryIn *temporaryInput
	for i, fn := range inFiles {
		if err := c.Err(); err != nil {
			err = errors.Join(err, closeMergeInputs(files))
			if temporaryIn != nil {
				err = temporaryIn.finalize("merge", err)
			}
			return nil, nil, nil, err
		}
		rs, f, in, err := mergeReader(c, conf, fn, i)
		if err != nil {
			err = errors.Join(err, closeMergeInputs(files))
			if temporaryIn != nil {
				err = temporaryIn.finalize("merge", err)
			}
			return nil, nil, nil, err
		}
		if f != nil {
			files = append(files, f)
		}
		readers = append(readers, rs)
		if in != nil {
			temporaryIn = in
		}
	}
	return readers, files, temporaryIn, nil
}

func mergeCreateRaw(c context.Context, cmd *Command) ([]string, error) {
	readers, files, temporaryIn, err := mergeReaders(c, cmd.Conf, cmd.InFiles)
	if err != nil {
		return nil, err
	}

	_, w, finalize, err := streamInOutForOperation(c, cmd.Conf, "", *cmd.OutFile, "merge")
	if err != nil {
		err = errors.Join(err, closeMergeInputs(files))
		if temporaryIn != nil {
			err = temporaryIn.finalize("merge", err)
		}
		return nil, err
	}

	err = errors.Join(api.MergeRaw(c, readers, w, cmd.BoolVal1, cmd.Conf), closeMergeInputs(files))
	if temporaryIn != nil {
		err = temporaryIn.finalize("merge", err)
	}
	return nil, finalize(err)
}

func reportMergeProgress(cmd *Command) {
	if commandWritesPDFToStdout(cmd) {
		return
	}
	reportCommandOutputPath(cmd)
	if cmd.Conf != nil && cmd.Conf.CreateBookmarks {
		reportCommandProgress(cmd, "creating bookmarks...\n")
	}
	for _, inFile := range cmd.InFiles {
		reportCommandProgress(cmd, "%s\n", inFile)
	}
}

func mergeCreate(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:     "merge",
		outFile:       commandStringRequiredNonEmpty,
		minInputFiles: 1,
	}); err != nil {
		return nil, err
	}
	stdinCount := mergeStdinCount(cmd.InFiles)
	if stdinCount > 1 {
		return nil, fmt.Errorf("pdfcpu: merge: only one stdin input supported")
	}
	reportMergeProgress(cmd)
	if stdinCount == 1 {
		return mergeCreateRaw(c, cmd)
	}
	if *cmd.OutFile == "-" {
		log.SetCLILogger(nil)
		return nil, api.Merge(c, "", cmd.InFiles, os.Stdout, cmd.Conf, cmd.BoolVal1)
	}
	return nil, api.MergeCreateFile(c, cmd.InFiles, *cmd.OutFile, cmd.BoolVal1, cmd.Conf)
}

func mergeCreateZip(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:     "merge zip",
		outFile:       commandStringRequiredNonEmpty,
		minInputFiles: 2,
		maxInputFiles: 2,
	}); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.OutFile != "-" {
		return nil, api.MergeCreateZipFile(c, cmd.InFiles[0], cmd.InFiles[1], *cmd.OutFile, cmd.Conf)
	}
	log.SetCLILogger(nil)
	f1, err := os.Open(cmd.InFiles[0])
	if err != nil {
		return nil, err
	}
	defer f1.Close()

	f2, err := os.Open(cmd.InFiles[1])
	if err != nil {
		return nil, err
	}
	defer f2.Close()

	return nil, api.MergeCreateZip(c, f1, f2, os.Stdout, cmd.Conf)
}

func mergeAppend(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:     "merge append",
		outFile:       commandStringRequiredNonEmpty,
		minInputFiles: 1,
	}); err != nil {
		return nil, err
	}
	if *cmd.OutFile == "-" {
		return nil, fmt.Errorf("pdfcpu: merge append: stdout not supported")
	}
	if _, err := os.Stat(*cmd.OutFile); err == nil {
		reportCommandProgress(cmd, "appending to %s...\n", *cmd.OutFile)
	} else {
		reportCommandOutputPath(cmd)
	}
	if cmd.Conf != nil && cmd.Conf.CreateBookmarks {
		reportCommandProgress(cmd, "creating bookmarks...\n")
	}
	for _, inFile := range cmd.InFiles {
		reportCommandProgress(cmd, "%s\n", inFile)
	}
	return nil, api.MergeAppendFile(c, cmd.InFiles, *cmd.OutFile, cmd.BoolVal1, cmd.Conf)
}

func reportSplitProgress(cmd *Command, inFile string) {
	reportCommandProgress(cmd, "splitting %s to %s/...\n", inFile, *cmd.OutDir)
}

func split(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation: "split",
		inFile:    commandStringRequiredNonEmpty,
		outDir:    commandStringRequiredNonEmpty,
	}); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		reportSplitProgress(cmd, "stdin.pdf")
		return withStdinReadSeeker(c, cmd.Conf, "split", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.Split(c, rs, *cmd.OutDir, "stdin.pdf", cmd.IntVal, cmd.Conf)
		})
	}
	reportSplitProgress(cmd, *cmd.InFile)
	return nil, api.SplitFile(c, *cmd.InFile, *cmd.OutDir, cmd.IntVal, cmd.Conf)
}

func splitByPageNr(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:        "split by page number",
		inFile:           commandStringRequiredNonEmpty,
		outDir:           commandStringRequiredNonEmpty,
		minIntValues:     1,
		missingIntValues: api.ErrMissingSplitPageNumbers,
	}); err != nil {
		return nil, err
	}
	if *cmd.InFile == "-" {
		reportSplitProgress(cmd, "stdin.pdf")
		return withStdinReadSeeker(c, cmd.Conf, "split by page number", func(rs io.ReadSeeker) ([]string, error) {
			return nil, api.SplitByPageNr(c, rs, *cmd.OutDir, "stdin.pdf", cmd.IntVals, cmd.Conf)
		})
	}
	reportSplitProgress(cmd, *cmd.InFile)
	return nil, api.SplitByPageNrFile(c, *cmd.InFile, *cmd.OutDir, cmd.IntVals, cmd.Conf)
}

func trim(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation: "trim",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.TrimFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "trim")
	if err != nil {
		return nil, fmt.Errorf("trim: prepare input/output: %w", err)
	}
	return nil, finalize(api.Trim(c, rs, w, cmd.PageSelection, cmd.Conf))
}

func collect(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation: "collect",
		inFile:    commandStringRequiredNonEmpty,
		outFile:   commandStringRequired,
	}); err != nil {
		return nil, err
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.CollectFile(c, *cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Conf)
	}

	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "collect")
	if err != nil {
		return nil, err
	}
	return nil, finalize(api.Collect(c, rs, w, cmd.PageSelection, cmd.Conf))
}

func listInfo(c context.Context, rs io.ReadSeeker, inFile string, selectedPages []string, fonts bool, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	info, err := api.PDFInfo(c, rs, inFile, selectedPages, fonts, conf)
	if err != nil {
		return nil, err
	}

	pages, err := api.PagesForSelection(info.PageCount, selectedPages, false)
	if err != nil {
		return nil, err
	}

	ss, err := pdfcpu.ListInfo(c, info, pages, fonts)
	if err != nil {
		return nil, fmt.Errorf("list info: render output: %w", err)
	}

	return append([]string{inFile + ":"}, ss...), err
}

// ListInfoFile returns formatted information about inFile and supports cancellation.
func ListInfoFile(c context.Context, inFile string, selectedPages []string, fonts bool, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, commandValidationError("list info", api.ErrMissingPDFInput)
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("list info: open %s: %w", inFile, err)
	}
	defer f.Close()

	return listInfo(c, f, inFile, selectedPages, fonts, conf)
}

func normalizeInfoBox(box *model.Box, unit types.DisplayUnit) {
	if box == nil {
		return
	}
	box.Rect = box.Rect.ConvertToUnit(unit)
	box.Rect.LL.X = math.Round(box.Rect.LL.X*100) / 100
	box.Rect.LL.Y = math.Round(box.Rect.LL.Y*100) / 100
	box.Rect.UR.X = math.Round(box.Rect.UR.X*100) / 100
	box.Rect.UR.Y = math.Round(box.Rect.UR.Y*100) / 100
}

func normalizeInfoPageBoundaries(pb *model.PageBoundaries, unit types.DisplayUnit) {
	d := pb.CropBox().Dimensions()
	if pb.Rot%180 != 0 {
		d.Width, d.Height = d.Height, d.Width
	}
	pb.Orientation = "portrait"
	if d.Landscape() {
		pb.Orientation = "landscape"
	}
	normalizeInfoBox(pb.Media, unit)
	normalizeInfoBox(pb.Crop, unit)
	normalizeInfoBox(pb.Trim, unit)
	normalizeInfoBox(pb.Bleed, unit)
	normalizeInfoBox(pb.Art, unit)
}

func jsonInfo(c context.Context, info *pdfcpu.PDFInfo, pages types.IntSet) (map[string]model.PageBoundaries, []types.Dim, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, nil, err
	}
	if len(pages) > 0 {
		pbs := map[string]model.PageBoundaries{}
		for i, pb := range info.PageBoundaries {
			if err := c.Err(); err != nil {
				return nil, nil, err
			}
			if _, found := pages[i+1]; !found {
				continue
			}
			normalizeInfoPageBoundaries(&pb, info.Unit)
			pbs[strconv.Itoa(i+1)] = pb
		}
		return pbs, nil, c.Err()
	}

	var dims []types.Dim
	for k, v := range info.PageDimensions {
		if err := c.Err(); err != nil {
			return nil, nil, err
		}
		if v {
			dc := k.ConvertToUnit(info.Unit)
			dc.Width = math.Round(dc.Width*100) / 100
			dc.Height = math.Round(dc.Height*100) / 100
			dims = append(dims, dc)
		}
	}
	return nil, dims, c.Err()
}

func listInfoJSON(c context.Context, rs io.ReadSeeker, inFile string, selectedPages []string, fonts bool, conf *model.Configuration) (*pdfcpu.PDFInfo, error) {
	info, err := api.PDFInfo(c, rs, inFile, selectedPages, fonts, conf)
	if err != nil {
		return nil, err
	}

	pages, err := api.PagesForSelection(info.PageCount, selectedPages, false)
	if err != nil {
		return nil, err
	}

	info.Boundaries, info.Dimensions, err = jsonInfo(c, info, pages)
	if err != nil {
		return nil, err
	}

	return info, c.Err()
}

type infoJSONContextProcessor func(
	context.Context,
	io.ReadSeeker,
	string,
	[]string,
	bool,
	*model.Configuration,
) (*pdfcpu.PDFInfo, error)

func listInfoFileJSON(c context.Context, fn string, selectedPages []string, fonts bool, conf *model.Configuration, process infoJSONContextProcessor) (*pdfcpu.PDFInfo, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("list info: open %s: %w", fn, err)
	}

	info, processErr := process(c, f, fn, selectedPages, fonts, conf)
	return info, errors.Join(processErr, f.Close())
}

func listInfoFilesJSON(c context.Context, inFiles []string, selectedPages []string, fonts bool, conf *model.Configuration) ([]string, error) {
	var infos []*pdfcpu.PDFInfo

	for _, fn := range inFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		info, err := listInfoFileJSON(c, fn, selectedPages, fonts, conf, listInfoJSON)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	return jsonInfoOutput(c, infos)
}

func jsonInfoOutput(c context.Context, infos []*pdfcpu.PDFInfo) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	s := struct {
		Header pdfcpu.Header     `json:"header"`
		Infos  []*pdfcpu.PDFInfo `json:"infos"`
	}{
		Header: pdfcpu.Header{Version: "pdfcpu " + model.VersionStr, Creation: time.Now().Format("2006-01-02 15:04:05 MST")},
		Infos:  infos,
	}

	bb, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("list info: encode JSON: %w", err)
	}

	return []string{string(bb)}, c.Err()
}

// ListInfoFiles returns formatted information about inFiles and supports cancellation.
func ListInfoFiles(c context.Context, inFiles []string, selectedPages []string, fonts, json bool, conf *model.Configuration) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandInputFiles(inFiles, 1, 0, nil); err != nil {
		return nil, commandValidationError("list info", err)
	}
	if json {
		return listInfoFilesJSON(c, inFiles, selectedPages, fonts, conf)
	}

	var ss []string
	var errs []error

	for i, fn := range inFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		if i > 0 {
			ss = append(ss, "")
		}
		ssx, err := ListInfoFile(c, fn, selectedPages, fonts, conf)
		if err != nil {
			if len(inFiles) == 1 {
				return nil, err
			}
			errs = append(errs, err)
			continue
		}
		ss = append(ss, ssx...)
	}

	return ss, errors.Join(errors.Join(errs...), c.Err())
}

func listInfoInput(c context.Context, fn string, selectedPages []string, fonts, json bool, conf *model.Configuration) ([]string, *pdfcpu.PDFInfo, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, nil, err
	}
	if fn == "-" {
		type result struct {
			ss   []string
			info *pdfcpu.PDFInfo
		}
		r, err := withStdinReadSeeker(c, conf, "list info", func(rs io.ReadSeeker) (result, error) {
			ss, info, err := listInfoReadSeeker(c, rs, fn, selectedPages, fonts, json, conf)
			return result{ss: ss, info: info}, err
		})
		return r.ss, r.info, err
	}
	rs, err := os.Open(fn)
	if err != nil {
		return nil, nil, fmt.Errorf("list info: open %s: %w", fn, err)
	}
	defer rs.Close()

	return listInfoReadSeeker(c, rs, fn, selectedPages, fonts, json, conf)
}

func listInfoReadSeeker(c context.Context, rs io.ReadSeeker, fn string, selectedPages []string, fonts, json bool, conf *model.Configuration) ([]string, *pdfcpu.PDFInfo, error) {
	if json {
		info, err := listInfoJSON(c, rs, fn, selectedPages, fonts, conf)
		return nil, info, err
	}
	ss, err := listInfo(c, rs, fn, selectedPages, fonts, conf)
	return ss, nil, err
}

func reportRelaxedValidation(cmd *Command, operation string) error {
	if cmd.ErrorOutput == nil {
		return nil
	}
	if _, err := fmt.Fprintf(cmd.ErrorOutput, "%s: using relaxed validation\n", operation); err != nil {
		return fmt.Errorf("%s: report validation mode: %w", operation, err)
	}
	return nil
}

func listInfoInputs(c context.Context, cmd *Command) ([]string, error) {
	var ss []string
	var infos []*pdfcpu.PDFInfo
	var errs []error
	for i, fn := range cmd.InFiles {
		if err := c.Err(); err != nil {
			return nil, err
		}
		if i > 0 && !cmd.BoolVal2 {
			ss = append(ss, "")
		}

		ssx, info, err := listInfoInput(c, fn, cmd.PageSelection, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
		if err != nil {
			if len(cmd.InFiles) == 1 {
				return nil, err
			}
			errs = append(errs, err)
			continue
		}
		if cmd.BoolVal2 {
			infos = append(infos, info)
			continue
		}
		ss = append(ss, ssx...)
	}

	if err := errors.Join(errs...); err != nil {
		return ss, err
	}
	if cmd.BoolVal2 {
		return jsonInfoOutput(c, infos)
	}
	return ss, c.Err()
}

func listInfoCommand(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:     "list info",
		minInputFiles: 1,
	}); err != nil {
		return nil, err
	}
	if err := reportRelaxedValidation(cmd, "info"); err != nil {
		return nil, err
	}
	if !slices.Contains(cmd.InFiles, "-") {
		return ListInfoFiles(c, cmd.InFiles, cmd.PageSelection, cmd.BoolVal1, cmd.BoolVal2, cmd.Conf)
	}
	return listInfoInputs(c, cmd)
}

func dump(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:        "dump",
		inFile:           commandStringRequiredNonEmpty,
		minIntValues:     2,
		missingIntValues: ErrInvalidCommandArguments,
	}); err != nil {
		return nil, err
	}
	if err := reportRelaxedValidation(cmd, "dump"); err != nil {
		return nil, err
	}
	mode := cmd.IntVals[0]
	objNr := cmd.IntVals[1]

	conf := model.NewDefaultConfiguration()
	if cmd.Conf != nil {
		conf = cmd.Conf.Clone()
	}
	conf.Cmd = model.DUMP
	conf.ValidationMode = model.ValidationRelaxed

	f, err := os.Open(*cmd.InFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ctx, err := api.ReadContext(c, f, conf)
	if err != nil {
		return nil, fmt.Errorf("read context: %w", err)
	}

	if err = api.ValidateContext(c, ctx); err != nil {
		return nil, dumpValidationError(err)
	}

	if err := ctx.DumpObject(c, objNr, mode); err != nil {
		return nil, fmt.Errorf("dump object %d: %w", objNr, err)
	}
	return nil, c.Err()
}

func dumpValidationError(err error) error {
	prefix := "validation error"
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		prefix += fmt.Sprintf(" (obj#:%d)", validationErr.ObjectNumber())
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func create(c context.Context, cmd *Command) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := validateCommandRequirements(cmd, commandRequirements{
		operation:  "create",
		inFile:     commandStringRequired,
		outFile:    commandStringRequired,
		inFileJSON: commandStringRequiredNonEmpty,
	}); err != nil {
		return nil, err
	}
	if *cmd.InFile == "" && *cmd.OutFile == "" {
		return nil, commandValidationError("create", api.ErrMissingPDFInput)
	}
	if *cmd.InFile != "" && *cmd.InFile != "-" {
		reportCommandProgress(cmd, "reading %s...\n", *cmd.InFile)
	}
	reportCommandOutputPath(cmd)
	if *cmd.InFile != "-" && *cmd.OutFile != "-" {
		return nil, api.CreateFile(c, *cmd.InFile, *cmd.InFileJSON, *cmd.OutFile, cmd.Conf)
	}

	rd, err := os.Open(*cmd.InFileJSON)
	if err != nil {
		return nil, err
	}
	rs, w, finalize, err := streamInOutForOperation(c, cmd.Conf, *cmd.InFile, *cmd.OutFile, "create")
	if err != nil {
		_ = rd.Close()
		return nil, err
	}
	opErr := api.Create(c, rs, rd, w, cmd.Conf)
	opErr = errors.Join(opErr, closeStreamFile(rd, "create: close JSON input"))
	return nil, finalize(opErr)
}

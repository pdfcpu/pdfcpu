/*
Copyright 2023 The pdfcpu Authors.

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

package api

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/fileutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func cloneCutConfiguration(cut *model.Cut) *model.Cut {
	if cut == nil {
		return nil
	}
	clone := *cut
	clone.Hor = slices.Clone(cut.Hor)
	clone.Vert = slices.Clone(cut.Vert)
	if cut.PageDim != nil {
		dim := *cut.PageDim
		clone.PageDim = &dim
	}
	if cut.BgColor != nil {
		color := *cut.BgColor
		clone.BgColor = &color
	}
	return &clone
}

func invalidCutNumber(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0)
}

func validateCutOptions(cut *model.Cut) error {
	if cut == nil {
		return ErrMissingCutConfiguration
	}
	if invalidCutNumber(cut.Margin) || cut.Margin < 0 {
		return fmt.Errorf("margin must be finite and >= 0: %w", ErrInvalidCutConfiguration)
	}
	return nil
}

func validateCutPoints(name string, points []float64) error {
	for i, f := range points {
		if invalidCutNumber(f) || f < 0 || f >= 1 {
			return fmt.Errorf("%s cut point %d must be finite and >= 0 and < 1: %w", name, i+1, ErrInvalidCutConfiguration)
		}
	}
	sorted := slices.Clone(points)
	sort.Float64s(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			return fmt.Errorf("duplicate %s cut point %.5f: %w", name, sorted[i], ErrInvalidCutConfiguration)
		}
	}
	return nil
}

func validateCutConfiguration(cut *model.Cut) error {
	if err := validateCutOptions(cut); err != nil {
		return err
	}
	if len(cut.Hor) == 0 && len(cut.Vert) == 0 {
		return fmt.Errorf("missing horizontal or vertical cut points: %w", ErrInvalidCutConfiguration)
	}
	if err := validateCutPoints("horizontal", cut.Hor); err != nil {
		return err
	}
	return validateCutPoints("vertical", cut.Vert)
}

func validateNDownConfiguration(n int, cut *model.Cut) error {
	if err := validateCutOptions(cut); err != nil {
		return err
	}
	switch n {
	case 2, 3, 4, 6, 8, 9, 12, 16:
		return nil
	}
	return fmt.Errorf("n-down value %d must be one of 2, 3, 4, 6, 8, 9, 12, 16: %w", n, ErrInvalidCutConfiguration)
}

func validatePosterConfiguration(cut *model.Cut) error {
	if err := validateCutOptions(cut); err != nil {
		return err
	}
	if cut.PageSize == "" && !cut.UserDim {
		return fmt.Errorf("missing dimensions or form size: %w", ErrInvalidCutConfiguration)
	}
	if invalidCutNumber(cut.Scale) || cut.Scale < 1 {
		return fmt.Errorf("scale factor must be finite and >= 1: %w", ErrInvalidCutConfiguration)
	}
	if cut.PageDim == nil {
		return fmt.Errorf("missing dimensions: %w", ErrInvalidCutConfiguration)
	}
	w, h := cut.PageDim.Width, cut.PageDim.Height
	if invalidCutNumber(w) || invalidCutNumber(h) || w <= 0 || h <= 0 {
		return fmt.Errorf("dimensions must be finite and > 0: %w", ErrInvalidCutConfiguration)
	}
	return nil
}

func selectedCutPages(pageCount int, selectedPages []string, operation string) ([]int, error) {
	pages, err := PagesForPageSelection(pageCount, selectedPages, true, true)
	if err != nil {
		return nil, fmt.Errorf("%s: parse page selection: %w", operation, err)
	}
	return sortedPages(pages), nil
}

func prepareForCut(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration, operation string) (*model.Context, []int, error) {
	if rs == nil {
		return nil, nil, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", operation, err)
	}

	pages, err := selectedCutPages(ctx.PageCount, selectedPages, operation)
	if err != nil {
		return nil, nil, err
	}
	return ctx, pages, nil
}

type cutOutputFile interface {
	io.Writer
	io.Closer
	Chmod(os.FileMode) error
	Name() string
}

type cutOutputOperations struct {
	stat          func(string) (os.FileInfo, error)
	createTemp    func(string, string) (cutOutputFile, error)
	writeAndFlush func(*model.Context, io.Writer) (error, error)
	rename        func(string, string) error
	remove        func(string) error
}

func createCutTemporaryOutput(dir, pattern string) (cutOutputFile, error) {
	name := strings.Replace(pattern, "*", rand.Text(), 1)
	return os.OpenFile(filepath.Join(dir, name), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
}

func defaultCutOutputOperations() cutOutputOperations {
	return cutOutputOperations{
		stat:          os.Stat,
		createTemp:    createCutTemporaryOutput,
		writeAndFlush: writeAndFlushCutContext,
		rename:        fileutil.ReplaceFile,
		remove:        os.Remove,
	}
}

func cutDestinationMode(outFile, operation string, ops cutOutputOperations) (os.FileMode, bool, error) {
	stat := ops.stat
	if stat == nil {
		stat = os.Stat
	}
	info, err := stat(outFile)
	if err == nil {
		return info.Mode().Perm(), true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	return 0, false, fmt.Errorf("%s: write output %s: stat destination: %w", operation, outFile, err)
}

func writeAndFlushCutContext(ctx *model.Context, w io.Writer) (error, error) {
	if f, ok := w.(*os.File); ok {
		ctx.Write.Fp = f
	}
	ctx.Write.Writer = bufio.NewWriter(w)
	writeErr := pdfcpu.WriteContext(ctx)
	return writeErr, ctx.Write.Flush()
}

func cutOutputPhaseError(err error, operation, outFile, phase string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: write output %s: %s: %w", operation, outFile, phase, err)
}

func removeCutTemporaryOutput(tmpFile, operation, outFile string, ops cutOutputOperations) error {
	if err := ops.remove(tmpFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: write output %s: remove temporary output: %w", operation, outFile, err)
	}
	return nil
}

func writeCutOutputWith(ctx *model.Context, outFile, operation string, ops cutOutputOperations) (err error) {
	logWritingTo(outFile)
	destinationMode, destinationExists, err := cutDestinationMode(outFile, operation, ops)
	if err != nil {
		return err
	}
	pattern := "." + filepath.Base(outFile) + ".tmp-*"
	f, err := ops.createTemp(filepath.Dir(outFile), pattern)
	if err != nil {
		return fmt.Errorf("%s: write output %s: create temporary output: %w", operation, outFile, err)
	}
	closed := false
	committed := false
	defer func() {
		if !closed {
			err = errors.Join(err, cutOutputPhaseError(f.Close(), operation, outFile, "close"))
			closed = true
		}
		if !committed {
			err = errors.Join(err, removeCutTemporaryOutput(f.Name(), operation, outFile, ops))
		}
	}()
	defer fault.Catch(&err)
	if destinationExists {
		if err := f.Chmod(destinationMode); err != nil {
			return fmt.Errorf("%s: write output %s: set temporary output permissions: %w", operation, outFile, err)
		}
	}

	writeErr, flushErr := ops.writeAndFlush(ctx, f)
	closeErr := f.Close()
	closed = true
	err = errors.Join(
		cutOutputPhaseError(writeErr, operation, outFile, "write"),
		cutOutputPhaseError(flushErr, operation, outFile, "flush"),
		cutOutputPhaseError(closeErr, operation, outFile, "close"),
	)
	if err != nil {
		return err
	}
	if err := ops.rename(f.Name(), outFile); err != nil {
		return fmt.Errorf("%s: write output %s: rename temporary output: %w", operation, outFile, err)
	}
	committed = true
	return nil
}

func writeCutOutput(ctx *model.Context, outFile, operation string) error {
	return writeCutOutputWith(ctx, outFile, operation, defaultCutOutputOperations())
}

func writePosterPage(ctxSrc *model.Context, pageNr int, outDir, fileName string, cut *model.Cut, conf *model.Configuration) error {
	ctxDest, err := pdfcpu.PosterPage(ctxSrc, pageNr, cut)
	if err != nil {
		return fmt.Errorf("poster: process page %d: %w", pageNr, err)
	}

	outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))

	if conf.PostProcessValidate {
		if err = ValidateContext(ctxDest); err != nil {
			return fmt.Errorf("poster: validate output page %d: %w", pageNr, err)
		}
	}

	return writeCutOutput(ctxDest, outFile, "poster")
}

// Poster applies cut for selected pages of rs and generates corresponding poster tiles in outDir.
// Each generated output is written atomically. Outputs completed for earlier pages remain if a later page fails.
func Poster(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	cut = cloneCutConfiguration(cut)
	if err := validatePosterConfiguration(cut); err != nil {
		return fmt.Errorf("poster: validate configuration: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.POSTER
	fileName = sanitizeFilenamePart(fileName, "poster")

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf, "poster")
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: nothing to cut!")
		}
		return nil
	}

	for _, pageNr := range pages {
		if err := writePosterPage(ctxSrc, pageNr, outDir, fileName, cut, conf); err != nil {
			return err
		}
	}

	return nil
}

// PosterFile applies cut for selected pages of inFile and generates corresponding poster tiles in outDir.
// Each generated output is written atomically. Outputs completed for earlier pages remain if a later page fails.
func PosterFile(inFile, outDir, outFile string, selectedPages []string, cut *model.Cut, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	cut = cloneCutConfiguration(cut)
	if err := validatePosterConfiguration(cut); err != nil {
		return fmt.Errorf("poster: validate configuration: %w", err)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("poster: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "poster: close input"))
	}()

	if log.CLIEnabled() {
		log.CLI.Printf("creating poster pages from %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	return Poster(f, outDir, outFile, selectedPages, cut, conf)
}

func writeNDownPage(ctxSrc *model.Context, pageNr, n int, outDir, fileName string, cut *model.Cut, conf *model.Configuration) error {
	ctxDest, err := pdfcpu.NDownPage(ctxSrc, pageNr, n, cut)
	if err != nil {
		return fmt.Errorf("ndown: process page %d: %w", pageNr, err)
	}

	if conf.PostProcessValidate {
		if err = ValidateContext(ctxDest); err != nil {
			return fmt.Errorf("ndown: validate output page %d: %w", pageNr, err)
		}
	}

	outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
	return writeCutOutput(ctxDest, outFile, "ndown")
}

// NDown applies n & cutConf for selected pages of rs and writes results to outDir.
// Each generated output is written atomically. Outputs completed for earlier pages remain if a later page fails.
func NDown(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, n int, cut *model.Cut, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	cut = cloneCutConfiguration(cut)
	if err := validateNDownConfiguration(n, cut); err != nil {
		return fmt.Errorf("ndown: validate configuration: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NDOWN
	fileName = sanitizeFilenamePart(fileName, "ndown")

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf, "ndown")
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: nothing to cut!")
		}
		return nil
	}

	for _, pageNr := range pages {
		if err := writeNDownPage(ctxSrc, pageNr, n, outDir, fileName, cut, conf); err != nil {
			return err
		}
	}

	return nil
}

// NDownFile applies n & cutConf for selected pages of inFile and writes results to outDir.
// Each generated output is written atomically. Outputs completed for earlier pages remain if a later page fails.
func NDownFile(inFile, outDir, outFile string, selectedPages []string, n int, cut *model.Cut, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	cut = cloneCutConfiguration(cut)
	if err := validateNDownConfiguration(n, cut); err != nil {
		return fmt.Errorf("ndown: validate configuration: %w", err)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("ndown: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "ndown: close input"))
	}()

	if log.CLIEnabled() {
		log.CLI.Printf("ndown %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	return NDown(f, outDir, outFile, selectedPages, n, cut, conf)
}

func normalizeCut(cut *model.Cut) {
	sort.Float64s(cut.Hor)
	if len(cut.Hor) == 0 || cut.Hor[0] > 0 {
		cut.Hor = append([]float64{0}, cut.Hor...)
	}

	sort.Float64s(cut.Vert)
	if len(cut.Vert) == 0 || cut.Vert[0] > 0 {
		cut.Vert = append([]float64{0}, cut.Vert...)
	}
}

func writeCutPage(ctxSrc *model.Context, pageNr int, outDir, fileName string, cut *model.Cut, conf *model.Configuration) error {
	ctxDest, err := pdfcpu.CutPage(ctxSrc, pageNr, cut)
	if err != nil {
		return fmt.Errorf("cut: process page %d: %w", pageNr, err)
	}

	if conf.PostProcessValidate {
		if err = ValidateContext(ctxDest); err != nil {
			return fmt.Errorf("cut: validate output page %d: %w", pageNr, err)
		}
	}

	outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
	return writeCutOutput(ctxDest, outFile, "cut")
}

// Cut applies cutConf for selected pages of rs and writes results to outDir.
// Each generated output is written atomically. Outputs completed for earlier pages remain if a later page fails.
func Cut(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	cut = cloneCutConfiguration(cut)
	if err := validateCutConfiguration(cut); err != nil {
		return fmt.Errorf("cut: validate configuration: %w", err)
	}
	normalizeCut(cut)

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CUT
	fileName = sanitizeFilenamePart(fileName, "cut")

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf, "cut")
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: nothing to cut!")
		}
		return nil
	}

	for _, pageNr := range pages {
		if err := writeCutPage(ctxSrc, pageNr, outDir, fileName, cut, conf); err != nil {
			return err
		}
	}

	return nil
}

// CutFile applies cutConf for selected pages of inFile and writes results to outDir.
// Each generated output is written atomically. Outputs completed for earlier pages remain if a later page fails.
func CutFile(inFile, outDir, outFile string, selectedPages []string, cut *model.Cut, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}
	cut = cloneCutConfiguration(cut)
	if err := validateCutConfiguration(cut); err != nil {
		return fmt.Errorf("cut: validate configuration: %w", err)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("cut: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "cut: close input"))
	}()

	if log.CLIEnabled() {
		log.CLI.Printf("cutting %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	return Cut(f, outDir, outFile, selectedPages, cut, conf)
}

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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func prepareForCut(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (*model.Context, types.IntSet, error) {
	if rs == nil {
		return nil, nil, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("read source PDF context: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, nil, fmt.Errorf("parse page selection: %w", err)
	}

	return ctx, pages, nil
}

func writePosterPage(ctxSrc *model.Context, pageNr int, outDir, fileName string, cut *model.Cut, conf *model.Configuration) error {
	ctxDest, err := pdfcpu.PosterPage(ctxSrc, pageNr, cut)
	if err != nil {
		return fmt.Errorf("create poster page %d: %w", pageNr, err)
	}

	outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
	logWritingTo(outFile)

	if conf.PostProcessValidate {
		if err = ValidateContext(ctxDest); err != nil {
			return fmt.Errorf("validate output page %d: %w", pageNr, err)
		}
	}

	if err := WriteContextFile(ctxDest, outFile); err != nil {
		return fmt.Errorf("write output file %s: %w", outFile, err)
	}

	return nil
}

// Poster applies cut for selected pages of rs and generates corresponding poster tiles in outDir.
func Poster(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if cut == nil {
		return errors.New("missing cut configuration")
	}

	if cut.PageSize == "" && !cut.UserDim {
		return errors.New("missing dimensions or form size")
	}

	if cut.Scale < 1 {
		return fmt.Errorf("invalid scale factor %.2f: must be >= 1.0", cut.Scale)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.POSTER
	fileName = sanitizeFilenamePart(fileName, "poster")

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: nothing to cut!")
		}
		return nil
	}

	for pageNr, v := range pages {
		if !v {
			continue
		}
		if err := writePosterPage(ctxSrc, pageNr, outDir, fileName, cut, conf); err != nil {
			return err
		}
	}

	return nil
}

// PosterFile applies cut for selected pages of inFile and generates corresponding poster tiles in outDir.
func PosterFile(inFile, outDir, outFile string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("creating poster pages from %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	if err := Poster(f, outDir, outFile, selectedPages, cut, conf); err != nil {
		return fmt.Errorf("create poster output: %w", err)
	}

	return nil
}

// NDown applies n & cutConf for selected pages of rs and writes results to outDir.
func NDown(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, n int, cut *model.Cut, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if cut == nil {
		return errors.New("missing cut configuration")
	}

	switch n {
	case 2, 3, 4, 6, 8, 9, 12, 16:
	default:
		return fmt.Errorf("invalid n-down value %d: must be one of 2, 3, 4, 6, 8, 9, 12, 16", n)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NDOWN
	fileName = sanitizeFilenamePart(fileName, "ndown")

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: nothing to cut!")
		}
		return nil
	}

	for pageNr, v := range pages {
		if !v {
			continue
		}
		ctxDest, err := pdfcpu.NDownPage(ctxSrc, pageNr, n, cut)
		if err != nil {
			return fmt.Errorf("create n-down page %d: %w", pageNr, err)
		}

		if conf.PostProcessValidate {
			if err = ValidateContext(ctxDest); err != nil {
				return fmt.Errorf("validate output page %d: %w", pageNr, err)
			}
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
		if log.CLIEnabled() {
			log.CLI.Printf("writing %s\n", outFile)
		}
		if err := WriteContextFile(ctxDest, outFile); err != nil {
			return fmt.Errorf("write output file %s: %w", outFile, err)
		}
	}

	return nil
}

// NDownFile applies n & cutConf for selected pages of inFile and writes results to outDir.
func NDownFile(inFile, outDir, outFile string, selectedPages []string, n int, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("ndown %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	if err := NDown(f, outDir, outFile, selectedPages, n, cut, conf); err != nil {
		return fmt.Errorf("create n-down output: %w", err)
	}

	return nil
}

func validateAndNormalizeCut(cut *model.Cut) error {
	sort.Float64s(cut.Hor)

	for _, f := range cut.Hor {
		if f < 0 || f >= 1 {
			return errors.New("invalid cut points: values must be >= 0 and < 1")
		}
	}
	if len(cut.Hor) == 0 || cut.Hor[0] > 0 {
		cut.Hor = append([]float64{0}, cut.Hor...)
	}

	sort.Float64s(cut.Vert)
	for _, f := range cut.Vert {
		if f < 0 || f >= 1 {
			return errors.New("invalid cut points: values must be >= 0 and < 1")
		}
	}
	if len(cut.Vert) == 0 || cut.Vert[0] > 0 {
		cut.Vert = append([]float64{0}, cut.Vert...)
	}

	return nil
}

func writeCutPage(ctxSrc *model.Context, pageNr int, outDir, fileName string, cut *model.Cut, conf *model.Configuration) error {
	ctxDest, err := pdfcpu.CutPage(ctxSrc, pageNr, cut)
	if err != nil {
		return fmt.Errorf("cut page %d: %w", pageNr, err)
	}

	if conf.PostProcessValidate {
		if err = ValidateContext(ctxDest); err != nil {
			return fmt.Errorf("validate output page %d: %w", pageNr, err)
		}
	}

	outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
	logWritingTo(outFile)

	if err := WriteContextFile(ctxDest, outFile); err != nil {
		return fmt.Errorf("write output file %s: %w", outFile, err)
	}

	return nil
}

// Cut applies cutConf for selected pages of rs and writes results to outDir.
func Cut(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if cut == nil {
		return errors.New("missing cut configuration")
	}

	if len(cut.Hor) == 0 && len(cut.Vert) == 0 {
		return errors.New("invalid cut configuration: missing horizontal or vertical cut points")
	}

	if err := validateAndNormalizeCut(cut); err != nil {
		return fmt.Errorf("validate cut configuration: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CUT
	fileName = sanitizeFilenamePart(fileName, "cut")

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: nothing to cut!")
		}
		return nil
	}

	for pageNr, v := range pages {
		if !v {
			continue
		}
		if err := writeCutPage(ctxSrc, pageNr, outDir, fileName, cut, conf); err != nil {
			return err
		}
	}

	return nil
}

// CutFile applies cutConf for selected pages of inFile and writes results to outDir.
func CutFile(inFile, outDir, outFile string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("cutting %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	if err := Cut(f, outDir, outFile, selectedPages, cut, conf); err != nil {
		return fmt.Errorf("cut pages: %w", err)
	}

	return nil
}

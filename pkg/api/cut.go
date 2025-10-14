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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mechiko/pdfcpu/pkg/log"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/model"
	"github.com/mechiko/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func prepareForCut(rs io.ReadSeeker, selectedPages []string, conf *model.Configuration) (*model.Context, types.IntSet, error) {
	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return nil, nil, err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return nil, nil, err
	}

	return ctx, pages, nil
}

// Poster applies cut for selected pages of rs and generates corresponding poster tiles in outDir.
func Poster(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Poster: missing rs")
	}

	if cut.PageSize == "" && !cut.UserDim {
		return errors.New("pdfcpu: poster - please supply either dimensions or form size ")
	}

	if cut.Scale < 1 {
		return errors.Errorf("pdfcpu: invalid scale factor %.2f: i >= 1.0\n", cut.Scale)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.POSTER

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		log.CLI.Println("aborted: nothing to cut!")
		return nil
	}

	for pageNr, v := range pages {
		if !v {
			continue
		}
		ctxDest, err := pdfcpu.PosterPage(ctxSrc, pageNr, cut)
		if err != nil {
			return err
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
		logWritingTo(outFile)

		if conf.PostProcessValidate {
			if err = ValidateContext(ctxDest); err != nil {
				return err
			}
		}

		if err := WriteContextFile(ctxDest, outFile); err != nil {
			return err
		}
	}

	return nil
}

// PosterFile applies cut for selected pages of inFile and generates corresponding poster tiles in outDir.
func PosterFile(inFile, outDir, outFile string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	log.CLI.Printf("ndown %s into %s/ ...\n", inFile, outDir)

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	return Poster(f, outDir, outFile, selectedPages, cut, conf)
}

// NDown applies n & cutConf for selected pages of rs and writes results to outDir.
func NDown(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, n int, cut *model.Cut, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu NDown: Please provide rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NDOWN

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
			return err
		}

		if conf.PostProcessValidate {
			if err = ValidateContext(ctxDest); err != nil {
				return err
			}
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
		if log.CLIEnabled() {
			log.CLI.Printf("writing %s\n", outFile)
		}
		if err := WriteContextFile(ctxDest, outFile); err != nil {
			return err
		}
	}

	return nil
}

// NDownFile applies n & cutConf for selected pages of inFile and writes results to outDir.
func NDownFile(inFile, outDir, outFile string, selectedPages []string, n int, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("ndown %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	return NDown(f, outDir, outFile, selectedPages, n, cut, conf)
}

func validateCut(cut *model.Cut) error {
	sort.Float64s(cut.Hor)

	for _, f := range cut.Hor {
		if f < 0 || f >= 1 {
			return errors.New("pdfcpu: Invalid cut points. Please consult pdfcpu help cut")
		}
	}
	if len(cut.Hor) == 0 || cut.Hor[0] > 0 {
		cut.Hor = append([]float64{0}, cut.Hor...)
	}

	sort.Float64s(cut.Vert)
	for _, f := range cut.Vert {
		if f < 0 || f >= 1 {
			return errors.New("pdfcpu: Invalid cut points. Please consult pdfcpu help cut")
		}
	}
	if len(cut.Vert) == 0 || cut.Vert[0] > 0 {
		cut.Vert = append([]float64{0}, cut.Vert...)
	}

	return nil
}

// Cut applies cutConf for selected pages of rs and writes results to outDir.
func Cut(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: Cut: missing rs")
	}

	if len(cut.Hor) == 0 && len(cut.Vert) == 0 {
		return errors.New("pdfcpu: Invalid cut configuration string: missing hor/ver cutpoints. Please consult pdfcpu help cut")
	}

	if err := validateCut(cut); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CUT

	ctxSrc, pages, err := prepareForCut(rs, selectedPages, conf)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		log.CLI.Println("aborted: nothing to cut!")
		return nil
	}

	for pageNr, v := range pages {
		if !v {
			continue
		}
		ctxDest, err := pdfcpu.CutPage(ctxSrc, pageNr, cut)
		if err != nil {
			return err
		}

		if conf.PostProcessValidate {
			if err = ValidateContext(ctxDest); err != nil {
				return err
			}
		}

		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, pageNr))
		logWritingTo(outFile)

		if err := WriteContextFile(ctxDest, outFile); err != nil {
			return err
		}
	}

	return nil
}

// CutFile applies cutConf for selected pages of inFile and writes results to outDir.
func CutFile(inFile, outDir, outFile string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if log.CLIEnabled() {
		log.CLI.Printf("cutting %s into %s/ ...\n", inFile, outDir)
	}

	if outFile == "" {
		outFile = strings.TrimSuffix(filepath.Base(inFile), ".pdf")
	}

	return Cut(f, outDir, outFile, selectedPages, cut, conf)
}

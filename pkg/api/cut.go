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
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

func Poster(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu poster: Please provide rs")
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.POSTER

	fromStart := time.Now()
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		log.CLI.Println("aborted: nothing to cut!")
		return nil
	}

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	for i, v := range pages {
		if !v {
			continue
		}
		// Cutting page i
		// cut.Scale
		// cut.Hor
		// cut,Vert
		ctxNew, err := pdfcpu.ExtractPage(ctx, i)
		if err != nil {
			return err
		}
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, i))
		log.CLI.Printf("writing %s\n", outFile)
		if err := WriteContextFile(ctxNew, outFile); err != nil {
			return err
		}
	}

	return nil
}

// PosterFile applies cut for selected pages of inFile and generates corresponding poster tiles in outDir.
func PosterFile(inFile, outDir string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()
	log.CLI.Printf("ndown %s into %s/ ...\n", inFile, outDir)
	return Poster(f, outDir, filepath.Base(inFile), selectedPages, cut, conf)
}

func NDown(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu ndown: Please provide rs")
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NDOWN

	fromStart := time.Now()
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		log.CLI.Println("aborted: nothing to cut!")
		return nil
	}

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	for i, v := range pages {
		if !v {
			continue
		}
		// Cutting page i
		// cut.Scale
		// cut.Hor
		// cut,Vert
		ctxNew, err := pdfcpu.ExtractPage(ctx, i)
		if err != nil {
			return err
		}
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, i))
		log.CLI.Printf("writing %s\n", outFile)
		if err := WriteContextFile(ctxNew, outFile); err != nil {
			return err
		}
	}

	return nil
}

// NDownFile applies cutConf for selected pages of inFile and writes results to outDir.
func NDownFile(inFile, outDir string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()
	log.CLI.Printf("ndown %s into %s/ ...\n", inFile, outDir)
	return NDown(f, outDir, filepath.Base(inFile), selectedPages, cut, conf)
}

func Cut(rs io.ReadSeeker, outDir, fileName string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu cut: Please provide rs")
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CUT

	fromStart := time.Now()
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		log.CLI.Println("aborted: nothing to cut!")
		return nil
	}

	fileName = strings.TrimSuffix(filepath.Base(fileName), ".pdf")

	for i, v := range pages {
		if !v {
			continue
		}
		// Cutting page i
		// cut.Scale
		// cut.Hor
		// cut,Vert
		ctxNew, err := pdfcpu.CutPage(ctx, i)
		if err != nil {
			return err
		}
		outFile := filepath.Join(outDir, fmt.Sprintf("%s_page_%d.pdf", fileName, i))
		log.CLI.Printf("writing %s\n", outFile)
		if err := WriteContextFile(ctxNew, outFile); err != nil {
			return err
		}
	}

	return nil
}

// CutFile applies cutConf for selected pages of inFile and writes results to outDir.
func CutFile(inFile, outDir string, selectedPages []string, cut *model.Cut, conf *model.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	defer f.Close()
	log.CLI.Printf("cutting %s into %s/ ...\n", inFile, outDir)
	return Cut(f, outDir, filepath.Base(inFile), selectedPages, cut, conf)
}

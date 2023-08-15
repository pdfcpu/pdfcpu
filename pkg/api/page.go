/*
	Copyright 2020 The pdfcpu Authors.

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
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// InsertPages inserts a blank page before or after every page selected of rs and writes the result to w.
func InsertPages(rs io.ReadSeeker, w io.Writer, selectedPages []string, before bool, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: InsertPages: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.INSERTPAGESAFTER
	if before {
		conf.Cmd = model.INSERTPAGESBEFORE
	}

	fromStart := time.Now()
	ctx, _, _, _, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	if err = ctx.InsertBlankPages(pages, before); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil
}

// InsertPagesFile inserts a blank page before or after every inFile page selected and writes the result to w.
func InsertPagesFile(inFile, outFile string, selectedPages []string, before bool, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return InsertPages(f1, f2, selectedPages, before, conf)
}

// RemovePages removes selected pages from rs and writes the result to w.
func RemovePages(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemovePages: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEPAGES

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false, true)
	if err != nil {
		return err
	}

	// ctx.Pagecount gets set during validation.
	if len(pages) >= ctx.PageCount {
		return errors.New("pdfcpu: operation invalid")
	}

	// No special context processing required.
	// WriteContext decides which pages get written by checking conf.Cmd

	ctx.Write.SelectedPages = pages
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "remove pages, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemovePagesFile removes selected inFile pages and writes the result to outFile..
func RemovePagesFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			err = os.Rename(tmpFile, inFile)
		}
	}()

	return RemovePages(f1, f2, selectedPages, conf)
}

// PageCount returns rs's page count.
func PageCount(rs io.ReadSeeker, conf *model.Configuration) (int, error) {
	if rs == nil {
		return 0, errors.New("pdfcpu: PageCount: missing rs")
	}

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return 0, err
	}

	if err := ValidateContext(ctx); err != nil {
		return 0, err
	}

	return ctx.PageCount, nil
}

// PageCountFile returns inFile's page count.
func PageCountFile(inFile string) (int, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return PageCount(f, model.NewDefaultConfiguration())
}

// PageDims returns a sorted slice of mediaBox dimensions for rs.
func PageDims(rs io.ReadSeeker, conf *model.Configuration) ([]types.Dim, error) {
	if rs == nil {
		return nil, errors.New("pdfcpu: PageDims: missing rs")
	}

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return nil, err
	}

	pd, err := ctx.PageDims()
	if err != nil {
		return nil, err
	}

	if len(pd) != ctx.PageCount {
		return nil, errors.New("pdfcpu: corrupt page dimensions")
	}

	return pd, nil
}

// PageDimsFile returns a sorted slice of mediaBox dimensions for inFile.
func PageDimsFile(inFile string) ([]types.Dim, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return PageDims(f, model.NewDefaultConfiguration())
}

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
	"errors"
	"io"
	"os"
	"sort"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// InsertPages inserts a blank page before or after every page selected of rs and writes the result to w.
func InsertPages(rs io.ReadSeeker, w io.Writer, selectedPages []string, before bool, pageConf *pdfcpu.PageConfiguration, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.INSERTPAGESAFTER
	if before {
		conf.Cmd = model.INSERTPAGESBEFORE
	}

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	var dim *types.Dim
	if pageConf != nil {
		dim = pageConf.PageDim
	}

	if err = ctx.InsertBlankPages(pages, dim, before); err != nil {
		return err
	}

	return Write(ctx, w, conf)
}

// InsertPagesFile inserts a blank page before or after every inFile page selected and writes the result to w.
func InsertPagesFile(inFile, outFile string, selectedPages []string, before bool, pageConf *pdfcpu.PageConfiguration, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
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

	if err = InsertPages(f1, f2, selectedPages, before, pageConf, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemovePages removes selected pages from rs and writes the result to w.
func RemovePages(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEPAGES

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return err
	}

	pages, err := RemainingPagesForPageRemoval(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if len(pages) == 0 {
		if log.CLIEnabled() {
			log.CLI.Println("aborted: missing page numbers!")
		}
		return nil
	}

	var pageNrs []int
	for k, v := range pages {
		if v {
			pageNrs = append(pageNrs, k)
		}
	}
	sort.Ints(pageNrs)

	ctxDest, err := pdfcpu.ExtractPages(ctx, pageNrs, false)
	if err != nil {
		return err
	}

	return Write(ctxDest, w, conf)
}

// RemovePagesFile removes selected inFile pages and writes the result to outFile..
func RemovePagesFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	if f2, tmpFile, err = createOutputFile(inFile, tmpFile); err != nil {
		_ = f1.Close()
		return err
	}

	defer func() {
		if !ok {
			_ = f2.Close()
			_ = f1.Close()
			os.Remove(tmpFile)
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

	if err = RemovePages(f1, f2, selectedPages, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// PageCount returns rs's page count.
func PageCount(rs io.ReadSeeker, conf *model.Configuration) (count int, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return 0, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
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
func PageDims(rs io.ReadSeeker, conf *model.Configuration) (pd []types.Dim, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadAndValidate(rs, conf)
	if err != nil {
		return nil, err
	}

	pd, err = ctx.PageDims()
	if err != nil {
		return nil, err
	}

	if len(pd) != ctx.PageCount {
		return nil, errors.New("corrupt page dimensions")
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

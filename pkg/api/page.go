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
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// validatePageConfiguration accepts nil pageConf or PageDim.
// A nil PageDim uses each selected page's effective MediaBox for its inserted blank page.
func validatePageConfiguration(pageConf *pdfcpu.PageConfiguration) error {
	if pageConf == nil || pageConf.PageDim == nil {
		return nil
	}
	if invalidPageDimension(pageConf.PageDim.Width) {
		return fmt.Errorf("width must be positive and finite: %w", ErrInvalidPageConfiguration)
	}
	if invalidPageDimension(pageConf.PageDim.Height) {
		return fmt.Errorf("height must be positive and finite: %w", ErrInvalidPageConfiguration)
	}
	return nil
}

func invalidPageDimension(v float64) bool {
	return v <= 0 || math.IsNaN(v) || math.IsInf(v, 0)
}

// InsertPages inserts a blank page before or after every selected page, writes the result to w and supports cancellation.
func InsertPages(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, before bool, pageConf *pdfcpu.PageConfiguration, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validatePageConfiguration(pageConf); err != nil {
		return fmt.Errorf("insert pages: validate page configuration: %w", err)
	}

	cmd := model.INSERTPAGESAFTER
	if before {
		cmd = model.INSERTPAGESBEFORE
	}
	conf = operationConfiguration(conf, cmd)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("insert pages: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("insert pages: parse page selection: %w", err)
	}

	var dim *types.Dim
	if pageConf != nil {
		dim = pageConf.PageDim
	}

	if err = ctx.InsertBlankPages(c, pages, dim, before); err != nil {
		return fmt.Errorf("insert pages: insert blank pages: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("insert pages: write output: %w", err)
	}
	return nil
}

// InsertPagesFile inserts a blank page before or after every selected page, writes the result to outFile and supports cancellation.
func InsertPagesFile(c context.Context, inFile, outFile string, selectedPages []string, before bool, pageConf *pdfcpu.PageConfiguration, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validatePageConfiguration(pageConf); err != nil {
		return fmt.Errorf("insert pages: validate page configuration: %w", err)
	}
	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("insert pages: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "insert pages")
	if err != nil {
		return errors.Join(
			fmt.Errorf("insert pages: create output: %w", err),
			closeFile(f1, "insert pages: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = InsertPages(c, f1, f2, selectedPages, before, pageConf, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemovePages removes selected pages from rs, writes the result to w and supports cancellation.
func RemovePages(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}

	conf = operationConfiguration(conf, model.REMOVEPAGES)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove pages: %w", err)
	}

	pages, err := RemainingPagesForRemoval(ctx.PageCount, selectedPages)
	if err != nil {
		return fmt.Errorf("remove pages: parse page selection: %w", err)
	}

	var pageNrs []int
	for k, v := range pages {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if v {
			pageNrs = append(pageNrs, k)
		}
	}
	sort.Ints(pageNrs)
	if len(pageNrs) == 0 {
		return fmt.Errorf("remove pages: no pages remaining: %w", pdfcpu.ErrMissingPageNumbers)
	}

	ctxDest, err := pdfcpu.ExtractPages(c, ctx, pageNrs, false)
	if err != nil {
		return fmt.Errorf("remove pages: extract remaining pages: %w", err)
	}

	if err = Write(c, ctxDest, w, conf); err != nil {
		return fmt.Errorf("remove pages: write output: %w", err)
	}
	return nil
}

// RemovePagesFile removes selected pages from inFile, writes the result to outFile and supports cancellation.
func RemovePagesFile(c context.Context, inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("remove pages: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove pages")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove pages: create output: %w", err),
			closeFile(f1, "remove pages: close input"),
		)
	}
	f2 = staged.output.file

	defer func() {
		if !ok {
			err = staged.cleanup(err)
			return
		}
		err = staged.commit()
	}()

	if err = RemovePages(c, f1, f2, selectedPages, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// PageCount returns rs's page count and supports cancellation.
func PageCount(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (count int, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	if rs == nil {
		return 0, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return 0, fmt.Errorf("page count: %w", err)
	}

	return ctx.PageCount, nil
}

// PageCountFile returns inFile's page count and supports cancellation.
func PageCountFile(c context.Context, inFile string) (count int, err error) {
	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	if inFile == "" {
		return 0, ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return 0, fmt.Errorf("page count: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "page count: close input"))
	}()

	return PageCount(c, f, model.NewDefaultConfiguration())
}

// PageDims returns media box dimensions for rs in ascending page order and supports cancellation.
func PageDims(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (pd []types.Dim, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadAndValidate(c, rs, conf)
	if err != nil {
		return nil, fmt.Errorf("page dimensions: %w", err)
	}

	pd, err = ctx.PageDims(c)
	if err != nil {
		return nil, fmt.Errorf("page dimensions: collect page dimensions: %w", err)
	}

	if len(pd) != ctx.PageCount {
		return nil, fmt.Errorf("page dimensions: corrupt result: got %d dimensions for %d pages", len(pd), ctx.PageCount)
	}

	return pd, nil
}

// PageDimsFile returns media box dimensions for inFile in ascending page order and supports cancellation.
func PageDimsFile(c context.Context, inFile string) (pd []types.Dim, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if inFile == "" {
		return nil, ErrMissingPDFInput
	}
	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("page dimensions: open input %s: %w", inFile, err)
	}
	defer func() {
		err = errors.Join(err, closeFile(f, "page dimensions: close input"))
	}()

	return PageDims(c, f, model.NewDefaultConfiguration())
}

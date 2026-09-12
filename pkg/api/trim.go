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
	"os"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func selectedPageNumbers(c context.Context, pageCount int, pages map[int]bool) ([]int, error) {
	pageNrs := make([]int, 0, len(pages))
	for pageNr := 1; pageNr <= pageCount; pageNr++ {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		if pages[pageNr] {
			pageNrs = append(pageNrs, pageNr)
		}
	}
	return pageNrs, nil
}

// Trim generates a trimmed version of rs, writes the result to w and supports cancellation.
func Trim(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.TRIM)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("trim: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, false)
	if err != nil {
		return fmt.Errorf("trim: parse page selection: %w", err)
	}

	pageNrs, err := selectedPageNumbers(c, ctx.PageCount, pages)
	if err != nil {
		return err
	}
	if len(pageNrs) == 0 {
		return nil
	}

	ctxDest, err := pdfcpu.ExtractPages(c, ctx, pageNrs, false)
	if err != nil {
		return fmt.Errorf("trim: extract pages: %w", err)
	}

	if conf.PostProcessValidate {
		if err = ValidateContext(c, ctxDest); err != nil {
			return fmt.Errorf("trim: validate output: %w", err)
		}
	}

	if err = WriteContext(c, ctxDest, w); err != nil {
		return fmt.Errorf("trim: write output: %w", err)
	}
	return nil
}

// TrimFile generates a trimmed version of inFile, writes the result to outFile and supports cancellation.
func TrimFile(c context.Context, inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("trim: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "trim")
	if err != nil {
		return errors.Join(
			fmt.Errorf("trim: create output: %w", err),
			closeFile(f1, "trim: close input"),
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

	if err = Trim(c, f1, f2, selectedPages, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

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

// Collect creates a custom PDF page sequence for selected pages of rs, writes it to w and supports cancellation.
func Collect(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.COLLECT)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}

	pages, err := PagesForPageCollection(ctx.PageCount, selectedPages)
	if err != nil {
		return fmt.Errorf("collect: parse page selection: %w", err)
	}

	if err := contextutil.Check(c); err != nil {
		return err
	}

	ctxDest, err := pdfcpu.ExtractPages(c, ctx, pages, false)
	if err != nil {
		return fmt.Errorf("collect: extract pages: %w", err)
	}

	if err = Write(c, ctxDest, w, conf); err != nil {
		return fmt.Errorf("collect: write output: %w", err)
	}
	return nil
}

// CollectFile creates a custom PDF page sequence for inFile, writes the result to outFile and supports cancellation.
func CollectFile(c context.Context, inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("collect: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "collect")
	if err != nil {
		return errors.Join(
			fmt.Errorf("collect: create output: %w", err),
			closeFile(f1, "collect: close input"),
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

	if err = Collect(c, f1, f2, selectedPages, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

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

func validateRotation(rotation int) error {
	if rotation%90 != 0 {
		return fmt.Errorf("rotation must be a multiple of 90: %w", ErrInvalidRotation)
	}
	return nil
}

// Rotate rotates selected pages of rs clockwise, writes the result to w and supports cancellation.
func Rotate(c context.Context, rs io.ReadSeeker, w io.Writer, rotation int, selectedPages []string, conf *model.Configuration) (err error) {
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
	if err := validateRotation(rotation); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ROTATE)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("rotate: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("rotate: parse page selection: %w", err)
	}

	if err := contextutil.Check(c); err != nil {
		return err
	}

	if err = pdfcpu.RotatePages(c, ctx, pages, rotation); err != nil {
		return fmt.Errorf("rotate: apply rotation: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("rotate: write output: %w", err)
	}
	return nil
}

// RotateFile rotates selected pages of inFile clockwise, writes the result to outFile and supports cancellation.
func RotateFile(c context.Context, inFile, outFile string, rotation int, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateRotation(rotation); err != nil {
		return err
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("rotate: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "rotate")
	if err != nil {
		return errors.Join(
			fmt.Errorf("rotate: create output: %w", err),
			closeFile(f1, "rotate: close input"),
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

	if err = Rotate(c, f1, f2, rotation, selectedPages, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

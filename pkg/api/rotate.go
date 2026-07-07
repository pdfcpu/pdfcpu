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
	"fmt"
	"io"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ErrInvalidRotation signals a rotation that is not a multiple of 90 degrees.
var ErrInvalidRotation = errors.New("invalid rotation")

func validateRotation(rotation int) error {
	if rotation%90 != 0 {
		return fmt.Errorf("rotation must be a multiple of 90: %w", ErrInvalidRotation)
	}
	return nil
}

// Rotate rotates selected pages of rs clockwise by rotation degrees and writes the result to w.
func Rotate(rs io.ReadSeeker, w io.Writer, rotation int, selectedPages []string, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validateRotation(rotation); err != nil {
		return err
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ROTATE

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("rotate: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("rotate: parse page selection: %w", err)
	}

	if err = pdfcpu.RotatePages(ctx, pages, rotation); err != nil {
		return fmt.Errorf("rotate: apply rotation: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("rotate: write output: %w", err)
	}
	return nil
}

// RotateFile rotates selected pages of inFile clockwise by rotation degrees and writes the result to outFile.
func RotateFile(inFile, outFile string, rotation int, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

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
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
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

	if err = Rotate(f1, f2, rotation, selectedPages, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

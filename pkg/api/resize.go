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
	"math"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func validateResizeConfiguration(res *model.Resize) error {
	if res == nil {
		return ErrMissingResizeConfiguration
	}
	if res.Scale != 0 {
		if invalidResizeScale(res) {
			return fmt.Errorf("scale factor: %w", ErrInvalidResizeConfiguration)
		}
		return nil
	}
	if res.PageDim == nil {
		return fmt.Errorf("missing scale factor or dimensions: %w", ErrInvalidResizeConfiguration)
	}
	w, h := res.PageDim.Width, res.PageDim.Height
	if invalidResizeDimension(w) || invalidResizeDimension(h) || w == 0 && h == 0 {
		return fmt.Errorf("dimensions: %w", ErrInvalidResizeConfiguration)
	}
	return nil
}

func invalidResizeScale(res *model.Resize) bool {
	return res.PageDim != nil || invalidResizeDimension(res.Scale) || res.Scale == 1
}

func invalidResizeDimension(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0) || v < 0
}

// Resize applies resizeConf for selected pages of rs and writes result to w.
func Resize(rs io.ReadSeeker, w io.Writer, selectedPages []string, resize *model.Resize, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validateResizeConfiguration(resize); err != nil {
		return fmt.Errorf("resize: validate configuration: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESIZE

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("resize: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("resize: parse page selection: %w", err)
	}

	if err = pdfcpu.Resize(ctx, pages, resize); err != nil {
		return fmt.Errorf("resize: apply pages: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("resize: write output: %w", err)
	}
	return nil
}

// ResizeFile applies resizeConf for selected pages of inFile and writes result to outFile.
func ResizeFile(inFile, outFile string, selectedPages []string, resize *model.Resize, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateResizeConfiguration(resize); err != nil {
		return fmt.Errorf("resize: validate configuration: %w", err)
	}

	if log.CLIEnabled() {
		log.CLI.Printf("resizing %s\n", inFile)
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("resize: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "resize")
	if err != nil {
		return errors.Join(
			fmt.Errorf("resize: create output: %w", err),
			closeFile(f1, "resize: close input"),
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

	if err = Resize(f1, f2, selectedPages, resize, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

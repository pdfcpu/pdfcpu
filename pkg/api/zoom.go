/*
Copyright 2024 The pdfcpu Authors.

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

func validateZoomConfiguration(zoom *model.Zoom) error {
	if zoom == nil {
		return ErrMissingZoomConfiguration
	}
	if !finiteZoomValue(zoom.Factor) || !finiteZoomValue(zoom.HMargin) || !finiteZoomValue(zoom.VMargin) {
		return fmt.Errorf("non-finite factor or margin: %w", ErrInvalidZoomConfiguration)
	}

	n := 0
	if zoom.Factor != 0 {
		n++
	}
	if zoom.HMargin != 0 {
		n++
	}
	if zoom.VMargin != 0 {
		n++
	}
	if n != 1 {
		return fmt.Errorf("supply exactly one factor or margin: %w", ErrInvalidZoomConfiguration)
	}
	if zoom.Factor < 0 || zoom.Factor == 1 {
		return fmt.Errorf("factor: %w", ErrInvalidZoomConfiguration)
	}
	return nil
}

func finiteZoomValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// Zoom applies zoom for selected pages of rs and writes the result to w.
func Zoom(rs io.ReadSeeker, w io.Writer, selectedPages []string, zoom *model.Zoom, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}
	if w == nil {
		return ErrMissingPDFWriter
	}
	if err := validateZoomConfiguration(zoom); err != nil {
		return fmt.Errorf("zoom: validate configuration: %w", err)
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ZOOM

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("zoom: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("zoom: parse page selection: %w", err)
	}

	if err = pdfcpu.Zoom(ctx, pages, zoom); err != nil {
		return fmt.Errorf("zoom: apply pages: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("zoom: write output: %w", err)
	}
	return nil
}

// ZoomFile applies zoom for selected pages of inFile and writes the result to outFile.
func ZoomFile(inFile, outFile string, selectedPages []string, zoom *model.Zoom, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}
	if err := validateZoomConfiguration(zoom); err != nil {
		return fmt.Errorf("zoom: validate configuration: %w", err)
	}
	if log.CLIEnabled() {
		log.CLI.Printf("zooming %s\n", inFile)
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("zoom: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}

	staged, err := openStagedOutput(f1, inFile, tmpFile, "zoom")
	if err != nil {
		return errors.Join(
			fmt.Errorf("zoom: create output: %w", err),
			closeFile(f1, "zoom: close input"),
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

	if err = Zoom(f1, f2, selectedPages, zoom, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

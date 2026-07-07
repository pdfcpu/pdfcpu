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
	"sort"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// WatermarkContext applies wm for selected pages to ctx.
func WatermarkContext(ctx *model.Context, selectedPages types.IntSet, wm *model.Watermark) error {
	if ctx == nil {
		return ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}

	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}

	if err := pdfcpu.AddWatermarks(ctx, selectedPages, wm); err != nil {
		return fmt.Errorf("%s: apply: %w", watermarkOperation(wm), err)
	}
	return nil
}

func watermarkOperation(wm *model.Watermark) string {
	if wm != nil && wm.Update {
		return "update watermarks"
	}
	return "add watermarks"
}

// AddWatermarksMap adds watermarks in m to corresponding pages in rs and writes the result to w.
func AddWatermarksMap(rs io.ReadSeeker, w io.Writer, m map[int]*model.Watermark, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.ADDWATERMARKS

	if err := validateWatermarkMap(m); err != nil {
		return err
	}

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("add watermarks: %w", err)
	}

	if err = pdfcpu.AddWatermarksMap(ctx, m); err != nil {
		return fmt.Errorf("add watermarks: apply: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("add watermarks: write output: %w", err)
	}
	return nil
}

func sortedWatermarkMapPages[T any](m map[int]T) []int {
	pageNrs := make([]int, 0, len(m))
	for pageNr := range m {
		pageNrs = append(pageNrs, pageNr)
	}
	sort.Ints(pageNrs)
	return pageNrs
}

func validateWatermarkMap(m map[int]*model.Watermark) error {
	if len(m) == 0 {
		return ErrMissingWatermarks
	}

	for _, pageNr := range sortedWatermarkMapPages(m) {
		wm := m[pageNr]
		if wm == nil {
			return fmt.Errorf("page %d: %w", pageNr, ErrMissingWatermarkConfiguration)
		}
	}

	return nil
}

func validateWatermarkSliceMap(m map[int][]*model.Watermark) error {
	if len(m) == 0 {
		return ErrMissingWatermarks
	}

	for _, pageNr := range sortedWatermarkMapPages(m) {
		wms := m[pageNr]
		if len(wms) == 0 {
			return fmt.Errorf("page %d: %w", pageNr, ErrMissingWatermarks)
		}
		for i, wm := range wms {
			if wm == nil {
				return fmt.Errorf("page %d, watermark %d: %w", pageNr, i, ErrMissingWatermarkConfiguration)
			}
		}
	}

	return nil
}

// AddWatermarksMapFile adds watermarks to corresponding pages in m of inFile and writes the result to outFile.
func AddWatermarksMapFile(inFile, outFile string, m map[int]*model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add watermarks: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "add watermarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add watermarks: create output: %w", err),
			closeFile(f1, "add watermarks: close input"),
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

	if err = AddWatermarksMap(f1, f2, m, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages in rs and writes the result to w.
func AddWatermarksSliceMap(rs io.ReadSeeker, w io.Writer, m map[int][]*model.Watermark, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.ADDWATERMARKS

	if err := validateWatermarkSliceMap(m); err != nil {
		return err
	}

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("add watermarks: %w", err)
	}

	if err = pdfcpu.AddWatermarksSliceMap(ctx, m); err != nil {
		return fmt.Errorf("add watermarks: apply: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("add watermarks: write output: %w", err)
	}
	return nil
}

// AddWatermarksSliceMapFile adds watermarks to corresponding pages in m of inFile and writes the result to outFile.
func AddWatermarksSliceMapFile(inFile, outFile string, m map[int][]*model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add watermarks: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "add watermarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("add watermarks: create output: %w", err),
			closeFile(f1, "add watermarks: close input"),
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

	if err = AddWatermarksSliceMap(f1, f2, m, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddWatermarks adds watermarks to all pages selected in rs and writes the result to w.
func AddWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, wm *model.Watermark, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.ADDWATERMARKS
	conf.OptimizeDuplicateContentStreams = false

	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}
	operation := watermarkOperation(wm)

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("%s: parse page selection: %w", operation, err)
	}

	if err = pdfcpu.AddWatermarks(ctx, pages, wm); err != nil {
		return fmt.Errorf("%s: apply: %w", operation, err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("%s: write output: %w", operation, err)
	}
	return nil
}

// AddWatermarksFile adds watermarks to all selected pages of inFile and writes the result to outFile.
func AddWatermarksFile(inFile, outFile string, selectedPages []string, wm *model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false
	operation := watermarkOperation(wm)

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("%s: open input %s: %w", operation, inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, operation)
	if err != nil {
		return errors.Join(
			fmt.Errorf("%s: create output: %w", operation, err),
			closeFile(f1, operation+": close input"),
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

	if err = AddWatermarks(f1, f2, selectedPages, wm, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemoveWatermarks removes watermarks from all pages selected in rs and writes the result to w.
func RemoveWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
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
	conf.Cmd = model.REMOVEWATERMARKS

	ctx, err := ReadValidateAndOptimize(rs, conf)
	if err != nil {
		return fmt.Errorf("remove watermarks: %w", err)
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return fmt.Errorf("remove watermarks: parse page selection: %w", err)
	}

	if err = pdfcpu.RemoveWatermarks(ctx, pages); err != nil {
		return fmt.Errorf("remove watermarks: apply: %w", err)
	}

	if err = Write(ctx, w, conf); err != nil {
		return fmt.Errorf("remove watermarks: write output: %w", err)
	}
	return nil
}

// RemoveWatermarksFile removes watermarks from all selected pages of inFile and writes the result to outFile.
func RemoveWatermarksFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("remove watermarks: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		logWritingTo(outFile)
	} else {
		logWritingTo(inFile)
	}
	staged, err := openStagedOutput(f1, inFile, tmpFile, "remove watermarks")
	if err != nil {
		return errors.Join(
			fmt.Errorf("remove watermarks: create output: %w", err),
			closeFile(f1, "remove watermarks: close input"),
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

	if err = RemoveWatermarks(f1, f2, selectedPages, conf); err != nil {
		return err
	}

	ok = true

	return nil
}

// HasWatermarks checks rs for watermarks.
func HasWatermarks(rs io.ReadSeeker, conf *model.Configuration) (ok bool, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return false, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return false, fmt.Errorf("detect watermarks: prepare PDF context: %w", err)
	}

	if err := pdfcpu.DetectWatermarks(ctx); err != nil {
		return false, fmt.Errorf("detect watermarks: inspect PDF: %w", err)
	}

	return ctx.Watermarked, nil
}

// HasWatermarksFile checks inFile for watermarks.
func HasWatermarksFile(inFile string, conf *model.Configuration) (ok bool, err error) {
	if inFile == "" {
		return false, ErrMissingPDFInput
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	f, err := os.Open(inFile)
	if err != nil {
		return false, fmt.Errorf("detect watermarks: open input %s: %w", inFile, err)
	}

	defer func() {
		err = errors.Join(err, closeFile(f, "detect watermarks: close input"))
	}()

	return HasWatermarks(f, conf)
}

// ImageWatermarkForReader returns an image watermark configuration for r.
func ImageWatermarkForReader(rd io.Reader, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if rd == nil {
		return nil, ErrMissingImageReader
	}

	wm, err := pdfcpu.ParseImageWatermarkDetails("", desc, onTop, u)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}

	wm.Update = update
	wm.Image = rd

	return wm, nil
}

// PDFWatermarkForReadSeeker returns a PDF watermark configuration.
// Apply watermark/stamp to destination file with pageNrSrc of rs for selected pages.
// If pageNr == 0 apply a multi watermark/stamp applying all src pages in ascending manner to destination pages.
func PDFWatermarkForReadSeeker(rs io.ReadSeeker, pageNrSrc int, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	wm, err := pdfcpu.ParsePDFWatermarkDetails("", desc, onTop, u)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}

	wm.Update = update
	wm.PDF = rs
	wm.PdfPageNrSrc = pageNrSrc

	return wm, nil
}

// PDFMultiWatermarkForReadSeeker returns a PDF watermark configuration.
// Define a source PDF watermark/stamp sequence using rs from page startPageNrSrc thru the last page of rs.
// Apply this sequence to the destination PDF file starting at page startPageNrDest for selected pages.
func PDFMultiWatermarkForReadSeeker(rs io.ReadSeeker, startPageNrSrc, startPageNrDest int, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	wm, err := pdfcpu.ParsePDFWatermarkDetails("", desc, onTop, u)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}

	wm.Update = update
	wm.PDF = rs
	wm.PdfMultiStartPageNrSrc = startPageNrSrc
	wm.PdfMultiStartPageNrDest = startPageNrDest

	return wm, nil
}

func parseWatermark(mode int, modeParm, desc string, onTop bool, u types.DisplayUnit) (*model.Watermark, error) {
	switch mode {
	case model.WMText:
		return pdfcpu.ParseTextWatermarkDetails(modeParm, desc, onTop, u)
	case model.WMImage:
		return pdfcpu.ParseImageWatermarkDetails(modeParm, desc, onTop, u)
	case model.WMPDF:
		return pdfcpu.ParsePDFWatermarkDetails(modeParm, desc, onTop, u)
	}
	return nil, fmt.Errorf("unsupported watermark mode: %d", mode)
}

func watermark(mode int, modeParm, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if err := pdfcpu.ValidateWatermarkModeParam(mode, modeParm, onTop); err != nil {
		return nil, fmt.Errorf("create watermark: validate configuration: %w", err)
	}
	wm, err := parseWatermark(mode, modeParm, desc, onTop, u)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}
	wm.Update = update
	return wm, nil
}

// TextWatermark returns a text watermark configuration.
func TextWatermark(text, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	return watermark(model.WMText, text, desc, onTop, update, u)
}

// ImageWatermark returns an image watermark configuration.
func ImageWatermark(fileName, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	return watermark(model.WMImage, fileName, desc, onTop, update, u)
}

// PDFWatermark returns a PDF watermark configuration.
func PDFWatermark(fileName, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	return watermark(model.WMPDF, fileName, desc, onTop, update, u)
}

// AddTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := TextWatermark(text, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := ImageWatermark(fileName, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksForReaderFile adds image stamps/watermarks to all selected pages of inFile for r and writes the result to outFile.
func AddImageWatermarksForReaderFile(inFile, outFile string, selectedPages []string, onTop bool, r io.Reader, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := ImageWatermarkForReader(r, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddPDFWatermarksFile adds PDF stamps/watermarks to inFile and writes the result to outFile.
func AddPDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermark(fileName, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddPDFWatermarksForReadSeekerFile adds PDF stamps/watermarks to inFile for rs and writes the result to outFile.
func AddPDFWatermarksForReadSeekerFile(inFile, outFile string, selectedPages []string, onTop bool, rs io.ReadSeeker, pageNrSrc int, desc string, conf *model.Configuration) error {
	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermarkForReadSeeker(rs, pageNrSrc, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdateTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdateTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := TextWatermark(text, desc, onTop, true, unit)
	if err != nil {
		return fmt.Errorf("update watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdateImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdateImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}
	wm, err := ImageWatermark(fileName, desc, onTop, true, unit)
	if err != nil {
		return fmt.Errorf("update watermarks: configure: %w", err)
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdatePDFWatermarksFile adds PDF stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdatePDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermark(fileName, desc, onTop, true, unit)
	if err != nil {
		return fmt.Errorf("update watermarks: configure: %w", err)
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

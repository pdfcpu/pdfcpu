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
	"sort"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// WatermarkContext applies wm for selected pages to ctx and supports cancellation.
// On failure, ctx may contain partial changes and callers must discard it.
func WatermarkContext(c context.Context, ctx *model.Context, selectedPages types.IntSet, wm *model.Watermark) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if ctx == nil {
		return ErrMissingPDFContext
	}
	if ctx.XRefTable == nil {
		return ErrMissingXRefTable
	}

	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}

	if err := pdfcpu.AddWatermarks(c, ctx, selectedPages, wm); err != nil {
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

// AddWatermarksMap adds watermarks in m to corresponding pages in rs,
// writes the result to w and supports cancellation.
func AddWatermarksMap(c context.Context, rs io.ReadSeeker, w io.Writer, m map[int]*model.Watermark, conf *model.Configuration) (err error) {
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

	if err := validateWatermarkMap(m); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ADDWATERMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add watermarks: %w", err)
	}

	if err = pdfcpu.AddWatermarksMap(c, ctx, m); err != nil {
		return fmt.Errorf("add watermarks: apply: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
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

// AddWatermarksMapFile adds watermarks to corresponding pages in m of inFile,
// writes the result to outFile and supports cancellation.
func AddWatermarksMapFile(c context.Context, inFile, outFile string, m map[int]*model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add watermarks: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
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

	if err = AddWatermarksMap(c, f1, f2, m, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages in rs,
// writes the result to w and supports cancellation.
func AddWatermarksSliceMap(c context.Context, rs io.ReadSeeker, w io.Writer, m map[int][]*model.Watermark, conf *model.Configuration) (err error) {
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

	if err := validateWatermarkSliceMap(m); err != nil {
		return err
	}

	conf = operationConfiguration(conf, model.ADDWATERMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("add watermarks: %w", err)
	}

	if err = pdfcpu.AddWatermarksSliceMap(c, ctx, m); err != nil {
		return fmt.Errorf("add watermarks: apply: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("add watermarks: write output: %w", err)
	}
	return nil
}

// AddWatermarksSliceMapFile adds watermarks to corresponding pages in m of inFile,
// writes the result to outFile and supports cancellation.
func AddWatermarksSliceMapFile(c context.Context, inFile, outFile string, m map[int][]*model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("add watermarks: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
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

	if err = AddWatermarksSliceMap(c, f1, f2, m, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// AddWatermarks adds watermarks to all selected pages in rs,
// writes the result to w and supports cancellation.
func AddWatermarks(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, wm *model.Watermark, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.ADDWATERMARKS)
	conf.OptimizeDuplicateContentStreams = false

	if wm == nil {
		return ErrMissingWatermarkConfiguration
	}
	operation := watermarkOperation(wm)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("%s: parse page selection: %w", operation, err)
	}
	if err := contextutil.Check(c); err != nil {
		return err
	}

	if err = pdfcpu.AddWatermarks(c, ctx, pages, wm); err != nil {
		return fmt.Errorf("%s: apply: %w", operation, err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("%s: write output: %w", operation, err)
	}
	return nil
}

// AddWatermarksFile adds watermarks to all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func AddWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, wm *model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false
	operation := watermarkOperation(wm)

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("%s: open input %s: %w", operation, inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
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

	if err = AddWatermarks(c, f1, f2, selectedPages, wm, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// RemoveWatermarks removes watermarks from all selected pages in rs,
// writes the result to w and supports cancellation.
func RemoveWatermarks(c context.Context, rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) (err error) {
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

	conf = operationConfiguration(conf, model.REMOVEWATERMARKS)

	ctx, err := ReadValidateAndOptimize(c, rs, conf, nil)
	if err != nil {
		return fmt.Errorf("remove watermarks: %w", err)
	}

	pages, err := PagesForSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return fmt.Errorf("remove watermarks: parse page selection: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return err
	}

	if err = pdfcpu.RemoveWatermarks(c, ctx, pages); err != nil {
		return fmt.Errorf("remove watermarks: apply: %w", err)
	}

	if err = Write(c, ctx, w, conf); err != nil {
		return fmt.Errorf("remove watermarks: write output: %w", err)
	}
	return nil
}

// RemoveWatermarksFile removes watermarks from all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func RemoveWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File
	ok := false

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	if f1, err = os.Open(inFile); err != nil {
		return fmt.Errorf("remove watermarks: open input %s: %w", inFile, err)
	}

	tmpFile := ""
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
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

	if err = RemoveWatermarks(c, f1, f2, selectedPages, conf); err != nil {
		return err
	}
	if err = contextutil.Check(c); err != nil {
		return err
	}

	ok = true

	return nil
}

// HasWatermarks checks rs for watermarks and supports cancellation.
func HasWatermarks(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (ok bool, err error) {
	defer fault.Catch(&err)

	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if rs == nil {
		return false, ErrMissingPDFReadSeeker
	}

	ctx, err := ReadContext(c, rs, conf)
	if err != nil {
		return false, fmt.Errorf("detect watermarks: prepare PDF context: %w", err)
	}

	if err := pdfcpu.DetectWatermarks(c, ctx); err != nil {
		return false, fmt.Errorf("detect watermarks: inspect PDF: %w", err)
	}

	return ctx.Watermarked, nil
}

// HasWatermarksFile checks inFile for watermarks and supports cancellation.
func HasWatermarksFile(c context.Context, inFile string, conf *model.Configuration) (ok bool, err error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
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

	return HasWatermarks(c, f, conf)
}

// ImageWatermarkForReader returns an image watermark configuration for r and supports cancellation.
func ImageWatermarkForReader(c context.Context, rd io.Reader, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rd == nil {
		return nil, ErrMissingImageReader
	}

	wm, err := pdfcpu.ParseImageWatermarkDetails(c, "", desc, onTop, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}

	wm.Update = update
	wm.Image = rd

	return wm, nil
}

// PDFWatermarkForReadSeeker returns a PDF watermark configuration and supports cancellation.
// Apply watermark/stamp to destination file with pageNrSrc of rs for selected pages.
// If pageNr == 0 apply a multi watermark/stamp applying all src pages in ascending manner to destination pages.
func PDFWatermarkForReadSeeker(c context.Context, rs io.ReadSeeker, pageNrSrc int, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	wm, err := pdfcpu.ParsePDFWatermarkDetails(c, "", desc, onTop, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}

	wm.Update = update
	wm.PDF = rs
	wm.PdfPageNrSrc = pageNrSrc

	return wm, nil
}

// PDFMultiWatermarkForReadSeeker returns a PDF watermark configuration and supports cancellation.
// Define a source PDF watermark/stamp sequence using rs from page startPageNrSrc thru the last page of rs.
// Apply this sequence to the destination PDF file starting at page startPageNrDest for selected pages.
func PDFMultiWatermarkForReadSeeker(c context.Context, rs io.ReadSeeker, startPageNrSrc, startPageNrDest int, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	wm, err := pdfcpu.ParsePDFWatermarkDetails(c, "", desc, onTop, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}

	wm.Update = update
	wm.PDF = rs
	wm.PdfMultiStartPageNrSrc = startPageNrSrc
	wm.PdfMultiStartPageNrDest = startPageNrDest

	return wm, nil
}

func parseWatermark(c context.Context, mode int, modeParm, desc string, onTop bool, u types.DisplayUnit,
	conf *model.Configuration) (*model.Watermark, error) {
	switch mode {
	case model.WMText:
		return pdfcpu.ParseTextWatermarkDetails(c, modeParm, desc, onTop, u, conf)
	case model.WMImage:
		return pdfcpu.ParseImageWatermarkDetails(c, modeParm, desc, onTop, u, conf)
	case model.WMPDF:
		return pdfcpu.ParsePDFWatermarkDetails(c, modeParm, desc, onTop, u, conf)
	}
	return nil, fmt.Errorf("unsupported watermark mode: %d", mode)
}

func watermark(c context.Context, mode int, modeParm, desc string, onTop, update bool, u types.DisplayUnit,
	conf *model.Configuration) (*model.Watermark, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if err := pdfcpu.ValidateWatermarkModeParam(mode, modeParm, onTop); err != nil {
		return nil, fmt.Errorf("create watermark: validate configuration: %w", err)
	}
	wm, err := parseWatermark(c, mode, modeParm, desc, onTop, u, conf)
	if err != nil {
		return nil, fmt.Errorf("create watermark: parse configuration: %w", err)
	}
	wm.Update = update
	return wm, nil
}

// TextWatermark returns a text watermark configuration and supports cancellation.
func TextWatermark(c context.Context, text, desc string, onTop, update bool, u types.DisplayUnit, conf *model.Configuration) (*model.Watermark, error) {
	return watermark(c, model.WMText, text, desc, onTop, update, u, conf)
}

// ImageWatermark returns an image watermark configuration and supports cancellation.
func ImageWatermark(c context.Context, fileName, desc string, onTop, update bool, u types.DisplayUnit, conf *model.Configuration) (*model.Watermark, error) {
	return watermark(c, model.WMImage, fileName, desc, onTop, update, u, conf)
}

// PDFWatermark returns a PDF watermark configuration and supports cancellation.
func PDFWatermark(c context.Context, fileName, desc string, onTop, update bool, u types.DisplayUnit, conf *model.Configuration) (*model.Watermark, error) {
	return watermark(c, model.WMPDF, fileName, desc, onTop, update, u, conf)
}

// AddTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func AddTextWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := TextWatermark(c, text, desc, onTop, false, unit, conf)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func AddImageWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := ImageWatermark(c, fileName, desc, onTop, false, unit, conf)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksForReaderFile adds image stamps/watermarks to all selected pages of inFile for r,
// writes the result to outFile and supports cancellation.
func AddImageWatermarksForReaderFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, r io.Reader, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := ImageWatermarkForReader(c, r, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// AddPDFWatermarksFile adds PDF stamps/watermarks to inFile,
// writes the result to outFile and supports cancellation.
func AddPDFWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermark(c, fileName, desc, onTop, false, unit, conf)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// AddPDFWatermarksForReadSeekerFile adds PDF stamps/watermarks to inFile for rs,
// writes the result to outFile and supports cancellation.
func AddPDFWatermarksForReadSeekerFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, rs io.ReadSeeker, pageNrSrc int, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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

	wm, err := PDFWatermarkForReadSeeker(c, rs, pageNrSrc, desc, onTop, false, unit)
	if err != nil {
		return fmt.Errorf("add watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// UpdateTextWatermarksFile updates text stamps/watermarks for all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func UpdateTextWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := TextWatermark(c, text, desc, onTop, true, unit, conf)
	if err != nil {
		return fmt.Errorf("update watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// UpdateImageWatermarksFile updates image stamps/watermarks for all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func UpdateImageWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}
	wm, err := ImageWatermark(c, fileName, desc, onTop, true, unit, conf)
	if err != nil {
		return fmt.Errorf("update watermarks: configure: %w", err)
	}
	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

// UpdatePDFWatermarksFile updates PDF stamps/watermarks for all selected pages of inFile,
// writes the result to outFile and supports cancellation.
func UpdatePDFWatermarksFile(c context.Context, inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if inFile == "" {
		return ErrMissingPDFInput
	}

	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermark(c, fileName, desc, onTop, true, unit, conf)
	if err != nil {
		return fmt.Errorf("update watermarks: configure: %w", err)
	}

	return AddWatermarksFile(c, inFile, outFile, selectedPages, wm, conf)
}

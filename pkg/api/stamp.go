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
	"io"
	"os"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// WatermarkContext applies wm for selected pages to ctx.
func WatermarkContext(ctx *model.Context, selectedPages types.IntSet, wm *model.Watermark) error {
	return pdfcpu.AddWatermarks(ctx, selectedPages, wm)
}

// AddWatermarksMap adds watermarks in m to corresponding pages in rs and writes the result to w.
func AddWatermarksMap(rs io.ReadSeeker, w io.Writer, m map[int]*model.Watermark, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddWatermarksMap: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDWATERMARKS

	if len(m) == 0 {
		return errors.New("pdfcpu: missing watermarks")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err = pdfcpu.AddWatermarksMap(ctx, m); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddWatermarksMapFile adds watermarks to corresponding pages in m of inFile and writes the result to outFile.
func AddWatermarksMapFile(inFile, outFile string, m map[int]*model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return AddWatermarksMap(f1, f2, m, conf)
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages in rs and writes the result to w.
func AddWatermarksSliceMap(rs io.ReadSeeker, w io.Writer, m map[int][]*model.Watermark, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddWatermarksSliceMap: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDWATERMARKS

	if len(m) == 0 {
		return errors.New("pdfcpu: missing watermarks")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err = pdfcpu.AddWatermarksSliceMap(ctx, m); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddWatermarksSliceMapFile adds watermarks to corresponding pages in m of inFile and writes the result to outFile.
func AddWatermarksSliceMapFile(inFile, outFile string, m map[int][]*model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return AddWatermarksSliceMap(f1, f2, m, conf)
}

// AddWatermarks adds watermarks to all pages selected in rs and writes the result to w.
func AddWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, wm *model.Watermark, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: AddWatermarks: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDWATERMARKS
	conf.OptimizeDuplicateContentStreams = false

	if wm == nil {
		return errors.New("pdfcpu: missing watermark configuration")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	if err = pdfcpu.AddWatermarks(ctx, pages, wm); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddWatermarksFile adds watermarks to all selected pages of inFile and writes the result to outFile.
func AddWatermarksFile(inFile, outFile string, selectedPages []string, wm *model.Watermark, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return AddWatermarks(f1, f2, selectedPages, wm, conf)
}

// RemoveWatermarks removes watermarks from all pages selected in rs and writes the result to w.
func RemoveWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *model.Configuration) error {
	if rs == nil {
		return errors.New("pdfcpu: RemoveWatermarks: missing rs")
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEWATERMARKS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := ReadValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true, true)
	if err != nil {
		return err
	}

	if err = pdfcpu.RemoveWatermarks(ctx, pages); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != model.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemoveWatermarksFile removes watermarks from all selected pages of inFile and writes the result to outFile.
func RemoveWatermarksFile(inFile, outFile string, selectedPages []string, conf *model.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		f1.Close()
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			if outFile == "" || inFile == outFile {
				os.Remove(tmpFile)
			}
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

	return RemoveWatermarks(f1, f2, selectedPages, conf)
}

// HasWatermarks checks rs for watermarks.
func HasWatermarks(rs io.ReadSeeker, conf *model.Configuration) (bool, error) {
	if rs == nil {
		return false, errors.New("pdfcpu: HasWatermarks: missing rs")
	}

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return false, err
	}

	if err := pdfcpu.DetectWatermarks(ctx); err != nil {
		return false, err
	}

	return ctx.Watermarked, nil
}

// HasWatermarksFile checks inFile for watermarks.
func HasWatermarksFile(inFile string, conf *model.Configuration) (bool, error) {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}

	f, err := os.Open(inFile)
	if err != nil {
		return false, err
	}

	defer f.Close()

	return HasWatermarks(f, conf)
}

// TextWatermark returns a text watermark configuration.
func TextWatermark(text, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	wm, err := pdfcpu.ParseTextWatermarkDetails(text, desc, onTop, u)
	if err != nil {
		return nil, err
	}

	wm.Update = update

	return wm, nil
}

// ImageWatermark returns an image watermark configuration.
func ImageWatermark(fileName, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	wm, err := pdfcpu.ParseImageWatermarkDetails(fileName, desc, onTop, u)
	if err != nil {
		return nil, err
	}

	wm.Update = update

	return wm, nil
}

// ImageWatermarkForReader returns an image watermark configuration for r.
func ImageWatermarkForReader(r io.Reader, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	wm, err := pdfcpu.ParseImageWatermarkDetails("", desc, onTop, u)
	if err != nil {
		return nil, err
	}

	wm.Update = update
	wm.Image = r

	return wm, nil
}

// PDFWatermark returns a PDF watermark configuration.
func PDFWatermark(fileName, desc string, onTop, update bool, u types.DisplayUnit) (*model.Watermark, error) {
	wm, err := pdfcpu.ParsePDFWatermarkDetails(fileName, desc, onTop, u)
	if err != nil {
		return nil, err
	}

	wm.Update = update

	return wm, nil
}

// AddTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := TextWatermark(text, desc, onTop, false, unit)
	if err != nil {
		return err
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := ImageWatermark(fileName, desc, onTop, false, unit)
	if err != nil {
		return err
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddImageWatermarksForReaderFile adds image stamps/watermarks to all selected pages of inFile for r and writes the result to outFile.
func AddImageWatermarksForReaderFile(inFile, outFile string, selectedPages []string, onTop bool, r io.Reader, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := ImageWatermarkForReader(r, desc, onTop, false, unit)
	if err != nil {
		return err
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// AddPDFWatermarksFile adds PDF stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddPDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermark(fileName, desc, onTop, false, unit)
	if err != nil {
		return err
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdateTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdateTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := TextWatermark(text, desc, onTop, true, unit)
	if err != nil {
		return err
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdateImageWatermarksFile adds image stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdateImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}
	wm, err := ImageWatermark(fileName, desc, onTop, true, unit)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

// UpdatePDFWatermarksFile adds PDF stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func UpdatePDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *model.Configuration) error {
	unit := types.POINTS
	if conf != nil {
		unit = conf.Unit
	}

	wm, err := PDFWatermark(fileName, desc, onTop, true, unit)
	if err != nil {
		return err
	}

	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

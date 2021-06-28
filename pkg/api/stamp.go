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
	"github.com/pkg/errors"
)

// WatermarkContext applies wm for selected pages to ctx.
func WatermarkContext(ctx *pdfcpu.Context, selectedPages pdfcpu.IntSet, wm *pdfcpu.Watermark) error {
	return ctx.AddWatermarks(selectedPages, wm)
}

// AddWatermarksMap adds watermarks in m to corresponding pages in rs and writes the result to w.
func AddWatermarksMap(rs io.ReadSeeker, w io.Writer, m map[int]*pdfcpu.Watermark, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDWATERMARKS

	if len(m) == 0 {
		return errors.New("pdfcpu: missing watermarks")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err = ctx.AddWatermarksMap(m); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func AddWatermarksMapFile(inFile, outFile string, m map[int]*pdfcpu.Watermark, conf *pdfcpu.Configuration) (err error) {
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
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddWatermarksMap(f1, f2, m, conf)
}

// AddWatermarksSliceMap adds watermarks in m to corresponding pages in rs and writes the result to w.
func AddWatermarksSliceMap(rs io.ReadSeeker, w io.Writer, m map[int][]*pdfcpu.Watermark, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDWATERMARKS

	if len(m) == 0 {
		return errors.New("pdfcpu: missing watermarks")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	from := time.Now()

	if err = ctx.AddWatermarksSliceMap(m); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func AddWatermarksSliceMapFile(inFile, outFile string, m map[int][]*pdfcpu.Watermark, conf *pdfcpu.Configuration) (err error) {
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
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddWatermarksSliceMap(f1, f2, m, conf)
}

// AddWatermarks adds watermarks to all pages selected in rs and writes the result to w.
func AddWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, wm *pdfcpu.Watermark, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDWATERMARKS

	if wm == nil {
		return errors.New("pdfcpu: missing watermark configuration")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = ctx.AddWatermarks(pages, wm); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func AddWatermarksFile(inFile, outFile string, selectedPages []string, wm *pdfcpu.Watermark, conf *pdfcpu.Configuration) (err error) {
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
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddWatermarks(f1, f2, selectedPages, wm, conf)
}

// RemoveWatermarks removes watermarks from all pages selected in rs and writes the result to w.
func RemoveWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEWATERMARKS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = ctx.RemoveWatermarks(pages); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdfcpu.ValidationNone {
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
func RemoveWatermarksFile(inFile, outFile string, selectedPages []string, conf *pdfcpu.Configuration) (err error) {
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
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return RemoveWatermarks(f1, f2, selectedPages, conf)
}

// HasWatermarks checks rs for watermarks.
func HasWatermarks(rs io.ReadSeeker, conf *pdfcpu.Configuration) (bool, error) {
	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return false, err
	}
	if err := ctx.DetectWatermarks(); err != nil {
		return false, err
	}

	return ctx.Watermarked, nil
}

// HasWatermarksFile checks inFile for watermarks.
func HasWatermarksFile(inFile string, conf *pdfcpu.Configuration) (bool, error) {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}

	f, err := os.Open(inFile)
	if err != nil {
		return false, err
	}

	defer f.Close()

	return HasWatermarks(f, conf)
}

// TextWatermark returns a text watermark configuration.
func TextWatermark(text, desc string, onTop, update bool, u pdfcpu.DisplayUnit) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParseTextWatermarkDetails(text, desc, onTop, u)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	return wm, nil
}

// ImageWatermark returns an image watermark configuration.
func ImageWatermark(fileName, desc string, onTop, update bool, u pdfcpu.DisplayUnit) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParseImageWatermarkDetails(fileName, desc, onTop, u)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	return wm, nil
}

// ImageWatermarkForReader returns an image watermark configuration for r.
func ImageWatermarkForReader(r io.Reader, desc string, onTop, update bool, u pdfcpu.DisplayUnit) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParseImageWatermarkDetails("", desc, onTop, u)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	wm.Image = r
	return wm, nil
}

// PDFWatermark returns a PDF watermark configuration.
func PDFWatermark(fileName, desc string, onTop, update bool, u pdfcpu.DisplayUnit) (*pdfcpu.Watermark, error) {
	wm, err := pdfcpu.ParsePDFWatermarkDetails(fileName, desc, onTop, u)
	if err != nil {
		return nil, err
	}
	wm.Update = update
	return wm, nil
}

// AddTextWatermarksFile adds text stamps/watermarks to all selected pages of inFile and writes the result to outFile.
func AddTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
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
func AddImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
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
func AddImageWatermarksForReaderFile(inFile, outFile string, selectedPages []string, onTop bool, r io.Reader, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
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
func AddPDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
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
func UpdateTextWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, text, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
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
func UpdateImageWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
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
func UpdatePDFWatermarksFile(inFile, outFile string, selectedPages []string, onTop bool, fileName, desc string, conf *pdfcpu.Configuration) error {
	unit := pdfcpu.POINTS
	if conf != nil {
		unit = conf.Unit
	}
	wm, err := PDFWatermark(fileName, desc, onTop, true, unit)
	if err != nil {
		return err
	}
	return AddWatermarksFile(inFile, outFile, selectedPages, wm, conf)
}

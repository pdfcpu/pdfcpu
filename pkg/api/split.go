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
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func spanFileName(fileName string, from, thru int) string {
	baseFileName := filepath.Base(fileName)
	fn := strings.TrimSuffix(baseFileName, ".pdf")
	fn = fn + "_" + strconv.Itoa(from)
	if from == thru {
		return fn + ".pdf"
	}
	return fn + "-" + strconv.Itoa(thru) + ".pdf"
}

func writeSpan(ctx *pdfcpu.Context, from, thru int, outDir, fileName string, forBookmark bool) error {
	selectedPages := PagesForPageRange(from, thru)

	ctxDest, err := pdfcpu.CreateContextWithXRefTable(nil, pdfcpu.PaperSize["A4"])
	if err != nil {
		return err
	}

	usePgCache := false
	if err := pdfcpu.AddPages(ctx, ctxDest, selectedPages, usePgCache); err != nil {
		return err
	}

	w := ctxDest.Write
	w.DirName = outDir
	w.FileName = fileName + ".pdf"
	if !forBookmark {
		w.FileName = spanFileName(fileName, from, thru)
		//log.CLI.Printf("writing to: <%s>\n", w.FileName)
	}

	return pdfcpu.Write(ctxDest)
}

func writePageSpan(ctx *pdfcpu.Context, from, thru int, outDir, fileName string, forBookmark bool) error {
	selectedPages := PagesForPageRange(from, thru)

	// Create context with copies of selectedPages.
	ctxNew, err := ctx.ExtractPages(selectedPages, false)
	if err != nil {
		return err
	}

	// Write context to file.
	outFile := filepath.Join(outDir, fileName+".pdf")
	if !forBookmark {
		outFile = filepath.Join(outDir, spanFileName(fileName, from, thru))
	}

	return WriteContextFile(ctxNew, outFile)
}

func outPageSpan(ctx *pdfcpu.Context, from, thru int, forBookmark bool, w io.Writer) error {
	selectedPages := PagesForPageRange(from, thru)

	// Create context with copies of selectedPages.
	ctxNew, err := ctx.ExtractPages(selectedPages, false)
	if err != nil {
		return err
	}

	return WriteContext(ctxNew, w)
}

func writePageSpansSplitAlongBookmarks(ctx *pdfcpu.Context, outDir string) error {
	bms, err := ctx.BookmarksForOutline()
	if err != nil {
		return err
	}
	for _, bm := range bms {
		fileName := strings.Replace(bm.Title, " ", "_", -1)
		from := bm.PageFrom
		thru := bm.PageThru
		if thru == 0 {
			thru = ctx.PageCount
		}
		forBookmark := true
		if err := writePageSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}
	}
	return nil
}

func outPageSpansSplitAlongBookmarks(ctx *pdfcpu.Context) (buffers []bytes.Buffer, err error) {
	bms, err := ctx.BookmarksForOutline()
	if err != nil {
		return
	}
	for _, bm := range bms {
		from := bm.PageFrom
		thru := bm.PageThru
		if thru == 0 {
			thru = ctx.PageCount
		}
		forBookmark := true

		var buffer bytes.Buffer
		writer := bufio.NewWriter(&buffer)
		if err = outPageSpan(ctx, from, thru, forBookmark, writer); err != nil {
			return
		}
		buffers = append(buffers, buffer)
	}
	return
}

func writePageSpans(ctx *pdfcpu.Context, span int, outDir, fileName string) error {
	if span == 0 {
		return writePageSpansSplitAlongBookmarks(ctx, outDir)
	}

	forBookmark := false

	for i := 0; i < ctx.PageCount/span; i++ {
		start := i * span
		from := start + 1
		thru := start + span
		if err := writePageSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}
	}

	// A possible last file has less than span pages.
	if ctx.PageCount%span > 0 {
		start := (ctx.PageCount / span) * span
		from := start + 1
		thru := ctx.PageCount
		if err := writePageSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}
	}

	return nil
}

func outPageSpans(ctx *pdfcpu.Context, span int) (buffers []bytes.Buffer, err error) {
	if span == 0 {
		return outPageSpansSplitAlongBookmarks(ctx)
	}

	forBookmark := false

	for i := 0; i < ctx.PageCount/span; i++ {
		start := i * span
		from := start + 1
		thru := start + span

		var buffer bytes.Buffer
		writer := bufio.NewWriter(&buffer)
		if err = outPageSpan(ctx, from, thru, forBookmark, writer); err != nil {
			return
		}
		buffers = append(buffers, buffer)
	}

	// A possible last file has less than span pages.
	if ctx.PageCount%span > 0 {
		start := (ctx.PageCount / span) * span
		from := start + 1
		thru := ctx.PageCount

		var buffer bytes.Buffer
		w := bufio.NewWriter(&buffer)
		if err = outPageSpan(ctx, from, thru, forBookmark, w); err != nil {
			return
		}
		buffers = append(buffers, buffer)
	}

	return
}

// Split generates a sequence of PDF files in outDir for the PDF stream read from rs obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
// Default span: 1
func Split(rs io.ReadSeeker, outDir, fileName string, span int, conf *pdfcpu.Configuration) error {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.SPLIT

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	fromWrite := time.Now()

	if err = writePageSpans(ctx, span, outDir, fileName); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "split", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// SplitBuffer generates a sequence of PDF files in []bytes.Buffer for the PDF stream read from rs obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
// Default span: 1
func SplitBuffer(rs io.ReadSeeker, span int, conf *pdfcpu.Configuration) (buffers []bytes.Buffer, err error) {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.SPLIT

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return
	}

	if err = ctx.EnsurePageCount(); err != nil {
		return
	}

	fromWrite := time.Now()

	buffers, err = outPageSpans(ctx, span)

	if err != nil {
		return
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "split", durRead, durVal, durOpt, durWrite, durTotal)

	return
}

// SplitFile generates a sequence of PDF files in outDir for inFile obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
// Default span: 1
func SplitFile(inFile, outDir string, span int, conf *pdfcpu.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	log.CLI.Printf("splitting %s to %s/...\n", inFile, outDir)

	defer func() {
		if err != nil {
			f.Close()
			return
		}
		err = f.Close()
	}()

	return Split(f, outDir, filepath.Base(inFile), span, conf)
}

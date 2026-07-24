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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sanitize"
)

// PageSpan represents a contiguous page range and its generated PDF stream.
type PageSpan struct {
	// From is the first page number of this span.
	From int

	// Thru is the last page number of this span.
	Thru int

	// Reader provides the PDF stream for this span.
	Reader io.Reader
}

func pageSpan(ctx *model.Context, from, thru int) (*PageSpan, error) {
	ctxNew, err := pdfcpu.ExtractPages(ctx, PagesForPageRange(from, thru), false)
	if err != nil {
		return nil, fmt.Errorf("extract page span %d-%d: %w", from, thru, err)
	}

	var b bytes.Buffer
	if err := WriteContext(ctxNew, &b); err != nil {
		return nil, fmt.Errorf("write page span %d-%d: %w", from, thru, err)
	}

	return &PageSpan{From: from, Thru: thru, Reader: &b}, nil
}

func spanFileName(fileName string, from, thru int) string {
	baseFileName := filepath.Base(fileName)
	fn := strings.TrimSuffix(baseFileName, ".pdf")
	fn = fn + "_" + strconv.Itoa(from)
	if from == thru {
		return fn + ".pdf"
	}
	return fn + "-" + strconv.Itoa(thru) + ".pdf"
}

func splitOutPath(outDir, name string, forBookmark bool, from, thru int) string {
	p := filepath.Join(outDir, name+".pdf")
	if !forBookmark {
		p = filepath.Join(outDir, spanFileName(name, from, thru))
	}
	return p
}

func writePageSpan(ctx *model.Context, from, thru int, outPath string) error {
	ps, err := pageSpan(ctx, from, thru)
	if err != nil {
		return err
	}

	logWritingTo(outPath)

	if err := pdfcpu.WriteReader(outPath, ps.Reader); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

func readSplitContext(rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SPLIT

	return ReadValidateAndOptimize(rs, conf)
}

func validateSplitSpan(span int) error {
	if span < 0 {
		return ErrInvalidSplitSpan
	}
	return nil
}

func validateSplitPageNumbers(ctx *model.Context, pageNrs []int) error {
	if len(pageNrs) < 1 {
		return ErrMissingSplitPageNumbers
	}
	if pageNrs[0] < 2 || pageNrs[0] > ctx.PageCount {
		return ErrInvalidSplitPageNumberSequence
	}
	for i := 1; i < len(pageNrs); i++ {
		if pageNrs[i] <= pageNrs[i-1] {
			return ErrInvalidSplitPageNumberSequence
		}
	}
	return nil
}

func splitOpError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

func pageSpansSplitAlongBookmarks(ctx *model.Context) ([]*PageSpan, error) {
	pss := []*PageSpan{}

	bms, err := pdfcpu.Bookmarks(ctx)
	if err != nil {
		return nil, fmt.Errorf("read bookmarks: %w", err)
	}
	if len(bms) == 0 {
		return nil, fmt.Errorf("split along bookmarks: %w", ErrNoBookmarks)
	}

	for _, bm := range bms {

		from, thru := bm.PageFrom, bm.PageThru
		if thru == 0 {
			thru = ctx.PageCount
		}

		ps, err := pageSpan(ctx, from, thru)
		if err != nil {
			return nil, fmt.Errorf("split bookmark %q: %w", bm.Title, err)
		}
		pss = append(pss, ps)

	}

	return pss, nil
}

func validatePositiveSplitSpan(span int) error {
	if span <= 0 {
		return ErrInvalidSplitSpan
	}
	return nil
}

func pageSpans(ctx *model.Context, span int) ([]*PageSpan, error) {
	if err := validatePositiveSplitSpan(span); err != nil {
		return nil, err
	}

	pss := []*PageSpan{}

	for i := 0; i < ctx.PageCount/span; i++ {
		start := i * span
		from := start + 1
		thru := start + span
		ps, err := pageSpan(ctx, from, thru)
		if err != nil {
			return nil, err
		}
		pss = append(pss, ps)
	}

	// A possible last file has less than span pages.
	if ctx.PageCount%span > 0 {
		start := (ctx.PageCount / span) * span
		from := start + 1
		thru := ctx.PageCount
		ps, err := pageSpan(ctx, from, thru)
		if err != nil {
			return nil, fmt.Errorf("final span %d-%d: %w", from, thru, err)
		}
		pss = append(pss, ps)
	}

	return pss, nil
}

func writePageSpans(ctx *model.Context, span int, outDir, fileName string) error {
	if err := validatePositiveSplitSpan(span); err != nil {
		return err
	}

	forBookmark := false

	for i := 0; i < ctx.PageCount/span; i++ {
		start := i * span
		from, thru := start+1, start+span
		path := splitOutPath(outDir, fileName, forBookmark, from, thru)
		if err := writePageSpan(ctx, from, thru, path); err != nil {
			return err
		}
	}

	// A possible last file has less than span pages.
	if ctx.PageCount%span > 0 {
		start := (ctx.PageCount / span) * span
		from, thru := start+1, ctx.PageCount
		path := splitOutPath(outDir, fileName, forBookmark, from, thru)
		if err := writePageSpan(ctx, from, thru, path); err != nil {
			return fmt.Errorf("final span %d-%d: %w", from, thru, err)
		}
	}

	return nil
}

func writePageSpansSplitAlongBookmarks(ctx *model.Context, outDir string) error {
	forBookmark := true

	bms, err := pdfcpu.Bookmarks(ctx)
	if err != nil {
		return fmt.Errorf("read bookmarks: %w", err)
	}
	if len(bms) == 0 {
		return fmt.Errorf("bookmarks: %w", ErrNoBookmarks)
	}

	for i, bm := range bms {
		fileName, err := sanitize.Path(bm.Title)
		if err != nil {
			fileName = "bookmark_" + strconv.Itoa(i+1)
		}
		from, thru := bm.PageFrom, bm.PageThru
		if thru == 0 {
			thru = ctx.PageCount
		}
		path := splitOutPath(outDir, fileName, forBookmark, from, thru)
		if err := writePageSpan(ctx, from, thru, path); err != nil {
			return fmt.Errorf("split bookmark %q: %w", bm.Title, err)
		}
	}

	return nil
}

func writePageSpansSplitAlongPages(ctx *model.Context, pageNrs []int, outDir, fileName string) error {
	// pageNumbers is a sorted sequence of page numbers.
	forBookmark := false
	from, thru := 1, 0

	if err := validateSplitPageNumbers(ctx, pageNrs); err != nil {
		return err
	}

	for i := range pageNrs {
		thru = pageNrs[i] - 1
		if thru >= ctx.PageCount {
			break
		}
		path := splitOutPath(outDir, fileName, forBookmark, from, thru)
		if err := writePageSpan(ctx, from, thru, path); err != nil {
			return fmt.Errorf("split before page %d: %w", pageNrs[i], err)
		}
		from = thru + 1
	}

	thru = ctx.PageCount
	path := splitOutPath(outDir, fileName, forBookmark, from, thru)
	if err := writePageSpan(ctx, from, thru, path); err != nil {
		return fmt.Errorf("final span %d-%d: %w", from, thru, err)
	}

	return nil
}

// SplitRaw returns page spans for the PDF stream read from rs obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
// Default span: 1
// SplitRaw is not used within this repository.
func SplitRaw(rs io.ReadSeeker, span int, conf *model.Configuration) (ps []*PageSpan, err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return nil, ErrMissingPDFReadSeeker
	}

	if err := validateSplitSpan(span); err != nil {
		return nil, err
	}

	ctx, err := readSplitContext(rs, conf)
	if err != nil {
		return nil, splitOpError("split", err)
	}

	if span == 0 {
		ps, err = pageSpansSplitAlongBookmarks(ctx)
		return ps, splitOpError("split", err)
	}
	ps, err = pageSpans(ctx, span)
	return ps, splitOpError("split", err)
}

// Split generates a sequence of PDF files in outDir for the PDF stream read from rs obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
// Default span: 1
func Split(rs io.ReadSeeker, outDir, fileName string, span int, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	if err := validateSplitSpan(span); err != nil {
		return err
	}

	ctx, err := readSplitContext(rs, conf)
	if err != nil {
		return splitOpError("split", err)
	}

	if span == 0 {
		return splitOpError("split", writePageSpansSplitAlongBookmarks(ctx, outDir))
	}
	return splitOpError("split", writePageSpans(ctx, span, outDir, fileName))
}

// SplitFile generates a sequence of PDF files in outDir for inFile obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
// Default span: 1
func SplitFile(inFile, outDir string, span int, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("split: open %s: %w", inFile, err)
	}
	if log.CLIEnabled() {
		log.CLI.Printf("splitting %s to %s/...\n", inFile, outDir)
	}

	defer func() {
		closeErr := f.Close()
		if err != nil {
			return
		}
		if closeErr != nil {
			err = fmt.Errorf("split: close %s: %w", inFile, closeErr)
		}
	}()

	if err = Split(f, outDir, filepath.Base(inFile), span, conf); err != nil {
		return fmt.Errorf("split %s: %w", inFile, err)
	}
	return nil
}

// SplitByPageNr splits rs before the specified 1-based page numbers and writes result files to outDir.
// Page numbers must be sorted, unique, and at least 2.
func SplitByPageNr(rs io.ReadSeeker, outDir, fileName string, pageNrs []int, conf *model.Configuration) (err error) {
	defer fault.Catch(&err)

	if rs == nil {
		return ErrMissingPDFReadSeeker
	}

	ctx, err := readSplitContext(rs, conf)
	if err != nil {
		return splitOpError("split by page number", err)
	}

	if err := writePageSpansSplitAlongPages(ctx, pageNrs, outDir, fileName); err != nil {
		return fmt.Errorf("split by page number: %w", err)
	}
	return nil
}

// SplitByPageNrFile splits inFile before the specified 1-based page numbers and writes result files to outDir.
// Page numbers must be sorted, unique, and at least 2.
func SplitByPageNrFile(inFile, outDir string, pageNrs []int, conf *model.Configuration) (err error) {
	if inFile == "" {
		return ErrMissingPDFInput
	}

	f, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("split by page number: open %s: %w", inFile, err)
	}
	if log.CLIEnabled() {
		log.CLI.Printf("splitting %s to %s/...\n", inFile, outDir)
	}

	defer func() {
		closeErr := f.Close()
		if err != nil {
			return
		}
		if closeErr != nil {
			err = fmt.Errorf("split by page number: close %s: %w", inFile, closeErr)
		}
	}()

	if err = SplitByPageNr(f, outDir, filepath.Base(inFile), pageNrs, conf); err != nil {
		return fmt.Errorf("split by page number %s: %w", inFile, err)
	}
	return nil
}

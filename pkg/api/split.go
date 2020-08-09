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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
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

func selectedPageRange(from, thru int) []int {
	s := make([]int, thru-from+1)
	for i := 0; i < len(s); i++ {
		s[i] = from + i
	}
	return s
}

func writeSpan(ctx *pdfcpu.Context, from, thru int, outDir, fileName string, forBookmark bool) error {
	selectedPages := selectedPageRange(from, thru)

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

type bookmark struct {
	title    string
	pageFrom int
	pageThru int // We assume, pageThru has to be at least pageFrom and reaches until before pageFrom of the next bookmark.
}

func dereferenceDestinationArray(ctx *pdfcpu.Context, key string) (pdfcpu.Array, error) {
	o, ok := ctx.Names["Dests"].Value(key)
	if !ok {
		return nil, errors.New("Corrupt named destination")
	}
	return ctx.DereferenceArray(o)
}

func positionToOutlineTreeLevel(ctx *pdfcpu.Context) (pdfcpu.Dict, *pdfcpu.IndirectRef, error) {
	// Load Dests nametree.
	if err := ctx.LocateNameTree("Dests", false); err != nil {
		return nil, nil, err
	}

	ir, err := ctx.Outlines()
	if err != nil {
		return nil, nil, err
	}
	if ir == nil {
		return nil, nil, errors.New("No bookmarks available")
	}

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return nil, nil, err
	}
	if d == nil {
		return nil, nil, errors.New("No bookmarks available")
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	// We consider Bookmarks at level 1 or 2 only.
	for *first == *last {
		//fmt.Println("first == last")
		if d, err = ctx.DereferenceDict(*first); err != nil {
			return nil, nil, err
		}
		first = d.IndirectRefEntry("First")
		last = d.IndirectRefEntry("Last")
	}

	return d, first, nil
}

func bookmarksForOutlineLevel1(ctx *pdfcpu.Context) ([]bookmark, error) {
	d, first, err := positionToOutlineTreeLevel(ctx)
	if err != nil {
		return nil, err
	}

	bms := []bookmark{}

	// Process linked list of outline items.
	for ir := first; ir != nil; ir = d.IndirectRefEntry("Next") {

		//objNr := ir.ObjectNumber.Value()
		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return nil, err
		}

		title, _ := pdfcpu.Text(d["Title"])
		//fmt.Printf("bookmark obj:%d title:%s\n", objNr, title)

		dest, found := d["Dest"]
		if !found {
			return nil, errors.New("No destination based bookmarks available")
		}

		var pageIndRef pdfcpu.IndirectRef

		dest, _ = ctx.Dereference(dest)

		switch dest := dest.(type) {

		case pdfcpu.Name:
			//fmt.Printf("dest is Name: %s\n", dest.Value())
			arr, err := dereferenceDestinationArray(ctx, dest.Value())
			if err != nil {
				return nil, err
			}
			pageIndRef = arr[0].(pdf.IndirectRef)

		case pdfcpu.StringLiteral:
			//fmt.Printf("dest is StringLiteral: %s\n", dest.Value())
			arr, err := dereferenceDestinationArray(ctx, dest.Value())
			if err != nil {
				return nil, err
			}
			pageIndRef = arr[0].(pdf.IndirectRef)

		case pdfcpu.HexLiteral:
			//fmt.Printf("dest is HexLiteral: %s\n", dest.Value())
			arr, err := dereferenceDestinationArray(ctx, dest.Value())
			if err != nil {
				return nil, err
			}
			pageIndRef = arr[0].(pdf.IndirectRef)

		case pdf.Array:
			pageIndRef = dest[0].(pdf.IndirectRef)

		}

		pageFrom, err := ctx.PageNumber(pageIndRef.ObjectNumber.Value())
		if err != nil {
			return nil, err
		}

		if len(bms) > 0 {
			if pageFrom > bms[len(bms)-1].pageFrom {
				bms[len(bms)-1].pageThru = pageFrom - 1
			} else {
				bms[len(bms)-1].pageThru = bms[len(bms)-1].pageFrom
			}
		}
		bms = append(bms, bookmark{title: title, pageFrom: pageFrom})
	}

	return bms, nil
}

func writePDFSequenceSplitAlongBookmarks(ctx *pdfcpu.Context, outDir string) error {
	bms, err := bookmarksForOutlineLevel1(ctx)
	if err != nil {
		return err
	}

	for _, bm := range bms {
		fileName := bm.title
		from := bm.pageFrom
		thru := bm.pageThru
		if thru == 0 {
			thru = ctx.PageCount
		}
		forBookmark := true
		if err := writeSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}
	}

	return nil
}

func writePDFSequence(ctx *pdfcpu.Context, span int, outDir, fileName string) error {
	if span == 0 {
		return writePDFSequenceSplitAlongBookmarks(ctx, outDir)
	}

	forBookmark := false

	for i := 0; i < ctx.PageCount/span; i++ {
		start := i * span
		from := start + 1
		thru := start + span
		if err := writeSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}
	}

	// A possible last file that has less than span pages.
	if ctx.PageCount%span > 0 {

		start := (ctx.PageCount / span) * span
		from := start + 1
		thru := ctx.PageCount

		if err := writeSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}

	}

	return nil
}

// Split generates a sequence of PDF files in outDir for the PDF stream read from rs obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
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

	if err = writePDFSequence(ctx, span, outDir, fileName); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "split", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// SplitFile generates a sequence of PDF files in outDir for inFile obeying given split span.
// The default span 1 creates a sequence of single page PDFs.
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
